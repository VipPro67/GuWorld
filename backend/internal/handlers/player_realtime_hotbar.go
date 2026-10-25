package handlers

import (
	"errors"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const maxRealtimeHotbarSlots = 6

type realtimeHotbarRequest struct {
	GuInstanceIDs []string `json:"gu_instance_ids"`
}

// UpdateRealtimeHotbarOrder assigns each equipped ACTIVE Gu to a hotbar number
// by saving its order in players.equipped_gus. It does not equip or unequip Gu.
func UpdateRealtimeHotbarOrder(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if db.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database chưa sẵn sàng"})
		return
	}
	var request realtimeHotbarRequest
	if err := c.ShouldBindJSON(&request); err != nil || len(request.GuInstanceIDs) > maxRealtimeHotbarSlots {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chọn tối đa 6 Cổ trùng cho hotbar realtime"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở giao dịch"})
		return
	}
	defer tx.Rollback()

	var currentOrder models.EquippedGusList
	if err := tx.QueryRowContext(c.Request.Context(), `
		SELECT COALESCE(equipped_gus, '[]'::jsonb)
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&currentOrder); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhân vật"})
		return
	}

	rows, err := tx.QueryContext(c.Request.Context(), `
		SELECT id FROM player_gu
		WHERE player_id = $1 AND equipped = true AND state = 'ACTIVE'
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Cổ đang trang bị"})
		return
	}
	active := make([]string, 0, maxRealtimeHotbarSlots)
	for rows.Next() {
		var instanceID string
		if err := rows.Scan(&instanceID); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc Cổ đang trang bị"})
			return
		}
		active = append(active, instanceID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc Cổ đang trang bị"})
		return
	}
	rows.Close()

	ordered, err := validateRealtimeHotbarOrder(request.GuInstanceIDs, active, currentOrder)
	if err != nil {
		message := "Thứ tự hotbar không hợp lệ"
		switch {
		case errors.Is(err, errHotbarTooManySlots):
			message = "Số nút gán vượt quá số Cổ đang trang bị hoặc giới hạn 6 nút"
		case errors.Is(err, errHotbarContainsUnequippedGu):
			message = "Chỉ có thể gán Cổ đang trang bị và hoạt động"
		case errors.Is(err, errHotbarDuplicateGu):
			message = "Mỗi Cổ chỉ được gán vào một nút"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	encoded, err := models.EquippedGusList(ordered).Value()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thứ tự hotbar"})
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `UPDATE players SET equipped_gus = $1::jsonb WHERE id = $2`, encoded, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thứ tự hotbar"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận thứ tự hotbar"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"equipped_gus": ordered, "message": "Đã gán Cổ trùng vào hotbar realtime"})
}

func validateRealtimeHotbarOrder(requested, equipped, currentOrder []string) ([]string, error) {
	if len(requested) > len(equipped) || len(requested) > maxRealtimeHotbarSlots {
		return nil, errHotbarTooManySlots
	}
	allowed := make(map[string]struct{}, len(equipped))
	for _, id := range equipped {
		if id == "" {
			return nil, errHotbarInvalidEquippedSet
		}
		allowed[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(requested))
	ordered := make([]string, 0, len(equipped))
	for _, id := range requested {
		if _, ok := allowed[id]; !ok {
			return nil, errHotbarContainsUnequippedGu
		}
		if _, ok := seen[id]; ok {
			return nil, errHotbarDuplicateGu
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	for _, id := range currentOrder {
		if _, ok := allowed[id]; ok {
			if _, included := seen[id]; !included {
				ordered = append(ordered, id)
				seen[id] = struct{}{}
			}
		}
	}
	remaining := make([]string, 0, len(equipped)-len(ordered))
	for id := range allowed {
		if _, included := seen[id]; !included {
			remaining = append(remaining, id)
		}
	}
	sort.Strings(remaining)
	ordered = append(ordered, remaining...)
	return ordered, nil
}

var (
	errHotbarTooManySlots         = errors.New("hotbar assignment exceeds equipped Gu or slot limit")
	errHotbarInvalidEquippedSet   = errors.New("equipped Gu set contains an empty ID")
	errHotbarContainsUnequippedGu = errors.New("hotbar contains an unequipped Gu")
	errHotbarDuplicateGu          = errors.New("hotbar contains a duplicate Gu")
)
