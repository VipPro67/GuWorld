package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestLootEntryEligible(t *testing.T) {
	cases := []struct {
		name string
		e    models.LootEntry
		rank int
		want bool
	}{
		{"rank match", models.LootEntry{Rank: 4}, 4, true},
		{"rank mismatch", models.LootEntry{Rank: 5}, 4, false},
		{"universal", models.LootEntry{Rank: 0}, 4, true},
		{"jackpot bypasses mismatch", models.LootEntry{Rank: 9, Jackpot: true}, 4, true},
		{"jackpot at own rank", models.LootEntry{Rank: 9, Jackpot: true}, 9, true},
	}
	for _, c := range cases {
		if got := lootEntryEligible(c.e, c.rank); got != c.want {
			t.Errorf("%s: lootEntryEligible(%+v, %d) = %v, want %v", c.name, c.e, c.rank, got, c.want)
		}
	}
}

func TestRollLootRankFilter(t *testing.T) {
	lt := models.LootTable{
		ID: "test",
		Entries: []models.LootEntry{
			{ItemID: "match", Rank: 4, MinQty: 1, MaxQty: 1, Chance: 1.0},
			{ItemID: "mismatch", Rank: 5, MinQty: 1, MaxQty: 1, Chance: 1.0},
			{ItemID: "universal", Rank: 0, MinQty: 1, MaxQty: 1, Chance: 1.0},
			{ItemID: "jackpot", Rank: 9, Jackpot: true, MinQty: 1, MaxQty: 1, Chance: 1.0},
			{ItemID: "zero_chance", Rank: 4, MinQty: 1, MaxQty: 1, Chance: 0.0},
		},
	}
	got := RollLoot(lt, 4)
	for _, want := range []string{"match", "universal", "jackpot"} {
		if got.Items[want] <= 0 {
			t.Errorf("RollLoot(rank 4) missing entry %q", want)
		}
	}
	for _, miss := range []string{"mismatch", "zero_chance"} {
		if got.Items[miss] > 0 {
			t.Errorf("RollLoot(rank 4) should not drop %q", miss)
		}
	}
}

// Region loot is scoped by the region rule's rank range, so its entries must
// roll regardless of the per-entry rank tag; otherwise a region-wide rule with
// single-rank entries would drop nothing at the other ranks it covers.
func TestRollRegionLootTableIgnoresEntryRank(t *testing.T) {
	lt := models.LootTable{
		ID: "region_test",
		Entries: []models.LootEntry{
			{ItemID: "signature", Rank: 3, MinQty: 1, MaxQty: 1, Chance: 1.0},
		},
	}
	got := rollRegionLootTable(lt, 5)
	if got.Items["signature"] <= 0 {
		t.Errorf("region loot (rank 5) should still drop rank-3-tagged entry, got %+v", got.Items)
	}
	// The global roll keeps the rank gate for common pool / per-monster tables.
	if RollLoot(lt, 5).Items["signature"] > 0 {
		t.Errorf("RollLoot should keep the rank gate")
	}
}

func TestGetGuDropChancePercentTiers(t *testing.T) {
	for rank := 1; rank <= 5; rank++ {
		boss := GetGuDropChancePercent(rank, "boss")
		elite := GetGuDropChancePercent(rank, "elite")
		normal := GetGuDropChancePercent(rank, "normal")
		if !(boss > elite) {
			t.Errorf("rank %d: expected boss (%.2f) > elite (%.2f)", rank, boss, elite)
		}
		if !(elite > normal) {
			t.Errorf("rank %d: expected elite (%.2f) > normal (%.2f)", rank, elite, normal)
		}
	}
	// Unknown tiers fall back to the elite table.
	if got, want := GetGuDropChancePercent(3, "weird"), GetGuDropChancePercent(3, "elite"); got != want {
		t.Errorf("unknown tier = %.2f, want elite %.2f", got, want)
	}
}

func TestGuDropTier(t *testing.T) {
	cases := map[string]string{
		"Normal":           "normal",
		"MONSTER":          "normal",
		"Elite":            "elite",
		"":                 "elite",
		"Boss":             "boss",
		"World_Boss":       "boss",
		"Tribulation_Boss": "boss",
	}
	for in, want := range cases {
		if got := guDropTier(in); got != want {
			t.Errorf("guDropTier(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRegionMonsterGuPoolInheritance(t *testing.T) {
	zoneReg.mu.Lock()
	prevZones, prevByID := zoneReg.zones, zoneReg.byID
	zoneReg.zones = []models.Region{
		{ID: "parent", PoolWeights: models.ZonePools{MonsterGus: []models.GuPoolEntry{
			{GuID: "parent_gu", Rank: 1},
			{GuID: "shared_gu", Rank: 1},
		}}},
		{ID: "child", ParentID: "parent", PoolWeights: models.ZonePools{MonsterGus: []models.GuPoolEntry{
			{GuID: "shared_gu", Rank: 2},
			{GuID: "child_gu", Rank: 1},
		}}},
	}
	zoneReg.byID = map[string]*models.Region{
		"parent": &zoneReg.zones[0],
		"child":  &zoneReg.zones[1],
	}
	zoneReg.mu.Unlock()
	defer func() {
		zoneReg.mu.Lock()
		zoneReg.zones, zoneReg.byID = prevZones, prevByID
		zoneReg.mu.Unlock()
	}()

	pool := regionMonsterGuPool("child")
	got := map[string]int{}
	for _, e := range pool {
		got[e.GuID] = e.Rank
	}
	if len(pool) != 3 {
		t.Fatalf("expected 3 unique Gu (child overrides parent on shared), got %d: %+v", len(pool), pool)
	}
	if got["shared_gu"] != 2 {
		t.Errorf("child entry should win for shared_gu, got rank %d", got["shared_gu"])
	}
	if _, ok := got["child_gu"]; !ok {
		t.Errorf("missing child_gu in pool")
	}
	if _, ok := got["parent_gu"]; !ok {
		t.Errorf("missing inherited parent_gu in pool")
	}
}

func TestRegionMonsterGuPoolEmptyRegion(t *testing.T) {
	if pool := regionMonsterGuPool(""); pool != nil {
		t.Errorf("expected nil pool for empty region, got %+v", pool)
	}
}

// Looted Tiên Cổ must reach the world announcement queue, while mortal Gu must
// not. Regression guard for CREATED_LOOT rank 6 being logged with no broadcast.
func TestQueueImmortalGuLootNotice(t *testing.T) {
	prev := models.GuRegistry
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_MORTAL_ONLY":    {3: {ID: "GU_MORTAL_ONLY", Rank: 3}},
		"GU_MIXED_VARIANTS": {5: {ID: "GU_MIXED_VARIANTS", Rank: 5}, 7: {ID: "GU_MIXED_VARIANTS", Rank: 7}},
	}
	t.Cleanup(func() { models.GuRegistry = prev })

	playerID := uuid.New()
	boss := "Thiên Ma"

	// Rolled rank 6 -> queued with the rolled rank.
	got := queueImmortalGuLootNotice(nil, playerID, "A", boss, "GU_RANK6", 6)
	if len(got) != 1 {
		t.Fatalf("rank 6 Gu: got %d announcements, want 1", len(got))
	}
	if got[0].Rank != 6 || got[0].GuID != "GU_RANK6" || got[0].PlayerID != playerID || got[0].BossName != boss || got[0].PlayerName != "A" {
		t.Errorf("rank 6 announcement = %+v, want the looted Gu/player/boss", got[0])
	}

	// Mortal-only Gu -> never announced.
	if got = queueImmortalGuLootNotice(nil, playerID, "A", boss, "GU_MORTAL_ONLY", 3); len(got) != 0 {
		t.Errorf("mortal Gu: got %d announcements, want 0", len(got))
	}

	// Gu id with both mortal and immortal variants but a mortal rolled rank is
	// still a Tiên Cổ; announce the immortal rank, not the rolled one.
	got = queueImmortalGuLootNotice(nil, playerID, "A", boss, "GU_MIXED_VARIANTS", 5)
	if len(got) != 1 {
		t.Fatalf("mixed-variant Gu: got %d announcements, want 1", len(got))
	}
	if got[0].Rank != 7 {
		t.Errorf("mixed-variant announcement rank = %d, want 7", got[0].Rank)
	}

	// A second loot appends instead of replacing.
	got = queueImmortalGuLootNotice(got, playerID, "B", boss, "GU_RANK6", 6)
	if len(got) != 2 {
		t.Errorf("append: got %d announcements, want 2", len(got))
	}
}
