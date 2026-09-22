package handlers

import (
==	"encoding/json"
	"reflect"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestDecodeActiveCombatTreatsDatabaseNullAsNoCombat(t *testing.T) {
	for _, raw := range [][]byte{nil, {}, []byte("null"), []byte("{}")} {
		combat, err := decodeActiveCombat(raw)
		if err != nil {
			t.Fatalf("decodeActiveCombat(%q) returned error: %v", raw, err)
		}
		if combat != nil {
			t.Fatalf("decodeActiveCombat(%q) = %#v, want nil", raw, combat)
		}
	}
}

func TestPlayerRealtimePayloadsCarryStateRevision(t *testing.T) {
	snapshot := playerRealtimeSnapshot(models.Player{Account: models.Account{StateRevision: 42}})
	if snapshot.StateRevision != 42 {
		t.Fatalf("snapshot state revision = %d, want 42", snapshot.StateRevision)
	}

	raw, err := json.Marshal(PlayerDeltaEvent{Type: PositionEventUpdated, StateRevision: 43})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["state_revision"] != float64(43) {
		t.Fatalf("delta state revision = %#v, want 43", payload["state_revision"])
func TestNormalizePlayerEventDomainsDefaultsDeduplicatesAndSorts(t *testing.T) {
	got := normalizePlayerEventDomains([]PlayerEventDomain{
		RecipesDomain, PlayerDomain, RecipesDomain, "",
	}, PlayerDomain)
	want := []PlayerEventDomain{PlayerDomain, RecipesDomain}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePlayerEventDomains() = %v, want %v", got, want)
	}

	got = normalizePlayerEventDomains(nil, GuInventoryDomain)
	if !reflect.DeepEqual(got, []PlayerEventDomain{GuInventoryDomain}) {
		t.Fatalf("fallback domains = %v", got)
	}
}

func TestPlayerEventRevisionIsMonotonic(t *testing.T) {
	first := nextPlayerEventRevision()
	second := nextPlayerEventRevision()
	if second <= first {
		t.Fatalf("revision did not increase: first=%d second=%d", first, second)
	}
}

func TestDecodeActiveCombatRejectsInvalidJSON(t *testing.T) {
	if _, err := decodeActiveCombat([]byte("{")); err == nil {
		t.Fatal("decodeActiveCombat accepted invalid JSON")
	}
}
