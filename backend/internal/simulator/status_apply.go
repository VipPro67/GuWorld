package simulator

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// dotType returns the DoTEffect.Type key for a statusfx status ID. Kept
// lowercase to match the existing ActiveDoT keys ("bleed", "poison", "regen").
func dotType(statusID string) string {
	return strings.ToLower(statusID)
}

// collectAllStatusOps gathers both explicit StatusOps and legacy tier fields on a SimAction.
func collectAllStatusOps(a *SimAction) []statusfx.StatusOp {
	if a == nil {
		return nil
	}
	ops := append([]statusfx.StatusOp(nil), a.StatusOps...)
	if a.BleedPerTurn > 0 {
		ops = append(ops, statusfx.StatusOp{ID: statusfx.StatusBleed, Value: a.BleedPerTurn, Turns: 3})
	}
	if a.PoisonPerTurn > 0 {
		ops = append(ops, statusfx.StatusOp{ID: statusfx.StatusPoison, Value: a.PoisonPerTurn, Turns: 3})
	}
	if a.StunTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusStun, a.StunTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.SlowTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusSlow, a.SlowTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.Slow1Tier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusSlow1, a.Slow1Tier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.SlowDamageTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusSlowDamage, a.SlowDamageTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.WeakTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusWeak, a.WeakTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.FreezeTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusFreeze, a.FreezeTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.SilenceTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusSilence, a.SilenceTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.ThornsTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusThorns, a.ThornsTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.RegenTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusRegen, a.RegenTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.AtkUpTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusAtkUp, a.AtkUpTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.DefUpTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusDefUp, a.DefUpTier); err == nil {
			ops = append(ops, op)
		}
	}
	if a.TauntTier > 0 {
		if op, err := statusfx.ResolveStatusWithTier(statusfx.StatusTaunt, a.TauntTier); err == nil {
			ops = append(ops, op)
		}
	}
	return ops
}

// applyStatusOps applies a normalized list of status operations to the actor
// (self-target buffs) and/or each resolved target (debuffs/DoTs). It consults
// the statusfx registry so adding a status needs no new if-chains here.
func applyStatusOps(actor *CombatantAuto, targets []*CombatantAuto, ops []statusfx.StatusOp, guRank int, turn int, rng *rand.Rand, logs *[]CombatLog) {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	if guRank <= 0 && actor != nil {
		guRank = actor.Rank
	}
	if guRank <= 0 {
		guRank = 1
	}
	for _, op := range ops {
		spec, ok := statusfx.Spec(op.ID)
		if !ok {
			continue
		}
		if spec.SelfTarget {
			applyStatusOpTo(actor, actor, spec, op, guRank, turn, rng, logs)
			continue
		}
		for _, target := range targets {
			if target == nil || target.HP <= 0 {
				continue
			}
			applyStatusOpTo(actor, target, spec, op, guRank, turn, rng, logs)
		}
	}
	// Re-compute multipliers after applying status ops so active effects (e.g. INVINCIBLE, STUN, ATK_UP, etc.) take effect immediately.
	ComputeEffectMultipliers(actor)
	for _, target := range targets {
		if target != nil {
			ComputeEffectMultipliers(target)
		}
	}
}

func applyStatusOpTo(actor, target *CombatantAuto, spec statusfx.StatusSpec, op statusfx.StatusOp, guRank int, turn int, rng *rand.Rand, logs *[]CombatLog) {
	if spec.SelfTarget {
		// Self-target proc gate based on Player Rank and Gu Rank (includes Caster Attenuation)
		if !statusfx.CanApplySelfStatus(rng, actor.Rank, guRank, spec, op.Chance) {
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "STATUS_MISS",
				TargetId:  actor.ID,
				Detail:    fmt.Sprintf("[%s] không kích hoạt thành công hiệu ứng [%s] lên bản thân (suy hao cảnh giới)!", actor.Name, spec.Label),
			})
			return
		}
		// Caster Attenuation: when casterRank > guRank, scale down numerical values (shield, heal, stat buffs)
		if actor.Rank > guRank && op.Value > 0 {
			attenuation := statusfx.CasterAttenuationFactor(guRank, actor.Rank)
			op.Value *= attenuation
		}
	} else {
		// Target suppression check based on GuRank vs TargetRank
		suppFactor, durationPenalty := statusfx.TargetSuppressionFactor(guRank, target.Rank)
		if suppFactor <= 0 {
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "STATUS_IMMUNE",
				Detail:    fmt.Sprintf("%s cảnh giới vượt trội, hoàn toàn miễn nhiễm hiệu ứng [%s] từ Cổ Rank %d!", target.Name, spec.Label, guRank),
			})
			return
		}
		if durationPenalty > 0 {
			op.Turns -= durationPenalty
			if op.Turns < 1 {
				op.Turns = 1
			}
		}

		// Opt-in proc gate (apply_status entry chance, rank-scaled at resolve).
		// Rolls before immunity/resist: an untriggered effect leaves no trace.
		if op.Chance > 0 {
			effChance := op.Chance * suppFactor
			if rng.Float64() >= effChance {
				*logs = append(*logs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "STATUS_MISS",
					TargetId:  target.ID,
					Detail:    fmt.Sprintf("[%s] không kích hoạt hiệu ứng [%s] (tỉ lệ %.0f%% bị áp chế)!", actor.Name, spec.Label, effChance*100),
				})
				return
			}
		}
		// Dream immunity (FORGET_DREAM): dream-path statuses never land.
		// Checked before the resist gate — immunity is absolute, not a roll.
		if target.ImmuneDreamStatus && spec.Path == "dream" {
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "STATUS_IMMUNE",
				Detail:    fmt.Sprintf("%s hoàn toàn miễn nhiễm hiệu ứng Mộng đạo [%s]!", target.Name, spec.Label),
			})
			return
		}
		// Nghịch Chuyển Cổ: absolute reversal of hostile effects. Every
		// enemy-sourced status aimed at the holder is undone — only raw
		// damage passes through. Checked before the resist gate.
		if hasReverseCalamity(target) {
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "STATUS_REVERSED",
				Detail:    fmt.Sprintf("Nghịch Chuyển Cổ đảo nghịch [%s] của %s về hư vô!", spec.Label, actor.Name),
			})
			return
		}
		// Resist gate (control effects subject to MentalResistance + generic CCResist + Rank scaling).
		// Combined multiplicatively, capped at 0.75.
		if spec.ResistGate && !statusfx.CanApplyStatusWithGuRank(rng, actor.Rank, target.Rank, guRank, actor.DaoMarks, target.DaoMarks, spec, gamefx.CombinedResist(target.MentalResistance, target.CCResist), op.Chance) {
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "STATUS_RESIST",
				Detail:    fmt.Sprintf("%s kháng lại %s bằng ý chí tinh thần!", target.Name, spec.Label),
			})
			return
		}
	}

	// Tenacity shortens incoming CC duration (min 1 turn). Buffs bypass.
	if !spec.SelfTarget && target.Tenacity > 0 {
		if scale := 1.0 - gamefx.CappedTenacity(target.Tenacity); scale < 1.0 {
			scaled := int(float64(op.Turns) * scale)
			if scaled < 1 {
				scaled = 1
			}
			op.Turns = scaled
		}
	}

	switch spec.Behaviour {
	case statusfx.BehaviourDot:
		val := op.Value
		if val <= 0 {
			val = 10
		}
		target.ActiveDoT = append(target.ActiveDoT, DoTEffect{
			Type:     dotType(op.ID),
			Value:    val,
			Turns:    op.Turns,
			DirectHP: spec.DirectHP,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    op.ID + "_APPLIED",
			Detail:    fmt.Sprintf("%s dính [%s] (%.0f sát thương/lượt trong %d lượt)!", target.Name, spec.Label, val, op.Turns),
		})

	case statusfx.BehaviourRegen:
		healVal := op.Value
		if healVal <= 0 {
			healVal = target.MaxHP * 0.05
		}
		target.ActiveDoT = append(target.ActiveDoT, DoTEffect{
			Type:  "regen",
			Value: healVal,
			Turns: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "REGEN_APPLIED",
			Detail:    fmt.Sprintf("%s nhận [%s] (hồi %.0f HP/lượt trong %d lượt)!", target.Name, spec.Label, healVal, op.Turns),
		})

	case statusfx.BehaviourStun:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "STUN_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], mất lượt trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourFreeze:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "FREEZE_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], mất lượt trong %d lượt (tan khi bị đánh)!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourSilence:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "SILENCE_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], chỉ có thể đánh thường trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourForceBasic: // FEAR
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "FEAR_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], chỉ có thể đánh thường trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourSleep: // SLEEP
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "SLEEP_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], mất lượt trong %d lượt (tỉnh lại khi bị tấn công)!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourBlind: // BLIND
		accPenalty := spec.Accuracy
		if accPenalty <= 0 {
			accPenalty = 0.35
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: int(accPenalty * 100), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "BLIND_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], giảm %.0f%% chính xác trong %d lượt!", target.Name, spec.Label, accPenalty*100, op.Turns),
		})

	case statusfx.BehaviourConfuse: // CONFUSE
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "CONFUSE_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], có thể tấn công nhầm đồng minh trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourCharm: // CHARM
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "CHARM_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], bị mê hoặc trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourBind: // BIND
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "BIND_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], mất lượt trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourWeak:
		pct := op.Value
		if pct <= 0 {
			pct = 20
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: int(pct), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "WEAK_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], giảm %.0f%% sát thương trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourSlow:
		pct := op.Value
		if pct <= 0 {
			pct = 25
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: int(pct), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "SLOW_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], giảm %.0f%% tốc độ trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourAtkUp:
		pct := op.Value
		if pct <= 0 {
			pct = 20
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: int(pct), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "ATK_UP_APPLIED",
			Detail:    fmt.Sprintf("%s nhận [%s], tăng +%.0f%% công kích trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourDefUp:
		pct := op.Value
		if pct <= 0 {
			pct = 20
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: int(pct), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "DEF_UP_APPLIED",
			Detail:    fmt.Sprintf("%s nhận [%s], tăng +%.0f%% phòng thủ trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourThorns:
		pct := op.Value
		if pct <= 0 {
			pct = 20
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: int(pct), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "THORNS_APPLIED",
			Detail:    fmt.Sprintf("%s nhận [%s], phản %.0f%% sát thương khi bị đánh trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourTaunt:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "TAUNT_APPLIED",
			Detail:    fmt.Sprintf("%s kích hoạt [%s], buộc kẻ địch tấn công mình trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourStealth:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "STEALTH_APPLIED",
			Detail:    fmt.Sprintf("%s vào trạng thái [%s], không thể bị chọn mục tiêu trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourHaste:
		pct := op.Value
		if pct <= 0 {
			pct = 20
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: int(pct), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "HASTE_APPLIED",
			Detail:    fmt.Sprintf("%s nhận [%s], tăng %.0f%% tốc độ trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourArmorBreak:
		pct := op.Value
		if pct <= 0 {
			pct = 30
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: int(pct), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "ARMOR_BREAK_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], giảm %.0f%% phòng thủ trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourDamageReduction:
		pct := op.Value
		if pct <= 0 {
			pct = 30
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: int(pct), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "DAMAGE_REDUCTION_APPLIED",
			Detail:    fmt.Sprintf("%s nhận [%s], giảm %.0f%% sát thương nhận vào trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourInvincible:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: 100, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "INVINCIBLE_APPLIED",
			Detail:    fmt.Sprintf("%s nhận [%s], miễn nhiễm mọi sát thương trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourReviveLock:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "REVIVE_LOCK_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], không thể hồi sinh trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourEssenceLeak:
		drain := op.Value
		if drain <= 0 {
			drain = 2.0
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: int(drain), Duration: op.Turns, SourcePath: spec.Path,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "ESSENCE_LEAK_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], tiêu hao chân nguyên mỗi lượt trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourGuSeal:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns, SourcePath: spec.Path,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "GU_SEAL_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], một Cổ trùng bị phong ấn trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourSoulWound:
		damage := op.Value
		if damage <= 0 {
			damage = 50
		}
		if target.VirtualShield > 0 {
			if target.VirtualShield >= damage {
				target.VirtualShield -= damage
			} else {
				damage -= target.VirtualShield
				target.VirtualShield = 0
			}
		}
		target.HP -= damage
		if target.HP < 0 {
			target.HP = 0
		}
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "SOUL_WOUND_APPLIED",
			Damage:    damage,
			Detail:    fmt.Sprintf("%s chịu %.0f sát thương thực từ [%s]!", target.Name, damage, spec.Label),
		})

	case statusfx.BehaviourThoughtChaos:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns, SourcePath: spec.Path,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "THOUGHT_CHAOS_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], sát chiêu có thể phát động thất bại trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourDaoConflict:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns, SourcePath: spec.Path,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "DAO_CONFLICT_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s], hiệu quả nhận hồi phục/hộ thuẫn bị giảm trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourBacklash:
		damage := op.Value
		if damage <= 0 {
			damage = 30
		}
		target.HP -= damage
		if target.HP < 0 {
			target.HP = 0
		}
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "BACKLASH_APPLIED",
			Damage:    damage,
			Detail:    fmt.Sprintf("%s bị [%s], chịu %.0f sát thương phản phệ!", target.Name, spec.Label, damage),
		})

	case statusfx.BehaviourAperture:
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns, SourcePath: spec.Path,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    op.ID + "_APPLIED",
			Detail:    fmt.Sprintf("%s bị [%s] (%d lượt)!", target.Name, spec.Label, op.Turns),
		})

	default:
		effType := "debuff"
		if spec.SelfTarget {
			effType = "buff"
		}
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: effType, Value: int(op.Value), Duration: op.Turns, SourcePath: spec.Path,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    op.ID + "_APPLIED",
			Detail:    fmt.Sprintf("%s nhận hiệu ứng [%s] (%d lượt).", target.Name, spec.Label, op.Turns),
		})
	}
}
