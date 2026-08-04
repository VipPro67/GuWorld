package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type ItemEffectHandler func(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error)

var ValidateItemEffects = map[string]ItemEffectHandler{
	"mortal_only":   mortalOnlyEffect,
	"immortal_only": immortalOnlyEffect,
	"min_rank":      minRankEffect,
	"max_rank":      maxRankEffect,
}

var ApplyItemEffects = map[string]ItemEffectHandler{
	"restore_hp":             restoreHPEffect,
	"restore_mana":           restoreManaEffect,
	"soul_boost":             soulBoostEffect,
	"remove_negative_status": removeNegativeStatusEffect,
	"lifespan_boost":         lifespanBoostEffect,
	"open_mine_chest":        openMineChestEffect,
	"heal_aperture":          healApertureEffect,
	"regen_hp_out_of_combat": regenHPOutOfCombatEffect,
	"increase_human_qi":      increaseHumanQi,
}

func healApertureEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["heal_aperture"]
	if val <= 0 {
		return "", nil
	}
	if !isImmortalCultivator(player) {
		return "", fmt.Errorf("%s chỉ dành cho Cổ Tiên để trị liệu tiên khiếu.", item.Name)
	}
	player.ApertureDamage -= val
	if player.ApertureDamage < 0 {
		player.ApertureDamage = 0
	}

	// Cập nhật HP tiên khiếu trong bảng immortal_apertures
	var apID string
	var apertureHP, maxApertureHP int
	var state string
	err := tx.QueryRow(`
		SELECT id, aperture_hp, max_aperture_hp, state 
		FROM immortal_apertures 
		WHERE player_id = $1
	`, player.ID).Scan(&apID, &apertureHP, &maxApertureHP, &state)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("không tìm thấy Tiên Khiếu của đạo hữu.")
	} else if err != nil {
		return "", fmt.Errorf("lỗi khi truy vấn Tiên Khiếu: %w", err)
	}

	newHP := apertureHP + int(val)
	if newHP > maxApertureHP {
		newHP = maxApertureHP
	}
	newState := state
	if state == "DAMAGED" && newHP >= maxApertureHP {
		newState = "STABLE"
	}

	_, err = tx.Exec(`
		UPDATE immortal_apertures 
		SET aperture_hp = $1, state = $2, updated_at = NOW() 
		WHERE id = $3
	`, newHP, newState, apID)
	if err != nil {
		return "", fmt.Errorf("lỗi khi cập nhật sinh lực Tiên Khiếu: %w", err)
	}

	return fmt.Sprintf("Đạo hữu sử dụng %s, khôi phục %.0f sinh lực Tiên Khiếu!", item.Name, val), nil
}

func regenHPOutOfCombatEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["regen_hp_out_of_combat"]
	if val <= 0 {
		return "", nil
	}

	// We'll just return success here and the main `UseItem` will handle Redis TTL or we can handle it here
	// But `cache` package might cause circular dependency if imported? No, `handlers` imports `cache` in `marketplace.go`.
	return fmt.Sprintf("Đạo hữu sử dụng %s, kích hoạt trạng thái Hồi Phục Sinh Lực trong 10 phút!", item.Name), nil
}

func increaseHumanQi(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["increase_human_qi"]
	if val <= 0 {
		return "", nil
	}

	player.HumanQi += val

	return fmt.Sprintf("Đạo hữu sử dụng %s, tăng 50.000 nhân khí!", item.Name), nil
}

func lifespanBoostEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["lifespan_boost"]
	player.Lifespan += val
	return fmt.Sprintf("Đạo hữu sử dụng %s, gia tăng %.0f năm thọ nguyên tối đa và hồi phục %.0f năm tuổi thọ!", item.Name, val, val), nil
}

func restoreManaEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	value := item.Effects["restore_mana"]
	player.CurrentMana += value
	if player.CurrentMana > player.MaxMana {
		player.CurrentMana = player.MaxMana
	}
	return fmt.Sprintf("Đạo hữu sử dụng %s, phục hồi %.0f Chân Nguyên!", item.Name, value), nil
}

func restoreHPEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	value := item.Effects["restore_hp"]
	player.HP += value
	if player.HP > player.BaseHP {
		player.HP = player.BaseHP
	}
	return fmt.Sprintf("Đạo hữu sử dụng %s, hồi phục %.0f HP!", item.Name, value), nil
}

func mortalOnlyEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	if isImmortalCultivator(player) {
		return "", fmt.Errorf("%s chỉ bổ sung Chân Nguyên phàm cấp; Cổ Tiên cần Tiên Nguyên Thạch hoặc Tiên Khiếu ngưng tụ Tiên Nguyên.", item.Name)
	}
	return "", nil
}

func immortalOnlyEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	if !isImmortalCultivator(player) {
		return "", fmt.Errorf("%s chỉ dành cho Cổ Tiên.", item.Name)
	}
	return "", nil
}

func minRankEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["min_rank"]
	if float64(player.Rank) < val {
		return "", fmt.Errorf("Yêu cầu tu vi tối thiểu %s để sử dụng %s.", getRankName(int(val)), item.Name)
	}
	return "", nil
}

func maxRankEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["max_rank"]
	if float64(player.Rank) > val {
		return "", fmt.Errorf("Tu vi vượt quá giới hạn tối đa %s để sử dụng %s.", getRankName(int(val)), item.Name)
	}
	return "", nil
}

func soulBoostEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["soul_boost"]
	if player.DaoMarks == nil {
		player.DaoMarks = make(models.DaoMarksMap)
	}
	player.DaoMarks["soul"] += int(val)
	return fmt.Sprintf("Hồn phách của đạo hữu được tráng kiện (+%.0f điểm Hồn lực).", val), nil
}

func removeNegativeStatusEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	if player.ActiveCombat != nil && len(player.ActiveCombat.PlayerStatusEffects) > 0 {
		player.ActiveCombat.PlayerStatusEffects = []models.StatusEffect{}
		return fmt.Sprintf("%s giải trừ toàn bộ trạng thái tiêu cực của đạo hữu trong trận chiến.", item.Name), nil
	}
	return "", nil
}

func formatSummaryOfRewards(reward map[string]interface{}) string {
	if reward == nil {
		return "Không có phần thưởng"
	}
	rewardType, _ := reward["type"].(string)
	switch rewardType {
	case "stones":
		qty, _ := reward["quantity"].(int)
		return fmt.Sprintf("Nhận được %d Nguyên Thạch", qty)
	case "gu":
		name, _ := reward["name"].(string)
		return fmt.Sprintf("Nhận được Cổ trùng [%s]", name)
	case "recipe":
		name, _ := reward["name"].(string)
		rate, _ := reward["completion_rate"].(float64)
		return fmt.Sprintf("Nhận được Bí phương [%s] (%.0f%%)", name, rate)
	case "material":
		name, _ := reward["name"].(string)
		qty, _ := reward["quantity"].(int)
		return fmt.Sprintf("Nhận được %dx %s", qty, name)
	default:
		return fmt.Sprintf("Nhận được %v", reward)
	}
}

func openMineChestEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	msg := fmt.Sprintf("Mở %s thành công!\n", item.Name)
	var reward map[string]interface{}

	// ── 50 % → Nguyên Thạch  |  50 % → Vật phẩm (vật liệu / cổ trùng / bí phương)
	if rand.Float64() < 0.50 {
		// ── STONES: 100-500
		stones := 100 + rand.Intn(401)
		player.PrimevalStones += stones
		msg += fmt.Sprintf("- 💎 Nhận được %d Nguyên Thạch\n", stones)
		reward = map[string]interface{}{
			"type":     "stones",
			"quantity": stones,
		}
	} else {
		// ── ITEM POOL ──
		// Roll type: Material 92.5%, Gu 5%, Recipe 2.5%
		roll := rand.Float64() * 100.0

		if roll < 5.0 {
			// ── GU DROP (5%) ──
			var guMsg string
			guMsg, reward = rollGuDrop(tx, player, item)
			if guMsg != "" {
				msg += guMsg
			} else {
				// Fallback to materials if no valid Gu found
				var matMsg string
				matMsg, reward = rollMaterialDrop(tx, player, item)
				msg += matMsg
			}
		} else if roll < 7.5 {
			// ── RECIPE DROP (2.5%) ──
			var recipeMsg string
			recipeMsg, reward = rollRecipeDrop(tx, player, item)
			if recipeMsg != "" {
				msg += recipeMsg
			} else {
				var matMsg string
				matMsg, reward = rollMaterialDrop(tx, player, item)
				msg += matMsg
			}
		} else {
			// ── MATERIAL DROP (92.5%) ──
			var matMsg string
			matMsg, reward = rollMaterialDrop(tx, player, item)
			msg += matMsg
		}
	}

	// Write Audit Log
	auditlog.Log(tx, auditlog.Entry{
		PlayerID:       &player.ID,
		Action:         "item.open_mine_chest",
		EntityType:     "item",
		EntityID:       item.ID,
		RequestSummary: fmt.Sprintf("Mở %s | %s", item.Name, formatSummaryOfRewards(reward)),
		Metadata: map[string]interface{}{
			"chest_id":   item.ID,
			"chest_name": item.Name,
			"reward":     reward,
		},
		Success: true,
	})

	return msg, nil
}

// guRarityWeight maps Gu rarity string → weight (lower = rarer = less likely)
func guRarityWeight(rarity string) float64 {
	switch rarity {
	case "COMMON":
		return 100.0
	case "UNCOMMON":
		return 60.0
	case "RARE":
		return 30.0
	case "EPIC":
		return 12.0
	case "LEGENDARY":
		return 4.0
	case "MYTHIC":
		return 1.0
	case "IMMORTAL":
		return 0.5
	case "IMMORTAL_6":
		return 0.2
	case "IMMORTAL_7":
		return 0.08
	case "IMMORTAL_8":
		return 0.03
	case "IMMORTAL_9":
		return 0.01
	default:
		return 50.0
	}
}

// rankDropMultiplier returns a weight multiplier for Gu/material of a given rank.
// Higher chest rank → lower multiplier → rarer drops. Rank ≥ 6 is extremely low.
func rankDropMultiplier(rank int) float64 {
	switch {
	case rank <= 1:
		return 1.0
	case rank == 2:
		return 0.8
	case rank == 3:
		return 0.5
	case rank == 4:
		return 0.25
	case rank == 5:
		return 0.10
	case rank == 6:
		return 0.005
	case rank == 7:
		return 0.001
	case rank == 8:
		return 0.0002
	default: // 9+
		return 0.00005
	}
}

// itemRarityWeight maps item rarity int → weight
func itemRarityWeight(rarity int) float64 {
	switch {
	case rarity <= 1:
		return 100.0
	case rarity == 2:
		return 50.0
	case rarity == 3:
		return 20.0
	case rarity == 4:
		return 5.0
	default:
		return 1.0
	}
}

func rollMaterialDrop(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, map[string]interface{}) {
	// Build weighted pool of materials
	type weightedMat struct {
		id     string
		weight float64
	}
	var pool []weightedMat
	var totalWeight float64

	for id, spec := range models.ItemRegistry {
		if spec.Type != "Material" {
			continue
		}
		w := rankDropMultiplier(spec.Rank)
		if w > 0 {
			pool = append(pool, weightedMat{id: id, weight: w})
			totalWeight += w
		}
	}

	if len(pool) == 0 || totalWeight <= 0 {
		return "- Hòm trống rỗng, không có gì đặc biệt.\n", map[string]interface{}{"type": "none"}
	}

	// Weighted random pick
	r := rand.Float64() * totalWeight
	var chosen string
	for _, m := range pool {
		r -= m.weight
		if r <= 0 {
			chosen = m.id
			break
		}
	}
	if chosen == "" {
		chosen = pool[0].id
	}

	InventorySvc.AddItem(tx, player.ID.String(), chosen, "materials", 1)
	matName := chosen
	if spec, ok := models.ItemRegistry[chosen]; ok {
		matName = spec.Name
	}
	return fmt.Sprintf("- Nhận được 1x %s\n", matName), map[string]interface{}{
		"type":     "material",
		"id":       chosen,
		"name":     matName,
		"quantity": 1,
	}
}

func rollGuDrop(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, map[string]interface{}) {
	excludedImmortal, err := db.GetExistingImmortalGuIDs(tx)
	if err != nil {
		logger.S.Warnf("rollGuDrop: failed to load existing immortal Gu: %v", err)
		excludedImmortal = map[string]bool{}
	}

	type weightedGu struct {
		id     string
		weight float64
	}
	var pool []weightedGu
	var totalWeight float64

	for id, ranks := range models.GuRegistry {
		if excludedImmortal[id] {
			continue
		}
		for _, spec := range ranks {
			w := rankDropMultiplier(spec.Rank)
			if w > 0 {
				pool = append(pool, weightedGu{id: id, weight: w})
				totalWeight += w
			}
		}
	}

	if len(pool) == 0 || totalWeight <= 0 {
		return "", nil
	}

	// Weighted random pick
	r := rand.Float64() * totalWeight
	var chosenID string
	for _, g := range pool {
		r -= g.weight
		if r <= 0 {
			chosenID = g.id
			break
		}
	}
	if chosenID == "" {
		chosenID = pool[0].id
	}

	// Hunger: random 0-10%
	hunger := rand.Intn(11) // 0 to 10
	spec, _ := models.GetGuAny(chosenID)

	newInstance := models.GuInstance{
		InstanceID: "gu-" + uuid.New().String(),
		GuID:       chosenID,
		Rank:       spec.Rank,
		Hunger:     hunger,
		Status:     "ACTIVE",
		Equipped:   false,
	}
	player.GuInventory = append(player.GuInventory, newInstance)

	// Save player Gu inventory changes immediately
	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		logger.S.Errorf("rollGuDrop: failed to save player Gu inventory: %v", err)
		return "", nil
	}

	guName := chosenID
	isImmortal := false
	if spec.ID != "" {
		guName = spec.Name
		isImmortal = spec.Rank >= 6 // Assuming rank 6+ is immortal
	}

	rewardMeta := map[string]interface{}{
		"type":        "gu",
		"id":          chosenID,
		"name":        guName,
		"instance_id": newInstance.InstanceID,
		"rank":        spec.Rank,
		"hunger":      hunger,
	}

	if isImmortal {
		return fmt.Sprintf("- Thiên đại cơ duyên! Đạo hữu nhận được Tiên Cổ: [%s]! (Độ no: %d%%)\n", guName, hunger), rewardMeta
	}
	return fmt.Sprintf("- Cơ duyên xảo hợp! Đạo hữu nhận được Cổ trùng: [%s] (Độ no: %d%%)\n", guName, hunger), rewardMeta
}

func rollRecipeDrop(tx *sql.Tx, player *models.Player, _ models.ItemSpec) (string, map[string]interface{}) {
	rows, err := db.DB.Query(`SELECT target_gu_id, target_rank, variant FROM refining_recipes`)
	if err != nil {
		return "", nil
	}
	defer rows.Close()

	type recipeInfo struct {
		id         string
		targetGuID string
	}
	var allRecipes []recipeInfo
	for rows.Next() {
		var targetGuID string
		var targetRank, variant int
		if err := rows.Scan(&targetGuID, &targetRank, &variant); err == nil {
			allRecipes = append(allRecipes, recipeInfo{
				id:         fmt.Sprintf("recipe_%s_%d_%d", targetGuID, targetRank, variant),
				targetGuID: targetGuID,
			})
		}
	}
	if len(allRecipes) == 0 {
		return "", nil
	}

	// Build map of already-unlocked recipes
	recipes, _ := RecipeSvc.ListByPlayer(player.ID.String())
	unlockedMap := make(map[string]float64)
	for _, r := range recipes {
		unlockedMap[r.RecipeID] = r.CompletionRate
	}

	// Filter to candidates: not yet unlocked, or unlocked but < 100%
	var candidates []recipeInfo
	for _, ri := range allRecipes {
		if rate, ok := unlockedMap[ri.id]; !ok || rate < 100.0 {
			candidates = append(candidates, ri)
		}
	}
	if len(candidates) == 0 {
		return "", nil // All recipes at 100%
	}

	chosen := candidates[rand.Intn(len(candidates))]
	completionRate := float64(rand.Intn(81)) // 0-80%

	// Find target Gu name for display
	targetGuName := chosen.targetGuID
	if spec, ok := models.GetGuAny(chosen.targetGuID); ok {
		targetGuName = spec.Name
	}

	// Add or update recipe
	if err := RecipeSvc.Upsert(tx, &services.RecipeProgress{
		PlayerID:       player.ID.String(),
		RecipeID:       chosen.id,
		CompletionRate: completionRate,
		IsUnlocked:     true,
	}); err != nil {
		logger.S.Errorf("rollRecipeDrop: failed to upsert recipe progress: %v", err)
		return "", nil
	}

	return fmt.Sprintf("- Lĩnh ngộ bí phương: [%s] (Độ hoàn thiện: %.0f%%)\n", targetGuName, completionRate), map[string]interface{}{
		"type":            "recipe",
		"id":              chosen.id,
		"target_gu_id":    chosen.targetGuID,
		"name":            targetGuName,
		"completion_rate": completionRate,
	}
}
