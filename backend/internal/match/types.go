package match

import (
	"math"
	"sync"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Vec2 is a position or direction in arena tiles.
type Vec2 struct{ X, Y float64 }

func (v Vec2) Add(o Vec2) Vec2      { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) Sub(o Vec2) Vec2      { return Vec2{v.X - o.X, v.Y - o.Y} }
func (v Vec2) Scale(f float64) Vec2 { return Vec2{v.X * f, v.Y * f} }
func (v Vec2) Len() float64         { return math.Hypot(v.X, v.Y) }
func (v Vec2) Dist(o Vec2) float64  { return math.Hypot(v.X-o.X, v.Y-o.Y) }
func (v Vec2) Angle() float64       { return math.Atan2(v.Y, v.X) }

func (v Vec2) Normalized() Vec2 {
	if l := v.Len(); l > 0 {
		return Vec2{v.X / l, v.Y / l}
	}
	return Vec2{}
}

// Obstacle is static arena geometry. Movement slides around it; it does not
// block casts (line-of-sight is out of scope for the demo).
type Obstacle struct {
	Kind string  `json:"kind"` // "circle" | "rect"
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	R    float64 `json:"r"` // circle
	W    float64 `json:"w"` // rect, X/Y is the min corner
	H    float64 `json:"h"`
}

// Status is a timed modifier on a participant. Only the ones that change
// movement matter in the realtime mode.
type Status struct {
	ID        string
	UntilTick int
}

// SkillPayload is the part of a Gu that actually hits: damage, shield, statuses.
// It is read from gu_dictionary.effects, never authored twice.
type SkillPayload struct {
	DamageFlat float64
	DamageMult float64
	ShieldFlat float64
	// ShieldSelf mirrors the operation's target selector: most shields in the
	// dictionary are self-casts that ride along with an offensive cast.
	ShieldSelf bool
	Statuses   []StatusApplication
}

// StatusApplication mirrors the STATUS operation in Block v2.
type StatusApplication struct {
	ID       string
	Duration int // ticks
	Chance   float64
}

// Skill is one castable Gu in a participant's kit.
type Skill struct {
	GuID     string
	Rank     int
	Profile  models.MobaProfile
	Payload  SkillPayload
	ManaCost float64
}

// ActiveCast is a cast that has been accepted and is winding up.
//
// AimPoint is an absolute arena position (not a direction) so the telegraph the
// opponent sees and the hit test that resolves the cast use the same point.
type ActiveCast struct {
	Skill     Skill
	Mode      models.MobaCastMode
	AimPoint  Vec2
	TargetID  uuid.UUID
	StartTick int
	EndTick   int
}

// Participant is one fighter in the arena.
type Participant struct {
	ID      uuid.UUID
	Name    string
	IsDummy bool

	X, Y   float64
	Facing float64

	HP, MaxHP     float64
	Mana, MaxMana float64
	Shield        float64

	Attack, Defense, Speed float64

	Kit []Skill

	Cast              *ActiveCast
	Cooldowns         map[string]int // gu_id -> tick the skill is ready again
	RecoveryUntilTick int
	Statuses          []Status

	// MoveIntent is the most recent accepted movement input.
	MoveIntent Vec2
	// Seq is the last input sequence number acknowledged to this participant.
	Seq int

	// Dummy AI state.
	aiThinkUntil int
	aiSkillIndex int
}

func (p *Participant) alive() bool { return p.HP > 0 }

// SpeedMultiplier folds every active status into a movement scale.
func (p *Participant) SpeedMultiplier(tick int) float64 {
	mult := 1.0
	for _, status := range p.Statuses {
		if status.UntilTick <= tick {
			continue
		}
		switch status.ID {
		case "STUN", "FREEZE", "BIND", "PETRIFY":
			return 0
		case "SLOW1", "SLOW":
			mult = math.Min(mult, 0.5)
		case "SLOW2":
			mult = math.Min(mult, 0.25)
		}
	}
	return mult
}

// Projectile is a travelling skillshot.
type Projectile struct {
	ID     int
	Owner  uuid.UUID
	Skill  Skill
	Pos    Vec2
	Dir    Vec2
	Origin Vec2
	Width  float64
	Speed  float64
	Range  float64
	Hit    map[uuid.UUID]bool
	// DamageScale lets a cast carry its own multiplier (e.g. MULTIPLIER ops).
	DamageScale float64
}

// Event is a one-shot occurrence the client renders as an effect.
type Event struct {
	Tick   int     `json:"tick"`
	Type   string  `json:"type"`
	Source string  `json:"source,omitempty"`
	Target string  `json:"target,omitempty"`
	GuID   string  `json:"gu_id,omitempty"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Damage float64 `json:"damage,omitempty"`
	Amount float64 `json:"amount,omitempty"`
}

// Match is a single live arena instance.
type Match struct {
	mu sync.Mutex

	ID      uuid.UUID
	Tick    int
	Ended   bool
	EndTick int
	Winner  uuid.UUID
	Reason  string

	Obstacles    []Obstacle
	Participants []*Participant
	Projectiles  []*Projectile

	nextProjectileID int
	pending          map[uuid.UUID]Input
	events           []Event
}

// Broadcaster is installed by the server with the WebSocket hub. Nil disables
// broadcasting (tests, headless runs).
var Broadcaster func(channel string, payload []byte, targets map[uuid.UUID]struct{})
