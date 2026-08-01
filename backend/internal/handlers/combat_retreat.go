package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func Retreat(c *gin.Context) {
	playerIDStr := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	playerUUID, err := uuid.Parse(playerIDStr)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	logMsg := "Đạo hữu trở về an toàn từ hành trình."

	if player.ActiveCombat != nil {
		logMsg = fmt.Sprintf("Đạo hữu rút lui khỏi %s.", player.ActiveCombat.EnemyName)

		player.ActiveCombat = nil

		player.TempLoot = models.TempLoot{
			GuInventory:        models.GuInventory{},
			MaterialsInventory: models.MaterialsInventory{},
			PrimevalStones:     0,
		}
	} else {
		var partyMembers []uuid.UUID
		if player.PartyID != nil {
			var membersRaw []byte
			errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
			if errPart == nil {
				var members []PartyMember
				json.Unmarshal(membersRaw, &members)
				for _, m := range members {
					mUUID, errParse := uuid.Parse(m.ID)
					if errParse == nil {
						var sameRegion bool
						errReg := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE id = $1 AND current_region = $2)", mUUID, player.CurrentRegion).Scan(&sameRegion)
						if errReg == nil && sameRegion {
							partyMembers = append(partyMembers, mUUID)
						}
					}
				}
			}
		}

		totalStones := player.TempLoot.PrimevalStones
		totalMats := player.TempLoot.MaterialsInventory
		totalGus := player.TempLoot.GuInventory

		if totalStones == 0 && len(totalMats) == 0 && len(totalGus) == 0 {
			logMsg = "Đạo hữu trở về an toàn từ hành trình. Không thu hoạch được tài nguyên nào."
		} else if len(partyMembers) > 1 {
			shares := make(map[uuid.UUID]*models.TempLoot)
			for _, mID := range partyMembers {
				shares[mID] = &models.TempLoot{
					GuInventory:        models.GuInventory{},
					MaterialsInventory: make(models.MaterialsInventory),
					PrimevalStones:     0,
				}
			}

			if totalStones > 0 {
				splitStones := totalStones / len(partyMembers)
				remainderStones := totalStones % len(partyMembers)
				for _, mID := range partyMembers {
					shares[mID].PrimevalStones = splitStones
				}
				shares[player.ID].PrimevalStones += remainderStones
			}

			for matID, qty := range totalMats {
				if qty > 0 {
					splitQty := qty / len(partyMembers)
					remainderQty := qty % len(partyMembers)
					for _, mID := range partyMembers {
						shares[mID].MaterialsInventory[matID] = splitQty
					}
					for i := 0; i < remainderQty; i++ {
						winnerID := partyMembers[rand.Intn(len(partyMembers))]
						shares[winnerID].MaterialsInventory[matID]++
					}
				}
			}

			for _, gu := range totalGus {
				winnerID := partyMembers[rand.Intn(len(partyMembers))]
				shares[winnerID].GuInventory = append(shares[winnerID].GuInventory, gu)
			}

			player.PrimevalStones += shares[player.ID].PrimevalStones
			for itemID, qty := range shares[player.ID].MaterialsInventory {
				InventorySvc.AddItem(tx, player.ID.String(), itemID, "materials", qty)
			}
			player.GuInventory = append(player.GuInventory, shares[player.ID].GuInventory...)

			for mID, share := range shares {
				if mID == player.ID {
					continue
				}
				var targetStones int
				errLoad := tx.QueryRow(`
					SELECT primeval_stones
					FROM players WHERE id = $1 FOR UPDATE`, mID,
				).Scan(&targetStones)
				if errLoad == nil {
					targetGu, errLoadGu := db.LoadPlayerGus(tx, mID.String(), false)
					if errLoadGu != nil {
						continue
					}

					targetStones += share.PrimevalStones
					for itemID, qty := range share.MaterialsInventory {
						InventorySvc.AddItem(tx, mID.String(), itemID, "materials", qty)
					}
					targetGu = append(targetGu, share.GuInventory...)

					_, _ = tx.Exec(`
						UPDATE players 
						SET primeval_stones = $1
						WHERE id = $2`, targetStones, mID)
					_ = db.SavePlayerGus(tx, mID.String(), targetGu)
				}
			}

			logMsg = "Đạo hữu trở về an toàn từ hành trình khám phá nhóm.\n"
			logMsg += "=== TỔNG THU HOẠCH TỔ ĐỘI ===\n"
			if totalStones > 0 {
				logMsg += fmt.Sprintf("- Nguyên Thạch: %d\n", totalStones)
			}
			for itemID, qty := range totalMats {
				if qty > 0 {
					logMsg += fmt.Sprintf("- %s: %d\n", itemID, qty)
				}
			}
			for _, gu := range totalGus {
				logMsg += fmt.Sprintf("- Cổ trùng: %s\n", gu.GuID)
			}

			logMsg += "\n=== PHÂN CHIA CHI TIẾT ===\n"
			for _, mID := range partyMembers {
				var uname string
				if mID == player.ID {
					uname = "Bạn"
				} else {
					_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", mID).Scan(&uname)
				}

				share := shares[mID]
				logMsg += fmt.Sprintf("[%s] nhận được:\n", uname)
				hasLoot := false
				if share.PrimevalStones > 0 {
					logMsg += fmt.Sprintf("  + %d Nguyên Thạch\n", share.PrimevalStones)
					hasLoot = true
				}
				for itemID, qty := range share.MaterialsInventory {
					if qty > 0 {
						logMsg += fmt.Sprintf("  + %dx %s\n", qty, itemID)
						hasLoot = true
					}
				}
				for _, gu := range share.GuInventory {
					logMsg += fmt.Sprintf("  + 1x Cổ trùng %s\n", gu.GuID)
					hasLoot = true
				}
				if !hasLoot {
					logMsg += "  (Không nhận được tài nguyên nào)\n"
				}
			}
		} else {
			TransferTempLootToInventory(tx, &player)

			logMsg = "Đạo hữu trở về an toàn từ hành trình.\n"
			logMsg += "=== TÀI NGUYÊN THU HOẠCH ===\n"
			if totalStones > 0 {
				logMsg += fmt.Sprintf("- Nguyên Thạch: +%d\n", totalStones)
			}
			for itemID, qty := range totalMats {
				if qty > 0 {
					logMsg += fmt.Sprintf("- %s: +%d\n", itemID, qty)
				}
			}
			for _, gu := range totalGus {
				logMsg += fmt.Sprintf("- Cổ trùng: %s\n", gu.GuID)
			}
		}

		player.TempLoot = models.TempLoot{
			GuInventory:        models.GuInventory{},
			MaterialsInventory: models.MaterialsInventory{},
			PrimevalStones:     0,
		}
	}

	player.X = 50
	player.Y = 50
	player.ExploredCells = models.ExploredCellsList{"50,50"}
	player.CurrentRegion = "Southern_Border"

	qiGained := AddHumanQi(&player, 2.0)

	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, active_combat = $3, x = $4, y = $5, explored_cells = $6, primeval_stones = $7,
		    temp_loot = $8, current_region = $9, human_qi = $10
		WHERE id = $11
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.ActiveCombat, player.X, player.Y, player.ExploredCells, player.PrimevalStones,
		player.TempLoot, player.CurrentRegion, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player status after retreat"})
		return
	}

	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player Gus after retreat"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "combat.retreat",
		RequestSummary: fmt.Sprintf("hadCombat=%v stones=%d gus=%d mats=%d", player.ActiveCombat != nil, player.TempLoot.PrimevalStones, len(player.TempLoot.GuInventory), len(player.TempLoot.MaterialsInventory)),
		IPAddress: c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"log":    logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

func Flee(c *gin.Context) {
	playerIDStr := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	playerUUID, err := uuid.Parse(playerIDStr)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có trận chiến nào để rút lui."})
		return
	}

	logMsg := fmt.Sprintf("Đạo hữu đã bỏ chạy khỏi %s.", player.ActiveCombat.EnemyName)
	player.ActiveCombat = nil

	qiGained := AddHumanQi(&player, 1.0)

	updateQuery := `
		UPDATE players
		SET active_combat = $1, human_qi = $2
		WHERE id = $3
	`
	_, err = tx.Exec(updateQuery, player.ActiveCombat, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player status after fleeing"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	auditlog.Logf(db.DB, &authID, "combat.flee", "enemy=%s", logMsg)
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"log":    logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

