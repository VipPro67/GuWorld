package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func GetPassiveStatsHandler(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Passive stats are a read-only projection. Do not invoke lazy hunger or
	// persistence here: polling this endpoint must never hold a player row lock.
	player, err := PlayerSvc.ProjectStatus(db.DB, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	result := GetPassiveStats(&player)
	c.JSON(http.StatusOK, result)
}

// ── Đạo Ngân (Dao Silver) Calculation ───────────────────────────────────────

// DaoNganPathDetail describes the contribution of one dao path to the đạo ngân score.
type DaoNganPathDetail struct {
	Path                       string  `json:"path"`                          // e.g. "strength", "fire", "sword"
	Amount                     int64   `json:"amount"`                        // raw dao mark count
	Multiplier                 float64 `json:"multiplier"`                    // effective combat multiplier: (1 + (amp-1)*eff)*conflict
	BaseMultiplier             float64 `json:"base_multiplier"`               // base dao mark amplification: 1 + amount/100
	Power                      float64 `json:"power"`                         // effective power contribution
	AttainmentLevel            int     `json:"attainment_level"`              // 0-4
	AttainmentName             string  `json:"attainment_name"`               // e.g. "Đại Tông Sư"
	RealmControlPct            float64 `json:"realm_control_pct"`             // e.g. 45.0 (%)
	RealmEfficiency            float64 `json:"realm_efficiency"`              // e.g. 1.25 (×)
	InternalConflictPenaltyPct float64 `json:"internal_conflict_penalty_pct"` // e.g. 16.5 (%)
	NaturalResistancePct       float64 `json:"natural_resistance_pct"`        // e.g. 9.3 (%)
	PurityPct                  float64 `json:"purity_pct"`                    // e.g. 71.4 (%)
}

// DaoNganResult is the full đạo ngân calculation response.
type DaoNganResult struct {
	TotalDaoNgan            float64             `json:"total_dao_ngan"`          // composite đạo ngân score
	TotalMarks              int64               `json:"total_marks"`             // sum of all dao marks
	UniquePathCount         int                 `json:"unique_path_count"`       // number of paths with marks
	AverageMultiplier       float64             `json:"average_multiplier"`      // mean multiplier across paths
	MaxMultiplier           float64             `json:"max_multiplier"`          // highest single-path multiplier
	MaxPath                 string              `json:"max_path"`                // path with highest multiplier
	DominantPath            string              `json:"dominant_path"`           // path with most dao marks
	GraceThreshold          int64               `json:"grace_threshold"`         // safe foreign mark threshold
	ConflictFloor           int64               `json:"conflict_floor"`          // total marks floor below which dao conflict is ignored
	BelowConflictFloor      bool                `json:"below_conflict_floor"`    // true when total marks <= ConflictFloor (conflict-free)
	HasNonConflictingDao    bool                `json:"has_non_conflicting_dao"` // non-conflicting dao marks trait
	DaoConflictReductionPct float64             `json:"dao_conflict_reduction_pct"`
	Paths                   []DaoNganPathDetail `json:"paths"` // per-path breakdown
}

// CalculateDaoNganDetailed computes the composite "đạo ngân" (dao silver) score
// and detailed combat breakdown using DaoCombatProfile.
func CalculateDaoNganDetailed(
	daoMarks models.DaoMarksMap,
	pathMasteries map[string]int,
	hasNonConflicting bool,
	daoBonus ...map[string]float64,
) DaoNganResult {
	bonus := map[string]float64{}
	if len(daoBonus) > 0 {
		bonus = daoBonus[0]
	}
	return CalculateDaoNganDetailedWithReduction(daoMarks, pathMasteries, hasNonConflicting, 0, bonus)
}

func CalculateDaoNganDetailedWithReduction(
	daoMarks models.DaoMarksMap,
	pathMasteries map[string]int,
	hasNonConflicting bool,
	conflictReductionPct float64,
	daoBonus map[string]float64,
) DaoNganResult {
	result := DaoNganResult{
		Paths: make([]DaoNganPathDetail, 0),
	}
	if daoMarks == nil {
		return result
	}
	profile := gamefx.BuildDaoCombatProfileWithReduction(daoMarks, pathMasteries, hasNonConflicting, conflictReductionPct, daoBonus)
	result.DominantPath = profile.DominantPath
	result.GraceThreshold = profile.GraceThreshold
	result.ConflictFloor = gamefx.DaoConflictTotalFloor
	result.BelowConflictFloor = profile.BelowConflictFloor
	result.HasNonConflictingDao = profile.HasNonConflictingDao
	result.DaoConflictReductionPct = profile.ConflictReductionPct

	var totalPower float64
	var totalMarks int64
	var maxMult float64
	var maxPath string
	var sumMult float64

	aggregatedMarks := make(map[string]int64)
	for path, amount := range daoMarks {
		if amount <= 0 {
			continue
		}
		clean := gamefx.CleanPath(path)
		aggregatedMarks[clean] += amount
	}

	for clean, amount := range aggregatedMarks {
		mod, ok := profile.Paths[clean]
		baseMult := gamefx.DaoAmplification(amount) + daoBonus[clean]
		effectiveMult := baseMult
		if ok {
			baseMult = mod.DaoAmplification
			daoAmp, realmEff, conflictMult := profile.GetOffensiveFactors(clean)
			effectiveMult = (1.0 + (daoAmp-1.0)*realmEff) * conflictMult
		}

		power := float64(amount) * effectiveMult

		detail := DaoNganPathDetail{
			Path:           clean,
			Amount:         amount,
			Multiplier:     effectiveMult,
			BaseMultiplier: baseMult,
			Power:          power,
		}
		if ok {
			detail.AttainmentLevel = mod.AttainmentLevel
			detail.AttainmentName = mod.AttainmentName
			detail.RealmControlPct = mod.RealmControl * 100.0
			detail.RealmEfficiency = mod.RealmEfficiency
			detail.InternalConflictPenaltyPct = (1.0 - mod.InternalConflictMult) * 100.0
			detail.NaturalResistancePct = mod.NaturalResistance * 100.0
			detail.PurityPct = mod.Purity * 100.0
		}

		result.Paths = append(result.Paths, detail)

		totalPower += power
		totalMarks += amount
		sumMult += effectiveMult
		result.UniquePathCount++

		if effectiveMult > maxMult {
			maxMult = effectiveMult
			maxPath = clean
		}
	}

	// Sort paths by power descending
	sort.Slice(result.Paths, func(i, j int) bool {
		return result.Paths[i].Power > result.Paths[j].Power
	})

	result.TotalDaoNgan = totalPower
	result.TotalMarks = totalMarks
	result.MaxMultiplier = maxMult
	result.MaxPath = maxPath
	if result.UniquePathCount > 0 {
		result.AverageMultiplier = sumMult / float64(result.UniquePathCount)
	}

	return result
}

// CalculateDaoNgan computes the composite "đạo ngân" (dao silver) score
// from all of a player's dao marks (backward-compatible wrapper).
func CalculateDaoNgan(daoMarks models.DaoMarksMap, daoBonus ...map[string]float64) DaoNganResult {
	return CalculateDaoNganDetailed(daoMarks, nil, false, daoBonus...)
}

// GetDaoNganHandler returns the đạo ngân calculation for the authenticated player.
func GetDaoNganHandler(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var daoJSON []byte
	err := db.DB.QueryRowContext(c.Request.Context(),
		`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID,
	).Scan(&daoJSON)
	if err == sql.ErrNoRows {
		daoJSON = []byte("{}")
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Dao Marks"})
		return
	}

	daoMap := make(models.DaoMarksMap)
	if len(daoJSON) > 0 {
		_ = json.Unmarshal(daoJSON, &daoMap)
	}

	var constitution, race string
	var pathMasteriesJSON []byte
	var bodyMods models.BodyModsList
	var guInventory models.GuInventory
	err = db.DB.QueryRowContext(c.Request.Context(),
		`SELECT COALESCE(p.constitution, ''), COALESCE(p.race, 'human'), COALESCE(p.path_masteries, '{}'::jsonb), COALESCE(p.body_mods, '[]'::jsonb), `+db.GuInventoryAggregate("p.id")+` FROM players p WHERE p.id = $1`, playerID,
	).Scan(&constitution, &race, &pathMasteriesJSON, &bodyMods, &guInventory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Dao effects"})
		return
	}

	pathMasteries := make(models.PathMasteriesMap)
	if len(pathMasteriesJSON) > 0 {
		_ = json.Unmarshal(pathMasteriesJSON, &pathMasteries)
	}

	playerModel := &models.Player{
		CharacterState: models.CharacterState{
			Constitution: constitution,
			Race:         race,
		},
		BodyState: models.BodyState{BodyMods: bodyMods},
	}
	playerModel.PathMasteries = pathMasteries
	playerModel.GuInventory = guInventory
	charFX := gamefx.Get(playerModel)

	result := CalculateDaoNganDetailedWithReduction(daoMap, pathMasteries, charFX.NonConflictingDaoMarks, charFX.DaoConflictReductionPct, charFX.DaoNganBonus)
	c.JSON(http.StatusOK, result)
}
