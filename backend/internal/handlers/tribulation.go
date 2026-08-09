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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
	var hpAfter sql.NullInt64
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
		h := int(hpAfter.Int64)
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
	EnemyID string   `json:"enemy_id" binding:"required"`
	GuID    string   `json:"gu_id"`  // Gu bug ID used for attack (legacy/fallback)
	GuIDs   []string `json:"gu_ids"` // List of Gu IDs to apply simultaneously
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

	// Support both single gu_id and multiple gu_ids
	guIDs := req.GuIDs
	if len(guIDs) == 0 && req.GuID != "" {
		guIDs = []string{req.GuID}
	}

	if len(guIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu cần chọn ít nhất một Cổ trùng để công kích"})
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

	// 1b. Player may only fight back while the Aperture is above 5% max HP
	var apHP, apMaxHP int
	err = tx.QueryRow(`
		SELECT aperture_hp, max_aperture_hp FROM immortal_apertures WHERE id = $1
	`, apertureID).Scan(&apHP, &apMaxHP)
	if err == nil && apHP*20 <= apMaxHP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu đang nguy cấp (HP ≤ 5%)! Quái tai kiếp đang càn quét, đạo hữu không thể ra tay. Hãy chờ Tiên Khiếu hồi phục hoặc kết toán vượt kiếp."})
		return
	}

	// 2. Fetch Player stats
	var player models.Player
	err = tx.QueryRow(`
		SELECT p.id, p.username, p.rank, p.sub_rank, p.current_mana, p.max_mana, p.hp, p.base_hp, p.primeval_stones, `+db.GuInventoryAggregate("p.id")+`, COALESCE(ia.dao_marks, '{}'::jsonb), p.path_masteries, p.growth_data
		FROM players p
		LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
		WHERE p.id = $1
	`, playerID).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.CurrentMana, &player.MaxMana, &player.HP, &player.BaseHP,
		&player.PrimevalStones, &player.GuInventory, &player.DaoMarks,
		&player.PathMasteries, &player.GrowthData,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin cổ sư"})
		return
	}

	// Verify player has all the selected Gu bugs and they are active and equipped
	var foundGus []*models.GuInstance
	for _, targetGuID := range guIDs {
		var foundGu *models.GuInstance
		for i := range player.GuInventory {
			if player.GuInventory[i].GuID == targetGuID && player.GuInventory[i].Status == "ACTIVE" && player.GuInventory[i].Equipped {
				foundGu = &player.GuInventory[i]
				break
			}
		}

		if foundGu == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo hữu không sở hữu, chưa trang bị (equipped) hoặc Cổ trùng %s không ở trạng thái ACTIVE", targetGuID)})
			return
		}
		foundGus = append(foundGus, foundGu)
	}

	totalManaCost := 0.0
	var guSpecs []models.GuDictionaryItem
	for _, fg := range foundGus {
		spec, ok := models.GetGuAny(fg.GuID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đặc tính Cổ trùng %s không hợp lệ", fg.GuID)})
			return
		}
		_, _, cost, errGu := CalculateGuEffect(&player, &spec)
		if errGu == nil {
			totalManaCost += cost
		} else {
			totalManaCost += spec.BaseManaCost
		}
		guSpecs = append(guSpecs, spec)
	}

	if player.CurrentMana < totalManaCost {
		manaType := "Chân Nguyên"
		if player.Rank >= 6 {
			manaType = "Tiên Nguyên"
		}

		guNames := []string{}
		for _, gu := range foundGus {
			guNames = append(guNames, gu.GuID)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s không đủ để thi triển %s (cần %.0f, hiện có %.0f)", manaType, strings.Join(guNames, ", "), totalManaCost, player.CurrentMana)})
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

	// 4. Calculate Player Damage (augmented by Dao Marks) and other effects across all Gu
	totalFinalDmg := 0.0
	totalHeal := 0.0
	totalShield := 0.0
	var guNames []string

	actor := models.CombatantFromPlayer(&player, CalculatePlayerStats(&player))

	// Always apply Basic Attack (Đánh thường)
	daoMult := gamefx.DaoMultiplierForPaths(actor.DaoMarks, []string{"strength"}, gamefx.Get(&player).DaoNganBonus)
	dmgVal := actor.BaseAttack
	if dmgVal <= 0 {
		dmgVal = 10.0
	}
	fullDmg := dmgVal * daoMult
	totalFinalDmg = fullDmg
	if daoMult > 1.0 {
		dmgDao := fullDmg - dmgVal
		guNames = append(guNames, fmt.Sprintf("Đánh thường (gây %.0f sát thương [Gốc: %.0f, Đạo Ngân: +%.0f])", fullDmg, dmgVal, dmgDao))
	} else {
		guNames = append(guNames, fmt.Sprintf("Đánh thường (gây %.0f sát thương)", dmgVal))
	}

	// Apply selected active Gu effects
	for _, spec := range guSpecs {
		dmgBase, defBase, _, errGuBase := CalculateGuEffectForCombatant(&actor, &spec, false, gamefx.Get(&player).DaoNganBonus, gamefx.Get(&player).PathDamageBonus)
		dmg, def, _, errGu := CalculateGuEffectForCombatant(&actor, &spec, true, gamefx.Get(&player).DaoNganBonus, gamefx.Get(&player).PathDamageBonus)

		if errGu == nil && errGuBase == nil {
			totalFinalDmg += dmg
			totalShield += def
		} else {
			activeEffects := models.GetFlatGuEffects(spec)
			baseDmg := 35.0
			if d, hasDmg := activeEffects["damage"]; hasDmg {
				baseDmg = d
			}
			totalFinalDmg += baseDmg
			dmg = baseDmg
			dmgBase = baseDmg
			if shieldVal, hasShield := activeEffects["defense"]; hasShield && shieldVal > 0 {
				totalShield += shieldVal
				def = shieldVal
				defBase = shieldVal
			}
		}

		activeEffects := models.GetFlatGuEffects(spec)
		healVal, hasHeal := activeEffects["heal"]
		if hasHeal && healVal > 0 {
			totalHeal += healVal
		}

		// Build detailed logs for this specific Gu
		var details []string
		if dmg > 0 {
			dmgDao := dmg - dmgBase
			if dmgDao > 0 {
				details = append(details, fmt.Sprintf("gây %.0f sát thương [Gốc: %.0f, Đạo Ngân: +%.0f]", dmg, dmgBase, dmgDao))
			} else {
				details = append(details, fmt.Sprintf("gây %.0f sát thương", dmg))
			}
		}
		if def > 0 {
			defDao := def - defBase
			if defDao > 0 {
				details = append(details, fmt.Sprintf("tạo %.0f Khiên [Gốc: %.0f, Đạo Ngân: +%.0f]", def, defBase, defDao))
			} else {
				details = append(details, fmt.Sprintf("tạo %.0f Khiên", def))
			}
		}
		if hasHeal && healVal > 0 {
			details = append(details, fmt.Sprintf("hồi %.0f HP", healVal))
		}

		guDesc := spec.Name
		if len(details) > 0 {
			guDesc += fmt.Sprintf(" (%s)", strings.Join(details, ", "))
		}
		guNames = append(guNames, guDesc)
	}

	netDmg := int(totalFinalDmg) - en.Defense
	if netDmg < 0 {
		netDmg = 0
	}

	en.HP -= netDmg
	if en.HP <= 0 {
		en.HP = 0
		en.IsDead = true
	}

	// 5. Player self-buffs from the Gu: heal, then raise a shield
	if totalHeal > 0 {
		player.HP += totalHeal
		if player.HP > gamefx.EffectiveMaxHP(&player) {
			player.HP = gamefx.EffectiveMaxHP(&player)
		}
	}
	shield := totalShield

	// 6. Enemy Counter-attacks! (Damages player HP and Aperture HP) only if they survived
	rawCounterDmg := 0
	counterDmg := 0
	playerHpDmg := 0.0
	absorbed := 0.0

	if !en.IsDead {
		rawCounterDmg = en.Attack
		if en.IsBoss {
			rawCounterDmg = en.Attack * 2
		}

		// Subtract player defense from counter attack damage
		counterDmg = rawCounterDmg - int(actor.BaseDefense)
		if counterDmg < 10 {
			counterDmg = 10
		}

		if player.Rank >= 6 {
			// Gu Immortal: Shield protects the Immortal Aperture from raw counterDmg
			if shield > 0 {
				absorbed = math.Min(shield, float64(counterDmg))
				shield -= absorbed
				counterDmg -= int(absorbed)
			}
			// Body damage is a fraction of the remaining counterDmg
			playerHpDmg = float64(counterDmg) / 1000.0
		} else {
			// Mortal: Shield protects body HP directly
			playerHpDmg = float64(counterDmg) / 10.0
			if shield > 0 {
				absorbed = math.Min(shield, playerHpDmg)
				shield -= absorbed
				playerHpDmg -= absorbed
			}
		}

		player.HP -= playerHpDmg
		if player.HP <= 0 {
			player.HP = 1.0 // Prevent instant death in normal mode
		}

		newApHP := apHP - counterDmg
		if newApHP < 0 {
			newApHP = 0
		}
		_, _ = tx.Exec(`UPDATE immortal_apertures SET aperture_hp = $1 WHERE id = $2`, newApHP, apertureID)
	}

	// 7. Update Database State
	player.CurrentMana -= totalManaCost
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
	var playerActionMsg string
	if netDmg > 0 {
		playerActionMsg = fmt.Sprintf("Kích hoạt %s => Gây %d sát thương lên %s", strings.Join(guNames, ", "), netDmg, en.Name)
	} else {
		playerActionMsg = fmt.Sprintf("Kích hoạt %s => Không thể xuyên qua phòng ngự của %s (gây 0 sát thương)", strings.Join(guNames, ", "), en.Name)
	}

	extras := []string{}
	if totalShield > 0 {
		extras = append(extras, fmt.Sprintf("tạo %.0f Khiên", totalShield))
	}
	if totalHeal > 0 {
		extras = append(extras, fmt.Sprintf("hồi phục %.0f HP", totalHeal))
	}
	if len(extras) > 0 {
		playerActionMsg += ", " + strings.Join(extras, " và ")
	}
	playerActionMsg += "."

	combatLog := playerActionMsg
	if en.IsDead {
		combatLog += fmt.Sprintf(" Kẻ địch %s đã bị tiêu diệt hoàn toàn!", en.Name)
	} else {
		targetType := "cơ thể"
		dmgSuffered := int(playerHpDmg)
		if player.Rank >= 6 {
			targetType = "Tiên Khiếu"
			dmgSuffered = counterDmg
		}

		counterMsg := fmt.Sprintf(" Kẻ địch %s phản công gây %d sát thương.", en.Name, rawCounterDmg)
		if absorbed > 0 {
			counterMsg += fmt.Sprintf(" Khiên hấp thụ %.0f sát thương.", absorbed)
		}
		counterMsg += fmt.Sprintf(" Đạo hữu gánh chịu %d sát thương lên %s.", dmgSuffered, targetType)
		combatLog += counterMsg
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       combatLog,
		"enemy_hp":      en.HP,
		"player_hp":     player.HP,
		"player_shield": shield,
		"player_mana":   player.CurrentMana,
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
		SELECT id, aperture_hp, max_aperture_hp, tribulation_count, next_tribulation_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, internal_years_passed
		FROM immortal_apertures WHERE id = $1
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.TribulationCount, &ap.NextTribulationYear,
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
		ap.State = "STABLE"
		ap.ApertureHP = ap.MaxApertureHP

		// Recalculate next tribulation year based on rank & counters
		var playerRank int
		_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&playerRank)
		nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
		if nextYear < 10 {
			nextYear = 10
		}
		ap.NextTribulationYear = nextYear

		// Check breakthrough conditions
		yearsAtRank := ap.InternalYearsPassed - ap.RankStartYear
		canBreak, newRank := models.CheckImmortalBreakthrough(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, yearsAtRank)
		if canBreak {
			_, _ = tx.Exec(`UPDATE players SET rank = $1 WHERE id = $2`, newRank, playerID)
			ap.RankStartYear = ap.InternalYearsPassed
			didBreakthrough = true
			breakthroughMsg = fmt.Sprintf(" Đột phá! Đạo hữu đã tiến vào %s!", getRankName(newRank))

			// Recalculate next tribulation year based on the new rank!
			nextYear, _ = models.GetNextTribulationInfo(newRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
			if nextYear < 10 {
				nextYear = 10
			}
			ap.NextTribulationYear = nextYear
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

		handleSuccess(ev.TribulationType)

		_, err = tx.Exec(`
			UPDATE immortal_apertures 
			SET state = $1, tribulation_count = $2, next_tribulation_year = $3, aperture_hp = $4,
				rank_start_year = $5, earth_calamity_count = $6, heavenly_tribulation_count = $7,
				grand_tribulation_count = $8, myriad_tribulation_count = $9, chaos_calamity_count = $10
			WHERE id = $11
		`, ap.State, ap.TribulationCount, ap.NextTribulationYear, ap.ApertureHP,
			ap.RankStartYear, ap.EarthCalamityCount, ap.HeavenlyTribulationCount,
			ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Tiên Khiếu"})
			return
		}

		_, err = tx.Exec(`
			UPDATE tribulation_events
			SET status = 'SUCCESS', aperture_hp_after = $1, ended_at = $2
			WHERE id = $3
		`, ap.ApertureHP, now, ev.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật đợt tai kiếp"})
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

			handleSuccess(ev.TribulationType)
			ap.ApertureHP = ap.MaxApertureHP / 2

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
				SET state = $1, tribulation_count = $2, next_tribulation_year = $3, aperture_hp = $4,
					rank_start_year = $5, earth_calamity_count = $6, heavenly_tribulation_count = $7,
					grand_tribulation_count = $8, myriad_tribulation_count = $9, chaos_calamity_count = $10
				WHERE id = $11
			`, ap.State, ap.TribulationCount, ap.NextTribulationYear, ap.ApertureHP,
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
			// Failed Tribulation — counters are already incremented on spawn
			var playerRank int
			_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&playerRank)
			// Compute nextYear using the incremented counts
			nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
			if nextYear < 10 {
				nextYear = 10
			}
			ap.NextTribulationYear = nextYear

			// Revert counts now
			if ap.TribulationCount > 0 {
				ap.TribulationCount--
			}
			switch ev.TribulationType {
			case "EARTH_CALAMITY":
				if ap.EarthCalamityCount > 0 {
					ap.EarthCalamityCount--
				}
			case "HEAVENLY_TRIBULATION":
				if ap.HeavenlyTribulationCount > 0 {
					ap.HeavenlyTribulationCount--
				}
			case "GRAND_TRIBULATION":
				if ap.GrandTribulationCount > 0 {
					ap.GrandTribulationCount--
				}
			case "MYRIAD_TRIBULATION":
				if ap.MyriadTribulationCount > 0 {
					ap.MyriadTribulationCount--
				}
			case "CHAOS_CALAMITY":
				if ap.ChaosCalamityCount > 0 {
					ap.ChaosCalamityCount--
				}
			}

			ap.State = "DAMAGED"
			ap.ApertureHP = ap.MaxApertureHP / 10

			_, _ = tx.Exec(`
				UPDATE aperture_resource_nodes 
				SET is_destroyed = true 
				WHERE id IN (
					SELECT id FROM aperture_resource_nodes 
					WHERE aperture_id = $1 AND is_destroyed = false 
					ORDER BY random() LIMIT 2
				)
			`, ap.ID)

			_, _ = tx.Exec(`
				UPDATE players 
				SET primeval_stones = GREATEST(0, primeval_stones - 2000), immortal_stones = GREATEST(0.0, immortal_stones - 5.0)
				WHERE id = $1
			`, playerID)

			_, _ = tx.Exec(`
				UPDATE immortal_apertures 
				SET state = $1, next_tribulation_year = $2, aperture_hp = $3,
					earth_calamity_count = $4, heavenly_tribulation_count = $5, grand_tribulation_count = $6,
					myriad_tribulation_count = $7, chaos_calamity_count = $8, tribulation_count = $9
				WHERE id = $10
			`, ap.State, ap.NextTribulationYear, ap.ApertureHP,
				ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount,
				ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.TribulationCount, ap.ID)

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
		ApertureHP               int
		MaxApertureHP            int
		InternalYearsPassed      float64
		NextTribulationYear      float64
		TribulationCount         int
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
		SELECT id, player_id, time_rate, aperture_hp, max_aperture_hp, internal_years_passed, next_tribulation_year, tribulation_count, essence_production_per_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, state
		FROM immortal_apertures
		WHERE id = $1
	`, apertureID).Scan(&ap.ID, &ap.PlayerID, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.TribulationCount, &ap.EssenceProductionPerYear,
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

	calcNextYear, calcNextType := models.GetNextTribulationInfo(rank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
	if calcNextYear < 10 {
		calcNextYear = 10
	}

	if ap.InternalYearsPassed >= calcNextYear {
		// Determine all types due at this year
		typesToInc := []string{}
		for _, tc := range models.GetTribulationTypesForRank(rank) {
			ty := tc.Interval * float64(models.GetTribTypeCount(&models.ImmortalAperture{
				EarthCalamityCount:       ap.EarthCalamityCount,
				HeavenlyTribulationCount: ap.HeavenlyTribulationCount,
				GrandTribulationCount:    ap.GrandTribulationCount,
				MyriadTribulationCount:   ap.MyriadTribulationCount,
				ChaosCalamityCount:       ap.ChaosCalamityCount,
			}, tc.Type)+1)
			if math.Abs(ty-calcNextYear) < 0.1 {
				typesToInc = append(typesToInc, tc.Type)
			}
		}

		// Increment counters for ALL types due at this year
		for _, tt := range typesToInc {
			switch tt {
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
		}

		_, err = db.DB.Exec(`
			UPDATE immortal_apertures 
			SET earth_calamity_count = $1, heavenly_tribulation_count = $2, grand_tribulation_count = $3,
				myriad_tribulation_count = $4, chaos_calamity_count = $5, tribulation_count = tribulation_count + 1,
				internal_years_passed = $6, updated_at = NOW()
			WHERE id = $7
		`, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.InternalYearsPassed, ap.ID)
		if err != nil {
			return err
		}

		spawnType := calcNextType
		err = spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.TribulationCount+1, ap.ApertureHP, ap.InternalYearsPassed, spawnType)
		if err != nil {
			return err
		}
	} else {
		_, err = db.DB.Exec(`
			UPDATE immortal_apertures 
			SET next_tribulation_year = $1, updated_at = NOW()
			WHERE id = $2
		`, calcNextYear, ap.ID)
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
func CheckAndTriggerTribulations() error {
	// Query stable apertures
	rows, err := db.DB.Query(`
		SELECT id, player_id, time_rate, aperture_hp, max_aperture_hp, internal_years_passed, next_tribulation_year, tribulation_count, essence_production_per_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count
		FROM immortal_apertures
		WHERE state IN ('STABLE', 'DAMAGED')
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
		RankStartYear            float64
		EarthCalamityCount       int
		HeavenlyTribulationCount int
		GrandTribulationCount    int
		MyriadTribulationCount   int
		ChaosCalamityCount       int
	}

	apertures := []ApertureCheck{}
	for rows.Next() {
		var ap ApertureCheck
		if err := rows.Scan(&ap.ID, &ap.PlayerID, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.TribulationCount, &ap.EssenceProductionPerYear,
			&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount); err == nil {
			apertures = append(apertures, ap)
		}
	}

	for _, ap := range apertures {
		// Advance internal year: 60s tick interval
		timeSpeedMultiplier := 1.0
		realYearsPassed := 60.0 / 36000.0

		// Deduct cultivator lifespan (real time, not scaled by TimeRate)
		var currentLifespan float64
		errLifespan := db.DB.QueryRow(`SELECT lifespan, time_speed_multiplier FROM players WHERE id = $1`, ap.PlayerID).Scan(&currentLifespan, &timeSpeedMultiplier)
		if errLifespan == nil {
			if timeSpeedMultiplier <= 0 {
				timeSpeedMultiplier = 1.0
			}
			if currentLifespan <= 0.0 {
				// Lifespan already depleted, time does not flow and no message spam
				continue
			}

			newLifespan := math.Max(0.0, currentLifespan-(realYearsPassed*timeSpeedMultiplier))
			_, _ = db.DB.Exec(`UPDATE players SET lifespan = $1 WHERE id = $2`, newLifespan, ap.PlayerID)

			if newLifespan <= 0.0 {
				_, _ = db.DB.Exec(`
					UPDATE immortal_apertures
					SET time_rate = 1.0, aperture_hp = max_aperture_hp / 10, state = 'DAMAGED', updated_at = NOW()
					WHERE id = $1
				`, ap.ID)

				_, _ = db.DB.Exec(`
					INSERT INTO messages (sender_id, receiver_id, content, channel)
					VALUES ('11111111-1111-1111-1111-111111111111', $1, 'Đạo hữu đã hết thọ nguyên! Tiên Khiếu suy kiệt, dòng thời gian ngưng trệ. Hãy mau kiếm Thọ Cổ để hồi sinh!', 'private')
				`, ap.PlayerID)

				logger.S.Warnf("Player %s lifespan depleted, aperture %s frozen", ap.PlayerID, ap.ID)
				continue
			}
		} else {
			timeSpeedMultiplier = 1.0
		}

		subjectiveYearsPassed := 60.0 * ap.TimeRate * timeSpeedMultiplier / 36000.0

		newYearsPassed := ap.InternalYearsPassed + subjectiveYearsPassed

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
								INSERT INTO messages (sender_id, receiver_id, content, channel)
								VALUES ('11111111-1111-1111-1111-111111111111', $1, 'Điểm tài nguyên ' || $2 || ' đã bị phá hủy do bất ổn sinh thái hoặc xung đột Đạo ngân!', 'private')
							`, ap.PlayerID, nd.Name)
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
									INSERT INTO messages (sender_id, receiver_id, content, channel)
									VALUES ('11111111-1111-1111-1111-111111111111', $1, 'Bộ tộc ' || $2 || ' trong Tiên Khiếu đã nổi loạn do bị áp bức dữ dội! Một số điểm tài nguyên và Tiên Khiếu đã bị hư hại.', 'private')
								`, ap.PlayerID, v.Race)
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

		// --- New tribulation scheduling logic ---
		// Recalculate next tribulation year based on rank + counters
		var rank int
		_ = db.DB.QueryRow(`SELECT rank FROM players WHERE id = $1`, ap.PlayerID).Scan(&rank)

		calcNextYear, calcNextType := models.GetNextTribulationInfo(rank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)

		if calcNextYear < 10 {
			calcNextYear = 10
		}

		if newYearsPassed >= calcNextYear {
			// Determine all types due at this year
			typesToInc := []string{}
			for _, tc := range models.GetTribulationTypesForRank(rank) {
				ty := tc.Interval * float64(models.GetTribTypeCount(&models.ImmortalAperture{
					EarthCalamityCount:       ap.EarthCalamityCount,
					HeavenlyTribulationCount: ap.HeavenlyTribulationCount,
					GrandTribulationCount:    ap.GrandTribulationCount,
					MyriadTribulationCount:   ap.MyriadTribulationCount,
					ChaosCalamityCount:       ap.ChaosCalamityCount,
				}, tc.Type)+1)
				if math.Abs(ty-calcNextYear) < 0.1 {
					typesToInc = append(typesToInc, tc.Type)
				}
			}

			// Increment counters for ALL types due at this year
			for _, tt := range typesToInc {
				switch tt {
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
			}

			_, _ = db.DB.Exec(`
				UPDATE immortal_apertures 
				SET earth_calamity_count = $1, heavenly_tribulation_count = $2, grand_tribulation_count = $3,
					myriad_tribulation_count = $4, chaos_calamity_count = $5, tribulation_count = tribulation_count + 1,
					internal_years_passed = $6, updated_at = NOW()
				WHERE id = $7
			`, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, newYearsPassed, ap.ID)

			// Spawn the highest priority type
			spawnType := calcNextType
			err := spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.TribulationCount+1, ap.ApertureHP, newYearsPassed, spawnType)
			if err != nil {
				logger.S.Errorf("Failed to spawn tribulation event for player %s: %v", ap.PlayerID, err)
			}
		} else {
			// Update counters and time in DB
			_, _ = db.DB.Exec(`
				UPDATE immortal_apertures 
				SET internal_years_passed = $1, earth_calamity_count = $2, heavenly_tribulation_count = $3,
					grand_tribulation_count = $4, myriad_tribulation_count = $5, chaos_calamity_count = $6,
					next_tribulation_year = $7, updated_at = NOW()
				WHERE id = $8
			`, newYearsPassed, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, calcNextYear, ap.ID)
		}
	}

	// 2. Check and auto-resolve pending active tribulations that exceeded 6 hours
	autoRows, err := db.DB.Query(`
		SELECT id, player_id
		FROM tribulation_events
		WHERE status = 'ACTIVE' AND started_at < NOW() - INTERVAL '6 hours'
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
			// Trigger a mock resolve
			autoResolveEvent(r.EventID, r.PlayerID)
		}
	}

	return nil
}

func spawnTribulationEventWorker(apertureID string, playerID string, tribCount int, currentHP int, newYearsPassed float64, tribulationType string) error {
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
	rewardMarks := 250 + difficulty*50
	switch tribulationType {
	case "HEAVENLY_TRIBULATION":
		rewardMarks = 750 + difficulty*150
	case "GRAND_TRIBULATION":
		rewardMarks = 7250 + difficulty*500
	case "MYRIAD_TRIBULATION":
		rewardMarks = 86750 + difficulty*2000
	case "CHAOS_CALAMITY":
		rewardMarks = 500000 + difficulty*10000
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
	case "CHAOS_CALAMITY":
		bossTemplateID = "monster_chaos_calamity_boss"
	}

	bossRank := 6
	switch tribulationType {
	case "EARTH_CALAMITY", "HEAVENLY_TRIBULATION":
		bossRank = 6
	case "GRAND_TRIBULATION":
		bossRank = 7
	case "MYRIAD_TRIBULATION":
		bossRank = 8
	case "CHAOS_CALAMITY":
		bossRank = 9
	}
	bossSubRank := 1

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
	for i := 0; i < minionCount; i++ {
		scaledHPm, scaledAtkm, scaledDefm := models.ScaleMonsterStats(minion.MaxHP, minion.BaseAttack, minion.BaseDefense, bossRank, bossSubRank, minion.Type)
		mHP := int(scaledHPm * diffMultiplier * daoBalancingMult)
		mAtk := int(scaledAtkm * diffMultiplier * daoBalancingMult)
		mDef := int(scaledDefm * diffMultiplier * daoBalancingMult)

		_, err = tx.Exec(`
			INSERT INTO tribulation_enemies (event_id, name, enemy_type, path, hp, max_hp, attack, defense, x, y, is_dead)
			VALUES ($1, $2, 'MINION', $3, $4, $4, $5, $6, $7, $8, false)
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

	// Fetch current counters from aperture
	var ap models.ImmortalAperture
	_ = tx.QueryRow(`
		SELECT id, aperture_hp, max_aperture_hp, tribulation_count, next_tribulation_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count
		FROM immortal_apertures WHERE id = $1
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.TribulationCount, &ap.NextTribulationYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount,
		&ap.MyriadTribulationCount, &ap.ChaosCalamityCount)
	if err != nil {
		return
	}

	var playerRank int
	_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&playerRank)
	// Compute nextYear using the incremented counts
	nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount,
		ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
	if nextYear < 10 {
		nextYear = 10
	}

	// Revert counts now
	if ap.TribulationCount > 0 {
		ap.TribulationCount--
	}
	switch ev.TribulationType {
	case "EARTH_CALAMITY":
		if ap.EarthCalamityCount > 0 {
			ap.EarthCalamityCount--
		}
	case "HEAVENLY_TRIBULATION":
		if ap.HeavenlyTribulationCount > 0 {
			ap.HeavenlyTribulationCount--
		}
	case "GRAND_TRIBULATION":
		if ap.GrandTribulationCount > 0 {
			ap.GrandTribulationCount--
		}
	case "MYRIAD_TRIBULATION":
		if ap.MyriadTribulationCount > 0 {
			ap.MyriadTribulationCount--
		}
	case "CHAOS_CALAMITY":
		if ap.ChaosCalamityCount > 0 {
			ap.ChaosCalamityCount--
		}
	}

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
			myriad_tribulation_count = $6, chaos_calamity_count = $7, tribulation_count = $8
		WHERE id = $9
	`, nextYear, newHP,
		ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount,
		ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.TribulationCount, ev.ApertureID)

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

// ProcessActiveTribulationGrid moves tribulation enemies towards resource nodes and attacks them
func ProcessActiveTribulationGrid() error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Fetch all active tribulation events
	rows, err := tx.Query(`
		SELECT id, aperture_id, player_id, tribulation_type
		FROM tribulation_events 
		WHERE status = 'ACTIVE'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type ActiveEvent struct {
		EventID         string
		ApertureID      string
		PlayerID        string
		TribulationType string
	}
	var events []ActiveEvent
	for rows.Next() {
		var ev ActiveEvent
		if err := rows.Scan(&ev.EventID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType); err == nil {
			events = append(events, ev)
		}
	}
	rows.Close()

	for _, ev := range events {
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
			continue
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
				_, err = tx.Exec(`
					UPDATE tribulation_enemies 
					SET x = $1, y = $2, updated_at = NOW() 
					WHERE id = $3
				`, newX, newY, en.ID)
				if err != nil {
					return fmt.Errorf("failed to update enemy coordinates: %w", err)
				}
			}
			continue
		}

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
				// Calculate damage buffs/reductions from nearby defense structures
				effectiveDmg := en.NodeDmg
				totalDamageReduction := 0
				totalHPBonus := 0

				for i := range nodes {
					buffNode := &nodes[i]
					if buffNode.ID == nearestNode.ID || buffNode.HP <= 0 {
						continue
					}
					// Manhattan distance from buff node to target node
					bdx := buffNode.X - nearestNode.X
					if bdx < 0 {
						bdx = -bdx
					}
					bdy := buffNode.Y - nearestNode.Y
					if bdy < 0 {
						bdy = -bdy
					}
					bdist := bdx + bdy

					// Guardian Statue: bonus HP to nearby nodes
					if buffNode.HPBonus > 0 && buffNode.AttackRange > 0 && bdist <= buffNode.AttackRange {
						totalHPBonus += buffNode.HPBonus
					}
					// Barracks: damage reduction to ALL nodes (attack_range=0 means global)
					if buffNode.DamageReduction > 0 && buffNode.AttackRange == 0 {
						totalDamageReduction += buffNode.DamageReduction
					}
					// Force Field: damage reduction to nearby nodes
					if buffNode.DamageReduction > 0 && buffNode.AttackRange > 0 && bdist <= buffNode.AttackRange {
						totalDamageReduction += buffNode.DamageReduction
					}
				}

				// Cap damage reduction at 80%
				if totalDamageReduction > 80 {
					totalDamageReduction = 80
				}
				// Apply damage reduction
				effectiveDmg = en.NodeDmg * (100 - totalDamageReduction) / 100
				if effectiveDmg < 1 {
					effectiveDmg = 1
				}

				// Apply damage (HP cannot go below 0, but node survives if HP > 0 even with bonus)
				nearestNode.HP -= effectiveDmg
				isDestroyed := false
				if nearestNode.HP <= 0 {
					nearestNode.HP = 0
					isDestroyed = true
				}
				_, err = tx.Exec(`
					UPDATE aperture_resource_nodes 
					SET hp = $1, is_destroyed = $2, updated_at = NOW() 
					WHERE id = $3
				`, nearestNode.HP, isDestroyed, nearestNode.ID)
				if err != nil {
					return fmt.Errorf("failed to update resource node HP: %w", err)
				}

				if isDestroyed {
					// Send system message
					_, err = tx.Exec(`
						INSERT INTO messages (sender_id, receiver_id, content, channel)
						VALUES ('11111111-1111-1111-1111-111111111111', $1, 'Tai kiếp đã phá hủy hoàn toàn ' || $2 || '!', 'private')
					`, ev.PlayerID, nearestNode.Name)
					if err != nil {
						return fmt.Errorf("failed to insert message for destroyed node: %w", err)
					}

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
				// Move towards the node (allows diagonal movement)
				// Wall block check: if the target cell has a wall, redirect to attack it
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

				// Check if target cell has a wall blocking movement
				var wallNode *TNode
				for i := range nodes {
					nd := &nodes[i]
					if nd.BlocksMovement && nd.X == targetX && nd.Y == targetY && nd.HP > 0 {
						wallNode = nd
						break
					}
				}

				if wallNode != nil {
					// Wall blocks path - attack the wall instead
					nearestNode = wallNode
					// Calculate damage buffs/reductions
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
					_, err = tx.Exec(`
						UPDATE aperture_resource_nodes 
						SET hp = $1, is_destroyed = $2, updated_at = NOW() 
						WHERE id = $3
					`, nearestNode.HP, isDestroyed, nearestNode.ID)
					if err != nil {
						return fmt.Errorf("failed to update wall HP: %w", err)
					}

					if isDestroyed {
						_, err = tx.Exec(`
							INSERT INTO messages (sender_id, receiver_id, content, channel)
							VALUES ('11111111-1111-1111-1111-111111111111', $1, 'Tai kiếp đã phá hủy Tường Thành!', 'private')
						`, ev.PlayerID)
						if err != nil {
							return fmt.Errorf("failed to insert message for destroyed wall: %w", err)
						}
						var remainingNodes []TNode
						for _, nd := range nodes {
							if nd.ID != wallNode.ID {
								remainingNodes = append(remainingNodes, nd)
							}
						}
						nodes = remainingNodes
					}
				} else {
					// No wall blocking, move normally
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
					_, err = tx.Exec(`
						UPDATE tribulation_enemies 
						SET x = $1, y = $2, updated_at = NOW() 
						WHERE id = $3
					`, en.X, en.Y, en.ID)
					if err != nil {
						return fmt.Errorf("failed to update enemy coordinates: %w", err)
					}
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
				// Manhattan distance from tower to enemy
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
					// Tower shoots at enemy - deal defense_power as direct HP damage to enemy
					// We need to update enemy HP in DB (enemies have HP in tribulation_enemies)
					_, err = tx.Exec(`
						UPDATE tribulation_enemies 
						SET hp = GREATEST(hp - $1, 0), 
							is_dead = CASE WHEN hp - $1 <= 0 THEN true ELSE false END,
							updated_at = NOW() 
						WHERE id = $2
					`, tower.DefensePower, en.ID)
					if err != nil {
						return fmt.Errorf("failed to apply watchtower damage: %w", err)
					}
				}
			}
		}
	}

	return tx.Commit()
}

func RegenerateImmortalApertures() error {
	// Heal aperture_hp by 0.5% of max_aperture_hp per minute (minimum 10 HP per minute)
	// If state is DAMAGED and aperture_hp reaches max_aperture_hp, set state back to STABLE
	_, err := db.DB.Exec(`
		UPDATE immortal_apertures
		SET 
			aperture_hp = LEAST(max_aperture_hp, aperture_hp + GREATEST(10, max_aperture_hp / 200)),
			state = CASE 
				WHEN state = 'DAMAGED' AND (aperture_hp + GREATEST(10, max_aperture_hp / 200)) >= max_aperture_hp THEN 'STABLE'
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
