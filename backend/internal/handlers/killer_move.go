package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	killerMoveSupportCostRate     = 0.8
	killerMoveBaseSuccessRate     = 95.0
	killerMoveSupportCountPenalty = 5.0
	// TODO: dùng cổ trùng để tăng tỉ lệ thành công thay vì Wisdom stat
	killerMoveAttainmentBonusFactor   = 2.0
	killerMovePathConflictPenaltyBase = 10.0
	killerMovePathOppositionPenalty   = 20.0
	killerMoveMinSuccessRate          = 5.0
	killerMoveMaxSuccessRate          = 100.0
	killerMoveBacklashManaFactor      = 0.5
	killerMoveMasteryBonus            = 15.0
	killerMoveMaxSupport              = 8
	killerMovePathMasteryGain         = 2
)

type CalculateKillerMoveRequest struct {
	CorePlayerGuID     string   `json:"core_player_gu_id" binding:"required"`
	SupportPlayerGuIDs []string `json:"support_player_gu_ids"`
}

type CreateKillerMoveRequest struct {
	Name               string   `json:"name" binding:"required"`
	MoveType           string   `json:"move_type"`
	CorePlayerGuID     string   `json:"core_player_gu_id" binding:"required"`
	SupportPlayerGuIDs []string `json:"support_player_gu_ids"`
}

type CastKillerMoveRequest struct {
	KillerMoveID string `json:"killer_move_id" binding:"required"`
	TargetID     string `json:"target_id"`
}

type KillerMovePreview struct {
	Name            string                  `json:"name,omitempty"`
	Path            string                  `json:"path"`
	MoveType        string                  `json:"move_type"`
	RankRequired    int                     `json:"rank_required"`
	CorePlayerGuID  string                  `json:"core_player_gu_id"`
	SupportGuIDs    []string                `json:"support_player_gu_ids"`
	TotalManaCost   float64                 `json:"total_mana_cost"`
	SuccessRate     float64                 `json:"success_rate"`
	DamageValue     float64                 `json:"damage_value"`
	DefenseValue    float64                 `json:"defense_value"`
	Cooldown        int                     `json:"cooldown"`
	ChannelTime     int                     `json:"channel_time"`
	Effects         map[string]float64      `json:"effects"`
	IsReliable      bool                    `json:"is_reliable"`
	IsInstantCast   bool                    `json:"is_instant_cast"`
	Warnings        []string                `json:"warnings"`
	SlotResults     []KillerMoveSlotPreview `json:"slot_results"`
	Slots           []models.KillerMoveSlot `json:"slots,omitempty"`
	CompoundEffects []CompoundEffectInfo    `json:"compound_effects,omitempty"`
}

type KillerMoveSlotPreview struct {
	PlayerGuID         string   `json:"player_gu_id"`
	GuID               string   `json:"gu_id"`
	GuName             string   `json:"gu_name"`
	SlotType           string   `json:"slot_type"`
	SlotIndex          int      `json:"slot_index"`
	CompatibilityScore int      `json:"compatibility_score"`
	Applied            bool     `json:"applied"`
	Warnings           []string `json:"warnings"`
}

type ownedGu struct {
	Instance models.GuInstance
	Spec     models.GuDictionaryItem
}

func CalculateKillerMove(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CalculateKillerMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	preview, err := CalculateKillerMoveStats(req, playerID.String())
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "database") || strings.Contains(err.Error(), "load player") {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preview)
}

func CreateKillerMove(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateKillerMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên sát chiêu phải từ 1 đến 80 ký tự"})
		return
	}

	preview, err := CalculateKillerMoveStats(CalculateKillerMoveRequest{CorePlayerGuID: req.CorePlayerGuID, SupportPlayerGuIDs: req.SupportPlayerGuIDs}, playerID.String())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.MoveType != "" {
		preview.MoveType = req.MoveType
	}
	effectsJSON, _ := json.Marshal(preview.Effects)
	moveID := uuid.New().String()

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO killer_moves (id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, success_rate, damage_value, defense_value, cooldown, channel_time, effects, is_mastered, is_reliable, is_instant_cast)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,false,$15,$16)
	`, moveID, playerID, name, preview.Path, preview.MoveType, preview.RankRequired, preview.CorePlayerGuID, preview.TotalManaCost, preview.SuccessRate, preview.DamageValue, preview.DefenseValue, preview.Cooldown, preview.ChannelTime, effectsJSON, preview.IsReliable, preview.IsInstantCast)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu sát chiêu"})
		return
	}

	if err := insertKillerMoveSlotsTx(tx, moveID, preview.CorePlayerGuID, preview.SupportGuIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu slot sát chiêu"})
		return
	}
	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 5, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật Nhân Khí"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất lưu sát chiêu"})
		return
	}

	move, err := loadKillerMove(playerID, moveID)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"id": moveID})
		return
	}
	c.JSON(http.StatusCreated, move)
}

func ListKillerMoves(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	moves, err := loadKillerMoves(playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách sát chiêu"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"killer_moves": moves})
}

func GetKillerMove(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	move, err := loadKillerMove(playerID, c.Param("id"))
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sát chiêu"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải sát chiêu"})
		return
	}
	c.JSON(http.StatusOK, move)
}

func DeleteKillerMove(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	res, err := db.DB.Exec(`DELETE FROM killer_moves WHERE id = $1 AND player_id = $2`, c.Param("id"), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa sát chiêu"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sát chiêu"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa sát chiêu"})
}

func CastKillerMove(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req CastKillerMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong trạng thái chiến đấu."})
		return
	}

	move, slots, err := loadKillerMoveTx(tx, playerID, req.KillerMoveID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sát chiêu"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải sát chiêu"})
		return
	}

	if player.ActiveCombat.SkillCooldowns == nil {
		player.ActiveCombat.SkillCooldowns = make(map[string]int)
	}
	if cd, exists := player.ActiveCombat.SkillCooldowns[req.KillerMoveID]; exists && cd > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Sát chiêu này đang trong thời gian hồi (%d lượt).", cd)})
		return
	}

	supportIDs := make([]string, 0)
	for _, slot := range slots {
		if slot.SlotType == models.KillerMoveSlotSupport {
			supportIDs = append(supportIDs, slot.PlayerGuID)
		}
	}
	preview, err := calculateKillerMoveStatsForPlayer(CalculateKillerMoveRequest{CorePlayerGuID: move.CorePlayerGuID, SupportPlayerGuIDs: supportIDs}, &player, tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if move.IsMastered {
		preview.SuccessRate += killerMoveMasteryBonus
		if preview.SuccessRate > killerMoveMaxSuccessRate {
			preview.SuccessRate = killerMoveMaxSuccessRate
		}
	}

	// V2: Channeling check
	if preview.ChannelTime > 0 {
		// Check if player is stunned/frozen/silenced (handled by combat state)
		hasChannelBlock := false
		if player.ActiveCombat != nil {
			for _, ally := range player.ActiveCombat.Allies {
				if ally.ID == player.ID.String() {
					for _, se := range ally.StatusEffects {
						if se.Type == "stun" || se.Type == "freeze" || se.Type == "silence" {
							hasChannelBlock = true
							break
						}
					}
					break
				}
			}
		}
		if hasChannelBlock {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tụ lực sát chiêu khi đang bị khống chế!"})
			return
		}
	}

	allGuIDs := append([]string{move.CorePlayerGuID}, supportIDs...)
	for _, gID := range allGuIDs {
		for _, gu := range player.GuInventory {
			if gu.InstanceID == gID {
				if gu.Hunger <= 0 {
					spec, _ := models.GetGu(gu.GuID, gu.Rank)
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cổ trùng %s trong sát chiêu đã kiệt sức (độ no = 0)!", spec.Name)})
					return
				}
			}
		}
	}

	if player.CurrentMana < preview.TotalManaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}

	player.CurrentMana -= preview.TotalManaCost

	beforeGus := append(models.GuInventory(nil), player.GuInventory...)
	for _, gID := range allGuIDs {
		for idx := range player.GuInventory {
			if player.GuInventory[idx].InstanceID == gID {
				player.GuInventory[idx].Hunger -= 1
				if player.GuInventory[idx].Hunger < 0 {
					player.GuInventory[idx].Hunger = 0
				}
				break
			}
		}
	}
	if errSync := db.SyncPlayerGus(tx, player.ID.String(), beforeGus, player.GuInventory); errSync != nil {
		logger.S.Warnf("Failed to sync player Gu hunger in CastKillerMove: %v", errSync)
	}
	logs := []string{fmt.Sprintf("Thi triển sát chiêu %s: -%.1f %s.", move.Name, preview.TotalManaCost, energyName(&player))}
	eventType := "KILLER_MOVE_CASTED"
	succeeded := rand.Float64()*100 <= preview.SuccessRate
	backlashDamage := 0.0

	if !succeeded {
		eventType = "KILLER_MOVE_FAILED"
		supportCount := len(supportIDs)
		backlashDamage = preview.TotalManaCost * killerMoveBacklashManaFactor * float64(supportCount)
		if backlashDamage < 1.0 {
			backlashDamage = preview.TotalManaCost * 0.5
		}
		player.HP = math.Max(0, player.HP-backlashDamage)
		// V2: Aperture damage on backlash
		player.ApertureDamage += backlashDamage * 0.1
		logs = append(logs, fmt.Sprintf("Sát chiêu thất bại (%.1f%%). Phản phệ gây %.1f sát thương, tổn thương Không Khiếu.", preview.SuccessRate, backlashDamage))
	} else {
		if !move.IsMastered {
			move.UsageCount++
			if move.UsageCount >= 50 {
				move.IsMastered = true
				logs = append(logs, "Sát chiêu đã đạt cảnh giới Đại Thành (Mastered)! Tăng độ ổn định cho các lần thi triển sau.")
			}
			_, _ = tx.Exec(`UPDATE killer_moves SET usage_count = $1, is_mastered = $2 WHERE id = $3`, move.UsageCount, move.IsMastered, move.ID)
		}

		logs = append(logs, fmt.Sprintf("Sát chiêu ổn định (%.1f%%), gây %.1f sát thương và tạo %.1f phòng ngự.", preview.SuccessRate, preview.DamageValue, preview.DefenseValue))

		// Log compound effects
		if len(preview.CompoundEffects) > 0 {
			for _, ce := range preview.CompoundEffects {
				logs = append(logs, fmt.Sprintf("⟐ Hiệu ứng dung hợp [%s]: %s", ce.Name, ce.Description))
			}
		}

		applyKillerMoveToCombat(player.ActiveCombat, player.ID.String(), req.TargetID, preview.DamageValue, preview.DefenseValue, preview.Effects, preview.CompoundEffects, &logs)

		// Sát chiêu thành công tăng Cảnh Giới Lưu Phái (tính theo rank sát chiêu và cảnh giới hiện tại).
		moveRank := move.RankRequired
		if moveRank < 1 {
			moveRank = 1
		}
		corePath := normalizePath(preview.Path)
		currentLevel := gamefx.PathMasteryLevel(player.PathMasteries[corePath])
		pathGain := gamefx.CalculatePathMasteryGain(moveRank, currentLevel, killerMovePathMasteryGain)
		if gained := applyPathMasteryGain(&player, corePath, pathGain, moveRank); gained > 0 {
			newLevel := gamefx.PathMasteryLevel(player.PathMasteries[corePath])
			logs = append(logs, fmt.Sprintf("Ngộ Đạo tăng tiến! +%d điểm Chân Ý lưu phái %s (Cảnh giới: %s).", gained, strings.Title(corePath), gamefx.PathMasteryLevelName(newLevel)))
		}

		player.ActiveCombat.SkillCooldowns[req.KillerMoveID] = preview.Cooldown
	}

	combatJSON, _ := json.Marshal(player.ActiveCombat)
	_, err = tx.Exec(`UPDATE players SET current_mana = $1, hp = $2, active_combat = $3, path_masteries = $4 WHERE id = $5`, player.CurrentMana, player.HP, string(combatJSON), player.PathMasteries, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái chiến đấu"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất thi triển sát chiêu"})
		return
	}

	respondActionGuOnly(c, http.StatusOK, player, gin.H{
		"event":            eventType,
		"success":          succeeded,
		"killer_move":      move,
		"preview":          preview,
		"backlash_damage":  backlashDamage,
		"logs":             logs,
		"compound_effects": preview.CompoundEffects,
		"active_combat":    player.ActiveCombat,
		"current_mana":     player.CurrentMana,
		"hp":               player.HP,
	})
}

func CalculateKillerMoveStats(req CalculateKillerMoveRequest, playerID string) (*KillerMovePreview, error) {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return nil, fmt.Errorf("player_id không hợp lệ")
	}
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("database transaction failure")
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load player")
	}
	preview, err := calculateKillerMoveStatsForPlayer(req, &player, tx)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return preview, nil
}

func calculateKillerMoveStatsForPlayer(req CalculateKillerMoveRequest, player *models.Player, tx *sql.Tx) (*KillerMovePreview, error) {
	core, supports, err := resolveKillerMoveGu(req, player, tx)
	if err != nil {
		return nil, err
	}

	coreActive := models.GetFlatGuEffects(core.Spec)
	totalMana := core.Spec.BaseManaCost

	// V2: DamageBase = CoreGu.base_damage_bonus, DefenseBase = CoreGu.base_defense_bonus
	damageBase := effectValue(coreActive, "base_damage_bonus", "damage")
	defenseBase := effectValue(coreActive, "base_defense_bonus", "defense", "add_base_defense")
	damageMultiplier := 1.0
	defenseMultiplier := 1.0
	effects := map[string]float64{}
	mergeEffects(effects, coreActive)
	slotResults := []KillerMoveSlotPreview{{
		PlayerGuID: core.Instance.InstanceID,
		GuID:       core.Instance.GuID,
		GuName:     core.Spec.Name,
		SlotType:   models.KillerMoveSlotCore,
		SlotIndex:  0,
		Applied:    true,
	}}

	var appliedSupportSpecs []*models.GuDictionaryItem
	var appliedSupportEffects []map[string]float64

	pathConflictPenalty := 0.0
	successRateBonus := 0.0
	channelTime := int(effectValue(coreActive, "channel_time", "channeling"))

	hasSovereignFetus := models.HasEquippedGu(player.GuInventory, "GU_SOVEREIGN_FETUS")

	attainment := float64(attainmentLevel(player, core.Spec.Path))

	conflictingMarks := int64(0)
	if player.DaoMarks != nil && !hasSovereignFetus && attainment < 4 {
		for path, amount := range player.DaoMarks {
			if !isPathInGu(core.Spec.Path, path) {
				conflictingMarks += amount
			}
		}
		if attainment >= 3 {
			conflictingMarks = int64(float64(conflictingMarks) * 0.5)
		}
	}

	for idx, support := range supports {
		totalMana += support.Spec.BaseManaCost * killerMoveSupportCostRate
		supportEffects := models.GetFlatGuEffects(support.Spec)

		// V2: Path Conflict Penalty
		conflictPct, isOpposing := calculatePathConflict(core.Spec, support.Spec)
		if hasSovereignFetus || attainment >= 4 {
			conflictPct = 0.0
			isOpposing = false
		} else if attainment >= 3 {
			conflictPct *= 0.5
		}
		pathConflictPenalty += conflictPct
		slotWarnings := []string{}
		if isOpposing {
			slotWarnings = append(slotWarnings, fmt.Sprintf("%s %s đối nghịch với %s! Phạt +%.0f%% thất bại.", support.Spec.Name, support.Spec.Path, core.Spec.Path, conflictPct))
		} else if conflictPct > 0 {
			slotWarnings = append(slotWarnings, fmt.Sprintf("%s %s khác với %s. Phạt +%.0f%% thất bại.", support.Spec.Name, support.Spec.Path, core.Spec.Path, conflictPct))
		}

		// V2: All supports apply (no threshold)
		damageBase += effectValue(supportEffects, "base_damage_bonus", "flat_damage", "damage")
		defenseBase += effectValue(supportEffects, "base_defense_bonus", "flat_defense", "defense")
		damageMultiplier *= 1 + effectValue(supportEffects, "damage_multiplier", "damage_boost")
		defenseMultiplier *= 1 + effectValue(supportEffects, "defense_multiplier", "defense_boost", "shield_multiplier")
		successRateBonus += effectValue(supportEffects, "success_rate_bonus", "stability_bonus")
		successRateBonus -= effectValue(supportEffects, "stability_penalty")
		channelTime += int(effectValue(supportEffects, "channeling_reduction"))
		mergeEffects(effects, supportEffects)
		appliedSupportSpecs = append(appliedSupportSpecs, &support.Spec)
		appliedSupportEffects = append(appliedSupportEffects, supportEffects)

		slotResults = append(slotResults, KillerMoveSlotPreview{
			PlayerGuID: support.Instance.InstanceID,
			GuID:       support.Instance.GuID,
			GuName:     support.Spec.Name,
			SlotType:   models.KillerMoveSlotSupport,
			SlotIndex:  idx + 1,
			Applied:    true,
			Warnings:   slotWarnings,
		})
	}

	// ── COMPOUND EFFECT DETECTION ──
	allCompounds := detectCompounds(coreActive, appliedSupportEffects, effects)
	synergyCompounds, synDmgBonus, synSuccessBonus, synSpecial := detectSynergy(&core.Spec, appliedSupportSpecs)
	allCompounds = append(allCompounds, synergyCompounds...)
	if synDmgBonus > 0 {
		damageMultiplier *= synDmgBonus
	}
	finalDamage, finalDefense, finalEffects := applyCompoundEffects(
		damageBase, defenseBase, effects, allCompounds,
		damageMultiplier, defenseMultiplier,
	)

	// ── V2: DaoBonus ──
	daoMarks := 0.0
	if player.DaoMarks != nil {
		maxDao := 0.0
		for _, path := range core.Spec.Path {
			dMark := float64(player.DaoMarks[normalizePath(path)])
			if dMark > maxDao {
				maxDao = dMark
			}
		}
		daoMarks = maxDao
	}
	daoBonus := 1.0 + daoMarks/100.0
	finalDamage *= daoBonus
	finalDefense *= daoBonus

	// ── V2: SUCCESS RATE ──
	// TODO: dùng cổ trùng để tăng tỉ lệ thành công thay vì Wisdom stat
	if attainment >= 2 {
		successRateBonus += 5.0
	}
	successPenalty := 0.0
	if conflictingMarks > 0 {
		successPenalty = math.Min(30.0, float64(conflictingMarks)/500.0)
	}
	successRate := killerMoveBaseSuccessRate +
		attainment*killerMoveAttainmentBonusFactor +
		successRateBonus -
		successPenalty -
		float64(len(supports))*killerMoveSupportCountPenalty -
		pathConflictPenalty +
		synSuccessBonus
	successRate = clamp(successRate, killerMoveMinSuccessRate, killerMoveMaxSuccessRate)

	if channelTime < 0 {
		channelTime = 0
	}

	// ── V2: MANA COST & COOLDOWN ADJUSTMENTS ──
	if attainment >= 2 {
		totalMana *= 0.85
	}
	if conflictingMarks > 0 {
		costMultiplier := 1.0 + float64(conflictingMarks)/20000.0
		totalMana *= costMultiplier
	}
	refineTier := gamefx.RefineManaTier(player.Rank, player.GuInventory)
	if refineTier > 0 {
		totalMana *= 0.5
	}

	cooldown := 1 + len(supports)/3
	if conflictingMarks > 0 {
		cooldownMultiplier := 1.0 + float64(conflictingMarks)/40000.0
		cooldown = int(math.Ceil(float64(cooldown) * cooldownMultiplier))
	}

	warnings := successRateWarnings(successRate)
	for _, slot := range slotResults {
		warnings = append(warnings, slot.Warnings...)
	}
	if len(synergyCompounds) > 0 {
		warnings = append(warnings, fmt.Sprintf("Kích hoạt Cổ trận liên hoàn: %s — %s", synergyCompounds[0].Name, synSpecial))
	}

	moveType := inferMoveType(finalDamage, finalDefense, finalEffects)
	rankRequired := core.Spec.Rank
	for _, support := range supports {
		if support.Spec.Rank > rankRequired {
			rankRequired = support.Spec.Rank
		}
	}

	return &KillerMovePreview{
		Path: func() string {
			if len(core.Spec.Path) > 0 {
				return core.Spec.Path[0]
			}
			return "Star Path"
		}(),
		MoveType:        moveType,
		RankRequired:    rankRequired,
		CorePlayerGuID:  core.Instance.InstanceID,
		SupportGuIDs:    req.SupportPlayerGuIDs,
		TotalManaCost:   round(totalMana),
		SuccessRate:     round(successRate),
		DamageValue:     round(finalDamage),
		DefenseValue:    round(finalDefense),
		Cooldown:        cooldown,
		ChannelTime:     channelTime,
		Effects:         finalEffects,
		IsReliable:      successRate >= 80,
		IsInstantCast:   channelTime <= 0 && successRate >= 80,
		Warnings:        warnings,
		SlotResults:     slotResults,
		CompoundEffects: allCompounds,
	}, nil
}

func resolveKillerMoveGu(req CalculateKillerMoveRequest, player *models.Player, tx *sql.Tx) (ownedGu, []ownedGu, error) {
	if len(req.SupportPlayerGuIDs) > killerMoveMaxSupport {
		return ownedGu{}, nil, fmt.Errorf("Sát chiêu chỉ hỗ trợ tối đa %d Cổ phụ trợ", killerMoveMaxSupport)
	}
	seen := map[string]bool{req.CorePlayerGuID: true}
	core, err := findUsableOwnedGu(player, req.CorePlayerGuID, tx)
	if err != nil {
		return ownedGu{}, nil, fmt.Errorf("Cổ hạch tâm không hợp lệ: %w", err)
	}
	supports := make([]ownedGu, 0, len(req.SupportPlayerGuIDs))
	for _, id := range req.SupportPlayerGuIDs {
		if id == "" {
			continue
		}
		if seen[id] {
			return ownedGu{}, nil, fmt.Errorf("Một Cổ không thể gắn vào nhiều slot trong cùng sát chiêu: %s", id)
		}
		seen[id] = true
		support, err := findUsableOwnedGu(player, id, tx)
		if err != nil {
			return ownedGu{}, nil, fmt.Errorf("Cổ phụ trợ không hợp lệ (%s): %w", id, err)
		}
		supports = append(supports, support)
	}
	return core, supports, nil
}

func findUsableOwnedGu(player *models.Player, instanceID string, tx *sql.Tx) (ownedGu, error) {
	inst, _, found := models.FindGuByInstanceID(player.GuInventory, instanceID)
	if !found {
		return ownedGu{}, fmt.Errorf("người chơi không sở hữu Cổ trùng này")
	}
	if inst.Status != "ACTIVE" {
		return ownedGu{}, fmt.Errorf("Cổ trùng không ở trạng thái ACTIVE")
	}
	if inst.Hunger <= 0 {
		return ownedGu{}, fmt.Errorf("Cổ trùng đã cạn đói, không thể kích hoạt")
	}
	locked, err := isGuLockedForExchange(tx, player.ID, instanceID)
	if err != nil {
		return ownedGu{}, err
	}
	if locked {
		return ownedGu{}, fmt.Errorf("Cổ trùng đang bị khóa trong giao dịch hoặc đấu giá")
	}
	spec, ok := models.GetGu(inst.GuID, inst.Rank)
	if !ok {
		return ownedGu{}, fmt.Errorf("không tìm thấy định nghĩa Cổ trùng %s", inst.GuID)
	}
	return ownedGu{Instance: *inst, Spec: spec}, nil
}

func isGuLockedForExchange(tx *sql.Tx, playerID uuid.UUID, instanceID string) (bool, error) {
	var auctionLocked bool
	err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM auction WHERE seller_id = $1 AND status = 'ACTIVE' AND item_data::text LIKE '%' || $2 || '%')`, playerID, instanceID).Scan(&auctionLocked)
	if err != nil {
		return false, err
	}
	if auctionLocked {
		return true, nil
	}
	var tradeLocked bool
	err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM trades WHERE (sender_id = $1 OR receiver_id = $1) AND status IN ('PENDING','ACCEPTED') AND (sender_offer::text LIKE '%' || $2 || '%' OR receiver_offer::text LIKE '%' || $2 || '%'))`, playerID, instanceID).Scan(&tradeLocked)
	return tradeLocked, err
}
