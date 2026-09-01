package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/internal/util"
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
	"increase_human_qi":      increaseHumanQi,
	"random_constitution":    randomConstitutionEffect,
	"random_race":            randomRaceEffect,
}

func randomConstitutionEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	defs := gamefx.AllConstitutions()
	if len(defs) == 0 {
		return "", fmt.Errorf("không có thể chất nào trong registry.")
	}
	if len(defs) == 1 {
		def := defs[0]
		player.Constitution = def.Key
		return fmt.Sprintf("Đạo hữu sử dụng %s, thức tỉnh thể chất [%s]!", item.Name, gamefx.ConstitutionDisplayName(def.Key)), nil
	}
	chosen := defs[rand.Intn(len(defs))]
	for chosen.Key == player.Constitution {
		chosen = defs[rand.Intn(len(defs))]
	}
	player.Constitution = chosen.Key
	return fmt.Sprintf("Đạo hữu sử dụng %s, thức tỉnh thể chất [%s]!", item.Name, gamefx.ConstitutionDisplayName(chosen.Key)), nil
}

func randomRaceEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	defs := gamefx.AllRaces()
	if len(defs) == 0 {
		return "", fmt.Errorf("không có chủng tộc nào trong registry.")
	}
	if len(defs) == 1 {
		def := defs[0]
		player.Race = def.Key
		return fmt.Sprintf("Đạo hữu sử dụng %s, chuyển hóa thành chủng tộc [%s]!", item.Name, gamefx.RaceDisplayName(def.Key)), nil
	}
	chosen := defs[rand.Intn(len(defs))]
	for chosen.Key == player.Race {
		chosen = defs[rand.Intn(len(defs))]
	}
	player.Race = chosen.Key
	return fmt.Sprintf("Đạo hữu sử dụng %s, chuyển hóa thành chủng tộc [%s]!", item.Name, gamefx.RaceDisplayName(chosen.Key)), nil
}

func healApertureEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["heal_aperture"]
	if val <= 0 {
		return "", nil
	}
	if !util.IsImmortalPlayer(player) {
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
	if player.HP > gamefx.EffectiveMaxHP(player) {
		player.HP = gamefx.EffectiveMaxHP(player)
	}
	return fmt.Sprintf("Đạo hữu sử dụng %s, hồi phục %.0f HP!", item.Name, value), nil
}

func mortalOnlyEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	if util.IsImmortalPlayer(player) {
		return "", fmt.Errorf("%s chỉ bổ sung Chân Nguyên phàm cấp; Cổ Tiên cần Tiên Nguyên Thạch hoặc Tiên Khiếu ngưng tụ Tiên Nguyên.", item.Name)
	}
	return "", nil
}

func immortalOnlyEffect(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, error) {
	if !util.IsImmortalPlayer(player) {
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
	// reward is implicitly used in messages

	// ── 20 % → Nguyên Thạch  |  80 % → Vật phẩm (vật liệu / bí phương)
	if rand.Float64() < 0.20 {
		// ── STONES: 100-500
		stones := int64(100 + rand.Intn(401))
		player.PrimevalStones += stones
		msg += fmt.Sprintf("- 💎 Nhận được %d Nguyên Thạch\n", stones)
		_ = map[string]interface{}{
			"type":     "stones",
			"quantity": stones,
		}
	} else {
		// ── ITEM POOL ──
		// Roll type: Material 80%, Recipe 20%
		roll := rand.Float64() * 100.0

		if roll < 40.0 {
			// ── RECIPE DROP (40%) ──
			var recipeMsg string
			recipeMsg, _ = rollRecipeDrop(tx, player, item)
			if recipeMsg != "" {
				msg += recipeMsg
			} else {
				var matMsg string
				matMsg, _ = rollMaterialDrop(tx, player, item)
				msg += matMsg
			}
		} else {
			// ── MATERIAL DROP (60%) ──
			var matMsg string
			matMsg, _ = rollMaterialDrop(tx, player, item)
			msg += matMsg
		}
	}


	return msg, nil
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

// effectiveDropLevel returns the drop-weighting level for a Gu/item, taking the
// higher of its rank and its rarity tier (1-9). A rank < 6 mortal Gu tagged
// with a high rarity is thus as rare as a high-rank Gu.
func effectiveDropLevel(rank, rarity int) int {
	if rarity > rank {
		return rarity
	}
	return rank
}

func rollMaterialDrop(tx *sql.Tx, player *models.Player, item models.ItemSpec) (string, map[string]interface{}) {
	pc := getPrecomputedPools()
	chosen := pc.rollMaterial()
	if chosen == "" {
		return "- Hòm trống rỗng, không có gì đặc biệt.\n", map[string]interface{}{"type": "none"}
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

func rollRecipeDrop(tx *sql.Tx, player *models.Player, _ models.ItemSpec) (string, map[string]interface{}) {
	pc := getPrecomputedPools()
	allRecipes := pc.recipesByRank["all"]
	if len(allRecipes) == 0 {
		return "", nil
	}

	// Build map of already-unlocked recipes
	recipes, _ := RecipeSvc.ListByPlayer(player.ID.String())
	unlockedMap := make(map[string]float64)
	deduceMap := make(map[string]int)
	for _, r := range recipes {
		unlockedMap[r.RecipeID] = r.CompletionRate
		deduceMap[r.RecipeID] = r.DeduceCount
	}

	// Filter to candidates: not yet unlocked, or unlocked but < 100%
	var candidates []weightedRecipeEntry
	for _, ri := range allRecipes {
		if rate, ok := unlockedMap[ri.id]; !ok || rate < 100.0 {
			candidates = append(candidates, ri)
		}
	}
	if len(candidates) == 0 {
		return "", nil // All recipes at 100%
	}

	var totalWeight float64
	for _, c := range candidates {
		totalWeight += c.weight
	}
	if totalWeight <= 0 {
		return "", nil
	}

	r := rand.Float64() * totalWeight
	chosen := candidates[len(candidates)-1]
	for _, c := range candidates {
		r -= c.weight
		if r <= 0 {
			chosen = c
			break
		}
	}
	completionRate := float64(rand.Intn(81)) // 0-80%

	// Find target Gu name for display
	targetGuName := chosen.targetGuID
	if spec, ok := models.GetGuAny(chosen.targetGuID); ok {
		targetGuName = spec.Name
	}

	// Duplicate handling: keep the higher completion rate
	existingRate, isDup := unlockedMap[chosen.id]
	effectiveRate := completionRate
	deduceCount := deduceMap[chosen.id]
	msg := fmt.Sprintf("- Lĩnh ngộ bí phương: [%s] (Độ hoàn thiện: %.0f%%)\n", targetGuName, completionRate)

	if isDup {
		if completionRate > existingRate {
			effectiveRate = completionRate
			msg = fmt.Sprintf("- Lĩnh ngộ bí phương: [%s] (Độ hoàn thiện: %.0f%% → %.0f%%)\n", targetGuName, existingRate, completionRate)
		} else {
			effectiveRate = existingRate
			msg = fmt.Sprintf("- Mở trùng bí phương: [%s] (Giữ mức hoàn thiện cao hơn: %.0f%%)\n", targetGuName, existingRate)
		}
	}

	// Add or update recipe
	if err := RecipeSvc.Upsert(tx, &services.RecipeProgress{
		PlayerID:       player.ID.String(),
		RecipeID:       chosen.id,
		CompletionRate: effectiveRate,
		IsUnlocked:     true,
		DeduceCount:    deduceCount,
	}); err != nil {
		logger.S.Errorf("rollRecipeDrop: failed to upsert recipe progress: %v", err)
		return "", nil
	}

	return msg, map[string]interface{}{
		"type":            "recipe",
		"id":              chosen.id,
		"target_gu_id":    chosen.targetGuID,
		"name":            targetGuName,
		"completion_rate": effectiveRate,
		"duplicate":       isDup,
	}
}
