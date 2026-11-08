package sim

import (
	"encoding/json"
	"math"

	"github.com/vippro67/GuWorld/backend/internal/models"
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

// StoredSkillState captures human skills and cooldowns before metamorphosis.
type StoredSkillState struct {
	Skills    [13]combat.Skill
	Cooldowns [13]uint64
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

	// Stored human skills for restoring upon revert
	SavedSkills StoredSkillState `json:"-"`
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

// FormSkillProfile defines the complete replacement action bar for a Full Metamorphosis form.
type FormSkillProfile struct {
	FormID      string
	FormName    string
	Skills      [7]combat.Skill // Slots 1..6 (Slot 6 is typically Revert Form)
	BasicAttack combat.Skill
}

// SwapSkillPalette replaces the entity's active skill bar (Slots 1..6) with the Form's skills.
func SwapSkillPalette(entity *Entity, currentTick uint64, profile FormSkillProfile) StoredSkillState {
	backup := StoredSkillState{
		Skills:    entity.Combat.Skills,
		Cooldowns: entity.Combat.Cooldowns,
	}

	// Apply replacement skills to slots 1..6
	for slot := 1; slot <= 6; slot++ {
		formSkill := profile.Skills[slot]
		if formSkill.ID != "" {
			formSkill.Slot = slot
			entity.Combat.Skills[slot] = formSkill
			entity.Combat.Cooldowns[slot] = 0 // Ready immediately upon morph
		} else {
			entity.Combat.Skills[slot] = combat.Skill{}
		}
	}

	entity.Revision++
	return backup
}

// RestoreSkillPalette restores the entity's human skill bar upon reverting.
func RestoreSkillPalette(entity *Entity, currentTick uint64, backup StoredSkillState) {
	for slot := 1; slot <= 6; slot++ {
		entity.Combat.Skills[slot] = backup.Skills[slot]
		// Retain remaining cooldowns or clear if elapsed
		cd := backup.Cooldowns[slot]
		if cd > currentTick {
			entity.Combat.Cooldowns[slot] = cd
		} else {
			entity.Combat.Cooldowns[slot] = 0
		}
	}
	entity.Revision++
}

// StartTransformation begins casting or immediately enters transformation.
func (w *World) StartTransformation(entity *Entity, req FormActivationRequest) bool {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.startTransformationLocked(entity, req)
}

func (w *World) startTransformationLocked(entity *Entity, req FormActivationRequest) bool {
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
	w.completeTransformationLocked(entity, req)
	return true
}

func (w *World) completeTransformationLocked(entity *Entity, req FormActivationRequest) {
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

	// If Full Metamorphosis, swap skill palette
	var savedSkills StoredSkillState
	if req.FormKind == TransformationFull {
		profile, found := GetFormSkillProfile(req.FormID)
		if found {
			savedSkills = SwapSkillPalette(entity, w.tick, profile)
		}
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
		ExpireTick:  expireTick,
		IsWindup:    false,
		SavedSkills: savedSkills,
	}

	// Also maintain compatibility with legacy Combat.FormActive
	entity.Combat.FormActive = true
	entity.Combat.FormManaDrainPerTick = req.ManaUpkeepPerSecond / float64(TickRate)
	entity.Combat.FormDefenseBonus = req.FormDefenseBonus

	// Scale collision radius
	entity.Radius = entity.Transformation.OriginalRadius * radiusMult
	entity.Revision++

	w.emitEvent(combat.Event{
		Type:     combat.EventTransformationStarted,
		SourceID: string(entity.ID),
		TargetID: string(entity.ID),
		SkillID:  req.FormID,
		Reason:   "metamorphosis_complete",
	})
}

// TickTransformation is called every 50ms (20Hz) in the world step loop.
func (w *World) TickTransformation() {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	w.tickTransformationLocked()
}

func (w *World) tickTransformationLocked() {
	for _, entity := range w.entities {
		if entity == nil || entity.Vitals.HP <= 0 {
			continue
		}

		// 1. Check pending windup completion
		if entity.Transformation.IsWindup {
			if w.tick >= entity.Transformation.WindupEndTick {
				if entity.Transformation.FormKind == TransformationGuHouse {
					w.completeGuHouseTransformationLocked(entity)
					continue
				}
				// Lookup authored form config or default
				rec, found := GetTransformationForm(entity.Transformation.FormID)
				req := FormActivationRequest{
					FormID:              entity.Transformation.FormID,
					FormName:            entity.Transformation.FormName,
					FormKind:            entity.Transformation.FormKind,
					ModelScale:          1.5,
					CollisionRadiusMult: 1.4,
					MaxFormHP:           150,
					FormDefenseBonus:    20,
					DamageReduction:     0.15,
					ManaUpkeepPerSecond: 10,
				}
				if found {
					req.FormName = rec.Name
					req.ModelScale = rec.ModelScale
					req.CollisionRadiusMult = rec.CollisionRadiusMult
					req.MaxFormHP = rec.MaxFormHP
					req.FormDefenseBonus = rec.FormDefense
					req.DamageReduction = rec.DamageReduction
					req.ManaUpkeepPerSecond = rec.ManaUpkeepSec
				}
				w.completeTransformationLocked(entity, req)
			}
			continue
		}

		if !entity.Transformation.Active {
			continue
		}

		// 2. Check time expiration
		if entity.Transformation.ExpireTick > 0 && w.tick >= entity.Transformation.ExpireTick {
			w.revertTransformationLocked(entity, "duration_expired", false)
			continue
		}

		// 3. Upkeep drain (Mana / Thought)
		upkeep := entity.Transformation.Upkeep
		if upkeep.ManaPerTick > 0 {
			if entity.Combat.Mana < upkeep.ManaPerTick {
				// Out of mana -> Force cancel with Exhaustion debuff
				w.revertTransformationLocked(entity, "mana_depleted", true)
				continue
			}
			entity.Combat.Mana -= upkeep.ManaPerTick
		}
		if upkeep.ThoughtPerTick > 0 {
			if entity.Combat.Thought < upkeep.ThoughtPerTick {
				w.revertTransformationLocked(entity, "thought_depleted", true)
				continue
			}
			entity.Combat.Thought -= upkeep.ThoughtPerTick
		}

		entity.Revision++
	}
}

// InterruptTransformation checks if an active windup should be cancelled by Hard CC.
func (w *World) InterruptTransformation(entity *Entity, ccType string) bool {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.interruptTransformationLocked(entity, ccType)
}

func (w *World) interruptTransformationLocked(entity *Entity, ccType string) bool {
	if entity == nil || !entity.Transformation.IsWindup {
		return false
	}

	formID := entity.Transformation.FormID
	entity.Transformation.IsWindup = false
	entity.Transformation.Active = false
	entity.Revision++

	w.emitEvent(combat.Event{
		Type:     combat.EventCastInterrupted,
		SourceID: string(entity.ID),
		SkillID:  formID,
		Reason:   "cc_interrupted_" + ccType,
	})
	return true
}

// ApplyFormDamage routes incoming damage to Form HP layer before actual HP.
// Returns remaining damage that spills over to regular HP.
func (w *World) ApplyFormDamage(target *Entity, incomingDamage float64) float64 {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.applyFormDamageLocked(target, incomingDamage)
}

func (w *World) applyFormDamageLocked(target *Entity, incomingDamage float64) float64 {
	if target == nil || !target.Transformation.Active {
		return incomingDamage
	}

	// Apply Form Damage Reduction
	reducedDamage := incomingDamage
	if target.Transformation.DamageReduction > 0 {
		reducedDamage *= (1.0 - math.Min(0.9, target.Transformation.DamageReduction))
	}

	// For Gu House: 100% absorption without spillover
	if target.Transformation.FormKind == TransformationGuHouse {
		if target.Transformation.CurrentFormHP > 0 {
			if target.Transformation.CurrentFormHP > reducedDamage {
				target.Transformation.CurrentFormHP -= reducedDamage
				target.Revision++
				return 0 // Absorbed completely
			}
			// Structural Breach: Form HP drops to 0, absorbs all incoming damage (no spillover), triggers breach backlash
			target.Transformation.CurrentFormHP = 0
			w.triggerGuHouseBreachLocked(target)
			return 0 // No spillover!
		}
		return 0
	}

	// Deduct from Form HP layer if available (standard Full / Partial metamorphosis)
	if target.Transformation.CurrentFormHP > 0 {
		if target.Transformation.CurrentFormHP >= reducedDamage {
			target.Transformation.CurrentFormHP -= reducedDamage
			target.Revision++
			return 0 // Absorbed completely
		}

		// Form HP Broken
		spillover := reducedDamage - target.Transformation.CurrentFormHP
		target.Transformation.CurrentFormHP = 0
		w.revertTransformationLocked(target, "form_broken", true)
		return spillover
	}

	return reducedDamage
}

func (w *World) triggerGuHouseBreachLocked(entity *Entity) {
	if entity == nil {
		return
	}
	w.revertTransformationLocked(entity, "structural_breach", false)

	// Backlash damage: 30% of entity.Vitals.MaxHP, retaining minimum 1 HP
	backlash := entity.Vitals.MaxHP * 0.30
	entity.Vitals.HP = math.Max(1.0, entity.Vitals.HP-backlash)

	// Stun for 60 ticks (3.0s)
	w.applyStatusLocked(nil, entity, combat.Skill{
		StatusID:            "STUN",
		StatusDurationTicks: 60,
		StatusDebuff:        true,
	})

	// Severe damage: Gu house durability drops to 0
	entity.GuHouseDurability = 0
	entity.Revision++
}

// RevertTransformation restores the entity back to standard form.
func (w *World) RevertTransformation(entity *Entity, reason string, applyExhaustion bool) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	w.revertTransformationLocked(entity, reason, applyExhaustion)
}

func (w *World) revertTransformationLocked(entity *Entity, reason string, applyExhaustion bool) {
	if entity == nil || (!entity.Transformation.Active && !entity.Transformation.IsWindup) {
		return
	}

	// Restore original hitbox radius
	if entity.Transformation.OriginalRadius > 0 {
		entity.Radius = entity.Transformation.OriginalRadius
	}

	// Restore human skill bar if Full Metamorphosis or Gu House was active
	if (entity.Transformation.FormKind == TransformationFull || entity.Transformation.FormKind == TransformationGuHouse) && entity.Transformation.SavedSkills.Skills[1].ID != "" {
		RestoreSkillPalette(entity, w.tick, entity.Transformation.SavedSkills)
	}

	formID := entity.Transformation.FormID
	formKind := entity.Transformation.FormKind
	maxFormHP := entity.Transformation.MaxFormHP
	currentFormHP := entity.Transformation.CurrentFormHP
	isBroken := reason == "form_broken" || reason == "mana_depleted" || reason == "structural_breach"

	if formKind == TransformationGuHouse && maxFormHP > 0 {
		if reason == "structural_breach" {
			entity.GuHouseDurability = 0
		} else {
			// Tiered durability degradation on safe exit: 0.5% min + 10% * (1 - Current/Max)
			hpRatio := math.Max(0, math.Min(1, currentFormHP/maxFormHP))
			wear := 0.5 + 10.0*(1.0-hpRatio)
			entity.GuHouseDurability = math.Max(0, entity.GuHouseDurability-wear)
			entity.GuHouseWearDamage += wear
		}
	}

	entity.Transformation = TransformationState{
		Active:         false,
		OriginalRadius: entity.Transformation.OriginalRadius,
	}

	// Also clear legacy Combat.FormActive
	entity.Combat.FormActive = false
	entity.Combat.FormManaDrainPerTick = 0
	entity.Combat.FormDefenseBonus = 0
	entity.Revision++

	eventType := combat.EventTransformationEnded
	if isBroken {
		eventType = combat.EventTransformationBroken
	}

	w.emitEvent(combat.Event{
		Type:     eventType,
		SourceID: string(entity.ID),
		TargetID: string(entity.ID),
		SkillID:  formID,
		Reason:   reason,
	})

	// If broken or exhausted, apply temporary exhaustion debuff (reduce speed)
	if applyExhaustion {
		w.applyStatusLocked(nil, entity, combat.Skill{
			StatusID:             "EXHAUSTION",
			StatusDurationTicks:  100, // 5s at 20Hz
			StatusMoveMultiplier: 0.7, // -30% movement speed
			StatusDebuff:         true,
		})
	}
}

// GetFormSkillProfile retrieves the authored skill profile for a form.
func GetFormSkillProfile(formID string) (FormSkillProfile, bool) {
	if formID == "FORM_FLAME_BEAR" {
		return GetFlameBearFormProfile(), true
	}

	rec, found := GetTransformationForm(formID)
	if !found {
		return FormSkillProfile{}, false
	}

	if rec.SkillsJSON != "" {
		var skills []combat.Skill
		if err := json.Unmarshal([]byte(rec.SkillsJSON), &skills); err == nil && len(skills) > 0 {
			var profile FormSkillProfile
			profile.FormID = rec.FormID
			profile.FormName = rec.Name
			for _, sk := range skills {
				if sk.Slot >= 1 && sk.Slot <= 6 {
					if sk.ID == "REVERT_METAMORPHOSIS" || sk.Slot == 6 {
						sk.ID = "REVERT_METAMORPHOSIS"
						sk.Mode = combat.ModeSelf
					} else if sk.Mode == "" {
						sk.Mode = combat.ModeDirection
					}
					profile.Skills[sk.Slot] = sk
				}
			}
			// Slot 6 default Revert if empty
			if profile.Skills[6].ID == "" {
				profile.Skills[6] = combat.Skill{
					ID: "REVERT_METAMORPHOSIS", Slot: 6, Name: "Giải Trừ Biến Thân", Mode: combat.ModeSelf,
					ManaCost: 0, CooldownTicks: 10,
				}
			}
			return profile, true
		}
	}

	return FormSkillProfile{}, false
}

// GetFlameBearFormProfile returns the authored skill palette for Thái Cổ Viêm Hùng.
func GetFlameBearFormProfile() FormSkillProfile {
	return FormSkillProfile{
		FormID:   "FORM_FLAME_BEAR",
		FormName: "Thái Cổ Viêm Hùng",
		Skills: [7]combat.Skill{
			{}, // 0 unused
			{
				ID: "BEAR_CLAW_SWIPE", Slot: 1, Name: "Hùng Trảo Cuồng Liệp", Mode: combat.ModeDirection,
				Range: 3.5, Damage: 65, ManaCost: 5, CooldownTicks: 20, // 1s
			},
			{
				ID: "BEAR_FERAL_CHARGE", Slot: 2, Name: "Bạo Nộ Cuồng Húc", Mode: combat.ModeDirection,
				Range: 8.0, Damage: 95, ManaCost: 15, CooldownTicks: 80, // 4s
			},
			{
				ID: "BEAR_EARTH_SHATTER", Slot: 3, Name: "Chấn Địa Nham Liệt", Mode: combat.ModeGround,
				Radius: 4.5, Damage: 120, ManaCost: 25, CooldownTicks: 120, // 6s
			},
			{
				ID: "BEAR_ROAR_INTIMIDATE", Slot: 4, Name: "Thao Thiết Thú Hống", Mode: combat.ModeSelf,
				Radius: 6.0, StatusID: "STUN", StatusDurationTicks: 30, ManaCost: 20, CooldownTicks: 160, // 8s
			},
			{
				ID: "BEAR_FLAME_RAMPAGE", Slot: 5, Name: "Vạn Thú Viêm Bạo", Mode: combat.ModeDirection,
				Range: 10.0, Damage: 250, ManaCost: 50, CooldownTicks: 300, // 15s
			},
			{
				ID: "REVERT_METAMORPHOSIS", Slot: 6, Name: "Giải Trừ Biến Thân", Mode: combat.ModeSelf,
				ManaCost: 0, CooldownTicks: 10,
			},
		},
	}
}

// CalculateGuHouseUpkeep computes the mana upkeep per second for an Immortal Gu House.
func CalculateGuHouseUpkeep(houseRank, playerRank, subRank, coreCount, auxCount int) float64 {
	if houseRank <= 0 {
		houseRank = 6
	}
	if playerRank <= 0 {
		playerRank = 6
	}
	if coreCount <= 0 {
		coreCount = 1
	}
	// 1. U_base = (N_core * 0.04 + N_aux * 0.0005) * S_aux
	uBaseRaw := float64(coreCount)*0.04 + float64(auxCount)*0.0005
	sAux := 1.0
	neededAux := 30 * coreCount
	if auxCount < neededAux {
		sAux += float64(neededAux-auxCount) / float64(100*coreCount)
	}
	uBase := uBaseRaw * sAux

	// 2. F_sub = 1.0 - (SubRank - 1) * 0.05
	if subRank < 1 {
		subRank = 1
	} else if subRank > 4 {
		subRank = 4
	}
	fSub := 1.0 - float64(subRank-1)*0.05

	// 3. M(R_H, R_P)
	m := 1.0
	rankDiff := houseRank - playerRank
	if rankDiff > 0 {
		// Player rank lower than house rank: penalty
		m = math.Pow(100.0, float64(rankDiff)) * math.Pow(1.5, float64(rankDiff))
	} else if rankDiff < 0 {
		// Player rank higher than house rank: discount
		pDiff := float64(-rankDiff)
		m = math.Pow(100.0, float64(rankDiff)) * math.Pow(0.8, pDiff)
	}

	return uBase * m * fSub
}

// BuildGuHouseSkillProfile generates the standard 6-slot House Strike palette.
func BuildGuHouseSkillProfile(house models.GuHouseCombatProfile) FormSkillProfile {
	rank := house.Rank
	if rank <= 0 {
		rank = 6
	}
	coreCount := len(house.CoreGuIDs)
	if coreCount <= 0 {
		coreCount = 1
	}
	auxCount := house.AuxGuCount

	// Slot 1: Artillery Strike (Pháo Kích)
	baseDmg := float64(rank*120 + coreCount*80)
	for _, act := range house.Actions {
		if act.BaseDamage > baseDmg {
			baseDmg = act.BaseDamage
		}
	}
	slot1Dmg := baseDmg * (1.0 + float64(auxCount)*0.005)
	slot1 := combat.Skill{
		Slot:             1,
		ID:               "HOUSE_STRIKE_ARTILLERY",
		Name:             "Trận Kích Pháo Lôi",
		Mode:             combat.ModeDirection,
		Range:            16.0,
		Damage:           slot1Dmg,
		ManaCost:         float64(10 + rank*2),
		CooldownTicks:    30, // 1.5s
		ProjectileSpeed:  24.0,
		ProjectileRadius: 0.6,
		ProjectileTicks:  40,
	}
	if len(house.CoreGuIDs) > 0 {
		slot1.GuInstanceID = house.CoreGuIDs[0]
	}

	// Slot 2: Siege Trample (Xung Kích)
	slot2Dmg := float64(rank*80 + auxCount*2)
	slot2 := combat.Skill{
		Slot:              2,
		ID:                "HOUSE_STRIKE_TRAMPLE",
		Name:              "Kim Cang Xung Kích",
		Mode:              combat.ModeDirection,
		Range:             10.0,
		Damage:            slot2Dmg,
		ManaCost:          float64(15 + rank*3),
		CooldownTicks:     80, // 4.0s
		MoveDistance:      10.0,
		MoveDurationTicks: 8,
		MoveBehavior:      "IMPACT",
		KnockbackDistance: 3.5,
	}

	// Slot 3: Fortress Barrier (Đại Trận Hộ Thể)
	shieldAmount := house.MaxHouseHP * 0.20
	if shieldAmount <= 0 {
		shieldAmount = 900
	}
	slot3 := combat.Skill{
		Slot:                3,
		ID:                  "HOUSE_STRIKE_BARRIER",
		Name:                "Hộ Thể Quang Thuẫn",
		Mode:                combat.ModeSelf,
		Shield:              shieldAmount,
		StatusDurationTicks: 200, // 10s
		ManaCost:            float64(25 + rank*4),
		CooldownTicks:       200, // 10s
	}
	if len(house.CoreGuIDs) > 1 {
		slot3.GuInstanceID = house.CoreGuIDs[1]
	}

	// Slot 4: Structural Mending (Tu Bổ Kết Cấu)
	healAmount := house.MaxHouseHP * 0.15
	if healAmount <= 0 {
		healAmount = 675
	}
	slot4 := combat.Skill{
		Slot:          4,
		ID:            "HOUSE_STRIKE_MENDING",
		Name:          "Đại Trận Tu Bổ",
		Mode:          combat.ModeSelf,
		Heal:          healAmount,
		WindupTicks:   20, // 1.0s
		ManaCost:      float64(35 + rank*5),
		CooldownTicks: 300, // 15s
	}

	// Slot 5: Ultimate Destruction (Thần Thông Tuyệt Diệt)
	slot5Dmg := float64(rank*250 + coreCount*150 + auxCount*10)
	slot5 := combat.Skill{
		Slot:           5,
		ID:             "HOUSE_STRIKE_ULTIMATE",
		Name:           "Đại Diệt Tuyệt Trận",
		Mode:           combat.ModeGround,
		Range:          12.0,
		Radius:         6.5,
		Damage:         slot5Dmg,
		WindupTicks:    30, // 1.5s
		ManaCost:       float64(60 + rank*10),
		CooldownTicks:  600, // 30s
		GuComponentIDs: house.CoreGuIDs,
	}

	// Slot 6: Disembark (Rời Tiên Cổ Phòng)
	slot6 := combat.Skill{
		Slot:          6,
		ID:            "REVERT_GU_HOUSE",
		Name:          "Rời Tiên Cổ Phòng",
		Mode:          combat.ModeSelf,
		ManaCost:      0,
		CooldownTicks: 20, // 1.0s
	}

	return FormSkillProfile{
		FormID:   house.ID,
		FormName: house.Name,
		Skills:   [7]combat.Skill{{}, slot1, slot2, slot3, slot4, slot5, slot6},
	}
}

func (w *World) completeGuHouseTransformationLocked(entity *Entity) {
	if entity == nil || entity.EquippedGuHouse == nil {
		return
	}
	house := entity.EquippedGuHouse

	if entity.Transformation.OriginalRadius == 0 {
		entity.Transformation.OriginalRadius = entity.Radius
	}

	radiusMult := 1.8
	modelScale := 1.8

	maxFormHP := house.MaxHouseHP
	if maxFormHP <= 0 {
		maxFormHP = 4500
	}
	formDefense := house.HouseDefense
	if formDefense <= 0 {
		formDefense = 80
	}
	dmgReduction := house.DamageReduction
	if dmgReduction <= 0 {
		dmgReduction = 0.25
	}

	playerRank := house.Rank
	subRank := 1
	if entity.Companion.Rank > 0 {
		playerRank = entity.Companion.Rank
	}

	upkeepPerSec := CalculateGuHouseUpkeep(house.Rank, playerRank, subRank, len(house.CoreGuIDs), house.AuxGuCount)

	profile := BuildGuHouseSkillProfile(*house)
	savedSkills := SwapSkillPalette(entity, w.tick, profile)

	entity.Transformation = TransformationState{
		Active:              true,
		FormID:              house.ID,
		FormName:            house.Name,
		FormKind:            TransformationGuHouse,
		ModelScale:          modelScale,
		CollisionRadiusMult: radiusMult,
		OriginalRadius:      entity.Transformation.OriginalRadius,
		MaxFormHP:           maxFormHP,
		CurrentFormHP:       maxFormHP,
		FormDefenseBonus:    formDefense,
		DamageReduction:     dmgReduction,
		Upkeep: UpkeepCost{
			ManaPerTick: upkeepPerSec / float64(TickRate),
		},
		ExpireTick:  0,
		IsWindup:    false,
		SavedSkills: savedSkills,
	}

	// Scale collision radius
	entity.Radius = entity.Transformation.OriginalRadius * radiusMult
	entity.Revision++

	w.emitEvent(combat.Event{
		Type:     combat.EventTransformationStarted,
		SourceID: string(entity.ID),
		TargetID: string(entity.ID),
		SkillID:  house.ID,
		Reason:   "gu_house_complete",
	})
}
