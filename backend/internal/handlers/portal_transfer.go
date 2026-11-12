package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type PortalTransferReq struct {
	TargetZoneID string  `json:"target_zone_id" binding:"required"`
	TargetX      float64 `json:"target_x"`
	TargetY      float64 `json:"target_y"`
}

// PortalTransfer transitions a character from their current zone/map to a target zone.
// Called when a player steps on or interacts with an authored map portal.
func PortalTransfer(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if rejectRealtimeOwned(c, authID) {
		return
	}

	var req PortalTransferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu chuyển vùng không hợp lệ: " + err.Error()})
		return
	}

	if req.TargetZoneID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vùng đích không được để trống"})
		return
	}

	targetZone := ZoneByID(req.TargetZoneID)
	targetName := req.TargetZoneID
	if targetZone != nil && targetZone.Name != "" {
		targetName = targetZone.Name
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kết nối cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}

	// Portal metadata is authored in map-local cells (0..width/height). Persist
	// the legacy world projection so realtime admission converts it back to the
	// same local cell instead of falling back to a safe spawn.
	newX := int(req.TargetX)
	newY := int(req.TargetY)

	if newX <= 0 && newY <= 0 && targetZone != nil {
		if targetZone.EntryX > 0 && targetZone.EntryY > 0 {
			newX = targetZone.EntryX
			newY = targetZone.EntryY
		} else {
			newX = targetZone.Width / 2
			newY = targetZone.Height / 2
		}
	}
	if targetZone != nil && (req.TargetX > 0 || req.TargetY > 0) {
		newX, newY = LegacyPortalDestination(targetZone, int(req.TargetX), int(req.TargetY))
	}
	if newX <= 0 {
		newX = 1000
	}
	if newY <= 0 {
		newY = 1000
	}

	transition, errMove := ExecuteSpatialTransition(c.Request.Context(), tx, &player, movementSpec{
		kind:         movementPortal,
		x:            newX,
		y:            newY,
		hasTarget:    true,
		targetRegion: req.TargetZoneID,
	}, publishPartyRealtimeEvent)
	if errMove != nil {
		if respondMovementError(c, errMove) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMove.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận chuyển vùng"})
		return
	}

	logMsg := fmt.Sprintf("🌀 Cổ Đại Truyền Tống Trận đưa đạo hữu tiến vào %s (%d, %d).", targetName, newX, newY)
	if transition.DebuffMessage != "" {
		logMsg += " " + transition.DebuffMessage
	}

	transition.PublishPostCommit()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": logMsg,
		"player":  player,
	})
}

// LegacyPortalDestination projects an authored map-local portal target into world coordinates.
func LegacyPortalDestination(target *models.Region, localX, localY int) (int, int) {
	x := localX
	y := localY
	if target != nil {
		if target.Width > 0 && x > target.Width-2 {
			x = target.Width - 2
		}
		if target.Height > 0 && y > target.Height-2 {
			y = target.Height - 2
		}
		if x < 1 {
			x = 1
		}
		if y < 1 {
			y = 1
		}
		if target.MaxX > target.MinX {
			x = target.MinX + x
		}
		if target.MaxY > target.MinY {
			y = target.MinY + y
		}
	}
	return x, y
}
