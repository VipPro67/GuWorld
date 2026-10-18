package match

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const tickMs = int(TickInterval / time.Millisecond)

// castState is the telegraph: everything another client needs to draw where the
// cast is aimed and what area it will cover before it lands.
type castState struct {
	GuID         string  `json:"gu_id"`
	StartTick    int     `json:"start_tick"`
	EndTick      int     `json:"ends_tick"`
	CastMode     string  `json:"cast_mode"`
	Shape        string  `json:"shape"`
	Radius       float64 `json:"radius,omitempty"`
	Length       float64 `json:"length,omitempty"`
	Width        float64 `json:"width,omitempty"`
	AngleDeg     float64 `json:"angle_deg,omitempty"`
	MoveDistance float64 `json:"move_distance,omitempty"`
	AimX         float64 `json:"aim_x"`
	AimY         float64 `json:"aim_y"`
	TargetID     string  `json:"target_id,omitempty"`
}

type participantState struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	IsDummy    bool           `json:"is_dummy,omitempty"`
	X          float64        `json:"x"`
	Y          float64        `json:"y"`
	Facing     float64        `json:"facing"`
	HP         float64        `json:"hp"`
	MaxHP      float64        `json:"max_hp"`
	Mana       float64        `json:"mana"`
	Shield     float64        `json:"shield"`
	Speed      float64        `json:"speed"`
	AckSeq     int            `json:"ack_seq"`
	Cast       *castState     `json:"cast,omitempty"`
	CooldownMs map[string]int `json:"cd_ms,omitempty"`
	Statuses   []string       `json:"statuses,omitempty"`
}

type projectileState struct {
	ID    int     `json:"id"`
	GuID  string  `json:"gu_id"`
	Owner string  `json:"owner"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	DX    float64 `json:"dx"`
	DY    float64 `json:"dy"`
}

// frame is the per-tick delta pushed to every participant.
type frame struct {
	Tick         int                `json:"tick"`
	Ended        bool               `json:"ended"`
	Winner       string             `json:"winner,omitempty"`
	Reason       string             `json:"reason,omitempty"`
	Participants []participantState `json:"participants"`
	Projectiles  []projectileState  `json:"projectiles"`
	Events       []Event            `json:"events"`
}

// Snapshot renders the current state. Safe to call from outside the tick loop.
func (m *Match) Snapshot() frame {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.frameLocked()
}

func (m *Match) frameLocked() frame {
	// Slices are built with a non-nil backing array so an empty list marshals as
	// [] rather than null. The client iterates these directly.
	events := make([]Event, len(m.events))
	copy(events, m.events)

	out := frame{
		Tick:         m.Tick,
		Ended:        m.Ended,
		Participants: make([]participantState, 0, len(m.Participants)),
		Projectiles:  make([]projectileState, 0, len(m.Projectiles)),
		Events:       events,
	}
	if m.Winner != uuid.Nil {
		out.Winner = m.Winner.String()
	}
	out.Reason = m.Reason

	for _, p := range m.Participants {
		state := participantState{
			ID: p.ID.String(), Name: p.Name, IsDummy: p.IsDummy,
			X: p.X, Y: p.Y, Facing: p.Facing,
			HP: p.HP, MaxHP: p.MaxHP, Mana: p.Mana, Shield: p.Shield,
			AckSeq: p.Seq,
		}
		if p.Cast != nil {
			profile := p.Cast.Skill.Profile
			cast := &castState{
				GuID:         p.Cast.Skill.GuID,
				StartTick:    p.Cast.StartTick,
				EndTick:      p.Cast.EndTick,
				CastMode:     string(profile.CastMode),
				Shape:        string(profile.Shape),
				Radius:       profile.Radius,
				Length:       profile.Length,
				Width:        profile.Width,
				AngleDeg:     profile.AngleDeg,
				MoveDistance: profile.MoveDistance,
				AimX:         p.Cast.AimPoint.X,
				AimY:         p.Cast.AimPoint.Y,
			}
			if p.Cast.TargetID != uuid.Nil {
				cast.TargetID = p.Cast.TargetID.String()
			}
			state.Cast = cast
		}
		if len(p.Cooldowns) > 0 {
			state.CooldownMs = make(map[string]int, len(p.Cooldowns))
			for guID, ready := range p.Cooldowns {
				if remaining := (ready - m.Tick) * tickMs; remaining > 0 {
					state.CooldownMs[guID] = remaining
				}
			}
		}
		for _, status := range p.Statuses {
			if status.UntilTick > m.Tick {
				state.Statuses = append(state.Statuses, status.ID)
			}
		}
		out.Participants = append(out.Participants, state)
	}

	for _, projectile := range m.Projectiles {
		out.Projectiles = append(out.Projectiles, projectileState{
			ID: projectile.ID, GuID: projectile.Skill.GuID, Owner: projectile.Owner.String(),
			X: projectile.Pos.X, Y: projectile.Pos.Y,
			DX: projectile.Dir.X, DY: projectile.Dir.Y,
		})
	}
	return out
}

// broadcast pushes the current frame to every human participant. Only targets
// listed here receive it, so a player cannot observe someone else's match.
func (m *Match) broadcast() {
	if Broadcaster == nil {
		return
	}
	targets := make(map[uuid.UUID]struct{}, len(m.Participants))
	for _, p := range m.Participants {
		if !p.IsDummy {
			targets[p.ID] = struct{}{}
		}
	}
	if len(targets) == 0 {
		return
	}
	payload, err := json.Marshal(m.frameLocked())
	if err != nil {
		return
	}
	Broadcaster("match:"+m.ID.String(), payload, targets)
}

// Setup is the static, one-off payload the client needs before the first tick:
// arena geometry, the participant roster and each kit's aim metadata.
//
// The movement constants are part of the payload so the client's prediction
// cannot drift from the server's simulation.
type Setup struct {
	MatchID             string        `json:"match_id"`
	Width               float64       `json:"width"`
	Height              float64       `json:"height"`
	ActorRadius         float64       `json:"actor_radius"`
	SpeedToTilesPerTick float64       `json:"speed_to_tiles_per_tick"`
	TickMs              int           `json:"tick_ms"`
	Obstacles           []Obstacle    `json:"obstacles"`
	Players             []SetupPlayer `json:"players"`
}

type SetupPlayer struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	IsDummy bool         `json:"is_dummy"`
	X       float64      `json:"x"`
	Y       float64      `json:"y"`
	HP      float64      `json:"hp"`
	MaxHP   float64      `json:"max_hp"`
	Mana    float64      `json:"mana"`
	Kit     []SetupSkill `json:"kit"`
}

type SetupSkill struct {
	GuID         string              `json:"gu_id"`
	Rank         int                 `json:"rank"`
	CastMode     models.MobaCastMode `json:"cast_mode"`
	Shape        models.MobaShape    `json:"shape"`
	Radius       float64             `json:"radius"`
	Length       float64             `json:"length"`
	Width        float64             `json:"width"`
	AngleDeg     float64             `json:"angle_deg"`
	CastRange    float64             `json:"cast_range"`
	CastTimeMs   int                 `json:"cast_time_ms"`
	CooldownMs   int                 `json:"cooldown_ms"`
	MoveDistance float64             `json:"move_distance"`
	ManaCost     float64             `json:"mana_cost"`
}

// MatchSetup renders the static match description.
func (m *Match) MatchSetup() Setup {
	m.mu.Lock()
	defer m.mu.Unlock()
	setup := Setup{
		MatchID:             m.ID.String(),
		Width:               ArenaWidth,
		Height:              ArenaHeight,
		ActorRadius:         ActorRadius,
		SpeedToTilesPerTick: SpeedToTilesPerTick,
		TickMs:              tickMs,
		Obstacles:           append(make([]Obstacle, 0, len(m.Obstacles)), m.Obstacles...),
		Players:             make([]SetupPlayer, 0, len(m.Participants)),
	}
	for _, p := range m.Participants {
		entry := SetupPlayer{
			ID: p.ID.String(), Name: p.Name, IsDummy: p.IsDummy,
			X: p.X, Y: p.Y, HP: p.HP, MaxHP: p.MaxHP, Mana: p.Mana,
			Kit: make([]SetupSkill, 0, len(p.Kit)),
		}
		for _, skill := range p.Kit {
			entry.Kit = append(entry.Kit, SetupSkill{
				GuID: skill.GuID, Rank: skill.Rank,
				CastMode: skill.Profile.CastMode, Shape: skill.Profile.Shape,
				Radius: skill.Profile.Radius, Length: skill.Profile.Length,
				Width: skill.Profile.Width, AngleDeg: skill.Profile.AngleDeg,
				CastRange: skill.Profile.CastRange, CastTimeMs: skill.Profile.CastTimeMs,
				CooldownMs: skill.Profile.CooldownMs, MoveDistance: skill.Profile.MoveDistance,
				ManaCost: skill.ManaCost,
			})
		}
		setup.Players = append(setup.Players, entry)
	}
	return setup
}
