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
	if prof.CooldownTurns != 1 {
		t.Fatalf("expected cooldown 1, got %d", prof.CooldownTurns)
	}
	if prof.WarmupTurns != 0 {
		t.Fatalf("expected warmup 0 for rank 3, got %d", prof.WarmupTurns)
	}
	if prof.TargetType != "SINGLE" {
		t.Fatalf("expected target SINGLE, got %s", prof.TargetType)
	}
}
