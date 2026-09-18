package simulator

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
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
			heal := dot.Value
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
				Detail:    fmt.Sprintf("%s hồi phục %.0f HP do Hồi Sinh (còn %.0f HP).", c.Name, heal, c.HP),
			})
		} else {
			c.HP -= dot.Value
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
			case "burn":
				label = "Thiêu Đốt"
			default:
				label = dot.Type
			}
			logs = append(logs, CombatLog{
				Turn:      turn,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    "DOT_TICK",
				Damage:    dot.Value,
				Detail:    fmt.Sprintf("%s chịu %.0f %s sát thương (còn %.0f HP).", c.Name, dot.Value, label, c.HP),
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
			// Effect expired — log removal with its label
			label := eff.ID
			if spec, ok := statusfx.Spec(eff.ID); ok {
				label = spec.Label
			}
			logs = append(logs, CombatLog{
				Turn: turn, ActorId: c.ID, ActorName: c.Name,
				Action: "EFFECT_EXPIRED",
				Detail: fmt.Sprintf("%s hết hiệu ứng [%s]!", c.Name, label),
			})
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
	c.IsSleeping = false
	c.IsFeared = false
	c.IsCharmed = false
	c.IsStealthed = false
	c.IsConfused = false
	c.IsBlinded = false
	c.IsBound = false
	c.IsReviveLocked = false
	c.AccuracyPenalty = 0.0
	c.DamageTakenReduction = 0.0

	// Reset new flags
	c.IsEssenceLeaking = false
	c.HasGuSeal = false
	c.HasSoulWound = false
	c.HasThoughtChaos = false
	c.HasDaoConflict = false

	speedMult := 1.0
	for _, eff := range c.ActiveEffects {
		spec, ok := statusfx.Spec(eff.ID)
		behaviour := ""
		if ok {
			behaviour = spec.Behaviour
		}

		switch behaviour {
		case statusfx.BehaviourSlow:
			speedMult *= (1.0 - float64(eff.Value)/100.0)
		case statusfx.BehaviourHaste:
			speedMult *= (1.0 + float64(eff.Value)/100.0)
		case statusfx.BehaviourWeak:
			c.DamageMultiplier *= (1.0 - float64(eff.Value)/100.0)
		case statusfx.BehaviourAtkUp:
			c.DamageMultiplier *= (1.0 + float64(eff.Value)/100.0)
		case statusfx.BehaviourDefUp:
			c.DefenseMultiplier *= (1.0 + float64(eff.Value)/100.0)
		case statusfx.BehaviourArmorBreak:
			c.DefenseMultiplier *= (1.0 - float64(eff.Value)/100.0)
		case statusfx.BehaviourThorns:
			c.ThornsDamage += float64(eff.Value) / 100.0
		case statusfx.BehaviourDamageReduction:
			if r := float64(eff.Value) / 100.0; r > c.DamageTakenReduction {
				c.DamageTakenReduction = r
			}
		case statusfx.BehaviourInvincible:
			c.DamageTakenReduction = 1.0
		case statusfx.BehaviourFreeze:
			c.IsFrozen = true
		case statusfx.BehaviourSilence:
			c.IsSilenced = true
		case statusfx.BehaviourForceBasic:
			c.IsFeared = true
		case statusfx.BehaviourSleep:
			c.IsSleeping = true
		case statusfx.BehaviourCharm:
			c.IsCharmed = true
		case statusfx.BehaviourStealth:
			c.IsStealthed = true
		case statusfx.BehaviourConfuse:
			c.IsConfused = true
		case statusfx.BehaviourBlind:
			c.IsBlinded = true
			c.AccuracyPenalty = float64(eff.Value) / 100.0
		case statusfx.BehaviourBind:
			c.IsBound = true
		case statusfx.BehaviourTaunt:
			c.HasTaunt = true
			c.TaunterID = c.ID
		case statusfx.BehaviourReviveLock:
			c.IsReviveLocked = true
		case statusfx.BehaviourEssenceLeak:
			c.IsEssenceLeaking = true
		case statusfx.BehaviourGuSeal:
			c.HasGuSeal = true
		case statusfx.BehaviourSoulWound:
			c.HasSoulWound = true
		case statusfx.BehaviourThoughtChaos:
			c.HasThoughtChaos = true
		case statusfx.BehaviourDaoConflict:
			c.HasDaoConflict = true
		default:
			switch eff.ID {
			case EffectSlow, EffectSlow1, EffectSlowDamage:
				speedMult *= (1.0 - float64(eff.Value)/100.0)
			case EffectWeak:
				c.DamageMultiplier *= (1.0 - float64(eff.Value)/100.0)
			case EffectFreeze:
				c.IsFrozen = true
			case EffectSilence:
				c.IsSilenced = true
			case EffectThorns:
				c.ThornsDamage += float64(eff.Value) / 100.0
			case EffectAtkUp:
				c.DamageMultiplier *= (1.0 + float64(eff.Value)/100.0)
			case EffectDefUp:
				c.DefenseMultiplier *= (1.0 + float64(eff.Value)/100.0)
			case EffectTaunt:
				c.HasTaunt = true
				c.TaunterID = c.ID
			}
		}
	}
	c.EffectiveSpeed = c.Speed * speedMult
}

// tickResourceDrain handles resource-drain effects such as ESSENCE_LEAK.
func tickResourceDrain(c *CombatantAuto, turn int) []CombatLog {
	var logs []CombatLog
	if c.IsEssenceLeaking {
		// Example: drain 5% of max mana per turn.
		drain := c.MaxMana * 0.05
		if drain > c.Mana {
			drain = c.Mana
		}
		c.Mana -= drain
		logs = append(logs, CombatLog{Turn: turn, ActorId: c.ID, ActorName: c.Name, Action: "ESSENCE_LEAK", Detail: fmt.Sprintf("%s mất %.0f Chân Nguyên do Rò Rỉ.", c.Name, drain)})
	}
	return logs
}

// DispelActorStatuses removes statuses of a given category from an actor.
// maxCount limits how many statuses can be removed; 0 = unlimited.
func DispelActorStatuses(actor *CombatantAuto, category statusfx.StatusCategory, maxCount int) []string {
	cleansed := []string{}
	remaining := []Effect{}
	for _, st := range actor.ActiveEffects {
		spec, ok := statusfx.Spec(st.ID)
		if !ok || spec.Category != category {
			remaining = append(remaining, st)
			continue
		}
		if maxCount > 0 && len(cleansed) >= maxCount {
			remaining = append(remaining, st)
			continue
		}
		// Trigger cleanse trap for BLEED from Blood path.
		if st.ID == statusfx.StatusBleed && spec.Path == "blood" {
			burst := float64(st.Value) * float64(st.Duration) * 0.5
			actor.HP -= burst
		}
		cleansed = append(cleansed, st.ID)
	}
	actor.ActiveEffects = remaining
	return cleansed
}

// ApplyThornsDamage applies thorns reflect damage from defender to attacker.
// Returns the damage dealt.
func ApplyThornsDamage(defender, attacker *CombatantAuto, damageDealt float64, turn int, logs *[]CombatLog) float64 {
	if defender.ThornsDamage <= 0 || damageDealt <= 0 {
		return 0
	}
	thornsNet := damageDealt * defender.ThornsDamage
	thornsNet = math.Max(1, thornsNet-attacker.BaseDef)
	attacker.HP -= thornsNet
	if attacker.HP < 0 {
		attacker.HP = 0
	}
	*logs = append(*logs, CombatLog{
		Turn:      turn,
		ActorId:   attacker.ID,
		ActorName: attacker.Name,
		Action:    "THORNS_DAMAGE",
		TargetId:  defender.ID,
		Damage:    thornsNet,
		Detail:    fmt.Sprintf("Gai nhọn từ %s phản %.0f sát thương lên %s! (còn %.0f HP)", defender.Name, thornsNet, attacker.Name, attacker.HP),
	})
	return thornsNet
}

// CheckBossPhaseTransitions tracks HP thresholds for bosses and triggers phases.
func CheckBossPhaseTransitions(state *CombatState, boss *CombatantAuto) []CombatLog {
	var logs []CombatLog
	if boss.Kind != "Boss" {
		return logs
	}

	hpPercent := boss.HP / boss.MaxHP

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
func CalculateDamage(attacker *CombatantAuto, defender *CombatantAuto, baseDamage float64, attackPath string) (float64, []CombatLog) {
	netDmg := baseDamage - defender.BaseDef
	if netDmg < 1 {
		netDmg = 1
	}

	finalDamage := math.Max(1.0, netDmg)

	var logs []CombatLog
	if defender.VirtualShield > 0 {
		if defender.VirtualShield >= finalDamage {
			defender.VirtualShield -= finalDamage
			logs = append(logs, CombatLog{
				ActorId:   defender.ID,
				ActorName: defender.Name,
				Action:    "SHIELD_ABSORB",
				Damage:    finalDamage,
				Detail:    fmt.Sprintf("%s dùng Khiên Ảo hấp thụ %.0f sát thương (còn %.0f khiên)", defender.Name, finalDamage, defender.VirtualShield),
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
				Detail:    fmt.Sprintf("Khiên Ảo của %s bị phá vỡ sau khi chặn %.0f sát thương!", defender.Name, absorbed),
			})
		}
	}

	return finalDamage, logs
}

// ProcessTriggers is kept for backward compatibility but no longer used by the engine.
func ProcessTriggers(state *CombatState, target *CombatantAuto, triggerType string, rng *rand.Rand) []CombatLog {
	return nil
}
