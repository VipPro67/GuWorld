package handlers

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
)

// ErrRealtimeRegionTransferDenied marks a policy/state refusal (rank gate, dead
// character, combat, unknown region). Gateway maps it to a 403 world.reject.
var ErrRealtimeRegionTransferDenied = errors.New("realtime region transfer denied")

// RealtimeRegionTransfer is one authoritative border crossing computed by the
// world manager. LocalX/LocalY are map units; the origin/scale pair is the
// destination map transform and is the only supported way to write players.x/y.
type RealtimeRegionTransfer struct {
	PlayerID         uuid.UUID
	TargetRegion     string
	TargetZoneID     string
	TargetInstanceID string
	LocationRevision int64
	LocalX           float64
	LocalY           float64
	LegacyOriginX    float64
	LegacyOriginY    float64
	LegacyScale      float64
	HP               float64
}

// PlayerTransferResult carries the persisted row and the optional wall debuff
// message so the gateway response stays identical to the legacy portal path.
type PlayerTransferResult struct {
	Player  models.Player
	Debuff  string
	Members []SyncedPartyMember
}

// PublishRegionTransfer fans out the persisted handoff after the gateway has
// released the source lease. Publishing before release makes an SSE-driven
// client reconnect race admission and fail with character_already_owned.
func PublishRegionTransfer(result PlayerTransferResult) {
	broadcastPlayerWorldPos(&result.Player)
	PublishPlayerRealtimeEvent(result.Player.ID, PlayerEventUpdated)
	publishPartyMovement(result.Members, publishPartyRealtimeEvent)
}

// ValidateRealtimeRegionTransfer performs the read-only half of the handoff:
// alive, not in combat, rank/Gu gate for the destination region. The gateway
// runs it before flushing the source-zone checkpoint so a denied border does
// not write anything.
func ValidateRealtimeRegionTransfer(ctx context.Context, playerID uuid.UUID, targetRegion string) error {
	if db.DB == nil {
		return errors.New("database not initialized")
	}
	target := ZoneByID(targetRegion)
	if target == nil {
		return fmt.Errorf("%w: unknown region %q", ErrRealtimeRegionTransferDenied, targetRegion)
	}
	loaded, err := playerstate.Load(db.DB, playerID, playerstate.Options{Gu: playerstate.GuEquipped})
	if err != nil {
		return err
	}
	if gamefx.IsDeadState(&loaded) || loaded.HP <= 0 {
		return fmt.Errorf("%w: character is not alive", ErrRealtimeRegionTransferDenied)
	}
	if err := checkRegionEntry(target.ID, loaded.Rank, loaded.GuInventory); err != nil {
		return fmt.Errorf("%w: %v", ErrRealtimeRegionTransferDenied, err)
	}
	return nil
}

// ApplyRealtimeRegionTransfer persists a server-owned edge handoff. The world
// coordinates written to players.x/y are always the legacy projection of the
// destination map, never the raw MMO cell coordinates.
func ApplyRealtimeRegionTransfer(ctx context.Context, req RealtimeRegionTransfer) (PlayerTransferResult, error) {
	if db.DB == nil {
		return PlayerTransferResult{}, errors.New("database not initialized")
	}
	if req.PlayerID == uuid.Nil || req.TargetRegion == "" {
		return PlayerTransferResult{}, fmt.Errorf("%w: missing player or region", ErrRealtimeRegionTransferDenied)
	}
	target := ZoneByID(req.TargetRegion)
	if target == nil {
		return PlayerTransferResult{}, fmt.Errorf("%w: unknown region %q", ErrRealtimeRegionTransferDenied, req.TargetRegion)
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return PlayerTransferResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	loaded, err := playerstate.Load(tx, req.PlayerID, playerstate.Options{ForUpdate: true, Gu: playerstate.GuEquipped})
	if err != nil {
		return PlayerTransferResult{}, err
	}
	if gamefx.IsDeadState(&loaded) || loaded.HP <= 0 {
		return PlayerTransferResult{}, fmt.Errorf("%w: character is not alive", ErrRealtimeRegionTransferDenied)
	}
	if err := checkRegionEntry(target.ID, loaded.Rank, loaded.GuInventory); err != nil {
		return PlayerTransferResult{}, fmt.Errorf("%w: %v", ErrRealtimeRegionTransferDenied, err)
	}
	var locationRevision int64
	if err := tx.QueryRowContext(ctx, `SELECT world_location_revision FROM players WHERE id=$1`, req.PlayerID).Scan(&locationRevision); err != nil {
		return PlayerTransferResult{}, err
	}
	if locationRevision != req.LocationRevision {
		return PlayerTransferResult{}, fmt.Errorf("realtime location changed before transfer: expected revision %d, found %d", req.LocationRevision, locationRevision)
	}

	scale := req.LegacyScale
	if scale <= 0 {
		scale = 1
	}
	legacyX := int(math.Round(req.LegacyOriginX + req.LocalX*scale))
	legacyY := int(math.Round(req.LegacyOriginY + req.LocalY*scale))
	if target.MaxX > target.MinX {
		legacyX = clampInt(legacyX, target.MinX+1, target.MaxX-1)
	}
	if target.MaxY > target.MinY {
		legacyY = clampInt(legacyY, target.MinY+1, target.MaxY-1)
	}

	debuffMessage := applyDestination(tx, &loaded, target.ID, legacyX, legacyY)
	hp := req.HP
	if hp <= 0 {
		hp = 1
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE players
		SET current_region = $1, x = $2, y = $3, hp = $4, statuses = $5,
		    map_instance_id = NULL, world_zone_id = $6, world_instance_id = $7,
		    world_x = $8, world_y = $9, world_safe_x = $8, world_safe_y = $9,
		    world_location_revision = world_location_revision + 1
		WHERE id = $10 AND world_location_revision = $11
	`, loaded.CurrentRegion, loaded.X, loaded.Y, hp, loaded.Statuses, req.TargetZoneID,
		req.TargetInstanceID, req.LocalX, req.LocalY, loaded.ID, req.LocationRevision)
	if err != nil {
		return PlayerTransferResult{}, err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return PlayerTransferResult{}, err
	} else if affected != 1 {
		return PlayerTransferResult{}, fmt.Errorf("realtime location changed before transfer commit")
	}
	syncedMembers := SyncPartyMembersLocation(tx, &loaded)
	if err := tx.Commit(); err != nil {
		return PlayerTransferResult{}, err
	}

	return PlayerTransferResult{Player: loaded, Debuff: debuffMessage, Members: syncedMembers}, nil
}

// LegacyPortalDestination projects an authored map-local portal target into the
// legacy world coordinates used by players.x/y. Portal metadata is MMO-local;
// writing it raw would make admission remap it outside the destination map and
// silently safe-spawn the player.
func LegacyPortalDestination(target *models.Region, localX, localY int) (int, int) {
	x := localX
	y := localY
	if target != nil {
		if target.Width > 0 && x > target.Width-2 {
			x = target.Width - 2
		}
		if target.Height > 0 && y > target.Height-2 {
			y = target.Height - 2
		}
		if x < 1 {
			x = 1
		}
		if y < 1 {
			y = 1
		}
		if target.MaxX > target.MinX {
			x = target.MinX + x
		}
		if target.MaxY > target.MinY {
			y = target.MinY + y
		}
	}
	return x, y
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
