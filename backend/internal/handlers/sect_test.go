package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestVaultGuHungerHelpers(t *testing.T) {
	// Legacy missing hunger (0 or negative) should default to max (100)
	n1 := normalizeVaultGuHunger("GU_WOLF", 1, 0)
	if n1 != 100 {
		t.Fatalf("expected 100 for legacy zero hunger, got %f", n1)
	}

	// Existing positive hunger should be kept
	n2 := normalizeVaultGuHunger("GU_WOLF", 1, 80)
	if n2 != 80 {
		t.Fatalf("expected 80 for normal hunger, got %f", n2)
	}

	// Merging higher donated hunger should raise vault group hunger
	m1 := mergeVaultGuHunger("GU_WOLF", 1, 50, 90)
	if m1 != 90 {
		t.Fatalf("expected 90 when donated hunger (90) > current (50), got %f", m1)
	}

	// Merging lower donated hunger should keep current higher hunger
	m2 := mergeVaultGuHunger("GU_WOLF", 1, 80, 40)
	if m2 != 80 {
		t.Fatalf("expected 80 when current hunger (80) > donated (40), got %f", m2)
	}
}

func TestSectVaultWithdrawHunger(t *testing.T) {
	// Case 1: Normal Gu with hunger rate > 0 should have hunger = 1
	specNormal := models.Gu{
		ID:         "GU_NORMAL",
		Rank:       2,
		HungerRate: 3,
		MaxHunger:  100,
	}
	gu1 := createVaultWithdrawnGu("GU_NORMAL", 2, specNormal)
	if gu1.Hunger != 1 {
		t.Fatalf("expected withdrawn normal Gu hunger to be 1, got %d", gu1.Hunger)
	}
	if gu1.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %s", gu1.Status)
	}
	if gu1.GuID != "GU_NORMAL" || gu1.Rank != 2 {
		t.Fatalf("expected ID GU_NORMAL and rank 2, got ID %s rank %d", gu1.GuID, gu1.Rank)
	}
	if gu1.InstanceID == "" {
		t.Fatal("expected non-empty instance ID")
	}

	// Case 2: Consumable Gu (hunger rate == 0) retains max hunger
	specConsumable := models.Gu{
		ID:         "GU_RELIC",
		Rank:       1,
		HungerRate: 0,
		MaxHunger:  50,
	}
	gu2 := createVaultWithdrawnGu("GU_RELIC", 1, specConsumable)
	if gu2.Hunger != 50 {
		t.Fatalf("expected consumable Gu to have hunger 50, got %d", gu2.Hunger)
	}

	// Case 3: Consumable Gu with 0 max hunger defaults to 100
	specZeroMax := models.Gu{
		ID:         "GU_ZERO",
		Rank:       1,
		HungerRate: 0,
		MaxHunger:  0,
	}
	gu3 := createVaultWithdrawnGu("GU_ZERO", 1, specZeroMax)
	if gu3.Hunger != 100 {
		t.Fatalf("expected consumable Gu with 0 max to default to 100, got %d", gu3.Hunger)
	}
}

func TestIsInsideSquareTerritory(t *testing.T) {
	sx, sy := 50, 50
	radius := 6 // e.g. main_hall level 1 => 5 + 1 = 6

	tests := []struct {
		name     string
		x, y     int
		expected bool
	}{
		{"center", 50, 50, true},
		{"top edge", 50, 44, true},
		{"bottom edge", 50, 56, true},
		{"left edge", 44, 50, true},
		{"right edge", 56, 50, true},
		{"top-left corner", 44, 44, true},
		{"bottom-right corner", 56, 56, true},
		{"just outside right", 57, 50, false},
		{"just outside top", 50, 43, false},
		{"just outside corner", 57, 57, false},
		{"far away", 100, 100, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsInsideSquareTerritory(tc.x, tc.y, sx, sy, radius)
			if actual != tc.expected {
				t.Errorf("at (%d, %d): expected %v, got %v", tc.x, tc.y, tc.expected, actual)
			}
		})
	}
}
