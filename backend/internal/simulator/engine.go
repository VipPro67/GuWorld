package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
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

func cloneCombatState(state *CombatState) *CombatState {
	if state == nil {
		return nil
	}
	clone := &CombatState{
		Turn:        state.Turn,
		Seed:        state.Seed,
		BossPhase:   state.BossPhase,
		VirtualMode: state.VirtualMode,
		Allies:      make([]*CombatantAuto, len(state.Allies)),
		Enemies:     make([]*CombatantAuto, len(state.Enemies)),
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
	return &clone
}

func getTarget(enemies []*CombatantAuto) *CombatantAuto {
	for _, e := range enemies {
		if e.GridX == 1 && e.HP > 0 {
			return e
		}
	}
	for _, e := range enemies {
		if e.GridX == 2 && e.HP > 0 {
			return e
		}
	}
	for _, e := range enemies {
		if e.GridX == 3 && e.HP > 0 {
			return e
		}
	}
	for _, e := range enemies {
		if e.HP > 0 {
			return e
		}
	}
	return nil
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
		hits = append(hits, alive[0])
		return hits

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
		hits = append(hits, alive[0])
		return hits

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
		hits = append(hits, alive[0])
		return hits

	case "ALL":
		return alive

	default: // "SINGLE"
		// priority by GridX (front first), then GridY mid
		var best *CombatantAuto
		for _, e := range alive {
			if e.GridX == targetX && e.GridY == targetY {
				best = e
				break
			}
		}
		if best == nil {
			for _, e := range alive {
				if e.GridX == targetX {
					best = e
					break
				}
			}
		}
		if best == nil {
			for _, e := range alive {
				if e.GridX == 1 && e.HP > 0 {
					best = e
					break
				}
			}
		}
		if best == nil {
			for _, e := range alive {
				if e.GridX == 2 && e.HP > 0 {
					best = e
					break
				}
			}
		}
		if best == nil {
			for _, e := range alive {
				if e.GridX == 3 && e.HP > 0 {
					best = e
					break
				}
			}
		}
		if best == nil && len(alive) > 0 {
			best = alive[0]
		}
		if best != nil {
			return []*CombatantAuto{best}
		}
		return nil
	}
}

// Simulate runs the deterministic auto-battle using pre-computed actions.
func Simulate(state *CombatState) []CombatLog {
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

		// ── Decrement active effects durations ──
		for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
			for _, c := range combatants {
				var kept []Effect
				for _, eff := range c.ActiveEffects {
					eff.Duration--
					if eff.Duration > 0 {
						kept = append(kept, eff)
					}
				}
				c.ActiveEffects = kept
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
			return allActors[i].Speed > allActors[j].Speed
		})

		for _, actor := range allActors {
			if actor.HP <= 0 || actor.IsDead {
				continue
			}

			// Check stun
			isStunned := false
			stunIdx := -1
			for idx, eff := range actor.ActiveEffects {
				if eff.ID == "stun" && eff.Duration >= 0 { // duration remains at least 0
					isStunned = true
					stunIdx = idx
					break
				}
			}
			if isStunned {
				// Ngã Ý Cổ 35% chance to cleanse stun
				if hasGuEquipped(actor, "GU_MY_WILL") && rand.Float64() < 0.35 {
					actor.ActiveEffects[stunIdx].Duration = 0
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "CLEANSE",
						Detail:    fmt.Sprintf("Ngã Ý Cổ kích hoạt! Giải trừ trạng thái Choáng cho %s đầu lượt!", actor.Name),
					})
					isStunned = false
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

			// Auto-restore mana using stones
			if actor.AutoRestoreMana && actor.Mana < actor.MaxMana {
				manaPct := float64(actor.Mana) / float64(actor.MaxMana) * 100
				if manaPct <= actor.ManaThreshold {
					regenMult := 1.0
					if actor.RefineManaTier > 0 {
						regenMult = 0.4 + actor.RefineManaTier*0.1
					}
					if actor.ImmortalStones >= 1 {
						actor.ImmortalStones -= 1
						restore := int(float64(100) * regenMult)
						actor.Mana += restore
						if actor.Mana > actor.MaxMana {
							actor.Mana = actor.MaxMana
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "MANA_RESTORE",
							Detail:    fmt.Sprintf("%s tự động nghiền nát Tiên Nguyên Thạch, khôi phục %d tiên nguyên!", actor.Name, restore),
						})
					} else if actor.PrimevalStones >= 100 {
						actor.PrimevalStones -= 100
						restore := int(float64(50) * regenMult)
						actor.Mana += restore
						if actor.Mana > actor.MaxMana {
							actor.Mana = actor.MaxMana
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "MANA_RESTORE",
							Detail:    fmt.Sprintf("%s tự động hấp thu 100 Nguyên Thạch, khôi phục %d chân nguyên!", actor.Name, restore),
						})
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

			// ── Multi-action per turn: execute all valid actions ──
			if len(actor.Actions) == 0 {
				continue
			}
			// Sort by priority once
			if actor.nextActionIdx == 0 {
				sort.SliceStable(actor.Actions, func(i, j int) bool {
					return actor.Actions[i].Priority < actor.Actions[j].Priority
				})
				actor.nextActionIdx = 1
			}

			var enemyTargets []*CombatantAuto
			if isAlly {
				enemyTargets = state.Enemies
			} else {
				enemyTargets = state.Allies
			}

			hasExecutedAny := false
			for ai := range actor.Actions {
				a := &actor.Actions[ai]

				if a.ManaCost > 0 && float64(actor.Mana) < a.ManaCost {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "INSUFFICIENT_MANA",
						Detail:    fmt.Sprintf("%s không đủ chân nguyên để dùng [%s] (cần %.0f, còn %d)", actor.Name, a.Name, a.ManaCost, actor.Mana),
					})
					continue
				}

				resolved := resolveTargets(enemyTargets, a)
				if len(resolved) == 0 {
					break
				}
				firstTarget := resolved[0]

				if !checkActionCondition(a, actor, firstTarget) {
					continue
				}

				hasExecutedAny = true
				manaCost := int(a.ManaCost)
				actor.Mana -= manaCost

				if manaCost > 0 {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "MANA_COST",
						Damage:    manaCost,
						Detail:    fmt.Sprintf("%s tiêu hao %d chân nguyên (còn %d/%d)", actor.Name, manaCost, actor.Mana, actor.MaxMana),
					})
				}

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
						break
					}
				}

				// Dao mark multiplier for this action's path (applies to damage, shield, heal)
				daoMult, hasDao := DaoMultiplier(actor.DaoMarks, a.Path)

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
					heal := baseHeal + daoHeal
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

				if a.SourceID == "GU_GUTS_GU" {
					actor.HP = actor.MaxHP
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "HEAL",
						Detail:    fmt.Sprintf("%s sử dụng Cổ Gan Dạ, hồi phục hoàn toàn HP linh hồn nhưng mất lượt hành động!", actor.Name),
					})
					break // Break action loop (Turn_Cost = 1)
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

					chosenEffect := effectsChoices[rand.Intn(len(effectsChoices))]
					chosenCost := costChoices[rand.Intn(len(costChoices))]

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

						targetDef := float64(target.BaseDef)
						if a.SourceID == "GU_BREAK_SPACE" {
							targetDef = math.Max(0, targetDef-30.0)
						}

						for h := 0; h < hitCount; h++ {
							hitDmg := a.Damage * dmgMultiplier * daoMult
							if h > 0 {
								hitDmg *= 0.7
							}
							netDmg := hitDmg - targetDef
							if netDmg < 1 {
								netDmg = 1
							}
							totalDmg += applyDamageToTarget(actor, target, netDmg, &turnLogs, turn, shieldBypass)
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
						}

						if a.PhantomDmg > 0 {
							phantomDef := float64(target.BaseDef)
							phantomNet := math.Max(1, (a.PhantomDmg*dmgMultiplier)-phantomDef)
							phantomDealt := applyDamageToTarget(actor, target, phantomNet, &turnLogs, turn, shieldBypass)
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

						// Apply status effects per target
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
						if a.StunTurns > 0 {
							target.ActiveEffects = append(target.ActiveEffects, Effect{
								ID: "stun", Type: "debuff", Value: a.StunTurns, Duration: a.StunTurns,
							})
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "STUN_APPLIED",
								Detail:    fmt.Sprintf("%s bị choáng trong %d lượt!", target.Name, a.StunTurns),
							})
						}
					}

					if shieldLog != nil {
						turnLogs = append(turnLogs, *shieldLog)
					}
					if healLog != nil {
						turnLogs = append(turnLogs, *healLog)
					}

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
						applyDamageToTarget(firstTarget, actor, thornsNet, &turnLogs, turn, false)
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
			}

			// Fallback: basic attack if no valid action
			if !hasExecutedAny {
				daoMult := 1.0
				if actor.DaoMarks != nil {
					if amt, ok := actor.DaoMarks["strength"]; ok && amt > 0 {
						if amt <= 10000 {
							daoMult = 1.0 + float64(amt)/100.0
						} else {
							daoMult = 1.0 + 100.0 + float64(amt-10000)/500.0
						}
					}
				}
				fallbackAction := &SimAction{
					ActionType: "BASIC_ATTACK",
					Name:       "Đánh thường",
					Damage:     float64(actor.BaseDmg) * daoMult,
					ManaCost:   0,
					TargetType: "SINGLE",
				}
				fallbackTargets := resolveTargets(enemyTargets, fallbackAction)
				if len(fallbackTargets) > 0 {
					target := fallbackTargets[0]
					netDmg := fallbackAction.Damage - float64(target.BaseDef)
					if netDmg < 1 {
						netDmg = 1
					}
					applied := applyDamageToTarget(actor, target, netDmg, &turnLogs, turn, false)
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

			// Dying checks for all enemies
			for _, tgt := range enemyTargets {
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

			// Boss phase transitions
			for _, tgt := range enemyTargets {
				if tgt.Kind == "Boss" {
					turnLogs = append(turnLogs, CheckBossPhaseTransitions(state, tgt)...)
				}
			}
		}

		logs = append(logs, turnLogs...)

		// ── Death triggers check (e.g. Xuân Thu Thiền) ──
		rewound := false
		for _, a := range state.Allies {
			if a.HP <= 0 && hasGuEquipped(a, "GU_SPRING_AUTUMN_CICADA") && !cicadaUsed {
				if rand.Float64() < 0.15 {
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
	snap := &CombatSnapshot{Turn: state.Turn}
	for _, a := range state.Allies {
		snap.Allies = append(snap.Allies, CombatantSnapshot{
			ID: a.ID, Name: a.Name, Kind: a.Kind,
			HP: a.HP, MaxHP: a.MaxHP, Mana: a.Mana, MaxMana: a.MaxMana,
			GridX: a.GridX, GridY: a.GridY, Shield: a.VirtualShield,
			Damage: float64(a.BaseDmg), Defense: float64(a.BaseDef),
			Speed: a.Speed, IsDead: a.IsDead,
		})
	}
	for _, e := range state.Enemies {
		snap.Enemies = append(snap.Enemies, CombatantSnapshot{
			ID: e.ID, Name: e.Name, Kind: e.Kind,
			HP: e.HP, MaxHP: e.MaxHP, Mana: e.Mana, MaxMana: e.MaxMana,
			GridX: e.GridX, GridY: e.GridY, Shield: e.VirtualShield,
			Damage: float64(e.BaseDmg), Defense: float64(e.BaseDef),
			Speed: e.Speed, IsDead: e.IsDead,
		})
	}
	return snap
}

func applyDamageToTarget(actor, target *CombatantAuto, damage float64, logs *[]CombatLog, turn int, shieldBypass bool) float64 {
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

	// Phá Không Cổ shield bypass
	if shieldBypass {
		target.HP -= int(damage)
		if target.HP < 0 {
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

	if target.VirtualShield > 0 {
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
			return 0
		}

		absorbed := target.VirtualShield
		damage -= float64(absorbed)
		target.VirtualShield = 0
		target.HP -= int(damage)
		if target.HP < 0 {
			target.HP = 0
		}
		totalDmg := float64(absorbed) + damage
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SHIELD_BREAK",
			Damage:    int(totalDmg),
			Detail:    fmt.Sprintf("%s tấn công %s gây %.0f sát thương (vỡ khiên, còn %d HP)", actor.Name, target.Name, totalDmg, target.HP),
		})
		return totalDmg
	}

	// Direct HP damage (no shield)
	target.HP -= int(damage)
	if target.HP < 0 {
		target.HP = 0
	}
	return damage
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
