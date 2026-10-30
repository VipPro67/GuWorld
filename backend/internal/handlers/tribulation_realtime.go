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
	"github.com/lib/pq"
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

// beginTribulationHandoff coordinates one tribulation map transition with an
// existing realtime lease. It flushes the live position before the handler
// writes the new region, so the old zone's disconnect checkpoint cannot revert
// the transition to the previous map. abort is safe to defer; finish must run
// after the database write commits. A non-zero status rejects the request.
func beginTribulationHandoff(playerID uuid.UUID) (finish, abort func(), realtime bool, status int, code, message string) {
	noop := func() {}
	if !middleware.IsRealtimeWorldOwned(playerID) {
		return noop, noop, false, 0, "", ""
	}
	handedOff, err := middleware.BeginRealtimeHandoff(playerID)
	if err != nil {
		return noop, noop, false, http.StatusServiceUnavailable, "realtime_checkpoint_unavailable",
			"Không thể tạm dừng điều khiển realtime để chuyển bản đồ, vui lòng thử lại."
	}
	if !handedOff {
		return noop, noop, false, http.StatusConflict, "realtime_world_owns_character",
			"Character is controlled by the realtime world"
	}
	finished := false
	finish = func() {
		if finished {
			return
		}
		middleware.FinishRealtimeHandoff(playerID)
		finished = true
	}
	abort = func() {
		if finished {
			return
		}
		middleware.AbortRealtimeHandoff(playerID)
		finished = true
	}
	return finish, abort, true, 0, "", ""
}

// RealtimeEngageTribulation loads the player's 20x20 Aperture nodes and active tribulation
// enemies for a real-time combat session in the Immortal Aperture.
func RealtimeEngageTribulation(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	finishHandoff, abortHandoff, realtimeTransfer, status, code, message := beginTribulationHandoff(playerID)
	if status != 0 {
		c.JSON(status, gin.H{"code": code, "error": message})
		return
	}
	defer abortHandoff()

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
			en.X, en.Y = int(projectApertureCell(en.X)), int(projectApertureCell(en.Y))
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
			n.X, n.Y = int(projectApertureCell(n.X)), int(projectApertureCell(n.Y))
			nodes = append(nodes, n)
		}
	}

	// 4–5. Save the exact return position and move into the event map atomically.
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu chuyển bản đồ"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var currRegion string
	var currX, currY float64
	if err := tx.QueryRowContext(c.Request.Context(), `
		SELECT current_region, COALESCE(world_x, x::double precision), COALESCE(world_y, y::double precision)
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&currRegion, &currX, &currY); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc vị trí hiện tại của nhân vật"})
		return
	}

	if currRegion != "" && currRegion != "aperture_tribulation" {
		returnRegion, returnX, returnY, atlasX, atlasY, locationErr := resolveApertureReturnLocation(
			c.Request.Context(), tx, playerID, currRegion, currX, currY,
		)
		if locationErr != nil {
			logger.S.Errorf("[APERTURE-RETURN-SAVE] resolve failed: player=%s aperture=%s region=%q err=%v", playerID, ev.ApertureID, currRegion, locationErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác thực vị trí trở về Tiên Khiếu"})
			return
		}
		result, err := tx.ExecContext(c.Request.Context(), `
			UPDATE immortal_apertures
			SET return_region = $1, return_x = $2, return_y = $3
			WHERE id = $4
		`, returnRegion, returnX, returnY, ev.ApertureID)
		if err != nil {
			logger.S.Errorf("[APERTURE-RETURN-SAVE] update failed: player=%s aperture=%s region=%q resolved_region=%q err=%v", playerID, ev.ApertureID, currRegion, returnRegion, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu vị trí trở về Tiên Khiếu"})
			return
		}
		if affected, rowsErr := result.RowsAffected(); rowsErr != nil || affected != 1 {
			logger.S.Errorf("[APERTURE-RETURN-SAVE] update affected %d rows: player=%s aperture=%s region=%q rows_error=%v", affected, playerID, ev.ApertureID, currRegion, rowsErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy Tiên Khiếu để lưu vị trí trở về"})
			return
		}
		if _, err := tx.ExecContext(c.Request.Context(), `
			INSERT INTO aperture_return_anchors(aperture_id, atlas_x, atlas_y, updated_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (aperture_id) DO UPDATE SET atlas_x = EXCLUDED.atlas_x, atlas_y = EXCLUDED.atlas_y, updated_at = NOW()
		`, ev.ApertureID, nullableIntArg(atlasX), nullableIntArg(atlasY)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu điểm neo vị trí trở về"})
			return
		}
	}

	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET current_region = 'aperture_tribulation', world_zone_id = 'aperture_tribulation',
		    world_x = 20, world_y = 20, x = 20, y = 20,
		    world_location_revision = world_location_revision + 1
		WHERE id = $1
	`, playerID)
	if err != nil {
		logger.S.Errorf("Failed to update player %s region to aperture_tribulation: %v", playerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật khu vực người chơi: " + err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất chuyển bản đồ"})
		return
	}
	finishHandoff()

	c.JSON(http.StatusOK, gin.H{
		"event_id":          ev.ID,
		"aperture_id":       ev.ApertureID,
		"tribulation_type":  ev.TribulationType,
		"path":              ev.Path,
		"reward_dao_marks":  ev.RewardDaoMarks,
		"zone_id":           "aperture_tribulation",
		"map_version":       "regions-v1",
		"content_version":   "region-content-v1",
		"player_spawn":      gin.H{"x": 20.0, "y": 20.0},
		"nodes":             nodes,
		"enemies":           enemies,
		"realtime_transfer": realtimeTransfer,
	})
}

// resolveApertureReturnLocation prevents stale realtime region IDs and
// out-of-bounds world coordinates from violating the aperture return-region
// foreign key or creating an exit point that cannot be restored later.
func resolveApertureReturnLocation(
	ctx context.Context,
	q interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	},
	playerID uuid.UUID,
	region string,
	x, y float64,
) (string, float64, float64, sql.NullInt64, sql.NullInt64, error) {
	var active bool
	var width, height, entryX, entryY int
	var safeX, safeY, atlasX, atlasY sql.NullInt64
	err := q.QueryRowContext(ctx, `
		SELECT is_active, width, height, entry_x, entry_y, safe_zone_x, safe_zone_y, atlas_x, atlas_y
		FROM regions WHERE id = $1
	`, region).Scan(&active, &width, &height, &entryX, &entryY, &safeX, &safeY, &atlasX, &atlasY)
	if err == nil && active && width > 0 && height > 0 {
		if x >= 0 && y >= 0 && x < float64(width) && y < float64(height) {
			return region, x, y, atlasX, atlasY, nil
		}
		if safeX.Valid && safeY.Valid && safeX.Int64 >= 0 && safeY.Int64 >= 0 && safeX.Int64 < int64(width) && safeY.Int64 < int64(height) {
			return region, float64(safeX.Int64), float64(safeY.Int64), atlasX, atlasY, nil
		}
		entryX = min(max(entryX, 0), width-1)
		entryY = min(max(entryY, 0), height-1)
		return region, float64(entryX), float64(entryY), atlasX, atlasY, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", 0, 0, sql.NullInt64{}, sql.NullInt64{}, fmt.Errorf("validate current region %q: %w", region, err)
	}
	region, x, y, err = resolveApertureExitLocation(ctx, q, playerID)
	if err != nil {
		return "", 0, 0, sql.NullInt64{}, sql.NullInt64{}, err
	}
	if err := q.QueryRowContext(ctx, `SELECT atlas_x, atlas_y FROM regions WHERE id = $1`, region).Scan(&atlasX, &atlasY); err != nil {
		return "", 0, 0, sql.NullInt64{}, sql.NullInt64{}, fmt.Errorf("read fallback region anchor %q: %w", region, err)
	}
	return region, x, y, atlasX, atlasY, nil
}

// RealtimeTribulationVictoryRequest is kept for API compatibility. Node HP is
// read from the authoritative realtime snapshot/DB, never trusted from the
// client.
type RealtimeTribulationVictoryRequest struct {
	NodeHP map[string]int `json:"node_hp"`
}

type realtimeTribulationVictoryResult struct {
	Message         string
	Rewards         map[string]int64
	UpdatedDaoMarks models.DaoMarksMap
	DidBreakthrough bool
	NewRank         int
	DestroyedNodes  int
	ReturnRegion    string
	ReturnX         float64
	ReturnY         float64
}

func (r realtimeTribulationVictoryResult) payload(realtimeTransfer bool) gin.H {
	return gin.H{
		"message":           r.Message,
		"rewards":           r.Rewards,
		"updated_dao_marks": r.UpdatedDaoMarks,
		"did_breakthrough":  r.DidBreakthrough,
		"new_rank":          r.NewRank,
		"destroyed_nodes":   r.DestroyedNodes,
		"return_region":     r.ReturnRegion,
		"return_x":          r.ReturnX,
		"return_y":          r.ReturnY,
		"realtime_transfer": realtimeTransfer,
	}
}

// RealtimeResolveTribulationVictory is the manual/fallback settlement endpoint.
// It only succeeds when every event enemy is already dead; the authoritative
// auto-settlement path runs from the realtime death handler.
func RealtimeResolveTribulationVictory(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	finishHandoff, abortHandoff, realtimeTransfer, status, code, message := beginTribulationHandoff(playerID)
	if status != 0 {
		c.JSON(status, gin.H{"code": code, "error": message})
		return
	}
	defer abortHandoff()

	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}
	result, status, err := settleRealtimeTribulationVictory(c.Request.Context(), playerID, eventID, c.ClientIP())
	if err != nil {
		if status == 0 {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	finishHandoff()
	c.JSON(http.StatusOK, result.payload(realtimeTransfer))
}

// reconcileTribulationEnemiesFromLiveState marks durable enemies dead when the
// authoritative realtime snapshot already reports HP <= 0. The periodic
// checkpoint is AOI-filtered and may lag behind the final kill, so settlement
// must not depend on that write having happened.
func reconcileTribulationEnemiesFromLiveState(ctx context.Context, tx *sql.Tx, playerID, eventID uuid.UUID) (int, error) {
	_, dead, ok := middleware.RealtimeTribulationEnemies(playerID)
	if !ok || len(dead) == 0 {
		return 0, nil
	}
	ids := make([]string, 0, len(dead))
	for id := range dead {
		if _, err := uuid.Parse(id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE tribulation_enemies
		SET hp = 0, is_dead = true, updated_at = NOW()
		WHERE event_id = $1 AND is_dead = false AND id = ANY($2::uuid[])
	`, eventID, pq.Array(ids))
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// settleRealtimeTribulationVictory commits the victory in one transaction. It
// is shared by the HTTP fallback and automatic settlement on the last realtime
// monster death. status is the HTTP status to surface for a refusal.
func settleRealtimeTribulationVictory(ctx context.Context, playerID, eventID uuid.UUID, ipAddress string) (realtimeTribulationVictoryResult, int, error) {
	var result realtimeTribulationVictoryResult
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi cơ sở dữ liệu")
	}
	defer func() { _ = tx.Rollback() }()

	var ev models.TribulationEvent
	err = tx.QueryRowContext(ctx, `
		SELECT id, aperture_id, player_id, tribulation_type, path, reward_dao_marks, started_at
		FROM tribulation_events
		WHERE id = $1 AND player_id = $2 AND status = 'ACTIVE'
		FOR UPDATE
	`, eventID, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.RewardDaoMarks, &ev.StartedAt)
	if err == sql.ErrNoRows {
		return result, http.StatusNotFound, fmt.Errorf("Đợt tai kiếp này đã kết thúc hoặc không tồn tại")
	} else if err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi truy vấn tai kiếp")
	}

	reconciled, reconcileErr := reconcileTribulationEnemiesFromLiveState(ctx, tx, playerID, eventID)
	if reconcileErr != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi đồng bộ quái tai kiếp từ realtime: %w", reconcileErr)
	}
	if reconciled > 0 {
		logger.S.Infof("Reconciled %d tribulation enemies from live snapshot for %s event %s", reconciled, playerID, ev.ID)
	}

	var alive int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1 AND is_dead = false
	`, ev.ID).Scan(&alive); err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi kiểm tra quái tai kiếp")
	}
	if alive > 0 {
		return result, http.StatusBadRequest, fmt.Errorf("Vẫn còn %d quái kiếp chưa bị tiêu diệt", alive)
	}

	// Node HP was synced from the authoritative zone snapshot during handoff.
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM aperture_resource_nodes WHERE aperture_id = $1 AND is_destroyed = true
	`, ev.ApertureID).Scan(&result.DestroyedNodes); err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi kiểm tra tài nguyên Tiên Khiếu")
	}

	var ap models.ImmortalAperture
	err = tx.QueryRowContext(ctx, `
		SELECT id, aperture_hp, max_aperture_hp, next_tribulation_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, internal_years_passed
		FROM immortal_apertures WHERE id = $1
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.NextTribulationYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.InternalYearsPassed)
	if err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi lấy thông tin Tiên Khiếu")
	}

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
	_ = tx.QueryRowContext(ctx, `SELECT rank, username FROM players WHERE id = $1`, playerID).Scan(&playerRank, &username)

	didBreakthrough := false
	breakthroughMsg := ""
	canBreak, newRank := models.CheckImmortalBreakthrough(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
	if canBreak {
		var pathMasteries models.PathMasteriesMap
		_ = tx.QueryRowContext(ctx, `SELECT COALESCE(path_masteries, '{}'::jsonb) FROM players WHERE id = $1`, playerID).Scan(&pathMasteries)
		reqLevel := gamefx.PathMasteryLevelRequiredForRank(newRank)
		if !gamefx.HasPathMasteryAtLeast(pathMasteries, reqLevel) {
			blockedMsg := fmt.Sprintf(" Chưa đủ điều kiện đột phá lên %s: cần ít nhất 1 lưu phái đạt %s (%d điểm).", models.GetRankName(newRank), gamefx.PathMasteryLevelName(reqLevel), gamefx.PathMasteryPointsForLevel(reqLevel))
			_, _ = tx.ExecContext(ctx, `INSERT INTO messages (sender_id, receiver_id, content, channel) VALUES ($1, $1, $2, 'system')`, playerID, "Đột phá bị chặn: "+blockedMsg)
			breakthroughMsg = blockedMsg
		} else {
			_, _ = tx.ExecContext(ctx, `UPDATE players SET rank = $1 WHERE id = $2`, newRank, playerID)
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

	rewards := distributeDaoMarks(ev.Path, ev.RewardDaoMarks)
	updatedDaoMarks, errDao := addMultipleDaoMarksTx(tx, playerID.String(), rewards)
	if errDao != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi cộng Đạo Ngân: %w", errDao)
	}

	now := time.Now()
	_, _ = tx.ExecContext(ctx, `UPDATE tribulation_events SET status = 'SUCCESS', aperture_hp_after = $1, ended_at = $2 WHERE id = $3`, ap.ApertureHP, now, ev.ID)
	if _, err = tx.ExecContext(ctx, `
		UPDATE immortal_apertures 
		SET state = $1, next_tribulation_year = $2, aperture_hp = $3,
			rank_start_year = $4, earth_calamity_count = $5, heavenly_tribulation_count = $6,
			grand_tribulation_count = $7, myriad_tribulation_count = $8, chaos_calamity_count = $9
		WHERE id = $10
	`, ap.State, ap.NextTribulationYear, ap.ApertureHP,
		ap.RankStartYear, ap.EarthCalamityCount, ap.HeavenlyTribulationCount,
		ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.ID); err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi cập nhật Tiên Khiếu")
	}

	exitRegion, exitX, exitY, exitErr := resolveApertureExitLocation(ctx, tx, playerID)
	if exitErr != nil {
		return result, http.StatusConflict, fmt.Errorf("Không thể tìm vị trí an toàn để trở về: %w", exitErr)
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE players 
		SET current_region = $1, world_zone_id = $1, world_x = $2, world_y = $3, x = $4, y = $5,
		    world_location_revision = world_location_revision + 1
		WHERE id = $6
	`, exitRegion, exitX, exitY, int(exitX), int(exitY), playerID); err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi lưu vị trí trở về")
	}

	if err := tx.Commit(); err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("Lỗi commit giao dịch")
	}

	rewardStrings := []string{}
	for path, amount := range rewards {
		rewardStrings = append(rewardStrings, fmt.Sprintf("%d Đạo Ngân hệ %s", amount, strings.ToUpper(path)))
	}
	rewardText := strings.Join(rewardStrings, ", ")
	msg := fmt.Sprintf("Vượt kiếp thành công trong Tiên Khiếu! Thu hoạch %s. Tiên Khiếu đã ổn định trở lại.", rewardText)
	if result.DestroyedNodes > 0 {
		msg += fmt.Sprintf(" (Có %d điểm tài nguyên bị tổn hại nặng)", result.DestroyedNodes)
	}
	msg += breakthroughMsg

	result.Message = msg
	result.Rewards = rewards
	result.UpdatedDaoMarks = updatedDaoMarks
	result.DidBreakthrough = didBreakthrough
	result.NewRank = playerRank
	result.ReturnRegion = exitRegion
	result.ReturnX = exitX
	result.ReturnY = exitY

	logger.S.Infof("Player %s completed realtime tribulation %s (Breakthrough: %v, IP: %s)", playerID, ev.ID, didBreakthrough, ipAddress)
	return result, 0, nil
}

type apertureExitQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func nullableIntArg(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}

// resolveApertureExitLocation restores an exact valid saved position. If its
// region was removed/disabled or the coordinates no longer fit, it uses the
// nearest active safe region from the atlas snapshot, then an active hub.
func resolveApertureExitLocation(ctx context.Context, q apertureExitQueryer, playerID uuid.UUID) (region string, x float64, y float64, err error) {
	var returnRegion sql.NullString
	var returnX, returnY sql.NullFloat64
	if err = q.QueryRowContext(ctx, `SELECT return_region, return_x, return_y FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&returnRegion, &returnX, &returnY); err != nil {
		return "", 0, 0, fmt.Errorf("read aperture return location: %w", err)
	}
	var anchorX, anchorY sql.NullInt64
	if err = q.QueryRowContext(ctx, `SELECT atlas_x, atlas_y FROM aperture_return_anchors WHERE aperture_id = (SELECT id FROM immortal_apertures WHERE player_id = $1)`, playerID).Scan(&anchorX, &anchorY); err != nil && err != sql.ErrNoRows {
		return "", 0, 0, fmt.Errorf("read aperture return anchor: %w", err)
	}
	if returnRegion.Valid && returnX.Valid && returnY.Valid {
		var active bool
		var width, height int
		if queryErr := q.QueryRowContext(ctx, `SELECT is_active, width, height FROM regions WHERE id = $1`, returnRegion.String).Scan(&active, &width, &height); queryErr == nil && active && width > 0 && height > 0 && returnX.Float64 >= 0 && returnY.Float64 >= 0 && returnX.Float64 < float64(width) && returnY.Float64 < float64(height) {
			return returnRegion.String, returnX.Float64, returnY.Float64, nil
		}
	}
	if !anchorX.Valid || !anchorY.Valid {
		if returnRegion.Valid {
			_ = q.QueryRowContext(ctx, `SELECT atlas_x, atlas_y FROM regions WHERE id = $1`, returnRegion.String).Scan(&anchorX, &anchorY)
		}
	}
	ax, ay := int64(0), int64(0)
	if anchorX.Valid && anchorY.Valid {
		ax, ay = anchorX.Int64, anchorY.Int64
	}
	const safeQuery = `SELECT r.id, COALESCE(r.entry_x, 0), COALESCE(r.entry_y, 0)
		FROM regions r
		WHERE r.is_active = TRUE AND r.zone_type IN ('KHU_VUC', 'SAFE_ZONE') AND left(r.id, 4) <> 'hub_'
		ORDER BY (POWER(r.atlas_x - $1, 2) + POWER(r.atlas_y - $2, 2)), r.id LIMIT 1`
	if err = q.QueryRowContext(ctx, safeQuery, ax, ay).Scan(&region, &x, &y); err == nil {
		return region, x, y, nil
	}
	const hubQuery = `SELECT id, COALESCE(entry_x, 0), COALESCE(entry_y, 0) FROM regions
		WHERE is_active = TRUE AND left(id, 4) = 'hub_'
		ORDER BY (POWER(atlas_x - $1, 2) + POWER(atlas_y - $2, 2)), id LIMIT 1`
	if err = q.QueryRowContext(ctx, hubQuery, ax, ay).Scan(&region, &x, &y); err == nil {
		return region, x, y, nil
	}
	return "", 0, 0, fmt.Errorf("no active safe region or hub is available for aperture exit")
}

func isHubRegion(regionID string) bool {
	return strings.HasPrefix(regionID, "hub_")
}

func isUsableExitRegion(regionID string) bool {
	return regionID != "" && regionID != "aperture_tribulation"
}

// RealtimeEscapeTribulation teleports player out of the tribulation map back to safe exit position.
func RealtimeEscapeTribulation(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	finishHandoff, abortHandoff, realtimeTransfer, status, code, message := beginTribulationHandoff(playerID)
	if status != 0 {
		c.JSON(status, gin.H{"code": code, "error": message})
		return
	}
	defer abortHandoff()

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu rời Tiên Khiếu"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	exitRegion, exitX, exitY, exitErr := resolveApertureExitLocation(c.Request.Context(), tx, playerID)
	if exitErr != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "aperture_exit_unavailable", "error": exitErr.Error()})
		return
	}
	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET current_region = $1, world_zone_id = $1, world_x = $2, world_y = $3, x = $4, y = $5,
		    world_location_revision = world_location_revision + 1
		WHERE id = $6
	`, exitRegion, exitX, exitY, int(exitX), int(exitY), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi rút lui khỏi Tiên Khiếu"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu vị trí trở về"})
		return
	}
	finishHandoff()

	c.JSON(http.StatusOK, gin.H{
		"message":           "Đã an toàn rời khỏi Tiên Khiếu trở về thế giới thực!",
		"return_region":     exitRegion,
		"return_x":          exitX,
		"return_y":          exitY,
		"realtime_transfer": realtimeTransfer,
	})
}
