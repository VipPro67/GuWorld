package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func GetImmortalAperture(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 1. Fetch player (read-only query)
	var player models.Player
	err := db.DB.QueryRow(`
		SELECT id, rank, aptitude_percent, human_qi, primeval_stones, immortal_stones, current_mana, time_speed_multiplier, COALESCE(race, '')
		FROM players WHERE id = $1
	`, playerID).Scan(&player.ID, &player.Rank, &player.AptitudePercent, &player.HumanQi, &player.PrimevalStones, &player.ImmortalStones, &player.CurrentMana, &player.TimeSpeedMultiplier, &player.Race)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cổ sư không tồn tại"})
		return
	}

	if player.Rank < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa thăng tiên (Rank 6), chưa có Tiên Khiếu."})
		return
	}

	// 2. Fetch aperture (read-only query)
	var ap models.ImmortalAperture
	var createdAt, updatedAt time.Time
	err = db.DB.QueryRow(`
		SELECT id, player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, state, dao_marks, created_at, updated_at
		FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&ap.ID, &ap.PlayerID, &ap.Grade, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.AreaSize, &ap.ImmortalEssence, &ap.EssenceProductionPerYear, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.State, &ap.DaoMarks, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		// Auto-create aperture (fallback for legacy players) — grade rolled from current Nhân Khí
		tx, txErr := db.DB.Begin()
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		defer tx.Rollback()

		ap, err = createAperture(tx, playerID.String(), player.HumanQi, player.CurrentMana)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khai mở Tiên Khiếu: " + err.Error()})
			return
		}
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu Tiên Khiếu: " + err.Error()})
			return
		}
		createdAt = ap.CreatedAt
		updatedAt = ap.UpdatedAt
	} else if err != nil {
		logger.S.Errorf("Error fetching aperture for player %s: %v", playerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn Tiên Khiếu"})
		return
	}

	ap.TribulationCount = ap.TotalTribulationCount()
	isImmortalPlayer := gamefx.Get(&player).Immortal
	if isImmortalPlayer {
		tx, txErr := db.DB.Begin()
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra trạng thái Tiên Cương"})
			return
		}
		suppressed, suppressErr := suppressTribulationsForDeadAperture(tx, playerID.String())
		if suppressErr != nil {
			_ = tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đồng bộ trạng thái Tiên Cương"})
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu trạng thái Tiên Cương"})
			return
		}
		if suppressed {
			ap.State = "STABLE"
		}
	}

	originalState := ap.State
	originalEssence := ap.ImmortalEssence
	originalYears := ap.InternalYearsPassed

	// Sync current mana in-memory
	ap.ImmortalEssence = int(player.CurrentMana)

	// Verify state consistency: aperture can ONLY be STABLE if there are no active tribulation events
	// and no living tribulation enemies.
	hasActiveTrib, errActive := db.HasActiveTribulationOrEnemies(db.DB, ap.ID)
	if errActive == nil {
		if hasActiveTrib && ap.State == "STABLE" {
			ap.State = "TRIBULATION_ACTIVE"
		} else if !hasActiveTrib && ap.State == "TRIBULATION_ACTIVE" && ap.ApertureHP >= ap.MaxApertureHP {
			ap.State = "STABLE"
		}
	}

	ap.CreatedAt = createdAt
	ap.UpdatedAt = updatedAt

	effectiveTimeRate := 0.0
	if ap.TimeRate > 0 {
		tsm := 1.0
		if !isImmortalPlayer {
			tsm = player.TimeSpeedMultiplier
			if tsm <= 0 {
				tsm = 1.0
			}
			if bonus, err := db.EquippedTimeSpeedBonus(db.DB, playerID.String()); err == nil {
				tsm += bonus
			}
		}
		effectiveTimeRate = ap.TimeRate * tsm
	}

	// Project internal_years_passed forward based on real time elapsed since updated_at.
	// This ensures the frontend always gets accurate time values even if other handlers
	// (e.g. SetApertureTimeRate) update updated_at without projecting internal years.
	now := time.Now()
	if ap.State == "STABLE" || ap.State == "DAMAGED" {
		elapsedSec := now.Sub(updatedAt).Seconds()
		if elapsedSec > 0 && effectiveTimeRate > 0 {
			elapsedInternalYears := elapsedSec * effectiveTimeRate / 36000.0
			ap.InternalYearsPassed += elapsedInternalYears
			ap.UpdatedAt = now
		}
	}

	// Consolidate updates into a single query only if changed
	if ap.State != originalState || ap.ImmortalEssence != originalEssence || ap.InternalYearsPassed != originalYears {
		_, errUpd := db.DB.Exec(`
			UPDATE immortal_apertures
			SET state = $1, immortal_essence = $2, internal_years_passed = $3, updated_at = $4
			WHERE id = $5
		`, ap.State, ap.ImmortalEssence, ap.InternalYearsPassed, ap.UpdatedAt, ap.ID)
		if errUpd != nil {
			logger.S.Warnf("GetImmortalAperture: failed to sync aperture update for %s: %v", ap.ID, errUpd)
		}
	}

	if !isImmortalPlayer {
		if (ap.State == "STABLE" || ap.State == "DAMAGED") && ap.NextTribulationYear > 0 && ap.InternalYearsPassed >= ap.NextTribulationYear {
			_ = checkAndTriggerSingleAperture(ap.ID)
			_ = db.DB.QueryRow(`SELECT state FROM immortal_apertures WHERE id = $1`, ap.ID).Scan(&ap.State)
		}

		if (ap.State == "STABLE" || ap.State == "DAMAGED") && effectiveTimeRate > 0 && ap.NextTribulationYear > ap.InternalYearsPassed {
			remainingInternalYears := ap.NextTribulationYear - ap.InternalYearsPassed
			arrival := ap.UpdatedAt.Add(time.Duration(remainingInternalYears/effectiveTimeRate*36000.0) * time.Second)
			ap.NextTribulationAt = &arrival
		}
	} else {
		ap.NextTribulationAt = nil
	}

	c.JSON(http.StatusOK, ap)
}

func seedInitialApertureNodes(tx *sql.Tx, apertureID string, grade string) {
	type StarterConfig struct {
		NodeType     string
		Name         string
		Path         string
		ProdKey      string
		ZoneType     string
		ProdType     string
		ProdAmount   int
		ProdInterval int
	}

	starterConfigs := []StarterConfig{
		{NodeType: "FARMING_ZONE", Name: "Khu Trồng Trọt (Rượu Thanh Trúc)", Path: "Wood Path", ProdKey: "green_bamboo_wine", ZoneType: "khu_trong", ProdType: "Material", ProdAmount: 10, ProdInterval: 10},
		{NodeType: "BREEDING_ZONE", Name: "Khu Chăn Nuôi (Thịt Hươu Linh)", Path: "Food Path", ProdKey: "spirit_deer_meat", ZoneType: "khu_nuoi", ProdType: "Material", ProdAmount: 5, ProdInterval: 10},
		{NodeType: "MINING_ZONE", Name: "Khu Khai Thác (Ngọc Thạch)", Path: "Earth Path", ProdKey: "jade_stone", ZoneType: "khu_khai_thac", ProdType: "Material", ProdAmount: 3, ProdInterval: 10},
	}

	count := 2
	switch grade {
	case "MID":
		count = 2
	case "HIGH", "EXTREME":
		count = 3
	}

	used := make(map[string]bool)
	for i := 0; i < count && i < len(starterConfigs); i++ {
		cfg := starterConfigs[i]

		x, y := rand.Intn(20), rand.Intn(20)
		key := fmt.Sprintf("%d,%d", x, y)
		for used[key] {
			x, y = rand.Intn(20), rand.Intn(20)
			key = fmt.Sprintf("%d,%d", x, y)
		}
		used[key] = true

		_, _ = tx.Exec(`
			INSERT INTO aperture_resource_nodes
				(aperture_id, node_type, name, path, level, hp, max_hp, x, y,
				 production_type, production_amount, production_interval_hours, last_collected_at,
				 node_category, defense_power, attack_range, hp_bonus, damage_reduction, blocks_movement,
				 required_gu_id, required_gu_rank, zone_type, food_path, item_tier, production_key)
			VALUES ($1, $2, $3, $4, 1, 1000, 1000, $5, $6, $7, $8, $9, NOW(),
				'RESOURCE', 0, 0, 0, 0, false,
				'', 0, $10, '', 'T1', $11)
		`, apertureID, cfg.NodeType, cfg.Name, cfg.Path, x, y,
			cfg.ProdType, cfg.ProdAmount, cfg.ProdInterval,
			cfg.ZoneType, cfg.ProdKey)
	}
}

// rollApertureGrade rolls the Tiên Khiếu grade based on Nhân Khí ratio q (0..1).
// Higher Nhân Khí increases the chance of higher grades (weights sum to a constant 21).
func rollApertureGrade(q float64) string {
	wExtreme := 1.0 + 9.0*q
	wHigh := 3.0 + 3.0*q
	wMid := 5.0
	roll := rand.Float64() * 21.0
	switch {
	case roll < wExtreme:
		return "EXTREME"
	case roll < wExtreme+wHigh:
		return "HIGH"
	case roll < wExtreme+wHigh+wMid:
		return "MID"
	default:
		return "LOW"
	}
}

// createAperture inserts a brand new Tiên Khiếu for a newly ascended player,
// rolls its grade from the Nhân Khí they had at ascension, and seeds starter nodes.
func createAperture(tx *sql.Tx, playerID string, humanQi, currentMana float64) (models.ImmortalAperture, error) {
	q := math.Min(1.0, humanQi/100000.0)
	ap := buildApertureByGrade(playerID, rollApertureGrade(q))
	ap.ImmortalEssence = int(currentMana)
	err := tx.QueryRow(`
		INSERT INTO immortal_apertures (player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, ap.PlayerID, ap.Grade, ap.TimeRate, ap.ApertureHP, ap.MaxApertureHP, ap.AreaSize, ap.ImmortalEssence, ap.EssenceProductionPerYear, ap.InternalYearsPassed, ap.NextTribulationYear, ap.State).Scan(&ap.ID, &ap.CreatedAt, &ap.UpdatedAt)
	if err != nil {
		return models.ImmortalAperture{}, err
	}
	// Seed a few random initial resource nodes for the new aperture
	seedInitialApertureNodes(tx, ap.ID, ap.Grade)
	return ap, nil
}

func buildApertureByGrade(playerID string, grade string) models.ImmortalAperture {
	switch grade {
	case "EXTREME":
		timeRate := 40.0 + float64(rand.Intn(16)) // 40 to 55
		return models.ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 timeRate,
			ApertureHP:               50000,
			MaxApertureHP:            50000,
			AreaSize:                 10000000,
			EssenceProductionPerYear: 50,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	case "HIGH":
		timeRate := 30.0 + float64(rand.Intn(11)) // 30 to 40
		return models.ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 timeRate,
			ApertureHP:               30000,
			MaxApertureHP:            30000,
			AreaSize:                 9000000,
			EssenceProductionPerYear: 30,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	case "MID":
		timeRate := 20.0 + float64(rand.Intn(9)) // 20 to 28
		return models.ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 timeRate,
			ApertureHP:               20000,
			MaxApertureHP:            20000,
			AreaSize:                 6000000,
			EssenceProductionPerYear: 20,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	default: // "LOW"
		timeRate := 10.0 + float64(rand.Intn(6)) // 10 to 15
		return models.ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 timeRate,
			ApertureHP:               10000,
			MaxApertureHP:            10000,
			AreaSize:                 3000000,
			EssenceProductionPerYear: 10,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	}
}

// GetDaoMarks returns player's Dao Marks
func GetDaoMarks(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var daoJSON []byte
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&daoJSON)
	if err == sql.ErrNoRows {
		daoJSON = []byte("{}")
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Dao Marks"})
		return
	}

	daoMap := make(map[string]int)
	if len(daoJSON) > 0 {
		_ = json.Unmarshal(daoJSON, &daoMap)
	}

	marks := make([]models.DaoMark, 0, len(daoMap))
	for path, amount := range daoMap {
		if amount <= 0 {
			continue
		}
		marks = append(marks, models.DaoMark{
			ID:       playerID.String() + ":" + path,
			PlayerID: playerID.String(),
			Path:     path,
			Amount:   int64(amount),
		})
	}
	sort.Slice(marks, func(i, j int) bool {
		return marks[i].Amount > marks[j].Amount
	})

	c.JSON(http.StatusOK, marks)
}

// GetApertureNodes returns resource nodes in player's aperture
func GetApertureNodes(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.path, n.level, n.hp, n.max_hp, n.x, n.y,
			n.production_type, n.production_amount, n.production_interval_hours, n.last_collected_at,
			n.is_destroyed, n.created_at, n.updated_at,
			COALESCE(n.node_category, 'RESOURCE'), COALESCE(n.defense_power, 0), COALESCE(n.attack_range, 0),
			COALESCE(n.hp_bonus, 0), COALESCE(n.damage_reduction, 0), COALESCE(n.blocks_movement, false),
			COALESCE(n.required_gu_id, ''), COALESCE(n.required_gu_rank, 0),
			COALESCE(n.zone_type, ''), COALESCE(n.food_path, ''), COALESCE(n.item_tier, '')
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE a.player_id = $1 AND n.is_destroyed = false
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy danh sách điểm tài nguyên: " + err.Error()})
		return
	}
	defer rows.Close()

	nodes := []models.ApertureResourceNode{}
	for rows.Next() {
		var n models.ApertureResourceNode
		var lastCol sql.NullTime
		err = rows.Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.Path, &n.Level, &n.HP, &n.MaxHP, &n.X, &n.Y,
			&n.ProductionType, &n.ProductionAmount, &n.ProductionIntervalHours, &lastCol,
			&n.IsDestroyed, &n.CreatedAt, &n.UpdatedAt,
			&n.NodeCategory, &n.DefensePower, &n.AttackRange,
			&n.HPBonus, &n.DamageReduction, &n.BlocksMovement,
			&n.RequiredGuID, &n.RequiredGuRank,
			&n.ZoneType, &n.FoodPath, &n.ItemTier)
		if err == nil {
			if lastCol.Valid {
				n.LastCollectedAt = &lastCol.Time
			}
			nodes = append(nodes, n)
		}
	}

	c.JSON(http.StatusOK, nodes)

}
