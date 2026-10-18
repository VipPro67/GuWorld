package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// buildSimCombatState constructs the deterministic simulator state for the
// player's active combat: initiator + pets + party allies + enemies.
//
// Used by AutoCombatStart (manual replay) and the skip-battle paths
// so both paths use identical combatants, turn order and calculations. The
// caller supplies the loadout (request loadout vs saved combat settings) and
// the already-derived isVirtual flag.
func buildSimCombatState(tx *sql.Tx, player *models.Player, loadout *simulator.CombatLoadout, isVirtual bool) (*simulator.CombatState, error) {
	simState := &simulator.CombatState{
		Turn:          0,
		Seed:          time.Now().UnixNano(),
		BossPhase:     0,
		VirtualMode:   isVirtual,
		ApertureHP:    player.ActiveCombat.ApertureHP,
		ApertureMaxHP: player.ActiveCombat.ApertureMaxHP,
	}

	simPlayer, err := buildSimCombatant(tx, player, isVirtual, true, loadout)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize initiator stats: %w", err)
	}
	applyPveTitleDamage(simPlayer, player.ActiveCombat)
	simState.Allies = append(simState.Allies, simPlayer)

	// Load other party members as simulated allies
	for _, ally := range player.ActiveCombat.Allies {
		if ally.ID == player.ID.String() {
			continue
		}
		// Pet allies: build sim combatant directly from ally data
		if ally.Kind == "Pet" {
			simPet := &simulator.CombatantAuto{
				ID:      ally.ID,
				Name:    ally.Name,
				Kind:    "Pet",
				Rank:    ally.Rank,
				SubRank: ally.SubRank,
				GridX:   ally.GridX,
				GridY:   ally.GridY,
				MaxHP:   ally.MaxHP,
				HP:      ally.HP,
				MaxMana: 10,
				Mana:    10,
				BaseDmg: ally.BaseAttack,
				BaseDef: ally.BaseDefense,
				Speed:   ally.Speed,
				Actions: []simulator.SimAction{{
					ActionType: "BASIC_ATTACK",
					Name:       "Đánh thường",
					Damage:     ally.BaseAttack,
					DamageMult: 1.0,
					ManaCost:   0,
				}},
			}
			simState.Allies = append(simState.Allies, simPet)
			continue
		}
		if ally.Kind == "VariantNPC" {
			simState.Allies = append(simState.Allies, buildSimVariantNPC(&ally))
			continue
		}
		// Party member allies
		allyUUID, errParse := uuid.Parse(ally.ID)
		if errParse == nil {
			mPlayer, errMember := evaluateManaTxEquippedReadonly(tx, allyUUID)
			if errMember == nil {
				simAlly, errBuild := buildSimCombatant(tx, &mPlayer, isVirtual, false, nil)
				if errBuild == nil {
					simAlly.Kind = "Ally"
					simAlly.GridX = ally.GridX
					simAlly.GridY = ally.GridY
					simAlly.HP = ally.HP // Use their combat starting HP
					if simAlly.HP > simAlly.MaxHP {
						simAlly.HP = simAlly.MaxHP
					}
					simState.Allies = append(simState.Allies, simAlly)
				}
			}
		}
	}

	// Build enemies
	for idx := range player.ActiveCombat.Enemies {
		enemy := &player.ActiveCombat.Enemies[idx]
		if enemy.Kind == "VariantNPC" {
			simState.Enemies = append(simState.Enemies, buildSimVariantNPC(enemy))
			continue
		}
		if simDef, ok := buildSimPlayerDefender(tx, enemy, player.ActiveCombat, isVirtual); ok {
			simState.Enemies = append(simState.Enemies, simDef)
			continue
		}
		simEnemy := buildSimEnemyCombatant(tx, enemy, player.ActiveCombat)
		simState.Enemies = append(simState.Enemies, simEnemy)
	}

	// Áp Chế Giới Vực giả (FAKE_REGIONAL_WALL): chỉ PvP player-vs-player; giảm
	// công/thủ/tốc của địch theo rank của địch (formula RegionalWallDebuff nhưng giảm 1 nửa).
	if player.ActiveCombat != nil && player.ActiveCombat.EnemyType == "PVP" && playerHasBattleStartMechanic(player, "FAKE_REGIONAL_WALL") {
		for i := range player.ActiveCombat.Enemies {
			if i >= len(simState.Enemies) || simState.Enemies[i] == nil {
				continue
			}
			pct, _ := gamefx.RegionalWallDebuff(player.ActiveCombat.Enemies[i].Rank, player.ActiveCombat.Enemies[i].SubRank)
			debuffPct := pct * 0.5
			factor := 1 + debuffPct
			simState.Enemies[i].BaseDmg *= factor
			simState.Enemies[i].BaseDef *= factor
			simState.Enemies[i].Speed *= factor
			if simState.Enemies[i].CustomState == nil {
				simState.Enemies[i].CustomState = map[string]int{}
			}
			simState.Enemies[i].CustomState["fake_regional_wall"] = 1
			enemyName := simState.Enemies[i].Name
			if enemyName == "" {
				enemyName = "Kẻ địch"
			}
			simState.InitialLogs = append(simState.InitialLogs, simulator.CombatLog{
				Turn:      0,
				ActorId:   player.ID.String(),
				ActorName: player.Username,
				Action:    "FAKE_REGIONAL_WALL",
				Detail:    fmt.Sprintf("Ngũ Hành Thiên Chướng Trận khai mở! Phản chiếu giới vực lên %s, làm suy giảm %.1f%% công/thủ/tốc!", enemyName, -debuffPct*100),
			})
		}
	}

	// Inject Steal Gu mechanics and pre-combat battle logs
	if player.ActiveCombat != nil && player.ActiveCombat.StealLogMessage != "" {
		actionType := "STEAL_GU"
		if player.ActiveCombat.PermStolenGuID != "" {
			actionType = "PERM_STEAL_GU"
		} else if player.ActiveCombat.TempStolenGuID != "" {
			actionType = "TEMP_STEAL_GU"
		}
		simState.InitialLogs = append(simState.InitialLogs, simulator.CombatLog{
			Turn:      0,
			ActorId:   player.ID.String(),
			ActorName: player.Username,
			Action:    actionType,
			Detail:    player.ActiveCombat.StealLogMessage,
		})
	}

	// Inject explicit steal miss/skip logs so replay distinguishes
	// "kích hoạt thất bại" from "không đủ điều kiện".
	if player.ActiveCombat != nil {
		for _, entry := range player.ActiveCombat.StealMissLogs {
			action := "STEAL_MISS"
			detail := entry
			if idx := strings.Index(entry, "|"); idx >= 0 {
				action = entry[:idx]
				detail = entry[idx+1:]
			}
			simState.InitialLogs = append(simState.InitialLogs, simulator.CombatLog{
				Turn:      0,
				ActorId:   player.ID.String(),
				ActorName: player.Username,
				Action:    action,
				Detail:    detail,
			})
		}
	}

	// Inject Tất Sát Cổ precondition logs for every sim combatant carrying
	// the mechanic, so replay shows "đã kích hoạt" vs "không đủ sát khí".
	for _, c := range append(append([]*simulator.CombatantAuto{}, simState.Allies...), simState.Enemies...) {
		if !hasSatKhiMechanic(c) {
			continue
		}
		level := 0
		if c.CustomState != nil {
			level = c.CustomState["sat_khi_level"]
		}
		if level > 10 {
			simState.InitialLogs = append(simState.InitialLogs, simulator.CombatLog{
				Turn:      0,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    "SAT_KHI_ACTIVE",
				Detail:    fmt.Sprintf("[Tất Sát Cổ] của %s đã kích hoạt: sát khí %d > 10, đòn tấn công xuyên khiên/bỏ qua né trong trận này!", c.Name, level),
			})
		} else {
			simState.InitialLogs = append(simState.InitialLogs, simulator.CombatLog{
				Turn:      0,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    "SAT_KHI_INACTIVE",
				Detail:    fmt.Sprintf("[Tất Sát Cổ] của %s không đủ điều kiện kích hoạt: sát khí %d <= 10 (cần > 10).", c.Name, level),
			})
		}
	}

	// For Temporary Steal: disable stolen Gu from victim, and grant skill to thief
	if player.ActiveCombat != nil && player.ActiveCombat.TempStolenGuID != "" {
		stolenGuID := player.ActiveCombat.TempStolenGuID
		stolenInstID := player.ActiveCombat.TempStolenGuInstanceID

		// Disable action on enemies
		for _, enemy := range simState.Enemies {
			var filteredActions []simulator.SimAction
			for _, act := range enemy.Actions {
				if act.SourceID == stolenGuID || (stolenInstID != "" && act.InstanceID == stolenInstID) {
					continue
				}
				filteredActions = append(filteredActions, act)
			}
			enemy.Actions = filteredActions
		}

		// Grant stolen Gu's active skill to player
		if stolenSpec, ok := models.GetGu(stolenGuID, player.ActiveCombat.TempStolenGuRank); ok {
			if hasCombatActiveEffects(&stolenSpec) {
				actor := models.CombatantFromPlayer(player, CalculatePlayerStats(player))
				stolenAction, err := buildSimActionFromGu(&actor, &stolenSpec, "GU_SKILL")
				if err == nil {
					attackFactor := statusStatFactor(player, "attack")
					defenseFactor := statusStatFactor(player, "defense")
					stolenAction.SourceID = stolenSpec.ID
					stolenAction.InstanceID = stolenInstID
					stolenAction.Damage *= attackFactor
					stolenAction.Defense *= defenseFactor
					simPlayer.Actions = append(simPlayer.Actions, stolenAction)
				}
			}
		}
	}

	return simState, nil
}

func applyPveTitleDamage(combatant *simulator.CombatantAuto, activeCombat *models.CombatState) {
	if combatant == nil || activeCombat == nil || activeCombat.TitleEffectSnapshot == nil ||
		!isPveMonsterType(activeCombat.EnemyType) ||
		!strings.EqualFold(activeCombat.TitleEffectSnapshot.MonsterType, activeCombat.EnemyType) ||
		activeCombat.TitleEffectSnapshot.DamagePct <= 0 {
		return
	}
	factor := 1 + activeCombat.TitleEffectSnapshot.DamagePct/100
	combatant.BaseDmg *= factor
	for i := range combatant.Actions {
		combatant.Actions[i].Damage *= factor
		combatant.Actions[i].BleedPerTurn *= factor
		combatant.Actions[i].PoisonPerTurn *= factor
		combatant.Actions[i].PhantomDmg *= factor
	}
}

func buildSimVariantNPC(actor *models.Combatant) *simulator.CombatantAuto {
	actions := []simulator.SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: actor.BaseAttack, DamageMult: 1}}
	mechanics := []simulator.CombatMechanic{}
	hunger := map[string]int{}
	for _, gu := range actor.GuInventory {
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok || gu.Status != "ACTIVE" || gu.Hunger <= 0 {
			continue
		}
		mechanics = append(mechanics, combatMechanicsForSpec(spec, false)...)
		hunger[gu.InstanceID] = gu.Hunger
		if hasCombatActiveEffects(&spec) {
			if act, err := buildSimActionFromGu(actor, &spec, "GU_SKILL"); err == nil {
				act.InstanceID = gu.InstanceID
				act.SourceID = gu.GuID
				actions = append(actions, act)
			}
		}
	}
	return &simulator.CombatantAuto{ID: actor.ID, Name: actor.Name, Kind: "VariantNPC", Rank: actor.Rank, SubRank: actor.SubRank,
		GridX: actor.GridX, GridY: actor.GridY, HP: actor.HP, MaxHP: actor.MaxHP, Mana: actor.CurrentMana, MaxMana: actor.MaxMana,
		BaseDmg: actor.BaseAttack, BaseDef: actor.BaseDefense, Speed: actor.Speed, Actions: actions, Mechanics: mechanics, GuHunger: hunger}
}

// buildSimPlayerDefender rebuilds a real-player PVP defender with the same
// buildSimCombatant pipeline used for the initiator, so passive mechanics,
// killer moves and global buffs load identically on both sides. Returns
// ok=false for monsters, pets, masked arena opponents (ARENA_IDENTITY_MASK)
// and tx-less callers, which fall back to buildSimEnemyCombatant.
func buildSimPlayerDefender(tx *sql.Tx, enemy *models.Combatant, activeCombat *models.CombatState, isVirtual bool) (*simulator.CombatantAuto, bool) {
	if tx == nil || enemy == nil || activeCombat == nil {
		return nil, false
	}
	isPlayer := enemy.Kind == "Player" ||
		(activeCombat.EnemyType == "PVP" && activeCombat.TargetPlayerID != "" &&
			enemy.Kind != "Pet" && enemy.Kind != "Boss" && enemy.Kind != "Normal")
	if !isPlayer {
		return nil, false
	}

	// (exploration) leave EnemyTemplateID empty, so fall back to TargetPlayerID.
	defenderID := enemy.ID
	if _, errParseID := uuid.Parse(defenderID); errParseID != nil {
		defenderID = activeCombat.TargetPlayerID
	}
	defUUID, errParse := uuid.Parse(defenderID)
	if errParse != nil {
		return nil, false
	}
	defPlayer, errLoad := evaluateManaTxEquippedReadonly(tx, defUUID)
	if errLoad != nil {
		logger.S.Warnf("buildSimPlayerDefender: failed to load defender %s: %v", enemy.ID, errLoad)
		return nil, false
	}
	simDef, errBuild := buildSimCombatant(tx, &defPlayer, isVirtual, false, nil)
	if errBuild != nil {
		logger.S.Warnf("buildSimPlayerDefender: failed to build defender %s: %v", enemy.ID, errBuild)
		return nil, false
	}

	// Keep the recorded encounter identity, grid slot and HP/mana snapshot.
	simDef.Kind = "Enemy"
	simDef.GridX = enemy.GridX
	simDef.GridY = enemy.GridY
	if enemy.MaxHP > 0 {
		simDef.MaxHP = enemy.MaxHP
	}
	if enemy.HP > 0 {
		simDef.HP = enemy.HP
	}
	if simDef.HP > simDef.MaxHP {
		simDef.HP = simDef.MaxHP
	}
	if enemy.MaxMana > 0 {
		simDef.MaxMana = enemy.MaxMana
	}
	if enemy.CurrentMana > 0 {
		simDef.Mana = enemy.CurrentMana
	}
	if simDef.Mana > simDef.MaxMana {
		simDef.Mana = simDef.MaxMana
	}
	return simDef, true
}

// applyLoveGuPenalties scans the combat log for Chaos Gamble (Love Gu) penalties
// and applies them to the player. Shared by all auto-combat paths
// so both paths penalize identically.
func applyLoveGuPenalties(player *models.Player, logs []simulator.CombatLog) {
	loveAptitudePenalty := 0
	loveLifespanPenalty := 0.0
	loveStonesPenalty := false
	for _, logMsg := range logs {
		if logMsg.Action != "CHAOS_GAMBLE" {
			continue
		}
		if strings.Contains(logMsg.Detail, "Trừ 10% Tư chất gốc vĩnh viễn") {
			loveAptitudePenalty += 10
		} else if strings.Contains(logMsg.Detail, "Trừ 5 năm tuổi thọ vĩnh viễn") {
			loveLifespanPenalty += 5.0
		} else if strings.Contains(logMsg.Detail, "Trừ 50% Nguyên Thạch hiện tại") {
			loveStonesPenalty = true
		}
	}

	if loveAptitudePenalty > 0 {
		player.AptitudePercent -= loveAptitudePenalty
		if player.AptitudePercent < 1 {
			player.AptitudePercent = 1
		}
	}
	if loveLifespanPenalty > 0 && !gamefx.Get(player).Immortal {
		player.Lifespan = math.Max(0.0, player.Lifespan-loveLifespanPenalty)
	}
	if loveStonesPenalty {
		player.PrimevalStones = player.PrimevalStones / 2
	}
}

// combatOutcome derives the post-simulation outcome from the sim state.
// alliesAlive is true only when the initiator is alive AND all enemies are dead.
// Returns the initiator's final HP too. Shared by AutoCombatStart and
// ResolveCombat so all paths judge victory identically.
func combatOutcome(simState *simulator.CombatState, playerID string) (enemiesAlive, alliesAlive bool, finalPlayerHP float64) {
	for _, e := range simState.Enemies {
		if e.HP > 0 {
			enemiesAlive = true
			break
		}
	}
	for _, a := range simState.Allies {
		if a.ID == playerID {
			alliesAlive = (a.HP > 0) && !enemiesAlive
			finalPlayerHP = a.HP
		}
	}
	return
}

// applySimConsumption deducts simulator-consumed healing items, primeval stones,
// immortal stones and equipped-Gu hunger from the player, returning the usage
// counts plus the corresponding log lines. Shared by AutoCombatStart and
// ResolveCombat so resource consumption is identical on all paths.
func applySimConsumption(tx *sql.Tx, player *models.Player, simPlayer *simulator.CombatantAuto, isVirtual bool, turn int) (itemsUsed int, stonesSpent, immortalStonesSpent float64, usageLogs []simulator.CombatLog) {
	itemsUsed = 0
	stonesSpent = 0
	immortalStonesSpent = 0

		if simPlayer != nil && len(simPlayer.GuUsedCounts) > 0 {
			beforeGus := append(models.GuInventory(nil), player.GuInventory...)
			hungerDeducted := false
			for idx := range player.GuInventory {
				g := &player.GuInventory[idx]
				count, used := simPlayer.GuUsedCounts[g.InstanceID]
				if !used && g.GuID != "" {
					count, used = simPlayer.GuUsedCounts[g.GuID]
				}
				if used && count > 0 {
					g.Hunger -= count
					if g.Hunger < 0 {
						g.Hunger = 0
					}
					hungerDeducted = true
				}
			}
			if hungerDeducted && tx != nil {
				if errSync := db.SyncPlayerGus(tx, player.ID.String(), beforeGus, player.GuInventory); errSync != nil {
					logger.S.Warnf("Failed to sync player Gu hunger after auto combat: %v", errSync)
				}
			}
			if tx != nil {
				for moveID, uses := range simPlayer.KillerMoveUsedCounts {
					if uses <= 0 {
						continue
					}
					if _, err := tx.Exec(`UPDATE player_killer_moves SET mastery_xp=LEAST(5000, mastery_xp+$1),updated_at=NOW() WHERE id=$2 AND player_id=$3`, uses, moveID, player.ID); err != nil {
						logger.S.Warnf("Failed to update killer move mastery %s: %v", moveID, err)
					}
				}
			}
		}

	if itemsUsed > 0 {
		usageLogs = append(usageLogs, simulator.CombatLog{
			Turn:   turn,
			Action: "AUTO_ITEM_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %d dược liệu hồi phục trong chiến đấu.", itemsUsed),
		})
	}
	if stonesSpent > 0 {
		usageLogs = append(usageLogs, simulator.CombatLog{
			Turn:   turn,
			Action: "AUTO_STONE_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %.3f Nguyên Thạch hồi phục chân nguyên.", stonesSpent),
		})
	}
	if immortalStonesSpent > 0 {
		usageLogs = append(usageLogs, simulator.CombatLog{
			Turn:   turn,
			Action: "AUTO_STONE_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %.3f Tiên Nguyên Thạch hồi phục tiên nguyên.", immortalStonesSpent),
		})
	}
	return
}

// hasSatKhiMechanic reports whether a sim combatant carries Tất Sát Cổ's
// ON_BATTLE_START/SAT_KHI_GUARANTEE passive.
func hasSatKhiMechanic(c *simulator.CombatantAuto) bool {
	if c == nil {
		return false
	}
	for _, m := range c.Mechanics {
		if m.Trigger == "ON_BATTLE_START" && m.ActionCode == "SAT_KHI_GUARANTEE" {
			return true
		}
	}
	return false
}

// combatTypeLabel returns the human-readable enemy-type label used in the
// COMBAT_TYPE log header. Shared by both auto-combat paths.
func combatTypeLabel(cm *models.CombatState) string {
	if cm == nil {
		return "Dã ngoại"
	}
	switch cm.EnemyType {
	case "MINE_PVE":
		return "Mỏ Linh Thạch (Yêu quái)"
	case "PVP":
		return "Đấu trường / PVP"
	case "TRIBULATION_BOSS", "TRIBULATION_MONSTER":
		return "Tai Kiếp Thiên Địa"
	default:
		return "Dã ngoại"
	}
}
