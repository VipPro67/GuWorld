package simulator_test

import (
	"encoding/json"
	"strings"
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

func TestDaoMarksApplyToBothSides(t *testing.T) {
	loadout := simulator.CombatLoadout{GuIds: []string{"GU_TEST"}}
	state := &simulator.CombatState{
		Turn: 0,
		Seed: 999,
		Allies: []*simulator.CombatantAuto{
			{
				ID:            "ally1",
				Name:          "Người Chơi",
				Kind:          "Player",
				GridX:         2,
				GridY:         2,
				MaxHP:         500,
				HP:            100,
				MaxMana:       100,
				Mana:          100,
				BaseDmg:       10,
				BaseDef:       0,
				DaoMarks:      simulator.DaoMarks{"fire": 100},
				Loadout:       &loadout,
				Actions: []simulator.SimAction{
					{ActionType: "GU_SKILL", Name: "Huyết Nguyệt Cổ", Path: []string{"Fire Path"}, Damage: 200, Defense: 50, Healing: 100, ManaCost: 10},
				},
			},
		},
		Enemies: []*simulator.CombatantAuto{
			{
				ID:       "enemy1",
				Name:     "Yêu Quái",
				Kind:     "Normal",
				GridX:    1,
				GridY:    2,
				MaxHP:    2000,
				HP:       2000,
				MaxMana:  100,
				Mana:     100,
				BaseDmg:  10,
				BaseDef:  0,
				DaoMarks: simulator.DaoMarks{"strength": 50},
				Actions: []simulator.SimAction{
					{ActionType: "GU_SKILL", Name: "Lực Cổ", Path: []string{"Strength Path"}, Damage: 100, ManaCost: 10},
				},
			},
		},
	}

	logs := simulator.Simulate(state)

	var allyDaoHit, enemyDaoHit, shieldDao, healDao bool
	for _, l := range logs {
		if l.Action == "GU_SKILL" && l.ActorId == "ally1" && l.TargetId == "enemy1" {
			if strings.Contains(l.Detail, "gây 200 + 200 sát thương do Đạo Ngân") {
				allyDaoHit = true
			}
		}
		if l.Action == "GU_SKILL" && l.ActorId == "enemy1" && l.TargetId == "ally1" {
			if strings.Contains(l.Detail, "gây 100 + 50 sát thương do Đạo Ngân") {
				enemyDaoHit = true
			}
		}
		if l.Action == "SHIELD_GAIN" && l.ActorId == "ally1" && strings.Contains(l.Detail, "50 + 50 Khiên Ảo do Đạo Ngân") {
			shieldDao = true
		}
		if l.Action == "HEAL" && l.ActorId == "ally1" && strings.Contains(l.Detail, "100 + 100 HP do Đạo Ngân") {
			healDao = true
		}
	}
	if !allyDaoHit {
		t.Fatalf("expected ally dao damage log")
	}
	if !enemyDaoHit {
		t.Fatalf("expected enemy dao damage log (both sides must apply dao marks)")
	}
	if !shieldDao {
		t.Fatalf("expected shield dao log")
	}
	if !healDao {
		t.Fatalf("expected heal dao log")
	}
}
