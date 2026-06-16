package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const (
	killerMoveSupportCostRate       = 0.8
	killerMoveBaseStability         = 95.0
	killerMoveSupportPenalty        = 8.0
	killerMovePathConflictPenalty   = 20.0
	killerMoveAttainmentBonus       = 5.0
	killerMoveMinStability          = 5.0
	killerMoveMaxStability          = 100.0
	killerMoveBacklashRate          = 1.5
	killerMoveMultiTaskPenalty      = 30.0
	killerMoveGrandmasterAttainment = 2
)

type CalculateKillerMoveRequest struct {
	CorePlayerGuID        string   `json:"core_player_gu_id" binding:"required"`
	SupportPlayerGuIDs    []string `json:"support_player_gu_ids"`
	ApplyMultiTaskPenalty bool     `json:"apply_multi_task_penalty"`
}

type CreateKillerMoveRequest struct {
	Name               string   `json:"name" binding:"required"`
	MoveType           string   `json:"move_type"`
	CorePlayerGuID     string   `json:"core_player_gu_id" binding:"required"`
	SupportPlayerGuIDs []string `json:"support_player_gu_ids"`
}

type CastKillerMoveRequest struct {
	KillerMoveID        string `json:"killer_move_id" binding:"required"`
	TargetID            string `json:"target_id"`
	UsedExtraGuThisTurn bool   `json:"used_extra_gu_this_turn"`
}

type KillerMovePreview struct {
	Name            string                  `json:"name,omitempty"`
	Path            string                  `json:"path"`
	MoveType        string                  `json:"move_type"`
	RankRequired    int                     `json:"rank_required"`
	CorePlayerGuID  string                  `json:"core_player_gu_id"`
	SupportGuIDs    []string                `json:"support_player_gu_ids"`
	TotalManaCost   float64                 `json:"total_mana_cost"`
	Stability       float64                 `json:"stability"`
	DamageValue     float64                 `json:"damage_value"`
	DefenseValue    float64                 `json:"defense_value"`
	Cooldown        int                     `json:"cooldown"`
	CastTime        int                     `json:"cast_time"`
	Effects         map[string]float64      `json:"effects"`
	IsStable        bool                    `json:"is_stable"`
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
		INSERT INTO killer_moves (id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, stability, damage_value, defense_value, cooldown, cast_time, effects, is_mastered, is_stable, is_instant_cast)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,false,$15,$16)
	`, moveID, playerID, name, preview.Path, preview.MoveType, preview.RankRequired, preview.CorePlayerGuID, preview.TotalManaCost, preview.Stability, preview.DamageValue, preview.DefenseValue, preview.Cooldown, preview.CastTime, effectsJSON, preview.IsStable, preview.IsInstantCast)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu sát chiêu"})
		return
	}

	if err := insertKillerMoveSlotsTx(tx, moveID, preview.CorePlayerGuID, preview.SupportGuIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu slot sát chiêu"})
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
	preview, err := calculateKillerMoveStatsForPlayer(CalculateKillerMoveRequest{CorePlayerGuID: move.CorePlayerGuID, SupportPlayerGuIDs: supportIDs, ApplyMultiTaskPenalty: req.UsedExtraGuThisTurn}, &player, tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if move.IsMastered {
		preview.Stability += 15.0
		if preview.Stability > 100.0 {
			preview.Stability = 100.0
		}
	}
	
	if player.CurrentMana < preview.TotalManaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}

	player.CurrentMana -= preview.TotalManaCost
	logs := []string{fmt.Sprintf("Thi triển sát chiêu %s: -%.1f %s.", move.Name, preview.TotalManaCost, energyName(&player))}
	eventType := "KILLER_MOVE_CASTED"
	succeeded := rand.Float64()*100 <= preview.Stability
	backlashDamage := 0.0

	if !succeeded {
		eventType = "KILLER_MOVE_FAILED"
		backlashDamage = preview.TotalManaCost * killerMoveBacklashRate
		player.HP = math.Max(0, player.HP-backlashDamage)
		logs = append(logs, fmt.Sprintf("Sát chiêu mất ổn định (%.1f%%). Phản phệ gây %.1f sát thương.", preview.Stability, backlashDamage))
	} else {
		if !move.IsMastered {
			move.UsageCount++
			if move.UsageCount >= 50 {
				move.IsMastered = true
				logs = append(logs, "Sát chiêu đã đạt cảnh giới Đại Thành (Mastered)! Tăng độ ổn định cho các lần thi triển sau.")
			}
			_, _ = tx.Exec(`UPDATE killer_moves SET usage_count = $1, is_mastered = $2 WHERE id = $3`, move.UsageCount, move.IsMastered, move.ID)
		}

		logs = append(logs, fmt.Sprintf("Sát chiêu ổn định (%.1f%%), gây %.1f sát thương và tạo %.1f phòng ngự.", preview.Stability, preview.DamageValue, preview.DefenseValue))

		// Log compound effects
		if len(preview.CompoundEffects) > 0 {
			for _, ce := range preview.CompoundEffects {
				logs = append(logs, fmt.Sprintf("⟐ Hiệu ứng dung hợp [%s]: %s", ce.Name, ce.Description))
			}
		}

		applyKillerMoveToCombat(player.ActiveCombat, player.ID.String(), req.TargetID, preview.DamageValue, preview.DefenseValue, preview.Effects, preview.CompoundEffects, &logs)
		player.ActiveCombat.SkillCooldowns[req.KillerMoveID] = preview.Cooldown
	}

	combatJSON, _ := json.Marshal(player.ActiveCombat)
	_, err = tx.Exec(`UPDATE players SET current_mana = $1, hp = $2, active_combat = $3 WHERE id = $4`, player.CurrentMana, player.HP, combatJSON, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái chiến đấu"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất thi triển sát chiêu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":           player,
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

	coreActive := models.GetGuEffectSection(core.Spec, "active")
	totalMana := core.Spec.BaseManaCost
	damageBase := effectValue(coreActive, "damage", "base_damage", "add_base_attack")
	defenseBase := effectValue(coreActive, "defense", "base_defense", "add_base_defense")
	damageMultiplier := 1.0
	defenseMultiplier := 1.0
	effects := map[string]float64{}
	mergeEffects(effects, coreActive)
	compatibilityPenalty := 0.0
	slotResults := []KillerMoveSlotPreview{{PlayerGuID: core.Instance.InstanceID, GuID: core.Instance.GuID, GuName: core.Spec.Name, SlotType: models.KillerMoveSlotCore, SlotIndex: 0, CompatibilityScore: 100, Applied: true}}

	// Collect applied support specs and effects for compounding
	var appliedSupportSpecs []*models.GuDictionaryItem
	var appliedSupportEffects []map[string]float64

	moveTypeSeed := inferMoveType(damageBase, defenseBase, effects)
	for idx, support := range supports {
		totalMana += support.Spec.BaseManaCost * killerMoveSupportCostRate
		supportEffects := models.GetGuEffectSection(support.Spec, "support")
		compatibilityScore, supportWarnings := calculateSupportCompatibility(core.Spec, support.Spec, moveTypeSeed)
		applied := compatibilityScore >= 30
		if applied {
			damageBase += effectValue(supportEffects, "flat_damage", "damage", "add_base_attack")
			defenseBase += effectValue(supportEffects, "flat_defense", "defense", "add_base_defense")
			damageMultiplier *= 1 + effectValue(supportEffects, "damage_multiplier", "damage_boost")
			defenseMultiplier *= 1 + effectValue(supportEffects, "defense_multiplier", "defense_boost")
			mergeEffects(effects, supportEffects)
			appliedSupportSpecs = append(appliedSupportSpecs, &support.Spec)
			appliedSupportEffects = append(appliedSupportEffects, supportEffects)
		} else {
			compatibilityPenalty += float64(30 - compatibilityScore)
		}
		slotResults = append(slotResults, KillerMoveSlotPreview{PlayerGuID: support.Instance.InstanceID, GuID: support.Instance.GuID, GuName: support.Spec.Name, SlotType: models.KillerMoveSlotSupport, SlotIndex: idx + 1, CompatibilityScore: compatibilityScore, Applied: applied, Warnings: supportWarnings})
	}

	// ── COMPOUND EFFECT DETECTION ──
	allCompounds := detectCompounds(coreActive, appliedSupportEffects, effects)
	synergyCompounds, synDmgBonus, synStabilityBonus, synSpecial := detectSynergy(&core.Spec, appliedSupportSpecs)
	allCompounds = append(allCompounds, synergyCompounds...)

	// Apply compound transforms to damage/defense/effects
	// Synergy bonus multiplies final damage
	if synDmgBonus > 0 {
		damageMultiplier *= synDmgBonus
	}
	finalDamage, finalDefense, finalEffects := applyCompoundEffects(
		damageBase, defenseBase, effects, allCompounds,
		damageMultiplier, defenseMultiplier,
	)

	// ── STABILITY ──
	stability := killerMoveBaseStability - float64(len(supports))*killerMoveSupportPenalty - compatibilityPenalty + float64(attainmentLevel(player, core.Spec.Path))*killerMoveAttainmentBonus + synStabilityBonus
	if req.ApplyMultiTaskPenalty {
		stability -= killerMoveMultiTaskPenalty
	}
	stability = clamp(stability, killerMoveMinStability, killerMoveMaxStability)
	warnings := stabilityWarnings(stability)
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
		Path:            core.Spec.Path,
		MoveType:        moveType,
		RankRequired:    rankRequired,
		CorePlayerGuID:  core.Instance.InstanceID,
		SupportGuIDs:    req.SupportPlayerGuIDs,
		TotalManaCost:   round(totalMana),
		Stability:       round(stability),
		DamageValue:     round(finalDamage),
		DefenseValue:    round(finalDefense),
		Cooldown:        1 + len(supports)/3,
		CastTime:        1,
		Effects:         finalEffects,
		IsStable:        stability >= 80,
		IsInstantCast:   len(supports) <= 2 && stability >= 80,
		Warnings:        warnings,
		SlotResults:     slotResults,
		CompoundEffects: allCompounds,
	}, nil
}

func resolveKillerMoveGu(req CalculateKillerMoveRequest, player *models.Player, tx *sql.Tx) (ownedGu, []ownedGu, error) {
	if len(req.SupportPlayerGuIDs) > 8 {
		return ownedGu{}, nil, fmt.Errorf("Sát chiêu chỉ hỗ trợ tối đa 8 Cổ phụ trợ")
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
	for _, inst := range player.GuInventory {
		if inst.InstanceID != instanceID {
			continue
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
		spec, ok := models.GuRegistry[inst.GuID]
		if !ok {
			return ownedGu{}, fmt.Errorf("không tìm thấy định nghĩa Cổ trùng %s", inst.GuID)
		}
		if player.Rank < spec.Rank {
			return ownedGu{}, fmt.Errorf("Rank người chơi không đủ để điều khiển %s", spec.Name)
		}
		return ownedGu{Instance: inst, Spec: spec}, nil
	}
	return ownedGu{}, fmt.Errorf("người chơi không sở hữu Cổ trùng này")
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

func loadKillerMoves(playerID uuid.UUID) ([]models.KillerMove, error) {
	rows, err := db.DB.Query(`SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, stability, damage_value, defense_value, cooldown, cast_time, effects, is_mastered, is_stable, is_instant_cast, created_at, updated_at FROM killer_moves WHERE player_id = $1 ORDER BY updated_at DESC`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	moves := []models.KillerMove{}
	for rows.Next() {
		m, err := scanKillerMove(rows)
		if err != nil {
			return nil, err
		}
		moves = append(moves, m)
	}
	for i := range moves {
		moves[i].Slots, _ = loadKillerMoveSlots(moves[i].ID)
	}
	return moves, rows.Err()
}

func loadKillerMove(playerID uuid.UUID, id string) (models.KillerMove, error) {
	row := db.DB.QueryRow(`SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, stability, damage_value, defense_value, cooldown, cast_time, effects, is_mastered, is_stable, is_instant_cast, created_at, updated_at FROM killer_moves WHERE id = $1 AND player_id = $2`, id, playerID)
	m, err := scanKillerMove(row)
	if err != nil {
		return m, err
	}
	m.Slots, _ = loadKillerMoveSlots(m.ID)
	return m, nil
}

func loadKillerMoveTx(tx *sql.Tx, playerID uuid.UUID, id string) (models.KillerMove, []models.KillerMoveSlot, error) {
	row := tx.QueryRow(`SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, stability, damage_value, defense_value, cooldown, cast_time, effects, is_mastered, is_stable, is_instant_cast, created_at, updated_at FROM killer_moves WHERE id = $1 AND player_id = $2 FOR UPDATE`, id, playerID)
	m, err := scanKillerMove(row)
	if err != nil {
		return m, nil, err
	}
	rows, err := tx.Query(`SELECT id, killer_move_id, player_gu_id, slot_type, slot_index, created_at FROM killer_move_slots WHERE killer_move_id = $1 ORDER BY slot_index ASC`, m.ID)
	if err != nil {
		return m, nil, err
	}
	defer rows.Close()
	slots, err := scanKillerMoveSlots(rows)
	return m, slots, err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanKillerMove(row rowScanner) (models.KillerMove, error) {
	var m models.KillerMove
	err := row.Scan(&m.ID, &m.PlayerID, &m.Name, &m.Path, &m.MoveType, &m.RankRequired, &m.CorePlayerGuID, &m.TotalManaCost, &m.Stability, &m.DamageValue, &m.DefenseValue, &m.Cooldown, &m.CastTime, &m.Effects, &m.IsMastered, &m.IsStable, &m.IsInstantCast, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func loadKillerMoveSlots(moveID string) ([]models.KillerMoveSlot, error) {
	rows, err := db.DB.Query(`SELECT id, killer_move_id, player_gu_id, slot_type, slot_index, created_at FROM killer_move_slots WHERE killer_move_id = $1 ORDER BY slot_index ASC`, moveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKillerMoveSlots(rows)
}

func scanKillerMoveSlots(rows *sql.Rows) ([]models.KillerMoveSlot, error) {
	slots := []models.KillerMoveSlot{}
	for rows.Next() {
		var s models.KillerMoveSlot
		if err := rows.Scan(&s.ID, &s.KillerMoveID, &s.PlayerGuID, &s.SlotType, &s.SlotIndex, &s.CreatedAt); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, rows.Err()
}

func insertKillerMoveSlotsTx(tx *sql.Tx, moveID, coreID string, supportIDs []string) error {
	_, err := tx.Exec(`INSERT INTO killer_move_slots (id, killer_move_id, player_gu_id, slot_type, slot_index) VALUES ($1,$2,$3,$4,0)`, uuid.New().String(), moveID, coreID, models.KillerMoveSlotCore)
	if err != nil {
		return err
	}
	for idx, id := range supportIDs {
		_, err = tx.Exec(`INSERT INTO killer_move_slots (id, killer_move_id, player_gu_id, slot_type, slot_index) VALUES ($1,$2,$3,$4,$5)`, uuid.New().String(), moveID, id, models.KillerMoveSlotSupport, idx+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func applyKillerMoveToCombat(combat *models.CombatState, casterID, targetID string, damage, defense float64, effects map[string]float64, compounds []CompoundEffectInfo, logs *[]string) {
	// Apply shield to caster
	for i := range combat.Allies {
		if combat.Allies[i].ID == casterID {
			combat.Allies[i].VirtualShield += defense
			break
		}
	}

	// Track compound-derived special effects
	hasIgnoreArmor := false
	hasLifesteal := false
	lifestealPct := 0.0
	hasThorns := false
	thornsDmg := 0.0
	hasMultiHit := false
	multiHitFactor := 1.0
	hasPhantomAttack := false
	phantomDmg := 0.0
	hasRegen := false
	regenVal := 0.0
	hasHemorrhage := false
	hemorrhageDmg := 0.0
	hasToxicWound := false
	toxicDmg := 0.0

	for _, c := range compounds {
		switch c.Type {
		case "hemorrhage":
			hasHemorrhage = true
			hemorrhageDmg = c.Value * 0.3
		case "regenerative_shield":
			hasRegen = true
			regenVal = c.Value * 0.15
		case "toxic_wound":
			hasToxicWound = true
			toxicDmg = c.Value * 0.3
		case "bloodthirsty_strength":
			hasLifesteal = true
			if effects["lifesteal"] > 0 {
				lifestealPct = effects["lifesteal"]
			} else {
				lifestealPct = 15.0
			}
		case "chain_damage":
			hasMultiHit = true
			if v, ok := effects["multi_hit"]; ok && v > 0 {
				multiHitFactor = 1.0 + v
			}
		case "counter_thorns":
			hasThorns = true
			if v, ok := effects["thorns_damage"]; ok && v > 0 {
				thornsDmg = v
			}
		case "phantom_onslaught":
			hasPhantomAttack = true
			if v, ok := effects["phantom_attack"]; ok && v > 0 {
				phantomDmg = v
			}
		case "concussive_blow":
			if v, ok := effects["stun"]; ok && v > 0 {
				combat.Enemies = applyStunToTargets(combat.Enemies, targetID, v, logs)
			}
		}
	}

	// Apply to enemies
	for i := range combat.Enemies {
		if combat.Enemies[i].HP <= 0 {
			continue
		}
		if targetID == "" || combat.Enemies[i].ID == targetID {
			finalDamage := damage

			// Armor penetration from compounds
			if hasIgnoreArmor {
				finalDamage -= combat.Enemies[i].BaseDefense * 0.2
			} else {
				finalDamage -= combat.Enemies[i].BaseDefense * 0.5
			}
			if finalDamage < 1.0 {
				finalDamage = 1.0
			}

			// Multi-hit (chain damage)
			if hasMultiHit && multiHitFactor > 1.0 {
				hitCount := int(multiHitFactor)
				if hitCount < 1 {
					hitCount = 1
				}
				totalHitDmg := 0.0
				for h := 0; h < hitCount; h++ {
					hitDmg := finalDamage * 0.6
					if h > 0 {
						hitDmg *= 0.7 // diminishing on chain hits
					}
					combat.Enemies[i].HP = math.Max(0, combat.Enemies[i].HP-hitDmg)
					totalHitDmg += hitDmg
				}
				*logs = append(*logs, fmt.Sprintf("%s trúng %d đòn liên hoàn, tổng %.1f sát thương!", combat.Enemies[i].Name, hitCount, totalHitDmg))
				finalDamage = totalHitDmg
			} else {
				combat.Enemies[i].HP = math.Max(0, combat.Enemies[i].HP-finalDamage)
				*logs = append(*logs, fmt.Sprintf("%s chịu %.1f sát thương từ sát chiêu.", combat.Enemies[i].Name, finalDamage))
			}

			// Phantom bonus attack
			if hasPhantomAttack && phantomDmg > 0 {
				combat.Enemies[i].HP = math.Max(0, combat.Enemies[i].HP-phantomDmg)
				*logs = append(*logs, fmt.Sprintf("Hư ảnh tấn công thêm %.1f sát thương!", phantomDmg))
				finalDamage += phantomDmg
			}

			// Lifesteal
			if hasLifesteal && lifestealPct > 0 {
				healAmount := finalDamage * (lifestealPct / 100.0)
				for a := range combat.Allies {
					if combat.Allies[a].ID == casterID {
						combat.Allies[a].HP = math.Min(combat.Allies[a].MaxHP, combat.Allies[a].HP+healAmount)
						*logs = append(*logs, fmt.Sprintf("%s hút %.1f máu từ sát chiêu!", combat.Allies[a].Name, healAmount))
						break
					}
				}
			}

			// Thorns damage (reflect)
			if hasThorns && thornsDmg > 0 {
				*logs = append(*logs, fmt.Sprintf("Khiên phản thương gây %.1f sát thương lên %s!", thornsDmg, combat.Enemies[i].Name))
			}

			// Apply debuffs from effects
			if effects != nil {
				if hasHemorrhage && hemorrhageDmg > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "bleed", Value: hemorrhageDmg, Turns: 3})
					*logs = append(*logs, fmt.Sprintf("%s bị xuất huyết trọng thương!", combat.Enemies[i].Name))
				}
				if hasToxicWound && toxicDmg > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "poison", Value: toxicDmg, Turns: 3})
					*logs = append(*logs, fmt.Sprintf("%s bị trúng độc từ vết thương!", combat.Enemies[i].Name))
				}
				if bleed, ok := effects["bleed_effect"]; ok && bleed > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "bleed", Value: bleed, Turns: 3})
					*logs = append(*logs, fmt.Sprintf("%s bị chảy máu!", combat.Enemies[i].Name))
				}
				if poison, ok := effects["poison"]; ok && poison > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "poison", Value: poison, Turns: 3})
					*logs = append(*logs, fmt.Sprintf("%s bị trúng độc!", combat.Enemies[i].Name))
				}
				if stun, ok := effects["stun"]; ok && stun > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "stun", Value: stun, Turns: 1})
					*logs = append(*logs, fmt.Sprintf("%s bị choáng!", combat.Enemies[i].Name))
				}
			}

			// Regen to caster (from regenerative shield)
			if hasRegen && regenVal > 0 {
				for a := range combat.Allies {
					if combat.Allies[a].ID == casterID {
						combat.Allies[a].HP = math.Min(combat.Allies[a].MaxHP, combat.Allies[a].HP+regenVal)
						break
					}
				}
				*logs = append(*logs, fmt.Sprintf("Khiên sinh mệnh hồi phục %.1f HP.", regenVal))
			}

			return
		}
	}
}

func effectValue(effects map[string]float64, keys ...string) float64 {
	for _, key := range keys {
		if v, ok := effects[key]; ok {
			return v
		}
	}
	return 0
}
func mergeEffects(dst map[string]float64, src map[string]float64) {
	for k, v := range src {
		dst[k] += v
	}
}
func calculateSupportCompatibility(core models.GuDictionaryItem, support models.GuDictionaryItem, moveType string) (int, []string) {
	score := 50
	sharedTags := sharedStringCount(append(core.CombatTags, core.SupportTags...), append(support.CombatTags, support.SupportTags...))
	if normalizePath(support.Path) == normalizePath(core.Path) {
		score += 30
	} else {
		score -= 20
	}
	if sharedTags > 0 {
		score += sharedTags * 10
	} else {
		score -= 15
	}
	if supportRoleMatchesMoveType(support, moveType) {
		score += 20
	} else {
		score -= 25
	}
	warnings := []string{}
	if score < 30 {
		warnings = append(warnings, fmt.Sprintf("%s không phù hợp với sát chiêu %s %s. Hiệu ứng phụ trợ không được cộng vào sát chiêu này.", support.Name, core.Path, moveType))
	}
	return score, warnings
}

func sharedStringCount(left []string, right []string) int {
	seen := map[string]bool{}
	for _, value := range left {
		key := strings.ToLower(strings.TrimSpace(value))
		if key != "" {
			seen[key] = true
		}
	}
	count := 0
	for _, value := range right {
		key := strings.ToLower(strings.TrimSpace(value))
		if seen[key] {
			count++
			seen[key] = false
		}
	}
	return count
}

func supportRoleMatchesMoveType(support models.GuDictionaryItem, moveType string) bool {
	move := strings.ToUpper(strings.TrimSpace(moveType))
	role := strings.ToUpper(strings.TrimSpace(support.PrimaryRole))
	if role == "" || move == "UTILITY" {
		return true
	}
	if strings.Contains(role, move) || strings.Contains(role, "SUPPORT") || strings.Contains(role, "BOOST") {
		return true
	}
	for _, combatRole := range support.CombatRoles {
		normalized := strings.ToUpper(strings.TrimSpace(combatRole))
		if strings.Contains(normalized, move) || strings.Contains(normalized, "SUPPORT") || strings.Contains(normalized, "BOOST") {
			return true
		}
	}
	return false
}

func normalizePath(path string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.ToLower(path), " path"))
}
func attainmentLevel(player *models.Player, path string) int {
	if player.PathMasteries == nil {
		return 0
	}
	level := player.PathMasteries[normalizePath(path)]
	switch strings.ToLower(level) {
	case "master", "đại sư", "dai su":
		return 1
	case "grandmaster", "tông sư", "tong su":
		return 2
	case "great grandmaster", "đại tông sư", "dai tong su":
		return 3
	case "supreme grandmaster", "vô thượng đại tông sư", "vo thuong dai tong su":
		return 4
	default:
		return 0
	}
}
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func round(v float64) float64 { return math.Round(v*100) / 100 }
func stabilityWarnings(stability float64) []string {
	if stability >= 80 {
		return []string{"Ổn định"}
	}
	if stability >= 50 {
		return []string{"Có rủi ro: sát chiêu có thể phản phệ"}
	}
	return []string{"Nguy hiểm: cấu trúc sát chiêu rất bất ổn"}
}
func applyStunToTargets(enemies []models.Combatant, targetID string, stunValue float64, logs *[]string) []models.Combatant {
	stunTurns := 1
	if stunValue > 50 {
		stunTurns = 2
	}
	for i := range enemies {
		if enemies[i].HP <= 0 {
			continue
		}
		if targetID == "" || enemies[i].ID == targetID {
			enemies[i].StatusEffects = append(enemies[i].StatusEffects, models.StatusEffect{Type: "stun", Value: stunValue, Turns: stunTurns})
			*logs = append(*logs, fmt.Sprintf("%s bị choáng %d lượt!", enemies[i].Name, stunTurns))
			return enemies
		}
	}
	return enemies
}

func inferMoveType(damage, defense float64, effects map[string]float64) string {
	if damage >= defense && damage > 0 {
		return "ATTACK"
	}
	if defense > 0 {
		return "DEFENSE"
	}
	keys := make([]string, 0, len(effects))
	for k := range effects {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return strings.ToUpper(keys[0])
	}
	return "UTILITY"
}
