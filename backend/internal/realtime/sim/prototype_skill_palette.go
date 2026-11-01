package sim

import (
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// FormSkillProfile defines the complete replacement action bar for a Full Metamorphosis form.
type FormSkillProfile struct {
	FormID      string
	FormName    string
	Skills      [7]combat.Skill // Slots 1..6 (Slot 6 is typically Revert Form)
	BasicAttack combat.Skill
}

// PartialOverlayProfile defines enhancements for a Partial Transformation (e.g. Wolf Skin, Eagle Wings).
type PartialOverlayProfile struct {
	FormID            string
	FormName          string
	EnhancedBasicDmg  float64
	BasicAttackStatus string // e.g. "BLEED"
	ExtraSkillSlot    int    // e.g. Slot 7 or 8
	ExtraSkill        combat.Skill
	SpeedMultiplier   float64
	DefenseBonus      float64
}

// StoredSkillState captures human skills and cooldowns before metamorphosis.
type StoredSkillState struct {
	Skills    [13]combat.Skill
	Cooldowns [13]uint64
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

// ValidateFormSkillExecution verifies if a cast command is legally allowed in the current form.
func ValidateFormSkillExecution(entity *Entity, slot int, isFormActive bool, formKind TransformationKind) (combat.Skill, string) {
	if entity == nil || entity.Vitals.HP <= 0 {
		return combat.Skill{}, "caster_dead"
	}
	if slot < 1 || slot > 12 {
		return combat.Skill{}, "invalid_slot"
	}

	skill := entity.Combat.Skills[slot]
	if skill.ID == "" {
		return combat.Skill{}, "empty_skill_slot"
	}

	// 1. Full Metamorphosis constraints:
	if isFormActive && formKind == TransformationFull {
		// Only skills belonging to the form are permitted
		if slot > 6 {
			return combat.Skill{}, "form_skill_locked_extra_slot"
		}
	}

	// 2. Resource check:
	if entity.Combat.Mana < skill.ManaCost {
		return combat.Skill{}, "insufficient_mana"
	}

	// 3. Cooldown check:
	// Assuming external cooldown verification against entity.Combat.Cooldowns[slot]
	return skill, ""
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
