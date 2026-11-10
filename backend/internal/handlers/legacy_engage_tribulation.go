package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func EngageTribulation(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}
	handedOff, err := middleware.BeginRealtimeHandoff(playerID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Không thể đồng bộ trạng thái thế giới trước khi nghênh chiến."})
		return
	}
	committed := false
	defer func() {
		if handedOff && !committed {
			middleware.AbortRealtimeHandoff(playerID)
		}
	}()

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	suppressed, err := suppressTribulationsForDeadAperture(tx, playerID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra trạng thái Tiên Cương"})
		return
	}
	if suppressed {
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Cương có Tử Khiếu, không phải chịu tai kiếp"})
		return
	}

	// 1. Validate Tribulation Event
	var evStatus string
	var apertureID string
	err = tx.QueryRow(`
		SELECT status, aperture_id FROM tribulation_events WHERE id = $1 AND player_id = $2
	`, eventID, playerID).Scan(&evStatus, &apertureID)
	if err == sql.ErrNoRows || evStatus != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đợt tai kiếp này đã kết thúc hoặc không tồn tại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra tai kiếp"})
		return
	}

	// 2. Aperture gate: player may only fight while the Aperture is above 5% max HP
	var apHP, apMaxHP float64
	err = tx.QueryRow(`
		SELECT aperture_hp, max_aperture_hp FROM immortal_apertures WHERE id = $1
	`, apertureID).Scan(&apHP, &apMaxHP)
	if err == nil && apHP*20 <= apMaxHP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu đang nguy cấp (HP ≤ 5%)! Quái tai kiếp đang càn quét, đạo hữu không thể ra tay. Hãy chờ Tiên Khiếu hồi phục hoặc kết toán vượt kiếp."})
		return
	}

	// 3. Load player (equipped/vital inventory) and reject if already in combat.
	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin cổ sư"})
		return
	}
	if player.ActiveCombat != nil {
		if player.ActiveCombat.TribulationEventID == eventIDStr {
			committed = true
			if handedOff {
				middleware.FinishRealtimeHandoff(playerID)
			}
			PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
			respondActionOnly(c, http.StatusOK, player, gin.H{
				"message":       "Tiếp tục trận chiến tai kiếp!",
				"active_combat": player.ActiveCombat,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đang trong trạng thái chiến đấu khác! Hãy kết thúc trận chiến trước."})
		return
	}

	// 4. Load all live enemies of this event (boss first for stable ordering).
	rows, err := tx.Query(`
		SELECT id, name, hp, max_hp, attack, defense, is_boss, rank, sub_rank
		FROM tribulation_enemies
		WHERE event_id = $1 AND is_dead = false
		ORDER BY is_boss DESC, max_hp DESC
	`, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy danh sách quái tai kiếp"})
		return
	}
	defer rows.Close()

	type liveEnemy struct {
		ID      string
		Name    string
		HP      float64
		MaxHP   float64
		Attack  float64
		Defense float64
		IsBoss  bool
		Rank    int
		SubRank int
	}
	var enemies []liveEnemy
	for rows.Next() {
		var e liveEnemy
		if err := rows.Scan(&e.ID, &e.Name, &e.HP, &e.MaxHP, &e.Attack, &e.Defense, &e.IsBoss, &e.Rank, &e.SubRank); err != nil {
			logger.S.Errorf("EngageTribulation rows.Scan failed for event %s: %v", eventID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc quái tai kiếp"})
			return
		}
		enemies = append(enemies, e)
	}
	if len(enemies) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không còn kẻ địch nào để chiến đấu. Hãy kết toán vượt kiếp."})
		return
	}

	first := enemies[0]
	combatType := "TRIBULATION_MONSTER"
	if first.IsBoss {
		combatType = "TRIBULATION_BOSS"
	}

	enemyRank := first.Rank
	if enemyRank < 6 {
		enemyRank = player.Rank
		if enemyRank < 6 {
			enemyRank = 6
		}
	}
	enemySubRank := first.SubRank
	if enemySubRank < 1 {
		enemySubRank = 1
	}

	base := models.CombatState{
		EnemyTemplateID:    first.ID,
		EnemyName:          first.Name,
		EnemyType:          combatType,
		CombatMode:         "real",
		EnemyHP:            float64(first.HP),
		EnemyMaxHP:         float64(first.MaxHP),
		EnemyDamage:        float64(first.Attack),
		EnemyDefense:       float64(first.Defense),
		EnemyRank:          enemyRank,
		EnemySubRank:       enemySubRank,
		TargetPlayerID:     "",
		TribulationEventID: eventIDStr,
		ApertureHP:         float64(apHP),
		ApertureMaxHP:      float64(apMaxHP),
	}

	state := StartCombat(tx, &base, &player)

	// Fix the first enemy combatant (built from flat fields): tribulation enemies
	// have no Gu inventory and use basic attacks only, so we just correct its Kind.
	if first.IsBoss {
		state.Enemies[0].Kind = "Boss"
	} else {
		state.Enemies[0].Kind = "Normal"
	}

	// Append remaining live enemies and assign grid slots.
	enemySlots := [][2]int{
		{2, 3}, {2, 1}, {1, 1}, {3, 1}, {2, 2}, {1, 2}, {3, 2}, {1, 3}, {3, 3},
	}
	for i := 1; i < len(enemies); i++ {
		e := enemies[i]
		kind := "Normal"
		if e.IsBoss {
			kind = "Boss"
		}
		eRank := e.Rank
		if eRank < 6 {
			eRank = enemyRank
		}
		eSubRank := e.SubRank
		if eSubRank < 1 {
			eSubRank = 1
		}
		c := models.Combatant{
			ID:          e.ID,
			Name:        e.Name,
			Kind:        kind,
			Rank:        eRank,
			SubRank:     eSubRank,
			HP:          float64(e.HP),
			MaxHP:       float64(e.MaxHP),
			CurrentMana: 100,
			MaxMana:     100,
			BaseAttack:  float64(e.Attack),
			BaseDefense: float64(e.Defense),
			Speed:       10.0,
		}
		if i < len(enemySlots) {
			c.GridX = enemySlots[i][0]
			c.GridY = enemySlots[i][1]
		}
		state.Enemies = append(state.Enemies, c)
	}

	combatBytes, err := json.Marshal(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo trạng thái chiến đấu"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET active_combat = $1, human_qi = LEAST(human_qi + 3, $2) WHERE id = $3`, string(combatBytes), MaxHumanQi, playerID)
	if err != nil {
		logger.S.Errorf("StartTribulation: failed to save active combat for player=%s, event=%s, enemies=%d: %v", playerID, eventIDStr, len(state.Enemies), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu trạng thái chiến đấu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	committed = true
	if handedOff {
		middleware.FinishRealtimeHandoff(playerID)
	}

	player.ActiveCombat = state
	player.HumanQi = math.Min(float64(MaxHumanQi), player.HumanQi+3)
	PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	respondActionOnly(c, http.StatusOK, player, gin.H{
		"message":       fmt.Sprintf("Xuất trận nghênh chiến %d quái tai kiếp! Nhân Khí +3.", len(enemies)),
		"active_combat": state,
	})
}
