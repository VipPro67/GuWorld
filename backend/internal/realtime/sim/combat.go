package sim

import (
	"math"
	"sort"
	"strconv"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// CastCommand is a server-validated cast intent. It shares the movement
// sequence stream so the latest control command wins on the zone tick.
type CastCommand struct {
	EntityID   EntityID
	LeaseToken string
	Seq        uint64
	Slot       uint8
	Mode       combat.Mode
	TargetID   EntityID
	Aim        Vec2
	Point      Vec2
}

type AttackCommand struct {
	EntityID   EntityID
	LeaseToken string
	Seq        uint64
	TargetID   EntityID
}

type PendingCast struct {
	Skill       combat.Skill
	TargetID    EntityID
	Aim         Vec2
	Point       Vec2
	ResolveTick uint64
	Seq         uint64
}

// CombatState is server-owned per-player combat state.
type CombatState struct {
	Mana                     float64
	MaxMana                  float64
	ManaRegenPerSecond       float64
	Thought                  float64
	MaxThought               float64
	ThoughtRegenPerSecond    float64
	BasicAttackDamage        float64
	GCDUntil                 uint64
	RecoveryUntil            uint64
	RecoveryPolicy           string
	BasicAttackCooldownUntil uint64
	Cooldowns                [7]uint64
	Skills                   [7]combat.Skill
	GuHunger                 [7]float64
	GuUses                   [7]uint64
	Pending                  *PendingCast
	Statuses                 []StatusEffect
	Hidden                   combat.HiddenPassive
}

func (w *World) SubmitBasicAttack(command AttackCommand) error {
	if command.EntityID == "" || command.Seq == 0 || command.TargetID == "" || len(command.TargetID) > MaxEntityIDLength {
		return ErrInvalidInput
	}
	if len(command.LeaseToken) > MaxEntityIDLength {
		return ErrInvalidInput
	}
	attack := command
	select {
	case w.inbox <- InputCommand{EntityID: command.EntityID, LeaseToken: command.LeaseToken, Seq: command.Seq, Attack: &attack}:
		return nil
	default:
		return ErrQueueFull
	}
}

func (w *World) startBasicAttackLocked(attacker *Entity, command AttackCommand) {
	reject := func(reason string) {
		w.emitEvent(combat.Event{Type: combat.EventCastRejected, SourceID: string(attacker.ID), TargetID: string(command.TargetID), SkillID: "BASIC_ATTACK", Reason: reason})
	}
	if attacker == nil || attacker.Kind != EntityPlayer || attacker.Vitals.HP <= 0 {
		return
	}
	if w.isStunned(attacker) {
		reject("stunned")
		return
	}
	if attacker.Combat.Pending != nil {
		reject("already_casting")
		return
	}
	if w.tick < attacker.Combat.RecoveryUntil {
		reject("recovering")
		return
	}
	if w.tick < attacker.Combat.GCDUntil {
		reject("global_cooldown")
		return
	}
	if w.hasStatus(attacker, "BLIND") {
		attacker.Combat.BasicAttackCooldownUntil = w.tick + combat.BasicAttackCooldownTicks
		attacker.Revision++
		w.emitEvent(combat.Event{Type: combat.EventBasicAttack, SourceID: string(attacker.ID), TargetID: string(command.TargetID), SkillID: "BASIC_ATTACK", DurationTicks: 2})
		w.emitEvent(combat.Event{Type: combat.EventDamageApplied, SourceID: string(attacker.ID), TargetID: string(command.TargetID), SkillID: "BASIC_ATTACK", X: attacker.Transform.Position.X, Y: attacker.Transform.Position.Y, Amount: 0, HP: attacker.Vitals.HP, Shield: attacker.Vitals.Shield, Reason: "blind"})
		return
	}
	target := w.entities[command.TargetID]
	if !w.canHarm(attacker, target) {
		reject("invalid_target")
		return
	}
	if distance(attacker.Transform.Position, target.Transform.Position) > combat.BasicAttackRange {
		reject("out_of_range")
		return
	}
	if w.tick < attacker.Combat.BasicAttackCooldownUntil {
		reject("cooldown")
		return
	}
	attacker.Combat.BasicAttackCooldownUntil = w.tick + combat.BasicAttackCooldownTicks
	attacker.Revision++
	w.removeStatusLocked(attacker, "STEALTH")
	w.emitEvent(combat.Event{Type: combat.EventBasicAttack, SourceID: string(attacker.ID), TargetID: string(target.ID), SkillID: "BASIC_ATTACK", DurationTicks: 2})
	w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(attacker.ID), TargetID: string(target.ID), SkillID: "BASIC_ATTACK", X: target.Transform.Position.X, Y: target.Transform.Position.Y})
	damage := attacker.Combat.BasicAttackDamage
	if damage <= 0 {
		damage = combat.BasicAttackDamage
	}
	w.applyDamageLocked(attacker, target, damage, "BASIC_ATTACK")
}

// NewPlayerCombat returns the mana pool, hidden passive and slot loadout. When
// no loadout is supplied the server preview skills are used (tests/demo); the
// domain adapter supplies compiled Gu skills for real characters.
func NewPlayerCombat(loadout ...combat.Skill) CombatState {
	if len(loadout) == 0 {
		loadout = combat.DefaultLoadout()
	}
	return newPlayerCombat(loadout)
}

// NewPlayerCombatWithLoadout keeps an empty player hotbar empty. Live admission
// uses this so missing realtime Gu profiles cannot turn into generic skills.
func NewPlayerCombatWithLoadout(loadout []combat.Skill) CombatState {
	state := newPlayerCombat(loadout)
	// The preview proc is demo content, not an innate player ability. Authored
	// Gu hidden passives must be attached explicitly when their runtime exists.
	state.Hidden = combat.HiddenPassive{}
	return state
}

func newPlayerCombat(loadout []combat.Skill) CombatState {
	state := CombatState{
		Mana: combat.DefaultManaMax, MaxMana: combat.DefaultManaMax,
		ManaRegenPerSecond: combat.DefaultManaRegen,
		BasicAttackDamage:  combat.BasicAttackDamage,
		Hidden:             combat.PreviewHiddenPassive(),
	}
	for _, skill := range loadout {
		if skill.Slot >= 1 && skill.Slot <= 6 && skill.ID != "" {
			state.Skills[skill.Slot] = skill
			state.GuHunger[skill.Slot] = skill.GuHunger
		}
	}
	return state
}

type projectile struct {
	ID              uint64
	SourceID        EntityID
	SkillID         string
	Skill           combat.Skill
	Position        Vec2
	Velocity        Vec2
	Radius          float64
	Damage          float64
	ExpireTick      uint64
	PierceRemaining int
	HitEntities     map[EntityID]bool
}

// ProjectileState is the render-safe projection of a server projectile.
type ProjectileState struct {
	ID       uint64  `json:"id"`
	SourceID string  `json:"source_id"`
	SkillID  string  `json:"skill_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Radius   float64 `json:"radius"`
}

func (w *World) SubmitCast(command CastCommand) error {
	if command.EntityID == "" || command.Seq == 0 || command.Slot < 1 || command.Slot > 6 {
		return ErrInvalidInput
	}
	if command.Mode != combat.ModeTarget && command.Mode != combat.ModeDirection && command.Mode != combat.ModeGround && command.Mode != combat.ModeSelf {
		return ErrInvalidInput
	}
	if !finite(command.Aim.X) || !finite(command.Aim.Y) || !finite(command.Point.X) || !finite(command.Point.Y) {
		return ErrInvalidInput
	}
	if len(command.TargetID) > MaxEntityIDLength {
		return ErrInvalidInput
	}
	cast := command
	select {
	case w.inbox <- InputCommand{EntityID: command.EntityID, LeaseToken: command.LeaseToken, Seq: command.Seq, Cast: &cast}:
		return nil
	default:
		return ErrQueueFull
	}
}

func (w *World) startCastLocked(caster *Entity, command CastCommand) {
	if caster == nil {
		return
	}
	reject := func(reason string) {
		w.emitEvent(combat.Event{
			Type: combat.EventCastRejected, SourceID: string(caster.ID),
			SkillID: commandSkillID(command.Slot), Reason: reason,
		})
	}
	if caster.Kind != EntityPlayer || caster.Vitals.HP <= 0 {
		reject("caster_not_ready")
		return
	}
	if caster.Combat.Pending != nil {
		reject("already_casting")
		return
	}
	if w.isStunned(caster) {
		reject("stunned")
		return
	}
	if w.hasStatus(caster, "SILENCE") || w.hasStatus(caster, "GU_SEAL") || w.hasStatus(caster, "FORGET_EGO") {
		reject("silenced")
		return
	}
	skill := caster.Combat.Skills[command.Slot]
	if skill.ID == "" || !skill.Valid() {
		reject("unknown_skill")
		return
	}
	if skill.IsToggle && skill.StatusID != "" && w.hasStatus(caster, skill.StatusID) {
		w.removeStatusLocked(caster, skill.StatusID)
		caster.Revision++
		return
	}
	if command.Mode != skill.Mode {
		reject("wrong_target_mode")
		return
	}
	if w.tick < caster.Combat.RecoveryUntil {
		reject("recovering")
		return
	}
	if w.tick < caster.Combat.GCDUntil {
		reject("global_cooldown")
		return
	}
	if caster.Combat.Cooldowns[skill.Slot] > w.tick {
		reject("cooldown")
		return
	}
	pending := PendingCast{
		Skill: skill, Aim: command.Aim, Point: command.Point,
		TargetID: command.TargetID, ResolveTick: w.tick + skill.WindupTicks, Seq: command.Seq,
	}
	switch skill.Mode {
	case combat.ModeTarget:
		target := w.entities[command.TargetID]
		if target == nil || !w.canApplySkill(caster, target, skill) {
			reject("invalid_target")
			return
		}
		if distance(caster.Transform.Position, target.Transform.Position) > skill.Range {
			reject("out_of_range")
			return
		}
	case combat.ModeGround:
		if distance(caster.Transform.Position, command.Point) > skill.Range {
			reject("out_of_range")
			return
		}
		if w.blocked(command.Point, 0.1) {
			reject("invalid_ground")
			return
		}
	case combat.ModeDirection:
		if math.Hypot(command.Aim.X, command.Aim.Y) <= 0.01 {
			reject("invalid_aim")
			return
		}
	}
	if skill.GuInstanceID != "" && caster.Combat.GuHunger[skill.Slot] < 1 {
		reject("gu_exhausted")
		return
	}
	// Turn combat consumes one hunger from the exact Gu instance on an attempt,
	// before backlash and the mana check. The zone keeps the balance until the
	// next atomic player checkpoint settles its per-instance usage delta.
	if skill.GuInstanceID != "" {
		caster.Combat.GuHunger[skill.Slot]--
		caster.Combat.GuUses[skill.Slot]++
		caster.Revision++
	}
	if skill.BacklashPct > 0 {
		backlash := caster.Vitals.MaxHP * skill.BacklashPct
		before := caster.Vitals.HP
		caster.Vitals.HP = math.Max(0, before-backlash)
		caster.Revision++
		w.emitEvent(combat.Event{Type: combat.EventDamageApplied, SourceID: string(caster.ID), TargetID: string(caster.ID), SkillID: "GU_BACKLASH", X: caster.Transform.Position.X, Y: caster.Transform.Position.Y, Amount: before - caster.Vitals.HP, HP: caster.Vitals.HP, Shield: caster.Vitals.Shield})
		if caster.Vitals.HP == 0 {
			w.emitEvent(combat.Event{Type: combat.EventEntityDied, SourceID: string(caster.ID), TargetID: string(caster.ID), SkillID: "GU_BACKLASH", X: caster.Transform.Position.X, Y: caster.Transform.Position.Y})
			return
		}
	}
	if caster.Combat.Mana < skill.ManaCost {
		reject("not_enough_mana")
		return
	}

	caster.Combat.Mana -= skill.ManaCost
	caster.Combat.GCDUntil = w.tick + combat.DefaultGCDTicks
	caster.Combat.Cooldowns[skill.Slot] = w.tick + skill.CooldownTicks
	caster.Combat.Pending = &pending
	caster.Revision++
	caster.moveIntent = Vec2{}
	caster.moveTarget = nil
	if skill.RequiresEnemyTarget || skill.Damage > 0 {
		w.removeStatusLocked(caster, "STEALTH")
	}
	w.emitEvent(combat.Event{
		Type: combat.EventCastAccepted, SourceID: string(caster.ID), TargetID: string(command.TargetID),
		SkillID: skill.ID, X: command.Point.X, Y: command.Point.Y, DurationTicks: skill.WindupTicks,
	})
}

func (w *World) tickCombat() {
	if w.tick%combat.ManaRegenInterval == 0 {
		for _, entity := range w.entities {
			if entity.Kind != EntityPlayer || entity.Vitals.HP <= 0 {
				continue
			}
			changed := false
			if entity.Combat.MaxMana > 0 && entity.Combat.Mana < entity.Combat.MaxMana {
				entity.Combat.Mana = math.Min(entity.Combat.MaxMana, entity.Combat.Mana+entity.Combat.ManaRegenPerSecond)
				changed = true
			}
			if entity.Combat.MaxThought > 0 && entity.Combat.Thought < entity.Combat.MaxThought {
				entity.Combat.Thought = math.Min(entity.Combat.MaxThought, entity.Combat.Thought+entity.Combat.ThoughtRegenPerSecond)
				changed = true
			}
			if changed {
				entity.Revision++
			}
		}
	}
	for _, entity := range w.entities {
		if entity.Combat.Pending != nil && w.tick >= entity.Combat.Pending.ResolveTick {
			w.resolveCastLocked(entity)
		}
	}
	w.updateProjectilesLocked()
	w.tickStatuses()
}

func (w *World) resolveCastLocked(caster *Entity) {
	pending := caster.Combat.Pending
	caster.Combat.Pending = nil
	caster.Revision++
	if pending == nil {
		return
	}
	skill := pending.Skill
	// Recovery runs after the cast resolves. It gates the next cast/basic
	// attack and scales ground movement according to the authored policy.
	if skill.RecoveryTicks > 0 || skill.MovementPolicy != "" {
		caster.Combat.RecoveryUntil = w.tick + skill.RecoveryTicks
		caster.Combat.RecoveryPolicy = skill.MovementPolicy
	}
	switch skill.Mode {
	case combat.ModeTarget:
		target := w.entities[pending.TargetID]
		if target == nil || !w.canApplySkill(caster, target, skill) || distance(caster.Transform.Position, target.Transform.Position) > skill.Range {
			w.emitEvent(combat.Event{Type: combat.EventCastRejected, SourceID: string(caster.ID), SkillID: skill.ID, Reason: "target_lost"})
			return
		}
		w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(caster.ID), TargetID: string(target.ID), SkillID: skill.ID, X: target.Transform.Position.X, Y: target.Transform.Position.Y})
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
			start := Vec2{X: caster.Transform.Position.X + dir.X*0.7, Y: caster.Transform.Position.Y + dir.Y*0.7}
			w.nextProjectileID++
			projectile := &projectile{
				ID: w.nextProjectileID, SourceID: caster.ID, SkillID: skill.ID, Skill: skill,
				Position: start,
				Velocity: Vec2{X: dir.X * skill.ProjectileSpeed / TickRate, Y: dir.Y * skill.ProjectileSpeed / TickRate},
				Radius:   skill.ProjectileRadius, Damage: skill.Damage,
				ExpireTick:      w.tick + skill.ProjectileTicks,
				PierceRemaining: skill.PierceCount,
				HitEntities:     make(map[EntityID]bool),
			}
			w.projectiles = append(w.projectiles, projectile)
			w.emitEvent(combat.Event{
				Type: combat.EventProjectileSpawn, SourceID: string(caster.ID), SkillID: skill.ID,
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
			w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(caster.ID), TargetID: string(hit.entity.ID), SkillID: skill.ID, X: hit.entity.Transform.Position.X, Y: hit.entity.Transform.Position.Y})
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
	// Heal and shield always resolve on the caster, matching the legacy
	// simulator where HEAL/SHIELD operations land on the acting character even
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

func (w *World) updateProjectilesLocked() {
	remaining := w.projectiles[:0]
	for _, projectile := range w.projectiles {
		if w.tick > projectile.ExpireTick {
			continue
		}
		from := projectile.Position
		to := Vec2{X: from.X + projectile.Velocity.X, Y: from.Y + projectile.Velocity.Y}
		projectile.Position = to
		if w.blocked(to, projectile.Radius) {
			continue
		}
		source := w.entities[projectile.SourceID]
		if source == nil {
			continue
		}
		if projectile.HitEntities == nil {
			projectile.HitEntities = make(map[EntityID]bool)
		}
		ids := make([]EntityID, 0, len(w.entities))
		for id := range w.entities {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		hit := false
		for _, id := range ids {
			target := w.entities[id]
			if target.ID == projectile.SourceID || !w.canHarm(source, target) || projectile.HitEntities[target.ID] {
				continue
			}
			if segmentDistance(from, to, target.Transform.Position) <= projectile.Radius+target.Radius {
				projectile.HitEntities[target.ID] = true
				w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(projectile.SourceID), TargetID: string(target.ID), SkillID: projectile.SkillID, X: target.Transform.Position.X, Y: target.Transform.Position.Y})
				w.applyDamageLocked(source, target, projectile.Damage, projectile.SkillID)
				if target.Vitals.HP > 0 {
					w.applyStatusLocked(source, target, projectile.Skill)
					if projectile.Skill.Cleanse {
						w.cleanseDebuffsLocked(target)
					}
					if projectile.Skill.KnockbackDistance > 0 && target.ID != source.ID {
						w.applyKnockbackLocked(source, target, projectile.Skill.KnockbackDistance)
					}
				}
				if projectile.PierceRemaining > 0 {
					projectile.PierceRemaining--
				} else {
					hit = true
					break
				}
			}
		}
		if !hit {
			remaining = append(remaining, projectile)
		}
	}
	w.projectiles = remaining
}

func isInsideGroundArea(casterPos, groundPoint, aim Vec2, targetPos Vec2, targetRadius float64, skill combat.Skill) bool {
	switch skill.Shape {
	case combat.ShapeCone:
		dir := Vec2{X: groundPoint.X - casterPos.X, Y: groundPoint.Y - casterPos.Y}
		if math.Hypot(dir.X, dir.Y) <= 0.01 {
			dir = aim
		}
		dirLen := math.Hypot(dir.X, dir.Y)
		if dirLen <= 0.01 {
			dir = Vec2{X: 1, Y: 0}
			dirLen = 1
		}
		toTarget := Vec2{X: targetPos.X - casterPos.X, Y: targetPos.Y - casterPos.Y}
		targetDist := math.Hypot(toTarget.X, toTarget.Y)
		maxRange := skill.Radius
		if maxRange <= 0 {
			maxRange = skill.Range
		}
		if targetDist > maxRange+targetRadius {
			return false
		}
		if targetDist <= targetRadius {
			return true
		}
		cosAngle := (dir.X*toTarget.X + dir.Y*toTarget.Y) / (dirLen * targetDist)
		halfAngleDeg := skill.AngleDeg / 2
		if halfAngleDeg <= 0 {
			halfAngleDeg = 30
		}
		minCos := math.Cos(halfAngleDeg * math.Pi / 180)
		return cosAngle >= minCos

	case combat.ShapeRectangle, combat.ShapeWall:
		length := skill.LengthUnits
		if length <= 0 {
			length = skill.Range
		}
		width := skill.WidthUnits
		if width <= 0 {
			width = 2.0
		}
		dir := Vec2{X: groundPoint.X - casterPos.X, Y: groundPoint.Y - casterPos.Y}
		if math.Hypot(dir.X, dir.Y) <= 0.01 {
			dir = aim
		}
		dirLen := math.Hypot(dir.X, dir.Y)
		if dirLen <= 0.01 {
			dir = Vec2{X: 1, Y: 0}
			dirLen = 1
		}
		u := Vec2{X: dir.X / dirLen, Y: dir.Y / dirLen}
		v := Vec2{X: targetPos.X - casterPos.X, Y: targetPos.Y - casterPos.Y}
		dotParallel := v.X*u.X + v.Y*u.Y
		distPerp := math.Abs(v.X*u.Y - v.Y*u.X)
		return dotParallel >= -targetRadius && dotParallel <= length+targetRadius && distPerp <= width/2+targetRadius

	case combat.ShapeDonut:
		d := distance(targetPos, groundPoint)
		outer := skill.Radius
		inner := skill.InnerRadius
		if inner >= outer {
			inner = outer * 0.5
		}
		return d <= outer+targetRadius && d >= inner-targetRadius

	case combat.ShapeCross:
		arm := skill.LengthUnits
		if arm <= 0 {
			arm = skill.Radius
		}
		width := skill.WidthUnits
		if width <= 0 {
			width = 1.5
		}
		dx := math.Abs(targetPos.X - groundPoint.X)
		dy := math.Abs(targetPos.Y - groundPoint.Y)
		inHorizontal := dx <= arm+targetRadius && dy <= width/2+targetRadius
		inVertical := dy <= arm+targetRadius && dx <= width/2+targetRadius
		return inHorizontal || inVertical

	default:
		return distance(targetPos, groundPoint) <= skill.Radius+targetRadius
	}
}

// PlayerVitals returns the authoritative HP pool for one zone player. It is a
// read-only lookup for trusted adapters (legacy inventory/item usage).
func (w *World) PlayerVitals(target EntityID) (hp, maxHP float64, ok bool) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	entity := w.entities[target]
	if entity == nil || entity.Kind != EntityPlayer {
		return 0, 0, false
	}
	return entity.Vitals.HP, entity.Vitals.MaxHP, true
}

// QueueHPDelta applies a trusted out-of-band HP change on the zone tick owner.
// Legacy HTTP actions (consumables and support Gu) use it so the authoritative
// MMO entity, HUD and checkpoint all observe the same result. Clients cannot
// reach this path directly. A negative delta is clamped at 0; a positive delta
// is capped at MaxHP.
func (w *World) QueueHPDelta(target EntityID, delta float64) (<-chan error, error) {
	if !finite(delta) || delta == 0 {
		return nil, ErrInvalidInput
	}
	request := lifecycleCommand{apply: func() error {
		entity := w.entities[target]
		if entity == nil || entity.Kind != EntityPlayer || entity.Vitals.HP <= 0 {
			return ErrInvalidEntity
		}
		before := entity.Vitals.HP
		next := before + delta
		if next < 0 {
			next = 0
		}
		if entity.Vitals.MaxHP > 0 && next > entity.Vitals.MaxHP {
			next = entity.Vitals.MaxHP
		}
		if next == before {
			return nil
		}
		entity.Vitals.HP = next
		entity.Revision++
		if next > before {
			w.emitEvent(combat.Event{
				Type: combat.EventHealApplied, SourceID: string(entity.ID), TargetID: string(entity.ID),
				SkillID: "ITEM_USE", Amount: next - before, HP: next, Shield: entity.Vitals.Shield,
			})
		} else {
			w.emitEvent(combat.Event{
				Type: combat.EventDamageApplied, TargetID: string(entity.ID),
				SkillID: "ITEM_USE", Amount: before - next, HP: next, Shield: entity.Vitals.Shield,
			})
		}
		return nil
	}, result: make(chan error, 1)}
	select {
	case w.lifecycleInbox <- request:
		return request.result, nil
	default:
		return nil, ErrLifecycleQueueFull
	}
}

// applyDamageLocked is the single server-side damage path. Shields absorb first,
// then HP; a player's windup is interrupted by incoming damage.
func (w *World) applyDamageLocked(source, target *Entity, amount float64, skillID string) float64 {
	if target == nil || target.Vitals.HP <= 0 || !finite(amount) || amount <= 0 {
		return 0
	}
	sourceID := ""
	if source != nil {
		sourceID = string(source.ID)
	}

	// 1. Check EVASION: target dodges the hit completely
	if w.hasStatus(target, "EVASION") {
		w.emitEvent(combat.Event{
			Type: combat.EventDamageApplied, SourceID: sourceID, TargetID: string(target.ID), SkillID: skillID,
			X: target.Transform.Position.X, Y: target.Transform.Position.Y,
			Amount: 0, HP: target.Vitals.HP, Shield: target.Vitals.Shield, Reason: "evaded",
		})
		return 0
	}

	// 2. Check INVINCIBLE: target takes 0 damage (Bất Hoại)
	if w.hasStatus(target, "INVINCIBLE") {
		w.emitEvent(combat.Event{
			Type: combat.EventDamageApplied, SourceID: sourceID, TargetID: string(target.ID), SkillID: skillID,
			X: target.Transform.Position.X, Y: target.Transform.Position.Y,
			Amount: 0, HP: target.Vitals.HP, Shield: target.Vitals.Shield, Reason: "immune",
		})
		return 0
	}

	// Mirror the legacy order: defense, natural Dao resistance, path resistance,
	// then the minimum-hit floor. The skill path was resolved at admission.
	path := "strength"
	if source != nil && skillID != "BASIC_ATTACK" {
		for _, skill := range source.Combat.Skills {
			if skill.ID == skillID && skill.Path != "" {
				path = skill.Path
				break
			}
		}
	}
	remaining := amount - math.Max(0, target.Defense)
	if target.DaoProfile != nil {
		remaining *= 1 - target.DaoProfile.GetDefensiveResistance(path)
	}
	remaining *= 1 - target.PathResistance[gamefx.CleanPath(path)]
	remaining = math.Max(1, remaining)
	shieldAbsorbed := 0.0
	if target.Vitals.Shield > 0 {
		absorbed := math.Min(target.Vitals.Shield, remaining)
		target.Vitals.Shield -= absorbed
		remaining -= absorbed
		shieldAbsorbed = absorbed
	}
	hpBefore := target.Vitals.HP
	damageReason := ""
	if remaining > 0 {
		// 3. Check LAST_STAND: HP cannot fall below 1 (Bất Tử)
		if w.hasStatus(target, "LAST_STAND") {
			newHP := math.Max(1.0, target.Vitals.HP-remaining)
			if newHP == 1.0 && (target.Vitals.HP-remaining) <= 0 {
				damageReason = "last_stand"
			}
			target.Vitals.HP = newHP
		} else {
			target.Vitals.HP = math.Max(0, target.Vitals.HP-remaining)
		}
	}
	hpDamage := hpBefore - target.Vitals.HP
	target.Revision++
	if hpDamage > 0 || shieldAbsorbed > 0 {
		w.removeStatusLocked(target, "STEALTH")
	}
	if target.Kind == EntityPlayer && target.Combat.Pending != nil {
		interrupted := target.Combat.Pending.Skill.ID
		target.Combat.Pending = nil
		interruptedBy := ""
		if source != nil {
			interruptedBy = string(source.ID)
		}
		w.emitEvent(combat.Event{Type: combat.EventCastInterrupted, SourceID: interruptedBy, TargetID: string(target.ID), SkillID: interrupted, Reason: "damaged"})
	}
	w.emitEvent(combat.Event{
		Type: combat.EventDamageApplied, SourceID: sourceID, TargetID: string(target.ID), SkillID: skillID,
		X: target.Transform.Position.X, Y: target.Transform.Position.Y,
		Amount: hpDamage, HP: target.Vitals.HP, Shield: target.Vitals.Shield, Reason: damageReason,
	})
	if source != nil && target.Kind == EntityMonster {
		w.recordThreatLocked(target, source, shieldAbsorbed+hpDamage)
	}
	if w.hasStatus(target, "THORNS") && source != nil && source.ID != target.ID && skillID != "THORNS" && hpDamage > 0 {
		thornsDmg := math.Max(1, hpDamage*0.35)
		w.applyDamageLocked(target, source, thornsDmg, "THORNS")
	}
	if target.Vitals.HP > 0 {
		w.tryHiddenPassiveLocked(target)
	}
	if target.Vitals.HP > 0 {
		return hpDamage
	}
	target.moveIntent = Vec2{}
	target.moveTarget = nil
	target.Combat.Pending = nil
	target.AI.Threat = ""
	target.AI.PatrolTarget = nil
	w.emitEvent(combat.Event{Type: combat.EventEntityDied, SourceID: sourceID, TargetID: string(target.ID), SkillID: skillID, X: target.Transform.Position.X, Y: target.Transform.Position.Y, TemplateID: target.AI.TemplateID, PersistentID: target.AI.PersistentID})
	if target.Kind == EntityMonster && target.AI.Active {
		target.AI.State = AIDead
		if !target.AI.NoRespawn {
			respawnTicks := target.AI.RespawnTicks
			if respawnTicks == 0 {
				respawnTicks = 20 * 60
			}
			target.AI.RespawnTick = w.tick + uint64(respawnTicks)
		}
	}
	return hpDamage
}

func (w *World) tryHiddenPassiveLocked(target *Entity) {
	passive := &target.Combat.Hidden
	if target.Kind != EntityPlayer || passive.ID == "" || passive.RequiredHits <= 0 {
		return
	}
	if target.Vitals.MaxHP <= 0 || target.Vitals.HP/target.Vitals.MaxHP > passive.HPPercentLTE {
		passive.HitCount = 0
		return
	}
	if w.tick < passive.NextReadyTick {
		return
	}
	passive.HitCount++
	if passive.HitCount < passive.RequiredHits {
		return
	}
	passive.HitCount = 0
	passive.NextReadyTick = w.tick + passive.ICDTicks
	passive.Discovered = true
	passive.TriggeredOnce = true
	amount := 0.0
	if passive.Shield > 0 {
		target.Vitals.Shield += passive.Shield
		amount += passive.Shield
	}
	if passive.Heal > 0 {
		before := target.Vitals.HP
		target.Vitals.HP = math.Min(target.Vitals.MaxHP, target.Vitals.HP+passive.Heal)
		amount += target.Vitals.HP - before
	}
	target.Revision++
	w.emitEvent(combat.Event{
		Type: combat.EventHiddenPassive, SourceID: string(target.ID), TargetID: string(target.ID),
		SkillID: passive.ID, Amount: amount, HP: target.Vitals.HP, Shield: target.Vitals.Shield,
		Reason: passive.ID, X: target.Transform.Position.X, Y: target.Transform.Position.Y,
	})
}

func (w *World) emitEvent(event combat.Event) {
	w.nextEventID++
	event.ID = w.nextEventID
	event.Tick = w.tick
	w.events = append(w.events, event)
}

func commandSkillID(slot uint8) string {
	return "HOTBAR_" + strconv.Itoa(int(slot))
}

func distance(left, right Vec2) float64 {
	return math.Hypot(left.X-right.X, left.Y-right.Y)
}

// segmentDistance returns the shortest distance from a point to a segment.
func segmentDistance(from, to, point Vec2) float64 {
	dx, dy := to.X-from.X, to.Y-from.Y
	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		return math.Hypot(point.X-from.X, point.Y-from.Y)
	}
	t := ((point.X-from.X)*dx + (point.Y-from.Y)*dy) / lengthSquared
	t = math.Max(0, math.Min(1, t))
	closestX, closestY := from.X+t*dx, from.Y+t*dy
	return math.Hypot(point.X-closestX, point.Y-closestY)
}

func (w *World) applyKnockbackLocked(source, target *Entity, dist float64) {
	if source == nil || target == nil || target.Vitals.HP <= 0 || dist <= 0 {
		return
	}
	dx := target.Transform.Position.X - source.Transform.Position.X
	dy := target.Transform.Position.Y - source.Transform.Position.Y
	l := math.Hypot(dx, dy)
	if l <= 0.01 {
		dx = 1
		dy = 0
		l = 1
	}
	dest := Vec2{
		X: target.Transform.Position.X + (dx/l)*dist,
		Y: target.Transform.Position.Y + (dy/l)*dist,
	}
	if !w.blocked(dest, target.Radius) {
		target.Transform.Position = dest
		target.Revision++
	}
}
