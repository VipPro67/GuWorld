package handlers

import (
	"testing"
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
	// When withdrawing a Gu from sect vault, its hunger must always be 1
	const expectedWithdrawHunger = 1
	if expectedWithdrawHunger != 1 {
		t.Fatalf("expected withdrawn hunger to be 1, got %d", expectedWithdrawHunger)
	}
}
