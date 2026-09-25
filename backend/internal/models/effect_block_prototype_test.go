package models_test

import (
	"encoding/json"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestModularEffectBlockUnmarshal(t *testing.T) {
	rawJSON := `{
		"blocks": [
			{
				"id": "moonlight_slash",
				"trigger": "ON_ACTION",
				"target": { "selector": "TARGET", "bypass_frontline": true },
				"operations": [
					{ "op": "DAMAGE", "sub_type": "FLAT", "value": 150.0, "path": "Moon Path" },
					{ "op": "STATUS", "status_id": "BLEED", "value": 25.0, "turns": 3, "chance": 0.6 }
				]
			},
			{
				"id": "moon_cloak_passive",
				"trigger": "ALWAYS_ON",
				"target": { "selector": "SELF" },
				"operations": [
					{ "op": "STAT_MOD", "stat": "DEFENSE", "sub_type": "ADD_FLAT", "value": 40.0 }
				]
			}
		]
	}`

	var effects models.GuEffects
	if err := json.Unmarshal([]byte(rawJSON), &effects); err != nil {
		t.Fatalf("Failed to unmarshal modular effect blocks: %v", err)
	}

	if len(effects.Blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(effects.Blocks))
	}

	b1 := effects.Blocks[0]
	if b1.Trigger != gameplayeffects.TriggerOnAction || b1.Target.Selector != gameplayeffects.TargetSingle || !b1.Target.BypassFrontline {
		t.Errorf("Block 1 metadata mismatch: %+v", b1)
	}
	if len(b1.Operations) != 2 {
		t.Fatalf("Expected 2 operations in block 1, got %d", len(b1.Operations))
	}
	if b1.Operations[0].Op != gameplayeffects.OpDamage || b1.Operations[0].Value != 150.0 {
		t.Errorf("Block 1 Op 0 mismatch: %+v", b1.Operations[0])
	}
	if b1.Operations[1].Op != gameplayeffects.OpStatus || b1.Operations[1].StatusID != "BLEED" || b1.Operations[1].Chance != 0.6 {
		t.Errorf("Block 1 Op 1 mismatch: %+v", b1.Operations[1])
	}

	b2 := effects.Blocks[1]
	if b2.Trigger != gameplayeffects.TriggerAlwaysOn || b2.Target.Selector != gameplayeffects.TargetSelf {
		t.Errorf("Block 2 metadata mismatch: %+v", b2)
	}
	if b2.Operations[0].Op != gameplayeffects.OpStatMod || b2.Operations[0].Stat != "DEFENSE" || b2.Operations[0].Value != 40.0 {
		t.Errorf("Block 2 Op 0 mismatch: %+v", b2.Operations[0])
	}
}

func TestCompileLegacyToBlocks(t *testing.T) {
	legacyActive := &models.ActiveEffect{
		DamageFlat:        200,
		ShieldFlat:        50,
		DisableGuNextTurn: true,
		CurveAttack:       true,
		ApplyStatus:       models.StatusApplications{{ID: "POISON", Value: 30, Turns: 2, Chance: 0.8}},
	}
	legacyPassive := &models.PassiveEffect{
		StatBaseAttack:  15,
		StatBaseDefense: 20,
	}
	legacyOther := []models.SpecialMechanic{
		{EventTrigger: "ON_DEATH", ActionCode: "REBORN", Params: []byte(`{"heal_percent": 0.3}`)},
	}

	blocks := models.CompileLegacyToBlocks(legacyActive, legacyPassive, legacyOther)
	if len(blocks) != 3 {
		t.Fatalf("Expected 3 compiled blocks, got %d", len(blocks))
	}

	// Block 0: Passive
	if blocks[0].Trigger != gameplayeffects.TriggerAlwaysOn || len(blocks[0].Operations) != 2 {
		t.Errorf("Passive block mismatch: %+v", blocks[0])
	}
	// Block 1: Active
	if blocks[1].Trigger != gameplayeffects.TriggerOnAction || !blocks[1].Target.BypassFrontline || len(blocks[1].Operations) != 4 {
		t.Errorf("Active block mismatch: %+v (ops=%d)", blocks[1], len(blocks[1].Operations))
	}
	// Block 2: Other (Reborn)
	if blocks[2].Trigger != gameplayeffects.TriggerOnDeath || blocks[2].Operations[0].Code != "REBORN" {
		t.Errorf("Other block mismatch: %+v", blocks[2])
	}
}
