package gamefx

import (
	"math"
	"strings"
)

// DaoGroup represents a broad philosophical rule category for cultivation paths.
type DaoGroup string

const (
	DaoGroupElemental          DaoGroup = "ELEMENTAL"
	DaoGroupSpaceTime          DaoGroup = "SPACE_TIME"
	DaoGroupRuleOrder          DaoGroup = "RULE_ORDER"
	DaoGroupKineticWeapon      DaoGroup = "KINETIC_WEAPON"
	DaoGroupBodyLife           DaoGroup = "BODY_LIFE"
	DaoGroupSoulMind           DaoGroup = "SOUL_MIND"
	DaoGroupInformationStealth DaoGroup = "INFORMATION_STEALTH"
	DaoGroupOrganicCorruption  DaoGroup = "ORGANIC_CORRUPTION"
	DaoGroupCraftAlchemy       DaoGroup = "CRAFT_ALCHEMY"
	DaoGroupHeavenHuman        DaoGroup = "HEAVEN_HUMAN"
)

// CanonicalPaths contains all 49 standardized path keys in GuWorld.
var CanonicalPaths = []string{
	// ELEMENTAL (13)
	"metal", "wood", "water", "fire", "earth", "wind", "lightning", "ice", "cloud", "light", "dark", "star", "sound",
	// SPACE_TIME (2)
	"space", "time",
	// RULE_ORDER (3)
	"rule", "restriction", "formation",
	// KINETIC_WEAPON (4)
	"sword", "blade", "weapon", "killing",
	// BODY_LIFE (4)
	"strength", "bone", "transformation", "food",
	// SOUL_MIND (5)
	"soul", "wisdom", "emotion", "enslavement", "dream",
	// INFORMATION_STEALTH (5)
	"information", "luck", "theft", "phantom", "painting",
	// ORGANIC_CORRUPTION (2)
	"blood", "poison",
	// CRAFT_ALCHEMY (2)
	"refinement", "pill",
	// HEAVEN_HUMAN (3)
	"heaven", "human", "qi",
}

// PathToDaoGroup maps each canonical path key to its corresponding DaoGroup.
var PathToDaoGroup = map[string]DaoGroup{
	// ELEMENTAL
	"metal": DaoGroupElemental, "wood": DaoGroupElemental, "water": DaoGroupElemental,
	"fire": DaoGroupElemental, "earth": DaoGroupElemental, "wind": DaoGroupElemental,
	"lightning": DaoGroupElemental, "ice": DaoGroupElemental, "cloud": DaoGroupElemental,
	"light": DaoGroupElemental, "dark": DaoGroupElemental, "star": DaoGroupElemental, "sound": DaoGroupElemental,

	// SPACE_TIME
	"space": DaoGroupSpaceTime, "time": DaoGroupSpaceTime,

	// RULE_ORDER
	"rule": DaoGroupRuleOrder, "restriction": DaoGroupRuleOrder, "formation": DaoGroupRuleOrder,

	// KINETIC_WEAPON
	"sword": DaoGroupKineticWeapon, "blade": DaoGroupKineticWeapon,
	"weapon": DaoGroupKineticWeapon, "killing": DaoGroupKineticWeapon,

	// BODY_LIFE
	"strength": DaoGroupBodyLife, "bone": DaoGroupBodyLife,
	"transformation": DaoGroupBodyLife, "food": DaoGroupBodyLife,

	// SOUL_MIND
	"soul": DaoGroupSoulMind, "wisdom": DaoGroupSoulMind,
	"emotion": DaoGroupSoulMind, "enslavement": DaoGroupSoulMind, "dream": DaoGroupSoulMind,

	// INFORMATION_STEALTH
	"information": DaoGroupInformationStealth, "luck": DaoGroupInformationStealth,
	"theft": DaoGroupInformationStealth, "phantom": DaoGroupInformationStealth, "painting": DaoGroupInformationStealth,
	"investigation": DaoGroupInformationStealth,

	// ORGANIC_CORRUPTION
	"blood": DaoGroupOrganicCorruption, "poison": DaoGroupOrganicCorruption, "death": DaoGroupOrganicCorruption,

	// CRAFT_ALCHEMY
	"refinement": DaoGroupCraftAlchemy, "pill": DaoGroupCraftAlchemy, "creation": DaoGroupCraftAlchemy,

	// RULE_ORDER / DESTRUCTION
	"destruction": DaoGroupRuleOrder,

	// HEAVEN_HUMAN
	"heaven": DaoGroupHeavenHuman, "human": DaoGroupHeavenHuman, "qi": DaoGroupHeavenHuman,
}

// groupCompatibilityMatrix stores default symmetric compatibility between DaoGroups (0.0 to 1.0).
var groupCompatibilityMatrix = map[DaoGroup]map[DaoGroup]float64{
	DaoGroupElemental: {
		DaoGroupElemental: 0.88, DaoGroupSpaceTime: 0.90, DaoGroupRuleOrder: 0.85,
		DaoGroupKineticWeapon: 0.85, DaoGroupBodyLife: 0.85, DaoGroupSoulMind: 0.85,
		DaoGroupInformationStealth: 0.85, DaoGroupOrganicCorruption: 0.80,
		DaoGroupCraftAlchemy: 0.92, DaoGroupHeavenHuman: 0.88,
	},
	DaoGroupSpaceTime: {
		DaoGroupElemental: 0.90, DaoGroupSpaceTime: 0.95, DaoGroupRuleOrder: 0.95,
		DaoGroupKineticWeapon: 0.88, DaoGroupBodyLife: 0.85, DaoGroupSoulMind: 0.90,
		DaoGroupInformationStealth: 0.92, DaoGroupOrganicCorruption: 0.82,
		DaoGroupCraftAlchemy: 0.90, DaoGroupHeavenHuman: 0.92,
	},
	DaoGroupRuleOrder: {
		DaoGroupElemental: 0.85, DaoGroupSpaceTime: 0.95, DaoGroupRuleOrder: 0.95,
		DaoGroupKineticWeapon: 0.90, DaoGroupBodyLife: 0.85, DaoGroupSoulMind: 0.90,
		DaoGroupInformationStealth: 0.94, DaoGroupOrganicCorruption: 0.80,
		DaoGroupCraftAlchemy: 0.94, DaoGroupHeavenHuman: 0.95,
	},
	DaoGroupKineticWeapon: {
		DaoGroupElemental: 0.85, DaoGroupSpaceTime: 0.88, DaoGroupRuleOrder: 0.90,
		DaoGroupKineticWeapon: 0.94, DaoGroupBodyLife: 0.90, DaoGroupSoulMind: 0.88,
		DaoGroupInformationStealth: 0.85, DaoGroupOrganicCorruption: 0.88,
		DaoGroupCraftAlchemy: 0.92, DaoGroupHeavenHuman: 0.88,
	},
	DaoGroupBodyLife: {
		DaoGroupElemental: 0.85, DaoGroupSpaceTime: 0.85, DaoGroupRuleOrder: 0.85,
		DaoGroupKineticWeapon: 0.90, DaoGroupBodyLife: 0.95, DaoGroupSoulMind: 0.88,
		DaoGroupInformationStealth: 0.85, DaoGroupOrganicCorruption: 0.92,
		DaoGroupCraftAlchemy: 0.88, DaoGroupHeavenHuman: 0.90,
	},
	DaoGroupSoulMind: {
		DaoGroupElemental: 0.85, DaoGroupSpaceTime: 0.90, DaoGroupRuleOrder: 0.90,
		DaoGroupKineticWeapon: 0.88, DaoGroupBodyLife: 0.88, DaoGroupSoulMind: 0.95,
		DaoGroupInformationStealth: 0.95, DaoGroupOrganicCorruption: 0.85,
		DaoGroupCraftAlchemy: 0.88, DaoGroupHeavenHuman: 0.90,
	},
	DaoGroupInformationStealth: {
		DaoGroupElemental: 0.85, DaoGroupSpaceTime: 0.92, DaoGroupRuleOrder: 0.94,
		DaoGroupKineticWeapon: 0.85, DaoGroupBodyLife: 0.85, DaoGroupSoulMind: 0.95,
		DaoGroupInformationStealth: 0.95, DaoGroupOrganicCorruption: 0.82,
		DaoGroupCraftAlchemy: 0.90, DaoGroupHeavenHuman: 0.92,
	},
	DaoGroupOrganicCorruption: {
		DaoGroupElemental: 0.80, DaoGroupSpaceTime: 0.82, DaoGroupRuleOrder: 0.80,
		DaoGroupKineticWeapon: 0.88, DaoGroupBodyLife: 0.92, DaoGroupSoulMind: 0.85,
		DaoGroupInformationStealth: 0.82, DaoGroupOrganicCorruption: 0.94,
		DaoGroupCraftAlchemy: 0.85, DaoGroupHeavenHuman: 0.85,
	},
	DaoGroupCraftAlchemy: {
		DaoGroupElemental: 0.92, DaoGroupSpaceTime: 0.90, DaoGroupRuleOrder: 0.94,
		DaoGroupKineticWeapon: 0.92, DaoGroupBodyLife: 0.88, DaoGroupSoulMind: 0.88,
		DaoGroupInformationStealth: 0.90, DaoGroupOrganicCorruption: 0.85,
		DaoGroupCraftAlchemy: 0.96, DaoGroupHeavenHuman: 0.92,
	},
	DaoGroupHeavenHuman: {
		DaoGroupElemental: 0.88, DaoGroupSpaceTime: 0.92, DaoGroupRuleOrder: 0.95,
		DaoGroupKineticWeapon: 0.88, DaoGroupBodyLife: 0.90, DaoGroupSoulMind: 0.90,
		DaoGroupInformationStealth: 0.92, DaoGroupOrganicCorruption: 0.85,
		DaoGroupCraftAlchemy: 0.92, DaoGroupHeavenHuman: 0.95,
	},
}

func pairKey(a, b string) string {
	a = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(a)), " path")
	b = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(b)), " path")
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// pairOverrides stores specific path pair compatibilities that override group defaults.
var pairOverrides = map[string]float64{
	// Tương khắc sâu
	pairKey("fire", "water"):  0.55,
	pairKey("fire", "ice"):    0.60,
	pairKey("light", "dark"):  0.55,
	pairKey("blood", "light"): 0.65,
	pairKey("poison", "food"): 0.60,

	// Tương sinh & Đồng nguyên
	pairKey("space", "time"):           0.97,
	pairKey("wisdom", "information"):   0.98,
	pairKey("wisdom", "soul"):          0.95,
	pairKey("strength", "bone"):        0.96,
	pairKey("strength", "blood"):       0.94,
	pairKey("sword", "blade"):          0.95,
	pairKey("sword", "killing"):        0.96,
	pairKey("refinement", "pill"):      0.97,
	pairKey("wood", "fire"):            0.95,
	pairKey("fire", "earth"):           0.95,
	pairKey("earth", "metal"):          0.95,
	pairKey("metal", "water"):          0.95,
	pairKey("water", "wood"):           0.95,
	pairKey("creation", "destruction"): 0.55,
	pairKey("life", "death"):           0.55,
}

// CleanPath normalizes a path string key to its canonical form.
func CleanPath(path string) string {
	clean := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(path)), " path")
	clean = strings.TrimSuffix(clean, "_path")
	switch clean {
	case "kill", "killing":
		return "killing"
	}
	return clean
}

// GetCompatibility returns the compatibility score between two paths (0.0 to 1.0).
func GetCompatibility(pathA, pathB string) float64 {
	a := CleanPath(pathA)
	b := CleanPath(pathB)
	if a == b {
		return 1.0
	}
	if val, ok := pairOverrides[pairKey(a, b)]; ok {
		return val
	}
	grpA, okA := PathToDaoGroup[a]
	grpB, okB := PathToDaoGroup[b]
	if !okA || !okB {
		return 0.85 // default safe fallback
	}
	if matrixA, ok := groupCompatibilityMatrix[grpA]; ok {
		if comp, ok := matrixA[grpB]; ok {
			return comp
		}
	}
	return 0.85
}

// DaoAmplification computes the multiplier for Dao Marks.
//
// Multiplier = 1.0 + marks / 100.0 (Mỗi 100 Đạo Ngân = +1.0x / +100% uy lực)
func DaoAmplification(marks int64) float64 {
	if marks <= 0 {
		return 1.0
	}
	return 1.0 + float64(marks)/100.0
}

// RealmControlForLevel returns the conflict reduction percentage for an attainment level (0 to 4).
//
// Level 0 (Ordinary):            0%
// Level 1 (Master):             10%
// Level 2 (Grandmaster):        25%
// Level 3 (Great Grandmaster):  45%
// Level 4 (Supreme Grandmaster): 100% (hoá giải tuyệt đối cục bộ)
func RealmControlForLevel(level int) float64 {
	switch level {
	case 1:
		return 0.10
	case 2:
		return 0.25
	case 3:
		return 0.45
	case 4:
		return 1.00 // Vô Thượng Đại Tông Sư: hoá giải 100% xung đột cục bộ
	}
	return 0.0
}

// RealmEfficiencyForLevel returns the Dao Mark usage efficiency multiplier for an attainment level (0 to 4).
//
// Level 0: 1.00
// Level 1: 1.05
// Level 2: 1.12
// Level 3: 1.25
// Level 4: 1.45
func RealmEfficiencyForLevel(level int) float64 {
	switch level {
	case 1:
		return 1.05
	case 2:
		return 1.12
	case 3:
		return 1.25
	case 4:
		return 1.45
	}
	return 1.00
}

// effectivePathCount computes N_eff (spec §4.2) for a casting path p given the
// active foreign marks and p's attainment level. Compatibility pairs contribute
// 2*(1-comp), halved for well-matched paths once the caster reaches Tông Sư (L2+).
func effectivePathCount(p string, active map[string]int64, level int) float64 {
	nEff := 1.0
	for q, amt := range active {
		if q == p || amt <= 0 {
			continue
		}
		comp := GetCompatibility(p, q)
		discount := 1.0
		if level >= 2 && comp >= 0.90 {
			discount = 0.50
		}
		nEff += 2.0 * (1.0 - comp) * discount
	}
	return nEff
}

// MultiPathFactor computes F_Multi (spec §2.2/§4.2): a progressive penalty that
// amplifies raw conflict as the number of irreconcilable paths grows.
func MultiPathFactor(nEff, purity float64) float64 {
	if nEff <= 1.0 {
		return 1.0
	}
	return 1.0 + math.Pow(nEff-1.0, 1.3)*(1.0-purity)*(1.0-purity)*4.0
}

// ManaTaxMultiplier returns the Tiên Nguyên cost multiplier for a cast (spec §3.1).
func ManaTaxMultiplier(conflictMult float64) float64 {
	if conflictMult >= 1.0 {
		return 1.0
	}
	return 1.0 + (1.0-conflictMult)*1.5
}

// CastFailureChance returns the fizzle probability for a cast given its conflict
// multiplier (spec §3.2). Zero at conflictMult >= 0.70, capped at 0.25.
func CastFailureChance(conflictMult float64) float64 {
	if conflictMult >= 0.70 {
		return 0.0
	}
	c := (0.70 - conflictMult) / (0.70 - 0.08) * 0.25
	if c > 0.25 {
		c = 0.25
	}
	return c
}

// BackfireChance returns the internal-wound backfire probability (spec §3.3).
// Zero at conflictMult >= 0.60, capped at 0.30.
func BackfireChance(conflictMult float64) float64 {
	if conflictMult >= 0.60 {
		return 0.0
	}
	c := (0.60 - conflictMult) / (0.60 - 0.08) * 0.30
	if c > 0.30 {
		c = 0.30
	}
	return c
}

// DaoPurity returns the dominant path marks, total marks, and the purity ratio
// (dominant / total) for a Dao Mark map. Purity is 0 for an empty map.
func DaoPurity(marks map[string]int64) (dominant, total int64, purity float64) {
	for _, amt := range marks {
		if amt <= 0 {
			continue
		}
		total += amt
		if amt > dominant {
			dominant = amt
		}
	}
	if total > 0 {
		purity = float64(dominant) / float64(total)
	}
	return dominant, total, purity
}

// EcologyPurityFactor returns the aperture ecology production multiplier from
// purity P (spec §5.1): 1.0 at P >= 0.70, degrading to a floor of 0.25.
func EcologyPurityFactor(purity float64) float64 {
	if purity >= 0.70 {
		return 1.0
	}
	f := 1.0 - (0.70-purity)*1.25
	if f < 0.25 {
		f = 0.25
	}
	return f
}

// EffectivePathCount returns N_eff relative to the aperture's dominant path
// (spec §4.2). Used to gate the Chaotic/Hybrid tribulation.
func EffectivePathCount(marks map[string]int64, levels map[string]int) float64 {
	_, total, _ := DaoPurity(marks)
	grace := int64(1000)
	if pct := int64(float64(total) * 0.02); pct > grace {
		grace = pct
	}
	active := make(map[string]int64)
	dominant := ""
	for k, v := range marks {
		if v <= 0 {
			continue
		}
		clean := CleanPath(k)
		active[clean] += v
	}
	var maxAmt int64
	for path, amount := range active {
		if amount > maxAmt || (amount == maxAmt && (dominant == "" || path < dominant)) {
			maxAmt = amount
			dominant = path
		}
	}
	for k, v := range active {
		if v <= grace {
			delete(active, k)
		}
	}
	return effectivePathCount(dominant, active, PathMasteryLevel(levels[dominant]))
}

// IsChaoticTribulation reports whether an aperture's Dao Mark composition forces
// the Chaotic/Hybrid tribulation (spec §5.2): purity P < 0.60 and N_eff >= 2.5.
func IsChaoticTribulation(marks map[string]int64, levels map[string]int) bool {
	_, total, purity := DaoPurity(marks)
	if total == 0 || purity >= 0.60 {
		return false
	}
	return EffectivePathCount(marks, levels) >= 2.5
}

// PathCombatModifiers contains the precomputed offensive and defensive modifiers for a single path.
type PathCombatModifiers struct {
	Path                 string  `json:"path"`
	DaoMarks             int64   `json:"dao_marks"`
	DaoAmplification     float64 `json:"dao_amplification"`
	AttainmentLevel      int     `json:"attainment_level"`
	AttainmentName       string  `json:"attainment_name"`
	RealmControl         float64 `json:"realm_control"`
	RealmEfficiency      float64 `json:"realm_efficiency"`
	InternalConflictMult float64 `json:"internal_conflict_mult"`
	NaturalResistance    float64 `json:"natural_resistance"`
	Purity               float64 `json:"purity"`
	EffectivePathCount   float64 `json:"effective_path_count"`
	MultiPathFactor      float64 `json:"multi_path_factor"`
}

// DaoCombatProfile contains the complete precomputed combat profile for a player or monster.
type DaoCombatProfile struct {
	TotalDaoMarks        int64                          `json:"total_dao_marks"`
	GraceThreshold       int64                          `json:"grace_threshold"`
	HasNonConflictingDao bool                           `json:"has_non_conflicting_dao"`
	ConflictReductionPct float64                        `json:"conflict_reduction_pct"`
	DominantPath         string                         `json:"dominant_path"`
	Paths                map[string]PathCombatModifiers `json:"paths"`
}

// BuildDaoCombatProfile constructs a fully precomputed DaoCombatProfile.
// Precomputation takes < 5 microseconds and allows O(1) lookups during combat simulations.
func BuildDaoCombatProfile(
	daoMarks map[string]int64,
	pathMasteries map[string]int,
	hasNonConflictingDao bool,
	daoBonus map[string]float64,
) *DaoCombatProfile {
	return BuildDaoCombatProfileWithReduction(daoMarks, pathMasteries, hasNonConflictingDao, 0, daoBonus)
}

// BuildDaoCombatProfileWithReduction builds a combat profile that applies a
// diminishing-return reduction to internal Dao conflict. hasNonConflictingDao
// is reserved for permanent body traits and also disables natural defense.
func BuildDaoCombatProfileWithReduction(
	daoMarks map[string]int64,
	pathMasteries map[string]int,
	hasNonConflictingDao bool,
	conflictReductionPct float64,
	daoBonus map[string]float64,
) *DaoCombatProfile {
	if hasNonConflictingDao {
		// Permanent body traits are the only source of complete immunity.
		conflictReductionPct = 100
	} else if conflictReductionPct < 0 {
		conflictReductionPct = 0
	} else if conflictReductionPct > 90 {
		conflictReductionPct = 90
	}
	var totalMarks int64
	var maxMarks int64
	var dominantPath string

	cleanDaoMarks := make(map[string]int64, len(daoMarks))
	for p, amt := range daoMarks {
		if amt <= 0 {
			continue
		}
		clean := CleanPath(p)
		cleanDaoMarks[clean] += amt
		totalMarks += amt
	}
	for path, amount := range cleanDaoMarks {
		if amount > maxMarks || (amount == maxMarks && (dominantPath == "" || path < dominantPath)) {
			maxMarks = amount
			dominantPath = path
		}
	}

	// Grace threshold: foreign marks <= max(1000, 2% total) are ignored in conflict calculation
	graceThreshold := int64(1000)
	if pct := int64(float64(totalMarks) * 0.02); pct > graceThreshold {
		graceThreshold = pct
	}

	// Active foreign marks for conflict calculation
	activeForeign := make(map[string]int64)
	for p, amt := range cleanDaoMarks {
		if amt > graceThreshold {
			activeForeign[p] = amt
		}
	}

	profile := &DaoCombatProfile{
		TotalDaoMarks:        totalMarks,
		GraceThreshold:       graceThreshold,
		HasNonConflictingDao: hasNonConflictingDao,
		ConflictReductionPct: conflictReductionPct,
		DominantPath:         dominantPath,
		Paths:                make(map[string]PathCombatModifiers, len(CanonicalPaths)),
	}

	// buildMod precomputes the offensive/defensive factors for one casting path.
	buildMod := func(p string, amt int64, points int, bonus float64) PathCombatModifiers {
		lvl := PathMasteryLevel(points)
		realmCtrl := RealmControlForLevel(lvl)
		realmEff := RealmEfficiencyForLevel(lvl)
		daoAmp := DaoAmplification(amt) + bonus

		purity := 0.0
		if totalMarks > 0 {
			purity = float64(amt) / float64(totalMarks)
		}

		// Raw offensive conflict against active foreign marks (spec §2.3/§4.2).
		var rawConflict float64
		for otherP, otherAmt := range activeForeign {
			if otherP == p {
				continue
			}
			rawConflict += float64(otherAmt) * (1.0 - GetCompatibility(p, otherP))
		}
		nEff := effectivePathCount(p, activeForeign, lvl)
		fMulti := MultiPathFactor(nEff, purity)
		effConflict := rawConflict * fMulti * (1.0 - realmCtrl) * (1.0 - conflictReductionPct/100.0)
		denominator := float64(amt) + math.Max(1000.0, 0.1*float64(totalMarks))
		conflictRatio := 0.0
		if denominator > 0 {
			conflictRatio = effConflict / denominator
		}
		// Skill sensitivity 0.8 as standard, K = 2.0; floor 0.08 (max 92% penalty).
		conflictMult := 1.0 / (1.0 + conflictRatio*0.8*2.0)
		if conflictMult < 0.08 {
			conflictMult = 0.08
		}

		// Natural defensive resistance when attacked by path p.
		var defConflict float64
		for defP, defAmt := range cleanDaoMarks {
			if defAmt <= 0 {
				continue
			}
			defConflict += float64(defAmt) * (1.0 - GetCompatibility(p, defP))
		}
		natResist := 0.0
		if !hasNonConflictingDao && defConflict > 0 {
			natResist = 0.30 * (defConflict / (defConflict + 50000.0))
		}

		return PathCombatModifiers{
			Path:                 p,
			DaoMarks:             amt,
			DaoAmplification:     daoAmp,
			AttainmentLevel:      lvl,
			AttainmentName:       PathMasteryLevelName(lvl),
			RealmControl:         realmCtrl,
			RealmEfficiency:      realmEff,
			InternalConflictMult: conflictMult,
			NaturalResistance:    natResist,
			Purity:               purity,
			EffectivePathCount:   nEff,
			MultiPathFactor:      fMulti,
		}
	}

	for _, p := range CanonicalPaths {
		amt := cleanDaoMarks[p]
		points := 0
		if pathMasteries != nil {
			points = pathMasteries[p]
			if points == 0 && p == "killing" {
				points = pathMasteries["kill"]
			}
		}
		bonus := 0.0
		if daoBonus != nil {
			bonus = daoBonus[p]
			if bonus == 0 && p == "killing" {
				bonus = daoBonus["kill"]
			}
		}
		profile.Paths[p] = buildMod(p, amt, points, bonus)
	}

	// Also precompute for any additional paths present in cleanDaoMarks (e.g. death, creation, destruction, investigation)
	for p, amt := range cleanDaoMarks {
		if _, exists := profile.Paths[p]; exists {
			continue
		}
		points := 0
		if pathMasteries != nil {
			points = pathMasteries[p]
		}
		bonus := 0.0
		if daoBonus != nil {
			bonus = daoBonus[p]
		}
		profile.Paths[p] = buildMod(p, amt, points, bonus)
	}

	return profile
}

// GetOffensiveFactors returns the (daoAmp, realmEff, conflictMult) for an attack path.
// Lookup is O(1).
func (p *DaoCombatProfile) GetOffensiveFactors(path string, sensitivity ...float64) (daoAmp float64, realmEff float64, conflictMult float64) {
	if p == nil {
		return 1.0, 1.0, 1.0
	}
	clean := CleanPath(path)
	mod, ok := p.Paths[clean]
	if !ok {
		return 1.0, 1.0, 1.0
	}
	if p.HasNonConflictingDao {
		return mod.DaoAmplification, mod.RealmEfficiency, 1.00
	}
	if len(sensitivity) > 0 && sensitivity[0] != 0.8 && sensitivity[0] > 0 {
		// Custom sensitivity scale
		sens := sensitivity[0]
		rawMult := mod.InternalConflictMult
		// Adjust from base 0.8 sensitivity
		if rawMult < 1.0 {
			penalty := (1.0 - rawMult) * (sens / 0.8)
			customMult := 1.0 - penalty
			if customMult < 0.08 {
				customMult = 0.08
			}
			return mod.DaoAmplification, mod.RealmEfficiency, customMult
		}
	}
	return mod.DaoAmplification, mod.RealmEfficiency, mod.InternalConflictMult
}

// GetDefensiveResistance returns the natural damage reduction percentage (0.0 to 0.30)
// when defending against an incoming attack of attackPath. Lookup is O(1).
func (p *DaoCombatProfile) GetDefensiveResistance(attackPath string) float64 {
	if p == nil || p.HasNonConflictingDao {
		return 0.0
	}
	clean := CleanPath(attackPath)
	if mod, ok := p.Paths[clean]; ok {
		return mod.NaturalResistance
	}
	return 0.0
}
