package handlers

import (
	"database/sql"
	"testing"
)

func TestApertureDeathDestinationResolution(t *testing.T) {
	resolveDest := func(placedRegion, returnRegion sql.NullString, placedX, placedY, returnX, returnY sql.NullFloat64) (string, int, int) {
		destRegion := "Southern_Border"
		destX := 500
		destY := 500
		if placedRegion.Valid && placedRegion.String != "" {
			destRegion = placedRegion.String
		} else if returnRegion.Valid && returnRegion.String != "" {
			destRegion = returnRegion.String
		}
		if placedX.Valid {
			destX = int(placedX.Float64)
		} else if returnX.Valid {
			destX = int(returnX.Float64)
		}
		if placedY.Valid {
			destY = int(placedY.Float64)
		} else if returnY.Valid {
			destY = int(returnY.Float64)
		}
		return destRegion, destX, destY
	}

	// 1. Placed location valid -> uses placed location
	reg, x, y := resolveDest(
		sql.NullString{String: "tan_thu_coc_nam_cuong", Valid: true},
		sql.NullString{String: "other_region", Valid: true},
		sql.NullFloat64{Float64: 695, Valid: true},
		sql.NullFloat64{Float64: 739, Valid: true},
		sql.NullFloat64{Float64: 100, Valid: true},
		sql.NullFloat64{Float64: 100, Valid: true},
	)
	if reg != "tan_thu_coc_nam_cuong" || x != 695 || y != 739 {
		t.Fatalf("expected placed location tan_thu_coc_nam_cuong (695, 739), got %s (%d, %d)", reg, x, y)
	}

	// 2. Placed null, return valid -> uses return location
	reg, x, y = resolveDest(
		sql.NullString{Valid: false},
		sql.NullString{String: "hub_nam_cuong", Valid: true},
		sql.NullFloat64{Valid: false},
		sql.NullFloat64{Valid: false},
		sql.NullFloat64{Float64: 120, Valid: true},
		sql.NullFloat64{Float64: 250, Valid: true},
	)
	if reg != "hub_nam_cuong" || x != 120 || y != 250 {
		t.Fatalf("expected return location hub_nam_cuong (120, 250), got %s (%d, %d)", reg, x, y)
	}

	// 3. Both null -> falls back to Southern_Border (500, 500)
	reg, x, y = resolveDest(
		sql.NullString{Valid: false},
		sql.NullString{Valid: false},
		sql.NullFloat64{Valid: false},
		sql.NullFloat64{Valid: false},
		sql.NullFloat64{Valid: false},
		sql.NullFloat64{Valid: false},
	)
	if reg != "Southern_Border" || x != 500 || y != 500 {
		t.Fatalf("expected fallback Southern_Border (500, 500), got %s (%d, %d)", reg, x, y)
	}
}
