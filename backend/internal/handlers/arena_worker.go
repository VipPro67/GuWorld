package handlers

import (
	"context"
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
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

	rows, err := tx.Query("SELECT player_id, rank FROM arena_rankings")
	if err != nil {
		logger.S.Error("ArenaSettlement Error: Failed to query rankings", err)
		return
	}

	type playerReward struct {
		playerID    string
		rewardCoins int
	}
	var rewards []playerReward

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

		rewards = append(rewards, playerReward{playerID: playerID, rewardCoins: rewardCoins})
	}
	rows.Close()

	for _, r := range rewards {
		errUpdate := InventorySvc.AddItem(tx, r.playerID, "arena_coin", "materials", r.rewardCoins)
		if errUpdate != nil {
			logger.S.Errorf("ArenaSettlement Error: Failed to give reward to %s: %v", r.playerID, errUpdate)
		}
	}

	if err := tx.Commit(); err != nil {
		logger.S.Error("ArenaSettlement Error: Failed to commit tx", err)
	} else {
		logger.S.Info("Daily Arena Settlement completed successfully.")
		_ = SendSystemMessage("world", sql.NullString{}, "Phần thưởng đấu trường đã được kết toán! Chúc mừng các đạo hữu đã đạt thứ hạng cao.")
	}
}
