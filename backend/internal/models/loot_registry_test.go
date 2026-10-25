package models

import (
	"testing"
)

func TestLootRegistry(t *testing.T) {
	lootMu.Lock()
	origTables := LootTableRegistry
	origRules := RegionLootRegistry
	origAncestors := regionAncestors

	LootTableRegistry = map[string]LootTable{
		"loot_wolf": {
			ID:   "loot_wolf",
			Name: "Loot Sói",
			Entries: []LootEntry{
				{ItemID: "wolf_fang", Chance: 0.5, MinQty: 1, MaxQty: 2, Rank: 1},
			},
		},
	}
	RegionLootRegistry = map[string][]RegionLootRule{
		"thanh_mao_son": {
			{
				RegionID:    "thanh_mao_son",
				LootTableID: "loot_thanh_mao_grass",
				MonsterType: "Beast",
				MinRank:     1,
				MaxRank:     3,
				Chance:      0.8,
				IsActive:    true,
			},
		},
		"southern_border": {
			{
				RegionID:    "southern_border",
				LootTableID: "loot_southern_soil",
				MonsterType: "Beast",
				MinRank:     1,
				MaxRank:     5,
				Chance:      0.4,
				IsActive:    true,
			},
		},
	}
	regionAncestors = map[string][]string{
		"thanh_mao_son": {"thanh_mao_son", "southern_border"},
	}
	lootMu.Unlock()

	defer func() {
		lootMu.Lock()
		LootTableRegistry = origTables
		RegionLootRegistry = origRules
		regionAncestors = origAncestors
		lootMu.Unlock()
	}()

	// 1. GetLootTable
	lt, ok := GetLootTable("loot_wolf")
	if !ok || lt.Name != "Loot Sói" {
		t.Fatalf("expected to find loot_wolf, got ok=%v lt=%+v", ok, lt)
	}
	if _, ok := GetLootTable("non_existent"); ok {
		t.Fatalf("expected non_existent to fail")
	}

	// 2. AllLootTables
	all := AllLootTables()
	if len(all) == 0 {
		t.Fatalf("expected non-empty all loot tables")
	}

	// 3. GetRegionLootRules with ancestry
	rules := GetRegionLootRules("thanh_mao_son", "beast", 2)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules (thanh_mao_son + southern_border), got %d", len(rules))
	}
	if rules[0].LootTableID != "loot_thanh_mao_grass" || rules[1].LootTableID != "loot_southern_soil" {
		t.Fatalf("unexpected rules returned: %+v", rules)
	}

	// 4. Rank out of bounds for subregion
	rulesHighRank := GetRegionLootRules("thanh_mao_son", "beast", 5)
	if len(rulesHighRank) != 1 || rulesHighRank[0].LootTableID != "loot_southern_soil" {
		t.Fatalf("expected only southern_border for rank 5, got %+v", rulesHighRank)
	}

	// 5. Monster type mismatch
	rulesOtherType := GetRegionLootRules("thanh_mao_son", "humanoid", 2)
	if len(rulesOtherType) != 0 {
		t.Fatalf("expected 0 rules for humanoid, got %d", len(rulesOtherType))
	}
}
