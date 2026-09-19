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
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type CreateBuybackOrderRequest struct {
	ItemType          string          `json:"item_type" binding:"required"` // Gu|Material|Recipe
	ItemTarget        json.RawMessage `json:"item_target" binding:"required"`
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
	ItemType          string          `json:"item_type"`
	ItemTarget        json.RawMessage `json:"item_target"`
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
	rows, err := q.Query(`SELECT item_id, quantity FROM player_inventory_items WHERE player_id = $1`, playerID)
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

// CreateBuybackOrder creates a new buy order on Bảo Hoàng Thiên order book
func CreateBuybackOrder(c *gin.Context) {
	var req CreateBuybackOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
	if req.DurationHours < 1 || req.DurationHours > 168 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời hạn đơn thu mua từ 1 đến 168 giờ"})
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

	buyer, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load buyer details"})
		return
	}
	if buyer.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể tạo đơn thu mua!"})
		return
	}
	if !HasBaoHoangAccess(buyer.GuInventory) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) và Thần Niệm Cổ (GU_DIVINE_SENSE) để sử dụng Bảo Hoàng Thiên!"})
		return
	}
	if buyer.Rank < 6 && req.Currency == "immortal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ sư dưới Lục Chuyển không thể dùng Tiên Nguyên Thạch"})
		return
	}

	totalItemCost := pricing.Quantize(req.UnitPrice * float64(req.QuantityRequested))
	upfrontFee := pricing.Quantize(totalItemCost * 0.04)
	totalLocked := totalItemCost + upfrontFee

	if req.Currency == "immortal" {
		if buyer.ImmortalStones < totalLocked {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch (Cần %.3f, hiện có %.3f)", totalLocked, buyer.ImmortalStones)})
			return
		}
		_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, totalLocked, buyer.ID)
	} else {
		if buyer.PrimevalStones < totalLocked {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch (Cần %.3f, hiện có %.3f)", totalLocked, buyer.PrimevalStones)})
			return
		}
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2`, totalLocked, buyer.ID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ tài sản"})
		return
	}

	endTime := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
	var orderID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO buyback_orders (buyer_id, item_type, item_target, unit_price, quantity_requested, quantity_filled, currency, upfront_fee, total_locked, status, duration_hours, end_time)
		VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8, 'ACTIVE', $9, $10)
		RETURNING id`,
		buyer.ID, req.ItemType, req.ItemTarget, req.UnitPrice, req.QuantityRequested, req.Currency, upfrontFee, totalLocked, req.DurationHours, endTime,
	).Scan(&orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo đơn thu mua"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit transaction error"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &buyer.ID, Action: "marketplace.create_buyback",
		EntityType: "buyback_order", EntityID: orderID.String(),
		RequestSummary: fmt.Sprintf("Tạo đơn thu mua %d %s (giá %.3f)", req.QuantityRequested, req.ItemType, req.UnitPrice),
		IPAddress: c.ClientIP(), Success: true,
	})

	tx2, _ := db.DB.Begin()
	updatedPlayer, _ := evaluateManaTx(tx2, buyer.ID)
	_ = tx2.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Tạo đơn thu mua thành công trên Bảo Hoàng Thiên!",
		"order_id": orderID.String(),
		"player":   updatedPlayer,
	})
}

// GetBuybackOrders lists active buyback orders
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
		SELECT b.id, b.buyer_id, p.username, b.item_type, b.item_target, b.unit_price, b.quantity_requested, b.quantity_filled, b.currency, b.upfront_fee, b.total_locked, b.status, b.duration_hours, b.end_time, b.created_at
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
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.BuyerUsername, &o.ItemType, &o.ItemTarget, &o.UnitPrice, &o.QuantityRequested, &o.QuantityFilled, &o.Currency, &o.UpfrontFee, &o.TotalLocked, &o.Status, &o.DurationHours, &o.EndTime, &o.CreatedAt); err == nil {
			res = append(res, o)
		}
	}
	c.JSON(http.StatusOK, res)
}

// SellToBuybackOrder executes matching sell against an active buyback order
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

	seller, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load seller details"})
		return
	}
	if seller.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể bán vật phẩm!"})
		return
	}
	if !HasBaoHoangAccess(seller.GuInventory) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ và Thần Niệm Cổ để giao dịch Bảo Hoàng Thiên!"})
		return
	}

	var o BuybackOrderResponse
	err = tx.QueryRow(`
		SELECT b.id, b.buyer_id, p.username, b.item_type, b.item_target, b.unit_price, b.quantity_requested, b.quantity_filled, b.currency, b.upfront_fee, b.total_locked, b.status, b.duration_hours, b.end_time, b.created_at
		FROM buyback_orders b
		JOIN players p ON p.id = b.buyer_id
		WHERE b.id = $1 FOR UPDATE`, orderUUID,
	).Scan(&o.ID, &o.BuyerID, &o.BuyerUsername, &o.ItemType, &o.ItemTarget, &o.UnitPrice, &o.QuantityRequested, &o.QuantityFilled, &o.Currency, &o.UpfrontFee, &o.TotalLocked, &o.Status, &o.DurationHours, &o.EndTime, &o.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Đơn thu mua không tồn tại"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Query error"})
		}
		return
	}

	if o.Status != "ACTIVE" || time.Now().After(o.EndTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn thu mua đã hết hạn hoặc không còn hoạt động"})
		return
	}

	if o.BuyerID == seller.ID.String() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể bán vào đơn thu mua của chính mình"})
		return
	}

	// Verify buyer dual-gu access
	buyerUUID := uuid.MustParse(o.BuyerID)
	buyer, err := evaluateManaTx(tx, buyerUUID)
	if err != nil || !HasBaoHoangAccess(buyer.GuInventory) {
		// Refund remaining funds and cancel order
		unfilled := o.QuantityRequested - o.QuantityFilled
		refundAmt := pricing.Quantize(float64(unfilled) * o.UnitPrice)
		if o.Currency == "immortal" {
			_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, refundAmt, buyerUUID)
		} else {
			_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, refundAmt, buyerUUID)
		}
		_, _ = tx.Exec(`UPDATE buyback_orders SET status = 'CANCELLED' WHERE id = $1`, orderUUID)
		_ = tx.Commit()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người mua không còn đủ điều kiện kết nối Bảo Hoàng Thiên. Đơn thu mua đã bị hủy và hoàn trả tiền."})
		return
	}

	remainingRequested := o.QuantityRequested - o.QuantityFilled
	sellQty := req.Quantity
	if sellQty <= 0 {
		sellQty = 1
	}
	if sellQty > remainingRequested {
		sellQty = remainingRequested
	}

	// Transfer item from seller to buyer
	if o.ItemType == "Material" {
		var targetMat string
		_ = json.Unmarshal(o.ItemTarget, &targetMat)
		if targetMat == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ItemTarget nguyên liệu không hợp lệ"})
			return
		}
		sellerMats := getPlayerMaterialsMap(tx, seller.ID)
		sellerMatQty := sellerMats[targetMat]
		if sellerMatQty < sellQty {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn không đủ nguyên liệu %s (Cần %d, hiện có %d)", targetMat, sellQty, sellerMatQty)})
			return
		}
		if err := deductMaterialTx(tx, seller.ID, targetMat, sellQty); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct material from seller"})
			return
		}
		if err := addMaterialTx(tx, buyer.ID, targetMat, sellQty); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add material to buyer"})
			return
		}
	} else if o.ItemType == "Gu" {
		var guInstIDs []string
		if len(req.GuInstanceIDs) > 0 {
			guInstIDs = req.GuInstanceIDs
		} else if req.GuInstanceID != "" {
			guInstIDs = []string{req.GuInstanceID}
		}
		if int64(len(guInstIDs)) != sellQty {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần chỉ định đúng %d Cổ trùng để bán", sellQty)})
			return
		}
		for _, instID := range guInstIDs {
			var ownerID uuid.UUID
			var isVital bool
			err := tx.QueryRow(`SELECT player_id, is_vital FROM player_gu WHERE id = $1`, instID).Scan(&ownerID, &isVital)
			if err != nil || ownerID != seller.ID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không sở hữu Cổ trùng này"})
				return
			}
			if isVital {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể bán Bổn Mạng Cổ!"})
				return
			}
			_, err = tx.Exec(`UPDATE player_gu SET player_id = $1, equipped = false WHERE id = $2`, buyer.ID, instID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi chuyển nhượng Cổ trùng"})
				return
			}
		}
	} else if o.ItemType == "Recipe" {
		var recTarget string
		_ = json.Unmarshal(o.ItemTarget, &recTarget)
		var hasRec bool
		_ = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM player_recipe_progress WHERE player_id = $1 AND recipe_id = $2)`, seller.ID, recTarget).Scan(&hasRec)
		if !hasRec {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không sở hữu phương thức phối chế này"})
			return
		}
		var compRate float64
		_ = tx.QueryRow(`SELECT completion_rate FROM player_recipe_progress WHERE player_id = $1 AND recipe_id = $2`, seller.ID, recTarget).Scan(&compRate)
		_, _ = tx.Exec(`DELETE FROM player_recipe_progress WHERE player_id = $1 AND recipe_id = $2`, seller.ID, recTarget)
		_, _ = tx.Exec(`INSERT INTO player_recipe_progress (player_id, recipe_id, completion_rate) VALUES ($1, $2, $3) ON CONFLICT (player_id, recipe_id) DO UPDATE SET completion_rate = GREATEST(player_recipe_progress.completion_rate, EXCLUDED.completion_rate)`, buyer.ID, recTarget, compRate)
	}

	// Calculate Payout & Commission (6% on filled portion)
	totalGross := pricing.Quantize(o.UnitPrice * float64(sellQty))
	commission := pricing.Quantize(totalGross * 0.06)
	sellerPayout := totalGross - commission

	if o.Currency == "immortal" {
		_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, sellerPayout, seller.ID)
	} else {
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, sellerPayout, seller.ID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thanh toán tiền cho người bán"})
		return
	}

	newFilled := o.QuantityFilled + sellQty
	newStatus := "ACTIVE"
	if newFilled >= o.QuantityRequested {
		newStatus = "FILLED"
	}
	_, err = tx.Exec(`UPDATE buyback_orders SET quantity_filled = $1, status = $2 WHERE id = $3`, newFilled, newStatus, orderUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái đơn thu mua"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit error"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &seller.ID, Action: "marketplace.sell_buyback",
		EntityType: "buyback_order", EntityID: orderUUID.String(),
		RequestSummary: fmt.Sprintf("Bán %d vật phẩm vào đơn thu mua (nhận %.3f)", sellQty, sellerPayout),
		IPAddress: c.ClientIP(), Success: true,
	})

	tx2, _ := db.DB.Begin()
	updatedSeller, _ := evaluateManaTx(tx2, seller.ID)
	_ = tx2.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Bán thành công %d vật phẩm vào đơn thu mua! Nhận %.3f %s (Phí hoa hồng 6%%: %.3f).", sellQty, sellerPayout, o.Currency, commission),
		"player":  updatedSeller,
	})
}

// CancelBuybackOrder cancels an active buyback order and refunds remaining locked funds
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

	buyer, err := evaluateManaTx(tx, authID)
	if err != nil || buyer.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể hủy đơn!"})
		return
	}

	var buyerID uuid.UUID
	var unitPrice float64
	var qtyRequested, qtyFilled int64
	var currency, status string

	err = tx.QueryRow(`
		SELECT buyer_id, unit_price, quantity_requested, quantity_filled, currency, status
		FROM buyback_orders WHERE id = $1 FOR UPDATE`, orderUUID,
	).Scan(&buyerID, &unitPrice, &qtyRequested, &qtyFilled, &currency, &status)

	if err != nil || buyerID != buyer.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn thu mua không tồn tại hoặc bạn không có quyền hủy"})
		return
	}
	if status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn thu mua không còn ở trạng thái ACTIVE"})
		return
	}

	unfilledQty := qtyRequested - qtyFilled
	if unfilledQty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn thu mua đã được khớp hoàn toàn"})
		return
	}

	refundAmount := pricing.Quantize(float64(unfilledQty) * unitPrice)
	if currency == "immortal" {
		_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, refundAmount, buyer.ID)
	} else {
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, refundAmount, buyer.ID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hoàn tiền cọc"})
		return
	}

	_, err = tx.Exec(`UPDATE buyback_orders SET status = 'CANCELLED' WHERE id = $1`, orderUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái đơn"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit error"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &buyer.ID, Action: "marketplace.cancel_buyback",
		EntityType: "buyback_order", EntityID: orderUUID.String(),
		RequestSummary: fmt.Sprintf("Hủy đơn thu mua, hoàn tiền cọc %.3f %s", refundAmount, currency),
		IPAddress: c.ClientIP(), Success: true,
	})

	tx2, _ := db.DB.Begin()
	updatedBuyer, _ := evaluateManaTx(tx2, buyer.ID)
	_ = tx2.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Hủy đơn thu mua thành công! Nhận lại %.3f %s cọc chưa khớp.", refundAmount, currency),
		"player":  updatedBuyer,
	})
}

// Helpers for material deduction & addition in transactions
func deductMaterialTx(tx *sql.Tx, playerID uuid.UUID, matID string, qty int64) error {
	var currentQty int64
	err := tx.QueryRow(`SELECT quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2 FOR UPDATE`, playerID, matID).Scan(&currentQty)
	if err != nil || currentQty < qty {
		return fmt.Errorf("insufficient material %s", matID)
	}
	if currentQty == qty {
		_, err = tx.Exec(`DELETE FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, playerID, matID)
	} else {
		_, err = tx.Exec(`UPDATE player_inventory_items SET quantity = quantity - $1 WHERE player_id = $2 AND item_id = $3`, qty, playerID, matID)
	}
	return err
}

func addMaterialTx(tx *sql.Tx, playerID uuid.UUID, matID string, qty int64) error {
	_, err := tx.Exec(`
		INSERT INTO player_inventory_items (player_id, item_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (player_id, item_id) DO UPDATE SET quantity = player_inventory_items.quantity + EXCLUDED.quantity`,
		playerID, matID, qty,
	)
	return err
}
