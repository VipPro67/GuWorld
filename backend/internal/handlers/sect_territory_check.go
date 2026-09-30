package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

type dbRowQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

type dbQueryRunner interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type SectTerritoryInfo struct {
	InTerritory bool
	SectID      string
	SectName    string
	MainHallLvl int
	MapX        int
	MapY        int
	Radius      int
	IsShielded  bool
	ShieldUntil *time.Time
}

// IsInsideSquareTerritory returns true if (x, y) is within the square territory centered at (mapX, mapY) with given radius.
func IsInsideSquareTerritory(x, y, mapX, mapY, radius int) bool {
	dx := x - mapX
	dy := y - mapY
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx <= radius && dy <= radius
}

// CheckPlayerSectTerritory checks if a player is currently inside their own sect's square territory.
func CheckPlayerSectTerritory(q dbRowQueryer, playerID uuid.UUID, region string, x, y int) SectTerritoryInfo {
	var info SectTerritoryInfo
	if q == nil {
		q = db.DB
	}
	if playerID == uuid.Nil {
		return info
	}

	var sRegion string
	var shieldUntil sql.NullTime
	err := q.QueryRow(`
		SELECT s.id, s.name, s.map_x, s.map_y, s.region, COALESCE((s.buildings->>'main_hall')::int, 1), s.shield_until
		FROM sect_members sm
		JOIN sects s ON sm.sect_id = s.id
		WHERE sm.player_id = $1
	`, playerID).Scan(&info.SectID, &info.SectName, &info.MapX, &info.MapY, &sRegion, &info.MainHallLvl, &shieldUntil)
	if err != nil || sRegion != region {
		return info
	}

	if shieldUntil.Valid {
		info.ShieldUntil = &shieldUntil.Time
		if shieldUntil.Time.After(time.Now()) {
			info.IsShielded = true
		}
	}

	info.Radius = 5 + info.MainHallLvl
	info.InTerritory = IsInsideSquareTerritory(x, y, info.MapX, info.MapY, info.Radius)
	return info
}

// TerritoriesOverlap returns true if two square territories centered at (x1, y1) with radius r1
// and (x2, y2) with radius r2 overlap (share any tiles).
func TerritoriesOverlap(x1, y1, r1, x2, y2, r2 int) bool {
	dx := x1 - x2
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y2
	if dy < 0 {
		dy = -dy
	}
	return dx <= (r1+r2) && dy <= (r1+r2)
}

// CheckTerritoryOverlap checks whether a proposed square territory (mapX, mapY, radius) in region
// overlaps with any other sect in the same region.
// excludeSectID is omitted from the check (e.g. self-check during relocation or building upgrade).
// Returns (hasOverlap, conflictingSectName, error).
func CheckTerritoryOverlap(q dbQueryRunner, region string, mapX, mapY, radius int, excludeSectID string) (bool, string, error) {
	if q == nil {
		q = db.DB
	}
	rows, err := q.Query(`
		SELECT id, name, map_x, map_y, COALESCE((buildings->>'main_hall')::int, 1)
		FROM sects
		WHERE region = $1 AND id != $2
	`, region, excludeSectID)
	if err != nil {
		return false, "", err
	}
	defer rows.Close()

	for rows.Next() {
		var otherID, otherName string
		var otherX, otherY, otherMainHall int
		if err := rows.Scan(&otherID, &otherName, &otherX, &otherY, &otherMainHall); err != nil {
			continue
		}
		otherRadius := 5 + otherMainHall
		if TerritoriesOverlap(mapX, mapY, radius, otherX, otherY, otherRadius) {
			return true, otherName, nil
		}
	}
	return false, "", rows.Err()
}

// FindValidExileCoordinates finds a valid outdoor coordinate in region where a sect with given radius
// will not overlap with any other sect's territory and does not land in a safe zone or out of bounds.
func FindValidExileCoordinates(q dbQueryRunner, region string, radius int, excludeSectID string) (int, int, error) {
	if q == nil {
		q = db.DB
	}

	var width, height int
	var zoneType string
	err := q.QueryRow(`
		SELECT width, height, COALESCE(zone_type, 'KHU_VUC')
		FROM regions WHERE id = $1
	`, region).Scan(&width, &height, &zoneType)
	if err != nil || width <= 0 || height <= 0 {
		width = 100
		height = 100
	}

	minX := radius + 2
	maxX := width - radius - 2
	if maxX < minX {
		minX = 1
		maxX = width - 1
	}

	minY := radius + 2
	maxY := height - radius - 2
	if maxY < minY {
		minY = 1
		maxY = height - 1
	}

	// Try up to 100 random coordinates
	for attempts := 0; attempts < 100; attempts++ {
		candX := minX + rand.Intn(maxX-minX+1)
		candY := minY + rand.Intn(maxY-minY+1)

		hasOverlap, _, errOverlap := CheckTerritoryOverlap(q, region, candX, candY, radius, excludeSectID)
		if errOverlap == nil && !hasOverlap {
			return candX, candY, nil
		}
	}

	// Systematic fallback scan if random attempts hit crowded areas
	step := radius + 1
	if step < 2 {
		step = 2
	}
	for x := minX; x <= maxX; x += step {
		for y := minY; y <= maxY; y += step {
			hasOverlap, _, errOverlap := CheckTerritoryOverlap(q, region, x, y, radius, excludeSectID)
			if errOverlap == nil && !hasOverlap {
				return x, y, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("Không tìm thấy vị trí thích hợp trong đại vực để hạ trại sơn môn")
}

