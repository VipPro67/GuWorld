package simulator

import "testing"

func TestGuHouseActionsRequireAnActiveUndestroyedHouse(t *testing.T) {
	action := &SimAction{IsFromGuHouse: true}
	actor := &CombatantAuto{GuHouse: &GuHouseState{ID: "house", HP: 100}}
	if !checkActionCondition(action, actor, nil) {
		t.Fatal("valid Gu House action should be available")
	}
	actor.GuHouse.IsDestroyed = true
	if checkActionCondition(action, actor, nil) {
		t.Fatal("destroyed Gu House action should be unavailable")
	}
	actor.GuHouse = nil
	if checkActionCondition(action, actor, nil) {
		t.Fatal("Gu House action without a house should be unavailable")
	}
}
