package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// HandleRealtimeAperturePlayerDeath handles player death inside an immortal aperture (aperture_interior).
// Case 1 (Host dies): Automatically concludes anchoring (Thu Khiếu) — closes portal, retracts aperture into host,
// evicts any active guests back outside to the portal entrance, and respawns the host at the portal entrance.
// Case 2 (Guest dies): Removes guest from guest roster, teleports guest outside to the portal entrance,
// while keeping the host's aperture and portal intact.
func HandleRealtimeAperturePlayerDeath(ctx context.Context, death realtimeworld.DeathEvent) error {
	playerID, err := uuid.Parse(string(death.TargetID))
	if err != nil {
		return nil
	}

	handedOff, err := middleware.BeginRealtimeHandoff(playerID)
	if err != nil {
		logger.S.Warnf("realtime aperture death handoff unavailable for %s: %v", playerID, err)
	}
	var realtimeFinished bool
	if handedOff {
		defer func() {
			if !realtimeFinished {
				middleware.AbortRealtimeHandoff(playerID)
			}
		}()
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin aperture death settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check if the dying player is the Host/Owner of an actively anchored aperture
	var apID uuid.UUID
	var placedRegion sql.NullString
	var placedX, placedY sql.NullFloat64
	var returnRegion sql.NullString
	var returnX, returnY sql.NullFloat64

	isHostErr := tx.QueryRowContext(ctx, `
		SELECT id, placed_region, placed_x, placed_y, return_region, return_x, return_y
		FROM immortal_apertures
		WHERE player_id = $1 AND is_placed = true
		FOR UPDATE
	`, playerID).Scan(&apID, &placedRegion, &placedX, &placedY, &returnRegion, &returnX, &returnY)

	if isHostErr == nil {
		// ==========================================
		// CASE 1: HOST DIES INSIDE APERTURE
		// ==========================================
		destRegion := "Southern_Border"
		destX := 500
		destY := 500
		if placedRegion.Valid && placedRegion.String != "" {
			destRegion = placedRegion.String
		} else if returnRegion.Valid && returnRegion.String != "" {
			destRegion = returnRegion.String
		}
		if placedX.Valid {
			destX = int(placedX.Float64)
		} else if returnX.Valid {
			destX = int(returnX.Float64)
		}
		if placedY.Valid {
			destY = int(placedY.Float64)
		} else if returnY.Valid {
			destY = int(returnY.Float64)
		}

		// 1. Deactivate Portal
		_, _ = tx.ExecContext(ctx, `
			UPDATE aperture_portals
			SET is_active = false, updated_at = NOW()
			WHERE aperture_id = $1
		`, apID)

		// 2. Retract aperture back into host body
		_, err = tx.ExecContext(ctx, `
			UPDATE immortal_apertures
			SET is_placed = false, placed_region = NULL, placed_x = NULL, placed_y = NULL, placed_at = NULL,
			    updated_at = NOW()
			WHERE id = $1
		`, apID)
		if err != nil {
			return fmt.Errorf("retract aperture on host death: %w", err)
		}

		// 3. Find and evict all active guests inside the aperture
		type guestInfo struct {
			id       uuid.UUID
			username string
		}
		var evictedGuests []guestInfo
		gRows, gErr := tx.QueryContext(ctx, `
			SELECT g.player_id, p.username
			FROM aperture_guest_roster g
			JOIN players p ON p.id = g.player_id
			WHERE g.aperture_id = $1
		`, apID)
		if gErr == nil {
			for gRows.Next() {
				var g guestInfo
				if scanErr := gRows.Scan(&g.id, &g.username); scanErr == nil {
					evictedGuests = append(evictedGuests, g)
				}
			}
			gRows.Close()
		}

		// Delete guests from roster
		_, _ = tx.ExecContext(ctx, `DELETE FROM aperture_guest_roster WHERE aperture_id = $1`, apID)

		// Move all guests outside to portal entrance
		for _, g := range evictedGuests {
			_, _ = tx.ExecContext(ctx, `
				UPDATE players
				SET current_region = $1, map_instance_id = NULL, x = ($2)::int, y = ($3)::int, world_zone_id = $1, world_instance_id = '',
				    world_x = ($2)::double precision, world_y = ($3)::double precision, world_safe_x = ($2)::double precision, world_safe_y = ($3)::double precision,
				    world_location_revision = world_location_revision + 1, version = version + 1
				WHERE id = $4
			`, destRegion, destX, destY, g.id)
		}

		// 4. Respawn Host at destRegion (destX, destY) with 1 HP
		_, err = tx.ExecContext(ctx, `
			UPDATE players
			SET hp = 1, current_region = $1, map_instance_id = NULL, x = ($2)::int, y = ($3)::int, world_zone_id = $1, world_instance_id = '',
			    world_x = ($2)::double precision, world_y = ($3)::double precision, world_safe_x = ($2)::double precision, world_safe_y = ($3)::double precision,
			    world_location_revision = world_location_revision + 1, version = version + 1
			WHERE id = $4
		`, destRegion, destX, destY, playerID)
		if err != nil {
			return fmt.Errorf("respawn host outside aperture: %w", err)
		}

		GrantPvpImmunity(tx, playerID, 10*time.Minute)

		_, _ = tx.ExecContext(ctx, `
			INSERT INTO player_world_state_v2 (player_id, hp, shield, map_version, content_version, updated_at)
			VALUES ($1, 1, 0, '', '', NOW())
			ON CONFLICT (player_id) DO UPDATE SET hp = 1, shield = 0, updated_at = NOW()
		`, playerID)

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit host aperture death settlement: %w", err)
		}

		if handedOff {
			middleware.FinishRealtimeHandoff(playerID)
			realtimeFinished = true
		}

		// Broadcast portal despawn so nearby players see the gateway close
		_ = cache.PublishEventQuick("world:cell", map[string]any{
			"type":       "portal_despawn",
			"region_id":  destRegion,
			"x":          destX,
			"y":          destY,
			"cell_type":  "AperturePortal",
			"content_id": apID.String(),
		})

		syncSpatialPlayer(playerID, destRegion, destX, destY, uuid.Nil)
		PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated, PlayerDomain)
		_ = notify.Send(db.DB, playerID, "aperture_death", "Tử Đạo Trong Tiên Khiếu",
			fmt.Sprintf("Chủ Khiếu đã tử đạo bên trong Tiên Khiếu! Tiên Khiếu đã tự động thu hồi vào nhục thân, Môn Hộ thế giới khép lại. Đạo hữu đã hoàn nguyên thân thể tại %s (%d, %d).", destRegion, destX, destY))

		// Sync and notify evicted guests
		for _, g := range evictedGuests {
			syncSpatialPlayer(g.id, destRegion, destX, destY, uuid.Nil)
			PublishPlayerRealtimeEvent(g.id, PlayerEventUpdated, PlayerDomain)
			_ = notify.Send(db.DB, g.id, "aperture_evicted", "Tiên Khiếu Đóng Cửa",
				"Chủ Khiếu đã tử đạo, Tiên Khiếu thu hồi vào thân thể. Đạo hữu đã được đưa ra ngoài vị trí Môn Hộ.")
		}

		logger.S.Infof("Host %s died inside aperture %s; automatically unanchored and respawned at %s (%d, %d), evicted %d guests",
			playerID, apID, destRegion, destX, destY, len(evictedGuests))
		return nil
	}

	// ==========================================
	// CASE 2: GUEST DIES INSIDE APERTURE
	// ==========================================
	var guestApertureID uuid.UUID
	isGuestErr := tx.QueryRowContext(ctx, `
		SELECT aperture_id FROM aperture_guest_roster WHERE player_id = $1
	`, playerID).Scan(&guestApertureID)

	if isGuestErr == nil {
		var portalRegion sql.NullString
		var portalX, portalY sql.NullFloat64
		_ = tx.QueryRowContext(ctx, `
			SELECT region_id, x, y FROM aperture_portals WHERE aperture_id = $1
		`, guestApertureID).Scan(&portalRegion, &portalX, &portalY)

		destRegion := "Southern_Border"
		destX := 500
		destY := 500
		if portalRegion.Valid && portalRegion.String != "" {
			destRegion = portalRegion.String
		}
		if portalX.Valid {
			destX = int(portalX.Float64)
		}
		if portalY.Valid {
			destY = int(portalY.Float64)
		}

		// Remove guest from roster
		_, _ = tx.ExecContext(ctx, `DELETE FROM aperture_guest_roster WHERE aperture_id = $1 AND player_id = $2`, guestApertureID, playerID)

		// Teleport guest outside to portal entrance with 1 HP
		_, err = tx.ExecContext(ctx, `
			UPDATE players
			SET hp = 1, current_region = $1, map_instance_id = NULL, x = ($2)::int, y = ($3)::int, world_zone_id = $1, world_instance_id = '',
			    world_x = ($2)::double precision, world_y = ($3)::double precision, world_safe_x = ($2)::double precision, world_safe_y = ($3)::double precision,
			    world_location_revision = world_location_revision + 1, version = version + 1
			WHERE id = $4
		`, destRegion, destX, destY, playerID)
		if err != nil {
			return fmt.Errorf("respawn guest outside aperture: %w", err)
		}

		GrantPvpImmunity(tx, playerID, 10*time.Minute)

		_, _ = tx.ExecContext(ctx, `
			INSERT INTO player_world_state_v2 (player_id, hp, shield, map_version, content_version, updated_at)
			VALUES ($1, 1, 0, '', '', NOW())
			ON CONFLICT (player_id) DO UPDATE SET hp = 1, shield = 0, updated_at = NOW()
		`, playerID)

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit guest aperture death settlement: %w", err)
		}

		if handedOff {
			middleware.FinishRealtimeHandoff(playerID)
			realtimeFinished = true
		}

		syncSpatialPlayer(playerID, destRegion, destX, destY, uuid.Nil)
		PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated, PlayerDomain)
		_ = notify.Send(db.DB, playerID, "aperture_guest_death", "Tử Đạo Trong Tiên Khiếu",
			fmt.Sprintf("Đạo hữu đã bị hạ gục bên trong Tiên Khiếu và được chuyển dịch trở lại bên ngoài Môn Hộ tại %s (%d, %d).", destRegion, destX, destY))

		logger.S.Infof("Guest %s died inside aperture %s; evicted to %s (%d, %d)", playerID, guestApertureID, destRegion, destX, destY)
		return nil
	}

	// Fallback if neither host nor guest in roster: fallback to regular death settlement
	_ = tx.Rollback()
	if handedOff {
		middleware.AbortRealtimeHandoff(playerID)
		realtimeFinished = true
	}
	return HandleRealtimePlayerDeath(ctx, death)
}
