package simulator

import (
	"fmt"
	"math/rand"
)

// Target selection logic: Simple Frontline-first
func getTarget(enemies []*CombatantAuto) *CombatantAuto {
	// Priority 1: Frontline alive
	for _, e := range enemies {
		if e.GridRow == 1 && e.HP > 0 {
			return e
		}
	}
	// Priority 2: Backline alive
	for _, e := range enemies {
		if e.GridRow == 2 && e.HP > 0 {
			return e
		}
	}
	// Fallback to any alive
	for _, e := range enemies {
		if e.HP > 0 {
			return e
		}
	}
	return nil
}

// Simulate runs the deterministic auto-battle engine until one side is wiped out or turn limit reached.
func Simulate(state *CombatState) []CombatLog {
	var logs []CombatLog
	rng := rand.New(rand.NewSource(state.Seed))

	// Pre-combat Initialization
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
		turnLogs := []CombatLog{}
		turnLogs = append(turnLogs, CombatLog{
			Turn:   turn,
			Action: "TURN_START",
			Detail: fmt.Sprintf("=== LƯỢT %d ===", turn),
		})

		// 4-turn rotation logic
		rotationPhase := turn % 4
		if rotationPhase == 0 {
			rotationPhase = 4
		}

		// Collect all actors to iterate
		var allActors []*CombatantAuto
		allActors = append(allActors, state.Allies...)
		allActors = append(allActors, state.Enemies...)

		// Sort by speed if we had speed, but here we just process Allies then Enemies deterministically
		// Do NOT use map iterations or goroutines here!

		for _, actor := range allActors {
			if actor.HP <= 0 || actor.IsDead {
				continue
			}

			// Pre-action triggers
			turnLogs = append(turnLogs, ProcessTriggers(state, actor, "HP_LOW", rng)...)

			var targets []*CombatantAuto
			var isAlly bool
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
				// No targets left
				continue
			}

			// Decide action based on rotation phase
			actionName := ""
			damageMultiplier := 1.0
			attackPath := "" // E.g. "fire", "water" (simplified)

			switch rotationPhase {
			case 1:
				// Turn 1: Basic attack / Buff
				actionName = "Đánh thường"
				damageMultiplier = 1.0
			case 2:
				// Turn 2: Killer Move 1 (Highest Damage)
				actionName = "Sát Chiêu 1: Hủy Diệt"
				damageMultiplier = 2.5
				attackPath = "fire" // Just an example path for testing Dao Marks reduction
			case 3:
				// Turn 3: Basic attack
				actionName = "Đánh thường"
				damageMultiplier = 1.0
			case 4:
				// Turn 4: Killer Move 2 (CC/Heal)
				actionName = "Sát Chiêu 2: Khống Chế"
				damageMultiplier = 1.5
				// Simplified CC apply logic
				target.ActiveEffects = append(target.ActiveEffects, Effect{ID: "stun", Type: "debuff", Duration: 1})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "CAST_CC",
					TargetId:  target.ID,
					Detail:    fmt.Sprintf("%s tung chiêu Khống Chế lên %s!", actor.Name, target.Name),
				})
			}

			baseDamage := int(float64(actor.BaseDmg) * damageMultiplier)
			
			// If it's fire damage, check target's Thuy Trao Co trigger
			if attackPath == "fire" {
				turnLogs = append(turnLogs, ProcessTriggers(state, target, "FIRE_DAMAGE", rng)...)
			}

			actualDamage, calcLogs := CalculateDamage(actor, target, baseDamage, attackPath)
			turnLogs = append(turnLogs, calcLogs...)

			// Apply fatal blow check
			if target.HP-actualDamage <= 0 {
				fatalLogs := ProcessTriggers(state, target, "FATAL_BLOW", rng)
				turnLogs = append(turnLogs, fatalLogs...)
			}

			if target.HP > 1 || len(ProcessTriggers(state, target, "FATAL_BLOW", rng)) == 0 { // Simple condition
				target.HP -= actualDamage
			}

			turnLogs = append(turnLogs, CombatLog{
				Turn:      turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "ATTACK",
				TargetId:  target.ID,
				Damage:    actualDamage,
				Detail:    fmt.Sprintf("%s dùng [%s] gây %d sát thương lên %s (Còn %d HP)", actor.Name, actionName, actualDamage, target.Name, target.HP),
			})

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

			// Check boss transitions
			if target.Kind == "Boss" {
				bossLogs := CheckBossPhaseTransitions(state, target)
				turnLogs = append(turnLogs, bossLogs...)
			}
		}

		logs = append(logs, turnLogs...)

		// Check Win/Loss Condition
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
