package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// applyAutoCombatMineOccupation handles mine PvE victory: robs the previous
// occupier, rewards the player, and takes the slot. Shared by AutoCombatStart
// and AutoResolveCombat so mine resolution is identical.
// Returns the generated logs and whether the slot was successfully occupied.
func applyAutoCombatMineOccupation(tx *sql.Tx, player *models.Player, activeCombat *models.CombatState, turn int) (mineLogs []simulator.CombatLog, occupied bool) {
	if activeCombat == nil || activeCombat.MineSlotID <= 0 {
		return nil, false
	}
	slotID := activeCombat.MineSlotID
	var rarity string
	var mineType string
	var currentOccupierStr *string
	var monsterIDStr *string
	var lastClaimedAt *time.Time
	err := tx.QueryRow("SELECT rarity, mine_type, occupier_id, monster_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE NOWAIT", slotID).Scan(&rarity, &mineType, &currentOccupierStr, &monsterIDStr, &lastClaimedAt)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			logger.S.Warnf("applyAutoCombatMineOccupation: mine slot %d is locked by concurrent transaction, skipping occupation", slotID)
			return nil, false
		}
		if err != sql.ErrNoRows {
			logger.S.Errorf("Failed to query mine slot %d: %v", slotID, err)
			return nil, false
		}
	} else if currentOccupierStr != nil {
		if defenderUUID, errParse := uuid.Parse(*currentOccupierStr); errParse == nil && defenderUUID != player.ID {
			notify.Send(tx, defenderUUID, "mine_lost", "Mỏ bị đánh chiếm!",
				fmt.Sprintf("Mỏ #%d (%s) của bạn đã bị đạo hữu [%s] tấn công và đánh chiếm thành công!", slotID, rarity, player.Username))
		}
		if lastClaimedAt != nil {
			stones, qi, chests := CalculateMineRewards(slotID, rarity, *lastClaimedAt)
			stonesHalf := stones / 2
			qiHalf := qi / 2.0
			chestsHalf := chests / 2
			if stonesHalf > 0 || qiHalf > 0.0 || chestsHalf > 0 {
				if errMine := AddMineRewardsToPlayerTx(tx, player.ID, stonesHalf, qiHalf, chestsHalf, mineType); errMine != nil {
					logger.S.Errorf("Failed to add mine rewards to player %s (slot %d): %v", player.ID, slotID, errMine)
					return mineLogs, false
				} else {
					if mineType == "immortal" {
						player.ImmortalStones += float64(stonesHalf)
					} else {
						player.PrimevalStones += float64(stonesHalf)
					}
					var stoneName = "Nguyên Thạch"
					if mineType == "immortal" {
						stoneName = "Tiên Nguyên Thạch"
					}
					mineLootMsg := fmt.Sprintf("Cướp mỏ thành công! +%d %s, +%.0f Nhân Khí, +%d Rương Mỏ", stonesHalf, stoneName, qiHalf, chestsHalf)
					notify.Send(tx, player.ID, "combat_loot", "Chiến Lợi Phẩm (Cướp Mỏ)", mineLootMsg)
					mineLogs = append(mineLogs, simulator.CombatLog{
						Turn:   turn,
						Action: "MINE_LOOT",
						Detail: mineLootMsg,
					})
				}
			} else {
				mineLogs = append(mineLogs, simulator.CombatLog{
					Turn:   turn,
					Action: "MINE_LOOT",
					Detail: "Mỏ đối thủ chưa tích lũy đủ tài nguyên để cướp.",
				})
			}
		}
	} else {
		mineLogs = append(mineLogs, simulator.CombatLog{
			Turn:   turn,
			Action: "MINE_LOOT",
			Detail: "Mỏ không có người chiếm giữ, không thể cướp tài nguyên.",
		})
	}

	now := time.Now()
	_, errUpdate := tx.Exec(`
					UPDATE mine_slots
					SET occupier_id = $1, monster_id = NULL, occupied_at = $2, last_claimed_at = $3
					WHERE id = $4
				`, player.ID, now, now, slotID)
	if errUpdate == nil {
		mineLogs = append(mineLogs, simulator.CombatLog{
			Turn:   turn,
			Action: "MINE_OCCUPY",
			Detail: fmt.Sprintf("Chiếm lĩnh thành công mỏ phẩm chất %s #%d!", rarity, slotID),
		})
		occupied = true
	} else {
		logger.S.Errorf("Failed to update mine slot in auto combat victory: %v", errUpdate)
	}
	return mineLogs, occupied
}
