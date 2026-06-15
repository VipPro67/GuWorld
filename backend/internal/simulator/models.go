package simulator

// CombatLoadout represents the input data from the player before the battle starts.
type CombatLoadout struct {
	CoreGuId      string
	SupportGuIds  []string
	KillerMoveIds []string
	ImmortalGuId  *string
}

// Effect represents an active buff or debuff on a combatant.
type Effect struct {
	ID       string // e.g. "poison", "shield", "stun"
	Type     string // "buff", "debuff"
	Value    int
	Duration int // Remaining turns
}

// DaoMarks represents the dao marks of a combatant, grouped by path.
type DaoMarks map[string]int

// CombatantAuto represents a stripped-down, deterministic entity in the combat grid.
type CombatantAuto struct {
	ID       string
	Name     string
	Kind     string // "Player", "Enemy", "Boss", "Minion"
	GridRow  int    // 1 for Frontline, 2 for Backline
	MaxHP    int
	HP       int
	MaxMana  int
	Mana     int
	BaseDmg  int
	BaseDef  int
	
	// Pre-combat Auto-Barrier
	VirtualShield int
	
	DaoMarks DaoMarks

	// Equipment / Loadout
	Loadout *CombatLoadout
	
	// Active effects (Shields, Stuns, etc.)
	ActiveEffects []Effect
	
	// Hidden state
	IsDead bool
}

// CombatState represents the mutable snapshot of the battlefield at any given moment.
type CombatState struct {
	Turn     int
	Seed     int64
	Allies   []*CombatantAuto
	Enemies  []*CombatantAuto
	
	// Boss specific tracking
	BossPhase int // 0 = 100-70%, 1 = 70-30%, 2 = 30-0%
}

// CombatLog represents a granular event to be stringified as JSON.
type CombatLog struct {
	Turn      int    `json:"turn"`
	ActorId   string `json:"actor_id"`
	ActorName string `json:"actor_name"`
	Action    string `json:"action"`   // e.g. "CAST_KILLER_MOVE", "BASIC_ATTACK", "TRIGGERED", "PHASE_TRANSITION"
	TargetId  string `json:"target_id,omitempty"`
	Damage    int    `json:"damage,omitempty"`
	IsCrit    bool   `json:"is_crit,omitempty"`
	Trigger   string `json:"trigger,omitempty"` // Name of the fixed trigger e.g. "THIET_HUYET_CO"
	Detail    string `json:"detail,omitempty"`  // Human readable flavor text
}
