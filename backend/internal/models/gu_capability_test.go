package models

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func TestInventoryHasGuCapabilityPreservesRuntimeGates(t *testing.T) {
	previous := GuRegistry
	t.Cleanup(func() { GuRegistry = previous })
	GuRegistry = map[string]map[int]GuDictionaryItem{
		"GU_CAP": {1: {
			ID: "GU_CAP", Rank: 1,
			Effects: GuEffects{Blocks: []gameplayeffects.Block{{
				Trigger: gameplayeffects.TriggerCapability,
				Target:  gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
				Operations: []gameplayeffects.Operation{{
					Op: gameplayeffects.OpUnique, Code: "MARKETPLACE_ACCESS",
				}},
			}}},
		}},
	}

	active := GuInstance{GuID: "GU_CAP", Rank: 1, Equipped: true, Status: "ACTIVE"}
	if !InventoryHasGuCapability(GuInventory{active}, "MARKETPLACE_ACCESS", true) {
		t.Fatal("equipped ACTIVE capability was not found")
	}
	active.Status = "DEAD"
	if InventoryHasGuCapability(GuInventory{active}, "MARKETPLACE_ACCESS", false) {
		t.Fatal("DEAD Gu exposed a capability")
	}
	active.Status, active.Equipped = "INACTIVE", false
	if InventoryHasGuCapability(GuInventory{active}, "MARKETPLACE_ACCESS", true) {
		t.Fatal("unequipped inactive Gu exposed an equipped-only capability")
	}
	if !InventoryHasGuCapability(GuInventory{active}, "MARKETPLACE_ACCESS", false) {
		t.Fatal("owned non-DEAD Gu should expose an ownership capability")
	}
}
