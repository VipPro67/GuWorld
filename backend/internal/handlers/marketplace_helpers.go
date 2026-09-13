package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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

func getUpfrontFee(listingType string, durationHours int, totalListedPrice int64) int64 {
	if listingType == "MARKET" {
		return int64(math.Round(float64(totalListedPrice) * 0.04))
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
