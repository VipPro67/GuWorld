package handlers

import (
	"context"
	"database/sql"
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
	// The default 15s statement_timeout (from the connection string) can abort
	// large bulk Gu inventory writes (e.g. a retreat that harvests many Gus).
	// Raise it for combat mutation flows; the surrounding context still caps the
	// whole request at 30s.
	if _, errSet := tx.Exec("SET LOCAL statement_timeout = '60000'"); errSet != nil {
		logger.S.Warnf("beginCombatTx: failed to SET LOCAL statement_timeout: %v", errSet)
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
	inventoryBefore := len(before)

	logMsg := "Đạo hữu trở về an toàn từ hành trình."

	hadCombat := player.ActiveCombat != nil

	if player.ActiveCombat != nil {
		logMsg = fmt.Sprintf("Đạo hữu rút lui khỏi %s.", player.ActiveCombat.EnemyName)
		player.ActiveCombat = nil
	}

	player.CurrentRegion, player.X, player.Y = NearestHub(player.X, player.Y)

	qiGained := AddHumanQi(&player, 2.0)

	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, active_combat = $3, x = $4, y = $5, primeval_stones = $6,
		    current_region = $7, human_qi = $8
		WHERE id = $9
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.ActiveCombat, player.X, player.Y, player.PrimevalStones,
		player.CurrentRegion, player.HumanQi, player.ID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to save player status after retreat")
		return
	}

	var dbBefore int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1`, player.ID).Scan(&dbBefore)

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		respondCombatTxErr(c, err, "Failed to save player Gus after retreat")
		return
	}

	inventoryAfter := len(player.GuInventory)
	var dbAfterTx int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1`, player.ID).Scan(&dbAfterTx)

	if err = tx.Commit(); err != nil {
		respondCombatTxErr(c, err, "Transaction commit failed")
		return
	}

	setWorldPosFromPlayer(&player)

	var dbAfter int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1`, player.ID).Scan(&dbAfter)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "combat.retreat",
		RequestSummary: fmt.Sprintf("hadCombat=%v invBefore=%d invAfter=%d added=%d dbBefore=%d dbAfterTx=%d dbAfter=%d",
			hadCombat, inventoryBefore, inventoryAfter, inventoryAfter-inventoryBefore, dbBefore, dbAfterTx, dbAfter),
		IPAddress: c.ClientIP(), Success: true,
	})
	logger.S.Infof("Retreat player=%s invBefore=%d invAfter=%d added=%d dbBefore=%d dbAfterTx=%d dbAfter=%d",
		player.ID, inventoryBefore, inventoryAfter, inventoryAfter-inventoryBefore, dbBefore, dbAfterTx, dbAfter)
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"log": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
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

	// Flee only needs equipped stats (EffectiveMaxHP) plus HP/mana — never the
	// full inventory — so use the light loader: no goroutine fetch of the full
	// ~12k Gu inventory while the row is locked. Death drops are applied via
	// applyDeathDropRecord, which loads the full inventory on demand.
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

	fleeChance := 0.70
	if player.ActiveCombat.EnemyType == "BOSS" || player.ActiveCombat.EnemyType == "TRIBULATION_BOSS" || player.ActiveCombat.EnemyType == "TRIBULATION_MONSTER" {
		fleeChance = 0.40
	}
	roll := rand.Float64()
	success := roll < fleeChance

	if !success {
		// Initialize allies/enemies if not already done
		if len(player.ActiveCombat.Allies) == 0 {
			player.ActiveCombat = InitializeCombatState(tx, player.ActiveCombat, &player)
		}

		var turnLogs []string
		turnLogs = append(turnLogs, fmt.Sprintf("Đạo hữu cố gắng rút lui khỏi %s thất bại! Tỷ lệ thành công: %.0f%%. Địch nhân thừa cơ tấn công!", enemyName, fleeChance*100))

		// Execute enemy actions
		for eIdx := range player.ActiveCombat.Enemies {
			enemy := &player.ActiveCombat.Enemies[eIdx]
			if enemy.HP <= 0 {
				continue
			}
			targetsIdx := ResolveTargets(player.ActiveCombat.Allies, "SINGLE", 2, 3, false)
			if len(targetsIdx) > 0 {
				targetAlly := &player.ActiveCombat.Allies[targetsIdx[0]]
				enemyCtx := &CombatContext{
					Attacker: enemy,
					Defender: targetAlly,
					Logs:     []string{},
				}
				enemyActions := BuildEnemyActions(enemyCtx, enemy)
				if len(enemyActions) > 0 {
					act := enemyActions[0]
					if act.ManaCost > 0 {
						if player.ActiveCombat.EnemyType != "PVP" && enemy.Kind != "Player" {
							hpCost := enemy.MaxHP * 0.05
							if hpCost < 1.0 {
								hpCost = 1.0
							}
							enemy.HP -= hpCost
							turnLogs = append(turnLogs, fmt.Sprintf("-> %s tiêu hao sinh lực (Mất %.1f HP) để thúc dục Cổ ký sinh!", enemy.Name, hpCost))
							if enemy.HP <= 0 {
								turnLogs = append(turnLogs, fmt.Sprintf("-> %s bị Cổ ký sinh vắt kiệt sinh mệnh đến chết!", enemy.Name))
								continue
							}
						} else if enemy.CurrentMana >= act.ManaCost {
							enemy.CurrentMana -= act.ManaCost
						}
					}

					isHealing := act.Def > 0 && (act.IsUtility || (len(act.StatusEffects) == 0 && act.Dmg == 0))
					isAttack := act.Dmg > 0 || len(act.StatusEffects) > 0

					if isHealing {
						healVal := act.Def
						enemy.HP += healVal
						if enemy.HP > enemy.MaxHP {
							enemy.HP = enemy.MaxHP
						}
						turnLogs = append(turnLogs, fmt.Sprintf("-> %s kích hoạt phòng ngự/trị liệu %s, hồi %.1f HP.", enemy.Name, act.Name, healVal))
					} else if isAttack {
						netDmg := act.Dmg - targetAlly.BaseDefense
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						dmgLog := applyDamageWithLogs(enemy.Name, targetAlly.Name, targetAlly, netDmg)
						turnLogs = append(turnLogs, fmt.Sprintf("-> %s kích hoạt %s tấn công!", enemy.Name, act.Name), dmgLog)
					} else {
						turnLogs = append(turnLogs, fmt.Sprintf("-> %s kích hoạt %s (Hỗ trợ)!", enemy.Name, act.Name))
					}
				} else {
					dmgVal := enemy.BaseAttack
					if dmgVal <= 0 {
						dmgVal = 5.0
					}
					netDmg := dmgVal - targetAlly.BaseDefense
					if netDmg < 1.0 {
						netDmg = 1.0
					}
					dmgLog := applyDamageWithLogs(enemy.Name, targetAlly.Name, targetAlly, netDmg)
					turnLogs = append(turnLogs, fmt.Sprintf("-> %s đánh thường!", enemy.Name), dmgLog)
				}
			}
		}

		// Update player and allies HP
		casterIdx := -1
		for idx, ally := range player.ActiveCombat.Allies {
			if ally.ID == player.ID.String() {
				casterIdx = idx
				break
			}
		}
		if casterIdx >= 0 {
			player.HP = player.ActiveCombat.Allies[casterIdx].HP
			player.CurrentMana = player.ActiveCombat.Allies[casterIdx].CurrentMana
		}

		for _, ally := range player.ActiveCombat.Allies {
			if ally.ID != player.ID.String() {
				if ally.Kind == "Pet" {
					if ally.HP <= 0 {
						_, _ = tx.Exec(`DELETE FROM player_pets WHERE id = $1 AND player_id = $2`, ally.ID, player.ID)
					} else {
						_, _ = tx.Exec(`UPDATE player_pets SET hp = $1, updated_at = NOW() WHERE id = $2 AND player_id = $3`,
							ally.HP, ally.ID, player.ID)
					}
					continue
				}
				allyUUID, errParse := uuid.Parse(ally.ID)
				if errParse == nil {
					allyHP := ally.HP
					if allyHP < 0 {
						allyHP = 0
					}
					_, _ = tx.Exec("UPDATE players SET hp = $1 WHERE id = $2", allyHP, allyUUID)
				}
			}
		}

		defeat := player.HP <= 0
		var defeatAudit *auditlog.Entry

		if defeat {
			turnLogs = append(turnLogs, "=== THẤT BẠI THẢM HẠI ===")
			isExploration := player.ActiveCombat.CombatMode != "virtual" && player.ActiveCombat.EnemyTemplateID != "monster_heavenly_tribulation"
			if isExploration {
				player.HP = 1.0
				player.CurrentRegion, player.X, player.Y = NearestHub(player.X, player.Y)

				turnLogs = append(turnLogs, "Đạo hữu bị đánh trọng thương ngất xỉu. Tỉnh dậy đã thấy ở an toàn.")
			} else {
				player.HP = 1.0
				turnLogs = append(turnLogs, "Đạo hữu bại trận!")
			}
			player.ActiveCombat = nil
		}

		// Update players table
		updateQuery := `
			UPDATE players
			SET hp = $1, current_mana = $2, primeval_stones = $3, x = $4, y = $5, current_region = $6, active_combat = $7
			WHERE id = $8
		`
		_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.PrimevalStones, player.X, player.Y, player.CurrentRegion, player.ActiveCombat, player.ID)
		if err != nil {
			respondCombatTxErr(c, err, "Failed to update player status after failed retreat")
			return
		}

		if err = tx.Commit(); err != nil {
			respondCombatTxErr(c, err, "Transaction commit failed")
			return
		}

		setWorldPosFromPlayer(&player)

		if defeatAudit != nil {
			auditlog.Log(db.DB, *defeatAudit)
		}
		auditlog.Logf(db.DB, &authID, "combat.flee_failed", "enemy=%s", enemyName)

		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"flee_success": false,
			"logs":         turnLogs,
			"enemy":        player.ActiveCombat,
			"defeat":       defeat,
		})
		return
	}

	isExploration := player.ActiveCombat.CombatMode != "virtual" && player.ActiveCombat.EnemyTemplateID != "monster_heavenly_tribulation" && player.ActiveCombat.MineSlotID == 0

	if isExploration {
		maxHP := gamefx.EffectiveMaxHP(&player)
		damage := maxHP * 0.20 // Lose 20% Max HP
		player.HP -= damage

		if player.HP <= 0 {
			player.HP = 1.0
			player.CurrentRegion, player.X, player.Y = NearestHub(player.X, player.Y)

			dropSummary := applyDeathDropRecord(tx, &authID, nil)
			_ = tx.QueryRow("SELECT primeval_stones, immortal_stones FROM players WHERE id = $1", authID).Scan(&player.PrimevalStones, &player.ImmortalStones)

			// Snapshot the loss before clearing; the audit insert runs AFTER
			// commit so the transaction stays short (no second pool connection
			// held while the row is locked).
			defeatAudit = &auditlog.Entry{
				PlayerID:       &authID,
				Action:         "combat.flee_death",
				RequestSummary: fmt.Sprintf("lost=%s", dropSummary),
				IPAddress:      c.ClientIP(),
				Success:        true,
			}

			if dropSummary != "" {
				logMsg = fmt.Sprintf("Đạo hữu cố gắng bỏ chạy khỏi %s nhưng bị thương nặng ngất xỉu. Tỉnh dậy đã thấy ở an toàn nhưng rơi rớt mất: %s.", enemyName, dropSummary)
			} else {
				logMsg = fmt.Sprintf("Đạo hữu cố gắng bỏ chạy khỏi %s nhưng bị thương nặng ngất xỉu. Tỉnh dậy đã thấy ở an toàn.", enemyName)
			}
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
		SET active_combat = $1, human_qi = $2, hp = $3, current_region = $4, x = $5, y = $6
		WHERE id = $7
	`
	_, err = tx.Exec(updateQuery, player.ActiveCombat, player.HumanQi, player.HP, player.CurrentRegion, player.X, player.Y, player.ID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to update player status after fleeing")
		return
	}

	if err = tx.Commit(); err != nil {
		respondCombatTxErr(c, err, "Transaction commit failed")
		return
	}

	setWorldPosFromPlayer(&player)

	// Best-effort audit logs AFTER commit — never inside the transaction.
	if defeatAudit != nil {
		auditlog.Log(db.DB, *defeatAudit)
	}
	auditlog.Logf(db.DB, &authID, "combat.flee", "enemy=%s", logMsg)
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"log": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}
