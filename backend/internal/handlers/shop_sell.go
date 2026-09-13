package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type SellShopRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
	ItemType string `json:"item_type" binding:"required"`
	Rank     int    `json:"rank"` // 0 = auto; >0 = the specific rank being sold
}

func SellShopItem(c *gin.Context) {
	var req SellShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	totalGain := int64(0)
	sellRank := 1
	var removedIDs []string
	if req.ItemType == "Material" || req.ItemType == "Consumable" {
		preferredCat := "materials"
		if req.ItemType == "Consumable" {
			preferredCat = "consumables"
		}

		rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, player.ID.String(), req.ItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking inventory"})
			return
		}

		type invItem struct {
			category string
			quantity int64
		}
		var invs []invItem
		totalQty := int64(0)
		for rows.Next() {
			var iv invItem
			if err := rows.Scan(&iv.category, &iv.quantity); err != nil {
				rows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error scanning inventory"})
				return
			}
			invs = append(invs, iv)
			totalQty += iv.quantity
		}
		rows.Close()

		if totalQty < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ vật phẩm để bán"})
			return
		}

		remainingDeduct := req.Quantity
		for _, iv := range invs {
			if iv.category == preferredCat {
				deduct := iv.quantity
				if deduct > remainingDeduct {
					deduct = remainingDeduct
				}
				if err := InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, iv.category, deduct); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
					return
				}
				remainingDeduct -= deduct
				if remainingDeduct <= 0 {
					break
				}
			}
		}
		if remainingDeduct > 0 {
			for _, iv := range invs {
				if iv.category != preferredCat {
					deduct := iv.quantity
					if deduct > remainingDeduct {
						deduct = remainingDeduct
					}
					if err := InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, iv.category, deduct); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
						return
					}
					remainingDeduct -= deduct
					if remainingDeduct <= 0 {
						break
					}
				}
			}
		}

		// Resolve sell price: prefer shop catalog price, fallback to item_dictionary rank
		var unitSellPrice int64
		if shopItem, err := findCaravanShopItem(req.ItemID, req.Rank); err == nil {
			unitSellPrice = pricing.SellPrice(shopItem.Price)
			sellRank = shopItem.Rank
		} else {
			var rank int
			var dbBasePrice int64
			_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, req.ItemID).Scan(&dbBasePrice, &rank)
			sellRank = rank
			unitSellPrice = pricing.SellPrice(pricing.ItemBasePrice(dbBasePrice, rank))
		}
		totalGain = unitSellPrice * req.Quantity
	} else if req.ItemType == "Gu" {
		var removed int64 = 0
		newInv := models.GuInventory{}
		for _, gu := range player.GuInventory {
			if gu.GuID == req.ItemID && !gu.IsVital && !gu.Equipped && removed < req.Quantity {
				removed++
				removedIDs = append(removedIDs, gu.InstanceID)
				continue
			}
			newInv = append(newInv, gu)
		}
		if removed < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Cổ trùng để bán hoặc Cổ là Bản Mệnh/Trang bị"})
			return
		}
		player.GuInventory = newInv
		// Resolve sell price: prefer shop catalog price, fallback to gu_dictionary rank
		var unitSellPrice int64
		if shopItem, err := findCaravanShopItem(req.ItemID, req.Rank); err == nil {
			unitSellPrice = pricing.SellPrice(shopItem.Price)
			sellRank = shopItem.Rank
		} else {
			var rank int
			var dbBasePrice int64
			if req.Rank > 0 {
				_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 AND rank=$2`, req.ItemID, req.Rank).Scan(&dbBasePrice, &rank)
			}
			if rank == 0 {
				_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 ORDER BY rank ASC LIMIT 1`, req.ItemID).Scan(&dbBasePrice, &rank)
			}
			sellRank = rank
			unitSellPrice = pricing.SellPrice(pricing.GuBasePrice(dbBasePrice, rank))
		}
		totalGain = unitSellPrice * req.Quantity
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_type phải là Gu hoặc Material"})
		return
	}

	// Pay out in the currency tier matching the item's rank
	if pricing.StoneTierByRank(sellRank) == "immortal" {
		player.ImmortalStones += totalGain
	} else {
		player.PrimevalStones += totalGain
	}
	player.EquippedGus = []string{}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}
	qiGained := 0.0
	if req.ItemType == "Gu" {
		if err := db.RemovePlayerGus(tx, removedIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory changes"})
			return
		}
	}
	sellCurrencyName := "Nguyên Thạch"
	if pricing.StoneTierByRank(sellRank) == "immortal" {
		sellCurrencyName = "Tiên Nguyên Thạch"
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2, equipped_gus=$3, human_qi=$4 WHERE id=$5`,
		player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save sell transaction"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	itemName := req.ItemID
	if req.ItemType == "Gu" {
		if spec, ok := models.GetGuAny(req.ItemID); ok {
			itemName = spec.Name
		}
	} else {
		_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, req.ItemID).Scan(&itemName)
	}

	action := "shop.sell_item"
	entityType := "item"
	if req.ItemType == "Gu" {
		action = "shop.sell_gu"
		entityType = "gu_instance"
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       req.ItemID,
		RequestSummary: fmt.Sprintf("Bán %dx %s (%s) cho Tiệm Buôn | Nhận %d %s", req.Quantity, itemName, req.ItemID, totalGain, sellCurrencyName),
		Metadata: map[string]interface{}{
			"item_id":       req.ItemID,
			"item_name":     itemName,
			"item_type":     req.ItemType,
			"quantity":      req.Quantity,
			"total_gain":    totalGain,
			"currency_type": sellCurrencyName,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Bán thành công, nhận %s! Nhân Khí +%.0f.", sellCurrencyName, qiGained)})
}

type SellShopItemInfo struct {
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
	ItemType string `json:"item_type" binding:"required"`
	Rank     int    `json:"rank"` // 0 = auto; >0 = the specific rank being sold
}

type SellShopBulkRequest struct {
	PlayerID string             `json:"player_id" binding:"required"`
	Items    []SellShopItemInfo `json:"items" binding:"required,dive"`
}

func SellShopItemsBulk(c *gin.Context) {
	var req SellShopBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm nào được chọn"})
		return
	}
	if len(req.Items) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi lần thanh lý tối đa 50 loại vật phẩm!"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	totalGainPrimeval := int64(0)
	totalGainImmortal := int64(0)
	guSold := false
	var removedIDs []string
	var auditSummaries []string

	addGain := func(sellRank int, gain int64) {
		if pricing.StoneTierByRank(sellRank) == "immortal" {
			totalGainImmortal += gain
		} else {
			totalGainPrimeval += gain
		}
	}

	for _, item := range req.Items {
		if item.ItemType == "Material" || item.ItemType == "Consumable" {
			preferredCat := "materials"
			if item.ItemType == "Consumable" {
				preferredCat = "consumables"
			}

			rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, player.ID.String(), item.ItemID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking inventory"})
				return
			}

			type invItem struct {
				category string
				quantity int64
			}
			var invs []invItem
			totalQty := int64(0)
			for rows.Next() {
				var iv invItem
				if err := rows.Scan(&iv.category, &iv.quantity); err != nil {
					rows.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error scanning inventory"})
					return
				}
				invs = append(invs, iv)
				totalQty += iv.quantity
			}
			rows.Close()

			if totalQty < item.Quantity {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ vật phẩm %s để bán", item.ItemID)})
				return
			}

			remainingDeduct := item.Quantity
			for _, iv := range invs {
				if iv.category == preferredCat {
					deduct := iv.quantity
					if deduct > remainingDeduct {
						deduct = remainingDeduct
					}
					if err := InventorySvc.RemoveItem(tx, player.ID.String(), item.ItemID, iv.category, deduct); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
						return
					}
					remainingDeduct -= deduct
					if remainingDeduct <= 0 {
						break
					}
				}
			}
			if remainingDeduct > 0 {
				for _, iv := range invs {
					if iv.category != preferredCat {
						deduct := iv.quantity
						if deduct > remainingDeduct {
							deduct = remainingDeduct
						}
						if err := InventorySvc.RemoveItem(tx, player.ID.String(), item.ItemID, iv.category, deduct); err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
							return
						}
						remainingDeduct -= deduct
						if remainingDeduct <= 0 {
							break
						}
					}
				}
			}

			var unitSellPrice int64
			sellRank := 1
			if shopItem, err := findCaravanShopItem(item.ItemID, item.Rank); err == nil {
				unitSellPrice = pricing.SellPrice(shopItem.Price)
				sellRank = shopItem.Rank
			} else {
				var rank int
				var dbBasePrice int64
				_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, item.ItemID).Scan(&dbBasePrice, &rank)
				sellRank = rank
				unitSellPrice = pricing.SellPrice(pricing.ItemBasePrice(dbBasePrice, rank))
			}
			gain := unitSellPrice * item.Quantity
			addGain(sellRank, gain)

			var name string
			_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, item.ItemID).Scan(&name)
			if name == "" {
				name = item.ItemID
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%d %s)", name, item.Quantity, gain, currencyLabel(sellRank)))

		} else if item.ItemType == "Gu" {
			var removed int64 = 0
			var newInv models.GuInventory
			for _, gu := range player.GuInventory {
				if gu.GuID == item.ItemID && !gu.IsVital && !gu.Equipped && gu.Status != "DEAD" && removed < item.Quantity {
					removed++
					removedIDs = append(removedIDs, gu.InstanceID)
					continue
				}
				newInv = append(newInv, gu)
			}
			if removed < item.Quantity {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Cổ trùng %s để bán hoặc Cổ là Bản Mệnh/Đang thiết bị/Đã chết", item.ItemID)})
				return
			}
			player.GuInventory = newInv
			guSold = true

			var unitSellPrice int64
			sellRank := 1
			if shopItem, err := findCaravanShopItem(item.ItemID, item.Rank); err == nil {
				unitSellPrice = pricing.SellPrice(shopItem.Price)
				sellRank = shopItem.Rank
			} else {
				var rank int
				var dbBasePrice int64
				if item.Rank > 0 {
					_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 AND rank=$2`, item.ItemID, item.Rank).Scan(&dbBasePrice, &rank)
				}
				if rank == 0 {
					_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 ORDER BY rank ASC LIMIT 1`, item.ItemID).Scan(&dbBasePrice, &rank)
				}
				sellRank = rank
				unitSellPrice = pricing.SellPrice(pricing.GuBasePrice(dbBasePrice, rank))
			}
			gain := unitSellPrice * item.Quantity
			addGain(sellRank, gain)

			var name string
			if spec, ok := models.GetGuAny(item.ItemID); ok {
				name = spec.Name
			}
			if name == "" {
				name = item.ItemID
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%d %s)", name, item.Quantity, gain, currencyLabel(sellRank)))
		} else if item.ItemType == "Recipe" {
			recipeToSell, err := RecipeSvc.Get(tx, player.ID.String(), item.ItemID)
			if err != nil || recipeToSell == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không tìm thấy Bí phương %s trong túi đồ", item.ItemID)})
				return
			}
			// Remove recipe from seller's unlocked list
			RecipeSvc.Delete(tx, player.ID.String(), item.ItemID)
			item.Quantity = 1 // Recipes are always singular

			var targetRank int
			_ = db.DB.QueryRow(`SELECT target_rank FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1`, item.ItemID).Scan(&targetRank)
			if targetRank <= 0 {
				targetRank = 1
			}

			unitSellPrice := pricing.RecipeSellPrice(targetRank)
			gain := unitSellPrice * item.Quantity
			addGain(targetRank, gain)

			name := fmt.Sprintf("Bí phương (%s)", item.ItemID)
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%d %s)", name, item.Quantity, gain, currencyLabel(targetRank)))
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item_type không hợp lệ (phải là Gu, Material, hoặc Recipe)"})
			return
		}
	}

	player.PrimevalStones += totalGainPrimeval
	player.ImmortalStones += totalGainImmortal

	player.EquippedGus = []string{}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}

	qiGained := 0.0

	if guSold {
		if err := db.RemovePlayerGus(tx, removedIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory changes"})
			return
		}
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2, equipped_gus=$3, human_qi=$4 WHERE id=$5`,
		player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save sell transaction"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	summaryStr := strings.Join(auditSummaries, ", ")
	if len(summaryStr) > 250 {
		summaryStr = summaryStr[:247] + "..."
	}

	gainSummary := fmt.Sprintf("+%d Nguyên Thạch", totalGainPrimeval)
	if totalGainImmortal > 0 {
		if totalGainPrimeval > 0 {
			gainSummary = fmt.Sprintf("+%d Nguyên Thạch và +%d Tiên Nguyên Thạch", totalGainPrimeval, totalGainImmortal)
		} else {
			gainSummary = fmt.Sprintf("+%d Tiên Nguyên Thạch", totalGainImmortal)
		}
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "shop.sell_bulk",
		EntityType:     "shop",
		RequestSummary: fmt.Sprintf("Bán sỉ %d loại cho Tiệm Buôn: %s | Tổng nhận: %s", len(req.Items), summaryStr, gainSummary),
		Metadata: map[string]interface{}{
			"item_count":          len(req.Items),
			"total_gain_primeval": totalGainPrimeval,
			"total_gain_immortal": totalGainImmortal,
			"summaries":           auditSummaries,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	notify.Send(db.DB, authID, "shop_sell", "Bán Hàng Tiệm Buôn",
		fmt.Sprintf("Đã bán sỉ %d loại vật phẩm/Cổ trùng, thu về %s.", len(req.Items), gainSummary))

	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Bán thành công %d loại vật phẩm, nhận %s! Nhân Khí +%.0f.", len(req.Items), gainSummary, qiGained)})
}
