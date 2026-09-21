package gamefx

import (
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// ScoutingRand is the RNG for Vận đạo luck rolls. Tests may reseed it.
// All rolls go through rollLuckFloat so concurrent requests stay race-free.
var ScoutingRand = rand.New(rand.NewSource(time.Now().UnixNano()))

var scoutingRandMu sync.Mutex

func rollLuckFloat() float64 {
	scoutingRandMu.Lock()
	defer scoutingRandMu.Unlock()
	return ScoutingRand.Float64()
}

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
	highestLevel := maxPathLevel(masteries, paths...)

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

// daoMult converts raw dao marks into a saturating multiplier.
// The old linear 1+m/1000 exploded for whales (100k marks = 100x, multiplied
// again by attainment mult); this asymptotes to 5x: 0 marks = 1x,
// 1000 marks ≈ 2.3x, 8000+ marks ≈ 5x.
func daoMult(marks ...int64) float64 {
	var total int64
	for _, m := range marks {
		total += m
	}
	if total <= 0 {
		return 1.0
	}
	return 1 + 4*float64(total)/(float64(total)+2000)
}

// maxPathLevel returns the highest attained realm level (0-4) among candidate paths.
func maxPathLevel(masteries models.PathMasteriesMap, paths ...string) int {
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
	return highestLevel
}

func sumInvestigation(gus []models.GuInstance) float64 {
	sum := 0.0
	for _, gu := range gus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok && spec.Effects.Passive != nil {
			sum += spec.Effects.Passive.Investigation
		}
	}
	return sum
}

func sumDisguise(gus []models.GuInstance) (disguise float64, perfect bool, conceal bool) {
	for _, gu := range gus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			if spec.Effects.Passive != nil {
				disguise += spec.Effects.Passive.DisguiseLevel * 5.0
			}
			for _, mechanic := range spec.Effects.Other {
				if mechanic.ActionCode == "PERFECT_DISGUISE" {
					perfect = true
				}
				if mechanic.ActionCode == "CONCEAL_GU" {
					conceal = true
				}
			}
		}
	}
	return disguise, perfect, conceal
}

func sumLuck(gus []models.GuInstance) float64 {
	sum := 0.0
	for _, gu := range gus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok && spec.Effects.Passive != nil {
			sum += spec.Effects.Passive.Luck
		}
	}
	return sum
}

// DeductionBreakdown computes the attacker's 3-pillar score (stat / realm+dao / clue).
// LuckRoll is left at 0; Evaluate* fills it in. extraClue carries the local-proximity
// bonus for inspect (+20) and is 0 for remote divination.
func DeductionBreakdown(
	viewer *models.Player,
	viewerGus []models.GuInstance,
	viewerDaoMarks models.DaoMarksMap,
	clues models.DivinationRequest,
	target *models.Player,
	resourceBonus float64,
	itemClueValid bool,
	suspectedGuValid bool,
	extraClue float64,
) models.ScoreDetail {
	// Pillar 1: Cổ stats (scouting).
	basePower := BaseRankPower(viewer.Rank)
	stat := basePower + sumInvestigation(viewerGus)

	// Pillar 2: cảnh giới + đạo ngân. Trí đạo là core mult.
	attainMult := GetPathAttainmentMultiplier(viewer.PathMasteries, "wisdom")
	daoBonus := 1.0
	if viewer.Rank >= 6 && viewerDaoMarks != nil {
		daoBonus = daoMult(viewerDaoMarks["wisdom"], viewerDaoMarks["information"])
	}
	realmDao := stat * (attainMult*daoBonus - 1)

	// Pillar 3: clue (thông tin phe công nắm). Validated server-side.
	clue := 0.0
	if clues.TargetUsername != "" {
		if target != nil && strings.EqualFold(clues.TargetUsername, target.Username) {
			clue += 20.0
		}
	} else {
		// Blind deduction penalty
		clue -= 40.0
	}
	if clues.SuspectedSect != "" {
		if clues.TargetSectName != "" && strings.EqualFold(clues.SuspectedSect, clues.TargetSectName) {
			clue += 10.0
		}
	}
	if clues.SuspectedRegion != "" {
		if target != nil && strings.EqualFold(clues.SuspectedRegion, target.CurrentRegion) {
			clue += 15.0
		}
	}
	if clues.SuspectedGu != "" && suspectedGuValid {
		clue += 10.0
	}
	if clues.ItemClueID != "" && itemClueValid {
		clue += 30.0
	}
	// Tín đạo phe công khuếch đại clue.
	faithLvl := maxPathLevel(viewer.PathMasteries, "information")
	clue *= 1 + 0.15*float64(faithLvl)
	// Tín đạo phe thủ phong tỏa / bóp méo clue.
	if target != nil {
		targetFaith := maxPathLevel(target.PathMasteries, "information")
		clue *= 1 - math.Min(0.5, float64(targetFaith)*0.12)
	}
	clue += extraClue
	// Hard cap: clue không gánh được full chỉ bằng manh mối.
	if clue > 40.0 {
		clue = 40.0
	} else if clue < -40.0 {
		clue = -40.0
	}

	// Resource investment bonus (capped at +25% of core power).
	corePower := stat + realmDao
	resBonus := resourceBonus
	if resBonus > corePower*0.25 {
		resBonus = corePower * 0.25
	}

	totalClue := clue + resBonus
	return models.ScoreDetail{
		Stat:     stat,
		RealmDao: realmDao,
		Clue:     totalClue,
		Total:    corePower + totalClue,
	}
}

// CalculateDeductionPower computes P_div for the divining player.
// Kept for compatibility; new code should use DeductionBreakdown.
func CalculateDeductionPower(
	viewer *models.Player,
	viewerGus []models.GuInstance,
	viewerDaoMarks models.DaoMarksMap,
	clues models.DivinationRequest,
	target *models.Player,
	resourceBonus float64,
) float64 {
	return DeductionBreakdown(viewer, viewerGus, viewerDaoMarks, clues, target, resourceBonus, false, false, 0).Total
}

// ConcealmentBreakdown computes the defender's 3-pillar score. Server-side only,
// never exposed to the client.
func ConcealmentBreakdown(
	target *models.Player,
	targetGus []models.GuInstance,
	targetDaoMarks models.DaoMarksMap,
	viewerRegion string,
) models.DefenseDetail {
	baseDefense := BaseRankPower(target.Rank)

	// Pillar 1: disguise / concealment stats.
	disguiseSum, hasPerfectDisguise, hasConceal := sumDisguise(targetGus)
	capabilityBonus := 0.0
	if hasPerfectDisguise {
		capabilityBonus += 80.0
	}
	if hasConceal {
		capabilityBonus += 30.0
	}
	stat := baseDefense + disguiseSum + capabilityBonus

	// Pillar 2: cảnh giới + đạo ngân.
	// Trí đối kháng trực tiếp, Ám che thân/vị/khí, Biến hóa đổi dấu vết (+điểm nhẹ).
	attainMult := GetPathAttainmentMultiplier(target.PathMasteries, "dark", "transformation", "wisdom", "concealment")
	daoBonus := 1.0
	if target.Rank >= 6 && targetDaoMarks != nil {
		daoBonus = daoMult(targetDaoMarks["concealment"], targetDaoMarks["dark"], targetDaoMarks["transformation"])
	}
	transLvl := maxPathLevel(target.PathMasteries, "transformation")
	darkLvl := maxPathLevel(target.PathMasteries, "dark")
	realmDao := stat*(attainMult*daoBonus-1) + float64(transLvl)*8.0

	// Thiên đạo veil: che / bóp méo thiên cơ, chỉ Tiên nhân.
	veil := 0.0
	if target.Rank >= 6 {
		heavenLvl := maxPathLevel(target.PathMasteries, "heaven")
		var heavenMarks int64
		if targetDaoMarks != nil {
			heavenMarks = targetDaoMarks["heaven"]
		}
		veil = float64(heavenLvl)*20.0 + float64(heavenMarks)/500.0
	}

	// Pillar 3 (anti-clue): rào cản không gian + Ám che vị trí.
	distance := 10.0 // Same region
	if viewerRegion != "" && target.CurrentRegion != "" && !strings.EqualFold(viewerRegion, target.CurrentRegion) {
		distance = 35.0 // Cross-region boundary
	}
	if darkLvl >= 1 {
		distance += 10.0
	}

	return models.DefenseDetail{
		Stat:     stat,
		RealmDao: realmDao,
		Veil:     veil,
		Distance: distance,
		Total:    stat + realmDao + veil + distance,
	}
}

// CalculateConcealmentDefense computes D_con for the target player.
// Kept for compatibility; new code should use ConcealmentBreakdown.
func CalculateConcealmentDefense(
	target *models.Player,
	targetGus []models.GuInstance,
	targetDaoMarks models.DaoMarksMap,
	viewerRegion string,
) float64 {
	return ConcealmentBreakdown(target, targetGus, targetDaoMarks, viewerRegion).Total
}

// heavenPierce is the attacker's Thiên đạo power to see through the defender's veil,
// capped at the veil value by the caller.
func heavenPierce(viewer *models.Player, viewerDaoMarks models.DaoMarksMap) float64 {
	lvl := maxPathLevel(viewer.PathMasteries, "heaven")
	if lvl == 0 && (viewerDaoMarks == nil || viewerDaoMarks["heaven"] == 0) {
		return 0
	}
	var heavenMarks int64
	if viewerDaoMarks != nil {
		heavenMarks = viewerDaoMarks["heaven"]
	}
	return float64(lvl)*10.0 + float64(heavenMarks)/500.0
}

// RollLuckDelta rolls the Vận đạo jitter for player-vs-player divination.
// Attacker Vận narrows the spread; a luckier defender drags the mean negative.
func RollLuckDelta(
	viewer *models.Player,
	viewerGus []models.GuInstance,
	target *models.Player,
	targetGus []models.GuInstance,
) float64 {
	vLvl := maxPathLevel(viewer.PathMasteries, "luck")
	tLvl := 0
	if target != nil {
		tLvl = maxPathLevel(target.PathMasteries, "luck")
	}
	width := 10.0 + 10.0*float64(max(vLvl, tLvl)) + 0.5*math.Abs(sumLuck(viewerGus)-sumLuck(targetGus))
	width /= 1.0 + 0.2*float64(vLvl)
	r := rollLuckFloat()*2 - 1
	bias := -3.0 * float64(tLvl-vLvl)
	return r*width + bias
}

// RollLuckGuDelta rolls the Vận jitter for Gu divination against a Gu of targetRank.
func RollLuckGuDelta(viewer *models.Player, viewerGus []models.GuInstance, targetRank int) float64 {
	vLvl := maxPathLevel(viewer.PathMasteries, "luck")
	width := (10.0 + 6.0*float64(targetRank)) / (1.0 + 0.2*float64(vLvl))
	r := rollLuckFloat()*2 - 1
	bias := -1.0*float64(targetRank) + 2.0*float64(vLvl)
	return r*width + bias
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// EvaluateInspect scores a local inspection. Same engine as remote divination,
// but local presence counts as a +20 proximity clue and there is no blind penalty.
func EvaluateInspect(
	viewer *models.Player,
	target *models.Player,
	viewerGus []models.GuInstance,
	targetGus []models.GuInstance,
	viewerDao models.DaoMarksMap,
	targetDao models.DaoMarksMap,
) (delta float64, myScore models.ScoreDetail) {
	clues := models.DivinationRequest{TargetUsername: target.Username}
	myScore = DeductionBreakdown(viewer, viewerGus, viewerDao, clues, target, 0, false, false, 20.0)
	def := ConcealmentBreakdown(target, targetGus, targetDao, viewer.CurrentRegion)

	pierce := heavenPierce(viewer, viewerDao)
	if pierce > def.Veil {
		pierce = def.Veil
	}
	luck := RollLuckDelta(viewer, viewerGus, target, targetGus)
	myScore.LuckRoll = luck
	myScore.Total += luck

	return myScore.Total - (def.Total - pierce), myScore
}

// EvaluateDivination computes the final delta and categorizes the outcome.
// Only the final delta is compared; myScore (attacker's own breakdown incl. luck)
// is returned for the client, defender numbers stay hidden.
func EvaluateDivination(
	viewer *models.Player,
	target *models.Player,
	viewerGus []models.GuInstance,
	targetGus []models.GuInstance,
	viewerDao models.DaoMarksMap,
	targetDao models.DaoMarksMap,
	clues models.DivinationRequest,
	resourceBonus float64,
	itemClueValid bool,
	suspectedGuValid bool,
) (statusType string, delta float64, backfireDamage float64, myScore models.ScoreDetail) {
	myScore = DeductionBreakdown(viewer, viewerGus, viewerDao, clues, target, resourceBonus, itemClueValid, suspectedGuValid, 0)
	def := ConcealmentBreakdown(target, targetGus, targetDao, viewer.CurrentRegion)

	pierce := heavenPierce(viewer, viewerDao)
	if pierce > def.Veil {
		pierce = def.Veil
	}
	luck := RollLuckDelta(viewer, viewerGus, target, targetGus)
	myScore.LuckRoll = luck
	myScore.Total += luck

	delta = myScore.Total - (def.Total - pierce)

	if delta >= 30.0 {
		return "full", delta, 0.0, myScore
	} else if delta >= 0.0 {
		return "partial", delta, 0.0, myScore
	} else if delta >= -30.0 {
		return "fogged", delta, 0.0, myScore
	}
	// Backfire: delta < -30
	damage := -delta * float64(target.Rank) * 5.0
	return "backfire", delta, damage, myScore
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

// GuRankDivisor scales divining power by target Gu rank: P_effective = P_raw/divisor.
// Reuses the power curve (BaseRankPower/10): R1-5 = 1/2/3/4/5, R6 = 10, R7 = 18,
// R8 = 30, R9 = 50. A whale's raw P fulls low-rank Gus but must still work for
// immortal ones — mirroring the rankPenalty of recipe deduction.
func GuRankDivisor(guRank int) float64 {
	return BaseRankPower(guRank) / 10.0
}

// EvaluateGuDivination calculates divination outcome when deducing a Gu worm.
// Raw power is divided by GuRankDivisor, so rank carries the scaling instead of
// a linear difficulty number. Remaining difficulty is only the strongest
// holder's concealment (x0.5, sleep state already blended at the call site)
// plus a flat natural veil of 15. Only the final delta is compared.
func EvaluateGuDivination(
	viewer *models.Player,
	viewerGus []models.GuInstance,
	viewerDao models.DaoMarksMap,
	targetGu models.GuDictionaryItem,
	holdersCount int,
	maxHolderDefense float64,
	resourceBonus float64,
) (statusType string, delta float64, backfireDamage float64, backfireSoulDamage float64, myScore models.ScoreDetail) {
	basePower := BaseRankPower(viewer.Rank)
	investigationSum := sumInvestigation(viewerGus)

	paths := append([]string{"wisdom", "information", "refinement"}, targetGu.Path...)
	attainMult := GetPathAttainmentMultiplier(viewer.PathMasteries, paths...)

	daoBonus := 1.0
	if viewer.Rank >= 6 && viewerDao != nil {
		daoBonus = daoMult(viewerDao["wisdom"], viewerDao["information"], viewerDao["refinement"])
	}

	stat := basePower + investigationSum
	realmDao := stat*(attainMult*daoBonus-1)

	resBonus := resourceBonus
	if resBonus > (stat+realmDao)*0.35 {
		resBonus = (stat + realmDao) * 0.35
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
		ownershipBonus = 250.0
	}

	divisor := GuRankDivisor(targetGu.Rank)

	clue := resBonus + ownershipBonus
	luck := RollLuckGuDelta(viewer, viewerGus, targetGu.Rank) / divisor
	myScore = models.ScoreDetail{
		Stat:        stat / divisor,
		RealmDao:    realmDao / divisor,
		Clue:        clue / divisor,
		LuckRoll:    luck,
		Total:       (stat + realmDao + clue) / divisor + luck,
		RankDivisor: divisor,
	}

	totalDifficulty := 15.0 // natural heaven veil on every Gu
	if holdersCount > 0 && maxHolderDefense > 0 && !viewerOwnsGu {
		totalDifficulty += maxHolderDefense * 0.5
	}

	delta = myScore.Total - totalDifficulty

	if delta >= 20.0 {
		return "full", delta, 0.0, 0.0, myScore
	} else if delta >= -10.0 {
		return "partial", delta, 0.0, 0.0, myScore
	} else if delta >= -40.0 {
		return "fogged", delta, 0.0, 0.0, myScore
	}
	// Backfire when trying to divine high-tier Gu beyond mortal perception
	dmg := -delta * float64(targetGu.Rank) * 3.5
	soulDmg := -delta * 1.5
	return "backfire", delta, dmg, soulDmg, myScore
}
