package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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

	// Check region rank and inventory restrictions if entering a gated region
	inventory, err := db.LoadPlayerGus(tx, authID.String(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra Cổ trùng nhân vật"})
		return
	}
	if err := checkRegionEntry(req.TargetZoneID, player.Rank, inventory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Chưa đủ điều kiện tiến vào %s: %v", targetName, err)})
		return
	}

	oldRegion := player.CurrentRegion
	newX := int(req.TargetX)
	newY := int(req.TargetY)

	// Fallback to entry coordinates if target coords are not specified or <= 0
	if newX <= 0 && newY <= 0 && targetZone != nil {
		if targetZone.EntryX > 0 && targetZone.EntryY > 0 {
			newX = targetZone.EntryX
			newY = targetZone.EntryY
		} else {
			newX = targetZone.Width / 2
			newY = targetZone.Height / 2
		}
	}
	if newX <= 0 {
		newX = 1000
	}
	if newY <= 0 {
		newY = 1000
	}

	player.CurrentRegion = req.TargetZoneID
	player.X = newX
	player.Y = newY

	debuffMsg := ""
	if oldRegion != req.TargetZoneID {
		debuffMsg = CheckAndApplyRegionalWallDebuffBetween(&player, oldRegion, req.TargetZoneID, tx)
	}

	_, err = tx.Exec(`
		UPDATE players
		SET current_region = $1, x = $2, y = $3, statuses = $4
		WHERE id = $5
	`, player.CurrentRegion, player.X, player.Y, player.Statuses, player.ID)
	if err != nil {
		logger.S.Errorf("PortalTransfer update failed for player=%s: %v", player.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu vị trí nhân vật"})
		return
	}

	syncedMembers := SyncPartyMembersLocation(tx, &player)
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận chuyển vùng"})
		return
	}

	logMsg := fmt.Sprintf("🌀 Đã tiến vào %s (%d, %d).", targetName, newX, newY)
	if debuffMsg != "" {
		logMsg += " " + debuffMsg
	}

	broadcastWorldPos(WorldPosEvent{
		ID:        player.ID.String(),
		Username:  player.Username,
		AvatarURL: player.AvatarURL,
		X:         player.X,
		Y:         player.Y,
		Region:    player.CurrentRegion,
		TS:        time.Now().UnixMilli(),
	})
	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated)

	for _, sm := range syncedMembers {
		broadcastWorldPos(WorldPosEvent{
			ID:        sm.ID.String(),
			Username:  sm.Username,
			AvatarURL: sm.AvatarURL,
			X:         sm.X,
			Y:         sm.Y,
			Region:    sm.Region,
			TS:        time.Now().UnixMilli(),
		})
		PublishPlayerRealtimeEvent(sm.ID, PlayerEventUpdated)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": logMsg,
		"player":  player,
	})
}
