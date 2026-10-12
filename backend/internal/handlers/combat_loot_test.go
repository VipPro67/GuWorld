package handlers

import (
	"testing"

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