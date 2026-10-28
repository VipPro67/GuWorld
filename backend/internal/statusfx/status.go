// Package statusfx is the canonical, data-driven status/effect registry shared
// by realtime combat and gameplay effect handlers.
//
// It owns no model types, so combat and gameplay packages can import it without
// an import cycle. Adding a status is a registry entry plus an optional runtime
// behaviour hook.
package statusfx

// Canonical status IDs. These are the uppercase strings written into
// models.ActiveEffect.ApplyStatus (and Gu.csv), matching the catalog.
const (
	// Tier-based statuses retain their data-defined scaling behavior.
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
	// StatusInvincible is Bất Hoại: takes no damage at all. StatusLastStand is
	// Bất Tử: still takes damage, but HP never reaches 0. Sát Khí (level > 10)
	// pierces both.
	StatusInvincible = "INVINCIBLE"
	StatusLastStand  = "LAST_STAND"
	StatusEvasion    = "EVASION"
	StatusReviveLock = "REVIVE_LOCK"

	// Lãng Quên statuses (Wisdom Path). FORGET_EGO seals the victim's Gu
	// (BehaviourGuSeal); FORGET_REALITY weakens all damage they deal
	// (BehaviourWeak, intensity from target_damage_reduce_percent).
	StatusForgetEgo     = "FORGET_EGO"
	StatusForgetReality = "FORGET_REALITY"

	// StatusFlying is the world-map-only "phi hành" status granted by flying
	// Gu. It carries no combat behaviour; handlers read it to skip ground
	// monster encounters. StatusPhiHanh is the legacy alias still accepted by
	// the activation handler.
	StatusFlying  = "FLYING"
	StatusPhiHanh = "PHI_HANH"

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
	CategoryPhysicalDoT StatusCategory = "PHYSICAL_DOT"
	CategoryMentalCC    StatusCategory = "MENTAL_CC"
	CategoryAperture    StatusCategory = "APERTURE"
	CategoryDaoConflict StatusCategory = "DAO_CONFLICT"
	CategoryBuff        StatusCategory = "BUFF"
	CategoryImmortal    StatusCategory = "IMMORTAL"
)

// Behaviour tags drive realtime status application and ticking. Each new status
// maps to one behaviour hook; unknown behaviours fall through to a no-op log.
const (
	BehaviourDot             = "dot"              // damage over time (bleed, poison, burn, slow_damage)
	BehaviourRegen           = "regen"            // heal over time
	BehaviourStun            = "stun"             // skip turn
	BehaviourFreeze          = "freeze"           // skip turn, breaks on damage
	BehaviourSilence         = "silence"          // only basic attack (cannot use skills)
	BehaviourForceBasic      = "force_basic"      // fear: only basic attack
	BehaviourConfuse         = "confuse"          // chance to mis-target
	BehaviourStealth         = "stealth"          // untargetable, breaks on attack
	BehaviourSleep           = "sleep"            // skip turn, breaks on damage
	BehaviourBlind           = "blind"            // accuracy penalty / miss chance
	BehaviourCharm           = "charm"            // redirect target (attack allies)
	BehaviourBind            = "bind"             // immobilize (skip turn)
	BehaviourTaunt           = "taunt"            // forces enemies to attack taunter
	BehaviourThorns          = "thorns"           // reflects damage back to attacker
	BehaviourAtkUp           = "atk_up"           // increases attack damage
	BehaviourDefUp           = "def_up"           // increases defense
	BehaviourWeak            = "weak"             // reduces damage dealt
	BehaviourSlow            = "slow"             // reduces speed
	BehaviourHaste           = "haste"            // speed up
	BehaviourArmorBreak      = "armor_break"      // defense down
	BehaviourDamageReduction = "damage_reduction" // reduce damage taken
	BehaviourInvincible      = "invincible"       // Bất Hoại: immune to all damage
	BehaviourLastStand       = "last_stand"       // Bất Tử: takes damage, HP floors at 1
	BehaviourEvasion         = "evasion"          // dodge chance (0..1 = Effect.Value/100)
	BehaviourReviveLock      = "revive_lock"      // prevent revival

	// Gu World specific behaviours
	BehaviourAperture     = "aperture"      // ESSENCE_LEAK, GU_SEAL, THOUGHT_CHAOS
	BehaviourEssenceLeak  = "essence_leak"  // Drain mana / essence per turn
	BehaviourGuSeal       = "gu_seal"       // Disable random Gu skills
	BehaviourSoulWound    = "soul_wound"    // SOUL_WOUND - true damage bypassing shield
	BehaviourThoughtChaos = "thought_chaos" // THOUGHT_CHAOS - killer move fail chance
	BehaviourDaoConflict  = "dao_conflict"  // DAO_CONFLICT - reduce heal/shield from allies
	BehaviourBacklash     = "backlash"      // BACKLASH - instant reflect on shield/killer move break

	// BehaviourWorldMap marks statuses that only affect the world map. Combat
	// loops intentionally fall through to their no-op default.
	BehaviourWorldMap = "world_map" // FLYING / PHI_HANH - skip ground monsters
)
