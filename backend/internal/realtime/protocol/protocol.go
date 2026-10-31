// Package protocol defines the versioned wire contract for the realtime world.
package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

const (
	Version        = 1
	MaxMessageSize = 16 * 1024
	MaxRequestID   = 64
	MaxEntityID    = 80
	MaxWorldCoord  = 1_000_000
)

const (
	TypeJoin     = "world.join"
	TypeWelcome  = "world.welcome"
	TypeInput    = "world.input"
	TypeMoveTo   = "world.move_target"
	TypeCast     = "world.cast"
	TypeAttack   = "world.attack"
	TypeEvents   = "world.events"
	TypeSnapshot = "world.snapshot"
	TypeTransfer = "world.transfer"
	TypeReject   = "world.reject"
	TypeResync   = "world.resync"
)

var ErrInvalidMessage = errors.New("invalid realtime world message")

// Envelope is shared by all client and server messages. Payload is decoded into
// a message-specific type only after the envelope has passed validation.
type Envelope struct {
	Version   int             `json:"v"`
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type JoinRequest struct {
	ZoneID         string `json:"zone_id"`
	MapVersion     string `json:"map_version"`
	ContentVersion string `json:"content_version"`
}

func (join JoinRequest) Validate() error {
	for _, value := range []string{join.ZoneID, join.MapVersion, join.ContentVersion} {
		if value == "" || len(value) > MaxEntityID || strings.TrimSpace(value) != value {
			return fmt.Errorf("%w: join requires bounded zone, map and content versions", ErrInvalidMessage)
		}
	}
	return nil
}

type Welcome struct {
	SessionID       string          `json:"session_id"`
	ZoneID          string          `json:"zone_id"`
	MapVersion      string          `json:"map_version"`
	ContentVersion  string          `json:"content_version"`
	TickRate        uint16          `json:"tick_rate"`
	MovementSpeed   float64         `json:"movement_speed"`
	ServerTick      uint64          `json:"server_tick"`
	EntityID        string          `json:"entity_id"`
	EntityRevision  uint64          `json:"entity_revision"`
	BaselineID      uint64          `json:"baseline_id"`
	MapHash         string          `json:"map_hash,omitempty"`
	Skills          []SkillInfo     `json:"skills,omitempty"`
	LoadoutRevision string          `json:"loadout_revision,omitempty"`
	Snapshot        json.RawMessage `json:"snapshot"`
}

// SkillInfo is the server-owned hotbar projection sent on join. Damage and
// validation stay server-side; the client uses it for mode/name/cooldowns.
type SkillInfo struct {
	Slot           uint8   `json:"slot"`
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Mode           string  `json:"mode"`
	Shape          string  `json:"shape,omitempty"`
	AngleDeg       float64 `json:"angle_deg,omitempty"`
	LengthUnits    float64 `json:"length_units,omitempty"`
	WidthUnits     float64 `json:"width_units,omitempty"`
	InnerRadius    float64 `json:"inner_radius,omitempty"`
	ManaCost       float64 `json:"mana_cost"`
	Range          float64 `json:"range"`
	RadiusUnits    float64 `json:"radius_units,omitempty"`
	CooldownTicks  uint64  `json:"cooldown_ticks"`
	WindupTicks    uint64  `json:"windup_ticks"`
	IsToggle       bool    `json:"is_toggle,omitempty"`
	FormActivation bool    `json:"form_activation,omitempty"`
	FormAbility    bool    `json:"form_ability,omitempty"`
}

type Input struct {
	Seq  uint64  `json:"seq"`
	Move *Vector `json:"move,omitempty"`
}

type Vector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type MoveTarget struct {
	Seq uint64  `json:"seq"`
	X   float64 `json:"x"`
	Y   float64 `json:"y"`
}

type Cast struct {
	Seq      uint64  `json:"seq"`
	Slot     uint8   `json:"slot"`
	Mode     string  `json:"mode,omitempty"`
	TargetID string  `json:"target_id,omitempty"`
	AimX     float64 `json:"aim_x,omitempty"`
	AimY     float64 `json:"aim_y,omitempty"`
	X        float64 `json:"x,omitempty"`
	Y        float64 `json:"y,omitempty"`
}

// Attack is a basic melee swing intent. The client may select a target or aim
// into space; damage, collision and cooldown are server-owned.
type Attack struct {
	Seq      uint64  `json:"seq"`
	TargetID string  `json:"target_id,omitempty"`
	AimX     float64 `json:"aim_x,omitempty"`
	AimY     float64 `json:"aim_y,omitempty"`
}

type Reject struct {
	Code      string `json:"code"`
	Message   string `json:"message,omitempty"`
	Retryable bool   `json:"retryable,omitempty"`
}

type Event struct {
	EventID  uint64          `json:"event_id"`
	Tick     uint64          `json:"tick"`
	Type     string          `json:"type"`
	SourceID string          `json:"source_id,omitempty"`
	TargetID string          `json:"target_id,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

type EntityState struct {
	EntityID    string   `json:"entity_id"`
	Revision    uint64   `json:"entity_revision"`
	Kind        string   `json:"kind,omitempty"`
	PartyID     string   `json:"party_id,omitempty"`
	PvPEnabled  bool     `json:"pvp_enabled,omitempty"`
	TemplateID  string   `json:"template_id,omitempty"`
	Name        string   `json:"name,omitempty"`
	AIState     string   `json:"ai_state,omitempty"`
	AggroTarget string   `json:"aggro_target_id,omitempty"`
	X           float64  `json:"x"`
	Y           float64  `json:"y"`
	HP          float64  `json:"hp"`
	MaxHP       float64  `json:"max_hp,omitempty"`
	Shield      float64  `json:"shield,omitempty"`
	Mana        float64  `json:"mana,omitempty"`
	MaxMana     float64  `json:"max_mana,omitempty"`
	Thought     float64  `json:"thought,omitempty"`
	MaxThought  float64  `json:"max_thought,omitempty"`
	Cooldowns   []uint64 `json:"cooldowns,omitempty"`
	CastSkillID string   `json:"cast_skill_id,omitempty"`
	CastEndTick uint64   `json:"cast_end_tick,omitempty"`
	CastWindup  uint64   `json:"cast_windup_ticks,omitempty"`
	FormActive  bool     `json:"form_active,omitempty"`
	Statuses    []string `json:"statuses,omitempty"`
	Stealthed   bool     `json:"stealthed,omitempty"`
	Flying      bool     `json:"flying,omitempty"`
}

type Snapshot struct {
	Tick        uint64            `json:"tick"`
	AckSeq      uint64            `json:"ack_seq"`
	BaselineID  uint64            `json:"baseline_id"`
	BaseTick    uint64            `json:"base_tick"`
	IsBaseline  bool              `json:"is_baseline"`
	Entities    []EntityState     `json:"entities"`
	Removed     []string          `json:"removed,omitempty"`
	Projectiles []ProjectileState `json:"projectiles,omitempty"`
}

type ProjectileState struct {
	ID       uint64  `json:"id"`
	SourceID string  `json:"source_id"`
	SkillID  string  `json:"skill_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Radius   float64 `json:"radius"`
}

// Transfer is the server-to-client region handoff notice. TargetX/TargetY are
// map-local (MMO) coordinates; LegacyX/LegacyY are the same point projected
// back into the players.x/y world space. The client only persists/display the
// legacy pair and reconnects to ToZoneID.
type Transfer struct {
	FromZoneID string  `json:"from_zone_id,omitempty"`
	ToZoneID   string  `json:"to_zone_id"`
	TransferID string  `json:"transfer_id,omitempty"`
	TargetX    float64 `json:"target_x"`
	TargetY    float64 `json:"target_y"`
	LegacyX    int     `json:"legacy_x"`
	LegacyY    int     `json:"legacy_y"`
	HP         float64 `json:"hp"`
}

type Resync struct {
	LastBaselineID uint64 `json:"last_baseline_id,omitempty"`
	LastEventID    uint64 `json:"last_event_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// EventBatch is the server-to-client world.events payload.
type EventBatch struct {
	FromEventID uint64  `json:"from_event_id"`
	Events      []Event `json:"events"`
}

// DecodeEnvelope only checks the envelope. Use DecodeClientMessage before
// submitting any client command to a world queue.
func DecodeEnvelope(data []byte) (Envelope, error) {
	if len(data) == 0 || len(data) > MaxMessageSize {
		return Envelope{}, fmt.Errorf("%w: message size must be 1..%d bytes", ErrInvalidMessage, MaxMessageSize)
	}
	var envelope Envelope
	if err := decodeStrict(data, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	if envelope.Version != Version {
		return Envelope{}, fmt.Errorf("%w: unsupported protocol version %d", ErrInvalidMessage, envelope.Version)
	}
	if !knownType(envelope.Type) {
		return Envelope{}, fmt.Errorf("%w: unknown message type %q", ErrInvalidMessage, envelope.Type)
	}
	if strings.TrimSpace(envelope.RequestID) != envelope.RequestID || len(envelope.RequestID) > MaxRequestID {
		return Envelope{}, fmt.Errorf("%w: invalid request_id", ErrInvalidMessage)
	}
	if len(envelope.Payload) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Payload), []byte("null")) {
		return Envelope{}, fmt.Errorf("%w: payload is required", ErrInvalidMessage)
	}
	return envelope, nil
}

// DecodeClientMessage enforces direction, strict payload shape and command
// limits. Identity is supplied by the authenticated gateway, never the payload.
func DecodeClientMessage(data []byte) (Envelope, any, error) {
	envelope, err := DecodeEnvelope(data)
	if err != nil {
		return Envelope{}, nil, err
	}
	switch envelope.Type {
	case TypeJoin:
		message, err := DecodePayload[JoinRequest](envelope)
		if err == nil {
			err = message.Validate()
		}
		return envelope, message, err
	case TypeInput:
		message, err := DecodePayload[Input](envelope)
		if err == nil {
			err = message.Validate()
		}
		return envelope, message, err
	case TypeMoveTo:
		message, err := DecodePayload[MoveTarget](envelope)
		if err == nil {
			err = message.Validate()
		}
		return envelope, message, err
	case TypeCast:
		message, err := DecodePayload[Cast](envelope)
		if err == nil {
			err = message.Validate()
		}
		return envelope, message, err
	case TypeAttack:
		message, err := DecodePayload[Attack](envelope)
		if err == nil {
			err = message.Validate()
		}
		return envelope, message, err
	case TypeResync:
		message, err := DecodePayload[Resync](envelope)
		return envelope, message, err
	default:
		return Envelope{}, nil, fmt.Errorf("%w: message type %q is server-only or not yet supported", ErrInvalidMessage, envelope.Type)
	}
}

func (attack Attack) Validate() error {
	if attack.Seq == 0 || len(attack.TargetID) > MaxEntityID || strings.TrimSpace(attack.TargetID) != attack.TargetID || !validCoordinate(attack.AimX) || !validCoordinate(attack.AimY) {
		return fmt.Errorf("%w: invalid basic attack", ErrInvalidMessage)
	}
	if attack.TargetID == "" && math.Hypot(attack.AimX, attack.AimY) <= 0.01 {
		return fmt.Errorf("%w: basic attack requires a target or aim", ErrInvalidMessage)
	}
	return nil
}

// DecodePayload performs strict decoding for a validated envelope payload.
func DecodePayload[T any](envelope Envelope) (T, error) {
	var value T
	if err := decodeStrict(envelope.Payload, &value); err != nil {
		return value, fmt.Errorf("%w: invalid %s payload: %v", ErrInvalidMessage, envelope.Type, err)
	}
	return value, nil
}

func (input Input) Validate() error {
	if input.Seq == 0 {
		return fmt.Errorf("%w: input seq must be positive", ErrInvalidMessage)
	}
	if input.Move == nil {
		return nil
	}
	return validateVector(*input.Move)
}

func (target MoveTarget) Validate() error {
	if target.Seq == 0 || !validCoordinate(target.X) || !validCoordinate(target.Y) {
		return fmt.Errorf("%w: invalid move_target", ErrInvalidMessage)
	}
	return nil
}

func (cast Cast) Validate() error {
	if cast.Seq == 0 || cast.Slot < 1 || cast.Slot > 12 || !validCoordinate(cast.AimX) || !validCoordinate(cast.AimY) || !validCoordinate(cast.X) || !validCoordinate(cast.Y) {
		return fmt.Errorf("%w: invalid cast", ErrInvalidMessage)
	}
	if len(cast.TargetID) > MaxEntityID || strings.TrimSpace(cast.TargetID) != cast.TargetID {
		return fmt.Errorf("%w: invalid target_id", ErrInvalidMessage)
	}
	if cast.Mode != "" && cast.Mode != "TARGET" && cast.Mode != "DIRECTION" && cast.Mode != "GROUND" && cast.Mode != "SELF" {
		return fmt.Errorf("%w: invalid cast mode", ErrInvalidMessage)
	}
	return nil
}

func validateVector(vector Vector) error {
	if !finite(vector.X) || !finite(vector.Y) || math.Hypot(vector.X, vector.Y) > 1.01 {
		return fmt.Errorf("%w: movement vector must be finite and have magnitude <= 1", ErrInvalidMessage)
	}
	return nil
}

func validCoordinate(value float64) bool {
	return finite(value) && math.Abs(value) <= MaxWorldCoord
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func decodeStrict(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func knownType(messageType string) bool {
	switch messageType {
	case TypeJoin, TypeWelcome, TypeInput, TypeMoveTo, TypeCast, TypeAttack, TypeEvents, TypeSnapshot, TypeTransfer, TypeReject, TypeResync:
		return true
	default:
		return false
	}
}
