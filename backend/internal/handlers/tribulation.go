package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
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
)

// GetActiveTribulation returns the active tribulation event and its enemies for the player
func GetActiveTribulation(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch active event
	var ev models.TribulationEvent
	var endedAt sql.NullTime
	var hpAfter sql.NullFloat64
	err = tx.QueryRow(`
		SELECT id, aperture_id, player_id, tribulation_type, path, difficulty, status, aperture_hp_before, aperture_hp_after, reward_dao_marks, started_at, ended_at
		FROM tribulation_events
		WHERE player_id = $1 AND status = 'ACTIVE'
		LIMIT 1
	`, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.Difficulty, &ev.Status, &ev.ApertureHPBefore, &hpAfter, &ev.RewardDaoMarks, &ev.StartedAt, &endedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, nil)
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn kiếp nạn"})
		return
	}

	if hpAfter.Valid {
		ev.ApertureHPAfter = &hpAfter.Float64
	}
	if endedAt.Valid {
		ev.EndedAt = &endedAt.Time
	}

	// 2. Fetch enemies
	rows, err := tx.Query(`
		SELECT id, event_id, name, enemy_type, path, hp, max_hp, attack, defense, target_node_id, x, y, is_boss, is_dead, rank, sub_rank
		FROM tribulation_enemies
		WHERE event_id = $1 AND is_dead = false
	`, ev.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy danh sách quái tai kiếp"})
		return
	}
	defer rows.Close()

	enemies := []models.TribulationEnemy{}
	for rows.Next() {
		var en models.TribulationEnemy
		var targetID sql.NullString
		err = rows.Scan(&en.ID, &en.EventID, &en.Name, &en.EnemyType, &en.Path, &en.HP, &en.MaxHP, &en.Attack, &en.Defense, &targetID, &en.X, &en.Y, &en.IsBoss, &en.IsDead, &en.Rank, &en.SubRank)
		if err == nil {
			if targetID.Valid {
				en.TargetNodeID = &targetID.String
			}
			enemies = append(enemies, en)
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"event":   ev,
		"enemies": enemies,
	})
}

// EngageTribulation starts a standard combat encounter against the live enemies
// of an active tribulation (tai kiếp) event. The resulting combat state is
// persisted into players.active_combat and fought via the shared auto-combat
// simulator (POST /api/combat/start). Enemy damage that overflows the player's
// shield + HP spills into the Immortal Aperture (Tiên Khiếu).
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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang ở trong trạng thái chiến đấu!"})
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
		MineSlotID:         0,
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

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Xuất trận nghênh chiến %d quái tai kiếp! Nhân Khí +3.", len(enemies))})
}

// ResolveTribulation resolves the tribulation event manually if all enemies are dead, or auto-resolves with risk
func ResolveTribulation(c *gin.Context) {
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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch active event
	var ev models.TribulationEvent
	err = tx.QueryRow(`
		SELECT id, aperture_id, player_id, tribulation_type, path, difficulty, reward_dao_marks, started_at
		FROM tribulation_events
		WHERE id = $1 AND player_id = $2 AND status = 'ACTIVE'
	`, eventID, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.Difficulty, &ev.RewardDaoMarks, &ev.StartedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Đợt tai kiếp này đã được giải quyết hoặc không tồn tại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn tai kiếp"})
		return
	}

	// 2. Check if all enemies are dead
	var remainingEnemies int
	err = tx.QueryRow(`
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1 AND is_dead = false
	`, ev.ID).Scan(&remainingEnemies)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra số quái còn sống"})
		return
	}

	var ap models.ImmortalAperture
	err = tx.QueryRow(`
		SELECT id, aperture_hp, max_aperture_hp, next_tribulation_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, internal_years_passed
		FROM immortal_apertures WHERE id = $1
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.NextTribulationYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.InternalYearsPassed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin Tiên Khiếu"})
		return
	}

	var msg string
	now := time.Now()
	didBreakthrough := false
	breakthroughMsg := ""
	updatedDaoMarks := make(map[string]int)

	// Helper: recalc next_year, check breakthrough
	handleSuccess := func(tribType string) {
		switch tribType {
		case "EARTH_CALAMITY":
			ap.EarthCalamityCount++
		case "HEAVENLY_TRIBULATION":
			ap.HeavenlyTribulationCount++
		case "GRAND_TRIBULATION":
			ap.GrandTribulationCount++
		case "MYRIAD_TRIBULATION":
			ap.MyriadTribulationCount++
		case "CHAOS_CALAMITY":
			ap.ChaosCalamityCount++
		}

		// Recalculate next tribulation year based on rank & counters
		var playerRank int
		var username string
		_ = tx.QueryRow(`SELECT rank, username FROM players WHERE id = $1`, playerID).Scan(&playerRank, &username)

		nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, ap.InternalYearsPassed)
		ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)

		// Check breakthrough conditions
		canBreak, newRank := models.CheckImmortalBreakthrough(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
		if canBreak {
			// Rank-up requirement: player must own at least one path (lưu phái)
			// at the required Cảnh Giới — Rank 7 = Tông Sư, Rank 8 = Đại Tông Sư,
			// Rank 9 = Vô Thượng Đại Tông Sư.
			var pathMasteries models.PathMasteriesMap
			_ = tx.QueryRow(`SELECT COALESCE(path_masteries, '{}'::jsonb) FROM players WHERE id = $1`, playerID).Scan(&pathMasteries)
			reqLevel := gamefx.PathMasteryLevelRequiredForRank(newRank)
			if !gamefx.HasPathMasteryAtLeast(pathMasteries, reqLevel) {
				blockedMsg := fmt.Sprintf(" Chưa đủ điều kiện đột phá lên %s: cần ít nhất 1 lưu phái đạt %s (%d điểm).", getRankName(newRank), gamefx.PathMasteryLevelName(reqLevel), gamefx.PathMasteryPointsForLevel(reqLevel))
				_, _ = tx.Exec(`INSERT INTO messages (sender_id, receiver_id, content, channel) VALUES ($1, $1, $2, 'system')`, playerID, "Đột phá bị chặn: "+blockedMsg)
				breakthroughMsg = blockedMsg
			} else {
				_, _ = tx.Exec(`UPDATE players SET rank = $1 WHERE id = $2`, newRank, playerID)
				ap.RankStartYear = math.Floor(ap.InternalYearsPassed)
				// Reset per-type counters: each rank has its own tribulation set
				// and cadence (lower types drop off), so the schedule restarts.
				ap.EarthCalamityCount = 0
				ap.HeavenlyTribulationCount = 0
				ap.GrandTribulationCount = 0
				ap.MyriadTribulationCount = 0
				ap.ChaosCalamityCount = 0
				didBreakthrough = true
				breakthroughMsg = fmt.Sprintf(" Đột phá! Đạo hữu đã tiến vào %s!", getRankName(newRank))

				if newRank == 7 || newRank == 8 || newRank == 9 {
					_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Chúc mừng đạo hữu %s đã đột phá thành công lên %s!", username, getRankName(newRank)))
					auditlog.Log(tx, auditlog.Entry{
						PlayerID:       &playerID,
						Action:         "player.breakthrough_success",
						EntityType:     "player",
						EntityID:       playerID.String(),
						RequestSummary: fmt.Sprintf("Thăng rank %d (%s) thành công", newRank, getRankName(newRank)),
						Success:        true,
					})
				}

				// Recalculate next tribulation year based on the new rank!
				nextYear, _ = models.GetNextTribulationInfo(newRank, ap.RankStartYear, ap.InternalYearsPassed)
				ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)
			}
		}

		// Update current event status and dead enemies to evaluate stability correctly
		_, _ = tx.Exec(`UPDATE tribulation_events SET status = 'SUCCESS', aperture_hp_after = $1, ended_at = $2 WHERE id = $3`, ap.ApertureHP, now, ev.ID)
		_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

		hasActive, errActive := db.HasActiveTribulationOrEnemies(tx, ap.ID)
		if errActive == nil && !hasActive {
			ap.State = "STABLE"
		} else {
			ap.State = "TRIBULATION_ACTIVE"
		}
	}

	if remainingEnemies == 0 {
		// Clear Victory! All enemies defeated = guaranteed success
		rewards := distributeDaoMarks(ev.Path, ev.RewardDaoMarks)
		daoMap, errDao := addMultipleDaoMarksTx(tx, playerID.String(), rewards)
		if errDao != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Đạo Ngân: " + errDao.Error()})
			return
		}
		updatedDaoMarks = daoMap

		ap.ApertureHP = ap.MaxApertureHP
		handleSuccess(ev.TribulationType)

		_, err = tx.Exec(`
			UPDATE immortal_apertures 
			SET state = $1, next_tribulation_year = $2, aperture_hp = $3,
				rank_start_year = $4, earth_calamity_count = $5, heavenly_tribulation_count = $6,
				grand_tribulation_count = $7, myriad_tribulation_count = $8, chaos_calamity_count = $9
			WHERE id = $10
		`, ap.State, ap.NextTribulationYear, ap.ApertureHP,
			ap.RankStartYear, ap.EarthCalamityCount, ap.HeavenlyTribulationCount,
			ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Tiên Khiếu"})
			return
		}

		rewardStrings := []string{}
		for path, amount := range rewards {
			rewardStrings = append(rewardStrings, fmt.Sprintf("%d Đạo Ngân hệ %s", amount, strings.ToUpper(path)))
		}
		rewardText := strings.Join(rewardStrings, ", ")
		msg = fmt.Sprintf("Chúc mừng đạo hữu vượt kiếp nạn thành công! Khai quang nhận %s. Tiên Khiếu của bạn đã ổn định trở lại.%s", rewardText, breakthroughMsg)

		if didBreakthrough {
			_, _ = tx.Exec(`
				INSERT INTO messages (sender_id, receiver_id, content, channel)
				VALUES ('11111111-1111-1111-1111-111111111111', $1, $2, 'private')
			`, playerID, breakthroughMsg)
		}
	} else {
		// Player resolves manually with remaining enemies (yields high risk of damage)
		successChance := 0.25
		if remainingEnemies == 1 {
			successChance = 0.50
		}

		win := rand.Float64() < successChance
		if win {
			reducedReward := ev.RewardDaoMarks / 2
			rewards := distributeDaoMarks(ev.Path, reducedReward)
			daoMap, errDao := addMultipleDaoMarksTx(tx, playerID.String(), rewards)
			if errDao != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Đạo Ngân: " + errDao.Error()})
				return
			}
			updatedDaoMarks = daoMap

			ap.ApertureHP = ap.MaxApertureHP / 2
			handleSuccess(ev.TribulationType)

			_, _ = tx.Exec(`
				UPDATE aperture_resource_nodes 
				SET is_destroyed = true 
				WHERE id IN (
					SELECT id FROM aperture_resource_nodes 
					WHERE aperture_id = $1 AND is_destroyed = false 
					ORDER BY random() LIMIT 1
				)
			`, ap.ID)

			_, _ = tx.Exec(`
				UPDATE immortal_apertures 
				SET state = $1, next_tribulation_year = $2, aperture_hp = $3,
					rank_start_year = $4, earth_calamity_count = $5, heavenly_tribulation_count = $6,
					grand_tribulation_count = $7, myriad_tribulation_count = $8, chaos_calamity_count = $9
				WHERE id = $10
			`, ap.State, ap.NextTribulationYear, ap.ApertureHP,
				ap.RankStartYear, ap.EarthCalamityCount, ap.HeavenlyTribulationCount,
				ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.ID)

			_, _ = tx.Exec(`
				UPDATE tribulation_events
				SET status = 'SUCCESS', aperture_hp_after = $1, ended_at = $2, reward_dao_marks = $3
				WHERE id = $4
			`, ap.ApertureHP, now, reducedReward, ev.ID)

			rewardStrings := []string{}
			for path, amount := range rewards {
				rewardStrings = append(rewardStrings, fmt.Sprintf("%d Đạo Ngân hệ %s", amount, strings.ToUpper(path)))
			}
			rewardText := strings.Join(rewardStrings, ", ")
			msg = fmt.Sprintf("Đạo hữu vượt kiếp chật vật thành công! Nhận %s, nhưng Tiên Khiếu bị chấn động mạnh (HP giảm còn 50%%), 1 Điểm tài nguyên ngẫu nhiên đã bị phá hủy.%s", rewardText, breakthroughMsg)

			if didBreakthrough {
				_, _ = tx.Exec(`
					INSERT INTO messages (sender_id, receiver_id, content, channel)
					VALUES ('11111111-1111-1111-1111-111111111111', $1, $2, 'private')
				`, playerID, breakthroughMsg)
			}
		} else {
			// Failed Tribulation
			var playerRank int
			_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&playerRank)

			// Counts are only incremented on success — a failed tribulation does
			// not count toward the total / per-type counters, but the next year
			// is still pushed forward to the next milestone for recovery time.
			nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, ap.InternalYearsPassed)
			ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)

			ap.State = "DAMAGED"
			ap.ApertureHP = ap.MaxApertureHP / 10

			// Phá hủy 30% tài nguyên
			var activeNodesCount int
			_ = tx.QueryRow(`SELECT COUNT(*) FROM aperture_resource_nodes WHERE aperture_id = $1 AND is_destroyed = false`, ap.ID).Scan(&activeNodesCount)
			nodesToDestroy := int(float64(activeNodesCount) * 0.3)
			if nodesToDestroy < 1 && activeNodesCount > 0 {
				nodesToDestroy = 1
			}

			if nodesToDestroy > 0 {
				_, _ = tx.Exec(`
					UPDATE aperture_resource_nodes 
					SET is_destroyed = true 
					WHERE id IN (
						SELECT id FROM aperture_resource_nodes 
						WHERE aperture_id = $1 AND is_destroyed = false 
						ORDER BY random() 
						LIMIT $2
					)
				`, ap.ID, nodesToDestroy)
			}

			_, _ = tx.Exec(`
				UPDATE players 
				SET primeval_stones = GREATEST(0, primeval_stones - 2000), immortal_stones = GREATEST(0.0, immortal_stones - 5.0)
				WHERE id = $1
			`, playerID)

			// Giảm diện tích và tốc độ thời gian
			_, _ = tx.Exec(`
				UPDATE immortal_apertures 
				SET state = $1, next_tribulation_year = $2, aperture_hp = $3,
					earth_calamity_count = $4, heavenly_tribulation_count = $5, grand_tribulation_count = $6,
					myriad_tribulation_count = $7, chaos_calamity_count = $8,
					area_size = GREATEST(0, CAST(area_size AS numeric) * 0.9),
					time_rate = GREATEST(1.0, time_rate * 0.9)
				WHERE id = $9
			`, ap.State, ap.NextTribulationYear, ap.ApertureHP,
				ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount,
				ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.ID)

			_, _ = tx.Exec(`
				UPDATE tribulation_events
				SET status = 'FAILED', aperture_hp_after = $1, ended_at = $2, reward_dao_marks = 0
				WHERE id = $3
			`, ap.ApertureHP, now, ev.ID)

			msg = "Kiếp nạn vượt thất bại! Thiên Ý cuồng bạo tàn phá Tiên Khiếu, Nguyên khí hư hao, bạn mất 2,000 Nguyên Thạch, 5 Tiên Nguyên, diện tích Tiên Khiếu và tốc độ thời gian bị giảm, 30% điểm tài nguyên bị san phẳng."
		}
	}

	// Clean up dead/remaining enemies
	_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 10, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Check if the player is immediately due for another tribulation
	_ = checkAndTriggerSingleAperture(ev.ApertureID)

	var finalState string
	_ = db.DB.QueryRow(`SELECT state FROM immortal_apertures WHERE id = $1`, ev.ApertureID).Scan(&finalState)

	var finalDaoMarks map[string]int
	if len(updatedDaoMarks) > 0 {
		finalDaoMarks = updatedDaoMarks
	} else {
		var finalDaoJSON []byte
		if err2 := db.DB.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&finalDaoJSON); err2 == nil {
			json.Unmarshal(finalDaoJSON, &finalDaoMarks)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   msg + " Nhân Khí +10.",
		"state":     finalState,
		"dao_marks": finalDaoMarks,
	})
}

func checkAndTriggerSingleAperture(apertureID string) error {
	var ap struct {
		ID                       string
		PlayerID                 string
		TimeRate                 float64
		ApertureHP               float64
		MaxApertureHP            float64
		InternalYearsPassed      float64
		NextTribulationYear      float64
		EssenceProductionPerYear int
		RankStartYear            float64
		EarthCalamityCount       int
		HeavenlyTribulationCount int
		GrandTribulationCount    int
		MyriadTribulationCount   int
		ChaosCalamityCount       int
		State                    string
	}

	err := db.DB.QueryRow(`
		SELECT id, player_id, time_rate, aperture_hp, max_aperture_hp, internal_years_passed, next_tribulation_year, essence_production_per_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, state
		FROM immortal_apertures
		WHERE id = $1
	`, apertureID).Scan(&ap.ID, &ap.PlayerID, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.EssenceProductionPerYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.State)
	if err != nil {
		return err
	}

	if ap.State != "STABLE" && ap.State != "DAMAGED" {
		return nil
	}

	var rank int
	err = db.DB.QueryRow(`SELECT rank FROM players WHERE id = $1`, ap.PlayerID).Scan(&rank)
	if err != nil {
		return err
	}

	calcNextYear, calcNextType := models.GetNextTribulationInfo(rank, ap.RankStartYear, ap.InternalYearsPassed)

	triggerYear := guardNextTribulationYear(calcNextYear, ap.InternalYearsPassed)
	storedYear := guardNextTribulationYear(ap.NextTribulationYear, ap.InternalYearsPassed)
	if storedYear >= ap.InternalYearsPassed || (ap.InternalYearsPassed-storedYear < 10.0) {
		if storedYear < triggerYear {
			triggerYear = storedYear
			if scheduledType := models.GetTribulationTypeAtYear(rank, ap.RankStartYear, storedYear); scheduledType != "" {
				calcNextType = scheduledType
			}
		} else if math.Abs(storedYear-triggerYear) < 0.001 {
			if scheduledType := models.GetTribulationTypeAtYear(rank, ap.RankStartYear, triggerYear); scheduledType != "" {
				calcNextType = scheduledType
			}
		}
	}

	if ap.InternalYearsPassed >= triggerYear {
		spawnType := calcNextType
		err = spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.EarthCalamityCount+ap.HeavenlyTribulationCount+ap.GrandTribulationCount+ap.MyriadTribulationCount+ap.ChaosCalamityCount+1, ap.ApertureHP, ap.InternalYearsPassed, spawnType, 1.0)
		if err != nil {
			return err
		}
	} else {
		_, err = db.DB.Exec(`
			UPDATE immortal_apertures 
			SET next_tribulation_year = $1, updated_at = NOW()
			WHERE id = $2
		`, triggerYear, ap.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func daoMarkGainMultiplierTx(tx *sql.Tx, playerID string) float64 {
	var constitution string
	if err := tx.QueryRow(`SELECT COALESCE(constitution, '') FROM players WHERE id = $1`, playerID).Scan(&constitution); err != nil {
		return 1.0
	}
	return 1.0 + gamefx.Get(&models.Player{CharacterState: models.CharacterState{Constitution: constitution}}).DaoMarkGainBonus
}

func addDaoMarksTx(tx *sql.Tx, playerID string, path string, amount int) error {
	amount = int(float64(amount) * daoMarkGainMultiplierTx(tx, playerID))
	var daoJSON []byte
	err := tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1 FOR UPDATE`, playerID).Scan(&daoJSON)
	if err != nil {
		return err
	}

	daoMap := make(map[string]int)
	if len(daoJSON) > 0 {
		if err := json.Unmarshal(daoJSON, &daoMap); err != nil {
			return err
		}
	}
	daoMap[path] += amount

	daoJSON, err = json.Marshal(daoMap)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE immortal_apertures SET dao_marks = $1 WHERE player_id = $2`, daoJSON, playerID)
	return err
}

func addMultipleDaoMarksTx(tx *sql.Tx, playerID string, rewards map[string]int) (map[string]int, error) {
	mult := daoMarkGainMultiplierTx(tx, playerID)
	for path, amount := range rewards {
		rewards[path] = int(float64(amount) * mult)
	}
	var daoJSON []byte
	err := tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1 FOR UPDATE`, playerID).Scan(&daoJSON)
	if err != nil {
		return nil, err
	}

	daoMap := make(map[string]int)
	if len(daoJSON) > 0 {
		if err := json.Unmarshal(daoJSON, &daoMap); err != nil {
			return nil, err
		}
	}
	for path, amount := range rewards {
		daoMap[path] += amount
	}

	daoJSON, err = json.Marshal(daoMap)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`UPDATE immortal_apertures SET dao_marks = $1 WHERE player_id = $2`, daoJSON, playerID)
	if err != nil {
		return nil, err
	}
	return daoMap, nil
}

func distributeDaoMarks(mainPath string, totalMarks int) map[string]int {
	rewards := make(map[string]int)
	if totalMarks <= 0 {
		return rewards
	}
	// Main path gets 70%
	mainAmount := int(float64(totalMarks) * 0.7)
	if mainAmount < 1 {
		mainAmount = totalMarks
	}
	rewards[mainPath] = mainAmount

	remaining := totalMarks - mainAmount
	if remaining > 0 {
		allPaths := models.AllGuPaths()
		otherPaths := []string{}
		for _, p := range allPaths {
			if p != mainPath {
				otherPaths = append(otherPaths, p)
			}
		}
		rand.Shuffle(len(otherPaths), func(i, j int) {
			otherPaths[i], otherPaths[j] = otherPaths[j], otherPaths[i]
		})

		numExtra := rand.Intn(2) + 1 // 1 or 2 extra paths
		if numExtra > len(otherPaths) {
			numExtra = len(otherPaths)
		}

		if numExtra == 1 {
			rewards[otherPaths[0]] = remaining
		} else {
			part1 := remaining / 2
			part2 := remaining - part1
			rewards[otherPaths[0]] = part1
			rewards[otherPaths[1]] = part2
		}
	}
	return rewards
}

// Background Worker Tick: Advance time and trigger tribulations
// guardNextTribulationYear ensures the next scheduled tribulation never lands
// behind the current internal year (e.g. after a rank-up, where the new rank's
// schedule is computed from intervals smaller than the already-passed year).
// A milestone within the current 10-year window (e.g. overdue by a few moments)
// is left as-is so that due events trigger cleanly.
// If nextYear is more than 10 years in the past (e.g. corrupted or stale data),
// it is pushed forward to the next valid milestone strictly greater than currentYears.
// Also guards against math.MaxFloat64 (infinity) returned when no valid tribulation
// type exists for the current rank (e.g., rank < 6 or rank > 9).
func guardNextTribulationYear(nextYear, currentYears float64) float64 {
	if math.IsInf(nextYear, 0) || nextYear == math.MaxFloat64 {
		return math.Ceil(currentYears) + 10
	}
	if currentYears-nextYear > 10.0 {
		return math.Floor(currentYears/10)*10 + 10
	}
	return math.Round(nextYear)
}

// Offline-protection tuning for tribulations. While the player is offline the
// tribulation is held (no spawn / no damage); the years still flow so players
// cannot game the calendar. A grace period after spawn/return lets them prepare.
const (
	tribulationOnlineWindow = 5 * time.Minute // how fresh last_activity must be to count as "online"
	tribulationGracePeriod  = 5 * time.Minute // enemies idle this long after spawn/return
	tribulationPenaltyAfter = 24 * time.Hour  // offline hold beyond this halves the Dao Ngân reward
)

var (
	wsConnectedMu sync.RWMutex
	wsConnectedFn func(playerID uuid.UUID) bool
)

// SetWSConnectedProvider sets the provider function for checking live WebSocket connections.
func SetWSConnectedProvider(fn func(playerID uuid.UUID) bool) {
	wsConnectedMu.Lock()
	defer wsConnectedMu.Unlock()
	wsConnectedFn = fn
}

func isWSConnected(playerID uuid.UUID) bool {
	wsConnectedMu.RLock()
	fn := wsConnectedFn
	wsConnectedMu.RUnlock()
	if fn == nil {
		return false
	}
	return fn(playerID)
}

// isPlayerOnline reports whether the player has been active within the online window.
// It checks 3 tiers:
// 1. Live WebSocket connection in ws.Hub (instant O(1), zero DB query).
// 2. In-memory recent activity from WebSocket pings/messages (lastActivityStore.recent).
// 3. Persistent players.last_activity in PostgreSQL (fallback).
func isPlayerOnline(playerID string) bool {
	uid, err := uuid.Parse(playerID)
	if err == nil {
		// Tier 1: Real-time WebSocket connection
		if isWSConnected(uid) {
			return true
		}
		// Tier 2: Recent activity in memory
		if IsPlayerActiveRecently(uid, tribulationOnlineWindow) {
			return true
		}
	}

	// Tier 3: Fallback to PostgreSQL
	if db.DB == nil {
		return false
	}
	var la sql.NullTime
	err = db.DB.QueryRow(`SELECT last_activity FROM players WHERE id = $1`, playerID).Scan(&la)
	if err != nil || !la.Valid {
		return false
	}
	return time.Since(la.Time) <= tribulationOnlineWindow
}
func CheckAndTriggerTribulations() error {
	// Query stable apertures
	rows, err := db.DB.Query(`
		SELECT id, player_id, time_rate, area_size, aperture_hp, max_aperture_hp, internal_years_passed, next_tribulation_year, essence_production_per_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, tribulation_pending_at
		FROM immortal_apertures
		WHERE state IN ('STABLE', 'DAMAGED', 'TRIBULATION_PENDING')
	`)
	if err != nil {
		return err
	}

	type ApertureCheck struct {
		ID                       string
		PlayerID                 string
		TimeRate                 float64
		AreaSize                 float64
		ApertureHP               float64
		MaxApertureHP            float64
		InternalYearsPassed      float64
		NextTribulationYear      float64
		EssenceProductionPerYear int
		RankStartYear            float64
		EarthCalamityCount       int
		HeavenlyTribulationCount int
		GrandTribulationCount    int
		MyriadTribulationCount   int
		ChaosCalamityCount       int
		TribulationPendingAt     sql.NullTime
	}

	apertures := []ApertureCheck{}
	for rows.Next() {
		var ap ApertureCheck
		if err := rows.Scan(&ap.ID, &ap.PlayerID, &ap.TimeRate, &ap.AreaSize, &ap.ApertureHP, &ap.MaxApertureHP, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.EssenceProductionPerYear,
			&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.TribulationPendingAt); err == nil {
			apertures = append(apertures, ap)
		}
	}

	if len(apertures) > 0 {
		// ── Batch-read: player lifespan, time_speed, rank (1 query) ──────────
		type pdata struct {
			lifespan float64
			ts       float64
			rank     int
			race     string
			isDead   bool
		}
		pMap := make(map[string]*pdata, len(apertures))
		{
			pids := make([]string, len(apertures))
			for i, ap := range apertures {
				pids[i] = ap.PlayerID
			}
			if pRows, qErr := db.DB.Query(`SELECT id, lifespan, time_speed_multiplier, rank, COALESCE(race, ''), COALESCE(is_dead, false) FROM players WHERE id = ANY($1::text[])`, pq.Array(pids)); qErr == nil {
				defer pRows.Close()
				for pRows.Next() {
					var id string
					var d pdata
					if err := pRows.Scan(&id, &d.lifespan, &d.ts, &d.rank, &d.race, &d.isDead); err == nil {
						pMap[id] = &d
					}
				}
			}
		}

		// ── Batch-read: all nodes for affected apertures (1 query) ────────────
		nodesByAperture := make(map[string][]models.ApertureResourceNode)
		{
			aids := make([]string, len(apertures))
			for i, ap := range apertures {
				aids[i] = ap.ID
			}
			if nRows, qErr := db.DB.Query(`
				SELECT aperture_id, id, node_type, name, path, hp, max_hp, x, y, is_destroyed, zone_type, production_amount
				FROM aperture_resource_nodes
				WHERE aperture_id = ANY($1::text[]) AND is_destroyed = false
			`, pq.Array(aids)); qErr == nil {
				defer nRows.Close()
				for nRows.Next() {
					var nd models.ApertureResourceNode
					var aid string
					if err := nRows.Scan(&aid, &nd.ID, &nd.NodeType, &nd.Name, &nd.Path, &nd.HP, &nd.MaxHP, &nd.X, &nd.Y, &nd.IsDestroyed, &nd.ZoneType, &nd.ProductionAmount); err == nil {
						nd.ApertureID = aid
						nodesByAperture[aid] = append(nodesByAperture[aid], nd)
					}
				}
			}
		}

		// ── Batch-read: all variant humans (1 query) ──────────────────────────
		type vhRow struct {
			id, race, laborIntensity string
			population               int
			happiness, oppression    float64
		}
		vhByAperture := make(map[string][]vhRow)
		{
			aids := make([]string, len(apertures))
			for i, ap := range apertures {
				aids[i] = ap.ID
			}
			if vRows, qErr := db.DB.Query(`
				SELECT aperture_id, id, race, population, happiness, oppression, labor_intensity
				FROM aperture_variant_humans
				WHERE aperture_id = ANY($1::text[])
			`, pq.Array(aids)); qErr == nil {
				defer vRows.Close()
				for vRows.Next() {
					var v vhRow
					var aid string
					if err := vRows.Scan(&aid, &v.id, &v.race, &v.population, &v.happiness, &v.oppression, &v.laborIntensity); err == nil {
						vhByAperture[aid] = append(vhByAperture[aid], v)
					}
				}
			}
		}

		// ── Accumulate batch writes ───────────────────────────────────────────
		type lifespanUp struct {
			id    string
			decay float64
		}
		type manaUp struct {
			id  string
			val float64
		}
		type nodeUp struct {
			id        string
			hp        int
			destroyed bool
		}
		type vhUp struct {
			id             string
			pop            int
			happy, oppress float64
		}
		type msgIns struct{ receiver, content string }

		var lifeUps []lifespanUp
		var manaUps []manaUp
		var nodeUps []nodeUp
		var vhUps []vhUp
		var msgs []msgIns

		// ── Process each aperture ─────────────────────────────────────────────
		for _, ap := range apertures {
			timeSpeedMultiplier := 1.0
			realYearsPassed := 60.0 / 36000.0

			// Look up player data from batch
			currentLifespan := 100.0
			playerRace := ""
			isDead := false
			if pd, ok := pMap[ap.PlayerID]; ok {
				currentLifespan = pd.lifespan
				timeSpeedMultiplier = pd.ts
				playerRace = pd.race
				isDead = pd.isDead
			}

			if isDead {
				continue
			}

			dummyP := models.Player{}
			dummyP.Race = playerRace
			isImmortal := gamefx.Get(&dummyP).Immortal

			if isImmortal {
				timeSpeedMultiplier = 1.0
			} else {
				if timeSpeedMultiplier <= 0 {
					timeSpeedMultiplier = 1.0
				}
				if bonus, err := db.EquippedTimeSpeedBonus(db.DB, ap.PlayerID); err == nil {
					timeSpeedMultiplier += bonus
				}
			}

			if isImmortal {
				// Dead Aperture (Tử Khiếu) Decay: LOW grade (~3M area / ~12.5 rate) takes ~5 days
				decayRate := 0.0017 * (1.0 + realYearsPassed)
				decayArea := 400 * (1.0 + realYearsPassed)
				newRate := math.Max(0.5, ap.TimeRate-decayRate)
				newArea := math.Max(50.0, ap.AreaSize-decayArea)
				if newRate != ap.TimeRate || newArea != ap.AreaSize {
					ap.TimeRate = newRate
					ap.AreaSize = newArea
					_, _ = db.DB.Exec(`UPDATE immortal_apertures SET time_rate = $1, area_size = $2, updated_at = NOW() WHERE id = $3`, newRate, newArea, ap.ID)
				}
			} else {
				if currentLifespan <= 0.0 {
					continue
				}

				lifespanDecay := realYearsPassed * timeSpeedMultiplier
				newLifespan := math.Max(0.0, currentLifespan-lifespanDecay)
				lifeUps = append(lifeUps, lifespanUp{id: ap.PlayerID, decay: lifespanDecay})

				if newLifespan <= 0.0 {
					_, _ = db.DB.Exec(`UPDATE players SET is_dead = true WHERE id = $1`, ap.PlayerID)
					// Freeze aperture (keep per-aperture Exec: different SET clause)
					_, _ = db.DB.Exec(`
						UPDATE immortal_apertures
						SET time_rate = 1.0, aperture_hp = max_aperture_hp / 10, state = 'DAMAGED', updated_at = NOW()
						WHERE id = $1
					`, ap.ID)
					msgs = append(msgs, msgIns{receiver: ap.PlayerID, content: "Đạo hữu đã hết thọ nguyên! Tiên Khiếu suy kiệt, dòng thời gian ngưng trệ. Hãy mau kiếm Thọ Cổ để hồi sinh!"})
					logger.S.Warnf("Player %s lifespan depleted, aperture %s frozen", ap.PlayerID, ap.ID)
					continue
				}
			}

			subjectiveYearsPassed := 60.0 * ap.TimeRate * timeSpeedMultiplier / 36000.0
			newYearsPassed := ap.InternalYearsPassed + subjectiveYearsPassed
			yearsDiff := math.Floor(newYearsPassed) - math.Floor(ap.InternalYearsPassed)

			if yearsDiff > 0 {
				addedEssence := float64(yearsDiff) * float64(ap.EssenceProductionPerYear)
				manaUps = append(manaUps, manaUp{id: ap.PlayerID, val: addedEssence})

				// Ecosystem & Conflicts Tick — read from pre-loaded nodes
				nodes := nodesByAperture[ap.ID]
				if len(nodes) > 0 {
					_, _, _, overgrazing, _, _ := EvaluateEcosystem(nodes)

					for _, nd := range nodes {
						decayDamage := 0
						if CheckNodeConflict(nd, nodes) {
							decayDamage += int(float64(nd.MaxHP) * 0.10 * float64(yearsDiff))
						}
						if overgrazing && (nd.NodeType == "FARMING_ZONE" || nd.ZoneType == "khu_trong" || nd.NodeType == "MOON_ORCHID_FIELD" || nd.NodeType == "GREEN_BAMBOO_GROVE" || nd.NodeType == "SWORD_LEAF_GRASS") {
							decayDamage += int(float64(nd.MaxHP) * 0.05 * float64(yearsDiff))
						}

						if decayDamage > 0 {
							newHP := nd.HP - decayDamage
							isDestroyed := false
							if newHP <= 0 {
								newHP = 0
								isDestroyed = true
							}
							nodeUps = append(nodeUps, nodeUp{id: nd.ID, hp: newHP, destroyed: isDestroyed})

							if isDestroyed {
								msgs = append(msgs, msgIns{receiver: ap.PlayerID, content: "Điểm tài nguyên " + nd.Name + " đã bị phá hủy do bất ổn sinh thái hoặc xung đột Đạo ngân!"})
							}
						}
					}

					// Variant Human Tick — read from pre-loaded data
					vhs := vhByAperture[ap.ID]
					if len(vhs) > 0 {
						totalFoodProd := int64(0)
						for _, nd := range nodes {
							if nd.NodeType == "FARMING_ZONE" || nd.NodeType == "BREEDING_ZONE" || nd.ZoneType == "khu_trong" || nd.ZoneType == "khu_nuoi" || nd.NodeType == "MOON_ORCHID_FIELD" || nd.NodeType == "GREEN_BAMBOO_GROVE" || nd.NodeType == "SWORD_LEAF_GRASS" || nd.NodeType == "SPIRIT_DEER_PASTURE" {
								totalFoodProd += nd.ProductionAmount
							}
						}

						for _, v := range vhs {
							happinessChange := 0.0
							oppressionChange := 0.0

							foodNeeded := float64(v.population) / 100.0
							if float64(totalFoodProd) < foodNeeded {
								happinessChange -= 10.0 * float64(yearsDiff)
							} else {
								happinessChange += 2.0 * float64(yearsDiff)
							}

							switch v.laborIntensity {
							case "LOW":
								happinessChange += 5.0 * float64(yearsDiff)
								oppressionChange -= 5.0 * float64(yearsDiff)
							case "NORMAL":
							case "HIGH":
								happinessChange -= 10.0 * float64(yearsDiff)
								oppressionChange += 10.0 * float64(yearsDiff)
							case "EXPLOIT":
								happinessChange -= 25.0 * float64(yearsDiff)
								oppressionChange += 25.0 * float64(yearsDiff)
							}

							newHappiness := math.Max(0.0, math.Min(100.0, v.happiness+happinessChange))
							newOppression := math.Max(0.0, math.Min(100.0, v.oppression+oppressionChange))

							rebellionRisk := (100.0 - newHappiness + newOppression) / 4.0
							newPop := v.population
							if newHappiness < 30.0 || newOppression > 70.0 {
								if rand.Float64() < (rebellionRisk / 100.0) {
									newPop = int(float64(v.population) * 0.8)
									// Rebellion random damage — keep as direct Exec (ORDER BY random)
									_, _ = db.DB.Exec(`
										UPDATE aperture_resource_nodes
										SET hp = hp / 2
										WHERE id IN (
											SELECT id FROM aperture_resource_nodes
											WHERE aperture_id = $1 AND is_destroyed = false
											ORDER BY random() LIMIT 2
										)
									`, ap.ID)
									ap.ApertureHP = math.Max(0.0, ap.ApertureHP-ap.MaxApertureHP*0.05)
									_, _ = db.DB.Exec(`UPDATE immortal_apertures SET aperture_hp = $1 WHERE id = $2`, ap.ApertureHP, ap.ID)
									msgs = append(msgs, msgIns{receiver: ap.PlayerID, content: "Bộ tộc " + v.race + " trong Tiên Khiếu đã nổi loạn do bị áp bức dữ dội! Một số điểm tài nguyên và Tiên Khiếu đã bị hư hại."})
								}
							}

							settlementCount := 0
							targetSettlementType := ""
							switch v.race {
							case "HAIRY_MAN":
								targetSettlementType = "HAIRY_MAN_SETTLEMENT"
							case "ROCKMAN":
								targetSettlementType = "ROCKMAN_TRIBE"
							case "FEATHERMAN":
								targetSettlementType = "FEATHERMAN_CANYON"
							}
							for _, nd := range nodes {
								if nd.NodeType == targetSettlementType {
									settlementCount++
								}
							}

							maxPop := 200 * settlementCount
							if settlementCount > 0 {
								if newHappiness > 60.0 {
									newPop = int(float64(v.population) * math.Pow(1.05, float64(yearsDiff)))
								}
								if newPop > maxPop {
									newPop = maxPop
								}
							} else {
								newPop = int(float64(v.population) * math.Pow(0.90, float64(yearsDiff)))
							}

							vhUps = append(vhUps, vhUp{id: v.id, pop: newPop, happy: newHappiness, oppress: newOppression})
						}
					}
				}
			}

			// Tribulation scheduling
			pd := pMap[ap.PlayerID]
			rank := 0
			if pd != nil {
				rank = pd.rank
			}

			calcNextYear, calcNextType := models.GetNextTribulationInfo(rank, ap.RankStartYear, ap.InternalYearsPassed)

			triggerYear := guardNextTribulationYear(calcNextYear, ap.InternalYearsPassed)
			storedYear := guardNextTribulationYear(ap.NextTribulationYear, ap.InternalYearsPassed)
			if storedYear >= ap.InternalYearsPassed || (ap.InternalYearsPassed-storedYear < 10.0) {
				if storedYear < triggerYear {
					triggerYear = storedYear
					if scheduledType := models.GetTribulationTypeAtYear(rank, ap.RankStartYear, storedYear); scheduledType != "" {
						calcNextType = scheduledType
					}
				} else if math.Abs(storedYear-triggerYear) < 0.001 {
					if scheduledType := models.GetTribulationTypeAtYear(rank, ap.RankStartYear, triggerYear); scheduledType != "" {
						calcNextType = scheduledType
					}
				}
			}

			if newYearsPassed >= triggerYear {
				if !isPlayerOnline(ap.PlayerID) {
					_, _ = db.DB.Exec(`
						UPDATE immortal_apertures
						SET state = 'TRIBULATION_PENDING',
						    tribulation_pending_at = COALESCE(tribulation_pending_at, NOW()),
						    internal_years_passed = $1, next_tribulation_year = $2, updated_at = NOW()
						WHERE id = $3
					`, newYearsPassed, triggerYear, ap.ID)
					continue
				}

				rewardMult := 1.0
				if ap.TribulationPendingAt.Valid && time.Since(ap.TribulationPendingAt.Time) > tribulationPenaltyAfter {
					rewardMult = 0.5
				}

				_, _ = db.DB.Exec(`
					UPDATE immortal_apertures
					SET internal_years_passed = $1, tribulation_pending_at = NULL, updated_at = NOW()
					WHERE id = $2
				`, newYearsPassed, ap.ID)

				spawnType := calcNextType
				err := spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.EarthCalamityCount+ap.HeavenlyTribulationCount+ap.GrandTribulationCount+ap.MyriadTribulationCount+ap.ChaosCalamityCount+1, ap.ApertureHP, newYearsPassed, spawnType, rewardMult)
				if err != nil {
					logger.S.Errorf("Failed to spawn tribulation event for player %s: %v", ap.PlayerID, err)
				}
			} else {
				_, _ = db.DB.Exec(`
					UPDATE immortal_apertures
					SET internal_years_passed = $1, next_tribulation_year = $2, updated_at = NOW()
					WHERE id = $3
				`, newYearsPassed, triggerYear, ap.ID)
			}
		}

		// ── Batch-write: all accumulated changes (single transaction) ─────────
		if len(lifeUps)+len(manaUps)+len(nodeUps)+len(vhUps)+len(msgs) > 0 {
			if tx, txErr := db.DB.Begin(); txErr == nil {
				defer tx.Rollback()

				// Batch UPDATE player lifespans
				if len(lifeUps) > 0 {
					ids := make([]string, len(lifeUps))
					vals := make([]float64, len(lifeUps))
					for i, u := range lifeUps {
						ids[i] = u.id
						vals[i] = u.decay
					}
					_, _ = tx.Exec(`UPDATE players AS p SET lifespan = GREATEST(0.0, p.lifespan - v.decay) FROM (SELECT unnest($1::text[]) AS id, unnest($2::float8[]) AS decay) AS v WHERE p.id = v.id`, pq.Array(ids), pq.Array(vals))
				}

				// Batch UPDATE player mana
				if len(manaUps) > 0 {
					ids := make([]string, 0, len(manaUps))
					vals := make([]float64, 0, len(manaUps))
					for _, u := range manaUps {
						if pd, ok := pMap[u.id]; ok {
							dummyP := models.Player{}
							dummyP.Race = pd.race
							if gamefx.Get(&dummyP).DisableManaRegen {
								continue
							}
						}
						ids = append(ids, u.id)
						vals = append(vals, u.val)
					}
					if len(ids) > 0 {
						_, _ = tx.Exec(`UPDATE players AS p SET current_mana = LEAST(p.max_mana, p.current_mana + v.val), last_mana_update = NOW() FROM (SELECT unnest($1::text[]) AS id, unnest($2::float8[]) AS val) AS v WHERE p.id = v.id`, pq.Array(ids), pq.Array(vals))
					}
				}

				// Batch UPDATE node HP
				if len(nodeUps) > 0 {
					ids := make([]string, len(nodeUps))
					hps := make([]int, len(nodeUps))
					dest := make([]bool, len(nodeUps))
					for i, u := range nodeUps {
						ids[i] = u.id
						hps[i] = u.hp
						dest[i] = u.destroyed
					}
					_, _ = tx.Exec(`UPDATE aperture_resource_nodes AS n SET hp = v.hp, is_destroyed = v.destroyed, updated_at = NOW() FROM (SELECT unnest($1::text[]) AS id, unnest($2::int[]) AS hp, unnest($3::bool[]) AS destroyed) AS v WHERE n.id = v.id`, pq.Array(ids), pq.Array(hps), pq.Array(dest))
				}

				// Batch UPDATE variant humans
				if len(vhUps) > 0 {
					ids := make([]string, len(vhUps))
					pops := make([]int, len(vhUps))
					happys := make([]float64, len(vhUps))
					oppresses := make([]float64, len(vhUps))
					for i, u := range vhUps {
						ids[i] = u.id
						pops[i] = u.pop
						happys[i] = u.happy
						oppresses[i] = u.oppress
					}
					_, _ = tx.Exec(`UPDATE aperture_variant_humans AS v SET population = d.pop, happiness = d.happy, oppression = d.oppress, updated_at = NOW() FROM (SELECT unnest($1::text[]) AS id, unnest($2::int[]) AS pop, unnest($3::float8[]) AS happy, unnest($4::float8[]) AS oppress) AS d WHERE v.id = d.id`, pq.Array(ids), pq.Array(pops), pq.Array(happys), pq.Array(oppresses))
				}

				// Batch INSERT messages
				if len(msgs) > 0 {
					receivers := make([]string, len(msgs))
					contents := make([]string, len(msgs))
					for i, m := range msgs {
						receivers[i] = m.receiver
						contents[i] = m.content
					}
					_, _ = tx.Exec(`INSERT INTO messages (sender_id, receiver_id, content, channel) SELECT '11111111-1111-1111-1111-111111111111', v.receiver, v.content, 'private' FROM (SELECT unnest($1::text[]) AS receiver, unnest($2::text[]) AS content) AS v`, pq.Array(receivers), pq.Array(contents))
				}

				_ = tx.Commit()
			}
		}

	}

	// 2. Check and auto-resolve pending active tribulations that exceeded 24 hours
	autoRows, err := db.DB.Query(`
		SELECT id, player_id
		FROM tribulation_events
		WHERE status = 'ACTIVE' AND started_at < NOW() - INTERVAL '24 hours'
	`)
	if err == nil {
		type EventResolve struct {
			EventID  string
			PlayerID string
		}
		toResolve := []EventResolve{}
		for autoRows.Next() {
			var ev EventResolve
			if err := autoRows.Scan(&ev.EventID, &ev.PlayerID); err == nil {
				toResolve = append(toResolve, ev)
			}
		}
		autoRows.Close()

		for _, r := range toResolve {
			logger.S.Infof("Auto-resolving expired tribulation event %s for player %s", r.EventID, r.PlayerID)
			autoResolveEvent(r.EventID, r.PlayerID)
		}
	}

	return nil
}

func spawnTribulationEventWorker(apertureID string, playerID string, tribCount int, currentHP float64, newYearsPassed float64, tribulationType string, rewardMult float64) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Pick a random path for the Calamity, prioritizing the player's Vital Gu path if they have one
	var vitalGuID string
	var vitalGuRank int
	errVital := tx.QueryRow(`SELECT gu_id, gu_rank FROM player_gu WHERE player_id = $1 AND is_vital = true LIMIT 1`, playerID).Scan(&vitalGuID, &vitalGuRank)

	supportedPaths := models.AllGuPaths()
	matchedPaths := []string{}
	if errVital == nil && vitalGuID != "" {
		if guItem, found := models.GetGu(vitalGuID, vitalGuRank); found {
			for _, p := range guItem.Path {
				normalizedP := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(p), " path"))
				for _, sp := range supportedPaths {
					if normalizedP == sp {
						matchedPaths = append(matchedPaths, sp)
					}
				}
			}
		}
	}

	var path string
	if len(matchedPaths) > 0 && rand.Float64() < 0.7 {
		path = matchedPaths[rand.Intn(len(matchedPaths))]
	} else {
		path = supportedPaths[rand.Intn(len(supportedPaths))]
	}

	var rank int
	err = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&rank)
	if err != nil {
		return err
	}

	var daoMarksRaw []byte
	err = tx.QueryRow(`SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&daoMarksRaw)
	if err != nil {
		return err
	}

	var daoMarks map[string]int
	if len(daoMarksRaw) > 0 {
		_ = json.Unmarshal(daoMarksRaw, &daoMarks)
	}

	maxDao := 0
	for _, val := range daoMarks {
		if val > maxDao {
			maxDao = val
		}
	}

	expectedDao := 40000.0
	switch rank {
	case 6:
		expectedDao = 40000.0
	case 7:
		expectedDao = 220000.0
	case 8:
		expectedDao = 1400000.0
	case 9:
		expectedDao = 8500000.0
	default:
		if rank < 6 {
			expectedDao = 1000.0
		} else {
			expectedDao = 8500000.0
		}
	}

	daoBalancingMult := 1.0
	if expectedDao > 0 {
		ratio := float64(maxDao) / expectedDao
		if ratio < 0.25 {
			ratio = 0.25
		}
		if rank < 9 && ratio > 3.0 {
			ratio = 3.0
		}
		daoBalancingMult = ratio
	}

	difficulty := 1 + tribCount
	rewardMarks := models.CalculateTribulationReward(rank, tribulationType, difficulty)
	if rewardMult <= 0 {
		rewardMult = 1.0
	}
	rewardMarks = int(float64(rewardMarks) * rewardMult)

	// Insert Tribulation Event
	eventID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO tribulation_events (id, aperture_id, player_id, tribulation_type, path, difficulty, status, aperture_hp_before, reward_dao_marks, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE', $7, $8, NOW())
	`, eventID, apertureID, playerID, tribulationType, path, difficulty, currentHP, rewardMarks)
	if err != nil {
		return err
	}

	// Spawn Enemies
	bossTemplateID := "monster_earth_calamity_boss"
	switch tribulationType {
	case "EARTH_CALAMITY":
		bossTemplateID = "monster_earth_calamity_boss"
	case "HEAVENLY_TRIBULATION":
		bossTemplateID = "monster_heaven_tribulation_boss"
	case "GRAND_TRIBULATION":
		bossTemplateID = "monster_grand_calamity_boss"
	case "MYRIAD_TRIBULATION":
		bossTemplateID = "monster_myriad_tribulation_boss"
	case "CHAOS_CALAMITY":
		bossTemplateID = "monster_chaos_calamity_boss"
	}

	bossRank, bossSubRank := models.GetTribulationRankAndSubRank(rank, tribulationType)

	var boss models.MonsterTemplate
	var bossGu []byte
	err = tx.QueryRow(`
		SELECT id, name, type, max_hp, base_attack, base_defense, gu_inventory, ai_profile, loot_table_id, description
		FROM monster_templates WHERE id = $1
	`, bossTemplateID).Scan(
		&boss.ID, &boss.Name, &boss.Type,
		&boss.MaxHP, &boss.BaseAttack, &boss.BaseDefense,
		&bossGu, &boss.AIProfile, &boss.LootTableID, &boss.Description,
	)
	if err != nil {
		return fmt.Errorf("load tribulation boss template %s from database: %w", bossTemplateID, err)
	}

	// Apply difficulty scaling
	diffMultiplier := 1.0 + float64(difficulty)*0.2
	scaledHP, scaledAtk, scaledDef := models.ScaleMonsterStats(boss.MaxHP, boss.BaseAttack, boss.BaseDefense, bossRank, bossSubRank, boss.Type)
	bossHP := int(scaledHP * diffMultiplier * daoBalancingMult)
	bossAtk := int(scaledAtk * diffMultiplier * daoBalancingMult)
	bossDef := int(scaledDef * diffMultiplier * daoBalancingMult)

	_, err = tx.Exec(`
		INSERT INTO tribulation_enemies (event_id, name, enemy_type, path, hp, max_hp, attack, defense, x, y, is_boss, is_dead, rank, sub_rank)
		VALUES ($1, $2, 'BOSS', $3, $4, $4, $5, $6, $7, $8, true, false, $9, $10)
	`, eventID, fmt.Sprintf("%s (%s Path)", boss.Name, strings.Title(path)), path, bossHP, bossAtk, bossDef, rand.Intn(20), rand.Intn(20), bossRank, bossSubRank)
	if err != nil {
		return err
	}

	// Spawn Minions
	var minion models.MonsterTemplate
	var minionGu []byte
	err = tx.QueryRow(`
		SELECT id, name, type, max_hp, base_attack, base_defense, gu_inventory, ai_profile, loot_table_id, description
		FROM monster_templates WHERE id = 'monster_tribulation_beast'
	`).Scan(
		&minion.ID, &minion.Name, &minion.Type,
		&minion.MaxHP, &minion.BaseAttack, &minion.BaseDefense,
		&minionGu, &minion.AIProfile, &minion.LootTableID, &minion.Description,
	)
	if err != nil {
		return fmt.Errorf("load tribulation minion template monster_tribulation_beast from database: %w", err)
	}

	minionCount := 1 + rand.Intn(3)
	minionSubRank := 1
	for i := 0; i < minionCount; i++ {
		scaledHPm, scaledAtkm, scaledDefm := models.ScaleMonsterStats(minion.MaxHP, minion.BaseAttack, minion.BaseDefense, bossRank, minionSubRank, minion.Type)
		mHP := int(scaledHPm * diffMultiplier * daoBalancingMult)
		mAtk := int(scaledAtkm * diffMultiplier * daoBalancingMult)
		mDef := int(scaledDefm * diffMultiplier * daoBalancingMult)

		_, err = tx.Exec(`
			INSERT INTO tribulation_enemies (event_id, name, enemy_type, path, hp, max_hp, attack, defense, x, y, is_dead, rank, sub_rank)
			VALUES ($1, $2, 'MINION', $3, $4, $4, $5, $6, $7, $8, false, $9, $10)
		`, eventID, fmt.Sprintf("%s #%d", minion.Name, i+1), path, mHP, mAtk, mDef, rand.Intn(20), rand.Intn(20), bossRank, minionSubRank)
		if err != nil {
			return err
		}
	}

	// Set aperture status to active tribulation and update internal_years_passed
	_, err = tx.Exec(`
		UPDATE immortal_apertures
		SET state = 'TRIBULATION_ACTIVE', internal_years_passed = $1, updated_at = NOW()
		WHERE id = $2
	`, newYearsPassed, apertureID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func autoResolveEvent(eventID string, playerID string) {
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var ev models.TribulationEvent
	err = tx.QueryRow(`
		SELECT id, aperture_id, player_id, tribulation_type, path, difficulty, started_at, aperture_hp_before
		FROM tribulation_events WHERE id = $1 AND status = 'ACTIVE'
	`, eventID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.Difficulty, &ev.StartedAt, &ev.ApertureHPBefore)
	if err != nil {
		return
	}

	// Fetch current counters from aperture
	var ap models.ImmortalAperture
	_ = tx.QueryRow(`
		SELECT id, aperture_hp, max_aperture_hp, next_tribulation_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, internal_years_passed
		FROM immortal_apertures WHERE id = $1
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.NextTribulationYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount,
		&ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.InternalYearsPassed)
	if err != nil {
		return
	}

	var playerRank int
	_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&playerRank)

	// Auto-resolve counts as a failure — counters are only incremented on
	// success, but the next year is still pushed to the next milestone.
	nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, ap.InternalYearsPassed)
	nextYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)

	var feathermanPop int
	_ = tx.QueryRow(`SELECT population FROM aperture_variant_humans WHERE aperture_id = $1 AND race = 'FEATHERMAN'`, ev.ApertureID).Scan(&feathermanPop)
	reduction := float64(feathermanPop) * 0.001
	if reduction > 0.5 {
		reduction = 0.5
	}

	hpFactor := 0.1 + 0.4*reduction
	newHP := int(float64(ev.ApertureHPBefore) * hpFactor)
	if newHP < 100 {
		newHP = 100
	}

	_, _ = tx.Exec(`
		UPDATE immortal_apertures 
		SET state = 'DAMAGED', next_tribulation_year = $1, aperture_hp = $2,
			earth_calamity_count = $3, heavenly_tribulation_count = $4, grand_tribulation_count = $5,
			myriad_tribulation_count = $6, chaos_calamity_count = $7
		WHERE id = $8
	`, nextYear, newHP,
		ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount,
		ap.MyriadTribulationCount, ap.ChaosCalamityCount, ev.ApertureID)

	_, _ = tx.Exec(`
		UPDATE tribulation_events
		SET status = 'FAILED', aperture_hp_after = $1, ended_at = NOW(), reward_dao_marks = 0
		WHERE id = $2
	`, newHP, ev.ID)

	_, _ = tx.Exec(`
		UPDATE aperture_resource_nodes 
		SET is_destroyed = true 
		WHERE id IN (
			SELECT id FROM aperture_resource_nodes 
			WHERE aperture_id = $1 AND is_destroyed = false 
			ORDER BY random() LIMIT 2
		)
	`, ev.ApertureID)

	_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

	_ = tx.Commit()
}

type ActiveTribulationEvent struct {
	EventID         string
	ApertureID      string
	PlayerID        string
	TribulationType string
	StartedAt       time.Time
}

// ProcessActiveTribulationGrid moves tribulation enemies towards resource nodes and attacks them.
// Position updates and watchtower damage are processed per event in isolated transactions with timeouts.
func ProcessActiveTribulationGrid() error {
	// 1. Fetch all active tribulation events outside of long-held transactions
	rows, err := db.DB.Query(`
		SELECT id, aperture_id, player_id, tribulation_type, started_at
		FROM tribulation_events 
		WHERE status = 'ACTIVE'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []ActiveTribulationEvent
	for rows.Next() {
		var ev ActiveTribulationEvent
		if err := rows.Scan(&ev.EventID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.StartedAt); err == nil {
			events = append(events, ev)
		}
	}
	rows.Close()

	// 2. Process each event in its own transaction so a single failing or slow event doesn't block the rest
	for _, ev := range events {
		if err := processActiveTribulationEvent(ev); err != nil {
			logger.S.Errorf("Error processing tribulation event %s for player %s: %v", ev.EventID, ev.PlayerID, err)
		}
	}
	return nil
}

func processActiveTribulationEvent(ev ActiveTribulationEvent) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SET LOCAL statement_timeout = '30000'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '10000'`); err != nil {
		return err
	}

	// Batch accumulators – reset per event.
	type posUpdate struct {
		ID string
		X  int
		Y  int
	}
	type nodeDmgAcc struct {
		HP          int
		IsDestroyed bool
	}

	// Freeze the tribulation while the player is offline: enemies hold their
	// position and never damage nodes. Bump started_at so the 24h auto-resolve
	// window only counts online time.
	if !isPlayerOnline(ev.PlayerID) {
		_, _ = tx.Exec(`UPDATE tribulation_events SET started_at = NOW() WHERE id = $1`, ev.EventID)
		return tx.Commit()
	}

	// Grace period after spawn/return: enemies stay idle for a short window.
	if time.Since(ev.StartedAt) < tribulationGracePeriod {
		return tx.Commit()
	}

	// Fixed node damage per hit, based on tribulation type
	nodeDmgPerHit := 1 // EARTH_CALAMITY
	switch ev.TribulationType {
	case "HEAVENLY_TRIBULATION":
		nodeDmgPerHit = 2
	case "GRAND_TRIBULATION":
		nodeDmgPerHit = 4
	case "MYRIAD_TRIBULATION":
		nodeDmgPerHit = 8
	case "CHAOS_CALAMITY":
		nodeDmgPerHit = 16
	}

	// Fetch living enemies
	enRows, err := tx.Query(`
		SELECT id, name, enemy_type, x, y 
		FROM tribulation_enemies 
		WHERE event_id = $1 AND is_dead = false
	`, ev.EventID)
	if err != nil {
		return fmt.Errorf("failed to query tribulation enemies: %w", err)
	}

	type TEnemy struct {
		ID        string
		Name      string
		EnemyType string
		X         int
		Y         int
		NodeDmg   int
	}
	var enemies []TEnemy
	for enRows.Next() {
		var en TEnemy
		if err := enRows.Scan(&en.ID, &en.Name, &en.EnemyType, &en.X, &en.Y); err != nil {
			enRows.Close()
			return fmt.Errorf("failed to scan enemy row: %w", err)
		}
		// Minions chip away at the mine; the boss deals double the minion damage
		en.NodeDmg = nodeDmgPerHit
		if en.EnemyType == "BOSS" {
			en.NodeDmg = nodeDmgPerHit * 2
		}
		enemies = append(enemies, en)
	}
	enRows.Close()

	if len(enemies) == 0 {
		return tx.Commit()
	}

	// Fetch non-destroyed resource nodes
	nodeRows, err := tx.Query(`
		SELECT id, name, hp, x, y,
			COALESCE(node_category, 'RESOURCE'), COALESCE(blocks_movement, false),
			COALESCE(defense_power, 0), COALESCE(attack_range, 0),
			COALESCE(hp_bonus, 0), COALESCE(damage_reduction, 0)
		FROM aperture_resource_nodes 
		WHERE aperture_id = $1 AND is_destroyed = false
	`, ev.ApertureID)
	if err != nil {
		return fmt.Errorf("failed to query aperture resource nodes: %w", err)
	}

	type TNode struct {
		ID              string
		Name            string
		HP              int
		X               int
		Y               int
		NodeCategory    string
		BlocksMovement  bool
		DefensePower    int
		AttackRange     int
		HPBonus         int
		DamageReduction int
	}
	var nodes []TNode
	for nodeRows.Next() {
		var nd TNode
		if err := nodeRows.Scan(&nd.ID, &nd.Name, &nd.HP, &nd.X, &nd.Y,
			&nd.NodeCategory, &nd.BlocksMovement,
			&nd.DefensePower, &nd.AttackRange,
			&nd.HPBonus, &nd.DamageReduction); err != nil {
			nodeRows.Close()
			return fmt.Errorf("failed to scan resource node row: %w", err)
		}
		nodes = append(nodes, nd)
	}
	nodeRows.Close()

	// Accumulate batch updates for this event
	var pendingPos []posUpdate
	pendingNodeDmg := make(map[string]*nodeDmgAcc)
	type pendingMsg struct {
		ReceiverID string
		Content    string
	}
	var pendingMsgs []pendingMsg
	// Track tower damage to enemies (enemyID -> total damage)
	pendingTowerDmg := make(map[string]int)

	if len(nodes) == 0 {
		// No nodes to attack: enemies wander around aimlessly
		for _, en := range enemies {
			dx, dy := 0, 0
			switch rand.Intn(4) {
			case 0:
				dx = 1
			case 1:
				dx = -1
			case 2:
				dy = 1
			case 3:
				dy = -1
			}
			newX := en.X + dx
			if newX < 0 {
				newX = 0
			} else if newX > 19 {
				newX = 19
			}
			newY := en.Y + dy
			if newY < 0 {
				newY = 0
			} else if newY > 19 {
				newY = 19
			}
			pendingPos = append(pendingPos, posUpdate{ID: en.ID, X: newX, Y: newY})
		}
	} else {
		// Process each enemy
		for _, en := range enemies {
			// Find nearest non-wall node (prefer defense nodes, then resource)
			var nearestNode *TNode
			minDist := 999999
			for i := range nodes {
				nd := &nodes[i]
				dx := nd.X - en.X
				if dx < 0 {
					dx = -dx
				}
				dy := nd.Y - en.Y
				if dy < 0 {
					dy = -dy
				}
				dist := dx + dy
				if dist < minDist {
					minDist = dist
					nearestNode = nd
				} else if dist == minDist && nearestNode != nil {
					// Tiebreak: prefer defense nodes over resource nodes
					if nd.NodeCategory == "DEFENSE" && nearestNode.NodeCategory != "DEFENSE" {
						nearestNode = nd
					}
				}
			}

			if nearestNode == nil {
				continue
			}

			dx := nearestNode.X - en.X
			dy := nearestNode.Y - en.Y
			absDx := dx
			if absDx < 0 {
				absDx = -absDx
			}
			absDy := dy
			if absDy < 0 {
				absDy = -absDy
			}

			if absDx <= 1 && absDy <= 1 {
				// Enemy is adjacent or on top of the node, attack!
				effectiveDmg := en.NodeDmg
				totalDamageReduction := 0
				totalHPBonus := 0

				for i := range nodes {
					buffNode := &nodes[i]
					if buffNode.ID == nearestNode.ID || buffNode.HP <= 0 {
						continue
					}
					bdx := buffNode.X - nearestNode.X
					if bdx < 0 {
						bdx = -bdx
					}
					bdy := buffNode.Y - nearestNode.Y
					if bdy < 0 {
						bdy = -bdy
					}
					bdist := bdx + bdy

					if buffNode.HPBonus > 0 && buffNode.AttackRange > 0 && bdist <= buffNode.AttackRange {
						totalHPBonus += buffNode.HPBonus
					}
					if buffNode.DamageReduction > 0 && buffNode.AttackRange == 0 {
						totalDamageReduction += buffNode.DamageReduction
					}
					if buffNode.DamageReduction > 0 && buffNode.AttackRange > 0 && bdist <= buffNode.AttackRange {
						totalDamageReduction += buffNode.DamageReduction
					}
				}

				if totalDamageReduction > 80 {
					totalDamageReduction = 80
				}
				effectiveDmg = en.NodeDmg * (100 - totalDamageReduction) / 100
				if effectiveDmg < 1 {
					effectiveDmg = 1
				}

				nearestNode.HP -= effectiveDmg
				isDestroyed := false
				if nearestNode.HP <= 0 {
					nearestNode.HP = 0
					isDestroyed = true
				}
				pendingNodeDmg[nearestNode.ID] = &nodeDmgAcc{HP: nearestNode.HP, IsDestroyed: isDestroyed}

				if isDestroyed {
					pendingMsgs = append(pendingMsgs, pendingMsg{
						ReceiverID: ev.PlayerID,
						Content:    fmt.Sprintf("Tai kiếp đã phá hủy hoàn toàn %s!", nearestNode.Name),
					})
					var remainingNodes []TNode
					for _, nd := range nodes {
						if nd.ID != nearestNode.ID {
							remainingNodes = append(remainingNodes, nd)
						}
					}
					nodes = remainingNodes
				}
			} else {
				// Move towards the node (allows diagonal movement)
				targetX := en.X
				if dx > 0 {
					targetX = en.X + 1
				} else if dx < 0 {
					targetX = en.X - 1
				}
				targetY := en.Y
				if dy > 0 {
					targetY = en.Y + 1
				} else if dy < 0 {
					targetY = en.Y - 1
				}

				var wallNode *TNode
				for i := range nodes {
					nd := &nodes[i]
					if nd.BlocksMovement && nd.X == targetX && nd.Y == targetY && nd.HP > 0 {
						wallNode = nd
						break
					}
				}

				if wallNode != nil {
					nearestNode = wallNode
					effectiveDmg := en.NodeDmg
					totalDamageReduction := 0
					for i := range nodes {
						buffNode := &nodes[i]
						if buffNode.ID == wallNode.ID || buffNode.HP <= 0 {
							continue
						}
						bdx := buffNode.X - wallNode.X
						if bdx < 0 {
							bdx = -bdx
						}
						bdy := buffNode.Y - wallNode.Y
						if bdy < 0 {
							bdy = -bdy
						}
						bdist := bdx + bdy
						if buffNode.DamageReduction > 0 && buffNode.AttackRange == 0 {
							totalDamageReduction += buffNode.DamageReduction
						}
						if buffNode.DamageReduction > 0 && buffNode.AttackRange > 0 && bdist <= buffNode.AttackRange {
							totalDamageReduction += buffNode.DamageReduction
						}
					}
					if totalDamageReduction > 80 {
						totalDamageReduction = 80
					}
					effectiveDmg = en.NodeDmg * (100 - totalDamageReduction) / 100
					if effectiveDmg < 1 {
						effectiveDmg = 1
					}

					nearestNode.HP -= effectiveDmg
					isDestroyed := false
					if nearestNode.HP <= 0 {
						nearestNode.HP = 0
						isDestroyed = true
					}
					pendingNodeDmg[nearestNode.ID] = &nodeDmgAcc{HP: nearestNode.HP, IsDestroyed: isDestroyed}

					if isDestroyed {
						pendingMsgs = append(pendingMsgs, pendingMsg{
							ReceiverID: ev.PlayerID,
							Content:    "Tai kiếp đã phá hủy Tường Thành!",
						})
						var remainingNodes []TNode
						for _, nd := range nodes {
							if nd.ID != wallNode.ID {
								remainingNodes = append(remainingNodes, nd)
							}
						}
						nodes = remainingNodes
					}
				} else {
					if dx > 0 {
						en.X++
					} else if dx < 0 {
						en.X--
					}
					if dy > 0 {
						en.Y++
					} else if dy < 0 {
						en.Y--
					}
					pendingPos = append(pendingPos, posUpdate{ID: en.ID, X: en.X, Y: en.Y})
				}
			}
		}

		// Watchtower attacks: towers shoot at enemies in range
		for i := range nodes {
			tower := &nodes[i]
			if tower.NodeCategory != "DEFENSE" || tower.DefensePower <= 0 || tower.AttackRange <= 0 || tower.HP <= 0 {
				continue
			}
			for _, en := range enemies {
				wdx := tower.X - en.X
				if wdx < 0 {
					wdx = -wdx
				}
				wdy := tower.Y - en.Y
				if wdy < 0 {
					wdy = -wdy
				}
				wdist := wdx + wdy
				if wdist <= tower.AttackRange {
					pendingTowerDmg[en.ID] += tower.DefensePower
				}
			}
		}
	}

	// --- Batch: update enemy positions ---
	if len(pendingPos) > 0 {
		ids := make([]string, len(pendingPos))
		xs := make([]int, len(pendingPos))
		ys := make([]int, len(pendingPos))
		for i, p := range pendingPos {
			ids[i] = p.ID
			xs[i] = p.X
			ys[i] = p.Y
		}
		_, err = tx.Exec(`
			UPDATE tribulation_enemies AS e
			SET x = v.x, y = v.y, updated_at = NOW()
			FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS x, unnest($3::int[]) AS y) AS v
			WHERE e.id = v.id
		`, pq.Array(ids), pq.Array(xs), pq.Array(ys))
		if err != nil {
			return fmt.Errorf("batch update enemy positions: %w", err)
		}
	}

	// --- Batch: update resource node HPs ---
	if len(pendingNodeDmg) > 0 {
		ndIDs := make([]string, 0, len(pendingNodeDmg))
		ndHPs := make([]int, 0, len(pendingNodeDmg))
		ndDestroyed := make([]bool, 0, len(pendingNodeDmg))
		for id, acc := range pendingNodeDmg {
			ndIDs = append(ndIDs, id)
			ndHPs = append(ndHPs, acc.HP)
			ndDestroyed = append(ndDestroyed, acc.IsDestroyed)
		}
		_, err = tx.Exec(`
			UPDATE aperture_resource_nodes AS n
			SET hp = v.hp, is_destroyed = v.is_destroyed, updated_at = NOW()
			FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS hp, unnest($3::bool[]) AS is_destroyed) AS v
			WHERE n.id = v.id
		`, pq.Array(ndIDs), pq.Array(ndHPs), pq.Array(ndDestroyed))
		if err != nil {
			return fmt.Errorf("batch update node HP: %w", err)
		}
	}

	// --- Batch: apply watchtower damage to enemies ---
	if len(pendingTowerDmg) > 0 {
		tIDs := make([]string, 0, len(pendingTowerDmg))
		tDmg := make([]int, 0, len(pendingTowerDmg))
		for enemyID, dmg := range pendingTowerDmg {
			tIDs = append(tIDs, enemyID)
			tDmg = append(tDmg, dmg)
		}
		_, err = tx.Exec(`
			UPDATE tribulation_enemies AS e
			SET hp = GREATEST(e.hp - v.dmg, 0),
			    is_dead = CASE WHEN e.hp - v.dmg <= 0 THEN true ELSE false END,
			    updated_at = NOW()
			FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS dmg) AS v
			WHERE e.id = v.id
		`, pq.Array(tIDs), pq.Array(tDmg))
		if err != nil {
			return fmt.Errorf("batch watchtower damage: %w", err)
		}
	}

	// --- Batch: insert system messages ---
	if len(pendingMsgs) > 0 {
		for _, m := range pendingMsgs {
			_, err = tx.Exec(`
				INSERT INTO messages (sender_id, receiver_id, content, channel)
				VALUES ('11111111-1111-1111-1111-111111111111', $1, $2, 'private')
			`, m.ReceiverID, m.Content)
			if err != nil {
				return fmt.Errorf("insert tribulation message: %w", err)
			}
		}
	}

	return tx.Commit()
}

func RegenerateImmortalApertures() error {
	// Cheap guard: skip the UPDATE entirely when no aperture needs healing.
	var needsHealing bool
	if err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM immortal_apertures WHERE state != 'DESTROYED' AND aperture_hp < max_aperture_hp)`).Scan(&needsHealing); err != nil {
		return err
	}
	if !needsHealing {
		return nil
	}

	// Heal aperture_hp by 0.5% of max_aperture_hp per minute (minimum 10 HP per minute)
	// If state is DAMAGED and aperture_hp reaches max_aperture_hp, set state back to STABLE
	_, err := db.DB.Exec(`
			UPDATE immortal_apertures
			SET 
				aperture_hp = LEAST(max_aperture_hp, aperture_hp + GREATEST(10, max_aperture_hp / 200)),
				state = CASE 
					WHEN (aperture_hp + GREATEST(10, max_aperture_hp / 200)) >= max_aperture_hp THEN
						CASE 
							WHEN NOT EXISTS (SELECT 1 FROM tribulation_events WHERE aperture_id = immortal_apertures.id AND status = 'ACTIVE')
							     AND NOT EXISTS (SELECT 1 FROM tribulation_enemies te JOIN tribulation_events ev ON te.event_id = ev.id WHERE ev.aperture_id = immortal_apertures.id AND te.is_dead = false)
							THEN 'STABLE'
							ELSE 'TRIBULATION_ACTIVE'
						END
					ELSE state
				END,
				updated_at = NOW()
			WHERE state != 'DESTROYED' AND aperture_hp < max_aperture_hp
		`)
	return err
}

func StartTribulationWorker(ctx context.Context) {
	ticker60 := time.NewTicker(60 * time.Second)
	ticker15 := time.NewTicker(15 * time.Second)

	go func() {
		defer ticker60.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.S.Info("Tribulation worker (60s) stopped")
				return
			case <-ticker60.C:
				err := CheckAndTriggerTribulations()
				if err != nil {
					logger.S.Errorf("Tribulation background worker error: %v", err)
				}
				errRegen := RegenerateImmortalApertures()
				if errRegen != nil {
					logger.S.Errorf("Aperture HP regeneration background worker error: %v", errRegen)
				}
			}
		}
	}()

	go func() {
		defer ticker15.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.S.Info("Tribulation worker (15s) stopped")
				return
			case <-ticker15.C:
				err := ProcessActiveTribulationGrid()
				if err != nil {
					logger.S.Errorf("Tribulation grid worker error: %v", err)
				}
			}
		}
	}()

	logger.S.Info("Tribulation and internal year worker started successfully (Interval: 60s)")
}
