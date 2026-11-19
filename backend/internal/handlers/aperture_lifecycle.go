package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type apertureExpiryCandidate struct {
	apertureID    string
	playerID      string
	lifespanDecay float64
}

func publicApertureRuinDuration() (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv("PUBLIC_APERTURE_RUIN_DURATION"))
	if raw == "" {
		return 0, errors.New("PUBLIC_APERTURE_RUIN_DURATION must be configured")
	}
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return 0, errors.New("PUBLIC_APERTURE_RUIN_DURATION must be a positive Go duration")
	}
	return duration, nil
}

// TransitionCollapsedPublicApertures atomically turns public apertures with a
// collapsed wall into time-limited ruins and cancels unsettled tribulations.
func TransitionCollapsedPublicApertures(ctx context.Context) error {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id::text FROM immortal_apertures
		WHERE player_id IS NULL AND public_state = 'PUBLIC' AND aperture_hp <= 0
		ORDER BY id LIMIT 50
	`)
	if err != nil {
		return fmt.Errorf("finding collapsed public apertures: %w", err)
	}
	var apertureIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("reading collapsed public aperture id: %w", err)
		}
		apertureIDs = append(apertureIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterating collapsed public apertures: %w", err)
	}
	rows.Close()
	if len(apertureIDs) == 0 {
		return nil
	}
	duration, err := publicApertureRuinDuration()
	if err != nil {
		return err
	}

	for _, apertureID := range apertureIDs {
		if err := transitionPublicApertureToRuin(ctx, apertureID, duration); err != nil {
			logger.S.Errorf("failed to transition public aperture %s to ruin: %v", apertureID, err)
		}
	}
	return nil
}

func transitionPublicApertureToRuin(ctx context.Context, apertureID string, duration time.Duration) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning public aperture ruin transition: %w", err)
	}
	defer tx.Rollback()
	var lockedID string
	err = tx.QueryRowContext(ctx, `
		SELECT id::text FROM immortal_apertures
		WHERE id = $1 AND player_id IS NULL AND public_state = 'PUBLIC' AND aperture_hp <= 0
		FOR UPDATE
	`, apertureID).Scan(&lockedID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("locking collapsed public aperture: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE immortal_apertures
		SET public_state = 'RUIN', public_ruin_expires_at = NOW() + ($2 * INTERVAL '1 second'), updated_at = NOW()
		WHERE id = $1 AND public_state = 'PUBLIC'
	`, lockedID, duration.Seconds()); err != nil {
		return fmt.Errorf("marking public aperture as ruin: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tribulation_events
		SET status = 'CANCELLED', stage = 'CANCELLED', ended_at = COALESCE(ended_at, NOW()), reward_dao_marks = 0
		WHERE aperture_id = $1 AND status = 'ACTIVE'
	`, lockedID); err != nil {
		return fmt.Errorf("cancelling unsettled ruin tribulations: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing public aperture ruin transition: %w", err)
	}
	publishWorldChange("map_cell", "aperture_ruin", "aperture_interior", lockedID, 0, 0, map[string]any{
		"public_state": "RUIN",
	})
	return nil
}

// expireApertureOwner turns a lifespan-expired player's aperture public and
// deletes the player atomically once any persisted combat has settled.
func expireApertureOwner(ctx context.Context, candidate apertureExpiryCandidate) (bool, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("beginning aperture expiry: %w", err)
	}
	defer tx.Rollback()

	var (
		lifespan      float64
		currentRegion string
		playerX       int
		playerY       int
		worldSafeX    int
		worldSafeY    int
		playerRank    int
		returnRegion  sql.NullString
		returnX       sql.NullFloat64
		returnY       sql.NullFloat64
		inCombat      bool
	)
	err = tx.QueryRowContext(ctx, `
		SELECT p.lifespan, COALESCE(p.current_region, ''), COALESCE(p.x, 500), COALESCE(p.y, 500),
		       COALESCE(p.world_safe_x, 500), COALESCE(p.world_safe_y, 500),
		       p.rank, a.return_region, a.return_x, a.return_y,
	       (COALESCE(p.active_combat <> 'null'::jsonb, FALSE) OR EXISTS (
	           SELECT 1 FROM combat_sessions cs
	           WHERE (cs.player_id = p.id OR cs.participant_id = p.id) AND cs.status = 'active'
	       ))
		FROM players p
		JOIN immortal_apertures a ON a.player_id = p.id
		WHERE p.id = $1 AND a.id = $2
		FOR UPDATE OF p, a
	`, candidate.playerID, candidate.apertureID).Scan(
		&lifespan, &currentRegion, &playerX, &playerY, &worldSafeX, &worldSafeY,
		&playerRank, &returnRegion, &returnX, &returnY, &inCombat,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("locking expired player and aperture: %w", err)
	}

	nextLifespan := math.Max(0, lifespan-candidate.lifespanDecay)
	if _, err := tx.ExecContext(ctx, `
		UPDATE players
		SET lifespan = $1, aperture_expiry_pending = ($1 <= 0),
		    version = version + 1, state_revision = state_revision + 1
		WHERE id = $2
	`, nextLifespan, candidate.playerID); err != nil {
		return false, fmt.Errorf("persisting expired lifespan: %w", err)
	}
	if nextLifespan > 0 || inCombat {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("committing pending aperture expiry: %w", err)
		}
		return false, nil
	}

	portalRegion := currentRegion
	portalX := float64(playerX)
	portalY := float64(playerY)
	if currentRegion == "aperture_interior" || strings.HasPrefix(currentRegion, "aperture:") {
		portalRegion = "Southern_Border"
		if returnRegion.Valid && returnRegion.String != "" {
			portalRegion = returnRegion.String
		}
		if returnX.Valid {
			portalX = returnX.Float64
		} else {
			portalX = float64(worldSafeX)
		}
		if returnY.Valid {
			portalY = returnY.Float64
		} else {
			portalY = float64(worldSafeY)
		}
	}
	if portalRegion == "" || portalRegion == "aperture_interior" || strings.HasPrefix(portalRegion, "aperture:") {
		portalRegion = "Southern_Border"
	}

	if err := tx.QueryRowContext(ctx, `
		INSERT INTO aperture_portals (aperture_id, owner_id, region_id, x, y, name, access_policy, concealment_power, is_active, updated_at)
		VALUES ($1, NULL, $2, $3, $4, 'Tiên Khiếu vô chủ', 'HIDDEN', 0, TRUE, NOW())
		ON CONFLICT (aperture_id) DO UPDATE
		SET owner_id = NULL, access_policy = 'HIDDEN', is_active = TRUE, updated_at = NOW()
		RETURNING region_id, x, y
	`, candidate.apertureID, portalRegion, portalX, portalY).Scan(&portalRegion, &portalX, &portalY); err != nil {
		return false, fmt.Errorf("preserving public aperture portal: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE immortal_apertures
		SET aperture_rank = $5, public_state = 'PUBLIC', public_ruin_expires_at = NULL,
		    is_placed = TRUE, placed_region = $2, placed_x = $3, placed_y = $4, updated_at = NOW()
		WHERE id = $1
	`, candidate.apertureID, portalRegion, portalX, portalY, playerRank); err != nil {
		return false, fmt.Errorf("publishing expired aperture: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tribulation_events
		SET status = 'CANCELLED', stage = 'CANCELLED', ended_at = COALESCE(ended_at, NOW()), reward_dao_marks = 0
		WHERE aperture_id = $1 AND player_id = $2 AND status = 'ACTIVE'
	`, candidate.apertureID, candidate.playerID); err != nil {
		return false, fmt.Errorf("cancelling owner-bound tribulation events: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM aperture_ownership WHERE aperture_id = $1`, candidate.apertureID); err != nil {
		return false, fmt.Errorf("removing expired aperture ownership: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE aperture_guest_roster SET granted_by = NULL WHERE granted_by = $1`, candidate.playerID); err != nil {
		return false, fmt.Errorf("detaching guest grants from expired owner: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		WITH ranked_npcs AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY id) - 1 AS slot
			FROM non_player_actors
			WHERE owner_player_id = $1 AND aperture_id = $2
			  AND actor_kind = 'VARIANT_NPC' AND lifecycle_state = 'ACTIVE'
		)
		UPDATE non_player_actors n
		SET owner_player_id = NULL, formation_slot = 0, role = 'NONE',
		    execution_state = 'GUARDING', behavior_profile = 'APERTURE_GUARD',
		    world_x = r.slot % 20, world_y = r.slot / 20,
		    updated_at = NOW(), version = version + 1
		FROM ranked_npcs r WHERE n.id = r.id
	`, candidate.playerID, candidate.apertureID); err != nil {
		return false, fmt.Errorf("detaching aperture NPCs from expired owner: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM players WHERE id = $1`, candidate.playerID); err != nil {
		return false, fmt.Errorf("deleting lifespan-expired player: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("committing public aperture expiry: %w", err)
	}

	if playerID, err := uuid.Parse(candidate.playerID); err == nil {
		ClearPendingPlayerQueues(candidate.playerID)
		middleware.InvalidateSessionCache(playerID)
		_ = cache.PublishEventQuick("player:"+candidate.playerID, PlayerRealtimeEvent{
			Type: SessionEventKicked, Message: "Thọ nguyên đã cạn; nhân vật đã rời thế gian.",
			Revision:       nextPlayerEventRevision(),
			ChangedDomains: []PlayerEventDomain{PlayerDomain},
		})
	}
	publishAperturePortalMarkerChange("portal_spawn", portalRegion)
	logger.S.Infof("lifespan-expired player %s deleted; aperture %s is now public", candidate.playerID, candidate.apertureID)
	return true, nil
}

// CloseExpiredPublicApertures closes expired ruins after their active combats
// settle. Guest positions, roster deletion, and aperture deletion are atomic.
func CloseExpiredPublicApertures(ctx context.Context) error {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id::text FROM immortal_apertures
		WHERE public_state IN ('RUIN', 'CLOSING')
		  AND public_ruin_expires_at IS NOT NULL
		  AND public_ruin_expires_at <= NOW()
		ORDER BY public_ruin_expires_at
		LIMIT 50
	`)
	if err != nil {
		return fmt.Errorf("finding expired public aperture ruins: %w", err)
	}
	var apertureIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("reading expired public aperture id: %w", err)
		}
		apertureIDs = append(apertureIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterating expired public aperture ruins: %w", err)
	}
	rows.Close()

	for _, apertureID := range apertureIDs {
		if err := closeExpiredPublicAperture(ctx, apertureID); err != nil {
			logger.S.Errorf("failed to close expired public aperture %s: %v", apertureID, err)
		}
	}
	return nil
}

func closeExpiredPublicAperture(ctx context.Context, apertureID string) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning public aperture close: %w", err)
	}
	defer tx.Rollback()

	var publicState string
	err = tx.QueryRowContext(ctx, `
		SELECT public_state FROM immortal_apertures
		WHERE id = $1 AND player_id IS NULL
		  AND public_ruin_expires_at IS NOT NULL AND public_ruin_expires_at <= NOW()
		FOR UPDATE
	`, apertureID).Scan(&publicState)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("locking expired public aperture: %w", err)
	}
	if publicState != "RUIN" && publicState != "CLOSING" {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE immortal_apertures SET public_state = 'CLOSING', updated_at = NOW() WHERE id = $1`, apertureID); err != nil {
		return fmt.Errorf("blocking new public aperture entrants: %w", err)
	}
	guestLocks, err := tx.QueryContext(ctx, `
		SELECT p.id FROM players p
		WHERE p.map_instance_id = $1 OR EXISTS (
			SELECT 1 FROM aperture_guest_roster g WHERE g.aperture_id = $1 AND g.player_id = p.id
		)
		ORDER BY p.id
		FOR UPDATE OF p
	`, apertureID)
	if err != nil {
		return fmt.Errorf("locking public aperture guests: %w", err)
	}
	for guestLocks.Next() {
		var ignoredID uuid.UUID
		if err := guestLocks.Scan(&ignoredID); err != nil {
			guestLocks.Close()
			return fmt.Errorf("reading locked public aperture guest: %w", err)
		}
	}
	if err := guestLocks.Err(); err != nil {
		guestLocks.Close()
		return fmt.Errorf("iterating locked public aperture guests: %w", err)
	}
	guestLocks.Close()

	var activeCombat bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM players p
			WHERE (p.map_instance_id = $1 OR EXISTS (
				SELECT 1 FROM aperture_guest_roster g WHERE g.aperture_id = $1 AND g.player_id = p.id
			)) AND (
				p.active_combat IS NOT NULL AND p.active_combat <> 'null'::jsonb
				OR EXISTS (
					SELECT 1 FROM combat_sessions cs
					WHERE (cs.player_id = p.id OR cs.participant_id = p.id) AND cs.status = 'active'
				)
			)
		)
	`, apertureID).Scan(&activeCombat); err != nil {
		return fmt.Errorf("checking public aperture guest combat: %w", err)
	}
	if activeCombat {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing public aperture closing state: %w", err)
		}
		publishWorldChange("map_cell", "aperture_closing", "aperture_interior", apertureID, 0, 0, map[string]any{
			"public_state": "CLOSING",
		})
		return nil
	}

	var returnRegion string
	var returnX, returnY float64
	err = tx.QueryRowContext(ctx, `
		SELECT region_id, x, y FROM aperture_portals WHERE aperture_id = $1
	`, apertureID).Scan(&returnRegion, &returnX, &returnY)
	if errors.Is(err, sql.ErrNoRows) || returnRegion == "" {
		returnRegion, returnX, returnY = "Southern_Border", 500, 500
	} else if err != nil {
		return fmt.Errorf("loading public aperture return location: %w", err)
	}

	guestRows, err := tx.QueryContext(ctx, `
		SELECT player_id::text FROM aperture_guest_roster WHERE aperture_id = $1
		UNION
		SELECT id::text FROM players WHERE map_instance_id = $1
	`, apertureID)
	if err != nil {
		return fmt.Errorf("loading public aperture guests: %w", err)
	}
	var guestIDs []string
	for guestRows.Next() {
		var guestID string
		if err := guestRows.Scan(&guestID); err != nil {
			guestRows.Close()
			return fmt.Errorf("reading public aperture guest: %w", err)
		}
		guestIDs = append(guestIDs, guestID)
	}
	if err := guestRows.Err(); err != nil {
		guestRows.Close()
		return fmt.Errorf("iterating public aperture guests: %w", err)
	}
	guestRows.Close()

	if _, err := tx.ExecContext(ctx, `
		UPDATE players p
		SET current_region = $2, map_instance_id = NULL, x = $3::int, y = $4::int,
		    world_zone_id = $2, world_instance_id = '',
		    world_safe_x = $3::double precision, world_safe_y = $4::double precision,
		    world_location_revision = world_location_revision + 1, version = version + 1
		WHERE p.map_instance_id = $1 OR EXISTS (
			SELECT 1 FROM aperture_guest_roster g WHERE g.aperture_id = $1 AND g.player_id = p.id
		)
	`, apertureID, returnRegion, returnX, returnY); err != nil {
		return fmt.Errorf("ejecting public aperture guests: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE public_aperture_site_configs
		SET aperture_id = NULL, lifecycle_state = 'CONSUMED', updated_at = NOW()
		WHERE aperture_id = $1
	`, apertureID); err != nil {
		return fmt.Errorf("marking public aperture site as consumed: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM non_player_actors
		WHERE aperture_id = $1 AND owner_player_id IS NULL AND actor_kind = 'VARIANT_NPC'
	`, apertureID); err != nil {
		return fmt.Errorf("removing expired public aperture NPCs: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM immortal_apertures WHERE id = $1 AND player_id IS NULL AND public_state = 'CLOSING'`, apertureID); err != nil {
		return fmt.Errorf("deleting expired public aperture: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing public aperture close: %w", err)
	}

	for _, guestID := range guestIDs {
		if guestUUID, err := uuid.Parse(guestID); err == nil {
			_ = cache.PublishEventQuick("player:"+guestID, PlayerRealtimeEvent{
				Type: PlayerEventUpdated, Message: "Tàn tích Tiên Khiếu đã đóng; đạo hữu được đưa về Môn Hộ.",
				Revision: nextPlayerEventRevision(), ChangedDomains: []PlayerEventDomain{PlayerDomain},
			})
			syncSpatialPlayer(guestUUID, returnRegion, int(returnX), int(returnY), uuid.Nil)
		}
	}
	publishAperturePortalMarkerChange("portal_despawn", returnRegion)
	return nil
}
