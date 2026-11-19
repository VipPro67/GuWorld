package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type apertureAbsorptionConfig struct {
	duration time.Duration
	cost     float64
}

func loadApertureAbsorptionConfig() (apertureAbsorptionConfig, error) {
	durationRaw := strings.TrimSpace(os.Getenv("APERTURE_ABSORPTION_DURATION"))
	costRaw := strings.TrimSpace(os.Getenv("APERTURE_ABSORPTION_COST_IMMORTAL_STONES"))
	if durationRaw == "" || costRaw == "" {
		return apertureAbsorptionConfig{}, errors.New("APERTURE_ABSORPTION_DURATION and APERTURE_ABSORPTION_COST_IMMORTAL_STONES must be configured")
	}
	duration, err := time.ParseDuration(durationRaw)
	if err != nil || duration <= 0 {
		return apertureAbsorptionConfig{}, errors.New("APERTURE_ABSORPTION_DURATION must be a positive Go duration")
	}
	cost, err := strconv.ParseFloat(costRaw, 64)
	if err != nil || math.IsNaN(cost) || math.IsInf(cost, 0) || cost <= 0 {
		return apertureAbsorptionConfig{}, errors.New("APERTURE_ABSORPTION_COST_IMMORTAL_STONES must be a positive number")
	}
	return apertureAbsorptionConfig{duration: duration, cost: cost}, nil
}

type absorptionAperture struct {
	id       string
	playerID string
	rank     int
	public   string
	daoMarks models.DaoMarksMap
	area     int64
	earth    int
	heavenly int
	grand    int
	myriad   int
	chaos    int
}

type absorptionEligibility struct {
	pathNames []string
	level     int
}

func checkApertureAbsorptionEligibility(rank int, targetMarks models.DaoMarksMap, mastery models.PathMasteriesMap) (absorptionEligibility, error) {
	if rank < 6 || rank > 9 {
		return absorptionEligibility{}, fmt.Errorf("rank mục tiêu %d chưa được hỗ trợ hấp thụ", rank)
	}
	dominant := map[string]int64{}
	var maxMarks int64
	for path, marks := range targetMarks {
		path = gamefx.CleanPath(path)
		if path == "" || marks <= 0 {
			continue
		}
		if marks > dominant[path] {
			dominant[path] = marks
		}
		if marks > maxMarks {
			maxMarks = marks
		}
	}
	if maxMarks <= 0 {
		return absorptionEligibility{}, errors.New("Tiên Khiếu mục tiêu không có Đạo Ngân dương để xác định lưu phái chủ đạo")
	}
	requiredLevel := map[int]int{6: 1, 7: 2, 8: 3, 9: 4}[rank]
	paths := make([]string, 0, len(dominant))
	for path, marks := range dominant {
		if marks == maxMarks {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		if gamefx.PathMasteryLevel(mastery[path]) < requiredLevel {
			return absorptionEligibility{pathNames: paths, level: requiredLevel}, fmt.Errorf("cần đạt %s ở lưu phái %s", gamefx.PathMasteryLevelName(requiredLevel), path)
		}
	}
	return absorptionEligibility{pathNames: paths, level: requiredLevel}, nil
}

func StartApertureAbsorption(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	targetID, err := uuid.Parse(c.Param("apertureId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã Tiên Khiếu không hợp lệ"})
		return
	}
	config, err := loadApertureAbsorptionConfig()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Nghi thức hấp thụ chưa được cấu hình", "details": err.Error()})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cơ sở dữ liệu", err))
		return
	}
	defer tx.Rollback()

	var rank int
	var currentRegion string
	var mapInstanceID sql.NullString
	var masteryJSON []byte
	var hasActiveCombat bool
	var immortalStones float64
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT p.rank, COALESCE(p.current_region, ''), p.map_instance_id::text,
		       COALESCE(p.path_masteries, '{}'::jsonb), COALESCE(p.active_combat <> 'null'::jsonb, FALSE), p.immortal_stones
		FROM players p WHERE p.id = $1 FOR UPDATE
	`, playerID).Scan(&rank, &currentRegion, &mapInstanceID, &masteryJSON, &hasActiveCombat, &immortalStones)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nhân vật không tồn tại"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc nhân vật hấp thụ", err))
		return
	}
	if hasActiveCombat {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể bắt đầu hấp thụ khi đang chiến đấu"})
		return
	}
	if immortalStones < config.cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Thạch để bắt đầu nghi thức hấp thụ"})
		return
	}
	if currentRegion != "aperture_interior" || !mapInstanceID.Valid || mapInstanceID.String != targetID.String() {
		c.JSON(http.StatusConflict, gin.H{"error": "Phải ở trong Tiên Khiếu công cộng cần hấp thụ"})
		return
	}
	var mastery models.PathMasteriesMap
	if err := json.Unmarshal(masteryJSON, &mastery); err != nil {
		httperr.Respond(c, httperr.Internal("Dữ liệu cảnh giới lưu phái không hợp lệ", err))
		return
	}

	apertures, err := lockAbsorptionApertures(c.Request.Context(), tx, targetID.String(), playerID.String())
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể khóa hai Tiên Khiếu", err))
		return
	}
	target, targetExists := apertures[targetID.String()]
	var source absorptionAperture
	for _, candidate := range apertures {
		if candidate.playerID == playerID.String() {
			source = candidate
		}
	}
	if !targetExists || target.playerID != "" || target.public != "PUBLIC" {
		c.JSON(http.StatusConflict, gin.H{"error": "Chỉ có thể hấp thụ Tiên Khiếu đang ở trạng thái công cộng"})
		return
	}
	if source.id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa có Tiên Khiếu để hấp thụ vào"})
		return
	}
	if rank < target.rank {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Cần ít nhất Rank %d để hấp thụ Tiên Khiếu này", target.rank)})
		return
	}
	if _, err := checkApertureAbsorptionEligibility(target.rank, target.daoMarks, mastery); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	var anyGuestCombat bool
	if err := tx.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM aperture_guest_roster g JOIN players p ON p.id = g.player_id
			WHERE g.aperture_id = $1 AND (
				COALESCE(p.active_combat <> 'null'::jsonb, FALSE) OR EXISTS (
					SELECT 1 FROM combat_sessions cs
					WHERE (cs.player_id = p.id OR cs.participant_id = p.id) AND cs.status = 'active'
				)
			)
		)
	`, targetID).Scan(&anyGuestCombat); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể kiểm tra giao tranh trong Tiên Khiếu", err))
		return
	}
	if anyGuestCombat {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể bắt đầu nghi thức khi khách trong Tiên Khiếu đang chiến đấu"})
		return
	}
	var finishesAt time.Time
	err = tx.QueryRowContext(c.Request.Context(), `
		INSERT INTO aperture_absorption_reservations
			(target_aperture_id, source_aperture_id, player_id, status, finishes_at, cost_immortal_stones)
		VALUES ($1, $2, $3, 'IN_PROGRESS', NOW() + ($4 * INTERVAL '1 second'), $5)
		RETURNING finishes_at
	`, targetID, source.id, playerID, config.duration.Seconds(), config.cost).Scan(&finishesAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Tiên Khiếu này hoặc đạo hữu đã có một nghi thức hấp thụ đang diễn ra"})
			return
		}
		httperr.Respond(c, httperr.Internal("Không thể giữ chỗ nghi thức hấp thụ", err))
		return
	}
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể bắt đầu nghi thức hấp thụ", err))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "Nghi thức hấp thụ đã bắt đầu", "target_aperture_id": targetID, "finishes_at": finishesAt})
}

func GetApertureAbsorptionStatus(c *gin.Context) {
	if _, ok := middleware.GetPlayerID(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	targetID, err := uuid.Parse(c.Param("apertureId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã Tiên Khiếu không hợp lệ"})
		return
	}
	var playerID, sourceID string
	var finishesAt time.Time
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT player_id::text, source_aperture_id::text, finishes_at
		FROM aperture_absorption_reservations
		WHERE target_aperture_id = $1 AND status = 'IN_PROGRESS'
	`, targetID).Scan(&playerID, &sourceID, &finishesAt)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusOK, gin.H{"in_progress": false})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc trạng thái nghi thức hấp thụ", err))
		return
	}
	viewerID, _ := middleware.GetPlayerID(c)
	c.JSON(http.StatusOK, gin.H{
		"in_progress": true,
		"is_mine":     playerID == viewerID.String(),
		"finishes_at": finishesAt,
	})
}

func lockAbsorptionApertures(ctx context.Context, tx *sql.Tx, targetID, playerID string) (map[string]absorptionAperture, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id::text, COALESCE(player_id::text, ''), aperture_rank, public_state,
		       COALESCE(dao_marks, '{}'::jsonb), area_size, earth_calamity_count,
		       heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count
		FROM immortal_apertures
		WHERE id = $1 OR player_id = $2
		ORDER BY id FOR UPDATE
	`, targetID, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]absorptionAperture, 2)
	for rows.Next() {
		var ap absorptionAperture
		var daoJSON []byte
		if err := rows.Scan(&ap.id, &ap.playerID, &ap.rank, &ap.public, &daoJSON, &ap.area, &ap.earth, &ap.heavenly, &ap.grand, &ap.myriad, &ap.chaos); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(daoJSON, &ap.daoMarks); err != nil {
			return nil, fmt.Errorf("decoding dao marks for aperture %s: %w", ap.id, err)
		}
		result[ap.id] = ap
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func SettleDueApertureAbsorptions(ctx context.Context) error {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT r.target_aperture_id::text
		FROM aperture_absorption_reservations r
		WHERE r.status = 'IN_PROGRESS' AND (
			r.interrupted_at IS NOT NULL OR r.finishes_at <= NOW() OR EXISTS (
				SELECT 1 FROM aperture_guest_roster g
				JOIN players p ON p.id = g.player_id
				WHERE g.aperture_id = r.target_aperture_id AND (
					COALESCE(p.active_combat <> 'null'::jsonb, FALSE) OR EXISTS (
						SELECT 1 FROM combat_sessions cs
						WHERE (cs.player_id = p.id OR cs.participant_id = p.id) AND cs.status = 'active'
					)
				)
			)
		)
		ORDER BY r.finishes_at LIMIT 50
	`)
	if err != nil {
		return fmt.Errorf("finding due aperture absorption reservations: %w", err)
	}
	var targetIDs []string
	for rows.Next() {
		var targetID string
		if err := rows.Scan(&targetID); err != nil {
			rows.Close()
			return fmt.Errorf("reading due absorption target: %w", err)
		}
		targetIDs = append(targetIDs, targetID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterating due absorption reservations: %w", err)
	}
	rows.Close()

	for _, targetID := range targetIDs {
		if err := settleApertureAbsorption(ctx, targetID); err != nil {
			logger.S.Errorf("aperture absorption settlement failed for %s: %v", targetID, err)
		}
	}
	return nil
}

type apertureCell struct{ x, y int }

func settleApertureAbsorption(ctx context.Context, targetID string) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning absorption settlement: %w", err)
	}
	defer tx.Rollback()

	var (
		sourceID, playerID string
		finishesAt         time.Time
		interruptedAt      sql.NullTime
		cost               float64
	)
	err = tx.QueryRowContext(ctx, `
		SELECT source_aperture_id::text, player_id::text, finishes_at, interrupted_at, cost_immortal_stones
		FROM aperture_absorption_reservations
		WHERE target_aperture_id = $1 AND status = 'IN_PROGRESS'
	`, targetID).Scan(&sourceID, &playerID, &finishesAt, &interruptedAt, &cost)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading absorption reservation: %w", err)
	}

	var rank int
	var masteryJSON []byte
	var immortalStones float64
	if err := tx.QueryRowContext(ctx, `
		SELECT rank, COALESCE(path_masteries, '{}'::jsonb), immortal_stones
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&rank, &masteryJSON, &immortalStones); err != nil {
		return fmt.Errorf("locking absorbing player: %w", err)
	}
	locked, err := lockAbsorptionApertures(ctx, tx, targetID, playerID)
	if err != nil {
		return fmt.Errorf("locking source and target apertures: %w", err)
	}
	target, targetOK := locked[targetID]
	source, sourceOK := locked[sourceID]
	if !targetOK || !sourceOK || target.playerID != "" || target.public != "PUBLIC" || source.playerID != playerID {
		return finishApertureAbsorptionWithoutMerge(ctx, tx, targetID, playerID, cost, immortalStones, "FAILED")
	}

	guestLocks, err := tx.QueryContext(ctx, `
		SELECT p.id FROM players p JOIN aperture_guest_roster g ON g.player_id = p.id
		WHERE g.aperture_id = $1 ORDER BY p.id FOR UPDATE OF p
	`, targetID)
	if err != nil {
		return fmt.Errorf("locking public aperture guests for absorption: %w", err)
	}
	for guestLocks.Next() {
		var ignored uuid.UUID
		if err := guestLocks.Scan(&ignored); err != nil {
			guestLocks.Close()
			return fmt.Errorf("reading guest lock: %w", err)
		}
	}
	if err := guestLocks.Err(); err != nil {
		guestLocks.Close()
		return fmt.Errorf("iterating guest locks: %w", err)
	}
	guestLocks.Close()
	// PvP marks the reservation while holding player rows. Lock all guest rows
	// first, then the reservation, to keep that ordering and avoid a lock cycle.
	err = tx.QueryRowContext(ctx, `
		SELECT finishes_at, interrupted_at, cost_immortal_stones
		FROM aperture_absorption_reservations
		WHERE target_aperture_id = $1 AND status = 'IN_PROGRESS'
		FOR UPDATE
	`, targetID).Scan(&finishesAt, &interruptedAt, &cost)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("locking absorption reservation: %w", err)
	}

	var activeCombat bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM aperture_guest_roster g JOIN players p ON p.id = g.player_id
			WHERE g.aperture_id = $1 AND (
				COALESCE(p.active_combat <> 'null'::jsonb, FALSE) OR EXISTS (
					SELECT 1 FROM combat_sessions cs
					WHERE (cs.player_id = p.id OR cs.participant_id = p.id) AND cs.status = 'active'
				)
			)
		)
	`, targetID).Scan(&activeCombat); err != nil {
		return fmt.Errorf("checking public aperture combat interruption: %w", err)
	}

	if activeCombat || interruptedAt.Valid {
		return finishApertureAbsorptionWithoutMerge(ctx, tx, targetID, playerID, cost, immortalStones, "INTERRUPTED")
	}
	if time.Now().Before(finishesAt) {
		return tx.Commit()
	}

	var mastery models.PathMasteriesMap
	if err := json.Unmarshal(masteryJSON, &mastery); err != nil {
		return fmt.Errorf("decoding absorber path mastery: %w", err)
	}
	if rank < target.rank {
		return finishApertureAbsorptionWithoutMerge(ctx, tx, targetID, playerID, cost, immortalStones, "FAILED")
	}
	if _, eligibilityErr := checkApertureAbsorptionEligibility(target.rank, target.daoMarks, mastery); eligibilityErr != nil {
		return finishApertureAbsorptionWithoutMerge(ctx, tx, targetID, playerID, cost, immortalStones, "FAILED")
	}
	if immortalStones < cost {
		_, err = tx.ExecContext(ctx, `UPDATE aperture_absorption_reservations SET status = 'FAILED', updated_at = NOW() WHERE target_aperture_id = $1`, targetID)
		if err != nil {
			return fmt.Errorf("recording insufficient settlement cost: %w", err)
		}
		return tx.Commit()
	}
	if err := chargeAbsorptionCost(ctx, tx, playerID, cost); err != nil {
		return err
	}

	returnRegion, returnX, returnY, err := aperturePortalReturnLocation(ctx, tx, targetID)
	if err != nil {
		return err
	}
	guestIDs, err := loadApertureGuestIDs(ctx, tx, targetID)
	if err != nil {
		return err
	}
	if err := mergeAbsorbedAperture(ctx, tx, source, target); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE players p SET current_region = $2, map_instance_id = NULL,
			x = $3::int, y = $4::int, world_zone_id = $2, world_instance_id = '',
			world_safe_x = $3::double precision, world_safe_y = $4::double precision,
			world_location_revision = world_location_revision + 1,
			version = version + 1, state_revision = state_revision + 1
		FROM aperture_guest_roster g WHERE g.aperture_id = $1 AND g.player_id = p.id
	`, targetID, returnRegion, returnX, returnY); err != nil {
		return fmt.Errorf("ejecting absorbed aperture guests: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE public_aperture_site_configs SET aperture_id = NULL, lifecycle_state = 'CONSUMED', updated_at = NOW()
		WHERE aperture_id = $1
	`, targetID); err != nil {
		return fmt.Errorf("marking absorbed world site consumed: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE aperture_absorption_reservations SET status = 'COMPLETED', updated_at = NOW() WHERE target_aperture_id = $1`, targetID); err != nil {
		return fmt.Errorf("completing absorption reservation: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM immortal_apertures WHERE id = $1 AND player_id IS NULL`, targetID); err != nil {
		return fmt.Errorf("deleting absorbed aperture: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing aperture absorption: %w", err)
	}

	if playerUUID, err := uuid.Parse(playerID); err == nil {
		_ = cache.PublishEventQuick("player:"+playerID, PlayerRealtimeEvent{
			Type: PlayerEventUpdated, Message: "Hấp thụ Tiên Khiếu thành công; diện tích, Đạo Ngân, tai kiếp và mỏ đã hợp nhất.",
			Revision: nextPlayerEventRevision(), ChangedDomains: []PlayerEventDomain{PlayerDomain},
		})
		syncSpatialPlayer(playerUUID, returnRegion, int(returnX), int(returnY), uuid.Nil)
	}
	for _, guestID := range guestIDs {
		if guestUUID, err := uuid.Parse(guestID); err == nil && guestID != playerID {
			_ = cache.PublishEventQuick("player:"+guestID, PlayerRealtimeEvent{
				Type: PlayerEventUpdated, Message: "Tiên Khiếu đã được hấp thụ; đạo hữu được đưa ra Môn Hộ.",
				Revision: nextPlayerEventRevision(), ChangedDomains: []PlayerEventDomain{PlayerDomain},
			})
			syncSpatialPlayer(guestUUID, returnRegion, int(returnX), int(returnY), uuid.Nil)
		}
	}
	publishAperturePortalMarkerChange("portal_despawn", returnRegion)
	return nil
}

func markApertureAbsorptionInterrupted(tx *sql.Tx, playerID uuid.UUID) error {
	_, err := tx.Exec(`
		UPDATE aperture_absorption_reservations r
		SET interrupted_at = COALESCE(r.interrupted_at, NOW()), updated_at = NOW()
		FROM players p
		WHERE p.id = $1 AND p.current_region = 'aperture_interior'
		  AND p.map_instance_id = r.target_aperture_id
		  AND r.status = 'IN_PROGRESS'
	`, playerID)
	if err != nil {
		return fmt.Errorf("marking public aperture absorption interrupted by PvP: %w", err)
	}
	return nil
}

func chargeAbsorptionCost(ctx context.Context, tx *sql.Tx, playerID string, cost float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE players SET immortal_stones = immortal_stones - $1,
			version = version + 1, state_revision = state_revision + 1
		WHERE id = $2 AND immortal_stones >= $1
	`, cost, playerID)
	if err != nil {
		return fmt.Errorf("charging absorption cost: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("không đủ Tiên Thạch để kết toán nghi thức hấp thụ")
	}
	return nil
}

func finishApertureAbsorptionWithoutMerge(ctx context.Context, tx *sql.Tx, targetID, playerID string, cost, available float64, status string) error {
	if available >= cost {
		if err := chargeAbsorptionCost(ctx, tx, playerID, cost); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE aperture_absorption_reservations SET status = $2, updated_at = NOW()
		WHERE target_aperture_id = $1 AND status = 'IN_PROGRESS'
	`, targetID, status); err != nil {
		return fmt.Errorf("recording absorption outcome %s: %w", status, err)
	}
	return tx.Commit()
}

func aperturePortalReturnLocation(ctx context.Context, tx *sql.Tx, apertureID string) (string, float64, float64, error) {
	var region string
	var x, y float64
	err := tx.QueryRowContext(ctx, `SELECT region_id, x, y FROM aperture_portals WHERE aperture_id = $1`, apertureID).Scan(&region, &x, &y)
	if errors.Is(err, sql.ErrNoRows) || region == "" {
		return "Southern_Border", 500, 500, nil
	}
	if err != nil {
		return "", 0, 0, fmt.Errorf("reading aperture portal return location: %w", err)
	}
	return region, x, y, nil
}

func loadApertureGuestIDs(ctx context.Context, tx *sql.Tx, apertureID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT player_id::text FROM aperture_guest_roster WHERE aperture_id = $1 ORDER BY player_id`, apertureID)
	if err != nil {
		return nil, fmt.Errorf("loading aperture guests: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("reading aperture guest id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func mergeAbsorbedAperture(ctx context.Context, tx *sql.Tx, source, target absorptionAperture) error {
	marks := make(models.DaoMarksMap, len(source.daoMarks)+len(target.daoMarks))
	for path, value := range source.daoMarks {
		marks[gamefx.CleanPath(path)] += value
	}
	for path, value := range target.daoMarks {
		marks[gamefx.CleanPath(path)] += value
	}
	marksJSON, err := json.Marshal(marks)
	if err != nil {
		return fmt.Errorf("encoding merged dao marks: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE immortal_apertures SET area_size = area_size + $1, dao_marks = $2::jsonb,
			earth_calamity_count = earth_calamity_count + $3,
			heavenly_tribulation_count = heavenly_tribulation_count + $4,
			grand_tribulation_count = grand_tribulation_count + $5,
			myriad_tribulation_count = myriad_tribulation_count + $6,
			chaos_calamity_count = chaos_calamity_count + $7, updated_at = NOW()
		WHERE id = $8 AND player_id = $9
	`, target.area, string(marksJSON), target.earth, target.heavenly, target.grand, target.myriad, target.chaos, source.id, source.playerID); err != nil {
		return fmt.Errorf("merging aperture area, dao marks, and tribulation counts: %w", err)
	}

	used := make(map[apertureCell]struct{})
	rows, err := tx.QueryContext(ctx, `SELECT x, y FROM aperture_resource_points WHERE aperture_id = $1`, source.id)
	if err != nil {
		return fmt.Errorf("loading destination mine cells: %w", err)
	}
	for rows.Next() {
		var cell apertureCell
		if err := rows.Scan(&cell.x, &cell.y); err != nil {
			rows.Close()
			return fmt.Errorf("reading destination mine cell: %w", err)
		}
		used[cell] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterating destination mine cells: %w", err)
	}
	rows.Close()

	targetRows, err := tx.QueryContext(ctx, `
		SELECT id::text, x, y FROM aperture_resource_points
		WHERE aperture_id = $1 AND scope = 'APERTURE' ORDER BY id FOR UPDATE
	`, target.id)
	if err != nil {
		return fmt.Errorf("locking absorbed mines: %w", err)
	}
	type mineCell struct {
		id   string
		cell apertureCell
	}
	var mines []mineCell
	for targetRows.Next() {
		var mine mineCell
		if err := targetRows.Scan(&mine.id, &mine.cell.x, &mine.cell.y); err != nil {
			targetRows.Close()
			return fmt.Errorf("reading absorbed mine: %w", err)
		}
		mines = append(mines, mine)
	}
	if err := targetRows.Err(); err != nil {
		targetRows.Close()
		return fmt.Errorf("iterating absorbed mines: %w", err)
	}
	targetRows.Close()
	for _, mine := range mines {
		cell, ok := nearestFreeApertureCell(mine.cell, used)
		if !ok {
			continue // The target deletion drops mines that cannot fit in the 20x20 destination.
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE aperture_resource_points SET aperture_id = $1, x = $2, y = $3, updated_at = NOW()
			WHERE id = $4 AND aperture_id = $5
		`, source.id, cell.x, cell.y, mine.id, target.id); err != nil {
			return fmt.Errorf("moving absorbed mine %s: %w", mine.id, err)
		}
		used[cell] = struct{}{}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE non_player_actors
		SET aperture_id = $1, owner_player_id = $2, execution_state = 'STORED',
		    role = 'NONE', formation_slot = 0, world_x = NULL, world_y = NULL,
		    behavior_profile = 'PASSIVE', updated_at = NOW(), version = version + 1
		WHERE aperture_id = $3 AND owner_player_id IS NULL
		  AND actor_kind = 'VARIANT_NPC' AND lifecycle_state = 'ACTIVE'
	`, source.id, source.playerID, target.id); err != nil {
		return fmt.Errorf("transferring absorbed aperture NPCs to new owner: %w", err)
	}
	return nil
}

func nearestFreeApertureCell(origin apertureCell, used map[apertureCell]struct{}) (apertureCell, bool) {
	best := apertureCell{}
	bestDistance := math.MaxInt
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			candidate := apertureCell{x: x, y: y}
			if _, occupied := used[candidate]; occupied {
				continue
			}
			distance := absInt(x-origin.x) + absInt(y-origin.y)
			if distance < bestDistance {
				best, bestDistance = candidate, distance
			}
		}
	}
	return best, bestDistance < math.MaxInt
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
