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
				blockedMsg := fmt.Sprintf(" Chưa đủ điều kiện đột phá lên %s: cần ít nhất 1 lưu phái đạt %s (%d điểm).", models.GetRankName(newRank), gamefx.PathMasteryLevelName(reqLevel), gamefx.PathMasteryPointsForLevel(reqLevel))
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
				breakthroughMsg = fmt.Sprintf(" Đột phá! Đạo hữu đã tiến vào %s!", models.GetRankName(newRank))

				if newRank == 7 || newRank == 8 || newRank == 9 {
					_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Chúc mừng đạo hữu %s đã đột phá thành công lên %s!", username, models.GetRankName(newRank)))
					auditlog.Log(tx, auditlog.Entry{
						PlayerID:       &playerID,
						Action:         "player.breakthrough_success",
						EntityType:     "player",
						EntityID:       playerID.String(),
						RequestSummary: fmt.Sprintf("Thăng rank %d (%s) thành công", newRank, models.GetRankName(newRank)),
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
