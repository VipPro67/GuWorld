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
