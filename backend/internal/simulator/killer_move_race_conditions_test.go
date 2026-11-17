package simulator

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func TestFilterOperationTargetsEvaluatesRacePerCandidate(t *testing.T) {
	actor := &CombatantAuto{ID: "actor", Race: "human"}
	targets := []*CombatantAuto{
		{ID: "merman", Race: "merman"},
		{ID: "rockman", Race: "rockman"},
	}

	got := filterOperationTargets(actor, targets, []gameplayeffects.Condition{{
		Field: "TARGET_RACE", Operator: "EQ", Value: "merman",
	}}, 1, nil, 1)
	if len(got) != 1 || got[0].ID != "merman" {
		t.Fatalf("filterOperationTargets(TARGET_RACE=merman) = %v, want only merman", targetIDs(got))
	}

	got = filterOperationTargets(actor, targets, []gameplayeffects.Condition{{
		Field: "ACTOR_RACE", Operator: "EQ", Value: "human",
	}}, 1, nil, 1)
	if len(got) != len(targets) {
		t.Fatalf("filterOperationTargets(ACTOR_RACE=human) = %v, want both candidates", targetIDs(got))
	}

	actor.Race = ""
	got = filterOperationTargets(actor, targets, []gameplayeffects.Condition{{
		Field: "ACTOR_RACE", Operator: "EQ", Value: "human",
	}}, 1, nil, 1)
	if len(got) != 0 {
		t.Fatalf("filterOperationTargets(missing ACTOR_RACE) = %v, want no candidates", targetIDs(got))
	}
}

func targetIDs(targets []*CombatantAuto) []string {
	ids := make([]string, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.ID)
	}
	return ids
}
