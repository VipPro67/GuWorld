package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
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
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type CreateListingRequest struct {
	ItemType            string   `json:"item_type" binding:"required"` // Gu|Material|Recipe
	GuInstanceID        string   `json:"gu_instance_id"`
	GuInstanceIDs       []string `json:"gu_instance_ids"`
	ItemID              string   `json:"item_id"`
	Quantity            int64    `json:"quantity"`
	Price               int64    `json:"price" binding:"required"`
	Currency            string   `json:"currency" binding:"required"` // primeval|immortal
	DurationHours       int      `json:"duration_hours" binding:"required"`
	ListingType         string   `json:"listing_type" binding:"required"` // MARKET|AUCTION|TRADE
	TargetBuyerUsername string   `json:"target_buyer_username"`
}

type PlaceBidRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
	Amount    int64  `json:"amount" binding:"required"`
}

type BuyListingRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
	Quantity  int64  `json:"quantity"`
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

func getUpfrontFee(listingType string, durationHours int) int64 {
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

	_, _, hasConnectingHeaven := models.FindFirstMatchingGu(player.GuInventory, func(gu models.GuInstance) bool {
		return gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD"
	})
	if !hasConnectingHeaven {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để truy cập Bảo Hoàng Thiên!"})
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
	var guIDsToSell []string
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
				if gu.Equipped {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể bán Cổ trùng đang trang bị"})
					return
				}
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
		player.EquippedGus = []string{}
		for _, gu := range models.FilterEquippedActive(player.GuInventory) {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
		req.Quantity = int64(len(guIDsToSell))

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
		qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), req.ItemID)
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
	if req.ItemType == "Gu" {
		if err := db.RemovePlayerGusForPlayer(tx, player.ID.String(), guIDsToSell); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
			return
		}
	}
	_, err = tx.Exec(`UPDATE players SET equipped_gus=$1, primeval_stones=$2, human_qi=$3 WHERE id=$4`, player.EquippedGus, player.PrimevalStones, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	itemName := GetItemName(req.ItemType, itemDataJSON)

	// Trigger System Chat Notification on Bảo Hoàng Thiên channel
	if req.ListingType == "MARKET" || req.ListingType == "AUCTION" {
		currencyName := "Nguyên Thạch"
		if req.Currency == "immortal" {
			currencyName = "Tiên Nguyên Thạch"
		}
		var msgContent string
		if req.ListingType == "AUCTION" {
			msgContent = fmt.Sprintf("Đạo hữu %s vừa đưa lên đấu giá %d %s với mức giá khởi điểm là %d %s tại Bảo Hoàng Thiên!", player.Username, req.Quantity, itemName, req.Price, currencyName)
		} else {
			msgContent = fmt.Sprintf("Đạo hữu %s vừa treo bán %d %s với giá %d %s tại Bảo Hoàng Thiên!", player.Username, req.Quantity, itemName, req.Price, currencyName)
		}
		_ = SendSystemMessage("general", sql.NullString{}, msgContent)
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.create_listing", EntityType: "auction",
		EntityID:       listingID,
		RequestSummary: fmt.Sprintf("người bán=%s (%s) | loại vật phẩm=%s | tên=%s | id=%s | sl=%d | giá=%d %s | loại=%s", player.Username, authID, req.ItemType, itemName, req.ItemID, req.Quantity, req.Price, req.Currency, req.ListingType),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đăng bán thành công", "listing_id": listingID})
}

func ListListings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1 AND gu_id = 'GU_CONNECTING_HEAVEN' AND state != 'DEAD'`, authID).Scan(&count)
	hasConnectingHeaven := err == nil && count > 0
	if !hasConnectingHeaven {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để truy cập Bảo Hoàng Thiên!"})
		return
	}

	cacheKey := "market_listings_user_" + authID.String()
	var cachedOut []gin.H
	if err := cache.GetJSON(cacheKey, &cachedOut); err == nil {
		c.JSON(http.StatusOK, cachedOut)
		return
	}

	var rows *sql.Rows
	var queryErr error
	if ok {
		rows, queryErr = db.DB.QueryContext(c.Request.Context(), `
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
		rows, queryErr = db.DB.QueryContext(c.Request.Context(), `
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
	if queryErr != nil {
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

	_, _, hasConnectingHeaven := models.FindFirstMatchingGu(player.GuInventory, func(gu models.GuInstance) bool {
		return gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD"
	})
	if !hasConnectingHeaven {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để truy cập Bảo Hoàng Thiên!"})
		return
	}

	var sellerID uuid.UUID
	var sellerName, itemType string
	var rawItemData []byte
	var currency, status, listingType string
	var current int64
	var bidder sql.NullString
	var end time.Time
	err = tx.QueryRow(`
		SELECT a.seller_id, p.username, a.currency, a.current_price, a.highest_bidder_id, a.status, a.end_time, a.listing_type, a.item_type, a.item_data 
		FROM auction a
		JOIN players p ON a.seller_id = p.id
		WHERE a.id=$1 FOR UPDATE
	`, req.ListingID).Scan(&sellerID, &sellerName, &currency, &current, &bidder, &status, &end, &listingType, &itemType, &rawItemData)
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
	if currency != "primeval" && currency != "immortal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại tiền tệ không hợp lệ"})
		return
	}
	if req.Amount <= current {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Giá phải cao hơn %d", current)})
		return
	}
	var neededAmount int64 = req.Amount
	if bidder.Valid && bidder.String == player.ID.String() {
		neededAmount = req.Amount - current
	}

	// Check balance and deduct based on currency type
	currencyName := "Nguyên Thạch"
	if currency == "immortal" {
		currencyName = "Tiên Nguyên Thạch"
	}

	if currency == "primeval" {
		if player.PrimevalStones < neededAmount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ " + currencyName})
			return
		}
		player.PrimevalStones -= neededAmount
	} else {
		if player.ImmortalStones < neededAmount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ " + currencyName})
			return
		}
		player.ImmortalStones -= neededAmount
	}

	// Refund the previous bidder
	if bidder.Valid && bidder.String != player.ID.String() {
		var prevBidder models.Player
		prevBidderID, _ := uuid.Parse(bidder.String)
		prevBidder, err = evaluateManaTx(tx, prevBidderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load previous bidder"})
			return
		}
		if currency == "primeval" {
			prevBidder.PrimevalStones += current
			_, err = tx.Exec(`UPDATE players SET primeval_stones=$1 WHERE id=$2`, prevBidder.PrimevalStones, prevBidder.ID)
		} else {
			prevBidder.ImmortalStones += current
			_, err = tx.Exec(`UPDATE players SET immortal_stones=$1 WHERE id=$2`, prevBidder.ImmortalStones, prevBidder.ID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "refund failed"})
			return
		}
		notify.Send(db.DB, prevBidder.ID, "auction_outbid", "Bị đè giá", fmt.Sprintf("Bạn đã bị đè giá ở phòng đấu giá cho vật phẩm đấu giá. Được hoàn trả %d %s.", current, currency))
	}

	if currency == "primeval" {
		_, err = tx.Exec(`UPDATE players SET primeval_stones=$1, human_qi=$2 WHERE id=$3`, player.PrimevalStones, player.HumanQi, player.ID)
	} else {
		_, err = tx.Exec(`UPDATE players SET immortal_stones=$1, human_qi=$2 WHERE id=$3`, player.ImmortalStones, player.HumanQi, player.ID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	_, err = tx.Exec(`UPDATE auction SET current_price=$1, highest_bidder_id=$2 WHERE id=$3`, req.Amount, player.ID, req.ListingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bid failed"})
		return
	}

	itemName := GetItemName(itemType, rawItemData)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.bid", EntityType: "auction",
		EntityID:       req.ListingID,
		RequestSummary: fmt.Sprintf("người đặt giá=%s (%s) | người bán=%s (%s) | vật phẩm=%s | giá=%d %s | giá cũ=%d %s", player.Username, authID, sellerName, sellerID, itemName, req.Amount, currency, current, currency),
		IPAddress:      c.ClientIP(), Success: true,
	})
	if sellerID != authID {
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID: &sellerID, Action: "marketplace.bid_notify", EntityType: "auction",
			EntityID:       req.ListingID,
			RequestSummary: fmt.Sprintf("người đặt giá=%s (%s) | người bán=%s (%s) | vật phẩm=%s | giá=%d %s", player.Username, authID, sellerName, sellerID, itemName, req.Amount, currency),
			IPAddress:      c.ClientIP(), Success: true,
		})
		currencyName := "Nguyên Thạch"
		if currency == "immortal" {
			currencyName = "Tiên Nguyên Thạch"
		}
		var buyerUsername string
		_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, authID).Scan(&buyerUsername)
		notify.Send(db.DB, sellerID, "auction_bid", "Ai đó đặt giá",
			fmt.Sprintf("%s đã đặt giá %d %s cho mặt hàng của bạn.", buyerUsername, req.Amount, currencyName))
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

	_, _, hasConnectingHeaven := models.FindFirstMatchingGu(buyer.GuInventory, func(gu models.GuInstance) bool {
		return gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD"
	})
	if !hasConnectingHeaven {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để truy cập Bảo Hoàng Thiên!"})
		return
	}

	buyerBefore := append(models.GuInventory(nil), buyer.GuInventory...)

	var sellerID uuid.UUID
	var itemType, currency, status, listingType string
	var currentPrice, qty int64
	var durationHours int
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này được dành riêng cho người chơi khác, bạn không thể mua!"})
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

	totalPrice := int64(purchaseQty) * currentPrice
	if currency == "primeval" {
		if buyer.PrimevalStones < totalPrice {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch"})
			return
		}
		buyer.PrimevalStones -= totalPrice
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
	sellerBefore := append(models.GuInventory(nil), seller.GuInventory...)

	if listingType == "TRADE" {
		_, _, sellerConnectingHeaven := models.FindFirstMatchingGu(seller.GuInventory, func(gu models.GuInstance) bool {
			return gu.GuID == "GU_CONNECTING_HEAVEN"
		})
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
	commission := int64(math.Round(float64(totalPrice) * commPct))
	payout := totalPrice - commission

	if currency == "primeval" {
		seller.PrimevalStones += payout
	} else {
		seller.ImmortalStones += payout
	}

	var itemDataToSend string
	if itemType == "Gu" {
		var gus []models.GuInstance
		if err := json.Unmarshal(rawItemData, &gus); err == nil && len(gus) >= int(purchaseQty) {
			purchasedGus := gus[:int(purchaseQty)]
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

	if err := updatePlayerInventory(tx, buyer, buyerBefore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật kho đồ người mua"})
		fmt.Println("Error updating buyer inventory:", err)
		return
	}

	if err := updatePlayerInventory(tx, seller, sellerBefore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật tài sản người bán"})
		return
	}

	if purchaseQty == qty {
		_, err = tx.Exec(`UPDATE auction SET status='COMPLETED' WHERE id=$1`, req.ListingID)
	} else {
		if itemType == "Gu" {
			var gus []models.GuInstance
			if err := json.Unmarshal(rawItemData, &gus); err == nil && len(gus) >= int(purchaseQty) {
				remainingGus := gus[int(purchaseQty):]
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

	itemName := GetItemName(itemType, rawItemData)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete purchase"})
		return
	}
	if sellerID != authID {
		ClearPendingPlayerQueues(sellerID.String())
		PublishPlayerRealtimeEvent(sellerID, PlayerEventUpdated)
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.buy", EntityType: "auction",
		EntityID:       req.ListingID,
		RequestSummary: fmt.Sprintf("người mua=%s (%s) | người bán=%s (%s) | vật phẩm=%s | sl=%d | giá=%d %s | loại=%s", buyer.Username, authID, seller.Username, sellerID, itemName, purchaseQty, totalPrice, currency, listingType),
		IPAddress:      c.ClientIP(), Success: true,
	})
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &sellerID, Action: "marketplace.sold", EntityType: "auction",
		EntityID:       req.ListingID,
		RequestSummary: fmt.Sprintf("người bán=%s (%s) | người mua=%s (%s) | vật phẩm=%s | sl=%d | nhận=%d %s (giá=%d, phí=%d) | loại=%s", seller.Username, sellerID, buyer.Username, authID, itemName, purchaseQty, payout, currency, totalPrice, commission, listingType),
		IPAddress:      c.ClientIP(), Success: true,
	})
	currencyName := "Nguyên Thạch"
	if currency == "immortal" {
		currencyName = "Tiên Nguyên Thạch"
	}
	if sellerID != authID {
		notify.Send(db.DB, sellerID, "market_sold", "Mặt hàng đã bán",
			fmt.Sprintf("%s đã mua %d %s với giá %d %s.", buyer.Username, purchaseQty, itemName, totalPrice, currencyName))
	}
	notify.Send(db.DB, authID, "market_bought", "Mua hàng thành công",
		fmt.Sprintf("Bạn đã mua %d %s với giá %d %s.", purchaseQty, itemName, totalPrice, currencyName))
	respondActionWithInventory(c, http.StatusOK, buyer, gin.H{"message": "Mua hàng thành công"})
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
	// Cheap guard: bail before any heavy row fetch / JSON unmarshal when no
	// auction is actually expired. The 30s ticker would otherwise repeat this
	// query 2880×/day per server even with zero activity.
	var hasExpired bool
	if err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM auction WHERE status='ACTIVE' AND end_time <= NOW())`).Scan(&hasExpired); err != nil {
		return err
	}
	if !hasExpired {
		return nil
	}

	rows, err := db.DB.Query(`SELECT id,seller_id,highest_bidder_id,item_type,item_data,quantity,current_price,currency,listing_type,duration_hours FROM auction WHERE status='ACTIVE' AND end_time <= NOW() ORDER BY end_time LIMIT 50`)
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
		var qty int64
		var durationHours int
		var currentPrice int64
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

func settleAuction(auctionID string, sellerID uuid.UUID, highestBidderID sql.NullString, itemType, itemData string, qty int64, currentPrice int64, currency string, listingType string, durationHours int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	seller, err := evaluateManaTx(tx, sellerID)
	if err != nil {
		return err
	}
	sellerBefore := append(models.GuInventory(nil), seller.GuInventory...)

	if highestBidderID.Valid {
		buyerID, err := uuid.Parse(highestBidderID.String)
		if err != nil {
			return err
		}

		buyer, err := evaluateManaTx(tx, buyerID)
		if err != nil {
			return err
		}
		buyerBefore := append(models.GuInventory(nil), buyer.GuInventory...)

		// Time-based commission fee
		commission := int64(math.Round(float64(currentPrice) * getCommissionPct(listingType, durationHours)))
		payout := currentPrice - commission

		if currency == "primeval" {
			seller.PrimevalStones += payout
		} else {
			seller.ImmortalStones += payout
		}

		if err := addAuctionItemToPlayer(tx, &buyer, itemType, itemData, qty); err != nil {
			return err
		}

		if err := updatePlayerInventory(tx, buyer, buyerBefore); err != nil {
			return err
		}
		if err := updatePlayerInventory(tx, seller, sellerBefore); err != nil {
			return err
		}

		if _, err := tx.Exec(`UPDATE auction SET status='COMPLETED' WHERE id=$1`, auctionID); err != nil {
			return err
		}

		itemName := GetItemName(itemType, []byte(itemData))
		auditlog.Log(tx, auditlog.Entry{
			PlayerID: &buyerID, Action: "marketplace.settle_auction_win", EntityType: "auction",
			EntityID:       auctionID,
			RequestSummary: fmt.Sprintf("người thắng đấu giá=%s (%s) | người bán=%s (%s) | vật phẩm=%s | sl=%d | giá=%d %s | loại=%s", buyer.Username, buyerID, seller.Username, sellerID, itemName, qty, currentPrice, currency, listingType),
			Success:        true,
		})
	} else {
		if err := addAuctionItemToPlayer(tx, &seller, itemType, itemData, qty); err != nil {
			return err
		}
		if err := updatePlayerInventory(tx, seller, sellerBefore); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE auction SET status='CANCELLED' WHERE id=$1`, auctionID); err != nil {
			return err
		}

		itemName := GetItemName(itemType, []byte(itemData))
		auditlog.Log(tx, auditlog.Entry{
			PlayerID: &sellerID, Action: "marketplace.settle_auction_expired", EntityType: "auction",
			EntityID:       auctionID,
			RequestSummary: fmt.Sprintf("đấu giá hết hạn không có người mua | người bán=%s (%s) | vật phẩm=%s | sl=%d | giá khởi điểm=%d %s | loại=%s", seller.Username, sellerID, itemName, qty, currentPrice, currency, listingType),
			Success:        true,
		})
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Notifications after successful commit (best-effort).
	currencyName := "Nguyên Thạch"
	if currency == "immortal" {
		currencyName = "Tiên Nguyên Thạch"
	}
	if highestBidderID.Valid {
		bidderUUID, _ := uuid.Parse(highestBidderID.String)
		notify.Send(db.DB, bidderUUID, "auction_won", "Thắng đấu giá",
			fmt.Sprintf("Bạn đã thắng đấu giá với giá %d %s!", currentPrice, currencyName))
		notify.Send(db.DB, sellerID, "auction_sold", "Đấu giá hoàn tất",
			fmt.Sprintf("Mặt hàng đã được bán với giá %d %s.", currentPrice, currencyName))
	} else {
		notify.Send(db.DB, sellerID, "auction_expired", "Đấu giá hết hạn",
			"Phiên đấu giá đã hết hạn và không có người đặt giá.")
	}
	return nil
}

func addAuctionItemToPlayer(tx *sql.Tx, player *models.Player, itemType, itemData string, qty int64) error {
	if itemType == "Gu" {
		var gus []models.GuInstance
		if err := json.Unmarshal([]byte(itemData), &gus); err == nil {
			if qty > int64(len(gus)) {
				qty = int64(len(gus))
			}
			for i := int64(0); i < qty; i++ {
				gu := gus[i]
				gu.InstanceID = uuid.New().String() // Tạo ID mới để tránh trùng lặp
				gu.Status = "SLEEPING"
				gu.Equipped = false
				gu.IsVital = false
				player.GuInventory = append(player.GuInventory, gu)
			}
		} else {
			// Tương thích ngược với định dạng cũ (chỉ lưu string guID)
			for i := int64(0); i < qty; i++ {
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

func updatePlayerInventory(tx *sql.Tx, player models.Player, before models.GuInventory) error {
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE players SET equipped_gus=$1, primeval_stones=$2, immortal_stones=$3, human_qi=$4 WHERE id=$5`, player.EquippedGus, player.PrimevalStones, player.ImmortalStones, player.HumanQi, player.ID)
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
	var itemType, status, listingType string
	var rawItemData []byte
	var qty int
	err = tx.QueryRow(`SELECT seller_id,item_type,item_data,quantity,status,listing_type FROM auction WHERE id=$1 FOR UPDATE`, req.ListingID).Scan(&sellerID, &itemType, &rawItemData, &qty, &status, &listingType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy listing"})
		return
	}
	if sellerID != authID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không phải chủ listing"})
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

	_, _, hasConnectingHeaven := models.FindFirstMatchingGu(player.GuInventory, func(gu models.GuInstance) bool {
		return gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD"
	})
	if !hasConnectingHeaven {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để truy cập Bảo Hoàng Thiên!"})
		return
	}
	playerBefore := append(models.GuInventory(nil), player.GuInventory...)
	if err := addAuctionItemToPlayer(tx, &player, itemType, itemData, int64(qty)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn trả vật phẩm"})
		return
	}
	if err := updatePlayerInventory(tx, player, playerBefore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
		return
	}
	_, err = tx.Exec(`UPDATE auction SET status='CANCELLED' WHERE id=$1`, req.ListingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hủy listing"})
		return
	}

	itemName := GetItemName(itemType, rawItemData)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.cancel_listing", EntityType: "auction",
		EntityID:       req.ListingID,
		RequestSummary: fmt.Sprintf("người hủy=%s (%s) | vật phẩm=%s | sl=%d | loại=%s", player.Username, authID, itemName, qty, listingType),
		IPAddress:      c.ClientIP(), Success: true,
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
			if spec, ok := models.GetGu(gus[0].GuID, gus[0].Rank); ok {
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
			if tg, _, _, err := ParseRecipeID(rec.RecipeID); err == nil {
				targetGuID = tg
			} else {
				_ = db.DB.QueryRow(`SELECT target_gu_id FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1`, rec.RecipeID).Scan(&targetGuID)
			}
			if targetGuID != "" {
				if spec, ok := models.GetGuAny(targetGuID); ok {
					return fmt.Sprintf("Phương thức phối chế %s", spec.Name)
				}
			}
			return fmt.Sprintf("Phương thức %s", rec.RecipeID)
		}
	}
	return "Vật phẩm không rõ"
}

// StartAuctionSettlementWorker runs a background loop to periodically settle expired auctions.
func StartAuctionSettlementWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := settleExpiredAuctions(); err != nil {
					logger.S.Errorf("Failed to settle expired auctions: %v", err)
				}
			}
		}
	}()
}
