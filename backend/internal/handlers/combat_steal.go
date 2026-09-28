package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// isOpenWorldPvP checks if a combat state qualifies as open-world PvP:
// EnemyType is PVP, CombatMode is "real" (not virtual arena), MineSlotID is 0 (not mine battle),
// and TargetPlayerID is non-empty.
func isOpenWorldPvP(cs *models.CombatState) bool {
	return cs != nil &&
		cs.EnemyType == "PVP" &&
		cs.CombatMode == "real" &&
		cs.MineSlotID == 0 &&
		cs.TargetPlayerID != ""
}

// permanentStealChance computes the proc chance for Permanent Steal (Vô Tướng Thủ, Rank 9):
// Base 25% + 10% * (thiefRank - targetRank).
// If target is Bản Mệnh Cổ (isVital == true), chance is reduced by 50% (* 0.5).
// No artificial cap, clamped between 0 and 1.0 (100%).
func permanentStealChance(thiefRank, targetRank int, isVital bool) float64 {
	chance := 0.25 + 0.10*float64(thiefRank-targetRank)
	if chance > 1.0 {
		chance = 1.0
	}
	if chance < 0 {
		chance = 0
	}
	if isVital {
		chance *= 0.5
	}
	return chance
}

// tempStealChance computes the proc chance for Temporary Steal (Ám Thâu Cổ, Rank 8):
// Base 40% + 5% * (thiefRank - targetRank).
// If target is Bản Mệnh Cổ (isVital == true), chance is reduced by 50% (* 0.5).
// Clamped between 0 and 1.0.
func tempStealChance(thiefRank, targetRank int, isVital bool) float64 {
	chance := 0.40 + 0.05*float64(thiefRank-targetRank)
	if isVital {
		chance *= 0.5
	}
	if chance > 1.0 {
		chance = 1.0
	}
	if chance < 0 {
		chance = 0
	}
	return chance
}

// hasStealMechanic checks if a Gu spec has the specified mechanic code.
func hasStealMechanic(spec models.GuDictionaryItem, code string) bool {
	if spec.ID == "GU_FORMLESS_THIEF_9" && code == "PERM_STEAL_GU" {
		return true
	}
	if spec.ID == "GU_SHADOW_STEAL_8" && code == "TEMP_STEAL_GU" {
		return true
	}
	_, found := models.GuNativeMechanic(spec, code)
	return found
}

// findStealGus scans a player's inventory for equipped and ACTIVE steal Gus.
func findStealGus(inventory models.GuInventory) (permStealGu *models.GuInstance, permSpec *models.GuDictionaryItem, tempStealGu *models.GuInstance, tempSpec *models.GuDictionaryItem) {
	for i := range inventory {
		gu := &inventory[i]
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		spec, found := models.GetGu(gu.GuID, gu.Rank)
		if !found {
			continue
		}
		if permStealGu == nil && hasStealMechanic(spec, "PERM_STEAL_GU") {
			permStealGu = gu
			permSpec = &spec
		}
		if tempStealGu == nil && hasStealMechanic(spec, "TEMP_STEAL_GU") {
			tempStealGu = gu
			tempSpec = &spec
		}
	}
	return
}

// handlePreCombatSteal executes the pre-combat steal check for open-world PvP.
// It activates once per combat at battle start.
func handlePreCombatSteal(tx *sql.Tx, combat *models.CombatState, attacker *models.Player) {
	if !isOpenWorldPvP(combat) {
		return
	}

	victimUUID, err := uuid.Parse(combat.TargetPlayerID)
	if err != nil {
		return
	}

	// 1. Check if attacker has a steal Gu
	permGu, permSpec, tempGu, tempSpec := findStealGus(attacker.GuInventory)

	// Attempt Permanent Steal from attacker first
	if permGu != nil {
		if executePermanentSteal(tx, combat, attacker, permGu, permSpec, victimUUID, combat.EnemyName, true) {
			return
		}
	}

	// Attempt Temporary Steal from attacker if perm steal didn't trigger
	if tempGu != nil {
		if executeTempSteal(tx, combat, attacker, tempGu, tempSpec, victimUUID, combat.EnemyName, true) {
			return
		}
	}

	// 2. If attacker has no steal Gu or neither succeeded, check defender Gus
	defPermGu, defPermSpec, defTempGu, defTempSpec := findStealGus(combat.EnemyGuInventory)

	if defPermGu != nil {
		if executePermanentSteal(tx, combat, attacker, defPermGu, defPermSpec, victimUUID, combat.EnemyName, false) {
			return
		}
	}

	if defTempGu != nil {
		if executeTempSteal(tx, combat, attacker, defTempGu, defTempSpec, victimUUID, combat.EnemyName, false) {
			return
		}
	}
}

// executePermanentSteal attempts to steal 1 Gu permanently.
// isAttackerThief is true if attacker is stealing from defender, false if defender is stealing from attacker.
func executePermanentSteal(tx *sql.Tx, combat *models.CombatState, attacker *models.Player, thiefGu *models.GuInstance, thiefSpec *models.GuDictionaryItem, defenderUUID uuid.UUID, defenderName string, isAttackerThief bool) bool {
	var pool models.GuInventory
	if isAttackerThief {
		pool = combat.EnemyGuInventory
	} else {
		pool = attacker.GuInventory
	}

	// Filter eligible Gus: equipped, ACTIVE, and rank <= thiefGu.Rank
	var eligible []models.GuInstance
	for _, g := range pool {
		if g.Equipped && g.Status == "ACTIVE" && g.Rank <= thiefGu.Rank {
			eligible = append(eligible, g)
		}
	}
	if len(eligible) == 0 {
		return false
	}

	// Select random target Gu
	targetGu := eligible[rand.Intn(len(eligible))]
	chance := permanentStealChance(thiefGu.Rank, targetGu.Rank, targetGu.IsVital)

	if rand.Float64() >= chance {
		logger.S.Debugf("Permanent steal missed: roll >= chance %.2f for target %s", chance, targetGu.GuID)
		return false
	}

	var thiefID, victimID string
	var thiefUUID, victimUUID uuid.UUID
	var thiefName, victimName string

	if isAttackerThief {
		thiefID = attacker.ID.String()
		thiefUUID = attacker.ID
		thiefName = attacker.Username
		victimID = defenderUUID.String()
		victimUUID = defenderUUID
		victimName = defenderName
	} else {
		thiefID = defenderUUID.String()
		thiefUUID = defenderUUID
		thiefName = defenderName
		victimID = attacker.ID.String()
		victimUUID = attacker.ID
		victimName = attacker.Username
	}

	// Transfer ownership in DB immediately
	if err := db.TransferGuOwnership(tx, targetGu.InstanceID, victimID, thiefID); err != nil {
		logger.S.Errorf("Failed to transfer Gu %s ownership from %s to %s: %v", targetGu.InstanceID, victimID, thiefID, err)
		return false
	}

	targetGuName := targetGu.GuID
	if spec, ok := models.GetGu(targetGu.GuID, targetGu.Rank); ok {
		targetGuName = spec.Name
	}

	// Update in-memory inventories
	if isAttackerThief {
		// Remove from combat.EnemyGuInventory
		var updatedEnemyGus models.GuInventory
		for _, g := range combat.EnemyGuInventory {
			if g.InstanceID != targetGu.InstanceID {
				updatedEnemyGus = append(updatedEnemyGus, g)
			}
		}
		combat.EnemyGuInventory = updatedEnemyGus

		// Add to attacker's inventory (in storage/unequipped)
		stolenCopy := targetGu
		stolenCopy.Equipped = false
		stolenCopy.IsVital = false
		attacker.GuInventory = append(attacker.GuInventory, stolenCopy)
	} else {
		// Defender stole from attacker: remove from attacker.GuInventory
		var updatedAttackerGus models.GuInventory
		for _, g := range attacker.GuInventory {
			if g.InstanceID != targetGu.InstanceID {
				updatedAttackerGus = append(updatedAttackerGus, g)
			}
		}
		attacker.GuInventory = updatedAttackerGus

		// Add to combat.EnemyGuInventory as unequipped
		stolenCopy := targetGu
		stolenCopy.Equipped = false
		stolenCopy.IsVital = false
		combat.EnemyGuInventory = append(combat.EnemyGuInventory, stolenCopy)
	}

	combat.PermStolenGuInstanceID = targetGu.InstanceID
	combat.PermStolenGuID = targetGu.GuID
	combat.PermStolenGuRank = targetGu.Rank
	combat.PermStolenGuName = targetGuName
	combat.StealLogMessage = fmt.Sprintf("Bàn tay vô hình [Vô Tướng Thủ] của %s đã xuyên không đoạt lấy [%s] của %s!", thiefName, targetGuName, victimName)

	// Send notifications
	notify.Send(tx, victimUUID, "GU_PERMANENT_STOLEN", "Cổ Trùng Bị Cướp!",
		fmt.Sprintf("Cổ sư %s đã dùng [Vô Tướng Thủ] cướp đi vĩnh viễn Cổ trùng [%s] của bạn ngay đầu trận chiến!", thiefName, targetGuName))
	notify.Send(tx, thiefUUID, "GU_PERMANENT_STOLEN_SUCCESS", "Đoạt Cổ Thành Công!",
		fmt.Sprintf("Vô Tướng Thủ đã thành công đoạt lấy Cổ trùng [%s] (Chuyển %d) từ %s!", targetGuName, targetGu.Rank, victimName))

	logger.S.Infof("Permanent steal succeeded: %s stole %s (Rank %d) from %s", thiefName, targetGuName, targetGu.Rank, victimName)
	return true
}

// executeTempSteal attempts to temporarily steal 1 Gu for the duration of the combat.
func executeTempSteal(tx *sql.Tx, combat *models.CombatState, attacker *models.Player, thiefGu *models.GuInstance, thiefSpec *models.GuDictionaryItem, defenderUUID uuid.UUID, defenderName string, isAttackerThief bool) bool {
	var pool models.GuInventory
	if isAttackerThief {
		pool = combat.EnemyGuInventory
	} else {
		pool = attacker.GuInventory
	}

	var eligible []models.GuInstance
	for _, g := range pool {
		if g.Equipped && g.Status == "ACTIVE" && g.Rank <= thiefGu.Rank {
			eligible = append(eligible, g)
		}
	}
	if len(eligible) == 0 {
		return false
	}

	targetGu := eligible[rand.Intn(len(eligible))]
	chance := tempStealChance(thiefGu.Rank, targetGu.Rank, targetGu.IsVital)

	if rand.Float64() >= chance {
		logger.S.Debugf("Temp steal missed: roll >= chance %.2f for target %s", chance, targetGu.GuID)
		return false
	}

	var thiefUUID, victimUUID uuid.UUID
	var thiefName, victimName string

	if isAttackerThief {
		thiefUUID = attacker.ID
		thiefName = attacker.Username
		victimUUID = defenderUUID
		victimName = defenderName
	} else {
		thiefUUID = defenderUUID
		thiefName = defenderName
		victimUUID = attacker.ID
		victimName = attacker.Username
	}

	targetGuName := targetGu.GuID
	if spec, ok := models.GetGu(targetGu.GuID, targetGu.Rank); ok {
		targetGuName = spec.Name
	}

	combat.TempStolenGuInstanceID = targetGu.InstanceID
	combat.TempStolenGuID = targetGu.GuID
	combat.TempStolenGuRank = targetGu.Rank
	combat.TempStolenGuName = targetGuName
	combat.StealLogMessage = fmt.Sprintf("[%s] của %s đã tạm thời đoạt lấy Cổ trùng [%s] của %s trong trận chiến này!", thiefSpec.Name, thiefName, targetGuName, victimName)

	notify.Send(tx, victimUUID, "GU_TEMP_STOLEN", "Cổ Trùng Bị Phong Tỏa!",
		fmt.Sprintf("Cổ sư %s đã dùng [%s] tạm thời đoạt lấy Cổ trùng [%s] của bạn trong trận này!", thiefName, thiefSpec.Name, targetGuName))
	notify.Send(tx, thiefUUID, "GU_TEMP_STOLEN_SUCCESS", "Phong Tỏa Cổ Trùng Địch!",
		fmt.Sprintf("[%s] đã tạm thời đoạt lấy [%s] (Chuyển %d) của %s và có thể sử dụng trong trận này!", thiefSpec.Name, targetGuName, targetGu.Rank, victimName))

	logger.S.Infof("Temp steal succeeded: %s temporarily stole %s (Rank %d) from %s", thiefName, targetGuName, targetGu.Rank, victimName)
	return true
}
