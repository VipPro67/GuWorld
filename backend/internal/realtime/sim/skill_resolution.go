package sim

import (
	"math"
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// Skill resolution engine. resolveCastLocked owns the authoritative rules for
// turning a windup-finished cast into state deltas via a structured execution pipeline:
//  1. Cast Pre-Check & Lifecycle (success/backfire, killer move usage, recovery locking)
//  2. Form & Transformation Resolution
//  3. Motion & Displacement Resolution (Dash/Blink)
//  4. Delivery Resolution (Target raycast, Direction projectiles, Ground geometry)
//  5. Unified Target Payload Application (Damage, Status, Cleanse, Knockback)
//  6. Self-Support Application (Caster Heal & Shield)
func (w *World) resolveCastLocked(caster *Entity) {
	pending := caster.Combat.Pending
	caster.Combat.Pending = nil
	caster.Revision++
	if pending == nil {
		return
	}
	if pending.IsCapture {
		w.resolveCaptureLocked(caster, pending)
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
		w.resolveFormActivationLocked(caster, skill)
		return
	}
	if skill.MoveDistance > 0 {
		recordKillerMoveUse()
		w.resolveMotionSkillLocked(caster, pending, skill)
		w.applySelfSupportLocked(caster, skill)
		return
	}
	switch skill.Mode {
	case combat.ModeTarget:
		target, ok := w.resolveTargetModeLocked(caster, pending, skill)
		if !ok || target == nil {
			return
		}
		recordKillerMoveUse()
		w.applyTargetPayloadLocked(caster, target, skill)

	case combat.ModeDirection:
		w.resolveDirectionModeLocked(caster, pending, skill)

	case combat.ModeGround:
		targets := w.resolveGroundModeLocked(caster, pending, skill)
		for _, target := range targets {
			w.applyTargetPayloadLocked(caster, target, skill)
		}

	case combat.ModeSelf:
		recordKillerMoveUse()
		w.applyStatusLocked(caster, caster, skill)
		if skill.Cleanse {
			w.cleanseDebuffsLocked(caster)
		}
	}
	// Heal and shield always resolve on the caster, matching the existing Gu contract where HEAL/SHIELD operations land on the
	// acting character even when the Gu's target mode is TARGET, DIRECTION or GROUND.
	w.applySelfSupportLocked(caster, skill)
}

// applyTargetPayloadLocked is the single authoritative applier for on-hit payloads:
// confirms hit event, deals damage, applies statuses, executes cleanse and applies knockback.
func (w *World) applyTargetPayloadLocked(caster, target *Entity, skill combat.Skill) {
	if caster == nil || target == nil {
		return
	}
	w.emitEvent(combat.Event{
		Type:      combat.EventHitConfirmed,
		SourceID:  string(caster.ID),
		TargetID:  string(target.ID),
		SkillID:   skill.ID,
		ImpactVFX: skill.Visuals.ImpactVFX,
		X:         target.Transform.Position.X,
		Y:         target.Transform.Position.Y,
	})
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
}

func (w *World) resolveFormActivationLocked(caster *Entity, skill combat.Skill) {
	formID := skill.FormID
	if formID == "" {
		formID = skill.ID
	}
	rec, found := GetTransformationForm(formID)
	formKind := TransformationFull
	if skill.FormKind != "" {
		formKind = TransformationKind(skill.FormKind)
	} else if found {
		formKind = rec.FormKind
	}
	modelScale := 1.5
	radiusMult := 1.4
	maxHP := 150.0
	formDefense := skill.FormDefenseBonus
	damageRed := 0.15
	manaUpkeepSec := skill.FormManaDrainPerTick * float64(TickRate)
	if found {
		modelScale = rec.ModelScale
		radiusMult = rec.CollisionRadiusMult
		maxHP = rec.MaxFormHP
		formDefense = rec.FormDefense
		damageRed = rec.DamageReduction
		manaUpkeepSec = rec.ManaUpkeepSec
	}
	if skill.FormModelScale > 0 {
		modelScale = skill.FormModelScale
	}
	if skill.FormCollisionRadius > 0 {
		radiusMult = skill.FormCollisionRadius
	}
	if skill.FormMaxHP > 0 {
		maxHP = skill.FormMaxHP
	}
	if skill.FormDamageReduction > 0 {
		damageRed = skill.FormDamageReduction
	}

	req := FormActivationRequest{
		FormID:              formID,
		FormName:            skill.Name,
		FormKind:            formKind,
		ModelScale:          modelScale,
		CollisionRadiusMult: radiusMult,
		MaxFormHP:           maxHP,
		FormDefenseBonus:    formDefense,
		DamageReduction:     damageRed,
		ManaUpkeepPerSecond: manaUpkeepSec,
	}
	w.completeTransformationLocked(caster, req)
}

func (w *World) resolveMotionSkillLocked(caster *Entity, pending *PendingCast, skill combat.Skill) {
	if !w.hasStatus(caster, "ROOT") && !w.hasStatus(caster, "BIND") && !w.isStunned(caster) {
		caster.dash = &dashState{skill: skill, direction: pending.Aim, remaining: skill.MoveDistance, ticksLeft: skill.MoveDurationTicks}
		caster.Revision++
	}
	if skill.Cleanse {
		w.cleanseDebuffsLocked(caster)
	}
}

func (w *World) resolveTargetModeLocked(caster *Entity, pending *PendingCast, skill combat.Skill) (*Entity, bool) {
	if pending.TargetID != "" {
		target := w.entities[pending.TargetID]
		if target == nil || !w.canApplySkill(caster, target, skill) || distance(caster.Transform.Position, target.Transform.Position) > skill.Range {
			w.emitEvent(combat.Event{Type: combat.EventCastRejected, SourceID: string(caster.ID), SkillID: skill.ID, Reason: "target_lost"})
			return nil, false
		}
		return target, true
	}
	aim := Vec2{X: pending.Point.X - caster.Transform.Position.X, Y: pending.Point.Y - caster.Transform.Position.Y}
	reach := math.Min(skill.Range, math.Hypot(aim.X, aim.Y))
	target := w.firstCollisionAlongAim(caster, aim, reach, combat.AimedHitPadding, func(candidate *Entity) bool {
		return w.canApplySkill(caster, candidate, skill)
	})
	if target == nil {
		// Targeted composites must not apply their self-support payload when no hostile target was hit.
		return nil, false
	}
	return target, true
}

func (w *World) resolveDirectionModeLocked(caster *Entity, pending *PendingCast, skill combat.Skill) {
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
}

func (w *World) resolveGroundModeLocked(caster *Entity, pending *PendingCast, skill combat.Skill) []*Entity {
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
	result := make([]*Entity, len(candidates))
	for i, c := range candidates {
		result[i] = c.entity
	}
	return result
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

func (w *World) resolveCaptureLocked(caster *Entity, pending *PendingCast) {
	if caster == nil || caster.Vitals.HP <= 0 {
		return
	}
	target := w.entities[pending.TargetID]
	if target == nil || target.Vitals.HP <= 0 || target.Kind != EntityMonster {
		w.emitEvent(combat.Event{
			Type:     combat.EventCaptureFailed,
			SourceID: string(caster.ID),
			TargetID: string(pending.TargetID),
			Reason:   "target_lost",
		})
		return
	}

	dist := distance(caster.Transform.Position, target.Transform.Position)
	if dist > 6.0 {
		w.emitEvent(combat.Event{
			Type:     combat.EventCaptureFailed,
			SourceID: string(caster.ID),
			TargetID: string(target.ID),
			Reason:   "out_of_range",
		})
		return
	}

	// Calculate capture chance:
	// P_capture = clamp((P_base * F_rank * F_hp * F_cc * F_soul) + B_gu, 0.02, 0.95)
	pBase := 0.40
	if target.AI.Rank >= 4 {
		pBase = 0.15
	} else if target.AI.Rank >= 2 {
		pBase = 0.25
	}

	casterRank := 1
	if caster.Companion.Rank > 0 {
		casterRank = caster.Companion.Rank
	}
	rankDiff := casterRank - target.AI.Rank
	fRank := math.Max(0.2, math.Min(2.0, 1.0+float64(rankDiff)*0.15))

	hpPct := target.Vitals.HP / math.Max(1.0, target.Vitals.MaxHP)
	fHP := 1.0 + (1.0-hpPct)*0.6

	fCC := 1.0
	if w.isStunned(target) || w.hasStatus(target, "BIND") || w.hasStatus(target, "ROOT") || w.hasStatus(target, "FREEZE") || w.hasStatus(target, "PETRIFY") {
		fCC = 1.3
	}

	fSoul := 1.0
	bGu := 0.15

	captureChance := math.Max(0.02, math.Min(0.95, (pBase*fRank*fHP*fCC*fSoul)+bGu))

	roll := w.rng.Float64()
	if roll < captureChance {
		w.emitEvent(combat.Event{
			Type:         combat.EventCaptureSuccess,
			SourceID:     string(caster.ID),
			TargetID:     string(target.ID),
			TemplateID:   target.AI.TemplateID,
			MonsterRank:  target.AI.Rank,
			PersistentID: target.AI.PersistentID,
		})
		w.despawnEntityLocked(target.ID)
	} else {
		w.emitEvent(combat.Event{
			Type:     combat.EventCaptureFailed,
			SourceID: string(caster.ID),
			TargetID: string(target.ID),
			Reason:   "resisted",
		})

		if target.AI.Rank >= 2 && w.rng.Float64() < 0.35 {
			backfireDmg := caster.Vitals.MaxHP * 0.10
			w.applyDamageLocked(target, caster, backfireDmg, "SOUL_BACKFIRE")
			w.applyStatusLocked(target, caster, combat.Skill{StatusID: "STUN", StatusDurationTicks: 40})
			w.emitEvent(combat.Event{
				Type:     combat.EventSoulBackfire,
				SourceID: string(caster.ID),
				TargetID: string(target.ID),
				Amount:   backfireDmg,
				Reason:   "will_rebound",
			})
		}
	}
}
