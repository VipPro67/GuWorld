package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// LastStandMinHP is the HP floor Bất Tử (LAST_STAND) holds: damage still
// applies, but the unit cannot be pushed to 0 while the buff is up.
const LastStandMinHP = 1.0

// tenacityPriority ranks which debuff gets dispelled first: hard lock (mất
// lượt) → action lock → disrupt/dao → DoT. Lower is dispelled first.

// applyLastStandFloor keeps target at >= LastStandMinHP while Bất Tử is active
// and Sát Khí is not piercing. It never heals: a unit already below the floor
// when the buff lands is raised to exactly the floor, and a unit already dead
// stays dead. Returns true when the floor was applied so callers can log it.
func applyLastStandFloor(target *CombatantAuto, satKhiPierce bool) bool {
	if target == nil || satKhiPierce || !target.IsLastStand {
		return false
	}
	if target.HP < LastStandMinHP {
		target.HP = LastStandMinHP
		return true
	}
	return false
}

// applyLethalLuckWard provides the Cẩu Thỉ Vận / Tử Địa Phùng Sinh mechanic for Luck Path:
// when lethal damage would push a unit to <= 0 HP, it has a chance to prevent death,
// floor HP at 1, and convert lethal damage into Chân Nguyên Giáp (Virtual Shield).
// Only triggers once per battle.
func applyLethalLuckWard(target *CombatantAuto, logs *[]CombatLog, turn int, rng *rand.Rand) bool {
	if target == nil || target.HP > 0 {
		return false
	}
	if !hasMechanic(target.Mechanics, "BEFORE_HIT", "LETHAL_LUCK_WARD") &&
		!hasMechanic(target.Mechanics, "COMBAT_PASSIVE", "LETHAL_LUCK_WARD") &&
		!hasMechanic(target.Mechanics, "ON_BATTLE_START", "LETHAL_LUCK_WARD") {
		return false
	}
	if target.CustomState == nil {
		target.CustomState = make(map[string]int)
	}
	if target.CustomState["lethal_luck_used"] > 0 {
		return false
	}

	luckChance := 0.25 // Base 25%
	if rng.Float64() < luckChance {
		target.CustomState["lethal_luck_used"] = 1
		target.HP = 1.0
		shieldGain := target.MaxHP * 0.35
		if shieldGain < 100 {
			shieldGain = 100
		}
		target.VirtualShield += shieldGain
		if logs != nil {
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "LETHAL_LUCK_WARD",
				Detail:    fmt.Sprintf("[Vận Đạo - Cẩu Thỉ Vận!] Vận khí ngút trời! %s thoát chết trong gang tấc (HP giữ lại 1, nhận thêm %.0f Chân Nguyên Giáp)!", target.Name, shieldGain),
			})
		}
		return true
	}
	return false
}

// applyNirvanaRebirth provides the Niết Bàn Hỏa (Nirvana Fire) mechanic for Fire Path:
// when lethal damage would push a unit to <= 0 HP, if the unit has the NIRVANA_FIRE mechanic
// (ON_DEATH, COMBAT_PASSIVE, or BEFORE_HIT), it consumes the fire to revive with 40% MaxHP,
// cleanses all debuffs, and retaliates with an AoE fire blast against all enemies.
// Only triggers once per battle.
func applyNirvanaRebirth(state *CombatState, target *CombatantAuto, logs *[]CombatLog, turn int) bool {
	if target == nil || target.HP > 0 {
		return false
	}
	if !actorHasMechanic(target, "ON_DEATH", "NIRVANA_FIRE") &&
		!actorHasMechanic(target, "COMBAT_PASSIVE", "NIRVANA_FIRE") &&
		!actorHasMechanic(target, "BEFORE_HIT", "NIRVANA_FIRE") {
		return false
	}
	if target.CustomState == nil {
		target.CustomState = make(map[string]int)
	}
	if target.CustomState["nirvana_fire_used"] > 0 {
		return false
	}
	target.CustomState["nirvana_fire_used"] = 1

	reviveHP := target.MaxHP * 0.40
	if reviveHP < 20 {
		reviveHP = 20
	}
	target.HP = reviveHP

	// Cleanse debuffs
	var cleanedEffects []Effect
	for _, eff := range target.ActiveEffects {
		if eff.Type != "debuff" {
			cleanedEffects = append(cleanedEffects, eff)
		}
	}
	target.ActiveEffects = cleanedEffects

	if logs != nil {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "NIRVANA_FIRE",
			Detail:    fmt.Sprintf("[Viêm Đạo - Niết Bàn Hỏa] Thân thể bốc cháy ngút trời, %s dục hỏa trùng sinh hồi sinh với %.0f HP và tẩy trừ toàn bộ hiệu ứng bất lợi!", target.Name, reviveHP),
		})
	}

	// Retaliate against enemies if state is provided
	if state != nil {
		isAlly := false
		for _, a := range state.Allies {
			if a.ID == target.ID {
				isAlly = true
				break
			}
		}
		opponents := state.Enemies
		if !isAlly {
			opponents = state.Allies
		}
		blastDmg := target.BaseDmg * 0.80
		if blastDmg < 30 {
			blastDmg = 30
		}
		for _, opp := range opponents {
			if opp != nil && opp.HP > 0 {
				opp.HP -= blastDmg
				if opp.HP < 0 {
					opp.HP = 0
				}
				if logs != nil {
					*logs = append(*logs, CombatLog{
						Turn:      turn,
						ActorId:   target.ID,
						ActorName: target.Name,
						Action:    "NIRVANA_FIRE",
						TargetId:  opp.ID,
						Damage:    blastDmg,
						Detail:    fmt.Sprintf("[Niết Bàn Hỏa] Dư chấn hỏa diễm trùng sinh thiêu đốt %s nhận %.0f sát thương!", opp.Name, blastDmg),
					})
				}
			}
		}
	}
	return true
}

// isImmuneToDamage reports whether Bất Hoại blocks damage entirely. Covers the
// registry buff, the raw "invulnerable" effect CHAOS_GAMBLE grants
// (engine.go:1492), and the legacy DamageTakenReduction == 1 sentinel from
// database rows that predate the IsInvincible field.
func isImmuneToDamage(target *CombatantAuto, satKhiPierce bool) bool {
	if target == nil || satKhiPierce {
		return false
	}
	if target.IsInvincible || target.DamageTakenReduction >= 1.0 {
		return true
	}
	for _, eff := range target.ActiveEffects {
		if strings.EqualFold(eff.ID, "invulnerable") || strings.EqualFold(eff.ID, statusfx.StatusInvincible) {
			return true
		}
	}
	return false
}

// satKhiPiercesImmunity reports whether the actor's Sát Khí is high enough to
// ignore both Bất Hoại and Bất Tử (Tất Sát Cổ, threshold > 10).
func satKhiPiercesImmunity(actor *CombatantAuto) bool {
	return actor != nil && actor.CustomState != nil &&
		hasMechanic(actor.Mechanics, "ON_BATTLE_START", "SAT_KHI_GUARANTEE") &&
		actor.CustomState["sat_khi_level"] > 10
}

// isGuaranteedHit reports whether an action or actor possesses the GUARANTEED_HIT
// mechanic (bypasses evasion, blindness, and first-turn evade).
func isGuaranteedHit(actor *CombatantAuto, a *SimAction) bool {
	if a != nil {
		if hasMechanic(a.Mechanics, "ON_HIT", "GUARANTEED_HIT") ||
			hasMechanic(a.Mechanics, "ON_ACTION", "GUARANTEED_HIT") ||
			hasMechanic(a.Mechanics, "ON_CAST", "GUARANTEED_HIT") ||
			hasMechanic(a.Mechanics, "ON_CAST", "CALCULATED_STRIKE") ||
			hasMechanic(a.Mechanics, "ON_ACTION", "CALCULATED_STRIKE") {
			return true
		}
	}
	if actor != nil {
		if hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "GUARANTEED_HIT") ||
			hasMechanic(actor.Mechanics, "ON_BATTLE_START", "GUARANTEED_HIT") {
			return true
		}
	}
	return false
}

// EvasionChanceCap is the hard ceiling on a combatant's combined dodge chance
// (EVASION status). Stacks merge by max, so this only binds for a single Gu
// granting >= 76% evasion.
const EvasionChanceCap = 0.75

func tenacityPriority(id string) int {
	switch id {
	case statusfx.StatusStun, statusfx.StatusBind, statusfx.StatusFreeze, statusfx.StatusSleep, statusfx.StatusDreamPrison,
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

			// Bất Hoại blocks DoT too — this path never reaches applyDamageToTarget.
			if isImmuneToDamage(c, false) {
				logs = append(logs, CombatLog{
					Turn:      turn,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "DOT_ABSORBED",
					Detail:    fmt.Sprintf("%s miễn nhiễm %s nhờ Bất Hoại!", c.Name, label),
				})
				continue
			}
			dmg := dot.Value
			if dot.DirectHP {
				c.HP -= dmg
				floored := applyLastStandFloor(c, false)
				if c.HP < 0 {
					c.HP = 0
				}
				dotDetail := fmt.Sprintf("%s chịu %.0f %s sát thương trực tiếp (còn %.0f HP).", c.Name, dmg, label, c.HP)
				if floored {
					dotDetail = fmt.Sprintf("%s chịu %.0f %s nhưng [Bất Tử] giữ lại 1 HP!", c.Name, dmg, label)
				}
				logs = append(logs, CombatLog{
					Turn:      turn,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "DOT_TICK",
					Damage:    dmg,
					Detail:    dotDetail,
				})
			} else {
				path := daoPathForAction([]string{dot.SourcePath})
				naturalResist, resonance := targetDaoFactors(c, path)
				dmg = math.Max(0, dmg*resonance*(1-naturalResist)*(1-pathBonus(c.PathResistance, []string{path}))*(1-c.DamageTakenReduction))
				hpDmg := dmg
				initialHP := c.HP
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
				lastStandSaved := false
				if hpDmg > 0 {
					c.HP -= hpDmg
					// Bất Tử absorbed the overflow: nothing spilled to HP.
					if applyLastStandFloor(c, false) {
						hpDmg = 0
						lastStandSaved = true
					}
					if c.HP < 0 {
						c.HP = 0
					}
				}
				if initialHP > c.HP && naturalResist > 0 {
					applyForeignDaoWound(c)
				}

				detail := ""
				if lastStandSaved {
					detail = fmt.Sprintf("%s chịu %.0f %s nhưng [Bất Tử] giữ lại 1 HP!", c.Name, dmg, label)
				} else if shieldDmg > 0 && hpDmg > 0 {
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
					Breakdown: &DamageBreakdown{BaseDamage: dot.Value, DaoMult: dot.SourceDaoMult, TargetDaoResonance: resonance, DaoPath: path, NaturalResist: naturalResist, PathResist: pathBonus(c.PathResistance, []string{path}), DamageReduction: c.DamageTakenReduction, FinalDamage: dmg},
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
	c.IsDreamPrison = false
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
	c.HealingReceivedPenalty = 0.0
	c.IsInvincible = false
	c.IsLastStand = false

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
		case statusfx.BehaviourHealingReceivedReduction:
			if r := float64(eff.Value) / 100.0; r > c.HealingReceivedPenalty {
				c.HealingReceivedPenalty = r
			}
		case statusfx.BehaviourInvincible:
			// Bất Hoại. DamageTakenReduction = 1.0 is kept as the sentinel the
			// damage gate reads; IsInvincible is what the damage paths that never
			// touch applyDamageToTarget check.
			c.DamageTakenReduction = 1.0
			c.IsInvincible = true
		case statusfx.BehaviourLastStand:
			c.IsLastStand = true
		case statusfx.BehaviourEvasion:
			// Merge by max (not sum) so stacking evade buffs never breaks the
			// dodge roll; hard cap 0.75 keeps a guaranteed hit possible.
			if r := float64(eff.Value) / 100.0; r > c.EvasionChance {
				c.EvasionChance = r
			}
			if c.EvasionChance > EvasionChanceCap {
				c.EvasionChance = EvasionChanceCap
			}
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
		case statusfx.BehaviourDreamPrison:
			c.IsDreamPrison = true
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
			case statusfx.StatusBerserk:
				c.DamageMultiplier += 0.35
				c.Tenacity += 0.30
			case statusfx.StatusCalculated:
				c.DamageMultiplier += 0.25
			case statusfx.StatusBloodSacrifice:
				c.DamageMultiplier += 0.50
			case statusfx.StatusBearForm:
				c.DefenseMultiplier *= 1.30
			case statusfx.StatusEagleForm:
				speedMult *= 1.40
				if c.EvasionChance < 0.25 {
					c.EvasionChance = 0.25
				}
			case statusfx.StatusBoneArmor:
				c.ThornsDamage += 0.35
				c.DefenseMultiplier *= 1.20
			case statusfx.StatusAdaptiveResist:
				if c.DamageTakenReduction < 0.25 {
					c.DamageTakenReduction = 0.25
				}
			case statusfx.StatusMetalAegis:
				c.DefenseMultiplier *= 1.35
				if c.DamageTakenReduction < 0.15 {
					c.DamageTakenReduction = 0.15
				}
			case statusfx.StatusCloudMist:
				if c.EvasionChance < 0.35 {
					c.EvasionChance = 0.35
				}
				if c.DamageTakenReduction < 0.20 {
					c.DamageTakenReduction = 0.20
				}
			}
		}
	}

	// BERSERK_STATE (Lực Đạo: Cuồng Man Bá Thể)
	if hasMechanic(c.Mechanics, "COMBAT_PASSIVE", "BERSERK_STATE") {
		if c.HP < c.MaxHP && c.MaxHP > 0 {
			missingPct := (c.MaxHP - c.HP) / c.MaxHP
			c.DamageMultiplier += missingPct * 0.80
			c.Tenacity += missingPct * 0.50
			if missingPct >= 0.70 { // HP < 30%: Enraged kháng choáng/đóng băng
				c.IsStunned = false
				c.IsFrozen = false
			}
		}
	}

	// SOUL_DEVOUR stacks (Hồn Đạo: Phệ Hồn Hóa Lực)
	if c.CustomState != nil && c.CustomState["soul_devour_stacks"] > 0 {
		stacks := c.CustomState["soul_devour_stacks"]
		if stacks > 5 {
			stacks = 5
		}
		c.DamageMultiplier += float64(stacks) * 0.10
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
		if st.ID == statusfx.StatusBleed && spec.Path == "blood" && !isImmuneToDamage(actor, false) {
			burst := float64(st.Value) * float64(st.Duration) * 0.5
			actor.HP -= burst
			applyLastStandFloor(actor, false)
			if actor.HP < 0 {
				actor.HP = 0
			}
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
	if isImmuneToDamage(attacker, false) {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   attacker.ID,
			ActorName: attacker.Name,
			Action:    "THORNS_ABSORBED",
			Detail:    fmt.Sprintf("%s miễn nhiễm %s phản đòn nhờ Bất Hoại!", attacker.Name, defender.Name),
		})
		return 0
	}
	attacker.HP -= thornsNet
	applyLastStandFloor(attacker, false)
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
	finalDamage := ResolveBaseDamage(baseDamage, defender.BaseDef)

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

// ResolveBaseDamage is the shared deterministic defense step used by both the
// turn-based simulator and the server-authoritative world runtime.
func ResolveBaseDamage(baseDamage, defense float64) float64 {
	return math.Max(1, baseDamage-defense)
}

// BasicDamageResult is the deterministic, side-effect-free result of routing
// one ordinary attack through defense, damage reduction, and a virtual shield.
// It is shared by turn-based combat flows.
type BasicDamageResult struct {
	Damage         float64
	HPDamage       float64
	ShieldAbsorbed float64
	HP             float64
	Shield         float64
}

// minHP is the HP floor (Bất Tử / LAST_STAND); pass 0 when no floor applies.
func ResolveBasicDamage(baseDamage, defense, reduction, hp, shield, minHP float64) BasicDamageResult {
	damage := ResolveBaseDamage(baseDamage, defense)
	if reduction < 0 {
		reduction = 0
	}
	if reduction > 1 {
		reduction = 1
	}
	damage *= 1 - reduction
	result := BasicDamageResult{Damage: damage, HP: hp, Shield: math.Max(0, shield)}
	result.ShieldAbsorbed = math.Min(result.Shield, damage)
	result.Shield -= result.ShieldAbsorbed
	result.HPDamage = damage - result.ShieldAbsorbed
	result.HP = math.Max(minHP, hp-result.HPDamage)
	if result.HP < 0 {
		result.HP = 0
	}
	return result
}

// ProcessTriggers is kept for backward compatibility but no longer used by the engine.
func ProcessTriggers(state *CombatState, target *CombatantAuto, triggerType string, rng *rand.Rand) []CombatLog {
	return nil
}
