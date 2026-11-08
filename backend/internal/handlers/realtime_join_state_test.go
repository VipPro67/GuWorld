package handlers

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRealtimeMapIDPrefersDetailedChildRegion(t *testing.T) {
	tests := []struct {
		name, regionID, zoneType, assigned, want string
	}{
		{"KHU_VUC overrides stale outer binding", "thanh_mao_son", "KHU_VUC", "world_map", "thanh_mao_son"},
		{"SAFE_ZONE overrides stale outer binding", "hub_dong_hai", "SAFE_ZONE", "world_map", "hub_dong_hai"},
		{"TRIBULATION uses region map directly", "aperture_tribulation", "TRIBULATION", "world_map", "aperture_tribulation"},
		{"NGU_VUC keeps outer map", "dai_vuc_dong_hai", "NGU_VUC", "world_map", "world_map"},
		{"special region keeps explicit assignment", "secret_realm", "SECRET_REALM", "realm_map", "realm_map"},
		{"missing assignment falls back to world map", "", "", "", "world_map"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := realtimeMapID(test.regionID, test.zoneType, test.assigned); got != test.want {
				t.Fatalf("realtimeMapID()=%q, want %q", got, test.want)
			}
		})
	}
}

func TestRealtimeMovementSpeedUsesPassiveMovementAboveBaseline(t *testing.T) {
	tests := []struct {
		movement float64
		want     float64
	}{
		{movement: 1, want: 4},
		{movement: 2, want: 4.2},
		{movement: 9, want: 5.6},
		{movement: 0, want: 4},
	}
	for _, test := range tests {
		if got := RealtimeMovementSpeed(test.movement); got != test.want {
			t.Errorf("RealtimeMovementSpeed(%v) = %v, want %v", test.movement, got, test.want)
		}
	}
}

func TestRealtimeCaptureGuBonusesProjectsEquippedCaptureCapability(t *testing.T) {
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_CAPTURE": {
			9: {
				ID:   "GU_CAPTURE",
				Rank: 9,
				Path: []string{"Enslavement Path"},
				Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
					Trigger: gameplayeffects.TriggerCapability,
					Target:  gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
					Operations: []gameplayeffects.Operation{{
						Op: gameplayeffects.OpUnique, Code: "CAPTURE_BONUS",
						Params: []byte(`{"value":0.2}`),
					}},
				}}},
			},
		},
	}

	bonuses, err := realtimeCaptureGuBonuses(models.GuInventory{
		{InstanceID: "capture", GuID: "GU_CAPTURE", Rank: 9, Hunger: 5, Status: "ACTIVE", Equipped: true},
		{InstanceID: "hungry", GuID: "GU_CAPTURE", Rank: 9, Hunger: 4, Status: "ACTIVE", Equipped: true},
		{InstanceID: "sleeping", GuID: "GU_CAPTURE", Rank: 9, Hunger: 5, Status: "SLEEPING", Equipped: true},
		{InstanceID: "unequipped", GuID: "GU_CAPTURE", Rank: 9, Hunger: 5, Status: "ACTIVE"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bonuses) != 1 || bonuses["capture"] != 0.2 {
		t.Fatalf("capture bonuses = %#v, want only capture: 0.2", bonuses)
	}
}
