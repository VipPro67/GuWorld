package sim

import (
	"math"
	"strconv"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// Realtime combat core: cast/basic-attack intents in, deterministic zone
// impact out. This file keeps the state types, interaction entry points and
// the windup/cooldown gating for player casts. The resolution rules live in
// skill_resolution.go, the damage pipeline in combat_damage.go and the pure
// AoE geometry in combat_geometry.go.

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
	Aim        Vec2
}

type CaptureCommand struct {
	EntityID   EntityID
	LeaseToken string
	Seq        uint64
	TargetID   EntityID
	GuID       string
}

type CompanionStanceCommand struct {
	EntityID      EntityID
	LeaseToken    string
	Seq           uint64
	CompanionID   EntityID
	Stance        CompanionStance
	FocusTargetID EntityID
}

type PendingCast struct {
	Skill       combat.Skill
	TargetID    EntityID
	Aim         Vec2
	Point       Vec2
	ResolveTick uint64
	Seq         uint64
	IsCapture   bool
	CaptureGuID string
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
	Cooldowns                [13]uint64
	Skills                   [13]combat.Skill
	GuHunger                 [13]float64
	GuUses                   [13]uint64
	GuMasteryUses            [13]uint64
	KillerMoveUses           [13]uint64
	BasicAttackUses          uint64
	FormActive               bool
	FormManaDrainPerTick     float64
	FormDefenseBonus         float64
	Pending                  *PendingCast
	Statuses                 []StatusEffect
	Hidden                   combat.HiddenPassive
}

func (w *World) SubmitBasicAttack(command AttackCommand) error {
	if command.EntityID == "" || command.Seq == 0 || len(command.TargetID) > MaxEntityIDLength || !finite(command.Aim.X) || !finite(command.Aim.Y) || (command.TargetID == "" && math.Hypot(command.Aim.X, command.Aim.Y) <= 0.01) {
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
	if w.tick < attacker.Combat.BasicAttackCooldownUntil {
		reject("cooldown")
		return
	}
	var target *Entity
	if command.TargetID != "" {
		target = w.entities[command.TargetID]
		if !w.canHarm(attacker, target) {
			reject("invalid_target")
			return
		}
		if distance(attacker.Transform.Position, target.Transform.Position) > combat.BasicAttackRange {
			reject("out_of_range")
			return
		}
	} else {
		target = w.firstCollisionAlongAim(attacker, command.Aim, combat.BasicAttackRange, combat.AimedHitPadding, func(target *Entity) bool {
			return w.canHarm(attacker, target)
		})
	}
	attacker.Combat.BasicAttackCooldownUntil = w.tick + combat.BasicAttackCooldownTicks
	attacker.Revision++
	w.removeStatusLocked(attacker, "STEALTH")
	w.emitEvent(combat.Event{Type: combat.EventBasicAttack, SourceID: string(attacker.ID), TargetID: string(command.TargetID), SkillID: "BASIC_ATTACK", DurationTicks: 2})
	if w.hasStatus(attacker, "BLIND") {
		if target != nil {
			w.emitEvent(combat.Event{Type: combat.EventDamageApplied, SourceID: string(attacker.ID), TargetID: string(target.ID), SkillID: "BASIC_ATTACK", X: target.Transform.Position.X, Y: target.Transform.Position.Y, Amount: 0, HP: target.Vitals.HP, Shield: target.Vitals.Shield, Reason: "blind"})
		}
		return
	}
	if target == nil {
		return
	}
	attacker.Combat.BasicAttackUses++
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
		if skill.Slot >= 1 && skill.Slot <= 12 && skill.ID != "" {
			state.Skills[skill.Slot] = skill
			if skill.Slot <= 6 || len(skill.GuComponentIDs) > 0 {
				state.GuHunger[skill.Slot] = skill.GuHunger
			}
		}
	}
	return state
}

type projectile struct {
	ID              uint64
	SourceID        EntityID
	SkillID         string
	Skill           combat.Skill
	RangeOrigin     Vec2
	Position        Vec2
	Velocity        Vec2
	Radius          float64
	Damage          float64
	Travelled       float64
	MaxTravel       float64
	ExpireTick      uint64
	PierceRemaining int
	HitEntities     map[EntityID]bool
}

// ProjectileState is the render-safe projection of a server projectile.
type ProjectileState struct {
	ID            uint64  `json:"id"`
	SourceID      string  `json:"source_id"`
	SkillID       string  `json:"skill_id"`
	ProjectileVFX string  `json:"projectile_vfx,omitempty"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Radius        float64 `json:"radius"`
}

func (w *World) SubmitCast(command CastCommand) error {
	if command.EntityID == "" || command.Seq == 0 || command.Slot < 1 || command.Slot > 12 {
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
	if (caster.Kind != EntityPlayer && caster.Kind != EntityCompanion) || caster.Vitals.HP <= 0 {
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
	if skill.MoveDistance > 0 && caster.dash != nil {
		reject("already_dashing")
		return
	}
	if command.Mode != skill.Mode {
		reject("wrong_target_mode")
		return
	}
	if skill.FormActivation && caster.Combat.FormActive {
		caster.Combat.FormActive = false
		caster.Combat.FormManaDrainPerTick = 0
		caster.Combat.FormDefenseBonus = 0
		caster.Revision++
		return
	}
	if skill.FormAbility && !caster.Combat.FormActive {
		reject("form_inactive")
		return
	}
	if skill.IsToggle && skill.StatusID != "" && w.hasStatus(caster, skill.StatusID) {
		w.removeStatusLocked(caster, skill.StatusID)
		caster.Revision++
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
		if command.TargetID != "" {
			target := w.entities[command.TargetID]
			if target == nil || (skill.MoveDistance == 0 && !w.canApplySkill(caster, target, skill)) {
				reject("invalid_target")
				return
			}
			if distance(caster.Transform.Position, target.Transform.Position) > skill.Range {
				reject("out_of_range")
				return
			}
		} else if skill.MoveDistance == 0 && math.Hypot(command.Aim.X, command.Aim.Y) <= 0.01 {
			reject("invalid_aim")
			return
		} else {
			pending.Point = Vec2{X: caster.Transform.Position.X + command.Aim.X, Y: caster.Transform.Position.Y + command.Aim.Y}
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
		if skill.MoveDistance == 0 && math.Hypot(command.Aim.X, command.Aim.Y) <= 0.01 {
			reject("invalid_aim")
			return
		}
	}
	if skill.MoveDistance > 0 {
		aim := command.Aim
		if math.Hypot(aim.X, aim.Y) <= 0.01 && command.TargetID != "" {
			if target := w.entities[command.TargetID]; target != nil {
				aim = Vec2{X: target.Transform.Position.X - caster.Transform.Position.X, Y: target.Transform.Position.Y - caster.Transform.Position.Y}
			}
		}
		if math.Hypot(aim.X, aim.Y) <= 0.01 && skill.Mode == combat.ModeGround {
			aim = Vec2{X: command.Point.X - caster.Transform.Position.X, Y: command.Point.Y - caster.Transform.Position.Y}
		}
		if math.Hypot(aim.X, aim.Y) <= 0.01 {
			aim = caster.moveIntent
		}
		if math.Hypot(aim.X, aim.Y) <= 0.01 {
			aim = caster.lastDirection
		}
		length := math.Hypot(aim.X, aim.Y)
		if length <= 0.01 {
			reject("invalid_aim")
			return
		}
		pending.Aim = Vec2{X: aim.X / length, Y: aim.Y / length}
	}
	if (skill.GuInstanceID != "" || len(skill.GuComponentIDs) > 0) && caster.Combat.GuHunger[skill.Slot] < combat.DefaultGuHungerCostPerCast {
		reject("gu_exhausted")
		return
	}
	// Gu and composite killer move casts consume DefaultGuHungerCostPerCast (0.1) hunger from every required
	// Gu on an attempt, before backlash and the mana check. The zone keeps the
	// balances until the next atomic player checkpoint settles usage by instance.
	if skill.GuInstanceID != "" || len(skill.GuComponentIDs) > 0 {
		caster.Combat.GuHunger[skill.Slot] = math.Max(0, caster.Combat.GuHunger[skill.Slot]-combat.DefaultGuHungerCostPerCast)
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
	// Chân Ý only counts accepted regular Gu casts; hunger above charges every
	// attempt, including casts rejected for insufficient mana.
	if skill.GuInstanceID != "" && skill.KillerMoveID == "" {
		caster.Combat.GuMasteryUses[skill.Slot]++
	}
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
		SkillID: skill.ID, SkillName: skill.Name, X: pending.Point.X, Y: pending.Point.Y, DurationTicks: skill.WindupTicks,
		CastVFX: skill.Visuals.CastVFX, ProjectileVFX: skill.Visuals.ProjectileVFX, ImpactVFX: skill.Visuals.ImpactVFX,
		DashDirX: pending.Aim.X, DashDirY: pending.Aim.Y, DashDurationTicks: skill.MoveDurationTicks,
	})
}

func (w *World) tickCombat() {
	w.decayShieldsLocked()
	for _, entity := range w.entities {
		if entity.Kind != EntityPlayer || !entity.Combat.FormActive || entity.Vitals.HP <= 0 {
			continue
		}
		drain := entity.Combat.FormManaDrainPerTick
		if drain <= 0 || entity.Combat.Mana < drain {
			entity.Combat.FormActive = false
			entity.Combat.FormManaDrainPerTick = 0
			entity.Combat.FormDefenseBonus = 0
			entity.Revision++
			continue
		}
		entity.Combat.Mana -= drain
		if entity.Combat.Mana <= 0 {
			entity.Combat.FormActive = false
			entity.Combat.FormManaDrainPerTick = 0
			entity.Combat.FormDefenseBonus = 0
		}
		entity.Revision++
	}
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

// Damage resolution lives in combat_damage.go, skill resolution in
// skill_resolution.go, and shared geometry in combat_geometry.go.

// Damage resolution lives in combat_damage.go, skill resolution in
// skill_resolution.go, and shared geometry in combat_geometry.go.

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

// PlayerPosition returns the live position for one zone player entity.
func (w *World) PlayerPosition(target EntityID) (Vec2, bool) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	entity := w.entities[target]
	if entity == nil || entity.Kind != EntityPlayer {
		return Vec2{}, false
	}
	return entity.Transform.Position, true
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

func (w *World) emitEvent(event combat.Event) {
	w.nextEventID++
	event.ID = w.nextEventID
	event.Tick = w.tick
	w.events = append(w.events, event)
}

func commandSkillID(slot uint8) string {
	return "HOTBAR_" + strconv.Itoa(int(slot))
}

func (w *World) SubmitCapture(command CaptureCommand) error {
	if command.EntityID == "" || command.Seq == 0 || command.TargetID == "" || len(command.TargetID) > MaxEntityIDLength {
		return ErrInvalidInput
	}
	cmd := command
	select {
	case w.inbox <- InputCommand{EntityID: command.EntityID, LeaseToken: command.LeaseToken, Seq: command.Seq, Capture: &cmd}:
		return nil
	default:
		return ErrQueueFull
	}
}

func (w *World) SubmitCompanionStance(command CompanionStanceCommand) error {
	if command.EntityID == "" || command.Seq == 0 || (command.Stance != StanceDefend && command.Stance != StanceFocus && command.Stance != StanceFollow) {
		return ErrInvalidInput
	}
	cmd := command
	select {
	case w.inbox <- InputCommand{EntityID: command.EntityID, LeaseToken: command.LeaseToken, Seq: command.Seq, CompanionStance: &cmd}:
		return nil
	default:
		return ErrQueueFull
	}
}

func (w *World) startCaptureLocked(caster *Entity, command CaptureCommand) {
	if caster == nil {
		return
	}
	reject := func(reason string) {
		w.emitEvent(combat.Event{
			Type: combat.EventCastRejected, SourceID: string(caster.ID),
			Reason: reason,
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
	if w.hasStatus(caster, "SILENCE") || w.hasStatus(caster, "GU_SEAL") {
		reject("silenced")
		return
	}

	target := w.entities[command.TargetID]
	if target == nil || target.Kind != EntityMonster || target.Vitals.HP <= 0 {
		reject("target_not_ready")
		return
	}
	if !target.AI.Active {
		reject("target_not_ready")
		return
	}

	if distance(caster.Transform.Position, target.Transform.Position) > 5.0 {
		reject("out_of_range")
		return
	}

	hpPct := target.Vitals.HP / target.Vitals.MaxHP
	hasCC := w.isStunned(target) || w.hasStatus(target, "BIND") || w.hasStatus(target, "ROOT") || w.hasStatus(target, "FREEZE") || w.hasStatus(target, "PETRIFY")
	if hpPct > 0.25 && !hasCC {
		reject("monster_not_subdued")
		return
	}

	captureManaCost := 20.0
	if caster.Combat.Mana < captureManaCost {
		reject("insufficient_mana")
		return
	}
	caster.Combat.Mana -= captureManaCost

	windupTicks := uint64(60) // 3.0s
	caster.Combat.Pending = &PendingCast{
		Skill: combat.Skill{
			ID:             "SKILL_ENSLAVE_CHANNELING",
			Name:           "Nô Thú Kết Ấn",
			MovementPolicy: "ROOT",
		},
		TargetID:    target.ID,
		ResolveTick: w.tick + windupTicks,
		Seq:         command.Seq,
		IsCapture:   true,
		CaptureGuID: command.GuID,
	}
	caster.Revision++

	w.emitEvent(combat.Event{
		Type:          combat.EventCaptureStarted,
		SourceID:      string(caster.ID),
		TargetID:      string(target.ID),
		DurationTicks: windupTicks,
	})
}

func (w *World) setCompanionStanceLocked(owner *Entity, command CompanionStanceCommand) {
	if owner == nil || owner.Kind != EntityPlayer {
		return
	}
	for _, entity := range w.entities {
		if entity.Kind == EntityCompanion && entity.Companion.OwnerID == owner.ID {
			if command.CompanionID == "" || entity.ID == command.CompanionID {
				entity.Companion.Stance = command.Stance
				if command.Stance == StanceFocus {
					entity.Companion.FocusTargetID = command.FocusTargetID
				} else {
					entity.Companion.FocusTargetID = ""
				}
				entity.Revision++
			}
		}
	}
}
