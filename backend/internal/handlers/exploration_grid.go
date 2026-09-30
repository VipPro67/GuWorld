package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// GetGrid retrieves the viewport viewport within vision range
func GetGrid(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Lock-free projection for viewport polls. Persist throttled in a short
	// Tx; viewport generation itself must not hold the players row lock.
	player, err := PlayerSvc.ProjectAndPersistBestEffort(authID, false)
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
	// The client only reads movement + investigation; skip shipping the rest
	// of the stat block in every viewport fetch.
	gridStats := gin.H{"movement": stats["movement"], "investigation": stats["investigation"]}

	// Cap the visual vision radius to 2 (corresponding to a 5x5 grid) to prevent UI breakage
	visualVisionRadius := visionRadius
	if visualVisionRadius > 2 {
		visualVisionRadius = 2
	}

	cells := []models.Cell{}
	for dx := -visualVisionRadius; dx <= visualVisionRadius; dx++ {
		for dy := -visualVisionRadius; dy <= visualVisionRadius; dy++ {
			cx := player.X + dx
			cy := player.Y + dy
			if cx >= 0 && cx < WorldWidth && cy >= 0 && cy < WorldHeight {
				cell := ResolveCell(cx, cy)
				if isCellClearedRecently(player.CurrentRegion, cx, cy, clearedCellCooldown) {
					cell = models.Cell{X: cx, Y: cy, Type: "Empty", Content: ""}
				}
				cells = append(cells, cell)
			}
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
	entries := worldPosIndex.Nearby(player.X, player.Y, visualVisionRadius, player.ID.String())
	if worldPosIndex.Len() == 0 {
		entries = queryNearbyPlayersSQL(c, player.ID, player.X, player.Y, visualVisionRadius)
	}
	for _, e := range entries {
		seenIDs[e.ID] = true
		nearbyPlayers = append(nearbyPlayers, NearbyPlayer{ID: e.ID, Username: e.Username, AvatarURL: e.AvatarURL, X: e.X, Y: e.Y})
	}

	// Always include party members regardless of distance for position sync
	if player.PartyID != nil {
		partyRows, errParty := db.DB.Query(`
			SELECT id, username, avatar_url, x, y
			FROM players
			WHERE party_id = $1
			  AND id != $2
		`, player.PartyID, player.ID)
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
func HasExplorationInventorySpace(player *models.Player, incomingCount int) bool {
	stats := CalculatePlayerStats(player)
	maxSlots := int(stats["storage"])
	if player.Rank >= 6 {
		maxSlots = 999
	}

	currentSize := len(player.GuInventory)
	invItems, _ := InventorySvc.ListByPlayer(player.ID.String())
	seen := map[string]bool{}
	for _, inv := range invItems {
		if !seen[inv.ItemID] {
			seen[inv.ItemID] = true
			currentSize++
		}
	}

	return (currentSize + incomingCount) <= maxSlots
}

// resolveCellArrival places the player at (x, y), derives the region / subzone
// and resolves the cell content the player stepped onto (resource pickup, wild
// Gu capture, monster / boss combat or player-initiated PVP), including the
// small adjacency ambush chance (skipped when allowAmbush is false — used by
// free path movement so the player only fights when stepping onto an actual
// monster cell, not randomly on empty cells). It mutates player in-place and
