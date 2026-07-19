package simulator

// CombatLoadout represents the input data from the player before the battle starts.
type CombatLoadout struct {
	GuIds         []string  `json:"GuIds"`
	KillerMoveIds []string  `json:"KillerMoveIds"`
	ImmortalGuId  *string   `json:"ImmortalGuId,omitempty"`

	AutoHealHP      bool    `json:"AutoHealHP"`
	HealThreshold   float64 `json:"HealThreshold"`
	AutoRestoreMana bool    `json:"AutoRestoreMana"`
	ManaThreshold   float64 `json:"ManaThreshold"`
}

// SimAction is a pre-computed action from a Gu, killer move, or basic attack.
// It is produced by the handler using CalculateGuEffectForCombatant + BuildCombatAction
// and consumed by the simulator engine.
type SimAction struct {
	ActionType   string  // "BASIC_ATTACK" | "GU_SKILL" | "KILLER_MOVE" | "HEAL" | "SHIELD"
	Name         string  // display name
	SourceID     string  // GuID or KillerMoveID for tactic matching
	Damage       float64 // raw damage before defense mitigation
	DamageMult   float64 // damage multiplier (applied on top of Damage)
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

	// Backlash from using Gu rank > actor rank
	BacklashDamage float64 // HP damage taken by caster from Gu backlash

	// Targeting (AOE support)
	TargetType string // "SINGLE" | "ROW" | "COL" | "CROSS" | "ALL"
	TargetX    int    // preferred grid X (1=Front, 2=Mid, 3=Back)
	TargetY    int    // preferred grid Y (1=Top, 2=Mid, 3=Bottom)

	// Priority (lower = higher priority, 0 = highest). Used by engine for action ordering.
	Priority int

	// Conditional-use fields. Empty condition = always use.
	UseCondition   string  // "always","hp_below","mana_above","enemy_hp_below"
	ConditionValue float64
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
	GridX    int    // 1=Front, 2=Mid, 3=Back
	GridY    int    // 1=Top, 2=Mid, 3=Bottom
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

	// Auto-use settings
	AutoHealHP      bool
	HealThreshold   float64
	AutoRestoreMana bool
	ManaThreshold   float64

	// Battle inventory
	HealCharges    int     // how many healing items available
	HealPotency    float64 // how much HP each heal restores
	PrimevalStones int
	ImmortalStones float64

	// Tracking
	ItemsUsedInBattle int

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

	// Virtual mode: illusion combat, auto-restore mana each turn
	VirtualMode bool
}

// CombatSnapshot captures full stats of all combatants at a point in time.
type CombatSnapshot struct {
	Turn      int                    `json:"turn"`
	Allies    []CombatantSnapshot   `json:"allies"`
	Enemies   []CombatantSnapshot   `json:"enemies"`
}

// CombatantSnapshot is a point-in-time snapshot of one combatant's stats.
type CombatantSnapshot struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`
	HP       int     `json:"hp"`
	MaxHP    int     `json:"max_hp"`
	Mana     int     `json:"mana"`
	MaxMana  int     `json:"max_mana"`
	GridX    int     `json:"grid_x"`
	GridY    int     `json:"grid_y"`
	Shield   int     `json:"shield"`
	Damage   float64 `json:"damage"`
	Defense  float64 `json:"defense"`
	Speed    float64 `json:"speed"`
	IsDead   bool    `json:"is_dead"`
}

// CombatLog represents a granular event to be stringified as JSON.
type CombatLog struct {
	Turn      int              `json:"turn"`
	ActorId   string           `json:"actor_id"`
	ActorName string           `json:"actor_name"`
	Action    string           `json:"action"`   // e.g. "CAST_KILLER_MOVE", "BASIC_ATTACK", "TRIGGERED", "PHASE_TRANSITION"
	TargetId  string           `json:"target_id,omitempty"`
	Damage    int              `json:"damage,omitempty"`
	IsCrit    bool             `json:"is_crit,omitempty"`
	Trigger   string           `json:"trigger,omitempty"` // Name of the fixed trigger e.g. "THIET_HUYET_CO"
	Detail    string           `json:"detail,omitempty"`  // Human readable flavor text
	Snapshot  *CombatSnapshot  `json:"snapshot,omitempty"`
}
