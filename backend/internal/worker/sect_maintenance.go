package worker

import (
	"context"
	"database/sql"
	"encoding/json"
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

// materialFeeItemsPerQuy is how many Linh tài units cost 1 Quỹ of daily
// maintenance, regardless of their rank.
const materialFeeItemsPerQuy = 200

// rankedQty is a single vault row: one rank and the quantity stored at it.
type rankedQty struct {
	Rank int
	Qty  int64
}

// vaultFeeInput is the raw content of a sect vault, ready to be priced.
type vaultFeeInput struct {
	MaterialQty   int64
	MortalGu      []rankedQty
	ImmortalGu    []rankedQty
	RecipeRanks   []int
	TreasuryLevel int
}

// computeVaultFees prices a sect vault for one maintenance cycle.
//   - Linh tài: floor(Tổng số lượng / 200) — đồng hạng mọi rank and exempt
//     from the treasury discount.
//   - Cổ phàm (Rank 1-5): sum(Rank * 10 * Số lượng)
//   - Tiên Cổ (Rank 6-9): sum(Rank * 200 * Số lượng)
//   - Bí phương: sum(TargetRank * 50 * Số lượng)
//
// The last three categories are discounted 50%-90% by treasury level; discount
// reports how much was saved in total (materials never contribute to it).
func computeVaultFees(in vaultFeeInput) (materials, mortal, immortal, recipes, discount int64) {
	if in.MaterialQty > 0 {
		materials = in.MaterialQty / materialFeeItemsPerQuy
	}
	for _, r := range in.MortalGu {
		mortal += int64(max(r.Rank, 1)) * 10 * max(r.Qty, 0)
	}
	for _, r := range in.ImmortalGu {
		immortal += int64(max(r.Rank, 1)) * 200 * max(r.Qty, 0)
	}
	for _, targetRank := range in.RecipeRanks {
		recipes += int64(max(targetRank, 1)) * 50
	}

	var saved int64
	mortal, saved = sectmechanics.DiscountVaultFee(mortal, in.TreasuryLevel)
	discount += saved
	immortal, saved = sectmechanics.DiscountVaultFee(immortal, in.TreasuryLevel)
	discount += saved
	recipes, saved = sectmechanics.DiscountVaultFee(recipes, in.TreasuryLevel)
	discount += saved
	return
}

// CalculateSectMaintenanceFee calculates the daily maintenance fee:
// Phí Bảo Trì = (Cấp TM * 2.000) + (Số TV * 200) + Phí Kho Tàng
// Trong đó Phí Kho Tàng do computeVaultFees định giá (xem comment ở đó).
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

	// 2. Materials — counted only; priced flat by computeVaultFees.
	matRows, err := q.QueryContext(ctx, `SELECT quantity FROM sect_vault_materials WHERE sect_id = $1 AND quantity > 0`, sectID)
	if err == nil {
		for matRows.Next() {
			var qty int64
			if err := matRows.Scan(&qty); err == nil && qty > 0 {
				breakdown.MaterialsCount += qty
			}
		}
		matRows.Close()
	}

	// 3. Gu
	var mortalGu, immortalGu []rankedQty
	guRows, err := q.QueryContext(ctx, `SELECT rank, quantity FROM sect_vault_gu WHERE sect_id = $1 AND quantity > 0`, sectID)
	if err == nil {
		for guRows.Next() {
			var rank int
			var qty int64
			if err := guRows.Scan(&rank, &qty); err == nil && qty > 0 {
				if rank < 6 {
					mortalGu = append(mortalGu, rankedQty{Rank: rank, Qty: qty})
					breakdown.MortalGuCount += qty
				} else {
					immortalGu = append(immortalGu, rankedQty{Rank: rank, Qty: qty})
					breakdown.ImmortalGuCount += qty
				}
			}
		}
		guRows.Close()
	}

	// 4. Recipes
	var recipeRanks []int
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
				recipeRanks = append(recipeRanks, targetRank)
				breakdown.RecipesCount++
			}
		}
		recRows.Close()
	}

	// 5. Price everything in one place so the formula stays testable.
	materialsFee, mortalGuFee, immortalGuFee, recipesFee, saved := computeVaultFees(vaultFeeInput{
		MaterialQty:   breakdown.MaterialsCount,
		MortalGu:      mortalGu,
		ImmortalGu:    immortalGu,
		RecipeRanks:   recipeRanks,
		TreasuryLevel: treasuryLevel,
	})
	breakdown.MaterialsFee = materialsFee
	breakdown.MortalGuFee = mortalGuFee
	breakdown.ImmortalGuFee = immortalGuFee
	breakdown.RecipesFee = recipesFee
	breakdown.TreasuryDiscount = saved

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
		SELECT id, level, funds, buildings, last_maintenance_at
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
		buildings       []byte
		lastMaintenance sql.NullTime
	}
	var sectsToProcess []sectRecord

	for rows.Next() {
		var s sectRecord
		if err := rows.Scan(&s.id, &s.level, &s.funds, &s.buildings, &s.lastMaintenance); err != nil {
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
		cutoff := now.Add(-24 * time.Hour)
		var remainingFunds int64
		errDeduct := db.QueryRowContext(ctx, `
			UPDATE sects SET funds = funds - $1, last_maintenance_at = $2
			WHERE id = $3 AND funds >= $1 AND (last_maintenance_at IS NULL OR last_maintenance_at <= $4)
			RETURNING funds
		`, fee, now, s.id, cutoff).Scan(&remainingFunds)

		if errDeduct == nil {
			vaultSlots := breakdown.MaterialsCount + breakdown.MortalGuCount + breakdown.ImmortalGuCount + int64(breakdown.RecipesCount)
			chatMsg := fmt.Sprintf("[Bảo Trì] Tông môn đã thanh toán %d Quỹ duy trì hôm nay (Thành viên: %d, Kho: %d). Quỹ còn lại: %d Quỹ.",
				fee, breakdown.MemberCount, vaultSlots, remainingFunds)
			sendSectSystemChatMessage(ctx, db, s.id, chatMsg)
			continue
		}
		if errDeduct != sql.ErrNoRows {
			logger.S.Errorf("SectMaintenanceWorker: failed to deduct fee for sect %s: %v", s.id, errDeduct)
			continue
		}

		// The UPDATE was rejected. It carries two independent guards, so re-read
		// the row to tell "already paid this window" from "cannot afford it" —
		// downgrading on the wrong one would demote a sect that has paid.
		var curFunds int64
		var curMaintenance sql.NullTime
		if errRead := db.QueryRowContext(ctx, `SELECT funds, last_maintenance_at FROM sects WHERE id = $1`, s.id).
			Scan(&curFunds, &curMaintenance); errRead != nil {
			if errRead != sql.ErrNoRows {
				logger.S.Errorf("SectMaintenanceWorker: failed to re-read sect %s: %v", s.id, errRead)
			}
			continue
		}
		if curMaintenance.Valid && curMaintenance.Time.After(cutoff) {
			continue
		}
		if curFunds >= fee {
			// Funds arrived between the two statements; the next tick retries.
			continue
		}

		newLevel := max(s.level-1, 1)
		if newLevel == s.level {
			// Already at the lowest level: there is nothing left to demote, just
			// close the window so the same complaint is not repeated every tick.
			if _, errStamp := db.ExecContext(ctx, `
				UPDATE sects SET last_maintenance_at = $1
				WHERE id = $2 AND (last_maintenance_at IS NULL OR last_maintenance_at <= $3)
			`, now, s.id, cutoff); errStamp != nil {
				logger.S.Errorf("SectMaintenanceWorker: failed to stamp maintenance for sect %s: %v", s.id, errStamp)
				continue
			}
			sendSectSystemChatMessage(ctx, db, s.id, fmt.Sprintf(
				"[Bảo Trì] Chưa đủ quỹ đóng phí duy trì (cần %d, hiện %d). Phí chưa được thu, tông giữ nguyên Cấp %d.",
				fee, curFunds, s.level))
			continue
		}

		// Demote the sect and shrink its territory with it: main_hall drops to
		// the new level and no other building may stay above it, so the radius
		// derived from main_hall shrinks too.
		newBuildings, errClamp := clampBuildings(s.buildings, newLevel)
		if errClamp != nil {
			logger.S.Errorf("SectMaintenanceWorker: failed to clamp buildings for sect %s: %v", s.id, errClamp)
			continue
		}
		var appliedLevel int
		errDown := db.QueryRowContext(ctx, `
			UPDATE sects SET level = $1, buildings = $2::jsonb, last_maintenance_at = $3
			WHERE id = $4 AND (last_maintenance_at IS NULL OR last_maintenance_at <= $5)
			RETURNING level
		`, newLevel, string(newBuildings), now, s.id, cutoff).Scan(&appliedLevel)
		if errDown != nil {
			if errDown != sql.ErrNoRows {
				logger.S.Errorf("SectMaintenanceWorker: failed to downgrade sect %s: %v", s.id, errDown)
			}
			continue
		}

		alertChat := fmt.Sprintf("[Bảo Trì] CẢNH BÁO: Tông môn không đủ quỹ duy trì (cần %d), cấp bậc bị giáng xuống Cấp %d!",
			fee, appliedLevel)
		sendSectSystemChatMessage(ctx, db, s.id, alertChat)

		notifySectLeader(ctx, db, s.id, s.level, appliedLevel, curFunds, fee)
	}
}

// clampBuildings rebuilds a sect's buildings JSON after a demotion: every
// building, main_hall included, is capped at newLevel, and a missing main_hall
// key is materialised so the sect level and the hall level stay in sync.
func clampBuildings(raw []byte, newLevel int) ([]byte, error) {
	buildings := map[string]int{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &buildings); err != nil {
			return nil, err
		}
	}
	if buildings == nil {
		// JSON `null` unmarshals into a nil map; assigning to it would panic.
		buildings = map[string]int{}
	}
	for name, level := range buildings {
		if level > newLevel {
			buildings[name] = newLevel
		}
	}
	if _, ok := buildings["main_hall"]; !ok {
		buildings["main_hall"] = newLevel
	}
	return json.Marshal(buildings)
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
