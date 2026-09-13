package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminMapCellResponse struct {
	ID             string                 `json:"id"`
	RegionID       string                 `json:"region_id"`
	X              int                    `json:"x"`
	Y              int                    `json:"y"`
	CellType       string                 `json:"cell_type"`
	ContentID      string                 `json:"content_id"`
	Metadata       map[string]interface{} `json:"metadata"`
	RespawnSeconds int                    `json:"respawn_seconds"`
	IsActive       bool                   `json:"is_active"`
}

// AdminListRegions lists regions with search and pagination
func AdminListRegions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := `SELECT id, name, COALESCE(description,''), width, height, min_rank, max_rank, is_active, 
	                 COALESCE(parent_id,''), zone_type, min_x, min_y, max_x, max_y, entry_x, entry_y,
	                 safe_zone_x, safe_zone_y,
	                 COALESCE(pool_weights::text, '{}') 
	          FROM regions`
	countQuery := `SELECT COUNT(*) FROM regions`

	var args []interface{}
	var countArgs []interface{}
	whereClauses := []string{}
	argIdx := 1

	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(id ILIKE $%d OR name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		countArgs = append(countArgs, "%"+search+"%")
		argIdx++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
		countQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, countArgs...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm số lượng khu vực"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách khu vực"})
		return
	}
	defer rows.Close()

	var list []models.Region
	for rows.Next() {
		var r models.Region
		var poolRaw string
		var parentID sql.NullString
		var safeZoneX sql.NullInt64
		var safeZoneY sql.NullInt64
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Width, &r.Height, &r.MinRank, &r.MaxRank, &r.IsActive,
			&parentID, &r.ZoneType, &r.MinX, &r.MinY, &r.MaxX, &r.MaxY, &r.EntryX, &r.EntryY, &safeZoneX, &safeZoneY, &poolRaw); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan error"})
			return
		}
		if parentID.Valid {
			r.ParentID = parentID.String
		}
		if poolRaw != "" {
			_ = json.Unmarshal([]byte(poolRaw), &r.PoolWeights)
		}
		if safeZoneX.Valid {
			val := int(safeZoneX.Int64)
			r.SafeZoneX = &val
		}
		if safeZoneY.Valid {
			val := int(safeZoneY.Int64)
			r.SafeZoneY = &val
		}
		list = append(list, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"last_page": (total + pageSize - 1) / pageSize,
	})
}

// AdminCreateRegion creates a new region
func AdminCreateRegion(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req models.Region
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID và Tên không được để trống"})
		return
	}

	poolWeightsBytes, err := json.Marshal(req.PoolWeights)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi dữ liệu pool_weights"})
		return
	}

	var parentID interface{}
	if req.ParentID != "" {
		parentID = req.ParentID
	} else {
		parentID = nil
	}

	query := `INSERT INTO regions (id, name, description, width, height, min_rank, max_rank, is_active, 
	                              parent_id, zone_type, min_x, min_y, max_x, max_y, entry_x, entry_y, safe_zone_x, safe_zone_y, pool_weights)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	_, err = db.DB.ExecContext(c.Request.Context(), query,
		req.ID, req.Name, req.Description, req.Width, req.Height, req.MinRank, req.MaxRank, req.IsActive,
		parentID, req.ZoneType, req.MinX, req.MinY, req.MaxX, req.MaxY, req.EntryX, req.EntryY, req.SafeZoneX, req.SafeZoneY, poolWeightsBytes)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo khu vực: " + err.Error()})
		return
	}

	// Reload the registry
	_ = LoadZoneRegistry(db.DB)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.create_region",
		EntityType: "region", EntityID: req.ID,
		RequestSummary: fmt.Sprintf("name=%s, type=%s", req.Name, req.ZoneType),
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Tạo khu vực thành công", "data": req})
}

// AdminUpdateRegion updates an existing region
func AdminUpdateRegion(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID khu vực không hợp lệ"})
		return
	}

	var req models.Region
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	poolWeightsBytes, err := json.Marshal(req.PoolWeights)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi dữ liệu pool_weights"})
		return
	}

	var parentID interface{}
	if req.ParentID != "" {
		parentID = req.ParentID
	} else {
		parentID = nil
	}

	query := `UPDATE regions 
          SET name = $1, description = $2, width = $3, height = $4, min_rank = $5, max_rank = $6, 
              is_active = $7, parent_id = $8, zone_type = $9, min_x = $10, min_y = $11, 
              max_x = $12, max_y = $13, entry_x = $14, entry_y = $15, safe_zone_x = $16, safe_zone_y = $17, pool_weights = $18, updated_at = NOW()
          WHERE id = $19`

	res, err := db.DB.ExecContext(c.Request.Context(), query,
		req.Name, req.Description, req.Width, req.Height, req.MinRank, req.MaxRank, req.IsActive,
		parentID, req.ZoneType, req.MinX, req.MinY, req.MaxX, req.MaxY, req.EntryX, req.EntryY, req.SafeZoneX, req.SafeZoneY, poolWeightsBytes, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật khu vực: " + err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy khu vực"})
		return
	}

	// Reload the registry
	_ = LoadZoneRegistry(db.DB)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.update_region",
		EntityType: "region", EntityID: id,
		RequestSummary: fmt.Sprintf("name=%s, type=%s", req.Name, req.ZoneType),
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật khu vực thành công"})
}

// AdminDeleteRegion deletes a region
func AdminDeleteRegion(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID khu vực không hợp lệ"})
		return
	}

	query := `DELETE FROM regions WHERE id = $1`
	res, err := db.DB.ExecContext(c.Request.Context(), query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa khu vực: " + err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy khu vực"})
		return
	}

	// Reload the registry
	_ = LoadZoneRegistry(db.DB)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.delete_region",
		EntityType: "region", EntityID: id,
		RequestSummary: fmt.Sprintf("id=%s", id),
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Xóa khu vực thành công"})
}

// AdminListMapCells lists map cells
func AdminListMapCells(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	regionID := c.Query("region_id")
	cellType := c.Query("cell_type")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := `SELECT id, region_id, x, y, cell_type, COALESCE(content_id,''), COALESCE(metadata::text, '{}'), 
	                 respawn_seconds, is_active 
	          FROM map_cells`
	countQuery := `SELECT COUNT(*) FROM map_cells`

	var args []interface{}
	var countArgs []interface{}
	whereClauses := []string{}
	argIdx := 1

	if regionID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("region_id = $%d", argIdx))
		args = append(args, regionID)
		countArgs = append(countArgs, regionID)
		argIdx++
	}

	if cellType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("cell_type = $%d", argIdx))
		args = append(args, cellType)
		countArgs = append(countArgs, cellType)
		argIdx++
	}

	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(content_id ILIKE $%d OR x::text = $%d OR y::text = $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		countArgs = append(countArgs, "%"+search+"%")
		argIdx++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
		countQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY region_id, x, y LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, countArgs...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm số lượng ô bản đồ"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách ô bản đồ"})
		return
	}
	defer rows.Close()

	var list []AdminMapCellResponse
	for rows.Next() {
		var cell AdminMapCellResponse
		var metaRaw string
		if err := rows.Scan(&cell.ID, &cell.RegionID, &cell.X, &cell.Y, &cell.CellType, &cell.ContentID,
			&metaRaw, &cell.RespawnSeconds, &cell.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan error"})
			return
		}
		if metaRaw != "" {
			_ = json.Unmarshal([]byte(metaRaw), &cell.Metadata)
		}
		list = append(list, cell)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"last_page": (total + pageSize - 1) / pageSize,
	})
}

// AdminCreateMapCell creates a map cell
func AdminCreateMapCell(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req AdminMapCellResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RegionID == "" || req.CellType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Region ID và Cell Type không được để trống"})
		return
	}

	metaBytes, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi dữ liệu metadata"})
		return
	}

	query := `INSERT INTO map_cells (region_id, x, y, cell_type, content_id, metadata, respawn_seconds, is_active)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id`

	var createdID string
	err = db.DB.QueryRowContext(c.Request.Context(), query,
		req.RegionID, req.X, req.Y, req.CellType, req.ContentID, metaBytes, req.RespawnSeconds, req.IsActive).Scan(&createdID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo ô bản đồ: " + err.Error()})
		return
	}

	req.ID = createdID

	// Reload the registry
	_ = LoadZoneRegistry(db.DB)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.create_map_cell",
		EntityType: "map_cell", EntityID: createdID,
		RequestSummary: fmt.Sprintf("region=%s, coords=(%d,%d), type=%s", req.RegionID, req.X, req.Y, req.CellType),
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Tạo ô bản đồ thành công", "data": req})
}

// AdminUpdateMapCell updates a map cell
func AdminUpdateMapCell(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID ô bản đồ không hợp lệ"})
		return
	}

	var req AdminMapCellResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metaBytes, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi dữ liệu metadata"})
		return
	}

	query := `UPDATE map_cells 
	          SET region_id = $1, x = $2, y = $3, cell_type = $4, content_id = $5, 
	              metadata = $6, respawn_seconds = $7, is_active = $8
	          WHERE id = $9`

	res, err := db.DB.ExecContext(c.Request.Context(), query,
		req.RegionID, req.X, req.Y, req.CellType, req.ContentID, metaBytes, req.RespawnSeconds, req.IsActive, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật ô bản đồ: " + err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy ô bản đồ"})
		return
	}

	// Reload the registry
	_ = LoadZoneRegistry(db.DB)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.update_map_cell",
		EntityType: "map_cell", EntityID: id,
		RequestSummary: fmt.Sprintf("region=%s, coords=(%d,%d), type=%s", req.RegionID, req.X, req.Y, req.CellType),
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật ô bản đồ thành công"})
}

// AdminDeleteMapCell deletes a map cell
func AdminDeleteMapCell(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID ô bản đồ không hợp lệ"})
		return
	}

	query := `DELETE FROM map_cells WHERE id = $1`
	res, err := db.DB.ExecContext(c.Request.Context(), query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa ô bản đồ: " + err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy ô bản đồ"})
		return
	}

	// Reload the registry
	_ = LoadZoneRegistry(db.DB)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.delete_map_cell",
		EntityType: "map_cell", EntityID: id,
		RequestSummary: fmt.Sprintf("id=%s", id),
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Xóa ô bản đồ thành công"})
}
