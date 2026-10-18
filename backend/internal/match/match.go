package match

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

const tickSeconds = float64(TickInterval) / float64(time.Second)

// NewMatch builds an empty arena with the demo layout.
func NewMatch(id uuid.UUID) *Match {
	return &Match{
		ID:        id,
		Obstacles: DefaultArena(),
		pending:   map[uuid.UUID]Input{},
	}
}

// DefaultArena is the demo layout: four pillars and two walls. Geometry is
// static, so the client can render it from the create response.
func DefaultArena() []Obstacle {
	return []Obstacle{
		{Kind: "circle", X: 12, Y: 7, R: 2},
		{Kind: "circle", X: 12, Y: 17, R: 2},
		{Kind: "circle", X: 28, Y: 7, R: 2},
		{Kind: "circle", X: 28, Y: 17, R: 2},
		{Kind: "rect", X: 18, Y: 2, W: 4, H: 2},
		{Kind: "rect", X: 18, Y: 20, W: 4, H: 2},
	}
}

func (m *Match) bounds() Vec2 { return Vec2{ArenaWidth, ArenaHeight} }

func ticksForMs(ms int) int {
	if ms <= 0 {
		return 0
	}
	return int(math.Ceil(float64(ms) / float64(TickInterval/time.Millisecond)))
}

// aimReach is how far a directional cast's telegraph extends, so its AimPoint
// is an absolute position rather than a bare direction.
func aimReach(profile models.MobaProfile) float64 {
	if profile.Length > 0 {
		return profile.Length
	}
	if profile.CastRange > 0 {
		return profile.CastRange
	}
	return 1
}

// AddParticipant registers a fighter. Callers must set Cooldowns via newCooldowns.
func (m *Match) AddParticipant(p *Participant) {
	if p.Cooldowns == nil {
		p.Cooldowns = map[string]int{}
	}
	m.Participants = append(m.Participants, p)
}

func (m *Match) find(id uuid.UUID) *Participant {
	for _, p := range m.Participants {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// HasPlayer reports whether the match contains the player. Locked, because the
// manager inspects matches that the tick loop may be stepping.
func (m *Match) HasPlayer(playerID uuid.UUID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.find(playerID) != nil
}

// IsEnded reports whether the match is over.
func (m *Match) IsEnded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Ended
}

// ShouldReap reports whether a finished match has stayed readable long enough
// for the last frames and the result to reach the client.
func (m *Match) ShouldReap() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Ended && m.Tick > m.EndTick+reapGraceTicks
}

// Step advances the match by exactly one tick. It is the only place that
// mutates authoritative state, so it runs entirely under the match lock.
func (m *Match) Step() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Ended {
		return
	}

	m.Tick++
	m.events = m.events[:0]

	// 1. Consume buffered input.
	for _, p := range m.Participants {
		input, ok := m.pending[p.ID]
		if !ok {
			continue
		}
		delete(m.pending, p.ID)
		m.applyInput(p, input)
		p.Seq = input.Seq
	}

	// 2. Dummy decides on its own.
	for _, p := range m.Participants {
		if p.IsDummy {
			m.dummyThink(p)
		}
	}

	// 3. Movement.
	for _, p := range m.Participants {
		if p.alive() {
			m.moveParticipant(p)
		}
	}

	// 4. Casts that finished winding up.
	for _, p := range m.Participants {
		if p.Cast != nil && m.Tick >= p.Cast.EndTick {
			m.resolveCast(p)
			p.Cast = nil
		}
	}

	// 5. Projectiles.
	m.advanceProjectiles()

	// 6. Timed effects and regeneration.
	for _, p := range m.Participants {
		p.Statuses = pruneStatuses(p.Statuses, m.Tick)
		if p.alive() && p.Mana < p.MaxMana {
			p.Mana = math.Min(p.MaxMana, p.Mana+ManaRegenPerTick)
		}
	}

	// 7. Outcome.
	m.checkEnd()

	// 8. Push the frame.
	m.broadcast()
}

func pruneStatuses(statuses []Status, tick int) []Status {
	kept := statuses[:0]
	for _, status := range statuses {
		if status.UntilTick > tick {
			kept = append(kept, status)
		}
	}
	return kept
}

func (m *Match) applyInput(p *Participant, input Input) {
	if !p.alive() {
		return
	}
	if input.Move != nil {
		move := Vec2{input.Move.DX, input.Move.DY}
		if move.Len() > 1 {
			move = move.Normalized()
		}
		p.MoveIntent = move
	}
	for _, request := range input.Casts {
		m.startCast(p, request)
	}
}

// startCast is the authority check for every cast: mode match, cooldown, mana,
// recovery and range. The client only ever sends intent.
func (m *Match) startCast(p *Participant, request CastInput) {
	if !p.alive() || p.Cast != nil || m.Tick < p.RecoveryUntilTick {
		return
	}
	skill, ok := p.skill(request.GuID, request.Rank)
	if !ok {
		return
	}
	if ready, ok := p.Cooldowns[skill.GuID]; ok && m.Tick < ready {
		return
	}
	if p.Mana < skill.ManaCost {
		return
	}
	if request.Mode != skill.Profile.CastMode {
		return
	}

	origin := Vec2{p.X, p.Y}
	cast := &ActiveCast{Skill: skill, Mode: request.Mode, StartTick: m.Tick}

	switch skill.Profile.CastMode {
	case models.MobaCastTarget:
		targetID, err := uuid.Parse(request.TargetID)
		if err != nil {
			return
		}
		target := m.find(targetID)
		if target == nil || !target.alive() || target.ID == p.ID {
			return
		}
		if origin.Dist(Vec2{target.X, target.Y}) > skill.Profile.CastRange {
			return
		}
		cast.TargetID = targetID
		cast.AimPoint = Vec2{target.X, target.Y}
	case models.MobaCastDirection:
		dir := Vec2{request.AX, request.AY}.Normalized()
		if dir.Len() == 0 {
			dir = Vec2{math.Cos(p.Facing), math.Sin(p.Facing)}
			if dir.Len() == 0 {
				dir = Vec2{1, 0}
			}
		}
		cast.AimPoint = origin.Add(dir.Scale(aimReach(skill.Profile)))
	case models.MobaCastGround:
		aim := Vec2{request.AX, request.AY}
		if origin.Dist(aim) > skill.Profile.CastRange {
			dir := aim.Sub(origin).Normalized()
			if dir.Len() == 0 {
				dir = Vec2{1, 0}
			}
			aim = origin.Add(dir.Scale(skill.Profile.CastRange))
		}
		cast.AimPoint = aim
	case models.MobaCastSelf:
		cast.AimPoint = origin
	default:
		return
	}

	if delta := cast.AimPoint.Sub(origin); delta.Len() > 0 {
		p.Facing = delta.Angle()
	}

	cast.EndTick = m.Tick + ticksForMs(skill.Profile.CastTimeMs)
	p.Cast = cast
	p.Mana -= skill.ManaCost
	p.Cooldowns[skill.GuID] = m.Tick + ticksForMs(skill.Profile.CooldownMs)

	m.emit(Event{
		Type: "cast_start", Source: p.ID.String(), GuID: skill.GuID,
		X: p.X, Y: p.Y,
	})
}

// resolveCast turns a finished wind-up into damage, a shield, a blink or a
// projectile.
func (m *Match) resolveCast(p *Participant) {
	cast := p.Cast
	profile := cast.Skill.Profile
	p.RecoveryUntilTick = m.Tick + ticksForMs(profile.RecoveryMs)

	origin := Vec2{p.X, p.Y}
	dir := cast.AimPoint.Sub(origin).Normalized()

	m.emit(Event{Type: "cast_end", Source: p.ID.String(), GuID: cast.Skill.GuID, X: p.X, Y: p.Y})

	// Blink first: the destination becomes the origin for the hit test.
	if profile.MoveDistance > 0 {
		destination := clampToBounds(origin.Add(dir.Scale(profile.MoveDistance)), m.bounds())
		if !blockedByObstacle(destination, m.Obstacles) {
			p.X, p.Y = destination.X, destination.Y
			origin = destination
			m.emit(Event{Type: "blink", Source: p.ID.String(), X: p.X, Y: p.Y})
		}
	}

	if profile.ProjectileSpeed > 0 {
		m.spawnProjectile(p, cast, dir)
		return
	}

	m.applyPayload(p, cast.Skill, m.resolveShape(p, cast, origin, dir))
}

// resolveShape is the hit test. Targets come back nearest-first so MaxTargets
// and non-piercing lines behave predictably.
func (m *Match) resolveShape(caster *Participant, cast *ActiveCast, origin, dir Vec2) []*Participant {
	profile := cast.Skill.Profile
	hits := make([]*Participant, 0, len(m.Participants))
	for _, other := range m.Participants {
		if other.ID == caster.ID || !other.alive() {
			continue
		}
		point := Vec2{other.X, other.Y}
		var hit bool
		switch profile.Shape {
		case models.MobaShapeSingle:
			if profile.CastMode == models.MobaCastTarget {
				hit = other.ID == cast.TargetID
			} else {
				hit = HitCircle(cast.AimPoint, point, 0.75)
			}
		case models.MobaShapeCircle:
			hit = HitCircle(cast.AimPoint, point, profile.Radius)
		case models.MobaShapeLine:
			hit = HitLine(origin, dir, point, profile.Length, profile.Width)
		case models.MobaShapeCone:
			hit = HitCone(origin, dir, point, profile.Length, profile.AngleDeg)
		}
		if hit {
			hits = append(hits, other)
		}
	}
	if len(hits) == 0 {
		return nil
	}

	sort.SliceStable(hits, func(i, j int) bool {
		return origin.Dist(Vec2{hits[i].X, hits[i].Y}) < origin.Dist(Vec2{hits[j].X, hits[j].Y})
	})
	if profile.Shape == models.MobaShapeLine && !profile.Pierce {
		hits = hits[:1]
	}
	if profile.MaxTargets > 0 && len(hits) > profile.MaxTargets {
		hits = hits[:profile.MaxTargets]
	}
	return hits
}

func (m *Match) applyPayload(caster *Participant, skill Skill, targets []*Participant) {
	for _, target := range targets {
		m.damage(caster, target, skill)
	}
	if skill.Payload.ShieldFlat > 0 {
		caster.Shield += skill.Payload.ShieldFlat
		m.emit(Event{Type: "shield", Source: caster.ID.String(), Amount: skill.Payload.ShieldFlat, X: caster.X, Y: caster.Y})
	}
}

// damage reuses the simulator's defense floor so realtime and turn-based combat
// agree on how a hit resolves.
func (m *Match) damage(caster, target *Participant, skill Skill) {
	raw := skill.Payload.DamageFlat + skill.Payload.DamageMult*caster.Attack
	amount := 0.0
	if raw > 0 {
		amount = simulator.ResolveBaseDamage(raw, target.Defense)
	}

	if absorbed := math.Min(target.Shield, amount); absorbed > 0 {
		target.Shield -= absorbed
		amount -= absorbed
		m.emit(Event{Type: "shield_absorb", Target: target.ID.String(), Amount: absorbed, X: target.X, Y: target.Y})
	}
	if amount > 0 {
		target.HP = math.Max(0, target.HP-amount)
	}

	for _, application := range skill.Payload.Statuses {
		chance := application.Chance
		if chance <= 0 {
			chance = 1
		}
		if chance < 1 && rand.Float64() > chance {
			continue
		}
		target.Statuses = append(target.Statuses, Status{ID: application.ID, UntilTick: m.Tick + application.Duration})
	}

	m.emit(Event{
		Type: "hit", Source: caster.ID.String(), Target: target.ID.String(),
		GuID: skill.GuID, Damage: amount, X: target.X, Y: target.Y,
	})
	if !target.alive() {
		m.emit(Event{Type: "death", Target: target.ID.String(), X: target.X, Y: target.Y})
	}
}

func (m *Match) moveParticipant(p *Participant) {
	// Casting roots the actor, which keeps the telegraph stable and gives the
	// opponent a readable window to step out of it.
	if p.Cast != nil || p.MoveIntent.Len() == 0 {
		return
	}
	speed := p.Speed * SpeedToTilesPerTick * p.SpeedMultiplier(m.Tick)
	if speed <= 0 {
		return
	}
	delta := p.MoveIntent.Scale(speed)
	next := slideMove(Vec2{p.X, p.Y}, delta, m.Obstacles, m.bounds())
	p.X, p.Y = next.X, next.Y
	if p.Cast == nil {
		p.Facing = delta.Angle()
	}
}

func (m *Match) spawnProjectile(caster *Participant, cast *ActiveCast, dir Vec2) {
	m.nextProjectileID++
	m.Projectiles = append(m.Projectiles, &Projectile{
		ID:     m.nextProjectileID,
		Owner:  caster.ID,
		Skill:  cast.Skill,
		Pos:    Vec2{caster.X, caster.Y},
		Origin: Vec2{caster.X, caster.Y},
		Dir:    dir,
		Width:  cast.Skill.Profile.Width,
		Speed:  cast.Skill.Profile.ProjectileSpeed,
		Range:  cast.Skill.Profile.CastRange,
		Hit:    map[uuid.UUID]bool{},
	})
}

func (m *Match) advanceProjectiles() {
	kept := m.Projectiles[:0]
	for _, projectile := range m.Projectiles {
		projectile.Pos = projectile.Pos.Add(projectile.Dir.Scale(projectile.Speed * tickSeconds))
		travelled := projectile.Origin.Dist(projectile.Pos)
		if travelled > projectile.Range {
			continue
		}

		caster := m.find(projectile.Owner)
		consumed := false
		for _, other := range m.Participants {
			if other.ID == projectile.Owner || !other.alive() || projectile.Hit[other.ID] {
				continue
			}
			if !HitLine(projectile.Origin, projectile.Dir, Vec2{other.X, other.Y}, travelled, projectile.Width) {
				continue
			}
			projectile.Hit[other.ID] = true
			if caster != nil {
				m.damage(caster, other, projectile.Skill)
			}
			if !projectile.Skill.Profile.Pierce {
				consumed = true
				break
			}
		}
		if !consumed {
			kept = append(kept, projectile)
		}
	}
	m.Projectiles = kept
}

func (m *Match) checkEnd() {
	alive := make([]*Participant, 0, len(m.Participants))
	for _, p := range m.Participants {
		if p.alive() {
			alive = append(alive, p)
		}
	}
	switch {
	case len(alive) == 0:
		m.finish(uuid.Nil, "draw")
	case len(alive) == 1:
		m.finish(alive[0].ID, "defeat")
	case m.Tick >= MatchTimeoutTicks:
		best := alive[0]
		for _, p := range alive[1:] {
			if p.HP/p.MaxHP > best.HP/best.MaxHP {
				best = p
			}
		}
		m.finish(best.ID, "timeout")
	}
}

func (m *Match) finish(winner uuid.UUID, reason string) {
	m.Ended = true
	m.EndTick = m.Tick
	m.Winner = winner
	m.Reason = reason
	m.emit(Event{Type: "match_end", Target: winner.String()})
}

// skill looks up a castable Gu in the participant's kit.
func (p *Participant) skill(guID string, rank int) (Skill, bool) {
	if guID == "BASIC_ATTACK" {
		return BasicAttackSkill(p.Attack), true
	}
	for _, skill := range p.Kit {
		if skill.GuID == guID && (rank <= 0 || skill.Rank == rank) {
			return skill, true
		}
	}
	return Skill{}, false
}

func (m *Match) emit(event Event) {
	event.Tick = m.Tick
	m.events = append(m.events, event)
}
