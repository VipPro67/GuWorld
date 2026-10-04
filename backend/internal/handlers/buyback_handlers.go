package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type CreateBuybackOrderRequest struct {
	OrderType         string          `json:"order_type"`                   // BUY|SELL (default BUY)
	ItemType          string          `json:"item_type" binding:"required"` // Gu|Material|Recipe
	ItemTarget        json.RawMessage `json:"item_target" binding:"required"`
	GuInstanceID      string          `json:"gu_instance_id"`
	GuInstanceIDs     []string        `json:"gu_instance_ids"`
	UnitPrice         float64         `json:"unit_price" binding:"required"`
	QuantityRequested int64           `json:"quantity_requested" binding:"required"`
	Currency          string          `json:"currency" binding:"required"` // primeval|immortal
	DurationHours     int             `json:"duration_hours" binding:"required"`
}

type SellToBuybackRequest struct {
	OrderID       string   `json:"order_id" binding:"required"`
	Quantity      int64    `json:"quantity"`
	GuInstanceID  string   `json:"gu_instance_id"`
	GuInstanceIDs []string `json:"gu_instance_ids"`
}

type CancelBuybackRequest struct {
	OrderID string `json:"order_id" binding:"required"`
}

type BuybackOrderResponse struct {
	ID                string          `json:"id"`
	BuyerID           string          `json:"buyer_id"`
	BuyerUsername     string          `json:"buyer_username"`
	OrderType         string          `json:"order_type"`
	ItemType          string          `json:"item_type"`
	ItemTarget        json.RawMessage `json:"item_target"`
	ItemData          json.RawMessage `json:"item_data"`
	UnitPrice         float64         `json:"unit_price"`
	QuantityRequested int64           `json:"quantity_requested"`
	QuantityFilled    int64           `json:"quantity_filled"`
	Currency          string          `json:"currency"`
	UpfrontFee        float64         `json:"upfront_fee"`
	TotalLocked       float64         `json:"total_locked"`
	Status            string          `json:"status"`
	DurationHours     int             `json:"duration_hours"`
	EndTime           time.Time       `json:"end_time"`
	CreatedAt         time.Time       `json:"created_at"`
}

func getPlayerMaterialsMap(q db.Queryer, playerID uuid.UUID) map[string]int64 {
	m := make(map[string]int64)
	rows, err := q.Query(`SELECT item_id, quantity FROM player_inventory_items WHERE player_id = $1 AND category = 'materials'`, playerID)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var itemID string
		var qty int64
		if err := rows.Scan(&itemID, &qty); err == nil {
			m[itemID] = qty
		}
	}
	return m
}

// CreateBuybackOrder creates a BUY (Thu Mua) or SELL (Đăng Bán) order on Bảo Hoàng Thiên Order Book
func CreateBuybackOrder(c *gin.Context) {
	var req CreateBuybackOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.OrderType != "SELL" {
		req.OrderType = "BUY"
	}
	if req.UnitPrice < 0.001 || !pricing.IsValidAmount(req.UnitPrice) || req.QuantityRequested < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn giá và số lượng phải hợp lệ"})
		return
	}
	if req.Currency != "primeval" && req.Currency != "immortal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Currency phải là primeval hoặc immortal"})
		return
	}
	if req.ItemType != "Gu" && req.ItemType != "Material" && req.ItemType != "Recipe" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ItemType phải là Gu, Material hoặc Recipe"})
		return
	}
	if req.ItemType == "Recipe" && req.QuantityRequested != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi đơn Bí phương chỉ được giao dịch số lượng 1"})
		return
	}
	if req.DurationHours < 1 || req.DurationHours > 168 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời hạn đơn giao dịch từ 1 đến 168 giờ"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	creator, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player details"})
		return
	}
	if gamefx.IsDeadState(&creator) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể tạo đơn giao dịch!"})
		return
	}
	if !HasBaoHoangAccess(creator.GuInventory) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) và Thần Niệm Cổ (GU_DIVINE_SENSE) để sử dụng Bảo Hoàng Thiên!"})
		return
	}
	if creator.Rank < 6 && req.Currency == "immortal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ sư dưới Lục Chuyển không thể dùng Tiên Nguyên Thạch"})
		return
	}

	totalItemCost := pricing.Quantize(req.UnitPrice * float64(req.QuantityRequested))
	upfrontFee := pricing.Quantize(totalItemCost * 0.04)
	var itemDataJSON []byte

	if req.OrderType == "BUY" {
		totalLocked := totalItemCost + upfrontFee
		if req.Currency == "immortal" {
			if creator.ImmortalStones < totalLocked {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch (Cần %.3f, hiện có %.3f)", totalLocked, creator.ImmortalStones)})
				return
			}
			_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, totalLocked, creator.ID)
		} else {
			if creator.PrimevalStones < totalLocked {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch (Cần %.3f, hiện có %.3f)", totalLocked, creator.PrimevalStones)})
				return
			}
			_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2`, totalLocked, creator.ID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ tài sản"})
			return
		}
	} else {
		// SELL Order: Lock 4% upfront fee + lock seller items
		if req.Currency == "immortal" {
			if creator.ImmortalStones < upfrontFee {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch nộp phí niêm yết (Cần %.3f, hiện có %.3f)", upfrontFee, creator.ImmortalStones)})
				return
			}
			_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, upfrontFee, creator.ID)
		} else {
			if creator.PrimevalStones < upfrontFee {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch nộp phí niêm yết (Cần %.3f, hiện có %.3f)", upfrontFee, creator.PrimevalStones)})
				return
			}
			_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2`, upfrontFee, creator.ID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ phí niêm yết"})
			return
		}

		if req.ItemType == "Material" {
			var targetMat string
			_ = json.Unmarshal(req.ItemTarget, &targetMat)
			mats := getPlayerMaterialsMap(tx, creator.ID)
			if mats[targetMat] < req.QuantityRequested {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s (Cần %d, hiện có %d)", targetMat, req.QuantityRequested, mats[targetMat])})
				return
			}
			if err := deductMaterialTx(tx, creator.ID, targetMat, req.QuantityRequested); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ nguyên liệu rao bán"})
				return
			}
		} else if req.ItemType == "Gu" {
			var guInstIDs []string
			if len(req.GuInstanceIDs) > 0 {
				guInstIDs = req.GuInstanceIDs
			} else if req.GuInstanceID != "" {
				guInstIDs = []string{req.GuInstanceID}
			}
			if int64(len(guInstIDs)) != req.QuantityRequested {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần chọn đúng %d Cổ trùng để đăng bán", req.QuantityRequested)})
				return
			}
			if hasDuplicateGuInstanceIDs(guInstIDs) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách Cổ trùng không được chứa ID trùng lặp"})
				return
			}
			var target struct {
				GuID string `json:"gu_id"`
				Rank int    `json:"rank"`
			}
			if err := json.Unmarshal(req.ItemTarget, &target); err != nil || target.GuID == "" || target.Rank < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Mục tiêu Cổ trùng không hợp lệ"})
				return
			}
			for _, instID := range guInstIDs {
				var ownerID uuid.UUID
				var guID, state string
				var rank int
				var isVital, equipped, isLocked bool
				err := tx.QueryRow(`SELECT player_id, gu_id, gu_rank, is_vital, equipped, is_locked, state FROM player_gu WHERE id = $1 FOR UPDATE`, instID).Scan(&ownerID, &guID, &rank, &isVital, &equipped, &isLocked, &state)
				if err != nil || ownerID != creator.ID {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không sở hữu Cổ trùng đăng bán"})
					return
				}
				if isVital {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đăng bán Bổn Mạng Cổ!"})
					return
				}
				if guID != target.GuID || rank != target.Rank {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đăng bán không đúng loại hoặc hạng đã chọn"})
					return
				}
				if equipped || isLocked || state == "DEAD" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đang trang bị, bị khóa hoặc đã chết không thể đăng bán"})
					return
				}
				_, err = tx.Exec(`UPDATE player_gu SET player_id = $1, equipped = false WHERE id = $2`, BHTHSystemUUID, instID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khóa Cổ trùng rao bán"})
					return
				}
			}
			itemDataJSON, _ = json.Marshal(guInstIDs)
		} else if req.ItemType == "Recipe" {
			var recTarget string
			_ = json.Unmarshal(req.ItemTarget, &recTarget)
			recProgress, err := RecipeSvc.Get(tx, creator.ID.String(), recTarget)
			if err != nil || recProgress == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn chưa khai mở Bí phương này để bán"})
				return
			}
			if err := RecipeSvc.Delete(tx, creator.ID.String(), recTarget); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khóa Bí phương rao bán"})
				return
			}
			itemDataJSON, _ = json.Marshal(recProgress)
		}
	}

	totalLocked := totalItemCost + upfrontFee
	endTime := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
	var orderID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO buyback_orders (buyer_id, order_type, item_type, item_target, item_data, unit_price, quantity_requested, quantity_filled, currency, upfront_fee, total_locked, status, duration_hours, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8, $9, $10, 'ACTIVE', $11, $12)
		RETURNING id`,
		creator.ID, req.OrderType, req.ItemType, req.ItemTarget, itemDataJSON, req.UnitPrice, req.QuantityRequested, req.Currency, upfrontFee, totalLocked, req.DurationHours, endTime,
	).Scan(&orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo đơn giao dịch"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit transaction error"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &creator.ID, Action: "marketplace.create_order",
		EntityType: "buyback_order", EntityID: orderID.String(),
		RequestSummary: fmt.Sprintf("Tạo đơn %s %d %s (giá %.3f)", req.OrderType, req.QuantityRequested, req.ItemType, req.UnitPrice),
		IPAddress:      c.ClientIP(), Success: true,
	})

	tx2, _ := db.DB.Begin()
	updatedPlayer, _ := evaluateManaTx(tx2, creator.ID)
	_ = tx2.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("Tạo đơn %s thành công trên Bảo Hoàng Thiên!", req.OrderType),
		"order_id": orderID.String(),
		"player":   updatedPlayer,
	})
}

// GetBuybackOrders lists active buyback & sell orders
func GetBuybackOrders(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
		return
	}
	player, err := evaluateManaTx(tx, authID)
	_ = tx.Commit()
	if err != nil || !HasBaoHoangAccess(player.GuInventory) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Thông Thiên Cổ và Thần Niệm Cổ để kết nối Bảo Hoàng Thiên!"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT b.id, b.buyer_id, p.username, b.order_type, b.item_type, b.item_target, COALESCE(b.item_data, 'null'::jsonb), b.unit_price, b.quantity_requested, b.quantity_filled, b.currency, b.upfront_fee, b.total_locked, b.status, b.duration_hours, b.end_time, b.created_at
		FROM buyback_orders b
		JOIN players p ON p.id = b.buyer_id
		WHERE b.status = 'ACTIVE' AND b.end_time > NOW()
		ORDER BY b.created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query error"})
		return
	}
	defer rows.Close()

	res := []BuybackOrderResponse{}
	for rows.Next() {
		var o BuybackOrderResponse
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.BuyerUsername, &o.OrderType, &o.ItemType, &o.ItemTarget, &o.ItemData, &o.UnitPrice, &o.QuantityRequested, &o.QuantityFilled, &o.Currency, &o.UpfrontFee, &o.TotalLocked, &o.Status, &o.DurationHours, &o.EndTime, &o.CreatedAt); err == nil {
			if o.OrderType == "" {
				o.OrderType = "BUY"
			}
			res = append(res, o)
		}
	}
	c.JSON(http.StatusOK, res)
}

// SellToBuybackOrder executes matching transaction against a BUY or SELL order
func SellToBuybackOrder(c *gin.Context) {
	var req SellToBuybackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orderUUID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OrderID không hợp lệ"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	actor, err := evaluateManaTx(tx, authID)
	if err != nil || gamefx.IsDeadState(&actor) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể thực hiện giao dịch!"})
		return
	}
	if !HasBaoHoangAccess(actor.GuInventory) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ và Thần Niệm Cổ để giao dịch Bảo Hoàng Thiên!"})
		return
	}

	var o BuybackOrderResponse
	err = tx.QueryRow(`
		SELECT b.id, b.buyer_id, p.username, b.order_type, b.item_type, b.item_target, COALESCE(b.item_data, 'null'::jsonb), b.unit_price, b.quantity_requested, b.quantity_filled, b.currency, b.upfront_fee, b.total_locked, b.status, b.duration_hours, b.end_time, b.created_at
		FROM buyback_orders b
		JOIN players p ON p.id = b.buyer_id
		WHERE b.id = $1 FOR UPDATE`, orderUUID,
	).Scan(&o.ID, &o.BuyerID, &o.BuyerUsername, &o.OrderType, &o.ItemType, &o.ItemTarget, &o.ItemData, &o.UnitPrice, &o.QuantityRequested, &o.QuantityFilled, &o.Currency, &o.UpfrontFee, &o.TotalLocked, &o.Status, &o.DurationHours, &o.EndTime, &o.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Đơn không tồn tại"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Query error"})
		}
		return
	}
	if o.OrderType == "" {
		o.OrderType = "BUY"
	}

	if o.Status != "ACTIVE" || time.Now().After(o.EndTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn giao dịch đã hết hạn hoặc không còn hoạt động"})
		return
	}

	if o.BuyerID == actor.ID.String() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tự giao dịch với đơn của chính mình"})
		return
	}

	remainingRequested := o.QuantityRequested - o.QuantityFilled
	tradeQty := req.Quantity
	if tradeQty <= 0 {
		tradeQty = 1
	}
	if tradeQty > remainingRequested {
		tradeQty = remainingRequested
	}

	if o.OrderType == "BUY" {
		// Actor is SELLER selling into a BUY Order
		totalGross := pricing.Quantize(o.UnitPrice * float64(tradeQty))
		commission := pricing.Quantize(totalGross * 0.06)
		sellerPayout := totalGross - commission

		buyerUUID := uuid.MustParse(o.BuyerID)
		buyer, err := evaluateManaTx(tx, buyerUUID)
		if err != nil || !HasBaoHoangAccess(buyer.GuInventory) {
			unfilled := o.QuantityRequested - o.QuantityFilled
			refundAmt := pricing.Quantize(float64(unfilled) * o.UnitPrice)
			if o.Currency == "immortal" {
				_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, refundAmt, buyerUUID)
			} else {
				_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, refundAmt, buyerUUID)
			}
			_, _ = tx.Exec(`UPDATE buyback_orders SET status = 'CANCELLED' WHERE id = $1`, orderUUID)
			_ = tx.Commit()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Người mua không còn đủ điều kiện kết nối Bảo Hoàng Thiên. Đơn thu mua đã bị hủy và hoàn tiền."})
			return
		}

		if o.ItemType == "Material" {
			var targetMat string
			_ = json.Unmarshal(o.ItemTarget, &targetMat)
			mats := getPlayerMaterialsMap(tx, actor.ID)
			if mats[targetMat] < tradeQty {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn không đủ nguyên liệu %s (Cần %d, hiện có %d)", targetMat, tradeQty, mats[targetMat])})
				return
			}
			_ = deductMaterialTx(tx, actor.ID, targetMat, tradeQty)
			_ = addMaterialTx(tx, buyerUUID, targetMat, tradeQty)
		} else if o.ItemType == "Gu" {
			var guInstIDs []string
			if len(req.GuInstanceIDs) > 0 {
				guInstIDs = req.GuInstanceIDs
			} else if req.GuInstanceID != "" {
				guInstIDs = []string{req.GuInstanceID}
			}
			if int64(len(guInstIDs)) != tradeQty {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần chọn đúng %d Cổ trùng để bán", tradeQty)})
				return
			}
			if hasDuplicateGuInstanceIDs(guInstIDs) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách Cổ trùng không được chứa ID trùng lặp"})
				return
			}
			var target struct {
				GuID string `json:"gu_id"`
				Rank int    `json:"rank"`
			}
			if err := json.Unmarshal(o.ItemTarget, &target); err != nil || target.GuID == "" || target.Rank < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Mục tiêu Cổ trùng của đơn thu mua không hợp lệ"})
				return
			}
			for _, instID := range guInstIDs {
				var ownerID uuid.UUID
				var guID, state string
				var rank int
				var isVital, equipped, isLocked bool
				err := tx.QueryRow(`SELECT player_id, gu_id, gu_rank, is_vital, equipped, is_locked, state FROM player_gu WHERE id = $1 FOR UPDATE`, instID).Scan(&ownerID, &guID, &rank, &isVital, &equipped, &isLocked, &state)
				if err != nil || ownerID != actor.ID {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không sở hữu Cổ trùng này"})
					return
				}
				if isVital {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể bán Bổn Mạng Cổ!"})
					return
				}
				if guID != target.GuID || rank != target.Rank {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng không đúng loại hoặc hạng mà đơn thu mua yêu cầu"})
					return
				}
				if equipped || isLocked || state == "DEAD" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đang trang bị, bị khóa hoặc đã chết không thể bán"})
					return
				}
				if _, err := tx.Exec(`UPDATE player_gu SET player_id = $1, equipped = false WHERE id = $2 AND player_id = $3`, buyerUUID, instID, actor.ID); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi chuyển giao Cổ trùng"})
					return
				}
			}
		} else if o.ItemType == "Recipe" {
			var recTarget string
			_ = json.Unmarshal(o.ItemTarget, &recTarget)
			recProgress, err := RecipeSvc.Get(tx, actor.ID.String(), recTarget)
			if err != nil || recProgress == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn chưa khai mở Bí phương này để bán"})
				return
			}
			if err := transferRecipe(tx, actor.ID.String(), buyerUUID.String(), recTarget, recProgress.CompletionRate); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi chuyển giao Bí phương"})
				return
			}
		}

		if o.Currency == "immortal" {
			_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, sellerPayout, actor.ID)
		} else {
			_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, sellerPayout, actor.ID)
		}

		newFilled := o.QuantityFilled + tradeQty
		newStatus := "ACTIVE"
		if newFilled >= o.QuantityRequested {
			newStatus = "FILLED"
		}
		_, _ = tx.Exec(`UPDATE buyback_orders SET quantity_filled = $1, status = $2 WHERE id = $3`, newFilled, newStatus, orderUUID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit error: " + err.Error()})
			return
		}

		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID: &actor.ID, Action: "marketplace.sell_buyback",
			EntityType: "buyback_order", EntityID: orderUUID.String(),
			RequestSummary: fmt.Sprintf("Bán %d vật phẩm vào đơn thu mua (nhận %.3f)", tradeQty, sellerPayout),
			IPAddress:      c.ClientIP(), Success: true,
		})

		// Notify the BUY order creator that their order was (partially) filled
		currencyLabel := "Nguyên Thạch"
		if o.Currency == "immortal" {
			currencyLabel = "Tiên Nguyên Thạch"
		}
		notify.Send(db.DB, buyerUUID, "buyback_filled", "Đơn Thu Mua Đã Khớp",
			fmt.Sprintf("Đạo hữu %s đã bán %d vật phẩm vào đơn thu mua của bạn. Bạn chi ra %.3f %s.", actor.Username, tradeQty, pricing.Quantize(o.UnitPrice*float64(tradeQty)), currencyLabel))

		tx2, _ := db.DB.Begin()
		updatedActor, _ := evaluateManaTx(tx2, actor.ID)
		_ = tx2.Commit()

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Bán thành công %d vật phẩm vào đơn thu mua! Nhận %.3f %s (Phí 6%%: %.3f).", tradeQty, sellerPayout, o.Currency, commission),
			"player":  updatedActor,
		})
	} else {
		// Actor is BUYER buying from a SELL Order
		totalCost := pricing.Quantize(o.UnitPrice * float64(tradeQty))
		commission := pricing.Quantize(totalCost * 0.06)
		sellerPayout := totalCost - commission
		sellerUUID := uuid.MustParse(o.BuyerID)

		if o.Currency == "immortal" {
			if actor.ImmortalStones < totalCost {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch (Cần %.3f, hiện có %.3f)", totalCost, actor.ImmortalStones)})
				return
			}
			_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, totalCost, actor.ID)
			_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, sellerPayout, sellerUUID)
		} else {
			if actor.PrimevalStones < totalCost {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch (Cần %.3f, hiện có %.3f)", totalCost, actor.PrimevalStones)})
				return
			}
			_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2`, totalCost, actor.ID)
			_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, sellerPayout, sellerUUID)
		}

		if o.ItemType == "Material" {
			var targetMat string
			_ = json.Unmarshal(o.ItemTarget, &targetMat)
			_ = addMaterialTx(tx, actor.ID, targetMat, tradeQty)
		} else if o.ItemType == "Gu" {
			var lockedGuIDs []string
			_ = json.Unmarshal(o.ItemData, &lockedGuIDs)
			if int64(len(lockedGuIDs)) < tradeQty {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng Cổ trùng khóa không đủ"})
				return
			}
			toDeliver, ok := lockedGuIDsForFill(lockedGuIDs, o.QuantityFilled, tradeQty)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng Cổ trùng khóa không đủ"})
				return
			}
			for _, instID := range toDeliver {
				_, _ = tx.Exec(`UPDATE player_gu SET player_id = $1, equipped = false WHERE id = $2`, actor.ID, instID)
			}
		} else if o.ItemType == "Recipe" {
			var recProgress services.RecipeProgress
			_ = json.Unmarshal(o.ItemData, &recProgress)
			finalRate := recProgress.CompletionRate
			if existing, err := RecipeSvc.Get(tx, actor.ID.String(), recProgress.RecipeID); err == nil && existing != nil {
				if existing.CompletionRate > finalRate {
					finalRate = existing.CompletionRate
				}
			}
			if finalRate > 100.0 {
				finalRate = 100.0
			}
			_ = RecipeSvc.Upsert(tx, &services.RecipeProgress{
				PlayerID:       actor.ID.String(),
				RecipeID:       recProgress.RecipeID,
				CompletionRate: finalRate,
				IsUnlocked:     true,
			})
		}

		newFilled := o.QuantityFilled + tradeQty
		newStatus := "ACTIVE"
		if newFilled >= o.QuantityRequested {
			newStatus = "FILLED"
		}
		_, _ = tx.Exec(`UPDATE buyback_orders SET quantity_filled = $1, status = $2 WHERE id = $3`, newFilled, newStatus, orderUUID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit error: " + err.Error()})
			return
		}

		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID: &actor.ID, Action: "marketplace.buy_sell_order",
			EntityType: "buyback_order", EntityID: orderUUID.String(),
			RequestSummary: fmt.Sprintf("Mua %d vật phẩm từ đơn rao bán (trả %.3f)", tradeQty, totalCost),
			IPAddress:      c.ClientIP(), Success: true,
		})

		// Notify the SELL order creator that their item was sold
		currencyLabel2 := "Nguyên Thạch"
		if o.Currency == "immortal" {
			currencyLabel2 = "Tiên Nguyên Thạch"
		}
		notify.Send(db.DB, sellerUUID, "sell_order_filled", "Đơn Đăng Bán Đã Khớp",
			fmt.Sprintf("Đạo hữu %s đã mua %d vật phẩm trong đơn đăng bán của bạn. Bạn nhận %.3f %s.", actor.Username, tradeQty, sellerPayout, currencyLabel2))

		tx2, _ := db.DB.Begin()
		updatedActor, _ := evaluateManaTx(tx2, actor.ID)
		_ = tx2.Commit()

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Mua thành công %d vật phẩm! Tiêu tốn %.3f %s.", tradeQty, totalCost, o.Currency),
			"player":  updatedActor,
		})
	}
}

func hasDuplicateGuInstanceIDs(ids []string) bool {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			return true
		}
		if _, exists := seen[id]; exists {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}

func lockedGuIDsForFill(ids []string, filled, quantity int64) ([]string, bool) {
	end := filled + quantity
	if filled < 0 || quantity < 1 || end > int64(len(ids)) {
		return nil, false
	}
	return ids[filled:end], true
}

func remainingLockedGuIDs(ids []string, filled int64) ([]string, bool) {
	if filled < 0 || filled > int64(len(ids)) {
		return nil, false
	}
	return ids[filled:], true
}

// CancelBuybackOrder cancels an active buyback or sell order and refunds remaining locked funds/items
func CancelBuybackOrder(c *gin.Context) {
	var req CancelBuybackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orderUUID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OrderID không hợp lệ"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	creator, err := evaluateManaTx(tx, authID)
	if err != nil || gamefx.IsDeadState(&creator) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể hủy đơn!"})
		return
	}

	var creatorID uuid.UUID
	var orderType, itemType string
	var itemTarget, itemData json.RawMessage
	var unitPrice float64
	var qtyRequested, qtyFilled int64
	var currency, status string

	err = tx.QueryRow(`
		SELECT buyer_id, order_type, item_type, item_target, COALESCE(item_data, 'null'::jsonb), unit_price, quantity_requested, quantity_filled, currency, status
		FROM buyback_orders WHERE id = $1 FOR UPDATE`, orderUUID,
	).Scan(&creatorID, &orderType, &itemType, &itemTarget, &itemData, &unitPrice, &qtyRequested, &qtyFilled, &currency, &status)

	if err != nil || creatorID != creator.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn không tồn tại hoặc bạn không có quyền hủy"})
		return
	}
	if status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn không còn ở trạng thái ACTIVE"})
		return
	}

	unfilledQty := qtyRequested - qtyFilled
	if unfilledQty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn đã được khớp hoàn toàn"})
		return
	}

	if orderType == "BUY" {
		refundAmount := pricing.Quantize(float64(unfilledQty) * unitPrice)
		if currency == "immortal" {
			_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, refundAmount, creator.ID)
		} else {
			_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, refundAmount, creator.ID)
		}
	} else {
		// SELL Order: restore unfilled items
		if itemType == "Material" {
			var targetMat string
			_ = json.Unmarshal(itemTarget, &targetMat)
			_ = addMaterialTx(tx, creator.ID, targetMat, unfilledQty)
		} else if itemType == "Gu" {
			var lockedGuIDs []string
			_ = json.Unmarshal(itemData, &lockedGuIDs)
			remainingGuIDs, ok := remainingLockedGuIDs(lockedGuIDs, qtyFilled)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu Cổ trùng khóa không hợp lệ"})
				return
			}
			for _, instID := range remainingGuIDs {
				_, _ = tx.Exec(`UPDATE player_gu SET player_id = $1 WHERE id = $2`, creator.ID, instID)
			}
		} else if itemType == "Recipe" {
			var recProgress services.RecipeProgress
			_ = json.Unmarshal(itemData, &recProgress)
			_ = RecipeSvc.Upsert(tx, &services.RecipeProgress{
				PlayerID:       creator.ID.String(),
				RecipeID:       recProgress.RecipeID,
				CompletionRate: recProgress.CompletionRate,
				IsUnlocked:     true,
			})
		}
	}

	_, err = tx.Exec(`UPDATE buyback_orders SET status = 'CANCELLED' WHERE id = $1`, orderUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái đơn: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit error: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &creator.ID, Action: "marketplace.cancel_order",
		EntityType: "buyback_order", EntityID: orderUUID.String(),
		RequestSummary: fmt.Sprintf("Hủy đơn %s thành công", orderType),
		IPAddress:      c.ClientIP(), Success: true,
	})

	tx2, _ := db.DB.Begin()
	updatedCreator, _ := evaluateManaTx(tx2, creator.ID)
	_ = tx2.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Hủy đơn %s thành công! Đã hoàn trả tài sản chưa khớp.", orderType),
		"player":  updatedCreator,
	})
}

// Helpers for material deduction & addition in transactions
func deductMaterialTx(tx *sql.Tx, playerID uuid.UUID, matID string, qty int64) error {
	var currentQty int64
	err := tx.QueryRow(`SELECT quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2 AND category = 'materials' FOR UPDATE`, playerID, matID).Scan(&currentQty)
	if err != nil || currentQty < qty {
		return fmt.Errorf("insufficient material %s", matID)
	}
	if currentQty == qty {
		_, err = tx.Exec(`DELETE FROM player_inventory_items WHERE player_id = $1 AND item_id = $2 AND category = 'materials'`, playerID, matID)
	} else {
		_, err = tx.Exec(`UPDATE player_inventory_items SET quantity = quantity - $1 WHERE player_id = $2 AND item_id = $3 AND category = 'materials'`, qty, playerID, matID)
	}
	return err
}

func addMaterialTx(tx *sql.Tx, playerID uuid.UUID, matID string, qty int64) error {
	_, err := tx.Exec(`
		INSERT INTO player_inventory_items (player_id, item_id, category, quantity)
		VALUES ($1, $2, 'materials', $3)
		ON CONFLICT (player_id, item_id, category) DO UPDATE SET quantity = player_inventory_items.quantity + EXCLUDED.quantity`,
		playerID, matID, qty,
	)
	return err
}
