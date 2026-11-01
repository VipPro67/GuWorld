package sim

import (
	"math"
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// dashState belongs to the zone tick owner. Duration and distance are fixed at
// cast resolution; ordinary movement, haste, root and slow cannot alter it.
type dashState struct {
	skill     combat.Skill
	direction Vec2
	remaining float64
	ticksLeft uint64
}

func (w *World) stepDashesLocked() {
	ids := make([]EntityID, 0)
	for id, entity := range w.entities {
		if entity.dash != nil {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		caster := w.entities[id]
		if caster == nil || caster.dash == nil {
			continue
		}
		dash := caster.dash
		if caster.Vitals.HP <= 0 || w.isStunned(caster) || dash.ticksLeft == 0 {
			w.endDashLocked(caster, "interrupted")
			continue
		}
		travel := dash.remaining / float64(dash.ticksLeft)
		steps := int(math.Ceil(travel / MaxMovementSubstep))
		if steps < 1 {
			steps = 1
		}
		stepDistance := travel / float64(steps)
		stopped := false
		for i := 0; i < steps; i++ {
			from := caster.Transform.Position
			to := Vec2{X: from.X + dash.direction.X*stepDistance, Y: from.Y + dash.direction.Y*stepDistance}
			safeDistance := stepDistance
			wall := w.blocked(to, caster.Radius)
			if wall {
				low, high := 0.0, stepDistance
				for attempt := 0; attempt < 12; attempt++ {
					mid := (low + high) / 2
					candidate := Vec2{X: from.X + dash.direction.X*mid, Y: from.Y + dash.direction.Y*mid}
					if w.blocked(candidate, caster.Radius) {
						high = mid
					} else {
						low = mid
					}
				}
				safeDistance = low
			}
			if dash.skill.MoveBehavior == "IMPACT" {
				target := w.firstCollisionAlongAim(caster, dash.direction, safeDistance, caster.Radius, func(other *Entity) bool {
					return w.canHarm(caster, other)
				})
				if target != nil {
					contact := dashContactDistance(from, dash.direction, target.Transform.Position, caster.Radius+target.Radius)
					advance := math.Max(0, math.Min(safeDistance, contact-0.001))
					caster.Transform.Position = Vec2{X: from.X + dash.direction.X*advance, Y: from.Y + dash.direction.Y*advance}
					caster.Revision++
					w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(caster.ID), TargetID: string(target.ID), SkillID: dash.skill.ID, ImpactVFX: dash.skill.Visuals.ImpactVFX, X: target.Transform.Position.X, Y: target.Transform.Position.Y})
					w.applyDamageLocked(caster, target, dash.skill.Damage, dash.skill.ID)
					if target.Vitals.HP > 0 {
						w.applyStatusLocked(caster, target, dash.skill)
						if dash.skill.KnockbackDistance > 0 {
							w.applyKnockbackLocked(caster, target, dash.skill.KnockbackDistance)
						}
					}
					stopped = true
					break
				}
			}
			caster.Transform.Position = Vec2{X: from.X + dash.direction.X*safeDistance, Y: from.Y + dash.direction.Y*safeDistance}
			caster.Revision++
			dash.remaining -= safeDistance
			if wall {
				stopped = true
				break
			}
		}
		dash.ticksLeft--
		if stopped || dash.ticksLeft == 0 || dash.remaining <= 0.00001 {
			reason := "complete"
			if stopped {
				reason = "stopped"
			}
			w.endDashLocked(caster, reason)
		}
	}
}

func (w *World) endDashLocked(caster *Entity, reason string) {
	skillID := caster.dash.skill.ID
	caster.dash = nil
	caster.Revision++
	w.emitEvent(combat.Event{Type: combat.EventDashEnded, SourceID: string(caster.ID), SkillID: skillID, Reason: reason})
}

func dashContactDistance(from, direction, center Vec2, radius float64) float64 {
	dx, dy := center.X-from.X, center.Y-from.Y
	along := dx*direction.X + dy*direction.Y
	lateralSq := dx*dx + dy*dy - along*along
	return math.Max(0, along-math.Sqrt(math.Max(0, radius*radius-lateralSq)))
}
