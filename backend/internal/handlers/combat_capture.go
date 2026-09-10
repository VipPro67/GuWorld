package handlers

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const captureHPThreshold = 0.30 // bắt khi HP ≤ 30% max

type CaptureRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
}

func CaptureMonster(c *gin.Context) {
	var req CaptureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: player mismatch"})
		return
	}

	tx, cancel := beginCombatTx(c)
	if tx == nil {
		return
	}
	defer cancel()
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to load player")
		return
	}

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": errLeader.Error()})
		return
	}

	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong trạng thái chiến đấu."})
		return
	}

	if player.ActiveCombat.EnemyTemplateID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể bắt quái vật này (không có template ID)."})
		return
	}

	monTemplate, exists := models.MonsterRegistry[player.ActiveCombat.EnemyTemplateID]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Monster template không tồn tại."})
		return
	}

	if !monTemplate.IsCapturable {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s không thể bị bắt.", monTemplate.Name)})
		return
	}

	// Capture is allowed at any HP — the hpFactor in the formula makes it harder at full HP
	// (at full HP, hpFactor ≈ 0, so only capture bonus matters)

	// Thu thập CaptureBonus từ cổ trùng equipped + active
	totalCaptureBonus := 0.0
	bestCaptureRank := 0
	for _, gu := range player.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			continue
		}
		if spec.Effects.Active != nil && spec.Effects.Active.CaptureBonus > 0 {
			totalCaptureBonus += spec.Effects.Active.CaptureBonus
			if spec.Rank > bestCaptureRank {
				bestCaptureRank = spec.Rank
			}
		}
	}

	if totalCaptureBonus == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có cổ trùng nào có khả năng bắt. Cần equip cổ trùng có CaptureBonus."})
		return
	}

	// Kiểm tra rank rule: cổ rank >= monster rank
	monsterRank := player.ActiveCombat.EnemyRank
	if bestCaptureRank < monsterRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cổ trùng rank %d không đủ bắt monster rank %d. Cần rank %d trở lên.", bestCaptureRank, monsterRank, monsterRank)})
		return
	}

	// Tính capture chance theo công thức
	hpRatio := 1.0 - player.ActiveCombat.EnemyHP/player.ActiveCombat.EnemyMaxHP
	if hpRatio < 0.1 {
		hpRatio = 0.1
	}
	if hpRatio > 1.0 {
		hpRatio = 1.0
	}
	hpFactor := math.Pow(hpRatio, 0.5)

	typeFactor := 1.0
	switch strings.ToUpper(monTemplate.Type) {
	case "ELITE":
		typeFactor = 0.6
	case "BOSS", "TRIBULATION_BOSS", "TRIBULATION_MONSTER":
		typeFactor = 0.0
	}

	chance := monTemplate.CaptureChance*hpFactor*typeFactor + totalCaptureBonus
	if chance > 1.0 {
		chance = 1.0
	}
	if chance < 0.01 {
		chance = 0.01
	}

	// Roll
	roll := rand.Float64()
	success := roll < chance

	if success {
		// Bắt thành công — tạo PlayerPet
		hp, atk, def := models.ScaleMonsterStats(
			monTemplate.MaxHP, monTemplate.BaseAttack, monTemplate.BaseDefense,
			player.ActiveCombat.EnemyRank, player.ActiveCombat.EnemySubRank, monTemplate.Type,
		)
		speed := 8.0 + float64(player.ActiveCombat.EnemyRank)*2.0 + float64(player.ActiveCombat.EnemySubRank)*0.5

		_, errPet := tx.Exec(`
			INSERT INTO player_pets (player_id, monster_template_id, nickname, rank, sub_rank,
				hp, max_hp, base_attack, base_defense, speed, count, hunger, max_hunger,
				pet_hunger_per_hour, state, formation_slot, captured_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,1,$11,$12,$13,'ACTIVE',0,NOW(),NOW())
		`,
			player.ID, monTemplate.ID, monTemplate.Name,
			player.ActiveCombat.EnemyRank, player.ActiveCombat.EnemySubRank,
			hp, hp, atk, def, speed,
			float64(defaultMaxHunger(monTemplate.Type)), float64(defaultMaxHunger(monTemplate.Type)), monTemplate.PetHungerPerHour,
		)
		if errPet != nil {
			logger.S.Errorf("CaptureMonster: failed to insert pet: %v", errPet)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu pet."})
			return
		}

		// Kết thúc combat
		enemyRank := player.ActiveCombat.EnemyRank
		enemySubRank := player.ActiveCombat.EnemySubRank
		enemyName := player.ActiveCombat.EnemyName
		enemyType := player.ActiveCombat.EnemyType

		if player.ClearedCells == nil {
			player.ClearedCells = make(models.ClearedCellsMap)
		}
		coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
		player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
		if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearedCellCooldown); err != nil {
			logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
		}
		if player.ActiveCombat != nil {
			_ = cache.BroadcastMonsterKilled(player.X, player.Y, player.ActiveCombat.EnemyName, player.CurrentRegion)
		}
		_ = cache.BroadcastCellCleared(player.CurrentRegion, player.X, player.Y, "Monster", player.ActiveCombat.EnemyTemplateID)

		player.ActiveCombat = nil

		_, errUpd := tx.Exec(`UPDATE players SET active_combat = NULL, cleared_cells = $1 WHERE id = $2`, player.ClearedCells, player.ID)
		if errUpd != nil {
			logger.S.Errorf("CaptureMonster: failed to clear combat: %v", errUpd)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái chiến đấu."})
			return
		}

		// Audit log
		_, _ = tx.Exec(`
			INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
			VALUES ($1, 'pet.capture', 'monster_template', $2, $3, $4, '', true, '', NOW())
		`, player.ID, monTemplate.ID, fmt.Sprintf("Bắt thành công %s", enemyName), fmt.Sprintf(`{"rank":%d,"sub_rank":%d}`, enemyRank, enemySubRank))

		if errTx := tx.Commit(); errTx != nil {
			respondCombatTxErr(c, errTx, "Transaction commit failed")
			return
		}

		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"captured":         true,
			"monster_name":     enemyName,
			"monster_id":       monTemplate.ID,
			"monster_type":     enemyType,
			"capture_chance":   chance,
			"roll":             roll,
			"message":          fmt.Sprintf("Bắt thành công %s!", enemyName),
		})
		return
	}

	// Bắt thất bại — combat vẫn tiếp diễn
	captureLog := fmt.Sprintf("Cố gắng bắt %s thất bại! Tỷ lệ: %.1f%%", player.ActiveCombat.EnemyName, chance*100)
	_, errHist := tx.Exec(`
		INSERT INTO combat_history (player_id, winner, logs, summary, enemy_name, enemy_type, region)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		player.ID, "CAPTURE_FAILED",
		fmt.Sprintf(`["%s"]`, captureLog),
		fmt.Sprintf(`{"capture_chance":%.4f,"roll":%.4f}`, chance, roll),
		player.ActiveCombat.EnemyName, player.ActiveCombat.EnemyType,
		player.CurrentRegion,
	)
	if errHist != nil {
		logger.S.Errorf("CaptureMonster: failed to insert combat_history: %v", errHist)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu lịch sử chiến đấu."})
		return
	}

	if errTx := tx.Commit(); errTx != nil {
		respondCombatTxErr(c, errTx, "Transaction commit failed")
		return
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"captured":       false,
		"monster_name":   player.ActiveCombat.EnemyName,
		"capture_chance": chance,
		"roll":           roll,
		"message":        fmt.Sprintf("Bắt %s thất bại! Tỷ lệ: %.1f%%", player.ActiveCombat.EnemyName, chance*100),
	})
}

func defaultMaxHunger(monsterType string) int {
	switch strings.ToUpper(monsterType) {
	case "ELITE":
		return 150
	case "BOSS", "TRIBULATION_BOSS":
		return 200
	default:
		return 100
	}
}
