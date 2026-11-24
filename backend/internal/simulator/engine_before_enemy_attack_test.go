package simulator

import (
	"reflect"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestTypedGuSealMatchesLegacyForgetIntent(t *testing.T) {
	newState := func(typed bool) *CombatState {
		attacker := newTestCombatant("attacker", "Player", 1000, 0)
		attacker.Speed = 100
		attacker.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 100}}
		defender := newTestCombatant("defender", "Enemy", 50, 0)
		defender.Speed = 1
		defender.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
		if typed {
			defender.PassiveOperationEffects = []SimActionOperation{{
				Compiled: gameplayeffects.CompiledOperation{Type: gameplayeffects.OpGuSeal, Trigger: gameplayeffects.TriggerBeforeEnemyAttack, GuSeal: &effectcatalog.GuSealParams{CounterKey: "forget_intent_success", Chance: 1, Turns: 1, MaxActivations: 8, BackfireAtLimit: true, BackfireTurns: 1}},
				Action:   &SimAction{IsEffectOperation: true},
			}}
		} else {
			defender.Mechanics = []CombatMechanic{mech("BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", `{"chance":1,"disable_target_gu_turns":1,"max_successful_activations":8,"backfire_after_max":true}`)}
		}
		return &CombatState{Seed: 71, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}}
	}
	legacyState, typedState := newState(false), newState(true)
	legacyLogs, typedLogs := Simulate(legacyState), Simulate(typedState)
	if !reflect.DeepEqual(typedLogs, legacyLogs) {
		t.Fatalf("typed GU_SEAL logs differ from legacy: typed=%+v legacy=%+v", typedLogs, legacyLogs)
	}
	sealed := false
	for _, effect := range typedState.Allies[0].ActiveEffects {
		sealed = sealed || effect.ID == statusfx.StatusGuSeal
	}
	if !sealed || typedState.Enemies[0].CustomState["forget_intent_success"] != 1 {
		t.Fatalf("typed GU_SEAL did not seal the attacker and increment the defender's success counter: effects=%+v state=%+v", typedState.Allies[0].ActiveEffects, typedState.Enemies[0].CustomState)
	}
}
