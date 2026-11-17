package handlers

import "testing"

func TestAperturePortalMarkerChangeOmitsCoordinatesAndPortalID(t *testing.T) {
	event := newAperturePortalMarkerChangeEvent("portal_spawn", "Southern_Border")
	if event.EntityType != "map_cell" || event.Action != "portal_spawn" || event.Scope.RegionID != "Southern_Border" {
		t.Fatalf("unexpected refresh event scope/action: %+v", event)
	}
	if event.X != 0 || event.Y != 0 || len(event.Data) != 0 {
		t.Fatalf("portal refresh event leaked marker coordinates or ID: %+v", event)
	}
}
