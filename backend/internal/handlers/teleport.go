package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TeleportRequest represents the legacy teleport request payload.
type TeleportRequest struct {
	RegionID string `json:"region_id" binding:"required"`
}

// TeleportToRegion handles POST /api/explore/teleport
// Money-based teleportation has been removed. Only Gu-based teleportation is allowed.
func TeleportToRegion(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": "Cơ chế dịch chuyển bằng tiền (Linh Thạch / Nguyên Thạch) đã bị hủy bỏ. Hãy sử dụng Cổ trùng có khả năng dịch chuyển.",
	})
}

