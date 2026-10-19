package handlers

import (
	"encoding/json"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// TestRegionRulesCacheInvalidatesWhenRulesChange verifies the decode cache is
// revalidated against the region's raw rules, so editing a region's rules takes
// effect without a backend restart.
func TestRegionRulesCacheInvalidatesWhenRulesChange(t *testing.T) {
	previousZones, previousByID := zoneReg.zones, zoneReg.byID
	t.Cleanup(func() { zoneReg.zones, zoneReg.byID = previousZones, previousByID })

	zoneReg.zones = []models.Region{{
		ID:       "cache_zone_test",
		ZoneType: ZoneTypeNamed,
		Rules:    json.RawMessage(`{"actions":{"deny":["combat"]}}`),
	}}
	zoneReg.byID = map[string]*models.Region{"cache_zone_test": &zoneReg.zones[0]}

	if err := checkRegionAction("cache_zone_test", "combat"); err == nil {
		t.Fatal("expected combat to be denied on first read")
	}
	// Rewrite the rules on the same region id: the cache must observe it.
	zoneReg.zones[0].Rules = json.RawMessage(`{"actions":{"deny":[]}}`)
	if err := checkRegionAction("cache_zone_test", "combat"); err != nil {
		t.Fatalf("expected combat allowed after rules change, got %v", err)
	}
}

// TestRegionRulesCacheKeepsParentInheritance verifies caching the per-region
// decode does not change the parent-lineage merge semantics.
func TestRegionRulesCacheKeepsParentInheritance(t *testing.T) {
	previousZones, previousByID := zoneReg.zones, zoneReg.byID
	t.Cleanup(func() { zoneReg.zones, zoneReg.byID = previousZones, previousByID })

	zoneReg.zones = []models.Region{
		{ID: "cache_parent_test", ZoneType: ZoneTypeLarge, Rules: json.RawMessage(`{"rank":{"min":3},"gu":{"max_rank":4}}`)},
		{ID: "cache_child_test", ZoneType: ZoneTypeNamed, ParentID: "cache_parent_test", Rules: json.RawMessage(`{"gu":{"max_rank":2}}`)},
	}
	zoneReg.byID = map[string]*models.Region{
		"cache_parent_test": &zoneReg.zones[0],
		"cache_child_test":  &zoneReg.zones[1],
	}

	rules, err := regionRulesFor("cache_child_test")
	if err != nil {
		t.Fatal(err)
	}
	if rules.Rank.Min != 3 {
		t.Fatalf("expected inherited rank.min 3, got %d", rules.Rank.Min)
	}
	if rules.Gu.MaxRank != 2 {
		t.Fatalf("expected child gu.max_rank to override parent, got %d", rules.Gu.MaxRank)
	}
}
