package simulator

import (
	"encoding/json"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// CombatLoadout represents the input data from the player before the battle starts.
type CombatLoadout struct {
	GuIds        []string `json:"GuIds"`
	ImmortalGuId *string  `json:"ImmortalGuId,omitempty"`

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
	ActionType     string   // "BASIC_ATTACK" | "GU_SKILL" | "KILLER_MOVE" | "HEAL" | "SHIELD"
	Name           string   // display name
	SourceID       string   // GuID or KillerMoveID for tactic matching
	InstanceID     string   // Gu instance ID (player_gu.id) when ActionType == "GU_SKILL"
	KillerMoveID   string   // persisted player killer-move id for mastery tracking
	ComponentGuIDs []string // Gu instances that lose hunger when this killer move is attempted
	SuccessChance  float64  // 0..1; only used by KILLER_MOVE
	Path           []string // dao paths used to compute the dao mark bonus in the engine
	Rank           int      // rank of the Gu/Skill
	Damage         float64  // raw damage before defense mitigation (excludes dao bonus; engine applies it)
	DamageMult     float64  // damage multiplier (applied on top of Damage)
	ShieldMult     float64  // shield multiplier (applied on top of Defense)
	Defense        float64  // shield value granted to self (excludes dao bonus)
	Healing        float64  // HP restored to self (excludes dao bonus)
	ManaCost       float64  // mana consumed on use
	ManaRestore    float64  // mana restored to self
	CleanseDebuffs bool     // purges debuffs/DoTs from self
	BleedPerTurn   float64  // bleed DoT per turn (applied to target)
	PoisonPerTurn  float64  // poison DoT per turn (applied to target)
	LifeStealPct   float64  // percentage of damage dealt healed back (0-1)
	MultiHit       int      // number of hits (chain_damage compound)
	PhantomDmg     float64  // bonus phantom damage
	ThornsDmg      float64  // reflect damage to attacker

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

	// Percent-based direct effects (0..1), from active.damage_hp_percent etc.
	// Generic: any Gu may set them; engine applies them after normal damage.
	DamageHPPercent    float64 // % target MaxHP as direct HP damage (bypasses shield)
	ShieldStripPercent float64 // % target current shield stripped
	ProcChance         float64 // 0 = always; >0 = gate status ops + bonus effects
	TargetWeakPct      float64 // WEAK intensity 0..1 applied to target
	BonusTurns         int     // duration for TargetWeakPct effects
	DisableGuTurns     int     // GU_SEAL turns applied to target (disable_gu_next_turn)

	// Backlash from using Gu rank > actor rank
	BacklashDamage float64 // HP damage taken by caster from Gu backlash

	// Targeting (AOE support)
	TargetType string // "SINGLE" | "ROW" | "COL" | "CROSS" | "ALL" | "ALL_ENEMIES"
	TargetX    int    // preferred grid X (1=Front, 2=Mid, 3=Back)
	TargetY    int    // preferred grid Y (1=Top, 2=Mid, 3=Bottom)
	Bypass     bool   // true if action bypasses frontline protection (e.g. CurveAttack)

	// Priority (lower = higher priority, 0 = highest). Used by engine for action ordering.
	Priority int

	// Conditional-use fields. Empty condition = always use.
	UseCondition   string // "always","hp_below","mana_above","enemy_hp_below"
	ConditionValue float64

	// Mechanics is the immutable, catalog-compiled behavior attached to this
	// action. SourceID remains metadata for logs/tactics; the engine must never
	// infer gameplay behavior from it.
	Mechanics []CombatMechanic

	// PersistentStatusKey declares a post-combat persistent player status
	// (gamefx/Status.csv key, e.g. "sinh_khi_tan") granted to the PvP defender
	// when this action lands a hit. Data-driven: authored on the Gu as a
	// UNIQUE PERSIST_STATUS_ON_HIT native hook (ON_HIT or ON_ACTION), resolved
	// by handlers at combat build time. Empty means no persistent effect.
	PersistentStatusKey string
	// PersistentPvPOnly restricts the persistent grant to real PvP hits.
	PersistentPvPOnly bool

	// CompiledOperations is the temporary v2 adapter payload. Legacy fields
	// above remain populated until simulator parity is complete.
	CompiledOperations []gameplayeffects.CompiledOperation
}

// CombatMechanic is the simulator-facing form of a compiled UNIQUE operation. Params stay
// immutable JSON so combat snapshots are deterministic and definitions can be
// extended without coupling the engine to Gu identifiers.
type CombatMechanic struct {
	Trigger    string
	ActionCode string
	Params     []byte // legacy authoring payload; never decoded by the simulator
	Values     map[string]any
}

// NewCombatMechanic compiles native mechanic parameters before simulation.
func NewCombatMechanic(trigger, actionCode string, params []byte) CombatMechanic {
	m := CombatMechanic{Trigger: trigger, ActionCode: actionCode, Params: append([]byte(nil), params...)}
	_ = json.Unmarshal(params, &m.Values)
	return m
}

// CompiledCombatMechanics adapts registry-compiled native mechanics without
// parsing JSON during combat construction or simulation.
func CompiledCombatMechanics(compiled *gameplayeffects.Compiled, action bool) []CombatMechanic {
	if compiled == nil {
		return nil
	}
	var result []CombatMechanic
	for _, op := range compiled.NativeMechanics() {
		isAction := op.Trigger == gameplayeffects.TriggerOnCast || op.Trigger == gameplayeffects.TriggerOnHit
		if isAction != action {
			continue
		}
		result = append(result, CombatMechanic{Trigger: string(op.Trigger), ActionCode: op.Code, Params: append([]byte(nil), op.Params...), Values: op.Values})
	}
	return result
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
	Type     string  // "bleed", "poison", "regen"
	Value    float64 // damage or heal per turn
	Turns    int     // remaining turns
	DirectHP bool    // true if damage subtracts directly from HP bypassing shield
}

// DaoMarks represents the dao marks of a combatant, grouped by path.
type DaoMarks map[string]int64

// CombatantAuto represents a stripped-down, deterministic entity in the combat grid.
type CombatantAuto struct {
	ID      string
	Name    string
	Kind    string // "Player", "Enemy", "Boss", "Minion"
	Rank    int
	SubRank int
	GridX   int // 1=Front, 2=Mid, 3=Back
	GridY   int // 1=Top, 2=Mid, 3=Bottom
	MaxHP   float64
	HP      float64
	MaxMana float64
	Mana    float64
	BaseDmg float64 // fallback basic-attack damage
	BaseDef float64
	Speed   float64

	// Lifespan (thọ nguyên). Set from the player on build; monsters leave 0 and
	// the engine falls back to lifespanByRank. Used by ON_BATTLE_START/SEIZE_INITIATIVE.
	Lifespan float64

	// Pre-combat Auto-Barrier
	VirtualShield float64

	DaoMarks      DaoMarks
	PathMasteries map[string]int // path -> mastery points

	// Per-battle Chân Ý (path mastery) gain cap and the amount already earned.
	// Set by the handler to PathMasteryBattleGainLimit(player rank); the engine
	// stops awarding points once PathMasteryGained reaches PathMasteryGainLimit.
	PathMasteryGainLimit int
	PathMasteryGained    int

	// Character effects from race/constitution/statuses. Computed by the handler
	// from gamefx.Get(player) and applied by the engine during battle.
	DaoBonus             map[string]float64       // additive dao-path multiplier (path affinity)
	PathDamageBonus      map[string]float64       // additive damage multiplier for a path
	PathResistance       map[string]float64       // additive damage reduction taken for a path
	HealingReceived      float64                  // multiplier bonus on healing received
	MentalResistance     float64                  // chance (0-1) to resist control effects
	Tenacity             float64                  // generic self-dispel chance + CC duration reduction (0-1)
	CCResist             float64                  // generic proc resist vs all debuffs (0-1)
	SoulDefenseBonus     float64                  // percentage reduction of soul damage taken
	LuckBonus            float64                  // flat increase of luck stat
	HasNonConflictingDao bool                     // true if character possesses non-conflicting dao marks trait
	DaoConflictReductionPct float64                // diminishing-return reduction from equipped Gu
	DaoProfile           *gamefx.DaoCombatProfile // precomputed O(1) dao conflict and natural defense profile

	// StatProvenance maps stat key ("attack"/"defense"/"speed"/
	// "healing_received") to the status-effect modification provenance computed
	// by the handler at build time, for breakdown display. Nil for monsters and
	// statusless combatants.
	StatProvenance map[string]*StatProvenance `json:"stat_provenance,omitempty"`

	// Equipment / Loadout
	Loadout            *CombatLoadout
	CategoryPriorities []string // Priority order of skill categories: "shield", "heal", "damage", "other"

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
	IsStunned         bool    // loses turn (STUN)
	IsFrozen          bool    // loses turn like stun, can break on damage
	HasTaunt          bool    // forces enemies to attack this unit
	TaunterID         string  // ID of the taunter (for target redirection)

	// ── New status-derived flags (driven by statusfx registry) ──
	IsSleeping     bool // loses turn, breaks on damage (SLEEP)
	IsFeared       bool // only basic attack, like silence (FEAR)
	IsCharmed      bool // forced to attack the charmer's enemies (CHARM)
	IsStealthed    bool // untargetable by single-target actions (STEALTH)
	IsConfused     bool // chance to mis-target an ally (CONFUSE)
	IsBlinded      bool // chance to miss (BLIND)
	IsBound        bool // immobilize, loses turn (BIND)
	IsReviveLocked bool // cannot be revived (REVIVE_LOCK)
	// New GuWorld specific flags
	IsEssenceLeaking     bool    // ESSENCE_LEAK drains Chân Nguyên each turn
	HasGuSeal            bool    // GU_SEAL disables one equipped Gu's active effects
	HasSoulWound         bool    // SOUL_WOUND true damage and CN cost increase
	HasThoughtChaos      bool    // THOUGHT_CHAOS can cause KillerMove failure
	HasDaoConflict       bool    // DAO_CONFLICT reduces heal/shield from allies
	AccuracyPenalty      float64 // 0..1 miss chance (BLIND)
	EvasionChance        float64 // 0..1 dodge chance from EVASION statuses (merged by max, cap EvasionChanceCap)
	ImmuneDreamStatus    bool    // immune to dream-path statuses (FORGET_DREAM)
	DamageTakenReduction float64 // 0..1 damage reduction multiplier (DAMAGE_REDUCTION_UP)
	// IsInvincible is Bất Hoại: takes no damage from any source, including the
	// DoT and %MaxHP paths that bypass applyDamageToTarget. IsLastStand is Bất
	// Tử: damage still lands but HP never reaches 0. Sát Khí (level > 10) pierces
	// both.
	IsInvincible       bool
	IsLastStand        bool
	CritRateBonus      float64            // extra crit chance (0..1) from Gu active effects
	IgnoreDefenseFlat  float64            // flat defense ignored by attacker
	IgnoreDefensePct   float64            // % defense ignored by attacker (0..1)
	PendingDamageBuffs map[string]float64 // key: GuID/SourceID, value: multiplier (applied once per unique ID, consumed on attack)
	PendingShieldBuffs map[string]float64 // key: GuID/SourceID, value: multiplier (applied once per unique ID, consumed on shield)

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
	PrimevalStones float64
	ImmortalStones float64

	// Tracking
	ItemsUsedInBattle    int
	GuHunger             map[string]int // gu_id or instance_id -> hunger remaining in battle
	GuUsedCounts         map[string]int // gu_id or instance_id -> times used in battle
	KillerMoveUsedCounts map[string]int // persisted killer-move id -> successful casts

	// Custom state for special Gu effects (e.g. GU_POEM_IMMORTAL stacks)
	CustomState map[string]int

	// Mechanics contains equipped/passive mechanics compiled at combat start.
	Mechanics []CombatMechanic

	// Internal action index for round-robin
	nextActionIdx int

	// Hidden state
	IsDead bool
}

// CombatState represents the mutable snapshot of the battlefield at any given moment.
type CombatState struct {
	Turn                 int
	Seed                 int64
	Allies               []*CombatantAuto
	Enemies              []*CombatantAuto
	CaptureTurnSnapshots bool

	// Boss specific tracking
	BossPhase int // 0 = 100-70%, 1 = 70-30%, 2 = 30-0%

	// Virtual mode: illusion combat, auto-restore mana each turn
	VirtualMode bool

	// Aperture (Tiên Khiếu) being defended during tribulation combat. When
	// ApertureMaxHP > 0, enemy damage that overflows the player's shield + HP
	// spills into the aperture. Tracked so the replay can show it depleting.
	ApertureHP    float64
	ApertureMaxHP float64

	// Pre-battle events logged at Turn 0 (e.g. Steal Gu)
	InitialLogs []CombatLog
}

// CombatSnapshot captures full stats of all combatants at a point in time.
type CombatSnapshot struct {
	Turn    int                 `json:"turn"`
	Allies  []CombatantSnapshot `json:"allies"`
	Enemies []CombatantSnapshot `json:"enemies"`

	ApertureHP    float64 `json:"aperture_hp"`
	ApertureMaxHP float64 `json:"aperture_max_hp"`
}

// CombatantSnapshot is a point-in-time snapshot of one combatant's stats.
type CombatantSnapshot struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Kind     string                 `json:"kind"`
	HP       float64                `json:"hp"`
	MaxHP    float64                `json:"max_hp"`
	Mana     float64                `json:"mana"`
	MaxMana  float64                `json:"max_mana"`
	GridX    int                    `json:"grid_x"`
	GridY    int                    `json:"grid_y"`
	Shield   float64                `json:"shield"`
	Damage   float64                `json:"damage"`
	Defense  float64                `json:"defense"`
	Speed    float64                `json:"speed"`
	IsDead   bool                   `json:"is_dead"`
	Statuses []CombatStatusSnapshot `json:"statuses,omitempty"`
}

// CombatStatusSnapshot captures a status effect at a particular replay point.
type CombatStatusSnapshot struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Tier           int     `json:"tier,omitempty"`
	Value          float64 `json:"value,omitempty"`
	RemainingTurns int     `json:"remaining_turns"`
	DirectHP       bool    `json:"direct_hp,omitempty"`
}

// StatModSource attributes one status-effect stat modification for the
// breakdown UI (e.g. Trọng Thương: attack −20%).
type StatModSource struct {
	Key  string  `json:"key"`  // status dictionary key
	Name string  `json:"name"` // display name (Vietnamese)
	Pct  float64 `json:"pct"`  // applied percentage, e.g. -0.5 => -50%
}

// StatProvenance contrasts a stat's pre-status value with its final combat
// value, itemizing the status modifications in between. Computed by the
// handler at combatant build time.
type StatProvenance struct {
	Stat       string          `json:"stat"`
	BaseValue  float64         `json:"base_value"`
	FinalValue float64         `json:"final_value"`
	Mods       []StatModSource `json:"mods,omitempty"`
}

// DamageBreakdown contains the exact formula components for detailed player inspection.
type DamageBreakdown struct {
	BaseDamage      float64 `json:"base_damage,omitempty"`
	SkillMult       float64 `json:"skill_mult,omitempty"`
	DaoMult         float64 `json:"dao_mult,omitempty"`
	BuffMult        float64 `json:"buff_mult,omitempty"`
	CritMult        float64 `json:"crit_mult,omitempty"`
	PathBonus       float64 `json:"path_bonus,omitempty"`
	TargetDef       float64 `json:"target_def,omitempty"`
	NaturalResist   float64 `json:"natural_resist,omitempty"`
	PathResist      float64 `json:"path_resist,omitempty"`
	DamageReduction float64 `json:"damage_reduction,omitempty"`
	ShieldAbsorbed  float64 `json:"shield_absorbed,omitempty"`
	HPDamage        float64 `json:"hp_damage,omitempty"`
	FinalDamage     float64 `json:"final_damage,omitempty"`
	RawDamage       float64 `json:"raw_damage,omitempty"`
	HitCount        int     `json:"hit_count,omitempty"`
	Reason          string  `json:"reason,omitempty"`

	// Status-effect provenance attached by the engine from the combatants'
	// StatProvenance maps, so the UI can show e.g. "Trọng Thương −50%".
	AttackerAttack  *StatProvenance `json:"attacker_attack,omitempty"`
	DefenderDefense *StatProvenance `json:"defender_defense,omitempty"`
	HealerHealing   *StatProvenance `json:"healer_healing,omitempty"`
	ShielderDefense *StatProvenance `json:"shielder_defense,omitempty"`

	// Healing formula components (skill heals and item heals).
	HealingBase     float64 `json:"healing_base,omitempty"`
	HealingDaoBonus float64 `json:"healing_dao_bonus,omitempty"`
	HealingFinal    float64 `json:"healing_final,omitempty"`

	// Shield formula components (skill shield and shield buffs).
	ShieldBase      float64 `json:"shield_base,omitempty"`
	ShieldSkillMult float64 `json:"shield_skill_mult,omitempty"`
	ShieldBuffMult  float64 `json:"shield_buff_mult,omitempty"`
	ShieldDaoMult   float64 `json:"shield_dao_mult,omitempty"`
	ShieldDaoBonus  float64 `json:"shield_dao_bonus,omitempty"`
	ShieldFinal     float64 `json:"shield_final,omitempty"`
}

// CombatLog represents a granular event to be stringified as JSON.
type CombatLog struct {
	Turn      int     `json:"turn"`
	ActorId   string  `json:"actor_id"`
	ActorName string  `json:"actor_name"`
	Action    string  `json:"action"` // e.g. "CAST_KILLER_MOVE", "BASIC_ATTACK", "TRIGGERED", "PHASE_TRANSITION"
	TargetId  string  `json:"target_id,omitempty"`
	Damage    float64 `json:"damage,omitempty"`
	IsCrit    bool    `json:"is_crit,omitempty"`
	Trigger   string  `json:"trigger,omitempty"` // Name of the fixed trigger e.g. "THIET_HUYET_CO"
	Detail    string  `json:"detail,omitempty"`  // Human readable flavor text
	// SourceGuID is the Gu (or killer-move) ID of the action that produced
	// this log. Lets post-combat hooks attribute hits to data-driven Gu
	// effects without parsing Detail strings.
	SourceGuID string `json:"source_gu_id,omitempty"`
	// HitLanded marks a strike that made contact with the target — including
	// hits fully absorbed by VirtualShield (shield-absorb counts as a hit).
	// Misses (evade/blind), immunes and dead-target swings leave it false.
	HitLanded bool             `json:"hit_landed,omitempty"`
	Breakdown *DamageBreakdown `json:"breakdown,omitempty"`
	Snapshot  *CombatSnapshot  `json:"snapshot,omitempty"`
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
	case 10:
		return "Hỗn Độn Tiên Nguyên"
	default:
		return "chân nguyên"
	}
}
