package simulator

import (
	"fmt"
	"math"
	"math/rand"
)

// tickDoTs applies damage-over-time effects and returns log entries.
func tickDoTs(c *CombatantAuto, turn int) []CombatLog {
	var logs []CombatLog
	var remaining []DoTEffect

	for _, dot := range c.ActiveDoT {
		if dot.Turns <= 0 {
			continue
		}
		if dot.Type == "regen" {
			heal := int(dot.Value)
			c.HP += heal
			if c.HP > c.MaxHP {
				c.HP = c.MaxHP
			}
			logs = append(logs, CombatLog{
				Turn:      turn,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    "REGEN_TICK",
				Damage:    heal,
				Detail:    fmt.Sprintf("%s hồi phục %d HP do Hồi Sinh (còn %d HP).", c.Name, heal, c.HP),
			})
		} else {
			c.HP -= int(dot.Value)
			if c.HP < 0 {
				c.HP = 0
			}
			label := ""
			switch dot.Type {
			case EffectBleed:
				label = "Xuất huyết"
			case EffectPoison:
				label = "Độc"
			case EffectSlowDamage:
				label = "Tê Cóng"
			default:
				label = dot.Type
			}
			logs = append(logs, CombatLog{
				Turn:      turn,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    "DOT_TICK",
				Damage:    int(dot.Value),
				Detail:    fmt.Sprintf("%s chịu %.0f %s sát thương (còn %d HP).", c.Name, dot.Value, label, c.HP),
			})
		}
		dot.Turns--
		if dot.Turns > 0 {
			remaining = append(remaining, dot)
		}
	}
	c.ActiveDoT = remaining
	return logs
}

// tickStatusEffects processes buff/debuff durations at the start of a combatant's turn.
// It decrements durations and removes expired effects.
func tickStatusEffects(c *CombatantAuto, turn int) []CombatLog {
	var logs []CombatLog
	var kept []Effect

	for _, eff := range c.ActiveEffects {
		eff.Duration--
		if eff.Duration <= 0 {
			// Effect expired — log removal for notable effects
			switch eff.ID {
			case EffectStun:
				logs = append(logs, CombatLog{
					Turn: turn, ActorId: c.ID, ActorName: c.Name,
					Action: "EFFECT_EXPIRED",
					Detail: fmt.Sprintf("%s hết choáng!", c.Name),
				})
			case EffectFreeze:
				logs = append(logs, CombatLog{
					Turn: turn, ActorId: c.ID, ActorName: c.Name,
					Action: "EFFECT_EXPIRED",
					Detail: fmt.Sprintf("%s hết đóng băng!", c.Name),
				})
			case EffectSilence:
				logs = append(logs, CombatLog{
					Turn: turn, ActorId: c.ID, ActorName: c.Name,
					Action: "EFFECT_EXPIRED",
					Detail: fmt.Sprintf("%s hết bị im lặng!", c.Name),
				})
			case EffectSlow:
				logs = append(logs, CombatLog{
					Turn: turn, ActorId: c.ID, ActorName: c.Name,
					Action: "EFFECT_EXPIRED",
					Detail: fmt.Sprintf("%s hết bị chậm!", c.Name),
				})
			case EffectSlowDamage:
				logs = append(logs, CombatLog{
					Turn: turn, ActorId: c.ID, ActorName: c.Name,
					Action: "EFFECT_EXPIRED",
					Detail: fmt.Sprintf("%s hết bị chậm sát thương!", c.Name),
				})
			case EffectWeak:
				logs = append(logs, CombatLog{
					Turn: turn, ActorId: c.ID, ActorName: c.Name,
					Action: "EFFECT_EXPIRED",
					Detail: fmt.Sprintf("%s hết bị yếu!", c.Name),
				})
			case EffectTaunt:
				logs = append(logs, CombatLog{
					Turn: turn, ActorId: c.ID, ActorName: c.Name,
					Action: "EFFECT_EXPIRED",
					Detail: fmt.Sprintf("%s hết khiêu khích!", c.Name),
				})
			}
			continue
		}
		kept = append(kept, eff)
	}
	c.ActiveEffects = kept
	return logs
}

// ComputeEffectMultipliers calculates derived combat stats from active effects.
// Call this after tickStatusEffects and before action execution.
func ComputeEffectMultipliers(c *CombatantAuto) {
	c.DamageMultiplier = 1.0
	c.DefenseMultiplier = 1.0
	c.ThornsDamage = 0.0
	c.IsSilenced = false
	c.IsFrozen = false
	c.HasTaunt = false
	c.TaunterID = ""

	speedMult := 1.0
	for _, eff := range c.ActiveEffects {
		switch eff.ID {
		case EffectSlow, EffectSlowDamage:
			speedMult *= (1.0 - float64(eff.Value)/100.0)
		case EffectWeak:
			c.DamageMultiplier *= (1.0 - float64(eff.Value)/100.0)
		case EffectFreeze:
			c.IsFrozen = true
		case EffectSilence:
			c.IsSilenced = true
		case EffectThorns:
			c.ThornsDamage += float64(eff.Value) / 100.0 // value is % of damage received
		case EffectRegen:
			// regen is handled via DoT tick, not multiplier
		case EffectAtkUp:
			c.DamageMultiplier *= (1.0 + float64(eff.Value)/100.0)
		case EffectDefUp:
			c.DefenseMultiplier *= (1.0 + float64(eff.Value)/100.0)
		case EffectTaunt:
			c.HasTaunt = true
			c.TaunterID = c.ID
		}
	}
	c.EffectiveSpeed = c.Speed * speedMult
}

// ApplyThornsDamage applies thorns reflect damage from defender to attacker.
// Returns the damage dealt.
func ApplyThornsDamage(defender, attacker *CombatantAuto, damageDealt float64, turn int, logs *[]CombatLog) float64 {
	if defender.ThornsDamage <= 0 || damageDealt <= 0 {
		return 0
	}
	thornsNet := damageDealt * defender.ThornsDamage
	thornsNet = math.Max(1, thornsNet-float64(attacker.BaseDef))
	attacker.HP -= int(thornsNet)
	if attacker.HP < 0 {
		attacker.HP = 0
	}
	*logs = append(*logs, CombatLog{
		Turn:      turn,
		ActorId:   attacker.ID,
		ActorName: attacker.Name,
		Action:    "THORNS_DAMAGE",
		TargetId:  defender.ID,
		Damage:    int(thornsNet),
		Detail:    fmt.Sprintf("Gai nhọn từ %s phản %.0f sát thương lên %s! (còn %d HP)", defender.Name, thornsNet, attacker.Name, attacker.HP),
	})
	return thornsNet
}

// CheckBossPhaseTransitions tracks HP thresholds for bosses and triggers phases.
func CheckBossPhaseTransitions(state *CombatState, boss *CombatantAuto) []CombatLog {
	var logs []CombatLog
	if boss.Kind != "Boss" {
		return logs
	}

	hpPercent := float64(boss.HP) / float64(boss.MaxHP)

	if hpPercent <= 0.70 && hpPercent > 0.30 && state.BossPhase < 1 {
		state.BossPhase = 1
		logs = append(logs, CombatLog{
			ActorId:   boss.ID,
			ActorName: boss.Name,
			Action:    "PHASE_TRANSITION",
			Detail:    fmt.Sprintf("%s bước vào Giai đoạn 2: Triệu hồi bầy tôi tớ!", boss.Name),
		})

		minion1 := &CombatantAuto{
			ID: boss.ID + "_minion1", Name: "Hộ vệ", Kind: "Minion", GridX: 1, GridY: 2,
			MaxHP: boss.MaxHP / 10, HP: boss.MaxHP / 10, BaseDmg: boss.BaseDmg / 5, BaseDef: boss.BaseDef / 5,
		}
		state.Enemies = append(state.Enemies, minion1)
		logs = append(logs, CombatLog{
			ActorId:   minion1.ID,
			ActorName: minion1.Name,
			Action:    "SUMMONED",
			Detail:    fmt.Sprintf("%s xuất hiện trên tiền tuyến!", minion1.Name),
		})
	}

	if hpPercent <= 0.30 && state.BossPhase < 2 {
		state.BossPhase = 2
		boss.BaseDmg *= 3
		boss.BaseDef = 0
		logs = append(logs, CombatLog{
			ActorId:   boss.ID,
			ActorName: boss.Name,
			Action:    "PHASE_TRANSITION",
			Detail:    fmt.Sprintf("%s bước vào Giai đoạn Cuồng Bạo (Enrage)! Sát thương x3, phòng thủ về 0!", boss.Name),
		})
	}

	return logs
}

// CalculateDamage is kept for backward compatibility but no longer used by the engine.
// Use proper defense mitigation via applyDamageToTarget in engine.go.
func CalculateDamage(attacker *CombatantAuto, defender *CombatantAuto, baseDamage int, attackPath string) (int, []CombatLog) {
	netDmg := float64(baseDamage) - float64(defender.BaseDef)
	if netDmg < 1 {
		netDmg = 1
	}

	finalDamage := int(math.Max(1.0, netDmg))

	var logs []CombatLog
	if defender.VirtualShield > 0 {
		if defender.VirtualShield >= finalDamage {
			defender.VirtualShield -= finalDamage
			logs = append(logs, CombatLog{
				ActorId:   defender.ID,
				ActorName: defender.Name,
				Action:    "SHIELD_ABSORB",
				Damage:    finalDamage,
				Detail:    fmt.Sprintf("%s dùng Khiên Ảo hấp thụ %d sát thương (còn %d khiên)", defender.Name, finalDamage, defender.VirtualShield),
			})
			return 0, logs
		} else {
			absorbed := defender.VirtualShield
			finalDamage -= absorbed
			defender.VirtualShield = 0
			logs = append(logs, CombatLog{
				ActorId:   defender.ID,
				ActorName: defender.Name,
				Action:    "SHIELD_BREAK",
				Damage:    absorbed,
				Detail:    fmt.Sprintf("Khiên Ảo của %s bị phá vỡ sau khi chặn %d sát thương!", defender.Name, absorbed),
			})
		}
	}

	return finalDamage, logs
}

// ProcessTriggers is kept for backward compatibility but no longer used by the engine.
func ProcessTriggers(state *CombatState, target *CombatantAuto, triggerType string, rng *rand.Rand) []CombatLog {
	return nil
}
