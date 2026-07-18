package simulator_test

import (
	"encoding/json"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestDeterministicSimulator(t *testing.T) {
	loadout := simulator.CombatLoadout{
		GuIds: []string{"GU_BLOOD_MOON"},
	}

	createInitialState := func() *simulator.CombatState {
		return &simulator.CombatState{
			Turn: 0,
			Seed: 12345,
			Allies: []*simulator.CombatantAuto{
				{
					ID:            "player1",
					Name:          "Cổ Sư A",
					Kind:          "Player",
					GridX:         2,
					GridY:         2,
					MaxHP:         100,
					HP:            100,
					BaseDmg:       50,
					BaseDef:       10,
					VirtualShield: 1000,
					DaoMarks:      simulator.DaoMarks{"fire": 10},
					Loadout:       &loadout,
					Actions: []simulator.SimAction{
						{ActionType: "GU_SKILL", Name: "Huyết Nguyệt Cổ", Damage: 80, ManaCost: 10},
						{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: 50, ManaCost: 0},
					},
				},
			},
			Enemies: []*simulator.CombatantAuto{
				{
					ID:      "boss1",
					Name:    "Thái Cổ Hoang Thú",
					Kind:    "Boss",
					GridX: 1,
					GridY: 2,
					MaxHP:   500,
					HP:      500,
					BaseDmg: 100,
					BaseDef: 50,
					DaoMarks: simulator.DaoMarks{},
					Actions: []simulator.SimAction{
						{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: 100, ManaCost: 0},
					},
				},
			},
		}
	}

	state1 := createInitialState()
	logs1 := simulator.Simulate(state1)

	state2 := createInitialState()
	logs2 := simulator.Simulate(state2)

	if len(logs1) != len(logs2) {
		t.Fatalf("Mismatched log lengths: %d vs %d", len(logs1), len(logs2))
	}

	for i := range logs1 {
		j1, _ := json.Marshal(logs1[i])
		j2, _ := json.Marshal(logs2[i])
		if string(j1) != string(j2) {
			t.Fatalf("Mismatched log at index %d:\nLog1: %s\nLog2: %s", i, string(j1), string(j2))
		}
	}
}
