package combat

// EventType is the stable gameplay event vocabulary shared with the client.
type EventType string

const (
	EventCastAccepted        EventType = "CAST_ACCEPTED"
	EventCastRejected        EventType = "CAST_REJECTED"
	EventBasicAttack         EventType = "BASIC_ATTACK"
	EventCastInterrupted     EventType = "CAST_INTERRUPTED"
	EventDashEnded           EventType = "DASH_ENDED"
	EventProjectileSpawn     EventType = "PROJECTILE_SPAWNED"
	EventHitConfirmed        EventType = "HIT_CONFIRMED"
	EventDamageApplied       EventType = "DAMAGE_APPLIED"
	EventHealApplied         EventType = "HEAL_APPLIED"
	EventShieldApplied       EventType = "SHIELD_APPLIED"
	EventStatusApplied       EventType = "STATUS_APPLIED"
	EventStatusExpired       EventType = "STATUS_EXPIRED"
	EventHiddenPassive       EventType = "HIDDEN_PASSIVE_TRIGGERED"
	EventMoveRejected        EventType = "MOVE_TARGET_REJECTED"
	EventEntityDied          EventType = "ENTITY_DIED"
	EventCaptureStarted      EventType = "CAPTURE_STARTED"
	EventCaptureInterrupted  EventType = "CAPTURE_INTERRUPTED"
	EventCaptureSuccess      EventType = "CAPTURE_SUCCESS"
	EventCaptureFailed       EventType = "CAPTURE_FAILED"
	EventSoulBackfire        EventType = "SOUL_BACKFIRE"
	EventCompanionInjured    EventType = "COMPANION_INJURED"
	EventCompanionPermadeath EventType = "COMPANION_PERMADEATH"
	EventTransformationStarted EventType = "TRANSFORMATION_STARTED"
	EventTransformationEnded   EventType = "TRANSFORMATION_ENDED"
	EventTransformationBroken  EventType = "TRANSFORMATION_BROKEN"
)

// Event is emitted by the zone tick owner. IDs are monotonic per zone and are
// sent to clients through world.events.
type Event struct {
	ID                uint64
	Tick              uint64
	Type              EventType
	SourceID          string
	TargetID          string
	SkillID           string
	SkillName         string
	CastVFX           string
	ProjectileVFX     string
	ImpactVFX         string
	X                 float64
	Y                 float64
	Amount            float64
	ShieldAbsorbed    float64
	HPDamage          float64
	HP                float64
	Shield            float64
	DurationTicks     uint64
	DashDirX          float64
	DashDirY          float64
	DashDurationTicks uint64
	Reason            string
	TemplateID        string
	MonsterRank       int
	PersistentID      string
	TargetKind        string
}
