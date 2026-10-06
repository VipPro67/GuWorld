package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

func AdminListMapGroundDrops(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	page, limit := adminPage(c, 25, 100)
	search := strings.TrimSpace(c.Query("search"))
	pattern := "%" + search + "%"
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if status != "active" && status != "expired" {
		status = ""
	}
	filter := `($1 = '' OR d.id::text ILIKE $2 OR d.region_id ILIKE $2 OR d.loser_name ILIKE $2 OR COALESCE(d.killer_name, '') ILIKE $2 OR d.loser_id::text ILIKE $2 OR COALESCE(d.killer_id::text, '') ILIKE $2 OR d.loot_data::text ILIKE $2) AND ($3 = '' OR ($3 = 'active' AND d.expires_at > NOW()) OR ($3 = 'expired' AND d.expires_at <= NOW()))`
	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM map_ground_drops d WHERE `+filter, search, pattern, status).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm Cổ rơi trên bản đồ"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT d.id, d.region_id, d.x, d.y, d.loser_id, d.loser_name,
		       d.killer_id, COALESCE(d.killer_name, ''), d.loot_data,
		       d.protected_until, d.expires_at, d.created_at
		FROM map_ground_drops d
		WHERE `+filter+`
		ORDER BY d.created_at DESC, d.id
		LIMIT $4 OFFSET $5`, search, pattern, status, limit, (page-1)*limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Cổ rơi trên bản đồ"})
		return
	}
	defer rows.Close()
	data := make([]gin.H, 0, limit)
	for rows.Next() {
		var id, regionID, loserID, loserName, killerName string
		var killerID sql.NullString
		var x, y int
		var lootData []byte
		var protectedUntil sql.NullTime
		var expiresAt, createdAt time.Time
		if err := rows.Scan(&id, &regionID, &x, &y, &loserID, &loserName, &killerID, &killerName, &lootData, &protectedUntil, &expiresAt, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc Cổ rơi trên bản đồ"})
			return
		}
		var protected any
		if protectedUntil.Valid {
			protected = protectedUntil.Time
		}
		var killer any
		if killerID.Valid {
			killer = killerID.String
		}
		data = append(data, gin.H{"id": id, "region_id": regionID, "x": x, "y": y, "loser_id": loserID, "loser_name": loserName, "killer_id": killer, "killer_name": killerName, "loot_data": json.RawMessage(lootData), "protected_until": protected, "expires_at": expiresAt, "created_at": createdAt, "is_expired": !expiresAt.After(time.Now())})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc Cổ rơi trên bản đồ"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "total": total, "page": page, "limit": limit})
}

type adminMapGroundDropUpdate struct {
	RegionID       *string         `json:"region_id"`
	X              *int            `json:"x"`
	Y              *int            `json:"y"`
	LoserID        *string         `json:"loser_id"`
	LoserName      *string         `json:"loser_name"`
	KillerID       *string         `json:"killer_id"`
	KillerName     *string         `json:"killer_name"`
	LootData       json.RawMessage `json:"loot_data"`
	ProtectedUntil *string         `json:"protected_until"`
	ExpiresAt      *string         `json:"expires_at"`
}

func AdminUpdateMapGroundDrop(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	dropID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID map drop không hợp lệ"})
		return
	}
	var req adminMapGroundDropUpdate
	if err := c.ShouldBindJSON(&req); err != nil || req.RegionID == nil || req.X == nil || req.Y == nil || req.LoserID == nil || req.LoserName == nil || req.KillerName == nil || req.ExpiresAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần đủ các field của map drop"})
		return
	}
	regionID := strings.TrimSpace(*req.RegionID)
	loserName := strings.TrimSpace(*req.LoserName)
	killerName := strings.TrimSpace(*req.KillerName)
	loserID, err := uuid.Parse(strings.TrimSpace(*req.LoserID))
	if err != nil || regionID == "" || len(regionID) > 50 || utf8.RuneCountInString(loserName) > 100 || utf8.RuneCountInString(killerName) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vùng hoặc thông tin người chơi không hợp lệ"})
		return
	}
	var killer any
	if req.KillerID != nil && strings.TrimSpace(*req.KillerID) != "" {
		killerID, parseErr := uuid.Parse(strings.TrimSpace(*req.KillerID))
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID người kết liễu không hợp lệ"})
			return
		}
		killer = killerID
	}
	var lootObject map[string]json.RawMessage
	if len(req.LootData) == 0 || json.Unmarshal(req.LootData, &lootObject) != nil || lootObject == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "loot_data phải là một JSON object hợp lệ"})
		return
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(*req.ExpiresAt))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at phải là thời gian RFC3339"})
		return
	}
	var protected any
	if req.ProtectedUntil != nil && strings.TrimSpace(*req.ProtectedUntil) != "" {
		value := strings.TrimSpace(*req.ProtectedUntil)
		protectedAt, parseErr := time.Parse(time.RFC3339Nano, value)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "protected_until phải là thời gian RFC3339 hoặc để trống"})
			return
		}
		protected = protectedAt
	}
	var updatedID string
	err = db.DB.QueryRowContext(c.Request.Context(), `
		UPDATE map_ground_drops
		SET region_id=$1, x=$2, y=$3, loser_id=$4, loser_name=$5, killer_id=$6,
		    killer_name=$7, loot_data=$8::jsonb, protected_until=$9, expires_at=$10
		WHERE id=$11
		RETURNING id::text`, regionID, *req.X, *req.Y, loserID, loserName, killer, killerName, string(req.LootData), protected, expiresAt, dropID).Scan(&updatedID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy map drop"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể cập nhật map drop; hãy kiểm tra vùng và người chơi"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật map drop", "id": updatedID})
}

func AdminDeleteMapGroundDrop(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	dropID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID map drop không hợp lệ"})
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM map_ground_drops WHERE id=$1`, dropID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa map drop"})
		return
	}
	deleted, _ := result.RowsAffected()
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy map drop"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa map drop"})
}
