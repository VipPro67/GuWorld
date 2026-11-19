package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminTeleportUpdateRequest struct {
	Name         string `json:"name"`
	FeeAmount    int64  `json:"fee_amount"`
	FeeEnabled   bool   `json:"fee_enabled"`
	IsActive     bool   `json:"is_active"`
	MaxRange     int    `json:"max_range"`
	AccessPolicy string `json:"access_policy"`
}

// AdminListTeleportArrays returns paginated world teleport arrays.
func AdminListTeleportArrays(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	search := strings.TrimSpace(c.Query("search"))
	regionFilter := strings.TrimSpace(c.Query("region_id"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseQuery := `
		FROM teleport_arrays ta
		JOIN regions r ON r.id = ta.region_id
		LEFT JOIN players p ON p.id = ta.owner_id
		LEFT JOIN sects s ON s.id = ta.sect_id
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (ta.name ILIKE $%d OR r.name ILIKE $%d OR p.username ILIKE $%d OR s.name ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if regionFilter != "" && regionFilter != "ALL" {
		baseQuery += fmt.Sprintf(" AND ta.region_id = $%d", argIdx)
		args = append(args, regionFilter)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đếm truyền tống trận", err))
		return
	}

	selectQuery := fmt.Sprintf(`
		SELECT ta.id, ta.name, ta.owner_id, COALESCE(p.username, 'Hệ thống'), ta.sect_id, COALESCE(s.name, ''),
		       ta.region_id, r.name, ta.x, ta.y, ta.tier, ta.max_range, ta.is_cross_region,
		       ta.direction_type, ta.access_policy, ta.fee_enabled, ta.fee_currency, ta.fee_amount,
		       ta.vault_balance, ta.current_durability, ta.max_durability, ta.is_active, ta.created_at, ta.updated_at
		%s
		ORDER BY ta.created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), selectQuery, args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc danh sách truyền tống trận", err))
		return
	}
	defer rows.Close()

	list := make([]gin.H, 0)
	for rows.Next() {
		var id, ownerID uuid.UUID
		var sectID sql.NullString
		var name, ownerName, sectName, regionID, regionName, tierVal, directionType, accessPolicy, feeCurrency string
		var x, y, maxRange, currentDurability, maxDurability int
		var isCrossRegion, feeEnabled, isActive bool
		var feeAmount, vaultBalance int64
		var createdAt, updatedAt interface{}

		if err := rows.Scan(&id, &name, &ownerID, &ownerName, &sectID, &sectName,
			&regionID, &regionName, &x, &y, &tierVal, &maxRange, &isCrossRegion,
			&directionType, &accessPolicy, &feeEnabled, &feeCurrency, &feeAmount,
			&vaultBalance, &currentDurability, &maxDurability, &isActive, &createdAt, &updatedAt); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu truyền tống trận", err))
			return
		}

		list = append(list, gin.H{
			"id":                 id.String(),
			"name":               name,
			"owner_id":           ownerID.String(),
			"owner_name":         ownerName,
			"sect_id":            sectID.String,
			"sect_name":          sectName,
			"region_id":          regionID,
			"region_name":        regionName,
			"x":                  x,
			"y":                  y,
			"tier":               tierVal,
			"max_range":          maxRange,
			"is_cross_region":    isCrossRegion,
			"direction_type":     directionType,
			"access_policy":      accessPolicy,
			"fee_enabled":        feeEnabled,
			"fee_currency":       feeCurrency,
			"fee_amount":         feeAmount,
			"vault_balance":      vaultBalance,
			"current_durability": currentDurability,
			"max_durability":     maxDurability,
			"is_active":          isActive,
			"created_at":         createdAt,
			"updated_at":         updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminUpdateTeleportArray updates settings for a teleport array.
func AdminUpdateTeleportArray(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	arrayID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID truyền tống trận không hợp lệ"})
		return
	}

	var req AdminTeleportUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	query := `
		UPDATE teleport_arrays
		SET name = COALESCE(NULLIF($1, ''), name),
		    fee_amount = $2,
		    fee_enabled = $3,
		    is_active = $4,
		    max_range = CASE WHEN $5 > 0 THEN $5 ELSE max_range END,
		    access_policy = COALESCE(NULLIF($6, ''), access_policy),
		    updated_at = NOW()
		WHERE id = $7
	`

	res, err := db.DB.ExecContext(c.Request.Context(), query, req.Name, req.FeeAmount, req.FeeEnabled, req.IsActive, req.MaxRange, req.AccessPolicy, arrayID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể cập nhật truyền tống trận", err))
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy truyền tống trận"})
		return
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "teleport_arrays",
		EntityID:       arrayID.String(),
		Action:         "admin.update_teleport_array",
		RequestSummary: fmt.Sprintf("name=%s fee_amount=%d is_active=%v", req.Name, req.FeeAmount, req.IsActive),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật truyền tống trận thành công"})
}
