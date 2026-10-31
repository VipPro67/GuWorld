package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể luyện Cổ!"})
		return
	}

	// 2b. Verify player has unlocked the recipe at 100% (ép luyện đã bị loại bỏ).
	recipeProgress, err := RecipeSvc.Get(tx, player.ID.String(), req.RecipeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở công thức này!"})
		return
	}
	if recipeProgress.CompletionRate < 100.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bí phương chưa hoàn thiện 100%! Hãy thôi diễn cho hoàn chỉnh rồi mới luyện."})
		return
	}

	// 3. Fetch Recipe
	recipe, err := fetchRefiningRecipe(tx, req.RecipeID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Refining recipe not found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	// 3a. Đường nhanh chỉ dành cho cổ thấp hơn cảnh giới người chơi và
	// không phải Tiên Cổ; còn lại bắt buộc qua phiên luyện có thời gian.
	if recipe.TargetRank >= 6 || recipe.TargetRank >= player.Rank {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cổ này cần luyện qua phiên có thời gian. Hãy bắt đầu một phiên luyện Cổ!",
			"code":  "REFINE_SESSION_REQUIRED",
		})
		return
	}

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
	_, materialQualityBonus, guRequirementsNotMet, selErr := validateRefineSelection(&player, &recipe, req.SelectedGuInstanceIDs)
	if selErr != nil {
		if strings.HasPrefix(selErr.Error(), "Gu bug not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": selErr.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": selErr.Error()})
		return
	}

	// Tập instance id đã chọn (đã trim) — dùng để phân loại Cổ bị tiêu hao.
	seenInstances := make(map[string]bool)
	for _, instID := range req.SelectedGuInstanceIDs {
		seenInstances[strings.TrimSpace(instID)] = true
	}

	// Cổ đang bị khóa bởi phiên luyện khác không thể dùng làm nguyên liệu.
	lockedCount := 0
	if err := tx.QueryRow(`SELECT COUNT(1) FROM player_gu WHERE player_id = $1 AND id = ANY($2::text[]) AND is_locked`,
		player.ID, pq.Array(req.SelectedGuInstanceIDs)).Scan(&lockedCount); err != nil {
		logger.S.Errorf("RefineGu: locked Gu check error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi kiểm tra Gu bị khóa"})
		return
	}
	if lockedCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Có Cổ nguyên liệu đang bị giữ trong phiên luyện khác!", "code": "REFINE_GU_LOCKED"})
		return
	}

	// Tiên cổ là duy nhất: xác định trước khi tính tỉ lệ để pity không giúp
	// tăng tỉ lệ cho một lần luyện chắc chắn thất bại vì Cổ đã tồn tại.
	isUniqueImmortalFail := false
	if recipe.TargetRank >= 6 {
		exists, err := db.CheckImmortalGuExists(tx, recipe.TargetGuID, recipe.TargetRank, req.SelectedGuInstanceIDs)
		if err != nil {
			logger.S.Errorf("RefineGu: CheckImmortalGuExists error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi kiểm tra Cổ duy nhất"})
			return
		}
		isUniqueImmortalFail = exists
	}
	pityEligible := !guRequirementsNotMet && !isUniqueImmortalFail

	// 7. Consume resources (Stones, Mana, Materials & Gus)
	player.CurrentMana -= manaCost
	player.PrimevalStones -= recipe.PrimevalStonesCost
	if recipe.ImmortalEssenceStonesCost > 0 {
		player.ImmortalStones -= recipe.ImmortalEssenceStonesCost
	}

	// 8. RNG Roll & 10 Factors Calculation — helper dùng chung với phiên luyện.
	rateSnapshot, err := computeRefineRate(tx, &player, recipe, nil, refineRateInput{
		materialQualityBonus: materialQualityBonus,
		pityEligible:         pityEligible,
		requirementsNotMet:   guRequirementsNotMet,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute refine rate: " + err.Error()})
		return
	}
	finalSuccessRate := rateSnapshot.Breakdown.FinalRate
	refinementRealm := rateSnapshot.RefinementRealm
	insightStacks := rateSnapshot.InsightStacks
	breakdown := rateSnapshot.Breakdown

	roll := rand.Float64()
	outcome := determineRefineOutcome(roll, finalSuccessRate, refinementRealm, recipe.TargetRank)

	var newGuInstance models.GuInstance
	var statusMsg string
	var statusStr string

	// Tiên cổ duy nhất đã tồn tại: không thể luyện thành công,
	// và cũng không bị phản phệ (backfire) vì thất bại này có nguyên nhân khách quan.
	if isUniqueImmortalFail && outcome != OutcomeFailNormal {
		outcome = OutcomeFailNormal
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
					if err == sql.ErrNoRows {
						c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matName)})
					} else {
						logger.S.Error("RefineGu FailSafe DeductItem error:", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi trừ nguyên liệu"})
					}
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
				if err == sql.ErrNoRows {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matName)})
				} else {
					logger.S.Error("RefineGu DeductItem error:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi trừ nguyên liệu"})
				}
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
				consumedGuIDs = append(consumedGuIDs, cleanInstID)
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
			cleanInstID := strings.TrimSpace(item.InstanceID)
			if seenInstances[item.InstanceID] || seenInstances[cleanInstID] {
				if item.IsVital {
					item.Hunger = 1
					item.Status = "DYING"
					item.Equipped = false
					newInventory = append(newInventory, item)
					hasVitalSurvived = true
				} else {
					consumedGuIDs = append(consumedGuIDs, cleanInstID)
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
			statusMsg += " [Luyện Cổ Phản Phệ]: Tỉ lệ Luyện Cổ giảm 95% trong 20 phút do cố luyện khi tâm thần mệt mỏi!"
		} else if hasBackfire {
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			statusMsg += " Trạng thái [Luyện Cổ Phản Phệ] (giảm 95% tỉ lệ) đã được gia hạn thêm 20 phút."
		} else if outcome == OutcomeFailBackfire {
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			statusMsg += " [Luyện Cổ Phản Phệ]: Thất bại cháy nổ khiến bạn bị Phản Phệ! Tỉ lệ Luyện Cổ giảm 95% trong 20 phút."
		} else {
			player.Statuses.Add("refine_fatigue", "refine", 10*time.Minute, 0)
			statusMsg += " [Tâm Thần Mệt Mỏi]: Tỉ lệ Luyện Cổ lượt tiếp theo giảm 50% trong 10 phút."
		}
	}

	// Per-rank pity chỉ đổi sau lần thử có công thức nguyên liệu đúng và mục tiêu
	// chưa tồn tại: thất bại +1 tầng; thành công reset tầng của rank đó.
	insightKey := gamefx.RefineInsightKey(recipe.TargetRank)
	if outcome == OutcomeSuccess && pityEligible {
		if refineInsightStacks(&player, recipe.TargetRank) > 0 || player.Statuses.Has(insightKey) {
			player.Statuses.Remove(insightKey)
			statusMsg += " [Tâm Đắc viên mãn]: Luyện thành công, pity rank này reset về 0."
		}
	} else if outcome != OutcomeSuccess && pityEligible {
		newStacks := insightStacks + 1
		player.Statuses.Add(insightKey, "refine", 0, float64(newStacks))
		statusMsg += fmt.Sprintf(" [Tâm Đắc R%d]: Tích %d tầng (+%d%% tỉ lệ luyện R%d, không giới hạn).", recipe.TargetRank, newStacks, newStacks, recipe.TargetRank)
	} else if isUniqueImmortalFail {
		statusMsg += " [Tâm Đắc]: Tiên cổ cùng bậc đã tồn tại; lần này không tăng tầng Tâm Đắc."
	} else if guRequirementsNotMet {
		statusMsg += " [Tâm Đắc]: Cổ nguyên liệu không khớp công thức; lần này không tăng tầng Tâm Đắc."
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
		if db.IsImmortalConflict(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Tiên cổ là duy nhất, đã tồn tại trong thiên hạ, luyện chế thất bại!"})
			return
		}
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
	if outcome == OutcomeSuccess {
		if err := TrackDailyMissionProgress(tx, player.ID, player.Rank, "refine:"+newGuInstance.InstanceID, "refine", "", 1); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật nhiệm vụ hằng ngày"})
			return
		}
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit refining transaction"})
		return
	}

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, ItemInventoryDomain, CombatStatsDomain)

	targetName := recipe.TargetGuID
	if targetSpec, ok := models.GetGu(recipe.TargetGuID, recipe.TargetRank); ok {
		targetName = targetSpec.Name
	}

	isSuccess := outcome == OutcomeSuccess

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "refine.gu", EntityType: "refining_recipe",
		EntityID:       req.RecipeID,
		RequestSummary: fmt.Sprintf("Luyện Cổ: công thức %s -> %s | Thành công: %v | Outcome: %s", req.RecipeID, targetName, isSuccess, outcome),
		Metadata: map[string]interface{}{
			"recipe_id":      req.RecipeID,
			"target_gu_id":   recipe.TargetGuID,
			"target_gu_name": targetName,
			"success":        isSuccess,
			"outcome":        outcome,
			"is_forced":      false,
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
