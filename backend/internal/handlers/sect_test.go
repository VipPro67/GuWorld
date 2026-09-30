package handlers

import (
	"encoding/json"
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

func TestTerritoriesOverlap(t *testing.T) {
	tests := []struct {
		name     string
		x1, y1   int
		r1       int
		x2, y2   int
		r2       int
		expected bool
	}{
		{
			name:     "identical center",
			x1: 50, y1: 50, r1: 6,
			x2: 50, y2: 50, r2: 6,
			expected: true,
		},
		{
			name:     "far away no overlap",
			x1: 20, y1: 20, r1: 6,
			x2: 80, y2: 80, r2: 6,
			expected: false,
		},
		{
			name:     "adjacent sharing border tiles dx = r1+r2 dy = 0",
			x1: 50, y1: 50, r1: 6,
			x2: 62, y2: 50, r2: 6, // dx = 12 <= 6+6 => overlap
			expected: true,
		},
		{
			name:     "just 1 tile beyond border dx = r1+r2+1",
			x1: 50, y1: 50, r1: 6,
			x2: 63, y2: 50, r2: 6, // dx = 13 > 12 => separated
			expected: false,
		},
		{
			name:     "separated in Y even if X overlaps",
			x1: 50, y1: 50, r1: 6,
			x2: 50, y2: 70, r2: 6, // dx = 0, dy = 20 > 12 => separated
			expected: false,
		},
		{
			name:     "separated in X even if Y overlaps",
			x1: 50, y1: 50, r1: 6,
			x2: 70, y2: 50, r2: 6, // dx = 20 > 12, dy = 0 => separated
			expected: false,
		},
		{
			name:     "different radii overlap",
			x1: 30, y1: 30, r1: 10, // Main Hall lvl 5 (radius 10)
			x2: 45, y2: 30, r2: 6,  // Main Hall lvl 1 (radius 6) -> dx = 15 <= 16
			expected: true,
		},
		{
			name:     "different radii separated",
			x1: 30, y1: 30, r1: 10,
			x2: 47, y2: 30, r2: 6, // dx = 17 > 16
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := TerritoriesOverlap(tc.x1, tc.y1, tc.r1, tc.x2, tc.y2, tc.r2)
			if actual != tc.expected {
				t.Errorf("TerritoriesOverlap(%d,%d,%d vs %d,%d,%d): expected %v, got %v",
					tc.x1, tc.y1, tc.r1, tc.x2, tc.y2, tc.r2, tc.expected, actual)
			}
		})
	}
}

func TestSectVaultJSONSerialization(t *testing.T) {
	// Verify SectVaultGuItem structure serialization matches client expectations
	guItem := SectVaultGuItem{
		ID:       "gu_xuan_bang",
		Rank:     3,
		Cost:     150,
		Quantity: 5,
		Hunger:   85.5,
	}
	bytes, err := json.Marshal(guItem)
	if err != nil {
		t.Fatalf("failed to marshal guItem: %v", err)
	}
	if string(bytes) == "" {
		t.Fatal("expected non-empty json")
	}
}

func TestPlanVaultGuDeductionsMultiRow(t *testing.T) {
	// Scenario: Player wants to buy 3 Gu, but vault has 3 separate rows with quantities [1, 1, 2]
	rows := []VaultGuRow{
		{ID: "row1", GuID: "GU_WOLF", Rank: 1, Cost: 100, Quantity: 1},
		{ID: "row2", GuID: "GU_WOLF", Rank: 1, Cost: 100, Quantity: 1},
		{ID: "row3", GuID: "GU_WOLF", Rank: 1, Cost: 120, Quantity: 2},
	}

	plans, totalCost, err := planVaultGuDeductions(rows, 3)
	if err != nil {
		t.Fatalf("expected plan to succeed, got error: %v", err)
	}

	if len(plans) != 3 {
		t.Fatalf("expected 3 deduction plans, got %d", len(plans))
	}

	// Row 1: take 1, deleted
	if plans[0].ID != "row1" || plans[0].Deduct != 1 || !plans[0].Del {
		t.Errorf("row1 plan mismatch: %+v", plans[0])
	}
	// Row 2: take 1, deleted
	if plans[1].ID != "row2" || plans[1].Deduct != 1 || !plans[1].Del {
		t.Errorf("row2 plan mismatch: %+v", plans[1])
	}
	// Row 3: take 1, not deleted (1 left)
	if plans[2].ID != "row3" || plans[2].Deduct != 1 || plans[2].Del {
		t.Errorf("row3 plan mismatch: %+v", plans[2])
	}

	// Cost: 100*1 + 100*1 + 120*1 = 320
	if totalCost != 320 {
		t.Errorf("expected totalCost 320, got %d", totalCost)
	}
}

func TestPlanVaultGuDeductionsInsufficient(t *testing.T) {
	// Total available is 2, player requests 3
	rows := []VaultGuRow{
		{ID: "row1", GuID: "GU_WOLF", Rank: 1, Cost: 100, Quantity: 1},
		{ID: "row2", GuID: "GU_WOLF", Rank: 1, Cost: 100, Quantity: 1},
	}

	_, _, err := planVaultGuDeductions(rows, 3)
	if err == nil {
		t.Fatal("expected error for insufficient vault quantity, got nil")
	}
}

func TestPlanVaultGuDeductionsImmortalSingleOnly(t *testing.T) {
	// Immortal Gu (Rank >= 6) must only be withdrawn 1 at a time
	rows := []VaultGuRow{
		{ID: "row1", GuID: "GU_SPRING_AUTUMN", Rank: 6, Cost: 5000, Quantity: 1, InstanceID: "inst-1"},
	}

	// Buying 1 should succeed
	plans, cost, err := planVaultGuDeductions(rows, 1)
	if err != nil {
		t.Fatalf("expected immortal gu qty 1 to succeed, got %v", err)
	}
	if len(plans) != 1 || cost != 5000 {
		t.Fatalf("unexpected plan or cost: %+v, %d", plans, cost)
	}

	// Buying > 1 should fail
	_, _, err = planVaultGuDeductions(rows, 2)
	if err == nil {
		t.Fatal("expected error when buying immortal gu > 1, got nil")
	}
}
