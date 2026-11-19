package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ============================================================================
// 1. Primeval Vein ("Nguyên Tuyền") Admin Types and Handlers
// ============================================================================

type AdminPrimevalVeinResponse struct {
	ID                string    `json:"id"`
	RegionID          string    `json:"region_id"`
	RegionName        string    `json:"region_name,omitempty"`
	X                 int       `json:"x"`
	Y                 int       `json:"y"`
	VeinType          string    `json:"vein_type"`
	Rarity            string    `json:"rarity"`
	AccumulatedSince  time.Time `json:"accumulated_since"`
	ProducedMinutes   int       `json:"produced_minutes"`
	ExpiresAt         time.Time `json:"expires_at"`
	CreatedAt         time.Time `json:"created_at"`
	AccumulatedStones int64     `json:"accumulated_stones"`
	AccumulatedQi     float64   `json:"accumulated_qi"`
	AccumulatedChests int       `json:"accumulated_chests"`
	CapMinutes        int       `json:"cap_minutes"`
	RemainingMinutes  int       `json:"remaining_minutes"`
	IsDepleted        bool      `json:"is_depleted"`
}

type AdminCreatePrimevalVeinRequest struct {
	RegionID     string `json:"region_id" binding:"required"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	VeinType     string `json:"vein_type"` // "mortal" or "immortal"
	Rarity       string `json:"rarity"`    // "Thường", "Hiếm", "Sử Thi", "Truyền Thuyết", "Siêu Cấp"
	LifetimeDays int    `json:"lifetime_days"`
}

type AdminUpdatePrimevalVeinRequest struct {
	VeinType          *string    `json:"vein_type"`
	Rarity            *string    `json:"rarity"`
	X                 *int       `json:"x"`
	Y                 *int       `json:"y"`
	ProducedMinutes   *int       `json:"produced_minutes"`
	ExpiresAt         *time.Time `json:"expires_at"`
	ResetAccumulation bool       `json:"reset_accumulation"`
}

// AdminListPrimevalVeins lists all primeval veins on the map with active accrual preview.
func AdminListPrimevalVeins(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	regionID := strings.TrimSpace(c.Query("region_id"))
	veinType := strings.TrimSpace(c.Query("vein_type"))
	rarity := strings.TrimSpace(c.Query("rarity"))
	search := strings.TrimSpace(c.Query("search"))

	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if regionID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("v.region_id = $%d", argIdx))
		args = append(args, regionID)
		argIdx++
	}
	if veinType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("v.vein_type = $%d", argIdx))
		args = append(args, veinType)
		argIdx++
	}
	if rarity != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("v.rarity = $%d", argIdx))
		args = append(args, rarity)
		argIdx++
	}
	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(v.region_id ILIKE $%d OR COALESCE(r.name, '') ILIKE $%d OR v.id::text ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT v.id::text, v.region_id, COALESCE(r.name, v.region_id), v.x, v.y,
		       v.vein_type, v.rarity, v.accumulated_since, v.produced_minutes, v.expires_at, v.created_at
		FROM primeval_veins v
		LEFT JOIN regions r ON r.id = v.region_id
		WHERE %s
		ORDER BY v.created_at DESC
	`, strings.Join(whereClauses, " AND "))

	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi tải danh sách Nguyên Tuyền", err))
		return
	}
	defer rows.Close()

	now := time.Now()
	list := make([]AdminPrimevalVeinResponse, 0)
	mortalCount := 0
	immortalCount := 0

	for rows.Next() {
		var item AdminPrimevalVeinResponse
		if errScan := rows.Scan(
			&item.ID, &item.RegionID, &item.RegionName, &item.X, &item.Y,
			&item.VeinType, &item.Rarity, &item.AccumulatedSince, &item.ProducedMinutes,
			&item.ExpiresAt, &item.CreatedAt,
		); errScan != nil {
			logger.S.Warnf("AdminListPrimevalVeins scan error: %v", errScan)
			continue
		}

		if item.VeinType == "immortal" {
			immortalCount++
		} else {
			mortalCount++
		}

		// Calculate live accrual preview
		vRow := veinRow{
			ID:               item.ID,
			RegionID:         item.RegionID,
			X:                item.X,
			Y:                item.Y,
			VeinType:         item.VeinType,
			Rarity:           item.Rarity,
			AccumulatedSince: item.AccumulatedSince,
			ProducedMinutes:  item.ProducedMinutes,
			ExpiresAt:        item.ExpiresAt,
		}
		_, stones, qi, chests := veinAccrual(vRow, now)
		item.AccumulatedStones = stones
		item.AccumulatedQi = qi
		item.AccumulatedChests = chests
		item.CapMinutes = veinCapMinutes(item.Rarity)
		rem := item.CapMinutes - item.ProducedMinutes
		if rem < 0 {
			rem = 0
		}
		item.RemainingMinutes = rem
		item.IsDepleted = item.ProducedMinutes >= item.CapMinutes || !item.ExpiresAt.After(now)

		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":           list,
		"total":          len(list),
		"mortal_count":   mortalCount,
		"immortal_count": immortalCount,
		"mortal_cap":     veinMortalCap,
		"immortal_cap":   veinImmortalCap,
	})
}

// AdminCreatePrimevalVein creates a primeval vein at specified coordinates.
func AdminCreatePrimevalVein(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req AdminCreatePrimevalVeinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	req.RegionID = strings.TrimSpace(req.RegionID)
	req.VeinType = strings.ToLower(strings.TrimSpace(req.VeinType))
	if req.VeinType == "" {
		req.VeinType = "mortal"
	}
	if req.VeinType != "mortal" && req.VeinType != "immortal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại Nguyên Tuyền phải là mortal hoặc immortal"})
		return
	}

	req.Rarity = strings.TrimSpace(req.Rarity)
	if req.Rarity == "" {
		req.Rarity = "Thường"
	}
	if _, ok := veinRarityRates[req.Rarity]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phẩm chất không hợp lệ"})
		return
	}

	days := req.LifetimeDays
	if days <= 0 {
		days = 30
	}
	expiresAt := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	// Verify region
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM regions WHERE id = $1)`, req.RegionID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khu vực không tồn tại"})
		return
	}

	// Check cell occupancy in primeval_veins
	var occupied bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM primeval_veins WHERE region_id = $1 AND x = $2 AND y = $3)`, req.RegionID, req.X, req.Y).Scan(&occupied); err != nil || occupied {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ này đã có Nguyên Tuyền"})
		return
	}

	var newID string
	err := db.DB.QueryRowContext(c.Request.Context(), `
		INSERT INTO primeval_veins (region_id, x, y, vein_type, rarity, accumulated_since, produced_minutes, expires_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), 0, $6)
		RETURNING id::text
	`, req.RegionID, req.X, req.Y, req.VeinType, req.Rarity, expiresAt).Scan(&newID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tạo Nguyên Tuyền", err))
		return
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.create_primeval_vein",
		EntityType:     "primeval_vein",
		EntityID:       newID,
		RequestSummary: fmt.Sprintf("Tạo Nguyên Tuyền %s (%s) tại %s (%d, %d)", req.Rarity, req.VeinType, req.RegionID, req.X, req.Y),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tạo Nguyên Tuyền thành công",
		"id":      newID,
	})
}

// AdminUpdatePrimevalVein updates primeval vein properties.
func AdminUpdatePrimevalVein(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID Nguyên Tuyền không hợp lệ"})
		return
	}

	var req AdminUpdatePrimevalVeinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var current AdminPrimevalVeinResponse
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT id::text, region_id, x, y, vein_type, rarity, accumulated_since, produced_minutes, expires_at
		FROM primeval_veins WHERE id = $1 FOR UPDATE
	`, id).Scan(&current.ID, &current.RegionID, &current.X, &current.Y, &current.VeinType, &current.Rarity, &current.AccumulatedSince, &current.ProducedMinutes, &current.ExpiresAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nguyên Tuyền không tồn tại"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc Nguyên Tuyền", err))
		return
	}

	newType := current.VeinType
	if req.VeinType != nil {
		t := strings.ToLower(strings.TrimSpace(*req.VeinType))
		if t == "mortal" || t == "immortal" {
			newType = t
		}
	}

	newRarity := current.Rarity
	if req.Rarity != nil {
		r := strings.TrimSpace(*req.Rarity)
		if _, ok := veinRarityRates[r]; ok {
			newRarity = r
		}
	}

	newX := current.X
	newY := current.Y
	if req.X != nil {
		newX = *req.X
	}
	if req.Y != nil {
		newY = *req.Y
	}

	if newX != current.X || newY != current.Y {
		var occupied bool
		if errScan := tx.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM primeval_veins WHERE region_id = $1 AND x = $2 AND y = $3 AND id <> $4)`, current.RegionID, newX, newY, id).Scan(&occupied); errScan != nil || occupied {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ mới đã có Nguyên Tuyền"})
			return
		}
	}

	newProduced := current.ProducedMinutes
	if req.ProducedMinutes != nil && *req.ProducedMinutes >= 0 {
		newProduced = *req.ProducedMinutes
	}

	newExpires := current.ExpiresAt
	if req.ExpiresAt != nil && !req.ExpiresAt.IsZero() {
		newExpires = *req.ExpiresAt
	}

	newAccSince := current.AccumulatedSince
	if req.ResetAccumulation {
		newAccSince = time.Now()
	}

	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE primeval_veins
		SET vein_type = $1, rarity = $2, x = $3, y = $4, produced_minutes = $5, expires_at = $6, accumulated_since = $7
		WHERE id = $8
	`, newType, newRarity, newX, newY, newProduced, newExpires, newAccSince, id)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể cập nhật Nguyên Tuyền", err))
		return
	}

	auditlog.LogContext(c.Request.Context(), tx, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.update_primeval_vein",
		EntityType:     "primeval_vein",
		EntityID:       id,
		RequestSummary: fmt.Sprintf("Cập nhật Nguyên Tuyền %s: type=%s, rarity=%s, coord=(%d,%d)", id, newType, newRarity, newX, newY),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	if err = tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit error", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật Nguyên Tuyền thành công"})
}

// AdminDeletePrimevalVein removes a primeval vein from the map.
func AdminDeletePrimevalVein(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID Nguyên Tuyền không hợp lệ"})
		return
	}

	res, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM primeval_veins WHERE id = $1`, id)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi xóa Nguyên Tuyền", err))
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nguyên Tuyền không tồn tại"})
		return
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.delete_primeval_vein",
		EntityType:     "primeval_vein",
		EntityID:       id,
		RequestSummary: fmt.Sprintf("Xóa Nguyên Tuyền ID %s", id),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Xóa Nguyên Tuyền thành công"})
}

// AdminSpawnRandomPrimevalVein manually triggers random vein spawning via worker logic.
func AdminSpawnRandomPrimevalVein(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req struct {
		VeinType string `json:"vein_type"`
	}
	_ = c.ShouldBindJSON(&req)
	veinType := strings.ToLower(strings.TrimSpace(req.VeinType))
	if veinType != "immortal" {
		veinType = "mortal"
	}

	if err := spawnRandomVein(c.Request.Context(), veinType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể sinh Nguyên Tuyền: " + err.Error()})
		return
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.spawn_random_primeval_vein",
		EntityType:     "primeval_vein",
		EntityID:       veinType,
		RequestSummary: fmt.Sprintf("Sinh ngẫu nhiên 1 Nguyên Tuyền (%s)", veinType),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Đã sinh ngẫu nhiên 1 Nguyên Tuyền (%s) thành công", veinType)})
}

// ============================================================================
// 2. World Resource Points ("Mỏ Tài Nguyên Thế Giới") Admin Handlers
// ============================================================================

type AdminWorldResourcePointResponse struct {
	ID             string                 `json:"id"`
	Scope          string                 `json:"scope"`
	RegionID       string                 `json:"region_id"`
	RegionName     string                 `json:"region_name,omitempty"`
	X              int                    `json:"x"`
	Y              int                    `json:"y"`
	PointType      string                 `json:"point_type"`
	Name           string                 `json:"name"`
	Path           string                 `json:"path"`
	Grade          string                 `json:"grade"`
	PointClass     string                 `json:"point_class"`
	Level          int                    `json:"level"`
	Vitality       int                    `json:"vitality"`
	Stock          map[string]int64       `json:"stock"`
	StockCap       int                    `json:"stock_cap"`
	LastProducedAt time.Time              `json:"last_produced_at"`
	State          string                 `json:"state"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Template       *ResourcePointTemplate `json:"template,omitempty"`
}

type AdminCreateWorldResourcePointRequest struct {
	RegionID   string `json:"region_id" binding:"required"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	PointType  string `json:"point_type" binding:"required"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Grade      string `json:"grade"`       // "LOW", "MID", "TOP"
	PointClass string `json:"point_class"` // "NORMAL", "FAMOUS"
	Level      int    `json:"level"`
	Vitality   int    `json:"vitality"`
	State      string `json:"state"`
}

type AdminUpdateWorldResourcePointRequest struct {
	Name       *string           `json:"name"`
	Path       *string           `json:"path"`
	Grade      *string           `json:"grade"`
	PointClass *string           `json:"point_class"`
	Level      *int              `json:"level"`
	Vitality   *int              `json:"vitality"`
	State      *string           `json:"state"`
	X          *int              `json:"x"`
	Y          *int              `json:"y"`
	Stock      *map[string]int64 `json:"stock"`
}

// AdminListWorldResourcePoints lists public world-scope resource points.
func AdminListWorldResourcePoints(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	regionID := strings.TrimSpace(c.Query("region_id"))
	pointType := strings.TrimSpace(c.Query("point_type"))
	grade := strings.TrimSpace(c.Query("grade"))
	pointClass := strings.TrimSpace(c.Query("point_class"))
	state := strings.TrimSpace(c.Query("state"))
	search := strings.TrimSpace(c.Query("search"))

	whereClauses := []string{"p.scope = 'WORLD'"}
	args := []interface{}{}
	argIdx := 1

	if regionID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.region_id = $%d", argIdx))
		args = append(args, regionID)
		argIdx++
	}
	if pointType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.point_type = $%d", argIdx))
		args = append(args, pointType)
		argIdx++
	}
	if grade != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.grade = $%d", argIdx))
		args = append(args, grade)
		argIdx++
	}
	if pointClass != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.point_class = $%d", argIdx))
		args = append(args, pointClass)
		argIdx++
	}
	if state != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.state = $%d", argIdx))
		args = append(args, state)
		argIdx++
	}
	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(p.name ILIKE $%d OR p.point_type ILIKE $%d OR p.region_id ILIKE $%d OR COALESCE(r.name, '') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT p.id::text, p.scope, COALESCE(p.region_id, ''), COALESCE(r.name, p.region_id),
		       p.x, p.y, p.point_type, p.name, p.path, p.grade, COALESCE(p.point_class, 'NORMAL'),
		       p.level, p.vitality, p.stock, p.stock_cap, p.last_produced_at, p.state, p.created_at, p.updated_at
		FROM aperture_resource_points p
		LEFT JOIN regions r ON r.id = p.region_id
		WHERE %s
		ORDER BY p.created_at DESC
	`, strings.Join(whereClauses, " AND "))

	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi tải điểm tài nguyên thế giới", err))
		return
	}
	defer rows.Close()

	list := make([]AdminWorldResourcePointResponse, 0)
	activeCount := 0
	exhaustedCount := 0
	famousCount := 0

	for rows.Next() {
		var item AdminWorldResourcePointResponse
		var stockJSON []byte
		if errScan := rows.Scan(
			&item.ID, &item.Scope, &item.RegionID, &item.RegionName,
			&item.X, &item.Y, &item.PointType, &item.Name, &item.Path, &item.Grade, &item.PointClass,
			&item.Level, &item.Vitality, &stockJSON, &item.StockCap, &item.LastProducedAt, &item.State,
			&item.CreatedAt, &item.UpdatedAt,
		); errScan != nil {
			logger.S.Warnf("AdminListWorldResourcePoints scan error: %v", errScan)
			continue
		}
		item.Stock = make(map[string]int64)
		_ = json.Unmarshal(stockJSON, &item.Stock)

		if tpl, ok := GetResourcePointTemplate(item.PointType); ok {
			item.Template = &tpl
		}

		if item.State == "ACTIVE" {
			activeCount++
		} else {
			exhaustedCount++
		}
		if item.PointClass == "FAMOUS" {
			famousCount++
		}

		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":            list,
		"total":           len(list),
		"active_count":    activeCount,
		"exhausted_count": exhaustedCount,
		"famous_count":    famousCount,
	})
}

// AdminCreateWorldResourcePoint creates a new world resource point.
func AdminCreateWorldResourcePoint(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req AdminCreateWorldResourcePointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	req.RegionID = strings.TrimSpace(req.RegionID)
	req.PointType = strings.TrimSpace(req.PointType)
	tpl, hasTemplate := GetResourcePointTemplate(req.PointType)
	if !hasTemplate {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Mẫu điểm tài nguyên %q không tồn tại", req.PointType)})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = tpl.Name
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		path = tpl.Path
	}
	grade := strings.ToUpper(strings.TrimSpace(req.Grade))
	if grade != "LOW" && grade != "MID" && grade != "TOP" {
		grade = "MID"
	}
	pointClass := strings.ToUpper(strings.TrimSpace(req.PointClass))
	if pointClass != "NORMAL" && pointClass != "FAMOUS" {
		pointClass = tpl.PointClass
		if pointClass == "" {
			pointClass = "NORMAL"
		}
	}
	level := req.Level
	if level <= 0 {
		level = 1
	}
	vitality := req.Vitality
	if vitality <= 0 {
		vitality = 100
	}
	state := strings.ToUpper(strings.TrimSpace(req.State))
	if state == "" {
		state = "ACTIVE"
	}

	// Verify region
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM regions WHERE id = $1)`, req.RegionID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khu vực không tồn tại"})
		return
	}

	// Check occupancy
	var occupied bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS(SELECT 1 FROM aperture_resource_points WHERE scope = 'WORLD' AND region_id = $1 AND x = $2 AND y = $3)
	`, req.RegionID, req.X, req.Y).Scan(&occupied); err != nil || occupied {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ này đã có điểm tài nguyên thế giới"})
		return
	}

	var newID string
	err := db.DB.QueryRowContext(c.Request.Context(), `
		INSERT INTO aperture_resource_points
			(scope, region_id, point_type, name, path, grade, point_class, level, vitality, stock, stock_cap, x, y, last_produced_at, state)
		VALUES ('WORLD', $1, $2, $3, $4, $5, $6, $7, $8, '{}'::jsonb, 0, $9, $10, NOW(), $11)
		RETURNING id::text
	`, req.RegionID, req.PointType, name, path, grade, pointClass, level, vitality, req.X, req.Y, state).Scan(&newID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tạo điểm tài nguyên thế giới", err))
		return
	}

	publishWorldPointsChanged(req.RegionID, req.X, req.Y)

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.create_world_resource_point",
		EntityType:     "aperture_resource_point",
		EntityID:       newID,
		RequestSummary: fmt.Sprintf("Tạo điểm tài nguyên %s (%s) tại %s (%d, %d)", name, req.PointType, req.RegionID, req.X, req.Y),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tạo điểm tài nguyên thế giới thành công",
		"id":      newID,
	})
}

// AdminUpdateWorldResourcePoint updates world-scope resource point properties.
func AdminUpdateWorldResourcePoint(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID điểm tài nguyên không hợp lệ"})
		return
	}

	var req AdminUpdateWorldResourcePointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var current struct {
		RegionID   string
		Name       string
		Path       string
		Grade      string
		PointClass string
		Level      int
		Vitality   int
		State      string
		X          int
		Y          int
	}
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT COALESCE(region_id, ''), name, path, grade, COALESCE(point_class, 'NORMAL'), level, vitality, state, x, y
		FROM aperture_resource_points WHERE scope = 'WORLD' AND id = $1 FOR UPDATE
	`, id).Scan(&current.RegionID, &current.Name, &current.Path, &current.Grade, &current.PointClass, &current.Level, &current.Vitality, &current.State, &current.X, &current.Y)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Điểm tài nguyên thế giới không tồn tại"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc điểm tài nguyên", err))
		return
	}

	newName := current.Name
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		newName = strings.TrimSpace(*req.Name)
	}
	newPath := current.Path
	if req.Path != nil {
		newPath = strings.TrimSpace(*req.Path)
	}
	newGrade := current.Grade
	if req.Grade != nil {
		g := strings.ToUpper(strings.TrimSpace(*req.Grade))
		if g == "LOW" || g == "MID" || g == "TOP" {
			newGrade = g
		}
	}
	newClass := current.PointClass
	if req.PointClass != nil {
		cl := strings.ToUpper(strings.TrimSpace(*req.PointClass))
		if cl == "NORMAL" || cl == "FAMOUS" {
			newClass = cl
		}
	}
	newLevel := current.Level
	if req.Level != nil && *req.Level > 0 {
		newLevel = *req.Level
	}
	newVitality := current.Vitality
	if req.Vitality != nil {
		newVitality = *req.Vitality
		if newVitality < 0 {
			newVitality = 0
		}
	}
	newState := current.State
	if req.State != nil {
		st := strings.ToUpper(strings.TrimSpace(*req.State))
		if st != "" {
			newState = st
		}
	}
	if newVitality <= 0 && newState == "ACTIVE" {
		newState = "EXHAUSTED"
	}
	if newVitality > 0 && newState == "EXHAUSTED" {
		newState = "ACTIVE"
	}

	newX := current.X
	newY := current.Y
	if req.X != nil {
		newX = *req.X
	}
	if req.Y != nil {
		newY = *req.Y
	}
	if newX != current.X || newY != current.Y {
		var occupied bool
		if errScan := tx.QueryRowContext(c.Request.Context(), `
			SELECT EXISTS(SELECT 1 FROM aperture_resource_points WHERE scope = 'WORLD' AND region_id = $1 AND x = $2 AND y = $3 AND id <> $4)
		`, current.RegionID, newX, newY, id).Scan(&occupied); errScan != nil || occupied {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ mới đã có điểm tài nguyên khác"})
			return
		}
	}

	setStock := ""
	args := []interface{}{newName, newPath, newGrade, newClass, newLevel, newVitality, newState, newX, newY, id}
	if req.Stock != nil {
		stockBytes, _ := json.Marshal(*req.Stock)
		setStock = ", stock = $11"
		args = append(args, stockBytes)
	}

	_, err = tx.ExecContext(c.Request.Context(), fmt.Sprintf(`
		UPDATE aperture_resource_points
		SET name = $1, path = $2, grade = $3, point_class = $4, level = $5, vitality = $6, state = $7, x = $8, y = $9, updated_at = NOW()%s
		WHERE id = $10 AND scope = 'WORLD'
	`, setStock), args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể cập nhật điểm tài nguyên", err))
		return
	}

	auditlog.LogContext(c.Request.Context(), tx, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.update_world_resource_point",
		EntityType:     "aperture_resource_point",
		EntityID:       id,
		RequestSummary: fmt.Sprintf("Cập nhật điểm tài nguyên %s: name=%s, vitality=%d, state=%s", id, newName, newVitality, newState),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	if err = tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit error", err))
		return
	}

	publishWorldPointsChanged(current.RegionID, newX, newY)

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật điểm tài nguyên thế giới thành công"})
}

// AdminDeleteWorldResourcePoint removes a world-scope resource point.
func AdminDeleteWorldResourcePoint(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID điểm tài nguyên không hợp lệ"})
		return
	}

	var regionID string
	var x, y int
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(region_id, ''), x, y FROM aperture_resource_points WHERE scope = 'WORLD' AND id = $1`, id).Scan(&regionID, &x, &y)

	res, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM aperture_resource_points WHERE scope = 'WORLD' AND id = $1`, id)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi xóa điểm tài nguyên", err))
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Điểm tài nguyên thế giới không tồn tại"})
		return
	}

	if regionID != "" {
		publishWorldPointsChanged(regionID, x, y)
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.delete_world_resource_point",
		EntityType:     "aperture_resource_point",
		EntityID:       id,
		RequestSummary: fmt.Sprintf("Xóa điểm tài nguyên thế giới ID %s", id),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Xóa điểm tài nguyên thế giới thành công"})
}

// AdminSweepWorldResourcePoints triggers the top-up sweep worker logic immediately.
func AdminSweepWorldResourcePoints(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req struct {
		RegionID string `json:"region_id"`
	}
	_ = c.ShouldBindJSON(&req)
	regionID := strings.TrimSpace(req.RegionID)

	if regionID != "" {
		addedNormal, addedFamous, err := sweepOneWorldRegion(c.Request.Context(), regionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi quét khu vực: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":      fmt.Sprintf("Quét bổ sung khu vực %s hoàn tất (+%d mỏ thường, +%d mỏ danh sơn)", regionID, addedNormal, addedFamous),
			"added_normal": addedNormal,
			"added_famous": addedFamous,
		})
	} else {
		EnsureWorldResourcePointsSweep(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"message": "Đã kích hoạt quét và bù đắp toàn bộ mỏ tài nguyên thế giới"})
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.sweep_world_resource_points",
		EntityType:     "aperture_resource_point",
		EntityID:       regionID,
		RequestSummary: fmt.Sprintf("Quét bổ sung mỏ thế giới (region=%s)", regionID),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})
}

// AdminGetResourcePointTemplates returns all point templates loaded from ResourcePointTemplate.csv.
func AdminGetResourcePointTemplates(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	list := make([]ResourcePointTemplate, 0, len(ResourcePointTemplates))
	for _, t := range ResourcePointTemplates {
		if t.PointType != "" {
			list = append(list, t)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].PointType < list[j].PointType
	})

	c.JSON(http.StatusOK, gin.H{"data": list})
}
