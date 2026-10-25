package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// WorldObjectView is a lightweight view of a persistent shared cell for the
// large world map (villages / tram nghi, bosses, high-tier resources).
type WorldObjectView struct {
	RegionID       string `json:"region_id"`
	X              int    `json:"x"`
	Y              int    `json:"y"`
	CellType       string `json:"cell_type"`
	ContentID      string `json:"content_id"`
	Rank           int    `json:"rank,omitempty"`
	RespawnSeconds int    `json:"respawn_seconds"`
	Cleared        bool   `json:"cleared"`
}

// ExecuteGetExploreObjects returns the persistent map cells without HTTP coupling.
func ExecuteGetExploreObjects() (int, any) {
	ensureZonesLoaded()

	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()

	out := make([]WorldObjectView, 0, len(zoneReg.persistent))
	for key, pc := range zoneReg.persistent {
		out = append(out, WorldObjectView{
			RegionID:       pc.RegionID,
			X:              pc.X,
			Y:              pc.Y,
			CellType:       pc.Type,
			ContentID:      pc.Content,
			Rank:           pc.Rank,
			RespawnSeconds: pc.RespawnSeconds,
			Cleared:        pc.cleared,
		})
		_ = key
	}
	return http.StatusOK, out
}

// GetExploreObjects returns the persistent map cells (map_cells) so the large
// world map can render villages, bosses and high-tier resource points.
func GetExploreObjects(c *gin.Context) {
	status, out := ExecuteGetExploreObjects()
	c.JSON(status, out)
}

// NearbyCell is a resolved non-empty cell (monster / boss / resource / wild Gu)
// near the player, for the large world map markers.
type NearbyCell struct {
	X       int       `json:"x"`
	Y       int       `json:"y"`
	Type    string    `json:"cell_type"`
	Content string    `json:"content"`
	Rank    int       `json:"rank,omitempty"`
	SubRank int       `json:"sub_rank,omitempty"`
	Region  string    `json:"region_id,omitempty"`
	Vein    *VeinInfo `json:"vein,omitempty"`
}

// ExecuteGetCellsRect resolves non-empty cells inside an explicit rectangle.
func ExecuteGetCellsRect(playerID uuid.UUID, minX, minY, maxX, maxY int) (int, any) {
	EnsureWorldPosLoaded(playerID.String())
	entry, hasScope := worldPosIndex.Get(playerID.String())

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= WorldWidth {
		maxX = WorldWidth - 1
	}
	if maxY >= WorldHeight {
		maxY = WorldHeight - 1
	}
	if maxX < minX || maxY < minY {
		return http.StatusBadRequest, gin.H{"error": "Rectangle tọa độ không hợp lệ."}
	}

	// Bound the resolved area so a single request stays cheap.
	const maxSide = 256
	if maxX-minX+1 > maxSide {
		maxX = minX + maxSide - 1
	}
	if maxY-minY+1 > maxSide {
		maxY = minY + maxSide - 1
	}

	instanceRegion := hasScope && entry.InstanceID != ""
	var zones []models.Region
	if !instanceRegion {
		zones = rectZoneCandidates(minX, minY, maxX, maxY, entry.MapID)
	}
	candidates := []NearbyCell{}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			regionID := entry.Region
			if !instanceRegion {
				regionID = regionAtFromRectZones(x, y, zones)
			}
			cell := ResolveCellInRegion(regionID, x, y)
			if cell.Type == "Empty" {
				continue
			}
			candidates = append(candidates, NearbyCell{X: cell.X, Y: cell.Y, Type: cell.Type, Content: cell.Content, Rank: cell.Rank, SubRank: cell.SubRank, Region: regionID})
		}
	}
	out, err := filterClearedCells(candidates)
	if err != nil {
		return http.StatusServiceUnavailable, gin.H{"error": "Trạng thái bản đồ tạm thời không khả dụng."}
	}
	if !instanceRegion {
		out = append(out, veinCellsForRegionOrBounds(entry.Region, minX, minY, maxX, maxY)...)
	}
	return http.StatusOK, out
}

// GetCellsRect resolves the non-empty cells inside an explicit rectangle
// (min_x/min_y/max_x/max_y). Used by the world map to prefetch the exact
// visible area or a band ahead of the player. Bounded to prevent abuse.
func GetCellsRect(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var minX, minY, maxX, maxY int
	fmt.Sscanf(c.Query("min_x"), "%d", &minX)
	fmt.Sscanf(c.Query("min_y"), "%d", &minY)
	fmt.Sscanf(c.Query("max_x"), "%d", &maxX)
	fmt.Sscanf(c.Query("max_y"), "%d", &maxY)

	status, res := ExecuteGetCellsRect(playerID, minX, minY, maxX, maxY)
	c.JSON(status, res)
}

func filterClearedCells(cells []NearbyCell) ([]NearbyCell, error) {
	coords := make([]cache.CellCoordKey, len(cells))
	for i, cell := range cells {
		coords[i] = cache.CellCoordKey{RegionID: cell.Region, X: cell.X, Y: cell.Y}
	}
	cleared, err := cache.GetClearedCells(coords)
	if err != nil {
		logger.S.Warnf("failed to batch load cleared world cells: %v", err)
		return nil, err
	}
	out := make([]NearbyCell, 0, len(cells)-len(cleared))
	for i, cell := range cells {
		if _, ok := cleared[coords[i]]; !ok {
			out = append(out, cell)
		}
	}
	return out, nil
}

// GetNearbyPlayers returns a list of other active players within a given radius
func GetNearbyPlayers(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var x, y, r int
	fmt.Sscanf(c.Query("x"), "%d", &x)
	fmt.Sscanf(c.Query("y"), "%d", &y)
	fmt.Sscanf(c.Query("r"), "%d", &r)

	if r <= 0 {
		r = 300
	}
	if r > 300 {
		r = 300
	}

	// AOI is always centered on the authoritative server position. Query
	// coordinates are accepted for backward compatibility but never trusted to
	// inspect an arbitrary part of the current region.
	region := ""
	mapID := "world_map"
	instanceID := mapInstanceIDForPlayer(authID.String())
	if entry, ok := worldPosIndex.Get(authID.String()); ok {
		x, y, region = entry.X, entry.Y, entry.Region
		mapID, instanceID = entry.MapID, entry.InstanceID
	}
	if region == "" {
		var xLoc, yLoc int
		var regLoc string
		var mapInst sql.NullString
		var uname, avatar string
		if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT x, y, COALESCE(current_region, ''), map_instance_id::text, username, COALESCE(avatar_url, '') FROM players WHERE id=$1`, authID).Scan(&xLoc, &yLoc, &regLoc, &mapInst, &uname, &avatar); err == nil {
			x, y, region = xLoc, yLoc, regLoc
			mapID = worldMapIDForRegion(regLoc)
			if mapInst.Valid && mapInst.String != "" {
				instanceID = mapInst.String
			}
			worldPosIndex.Set(WorldPosEntry{
				ID:         authID.String(),
				Username:   uname,
				AvatarURL:  avatar,
				X:          xLoc,
				Y:          yLoc,
				Region:     regLoc,
				MapID:      mapID,
				InstanceID: instanceID,
				TS:         time.Now().UnixMilli(),
			})
		}
	}
	TouchPlayerActivity(authID)

	type NearbyPlayer struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		X         int    `json:"x"`
		Y         int    `json:"y"`
	}

	// Fast path: serve from the in-memory position index. Fall back to SQL only
	// while the index is still cold (e.g. very early startup before the first
	// sync worker reconcile).
	entries := worldPosIndex.NearbyInScope(x, y, r, authID.String(), mapID, instanceID)
	if worldPosIndex.Len() == 0 {
		entries = queryNearbyPlayersSQL(c, authID, x, y, r, mapID, instanceID)
	}

	nearbyPlayers := make([]NearbyPlayer, 0, len(entries))
	for _, e := range entries {
		nearbyPlayers = append(nearbyPlayers, NearbyPlayer{
			ID:        e.ID,
			Username:  e.Username,
			AvatarURL: e.AvatarURL,
			X:         e.X,
			Y:         e.Y,
		})
	}

	c.JSON(http.StatusOK, nearbyPlayers)
}

// queryNearbyPlayersSQL is the legacy SQL path for nearby players, kept as a
// cold-start fallback for the in-memory position index.
func queryNearbyPlayersSQL(c *gin.Context, authID uuid.UUID, x, y, r int, mapID, instanceID string) []WorldPosEntry {
	entries := []WorldPosEntry{}
	rows, errQuery := db.DB.QueryContext(c.Request.Context(), `
		SELECT p.id, p.username, COALESCE(p.avatar_url, ''), p.x, p.y, COALESCE(p.current_region, ''),
		       COALESCE(rwm.world_map_id,'world_map'), COALESCE(p.map_instance_id::text,'')
		FROM players p LEFT JOIN region_world_maps rwm ON rwm.region_id=p.current_region
		WHERE p.id != $1
		  AND (p.last_activity >= NOW() - INTERVAL '5 minutes' OR p.last_mana_update >= NOW() - INTERVAL '5 minutes')
		  AND p.x >= $2 AND p.x <= $3
		  AND p.y >= $4 AND p.y <= $5
		  AND COALESCE(rwm.world_map_id,'world_map') = $6
		  AND COALESCE(p.map_instance_id::text,'') = $7
	`, authID, x-r, x+r, y-r, y+r, mapID, instanceID)
	if errQuery != nil {
		logger.S.Error("Failed to fetch nearby players:", errQuery)
		return entries
	}
	defer rows.Close()
	for rows.Next() {
		var e WorldPosEntry
		if errScan := rows.Scan(&e.ID, &e.Username, &e.AvatarURL, &e.X, &e.Y, &e.Region, &e.MapID, &e.InstanceID); errScan == nil {
			e.TS = time.Now().UnixMilli()
			entries = append(entries, e)
		}
	}
	return entries
}
