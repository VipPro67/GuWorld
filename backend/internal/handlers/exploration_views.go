package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
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

// GetExploreObjects returns the persistent map cells (map_cells) so the large
// world map can render villages, bosses and high-tier resource points.
func GetExploreObjects(c *gin.Context) {
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
	c.JSON(http.StatusOK, out)
}

// NearbyCell is a resolved non-empty cell (monster / boss / resource / wild Gu)
// near the player, for the large world map markers.
type NearbyCell struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Type    string `json:"cell_type"`
	Content string `json:"content"`
	Rank    int    `json:"rank,omitempty"`
	SubRank int    `json:"sub_rank,omitempty"`
	Region  string `json:"region_id,omitempty"`
}

// GetNearbyCells resolves the cells around the player (or an explicit
// ?x=&y=&r= box) and returns the non-empty ones so the big map can draw
// monster / boss / resource markers. Content is deterministic per zone cell,
// matching the grid gameplay (ResolveCell).
func GetNearbyCells(c *gin.Context) {
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
		r = 128
	}
	if r > 128 {
		r = 128
	}

	if x == 0 && y == 0 {
		player, err := loadPlayerForStatus(db.DB, authID, false)
		if err == nil {
			x, y = player.X, player.Y
		}
	}

	// Resolve the full box of cells around the center and keep the non-empty
	// ones. Box is at most (2r+1)^2 cells; with r capped at 128 this stays a
	// bounded, symmetric area around the player so the canvas markers align.
	candidates := []NearbyCell{}
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			cx := x + dx
			cy := y + dy
			if cx < 0 || cx >= WorldWidth || cy < 0 || cy >= WorldHeight {
				continue
			}
			cell := ResolveCell(cx, cy)
			if cell.Type == "Empty" {
				continue
			}
			regionID := DeriveRegionAt(cx, cy)
			candidates = append(candidates, NearbyCell{X: cell.X, Y: cell.Y, Type: cell.Type, Content: cell.Content, Rank: cell.Rank, SubRank: cell.SubRank, Region: regionID})
		}
	}
	out, err := filterClearedCells(candidates)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Trạng thái bản đồ tạm thời không khả dụng."})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GetCellsRect resolves the non-empty cells inside an explicit rectangle
// (min_x/min_y/max_x/max_y). Used by the world map to prefetch the exact
// visible area or a band ahead of the player. Bounded to prevent abuse.
func GetCellsRect(c *gin.Context) {
	if _, exists := middleware.GetPlayerID(c); !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var minX, minY, maxX, maxY int
	fmt.Sscanf(c.Query("min_x"), "%d", &minX)
	fmt.Sscanf(c.Query("min_y"), "%d", &minY)
	fmt.Sscanf(c.Query("max_x"), "%d", &maxX)
	fmt.Sscanf(c.Query("max_y"), "%d", &maxY)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rectangle tọa độ không hợp lệ."})
		return
	}

	// Bound the resolved area so a single request stays cheap.
	const maxSide = 256
	if maxX-minX+1 > maxSide {
		maxX = minX + maxSide - 1
	}
	if maxY-minY+1 > maxSide {
		maxY = minY + maxSide - 1
	}

	candidates := []NearbyCell{}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			cell := ResolveCell(x, y)
			if cell.Type == "Empty" {
				continue
			}
			regionID := DeriveRegionAt(x, y)
			candidates = append(candidates, NearbyCell{X: cell.X, Y: cell.Y, Type: cell.Type, Content: cell.Content, Rank: cell.Rank, SubRank: cell.SubRank, Region: regionID})
		}
	}
	out, err := filterClearedCells(candidates)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Trạng thái bản đồ tạm thời không khả dụng."})
		return
	}
	c.JSON(http.StatusOK, out)
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
	if entry, ok := worldPosIndex.Get(authID.String()); ok {
		x, y, region = entry.X, entry.Y, entry.Region
	}
	if region == "" {
		if player, err := loadPlayerForStatus(db.DB, authID, false); err == nil {
			x, y, region = player.X, player.Y, player.CurrentRegion
		}
	}

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
	entries := worldPosIndex.Nearby(region, x, y, r, authID.String())
	if worldPosIndex.Len() == 0 {
		entries = queryNearbyPlayersSQL(c, authID, region, x, y, r)
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
func queryNearbyPlayersSQL(c *gin.Context, authID uuid.UUID, region string, x, y, r int) []WorldPosEntry {
	entries := []WorldPosEntry{}
	rows, errQuery := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, username, COALESCE(avatar_url, ''), x, y, COALESCE(current_region, '')
		FROM players
		WHERE id != $1
		  AND (last_activity >= NOW() - INTERVAL '5 minutes' OR last_mana_update >= NOW() - INTERVAL '5 minutes')
		  AND x >= $2 AND x <= $3
		  AND y >= $4 AND y <= $5
		  AND current_region = $6
	`, authID, x-r, x+r, y-r, y+r, region)
	if errQuery != nil {
		logger.S.Error("Failed to fetch nearby players:", errQuery)
		return entries
	}
	defer rows.Close()
	for rows.Next() {
		var e WorldPosEntry
		if errScan := rows.Scan(&e.ID, &e.Username, &e.AvatarURL, &e.X, &e.Y, &e.Region); errScan == nil {
			e.TS = time.Now().UnixMilli()
			entries = append(entries, e)
		}
	}
	return entries
}
