package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func writeImportError(c *gin.Context, prefix string, err error) {
	var validation *effectcatalog.ValidationError
	if errors.As(err, &validation) {
		c.JSON(http.StatusBadRequest, gin.H{"error": prefix, "issues": validation.Issues})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": prefix + ": " + err.Error()})
}

// AdminImportData seeds all dictionary tables from CSV files.
// POST /api/admin/data/import
func AdminImportData(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	if err := db.SeedFromCSV(); err != nil {
		writeImportError(c, "Import thất bại", err)
		return
	}

	// Reload registries
	if err := models.LoadGuRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Gu registry after import: %v", err)
	}
	if err := models.LoadItemRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Item registry after import: %v", err)
	}
	if err := gamefx.LoadConstitutionRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Constitution registry after import: %v", err)
	}
	if err := gamefx.LoadRaceRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Race registry after import: %v", err)
	}
	if err := gamefx.LoadStatusRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Status registry after import: %v", err)
	}
	if err := models.LoadMonsterRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Monster registry after import: %v", err)
	}
	if err := LoadQuestRegistry(); err != nil {
		logger.S.Warnf("failed to reload Quest registry after import: %v", err)
	}
	if err := InitInstanceCache(); err != nil {
		logger.S.Warnf("failed to reload Instance registry after import: %v", err)
	}
	InvalidateDictionaryMemoryCache()
	invalidateRecipeCatalog()

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.import_data",
		EntityType: "system", EntityID: "",
		RequestSummary: "data import from CSV",
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Import dữ liệu từ CSV thành công"})
}

// AdminExportData exports all dictionary tables to CSV files.
// POST /api/admin/data/export
func AdminExportData(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	if err := db.ExportToCSV(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Export thất bại: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.export_data",
		EntityType: "system", EntityID: "",
		RequestSummary: "data export to CSV",
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Export dữ liệu ra CSV thành công"})
}

// AdminListTables returns the list of tables that can be individually imported/exported.
// GET /api/admin/data/tables
func AdminListTables(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"tables": db.AvailableTables()})
}

// AdminImportTable imports a single table from its CSV file.
// POST /api/admin/data/import/:table
func AdminImportTable(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	table := c.Param("table")

	if err := db.TableImport(table); err != nil {
		writeImportError(c, "Import bảng "+table+" thất bại", err)
		return
	}

	// Reload relevant registries
	switch table {
	case "item":
		if err := models.LoadItemRegistry(db.DB); err != nil {
			logger.S.Warnf("failed to reload Item registry after import: %v", err)
		}
		cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	case "gu":
		if err := models.LoadGuRegistry(db.DB); err != nil {
			logger.S.Warnf("failed to reload Gu registry after import: %v", err)
		}
		cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	case "constitution":
		if err := gamefx.LoadConstitutionRegistry(db.DB); err != nil {
			logger.S.Warnf("failed to reload Constitution registry after import: %v", err)
		}
		cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	case "race":
		if err := gamefx.LoadRaceRegistry(db.DB); err != nil {
			logger.S.Warnf("failed to reload Race registry after import: %v", err)
		}
		cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	case "status":
		if err := gamefx.LoadStatusRegistry(db.DB); err != nil {
			logger.S.Warnf("failed to reload Status registry after import: %v", err)
		}
		cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	case "monster", "loot_table":
		if err := models.LoadMonsterRegistry(db.DB); err != nil {
			logger.S.Warnf("failed to reload Monster registry after import: %v", err)
		}
		cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	case "region", "map_cell", "monster_spawn":
		if err := LoadZoneRegistry(db.DB); err != nil {
			logger.S.Warnf("failed to reload Zone registry after import: %v", err)
		}
		if err := models.LoadMonsterRegistry(db.DB); err != nil {
			logger.S.Warnf("failed to reload Monster registry after import: %v", err)
		}
		cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	case "quest":
		if err := LoadQuestRegistry(); err != nil {
			logger.S.Warnf("failed to reload Quest registry after import: %v", err)
		}
	case "recipe":
		invalidateRecipeCatalog()
	}
	InvalidateDictionaryMemoryCache()

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.import_table",
		EntityType: "table", EntityID: table,
		RequestSummary: "import table " + table + " from CSV",
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Import bảng " + table + " thành công"})
}

// AdminExportTable exports a single table to its CSV file.
// POST /api/admin/data/export/:table
func AdminExportTable(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	table := c.Param("table")

	if err := db.TableExport(table); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Export bảng " + table + " thất bại: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.export_table",
		EntityType: "table", EntityID: table,
		RequestSummary: "export table " + table + " to CSV",
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Export bảng " + table + " thành công"})
}

// AdminClearCache reloads all in-memory registries from the database and clears Redis caches.
// POST /api/admin/clear-cache
func AdminClearCache(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	if err := models.LoadGuRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Gu registry: %v", err)
	}
	if err := models.LoadItemRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Item registry: %v", err)
	}
	if err := gamefx.LoadConstitutionRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Constitution registry: %v", err)
	}
	if err := gamefx.LoadRaceRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Race registry: %v", err)
	}
	if err := gamefx.LoadStatusRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Status registry: %v", err)
	}
	if err := models.LoadMonsterRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Monster registry: %v", err)
	}
	if err := LoadQuestRegistry(); err != nil {
		logger.S.Warnf("failed to reload Quest registry: %v", err)
	}
	if err := LoadZoneRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Zone registry: %v", err)
	}
	if err := InitInstanceCache(); err != nil {
		logger.S.Warnf("failed to reload Instance cache: %v", err)
	}

	// Gacha pools are derived from the Gu/Item/recipe registries — force a rebuild.
	gachaPoolsMu.Lock()
	globalPrecomputedPools = nil
	gachaPoolsMu.Unlock()

	// Clear all known Redis cache keys (dictionary part keys included so
	// /api/dictionary/* rebuild from the freshly loaded registries)
	cache.Delete(
		"game_dictionary_unified",
		"game_dictionary_gus",
		"game_dictionary_items",
		"game_dictionary_monsters",
		"game_dictionary_constitutions",
		"game_dictionary_races",
		"game_dictionary_statuses",
		"game_dictionary_loot_tables",
		"game_dictionary_regions",
		"game_dictionary_monster_spawns",
		"game_quests_list",
		"explore_regions",
		"arena_shop_items",
		"topup_packages",
		"premium_shop_packages",
		"mine_slots_list",
		"beast_betting_current_round",
		"sects_list",
		"master_refining_recipes",
	)
	// Pattern-based keys: profile pages, day-scoped caravan catalog,
	// feedback (list + per-feedback comments), user-scoped market listings.
	if cache.Client != nil {
		for _, pattern := range []string{
			"profile_*",
			"caravan_shop_catalog:*",
			"feedback_comments:*",
			"feedbacks_*",
			"market_listings_user_*",
		} {
			iter := cache.Client.Scan(cache.Ctx, 0, pattern, 100).Iterator()
			for iter.Next(cache.Ctx) {
				cache.Client.Del(cache.Ctx, iter.Val())
			}
		}
	}
	InvalidateDictionaryMemoryCache()
	invalidateRecipeCatalog()

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa và tải lại toàn bộ cache dữ liệu thành công!"})
}

// AdminClearRedisCache completely flushes all keys from the connected Redis instance.
// POST /api/admin/clear-redis-cache
func AdminClearRedisCache(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	if cache.Client != nil {
		if err := cache.Client.FlushAll(cache.Ctx).Err(); err != nil {
			logger.S.Errorf("Failed to flush redis cache: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa toàn bộ cache Redis"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa toàn bộ cache Redis thành công!"})
}

// AdminGetAuditLogs returns all audit logs (admin only) with optional filtering.
// GET /api/admin/audit-logs
func AdminGetAuditLogs(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	action := c.Query("action")
	entityType := c.Query("entity_type")
	playerIDStr := c.Query("player_id")
	successStr := c.Query("success")
	search := c.Query("search")
	paging := parseAdminPageQuery(c, 50, map[string]string{
		"created_at": "created_at", "action": "action", "entity_type": "entity_type", "success": "success",
	}, "created_at")

	query := `SELECT id, player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	argCount := 0

	if action != "" {
		argCount++
		query += " AND action = $" + strconv.Itoa(argCount)
		args = append(args, action)
	}
	if entityType != "" {
		argCount++
		query += " AND entity_type = $" + strconv.Itoa(argCount)
		args = append(args, entityType)
	}
	if playerIDStr != "" {
		pid, err := uuid.Parse(playerIDStr)
		if err == nil {
			argCount++
			query += " AND player_id = $" + strconv.Itoa(argCount)
			args = append(args, pid)
		}
	}
	if successStr != "" {
		success, err := strconv.ParseBool(successStr)
		if err == nil {
			argCount++
			query += " AND success = $" + strconv.Itoa(argCount)
			args = append(args, success)
		}
	}
	if search != "" {
		argCount++
		query += " AND (action ILIKE $" + strconv.Itoa(argCount) + " OR request_summary ILIKE $" + strconv.Itoa(argCount) + " OR entity_id ILIKE $" + strconv.Itoa(argCount) + " OR error_message ILIKE $" + strconv.Itoa(argCount) + ")"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY " + paging.Sort + " " + paging.Direction + ", id DESC LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, paging.PageSize, paging.Offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể truy vấn nhật ký: " + err.Error()})
		return
	}
	defer rows.Close()

	logs := []AuditLogResponse{}
	for rows.Next() {
		var logRow AuditLogResponse
		var metaJSON []byte
		err := rows.Scan(
			&logRow.ID, &logRow.PlayerID, &logRow.Action, &logRow.EntityType,
			&logRow.EntityID, &logRow.RequestSummary, &metaJSON, &logRow.IPAddress,
			&logRow.Success, &logRow.ErrorMessage, &logRow.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc nhật ký: " + err.Error()})
			return
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &logRow.Metadata)
		}
		logs = append(logs, logRow)
	}

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	countArgs := make([]interface{}, 0)
	ci := 0
	if action != "" {
		ci++
		countQuery += " AND action = $" + strconv.Itoa(ci)
		countArgs = append(countArgs, action)
	}
	if entityType != "" {
		ci++
		countQuery += " AND entity_type = $" + strconv.Itoa(ci)
		countArgs = append(countArgs, entityType)
	}
	if playerIDStr != "" {
		pid, err := uuid.Parse(playerIDStr)
		if err == nil {
			ci++
			countQuery += " AND player_id = $" + strconv.Itoa(ci)
			countArgs = append(countArgs, pid)
		}
	}
	if successStr != "" {
		success, err := strconv.ParseBool(successStr)
		if err == nil {
			ci++
			countQuery += " AND success = $" + strconv.Itoa(ci)
			countArgs = append(countArgs, success)
		}
	}
	if search != "" {
		ci++
		countQuery += " AND (action ILIKE $" + strconv.Itoa(ci) + " OR request_summary ILIKE $" + strconv.Itoa(ci) + " OR entity_id ILIKE $" + strconv.Itoa(ci) + " OR error_message ILIKE $" + strconv.Itoa(ci) + ")"
		countArgs = append(countArgs, "%"+search+"%")
	}

	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, countArgs...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm nhật ký"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      paging.Page,
		"page_size": paging.PageSize,
	})
}
