package cache

import "testing"

func TestInstanceCellKeyIsolatesMapInstances(t *testing.T) {
	first := instanceCellKey("aperture_interior", "aperture-one", 4, 7)
	second := instanceCellKey("aperture_interior", "aperture-two", 4, 7)
	if first == second {
		t.Fatal("same coordinate in separate aperture instances must use different keys")
	}
	if first == cellKey("aperture_interior", 4, 7) {
		t.Fatal("instance-scoped cell key must not collide with region-scoped state")
	}
}
