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

func TestValidateItemEffectBlocksAcceptsTypedItemPrimitives(t *testing.T) {
	raw := []byte(`[{"id":"ITEM_EFFECTS","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"GRANT_RESOURCE","resource":"HUMAN_QI","value":50000},{"op":"RESTORE_RESOURCE","resource":"EARTH_QI","value":30000},{"op":"RESTORE_RESOURCE","resource":"HEAVEN_QI","value":30000},{"op":"MODIFY_APERTURE","sub_type":"AREA_SIZE","value":500000},{"op":"MODIFY_APERTURE","sub_type":"TIME_RATE","value":2},{"op":"RANDOMIZE_TRAIT","sub_type":"CONSTITUTION"},{"op":"RANDOMIZE_TRAIT","sub_type":"RACE"},{"op":"REPAIR_APERTURE","value":10000},{"op":"OPEN_REWARD_CONTAINER","sub_type":"MORTAL_MINE"},{"op":"OPEN_REWARD_CONTAINER","sub_type":"IMMORTAL_MINE"},{"op":"PLACE_WORLD_OBJECT","sub_type":"PRIMEVAL_SPRING"}]}]`)
	if err := ValidateItemEffectBlocks(raw); err != nil {
		t.Fatalf("typed qi resource item rejected: %v", err)
	}
	var item ItemSpec
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}
	item.Blocks = blocks
	if err := item.Effects.Scan(raw); err != nil {
		t.Fatalf("scan typed qi resource item: %v", err)
	}
	commands := item.EffectCommands()
	if len(commands) != 11 || commands[0].Code != "heal_aperture" || commands[1].Code != "increase_aperture_area" || commands[2].Code != "increase_aperture_time_rate" || commands[3].Code != "increase_human_qi" || commands[4].Code != "open_immortal_mine_chest" || commands[5].Code != "open_mine_chest" || commands[6].Code != "random_constitution" || commands[7].Code != "random_race" || commands[8].Code != "restore_earth_qi" || commands[9].Code != "restore_heaven_qi" || commands[10].Code != "spring_egg_hatch" {
		t.Fatalf("typed resource commands = %+v", commands)
	}
	for key, want := range map[string]float64{"heal_aperture": 10000, "increase_aperture_area": 500000, "increase_human_qi": 50000, "restore_earth_qi": 30000, "restore_heaven_qi": 30000, "increase_aperture_time_rate": 2, "random_constitution": 1, "random_race": 1, "open_immortal_mine_chest": 1, "open_mine_chest": 1, "spring_egg_hatch": 1} {
		if got := item.Effects[key]; got != want {
			t.Errorf("legacy projection %s = %v, want %v", key, got, want)
		}
	}
}
