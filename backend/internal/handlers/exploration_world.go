package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// GetRegions fetches all active regions (zones) from the database with ETag caching
func GetRegions(c *gin.Context) {
	getCachedDictionaryPartWithError(c, "explore_regions_v3", &[]models.Region{}, func() (interface{}, error) {
		query := `
			SELECT regions.id, regions.name, regions.description, regions.width, regions.height, regions.min_rank, regions.max_rank, regions.is_active, regions.created_at, regions.updated_at,
			       COALESCE(parent_id,''), COALESCE(zone_type,'KHU_VUC'), COALESCE(min_x,0), COALESCE(min_y,0),
			       COALESCE(max_x,0), COALESCE(max_y,0), COALESCE(entry_x,0), COALESCE(entry_y,0),
			       COALESCE(rwm.world_map_id, 'world_map'), COALESCE(regions.rules::text,'{}'),
			       COALESCE(ms.has_boss, FALSE) OR (regions.zone_type = 'BOSS_ZONE') AS has_boss,
			       COALESCE(ms.monster_count, 0) AS monster_count,
			       COALESCE(regions.atlas_x, 0), COALESCE(regions.atlas_y, 0)
			FROM regions
			LEFT JOIN region_world_maps rwm ON rwm.region_id = regions.id
			LEFT JOIN (
				SELECT region_id,
				       BOOL_OR(is_boss_spawn = TRUE) AS has_boss,
				       COUNT(DISTINCT monster_id) AS monster_count
				FROM monster_spawns
				WHERE is_active = TRUE
				GROUP BY region_id
			) ms ON ms.region_id = regions.id
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
				&r.ParentID, &r.ZoneType, &r.MinX, &r.MinY, &r.MaxX, &r.MaxY, &r.EntryX, &r.EntryY, &r.WorldMapID, &rulesJSON,
				&r.HasBoss, &r.MonsterCount, &r.AtlasX, &r.AtlasY); err != nil {
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

type RegionMonsterDetail struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"` // Normal, Elite, Boss
	MinRank      int    `json:"min_rank"`
	MaxRank      int    `json:"max_rank"`
	IsBoss       bool   `json:"is_boss"`
	IsCapturable bool   `json:"is_capturable"`
	// MaxHP/BaseAttack/BaseDefense are the template stats scaled to MinRank;
	// the *Max fields are the same stats at MaxRank. They differ only when a
	// spawn row spans a rank range.
	MaxHP       float64 `json:"max_hp"`
	BaseAttack  float64 `json:"base_attack"`
	BaseDefense float64 `json:"base_defense"`
	MaxHPMax    float64 `json:"max_hp_max"`
	AttackMax   float64 `json:"attack_max"`
	DefenseMax  float64 `json:"defense_max"`
	Description string  `json:"description"`

	// base* hold the raw, rank-independent template stats used to compute the
	// scaled values once the final rank range is known.
	baseHP  float64
	baseAtk float64
	baseDef float64
}

type RegionDropDetail struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Kind    string   `json:"kind"` // "item" | "gu" | "resource"
	Rank    int      `json:"rank,omitempty"`
	Chance  float64  `json:"chance"`
	MinQty  int      `json:"min_qty"`
	MaxQty  int      `json:"max_qty"`
	Sources []string `json:"sources,omitempty"`
}

type NamedResource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RegionDetailsResponse struct {
	RegionID     string                `json:"region_id"`
	Name         string                `json:"name"`
	ZoneType     string                `json:"zone_type"`
	MinRank      int                   `json:"min_rank"`
	MaxRank      int                   `json:"max_rank"`
	Description  string                `json:"description"`
	Monsters     []RegionMonsterDetail `json:"monsters"`
	Drops        []RegionDropDetail    `json:"drops"`
	WildGus      []NamedResource       `json:"wild_gus"`
	Resources    []NamedResource       `json:"resources"`
	HasBoss      bool                  `json:"has_boss"`
	MonsterCount int                   `json:"monster_count"`
}

func resolveGuName(guID string) string {
	if ranks, ok := models.GuRegistry[guID]; ok {
		for _, item := range ranks {
			if item.Name != "" {
				return item.Name
			}
		}
	}
	return guID
}

func resolveItemName(itemID string) string {
	if item, ok := models.ItemRegistry[itemID]; ok && item.Name != "" {
		return item.Name
	}
	return itemID
}

// GetRegionDetails returns the monsters, drop tables, and resources for a region.
func GetRegionDetails(c *gin.Context) {
	regionID := c.Param("id")
	if regionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã khu vực"})
		return
	}

	var reg models.Region
	var poolRaw, rulesRaw string
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, name, COALESCE(description,''), min_rank, max_rank, COALESCE(zone_type,'KHU_VUC'),
		       COALESCE(pool_weights::text,'{}'), COALESCE(rules::text,'{}')
		FROM regions WHERE id = $1
	`, regionID).Scan(&reg.ID, &reg.Name, &reg.Description, &reg.MinRank, &reg.MaxRank, &reg.ZoneType, &poolRaw, &rulesRaw)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Khu vực không tồn tại"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn khu vực"})
		return
	}
	if poolRaw != "" && poolRaw != "{}" {
		_ = json.Unmarshal([]byte(poolRaw), &reg.PoolWeights)
	}

	// 1. Quái vật từ monster_spawns
	monstersMap := make(map[string]*RegionMonsterDetail)
	lootTablesToMonsters := make(map[string][]string) // lootTableID -> []monsterName

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT monster_id, COALESCE(min_rank, 1), COALESCE(max_rank, 9), COALESCE(is_boss_spawn, FALSE)
		FROM monster_spawns
		WHERE region_id = $1 AND is_active = TRUE
	`, regionID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var mID string
			var minR, maxR int
			var isBoss bool
			if err := rows.Scan(&mID, &minR, &maxR, &isBoss); err == nil {
				detail, exists := monstersMap[mID]
				if !exists {
					tpl, hasTpl := models.MonsterRegistry[mID]
					name := mID
					mType := "Normal"
					var hp, atk, def float64
					var desc string
					var capturable bool
					if hasTpl {
						if tpl.Name != "" {
							name = tpl.Name
						}
						mType = tpl.Type
						hp = tpl.MaxHP
						atk = tpl.BaseAttack
						def = tpl.BaseDefense
						desc = tpl.Description
						capturable = tpl.IsCapturable
						if tpl.LootTableID != "" {
							lootTablesToMonsters[tpl.LootTableID] = append(lootTablesToMonsters[tpl.LootTableID], name)
						}
					}
					if isBoss || mType == "Boss" {
						isBoss = true
						mType = "Boss"
					}
					monstersMap[mID] = &RegionMonsterDetail{
						ID:           mID,
						Name:         name,
						Type:         mType,
						MinRank:      minR,
						MaxRank:      maxR,
						IsBoss:       isBoss,
						IsCapturable: capturable,
						Description:  desc,
						baseHP:       hp,
						baseAtk:      atk,
						baseDef:      def,
					}
				} else {
					if minR < detail.MinRank {
						detail.MinRank = minR
					}
					if maxR > detail.MaxRank {
						detail.MaxRank = maxR
					}
					if isBoss {
						detail.IsBoss = true
						detail.Type = "Boss"
					}
				}
			}
		}
	}

	// Quái Boss từ map_cells
	bossCells, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT content_id, COALESCE(rank, 1)
		FROM map_cells
		WHERE region_id = $1 AND cell_type = 'Boss' AND is_active = TRUE
	`, regionID)
	if err == nil {
		defer bossCells.Close()
		for bossCells.Next() {
			var mID string
			var rank int
			if err := bossCells.Scan(&mID, &rank); err == nil && mID != "" {
				if d, ok := monstersMap[mID]; ok {
					d.IsBoss = true
					d.Type = "Boss"
				} else {
					tpl, hasTpl := models.MonsterRegistry[mID]
					name := mID
					var hp, atk, def float64
					var desc string
					if hasTpl {
						if tpl.Name != "" {
							name = tpl.Name
						}
						hp = tpl.MaxHP
						atk = tpl.BaseAttack
						def = tpl.BaseDefense
						desc = tpl.Description
						if tpl.LootTableID != "" {
							lootTablesToMonsters[tpl.LootTableID] = append(lootTablesToMonsters[tpl.LootTableID], name)
						}
					}
					monstersMap[mID] = &RegionMonsterDetail{
						ID:          mID,
						Name:        name,
						Type:        "Boss",
						MinRank:     rank,
						MaxRank:     rank,
						IsBoss:      true,
						Description: desc,
						baseHP:      hp,
						baseAtk:     atk,
						baseDef:     def,
					}
				}
			}
		}
	}

	// 2. Tìm danh sách Loot Tables:
	// A) Loot tables từ RegionLootTables (cho khu vực và cả cha của nó)
	scope := []string{regionID}
	var parentID sql.NullString
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT parent_id FROM regions WHERE id = $1`, regionID).Scan(&parentID)
	if parentID.Valid && parentID.String != "" {
		scope = append(scope, parentID.String)
	}

	allLootTableIDs := make(map[string]bool)
	for ltID := range lootTablesToMonsters {
		allLootTableIDs[ltID] = true
	}

	rltRows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT loot_table_id, monster_type, chance
		FROM region_loot_tables
		WHERE region_id = ANY($1) AND is_active = TRUE
	`, pq.Array(scope))
	if err == nil {
		defer rltRows.Close()
		for rltRows.Next() {
			var ltID, mType string
			var chance float64
			if err := rltRows.Scan(&ltID, &mType, &chance); err == nil && ltID != "" {
				allLootTableIDs[ltID] = true
				sourceLabel := fmt.Sprintf("Khu vực (%s)", mType)
				lootTablesToMonsters[ltID] = append(lootTablesToMonsters[ltID], sourceLabel)
			}
		}
	}

	// 3. Nạp chi tiết các loot tables từ DB
	dropsMap := make(map[string]*RegionDropDetail)
	if len(allLootTableIDs) > 0 {
		ltIDList := make([]string, 0, len(allLootTableIDs))
		for id := range allLootTableIDs {
			ltIDList = append(ltIDList, id)
		}
		ltRows, err := db.DB.QueryContext(c.Request.Context(), `
			SELECT id, entries::text
			FROM loot_tables
			WHERE id = ANY($1)
		`, pq.Array(ltIDList))
		if err == nil {
			defer ltRows.Close()
			for ltRows.Next() {
				var ltID, entriesRaw string
				if err := ltRows.Scan(&ltID, &entriesRaw); err == nil && entriesRaw != "" {
					var entries []models.LootEntry
					if json.Unmarshal([]byte(entriesRaw), &entries) == nil {
						sources := lootTablesToMonsters[ltID]
						for _, e := range entries {
							if e.ItemID != "" {
								key := "item:" + e.ItemID
								if existing, ok := dropsMap[key]; ok {
									if e.Chance > existing.Chance {
										existing.Chance = e.Chance
									}
									if e.MaxQty > existing.MaxQty {
										existing.MaxQty = e.MaxQty
									}
									for _, s := range sources {
										existing.Sources = appendUnique(existing.Sources, s)
									}
								} else {
									dropsMap[key] = &RegionDropDetail{
										ID:      e.ItemID,
										Name:    resolveItemName(e.ItemID),
										Kind:    "item",
										Rank:    e.Rank,
										Chance:  e.Chance,
										MinQty:  e.MinQty,
										MaxQty:  e.MaxQty,
										Sources: appendUnique(nil, sources...),
									}
								}
							}
							if e.GuID != "" {
								key := "gu:" + e.GuID
								if existing, ok := dropsMap[key]; ok {
									if e.Chance > existing.Chance {
										existing.Chance = e.Chance
									}
									for _, s := range sources {
										existing.Sources = appendUnique(existing.Sources, s)
									}
								} else {
									dropsMap[key] = &RegionDropDetail{
										ID:      e.GuID,
										Name:    resolveGuName(e.GuID),
										Kind:    "gu",
										Rank:    e.Rank,
										Chance:  e.Chance,
										MinQty:  e.MinQty,
										MaxQty:  e.MaxQty,
										Sources: appendUnique(nil, sources...),
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 4. Wild Gus & Resources từ PoolWeights
	wildGusList := make([]NamedResource, 0)
	for _, guID := range reg.PoolWeights.WildGus {
		if guID != "" {
			wildGusList = append(wildGusList, NamedResource{
				ID:   guID,
				Name: resolveGuName(guID),
			})
		}
	}

	resourcesList := make([]NamedResource, 0)
	for _, resID := range reg.PoolWeights.Resources {
		if resID != "" {
			resourcesList = append(resourcesList, NamedResource{
				ID:   resID,
				Name: resolveItemName(resID),
			})
		}
	}

	// Chuyển maps thành slices
	monstersList := make([]RegionMonsterDetail, 0, len(monstersMap))
	hasBoss := reg.ZoneType == "BOSS_ZONE"
	for _, m := range monstersMap {
		if m.IsBoss {
			hasBoss = true
		}
		// Scale the raw template stats to the rank range the monster spawns at
		// (subRank 1). Without this the list shows rank-independent base stats,
		// which read far weaker than the monster actually is at its rank.
		minR, maxR := m.MinRank, m.MaxRank
		if minR < 1 {
			minR = 1
		}
		if maxR < minR {
			maxR = minR
		}
		m.MaxHP, m.BaseAttack, m.BaseDefense = models.ScaleMonsterStats(m.baseHP, m.baseAtk, m.baseDef, minR, 1, m.Type)
		m.MaxHPMax, m.AttackMax, m.DefenseMax = models.ScaleMonsterStats(m.baseHP, m.baseAtk, m.baseDef, maxR, 1, m.Type)
		monstersList = append(monstersList, *m)
	}

	dropsList := make([]RegionDropDetail, 0, len(dropsMap))
	for _, d := range dropsMap {
		dropsList = append(dropsList, *d)
	}

	c.JSON(http.StatusOK, RegionDetailsResponse{
		RegionID:     reg.ID,
		Name:         reg.Name,
		ZoneType:     reg.ZoneType,
		MinRank:      reg.MinRank,
		MaxRank:      reg.MaxRank,
		Description:  reg.Description,
		Monsters:     monstersList,
		Drops:        dropsList,
		WildGus:      wildGusList,
		Resources:    resourcesList,
		HasBoss:      hasBoss,
		MonsterCount: len(monstersList),
	})
}

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice)+len(items))
	for _, s := range slice {
		seen[s] = true
	}
	for _, it := range items {
		if it != "" && !seen[it] {
			slice = append(slice, it)
			seen[it] = true
		}
	}
	return slice
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
