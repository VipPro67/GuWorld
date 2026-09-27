package effectcatalog

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

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

var TriggerCatalog = values("ALWAYS_ON", "ON_ACTION", "ON_HIT", "BEFORE_HIT", "ON_TURN_START", "ON_DEATH", "ON_TICK", "CAPABILITY", "COMBAT_PASSIVE", "PASSIVE", "ON_BATTLE_START", "BEFORE_ENEMY_ATTACK", "ON_CAST", "ON_CONSUME", "ON_USE")
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
	"STAT_MOD":  {values("ADD_FLAT", "MULTIPLY_PERCENT", "SET", "GROWTH_CAP"), anyTriggers, combatTargets, values("op", "sub_type", "target", "stat", "value", "unit", "turns", "stack_policy"), []string{"stat", "value"}, false, false},
	"STATUS":    {nil, combatTriggers, combatTargets, values("op", "target", "status_id", "value", "turns", "chance", "stack_policy"), []string{"status_id", "turns", "chance"}, true, false},
	"BARRIER":   {values("FLAT", "PERCENT"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "turns", "stack_policy"), []string{"value", "turns"}, true, false},
	"CLEANSE":   {values("ALL", "DEBUFF", "STATUS"), anyTriggers, combatTargets, values("op", "sub_type", "target", "status_id"), nil, true, false},
	"UNIQUE":    {nil, anyTriggers, combatTargets, values("op", "target", "code", "params"), []string{"code"}, false, false},
}
var StatCatalog = values("ATTACK", "DEFENSE", "HP", "SPEED", "DAMAGE_DEALT", "ACCURACY", "EVASION", "TENACITY", "CC_RESIST", "LUCK", "CRIT_RATE", "ALL", "MANA", "MOVEMENT", "INVESTIGATION", "HEARING", "DISGUISE", "TIME_SPEED", "REFINE_SUCCESS", "REFINE_MANA_TIER", "STORAGE", "DAO_MARK", "PATH_CONFLICT_REDUCTION", "PATH_RESISTANCE")
var ResourceCatalog = values("HP", "MANA", "SOUL", "THOUGHT", "APTITUDE", "LIFESPAN", "STRESS", "PRIMEVAL_STONE", "IMMORTAL_STONE")
var StatusCatalog = func() map[string]bool {
	m := map[string]bool{}
	for id := range statusfx.Registry {
		m[id] = true
	}
	return m
}()
var UnitCatalog = values("FLAT", "PERCENT", "RATIO", "POINTS")
var StackPolicyCatalog = values("REPLACE", "REFRESH", "STACK", "MAX", "IGNORE")
var ConditionFields = values("HP_PERCENT", "ACTOR_HP_PERCENT", "MANA_PERCENT", "TURN", "TARGET_HP_PERCENT", "STACKS", "RANK", "MASTERY")
var ConditionOperators = values("EQ", "NE", "LT", "LTE", "GT", "GTE")

type ParamSpec struct {
	Type     string          `json:"type"`
	Required bool            `json:"required"`
	Min      *float64        `json:"min,omitempty"`
	Max      *float64        `json:"max,omitempty"`
	Enum     map[string]bool `json:"enum,omitempty"`
}

// ValidateNativeParams is shared by the strict JSON validator and the typed
// compiler so native mechanics cannot acquire two different parameter rules.
func ValidateNativeParams(code string, raw json.RawMessage) error {
	hook, ok := NativeHooks[code]
	if !ok {
		return fmt.Errorf("unknown native hook %q", code)
	}
	params := map[string]json.RawMessage{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &params); err != nil || params == nil {
			return fmt.Errorf("params must be an object")
		}
	}
	for key := range params {
		if _, ok := hook.Params[key]; !ok {
			return fmt.Errorf("unknown native parameter %q", key)
		}
	}
	for key, spec := range hook.Params {
		value, exists := params[key]
		if spec.Required && !exists {
			return fmt.Errorf("native parameter %q is required", key)
		}
		if exists && !spec.validate(value) {
			return fmt.Errorf("native parameter %q is invalid", key)
		}
	}
	return nil
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
	Domain   string
}

func number(required bool, min, max float64) ParamSpec {
	return ParamSpec{Type: "number", Required: required, Min: &min, Max: &max}
}

var NativeHooks = map[string]NativeHook{
	"REBORN":                  {values("ON_DEATH"), map[string]ParamSpec{"chance": number(true, 0, 1), "mode": {Type: "string", Enum: values("rewind", "in_place")}, "max_uses": number(false, 1, 10), "turn1_guaranteed": {Type: "bool"}, "revive_hp_percent": number(false, 0, 1), "clear_debuffs": {Type: "bool"}, "empower_attack_percent": number(false, 0, 5), "empower_speed_percent": number(false, 0, 5), "empower_shield_percent": number(false, 0, 5), "empower_turns": number(false, 1, 100)}, "combat"},
	"FORGET_INTENT_INTERRUPT": {values("BEFORE_ENEMY_ATTACK"), map[string]ParamSpec{"chance": number(true, 0, 1), "disable_target_gu_turns": number(false, 0, 10), "max_successful_activations": number(false, 1, 100), "backfire_after_max": {Type: "bool"}}, "combat"},
	"FORGET_WILL_CONFUSE":     {values("BEFORE_ENEMY_ATTACK"), map[string]ParamSpec{"activation_chance": number(false, 0, 1), "hit_user_chance": number(false, 0, 1), "hit_self_chance": number(false, 0, 1), "miss_chance": number(false, 0, 1), "backfire_chance": number(false, 0, 1), "backfire_disable_turns": number(false, 0, 10)}, "combat"},
	"SACRIFICE_NPC":           {values("CAPABILITY"), map[string]ParamSpec{"aptitude_gain": number(true, 1, 99), "aptitude_cap": number(true, 1, 100)}, "capability"},
}

func init() {
	register := func(trigger, domain string, codes ...string) {
		for _, code := range codes {
			if _, exists := NativeHooks[code]; !exists {
				NativeHooks[code] = NativeHook{Triggers: values(trigger), Params: map[string]ParamSpec{}, Domain: domain}
			}
		}
	}
	register("ON_CONSUME", "item_use", "SACRIFICE_APTITUDE", "RANK_PROGRESS", "ADD_SOUL_MAX", "ADD_LIFESPAN_YEARS", "RESTORE_SOUL_HP", "RESTORE_MENTAL", "INCREASE_TIME_SPEED", "REDUCE_STRESS", "ADD_THOUGHT_MAX", "DECREASE_TIME_SPEED", "MAN_BEAST_BURIAL", "POISON_VOW", "SPRING_EGG_HATCH", "STONE_APERTURE", "THIRD_WATCH")
	register("ON_TICK", "growth", "MANA_REGEN_TICK", "THOUGHT_REGEN_TICK", "SOUL_REGEN_TICK", "GENERATE_STONES_TICK", "GENERATE_IMMORTAL_STONES_TICK")
	register("ALWAYS_ON", "growth", "GROWTH_DURATION_HOURS")
	register("ON_USE", "world", "RECORD_MESSAGE")
	register("ON_TURN_START", "combat", "POEM_INTENT")
	register("ON_BATTLE_START", "combat", "SEIZE_INITIATIVE")
	register("ON_CAST", "combat", "SUMMON_WOLF_FRONTLINE", "FULL_SOUL_HEAL", "SWORD_DRAGON_TRANSFORM", "BORROW_PAST_POWER", "UNRAVEL_RIDDLE", "CHAOS_GAMBLE")
	register("ON_HIT", "combat", "BYPASS_SHIELD")
	register("BEFORE_HIT", "combat", "FIRST_TURN_EVADE")
	register("COMBAT_PASSIVE", "combat", "DOUBLE_DAMAGE_DEALT")
	register("CAPABILITY", "capability", "MARKETPLACE_ACCESS", "REMOTE_TRADE", "CHAT_CONNECT", "CHAT_PERCEIVE", "IGNORE_MOVEMENT_RANGE", "LOOT_PREVIEW", "DISGUISE", "CHANGE_FORM_DISGUISE", "PERFECT_DISGUISE", "ARENA_IDENTITY_MASK", "PORTAL_CREATE", "RANDOM_PORTAL", "CAPTURE_BONUS")
	// Schemas below describe parameters present in the authoritative CSV data.
	// Hooks without parameters retain the closed EMPTY schema.
	valueSchema := map[string]ParamSpec{"value": {Type: "number", Required: true}}
	for _, code := range []string{"ADD_LIFESPAN_YEARS", "ADD_SOUL_MAX", "ADD_THOUGHT_MAX", "DECREASE_TIME_SPEED", "GENERATE_IMMORTAL_STONES_TICK", "GENERATE_STONES_TICK", "GROWTH_DURATION_HOURS", "INCREASE_TIME_SPEED", "MANA_REGEN_TICK", "REDUCE_STRESS", "RESTORE_MENTAL", "RESTORE_SOUL_HP", "SOUL_REGEN_TICK", "THOUGHT_REGEN_TICK"} {
		hook := NativeHooks[code]
		hook.Params = valueSchema
		NativeHooks[code] = hook
	}
	hook := NativeHooks["CAPTURE_BONUS"]
	hook.Params = valueSchema
	NativeHooks["CAPTURE_BONUS"] = hook
	hook = NativeHooks["SEIZE_INITIATIVE"]
	hook.Params = map[string]ParamSpec{"lifespan_ratio": {Type: "number", Required: true}}
	NativeHooks["SEIZE_INITIATIVE"] = hook
	hook = NativeHooks["RECORD_MESSAGE"]
	hook.Params = map[string]ParamSpec{"range": {Type: "integer", Required: true}}
	NativeHooks["RECORD_MESSAGE"] = hook
}

type Metadata struct {
	SchemaVersion      int                  `json:"schema_version"`
	Triggers           []string             `json:"triggers"`
	Targets            []string             `json:"targets"`
	Operations         []OperationMetadata  `json:"operations"`
	Stats              []string             `json:"stats"`
	Statuses           []string             `json:"statuses"`
	Resources          []string             `json:"resources"`
	Units              []string             `json:"units"`
	StackPolicies      []string             `json:"stack_policies"`
	ConditionFields    []string             `json:"condition_fields"`
	ConditionOperators []string             `json:"condition_operators"`
	NativeCodes        []string             `json:"native_codes"`
	NativeHooks        []NativeHookMetadata `json:"native_hooks"`
}
type OperationMetadata struct {
	Name           string   `json:"name"`
	SubTypes       []string `json:"sub_types"`
	Triggers       []string `json:"triggers"`
	Targets        []string `json:"targets"`
	AllowedFields  []string `json:"allowed_fields"`
	RequiredFields []string `json:"required_fields"`
}
type NativeHookMetadata struct {
	Code     string               `json:"code"`
	Domain   string               `json:"domain"`
	Triggers []string             `json:"triggers"`
	Params   map[string]ParamSpec `json:"params"`
}

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func CatalogMetadata() Metadata {
	ops := make([]OperationMetadata, 0, len(OperationCatalog))
	for _, name := range sortedKeys(OperationCatalog) {
		d := OperationCatalog[name]
		ops = append(ops, OperationMetadata{name, sortedKeys(d.SubTypes), sortedKeys(d.Triggers), sortedKeys(d.Targets), sortedKeys(d.Allowed), append([]string(nil), d.Required...)})
	}
	hooks := make([]NativeHookMetadata, 0, len(NativeHooks))
	for _, code := range sortedKeys(NativeHooks) {
		h := NativeHooks[code]
		hooks = append(hooks, NativeHookMetadata{code, h.Domain, sortedKeys(h.Triggers), h.Params})
	}
	return Metadata{BlockSchemaVersion, sortedKeys(TriggerCatalog), sortedKeys(TargetCatalog), ops, sortedKeys(StatCatalog), sortedKeys(StatusCatalog), sortedKeys(ResourceCatalog), sortedKeys(UnitCatalog), sortedKeys(StackPolicyCatalog), sortedKeys(ConditionFields), sortedKeys(ConditionOperators), sortedKeys(NativeHooks), hooks}
}
