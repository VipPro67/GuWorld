package effectcatalog

import "sort"

type Disposition string

const (
	DispositionSupported Disposition = "supported"
	DispositionPartial   Disposition = "partial"
	DispositionMetadata  Disposition = "metadata"
	DispositionMigrate   Disposition = "migrate"
	DispositionKnownBug  Disposition = "known_bug"
)

func (d Disposition) Valid() bool {
	switch d {
	case DispositionSupported, DispositionPartial, DispositionMetadata, DispositionMigrate, DispositionKnownBug:
		return true
	default:
		return false
	}
}

type Definition struct {
	Section     string      `json:"section"`
	Key         string      `json:"key"`
	Disposition Disposition `json:"disposition"`
	Owner       string      `json:"owner"`
}

var fieldDefinitions = buildFieldDefinitions()

func buildFieldDefinitions() map[string]map[string]Definition {
	definitions := map[string]map[string]Definition{}
	add := func(section string, disposition Disposition, owner string, keys ...string) {
		if definitions[section] == nil {
			definitions[section] = map[string]Definition{}
		}
		for _, key := range keys {
			if _, exists := definitions[section][key]; exists {
				panic("duplicate effect definition: " + section + "." + key)
			}
			definitions[section][key] = Definition{Section: section, Key: key, Disposition: disposition, Owner: owner}
		}
	}

	add("active", DispositionSupported, "combat",
		"damage_flat", "heal_flat", "shield_flat", "damage_mult", "shield_mult", "apply_status", "status_tick_damage", "status_duration_ticks",
		"target_selector", "speed_boost_flat", "ignore_defense", "ignore_defense_pct", "crit_rate_boost", "all_stats_buff_percent",
		"lifesteal_rate", "blood_drain", "curve_attack", "summon_phantom", "freeze", "slow", "slow1", "weak", "stun",
		"silence", "bleed", "poison", "thorns", "regen", "atk_up", "def_up", "taunt",
		"damage_hp_percent", "shield_flat_reduce_percent", "disable_gu_next_turn",
		"forget_reality_chance", "self_evasion_boost_percent", "target_damage_reduce_percent",
		"damage_path")
	add("active", DispositionSupported, "world", "capture_bonus", "portal_create", "random_portal")
	add("active", DispositionPartial, "combat-legacy", "range", "mana_regen", "cleansing_power", "wood_transformation", "refine_mana_tier", "summon_growth_phantoms", "summon_count_per_tick", "phantom_damage_scale")
	add("active", DispositionMigrate, "effect-runtime-v2", "replicate", "generate_gu", "reveal_all_stealth", "reveal_weakness", "invulnerable_ticks", "summon_monster_id", "sleep", "decoy_count", "convert_damage_to_heal_percent", "absorb_damage_flat")
	add("active", DispositionMigrate, "passive-stats", "stat_base_attack", "stat_base_defense", "stat_base_hp", "evasion")
	add("active", DispositionMetadata, "content", "description")

	add("passive", DispositionSupported, "player-stats", "stat_base_attack", "stat_base_defense", "stat_base_hp", "stat_base_speed", "movement", "investigation", "hearing", "evasion", "luck", "accuracy", "damage_multiplier_on_hit", "cleansing_power", "disguise_level", "tenacity", "cc_resist")
	add("passive", DispositionSupported, "combat", "anti_heaven_damage_reduction_percent", "anti_earth_damage_reduction_percent", "anti_dream_damage_reduction_percent", "immune_dream_status", "forget_heaven_protection", "forget_earth_protection")
	add("passive", DispositionSupported, "refinement", "success_rate_bonus", "refine_mana_tier", "soul_refinement")
	add("passive", DispositionSupported, "inventory", "storage_capacity")
	add("passive", DispositionSupported, "time", "time_speed_multiplier")
	add("passive", DispositionKnownBug, "effect-runtime-v2", "path_conflict_reduction")
	add("passive", DispositionMigrate, "effect-runtime-v2", "damage_mult", "stat_base_luck", "dao_mark_multiplier", "kill_path_damage_buff", "luck_flat", "storage_slots", "kill_path_dao_marks_mult")
	add("passive", DispositionMetadata, "content", "description")

	add("growth", DispositionSupported, "growth-worker", "max_attack", "max_defense", "max_hp", "max_speed", "hours_to_max")
	return definitions
}

func DefinitionFor(section, key string) (Definition, bool) {
	definition, ok := fieldDefinitions[section][key]
	return definition, ok
}

func Definitions() []Definition {
	result := make([]Definition, 0)
	for _, byKey := range fieldDefinitions {
		for _, definition := range byKey {
			result = append(result, definition)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Section == result[j].Section {
			return result[i].Key < result[j].Key
		}
		return result[i].Section < result[j].Section
	})
	return result
}

type MechanicDefinition struct {
	Trigger     string      `json:"trigger"`
	ActionCode  string      `json:"action_code"`
	Disposition Disposition `json:"disposition"`
	Owner       string      `json:"owner"`
}

var mechanicDefinitions = buildMechanicDefinitions()

func buildMechanicDefinitions() map[string]MechanicDefinition {
	result := map[string]MechanicDefinition{}
	add := func(trigger string, disposition Disposition, owner string, actions ...string) {
		for _, action := range actions {
			key := trigger + "\x00" + action
			if _, exists := result[key]; exists {
				panic("duplicate mechanic definition: " + trigger + "/" + action)
			}
			result[key] = MechanicDefinition{Trigger: trigger, ActionCode: action, Disposition: disposition, Owner: owner}
		}
	}
	add("ON_CONSUME", DispositionSupported, "gu-consume", "SACRIFICE_APTITUDE", "RANK_PROGRESS", "ADD_SOUL_MAX", "ADD_LIFESPAN_YEARS", "RESTORE_SOUL_HP", "RESTORE_MENTAL", "INCREASE_TIME_SPEED", "REDUCE_STRESS", "ADD_THOUGHT_MAX", "DECREASE_TIME_SPEED")
	add("ON_TICK", DispositionSupported, "resource-regen", "MANA_REGEN_TICK", "THOUGHT_REGEN_TICK", "SOUL_REGEN_TICK")
	add("ON_TICK", DispositionSupported, "hunger-worker", "GENERATE_STONES_TICK", "GENERATE_IMMORTAL_STONES_TICK")
	add("ON_CONSUME", DispositionMigrate, "effect-runtime-v2", "MAN_BEAST_BURIAL", "POISON_VOW", "SPRING_EGG_HATCH", "STONE_APERTURE", "THIRD_WATCH")
	add("ON_USE", DispositionMigrate, "effect-runtime-v2", "RECORD_MESSAGE")
	add("ON_DEATH", DispositionSupported, "combat", "REBORN")
	add("BEFORE_ENEMY_ATTACK", DispositionSupported, "combat", "FORGET_INTENT_INTERRUPT", "FORGET_WILL_CONFUSE")
	add("ON_TURN_START", DispositionSupported, "combat", "POEM_INTENT")
	add("ON_BATTLE_START", DispositionSupported, "combat", "SEIZE_INITIATIVE")
	add("ON_CAST", DispositionSupported, "combat", "SUMMON_WOLF_FRONTLINE", "FULL_SOUL_HEAL", "SWORD_DRAGON_TRANSFORM", "BORROW_PAST_POWER", "UNRAVEL_RIDDLE", "CHAOS_GAMBLE")
	add("ON_HIT", DispositionSupported, "combat", "BYPASS_SHIELD")
	add("BEFORE_HIT", DispositionSupported, "combat", "FIRST_TURN_EVADE")
	add("PASSIVE", DispositionSupported, "combat", "DOUBLE_DAMAGE_DEALT")
	// COMBAT_PASSIVE is the canonical alias for the PASSIVE trigger. New data
	// should use it; PASSIVE is retained so existing Gu rows keep validating.
	// Both mean the same ON_EVENT timing (see timing.go), never the passive
	// section (ALWAYS_ON character stats).
	add("COMBAT_PASSIVE", DispositionSupported, "combat", "DOUBLE_DAMAGE_DEALT")
	add("CAPABILITY", DispositionSupported, "world", "MARKETPLACE_ACCESS", "REMOTE_TRADE", "CHAT_CONNECT", "CHAT_PERCEIVE", "IGNORE_MOVEMENT_RANGE", "LOOT_PREVIEW", "DISGUISE", "CHANGE_FORM_DISGUISE", "PERFECT_DISGUISE", "ARENA_IDENTITY_MASK", "SACRIFICE_NPC")
	return result
}

func MechanicDefinitionFor(trigger, action string) (MechanicDefinition, bool) {
	definition, ok := mechanicDefinitions[trigger+"\x00"+action]
	return definition, ok
}

func MechanicDefinitions() []MechanicDefinition {
	result := make([]MechanicDefinition, 0, len(mechanicDefinitions))
	for _, definition := range mechanicDefinitions {
		result = append(result, definition)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Trigger == result[j].Trigger {
			return result[i].ActionCode < result[j].ActionCode
		}
		return result[i].Trigger < result[j].Trigger
	})
	return result
}

var statusDefinitions = func() map[string]Definition {
	result := map[string]Definition{}
	add := func(disposition Disposition, owner string, ids ...string) {
		for _, id := range ids {
			if _, exists := result[id]; exists {
				panic("duplicate status definition: " + id)
			}
			result[id] = Definition{Section: "status", Key: id, Disposition: disposition, Owner: owner}
		}
	}
	add(DispositionSupported, "statusfx", "BLEED", "POISON", "BURN", "FEAR", "CONFUSE", "STEALTH", "SLEEP", "BLIND", "CHARM", "BIND", "FORGET_EGO", "FORGET_REALITY",
		"SLOW", "SLOW1", "SLOW_DAMAGE", "WEAK", "REGEN", "ATK_UP", "DEF_UP", "THORNS", "TAUNT",
		"HASTE", "ARMOR_BREAK", "DAMAGE_REDUCTION_UP", "INVINCIBLE", "REVIVE_LOCK",
		"ESSENCE_LEAK", "GU_SEAL", "SOUL_WOUND", "THOUGHT_CHAOS", "DAO_CONFLICT", "BACKLASH")
	add(DispositionPartial, "combat-legacy", "STUN", "FREEZE", "SILENCE")
	add(DispositionMigrate, "effect-runtime-v2", "ROOT", "CONFUSION", "DECAY")
	return result
}()

func StatusDefinitionFor(id string) (Definition, bool) {
	definition, ok := statusDefinitions[id]
	return definition, ok
}

func StatusDefinitions() []Definition {
	result := make([]Definition, 0, len(statusDefinitions))
	for _, definition := range statusDefinitions {
		result = append(result, definition)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}
