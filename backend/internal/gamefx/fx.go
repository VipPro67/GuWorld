// Package gamefx resolves gameplay modifiers granted by a player's character
// attributes (race, gender, constitution, statuses). It is shared by the
// handler and service layers to avoid an import cycle.
package gamefx

import (
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Effects aggregates the gameplay modifiers granted by a player's
// race (chủng tộc), constitution (thể chất) and active statuses (trạng thái).
type Effects struct {
	// StatMods maps a passive stat key ("attack", "defense", "speed", ...) to a
	// percentage modifier applied on the stat total. E.g. 0.10 => +10%.
	StatMods map[string]float64 `json:"stat_mods"`
	// RefineBonus is an additive success-rate bonus for luyện cổ (refining),
	// in the 0..1 range.
	RefineBonus float64 `json:"refine"`
	// DaoNganBonus maps a dao path to an additive multiplier for that path's
	// đạo ngân contribution in combat. E.g. {"ice": 0.2} => ×1.2 for ice.
	DaoNganBonus map[string]float64 `json:"dao_mark"`
	// PathDamageBonus maps a dao path to a percentage damage multiplier for
	// abilities of that path. E.g. {"dream": 0.15} => +15% dream-path damage.
	PathDamageBonus map[string]float64 `json:"path_damage"`
	// PathResistanceBonus maps a dao path to a percentage reduction of damage
	// taken from that path. E.g. {"fire": 0.10} => -10% fire damage taken.
	PathResistanceBonus map[string]float64 `json:"path_resistance"`
	// LuckBonus is a flat increase of the luck stat.
	LuckBonus float64 `json:"luck"`
	// MaxManaBonus is a percentage increase of the mana pool capacity.
	MaxManaBonus float64 `json:"max_mana"`
	// ManaRegenBonus is a percentage increase of mana regeneration rate.
	ManaRegenBonus float64 `json:"mana_regen"`
	// CultivationEfficiencyBonus is a percentage increase of cultivation
	// gains (phá vách wall progress, ...).
	CultivationEfficiencyBonus float64 `json:"cultivation_efficiency"`
	// DaoMarkGainBonus is a percentage increase of Đạo Ngân earned.
	DaoMarkGainBonus float64 `json:"dao_mark_gain"`
	// HealingReceivedBonus is a percentage increase of healing the player
	// receives (and HP regeneration).
	HealingReceivedBonus float64 `json:"healing_received"`
	// MentalResistanceBonus is a percentage increase of resistance to
	// control/mental effects.
	MentalResistanceBonus float64 `json:"mental_resistance"`
	// SoulDefenseBonus is a percentage reduction of soul damage taken.
	SoulDefenseBonus float64 `json:"soul_defense"`
	// ApertureStressBonus is an additive multiplier on the aperture stress
	// (áp lực không khiếu) gained per trùng kích attempt. 1.0 => stress ×2.
	ApertureStressBonus float64 `json:"aperture_stress_bonus"`
}

// ValidRaces returns the set of accepted race keys.
func ValidRaces() map[string]bool {
	defs := effectiveRaces()
	out := make(map[string]bool, len(defs))
	for key := range defs {
		out[key] = true
	}
	return out
}

// ValidGenders returns the set of accepted gender keys.
func ValidGenders() map[string]bool {
	return map[string]bool{
		"male":   true,
		"female": true,
	}
}

// ValidConstitutions returns the set of accepted constitution keys.
func ValidConstitutions() map[string]bool {
	defs := effectiveConstitutions()
	out := make(map[string]bool, len(defs))
	for key := range defs {
		out[key] = true
	}
	return out
}

// ValidStatuses returns the set of accepted status keys.
func ValidStatuses() map[string]bool {
	return map[string]bool{
		"severely_wounded": true, // Trọng Thương
	}
}

// Get returns all effects from a player's character fields.
func Get(p *models.Player) Effects {
	fx := Effects{
		StatMods:            map[string]float64{},
		DaoNganBonus:        map[string]float64{},
		PathDamageBonus:     map[string]float64{},
		PathResistanceBonus: map[string]float64{},
	}
	if p == nil {
		return fx
	}

	if def, ok := effectiveRaces()[p.Race]; ok {
		addEffects(&fx, def.Effects)
	}

	for _, status := range p.Statuses {
		switch status {
		case "severely_wounded": // Trọng Thương
			fx.StatMods["attack"] += -0.20
			fx.StatMods["defense"] += -0.20
			fx.StatMods["speed"] += -0.20
		}
	}

	if def, ok := effectiveConstitutions()[p.Constitution]; ok {
		addEffects(&fx, def.Effects)
	}

	return fx
}

// MaxManaBonus returns the constitution percentage bonus on max mana.
func MaxManaBonus(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	return Get(p).MaxManaBonus
}

// EffectiveMaxMana returns the player's mana pool capacity including the
// constitution MaxManaBonus.
func EffectiveMaxMana(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	return p.MaxMana * (1.0 + MaxManaBonus(p))
}

// PathMasteryMultiplier returns the highest path-mastery multiplier across the
// given paths. 1.0 means no mastery bonus.
func PathMasteryMultiplier(pathMasteries map[string]string, paths []string) float64 {
	masteryMultiplier := 1.0
	if pathMasteries == nil {
		return masteryMultiplier
	}
	for _, path := range paths {
		cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
		mLevel, ok := pathMasteries[cleanPath]
		if !ok {
			mLevel, ok = pathMasteries[path]
		}
		if !ok {
			continue
		}
		mult := 1.0
		switch mLevel {
		case "Ordinary":
			mult = 1.0
		case "Master":
			mult = 1.1
		case "Grandmaster":
			mult = 1.25
		case "Supreme Grandmaster":
			mult = 1.5
		}
		if mult > masteryMultiplier {
			masteryMultiplier = mult
		}
	}
	return masteryMultiplier
}

// PassiveHPBonus returns the total flat HP granted by equipped ACTIVE Gus
// (passive.stat_base_hp), scaled by path mastery and dao marks. Mirrors the
// "hp" key computed in GetPassiveStats.
func PassiveHPBonus(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	total := 0.0
	bestRank := map[string]int{}
	for _, gu := range p.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if r, ok := bestRank[gu.GuID]; !ok || gu.Rank > r {
			bestRank[gu.GuID] = gu.Rank
		}
	}
	applied := map[string]bool{}
	for _, gu := range p.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if gu.Rank != bestRank[gu.GuID] || applied[gu.GuID] {
			continue
		}
		applied[gu.GuID] = true
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok || spec.Effects.Passive == nil || spec.Effects.Passive.StatBaseHP <= 0 {
			continue
		}
		mult := PathMasteryMultiplier(p.PathMasteries, spec.Path)
		daoMult := DaoMultiplierForPaths(p.DaoMarks, spec.Path, nil)
		total += spec.Effects.Passive.StatBaseHP * mult * daoMult
	}
	return total
}

// EffectiveMaxHP returns the player's max HP including passive HP bonuses
// from equipped Gus. This is the authoritative max HP for HUD, regen caps and
// combat.
func EffectiveMaxHP(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	return p.BaseHP + PassiveHPBonus(p)
}

// RaceDisplayName returns the Vietnamese label for a race key.
func RaceDisplayName(race string) string {
	if def, ok := effectiveRaces()[race]; ok {
		return def.Name
	}
	return race
}

// GenderDisplayName returns the Vietnamese label for a gender key.
func GenderDisplayName(gender string) string {
	switch gender {
	case "male":
		return "Nam"
	case "female":
		return "Nữ"
	default:
		return gender
	}
}

// ConstitutionDisplayName returns the Vietnamese label for a constitution key.
func ConstitutionDisplayName(constitution string) string {
	if def, ok := effectiveConstitutions()[constitution]; ok {
		return def.Name
	}
	return constitution
}

// StatusDisplayName returns the Vietnamese label for a status key.
func StatusDisplayName(status string) string {
	switch status {
	case "severely_wounded":
		return "Trọng Thương"
	default:
		return status
	}
}

// DaoPathMultiplier computes the standard dao-marks multiplier for a single
// path and applies an additive per-path bonus (e.g. from a constitution).
// marks <= 0 means no dao marks for the path (base multiplier 1.0).
func DaoPathMultiplier(marks int, bonus float64) float64 {
	if marks <= 0 {
		return 1.0 + bonus
	}
	if marks <= 10000 {
		return 1.0 + float64(marks)/100.0 + bonus
	}
	return 1.0 + 100.0 + float64(marks-10000)/500.0 + bonus
}

// DaoMultiplierForPaths computes the maximum dao multiplier across the given
// paths, including per-path bonuses from daoBonus (e.g. DaoNganBonus).
func DaoMultiplierForPaths(daoMarks map[string]int, paths []string, daoBonus map[string]float64) float64 {
	if len(paths) == 0 {
		paths = []string{"strength"}
	}
	maxMult := 1.0
	for _, path := range paths {
		cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
		var amt int
		if daoMarks != nil {
			amt = daoMarks[cleanPath]
		}
		bonus := 0.0
		if daoBonus != nil {
			bonus = daoBonus[cleanPath]
		}
		mult := DaoPathMultiplier(amt, bonus)
		if mult > maxMult {
			maxMult = mult
		}
	}
	return maxMult
}

// PathDamageMultiplier returns the percentage damage multiplier for a path
// from pathDamage (e.g. PathDamageBonus). 1.0 means no bonus.
func PathDamageMultiplier(path string, pathDamage map[string]float64) float64 {
	if pathDamage == nil {
		return 1.0
	}
	cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
	return 1.0 + pathDamage[cleanPath]
}

// PathDamageMultiplierForPaths returns the maximum path-damage multiplier
// across the given paths. 1.0 means no bonus.
func PathDamageMultiplierForPaths(paths []string, pathDamage map[string]float64) float64 {
	if pathDamage == nil {
		return 1.0
	}
	if len(paths) == 0 {
		paths = []string{"strength"}
	}
	maxMult := 1.0
	for _, path := range paths {
		if mult := PathDamageMultiplier(path, pathDamage); mult > maxMult {
			maxMult = mult
		}
	}
	return maxMult
}
