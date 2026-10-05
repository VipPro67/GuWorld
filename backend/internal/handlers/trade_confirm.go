package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

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
	var senderName, receiverName string
	var status string
	var senderLocked, receiverLocked bool
	err = tx.QueryRow(`
		SELECT t.sender_id, p1.username, t.receiver_id, p2.username, t.status, t.sender_locked, t.receiver_locked 
		FROM trades t
		JOIN players p1 ON t.sender_id = p1.id
		JOIN players p2 ON t.receiver_id = p2.id
		WHERE t.id = $1 FOR UPDATE OF t
	`, tradeID).Scan(&senderID, &senderName, &receiverID, &receiverName, &status, &senderLocked, &receiverLocked)
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
	if offer.PrimevalStones < 0 || offer.ImmortalStones < 0 ||
		!pricing.IsValidAmount(offer.PrimevalStones) || !pricing.IsValidAmount(offer.ImmortalStones) {
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
		targetGu, _, found := models.FindGuByInstanceID(player.GuInventory, instID)
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
	var senderName, receiverName string
	var status string
	err = tx.QueryRow(`
		SELECT t.sender_id, p1.username, t.receiver_id, p2.username, t.status 
		FROM trades t
		JOIN players p1 ON t.sender_id = p1.id
		JOIN players p2 ON t.receiver_id = p2.id
		WHERE t.id = $1 FOR UPDATE OF t
	`, tradeID).Scan(&senderID, &senderName, &receiverID, &receiverName, &status)
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

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	notifyTradeParticipants(senderID, receiverID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã khóa vật phẩm giao dịch. Nhân Khí +1."})
}

func UnlockTrade(c *gin.Context) {
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
	var senderName, receiverName string
	var status string
	err = tx.QueryRow(`
		SELECT t.sender_id, p1.username, t.receiver_id, p2.username, t.status 
		FROM trades t
		JOIN players p1 ON t.sender_id = p1.id
		JOIN players p2 ON t.receiver_id = p2.id
		WHERE t.id = $1 FOR UPDATE OF t
	`, tradeID).Scan(&senderID, &senderName, &receiverID, &receiverName, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Giao dịch không tồn tại"})
		return
	}

	if status != "ACCEPTED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái giao dịch không cho phép mở khóa"})
		return
	}

	if authID == senderID {
		_, err = tx.Exec(`UPDATE trades SET sender_locked = false, sender_confirmed = false, receiver_confirmed = false, updated_at = NOW() WHERE id = $1`, tradeID)
	} else if authID == receiverID {
		_, err = tx.Exec(`UPDATE trades SET receiver_locked = false, sender_confirmed = false, receiver_confirmed = false, updated_at = NOW() WHERE id = $1`, tradeID)
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unlock failed"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	notifyTradeParticipants(senderID, receiverID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã mở khóa giao dịch để chỉnh sửa."})
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
	var senderName, receiverName string
	var status string
	var senderOfferBytes, receiverOfferBytes []byte
	var senderLocked, receiverLocked, senderConfirmed, receiverConfirmed bool

	err = tx.QueryRow(`
		SELECT t.sender_id, p1.username, t.receiver_id, p2.username, t.status, t.sender_offer, t.receiver_offer, 
		       t.sender_locked, t.receiver_locked, t.sender_confirmed, t.receiver_confirmed
		FROM trades t
		JOIN players p1 ON t.sender_id = p1.id
		JOIN players p2 ON t.receiver_id = p2.id
		WHERE t.id = $1 FOR UPDATE OF t
	`, tradeID).Scan(&senderID, &senderName, &receiverID, &receiverName, &status, &senderOfferBytes, &receiverOfferBytes,
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

		// Load players in deterministic order by UUID to prevent deadlocks
		var sender, receiver models.Player
		if senderID.String() < receiverID.String() {
			sender, err = evaluateManaTx(tx, senderID)
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
			sender, err = evaluateManaTx(tx, senderID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sender details"})
				return
			}
		}

		if gamefx.IsDeadState(&sender) || gamefx.IsDeadState(&receiver) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Một trong hai bên tham gia giao dịch đã rơi vào cõi chết. Giao dịch bị hủy!"})
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
			_, _, senderConnecting := models.FindFirstMatchingGu(sender.GuInventory, func(gu models.GuInstance) bool {
				return models.InventoryHasGuCapability(models.GuInventory{gu}, "REMOTE_TRADE", false)
			})
			_, _, receiverConnecting := models.FindFirstMatchingGu(receiver.GuInventory, func(gu models.GuInstance) bool {
				return models.InventoryHasGuCapability(models.GuInventory{gu}, "REMOTE_TRADE", false)
			})
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
			_, idx, found := models.FindGuByInstanceID(sender.GuInventory, instID)
			if !found {
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
			_, idx, found := models.FindGuByInstanceID(receiver.GuInventory, instID)
			if !found {
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
	var senderOffer, receiverOffer TradeOffer
	_ = json.Unmarshal(senderOfferBytes, &senderOffer)
	_ = json.Unmarshal(receiverOfferBytes, &receiverOffer)

	var senderOfferStr, receiverOfferStr string
	if senderConfirmed && receiverConfirmed {
		senderOfferStr = formatOffer(tx, senderOffer)
		receiverOfferStr = formatOffer(tx, receiverOffer)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	notifyTradeParticipants(senderID, receiverID)

	// Notify both parties about Gu transfers when trade completes
	if senderConfirmed && receiverConfirmed {
		if len(senderOffer.Gus) > 0 {
			guNames := formatGuNames(db.DB, senderOffer.Gus)
			notify.Send(db.DB, senderID, "trade_gu_sent", "Giao dịch - Cổ trùng đã chuyển",
				fmt.Sprintf("Bạn đã chuyển %d Cổ trùng cho %s: %s", len(senderOffer.Gus), receiverName, guNames))
			notify.Send(db.DB, receiverID, "trade_gu_received", "Giao dịch - Nhận Cổ trùng",
				fmt.Sprintf("%s đã chuyển cho bạn %d Cổ trùng: %s", senderName, len(senderOffer.Gus), guNames))
		}
		if len(receiverOffer.Gus) > 0 {
			guNames := formatGuNames(db.DB, receiverOffer.Gus)
			notify.Send(db.DB, receiverID, "trade_gu_sent", "Giao dịch - Cổ trùng đã chuyển",
				fmt.Sprintf("Bạn đã chuyển %d Cổ trùng cho %s: %s", len(receiverOffer.Gus), senderName, guNames))
			notify.Send(db.DB, senderID, "trade_gu_received", "Giao dịch - Nhận Cổ trùng",
				fmt.Sprintf("%s đã chuyển cho bạn %d Cổ trùng: %s", receiverName, len(receiverOffer.Gus), guNames))
		}
	}

	action := "trade.confirm"
	var confirmerName string
	if authID == senderID {
		confirmerName = senderName
	} else {
		confirmerName = receiverName
	}
	summary := fmt.Sprintf("người xác nhận=%s (%s) | người gửi=%s (%s) - người nhận=%s (%s)", confirmerName, authID, senderName, senderID, receiverName, receiverID)
	var metadata map[string]interface{}

	if senderConfirmed && receiverConfirmed {
		action = "trade.complete"
		summary = fmt.Sprintf("Hoàn tất giao dịch: %s (%s) gửi [%s] ↔ %s (%s) gửi [%s]", senderName, senderID, senderOfferStr, receiverName, receiverID, receiverOfferStr)
		metadata = map[string]interface{}{
			"sender_id":      senderID.String(),
			"sender_name":    senderName,
			"sender_offer":   senderOffer,
			"receiver_id":    receiverID.String(),
			"receiver_name":  receiverName,
			"receiver_offer": receiverOffer,
		}
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: action, EntityType: "trade",
		EntityID: tradeID, RequestSummary: summary, Metadata: metadata,
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":            "Xác nhận thành công",
		"sender_confirmed":   senderConfirmed,
		"receiver_confirmed": receiverConfirmed,
	})
}
