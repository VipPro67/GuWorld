package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

func RefineGu(c *gin.Context) {
	var req RefineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipe_id is required"})
		return
	}

	// 1. Get auth player ID
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// 2. Fetch and lock player details
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player status"})
		}
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if player.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể luyện Cổ!"})
		return
	}

	// 2b. Verify player has unlocked the recipe
	var unlockedRate float64 = 0.0
	recipeProgress, err := RecipeSvc.Get(tx, player.ID.String(), req.RecipeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở công thức này!"})
		return
	}
	unlockedRate = recipeProgress.CompletionRate
	isForcedRefine := unlockedRate < 100.0

	// 3. Fetch Recipe
	targetGuID, targetRank, variant, err := ParseRecipeID(req.RecipeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID công thức không hợp lệ"})
		return
	}

	var recipe models.RefiningRecipe
	err = tx.QueryRow(`
		SELECT target_gu_id, target_rank, variant, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost 
		FROM refining_recipes 
		WHERE target_gu_id = $1 AND target_rank = $2 AND variant = $3`, targetGuID, targetRank, variant).Scan(
		&recipe.TargetGuID, &recipe.TargetRank, &recipe.Variant, &recipe.RequiredGus,
		&recipe.RequiredMaterials, &recipe.SuccessRate, &recipe.PrimevalStonesCost,
		&recipe.ImmortalEssenceStonesCost,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Refining recipe not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recipe"})
		}
		return
	}
	recipe.ID = req.RecipeID

	// 3b. Calculate Mana Cost
	manaCost, err := calculateRefineManaCost(player.Rank, recipe.TargetRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Validate Costs (Primeval Stones & Mana)
	if player.CurrentMana < manaCost {
		unit := "Chân Nguyên"
		if player.Rank >= 6 {
			unit = "Tiên Nguyên"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ %s! Cần %.1f", unit, manaCost)})
		return
	}

	if player.PrimevalStones < recipe.PrimevalStonesCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough Primeval Stones"})
		return
	}

	// 4b. Validate Costs (Immortal Essence Stones)
	if recipe.ImmortalEssenceStonesCost > 0 && player.ImmortalStones < recipe.ImmortalEssenceStonesCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough Immortal Essence Stones"})
		return
	}

	// 5. Validate Costs (Required Materials)
	for matID, reqQty := range recipe.RequiredMaterials {
		currentQty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), matID)
		if currentQty < reqQty {
			matName := matID
			if itemSpec, ok := models.ItemRegistry[matID]; ok && itemSpec.Name != "" {
				matName = itemSpec.Name
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matName)})
			return
		}
	}

	// 6. Validate Selected Gu Instance IDs
	totalReqGusCount := 0
	for _, req := range recipe.RequiredGus {
		totalReqGusCount += req.Qty
	}
	if len(req.SelectedGuInstanceIDs) != totalReqGusCount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect number of Gu bugs selected for refining"})
		return
	}

	// Count Gu types (matching the required rank or higher) and verify ownership
	selectedGusCounts := make(map[string]int)
	seenInstances := make(map[string]bool)
	materialQualityBonus := 0.0

	for _, instID := range req.SelectedGuInstanceIDs {
		instID = strings.TrimSpace(instID)
		if seenInstances[instID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate Gu instance selected"})
			return
		}
		seenInstances[instID] = true

		item, _, found := models.FindGuByInstanceID(player.GuInventory, instID)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in your inventory: " + instID})
			return
		}
		if item.Status == "DEAD" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot use a DEAD Gu bug for refining"})
			return
		}
		// Count instances that satisfy the recipe requirement (equal or higher rank).
		if reqGu, ok := recipe.RequiredGus[item.GuID]; ok && (reqGu.Rank == 0 || item.Rank >= reqGu.Rank) {
			selectedGusCounts[item.GuID]++
			if reqGu.Rank > 0 && item.Rank > reqGu.Rank {
				materialQualityBonus += 0.02 * float64(item.Rank-reqGu.Rank)
			}
		}
	}

	// Check if selection matches recipe required Gus.
	guRequirementsNotMet := false
	for guID, reqGu := range recipe.RequiredGus {
		if selectedGusCounts[guID] != reqGu.Qty {
			guRequirementsNotMet = true
			break
		}
	}

	// 7. Consume resources (Stones, Mana, Materials & Gus)
	player.CurrentMana -= manaCost
	player.PrimevalStones -= recipe.PrimevalStonesCost
	if recipe.ImmortalEssenceStonesCost > 0 {
		player.ImmortalStones -= recipe.ImmortalEssenceStonesCost
	}

	// 8. RNG Roll & 10 Factors Calculation
	var pathRaw []byte
	err = tx.QueryRow(`SELECT path FROM gu_dictionary WHERE id = $1 AND rank = $2`, recipe.TargetGuID, recipe.TargetRank).Scan(&pathRaw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query target Gu path: " + err.Error()})
		return
	}

	var paths []string
	if err := json.Unmarshal(pathRaw, &paths); err != nil {
		paths = []string{string(pathRaw)}
	}

	// Calculate target path attainment bonus & points
	targetPathMastery := 0.0
	targetPathPoints := 0
	if player.PathMasteries != nil {
		for _, path := range paths {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			if points, ok := player.PathMasteries[cleanPath]; ok {
				if points > targetPathPoints {
					targetPathPoints = points
				}
				bonus := 0.0
				switch gamefx.PathMasteryLevel(points) {
				case 1:
					bonus = 0.02
				case 2:
					bonus = 0.05
				case 3:
					bonus = 0.08
				case 4:
					bonus = 0.10
				}
				if bonus > targetPathMastery {
					targetPathMastery = bonus
				}
			}
		}
	}

	// Refinement Dao Marks & Mastery
	refinementMarks := int64(0)
	refinementPoints := 0
	if player.DaoMarks != nil {
		refinementMarks = player.DaoMarks["refinement"]
	}
	if player.PathMasteries != nil {
		refinementPoints = player.PathMasteries["refinement"]
	}

	rankScale := 1.0
	switch recipe.TargetRank {
	case 1:
		rankScale = 1.0
	case 2:
		rankScale = 2.0
	case 3:
		rankScale = 5.0
	case 4:
		rankScale = 20.0
	case 5:
		rankScale = 100.0
	case 6:
		rankScale = 10000.0
	case 7:
		rankScale = 1000000.0
	case 8:
		rankScale = 100000000.0
	case 9:
		rankScale = 10000000000.0
	}

	masteryMultiplier := 1.0 + float64(targetPathPoints)
	refinementDaoBonus := (0.00632 * math.Sqrt(float64(refinementMarks)) * masteryMultiplier) / rankScale

	refinementRealm := gamefx.PathMasteryLevel(refinementPoints)
	refinementMasteryBonus := 0.0
	switch refinementRealm {
	case 1:
		refinementMasteryBonus = 0.05
	case 2:
		refinementMasteryBonus = 0.10
	case 3:
		refinementMasteryBonus = 0.18
	case 4:
		refinementMasteryBonus = 0.25
	}

	// Luck Bonus
	luckMarks := int64(0)
	if player.DaoMarks != nil {
		luckMarks = player.DaoMarks["luck"]
	}
	fx := gamefx.Get(&player)
	luckBonus := math.Min(0.10, (float64(luckMarks)*0.001)+fx.LuckBonus/100.0)

	// Environment Bonus
	var regionPathsRaw []byte
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(paths, '[]') FROM regions WHERE id = $1`, player.CurrentRegion).Scan(&regionPathsRaw); err != nil {
		logger.S.Warnf("RefineGu: failed to query region paths for region %s: %v", player.CurrentRegion, err)
	}
	var regionPaths []string
	if len(regionPathsRaw) > 0 {
		_ = json.Unmarshal(regionPathsRaw, &regionPaths)
	}
	envBonus := 0.0
	for _, rp := range regionPaths {
		cleanRP := strings.TrimSuffix(strings.ToLower(rp), " path")
		for _, tp := range paths {
			cleanTP := strings.TrimSuffix(strings.ToLower(tp), " path")
			if cleanRP == cleanTP {
				envBonus = 0.05
				break
			}
		}
		if envBonus > 0 {
			break
		}
	}

	// Support Bonus & Status/Race Refine Effects
	supportGuBonus := collectRefineSupportBonus(&player)
	posRefineBonus := 0.0
	negRefinePenalty := 0.0
	if fx.RefineBonus > 0 {
		posRefineBonus = fx.RefineBonus
	} else if fx.RefineBonus < 0 {
		negRefinePenalty = -fx.RefineBonus
	}
	supportBonus := supportGuBonus + posRefineBonus

	// Apply Immortal Penalty to flat bonuses for Rank 6+
	immortalPenalty := 1.0
	switch recipe.TargetRank {
	case 6:
		immortalPenalty = 2.0
	case 7:
		immortalPenalty = 5.0
	case 8:
		immortalPenalty = 20.0
	case 9:
		immortalPenalty = 100.0
	}

	if immortalPenalty > 1.0 {
		refinementMasteryBonus /= immortalPenalty
		targetPathMastery /= immortalPenalty
		luckBonus /= immortalPenalty
		supportBonus /= immortalPenalty
		envBonus /= immortalPenalty
		materialQualityBonus /= immortalPenalty
	}

	// Final Rate Calculation
	baseRate := recipe.SuccessRate
	sumPositiveBonus := refinementDaoBonus + refinementMasteryBonus + targetPathMastery + luckBonus + supportBonus + envBonus + materialQualityBonus

	rawPositiveRate := baseRate + sumPositiveBonus
	completionPenalty := 1.0
	if isForcedRefine {
		completionPenalty = unlockedRate / 100.0
		rawPositiveRate = rawPositiveRate * completionPenalty
	}

	// Cap positive rate at 1.0 (100%) BEFORE applying negative debuff penalties
	cappedPositiveRate := rawPositiveRate
	if cappedPositiveRate > 1.0 {
		cappedPositiveRate = 1.0
	}

	// Subtract debuff penalties (e.g., refine_backfire, refine_fatigue) after capping
	finalSuccessRate := cappedPositiveRate - negRefinePenalty
	if finalSuccessRate < 0.0 {
		finalSuccessRate = 0.0
	}
	if guRequirementsNotMet {
		finalSuccessRate = 0.0
	}

	roll := rand.Float64()
	outcome := determineRefineOutcome(roll, finalSuccessRate, refinementRealm, recipe.TargetRank)

	breakdown := RefineRateBreakdown{
		BaseRate:           baseRate,
		CompletionPenalty:  completionPenalty,
		ComboAccuracy:      !guRequirementsNotMet,
		MaterialQuality:    materialQualityBonus,
		RefinementDaoMarks: refinementDaoBonus,
		RefinementMastery:  refinementMasteryBonus,
		TargetPathMastery:  targetPathMastery,
		LuckBonus:          luckBonus,
		SupportBonus:       supportBonus,
		StatusPenalty:      negRefinePenalty,
		EnvBonus:           envBonus,
		FinalRate:          finalSuccessRate,
		RefinementRealm:    refinementRealm,
	}

	var newGuInstance models.GuInstance
	var statusMsg string
	var statusStr string
	var isUniqueImmortalFail bool

	// Tiên cổ duy nhất check
	if recipe.TargetRank >= 6 {
		exists, _ := db.CheckImmortalGuExists(tx, recipe.TargetGuID, recipe.TargetRank, req.SelectedGuInstanceIDs)
		if exists {
			isUniqueImmortalFail = true
			// Tiên cổ duy nhất đã tồn tại: không thể luyện thành công,
			// và cũng không bị phản phệ (backfire) vì thất bại này có nguyên nhân khách quan.
			if outcome != OutcomeFailNormal {
				outcome = OutcomeFailNormal
			}
		}
	}

	// Consume Materials based on outcome
	if outcome == OutcomeFailSafe {
		for matID, reqQty := range recipe.RequiredMaterials {
			deductQty := int64(math.Ceil(float64(reqQty) * 0.5))
			if deductQty > 0 {
				if err := InventorySvc.DeductItem(tx, player.ID.String(), matID, deductQty); err != nil {
					matName := matID
					if itemSpec, ok := models.ItemRegistry[matID]; ok && itemSpec.Name != "" {
						matName = itemSpec.Name
					}
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matName)})
					return
				}
			}
		}
	} else {
		for matID, reqQty := range recipe.RequiredMaterials {
			if err := InventorySvc.DeductItem(tx, player.ID.String(), matID, reqQty); err != nil {
				matName := matID
				if itemSpec, ok := models.ItemRegistry[matID]; ok && itemSpec.Name != "" {
					matName = itemSpec.Name
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matName)})
				return
			}
		}
	}

	hasVitalIngredient := false
	hasVitalSurvived := false
	var refineEventMsg string
	consumedGuIDs := []string{} // Track consumed Gus for explicit delete.

	if outcome == OutcomeSuccess {
		statusStr = "success"
		statusMsg = "Hợp luyện thành công! Đạo hữu luyện chế được Cổ trùng mới."
		newInventory := models.GuInventory{}
		for _, item := range player.GuInventory {
			cleanInstID := strings.TrimSpace(item.InstanceID)
			if seenInstances[item.InstanceID] || seenInstances[cleanInstID] {
				consumedGuIDs = append(consumedGuIDs, item.InstanceID)
				if item.IsVital {
					hasVitalIngredient = true
				}
				if spec, ok := models.GetGu(item.GuID, item.Rank); ok {
					if msg, err := HandleGuEvent(&player, spec, item, "ON_REFINE_MATERIAL"); err == nil && msg != "" {
						refineEventMsg = msg
					}
				}
			} else {
				newInventory = append(newInventory, item)
			}
		}
		if refineEventMsg != "" {
			statusMsg += " " + refineEventMsg
		}

		newGuInstance = models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       recipe.TargetGuID,
			Rank:       recipe.TargetRank,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
			IsVital:    hasVitalIngredient,
		}
		newInventory = append(newInventory, newGuInstance)
		player.GuInventory = newInventory

		guName := newGuInstance.GuID
		if targetSpec, ok := models.GetGu(newGuInstance.GuID, newGuInstance.Rank); ok && targetSpec.Name != "" {
			guName = targetSpec.Name
		}
		refineSummary := fmt.Sprintf("Cổ Sư [%s] đã hợp luyện thành công Cổ trùng [%s] R%d!", player.Username, guName, newGuInstance.Rank)
		_ = db.LogGuHistory(tx, newGuInstance.InstanceID, newGuInstance.GuID, newGuInstance.Rank, "CREATED_REFINE", &player.ID, nil, refineSummary, map[string]interface{}{
			"recipe_id": req.RecipeID,
			"gu_name":   guName,
		})

		// Add refinement path mastery (Luyện Đạo)
		refineGain := calculateRefinementPathGain(recipe.TargetRank, refinementRealm)
		if refineGain > 0 {
			if player.PathMasteries == nil {
				player.PathMasteries = make(models.PathMasteriesMap)
			}
			player.PathMasteries["refinement"] += refineGain
			newRefineLevel := gamefx.PathMasteryLevel(player.PathMasteries["refinement"])
			statusMsg += fmt.Sprintf(" Ngộ Đạo tăng tiến! +%d điểm Chân Ý lưu phái Luyện Đạo (Cảnh giới: %s).", refineGain, gamefx.PathMasteryLevelName(newRefineLevel))
		}
		if newGuInstance.Rank >= 6 {
			guName := newGuInstance.GuID
			if targetSpec, ok := models.GetGu(newGuInstance.GuID, newGuInstance.Rank); ok {
				guName = targetSpec.Name
			}
			message := fmt.Sprintf("Đạo hữu %s đã luyện chế thành công Cổ Trùng [%s] R%d!", player.Username, guName, newGuInstance.Rank)
			_ = SendSystemMessage("world", sql.NullString{}, message)
		}
	} else if outcome == OutcomeFailSafe {
		statusStr = "failed"
		statusMsg = "Luyện cổ thất bại nhưng Cảnh Giới đủ cao giúp bảo toàn Cổ nguyên liệu! Chỉ mất 50% nguyên liệu phụ."
	} else {
		statusStr = "failed"
		if isUniqueImmortalFail {
			statusMsg = "Tiên cổ là duy nhất, đã tồn tại trong thiên hạ, luyện chế thất bại!"
		} else if guRequirementsNotMet {
			statusMsg = "Cổ nguyên liệu không khớp công thức! Các nguyên liệu đã hóa thành tro bụi."
		} else {
			statusMsg = "Hợp luyện thất bại! Các nguyên liệu thường đã hóa thành tro bụi."
		}

		newInventory := models.GuInventory{}
		for _, item := range player.GuInventory {
			if seenInstances[item.InstanceID] {
				if item.IsVital {
					item.Hunger = 1
					item.Status = "DYING"
					item.Equipped = false
					newInventory = append(newInventory, item)
					hasVitalSurvived = true
				} else {
					consumedGuIDs = append(consumedGuIDs, item.InstanceID)
				}
				if spec, ok := models.GetGu(item.GuID, item.Rank); ok {
					if msg, err := HandleGuEvent(&player, spec, item, "ON_REFINE_MATERIAL"); err == nil && msg != "" {
						refineEventMsg = msg
					}
				}
			} else {
				newInventory = append(newInventory, item)
			}
		}
		player.GuInventory = newInventory
		if refineEventMsg != "" {
			statusMsg += " " + refineEventMsg
		}

		if outcome == OutcomeFailBackfire {
			hpLoss := gamefx.EffectiveMaxHP(&player) * 0.50
			player.HP -= hpLoss
			if player.HP < 10.0 {
				player.HP = 10.0
			}
			soulLoss := int(math.Ceil(float64(player.MaxSoul) * 0.30))
			player.SoulHP -= soulLoss
			if player.SoulHP < 1 {
				player.SoulHP = 1
			}
			statusMsg += fmt.Sprintf(" Luyện cổ CHÁY NỔ (Backfire)! Phản phệ dữ dội, tổn thất %d Hồn Phách và %.0f HP.", soulLoss, hpLoss)
		} else if isForcedRefine {
			player.HP -= 30.0
			if player.HP < 10.0 {
				player.HP = 10.0
			}
			statusMsg += " Ép Luyện thất bại! Phản phệ chấn thương kinh mạch, tổn thất 30 HP."
		}

		if hasVitalSurvived {
			statusMsg += " Riêng Cổ Bản Mệnh vẫn sống sót nhưng rơi vào trạng thái Hấp Hối!"
		}
	}

	statusMsg = fmt.Sprintf("%s (Tỉ lệ: %.1f%%)", statusMsg, finalSuccessRate*100.0)

	// Recalculate equipped_gus list
	player.EquippedGus = []string{}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}

	qiGained := 0.0

	// Apply / escalate refine fatigue and backfire statuses
	// Only trigger fatigue when refining Immortal Gu (Rank >= 6) or Gu with Rank >= player's Rank.
	isTiringRefine := recipe.TargetRank >= 6 || recipe.TargetRank >= player.Rank
	activeStatuses := player.Statuses.Active()
	hasFatigue := activeStatuses.Has("refine_fatigue")
	hasBackfire := activeStatuses.Has("refine_backfire")

	if isTiringRefine || outcome == OutcomeFailBackfire {
		if hasFatigue {
			player.Statuses.Remove("refine_fatigue")
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			statusMsg += " [Luyện Cổ Phản Phệ]: Tỉ lệ Luyện Cổ giảm 50% trong 20 phút do cố luyện khi tâm thần mệt mỏi!"
		} else if hasBackfire {
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			statusMsg += " Trạng thái [Luyện Cổ Phản Phệ] (giảm 50% tỉ lệ) đã được gia hạn thêm 20 phút."
		} else if outcome == OutcomeFailBackfire {
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			statusMsg += " [Luyện Cổ Phản Phệ]: Thất bại cháy nổ khiến bạn bị Phản Phệ! Tỉ lệ Luyện Cổ giảm 50% trong 20 phút."
		} else {
			player.Statuses.Add("refine_fatigue", "refine", 10*time.Minute, 0)
			statusMsg += " [Tâm Thần Mệt Mỏi]: Tỉ lệ Luyện Cổ lượt tiếp theo giảm 25% trong 10 phút."
		}
	}

	statusesRaw, err := player.Statuses.Value()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize player statuses"})
		return
	}

	// 9. Update player status in database
	// Explicit delete — SyncPlayerGus no longer deletes.
	if len(consumedGuIDs) > 0 {
		if err := db.RemovePlayerGus(tx, consumedGuIDs); err != nil {
			logger.S.Error("RefineGu: failed to remove consumed Gus:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu túi đồ: " + err.Error()})
			return
		}
	}
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		logger.S.Error("DEBUG SyncPlayerGus failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu túi đồ: " + err.Error()})
		return
	}
	updateQuery := `
		UPDATE players
		SET primeval_stones = $1, immortal_stones = $2, equipped_gus = $3, hp = $4, max_mana = $5, current_mana = $6, human_qi = $7, path_masteries = $8, soul_hp = $9, statuses = $10
		WHERE id = $11
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.HP, player.MaxMana, player.CurrentMana, player.HumanQi, player.PathMasteries, player.SoulHP, statusesRaw, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refining results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit refining transaction"})
		return
	}

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated)

	targetName := recipe.TargetGuID
	if targetSpec, ok := models.GetGu(recipe.TargetGuID, recipe.TargetRank); ok {
		targetName = targetSpec.Name
	}

	isSuccess := outcome == OutcomeSuccess

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "refine.gu", EntityType: "refining_recipe",
		EntityID:       req.RecipeID,
		RequestSummary: fmt.Sprintf("Luyện Cổ: công thức %s -> %s | Thành công: %v | Ép luyện: %v | Outcome: %s", req.RecipeID, targetName, isSuccess, isForcedRefine, outcome),
		Metadata: map[string]interface{}{
			"recipe_id":      req.RecipeID,
			"target_gu_id":   recipe.TargetGuID,
			"target_gu_name": targetName,
			"success":        isSuccess,
			"outcome":        outcome,
			"is_forced":      isForcedRefine,
		},
		IPAddress: c.ClientIP(), Success: isSuccess,
	})

	notifyTitle := "Luyện Cổ Thất Bại"
	if isSuccess {
		notifyTitle = "Luyện Cổ Thành Công"
	}
	notify.Send(db.DB, player.ID, "SYSTEM", notifyTitle, statusMsg)

	// Return response
	if isSuccess {
		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"status":         statusStr,
			"message":        statusMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
			"new_gu":         newGuInstance,
			"rate_breakdown": breakdown,
		})
	} else {
		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"status":         statusStr,
			"message":        statusMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
			"rate_breakdown": breakdown,
		})
	}
}
