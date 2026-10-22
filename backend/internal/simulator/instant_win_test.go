package simulator

import (
	"strings"
	"testing"
)

// INSTANT_WIN (Huyền Cổ) must end the fight before the enemy's first action,
// bypassing shields, damage immunity and Bất Tử.
func TestInstantWinWipesEnemiesAtBattleStart(t *testing.T) {
	ally := newTestCombatant("p", "Player", 100, 10)
	ally.Mechanics = []CombatMechanic{{Trigger: "ON_BATTLE_START", ActionCode: "INSTANT_WIN"}}

	enemy := newTestCombatant("e", "Enemy", 5000, 999)
	enemy.Speed = 1000
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: 999}}

	logs := Simulate(&CombatState{Seed: 7, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})

	if enemy.HP != 0 || !enemy.IsDead {
		t.Fatalf("enemy HP=%v IsDead=%v, want dead", enemy.HP, enemy.IsDead)
	}
	if ally.HP != 100 {
		t.Fatalf("ally HP=%v, want untouched", ally.HP)
	}
	var won bool
	for _, l := range logs {
		if l.Action == "BATTLE_END" && strings.Contains(l.Detail, "Phe Ta") {
			won = true
		}
	}
	if !won {
		t.Fatalf("expected an ally victory in the battle-end log, got %+v", logs)
	}
}
