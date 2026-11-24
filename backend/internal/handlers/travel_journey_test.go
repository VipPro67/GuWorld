package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestTravelStepDuration(t *testing.T) {
	diagonalLength := 1.41421356237
	diagonalDuration := time.Duration(10 * diagonalLength * float64(time.Second))
	tests := []struct {
		name     string
		movement float64
		length   float64
		want     time.Duration
	}{
		{name: "base cardinal step", movement: 1, length: 1, want: 10 * time.Second},
		{name: "diagonal step", movement: 1, length: diagonalLength, want: diagonalDuration},
		{name: "minimum clamp", movement: 100, length: 1, want: time.Second},
		{name: "maximum clamp", movement: 0.01, length: 1, want: 60 * time.Second},
		{name: "no movement", movement: 0, length: 1, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := travelStepDuration(test.movement, test.length); got != test.want {
				t.Fatalf("travelStepDuration(%v, %v) = %v, want %v", test.movement, test.length, got, test.want)
			}
		})
	}
}

func TestTravelNextStepCrossesAdjacentZoneThroughWalkingEdge(t *testing.T) {
	previousZones, previousByID := zoneReg.zones, zoneReg.byID
	t.Cleanup(func() {
		zoneReg.zones, zoneReg.byID = previousZones, previousByID
	})
	zoneReg.zones = []models.Region{
		{ID: "travel_a", ZoneType: ZoneTypeNamed, Width: 10, Height: 10, AtlasX: 0, AtlasY: 0, MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
		{ID: "travel_b", ZoneType: ZoneTypeNamed, Width: 10, Height: 10, AtlasX: 10, AtlasY: 0, MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
	}
	zoneReg.byID = make(map[string]*models.Region, len(zoneReg.zones))
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}

	step, length, ok := travelNextStep("travel_a", "travel_b", 10, 5, 3, 5)
	if !ok || step != [2]int{11, 5} || length != 1 {
		t.Fatalf("travelNextStep at zone edge = (%v, %v, %v), want ([11 5], 1, true)", step, length, ok)
	}
	step, _, ok = travelNextStep("travel_a", "travel_b", 5, 5, 3, 5)
	if !ok || step != [2]int{6, 5} {
		t.Fatalf("travelNextStep toward zone edge = (%v, %v), want ([6 5], true)", step, ok)
	}
}

func TestEstimateTravelRouteAcrossAdjacentZones(t *testing.T) {
	previousZones, previousByID := zoneReg.zones, zoneReg.byID
	t.Cleanup(func() {
		zoneReg.zones, zoneReg.byID = previousZones, previousByID
	})
	zoneReg.zones = []models.Region{
		{ID: "travel_a", ZoneType: ZoneTypeNamed, Width: 10, Height: 10, AtlasX: 0, AtlasY: 0, MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
		{ID: "travel_b", ZoneType: ZoneTypeNamed, Width: 10, Height: 10, AtlasX: 10, AtlasY: 0, MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
	}
	zoneReg.byID = make(map[string]*models.Region, len(zoneReg.zones))
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}

	lengths, regions, ok := estimateTravelRoute(nil, "travel_a", "travel_b", 5, 5, 3, 5)
	if !ok || len(regions) != 2 || regions[0] != "travel_a" || regions[1] != "travel_b" {
		t.Fatalf("estimateTravelRoute region path = (%v, %v), want [travel_a travel_b], true", regions, ok)
	}
	if got := sumTravelDistance(lengths); got != 7 {
		t.Fatalf("estimated route distance = %v, want 7", got)
	}
}

func TestCalculatePlayerStatsAtUsesStatusAtScheduledTime(t *testing.T) {
	previousRegistry := gamefx.StatusRegistry
	gamefx.StatusRegistry = map[string]gamefx.StatusDef{
		"travel_movement_test": {
			Key:     "travel_movement_test",
			Effects: gamefx.Effects{StatMods: map[string]float64{"movement": 2}},
		},
	}
	t.Cleanup(func() { gamefx.StatusRegistry = previousRegistry })

	now := time.Now().UTC()
	scheduledAt := now.Add(-2 * time.Minute)
	expiresAt := now.Add(-time.Minute)
	player := models.Player{CultivationState: models.CultivationState{Rank: 1, SubRank: 1}}
	player.Statuses = models.StatusList{{ID: "travel_movement_test", ExpiresAt: &expiresAt}}

	if got := CalculatePlayerStats(&player)["movement"]; got != 1 {
		t.Fatalf("current movement = %v, want expired buff to be absent", got)
	}
	if got := calculatePlayerStatsAt(&player, scheduledAt)["movement"]; got != 3 {
		t.Fatalf("movement at scheduled time = %v, want buff to apply to catch-up step", got)
	}
}
