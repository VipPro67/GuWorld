package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestLegacyPortalDestinationProjectsMapCellToWorld(t *testing.T) {
	target := &models.Region{
		ID: "nc_phong_bao_dinh", MinX: 8000, MinY: 0, MaxX: 10000, MaxY: 2000,
		Width: 2000, Height: 2000,
	}
	x, y := LegacyPortalDestination(target, 1998, 1000)
	if x != 9998 || y != 1000 {
		t.Fatalf("portal local cell projected to %d,%d; want 9998,1000", x, y)
	}

	// Out-of-range authored cells are clamped inside the map before projection.
	x, y = LegacyPortalDestination(target, 5000, -20)
	if x != 9998 || y != 1 {
		t.Fatalf("clamped portal projected to %d,%d; want 9998,1", x, y)
	}
}
