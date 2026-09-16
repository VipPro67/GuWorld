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
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

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

func settleAuction(auctionID string, sellerID uuid.UUID, highestBidderID sql.NullString, itemType, itemData string, qty int64, currentPrice float64, currency string, listingType string, durationHours int) error {
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
		commission := math.Round(currentPrice*getCommissionPct(listingType, durationHours)*1000) / 1000
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
				if gu.InstanceID == "" {
					gu.InstanceID = uuid.New().String()
				}
				gu.Status = "SLEEPING"
				gu.Equipped = false
				gu.IsVital = false
				player.GuInventory = append(player.GuInventory, gu)

				guName := gu.GuID
				if guSpec, ok := models.GetGu(gu.GuID, gu.Rank); ok && guSpec.Name != "" {
					guName = guSpec.Name
				}
				summary := fmt.Sprintf("Cổ Sư [%s] nhận Cổ trùng [%s] R%d từ Bảo Hoàng Thiên.", player.Username, guName, gu.Rank)
				_ = db.LogGuHistory(tx, gu.InstanceID, gu.GuID, gu.Rank, "MARKET_SOLD", nil, &player.ID, summary, map[string]interface{}{
					"gu_name": guName,
				})
			}
		} else {
			// Tương thích ngược với định dạng cũ (chỉ lưu string guID)
			for i := int64(0); i < qty; i++ {
				g := mapGuFromRaw(itemData)
				player.GuInventory = append(player.GuInventory, g)
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
