package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func hasGuEquipped(c *CombatantAuto, guID string) bool {
	if c == nil || c.Loadout == nil {
		return false
	}
	for _, id := range c.Loadout.GuIds {
		if id == guID {
			return true
		}
	}
	if c.Loadout.ImmortalGuId != nil && *c.Loadout.ImmortalGuId == guID {
		return true
	}
	return false
}

// DaoMultiplier returns the dao mark damage multiplier for a combatant's paths,
// mirroring the formula in handlers/combat_effects.go. Returns 1.0 and false when
// no matching dao mark is present.
func DaoMultiplier(dm DaoMarks, paths []string) (float64, bool) {
	if dm == nil || len(paths) == 0 {
		return 1.0, false
	}
	maxMult := 1.0
	found := false
	for _, path := range paths {
		cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
		amt, ok := dm[cleanPath]
		if !ok || amt <= 0 {
			continue
		}
		found = true
		var mult float64
		if amt <= 10000 {
			mult = 1.0 + float64(amt)/100.0
		} else {
			mult = 1.0 + 100.0 + float64(amt-10000)/500.0
		}
		if mult > maxMult {
			maxMult = mult
		}
	}
	if !found {
		return 1.0, false
	}
	return maxMult, true
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
		clean := strings.TrimSuffix(strings.ToLower(p), " path")
		if b := bonus[clean]; b > maxB {
			maxB = b
		}
	}
	return maxB
}

// actorDaoMultiplier returns the dao multiplier for an action's paths including
// per-path bonuses from the combatant's character effects (race/constitution).
func actorDaoMultiplier(c *CombatantAuto, paths []string) (float64, bool) {
	mult, has := DaoMultiplier(c.DaoMarks, paths)
	if bonus := pathBonus(c.DaoBonus, paths); bonus != 0 {
		mult += bonus
		has = true
	}
	return mult, has
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
	}
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
	return &clone
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
	targetType := action.TargetType
	if targetType == "" {
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

	case "COL":
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

	case "ALL":
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
	cicadaUsed := false
	initialState := cloneCombatState(state)

	startSnap := buildSnapshot(state)
	logs = append(logs, CombatLog{
		Turn:     0,
		Action:   "BATTLE_START",
		Detail:   "Trận chiến bắt đầu.",
		Snapshot: startSnap,
	})

	maxTurns := 100

	// Persistent action cursor across turns — each actor resumes from where
	// they left off, enabling alternating damage turns.
	actionCursor := make(map[*CombatantAuto]int)

	// Pre-sort every combatant's actions by priority once before the battle.
	for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
		for _, c := range combatants {
			sort.SliceStable(c.Actions, func(i, j int) bool {
				return c.Actions[i].Priority < c.Actions[j].Priority
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

		// ── Tick status effect durations and compute multipliers ──
		for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
			for _, c := range combatants {
				if c.HP <= 0 {
					continue
				}
				turnLogs = append(turnLogs, tickStatusEffects(c, turn)...)
				ComputeEffectMultipliers(c)

				// Thơ Cổ (GU_POEM_IMMORTAL) - Start of turn logic
				if hasGuEquipped(c, "GU_POEM_IMMORTAL") {
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

		// Pre-compute confuse results once per turn to avoid RNG drift.
		confusedResult := make(map[*CombatantAuto]bool)
		for _, a := range allActors {
			if a.IsConfused {
				confusedResult[a] = rng.Float64() < 0.40
			}
		}

		executedOnce := make(map[*CombatantAuto]bool)

		// postActionChecks marks newly-dead combatants and runs boss phase
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
					turnLogs = append(turnLogs, CheckBossPhaseTransitions(state, tgt)...)
				}
			}
		}

		anyActionsRemain := true
		for anyActionsRemain {
			anyActionsRemain = false
			for _, actor := range allActors {
				if actor.HP <= 0 || actor.IsDead {
					continue
				}

				// Check stun
				isStunned := false
				for _, eff := range actor.ActiveEffects {
					if eff.ID == EffectStun && eff.Duration >= 0 { // duration remains at least 0
						isStunned = true
						break
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
					continue
				}

				// Check freeze (like stun but can be broken by damage)
				if actor.IsFrozen {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "FROZEN",
						Detail:    fmt.Sprintf("%s bị đóng băng, bỏ lượt!", actor.Name),
					})
					continue
				}

				// Check sleep (like stun but breaks on damage)
				if actor.IsSleeping {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SLEEPING",
						Detail:    fmt.Sprintf("%s đang ngủ, bỏ lượt!", actor.Name),
					})
					continue
				}

				// Check bind (immobilized, loses turn)
				if actor.IsBound {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "BOUND",
						Detail:    fmt.Sprintf("%s bị trói, không thể hành động!", actor.Name),
					})
					continue
				}

				// Check silence / fear — force basic attack only
				if actor.IsSilenced || actor.IsFeared {
					// Will be handled in action loop below (skip non-basic actions)
				}

				// Auto-use healing items
				if actor.AutoHealHP && actor.HP > 0 && actor.HealCharges > 0 && actor.ItemsUsedInBattle < 3 {
					hpPct := float64(actor.HP) / float64(actor.MaxHP) * 100
					if hpPct <= actor.HealThreshold {
						healVal := int(actor.HealPotency)
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
							Detail:    fmt.Sprintf("%s tự động dùng dược liệu hồi phục %d HP! (đã dùng %d/3)", actor.Name, healVal, actor.ItemsUsedInBattle),
						})
					}
				}

// Auto-restore mana using stones (rank-scaled formula matching manual recharge)
			if actor.AutoRestoreMana && actor.Mana < actor.MaxMana {
				manaPct := float64(actor.Mana) / float64(actor.MaxMana) * 100
				if manaPct <= actor.ManaThreshold {
					regenMult := 1.0
					if actor.RefineManaTier > 0 {
						regenMult = 0.4 + actor.RefineManaTier*0.1
					}
					neededMana := float64(actor.MaxMana - actor.Mana)

					if actor.Rank >= 6 {
						// Tiên Nguyên (Rank 6+): manaGain = stones / 100^(rank-6)
						if actor.ImmortalStones >= 1 {
							manaPerStone := (1.0 / math.Pow(100, float64(actor.Rank-6))) * regenMult
							if manaPerStone > 0 {
								stonesNeeded := math.Ceil(neededMana / manaPerStone)
								stonesToConsume := math.Min(stonesNeeded, float64(actor.ImmortalStones))
								restore := int(stonesToConsume * manaPerStone)
								actor.ImmortalStones -= int64(stonesToConsume)
								actor.Mana += restore
								if actor.Mana > actor.MaxMana {
									actor.Mana = actor.MaxMana
								}
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "MANA_RESTORE",
									Detail:    fmt.Sprintf("%s tự động nghiền nát %d Tiên Nguyên Thạch, khôi phục %d tiên nguyên!", actor.Name, int64(stonesToConsume), restore),
								})
							}
						}
					} else {
						// Chân Nguyên (Rank 1-5): manaPerStone = 20 / (10^(rank-1) * 2^(subRank-1))
						if actor.PrimevalStones >= 1 {
							rankMult := math.Pow(10.0, float64(actor.Rank-1))
							subRankMult := math.Pow(2.0, float64(actor.SubRank-1))
							manaPerStone := (20.0 / (rankMult * subRankMult)) * regenMult
							if manaPerStone > 0 {
								stonesNeeded := math.Ceil(neededMana / manaPerStone)
								stonesToConsume := math.Min(stonesNeeded, float64(actor.PrimevalStones))
								restore := int(stonesToConsume * manaPerStone)
								actor.PrimevalStones -= int64(stonesToConsume)
								actor.Mana += restore
								if actor.Mana > actor.MaxMana {
									actor.Mana = actor.MaxMana
								}
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "MANA_RESTORE",
									Detail:    fmt.Sprintf("%s tự động hấp thu %d Nguyên Thạch, khôi phục %d %s!", actor.Name, int64(stonesToConsume), restore, actor.GetManaName()),
								})
							}
						}
					}
				}
			}

				// Determine target side
				var targets []*CombatantAuto
				isAlly := false
				for _, a := range state.Allies {
					if a.ID == actor.ID {
						isAlly = true
						break
					}
				}
				if isAlly {
					targets = state.Enemies
				} else {
					targets = state.Allies
				}

				target := getTarget(targets)
				if target == nil {
					continue
				}

				// ── Taunt: redirect target to taunter ──
				if !isAlly {
					for _, ally := range state.Allies {
						if ally.HasTaunt && ally.HP > 0 {
							target = ally
							break
						}
					}
				} else {
					for _, enemy := range state.Enemies {
						if enemy.HasTaunt && enemy.HP > 0 {
							target = enemy
							break
						}
					}
				}

			// ── Multi-action per turn: actions resolve one at a time; each
			// damage-dealing action yields to the next actor (alternating turns).
			// Non-damage actions (buffs, heals, shields) execute freely. ──
				if len(actor.Actions) == 0 {
					continue
				}

				var enemyTargets []*CombatantAuto
				if isAlly {
					enemyTargets = state.Enemies
				} else {
					enemyTargets = state.Allies
				}

				// ── Charm: charmed combatant attacks its own side ──
				if actor.IsCharmed {
					if isAlly {
						enemyTargets = state.Allies
					} else {
						enemyTargets = state.Enemies
					}
				}

				// ── Confuse: chance to mis-target an ally instead of an enemy ──
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

				// runOne attempts a single action for the given actor. The body is
				// identical to the previous inline action loop; skip/abort control
				// flow is expressed through the returned actionOutcome.
				runOne := func(actor *CombatantAuto, a *SimAction, isAlly bool, enemyTargets []*CombatantAuto) actionOutcome {
					res := actionOutcome{}

					// Silence / Fear: skip skills and killer moves, only allow basic attack
if (actor.IsSilenced || actor.IsFeared) && a.ActionType != "BASIC_ATTACK" {
			return res
			}

			// Path mastery handling (mirrors original logic) – awarded before mana check
			if (a.ActionType == "GU_SKILL" || a.ActionType == "BASIC_ATTACK") && actor.Kind == "Player" && !state.VirtualMode {
				remaining := actor.PathMasteryGainLimit - actor.PathMasteryGained
				if remaining > 0 {
					cleanPath := "strength"
					pathGain := 1
					if a.ActionType == "GU_SKILL" && len(a.Path) > 0 {
						cleanPath = strings.TrimSuffix(strings.ToLower(a.Path[0]), " path")
						if a.Rank >= 6 {
							pathGain = 10
						}
					}
					clamped := pathGain > remaining
					if clamped {
						pathGain = remaining
					}
					if actor.PathMasteries == nil {
						actor.PathMasteries = make(map[string]int)
					}
					actor.PathMasteryGained += pathGain
					actor.PathMasteries[cleanPath] += pathGain

					pts := actor.PathMasteries[cleanPath]
					lvlName := "Sơ Học"
					if pts >= 10000 {
						lvlName = "Vô Thượng Tông Sư"
					} else if pts >= 2500 {
						lvlName = "Đại Tông Sư"
					} else if pts >= 500 {
						lvlName = "Tông Sư"
					} else if pts >= 100 {
						lvlName = "Đại Sư"
					}

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

					if a.ManaCost > 0 && float64(actor.Mana) < a.ManaCost {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "INSUFFICIENT_MANA",
							Detail:    fmt.Sprintf("%s không đủ %s để dùng [%s] (cần %.0f, còn %d)", actor.Name, actor.GetManaName(), a.Name, a.ManaCost, actor.Mana),
						})
						return res
					}

					resolved := resolveTargets(enemyTargets, a)
					if len(resolved) == 0 {
						res.abort = true
						return res
					}
					firstTarget := resolved[0]
					res.firstTarget = firstTarget

					if !checkActionCondition(a, actor, firstTarget) {
						return res
					}

					res.executed = true
					actionStartIndex := len(turnLogs)
					manaCost := int(a.ManaCost)
					actor.Mana -= manaCost

					if manaCost > 0 {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "MANA_COST",
							Damage:    manaCost,
							Detail:    fmt.Sprintf("%s tiêu hao %d %s (còn %d)", actor.Name, manaCost, actor.GetManaName(), actor.Mana),
						})
					}

// Path mastery handling moved earlier (original block removed)

					// Backlash damage from using Gu rank > actor rank
					if a.BacklashDamage > 0 {
						backlash := int(a.BacklashDamage)
						actor.HP -= backlash
						if actor.HP < 0 {
							actor.HP = 0
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "DAMAGE",
							Damage:    backlash,
							Detail:    fmt.Sprintf("[Phản Phệ] Cổ trùng Rank cao phản phệ %s, trừ %d HP (còn %d HP)!", actor.Name, backlash, actor.HP),
						})
						if actor.HP <= 0 {
							res.abort = true
							return res
						}
					}

					// Dao mark multiplier for this action's path (applies to damage, shield, heal)
					daoMult, hasDao := actorDaoMultiplier(actor, a.Path)

					// Buffer shield/heal logs — emit after action name
					var shieldLog, healLog *CombatLog

					// Apply shield to self
					if a.Defense > 0 {
						baseShield := int(a.Defense)
						daoShield := 0
						if hasDao && daoMult > 1.0 {
							daoShield = int(float64(baseShield) * (daoMult - 1.0))
						}
						actor.VirtualShield += baseShield + daoShield
						sl := CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "SHIELD_GAIN",
							Damage:    baseShield + daoShield,
							Detail:    fmt.Sprintf("%s nhận %d Khiên Ảo!", actor.Name, baseShield),
						}
						if daoShield > 0 {
							sl.Detail = fmt.Sprintf("%s nhận %d + %d Khiên Ảo do Đạo Ngân!", actor.Name, baseShield, daoShield)
						}
						shieldLog = &sl
					}

					// Apply healing to self
					if a.Healing > 0 {
						baseHeal := int(a.Healing)
						daoHeal := 0
						if hasDao && daoMult > 1.0 {
							daoHeal = int(float64(baseHeal) * (daoMult - 1.0))
						}
						heal := int(float64(baseHeal+daoHeal) * (1.0 + actor.HealingReceived))
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
							Detail:    fmt.Sprintf("%s hồi phục %d HP!", actor.Name, baseHeal),
						}
						if daoHeal > 0 {
							hl.Detail = fmt.Sprintf("%s hồi phục %d + %d HP do Đạo Ngân!", actor.Name, baseHeal, daoHeal)
						}
						healLog = &hl
					}

					// ── Special Actions preprocessing ──
					if a.SourceID == "GU_WOLF_ENSLAVEMENT_CORE" {
						minionHP := int(float64(actor.MaxHP) * 0.50)
						minion := &CombatantAuto{
							ID:      actor.ID + "_dialang",
							Name:    "Địa Lang Vương (Triệu hồi)",
							Kind:    "Minion",
							GridX:   1, // Frontline
							GridY:   2, // Mid
							MaxHP:   minionHP,
							HP:      minionHP,
							BaseDmg: actor.BaseDmg / 3,
							BaseDef: actor.BaseDef / 2,
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
							Detail:    fmt.Sprintf("%s triệu hồi Địa Lang Vương (%d HP) lên tiền tuyến để đỡ đòn!", actor.Name, minionHP),
						})
					}

					if a.SourceID == "GU_GUTS" {
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

					if a.SourceID == "GU_SWORD_DRAGON_TRANSFORM" {
						shieldGain := actor.HP - 1
						if shieldGain > 0 {
							actor.HP = 1
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
								Detail:    fmt.Sprintf("%s hóa thân thành Thượng Cổ Kiếm Long! Chuyển %d HP thành Khiên Ảo, tăng 50%% sát thương trong 3 lượt!", actor.Name, shieldGain),
							})
						}
					}

					if a.SourceID == "GU_PRESENT_DAYS_PAST" {
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

					if a.SourceID == "GU_UNRAVEL_RIDDLE" {
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

					if a.SourceID == "GU_LOVE_GU" {
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
							damageToEnemy := float64(firstTarget.HP) * 0.90
							firstTarget.HP -= int(damageToEnemy)
							if firstTarget.HP < 1 {
								firstTarget.HP = 1
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
							Detail:    fmt.Sprintf("Cổ Tình Yêu kích hoạt! %s Cái giá phải trả: %s", effectMsg, costMsg),
						})

						if chosenEffect == "crit_500" {
							a.Damage *= 5.0
						}
					}

					// ── Apply damage & compound effects to all resolved targets ──
					totalDamageDealt := 0
					hasDamage := a.ActionType == "BASIC_ATTACK" || a.Damage > 0
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

						if hasGuEquipped(actor, "GU_SOVEREIGN_FETUS") {
							dmgMultiplier *= 2.0
						}

						shieldBypass := (a.SourceID == "GU_BREAK_SPACE")
						hitCount := a.MultiHit
						if hitCount < 1 {
							hitCount = 1
						}

						for _, target := range resolved {
							totalDmg := 0.0

							// Blind: chance to miss the target entirely
							if actor.IsBlinded && actor.AccuracyPenalty > 0 {
								if rng.Float64() < actor.AccuracyPenalty {
									turnLogs = append(turnLogs, CombatLog{
										Turn:      turn,
										ActorId:   actor.ID,
										ActorName: actor.Name,
										Action:    "BLIND_MISS",
										TargetId:  target.ID,
										Detail:    fmt.Sprintf("%s bị mù, đòn tấn công đánh trượt %s!", actor.Name, target.Name),
									})
									continue
								}
							}

							targetDef := float64(target.BaseDef) * target.DefenseMultiplier
							if a.SourceID == "GU_BREAK_SPACE" {
								targetDef = math.Max(0, targetDef-30.0)
							}

							for h := 0; h < hitCount; h++ {
								hitDmg := a.Damage * dmgMultiplier * daoMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path))
								if h > 0 {
									hitDmg *= 0.7
								}
								netDmg := hitDmg - targetDef
								netDmg *= (1.0 - pathBonus(target.PathResistance, a.Path))
								if netDmg < 1 {
									netDmg = 1
								}
								totalDmg += applyDamageToTarget(state, actor, target, netDmg, &turnLogs, turn, shieldBypass, !isAlly, rng)
							}
							totalDamageDealt += int(totalDmg)

							if int(totalDmg) > 0 {
								damageDetail := fmt.Sprintf("%s dùng [%s] lên %s, gây %d sát thương (còn %d HP)", actor.Name, a.Name, target.Name, int(totalDmg), target.HP)
								if hasDao && daoMult > 1.0 {
									appliedTotal := int(totalDmg)
									daoShare := int(float64(appliedTotal) * (daoMult - 1.0) / daoMult)
									if daoShare > 0 {
										baseShare := appliedTotal - daoShare
										damageDetail = fmt.Sprintf("%s dùng [%s] lên %s, gây %d + %d sát thương do Đạo Ngân (còn %d HP)", actor.Name, a.Name, target.Name, baseShare, daoShare, target.HP)
									}
								}
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    a.ActionType,
									TargetId:  target.ID,
									Damage:    int(totalDmg),
									Detail:    damageDetail,
								})
							} else if target.VirtualShield > 0 {
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    a.ActionType,
									TargetId:  target.ID,
									Damage:    0,
									Detail:    fmt.Sprintf("%s dùng [%s] lên %s, toàn bộ sát thương bị khiên hấp thụ (còn %d khiên)", actor.Name, a.Name, target.Name, target.VirtualShield),
								})
							}

							if a.PhantomDmg > 0 {
								phantomDef := float64(target.BaseDef)
								phantomNet := (a.PhantomDmg * dmgMultiplier) - phantomDef
								phantomNet *= (1.0 - pathBonus(target.PathResistance, a.Path))
								phantomNet = math.Max(1, phantomNet)
								phantomDealt := applyDamageToTarget(state, actor, target, phantomNet, &turnLogs, turn, shieldBypass, !isAlly, rng)
								totalDamageDealt += int(phantomDealt)
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "PHANTOM_ATTACK",
									TargetId:  target.ID,
									Damage:    int(phantomDealt),
									Detail:    fmt.Sprintf("Hư ảnh của %s tấn công thêm %d sát thương lên %s!", actor.Name, int(phantomDealt), target.Name),
								})
							}

							// ── Freeze break: if target is frozen and takes damage, 50% chance to break ──
							if target.IsFrozen && int(totalDmg) > 0 {
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
								Damage:    int(thornsNet),
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
							healAmount := int(float64(totalDamageDealt) * a.LifeStealPct)
							hpNeeded := actor.MaxHP - actor.HP
							if healAmount <= hpNeeded {
								actor.HP += healAmount
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "LIFESTEAL",
									Damage:    healAmount,
									Detail:    fmt.Sprintf("%s hút %d máu từ sát chiêu, hồi phục HP!", actor.Name, healAmount),
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
									Detail:    fmt.Sprintf("%s hút %d máu từ sát chiêu, HP đầy, chuyển %d thành khiên!", actor.Name, healAmount, excess),
								})
							}
						}
					} else {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    a.ActionType,
							TargetId:  firstTarget.ID,
							Detail:    fmt.Sprintf("%s dùng [%s]", actor.Name, a.Name),
						})

						if shieldLog != nil {
							turnLogs = append(turnLogs, *shieldLog)
						}
						if healLog != nil {
							turnLogs = append(turnLogs, *healLog)
						}
					}

					// ── Apply status effects + self-buffs (always, no damage required) ──
					for _, target := range resolved {
						if a.BleedPerTurn > 0 {
							target.ActiveDoT = append(target.ActiveDoT, DoTEffect{
								Type: "bleed", Value: a.BleedPerTurn, Turns: 3,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "BLEED_APPLIED",
								Detail:    fmt.Sprintf("%s bị chảy máu (%.0f/lượt trong 3 lượt)!", target.Name, a.BleedPerTurn),
							})
						}
						if a.PoisonPerTurn > 0 {
							target.ActiveDoT = append(target.ActiveDoT, DoTEffect{
								Type: "poison", Value: a.PoisonPerTurn, Turns: 3,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "POISON_APPLIED",
								Detail:    fmt.Sprintf("%s bị trúng độc (%.0f/lượt trong 3 lượt)!", target.Name, a.PoisonPerTurn),
							})
						}
						if a.StunTier > 0 && !resistControl(target, rng) {
							dur := TierToDuration(EffectStun, a.StunTier)
							target.ActiveEffects = append(target.ActiveEffects, Effect{
								ID: EffectStun, Type: "debuff", Tier: a.StunTier, Value: 1, Duration: dur,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "STUN_APPLIED",
								Detail:    fmt.Sprintf("%s dùng [%s] → %s bị choáng %d lượt!", actor.Name, a.Name, target.Name, dur),
							})
						}
						if a.SlowTier > 0 && !resistControl(target, rng) {
							pct := int(TierToPercent(EffectSlow, a.SlowTier))
							dur := TierToDuration(EffectSlow, a.SlowTier)
							target.ActiveEffects = append(target.ActiveEffects, Effect{
								ID: EffectSlow, Type: "debuff", Tier: a.SlowTier, Value: pct, Duration: dur,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "SLOW_APPLIED",
								Detail:    fmt.Sprintf("%s bị chậm %d%% trong %d lượt!", target.Name, pct, dur),
							})
						}
						if a.Slow1Tier > 0 {
							pct := int(TierToPercent(EffectSlow1, a.Slow1Tier))
							dur := TierToDuration(EffectSlow1, a.Slow1Tier)
							target.ActiveEffects = append(target.ActiveEffects, Effect{
								ID: EffectSlow1, Type: "debuff", Tier: a.Slow1Tier, Value: pct, Duration: dur,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "SLOW1_APPLIED",
								Detail:    fmt.Sprintf("%s bị chậm %d%% trong %d lượt (không thể kháng)!", target.Name, pct, dur),
							})
						}
						if a.SlowDamageTier > 0 && !resistControl(target, rng) {
							pct := int(TierToPercent(EffectSlowDamage, a.SlowDamageTier))
							dur := TierToDuration(EffectSlowDamage, a.SlowDamageTier)
							target.ActiveEffects = append(target.ActiveEffects, Effect{
								ID: EffectSlowDamage, Type: "debuff", Tier: a.SlowDamageTier, Value: pct, Duration: dur,
							})
							target.ActiveDoT = append(target.ActiveDoT, DoTEffect{
								Type: EffectSlowDamage, Value: float64(pct), Turns: dur,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "SLOW_DAMAGE_APPLIED",
								Detail:    fmt.Sprintf("%s bị chậm sát thương %d%% và mất %d HP/lượt trong %d lượt!", target.Name, pct, pct, dur),
							})
						}
						if a.WeakTier > 0 && !resistControl(target, rng) {
							pct := int(TierToPercent(EffectWeak, a.WeakTier))
							dur := TierToDuration(EffectWeak, a.WeakTier)
							target.ActiveEffects = append(target.ActiveEffects, Effect{
								ID: EffectWeak, Type: "debuff", Tier: a.WeakTier, Value: pct, Duration: dur,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "WEAK_APPLIED",
								Detail:    fmt.Sprintf("%s bị yếu, giảm %d%% sát thương trong %d lượt!", target.Name, pct, dur),
							})
						}
						if a.FreezeTier > 0 && !resistControl(target, rng) {
							dur := TierToDuration(EffectFreeze, a.FreezeTier)
							target.ActiveEffects = append(target.ActiveEffects, Effect{
								ID: EffectFreeze, Type: "debuff", Tier: a.FreezeTier, Value: 1, Duration: dur,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "FREEZE_APPLIED",
								Detail:    fmt.Sprintf("%s bị đóng băng trong %d lượt!", target.Name, dur),
							})
						}
						if a.SilenceTier > 0 && !resistControl(target, rng) {
							dur := TierToDuration(EffectSilence, a.SilenceTier)
							target.ActiveEffects = append(target.ActiveEffects, Effect{
								ID: EffectSilence, Type: "debuff", Tier: a.SilenceTier, Value: 1, Duration: dur,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "SILENCE_APPLIED",
								Detail:    fmt.Sprintf("%s dùng [%s] → %s bị im lặng %d lượt!", actor.Name, a.Name, target.Name, dur),
							})
						}
					}

					if a.ThornsTier > 0 {
						pct := int(TierToPercent(EffectThorns, a.ThornsTier))
						dur := TierToDuration(EffectThorns, a.ThornsTier)
						actor.ActiveEffects = append(actor.ActiveEffects, Effect{
							ID: EffectThorns, Type: "buff", Tier: a.ThornsTier, Value: pct, Duration: dur,
						})
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "THORNS_BUFF",
							Detail:    fmt.Sprintf("%s nhận Khiên Gai, phản %d%% sát thương trong %d lượt!", actor.Name, pct, dur),
						})
					}
					if a.RegenTier > 0 {
						hpPct := TierToPercent(EffectRegen, a.RegenTier)
						dur := TierToDuration(EffectRegen, a.RegenTier)
						regenPerTurn := float64(actor.MaxHP) * hpPct / 100.0
						actor.ActiveDoT = append(actor.ActiveDoT, DoTEffect{
							Type: "regen", Value: regenPerTurn, Turns: dur,
						})
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "REGEN_BUFF",
							Detail:    fmt.Sprintf("%s nhận Hồi Sinh, hồi %.0f HP/lượt trong %d lượt!", actor.Name, regenPerTurn, dur),
						})
					}
					if a.AtkUpTier > 0 {
						pct := int(TierToPercent(EffectAtkUp, a.AtkUpTier))
						dur := TierToDuration(EffectAtkUp, a.AtkUpTier)
						actor.ActiveEffects = append(actor.ActiveEffects, Effect{
							ID: EffectAtkUp, Type: "buff", Tier: a.AtkUpTier, Value: pct, Duration: dur,
						})
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "ATK_UP_BUFF",
							Detail:    fmt.Sprintf("%s nhận +%d%% Công Kích trong %d lượt!", actor.Name, pct, dur),
						})
					}
					if a.DefUpTier > 0 {
						pct := int(TierToPercent(EffectDefUp, a.DefUpTier))
						dur := TierToDuration(EffectDefUp, a.DefUpTier)
						actor.ActiveEffects = append(actor.ActiveEffects, Effect{
							ID: EffectDefUp, Type: "buff", Tier: a.DefUpTier, Value: pct, Duration: dur,
						})
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "DEF_UP_BUFF",
							Detail:    fmt.Sprintf("%s nhận +%d%% Phòng Thủ trong %d lượt!", actor.Name, pct, dur),
						})
					}
					if a.TauntTier > 0 {
						dur := TierToDuration(EffectTaunt, a.TauntTier)
						actor.ActiveEffects = append(actor.ActiveEffects, Effect{
							ID: EffectTaunt, Type: "buff", Tier: a.TauntTier, Value: 1, Duration: dur,
						})
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "TAUNT_BUFF",
							Detail:    fmt.Sprintf("%s khiêu khích, buộc kẻ địch tấn công mình trong %d lượt!", actor.Name, dur),
						})
					}

					// ── Registry-driven statuses (BURN, FEAR, CONFUSE, ...) ──
					if len(a.StatusOps) > 0 {
						applyStatusOps(actor, resolved, a.StatusOps, turn, rng, &turnLogs)
					}

					// Merge action logs into a single consolidated log entry for this Gu/Action
					if len(turnLogs) > actionStartIndex {
						logs := make([]CombatLog, len(turnLogs)-actionStartIndex)
						copy(logs, turnLogs[actionStartIndex:])

						var useLog *CombatLog
						var costLog *CombatLog
						var gainLog *CombatLog
						var resultLogs []CombatLog

						for i := range logs {
							log := &logs[i]
							if strings.Contains(log.Detail, " dùng [") && useLog == nil {
								useLog = log
							} else if log.Action == "MANA_COST" {
								costLog = log
							} else if log.Action == "PATH_MASTERY_GAIN" {
								gainLog = log
							} else if log.Action != "PATH_MASTERY_CAP" { // skip or merge cap if needed
								resultLogs = append(resultLogs, *log)
							}
						}

						if useLog != nil {
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

				// ── Multi-action per turn: actions resolve one at a time; after
				// each damage-dealing action the actor yields so the next actor
				// (sorted by speed) gets to act. Non-damage actions (buffs, heals,
				// shields, DoTs) execute freely without yielding. ──
				startCursor := actionCursor[actor]
				ai := actionCursor[actor]
				for ai < len(actor.Actions) {
					out := runOne(actor, &actor.Actions[ai], isAlly, enemyTargets)
					postActionChecks(enemyTargets)
					if out.abort {
						actionCursor[actor] = len(actor.Actions)
						break
					}
					ai++
					actionCursor[actor] = ai
					if out.executed {
						executedOnce[actor] = true
						if out.hasDamage {
							break // Damage → yield to next actor
						}
					}
				}
				if actionCursor[actor] > startCursor {
					anyActionsRemain = true
				}

				// Fallback: basic attack if no valid action
				if !executedOnce[actor] {
					daoMult, _ := actorDaoMultiplier(actor, []string{"strength"})
					// Apply weak debuff to fallback basic attack
					fallbackDmgMult := actor.DamageMultiplier
					fallbackAction := &SimAction{
						ActionType: "BASIC_ATTACK",
						Name:       "Đánh thường",
						Damage:     float64(actor.BaseDmg) * daoMult * fallbackDmgMult,
						ManaCost:   0,
						TargetType: "SINGLE",
					}
					fallbackTargets := resolveTargets(enemyTargets, fallbackAction)
					if len(fallbackTargets) > 0 {
						target := fallbackTargets[0]
						netDmg := fallbackAction.Damage - float64(target.BaseDef)
						netDmg *= (1.0 - pathBonus(target.PathResistance, []string{"strength"}))
						if netDmg < 1 {
							netDmg = 1
						}
						applied := applyDamageToTarget(state, actor, target, netDmg, &turnLogs, turn, false, !isAlly, rng)
						detailMsg := fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f sát thương (còn %d HP)", actor.Name, fallbackAction.Name, target.Name, applied, target.HP)
						if daoMult > 1.0 {
							baseDmgVal := float64(actor.BaseDmg)
							daoDmgVal := baseDmgVal * (daoMult - 1.0)
							detailMsg = fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f + %.0f sát thương do Đạo Ngân (còn %d HP)", actor.Name, fallbackAction.Name, target.Name, baseDmgVal, daoDmgVal, target.HP)
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "BASIC_ATTACK",
							TargetId:  target.ID,
							Damage:    int(applied),
							Detail:    detailMsg,
						})
					}
				}

				postActionChecks(enemyTargets)
			}
		} // end for anyActionsRemain

		// Reset cursors for next turn when all actions exhausted.
		for _, actor := range allActors {
			if actionCursor[actor] >= len(actor.Actions) {
				actionCursor[actor] = 0
			}
		}

		logs = append(logs, turnLogs...)

		// ── Death triggers check (e.g. Xuân Thu Thiền) ──
		rewound := false
		for _, a := range state.Allies {
			if a.HP <= 0 && hasGuEquipped(a, "GU_SPRING_AUTUMN_CICADA") && !cicadaUsed && !a.IsReviveLocked {
				if rng.Float64() < 0.15 {
					cicadaUsed = true
					rewound = true
					logs = append(logs, CombatLog{
						Turn:      turn,
						ActorId:   a.ID,
						ActorName: a.Name,
						Action:    "TRIGGERED",
						Trigger:   "GU_SPRING_AUTUMN_CICADA",
						Detail:    "Xuân Thu Thiền kích hoạt! Nghịch chuyển thời không, quay về Lượt 1!",
					})
					break
				}
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
			Damage: float64(a.BaseDmg), Defense: float64(a.BaseDef),
			Speed: a.EffectiveSpeed, IsDead: a.IsDead,
		})
	}
	for _, e := range state.Enemies {
		snap.Enemies = append(snap.Enemies, CombatantSnapshot{
			ID: e.ID, Name: e.Name, Kind: e.Kind,
			HP: e.HP, MaxHP: e.MaxHP, Mana: e.Mana, MaxMana: e.MaxMana,
			GridX: e.GridX, GridY: e.GridY, Shield: e.VirtualShield,
			Damage: float64(e.BaseDmg), Defense: float64(e.BaseDef),
			Speed: e.EffectiveSpeed, IsDead: e.IsDead,
		})
	}
	return snap
}

func applyDamageToTarget(state *CombatState, actor, target *CombatantAuto, damage float64, logs *[]CombatLog, turn int, shieldBypass bool, spillToAperture bool, rng *rand.Rand) float64 {
	if damage < 1 {
		damage = 1
	}

	// Định Tiên Du first turn evade
	if turn == 1 && hasGuEquipped(target, "GU_FIXED_IMMORTAL_TRAVEL") {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "EVADE",
			Detail:    fmt.Sprintf("%s kích hoạt Định Tiên Du, né tránh đòn đánh đầu tiên hoàn toàn!", target.Name),
		})
		return 0
	}

	// Cổ Tình Yêu invulnerable check
	isInvulnerable := false
	for _, eff := range target.ActiveEffects {
		if eff.ID == "invulnerable" && eff.Duration >= 0 { // remains active
			isInvulnerable = true
			break
		}
	}
	if isInvulnerable {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "BLOCKED",
			Detail:    fmt.Sprintf("%s miễn nhiễm mọi sát thương nhờ sức mạnh bất tử của Cổ Tình Yêu!", target.Name),
		})
		return 0
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
		target.HP -= int(damage)
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
			Damage:    int(damage),
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
		if hasGuEquipped(actor, "GU_POEM_IMMORTAL") {
			stacks := actor.CustomState["thi_y"]

			// Xuyên khiên (chỉ 1 lần mỗi lượt)
			if stacks >= 3 && !shieldBypass && actor.CustomState["thi_y_pierce_turn"] != turn {
				actor.CustomState["thi_y_pierce_turn"] = turn

				usableMultiples := stacks / 3
				ignorePct := float64(usableMultiples) * 0.30
				if ignorePct > 1.0 {
					ignorePct = 1.0
				}

				shieldIgnored := float64(target.VirtualShield) * ignorePct
				if shieldIgnored > damage {
					shieldIgnored = damage
				}

				if shieldIgnored > 0 {
					target.HP -= int(shieldIgnored)
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

		if float64(target.VirtualShield) >= damage {
			target.VirtualShield -= int(damage)
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "SHIELD_ABSORB",
				Damage:    int(damage),
				Detail:    fmt.Sprintf("%s tấn công %s gây %.0f sát thương (còn %d khiên, %d HP)", actor.Name, target.Name, damage, target.VirtualShield, target.HP),
			})
			return hpDamageDealt // Trả về hpDamageDealt để caller không log double damage
		}

		absorbed := float64(target.VirtualShield)
		damage -= absorbed
		target.VirtualShield = 0
		target.HP -= int(damage)
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
			Damage:    int(totalDmg),
			Detail:    fmt.Sprintf("%s tấn công %s gây %.0f sát thương (phá vỡ %.0f khiên, còn %d HP)", actor.Name, target.Name, totalDmg, absorbed, target.HP),
		})
		return hpDamageDealt // Chỉ trả về lượng sát thương trừ vào HP để caller log
	}

	// Direct HP damage (no shield)
	target.HP -= int(damage)
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
		Detail:    fmt.Sprintf("Tiên Khiếu chịu %d sát thương xuyên phá (còn %d/%d)!", overflow, state.ApertureHP, state.ApertureMaxHP),
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
		return float64(actor.HP)/float64(actor.MaxHP)*100 <= action.ConditionValue
	case "mana_above":
		if actor.MaxMana <= 0 {
			return true
		}
		return float64(actor.Mana)/float64(actor.MaxMana)*100 >= action.ConditionValue
	case "enemy_hp_below":
		if target == nil || target.MaxHP <= 0 {
			return true
		}
		return float64(target.HP)/float64(target.MaxHP)*100 <= action.ConditionValue
	default:
		return true
	}
}
