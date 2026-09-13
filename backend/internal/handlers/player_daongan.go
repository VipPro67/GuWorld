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
	Path       string  `json:"path"`       // e.g. "strength", "fire", "sword"
	Amount     int64   `json:"amount"`     // raw dao mark count
	Multiplier float64 `json:"multiplier"` // combat multiplier from this path
	Power      float64 `json:"power"`      // effective power contribution
}

// DaoNganResult is the full đạo ngân calculation response.
type DaoNganResult struct {
	TotalDaoNgan      float64             `json:"total_dao_ngan"`     // composite đạo ngân score
	TotalMarks        int64               `json:"total_marks"`        // sum of all dao marks
	UniquePathCount   int                 `json:"unique_path_count"`  // number of paths with marks
	AverageMultiplier float64             `json:"average_multiplier"` // mean multiplier across paths
	MaxMultiplier     float64             `json:"max_multiplier"`     // highest single-path multiplier
	MaxPath           string              `json:"max_path"`           // path with highest multiplier
	Paths             []DaoNganPathDetail `json:"paths"`              // per-path breakdown
}

// CalculateDaoNgan computes the composite "đạo ngân" (dao silver) score
// from all of a player's dao marks.
//
// Đạo Ngân represents the cumulative cultivation attainment across all dao paths.
// The formula sums each path's effective power:
//
//	power(marks) = marks * multiplier(marks)
//
// where multiplier follows the standard combat formula:
//
//	multiplier = 1.0 + marks/100
//
// TotalDaoNgan = Σ power(path) for all paths with marks > 0
func CalculateDaoNgan(daoMarks models.DaoMarksMap, daoBonus ...map[string]float64) DaoNganResult {
	result := DaoNganResult{
		Paths: make([]DaoNganPathDetail, 0),
	}

	if daoMarks == nil {
		return result
	}

	var totalPower float64
	var totalMarks int64
	var maxMult float64
	var maxPath string
	var sumMult float64

	bonus := map[string]float64{}
	if len(daoBonus) > 0 {
		bonus = daoBonus[0]
	}

	for path, amount := range daoMarks {
		if amount <= 0 {
			continue
		}

		mult := gamefx.DaoPathMultiplier(amount, bonus[path])

		power := float64(amount) * mult

		result.Paths = append(result.Paths, DaoNganPathDetail{
			Path:       path,
			Amount:     amount,
			Multiplier: mult,
			Power:      power,
		})

		totalPower += power
		totalMarks += amount
		sumMult += mult
		result.UniquePathCount++

		if mult > maxMult {
			maxMult = mult
			maxPath = path
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

	var constitution string
	_ = db.DB.QueryRowContext(c.Request.Context(),
		`SELECT COALESCE(constitution, '') FROM players WHERE id = $1`, playerID,
	).Scan(&constitution)

	daoBonus := map[string]float64{}
	if constitution != "" {
		daoBonus = gamefx.Get(&models.Player{CharacterState: models.CharacterState{Constitution: constitution}}).DaoNganBonus
	}

	result := CalculateDaoNgan(daoMap, daoBonus)
	c.JSON(http.StatusOK, result)
}

// flushGrowthOnUnequip converts the pending equipped-time of a Gu with a growth
// effect into permanent boosts when the Gu is unequipped (manually or by the
// equipped-list cleanup in EquipGu). No-op when the Gu has no growth effect.
