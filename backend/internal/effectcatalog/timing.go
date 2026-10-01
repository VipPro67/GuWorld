package effectcatalog

// Timing classifies WHEN an effect fires. It is orthogonal to Disposition:
// Timing says when an effect would fire, Disposition (supported / partial /
// migrate / ...) says whether a runtime currently wires it.
//
//	ALWAYS_ON  — owned/equipped/active is enough (race, constitution,
//	             persistent statuses, equipped ACTIVE Gu passives).
//	             Resolved by gamefx.Get on every read.
//	ON_ACTION  — fires when the player uses something (a combat strike from
//	             active.*, consuming a Gu via ON_CONSUME, using one via ON_USE).
//	ON_EVENT   — fires on a combat-engine hook (ON_CAST, ON_HIT, BEFORE_HIT,
//	             BEFORE_ENEMY_ATTACK, ON_TURN_START, ON_BATTLE_START, ON_DEATH,
//	             COMBAT_PASSIVE) or as a transient combat status.
//	SCHEDULED  — fires on a clock (growth.hours_to_max, ON_TICK workers and
//	             lazy regen, status expiry in real minutes).
//	CAPABILITY — never fires; gates an action (marketplace, trade, chat, loot
//	             preview, disguise, movement). Checked via GuMechanic /
//	             InventoryHasGuMechanic.
//	META       — descriptive only (description), ignored by runtimes.
type Timing string

const (
	TimingAlwaysOn   Timing = "ALWAYS_ON"
	TimingOnAction   Timing = "ON_ACTION"
	TimingOnEvent    Timing = "ON_EVENT"
	TimingScheduled  Timing = "SCHEDULED"
	TimingCapability Timing = "CAPABILITY"
	TimingMeta       Timing = "META"
)

// FieldTiming maps a (section, key) catalog field to its firing timing.
// Unknown fields return false so new keys cannot silently fall through.
func FieldTiming(section, key string) (Timing, bool) {
	if _, ok := DefinitionFor(section, key); !ok {
		return "", false
	}
	if key == "description" {
		return TimingMeta, true
	}
	switch section {
	case "passive":
		return TimingAlwaysOn, true
	case "active":
		return TimingOnAction, true
	case "growth":
		return TimingScheduled, true
	}
	return "", false
}

// MechanicTiming maps an event_trigger to its firing timing.
func MechanicTiming(trigger string) (Timing, bool) {
	switch trigger {
	case "ALWAYS_ON":
		return TimingAlwaysOn, true
	case "ON_CONSUME", "ON_USE":
		return TimingOnAction, true
	case "ON_TICK":
		return TimingScheduled, true
	case "CAPABILITY":
		return TimingCapability, true
	case "ON_CAST", "ON_HIT", "BEFORE_HIT", "BEFORE_ENEMY_ATTACK",
		"ON_TURN_START", "ON_BATTLE_START", "ON_DEATH",
		"PASSIVE", "COMBAT_PASSIVE":
		return TimingOnEvent, true
	}
	return "", false
}

// StatusTiming maps a manifest combat status to its timing. Manifest statuses
// are transient ON_EVENT effects (applied by ON_ACTION strikes, ticked per
// combat turn). Persistent player statuses (status_dictionary via gamefx) are
// ALWAYS_ON instead and live outside this manifest.
func StatusTiming(id string) (Timing, bool) {
	if _, ok := StatusDefinitionFor(id); !ok {
		return "", false
	}
	return TimingOnEvent, true
}
