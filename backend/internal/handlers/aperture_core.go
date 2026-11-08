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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
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
		SELECT id, player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, state, dao_marks, COALESCE(heaven_qi, 100000), COALESCE(earth_qi, 100000), COALESCE(max_heaven_qi, 100000), COALESCE(max_earth_qi, 100000), COALESCE(is_placed, false), placed_region, placed_x, placed_y, placed_at, created_at, updated_at
		FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&ap.ID, &ap.PlayerID, &ap.Grade, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.AreaSize, &ap.ImmortalEssence, &ap.EssenceProductionPerYear, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.State, &ap.DaoMarks, &ap.HeavenQi, &ap.EarthQi, &ap.MaxHeavenQi, &ap.MaxEarthQi, &ap.IsPlaced, &ap.PlacedRegion, &ap.PlacedX, &ap.PlacedY, &ap.PlacedAt, &createdAt, &updatedAt)

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
		if ap.IsPlaced {
			// Hạ khiếu: nhánh sông Quang Âm hòa nhập ngoại giới -> giảm tốc về 1.0 (hoãn kiếp nạn, giảm tốc già hóa)
			effectiveTimeRate = 1.0
		}
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

	// Tam Khí Cân Bằng: Nhân Khí lúc Thăng Tiên định hình trần Thiên Khí và Địa Khí khởi đầu!
	initialQi := math.Max(100000.0, humanQi)
	ap.MaxHeavenQi = initialQi
	ap.MaxEarthQi = initialQi
	ap.HeavenQi = initialQi
	ap.EarthQi = initialQi

	err := tx.QueryRow(`
		INSERT INTO immortal_apertures (player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, state, heaven_qi, earth_qi, max_heaven_qi, max_earth_qi)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, ap.PlayerID, ap.Grade, ap.TimeRate, ap.ApertureHP, ap.MaxApertureHP, ap.AreaSize, ap.ImmortalEssence, ap.EssenceProductionPerYear, ap.InternalYearsPassed, ap.NextTribulationYear, ap.State, ap.HeavenQi, ap.EarthQi, ap.MaxHeavenQi, ap.MaxEarthQi).Scan(&ap.ID, &ap.CreatedAt, &ap.UpdatedAt)
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

// AnchorAperture places (anchors) an immortal aperture onto a world map cell.
// According to Reverend Insanity canon, the host and aperture are unified:
// anchoring roots the aperture into the world, spawns an active World Gateway (Môn Hộ),
// and places the host INSIDE their aperture (they cannot roam the outer world while anchored).
func AnchorAperture(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		PortalName   string `json:"portal_name"`
		AccessPolicy string `json:"access_policy"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.PortalName == "" {
		req.PortalName = "Môn Hộ Tiên Khiếu"
	}
	validPolicies := map[string]bool{
		"OPEN": true, "PARTY": true, "SECT": true, "KEY": true, "APPROVAL": true, "HIDDEN": true,
	}
	if !validPolicies[req.AccessPolicy] {
		req.AccessPolicy = "OPEN"
	}

	var inCombatSession bool
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM realtime_combat_session_participants
			WHERE player_id = $1 AND status = 'active'
		)
	`, playerID).Scan(&inCombatSession)
	if inCombatSession {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể Hạ Khiếu khi đang trong trận chiến thời gian thực."})
		return
	}

	handedOff, handoffErr := middleware.BeginRealtimeHandoff(playerID)
	if handoffErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Không thể tạm dừng điều khiển thời gian thực, vui lòng thử lại."})
		return
	}
	var realtimeFinished bool
	if handedOff {
		defer func() {
			if !realtimeFinished {
				middleware.AbortRealtimeHandoff(playerID)
			}
		}()
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var player models.Player
	err = tx.QueryRow(`
		SELECT id, rank, COALESCE(current_region, ''), COALESCE(x, 500), COALESCE(y, 500)
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&player.ID, &player.Rank, &player.CurrentRegion, &player.X, &player.Y)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cổ sư không tồn tại"})
		return
	}

	if player.Rank < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ Cổ Tiên (Rank 6+) mới có thể Hạ Khiếu."})
		return
	}

	if liveRegion, liveX, liveY, ok := middleware.RealtimePlayerLocation(playerID); ok && liveRegion != "" {
		player.CurrentRegion = liveRegion
		player.X = liveX
		player.Y = liveY
	}

	if player.CurrentRegion == "aperture_interior" || strings.HasPrefix(player.CurrentRegion, "aperture:") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể Hạ Khiếu khi đang ở bên trong một Tiên Khiếu khác."})
		return
	}

	var ap models.ImmortalAperture
	err = tx.QueryRow(`
		SELECT id, is_placed, state, heaven_qi, earth_qi, max_heaven_qi, max_earth_qi
		FROM immortal_apertures WHERE player_id = $1 FOR UPDATE
	`, playerID).Scan(&ap.ID, &ap.IsPlaced, &ap.State, &ap.HeavenQi, &ap.EarthQi, &ap.MaxHeavenQi, &ap.MaxEarthQi)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chưa khai mở Tiên Khiếu"})
		return
	}

	if ap.IsPlaced {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu đang trong trạng thái Hạ Khiếu."})
		return
	}

	if ap.State == "DESTROYED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu đã bị hủy diệt, không thể Hạ Khiếu."})
		return
	}

	// 1. Update immortal_apertures placement and record return location
	_, err = tx.Exec(`
		UPDATE immortal_apertures
		SET is_placed = true, placed_region = $1, placed_x = $2, placed_y = $3, placed_at = NOW(),
		    return_region = $1, return_x = $2, return_y = $3, updated_at = NOW()
		WHERE id = $4
	`, player.CurrentRegion, float64(player.X), float64(player.Y), ap.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái Hạ Khiếu"})
		return
	}

	// 2. Upsert World Gateway (Môn Hộ)
	_, err = tx.Exec(`
		INSERT INTO aperture_portals (aperture_id, owner_id, region_id, x, y, name, access_policy, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW())
		ON CONFLICT (aperture_id) DO UPDATE
		SET region_id = EXCLUDED.region_id, x = EXCLUDED.x, y = EXCLUDED.y, name = EXCLUDED.name,
		    access_policy = EXCLUDED.access_policy, is_active = true, updated_at = NOW()
	`, ap.ID, player.ID, player.CurrentRegion, float64(player.X), float64(player.Y), req.PortalName, req.AccessPolicy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khởi tạo Môn Hộ thế giới: " + err.Error()})
		return
	}

	// 3. Move host body inside the aperture instance (Host cannot be outside while anchored)
	apertureInstanceID := fmt.Sprintf("aperture:%s", ap.ID)
	_, err = tx.Exec(`
		UPDATE players
		SET current_region = 'aperture_interior', map_instance_id = $1, x = 10, y = 10,
		    world_zone_id = 'aperture_interior', world_instance_id = $2,
		    world_x = 10, world_y = 10, world_safe_x = 10, world_safe_y = 10,
		    world_location_revision = world_location_revision + 1, version = version + 1
		WHERE id = $3
	`, ap.ID, apertureInstanceID, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển dịch thân thể vào Tiên Khiếu: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu dữ liệu Hạ Khiếu"})
		return
	}

	if handedOff {
		middleware.FinishRealtimeHandoff(playerID)
		realtimeFinished = true
	}
	apUUID, _ := uuid.Parse(ap.ID)
	syncSpatialPlayer(playerID, "aperture_interior", 10, 10, apUUID)

	_ = cache.PublishEventQuick("world:cell", map[string]any{
		"type":       "portal_spawn",
		"region_id":  player.CurrentRegion,
		"x":          player.X,
		"y":          player.Y,
		"cell_type":  "AperturePortal",
		"content_id": ap.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Hạ Khiếu thành công tại %s (%d, %d). Môn Hộ đã hiển hiện ngoài thế giới. Thân thể Cổ Tiên đã bước vào tọa trấn bên trong Tiên Khiếu, dòng sông Quang Âm đồng bộ với ngoại giới (1.0x). Đang hấp thụ Thiên Khí và Địa Khí...",
			player.CurrentRegion, player.X, player.Y),
		"is_placed":        true,
		"placed_region":    player.CurrentRegion,
		"placed_x":         player.X,
		"placed_y":         player.Y,
		"aperture_region":  "aperture_interior",
		"aperture_instance": apertureInstanceID,
		"portal_name":      req.PortalName,
		"access_policy":    req.AccessPolicy,
	})
}

// UnanchorAperture retracts the immortal aperture back into the immortal's body,
// concluding the anchoring session and restoring the host back to the world map.
func UnanchorAperture(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var inCombatSession bool
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM realtime_combat_session_participants
			WHERE player_id = $1 AND status = 'active'
		)
	`, playerID).Scan(&inCombatSession)
	if inCombatSession {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể thu khiếu khi đang trong trận chiến thời gian thực."})
		return
	}

	handedOff, handoffErr := middleware.BeginRealtimeHandoff(playerID)
	if handoffErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Không thể tạm dừng điều khiển thời gian thực, vui lòng thử lại."})
		return
	}
	var realtimeFinished bool
	if handedOff {
		defer func() {
			if !realtimeFinished {
				middleware.AbortRealtimeHandoff(playerID)
			}
		}()
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var ap models.ImmortalAperture
	var placedAt *time.Time
	var placedRegion sql.NullString
	var placedX, placedY sql.NullFloat64
	err = tx.QueryRow(`
		SELECT id, is_placed, state, heaven_qi, earth_qi, max_heaven_qi, max_earth_qi, placed_at,
		       placed_region, placed_x, placed_y
		FROM immortal_apertures WHERE player_id = $1 FOR UPDATE
	`, playerID).Scan(&ap.ID, &ap.IsPlaced, &ap.State, &ap.HeavenQi, &ap.EarthQi, &ap.MaxHeavenQi, &ap.MaxEarthQi, &placedAt,
		&placedRegion, &placedX, &placedY)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chưa khai mở Tiên Khiếu"})
		return
	}

	if !ap.IsPlaced {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu hiện không trong trạng thái Hạ Khiếu."})
		return
	}

	// Safety Lock 1: Check if there are active guests inside the aperture
	var guestCount int
	err = tx.QueryRow(`SELECT COUNT(1) FROM aperture_guest_roster WHERE aperture_id = $1`, ap.ID).Scan(&guestCount)
	if err == nil && guestCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể thu khiếu khi còn %d ngoại nhân bên trong Tiên Khiếu (Nguy cơ Đạo Ngân xé rách vách ngăn).", guestCount)})
		return
	}

	// Safety Lock 3: Check if active tribulation or enemies
	hasTrib, err := db.HasActiveTribulationOrEnemies(tx, ap.ID)
	if err == nil && hasTrib {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thu khiếu khi đang trong quá trình độ kiếp."})
		return
	}

	elapsedMinutes := 0.0
	if placedAt != nil {
		elapsedMinutes = time.Since(*placedAt).Minutes()
		if elapsedMinutes < 0 {
			elapsedMinutes = 0
		}
	}

	// Nạp Tam Khí: mỗi phút hạ khiếu nạp lại +1% trần max_heaven_qi và max_earth_qi
	gainedHeavenQi := elapsedMinutes * (ap.MaxHeavenQi * 0.01)
	gainedEarthQi := elapsedMinutes * (ap.MaxEarthQi * 0.01)

	newHeavenQi := math.Min(ap.MaxHeavenQi, ap.HeavenQi+gainedHeavenQi)
	newEarthQi := math.Min(ap.MaxEarthQi, ap.EarthQi+gainedEarthQi)

	newState := ap.State
	if newState == "UNBALANCED" && newHeavenQi > 0 && newEarthQi > 0 {
		newState = "STABLE"
	}

	// Deactivate World Gateway
	_, _ = tx.Exec(`
		UPDATE aperture_portals
		SET is_active = false, updated_at = NOW()
		WHERE aperture_id = $1
	`, ap.ID)

	// Update immortal_apertures (retract into host body)
	_, err = tx.Exec(`
		UPDATE immortal_apertures
		SET is_placed = false, placed_region = NULL, placed_x = NULL, placed_y = NULL, placed_at = NULL,
		    heaven_qi = $1, earth_qi = $2, state = $3, updated_at = NOW()
		WHERE id = $4
	`, newHeavenQi, newEarthQi, newState, ap.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thu hồi Tiên Khiếu"})
		return
	}

	// Move host back to the world map at the placement location
	destRegion := "Southern_Border"
	destX := 500
	destY := 500
	if placedRegion.Valid && placedRegion.String != "" {
		destRegion = placedRegion.String
	}
	if placedX.Valid {
		destX = int(placedX.Float64)
	}
	if placedY.Valid {
		destY = int(placedY.Float64)
	}

	_, err = tx.Exec(`
		UPDATE players
		SET current_region = $1, map_instance_id = NULL, x = ($2)::int, y = ($3)::int, world_zone_id = $1, world_instance_id = '',
		    world_x = ($2)::double precision, world_y = ($3)::double precision, world_safe_x = ($2)::double precision, world_safe_y = ($3)::double precision,
		    world_location_revision = world_location_revision + 1, version = version + 1
		WHERE id = $4
	`, destRegion, destX, destY, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hoàn nguyên thân thể về thế giới: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu dữ liệu thu hồi"})
		return
	}

	if handedOff {
		middleware.FinishRealtimeHandoff(playerID)
		realtimeFinished = true
	}
	syncSpatialPlayer(playerID, destRegion, destX, destY, uuid.Nil)

	_ = cache.PublishEventQuick("world:cell", map[string]any{
		"type":       "portal_despawn",
		"region_id":  destRegion,
		"x":          destX,
		"y":          destY,
		"cell_type":  "AperturePortal",
		"content_id": ap.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Thu Khiếu thành công! Môn Hộ thế giới đã khép lại. Tiên Khiếu đã thu hồi vào nhục thân. Đã nạp +%.0f Thiên Khí và +%.0f Địa Khí. Tốc độ thời gian Tiên Khiếu đã khôi phục bình thường. Thân thể Cổ Tiên đã trở lại %s (%d, %d).",
			gainedHeavenQi, gainedEarthQi, destRegion, destX, destY),
		"heaven_qi":       newHeavenQi,
		"earth_qi":        newEarthQi,
		"max_heaven_qi":   ap.MaxHeavenQi,
		"max_earth_qi":    ap.MaxEarthQi,
		"state":           newState,
		"is_placed":       false,
		"current_region":  destRegion,
		"x":               destX,
		"y":               destY,
	})
}
