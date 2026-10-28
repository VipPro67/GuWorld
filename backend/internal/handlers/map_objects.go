package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// MapMarkerView is the unified contract for a static structure marker on the
// world map (sects today, NPCs/mines/merchants later). Dynamic entities keep
// their existing transports: veins and ground drops arrive through
// /explore/cells/rect, monsters through the realtime world snapshot.
//
// Coordinates are local map cells of RegionID, matching players.x/y and the
// cells feed. See docs/GAME_DATA.md → World coordinates.
type MapMarkerView struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	RegionID string `json:"region_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Name     string `json:"name"`
	Detail   any    `json:"detail,omitempty"`
}

type SectMarker struct {
	Level       int            `json:"level"`
	ShieldUntil *time.Time     `json:"shield_until"`
	Buildings   map[string]int `json:"buildings"`
	IsMine      bool           `json:"is_mine"`
}

type TeleportMarker struct {
	OwnerID      string `json:"owner_id"`
	Tier         string `json:"tier"`
	AccessPolicy string `json:"access_policy"`
	IsActive     bool   `json:"is_active"`
}

// sectMarkersInRegion loads sect bases for one map. Defenders are deliberately
// not loaded: markers only need position, level and shield state, and the
// defender list costs one query per sect in GetSectsMap.
func sectMarkersInRegion(regionID string, mySectID *string) []MapMarkerView {
	rows, err := db.DB.Query(`
		SELECT s.id, s.name, s.level, s.map_x, s.map_y, s.shield_until, s.buildings
		FROM sects s
		WHERE s.region = $1`, regionID)
	if err != nil {
		logger.S.Warnf("Failed to load sect markers for region %s: %v", regionID, err)
		return nil
	}
	defer rows.Close()

	out := make([]MapMarkerView, 0)
	for rows.Next() {
		var (
			id, name string
			level    int
			x, y     int
			shield   sql.NullTime
			raw      []byte
		)
		if err := rows.Scan(&id, &name, &level, &x, &y, &shield, &raw); err != nil {
			continue
		}
		detail := SectMarker{Level: level, Buildings: map[string]int{}}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &detail.Buildings)
		}
		if shield.Valid {
			t := shield.Time
			detail.ShieldUntil = &t
		}
		if mySectID != nil && *mySectID == id {
			detail.IsMine = true
		}
		out = append(out, MapMarkerView{
			ID:       id,
			Kind:     "sect",
			RegionID: regionID,
			X:        x,
			Y:        y,
			Name:     name,
			Detail:   detail,
		})
	}
	return out
}

// teleportMarkersInRegion returns the arrays this player may see on the map.
// Visibility mirrors GetMyTeleportArrays (owner OR public OR own sect); private
// arrays of other players must never leak into a marker feed.
func teleportMarkersInRegion(regionID string, authID uuid.UUID, mySectID *string) []MapMarkerView {
	query := `
		SELECT t.id, t.name, t.owner_id, t.x, t.y, t.tier, t.access_policy, t.is_active
		FROM teleport_arrays t
		WHERE t.is_active = TRUE AND t.region_id = $2
		  AND (t.owner_id = $1 OR t.access_policy = 'PUBLIC'`
	args := []interface{}{authID, regionID}
	if mySectID != nil && *mySectID != "" {
		query += " OR (t.sect_id = $3 AND t.access_policy = 'SECT')"
		args = append(args, *mySectID)
	}
	query += ")"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		logger.S.Warnf("Failed to load teleport markers for region %s: %v", regionID, err)
		return nil
	}
	defer rows.Close()

	out := make([]MapMarkerView, 0)
	for rows.Next() {
		var (
			id, name, ownerID string
			x, y              int
			detail            TeleportMarker
		)
		if err := rows.Scan(&id, &name, &ownerID, &x, &y, &detail.Tier, &detail.AccessPolicy, &detail.IsActive); err != nil {
			continue
		}
		detail.OwnerID = ownerID
		out = append(out, MapMarkerView{
			ID:       id,
			Kind:     "teleport",
			RegionID: regionID,
			X:        x,
			Y:        y,
			Name:     name,
			Detail:   detail,
		})
	}
	return out
}

// GetZoneMapObjects returns static structure markers for one map. The client
// merges this with the cells feed and the realtime snapshot into a single
// marker view-model, so the map and the radar cannot drift apart.
func GetZoneMapObjects(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	regionID := strings.TrimSpace(c.Query("region"))
	if regionID == "" {
		player, err := playerstate.Load(db.DB, authID, playerstate.Options{})
		if err != nil {
			if playerstate.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác định khu vực hiện tại."})
			return
		}
		regionID = player.CurrentRegion
	}
	if regionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu khu vực."})
		return
	}

	mySectID := getPlayerSectID(nil, authID)

	out := make([]MapMarkerView, 0)
	out = append(out, sectMarkersInRegion(regionID, mySectID)...)
	out = append(out, teleportMarkersInRegion(regionID, authID, mySectID)...)
	c.JSON(http.StatusOK, out)
}
