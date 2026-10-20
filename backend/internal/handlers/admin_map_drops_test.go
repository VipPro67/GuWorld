package handlers

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRestoreDroppedGus(t *testing.T) {
	got := restoreDroppedGus(models.GuInventory{
		{InstanceID: "", GuID: "moonlight_gu", Rank: 3, Hunger: 0, Status: "", Equipped: true, IsVital: true},
		{InstanceID: "gu-keep", GuID: "water_gu", Rank: 2, Hunger: 40, Status: "SLEEPING", Equipped: true, IsVital: true},
	})
	if len(got) != 2 {
		t.Fatalf("restored %d Gu, want 2", len(got))
	}
	first := got[0]
	if !strings.HasPrefix(first.InstanceID, "gu-") || first.Hunger != 100 || first.Status != "ACTIVE" || first.Equipped || first.IsVital || first.EquippedAt != nil {
		t.Fatalf("first Gu not normalised: %+v", first)
	}
	second := got[1]
	if second.InstanceID != "gu-keep" || second.Hunger != 40 || second.Status != "SLEEPING" || second.Equipped || second.IsVital {
		t.Fatalf("second Gu lost stored state: %+v", second)
	}
}
