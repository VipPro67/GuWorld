package pricing

import "testing"

func TestCurrencyPrecision(t *testing.T) {
	if got := SellPrice(1); got != 0.1 {
		t.Fatalf("SellPrice(1) = %v, want 0.1", got)
	}
	if got := SellPrice(0.001); got != 0.001 {
		t.Fatalf("minimum sell price = %v, want 0.001", got)
	}
	if got := Quantize(1.2346); got != 1.235 {
		t.Fatalf("Quantize = %v, want 1.235", got)
	}
	if !IsValidAmount(0.001) || IsValidAmount(0.0001) {
		t.Fatal("0.001 precision validation failed")
	}
}
