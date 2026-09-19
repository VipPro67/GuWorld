package gamefx

import (
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// BaseRankPower returns base calculation power corresponding to player Rank
func BaseRankPower(rank int) float64 {
	switch rank {
	case 1:
		return 10.0
	case 2:
		return 20.0
	case 3:
		return 30.0
	case 4:
		return 40.0
	case 5:
		return 50.0
	case 6:
		return 100.0
	case 7:
		return 180.0
	case 8:
		return 300.0
	case 9:
		return 500.0
	default:
		if rank < 1 {
			return 10.0
		}
		return 500.0
	}
}

// GetPathAttainmentMultiplier returns multiplier based on highest attainment level among candidate paths
func GetPathAttainmentMultiplier(masteries models.PathMasteriesMap, paths ...string) float64 {
	highestLevel := 0
	for _, p := range paths {
		clean := strings.ToLower(strings.TrimSpace(p))
		clean = strings.TrimSuffix(clean, " path")
		for pathKey, pts := range masteries {
			normKey := strings.ToLower(strings.TrimSpace(pathKey))
			normKey = strings.TrimSuffix(normKey, " path")
			if strings.Contains(normKey, clean) || strings.Contains(clean, normKey) {
				lvl := PathMasteryLevel(pts)
				if lvl > highestLevel {
					highestLevel = lvl
				}
			}
		}
	}

	switch highestLevel {
	case 1:
		return 1.2 // Master (Đại Sư)
	case 2:
		return 1.5 // Grandmaster (Tông Sư)
	case 3:
		return 2.0 // Great Grandmaster (Đại Tông Sư)
	case 4:
		return 3.0 // Supreme Grandmaster (Vô Thượng)
	default:
		return 1.0 // Ordinary (Phàm Thai)
	}
}

// CalculateDeductionPower computes P_div for the divining player
func CalculateDeductionPower(
	viewer *models.Player,
	viewerGus []models.GuInstance,
	viewerDaoMarks models.DaoMarksMap,
	clues models.DivinationRequest,
	target *models.Player,
	resourceBonus float64,
) float64 {
	basePower := BaseRankPower(viewer.Rank)

	// Add Gu passive Investigation stats
	investigationSum := 0.0
	for _, gu := range viewerGus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok && spec.Effects.Passive != nil {
			investigationSum += spec.Effects.Passive.Investigation
		}
	}

	attainMult := GetPathAttainmentMultiplier(viewer.PathMasteries, "wisdom", "information", "investigation")

	daoBonus := 1.0
	if viewer.Rank >= 6 && viewerDaoMarks != nil {
		wMarks := viewerDaoMarks["wisdom"]
		iMarks := viewerDaoMarks["information"]
		daoBonus = 1.0 + float64(wMarks+iMarks)/1000.0
	}

	corePower := (basePower + investigationSum) * attainMult * daoBonus

	// Clue Bonuses
	clueBonus := 0.0
	if clues.TargetUsername != "" {
		if target != nil && strings.EqualFold(clues.TargetUsername, target.Username) {
			clueBonus += 20.0
		}
	} else {
		// Blind deduction penalty
		clueBonus -= 40.0
	}

	if clues.SuspectedSect != "" {
		clueBonus += 10.0
	}

	if clues.SuspectedRegion != "" {
		if target != nil && strings.EqualFold(clues.SuspectedRegion, target.CurrentRegion) {
			clueBonus += 15.0
		}
	}

	if clues.ItemClueID != "" {
		clueBonus += 30.0
	}

	// Resource investment bonus (capped at +25% of corePower)
	resBonus := resourceBonus
	if resBonus > corePower*0.25 {
		resBonus = corePower * 0.25
	}

	totalPower := corePower + clueBonus + resBonus
	if totalPower < 1.0 {
		totalPower = 1.0
	}
	return totalPower
}

// CalculateConcealmentDefense computes D_con for the target player
func CalculateConcealmentDefense(
	target *models.Player,
	targetGus []models.GuInstance,
	targetDaoMarks models.DaoMarksMap,
	viewerRegion string,
) float64 {
	baseDefense := BaseRankPower(target.Rank)

	disguiseSum := 0.0
	hasPerfectDisguise := false
	for _, gu := range targetGus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			if spec.Effects.Passive != nil {
				disguiseSum += spec.Effects.Passive.DisguiseLevel * 5.0
			}
			for _, mechanic := range spec.Effects.Other {
				if mechanic.ActionCode == "PERFECT_DISGUISE" {
					hasPerfectDisguise = true
				}
			}
		}
	}

	capabilityBonus := 0.0
	if hasPerfectDisguise {
		capabilityBonus += 80.0
	}

	attainMult := GetPathAttainmentMultiplier(target.PathMasteries, "dark", "transformation", "wisdom", "concealment")

	daoBonus := 1.0
	if target.Rank >= 6 && targetDaoMarks != nil {
		cMarks := targetDaoMarks["concealment"]
		dMarks := targetDaoMarks["dark"]
		tMarks := targetDaoMarks["transformation"]
		daoBonus = 1.0 + float64(cMarks+dMarks+tMarks)/1000.0
	}

	// Distance penalty (Giới bích / Hai thiên)
	distancePenalty := 10.0 // Same region
	if viewerRegion != "" && target.CurrentRegion != "" && !strings.EqualFold(viewerRegion, target.CurrentRegion) {
		distancePenalty = 35.0 // Cross-region boundary
	}

	totalDefense := (baseDefense + disguiseSum + capabilityBonus) * attainMult * daoBonus + distancePenalty
	return totalDefense
}

// EvaluateDivination computes delta score and categorizes outcome
func EvaluateDivination(
	viewer *models.Player,
	target *models.Player,
	viewerGus []models.GuInstance,
	targetGus []models.GuInstance,
	viewerDao models.DaoMarksMap,
	targetDao models.DaoMarksMap,
	clues models.DivinationRequest,
	resourceBonus float64,
) (statusType string, delta float64, backfireDamage float64) {
	pDiv := CalculateDeductionPower(viewer, viewerGus, viewerDao, clues, target, resourceBonus)
	dCon := CalculateConcealmentDefense(target, targetGus, targetDao, viewer.CurrentRegion)

	delta = pDiv - dCon

	if delta >= 30.0 {
		return "full", delta, 0.0
	} else if delta >= 0.0 {
		return "partial", delta, 0.0
	} else if delta >= -30.0 {
		return "fogged", delta, 0.0
	} else {
		// Backfire: delta < -30
		damage := -delta * float64(target.Rank) * 5.0
		return "backfire", delta, damage
	}
}

// CalculateSeverCluesPower calculates the ability to purge tracking marks
func CalculateSeverCluesPower(player *models.Player, gus []models.GuInstance) float64 {
	basePower := BaseRankPower(player.Rank)
	attainMult := GetPathAttainmentMultiplier(player.PathMasteries, "sword", "wisdom", "dark", "rule")

	guBonus := 0.0
	for _, gu := range gus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			for _, mechanic := range spec.Effects.Other {
				if mechanic.ActionCode == "SEVER_CLUES" || mechanic.ActionCode == "PERFECT_DISGUISE" {
					guBonus += 30.0
				}
			}
		}
	}

	return (basePower + guBonus) * attainMult
}

// EvaluateGuDivination calculates divination outcome when deducing a Gu worm
func EvaluateGuDivination(
	viewer *models.Player,
	viewerGus []models.GuInstance,
	viewerDao models.DaoMarksMap,
	targetGu models.GuDictionaryItem,
	holdersCount int,
	maxHolderDefense float64,
	resourceBonus float64,
) (statusType string, delta float64, backfireDamage float64, backfireSoulDamage float64) {
	basePower := BaseRankPower(viewer.Rank)

	investigationSum := 0.0
	for _, gu := range viewerGus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok && spec.Effects.Passive != nil {
			investigationSum += spec.Effects.Passive.Investigation
		}
	}

	paths := append([]string{"wisdom", "information", "refinement"}, targetGu.Path...)
	attainMult := GetPathAttainmentMultiplier(viewer.PathMasteries, paths...)

	daoBonus := 1.0
	if viewer.Rank >= 6 && viewerDao != nil {
		wMarks := viewerDao["wisdom"]
		iMarks := viewerDao["information"]
		rMarks := viewerDao["refinement"]
		daoBonus = 1.0 + float64(wMarks+iMarks+rMarks)/1000.0
	}

	corePower := (basePower + investigationSum) * attainMult * daoBonus

	resBonus := resourceBonus
	if resBonus > corePower*0.35 {
		resBonus = corePower * 0.35
	}

	// Ownership resonance bonus: If the viewer holds or equips this Gu, their perception resonates directly
	viewerOwnsGu := false
	for _, gu := range viewerGus {
		if strings.EqualFold(gu.GuID, targetGu.ID) && (targetGu.Rank == 0 || gu.Rank == targetGu.Rank) {
			viewerOwnsGu = true
			break
		}
	}
	if !viewerOwnsGu && viewer != nil {
		for _, gu := range viewer.GuInventory {
			if strings.EqualFold(gu.GuID, targetGu.ID) && (targetGu.Rank == 0 || gu.Rank == targetGu.Rank) {
				viewerOwnsGu = true
				break
			}
		}
	}
	ownershipBonus := 0.0
	if viewerOwnsGu {
		ownershipBonus = 200.0
	}

	pDiv := corePower + resBonus + ownershipBonus

	// Target Gu Difficulty
	var baseGuDifficulty float64
	if targetGu.Rank >= 6 {
		baseGuDifficulty = float64(targetGu.Rank)*40.0 + 30.0
	} else {
		baseGuDifficulty = float64(targetGu.Rank)*12.0 + 8.0
	}

	totalDifficulty := baseGuDifficulty
	if holdersCount > 0 && maxHolderDefense > 0 && !viewerOwnsGu {
		totalDifficulty += maxHolderDefense * 0.4
	}

	delta = pDiv - totalDifficulty

	if delta >= 20.0 {
		return "full", delta, 0.0, 0.0
	} else if delta >= -10.0 {
		return "partial", delta, 0.0, 0.0
	} else if delta >= -40.0 {
		return "fogged", delta, 0.0, 0.0
	} else {
		// Backfire when trying to divine high-tier Gu beyond mortal perception
		dmg := -delta * float64(targetGu.Rank) * 3.5
		soulDmg := -delta * 1.5
		return "backfire", delta, dmg, soulDmg
	}
}

