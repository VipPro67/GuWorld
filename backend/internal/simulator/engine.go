package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func hasMechanic(mechanics []CombatMechanic, trigger, actionCode string) bool {
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			return true
		}
	}
	return false
}

func mechanicFloat(mechanics []CombatMechanic, trigger, actionCode, key string, fallback float64) float64 {
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			if value, ok := mechanic.Values[key].(float64); ok {
				return value
			}
		}
	}
	return fallback
}

func mechanicString(mechanics []CombatMechanic, trigger, actionCode, key, fallback string) string {
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			if value, ok := mechanic.Values[key].(string); ok {
				return value
			}
		}
	}
	return fallback
}

func mechanicInt(mechanics []CombatMechanic, trigger, actionCode, key string, fallback int) int {
	return int(mechanicFloat(mechanics, trigger, actionCode, key, float64(fallback)))
}

func mechanicBool(mechanics []CombatMechanic, trigger, actionCode, key string, fallback bool) bool {
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			if value, ok := mechanic.Values[key].(bool); ok {
				return value
			}
		}
	}
	return fallback
}

// lifespanByRank approximates an enemy's thọ nguyên from its rank. Players use
// their tracked Lifespan; monsters have none, so this is the calibration knob
// for ON_BATTLE_START/SEIZE_INITIATIVE. Tune here, not per-Gu.
func lifespanByRank(rank, subRank int) float64 {
	table := [...]float64{10, 20, 50, 100, 200, 500, 1000, 2000, 5000}
	if rank < 1 {
		rank = 1
	}
	if rank > len(table) {
		rank = len(table)
	}
	return table[rank-1] + float64(subRank)*10
}

// combatantLifespan returns the tracked lifespan, falling back to the rank
// approximation for combatants (monsters) that have none.
func combatantLifespan(c *CombatantAuto) float64 {
	if c == nil {
		return 0
	}
	if c.Lifespan > 0 {
		return c.Lifespan
	}
	return lifespanByRank(c.Rank, c.SubRank)
}

// opponentPrimary returns the first combatant on the opposing side, used as the
// reference for lifespan-gated battle-start mechanics.
func opponentPrimary(state *CombatState, isAlly bool) *CombatantAuto {
	if isAlly {
		if len(state.Enemies) > 0 {
			return state.Enemies[0]
		}
		return nil
	}
	if len(state.Allies) > 0 {
		return state.Allies[0]
	}
	return nil
}

// seizeInitiative scans for a combatant that, on turn 1, may act before everyone
// else via ON_BATTLE_START/SEIZE_INITIATIVE when the opposing primary's lifespan
// does not exceed lifespan_ratio × its own. Returns the winner (or nil).
func seizeInitiative(state *CombatState, allActors []*CombatantAuto) *CombatantAuto {
	for _, actor := range allActors {
		if !hasMechanic(actor.Mechanics, "ON_BATTLE_START", "SEIZE_INITIATIVE") {
			continue
		}
		isAlly := false
		for _, a := range state.Allies {
			if a.ID == actor.ID {
				isAlly = true
				break
			}
		}
		opp := opponentPrimary(state, isAlly)
		selfLife := combatantLifespan(actor)
		if opp == nil || selfLife <= 0 {
			continue
		}
		ratio := mechanicFloat(actor.Mechanics, "ON_BATTLE_START", "SEIZE_INITIATIVE", "lifespan_ratio", 3)
		if combatantLifespan(opp) <= ratio*selfLife {
			return actor
		}
	}
	return nil
}

// sealGuTurns seals a combatant's Gu for n full turns. Duration is n+1
// because durations tick at the next turn start before acting.
func sealGuTurns(c *CombatantAuto, turns int) {
	if c == nil {
		return
	}
	if turns < 1 {
		turns = 1
	}
	c.ActiveEffects = append(c.ActiveEffects, Effect{
		ID: statusfx.StatusGuSeal, Type: "debuff", Value: 1, Duration: turns + 1,
	})
	ComputeEffectMultipliers(c)
}

// applyRebornEmpower stacks ATK_UP + HASTE buffs and an immediate shield on
// a revived combatant. One stack per successful resurrection (useCount for
// the log). Zero params = no-op so legacy REBORN data is unaffected.
func applyRebornEmpower(c *CombatantAuto, mechanics []CombatMechanic, turn, useCount int, logs *[]CombatLog) {
	atk := mechanicFloat(mechanics, "ON_DEATH", "REBORN", "empower_attack_percent", 0)
	spd := mechanicFloat(mechanics, "ON_DEATH", "REBORN", "empower_speed_percent", 0)
	sh := mechanicFloat(mechanics, "ON_DEATH", "REBORN", "empower_shield_percent", 0)
	turns := mechanicInt(mechanics, "ON_DEATH", "REBORN", "empower_turns", 100)
	if atk <= 0 && spd <= 0 && sh <= 0 {
		return
	}
	if turns < 2 {
		turns = 2
	}
	if atk > 0 {
		c.ActiveEffects = append(c.ActiveEffects, Effect{
			ID: statusfx.StatusAtkUp, Type: "buff", Value: int(atk * 100), Duration: turns,
		})
	}
	if spd > 0 {
		c.ActiveEffects = append(c.ActiveEffects, Effect{
			ID: statusfx.StatusHaste, Type: "buff", Value: int(spd * 100), Duration: turns,
		})
	}
	if sh > 0 {
		c.VirtualShield += c.MaxHP * sh
	}
	ComputeEffectMultipliers(c)
	*logs = append(*logs, CombatLog{
		Turn:      turn,
		ActorId:   c.ID,
		ActorName: c.Name,
		Action:    "REBORN_EMPOWER",
		Detail:    fmt.Sprintf("%s hồi sinh lần %d: +%d%% công, +%d%% tốc, +%.0f khiên!", c.Name, useCount, int(atk*100), int(spd*100), c.MaxHP*sh),
	})
}

// DaoMultiplier returns the dao mark damage multiplier for a combatant's paths,
// using the unified DaoCombatProfile.
func DaoMultiplier(dm DaoMarks, paths []string) (float64, bool) {
	if dm == nil || len(paths) == 0 {
		return 1.0, false
	}
	daoMarks := make(map[string]int64, len(dm))
	for k, v := range dm {
		daoMarks[k] = v
	}
	profile := gamefx.BuildDaoCombatProfile(daoMarks, nil, false, nil)
	pathKey := paths[0]
	daoAmp, realmEff, conflictMult := profile.GetOffensiveFactors(pathKey)
	mult := (1.0 + (daoAmp-1.0)*realmEff) * conflictMult
	return mult, true
}

// pathBonus returns the maximum additive bonus for any path in paths from a
// per-path bonus map (e.g. character DaoBonus / PathDamageBonus / PathResistance).
func pathBonus(bonus map[string]float64, paths []string) float64 {
	if bonus == nil {
		return 0
	}
	if len(paths) == 0 {
		paths = []string{"strength"}
	}
	maxB := 0.0
	for _, p := range paths {
		clean := gamefx.CleanPath(p)
		if b := bonus[clean]; b > maxB {
			maxB = b
		}
	}
	return maxB
}

// actorDaoMultiplier returns the dao multiplier for an action's paths including
// per-path bonuses, realm efficiency, and internal conflict from DaoProfile.
func actorDaoMultiplier(c *CombatantAuto, paths []string) (float64, bool) {
	if c == nil {
		return 1.0, false
	}
	if c.DaoProfile == nil {
		var daoMarks map[string]int64
		if c.DaoMarks != nil {
			daoMarks = make(map[string]int64, len(c.DaoMarks))
			for k, v := range c.DaoMarks {
				daoMarks[k] = v
			}
		}
		c.DaoProfile = gamefx.BuildDaoCombatProfile(daoMarks, c.PathMasteries, c.HasNonConflictingDao, c.DaoBonus)
	}
	pathKey := "strength"
	if len(paths) > 0 {
		pathKey = paths[0]
	}
	daoAmp, realmEff, conflictMult := c.DaoProfile.GetOffensiveFactors(pathKey)
	mult := (1.0 + (daoAmp-1.0)*realmEff) * conflictMult
	return mult, true
}

// resistControl returns true if the combatant's mental resistance causes a
// control effect to be resisted. Uses the provided rng for deterministic combat.
func resistControl(c *CombatantAuto, rng *rand.Rand) bool {
	if c == nil || c.MentalResistance <= 0 {
		return false
	}
	return rng.Float64() < c.MentalResistance
}

func cloneCombatState(state *CombatState) *CombatState {
	if state == nil {
		return nil
	}
	clone := &CombatState{
		Turn:          state.Turn,
		Seed:          state.Seed,
		BossPhase:     state.BossPhase,
		VirtualMode:   state.VirtualMode,
		ApertureHP:    state.ApertureHP,
		ApertureMaxHP: state.ApertureMaxHP,
		Allies:        make([]*CombatantAuto, len(state.Allies)),
		Enemies:       make([]*CombatantAuto, len(state.Enemies)),
	}
	for i, a := range state.Allies {
		clone.Allies[i] = cloneCombatant(a)
	}
	for i, e := range state.Enemies {
		clone.Enemies[i] = cloneCombatant(e)
	}
	return clone
}

func cloneCombatant(c *CombatantAuto) *CombatantAuto {
	if c == nil {
		return nil
	}
	clone := *c
	if c.Actions != nil {
		clone.Actions = make([]SimAction, len(c.Actions))
		copy(clone.Actions, c.Actions)
		for i := range clone.Actions {
			clone.Actions[i].Mechanics = cloneMechanics(c.Actions[i].Mechanics)
		}
	}
	clone.Mechanics = cloneMechanics(c.Mechanics)
	if c.ActiveEffects != nil {
		clone.ActiveEffects = make([]Effect, len(c.ActiveEffects))
		copy(clone.ActiveEffects, c.ActiveEffects)
	}
	if c.ActiveDoT != nil {
		clone.ActiveDoT = make([]DoTEffect, len(c.ActiveDoT))
		copy(clone.ActiveDoT, c.ActiveDoT)
	}
	if c.DaoMarks != nil {
		clone.DaoMarks = make(DaoMarks)
		for k, v := range c.DaoMarks {
			clone.DaoMarks[k] = v
		}
	}
	if c.CustomState != nil {
		clone.CustomState = make(map[string]int)
		for k, v := range c.CustomState {
			clone.CustomState[k] = v
		}
	} else {
		clone.CustomState = make(map[string]int)
	}
	clone.HasNonConflictingDao = c.HasNonConflictingDao
	clone.DaoProfile = c.DaoProfile
	return &clone
}

func cloneMechanics(source []CombatMechanic) []CombatMechanic {
	if source == nil {
		return nil
	}
	result := make([]CombatMechanic, len(source))
	copy(result, source)
	for i := range result {
		result[i].Params = append([]byte(nil), source[i].Params...)
		if source[i].Values != nil {
			result[i].Values = make(map[string]any, len(source[i].Values))
			for key, value := range source[i].Values {
				result[i].Values[key] = value
			}
		}
	}
	return result
}

func getFirstAliveDeterministic(alive []*CombatantAuto) *CombatantAuto {
	for _, col := range []int{1, 2, 3} {
		for _, targetY := range []int{1, 2, 3} {
			for _, e := range alive {
				if e.GridX == col && e.GridY == targetY {
					return e
				}
			}
		}
	}
	if len(alive) > 0 {
		return alive[0]
	}
	return nil
}

func getTarget(enemies []*CombatantAuto) *CombatantAuto {
	var alive []*CombatantAuto
	for _, e := range enemies {
		if e.HP > 0 {
			alive = append(alive, e)
		}
	}
	return getFirstAliveDeterministic(alive)
}

// resolveTargets returns indices of alive targets based on the action's targeting type.
// It mirrors the logic in handlers/combat_targeting.go's ResolveTargets.
func resolveTargets(enemies []*CombatantAuto, action *SimAction) []*CombatantAuto {
	targetType := strings.ToUpper(strings.TrimSpace(action.TargetType))
	if targetType == "" || targetType == "TARGET" {
		targetType = "SINGLE"
	}

	var alive []*CombatantAuto
	for _, e := range enemies {
		if e.HP > 0 {
			alive = append(alive, e)
		}
	}
	if len(alive) == 0 {
		return nil
	}

	// Stealth: single-target actions cannot select stealthed combatants unless
	// there are no other valid targets.
	if targetType == "SINGLE" {
		var visible []*CombatantAuto
		for _, e := range alive {
			if !e.IsStealthed {
				visible = append(visible, e)
			}
		}
		if len(visible) > 0 {
			alive = visible
		}
	}

	targetX := action.TargetX
	targetY := action.TargetY

	switch targetType {
	case "ROW":
		var hits []*CombatantAuto
		for _, e := range alive {
			if e.GridX == targetX {
				hits = append(hits, e)
			}
		}
		if len(hits) > 0 {
			return hits
		}
		// fallback: first alive row
		if fallback := getFirstAliveDeterministic(alive); fallback != nil {
			return []*CombatantAuto{fallback}
		}
		return []*CombatantAuto{alive[0]}

	case "COL", "COLUMN":
		var hits []*CombatantAuto
		for _, e := range alive {
			if e.GridY == targetY {
				hits = append(hits, e)
			}
		}
		if len(hits) > 0 {
			return hits
		}
		if fallback := getFirstAliveDeterministic(alive); fallback != nil {
			return []*CombatantAuto{fallback}
		}
		return []*CombatantAuto{alive[0]}

	case "CROSS":
		var hits []*CombatantAuto
		for _, e := range alive {
			dx := e.GridX - targetX
			dy := e.GridY - targetY
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}
			if (e.GridX == targetX && e.GridY == targetY) || (dx == 1 && dy == 1) {
				hits = append(hits, e)
			}
		}
		if len(hits) > 0 {
			return hits
		}
		if fallback := getFirstAliveDeterministic(alive); fallback != nil {
			return []*CombatantAuto{fallback}
		}
		return []*CombatantAuto{alive[0]}

	case "ALL", "ALL_ENEMIES":
		return alive

	default: // "SINGLE"
		var best *CombatantAuto
		if action.Bypass {
			// 1. Exact match targetX, targetY
			for _, e := range alive {
				if e.GridX == targetX && e.GridY == targetY {
					best = e
					break
				}
			}
			// 2. Match targetX, prioritize GridY 1 -> 3
			if best == nil {
				for _, yVal := range []int{1, 2, 3} {
					for _, e := range alive {
						if e.GridX == targetX && e.GridY == yVal {
							best = e
							break
						}
					}
					if best != nil {
						break
					}
				}
			}
		}
		// 3. Fallback to columns 1, 2, 3, prioritizing GridY 1 -> 3 (standard non-bypass targeting)
		if best == nil {
			for _, col := range []int{1, 2, 3} {
				for _, yVal := range []int{1, 2, 3} {
					for _, e := range alive {
						if e.GridX == col && e.GridY == yVal {
							best = e
							break
						}
					}
					if best != nil {
						break
					}
				}
				if best != nil {
					break
				}
			}
		}
		if best == nil && len(alive) > 0 {
			best = getFirstAliveDeterministic(alive)
		}
		if best != nil {
			return []*CombatantAuto{best}
		}
		return nil
	}
}

// actionOutcome reports the result of executing a single SimAction inside the
// interleaved battle loop.
type actionOutcome struct {
	executed    bool
	hasDamage   bool
	abort       bool
	firstTarget *CombatantAuto
}

// Simulate runs the deterministic auto-battle using pre-computed actions.
func Simulate(state *CombatState) []CombatLog {
	for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
		for _, c := range combatants {
			if c.CustomState == nil {
				c.CustomState = make(map[string]int)
			}
		}
	}

	// Deterministic RNG seeded from the battle seed.
	rng := rand.New(rand.NewSource(state.Seed))

	var logs []CombatLog
	rebornUses := make(map[string]int)
	initialState := cloneCombatState(state)

	startSnap := buildSnapshot(state)
	logs = append(logs, CombatLog{
		Turn:     0,
		Action:   "BATTLE_START",
		Detail:   "Trận chiến bắt đầu.",
		Snapshot: startSnap,
	})
	if len(state.InitialLogs) > 0 {
		for _, initLog := range state.InitialLogs {
			initLog.Turn = 0
			if initLog.Snapshot == nil {
				initLog.Snapshot = startSnap
			}
			logs = append(logs, initLog)
		}
	}

	maxTurns := 100


	// Pre-sort every combatant's actions stably based on explicit Priority if set,
	// preserving the player's exact per-Gu loadout sequence:
	for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
		for _, c := range combatants {
			sort.SliceStable(c.Actions, func(i, j int) bool {
				ai := c.Actions[i]
				aj := c.Actions[j]

				pI := ai.Priority
				pJ := aj.Priority
				if pI == 0 && ai.ActionType != "BASIC_ATTACK" {
					pI = 500
				}
				if pJ == 0 && aj.ActionType != "BASIC_ATTACK" {
					pJ = 500
				}

				if pI != pJ {
					return pI < pJ
				}

				if len(c.CategoryPriorities) > 0 {
					catI := getCategoryPriorityIndex(GetSimActionCategory(ai), c.CategoryPriorities)
					catJ := getCategoryPriorityIndex(GetSimActionCategory(aj), c.CategoryPriorities)
					if catI != catJ {
						return catI < catJ
					}
				} else {
					catI := GetSimActionCategory(ai)
					catJ := GetSimActionCategory(aj)
					if catI == "shield" && catJ != "shield" {
						return true
					}
					if catJ == "shield" && catI != "shield" {
						return false
					}
				}

				return false
			})
		}
	}

	for turn := 1; turn <= maxTurns; turn++ {
		state.Turn = turn
		var turnLogs []CombatLog
		turnLogs = append(turnLogs, CombatLog{
			Turn:   turn,
			Action: "TURN_START",
			Detail: fmt.Sprintf("=== LƯỢT %d ===", turn),
		})

		// ── Tick DoTs for all combatants ──
		for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
			for _, c := range combatants {
				if c.HP <= 0 {
					continue
				}
				turnLogs = append(turnLogs, tickDoTs(c, turn)...)
			}
		}

		// ── Tick resource drain effects (ESSENCE_LEAK) ──
		for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
			for _, c := range combatants {
				if c.HP <= 0 {
					continue
				}
				turnLogs = append(turnLogs, tickResourceDrain(c, turn)...)
			}
		}

		// ── Tenacity self-dispel (after DoT damage ticked, before durations tick) ──
		for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
			for _, c := range combatants {
				if c.HP <= 0 {
					continue
				}
				turnLogs = append(turnLogs, TryTenacityDispel(c, turn, rng)...)
			}
		}

		// ── Tick status effect durations and compute multipliers ──
		for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
			for _, c := range combatants {
				if c.HP <= 0 {
					continue
				}
				turnLogs = append(turnLogs, tickStatusEffects(c, turn)...)
				ComputeEffectMultipliers(c)

				// Catalog-driven Poem Intent mechanic - start-of-turn logic.
				if hasMechanic(c.Mechanics, "ON_TURN_START", "POEM_INTENT") {
					c.CustomState["thi_y"] += 1
					c.ActiveEffects = append(c.ActiveEffects, Effect{
						ID:       "thi_ho",
						Type:     "damage_reduction",
						Value:    40, // 40% reduction
						Duration: 2,  // 2 so it lasts until the end of the turn
					})
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   c.ID,
						ActorName: c.Name,
						Action:    "BUFF",
						Detail:    fmt.Sprintf("[Thơ Cổ] Tích tụ 1 tầng Thi Ý (hiện có %d tầng), tạo Thi Hộ giảm 40%% sát thương nhận vào trong lượt!", c.CustomState["thi_y"]),
					})

					if c.CustomState["thi_y"] >= 10 {
						// Immediately destroy all enemies' shields if reaching 10 at start of turn
						c.CustomState["thi_y"] = 0
						var enemyTargets []*CombatantAuto
						isAlly := false
						for _, a := range state.Allies {
							if a.ID == c.ID {
								isAlly = true
								break
							}
						}
						if isAlly {
							enemyTargets = state.Enemies
						} else {
							enemyTargets = state.Allies
						}

						for _, e := range enemyTargets {
							if e.VirtualShield > 0 {
								e.VirtualShield = 0
							}
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   c.ID,
							ActorName: c.Name,
							Action:    "SHIELD_BREAK",
							Detail:    fmt.Sprintf("[Thơ Cổ] Thi Ý viên mãn (10 tầng)! Phá vỡ hoàn toàn khiên ảo của toàn bộ kẻ địch và reset Thi Ý!"),
						})
					}
				}
			}
		}

		// ── Virtual mode: restore mana for all combatants each turn ──
		if state.VirtualMode {
			for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
				for _, c := range combatants {
					if c.HP <= 0 {
						continue
					}
					c.Mana = c.MaxMana
				}
			}
		}

		// ── Process actors ──
		var allActors []*CombatantAuto
		allActors = append(allActors, state.Allies...)
		allActors = append(allActors, state.Enemies...)

		sort.SliceStable(allActors, func(i, j int) bool {
			return allActors[i].EffectiveSpeed > allActors[j].EffectiveSpeed
		})

		// ── Thỉnh Đạo Hữu Tiên Hành Cổ: seize the opening turn ──
		// Once, on turn 1, the seizer moves ahead of everyone else (enemies act
		// after it, in normal speed order). Not a lockout.
		if turn == 1 {
			if seizer := seizeInitiative(state, allActors); seizer != nil {
				reordered := make([]*CombatantAuto, 0, len(allActors))
				reordered = append(reordered, seizer)
				for _, a := range allActors {
					if a != seizer {
						reordered = append(reordered, a)
					}
				}
				allActors = reordered
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   seizer.ID,
					ActorName: seizer.Name,
					Action:    "SEIZE_INITIATIVE",
					Detail:    fmt.Sprintf("[Thỉnh Đạo Hữu Tiên Hành] Thọ nguyên đối thủ chưa gấp ba lần đạo hữu — %s nắm quyền tiên thủ!", seizer.Name),
				})
			}
		}

		// Pre-compute confuse results once per turn to avoid RNG drift.
		confusedResult := make(map[*CombatantAuto]bool)
		for _, a := range allActors {
			if a.IsConfused {
				confusedResult[a] = rng.Float64() < 0.40
			}
		}

		// transitions for the opposing side after each executed action.
		postActionChecks := func(opposing []*CombatantAuto) {
			for _, tgt := range opposing {
				if tgt.HP <= 0 && !tgt.IsDead {
					tgt.HP = 0
					tgt.IsDead = true
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   tgt.ID,
						ActorName: tgt.Name,
						Action:    "DIED",
						Detail:    fmt.Sprintf("%s đã gục ngã!", tgt.Name),
					})
				}
			}
			for _, tgt := range opposing {
				if tgt.Kind == "Boss" {
					turnLogs = append(turnLogs, CheckBossPhaseTransitions(state, tgt, turn)...)
				}
			}
		}

		var runOne func(actor *CombatantAuto, a *SimAction, isAlly bool, enemyTargets []*CombatantAuto) actionOutcome
		runOne = func(actor *CombatantAuto, a *SimAction, isAlly bool, enemyTargets []*CombatantAuto) actionOutcome {
			res := actionOutcome{}

			if (actor.IsSilenced || actor.IsFeared) && a.ActionType != "BASIC_ATTACK" {
				return res
			}

			if actor.HasGuSeal && a.ActionType == "GU_SKILL" {
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "GU_SEALED",
					Detail:    fmt.Sprintf("%s bị phong ấn Cổ, không thể dùng [%s] lượt này!", actor.Name, a.Name),
				})
				return res
			}

			// ── BEFORE_ENEMY_ATTACK: FORGET_INTENT interrupt ──
			// The attack itself still lands; the attacker's Gu is sealed for
			// the following turns. Independent roll per defending owner.
			if a.ActionType == "BASIC_ATTACK" || a.Damage > 0 {
				for _, d := range enemyTargets {
					if d == nil || d.HP <= 0 || d.IsDead {
						continue
					}
					if !hasMechanic(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT") {
						continue
					}
					chance := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "chance", 0)
					if rng.Float64() >= chance {
						continue
					}
					if d.CustomState == nil {
						d.CustomState = make(map[string]int)
					}
					maxSucc := mechanicInt(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "max_successful_activations", 8)
					if d.CustomState["forget_intent_success"] >= maxSucc {
						if mechanicBool(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "backfire_after_max", true) {
							sealGuTurns(d, 1)
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   d.ID,
								ActorName: d.Name,
								Action:    "FORGET_INTENT_BACKFIRE",
								Detail:    fmt.Sprintf("%s đã lãng quên quá nhiều (%d lần), bị phản phệ phong ấn Cổ!", d.Name, maxSucc),
							})
						}
						continue
					}
					d.CustomState["forget_intent_success"]++
					disableTurns := mechanicInt(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "disable_target_gu_turns", 1)
					sealGuTurns(actor, disableTurns)
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "FORGET_INTENT",
						Detail:    fmt.Sprintf("%s quên mất ý niệm sau đòn này, [%s] bị phong ấn %d lượt! (%d/%d)", actor.Name, a.Name, disableTurns, d.CustomState["forget_intent_success"], maxSucc),
					})
				}
			}

			if a.ActionType == "GU_SKILL" && actor.Kind == "Player" {
				guKey := a.InstanceID
				if guKey == "" {
					guKey = a.SourceID
				}
				if actor.GuHunger != nil {
					h, exists := actor.GuHunger[guKey]
					if !exists && a.SourceID != "" && guKey != a.SourceID {
						h, exists = actor.GuHunger[a.SourceID]
						if exists {
							guKey = a.SourceID
						}
					}
					if exists && h <= 0 {
						return res
					}
					if exists {
						actor.GuHunger[guKey]--
					}
				}
				if actor.GuUsedCounts == nil {
					actor.GuUsedCounts = make(map[string]int)
				}
				if guKey != "" {
					actor.GuUsedCounts[guKey]++
				}
			}

			guRank := 1
			if a.Rank > 0 {
				guRank = a.Rank
			}
			backlashDamage := 0.0
			if a.BacklashDamage > 0 {
				backlashDamage = a.BacklashDamage
			} else if a.ActionType == "GU_SKILL" {
				backlashDamage = gamefx.HigherRankHPCost(guRank, actor.Rank, actor.MaxHP)
			}

			if backlashDamage > 0 {
				actor.HP -= backlashDamage
				if actor.HP < 0 {
					actor.HP = 0
				}
				guName := a.Name
				if guName == "" {
					guName = "Cổ trùng"
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "GU_BACKLASH",
					Damage:    backlashDamage,
					Detail:    fmt.Sprintf("%s chịu %d sát thương phản phệ do dùng %s (Rank %d > Rank %d)!", actor.Name, int(backlashDamage), guName, guRank, actor.Rank),
				})
				if actor.HP == 0 {
					actor.IsDead = true
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "DIED",
						Detail:    fmt.Sprintf("%s đã gục ngã do phản phệ Cổ!", actor.Name),
					})
					res.abort = true
					return res
				}
			}

			actionStartIndex := len(turnLogs)

			if actor.Kind == "Player" {
				if a.ManaCost > 0 && actor.Mana < a.ManaCost {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "INSUFFICIENT_MANA",
						Detail:    fmt.Sprintf("%s không đủ Chân Nguyên (cần %.0f, còn %.0f) để kích hoạt [%s]!", actor.Name, a.ManaCost, actor.Mana, a.Name),
					})
					return res
				}
				if a.ManaCost > 0 {
					actor.Mana -= a.ManaCost
				}
			}

			if (a.ActionType == "GU_SKILL" || a.ActionType == "BASIC_ATTACK" || a.ActionType == "KILLER_MOVE") && actor.Kind == "Player" && !state.VirtualMode {
				cleanPath := "strength"
				guRank := 1
				baseGain := 1
				if a.ActionType == "GU_SKILL" {
					guRank = a.Rank
					if guRank < 1 {
						guRank = 1
					}
					if len(a.Path) > 0 {
						cleanPath = gamefx.CleanPath(a.Path[0])
					}
					if guRank >= 6 {
						baseGain = 10
					}
				} else if a.ActionType == "KILLER_MOVE" {
					guRank = a.Rank
					if guRank < 1 {
						guRank = 1
					}
					if len(a.Path) > 0 {
						cleanPath = gamefx.CleanPath(a.Path[0])
					}
					baseGain = 2
					if guRank >= 6 {
						baseGain = 10
					}
				}

				isImmortal := guRank >= 6
				remaining := actor.PathMasteryGainLimit - actor.PathMasteryGained
				if isImmortal || remaining > 0 {
					if actor.PathMasteries == nil {
						actor.PathMasteries = make(map[string]int)
					}
					pts := actor.PathMasteries[cleanPath]
					currentLevel := gamefx.PathMasteryLevel(pts)
					pathGain := gamefx.CalculatePathMasteryGain(guRank, currentLevel, baseGain)

					if pathGain > 0 {
						clamped := false
						if !isImmortal {
							if pathGain > remaining {
								pathGain = remaining
								clamped = true
							}
							actor.PathMasteryGained += pathGain
							if actor.PathMasteryGained >= actor.PathMasteryGainLimit {
								clamped = true
							}
						}
						actor.PathMasteries[cleanPath] += pathGain

						newPts := actor.PathMasteries[cleanPath]
						newLvl := gamefx.PathMasteryLevel(newPts)
						lvlName := gamefx.PathMasteryLevelName(newLvl)

						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "PATH_MASTERY_GAIN",
							Detail:    fmt.Sprintf("-> Ngộ Đạo tăng tiến (%s)! +%d điểm Chân Ý %s (Cảnh giới: %s).", a.Name, pathGain, strings.Title(cleanPath), lvlName),
						})
						if clamped {
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "PATH_MASTERY_CAP",
								Detail:    fmt.Sprintf("-> %s đã đạt giới hạn %d điểm Chân Ý trong trận này, không thể ngộ đạo thêm.", actor.Name, actor.PathMasteryGainLimit),
							})
						}
					}
				}
			}

			resolved := resolveTargets(enemyTargets, a)
			// ── BEFORE_ENEMY_ATTACK: FORGET_WILL confuse ──
			// Redirects (or fizzles) the just-resolved targets. First
			// defending owner that procs decides; backfire seals the owner.
			if len(resolved) > 0 && (a.ActionType == "BASIC_ATTACK" || a.Damage > 0) {
				for _, d := range enemyTargets {
					if d == nil || d.HP <= 0 || d.IsDead {
						continue
					}
					if !hasMechanic(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE") {
						continue
					}
					if rng.Float64() >= mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "activation_chance", 0.35) {
						continue
					}
					if bf := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "backfire_chance", 0.2); rng.Float64() < bf {
						sealGuTurns(d, mechanicInt(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "backfire_disable_turns", 2))
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   d.ID,
							ActorName: d.Name,
							Action:    "FORGET_WILL_BACKFIRE",
							Detail:    fmt.Sprintf("%s bị phản phệ khi bóp méo ý chí, Cổ bị phong ấn!", d.Name),
						})
					}
					r := rng.Float64()
					hu := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "hit_user_chance", 0.15)
					hs := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "hit_self_chance", 0.35)
					switch {
					case r < hu:
						resolved = []*CombatantAuto{d}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "FORGET_WILL_REDIRECT",
							Detail:    fmt.Sprintf("%s quên mất mục tiêu, đòn [%s] lạc sang %s!", actor.Name, a.Name, d.Name),
						})
					case r < hu+hs:
						resolved = []*CombatantAuto{actor}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "FORGET_WILL_SELFHIT",
							Detail:    fmt.Sprintf("%s quên mất mục tiêu, tự đánh bản thân bằng [%s]!", actor.Name, a.Name),
						})
					default:
						resolved = nil
						res.executed = true
						res.hasDamage = false
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "FORGET_WILL_MISS",
							Detail:    fmt.Sprintf("%s quên mất mục tiêu, đòn [%s] đánh hụt hoàn toàn!", actor.Name, a.Name),
						})
						return res
					}
					break
				}
			}
			if len(resolved) == 0 {
				return res
			}

			res.executed = true
			res.hasDamage = a.ActionType == "BASIC_ATTACK" || a.Damage > 0

			firstTarget := resolved[0]

				// Dao mark multiplier for this action's path (applies to damage, shield, heal)
				daoMult, hasDao := actorDaoMultiplier(actor, a.Path)

				// Buffer shield/heal/mana/cleanse logs — emit after action name
				var shieldLog, healLog, manaLog, cleanseLog *CombatLog

				// Apply shield to self
				if a.Defense > 0 {
					baseShield := a.Defense
					shieldMult := a.ShieldMult
					if shieldMult <= 0 {
						shieldMult = 1.0
					}
					pendingShield := getPendingShieldMult(actor)
					if pendingShield > 1.0 {
						shieldMult *= pendingShield
						actor.PendingShieldBuffs = nil // consume all pending shield buffs
					}
					baseShield = a.Defense * shieldMult
					daoShield := 0.0
					if hasDao && daoMult > 1.0 {
						daoShield = baseShield * (daoMult - 1.0)
					}
					actor.VirtualShield += baseShield + daoShield
					sl := CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SHIELD_GAIN",
						Damage:    baseShield + daoShield,
						Detail:    fmt.Sprintf("%s nhận %.0f Khiên Ảo!", actor.Name, baseShield),
					}
					if daoShield > 0 {
						sl.Detail = fmt.Sprintf("%s nhận %.0f + %.0f Khiên Ảo do Đạo Ngân!", actor.Name, baseShield, daoShield)
					}
					shieldLog = &sl
				}

				// Apply healing to self
				if a.Healing > 0 {
					baseHeal := a.Healing
					daoHeal := 0.0
					if hasDao && daoMult > 1.0 {
						daoHeal = baseHeal * (daoMult - 1.0)
					}
					heal := (baseHeal + daoHeal) * (1.0 + actor.HealingReceived)
					actor.HP += heal
					if actor.HP > actor.MaxHP {
						actor.HP = actor.MaxHP
					}
					hl := CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "HEAL",
						Damage:    heal,
						Detail:    fmt.Sprintf("%s hồi phục %.0f HP!", actor.Name, heal),
						Breakdown: &DamageBreakdown{
							HealingBase:     baseHeal,
							HealingDaoBonus: daoHeal,
							HealingFinal:    heal,
							HealerHealing:   actor.StatProvenance["healing_received"],
						},
					}
					if daoHeal > 0 {
						hl.Detail = fmt.Sprintf("%s hồi phục %.0f HP (gốc %.0f + Đạo Ngân %.0f)!", actor.Name, heal, baseHeal, daoHeal)
					}
					healLog = &hl
				}

				// Apply mana restore to self
				if a.ManaRestore > 0 {
					needed := actor.MaxMana - actor.Mana
					restore := math.Min(needed, a.ManaRestore)
					if restore > 0 {
						actor.Mana += restore
						ml := CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "MANA_RESTORE",
							Damage:    restore,
							Detail:    fmt.Sprintf("%s khôi phục %.0f %s!", actor.Name, restore, actor.GetManaName()),
						}
						manaLog = &ml
					}
				}

				// Apply cleanse to self
				if a.CleanseDebuffs {
					var keptEffects []Effect
					cleansedCount := 0
					for _, eff := range actor.ActiveEffects {
						if eff.Type == "debuff" {
							cleansedCount++
						} else {
							keptEffects = append(keptEffects, eff)
						}
					}
					actor.ActiveEffects = keptEffects
					if len(actor.ActiveDoT) > 0 {
						cleansedCount += len(actor.ActiveDoT)
						actor.ActiveDoT = nil
					}
					if cleansedCount > 0 {
						ComputeEffectMultipliers(actor)
						cl := CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "CLEANSE",
							Detail:    fmt.Sprintf("%s tịnh hóa bản thân, xóa bỏ %d hiệu ứng bất lợi!", actor.Name, cleansedCount),
						}
						cleanseLog = &cl
					}
				}

				// ── Special Actions preprocessing ──
				if hasMechanic(a.Mechanics, "ON_CAST", "SUMMON_WOLF_FRONTLINE") {
					minionHP := actor.MaxHP * 0.50
					minion := &CombatantAuto{
						ID:      actor.ID + "_dialang",
						Name:    "Địa Lang Vương (Triệu hồi)",
						Kind:    "Minion",
						GridX:   1, // Frontline
						GridY:   2, // Mid
						MaxHP:   minionHP,
						HP:      minionHP,
						BaseDmg: actor.BaseDmg / 3.0,
						BaseDef: actor.BaseDef / 2.0,
					}
					if isAlly {
						state.Allies = append(state.Allies, minion)
					} else {
						state.Enemies = append(state.Enemies, minion)
					}
					actor.GridX = 2 // Move actor to Mid
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SUMMONED",
						Detail:    fmt.Sprintf("%s triệu hồi Địa Lang Vương (%.0f HP) lên tiền tuyến để đỡ đòn!", actor.Name, minionHP),
					})
				}

				if hasMechanic(a.Mechanics, "ON_CAST", "FULL_SOUL_HEAL") {
					actor.HP = actor.MaxHP
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "HEAL",
						Detail:    fmt.Sprintf("%s sử dụng Cổ Gan Dạ, hồi phục hoàn toàn HP linh hồn nhưng mất lượt hành động!", actor.Name),
					})
					res.abort = true // Break action loop (Turn_Cost = 1)
					return res
				}

				if hasMechanic(a.Mechanics, "ON_CAST", "SWORD_DRAGON_TRANSFORM") {
					shieldGain := actor.HP - 1.0
					if shieldGain > 0 {
						actor.HP = 1.0
						actor.VirtualShield += shieldGain
						actor.ActiveEffects = append(actor.ActiveEffects, Effect{
							ID:       "sword_dragon_form",
							Type:     "buff",
							Value:    50,
							Duration: 3,
						})
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "TRANSFORM",
							Detail:    fmt.Sprintf("%s hóa thân thành Thượng Cổ Kiếm Long! Chuyển %.0f HP thành Khiên Ảo, tăng 50%% sát thương trong 3 lượt!", actor.Name, shieldGain),
						})
					}
				}

				if hasMechanic(a.Mechanics, "ON_CAST", "BORROW_PAST_POWER") {
					actor.ActiveEffects = append(actor.ActiveEffects, Effect{
						ID:       "borrow_past_power",
						Type:     "buff",
						Value:    40,
						Duration: 3,
					})
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "BUFF",
						Detail:    fmt.Sprintf("%s mượn sức mạnh đỉnh phong từ quá khứ (+40%% công kích trong 3 lượt)!", actor.Name),
					})
				}

				if hasMechanic(a.Mechanics, "ON_CAST", "UNRAVEL_RIDDLE") {
					dispelledCount := 0
					var keptEffects []Effect
					for _, eff := range firstTarget.ActiveEffects {
						if eff.Type == "buff" && dispelledCount < 2 {
							dispelledCount++
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "DISPEL",
								TargetId:  firstTarget.ID,
								Detail:    fmt.Sprintf("%s hóa giải hiệu ứng lợi ích [%s] của %s!", actor.Name, eff.ID, firstTarget.Name),
							})
						} else {
							keptEffects = append(keptEffects, eff)
						}
					}
					firstTarget.ActiveEffects = keptEffects

					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "INTERRUPT",
						TargetId:  firstTarget.ID,
						Detail:    fmt.Sprintf("%s sử dụng Giải Mê Cổ ngắt đòn tụ lực sát chiêu của %s!", actor.Name, firstTarget.Name),
					})
				}

				if hasMechanic(a.Mechanics, "ON_CAST", "CHAOS_GAMBLE") {
					effectsChoices := []string{"crit_500", "invulnerable_1_turn", "reduce_enemy_hp_90"}
					costChoices := []string{"aptitude_10", "lifespan_5", "luck_50"}

					chosenEffect := effectsChoices[rng.Intn(len(effectsChoices))]
					chosenCost := costChoices[rng.Intn(len(costChoices))]

					effectMsg := ""
					if chosenEffect == "crit_500" {
						effectMsg = "Tấn công chí mạng tăng 500%!"
					} else if chosenEffect == "invulnerable_1_turn" {
						effectMsg = "Miễn nhiễm mọi sát thương trong 1 lượt (Bất tử)!"
						actor.ActiveEffects = append(actor.ActiveEffects, Effect{
							ID:       "invulnerable",
							Type:     "buff",
							Value:    1,
							Duration: 1,
						})
					} else if chosenEffect == "reduce_enemy_hp_90" {
						effectMsg = "Giảm trực tiếp 90% lượng HP hiện tại của kẻ địch!"
						damageToEnemy := firstTarget.HP * 0.90
						firstTarget.HP -= damageToEnemy
						if firstTarget.HP < 1.0 {
							firstTarget.HP = 1.0
						}
					}

					costMsg := ""
					if chosenCost == "aptitude_10" {
						costMsg = "Trừ 10% Tư chất gốc vĩnh viễn!"
					} else if chosenCost == "lifespan_5" {
						costMsg = "Trừ 5 năm tuổi thọ vĩnh viễn!"
					} else if chosenCost == "luck_50" {
						costMsg = "Trừ 50% Nguyên Thạch hiện tại!"
					}

					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "CHAOS_GAMBLE",
						Detail:    fmt.Sprintf("[%s] kích hoạt Thần Thông Cổ Nhiệm Ý! %s Cái giá phải trả: %s", a.Name, effectMsg, costMsg),
					})

					if chosenEffect == "crit_500" {
						a.Damage *= 5.0
					}
				}

				// ── Apply damage & compound effects to all resolved targets ──
				totalDamageDealt := 0.0
				hasDamage := a.ActionType == "BASIC_ATTACK" || a.Damage > 0 ||
					a.DamageHPPercent > 0 || a.ShieldStripPercent > 0
				res.hasDamage = hasDamage

				if hasDamage {
					dmgMultiplier := 1.0
					for _, eff := range actor.ActiveEffects {
						if eff.ID == "sword_dragon_form" && eff.Duration >= 0 {
							dmgMultiplier *= 1.50
						}
						if eff.ID == "borrow_past_power" && eff.Duration >= 0 {
							dmgMultiplier *= 1.40
						}
					}

					// Apply weak debuff (reduces damage dealt) and atk_up buff
					dmgMultiplier *= actor.DamageMultiplier

					// COMBAT_PASSIVE is the canonical trigger; bare PASSIVE is
					// accepted for rows authored before the rename (same
					// ON_EVENT timing, unrelated to the passive section).
					if hasMechanic(actor.Mechanics, "PASSIVE", "DOUBLE_DAMAGE_DEALT") ||
						hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "DOUBLE_DAMAGE_DEALT") {
						dmgMultiplier *= 2.0
					}

					pendingDamage := getPendingDamageMult(actor)
					if pendingDamage > 1.0 {
						dmgMultiplier *= pendingDamage
						actor.PendingDamageBuffs = nil // consume all pending damage buffs
					}

					shieldBypass := hasMechanic(a.Mechanics, "ON_HIT", "BYPASS_SHIELD") || (hasMechanic(actor.Mechanics, "ON_BATTLE_START", "SAT_KHI_GUARANTEE") && actor.CustomState != nil && actor.CustomState["sat_khi_level"] > 10)
					hitCount := a.MultiHit
					if hitCount < 1 {
						hitCount = 1
					}

					// Targets whose strike was negated by FIRST_TURN_EVADE carry
					// no percent riders (damage_hp_percent/shield strip below).
					evadedThisStrike := make(map[*CombatantAuto]bool)
					for _, target := range resolved {
						totalDmg := 0.0
						hadEvaded := target.CustomState != nil && target.CustomState["first_turn_evade_used"] == turn

						// Evasion: dodge chance from EVASION buffs, reduced by attacker Accuracy
						satKhiGuaranteed := hasMechanic(actor.Mechanics, "ON_BATTLE_START", "SAT_KHI_GUARANTEE") && actor.CustomState != nil && actor.CustomState["sat_khi_level"] > 10
						if missCh := target.EvasionChance - actor.Accuracy; missCh > 0 && rng.Float64() < missCh && !satKhiGuaranteed {
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "EVADE_MISS",
								TargetId:  target.ID,
								Detail:    fmt.Sprintf("%s né tránh đòn [%s] của %s!", target.Name, a.Name, actor.Name),
								Breakdown: &DamageBreakdown{
									BaseDamage:     a.Damage,
									AttackerAttack: actor.StatProvenance["attack"],
									Reason:         "Né tránh thành công (Evasion)",
								},
							})
							continue
						}

						// Blind: chance to miss the target entirely
						if actor.IsBlinded && actor.AccuracyPenalty > 0 {
							if rng.Float64() < actor.AccuracyPenalty {
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "BLIND_MISS",
									TargetId:  target.ID,
									Detail:    fmt.Sprintf("%s bị mù, đòn tấn công [%s] đánh trượt %s!", actor.Name, a.Name, target.Name),
						Breakdown: &DamageBreakdown{
							BaseDamage:     a.Damage,
							AttackerAttack: actor.StatProvenance["attack"],
							Reason:         "Đánh trượt do trạng thái Mù (Blind)",
						},
								})
								continue
							}
						}

					targetDef := target.BaseDef * target.DefenseMultiplier
					// ON_HIT/BYPASS_SHIELD means one thing only: the hit ignores
					// VirtualShield (resolved in applyDamageToTarget). Defense
					// shredding belongs to ignore_defense_* fields, not here.
					// Apply ignore_defense from attacker
						if actor.IgnoreDefenseFlat > 0 {
							targetDef = math.Max(0, targetDef-actor.IgnoreDefenseFlat)
						}
						if actor.IgnoreDefensePct > 0 {
							targetDef = math.Max(0, targetDef*(1.0-actor.IgnoreDefensePct))
						}

						initialShield := target.VirtualShield
						initialHP := target.HP
						isCritHit := false

						skillMult := a.DamageMult
						if skillMult <= 0 {
							skillMult = 1.0
						}

						totalHitDmg := 0.0
						for h := 0; h < hitCount; h++ {
							hitDmg := a.Damage * skillMult * dmgMultiplier * daoMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path))
							if h > 0 {
								hitDmg *= 0.7
							}
							// Critical hit check
							if actor.CritRateBonus > 0 && rng.Float64() < actor.CritRateBonus {
								hitDmg *= 1.5
								isCritHit = true
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "CRIT",
									TargetId:  target.ID,
									Detail:    fmt.Sprintf("%s chí mạng! Đòn %s gây 150%% sát thương!", actor.Name, a.Name),
								})
							}
							netDmg := hitDmg - targetDef
							natResVal := 0.0
							if target.DaoProfile != nil && len(a.Path) > 0 {
								natResVal = target.DaoProfile.GetDefensiveResistance(a.Path[0])
								netDmg *= (1.0 - natResVal)
							}
							netDmg *= (1.0 - pathBonus(target.PathResistance, a.Path))
							if netDmg < 1 {
								netDmg = 1
							}
							totalHitDmg += netDmg
							totalDmg += applyDamageToTarget(state, actor, target, netDmg, &turnLogs, turn, shieldBypass, !isAlly, rng)
						}
						if !hadEvaded && target.CustomState != nil && target.CustomState["first_turn_evade_used"] == turn {
							evadedThisStrike[target] = true
						}
						totalDamageDealt += totalDmg

						shieldAbsorbed := math.Max(0, initialShield-target.VirtualShield)
						hpLost := math.Max(0, initialHP-target.HP)
						critMultVal := 1.0
						if isCritHit {
							critMultVal = 1.5
						}

						rawDmg := a.Damage * skillMult * dmgMultiplier * daoMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path)) * critMultVal

						natResFinal := 0.0
						if target.DaoProfile != nil && len(a.Path) > 0 {
							natResFinal = target.DaoProfile.GetDefensiveResistance(a.Path[0])
						}

						breakdown := &DamageBreakdown{
							BaseDamage:      a.Damage,
							SkillMult:       skillMult,
							DaoMult:         daoMult,
							BuffMult:        dmgMultiplier,
							CritMult:        critMultVal,
							PathBonus:       pathBonus(actor.PathDamageBonus, a.Path),
							TargetDef:       targetDef,
							NaturalResist:   natResFinal,
							PathResist:      pathBonus(target.PathResistance, a.Path),
							DamageReduction: target.DamageTakenReduction,
							ShieldAbsorbed:  shieldAbsorbed,
							HPDamage:        hpLost,
							FinalDamage:     totalHitDmg,
							RawDamage:       rawDmg,
							HitCount:        hitCount,
							AttackerAttack:  actor.StatProvenance["attack"],
							DefenderDefense: target.StatProvenance["defense"],
						}

						if totalHitDmg > 0 {
							var damageDetail string
							if shieldAbsorbed > 0 && hpLost > 0 {
								damageDetail = fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f sát thương (phá vỡ %.0f khiên, trừ %.0f HP, còn %.0f HP)", actor.Name, a.Name, target.Name, totalHitDmg, shieldAbsorbed, hpLost, target.HP)
							} else if shieldAbsorbed > 0 && hpLost == 0 {
								damageDetail = fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f sát thương (bị Khiên Ảo cản %.0f, còn %.0f khiên, %.0f HP)", actor.Name, a.Name, target.Name, totalHitDmg, shieldAbsorbed, target.VirtualShield, target.HP)
							} else {
								damageDetail = fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f sát thương (còn %.0f HP)", actor.Name, a.Name, target.Name, totalHitDmg, target.HP)
							}

							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    a.ActionType,
								TargetId:  target.ID,
								Damage:    totalHitDmg,
								IsCrit:    isCritHit,
								Detail:    damageDetail,
								Breakdown: breakdown,
							})
						} else {
							if target.VirtualShield > 0 {
								breakdown.Reason = "Toàn bộ sát thương bị Khiên Ảo hấp thụ"
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    a.ActionType,
									TargetId:  target.ID,
									Damage:    0,
									Detail:    fmt.Sprintf("%s dùng [%s] lên %s, toàn bộ sát thương bị Khiên Ảo hấp thụ (còn %.0f khiên, %.0f HP)", actor.Name, a.Name, target.Name, target.VirtualShield, target.HP),
									Breakdown: breakdown,
								})
							} else if target.HP <= 0 {
								breakdown.Reason = "Mục tiêu đã gục ngã từ trước"
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    a.ActionType,
									TargetId:  target.ID,
									Damage:    0,
									Detail:    fmt.Sprintf("%s dùng [%s] lên %s (mục tiêu đã gục ngã từ trước)!", actor.Name, a.Name, target.Name),
									Breakdown: breakdown,
								})
							} else {
								breakdown.Reason = "Đối phương phòng thủ kiên cố hoặc miễn nhiễm toàn bộ sát thương"
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    a.ActionType,
									TargetId:  target.ID,
									Damage:    0,
									Detail:    fmt.Sprintf("%s dùng [%s] lên %s, gây 0 sát thương do đối phương phòng thủ/miễn nhiễm (còn %.0f HP)!", actor.Name, a.Name, target.Name, target.HP),
									Breakdown: breakdown,
								})
							}
						}

						if a.PhantomDmg > 0 {
							phantomDef := target.BaseDef
							phantomNet := (a.PhantomDmg * dmgMultiplier) - phantomDef
							phantomNet *= (1.0 - pathBonus(target.PathResistance, a.Path))
							phantomNet = math.Max(1, phantomNet)
							phantomDealt := applyDamageToTarget(state, actor, target, phantomNet, &turnLogs, turn, shieldBypass, !isAlly, rng)
							totalDamageDealt += phantomDealt
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "PHANTOM_ATTACK",
								TargetId:  target.ID,
								Damage:    phantomDealt,
								Detail:    fmt.Sprintf("Hư ảnh của %s tấn công thêm %.0f sát thương lên %s!", actor.Name, phantomDealt, target.Name),
							})
						}

						// ── Freeze break: if target is frozen and takes damage, 50% chance to break ──
						if target.IsFrozen && totalDmg > 0 {
							if rng.Float64() < 0.50 {
								var unfrozenKept []Effect
								for _, eff := range target.ActiveEffects {
									if eff.ID == EffectFreeze {
										turnLogs = append(turnLogs, CombatLog{
											Turn:      turn,
											ActorId:   target.ID,
											ActorName: target.Name,
											Action:    "FREEZE_BREAK",
											Detail:    fmt.Sprintf("Sức mạnh từ đòn tấn công phá vỡ băng giá! %s thoát đóng băng!", target.Name),
										})
										continue
									}
									unfrozenKept = append(unfrozenKept, eff)
								}
								target.ActiveEffects = unfrozenKept
								target.IsFrozen = false
							}
						}
					}

					if shieldLog != nil {
						turnLogs = append(turnLogs, *shieldLog)
					}
					if healLog != nil {
						turnLogs = append(turnLogs, *healLog)
					}
					if manaLog != nil {
						turnLogs = append(turnLogs, *manaLog)
					}
					if cleanseLog != nil {
						turnLogs = append(turnLogs, *cleanseLog)
					}

					// ── Multi-hit log ──
					if hitCount > 1 && len(resolved) > 0 {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "MULTI_HIT",
							TargetId:  resolved[0].ID,
							Detail:    fmt.Sprintf("%s trúng %d đòn liên hoàn!", resolved[0].Name, hitCount),
						})
					}

					if a.ThornsDmg > 0 && totalDamageDealt > 0 {
						thornsNet := math.Max(1, a.ThornsDmg-float64(actor.BaseDef))
						applyDamageToTarget(state, firstTarget, actor, thornsNet, &turnLogs, turn, false, false, rng)
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "THORNS",
							TargetId:  firstTarget.ID,
							Damage:    thornsNet,
							Detail:    fmt.Sprintf("Phản thương từ %s gây %.0f sát thương lên %s!", firstTarget.Name, thornsNet, actor.Name),
						})
					}

					// ── Thorns buff reflect: defender reflects % of damage dealt ──
					for _, target := range resolved {
						if target.ThornsDamage > 0 && totalDamageDealt > 0 {
							thornsNet := ApplyThornsDamage(target, actor, float64(totalDamageDealt), turn, &turnLogs)
							_ = thornsNet
						}
					}

					if a.LifeStealPct > 0 && totalDamageDealt > 0 {
						healAmount := totalDamageDealt * a.LifeStealPct
						hpNeeded := actor.MaxHP - actor.HP
						if healAmount <= hpNeeded {
							actor.HP += healAmount
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "LIFESTEAL",
								Damage:    healAmount,
								Detail:    fmt.Sprintf("%s hút %.0f máu từ sát chiêu, hồi phục HP!", actor.Name, healAmount),
							})
						} else {
							actor.HP = actor.MaxHP
							excess := healAmount - hpNeeded
							actor.VirtualShield += excess
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "LIFESTEAL_SHIELD",
								Damage:    excess,
								Detail:    fmt.Sprintf("%s hút %.0f máu từ sát chiêu, HP đầy, chuyển %.0f thành khiên!", actor.Name, healAmount, excess),
							})
						}
					}

					actionRank := a.Rank
					if actionRank <= 0 {
						actionRank = actor.Rank
					}

					// ── Percent-based direct effects (damage_hp_percent, shield strip) ──
					// A strike negated by FIRST_TURN_EVADE carries no riders.
					if a.DamageHPPercent > 0 || a.ShieldStripPercent > 0 {
						for _, target := range resolved {
							if target == nil || target.HP <= 0 {
								continue
							}
							if evadedThisStrike[target] {
								continue
							}
							if a.ShieldStripPercent > 0 && target.VirtualShield > 0 {
								stripPct := a.ShieldStripPercent
								if target.Rank > actionRank {
									supp, _ := statusfx.TargetSuppressionFactor(actionRank, target.Rank)
									stripPct *= supp
								}
								strip := target.VirtualShield * stripPct
								target.VirtualShield -= strip
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "SHIELD_STRIP",
									TargetId:  target.ID,
									Damage:    strip,
									Detail:    fmt.Sprintf("[%s] tước %.0f khiên của %s (còn %.0f khiên)!", a.Name, strip, target.Name, target.VirtualShield),
								})
							}
							if a.DamageHPPercent > 0 {
								hpPct := a.DamageHPPercent
								if target.Rank > actionRank {
									supp, _ := statusfx.TargetSuppressionFactor(actionRank, target.Rank)
									hpPct *= supp
								}
								if hpPct > 0 {
									direct := target.MaxHP * hpPct
									target.HP -= direct
									if target.HP < 0 {
										target.HP = 0
									}
									turnLogs = append(turnLogs, CombatLog{
										Turn:      turn,
										ActorId:   actor.ID,
										ActorName: actor.Name,
										Action:    "HP_PERCENT",
										TargetId:  target.ID,
										Damage:    direct,
										Detail:    fmt.Sprintf("[%s] đánh thẳng vào bản ngã, %s mất %.0f HP trực tiếp (còn %.0f HP)!", a.Name, target.Name, direct, target.HP),
									})
								} else {
									turnLogs = append(turnLogs, CombatLog{
										Turn:      turn,
										ActorId:   actor.ID,
										ActorName: actor.Name,
										Action:    "STATUS_IMMUNE",
										TargetId:  target.ID,
										Detail:    fmt.Sprintf("%s cảnh giới vượt trội, hoàn toàn miễn nhiễm sát thương trực tiếp từ [%s]!", target.Name, a.Name),
									})
								}
							}
						}
					}
				} else {
					detailMsg := fmt.Sprintf("%s dùng [%s]", actor.Name, a.Name)
					hasBuff := false
					buffKey := a.SourceID
					if buffKey == "" {
						buffKey = a.Name
					}
					if a.DamageMult > 1.0 {
						if actor.PendingDamageBuffs == nil {
							actor.PendingDamageBuffs = make(map[string]float64)
						}
						actor.PendingDamageBuffs[buffKey] = a.DamageMult
						detailMsg = fmt.Sprintf("%s dùng [%s], tăng %.1fx sát thương cho đòn tấn công kế tiếp!", actor.Name, a.Name, a.DamageMult)
						hasBuff = true
					}
					if a.ShieldMult > 1.0 {
						if actor.PendingShieldBuffs == nil {
							actor.PendingShieldBuffs = make(map[string]float64)
						}
						actor.PendingShieldBuffs[buffKey] = a.ShieldMult
						if hasBuff {
							detailMsg += fmt.Sprintf(" (và tăng %.1fx khiên kế tiếp)", a.ShieldMult)
						} else {
							detailMsg = fmt.Sprintf("%s dùng [%s], tăng %.1fx khiên cho kỹ năng tạo khiên kế tiếp!", actor.Name, a.Name, a.ShieldMult)
						}
					}

					targetID := firstTarget.ID
					if strings.EqualFold(a.TargetType, "SELF") {
						targetID = actor.ID
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    a.ActionType,
						TargetId:  targetID,
						Detail:    detailMsg,
					})

					if shieldLog != nil {
						turnLogs = append(turnLogs, *shieldLog)
					}
					if healLog != nil {
						turnLogs = append(turnLogs, *healLog)
					}
					if manaLog != nil {
						turnLogs = append(turnLogs, *manaLog)
					}
					if cleanseLog != nil {
						turnLogs = append(turnLogs, *cleanseLog)
					}
				}

				// ── Apply status effects + self-buffs (always, no damage required) ──
				// ProcChance gates the whole bonus package (status ops, self
				// evasion, target weak, Gu seal); base damage already landed.
				procOK := a.ProcChance <= 0 || rng.Float64() < a.ProcChance
				if !procOK {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "PROC_FAIL",
						Detail:    fmt.Sprintf("[%s] không kích hoạt hiệu ứng bổ sung!", a.Name),
					})
				} else {
					allOps := collectAllStatusOps(a)
					actionRank := a.Rank
					if actionRank <= 0 {
						actionRank = actor.Rank
					}
					if len(allOps) > 0 {
						applyStatusOps(actor, resolved, allOps, actionRank, turn, rng, &turnLogs)
					}
					if a.SelfEvasionPct > 0 {
						evaPct := a.SelfEvasionPct
						if actor.Rank > actionRank {
							att := statusfx.CasterAttenuationFactor(actionRank, actor.Rank)
							evaPct *= att
						}
						turns := a.BonusTurns + 1
						if turns < 2 {
							turns = 2
						}
						actor.ActiveEffects = append(actor.ActiveEffects, Effect{
							ID: "EVASION", Type: "buff", Value: int(evaPct * 100), Duration: turns,
						})
						ComputeEffectMultipliers(actor)
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "EVASION_GAIN",
							Detail:    fmt.Sprintf("%s nhận +%.0f%% né tránh trong %d lượt!", actor.Name, evaPct*100, a.BonusTurns),
						})
					}
					if a.TargetWeakPct > 0 {
						hasFR := false
						for _, op := range allOps {
							if op.ID == statusfx.StatusForgetReality {
								hasFR = true
								break
							}
						}
						if !hasFR {
							turns := a.BonusTurns
							if turns <= 0 {
								turns = 2
							}
							applyStatusOps(actor, resolved, []statusfx.StatusOp{{ID: statusfx.StatusWeak, Value: a.TargetWeakPct * 100, Turns: turns}}, actionRank, turn, rng, &turnLogs)
						}
					}
					if a.DisableGuTurns > 0 {
						for _, target := range resolved {
							if target == nil || target.HP <= 0 {
								continue
							}
							turns := a.DisableGuTurns
							if target.Rank > actionRank {
								supp, penalty := statusfx.TargetSuppressionFactor(actionRank, target.Rank)
								if supp <= 0 {
									turnLogs = append(turnLogs, CombatLog{
										Turn:      turn,
										ActorId:   target.ID,
										ActorName: target.Name,
										Action:    "STATUS_IMMUNE",
										Detail:    fmt.Sprintf("%s cảnh giới vượt trội, hoàn toàn miễn nhiễm hiệu ứng phong ấn Cổ từ [%s]!", target.Name, a.Name),
									})
									continue
								}
								turns -= penalty
								if turns < 1 {
									turns = 1
								}
							}
							sealGuTurns(target, turns)
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "GU_SEAL",
								Detail:    fmt.Sprintf("%s bị phong ấn Cổ trong %d lượt, không thể dùng Cổ!", target.Name, turns),
							})
						}
					}
				}

				// Merge action logs into a single consolidated log entry for this Gu/Action
				if len(turnLogs) > actionStartIndex {
					logs := make([]CombatLog, len(turnLogs)-actionStartIndex)
					copy(logs, turnLogs[actionStartIndex:])

					var useLog *CombatLog
					var costLog *CombatLog
					var gainLog *CombatLog
					var resultLogs []CombatLog
					var extraBreakdown *DamageBreakdown

					for i := range logs {
						log := &logs[i]
						if strings.Contains(log.Detail, " dùng [") && useLog == nil {
							useLog = log
						} else if log.Action == "MANA_COST" {
							costLog = log
						} else if log.Action == "PATH_MASTERY_GAIN" {
							gainLog = log
						} else if log.Action == "SHIELD_ABSORB" || log.Action == "SHIELD_BREAK" || log.Action == "SHIELD_BYPASS_ATTACK" || log.Action == "PATH_MASTERY_CAP" {
							// Skip redundant damage outcome logs already detailed in useLog
							continue
						} else {
							if log.Breakdown != nil && extraBreakdown == nil {
								// Preserve formula breakdowns (heal, etc.) from
								// logs folded into the consolidated entry.
								extraBreakdown = log.Breakdown
							}
							resultLogs = append(resultLogs, *log)
						}
					}

					if useLog != nil {
						if useLog.Breakdown == nil && extraBreakdown != nil {
							useLog.Breakdown = extraBreakdown
						}
						mergedDetail := useLog.Detail

						cleanDetail := func(detail string, actorName string) string {
							prefix := actorName + " "
							if strings.HasPrefix(detail, prefix) {
								return strings.TrimPrefix(detail, prefix)
							}
							return detail
						}

						if costLog != nil {
							mergedDetail += " " + cleanDetail(costLog.Detail, actor.Name)
						}

						for _, rLog := range resultLogs {
							mergedDetail += ", " + cleanDetail(rLog.Detail, actor.Name)
						}

						if gainLog != nil {
							parts := strings.Split(gainLog.Detail, "! ")
							if len(parts) > 1 {
								mergedDetail += " (Ngộ Đạo " + strings.TrimSuffix(parts[1], ".") + ")"
							} else {
								mergedDetail += " (" + strings.TrimPrefix(strings.TrimSuffix(gainLog.Detail, "."), "-> ") + ")"
							}
						}

						useLog.Detail = mergedDetail
						turnLogs = append(turnLogs[:actionStartIndex], *useLog)
					}
				}
				return res
			}

		// ── Pre-Attack Defense Pass ──
		// All alive combatants execute pure defensive actions (shield buffs) before attacks land.
		executedInPrePass := make(map[*CombatantAuto]map[int]bool)

		for _, actor := range allActors {
			if actor.HP <= 0 || actor.IsDead {
				continue
			}

			// Skip if CC'd
			isDisabled := false
			for _, eff := range actor.ActiveEffects {
				if (strings.EqualFold(eff.ID, EffectStun) || strings.EqualFold(eff.ID, statusfx.StatusStun)) && eff.Duration >= 0 {
					isDisabled = true
					break
				}
			}
			if actor.IsStunned || actor.IsFrozen || actor.IsSleeping || actor.IsBound || actor.IsSilenced || actor.IsFeared {
				isDisabled = true
			}
			if isDisabled {
				continue
			}

			// Determine target side
			isAlly := false
			for _, a := range state.Allies {
				if a.ID == actor.ID {
					isAlly = true
					break
				}
			}
			var enemyTargets []*CombatantAuto
			if isAlly {
				enemyTargets = state.Enemies
			} else {
				enemyTargets = state.Allies
			}

			if actor.IsCharmed {
				if isAlly {
					enemyTargets = state.Allies
				} else {
					enemyTargets = state.Enemies
				}
			}

			for ai := 0; ai < len(actor.Actions); ai++ {
				a := &actor.Actions[ai]
				if (a.Defense > 0 || a.DefUpTier > 0 || GetSimActionCategory(*a) == "shield") && a.Damage == 0 {
					var primaryTgt *CombatantAuto
					if len(enemyTargets) > 0 {
						primaryTgt = enemyTargets[0]
					}
					if checkActionCondition(a, actor, primaryTgt) {
						out := runOne(actor, a, isAlly, enemyTargets)
						postActionChecks(enemyTargets)
						if out.executed {
							if executedInPrePass[actor] == nil {
								executedInPrePass[actor] = make(map[int]bool)
							}
							executedInPrePass[actor][ai] = true
						}
						if out.abort || actor.HP <= 0 || actor.IsDead {
							break
						}
					}
				}
			}
		}

		// ── Sequential Combatant Turn Loop ──
		ccCheckedThisTurn := make(map[*CombatantAuto]bool)
		ccDisabledThisTurn := make(map[*CombatantAuto]bool)

		for _, actor := range allActors {
			if actor.HP <= 0 || actor.IsDead {
				continue
			}

			// Check CC and Auto-restores once per turn on actor's turn
			if !ccCheckedThisTurn[actor] {
				ccCheckedThisTurn[actor] = true

				// Check stun
				isStunned := actor.IsStunned
				if !isStunned {
					for _, eff := range actor.ActiveEffects {
						if (strings.EqualFold(eff.ID, EffectStun) || strings.EqualFold(eff.ID, statusfx.StatusStun)) && eff.Duration >= 0 {
							isStunned = true
							break
						}
					}
				}
				if isStunned {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "STUNNED",
						Detail:    fmt.Sprintf("%s bị choáng, bỏ lượt!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				} else if actor.IsFrozen {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "FROZEN",
						Detail:    fmt.Sprintf("%s bị đóng băng, bỏ lượt!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				} else if actor.IsSleeping {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SLEEPING",
						Detail:    fmt.Sprintf("%s đang ngủ, bỏ lượt!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				} else if actor.IsBound {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "BOUND",
						Detail:    fmt.Sprintf("%s bị trói, không thể hành động!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				}

				if !ccDisabledThisTurn[actor] {
					// Auto-use healing items
					if actor.AutoHealHP && actor.HP > 0 && actor.HealCharges > 0 && actor.ItemsUsedInBattle < 3 {
						hpPct := actor.HP / actor.MaxHP * 100
						if hpPct <= actor.HealThreshold {
							baseHeal := actor.HealPotency
							healVal := baseHeal * (1.0 + actor.HealingReceived)
							actor.HP += healVal
							if actor.HP > actor.MaxHP {
								actor.HP = actor.MaxHP
							}
							actor.HealCharges--
							actor.ItemsUsedInBattle++
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "ITEM_HEAL",
								Damage:    healVal,
								Detail:    fmt.Sprintf("%s tự động dùng dược liệu hồi phục %.0f HP! (đã dùng %d/3)", actor.Name, healVal, actor.ItemsUsedInBattle),
								Breakdown: &DamageBreakdown{
									HealingBase:   baseHeal,
									HealingFinal:  healVal,
									HealerHealing: actor.StatProvenance["healing_received"],
								},
							})
						}
					}

					// Auto-restore mana using stones
					if actor.AutoRestoreMana && actor.Mana < actor.MaxMana {
						manaPct := actor.Mana / actor.MaxMana * 100
						if manaPct <= actor.ManaThreshold {
							regenMult := 1.0
							if actor.RefineManaTier > 0 {
								regenMult = 0.4 + actor.RefineManaTier*0.1
							}
							neededMana := actor.MaxMana - actor.Mana

							if actor.Rank >= 6 {
								if actor.ImmortalStones >= 1 {
									manaPerStone := (1.0 / math.Pow(100, float64(actor.Rank-6))) * regenMult
									if manaPerStone > 0 {
										stonesNeeded := math.Ceil(neededMana / manaPerStone)
										stonesToConsume := math.Min(stonesNeeded, float64(actor.ImmortalStones))
										restore := stonesToConsume * manaPerStone
										actor.ImmortalStones -= stonesToConsume
										actor.Mana += restore
										if actor.Mana > actor.MaxMana {
											actor.Mana = actor.MaxMana
										}
										turnLogs = append(turnLogs, CombatLog{
											Turn:      turn,
											ActorId:   actor.ID,
											ActorName: actor.Name,
											Action:    "MANA_RESTORE",
											Detail:    fmt.Sprintf("%s tự động nghiền nát %d Tiên Nguyên Thạch, khôi phục %.0f tiên nguyên!", actor.Name, int64(stonesToConsume), restore),
										})
									}
								}
							} else {
								if actor.PrimevalStones >= 1 {
									rankMult := math.Pow(10.0, float64(actor.Rank-1))
									subRankMult := math.Pow(2.0, float64(actor.SubRank-1))
									manaPerStone := (20.0 / (rankMult * subRankMult)) * regenMult
									if manaPerStone > 0 {
										stonesNeeded := math.Ceil(neededMana / manaPerStone)
										stonesToConsume := math.Min(stonesNeeded, float64(actor.PrimevalStones))
										restore := stonesToConsume * manaPerStone
										actor.PrimevalStones -= stonesToConsume
										actor.Mana += restore
										if actor.Mana > actor.MaxMana {
											actor.Mana = actor.MaxMana
										}
										turnLogs = append(turnLogs, CombatLog{
											Turn:      turn,
											ActorId:   actor.ID,
											ActorName: actor.Name,
											Action:    "MANA_RESTORE",
											Detail:    fmt.Sprintf("%s tự động hấp thu %d Nguyên Thạch, khôi phục %.0f %s!", actor.Name, int64(stonesToConsume), restore, actor.GetManaName()),
										})
									}
								}
							}
						}
					}
				}
			}

			if ccDisabledThisTurn[actor] {
				continue
			}

			// Determine target side
			isAlly := false
			for _, a := range state.Allies {
				if a.ID == actor.ID {
					isAlly = true
					break
				}
			}

			var enemyTargets []*CombatantAuto
			if isAlly {
				enemyTargets = state.Enemies
			} else {
				enemyTargets = state.Allies
			}

			if actor.IsCharmed {
				if isAlly {
					enemyTargets = state.Allies
				} else {
					enemyTargets = state.Enemies
				}
			}

			if actor.IsConfused && confusedResult[actor] {
				if isAlly {
					enemyTargets = state.Allies
				} else {
					enemyTargets = state.Enemies
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "CONFUSED",
					Detail:    fmt.Sprintf("%s bị mê loạn, tấn công nhầm đồng minh!", actor.Name),
				})
			}



			// Non-damaging support actions (shield, heal, buff) resolve without ending the turn pass.
			// The turn pass ends only when an attack / damage action (Damage > 0 or BASIC_ATTACK) is executed.
			executedThisTurn := make([]bool, len(actor.Actions))
			if preMap, ok := executedInPrePass[actor]; ok {
				for ai, done := range preMap {
					if done && ai < len(executedThisTurn) {
						executedThisTurn[ai] = true
					}
				}
			}
			executedDamageThisTurn := false

			for {
				executedActionInLoop := false
				for ai := 0; ai < len(actor.Actions); ai++ {
					if executedThisTurn[ai] {
						continue
					}
					a := &actor.Actions[ai]

					var primaryTgt *CombatantAuto
					if len(enemyTargets) > 0 {
						primaryTgt = enemyTargets[0]
					}
					if !checkActionCondition(a, actor, primaryTgt) {
						continue
					}

					out := runOne(actor, a, isAlly, enemyTargets)
					postActionChecks(enemyTargets)
					if out.abort {
						executedDamageThisTurn = true
						break
					}
					if out.executed {
						executedThisTurn[ai] = true
						executedActionInLoop = true
						if out.hasDamage {
							executedDamageThisTurn = true
							break // Attack / damage action ends the turn pass!
						}
					}
				}

				if executedDamageThisTurn || !executedActionInLoop {
					break
				}
			}

			// Fallback: Basic attack if no attack / damage action was executed in this turn pass.
			// Routed through runOne so BEFORE_ENEMY_ATTACK (FORGET_INTENT/WILL),
			// backlash, mana, hunger and path-mastery behave exactly like a
			// normal action. runOne applies dao/path/crit multipliers itself,
			// hence the raw BaseDmg here.
			if !executedDamageThisTurn {
				fallbackAction := &SimAction{
					ActionType: "BASIC_ATTACK",
					Name:       "Đánh thường",
					Damage:     actor.BaseDmg,
					DamageMult: 1.0,
					ManaCost:   0,
					TargetType: "SINGLE",
					Path:       []string{"strength"},
				}
				out := runOne(actor, fallbackAction, isAlly, enemyTargets)
				_ = out // fallback always counts as the turn pass action
				postActionChecks(enemyTargets)
			}
		}

		logs = append(logs, turnLogs...)

		// ── Death triggers check (REBORN: rewind or in-place) ──
		// Allies may rewind the battle or stand back up; enemies only stand
		// back up in place — an enemy rewind would discard the fight fought.
		rewound := false
		var rewoundID string
		var rewindMechs []CombatMechanic
		tryReborn := func(c *CombatantAuto, allowRewind bool) (didRewind bool) {
			if c.HP > 0 || !hasMechanic(c.Mechanics, "ON_DEATH", "REBORN") || c.IsReviveLocked {
				return false
			}
			mechs := c.Mechanics
			maxUses := mechanicInt(mechs, "ON_DEATH", "REBORN", "max_uses", 1)
			if maxUses < 1 {
				maxUses = 1
			}
			if rebornUses[c.ID] >= maxUses {
				return false
			}
			guaranteed := turn == 1 && mechanicBool(mechs, "ON_DEATH", "REBORN", "turn1_guaranteed", false)
			if !guaranteed && rng.Float64() >= mechanicFloat(mechs, "ON_DEATH", "REBORN", "chance", 0) {
				return false
			}
			rebornUses[c.ID]++
			if allowRewind && strings.ToLower(mechanicString(mechs, "ON_DEATH", "REBORN", "mode", "rewind")) != "in_place" {
				return true
			}
			hpPct := mechanicFloat(mechs, "ON_DEATH", "REBORN", "revive_hp_percent", 0.5)
			c.HP = math.Max(1, c.MaxHP*hpPct)
			c.IsDead = false
			if mechanicBool(mechs, "ON_DEATH", "REBORN", "clear_debuffs", true) {
				var kept []Effect
				for _, eff := range c.ActiveEffects {
					if !isDispellableEffect(eff.ID) {
						kept = append(kept, eff)
					}
				}
				c.ActiveEffects = kept
				c.ActiveDoT = nil
			}
			applyRebornEmpower(c, mechs, turn, rebornUses[c.ID], &logs)
			ComputeEffectMultipliers(c)
			logs = append(logs, CombatLog{
				Turn:      turn,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    "TRIGGERED",
				Trigger:   "REBORN",
				Detail:    fmt.Sprintf("%s lãng quên cái chết, vùng dậy với %.0f HP (lần %d/%d)!", c.Name, c.HP, rebornUses[c.ID], maxUses),
			})
			return false
		}
		for _, a := range state.Allies {
			if tryReborn(a, true) {
				rewound = true
				rewoundID = a.ID
				rewindMechs = a.Mechanics
				logs = append(logs, CombatLog{
					Turn:      turn,
					ActorId:   a.ID,
					ActorName: a.Name,
					Action:    "TRIGGERED",
					Trigger:   "REBORN",
					Detail:    "Xuân Thu Thiền kích hoạt! Nghịch chuyển thời không, quay về Lượt 1!",
				})
				break
			}
		}
		if !rewound {
			for _, e := range state.Enemies {
				tryReborn(e, false)
			}
		}
		if rewound {
			initialState.Turn = 0
			initialState.BossPhase = 0
			state.Allies = make([]*CombatantAuto, len(initialState.Allies))
			for i, a := range initialState.Allies {
				state.Allies[i] = cloneCombatant(a)
			}
			state.Enemies = make([]*CombatantAuto, len(initialState.Enemies))
			for i, e := range initialState.Enemies {
				state.Enemies[i] = cloneCombatant(e)
			}
			state.ApertureHP = initialState.ApertureHP
			state.ApertureMaxHP = initialState.ApertureMaxHP
			for _, na := range state.Allies {
				if na.ID == rewoundID {
					for i := 0; i < rebornUses[rewoundID]; i++ {
						applyRebornEmpower(na, rewindMechs, turn, i+1, &logs)
					}
					break
				}
			}
			turn = 0 // loops will increment it to 1
			continue
		}

		// ── Win/Loss check ──
		alliesAlive := false
		for _, a := range state.Allies {
			if a.HP > 0 {
				alliesAlive = true
				break
			}
		}
		enemiesAlive := false
		for _, e := range state.Enemies {
			if e.HP > 0 {
				enemiesAlive = true
				break
			}
		}

		if !alliesAlive || !enemiesAlive {
			winner := "Phe Địch"
			if alliesAlive {
				winner = "Phe Ta"
			}
			endSnap := buildSnapshot(state)
			logs = append(logs, CombatLog{
				Turn:     turn,
				Action:   "BATTLE_END",
				Detail:   fmt.Sprintf("Trận chiến kết thúc. %s chiến thắng!", winner),
				Snapshot: endSnap,
			})
			break
		}

		if turn == maxTurns {
			endSnap := buildSnapshot(state)
			logs = append(logs, CombatLog{
				Turn:     turn,
				Action:   "BATTLE_END",
				Detail:   "Hết thời gian! Người thách đấu đã thua.",
				Snapshot: endSnap,
			})
		}
	}

	return logs
}

// buildSnapshot creates a CombatSnapshot from the current state.
func buildSnapshot(state *CombatState) *CombatSnapshot {
	snap := &CombatSnapshot{Turn: state.Turn, ApertureHP: state.ApertureHP, ApertureMaxHP: state.ApertureMaxHP}
	for _, a := range state.Allies {
		snap.Allies = append(snap.Allies, CombatantSnapshot{
			ID: a.ID, Name: a.Name, Kind: a.Kind,
			HP: a.HP, MaxHP: a.MaxHP, Mana: a.Mana, MaxMana: a.MaxMana,
			GridX: a.GridX, GridY: a.GridY, Shield: a.VirtualShield,
			Damage: a.BaseDmg, Defense: a.BaseDef,
			Speed: a.EffectiveSpeed, IsDead: a.IsDead,
		})
	}
	for _, e := range state.Enemies {
		snap.Enemies = append(snap.Enemies, CombatantSnapshot{
			ID: e.ID, Name: e.Name, Kind: e.Kind,
			HP: e.HP, MaxHP: e.MaxHP, Mana: e.Mana, MaxMana: e.MaxMana,
			GridX: e.GridX, GridY: e.GridY, Shield: e.VirtualShield,
			Damage: e.BaseDmg, Defense: e.BaseDef,
			Speed: e.EffectiveSpeed, IsDead: e.IsDead,
		})
	}
	return snap
}

func applyDamageToTarget(state *CombatState, actor, target *CombatantAuto, damage float64, logs *[]CombatLog, turn int, shieldBypass bool, spillToAperture bool, rng *rand.Rand) float64 {
	if damage < 1 {
		damage = 1
	}

	// Định Tiên Du first turn evade: dodges the FIRST strike on turn 1 only.
	// Multi-hit/AoE follow-ups land normally, as do percent riders attached
	// to the evaded strike (skipped via evadedThisStrike in runOne).
	if turn == 1 && hasMechanic(target.Mechanics, "BEFORE_HIT", "FIRST_TURN_EVADE") {
		if target.CustomState == nil {
			target.CustomState = make(map[string]int)
		}
		if target.CustomState["first_turn_evade_used"] != turn {
			target.CustomState["first_turn_evade_used"] = turn
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "EVADE",
				Detail:    fmt.Sprintf("%s kích hoạt Định Tiên Du, né tránh đòn đánh đầu tiên hoàn toàn!", target.Name),
			})
			return 0
		}
	}

	// Invulnerable check (INVINCIBLE status or mechanic-granted "invulnerable")
	isInvulnerable := target.DamageTakenReduction >= 1.0
	if !isInvulnerable {
		for _, eff := range target.ActiveEffects {
			if (strings.EqualFold(eff.ID, "invulnerable") || strings.EqualFold(eff.ID, statusfx.StatusInvincible)) && eff.Duration >= 0 { // remains active
				isInvulnerable = true
				break
			}
		}
	}
	if isInvulnerable && !(actor.CustomState != nil && hasMechanic(actor.Mechanics, "ON_BATTLE_START", "SAT_KHI_GUARANTEE") && actor.CustomState["sat_khi_level"] > 10) {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "BLOCKED",
			Detail:    fmt.Sprintf("%s miễn nhiễm mọi sát thương nhờ trạng thái Bất Tử!", target.Name),
		})
		return 0
	}
	if isInvulnerable {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "BLOCKED_BREAK",
			Detail:    fmt.Sprintf("Sát khí Tất Sát xuyên qua Bất Tử! %s vẫn nhận sát thương!", target.Name),
		})
	}

	// ── Freeze break: taking damage has 50% chance to break freeze ──
	if target.IsFrozen && damage > 0 {
		if rng.Float64() < 0.50 {
			var unfrozenKept []Effect
			for _, eff := range target.ActiveEffects {
				if eff.ID == EffectFreeze {
					*logs = append(*logs, CombatLog{
						Turn:      turn,
						ActorId:   target.ID,
						ActorName: target.Name,
						Action:    "FREEZE_BREAK",
						Detail:    fmt.Sprintf("Sát thương phá vỡ băng giá! %s thoát đóng băng!", target.Name),
					})
					continue
				}
				unfrozenKept = append(unfrozenKept, eff)
			}
			target.ActiveEffects = unfrozenKept
			target.IsFrozen = false
		}
	}

	// ── Sleep break: taking damage wakes the target ──
	if target.IsSleeping && damage > 0 {
		var wokeKept []Effect
		for _, eff := range target.ActiveEffects {
			if eff.ID == statusfx.StatusSleep {
				*logs = append(*logs, CombatLog{
					Turn:      turn,
					ActorId:   target.ID,
					ActorName: target.Name,
					Action:    "SLEEP_BREAK",
					Detail:    fmt.Sprintf("%s bị đánh thức!", target.Name),
				})
				continue
			}
			wokeKept = append(wokeKept, eff)
		}
		target.ActiveEffects = wokeKept
		target.IsSleeping = false
	}

	// Phá Không Cổ shield bypass
	if shieldBypass {
		target.HP -= damage
		if spillToAperture {
			redirectApertureOverflow(state, target, logs, turn)
		} else if target.HP < 0 {
			target.HP = 0
		}
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SHIELD_BYPASS_ATTACK",
			TargetId:  target.ID,
			Damage:    damage,
			Detail:    fmt.Sprintf("%s sử dụng Phá Không Cổ xuyên thẳng qua Chân Nguyên Giáp, gây %d sát thương lên %s!", actor.Name, int(damage), target.Name),
		})
		return damage
	}

	// Thơ Cổ: "thi_ho" damage reduction (40%)
	for _, eff := range target.ActiveEffects {
		if eff.ID == "thi_ho" && eff.Duration >= 0 {
			reducedDmg := damage * 0.40
			damage -= reducedDmg
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "DEFEND",
				Detail:    fmt.Sprintf("[Thơ Cổ] Thi Hộ cản lại 40%% sát thương (giảm %.0f sát thương)!", reducedDmg),
			})
			break
		}
	}

	// Registry-driven damage reduction (DAMAGE_REDUCTION_UP / INVINCIBLE).
	if target.DamageTakenReduction > 0 {
		if target.DamageTakenReduction >= 1.0 {
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "BLOCKED",
				Detail:    fmt.Sprintf("%s miễn nhiễm sát thương (Bất Tử)!", target.Name),
			})
			return 0
		}
		reduced := damage * target.DamageTakenReduction
		damage -= reduced
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "DAMAGE_REDUCTION",
			Detail:    fmt.Sprintf("%s giảm %.0f%% sát thương nhận vào (cản %.0f sát thương)!", target.Name, target.DamageTakenReduction*100, reduced),
		})
	}

	hpDamageDealt := 0.0

	if target.VirtualShield > 0 {
		// Thơ Cổ: Ghi nhận khiên và xuyên khiên
		if hasMechanic(actor.Mechanics, "ON_TURN_START", "POEM_INTENT") {
			stacks := actor.CustomState["thi_y"]

			// Xuyên khiên (chỉ 1 lần mỗi lượt)
			if stacks >= 3 && !shieldBypass && actor.CustomState["thi_y_pierce_turn"] != turn {
				actor.CustomState["thi_y_pierce_turn"] = turn

				usableMultiples := stacks / 3
				ignorePct := float64(usableMultiples) * 0.30
				if ignorePct > 1.0 {
					ignorePct = 1.0
				}

				shieldIgnored := target.VirtualShield * ignorePct
				if shieldIgnored > damage {
					shieldIgnored = damage
				}

				if shieldIgnored > 0 {
					target.HP -= shieldIgnored
					if target.HP < 0 {
						target.HP = 0
					}
					damage -= shieldIgnored
					hpDamageDealt += shieldIgnored
					*logs = append(*logs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SHIELD_PIERCE",
						Detail:    fmt.Sprintf("[Thơ Cổ] Với uy năng từ %d tầng Thi Ý, đòn đánh xuyên thẳng qua %.0f%% khiên (gây %.0f sát thương thẳng vào HP)!", stacks, ignorePct*100, shieldIgnored),
					})
				}
			}

			// Thu thập thông tin khi chạm khiên (chỉ 1 lần mỗi lượt)
			if actor.CustomState["thi_y_shield_turn"] != turn {
				actor.CustomState["thi_y_shield_turn"] = turn
				actor.CustomState["thi_y"] += 1
				*logs = append(*logs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BUFF",
					Detail:    fmt.Sprintf("[Thơ Cổ] Chạm trán khiên, thu thập tin tức! Nhận thêm 1 tầng Thi Ý (hiện có %d tầng)!", actor.CustomState["thi_y"]),
				})

				if actor.CustomState["thi_y"] >= 10 {
					actor.CustomState["thi_y"] = 0
					target.VirtualShield = 0
					*logs = append(*logs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SHIELD_BREAK",
						Detail:    fmt.Sprintf("[Thơ Cổ] Thi Ý viên mãn (10 tầng)! Nhìn thấu vạn vật, phá nát toàn bộ khiên ảo của %s!", target.Name),
					})
				}
			}
		}

		if damage <= 0 {
			return hpDamageDealt
		}

		if target.VirtualShield >= damage {
			target.VirtualShield -= damage
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "SHIELD_ABSORB",
				Damage:    damage,
				Detail:    fmt.Sprintf("%s tấn công %s gây %.0f sát thương (còn %.0f khiên, %.0f HP)", actor.Name, target.Name, damage, target.VirtualShield, target.HP),
			})
			return hpDamageDealt // Trả về hpDamageDealt để caller không log double damage
		}

		absorbed := target.VirtualShield
		damage -= absorbed
		target.VirtualShield = 0
		target.HP -= damage
		if spillToAperture {
			redirectApertureOverflow(state, target, logs, turn)
		} else if target.HP < 0 {
			target.HP = 0
		}
		hpDamageDealt += damage
		totalDmg := absorbed + damage
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SHIELD_BREAK",
			Damage:    totalDmg,
			Detail:    fmt.Sprintf("%s tấn công %s gây %.0f sát thương (phá vỡ %.0f khiên, còn %.0f HP)", actor.Name, target.Name, totalDmg, absorbed, target.HP),
		})
		return hpDamageDealt // Chỉ trả về lượng sát thương trừ vào HP để caller log
	}

	// Direct HP damage (no shield)
	target.HP -= damage
	if spillToAperture {
		redirectApertureOverflow(state, target, logs, turn)
	} else if target.HP < 0 {
		target.HP = 0
	}
	return hpDamageDealt + damage
}

// redirectApertureOverflow spills damage that reduced a defending ally below 0 HP
// into the Immortal Aperture (Tiên Khiếu). Called only for tribulation combat where
// ApertureMaxHP > 0 and the attacker is an enemy.
func redirectApertureOverflow(state *CombatState, target *CombatantAuto, logs *[]CombatLog, turn int) {
	if state == nil || state.ApertureMaxHP <= 0 {
		return
	}
	if target.HP >= 0 {
		return
	}
	overflow := -target.HP
	target.HP = 0
	state.ApertureHP -= overflow
	if state.ApertureHP < 0 {
		state.ApertureHP = 0
	}
	*logs = append(*logs, CombatLog{
		Turn:      turn,
		ActorId:   target.ID,
		ActorName: target.Name,
		Action:    "APERTURE_DAMAGE",
		Damage:    overflow,
		Detail:    fmt.Sprintf("Tiên Khiếu chịu %.0f sát thương xuyên phá (còn %.0f/%.0f)!", overflow, state.ApertureHP, state.ApertureMaxHP),
	})
}

// checkActionCondition returns true if the action's use condition is met.
func checkActionCondition(action *SimAction, actor, target *CombatantAuto) bool {
	switch action.UseCondition {
	case "":
		return true
	case "hp_below":
		if actor.MaxHP <= 0 {
			return true
		}
		return actor.HP/actor.MaxHP*100 <= action.ConditionValue
	case "mana_above":
		if actor.MaxMana <= 0 {
			return true
		}
		return actor.Mana/actor.MaxMana*100 >= action.ConditionValue
	case "enemy_hp_below":
		if target == nil || target.MaxHP <= 0 {
			return true
		}
		return float64(target.HP)/float64(target.MaxHP)*100 <= action.ConditionValue
	default:
		return true
	}
}

// GetSimActionCategory returns the primary category for a SimAction: "shield", "heal", "damage", or "other".
func GetSimActionCategory(act SimAction) string {
	if act.Defense > 0 || act.ShieldMult > 1.0 || act.DefUpTier > 0 || act.ActionType == "SHIELD" {
		return "shield"
	}
	if act.Healing > 0 || act.RegenTier > 0 || act.LifeStealPct > 0 || act.ActionType == "HEAL" {
		return "heal"
	}
	if act.Damage > 0 || act.DamageMult > 1.0 || act.ActionType == "BASIC_ATTACK" {
		return "damage"
	}
	return "other"
}

func getCategoryPriorityIndex(cat string, priorities []string) int {
	if len(priorities) == 0 {
		priorities = []string{"other", "shield", "heal", "damage"}
	}
	for idx, p := range priorities {
		if p == cat {
			return idx
		}
	}
	return 999
}

func getPendingDamageMult(actor *CombatantAuto) float64 {
	if len(actor.PendingDamageBuffs) == 0 {
		return 1.0
	}
	mult := 1.0
	for _, m := range actor.PendingDamageBuffs {
		if m > 0 {
			mult *= m
		}
	}
	return mult
}

func getPendingShieldMult(actor *CombatantAuto) float64 {
	if len(actor.PendingShieldBuffs) == 0 {
		return 1.0
	}
	mult := 1.0
	for _, m := range actor.PendingShieldBuffs {
		if m > 0 {
			mult *= m
		}
	}
	return mult
}
