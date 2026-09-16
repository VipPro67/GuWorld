package handlers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
)

type TradeOffer struct {
	PrimevalStones float64            `json:"primeval_stones"`
	ImmortalStones float64            `json:"immortal_stones"`
	Gus            []string           `json:"gus"`       // slice of Gu instance IDs
	Materials      map[string]int64   `json:"materials"` // map of itemID -> quantity
	Recipes        map[string]float64 `json:"recipes"`   // map of recipeID -> completion_rate
}

func getGuName(guID string, rank int) string {
	if spec, ok := models.GetGu(guID, rank); ok {
		return spec.Name
	}
	return guID
}
func getMaterialName(matID string) string {
	if spec, ok := models.ItemRegistry[matID]; ok {
		return spec.Name
	}
	return matID
}

func getRecipeName(recipeID string) string {
	var targetGuID string
	if tg, _, _, err := ParseRecipeID(recipeID); err == nil {
		targetGuID = tg
	} else {
		_ = db.DB.QueryRow(`SELECT target_gu_id FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1`, recipeID).Scan(&targetGuID)
	}
	if targetGuID != "" {
		if spec, ok := models.GetGuAny(targetGuID); ok {
			return fmt.Sprintf("Bí phương %s", spec.Name)
		}
	}
	return fmt.Sprintf("Bí phương %s", recipeID)
}

// formatGuNames returns a human-readable list of Gu names for a set of instance IDs.
func formatGuNames(q db.Queryer, instanceIDs []string) string {
	if len(instanceIDs) == 0 {
		return ""
	}
	var names []string
	for _, id := range instanceIDs {
		var guID string
		var rank int
		if err := q.QueryRow(`SELECT gu_id, gu_rank FROM player_gu WHERE id = $1`, id).Scan(&guID, &rank); err != nil {
			names = append(names, id)
			continue
		}
		names = append(names, getGuName(guID, rank))
	}
	return strings.Join(names, ", ")
}

func getGuNamesFromInstances(tx *sql.Tx, instIDs []string) []string {
	if len(instIDs) == 0 {
		return nil
	}
	names := []string{}
	for _, instID := range instIDs {
		var guID string
		var rank int
		err := tx.QueryRow(`SELECT gu_id, rank FROM player_gu WHERE id = $1`, instID).Scan(&guID, &rank)
		if err == nil {
			names = append(names, fmt.Sprintf("%s (Hạng %d)", getGuName(guID, rank), rank))
		} else {
			names = append(names, instID)
		}
	}
	return names
}

func formatOffer(tx *sql.Tx, offer TradeOffer) string {
	var parts []string
	if offer.PrimevalStones > 0 {
		parts = append(parts, fmt.Sprintf("%.3f Nguyên Thạch", offer.PrimevalStones))
	}
	if offer.ImmortalStones > 0 {
		parts = append(parts, fmt.Sprintf("%.3f Tiên Nguyên Thạch", offer.ImmortalStones))
	}
	if len(offer.Gus) > 0 {
		guNames := getGuNamesFromInstances(tx, offer.Gus)
		parts = append(parts, fmt.Sprintf("Cổ trùng: %s", strings.Join(guNames, ", ")))
	}
	if len(offer.Materials) > 0 {
		var matParts []string
		for k, v := range offer.Materials {
			matParts = append(matParts, fmt.Sprintf("%s x%d", getMaterialName(k), v))
		}
		parts = append(parts, fmt.Sprintf("Vật phẩm: %s", strings.Join(matParts, ", ")))
	}
	if len(offer.Recipes) > 0 {
		var recParts []string
		for k, v := range offer.Recipes {
			recParts = append(recParts, fmt.Sprintf("%s (%.1f%%)", getRecipeName(k), v))
		}
		parts = append(parts, fmt.Sprintf("Bí phương: %s", strings.Join(recParts, ", ")))
	}
	if len(parts) == 0 {
		return "Trống"
	}
	return strings.Join(parts, ", ")
}

func getTradeParticipantsWithUsernames(tradeID string) (senderID uuid.UUID, senderName string, receiverID uuid.UUID, receiverName string, err error) {
	err = db.DB.QueryRow(`
		SELECT t.sender_id, p1.username, t.receiver_id, p2.username
		FROM trades t
		JOIN players p1 ON t.sender_id = p1.id
		JOIN players p2 ON t.receiver_id = p2.id
		WHERE t.id = $1
	`, tradeID).Scan(&senderID, &senderName, &receiverID, &receiverName)
	return
}

type InitiateTradeRequest struct {
	ReceiverUsername string `json:"receiver_username" binding:"required"`
}

// transferRecipe moves a recipe from one player to another in a trade.
// The receiver keeps the higher completion rate, capped at 100%.
func transferRecipe(tx *sql.Tx, fromPlayerID, toPlayerID, recipeID string, incomingRate float64) error {
	if _, err := RecipeSvc.Get(tx, fromPlayerID, recipeID); err != nil {
		return fmt.Errorf("bên gửi không sở hữu Bí phương %s", recipeID)
	}
	if err := RecipeSvc.Delete(tx, fromPlayerID, recipeID); err != nil {
		return fmt.Errorf("không thể xóa Bí phương %s từ bên gửi", recipeID)
	}
	var finalRate float64 = incomingRate
	if existing, err := RecipeSvc.Get(tx, toPlayerID, recipeID); err == nil && existing != nil {
		if existing.CompletionRate > finalRate {
			finalRate = existing.CompletionRate
		}
	}
	if finalRate > 100.0 {
		finalRate = 100.0
	}
	return RecipeSvc.Upsert(tx, &services.RecipeProgress{
		PlayerID:       toPlayerID,
		RecipeID:       recipeID,
		CompletionRate: finalRate,
		IsUnlocked:     true,
	})
}
