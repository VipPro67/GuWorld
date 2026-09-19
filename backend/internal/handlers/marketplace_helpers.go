package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

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

func getUpfrontFee(listingType string, durationHours int, totalListedPrice float64) float64 {
	if listingType == "MARKET" {
		return pricing.Quantize(totalListedPrice * 0.04)
	}
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
	return 0
}

func getCommissionPct(listingType string, durationHours int) float64 {
	if listingType == "MARKET" {
		return 0.06
	}
	if listingType == "AUCTION" {
		return 0.15
	}
	return 0.0
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

// HasBaoHoangAccess checks if player inventory has both GU_CONNECTING_HEAVEN and GU_DIVINE_SENSE alive
func HasBaoHoangAccess(inventory models.GuInventory) bool {
	hasConnectingHeaven := false
	hasDivineSense := false
	for _, g := range inventory {
		if g.Status == "DEAD" {
			continue
		}
		if g.GuID == "GU_CONNECTING_HEAVEN" {
			hasConnectingHeaven = true
		}
		if g.GuID == "GU_DIVINE_SENSE" {
			hasDivineSense = true
		}
	}
	if !hasConnectingHeaven {
		_, _, hasConnectingHeaven = models.FindFirstMatchingGu(inventory, func(gu models.GuInstance) bool {
			return models.InventoryHasGuMechanic(models.GuInventory{gu}, "MARKETPLACE_ACCESS", false)
		})
	}
	return hasConnectingHeaven && hasDivineSense
}

// StartAuctionSettlementWorker runs a background loop to periodically settle expired auctions, buyback orders, and trade offers.
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
				if err := settleExpiredBuybackOrders(); err != nil {
					logger.S.Errorf("Failed to settle expired buyback orders: %v", err)
				}
				if err := settleExpiredTradeOffers(); err != nil {
					logger.S.Errorf("Failed to settle expired trade offers: %v", err)
				}
			}
		}
	}()
}

func settleExpiredBuybackOrders() error {
	rows, err := db.DB.Query(`SELECT id, buyer_id, unit_price, quantity_requested, quantity_filled, currency FROM buyback_orders WHERE status = 'ACTIVE' AND end_time <= NOW()`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type expiredBuyback struct {
		id                uuid.UUID
		buyerID           uuid.UUID
		unitPrice         float64
		quantityRequested int64
		quantityFilled    int64
		currency          string
	}
	var list []expiredBuyback
	for rows.Next() {
		var eb expiredBuyback
		if err := rows.Scan(&eb.id, &eb.buyerID, &eb.unitPrice, &eb.quantityRequested, &eb.quantityFilled, &eb.currency); err == nil {
			list = append(list, eb)
		}
	}

	for _, eb := range list {
		unfilledQty := eb.quantityRequested - eb.quantityFilled
		if unfilledQty <= 0 {
			db.DB.Exec(`UPDATE buyback_orders SET status = 'FILLED' WHERE id = $1`, eb.id)
			continue
		}
		refundAmount := pricing.Quantize(float64(unfilledQty) * eb.unitPrice)
		tx, err := db.DB.Begin()
		if err != nil {
			continue
		}
		var currentStatus string
		if err := tx.QueryRow(`SELECT status FROM buyback_orders WHERE id = $1 FOR UPDATE`, eb.id).Scan(&currentStatus); err != nil || currentStatus != "ACTIVE" {
			tx.Rollback()
			continue
		}
		if eb.currency == "immortal" {
			_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, refundAmount, eb.buyerID)
		} else {
			_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, refundAmount, eb.buyerID)
		}
		_, _ = tx.Exec(`UPDATE buyback_orders SET status = 'EXPIRED' WHERE id = $1`, eb.id)
		tx.Commit()
	}
	return nil
}

func settleExpiredTradeOffers() error {
	rows, err := db.DB.Query(`SELECT id, creator_id, offered_data FROM trade_offers WHERE status = 'ACTIVE' AND end_time <= NOW()`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type expiredOffer struct {
		id          uuid.UUID
		creatorID   uuid.UUID
		offeredJSON []byte
	}
	var list []expiredOffer
	for rows.Next() {
		var eo expiredOffer
		if err := rows.Scan(&eo.id, &eo.creatorID, &eo.offeredJSON); err == nil {
			list = append(list, eo)
		}
	}

	for _, eo := range list {
		var offer models.TradeOfferData
		if err := json.Unmarshal(eo.offeredJSON, &offer); err != nil {
			continue
		}
		tx, err := db.DB.Begin()
		if err != nil {
			continue
		}
		var currentStatus string
		if err := tx.QueryRow(`SELECT status FROM trade_offers WHERE id = $1 FOR UPDATE`, eo.id).Scan(&currentStatus); err != nil || currentStatus != "ACTIVE" {
			tx.Rollback()
			continue
		}
		if err := restoreTradeOfferItems(tx, eo.creatorID, offer); err != nil {
			tx.Rollback()
			continue
		}
		_, _ = tx.Exec(`UPDATE trade_offers SET status = 'EXPIRED' WHERE id = $1`, eo.id)
		tx.Commit()
	}
	return nil
}

