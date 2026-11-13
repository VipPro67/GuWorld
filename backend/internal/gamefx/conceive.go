package gamefx

import (
	"math"
	"math/rand"
)

const (
	// ConceiveAccuracyBase is the starting base accuracy rate for recipe conception.
	ConceiveAccuracyBase = 0.60
	// ConceiveAccuracyMin is the floor for accuracy rate (15%).
	ConceiveAccuracyMin = 0.15
	// ConceiveAccuracyMax is the ceiling for accuracy rate (88% - 12% uncertainty).
	ConceiveAccuracyMax = 0.88

	// ConceiveRealmPositiveStep is bonus accuracy per realm level above required (+8%).
	ConceiveRealmPositiveStep = 0.08
	// ConceiveRealmNegativeStep is penalty per realm level below required (-20%).
	ConceiveRealmNegativeStep = 0.20

	// ConceiveWisdomStep is bonus per wisdom realm level (+4%).
	ConceiveWisdomStep = 0.04
	// ConceiveDaoMarkFactor scales wisdom dao marks bonus.
	ConceiveDaoMarkFactor = 0.004
	// ConceiveDaoMarkCap caps dao mark bonus at +8%.
	ConceiveDaoMarkCap = 0.08

	// ConceiveBaseProgressMin is base progress floor (15%).
	ConceiveBaseProgressMin = 15.0
	// ConceiveBaseProgressRange is random additional base progress (0% - 15%).
	ConceiveBaseProgressRange = 15.0
)

// ConceiveThoughtCosts maps Gu rank (1-9) to Niệm Lực cost.
var ConceiveThoughtCosts = [10]float64{
	0: 30.0,
	1: 30.0,
	2: 50.0,
	3: 80.0,
	4: 130.0,
	5: 200.0,
	6: 400.0,
	7: 600.0,
	8: 900.0,
	9: 1500.0,
}

// ConceivePrimevalStoneCosts maps mortal Gu rank (1-5) to Nguyên Thạch cost.
var ConceivePrimevalStoneCosts = [10]float64{
	0: 50.0,
	1: 50.0,
	2: 150.0,
	3: 500.0,
	4: 2000.0,
	5: 10000.0,
	6: 0.0,
	7: 0.0,
	8: 0.0,
	9: 0.0,
}

// ConceiveImmortalStoneCosts maps immortal Gu rank (6-9) to Tiên Nguyên Thạch cost.
var ConceiveImmortalStoneCosts = [10]float64{
	0: 0.0,
	1: 0.0,
	2: 0.0,
	3: 0.0,
	4: 0.0,
	5: 0.0,
	6: 100.0,
	7: 500.0,
	8: 2500.0,
	9: 10000.0,
}

// RequiredAttainmentRealm returns the benchmark realm level required for a Gu rank.
// Rank 1-2: Ordinary (0), Rank 3-4: Master (1), Rank 5-6: Grandmaster (2),
// Rank 7-8: Great Grandmaster (3), Rank 9: Supreme Grandmaster (4).
func RequiredAttainmentRealm(rank int) int {
	switch {
	case rank <= 2:
		return 0
	case rank <= 4:
		return 1
	case rank <= 6:
		return 2
	case rank <= 8:
		return 3
	default:
		return 4
	}
}

// CalculateConceiveAccuracy computes the accuracy chance to conceive the target Gu.
func CalculateConceiveAccuracy(targetRank, targetRealm, wisdomRealm int, wisdomDaoMarks int64, catalystBonus float64) float64 {
	req := RequiredAttainmentRealm(targetRank)
	diff := targetRealm - req

	var realmMod float64
	if diff >= 0 {
		realmMod = float64(diff) * ConceiveRealmPositiveStep
	} else {
		realmMod = float64(diff) * ConceiveRealmNegativeStep
	}

	wisdomMod := float64(wisdomRealm) * ConceiveWisdomStep

	daoMod := 0.0
	if wisdomDaoMarks > 0 {
		daoMod = math.Min(ConceiveDaoMarkCap, math.Sqrt(float64(wisdomDaoMarks))*ConceiveDaoMarkFactor)
	}

	acc := ConceiveAccuracyBase + realmMod + wisdomMod + daoMod + catalystBonus
	if acc < ConceiveAccuracyMin {
		acc = ConceiveAccuracyMin
	}
	if acc > ConceiveAccuracyMax {
		acc = ConceiveAccuracyMax
	}
	return acc
}

// CalculateBacklashRisk computes the probability of Trí Đạo Phản Phệ.
func CalculateBacklashRisk(playerRank, targetRank, wisdomRealm, refineRealm int) float64 {
	// Mortals attempting immortal rank (rank >= 6) is completely catastrophic
	if playerRank <= 5 && targetRank >= 6 {
		return 1.0
	}

	if targetRank > playerRank {
		diff := targetRank - playerRank
		return 0.35 + float64(diff)*0.25
	}

	// High rank mortals (rank >= 4) with zero wisdom and refinement attainment
	if targetRank >= 4 && wisdomRealm == 0 && refineRealm == 0 {
		return 0.15
	}

	return 0.0
}

// ConceiveThoughtCost returns the thought cost for conceiving a recipe of rank.
func ConceiveThoughtCost(rank int) float64 {
	if rank < 1 {
		return ConceiveThoughtCosts[1]
	}
	if rank > 9 {
		return ConceiveThoughtCosts[9]
	}
	return ConceiveThoughtCosts[rank]
}

// ConceiveStoneCost returns the primeval stone cost for mortal ranks 1-5.
func ConceiveStoneCost(rank int) float64 {
	if rank < 1 {
		return ConceivePrimevalStoneCosts[1]
	}
	if rank > 9 {
		return ConceivePrimevalStoneCosts[9]
	}
	return ConceivePrimevalStoneCosts[rank]
}

// ConceiveImmortalStoneCost returns the immortal stone cost for ranks 6-9.
func ConceiveImmortalStoneCost(rank int) float64 {
	if rank < 1 {
		return 0.0
	}
	if rank > 9 {
		return ConceiveImmortalStoneCosts[9]
	}
	return ConceiveImmortalStoneCosts[rank]
}

// GuCatalystBonus calculates the accuracy bonus when sacrificing a living Gu.
func GuCatalystBonus(guRank, targetRank int) float64 {
	if guRank >= targetRank {
		return 0.20
	}
	if guRank == targetRank-1 {
		return 0.16
	}
	return 0.12
}

// MaterialCatalystBonus calculates the accuracy bonus when offering materials.
func MaterialCatalystBonus(matRank, targetRank int) float64 {
	if matRank >= targetRank {
		return 0.12
	}
	return 0.08
}

// MaterialCatalystRequiredQty calculates the required material count.
func MaterialCatalystRequiredQty(matRank, targetRank int) int64 {
	if matRank >= targetRank {
		if targetRank >= 6 {
			return 5
		}
		return 10
	}
	if matRank == targetRank-1 {
		return 25
	}
	return 50
}

// CalculateInitialConceiveProgress returns initial progress percentage (15% - 35%)
// and whether Flash of Inspiration (100% instant finish) triggered.
func CalculateInitialConceiveProgress(wisdomRealm, refineRealm, targetRealm, targetRank int) (float64, bool) {
	mult := DeductionRealmMultiplier(wisdomRealm, refineRealm, targetRealm)
	rankScale := math.Sqrt(float64(targetRank))
	if rankScale <= 0 {
		rankScale = 1.0
	}

	inspirationChance := (0.02 * mult) / rankScale
	if rand.Float64() < inspirationChance {
		return 100.0, true
	}

	base := ConceiveBaseProgressMin + rand.Float64()*ConceiveBaseProgressRange
	bonus := float64(wisdomRealm)*2.0 + float64(refineRealm)*1.5 + float64(targetRealm)*1.0
	initialRate := math.Round((base+bonus)*10) / 10.0
	if initialRate > 35.0 {
		initialRate = 35.0
	}
	return initialRate, false
}
