package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type CreateListingRequest struct {
	ItemType            string  `json:"item_type" binding:"required"` // Gu|Material|Recipe
	GuInstanceID        string  `json:"gu_instance_id"`
	ItemID              string  `json:"item_id"`
	Quantity            int     `json:"quantity"`
	Price               float64 `json:"price" binding:"required"`
	Currency            string  `json:"currency" binding:"required"` // primeval|immortal
	DurationHours       int     `json:"duration_hours" binding:"required"`
	ListingType         string  `json:"listing_type" binding:"required"` // MARKET|AUCTION|TRADE
	TargetBuyerUsername string  `json:"target_buyer_username"`
}

type PlaceBidRequest struct {
	ListingID string  `json:"listing_id" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
}

type BuyListingRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
	Quantity  int    `json:"quantity"`
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
	if req.ItemType != "Gu" && req.ItemType != "Material" && req.ItemType != "Recipe" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_type phải là Gu, Material hoặc Recipe"})
		return
	}
	if req.ListingType != "MARKET" && req.ListingType != "AUCTION" && req.ListingType != "TRADE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "listing_type phải là MARKET, AUCTION hoặc TRADE"})
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

	if player.Rank < 6 && req.Currency == "immortal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ sư dưới Lục Chuyển không thể giao dịch bằng Tiên Nguyên Thạch"})
		return
	}

	var targetBuyerID uuid.UUID
	if req.ListingType == "TRADE" {
		if req.TargetBuyerUsername == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Giao dịch Bảo Hoàng Thiên yêu cầu phải chỉ định người nhận (TargetBuyerUsername)"})
			return
		}
		hasConnectingHeaven := false
		for _, gu := range player.GuInventory {
			if gu.GuID == "GU_CONNECTING_HEAVEN" {
				hasConnectingHeaven = true
				break
			}
		}
		if !hasConnectingHeaven {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Giao dịch trực tiếp yêu cầu cần có Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN)!"})
			return
		}

		err = tx.QueryRow(`SELECT id FROM players WHERE username = $1`, req.TargetBuyerUsername).Scan(&targetBuyerID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy người nhận chỉ định"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn cơ sở dữ liệu"})
			}
			return
		}
		if targetBuyerID == player.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tự giao dịch với bản thân"})
			return
		}
	}

	// Calculate and charge upfront listing fee (always in Primeval Stones)
	basicFee := 5
	if req.ListingType == "AUCTION" {
		basicFee = 20
	} else if req.ListingType == "TRADE" {
		basicFee = 0
	}
	if player.PrimevalStones < basicFee {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch để chi trả phí đăng bán cơ bản (cần %d Nguyên Thạch)", basicFee)})
		return
	}
	player.PrimevalStones -= basicFee

	var itemDataJSON []byte
	if req.ItemType == "Recipe" {
		idx := -1
		for i, ur := range player.UnlockedRecipes {
			if ur.RecipeID == req.ItemID {
				idx = i
				break
			}
		}
		if idx == -1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn chưa khai mở phương thức này để bán"})
			return
		}
		recipeToSell := player.UnlockedRecipes[idx]
		
		// Remove recipe from seller's unlocked list
		player.UnlockedRecipes = append(player.UnlockedRecipes[:idx], player.UnlockedRecipes[idx+1:]...)
		req.Quantity = 1 // Recipes are single instance
		
		itemDataJSON, err = json.Marshal(recipeToSell)
		if err != nil {
			log.Printf("Failed to marshal recipe: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đăng bán phương thức"})
			return
		}
	} else if req.ItemType == "Gu" {
		idx := -1
		for i, gu := range player.GuInventory {
			if gu.InstanceID == req.GuInstanceID {
				idx = i
				break
			}
		}
		if idx == -1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Cổ trùng để đăng bán"})
			return
		}
		if player.GuInventory[idx].IsVital {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đấu giá Bản Mệnh Cổ"})
			return
		}
		itemData := player.GuInventory[idx].GuID
		itemDataJSON, err = json.Marshal(itemData)
		if err != nil {
			log.Printf("Failed to marshal Gu: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đăng bán Cổ trùng"})
			return
		}
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
		itemData := req.ItemID
		itemDataJSON, err = json.Marshal(itemData)
		if err != nil {
			log.Printf("Failed to marshal material: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đăng bán vật phẩm"})
			return
		}
	}

	listingID := uuid.New().String()
	end := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
	var targetBuyerParam interface{}
	if req.ListingType == "TRADE" {
		targetBuyerParam = targetBuyerID
	} else {
		targetBuyerParam = nil
	}
	_, err = tx.Exec(`INSERT INTO auction(id,seller_id,item_type,item_data,quantity,currency,start_price,current_price,highest_bidder_id,end_time,status,listing_type,created_at,target_buyer_id) VALUES($1,$2,$3,$4,$5,$6,$7,$7,NULL,$8,'ACTIVE',$9,NOW(),$10)`, listingID, player.ID, req.ItemType, itemDataJSON, req.Quantity, req.Currency, req.Price, end, req.ListingType, targetBuyerParam)
	if err != nil {
		log.Printf("Failed to create auction listing: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo phiên giao dịch"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET gu_inventory=$1, materials_inventory=$2, equipped_gus=$3, unlocked_recipes=$4, primeval_stones=$5 WHERE id=$6`, player.GuInventory, player.MaterialsInventory, player.EquippedGus, player.UnlockedRecipes, player.PrimevalStones, player.ID)
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
	if err := settleExpiredAuctions(); err != nil {
		log.Printf("Failed to settle expired auctions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to settle expired auctions"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	var rows *sql.Rows
	var err error
	if ok {
		rows, err = db.DB.Query(`
			SELECT 
				a.id, a.seller_id, a.item_type, a.item_data, a.quantity, a.currency, 
				a.start_price, a.current_price, a.highest_bidder_id, a.end_time, 
				a.status, a.listing_type, a.created_at, a.target_buyer_id,
				p_buyer.username AS target_buyer_username
			FROM auction a
			LEFT JOIN players p_buyer ON a.target_buyer_id = p_buyer.id
			WHERE a.status='ACTIVE' AND a.end_time > NOW() 
			  AND (a.target_buyer_id IS NULL OR a.target_buyer_id = $1 OR a.seller_id = $1)
			ORDER BY a.created_at DESC
		`, authID)
	} else {
		rows, err = db.DB.Query(`
			SELECT 
				a.id, a.seller_id, a.item_type, a.item_data, a.quantity, a.currency, 
				a.start_price, a.current_price, a.highest_bidder_id, a.end_time, 
				a.status, a.listing_type, a.created_at, a.target_buyer_id,
				p_buyer.username AS target_buyer_username
			FROM auction a
			LEFT JOIN players p_buyer ON a.target_buyer_id = p_buyer.id
			WHERE a.status='ACTIVE' AND a.end_time > NOW() AND a.target_buyer_id IS NULL
			ORDER BY a.created_at DESC
		`)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query marketplace"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, seller, it, cur, status, lt string
		var qty int
		var start, current float64
		var bidder sql.NullString
		var end, created time.Time
		var rawItemData []byte
		var targetBuyerID sql.NullString
		var targetBuyerUsername sql.NullString
		if err := rows.Scan(&id, &seller, &it, &rawItemData, &qty, &cur, &start, &current, &bidder, &end, &status, &lt, &created, &targetBuyerID, &targetBuyerUsername); err != nil {
			log.Printf("Warning: failed to scan auction row: %v", err)
			continue
		}
		var data interface{}
		if err := json.Unmarshal(rawItemData, &data); err != nil {
			log.Printf("Warning: failed to unmarshal auction item_data: %v", err)
			data = string(rawItemData)
		}
		out = append(out, gin.H{
			"id":                    id,
			"seller_id":             seller,
			"item_type":             it,
			"item_data":             data,
			"quantity":              qty,
			"currency":              cur,
			"start_price":           start,
			"current_price":         current,
			"highest_bidder_id":     bidder.String,
			"end_time":              end,
			"status":                status,
			"listing_type":          lt,
			"created_at":            created,
			"target_buyer_id":       targetBuyerID.String,
			"target_buyer_username": targetBuyerUsername.String,
		})
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
	var currency, status, listingType string
	var current float64
	var bidder sql.NullString
	var end time.Time
	err = tx.QueryRow(`SELECT currency,current_price,highest_bidder_id,status,end_time,listing_type FROM auction WHERE id=$1 FOR UPDATE`, req.ListingID).Scan(&currency, &current, &bidder, &status, &end, &listingType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phiên giao dịch"})
		return
	}
	if listingType != "AUCTION" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ có thể đấu giá vật phẩm trong phòng đấu giá"})
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
	var neededAmount float64 = req.Amount
	if bidder.Valid && bidder.String == player.ID.String() {
		neededAmount = req.Amount - current
	}
	if float64(player.PrimevalStones) < neededAmount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch"})
		return
	}
	player.PrimevalStones -= int(neededAmount)
	if bidder.Valid && bidder.String != player.ID.String() {
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id=$2`, int(current), bidder.String)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tiền cho người đặt giá trước"})
			return
		}
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1 WHERE id=$2`, player.PrimevalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	_, err = tx.Exec(`UPDATE auction SET current_price=$1, highest_bidder_id=$2 WHERE id=$3`, req.Amount, player.ID, req.ListingID)
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

func BuyListing(c *gin.Context) {
	var req BuyListingRequest
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

	buyer, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load buyer details"})
		return
	}

	var sellerID uuid.UUID
	var itemType, currency, status, listingType string
	var currentPrice float64
	var qty int
	var rawItemData []byte
	var targetBuyerID sql.NullString
	err = tx.QueryRow(`SELECT seller_id, item_type, item_data, quantity, currency, current_price, status, listing_type, target_buyer_id FROM auction WHERE id=$1 FOR UPDATE`, req.ListingID).Scan(&sellerID, &itemType, &rawItemData, &qty, &currency, &currentPrice, &status, &listingType, &targetBuyerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phiên giao dịch"})
		return
	}

	if status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phiên giao dịch đã kết thúc hoặc không còn hoạt động"})
		return
	}
	if listingType != "MARKET" && listingType != "TRADE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này thuộc sàn đấu giá, không thể mua trực tiếp"})
		return
	}
	if listingType == "TRADE" {
		if !targetBuyerID.Valid || targetBuyerID.String != buyer.ID.String() {
			c.JSON(http.StatusForbidden, gin.H{"error": "Vật phẩm này được dành riêng cho người chơi khác, bạn không thể mua!"})
			return
		}
		hasConnectingHeaven := false
		for _, gu := range buyer.GuInventory {
			if gu.GuID == "GU_CONNECTING_HEAVEN" {
				hasConnectingHeaven = true
				break
			}
		}
		if !hasConnectingHeaven {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Để thực hiện giao dịch trực tiếp, bạn cần có Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN)!"})
			return
		}
	}
	if sellerID == authID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thể tự mua vật phẩm của chính mình"})
		return
	}

	// Validate buy quantity
	purchaseQty := req.Quantity
	if purchaseQty <= 0 {
		purchaseQty = qty // Default to buying the full stack if quantity is omitted or <= 0
	}
	if purchaseQty > qty {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Số lượng yêu cầu (%d) vượt quá số lượng hiện có (%d)", purchaseQty, qty)})
		return
	}
	if (itemType == "Gu" || itemType == "Recipe") && purchaseQty != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể mua lẻ Cổ trùng hoặc Bí phương"})
		return
	}

	totalPrice := float64(purchaseQty) * currentPrice
	if currency == "primeval" {
		if float64(buyer.PrimevalStones) < totalPrice {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch"})
			return
		}
		buyer.PrimevalStones -= int(totalPrice)
	} else {
		if buyer.ImmortalStones < totalPrice {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Nguyên Thạch"})
			return
		}
		buyer.ImmortalStones -= totalPrice
	}

	seller, err := evaluateManaTx(tx, sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load seller details"})
		return
	}

	if listingType == "TRADE" {
		sellerConnectingHeaven := false
		for _, gu := range seller.GuInventory {
			if gu.GuID == "GU_CONNECTING_HEAVEN" {
				sellerConnectingHeaven = true
				break
			}
		}
		if !sellerConnectingHeaven {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Người bán không còn sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để kết nối Bảo Hoàng Thiên, giao dịch thất bại!"})
			return
		}
	}

	// 2% commission fee for MARKET, 0% for TRADE
	commission := totalPrice * 0.02
	if listingType == "TRADE" {
		commission = 0.0
	}
	payout := totalPrice - commission

	if currency == "primeval" {
		seller.PrimevalStones += int(payout)
	} else {
		seller.ImmortalStones += payout
	}

	var itemData string
	if err := json.Unmarshal(rawItemData, &itemData); err != nil {
		itemData = string(rawItemData)
	}

	if err := addAuctionItemToPlayer(&buyer, itemType, itemData, purchaseQty); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển giao vật phẩm"})
		return
	}

	if err := updatePlayerInventory(tx, buyer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật kho đồ người mua"})
		return
	}

	if err := updatePlayerInventory(tx, seller); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật tài sản người bán"})
		return
	}

	if purchaseQty == qty {
		_, err = tx.Exec(`UPDATE auction SET status='COMPLETED' WHERE id=$1`, req.ListingID)
	} else {
		_, err = tx.Exec(`UPDATE auction SET quantity=quantity-$1 WHERE id=$2`, purchaseQty, req.ListingID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái đơn hàng"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete purchase"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mua hàng thành công", "player": buyer})
}

func SettleExpiredAuctions(c *gin.Context) {
	if err := settleExpiredAuctions(); err != nil {
		log.Printf("Failed to settle expired auctions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to settle expired auctions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired auctions settled"})
}

func settleExpiredAuctions() error {
	rows, err := db.DB.Query(`SELECT id,seller_id,highest_bidder_id,item_type,item_data,quantity,current_price,currency,listing_type FROM auction WHERE status='ACTIVE' AND end_time <= NOW()`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var sellerID uuid.UUID
		var highestBidderID sql.NullString
		var itemType string
		var rawItemData []byte
		var qty int
		var currentPrice float64
		var currency, listingType string
		if err := rows.Scan(&id, &sellerID, &highestBidderID, &itemType, &rawItemData, &qty, &currentPrice, &currency, &listingType); err != nil {
			log.Printf("Failed to scan expired auction row: %v", err)
			continue
		}

		var itemData string
		if err := json.Unmarshal(rawItemData, &itemData); err != nil {
			itemData = string(rawItemData)
		}

		if err := settleAuction(id, sellerID, highestBidderID, itemType, itemData, qty, currentPrice, currency, listingType); err != nil {
			log.Printf("Failed to settle auction %s: %v", id, err)
			continue
		}
	}

	return rows.Err()
}

func settleAuction(auctionID string, sellerID uuid.UUID, highestBidderID sql.NullString, itemType, itemData string, qty int, currentPrice float64, currency string, listingType string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	seller, err := evaluateManaTx(tx, sellerID)
	if err != nil {
		return err
	}

	if highestBidderID.Valid {
		buyerID, err := uuid.Parse(highestBidderID.String)
		if err != nil {
			return err
		}

		buyer, err := evaluateManaTx(tx, buyerID)
		if err != nil {
			return err
		}

		// 5% commission fee for AUCTION
		commission := currentPrice * 0.05
		payout := currentPrice - commission

		if currency == "primeval" {
			seller.PrimevalStones += int(payout)
		} else {
			seller.ImmortalStones += payout
		}

		if err := addAuctionItemToPlayer(&buyer, itemType, itemData, qty); err != nil {
			return err
		}

		if err := updatePlayerInventory(tx, buyer); err != nil {
			return err
		}
		if err := updatePlayerInventory(tx, seller); err != nil {
			return err
		}

		if _, err := tx.Exec(`UPDATE auction SET status='COMPLETED' WHERE id=$1`, auctionID); err != nil {
			return err
		}
	} else {
		if err := addAuctionItemToPlayer(&seller, itemType, itemData, qty); err != nil {
			return err
		}
		if err := updatePlayerInventory(tx, seller); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE auction SET status='CANCELLED' WHERE id=$1`, auctionID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func addAuctionItemToPlayer(player *models.Player, itemType, itemData string, qty int) error {
	if itemType == "Gu" {
		player.GuInventory = append(player.GuInventory, mapGuFromRaw(itemData))
		return nil
	}

	if itemType == "Recipe" {
		var recipeInfo models.UnlockedRecipe
		if err := json.Unmarshal([]byte(itemData), &recipeInfo); err == nil {
			found := false
			for i, ur := range player.UnlockedRecipes {
				if ur.RecipeID == recipeInfo.RecipeID {
					player.UnlockedRecipes[i].CompletionRate += recipeInfo.CompletionRate
					if player.UnlockedRecipes[i].CompletionRate > 100.0 {
						player.UnlockedRecipes[i].CompletionRate = 100.0
					}
					found = true
					break
				}
			}
			if !found {
				player.UnlockedRecipes = append(player.UnlockedRecipes, recipeInfo)
			}
		}
		return nil
	}

	if player.MaterialsInventory == nil {
		player.MaterialsInventory = map[string]int{}
	}
	player.MaterialsInventory[itemData] += qty
	return nil
}

func updatePlayerInventory(tx *sql.Tx, player models.Player) error {
	_, err := tx.Exec(`UPDATE players SET gu_inventory=$1, materials_inventory=$2, equipped_gus=$3, primeval_stones=$4, immortal_stones=$5, unlocked_recipes=$6 WHERE id=$7`, player.GuInventory, player.MaterialsInventory, player.EquippedGus, player.PrimevalStones, player.ImmortalStones, player.UnlockedRecipes, player.ID)
	return err
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
	var sellerID uuid.UUID
	var itemType, status string
	var rawItemData []byte
	var qty int
	err = tx.QueryRow(`SELECT seller_id,item_type,item_data,quantity,status FROM auction WHERE id=$1 FOR UPDATE`, req.ListingID).Scan(&sellerID, &itemType, &rawItemData, &qty, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy listing"})
		return
	}
	if sellerID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải chủ listing"})
		return
	}
	itemData := ""
	if err := json.Unmarshal(rawItemData, &itemData); err != nil {
		itemData = string(rawItemData)
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
	if err := addAuctionItemToPlayer(&player, itemType, itemData, qty); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn trả vật phẩm"})
		return
	}
	if err := updatePlayerInventory(tx, player); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
		return
	}
	_, err = tx.Exec(`UPDATE auction SET status='CANCELLED' WHERE id=$1`, req.ListingID)
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
