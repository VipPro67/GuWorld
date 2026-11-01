package handlers

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestAscensionGradeForHumanQi(t *testing.T) {
	cases := []struct {
		qi   float64
		want string
	}{
		{0, ""},
		{99999, ""},
		{100000, "LOW"},
		{229999, "LOW"},
		{230000, "MID"},
		{369999, "MID"},
		{370000, "HIGH"},
		{479999, "HIGH"},
		{480000, "EXTREME"},
		{500000, "EXTREME"},
	}
	for _, c := range cases {
		if got := AscensionGradeForHumanQi(c.qi); got != c.want {
			t.Errorf("AscensionGradeForHumanQi(%v) = %q, want %q", c.qi, got, c.want)
		}
	}
}

func TestEligibleAscensionReleaseGu(t *testing.T) {
	player := &models.Player{
		DaoState: models.DaoState{
			PathMasteries: models.PathMasteriesMap{
				"fire": 15000, // Đại Sư
				"ice":  9999,  // Phàm Thai
			},
		},
		GuInventory: models.GuInventory{
			{InstanceID: "vital", GuID: "fire_gu_5", Rank: 5, IsVital: true, Status: "ACTIVE"},
			{InstanceID: "fire5", GuID: "fire_gu_5", Rank: 5, Status: "ACTIVE"},
			{InstanceID: "fire4", GuID: "fire_gu_4", Rank: 4, Status: "ACTIVE"},
			{InstanceID: "ice5", GuID: "ice_gu_5", Rank: 5, Status: "ACTIVE"},
			{InstanceID: "dead5", GuID: "fire_gu_5b", Rank: 5, Status: "DEAD"},
		},
	}

	// Registry lookup happens against models.GuRegistry; stub the entries used above.
	prev := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = prev })
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"fire_gu_5":  {5: {ID: "fire_gu_5", Name: "Hỏa Cổ 5", Path: []string{"Fire Path"}, Rank: 5}},
		"fire_gu_4":  {4: {ID: "fire_gu_4", Name: "Hỏa Cổ 4", Path: []string{"Fire Path"}, Rank: 4}},
		"ice_gu_5":   {5: {ID: "ice_gu_5", Name: "Băng Cổ 5", Path: []string{"Ice Path"}, Rank: 5}},
		"fire_gu_5b": {5: {ID: "fire_gu_5b", Name: "Hỏa Cổ 5b", Path: []string{"Fire Path"}, Rank: 5}},
	}

	eligible := eligibleAscensionReleaseGu(player)
	if len(eligible) != 1 || eligible[0].InstanceID != "fire5" {
		t.Fatalf("eligibleAscensionReleaseGu = %+v, want only [fire5]", eligible)
	}
}

func TestAscensionWorldSpawnerHook(t *testing.T) {
	called := false
	SetAscensionWorldSpawner(func(ctx context.Context, playerID uuid.UUID, zoneID string, x, y float64, hp, atk, def float64) error {
		called = true
		return nil
	})
	t.Cleanup(func() { SetAscensionWorldSpawner(nil) })

	ascensionWorldSpawnerMu.RLock()
	spawner := ascensionWorldSpawner
	ascensionWorldSpawnerMu.RUnlock()

	if spawner == nil {
		t.Fatal("expected non-nil ascensionWorldSpawner")
	}
	_ = spawner(context.Background(), uuid.New(), "Southern_Border", 10, 10, 100, 10, 5)
	if !called {
		t.Fatal("expected ascensionWorldSpawner to be invoked")
	}
}

func TestAscensionMinHumanQiRequirement(t *testing.T) {
	if AscensionMinHumanQi != 100000.0 {
		t.Errorf("AscensionMinHumanQi = %v, want 100000.0", AscensionMinHumanQi)
	}
	if grade := AscensionGradeForHumanQi(AscensionMinHumanQi - 1); grade != "" {
		t.Errorf("AscensionGradeForHumanQi(min - 1) = %q, want empty string", grade)
	}
	if grade := AscensionGradeForHumanQi(AscensionMinHumanQi); grade != "LOW" {
		t.Errorf("AscensionGradeForHumanQi(min) = %q, want LOW", grade)
	}
}
