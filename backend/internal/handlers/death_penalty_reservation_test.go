package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestSelectEligibleGusNeverDropsActiveProductComponents(t *testing.T) {
	oldEnabled, oldPct := deathPenaltyConfig.GusEnabled, deathPenaltyConfig.GusPct
	deathPenaltyConfig.GusEnabled = true
	deathPenaltyConfig.GusPct = 100
	t.Cleanup(func() {
		deathPenaltyConfig.GusEnabled = oldEnabled
		deathPenaltyConfig.GusPct = oldPct
	})

	gus := models.GuInventory{{InstanceID: "reserved-core", GuID: "test_gu", Rank: 6}}
	remaining, dropped := selectEligibleGus(gus, 1, 10, map[string]bool{"reserved-core": true})
	if len(remaining) != 1 || len(dropped) != 0 {
		t.Fatalf("reserved Gu selection = remaining %d, dropped %d; want 1 and 0", len(remaining), len(dropped))
	}
}
