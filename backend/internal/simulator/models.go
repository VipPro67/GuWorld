package simulator

// CombatLoadout represents the input data from the player before the battle starts.
type CombatLoadout struct {
	CoreGuId      string
	SupportGuIds  []string
	KillerMoveIds []string
	ImmortalGuId  *string
}

// SimAction is a pre-computed action from a Gu, killer move, or basic attack.
// It is produced by the handler using CalculateGuEffectForCombatant + BuildCombatAction
// and consumed by the simulator engine.
type SimAction struct {
	ActionType   string  // "BASIC_ATTACK" | "GU_SKILL" | "KILLER_MOVE" | "HEAL" | "SHIELD"
	Name         string  // display name
	Damage       float64 // raw damage before defense mitigation
	Defense      float64 // shield value granted to self
	Healing      float64 // HP restored to self
	ManaCost     float64 // mana consumed on use
	StunTurns    int     // turns of stun applied to target
	BleedPerTurn float64 // bleed DoT per turn (applied to target)
	PoisonPerTurn float64 // poison DoT per turn (applied to target)
	LifeStealPct float64 // percentage of damage dealt healed back (0-1)
	MultiHit     int     // number of hits (chain_damage compound)
	PhantomDmg   float64 // bonus phantom damage
	ThornsDmg    float64 // reflect damage to attacker
}

// Effect represents an active buff or debuff on a combatant.
type Effect struct {
	ID       string // e.g. "poison", "shield", "stun"
	Type     string // "buff", "debuff"
	Value    int
	Duration int // remaining turns
}

// DoTEffect tracks a damage-over-time effect on a combatant.
type DoTEffect struct {
	Type  string  // "bleed", "poison"
	Value float64 // damage per turn
	Turns int     // remaining turns
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
	BaseDmg  int // fallback basic-attack damage
	BaseDef  int

	// Pre-combat Auto-Barrier
	VirtualShield int

	DaoMarks DaoMarks

	// Equipment / Loadout
	Loadout *CombatLoadout

	// Pre-computed actions (produced by handler, consumed by simulator)
	Actions []SimAction

	// Active effects (Shields, Stuns, etc.)
	ActiveEffects []Effect

	// Damage-over-time effects
	ActiveDoT []DoTEffect

	// Internal action index for round-robin
	nextActionIdx int

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
