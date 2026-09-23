package simulator

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// tenacityPriority ranks which debuff gets dispelled first: hard lock (mất
// lượt) → action lock → disrupt/dao → DoT. Lower is dispelled first.
func tenacityPriority(id string) int {
	switch id {
	case statusfx.StatusStun, statusfx.StatusBind, statusfx.StatusFreeze, statusfx.StatusSleep,
		EffectStun, EffectFreeze:
		return 0
	case statusfx.StatusSilence, statusfx.StatusFear,
		EffectSilence:
		return 1
	default:
		return 2
	}
}

// isDispellableEffect reports whether an ActiveEffects entry is a debuff
// (anything that is not a self buff). Unknown IDs default to dispellable so
// new debuffs are covered without code changes.
func isDispellableEffect(id string) bool {
	if spec, ok := statusfx.Spec(id); ok {
		return spec.Category != statusfx.CategoryBuff
	}
	switch id {
	case EffectThorns, EffectRegen, EffectAtkUp, EffectDefUp, EffectTaunt:
		return false
	default:
		return true
	}
}

// TryTenacityDispel rolls the combatant's generic Tenacity at turn start and
// removes at most one debuff (ActiveEffects first by priority, else one DoT).
// Tenacity aggregates from Gu passive / race / constitution / status — the
// engine never branches on a Gu ID. Returns log entries (nil when no roll win).
func TryTenacityDispel(c *CombatantAuto, turn int, rng *rand.Rand) []CombatLog {
	if c == nil || c.HP <= 0 || c.Tenacity <= 0 || rng == nil {
		return nil
	}
	best := -1
	for i, eff := range c.ActiveEffects {
		if !isDispellableEffect(eff.ID) {
			continue
		}
		if best == -1 || tenacityPriority(eff.ID) < tenacityPriority(c.ActiveEffects[best].ID) {
			best = i
		}
	}
	hasDoT := false
	for _, dot := range c.ActiveDoT {
		if dot.Type != "regen" && dot.Turns > 0 {
			hasDoT = true
			break
		}
	}
	if best == -1 && !hasDoT {
		return nil
	}
	if rng.Float64() >= gamefx.CappedTenacity(c.Tenacity) {
		return nil
	}
	if best != -1 {
		removed := c.ActiveEffects[best]
		label := removed.ID
		if spec, ok := statusfx.Spec(removed.ID); ok {
			label = spec.Label
		}
		c.ActiveEffects = append(c.ActiveEffects[:best], c.ActiveEffects[best+1:]...)
		return []CombatLog{{
			Turn: turn, ActorId: c.ID, ActorName: c.Name,
			Action: "TENACITY_DISPEL",
			Detail: fmt.Sprintf("%s dùng ý chí (Tenacity) tự giải [%s]!", c.Name, label),
		}}
	}
	for i, dot := range c.ActiveDoT {
		if dot.Type != "regen" && dot.Turns > 0 {
			label := dot.Type
			c.ActiveDoT = append(c.ActiveDoT[:i], c.ActiveDoT[i+1:]...)
			return []CombatLog{{
				Turn: turn, ActorId: c.ID, ActorName: c.Name,
				Action: "TENACITY_DISPEL",
				Detail: fmt.Sprintf("%s dùng ý chí (Tenacity) tự giải [%s]!", c.Name, label),
			}}
		}
	}
	return nil
}

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

			dmg := dot.Value
			if dot.DirectHP {
				c.HP -= dmg
				if c.HP < 0 {
					c.HP = 0
				}
				logs = append(logs, CombatLog{
					Turn:      turn,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "DOT_TICK",
					Damage:    dmg,
					Detail:    fmt.Sprintf("%s chịu %.0f %s sát thương trực tiếp (còn %.0f HP).", c.Name, dmg, label, c.HP),
				})
			} else {
				hpDmg := dmg
				shieldDmg := 0.0
				if c.VirtualShield > 0 {
					if c.VirtualShield >= dmg {
						shieldDmg = dmg
						c.VirtualShield -= dmg
						hpDmg = 0
					} else {
						shieldDmg = c.VirtualShield
						hpDmg = dmg - c.VirtualShield
						c.VirtualShield = 0
					}
				}
				if hpDmg > 0 {
					c.HP -= hpDmg
					if c.HP < 0 {
						c.HP = 0
					}
				}

				detail := ""
				if shieldDmg > 0 && hpDmg > 0 {
					detail = fmt.Sprintf("%s chịu %.0f %s sát thương (khiên trừ %.0f, còn %.0f khiên, %.0f HP).", c.Name, dmg, label, shieldDmg, c.VirtualShield, c.HP)
				} else if shieldDmg > 0 {
					detail = fmt.Sprintf("%s chịu %.0f %s sát thương bị khiên hấp thụ (còn %.0f khiên, %.0f HP).", c.Name, dmg, label, c.VirtualShield, c.HP)
				} else {
					detail = fmt.Sprintf("%s chịu %.0f %s sát thương (còn %.0f HP).", c.Name, dmg, label, c.HP)
				}

				logs = append(logs, CombatLog{
					Turn:      turn,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "DOT_TICK",
					Damage:    dmg,
					Detail:    detail,
				})
			}
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
	c.IsStunned = false
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
	c.EvasionChance = 0.0
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
		case statusfx.BehaviourStun:
			c.IsStunned = true
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
			case EffectStun, statusfx.StatusStun:
				c.IsStunned = true
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
			case "EVASION":
				if r := float64(eff.Value) / 100.0; r > c.EvasionChance {
					c.EvasionChance = r
				}
				if c.EvasionChance > 0.75 {
					c.EvasionChance = 0.75
				}
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
func CheckBossPhaseTransitions(state *CombatState, boss *CombatantAuto, turn int) []CombatLog {
	var logs []CombatLog
	if boss.Kind != "Boss" || boss.HP <= 0 || boss.IsDead {
		return logs
	}

	hpPercent := boss.HP / boss.MaxHP

	if hpPercent <= 0.70 && hpPercent > 0.30 && state.BossPhase < 1 {
		state.BossPhase = 1
		logs = append(logs, CombatLog{
			Turn:      turn,
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
			Turn:      turn,
			ActorId:   minion1.ID,
			ActorName: minion1.Name,
			Action:    "SUMMONED",
			Detail:    fmt.Sprintf("%s xuất hiện trên tiền tuyến!", minion1.Name),
		})
	}

	if hpPercent <= 0.30 && hpPercent > 0 && state.BossPhase < 2 {
		state.BossPhase = 2
		boss.BaseDmg *= 3
		boss.BaseDef = 0
		logs = append(logs, CombatLog{
			Turn:      turn,
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
