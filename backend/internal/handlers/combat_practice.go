package handlers

import (
	"database/sql"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type PracticeCombatRequest struct {
	DummyHP     float64 `json:"dummy_hp"`
	DummyShield float64 `json:"dummy_shield"`
	DummyAtk    float64 `json:"dummy_atk"`
	DummyDef    float64 `json:"dummy_def"`
	DummySpeed  float64 `json:"dummy_speed"`
}

type PracticeCombatConfig struct {
	DummyHP     float64 `json:"dummy_hp"`
	DummyShield float64 `json:"dummy_shield"`
	DummyAtk    float64 `json:"dummy_atk"`
	DummyDef    float64 `json:"dummy_def"`
	DummySpeed  float64 `json:"dummy_speed"`
}

// StartPracticeCombat starts a virtual turn-based training encounter.
func StartPracticeCombat(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req PracticeCombatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config := PracticeCombatConfig{
		DummyHP:     boundedPracticeValue(req.DummyHP, 100000, 1, 1e12),
		DummyShield: boundedPracticeValue(req.DummyShield, 0, 0, 1e12),
		DummyAtk:    boundedPracticeValue(req.DummyAtk, 10, 0, 1e9),
		DummyDef:    boundedPracticeValue(req.DummyDef, 5, 0, 1e9),
		DummySpeed:  boundedPracticeValue(req.DummySpeed, 1, 0, 100),
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở phiên tập luyện."})
		return
	}
	defer tx.Rollback()
	if _, sessionErr := loadCombatSession(tx, playerID, nil, true); sessionErr == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu đang có phiên chiến đấu."})
		return
	} else if sessionErr != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra phiên chiến đấu."})
		return
	}
	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không thể đọc trạng thái nhân vật."})
		return
	}
	if player.ActiveCombat != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "combat_active", "error": "Nhân vật đang chiến đấu."})
		return
	}
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Nhân vật không thể bắt đầu chiến đấu khi đã chết."})
		return
	}
	dummy := models.CombatState{
		EnemyTemplateID: "training_dummy", EnemyName: "Hình nhân", EnemyType: "PRACTICE", CombatMode: "virtual",
		EnemyHP: config.DummyHP, EnemyMaxHP: config.DummyHP, EnemyDamage: config.DummyAtk, EnemyDefense: config.DummyDef,
		EnemySpeed: config.DummySpeed, EnemyRank: 1,
	}
	player.ActiveCombat = InitializeCombatState(tx, &dummy, &player)
	if len(player.ActiveCombat.Enemies) > 0 {
		player.ActiveCombat.Enemies[0].VirtualShield = config.DummyShield
	}
	if _, err := tx.ExecContext(c.Request.Context(), `UPDATE players SET active_combat=$1 WHERE id=$2`, player.ActiveCombat, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu phiên tập luyện."})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu phiên tập luyện."})
		return
	}
	PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	respondActionOnly(c, http.StatusOK, player, gin.H{"is_practice": true, "active_combat": player.ActiveCombat})
}

// EndPracticeCombat clears an unresolved practice encounter without applying
// combat costs or world movement.
func EndPracticeCombat(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kết thúc tập luyện."})
		return
	}
	defer tx.Rollback()
	activeSession, sessionErr := loadCombatSession(tx, playerID, nil, true)
	if sessionErr != nil && sessionErr != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra phiên tập luyện."})
		return
	}
	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không thể đọc trạng thái nhân vật."})
		return
	}
	if player.ActiveCombat == nil || player.ActiveCombat.EnemyType != "PRACTICE" {
		c.JSON(http.StatusConflict, gin.H{"code": "training_session_inactive", "error": "Không có phiên tập luyện đang mở."})
		return
	}
	if sessionErr == nil {
		if _, err := tx.Exec(`UPDATE combat_sessions SET status='abandoned', pending_actor='', deadline_at=NULL, updated_at=NOW() WHERE id=$1 AND status='active'`, activeSession.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kết thúc phiên tập luyện."})
			return
		}
	}
	player.ActiveCombat = nil
	if _, err := tx.ExecContext(c.Request.Context(), `UPDATE players SET active_combat=NULL WHERE id=$1`, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kết thúc tập luyện."})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kết thúc tập luyện."})
		return
	}
	PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
	c.JSON(http.StatusOK, gin.H{"is_practice": false})
}

func boundedPracticeValue(value, fallback, minimum, maximum float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	if value == 0 && fallback != 0 {
		return fallback
	}
	return math.Max(minimum, math.Min(maximum, value))
}
