package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// applyCombatVictoryLoot rolls and distributes loot after a won auto-combat.
//
// This is the single implementation shared by AutoCombatStart (manual/replay)
// and AutoResolveCombat (skip battle), using the AutoCombatStart semantics:
// PVP drop record + loot-table roll + Gu-drop chance, with random per-item
// winners when in a party and direct inventory grants when solo.
func applyCombatVictoryLoot(tx *sql.Tx, player *models.Player, activeCombat *models.CombatState, partyMembers []uuid.UUID, isVirtual bool) (lootDetailLogs []string) {
	if activeCombat != nil && activeCombat.EnemyType == "PVP" && !isVirtual {
		if activeCombat.TargetPlayerID != "" && activeCombat.TargetPlayerID != "dummy" {
			targetUUID, parseErr := uuid.Parse(activeCombat.TargetPlayerID)
			if parseErr == nil {
				var targetHP float64
				var targetX, targetY int
				var targetRegion string
				if errHP := tx.QueryRow("SELECT hp, x, y, current_region FROM players WHERE id = $1 FOR UPDATE", targetUUID).Scan(&targetHP, &targetX, &targetY, &targetRegion); errHP == nil {
					targetHP -= 30.0
					if targetHP < 10.0 {
						targetHP = 10.0
					}
					if activeCombat.MineSlotID <= 0 {
						dropSummary := applyDeathDropRecord(tx, &targetUUID, &player.ID, true, DeathPos{Region: targetRegion, X: targetX, Y: targetY})
						hubRegion, hubX, hubY := NearestHub(targetX, targetY)
						_, _ = tx.Exec("UPDATE players SET hp = $1, current_region = $2, x = $3, y = $4, active_combat = NULL WHERE id = $5", targetHP, hubRegion, hubX, hubY, targetUUID)
						if dropSummary != "" {
							pvpMsg := fmt.Sprintf("Chiến thắng PVP vang dội! Rơi rớt từ bại binh: %s.", dropSummary)
							lootDetailLogs = append(lootDetailLogs, pvpMsg)
							notify.Send(tx, player.ID, "combat_loot", "Chiến lợi phẩm PVP", pvpMsg)
						} else {
							pvpMsg := "Chiến thắng PVP vang dội! Bại binh không có tài sản rơi rớt."
							lootDetailLogs = append(lootDetailLogs, pvpMsg)
							notify.Send(tx, player.ID, "combat_loot", "Chiến lợi phẩm PVP", pvpMsg)
						}
					} else {
						_, _ = tx.Exec("UPDATE players SET hp = $1, active_combat = NULL WHERE id = $2", targetHP, targetUUID)
						pvpMsg := "Đánh chiếm Mỏ thành công! Bại binh bị tước quyền chiếm giữ."
						lootDetailLogs = append(lootDetailLogs, pvpMsg)
					}
				}
			}
		}
	} else if activeCombat != nil && activeCombat.LootTableID != "" {
		lt, err := db.LoadLootTable(tx, activeCombat.LootTableID)
		if err == nil {
			loot := RollLoot(lt, activeCombat.EnemyRank)

			if len(activeCombat.EnemyGuInventory) > 0 {
				for _, eg := range activeCombat.EnemyGuInventory {
					guSpec, _ := models.GetGuAny(eg.GuID)
					dropChance := GetGuDropChancePercent(guSpec.Rank, isBossEnemy(activeCombat))
					if rand.Float64()*100.0 < dropChance {
						isImmortal := guSpec.IsImmortal || guSpec.Rank >= 6 || models.HasImmortalRank(eg.GuID)
						if isImmortal {
							exists, err := db.CheckImmortalGuExists(tx, eg.GuID, guSpec.Rank, nil)
							if err == nil && exists {
								continue
							}
						}
						loot.Gus = append(loot.Gus, eg.GuID)
					}
				}
			}

			if len(partyMembers) > 1 {
				lootDetailLogs = append(lootDetailLogs, "=== PHÂN CHIA CHIẾN LỢI PHẨM TỔ ĐỘI ===")
				stoneQty := loot.Items["primeval_stone"]
				if stoneQty > 0 {
					splitStones := stoneQty / len(partyMembers)
					if splitStones < 1 {
						splitStones = 1
					}
					for _, memberID := range partyMembers {
						if memberID == player.ID {
							player.PrimevalStones += float64(splitStones)
							lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch.", splitStones))
						} else {
							_, _ = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", splitStones, memberID)
							var uname string
							_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", memberID).Scan(&uname)
							lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s nhận được %d Nguyên Thạch.", uname, splitStones))
						}
					}
				}

				for matID, qty := range loot.Items {
					if matID == "primeval_stone" {
						continue
					}
					itemName := matID
					if item, ok := models.ItemRegistry[matID]; ok {
						itemName = item.Name
					}
					for q := 0; q < qty; q++ {
						winnerID := partyMembers[rand.Intn(len(partyMembers))]
						if winnerID == player.ID {
							if err := InventorySvc.AddItem(tx, player.ID.String(), matID, "materials", 1); err != nil {
								logger.S.Errorf("Failed to add loot item %s to player %s: %v", matID, player.ID, err)
							}
							lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn nhặt được 1x %s.", itemName))
						} else {
							var uname string
							_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)
							if err := InventorySvc.AddItem(tx, winnerID.String(), matID, "materials", 1); err != nil {
								logger.S.Errorf("Failed to distribute loot %s to party member %s: %v", matID, winnerID, err)
							}
							lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s nhặt được 1x %s.", uname, itemName))
						}
					}
				}

				for _, gID := range loot.Gus {
					winnerID := partyMembers[rand.Intn(len(partyMembers))]
					guSpec, _ := models.GetGuAny(gID)
					guName := gID
					if guSpec.Name != "" {
						guName = guSpec.Name
					}
					newGu := models.GuInstance{
						InstanceID: uuid.New().String(),
						GuID:       gID,
						Rank:       guSpec.Rank,
						Hunger:     100,
						Status:     "ACTIVE",
						Equipped:   false,
					}
					if winnerID == player.ID {
						player.GuInventory = append(player.GuInventory, newGu)
						lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn bắt được 1x %s!", guName))
					} else {
						var uname string
						_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)
						_ = db.AddPlayerGu(tx, winnerID.String(), newGu)
						lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s thu phục thành công Cổ trùng: %s!", uname, guName))
					}
				}
			} else {
				stoneQty := loot.Items["primeval_stone"]
				player.PrimevalStones += float64(stoneQty)
				for matID, qty := range loot.Items {
					if matID == "primeval_stone" {
						continue
					}
					InventorySvc.AddItem(tx, player.ID.String(), matID, "materials", int64(qty))
				}
				for _, guID := range loot.Gus {
					guSpec, _ := models.GetGuAny(guID)
					newGu := models.GuInstance{
						InstanceID: uuid.New().String(), GuID: guID, Rank: guSpec.Rank, Hunger: 0, Status: "ACTIVE", Equipped: false, IsVital: false,
					}
					player.GuInventory = append(player.GuInventory, newGu)
				}

				var lootParts []string
				if stoneQty > 0 {
					lootParts = append(lootParts, fmt.Sprintf("%d Nguyên Thạch", stoneQty))
				}
				for matID, qty := range loot.Items {
					if matID == "primeval_stone" || qty <= 0 {
						continue
					}
					itemName := matID
					if item, ok := models.ItemRegistry[matID]; ok {
						itemName = item.Name
					}
					lootParts = append(lootParts, fmt.Sprintf("%dx %s", qty, itemName))
				}
				for _, guID := range loot.Gus {
					guName := guID
					if guSpec, ok := models.GetGuAny(guID); ok {
						guName = guSpec.Name
					}
					lootParts = append(lootParts, fmt.Sprintf("1x %s (Cổ trùng)", guName))
				}

				if len(lootParts) > 0 {
					lootMsg := "Nhặt được: " + strings.Join(lootParts, ", ") + "."
					lootDetailLogs = append(lootDetailLogs, lootMsg)
					notify.Send(tx, player.ID, "combat_loot", "Chiến lợi phẩm", lootMsg)
				}
			}
		}
	}
	return lootDetailLogs
}
