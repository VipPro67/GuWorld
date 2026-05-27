package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type CreateListingRequest struct {
	ItemType      string  `json:"item_type" binding:"required"` // Gu|Material
	GuInstanceID  string  `json:"gu_instance_id"`
	ItemID        string  `json:"item_id"`
	Quantity      int     `json:"quantity"`
	Price         float64 `json:"price" binding:"required"`
	Currency      string  `json:"currency" binding:"required"` // primeval|immortal
	DurationHours int     `json:"duration_hours" binding:"required"`
}

type PlaceBidRequest struct {
	ListingID string  `json:"listing_id" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
}

func CreateListing(c *gin.Context) {
	var req CreateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Price <= 0 || req.DurationHours < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Giá và thời gian phải hợp lệ"})
		return
	}
	if req.Currency != "primeval" && req.Currency != "immortal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "currency phải là primeval hoặc immortal"})
		return
	}
	if req.ItemType != "Gu" && req.ItemType != "Material" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_type phải là Gu hoặc Material"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}
	if player.Rank < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần đạt Lục Chuyển để dùng Bảo Hoàng Thiên"})
		return
	}

	itemData := ""
	if req.ItemType == "Gu" {
		idx := -1
		for i, gu := range player.GuInventory {
			if gu.InstanceID == req.GuInstanceID {
				idx = i
				break
			}
		}
		if idx == -1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Cổ để đăng bán"})
			return
		}
		if player.GuInventory[idx].IsVital {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đấu giá Bản Mệnh Cổ"})
			return
		}
		itemData = player.GuInventory[idx].GuID
		newInv := player.GuInventory[:0]
		for i, gu := range player.GuInventory {
			if i != idx {
				newInv = append(newInv, gu)
			}
		}
		player.GuInventory = newInv
	} else {
		if req.Quantity <= 0 {
			req.Quantity = 1
		}
		qty := player.MaterialsInventory[req.ItemID]
		if qty < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ vật phẩm"})
			return
		}
		player.MaterialsInventory[req.ItemID] -= req.Quantity
		if player.MaterialsInventory[req.ItemID] <= 0 {
			delete(player.MaterialsInventory, req.ItemID)
		}
		itemData = req.ItemID
	}
	listingID := uuid.New().String()
	end := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
	_, err = tx.Exec(`INSERT INTO bao_hoang_listings(id,seller_id,item_type,item_data,quantity,currency,start_price,current_price,highest_bidder_id,end_time,status,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$7,NULL,$8,'ACTIVE',NOW())`, listingID, player.ID, req.ItemType, itemData, req.Quantity, req.Currency, req.Price, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo phiên đấu giá"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET gu_inventory=$1, materials_inventory=$2, equipped_gus=$3 WHERE id=$4`, player.GuInventory, player.MaterialsInventory, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đăng bán thành công", "listing_id": listingID})
}

func ListListings(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id,seller_id,item_type,item_data,quantity,currency,start_price,current_price,highest_bidder_id,end_time,status,created_at FROM bao_hoang_listings WHERE status='ACTIVE' AND end_time > NOW() ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, seller, it, data, cur, status string
		var qty int
		var start, current float64
		var bidder sql.NullString
		var end, created time.Time
		_ = rows.Scan(&id, &seller, &it, &data, &qty, &cur, &start, &current, &bidder, &end, &status, &created)
		out = append(out, gin.H{"id": id, "seller_id": seller, "item_type": it, "item_data": data, "quantity": qty, "currency": cur, "start_price": start, "current_price": current, "highest_bidder_id": bidder.String, "end_time": end, "status": status, "created_at": created})
	}
	c.JSON(http.StatusOK, out)
}

func PlaceBid(c *gin.Context) {
	var req PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}
	var currency, status string
	var current float64
	var bidder sql.NullString
	var end time.Time
	err = tx.QueryRow(`SELECT currency,current_price,highest_bidder_id,status,end_time FROM bao_hoang_listings WHERE id=$1 FOR UPDATE`, req.ListingID).Scan(&currency, &current, &bidder, &status, &end)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phiên đấu giá"})
		return
	}
	if status != "ACTIVE" || time.Now().After(end) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phiên đấu giá đã kết thúc"})
		return
	}
	if currency != "primeval" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đấu giá chỉ dùng Nguyên Thạch"})
		return
	}
	if req.Amount <= current {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Giá phải cao hơn %.2f", current)})
		return
	}
	if float64(player.PrimevalStones) < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch"})
		return
	}
	player.PrimevalStones -= int(req.Amount)
	if bidder.Valid {
		_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id=$2`, int(current), bidder.String)
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1 WHERE id=$2`, player.PrimevalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	_, err = tx.Exec(`UPDATE bao_hoang_listings SET current_price=$1, highest_bidder_id=$2 WHERE id=$3`, req.Amount, player.ID, req.ListingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bid failed"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đặt giá thành công"})
}

type CancelListingRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
}

func CancelListing(c *gin.Context) {
	var req CancelListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	var sellerID, itemType, itemData, status string
	var qty int
	err = tx.QueryRow(`SELECT seller_id,item_type,item_data,quantity,status FROM bao_hoang_listings WHERE id=$1 FOR UPDATE`, req.ListingID).Scan(&sellerID, &itemType, &itemData, &qty, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy listing"})
		return
	}
	if sellerID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải chủ listing"})
		return
	}
	if status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Listing không còn hoạt động"})
		return
	}
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}
	if itemType == "Gu" {
		player.GuInventory = append(player.GuInventory, mapGuFromRaw(itemData))
	} else {
		if player.MaterialsInventory == nil {
			player.MaterialsInventory = map[string]int{}
		}
		player.MaterialsInventory[itemData] += qty
	}
	_, err = tx.Exec(`UPDATE players SET gu_inventory=$1, materials_inventory=$2, equipped_gus=$3 WHERE id=$4`, player.GuInventory, player.MaterialsInventory, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trả vật phẩm"})
		return
	}
	_, err = tx.Exec(`UPDATE bao_hoang_listings SET status='CANCELLED' WHERE id=$1`, req.ListingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hủy listing"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy listing và hoàn trả vật phẩm"})
}

func mapGuFromRaw(guID string) models.GuInstance {
	return models.GuInstance{InstanceID: uuid.New().String(), GuID: guID, Hunger: 100, Status: "SLEEPING", Equipped: false, IsVital: false}
}
