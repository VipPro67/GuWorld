package handlers

import (
	"errors"
	"reflect"
	"testing"
)

func TestValidateRealtimeHotbarOrderMovesChosenGusFirstAndPreservesTheRest(t *testing.T) {
	got, err := validateRealtimeHotbarOrder([]string{"gu-c", "gu-a"}, []string{"gu-a", "gu-b", "gu-c"}, []string{"gu-a", "gu-b", "gu-c"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gu-c", "gu-a", "gu-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order=%v, want %v", got, want)
	}
}

func TestValidateRealtimeHotbarOrderRejectsDuplicateAndUnequippedGus(t *testing.T) {
	_, err := validateRealtimeHotbarOrder([]string{"gu-a", "gu-a"}, []string{"gu-a", "gu-b"}, nil)
	if !errors.Is(err, errHotbarDuplicateGu) {
		t.Fatalf("duplicate error=%v", err)
	}
	_, err = validateRealtimeHotbarOrder([]string{"gu-c"}, []string{"gu-a", "gu-b"}, nil)
	if !errors.Is(err, errHotbarContainsUnequippedGu) {
		t.Fatalf("unequipped error=%v", err)
	}
}

func TestValidateRealtimeHotbarOrderCapsAtSix(t *testing.T) {
	ids := []string{"1", "2", "3", "4", "5", "6", "7"}
	_, err := validateRealtimeHotbarOrder(ids, ids, nil)
	if !errors.Is(err, errHotbarTooManySlots) {
		t.Fatalf("slot limit error=%v", err)
	}
}

// TestValidateRealtimeHotbarOrderKeepsSavedAssignedSlots pins the hotbar
// guarantee for the save endpoint: explicitly requested members must land on
// exactly the slot the player chose (index+1), and omitted valid members are
// appended behind them in their previously saved order.
func TestValidateRealtimeHotbarOrderKeepsSavedAssignedSlots(t *testing.T) {
	current := []string{"h3", "h1", "h2"}
	equipped := []string{"h1", "h2", "h3", "h4"}

	got, err := validateRealtimeHotbarOrder([]string{"h1", "h2"}, equipped, current)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"h1", "h2", "h3", "h4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("saved order=%v, want %v (h3 must not be re-sorted)", got, want)
	}

	// Stale currentOrder entry (no longer valid) must disappear; unseen valid
	// member h4 is appended once, deterministically.
	currentStale := []string{"h3", "h1", "stale"}
	got, err = validateRealtimeHotbarOrder([]string{"h1", "h2"}, equipped, currentStale)
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"h1", "h2", "h3", "h4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stale id must be pruned, got %v", got)
	}
}

// TestValidateRealtimeHotbarOrderAllowsEmptySlots pins the hole semantics:
// deliberate "" slots do not consume equipped capacity, so a layout such as
// [a, "", b] with only two equipped Gu is valid.
func TestValidateRealtimeHotbarOrderAllowsEmptySlots(t *testing.T) {
	got, err := validateRealtimeHotbarOrder(
		[]string{"gu-a", "", "gu-b"}, []string{"gu-a", "gu-b"}, []string{"gu-a", "gu-b"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gu-a", "gu-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order=%v, want %v", got, want)
	}

	// All-empty (cleared) layout is also valid.
	got, err = validateRealtimeHotbarOrder([]string{""}, []string{"gu-a"}, []string{"gu-a"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"gu-a"}) {
		t.Fatalf("cleared layout must keep equipped set, got %v", got)
	}
}
