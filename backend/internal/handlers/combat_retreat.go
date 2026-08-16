package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// isTimeoutOrBusy reports whether an error is a context deadline/cancel, a
// Postgres lock timeout (55P03), a canceled statement (57014) or a deadlock
// (40P01) — all of which the client should retry rather than treat as fatal.
func isTimeoutOrBusy(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "55P03", "57014", "40P01":
			return true
		}
	}
	return false
}

// beginCombatTx opens a bounded transaction whose lock waits cannot hang
// indefinitely: the context caps the whole request and SET LOCAL lock_timeout
// aborts any row-lock wait after 5 seconds. The shared busy message is used by
// combat mutation handlers so clients surface a retriable error.
// On failure it responds itself and returns (nil, nil).
func beginCombatTx(c *gin.Context) (*sql.Tx, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		cancel()
		if isTimeoutOrBusy(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Hệ thống đang bận — quá nhiều thao tác cùng lúc, đạo hữu thử lại chút nữa nhé."})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		}
		return nil, nil
	}
	if _, errSet := tx.Exec("SET LOCAL lock_timeout = '5000'"); errSet != nil {
		logger.S.Warnf("beginCombatTx: failed to SET LOCAL lock_timeout: %v", errSet)
	}
	return tx, cancel
}

// respondCombatTxErr maps transaction errors to a friendly client response.
func respondCombatTxErr(c *gin.Context, err error, fallback string) {
	if err == nil {
		return
	}
	if isTimeoutOrBusy(err) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Hệ thống đang bận — quá nhiều thao tác cùng lúc, đạo hữu thử lại chút nữa nhé."})
		return
	}
	msg := fallback
	if msg == "" {
		msg = "Giao dịch thất bại, vui lòng thử lại."
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
}

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

	tx, cancel := beginCombatTx(c)
	if tx == nil {
		return
	}
	defer cancel()
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to lock player Aperture")
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	logMsg := "Đạo hữu trở về an toàn từ hành trình."

	hadCombat := player.ActiveCombat != nil
	originalStones := player.TempLoot.PrimevalStones
	originalGus := len(player.TempLoot.GuInventory)
	originalMats := len(player.TempLoot.MaterialsInventory)

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
				splitStones := totalStones / float64(len(partyMembers))
				remainderStones := totalStones - splitStones*float64(len(partyMembers))
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
			InventorySvc.BulkAddItems(tx, player.ID.String(), "materials", shares[player.ID].MaterialsInventory)
			player.GuInventory = append(player.GuInventory, shares[player.ID].GuInventory...)

			for mID, share := range shares {
				if mID == player.ID {
					continue
				}
				var targetStones float64
				errLoad := tx.QueryRow(`
					SELECT primeval_stones
					FROM players WHERE id = $1 FOR UPDATE`, mID,
				).Scan(&targetStones)
				if errLoad == nil {
					targetStones += share.PrimevalStones
					InventorySvc.BulkAddItems(tx, mID.String(), "materials", share.MaterialsInventory)

					_, _ = tx.Exec(`
						UPDATE players 
						SET primeval_stones = $1
						WHERE id = $2`, targetStones, mID)
					_ = db.AddPlayerGusBulk(tx, mID.String(), share.GuInventory)
				}
			}

			logMsg = "Đạo hữu trở về an toàn từ hành trình khám phá nhóm.\n"
			logMsg += "=== TỔNG THU HOẠCH TỔ ĐỘI ===\n"
			if totalStones > 0 {
				logMsg += fmt.Sprintf("- Nguyên Thạch: %.0f\n", totalStones)
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
					logMsg += fmt.Sprintf("  + %.0f Nguyên Thạch\n", share.PrimevalStones)
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
				logMsg += fmt.Sprintf("- Nguyên Thạch: +%.0f\n", totalStones)
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

	player.CurrentRegion = HubRegionID
	player.X, player.Y = hubEntryPoint()

	qiGained := AddHumanQi(&player, 2.0)

	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, active_combat = $3, x = $4, y = $5, primeval_stones = $6,
		    temp_loot = $7, current_region = $8, human_qi = $9
		WHERE id = $10
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.ActiveCombat, player.X, player.Y, player.PrimevalStones,
		player.TempLoot, player.CurrentRegion, player.HumanQi, player.ID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to save player status after retreat")
		return
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		respondCombatTxErr(c, err, "Failed to save player Gus after retreat")
		return
	}

	if err = tx.Commit(); err != nil {
		respondCombatTxErr(c, err, "Transaction commit failed")
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "combat.retreat",
		RequestSummary: fmt.Sprintf("hadCombat=%v stones=%.0f gus=%d mats=%d", hadCombat, originalStones, originalGus, originalMats),
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

	tx, cancel := beginCombatTx(c)
	if tx == nil {
		return
	}
	defer cancel()
	defer tx.Rollback()

	// Flee only needs equipped stats (EffectiveMaxHP) plus HP/mana/temp_loot —
	// never the full inventory — so use the light loader: no goroutine fetch
	// of the full ~12k Gu inventory while the row is locked.
	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to lock player Aperture")
		return
	}

	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có trận chiến nào để rút lui."})
		return
	}

	enemyName := player.ActiveCombat.EnemyName
	var logMsg string
	var defeatAudit *auditlog.Entry

	isExploration := player.ActiveCombat.CombatMode != "virtual" && player.ActiveCombat.EnemyTemplateID != "monster_heavenly_tribulation"

	if isExploration {
		maxHP := gamefx.EffectiveMaxHP(&player)
		damage := maxHP * 0.20 // Lose 20% Max HP
		player.HP -= damage

		if player.HP <= 0 {
			player.HP = 1.0
			player.CurrentRegion = HubRegionID
			player.X, player.Y = hubEntryPoint()

			// Snapshot the loss before clearing; the audit insert runs AFTER
			// commit so the transaction stays short (no second pool connection
			// held while the row is locked).
			defeatAudit = &auditlog.Entry{
				PlayerID:       &authID,
				Action:         "combat.flee_death",
				RequestSummary: fmt.Sprintf("Lost temp_loot: stones=%.0f gus=%d mats=%d", player.TempLoot.PrimevalStones, len(player.TempLoot.GuInventory), len(player.TempLoot.MaterialsInventory)),
				IPAddress:      c.ClientIP(),
				Success:        true,
			}

			player.TempLoot = models.TempLoot{
				GuInventory:        models.GuInventory{},
				MaterialsInventory: models.MaterialsInventory{},
				PrimevalStones:     0,
			}
			logMsg = fmt.Sprintf("Đạo hữu cố gắng bỏ chạy khỏi %s nhưng bị thương nặng ngất xỉu. Tỉnh dậy đã thấy ở an toàn nhưng mất sạch tài nguyên thu thập.", enemyName)
		} else {
			logMsg = fmt.Sprintf("Đạo hữu đã bỏ chạy khỏi %s. Chịu thương tổn %.0f HP.", enemyName, damage)
		}
	} else {
		logMsg = fmt.Sprintf("Đạo hữu đã bỏ chạy khỏi %s.", enemyName)
	}

	player.ActiveCombat = nil

	qiGained := AddHumanQi(&player, 1.0)

	updateQuery := `
		UPDATE players
		SET active_combat = $1, human_qi = $2, hp = $3, current_region = $4, x = $5, y = $6, temp_loot = $7
		WHERE id = $8
	`
	_, err = tx.Exec(updateQuery, player.ActiveCombat, player.HumanQi, player.HP, player.CurrentRegion, player.X, player.Y, player.TempLoot, player.ID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to update player status after fleeing")
		return
	}

	if err = tx.Commit(); err != nil {
		respondCombatTxErr(c, err, "Transaction commit failed")
		return
	}

	// Best-effort audit logs AFTER commit — never inside the transaction.
	if defeatAudit != nil {
		auditlog.Log(db.DB, *defeatAudit)
	}
	auditlog.Logf(db.DB, &authID, "combat.flee", "enemy=%s", logMsg)
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"log":    logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

