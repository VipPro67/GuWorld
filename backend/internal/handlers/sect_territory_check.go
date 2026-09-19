package handlers

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

type dbRowQueryer interface {
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
