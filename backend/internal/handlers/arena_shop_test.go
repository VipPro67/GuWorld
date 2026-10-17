package handlers

import (
	"fmt"
	"testing"
)

func TestSelectArenaRandomResources(t *testing.T) {
	pool := make([]ShopItem, 100)
	for i := range pool {
		pool[i] = ShopItem{ID: fmt.Sprintf("res_%03d", i), Rank: 6}
	}

	got := selectArenaRandomResources(pool)
	if len(got) != maxArenaRandomResources {
		t.Fatalf("expected %d resources, got %d", maxArenaRandomResources, len(got))
	}

	inPool := make(map[string]bool, len(pool))
	for _, it := range pool {
		inPool[it.ID] = true
	}
	for _, it := range got {
		if !inPool[it.ID] {
			t.Fatalf("selected id %q not in pool", it.ID)
		}
	}

	// Stable within the same rotation window (same hourly seed).
	again := selectArenaRandomResources(pool)
	for i := range got {
		if got[i].ID != again[i].ID {
			t.Fatalf("selection not stable: %q != %q", got[i].ID, again[i].ID)
		}
	}
}

func TestSelectArenaRandomResourcesSmallPool(t *testing.T) {
	pool := []ShopItem{{ID: "a"}, {ID: "b"}}
	if got := selectArenaRandomResources(pool); len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
}
