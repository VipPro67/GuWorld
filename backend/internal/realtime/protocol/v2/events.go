package v2

// Event type ids are stable on the wire. Append new types; never renumber.
const (
	EventCastAccepted          uint8 = 1
	EventCastRejected          uint8 = 2
	EventBasicAttack           uint8 = 3
	EventCastInterrupted       uint8 = 4
	EventDashEnded             uint8 = 5
	EventProjectileSpawn       uint8 = 6
	EventHitConfirmed          uint8 = 7
	EventDamageApplied         uint8 = 8
	EventHealApplied           uint8 = 9
	EventShieldApplied         uint8 = 10
	EventStatusApplied         uint8 = 11
	EventStatusExpired         uint8 = 12
	EventHiddenPassive         uint8 = 13
	EventMoveRejected          uint8 = 14
	EventEntityDied            uint8 = 15
	EventCaptureStarted        uint8 = 16
	EventCaptureInterrupted    uint8 = 17
	EventCaptureSuccess        uint8 = 18
	EventCaptureFailed         uint8 = 19
	EventSoulBackfire          uint8 = 20
	EventCompanionInjured      uint8 = 21
	EventCompanionPermadeath   uint8 = 22
	EventTransformationStarted uint8 = 23
	EventTransformationEnded   uint8 = 24
	EventTransformationBroken  uint8 = 25
)

var eventTypeIDs = map[string]uint8{
	"CAST_ACCEPTED": EventCastAccepted, "CAST_REJECTED": EventCastRejected,
	"BASIC_ATTACK": EventBasicAttack, "CAST_INTERRUPTED": EventCastInterrupted,
	"DASH_ENDED": EventDashEnded, "PROJECTILE_SPAWNED": EventProjectileSpawn,
	"HIT_CONFIRMED": EventHitConfirmed, "DAMAGE_APPLIED": EventDamageApplied,
	"HEAL_APPLIED": EventHealApplied, "SHIELD_APPLIED": EventShieldApplied,
	"STATUS_APPLIED": EventStatusApplied, "STATUS_EXPIRED": EventStatusExpired,
	"HIDDEN_PASSIVE_TRIGGERED": EventHiddenPassive, "MOVE_TARGET_REJECTED": EventMoveRejected,
	"ENTITY_DIED": EventEntityDied, "CAPTURE_STARTED": EventCaptureStarted,
	"CAPTURE_INTERRUPTED": EventCaptureInterrupted, "CAPTURE_SUCCESS": EventCaptureSuccess,
	"CAPTURE_FAILED": EventCaptureFailed, "SOUL_BACKFIRE": EventSoulBackfire,
	"COMPANION_INJURED": EventCompanionInjured, "COMPANION_PERMADEATH": EventCompanionPermadeath,
	"TRANSFORMATION_STARTED": EventTransformationStarted, "TRANSFORMATION_ENDED": EventTransformationEnded,
	"TRANSFORMATION_BROKEN": EventTransformationBroken,
}

var eventTypeNames = func() map[uint8]string {
	out := make(map[uint8]string, len(eventTypeIDs))
	for name, id := range eventTypeIDs {
		out[id] = name
	}
	return out
}()

// EventTypeID maps a domain event name to its wire id (0 = unknown).
func EventTypeID(name string) uint8 { return eventTypeIDs[name] }

// EventTypeName maps a wire id back to the domain name ("" = unknown).
func EventTypeName(id uint8) string { return eventTypeNames[id] }
