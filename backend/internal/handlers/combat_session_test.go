package handlers

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestCombatSessionSideProjectsUniqueActionsAndGuDetails(t *testing.T) {
	side := combatSessionSide([]*simulator.CombatantAuto{{
		ID:   "player",
		Name: "Player",
		Actions: []simulator.SimAction{
			{ActionType: "BASIC_ATTACK", Name: "Đánh thường"},
			{ActionType: "GU_SKILL", SourceID: "gu-fire", InstanceID: "instance-fire", Name: "Hỏa Cổ", Rank: 3, ManaCost: 12, Damage: 40, Path: []string{"Hỏa Đạo"}},
		},
	}}, "player", nil)
	actions, ok := side[0]["actions"].([]gin.H)
	if !ok {
		t.Fatal("pending combatant actions were not returned")
	}

	basicAttacks := 0
	var guAction gin.H
	for _, action := range actions {
		switch action["action_type"] {
		case "BASIC_ATTACK":
			basicAttacks++
		case "GU_SKILL":
			guAction = action
		}
	}
	if basicAttacks != 1 {
		t.Fatalf("got %d basic attacks, want 1", basicAttacks)
	}
	if guAction == nil || guAction["source_id"] != "gu-fire" || guAction["instance_id"] != "instance-fire" ||
		guAction["rank"] != 3 || guAction["mana_cost"] != float64(12) || guAction["damage"] != float64(40) {
		t.Fatalf("Gu action details were not projected to the client: %#v", guAction)
	}
}

func TestCombatSessionActionsExposeTargetsAllowedByFormation(t *testing.T) {
	player := &simulator.CombatantAuto{
		ID: "player", Kind: "Player", HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100, BaseDmg: 10,
		Actions: []simulator.SimAction{
			{ActionType: "BASIC_ATTACK", Name: "Đánh thường", TargetType: "SINGLE"},
			{ActionType: "GU_SKILL", SourceID: "gu-bypass", InstanceID: "owned-bypass", Name: "Xuyên Hàng", TargetType: "SINGLE", Bypass: true},
		},
	}
	front := &simulator.CombatantAuto{ID: "front", Name: "Hàng trước", HP: 100, MaxHP: 100, GridX: 1, GridY: 1}
	back := &simulator.CombatantAuto{ID: "back", Name: "Hàng sau", HP: 100, MaxHP: 100, GridX: 3, GridY: 1}
	state := &simulator.CombatState{
		Allies:  []*simulator.CombatantAuto{player},
		Enemies: []*simulator.CombatantAuto{front, back},
		Interactive: &simulator.InteractiveCombat{
			PendingActor: "player",
		},
	}

	response := combatSessionResponse(combatSessionRow{}, state, nil)
	allies, ok := response["allies"].([]gin.H)
	if !ok || len(allies) != 1 {
		t.Fatalf("combat response allies = %#v", response["allies"])
	}
	actions, ok := allies[0]["actions"].([]gin.H)
	if !ok {
		t.Fatal("pending combatant actions were not returned")
	}

	for _, action := range actions {
		var want []string
		switch action["action_type"] {
		case "BASIC_ATTACK":
			want = []string{"front"}
		case "GU_SKILL":
			want = []string{"front", "back"}
		default:
			continue
		}
		targets, ok := action["target_ids"].([]string)
		if !ok || len(targets) != len(want) {
			t.Fatalf("%s target_ids = %#v, want %v", action["action_type"], action["target_ids"], want)
		}
		for i := range want {
			if targets[i] != want[i] {
				t.Fatalf("%s target_ids = %v, want %v", action["action_type"], targets, want)
			}
		}
	}

	back.IsStealthed = true
	response = combatSessionResponse(combatSessionRow{}, state, nil)
	allies, ok = response["allies"].([]gin.H)
	if !ok || len(allies) != 1 {
		t.Fatalf("combat response allies with stealthed target = %#v", response["allies"])
	}
	actions, ok = allies[0]["actions"].([]gin.H)
	if !ok {
		t.Fatal("pending combatant actions were not returned with stealthed target")
	}
	for _, action := range actions {
		if action["action_type"] != "BASIC_ATTACK" && action["action_type"] != "GU_SKILL" {
			continue
		}
		targets, ok := action["target_ids"].([]string)
		if !ok || len(targets) != 1 || targets[0] != "front" {
			t.Fatalf("%s target_ids with stealthed back target = %#v, want [front]", action["action_type"], action["target_ids"])
		}
	}
}

func TestCombatSessionResponseCompletesTerminalReplay(t *testing.T) {
	row := combatSessionRow{Status: "active"}
	state := &simulator.CombatState{
		Allies:  []*simulator.CombatantAuto{{ID: "player", HP: 100}},
		Enemies: []*simulator.CombatantAuto{{ID: "enemy", HP: 0, IsDead: true}},
		Interactive: &simulator.InteractiveCombat{
			PendingActor: "",
		},
	}

	response := combatSessionResponse(row, state, []simulator.CombatLog{{
		Action: "BATTLE_END",
		Detail: "Trận chiến kết thúc. Phe Ta chiến thắng!",
	}})
	if response["status"] != "complete" {
		t.Fatalf("terminal replay response status = %v, want complete", response["status"])
	}

	row.ParticipantID = uuid.NullUUID{Valid: true}
	row.PendingPlans = []byte(`{"player":{"actor_id":"player"}}`)
	response = combatSessionResponseForPlayer(row, state, []simulator.CombatLog{{
		Action: "BATTLE_END",
	}}, uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	if response["pending_actor"] != "" {
		t.Fatalf("completed session pending_actor = %v, want empty", response["pending_actor"])
	}

	state.Interactive.PendingActor = "player"
	response = combatSessionResponse(row, state, []simulator.CombatLog{{
		Action: "BATTLE_END",
	}})
	if response["status"] != "active" {
		t.Fatalf("session awaiting an actor response status = %v, want active", response["status"])
	}

	state.Interactive.PendingActor = ""
	response = combatSessionResponse(row, state, nil)
	if response["status"] != "active" {
		t.Fatalf("session without terminal log response status = %v, want active", response["status"])
	}
}
