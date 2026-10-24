package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// immortalGuLootAnnouncement is a world-chat announcement (Tiên Cổ xuất thế)
// queued by the loot code and published by the caller only after the
// surrounding transaction commits.
type immortalGuLootAnnouncement struct {
	PlayerID   uuid.UUID
	PlayerName string
	BossName   string
	GuID       string
	Rank       int
}

// immortalRankOf returns the rank >= 6 of guID, either the rolled rank or — for
// Gu ids that also have mortal variants — the highest immortal variant in the
// registry. Returns 0 when the Gu is not a Tiên Cổ.
func immortalRankOf(guID string, rank int) int {
	if rank >= 6 {
		return rank
	}
	if !models.HasImmortalRank(guID) {
		return 0
	}
	best := 0
	for r := range models.GuRegistry[guID] {
		if r >= 6 && r > best {
			best = r
		}
	}
	return best
}

// queueImmortalGuLootNotice appends a world announcement when the looted Gu is
// a Tiên Cổ. Broadcasting happens after commit (see CombatResolution.AfterCommit)
// so a rollback can never announce a Gu that was never actually added.
func queueImmortalGuLootNotice(out []immortalGuLootAnnouncement, playerID uuid.UUID, playerName, bossName, guID string, rank int) []immortalGuLootAnnouncement {
	immortalRank := immortalRankOf(guID, rank)
	if immortalRank == 0 {
		return out
	}
	return append(out, immortalGuLootAnnouncement{
		PlayerID:   playerID,
		PlayerName: playerName,
		BossName:   bossName,
		GuID:       guID,
		Rank:       immortalRank,
	})
}

// applyCombatVictoryLoot rolls and distributes loot after a won auto-combat.
//
// This is the single implementation shared by AutoCombatStart (manual/replay)
// and ResolveCombat (skip battle), using the AutoCombatStart semantics:
// PVP drop record + loot-table roll + Gu-drop chance, with random per-item
// winners when in a party and direct inventory grants when solo.
//
// Looted Tiên Cổ (Rank >= 6) are returned as announcements instead of being
// broadcast here: the caller must publish them only once its transaction has
// committed.
func applyCombatVictoryLoot(tx *sql.Tx, player *models.Player, activeCombat *models.CombatState, partyMembers []uuid.UUID, isVirtual bool) (lootDetailLogs []string, pendingNotifications []notify.Notification, immortalAnnouncements []immortalGuLootAnnouncement) {
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
					dropSummary := applyDeathDropRecord(tx, &targetUUID, &player.ID, true, DeathPos{Region: targetRegion, X: targetX, Y: targetY})
					respawn := ResolveDeathRespawn(targetRegion, targetX, targetY)
					hubRegion, hubX, hubY := respawn.Region, respawn.X, respawn.Y
					GrantPvpImmunity(tx, targetUUID, 10*time.Minute)
					worldPosIndex.Set(WorldPosEntry{
						ID:     targetUUID.String(),
						X:      hubX,
						Y:      hubY,
						Region: hubRegion,
						TS:     time.Now().UnixMilli(),
					})
					_, _ = tx.Exec("UPDATE players SET hp = $1, current_region = $2, x = $3, y = $4, active_combat = NULL WHERE id = $5", targetHP, hubRegion, hubX, hubY, targetUUID)
					if dropSummary != "" {
						pvpMsg := fmt.Sprintf("Chiến thắng PVP vang dội! Rơi rớt từ bại binh: %s.", dropSummary)
						lootDetailLogs = append(lootDetailLogs, pvpMsg)
						if n, err := notify.SendDeferred(tx, player.ID, "combat_loot", "Chiến lợi phẩm PVP", pvpMsg); err == nil {
							pendingNotifications = append(pendingNotifications, n)
						}
					} else {
						pvpMsg := "Chiến thắng PVP vang dội! Bại binh không có tài sản rơi rớt."
						lootDetailLogs = append(lootDetailLogs, pvpMsg)
						if n, err := notify.SendDeferred(tx, player.ID, "combat_loot", "Chiến lợi phẩm PVP", pvpMsg); err == nil {
							pendingNotifications = append(pendingNotifications, n)
						}
					}
				}
			}
		}
	} else if activeCombat != nil && activeCombat.LootTableID != "" {
		loot, rollErr := RollMonsterLoot(tx, activeCombat.LootTableID, activeCombat.EnemyRank, activeCombat.WorldMonsterRegionID, activeCombat.EnemyType)
		if rollErr != nil {
			lootDetailLogs = append(lootDetailLogs, "Không thể roll chiến lợi phẩm.")
		}

		if len(activeCombat.EnemyGuInventory) > 0 {
			for _, eg := range activeCombat.EnemyGuInventory {
				guSpec, _ := models.GetGuAny(eg.GuID)
				dropChance := GetGuDropChancePercent(guSpec.Rank, guDropTierForCombat(activeCombat))
				if rand.Float64()*100.0 < dropChance {
					isImmortal := guSpec.Rank >= 6 || models.HasImmortalRank(eg.GuID)
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

		// Boss name used by the Tiên Cổ world announcement, mirroring ApplyLoot.
		lootBossName := "Thái Cổ Yêu Quái"
		if activeCombat.EnemyName != "" {
			lootBossName = activeCombat.EnemyName
		}

		if len(partyMembers) > 1 {
			lootDetailLogs = append(lootDetailLogs, "=== PHÂN CHIA CHIẾN LỢI PHẨM TỔ ĐỘI ===")
			stoneQty := loot.Items["primeval_stone"]
			if stoneQty > 0 {
				splitStones := stoneQty / len(partyMembers)
				remaining := stoneQty
				for _, memberID := range partyMembers {
					if remaining <= 0 {
						break
					}
					give := splitStones
					if give < 1 {
						give = 1
					}
					if give > remaining {
						give = remaining
					}
					remaining -= give
					if memberID == player.ID {
						player.PrimevalStones += float64(give)
						lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch.", give))
					} else {
						_, _ = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", give, memberID)
						var uname string
						_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", memberID).Scan(&uname)
						lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s nhận được %d Nguyên Thạch.", uname, give))
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
				winnerName := player.Username
				newGu := models.GuInstance{
					InstanceID: uuid.New().String(),
					GuID:       gID,
					Rank:       guSpec.Rank,
					Hunger:     0,
					Status:     "ACTIVE",
					Equipped:   false,
				}
				if winnerID == player.ID {
					player.GuInventory = append(player.GuInventory, newGu)
					lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn bắt được 1x %s!", guName))
				} else {
					_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&winnerName)
					if err := db.AddPlayerGu(tx, winnerID.String(), newGu); err != nil {
						if db.IsImmortalConflict(err) {
							lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Tiên Cổ %s đã tồn tại trong thiên hạ, không thể thu phục.", guName))
							continue
						}
						logger.S.Errorf("Failed to distribute loot Gu %s to %s: %v", gID, winnerID, err)
						continue
					}
					lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s thu phục thành công Cổ trùng: %s!", winnerName, guName))
				}
				_ = db.LogGuHistory(tx, newGu.InstanceID, newGu.GuID, newGu.Rank, "CREATED_LOOT", &winnerID, nil,
					fmt.Sprintf("Thu phục Cổ trùng [%s] R%d sau chiến thắng.", guName, newGu.Rank),
					map[string]interface{}{"gu_name": guName})

				immortalAnnouncements = queueImmortalGuLootNotice(immortalAnnouncements, winnerID, winnerName, lootBossName, gID, guSpec.Rank)
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
				_ = db.LogGuHistory(tx, newGu.InstanceID, newGu.GuID, newGu.Rank, "CREATED_LOOT", &player.ID, nil,
					fmt.Sprintf("Thu phục Cổ trùng [%s] R%d sau chiến thắng.", guSpec.Name, newGu.Rank),
					map[string]interface{}{"gu_name": guSpec.Name})

				immortalAnnouncements = queueImmortalGuLootNotice(immortalAnnouncements, player.ID, player.Username, lootBossName, guID, guSpec.Rank)
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
				if n, err := notify.SendDeferred(tx, player.ID, "combat_loot", "Chiến lợi phẩm", lootMsg); err == nil {
					pendingNotifications = append(pendingNotifications, n)
				}
			}
		}
	}
	return lootDetailLogs, pendingNotifications, immortalAnnouncements
}
