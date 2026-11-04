package v2

// StatusCode maps canonical simulator status IDs to compact, stable wire IDs.
// Keep this mapping in sync with frontend/src/game/network/v2/status.ts.
func StatusCode(id string) uint8 {
	switch id {
	case "BURN":
		return 1
	case "BLEED":
		return 2
	case "BIND":
		return 3
	case "POISON":
		return 4
	case "STUN":
		return 5
	case "CONFUSE":
		return 6
	case "FEAR":
		return 7
	case "EVASION":
		return 8
	case "FREEZE":
		return 9
	case "BLIND":
		return 10
	case "SILENCE":
		return 11
	case "CHARM":
		return 12
	case "SLEEP":
		return 13
	case "INVINCIBLE":
		return 14
	case "SLOW":
		return 15
	case "HASTE":
		return 16
	case "SLOW1":
		return 17
	case "FLYING", "FLIGHT":
		return 18
	case "STEALTH", "INVISIBILITY":
		return 19
	case "REVIVE_LOCK":
		return 20
	case "WEAK":
		return 21
	case "FORM_WOLF_PARTIAL":
		return 22
	case "SLOW_DAMAGE":
		return 23
	case "THORNS":
		return 24
	case "REGEN":
		return 25
	case "ATK_UP":
		return 26
	case "DEF_UP":
		return 27
	case "TAUNT":
		return 28
	case "ARMOR_BREAK":
		return 29
	case "DAMAGE_REDUCTION_UP":
		return 30
	case "LAST_STAND":
		return 31
	case "FORGET_EGO":
		return 32
	case "FORGET_REALITY":
		return 33
	case "PHI_HANH":
		return 34
	case "ESSENCE_LEAK":
		return 35
	case "GU_SEAL":
		return 36
	case "SOUL_WOUND":
		return 37
	case "THOUGHT_CHAOS":
		return 38
	case "DAO_CONFLICT":
		return 39
	case "BACKLASH":
		return 40
	case "FORM_TURTLE_PARTIAL":
		return 41
	case "ROOT":
		return 42
	case "PETRIFY":
		return 43
	default:
		return 0
	}
}
