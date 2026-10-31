package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type CreateListingRequest struct {
	ItemType            string   `json:"item_type" binding:"required"` // Gu|Material|Recipe|ResourcePoint
	GuInstanceID        string   `json:"gu_instance_id"`
	GuInstanceIDs       []string `json:"gu_instance_ids"`
	ItemID              string   `json:"item_id"`
	Quantity            int64    `json:"quantity"`
	Price               float64  `json:"price" binding:"required"`
	Currency            string   `json:"currency" binding:"required"` // primeval|immortal
	DurationHours       int      `json:"duration_hours" binding:"required"`
	ListingType         string   `json:"listing_type" binding:"required"` // MARKET|AUCTION|TRADE
	TargetBuyerUsername string   `json:"target_buyer_username"`
}

// ResourcePointItemData is the item_data payload for ResourcePoint listings:
// the point itself never leaves the seller's aperture table, only the details
// of which point is being sold travel with the listing.
type ResourcePointItemData struct {
	PointID   string `json:"point_id"`
	PointName string `json:"point_name"`
}

type PlaceBidRequest struct {
	ListingID string  `json:"listing_id" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
}

type BuyListingRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
	Quantity  int64  `json:"quantity"`
}

// Duration-based fee tiers
func CreateListing(c *gin.Context) {
	var req CreateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Price < 0.001 || !pricing.IsValidAmount(req.Price) || req.DurationHours < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Giá và thời gian phải hợp lệ"})
		return
	}
	if req.Currency != "primeval" && req.Currency != "immortal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "currency phải là primeval hoặc immortal"})
		return
	}
	if req.ItemType != "Gu" && req.ItemType != "Material" && req.ItemType != "Recipe" && req.ItemType != "ResourcePoint" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_type phải là Gu, Material, Recipe hoặc ResourcePoint"})
		return
	}
	if req.ListingType != "MARKET" && req.ListingType != "AUCTION" && req.ListingType != "TRADE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "listing_type phải là MARKET, AUCTION hoặc TRADE"})
		return
	}
	if req.ListingType == "AUCTION" && (req.DurationHours < 1 || req.DurationHours > 24) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian đấu giá phải từ 1 đến 24 giờ (dưới 1 ngày)"})
		return
	}
	if req.ListingType == "MARKET" && (req.DurationHours < 1 || req.DurationHours > 168) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian đăng bán chợ chỉ từ 1 đến 7 ngày (tối đa 168 giờ)"})
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
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể tạo giao dịch Bảo Hoàng Thiên!"})
		return
	}

	_, _, hasConnectingHeaven := models.FindFirstMatchingGu(player.GuInventory, func(gu models.GuInstance) bool {
		return models.InventoryHasGuCapability(models.GuInventory{gu}, "MARKETPLACE_ACCESS", false)
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

		err = tx.QueryRow(`SELECT id FROM players WHERE LOWER(username) = LOWER($1)`, strings.TrimSpace(req.TargetBuyerUsername)).Scan(&targetBuyerID)
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

	// Calculate quantity for upfront fee calculation
	var effectiveQty int64 = req.Quantity
	if req.ItemType == "Gu" {
		if len(req.GuInstanceIDs) > 0 {
			effectiveQty = int64(len(req.GuInstanceIDs))
		} else if req.GuInstanceID != "" {
			effectiveQty = 1
		}
	} else if req.ItemType == "Recipe" || req.ItemType == "ResourcePoint" {
		effectiveQty = 1
	}
	if effectiveQty <= 0 {
		effectiveQty = 1
	}

	totalListedPrice := req.Price * float64(effectiveQty)
	upfrontFee := getUpfrontFee(req.ListingType, req.DurationHours, totalListedPrice)
	if req.ListingType == "TRADE" {
		upfrontFee = 0
	}

	if req.ListingType == "MARKET" && req.Currency == "immortal" {
		if player.ImmortalStones < upfrontFee {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch để chi trả phí đăng bán cơ bản (cần %.3f Tiên Nguyên Thạch)", upfrontFee)})
			return
		}
		player.ImmortalStones -= upfrontFee
	} else {
		if player.PrimevalStones < upfrontFee {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch để chi trả phí đăng bán cơ bản (cần %.3f Nguyên Thạch)", upfrontFee)})
			return
		}
		player.PrimevalStones -= upfrontFee
	}

	var itemDataJSON []byte
	var guIDsToSell []string
	var gusToSell []models.GuInstance
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
		// Tiên Cổ rank>=6: mỗi đơn chỉ 1 instance, giữ instance_id xuyên suốt.
		if len(gusToSell) > 1 {
			for _, gu := range gusToSell {
				if gu.Rank >= 6 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Cổ (rank>=6) mỗi đơn chỉ đăng bán 1 instance"})
					return
				}
			}
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
	} else if req.ItemType == "ResourcePoint" {
		// canon: điểm tài nguyên thế sự do một Tiên Khiếu sở hữu mới bán được
		// trên Bảo Hoàng Thiên (mâm Nguyệt / Phong Tuyết Đỉnh truyền tay).
		if req.ItemID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không chỉ định điểm tài nguyên để đăng bán"})
			return
		}
		if req.Quantity > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi đơn chỉ có thể niêm yết 1 điểm tài nguyên"})
			return
		}
		req.Quantity = 1
		pointID, err := uuid.Parse(req.ItemID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mã điểm tài nguyên không hợp lệ"})
			return
		}
		var pointName, scope, pointState string
		var listed bool
		var apertureOwner uuid.UUID
		err = tx.QueryRow(`
			SELECT p.name, COALESCE(p.scope, 'APERTURE'), p.state, COALESCE(p.listed, false), a.player_id
			FROM aperture_resource_points p
			LEFT JOIN immortal_apertures a ON a.id = p.aperture_id
			WHERE p.id = $1
			FOR UPDATE OF p
		`, pointID).Scan(&pointName, &scope, &pointState, &listed, &apertureOwner)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Điểm tài nguyên không tồn tại"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
			return
		}
		if scope != "APERTURE" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm tài nguyên thế giới chưa về Tiên Khiếu nào, không thể niêm yết"})
			return
		}
		if pointState != "ACTIVE" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm tài nguyên đã cạn kiệt sinh khí, không thể niêm yết"})
			return
		}
		if apertureOwner != player.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm tài nguyên không thuộc Tiên Khiếu của đạo hữu"})
			return
		}
		if listed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm tài nguyên này đã được niêm yết trên Bảo Hoàng Thiên"})
			return
		}
		itemDataJSON, err = json.Marshal(ResourcePointItemData{PointID: req.ItemID, PointName: pointName})
		if err != nil {
			logger.S.Errorf("Failed to marshal resource point: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đăng bán điểm tài nguyên"})
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
	if req.ItemType == "ResourcePoint" {
		// Column guard: exactly one listing may ever claim the point.
		res, err := tx.Exec(`UPDATE aperture_resource_points SET listed = true, updated_at = NOW() WHERE id = $1 AND listed = false`, req.ItemID)
		if err != nil {
			logger.S.Errorf("Failed to mark resource point listed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể niêm yết điểm tài nguyên"})
			return
		}
		if rows, err := res.RowsAffected(); err != nil || rows == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Điểm tài nguyên vừa được niêm yết bởi giao dịch khác", "code": "RESOURCE_BUSY"})
			return
		}
	}
	if req.ItemType == "Gu" {
		if err := db.RemovePlayerGusForPlayer(tx, player.ID.String(), guIDsToSell); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
			return
		}
		for _, g := range gusToSell {
			guName := g.GuID
			if spec, ok := models.GetGu(g.GuID, g.Rank); ok && spec.Name != "" {
				guName = spec.Name
			}
			summary := fmt.Sprintf("Cổ Sư [%s] niêm yết Cổ trùng [%s] R%d lên Bảo Hoàng Thiên (Giá: %.3f %s).", player.Username, guName, g.Rank, req.Price, req.Currency)
			_ = db.LogGuHistory(tx, g.InstanceID, g.GuID, g.Rank, "MARKET_LISTED", &player.ID, nil, summary, map[string]interface{}{
				"listing_id": listingID,
				"price":      req.Price,
				"currency":   req.Currency,
			})
		}
	}
	_, err = tx.Exec(`UPDATE players SET equipped_gus=$1, primeval_stones=$2, immortal_stones=$3, human_qi=$4 WHERE id=$5`, player.EquippedGus, player.PrimevalStones, player.ImmortalStones, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated, marketplaceItemDomains(req.ItemType)...)

	itemName := GetItemName(req.ItemType, itemDataJSON)

	// Trigger System Chat Notification on Bảo Hoàng Thiên channel
	if req.ListingType == "MARKET" || req.ListingType == "AUCTION" {
		currencyName := "Nguyên Thạch"
		if req.Currency == "immortal" {
			currencyName = "Tiên Nguyên Thạch"
		}
		var msgContent string
		if req.ListingType == "AUCTION" {
			msgContent = fmt.Sprintf("Đạo hữu %s vừa đưa lên đấu giá %d %s với mức giá khởi điểm là %.3f %s tại Bảo Hoàng Thiên!", player.Username, req.Quantity, itemName, req.Price, currencyName)
		} else {
			msgContent = fmt.Sprintf("Đạo hữu %s vừa treo bán %d %s với giá %.3f %s tại Bảo Hoàng Thiên!", player.Username, req.Quantity, itemName, req.Price, currencyName)
		}
		_ = SendSystemMessage("general", sql.NullString{}, msgContent)
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.create_listing", EntityType: "auction",
		EntityID:       listingID,
		RequestSummary: fmt.Sprintf("người bán=%s (%s) | loại vật phẩm=%s | tên=%s | id=%s | sl=%d | giá=%.3f %s | loại=%s", player.Username, authID, req.ItemType, itemName, req.ItemID, req.Quantity, req.Price, req.Currency, req.ListingType),
		IPAddress:      c.ClientIP(), Success: true,
	})
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": "Đăng bán thành công", "listing_id": listingID})
}

func ListListings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Resolve catalog behavior from the in-memory compiled definition. Do not
	// scan or interpret gu_dictionary.effects JSON on the request path.
	inventory, err := db.LoadPlayerGus(db.DB, authID.String(), false)
	hasConnectingHeaven := err == nil && models.InventoryHasGuCapability(inventory, "MARKETPLACE_ACCESS", false)
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
	if req.Amount < 0.001 || !pricing.IsValidAmount(req.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Giá đấu phải là bội số của 0.001"})
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
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể đấu giá Bảo Hoàng Thiên!"})
		return
	}

	_, _, hasConnectingHeaven := models.FindFirstMatchingGu(player.GuInventory, func(gu models.GuInstance) bool {
		return models.InventoryHasGuCapability(models.GuInventory{gu}, "MARKETPLACE_ACCESS", false)
	})
	if !hasConnectingHeaven {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để truy cập Bảo Hoàng Thiên!"})
		return
	}

	var sellerID uuid.UUID
	var sellerName, itemType string
	var rawItemData []byte
	var currency, status, listingType string
	var current float64
	var bidder sql.NullString
	var end time.Time
	err = tx.QueryRow(`
		SELECT a.seller_id, p.username, a.currency, a.current_price, a.highest_bidder_id, a.status, a.end_time, a.listing_type, a.item_type, a.item_data 
		FROM auction a
		JOIN players p ON a.seller_id = p.id
		WHERE a.id=$1 FOR UPDATE OF a NOWAIT
	`, req.ListingID).Scan(&sellerID, &sellerName, &currency, &current, &bidder, &status, &end, &listingType, &itemType, &rawItemData)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Phiên đấu giá đang có người khác đặt giá, vui lòng thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
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
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Giá phải cao hơn %.3f", current)})
		return
	}
	neededAmount := req.Amount
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

	// Refund the previous bidder — atomic increment, no full player load needed
	var prevBidderID uuid.UUID
	if bidder.Valid && bidder.String != player.ID.String() {
		prevBidderID, _ = uuid.Parse(bidder.String)
		refundCol := "primeval_stones"
		if currency == "immortal" {
			refundCol = "immortal_stones"
		}
		_, err = tx.Exec(
			fmt.Sprintf(`UPDATE players SET %s = %s + $1 WHERE id = $2`, refundCol, refundCol),
			current, prevBidderID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "refund failed"})
			return
		}
		notify.Send(tx, prevBidderID, "auction_outbid", "Bị đè giá", fmt.Sprintf("Bạn đã bị đè giá ở phòng đấu giá cho vật phẩm đấu giá. Được hoàn trả %.3f %s.", current, currency))
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
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated, PlayerDomain)
	if prevBidderID != uuid.Nil && prevBidderID != player.ID {
		ClearPendingPlayerQueues(prevBidderID.String())
		PublishPlayerRealtimeEvent(prevBidderID, PlayerEventUpdated, PlayerDomain)
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.bid", EntityType: "auction",
		EntityID:       req.ListingID,
		RequestSummary: fmt.Sprintf("người đặt giá=%s (%s) | người bán=%s (%s) | vật phẩm=%s | giá=%.3f %s | giá cũ=%.3f %s", player.Username, authID, sellerName, sellerID, itemName, req.Amount, currency, current, currency),
		IPAddress:      c.ClientIP(), Success: true,
	})
	if sellerID != authID {
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID: &sellerID, Action: "marketplace.bid_notify", EntityType: "auction",
			EntityID:       req.ListingID,
			RequestSummary: fmt.Sprintf("người đặt giá=%s (%s) | người bán=%s (%s) | vật phẩm=%s | giá=%.3f %s", player.Username, authID, sellerName, sellerID, itemName, req.Amount, currency),
			IPAddress:      c.ClientIP(), Success: true,
		})
		currencyName := "Nguyên Thạch"
		if currency == "immortal" {
			currencyName = "Tiên Nguyên Thạch"
		}
		// Use player.Username already loaded — no extra query needed
		notify.Send(db.DB, sellerID, "auction_bid", "Ai đó đặt giá",
			fmt.Sprintf("%s đã đặt giá %.3f %s cho mặt hàng của bạn.", player.Username, req.Amount, currencyName))
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
	if gamefx.IsDeadState(&buyer) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể mua vật phẩm Bảo Hoàng Thiên!"})
		return
	}

	_, _, hasConnectingHeaven := models.FindFirstMatchingGu(buyer.GuInventory, func(gu models.GuInstance) bool {
		return models.InventoryHasGuCapability(models.GuInventory{gu}, "MARKETPLACE_ACCESS", false)
	})
	if !hasConnectingHeaven {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để truy cập Bảo Hoàng Thiên!"})
		return
	}

	buyerBefore := append(models.GuInventory(nil), buyer.GuInventory...)

	var sellerID uuid.UUID
	var itemType, currency, status, listingType string
	var currentPrice float64
	var qty int64
	var durationHours int
	var rawItemData []byte
	var targetBuyerID sql.NullString
	err = tx.QueryRow(`SELECT seller_id, item_type, item_data, quantity, currency, current_price, status, listing_type, target_buyer_id, duration_hours FROM auction WHERE id=$1 FOR UPDATE NOWAIT`, req.ListingID).Scan(&sellerID, &itemType, &rawItemData, &qty, &currency, &currentPrice, &status, &listingType, &targetBuyerID, &durationHours)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Vật phẩm đang có người thanh toán hoặc cập nhật, vui lòng thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
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

	totalPrice := float64(purchaseQty) * currentPrice
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
			return models.InventoryHasGuCapability(models.GuInventory{gu}, "MARKETPLACE_ACCESS", false)
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
	commission := math.Round(totalPrice*commPct*1000) / 1000
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
	} else if itemType == "ResourcePoint" {
		itemDataToSend = string(rawItemData)
	} else {
		var itemData string
		if err := json.Unmarshal(rawItemData, &itemData); err != nil {
			itemData = string(rawItemData)
		}
		itemDataToSend = itemData
	}

	if itemType == "ResourcePoint" {
		// canon: điểm tài nguyên đổi chủ thẳng vào Tiên Khiếu của người mua
		if err := bindResourcePointToBuyer(tx, &buyer, sellerID, itemDataToSend); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "RESOURCE_BUSY"})
			return
		}
	} else if err := addAuctionItemToPlayer(tx, &buyer, itemType, itemDataToSend, purchaseQty); err != nil {
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
		PublishPlayerRealtimeEvent(sellerID, PlayerEventUpdated, PlayerDomain)
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.buy", EntityType: "auction",
		EntityID:       req.ListingID,
		RequestSummary: fmt.Sprintf("người mua=%s (%s) | người bán=%s (%s) | vật phẩm=%s | sl=%d | giá=%.3f %s | loại=%s", buyer.Username, authID, seller.Username, sellerID, itemName, purchaseQty, totalPrice, currency, listingType),
		IPAddress:      c.ClientIP(), Success: true,
	})
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &sellerID, Action: "marketplace.sold", EntityType: "auction",
		EntityID:       req.ListingID,
		RequestSummary: fmt.Sprintf("người bán=%s (%s) | người mua=%s (%s) | vật phẩm=%s | sl=%d | nhận=%.3f %s (giá=%.3f, phí=%.3f) | loại=%s", seller.Username, sellerID, buyer.Username, authID, itemName, purchaseQty, payout, currency, totalPrice, commission, listingType),
		IPAddress:      c.ClientIP(), Success: true,
	})
	currencyName := "Nguyên Thạch"
	if currency == "immortal" {
		currencyName = "Tiên Nguyên Thạch"
	}
	if sellerID != authID {
		notify.Send(db.DB, sellerID, "market_sold", "Mặt hàng đã bán",
			fmt.Sprintf("%s đã mua %d %s với giá %.3f %s.", buyer.Username, purchaseQty, itemName, totalPrice, currencyName))
	}
	notify.Send(db.DB, authID, "market_bought", "Mua hàng thành công",
		fmt.Sprintf("Bạn đã mua %d %s với giá %.3f %s.", purchaseQty, itemName, totalPrice, currencyName))
	respondActionWithInventory(c, http.StatusOK, buyer, gin.H{"message": "Mua hàng thành công"})
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
	var highestBidderID sql.NullString
	err = tx.QueryRow(`SELECT seller_id,item_type,item_data,quantity,status,listing_type,highest_bidder_id FROM auction WHERE id=$1 FOR UPDATE NOWAIT`, req.ListingID).Scan(&sellerID, &itemType, &rawItemData, &qty, &status, &listingType, &highestBidderID)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Phiên giao dịch đang có người thao tác, vui lòng thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy listing"})
		return
	}
	if sellerID != authID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không phải chủ listing"})
		return
	}
	if listingType == "AUCTION" && highestBidderID.Valid && highestBidderID.String != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể hủy phiên đấu giá khi đã có người đặt giá"})
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
		return models.InventoryHasGuCapability(models.GuInventory{gu}, "MARKETPLACE_ACCESS", false)
	})
	if !hasConnectingHeaven {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) để truy cập Bảo Hoàng Thiên!"})
		return
	}
	playerBefore := append(models.GuInventory(nil), player.GuInventory...)
	if itemType == "ResourcePoint" {
		// Điểm tài nguyên chưa bao giờ rời Tiên Khiếu của người bán: hủy chỉ
		// cần gỡ cờ listed, không có gì để hoàn trả vào túi đồ.
		if err := unlistResourcePoint(tx, itemData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn trả điểm tài nguyên"})
			return
		}
	} else {
		if err := addAuctionItemToPlayer(tx, &player, itemType, itemData, int64(qty)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn trả vật phẩm"})
			return
		}
		if err := updatePlayerInventory(tx, player, playerBefore); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật túi đồ"})
			return
		}
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
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated, marketplaceItemDomains(itemType)...)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "marketplace.cancel_listing", EntityType: "auction",
		EntityID:       req.ListingID,
		RequestSummary: fmt.Sprintf("người hủy=%s (%s) | vật phẩm=%s | sl=%d | loại=%s", player.Username, authID, itemName, qty, listingType),
		IPAddress:      c.ClientIP(), Success: true,
	})
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": "Đã hủy listing và hoàn trả vật phẩm"})
}

func marketplaceItemDomains(itemType string) []PlayerEventDomain {
	domains := []PlayerEventDomain{PlayerDomain}
	if itemType == "Gu" {
		return append(domains, GuInventoryDomain, CombatStatsDomain)
	}
	if itemType == "ResourcePoint" {
		// Điểm tài nguyên chứa Cổ trùng hoang dã (match Gu branch behavior),
		// nhưng khía cạnh items/điểm tài nguyên cũng cần làm mới.
		return append(domains, ItemInventoryDomain, GuInventoryDomain)
	}
	return append(domains, ItemInventoryDomain)
}
