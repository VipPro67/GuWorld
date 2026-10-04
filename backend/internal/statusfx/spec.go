package statusfx

import (
	"math/rand"
	"strings"
)

// StatusOp is the normalized, resolved representation of a single status
// application produced from a Gu/killer-move active effect.
//
// Value and Turns carry the catalog semantics: Value is the intensity (DoT
// damage per tick, or percent), Turns is the real duration in turns.
type StatusOp struct {
	ID     string
	Value  float64
	Turns  int
	Chance float64 // proc chance per application (0..1); 0 = always applies
	Unit   StatusUnit
}

type StatusUnit string

const (
	UnitFlat    StatusUnit = "flat"
	UnitPercent StatusUnit = "percent"
)

// StatusSpec declares how a status behaves. It is the single source of truth
// the simulator consults to apply/tick a status, so new statuses are a data
// entry plus (optionally) one behaviour hook.
type StatusSpec struct {
	ID            string
	Behaviour     string // see Behaviour* constants
	Label         string // Vietnamese display name
	Category      StatusCategory
	Path          string  // Dao path this status belongs to (e.g., "blood", "fire", "ice", "dream", etc.)
	BaseChance    float64 // Base proc chance 0..1 (T1: 0.3, T2: 0.45, T3: 0.6, T4: 0.75, T5: 0.9)
	SelfTarget    bool    // buff applied to the caster instead of the target
	ResistGate    bool    // subject to MentalResistance
	BreakOnDamage bool    // removed when the combatant takes damage (sleep)
	SkipTurn      bool    // combatant loses its turn (sleep/bind)
	ForceBasic    bool    // only basic attacks allowed (fear)
	Untargetable  bool    // cannot be selected by single-target actions (stealth)
	Redirect      bool    // forces attackers to redirect (charm)
	Accuracy      float64 // miss/accuracy penalty 0..1 (blind)
	DefaultTurns  int     // fallback duration when CSV has none
	DirectHP      bool    // true if status/DoT damage pierces shield to deal direct HP damage
}

// Registry maps a canonical status ID to its spec.
var Registry = map[string]StatusSpec{
	// ── Physical / Elemental DoT & HoT ──
	StatusBleed: {
		ID: StatusBleed, Behaviour: BehaviourDot, Label: "Xuất Huyết",
		Category: CategoryPhysicalDoT, Path: "blood", BaseChance: 0.6,
		DefaultTurns: 3,
	},
	StatusPoison: {
		ID: StatusPoison, Behaviour: BehaviourDot, Label: "Độc",
		Category: CategoryPhysicalDoT, Path: "poison", BaseChance: 0.6,
		DefaultTurns: 3,
	},
	StatusBurn: {
		ID: StatusBurn, Behaviour: BehaviourDot, Label: "Thiêu Đốt",
		Category: CategoryPhysicalDoT, Path: "fire", BaseChance: 0.6,
		DefaultTurns: 3,
	},
	StatusSlowDamage: {
		ID: StatusSlowDamage, Behaviour: BehaviourDot, Label: "Tê Cóng",
		Category: CategoryPhysicalDoT, Path: "ice", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusRegen: {
		ID: StatusRegen, Behaviour: BehaviourRegen, Label: "Hồi Phục",
		Category: CategoryBuff, Path: "healing", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 3,
	},

	// ── Crowd Control (CC) & Mental Debuffs ──
	StatusStun: {
		ID: StatusStun, Behaviour: BehaviourStun, Label: "Choáng",
		Category: CategoryMentalCC, Path: "soul", BaseChance: 0.6,
		ResistGate: true, SkipTurn: true, DefaultTurns: 1,
	},
	StatusFreeze: {
		ID: StatusFreeze, Behaviour: BehaviourFreeze, Label: "Đóng Băng",
		Category: CategoryMentalCC, Path: "ice", BaseChance: 0.6,
		ResistGate: true, SkipTurn: true, BreakOnDamage: true, DefaultTurns: 1,
	},
	StatusSleep: {
		ID: StatusSleep, Behaviour: BehaviourSleep, Label: "Ngủ",
		Category: CategoryMentalCC, Path: "dream", BaseChance: 0.6,
		ResistGate: true, SkipTurn: true, BreakOnDamage: true, DefaultTurns: 2,
	},
	StatusBind: {
		ID: StatusBind, Behaviour: BehaviourBind, Label: "Trói",
		Category: CategoryMentalCC, Path: "earth", BaseChance: 0.6,
		ResistGate: true, SkipTurn: true, DefaultTurns: 3,
	},
	StatusSilence: {
		ID: StatusSilence, Behaviour: BehaviourSilence, Label: "Khóa Chiêu",
		Category: CategoryMentalCC, Path: "wisdom", BaseChance: 0.6,
		ResistGate: true, ForceBasic: true, DefaultTurns: 2,
	},
	StatusFear: {
		ID: StatusFear, Behaviour: BehaviourForceBasic, Label: "Khiếp Sợ",
		Category: CategoryMentalCC, Path: "soul", BaseChance: 0.6,
		ResistGate: true, ForceBasic: true, DefaultTurns: 2,
	},
	StatusConfuse: {
		ID: StatusConfuse, Behaviour: BehaviourConfuse, Label: "Mê Loạn",
		Category: CategoryMentalCC, Path: "dream", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 2,
	},
	StatusCharm: {
		ID: StatusCharm, Behaviour: BehaviourCharm, Label: "Mê Hoặc",
		Category: CategoryMentalCC, Path: "charm", BaseChance: 0.6,
		ResistGate: true, Redirect: true, DefaultTurns: 3,
	},
	StatusBlind: {
		ID: StatusBlind, Behaviour: BehaviourBlind, Label: "Mù",
		Category: CategoryMentalCC, Path: "light", BaseChance: 0.6,
		ResistGate: true, Accuracy: 0.35, DefaultTurns: 2,
	},
	StatusWeak: {
		ID: StatusWeak, Behaviour: BehaviourWeak, Label: "Suy Yếu",
		Category: CategoryMentalCC, Path: "strength", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusSlow: {
		ID: StatusSlow, Behaviour: BehaviourSlow, Label: "Làm Chậm",
		Category: CategoryMentalCC, Path: "time", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusSlow1: {
		ID: StatusSlow1, Behaviour: BehaviourSlow, Label: "Làm Chậm Tuyệt Đối",
		Category: CategoryBuff, Path: "time", BaseChance: 1.0,
		ResistGate: false, DefaultTurns: 3,
	},

	// ── Buffs & Combat Stances ──
	StatusAtkUp: {
		ID: StatusAtkUp, Behaviour: BehaviourAtkUp, Label: "Tăng Công",
		Category: CategoryBuff, Path: "strength", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 3,
	},
	StatusDefUp: {
		ID: StatusDefUp, Behaviour: BehaviourDefUp, Label: "Tăng Thủ",
		Category: CategoryBuff, Path: "metal", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 3,
	},
	StatusThorns: {
		ID: StatusThorns, Behaviour: BehaviourThorns, Label: "Phản Sát Thương",
		Category: CategoryBuff, Path: "wood", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 3,
	},
	StatusTaunt: {
		ID: StatusTaunt, Behaviour: BehaviourTaunt, Label: "Khiêu Khích",
		Category: CategoryBuff, Path: "earth", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 2,
	},
	StatusStealth: {
		ID: StatusStealth, Behaviour: BehaviourStealth, Label: "Ẩn Thân",
		Category: CategoryBuff, Path: "shadow", BaseChance: 1.0,
		SelfTarget: true, Untargetable: true, DefaultTurns: 2,
	},
	StatusHaste: {
		ID: StatusHaste, Behaviour: BehaviourHaste, Label: "Tăng Tốc",
		Category: CategoryBuff, Path: "wind", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 3,
	},
	StatusArmorBreak: {
		ID: StatusArmorBreak, Behaviour: BehaviourArmorBreak, Label: "Phá Giáp",
		Category: CategoryBuff, Path: "metal", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusDamageReduction: {
		ID: StatusDamageReduction, Behaviour: BehaviourDamageReduction, Label: "Giảm Sát Thương",
		Category: CategoryBuff, Path: "earth", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 3,
	},
	// INVINCIBLE is Bất Hoại: blocks every damage path, including DoT and
	// %MaxHP riders that never reach applyDamageToTarget. CategoryBuff keeps it
	// non-dispellable. Strength is fixed (Value 100 is the sentinel for
	// DamageTakenReduction >= 1.0); op.Value is ignored.
	StatusInvincible: {
		ID: StatusInvincible, Behaviour: BehaviourInvincible, Label: "Bất Hoại",
		Category: CategoryBuff, Path: "void", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 1,
	},
	// LAST_STAND is Bất Tử: damage lands normally, but HP cannot fall below
	// LastStandMinHP while the buff is up. CategoryImmortal (not CategoryBuff)
	// makes it dispellable, giving cleanse a counterplay to a save effect.
	StatusLastStand: {
		ID: StatusLastStand, Behaviour: BehaviourLastStand, Label: "Bất Tử",
		Category: CategoryImmortal, Path: "rules", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 1,
	},
	// EVASION is the only dodge source in combat. Self buff, never dispellable,
	// never gated by CC resist; its proc chance comes from the apply_status
	// entry (rank-scaled in combat_auto_resolve) and its strength from op.Value
	// (percent). The simulator merges multiple EVASION effects by max, capped
	// at 0.75 in ComputeEffectMultipliers.
	StatusEvasion: {
		ID: StatusEvasion, Behaviour: BehaviourEvasion, Label: "Né Tránh",
		Category: CategoryBuff, Path: "void", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 1,
	},
	StatusReviveLock: {
		ID: StatusReviveLock, Behaviour: BehaviourReviveLock, Label: "Khóa Hồi Sinh",
		Category: CategoryImmortal, Path: "rules", BaseChance: 1.0,
		ResistGate: true, DefaultTurns: 3,
	},

	// ── Gu World Dao Mechanics ──
	StatusEssenceLeak: {
		ID: StatusEssenceLeak, Behaviour: BehaviourEssenceLeak, Label: "Ô Khiếu / Rò Rỉ Chân Nguyên",
		Category: CategoryAperture, Path: "earth", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusGuSeal: {
		ID: StatusGuSeal, Behaviour: BehaviourGuSeal, Label: "Phong Cổ / Trầm Miên",
		Category: CategoryAperture, Path: "refinement", BaseChance: 0.5,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusSoulWound: {
		ID: StatusSoulWound, Behaviour: BehaviourSoulWound, Label: "Hồn Thương",
		Category: CategoryDaoConflict, Path: "soul", BaseChance: 0.5,
		ResistGate: true, DefaultTurns: 3, DirectHP: true,
	},
	StatusThoughtChaos: {
		ID: StatusThoughtChaos, Behaviour: BehaviourThoughtChaos, Label: "Tạp Niệm / Loạn Ý",
		Category: CategoryAperture, Path: "wisdom", BaseChance: 0.5,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusDaoConflict: {
		ID: StatusDaoConflict, Behaviour: BehaviourDaoConflict, Label: "Xung Đột Đạo Ngân",
		Category: CategoryDaoConflict, Path: "rules", BaseChance: 0.5,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusBacklash: {
		ID: StatusBacklash, Behaviour: BehaviourBacklash, Label: "Phản Phệ",
		Category: CategoryImmortal, Path: "rules", BaseChance: 1.0,
		ResistGate: false, DefaultTurns: 1,
	},

	// Lãng Quên statuses (Wisdom Path). FORGET_EGO seals the victim's Gu
	// (BehaviourGuSeal, cant use Gu skills); FORGET_REALITY weakens all
	// damage they deal (BehaviourWeak, intensity from
	// active.target_damage_reduce_percent).
	StatusForgetEgo: {
		ID: StatusForgetEgo, Behaviour: BehaviourGuSeal, Label: "Lãng Quên Bản Ngã",
		Category: CategoryMentalCC, Path: "wisdom", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 1,
	},
	StatusForgetReality: {
		ID: StatusForgetReality, Behaviour: BehaviourWeak, Label: "Lãng Quên Thực Tại",
		Category: CategoryMentalCC, Path: "wisdom", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 2,
	},
}

// Spec returns the spec for a status ID (case-insensitive) and whether it is
// known to the registry.
func Spec(id string) (StatusSpec, bool) {
	s, ok := Registry[strings.ToUpper(strings.TrimSpace(id))]
	return s, ok
}

// TierToPercent converts a tier (1-5) to a percentage value for the status.
func TierToPercent(effectType string, tier int) float64 {
	if tier < 1 {
		tier = 1
	}
	if tier > 5 {
		tier = 5
	}
	switch strings.ToUpper(strings.TrimSpace(effectType)) {
	case StatusSlow, StatusSlow1, StatusSlowDamage:
		return []float64{0, 15, 25, 35, 45, 55}[tier]
	case StatusWeak:
		return []float64{0, 10, 20, 30, 40, 50}[tier]
	case StatusFreeze, StatusSilence, StatusStun:
		return []float64{0, 1, 1, 2, 2, 3}[tier]
	case StatusThorns:
		return []float64{0, 10, 15, 20, 25, 30}[tier]
	case StatusRegen:
		return []float64{0, 3, 5, 8, 12, 16}[tier]
	case StatusAtkUp, StatusDefUp, StatusArmorBreak, StatusDamageReduction:
		return []float64{0, 10, 15, 20, 25, 30}[tier]
	case StatusTaunt:
		return []float64{0, 1, 2, 2, 3, 3}[tier]
	default:
		return float64(tier * 10)
	}
}

// TierToDuration returns the duration (in turns) for an effect at given tier.
func TierToDuration(effectType string, tier int) int {
	if tier < 1 {
		tier = 1
	}
	if tier > 5 {
		tier = 5
	}
	switch strings.ToUpper(strings.TrimSpace(effectType)) {
	case StatusSlow, StatusSlow1, StatusSlowDamage:
		return []int{0, 3, 3, 4, 4, 5}[tier]
	case StatusWeak:
		return []int{0, 3, 3, 4, 4, 5}[tier]
	case StatusFreeze:
		return []int{0, 1, 1, 2, 2, 3}[tier]
	case StatusSilence:
		return []int{0, 2, 2, 3, 3, 4}[tier]
	case StatusThorns:
		return []int{0, 3, 3, 4, 4, 5}[tier]
	case StatusRegen:
		return []int{0, 3, 4, 4, 5, 5}[tier]
	case StatusAtkUp, StatusDefUp, StatusArmorBreak, StatusDamageReduction:
		return []int{0, 3, 3, 4, 4, 5}[tier]
	case StatusStun:
		return []int{0, 1, 1, 2, 2, 3}[tier]
	case StatusTaunt:
		return []int{0, 2, 2, 3, 3, 4}[tier]
	default:
		return 3
	}
}

// ResolveStatus normalizes a raw apply_status + value/duration pair into a
// StatusOp using catalog semantics. Unknown IDs return an error so they are
// surfaced instead of silently dropped.
func ResolveStatus(id string, value float64, turns int) (StatusOp, error) {
	spec, ok := Spec(id)
	if !ok {
		return StatusOp{}, &UnknownStatusError{ID: id}
	}
	// If duration is 0, check default or derive from tier
	if turns <= 0 {
		turns = spec.DefaultTurns
	}
	unit := UnitPercent
	if spec.Behaviour == BehaviourDot || spec.Behaviour == BehaviourRegen || spec.DirectHP {
		unit = UnitFlat
	}
	return StatusOp{ID: spec.ID, Value: value, Turns: turns, Unit: unit}, nil
}

// ResolveStatusWithTier normalizes a status application when specified by tier (1-5).
func ResolveStatusWithTier(id string, tier int) (StatusOp, error) {
	spec, ok := Spec(id)
	if !ok {
		return StatusOp{}, &UnknownStatusError{ID: id}
	}
	val := TierToPercent(spec.ID, tier)
	dur := TierToDuration(spec.ID, tier)
	unit := UnitPercent
	if spec.Behaviour == BehaviourDot || spec.Behaviour == BehaviourRegen || spec.DirectHP {
		unit = UnitFlat
	}
	return StatusOp{ID: spec.ID, Value: val, Turns: dur, Unit: unit}, nil
}

// UnknownStatusError reports an apply_status key not present in the registry.
type UnknownStatusError struct {
	ID string
}

func (e *UnknownStatusError) Error() string {
	return "statusfx: unknown status " + e.ID
}

// RankPowerWeight returns the relative power scaling weight for a given rank
// based on the canonical power scale:
// Mortal: Rank 1..5 = {1, 2, 5, 20, 100}
// Immortal: Rank 6..9 = {1000, 10000, 100000, 1000000} (10x step from mortal peak)
func RankPowerWeight(rank int) float64 {
	weights := [...]float64{
		1: 1,
		2: 2,
		3: 5,
		4: 20,
		5: 100,
		6: 1000,
		7: 10000,
		8: 100000,
		9: 1000000,
	}
	if rank < 1 {
		return 1
	}
	if rank >= len(weights) {
		return weights[len(weights)-1]
	}
	return weights[rank]
}

// CasterAttenuationFactor calculates the attenuation coefficient when a high-rank
// caster uses a lower-rank Gu to buff/heal/protect themselves or allies.
// Returns a factor between 0.0001 and 1.0.
func CasterAttenuationFactor(guRank, casterRank int) float64 {
	if casterRank <= guRank {
		return 1.0
	}
	wGu := RankPowerWeight(guRank)
	wCaster := RankPowerWeight(casterRank)
	factor := wGu / wCaster
	if factor < 0.0001 {
		return 0.0001
	}
	return factor
}

// TargetSuppressionFactor calculates the realm suppression when a lower-rank Gu
// targets a higher-rank opponent with hostile effects (debuffs, CC, DoTs, direct HP).
// Returns chanceFactor (0.0 to 1.0) and durationPenalty (turns reduced).
// When targetRank - guRank >= 5, chanceFactor is 0.0 (immune).
func TargetSuppressionFactor(guRank, targetRank int) (chanceFactor float64, durationPenalty int) {
	if targetRank <= guRank {
		return 1.0, 0
	}
	gap := targetRank - guRank
	durationPenalty = gap / 2
	if gap >= 5 {
		return 0.0, durationPenalty
	}
	factor := 1.0 - 0.20*float64(gap)
	if factor < 0 {
		factor = 0
	}
	return factor, durationPenalty
}

// CanApplySelfStatus tính toán xác suất kích hoạt hiệu ứng lên bản thân (Self-Target Buff/Heal)
// dựa trên Cảnh Giới Nhân Vật (Player Rank) và Rank Gu, có tính đến Caster Attenuation khi casterRank > guRank.
func CanApplySelfStatus(rng *rand.Rand, casterPlayerRank, guRank int, spec StatusSpec, baseChance float64) bool {
	chance := baseChance
	if chance <= 0 {
		chance = spec.BaseChance
	}
	if chance <= 0 {
		chance = 1.0
	}
	// Tăng 5% xác suất với mỗi cấp Gu Rank trên Rank 1
	if guRank > 1 {
		chance += 0.05 * float64(guRank-1)
	}
	// Áp dụng Caster Attenuation khi casterPlayerRank > guRank:
	if casterPlayerRank > guRank {
		attenuation := CasterAttenuationFactor(guRank, casterPlayerRank)
		chance *= attenuation
	}
	if chance >= 1.0 {
		return true
	}
	if chance <= 0.0 {
		return false
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	return rng.Float64() < chance
}

// CanApplyStatusWithGuRank tính toán xác suất tác động debuff/CC lên đối phương dựa trên
// Cảnh Giới người dùng (casterRank), Cảnh Giới đối phương (targetRank), Rank Gu (guRank) và Đạo Ngân.
func CanApplyStatusWithGuRank(rng *rand.Rand, casterRank, targetRank, guRank int, casterDaoMarks, targetDaoMarks map[string]int64, spec StatusSpec, targetStatusResist float64, opChance float64) bool {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	baseChance := spec.BaseChance
	if opChance > 0 {
		baseChance = opChance
	}
	if guRank > 1 {
		baseChance += 0.05 * float64(guRank-1)
	}
	if baseChance > 0.95 {
		baseChance = 0.95
	}

	// 1. Áp chế cảnh giới từ chênh lệch TargetRank vs GuRank
	suppressionFactor, _ := TargetSuppressionFactor(guRank, targetRank)
	if suppressionFactor <= 0 {
		return false // Miễn dịch hoàn toàn
	}

	// 2. Hệ số chênh lệch Chuyển (CasterRank vs TargetRank)
	rankDiff := casterRank - targetRank
	rankFactor := 1.0
	switch {
	case rankDiff >= 2:
		rankFactor = 1.5
	case rankDiff == 1:
		rankFactor = 1.2
	case rankDiff == -1:
		rankFactor = 0.6
	case rankDiff <= -2:
		rankFactor = 0.1
	}

	// Tiên Cổ (Rank 6+) bảo lưu uy lực tối thiểu
	if casterRank >= 6 && rankFactor < 1.0 {
		rankFactor = 1.0
	}
	// Tiên Cổ đánh vào Phàm Cổ (Rank <= 5) tự động nâng lên 1.5
	if casterRank >= 6 && targetRank <= 5 {
		rankFactor = 1.5
	}

	// 3. Chênh lệch Đạo Ngân theo Lưu Phái
	daoMultiplier := 1.0
	if casterDaoMarks != nil || targetDaoMarks != nil {
		cMark := int64(0)
		tMark := int64(0)
		if casterDaoMarks != nil {
			cMark = casterDaoMarks[spec.Path]
		}
		if targetDaoMarks != nil {
			tMark = targetDaoMarks[spec.Path]
		}
		daoDiff := cMark - tMark
		daoMultiplier = 1.0 + (float64(daoDiff) / 100.0)
		if daoMultiplier < 0.2 {
			daoMultiplier = 0.2
		}
	}

	// 4. Tính toán xác suất cuối cùng
	finalProcChance := baseChance * suppressionFactor * daoMultiplier * rankFactor * (1.0 - targetStatusResist)
	if finalProcChance >= 1.0 {
		return true
	}
	if finalProcChance <= 0.0 {
		return false
	}
	return rng.Float64() < finalProcChance
}

// CanApplyStatus tính toán xác suất thành công dựa trên Cảnh Giới (Rank) và Đạo Ngân.
// Trả về true nếu hiệu ứng được áp dụng thành công.
func CanApplyStatus(rng *rand.Rand, casterRank, targetRank int, casterDaoMarks, targetDaoMarks map[string]int64, spec StatusSpec, targetStatusResist float64) bool {
	return CanApplyStatusWithGuRank(rng, casterRank, targetRank, 1, casterDaoMarks, targetDaoMarks, spec, targetStatusResist, 0)
}
