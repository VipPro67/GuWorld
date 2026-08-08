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
	StatMods map[string]float64
	// RefineBonus is an additive success-rate bonus for luyện cổ (refining),
	// in the 0..1 range.
	RefineBonus float64
	// DaoNganBonus maps a dao path to an additive multiplier for that path's
	// đạo ngân contribution in combat. E.g. {"ice": 0.2} => ×1.2 for ice.
	DaoNganBonus map[string]float64
	// PathDamageBonus maps a dao path to a percentage damage multiplier for
	// abilities of that path. E.g. {"dream": 0.15} => +15% dream-path damage.
	PathDamageBonus map[string]float64
	// PathResistanceBonus maps a dao path to a percentage reduction of damage
	// taken from that path. E.g. {"fire": 0.10} => -10% fire damage taken.
	PathResistanceBonus map[string]float64
	// LuckBonus is a flat increase of the luck stat.
	LuckBonus float64
	// MaxManaBonus is a percentage increase of the mana pool capacity.
	MaxManaBonus float64
	// ManaRegenBonus is a percentage increase of mana regeneration rate.
	ManaRegenBonus float64
	// CultivationEfficiencyBonus is a percentage increase of cultivation
	// gains (phá vách wall progress, ...).
	CultivationEfficiencyBonus float64
	// DaoMarkGainBonus is a percentage increase of Đạo Ngân earned.
	DaoMarkGainBonus float64
	// HealingReceivedBonus is a percentage increase of healing the player
	// receives (and HP regeneration).
	HealingReceivedBonus float64
	// MentalResistanceBonus is a percentage increase of resistance to
	// control/mental effects.
	MentalResistanceBonus float64
	// SoulDefenseBonus is a percentage reduction of soul damage taken.
	SoulDefenseBonus float64
}

// ValidRaces returns the set of accepted race keys.
func ValidRaces() map[string]bool {
	return map[string]bool{
		"human":                true, // Nhân Tộc
		"fur_people":           true, // Người Lông
		"merman":               true, // Nhân Ngư
		"feathered":            true, // Vũ Dân
		"inkman":               true, // Mặc Nhân
		"rockman":              true, // Thạch Nhân
		"snowman":              true, // Tuyết Dân
		"variant_beastman":     true, // Thú Nhân
		"dragonman":            true, // Long Nhân
		"giant_sun_descendant": true, // Cự Dương Huyết Mạch
	}
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
	return map[string]bool{
		"desolate_ancient_moon_body":              true, // Hoang Cổ Nguyệt Thể
		"verdant_great_sun_body":                  true, // Thanh Liên Đại Nhật Thể
		"northern_dark_ice_soul_body":             true, // Bắc Minh Băng Phách Thể
		"north_ming_ice_body":                     true, // alias cũ của Bắc Minh Băng Phách Thể
		"ancient_yin_virtue_body":                 true, // Cổ Nguyệt Âm Đức Thể
		"blazing_glory_lightning_brilliance_body": true, // Hạo Kiếp Viêm Quang Lôi Quang Thể
		"pure_dream_reality_seeker_body":          true, // Thuần Mộng Cầu Chân Thể
		"unrestrained_heavenly_mark_body":         true, // Vô Cực Thiên Ngân Thể
		"great_strength_true_martial_body":        true, // Đại Lực Chân Vũ Thể
		"carefree_lotus_heart_body":               true, // Tiêu Dao Liên Tâm Thể
		"ancient_soul_virtue_body":                true, // Cổ Hồn Âm Đức Thể
	}
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

	switch p.Race {
	case "human": // Nhân Tộc
		// không bonus đặc biệt
	case "fur_people": // Người Lông
		fx.RefineBonus += 0.10
	case "merman": // Nhân Ngư
		fx.DaoNganBonus["water"] += 0.10
		fx.PathDamageBonus["water"] += 0.05
	case "feathered": // Vũ Dân
		fx.StatMods["speed"] += 0.10
		fx.DaoNganBonus["space"] += 0.05
	case "inkman": // Mặc Nhân
		fx.DaoNganBonus["wisdom"] += 0.08
		fx.DaoNganBonus["soul"] += 0.05
	case "rockman": // Thạch Nhân
		fx.StatMods["defense"] += 0.15
		fx.StatMods["speed"] += -0.10
	case "snowman": // Tuyết Dân
		fx.DaoNganBonus["ice"] += 0.15
		fx.PathResistanceBonus["fire"] += 0.10
	case "variant_beastman": // Thú Nhân
		fx.StatMods["attack"] += 0.08
		fx.StatMods["defense"] += 0.05
		fx.RefineBonus += -0.05
	case "dragonman": // Long Nhân
		fx.StatMods["attack"] += 0.10
		fx.StatMods["defense"] += 0.10
		fx.RefineBonus += -0.05
	case "giant_sun_descendant": // Cự Dương Huyết Mạch
		fx.LuckBonus += 5
		fx.CultivationEfficiencyBonus += 0.05
	}

	for _, status := range p.Statuses {
		switch status {
		case "severely_wounded": // Trọng Thương
			fx.StatMods["attack"] += -0.20
			fx.StatMods["defense"] += -0.20
			fx.StatMods["speed"] += -0.20
		}
	}

	switch p.Constitution {
	case "desolate_ancient_moon_body": // Hoang Cổ Nguyệt Thể
		fx.CultivationEfficiencyBonus += 0.20
		fx.MaxManaBonus += 0.15
	case "verdant_great_sun_body": // Thanh Liên Đại Nhật Thể
		fx.DaoNganBonus["fire"] += 0.20
		fx.StatMods["attack"] += 0.10
	case "northern_dark_ice_soul_body", "north_ming_ice_body": // Bắc Minh Băng Phách Thể
		fx.DaoNganBonus["ice"] += 0.20
		fx.StatMods["defense"] += 0.10
	case "ancient_yin_virtue_body": // Cổ Nguyệt Âm Đức Thể
		fx.DaoNganBonus["soul"] += 0.15
		fx.HealingReceivedBonus += 0.10
	case "blazing_glory_lightning_brilliance_body": // Hạo Kiếp Viêm Quang Lôi Quang Thể
		fx.DaoNganBonus["lightning"] += 0.20
		fx.StatMods["speed"] += 0.10
	case "pure_dream_reality_seeker_body": // Thuần Mộng Cầu Chân Thể
		fx.DaoNganBonus["dream"] += 0.25
		fx.PathDamageBonus["dream"] += 0.15
	case "unrestrained_heavenly_mark_body": // Vô Cực Thiên Ngân Thể
		fx.DaoMarkGainBonus += 0.15
	case "great_strength_true_martial_body": // Đại Lực Chân Vũ Thể
		fx.DaoNganBonus["strength"] += 0.20
		fx.StatMods["attack"] += 0.15
	case "carefree_lotus_heart_body": // Tiêu Dao Liên Tâm Thể
		fx.DaoNganBonus["wisdom"] += 0.15
		fx.MentalResistanceBonus += 0.20
	case "ancient_soul_virtue_body": // Cổ Hồn Âm Đức Thể
		fx.DaoNganBonus["soul"] += 0.20
		fx.SoulDefenseBonus += 0.15
	}

	// Mọi thể chất đều tăng hồi mana
	if p.Constitution != "" && ValidConstitutions()[p.Constitution] {
		fx.ManaRegenBonus += 1
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

// RaceDisplayName returns the Vietnamese label for a race key.
func RaceDisplayName(race string) string {
	switch race {
	case "human":
		return "Nhân Tộc"
	case "fur_people":
		return "Người Lông"
	case "merman":
		return "Nhân Ngư"
	case "feathered":
		return "Vũ Dân"
	case "inkman":
		return "Mặc Nhân"
	case "rockman":
		return "Thạch Nhân"
	case "snowman":
		return "Tuyết Dân"
	case "variant_beastman":
		return "Thú Nhân"
	case "dragonman":
		return "Long Nhân"
	case "giant_sun_descendant":
		return "Cự Dương Huyết Mạch"
	default:
		return race
	}
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
	switch constitution {
	case "desolate_ancient_moon_body":
		return "Hoang Cổ Nguyệt Thể"
	case "verdant_great_sun_body":
		return "Thanh Liên Đại Nhật Thể"
	case "northern_dark_ice_soul_body", "north_ming_ice_body":
		return "Bắc Minh Băng Phách Thể"
	case "ancient_yin_virtue_body":
		return "Cổ Nguyệt Âm Đức Thể"
	case "blazing_glory_lightning_brilliance_body":
		return "Hạo Kiếp Viêm Quang Lôi Quang Thể"
	case "pure_dream_reality_seeker_body":
		return "Thuần Mộng Cầu Chân Thể"
	case "unrestrained_heavenly_mark_body":
		return "Vô Cực Thiên Ngân Thể"
	case "great_strength_true_martial_body":
		return "Đại Lực Chân Vũ Thể"
	case "carefree_lotus_heart_body":
		return "Tiêu Dao Liên Tâm Thể"
	case "ancient_soul_virtue_body":
		return "Cổ Hồn Âm Đức Thể"
	default:
		return constitution
	}
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
