package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// deathPenaltyConfig — mức phạt khi chết (áp vào túi thật).
// Chết PvE rơi vào hư không; chết PvP rơi cho kẻ thắng.
// Chỉnh trực tiếp các giá trị bên dưới để tùy biến.
var deathPenaltyConfig = struct {
	// Tiền (Nguyên Thạch / Tiên Nguyên Thạch) — % hao hụt khi chết
	PrimevalStonePct float64
	ImmortalStonePct float64

	// Vật phẩm (nguyên liệu trong túi thật)
	ItemsEnabled    bool
	ItemsTypePct    float64
	ItemsMaxPerType int

	// Cổ trùng (luôn loại trừ Cổ đang trang bị & Cổ Bản Mệnh)
	GusEnabled  bool
	GusPct      float64
	GusMaxCount int

	// Bí phương (player_recipe_progress)
	RecipesEnabled  bool
	RecipesPct      float64
	RecipesMaxCount int
}{
	PrimevalStonePct: 10,
	ImmortalStonePct: 10,

	ItemsEnabled:    true,
	ItemsTypePct:    10,
	ItemsMaxPerType: 10,

	GusEnabled:  true,
	GusPct:      10,
	GusMaxCount: 10,

	RecipesEnabled:  true,
	RecipesPct:      10,
	RecipesMaxCount: 10,
}

// applyDeathDrop applies the death penalty to a fully-loaded in-memory loser.
// The loser's PrimevalStones / ImmortalStones / GuInventory are mutated so the
// caller's subsequent full-row write stays consistent; all DB-side effects
// (winner gains, gu deletes, materials, recipes) are performed here directly.
func applyDeathDrop(tx *sql.Tx, loser *models.Player, winnerID *uuid.UUID) string {
	remainingGus, droppedGus := selectEligibleGus(loser.GuInventory)
	loser.GuInventory = remainingGus

	primevalLoss := math.Floor(loser.PrimevalStones * deathPenaltyConfig.PrimevalStonePct / 100)
	immortalLoss := math.Floor(loser.ImmortalStones * deathPenaltyConfig.ImmortalStonePct / 100)
	if primevalLoss < 0 {
		primevalLoss = 0
	}
	if immortalLoss < 0 {
		immortalLoss = 0
	}
	loser.PrimevalStones -= primevalLoss
	loser.ImmortalStones -= immortalLoss

	return persistDeathDrop(tx, loser.ID, winnerID, primevalLoss, immortalLoss, droppedGus)
}

// applyDeathDropRecord applies the death penalty to a player record that is not
// fully loaded in memory (the PvP defender on the winner's side). All reads and
// writes happen via SQL within the current transaction.
func applyDeathDropRecord(tx *sql.Tx, loserID, winnerID *uuid.UUID) string {
	var primevalStones, immortalStones float64
	err := tx.QueryRow(`
		SELECT primeval_stones, immortal_stones
		FROM players WHERE id = $1 FOR UPDATE
	`, *loserID).Scan(&primevalStones, &immortalStones)
	if err != nil {
		return ""
	}

	primevalLoss := math.Floor(primevalStones * deathPenaltyConfig.PrimevalStonePct / 100)
	immortalLoss := math.Floor(immortalStones * deathPenaltyConfig.ImmortalStonePct / 100)
	if primevalLoss < 0 {
		primevalLoss = 0
	}
	if immortalLoss < 0 {
		immortalLoss = 0
	}

	allGus, _ := db.LoadPlayerGus(tx, loserID.String(), false)
	_, droppedGus := selectEligibleGus(allGus)

	return persistDeathDrop(tx, *loserID, winnerID, primevalLoss, immortalLoss, droppedGus)
}

// persistDeathDrop writes the computed losses to the DB and builds the summary.
func persistDeathDrop(tx *sql.Tx, loserID uuid.UUID, winnerID *uuid.UUID, primevalLoss, immortalLoss float64, droppedGus models.GuInventory) string {
	var lost []string

	// Tiền — trừ bên thua, cộng bên thắng (nếu PvP)
	if primevalLoss >= 1 || immortalLoss >= 1 {
		if _, err := tx.Exec(`
			UPDATE players SET
				primeval_stones = GREATEST(0, primeval_stones - $1),
				immortal_stones = GREATEST(0, immortal_stones - $2)
			WHERE id = $3
		`, primevalLoss, immortalLoss, loserID); err != nil {
			return ""
		}
		if winnerID != nil {
			_, _ = tx.Exec(`
				UPDATE players SET
					primeval_stones = primeval_stones + $1,
					immortal_stones = immortal_stones + $2
				WHERE id = $3
			`, primevalLoss, immortalLoss, *winnerID)
		}
		if primevalLoss >= 1 {
			lost = append(lost, fmt.Sprintf("%.0f Nguyên Thạch", primevalLoss))
		}
		if immortalLoss >= 1 {
			lost = append(lost, fmt.Sprintf("%.0f Tiên Nguyên Thạch", immortalLoss))
		}
	}

	// Vật phẩm (nguyên liệu túi thật)
	if deathPenaltyConfig.ItemsEnabled {
		lost = append(lost, dropMaterials(tx, loserID, winnerID)...)
	}

	// Cổ trùng
	if deathPenaltyConfig.GusEnabled && len(droppedGus) > 0 {
		ids := make([]string, 0, len(droppedGus))
		for _, g := range droppedGus {
			ids = append(ids, g.InstanceID)
			name := g.GuID
			if guDef, ok := models.GetGuAny(g.GuID); ok && guDef.Name != "" {
				name = guDef.Name
			}
			lost = append(lost, fmt.Sprintf("Cổ trùng %s", name))
		}
		_ = db.RemovePlayerGus(tx, ids)
		if winnerID != nil {
			for _, g := range droppedGus {
				_ = db.AddPlayerGu(tx, winnerID.String(), g)
			}
		}
	}

	// Bí phương
	if deathPenaltyConfig.RecipesEnabled {
		lost = append(lost, dropRecipes(tx, loserID, winnerID)...)
	}

	if len(lost) == 0 {
		return ""
	}
	return joinDropList(lost)
}

// selectEligibleGus returns the remaining inventory and the randomly chosen,
// non-equipped, non-vital Gus to drop.
func selectEligibleGus(gus models.GuInventory) (models.GuInventory, models.GuInventory) {
	eligible := make(models.GuInventory, 0, len(gus))
	for _, g := range gus {
		if g.Equipped || g.IsVital {
			continue
		}
		eligible = append(eligible, g)
	}
	if !deathPenaltyConfig.GusEnabled || len(eligible) == 0 {
		return gus, models.GuInventory{}
	}

	count := int(math.Ceil(float64(len(eligible)) * deathPenaltyConfig.GusPct / 100))
	if count > deathPenaltyConfig.GusMaxCount {
		count = deathPenaltyConfig.GusMaxCount
	}
	if count > len(eligible) {
		count = len(eligible)
	}

	rand.Shuffle(len(eligible), func(i, j int) { eligible[i], eligible[j] = eligible[j], eligible[i] })
	dropped := eligible[:count]

	dropIDs := make(map[string]bool, len(dropped))
	for _, g := range dropped {
		dropIDs[g.InstanceID] = true
	}
	remaining := make(models.GuInventory, 0, len(gus))
	for _, g := range gus {
		if dropIDs[g.InstanceID] {
			continue
		}
		remaining = append(remaining, g)
	}
	return remaining, dropped
}

// dropMaterials samples a portion of the loser's real inventory and moves it to
// the winner (or discards it). Returns human-readable lost lines.
func dropMaterials(tx *sql.Tx, loserID uuid.UUID, winnerID *uuid.UUID) []string {
	rows, err := tx.Query(`
		SELECT item_id, category, quantity
		FROM player_inventory_items
		WHERE player_id = $1 AND quantity > 0
	`, loserID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	type row struct {
		itemID   string
		category string
		quantity int
	}
	var items []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.itemID, &r.category, &r.quantity); err != nil {
			continue
		}
		items = append(items, r)
	}

	if len(items) == 0 {
		return nil
	}

	pickCount := int(math.Ceil(float64(len(items)) * deathPenaltyConfig.ItemsTypePct / 100))
	if pickCount > len(items) {
		pickCount = len(items)
	}

	rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })

	var lost []string
	for _, r := range items[:pickCount] {
		qty := 1 + rand.Intn(deathPenaltyConfig.ItemsMaxPerType)
		if qty > r.quantity {
			qty = r.quantity
		}
		if qty <= 0 {
			continue
		}
		if err := InventorySvc.RemoveItem(tx, loserID.String(), r.itemID, r.category, qty); err != nil {
			continue
		}
		if winnerID != nil {
			_ = InventorySvc.AddItem(tx, winnerID.String(), r.itemID, r.category, qty)
		}
		name := r.itemID
		if spec, ok := models.ItemRegistry[r.itemID]; ok && spec.Name != "" {
			name = spec.Name
		}
		lost = append(lost, fmt.Sprintf("%dx %s", qty, name))
	}
	return lost
}

// dropRecipes samples a portion of the loser's unlocked recipes and moves them
// to the winner (or discards them). Returns human-readable lost lines.
func dropRecipes(tx *sql.Tx, loserID uuid.UUID, winnerID *uuid.UUID) []string {
	rows, err := tx.Query(`
		SELECT recipe_id FROM player_recipe_progress
		WHERE player_id = $1 AND is_unlocked = true
	`, loserID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var recipeIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		recipeIDs = append(recipeIDs, id)
	}
	if len(recipeIDs) == 0 {
		return nil
	}

	pickCount := int(math.Ceil(float64(len(recipeIDs)) * deathPenaltyConfig.RecipesPct / 100))
	if pickCount > deathPenaltyConfig.RecipesMaxCount {
		pickCount = deathPenaltyConfig.RecipesMaxCount
	}
	if pickCount > len(recipeIDs) {
		pickCount = len(recipeIDs)
	}

	rand.Shuffle(len(recipeIDs), func(i, j int) { recipeIDs[i], recipeIDs[j] = recipeIDs[j], recipeIDs[i] })

	var lost []string
	for _, id := range recipeIDs[:pickCount] {
		existing, _ := RecipeSvc.Get(tx, loserID.String(), id)
		if existing == nil {
			continue
		}
		if err := RecipeSvc.Delete(tx, loserID.String(), id); err != nil {
			continue
		}
		if winnerID != nil {
			existing.PlayerID = winnerID.String()
			_ = RecipeSvc.Upsert(tx, existing)
		}
		name := id
		if guDef, ok := models.GetGuAny(id); ok && guDef.Name != "" {
			name = guDef.Name
		}
		lost = append(lost, fmt.Sprintf("Bí phương %s", name))
	}
	return lost
}

func joinDropList(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
