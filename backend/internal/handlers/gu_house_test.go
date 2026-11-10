package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGetGuTargetType(t *testing.T) {
	singleGu := models.Gu{
		ID:   "single_gu",
		Rank: 1,
	}
	if tt := getGuTargetType(singleGu); tt != "SINGLE" {
		t.Fatalf("expected SINGLE, got %s", tt)
	}

	aoeGu := models.Gu{
		ID:   "aoe_gu",
		Rank: 3,
		SkillProfile: &models.GuSkillProfile{
			Shape: models.SkillShapeCircle,
		},
	}
	if tt := getGuTargetType(aoeGu); tt != "ALL_ENEMIES" {
		t.Fatalf("expected ALL_ENEMIES, got %s", tt)
	}

	lineGu := models.Gu{
		ID:   "line_gu",
		Rank: 2,
		SkillProfile: &models.GuSkillProfile{
			Shape: models.SkillShapeLine,
		},
	}
	if tt := getGuTargetType(lineGu); tt != "ROW" {
		t.Fatalf("expected ROW, got %s", tt)
	}
}

func TestCompileKillerMoveCombatProfile(t *testing.T) {
	mainSpec := models.Gu{
		ID:   "main_gu",
		Name: "Hạch Tâm Cổ",
		Rank: 3,
		Path: []string{"strength"},
	}
	main := ownedKillerMoveGu{
		InstanceID: "inst_main",
		GuID:       "main_gu",
		Rank:       3,
	}
	components := []killerMoveComponent{
		{InstanceID: "inst_main", GuID: "main_gu", Rank: 3},
		{InstanceID: "inst_sub", GuID: "sub_gu", Rank: 3},
	}

	prof := compileKillerMoveCombatProfile(
		mainSpec,
		main,
		components,
		nil,
		nil,
		"strength",
		1,
		0.85,
		1.1,
		0.80,
		50,
	)

	if prof.Rank != 3 {
		t.Fatalf("expected rank 3, got %d", prof.Rank)
	}
	if prof.RealtimeCooldownMS == 0 || prof.RealtimeWindupMS == 0 {
		t.Fatalf("expected explicit realtime timing, got windup=%d cooldown=%d", prof.RealtimeWindupMS, prof.RealtimeCooldownMS)
	}
	if prof.TargetType != "SINGLE" {
		t.Fatalf("expected target SINGLE, got %s", prof.TargetType)
	}
}

func TestKillerMoveStarvingComponentAvailability(t *testing.T) {
	ownedSet := map[string]ownedKillerMoveGu{
		"inst_1": {InstanceID: "inst_1", GuID: "gu_a", Rank: 1, Hunger: 100, State: "ACTIVE"},
		"inst_2": {InstanceID: "inst_2", GuID: "gu_b", Rank: 1, Hunger: 0, State: "ACTIVE"},
	}

	components := []killerMoveComponent{
		{InstanceID: "inst_1", GuID: "gu_a", Rank: 1},
		{InstanceID: "inst_2", GuID: "gu_b", Rank: 1},
	}

	available, hasStarvingGu, missingGu := evaluateKillerMoveComponents(components, ownedSet)
	if !available {
		t.Fatalf("expected available=true when all components are owned and alive, got available=false")
	}
	if !hasStarvingGu {
		t.Fatalf("expected has_starving_gu=true when a component has hunger=0, got false")
	}
	if len(missingGu) > 0 {
		t.Fatalf("expected 0 missing gu, got %d", len(missingGu))
	}

	// Missing component test
	delete(ownedSet, "inst_2")
	available2, _, missingGu2 := evaluateKillerMoveComponents(components, ownedSet)
	if available2 {
		t.Fatalf("expected available=false when component is missing, got true")
	}
	if len(missingGu2) != 1 || missingGu2[0] != "inst_2" {
		t.Fatalf("expected missing gu inst_2, got %v", missingGu2)
	}
}
