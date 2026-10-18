package match

import (
	"math"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// DummyKitRefs is the practice opponent's rotation: one of each pattern the
// player must learn to dodge (point target, skillshot, delayed ground AoE).
var DummyKitRefs = []GuRef{
	{"GU_BLADE_AURA_1", 1},
	{"GU_WATER_ARROW", 1},
	{"GU_STAR_FALL_1", 1},
}

// NewDummy builds the practice opponent.
func NewDummy(id uuid.UUID, x, y float64) *Participant {
	return &Participant{
		ID: id, Name: "Hình Nhân Gỗ", IsDummy: true,
		X: x, Y: y,
		HP: DummyHP, MaxHP: DummyHP,
		Mana: MaxMana, MaxMana: MaxMana,
		Attack: DummyAttack, Defense: DummyDefense, Speed: DummySpeed,
		Kit: BuildKit(DummyKitRefs),
	}
}

// dummyThink runs the practice opponent: step out of anything telegraphed at it
// that it can actually escape, otherwise keep up the skill rotation.
func (m *Match) dummyThink(p *Participant) {
	if !p.alive() {
		return
	}
	// Casting roots the dummy, exactly like the player.
	if p.Cast != nil {
		return
	}
	if m.dummyDodge(p) {
		return
	}
	p.MoveIntent = Vec2{}

	if len(p.Kit) == 0 || m.Tick < p.aiThinkUntil || m.Tick < p.RecoveryUntilTick {
		return
	}
	target := m.humanOpponent(p)
	if target == nil {
		return
	}

	skill := p.Kit[p.aiSkillIndex%len(p.Kit)]
	p.aiSkillIndex++
	p.aiThinkUntil = m.Tick + DummyThinkTicks

	origin := Vec2{p.X, p.Y}
	aim := Vec2{target.X, target.Y}.Add(Vec2{
		X: (rand.Float64()*2 - 1) * DummyAimErrorTiles,
		Y: (rand.Float64()*2 - 1) * DummyAimErrorTiles,
	})

	request := CastInput{GuID: skill.GuID, Rank: skill.Rank, Mode: skill.Profile.CastMode}
	switch skill.Profile.CastMode {
	case models.MobaCastTarget:
		if origin.Dist(Vec2{target.X, target.Y}) > skill.Profile.CastRange {
			return
		}
		request.TargetID = target.ID.String()
	case models.MobaCastDirection:
		dir := aim.Sub(origin).Normalized()
		if dir.Len() == 0 {
			return
		}
		request.AX, request.AY = dir.X, dir.Y
	case models.MobaCastGround:
		if origin.Dist(aim) > skill.Profile.CastRange {
			return
		}
		request.AX, request.AY = aim.X, aim.Y
	default:
		return
	}
	m.startCast(p, request)
}

// dummyDodge steps out of an incoming telegraphed cast — but only when the
// dummy can genuinely clear the shape before the cast lands. That keeps fast
// skills viable and makes the dodge look like a real decision rather than a
// scripted teleport.
func (m *Match) dummyDodge(p *Participant) bool {
	threat := m.humanOpponent(p)
	if threat == nil || threat.Cast == nil {
		return false
	}
	cast := threat.Cast
	remaining := cast.EndTick - m.Tick
	if remaining <= 0 {
		return false
	}

	origin := Vec2{threat.X, threat.Y}
	point := Vec2{p.X, p.Y}
	if !castThreatens(cast, origin, point) {
		return false
	}

	escape, ok := escapeDirection(cast, origin, point)
	if !ok {
		return false
	}
	required := escapeDistance(cast, origin, point, escape)
	if math.IsInf(required, 1) {
		return false
	}
	speed := p.Speed * SpeedToTilesPerTick * p.SpeedMultiplier(m.Tick)
	if speed*float64(remaining) < required {
		return false
	}

	p.MoveIntent = escape
	return true
}

// escapeDirection picks which way to leave a telegraphed shape. Point-and-click
// casts cannot be dodged, so they report no direction.
func escapeDirection(cast *ActiveCast, origin, point Vec2) (Vec2, bool) {
	profile := cast.Skill.Profile
	switch profile.Shape {
	case models.MobaShapeCircle:
		// Straight out of the marker.
		escape := point.Sub(cast.AimPoint).Normalized()
		if escape.Len() == 0 {
			escape = Vec2{X: 1}
		}
		return escape, true
	case models.MobaShapeLine, models.MobaShapeCone:
		// Sideways out of the capsule/sector, on the nearer side.
		dir := cast.AimPoint.Sub(origin).Normalized()
		escape := Vec2{X: -dir.Y, Y: dir.X}
		if escape.Len() == 0 {
			return Vec2{}, false
		}
		offset := point.Sub(origin)
		if escape.X*offset.X+escape.Y*offset.Y < 0 {
			escape = escape.Scale(-1)
		}
		return escape, true
	default:
		return Vec2{}, false
	}
}

// escapeDistance walks along escape until the point leaves the shape. Numeric
// on purpose: no per-shape algebra to keep in sync with the hit tests.
func escapeDistance(cast *ActiveCast, origin, point, escape Vec2) float64 {
	const step = 0.25
	for travelled := step; travelled <= 24; travelled += step {
		if !castThreatens(cast, origin, point.Add(escape.Scale(travelled))) {
			return travelled
		}
	}
	return math.Inf(1)
}

// castThreatens reports whether a winding-up cast would hit point where it
// currently points. This is the same geometry the resolution uses, so what the
// telegraph shows is what actually lands.
func castThreatens(cast *ActiveCast, origin, point Vec2) bool {
	profile := cast.Skill.Profile
	dir := cast.AimPoint.Sub(origin).Normalized()
	switch profile.Shape {
	case models.MobaShapeCircle:
		return HitCircle(cast.AimPoint, point, profile.Radius)
	case models.MobaShapeLine:
		return HitLine(origin, dir, point, profile.Length, profile.Width)
	case models.MobaShapeCone:
		return HitCone(origin, dir, point, profile.Length, profile.AngleDeg)
	default:
		return false
	}
}

func (m *Match) humanOpponent(p *Participant) *Participant {
	for _, other := range m.Participants {
		if other.ID != p.ID && !other.IsDummy && other.alive() {
			return other
		}
	}
	return nil
}
