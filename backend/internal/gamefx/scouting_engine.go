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

// GetPathAttainment returns the flat base and the mild multiplier granted by the
// highest Cảnh giới lưu phái level (0-4) attained among the candidate paths.
// The base is flat so cultivation rank stays the only rank-dependent term.
func GetPathAttainment(masteries models.PathMasteriesMap, paths ...string) (base, mult float64) {
	lvl := maxPathLevel(masteries, paths...)
	if lvl < 0 {
		lvl = 0
	}
	if lvl >= len(ScoutingAttainBase) {
		lvl = len(ScoutingAttainBase) - 1
	}
	return ScoutingAttainBase[lvl], ScoutingAttainMult[lvl]
}

// scoutingDaoMult converts raw dao marks into a saturating multiplier.
// Deliberately the weakest pillar: capped at +20% of stat (ScoutingDaoCeiling)
// so it can never outscale Cảnh giới lưu phái or equipped Gu. Unlike the old
// ×5 asymptote this also keeps the multiplier off the Gu-invested stat, so
// stacking trinh sát Gu stays worthwhile at high realm.
func scoutingDaoMult(marks ...int64) float64 {
	var total int64
	for _, m := range marks {
		total += m
	}
	if total <= 0 {
		return 1.0
	}
	return 1 + ScoutingDaoCeiling*float64(total)/(float64(total)+ScoutingDaoSoftcap)
}

// scoutingRankNorm softens the rank gap between the two sides so a 2-rank
// difference loses the roll instead of instantly backfiring.
func scoutingRankNorm(targetRank int) float64 {
	power := BaseRankPower(targetRank)
	if power <= 0 {
		power = BaseRankPower(1)
	}
	return math.Pow(10.0/power, ScoutingRankNormExp)
}

// maxPathLevel returns the highest attained realm level (0-4) among candidate paths.
// Matching is exact after CleanPath normalisation. The previous substring match
// let unrelated paths alias each other — probe "information" matched "formation",
// and "transformation" matched "formation" — handing free attainment levels.
func maxPathLevel(masteries models.PathMasteriesMap, paths ...string) int {
	highestLevel := 0
	for _, p := range paths {
		clean := CleanPath(p)
		if clean == "" {
			continue
		}
		for pathKey, pts := range masteries {
			if CleanPath(pathKey) != clean {
				continue
			}
			if lvl := PathMasteryLevel(pts); lvl > highestLevel {
				highestLevel = lvl
			}
		}
	}
	return highestLevel
}

func sumInvestigation(gus []models.GuInstance) float64 {
	sum := 0.0
	for _, gu := range gus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			sum += models.GuStatValue(spec, "INVESTIGATION") * ScoutingInvestigationWeight
		}
	}
	return sum
}

func sumDisguise(gus []models.GuInstance) (disguise float64, perfect bool, conceal bool) {
	for _, gu := range gus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			disguise += models.GuStatValue(spec, "DISGUISE") * ScoutingDisguiseWeight
			_, hasPerfect := models.GuNativeMechanic(spec, "PERFECT_DISGUISE")
			_, hasConceal := models.GuNativeMechanic(spec, "CONCEAL_GU")
			perfect = perfect || hasPerfect
			conceal = conceal || hasConceal
		}
	}
	return disguise, perfect, conceal
}

func sumLuck(gus []models.GuInstance) float64 {
	sum := 0.0
	for _, gu := range gus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			sum += models.GuStatValue(spec, "LUCK")
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
	// Pillar 1: Cổ stats (scouting). INVESTIGATION carries ScoutingInvestigationWeight
	// so trinh sát Gu are a real pillar instead of a rounding error.
	basePower := BaseRankPower(viewer.Rank)
	stat := basePower + sumInvestigation(viewerGus)

	// Pillar 2: Cảnh giới lưu phái (dominant) + đạo ngân (minor).
	// The flat attain base is the single biggest lever on the pillar, and dao
	// marks can no longer inflate it.
	attainBase, attainMult := GetPathAttainment(viewer.PathMasteries, "wisdom")
	daoBonus := 1.0
	if viewer.Rank >= 6 && viewerDaoMarks != nil {
		daoBonus = scoutingDaoMult(viewerDaoMarks["wisdom"], viewerDaoMarks["information"])
	}
	realmDao := stat*(attainMult*daoBonus-1) + attainBase

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
		Stat:       stat,
		RealmDao:   realmDao,
		Clue:       totalClue,
		Total:      corePower + totalClue,
		AttainBase: attainBase,
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

	// Pillar 2: Cảnh giới lưu phái (same flat base as the attacker) + đạo ngân.
	// Trí đối kháng trực tiếp, Ám che thân/vị/khí, Biến hóa đổi dấu vết (+điểm nhẹ).
	attainBase, attainMult := GetPathAttainment(target.PathMasteries, "dark", "transformation", "wisdom", "concealment")
	daoBonus := 1.0
	if target.Rank >= 6 && targetDaoMarks != nil {
		daoBonus = scoutingDaoMult(targetDaoMarks["concealment"], targetDaoMarks["dark"], targetDaoMarks["transformation"])
	}
	transLvl := maxPathLevel(target.PathMasteries, "transformation")
	darkLvl := maxPathLevel(target.PathMasteries, "dark")
	realmDao := stat*(attainMult*daoBonus-1) + attainBase + float64(transLvl)*8.0

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

	delta = (myScore.Total - (def.Total - pierce)) * scoutingRankNorm(target.Rank)
	return delta, myScore
}

// EvaluateDivination computes the final delta and categorizes the outcome.
// Only the final delta is compared; myScore (attacker's own breakdown incl. luck)
// is returned for the client, defender numbers stay hidden.
// backfireRatio is the fraction of the attacker's max HP / max Soul to burn —
// resolved against real max values by the caller so it is never unbounded.
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
) (statusType string, delta float64, backfireRatio float64, myScore models.ScoreDetail) {
	myScore = DeductionBreakdown(viewer, viewerGus, viewerDao, clues, target, resourceBonus, itemClueValid, suspectedGuValid, 0)
	def := ConcealmentBreakdown(target, targetGus, targetDao, viewer.CurrentRegion)

	pierce := heavenPierce(viewer, viewerDao)
	if pierce > def.Veil {
		pierce = def.Veil
	}
	luck := RollLuckDelta(viewer, viewerGus, target, targetGus)
	myScore.LuckRoll = luck
	myScore.Total += luck

	// Chuẩn hoá chênh rank: hai bên ở xa nhau về tu vi bị kéo lại gần nhau,
	// nên lệch 2 bậc chỉ là mất trắng chứ không phản phệ tức thời.
	delta = (myScore.Total - (def.Total - pierce)) * scoutingRankNorm(target.Rank)

	switch {
	case delta >= 30.0:
		return "full", delta, 0.0, myScore
	case delta >= 0.0:
		return "partial", delta, 0.0, myScore
	case delta >= -30.0:
		return "fogged", delta, 0.0, myScore
	}
	// Backfire: delta < -30
	return "backfire", delta, ScoutingBackfireRatio, myScore
}

// CalculateSeverCluesPower calculates the ability to purge tracking marks
func CalculateSeverCluesPower(player *models.Player, gus []models.GuInstance) float64 {
	basePower := BaseRankPower(player.Rank)
	_, attainMult := GetPathAttainment(player.PathMasteries, "sword", "wisdom", "dark", "rule")

	guBonus := 0.0
	for _, gu := range gus {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			if _, ok := models.GuNativeMechanic(spec, "SEVER_CLUES"); ok {
				guBonus += 30.0
			} else if _, ok := models.GuNativeMechanic(spec, "PERFECT_DISGUISE"); ok {
				guBonus += 30.0
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
) (statusType string, delta float64, backfireHPRatio float64, backfireSoulRatio float64, myScore models.ScoreDetail) {
	basePower := BaseRankPower(viewer.Rank)
	investigationSum := sumInvestigation(viewerGus)

	paths := append([]string{"wisdom", "information", "refinement"}, targetGu.Path...)
	attainBase, attainMult := GetPathAttainment(viewer.PathMasteries, paths...)

	daoBonus := 1.0
	if viewer.Rank >= 6 && viewerDao != nil {
		daoBonus = scoutingDaoMult(viewerDao["wisdom"], viewerDao["information"], viewerDao["refinement"])
	}

	stat := basePower + investigationSum
	realmDao := stat*(attainMult*daoBonus-1) + attainBase

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
		Total:       (stat+realmDao+clue)/divisor + luck,
		AttainBase:  attainBase / divisor,
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
	// Backfire when trying to divine high-tier Gu beyond mortal perception.
	// Returned as fractions of the diviner's max HP / max Soul so the caller can
	// resolve them against real maxima instead of an unbounded absolute number.
	return "backfire", delta, ScoutingBackfireRatio, ScoutingBackfireRatio, myScore
}
