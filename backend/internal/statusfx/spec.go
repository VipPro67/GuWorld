package statusfx

import "strings"

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
		DefaultTurns: 3,
	},
	StatusPoison: {
		ID: StatusPoison, Behaviour: BehaviourDot, Label: "Độc",
		DefaultTurns: 3,
	},
	StatusBurn: {
		ID: StatusBurn, Behaviour: BehaviourDot, Label: "Thiêu Đốt",
		DefaultTurns: 3,
	},
	StatusFear: {
		ID: StatusFear, Behaviour: BehaviourForceBasic, Label: "Khiếp Sợ",
		ResistGate: true, ForceBasic: true, DefaultTurns: 2,
	},
	StatusConfuse: {
		ID: StatusConfuse, Behaviour: BehaviourConfuse, Label: "Mê Loạn",
		ResistGate: true, DefaultTurns: 2,
	},
	StatusStealth: {
		ID: StatusStealth, Behaviour: BehaviourStealth, Label: "Ẩn Thân",
		SelfTarget: true, Untargetable: true, DefaultTurns: 2,
	},
	StatusSleep: {
		ID: StatusSleep, Behaviour: BehaviourSleep, Label: "Ngủ",
		ResistGate: true, SkipTurn: true, BreakOnDamage: true, DefaultTurns: 2,
	},
	StatusBlind: {
		ID: StatusBlind, Behaviour: BehaviourBlind, Label: "Mù",
		ResistGate: true, Accuracy: 0.35, DefaultTurns: 2,
	},
	StatusCharm: {
		ID: StatusCharm, Behaviour: BehaviourCharm, Label: "Mê Hoặc",
		ResistGate: true, Redirect: true, DefaultTurns: 3,
	},
	StatusBind: {
		ID: StatusBind, Behaviour: BehaviourBind, Label: "Trói",
		ResistGate: true, SkipTurn: true, DefaultTurns: 3,
	},

	// Catalog statuses — registered so they resolve, not yet combat-wired.
	StatusHaste: {
		ID: StatusHaste, Behaviour: BehaviourHaste, Label: "Tăng Tốc",
		SelfTarget: true, DefaultTurns: 3,
	},
	StatusArmorBreak: {
		ID: StatusArmorBreak, Behaviour: BehaviourArmorBreak, Label: "Phá Giáp",
		ResistGate: true, DefaultTurns: 3,
	},
	StatusDamageReduction: {
		ID: StatusDamageReduction, Behaviour: BehaviourDamageReduction, Label: "Giảm Sát Thương",
		SelfTarget: true, DefaultTurns: 3,
	},
	StatusInvincible: {
		ID: StatusInvincible, Behaviour: BehaviourInvincible, Label: "Bất Tử",
		SelfTarget: true, DefaultTurns: 1,
	},
	StatusReviveLock: {
		ID: StatusReviveLock, Behaviour: BehaviourReviveLock, Label: "Khóa Hồi Sinh",
		ResistGate: true, DefaultTurns: 3,
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
