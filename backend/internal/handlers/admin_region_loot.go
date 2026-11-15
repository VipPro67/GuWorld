package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminRegionLootRequest struct {
	RegionID    string  `json:"region_id" binding:"required"`
	LootTableID string  `json:"loot_table_id" binding:"required"`
	MonsterType string  `json:"monster_type" binding:"required"`
	MinRank     int     `json:"min_rank"`
	MaxRank     int     `json:"max_rank"`
	Chance      float64 `json:"chance"`
	IsActive    *bool   `json:"is_active"`
}

// AdminListRegionLootTables lists region loot mappings with filters and pagination.
func AdminListRegionLootTables(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	search := strings.TrimSpace(c.Query("search"))
	regionFilter := strings.TrimSpace(c.Query("region_id"))
	monsterTypeFilter := strings.TrimSpace(c.Query("monster_type"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	baseQuery := `
		FROM region_loot_tables rlt
		LEFT JOIN regions r ON r.id = rlt.region_id
		LEFT JOIN loot_tables lt ON lt.id = rlt.loot_table_id
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (rlt.region_id ILIKE $%d OR rlt.loot_table_id ILIKE $%d OR r.name ILIKE $%d OR lt.name ILIKE $%d OR rlt.monster_type ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if regionFilter != "" {
		baseQuery += fmt.Sprintf(" AND rlt.region_id = $%d", argIdx)
		args = append(args, regionFilter)
		argIdx++
	}
	if monsterTypeFilter != "" {
		baseQuery += fmt.Sprintf(" AND rlt.monster_type = $%d", argIdx)
		args = append(args, monsterTypeFilter)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đếm dữ liệu loot khu vực", err))
		return
	}

	selectQuery := fmt.Sprintf(`
		SELECT rlt.region_id, COALESCE(r.name, rlt.region_id), rlt.loot_table_id, COALESCE(lt.name, rlt.loot_table_id), rlt.monster_type, rlt.min_rank, rlt.max_rank, rlt.chance, rlt.is_active, rlt.created_at, rlt.updated_at
		%s
		ORDER BY rlt.region_id ASC, rlt.monster_type ASC, rlt.loot_table_id ASC
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), selectQuery, args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc danh sách loot khu vực", err))
		return
	}
	defer rows.Close()

	list := make([]gin.H, 0)
	for rows.Next() {
		var regionID, regionName, lootTableID, lootTableName, monsterType string
		var minRank, maxRank int
		var chance float64
		var isActive bool
		var createdAt, updatedAt interface{}

		if err := rows.Scan(&regionID, &regionName, &lootTableID, &lootTableName, &monsterType, &minRank, &maxRank, &chance, &isActive, &createdAt, &updatedAt); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu loot khu vực", err))
			return
		}

		list = append(list, gin.H{
			"region_id":       regionID,
			"region_name":     regionName,
			"loot_table_id":   lootTableID,
			"loot_table_name": lootTableName,
			"monster_type":    monsterType,
			"min_rank":        minRank,
			"max_rank":        maxRank,
			"chance":          chance,
			"is_active":       isActive,
			"created_at":      createdAt,
			"updated_at":      updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminCreateRegionLootTable creates a new region loot table mapping.
func AdminCreateRegionLootTable(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req AdminRegionLootRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	if req.MinRank <= 0 {
		req.MinRank = 1
	}
	if req.MaxRank <= 0 {
		req.MaxRank = 9
	}
	if req.Chance <= 0 {
		req.Chance = 1.0
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	query := `
		INSERT INTO region_loot_tables (region_id, loot_table_id, monster_type, min_rank, max_rank, chance, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (region_id, loot_table_id, monster_type) DO UPDATE
		SET min_rank = EXCLUDED.min_rank, max_rank = EXCLUDED.max_rank, chance = EXCLUDED.chance, is_active = EXCLUDED.is_active, updated_at = NOW()
	`

	if _, err := db.DB.ExecContext(c.Request.Context(), query, req.RegionID, req.LootTableID, req.MonsterType, req.MinRank, req.MaxRank, req.Chance, isActive); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể lưu cấu hình loot khu vực: "+err.Error(), err))
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "region_loot_tables",
		EntityID:       fmt.Sprintf("%s:%s:%s", req.RegionID, req.LootTableID, req.MonsterType),
		Action:         "admin.create_region_loot",
		RequestSummary: fmt.Sprintf("region_id=%s loot_table_id=%s monster_type=%s chance=%.2f", req.RegionID, req.LootTableID, req.MonsterType, req.Chance),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã lưu cấu hình loot khu vực thành công"})
}

// AdminUpdateRegionLootTable updates an existing mapping.
func AdminUpdateRegionLootTable(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	regionID := c.Param("region_id")
	lootTableID := c.Param("loot_table_id")
	monsterType := c.Param("monster_type")

	var req AdminRegionLootRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	if req.MinRank <= 0 {
		req.MinRank = 1
	}
	if req.MaxRank <= 0 {
		req.MaxRank = 9
	}
	if req.Chance <= 0 {
		req.Chance = 1.0
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	res, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE region_loot_tables
		SET min_rank = $1, max_rank = $2, chance = $3, is_active = $4, updated_at = NOW()
		WHERE region_id = $5 AND loot_table_id = $6 AND monster_type = $7
	`, req.MinRank, req.MaxRank, req.Chance, isActive, regionID, lootTableID, monsterType)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể cập nhật cấu hình loot", err))
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cấu hình loot khu vực"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "region_loot_tables",
		EntityID:       fmt.Sprintf("%s:%s:%s", regionID, lootTableID, monsterType),
		Action:         "admin.update_region_loot",
		RequestSummary: fmt.Sprintf("min_rank=%d max_rank=%d chance=%.2f is_active=%v", req.MinRank, req.MaxRank, req.Chance, isActive),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật cấu hình loot khu vực thành công"})
}

// AdminDeleteRegionLootTable removes a mapping.
func AdminDeleteRegionLootTable(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	regionID := c.Param("region_id")
	lootTableID := c.Param("loot_table_id")
	monsterType := c.Param("monster_type")

	res, err := db.DB.ExecContext(c.Request.Context(), `
		DELETE FROM region_loot_tables
		WHERE region_id = $1 AND loot_table_id = $2 AND monster_type = $3
	`, regionID, lootTableID, monsterType)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể xóa cấu hình loot", err))
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cấu hình loot khu vực"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "region_loot_tables",
		EntityID:       fmt.Sprintf("%s:%s:%s", regionID, lootTableID, monsterType),
		Action:         "admin.delete_region_loot",
		RequestSummary: fmt.Sprintf("region_id=%s loot_table_id=%s monster_type=%s", regionID, lootTableID, monsterType),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cấu hình loot khu vực thành công"})
}
