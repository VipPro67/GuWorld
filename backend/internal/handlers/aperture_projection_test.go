package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestProjectImmortalApertureViewProjectsManaAndYears(t *testing.T) {
	now := time.Date(2026, time.November, 21, 0, 0, 0, 0, time.UTC)
	updatedAt := now.Add(-48 * time.Hour)
	player := models.Player{
		ResourceState:    models.ResourceState{CurrentMana: 75},
		CultivationState: models.CultivationState{TimeSpeedMultiplier: 2},
	}
	aperture := models.ImmortalAperture{
		State:               "STABLE",
		ImmortalEssence:     10,
		TimeRate:            1,
		InternalYearsPassed: 3,
		NextTribulationYear: 10,
		UpdatedAt:           updatedAt,
	}

	got := projectImmortalApertureView(aperture, player, updatedAt, 0.5, false, true, now)

	if got.aperture.ImmortalEssence != 75 {
		t.Fatalf("ImmortalEssence = %d; want 75", got.aperture.ImmortalEssence)
	}
	if got.aperture.InternalYearsPassed != 8 {
		t.Fatalf("InternalYearsPassed = %v; want 8", got.aperture.InternalYearsPassed)
	}
	if got.effectiveTimeRate != 2.5 {
		t.Fatalf("effectiveTimeRate = %v; want 2.5", got.effectiveTimeRate)
	}
	if !got.shouldPersist {
		t.Fatal("shouldPersist = false; want true")
	}
	if got.shouldTriggerTribulation {
		t.Fatal("shouldTriggerTribulation = true; want false")
	}
}

func TestProjectImmortalApertureViewReconcilesTribulationState(t *testing.T) {
	now := time.Date(2026, time.November, 21, 0, 0, 0, 0, time.UTC)
	updatedAt := now.Add(-time.Hour)
	player := models.Player{CultivationState: models.CultivationState{TimeSpeedMultiplier: 1}}
	aperture := models.ImmortalAperture{
		State:               "TRIBULATION_ACTIVE",
		ApertureHP:          100,
		MaxApertureHP:       100,
		TimeRate:            1,
		InternalYearsPassed: 4,
		NextTribulationYear: 10,
		UpdatedAt:           updatedAt,
	}

	got := projectImmortalApertureView(aperture, player, updatedAt, 0, false, true, now)

	if got.aperture.State != "STABLE" {
		t.Fatalf("State = %q; want STABLE", got.aperture.State)
	}
	if !got.shouldPersist {
		t.Fatal("shouldPersist = false after state repair; want true")
	}
	if got.aperture.UpdatedAt != now {
		t.Fatalf("UpdatedAt = %v; want %v", got.aperture.UpdatedAt, now)
	}
}

func TestProjectImmortalApertureViewProjectsAnchoredQiAndTribulationArrival(t *testing.T) {
	now := time.Date(2026, time.November, 21, 0, 0, 0, 0, time.UTC)
	updatedAt := now.Add(-24 * time.Hour)
	placedAt := now.Add(-30 * time.Minute)
	region := "bai_thien"
	player := models.Player{CultivationState: models.CultivationState{TimeSpeedMultiplier: 5}}
	aperture := models.ImmortalAperture{
		State:               "STABLE",
		TimeRate:            20,
		InternalYearsPassed: 2,
		NextTribulationYear: 2.5,
		HeavenQi:            0,
		EarthQi:             0,
		MaxHeavenQi:         100,
		MaxEarthQi:          100,
		IsPlaced:            true,
		PlacedRegion:        &region,
		PlacedAt:            &placedAt,
		UpdatedAt:           updatedAt,
	}

	got := projectImmortalApertureView(aperture, player, updatedAt, 10, false, true, now)

	if got.effectiveTimeRate != 1 {
		t.Fatalf("effectiveTimeRate = %v; want 1 while anchored in Bai Thien", got.effectiveTimeRate)
	}
	if got.aperture.HeavenQi != 90 || got.aperture.EarthQi != 9 {
		t.Fatalf("projected Qi = (%v, %v); want (90, 9)", got.aperture.HeavenQi, got.aperture.EarthQi)
	}
	if !got.shouldTriggerTribulation {
		t.Fatal("shouldTriggerTribulation = false; want true after projected time passes threshold")
	}
	arrival := apertureNextTribulationAt(got.aperture, got.effectiveTimeRate, false)
	if arrival != nil {
		t.Fatalf("arrival = %v; want nil after projected time passes threshold", arrival)
	}
}

func TestApertureNextTribulationAtUsesEffectiveRate(t *testing.T) {
	updatedAt := time.Date(2026, time.November, 21, 0, 0, 0, 0, time.UTC)
	aperture := models.ImmortalAperture{
		State:               "DAMAGED",
		UpdatedAt:           updatedAt,
		InternalYearsPassed: 2,
		NextTribulationYear: 4,
	}

	got := apertureNextTribulationAt(aperture, 2, false)
	if want := updatedAt.Add(24 * time.Hour); got == nil || !got.Equal(want) {
		t.Fatalf("arrival = %v; want %v", got, want)
	}
	if got := apertureNextTribulationAt(aperture, 2, true); got != nil {
		t.Fatalf("immortal arrival = %v; want nil", got)
	}
}
