package simulator

import (
	"strconv"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

var benchmarkCombatLogs []CombatLog

func BenchmarkSimulate(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		state := benchmarkCombatState()
		benchmarkCombatLogs = Simulate(state)
	}
}

func benchmarkCombatState() *CombatState {
	allies := make([]*CombatantAuto, 3)
	enemies := make([]*CombatantAuto, 3)
	for i := range allies {
		ally := newTestCombatant("ally", "Player", 2000, 45)
		ally.ID += strconv.Itoa(i)
		ally.Actions = []SimAction{
			{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 45},
			{
				ActionType: "GU_SKILL", Name: "Hỏa Diệm", Damage: 60,
				StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusBurn, Value: 20, Turns: 2}},
			},
		}
		allies[i] = ally

		enemy := newTestCombatant("enemy", "Enemy", 2000, 40)
		enemy.ID += strconv.Itoa(i)
		enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 40}}
		enemies[i] = enemy
	}
	return &CombatState{Seed: 42, Allies: allies, Enemies: enemies}
}
