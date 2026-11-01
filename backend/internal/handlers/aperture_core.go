package handlers

import (
	"database/sql"
	"encoding/json"
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
	if isImmortalPlayer && raceHasDeadAperture(player.Race) {
		tx, txErr := db.DB.BeginTx(c.Request.Context(), nil)
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

	// Consolidate updates: only persist real state transitions or periodic checkpoints (>60s)
	// to avoid hammering the database with row-locking UPDATEs on every GET request/poll.
	shouldPersist := ap.State != originalState || ap.ImmortalEssence != originalEssence || (now.Sub(updatedAt) >= 60*time.Second && ap.InternalYearsPassed != originalYears)
	if shouldPersist {
		_, errUpd := db.DB.ExecContext(c.Request.Context(), `
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
	// Seed a few random initial resource points for the new aperture
	if err := GenerateApertureResourcePoints(tx, ap.ID, ap.PlayerID, models.DaoMarksMap{}, ap.Grade, seedPointCountForGrade(ap.Grade)); err != nil {
		logger.S.Warnf("createAperture: seed resource points for %s: %v", ap.ID, err)
	}
	return ap, nil
}

// seedPointCountForGrade mirrors the old starter-node counts.
func seedPointCountForGrade(grade string) int {
	switch grade {
	case "HIGH", "EXTREME":
		return 3
	default:
		return 2
	}
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

// GetApertureNodes returns the DEFENSE structures of the player's aperture
// (walls, watchtowers…). Production no longer lives on this table — it is
// owned by resource points (GET /immortal-aperture/points).
func GetApertureNodes(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.path, n.level, n.hp, n.max_hp, n.x, n.y,
			n.is_destroyed, n.created_at, n.updated_at,
			COALESCE(n.node_category, 'DEFENSE'), COALESCE(n.defense_power, 0), COALESCE(n.attack_range, 0),
			COALESCE(n.hp_bonus, 0), COALESCE(n.damage_reduction, 0), COALESCE(n.blocks_movement, false),
			COALESCE(n.required_gu_id, ''), COALESCE(n.required_gu_rank, 0)
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE a.player_id = $1 AND n.is_destroyed = false
		  AND COALESCE(n.node_category, 'DEFENSE') = 'DEFENSE'
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy danh sách công trình: " + err.Error()})
		return
	}
	defer rows.Close()

	nodes := []models.ApertureResourceNode{}
	for rows.Next() {
		var n models.ApertureResourceNode
		err = rows.Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.Path, &n.Level, &n.HP, &n.MaxHP, &n.X, &n.Y,
			&n.IsDestroyed, &n.CreatedAt, &n.UpdatedAt,
			&n.NodeCategory, &n.DefensePower, &n.AttackRange,
			&n.HPBonus, &n.DamageReduction, &n.BlocksMovement,
			&n.RequiredGuID, &n.RequiredGuRank)
		if err == nil {
			nodes = append(nodes, n)
		}
	}
	c.JSON(http.StatusOK, nodes)
}
