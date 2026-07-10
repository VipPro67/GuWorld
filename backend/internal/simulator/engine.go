package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

func hasGuEquipped(c *CombatantAuto, guID string) bool {
	if c == nil || c.Loadout == nil {
		return false
	}
	if c.Loadout.CoreGuId == guID {
		return true
	}
	for _, id := range c.Loadout.SupportGuIds {
		if id == guID {
			return true
		}
	}
	if c.Loadout.ImmortalGuId != nil && *c.Loadout.ImmortalGuId == guID {
		return true
	}
	return false
}

func cloneCombatState(state *CombatState) *CombatState {
	if state == nil {
		return nil
	}
	clone := &CombatState{
		Turn:      state.Turn,
		Seed:      state.Seed,
		BossPhase: state.BossPhase,
		Allies:    make([]*CombatantAuto, len(state.Allies)),
		Enemies:   make([]*CombatantAuto, len(state.Enemies)),
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
		if e.GridRow == 1 && e.HP > 0 {
			return e
		}
	}
	for _, e := range enemies {
		if e.GridRow == 2 && e.HP > 0 {
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

// Simulate runs the deterministic auto-battle using pre-computed actions.
func Simulate(state *CombatState) []CombatLog {
	var logs []CombatLog
	cicadaUsed := false
	initialState := cloneCombatState(state)

	totalShield := 0
	for _, p := range state.Allies {
		totalShield += p.VirtualShield
	}
	shieldMsg := "không có Khiên Ảo"
	if totalShield > 0 {
		shieldMsg = fmt.Sprintf("tổng cộng %d Khiên Ảo", totalShield)
	}
	logs = append(logs, CombatLog{
		Turn:   0,
		Action: "BATTLE_START",
		Detail: fmt.Sprintf("Trận chiến bắt đầu. Kích hoạt Giáp Ảo trước trận... (%s)", shieldMsg),
	})

	for _, p := range state.Allies {
		if p.VirtualShield > 0 {
			logs = append(logs, CombatLog{
				Turn:      0,
				ActorId:   p.ID,
				ActorName: p.Name,
				Action:    "PRE_COMBAT_BARRIER",
				Damage:    p.VirtualShield,
				Detail:    fmt.Sprintf("%s được bao bọc bởi %d Giáp Ảo!", p.Name, p.VirtualShield),
			})
		}
	}

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

		// ── Process actors ──
		var allActors []*CombatantAuto
		allActors = append(allActors, state.Allies...)
		allActors = append(allActors, state.Enemies...)

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
					if actor.ImmortalStones >= 1 {
						actor.ImmortalStones -= 1
						actor.Mana += 100
						if actor.Mana > actor.MaxMana {
							actor.Mana = actor.MaxMana
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "MANA_RESTORE",
							Detail:    fmt.Sprintf("%s tự động nghiền nát Tiên Nguyên Thạch, khôi phục 100 tiên nguyên!", actor.Name),
						})
					} else if actor.PrimevalStones >= 100 {
						actor.PrimevalStones -= 100
						actor.Mana += 50
						if actor.Mana > actor.MaxMana {
							actor.Mana = actor.MaxMana
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "MANA_RESTORE",
							Detail:    fmt.Sprintf("%s tự động hấp thu 100 Nguyên Thạch, khôi phục 50 chân nguyên!", actor.Name),
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
					continue
				}

				target := getTarget(enemyTargets)
				if target == nil {
					break
				}

				if !checkActionCondition(a, actor, target) {
					continue
				}

				hasExecutedAny = true
				actor.Mana -= int(a.ManaCost)

				// Buffer shield/heal logs — emit after action name
				var shieldLog, healLog *CombatLog

				// Apply shield to self
				if a.Defense > 0 {
					actor.VirtualShield += int(a.Defense)
					sl := CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SHIELD_GAIN",
						Damage:    int(a.Defense),
						Detail:    fmt.Sprintf("%s nhận %.0f Khiên Ảo!", actor.Name, a.Defense),
					}
					shieldLog = &sl
				}

				// Apply healing to self
				if a.Healing > 0 {
					heal := int(a.Healing)
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
						Detail:    fmt.Sprintf("%s hồi phục %d HP!", actor.Name, heal),
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
						GridRow: 1, // Frontline
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
					actor.GridRow = 2 // Move actor to Backline
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
					for _, eff := range target.ActiveEffects {
						if eff.Type == "buff" && dispelledCount < 2 {
							dispelledCount++
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "DISPEL",
								TargetId:  target.ID,
								Detail:    fmt.Sprintf("%s hóa giải hiệu ứng lợi ích [%s] của %s!", actor.Name, eff.ID, target.Name),
							})
						} else {
							keptEffects = append(keptEffects, eff)
						}
					}
					target.ActiveEffects = keptEffects

					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "INTERRUPT",
						TargetId:  target.ID,
						Detail:    fmt.Sprintf("%s sử dụng Giải Mê Cổ ngắt đòn tụ lực sát chiêu của %s!", actor.Name, target.Name),
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
						damageToEnemy := float64(target.HP) * 0.90
						target.HP -= int(damageToEnemy)
						if target.HP < 1 {
							target.HP = 1
						}
					}

					costMsg := ""
					if chosenCost == "aptitude_10" {
						costMsg = "Trừ 10% Tư chất gốc vĩnh viễn!"
					} else if chosenCost == "lifespan_5" {
						costMsg = "Trừ 5 năm tuổi thọ vĩnh viễn!"
					} else if chosenCost == "luck_50" {
						costMsg = "Trừ 50% Nguyên thạch hiện tại!"
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

				// ── Apply damage & compound effects ──
				totalDamageDealt := 0

				if a.ActionType == "BASIC_ATTACK" || a.Damage > 0 {
					hitCount := a.MultiHit
					if hitCount < 1 {
						hitCount = 1
					}
					totalDmg := 0.0

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
					targetDef := float64(target.BaseDef)
					if a.SourceID == "GU_BREAK_SPACE" {
						targetDef = math.Max(0, targetDef - 30.0)
					}

					for h := 0; h < hitCount; h++ {
						hitDmg := a.Damage * dmgMultiplier
						if h > 0 {
							hitDmg *= 0.7
						}
						netDmg := hitDmg - targetDef
						if netDmg < 1 {
							netDmg = 1
						}
						totalDmg += applyDamageToTarget(actor, target, netDmg, &turnLogs, turn, shieldBypass)
					}
					totalDamageDealt = int(totalDmg)

					if totalDamageDealt > 0 {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    a.ActionType,
							TargetId:  target.ID,
							Damage:    totalDamageDealt,
							Detail:    fmt.Sprintf("%s dùng [%s] lên %s, gây %d sát thương (còn %d HP)", actor.Name, a.Name, target.Name, totalDamageDealt, target.HP),
						})
					}

					if shieldLog != nil {
						turnLogs = append(turnLogs, *shieldLog)
					}
					if healLog != nil {
						turnLogs = append(turnLogs, *healLog)
					}

					if hitCount > 1 {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "MULTI_HIT",
							TargetId:  target.ID,
							Detail:    fmt.Sprintf("%s trúng %d đòn liên hoàn!", actor.Name, hitCount),
						})
					}

					if a.PhantomDmg > 0 {
						phantomNet := math.Max(1, (a.PhantomDmg*dmgMultiplier)-float64(target.BaseDef))
						phantomDealt := applyDamageToTarget(actor, target, phantomNet, &turnLogs, turn, shieldBypass)
						totalDamageDealt += int(phantomDealt)
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "PHANTOM_ATTACK",
							TargetId:  target.ID,
							Damage:    int(phantomDealt),
							Detail:    fmt.Sprintf("Hư ảnh của %s tấn công thêm %d sát thương!", actor.Name, int(phantomDealt)),
						})
					}

					if a.ThornsDmg > 0 && totalDamageDealt > 0 {
						thornsNet := math.Max(1, a.ThornsDmg-float64(actor.BaseDef))
						applyDamageToTarget(target, actor, thornsNet, &turnLogs, turn, false)
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "THORNS",
							TargetId:  target.ID,
							Damage:    int(thornsNet),
							Detail:    fmt.Sprintf("Phản thương từ %s gây %.0f sát thương lên %s!", target.Name, thornsNet, actor.Name),
						})
					}

					if a.LifeStealPct > 0 && totalDamageDealt > 0 {
						healAmount := int(float64(totalDamageDealt) * a.LifeStealPct)
						actor.HP += healAmount
						if actor.HP > actor.MaxHP {
							actor.HP = actor.MaxHP
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "LIFESTEAL",
							Damage:    healAmount,
							Detail:    fmt.Sprintf("%s hút %d máu từ sát chiêu!", actor.Name, healAmount),
						})
					}

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
				} else {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    a.ActionType,
						TargetId:  target.ID,
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
				target := getTarget(enemyTargets)
				if target != nil {
					ba := SimAction{
						ActionType: "BASIC_ATTACK",
						Name:       "Đánh thường",
						Damage:     float64(actor.BaseDmg),
						ManaCost:   0,
					}
					netDmg := ba.Damage - float64(target.BaseDef)
					if netDmg < 1 {
						netDmg = 1
					}
					applied := applyDamageToTarget(actor, target, netDmg, &turnLogs, turn, false)
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "BASIC_ATTACK",
						TargetId:  target.ID,
						Damage:    int(applied),
						Detail:    fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f sát thương (còn %d HP)", actor.Name, ba.Name, target.Name, applied, target.HP),
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
			logs = append(logs, CombatLog{
				Turn:   turn,
				Action: "BATTLE_END",
				Detail: fmt.Sprintf("Trận chiến kết thúc. %s chiến thắng!", winner),
			})
			break
		}

		if turn == maxTurns {
			logs = append(logs, CombatLog{
				Turn:   turn,
				Action: "BATTLE_END",
				Detail: "Hết thời gian! Người thách đấu đã thua.",
			})
		}
	}

	return logs
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
