package handlers

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartArenaSettlementWorker runs every minute, checking if it is 22:00
func StartArenaSettlementWorker(ctx context.Context) {
	logger.S.Info("Starting Arena Settlement Worker...")

	var lastSettlementDay int

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.S.Info("Arena Settlement Worker stopped.")
				return
			case t := <-ticker.C:
				if t.Hour() == 22 && t.Minute() == 30 {
					currentDay := t.YearDay()
					if lastSettlementDay != currentDay {
						logger.S.Info("Running Daily Arena Settlement...")
						runArenaSettlement()
						lastSettlementDay = currentDay
					}
				}
			}
		}
	}()
}

func runArenaSettlement() {
	tx, err := db.DB.Begin()
	if err != nil {
		logger.S.Error("ArenaSettlement Error: Failed to start tx", err)
		return
	}
	defer tx.Rollback()

	// Get top 50 rankings to award rewards
	rows, err := tx.Query("SELECT player_id, rank FROM arena_rankings WHERE rank <= 50 ORDER BY rank ASC")
	if err != nil {
		logger.S.Error("ArenaSettlement Error: Failed to query rankings", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var playerID string
		var rank int
		if err := rows.Scan(&playerID, &rank); err != nil {
			continue
		}

		rewardCoins := 50
		if rank == 1 {
			rewardCoins = 1000
		} else if rank == 2 {
			rewardCoins = 800
		} else if rank == 3 {
			rewardCoins = 500
		} else if rank <= 10 {
			rewardCoins = 200
		}

		// materials_inventory no longer stored on players table save it in public.player_inventory_items
		_, errUpdate := tx.Exec(`
			INSERT INTO player_inventory_items (player_id, item_id, category, quantity, updated_at)
			VALUES ($1, 'arena_coin', 'materials', $2, NOW())
			ON CONFLICT (player_id, item_id, category)
			DO UPDATE SET quantity = player_inventory_items.quantity + EXCLUDED.quantity, updated_at = NOW()
		`, playerID, rewardCoins)

		if errUpdate != nil {
			logger.S.Errorf("ArenaSettlement Error: Failed to give reward to %s: %v", playerID, errUpdate)
		} else {
			if pUUID, errParse := uuid.Parse(playerID); errParse == nil {
				auditlog.Logf(tx, &pUUID, "arena.settle_reward", "rank=%d rewardCoins=%d", rank, rewardCoins)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		logger.S.Error("ArenaSettlement Error: Failed to commit tx", err)
	} else {
		logger.S.Info("Daily Arena Settlement completed successfully.")
		_ = SendSystemMessage("world", sql.NullString{}, "Phần thưởng đấu trường đã được kết toán! Chúc mừng các đạo hữu đã đạt thứ hạng cao.")
	}
}
