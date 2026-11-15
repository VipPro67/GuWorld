package handlers

import (
	"reflect"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
)

func TestHasDuplicateGuInstanceIDs(t *testing.T) {
	if hasDuplicateGuInstanceIDs([]string{"gu-1", "gu-2"}) {
		t.Fatal("unique Gu IDs were reported as duplicates")
	}
	if !hasDuplicateGuInstanceIDs([]string{"gu-1", "gu-1"}) {
		t.Fatal("duplicate Gu IDs were accepted")
	}
}

func TestMatchRequestedTradeGus(t *testing.T) {
	inventory := models.GuInventory{
		{InstanceID: "gu-1", GuID: "wolf", Status: "ACTIVE"},
		{InstanceID: "gu-2", GuID: "wolf", Status: "ACTIVE"},
		{InstanceID: "gu-dead", GuID: "wolf", Status: "DEAD"},
		{InstanceID: "gu-equipped", GuID: "hawk", Status: "ACTIVE", Equipped: true},
		{InstanceID: "gu-vital", GuID: "dragon", Status: "ACTIVE", IsVital: true},
	}
	tests := []struct {
		name        string
		requested   []string
		want        []string
		wantMissing string
		wantOK      bool
	}{
		{name: "match in inventory order", requested: []string{"wolf"}, want: []string{"gu-1"}, wantOK: true},
		{name: "consume distinct instances", requested: []string{"wolf", "wolf"}, want: []string{"gu-1", "gu-2"}, wantOK: true},
		{name: "match instance id", requested: []string{"gu-2"}, want: []string{"gu-2"}, wantOK: true},
		{name: "skip unavailable instances", requested: []string{"hawk"}, wantMissing: "hawk"},
		{name: "reject over-request", requested: []string{"wolf", "wolf", "wolf"}, wantMissing: "wolf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, missing, ok := matchRequestedTradeGus(tt.requested, inventory)
			if ok != tt.wantOK || missing != tt.wantMissing || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("matchRequestedTradeGus(%v) = (%v, %q, %v), want (%v, %q, %v)",
					tt.requested, got, missing, ok, tt.want, tt.wantMissing, tt.wantOK)
			}
		})
	}
}

func TestMaterialBatchEntries(t *testing.T) {
	got := materialBatchEntries(map[string]int64{"zinc": 3, "amber": 2, "empty": 0, "negative": -1})
	want := []services.ItemBatchEntry{
		{ItemID: "amber", Category: "materials", Quantity: 2},
		{ItemID: "zinc", Category: "materials", Quantity: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("materialBatchEntries() = %#v, want %#v", got, want)
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
