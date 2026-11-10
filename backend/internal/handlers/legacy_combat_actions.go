package handlers

import (
	"fmt"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// Action represents a dynamic combat action compiled from Gu effects
type Action struct {
	Name           string
	Dmg            float64
	Def            float64
	ManaCost       float64
	IsUtility      bool
	LifeStealRate  float64
	StatusEffects  []models.StatusEffect
	Logs           []string
	GuID           string
	Priority       int
	UseCondition   string
	ConditionValue float64
}

// legacyCombatStatusType maps canonical statusfx IDs to the StatusEffect.Type
// keys ticked by the turn-based combat path. Adding a new status only needs a
// row here plus its statusfx registry spec.
var legacyCombatStatusType = map[string]string{
	statusfx.StatusBleed:         "bleed",
	statusfx.StatusPoison:        "poison",
	statusfx.StatusStun:          "stun",
	statusfx.StatusSlow:          "slow",
	statusfx.StatusSlow1:         "slow1",
	statusfx.StatusSlowDamage:    "slow_damage",
	statusfx.StatusWeak:          "weak",
	statusfx.StatusFreeze:        "freeze",
	statusfx.StatusSilence:       "silence",
	statusfx.StatusThorns:        "thorns_buff",
	statusfx.StatusRegen:         "regen",
	statusfx.StatusAtkUp:         "atk_up",
	statusfx.StatusDefUp:         "def_up",
	statusfx.StatusTaunt:         "taunt",
	statusfx.StatusForgetEgo:     "gu_seal",
	statusfx.StatusForgetReality: "weak",
}

func combatStatusLogLine(name string, op statusfx.StatusOp, value float64) string {
	switch op.ID {
	case statusfx.StatusBleed:
		return fmt.Sprintf("%s gây trạng thái xuất huyết (trừ %.1f HP mỗi lượt).", name, value)
	case statusfx.StatusPoison:
		return fmt.Sprintf("%s gây trạng thái trúng độc (trừ %.1f HP mỗi lượt).", name, value)
	case statusfx.StatusStun:
		return fmt.Sprintf("%s gây choáng (%d lượt).", name, op.Turns)
	case statusfx.StatusSlow:
		return fmt.Sprintf("%s gây chậm (%.0f%% speed, %d lượt).", name, value, op.Turns)
	case statusfx.StatusSlow1:
		return fmt.Sprintf("%s gây chậm (%.0f%% speed, %d lượt, không thể kháng).", name, value, op.Turns)
	case statusfx.StatusSlowDamage:
		return fmt.Sprintf("%s gây chậm sát thương (%.0f%% speed và trừ %.1f HP mỗi lượt, %d lượt).", name, value, value, op.Turns)
	case statusfx.StatusWeak:
		return fmt.Sprintf("%s gây yếu (%.0f%% giảm dmg, %d lượt).", name, value, op.Turns)
	case statusfx.StatusFreeze:
		return fmt.Sprintf("%s gây đóng băng (%d lượt).", name, op.Turns)
	case statusfx.StatusSilence:
		return fmt.Sprintf("%s gây im lặng (%d lượt).", name, op.Turns)
	case statusfx.StatusThorns:
		return fmt.Sprintf("%s nhận Khiên Gai (phản %.0f%% sát thương, %d lượt).", name, value, op.Turns)
	case statusfx.StatusRegen:
		return fmt.Sprintf("%s nhận Hồi Sinh (%.0f%% HP max/lượt, %d lượt).", name, value, op.Turns)
	case statusfx.StatusAtkUp:
		return fmt.Sprintf("%s nhận +%.0f%% Công Kích (%d lượt).", name, value, op.Turns)
	case statusfx.StatusDefUp:
		return fmt.Sprintf("%s nhận +%.0f%% Phòng Thủ (%d lượt).", name, value, op.Turns)
	case statusfx.StatusTaunt:
		return fmt.Sprintf("%s khiêu khích (%d lượt).", name, op.Turns)
	case statusfx.StatusForgetEgo:
		return fmt.Sprintf("%s khiến địch quên bản ngã, phong ấn Cổ (%d lượt).", name, op.Turns)
	case statusfx.StatusForgetReality:
		return fmt.Sprintf("%s gây lãng quên thực tại (giảm %.0f%% sát thương địch, %d lượt).", name, value, op.Turns)
	}
	return ""
}

func BuildCombatAction(ctx *CombatContext, spec models.GuDictionaryItem) (Action, error) {
	dmg, def, cost, err := CalculateGuEffectForCombatant(ctx.Attacker, &spec, true)
	if err != nil {
		return Action{}, err
	}

	action := Action{
		Name:     spec.Name,
		ManaCost: cost,
	}

	if spec.Effects.Active == nil {
		return action, nil
	}

	active := spec.Effects.Active

	if active.DamageFlat > 0 {
		action.Dmg += dmg
	}
	if active.ShieldFlat > 0 {
		action.Def += def
	}

	if active.Range > 0 {
		action.Logs = append(action.Logs, fmt.Sprintf("%s kéo dài khoảng cách tấn công lên %.0f thước.", spec.Name, active.Range))
	}
	if active.CurveAttack {
		action.Logs = append(action.Logs, fmt.Sprintf("%s uốn lượn đường bay tấn công vòng qua phòng ngự trực diện.", spec.Name))
	}
	if active.ManaRegen > 0 {
		ctx.Attacker.CurrentMana += active.ManaRegen
		if ctx.Attacker.CurrentMana > ctx.Attacker.MaxMana {
			ctx.Attacker.CurrentMana = ctx.Attacker.MaxMana
		}
		action.IsUtility = true
		action.Logs = append(action.Logs, fmt.Sprintf("%s hồi phục %.1f Chân Nguyên.", spec.Name, active.ManaRegen))
	}
	if active.WoodTransformation {
		action.Def += 30.0
		action.Logs = append(action.Logs, fmt.Sprintf("%s hóa thân thành yêu tinh cây, tăng thêm phòng ngự (+30.0 Thủ).", spec.Name))
	}
	if active.BloodDrain {
		action.LifeStealRate += 0.5
		action.Logs = append(action.Logs, fmt.Sprintf("%s kích hoạt cơ chế hút máu từ địch.", spec.Name))
	}
	if active.Replicate {
		action.Logs = append(action.Logs, fmt.Sprintf("%s sinh sản nhân bản ma cổ.", spec.Name))
	}
	// Registry-driven statuses: every apply_status entry resolves through
	// statusfx and maps to the StatusEffect types this combat path ticks.
	for _, op := range active.ResolveStatusApplications() {
		legacyType, known := legacyCombatStatusType[op.ID]
		if !known {
			continue
		}
		value := op.Value
		switch op.ID {
		case statusfx.StatusStun, statusfx.StatusFreeze, statusfx.StatusSilence, statusfx.StatusTaunt, statusfx.StatusForgetEgo:
			value = 1
		}
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  legacyType,
			Value: value,
			Turns: op.Turns,
		})
		if line := combatStatusLogLine(spec.Name, op, value); line != "" {
			action.Logs = append(action.Logs, line)
		}
	}
	if active.LifeStealRate > 0 {
		action.LifeStealRate += active.LifeStealRate
	}
	if active.RefineManaTier > 0 {
		boost := active.RefineManaTier * 0.20
		action.Dmg += action.Dmg * boost
		action.IsUtility = true
		action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s nâng phẩm Chân Nguyên (+%.0f%% Sát thương).", spec.Name, active.RefineManaTier*20))
	}
	if active.DamageMult > 1.0 {
		action.Dmg *= active.DamageMult
		action.IsUtility = true
		action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s, tăng uy lực (x%.0f sát thương).", spec.Name, active.DamageMult))
	}
	if active.CleansingPower > 0 {
		action.IsUtility = true
		action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s! Thanh tẩy dị lực và loại bỏ tạp chất trong Không khiếu.", spec.Name))
	}
	if active.SummonPhantom {
		action.Dmg += ctx.Attacker.BaseAttack
		action.Logs = append(action.Logs, fmt.Sprintf("%s triệu hồi thú lực hư ảnh, gây %.1f sát thương lực đạo.", spec.Name, ctx.Attacker.BaseAttack))
	}
	if active.SummonGrowthPhantoms {
		fg := ctx.Attacker.FullyGrown
		if len(fg) > 0 && active.PhantomDamageScale > 0 {
			count := int(active.SummonCountPerTick * float64(spec.Rank))
			if count <= 0 {
				count = 1
			}
			if count > len(fg) {
				count = len(fg)
			}
			totalPhantomDmg := 0.0
			for i := 0; i < count; i++ {
				gu := fg[i]
				baseStat := gu.Attack
				if baseStat <= 0 {
					baseStat = gu.HP * 0.1 // fallback: use HP/10 as base
				}
				dmg := baseStat * active.PhantomDamageScale
				totalPhantomDmg += dmg
				action.Logs = append(action.Logs, fmt.Sprintf("Hư ảnh của %s tấn công gây %.1f sát thương.", gu.GuID, dmg))
			}
			action.Dmg += totalPhantomDmg
			action.Logs = append(action.Logs, fmt.Sprintf("%s triệu hồi %d hư ảnh từ các cổ đã khai thác, tổng sát thương %.1f.", spec.Name, count, totalPhantomDmg))
		}
	}
	if active.GenerateGu {
		action.Logs = append(action.Logs, fmt.Sprintf("%s bắt đầu ngưng tụ Sinh Cơ Diệp.", spec.Name))
	}
	if active.DamageHPPercent > 0 {
		direct := ctx.Defender.MaxHP * active.DamageHPPercent
		action.Dmg += direct
		action.Logs = append(action.Logs, fmt.Sprintf("%s đánh thẳng vào bản ngã, gây thêm %.1f sát thương trực tiếp.", spec.Name, direct))
	}
	if active.ShieldFlatReducePercent > 0 && ctx.Defender.VirtualShield > 0 {
		strip := ctx.Defender.VirtualShield * active.ShieldFlatReducePercent
		ctx.Defender.VirtualShield -= strip
		action.Logs = append(action.Logs, fmt.Sprintf("%s tước %.1f khiên của địch (còn %.1f).", spec.Name, strip, ctx.Defender.VirtualShield))
	}
	if active.DisableGuNextTurn {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "gu_seal",
			Value: 1,
			Turns: 1,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s phong ấn Cổ của địch (không thể dùng Cổ lượt sau).", spec.Name))
	}
	if active.StatBaseAttackFlat > 0 {
		action.Dmg += active.StatBaseAttackFlat
		action.Logs = append(action.Logs, fmt.Sprintf("%s cộng thêm %.1f sát thương.", spec.Name, active.StatBaseAttackFlat))
	}

	return action, nil
}

func ApplyPostDamageEffects(ctx *CombatContext, playerAction *Action, enemyAction *Action, playerHit bool, enemyHit bool, playerNetDmg float64, enemyNetDmg float64) {
	// 1. Lifesteal – heal HP first, excess becomes VirtualShield
	if playerHit && playerNetDmg > 0 && playerAction.LifeStealRate > 0 {
		heal := playerNetDmg * playerAction.LifeStealRate
		hpNeeded := ctx.Attacker.MaxHP - ctx.Attacker.HP
		if heal <= hpNeeded {
			ctx.Attacker.HP += heal
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu hút máu địch, hồi phục %.1f HP.", heal))
		} else {
			ctx.Attacker.HP = ctx.Attacker.MaxHP
			excess := heal - hpNeeded
			ctx.Attacker.VirtualShield += excess
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu hút máu địch (%.1f), HP đầy, chuyển %.1f thành khiên hộ thể.", heal, excess))
		}
	}

	if enemyHit && enemyNetDmg > 0 && enemyAction.LifeStealRate > 0 {
		heal := enemyNetDmg * enemyAction.LifeStealRate
		hpNeeded := ctx.Defender.MaxHP - ctx.Defender.HP
		if heal <= hpNeeded {
			ctx.Defender.HP += heal
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Kẻ địch hút máu đối phương, hồi phục %.1f HP.", heal))
		} else {
			ctx.Defender.HP = ctx.Defender.MaxHP
			excess := heal - hpNeeded
			ctx.Defender.VirtualShield += excess
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Kẻ địch hút máu đối phương (%.1f), HP đầy, chuyển %.1f thành khiên hộ thể.", heal, excess))
		}
	}

	// 2. Status Effects
	if playerHit && playerNetDmg > 0 && len(playerAction.StatusEffects) > 0 {
		ctx.Defender.StatusEffects = append(ctx.Defender.StatusEffects, playerAction.StatusEffects...)
	}

	if enemyHit && enemyNetDmg > 0 && len(enemyAction.StatusEffects) > 0 {
		ctx.Defender.StatusEffects = append(ctx.Defender.StatusEffects, enemyAction.StatusEffects...)
	}
}

// ── GENERIC EVENT DISPATCHER ─────────────────────────────────────────────
