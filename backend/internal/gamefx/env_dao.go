package gamefx

import (
	"math"
	"sort"
)

// EnvDaoConfig holds all tunable thresholds for environmental Dao Marks and
// Venerable Dao Domains. Keeping them in a single struct makes balancing easy.
type EnvDaoConfig struct {
	// Natural Dao Marks (unowned environment)
	NaturalSaturationK   float64 // M / (M + K) saturation midpoint
	NaturalMaxBoost      float64 // e.g. 0.35 -> 1.35x max for aligned path
	NaturalMaxPenalty    float64 // e.g. 0.25 -> 0.75x floor for conflicting path
	MortalPenaltyFactor  float64 // Rank 1-5 penalty scale (e.g. 0.5 -> half penalty)
	MortalMinFloor       float64 // Rank 1-5 absolute floor (e.g. 0.85x)

	// Dao Domain (Venerable-refined territory, scaled by control_pct 0..100)
	DomainActivationPct     float64 // Minimum control_pct for domain effects (e.g. 25)
	DomainOwnerMaxMult      float64 // Overlord ("bá chủ") multiplier at 100% (e.g. 2.20x)
	DomainAllyMaxMult       float64 // Ally/sect multiplier at 100% (e.g. 1.25x)
	DomainEnemyFloor        float64 // Enemy off-path floor at 100% (e.g. 0.45x)
	DomainEnemySamePathMin  float64 // Enemy same-path will suppression floor (e.g. 0.85x)

	// Stability & Mana Tax penalties when envMult < StabilityPenaltyThreshold
	StabilityPenaltyThreshold float64 // e.g. 0.90
	MaxManaTaxPenalty         float64 // e.g. +0.30 (1.30x)
	MaxSuccessPenaltyPct      float64 // e.g. 0.15 (-15% percentage points)
	SuccessChanceFloor        float64 // e.g. 0.50 (never pull >=50% skills below 50%)
	MaxDomainBackfireBonus    float64 // e.g. +0.10 (+10% backfire in enemy domain)

	// Aperture Hạ Khiếu Qi multipliers at 100% control_pct
	DomainQiOwnerBonus float64 // e.g. +0.50 -> 1.50x
	DomainQiAllyBonus  float64 // e.g. +0.30 -> 1.30x
	DomainQiEnemyFloor float64 // e.g. 0.65x

	// Venerable refinement rules
	MaxConcurrentDomains int
}

// DefaultEnvDaoConfig returns the default production-tuned configuration.
var DefaultEnvDaoConfig = EnvDaoConfig{
	NaturalSaturationK:        15000.0,
	NaturalMaxBoost:           0.35,
	NaturalMaxPenalty:         0.25,
	MortalPenaltyFactor:       0.50,
	MortalMinFloor:            0.85,
	DomainActivationPct:       25.0,
	DomainOwnerMaxMult:        2.20,
	DomainAllyMaxMult:         1.25,
	DomainEnemyFloor:          0.45,
	DomainEnemySamePathMin:    0.85,
	StabilityPenaltyThreshold: 0.90,
	MaxManaTaxPenalty:         0.30,
	MaxSuccessPenaltyPct:      0.15,
	SuccessChanceFloor:        0.50,
	MaxDomainBackfireBonus:    0.10,
	DomainQiOwnerBonus:        0.50,
	DomainQiAllyBonus:         0.30,
	DomainQiEnemyFloor:        0.65,
	MaxConcurrentDomains:      5,
}

// EnvCombatContext describes the environmental Dao Marks & optional Dao Domain
// governing a combat instance.
type EnvCombatContext struct {
	RegionID          string             `json:"region_id,omitempty"`
	RegionName        string             `json:"region_name,omitempty"`
	IsAperture        bool               `json:"is_aperture,omitempty"`
	NaturalDaoMarks   map[string]float64 `json:"natural_dao_marks,omitempty"`
	DomainOwnerID     string             `json:"domain_owner_id,omitempty"`
	DomainOwnerName   string             `json:"domain_owner_name,omitempty"`
	DomainPath        string             `json:"domain_path,omitempty"`
	DomainControlPct  float64            `json:"domain_control_pct,omitempty"`
}

// EnvSkillModifiers is the computed effect of the environment on one action.
type EnvSkillModifiers struct {
	PowerMult          float64 // Multiplier for damage / shield / healing
	ManaTaxMult        float64 // Additional multiplier on mana cost (>= 1.0)
	SuccessChanceDelta float64 // Additive delta on killer move success chance (<= 0)
	BackfireBonus      float64 // Additive backfire chance in enemy domain (>= 0)
	Reason             string
}

// DominantEnvPath returns the highest-mark path and its mark value.
func DominantEnvPath(marks map[string]float64) (string, float64) {
	if len(marks) == 0 {
		return "", 0
	}
	keys := make([]string, 0, len(marks))
	for k := range marks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	bestPath := ""
	bestVal := 0.0
	for _, k := range keys {
		if marks[k] > bestVal {
			bestVal = marks[k]
			bestPath = CleanPath(k)
		}
	}
	return bestPath, bestVal
}

// ResolveEnvSkillModifiers computes environmental modifiers for a single action.
func ResolveEnvSkillModifiers(
	cfg EnvDaoConfig,
	env *EnvCombatContext,
	actionPath string,
	actorID string,
	actorRank int,
	isAllyOfOwner bool,
	hasNonConflictingDao bool,
) EnvSkillModifiers {
	res := EnvSkillModifiers{PowerMult: 1.0, ManaTaxMult: 1.0}
	if env == nil {
		return res
	}
	cleanAct := CleanPath(actionPath)
	if cleanAct == "" {
		cleanAct = "strength"
	}

	// 1. Dao Domain active (Venerable-refined region or private aperture owner)
	if env.DomainOwnerID != "" && env.DomainControlPct >= cfg.DomainActivationPct {
		pct := math.Max(0, math.Min(1.0, env.DomainControlPct/100.0))
		domPath := CleanPath(env.DomainPath)
		if actorID != "" && actorID == env.DomainOwnerID {
			if domPath == "" || cleanAct == domPath {
				res.PowerMult = 1.0 + (cfg.DomainOwnerMaxMult-1.0)*pct
				res.Reason = "Đạo Vực (Bá Chủ)"
			} else {
				res.Reason = "Đạo Vực (Miễn phạt chủ nhân)"
			}
			return res
		}
		if isAllyOfOwner {
			if domPath != "" && cleanAct == domPath {
				res.PowerMult = 1.0 + (cfg.DomainAllyMaxMult-1.0)*pct
				res.Reason = "Đạo Vực (Đồng minh)"
			} else {
				res.Reason = "Đạo Vực (Miễn phạt đồng minh)"
			}
			return res
		}
		// Enemy inside Dao Domain
		if domPath != "" && cleanAct == domPath {
			res.PowerMult = 1.0 - (1.0-cfg.DomainEnemySamePathMin)*pct
			res.Reason = "Đạo Vực (Áp chế ý chí)"
		} else {
			floor := cfg.DomainEnemyFloor
			if actorRank > 0 && actorRank <= 5 {
				floor = math.Max(floor, cfg.MortalMinFloor)
			}
			res.PowerMult = 1.0 - (1.0-floor)*pct
			res.Reason = "Đạo Vực (Bài xích)"
		}
		applyStabilityPenalty(&res, cfg, actorRank, true, pct)
		return res
	}

	// 2. Natural Dao Marks (unowned environment)
	domPath, domMarks := DominantEnvPath(env.NaturalDaoMarks)
	if domPath == "" || domMarks <= 0 {
		return res
	}
	sat := domMarks / (domMarks + math.Max(1.0, cfg.NaturalSaturationK))
	if cleanAct == domPath {
		res.PowerMult = 1.0 + cfg.NaturalMaxBoost*sat
		res.Reason = "Đạo Ngân Tự Nhiên (Cộng hưởng)"
		return res
	}
	if hasNonConflictingDao {
		res.Reason = "Chí Tôn Tiên Thể (Miễn bài xích tự nhiên)"
		return res
	}
	compat := GetCompatibility(cleanAct, domPath)
	if compat < 0.85 {
		penalty := cfg.NaturalMaxPenalty * sat * ((0.85 - compat) / 0.85)
		if actorRank > 0 && actorRank <= 5 {
			penalty *= cfg.MortalPenaltyFactor
		}
		res.PowerMult = math.Max(1.0-cfg.NaturalMaxPenalty, 1.0-penalty)
		if actorRank > 0 && actorRank <= 5 && res.PowerMult < cfg.MortalMinFloor {
			res.PowerMult = cfg.MortalMinFloor
		}
		res.Reason = "Đạo Ngân Tự Nhiên (Xung khắc)"
		applyStabilityPenalty(&res, cfg, actorRank, false, sat)
	}
	return res
}

func applyStabilityPenalty(res *EnvSkillModifiers, cfg EnvDaoConfig, actorRank int, inDomain bool, intensity float64) {
	if actorRank > 0 && actorRank <= 5 {
		return
	}
	if res.PowerMult >= cfg.StabilityPenaltyThreshold {
		return
	}
	denom := math.Max(0.05, cfg.StabilityPenaltyThreshold-cfg.DomainEnemyFloor)
	severity := math.Min(1.0, (cfg.StabilityPenaltyThreshold-res.PowerMult)/denom)
	res.ManaTaxMult = 1.0 + cfg.MaxManaTaxPenalty*severity
	res.SuccessChanceDelta = -cfg.MaxSuccessPenaltyPct * severity
	if inDomain {
		res.BackfireBonus = cfg.MaxDomainBackfireBonus * severity * intensity
	}
}
