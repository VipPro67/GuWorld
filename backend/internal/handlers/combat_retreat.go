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
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": errLeader.Error()})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)
	inventoryBefore := len(before)

	logMsg := "Đạo hữu trở về an toàn từ hành trình."

	_ = player.ActiveCombat != nil
	var worldMonsterCombat *models.CombatState

	if player.ActiveCombat != nil {
		encounter := *player.ActiveCombat
		worldMonsterCombat = &encounter
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
	completeWorldMonsterCombat(c.Request.Context(), worldMonsterCombat, "Phe Địch", player.ID.String())

	ClearPendingPlayerQueues(player.ID.String())
	_ = cache.Delete("player_snapshot:" + player.ID.String())
	PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)

	setWorldPosFromPlayer(&player)

	var dbAfter int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1`, player.ID).Scan(&dbAfter)
	logger.S.Infof("Retreat player=%s invBefore=%d invAfter=%d added=%d dbBefore=%d dbAfterTx=%d dbAfter=%d",
		player.ID, inventoryBefore, inventoryAfter, inventoryAfter-inventoryBefore, dbBefore, dbAfterTx, dbAfter)
	respondActionOnly(c, http.StatusOK, player, gin.H{
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

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": errLeader.Error()})
		return
	}

	if player.ActiveCombat == nil {
		ClearPendingPlayerQueues(player.ID.String())
		_ = cache.Delete("player_snapshot:" + player.ID.String())
		PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
		respondActionOnly(c, http.StatusOK, player, gin.H{
			"log": "Không có trận chiến nào đang diễn ra, đạo hữu an toàn rút lui.",
		})
		return
	}
	worldMonsterCombat := *player.ActiveCombat

	enemyName := player.ActiveCombat.EnemyName
	var logMsg string

	// Mine and PvP combat always allow a free escape — fleeing is not gated by
	// a random roll and carries no penalty for those encounter types.
	freeFlee := fleeFreeRetreat(player.ActiveCombat)

	fleeChance := 0.70
	if player.ActiveCombat.EnemyType == "BOSS" || player.ActiveCombat.EnemyType == "TRIBULATION_BOSS" || player.ActiveCombat.EnemyType == "TRIBULATION_MONSTER" {
		fleeChance = 0.40
	}
	roll := rand.Float64()
	success := freeFlee || roll < fleeChance

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
						_, _ = tx.Exec(`DELETE FROM non_player_actors WHERE id = $1 AND owner_player_id = $2 AND actor_kind = 'COMPANION_BEAST'`, ally.ID, player.ID)
					} else {
						_, _ = tx.Exec(`UPDATE non_player_actors SET hp = $1, updated_at = NOW(), version = version + 1 WHERE id = $2 AND owner_player_id = $3 AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'`,
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
		enemyCombatState := player.ActiveCombat

		// If the counterattack killed every remaining enemy (e.g. the enemy
		// self-destructed through its parasitic Gu cost), the player wins on the
		// spot instead of continuing the fight.
		victory := false
		if !defeat {
			victory = true
			for _, enemy := range player.ActiveCombat.Enemies {
				if enemy.HP > 0 {
					victory = false
					break
				}
			}
		}

		if victory {
			victoryLogs := fleeFailVictory(tx, &player)
			turnLogs = append(turnLogs, victoryLogs...)

			updateQuery := `
				UPDATE players
				SET hp = $1, current_mana = $2, primeval_stones = $3, x = $4, y = $5, current_region = $6,
				    active_combat = $7, human_qi = $8, cleared_cells = $9
				WHERE id = $10
			`
			_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.PrimevalStones, player.X, player.Y, player.CurrentRegion,
				player.ActiveCombat, player.HumanQi, player.ClearedCells, player.ID)
			if err != nil {
				respondCombatTxErr(c, err, "Failed to update player status after fleeing")
				return
			}

			if err = tx.Commit(); err != nil {
				respondCombatTxErr(c, err, "Transaction commit failed")
				return
			}

			ClearPendingPlayerQueues(player.ID.String())
			_ = cache.Delete("player_snapshot:" + player.ID.String())
			PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)

			setWorldPosFromPlayer(&player)

			player.GuInventory, _ = db.LoadPlayerGus(db.DB, player.ID.String(), false)
			respondActionWithInventory(c, http.StatusOK, player, gin.H{
				"flee_success": false,
				"victory":      true,
				"defeat":       false,
				"logs":         turnLogs,
				"enemy":        enemyCombatState,
			})
			return
		}

		if defeat {
			turnLogs = append(turnLogs, "=== THẤT BẠI THẢM HẠI ===")
			isTribulation := player.ActiveCombat.TribulationEventID != "" || player.ActiveCombat.EnemyTemplateID == "monster_heavenly_tribulation"
			isExploration := player.ActiveCombat.CombatMode != "virtual" && !isTribulation
			isMine := player.ActiveCombat.MineSlotID > 0
			if isExploration {
				deathRegion, deathX, deathY := player.CurrentRegion, player.X, player.Y
				player.HP = 1.0
				player.CurrentRegion, player.X, player.Y = ResolveRespawnPosition(tx, authID, deathRegion, deathX, deathY)

				turnLogs = append(turnLogs, "Đạo hữu bị đánh trọng thương ngất xỉu. Tỉnh dậy đã được đưa về nơi an dưỡng.")

				// Death penalty applies to real PvE exploration and normal PvP only.
				// Mine and arena (virtual) defeats never drop items/stones.
				if !isMine {
					var winnerID *uuid.UUID
					if player.ActiveCombat.TargetPlayerID != "" {
						if winner, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID); parseErr == nil {
							winnerID = &winner
						}
					}
					killerName := ""
					if winnerID == nil {
						killerName = enemyName
					}
					dropSummary := applyDeathDropRecord(tx, &authID, winnerID, false, DeathPos{Region: deathRegion, X: deathX, Y: deathY, KillerName: killerName})
					_ = tx.QueryRow("SELECT primeval_stones, immortal_stones FROM players WHERE id = $1", authID).Scan(&player.PrimevalStones, &player.ImmortalStones)
					if dropSummary != "" {
						if winnerID != nil {
							turnLogs = append(turnLogs, fmt.Sprintf("Đạo hữu bại trận và bị đối thủ cướp đoạt: %s.", dropSummary))
						} else {
							turnLogs = append(turnLogs, fmt.Sprintf("Đạo hữu bại trận và rơi rớt mất: %s.", dropSummary))
						}
					}
				}
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

		if defeat {
			ClearPendingPlayerQueues(player.ID.String())
			_ = cache.Delete("player_snapshot:" + player.ID.String())
			PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
			completeWorldMonsterCombat(c.Request.Context(), &worldMonsterCombat, "Phe Địch", player.ID.String())
		} else {
			PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
		}

		setWorldPosFromPlayer(&player)

		if defeat {
			player.GuInventory, _ = db.LoadPlayerGus(db.DB, player.ID.String(), false)
			respondActionWithInventory(c, http.StatusOK, player, gin.H{
				"flee_success": false,
				"victory":      false,
				"logs":         turnLogs,
				"enemy":        enemyCombatState,
				"defeat":       defeat,
			})
			return
		}

		respondActionOnly(c, http.StatusOK, player, gin.H{
			"flee_success": false,
			"victory":      false,
			"logs":         turnLogs,
			"enemy":        enemyCombatState,
			"defeat":       defeat,
		})
		return
	}

	isTribulation := player.ActiveCombat.TribulationEventID != "" || player.ActiveCombat.EnemyTemplateID == "monster_heavenly_tribulation" || player.ActiveCombat.EnemyType == "TRIBULATION_BOSS" || player.ActiveCombat.EnemyType == "TRIBULATION_MONSTER"
	isExploration := player.ActiveCombat.CombatMode != "virtual" && !isTribulation && player.ActiveCombat.MineSlotID == 0 && player.ActiveCombat.EnemyType != "PVP"

	droppedOnFlee := false
	if isExploration {
		maxHP := gamefx.EffectiveMaxHP(&player)
		damage := maxHP * 0.20 // Lose 20% Max HP
		player.HP -= damage

		if player.HP <= 0 {
			deathRegion, deathX, deathY := player.CurrentRegion, player.X, player.Y
			player.HP = 1.0
			dropSummary := applyDeathDropRecord(tx, &authID, nil, false, DeathPos{Region: deathRegion, X: deathX, Y: deathY, KillerName: enemyName})
			player.CurrentRegion, player.X, player.Y = ResolveRespawnPosition(tx, authID, deathRegion, deathX, deathY)
			droppedOnFlee = true
			_ = tx.QueryRow("SELECT primeval_stones, immortal_stones FROM players WHERE id = $1", authID).Scan(&player.PrimevalStones, &player.ImmortalStones)

			if dropSummary != "" {
				logMsg = fmt.Sprintf("Đạo hữu cố gắng bỏ chạy khỏi %s nhưng bị thương nặng ngất xỉu. Tỉnh dậy đã được đưa về nơi an dưỡng nhưng rơi rớt mất: %s.", enemyName, dropSummary)
			} else {
				logMsg = fmt.Sprintf("Đạo hữu cố gắng bỏ chạy khỏi %s nhưng bị thương nặng ngất xỉu. Tỉnh dậy đã được đưa về nơi an dưỡng.", enemyName)
			}
		} else {
			logMsg = fmt.Sprintf("Đạo hữu đã bỏ chạy khỏi %s. Chịu thương tổn %.0f HP.", enemyName, damage)
		}
	} else if isTribulation {
		logMsg = fmt.Sprintf("Đạo hữu đã rút lui an toàn khỏi %s.", enemyName)
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
	completeWorldMonsterCombat(c.Request.Context(), &worldMonsterCombat, "Phe Địch", player.ID.String())

	ClearPendingPlayerQueues(player.ID.String())
	_ = cache.Delete("player_snapshot:" + player.ID.String())
	PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)

	setWorldPosFromPlayer(&player)

	if droppedOnFlee {
		player.GuInventory, _ = db.LoadPlayerGus(db.DB, player.ID.String(), false)
		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"log": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		})
		return
	}

	respondActionOnly(c, http.StatusOK, player, gin.H{
		"log": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

// fleeFreeRetreat reports whether an active combat allows a guaranteed,
// penalty-free escape (PvP, mine encounters, and tribulation encounters).
func fleeFreeRetreat(cs *models.CombatState) bool {
	if cs == nil {
		return false
	}
	if cs.TribulationEventID != "" || cs.EnemyTemplateID == "monster_heavenly_tribulation" ||
		cs.EnemyType == "TRIBULATION_BOSS" || cs.EnemyType == "TRIBULATION_MONSTER" {
		return true
	}
	return cs.EnemyType == "PVP" || cs.MineSlotID > 0
}

// fleeFailVictory handles the edge case where a failed flee's enemy counterattack
// ends up killing every remaining enemy (e.g. the enemy self-destructs through
// its parasitic Gu cost). The player wins on the spot: loot is granted, the
// enemy is cleared from the map (boss cell / explored cell), and the player
// stays at their current position. Returns the victory logs.
func fleeFailVictory(tx *sql.Tx, player *models.Player) []string {
	var logs []string
	cs := player.ActiveCombat
	if cs == nil {
		return logs
	}

	isPvE := cs.CombatMode != "virtual" && cs.EnemyTemplateID != "monster_heavenly_tribulation" &&
		cs.MineSlotID == 0 && cs.EnemyType != "PVP"

	logs = append(logs, "=== CHIẾN THẮNG BẤT NGỜ ===")
	logs = append(logs, "Đạo hữu cố gắng bỏ chạy nhưng địch nhân tự chuốc lấy diệt vong!")

	// The Flee handler uses the light (equipped-only) inventory loader, so the
	// before-snapshot is a subset of the post-loot inventory — SyncPlayerGus
	// therefore only inserts the newly looted Gus without deleting anything.
	before := append(models.GuInventory(nil), player.GuInventory...)

	if isPvE && cs.LootTableID != "" {
		if lt, err := db.LoadLootTable(tx, cs.LootTableID); err == nil {
			loot := RollLoot(lt, cs.EnemyRank)
			if cs.EnemyGuInventory != nil && len(cs.EnemyGuInventory) > 0 {
				for _, eg := range cs.EnemyGuInventory {
					guSpec, _ := models.GetGuAny(eg.GuID)
					dropChance := GetGuDropChancePercent(guSpec.Rank, isBossEnemy(cs))
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
			logs = append(logs, ApplyLoot(tx, player, loot)...)
		}
	}

	markBossCleared(tx, player)
	if player.ClearedCells == nil {
		player.ClearedCells = make(models.ClearedCellsMap)
	}
	coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
	player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
	if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearedCellCooldown); err != nil {
		logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
	}
	if player.ActiveCombat != nil {
		_ = cache.BroadcastMonsterKilled(player.X, player.Y, player.ActiveCombat.EnemyName, player.CurrentRegion)
	}
	_ = cache.BroadcastCellCleared(player.CurrentRegion, player.X, player.Y, "Monster", player.ActiveCombat.EnemyTemplateID)

	qiGained := AddHumanQi(player, 20.0)
	logs = append(logs, fmt.Sprintf("Chiến thắng đem lại sự tin tưởng của bá tánh. Nhân Khí +%.0f.", qiGained))

	player.ActiveCombat = nil

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		return logs
	}
	return logs
}
