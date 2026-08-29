package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type TradeOffer struct {
	PrimevalStones float64            `json:"primeval_stones"`
	ImmortalStones float64            `json:"immortal_stones"`
	Gus            []string           `json:"gus"`       // slice of Gu instance IDs
	Materials      map[string]int     `json:"materials"` // map of itemID -> quantity
	Recipes        map[string]float64 `json:"recipes"`   // map of recipeID -> completion_rate
}

type InitiateTradeRequest struct {
	ReceiverUsername string `json:"receiver_username" binding:"required"`
}

// transferRecipe moves a recipe from one player to another in a trade.
// The receiver keeps the higher completion rate, capped at 100%.
func transferRecipe(tx *sql.Tx, fromPlayerID, toPlayerID, recipeID string, incomingRate float64) error {
	if _, err := RecipeSvc.Get(tx, fromPlayerID, recipeID); err != nil {
		return fmt.Errorf("bên gửi không sở hữu Bí phương %s", recipeID)
	}
	if err := RecipeSvc.Delete(tx, fromPlayerID, recipeID); err != nil {
		return fmt.Errorf("không thể xóa Bí phương %s từ bên gửi", recipeID)
	}
	var finalRate float64 = incomingRate
	if existing, err := RecipeSvc.Get(tx, toPlayerID, recipeID); err == nil && existing != nil {
		if existing.CompletionRate > finalRate {
			finalRate = existing.CompletionRate
		}
	}
	if finalRate > 100.0 {
		finalRate = 100.0
	}
	return RecipeSvc.Upsert(tx, &services.RecipeProgress{
		PlayerID:       toPlayerID,
		RecipeID:       recipeID,
		CompletionRate: finalRate,
		IsUnlocked:     true,
	})
}

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

	// Load sender
	sender, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sender details"})
		return
	}

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

	if receiverID == sender.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tự giao dịch với chính mình"})
		return
	}

	// Load receiver
	receiver, err := evaluateManaTx(tx, receiverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load receiver details"})
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
		senderHasConnecting := false
		for _, gu := range sender.GuInventory {
			if gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD" {
				senderHasConnecting = true
				break
			}
		}
		if !senderHasConnecting {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu cần có Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) hoặc ở gần đối phương để mở kết nối giao dịch!"})
			return
		}

		// Verify receiver has Connecting Heaven Gu
		receiverHasConnecting := false
		for _, gu := range receiver.GuInventory {
			if gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD" {
				receiverHasConnecting = true
				break
			}
		}
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

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 2, $1) WHERE id = $2", MaxHumanQi, sender.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit trade request"})
		return
	}
	notifyTradeParticipants(sender.ID, receiver.ID)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "trade.initiate", EntityType: "trade",
		EntityID:       tradeID.String(),
		RequestSummary: fmt.Sprintf("người gửi=%s → người nhận=%s", authID, receiverID),
		IPAddress:      c.ClientIP(), Success: true,
	})
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

	trade, err := loadActiveTrade(authID)
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

	senderID, receiverID, err := getTradeParticipants(tradeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Giao dịch không tồn tại"})
		return
	}

	_, err = db.DB.Exec(`
		UPDATE trades 
		SET status = 'ACCEPTED', updated_at = NOW() 
		WHERE id = $1 AND receiver_id = $2 AND status = 'PENDING'
	`, tradeID, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chấp nhận giao dịch"})
		return
	}

	notifyTradeParticipants(senderID, receiverID)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "trade.accept", EntityType: "trade",
		EntityID: tradeID, IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã chấp nhận giao dịch"})
}

func DeclineTrade(c *gin.Context) {
	tradeID := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	senderID, receiverID, err := getTradeParticipants(tradeID)
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
		EntityID: tradeID, IPAddress: c.ClientIP(), Success: true,
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

	senderID, receiverID, err := getTradeParticipants(tradeID)
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
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "trade.cancel", EntityType: "trade",
		EntityID: tradeID, IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy giao dịch"})
}

func UpdateTradeOffer(c *gin.Context) {
	tradeID := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var offer TradeOffer
	if err := c.ShouldBindJSON(&offer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu vật phẩm giao dịch không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// 1. Load trade details
	var senderID, receiverID uuid.UUID
	var status string
	var senderLocked, receiverLocked bool
	err = tx.QueryRow(`
		SELECT sender_id, receiver_id, status, sender_locked, receiver_locked 
		FROM trades WHERE id = $1 FOR UPDATE
	`, tradeID).Scan(&senderID, &receiverID, &status, &senderLocked, &receiverLocked)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phiên giao dịch"})
		return
	}

	if status != "ACCEPTED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phiên giao dịch không ở trạng thái sẵn sàng để trao đổi"})
		return
	}

	var isSender bool
	if authID == senderID {
		isSender = true
		if senderLocked {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Giao dịch đã bị khóa, không thể chỉnh sửa"})
			return
		}
	} else if authID == receiverID {
		isSender = false
		if receiverLocked {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Giao dịch đã bị khóa, không thể chỉnh sửa"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của phiên giao dịch này"})
		return
	}

	// 2. Load player inventories & check assets
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player details"})
		return
	}

	// Validate Stones
	if offer.PrimevalStones < 0 || offer.ImmortalStones < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng đá giao dịch không hợp lệ"})
		return
	}
	if player.PrimevalStones < offer.PrimevalStones {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không đủ Nguyên Thạch"})
		return
	}
	if player.ImmortalStones < offer.ImmortalStones {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không đủ Tiên Nguyên Thạch"})
		return
	}

	// Validate Gu instances
	for _, instID := range offer.Gus {
		found := false
		var targetGu models.GuInstance
		for _, g := range player.GuInventory {
			if g.InstanceID == instID {
				found = true
				targetGu = g
				break
			}
		}
		if !found {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không tìm thấy Cổ trùng trong túi: %s", instID)})
			return
		}
		if targetGu.IsVital {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể giao dịch Bản Mệnh Cổ: %s", targetGu.GuID)})
			return
		}
		if targetGu.Equipped {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể giao dịch Cổ trùng đang trang bị: %s", targetGu.GuID)})
			return
		}
	}

	// Validate Materials
	for matID, qty := range offer.Materials {
		if qty < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng vật phẩm giao dịch không hợp lệ"})
			return
		}
		availableQty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), matID)
		if availableQty < qty {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ số lượng vật phẩm trong kho: %s", matID)})
			return
		}
	}

	// Validate Recipes
	for recipeID, rate := range offer.Recipes {
		if rate < 0 || rate > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tỷ lệ hoàn thiện Bí phương không hợp lệ: %s", recipeID)})
			return
		}
		existing, err := RecipeSvc.Get(tx, player.ID.String(), recipeID)
		if err != nil || existing == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn không sở hữu Bí phương: %s", recipeID)})
			return
		}
	}

	// Serialize offer
	offerJSON, err := json.Marshal(offer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize offer"})
		return
	}

	// 3. Update DB
	// Every time an offer is updated, we unlock both sides and unconfirm both sides to prevent bait-and-switch!
	if isSender {
		_, err = tx.Exec(`
			UPDATE trades 
			SET sender_offer = $1, sender_locked = false, receiver_locked = false, 
			    sender_confirmed = false, receiver_confirmed = false, updated_at = NOW()
			WHERE id = $2
		`, offerJSON, tradeID)
	} else {
		_, err = tx.Exec(`
			UPDATE trades 
			SET receiver_offer = $1, sender_locked = false, receiver_locked = false, 
			    sender_confirmed = false, receiver_confirmed = false, updated_at = NOW()
			WHERE id = $2
		`, offerJSON, tradeID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update trade offer"})
		return
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 1, $1) WHERE id = $2", MaxHumanQi, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	notifyTradeParticipants(senderID, receiverID)

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật ưu đãi thành công, trạng thái khóa của 2 bên đã được đặt lại. Nhân Khí +1."})
}

func LockTrade(c *gin.Context) {
	tradeID := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var senderID, receiverID uuid.UUID
	var status string
	err = tx.QueryRow(`
		SELECT sender_id, receiver_id, status FROM trades WHERE id = $1 FOR UPDATE
	`, tradeID).Scan(&senderID, &receiverID, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Giao dịch không tồn tại"})
		return
	}

	if status != "ACCEPTED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái giao dịch không cho phép khóa"})
		return
	}

	if authID == senderID {
		_, err = tx.Exec(`UPDATE trades SET sender_locked = true, updated_at = NOW() WHERE id = $1`, tradeID)
	} else if authID == receiverID {
		_, err = tx.Exec(`UPDATE trades SET receiver_locked = true, updated_at = NOW() WHERE id = $1`, tradeID)
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lock failed"})
		return
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 1, $1) WHERE id = $2", MaxHumanQi, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	notifyTradeParticipants(senderID, receiverID)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "trade.lock", EntityType: "trade",
		EntityID: tradeID, IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã khóa vật phẩm giao dịch. Nhân Khí +1."})
}

func ConfirmTrade(c *gin.Context) {
	tradeID := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var senderID, receiverID uuid.UUID
	var status string
	var senderOfferBytes, receiverOfferBytes []byte
	var senderLocked, receiverLocked, senderConfirmed, receiverConfirmed bool

	err = tx.QueryRow(`
		SELECT sender_id, receiver_id, status, sender_offer, receiver_offer, 
		       sender_locked, receiver_locked, sender_confirmed, receiver_confirmed
		FROM trades WHERE id = $1 FOR UPDATE
	`, tradeID).Scan(&senderID, &receiverID, &status, &senderOfferBytes, &receiverOfferBytes,
		&senderLocked, &receiverLocked, &senderConfirmed, &receiverConfirmed)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Giao dịch không tồn tại"})
		return
	}

	if status != "ACCEPTED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Giao dịch không ở trạng thái hoạt động"})
		return
	}

	if !senderLocked || !receiverLocked {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Yêu cầu cả 2 bên phải khóa giao dịch trước khi xác nhận!"})
		return
	}

	if authID == senderID {
		senderConfirmed = true
		_, err = tx.Exec(`UPDATE trades SET sender_confirmed = true, updated_at = NOW() WHERE id = $1`, tradeID)
	} else if authID == receiverID {
		receiverConfirmed = true
		_, err = tx.Exec(`UPDATE trades SET receiver_confirmed = true, updated_at = NOW() WHERE id = $1`, tradeID)
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Confirm update failed"})
		return
	}

	// If both players have confirmed, execute the trade swap!
	if senderConfirmed && receiverConfirmed {
		var senderOffer, receiverOffer TradeOffer
		_ = json.Unmarshal(senderOfferBytes, &senderOffer)
		_ = json.Unmarshal(receiverOfferBytes, &receiverOffer)

		// Load players
		sender, err := evaluateManaTx(tx, senderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sender details"})
			return
		}

		receiver, err := evaluateManaTx(tx, receiverID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load receiver details"})
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
			// Double-check Connecting Heaven Gu on both sides
			senderConnecting := false
			for _, gu := range sender.GuInventory {
				if gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD" {
					senderConnecting = true
					break
				}
			}
			receiverConnecting := false
			for _, gu := range receiver.GuInventory {
				if gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD" {
					receiverConnecting = true
					break
				}
			}
			if !senderConnecting || !receiverConnecting {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Giao dịch thất bại! Cần phải có Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) hoặc ở gần đối phương để duy trì kết nối."})
				return
			}
		}

		// 1. Verify and deduct sender assets
		if sender.PrimevalStones < senderOffer.PrimevalStones || sender.ImmortalStones < senderOffer.ImmortalStones {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Người gửi không đủ đá linh thạch"})
			return
		}
		sender.PrimevalStones -= senderOffer.PrimevalStones
		sender.ImmortalStones -= senderOffer.ImmortalStones

		// Deduct sender Gu
		for _, instID := range senderOffer.Gus {
			idx := -1
			for i, gu := range sender.GuInventory {
				if gu.InstanceID == instID {
					idx = i
					break
				}
			}
			if idx == -1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Người gửi thiếu Cổ trùng trang bị trong ưu đãi"})
				return
			}
			if sender.GuInventory[idx].IsVital {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể giao dịch Bản Mệnh Cổ"})
				return
			}
			if sender.GuInventory[idx].Equipped {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể giao dịch Cổ trùng đang trang bị"})
				return
			}
			// Add to receiver
			receiver.GuInventory = append(receiver.GuInventory, sender.GuInventory[idx])
			// Remove from sender
			sender.GuInventory = append(sender.GuInventory[:idx], sender.GuInventory[idx+1:]...)
		}

		// Deduct sender materials
		for matID, qty := range senderOffer.Materials {
			avail, _ := InventorySvc.GetQuantity(tx, sender.ID.String(), matID)
			if avail < qty {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bên gửi thiếu vật phẩm: %s", matID)})
				return
			}
		}
		if err := InventorySvc.BulkRemoveItems(tx, sender.ID.String(), "materials", senderOffer.Materials); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ vật phẩm của người gửi"})
			return
		}
		// Add to receiver
		if err := InventorySvc.BulkAddItems(tx, receiver.ID.String(), "materials", senderOffer.Materials); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm vật phẩm cho người nhận"})
			return
		}

		// Deduct sender recipes
		for recipeID, rate := range senderOffer.Recipes {
			if err := transferRecipe(tx, sender.ID.String(), receiver.ID.String(), recipeID, rate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể chuyển Bí phương: %v", err)})
				return
			}
		}

		// 2. Verify and deduct receiver assets
		if receiver.PrimevalStones < receiverOffer.PrimevalStones || receiver.ImmortalStones < receiverOffer.ImmortalStones {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Người nhận không đủ đá linh thạch"})
			return
		}
		receiver.PrimevalStones -= receiverOffer.PrimevalStones
		receiver.ImmortalStones -= receiverOffer.ImmortalStones

		// Deduct receiver Gu
		for _, instID := range receiverOffer.Gus {
			idx := -1
			for i, gu := range receiver.GuInventory {
				if gu.InstanceID == instID {
					idx = i
					break
				}
			}
			if idx == -1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Người nhận thiếu Cổ trùng trang bị trong ưu đãi"})
				return
			}
			if receiver.GuInventory[idx].IsVital {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể giao dịch Bản Mệnh Cổ"})
				return
			}
			if receiver.GuInventory[idx].Equipped {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể giao dịch Cổ trùng đang trang bị"})
				return
			}
			// Add to sender
			sender.GuInventory = append(sender.GuInventory, receiver.GuInventory[idx])
			// Remove from receiver
			receiver.GuInventory = append(receiver.GuInventory[:idx], receiver.GuInventory[idx+1:]...)
		}

		// Deduct receiver materials
		for matID, qty := range receiverOffer.Materials {
			avail, _ := InventorySvc.GetQuantity(tx, receiver.ID.String(), matID)
			if avail < qty {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bên nhận thiếu vật phẩm: %s", matID)})
				return
			}
		}
		if err := InventorySvc.BulkRemoveItems(tx, receiver.ID.String(), "materials", receiverOffer.Materials); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ vật phẩm của người nhận"})
			return
		}
		// Add to sender
		if err := InventorySvc.BulkAddItems(tx, sender.ID.String(), "materials", receiverOffer.Materials); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm vật phẩm cho người gửi"})
			return
		}

		// Deduct receiver recipes
		for recipeID, rate := range receiverOffer.Recipes {
			if err := transferRecipe(tx, receiver.ID.String(), sender.ID.String(), recipeID, rate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể chuyển Bí phương: %v", err)})
				return
			}
		}

		// 3. Add swap items to balances
		sender.PrimevalStones += receiverOffer.PrimevalStones
		sender.ImmortalStones += receiverOffer.ImmortalStones
		receiver.PrimevalStones += senderOffer.PrimevalStones
		receiver.ImmortalStones += senderOffer.ImmortalStones

		AddHumanQi(&sender, 10.0)
		AddHumanQi(&receiver, 10.0)

		// Move the offered Gu instances to their new owners. Offered Gus can never
		// be equipped or vital (validated above), so ownership is the only change;
		// this avoids rewriting both players' entire Gu inventories.
		for _, instID := range senderOffer.Gus {
			if _, err := tx.Exec(`UPDATE player_gu SET player_id = $1, updated_at = NOW() WHERE id = $2`, receiver.ID, instID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển Cổ trùng trong giao dịch"})
				return
			}
		}
		for _, instID := range receiverOffer.Gus {
			if _, err := tx.Exec(`UPDATE player_gu SET player_id = $1, updated_at = NOW() WHERE id = $2`, sender.ID, instID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển Cổ trùng trong giao dịch"})
				return
			}
		}

		// Update both players' balances in the database
		if _, err := tx.Exec(`UPDATE players SET equipped_gus=$1, primeval_stones=$2, immortal_stones=$3, human_qi=$4 WHERE id=$5`, sender.EquippedGus, sender.PrimevalStones, sender.ImmortalStones, sender.HumanQi, sender.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ người gửi"})
			return
		}
		if _, err := tx.Exec(`UPDATE players SET equipped_gus=$1, primeval_stones=$2, immortal_stones=$3, human_qi=$4 WHERE id=$5`, receiver.EquippedGus, receiver.PrimevalStones, receiver.ImmortalStones, receiver.HumanQi, receiver.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ người nhận"})
			return
		}

		// Update trade status to COMPLETED
		_, err = tx.Exec(`UPDATE trades SET status = 'COMPLETED', updated_at = NOW() WHERE id = $1`, tradeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update trade completion status"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	notifyTradeParticipants(senderID, receiverID)
	action := "trade.confirm"
	if senderConfirmed && receiverConfirmed {
		action = "trade.complete"
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: action, EntityType: "trade",
		EntityID: tradeID, IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":            "Xác nhận thành công",
		"sender_confirmed":   senderConfirmed,
		"receiver_confirmed": receiverConfirmed,
	})
}
