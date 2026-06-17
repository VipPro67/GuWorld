package simulator

import (
	"fmt"
	"math"
	"sort"
)

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

	logs = append(logs, CombatLog{
		Turn:   0,
		Action: "BATTLE_START",
		Detail: "Trận chiến bắt đầu. Kích hoạt Giáp Ảo trước trận...",
	})

	for _, p := range state.Allies {
		if p.VirtualShield > 0 {
			logs = append(logs, CombatLog{
				Turn:      0,
				ActorId:   p.ID,
				ActorName: p.Name,
				Action:    "PRE_COMBAT_BARRIER",
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

		// ── Decrement stun durations ──
		for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
			for _, c := range combatants {
				var kept []Effect
				for _, eff := range c.ActiveEffects {
					if eff.ID == "stun" {
						eff.Duration--
						if eff.Duration > 0 {
							kept = append(kept, eff)
						}
					} else {
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
			for _, eff := range actor.ActiveEffects {
				if eff.ID == "stun" && eff.Duration > 0 {
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

			// ── Select action by priority + conditions ──
			if len(actor.Actions) == 0 {
				continue
			}
			// Sort by priority once (stable, first use)
			if actor.nextActionIdx == 0 {
				sort.SliceStable(actor.Actions, func(i, j int) bool {
					return actor.Actions[i].Priority < actor.Actions[j].Priority
				})
			}
			actor.nextActionIdx++

			var selectedAction *SimAction
			for i := range actor.Actions {
				a := &actor.Actions[i]
				if a.ManaCost > 0 && float64(actor.Mana) < a.ManaCost {
					continue
				}
				if !checkActionCondition(a, actor, target) {
					continue
				}
				selectedAction = a
				break
			}
			if selectedAction == nil {
				// Fallback: basic attack (zero mana, always usable)
				for i := range actor.Actions {
					if actor.Actions[i].ActionType == "BASIC_ATTACK" {
						selectedAction = &actor.Actions[i]
						break
					}
				}
				if selectedAction == nil {
					// Create a basic attack on-the-fly
					ba := SimAction{
						ActionType: "BASIC_ATTACK",
						Name:       "Đánh thường",
						Damage:     float64(actor.BaseDmg),
						ManaCost:   0,
					}
					actor.Actions = append(actor.Actions, ba)
					selectedAction = &actor.Actions[len(actor.Actions)-1]
				}
			}
			action := *selectedAction

			// ── Resolve the action ──
			turnLogs = append(turnLogs, CombatLog{
				Turn:      turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    action.ActionType,
				TargetId:  target.ID,
				Detail:    fmt.Sprintf("%s sử dụng [%s]", actor.Name, action.Name),
			})

			actor.Mana -= int(action.ManaCost)

			// Apply shield to self
			if action.Defense > 0 {
				actor.VirtualShield += int(action.Defense)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SHIELD_GAIN",
					Detail:    fmt.Sprintf("%s nhận %.0f Khiên Ảo!", actor.Name, action.Defense),
				})
			}

			// Apply healing to self
			if action.Healing > 0 {
				heal := int(action.Healing)
				actor.HP += heal
				if actor.HP > actor.MaxHP {
					actor.HP = actor.MaxHP
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "HEAL",
					Detail:    fmt.Sprintf("%s hồi phục %d HP!", actor.Name, heal),
				})
			}

			// ── Apply damage & compound effects ──
			totalDamageDealt := 0

			if action.ActionType == "BASIC_ATTACK" || action.Damage > 0 {
				// Multi-hit
				hitCount := action.MultiHit
				if hitCount < 1 {
					hitCount = 1
				}
				totalDmg := 0.0
				for h := 0; h < hitCount; h++ {
					hitDmg := action.Damage
					if h > 0 {
						hitDmg *= 0.7 // chain diminishing
					}
					// Defense mitigation
					netDmg := hitDmg - float64(target.BaseDef)
					if netDmg < 1 {
						netDmg = 1
					}
					totalDmg += applyDamageToTarget(actor, target, netDmg, &turnLogs, turn)
				}
				totalDamageDealt = int(totalDmg)

				// Log HP damage
				if totalDamageDealt > 0 {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "DAMAGE",
						TargetId:  target.ID,
						Damage:    totalDamageDealt,
						Detail:    fmt.Sprintf("%s gây %d sát thương lên %s (còn %d HP)", actor.Name, totalDamageDealt, target.Name, target.HP),
					})
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

				// Phantom damage
				if action.PhantomDmg > 0 {
					phantomNet := math.Max(1, action.PhantomDmg-float64(target.BaseDef))
					phantomDealt := applyDamageToTarget(actor, target, phantomNet, &turnLogs, turn)
					totalDamageDealt += int(phantomDealt)
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "PHANTOM_ATTACK",
						TargetId:  target.ID,
						Damage:    int(phantomDealt),
						Detail:    fmt.Sprintf("Hư ảnh tấn công thêm %.0f sát thương!", phantomDealt),
					})
				}

				// Thorns (reflect) to attacker
				if action.ThornsDmg > 0 && totalDamageDealt > 0 {
					thornsNet := math.Max(1, action.ThornsDmg-float64(actor.BaseDef))
					applyDamageToTarget(target, actor, thornsNet, &turnLogs, turn)
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

				// Lifesteal
				if action.LifeStealPct > 0 && totalDamageDealt > 0 {
					healAmount := int(float64(totalDamageDealt) * action.LifeStealPct)
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

				// Apply status effects
				if action.BleedPerTurn > 0 {
					target.ActiveDoT = append(target.ActiveDoT, DoTEffect{
						Type: "bleed", Value: action.BleedPerTurn, Turns: 3,
					})
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   target.ID,
						ActorName: target.Name,
						Action:    "BLEED_APPLIED",
						Detail:    fmt.Sprintf("%s bị chảy máu (%.0f/lượt trong 3 lượt)!", target.Name, action.BleedPerTurn),
					})
				}
				if action.PoisonPerTurn > 0 {
					target.ActiveDoT = append(target.ActiveDoT, DoTEffect{
						Type: "poison", Value: action.PoisonPerTurn, Turns: 3,
					})
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   target.ID,
						ActorName: target.Name,
						Action:    "POISON_APPLIED",
						Detail:    fmt.Sprintf("%s bị trúng độc (%.0f/lượt trong 3 lượt)!", target.Name, action.PoisonPerTurn),
					})
				}
				if action.StunTurns > 0 {
					target.ActiveEffects = append(target.ActiveEffects, Effect{
						ID: "stun", Type: "debuff", Value: action.StunTurns, Duration: action.StunTurns,
					})
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   target.ID,
						ActorName: target.Name,
						Action:    "STUN_APPLIED",
						Detail:    fmt.Sprintf("%s bị choáng trong %d lượt!", target.Name, action.StunTurns),
					})
				}
			}

			// Dying check
			if target.HP <= 0 {
				target.HP = 0
				target.IsDead = true
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   target.ID,
					ActorName: target.Name,
					Action:    "DIED",
					Detail:    fmt.Sprintf("%s đã gục ngã!", target.Name),
				})
			}

			// Boss phase transitions
			if target.Kind == "Boss" {
				turnLogs = append(turnLogs, CheckBossPhaseTransitions(state, target)...)
			}
		}

		logs = append(logs, turnLogs...)

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
	}

	return logs
}

func applyDamageToTarget(actor, target *CombatantAuto, damage float64, logs *[]CombatLog, turn int) float64 {
	if damage < 1 {
		damage = 1
	}

	// Virtual shield absorption
	if target.VirtualShield > 0 {
		if float64(target.VirtualShield) >= damage {
			target.VirtualShield -= int(damage)
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "SHIELD_ABSORB",
				Damage:    int(damage),
				Detail:    fmt.Sprintf("%s dùng Khiên Ảo hấp thụ %.0f sát thương (còn %d khiên)", target.Name, damage, target.VirtualShield),
			})
			return 0
		} else {
			absorbed := target.VirtualShield
			damage -= float64(absorbed)
			target.VirtualShield = 0
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "SHIELD_BREAK",
				Damage:    absorbed,
				Detail:    fmt.Sprintf("Khiên Ảo của %s bị phá vỡ sau khi chặn %d sát thương!", target.Name, absorbed),
			})
		}
	}

	// Direct HP damage
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
