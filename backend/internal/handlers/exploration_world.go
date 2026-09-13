package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// GetRegions fetches all active regions (zones) from the database
func GetRegions(c *gin.Context) {
	var regions []models.Region
	if err := cache.GetJSON("explore_regions", &regions); err == nil && len(regions) > 0 {
		c.JSON(http.StatusOK, regions)
		return
	}

	query := `
		SELECT id, name, description, width, height, min_rank, max_rank, is_active, created_at, updated_at,
		       COALESCE(parent_id,''), COALESCE(zone_type,'KHU_VUC'), COALESCE(min_x,0), COALESCE(min_y,0),
		       COALESCE(max_x,0), COALESCE(max_y,0), COALESCE(entry_x,0), COALESCE(entry_y,0)
		FROM regions
		WHERE is_active = TRUE
		ORDER BY min_rank ASC, name ASC
	`
	rows, err := db.DB.QueryContext(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải dữ liệu bản đồ"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var r models.Region
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Width, &r.Height, &r.MinRank, &r.MaxRank, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
			&r.ParentID, &r.ZoneType, &r.MinX, &r.MinY, &r.MaxX, &r.MaxY, &r.EntryX, &r.EntryY); err != nil {
			continue
		}
		regions = append(regions, r)
	}

	_ = cache.SetJSON("explore_regions", regions, 24*time.Hour)
	c.JSON(http.StatusOK, regions)
}

// GetWorld returns the shared world geometry for the frontend overview.
func GetWorld(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"width":           WorldWidth,
		"height":          WorldHeight,
		"large_zone_size": LargeZoneSize,
		"sub_zone_size":   SubZoneSize,
		"hub_region_id":   HubRegionID,
		"hub_x":           HubEntryPoint.X,
		"hub_y":           HubEntryPoint.Y,
	})
}

func savePlayerMovement(tx *sql.Tx, player *models.Player, combatJSON any) error {
	if ZoneByID(player.CurrentRegion) == nil {
		player.CurrentRegion = HubRegionID
	}

	// Movement awards resources and may mutate hunger/progress in the same
	// transaction. Persist the complete authoritative result synchronously so a
	// Redis snapshot cannot survive a rollback or overwrite a newer mutation.
	_, err := tx.Exec(`
		UPDATE players
		SET current_mana = $1, x = $2, y = $3, current_region = $4, cleared_cells = $5, active_combat = $6, last_mana_update = $7, primeval_stones = $8, hp = $9, max_mana = $10, human_qi = $11, visited_subzones = $12, version = version + 1
		WHERE id = $13
	`, player.CurrentMana, player.X, player.Y, player.CurrentRegion, player.ClearedCells, combatJSON, time.Now(), player.PrimevalStones, player.HP, player.MaxMana, player.HumanQi, player.VisitedSubZones, player.ID)
	if err == nil {
		player.Version++
	}
	return err
}
