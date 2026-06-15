package simulator

import (
	"fmt"
	"math"
	"math/rand"
)

// CalculateDamage calculates effective damage based on Dao Marks and Virtual Shields.
// Returns the actual HP damage taken.
func CalculateDamage(attacker *CombatantAuto, defender *CombatantAuto, baseDamage int, attackPath string) (int, []CombatLog) {
	var logs []CombatLog

	// 1. Logarithmic Damage Reduction from Dao Marks
	// e.g. Attacker uses Fire (Viêm Đạo), Defender has Water (Thủy Đạo) marks.
	// For simplicity, we check if the defender has the same or countering path marks.
	defenderMarks := 0
	if attackPath != "" && defender.DaoMarks != nil {
		if marks, ok := defender.DaoMarks[attackPath]; ok {
			// Using same path marks for defense as baseline, but counter path is better.
			// Simplification: just use the specific path marks if available to reduce damage.
			defenderMarks = marks
		}
	}

	effectiveDamage := float64(baseDamage)
	if defenderMarks > 0 {
		// Logarithmic reduction formula: e.g. Dmg = Dmg / (1 + log10(marks/10 + 1))
		reductionFactor := 1.0 + math.Log10(float64(defenderMarks)/10.0+1.0)
		effectiveDamage = effectiveDamage / reductionFactor
	}
	
	finalDamage := int(math.Max(1.0, effectiveDamage))

	// 2. Pre-combat Virtual Shield
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

// ProcessTriggers checks for passive triggers and returns any resulting logs.
func ProcessTriggers(state *CombatState, target *CombatantAuto, triggerType string, rng *rand.Rand) []CombatLog {
	var logs []CombatLog

	// Only process if the target is a player/ally with a loadout
	if target.Loadout == nil {
		return logs
	}

	// Helper to check if a specific Gu is equipped (either as Core, Support or Immortal)
	hasGu := func(guID string) bool {
		if target.Loadout.CoreGuId == guID {
			return true
		}
		if target.Loadout.ImmortalGuId != nil && *target.Loadout.ImmortalGuId == guID {
			return true
		}
		for _, id := range target.Loadout.SupportGuIds {
			if id == guID {
				return true
			}
		}
		return false
	}

	switch triggerType {
	case "HP_LOW":
		// Thiết Huyết Cổ: Khi HP < 30% -> Hồi máu/Tạo khiên
		if target.HP > 0 && float64(target.HP)/float64(target.MaxHP) <= 0.3 {
			if hasGu("thiet_huyet_co") { // Using theoretical IDs
				healAmount := int(float64(target.MaxHP) * 0.2)
				target.HP += healAmount
				if target.HP > target.MaxHP {
					target.HP = target.MaxHP
				}
				logs = append(logs, CombatLog{
					ActorId:   target.ID,
					ActorName: target.Name,
					Action:    "TRIGGERED",
					Trigger:   "Thiết Huyết Cổ",
					Detail:    fmt.Sprintf("%s kích hoạt Thiết Huyết Cổ, phục hồi %d HP!", target.Name, healAmount),
				})
			}
		}
	case "FATAL_BLOW":
		// Kim Cang Cổ: Chặn đòn sát thủ (Giữ lại 1 HP)
		if hasGu("kim_cang_co") {
			target.HP = 1
			logs = append(logs, CombatLog{
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "TRIGGERED",
				Trigger:   "Kim Cang Cổ",
				Detail:    fmt.Sprintf("%s bị chí mạng nhưng Kim Cang Cổ kích hoạt, giữ lại 1 HP!", target.Name),
			})
		}
	case "CC_APPLIED":
		// Thanh Tâm Cổ: Giải trừ khống chế
		if hasGu("thanh_tam_co") {
			// Remove debuffs
			var newEffects []Effect
			for _, eff := range target.ActiveEffects {
				if eff.Type != "debuff" {
					newEffects = append(newEffects, eff)
				}
			}
			target.ActiveEffects = newEffects
			logs = append(logs, CombatLog{
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "TRIGGERED",
				Trigger:   "Thanh Tâm Cổ",
				Detail:    fmt.Sprintf("Thanh Tâm Cổ của %s lập tức thanh tẩy mọi trạng thái bất lợi!", target.Name),
			})
		}
	case "FIRE_DAMAGE":
		// Thủy Tráo Cổ: Tỷ lệ 30% chặn sát thương hỏa
		if hasGu("thuy_trao_co") {
			if rng.Float64() <= 0.30 {
				target.VirtualShield += 5000 // Add temporary shield
				logs = append(logs, CombatLog{
					ActorId:   target.ID,
					ActorName: target.Name,
					Action:    "TRIGGERED",
					Trigger:   "Thủy Tráo Cổ",
					Detail:    fmt.Sprintf("%s kích hoạt Thủy Tráo Cổ, tạo 5000 Khiên Thủy Đạo chặn lửa!", target.Name),
				})
			}
		}
	}

	return logs
}

// CheckBossPhaseTransitions tracks HP threshold for bosses and triggers phases
func CheckBossPhaseTransitions(state *CombatState, boss *CombatantAuto) []CombatLog {
	var logs []CombatLog
	if boss.Kind != "Boss" {
		return logs
	}

	hpPercent := float64(boss.HP) / float64(boss.MaxHP)

	// 70% to 30% - Summoning phase
	if hpPercent <= 0.70 && hpPercent > 0.30 && state.BossPhase < 1 {
		state.BossPhase = 1
		logs = append(logs, CombatLog{
			ActorId:   boss.ID,
			ActorName: boss.Name,
			Action:    "PHASE_TRANSITION",
			Detail:    fmt.Sprintf("%s bước vào Giai đoạn 2: Triệu hồi bầy tôi tớ!", boss.Name),
		})
		
		// In a real scenario, we'd append minion CombatantAutos to state.Enemies here
		minion1 := &CombatantAuto{
			ID: boss.ID + "_minion1", Name: "Hộ vệ", Kind: "Minion", GridRow: 1, MaxHP: boss.MaxHP / 10, HP: boss.MaxHP / 10, BaseDmg: boss.BaseDmg / 5, BaseDef: boss.BaseDef / 5,
		}
		state.Enemies = append(state.Enemies, minion1)
		logs = append(logs, CombatLog{
			ActorId:   minion1.ID,
			ActorName: minion1.Name,
			Action:    "SUMMONED",
			Detail:    fmt.Sprintf("%s xuất hiện trên tiền tuyến!", minion1.Name),
		})
	}

	// Below 30% - Enrage phase
	if hpPercent <= 0.30 && state.BossPhase < 2 {
		state.BossPhase = 2
		boss.BaseDmg *= 3 // Triple damage
		boss.BaseDef = 0  // Drops defense
		logs = append(logs, CombatLog{
			ActorId:   boss.ID,
			ActorName: boss.Name,
			Action:    "PHASE_TRANSITION",
			Detail:    fmt.Sprintf("%s bước vào Giai đoạn Cuồng Bạo (Enrage)! Sát thương x3, phòng thủ về 0!", boss.Name),
		})
	}

	return logs
}
