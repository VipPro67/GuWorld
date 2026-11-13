package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ConceiveCandidate represents an eligible Gu that can be targeted for conception.
type ConceiveCandidate struct {
	ID                     string   `json:"id"`
	Rank                   int      `json:"rank"`
	Name                   string   `json:"name"`
	Paths                  []string `json:"paths"`
	PrimaryPath            string   `json:"primary_path"`
	ImageURL               string   `json:"image_url"`
	Description            string   `json:"description"`
	ExistingCompletionRate float64  `json:"existing_completion_rate"`
	IsUnlocked             bool     `json:"is_unlocked"`
	RequiredThought        float64  `json:"required_thought"`
	RequiredStones         float64  `json:"required_stones"`
	RequiredImmortalStones float64  `json:"required_immortal_stones"`
	EstimatedAccuracy      float64  `json:"estimated_accuracy"`
	HasBacklashRisk        bool     `json:"has_backlash_risk"`
	BacklashRiskChance     float64  `json:"backlash_risk_chance"`
}

// ConceiveRecipeRequest represents the request body for POST /api/refine/conceive.
type ConceiveRecipeRequest struct {
	TargetGuID           string `json:"target_gu_id" binding:"required"`
	TargetRank           int    `json:"target_rank" binding:"required,min=1,max=9"`
	CatalystGuInstanceID string `json:"catalyst_gu_instance_id,omitempty"`
	CatalystItemID       string `json:"catalyst_item_id,omitempty"`
}

// BacklashDetail represents the penalty inflicted on Trí Đạo Phản Phệ.
type BacklashDetail struct {
	HPLost                    float64 `json:"hp_lost"`
	ManaLost                  float64 `json:"mana_lost"`
	ThoughtLost               float64 `json:"thought_lost"`
	SoulInjuryCooldownMinutes int     `json:"soul_injury_cooldown_minutes"`
}

// ConceiveRecipeResponse represents the response body for POST /api/refine/conceive.
type ConceiveRecipeResponse struct {
	Status           string          `json:"status"` // "SUCCESS" | "DEVIATED" | "BACKLASH"
	IsDeviated       bool            `json:"is_deviated"`
	RecipeID         string          `json:"recipe_id,omitempty"`
	GuID             string          `json:"gu_id,omitempty"`
	GuName           string          `json:"gu_name,omitempty"`
	Rank             int             `json:"rank,omitempty"`
	CompletionRate   float64         `json:"completion_rate"`
	IsInstant100     bool            `json:"is_instant_100"`
	Message          string          `json:"message"`
	CatalystConsumed string          `json:"catalyst_consumed,omitempty"`
	ThoughtCost      float64         `json:"thought_cost"`
	StonesCost       float64         `json:"stones_cost"`
	Backlash         *BacklashDetail `json:"backlash,omitempty"`
}

// extractPlayerPathPoints returns attainment points for a path from player path masteries.
func extractPlayerPathPoints(pathMasteries map[string]int, p string) int {
	if pathMasteries == nil {
		return 0
	}
	clean := strings.TrimSuffix(strings.ToLower(strings.Trim(p, " []\"")), " path")
	if pts, ok := pathMasteries[clean]; ok {
		return pts
	}
	if pts, ok := pathMasteries[p]; ok {
		return pts
	}
	return 0
}

// GetConceiveCandidates returns eligible Gu that the player can target for recipe conception.
func GetConceiveCandidates(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	player, err := PlayerSvc.ProjectStatusEquipped(db.DB, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player state"})
		return
	}

	pathFilter := strings.TrimSpace(c.Query("path"))
	rankFilter := 0
	if r, err := strconv.Atoi(c.Query("rank")); err == nil && r >= 1 && r <= 9 {
		rankFilter = r
	}
	searchFilter := strings.TrimSpace(c.Query("search"))

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	query := `
		SELECT 
			g.id, g.rank, g.name, g.path, g.image_url, g.description,
			COALESCE(p.completion_rate, 0) as existing_rate,
			COALESCE(p.is_unlocked, false) as is_unlocked
		FROM gu_dictionary g
		LEFT JOIN player_recipe_progress p 
			ON p.player_id = $1 
			AND p.recipe_id = ('recipe_' || g.id || '_' || g.rank::text || '_1')
		WHERE ($2 = '' OR g.path::text ILIKE '%' || $2 || '%')
		  AND ($3 = 0 OR g.rank = $3)
		  AND ($4 = '' OR g.name ILIKE '%' || $4 || '%' OR g.id ILIKE '%' || $4 || '%')
		  AND (p.completion_rate IS NULL OR p.completion_rate < 100.0)
		ORDER BY g.rank ASC, g.name ASC, g.id ASC
		LIMIT $5 OFFSET $6
	`

	rows, err := db.DB.QueryContext(c.Request.Context(), query, player.ID.String(), pathFilter, rankFilter, searchFilter, limit, offset)
	if err != nil {
		logger.S.Errorf("GetConceiveCandidates query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query candidates"})
		return
	}
	defer rows.Close()

	wisdomRealm := gamefx.PathMasteryLevel(extractPlayerPathPoints(player.PathMasteries, "wisdom"))
	refineRealm := gamefx.PathMasteryLevel(extractPlayerPathPoints(player.PathMasteries, "refinement"))
	wisdomDaoMarks := int64(0)
	if player.DaoMarks != nil {
		wisdomDaoMarks = player.DaoMarks["wisdom"]
	}

	candidates := make([]ConceiveCandidate, 0)
	for rows.Next() {
		var (
			cnd         ConceiveCandidate
			pathBytes   []byte
			description sql.NullString
			imageURL    sql.NullString
		)

		if err := rows.Scan(
			&cnd.ID, &cnd.Rank, &cnd.Name, &pathBytes, &imageURL, &description,
			&cnd.ExistingCompletionRate, &cnd.IsUnlocked,
		); err != nil {
			continue
		}

		if description.Valid {
			cnd.Description = description.String
		}
		if imageURL.Valid {
			cnd.ImageURL = imageURL.String
		}

		var paths []string
		if err := json.Unmarshal(pathBytes, &paths); err == nil && len(paths) > 0 {
			cnd.Paths = paths
			cnd.PrimaryPath = gamefx.CleanPath(paths[0])
		} else {
			cnd.Paths = []string{string(pathBytes)}
			cnd.PrimaryPath = gamefx.CleanPath(string(pathBytes))
		}

		targetPts := extractPlayerPathPoints(player.PathMasteries, cnd.PrimaryPath)
		targetRealm := gamefx.PathMasteryLevel(targetPts)

		cnd.RequiredThought = gamefx.ConceiveThoughtCost(cnd.Rank)
		cnd.RequiredStones = gamefx.ConceiveStoneCost(cnd.Rank)
		cnd.RequiredImmortalStones = gamefx.ConceiveImmortalStoneCost(cnd.Rank)

		cnd.EstimatedAccuracy = gamefx.CalculateConceiveAccuracy(cnd.Rank, targetRealm, wisdomRealm, wisdomDaoMarks, 0.0)
		risk := gamefx.CalculateBacklashRisk(player.Rank, cnd.Rank, wisdomRealm, refineRealm)
		cnd.BacklashRiskChance = risk
		cnd.HasBacklashRisk = risk > 0.0

		candidates = append(candidates, cnd)
	}

	c.JSON(http.StatusOK, gin.H{
		"candidates":             candidates,
		"player_thought":         player.CurrentThought,
		"player_max_thought":     player.MaxThought,
		"player_stones":          player.PrimevalStones,
		"player_immortal_stones": player.ImmortalStones,
		"player_rank":            player.Rank,
	})
}

// ConceiveRecipe executes the targeted recipe conception action.
func ConceiveRecipe(c *gin.Context) {
	var req ConceiveRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thông tin Cổ mục tiêu không hợp lệ"})
		return
	}

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

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player status"})
		}
		return
	}

	// 1. Check Cooldown / Soul Injury status
	if player.Statuses.Has("wisdom_backlash_soul_injury") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hồn phách đạo hữu đang chấn thương do Trí Đạo phản phệ, tạm thời không thể suy tính thiên cơ!"})
		return
	}

	// 2. Validate Target Gu
	var (
		targetGuName string
		targetPathRaw []byte
	)
	err = tx.QueryRow(`SELECT name, path FROM gu_dictionary WHERE id = $1 AND rank = $2`, req.TargetGuID, req.TargetRank).Scan(&targetGuName, &targetPathRaw)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng mục tiêu trong Cổ Phổ!"})
		return
	}

	var targetPaths []string
	if err := json.Unmarshal(targetPathRaw, &targetPaths); err != nil || len(targetPaths) == 0 {
		targetPaths = []string{string(targetPathRaw)}
	}
	targetPrimaryPath := gamefx.CleanPath(targetPaths[0])

	// 3. Out-of-Scope check: Mortal cannot conceive Immortal Gu
	if player.Rank <= 5 && req.TargetRank >= 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phàm nhân không thể suy tính Tiên Cổ phương! Cần thăng tiên mới có thể khám phá thiên cơ Tiên Cổ."})
		return
	}

	// 4. Check if player already completed this recipe 100%
	targetRecipeID := fmt.Sprintf("recipe_%s_%d_1", req.TargetGuID, req.TargetRank)
	existingProg, err := RecipeSvc.Get(tx, player.ID.String(), targetRecipeID)
	if err == nil && existingProg != nil && existingProg.CompletionRate >= 100.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã hoàn toàn nắm giữ 100% bí phương của Cổ trùng này!"})
		return
	}

	// 5. Cost verification
	thoughtCost := gamefx.ConceiveThoughtCost(req.TargetRank)
	stoneCost := gamefx.ConceiveStoneCost(req.TargetRank)
	immortalStoneCost := gamefx.ConceiveImmortalStoneCost(req.TargetRank)

	if player.CurrentThought < thoughtCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Niệm Lực để khởi niệm (cần %.0f, hiện có %.1f)!", thoughtCost, player.CurrentThought)})
		return
	}
	if player.PrimevalStones < stoneCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch (cần %.0f, hiện có %.0f)!", stoneCost, player.PrimevalStones)})
		return
	}
	if immortalStoneCost > 0 && player.ImmortalStones < immortalStoneCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch (cần %.0f, hiện có %.0f)!", immortalStoneCost, player.ImmortalStones)})
		return
	}

	// 6. Catalyst Verification & Consumption
	catalystBonus := 0.0
	catalystConsumedName := ""

	if req.CatalystGuInstanceID != "" {
		// Sacrificing a living specimen Gu from inventory
		guInst, _, found := models.FindGuByInstanceID(player.GuInventory, req.CatalystGuInstanceID)
		if !found {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Cổ phôi trong Không Khiếu!"})
			return
		}
		if guInst.IsVital {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể dùng Cổ Bản Mệnh làm vật dẫn hiến tế!"})
			return
		}
		var isGuLocked bool
		if err := tx.QueryRow(`SELECT is_locked FROM player_gu WHERE id = $1 AND player_id = $2`, req.CatalystGuInstanceID, player.ID.String()).Scan(&isGuLocked); err == nil && isGuLocked {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đang bị khóa, hãy mở khóa trước khi dùng làm vật dẫn!"})
			return
		}
		if guInst.Equipped {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đang trang bị, hãy tháo ra trước khi dùng làm vật dẫn!"})
			return
		}
		if guInst.Status == "DEAD" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đã chết, không còn mang mạch đạo ngân sống để phân tích!"})
			return
		}
		if guInst.Rank > req.TargetRank {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Phẩm giai Cổ phôi không được cao hơn Cổ mục tiêu!"})
			return
		}

		// Verify path match
		var guSpec models.GuDictionaryItem
		if spec, ok := models.GetGu(guInst.GuID, guInst.Rank); ok {
			guSpec = spec
		} else {
			var specPathRaw []byte
			_ = tx.QueryRow(`SELECT name, path FROM gu_dictionary WHERE id = $1 AND rank = $2`, guInst.GuID, guInst.Rank).Scan(&guSpec.Name, &specPathRaw)
			_ = json.Unmarshal(specPathRaw, &guSpec.Path)
		}

		guMatched := false
		for _, p := range guSpec.Path {
			if gamefx.CleanPath(p) == targetPrimaryPath {
				guMatched = true
				break
			}
		}
		if !guMatched {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ phôi không cùng Lưu Phái với Cổ mục tiêu!"})
			return
		}

		catalystBonus = gamefx.GuCatalystBonus(guInst.Rank, req.TargetRank)
		catalystConsumedName = fmt.Sprintf("%s (Chuyển %d)", guSpec.Name, guInst.Rank)

		// Remove Gu instance from player_gu table and memory inventory
		if err := db.RemovePlayerGuForPlayer(tx, player.ID.String(), req.CatalystGuInstanceID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tiêu hao Cổ phôi"})
			return
		}
		player.GuInventory = models.RemoveGuByInstanceID(player.GuInventory, req.CatalystGuInstanceID)

	} else if req.CatalystItemID != "" {
		// Using material catalyst
		var (
			matName string
			matType string
			matRank int
			matPathRaw []byte
		)
		err = tx.QueryRow(`SELECT name, item_type, rank, path FROM item_dictionary WHERE id = $1`, req.CatalystItemID).Scan(&matName, &matType, &matRank, &matPathRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vật liệu dẫn dắt không tồn tại!"})
			return
		}
		if matType != "Material" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ có thể dùng vật phẩm thuộc loại Nguyên Liệu (Material) làm vật dẫn!"})
			return
		}
		if matRank > req.TargetRank {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Phẩm giai nguyên liệu không được cao hơn Cổ mục tiêu!"})
			return
		}

		var matPaths []string
		if err := json.Unmarshal(matPathRaw, &matPaths); err != nil || len(matPaths) == 0 {
			rawStr := strings.TrimSpace(string(matPathRaw))
			if rawStr != "" && rawStr != "[]" {
				matPaths = []string{rawStr}
			}
		}
		matMatched := false
		for _, mp := range matPaths {
			if gamefx.CleanPath(mp) == targetPrimaryPath {
				matMatched = true
				break
			}
		}
		if !matMatched {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nguyên liệu dẫn dắt không cùng Lưu Phái với Cổ mục tiêu!"})
			return
		}

		requiredQty := gamefx.MaterialCatalystRequiredQty(matRank, req.TargetRank)
		if InventorySvc == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Dịch vụ kho đồ không khả dụng"})
			return
		}
		if err := InventorySvc.DeductItem(tx, player.ID.String(), req.CatalystItemID, requiredQty); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ số lượng nguyên liệu (cần %d cái %s)!", requiredQty, matName)})
			return
		}

		catalystBonus = gamefx.MaterialCatalystBonus(matRank, req.TargetRank)
		catalystConsumedName = fmt.Sprintf("%d x %s", requiredQty, matName)
	}

	// 7. Deduct resource costs
	player.CurrentThought -= thoughtCost
	player.PrimevalStones -= stoneCost
	if immortalStoneCost > 0 {
		player.ImmortalStones -= immortalStoneCost
	}

	// 8. Attainment and Realm points
	wisdomRealm := gamefx.PathMasteryLevel(extractPlayerPathPoints(player.PathMasteries, "wisdom"))
	refineRealm := gamefx.PathMasteryLevel(extractPlayerPathPoints(player.PathMasteries, "refinement"))
	targetRealm := gamefx.PathMasteryLevel(extractPlayerPathPoints(player.PathMasteries, targetPrimaryPath))
	wisdomDaoMarks := int64(0)
	if player.DaoMarks != nil {
		wisdomDaoMarks = player.DaoMarks["wisdom"]
	}

	// 9. Evaluate Trí Đạo Phản Phệ (Backlash)
	backlashRisk := gamefx.CalculateBacklashRisk(player.Rank, req.TargetRank, wisdomRealm, refineRealm)
	if backlashRisk > 0 && rand.Float64() < backlashRisk {
		// TRIGGER BACKLASH!
		hpLoss := math.Round(player.HP*0.5*10) / 10.0
		manaLoss := math.Round(player.CurrentMana*0.5*10) / 10.0
		thoughtLoss := player.CurrentThought

		player.CurrentThought = 0
		player.CurrentMana = math.Max(0.0, player.CurrentMana-manaLoss)
		player.HP = math.Max(1.0, player.HP-hpLoss)

		soulCooldown := 0
		if req.TargetRank >= 6 {
			soulCooldown = 20
			player.Statuses.Add("wisdom_backlash_soul_injury", "conceive", 20*time.Minute, 0)
		}

		// Save player state in DB
		_, err = tx.Exec(`
			UPDATE players
			SET current_thought = $1, primeval_stones = $2, immortal_stones = $3,
			    current_mana = $4, hp = $5, statuses = $6,
			    state_revision = COALESCE(state_revision, 0) + 1
			WHERE id = $7
		`, player.CurrentThought, player.PrimevalStones, player.ImmortalStones,
			player.CurrentMana, player.HP, player.Statuses, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu kết quả phản phệ"})
			return
		}

		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit phản phệ"})
			return
		}

		player.StateRevision++
		ClearPendingPlayerQueues(player.ID.String())
		PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, ItemInventoryDomain)

		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID:       &authID,
			Action:         "recipe.conceive.backlash",
			EntityType:     "refining_recipe",
			EntityID:       targetRecipeID,
			RequestSummary: fmt.Sprintf("Trí Đạo Phản Phệ khi suy tính %s (Mất %.0f HP, %.0f Mana)", targetGuName, hpLoss, manaLoss),
			Success:        true,
			IPAddress:      c.ClientIP(),
		})

		c.JSON(http.StatusOK, ConceiveRecipeResponse{
			Status:           "BACKLASH",
			IsDeviated:       false,
			Message:          "Trí Đạo Phản Phệ! Cưỡng cầu thiên cơ thất bại khiến đạo hữu thổ huyết chấn thương, Niệm Lực tiêu tan hoàn toàn!",
			ThoughtCost:      thoughtCost,
			StonesCost:       stoneCost,
			CatalystConsumed: catalystConsumedName,
			Backlash: &BacklashDetail{
				HPLost:                    hpLoss,
				ManaLost:                  manaLoss,
				ThoughtLost:               thoughtLoss,
				SoulInjuryCooldownMinutes: soulCooldown,
			},
		})
		return
	}

	// 10. Accuracy & Conceive Outcome (No Backlash)
	accuracy := gamefx.CalculateConceiveAccuracy(req.TargetRank, targetRealm, wisdomRealm, wisdomDaoMarks, catalystBonus)
	accRoll := rand.Float64()

	finalGuID := req.TargetGuID
	finalRank := req.TargetRank
	finalGuName := targetGuName
	isDeviated := false

	if accRoll > accuracy {
		// Deviation occurs!
		deviatedID, deviatedRank, deviatedName, found := selectDeviationCandidate(tx, player.ID.String(), targetPrimaryPath, req.TargetRank, req.TargetGuID)
		if found {
			finalGuID = deviatedID
			finalRank = deviatedRank
			finalGuName = deviatedName
			isDeviated = true
		}
	}

	// 11. Initial Completion Rate & Inspiration Flash
	initialRate, isInstant100 := gamefx.CalculateInitialConceiveProgress(wisdomRealm, refineRealm, targetRealm, finalRank)
	finalRecipeID := fmt.Sprintf("recipe_%s_%d_1", finalGuID, finalRank)

	// Upsert recipe progress
	existingRecipeProg, err := RecipeSvc.Get(tx, player.ID.String(), finalRecipeID)
	now := time.Now()
	var finalRate float64

	if err == nil && existingRecipeProg != nil {
		finalRate = math.Max(existingRecipeProg.CompletionRate, initialRate)
		existingRecipeProg.CompletionRate = finalRate
		existingRecipeProg.IsUnlocked = true
		existingRecipeProg.DeduceCount++
		existingRecipeProg.LastDeduceAt = &now
		if err := RecipeSvc.Upsert(tx, existingRecipeProg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu tiến trình tàn phương"})
			return
		}
	} else {
		finalRate = initialRate
		if err := RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       player.ID.String(),
			RecipeID:       finalRecipeID,
			CompletionRate: finalRate,
			IsUnlocked:     true,
			DeduceCount:    1,
			LastDeduceAt:   &now,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo tàn phương"})
			return
		}
	}

	// 12. Save player state
	_, err = tx.Exec(`
		UPDATE players
		SET current_thought = $1, primeval_stones = $2, immortal_stones = $3,
		    state_revision = COALESCE(state_revision, 0) + 1
		WHERE id = $4
	`, player.CurrentThought, player.PrimevalStones, player.ImmortalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu tài nguyên người chơi"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit giao dịch"})
		return
	}

	player.StateRevision++
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, ItemInventoryDomain, RecipesDomain)

	// Format response message
	var msg string
	if isInstant100 {
		msg = fmt.Sprintf("Linh quang chớp lóe! Đạo hữu đốn ngộ huyền cơ vạn cổ, lập tức hoàn thiện 100%% Bí Phương [%s]!", finalGuName)
	} else if isDeviated {
		msg = fmt.Sprintf("Thiên cơ rẽ nhánh! Ý niệm chuyển hóa, đạo hữu lĩnh ngộ được Tàn Phương Cổ liên quan [%s] (Độ hoàn thiện: %.1f%%)!", finalGuName, finalRate)
	} else {
		msg = fmt.Sprintf("Khai sáng thành công! Đạo hữu đã lĩnh ngộ được Tàn Phương [%s] (Độ hoàn thiện: %.1f%%)!", finalGuName, finalRate)
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "recipe.conceive",
		EntityType:     "refining_recipe",
		EntityID:       finalRecipeID,
		RequestSummary: fmt.Sprintf("Khai sáng bí phương %s: %s", finalRecipeID, msg),
		Metadata: map[string]interface{}{
			"target_gu_id":   req.TargetGuID,
			"target_rank":    req.TargetRank,
			"final_gu_id":    finalGuID,
			"final_rank":     finalRank,
			"is_deviated":    isDeviated,
			"is_instant_100": isInstant100,
			"initial_rate":   finalRate,
			"catalyst":       catalystConsumedName,
		},
		Success:   true,
		IPAddress: c.ClientIP(),
	})

	statusStr := "SUCCESS"
	if isDeviated {
		statusStr = "DEVIATED"
	}

	c.JSON(http.StatusOK, ConceiveRecipeResponse{
		Status:           statusStr,
		IsDeviated:       isDeviated,
		RecipeID:         finalRecipeID,
		GuID:             finalGuID,
		GuName:           finalGuName,
		Rank:             finalRank,
		CompletionRate:   finalRate,
		IsInstant100:     isInstant100,
		Message:          msg,
		CatalystConsumed: catalystConsumedName,
		ThoughtCost:      thoughtCost,
		StonesCost:       stoneCost,
	})
}

// selectDeviationCandidate chooses a related Gu of the same path and rank or rank-1 that the player hasn't 100% unlocked.
func selectDeviationCandidate(tx *sql.Tx, playerID, path string, targetRank int, targetGuID string) (string, int, string, bool) {
	query := `
		SELECT g.id, g.rank, g.name
		FROM gu_dictionary g
		LEFT JOIN player_recipe_progress p 
			ON p.player_id = $1 
			AND p.recipe_id = ('recipe_' || g.id || '_' || g.rank::text || '_1')
		WHERE g.path::text ILIKE '%' || $2 || '%'
		  AND (g.rank = $3 OR ($3 > 1 AND g.rank = $3 - 1))
		  AND g.id != $4
		  AND (p.completion_rate IS NULL OR p.completion_rate < 100.0)
		ORDER BY -ln(greatest(random(), 1e-9)) / (CASE WHEN g.rank = $3 THEN 2.0 ELSE 1.0 END) ASC
		LIMIT 1
	`
	var (
		id   string
		rank int
		name string
	)
	err := tx.QueryRow(query, playerID, path, targetRank, targetGuID).Scan(&id, &rank, &name)
	if err != nil {
		return "", 0, "", false
	}
	return id, rank, name, true
}
