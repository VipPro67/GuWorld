package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// AutoCombatStartRequest payload for starting automated combat
type AutoCombatStartRequest struct {
	Loadout simulator.CombatLoadout `json:"loadout" binding:"required"`
}

// CombatantInfo returned to frontend for visual replay
type CombatantInfo struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Kind          string             `json:"kind"`
	MaxHP         float64            `json:"max_hp"`
	HP            float64            `json:"hp"`
	MaxMana       float64            `json:"max_mana"`
	Mana          float64            `json:"mana"`
	GridX         int                `json:"grid_x"`
	GridY         int                `json:"grid_y"`
	Shield        float64            `json:"shield"`
	Damage        float64            `json:"damage,omitempty"`
	Defense       float64            `json:"defense,omitempty"`
	Speed         float64            `json:"speed,omitempty"`
	KindDetail    string             `json:"kind_detail,omitempty"`
	Rank          int                `json:"rank,omitempty"`
	SubRank       int                `json:"sub_rank,omitempty"`
	DaoMarks      models.DaoMarksMap `json:"dao_marks,omitempty"`
	PathMasteries map[string]int     `json:"path_masteries,omitempty"`
	GuInventory   models.GuInventory `json:"gu_inventory,omitempty"`
}

// AutoCombatStart initializes and fully simulates a combat, returning the deterministic log.
func AutoCombatStart(c *gin.Context) {
	var req AutoCombatStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if player.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể bắt đầu giao tranh!"})
		return
	}

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
		return
	}

	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong trạng thái chiến đấu."})
		return
	}

	isVirtual := player.ActiveCombat.CombatMode == "virtual"
	isTribulation := player.ActiveCombat.TribulationEventID != ""

	simState, err := buildSimCombatState(tx, &player, &req.Loadout, isVirtual)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize initiator stats"})
		return
	}
	simPlayer := simState.Allies[0]

	// ── Build combatant info for frontend replay ──
	var combatantInfos []CombatantInfo
	for _, ally := range simState.Allies {
		cInst := CombatantInfo{
			ID:            ally.ID,
			Name:          ally.Name,
			Kind:          "ally",
			MaxHP:         ally.MaxHP,
			HP:            ally.HP,
			MaxMana:       ally.MaxMana,
			Mana:          ally.Mana,
			GridX:         ally.GridX,
			GridY:         ally.GridY,
			Shield:        ally.VirtualShield,
			Damage:        ally.BaseDmg,
			Defense:       ally.BaseDef,
			Speed:         ally.Speed,
			KindDetail:    ally.Kind,
			Rank:          ally.Rank,
			SubRank:       ally.SubRank,
			DaoMarks:      models.DaoMarksMap(ally.DaoMarks),
			PathMasteries: ally.PathMasteries,
		}
		if ally.ID == player.ID.String() {
			cInst.GuInventory = filterCombatGuInventory(player.GuInventory, req.Loadout.GuIds, player.CombatSettings)
		}
		combatantInfos = append(combatantInfos, cInst)
	}
	for _, e := range player.ActiveCombat.Enemies {
		combatantInfos = append(combatantInfos, CombatantInfo{
			ID:            e.ID,
			Name:          e.Name,
			Kind:          "enemy",
			MaxHP:         e.MaxHP,
			HP:            e.HP,
			MaxMana:       e.MaxMana,
			Mana:          e.CurrentMana,
			GridX:         e.GridX,
			GridY:         e.GridY,
			Damage:        e.BaseAttack,
			Defense:       e.BaseDefense,
			Speed:         e.Speed,
			KindDetail:    e.Kind,
			Rank:          e.Rank,
			SubRank:       e.SubRank,
			DaoMarks:      e.DaoMarks,
			PathMasteries: e.PathMasteries,
			GuInventory:   e.GuInventory,
		})
	}

	// 4. Run Simulator
	logs := simulator.Simulate(simState)

	// Apply Love Gu (Chaos Gamble) penalties
	applyLoveGuPenalties(&player, logs)

	// 5. Apply outcome to DB
	_, alliesAlive, finalPlayerHP := combatOutcome(simState, player.ID.String())

	itemsUsed, stonesSpent, _, usageLogs := applySimConsumption(tx, &player, simPlayer, isVirtual, simState.Turn)
	logs = append(logs, usageLogs...)

	// ── Grab final mana from sim ──
	var finalMana float64
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			finalMana = a.Mana
			break
		}
	}

	var updateQuery string
	var args []interface{}

	// ── Combat type label for logs ──
	enemyTypeName := combatTypeLabel(player.ActiveCombat)
	logs = append([]simulator.CombatLog{{
		Turn:   0,
		Action: "COMBAT_TYPE",
		Detail: fmt.Sprintf("=== %s ===", enemyTypeName),
	}}, logs...)

	if alliesAlive {
		if isVirtual {
			if err := InventorySvc.AddItem(tx, player.ID.String(), "arena_coin", "materials", 50); err != nil {
				logger.S.Errorf("Failed to award arena_coin to %s: %v", player.ID, err)
			} else {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "ARENA_REWARD",
					Detail: "Chiến thắng Võ Đài! Nhận thưởng 50 Xu Đấu Trường.",
				})
			}
			if player.ActiveCombat != nil && player.ActiveCombat.TargetPlayerID != "" && player.ActiveCombat.TargetPlayerID != "dummy" {
				targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
				if parseErr == nil {
					var playerRank, targetRank int
					errPR := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", player.ID).Scan(&playerRank)
					errTR := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", targetUUID).Scan(&targetRank)
					if errPR == nil && errTR == nil && playerRank > targetRank {
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = -1 WHERE player_id = $1", player.ID)
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", playerRank, targetUUID)
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", targetRank, player.ID)
						_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", playerRank, targetUUID)
						_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", targetRank, player.ID)
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "ARENA_RANK_UP",
							Detail: fmt.Sprintf("Chúc mừng! Đạo hữu đã vượt cấp khiêu chiến thành công, thăng lên Hạng %d Võ Đài!", targetRank),
						})
					}
				}
			}
			updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
			args = append(args, player.ID)
		} else {
			var partyMembers []uuid.UUID
			for _, ally := range player.ActiveCombat.Allies {
				if ally.Kind == "Pet" {
					continue
				}
				allyUUID, errParse := uuid.Parse(ally.ID)
				if errParse == nil {
					partyMembers = append(partyMembers, allyUUID)
				}
			}

			lootDetailLogs := applyCombatVictoryLoot(tx, &player, player.ActiveCombat, partyMembers, isVirtual)

			markBossCleared(tx, &player)

			for _, lMsg := range lootDetailLogs {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "LOOT",
					Detail: lMsg,
				})
			}

			mineLogs, _ := applyAutoCombatMineOccupation(tx, &player, player.ActiveCombat, simState.Turn)
			logs = append(logs, mineLogs...)

			// Sect War Victory
			if player.ActiveCombat.TargetSectID != "" {
				sectWarLogs := ApplySectWarVictory(tx, &player, player.ActiveCombat)
				for _, swMsg := range sectWarLogs {
					logs = append(logs, simulator.CombatLog{
						Turn:   simState.Turn,
						Action: "SECT_WAR",
						Detail: swMsg,
					})
				}
			}

			if player.ClearedCells == nil {
				player.ClearedCells = make(models.ClearedCellsMap)
			}
			if !isTribulation && player.ActiveCombat.TargetSectID == "" {
				coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
				if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearedCellCooldown); err != nil {
					logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
				}
				if player.ActiveCombat != nil {
					_ = cache.BroadcastMonsterKilled(player.X, player.Y, player.ActiveCombat.EnemyName, player.CurrentRegion)
				}
				_ = cache.BroadcastCellCleared(player.CurrentRegion, player.X, player.Y, "Monster", player.ActiveCombat.EnemyTemplateID)
			}

			updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, primeval_stones = $3, immortal_stones = $4, aptitude_percent = $5, lifespan = $6, path_masteries = $7, cleared_cells = $8, is_dead = $9, version = version + 1 WHERE id = $10`
			args = append(args, finalPlayerHP, finalMana, player.PrimevalStones, player.ImmortalStones, player.AptitudePercent, player.Lifespan, player.PathMasteries, player.ClearedCells, player.IsDead, player.ID)
		}
	} else {
		defUpdateQuery, defArgs, defLogs := applyAutoCombatDefeat(tx, &player, simState, isVirtual, isTribulation, finalMana)
		updateQuery = defUpdateQuery
		args = append(args, defArgs...)
		logs = append(logs, defLogs...)
	}

	logs, err = finalizeAutoCombatOutcome(tx, &player, simState, isVirtual, isTribulation, alliesAlive, updateQuery, args, finalPlayerHP, finalMana, before, logs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat outcome"})
		return
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 3, $1) WHERE id = $2", MaxHumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit combat transaction"})
		return
	}

	setWorldPosFromPlayer(&player)

	winner := "Phe Địch"
	if alliesAlive {
		winner = "Phe Ta"
	}

	// ── Save combat history (non-critical) ──
	enemyName := ""
	enemyType := ""
	var targetPlayerID string
	if player.ActiveCombat != nil {
		enemyName = player.ActiveCombat.EnemyName
		enemyType = player.ActiveCombat.EnemyType
		targetPlayerID = player.ActiveCombat.TargetPlayerID
	}
	go func() {
		summary := buildCombatSummaryFromLogs(logs, finalPlayerHP, itemsUsed, int64(stonesSpent))
		combatantInfosIface := make([]interface{}, len(combatantInfos))
		for i, ci := range combatantInfos {
			combatantInfosIface[i] = ci
		}
		SaveCombatHistory(player.ID, winner, toInterfaceSlice(logs), summary, combatantInfosIface, enemyName, enemyType, player.CurrentRegion)

		// Also save history for PvP target player (defender)
		if enemyType == "PVP" && targetPlayerID != "" && targetPlayerID != "dummy" {
			defenderWinner := "Phe Ta"
			if alliesAlive { // Attacker won, so defender lost!
				defenderWinner = "Phe Địch"
			}
			defenderCombatantInfos := make([]interface{}, len(combatantInfos))
			for i, ci := range combatantInfos {
				defCI := ci
				if defCI.Kind == "ally" {
					defCI.Kind = "enemy"
				} else if defCI.Kind == "enemy" {
					defCI.Kind = "ally"
				}
				defenderCombatantInfos[i] = defCI
			}
			SaveCombatHistory(uuid.MustParse(targetPlayerID), defenderWinner, toInterfaceSlice(logs), summary, defenderCombatantInfos, player.Username, "PVP", player.CurrentRegion)
		}
	}()
	c.JSON(http.StatusOK, gin.H{
		"message":         "Chiến đấu hoàn tất. Nhân Khí +3.",
		"logs":            logs,
		"winner":          winner,
		"combatants":      combatantInfos,
		"aperture_hp":     simState.ApertureHP,
		"aperture_max_hp": simState.ApertureMaxHP,
	})
}

// buildSimEnemyCombatant constructs a simulator.CombatantAuto for an enemy/defender in combat.
// If the enemy is a PVP defender (player) and has CombatSettings, it mirrors the exact
// loadout ordering, action priorities, conditions, killer moves, and category priorities
// used by buildSimCombatant for player combatants.
func buildSimEnemyCombatant(tx *sql.Tx, enemy *models.Combatant, activeCombat *models.CombatState) *simulator.CombatantAuto {
	var enemyShield float64
	for _, guInst := range enemy.GuInventory {
		if !guInst.Equipped || guInst.Status != "ACTIVE" {
			continue
		}
		spec, ok := models.GetGu(guInst.GuID, guInst.Rank)
		if !ok {
			continue
		}
		if spec.Effects.Active != nil && spec.Effects.Active.ShieldFlat > 0 {
			enemyShield += spec.Effects.Active.ShieldFlat
		} else {
			for _, block := range spec.Effects.Blocks {
				if block.Trigger == gameplayeffects.TriggerOnAction {
					for _, op := range block.Operations {
						if op.Op == gameplayeffects.OpShield && (strings.EqualFold(op.SubType, "FLAT") || op.SubType == "") {
							enemyShield += op.Value
						}
					}
				}
			}
		}
	}

	eHP := enemy.HP
	if eHP <= 0 {
		eHP = enemy.MaxHP
	}
	enemyMana := enemy.CurrentMana
	if activeCombat != nil && activeCombat.CombatMode == "virtual" {
		eHP = enemy.MaxHP
		enemyMana = enemy.MaxMana
	}

	enemyKind := "Enemy"
	if enemy.Kind == "Pet" {
		enemyKind = "Pet"
	} else if enemy.Kind == "Boss" {
		enemyKind = "Boss"
	}

	cs := enemy.CombatSettings
	if cs == nil && activeCombat != nil {
		cs = activeCombat.EnemyCombatSettings
	}

	simEnemy := &simulator.CombatantAuto{
		ID:            enemy.ID,
		Name:          enemy.Name,
		Kind:          enemyKind,
		Rank:          enemy.Rank,
		SubRank:       enemy.SubRank,
		GridX:         enemy.GridX,
		GridY:         enemy.GridY,
		MaxHP:         enemy.MaxHP,
		HP:            eHP,
		MaxMana:       enemy.MaxMana,
		Mana:          enemyMana,
		BaseDmg:       enemy.BaseAttack,
		BaseDef:       enemy.BaseDefense,
		Speed:         enemy.Speed,
		VirtualShield: enemyShield,
		DaoMarks: func() simulator.DaoMarks {
			dm := make(simulator.DaoMarks)
			for k, v := range enemy.DaoMarks {
				dm[k] = v
			}
			return dm
		}(),
		PathMasteries: enemy.PathMasteries,
	}

	if simEnemy.Rank < 1 {
		simEnemy.Rank = 1
	}

	if activeCombat != nil {
		simEnemy.PathResistance = activeCombat.EnemyPathResistance
		simEnemy.MentalResistance = activeCombat.EnemyMentalResistance
		simEnemy.Tenacity = activeCombat.EnemyTenacity
		simEnemy.CCResist = activeCombat.EnemyCCResist
		simEnemy.DaoBonus = activeCombat.EnemyDaoBonus
		simEnemy.PathDamageBonus = activeCombat.EnemyPathDamageBonus
		simEnemy.HealingReceived = activeCombat.EnemyHealingReceived
		simEnemy.SoulDefenseBonus = activeCombat.EnemySoulDefenseBonus
		simEnemy.LuckBonus = activeCombat.EnemyLuckBonus
	}

	simEnemy.DaoProfile = gamefx.BuildDaoCombatProfile(
		enemy.DaoMarks,
		enemy.PathMasteries,
		false,
		simEnemy.DaoBonus,
	)

	// Highest-rank equipped ACTIVE instance per GuID, used to resolve loadout specs.
	// Unequipped Gus never count (mirrors buildSimCombatant); hunger is
	// intentionally ignored — monster Gus run at Hunger 0 by design and the
	// engine hunger-gate is Player-only.
	instByGuID := make(map[string]models.GuInstance)
	for _, g := range enemy.GuInventory {
		if !g.Equipped || g.Status != "ACTIVE" {
			continue
		}
		if best, ok := instByGuID[g.GuID]; !ok || g.Rank > best.Rank {
			instByGuID[g.GuID] = g
		}
	}
	loadoutSpec := func(guID string) (models.GuDictionaryItem, bool) {
		inst, ok := instByGuID[guID]
		if !ok {
			return models.GuDictionaryItem{}, false
		}
		if spec, found := models.GetGu(guID, inst.Rank); found {
			return spec, true
		}
		return models.GetGuAny(guID)
	}

	// Active GuIDs honoring the defender's saved loadout order when configured,
	// otherwise every equipped ACTIVE Gu.
	var selectedGuIDs []string
	if cs != nil && len(cs.GuPriorities) > 0 {
		seen := make(map[string]bool, len(cs.GuPriorities))
		for _, t := range cs.GuPriorities {
			if seen[t.GuID] {
				continue
			}
			if _, ok := instByGuID[t.GuID]; !ok {
				continue
			}
			seen[t.GuID] = true
			selectedGuIDs = append(selectedGuIDs, t.GuID)
		}
	}
	if len(selectedGuIDs) == 0 {
		seen := make(map[string]bool, len(enemy.GuInventory))
		for _, g := range enemy.GuInventory {
			if !g.Equipped || g.Status != "ACTIVE" || seen[g.GuID] {
				continue
			}
			seen[g.GuID] = true
			selectedGuIDs = append(selectedGuIDs, g.GuID)
		}
	}

	// Build Gu actions from the selected loadout, dropping duplicate GuIDs.
	var enemyActions []simulator.SimAction
	for _, guID := range selectedGuIDs {
		spec, ok := loadoutSpec(guID)
		if !ok || !hasCombatActiveEffects(&spec) {
			continue
		}
		act, err := buildSimActionFromGu(enemy, &spec, "GU_SKILL")
		if err != nil {
			continue
		}
		act.SourceID = guID
		if inst, ok := instByGuID[guID]; ok {
			act.InstanceID = inst.InstanceID
		}
		enemyActions = append(enemyActions, act)
	}

	// For PVP defender with UUID, load active killer moves if any
	isPVP := activeCombat != nil && (activeCombat.EnemyType == "PVP" || enemy.Kind == "Player")
	if isPVP && activeCombat.TargetPlayerID != "" {
		if targetUUID, errParse := uuid.Parse(activeCombat.TargetPlayerID); errParse == nil && tx != nil {
			var selectedKMIDs []string
			if cs != nil && len(cs.KMPriorities) > 0 {
				for _, kp := range cs.KMPriorities {
					selectedKMIDs = append(selectedKMIDs, kp.KillerMoveID)
				}
			}
			kmMap, slotsMap, errKM := loadKillerMovesBatch(tx, targetUUID, selectedKMIDs)
			if errKM == nil {
				ownedGuInstMap := make(map[string]bool)
				for _, g := range enemy.GuInventory {
					if !g.Equipped || g.Status != "ACTIVE" {
						continue
					}
					ownedGuInstMap[g.InstanceID] = true
				}
				for kmID, km := range kmMap {
					slots := slotsMap[kmID]
					if ownedGuInstMap[km.CorePlayerGuID] {
						allValid := true
						for _, slot := range slots {
							if !ownedGuInstMap[slot.PlayerGuID] {
								allValid = false
								break
							}
						}
						if allValid {
							act := buildSimActionFromKillerMove(km, slots, enemy.DaoMarks)
							act.SourceID = kmID
							enemyActions = append(enemyActions, act)
						}
					}
				}
			}
		}
	}

	// Basic attack fallback
	enemyActions = append(enemyActions, simulator.SimAction{
		ActionType: "BASIC_ATTACK",
		Name:       "Đánh thường",
		Damage:     float64(enemy.BaseAttack),
		DamageMult: 1.0,
		ManaCost:   0,
		Priority:   999,
	})

	simEnemy.CritRateBonus = computeGlobalCritRate(selectedGuIDs, loadoutSpec)
	simEnemy.IgnoreDefenseFlat, simEnemy.IgnoreDefensePct = computeGlobalIgnoreDefense(selectedGuIDs, loadoutSpec)
	if cs != nil {
		for i := range enemyActions {
			act := &enemyActions[i]
			switch act.ActionType {
			case "GU_SKILL":
				for _, t := range cs.GuPriorities {
					if t.GuID == act.SourceID {
						act.Priority = t.Priority
						act.UseCondition = t.UseCondition
						act.ConditionValue = t.ConditionValue
						break
					}
				}
			case "KILLER_MOVE":
				for _, t := range cs.KMPriorities {
					if t.KillerMoveID == act.SourceID {
						act.Priority = t.Priority
						act.UseCondition = t.UseCondition
						act.ConditionValue = t.ConditionValue
						break
					}
				}
			}
		}

		if len(cs.CategoryPriorities) > 0 {
			simEnemy.CategoryPriorities = cs.CategoryPriorities
		} else {
			simEnemy.CategoryPriorities = []string{"other", "shield", "heal", "damage"}
		}

		simEnemy.AutoHealHP = cs.AutoHealHP
		simEnemy.HealThreshold = cs.HealThreshold
		simEnemy.AutoRestoreMana = cs.AutoRestoreMana
		simEnemy.ManaThreshold = cs.ManaThreshold
	}

	// Virtual illusions grant unlimited recovery resources so both sides can
	// freely restore mana, matching the initiator's virtual-mode behavior.
	if activeCombat != nil && activeCombat.CombatMode == "virtual" {
		simEnemy.PrimevalStones = 999999
		simEnemy.ImmortalStones = 999999
		simEnemy.AutoRestoreMana = true
	}

	// Passive mechanics come from every equipped, active Gu in enemy inventory.
	// Hunger is not required here: monster Gus fight at Hunger 0 by design and
	// the engine hunger-gate applies to players only.
	var enemyMechanics []simulator.CombatMechanic
	for _, guInst := range enemy.GuInventory {
		if !guInst.Equipped || guInst.Status != "ACTIVE" {
			continue
		}
		spec, ok := models.GetGu(guInst.GuID, guInst.Rank)
		if !ok {
			spec, ok = models.GetGuAny(guInst.GuID)
		}
		if !ok {
			continue
		}
		enemyMechanics = append(enemyMechanics, combatMechanicsForSpec(spec, false)...)
	}
	simEnemy.Mechanics = enemyMechanics

	simEnemy.GuHunger = func() map[string]int {
		gh := make(map[string]int)
		for _, inst := range enemy.GuInventory {
			if !inst.Equipped || inst.Status != "ACTIVE" {
				continue
			}
			gh[inst.InstanceID] = inst.Hunger
		}
		return gh
	}()
	simEnemy.Actions = enemyActions
	return simEnemy
}

// statusStatFactor returns the multiplicative factor the player's active
// statuses apply to one stat ("attack"/"defense"/...): the product of
// (1 + stat_mod) over every active status. Used to scale Gu skill / killer
// move action output so suppression (e.g. Áp Chế Giới Vực −93% tấn công)
// visibly weakens skills instead of being diluted inside the additive
// (DamageFlat + attack) skill formula. Clamped at 0 (never negative).
func statusStatFactor(player *models.Player, stat string) float64 {
	if player == nil || len(player.Statuses) == 0 {
		return 1
	}
	factor := 1.0
	for _, st := range player.Statuses.Active() {
		for _, def := range gamefx.AllStatuses() {
			if def.Key != st.ID {
				continue
			}
			if pct := def.EffectsForValue(st.Value).StatMods[stat]; pct != 0 {
				factor *= (1.0 + pct)
			}
			break
		}
	}
	if factor < 0 {
		return 0
	}
	return factor
}

// buildStatProvenance itemizes how active statuses modified the player's
// combat stats, by re-running the stat pipeline with the percentage stat_mods
// statuses stripped. Attached to simulator combatants so the battle breakdown
// can show e.g. "Trọng Thương −50%" provenance. Returns nil when no active
// status modifies attack/defense/speed/healing_received.
func buildStatProvenance(player *models.Player, finalStats map[string]float64, finalHealingReceived float64) map[string]*simulator.StatProvenance {
	if player == nil || len(player.Statuses) == 0 {
		return nil
	}

	// Flat buff statuses (buff_atk/def/speed/hp) are kept in the base snapshot
	// so the base→final delta isolates exactly the percentage stat_mods.
	basePlayer := *player
	basePlayer.Statuses = nil
	for _, st := range player.Statuses {
		switch st.ID {
		case "buff_atk", "buff_def", "buff_speed", "buff_hp":
			basePlayer.Statuses = append(basePlayer.Statuses, st)
		}
	}
	baseStats := CalculatePlayerStats(&basePlayer)
	baseHealing := gamefx.Get(&basePlayer).HealingReceivedBonus

	modsByStat := map[string][]simulator.StatModSource{}
	for _, st := range player.Statuses.Active() {
		for _, def := range gamefx.AllStatuses() {
			if def.Key != st.ID {
				continue
			}
			eff := def.EffectsForValue(st.Value)
			add := func(stat string, pct float64) {
				if pct == 0 {
					return
				}
				modsByStat[stat] = append(modsByStat[stat], simulator.StatModSource{Key: def.Key, Name: def.Name, Pct: pct})
			}
			add("attack", eff.StatMods["attack"])
			add("defense", eff.StatMods["defense"])
			add("speed", eff.StatMods["speed"])
			add("healing_received", eff.HealingReceivedBonus)
			break
		}
	}

	out := map[string]*simulator.StatProvenance{}
	for stat, mods := range modsByStat {
		if len(mods) == 0 {
			continue
		}
		pv := &simulator.StatProvenance{Stat: stat, Mods: mods}
		if stat == "healing_received" {
			pv.BaseValue = baseHealing
			pv.FinalValue = finalHealingReceived
		} else {
			pv.BaseValue = baseStats[stat]
			pv.FinalValue = finalStats[stat]
		}
		out[stat] = pv
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildSimCombatant constructs a simulator.CombatantAuto from player + Gu loadout.
// Shared by AutoCombatStart (combat_auto.go) and AutoResolveCombat (combat_auto_resolve.go).
func buildSimCombatant(tx *sql.Tx, player *models.Player, isVirtual bool, isInitiator bool, reqLoadout *simulator.CombatLoadout) (*simulator.CombatantAuto, error) {
	playerStats := CalculatePlayerStats(player)
	actor := models.CombatantFromPlayer(player, playerStats)

	var allActions []simulator.SimAction
	var combatantMechanics []simulator.CombatMechanic

	// Build owned equipped+ACTIVE Gu sets and lookup maps
	ownedGuTypeSet := make(map[string]bool)                  // gu_id → true
	ownedGuInstanceSet := make(map[string]bool)              // instance_id → true
	equippedInstByID := make(map[string]models.GuInstance)   // instance_id → equipped ACTIVE instance
	equippedInstByGuID := make(map[string]models.GuInstance) // gu_id → highest-rank equipped ACTIVE instance
	var allEquippedActiveGus []models.GuInstance

	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" && gu.Hunger > 0 {
			ownedGuTypeSet[gu.GuID] = true
			ownedGuInstanceSet[gu.InstanceID] = true
			equippedInstByID[gu.InstanceID] = gu
			allEquippedActiveGus = append(allEquippedActiveGus, gu)
			if best, ok := equippedInstByGuID[gu.GuID]; !ok || gu.Rank > best.Rank {
				equippedInstByGuID[gu.GuID] = gu
			}
		}
	}

	resolveGuInstance := func(id string) (models.GuInstance, bool) {
		if inst, ok := equippedInstByID[id]; ok {
			return inst, true
		}
		if inst, ok := equippedInstByGuID[id]; ok {
			return inst, true
		}
		return models.GuInstance{}, false
	}

	resolveGuSpec := func(inst models.GuInstance) (models.GuDictionaryItem, bool) {
		if spec, found := models.GetGu(inst.GuID, inst.Rank); found {
			return spec, true
		}
		return models.GetGuAny(inst.GuID)
	}

	loadoutSpec := func(id string) (models.GuDictionaryItem, bool) {
		if inst, ok := resolveGuInstance(id); ok {
			return resolveGuSpec(inst)
		}
		return models.GuDictionaryItem{}, false
	}

	var selectedGuInstances []models.GuInstance
	var selectedGuIDs []string
	seenSelectedInstIDs := make(map[string]bool)

	addSelectedGuInstance := func(inst models.GuInstance) {
		if !seenSelectedInstIDs[inst.InstanceID] {
			seenSelectedInstIDs[inst.InstanceID] = true
			selectedGuInstances = append(selectedGuInstances, inst)
			selectedGuIDs = append(selectedGuIDs, inst.InstanceID)
		}
	}

	var selectedKMIDs []string
	var autoHealHP bool
	var healThreshold float64
	var autoRestoreMana bool
	var manaThreshold float64

	if isInitiator && reqLoadout != nil && (len(reqLoadout.GuIds) > 0 || len(reqLoadout.KillerMoveIds) > 0) {
		for _, id := range reqLoadout.GuIds {
			if inst, ok := resolveGuInstance(id); ok {
				addSelectedGuInstance(inst)
			}
		}
		selectedKMIDs = reqLoadout.KillerMoveIds
		autoHealHP = reqLoadout.AutoHealHP
		healThreshold = reqLoadout.HealThreshold
		autoRestoreMana = reqLoadout.AutoRestoreMana
		manaThreshold = reqLoadout.ManaThreshold
	} else {
		// Non-initiator or initiator with empty loadout priorities:
		// use the player's saved combat loadout (gu_priorities)
		// when configured, otherwise fall back to all equipped active Gus.
		settings := player.CombatSettings
		if len(settings.GuPriorities) > 0 {
			for _, t := range settings.GuPriorities {
				if inst, ok := resolveGuInstance(t.GuID); ok {
					addSelectedGuInstance(inst)
				}
			}
		}
		if len(selectedGuInstances) == 0 {
			for _, inst := range allEquippedActiveGus {
				addSelectedGuInstance(inst)
			}
		}

		// Automatically select all active killer moves they have
		kmMap, slotsMap, errLoadBatch := loadKillerMovesBatch(tx, player.ID, nil)
		if errLoadBatch == nil {
			for kmID, km := range kmMap {
				slots := slotsMap[kmID]
				// Check if component Gus are active
				if ownedGuInstanceSet[km.CorePlayerGuID] {
					allSlotsValid := true
					for _, slot := range slots {
						if !ownedGuInstanceSet[slot.PlayerGuID] {
							allSlotsValid = false
							break
						}
					}
					if allSlotsValid {
						selectedKMIDs = append(selectedKMIDs, kmID)
					}
				}
			}
		}

		// Use saved settings or loadout overrides
		if reqLoadout != nil {
			autoHealHP = reqLoadout.AutoHealHP
			healThreshold = reqLoadout.HealThreshold
			autoRestoreMana = reqLoadout.AutoRestoreMana
			manaThreshold = reqLoadout.ManaThreshold
		} else {
			autoHealHP = player.CombatSettings.AutoHealHP
			healThreshold = player.CombatSettings.HealThreshold
			autoRestoreMana = player.CombatSettings.AutoRestoreMana
			manaThreshold = player.CombatSettings.ManaThreshold
		}
	}

	// Passive mechanics come from every equipped, active, fed Gu, independent
	// of whether that Gu is selected as an action.
	for _, instance := range allEquippedActiveGus {
		spec, ok := resolveGuSpec(instance)
		if !ok {
			continue
		}
		combatantMechanics = append(combatantMechanics, combatMechanicsForSpec(spec, false)...)
	}

	// Status suppression/enhancement factors scale the whole action output of
	// Gu skills and killer moves (not the basic attack, whose damage IS the
	// suppressed attack stat).
	attackFactor := statusStatFactor(player, "attack")
	defenseFactor := statusStatFactor(player, "defense")

	// Add selected Gus
	for _, inst := range selectedGuInstances {
		if spec, ok := resolveGuSpec(inst); ok {
			if hasCombatActiveEffects(&spec) {
				act, err := buildSimActionFromGu(&actor, &spec, "GU_SKILL")
				if err != nil {
					continue
				}
				act.SourceID = inst.GuID
				act.InstanceID = inst.InstanceID
				act.Damage *= attackFactor
				act.Defense *= defenseFactor
				allActions = append(allActions, act)
			}
		}
	}

	// Add killer moves
	if len(selectedKMIDs) > 0 {
		kmMap, slotsMap, errLoadBatch := loadKillerMovesBatch(tx, player.ID, selectedKMIDs)
		if errLoadBatch == nil {
			for _, kmID := range selectedKMIDs {
				km, ok := kmMap[kmID]
				if !ok {
					continue
				}
				slots := slotsMap[kmID]
				act := buildSimActionFromKillerMove(km, slots, player.DaoMarks)
				act.SourceID = kmID
				act.Damage *= attackFactor
				act.Defense *= defenseFactor
				allActions = append(allActions, act)
			}
		}
	}

	// Basic attack fallback
	allActions = append(allActions, simulator.SimAction{
		ActionType: "BASIC_ATTACK",
		Name:       "Đánh thường",
		Damage:     playerStats["attack"],
		DamageMult: 1.0,
		ManaCost:   0,
	})

	// Apply persistent tactics (priority + conditions) to actions
	settings := player.CombatSettings
	for i := range allActions {
		act := &allActions[i]
		switch act.ActionType {
		case "GU_SKILL":
			for _, t := range settings.GuPriorities {
				if t.GuID == act.SourceID || (act.InstanceID != "" && t.GuID == act.InstanceID) {
					act.Priority = t.Priority
					act.UseCondition = t.UseCondition
					act.ConditionValue = t.ConditionValue
					break
				}
			}
		case "KILLER_MOVE":
			for _, t := range settings.KMPriorities {
				if t.KillerMoveID == act.SourceID {
					act.Priority = t.Priority
					act.UseCondition = t.UseCondition
					act.ConditionValue = t.ConditionValue
					break
				}
			}
		}
		if act.ActionType == "BASIC_ATTACK" {
			act.Priority = 999
		}
	}

	// Virtual Shield from loadout
	virtualShield := 0
	scanShield := func(spec models.GuDictionaryItem) {
		if spec.Effects.Active != nil && spec.Effects.Active.ShieldFlat > 0 {
			virtualShield += int(spec.Effects.Active.ShieldFlat)
		} else {
			for _, block := range spec.Effects.Blocks {
				if block.Trigger == gameplayeffects.TriggerOnAction {
					for _, op := range block.Operations {
						if op.Op == gameplayeffects.OpShield && (strings.EqualFold(op.SubType, "FLAT") || op.SubType == "") {
							virtualShield += int(op.Value)
						}
					}
				}
			}
		}
	}
	for _, inst := range selectedGuInstances {
		if spec, ok := resolveGuSpec(inst); ok {
			scanShield(spec)
		}
	}

	playerDaoMarks := make(simulator.DaoMarks)
	for k, v := range player.DaoMarks {
		playerDaoMarks[k] = v
	}

	charFX := gamefx.Get(player)

	// Aggregate Gu passive path counters (anti_heaven/earth/dream) into
	// PathResistance, keyed by normalized path ("heaven", "earth", "dream").
	// Copy the map: charFX may be shared/cached per player.
	pathResistance := make(map[string]float64, len(charFX.PathResistanceBonus)+3)
	for k, v := range charFX.PathResistanceBonus {
		pathResistance[k] = v
	}
	immuneDream := false
	for _, inst := range allEquippedActiveGus {
		spec, ok := resolveGuSpec(inst)
		if !ok {
			continue
		}
		// Block v2 only: per-path resistance + dream immunity come from
		// compiled STAT_MOD PATH_RESISTANCE / UNIQUE IMMUNE_DREAM_STATUS.
		// Legacy PassiveEffect.Anti* flat fields are never read.
		for path, pct := range models.GuPathResistance(spec) {
			pathResistance[path] += pct
		}
		if models.GuImmuneDream(spec) {
			immuneDream = true
		}
	}

	// Scan inventory for healing items
	healPotency := 0.0
	healCharges := 0
	if InventorySvc != nil {
		invItems, _ := InventorySvc.ListByPlayer(player.ID.String())
		for _, inv := range invItems {
			if inv.Quantity <= 0 {
				continue
			}
			item, ok := models.ItemRegistry[inv.ItemID]
			if !ok || item.Type != "Consumable" {
				continue
			}
			if healVal, ok := item.Effects["restore_hp"]; ok && healVal > 0 {
				if healPotency == 0 {
					healPotency = healVal
				}
				healCharges += int(inv.Quantity)
			}
		}
	}

	baseHP := player.BaseHP + playerStats["hp"]
	simMaxHP := baseHP
	if simMaxHP <= 0 {
		simMaxHP = 100
	}
	simHP := simMaxHP
	if !isVirtual {
		simHP = player.HP
	}
	if simHP > simMaxHP {
		simHP = simMaxHP
	}
	if simHP <= 0 {
		simHP = simMaxHP
	}

	simMana := player.MaxMana
	if !isVirtual {
		simMana = player.CurrentMana
	}

	simStones := player.PrimevalStones
	simImmortalStones := player.ImmortalStones
	if isVirtual {
		simStones = 999999
		simImmortalStones = 999999
	}

	if player.PathMasteries == nil {
		player.PathMasteries = make(models.PathMasteriesMap)
	}

	satKhiLevel := 0
	for _, st := range player.Statuses {
		if st.ID == "sat_khi" && st.Value > 0 {
			satKhiLevel = int(st.Value)
			break
		}
	}

	simCombatant := &simulator.CombatantAuto{
		ID:                   player.ID.String(),
		Name:                 player.Username,
		Kind:                 "Player",
		Rank:                 player.Rank,
		SubRank:              player.SubRank,
		GridX:                2,
		GridY:                3,
		MaxHP:                simMaxHP,
		CustomState:          map[string]int{"sat_khi_level": satKhiLevel},
		HP:                   simHP,
		MaxMana:              player.MaxMana,
		Mana:                 simMana,
		BaseDmg:              playerStats["attack"],
		BaseDef:              playerStats["defense"],
		Speed:                playerStats["speed"],
		Lifespan:             player.Lifespan,
		VirtualShield:        float64(virtualShield),
		DaoMarks:             simulator.DaoMarks(player.DaoMarks),
		PathMasteries:        player.PathMasteries,
		PathMasteryGainLimit: gamefx.PathMasteryBattleGainLimit(player.Rank),
		DaoBonus:             charFX.DaoNganBonus,
		PathDamageBonus:      charFX.PathDamageBonus,
		PathResistance:       pathResistance,
		HealingReceived:      charFX.HealingReceivedBonus,
		MentalResistance:     charFX.MentalResistanceBonus,
		Tenacity:             charFX.Tenacity,
		CCResist:             charFX.CCResist,
		ImmuneDreamStatus:    immuneDream,
		SoulDefenseBonus:     charFX.SoulDefenseBonus,
		LuckBonus:            charFX.LuckBonus,
		HasNonConflictingDao: charFX.NonConflictingDaoMarks,
		DaoProfile: gamefx.BuildDaoCombatProfile(
			player.DaoMarks,
			player.PathMasteries,
			charFX.NonConflictingDaoMarks,
			charFX.DaoNganBonus,
		),
		Actions:            allActions,
		CategoryPriorities: player.CombatSettings.CategoryPriorities,

		AutoHealHP:        autoHealHP,
		HealThreshold:     healThreshold,
		AutoRestoreMana:   isVirtual || autoRestoreMana,
		ManaThreshold:     manaThreshold,
		HealCharges:       healCharges,
		HealPotency:       healPotency,
		PrimevalStones:    simStones,
		ImmortalStones:    simImmortalStones,
		RefineManaTier:    gamefx.RefineManaTier(player.Rank, player.GuInventory),
		Accuracy:          playerStats["accuracy"],
		CritRateBonus:     computeGlobalCritRate(selectedGuIDs, loadoutSpec),
		IgnoreDefenseFlat: func() float64 { f, _ := computeGlobalIgnoreDefense(selectedGuIDs, loadoutSpec); return f }(),
		IgnoreDefensePct:  func() float64 { _, p := computeGlobalIgnoreDefense(selectedGuIDs, loadoutSpec); return p }(),
		GuHunger: func() map[string]int {
			gh := make(map[string]int)
			for _, inst := range player.GuInventory {
				if inst.Equipped && inst.Status == "ACTIVE" {
					gh[inst.InstanceID] = inst.Hunger
				}
			}
			return gh
		}(),
		GuUsedCounts: make(map[string]int),
		Mechanics:    combatantMechanics,
	}
	simCombatant.StatProvenance = buildStatProvenance(player, playerStats, charFX.HealingReceivedBonus)

	if isInitiator && reqLoadout != nil {
		simCombatant.Loadout = reqLoadout
	} else {
		simCombatant.Loadout = &simulator.CombatLoadout{
			GuIds:           selectedGuIDs,
			KillerMoveIds:   selectedKMIDs,
			AutoHealHP:      autoHealHP,
			HealThreshold:   healThreshold,
			AutoRestoreMana: autoRestoreMana,
			ManaThreshold:   manaThreshold,
		}
	}

	return simCombatant, nil
}
