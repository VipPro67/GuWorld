package sim

import (
	"math"
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// Skill resolution engine. resolveCastLocked owns the authoritative rules for
// turning a windup-finished cast into state deltas: success/backfire chance,
// target acquisition per skill mode, projectile spawning, AoE shape filtering
// (isInsideGroundArea, defined in combat_geometry.go), movement dash and
// caster-side heal/shield support. It mutates the zone through the shared
// locked helpers on *World so every side effect (events, revisions, threat,
// statuses) stays on one code path.

func (w *World) resolveCastLocked(caster *Entity) {
	pending := caster.Combat.Pending
	caster.Combat.Pending = nil
	caster.Revision++
	if pending == nil {
		return
	}
	skill := pending.Skill
	if skill.SuccessChance > 0 && w.rng.Float64() >= skill.SuccessChance {
		if skill.BackfireHPPercent > 0 {
			backfire := caster.Vitals.MaxHP * skill.BackfireHPPercent
			w.applyDamageLocked(caster, caster, backfire, "KILLER_MOVE_BACKFIRE")
		}
		w.emitEvent(combat.Event{Type: combat.EventCastRejected, SourceID: string(caster.ID), SkillID: skill.ID, Reason: "backfire"})
		return
	}
	recordKillerMoveUse := func() {
		if skill.KillerMoveID != "" {
			caster.Combat.KillerMoveUses[skill.Slot]++
		}
	}
	// Recovery runs after the cast resolves. It gates the next cast/basic
	// attack and scales ground movement according to the authored policy.
	if skill.RecoveryTicks > 0 || skill.MovementPolicy != "" {
		caster.Combat.RecoveryUntil = w.tick + skill.RecoveryTicks
		caster.Combat.RecoveryPolicy = skill.MovementPolicy
	}
	if skill.FormActivation {
		caster.Combat.FormActive = true
		caster.Combat.FormManaDrainPerTick = skill.FormManaDrainPerTick
		caster.Combat.FormDefenseBonus = skill.FormDefenseBonus
		caster.Revision++
		return
	}
	switch skill.Mode {
	case combat.ModeTarget:
		var target *Entity
		if pending.TargetID != "" {
			target = w.entities[pending.TargetID]
			if target == nil || !w.canApplySkill(caster, target, skill) || distance(caster.Transform.Position, target.Transform.Position) > skill.Range {
				w.emitEvent(combat.Event{Type: combat.EventCastRejected, SourceID: string(caster.ID), SkillID: skill.ID, Reason: "target_lost"})
				return
			}
		} else {
			aim := Vec2{X: pending.Point.X - caster.Transform.Position.X, Y: pending.Point.Y - caster.Transform.Position.Y}
			reach := math.Min(skill.Range, math.Hypot(aim.X, aim.Y))
			target = w.firstCollisionAlongAim(caster, aim, reach, combat.AimedHitPadding, func(candidate *Entity) bool {
				return w.canApplySkill(caster, candidate, skill)
			})
			if target == nil {
				break
			}
		}
		recordKillerMoveUse()
		w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(caster.ID), TargetID: string(target.ID), SkillID: skill.ID, ImpactVFX: skill.Visuals.ImpactVFX, X: target.Transform.Position.X, Y: target.Transform.Position.Y})
		w.applyDamageLocked(caster, target, skill.Damage, skill.ID)
		if target.Vitals.HP > 0 {
			w.applyStatusLocked(caster, target, skill)
			if skill.Cleanse {
				w.cleanseDebuffsLocked(target)
			}
			if skill.KnockbackDistance > 0 && target.ID != caster.ID {
				w.applyKnockbackLocked(caster, target, skill.KnockbackDistance)
			}
		}
	case combat.ModeDirection:
		length := math.Hypot(pending.Aim.X, pending.Aim.Y)
		if length <= 0.01 {
			w.emitEvent(combat.Event{Type: combat.EventCastRejected, SourceID: string(caster.ID), SkillID: skill.ID, Reason: "invalid_aim"})
			return
		}
		direction := Vec2{X: pending.Aim.X / length, Y: pending.Aim.Y / length}
		baseAngle := math.Atan2(direction.Y, direction.X)
		spreadCount := skill.SpreadCount
		if spreadCount <= 0 {
			spreadCount = 1
		}
		spreadAngleRad := skill.SpreadAngleDeg * math.Pi / 180
		if spreadAngleRad <= 0 && spreadCount > 1 {
			spreadAngleRad = 30 * math.Pi / 180
		}
		for i := 0; i < spreadCount; i++ {
			angle := baseAngle
			if spreadCount > 1 {
				fraction := float64(i)/float64(spreadCount-1) - 0.5
				angle = baseAngle + fraction*spreadAngleRad
			}
			dir := Vec2{X: math.Cos(angle), Y: math.Sin(angle)}
			spawnOffset := math.Min(0.7, skill.Range)
			start := Vec2{X: caster.Transform.Position.X + dir.X*spawnOffset, Y: caster.Transform.Position.Y + dir.Y*spawnOffset}
			w.nextProjectileID++
			projectile := &projectile{
				ID: w.nextProjectileID, SourceID: caster.ID, SkillID: skill.ID, Skill: skill,
				RangeOrigin: caster.Transform.Position, Position: start,
				Velocity: Vec2{X: dir.X * skill.ProjectileSpeed / TickRate, Y: dir.Y * skill.ProjectileSpeed / TickRate},
				Radius:   skill.ProjectileRadius, Damage: skill.Damage, Travelled: spawnOffset, MaxTravel: skill.Range,
				ExpireTick:      w.tick + skill.ProjectileTicks,
				PierceRemaining: skill.PierceCount,
				HitEntities:     make(map[EntityID]bool),
			}
			w.projectiles = append(w.projectiles, projectile)
			w.emitEvent(combat.Event{
				Type: combat.EventProjectileSpawn, SourceID: string(caster.ID), SkillID: skill.ID, ProjectileVFX: skill.Visuals.ProjectileVFX,
				X: start.X, Y: start.Y, DurationTicks: skill.ProjectileTicks,
			})
		}
	case combat.ModeGround:
		type candidate struct {
			entity   *Entity
			distance float64
		}
		candidates := make([]candidate, 0)
		for _, target := range w.entities {
			if !w.canHarm(caster, target) {
				continue
			}
			if w.hasStatus(target, "FLYING") {
				continue
			}
			if !isInsideGroundArea(caster.Transform.Position, pending.Point, pending.Aim, target.Transform.Position, target.Radius, skill) {
				continue
			}
			targetDistance := distance(target.Transform.Position, pending.Point)
			candidates = append(candidates, candidate{entity: target, distance: targetDistance})
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].distance != candidates[j].distance {
				return candidates[i].distance < candidates[j].distance
			}
			return candidates[i].entity.ID < candidates[j].entity.ID
		})
		if skill.MaxTargets > 0 && len(candidates) > skill.MaxTargets {
			candidates = candidates[:skill.MaxTargets]
		}
		for _, hit := range candidates {
			w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(caster.ID), TargetID: string(hit.entity.ID), SkillID: skill.ID, ImpactVFX: skill.Visuals.ImpactVFX, X: hit.entity.Transform.Position.X, Y: hit.entity.Transform.Position.Y})
			w.applyDamageLocked(caster, hit.entity, skill.Damage, skill.ID)
			if hit.entity.Vitals.HP > 0 {
				w.applyStatusLocked(caster, hit.entity, skill)
				if skill.Cleanse {
					w.cleanseDebuffsLocked(hit.entity)
				}
				if skill.KnockbackDistance > 0 && hit.entity.ID != caster.ID {
					w.applyKnockbackLocked(caster, hit.entity, skill.KnockbackDistance)
				}
			}
		}
	case combat.ModeSelf:
		recordKillerMoveUse()
		w.applyStatusLocked(caster, caster, skill)
		if skill.Cleanse {
			w.cleanseDebuffsLocked(caster)
		}
	}
	if skill.MoveDistance > 0 && caster != nil && caster.Vitals.HP > 0 {
		var dir Vec2
		if skill.Mode == combat.ModeDirection && math.Hypot(pending.Aim.X, pending.Aim.Y) > 0.01 {
			aimLen := math.Hypot(pending.Aim.X, pending.Aim.Y)
			dir = Vec2{X: pending.Aim.X / aimLen, Y: pending.Aim.Y / aimLen}
		} else if skill.Mode == combat.ModeGround && distance(caster.Transform.Position, pending.Point) > 0.01 {
			pDist := distance(caster.Transform.Position, pending.Point)
			dir = Vec2{X: (pending.Point.X - caster.Transform.Position.X) / pDist, Y: (pending.Point.Y - caster.Transform.Position.Y) / pDist}
		} else {
			dir = Vec2{X: 1, Y: 0}
		}
		dest := Vec2{
			X: caster.Transform.Position.X + dir.X*skill.MoveDistance,
			Y: caster.Transform.Position.Y + dir.Y*skill.MoveDistance,
		}
		if !w.blocked(dest, caster.Radius) {
			caster.Transform.Position = dest
			caster.Revision++
		}
	}
	// Heal and shield always resolve on the caster, matching the existing Gu contract where HEAL/SHIELD operations land on the
	// acting character even
	// when the Gu's target mode is TARGET, DIRECTION or GROUND.
	w.applySelfSupportLocked(caster, skill)
}

// applySelfSupportLocked is the single out-of-target healing/shield path for
// every cast mode. It runs after damage/status resolution so a failed target
// validation never heals the caster.
func (w *World) applySelfSupportLocked(caster *Entity, skill combat.Skill) {
	if caster == nil || caster.Vitals.HP <= 0 {
		return
	}
	if skill.Heal > 0 {
		healVal := skill.Heal
		if w.hasStatus(caster, "BURN") {
			healVal *= 0.5
		}
		before := caster.Vitals.HP
		caster.Vitals.HP = math.Min(caster.Vitals.MaxHP, caster.Vitals.HP+healVal)
		caster.Revision++
		w.emitEvent(combat.Event{Type: combat.EventHealApplied, SourceID: string(caster.ID), TargetID: string(caster.ID), SkillID: skill.ID, Amount: caster.Vitals.HP - before, HP: caster.Vitals.HP})
	}
	if skill.Shield > 0 {
		caster.Vitals.Shield += skill.Shield
		caster.Revision++
		w.emitEvent(combat.Event{Type: combat.EventShieldApplied, SourceID: string(caster.ID), TargetID: string(caster.ID), SkillID: skill.ID, Amount: skill.Shield, Shield: caster.Vitals.Shield, HP: caster.Vitals.HP})
	}
}
