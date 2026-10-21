package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// MaxMapVisionRadius caps how many tiles a player can see around themselves
// on the world map, regardless of stacked INVESTIGATION stats.
const MaxMapVisionRadius = 25

// GetGrid retrieves the viewport viewport within vision range
func GetGrid(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	statsOnly := c.Query("stats_only") == "true"
	var player models.Player
	var err error
	if statsOnly {
		// Range-only UI reads do not need to persist mana or acquire a write lock.
		player, err = PlayerSvc.ProjectStatusEquipped(db.DB, authID)
	} else {
		// Full grid reads persist mana only through the existing throttled path.
		player, err = PlayerSvc.ProjectAndPersistBestEffort(authID, false)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	cleaned := cleanupExpiredClearedCells(&player, clearedCellCooldown)
	if cleaned {
		_, err = db.DB.Exec("UPDATE players SET cleared_cells = $1 WHERE id = $2", player.ClearedCells, player.ID)
		if err != nil {
			logger.S.Error("Failed to save cleaned up cleared_cells:", err)
		}
	}

	stats := CalculatePlayerStats(&player)
	visionRadius := int(stats["investigation"])
	if visionRadius < 1 {
		visionRadius = 1
	}
	// Hard upper cap on map vision so stacked INVESTIGATION stats cannot
	// reveal the entire world. The raw stat is hidden behind this cap for
	// every consumer of the grid viewport.
	if visionRadius > MaxMapVisionRadius {
		visionRadius = MaxMapVisionRadius
	}
	// The client only reads movement + investigation; skip shipping the rest
	// of the stat block in every viewport fetch.
	gridStats := gin.H{"movement": stats["movement"], "investigation": visionRadius}
	visualVisionRadius := visionRadius
	if visualVisionRadius > 2 {
		visualVisionRadius = 2
	}
	if statsOnly {
		c.JSON(http.StatusOK, gin.H{
			"x":             player.X,
			"y":             player.Y,
			"vision_radius": visualVisionRadius,
			"stats":         gridStats,
			"world_width":   WorldWidth,
			"world_height":  WorldHeight,
		})
		return
	}

	mapID := worldMapIDForRegion(player.CurrentRegion)
	mapInstanceID := mapInstanceIDForPlayer(player.ID.String())

	cellCoords := make([]cache.CellCoordKey, 0, (2*visualVisionRadius+1)*(2*visualVisionRadius+1))
	for dx := -visualVisionRadius; dx <= visualVisionRadius; dx++ {
		for dy := -visualVisionRadius; dy <= visualVisionRadius; dy++ {
			cx := player.X + dx
			cy := player.Y + dy
			if cx >= 0 && cx < WorldWidth && cy >= 0 && cy < WorldHeight {
				cellCoords = append(cellCoords, cache.CellCoordKey{RegionID: player.CurrentRegion, X: cx, Y: cy})
			}
		}
	}
	clearedCells, errCleared := cache.GetClearedCells(cellCoords)
	if errCleared != nil {
		logger.S.Errorf("Failed to batch-read cleared map cells: %v", errCleared)
		// Preserve the old per-cell fallback behavior if Redis cannot serve MGET.
		clearedCells = make(map[cache.CellCoordKey]cache.MapCellState)
		for _, coord := range cellCoords {
			cleared, clearedBy, err := cache.IsCellCleared(coord.RegionID, coord.X, coord.Y)
			if err != nil {
				logger.S.Errorf("Failed to check Redis cell state for (%d,%d) in region %s: %v", coord.X, coord.Y, coord.RegionID, err)
				continue
			}
			if cleared {
				clearedCells[coord] = cache.MapCellState{ClearedBy: clearedBy}
			}
		}
	}

	cells := make([]models.Cell, 0, len(cellCoords))
	for _, coord := range cellCoords {
		cell := ResolveCellInRegion(coord.RegionID, coord.X, coord.Y)
		if _, isCleared := clearedCells[coord]; isCleared {
			cell = models.Cell{X: coord.X, Y: coord.Y, Type: "Empty", Content: ""}
		}
		cells = append(cells, cell)
	}
	if worldMapIDForRegion(player.CurrentRegion) == "world_map" {
		cellIndex := make(map[[2]int]int, len(cells))
		for i, cell := range cells {
			cellIndex[[2]int{cell.X, cell.Y}] = i
		}
		for _, vc := range veinCellsInBounds(player.X-visualVisionRadius, player.Y-visualVisionRadius, player.X+visualVisionRadius, player.Y+visualVisionRadius) {
			veinCell := models.Cell{X: vc.X, Y: vc.Y, Type: vc.Type, Content: vc.Content}
			if i, ok := cellIndex[[2]int{vc.X, vc.Y}]; ok {
				cells[i] = veinCell
				continue
			}
			cellIndex[[2]int{vc.X, vc.Y}] = len(cells)
			cells = append(cells, veinCell)
		}
	}

	type NearbyPlayer struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		X         int    `json:"x"`
		Y         int    `json:"y"`
	}
	nearbyPlayers := []NearbyPlayer{}
	seenIDs := map[string]bool{}

	// Radius-based nearby players come from the in-memory position index
	// (fast path). SQL fallback only while the index is cold.
	entries := worldPosIndex.NearbyInScope(player.X, player.Y, visualVisionRadius, player.ID.String(), mapID, mapInstanceID)
	if worldPosIndex.Len() == 0 {
		entries = queryNearbyPlayersSQL(c, player.ID, player.X, player.Y, visualVisionRadius, mapID, mapInstanceID)
	}
	for _, e := range entries {
		seenIDs[e.ID] = true
		nearbyPlayers = append(nearbyPlayers, NearbyPlayer{ID: e.ID, Username: e.Username, AvatarURL: e.AvatarURL, X: e.X, Y: e.Y})
	}

	// Always include party members regardless of distance for position sync
	if player.PartyID != nil {
		partyRows, errParty := db.DB.Query(`
			SELECT p.id, p.username, COALESCE(p.avatar_url,''), p.x, p.y
			FROM players p
			LEFT JOIN region_world_maps rwm ON rwm.region_id = p.current_region
			WHERE p.party_id = $1
			  AND p.id != $2
			  AND COALESCE(rwm.world_map_id,'world_map') = $3
			  AND COALESCE(p.map_instance_id::text,'') = $4
		`, player.PartyID, player.ID, mapID, mapInstanceID)
		if errParty == nil {
			defer partyRows.Close()
			for partyRows.Next() {
				var np NearbyPlayer
				if errScan := partyRows.Scan(&np.ID, &np.Username, &np.AvatarURL, &np.X, &np.Y); errScan == nil {
					if !seenIDs[np.ID] {
						nearbyPlayers = append(nearbyPlayers, np)
						seenIDs[np.ID] = true
					}
				}
			}
		}
	}

	// Ground drops query within visible radius
	type GroundDropItem struct {
		ID             string         `json:"id"`
		RegionID       string         `json:"region_id"`
		X              int            `json:"x"`
		Y              int            `json:"y"`
		LoserID        string         `json:"loser_id"`
		LoserName      string         `json:"loser_name"`
		KillerID       string         `json:"killer_id,omitempty"`
		KillerName     string         `json:"killer_name,omitempty"`
		LootData       GroundLootData `json:"loot_data"`
		IsProtected    bool           `json:"is_protected"`
		ProtectedUntil *time.Time     `json:"protected_until,omitempty"`
		ExpiresAt      time.Time      `json:"expires_at"`
		CreatedAt      time.Time      `json:"created_at"`
	}

	groundDrops := []GroundDropItem{}
	minX := player.X - visualVisionRadius
	maxX := player.X + visualVisionRadius
	minY := player.Y - visualVisionRadius
	maxY := player.Y + visualVisionRadius

	dropRows, errDrops := db.DB.Query(`
		SELECT id, region_id, x, y, loser_id, loser_name, killer_id, killer_name,
		       loot_data, protected_until, expires_at, created_at
		FROM map_ground_drops
		WHERE region_id = $1
		  AND x BETWEEN $2 AND $3
		  AND y BETWEEN $4 AND $5
		  AND expires_at > NOW()
		ORDER BY created_at DESC
	`, player.CurrentRegion, minX, maxX, minY, maxY)
	if errDrops == nil {
		defer dropRows.Close()
		now := time.Now()
		for dropRows.Next() {
			var d GroundDropItem
			var killerID, killerName sql.NullString
			var lootBytes []byte
			var protectedUntil sql.NullTime

			if errScan := dropRows.Scan(
				&d.ID, &d.RegionID, &d.X, &d.Y, &d.LoserID, &d.LoserName,
				&killerID, &killerName, &lootBytes, &protectedUntil, &d.ExpiresAt, &d.CreatedAt,
			); errScan == nil {
				if IsSafeZone(d.RegionID, d.X, d.Y) {
					continue
				}
				if killerID.Valid {
					d.KillerID = killerID.String
				}
				if killerName.Valid {
					d.KillerName = killerName.String
				}
				if protectedUntil.Valid {
					d.ProtectedUntil = &protectedUntil.Time
				}
				json.Unmarshal(lootBytes, &d.LootData)

				if protectedUntil.Valid && protectedUntil.Time.After(now) {
					if killerID.Valid && killerID.String != player.ID.String() {
						d.IsProtected = true
					}
				}

				groundDrops = append(groundDrops, d)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"x":              player.X,
		"y":              player.Y,
		"vision_radius":  visualVisionRadius,
		"cells":          cells,
		"stats":          gridStats,
		"players_on_map": nearbyPlayers,
		"ground_drops":   groundDrops,
		"world_width":    WorldWidth,
		"world_height":   WorldHeight,
	})
}

// HasExplorationInventorySpace checks bag capacity of the real inventory.
func HasExplorationInventorySpace(player *models.Player, incomingCount int, territory SectTerritoryInfo) bool {
	stats := calculatePlayerStatsWithTerritory(player, territory)
	maxSlots := int(stats["storage"])
	if player.Rank >= 6 {
		maxSlots = 999
	}

	currentSize := len(player.GuInventory) + len(player.MaterialsInventory)
	if player.MaterialsInventory == nil && InventorySvc != nil {
		// loadRow normally supplies the aggregated map already. Keep the old
		// fallback for callers whose player snapshot could not load inventory.
		invItems, _ := InventorySvc.ListByPlayer(player.ID.String())
		seen := make(map[string]struct{}, len(invItems))
		for _, inv := range invItems {
			seen[inv.ItemID] = struct{}{}
		}
		currentSize += len(seen)
	}

	return (currentSize + incomingCount) <= maxSlots
}

// resolveCellArrival places the player at (x, y), derives the region / subzone
// and resolves the cell content the player stepped onto (resource pickup, wild
// Gu capture, monster / boss combat or player-initiated PVP), including the
// small adjacency ambush chance (skipped when allowAmbush is false — used by
// free path movement so the player only fights when stepping onto an actual
// monster cell, not randomly on empty cells). It mutates player in-place and
