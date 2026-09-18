// Package statusfx is the canonical, data-driven status/effect registry shared
// by the simulator engine and the legacy combat handlers.
//
// It owns nothing from models/simulator (stdlib only) so both packages can
// import it without an import cycle. Adding a new status is a single registry
// entry + an optional behaviour hook in the engine; no new if-chains.
package statusfx

// Canonical status IDs. These are the uppercase strings written into
// models.ActiveEffect.ApplyStatus (and Gu.csv), matching the catalog.
const (
	// Legacy tier-based statuses (handled by the simulator tier pipeline today).
	StatusBleed      = "BLEED"
	StatusPoison     = "POISON"
	StatusStun       = "STUN"
	StatusSlow       = "SLOW"
	StatusSlow1      = "SLOW1"
	StatusSlowDamage = "SLOW_DAMAGE"
	StatusWeak       = "WEAK"
	StatusFreeze     = "FREEZE"
	StatusSilence    = "SILENCE"
	StatusThorns     = "THORNS"
	StatusRegen      = "REGEN"
	StatusAtkUp      = "ATK_UP"
	StatusDefUp      = "DEF_UP"
	StatusTaunt      = "TAUNT"

	// Newly wired statuses (previously dropped silently).
	StatusBurn    = "BURN"
	StatusFear    = "FEAR"
	StatusConfuse = "CONFUSE"
	StatusStealth = "STEALTH"
	StatusSleep   = "SLEEP"
	StatusBlind   = "BLIND"
	StatusCharm   = "CHARM"
	StatusBind    = "BIND"

	// Catalog statuses registered for future wiring (not yet combat-wired).
	StatusHaste           = "HASTE"
	StatusArmorBreak      = "ARMOR_BREAK"
	StatusDamageReduction = "DAMAGE_REDUCTION_UP"
	StatusInvincible      = "INVINCIBLE"
	StatusReviveLock      = "REVIVE_LOCK"

	// Gu World Dao Mechanics - New Status Effects
	StatusEssenceLeak  = "ESSENCE_LEAK"
	StatusGuSeal       = "GU_SEAL"
	StatusSoulWound    = "SOUL_WOUND"
	StatusThoughtChaos = "THOUGHT_CHAOS"
	StatusDaoConflict  = "DAO_CONFLICT"
	StatusBacklash     = "BACKLASH"
)

// StatusCategory groups statuses by their nature for dispel/cleanse mechanics.
type StatusCategory string

const (
	CategoryPhysicalDoT   StatusCategory = "PHYSICAL_DOT"
	CategoryMentalCC      StatusCategory = "MENTAL_CC"
	CategoryAperture      StatusCategory = "APERTURE"
	CategoryDaoConflict   StatusCategory = "DAO_CONFLICT"
	CategoryBuff          StatusCategory = "BUFF"
	CategoryImmortal      StatusCategory = "IMMORTAL"
)

// Behaviour tags drive the simulator's apply/tick dispatch. Each new status
// maps to one behaviour hook; unknown behaviours fall through to a no-op log.
const (
	BehaviourDot             = "dot"               // damage over time (bleed, poison, burn, slow_damage)
	BehaviourRegen           = "regen"             // heal over time
	BehaviourStun            = "stun"              // skip turn
	BehaviourFreeze          = "freeze"            // skip turn, breaks on damage
	BehaviourSilence         = "silence"           // only basic attack (cannot use skills)
	BehaviourForceBasic      = "force_basic"       // fear: only basic attack
	BehaviourConfuse         = "confuse"           // chance to mis-target
	BehaviourStealth         = "stealth"           // untargetable, breaks on attack
	BehaviourSleep           = "sleep"             // skip turn, breaks on damage
	BehaviourBlind           = "blind"             // accuracy penalty / miss chance
	BehaviourCharm           = "charm"             // redirect target (attack allies)
	BehaviourBind            = "bind"              // immobilize (skip turn)
	BehaviourTaunt           = "taunt"             // forces enemies to attack taunter
	BehaviourThorns          = "thorns"            // reflects damage back to attacker
	BehaviourAtkUp           = "atk_up"            // increases attack damage
	BehaviourDefUp           = "def_up"            // increases defense
	BehaviourWeak            = "weak"              // reduces damage dealt
	BehaviourSlow            = "slow"              // reduces speed
	BehaviourHaste           = "haste"             // speed up
	BehaviourArmorBreak      = "armor_break"       // defense down
	BehaviourDamageReduction = "damage_reduction"  // reduce damage taken
	BehaviourInvincible      = "invincible"        // immune to damage
	BehaviourReviveLock      = "revive_lock"       // prevent revival

	// Gu World specific behaviours
	BehaviourAperture      = "aperture"       // ESSENCE_LEAK, GU_SEAL, THOUGHT_CHAOS
	BehaviourEssenceLeak   = "essence_leak"   // Drain mana / essence per turn
	BehaviourGuSeal        = "gu_seal"        // Disable random Gu skills
	BehaviourSoulWound     = "soul_wound"     // SOUL_WOUND - true damage bypassing shield
	BehaviourThoughtChaos  = "thought_chaos"  // THOUGHT_CHAOS - killer move fail chance
	BehaviourDaoConflict   = "dao_conflict"   // DAO_CONFLICT - reduce heal/shield from allies
	BehaviourBacklash      = "backlash"       // BACKLASH - instant reflect on shield/killer move break
)