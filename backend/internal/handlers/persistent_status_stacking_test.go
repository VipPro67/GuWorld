package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestApplyPersistentStackingStatusUsesAuthoredTiersAndCap(t *testing.T) {
	spec := models.PersistentHitStatusSpec{
		StatusKey: gamefx.StatusSinhKhiTan, PvPOnly: true, Configured: true,
		MaxStacks: 2, DurationHours: 12, DispelRule: "DECREMENT_ONE",
		TierScaling: []effects.StackingStatusTier{{Tier: 1, StatPercent: -15}, {Tier: 2, StatPercent: -40}},
	}
	var statuses models.StatusList
	for _, want := range []struct {
		tier  int
		value float64
	}{{1, -0.15}, {2, -0.40}, {2, -0.40}} {
		gotTier, gotValue, ok := ApplyPersistentStackingStatusToList(&statuses, spec, "GU_TEST")
		if !ok || gotTier != want.tier || gotValue != want.value {
			t.Fatalf("apply returned (%d, %v, %t), want (%d, %v, true)", gotTier, gotValue, ok, want.tier, want.value)
		}
	}
	status := statuses[0]
	if status.Stacking == nil || status.Stacking.MaxStacks != 2 || status.Stacking.DurationHours != 12 || status.Stacking.CurrentTier != 2 {
		t.Fatalf("stacking metadata did not retain CSV policy: %+v", status.Stacking)
	}
	if status.ExpiresAt == nil || time.Until(*status.ExpiresAt) < 11*time.Hour {
		t.Fatalf("status expiry does not use configured duration: %v", status.ExpiresAt)
	}
}

func TestCleansePersistentStackingStatusUsesAuthoredDispelRule(t *testing.T) {
	newSpec := func(rule string) models.PersistentHitStatusSpec {
		return models.PersistentHitStatusSpec{
			StatusKey: gamefx.StatusSinhKhiTan, PvPOnly: true, Configured: true,
			MaxStacks: 2, DurationHours: 12, DispelRule: rule,
			TierScaling: []effects.StackingStatusTier{{Tier: 1, StatPercent: -15}, {Tier: 2, StatPercent: -40}},
		}
	}

	var decrement models.StatusList
	ApplyPersistentStackingStatusToList(&decrement, newSpec("DECREMENT_ONE"), "GU_TEST")
	ApplyPersistentStackingStatusToList(&decrement, newSpec("DECREMENT_ONE"), "GU_TEST")
	if tier, found := CleansePersistentStackingStatusOneTierFromList(&decrement, gamefx.StatusSinhKhiTan); !found || tier != 1 {
		t.Fatalf("DECREMENT_ONE cleanse returned tier=%d found=%t", tier, found)
	}
	if len(decrement) != 1 || decrement[0].Value != -0.15 {
		t.Fatalf("DECREMENT_ONE did not preserve the remaining tier: %+v", decrement)
	}

	var clearAll models.StatusList
	ApplyPersistentStackingStatusToList(&clearAll, newSpec("CLEAR_ALL"), "GU_TEST")
	ApplyPersistentStackingStatusToList(&clearAll, newSpec("CLEAR_ALL"), "GU_TEST")
	if tier, found := CleansePersistentStackingStatusOneTierFromList(&clearAll, gamefx.StatusSinhKhiTan); !found || tier != 0 {
		t.Fatalf("CLEAR_ALL cleanse returned tier=%d found=%t", tier, found)
	}
	if len(clearAll) != 0 {
		t.Fatalf("CLEAR_ALL left a status behind: %+v", clearAll)
	}
}

func TestStackingCleanseLeavesUnconfiguredStatusesToFullRevoke(t *testing.T) {
	statuses := models.StatusList{{ID: "ordinary_debuff", Value: -0.2}}
	if _, found := CleansePersistentStackingStatusOneTierFromList(&statuses, "ordinary_debuff"); found {
		t.Fatal("ordinary status was incorrectly treated as a configured stack")
	}
	if len(statuses) != 1 {
		t.Fatalf("stacking cleanse mutated ordinary status: %+v", statuses)
	}
}
