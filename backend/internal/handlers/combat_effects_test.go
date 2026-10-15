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

	_, def, _, err := CalculateGuEffectForCombatant(actor, guBlock, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: baseDef (20) * powerScale (0.70) = 14.0
	expectedDef := 20.0 * 0.70
	if def < expectedDef-0.001 || def > expectedDef+0.001 {
		t.Fatalf("expected effectiveDefense ~ %.2f, got %.2f", expectedDef, def)
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

	_, defActive, _, err := CalculateGuEffectForCombatant(actor, guActive, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if defActive < expectedDef-0.001 || defActive > expectedDef+0.001 {
		t.Fatalf("expected legacy effectiveDefense ~ %.2f, got %.2f", expectedDef, defActive)
	}
}
