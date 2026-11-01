package effects

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

const MaxHotbarSlots = 6

// EquippedGu is the minimal immutable loadout input from the character adapter.
type EquippedGu struct {
	InstanceID string
	GU         string
	Rank       int
}

type LoadoutResult struct {
	Skills   []combat.Skill
	Skipped  []EquippedGu
	Revision string
}

// BuildLoadout maps equipped Gu instances to hotbar slots 1..6 using the
// compiled realtime catalog. Unsupported/missing profiles are skipped (and
// reported) rather than cast as an empty skill.
func BuildLoadout(catalog *CompiledCatalog, equipped []EquippedGu) LoadoutResult {
	result := LoadoutResult{}
	formAdded := false
	if catalog == nil {
		result.Skipped = append(result.Skipped, equipped...)
		return result
	}
	ordered := append([]EquippedGu(nil), equipped...)
	hash := sha256.New()
	for index, instance := range ordered {
		_, _ = hash.Write([]byte(instance.InstanceID + "\x00" + instance.GU + "\x00" + strconv.Itoa(instance.Rank) + "\x00"))
		if index >= MaxHotbarSlots {
			result.Skipped = append(result.Skipped, instance)
			continue
		}
		profile, ok := catalog.Profile(instance.GU, instance.Rank)
		if !ok {
			result.Skipped = append(result.Skipped, instance)
			continue
		}
		if profile.Active != nil {
			skill := profile.Active.Skill
			skill.Slot = index + 1
			skill.Rank = instance.Rank
			if skill.Valid() {
				skill.GuInstanceID = instance.InstanceID
				result.Skills = append(result.Skills, skill)
			}
		}
		if !formAdded && profile.FormActivation != nil && profile.FormAbility != nil {
			activation, ability := *profile.FormActivation, profile.FormAbility.Skill
			if activation.Valid() && ability.Valid() {
				result.Skills = append(result.Skills, activation, ability)
				formAdded = true
			}
		}
	}
	result.Revision = hex.EncodeToString(hash.Sum(nil))
	sort.SliceStable(result.Skills, func(i, j int) bool { return result.Skills[i].Slot < result.Skills[j].Slot })
	return result
}

// SkillInfos projects a loadout for world.welcome. Damage/passives stay server
// side; the client receives only mode/name/range/cooldowns.
func SkillInfos(skills []combat.Skill) []SkillInfo {
	infos := make([]SkillInfo, 0, len(skills))
	for _, skill := range skills {
		infos = append(infos, SkillInfo{
			Slot: uint8(skill.Slot), ID: skill.ID, Name: skill.Name, Mode: string(skill.Mode),
			Shape: string(skill.Shape), AngleDeg: skill.AngleDeg, LengthUnits: skill.LengthUnits,
			WidthUnits: skill.WidthUnits, InnerRadius: skill.InnerRadius,
			ManaCost: skill.ManaCost, Range: skill.Range, RadiusUnits: skill.Radius,
			CooldownTicks: skill.CooldownTicks, WindupTicks: skill.WindupTicks,
			MoveDistance: skill.MoveDistance, MoveDurationTicks: skill.MoveDurationTicks, MoveBehavior: skill.MoveBehavior,
			IsToggle:       skill.IsToggle,
			FormActivation: skill.FormActivation, FormAbility: skill.FormAbility,
		})
	}
	return infos
}

// SkillInfo mirrors protocol.SkillInfo without importing protocol (protocol
// must not depend on effects). Gateway converts this DTO.
type SkillInfo struct {
	Slot              uint8
	ID                string
	Name              string
	Mode              string
	Shape             string
	AngleDeg          float64
	LengthUnits       float64
	WidthUnits        float64
	InnerRadius       float64
	ManaCost          float64
	Range             float64
	RadiusUnits       float64
	CooldownTicks     uint64
	WindupTicks       uint64
	MoveDistance      float64
	MoveDurationTicks uint64
	MoveBehavior      string
	IsToggle          bool
	FormActivation    bool
	FormAbility       bool
}
