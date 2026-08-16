package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestCleanupInvalidEquipped(t *testing.T) {
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_STRENGTH": {
			1: {ID: "GU_STRENGTH", Name: "Lực Cổ", Rank: 1, Type: "Passive"},
			9: {ID: "GU_STRENGTH", Name: "Lực Cổ", Rank: 9, Type: "Passive"},
		},
		"GU_CONSUMABLE": {
			1: {ID: "GU_CONSUMABLE", Name: "Tiêu Hao Cổ", Rank: 1, Type: "Consumable"},
		},
		"GU_GROWTH": {
			9: {ID: "GU_GROWTH", Name: "Trưởng Thành Cổ", Rank: 9, Type: "Passive",
				Effects: models.GuEffects{Growth: &models.GrowthEffect{HoursToMax: 10, MaxAttack: 100}}},
		},
	}

	now := time.Now()
	hourAgo := now.Add(-time.Hour)
	p := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		GuInventory: models.GuInventory{
			{InstanceID: "over-rank", GuID: "GU_STRENGTH", Rank: 9, Status: "ACTIVE", Equipped: true},
			{InstanceID: "consumable", GuID: "GU_CONSUMABLE", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "unknown", GuID: "GU_MISSING", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "growth", GuID: "GU_GROWTH", Rank: 9, Status: "ACTIVE", Equipped: true, EquippedAt: &hourAgo},
			{InstanceID: "valid", GuID: "GU_STRENGTH", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "vital", GuID: "GU_STRENGTH", Rank: 9, Status: "ACTIVE", Equipped: true, IsVital: true},
			{InstanceID: "not-equipped", GuID: "GU_STRENGTH", Rank: 9, Status: "ACTIVE", Equipped: false},
		},
	}

	cleaned := cleanupInvalidEquipped(p)

	wantCleaned := map[string]bool{"over-rank": true, "consumable": true, "unknown": true, "growth": true}
	if len(cleaned) != len(wantCleaned) {
		t.Fatalf("expected %d cleaned Gus, got %d: %+v", len(wantCleaned), len(cleaned), cleaned)
	}
	for _, gu := range cleaned {
		if !wantCleaned[gu.InstanceID] {
			t.Fatalf("unexpected cleaned Gu %s", gu.InstanceID)
		}
		if gu.Equipped {
			t.Fatalf("cleaned Gu %s should be unequipped", gu.InstanceID)
		}
	}

	byID := map[string]models.GuInstance{}
	for _, gu := range p.GuInventory {
		byID[gu.InstanceID] = gu
	}

	for id := range wantCleaned {
		if byID[id].Equipped {
			t.Fatalf("Gu %s should have been unequipped", id)
		}
	}
	if !byID["valid"].Equipped {
		t.Fatalf("valid Gu should stay equipped")
	}
	if !byID["vital"].Equipped {
		t.Fatalf("vital Gu must never be auto-unequipped")
	}
	if byID["not-equipped"].Equipped {
		t.Fatalf("non-equipped Gu must not be touched")
	}

	if byID["growth"].EquippedAt != nil {
		t.Fatalf("growth Gu EquippedAt should be reset after cleanup")
	}
	if byID["growth"].GrowthHours < 0.9 {
		t.Fatalf("growth Gu should have accrued ~1h of growth, got %v", byID["growth"].GrowthHours)
	}
	if p.GrowthData.Boosts.Attack < 9 {
		t.Fatalf("growth flush should add ~10 attack, got %v", p.GrowthData.Boosts.Attack)
	}

	RebuildEquippedGus(p)
	wantEquipped := []string{"valid", "vital"}
	if len(p.EquippedGus) != len(wantEquipped) {
		t.Fatalf("expected equipped_gus %v, got %v", wantEquipped, p.EquippedGus)
	}
	for i, id := range wantEquipped {
		if p.EquippedGus[i] != id {
			t.Fatalf("expected equipped_gus[%d] = %s, got %s", i, id, p.EquippedGus[i])
		}
	}
}

func TestCleanupInvalidEquippedNoInvalidGus(t *testing.T) {
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_STRENGTH": {1: {ID: "GU_STRENGTH", Name: "Lực Cổ", Rank: 1, Type: "Passive"}},
	}
	p := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		GuInventory: models.GuInventory{
			{InstanceID: "a", GuID: "GU_STRENGTH", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "b", GuID: "GU_STRENGTH", Rank: 1, Status: "ACTIVE", Equipped: false},
		},
	}
	if cleaned := cleanupInvalidEquipped(p); len(cleaned) != 0 {
		t.Fatalf("expected no cleanup, got %+v", cleaned)
	}
}
