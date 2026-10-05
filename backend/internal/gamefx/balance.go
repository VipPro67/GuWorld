package gamefx

import (
	"math"
	"time"
)

// ========================================================================
// 1. CẢNH GIỚI LƯU PHÁI (PATH MASTERY / ATTAINMENT)
// ========================================================================

const (
	// PathMasteryMasterPoints is the milestone to reach Master (level 1 / Đại Sư).
	PathMasteryMasterPoints = 10000
	// PathMasteryGrandmasterPoints is the milestone to reach Grandmaster (level 2 / Tông Sư).
	PathMasteryGrandmasterPoints = 100000
	// PathMasteryGreatGrandmasterPoints is the milestone to reach Great Grandmaster (level 3 / Đại Tông Sư).
	PathMasteryGreatGrandmasterPoints = 1000000
	// PathMasterySupremeGrandmasterPoints is the milestone to reach Supreme Grandmaster (level 4 / Vô Thượng Đại Tông Sư).
	PathMasterySupremeGrandmasterPoints = 10000000
)

// PathMasteryLevelMultipliers maps an attainment level (0-4) to its bonus
// multiplier used in combat (mirrors the old level-name table).
// Level 0 (Ordinary): 1.0
// Level 1 (Master): 1.1
// Level 2 (Grandmaster): 1.25
// Level 3 (Great Grandmaster): 1.4
// Level 4 (Supreme Grandmaster): 1.5
var PathMasteryLevelMultipliers = [5]float64{1.0, 1.1, 1.25, 1.4, 1.5}

const (
	// PathMasteryEfficiencyLossPerLevel is the reduction in mastery gain per path level attained (20% per level).
	PathMasteryEfficiencyLossPerLevel = 0.20
	// PathMasteryBattleGainPerRankMultiplier is the cap multiplier per battle based on cultivation rank (rank * 10).
	PathMasteryBattleGainPerRankMultiplier = 10
)

// PathMasteryGuUseBaseGains defines the base Chân Ý earned each time a Gu is
// successfully used. Higher-rank Gu grant more points to keep progression in
// step with the path-mastery milestones.
var PathMasteryGuUseBaseGains = [10]int{
	0: 0,
	1: 1,
	2: 2,
	3: 5,
	4: 10,
	5: 20,
	6: 50,
	7: 100,
	8: 200,
	9: 300,
}

// PathMasteryGuUseBaseGain returns the rank-scaled base Chân Ý gain for
// using a Gu. Ranks outside the supported range are clamped to ranks 1..9.
func PathMasteryGuUseBaseGain(rank int) int {
	if rank < 1 {
		rank = 1
	} else if rank > 9 {
		rank = 9
	}
	return PathMasteryGuUseBaseGains[rank]
}

// ========================================================================
// 2. BẬC QUY MÔ / RANK SCALE & MANA TABLE (CHUẨN HÓA 9 CHUYỂN)
// ========================================================================

// RankScaleTable defines the exponential cost and power scale across ranks 1 to 9.
// Mortals: Rank 1=1, Rank 2=2, Rank 3=5, Rank 4=20, Rank 5=100.
// Immortals: Rank 6=10,000 (10^4), Rank 7=1,000,000 (10^6), Rank 8=100,000,000 (10^8), Rank 9=10,000,000,000 (10^10).
var RankScaleTable = [10]float64{
	0: 1.0,
	1: 1.0,
	2: 2.0,
	3: 5.0,
	4: 20.0,
	5: 100.0,
	6: 10000.0,
	7: 1000000.0,
	8: 100000000.0,
	9: 10000000000.0,
}

// RankScale returns the standard rank scale factor (float64) for a given rank (1-9).
func RankScale(rank int) float64 {
	if rank <= 1 {
		return RankScaleTable[1]
	}
	if rank >= 9 {
		return RankScaleTable[9]
	}
	return RankScaleTable[rank]
}

// InstantDeductionChance calculates the probability of instantly completing
// a recipe based on the Gu's rank and the target recipe's rank.
// chance = base + (guRank - targetRank) * per_rank_step, clamped to [min, max]
func InstantDeductionChance(guRank, targetRank int, base, perRankStep, minChance, maxChance float64) float64 {
	chance := base + float64(guRank-targetRank)*perRankStep
	if chance < minChance {
		chance = minChance
	}
	if chance > maxChance {
		chance = maxChance
	}
	return chance
}

// SealingStoneCostByRank returns the number of sealing stones (Phong Ấn Thạch)
// required to seal a Gu of the given rank (1-9).
func SealingStoneCostByRank(rank int) int64 {
	if rank <= 1 {
		return int64(RankScaleTable[1])
	}
	if rank >= 9 {
		return int64(RankScaleTable[9])
	}
	return int64(RankScaleTable[rank])
}

// ========================================================================
// 3. HỢP LUYỆN CỔ (GU REFINEMENT BALANCE)
// ========================================================================

const (
	// RefinementDaoMarkFactor is the scaling factor for Dao Marks in refinement success rate calculation.
	RefinementDaoMarkFactor = 0.00632
	// RefinementLuckDaoMarkFactor is the factor converting luck dao marks to bonus rate.
	RefinementLuckDaoMarkFactor = 0.001
	// RefinementLuckBonusCap is the maximum luck bonus allowed.
	RefinementLuckBonusCap = 0.10
	// RefinementEnvBonus is the bonus rate granted when region path matches target Gu path.
	RefinementEnvBonus = 0.05
	// RefinementMaterialRankDiffBonus is the bonus per rank an ingredient exceeds recipe requirement.
	RefinementMaterialRankDiffBonus = 0.02
	// RefineSafeZoneFactor defines how much failed rolls can be mitigated into FAIL_SAFE based on realm.
	RefineSafeZoneFactor = 0.25
	// RefineBackfireThresholdMortal is the roll threshold above which mortal refining backfires.
	RefineBackfireThresholdMortal = 0.90
	// RefineBackfireThresholdImmortal is the roll threshold above which immortal refining backfires.
	RefineBackfireThresholdImmortal = 0.75
	// RefineManaBaseCostMortal is the base mana required for mortal Gu refining.
	RefineManaBaseCostMortal = 20.0
	// RefineManaBaseCostImmortal is the base mana required for immortal Gu refining.
	RefineManaBaseCostImmortal = 10.0
	// RefineBackfireHPLossRatio is the fraction of max HP lost on backfire.
	RefineBackfireHPLossRatio = 0.50
	// RefineBackfireSoulLossRatio is the fraction of max Soul lost on backfire.
	RefineBackfireSoulLossRatio = 0.30
	// RefineForcedFailHPLoss is deprecated (ép luyện đã bị loại bỏ:
	// luyện cổ yêu cầu bí phương 100%). Kept for backward compatibility.
	RefineForcedFailHPLoss = 30.0
)

// RefinementMasteryBonuses maps refinement path attainment realm (0-4) to success rate bonus.
// Realm 0: 0.0, Realm 1: 0.05, Realm 2: 0.10, Realm 3: 0.18, Realm 4: 0.25.
var RefinementMasteryBonuses = [5]float64{0.0, 0.05, 0.10, 0.18, 0.25}

// RefinementMasteryBonus returns the refinement success bonus for a given realm level (0-4).
func RefinementMasteryBonus(realm int) float64 {
	if realm <= 0 {
		return 0.0
	}
	if realm >= len(RefinementMasteryBonuses) {
		return RefinementMasteryBonuses[len(RefinementMasteryBonuses)-1]
	}
	return RefinementMasteryBonuses[realm]
}

// TargetPathMasteryBonuses maps target path attainment realm (0-4) to success rate bonus.
// Realm 0: 0.0, Realm 1: 0.02, Realm 2: 0.05, Realm 3: 0.08, Realm 4: 0.10.
var TargetPathMasteryBonuses = [5]float64{0.0, 0.02, 0.05, 0.08, 0.10}

// TargetPathMasteryBonus returns the target path success bonus for a given realm level (0-4).
func TargetPathMasteryBonus(realm int) float64 {
	if realm <= 0 {
		return 0.0
	}
	if realm >= len(TargetPathMasteryBonuses) {
		return TargetPathMasteryBonuses[len(TargetPathMasteryBonuses)-1]
	}
	return TargetPathMasteryBonuses[realm]
}

// RefinementImmortalPenalties maps immortal rank (6-9) to the divider applied to flat bonuses.
var RefinementImmortalPenalties = [10]float64{
	0: 1.0, 1: 1.0, 2: 1.0, 3: 1.0, 4: 1.0, 5: 1.0,
	6: 2.0,
	7: 5.0,
	8: 20.0,
	9: 100.0,
}

// RefinementImmortalPenalty returns the immortal penalty divider for rank >= 6.
func RefinementImmortalPenalty(targetRank int) float64 {
	if targetRank < 6 {
		return 1.0
	}
	if targetRank >= 9 {
		return RefinementImmortalPenalties[9]
	}
	return RefinementImmortalPenalties[targetRank]
}

// RefinementDaoBonus calculates the bonus rate granted by refinement dao marks and target path points.
func RefinementDaoBonus(refinementMarks int64, targetPathPoints int, targetRank int) float64 {
	if refinementMarks <= 0 {
		return 0.0
	}
	masteryMultiplier := 1.0 + float64(targetPathPoints)
	scale := RankScale(targetRank)
	return (RefinementDaoMarkFactor * math.Sqrt(float64(refinementMarks)) * masteryMultiplier) / scale
}

// RefinementLuckBonus calculates the luck bonus from luck dao marks and active status modifiers.
func RefinementLuckBonus(luckMarks int64, statusLuckBonus float64) float64 {
	bonus := (float64(luckMarks) * RefinementLuckDaoMarkFactor) + (statusLuckBonus / 100.0)
	return math.Min(RefinementLuckBonusCap, bonus)
}

// RefinePathBaseGains defines the base Chân Ý points gained for successfully refining a Gu of rank 1-9.
var RefinePathBaseGains = [10]int{
	0: 0,
	1: 2,
	2: 5,
	3: 15,
	4: 40,
	5: 100,
	6: 500,
	7: 2500,
	8: 15000,
	9: 50000,
}

// CalculateRefinementPathGain returns the Chân Ý points gained from refining a Gu of targetRank.
func CalculateRefinementPathGain(targetRank, refinementRealm int) int {
	baseGain := 0
	if targetRank >= 1 && targetRank <= 9 {
		baseGain = RefinePathBaseGains[targetRank]
	} else if targetRank > 9 {
		baseGain = RefinePathBaseGains[9]
	}
	return CalculatePathMasteryGain(targetRank, refinementRealm, baseGain)
}

// ========================================================================
// 4. THÔI DIỄN CÔNG THỨC (RECIPE DEDUCTION BALANCE)
// ========================================================================

const (
	// DeductionWisdomRealmWeight is the multiplier for wisdom realm in deduction.
	DeductionWisdomRealmWeight = 3.0
	// DeductionRefineRealmWeight is the multiplier for refinement realm in deduction.
	DeductionRefineRealmWeight = 2.0
	// DeductionTargetRealmWeight is the multiplier for target path realm in deduction.
	DeductionTargetRealmWeight = 1.0
	// DeductionBaseImprovementMin is the minimum base progress percentage added per deduction.
	DeductionBaseImprovementMin = 10.0
	// DeductionBaseImprovementRange is the random progress range (10% to 25%).
	DeductionBaseImprovementRange = 15.0
	// DeductionInspirationFlashBaseChance is the base flash of inspiration chance.
	DeductionInspirationFlashBaseChance = 0.03
)

// DeductionRealmMultiplier computes the combined realm multiplier for recipe deduction.
func DeductionRealmMultiplier(wisdomRealm, refineRealm, targetRealm int) float64 {
	return 1.0 +
		float64(wisdomRealm)*DeductionWisdomRealmWeight +
		float64(refineRealm)*DeductionRefineRealmWeight +
		float64(targetRealm)*DeductionTargetRealmWeight
}

// ========================================================================
// 5. PHẦN THƯỞNG CHIẾN ĐẤU (COMBAT LOOT SCALING)
// ========================================================================

// LootChanceMultiplier returns the drop chance multiplier based on rank.
func LootChanceMultiplier(rank int) float64 {
	switch rank {
	case 1:
		return 1.0
	case 2:
		return 1.1
	case 3:
		return 1.2
	case 4:
		return 1.4
	case 5:
		return 1.6
	case 6:
		return 1.0
	case 7:
		return 1.5
	case 8:
		return 2.5
	case 9:
		return 4.0
	default:
		if rank > 9 {
			return 4.0
		}
		return 1.0
	}
}

// LootQuantityMultiplier returns the drop quantity multiplier based on rank.
func LootQuantityMultiplier(rank int) float64 {
	switch rank {
	case 1:
		return 1.0
	case 2:
		return 2.0
	case 3:
		return 3.0
	case 4:
		return 4.0
	case 5:
		return 5.0
	case 6:
		return 1.0
	case 7:
		return 3.0
	case 8:
		return 5.0
	case 9:
		return 8.0
	default:
		if rank > 9 {
			return 8.0
		}
		return 1.0
	}
}

// ========================================================================
// 6. THAY ĐỔI DANH XƯNG & HÌNH PHẠT CỔ TIÊN (RENAME & RE-IDENTITY)
// ========================================================================

// RenameCost returns the currency type ("free", "primeval", "immortal") and the cost amount for a given rank.
func RenameCost(rank int) (string, float64) {
	switch {
	case rank <= 2:
		return "free", 0
	case rank == 3:
		return "primeval", 5000
	case rank == 4:
		return "primeval", 20000
	case rank == 5:
		return "primeval", 100000
	case rank == 6:
		return "immortal", 100
	case rank == 7:
		return "immortal", 500
	case rank == 8:
		return "immortal", 2500
	case rank >= 9:
		return "immortal", 10000
	default:
		return "free", 0
	}
}

// ImmortalPenaltyRates returns the loss rates for dao marks, path masteries, aperture HP, and stress gain for a given immortal rank (6-9).
func ImmortalPenaltyRates(rank int) (daoLossRate float64, masteryLossRate float64, hpLossRate float64, stressGain float64) {
	switch rank {
	case 6:
		return 0.30, 0.30, 0.20, 30.0
	case 7:
		return 0.40, 0.40, 0.30, 50.0
	case 8:
		return 0.50, 0.50, 0.40, 80.0
	case 9:
		return 0.70, 0.70, 0.50, 100.0
	default:
		if rank > 9 {
			return 0.70, 0.70, 0.50, 100.0
		}
		return 0.30, 0.30, 0.20, 30.0
	}
}

// ========================================================================
// 7. TRINH SÁT / SUY DIỄN THIÊN CƠ (SCOUTING BALANCE)
// ========================================================================

// Trinh sát có 3 trụ, theo thứ tự ưu tiên giảm dần:
//  1. Cảnh giới lưu phái (path_masteries) — trụ số một
//  2. Cổ trùng (INVESTIGATION)
//  3. Đạo ngân (immortal_apertures.dao_marks)

// ScoutingAttainBase là base PHẲNG theo bậc cảnh giới (0-4), cộng thẳng vào
// realm_dao mà không nhân rank. Nhờ vậy tu vi là yếu tố phụ thuộc rank duy
// nhất, còn cảnh giới trả về giá trị ở mọi bậc.
var ScoutingAttainBase = [5]float64{0, 50, 140, 280, 470}

// ScoutingAttainMult là phần nhân nhẹ lên stat theo bậc cảnh giới. Giữ ở mức
// thấp (trần ×1.30) để đạo ngân không thể nhấn chìm cảnh giới.
var ScoutingAttainMult = [5]float64{1.00, 1.05, 1.12, 1.20, 1.30}

// ScoutingInvestigationWeight hệ số cho chỉ số INVESTIGATION của Cổ trang bị.
// Đối xứng với DISGUISE ×5 của phe thủ (xem ScoutingDisguiseWeight) để công và
// thủ cùng lượng Cổ thì cân bằng.
const ScoutingInvestigationWeight = 5.0

// ScoutingDisguiseWeight là hệ số cho chỉ số DISGUISE của Cổ trang bị (phe thủ).
const ScoutingDisguiseWeight = 5.0

// Đạo ngân là trụ yếu nhất: hệ số bão hòa, trần +20% stat.
const (
	ScoutingDaoCeiling = 0.20
	ScoutingDaoSoftcap = 3000.0
)

// ScoutingRankNormExp là số mũ của chuẩn hoá chênh rank áp lên delta cuối.
// Mũ 0.6 thay vì 1.0 để lệch 2 bậc là mất trắng nhưng không phản phệ ngay.
const ScoutingRankNormExp = 0.6

// ScoutingBackfireRatio là trần sát thương phản phệ tính theo % max HP / max
// Soul, thay cho công thức cũ |delta| × rank × 5 không trần (một lần dò có thể
// trừ hàng chục nghìn HP khi đối thủ không có gì).
const ScoutingBackfireRatio = 0.15

// ========================================================================
// 8. GIỚI VỰC BÍCH (REGIONAL WALL SUPPRESSION)
// ========================================================================

// RegionalWallDebuff calculates the debuff percentage (negative float64) and duration
// based on player rank and sub-rank when crossing between different NGU_VUC regions.
func RegionalWallDebuff(rank int, subRank int) (float64, time.Duration) {
	if rank < 1 {
		rank = 1
	}
	if subRank < 1 {
		subRank = 1
	}

	var basePct float64
	var baseDuration time.Duration

	switch rank {
	case 1:
		basePct = 0.10
		baseDuration = 5 * time.Hour
	case 2:
		basePct = 0.15
		baseDuration = 8 * time.Hour
	case 3:
		basePct = 0.20
		baseDuration = 10 * time.Hour
	case 4:
		basePct = 0.25
		baseDuration = 12 * time.Hour
	case 5:
		basePct = 0.30
		baseDuration = 15 * time.Hour
	case 6:
		basePct = 0.45
		baseDuration = 20 * time.Hour
	case 7:
		basePct = 0.55
		baseDuration = 25 * time.Hour
	case 8:
		basePct = 0.65
		baseDuration = 30 * time.Hour
	default: // Rank 9+
		basePct = 0.75
		baseDuration = 40 * time.Hour
	}

	subRankBonus := float64(subRank-1) * 0.01
	if rank >= 6 {
		subRankBonus = float64(subRank-1) * 0.02
	}

	totalDebuffPct := -(basePct + subRankBonus)
	if totalDebuffPct < -0.90 {
		totalDebuffPct = -0.90
	}

	return totalDebuffPct, baseDuration
}
