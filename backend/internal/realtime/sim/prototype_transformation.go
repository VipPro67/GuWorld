package sim

import (
	"math"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// TransformationKind classifies the degree of bodily metamorphosis.
type TransformationKind string

const (
	TransformationNone    TransformationKind = ""
	TransformationPartial TransformationKind = "PARTIAL"  // Biến thân bộ phận (vuốt, da, cánh, vảy)
	TransformationFull    TransformationKind = "FULL"     // Biến thân toàn phần (thú, ma thần)
	TransformationGuHouse TransformationKind = "GU_HOUSE" // Dung nhập Tiên Cổ Phòng
)

// UpkeepCost defines resources drained per 20Hz tick (50ms).
type UpkeepCost struct {
	ManaPerTick    float64 `json:"mana_per_tick"`
	ThoughtPerTick float64 `json:"thought_per_tick"`
}

// TransformationState defines the active metamorphosis profile on an Entity in the 20Hz zone sim.
type TransformationState struct {
	Active              bool               `json:"active"`
	FormID              string             `json:"form_id"`
	FormName            string             `json:"form_name"`
	FormKind            TransformationKind `json:"form_kind"`
	ModelScale          float64            `json:"model_scale"`
	CollisionRadiusMult float64            `json:"collision_radius_mult"`
	OriginalRadius      float64            `json:"original_radius"`

	// Form HP Layer & Armor
	MaxFormHP        float64 `json:"max_form_hp"`
	CurrentFormHP    float64 `json:"current_form_hp"`
	FormDefenseBonus float64 `json:"form_defense_bonus"`
	DamageReduction  float64 `json:"damage_reduction"` // 0.0 to 1.0 (e.g. 0.25 = 25% damage reduction)

	// Upkeep drain per tick (50ms)
	Upkeep     UpkeepCost `json:"upkeep"`
	ExpireTick uint64     `json:"expire_tick"` // 0 if sustained indefinitely until mana depletion

	// Channeling / Windup stage
	IsWindup      bool   `json:"is_windup"`
	WindupEndTick uint64 `json:"windup_end_tick"`
}

// FormActivationRequest carries authoring data to initiate metamorphosis.
type FormActivationRequest struct {
	FormID              string
	FormName            string
	FormKind            TransformationKind
	ModelScale          float64
	CollisionRadiusMult float64
	MaxFormHP           float64
	FormDefenseBonus    float64
	DamageReduction     float64
	ManaUpkeepPerSecond float64
	DurationSeconds     float64
	WindupTicks         uint64
	InitialManaCost     float64
}

// StartTransformation begins casting or immediately enters transformation.
func (w *World) StartTransformation(entity *Entity, req FormActivationRequest) bool {
	if entity == nil || entity.Vitals.HP <= 0 {
		return false
	}
	if entity.Combat.Mana < req.InitialManaCost {
		return false
	}

	// Consume initial mana
	entity.Combat.Mana -= req.InitialManaCost

	// If windup is required, enter windup stage
	if req.WindupTicks > 0 {
		entity.Transformation = TransformationState{
			Active:        false,
			FormID:        req.FormID,
			FormName:      req.FormName,
			FormKind:      req.FormKind,
			IsWindup:      true,
			WindupEndTick: w.tick + req.WindupTicks,
		}
		entity.Revision++
		w.emitEvent(combat.Event{
			Type:     combat.EventCastAccepted,
			SourceID: string(entity.ID),
			SkillID:  req.FormID,
			Reason:   "transform_windup",
		})
		return true
	}

	// Instant transformation
	w.completeTransformation(entity, req)
	return true
}

func (w *World) completeTransformation(entity *Entity, req FormActivationRequest) {
	if entity.Transformation.OriginalRadius == 0 {
		entity.Transformation.OriginalRadius = entity.Radius
	}

	radiusMult := req.CollisionRadiusMult
	if radiusMult <= 0 {
		radiusMult = 1.0
	}
	modelScale := req.ModelScale
	if modelScale <= 0 {
		modelScale = 1.0
	}

	expireTick := uint64(0)
	if req.DurationSeconds > 0 {
		expireTick = w.tick + uint64(req.DurationSeconds*float64(TickRate))
	}

	entity.Transformation = TransformationState{
		Active:              true,
		FormID:              req.FormID,
		FormName:            req.FormName,
		FormKind:            req.FormKind,
		ModelScale:          modelScale,
		CollisionRadiusMult: radiusMult,
		OriginalRadius:      entity.Transformation.OriginalRadius,
		MaxFormHP:           req.MaxFormHP,
		CurrentFormHP:       req.MaxFormHP,
		FormDefenseBonus:    req.FormDefenseBonus,
		DamageReduction:     req.DamageReduction,
		Upkeep: UpkeepCost{
			ManaPerTick: req.ManaUpkeepPerSecond / float64(TickRate),
		},
		ExpireTick: expireTick,
		IsWindup:   false,
	}

	// Scale collision radius
	entity.Radius = entity.Transformation.OriginalRadius * radiusMult
	entity.Revision++

	w.emitEvent(combat.Event{
		Type:     combat.EventStatusApplied,
		SourceID: string(entity.ID),
		TargetID: string(entity.ID),
		SkillID:  req.FormID,
		Reason:   "metamorphosis_complete",
	})
}

// TickTransformation is called every 50ms (20Hz) in the world step loop.
func (w *World) TickTransformation() {
	for _, entity := range w.entities {
		if entity == nil || entity.Vitals.HP <= 0 {
			continue
		}

		// 1. Check pending windup completion
		if entity.Transformation.IsWindup {
			if w.tick >= entity.Transformation.WindupEndTick {
				w.completeTransformation(entity, FormActivationRequest{
					FormID:              entity.Transformation.FormID,
					FormName:            entity.Transformation.FormName,
					FormKind:            entity.Transformation.FormKind,
					ModelScale:          1.5,
					CollisionRadiusMult: 1.4,
					MaxFormHP:           150,
					FormDefenseBonus:    20,
					DamageReduction:     0.15,
					ManaUpkeepPerSecond: 10,
				})
			}
			continue
		}

		if !entity.Transformation.Active {
			continue
		}

		// 2. Check time expiration
		if entity.Transformation.ExpireTick > 0 && w.tick >= entity.Transformation.ExpireTick {
			w.RevertTransformation(entity, "duration_expired", false)
			continue
		}

		// 3. Upkeep drain (Mana / Thought)
		upkeep := entity.Transformation.Upkeep
		if upkeep.ManaPerTick > 0 {
			if entity.Combat.Mana < upkeep.ManaPerTick {
				// Out of mana -> Force cancel with Exhaustion debuff
				w.RevertTransformation(entity, "mana_depleted", true)
				continue
			}
			entity.Combat.Mana -= upkeep.ManaPerTick
		}
		if upkeep.ThoughtPerTick > 0 {
			if entity.Combat.Thought < upkeep.ThoughtPerTick {
				w.RevertTransformation(entity, "thought_depleted", true)
				continue
			}
			entity.Combat.Thought -= upkeep.ThoughtPerTick
		}

		entity.Revision++
	}
}

// InterruptTransformation checks if an active windup should be cancelled by Hard CC.
func (w *World) InterruptTransformation(entity *Entity, ccType string) bool {
	if entity == nil || !entity.Transformation.IsWindup {
		return false
	}

	entity.Transformation.IsWindup = false
	entity.Transformation.Active = false
	entity.Revision++

	w.emitEvent(combat.Event{
		Type:     combat.EventCastInterrupted,
		SourceID: string(entity.ID),
		SkillID:  entity.Transformation.FormID,
		Reason:   "cc_interrupted_" + ccType,
	})
	return true
}

// ApplyFormDamage routes incoming damage to Form HP layer before actual HP.
// Returns remaining damage that spills over to regular HP.
func (w *World) ApplyFormDamage(target *Entity, incomingDamage float64) float64 {
	if target == nil || !target.Transformation.Active {
		return incomingDamage
	}

	// Apply Form Damage Reduction
	reducedDamage := incomingDamage
	if target.Transformation.DamageReduction > 0 {
		reducedDamage *= (1.0 - math.Min(0.9, target.Transformation.DamageReduction))
	}

	// Deduct from Form HP layer if available
	if target.Transformation.CurrentFormHP > 0 {
		if target.Transformation.CurrentFormHP >= reducedDamage {
			target.Transformation.CurrentFormHP -= reducedDamage
			target.Revision++
			return 0 // Absorbed completely
		}

		// Form HP Broken
		spillover := reducedDamage - target.Transformation.CurrentFormHP
		target.Transformation.CurrentFormHP = 0
		w.RevertTransformation(target, "form_broken", true)
		return spillover
	}

	return reducedDamage
}

// RevertTransformation restores the entity back to standard form.
func (w *World) RevertTransformation(entity *Entity, reason string, applyExhaustion bool) {
	if entity == nil || (!entity.Transformation.Active && !entity.Transformation.IsWindup) {
		return
	}

	// Restore original hitbox radius
	if entity.Transformation.OriginalRadius > 0 {
		entity.Radius = entity.Transformation.OriginalRadius
	}

	formID := entity.Transformation.FormID
	entity.Transformation = TransformationState{
		Active:         false,
		OriginalRadius: entity.Transformation.OriginalRadius,
	}
	entity.Revision++

	w.emitEvent(combat.Event{
		Type:     combat.EventStatusExpired,
		SourceID: string(entity.ID),
		TargetID: string(entity.ID),
		SkillID:  formID,
		Reason:   reason,
	})

	// If broken or exhausted, apply temporary exhaustion debuff (reduce speed & defense)
	if applyExhaustion {
		w.applyStatusLocked(nil, entity, combat.Skill{
			StatusID:             "EXHAUSTION",
			StatusDurationTicks:  100, // 5s at 20Hz
			StatusMoveMultiplier: 0.7, // -30% movement speed
			StatusDebuff:         true,
		})
	}
}
