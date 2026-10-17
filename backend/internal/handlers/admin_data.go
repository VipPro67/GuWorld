package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
)

func writeImportError(c *gin.Context, prefix string, err error) {
	var validation *effectcatalog.ValidationError
	if errors.As(err, &validation) {
		c.JSON(http.StatusBadRequest, gin.H{"error": prefix, "issues": validation.Issues})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": prefix + ": " + err.Error()})
}

const importPreflightTTL = 15 * time.Minute
const maxImportUploadBytes = 10 << 20

type stagedImport struct {
	Owner     uuid.UUID
	Table     string
	Sources   map[string][]byte
	Digest    string
	ExpiresAt time.Time
}

var stagedImports = struct {
	sync.Mutex
	items map[string]stagedImport
}{items: map[string]stagedImport{}}

func readImportUpload(c *gin.Context) ([]byte, error) {
	file, err := c.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		return nil, err
	}
	if file.Size <= 0 {
		return nil, errors.New("file CSV rỗng")
	}
	if file.Size > maxImportUploadBytes {
		return nil, fmt.Errorf("file CSV vượt giới hạn %d MB", maxImportUploadBytes>>20)
	}
	opened, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	return io.ReadAll(io.LimitReader(opened, maxImportUploadBytes+1))
}

func preflightImport(c *gin.Context, table string) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	upload, err := readImportUpload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sources, err := db.ReadImportSources(table, upload)
	if err != nil {
		writeImportError(c, "Không thể đọc nguồn import", err)
		return
	}
	plan, _, err := db.RunImportPlanSources(c.Request.Context(), table, sources, true)
	if err != nil {
		writeImportError(c, "Preflight thất bại", err)
		return
	}
	digest := db.ImportSourcesDigest(sources)
	token := uuid.NewString()
	expiresAt := time.Now().Add(importPreflightTTL)
	stagedImports.Lock()
	for key, item := range stagedImports.items {
		if time.Now().After(item.ExpiresAt) {
			delete(stagedImports.items, key)
		}
	}
	stagedImports.items[token] = stagedImport{Owner: adminID, Table: table, Sources: sources, Digest: digest, ExpiresAt: expiresAt}
	stagedImports.Unlock()
	c.JSON(http.StatusOK, gin.H{"preflight_id": token, "expires_at": expiresAt, "source_digest": digest, "plan": plan})
}

// AdminImportData performs a no-write dry-run for every server-side dataset.
func AdminImportData(c *gin.Context) { preflightImport(c, "") }

// AdminImportTable performs a no-write dry-run for one server CSV or upload.
func AdminImportTable(c *gin.Context) { preflightImport(c, c.Param("table")) }

func reloadImportedData() []string {
	warnings := []string{}
	reload := func(label string, fn func() error) {
		if err := fn(); err != nil {
			warnings = append(warnings, label+": "+err.Error())
		}
	}
	reload("Gu registry", func() error { return models.LoadGuRegistry(db.DB) })
	reload("Item registry", func() error { return models.LoadItemRegistry(db.DB) })
	reload("Constitution registry", func() error { return gamefx.LoadConstitutionRegistry(db.DB) })
	reload("Race registry", func() error { return gamefx.LoadRaceRegistry(db.DB) })
	reload("Status registry", func() error { return gamefx.LoadStatusRegistry(db.DB) })
	reload("Monster registry", func() error { return models.LoadMonsterRegistry(db.DB) })
	reload("Quest registry", LoadQuestRegistry)
	reload("Zone registry", func() error { return LoadZoneRegistry(db.DB) })
	reload("Instance cache", InitInstanceCache)
	InvalidateDictionaryMemoryCache()
	invalidateRecipeCatalog()
	invalidateCachedDictionaryPart("explore_regions_v2")
	cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_statuses", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns", "game_dictionary_region_loot", "game_quests_list", "explore_regions")
	return warnings
}

func commitImport(c *gin.Context, table string) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var request struct {
		PreflightID string `json:"preflight_id"`
		Reason      string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.PreflightID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preflight_id là bắt buộc"})
		return
	}
	stagedImports.Lock()
	stage, exists := stagedImports.items[request.PreflightID]
	if exists && stage.Owner == adminID && stage.Table == table && time.Now().Before(stage.ExpiresAt) {
		delete(stagedImports.items, request.PreflightID)
	} else {
		exists = false
	}
	stagedImports.Unlock()
	if !exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Preflight không tồn tại, đã hết hạn, đã dùng hoặc không thuộc admin này"})
		return
	}
	digest := db.ImportSourcesDigest(stage.Sources)
	if digest != stage.Digest {
		c.JSON(http.StatusConflict, gin.H{"error": "Nguồn CSV đã thay đổi; hãy chạy preflight lại"})
		return
	}
	plan, tx, err := db.RunImportPlanSources(c.Request.Context(), table, stage.Sources, false)
	if err != nil {
		writeImportError(c, "Commit import thất bại", err)
		return
	}
	if !plan.CanCommit || tx == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Preflight commit phát hiện lỗi nghiêm trọng", "plan": plan})
		return
	}
	action := "admin.import_table"
	entityID := table
	if table == "" {
		action = "admin.import_data"
		entityID = "all"
	}
	if err := auditlog.LogChecked(tx, auditlog.Entry{PlayerID: &adminID, Action: action, EntityType: "game_data", EntityID: entityID, RequestSummary: strings.TrimSpace(request.Reason), IPAddress: c.ClientIP(), Success: true, Metadata: map[string]interface{}{"preflight_id": request.PreflightID, "source_digest": stage.Digest, "totals": plan.Totals}}); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi audit; transaction đã rollback"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể commit transaction: " + err.Error()})
		return
	}
	var warnings []string
	if table == "" {
		warnings = reloadImportedData()
	} else if dataset, ok := db.DatasetByKey(table); ok {
		// Single-dataset commits only touch the declaring registries/caches;
		// reloading every registry (monster/quest/zone/instance/…) is wasteful.
		warnings = applyDatasetRuntimeBehavior([]db.Dataset{dataset})
	} else {
		warnings = []string{}
	}
	if len(warnings) > 0 {
		monitor.GlobalAdminOperations.RecordImportFailure(action)
		logger.L.Error("admin_import_failed", zap.String("action", action), zap.String("stage", "runtime_reload"), zap.String("table", entityID), zap.Strings("warnings", warnings), zap.String("player_id", adminID.String()))
	}
	c.JSON(http.StatusOK, gin.H{"message": "Import đã commit", "committed": true, "plan": plan, "reload_ok": len(warnings) == 0, "reload_warnings": warnings, "restart_required": len(warnings) > 0})
}

func AdminCommitImportData(c *gin.Context)  { commitImport(c, "") }
func AdminCommitImportTable(c *gin.Context) { commitImport(c, c.Param("table")) }

var dictionaryCacheKeys = []string{
	"game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items",
	"game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races",
	"game_dictionary_statuses", "game_dictionary_loot_tables", "game_dictionary_regions",
	"game_dictionary_monster_spawns", "game_dictionary_region_loot",
}

func applyDatasetRuntimeBehavior(datasets []db.Dataset) []string {
	caches, reloads := map[string]bool{}, map[string]bool{}
	for _, dataset := range datasets {
		for _, behavior := range dataset.CacheInvalidation {
			caches[behavior] = true
		}
		for _, registry := range dataset.RegistryReload {
			reloads[registry] = true
		}
	}
	warnings := []string{}
	reload := func(name string, fn func() error) {
		if reloads[name] {
			if err := fn(); err != nil {
				logger.S.Warnf("failed to reload %s after import: %v", name, err)
				warnings = append(warnings, name+": "+err.Error())
			}
		}
	}
	reload("gu", func() error { return models.LoadGuRegistry(db.DB) })
	reload("item", func() error { return models.LoadItemRegistry(db.DB) })
	reload("constitution", func() error { return gamefx.LoadConstitutionRegistry(db.DB) })
	reload("race", func() error { return gamefx.LoadRaceRegistry(db.DB) })
	reload("status", func() error { return gamefx.LoadStatusRegistry(db.DB) })
	reload("monster", func() error { return models.LoadMonsterRegistry(db.DB) })
	reload("quest", LoadQuestRegistry)
	reload("zone", func() error { return LoadZoneRegistry(db.DB) })
	reload("instance", InitInstanceCache)

	if caches["dictionary"] {
		cache.Delete(dictionaryCacheKeys...)
		InvalidateDictionaryMemoryCache()
	}
	if caches["recipe_catalog"] {
		invalidateRecipeCatalog()
	}
	if caches["shop_catalog"] {
		invalidateShopCatalogCache()
	}
	if caches["arena_shop"] {
		cache.Delete("arena_shop_items")
	}
	if caches["quests"] {
		cache.Delete("game_quests_list")
	}
	if caches["world_monsters"] {
		if err := cache.PurgeWorldMonstersForRegions(cache.Ctx,
			"dai_vuc_dong_hai", "hub_dong_hai", "tan_thu_coc_dong_hai", "ngoc_ho_dao",
			"shadow_glimpse_trench", "binh_kinh_hai", "long_ngu_dao", "thao_thiet_hai_khau",
			"turbulent_current_sea", "khi_hai_vuc", "van_sam_hai_vuc", "bach_sa_quan",
			"long_mon_hai_vuc", "van_luu_tinh_hai"); err != nil {
			logger.S.Warnf("failed to purge world-monster cache after region import: %v", err)
		}
	}
	// The old full reload invalidated these on every commit; region/explore
	// caches are cheap to rebuild, so keep that guarantee when narrowing scope.
	invalidateCachedDictionaryPart("explore_regions_v2")
	cache.Delete("explore_regions")
	return warnings
}

// AdminImportData seeds all dictionary tables from CSV files.
// POST /api/admin/data/import
func adminImportDataLegacy(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	if err := db.SeedFromCSV(); err != nil {
		writeImportError(c, "Import thất bại", err)
		return
	}

	applyDatasetRuntimeBehavior(db.Datasets())

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
	c.JSON(http.StatusOK, gin.H{"tables": db.AvailableTables(), "datasets": db.DatasetInfos()})
}

// AdminImportTable imports a single table from its CSV file.
// POST /api/admin/data/import/:table
func adminImportTableLegacy(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	table := c.Param("table")

	if err := db.TableImport(table); err != nil {
		writeImportError(c, "Import bảng "+table+" thất bại", err)
		return
	}

	dataset, _ := db.DatasetByKey(table)
	applyDatasetRuntimeBehavior([]db.Dataset{dataset})

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
		"game_dictionary_region_loot",
		"game_quests_list",
		"explore_regions",
		"explore_regions_v2",
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

	query := `SELECT id, COALESCE(player_id::text, ''), actor_username, action, effective_roles, effective_permission, entity_type, entity_id, change_data, reason, request_id, COALESCE(request_summary, ''), metadata, ip_address, success, error_message, created_at FROM audit_logs WHERE 1=1`
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
		var metaJSON, rolesJSON, changeJSON []byte
		err := rows.Scan(
			&logRow.ID, &logRow.PlayerID, &logRow.ActorUsername, &logRow.Action, &rolesJSON, &logRow.Permission, &logRow.EntityType,
			&logRow.EntityID, &changeJSON, &logRow.Reason, &logRow.RequestID, &logRow.RequestSummary, &metaJSON, &logRow.IPAddress,
			&logRow.Success, &logRow.ErrorMessage, &logRow.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc nhật ký: " + err.Error()})
			return
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &logRow.Metadata)
		}
		_ = json.Unmarshal(rolesJSON, &logRow.EffectiveRoles)
		_ = json.Unmarshal(changeJSON, &logRow.ChangeData)
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
	if len(countArgs) == 0 {
		var err error
		total, err = fastCountTable(c, "audit_logs", "admin:audit_logs:total_count", 60*time.Second)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm nhật ký"})
			return
		}
	} else {
		if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, countArgs...).Scan(&total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm nhật ký"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      paging.Page,
		"page_size": paging.PageSize,
	})
}
