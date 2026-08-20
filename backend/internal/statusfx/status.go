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
)

// Behaviour tags drive the simulator's apply/tick dispatch. Each new status
// maps to one behaviour hook; unknown behaviours fall through to a no-op log.
const (
	BehaviourDot             = "dot"              // damage over time (burn)
	BehaviourForceBasic      = "force_basic"      // fear: only basic attack
	BehaviourConfuse         = "confuse"          // chance to mis-target
	BehaviourStealth         = "stealth"          // untargetable, breaks on attack
	BehaviourSleep           = "sleep"            // skip turn, breaks on damage
	BehaviourBlind           = "blind"            // accuracy penalty / miss chance
	BehaviourCharm           = "charm"            // redirect target (attack allies)
	BehaviourBind            = "bind"             // immobilize (skip turn)
	BehaviourHaste           = "haste"            // speed up (catalog)
	BehaviourArmorBreak      = "armor_break"      // defense down (catalog)
	BehaviourDamageReduction = "damage_reduction" // reduce damage taken (catalog)
	BehaviourInvincible      = "invincible"       // immune to damage (catalog)
	BehaviourReviveLock      = "revive_lock"      // prevent revival (catalog)
)
