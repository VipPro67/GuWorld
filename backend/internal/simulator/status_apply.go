package simulator

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// dotType returns the DoTEffect.Type key for a statusfx status ID. Kept
// lowercase to match the existing ActiveDoT keys ("bleed", "poison", "regen").
func dotType(statusID string) string {
	return strings.ToLower(statusID)
}

// applyStatusOps applies a normalized list of status operations to the actor
// (self-target buffs) and/or each resolved target (debuffs/DoTs). It consults
// the statusfx registry so adding a status needs no new if-chains here.
func applyStatusOps(actor *CombatantAuto, targets []*CombatantAuto, ops []statusfx.StatusOp, turn int, rng *rand.Rand, logs *[]CombatLog) {
	for _, op := range ops {
		spec, ok := statusfx.Spec(op.ID)
		if !ok {
			continue
		}
		if spec.SelfTarget {
			applyStatusOpTo(actor, actor, spec, op, turn, rng, logs)
			continue
		}
		for _, target := range targets {
			if target == nil || target.HP <= 0 {
				continue
			}
			applyStatusOpTo(actor, target, spec, op, turn, rng, logs)
		}
	}
}

func applyStatusOpTo(actor, target *CombatantAuto, spec statusfx.StatusSpec, op statusfx.StatusOp, turn int, rng *rand.Rand, logs *[]CombatLog) {
	// Resist gate (control effects subject to MentalResistance) using advanced proc formula.
	if spec.ResistGate && !statusfx.CanApplyStatus(rng, actor.Rank, target.Rank, actor.DaoMarks, target.DaoMarks, spec, target.MentalResistance) {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "STATUS_RESIST",
			Detail:    fmt.Sprintf("%s kháng lại %s bằng ý chí tinh thần!", target.Name, spec.Label),
		})
		return
	}

	switch spec.Behaviour {
	case statusfx.BehaviourDot:
		target.ActiveDoT = append(target.ActiveDoT, DoTEffect{
			Type:  dotType(op.ID),
			Value: op.Value,
			Turns: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    op.ID + "_APPLIED",
			Detail:    fmt.Sprintf("%s dùng [%s] → %s bị %s (%.0f/lượt trong %d lượt)!", actor.Name, "", target.Name, spec.Label, op.Value, op.Turns),
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
			Detail:    fmt.Sprintf("%s bị %s, chỉ còn tấn công cơ bản trong %d lượt!", target.Name, spec.Label, op.Turns),
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
			Detail:    fmt.Sprintf("%s bị %s, mất lượt trong %d lượt (tỉnh lại khi bị tấn công)!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourBlind: // BLIND
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "debuff", Value: int(spec.Accuracy * 100), Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "BLIND_APPLIED",
			Detail:    fmt.Sprintf("%s bị %s, giảm %.0f%% chính xác trong %d lượt!", target.Name, spec.Label, spec.Accuracy*100, op.Turns),
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
			Detail:    fmt.Sprintf("%s bị %s, có thể tấn công nhầm đồng minh trong %d lượt!", target.Name, spec.Label, op.Turns),
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
			Detail:    fmt.Sprintf("%s bị %s, bị điều khiển trong %d lượt!", target.Name, spec.Label, op.Turns),
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
			Detail:    fmt.Sprintf("%s bị %s, mất lượt trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourStealth: // STEALTH (self buff)
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: 1, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "STEALTH_APPLIED",
			Detail:    fmt.Sprintf("%s vào trạng thái %s, không thể bị chọn mục tiêu trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

	case statusfx.BehaviourHaste: // HASTE (self buff, speed up)
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
			Detail:    fmt.Sprintf("%s nhận %s, tăng %.0f%% tốc độ trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourArmorBreak: // ARMOR_BREAK (debuff, defense down)
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
			Detail:    fmt.Sprintf("%s bị %s, giảm %.0f%% phòng thủ trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourDamageReduction: // DAMAGE_REDUCTION_UP (self buff)
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
			Detail:    fmt.Sprintf("%s nhận %s, giảm %.0f%% sát thương nhận vào trong %d lượt!", target.Name, spec.Label, pct, op.Turns),
		})

	case statusfx.BehaviourInvincible: // INVINCIBLE (self buff, immune)
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID: spec.ID, Type: "buff", Value: 100, Duration: op.Turns,
		})
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "INVINCIBLE_APPLIED",
			Detail:    fmt.Sprintf("%s nhận %s, miễn nhiễm mọi sát thương trong %d lượt!", target.Name, spec.Label, op.Turns),
		})

case statusfx.BehaviourReviveLock: // REVIVE_LOCK (debuff, prevent revival)
	target.ActiveEffects = append(target.ActiveEffects, Effect{
		ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns,
	})
	*logs = append(*logs, CombatLog{
		Turn:      turn,
		ActorId:   target.ID,
		ActorName: target.Name,
		Action:    "REVIVE_LOCK_APPLIED",
		Detail:    fmt.Sprintf("%s bị %s, không thể hồi sinh trong %d lượt!", target.Name, spec.Label, op.Turns),
	})

	case statusfx.BehaviourAperture:
		// Aperture related effects: ESSENCE_LEAK, GU_SEAL, THOUGHT_CHAOS
		target.ActiveEffects = append(target.ActiveEffects, Effect{ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns, SourcePath: spec.Path})
		*logs = append(*logs, CombatLog{Turn: turn, ActorId: target.ID, ActorName: target.Name, Action: op.ID + "_APPLIED", Detail: fmt.Sprintf("%s bị %s (%d lượt)", target.Name, spec.Label, op.Turns)})

	case statusfx.BehaviourSoulWound:
		// Apply true damage now (ignores shield/armor). Value is damage amount.
		damage := op.Value
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
		*logs = append(*logs, CombatLog{Turn: turn, ActorId: target.ID, ActorName: target.Name, Action: "SOUL_WOUND", Damage: damage, Detail: fmt.Sprintf("%s chịu %.0f sát thương thực (SOUL_WOUND)", target.Name, damage)})

	case statusfx.BehaviourThoughtChaos:
		// Simply set flag via Effect entry; resolution handled in ComputeEffectMultipliers.
		target.ActiveEffects = append(target.ActiveEffects, Effect{ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns, SourcePath: spec.Path})
		*logs = append(*logs, CombatLog{Turn: turn, ActorId: target.ID, ActorName: target.Name, Action: "THOUGHT_CHAOS_APPLIED", Detail: fmt.Sprintf("%s bị %s (%d lượt)", target.Name, spec.Label, op.Turns)})

	case statusfx.BehaviourDaoConflict:
		// Apply as debuff; flag used in damage calculations.
		target.ActiveEffects = append(target.ActiveEffects, Effect{ID: spec.ID, Type: "debuff", Value: 1, Duration: op.Turns, SourcePath: spec.Path})
		*logs = append(*logs, CombatLog{Turn: turn, ActorId: target.ID, ActorName: target.Name, Action: "DAO_CONFLICT_APPLIED", Detail: fmt.Sprintf("%s bị %s (%d lượt)", target.Name, spec.Label, op.Turns)})

	case statusfx.BehaviourBacklash:
		// Immediate backlash damage to target.
		damage := op.Value
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
		*logs = append(*logs, CombatLog{Turn: turn, ActorId: target.ID, ActorName: target.Name, Action: "BACKLASH_APPLIED", Damage: damage, Detail: fmt.Sprintf("%s chịu phản phệ %.0f sát thương", target.Name, damage)})

	default:
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "STATUS_UNKNOWN",
			Detail:    fmt.Sprintf("Trạng thái %s chưa được xử lý.", spec.Label),
		})
	}
}
