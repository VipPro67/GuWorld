package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
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
	PrimevalStonePct: 20,
	ImmortalStonePct: 20,

	ItemsEnabled:    true,
	ItemsTypePct:    20,
	ItemsMaxPerType: 20,

	GusEnabled:  true,
	GusPct:      20,
	GusMaxCount: 20,

	RecipesEnabled:  true,
	RecipesPct:      20,
	RecipesMaxCount: 20,
}

// AddSatKhiStatus adds or increments Sát Khí status (sat_khi) for a PvP killer.
func AddSatKhiStatus(tx *sql.Tx, killerID *uuid.UUID) {
	if killerID == nil {
		return
	}
	var rawStatuses []byte
	err := tx.QueryRow(`SELECT statuses FROM players WHERE id = $1 FOR UPDATE`, *killerID).Scan(&rawStatuses)
	if err != nil {
		return
	}

	var sl models.StatusList
	if len(rawStatuses) > 0 {
		_ = json.Unmarshal(rawStatuses, &sl)
	}

	currentLevel := 0
	for _, st := range sl {
		if st.ID == "sat_khi" {
			if st.Value > 0 {
				currentLevel = int(st.Value)
			} else {
				currentLevel = 1
			}
		}
	}

	newLevel := currentLevel + 1 // Bỏ giới hạn cấp Sát Khí (tăng liên tục theo số mạng đồ sát)

	sl.Add("sat_khi", "pvp_kill", 2*time.Hour, float64(newLevel))
	newBytes, _ := json.Marshal(sl)
	_, _ = tx.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, newBytes, *killerID)

	notify.Send(tx, *killerID, "sat_khi_increase", "Sát Khí gia tăng!",
		fmt.Sprintf("Tích tụ oán khí đồ sát! Sát Khí của bạn tăng lên Tầng %d. Nếu bị đánh bại, lượng tài sản đánh rơi sẽ gia tăng x%.1f!", newLevel, 1.0+float64(newLevel)*0.5))
}

// getSatKhiMultiplier checks if player has sat_khi status and returns (multiplier, level).
func getSatKhiMultiplier(tx *sql.Tx, playerID uuid.UUID) (float64, int) {
	var rawStatuses []byte
	err := tx.QueryRow(`SELECT statuses FROM players WHERE id = $1`, playerID).Scan(&rawStatuses)
	if err != nil || len(rawStatuses) == 0 {
		return 1.0, 0
	}

	var sl models.StatusList
	if err := json.Unmarshal(rawStatuses, &sl); err != nil {
		return 1.0, 0
	}

	active := sl.Active()
	for _, st := range active {
		if st.ID == "sat_khi" {
			level := int(st.Value)
			if level <= 0 {
				level = 1
			}
			mult := 1.0 + float64(level)*0.5
			return mult, level
		}
	}

	return 1.0, 0
}

// reduceSatKhiOnDeath decrements or removes sat_khi status after dying.
func reduceSatKhiOnDeath(tx *sql.Tx, playerID uuid.UUID) {
	var rawStatuses []byte
	err := tx.QueryRow(`SELECT statuses FROM players WHERE id = $1 FOR UPDATE`, playerID).Scan(&rawStatuses)
	if err != nil || len(rawStatuses) == 0 {
		return
	}

	var sl models.StatusList
	if err := json.Unmarshal(rawStatuses, &sl); err != nil {
		return
	}

	hasSatKhi := false
	currentLevel := 0
	for _, st := range sl {
		if st.ID == "sat_khi" {
			hasSatKhi = true
			currentLevel = int(st.Value)
			if currentLevel <= 0 {
				currentLevel = 1
			}
		}
	}

	if !hasSatKhi {
		return
	}

	if currentLevel > 1 {
		sl.Add("sat_khi", "pvp_death", 2*time.Hour, float64(currentLevel-1))
	} else {
		sl.Remove("sat_khi")
	}

	newBytes, _ := json.Marshal(sl)
	_, _ = tx.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, newBytes, playerID)
}

type DeathPos struct {
	Region string
	X      int
	Y      int
}

// applyDeathDrop applies the death penalty to a fully-loaded in-memory loser.
func applyDeathDrop(tx *sql.Tx, loser *models.Player, winnerID *uuid.UUID, isAggressor bool, deathPos ...DeathPos) string {
	mult, satKhiLevel := getSatKhiMultiplier(tx, loser.ID)
	remainingGus, droppedGus := selectEligibleGus(loser.GuInventory, mult, satKhiLevel)
	loser.GuInventory = remainingGus

	effectiveStonePct := math.Min(100.0, deathPenaltyConfig.PrimevalStonePct*mult)
	primevalLoss := int64(math.Floor(float64(loser.PrimevalStones) * effectiveStonePct / 100))
	effectiveImmortalPct := math.Min(100.0, deathPenaltyConfig.ImmortalStonePct*mult)
	immortalLoss := int64(math.Floor(float64(loser.ImmortalStones) * effectiveImmortalPct / 100))
	if primevalLoss < 0 {
		primevalLoss = 0
	}
	if immortalLoss < 0 {
		immortalLoss = 0
	}
	if primevalLoss > loser.PrimevalStones {
		primevalLoss = loser.PrimevalStones
	}
	if immortalLoss > loser.ImmortalStones {
		immortalLoss = loser.ImmortalStones
	}
	loser.PrimevalStones -= primevalLoss
	loser.ImmortalStones -= immortalLoss

	dp := deathPos
	if len(dp) == 0 && loser.CurrentRegion != "" {
		dp = []DeathPos{{Region: loser.CurrentRegion, X: loser.X, Y: loser.Y}}
	}

	return persistDeathDrop(tx, loser.ID, winnerID, isAggressor, primevalLoss, immortalLoss, droppedGus, mult, satKhiLevel, dp...)
}

// applyDeathDropRecord applies the death penalty to a player record that is not fully loaded in memory.
func applyDeathDropRecord(tx *sql.Tx, loserID, winnerID *uuid.UUID, isAggressor bool, deathPos ...DeathPos) string {
	var primevalStones, immortalStones int64
	err := tx.QueryRow(`
		SELECT primeval_stones, immortal_stones
		FROM players WHERE id = $1 FOR UPDATE
	`, *loserID).Scan(&primevalStones, &immortalStones)
	if err != nil {
		return ""
	}

	mult, satKhiLevel := getSatKhiMultiplier(tx, *loserID)

	effectiveStonePct := math.Min(100.0, deathPenaltyConfig.PrimevalStonePct*mult)
	primevalLoss := int64(math.Floor(float64(primevalStones) * effectiveStonePct / 100))
	effectiveImmortalPct := math.Min(100.0, deathPenaltyConfig.ImmortalStonePct*mult)
	immortalLoss := int64(math.Floor(float64(immortalStones) * effectiveImmortalPct / 100))
	if primevalLoss < 0 {
		primevalLoss = 0
	}
	if immortalLoss < 0 {
		immortalLoss = 0
	}
	if primevalLoss > primevalStones {
		primevalLoss = primevalStones
	}
	if immortalLoss > immortalStones {
		immortalLoss = immortalStones
	}

	allGus, _ := db.LoadPlayerGus(tx, loserID.String(), false)
	_, droppedGus := selectEligibleGus(allGus, mult, satKhiLevel)

	return persistDeathDrop(tx, *loserID, winnerID, isAggressor, primevalLoss, immortalLoss, droppedGus, mult, satKhiLevel, deathPos...)
}

type DroppedItem struct {
	ItemID   string `json:"item_id"`
	Category string `json:"category"`
	Quantity int    `json:"quantity"`
}

type GroundLootData struct {
	PrimevalStones int64              `json:"primeval_stones,omitempty"`
	ImmortalStones int64              `json:"immortal_stones,omitempty"`
	Items          []DroppedItem      `json:"items,omitempty"`
	Gus            models.GuInventory `json:"gus,omitempty"`
	Recipes        []string           `json:"recipes,omitempty"`
}

// persistDeathDrop writes the computed losses to DB, creates a map_ground_drops record, and builds summary.
func persistDeathDrop(tx *sql.Tx, loserID uuid.UUID, winnerID *uuid.UUID, isAggressor bool, primevalLoss, immortalLoss int64, droppedGus models.GuInventory, mult float64, satKhiLevel int, deathPos ...DeathPos) string {
	var lost []string
	lootData := GroundLootData{
		PrimevalStones: primevalLoss,
		ImmortalStones: immortalLoss,
		Gus:            droppedGus,
	}

	// Tiền — trừ bên thua
	if primevalLoss >= 1 || immortalLoss >= 1 {
		if _, err := tx.Exec(`
			UPDATE players SET
				primeval_stones = GREATEST(0, primeval_stones - $1),
				immortal_stones = GREATEST(0, immortal_stones - $2)
			WHERE id = $3
		`, primevalLoss, immortalLoss, loserID); err != nil {
			return ""
		}
		if primevalLoss >= 1 {
			lost = append(lost, fmt.Sprintf("%d Nguyên Thạch", primevalLoss))
		}
		if immortalLoss >= 1 {
			lost = append(lost, fmt.Sprintf("%d Tiên Nguyên Thạch", immortalLoss))
		}
	}

	// Vật phẩm (nguyên liệu túi thật)
	if deathPenaltyConfig.ItemsEnabled {
		matItems, matLogs := dropMaterials(tx, loserID, mult, satKhiLevel)
		lootData.Items = matItems
		lost = append(lost, matLogs...)
	}

	var loserRegion sql.NullString
	var loserX, loserY int
	var loserName string

	if len(deathPos) > 0 && deathPos[0].Region != "" {
		loserRegion.String = deathPos[0].Region
		loserRegion.Valid = true
		loserX = deathPos[0].X
		loserY = deathPos[0].Y
		_ = tx.QueryRow(`SELECT username FROM players WHERE id = $1`, loserID).Scan(&loserName)
	} else {
		_ = tx.QueryRow(`SELECT current_region, x, y, username FROM players WHERE id = $1`, loserID).Scan(&loserRegion, &loserX, &loserY, &loserName)
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
			deathSummary := fmt.Sprintf("Cổ Sư [%s] tử đạo, đánh rơi Cổ trùng [%s] R%d xuống đất.", loserName, name, g.Rank)
			_ = db.LogGuHistory(tx, g.InstanceID, g.GuID, g.Rank, "DEATH_DROPPED", &loserID, winnerID, deathSummary, map[string]interface{}{
				"gu_name": name,
			})
		}
		_ = db.RemovePlayerGus(tx, ids)
	}

	// Bí phương
	if deathPenaltyConfig.RecipesEnabled {
		recipeIDs, recipeLogs := dropRecipes(tx, loserID, mult, satKhiLevel)
		lootData.Recipes = recipeIDs
		lost = append(lost, recipeLogs...)
	}

	// Nếu người thắng là kẻ chủ động tấn công (aggressor), tăng Sát Khí cho kẻ chiến thắng
	if winnerID != nil && isAggressor {
		AddSatKhiStatus(tx, winnerID)
	}

	// Nếu người thua có Sát Khí, giảm 1 tầng Sát Khí sau khi trả nợ máu bằng tử đạo
	if satKhiLevel > 0 {
		reduceSatKhiOnDeath(tx, loserID)
	}

	var killerName string
	if winnerID != nil {
		_ = tx.QueryRow(`SELECT username FROM players WHERE id = $1`, *winnerID).Scan(&killerName)
	}

	summary := ""
	if len(lost) > 0 {
		summary = joinDropList(lost)
		if satKhiLevel > 0 {
			summary += fmt.Sprintf(" [Oán Khí Tầng %d: Mất đồ x%.1f]", satKhiLevel, mult)
		}

		if loserRegion.Valid && loserRegion.String != "" && !IsSafeZone(loserRegion.String, loserX, loserY) {
			lootBytes, _ := json.Marshal(lootData)
			_, _ = tx.Exec(`
				INSERT INTO map_ground_drops (
					region_id, x, y, loser_id, loser_name, killer_id, killer_name,
					loot_data, protected_until, expires_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7,
					$8, NOW() + INTERVAL '3 minutes', NOW() + INTERVAL '30 minutes'
				)
			`, loserRegion.String, loserX, loserY, loserID, loserName, winnerID, killerName, lootBytes)
		}
	}

	// Notify loser and (PvP) winner — best-effort, non-fatal.
	if winnerID != nil {
		attackerName := killerName
		if attackerName == "" {
			attackerName = "Kẻ thù"
		}
		if summary != "" {
			notify.Send(tx, loserID, "death_drop", "Tử đạo - Bị hạ gục",
				fmt.Sprintf("Bạn đã bị Cổ Sư [%s] hạ gục tại (%d, %d) và rơi rớt tài sản: %s. Bạn đã được truyền tống về Vùng An Toàn!", attackerName, loserX, loserY, summary))
			notify.Send(tx, *winnerID, "loot_gained", "Kẻ thù bị hạ gục",
				fmt.Sprintf("Bạn đã hạ gục Cổ Sư [%s]! Tài sản của đối phương rơi tại (%d, %d): %s.", loserName, loserX, loserY, summary))
		} else {
			notify.Send(tx, loserID, "death_drop", "Tử đạo - Bị hạ gục",
				fmt.Sprintf("Bạn đã bị Cổ Sư [%s] hạ gục tại (%d, %d). May mắn không bị rơi rớt tài sản! Bạn đã được truyền tống về Vùng An Toàn!", attackerName, loserX, loserY))
			notify.Send(tx, *winnerID, "loot_gained", "Kẻ thù bị hạ gục",
				fmt.Sprintf("Bạn đã hạ gục Cổ Sư [%s] tại (%d, %d)! Đối phương không có tài sản rơi rớt.", loserName, loserX, loserY))
		}
	} else if summary != "" {
		notify.Send(tx, loserID, "death_drop", "Tử đạo - Mất tài sản",
			fmt.Sprintf("Bạn đã tử đạo và đánh rơi tài sản tại (%d, %d): %s.", loserX, loserY, summary))
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &loserID,
		Action:         "combat.death_penalty",
		EntityType:     "player",
		EntityID:       loserID.String(),
		RequestSummary: fmt.Sprintf("Tử đạo đánh rơi tài sản: %s", summary),
		Metadata: map[string]interface{}{
			"lost_items": lost,
			"winner_id":  winnerID,
		},
		Success: true,
	})

	return summary
}

// selectEligibleGus returns the remaining inventory and the randomly chosen,
// non-equipped, non-vital Gus to drop.
func selectEligibleGus(gus models.GuInventory, mult float64, satKhiLevel int) (models.GuInventory, models.GuInventory) {
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

	effectivePct := math.Min(100.0, deathPenaltyConfig.GusPct*mult)

	count := int(math.Ceil(float64(len(eligible)) * effectivePct / 100.0))
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

// dropMaterials samples a portion of the loser's real inventory by percentage and removes it in batch. Returns dropped items and human-readable lost lines.
func dropMaterials(tx *sql.Tx, loserID uuid.UUID, mult float64, satKhiLevel int) ([]DroppedItem, []string) {
	rows, err := tx.Query(`
		SELECT item_id, category, quantity
		FROM player_inventory_items
		WHERE player_id = $1 AND quantity > 0
	`, loserID)
	if err != nil {
		return nil, nil
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
		return nil, nil
	}

	effectivePct := math.Min(100.0, deathPenaltyConfig.ItemsTypePct*mult)

	pickCount := int(math.Ceil(float64(len(items)) * effectivePct / 100.0))
	if pickCount > len(items) {
		pickCount = len(items)
	}

	rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })

	var dropped []DroppedItem
	var lost []string
	itemsByCat := make(map[string]map[string]int64)

	for _, r := range items[:pickCount] {
		// Calculated percentage of stack quantity
		qty := int(math.Ceil(float64(r.quantity) * effectivePct / 100.0))
		if qty > r.quantity {
			qty = r.quantity
		}
		if qty <= 0 {
			qty = 1
		}

		if itemsByCat[r.category] == nil {
			itemsByCat[r.category] = make(map[string]int64)
		}
		itemsByCat[r.category][r.itemID] = int64(qty)

		dropped = append(dropped, DroppedItem{
			ItemID:   r.itemID,
			Category: r.category,
			Quantity: qty,
		})
		name := r.itemID
		if spec, ok := models.ItemRegistry[r.itemID]; ok && spec.Name != "" {
			name = spec.Name
		}
		lost = append(lost, fmt.Sprintf("%dx %s", qty, name))
	}

	// Batch remove inventory items by category
	for cat, itemMap := range itemsByCat {
		_ = InventorySvc.BulkRemoveItems(tx, loserID.String(), cat, itemMap)
	}

	return dropped, lost
}

// dropRecipes samples a portion of the loser's unlocked recipes by percentage and removes them in batch. Returns dropped recipe IDs and human-readable lost lines.
func dropRecipes(tx *sql.Tx, loserID uuid.UUID, mult float64, satKhiLevel int) ([]string, []string) {
	rows, err := tx.Query(`
		SELECT recipe_id FROM player_recipe_progress
		WHERE player_id = $1 AND is_unlocked = true
	`, loserID)
	if err != nil {
		return nil, nil
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
		return nil, nil
	}

	effectivePct := math.Min(100.0, deathPenaltyConfig.RecipesPct*mult)

	pickCount := int(math.Ceil(float64(len(recipeIDs)) * effectivePct / 100.0))
	if pickCount > len(recipeIDs) {
		pickCount = len(recipeIDs)
	}

	rand.Shuffle(len(recipeIDs), func(i, j int) { recipeIDs[i], recipeIDs[j] = recipeIDs[j], recipeIDs[i] })

	droppedIDs := recipeIDs[:pickCount]
	var lost []string
	for _, id := range droppedIDs {
		name := id
		if guDef, ok := models.GetGuAny(id); ok && guDef.Name != "" {
			name = guDef.Name
		}
		lost = append(lost, fmt.Sprintf("Bí phương %s", name))
	}

	// Batch delete recipes in a single query
	if len(droppedIDs) > 0 {
		_ = RecipeSvc.DeleteBatch(tx, loserID.String(), droppedIDs)
	}

	return droppedIDs, lost
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
