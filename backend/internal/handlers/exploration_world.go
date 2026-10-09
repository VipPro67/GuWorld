package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// GetRegions fetches all active regions (zones) from the database with ETag caching
func GetRegions(c *gin.Context) {
	getCachedDictionaryPartWithError(c, "explore_regions_v2", &[]models.Region{}, func() (interface{}, error) {
		query := `
			SELECT id, name, description, width, height, min_rank, max_rank, is_active, regions.created_at, regions.updated_at,
			       COALESCE(parent_id,''), COALESCE(zone_type,'KHU_VUC'), COALESCE(min_x,0), COALESCE(min_y,0),
			       COALESCE(max_x,0), COALESCE(max_y,0), COALESCE(entry_x,0), COALESCE(entry_y,0),
			       COALESCE(rwm.world_map_id, 'world_map'), COALESCE(regions.rules::text,'{}')
			FROM regions
			LEFT JOIN region_world_maps rwm ON rwm.region_id = regions.id
			WHERE is_active = TRUE
			ORDER BY min_rank ASC, name ASC
		`
		rows, err := db.DB.QueryContext(c.Request.Context(), query)
		if err != nil {
			return nil, fmt.Errorf("query active regions: %w", err)
		}
		defer rows.Close()

		regions := make([]models.Region, 0)
		for rows.Next() {
			var r models.Region
			var rulesJSON string
			if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Width, &r.Height, &r.MinRank, &r.MaxRank, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
				&r.ParentID, &r.ZoneType, &r.MinX, &r.MinY, &r.MaxX, &r.MaxY, &r.EntryX, &r.EntryY, &r.WorldMapID, &rulesJSON); err != nil {
				return nil, fmt.Errorf("scan active region: %w", err)
			}
			r.Rules = json.RawMessage(rulesJSON)
			if effectiveRules, err := regionRulesFor(r.ID); err == nil {
				if raw, err := json.Marshal(effectiveRules); err == nil {
					r.EffectiveRules = raw
				}
			}
			regions = append(regions, r)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate active regions: %w", err)
		}
		return regions, nil
	}, func(value interface{}) bool {
		regions, ok := value.(*[]models.Region)
		if ok {
			return len(*regions) > 0
		}
		builtRegions, ok := value.([]models.Region)
		return ok && len(builtRegions) > 0
	})
}

// GetWorldMaps exposes the configured world layers. Inactive layers stay
// visible to admin/development clients but cannot yet be entered by players.
func GetWorldMaps(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, name, description, map_type, width, height, min_rank, max_rank,
		       COALESCE(rules::text, '{}'), is_active
		FROM world_maps ORDER BY CASE id WHEN 'world_map' THEN 0 WHEN 'bai_thien' THEN 1 WHEN 'hac_thien' THEN 2 ELSE 3 END, id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách tầng bản đồ"})
		return
	}
	defer rows.Close()
	type worldMap struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		MapType     string          `json:"map_type"`
		Width       int             `json:"width"`
		Height      int             `json:"height"`
		MinRank     int             `json:"min_rank"`
		MaxRank     int             `json:"max_rank"`
		Rules       json.RawMessage `json:"rules"`
		IsActive    bool            `json:"is_active"`
	}
	maps := make([]worldMap, 0)
	for rows.Next() {
		var item worldMap
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.MapType, &item.Width, &item.Height, &item.MinRank, &item.MaxRank, &item.Rules, &item.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu tầng bản đồ"})
			return
		}
		maps = append(maps, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách tầng bản đồ"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"maps": maps})
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
		SET current_mana = $1, x = $2, y = $3, current_region = $4, cleared_cells = $5, active_combat = $6, last_mana_update = $7, primeval_stones = $8, hp = $9, max_mana = $10, human_qi = $11, visited_subzones = $12, statuses = $13, version = version + 1
		WHERE id = $14
	`, player.CurrentMana, player.X, player.Y, player.CurrentRegion, player.ClearedCells, combatJSON, time.Now(), player.PrimevalStones, player.HP, player.MaxMana, player.HumanQi, player.VisitedSubZones, player.Statuses, player.ID)
	if err == nil {
		player.Version++
	}
	return err
}
