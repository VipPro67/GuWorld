package simulator

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestCollectAllStatusOpsUsesGuInstanceForLifecycleIdentity(t *testing.T) {
	action := &SimAction{
		SourceID:   "gu-fire",
		InstanceID: "instance-7",
		StatusOps:  []statusfx.StatusOp{{ID: "BURN", EffectID: "ignite:0", Context: "combat", StackPolicy: "STACK"}},
	}
	got := collectAllStatusOps(action)
	if len(got) != 1 || got[0].SourceID != "instance-7" || got[0].EffectID != "ignite:0" {
		t.Fatalf("status provenance lost its source instance or effect definition: %+v", got)
	}
}

func TestMergeCombatEffectRefreshesSameSourceAndKeepsMaxWinnerSnapshot(t *testing.T) {
	effects := []Effect{{ID: "ATK_UP", Value: 10, Duration: 2, SourceID: "gu-a", EffectID: "block:0", Context: "combat", StackPolicy: "MAX"}}
	mergeCombatEffect(&effects, Effect{ID: "ATK_UP", Value: 8, Duration: 5, SourceID: "gu-a", EffectID: "block:0", Context: "combat", StackPolicy: "MAX"}, "MAX")
	if len(effects) != 1 || effects[0].Value != 8 || effects[0].Duration != 5 {
		t.Fatalf("same-source refresh must replace the applied snapshot: %+v", effects)
	}
	mergeCombatEffect(&effects, Effect{ID: "ATK_UP", Value: 12, Duration: 2, SourceID: "gu-b", EffectID: "block:0", Context: "combat", StackPolicy: "MAX"}, "MAX")
	if len(effects) != 1 || effects[0].SourceID != "gu-b" || effects[0].Value != 12 || effects[0].Duration != 2 {
		t.Fatalf("higher-value instance must retain its own provenance and duration: %+v", effects)
	}
	mergeCombatEffect(&effects, Effect{ID: "ATK_UP", Value: 12, Duration: 4, SourceID: "gu-c", EffectID: "block:0", Context: "combat", StackPolicy: "MAX"}, "MAX")
	if len(effects) != 1 || effects[0].SourceID != "gu-c" || effects[0].Duration != 4 {
		t.Fatalf("equal values must prefer the longer remaining duration: %+v", effects)
	}
}

func TestMergeCombatEffectKeepsDifferentSourcesByDefault(t *testing.T) {
	effects := []Effect{{ID: "ATK_UP", Value: 10, Duration: 2, SourceID: "gu-a", EffectID: "block:0", Context: "combat"}}
	mergeCombatEffect(&effects, Effect{ID: "ATK_UP", Value: 12, Duration: 4, SourceID: "gu-b", EffectID: "block:0", Context: "combat"}, "")
	if len(effects) != 2 {
		t.Fatalf("different sources should remain independent by default: %+v", effects)
	}
	mergeCombatEffect(&effects, Effect{ID: "ATK_UP", Value: 15, Duration: 3, SourceID: "gu-c", EffectID: "block:0", Context: "combat"}, "STACK")
	if len(effects) != 3 {
		t.Fatalf("STACK should retain each source as an independent instance: %+v", effects)
	}
}

func TestMergeCombatEffectPolicyCollapsesPreviouslyStackedSources(t *testing.T) {
	effects := []Effect{
		{ID: "ATK_UP", Value: 10, Duration: 3, SourceID: "gu-a", EffectID: "block:0", Context: "combat"},
		{ID: "ATK_UP", Value: 12, Duration: 2, SourceID: "gu-b", EffectID: "block:0", Context: "combat"},
	}
	mergeCombatEffect(&effects, Effect{ID: "ATK_UP", Value: 11, Duration: 5, SourceID: "gu-c", EffectID: "block:0", Context: "combat"}, "MAX")
	if len(effects) != 1 || effects[0].SourceID != "gu-b" || effects[0].Value != 12 {
		t.Fatalf("MAX must select one winner across previously stacked sources: %+v", effects)
	}
}

func TestMergeCombatEffectReplaceAndIgnore(t *testing.T) {
	effects := []Effect{{ID: "ATK_UP", Value: 10, Duration: 2, SourceID: "gu-a", EffectID: "old:0", Context: "combat"}}
	mergeCombatEffect(&effects, Effect{ID: "ATK_UP", Value: 12, Duration: 4, SourceID: "gu-b", EffectID: "new:0", Context: "combat"}, "IGNORE")
	if len(effects) != 1 || effects[0].SourceID != "gu-a" {
		t.Fatalf("IGNORE should keep the already-active status: %+v", effects)
	}
	mergeCombatEffect(&effects, Effect{ID: "ATK_UP", Value: 12, Duration: 4, SourceID: "gu-b", EffectID: "new:0", Context: "combat"}, "REPLACE")
	if len(effects) != 1 || effects[0].SourceID != "gu-b" {
		t.Fatalf("REPLACE should keep only the newly applied status: %+v", effects)
	}
}

func TestMergeCombatDoTMinKeepsWinningSnapshot(t *testing.T) {
	dots := []DoTEffect{{Type: "burn", Value: 20, Turns: 2, SourceID: "gu-a", EffectID: "block:0", Context: "combat"}}
	mergeCombatDoT(&dots, DoTEffect{Type: "burn", Value: 10, Turns: 2, SourceID: "gu-b", EffectID: "block:0", Context: "combat"}, "MIN")
	if len(dots) != 1 || dots[0].SourceID != "gu-b" || dots[0].Value != 10 {
		t.Fatalf("lower-value DoT must win with its provenance: %+v", dots)
	}
	mergeCombatDoT(&dots, DoTEffect{Type: "burn", Value: 10, Turns: 5, SourceID: "gu-c", EffectID: "block:0", Context: "combat"}, "MIN")
	if len(dots) != 1 || dots[0].SourceID != "gu-c" || dots[0].Turns != 5 {
		t.Fatalf("equal values must prefer the longer remaining duration: %+v", dots)
	}
}

func TestMergeCombatDoTPolicyCollapsesPreviouslyStackedSources(t *testing.T) {
	dots := []DoTEffect{
		{Type: "burn", Value: 20, Turns: 3, SourceID: "gu-a", EffectID: "block:0", Context: "combat"},
		{Type: "burn", Value: 15, Turns: 2, SourceID: "gu-b", EffectID: "block:0", Context: "combat"},
	}
	mergeCombatDoT(&dots, DoTEffect{Type: "burn", Value: 18, Turns: 5, SourceID: "gu-c", EffectID: "block:0", Context: "combat"}, "MIN")
	if len(dots) != 1 || dots[0].SourceID != "gu-b" || dots[0].Value != 15 {
		t.Fatalf("MIN must select one winner across previously stacked DoTs: %+v", dots)
	}
}
