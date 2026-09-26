package effectcatalog

import "encoding/json"

type OperationDefinition struct {
	SubTypes, Triggers, Targets map[string]bool
	Allowed                     map[string]bool
	Required                    []string
	NonNegative, Ratio          bool
}

func values(v ...string) map[string]bool {
	m := map[string]bool{}
	for _, s := range v {
		m[s] = true
	}
	return m
}

var TriggerCatalog = map[string]struct{ Valid bool }{"ALWAYS_ON": {true}, "ON_ACTION": {true}, "ON_HIT": {true}, "BEFORE_HIT": {true}, "ON_TURN_START": {true}, "ON_DEATH": {true}, "ON_TICK": {true}, "CAPABILITY": {true}, "COMBAT_PASSIVE": {true}, "ON_BATTLE_START": {true}, "BEFORE_ENEMY_ATTACK": {true}, "ON_CAST": {true}, "ON_CONSUME": {true}, "ON_USE": {true}}
var TargetCatalog = values("SELF", "TARGET", "ROW", "COL", "ALL_ENEMIES", "ALL_ALLIES", "ALL")
var combatTriggers = values("ON_ACTION", "ON_HIT", "BEFORE_HIT", "ON_TURN_START", "ON_DEATH", "COMBAT_PASSIVE", "ON_BATTLE_START", "BEFORE_ENEMY_ATTACK", "ON_CAST")
var anyTriggers = func() map[string]bool {
	m := map[string]bool{}
	for k := range TriggerCatalog {
		m[k] = true
	}
	return m
}()
var combatTargets = values("SELF", "TARGET", "ROW", "COL", "ALL_ENEMIES", "ALL_ALLIES", "ALL")
var OperationCatalog = map[string]OperationDefinition{
	"DAMAGE":    {values("FLAT", "MULTIPLIER", "MAX_HP_PERCENT"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "path", "chance"), []string{"value"}, true, false},
	"SHIELD":    {values("FLAT", "MULTIPLIER"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "turns"), []string{"value"}, true, false},
	"HEAL":      {values("FLAT", "PERCENT", "LIFESTEAL", "RESOURCE"), anyTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "resource"), []string{"value"}, true, false},
	"DIRECT_HP": {values("FLAT", "MAX_HP_PERCENT", "SHIELD_STRIP"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit"), []string{"value"}, true, false},
	"STAT_MOD":  {values("ADD_FLAT", "MULTIPLY_PERCENT", "SET"), anyTriggers, combatTargets, values("op", "sub_type", "target", "stat", "value", "unit", "turns", "stack_policy"), []string{"stat", "value"}, false, false},
	"STATUS":    {nil, combatTriggers, combatTargets, values("op", "target", "status_id", "value", "turns", "chance", "stack_policy"), []string{"status_id", "turns", "chance"}, true, false},
	"BARRIER":   {values("FLAT", "PERCENT"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "turns", "stack_policy"), []string{"value", "turns"}, true, false},
	"CLEANSE":   {values("ALL", "DEBUFF", "STATUS"), anyTriggers, combatTargets, values("op", "sub_type", "target", "status_id"), nil, true, false},
	"UNIQUE":    {nil, anyTriggers, combatTargets, values("op", "target", "code", "params"), []string{"code"}, false, false},
}
var StatCatalog = values("ATTACK", "DEFENSE", "HP", "SPEED", "DAMAGE_DEALT", "ACCURACY", "EVASION", "TENACITY", "CC_RESIST", "LUCK", "CRIT_RATE", "ALL")
var ResourceCatalog = values("HP", "MANA", "SOUL", "THOUGHT", "APTITUDE", "LIFESPAN", "STRESS", "PRIMEVAL_STONE", "IMMORTAL_STONE")
var StatusCatalog = values("BLEED", "POISON", "BURN", "FEAR", "CONFUSE", "STEALTH", "SLEEP", "BLIND", "CHARM", "BIND", "FORGET_EGO", "FORGET_REALITY", "SLOW", "SLOW1", "SLOW_DAMAGE", "WEAK", "REGEN", "ATK_UP", "DEF_UP", "THORNS", "TAUNT", "HASTE", "ARMOR_BREAK", "DAMAGE_REDUCTION_UP", "INVINCIBLE", "REVIVE_LOCK", "ESSENCE_LEAK", "GU_SEAL", "SOUL_WOUND", "THOUGHT_CHAOS", "DAO_CONFLICT", "BACKLASH", "STUN", "FREEZE", "SILENCE", "ROOT", "CONFUSION", "DECAY")
var UnitCatalog = values("FLAT", "PERCENT", "RATIO", "POINTS")
var StackPolicyCatalog = values("REPLACE", "REFRESH", "STACK", "MAX", "IGNORE")
var ConditionFields = values("HP_PERCENT", "MANA_PERCENT", "TURN", "TARGET_HP_PERCENT", "STACKS")
var ConditionOperators = values("EQ", "NE", "LT", "LTE", "GT", "GTE")

type ParamSpec struct {
	Type     string
	Required bool
	Min, Max *float64
	Enum     map[string]bool
}

func (p ParamSpec) validate(raw json.RawMessage) bool {
	switch p.Type {
	case "bool":
		var x bool
		return json.Unmarshal(raw, &x) == nil
	case "string":
		var x string
		if json.Unmarshal(raw, &x) != nil {
			return false
		}
		return len(p.Enum) == 0 || p.Enum[x]
	case "integer":
		var x int
		if json.Unmarshal(raw, &x) != nil {
			return false
		}
		return (p.Min == nil || float64(x) >= *p.Min) && (p.Max == nil || float64(x) <= *p.Max)
	default:
		var x float64
		if json.Unmarshal(raw, &x) != nil {
			return false
		}
		return (p.Min == nil || x >= *p.Min) && (p.Max == nil || x <= *p.Max)
	}
}

type NativeHook struct {
	Triggers map[string]bool
	Params   map[string]ParamSpec
}

func number(required bool, min, max float64) ParamSpec {
	return ParamSpec{Type: "number", Required: required, Min: &min, Max: &max}
}

var NativeHooks = map[string]NativeHook{
	"REBORN":                  {values("ON_DEATH"), map[string]ParamSpec{"chance": number(true, 0, 1), "mode": {Type: "string", Enum: values("rewind", "in_place")}, "max_uses": number(false, 1, 10), "turn1_guaranteed": {Type: "bool"}, "revive_hp_percent": number(false, 0, 1), "clear_debuffs": {Type: "bool"}, "empower_attack_percent": number(false, 0, 5), "empower_speed_percent": number(false, 0, 5), "empower_shield_percent": number(false, 0, 5), "empower_turns": number(false, 1, 100)}},
	"FORGET_INTENT_INTERRUPT": {values("BEFORE_ENEMY_ATTACK"), map[string]ParamSpec{"chance": number(true, 0, 1), "disable_target_gu_turns": number(false, 0, 10), "max_successful_activations": number(false, 1, 100), "backfire_after_max": {Type: "bool"}}},
	"FORGET_WILL_CONFUSE":     {values("BEFORE_ENEMY_ATTACK"), map[string]ParamSpec{"activation_chance": number(false, 0, 1), "hit_user_chance": number(false, 0, 1), "hit_self_chance": number(false, 0, 1), "miss_chance": number(false, 0, 1), "backfire_chance": number(false, 0, 1), "backfire_disable_turns": number(false, 0, 10)}},
	"SACRIFICE_NPC":           {values("CAPABILITY"), map[string]ParamSpec{"aptitude_gain": number(true, 1, 99), "aptitude_cap": number(true, 1, 100)}},
}

func init() {
	register := func(trigger string, codes ...string) {
		for _, code := range codes {
			if _, exists := NativeHooks[code]; !exists {
				NativeHooks[code] = NativeHook{Triggers: values(trigger), Params: map[string]ParamSpec{}}
			}
		}
	}
	register("ON_CONSUME", "SACRIFICE_APTITUDE", "RANK_PROGRESS", "ADD_SOUL_MAX", "ADD_LIFESPAN_YEARS", "RESTORE_SOUL_HP", "RESTORE_MENTAL", "INCREASE_TIME_SPEED", "REDUCE_STRESS", "ADD_THOUGHT_MAX", "DECREASE_TIME_SPEED", "MAN_BEAST_BURIAL", "POISON_VOW", "SPRING_EGG_HATCH", "STONE_APERTURE", "THIRD_WATCH")
	register("ON_TICK", "MANA_REGEN_TICK", "THOUGHT_REGEN_TICK", "SOUL_REGEN_TICK", "GENERATE_STONES_TICK", "GENERATE_IMMORTAL_STONES_TICK")
	register("ON_USE", "RECORD_MESSAGE")
	register("ON_TURN_START", "POEM_INTENT")
	register("ON_BATTLE_START", "SEIZE_INITIATIVE")
	register("ON_CAST", "SUMMON_WOLF_FRONTLINE", "FULL_SOUL_HEAL", "SWORD_DRAGON_TRANSFORM", "BORROW_PAST_POWER", "UNRAVEL_RIDDLE", "CHAOS_GAMBLE")
	register("ON_HIT", "BYPASS_SHIELD")
	register("BEFORE_HIT", "FIRST_TURN_EVADE")
	register("COMBAT_PASSIVE", "DOUBLE_DAMAGE_DEALT")
	register("CAPABILITY", "MARKETPLACE_ACCESS", "REMOTE_TRADE", "CHAT_CONNECT", "CHAT_PERCEIVE", "IGNORE_MOVEMENT_RANGE", "LOOT_PREVIEW", "DISGUISE", "CHANGE_FORM_DISGUISE", "PERFECT_DISGUISE", "ARENA_IDENTITY_MASK")
}
