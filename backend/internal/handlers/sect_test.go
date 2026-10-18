package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

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

func TestStockVaultRejectsConsumableAndCurrency(t *testing.T) {
	// Register mock items in ItemRegistry
	models.ItemRegistry["test_consumable_item"] = models.ItemSpec{
		ID:       "test_consumable_item",
		Name:     "Test Pill",
		Type:     "Consumable",
		ItemType: "Consumable",
		Rank:     1,
	}
	models.ItemRegistry["test_currency_item"] = models.ItemSpec{
		ID:       "test_currency_item",
		Name:     "Test Coin",
		Type:     "Currency",
		ItemType: "Currency",
		Rank:     1,
	}
	models.ItemRegistry["test_material_item"] = models.ItemSpec{
		ID:        "test_material_item",
		Name:      "Test Ore",
		Type:      "Material",
		ItemType:  "Material",
		Rank:      1,
		BasePrice: 2.0,
	}

	// 1. Direct itemType consumable
	_, _, code, _ := stockItemUnitCost(nil, 10, "consumable", "test_consumable_item")
	if code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for consumable itemType, got %d", code)
	}

	// 2. Direct itemType currency
	_, _, code, _ = stockItemUnitCost(nil, 10, "currency", "test_currency_item")
	if code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for currency itemType, got %d", code)
	}

	// 3. itemType material but item is Consumable in dictionary
	_, _, code, _ = stockItemUnitCost(nil, 10, "material", "test_consumable_item")
	if code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for material that is Consumable, got %d", code)
	}

	// 4. itemType material but item is Currency in dictionary
	_, _, code, _ = stockItemUnitCost(nil, 10, "material", "test_currency_item")
	if code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for material that is Currency, got %d", code)
	}

	// 5. Normal material item should succeed and have 1.5x price
	unitCost, effRank, code, msg := stockItemUnitCost(nil, 10, "material", "test_material_item")
	if code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid Material item, got %d (msg: %s)", code, msg)
	}
	if effRank != 1 {
		t.Fatalf("expected effective rank 1, got %d", effRank)
	}
	// ItemBasePriceByRank(1) = 2.0 -> 2.0 * 1.5 = 3
	if unitCost != 3 {
		t.Fatalf("expected unitCost 3 (1.5x base price 2), got %d", unitCost)
	}

	// 6. Immortal material (Rank 6): requires sect level >= 50
	models.ItemRegistry["test_immortal_item"] = models.ItemSpec{
		ID:        "test_immortal_item",
		Name:      "Test Immortal Material",
		Type:      "Material",
		ItemType:  "Material",
		Rank:      6,
		BasePrice: 200.0,
	}
	// Level 40 sect cannot stock Rank 6
	_, _, code, _ = stockItemUnitCost(nil, 40, "material", "test_immortal_item")
	if code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Rank 6 material on sect level 40, got %d", code)
	}
	// Level 50 sect can stock Rank 6
	unitCost6, effRank6, code, msg := stockItemUnitCost(nil, 50, "material", "test_immortal_item")
	if code != http.StatusOK {
		t.Fatalf("expected 200 OK for Rank 6 material on sect level 50, got %d (msg: %s)", code, msg)
	}
	if effRank6 != 6 {
		t.Fatalf("expected effective rank 6, got %d", effRank6)
	}
	// ItemBasePriceByRank(6) = 200.0 -> * 1000 * 1.5 = 300000
	if unitCost6 != 300000 {
		t.Fatalf("expected unitCost6 300000, got %d", unitCost6)
	}
}

// Only Chưởng Môn and Hộ Pháp may destroy vault items; elders must not.
func TestCanDestroyVaultItem(t *testing.T) {
	allowed := []string{"CHUONG_MON", "HO_PHAP"}
	for _, rank := range allowed {
		if !canDestroyVaultItem(rank) {
			t.Fatalf("expected %s to be allowed to destroy vault items", rank)
		}
	}
	denied := []string{"TRUONG_LAO", "CHAN_TRUYEN", "TINH_ANH", "NOI_MON", "NGOAI_MON", ""}
	for _, rank := range denied {
		if canDestroyVaultItem(rank) {
			t.Fatalf("expected %s to be denied from destroying vault items", rank)
		}
	}
}

func TestStockSectVaultGatingAndPricing(t *testing.T) {
	// Register mock items for ranks 1 through 9
	for r := 1; r <= 9; r++ {
		models.ItemRegistry[fmt.Sprintf("mock_mat_r%d", r)] = models.ItemSpec{
			ID:        fmt.Sprintf("mock_mat_r%d", r),
			Name:      fmt.Sprintf("Mock Material R%d", r),
			Type:      "Material",
			ItemType:  "Material",
			Rank:      r,
			BasePrice: float64(r * 10),
		}
	}

	// Level 0 sect: maxLevel 1
	_, effRank, code, _ := stockItemUnitCost(nil, 0, "material", "mock_mat_r1")
	if code != http.StatusOK || effRank != 1 {
		t.Fatalf("expected level 0 sect to allow rank 1, got code=%d rank=%d", code, effRank)
	}
	_, _, code, _ = stockItemUnitCost(nil, 0, "material", "mock_mat_r2")
	if code != http.StatusForbidden {
		t.Fatalf("expected level 0 sect to reject rank 2, got code=%d", code)
	}

	// Level 10 sect: maxLevel 2
	_, effRank, code, _ = stockItemUnitCost(nil, 10, "material", "mock_mat_r2")
	if code != http.StatusOK || effRank != 2 {
		t.Fatalf("expected level 10 sect to allow rank 2, got code=%d rank=%d", code, effRank)
	}
	_, _, code, _ = stockItemUnitCost(nil, 10, "material", "mock_mat_r3")
	if code != http.StatusForbidden {
		t.Fatalf("expected level 10 sect to reject rank 3, got code=%d", code)
	}

	// Level 50 sect: maxLevel 6 (Rank 6 Tiên tài)
	unitCost6, effRank6, code, _ := stockItemUnitCost(nil, 50, "material", "mock_mat_r6")
	if code != http.StatusOK || effRank6 != 6 {
		t.Fatalf("expected level 50 sect to allow rank 6, got code=%d rank=%d", code, effRank6)
	}
	// BasePrice = 60, Rank 6 multiplier = 1000, 1.5x import multiplier = 60 * 1000 * 1.5 = 90,000
	if unitCost6 != 90000 {
		t.Fatalf("expected unitCost 90000 for rank 6 material, got %d", unitCost6)
	}
	// Level 50 cannot stock rank 7
	_, _, code, _ = stockItemUnitCost(nil, 50, "material", "mock_mat_r7")
	if code != http.StatusForbidden {
		t.Fatalf("expected level 50 sect to reject rank 7, got code=%d", code)
	}

	// Level 80 sect: maxLevel 9 (Rank 9 Tiên tài)
	unitCost9, effRank9, code, _ := stockItemUnitCost(nil, 80, "material", "mock_mat_r9")
	if code != http.StatusOK || effRank9 != 9 {
		t.Fatalf("expected level 80 sect to allow rank 9, got code=%d rank=%d", code, effRank9)
	}
	// BasePrice = 90, Rank 9 multiplier = 1000, 1.5x import multiplier = 90 * 1000 * 1.5 = 135,000
	if unitCost9 != 135000 {
		t.Fatalf("expected unitCost 135000 for rank 9 material, got %d", unitCost9)
	}
}

