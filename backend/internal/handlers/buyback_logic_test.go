package handlers

import (
	"reflect"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestHasDuplicateGuInstanceIDs(t *testing.T) {
	if hasDuplicateGuInstanceIDs([]string{"gu-1", "gu-2"}) {
		t.Fatal("unique Gu IDs were reported as duplicates")
	}
	if !hasDuplicateGuInstanceIDs([]string{"gu-1", "gu-1"}) {
		t.Fatal("duplicate Gu IDs were accepted")
	}
}

func TestValidateTradeOfferDataRejectsInvalidAssets(t *testing.T) {
	tests := []models.TradeOfferData{
		{PrimevalStones: -1},
		{Gus: []string{"gu-1", "gu-1"}},
		{Materials: map[string]int64{"mat": -1}},
		{Recipes: map[string]float64{"recipe": 101}},
	}
	for _, data := range tests {
		if err := validateTradeOfferData(data); err == nil {
			t.Fatalf("invalid trade offer was accepted: %#v", data)
		}
	}
}

func TestLockedGuIDsForPartialFills(t *testing.T) {
	ids := []string{"gu-1", "gu-2", "gu-3"}
	got, ok := lockedGuIDsForFill(ids, 1, 2)
	if !ok || !reflect.DeepEqual(got, []string{"gu-2", "gu-3"}) {
		t.Fatalf("later fill got %v, %v; want [gu-2 gu-3], true", got, ok)
	}
}

func TestRemainingLockedGuIDsExcludesFilledInstances(t *testing.T) {
	got, ok := remainingLockedGuIDs([]string{"sold", "locked-1", "locked-2"}, 1)
	if !ok || !reflect.DeepEqual(got, []string{"locked-1", "locked-2"}) {
		t.Fatalf("remaining IDs got %v, %v", got, ok)
	}
}
