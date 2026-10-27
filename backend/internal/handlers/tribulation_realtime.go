package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math"
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

// RealtimeApertureNode represents an authored resource node on the 20x20 Aperture grid.
type RealtimeApertureNode struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	NodeType       string `json:"node_type"`
	Path           string `json:"path"`
	X              int    `json:"x"`
	Y              int    `json:"y"`
	HP             int    `json:"hp"`
	MaxHP          int    `json:"max_hp"`
	DefensePower   int    `json:"defense_power"`
	BlocksMovement bool   `json:"blocks_movement"`
}

// RealtimeTribulationEnemy represents an active tribulation enemy on the Aperture map.
type RealtimeTribulationEnemy struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	EnemyType string  `json:"enemy_type"`
	Path      string  `json:"path"`
	HP        float64 `json:"hp"`
	MaxHP     float64 `json:"max_hp"`
	Attack    float64 `json:"attack"`
	Defense   float64 `json:"defense"`
	X         int     `json:"x"`
	Y         int     `json:"y"`
	IsBoss    bool    `json:"is_boss"`
	Rank      int     `json:"rank"`
	SubRank   int     `json:"sub_rank"`
}

// RealtimeEngageTribulation loads the player's 20x20 Aperture nodes and active tribulation
// enemies for a real-time combat session in the Immortal Aperture.
func RealtimeEngageTribulation(c *gin.Context) {
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

	// 1. Fetch active event
	var ev models.TribulationEvent
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, aperture_id, player_id, tribulation_type, path, status, aperture_hp_before, reward_dao_marks, started_at
		FROM tribulation_events
		WHERE id = $1 AND player_id = $2 AND status = 'ACTIVE'
	`, eventID, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.Status, &ev.ApertureHPBefore, &ev.RewardDaoMarks, &ev.StartedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đợt tai kiếp này đã kết thúc hoặc không tồn tại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn tai kiếp: " + err.Error()})
		return
	}

	// 2. Fetch alive enemies
	enemyRows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, name, enemy_type, path, hp, max_hp, attack, defense, x, y, is_boss, rank, sub_rank
		FROM tribulation_enemies
		WHERE event_id = $1 AND is_dead = false
		ORDER BY is_boss DESC, max_hp DESC
	`, ev.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy danh sách quái tai kiếp"})
		return
	}
	defer enemyRows.Close()

	var enemies []RealtimeTribulationEnemy
	for enemyRows.Next() {
		var en RealtimeTribulationEnemy
		if err := enemyRows.Scan(&en.ID, &en.Name, &en.EnemyType, &en.Path, &en.HP, &en.MaxHP, &en.Attack, &en.Defense, &en.X, &en.Y, &en.IsBoss, &en.Rank, &en.SubRank); err == nil {
			enemies = append(enemies, en)
		}
	}

	// 3. Fetch active resource nodes in the player's Immortal Aperture
	nodeRows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, name, node_type, COALESCE(path, ''), x, y, hp, max_hp, defense_power, blocks_movement
		FROM aperture_resource_nodes
		WHERE aperture_id = $1 AND is_destroyed = false
		ORDER BY y ASC, x ASC
	`, ev.ApertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy danh sách tài nguyên Tiên Khiếu"})
		return
	}
	defer nodeRows.Close()

	var nodes []RealtimeApertureNode
	for nodeRows.Next() {
		var n RealtimeApertureNode
		if err := nodeRows.Scan(&n.ID, &n.Name, &n.NodeType, &n.Path, &n.X, &n.Y, &n.HP, &n.MaxHP, &n.DefensePower, &n.BlocksMovement); err == nil {
			nodes = append(nodes, n)
		}
	}

	// 4. Record current world position before entering Aperture (if not already inside)
	var currRegion string
	var currX, currY float64
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT COALESCE(current_region, 'hub_nam_cuong'), COALESCE(world_x, x::double precision, 500), COALESCE(world_y, y::double precision, 500)
		FROM players WHERE id = $1
	`, playerID).Scan(&currRegion, &currX, &currY)

	if currRegion != "" && currRegion != "aperture_tribulation" {
		_, _ = db.DB.ExecContext(c.Request.Context(), `
			UPDATE immortal_apertures
			SET return_region = $1, return_x = $2, return_y = $3
			WHERE id = $4
		`, currRegion, currX, currY, ev.ApertureID)
	}

	// 5. Update player's region to aperture_tribulation
	_, err = db.DB.ExecContext(c.Request.Context(), `
		UPDATE players
		SET current_region = 'aperture_tribulation', world_zone_id = 'aperture_tribulation', world_x = 10, world_y = 10, x = 10, y = 10
		WHERE id = $1
	`, playerID)
	if err != nil {
		logger.S.Errorf("Failed to update player %s region to aperture_tribulation: %v", playerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật khu vực người chơi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":         ev.ID,
		"aperture_id":      ev.ApertureID,
		"tribulation_type": ev.TribulationType,
		"path":             ev.Path,
		"reward_dao_marks": ev.RewardDaoMarks,
		"zone_id":          "aperture_tribulation",
		"map_version":      "regions-v1",
		"content_version":  "region-content-v1",
		"player_spawn":     gin.H{"x": 10.0, "y": 10.0},
		"nodes":            nodes,
		"enemies":          enemies,
	})
}

// RealtimeTribulationVictoryRequest is the payload sent when a player defeats all
// tribulation enemies in the real-time Aperture map.
type RealtimeTribulationVictoryRequest struct {
	NodeHP map[string]int `json:"node_hp"` // node_id -> remaining HP
}

// RealtimeResolveTribulationVictory marks the tribulation victory, updates resource node
// damage, awards Dao Marks, updates Aperture state, and checks for rank breakthroughs.
func RealtimeResolveTribulationVictory(c *gin.Context) {
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

	var req RealtimeTribulationVictoryRequest
	_ = c.ShouldBindJSON(&req)

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch active event
	var ev models.TribulationEvent
	err = tx.QueryRow(`
		SELECT id, aperture_id, player_id, tribulation_type, path, reward_dao_marks, started_at
		FROM tribulation_events
		WHERE id = $1 AND player_id = $2 AND status = 'ACTIVE'
	`, eventID, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.RewardDaoMarks, &ev.StartedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Đợt tai kiếp này đã kết thúc hoặc không tồn tại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn tai kiếp"})
		return
	}

	// 2. Mark all enemies as defeated
	_, err = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái quái tai kiếp"})
		return
	}

	// 3. Update remaining HP of resource nodes and mark destroyed if HP <= 0
	destroyedCount := 0
	if req.NodeHP != nil {
		for nodeIDStr, currentHP := range req.NodeHP {
			nodeID, parseErr := uuid.Parse(nodeIDStr)
			if parseErr != nil {
				continue
			}
			isDestroyed := currentHP <= 0
			if currentHP < 0 {
				currentHP = 0
			}
			if isDestroyed {
				destroyedCount++
			}
			_, _ = tx.Exec(`
				UPDATE aperture_resource_nodes
				SET hp = $1, is_destroyed = $2, updated_at = NOW()
				WHERE id = $3 AND aperture_id = $4
			`, currentHP, isDestroyed, nodeID, ev.ApertureID)
		}
	}

	// 4. Fetch Aperture state
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

	// 5. Increment calamity counters and check breakthrough
	switch ev.TribulationType {
	case "EARTH_CALAMITY":
		ap.EarthCalamityCount++
	case "HEAVENLY_TRIBULATION":
		ap.HeavenlyTribulationCount++
	case "GRAND_TRIBULATION":
		ap.GrandTribulationCount++
	case "MYRIAD_TRIBULATION":
		ap.MyriadTribulationCount++
	case "CHAOS_CALAMITY", "CHAOTIC_TRIBULATION":
		ap.ChaosCalamityCount++
	}

	var playerRank int
	var username string
	_ = tx.QueryRow(`SELECT rank, username FROM players WHERE id = $1`, playerID).Scan(&playerRank, &username)

	didBreakthrough := false
	breakthroughMsg := ""
	canBreak, newRank := models.CheckImmortalBreakthrough(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
	if canBreak {
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
			ap.EarthCalamityCount = 0
			ap.HeavenlyTribulationCount = 0
			ap.GrandTribulationCount = 0
			ap.MyriadTribulationCount = 0
			ap.ChaosCalamityCount = 0
			didBreakthrough = true
			breakthroughMsg = fmt.Sprintf(" Đột phá! Đạo hữu đã tiến vào %s!", models.GetRankName(newRank))

			if newRank >= 7 {
				_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Chúc mừng đạo hữu %s đã độ kiếp thành công thăng cấp %s!", username, models.GetRankName(newRank)))
				auditlog.Log(tx, auditlog.Entry{
					PlayerID:       &playerID,
					Action:         "player.breakthrough_success",
					EntityType:     "player",
					EntityID:       playerID.String(),
					RequestSummary: fmt.Sprintf("Thăng rank %d (%s) thành công qua độ kiếp realtime", newRank, models.GetRankName(newRank)),
					Success:        true,
				})
			}
			playerRank = newRank
		}
	}

	nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, ap.InternalYearsPassed)
	ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)
	ap.ApertureHP = ap.MaxApertureHP
	ap.State = "STABLE"

	// 6. Award Dao Marks
	rewards := distributeDaoMarks(ev.Path, ev.RewardDaoMarks)
	updatedDaoMarks, errDao := addMultipleDaoMarksTx(tx, playerID.String(), rewards)
	if errDao != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Đạo Ngân: " + errDao.Error()})
		return
	}

	now := time.Now()
	_, _ = tx.Exec(`UPDATE tribulation_events SET status = 'SUCCESS', aperture_hp_after = $1, ended_at = $2 WHERE id = $3`, ap.ApertureHP, now, ev.ID)
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

	// 7. Clear any lingering active combat and return player to original position or placed aperture entrance
	exitRegion, exitX, exitY := resolveApertureExitLocation(c.Request.Context(), playerID)
	_, _ = tx.Exec(`
		UPDATE players 
		SET active_combat = NULL, current_region = $1, world_zone_id = $1, world_x = $2, world_y = $3, x = $4, y = $5 
		WHERE id = $6
	`, exitRegion, exitX, exitY, int(exitX), int(exitY), playerID)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit giao dịch"})
		return
	}

	rewardStrings := []string{}
	for path, amount := range rewards {
		rewardStrings = append(rewardStrings, fmt.Sprintf("%d Đạo Ngân hệ %s", amount, strings.ToUpper(path)))
	}
	rewardText := strings.Join(rewardStrings, ", ")
	msg := fmt.Sprintf("Vượt kiếp thành công trong Tiên Khiếu! Thu hoạch %s. Tiên Khiếu đã ổn định trở lại.", rewardText)
	if destroyedCount > 0 {
		msg += fmt.Sprintf(" (Có %d điểm tài nguyên bị tổn hại nặng)", destroyedCount)
	}
	msg += breakthroughMsg

	logger.S.Infof("Player %s completed realtime tribulation %s (Breakthrough: %v)", playerID, ev.ID, didBreakthrough)

	c.JSON(http.StatusOK, gin.H{
		"message":           msg,
		"rewards":           rewards,
		"updated_dao_marks": updatedDaoMarks,
		"did_breakthrough":  didBreakthrough,
		"new_rank":          playerRank,
		"destroyed_nodes":   destroyedCount,
		"return_region":     exitRegion,
		"return_x":          exitX,
		"return_y":          exitY,
	})
}

// resolveApertureExitLocation determines where the player exits to:
// If aperture is placed in world (is_placed == true), exit to (placed_region, placed_x, placed_y).
// Otherwise, exit to return_region (saved before entering Aperture), fallback to hub_nam_cuong.
func resolveApertureExitLocation(ctx context.Context, playerID uuid.UUID) (region string, x float64, y float64) {
	region = "hub_nam_cuong"
	x, y = 500, 500

	var isPlaced bool
	var placedRegion, returnRegion sql.NullString
	var placedX, placedY, returnX, returnY sql.NullFloat64

	err := db.DB.QueryRowContext(ctx, `
		SELECT is_placed, placed_region, placed_x, placed_y, return_region, return_x, return_y
		FROM immortal_apertures
		WHERE player_id = $1
	`, playerID).Scan(&isPlaced, &placedRegion, &placedX, &placedY, &returnRegion, &returnX, &returnY)

	if err == nil {
		if isPlaced && placedRegion.Valid && placedRegion.String != "" {
			region = placedRegion.String
			if placedX.Valid {
				x = placedX.Float64
			}
			if placedY.Valid {
				y = placedY.Float64
			}
			return
		}
		if returnRegion.Valid && returnRegion.String != "" {
			region = returnRegion.String
			if returnX.Valid {
				x = returnX.Float64
			}
			if returnY.Valid {
				y = returnY.Float64
			}
			return
		}
	}
	return
}

// RealtimeEscapeTribulation teleports player out of the tribulation map back to safe exit position.
func RealtimeEscapeTribulation(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	exitRegion, exitX, exitY := resolveApertureExitLocation(c.Request.Context(), playerID)
	_, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE players
		SET current_region = $1, world_zone_id = $1, world_x = $2, world_y = $3, x = $4, y = $5
		WHERE id = $6
	`, exitRegion, exitX, exitY, int(exitX), int(exitY), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi rút lui khỏi Tiên Khiếu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Đã an toàn rời khỏi Tiên Khiếu trở về thế giới thực!",
		"return_region": exitRegion,
		"return_x":      exitX,
		"return_y":      exitY,
	})
}
