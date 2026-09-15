package handlers

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// applyAutoCombatDefeat handles the losing path for auto-combat: virtual arena
// consolation, mine defender notification, death drop + hub teleport, party
// cleanup, and the player update query. Shared by AutoCombatStart and
// AutoResolveCombat so defeat resolution is identical (including virtual mode).
func applyAutoCombatDefeat(tx *sql.Tx, player *models.Player, simState *simulator.CombatState, isVirtual, isTribulation bool, finalMana float64) (updateQuery string, args []interface{}, defLogs []simulator.CombatLog) {
	if isVirtual {
		updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
		args = append(args, player.ID)
		if err := InventorySvc.AddItem(tx, player.ID.String(), "arena_coin", "materials", 10); err != nil {
			logger.S.Errorf("Failed to award arena_coin to %s: %v", player.ID, err)
		} else {
			defLogs = append(defLogs, simulator.CombatLog{
				Turn:   simState.Turn,
				Action: "ARENA_REWARD",
				Detail: "Thua cuộc tại Võ Đài. Nhận an ủi 10 Xu Đấu Trường.",
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
	if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 && player.ActiveCombat.TargetPlayerID != "" {
		if defenderUUID, errParse := uuid.Parse(player.ActiveCombat.TargetPlayerID); errParse == nil && defenderUUID != player.ID {
			var rarity string
			_ = tx.QueryRow("SELECT rarity FROM mine_slots WHERE id = $1", player.ActiveCombat.MineSlotID).Scan(&rarity)
			notify.Send(tx, defenderUUID, "mine_defended", "Phòng thủ Mỏ thành công!",
				fmt.Sprintf("Đạo hữu [%s] đã tấn công Mỏ #%d (%s) của bạn nhưng đã bị lực lượng phòng thủ đánh bại!", player.Username, player.ActiveCombat.MineSlotID, rarity))
		}
	}
	if isTribulation {
		defeatMsg = "Đạo hữu kiệt sức trước quái tai kiếp! Tiên Khiếu phải gánh chịu phần sát thương còn lại."
	} else if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 {
		defeatMsg = "Thất bại trong trận tranh đoạt Mỏ! May mắn không tổn hại đến bản thể và tài sản."
	} else if player.ActiveCombat != nil && player.ActiveCombat.TargetSectID != "" {
		defeatMsg = "Công kích Tông Môn thất bại! Đạo hữu trọng thương lui về thành trấn an dưỡng."
		sectDefeatLogs := ApplySectWarDefeat(tx, player, player.ActiveCombat)
		for _, sLog := range sectDefeatLogs {
			defLogs = append(defLogs, simulator.CombatLog{
				Turn:   simState.Turn,
				Action: "SECT_WAR",
				Detail: sLog,
			})
		}
		player.CurrentRegion, player.X, player.Y = NearestHub(player.X, player.Y)
	} else {
		deathRegion, deathX, deathY := player.CurrentRegion, player.X, player.Y
		var winnerID *uuid.UUID
		if player.ActiveCombat != nil && player.ActiveCombat.EnemyType == "PVP" && player.ActiveCombat.TargetPlayerID != "" && player.ActiveCombat.TargetPlayerID != "dummy" {
			if tUUID, errParse := uuid.Parse(player.ActiveCombat.TargetPlayerID); errParse == nil {
				winnerID = &tUUID
			}
		}
		dropSummary := applyDeathDrop(tx, player, winnerID, false, DeathPos{Region: deathRegion, X: deathX, Y: deathY})
		player.CurrentRegion, player.X, player.Y = NearestHub(deathX, deathY)
		if dropSummary != "" {
			defeatMsg = fmt.Sprintf("Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết nhưng rơi rớt mất: %s.", dropSummary)
		}
	}

	for _, ally := range player.ActiveCombat.Allies {
		if ally.Kind != "Ally" && ally.Kind != "Player" {
			continue
		}
		allyUUID, errParse := uuid.Parse(ally.ID)
		if errParse == nil && allyUUID != player.ID {
			_, _ = tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", allyUUID)
		}
	}

	defLogs = append(defLogs, simulator.CombatLog{
		Turn:   simState.Turn,
		Action: "BATTLE_END",
		Detail: defeatMsg,
	})

	updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, current_region = $3, x = $4, y = $5, aptitude_percent = $6, lifespan = $7, primeval_stones = $8, immortal_stones = $9, path_masteries = $10, is_dead = $11, version = version + 1 WHERE id = $12`
	args = append(args, player.HP, finalMana, player.CurrentRegion, player.X, player.Y, player.AptitudePercent, player.Lifespan, player.PrimevalStones, player.ImmortalStones, player.PathMasteries, player.IsDead, player.ID)
	return
}
