package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type FeedPetRequest struct {
	PetID  string `json:"pet_id" binding:"required"`
	ItemID string `json:"item_id" binding:"required"`
}

type FormationRequest struct {
	Slots []string `json:"slots" binding:"required"` // ordered list of pet IDs
}

type SleepRequest struct {
	PetID    string `json:"pet_id" binding:"required"`
	Sleeping bool   `json:"sleeping"`
}

type ReleaseRequest struct {
	PetID string `json:"pet_id" binding:"required"`
}

type MergeRequest struct {
	TargetID  string   `json:"target_id" binding:"required"`
	SourceIDs []string `json:"source_ids" binding:"required"`
}

func ListPlayerPets(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	pets, err := models.LoadPlayerPets(db.DB, authID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load pets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pets": pets})
}

func FeedPet(c *gin.Context) {
	var req FeedPetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Load pet
	var petID, nickname, state string
	var hunger, maxHunger, count float64
	err = tx.QueryRow(`
		SELECT id, nickname, hunger, max_hunger, state, count
		FROM player_pets WHERE id = $1 AND player_id = $2 FOR UPDATE
	`, req.PetID, authID).Scan(&petID, &nickname, &hunger, &maxHunger, &state, &count)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pet không tồn tại."})
		return
	}

	// Get item spec and validate it is food
	item, ok := models.ItemRegistry[req.ItemID]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không hợp lệ."})
		return
	}
	if item.Type != "Food" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ có thể cho nô thú ăn vật phẩm loại thức ăn (Food)."})
		return
	}

	// Check item exists in inventory (any bucket)
	qty, itemCategory, err := InventorySvc.GetQuantityInAnyCategory(tx, authID.String(), req.ItemID, "consumables", "materials")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra túi đồ."})
		return
	}
	if qty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm này trong túi đồ."})
		return
	}

	// Get item hunger value (data-driven; fallback to rank*10)
	hungerValue := item.Effects["hunger"]
	if hungerValue <= 0 {
		hungerValue = float64(item.Rank * 10)
	}

	// Consume item
	if err := InventorySvc.RemoveItem(tx, authID.String(), req.ItemID, itemCategory, 1); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tiêu hao vật phẩm."})
		return
	}

	hungerValue /= count

	// Feed
	newHunger := hunger + hungerValue
	if newHunger > maxHunger {
		newHunger = maxHunger
	}
	_, err = tx.Exec(`UPDATE player_pets SET hunger = $1, updated_at = NOW() WHERE id = $2`, newHunger, petID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pet hunger"})
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin người chơi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "pet.feed",
		EntityType:     "player_pet",
		EntityID:       petID,
		RequestSummary: fmt.Sprintf("Cho linh thú %s ăn: 1x %s | Độ no: %.0f/%.0f", nickname, item.Name, newHunger, maxHunger),
		Metadata: map[string]interface{}{
			"pet_id":     petID,
			"pet_name":   nickname,
			"item_id":    req.ItemID,
			"item_name":  item.Name,
			"new_hunger": newHunger,
			"max_hunger": maxHunger,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"pet_id":     petID,
		"nickname":   nickname,
		"hunger":     newHunger,
		"max_hunger": maxHunger,
		"count":      count,
		"restored":   hungerValue,
		"item_used":  req.ItemID,
	})
}

func SetPetSleep(c *gin.Context) {
	var req SleepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	newState := "ACTIVE"
	if req.Sleeping {
		newState = "SLEEPING"
	}

	result, err := db.DB.Exec(`
		UPDATE player_pets SET state = $1, updated_at = NOW()
		WHERE id = $2 AND player_id = $3
	`, newState, req.PetID, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pet state"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pet không tồn tại."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pet_id": req.PetID, "state": newState})
}

func SetPetFormation(c *gin.Context) {
	var req FormationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Load current player rank for max formation slots
	var playerRank int
	err := db.DB.QueryRow(`SELECT rank FROM players WHERE id = $1`, authID).Scan(&playerRank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	// Calculate max formation slots (1 + rank, cap 8)
	maxSlots := 1 + playerRank
	if maxSlots > 8 {
		maxSlots = 8
	}

	if len(req.Slots) > maxSlots {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quá số slot đội hình cho phép (tối đa " + itoa(maxSlots) + ")."})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// Reset all formation slots
	_, err = tx.Exec(`UPDATE player_pets SET formation_slot = 0 WHERE player_id = $1`, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset formation"})
		return
	}

	// Set new formation
	for i, petIDStr := range req.Slots {
		petUUID, errParse := uuid.Parse(petIDStr)
		if errParse != nil {
			continue
		}
		_, errUpd := tx.Exec(`
			UPDATE player_pets SET formation_slot = $1, updated_at = NOW()
			WHERE id = $2 AND player_id = $3
		`, i+1, petUUID, authID)
		if errUpd != nil {
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"formation": req.Slots, "max_slots": maxSlots})
}

func ReleasePet(c *gin.Context) {
	var req ReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := db.DB.Exec(`
		DELETE FROM player_pets WHERE id = $1 AND player_id = $2
	`, req.PetID, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to release pet"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pet không tồn tại."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"released": true, "pet_id": req.PetID})
}

func MergePets(c *gin.Context) {
	var req MergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if len(req.SourceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần chọn ít nhất 1 pet nguồn để gộp."})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// Load target
	var targetID, targetTemplateID, targetNickname string
	var targetRank, targetSubRank, targetCount int
	var targetHP, targetMaxHP, targetAtk, targetDef, targetSpeed float64
	err = tx.QueryRow(`
		SELECT id, monster_template_id, nickname, rank, sub_rank, hp, max_hp, base_attack, base_defense, speed, count
		FROM player_pets WHERE id = $1 AND player_id = $2 FOR UPDATE
	`, req.TargetID, authID).Scan(
		&targetID, &targetTemplateID, &targetNickname, &targetRank, &targetSubRank,
		&targetHP, &targetMaxHP, &targetAtk, &targetDef, &targetSpeed, &targetCount,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target pet không tồn tại."})
		return
	}

	// Load player's max_soul to compute pack merge limit
	var maxSoul int
	err = tx.QueryRow(`SELECT max_soul FROM players WHERE id = $1`, authID).Scan(&maxSoul)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc Hồn Phách."})
		return
	}
	mergeLimit := mergePackLimit(maxSoul)

	// Validate all sources: same template, same player, formation_slot=0.
	// Sum each source's `count` so merging a pack source adds all its individuals.
	addedCount := 0
	for _, srcID := range req.SourceIDs {
		var srcTemplateID string
		var srcFormationSlot, srcCount int
		errSrc := tx.QueryRow(`
			SELECT monster_template_id, formation_slot, count
			FROM player_pets WHERE id = $1 AND player_id = $2 FOR UPDATE
		`, srcID, authID).Scan(&srcTemplateID, &srcFormationSlot, &srcCount)
		if errSrc != nil {
			continue
		}
		if srcTemplateID != targetTemplateID {
			continue
		}
		if srcFormationSlot > 0 {
			continue // can't merge from formation
		}
		addedCount += srcCount
	}

	if addedCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có pet nguồn hợp lệ (cùng loại, không trong đội hình)."})
		return
	}

	// Merge: increase count, recompute stats sub-linearly from base (no compound)
	newCount := targetCount + addedCount
	if newCount > mergeLimit {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       fmt.Sprintf("Bầy tối đa %d cá thể (giới hạn = max_soul/%d). Đạo hữu cần tăng Hồn Phách để gộp thêm.", mergeLimit, mergeSoulDivisor),
			"merge_limit": mergeLimit,
		})
		return
	}

	newMaxHP := packScaledStat(rebaseStat(targetMaxHP, targetCount), newCount)
	newAtk := packScaledStat(rebaseStat(targetAtk, targetCount), newCount)
	newDef := packScaledStat(rebaseStat(targetDef, targetCount), newCount)

	_, err = tx.Exec(`
		UPDATE player_pets
		SET count = $1, hp = $2, max_hp = $3, base_attack = $4, base_defense = $5, updated_at = NOW()
		WHERE id = $6
	`, newCount, newMaxHP, newMaxHP, newAtk, newDef, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to merge pets"})
		return
	}

	// Delete valid sources
	for _, srcID := range req.SourceIDs {
		var srcTemplateID string
		var srcFormationSlot int
		errSrc := tx.QueryRow(`
			SELECT monster_template_id, formation_slot
			FROM player_pets WHERE id = $1 AND player_id = $2
		`, srcID, authID).Scan(&srcTemplateID, &srcFormationSlot)
		if errSrc != nil {
			continue
		}
		if srcTemplateID != targetTemplateID || srcFormationSlot > 0 {
			continue
		}
		_, _ = tx.Exec(`DELETE FROM player_pets WHERE id = $1 AND player_id = $2`, srcID, authID)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	displayName := targetNickname
	if newCount > 1 {
		displayName = "Bầy " + targetNickname + " (" + itoa(newCount) + ")"
	}

	c.JSON(http.StatusOK, gin.H{
		"target_id":   targetID,
		"nickname":    displayName,
		"count":       newCount,
		"max_hp":      newMaxHP,
		"attack":      newAtk,
		"defense":     newDef,
		"merge_limit": mergeLimit,
	})
}

const (
	// mergeSoulDivisor divides max_soul to derive the per-pack merge limit.
	mergeSoulDivisor = 10
	// packScaleExponent controls sub-linear stat scaling of a merged pack.
	packScaleExponent = 0.65
)

// mergePackLimit returns the max individuals allowed in a single pack,
// proportional to the player's max_soul (min 1).
func mergePackLimit(maxSoul int) int {
	limit := maxSoul / mergeSoulDivisor
	if limit < 1 {
		return 1
	}
	return limit
}

// rebaseStat converts a stored (possibly already scaled) stat back to the
// base single-individual value using the current pack count.
func rebaseStat(current float64, currentCount int) float64 {
	if currentCount <= 1 {
		return current
	}
	return current / mathPow(float64(currentCount), packScaleExponent)
}

// packScaledStat scales a base single-individual stat for a pack of `count`.
func packScaledStat(base float64, count int) float64 {
	return mathRound(base * mathPow(float64(count), packScaleExponent))
}

func mathPow(x, y float64) float64 {
	if y == 0 {
		return 1
	}
	if x <= 0 {
		return 0
	}
	// x^y = exp(y * ln(x))
	return math.Exp(y * math.Log(x))
}

func mathRound(x float64) float64 {
	return math.Round(x*10) / 10
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
