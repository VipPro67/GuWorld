package sectmechanics

import (
	"math"
	"testing"
)

func TestTreasuryProtection(t *testing.T) {
	if got := PlunderChance(1); math.Abs(got-0.18) > 1e-9 {
		t.Fatalf("level 1 chance = %v", got)
	}
	if got := PlunderChance(99); got != 0.05 {
		t.Fatalf("capped chance = %v", got)
	}
	if got := RollPlunderQuantity(1, 0.18, 0.10); got != 1 {
		t.Fatalf("singleton should drop on successful roll: %d", got)
	}
	if got := RollPlunderQuantity(1, 0.18, 0.20); got != 0 {
		t.Fatalf("singleton should be protected on failed roll: %d", got)
	}
}

func TestDiscountVaultFee(t *testing.T) {
	if got, saved := DiscountVaultFee(1000, 1); got != 500 || saved != 500 {
		t.Fatalf("level 1 = %d saved %d", got, saved)
	}
	if got, saved := DiscountVaultFee(1000, 9); got != 100 || saved != 900 {
		t.Fatalf("level 9 = %d saved %d", got, saved)
	}
}
