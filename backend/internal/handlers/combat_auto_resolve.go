package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// AutoResolveCombat runs a combat simulation using the player's saved combat settings
// and applies the results to the database. Used for "skip battle" feature.
// Returns: winner ("Phe Ta" or "Phe Địch"), logs, summary info map, error.
func AutoResolveCombat(tx *sql.Tx, player *models.Player) (string, []simulator.CombatLog, map[string]interface{}, error) {
	if player.ActiveCombat == nil {
		return "", nil, nil, fmt.Errorf("no active combat")
	}

	isVirtual := player.ActiveCombat.CombatMode == "virtual"
	isTribulation := player.ActiveCombat.TribulationEventID != ""

	// Build loadout from player's saved combat settings
	loadout := simulator.CombatLoadout{
		AutoHealHP:      player.CombatSettings.AutoHealHP,
		HealThreshold:   player.CombatSettings.HealThreshold,
		AutoRestoreMana: player.CombatSettings.AutoRestoreMana,
		ManaThreshold:   player.CombatSettings.ManaThreshold,
	}

	simState, err := buildSimCombatState(tx, player, &loadout, isVirtual)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to initialize initiator stats: %w", err)
	}
	simPlayer := simState.Allies[0]

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
			cInst.GuInventory = filterCombatGuInventory(player.GuInventory, nil, player.CombatSettings)
		}
		combatantInfos = append(combatantInfos, cInst)
	}
	if player.ActiveCombat != nil {
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
	}

	// Run Simulator
	logs := simulator.Simulate(simState)

	_, alliesAlive, finalPlayerHP := combatOutcome(simState, player.ID.String())

	// Apply outcome to DB (simplified from AutoCombatStart)
	before := append(models.GuInventory(nil), player.GuInventory...)

	applyLoveGuPenalties(player, logs)

	itemsUsed, stonesSpent, _, usageLogs := applySimConsumption(tx, player, simPlayer, isVirtual, simState.Turn)

	var finalMana float64
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			finalMana = a.Mana
			break
		}
	}

	logs = append(logs, usageLogs...)

	var updateQuery string
	var args []interface{}

	enemyName := ""
	if player.ActiveCombat != nil {
		enemyName = player.ActiveCombat.EnemyName
	}
	enemyType := combatTypeLabel(player.ActiveCombat)
	logs = append([]simulator.CombatLog{{
		Turn:   0,
		Action: "COMBAT_TYPE",
		Detail: fmt.Sprintf("=== %s ===", enemyType),
	}}, logs...)

	summary := make(map[string]interface{})

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
			updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
			args = append(args, player.ID)
			summary["reward"] = "50 Xu Đấu Trường"
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

			lootDetailLogs := applyCombatVictoryLoot(tx, player, player.ActiveCombat, partyMembers, isVirtual)

			markBossCleared(tx, player)

			for _, lMsg := range lootDetailLogs {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "LOOT",
					Detail: lMsg,
				})
			}

			mineLogs, mineOccupied := applyAutoCombatMineOccupation(tx, player, player.ActiveCombat, simState.Turn)
			logs = append(logs, mineLogs...)
			if mineOccupied {
				summary["mine_occupied"] = player.ActiveCombat.MineSlotID
			}

			// Sect War Victory/Defeat
			if player.ActiveCombat.TargetSectID != "" {
				// Determine winner: check combat logs for "Trận kết thúc. Phe X thắng!"
				var winner string
				for i := len(logs) - 1; i >= 0; i-- {
					logDetail := logs[i].Detail
					if strings.Contains(logDetail, "Trận chiến kết thúc") {
						if strings.Contains(logDetail, "Phe Ta") {
							winner = "attacker"
						} else if strings.Contains(logDetail, "Phe Địch") {
							winner = "defender"
						}
						break
					}
				}

				if winner == "attacker" {
					// Attacker won - apply victory: plunder, shield, rewards
					sectWarLogs := ApplySectWarVictory(tx, player, player.ActiveCombat)
					for _, swMsg := range sectWarLogs {
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "SECT_WAR",
							Detail: swMsg,
						})
					}
					summary["sect_war_won"] = true
				} else {
					// Defender won (attacker lost) - apply defeat: contribution points, shield, teleport
					sectDefeatLogs := ApplySectWarDefeat(tx, player, player.ActiveCombat)
					for _, sdMsg := range sectDefeatLogs {
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "SECT_WAR_DEFEAST",
							Detail: sdMsg,
						})
					}
					// Apply death-like penalty: teleport to hub, death drop
					player.HP = 1.0
					deathRegion, deathX, deathY := player.CurrentRegion, player.X, player.Y
					player.CurrentRegion, player.X, player.Y = NearestHub(deathX, deathY)

					// Apply death drop (may drop items)
					dropSummary := applyDeathDrop(tx, player, nil, false, DeathPos{Region: deathRegion, X: deathX, Y: deathY})
					if dropSummary != "" {
						playerMsg := fmt.Sprintf("Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết nhưng rơi rớt mất: %s.", dropSummary)
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "BATTLE_END",
							Detail: playerMsg,
						})
					}

					// Update player position in DB
					_, _ = tx.Exec(`UPDATE players SET current_region = $1, x = $2, y = $3 WHERE id = $4`,
						player.CurrentRegion, player.X, player.Y, player.ID)
				}
			}

			if player.ActiveCombat.TargetSectID == "" {
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
			}
			clearedCellsJSON, _ := json.Marshal(player.ClearedCells)

			player.HP = float64(finalPlayerHP)
			updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, primeval_stones = $3, immortal_stones = $4, path_masteries = $5, aptitude_percent = $6, lifespan = $7, cleared_cells = $8, is_dead = $9, version = version + 1 WHERE id = $10`
			args = append(args, player.HP, finalMana, player.PrimevalStones, player.ImmortalStones, player.PathMasteries, player.AptitudePercent, player.Lifespan, clearedCellsJSON, player.IsDead, player.ID)

			summary["loot"] = lootDetailLogs
		}
	} else {
		defUpdateQuery, defArgs, defLogs := applyAutoCombatDefeat(tx, player, simState, isVirtual, isTribulation, finalMana)
		updateQuery = defUpdateQuery
		args = append(args, defArgs...)
		logs = append(logs, defLogs...)
	}

	logs, err = finalizeAutoCombatOutcome(tx, player, simState, isVirtual, isTribulation, alliesAlive, updateQuery, args, finalPlayerHP, finalMana, before, logs)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to finalize combat outcome: %w", err)
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 3, $1) WHERE id = $2", MaxHumanQi, player.ID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to update human_qi: %w", err)
	}

	winner := "Phe Địch"
	if alliesAlive {
		winner = "Phe Ta"
	}

	// Save combat history in background
	targetPlayerID := ""
	if player.ActiveCombat != nil {
		targetPlayerID = player.ActiveCombat.TargetPlayerID
	}
	go func() {
		summaryStr := buildCombatSummaryFromLogs(logs, finalPlayerHP, itemsUsed, int64(stonesSpent))
		combatantInfosIface := make([]interface{}, len(combatantInfos))
		for i, ci := range combatantInfos {
			combatantInfosIface[i] = ci
		}
		SaveCombatHistory(player.ID, winner, toInterfaceSlice(logs), summaryStr, combatantInfosIface, enemyName, enemyType, player.CurrentRegion)

		if enemyType == "PVP" && targetPlayerID != "" && targetPlayerID != "dummy" {
			defenderWinner := "Phe Ta"
			if alliesAlive {
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
			SaveCombatHistory(uuid.MustParse(targetPlayerID), defenderWinner, toInterfaceSlice(logs), summaryStr, defenderCombatantInfos, player.Username, "PVP", player.CurrentRegion)
		}
	}()

	summary["winner"] = winner
	summary["enemy_name"] = enemyName
	summary["enemy_type"] = enemyType

	player.ActiveCombat = nil
	return winner, logs, summary, nil
}

// ── helpers ────────────────────────────────────────────────────────────

func scanKillerMoveRow(rows *sql.Rows) (models.KillerMove, error) {
	var m models.KillerMove
	err := rows.Scan(
		&m.ID, &m.PlayerID, &m.Name, &m.Path, &m.MoveType, &m.RankRequired,
		&m.CorePlayerGuID, &m.TotalManaCost, &m.SuccessRate, &m.DamageValue,
		&m.DefenseValue, &m.Cooldown, &m.ChannelTime, &m.Effects, &m.IsMastered,
		&m.IsReliable, &m.IsInstantCast, &m.CreatedAt, &m.UpdatedAt,
	)
	return m, err
}

func loadKillerMovesBatch(tx *sql.Tx, playerID uuid.UUID, ids []string) (map[string]models.KillerMove, map[string][]models.KillerMoveSlot, error) {
	if tx == nil {
		return nil, nil, nil
	}
	var query string
	var args []any
	if ids == nil {
		query = `SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, success_rate, damage_value, defense_value, cooldown, channel_time, effects, is_mastered, is_reliable, is_instant_cast, created_at, updated_at 
		         FROM killer_moves WHERE player_id = $1`
		args = []any{playerID}
	} else {
		if len(ids) == 0 {
			return nil, nil, nil
		}
		query = `SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, success_rate, damage_value, defense_value, cooldown, channel_time, effects, is_mastered, is_reliable, is_instant_cast, created_at, updated_at 
		         FROM killer_moves WHERE player_id = $1 AND id = ANY($2)`
		args = []any{playerID, pq.Array(ids)}
	}

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	kmMap := make(map[string]models.KillerMove)
	var kmIDs []string
	for rows.Next() {
		m, err := scanKillerMoveRow(rows)
		if err != nil {
			return nil, nil, err
		}
		kmMap[m.ID] = m
		kmIDs = append(kmIDs, m.ID)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	slotsMap := make(map[string][]models.KillerMoveSlot)
	if len(kmIDs) > 0 {
		slotRows, err := tx.Query(`SELECT id, killer_move_id, player_gu_id, slot_type, slot_index, created_at 
		                           FROM killer_move_slots WHERE killer_move_id = ANY($1) 
		                           ORDER BY killer_move_id, slot_index ASC`, pq.Array(kmIDs))
		if err != nil {
			return nil, nil, err
		}
		defer slotRows.Close()

		for slotRows.Next() {
			var s models.KillerMoveSlot
			if err := slotRows.Scan(&s.ID, &s.KillerMoveID, &s.PlayerGuID, &s.SlotType, &s.SlotIndex, &s.CreatedAt); err != nil {
				return nil, nil, err
			}
			slotsMap[s.KillerMoveID] = append(slotsMap[s.KillerMoveID], s)
		}
		if err = slotRows.Err(); err != nil {
			return nil, nil, err
		}
	}

	return kmMap, slotsMap, nil
}

func buildSimActionFromGu(actor *models.Combatant, spec *models.GuDictionaryItem, actionType string) (simulator.SimAction, error) {
	// Mortal users cannot use immortal Gu that cost mana (skip the action).
	if !gamefx.CanUseGuWithMana(actor.Rank, spec) {
		return simulator.SimAction{}, fmt.Errorf("Tu vi %s chưa đủ để sử dụng Tiên Cổ %s (Rank %d) có hao phí!", models.GetRankName(actor.Rank), spec.Name, spec.Rank)
	}

	dmg, def, cost, err := CalculateGuEffectForCombatant(actor, spec, false)
	if err != nil {
		return simulator.SimAction{
			ActionType: actionType,
			Name:       spec.Name,
			Path:       spec.Path,
			Rank:       spec.Rank,
			ManaCost:   cost,
		}, nil
	}

	act := simulator.SimAction{
		ActionType: actionType,
		Name:       spec.Name,
		Path:       spec.Path,
		Rank:       spec.Rank,
		Damage:     dmg,
		Defense:    def,
		ManaCost:   cost,
	}
	for _, mechanic := range spec.Effects.Other {
		if mechanic.EventTrigger == "ON_CAST" || mechanic.EventTrigger == "ON_HIT" {
			act.Mechanics = append(act.Mechanics, simulator.CombatMechanic{
				Trigger: mechanic.EventTrigger, ActionCode: mechanic.ActionCode, Params: append([]byte(nil), mechanic.Params...),
			})
		}
	}

	if actor.Rank < spec.Rank {
		act.BacklashDamage = gamefx.HigherRankHPCost(spec.Rank, actor.Rank, actor.MaxHP)
	}

	if spec.Effects.Active != nil {
		active := spec.Effects.Active
		if active.ShieldFlat > 0 {
			act.Defense += active.ShieldFlat
		}
		// Registry-driven statuses (BLEED/POISON/BURN/FEAR/CONFUSE/STEALTH/
		// SLEEP/BLIND/CHARM/BIND + catalog). Resolved with real turns from
		// status_duration_ticks instead of a hardcoded 3.
		if active.ApplyStatus != "" {
			switch active.ApplyStatus {
			// Tier-based legacy statuses keep their existing field mapping.
			case "STUN", "SLOW", "SLOW1", "SLOW_DAMAGE", "WEAK", "FREEZE",
				"SILENCE", "THORNS", "REGEN", "ATK_UP", "DEF_UP", "TAUNT":
				// handled below
			default:
				if op, err := statusfx.ResolveStatus(active.ApplyStatus, active.StatusTickDamage, active.StatusDurationTicks); err == nil {
					act.StatusOps = append(act.StatusOps, op)
				} else {
					logger.S.Warnf("Gu %s: %v", spec.ID, err)
				}
			}
		}
		if active.ApplyStatus == "STUN" {
			act.StunTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "SLOW" {
			act.SlowTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "SLOW1" {
			act.Slow1Tier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "SLOW_DAMAGE" {
			act.SlowDamageTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "WEAK" {
			act.WeakTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "FREEZE" {
			act.FreezeTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "SILENCE" {
			act.SilenceTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "THORNS" {
			act.ThornsTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "REGEN" {
			act.RegenTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "ATK_UP" {
			act.AtkUpTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "DEF_UP" {
			act.DefUpTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "TAUNT" {
			act.TauntTier = active.StatusDurationTicks
		}
		if active.LifeStealRate > 0 {
			act.LifeStealPct += active.LifeStealRate
		}
		if active.HealFlat > 0 {
			act.Healing += active.HealFlat
		}
		if active.BloodDrain {
			act.LifeStealPct += 0.5
		}
		if active.SummonPhantom {
			act.PhantomDmg += actor.BaseAttack
		}
		if active.RefineManaTier > 0 {
			boost := active.RefineManaTier * 0.20
			act.Damage += act.Damage * boost
		}
		if active.WoodTransformation {
			act.Defense += 30.0
		}
		if active.DamageMult > 1.0 {
			act.DamageMult = active.DamageMult
		}
		if active.TargetSelector != "" {
			act.TargetType = active.TargetSelector
			act.TargetX = actor.GridX
			act.TargetY = actor.GridY
		}
		act.Bypass = active.CurveAttack
	}

	if act.TargetType == "" {
		act.TargetType = "SINGLE"
	}

	return act, nil
}

// hasCombatActiveEffects returns true if the Gu's active effects produce a meaningful combat action.
func hasCombatActiveEffects(spec *models.GuDictionaryItem) bool {
	for _, mechanic := range spec.Effects.Other {
		if mechanic.EventTrigger == "ON_CAST" || mechanic.EventTrigger == "ON_HIT" {
			return true
		}
	}
	if spec.Effects.Active == nil {
		return false
	}
	active := spec.Effects.Active
	return active.DamageFlat > 0 || active.DamageMult > 0 || active.ShieldFlat > 0 ||
		active.ApplyStatus != "" || active.LifeStealRate > 0 || active.HealFlat > 0 ||
		active.BloodDrain || active.SummonPhantom || active.RefineManaTier > 0 ||
		active.WoodTransformation || active.ManaRegen > 0 || active.CleansingPower > 0 ||
		active.IgnoreDefense > 0 || active.IgnoreDefensePct > 0 || active.CritRateBoost > 0 ||
		active.AllStatsBuffPct > 0 || active.SpeedBoostFlat > 0 || active.SummonGrowthPhantoms
}

// hasCombatEffects returns true if the Gu has any active action, passive mechanic, or combat-relevant passive stat.
func hasCombatEffects(spec *models.GuDictionaryItem) bool {
	if hasCombatActiveEffects(spec) {
		return true
	}
	for _, mechanic := range spec.Effects.Other {
		if mechanic.EventTrigger != "ON_CAST" && mechanic.EventTrigger != "ON_HIT" {
			return true
		}
	}
	if spec.Effects.Passive != nil {
		p := spec.Effects.Passive
		if p.StatBaseAttack > 0 || p.StatBaseDefense > 0 || p.StatBaseHP > 0 || p.StatBaseSpeed > 0 || p.DamageMultiplierOnHit > 0 || p.Evasion > 0 || p.PathConflictReduction > 0 {
			return true
		}
	}
	return false
}

func computeGlobalGuDamageMultiplier(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	multiplier := 1.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}

		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		if spec.Effects.Active.DamageMult > 1.0 {
			multiplier *= spec.Effects.Active.DamageMult
		}
	}
	return multiplier
}

// computeGlobalIgnoreDefense aggregates ignore_defense from all equipped active Gu (additive, capped at 0.8).
func computeGlobalIgnoreDefense(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) (flat float64, pct float64) {
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		flat += spec.Effects.Active.IgnoreDefense
		pct += spec.Effects.Active.IgnoreDefensePct
	}
	if pct > 0.8 {
		pct = 0.8
	}
	return flat, pct
}

// computeGlobalCritRate aggregates crit_rate_boost from all equipped active Gu (additive, capped at 0.8).
func computeGlobalCritRate(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	total := 0.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		total += spec.Effects.Active.CritRateBoost
	}
	if total > 0.8 {
		total = 0.8
	}
	return total
}

// computeGlobalSpeedBoost aggregates speed_boost_flat from all equipped active Gu (additive).
func computeGlobalSpeedBoost(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	total := 0.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		total += spec.Effects.Active.SpeedBoostFlat
	}
	return total
}

// computeGlobalAllStatsBuff aggregates all_stats_buff_percent from all equipped active Gu (additive, capped at 0.5).
func computeGlobalAllStatsBuff(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	total := 0.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		total += spec.Effects.Active.AllStatsBuffPct
	}
	if total > 0.5 {
		total = 0.5
	}
	return total
}

func buildSimActionFromKillerMove(km models.KillerMove, _ []models.KillerMoveSlot, daoMarks models.DaoMarksMap) simulator.SimAction {
	act := simulator.SimAction{
		ActionType: "KILLER_MOVE",
		Name:       km.Name,
		Damage:     km.DamageValue,
		Defense:    km.DefenseValue,
		ManaCost:   km.TotalManaCost,
		Path:       []string{km.Path},
	}

	// Strip the dao bonus baked in at creation so the engine re-applies it
	// using the current dao marks (and reports it in the combat log).
	if mult, ok := simulator.DaoMultiplier(simulator.DaoMarks(daoMarks), act.Path); ok && mult > 1.0 {
		act.Damage /= mult
		act.Defense /= mult
	}

	var effects map[string]float64
	if err := json.Unmarshal(km.Effects, &effects); err == nil {
		if v, ok := effects["bleed"]; ok {
			act.BleedPerTurn += v
		}
		if v, ok := effects["poison"]; ok {
			act.PoisonPerTurn += v
		}
		if v, ok := effects["stun"]; ok {
			act.StunTier = int(v)
		}
		if v, ok := effects["life_steal"]; ok {
			act.LifeStealPct += v / 100.0
		}
		if v, ok := effects["blood_drain"]; ok {
			act.LifeStealPct += v / 100.0
		}
		if v, ok := effects["multi_hit"]; ok {
			act.MultiHit = int(v)
		}
		if v, ok := effects["phantom_attack"]; ok {
			act.PhantomDmg += v
		}
		if v, ok := effects["thorns_damage"]; ok {
			act.ThornsDmg += v
		}
		if v, ok := effects["heal"]; ok {
			act.Healing += v
		}
		// ── New tier-based effects ──
		if v, ok := effects["slow"]; ok {
			act.SlowTier = int(v)
		}
		if v, ok := effects["slow1"]; ok {
			act.Slow1Tier = int(v)
		}
		if v, ok := effects["slow_damage"]; ok {
			act.SlowDamageTier = int(v)
		}
		if v, ok := effects["weak"]; ok {
			act.WeakTier = int(v)
		}
		if v, ok := effects["freeze"]; ok {
			act.FreezeTier = int(v)
		}
		if v, ok := effects["silence"]; ok {
			act.SilenceTier = int(v)
		}
		if v, ok := effects["thorns_buff"]; ok {
			act.ThornsTier = int(v)
		}
		if v, ok := effects["regen"]; ok {
			act.RegenTier = int(v)
		}
		if v, ok := effects["atk_up"]; ok {
			act.AtkUpTier = int(v)
		}
		if v, ok := effects["def_up"]; ok {
			act.DefUpTier = int(v)
		}
		if v, ok := effects["taunt"]; ok {
			act.TauntTier = int(v)
		}
	}

	act.TargetType = "SINGLE"
	return act
}
