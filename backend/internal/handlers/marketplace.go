package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type CreateListingRequest struct {
	ItemType            string   `json:"item_type" binding:"required"` // Gu|Material|Recipe
	GuInstanceID        string   `json:"gu_instance_id"`
	GuInstanceIDs       []string `json:"gu_instance_ids"`
	ItemID              string   `json:"item_id"`
	Quantity            int      `json:"quantity"`
	Price               float64  `json:"price" binding:"required"`
	Currency            string   `json:"currency" binding:"required"` // primeval|immortal
	DurationHours       int      `json:"duration_hours" binding:"required"`
	ListingType         string   `json:"listing_type" binding:"required"` // MARKET|AUCTION|TRADE
	TargetBuyerUsername string   `json:"target_buyer_username"`
}

type PlaceBidRequest struct {
	ListingID string  `json:"listing_id" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
}

type BuyListingRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
	Quantity  int    `json:"quantity"`
}

// Duration-based fee tiers
func getFeeTier(durationHours int) int {
	switch {
	case durationHours <= 24:
		return 1
	case durationHours <= 72:
		return 2
	case durationHours <= 168:
		return 3
	default:
		return 4
	}
}

func getUpfrontFee(listingType string, durationHours int) int {
	tier := getFeeTier(durationHours)
	if listingType == "AUCTION" {
		switch tier {
		case 1:
			return 20
		case 2:
			return 40
		case 3:
			return 80
		default:
			return 200
		}
	}
	// MARKET
	switch tier {
	case 1:
		return 5
	case 2:
		return 10
	case 3:
		return 20
	default:
		return 50
	}
}

func getCommissionPct(listingType string, durationHours int) float64 {
	tier := getFeeTier(durationHours)
	if listingType == "AUCTION" {
		switch tier {
		case 1:
			return 0.05
		case 2:
			return 0.06
		case 3:
			return 0.08
		default:
			return 0.12
		}
	}
	// MARKET
	switch tier {
	case 1:
		return 0.02
	case 2:
		return 0.03
	case 3:
		return 0.05
	default:
		return 0.08
	}
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
	upfrontFee := getUpfrontFee(req.ListingType, req.DurationHours)
	if req.ListingType == "TRADE" {
		upfrontFee = 0
	}
	if player.PrimevalStones < upfrontFee {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch để chi trả phí đăng bán cơ bản (cần %d Nguyên Thạch)", upfrontFee)})
		return
	}
	player.PrimevalStones -= upfrontFee

	var itemDataJSON []byte
	if req.ItemType == "Recipe" {
		recipeToSell, err := RecipeSvc.Get(tx, player.ID.String(), req.ItemID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn chưa khai mở phương thức này để bán"})
			return
		}
		// Remove recipe from seller's unlocked list
		RecipeSvc.Delete(tx, player.ID.String(), req.ItemID)
		req.Quantity = 1 // Recipes are single instance
		
		itemDataJSON, err = json.Marshal(recipeToSell)
		if err != nil {
			logger.S.Errorf("Failed to marshal recipe: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đăng bán phương thức"})
			return
		}
	} else if req.ItemType == "Gu" {
		var guIDsToSell []string
		if len(req.GuInstanceIDs) > 0 {
			guIDsToSell = req.GuInstanceIDs
		} else if req.GuInstanceID != "" {
			guIDsToSell = []string{req.GuInstanceID}
		}

		if len(guIDsToSell) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không chỉ định Cổ trùng để đăng bán"})
			return
		}

		// Find and remove all instances from player's inventory
		var commonGuID string
		guMap := make(map[string]bool)
		for _, instID := range guIDsToSell {
			guMap[instID] = true
		}

		var newInventory []models.GuInstance
		var gusToSell []models.GuInstance
		var removedCount int

		for _, gu := range player.GuInventory {
			if guMap[gu.InstanceID] {
				if gu.IsVital {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể bán Bản Mệnh Cổ"})
					return
				}
				if gu.Status == "DEAD" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể bán Cổ trùng đã chết"})
					return
				}
				if commonGuID == "" {
					commonGuID = gu.GuID
				} else if commonGuID != gu.GuID {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Tất cả Cổ trùng trong đơn hàng gộp phải cùng loại"})
					return
				}
				gusToSell = append(gusToSell, gu)
				removedCount++
			} else {
				newInventory = append(newInventory, gu)
			}
		}

		if removedCount != len(guIDsToSell) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Một số Cổ trùng không tìm thấy hoặc không thuộc sở hữu của đạo hữu"})
			return
		}

		player.GuInventory = newInventory
		req.Quantity = len(guIDsToSell)

		itemDataJSON, err = json.Marshal(gusToSell)
		if err != nil {
			logger.S.Errorf("Failed to marshal Gu: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đăng bán Cổ trùng"})
			return
		}
	} else {
		if req.Quantity <= 0 {
			req.Quantity = 1
		}
		qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), req.ItemID, "materials")
		if err != nil || qty < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ vật phẩm"})
			return
		}
		if err := InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, "materials", req.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật vật phẩm"})
			return
		}
		itemData := req.ItemID
		itemDataJSON, err = json.Marshal(itemData)
		if err != nil {
			logger.S.Errorf("Failed to marshal material: %v", err)
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
	_, err = tx.Exec(`INSERT INTO auction(id,seller_id,item_type,item_data,quantity,currency,start_price,current_price,highest_bidder_id,end_time,status,listing_type,created_at,target_buyer_id,duration_hours) VALUES($1,$2,$3,$4,$5,$6,$7,$7,NULL,$8,'ACTIVE',$9,NOW(),$10,$11)`, listingID, player.ID, req.ItemType, itemDataJSON, req.Quantity, req.Currency, req.Price, end, req.ListingType, targetBuyerParam, req.DurationHours)
	if err != nil {
		logger.S.Errorf("Failed to create auction listing: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo phiên giao dịch"})
		return
	}
	AddHumanQi(&player, 5.0)
	_, err = tx.Exec(`UPDATE players SET gu_inventory=$1, equipped_gus=$2, primeval_stones=$3, human_qi=$4 WHERE id=$5`, player.GuInventory, player.EquippedGus, player.PrimevalStones, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Trigger System Chat Notification on Bảo Hoàng Thiên channel
	if req.ListingType == "MARKET" || req.ListingType == "AUCTION" {
		itemName := GetItemName(req.ItemType, itemDataJSON)
		currencyName := "Nguyên Thạch"
		if req.Currency == "immortal" {
			currencyName = "Tiên Nguyên Thạch"
		}
		var msgContent string
		if req.ListingType == "AUCTION" {
			msgContent = fmt.Sprintf("Đạo hữu %s vừa đưa lên đấu giá %d %s với mức giá khởi điểm là %.0f %s tại Bảo Hoàng Thiên!", player.Username, req.Quantity, itemName, req.Price, currencyName)
		} else {
			msgContent = fmt.Sprintf("Đạo hữu %s vừa treo bán %d %s với giá %.0f %s tại Bảo Hoàng Thiên!", player.Username, req.Quantity, itemName, req.Price, currencyName)
		}
		_ = SendSystemMessage("general", sql.NullString{}, msgContent)
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.create_listing", EntityType: "auction",
		EntityID: listingID,
		RequestSummary: fmt.Sprintf("%s | %s | sl=%d | %.0f %s | loại=%s", req.ItemType, req.ItemID, req.Quantity, req.Price, req.Currency, req.ListingType),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đăng bán thành công", "listing_id": listingID})
}

func ListListings(c *gin.Context) {
	if err := settleExpiredAuctions(); err != nil {
		logger.S.Errorf("Failed to settle expired auctions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to settle expired auctions"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	cacheKey := "market_listings_public"
	if ok {
		cacheKey = "market_listings_user_" + authID.String()
	}

	var cachedOut []gin.H
	if err := cache.GetJSON(cacheKey, &cachedOut); err == nil {
		c.JSON(http.StatusOK, cachedOut)
		return
	}

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
			logger.S.Warnf("failed to scan auction row: %v", err)
			continue
		}
		var data interface{}
		if err := json.Unmarshal(rawItemData, &data); err != nil {
			logger.S.Warnf("failed to unmarshal auction item_data: %v", err)
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
	_ = cache.SetJSON(cacheKey, out, 10*time.Second)
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
	var sellerID uuid.UUID
	var currency, status, listingType string
	var current float64
	var bidder sql.NullString
	var end time.Time
	err = tx.QueryRow(`SELECT seller_id,currency,current_price,highest_bidder_id,status,end_time,listing_type FROM auction WHERE id=$1 FOR UPDATE`, req.ListingID).Scan(&sellerID, &currency, &current, &bidder, &status, &end, &listingType)
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
	AddHumanQi(&player, 3.0)
	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1, human_qi=$2 WHERE id=$3`, player.PrimevalStones, player.HumanQi, player.ID)
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
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.bid", EntityType: "auction",
		EntityID: req.ListingID,
		RequestSummary: fmt.Sprintf("giá=%.0f | hiện tại=%.0f", req.Amount, current),
		IPAddress: c.ClientIP(), Success: true,
	})
	if sellerID != authID {
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID: &sellerID, Action: "marketplace.bid_notify", EntityType: "auction",
			EntityID: req.ListingID,
			RequestSummary: fmt.Sprintf("người mua=%s | giá=%.0f", authID, req.Amount),
			IPAddress: c.ClientIP(), Success: true,
		})
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
	var qty, durationHours int
	var rawItemData []byte
	var targetBuyerID sql.NullString
	err = tx.QueryRow(`SELECT seller_id, item_type, item_data, quantity, currency, current_price, status, listing_type, target_buyer_id, duration_hours FROM auction WHERE id=$1 FOR UPDATE`, req.ListingID).Scan(&sellerID, &itemType, &rawItemData, &qty, &currency, &currentPrice, &status, &listingType, &targetBuyerID, &durationHours)
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
	if itemType == "Recipe" && purchaseQty != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể mua lẻ Bí phương"})
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

	// Time-based commission fee, 0% for TRADE
	commPct := getCommissionPct(listingType, durationHours)
	if listingType == "TRADE" {
		commPct = 0.0
	}
	commission := totalPrice * commPct
	payout := totalPrice - commission

	if currency == "primeval" {
		seller.PrimevalStones += int(payout)
	} else {
		seller.ImmortalStones += payout
	}

	var itemDataToSend string
	if itemType == "Gu" {
		var gus []models.GuInstance
		if err := json.Unmarshal(rawItemData, &gus); err == nil && len(gus) >= purchaseQty {
			purchasedGus := gus[:purchaseQty]
			purchasedJSON, _ := json.Marshal(purchasedGus)
			itemDataToSend = string(purchasedJSON)
		} else {
			itemDataToSend = string(rawItemData)
		}
	} else {
		var itemData string
		if err := json.Unmarshal(rawItemData, &itemData); err != nil {
			itemData = string(rawItemData)
		}
		itemDataToSend = itemData
	}

	if err := addAuctionItemToPlayer(tx, &buyer, itemType, itemDataToSend, purchaseQty); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển giao vật phẩm"})
		return
	}

	AddHumanQi(&buyer, 5.0)
	AddHumanQi(&seller, 5.0)

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
		if itemType == "Gu" {
			var gus []models.GuInstance
			if err := json.Unmarshal(rawItemData, &gus); err == nil && len(gus) >= purchaseQty {
				remainingGus := gus[purchaseQty:]
				remainingJSON, _ := json.Marshal(remainingGus)
				_, err = tx.Exec(`UPDATE auction SET quantity=quantity-$1, item_data=$2 WHERE id=$3`, purchaseQty, remainingJSON, req.ListingID)
			} else {
				_, err = tx.Exec(`UPDATE auction SET quantity=quantity-$1 WHERE id=$2`, purchaseQty, req.ListingID)
			}
		} else {
			_, err = tx.Exec(`UPDATE auction SET quantity=quantity-$1 WHERE id=$2`, purchaseQty, req.ListingID)
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái đơn hàng"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete purchase"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.buy", EntityType: "auction",
		EntityID: req.ListingID,
		RequestSummary: fmt.Sprintf("seller=%s | sl=%d | %.0f %s", sellerID, purchaseQty, totalPrice, currency),
		IPAddress: c.ClientIP(), Success: true,
	})
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &sellerID, Action: "marketplace.sold", EntityType: "auction",
		EntityID: req.ListingID,
		RequestSummary: fmt.Sprintf("người mua=%s | sl=%d | %.0f %s", authID, purchaseQty, totalPrice, currency),
		IPAddress: c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, buyer, gin.H{"message": "Mua hàng thành công"})
}

func SettleExpiredAuctions(c *gin.Context) {
	if err := settleExpiredAuctions(); err != nil {
		logger.S.Errorf("Failed to settle expired auctions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to settle expired auctions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired auctions settled"})
}

func settleExpiredAuctions() error {
	rows, err := db.DB.Query(`SELECT id,seller_id,highest_bidder_id,item_type,item_data,quantity,current_price,currency,listing_type,duration_hours FROM auction WHERE status='ACTIVE' AND end_time <= NOW()`)
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
		var qty, durationHours int
		var currentPrice float64
		var currency, listingType string
		if err := rows.Scan(&id, &sellerID, &highestBidderID, &itemType, &rawItemData, &qty, &currentPrice, &currency, &listingType, &durationHours); err != nil {
			logger.S.Errorf("Failed to scan expired auction row: %v", err)
			continue
		}

		var itemData string
		if err := json.Unmarshal(rawItemData, &itemData); err != nil {
			itemData = string(rawItemData)
		}

		if err := settleAuction(id, sellerID, highestBidderID, itemType, itemData, qty, currentPrice, currency, listingType, durationHours); err != nil {
			logger.S.Errorf("Failed to settle auction %s: %v", id, err)
			continue
		}
	}

	return rows.Err()
}

func settleAuction(auctionID string, sellerID uuid.UUID, highestBidderID sql.NullString, itemType, itemData string, qty int, currentPrice float64, currency string, listingType string, durationHours int) error {
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

		// Time-based commission fee
		commission := currentPrice * getCommissionPct(listingType, durationHours)
		payout := currentPrice - commission

		if currency == "primeval" {
			seller.PrimevalStones += int(payout)
		} else {
			seller.ImmortalStones += payout
		}

		if err := addAuctionItemToPlayer(tx, &buyer, itemType, itemData, qty); err != nil {
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
		if err := addAuctionItemToPlayer(tx, &seller, itemType, itemData, qty); err != nil {
			return err
		}
		if err := updatePlayerInventory(tx, seller); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE auction SET status='CANCELLED' WHERE id=$1`, auctionID); err != nil {
			return err
		}
	}

	if highestBidderID.Valid {
		bidderUUID, _ := uuid.Parse(highestBidderID.String)
		auditlog.Log(tx, auditlog.Entry{
			PlayerID: &bidderUUID, Action: "marketplace.settle_auction_win", EntityType: "auction",
			EntityID: auctionID,
			RequestSummary: fmt.Sprintf("seller=%s | %.0f %s | sl=%d", sellerID, currentPrice, currency, qty),
			Success: true,
		})
	} else {
		auditlog.Log(tx, auditlog.Entry{
			PlayerID: &sellerID, Action: "marketplace.settle_auction_expired", EntityType: "auction",
			EntityID: auctionID, RequestSummary: "Không ai đặt giá",
			Success: true,
		})
	}
	return tx.Commit()
}

func addAuctionItemToPlayer(tx *sql.Tx, player *models.Player, itemType, itemData string, qty int) error {
	if itemType == "Gu" {
		var gus []models.GuInstance
		if err := json.Unmarshal([]byte(itemData), &gus); err == nil {
			if qty > len(gus) {
				qty = len(gus)
			}
			for i := 0; i < qty; i++ {
				gu := gus[i]
				gu.InstanceID = uuid.New().String() // Tạo ID mới để tránh trùng lặp
				gu.Status = "SLEEPING"
				gu.Equipped = false
				gu.IsVital = false
				player.GuInventory = append(player.GuInventory, gu)
			}
		} else {
			// Tương thích ngược với định dạng cũ (chỉ lưu string guID)
			for i := 0; i < qty; i++ {
				player.GuInventory = append(player.GuInventory, mapGuFromRaw(itemData))
			}
		}
		return nil
	}

	if itemType == "Recipe" {
		var recipeInfo models.UnlockedRecipe
		if err := json.Unmarshal([]byte(itemData), &recipeInfo); err == nil {
			existing, err := RecipeSvc.Get(tx, player.ID.String(), recipeInfo.RecipeID)
			if err == nil {
				completionRate := existing.CompletionRate + recipeInfo.CompletionRate
				if completionRate > 100.0 {
					completionRate = 100.0
				}
				RecipeSvc.Upsert(tx, &services.RecipeProgress{
					PlayerID:       player.ID.String(),
					RecipeID:       recipeInfo.RecipeID,
					CompletionRate: completionRate,
					IsUnlocked:     true,
				})
			} else {
				RecipeSvc.Upsert(tx, &services.RecipeProgress{
					PlayerID:       player.ID.String(),
					RecipeID:       recipeInfo.RecipeID,
					CompletionRate: recipeInfo.CompletionRate,
					IsUnlocked:     true,
				})
			}
		}
		return nil
	}

	InventorySvc.AddItem(tx, player.ID.String(), itemData, "materials", qty)
	return nil
}

func updatePlayerInventory(tx *sql.Tx, player models.Player) error {
	_, err := tx.Exec(`UPDATE players SET gu_inventory=$1, equipped_gus=$2, primeval_stones=$3, immortal_stones=$4, human_qi=$5 WHERE id=$6`, player.GuInventory, player.EquippedGus, player.PrimevalStones, player.ImmortalStones, player.HumanQi, player.ID)
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
	if err := addAuctionItemToPlayer(tx, &player, itemType, itemData, qty); err != nil {
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
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.cancel_listing", EntityType: "auction",
		EntityID: req.ListingID, RequestSummary: itemType + " | sl=" + fmt.Sprint(qty),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy listing và hoàn trả vật phẩm"})
}

func mapGuFromRaw(guID string) models.GuInstance {
	spec, _ := models.GetGuAny(guID)
	return models.GuInstance{InstanceID: uuid.New().String(), GuID: guID, Rank: spec.Rank, Hunger: 100, Status: "SLEEPING", Equipped: false, IsVital: false}
}

// GetItemName returns a user-friendly name for listed/auctioned items
func GetItemName(itemType string, itemDataJSON []byte) string {
	if itemType == "Gu" {
		var gus []models.GuInstance
		if err := json.Unmarshal(itemDataJSON, &gus); err == nil && len(gus) > 0 {
			if spec, ok := models.GetGuAny(gus[0].GuID); ok {
				return spec.Name
			}
			return gus[0].GuID
		}
	} else if itemType == "Material" {
		var matID string
		if err := json.Unmarshal(itemDataJSON, &matID); err == nil {
			if spec, ok := models.ItemRegistry[matID]; ok {
				return spec.Name
			}
			return matID
		}
	} else if itemType == "Recipe" {
		var rec models.UnlockedRecipe
		if err := json.Unmarshal(itemDataJSON, &rec); err == nil {
			var targetGuID string
			err := db.DB.QueryRow(`SELECT target_gu_id FROM refining_recipes WHERE id = $1`, rec.RecipeID).Scan(&targetGuID)
			if err == nil {
				if spec, ok := models.GetGuAny(targetGuID); ok {
					return fmt.Sprintf("Phương thức phối chế %s", spec.Name)
				}
			}
			return fmt.Sprintf("Phương thức %s", rec.RecipeID)
		}
	}
	return "Vật phẩm không rõ"
}
