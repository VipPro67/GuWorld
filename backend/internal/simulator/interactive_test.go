package simulator

import "testing"

func interactiveTestState(commands []CombatCommand) *CombatState {
	ally := newTestCombatant("player", "Player", 1000, 100)
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", SourceID: "gu-fire", InstanceID: "owned-fire", Name: "Hỏa", Damage: 100, TargetType: "SINGLE"}}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 20)
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: 20, TargetType: "SINGLE"}}
	return &CombatState{
		Seed: 42, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy},
		Interactive: &InteractiveCombat{
			ManualActors: map[string]bool{"player": true}, Commands: commands,
		},
	}
}

func TestInteractiveCombatPausesAndReplaysOneCommand(t *testing.T) {
	first := interactiveTestState(nil)
	logs := Simulate(first)
	if first.Interactive.PendingActor != "player" || first.Enemies[0].HP != 1000 {
		t.Fatalf("initial decision did not pause before damage: pending=%q hp=%g", first.Interactive.PendingActor, first.Enemies[0].HP)
	}
	if len(logs) == 0 || logs[len(logs)-1].Action != "AWAIT_COMMAND" || logs[len(logs)-1].Snapshot == nil {
		t.Fatal("pause must include a recoverable battlefield snapshot")
	}

	command := CombatCommand{ActorID: "player", ActionType: "GU_SKILL", SourceID: "gu-fire", InstanceID: "owned-fire", TargetID: "enemy"}
	second := interactiveTestState([]CombatCommand{command})
	Simulate(second)
	if second.Interactive.Error != "" || second.Interactive.Consumed != 1 || second.Interactive.PendingActor != "player" {
		t.Fatalf("command replay failed: %+v", second.Interactive)
	}
	if second.Enemies[0].HP >= 1000 {
		t.Fatal("selected Gu action did not damage the target")
	}
	if second.Allies[0].HP >= 1000 {
		t.Fatal("NPC did not act before the next human decision")
	}
}

func TestInteractiveCombatRejectsUnloadedAction(t *testing.T) {
	state := interactiveTestState([]CombatCommand{{ActorID: "player", ActionType: "GU_SKILL", SourceID: "gu-unloaded", TargetID: "enemy"}})
	Simulate(state)
	if state.Interactive.Error == "" || state.Interactive.Consumed != 0 || state.Enemies[0].HP != 1000 {
		t.Fatalf("invalid command changed combat state: %+v, enemy hp=%g", state.Interactive, state.Enemies[0].HP)
	}
}

func TestInteractiveCombatAcceptsAutoActionForPlayerTurn(t *testing.T) {
	state := interactiveTestState([]CombatCommand{{ActorID: "player", ActionType: "AUTO"}})
	logs := Simulate(state)
	if state.Interactive.Error != "" || state.Interactive.Consumed != 1 {
		t.Fatalf("auto command was not consumed: %+v", state.Interactive)
	}
	if state.Enemies[0].HP >= 1000 {
		t.Fatal("auto command did not select and execute a legal action")
	}
	if len(logs) == 0 || (logs[len(logs)-1].Action == "AWAIT_COMMAND" && logs[len(logs)-1].Turn == 1) {
		t.Fatal("auto command should resolve the player's activation before pausing again")
	}
}

func TestInteractiveCombatFastNPCActsMoreOften(t *testing.T) {
	state := interactiveTestState(nil)
	state.Allies[0].Speed = 1
	state.Enemies[0].Speed = 9
	Simulate(state)
	if state.Interactive.PendingActor != "player" {
		t.Fatalf("pending actor = %q", state.Interactive.PendingActor)
	}
	if state.Allies[0].HP > 960 {
		t.Fatalf("fast enemy should act more than once before player: hp=%g", state.Allies[0].HP)
	}
}

func TestAutoCombatHonorsNPCBehaviorProfile(t *testing.T) {
	t.Run("burst selects strongest damage action", func(t *testing.T) {
		actor := newTestCombatant("npc", "Boss", 1000, 20)
		actor.AIProfile = "boss_burst"
		actor.Actions = []SimAction{
			{ActionType: "BASIC_ATTACK", Name: "basic", Damage: 20, DamageMult: 1, TargetType: "SINGLE"},
			{ActionType: "GU_SKILL", SourceID: "blast", Name: "blast", Damage: 80, DamageMult: 1, TargetType: "SINGLE"},
		}
		action, _ := chooseAutoCombatAction(actor, []*CombatantAuto{newTestCombatant("target", "Target", 1000, 10)})
		if action == nil || action.SourceID != "blast" {
			t.Fatalf("boss_burst chose %#v, want blast", action)
		}
	})

	t.Run("elite beast profile selects strongest damage action", func(t *testing.T) {
		actor := newTestCombatant("npc", "Elite", 1000, 20)
		actor.AIProfile = "beast_elite"
		actor.Actions = []SimAction{
			{ActionType: "BASIC_ATTACK", Name: "basic", Damage: 20, DamageMult: 1, TargetType: "SINGLE"},
			{ActionType: "GU_SKILL", SourceID: "claw", Name: "claw", Damage: 80, DamageMult: 1, TargetType: "SINGLE"},
		}
		action, _ := chooseAutoCombatAction(actor, []*CombatantAuto{newTestCombatant("target", "Target", 1000, 10)})
		if action == nil || action.SourceID != "claw" {
			t.Fatalf("beast_elite chose %#v, want claw", action)
		}
	})

	t.Run("defensive profile heals itself at low health", func(t *testing.T) {
		actor := newTestCombatant("npc", "Boss", 1000, 20)
		actor.HP = 200
		actor.AIProfile = "defensive"
		actor.Actions = []SimAction{
			{ActionType: "BASIC_ATTACK", Name: "basic", Damage: 100, DamageMult: 1, TargetType: "SINGLE"},
			{ActionType: "GU_SKILL", SourceID: "mend", Name: "mend", Healing: 90, TargetType: "SELF"},
		}
		action, target := chooseAutoCombatAction(actor, []*CombatantAuto{newTestCombatant("target", "Target", 1000, 10)})
		if action == nil || action.SourceID != "mend" || target != actor {
			t.Fatalf("defensive profile chose action=%#v target=%#v, want self heal", action, target)
		}
	})
}

func TestInteractiveFleeUsesSpeedAndReplaysDeterministically(t *testing.T) {
	fast := newTestCombatant("fast", "Fast", 100, 20)
	slow := newTestCombatant("slow", "Slow", 100, 20)
	fast.Speed = 10
	slow.Speed = 2
	if FleeChance(fast, []*CombatantAuto{slow}) <= FleeChance(slow, []*CombatantAuto{fast}) {
		t.Fatal("faster actor should have a higher flee chance")
	}

	run := func() (*CombatState, []CombatLog) {
		state := interactiveTestState([]CombatCommand{{ActorID: "player", ActionType: "FLEE"}})
		state.Seed = 42
		state.Allies[0].Speed = 100
		return state, Simulate(state)
	}
	first, firstLogs := run()
	second, secondLogs := run()
	if first.FledBy != second.FledBy || len(firstLogs) != len(secondLogs) {
		t.Fatalf("flee replay diverged: first=%q/%d second=%q/%d", first.FledBy, len(firstLogs), second.FledBy, len(secondLogs))
	}
	if first.Interactive.Error != "" || first.Interactive.Consumed != 1 {
		t.Fatalf("flee command was not consumed: %+v", first.Interactive)
	}
	if first.FledBy == "" && first.Interactive.PendingActor != "player" {
		t.Fatalf("failed flee should spend one activation and reach a later player decision: %+v", first.Interactive)
	}
	foundFleeLog := false
	for _, entry := range firstLogs {
		if entry.Action == "FLEE_SUCCESS" || entry.Action == "FLEE_FAILED" {
			foundFleeLog = true
			break
		}
	}
	if !foundFleeLog {
		t.Fatal("flee outcome was not recorded in the replay")
	}
}
