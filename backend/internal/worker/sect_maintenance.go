package worker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/sectmechanics"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type QueryContextRunner interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type SectMaintenanceBreakdown struct {
	LevelFee         int64 `json:"level_fee"`
	SectLevel        int   `json:"sect_level"`
	MemberFee        int64 `json:"member_fee"`
	MemberCount      int   `json:"member_count"`
	MaterialsFee     int64 `json:"materials_fee"`
	MaterialsCount   int64 `json:"materials_count"`
	MortalGuFee      int64 `json:"mortal_gu_fee"`
	MortalGuCount    int64 `json:"mortal_gu_count"`
	ImmortalGuFee    int64 `json:"immortal_gu_fee"`
	ImmortalGuCount  int64 `json:"immortal_gu_count"`
	RecipesFee       int64 `json:"recipes_fee"`
	RecipesCount     int   `json:"recipes_count"`
	TreasuryLevel    int   `json:"treasury_level"`
	TreasuryDiscount int64 `json:"treasury_discount"`
	TotalFee         int64 `json:"total_fee"`
}

// CalculateSectMaintenanceFee calculates the daily maintenance fee:
// Phí Bảo Trì = (Cấp TM * 2.000) + (Số TV * 200) + Phí Kho Tàng
// Trong đó Phí Kho Tàng (trước chiết khấu 50%-90% theo cấp Bảo Khố):
// - Linh tài: sum(Rank * 5 * Số lượng)
// - Cổ phàm (Rank 1-5): sum(Rank * 10 * Số lượng)
// - Tiên Cổ (Rank 6-9): sum(Rank * 200 * Số lượng)
// - Bí phương: sum(TargetRank * 50 * Số lượng)
func CalculateSectMaintenanceFee(ctx context.Context, q QueryContextRunner, sectID string, sectLevel int) (int64, SectMaintenanceBreakdown, error) {
	if sectLevel < 1 {
		sectLevel = 1
	}

	var breakdown SectMaintenanceBreakdown
	breakdown.SectLevel = sectLevel
	breakdown.LevelFee = int64(sectLevel) * 2000
	var treasuryLevel int
	if err := q.QueryRowContext(ctx, `SELECT COALESCE((buildings->>'treasury')::int, 1) FROM sects WHERE id = $1`, sectID).Scan(&treasuryLevel); err != nil {
		return 0, breakdown, fmt.Errorf("failed to load treasury level: %w", err)
	}
	breakdown.TreasuryLevel = treasuryLevel

	// 1. Members
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM sect_members WHERE sect_id = $1`, sectID).Scan(&breakdown.MemberCount)
	if err != nil && err != sql.ErrNoRows {
		return 0, breakdown, fmt.Errorf("failed to count sect members: %w", err)
	}
	breakdown.MemberFee = int64(breakdown.MemberCount) * 200

	// 2. Materials
	matRows, err := q.QueryContext(ctx, `SELECT item_id, quantity FROM sect_vault_materials WHERE sect_id = $1 AND quantity > 0`, sectID)
	if err == nil {
		for matRows.Next() {
			var itemID string
			var qty int64
			if err := matRows.Scan(&itemID, &qty); err == nil {
				rank := 1
				if spec, ok := models.ItemRegistry[itemID]; ok && spec.Rank > 0 {
					rank = spec.Rank
				}
				breakdown.MaterialsFee += int64(rank) * 5 * qty
				breakdown.MaterialsCount += qty
			}
		}
		matRows.Close()
	}

	// 3. Gu
	guRows, err := q.QueryContext(ctx, `SELECT rank, quantity FROM sect_vault_gu WHERE sect_id = $1 AND quantity > 0`, sectID)
	if err == nil {
		for guRows.Next() {
			var rank int
			var qty int64
			if err := guRows.Scan(&rank, &qty); err == nil {
				if rank <= 0 {
					rank = 1
				}
				if rank < 6 {
					breakdown.MortalGuFee += int64(rank) * 10 * qty
					breakdown.MortalGuCount += qty
				} else {
					breakdown.ImmortalGuFee += int64(rank) * 200 * qty
					breakdown.ImmortalGuCount += qty
				}
			}
		}
		guRows.Close()
	}

	// 4. Recipes
	recRows, err := q.QueryContext(ctx, `SELECT recipe_id FROM sect_vault_recipes WHERE sect_id = $1`, sectID)
	if err == nil {
		for recRows.Next() {
			var recID string
			if err := recRows.Scan(&recID); err == nil {
				targetRank := 1
				if _, tr, _, parseErr := models.ParseRecipeID(recID); parseErr == nil && tr > 0 {
					targetRank = tr
				} else if guDef, ok := models.GetGuAny(recID); ok && guDef.Rank > 0 {
					targetRank = guDef.Rank
				}
				breakdown.RecipesFee += int64(targetRank) * 50
				breakdown.RecipesCount++
			}
		}
		recRows.Close()
	}

	var saved int64
	breakdown.MaterialsFee, saved = sectmechanics.DiscountVaultFee(breakdown.MaterialsFee, treasuryLevel)
	breakdown.TreasuryDiscount += saved
	breakdown.MortalGuFee, saved = sectmechanics.DiscountVaultFee(breakdown.MortalGuFee, treasuryLevel)
	breakdown.TreasuryDiscount += saved
	breakdown.ImmortalGuFee, saved = sectmechanics.DiscountVaultFee(breakdown.ImmortalGuFee, treasuryLevel)
	breakdown.TreasuryDiscount += saved
	breakdown.RecipesFee, saved = sectmechanics.DiscountVaultFee(breakdown.RecipesFee, treasuryLevel)
	breakdown.TreasuryDiscount += saved
	breakdown.TotalFee = breakdown.LevelFee + breakdown.MemberFee + breakdown.MaterialsFee + breakdown.MortalGuFee + breakdown.ImmortalGuFee + breakdown.RecipesFee
	return breakdown.TotalFee, breakdown, nil
}

func StartSectMaintenanceWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Sect Maintenance Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Sect Maintenance Worker stopped")
				return
			case <-ticker.C:
				processSectMaintenanceTick(ctx, db)
			}
		}
	}()
}

func processSectMaintenanceTick(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, level, funds, last_maintenance_at
		FROM sects
	`)
	if err != nil {
		logger.S.Errorf("SectMaintenanceWorker: failed to query sects: %v", err)
		return
	}
	defer rows.Close()

	type sectRecord struct {
		id              string
		level           int
		funds           int64
		lastMaintenance sql.NullTime
	}
	var sectsToProcess []sectRecord

	for rows.Next() {
		var s sectRecord
		if err := rows.Scan(&s.id, &s.level, &s.funds, &s.lastMaintenance); err != nil {
			logger.S.Errorf("SectMaintenanceWorker: failed to scan sect: %v", err)
			continue
		}
		if s.lastMaintenance.Valid && s.lastMaintenance.Time.After(time.Now().Add(-24*time.Hour)) {
			continue
		}
		sectsToProcess = append(sectsToProcess, s)
	}
	rows.Close()

	for _, s := range sectsToProcess {
		fee, breakdown, err := CalculateSectMaintenanceFee(ctx, db, s.id, s.level)
		if err != nil {
			logger.S.Errorf("SectMaintenanceWorker: error calculating fee for sect %s: %v", s.id, err)
			continue
		}

		now := time.Now()
		if s.funds >= fee {
			newFunds := s.funds - fee
			_, err := db.ExecContext(ctx, `
				UPDATE sects SET funds = $1, last_maintenance_at = $2
				WHERE id = $3
			`, newFunds, now, s.id)
			if err != nil {
				logger.S.Errorf("SectMaintenanceWorker: failed to deduct fee for sect %s: %v", s.id, err)
				continue
			}

			vaultSlots := breakdown.MaterialsCount + breakdown.MortalGuCount + breakdown.ImmortalGuCount + int64(breakdown.RecipesCount)
			chatMsg := fmt.Sprintf("[Bảo Trì] Tông môn đã thanh toán %d Quỹ duy trì hôm nay (Thành viên: %d, Kho: %d). Quỹ còn lại: %d Quỹ.",
				fee, breakdown.MemberCount, vaultSlots, newFunds)
			sendSectSystemChatMessage(ctx, db, s.id, chatMsg)
		} else {
			newLevel := s.level - 1
			if newLevel < 1 {
				newLevel = 1
			}
			_, err := db.ExecContext(ctx, `
				UPDATE sects SET level = $1, last_maintenance_at = $2
				WHERE id = $3
			`, newLevel, now, s.id)
			if err != nil {
				logger.S.Errorf("SectMaintenanceWorker: failed to downgrade sect %s: %v", s.id, err)
				continue
			}

			alertChat := fmt.Sprintf("[Bảo Trì] CẢNH BÁO: Tông môn không đủ quỹ duy trì (%d/%d), cấp bậc bị giáng xuống Cấp %d!",
				s.funds, fee, newLevel)
			sendSectSystemChatMessage(ctx, db, s.id, alertChat)

			notifySectLeader(ctx, db, s.id, s.level, newLevel, s.funds, fee)
		}
	}
}

func sendSectSystemChatMessage(ctx context.Context, db *sql.DB, sectID, content string) {
	globalUUID := "00000000-0000-0000-0000-000000000000"
	systemUUID := "11111111-1111-1111-1111-111111111111"

	_, err := db.ExecContext(ctx, `
		INSERT INTO messages (sender_id, receiver_id, content, channel, sect_id)
		VALUES ($1, $2, $3, 'sect', $4)
	`, systemUUID, globalUUID, content, sectID)
	if err != nil {
		logger.S.Errorf("sendSectSystemChatMessage failed for sect %s: %v", sectID, err)
		return
	}
	_ = cache.PublishEventQuick("chat:sect:"+sectID, map[string]any{"type": "new_message"})
}

func notifySectLeader(ctx context.Context, db *sql.DB, sectID string, oldLevel, newLevel int, funds, fee int64) {
	var leaderID sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT player_id FROM sect_members
		WHERE sect_id = $1 AND member_rank = 'CHUONG_MON'
		LIMIT 1
	`).Scan(&leaderID)
	if err != nil || !leaderID.Valid {
		logger.S.Debugf("SectMaintenanceWorker: no leader found for sect %s", sectID)
		return
	}

	content := fmt.Sprintf("Tông môn không đủ ngân quỹ chi trả phí bảo trì định kỳ (Cần %d Quỹ nhưng hiện chỉ có %d Quỹ)! Sơn môn đã bị giáng từ Cấp %d xuống Cấp %d. Hãy sớm quyên góp và bổ sung quỹ để tránh tiếp tục suy thoái.", fee, funds, oldLevel, newLevel)
	_, err = db.ExecContext(ctx, `
		INSERT INTO system_messages (player_id, type, title, content)
		VALUES ($1, 'SECT_MAINTENANCE', 'CẢNH BÁO TÔNG MÔN SUY THOÁI', $2)
	`, leaderID.String, content)
	if err != nil {
		logger.S.Errorf("SectMaintenanceWorker: failed to notify sect leader %s: %v", leaderID.String, err)
	}
}
