package simulator

import "github.com/vippro67/GuWorld/backend/internal/statusfx"

// CombatLoadout represents the input data from the player before the battle starts.
type CombatLoadout struct {
	GuIds         []string `json:"GuIds"`
	KillerMoveIds []string `json:"KillerMoveIds"`
	ImmortalGuId  *string  `json:"ImmortalGuId,omitempty"`

	AutoHealHP      bool    `json:"AutoHealHP"`
	HealThreshold   float64 `json:"HealThreshold"`
	AutoRestoreMana bool    `json:"AutoRestoreMana"`
	ManaThreshold   float64 `json:"ManaThreshold"`
}

// ── Status Effect Type Constants ─────────────────────────────────────────
// Tier system: Value stores the tier (1-5), actual % computed by tierToPercent().
const (
	EffectSlow       = "slow"        // reduces speed → changes turn order (MentalResistance gated)
	EffectSlow1      = "slow1"       // reduces speed → guaranteed apply (no resist check)
	EffectSlowDamage = "slow_damage" // reduces speed and deals damage per turn
	EffectWeak       = "weak"        // reduces damage dealt
	EffectFreeze     = "freeze"      // like stun but can be broken by taking damage
	EffectSilence    = "silence"     // prevents skill use, only basic attack
	EffectThorns     = "thorns"      // reflects damage when hit
	EffectRegen      = "regen"       // heals HP each turn
	EffectAtkUp      = "atk_up"      // increases attack damage
	EffectDefUp      = "def_up"      // increases defense
	EffectTaunt      = "taunt"       // forces enemies to attack the taunter
	EffectStun       = "stun"        // skips turn entirely
	EffectBleed      = "bleed"       // DoT bleed
	EffectPoison     = "poison"      // DoT poison
)

// tierToPercent converts a tier (1-5) to a percentage value for the effect.
// slow:    T1=15%, T2=25%, T3=35%, T4=45%, T5=55% (MentalResistance gated)
// slow1:   T1=15%, T2=25%, T3=35%, T4=45%, T5=55% (guaranteed apply)
// weak:    T1=10%, T2=20%, T3=30%, T4=40%, T5=50%
// freeze:  T1=1t,  T2=1t,  T3=2t,  T4=2t,  T5=3t (duration)
// silence: T1=1t,  T2=1t,  T3=2t,  T4=2t,  T5=3t
// thorns:  T1=10%, T2=15%, T3=20%, T4=25%, T5=30% (of damage received)
// regen:   T1=3%,  T2=5%,  T3=8%,  T4=12%, T5=16% (of max HP per turn)
// atk_up:  T1=10%, T2=15%, T3=20%, T4=25%, T5=30%
// def_up:  T1=10%, T2=15%, T3=20%, T4=25%, T5=30%
// stun:    T1=1t,  T2=1t,  T3=2t,  T4=2t,  T5=3t
// taunt:   T1=1t,  T2=2t,  T3=2t,  T4=3t,  T5=3t
func TierToPercent(effectType string, tier int) float64 {
	if tier < 1 {
		tier = 1
	}
	if tier > 5 {
		tier = 5
	}
	switch effectType {
	case EffectSlow, EffectSlow1, EffectSlowDamage:
		return []float64{0, 15, 25, 35, 45, 55}[tier]
	case EffectWeak:
		return []float64{0, 10, 20, 30, 40, 50}[tier]
	case EffectFreeze, EffectSilence:
		return []float64{0, 1, 1, 2, 2, 3}[tier] // duration in turns
	case EffectThorns:
		return []float64{0, 10, 15, 20, 25, 30}[tier] // % of damage reflected
	case EffectRegen:
		return []float64{0, 3, 5, 8, 12, 16}[tier] // % of max HP
	case EffectAtkUp, EffectDefUp:
		return []float64{0, 10, 15, 20, 25, 30}[tier]
	case EffectStun:
		return []float64{0, 1, 1, 2, 2, 3}[tier]
	case EffectTaunt:
		return []float64{0, 1, 2, 2, 3, 3}[tier]
	default:
		return 0
	}
}

// tierToDuration returns the default duration (in turns) for an effect at given tier.
func TierToDuration(effectType string, tier int) int {
	if tier < 1 {
		tier = 1
	}
	if tier > 5 {
		tier = 5
	}
	switch effectType {
	case EffectSlow, EffectSlow1, EffectSlowDamage:
		return []int{0, 3, 3, 4, 4, 5}[tier]
	case EffectWeak:
		return []int{0, 3, 3, 4, 4, 5}[tier]
	case EffectFreeze:
		return []int{0, 1, 1, 2, 2, 3}[tier]
	case EffectSilence:
		return []int{0, 2, 2, 3, 3, 4}[tier]
	case EffectThorns:
		return []int{0, 3, 3, 4, 4, 5}[tier]
	case EffectRegen:
		return []int{0, 3, 4, 4, 5, 5}[tier]
	case EffectAtkUp, EffectDefUp:
		return []int{0, 3, 3, 4, 4, 5}[tier]
	case EffectStun:
		return []int{0, 1, 1, 2, 2, 3}[tier]
	case EffectTaunt:
		return []int{0, 2, 2, 3, 3, 4}[tier]
	default:
		return 1
	}
}

// SimAction is a pre-computed action from a Gu, killer move, or basic attack.
// It is produced by the handler using CalculateGuEffectForCombatant + BuildCombatAction
// and consumed by the simulator engine.
type SimAction struct {
	ActionType    string   // "BASIC_ATTACK" | "GU_SKILL" | "KILLER_MOVE" | "HEAL" | "SHIELD"
	Name          string   // display name
	SourceID      string   // GuID or KillerMoveID for tactic matching
	Path          []string // dao paths used to compute the dao mark bonus in the engine
	Rank          int      // rank of the Gu/Skill
	Damage        float64  // raw damage before defense mitigation (excludes dao bonus; engine applies it)
	DamageMult    float64  // damage multiplier (applied on top of Damage)
	Defense       float64  // shield value granted to self (excludes dao bonus)
	Healing       float64  // HP restored to self (excludes dao bonus)
	ManaCost      float64  // mana consumed on use
	BleedPerTurn  float64  // bleed DoT per turn (applied to target)
	PoisonPerTurn float64  // poison DoT per turn (applied to target)
	LifeStealPct  float64  // percentage of damage dealt healed back (0-1)
	MultiHit      int      // number of hits (chain_damage compound)
	PhantomDmg    float64  // bonus phantom damage
	ThornsDmg     float64  // reflect damage to attacker

	// ── New status effect fields (tier-based) ──
	StunTier       int // tier (1-5) for stun on target
	SlowTier       int // tier (1-5) for slow debuff on target (gated)
	Slow1Tier      int // tier (1-5) for slow debuff on target (guaranteed)
	SlowDamageTier int // tier (1-5) for slow damage debuff on target
	WeakTier       int // tier (1-5) for weak debuff on target
	FreezeTier     int // tier (1-5) for freeze debuff on target
	SilenceTier    int // tier (1-5) for silence debuff on target
	ThornsTier     int // tier (1-5) for thorns buff on self
	RegenTier      int // tier (1-5) for regen buff on self
	AtkUpTier      int // tier (1-5) for attack up buff on self
	DefUpTier      int // tier (1-5) for defense up buff on self
	TauntTier      int // tier (1-5) for taunt buff on self

	// StatusOps holds normalized status applications resolved by the statusfx
	// registry (catalog semantics: Value = intensity, Turns = duration). Used by
	// the engine for statuses driven by the data registry (e.g. BURN, FEAR, ...).
	StatusOps []statusfx.StatusOp

	// Backlash from using Gu rank > actor rank
	BacklashDamage float64 // HP damage taken by caster from Gu backlash

	// Targeting (AOE support)
	TargetType string // "SINGLE" | "ROW" | "COL" | "CROSS" | "ALL"
	TargetX    int    // preferred grid X (1=Front, 2=Mid, 3=Back)
	TargetY    int    // preferred grid Y (1=Top, 2=Mid, 3=Bottom)
	Bypass     bool   // true if action bypasses frontline protection (e.g. CurveAttack)

	// Priority (lower = higher priority, 0 = highest). Used by engine for action ordering.
	Priority int

	// Conditional-use fields. Empty condition = always use.
	UseCondition   string // "always","hp_below","mana_above","enemy_hp_below"
	ConditionValue float64
}

// Effect represents an active buff or debuff on a combatant.
type Effect struct {
	ID         string // effect constant (EffectSlow, EffectWeak, etc.)
	Type       string // "buff" or "debuff"
	Tier       int    // tier level (1-5) for scaling
	Value      int    // computed percent or duration (from TierToPercent / TierToDuration)
	Duration   int    // remaining turns
	SourcePath string // optional dao path source for cleansing traps
}

// DoTEffect tracks a damage-over-time or heal-over-time effect on a combatant.
type DoTEffect struct {
	Type  string  // "bleed", "poison", "regen"
	Value float64 // damage or heal per turn
	Turns int     // remaining turns
}

// DaoMarks represents the dao marks of a combatant, grouped by path.
type DaoMarks map[string]int

// CombatantAuto represents a stripped-down, deterministic entity in the combat grid.
type CombatantAuto struct {
	ID      string
	Name    string
	Kind    string // "Player", "Enemy", "Boss", "Minion"
	Rank    int
	SubRank int
	GridX   int // 1=Front, 2=Mid, 3=Back
	GridY   int // 1=Top, 2=Mid, 3=Bottom
	MaxHP   int
	HP      int
	MaxMana int
	Mana    int
	BaseDmg int // fallback basic-attack damage
	BaseDef int
	Speed   float64

	// Pre-combat Auto-Barrier
	VirtualShield int

	DaoMarks      DaoMarks
	PathMasteries map[string]int // path -> mastery points

	// Per-battle Chân Ý (path mastery) gain cap and the amount already earned.
	// Set by the handler to PathMasteryBattleGainLimit(player rank); the engine
	// stops awarding points once PathMasteryGained reaches PathMasteryGainLimit.
	PathMasteryGainLimit int
	PathMasteryGained    int

	// Character effects from race/constitution/statuses. Computed by the handler
	// from gamefx.Get(player) and applied by the engine during battle.
	DaoBonus         map[string]float64 // additive dao-path multiplier (path affinity)
	PathDamageBonus  map[string]float64 // additive damage multiplier for a path
	PathResistance   map[string]float64 // additive damage reduction taken for a path
	HealingReceived  float64            // multiplier bonus on healing received
	MentalResistance float64            // chance (0-1) to resist control effects

	// Equipment / Loadout
	Loadout *CombatLoadout

	// Pre-computed actions (produced by handler, consumed by simulator)
	Actions []SimAction

	// Active effects (Shields, Stuns, etc.)
	ActiveEffects []Effect

	// Damage-over-time effects (bleed, poison, regen)
	ActiveDoT []DoTEffect

	// ── Derived combat stats (computed each turn from active effects) ──
	EffectiveSpeed    float64 // speed after slow debuff
	DamageMultiplier  float64 // multiplier from weak debuff / atk_up buff (default 1.0)
	DefenseMultiplier float64 // multiplier from def_up buff (default 1.0)
	ThornsDamage      float64 // % of damage reflected back (from thorns buff)
	IsSilenced        bool    // cannot use skills, only basic attack
	IsFrozen          bool    // loses turn like stun, can break on damage
	HasTaunt          bool    // forces enemies to attack this unit
	TaunterID         string  // ID of the taunter (for target redirection)

	// ── New status-derived flags (driven by statusfx registry) ──
	IsSleeping           bool    // loses turn, breaks on damage (SLEEP)
	IsFeared             bool    // only basic attack, like silence (FEAR)
	IsCharmed            bool    // forced to attack the charmer's enemies (CHARM)
	IsStealthed          bool    // untargetable by single-target actions (STEALTH)
	IsConfused           bool    // chance to mis-target an ally (CONFUSE)
	IsBlinded            bool    // chance to miss (BLIND)
	IsBound              bool    // immobilize, loses turn (BIND)
	IsReviveLocked       bool    // cannot be revived (REVIVE_LOCK)
	// New GuWorld specific flags
	IsEssenceLeaking    bool    // ESSENCE_LEAK drains Chân Nguyên each turn
	HasGuSeal            bool    // GU_SEAL disables one equipped Gu's active effects
	HasSoulWound         bool    // SOUL_WOUND true damage and CN cost increase
	HasThoughtChaos      bool    // THOUGHT_CHAOS can cause KillerMove failure
	HasDaoConflict       bool    // DAO_CONFLICT reduces heal/shield from allies
	AccuracyPenalty      float64 // 0..1 miss chance (BLIND)
	DamageTakenReduction float64 // 0..1 damage reduction multiplier (DAMAGE_REDUCTION_UP / INVINCIBLE)

	// Auto-use settings
	AutoHealHP      bool
	HealThreshold   float64
	AutoRestoreMana bool
	ManaThreshold   float64

	// Mana refinement tier (from equipped Gu)
	RefineManaTier float64

	// Battle inventory
	HealCharges    int     // how many healing items available
	HealPotency    float64 // how much HP each heal restores
	PrimevalStones int64
	ImmortalStones int64

	// Tracking
	ItemsUsedInBattle int

	// Custom state for special Gu effects (e.g. GU_POEM_IMMORTAL stacks)
	CustomState map[string]int

	// Internal action index for round-robin
	nextActionIdx int

	// Hidden state
	IsDead bool
}

// CombatState represents the mutable snapshot of the battlefield at any given moment.
type CombatState struct {
	Turn    int
	Seed    int64
	Allies  []*CombatantAuto
	Enemies []*CombatantAuto

	// Boss specific tracking
	BossPhase int // 0 = 100-70%, 1 = 70-30%, 2 = 30-0%

	// Virtual mode: illusion combat, auto-restore mana each turn
	VirtualMode bool

	// Aperture (Tiên Khiếu) being defended during tribulation combat. When
	// ApertureMaxHP > 0, enemy damage that overflows the player's shield + HP
	// spills into the aperture. Tracked so the replay can show it depleting.
	ApertureHP    int
	ApertureMaxHP int
}

// CombatSnapshot captures full stats of all combatants at a point in time.
type CombatSnapshot struct {
	Turn    int                 `json:"turn"`
	Allies  []CombatantSnapshot `json:"allies"`
	Enemies []CombatantSnapshot `json:"enemies"`

	ApertureHP    int `json:"aperture_hp"`
	ApertureMaxHP int `json:"aperture_max_hp"`
}

// CombatantSnapshot is a point-in-time snapshot of one combatant's stats.
type CombatantSnapshot struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Kind    string  `json:"kind"`
	HP      int     `json:"hp"`
	MaxHP   int     `json:"max_hp"`
	Mana    int     `json:"mana"`
	MaxMana int     `json:"max_mana"`
	GridX   int     `json:"grid_x"`
	GridY   int     `json:"grid_y"`
	Shield  int     `json:"shield"`
	Damage  float64 `json:"damage"`
	Defense float64 `json:"defense"`
	Speed   float64 `json:"speed"`
	IsDead  bool    `json:"is_dead"`
}

// CombatLog represents a granular event to be stringified as JSON.
type CombatLog struct {
	Turn      int             `json:"turn"`
	ActorId   string          `json:"actor_id"`
	ActorName string          `json:"actor_name"`
	Action    string          `json:"action"` // e.g. "CAST_KILLER_MOVE", "BASIC_ATTACK", "TRIGGERED", "PHASE_TRANSITION"
	TargetId  string          `json:"target_id,omitempty"`
	Damage    int             `json:"damage,omitempty"`
	IsCrit    bool            `json:"is_crit,omitempty"`
	Trigger   string          `json:"trigger,omitempty"` // Name of the fixed trigger e.g. "THIET_HUYET_CO"
	Detail    string          `json:"detail,omitempty"`  // Human readable flavor text
	Snapshot  *CombatSnapshot `json:"snapshot,omitempty"`
}

func (c *CombatantAuto) GetManaName() string {
	if c.Kind != "Player" {
		return "chân nguyên"
	}
	switch c.Rank {
	case 1:
		return "Thanh Đồng Chân Nguyên"
	case 2:
		return "Xích Thiết Chân Nguyên"
	case 3:
		return "Bạch Ngân Chân Nguyên"
	case 4:
		return "Hoàng Kim Chân Nguyên"
	case 5:
		return "Tử Tinh Chân Nguyên"
	case 6:
		return "Thanh Đề Tiên Nguyên"
	case 7:
		return "Hồng Tảo Tiên Nguyên"
	case 8:
		return "Bạch Lệ Tiên Nguyên"
	case 9:
		return "Hoàng Hạnh Tiên Nguyên"
	default:
		return "chân nguyên"
	}
}
