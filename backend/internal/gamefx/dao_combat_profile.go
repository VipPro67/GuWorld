package gamefx

import (
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
	"weapon": DaoGroupKineticWeapon, "killing": DaoGroupKineticWeapon, "kill": DaoGroupKineticWeapon,

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
	pairKey("fire", "water"):   0.55,
	pairKey("fire", "ice"):     0.60,
	pairKey("light", "dark"):   0.55,
	pairKey("blood", "light"):  0.65,
	pairKey("poison", "food"):  0.60,

	// Tương sinh & Đồng nguyên
	pairKey("space", "time"):          0.97,
	pairKey("wisdom", "information"):  0.98,
	pairKey("wisdom", "soul"):         0.95,
	pairKey("strength", "bone"):       0.96,
	pairKey("strength", "blood"):      0.94,
	pairKey("sword", "blade"):         0.95,
	pairKey("sword", "killing"):       0.96,
	pairKey("refinement", "pill"):     0.97,
	pairKey("wood", "fire"):           0.95,
	pairKey("fire", "earth"):          0.95,
	pairKey("earth", "metal"):         0.95,
	pairKey("metal", "water"):         0.95,
	pairKey("water", "wood"):          0.95,
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
// Level 4 (Supreme Grandmaster): 80%
func RealmControlForLevel(level int) float64 {
	switch level {
	case 1:
		return 0.10
	case 2:
		return 0.25
	case 3:
		return 0.45
	case 4:
		return 0.80
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
}

// DaoCombatProfile contains the complete precomputed combat profile for a player or monster.
type DaoCombatProfile struct {
	TotalDaoMarks        int64                          `json:"total_dao_marks"`
	GraceThreshold       int64                          `json:"grace_threshold"`
	HasNonConflictingDao bool                           `json:"has_non_conflicting_dao"`
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
		if cleanDaoMarks[clean] > maxMarks {
			maxMarks = cleanDaoMarks[clean]
			dominantPath = clean
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
		DominantPath:         dominantPath,
		Paths:                make(map[string]PathCombatModifiers, len(CanonicalPaths)),
	}

	for _, p := range CanonicalPaths {
		amt := cleanDaoMarks[p]
		points := 0
		if pathMasteries != nil {
			points = pathMasteries[p]
			if points == 0 && p == "killing" {
				points = pathMasteries["kill"]
			} else if points == 0 && p == "kill" {
				points = pathMasteries["killing"]
			}
		}
		lvl := PathMasteryLevel(points)
		realmCtrl := RealmControlForLevel(lvl)
		realmEff := RealmEfficiencyForLevel(lvl)

		bonus := 0.0
		if daoBonus != nil {
			bonus = daoBonus[p]
			if bonus == 0 && p == "killing" {
				bonus = daoBonus["kill"]
			} else if bonus == 0 && p == "kill" {
				bonus = daoBonus["killing"]
			}
		}
		daoAmp := DaoAmplification(amt) + bonus

		// Calculate Raw Offensive Conflict against active foreign marks
		var rawConflict float64
		for otherP, otherAmt := range activeForeign {
			if otherP == p {
				continue
			}
			comp := GetCompatibility(p, otherP)
			rawConflict += float64(otherAmt) * (1.0 - comp)
		}
		effConflict := rawConflict * (1.0 - realmCtrl)
		conflictRatio := effConflict / (float64(amt) + 50000.0)

		// Skill sensitivity 0.8 as standard, K = 2.0
		conflictMult := 1.0 / (1.0 + conflictRatio*0.8*2.0)
		if conflictMult < 0.60 {
			conflictMult = 0.60 // Floor protection: max 40% penalty
		}

		// Calculate Natural Defensive Resistance when attacked by path p
		var defConflict float64
		for defP, defAmt := range cleanDaoMarks {
			if defAmt <= 0 {
				continue
			}
			comp := GetCompatibility(p, defP)
			defConflict += float64(defAmt) * (1.0 - comp)
		}
		natResist := 0.0
		if !hasNonConflictingDao && defConflict > 0 {
			natResist = 0.30 * (defConflict / (defConflict + 50000.0))
		}

		purity := 0.0
		if totalMarks > 0 {
			purity = float64(amt) / float64(totalMarks)
		}

		profile.Paths[p] = PathCombatModifiers{
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
		}
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
		lvl := PathMasteryLevel(points)
		realmCtrl := RealmControlForLevel(lvl)
		realmEff := RealmEfficiencyForLevel(lvl)

		bonus := 0.0
		if daoBonus != nil {
			bonus = daoBonus[p]
		}
		daoAmp := DaoAmplification(amt) + bonus

		var rawConflict float64
		for otherP, otherAmt := range activeForeign {
			if otherP == p {
				continue
			}
			comp := GetCompatibility(p, otherP)
			rawConflict += float64(otherAmt) * (1.0 - comp)
		}
		effConflict := rawConflict * (1.0 - realmCtrl)
		conflictRatio := effConflict / (float64(amt) + 50000.0)

		conflictMult := 1.0 / (1.0 + conflictRatio*0.8*2.0)
		if conflictMult < 0.60 {
			conflictMult = 0.60
		}

		var defConflict float64
		for defP, defAmt := range cleanDaoMarks {
			if defAmt <= 0 {
				continue
			}
			comp := GetCompatibility(p, defP)
			defConflict += float64(defAmt) * (1.0 - comp)
		}
		natResist := 0.0
		if !hasNonConflictingDao && defConflict > 0 {
			natResist = 0.30 * (defConflict / (defConflict + 50000.0))
		}

		purity := 0.0
		if totalMarks > 0 {
			purity = float64(amt) / float64(totalMarks)
		}

		profile.Paths[p] = PathCombatModifiers{
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
		}
	}

	// Mirror alias for killing <-> kill so direct lookups by either key succeed
	if mod, ok := profile.Paths["killing"]; ok {
		profile.Paths["kill"] = mod
	} else if mod, ok := profile.Paths["kill"]; ok {
		profile.Paths["killing"] = mod
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
			if customMult < 0.60 {
				customMult = 0.60
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
