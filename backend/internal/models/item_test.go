package models

import (
	"encoding/json"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func TestItemSpecCompiledProjection(t *testing.T) {
	raw := []byte(`[{"id":"HEAL","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}}]}]`)
	var item ItemSpec
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}
	item.Blocks = blocks
	if err := item.Effects.Scan(raw); err != nil {
		t.Fatal(err)
	}

	compiled := item.Compiled()
	if compiled == nil {
		t.Fatal("expected item.Compiled() to be non-nil")
	}

	// Verify legacy map was also populated
	if item.Effects["restore_hp"] != 50 {
		t.Fatalf("expected item.Effects['restore_hp'] == 50, got %v", item.Effects["restore_hp"])
	}

	// Evaluate ON_CONSUME trigger
	res, err := compiled.Evaluate(gameplayeffects.Context{}, gameplayeffects.Event{Trigger: gameplayeffects.TriggerOnConsume})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(res.Commands))
	}
	if res.Commands[0].Code != "RESTORE_HP" {
		t.Fatalf("expected command code RESTORE_HP, got %s", res.Commands[0].Code)
	}
}

func TestValidateItemEffectBlocksRejectsCombatTriggers(t *testing.T) {
	invalidJSON := []byte(`[{"id":"ATTACK","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10}]}]`)
	if err := ValidateItemEffectBlocks(invalidJSON); err == nil {
		t.Fatal("expected ValidateItemEffectBlocks to reject ON_ACTION trigger in item definition")
	}
}

func TestValidateItemEffectBlocksAcceptsValidItem(t *testing.T) {
	validJSON := []byte(`[{"id":"HP_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":30}}]}]`)
	if err := ValidateItemEffectBlocks(validJSON); err != nil {
		t.Fatalf("expected valid item block to pass, got: %v", err)
	}
}
