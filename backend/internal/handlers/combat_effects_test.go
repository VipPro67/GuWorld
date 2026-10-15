package handlers

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestCalculateGuEffectForCombatant_ShieldFlat(t *testing.T) {
	actor := &models.Combatant{
		Rank:        1,
		BaseDefense: 10,
		PathMasteries: map[string]int{
			"blade": 0, // Level 0 -> powerScale = 0.70
		},
	}

	// 1. Gu with Blocks (OpShield, FLAT = 20)
	guBlock := &models.GuDictionaryItem{
		ID:   "GU_BLADE_SHIELD_1",
		Rank: 1,
		Path: []string{"Blade Path"},
		Effects: models.GuEffects{
			Blocks: []gameplayeffects.Block{
				{
					Trigger: gameplayeffects.TriggerOnAction,
					Operations: []gameplayeffects.Operation{
						{
							Op:      gameplayeffects.OpShield,
							SubType: "FLAT",
							Value:   20,
						},
					},
				},
			},
		},
	}

	// applyDao=false (used by simulator to get raw Gu stats): returns baseDef = 20.0
	_, defRaw, _, err := CalculateGuEffectForCombatant(actor, guBlock, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defRaw != 20.0 {
		t.Fatalf("expected raw def = 20.0, got %.2f", defRaw)
	}

	// Attainment scale for level 0 is 1.0 (Ordinary)
	scale, reason := GetGuAttainmentScale(actor, guBlock)
	if scale != 1.0 || reason != "" {
		t.Fatalf("expected scale 1.0 and empty reason for level 0, got scale=%.2f, reason=%q", scale, reason)
	}

	// applyDao=true with level 0: returns baseDef * 1.0 = 20.0
	_, defScaled, _, err := CalculateGuEffectForCombatant(actor, guBlock, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defScaled != 20.0 {
		t.Fatalf("expected scaled def = 20.0, got %.2f", defScaled)
	}

	// Attainment scale for level 1 (Master, >= 10,000 points) is 1.10
	actorMaster := &models.Combatant{
		Rank: 1,
		PathMasteries: map[string]int{
			"blade": 10000,
		},
	}
	scaleMaster, reasonMaster := GetGuAttainmentScale(actorMaster, guBlock)
	if scaleMaster != 1.10 || reasonMaster == "" {
		t.Fatalf("expected scale 1.10 for Master, got %.2f, reason=%q", scaleMaster, reasonMaster)
	}

	// 2. Gu with legacy Active (ShieldFlat = 20)
	guActive := &models.GuDictionaryItem{
		ID:   "GU_LEGACY_SHIELD_1",
		Rank: 1,
		Path: []string{"Blade Path"},
		Effects: models.GuEffects{
			Active: &models.ActiveEffect{
				ShieldFlat: 20,
			},
		},
	}

	_, defActiveRaw, _, err := CalculateGuEffectForCombatant(actor, guActive, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defActiveRaw != 20.0 {
		t.Fatalf("expected legacy raw def = 20.0, got %.2f", defActiveRaw)
	}
}
