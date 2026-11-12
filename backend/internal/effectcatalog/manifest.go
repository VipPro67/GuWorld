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
		"forget_reality_chance", "target_damage_reduce_percent",
		"damage_path")
	add("active", DispositionSupported, "world", "capture_bonus", "portal_create", "random_portal")
	add("active", DispositionPartial, "combat-legacy", "range", "mana_regen", "cleansing_power", "wood_transformation", "refine_mana_tier", "summon_growth_phantoms", "summon_count_per_tick", "phantom_damage_scale")
	add("active", DispositionMigrate, "effect-runtime-v2", "replicate", "generate_gu", "reveal_all_stealth", "reveal_weakness", "invulnerable_ticks", "summon_monster_id", "sleep", "decoy_count", "convert_damage_to_heal_percent", "absorb_damage_flat")
	// EVASION/ACCURACY are no longer passive stats: evasion is now the
	// self-targeted EVASION status (see statusDefinitions), and the accuracy
	// stat was removed outright. Legacy payloads carrying these keys are only
	// warned about, never rejected.
	add("active", DispositionMigrate, "removed-passive-evasion", "evasion", "self_evasion_boost_percent")
	add("active", DispositionMigrate, "passive-stats", "stat_base_attack", "stat_base_defense", "stat_base_hp")
	add("active", DispositionMetadata, "content", "description")

	add("passive", DispositionSupported, "player-stats", "stat_base_attack", "stat_base_defense", "stat_base_hp", "stat_base_speed", "movement", "investigation", "hearing", "luck", "damage_multiplier_on_hit", "cleansing_power", "disguise_level", "tenacity", "cc_resist")
	add("passive", DispositionMigrate, "removed-passive-evasion", "evasion", "accuracy")
	add("passive", DispositionSupported, "combat", "anti_heaven_damage_reduction_percent", "anti_earth_damage_reduction_percent", "anti_dream_damage_reduction_percent", "immune_dream_status", "forget_heaven_protection", "forget_earth_protection")
	add("passive", DispositionSupported, "refinement", "success_rate_bonus", "refine_mana_tier", "soul_refinement")
	add("passive", DispositionSupported, "inventory", "storage_capacity")
	add("passive", DispositionSupported, "time", "time_speed_multiplier")
	add("passive", DispositionSupported, "player-stats", "path_conflict_reduction")
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
	add("ON_CONSUME", DispositionSupported, "gu-consume", "SOVEREIGN_BODY_TRANSFORM", "SACRIFICE_APTITUDE", "RANK_PROGRESS", "ADD_SOUL_MAX", "ADD_LIFESPAN_YEARS", "RESTORE_SOUL_HP", "RESTORE_MENTAL", "INCREASE_TIME_SPEED", "REDUCE_STRESS", "ADD_THOUGHT_MAX", "DECREASE_TIME_SPEED", "DEDUCTION_INSTANT_FINISH")
	add("ALWAYS_ON", DispositionSupported, "deduction", "DEDUCTION_INSTANT_FINISH")
	add("ON_TICK", DispositionSupported, "resource-regen", "MANA_REGEN_TICK", "THOUGHT_REGEN_TICK", "SOUL_REGEN_TICK")
	add("ON_TICK", DispositionSupported, "hunger-worker", "GENERATE_STONES_TICK", "GENERATE_IMMORTAL_STONES_TICK")
	add("ON_CONSUME", DispositionSupported, "gu-consume", "MAN_BEAST_BURIAL")
	add("ON_CONSUME", DispositionMigrate, "effect-runtime-v2", "POISON_VOW", "SPRING_EGG_HATCH", "STONE_APERTURE", "THIRD_WATCH")
	add("ON_USE", DispositionMigrate, "effect-runtime-v2", "RECORD_MESSAGE")
	add("ON_DEATH", DispositionSupported, "combat", "REBORN")
	add("BEFORE_ENEMY_ATTACK", DispositionSupported, "combat", "FORGET_INTENT_INTERRUPT", "FORGET_WILL_CONFUSE")
	add("ON_TURN_START", DispositionSupported, "combat", "POEM_INTENT")
	add("ON_BATTLE_START", DispositionSupported, "combat", "SEIZE_INITIATIVE", "SAT_KHI_GUARANTEE", "PERM_STEAL_GU", "TEMP_STEAL_GU", "SUPPRESS_PRIORITY", "SUMMON_WOLF_FRONTLINE", "SUMMON_MONSTER", "INSTANT_WIN", "LETHAL_LUCK_WARD")
	add("ON_CAST", DispositionSupported, "combat", "SUMMON_WOLF_FRONTLINE", "FULL_SOUL_HEAL", "SWORD_DRAGON_TRANSFORM", "BORROW_PAST_POWER", "UNRAVEL_RIDDLE", "CHAOS_GAMBLE", "SUMMON_MONSTER", "LUCK_ROULETTE", "DREAM_UNRAVEL", "BLOOD_SACRIFICE", "CALCULATED_STRIKE", "TIME_REWIND", "FORM_BEAR", "FORM_EAGLE", "BONE_THORNS", "METAL_AEGIS", "WOOD_REGEN_BURST", "WATER_CLEANSE", "WATER_MIRROR_SHIELD", "EARTH_SHIELD", "WIND_HASTE_BURST", "LIGHT_PURIFY", "DARK_STEALTH", "STAR_THOUGHT_RESTORE", "CLOUD_MIST", "TIEP_THIEN_DIA", "VAN_DAN_DONG_TAM", "QI_WALL", "SPATIAL_SHIFT", "FORMATION_DOMAIN")
	add("ON_HIT", DispositionSupported, "combat", "BYPASS_SHIELD", "GUARANTEED_HIT", "BEAST_PHANTOM_STRIKE", "SOUL_SHOCK", "MANA_STEAL", "SWORD_ARMOR_PIERCE", "EXECUTE_STRIKE", "BONE_SPIKE", "ARMOR_BREAK_HEAVY", "WOOD_BIND", "FIRE_INFERNO", "LIGHTNING_STUN_BURST", "CHAIN_LIGHTNING", "ICE_DEEP_FREEZE", "ICE_FROSTBITE", "LIGHT_BLINDING_FLASH", "DARK_CURSE_WEAK", "SOUND_DISRUPT_SILENCE", "SOUND_RESONANCE", "DIMENSION_SLASH", "JUSTICE_VERDICT", "BLINDING_MIRAGE", "PAINTING_IMPRISONMENT")
	add("ON_ACTION", DispositionSupported, "combat", "GUARANTEED_HIT", "LUCK_ROULETTE", "DREAM_UNRAVEL", "CALCULATED_STRIKE", "BLOOD_SACRIFICE", "TIME_REWIND", "SOUL_SHOCK", "SOUL_DEVOUR", "BEAST_PHANTOM_STRIKE", "HEAVY_LIFESTEAL", "BLEED_BURST", "MANA_STEAL", "BUFF_STEAL", "PACK_FRENZY", "POISON_DETONATE", "SWORD_ARMOR_PIERCE", "EXECUTE_STRIKE", "BONE_SPIKE", "FORM_BEAR", "FORM_EAGLE", "BONE_THORNS", "METAL_AEGIS", "ARMOR_BREAK_HEAVY", "WOOD_REGEN_BURST", "WOOD_BIND", "WATER_CLEANSE", "WATER_MIRROR_SHIELD", "FIRE_INFERNO", "IGNITE_BURST", "EARTH_SHIELD", "EARTH_GRAVITY", "WIND_HASTE_BURST", "LIGHTNING_STUN_BURST", "CHAIN_LIGHTNING", "ICE_DEEP_FREEZE", "ICE_FROSTBITE", "LIGHT_BLINDING_FLASH", "LIGHT_PURIFY", "DARK_STEALTH", "DARK_CURSE_WEAK", "STAR_FIRE_RAIN", "STAR_THOUGHT_RESTORE", "SOUND_DISRUPT_SILENCE", "SOUND_RESONANCE", "CLOUD_MIST", "HEAVEN_SUPPRESSION", "TIEP_THIEN_DIA", "HUMAN_WILL", "VAN_DAN_DONG_TAM", "QI_WALL", "QI_EXPLOSION", "SPATIAL_SHIFT", "DIMENSION_SLASH", "RULE_IMPOSITION", "JUSTICE_VERDICT", "FORMATION_DOMAIN", "BLINDING_MIRAGE", "PAINTING_IMPRISONMENT", "PASSION_OVERLOAD", "GU_DETONATION")
	add("BEFORE_HIT", DispositionSupported, "combat", "FIRST_TURN_EVADE", "LETHAL_LUCK_WARD", "DAO_ADAPTIVE_RESIST", "MANA_EAT")
	add("PASSIVE", DispositionSupported, "combat", "DOUBLE_DAMAGE_DEALT")
	add("COMBAT_PASSIVE", DispositionSupported, "combat", "DOUBLE_DAMAGE_DEALT", "REVERSE_CALAMITY", "GUARANTEED_HIT", "LETHAL_LUCK_WARD", "BERSERK_STATE", "PACK_FRENZY", "DAO_ADAPTIVE_RESIST", "MANA_EAT", "BONE_THORNS", "WATER_MIRROR_SHIELD", "HUMAN_WILL", "PASSION_OVERLOAD")
	add("CAPABILITY", DispositionSupported, "world", "MARKETPLACE_ACCESS", "REMOTE_TRADE", "CHAT_CONNECT", "CHAT_PERCEIVE", "IGNORE_MOVEMENT_RANGE", "IGNORE_MONSTER_ENCOUNTER", "LOOT_PREVIEW", "DISGUISE", "CHANGE_FORM_DISGUISE", "PERFECT_DISGUISE", "ARENA_IDENTITY_MASK", "DEDUCTION_INSTANT_FINISH", "CAPTURE_BONUS", "CONCEAL_GU", "SEVER_CLUES", "PORTAL_CREATE", "RANDOM_PORTAL")
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
		"HASTE", "ARMOR_BREAK", "DAMAGE_REDUCTION_UP", "INVINCIBLE", "LAST_STAND", "EVASION", "REVIVE_LOCK",
		"ESSENCE_LEAK", "GU_SEAL", "SOUL_WOUND", "THOUGHT_CHAOS", "DAO_CONFLICT", "BACKLASH", "FLYING", "PHI_HANH",
		"DREAM_PRISON", "DREAM_ILLUSION", "LUCK_SIPHON", "LUCK_CURSE_REVERSE",
		"BERSERK", "CALCULATED", "BLOOD_SACRIFICE",
		"BEAR_FORM", "EAGLE_FORM", "ADAPTIVE_RESIST", "BONE_ARMOR",
		"METAL_AEGIS", "CLOUD_MIST")
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
