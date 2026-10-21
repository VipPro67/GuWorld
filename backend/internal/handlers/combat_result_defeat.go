package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// applyAutoCombatDefeat handles the losing path for auto-combat: virtual arena
// consolation, death drop + hub teleport, party cleanup, and the player update
// query. Shared by AutoCombatStart and
// ResolveCombat so defeat resolution is identical (including virtual mode).
func applyAutoCombatDefeat(tx *sql.Tx, player *models.Player, simState *simulator.CombatState, isVirtual, isTribulation bool, finalMana float64) (updateQuery string, args []interface{}, defLogs []simulator.CombatLog) {
	if isVirtual {
		updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
		args = append(args, player.ID)
		if err := InventorySvc.AddItem(tx, player.ID.String(), "arena_coin", "materials", 1); err != nil {
			logger.S.Errorf("Failed to award arena_coin to %s: %v", player.ID, err)
		} else {
			defLogs = append(defLogs, simulator.CombatLog{
				Turn:   simState.Turn,
				Action: "ARENA_REWARD",
				Detail: "Thua cuộc tại Võ Đài. Nhận an ủi 1 Xu Đấu Trường.",
			})
		}
		defLogs = append(defLogs, simulator.CombatLog{
			Turn:   simState.Turn,
			Action: "BATTLE_END",
			Detail: "Thua cuộc trong đấu trường ảnh! Bản thể không bị ảnh hưởng.",
		})
		return
	}

	player.HP = 1.0
	defeatMsg := "Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết."
	deathRegion, deathX, deathY := player.CurrentRegion, player.X, player.Y
	isPVP := player.ActiveCombat != nil && player.ActiveCombat.EnemyType == "PVP"
	didRespawn := false
	var winnerID *uuid.UUID

	if isTribulation {
		defeatMsg = "Đạo hữu kiệt sức trước quái tai kiếp! Tiên Khiếu phải gánh chịu phần sát thương còn lại."
	} else if player.ActiveCombat != nil && player.ActiveCombat.TargetSectID != "" {
		defeatMsg = "Công kích Tông Môn thất bại! Đạo hữu trọng thương lui về an dưỡng."
		sectDefeatLogs := ApplySectWarDefeat(tx, player, player.ActiveCombat)
		for _, sLog := range sectDefeatLogs {
			defLogs = append(defLogs, simulator.CombatLog{
				Turn:   simState.Turn,
				Action: "SECT_WAR",
				Detail: sLog,
			})
		}
		dropSummary := applyDeathDrop(tx, player, nil, false, DeathPos{Region: deathRegion, X: deathX, Y: deathY})
		player.CurrentRegion, player.X, player.Y = ResolveRespawnPosition(tx, player.ID, deathRegion, deathX, deathY)
		didRespawn = true
		if dropSummary != "" {
			defeatMsg = fmt.Sprintf("Công kích Tông Môn thất bại! Đạo hữu trọng thương và rơi rớt mất: %s.", dropSummary)
		}
	} else {
		if isPVP && player.ActiveCombat.TargetPlayerID != "" && player.ActiveCombat.TargetPlayerID != "dummy" {
			if tUUID, errParse := uuid.Parse(player.ActiveCombat.TargetPlayerID); errParse == nil {
				winnerID = &tUUID
			}
		}
		killerName := ""
		if winnerID == nil && player.ActiveCombat != nil {
			killerName = player.ActiveCombat.EnemyName
		}
		dropSummary := applyDeathDrop(tx, player, winnerID, false, DeathPos{Region: deathRegion, X: deathX, Y: deathY, KillerName: killerName})
		player.CurrentRegion, player.X, player.Y = ResolveRespawnPosition(tx, player.ID, deathRegion, deathX, deathY)
		didRespawn = true
		setWorldPosFromPlayer(player)
		if dropSummary != "" {
			defeatMsg = fmt.Sprintf("Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết nhưng rơi rớt mất: %s.", dropSummary)
		}
	}
	if didRespawn {
		defeatMsg += " " + RespawnLocationMessage(deathRegion, player.CurrentRegion, player.X, player.Y)
	}

	for _, ally := range player.ActiveCombat.Allies {
		if ally.Kind != "Ally" && ally.Kind != "Player" {
			continue
		}
		allyUUID, errParse := uuid.Parse(ally.ID)
		if errParse == nil && allyUUID != player.ID {
			if isPVP || (player.ActiveCombat != nil && player.ActiveCombat.TargetSectID != "") {
				// All party members suffer death drop and respawn in PvP / Sect War defeat
				_ = applyDeathDropRecord(tx, &allyUUID, winnerID, false, DeathPos{Region: deathRegion, X: deathX, Y: deathY})
				aReg, aX, aY := ResolveRespawnPosition(tx, allyUUID, deathRegion, deathX, deathY)
				worldPosIndex.Set(WorldPosEntry{
					ID:        allyUUID.String(),
					Username:  ally.Name,
					AvatarURL: ally.AvatarURL,
					X:         aX,
					Y:         aY,
					Region:    aReg,
					TS:        time.Now().UnixMilli(),
				})
				_, _ = tx.Exec("UPDATE players SET hp = 1.0, current_region = $1, x = $2, y = $3, active_combat = NULL WHERE id = $4", aReg, aX, aY, allyUUID)
			} else {
				// Persist ally HP/mana from sim (mirror victory path)
				var allyHP, allyMana float64
				for _, sa := range simState.Allies {
					if sa.ID == ally.ID {
						allyHP = sa.HP
						allyMana = sa.Mana
						break
					}
				}
				if allyHP <= 0 {
					allyHP = 1
				}
				_, _ = tx.Exec("UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL WHERE id = $3", allyHP, allyMana, allyUUID)
			}
		}
	}

	defLogs = append(defLogs, simulator.CombatLog{
		Turn:   simState.Turn,
		Action: "BATTLE_END",
		Detail: defeatMsg,
	})

	updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, current_region = $3, x = $4, y = $5, aptitude_percent = $6, lifespan = $7, primeval_stones = $8, immortal_stones = $9, path_masteries = $10, version = version + 1 WHERE id = $11`
	args = append(args, player.HP, finalMana, player.CurrentRegion, player.X, player.Y, player.AptitudePercent, player.Lifespan, player.PrimevalStones, player.ImmortalStones, player.PathMasteries, player.ID)
	return
}
