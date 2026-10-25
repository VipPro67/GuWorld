package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Movement is one concept with many entrances: walking on the world grid, map
// portals, teleport arrays, Gu-driven teleports, spatial-realm gates and party
// joins. This file owns the shared chain:
//
//	ownership -> bounds -> alive -> combat-free -> party leader -> region policy
//	-> destination -> regional wall -> caller's persistence -> party drag
//	-> post-commit publication
//
// Persistence deliberately stays with the caller's transaction: each entrance
// writes a different set of columns (instance id, stones, active combat,
// party id). What must not differ is the guard order, the destination write,
// the wall debuff and the realtime publication.

type movementKind int

const (
	movementWalk movementKind = iota
	movementPortal
	movementTeleport
	movementGuTeleport
	movementPartyJoin
)

// movementPolicy is the part that actually differs between entrances. Keeping
// it in one table is what stops a new entrance from quietly skipping a check.
type movementPolicy struct {
	requirePartyLeader bool
	requireRegionEntry bool
	// regionEntryStatus and actionStatus preserve each entrance's historical
	// HTTP status, so routing a handler through the chain does not change what
	// clients already handle.
	regionEntryStatus int
	sourceAction      string
	targetAction      string
	actionStatus      int
}

var movementPolicies = map[movementKind]movementPolicy{
	movementWalk:       {requirePartyLeader: true},
	movementPortal:     {requireRegionEntry: true, regionEntryStatus: http.StatusBadRequest},
	movementTeleport:   {requireRegionEntry: true, regionEntryStatus: http.StatusBadRequest, targetAction: "teleport", actionStatus: http.StatusBadRequest},
	movementGuTeleport: {requireRegionEntry: true, regionEntryStatus: http.StatusForbidden},
	movementPartyJoin:  {},
}

// movementSpec describes one intent. Region fields are optional: an empty
// targetRegion means "stay in the current region" and hasTarget=false means
// the caller already knows the destination is server-chosen.
type movementSpec struct {
	kind         movementKind
	x, y         int
	hasTarget    bool
	targetRegion string
}

// movementError is the only failure shape the chain returns for guard
// violations, so every entrance can answer with the same JSON.
type movementError struct {
	status  int
	code    string
	message string
}

func (e *movementError) Error() string { return e.message }

func movementDenied(status int, message string) *movementError {
	return &movementError{status: status, message: message}
}

// validateMovementTarget is the database-free coordinate check. Callers use it
// before opening a transaction so malformed input never reaches the DB; the
// full chain also runs it, so there is still one rule.
func validateMovementTarget(spec movementSpec) *movementError {
	if spec.hasTarget && !inWorld(spec.x, spec.y) {
		return movementDenied(http.StatusBadRequest, "Tọa độ vượt ra ngoài ranh giới bản đồ.")
	}
	return nil
}

// validateMovementState is the database-free half of the chain. Kept pure so
// the precedence and messages are testable without a transaction.
func validateMovementState(player *models.Player, spec movementSpec, ownedByRealtime bool) *movementError {
	if ownedByRealtime {
		return &movementError{
			status:  http.StatusConflict,
			code:    "realtime_world_owns_character",
			message: "Character is controlled by the realtime world",
		}
	}
	if err := validateMovementTarget(spec); err != nil {
		return err
	}
	if gamefx.IsDeadState(player) {
		return movementDenied(http.StatusForbidden, "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể di chuyển!")
	}
	if player.HP <= 0 {
		return movementDenied(http.StatusBadRequest, "Đạo hữu đang trong trạng thái trọng thương, không thể tiếp tục di chuyển!")
	}
	if player.ActiveCombat != nil {
		return movementDenied(http.StatusBadRequest, "Bạn đang trong trạng thái chiến đấu! Hãy kết thúc trận chiến trước.")
	}
	return nil
}

// checkMovement runs the full guard for a player-initiated movement.
func checkMovement(tx *sql.Tx, player *models.Player, spec movementSpec) error {
	if err := validateMovementState(player, spec, middleware.IsRealtimeWorldOwned(player.ID)); err != nil {
		return err
	}
	policy, known := movementPolicies[spec.kind]
	if !known {
		// A new entrance without a policy entry would silently skip every
		// kind-specific check; fail loudly instead.
		return &movementError{status: http.StatusInternalServerError, message: "unknown movement kind"}
	}

	regionChanged := spec.targetRegion != "" && spec.targetRegion != player.CurrentRegion
	if policy.requireRegionEntry && regionChanged {
		if err := checkRegionEntry(spec.targetRegion, player.Rank, player.GuInventory); err != nil {
			status := policy.regionEntryStatus
			if status == 0 {
				status = http.StatusForbidden
			}
			return movementDenied(status, err.Error())
		}
	}
	if policy.sourceAction != "" {
		if err := checkRegionAction(player.CurrentRegion, policy.sourceAction); err != nil {
			return movementDenied(actionStatus(policy), err.Error())
		}
	}
	if policy.targetAction != "" && spec.targetRegion != "" {
		if err := checkRegionAction(spec.targetRegion, policy.targetAction); err != nil {
			return movementDenied(actionStatus(policy), err.Error())
		}
	}
	if policy.requirePartyLeader {
		if err := checkPartyLeaderOrSolo(tx, player); err != nil {
			return movementDenied(http.StatusBadRequest, err.Error())
		}
	}
	return nil
}

func actionStatus(policy movementPolicy) int {
	if policy.actionStatus != 0 {
		return policy.actionStatus
	}
	return http.StatusForbidden
}

// rejectRealtimeOwned stops a legacy movement mutation before it opens a
// transaction, so an MMO-owned character is not even charged mana by the
// legacy path. The full guard still repeats the check at the seam.
func rejectRealtimeOwned(c *gin.Context, playerID uuid.UUID) bool {
	if !middleware.IsRealtimeWorldOwned(playerID) {
		return false
	}
	c.JSON(http.StatusConflict, gin.H{
		"code":  "realtime_world_owns_character",
		"error": "Character is controlled by the realtime world",
	})
	return true
}

// movementErrorPayload renders a guard violation without needing a gin context,
// so both HTTP adapters and the WebSocket command path share one response.
func movementErrorPayload(err error) (int, gin.H, bool) {
	me, ok := err.(*movementError)
	if !ok {
		return 0, nil, false
	}
	payload := gin.H{"error": me.message}
	if me.code != "" {
		payload["code"] = me.code
	}
	return me.status, payload, true
}

// respondMovementError writes the shared guard response. It reports whether the
// error was a movement violation.
func respondMovementError(c *gin.Context, err error) bool {
	status, payload, ok := movementErrorPayload(err)
	if !ok {
		return false
	}
	c.JSON(status, payload)
	return true
}

// applyDestination moves the in-memory player and applies the regional wall
// debuff when the region changes. Persistence stays with the caller.
func applyDestination(tx *sql.Tx, player *models.Player, region string, x, y int) string {
	oldRegion := player.CurrentRegion
	if region != "" {
		player.CurrentRegion = region
	}
	player.X = x
	player.Y = y
	if player.CurrentRegion != oldRegion {
		return CheckAndApplyRegionalWallDebuffBetween(player, oldRegion, player.CurrentRegion, tx)
	}
	return ""
}

// placePlayerAt sets a server-driven destination (respawn, realm return,
// onboarding). It never applies a wall debuff.
func placePlayerAt(player *models.Player, region string, x, y int) {
	if region != "" {
		player.CurrentRegion = region
	}
	player.X = x
	player.Y = y
}

type partyMovementPublish int

const (
	publishPartyNone partyMovementPublish = iota
	publishPartyPositionDelta
	publishPartyRealtimeEvent
)

// broadcastPlayerWorldPos is the one construction of a position event.
func broadcastPlayerWorldPos(player *models.Player) {
	broadcastWorldPos(WorldPosEvent{
		ID:        player.ID.String(),
		Username:  player.Username,
		AvatarURL: player.AvatarURL,
		X:         player.X,
		Y:         player.Y,
		Region:    player.CurrentRegion,
		TS:        time.Now().UnixMilli(),
	})
}

// publishPartyMovement broadcasts each dragged member and the domain delta the
// entrance historically used.
func publishPartyMovement(members []SyncedPartyMember, mode partyMovementPublish) {
	if mode == publishPartyNone {
		return
	}
	for _, member := range members {
		broadcastWorldPos(WorldPosEvent{
			ID:        member.ID.String(),
			Username:  member.Username,
			AvatarURL: member.AvatarURL,
			X:         member.X,
			Y:         member.Y,
			Region:    member.Region,
			TS:        time.Now().UnixMilli(),
		})
		if mode == publishPartyPositionDelta {
			PublishPlayerPositionDelta(member.ID, member.X, member.Y)
		} else {
			PublishPlayerRealtimeEvent(member.ID, PlayerEventUpdated)
		}
	}
}
