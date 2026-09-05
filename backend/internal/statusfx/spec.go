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
	ID    string
	Value float64
	Turns int
}

// StatusSpec declares how a status behaves. It is the single source of truth
// the simulator consults to apply/tick a status, so new statuses are a data
// entry plus (optionally) one behaviour hook.
type StatusSpec struct {
	ID            string
	Behaviour     string  // see Behaviour* constants
	Label         string  // Vietnamese display name
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
}

// Registry maps a canonical status ID to its spec.
var Registry = map[string]StatusSpec{
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
	StatusStealth: {
		ID: StatusStealth, Behaviour: BehaviourStealth, Label: "Ẩn Thân",
		Category: CategoryBuff, Path: "shadow", BaseChance: 1.0,
		SelfTarget: true, Untargetable: true, DefaultTurns: 2,
	},
	StatusSleep: {
		ID: StatusSleep, Behaviour: BehaviourSleep, Label: "Ngủ",
		Category: CategoryMentalCC, Path: "dream", BaseChance: 0.6,
		ResistGate: true, SkipTurn: true, BreakOnDamage: true, DefaultTurns: 2,
	},
	StatusBlind: {
		ID: StatusBlind, Behaviour: BehaviourBlind, Label: "Mù",
		Category: CategoryMentalCC, Path: "light", BaseChance: 0.6,
		ResistGate: true, Accuracy: 0.35, DefaultTurns: 2,
	},
	StatusCharm: {
		ID: StatusCharm, Behaviour: BehaviourCharm, Label: "Mê Hoặc",
		Category: CategoryMentalCC, Path: "charm", BaseChance: 0.6,
		ResistGate: true, Redirect: true, DefaultTurns: 3,
	},
	StatusBind: {
		ID: StatusBind, Behaviour: BehaviourBind, Label: "Trói",
		Category: CategoryMentalCC, Path: "earth", BaseChance: 0.6,
		ResistGate: true, SkipTurn: true, DefaultTurns: 3,
	},

	// Catalog statuses — registered so they resolve, not yet combat-wired.
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
	StatusInvincible: {
		ID: StatusInvincible, Behaviour: BehaviourInvincible, Label: "Bất Tử",
		Category: CategoryBuff, Path: "void", BaseChance: 1.0,
		SelfTarget: true, DefaultTurns: 1,
	},
	StatusReviveLock: {
		ID: StatusReviveLock, Behaviour: BehaviourReviveLock, Label: "Khóa Hồi Sinh",
		Category: CategoryImmortal, Path: "rules", BaseChance: 1.0,
		ResistGate: true, DefaultTurns: 3,
	},

	// Gu World Dao Mechanics - New Status Effects
	StatusEssenceLeak: {
		ID: StatusEssenceLeak, Behaviour: BehaviourAperture, Label: "Ô Khiếu / Rò Rỉ Chân Nguyên",
		Category: CategoryAperture, Path: "earth", BaseChance: 0.6,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusGuSeal: {
		ID: StatusGuSeal, Behaviour: BehaviourAperture, Label: "Phong Cổ / Trầm Miên",
		Category: CategoryAperture, Path: "refinement", BaseChance: 0.5,
		ResistGate: true, DefaultTurns: 3,
	},
	StatusSoulWound: {
		ID: StatusSoulWound, Behaviour: BehaviourSoulWound, Label: "Hồn Thương",
		Category: CategoryDaoConflict, Path: "soul", BaseChance: 0.5,
		ResistGate: true, DefaultTurns: 3,
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
}

// Spec returns the spec for a status ID (case-insensitive) and whether it is
// known to the registry.
func Spec(id string) (StatusSpec, bool) {
	s, ok := Registry[strings.ToUpper(strings.TrimSpace(id))]
	return s, ok
}

// ResolveStatus normalizes a raw apply_status + value/duration pair into a
// StatusOp using catalog semantics. Unknown IDs return an error so they are
// surfaced instead of silently dropped.
func ResolveStatus(id string, value float64, turns int) (StatusOp, error) {
	spec, ok := Spec(id)
	if !ok {
		return StatusOp{}, &UnknownStatusError{ID: id}
	}
	if turns <= 0 {
		turns = spec.DefaultTurns
	}
	return StatusOp{ID: spec.ID, Value: value, Turns: turns}, nil
}

// UnknownStatusError reports an apply_status key not present in the registry.
type UnknownStatusError struct {
	ID string
}

func (e *UnknownStatusError) Error() string {
	return "statusfx: unknown status " + e.ID
}

// CanApplyStatus tính toán xác suất thành công dựa trên Cảnh Giới (Rank) và Đạo Ngân.
// Trả về true nếu hiệu ứng được áp dụng thành công.
func CanApplyStatus(casterRank, targetRank int, casterDaoMarks, targetDaoMarks map[string]int64, spec StatusSpec, targetStatusResist float64) bool {
	// 1. Hệ số chênh lệch Chuyển (Rank)
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

	// 2. Chênh lệch Đạo Ngân theo Lưu Phái
	daoDiff := casterDaoMarks[spec.Path] - targetDaoMarks[spec.Path]
	daoMultiplier := 1.0 + (float64(daoDiff) / 100.0)
	if daoMultiplier < 0.2 {
		daoMultiplier = 0.2
	}

	// 3. Tính toán xác suất cuối cùng
	finalProcChance := spec.BaseChance * daoMultiplier * rankFactor * (1.0 - targetStatusResist)

	// Clamp to [0, 1]
	if finalProcChance > 1.0 {
		finalProcChance = 1.0
	}
	if finalProcChance < 0.0 {
		finalProcChance = 0.0
	}

	return rand.Float64() < finalProcChance
}