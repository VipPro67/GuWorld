package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
	var hpAfter sql.NullInt32
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
		h := int(hpAfter.Int32)
		ev.ApertureHPAfter = &h
	}
	if endedAt.Valid {
		ev.EndedAt = &endedAt.Time
	}

	// 2. Fetch enemies
	rows, err := tx.Query(`
		SELECT id, event_id, name, enemy_type, path, hp, max_hp, attack, defense, target_node_id, x, y, is_boss, is_dead
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
		err = rows.Scan(&en.ID, &en.EventID, &en.Name, &en.EnemyType, &en.Path, &en.HP, &en.MaxHP, &en.Attack, &en.Defense, &targetID, &en.X, &en.Y, &en.IsBoss, &en.IsDead)
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

type AttackEnemyRequest struct {
	EnemyID string `json:"enemy_id" binding:"required"`
	GuID    string `json:"gu_id" binding:"required"` // Gu bug ID used for attack
}

// AttackTribulationEnemy allows the player to use a Gu to damage a tribulation boss/enemy
func AttackTribulationEnemy(c *gin.Context) {
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

	var req AttackEnemyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
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

	// 2. Fetch Player stats
	var player models.Player
	err = tx.QueryRow(`
		SELECT id, username, current_mana, max_mana, hp, base_hp, primeval_stones, gu_inventory
		FROM players WHERE id = $1
	`, playerID).Scan(&player.ID, &player.Username, &player.CurrentMana, &player.MaxMana, &player.HP, &player.BaseHP, &player.PrimevalStones, &player.GuInventory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin cổ sư"})
		return
	}

	// Verify player has the Gu bug and it's active
	var foundGu *models.GuInstance
	for i := range player.GuInventory {
		if player.GuInventory[i].GuID == req.GuID && player.GuInventory[i].Status == "ACTIVE" {
			foundGu = &player.GuInventory[i]
			break
		}
	}

	if foundGu == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không sở hữu hoặc Cổ trùng này không ở trạng thái ACTIVE"})
		return
	}

	spec, ok := models.GuRegistry[req.GuID]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đặc tính Cổ trùng không hợp lệ"})
		return
	}

	manaCost := spec.BaseManaCost
	if player.CurrentMana < manaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chân Nguyên không đủ để thi triển Sát Chiêu Cổ trùng"})
		return
	}

	// 3. Fetch Enemy
	var en models.TribulationEnemy
	err = tx.QueryRow(`
		SELECT id, name, hp, max_hp, attack, defense, is_boss, is_dead
		FROM tribulation_enemies WHERE id = $1 AND event_id = $2
	`, req.EnemyID, eventID).Scan(&en.ID, &en.Name, &en.HP, &en.MaxHP, &en.Attack, &en.Defense, &en.IsBoss, &en.IsDead)
	if err == sql.ErrNoRows || en.IsDead {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kẻ thù tai kiếp này đã bị tiêu diệt hoặc không tồn tại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin kẻ thù"})
		return
	}

	// 4. Calculate Player Damage (augmented by Dao Marks)
	baseDmg := 35.0
	activeEffects := models.GetGuEffectSection(spec, "active")
	if d, hasDmg := activeEffects["damage"]; hasDmg {
		baseDmg = d
	}

	// Fetch Dao Mark Multiplier
	var daoAmount int
	cleanPath := strings.TrimSuffix(strings.ToLower(spec.Path), " path")
	_ = tx.QueryRow(`
		SELECT amount FROM dao_marks WHERE player_id = $1 AND path = $2
	`, playerID, cleanPath).Scan(&daoAmount)

	// Dao mark soft cap formula
	daoMultiplier := 1.0
	if daoAmount > 0 {
		if daoAmount <= 10000 {
			daoMultiplier = 1.0 + float64(daoAmount)/100.0
		} else {
			daoMultiplier = 1.0 + 100.0 + float64(daoAmount-10000)/500.0
		}
	}

	finalDmg := baseDmg * daoMultiplier
	netDmg := int(finalDmg) - en.Defense
	if netDmg < 10 {
		netDmg = 10
	}

	en.HP -= netDmg
	if en.HP <= 0 {
		en.HP = 0
		en.IsDead = true
	}

	// 5. Enemy Counter-attacks! (Damages player HP and Aperture HP)
	counterDmg := en.Attack
	if en.IsBoss {
		counterDmg = en.Attack * 2
	}

	playerHpDmg := float64(counterDmg) / 10.0
	if player.Rank >= 6 {
		playerHpDmg = float64(counterDmg) / 1000.0 // Protected by Dao Marks
	}
	player.HP -= playerHpDmg
	if player.HP <= 0 {
		player.HP = 1.0 // Prevent instant death in normal mode
	}

	var apHP, apMaxHP int
	err = tx.QueryRow(`
		SELECT aperture_hp, max_aperture_hp FROM immortal_apertures WHERE id = $1
	`, apertureID).Scan(&apHP, &apMaxHP)
	if err == nil {
		newApHP := apHP - counterDmg
		if newApHP < 0 {
			newApHP = 0
		}
		_, _ = tx.Exec(`UPDATE immortal_apertures SET aperture_hp = $1 WHERE id = $2`, newApHP, apertureID)
	}

	// 6. Update Database State
	player.CurrentMana -= manaCost
	_, err = tx.Exec(`
		UPDATE players
		SET current_mana = $1, hp = $2
		WHERE id = $3
	`, player.CurrentMana, player.HP, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu trạng thái Cổ sư"})
		return
	}

	_, err = tx.Exec(`
		UPDATE tribulation_enemies
		SET hp = $1, is_dead = $2
		WHERE id = $3
	`, en.HP, en.IsDead, en.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu trạng thái quái"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Formulate log message
	combatLog := fmt.Sprintf("Thi triển %s: Đạo hữu gây %d sát thương lên %s (Đạo Ngân buff x%.1f). Gánh chịu phản chấn %d sát thương lên cơ thể và Tiên Khiếu.", spec.Name, netDmg, en.Name, daoMultiplier, counterDmg)
	if en.IsDead {
		combatLog += fmt.Sprintf(" Kẻ địch %s đã bị tiêu diệt hoàn toàn!", en.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     combatLog,
		"enemy_hp":    en.HP,
		"player_hp":   player.HP,
		"player_mana": player.CurrentMana,
	})
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
		SELECT id, aperture_hp, max_aperture_hp, tribulation_count, next_tribulation_year FROM immortal_apertures WHERE id = $1
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.TribulationCount, &ap.NextTribulationYear)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin Tiên Khiếu"})
		return
	}

	var msg string
	now := time.Now()

	if remainingEnemies == 0 && ap.ApertureHP > 0 {
		// Clear Victory!
		// Add Dao Marks
		err = addDaoMarksTx(tx, playerID.String(), ev.Path, ev.RewardDaoMarks)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Đạo Ngân: " + err.Error()})
			return
		}

		// Update Aperture state
		ap.State = "STABLE"
		ap.TribulationCount++
		ap.NextTribulationYear += 10.0
		// Restore some HP
		ap.ApertureHP = ap.MaxApertureHP

		_, err = tx.Exec(`
			UPDATE immortal_apertures 
			SET state = $1, tribulation_count = $2, next_tribulation_year = $3, aperture_hp = $4
			WHERE id = $5
		`, ap.State, ap.TribulationCount, ap.NextTribulationYear, ap.ApertureHP, ap.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Tiên Khiếu"})
			return
		}

		// Update event status
		_, err = tx.Exec(`
			UPDATE tribulation_events
			SET status = 'SUCCESS', aperture_hp_after = $1, ended_at = $2
			WHERE id = $3
		`, ap.ApertureHP, now, ev.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật đợt tai kiếp"})
			return
		}

		msg = fmt.Sprintf("Chúc mừng đạo hữu vượt kiếp nạn thành công! Khai quang nhận %d Đạo Ngân hệ %s. Tiên Khiếu của bạn đã ổn định trở lại.", ev.RewardDaoMarks, strings.ToUpper(ev.Path))
	} else {
		// Player resolves manually with remaining enemies (yields high risk of damage)
		successChance := 0.25 // 25% base chance to resolve offline/manually if enemies remain
		if remainingEnemies == 1 {
			successChance = 0.50
		}

		win := rand.Float64() < successChance
		if win {
			// Scraped a win but lost resource nodes
			reducedReward := ev.RewardDaoMarks / 2
			_ = addDaoMarksTx(tx, playerID.String(), ev.Path, reducedReward)

			ap.State = "STABLE"
			ap.TribulationCount++
			ap.NextTribulationYear += 10.0
			ap.ApertureHP = ap.MaxApertureHP / 2 // Heavily damaged

			// Destroy a random resource node
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
				SET state = $1, tribulation_count = $2, next_tribulation_year = $3, aperture_hp = $4
				WHERE id = $5
			`, ap.State, ap.TribulationCount, ap.NextTribulationYear, ap.ApertureHP, ap.ID)

			_, _ = tx.Exec(`
				UPDATE tribulation_events
				SET status = 'SUCCESS', aperture_hp_after = $1, ended_at = $2, reward_dao_marks = $3
				WHERE id = $4
			`, ap.ApertureHP, now, reducedReward, ev.ID)

			msg = fmt.Sprintf("Đạo hữu vượt kiếp chật vật thành công! Nhận %d Đạo Ngân hệ %s, nhưng Tiên Khiếu bị chấn động mạnh (HP giảm còn 50%%), 1 Điểm tài nguyên ngẫu nhiên đã bị phá hủy.", reducedReward, strings.ToUpper(ev.Path))
		} else {
			// Failed Tribulation
			ap.State = "DAMAGED"
			ap.NextTribulationYear += 10.0
			ap.ApertureHP = ap.MaxApertureHP / 10 // Critically low

			// Destroy 2 random resource nodes
			_, _ = tx.Exec(`
				UPDATE aperture_resource_nodes 
				SET is_destroyed = true 
				WHERE id IN (
					SELECT id FROM aperture_resource_nodes 
					WHERE aperture_id = $1 AND is_destroyed = false 
					ORDER BY random() LIMIT 2
				)
			`, ap.ID)

			// Deduct some player resources
			_, _ = tx.Exec(`
				UPDATE players 
				SET primeval_stones = GREATEST(0, primeval_stones - 2000), immortal_stones = GREATEST(0.0, immortal_stones - 5.0)
				WHERE id = $1
			`, playerID)

			_, _ = tx.Exec(`
				UPDATE immortal_apertures 
				SET state = $1, next_tribulation_year = $2, aperture_hp = $3
				WHERE id = $4
			`, ap.State, ap.NextTribulationYear, ap.ApertureHP, ap.ID)

			_, _ = tx.Exec(`
				UPDATE tribulation_events
				SET status = 'FAILED', aperture_hp_after = $1, ended_at = $2, reward_dao_marks = 0
				WHERE id = $3
			`, ap.ApertureHP, now, ev.ID)

			msg = "Kiếp nạn vượt thất bại! Thiên Ý cuồng bạo tàn phá Tiên Khiếu, Nguyên khí hư hao, bạn mất 2,000 Nguyên Thạch, 5 Tiên Nguyên và nhiều điểm tài nguyên bị san phẳng."
		}
	}

	// Clean up dead/remaining enemies
	_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": msg,
		"state":   ap.State,
	})
}

func addDaoMarksTx(tx *sql.Tx, playerID string, path string, amount int) error {
	var count int
	err := tx.QueryRow(`
		SELECT COUNT(1) FROM dao_marks WHERE player_id = $1 AND path = $2
	`, playerID, path).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = tx.Exec(`
			INSERT INTO dao_marks (player_id, path, amount)
			VALUES ($1, $2, $3)
		`, playerID, path, amount)
	} else {
		_, err = tx.Exec(`
			UPDATE dao_marks 
			SET amount = amount + $1, updated_at = NOW() 
			WHERE player_id = $2 AND path = $3
		`, amount, playerID, path)
	}
	if err != nil {
		return err
	}

	// Synchronize back to players.dao_marks JSONB
	rows, err := tx.Query(`SELECT path, amount FROM dao_marks WHERE player_id = $1`, playerID)
	if err != nil {
		return err
	}
	defer rows.Close()

	daoMap := make(map[string]int)
	for rows.Next() {
		var p string
		var a int
		if err := rows.Scan(&p, &a); err == nil {
			daoMap[p] = a
		}
	}

	daoJSON, err := json.Marshal(daoMap)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE players SET dao_marks = $1 WHERE id = $2`, daoJSON, playerID)
	return err
}

// Background Worker Tick: Advance time and trigger tribulations
func CheckAndTriggerTribulations() error {
	// Query stable apertures
	rows, err := db.DB.Query(`
		SELECT id, player_id, time_rate, aperture_hp, max_aperture_hp, internal_years_passed, next_tribulation_year, tribulation_count, essence_production_per_year
		FROM immortal_apertures
		WHERE state = 'STABLE'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type ApertureCheck struct {
		ID                       string
		PlayerID                 string
		TimeRate                 float64
		ApertureHP               int
		MaxApertureHP            int
		InternalYearsPassed      float64
		NextTribulationYear      float64
		TribulationCount         int
		EssenceProductionPerYear int
	}

	apertures := []ApertureCheck{}
	for rows.Next() {
		var ap ApertureCheck
		if err := rows.Scan(&ap.ID, &ap.PlayerID, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.TribulationCount, &ap.EssenceProductionPerYear); err == nil {
			apertures = append(apertures, ap)
		}
	}

	for _, ap := range apertures {
		// Advance internal year: 60s tick interval
		yearsPassed := 60.0 * ap.TimeRate / 72000.0

		// Deduct cultivator lifespan
		var currentLifespan float64
		errLifespan := db.DB.QueryRow(`SELECT lifespan FROM players WHERE id = $1`, ap.PlayerID).Scan(&currentLifespan)
		if errLifespan == nil {
			newLifespan := math.Max(0.0, currentLifespan-yearsPassed)
			_, _ = db.DB.Exec(`UPDATE players SET lifespan = $1 WHERE id = $2`, newLifespan, ap.PlayerID)

			if newLifespan <= 0.0 {
				// Lifespan depleted! Freeze time flow, reduce HP, change state to DAMAGED
				_, _ = db.DB.Exec(`
					UPDATE immortal_apertures
					SET time_rate = 1.0, aperture_hp = max_aperture_hp / 10, state = 'DAMAGED', updated_at = NOW()
					WHERE id = $1
				`, ap.ID)

				// Send system message
				_, _ = db.DB.Exec(`
					INSERT INTO messages (sender, content, channel, player_id, created_at)
					VALUES ('Thiên Địa Chí Tôn', 'Đạo hữu đã hết thọ nguyên! Tiên Khiếu suy kiệt, dòng thời gian ngưng trệ. Hãy mau kiếm Thọ Cổ để hồi sinh!', 'private', $1, NOW())
				`, ap.PlayerID)

				log.Printf("Player %s lifespan depleted, aperture %s frozen", ap.PlayerID, ap.ID)
				continue // Skip the rest of the tick for this aperture
			}
		}

		newYearsPassed := ap.InternalYearsPassed + yearsPassed

		// Sync essence production: when crossing a year boundary, Cổ Tiên receives their annual essence
		yearsDiff := math.Floor(newYearsPassed) - math.Floor(ap.InternalYearsPassed)
		if yearsDiff > 0 {
			addedEssence := float64(yearsDiff) * float64(ap.EssenceProductionPerYear)
			_, _ = db.DB.Exec(`
				UPDATE players 
				SET current_mana = LEAST(max_mana, current_mana + $1), last_mana_update = NOW()
				WHERE id = $2
			`, addedEssence, ap.PlayerID)

			// 1. Ecosystem & Conflicts Tick
			nodeRows, errNodes := db.DB.Query(`
				SELECT id, node_type, path, hp, max_hp, x, y, is_destroyed 
				FROM aperture_resource_nodes 
				WHERE aperture_id = $1 AND is_destroyed = false
			`, ap.ID)
			if errNodes == nil {
				var nodes []models.ApertureResourceNode
				for nodeRows.Next() {
					var nd models.ApertureResourceNode
					if errScan := nodeRows.Scan(&nd.ID, &nd.NodeType, &nd.Path, &nd.X, &nd.Y, &nd.IsDestroyed); errScan == nil {
						nodes = append(nodes, nd)
					}
				}
				nodeRows.Close()

				_, _, _, overgrazing, _, _ := EvaluateEcosystem(nodes)

				for _, nd := range nodes {
					decayDamage := 0
					if CheckNodeConflict(nd, nodes) {
						decayDamage += int(float64(nd.MaxHP) * 0.10 * float64(yearsDiff))
					}
					if overgrazing && (nd.NodeType == "MOON_ORCHID_FIELD" || nd.NodeType == "GREEN_BAMBOO_GROVE" || nd.NodeType == "SWORD_LEAF_GRASS") {
						decayDamage += int(float64(nd.MaxHP) * 0.05 * float64(yearsDiff))
					}

					if decayDamage > 0 {
						var curHP int
						_ = db.DB.QueryRow(`SELECT hp FROM aperture_resource_nodes WHERE id = $1`, nd.ID).Scan(&curHP)
						newHP := curHP - decayDamage
						isDestroyed := false
						if newHP <= 0 {
							newHP = 0
							isDestroyed = true
						}
						_, _ = db.DB.Exec(`
							UPDATE aperture_resource_nodes 
							SET hp = $1, is_destroyed = $2, updated_at = NOW() 
							WHERE id = $3
						`, newHP, isDestroyed, nd.ID)

						if isDestroyed {
							_, _ = db.DB.Exec(`
								INSERT INTO messages (sender, content, channel, player_id, created_at)
								VALUES ('Tiên Khiếu Hệ Thống', 'Điểm tài nguyên ' || $1 || ' đã bị phá hủy do bất ổn sinh thái hoặc xung đột Đạo ngân!', 'private', $2, NOW())
							`, nd.Name, ap.PlayerID)
						}
					}
				}

				// 2. Variant Human Tick
				type VHCheck struct {
					ID             string
					Race           string
					Population     int
					Happiness      float64
					Oppression     float64
					LaborIntensity string
				}
				vRows, errV := db.DB.Query(`
					SELECT id, race, population, happiness, oppression, labor_intensity 
					FROM aperture_variant_humans 
					WHERE aperture_id = $1
				`, ap.ID)
				if errV == nil {
					var vhs []VHCheck
					for vRows.Next() {
						var v VHCheck
						if errScan := vRows.Scan(&v.ID, &v.Race, &v.Population, &v.Happiness, &v.Oppression, &v.LaborIntensity); errScan == nil {
							vhs = append(vhs, v)
						}
					}
					vRows.Close()

					totalFoodProd := 0
					for _, nd := range nodes {
						if nd.NodeType == "MOON_ORCHID_FIELD" || nd.NodeType == "GREEN_BAMBOO_GROVE" || nd.NodeType == "SWORD_LEAF_GRASS" || nd.NodeType == "SPIRIT_DEER_PASTURE" {
							totalFoodProd += nd.ProductionAmount
						}
					}

					for _, v := range vhs {
						happinessChange := 0.0
						oppressionChange := 0.0

						foodNeeded := float64(v.Population) / 100.0
						if float64(totalFoodProd) < foodNeeded {
							happinessChange -= 10.0 * float64(yearsDiff)
						} else {
							happinessChange += 2.0 * float64(yearsDiff)
						}

						switch v.LaborIntensity {
						case "LOW":
							happinessChange += 5.0 * float64(yearsDiff)
							oppressionChange -= 5.0 * float64(yearsDiff)
						case "NORMAL":
							// No changes
						case "HIGH":
							happinessChange -= 10.0 * float64(yearsDiff)
							oppressionChange += 10.0 * float64(yearsDiff)
						case "EXPLOIT":
							happinessChange -= 25.0 * float64(yearsDiff)
							oppressionChange += 25.0 * float64(yearsDiff)
						}

						newHappiness := math.Max(0.0, math.Min(100.0, v.Happiness+happinessChange))
						newOppression := math.Max(0.0, math.Min(100.0, v.Oppression+oppressionChange))

						rebellionRisk := (100.0 - newHappiness + newOppression) / 4.0
						if newHappiness < 30.0 || newOppression > 70.0 {
							if rand.Float64() < (rebellionRisk / 100.0) {
								v.Population = int(float64(v.Population) * 0.8)
								_, _ = db.DB.Exec(`
									UPDATE aperture_resource_nodes 
									SET hp = hp / 2 
									WHERE id IN (
										SELECT id FROM aperture_resource_nodes 
										WHERE aperture_id = $1 AND is_destroyed = false 
										ORDER BY random() LIMIT 2
									)
								`, ap.ID)
								ap.ApertureHP = int(math.Max(0.0, float64(ap.ApertureHP)-float64(ap.MaxApertureHP)*0.05))
								_, _ = db.DB.Exec(`UPDATE immortal_apertures SET aperture_hp = $1 WHERE id = $2`, ap.ApertureHP, ap.ID)

								_, _ = db.DB.Exec(`
									INSERT INTO messages (sender, content, channel, player_id, created_at)
									VALUES ('Tiên Khiếu Hệ Thống', 'Bộ tộc ' || $1 || ' trong Tiên Khiếu đã nổi loạn do bị áp bức dữ dội! Một số điểm tài nguyên và Tiên Khiếu đã bị hư hại.', 'private', $2, NOW())
								`, v.Race, ap.PlayerID)
							}
						}

						settlementCount := 0
						targetSettlementType := ""
						switch v.Race {
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
						newPop := v.Population
						if settlementCount > 0 {
							if newHappiness > 60.0 {
								newPop = int(float64(v.Population) * math.Pow(1.05, float64(yearsDiff)))
							}
							if newPop > maxPop {
								newPop = maxPop
							}
						} else {
							newPop = int(float64(v.Population) * math.Pow(0.90, float64(yearsDiff)))
						}

						_, _ = db.DB.Exec(`
							UPDATE aperture_variant_humans
							SET population = $1, happiness = $2, oppression = $3, updated_at = NOW()
							WHERE id = $4
						`, newPop, newHappiness, newOppression, v.ID)
					}
				}
			}
		}

		if newYearsPassed >= ap.NextTribulationYear {
			// Trigger Tribulation!
			err := spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.TribulationCount, ap.ApertureHP, newYearsPassed)
			if err != nil {
				log.Printf("Failed to spawn tribulation event for player %s: %v", ap.PlayerID, err)
			}
		} else {
			// Update time in DB
			_, _ = db.DB.Exec(`
				UPDATE immortal_apertures 
				SET internal_years_passed = $1, updated_at = NOW() 
				WHERE id = $2
			`, newYearsPassed, ap.ID)
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
			log.Printf("Auto-resolving expired tribulation event %s for player %s", r.EventID, r.PlayerID)
			// Trigger a mock resolve
			autoResolveEvent(r.EventID, r.PlayerID)
		}
	}

	return nil
}

func spawnTribulationEventWorker(apertureID string, playerID string, tribCount int, currentHP int, newYearsPassed float64) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Pick a random path for the Calamity
	paths := []string{"fire", "water", "ice", "blood", "sword", "strength", "soul", "wisdom", "time", "space"}
	path := paths[rand.Intn(len(paths))]

	// Determine tribulation type based on Rank (currently player rank)
	var rank int
	err = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&rank)
	if err != nil {
		return err
	}

	tribulationType := "EARTH_CALAMITY"
	if rank == 7 {
		if (tribCount+1)%10 == 0 {
			tribulationType = "GRAND_TRIBULATION"
		} else if (tribCount+1)%5 == 0 {
			tribulationType = "HEAVENLY_TRIBULATION"
		}
	} else if rank >= 8 {
		if (tribCount+1)%10 == 0 {
			tribulationType = "MYRIAD_TRIBULATION"
		} else {
			tribulationType = "HEAVENLY_TRIBULATION"
		}
	} else { // Rank 6
		if (tribCount+1)%10 == 0 {
			tribulationType = "HEAVENLY_TRIBULATION"
		}
	}

	difficulty := 1 + tribCount
	rewardMarks := 250 + difficulty*50
	switch tribulationType {
	case "HEAVENLY_TRIBULATION":
		rewardMarks = 750 + difficulty*150
	case "GRAND_TRIBULATION":
		rewardMarks = 7250 + difficulty*500
	case "MYRIAD_TRIBULATION":
		rewardMarks = 86750 + difficulty*2000
	}

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
	}

	var boss models.MonsterTemplate
	var bossGu []byte
	err = tx.QueryRow(`
		SELECT id, name, type, rank, sub_rank, path, max_hp, base_attack, base_defense, max_mana, current_mana, gu_inventory, ai_profile, loot_table_id, description
		FROM monster_templates WHERE id = $1
	`, bossTemplateID).Scan(
		&boss.ID, &boss.Name, &boss.Type, &boss.Rank, &boss.SubRank, &boss.Path,
		&boss.MaxHP, &boss.BaseAttack, &boss.BaseDefense, &boss.MaxMana, &boss.CurrentMana,
		&bossGu, &boss.AIProfile, &boss.LootTableID, &boss.Description,
	)
	if err != nil {
		return fmt.Errorf("load tribulation boss template %s from database: %w", bossTemplateID, err)
	}

	// Apply difficulty scaling
	diffMultiplier := 1.0 + float64(difficulty)*0.2
	scaledHP, scaledAtk, scaledDef := models.ScaleMonsterStats(boss.MaxHP, boss.BaseAttack, boss.BaseDefense, boss.Rank, boss.Type)
	bossHP := int(scaledHP * diffMultiplier)
	bossAtk := int(scaledAtk * diffMultiplier)
	bossDef := int(scaledDef * diffMultiplier)

	_, err = tx.Exec(`
		INSERT INTO tribulation_enemies (event_id, name, enemy_type, path, hp, max_hp, attack, defense, x, y, is_boss, is_dead)
		VALUES ($1, $2, 'BOSS', $3, $4, $4, $5, $6, $7, $8, true, false)
	`, eventID, fmt.Sprintf("%s (%s Path)", boss.Name, strings.Title(path)), path, bossHP, bossAtk, bossDef, rand.Intn(20), rand.Intn(20))
	if err != nil {
		return err
	}

	// Spawn Minions
	var minion models.MonsterTemplate
	var minionGu []byte
	err = tx.QueryRow(`
		SELECT id, name, type, rank, sub_rank, path, max_hp, base_attack, base_defense, max_mana, current_mana, gu_inventory, ai_profile, loot_table_id, description
		FROM monster_templates WHERE id = 'monster_tribulation_beast'
	`).Scan(
		&minion.ID, &minion.Name, &minion.Type, &minion.Rank, &minion.SubRank, &minion.Path,
		&minion.MaxHP, &minion.BaseAttack, &minion.BaseDefense, &minion.MaxMana, &minion.CurrentMana,
		&minionGu, &minion.AIProfile, &minion.LootTableID, &minion.Description,
	)
	if err != nil {
		return fmt.Errorf("load tribulation minion template monster_tribulation_beast from database: %w", err)
	}

	minionCount := 1 + rand.Intn(3)
	for i := 0; i < minionCount; i++ {
		scaledHPm, scaledAtkm, scaledDefm := models.ScaleMonsterStats(minion.MaxHP, minion.BaseAttack, minion.BaseDefense, minion.Rank, minion.Type)
		mHP := int(scaledHPm * diffMultiplier)
		mAtk := int(scaledAtkm * diffMultiplier)
		mDef := int(scaledDefm * diffMultiplier)

		_, err = tx.Exec(`
			INSERT INTO tribulation_enemies (event_id, name, enemy_type, path, hp, max_hp, attack, defense, x, y, is_boss, is_dead)
			VALUES ($1, $2, 'MINION', $3, $4, $4, $5, $6, $7, $8, false, false)
		`, eventID, fmt.Sprintf("%s #%d", minion.Name, i+1), path, mHP, mAtk, mDef, rand.Intn(20), rand.Intn(20))
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

	// Offline failure (auto-resolve always defaults to fail/damaged status)
	var feathermanPop int
	_ = tx.QueryRow(`SELECT population FROM aperture_variant_humans WHERE aperture_id = $1 AND race = 'FEATHERMAN'`, ev.ApertureID).Scan(&feathermanPop)
	reduction := float64(feathermanPop) * 0.001
	if reduction > 0.5 {
		reduction = 0.5
	}

	hpFactor := 0.1 + 0.4*reduction // default 0.1, up to 0.3 if max feathermen
	newHP := int(float64(ev.ApertureHPBefore) * hpFactor)
	if newHP < 100 {
		newHP = 100
	}

	_, _ = tx.Exec(`
		UPDATE immortal_apertures 
		SET state = 'DAMAGED', next_tribulation_year = next_tribulation_year + 10.0, aperture_hp = $1
		WHERE id = $2
	`, newHP, ev.ApertureID)

	_, _ = tx.Exec(`
		UPDATE tribulation_events
		SET status = 'FAILED', aperture_hp_after = $1, ended_at = NOW(), reward_dao_marks = 0
		WHERE id = $2
	`, newHP, ev.ID)

	// Destroy 2 resource nodes
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

// ProcessActiveTribulationGrid moves tribulation enemies towards resource nodes and attacks them
func ProcessActiveTribulationGrid() error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Fetch all active tribulation events
	rows, err := tx.Query(`
		SELECT id, aperture_id, player_id 
		FROM tribulation_events 
		WHERE status = 'ACTIVE'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type ActiveEvent struct {
		EventID    string
		ApertureID string
		PlayerID   string
	}
	var events []ActiveEvent
	for rows.Next() {
		var ev ActiveEvent
		if err := rows.Scan(&ev.EventID, &ev.ApertureID, &ev.PlayerID); err == nil {
			events = append(events, ev)
		}
	}
	rows.Close()

	for _, ev := range events {
		// Fetch living enemies
		enRows, err := tx.Query(`
			SELECT id, name, x, y, attack 
			FROM tribulation_enemies 
			WHERE event_id = $1 AND is_dead = false
		`, ev.EventID)
		if err != nil {
			continue
		}

		type TEnemy struct {
			ID     string
			Name   string
			X      int
			Y      int
			Attack int
		}
		var enemies []TEnemy
		for enRows.Next() {
			var en TEnemy
			if err := enRows.Scan(&en.ID, &en.Name, &en.X, &en.Y, &en.Attack); err == nil {
				enemies = append(enemies, en)
			}
		}
		enRows.Close()

		if len(enemies) == 0 {
			continue
		}

		// Fetch non-destroyed resource nodes
		nodeRows, err := tx.Query(`
			SELECT id, name, hp, x, y 
			FROM aperture_resource_nodes 
			WHERE aperture_id = $1 AND is_destroyed = false
		`, ev.ApertureID)
		if err != nil {
			continue
		}

		type TNode struct {
			ID   string
			Name string
			HP   int
			X    int
			Y    int
		}
		var nodes []TNode
		for nodeRows.Next() {
			var nd TNode
			if err := nodeRows.Scan(&nd.ID, &nd.Name, &nd.HP, &nd.X, &nd.Y); err == nil {
				nodes = append(nodes, nd)
			}
		}
		nodeRows.Close()

		if len(nodes) == 0 {
			continue // No nodes to attack
		}

		// Process each enemy
		for _, en := range enemies {
			// Find nearest node
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
				nearestNode.HP -= en.Attack
				isDestroyed := false
				if nearestNode.HP <= 0 {
					nearestNode.HP = 0
					isDestroyed = true
				}
				_, _ = tx.Exec(`
					UPDATE aperture_resource_nodes 
					SET hp = $1, is_destroyed = $2, updated_at = NOW() 
					WHERE id = $3
				`, nearestNode.HP, isDestroyed, nearestNode.ID)

				if isDestroyed {
					// Send system message
					_, _ = tx.Exec(`
						INSERT INTO messages (sender, content, channel, player_id, created_at)
						VALUES ('Tiên Khiếu Hệ Thống', 'Tai kiếp đã phá hủy hoàn toàn ' || $1 || '!', 'private', $2, NOW())
					`, nearestNode.Name, ev.PlayerID)

					// Remove from local list so other enemies target something else
					var remainingNodes []TNode
					for _, nd := range nodes {
						if nd.ID != nearestNode.ID {
							remainingNodes = append(remainingNodes, nd)
						}
					}
					nodes = remainingNodes
				}
			} else {
				// Move towards the node
				if absDx > absDy {
					if dx > 0 {
						en.X++
					} else {
						en.X--
					}
				} else {
					if dy > 0 {
						en.Y++
					} else {
						en.Y--
					}
				}
				_, _ = tx.Exec(`
					UPDATE tribulation_enemies 
					SET x = $1, y = $2, updated_at = NOW() 
					WHERE id = $3
				`, en.X, en.Y, en.ID)
			}
		}
	}

	return tx.Commit()
}

func StartTribulationWorker() {
	ticker60 := time.NewTicker(60 * time.Second)
	ticker15 := time.NewTicker(15 * time.Second)

	go func() {
		for range ticker60.C {
			err := CheckAndTriggerTribulations()
			if err != nil {
				log.Printf("Tribulation background worker error: %v", err)
			}
		}
	}()

	go func() {
		for range ticker15.C {
			err := ProcessActiveTribulationGrid()
			if err != nil {
				log.Printf("Tribulation grid worker error: %v", err)
			}
		}
	}()

	log.Println("Tribulation and internal year worker started successfully (Interval: 60s/15s)")
}
