package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func InitiateTrade(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req InitiateTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên người nhận là bắt buộc"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Find receiver
	var receiverID uuid.UUID
	err = tx.QueryRow(`SELECT id FROM players WHERE username = $1`, req.ReceiverUsername).Scan(&receiverID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy người nhận chỉ định"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		}
		return
	}

	if receiverID == authID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tự giao dịch với chính mình"})
		return
	}

	var sender, receiver models.Player
	if authID.String() < receiverID.String() {
		sender, err = evaluateManaTx(tx, authID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sender details"})
			return
		}
		receiver, err = evaluateManaTx(tx, receiverID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load receiver details"})
			return
		}
	} else {
		receiver, err = evaluateManaTx(tx, receiverID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load receiver details"})
			return
		}
		sender, err = evaluateManaTx(tx, authID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sender details"})
			return
		}
	}

	if sender.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể giao dịch!"})
		return
	}
	if receiver.IsDead {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đối phương đã rơi vào cõi chết. Không thể giao dịch!"})
		return
	}

	isCloseEnough := false
	if sender.CurrentRegion == receiver.CurrentRegion {
		dist := int(math.Max(math.Abs(float64(sender.X-receiver.X)), math.Abs(float64(sender.Y-receiver.Y))))
		if dist <= 10 {
			isCloseEnough = true
		}
	}

	if !isCloseEnough {
		// Verify sender has Connecting Heaven Gu
		_, _, senderHasConnecting := models.FindFirstMatchingGu(sender.GuInventory, func(gu models.GuInstance) bool {
			return models.InventoryHasGuCapability(models.GuInventory{gu}, "REMOTE_TRADE", false)
		})
		if !senderHasConnecting {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu cần có Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) hoặc ở gần đối phương để mở kết nối giao dịch!"})
			return
		}

		// Verify receiver has Connecting Heaven Gu
		_, _, receiverHasConnecting := models.FindFirstMatchingGu(receiver.GuInventory, func(gu models.GuInstance) bool {
			return models.InventoryHasGuCapability(models.GuInventory{gu}, "REMOTE_TRADE", false)
		})
		if !receiverHasConnecting {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đối phương không sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để kết nối giao dịch từ xa!"})
			return
		}
	}

	// Check if either player is already in an active trade (PENDING or ACCEPTED)
	var activeExists bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM trades 
			WHERE (sender_id = $1 OR receiver_id = $1 OR sender_id = $2 OR receiver_id = $2) 
			  AND status IN ('PENDING', 'ACCEPTED')
		)
	`, sender.ID, receiver.ID).Scan(&activeExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database check failed"})
		return
	}
	if activeExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Một trong hai bên đang bận giao dịch khác chưa hoàn tất"})
		return
	}

	// Insert trade session
	var tradeID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO trades (sender_id, receiver_id, status)
		VALUES ($1, $2, 'PENDING')
		RETURNING id
	`, sender.ID, receiver.ID).Scan(&tradeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo phòng giao dịch"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit trade request"})
		return
	}
	notifyTradeParticipants(sender.ID, receiver.ID)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Yêu cầu giao dịch đã được gửi! Nhân Khí +2.",
		"trade_id": tradeID,
	})
}

func GetActiveTrade(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	trade, err := LoadActiveTrade(authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query active trade: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"trade": trade})
}

func AcceptTrade(c *gin.Context) {
	tradeID := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	senderID, _, receiverID, _, err := getTradeParticipantsWithUsernames(tradeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Giao dịch không tồn tại"})
		return
	}

	result, err := db.DB.Exec(`
		UPDATE trades 
		SET status = 'ACCEPTED', updated_at = NOW() 
		WHERE id = $1 AND receiver_id = $2 AND status = 'PENDING'
	`, tradeID, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chấp nhận giao dịch"})
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận trạng thái giao dịch"})
		return
	}
	if rowsAffected != 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "Giao dịch không còn ở trạng thái chờ chấp nhận"})
		return
	}

	notifyTradeParticipants(senderID, receiverID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã chấp nhận giao dịch"})
}

func DeclineTrade(c *gin.Context) {
	tradeID := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	senderID, senderName, receiverID, receiverName, err := getTradeParticipantsWithUsernames(tradeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Giao dịch không tồn tại"})
		return
	}

	_, err = db.DB.Exec(`
		UPDATE trades 
		SET status = 'DECLINED', updated_at = NOW() 
		WHERE id = $1 AND receiver_id = $2 AND status = 'PENDING'
	`, tradeID, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể từ chối giao dịch"})
		return
	}

	notifyTradeParticipants(senderID, receiverID)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "trade.decline", EntityType: "trade",
		EntityID:       tradeID,
		RequestSummary: fmt.Sprintf("người nhận=%s (%s) đã từ chối giao dịch từ người gửi=%s (%s)", receiverName, receiverID, senderName, senderID),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã từ chối giao dịch"})
}

func CancelTrade(c *gin.Context) {
	tradeID := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	senderID, senderName, receiverID, receiverName, err := getTradeParticipantsWithUsernames(tradeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Giao dịch không tồn tại"})
		return
	}

	_, err = db.DB.Exec(`
		UPDATE trades 
		SET status = 'CANCELLED', updated_at = NOW() 
		WHERE id = $1 AND (sender_id = $2 OR receiver_id = $2) AND status IN ('PENDING', 'ACCEPTED')
	`, tradeID, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hủy giao dịch"})
		return
	}

	notifyTradeParticipants(senderID, receiverID)
	var cancellerName string
	if authID == senderID {
		cancellerName = senderName
	} else {
		cancellerName = receiverName
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "trade.cancel", EntityType: "trade",
		EntityID:       tradeID,
		RequestSummary: fmt.Sprintf("người hủy=%s (%s) đã hủy giao dịch giữa người gửi=%s (%s) và người nhận=%s (%s)", cancellerName, authID, senderName, senderID, receiverName, receiverID),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy giao dịch"})
}
