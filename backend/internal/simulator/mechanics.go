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
		c.HP -= int(dot.Value)
		if c.HP < 0 {
			c.HP = 0
		}
		label := ""
		switch dot.Type {
		case "bleed":
			label = "Bleed"
		case "poison":
			label = "Poison"
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
		dot.Turns--
		if dot.Turns > 0 {
			remaining = append(remaining, dot)
		}
	}
	c.ActiveDoT = remaining
	return logs
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
			ID: boss.ID + "_minion1", Name: "Hộ vệ", Kind: "Minion", GridRow: 1,
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
