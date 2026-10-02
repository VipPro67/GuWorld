package models

import "testing"

func TestCanTransitionEnforcesExecutionOwner(t *testing.T) {
	actor := NonPlayerActor{LifecycleState: ActorLifecycleActive, ExecutionState: ActorStateStored}
	if !CanTransition(actor, ActorStateStored, ActorStateDeploying) {
		t.Fatal("expected STORED -> DEPLOYING to be allowed")
	}
	if CanTransition(actor, ActorStateStored, ActorStateCombat) {
		t.Fatal("expected STORED -> COMBAT to be rejected")
	}
	if CanTransition(actor, ActorStateWorldIdle, ActorStateMoving) {
		t.Fatal("expected stale from-state to be rejected")
	}
	actor.LifecycleState = ActorLifecycleDead
	if CanTransition(actor, ActorStateStored, ActorStateDeploying) {
		t.Fatal("expected terminal lifecycle to reject transitions")
	}
}

func TestCanExecuteUsesCapabilitiesNotKind(t *testing.T) {
	actor := NonPlayerActor{
		ActorKind:      ActorKindMonster,
		LifecycleState: ActorLifecycleActive,
		Capabilities:   ActorCapabilities{CapabilityTrade: true},
	}
	if !CanExecute(actor, CapabilityTrade) {
		t.Fatal("expected persisted capability to allow execution regardless of kind")
	}
	if CanExecute(actor, CapabilityCombat) {
		t.Fatal("expected absent capability to deny execution")
	}
}

func TestCanReceiveCommandRequiresOwnerAndCapability(t *testing.T) {
	owner := "player-1"
	actor := NonPlayerActor{
		OwnerPlayerID:  &owner,
		LifecycleState: ActorLifecycleActive,
		Capabilities:   ActorCapabilities{CapabilityFollow: true},
	}
	if err := CanReceiveCommand(actor, owner, ActorCommandFollow); err != nil {
		t.Fatalf("expected owner FOLLOW command to pass: %v", err)
	}
	if err := CanReceiveCommand(actor, "player-2", ActorCommandFollow); err == nil {
		t.Fatal("expected command from non-owner to fail")
	}
	if err := CanReceiveCommand(actor, owner, ActorCommandAttack); err == nil {
		t.Fatal("expected command without capability to fail")
	}
}
