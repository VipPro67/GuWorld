package effectcatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

type OperationDefinition struct {
	SubTypes, Triggers, Targets map[string]bool
	Allowed                     map[string]bool
	Required                    []string
	NonNegative, Ratio          bool
}

// RankScaledDebuffParams configures a rank-sensitive PvP debuff whose
// intensity follows an existing game rule and whose duration scales by rank.
type RankScaledDebuffParams struct {
	StrengthFactor         float64 `json:"strength_factor"`
	StatusID               string  `json:"status_id"`
	DurationBaseMinutes    int     `json:"duration_base_minutes"`
	DurationPerRankMinutes int     `json:"duration_per_rank_minutes"`
	SourceName             string  `json:"source_name"`
}

// GuSealParams configures a reactive Gu seal and its per-battle activation cap.
type GuSealParams struct {
	CounterKey      string  `json:"counter_key"`
	Chance          float64 `json:"chance"`
	Turns           int     `json:"turns"`
	MaxActivations  int     `json:"max_activations"`
	BackfireAtLimit bool    `json:"backfire_at_limit"`
	BackfireTurns   int     `json:"backfire_turns"`
}

// LuckRouletteParams controls every numeric part of the luck outcome table.
type LuckRouletteParams struct {
	GreatChancePercent      float64 `json:"great_chance_percent"`
	CurseGreatChanceBonus   float64 `json:"curse_great_chance_bonus"`
	MediumRangePercent      float64 `json:"medium_range_percent"`
	NeutralRangePercent     float64 `json:"neutral_range_percent"`
	GreatDamageMultiplier   float64 `json:"great_damage_multiplier"`
	MediumDamageMultiplier  float64 `json:"medium_damage_multiplier"`
	NeutralDamageMultiplier float64 `json:"neutral_damage_multiplier"`
	BadDamageMultiplier     float64 `json:"bad_damage_multiplier"`
	GreatManaCostMultiplier float64 `json:"great_mana_cost_multiplier"`
	BadCurrentHPLossPercent float64 `json:"bad_current_hp_loss_percent"`
	MinimumBadHPLoss        float64 `json:"minimum_bad_hp_loss"`
	CurseDurationTurns      int     `json:"curse_duration_turns"`
}

func ParseLuckRouletteParams(raw json.RawMessage) (LuckRouletteParams, error) {
	var params LuckRouletteParams
	if len(raw) == 0 || json.Unmarshal(raw, &params) != nil {
		return params, fmt.Errorf("LUCK_ROULETTE params must be an object")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return params, fmt.Errorf("LUCK_ROULETTE params must be an object")
	}
	allowed := map[string]bool{
		"great_chance_percent": true, "curse_great_chance_bonus": true, "medium_range_percent": true, "neutral_range_percent": true,
		"great_damage_multiplier": true, "medium_damage_multiplier": true, "neutral_damage_multiplier": true, "bad_damage_multiplier": true,
		"great_mana_cost_multiplier": true, "bad_current_hp_loss_percent": true, "minimum_bad_hp_loss": true, "curse_duration_turns": true,
	}
	for field := range fields {
		if !allowed[field] {
			return params, fmt.Errorf("LUCK_ROULETTE does not accept param %q", field)
		}
	}
	if len(fields) != len(allowed) {
		return params, fmt.Errorf("LUCK_ROULETTE requires all outcome chances, multipliers, HP loss and curse duration params")
	}
	numbers := []float64{params.GreatChancePercent, params.CurseGreatChanceBonus, params.MediumRangePercent, params.NeutralRangePercent}
	for _, value := range numbers {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
			return params, fmt.Errorf("LUCK_ROULETTE chance and range values must be within [0,100]")
		}
	}
	if params.GreatChancePercent+params.MediumRangePercent+params.NeutralRangePercent > 100 {
		return params, fmt.Errorf("LUCK_ROULETTE normal outcome ranges must total at most 100")
	}
	multipliers := []float64{params.GreatDamageMultiplier, params.MediumDamageMultiplier, params.NeutralDamageMultiplier, params.BadDamageMultiplier, params.GreatManaCostMultiplier, params.BadCurrentHPLossPercent}
	for _, value := range multipliers {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 10 {
			return params, fmt.Errorf("LUCK_ROULETTE multipliers and HP loss percent must be within [0,10]")
		}
	}
	if math.IsNaN(params.MinimumBadHPLoss) || math.IsInf(params.MinimumBadHPLoss, 0) || params.MinimumBadHPLoss < 0 || params.MinimumBadHPLoss > 1e12 || params.CurseDurationTurns < 1 || params.CurseDurationTurns > 100 {
		return params, fmt.Errorf("LUCK_ROULETTE minimum HP loss or curse duration is out of range")
	}
	return params, nil
}

func ValidateLuckRouletteParams(raw json.RawMessage) error {
	_, err := ParseLuckRouletteParams(raw)
	return err
}

func ParseGuSealParams(raw json.RawMessage) (GuSealParams, error) {
	var params GuSealParams
	if len(raw) == 0 || json.Unmarshal(raw, &params) != nil {
		return params, fmt.Errorf("GU_SEAL params must be an object")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return params, fmt.Errorf("GU_SEAL params must be an object")
	}
	allowed := map[string]bool{"counter_key": true, "chance": true, "turns": true, "max_activations": true, "backfire_at_limit": true, "backfire_turns": true}
	for field := range fields {
		if !allowed[field] {
			return params, fmt.Errorf("GU_SEAL does not accept param %q", field)
		}
	}
	if len(fields) != len(allowed) {
		return params, fmt.Errorf("GU_SEAL requires counter_key, chance, turns, max_activations, backfire_at_limit and backfire_turns")
	}
	if !validCounterKey(params.CounterKey) {
		return params, fmt.Errorf("GU_SEAL counter_key must be a lowercase counter identifier")
	}
	if math.IsNaN(params.Chance) || math.IsInf(params.Chance, 0) || params.Chance < 0 || params.Chance > 1 || params.Turns < 1 || params.Turns > 10 || params.MaxActivations < 1 || params.MaxActivations > 100 || params.BackfireTurns < 0 || params.BackfireTurns > 10 {
		return params, fmt.Errorf("GU_SEAL chance, turns, max_activations or backfire_turns is out of range")
	}
	if params.BackfireAtLimit && params.BackfireTurns == 0 {
		return params, fmt.Errorf("GU_SEAL backfire_turns must be positive when backfire_at_limit is enabled")
	}
	return params, nil
}

func ValidateGuSealParams(raw json.RawMessage) error {
	_, err := ParseGuSealParams(raw)
	return err
}

// ParseRankScaledDebuffParams validates and decodes the authored config.
func ParseRankScaledDebuffParams(subType string, raw json.RawMessage) (RankScaledDebuffParams, error) {
	var params RankScaledDebuffParams
	if subType != "REGIONAL_WALL" {
		return params, fmt.Errorf("unknown RANK_SCALED_DEBUFF subtype %q", subType)
	}
	if len(raw) == 0 || json.Unmarshal(raw, &params) != nil {
		return params, fmt.Errorf("RANK_SCALED_DEBUFF params must be an object")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return params, fmt.Errorf("RANK_SCALED_DEBUFF params must be an object")
	}
	allowed := map[string]bool{"strength_factor": true, "status_id": true, "duration_base_minutes": true, "duration_per_rank_minutes": true, "source_name": true}
	for field := range fields {
		if !allowed[field] {
			return params, fmt.Errorf("RANK_SCALED_DEBUFF does not accept param %q", field)
		}
	}
	if len(fields) != len(allowed) {
		return params, fmt.Errorf("RANK_SCALED_DEBUFF requires strength_factor, status_id, duration_base_minutes, duration_per_rank_minutes and source_name")
	}
	if math.IsNaN(params.StrengthFactor) || math.IsInf(params.StrengthFactor, 0) || params.StrengthFactor <= 0 || params.StrengthFactor > 1 {
		return params, fmt.Errorf("RANK_SCALED_DEBUFF strength_factor must be in (0,1]")
	}
	if err := ValidatePersistentStatusID(params.StatusID); err != nil {
		return params, fmt.Errorf("RANK_SCALED_DEBUFF status_id: %w", err)
	}
	if params.DurationBaseMinutes < 1 || params.DurationBaseMinutes > 10080 || params.DurationPerRankMinutes < 0 || params.DurationPerRankMinutes > 1440 {
		return params, fmt.Errorf("RANK_SCALED_DEBUFF duration values are out of range")
	}
	if strings.TrimSpace(params.SourceName) == "" || len(params.SourceName) > 120 {
		return params, fmt.Errorf("RANK_SCALED_DEBUFF source_name must contain 1 to 120 characters")
	}
	return params, nil
}

// ValidateRankScaledDebuffParams validates the typed rank-scaled debuff config.
func ValidateRankScaledDebuffParams(subType string, raw json.RawMessage) error {
	_, err := ParseRankScaledDebuffParams(subType, raw)
	return err
}

// DispelToResourcesParams controls the dispel-to-resources scaling in RESOURCE_CONVERSION.
type DispelToResourcesParams struct {
	BaseConversionUnits  float64 `json:"base_conversion_units"`
	ManaPercentPerUnit   float64 `json:"mana_percent_per_unit"`
	ShieldPercentPerUnit float64 `json:"shield_percent_per_unit"`
	ClearControlStates   bool    `json:"clear_control_states"`
}

func ParseDispelToResourcesParams(raw json.RawMessage) (DispelToResourcesParams, error) {
	var params DispelToResourcesParams
	if len(raw) == 0 || json.Unmarshal(raw, &params) != nil {
		return params, fmt.Errorf("DISPEL_TO_RESOURCES params must be an object")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return params, fmt.Errorf("DISPEL_TO_RESOURCES params must be an object")
	}
	allowed := map[string]bool{
		"base_conversion_units": true, "mana_percent_per_unit": true,
		"shield_percent_per_unit": true, "clear_control_states": true,
	}
	for field := range fields {
		if !allowed[field] {
			return params, fmt.Errorf("DISPEL_TO_RESOURCES does not accept param %q", field)
		}
	}
	if math.IsNaN(params.BaseConversionUnits) || math.IsInf(params.BaseConversionUnits, 0) || params.BaseConversionUnits < 0 || params.BaseConversionUnits > 100 {
		return params, fmt.Errorf("DISPEL_TO_RESOURCES base_conversion_units is out of range [0, 100]")
	}
	if math.IsNaN(params.ManaPercentPerUnit) || math.IsInf(params.ManaPercentPerUnit, 0) || params.ManaPercentPerUnit < 0 || params.ManaPercentPerUnit > 10 {
		return params, fmt.Errorf("DISPEL_TO_RESOURCES mana_percent_per_unit is out of range [0, 10]")
	}
	if math.IsNaN(params.ShieldPercentPerUnit) || math.IsInf(params.ShieldPercentPerUnit, 0) || params.ShieldPercentPerUnit < 0 || params.ShieldPercentPerUnit > 10 {
		return params, fmt.Errorf("DISPEL_TO_RESOURCES shield_percent_per_unit is out of range [0, 10]")
	}
	return params, nil
}

func ValidateDispelToResourcesParams(raw json.RawMessage) error {
	_, err := ParseDispelToResourcesParams(raw)
	return err
}

// TargetRedirectionParams controls reactive attack redirection and confusion.
type TargetRedirectionParams struct {
	ActivationChance     float64 `json:"activation_chance"`
	HitUserChance        float64 `json:"hit_user_chance"`
	HitSelfChance        float64 `json:"hit_self_chance"`
	MissChance           float64 `json:"miss_chance"`
	BackfireChance       float64 `json:"backfire_chance"`
	BackfireDisableTurns int     `json:"backfire_disable_turns"`
}

func ParseTargetRedirectionParams(raw json.RawMessage) (TargetRedirectionParams, error) {
	var params TargetRedirectionParams
	if len(raw) == 0 || json.Unmarshal(raw, &params) != nil {
		return params, fmt.Errorf("TARGET_REDIRECTION params must be an object")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return params, fmt.Errorf("TARGET_REDIRECTION params must be an object")
	}
	allowed := map[string]bool{
		"activation_chance": true, "hit_user_chance": true, "hit_self_chance": true,
		"miss_chance": true, "backfire_chance": true, "backfire_disable_turns": true,
	}
	for field := range fields {
		if !allowed[field] {
			return params, fmt.Errorf("TARGET_REDIRECTION does not accept param %q", field)
		}
	}
	chances := []float64{params.ActivationChance, params.HitUserChance, params.HitSelfChance, params.MissChance, params.BackfireChance}
	for _, c := range chances {
		if math.IsNaN(c) || math.IsInf(c, 0) || c < 0 || c > 1 {
			return params, fmt.Errorf("TARGET_REDIRECTION chance values must be within [0, 1]")
		}
	}
	if params.BackfireDisableTurns < 0 || params.BackfireDisableTurns > 100 {
		return params, fmt.Errorf("TARGET_REDIRECTION backfire_disable_turns must be within [0, 100]")
	}
	return params, nil
}

func ValidateTargetRedirectionParams(subType string, raw json.RawMessage) error {
	if subType != "CONFUSION" {
		return fmt.Errorf("unknown TARGET_REDIRECTION sub_type %q", subType)
	}
	_, err := ParseTargetRedirectionParams(raw)
	return err
}

// ChaosGambleParams controls chaos gambling choices.
type ChaosGambleParams struct {
	EffectChoices []string `json:"effect_choices,omitempty"`
	CostChoices   []string `json:"cost_choices,omitempty"`
}

func ParseChaosGambleParams(raw json.RawMessage) (ChaosGambleParams, error) {
	var params ChaosGambleParams
	if len(raw) == 0 || string(raw) == "null" {
		return params, nil
	}
	if json.Unmarshal(raw, &params) != nil {
		return params, fmt.Errorf("OUTCOME_ROULETTE CHAOS params must be an object")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return params, fmt.Errorf("OUTCOME_ROULETTE CHAOS params must be an object")
	}
	allowed := map[string]bool{"effect_choices": true, "cost_choices": true}
	for field := range fields {
		if !allowed[field] {
			return params, fmt.Errorf("OUTCOME_ROULETTE CHAOS does not accept param %q", field)
		}
	}
	validEffects := map[string]bool{"crit_500": true, "invulnerable_1_turn": true, "reduce_enemy_hp_90": true}
	for _, eff := range params.EffectChoices {
		if !validEffects[eff] {
			return params, fmt.Errorf("invalid effect_choice %q", eff)
		}
	}
	validCosts := map[string]bool{"aptitude_10": true, "lifespan_5": true, "luck_50": true}
	for _, cost := range params.CostChoices {
		if !validCosts[cost] {
			return params, fmt.Errorf("invalid cost_choice %q", cost)
		}
	}
	return params, nil
}

func ValidateOutcomeRouletteParams(subType string, raw json.RawMessage) error {
	switch subType {
	case "LUCK":
		return ValidateLuckRouletteParams(raw)
	case "CHAOS":
		_, err := ParseChaosGambleParams(raw)
		return err
	default:
		return fmt.Errorf("unknown OUTCOME_ROULETTE sub_type %q", subType)
	}
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
var CapabilityCatalog = values("MARKETPLACE_ACCESS", "REMOTE_TRADE", "CHAT_CONNECT", "CHAT_PERCEIVE", "IGNORE_MOVEMENT_RANGE", "IGNORE_MONSTER_ENCOUNTER", "LOOT_PREVIEW", "DISGUISE", "CHANGE_FORM_DISGUISE", "PERFECT_DISGUISE", "CONCEAL_GU", "SEVER_CLUES", "PORTAL_CREATE", "RANDOM_PORTAL", "CAPTURE_BONUS")
var OperationCatalog = map[string]OperationDefinition{
	"DAMAGE":                  {values("FLAT", "MULTIPLIER", "MAX_HP_PERCENT"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "path", "chance", "phase"), []string{"value"}, true, false},
	"BONUS_STRIKE":            {nil, values("ON_ACTION"), values("TARGET"), values("op", "target", "chance", "value", "min_damage", "log_action", "message"), []string{"chance", "value", "min_damage", "log_action", "message"}, false, false},
	"CHAIN_DAMAGE":            {nil, values("ON_ACTION"), values("ALL_ENEMIES"), values("op", "target", "value", "min_damage", "target_count", "log_action", "message"), []string{"value", "min_damage", "target_count", "log_action", "message"}, false, false},
	"DAMAGE_MODIFIER":         {values("MISSING_HP", "TARGET_DEBUFF_COUNT", "TARGET_HAS_STATUS", "MINION_COUNT", "COUNTER_COUNT"), values("ON_ACTION", "COMBAT_PASSIVE"), combatTargets, values("op", "sub_type", "target", "value", "per_missing_hp", "per_target_debuff", "per_minion", "per_stack", "key", "status_id", "phase", "log_action", "message"), []string{"sub_type", "value"}, true, false},
	"DEFENSE_PENETRATION":     {nil, values("ON_ACTION"), combatTargets, values("op", "target", "value"), []string{"value"}, false, false},
	"TARGET_DAMAGE_MODIFIER":  {values("REDUCE"), values("ON_ACTION"), combatTargets, values("op", "sub_type", "target", "value", "min_multiplier", "log_action", "message"), []string{"sub_type", "value", "min_multiplier"}, false, false},
	"MITIGATE_DAMAGE":         {values("REDUCE", "CONVERT_TO_SHIELD"), values("BEFORE_HIT"), values("SELF"), values("op", "sub_type", "target", "value", "log_action", "message"), []string{"sub_type", "value"}, false, false},
	"DETONATE_STATUS":         {nil, values("ON_ACTION"), values("TARGET"), values("op", "target", "status_id", "per_stack", "per_strength", "max_hp_percent", "min_damage", "phase", "log_action", "message"), []string{"status_id", "per_stack", "per_strength", "max_hp_percent", "min_damage", "phase", "log_action", "message"}, true, false},
	"HP_COST":                 {values("CURRENT_HP_PERCENT"), values("ON_ACTION"), values("SELF"), values("op", "sub_type", "target", "value", "min_cost", "min_remaining", "phase", "log_action", "message"), []string{"sub_type", "value", "min_cost", "min_remaining", "log_action", "message"}, true, false},
	"SHIELD":                  {values("FLAT", "PERCENT", "MULTIPLIER"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "turns"), []string{"value"}, true, false},
	"HEAL":                    {values("FLAT", "PERCENT", "LIFESTEAL", "RESOURCE", "COUNTER_PERCENT"), anyTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "resource", "suppress_log", "key", "phase"), []string{"value"}, true, false},
	"DIRECT_HP":               {values("FLAT", "MAX_HP_PERCENT", "SHIELD_STRIP"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "rank_suppression"), []string{"value"}, true, false},
	"STAT_MOD":                {values("ADD_FLAT", "MULTIPLY_PERCENT", "SET", "GROWTH_CAP", "ADD_COUNTER_DELTA"), anyTriggers, combatTargets, values("op", "sub_type", "target", "stat", "value", "unit", "path", "turns", "stack_policy", "key", "per_stack"), []string{"stat", "value"}, false, false},
	"SET_RULE":                {values("IMMUNE_DREAM_STATUS", "IMMORTAL", "DISABLE_HP_REGEN", "DISABLE_MANA_REGEN", "DISABLE_RESOURCE_REGEN", "DISABLE_SOUL_REGEN"), values("ALWAYS_ON"), values("SELF"), values("op", "sub_type", "target", "enabled"), []string{"sub_type", "enabled"}, false, false},
	"TRANSFER_RESOURCE":       {values("PERCENT_OF_TARGET", "PERCENT_OF_MAX_HP", "PERCENT_OF_TARGET_SHIELD"), values("ON_ACTION"), values("TARGET"), values("op", "sub_type", "target", "resource", "value", "min_amount", "phase", "log_action", "message"), []string{"sub_type", "resource", "value"}, false, true},
	"RESOURCE_CONVERSION":     {values("CURRENT_HP_TO_SHIELD", "DISPEL_TO_RESOURCES"), values("ON_CAST", "ON_ACTION"), values("SELF"), values("op", "sub_type", "target", "value", "min_remaining", "log_action", "message", "params"), []string{"sub_type"}, true, false},
	"STATUS":                  {nil, anyTriggers, combatTargets, values("op", "target", "status_id", "value", "turns", "chance", "stack_policy", "application_mode", "phase", "message", "log_action", "suppress_log"), []string{"status_id", "turns", "chance"}, true, false},
	"BARRIER":                 {values("FLAT", "PERCENT"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value", "unit", "turns", "stack_policy"), []string{"value", "turns"}, true, false},
	"CLEANSE":                 {values("ALL", "DEBUFF", "STATUS"), anyTriggers, combatTargets, values("op", "sub_type", "target", "status_id", "phase"), nil, true, false},
	"DISPEL":                  {values("BUFFS", "ALL_BUFFS", "PROTECTION"), values("ON_ACTION", "ON_CAST"), combatTargets, values("op", "sub_type", "target", "value", "log_action", "message"), []string{"sub_type"}, false, false},
	"REQUIRE":                 {values("IMMORTAL"), values("ON_CONSUME"), values("SELF"), values("op", "sub_type", "target"), []string{"sub_type"}, false, false},
	"GRANT_RESOURCE":          {nil, values("ON_CONSUME"), values("SELF"), values("op", "target", "resource", "value"), []string{"resource", "value"}, true, false},
	"RESTORE_RESOURCE":        {nil, values("ON_CONSUME"), values("SELF"), values("op", "target", "resource", "value"), []string{"resource", "value"}, true, false},
	"MODIFY_MAX_RESOURCE":     {nil, values("ON_CONSUME"), values("SELF"), values("op", "target", "resource", "value"), []string{"resource", "value"}, true, false},
	"MODIFY_APTITUDE":         {values("ADD_FLAT"), values("ON_CONSUME"), values("SELF"), values("op", "sub_type", "target", "value"), []string{"sub_type", "value"}, true, false},
	"GRANT_BODY_MOD":          {values("SOVEREIGN_IMMORTAL"), values("ON_CONSUME"), values("SELF"), values("op", "sub_type", "target"), []string{"sub_type"}, false, false},
	"ADVANCE_PROGRESSION":     {nil, values("ON_CONSUME"), values("SELF"), values("op", "target"), nil, false, false},
	"ADVANCE_MAJOR_REALM":     {nil, values("ON_CONSUME"), values("SELF"), values("op", "target"), nil, false, false},
	"PETRIFY_APERTURE":        {nil, values("ON_CONSUME"), values("SELF"), values("op", "target"), nil, false, false},
	"MODIFY_APERTURE":         {values("AREA_SIZE", "TIME_RATE", "STRESS_REDUCE", "TIME_RATE_ADD", "TIME_RATE_SUBTRACT", "TIME_RATE_MULTIPLY"), values("ON_CONSUME"), values("SELF"), values("op", "target", "sub_type", "value"), []string{"sub_type", "value"}, true, false},
	"RANDOMIZE_TRAIT":         {values("RACE", "CONSTITUTION"), values("ON_CONSUME"), values("SELF"), values("op", "target", "sub_type"), []string{"sub_type"}, false, false},
	"REPAIR_APERTURE":         {nil, values("ON_CONSUME"), values("SELF"), values("op", "target", "value"), []string{"value"}, true, false},
	"OPEN_REWARD_CONTAINER":   {values("MORTAL_MINE", "IMMORTAL_MINE"), values("ON_CONSUME"), values("SELF"), values("op", "target", "sub_type"), []string{"sub_type"}, false, false},
	"PLACE_WORLD_OBJECT":      {values("PRIMEVAL_SPRING"), values("ON_CONSUME"), values("SELF"), values("op", "target", "sub_type"), []string{"sub_type"}, false, false},
	"DISPLACE":                {values("KNOCKBACK", "PULL", "SWAP", "RETREAT", "SET_ROW"), combatTriggers, combatTargets, values("op", "sub_type", "target", "value"), []string{"sub_type"}, true, false},
	"HIT_RULE":                {values("BYPASS_SHIELD", "COUNTER_THRESHOLD", "FIRST_TURN_EVADE", "GUARANTEED", "LETHAL_LUCK_WARD", "MULTI_HIT"), values("BEFORE_HIT", "ON_ACTION", "ON_BATTLE_START"), combatTargets, values("op", "sub_type", "target", "params"), []string{"sub_type"}, false, false},
	"TURN_ORDER_RULE":         {values("SEIZE_INITIATIVE", "SUPPRESS_ENEMY_PRIORITY"), values("ON_BATTLE_START"), values("SELF"), values("op", "sub_type", "target", "params"), []string{"sub_type"}, false, false},
	"END_BATTLE":              {values("FORCE_VICTORY"), values("ON_BATTLE_START"), values("ALL_ENEMIES"), values("op", "sub_type", "target", "log_action", "message"), []string{"sub_type"}, false, false},
	"GRANT_CAPABILITY":        {CapabilityCatalog, values("CAPABILITY"), values("SELF"), values("op", "sub_type", "target", "params"), []string{"sub_type"}, false, false},
	"GROWTH_DURATION":         {nil, values("ALWAYS_ON"), values("SELF"), values("op", "target", "value"), []string{"value"}, true, false},
	"PERIODIC_RESOURCE":       {values("REGEN_PER_HOUR", "GENERATION_WORKER_TICK"), values("ON_TICK"), values("SELF"), values("op", "sub_type", "target", "resource", "value"), []string{"sub_type", "resource", "value"}, true, false},
	"DEDUCTION_FINISH_CHANCE": {nil, values("ALWAYS_ON"), values("SELF"), values("op", "target", "params"), []string{"params"}, false, false},
	"SUMMON":                  {nil, values("ON_BATTLE_START", "ON_CAST"), values("SELF"), values("op", "target", "params"), []string{"params"}, false, false},
	"NOTIFY":                  {nil, values("ON_USE", "ON_CONSUME", "ON_ACTION", "BEFORE_HIT", "COMBAT_PASSIVE"), values("SELF"), values("op", "target", "message", "log_action", "phase", "key"), []string{"message"}, false, false},
	"APPLY_STACKING_STATUS":   {nil, combatTriggers, combatTargets, values("op", "target", "status_key", "max_stacks", "duration_hours", "dispel_rule", "pvp_only", "tier_scaling"), []string{"status_key", "max_stacks", "duration_hours", "dispel_rule", "pvp_only", "tier_scaling"}, false, false},
	"COUNTER_ADD":             {nil, combatTriggers, combatTargets, values("op", "target", "key", "delta", "min", "max"), []string{"key", "delta"}, false, false},
	"THRESHOLD_RECOVERY":      {nil, values("BEFORE_HIT"), values("SELF"), values("op", "target", "hp_threshold", "heal_percent", "max_uses", "cleanse_debuffs", "log_action", "message"), []string{"hp_threshold", "heal_percent", "max_uses", "cleanse_debuffs", "log_action", "message"}, false, false},
	"TENACITY_MODIFIER":       {nil, values("COMBAT_PASSIVE"), values("SELF"), values("op", "target", "value", "per_missing_hp"), []string{"per_missing_hp"}, false, false},
	"STATUS_IMMUNITY":         {values("STUN", "FREEZE"), values("COMBAT_PASSIVE"), values("SELF"), values("op", "sub_type", "target"), []string{"sub_type"}, false, false},
	"SLOT_INTERACTION":        {values("PERMANENT_STEAL", "TEMPORARY_STEAL"), values("ON_BATTLE_START"), values("TARGET"), values("op", "sub_type", "target", "params"), []string{"sub_type", "params"}, false, false},
	"REVIVE":                  {nil, values("ON_DEATH"), values("SELF"), values("op", "sub_type", "target", "params"), []string{"params"}, false, false},
	"EFFECT_IMMUNITY":         {values("HOSTILE_EFFECTS"), values("COMBAT_PASSIVE"), values("SELF"), values("op", "sub_type", "target"), []string{"sub_type"}, false, false},
	"RANK_SCALED_DEBUFF":      {values("REGIONAL_WALL"), values("ON_BATTLE_START"), values("TARGET"), values("op", "sub_type", "target", "params"), []string{"sub_type", "params"}, false, false},
	"GU_SEAL":                 {values("REACTIVE"), values("BEFORE_ENEMY_ATTACK"), values("SELF"), values("op", "sub_type", "target", "params"), []string{"sub_type", "params"}, false, false},
	"TARGET_REDIRECTION":      {values("CONFUSION"), values("BEFORE_ENEMY_ATTACK"), values("SELF"), values("op", "sub_type", "target", "params"), []string{"sub_type", "params"}, false, false},
	"OUTCOME_ROULETTE":        {values("LUCK", "CHAOS"), values("ON_ACTION", "ON_CAST"), combatTargets, values("op", "sub_type", "target", "params"), []string{"sub_type", "params"}, false, false},
	"REVOKE_STATUS":           {nil, values("ON_BATTLE_START"), values("SELF"), values("op", "target", "status_id"), []string{"status_id"}, false, false},
	"UNIQUE":                  {nil, anyTriggers, combatTargets, values("op", "target", "code", "params"), []string{"code"}, false, false},
}

// ValidateOperationCombination checks constraints that depend on multiple
// authored fields and therefore cannot be expressed by an individual catalog
// allowlist. It is shared by Block v2 validation and runtime compilation.
func ValidateOperationCombination(operation, subType, trigger, target string) error {
	switch operation {
	case "MODIFY_APTITUDE":
		if subType != "ADD_FLAT" || trigger != "ON_CONSUME" || target != "SELF" {
			return fmt.Errorf("MODIFY_APTITUDE ADD_FLAT requires ON_CONSUME and SELF")
		}
	case "GRANT_BODY_MOD":
		if subType != "SOVEREIGN_IMMORTAL" || trigger != "ON_CONSUME" || target != "SELF" {
			return fmt.Errorf("GRANT_BODY_MOD SOVEREIGN_IMMORTAL requires ON_CONSUME and SELF")
		}
	case "RESOURCE_CONVERSION":
		if subType == "CURRENT_HP_TO_SHIELD" {
			if trigger != "ON_CAST" || target != "SELF" {
				return fmt.Errorf("RESOURCE_CONVERSION CURRENT_HP_TO_SHIELD requires ON_CAST and SELF")
			}
		} else if subType == "DISPEL_TO_RESOURCES" {
			if (trigger != "ON_ACTION" && trigger != "ON_CAST") || target != "SELF" {
				return fmt.Errorf("RESOURCE_CONVERSION DISPEL_TO_RESOURCES requires ON_ACTION or ON_CAST and SELF")
			}
		} else {
			return fmt.Errorf("unknown RESOURCE_CONVERSION sub_type: %s", subType)
		}
	case "TARGET_REDIRECTION":
		if subType != "CONFUSION" || trigger != "BEFORE_ENEMY_ATTACK" || target != "SELF" {
			return fmt.Errorf("TARGET_REDIRECTION CONFUSION requires BEFORE_ENEMY_ATTACK and SELF")
		}
	case "OUTCOME_ROULETTE":
		if subType == "LUCK" {
			if trigger != "ON_ACTION" && trigger != "ON_CAST" {
				return fmt.Errorf("OUTCOME_ROULETTE LUCK requires ON_ACTION or ON_CAST")
			}
		} else if subType == "CHAOS" {
			if trigger != "ON_CAST" || target != "SELF" {
				return fmt.Errorf("OUTCOME_ROULETTE CHAOS requires ON_CAST and SELF")
			}
		} else {
			return fmt.Errorf("unknown OUTCOME_ROULETTE sub_type: %s", subType)
		}
	case "DAMAGE_MODIFIER":
		if (subType != "MISSING_HP" && subType != "TARGET_DEBUFF_COUNT" && subType != "TARGET_HAS_STATUS" && subType != "MINION_COUNT" && subType != "COUNTER_COUNT") || (trigger != "ON_ACTION" && trigger != "COMBAT_PASSIVE") {
			return fmt.Errorf("DAMAGE_MODIFIER %s requires ON_ACTION or COMBAT_PASSIVE", subType)
		}
		if trigger == "COMBAT_PASSIVE" && subType != "MINION_COUNT" && subType != "MISSING_HP" && subType != "COUNTER_COUNT" {
			return fmt.Errorf("DAMAGE_MODIFIER COMBAT_PASSIVE supports MINION_COUNT, MISSING_HP, or COUNTER_COUNT")
		}
		if (subType == "MINION_COUNT" || subType == "COUNTER_COUNT") && target != "SELF" {
			return fmt.Errorf("DAMAGE_MODIFIER %s requires SELF target", subType)
		}
	case "STAT_MOD":
		if subType == "ADD_COUNTER_DELTA" && (trigger != "BEFORE_HIT" || target != "SELF") {
			return fmt.Errorf("STAT_MOD ADD_COUNTER_DELTA requires BEFORE_HIT and SELF")
		}
	case "HEAL":
		if subType == "COUNTER_PERCENT" && (trigger != "COMBAT_PASSIVE" || target != "SELF") {
			return fmt.Errorf("HEAL COUNTER_PERCENT requires COMBAT_PASSIVE and SELF")
		}
	case "TENACITY_MODIFIER":
		if trigger != "COMBAT_PASSIVE" || target != "SELF" {
			return fmt.Errorf("TENACITY_MODIFIER requires COMBAT_PASSIVE and SELF")
		}
	case "STATUS_IMMUNITY":
		if (subType != "STUN" && subType != "FREEZE") || trigger != "COMBAT_PASSIVE" || target != "SELF" {
			return fmt.Errorf("STATUS_IMMUNITY STUN or FREEZE requires COMBAT_PASSIVE and SELF")
		}
	case "SLOT_INTERACTION":
		if (subType != "PERMANENT_STEAL" && subType != "TEMPORARY_STEAL") || trigger != "ON_BATTLE_START" || target != "TARGET" {
			return fmt.Errorf("SLOT_INTERACTION %s requires ON_BATTLE_START and TARGET", subType)
		}
	case "REVIVE":
		if trigger != "ON_DEATH" || target != "SELF" {
			return fmt.Errorf("REVIVE requires ON_DEATH and SELF")
		}
	case "EFFECT_IMMUNITY":
		if subType != "HOSTILE_EFFECTS" || trigger != "COMBAT_PASSIVE" || target != "SELF" {
			return fmt.Errorf("EFFECT_IMMUNITY HOSTILE_EFFECTS requires COMBAT_PASSIVE and SELF")
		}
	case "RANK_SCALED_DEBUFF":
		if subType != "REGIONAL_WALL" || trigger != "ON_BATTLE_START" || target != "TARGET" {
			return fmt.Errorf("RANK_SCALED_DEBUFF REGIONAL_WALL requires ON_BATTLE_START and TARGET")
		}
	case "GU_SEAL":
		if subType != "REACTIVE" || trigger != "BEFORE_ENEMY_ATTACK" || target != "SELF" {
			return fmt.Errorf("GU_SEAL REACTIVE requires BEFORE_ENEMY_ATTACK and SELF")
		}
	case "TARGET_DAMAGE_MODIFIER":
		if subType != "REDUCE" || trigger != "ON_ACTION" || target == "SELF" || target == "ALL_ALLIES" {
			return fmt.Errorf("TARGET_DAMAGE_MODIFIER REDUCE requires ON_ACTION and a hostile target")
		}
	case "BONUS_STRIKE":
		if trigger != "ON_ACTION" || target != "TARGET" {
			return fmt.Errorf("BONUS_STRIKE requires ON_ACTION and TARGET")
		}
	case "CHAIN_DAMAGE":
		if trigger != "ON_ACTION" || target != "ALL_ENEMIES" {
			return fmt.Errorf("CHAIN_DAMAGE requires ON_ACTION and ALL_ENEMIES")
		}
	case "DISPEL":
		if trigger != "ON_ACTION" && trigger != "ON_CAST" {
			return fmt.Errorf("DISPEL requires ON_ACTION or ON_CAST")
		}
	case "STATUS":
		if trigger == "ON_CAST" && target != "SELF" {
			return fmt.Errorf("STATUS ON_CAST requires SELF")
		}
	case "DEFENSE_PENETRATION":
		if trigger != "ON_ACTION" || target == "SELF" || target == "ALL_ALLIES" {
			return fmt.Errorf("DEFENSE_PENETRATION requires ON_ACTION and a hostile target")
		}
	case "MITIGATE_DAMAGE":
		if trigger != "BEFORE_HIT" || target != "SELF" {
			return fmt.Errorf("MITIGATE_DAMAGE requires BEFORE_HIT and SELF")
		}
	case "DETONATE_STATUS":
		if trigger != "ON_ACTION" || target != "TARGET" {
			return fmt.Errorf("DETONATE_STATUS requires ON_ACTION and TARGET")
		}
	case "HP_COST":
		if subType != "CURRENT_HP_PERCENT" || trigger != "ON_ACTION" || target != "SELF" {
			return fmt.Errorf("HP_COST CURRENT_HP_PERCENT requires ON_ACTION and SELF")
		}
	case "COUNTER_ADD":
		if trigger != "ON_ACTION" && trigger != "ON_TURN_START" && trigger != "BEFORE_HIT" {
			return fmt.Errorf("COUNTER_ADD requires ON_ACTION, ON_TURN_START, or BEFORE_HIT")
		}
		if (trigger == "ON_TURN_START" || trigger == "BEFORE_HIT") && target != "SELF" {
			return fmt.Errorf("COUNTER_ADD %s requires SELF", trigger)
		}
	case "REVOKE_STATUS":
		if trigger != "ON_BATTLE_START" || target != "SELF" {
			return fmt.Errorf("REVOKE_STATUS requires ON_BATTLE_START and SELF")
		}
	case "DISPLACE":
		if subType == "SET_ROW" && (trigger != "ON_BATTLE_START" || target != "SELF") {
			return fmt.Errorf("DISPLACE SET_ROW requires ON_BATTLE_START and SELF")
		}
		if trigger == "ON_BATTLE_START" && (target != "SELF" || (subType != "RETREAT" && subType != "SET_ROW")) {
			return fmt.Errorf("DISPLACE ON_BATTLE_START requires RETREAT or SET_ROW and SELF")
		}
	case "END_BATTLE":
		if subType != "FORCE_VICTORY" || trigger != "ON_BATTLE_START" || target != "ALL_ENEMIES" {
			return fmt.Errorf("END_BATTLE FORCE_VICTORY requires ON_BATTLE_START and ALL_ENEMIES")
		}
	case "NOTIFY":
		if (trigger == "ON_ACTION" || trigger == "BEFORE_HIT" || trigger == "COMBAT_PASSIVE") && target != "SELF" {
			return fmt.Errorf("NOTIFY combat log requires SELF target")
		}
	case "APPLY_STACKING_STATUS":
		if trigger != "ON_HIT" {
			return fmt.Errorf("APPLY_STACKING_STATUS requires ON_HIT")
		}
		if target != "TARGET" {
			return fmt.Errorf("APPLY_STACKING_STATUS currently requires TARGET")
		}
	case "THRESHOLD_CHECK":
		return fmt.Errorf("THRESHOLD_CHECK has no runtime executor")
	case "THRESHOLD_RECOVERY":
		if trigger != "BEFORE_HIT" || target != "SELF" {
			return fmt.Errorf("THRESHOLD_RECOVERY requires BEFORE_HIT and SELF")
		}
	case "HIT_RULE":
		switch subType {
		case "COUNTER_THRESHOLD":
			if trigger != "ON_BATTLE_START" || target != "SELF" {
				return fmt.Errorf("HIT_RULE COUNTER_THRESHOLD requires ON_BATTLE_START and SELF")
			}
		case "MULTI_HIT":
			if trigger != "ON_ACTION" || target == "SELF" {
				return fmt.Errorf("HIT_RULE MULTI_HIT requires ON_ACTION and a hostile target")
			}
		case "FIRST_TURN_EVADE":
			if trigger != "BEFORE_HIT" || target != "SELF" {
				return fmt.Errorf("HIT_RULE FIRST_TURN_EVADE requires BEFORE_HIT and SELF")
			}
		case "LETHAL_LUCK_WARD":
			if trigger != "BEFORE_HIT" || target != "SELF" {
				return fmt.Errorf("HIT_RULE LETHAL_LUCK_WARD requires BEFORE_HIT and SELF")
			}
		case "GUARANTEED", "BYPASS_SHIELD":
			if trigger != "ON_ACTION" || target == "SELF" {
				return fmt.Errorf("HIT_RULE %s requires ON_ACTION and a hostile target", subType)
			}
		}
	case "TURN_ORDER_RULE":
		if trigger != "ON_BATTLE_START" || target != "SELF" {
			return fmt.Errorf("TURN_ORDER_RULE requires ON_BATTLE_START and SELF")
		}
	case "DEDUCTION_FINISH_CHANCE":
		if trigger != "ALWAYS_ON" || target != "SELF" {
			return fmt.Errorf("DEDUCTION_FINISH_CHANCE requires ALWAYS_ON and SELF")
		}
	}
	return nil
}

// ValidateReviveParams keeps specialized resurrection behavior behind the
// typed REVIVE operation while reusing each executor's strict parameter schema.
func ValidateReviveParams(subType string, params json.RawMessage) error {
	code := "REBORN"
	if subType != "" {
		if subType != "NIRVANA_FIRE" {
			return fmt.Errorf("unknown REVIVE subtype %q", subType)
		}
		code = subType
	}
	return ValidateNativeParams(code, params)
}

// ValidateResourceConversion restricts conversions to the implemented combat contract.
func ValidateResourceConversion(subType string, fraction, minRemaining float64, logAction, message string) error {
	if subType != "CURRENT_HP_TO_SHIELD" || math.IsNaN(fraction) || math.IsInf(fraction, 0) || fraction <= 0 || fraction > 1 ||
		math.IsNaN(minRemaining) || math.IsInf(minRemaining, 0) || minRemaining < 1 || minRemaining > 1e12 {
		return fmt.Errorf("CURRENT_HP_TO_SHIELD requires value in (0,1] and min_remaining >= 1")
	}
	if logAction != "" {
		if err := ValidateCombatLogAction(logAction); err != nil {
			return err
		}
	}
	if message != "" {
		return ValidateNotifyMessage(message)
	}
	return nil
}

// ValidatePersistentStatusID checks the identifier syntax for player-persisted
// statuses, which use a separate registry from turn-based combat statusfx.
func ValidatePersistentStatusID(statusID string) error {
	if statusID == "" || len(statusID) > 64 {
		return fmt.Errorf("persistent status_id must contain 1 to 64 characters")
	}
	for _, r := range statusID {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return fmt.Errorf("persistent status_id %q must use lowercase letters, digits, or underscores", statusID)
		}
	}
	return nil
}

// ValidateCounterAdd rejects counter updates that can only produce silent no-ops
// or contradictory bounds.
func ValidateCounterAdd(key string, delta, minimum, maximum int) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("COUNTER_ADD requires a non-empty key")
	}
	if delta == 0 {
		return fmt.Errorf("COUNTER_ADD delta must not be zero")
	}
	if maximum < 0 {
		return fmt.Errorf("COUNTER_ADD maximum must be zero (unbounded) or positive")
	}
	if maximum > 0 && minimum > maximum {
		return fmt.Errorf("COUNTER_ADD minimum must not exceed maximum")
	}
	return nil
}

// ValidateSlotInteractionParams validates the cross-combat Gu-slot transfer
// contract. Chance formula fields are explicit so CSV data owns tuning.
func ValidateSlotInteractionParams(mode string, raw json.RawMessage) error {
	var params struct {
		BaseChance      *float64 `json:"base_chance"`
		ChanceDelta     *float64 `json:"chance_delta"`
		VitalPenalty    *float64 `json:"vital_penalty"`
		TargetSelection string   `json:"target_selection"`
		Duration        string   `json:"duration"`
	}
	var fields map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &fields) != nil {
		return fmt.Errorf("SLOT_INTERACTION params must be an object")
	}
	allowed := map[string]bool{"base_chance": true, "chance_delta": true, "vital_penalty": true, "target_selection": true, "duration": true}
	for field := range fields {
		if !allowed[field] {
			return fmt.Errorf("SLOT_INTERACTION does not support param %q", field)
		}
	}
	if json.Unmarshal(raw, &params) != nil || params.BaseChance == nil || params.ChanceDelta == nil || params.VitalPenalty == nil {
		return fmt.Errorf("SLOT_INTERACTION requires base_chance, chance_delta, and vital_penalty")
	}
	for name, value := range map[string]float64{"base_chance": *params.BaseChance, "chance_delta": *params.ChanceDelta, "vital_penalty": *params.VitalPenalty} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
			return fmt.Errorf("SLOT_INTERACTION %s must be in [0,1]", name)
		}
	}
	if params.TargetSelection != "RANDOM" {
		return fmt.Errorf("SLOT_INTERACTION target_selection must be RANDOM")
	}
	switch mode {
	case "PERMANENT_STEAL":
		if params.Duration != "" {
			return fmt.Errorf("SLOT_INTERACTION PERMANENT_STEAL does not accept duration")
		}
	case "TEMPORARY_STEAL":
		if params.Duration != "FULL_COMBAT" {
			return fmt.Errorf("SLOT_INTERACTION TEMPORARY_STEAL requires duration FULL_COMBAT")
		}
	default:
		return fmt.Errorf("SLOT_INTERACTION mode must be PERMANENT_STEAL or TEMPORARY_STEAL")
	}
	return nil
}

type StackingStatusTierValue struct {
	Tier        int     `json:"tier"`
	StatPercent float64 `json:"stat_percent"`
}

// ValidateStackingStatus closes the authored contract consumed by the
// post-combat persistent-status adapter.
func ValidateStackingStatus(statusKey string, maxStacks int, durationHours float64, dispelRule string, pvpOnly bool, tiers []StackingStatusTierValue) error {
	if strings.TrimSpace(statusKey) == "" {
		return fmt.Errorf("APPLY_STACKING_STATUS requires status_key")
	}
	if maxStacks < 1 || maxStacks > 100 {
		return fmt.Errorf("APPLY_STACKING_STATUS max_stacks must be between 1 and 100")
	}
	if math.IsNaN(durationHours) || math.IsInf(durationHours, 0) || durationHours <= 0 || durationHours > 87600 {
		return fmt.Errorf("APPLY_STACKING_STATUS duration_hours must be in (0, 87600]")
	}
	if !pvpOnly {
		return fmt.Errorf("APPLY_STACKING_STATUS currently supports persistent PvP status only")
	}
	dispelRule = strings.ToUpper(strings.TrimSpace(dispelRule))
	if dispelRule != "DECREMENT_ONE" && dispelRule != "CLEAR_ALL" {
		return fmt.Errorf("APPLY_STACKING_STATUS dispel_rule must be DECREMENT_ONE or CLEAR_ALL")
	}
	if len(tiers) != maxStacks {
		return fmt.Errorf("APPLY_STACKING_STATUS tier_scaling must define every tier from 1 through max_stacks")
	}
	seen := make(map[int]bool, len(tiers))
	for _, tier := range tiers {
		if tier.Tier < 1 || tier.Tier > maxStacks || seen[tier.Tier] {
			return fmt.Errorf("APPLY_STACKING_STATUS tier_scaling tiers must be unique integers from 1 through max_stacks")
		}
		if math.IsNaN(tier.StatPercent) || math.IsInf(tier.StatPercent, 0) || tier.StatPercent >= 0 || tier.StatPercent < -100 {
			return fmt.Errorf("APPLY_STACKING_STATUS stat_percent values must be between -100 and 0")
		}
		seen[tier.Tier] = true
	}
	return nil
}

// ValidateCombatStatusID verifies a status identifier used by a combat operation.
func ValidateCombatStatusID(statusID string) error {
	statusID = strings.ToUpper(strings.TrimSpace(statusID))
	if !StatusCatalog[statusID] {
		return fmt.Errorf("unknown status_id %q", statusID)
	}
	return nil
}

// ValidateDeductionFinishChanceParams checks the authored rank-scaled chance
// used by recipe deduction. Omitted values retain the consumer's established
// defaults; supplied values must be finite probabilities (or a signed step).
func ValidateDeductionFinishChanceParams(raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("DEDUCTION_FINISH_CHANCE params are required")
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(raw, &params); err != nil || params == nil {
		return fmt.Errorf("DEDUCTION_FINISH_CHANCE params must be an object")
	}
	allowed := map[string]bool{"base_chance": true, "per_rank_step": true, "min_chance": true, "max_chance": true}
	values := map[string]float64{}
	for key, value := range params {
		if !allowed[key] {
			return fmt.Errorf("DEDUCTION_FINISH_CHANCE does not support param %q", key)
		}
		var number float64
		if strings.TrimSpace(string(value)) == "null" || json.Unmarshal(value, &number) != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return fmt.Errorf("DEDUCTION_FINISH_CHANCE %s must be a finite number", key)
		}
		if key == "per_rank_step" {
			if number < -1 || number > 1 {
				return fmt.Errorf("DEDUCTION_FINISH_CHANCE per_rank_step must be between -1 and 1")
			}
		} else if number < 0 || number > 1 {
			return fmt.Errorf("DEDUCTION_FINISH_CHANCE %s must be between 0 and 1", key)
		}
		values[key] = number
	}
	if min, ok := values["min_chance"]; ok {
		if max, exists := values["max_chance"]; exists && min > max {
			return fmt.Errorf("DEDUCTION_FINISH_CHANCE min_chance cannot exceed max_chance")
		}
	}
	return nil
}

// ValidateStatusDetonation checks a bounded formula for consuming status
// stacks and applying direct HP damage in combat.
func ValidateStatusDetonation(statusID string, perStack, perStrength, maxHPPercent, minDamage float64, phase string) error {
	statusID = strings.ToUpper(strings.TrimSpace(statusID))
	if !StatusCatalog[statusID] {
		return fmt.Errorf("DETONATE_STATUS requires a known status_id")
	}
	if math.IsNaN(perStack) || math.IsInf(perStack, 0) || perStack < 0 || perStack > 1e12 ||
		math.IsNaN(perStrength) || math.IsInf(perStrength, 0) || perStrength < 0 || perStrength > 1e6 ||
		math.IsNaN(maxHPPercent) || math.IsInf(maxHPPercent, 0) || maxHPPercent < 0 || maxHPPercent > 10 ||
		math.IsNaN(minDamage) || math.IsInf(minDamage, 0) || minDamage < 0 || minDamage > 1e12 ||
		(perStack == 0 && perStrength == 0 && maxHPPercent == 0) {
		return fmt.Errorf("DETONATE_STATUS requires non-negative formula values and at least one positive damage coefficient")
	}
	if phase != "EARLY_RIDER" && phase != "LATE_RIDER" {
		return fmt.Errorf("DETONATE_STATUS phase must be EARLY_RIDER or LATE_RIDER")
	}
	return nil
}

// ValidateHPCost validates a bounded self-HP cost for a combat action.
func ValidateHPCost(subType string, percent, minCost, minRemaining float64, logAction, message string) error {
	if subType != "CURRENT_HP_PERCENT" {
		return fmt.Errorf("unknown HP_COST subtype %q", subType)
	}
	if math.IsNaN(percent) || math.IsInf(percent, 0) || percent <= 0 || percent > 1 ||
		math.IsNaN(minCost) || math.IsInf(minCost, 0) || minCost < 0 || minCost > 1e12 ||
		math.IsNaN(minRemaining) || math.IsInf(minRemaining, 0) || minRemaining < 1 || minRemaining > 1e12 {
		return fmt.Errorf("HP_COST requires percent in (0,1], non-negative min_cost, and min_remaining of at least 1")
	}
	if err := ValidateCombatLogAction(logAction); err != nil {
		return err
	}
	return ValidateNotifyMessage(message)
}

// ValidateModifyAptitude validates a positive integer aptitude increase.
func ValidateModifyAptitude(subType string, value float64) error {
	if subType != "ADD_FLAT" {
		return fmt.Errorf("unknown MODIFY_APTITUDE subtype %q", subType)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 1 || value > 99 || math.Trunc(value) != value {
		return fmt.Errorf("MODIFY_APTITUDE value must be an integer between 1 and 99")
	}
	return nil
}

// ValidateDamageModifier validates a dynamic combat damage multiplier and its
// optional authored combat log.
func ValidateDamageModifier(subType string, base, perMissingHP, perTargetDebuff, perMinion, perCounter float64, counterKey, statusID, logAction, message string) error {
	if math.IsNaN(base) || math.IsInf(base, 0) || base <= 0 || base > 10 {
		return fmt.Errorf("DAMAGE_MODIFIER base multiplier must be greater than 0 and at most 10")
	}
	switch subType {
	case "MISSING_HP":
		if math.IsNaN(perMissingHP) || math.IsInf(perMissingHP, 0) || perMissingHP < 0 || perMissingHP > 10 || perTargetDebuff != 0 || perMinion != 0 || perCounter != 0 || counterKey != "" {
			return fmt.Errorf("DAMAGE_MODIFIER MISSING_HP requires per_missing_hp between 0 and 10 and no per_target_debuff")
		}
	case "TARGET_DEBUFF_COUNT":
		if math.IsNaN(perTargetDebuff) || math.IsInf(perTargetDebuff, 0) || perTargetDebuff < 0 || perTargetDebuff > 10 || perMissingHP != 0 || perMinion != 0 || perCounter != 0 || counterKey != "" {
			return fmt.Errorf("DAMAGE_MODIFIER TARGET_DEBUFF_COUNT requires per_target_debuff between 0 and 10 and no per_missing_hp")
		}
	case "TARGET_HAS_STATUS":
		statusID = strings.ToUpper(strings.TrimSpace(statusID))
		if !StatusCatalog[statusID] || perMissingHP != 0 || perTargetDebuff != 0 || perMinion != 0 || perCounter != 0 || counterKey != "" {
			return fmt.Errorf("DAMAGE_MODIFIER TARGET_HAS_STATUS requires a known status_id and no other coefficient")
		}
	case "MINION_COUNT":
		if math.IsNaN(perMinion) || math.IsInf(perMinion, 0) || perMinion <= 0 || perMinion > 10 || perMissingHP != 0 || perTargetDebuff != 0 || perCounter != 0 || counterKey != "" {
			return fmt.Errorf("DAMAGE_MODIFIER MINION_COUNT requires per_minion between 0 and 10 and no other coefficient")
		}
	case "COUNTER_COUNT":
		if strings.TrimSpace(counterKey) == "" || math.IsNaN(perCounter) || math.IsInf(perCounter, 0) || perCounter < 0 || perCounter > 10 || perMissingHP != 0 || perTargetDebuff != 0 || perMinion != 0 {
			return fmt.Errorf("DAMAGE_MODIFIER COUNTER_COUNT requires key and per_stack between 0 and 10 with no other coefficient")
		}
	default:
		return fmt.Errorf("unknown DAMAGE_MODIFIER subtype %q", subType)
	}
	if message != "" {
		if err := ValidateNotifyMessage(message); err != nil {
			return fmt.Errorf("DAMAGE_MODIFIER message: %w", err)
		}
		if err := ValidateCombatLogAction(logAction); err != nil {
			return fmt.Errorf("DAMAGE_MODIFIER log_action: %w", err)
		}
	} else if logAction != "" {
		return fmt.Errorf("DAMAGE_MODIFIER log_action requires message")
	}
	return nil
}

// ValidateCounterHeal rejects empty/invalid counter-scaled healing formulas.
func ValidateCounterHeal(key string, perCounter float64) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("HEAL COUNTER_PERCENT requires key")
	}
	if math.IsNaN(perCounter) || math.IsInf(perCounter, 0) || perCounter <= 0 || perCounter > 10 {
		return fmt.Errorf("HEAL COUNTER_PERCENT value must be greater than 0 and at most 10")
	}
	return nil
}

// ValidateTargetDamageModifier validates a permanent-in-combat reduction to a
// target's outgoing damage multiplier and its optional authored log.
func ValidateTargetDamageModifier(subType string, reduction, minimumMultiplier float64, logAction, message string) error {
	if subType != "REDUCE" {
		return fmt.Errorf("unknown TARGET_DAMAGE_MODIFIER subtype %q", subType)
	}
	if math.IsNaN(reduction) || math.IsInf(reduction, 0) || reduction <= 0 || reduction > 1 {
		return fmt.Errorf("TARGET_DAMAGE_MODIFIER REDUCE value must be greater than 0 and at most 1")
	}
	if math.IsNaN(minimumMultiplier) || math.IsInf(minimumMultiplier, 0) || minimumMultiplier < 0 || minimumMultiplier > 1 {
		return fmt.Errorf("TARGET_DAMAGE_MODIFIER min_multiplier must be between 0 and 1")
	}
	if message != "" {
		if err := ValidateNotifyMessage(message); err != nil {
			return fmt.Errorf("TARGET_DAMAGE_MODIFIER message: %w", err)
		}
		if err := ValidateCombatLogAction(logAction); err != nil {
			return fmt.Errorf("TARGET_DAMAGE_MODIFIER log_action: %w", err)
		}
	} else if logAction != "" {
		return fmt.Errorf("TARGET_DAMAGE_MODIFIER log_action requires message")
	}
	return nil
}

// ValidateDefensePenetration validates a typed fraction of target defense to
// ignore for the current combatant's subsequent attacks.
func ValidateDefensePenetration(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > 1 {
		return fmt.Errorf("DEFENSE_PENETRATION value must be greater than 0 and at most 1")
	}
	return nil
}

// ValidateBonusStrike validates an authored probabilistic follow-up hit.
func ValidateBonusStrike(chance, damageMultiplier, minDamage float64, logAction, message string) error {
	if math.IsNaN(chance) || math.IsInf(chance, 0) || chance <= 0 || chance > 1 {
		return fmt.Errorf("BONUS_STRIKE chance must be greater than 0 and at most 1")
	}
	if math.IsNaN(damageMultiplier) || math.IsInf(damageMultiplier, 0) || damageMultiplier <= 0 || damageMultiplier > 10 {
		return fmt.Errorf("BONUS_STRIKE value must be greater than 0 and at most 10")
	}
	if math.IsNaN(minDamage) || math.IsInf(minDamage, 0) || minDamage < 0 || minDamage > 1e9 {
		return fmt.Errorf("BONUS_STRIKE min_damage must be between 0 and 1000000000")
	}
	if err := ValidateCombatLogAction(logAction); err != nil {
		return fmt.Errorf("BONUS_STRIKE log_action: %w", err)
	}
	if err := ValidateNotifyMessage(message); err != nil {
		return fmt.Errorf("BONUS_STRIKE message: %w", err)
	}
	return nil
}

// ValidateChainDamage checks a bounded direct follow-up damage operation.
func ValidateChainDamage(damageFraction, minDamage float64, targetCount int, logAction, message string) error {
	if math.IsNaN(damageFraction) || math.IsInf(damageFraction, 0) || damageFraction <= 0 || damageFraction > 10 {
		return fmt.Errorf("CHAIN_DAMAGE value must be greater than 0 and at most 10")
	}
	if math.IsNaN(minDamage) || math.IsInf(minDamage, 0) || minDamage < 0 || minDamage > 1e9 {
		return fmt.Errorf("CHAIN_DAMAGE min_damage must be between 0 and 1000000000")
	}
	if targetCount < 1 || targetCount > 10 {
		return fmt.Errorf("CHAIN_DAMAGE target_count must be between 1 and 10")
	}
	if err := ValidateCombatLogAction(logAction); err != nil {
		return fmt.Errorf("CHAIN_DAMAGE log_action: %w", err)
	}
	if err := ValidateNotifyMessage(message); err != nil {
		return fmt.Errorf("CHAIN_DAMAGE message: %w", err)
	}
	return nil
}

// ValidateMitigateDamage validates a before-hit damage conversion rule.
func ValidateMitigateDamage(subType string, value float64, logAction, message string) error {
	if subType != "REDUCE" && subType != "CONVERT_TO_SHIELD" {
		return fmt.Errorf("unknown MITIGATE_DAMAGE subtype %q", subType)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > 1 {
		return fmt.Errorf("MITIGATE_DAMAGE value must be greater than 0 and at most 1")
	}
	if (logAction == "") != (message == "") {
		return fmt.Errorf("MITIGATE_DAMAGE log_action and message must be provided together")
	}
	if logAction != "" {
		if err := ValidateCombatLogAction(logAction); err != nil {
			return err
		}
		if err := ValidateNotifyMessage(message); err != nil {
			return err
		}
	}
	return nil
}

// ValidateDispel bounds the number of positive effects an authored combat
// operation can remove and validates its optional log template.
func ValidateDispel(subType string, maxCount float64, logAction, message string) error {
	switch subType {
	case "BUFFS":
		if math.IsNaN(maxCount) || math.IsInf(maxCount, 0) || maxCount < 1 || maxCount > 10 || math.Trunc(maxCount) != maxCount {
			return fmt.Errorf("DISPEL BUFFS value must be an integer between 1 and 10")
		}
	case "ALL_BUFFS":
		if maxCount != 0 {
			return fmt.Errorf("DISPEL ALL_BUFFS does not accept a value")
		}
	case "PROTECTION":
		if maxCount != 0 {
			return fmt.Errorf("DISPEL PROTECTION does not accept a value")
		}
	default:
		return fmt.Errorf("unknown DISPEL subtype %q", subType)
	}
	if message != "" {
		if err := ValidateNotifyMessage(message); err != nil {
			return fmt.Errorf("DISPEL message: %w", err)
		}
		if err := ValidateCombatLogAction(logAction); err != nil {
			return fmt.Errorf("DISPEL log_action: %w", err)
		}
	} else if logAction != "" {
		return fmt.Errorf("DISPEL log_action requires message")
	}
	return nil
}

// ValidateCounterStatModifier validates a stat increase tied to an applied
// incoming-hit counter delta. The runtime uses the effective counter delta so
// hitting a configured cap does not grant extra stats.
func ValidateCounterStatModifier(stat, key string, perStack float64) error {
	stat = strings.ToUpper(strings.TrimSpace(stat))
	if stat != "ATTACK" && stat != "DEFENSE" {
		return fmt.Errorf("STAT_MOD ADD_COUNTER_DELTA supports ATTACK or DEFENSE")
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("STAT_MOD ADD_COUNTER_DELTA requires a non-empty key")
	}
	if math.IsNaN(perStack) || math.IsInf(perStack, 0) || perStack <= 0 || perStack > 1e9 {
		return fmt.Errorf("STAT_MOD ADD_COUNTER_DELTA per_stack must be greater than 0 and at most 1000000000")
	}
	return nil
}

// ValidateThresholdRecovery checks a bounded, deterministic low-health recovery.
func ValidateThresholdRecovery(hpThreshold, healPercent float64, maxUses int, cleanseDebuffs bool, logAction, message string) error {
	if math.IsNaN(hpThreshold) || math.IsInf(hpThreshold, 0) || hpThreshold <= 0 || hpThreshold > 1 {
		return fmt.Errorf("THRESHOLD_RECOVERY hp_threshold must be greater than 0 and at most 1")
	}
	if math.IsNaN(healPercent) || math.IsInf(healPercent, 0) || healPercent <= 0 || healPercent > 10 {
		return fmt.Errorf("THRESHOLD_RECOVERY heal_percent must be greater than 0 and at most 10")
	}
	if maxUses < 1 || maxUses > 100 {
		return fmt.Errorf("THRESHOLD_RECOVERY max_uses must be between 1 and 100")
	}
	if !cleanseDebuffs {
		return fmt.Errorf("THRESHOLD_RECOVERY requires cleanse_debuffs")
	}
	if err := ValidateCombatLogAction(logAction); err != nil {
		return fmt.Errorf("THRESHOLD_RECOVERY log_action: %w", err)
	}
	if err := ValidateNotifyMessage(message); err != nil {
		return fmt.Errorf("THRESHOLD_RECOVERY message: %w", err)
	}
	return nil
}

// ValidateTurnOrderRuleParams validates data-driven battle-start priority rules.
func ValidateTurnOrderRuleParams(subType string, raw json.RawMessage) error {
	if subType == "SUPPRESS_ENEMY_PRIORITY" {
		if len(raw) != 0 {
			return fmt.Errorf("TURN_ORDER_RULE SUPPRESS_ENEMY_PRIORITY does not accept params")
		}
		return nil
	}
	if subType != "SEIZE_INITIATIVE" {
		return fmt.Errorf("unknown TURN_ORDER_RULE subtype %q", subType)
	}
	var values map[string]float64
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil || len(values) != 1 {
		return fmt.Errorf("TURN_ORDER_RULE SEIZE_INITIATIVE params must contain lifespan_ratio")
	}
	ratio, ok := values["lifespan_ratio"]
	if !ok || math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio <= 0 || ratio > 100 {
		return fmt.Errorf("TURN_ORDER_RULE SEIZE_INITIATIVE lifespan_ratio must be greater than 0 and at most 100")
	}
	return nil
}

// ValidateHitRuleParams validates authored parameters for specialized hit rules.
func ValidateHitRuleParams(subType string, raw json.RawMessage) error {
	if subType == "COUNTER_THRESHOLD" {
		var params struct {
			Key            string   `json:"key"`
			Threshold      *float64 `json:"threshold"`
			Guaranteed     bool     `json:"guaranteed"`
			BypassShield   bool     `json:"bypass_shield"`
			BypassImmunity bool     `json:"bypass_immunity"`
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if len(raw) == 0 || decoder.Decode(&params) != nil || decoder.Decode(&struct{}{}) != io.EOF || strings.TrimSpace(params.Key) == "" || params.Threshold == nil || math.IsNaN(*params.Threshold) || math.IsInf(*params.Threshold, 0) || *params.Threshold < 0 || *params.Threshold > 1e6 || *params.Threshold != math.Trunc(*params.Threshold) || (!params.Guaranteed && !params.BypassShield && !params.BypassImmunity) {
			return fmt.Errorf("HIT_RULE COUNTER_THRESHOLD params require key, integer threshold, and at least one hit rule")
		}
		return nil
	}
	if subType == "MULTI_HIT" {
		var values map[string]float64
		if len(raw) == 0 || json.Unmarshal(raw, &values) != nil || len(values) != 1 {
			return fmt.Errorf("HIT_RULE MULTI_HIT params must contain count")
		}
		count, ok := values["count"]
		if !ok || math.IsNaN(count) || math.IsInf(count, 0) || count < 2 || count > 10 || count != math.Trunc(count) {
			return fmt.Errorf("HIT_RULE MULTI_HIT count must be an integer between 2 and 10")
		}
		return nil
	}
	if subType != "LETHAL_LUCK_WARD" {
		if len(raw) != 0 {
			return fmt.Errorf("HIT_RULE %s does not accept params", subType)
		}
		return nil
	}
	var values map[string]float64
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil || len(values) != 3 {
		return fmt.Errorf("HIT_RULE LETHAL_LUCK_WARD params must contain chance, shield_percent, and shield_min")
	}
	chance, hasChance := values["chance"]
	shieldPercent, hasPercent := values["shield_percent"]
	shieldMin, hasMin := values["shield_min"]
	if !hasChance || !hasPercent || !hasMin || math.IsNaN(chance) || math.IsInf(chance, 0) || chance < 0 || chance > 1 || math.IsNaN(shieldPercent) || math.IsInf(shieldPercent, 0) || shieldPercent < 0 || shieldPercent > 10 || math.IsNaN(shieldMin) || math.IsInf(shieldMin, 0) || shieldMin < 0 || shieldMin > 1e9 {
		return fmt.Errorf("HIT_RULE LETHAL_LUCK_WARD params are outside supported ranges")
	}
	return nil
}

// ValidateNotifyMessage keeps player-visible text bounded and non-empty.
func ValidateNotifyMessage(message string) error {
	message = strings.TrimSpace(message)
	if message == "" || len([]rune(message)) > 500 {
		return fmt.Errorf("message must contain 1 to 500 characters")
	}
	return nil
}

// ValidateNotifyOperation keeps combat passive notifications on an executable
// phase and requires a counter source for dynamic counter payout placeholders.
func ValidateNotifyOperation(trigger, target, phase, key, message string) error {
	if err := ValidateNotifyMessage(message); err != nil {
		return err
	}
	if trigger == "COMBAT_PASSIVE" {
		if target != "SELF" || phase != "FINAL_RIDER" {
			return fmt.Errorf("NOTIFY COMBAT_PASSIVE requires SELF and FINAL_RIDER")
		}
	} else if trigger == "ON_ACTION" {
		if target != "SELF" || (phase != "" && phase != "PRE_CHAIN") || key != "" {
			return fmt.Errorf("NOTIFY ON_ACTION requires SELF and supports only PRE_CHAIN without a key")
		}
	} else if trigger == "BEFORE_HIT" {
		if target != "SELF" || phase != "BEFORE_HIT" || !validCounterKey(key) || !strings.Contains(message, "{counter_count}") {
			return fmt.Errorf("NOTIFY BEFORE_HIT requires SELF, BEFORE_HIT phase, counter key, and {counter_count}")
		}
	} else if phase != "" || key != "" {
		return fmt.Errorf("NOTIFY phase and key require a supported combat trigger")
	}
	needsCounter := strings.Contains(message, "{counter_count}") || strings.Contains(message, "{damage_percent}") || strings.Contains(message, "{heal_amount}")
	if needsCounter && ((trigger != "COMBAT_PASSIVE" && trigger != "BEFORE_HIT") || !validCounterKey(key)) {
		return fmt.Errorf("dynamic counter placeholders require COMBAT_PASSIVE or BEFORE_HIT and a valid key")
	}
	if key != "" && !validCounterKey(key) {
		return fmt.Errorf("NOTIFY key must be a lowercase counter identifier")
	}
	return nil
}

func validCounterKey(key string) bool {
	if key == "" || key[0] < 'a' || key[0] > 'z' {
		return false
	}
	for _, char := range key[1:] {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

// ValidateCombatLogAction restricts authored status log identifiers to the
// stable uppercase token format used by CombatLog.Action.
func ValidateCombatLogAction(action string) error {
	if action == "" || len(action) > 64 || action[0] < 'A' || action[0] > 'Z' {
		return fmt.Errorf("log_action must be an uppercase identifier")
	}
	for _, r := range action[1:] {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return fmt.Errorf("log_action must be an uppercase identifier")
		}
	}
	return nil
}

// ValidatePeriodicResource keeps the resource/subtype matrix closed so a
// catalog-valid operation cannot compile into an executor no-op.
func ValidatePeriodicResource(subType, resource string) error {
	switch subType {
	case "REGEN_PER_HOUR":
		if resource == "MANA" || resource == "THOUGHT" || resource == "SOUL" {
			return nil
		}
	case "GENERATION_WORKER_TICK":
		if resource == "PRIMEVAL_STONE" || resource == "IMMORTAL_STONE" {
			return nil
		}
	}
	return fmt.Errorf("resource %q is not supported by periodic subtype %q", resource, subType)
}

// ValidateDisplacement keeps authored movement values integral and SET_ROW
// within the combat grid. Zero remains the default-step sentinel for movement.
func ValidateDisplacement(subType string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || math.Trunc(value) != value {
		return fmt.Errorf("DISPLACE value must be a non-negative integer")
	}
	if subType == "SET_ROW" && (value < 1 || value > 5) {
		return fmt.Errorf("DISPLACE SET_ROW value must be between 1 and 5")
	}
	return nil
}

// ValidateResourceTransfer closes the transfer matrix so an accepted typed
// command cannot compile into an unsupported resource operation.
func ValidateResourceTransfer(subType, resource string, fraction, minAmount float64, phase, logAction, message string) error {
	if math.IsNaN(fraction) || math.IsInf(fraction, 0) || fraction <= 0 || fraction > 1 ||
		math.IsNaN(minAmount) || math.IsInf(minAmount, 0) || minAmount < 0 || minAmount > 1e12 {
		return fmt.Errorf("resource transfer value must be in (0,1] and min_amount must be non-negative")
	}
	switch {
	case subType == "PERCENT_OF_TARGET" && resource == "MANA":
		if minAmount != 0 || phase != "" || logAction != "" {
			return fmt.Errorf("MANA PERCENT_OF_TARGET does not support min_amount, phase or log_action")
		}
		if message != "" {
			return ValidateNotifyMessage(message)
		}
		return nil
	case subType == "PERCENT_OF_MAX_HP" && resource == "HP":
		if phase != "LATE_RIDER" {
			return fmt.Errorf("HP PERCENT_OF_MAX_HP requires phase LATE_RIDER")
		}
		if err := ValidateCombatLogAction(logAction); err != nil {
			return err
		}
		return ValidateNotifyMessage(message)
	case subType == "PERCENT_OF_TARGET_SHIELD" && resource == "HP":
		if minAmount != 0 || phase != "EARLY_RIDER" {
			return fmt.Errorf("HP PERCENT_OF_TARGET_SHIELD requires no min_amount and phase EARLY_RIDER")
		}
		if err := ValidateCombatLogAction(logAction); err != nil {
			return err
		}
		return ValidateNotifyMessage(message)
	default:
		return fmt.Errorf("unsupported resource transfer %q/%q", resource, subType)
	}
}

var StatCatalog = values("ATTACK", "DEFENSE", "HP", "SPEED", "DAMAGE_DEALT", "TENACITY", "CC_RESIST", "LUCK", "CRIT_RATE", "ALL", "MANA", "MOVEMENT", "INVESTIGATION", "HEARING", "DISGUISE", "TIME_SPEED", "REFINE_SUCCESS", "REFINE_MANA_TIER", "STORAGE", "SOUL_REFINEMENT", "CLEANSING_POWER", "DAO_MARK", "DAO_MARK_GAIN", "CULTIVATION_EFFICIENCY", "HP_REGEN", "HEALING_RECEIVED", "MAX_MANA", "MANA_REGEN", "SOUL_REGEN", "SOUL_DEFENSE", "MENTAL_RESISTANCE", "APERTURE_STRESS", "PATH_CONFLICT_REDUCTION", "PATH_DAMAGE", "PATH_RESISTANCE")
var ResourceCatalog = values("HP", "MANA", "SOUL", "THOUGHT", "APTITUDE", "LIFESPAN", "STRESS", "PRIMEVAL_STONE", "IMMORTAL_STONE", "DAO_MARK", "HUMAN_QI", "EARTH_QI", "HEAVEN_QI")
var StatusCatalog = func() map[string]bool {
	m := map[string]bool{}
	for id := range statusfx.Registry {
		m[id] = true
	}
	return m
}()
var UnitCatalog = values("FLAT", "PERCENT", "RATIO", "POINTS")
var StackPolicyCatalog = values("REPLACE", "REFRESH", "STACK", "MAX", "MIN", "IGNORE")
var ConditionValueTypes = map[string]string{
	"HP_PERCENT":        "number",
	"ACTOR_HP_PERCENT":  "number",
	"MANA_PERCENT":      "number",
	"TURN":              "number",
	"TARGET_HP_PERCENT": "number",
	"STACKS":            "number",
	"RANK":              "number",
	"MASTERY":           "number",
	"ACTOR_RACE":        "string",
	"TARGET_RACE":       "string",
}
var ConditionFields = func() map[string]bool {
	fields := make(map[string]bool, len(ConditionValueTypes))
	for field := range ConditionValueTypes {
		fields[field] = true
	}
	return fields
}()
var ConditionOperators = values("EQ", "NE", "LT", "LTE", "GT", "GTE")

// NormalizeConditionValue validates a condition value against its registered
// field type and returns the runtime representation used by ConditionsMatch.
func NormalizeConditionValue(field, operator string, value any) (any, error) {
	field = strings.ToUpper(strings.TrimSpace(field))
	operator = strings.ToUpper(strings.TrimSpace(operator))
	if !ConditionFields[field] {
		return nil, fmt.Errorf("unknown condition field %q", field)
	}
	if !ConditionOperators[operator] {
		return nil, fmt.Errorf("unknown condition operator %q", operator)
	}
	if ConditionValueTypes[field] == "string" {
		if operator != "EQ" && operator != "NE" {
			return nil, fmt.Errorf("condition field %q only supports EQ and NE", field)
		}
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("condition field %q requires a non-empty string value", field)
		}
		return strings.TrimSpace(text), nil
	}

	number, ok := conditionNumber(value)
	if !ok {
		return nil, fmt.Errorf("condition field %q requires a numeric value", field)
	}
	return number, nil
}

func conditionNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, !math.IsNaN(number) && !math.IsInf(number, 0)
	case float32:
		return float64(number), !math.IsNaN(float64(number)) && !math.IsInf(float64(number), 0)
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case int32:
		return float64(number), true
	case int16:
		return float64(number), true
	case int8:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint64:
		return float64(number), true
	case uint32:
		return float64(number), true
	case uint16:
		return float64(number), true
	case uint8:
		return float64(number), true
	default:
		return 0, false
	}
}

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
	if code == "LUCK_ROULETTE" {
		return ValidateLuckRouletteParams(raw)
	}
	return nil
}

// ValidateSummonParams applies the SUMMON_MONSTER parameter schema to the
// typed SUMMON operation during the migration period.
func ValidateSummonParams(raw json.RawMessage) error {
	if err := ValidateNativeParams("SUMMON_MONSTER", raw); err != nil {
		return err
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(raw, &params); err != nil {
		return fmt.Errorf("summon params must be an object: %w", err)
	}
	var message string
	_ = json.Unmarshal(params["log_message"], &message)
	message = strings.TrimSpace(message)
	if _, exists := params["log_message"]; exists && message == "" {
		return fmt.Errorf("log_message must contain text")
	}
	var name string
	_ = json.Unmarshal(params["name"], &name)
	if _, exists := params["name"]; exists && (strings.TrimSpace(name) == "" || len([]rune(name)) > 100) {
		return fmt.Errorf("name must contain 1 to 100 characters")
	}
	var suffix string
	_ = json.Unmarshal(params["id_suffix"], &suffix)
	if _, exists := params["id_suffix"]; exists {
		if suffix == "" || len(suffix) > 48 {
			return fmt.Errorf("id_suffix must contain 1 to 48 lowercase letters, digits, or underscores")
		}
		for _, r := range suffix {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
				return fmt.Errorf("id_suffix must contain 1 to 48 lowercase letters, digits, or underscores")
			}
		}
	}
	if rawRank, exists := params["rank"]; exists {
		var rank int
		if json.Unmarshal(rawRank, &rank) != nil || rank < 0 || rank > 100 {
			return fmt.Errorf("rank must be an integer between 0 and 100")
		}
	}
	if message == "" {
		return nil
	}
	if err := ValidateNotifyMessage(message); err != nil {
		return fmt.Errorf("log_message: %w", err)
	}
	allowed := map[string]bool{"actor": true, "summon": true, "hp": true, "damage": true, "dao_note": true}
	for rest := message; ; {
		start := strings.IndexByte(rest, '{')
		if start < 0 {
			if strings.Contains(rest, "}") {
				return fmt.Errorf("log_message has an unmatched closing brace")
			}
			break
		}
		if strings.Contains(rest[:start], "}") {
			return fmt.Errorf("log_message has an unmatched closing brace")
		}
		end := strings.IndexByte(rest[start+1:], '}')
		if end < 0 {
			return fmt.Errorf("log_message has an unmatched opening brace")
		}
		key := rest[start+1 : start+1+end]
		if !allowed[key] {
			return fmt.Errorf("log_message placeholder %q is unsupported", key)
		}
		rest = rest[start+2+end:]
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
	case "number_map":
		var values map[string]json.RawMessage
		if json.Unmarshal(raw, &values) != nil || len(values) == 0 {
			return false
		}
		for key, value := range values {
			if strings.TrimSpace(key) == "" {
				return false
			}
			var number float64
			if json.Unmarshal(value, &number) != nil {
				return false
			}
		}
		return true
	case "any":
		return len(raw) > 0 && string(raw) != "null"
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
	"REBORN":                  {values("ON_DEATH"), map[string]ParamSpec{"chance": number(true, 0, 1), "mode": {Type: "string", Enum: values("rewind", "in_place")}, "max_uses": number(false, 0, 10), "turn1_guaranteed": {Type: "bool"}, "revive_hp_percent": number(false, 0, 1), "clear_debuffs": {Type: "bool"}, "empower_attack_percent": number(false, 0, 5), "empower_speed_percent": number(false, 0, 5), "empower_shield_percent": number(false, 0, 5), "empower_turns": number(false, 1, 100)}, "combat"},
	"LUCK_ROULETTE":           {values("ON_CAST", "ON_ACTION"), map[string]ParamSpec{"great_chance_percent": number(true, 0, 100), "curse_great_chance_bonus": number(true, 0, 100), "medium_range_percent": number(true, 0, 100), "neutral_range_percent": number(true, 0, 100), "great_damage_multiplier": number(true, 0, 10), "medium_damage_multiplier": number(true, 0, 10), "neutral_damage_multiplier": number(true, 0, 10), "bad_damage_multiplier": number(true, 0, 10), "great_mana_cost_multiplier": number(true, 0, 10), "bad_current_hp_loss_percent": number(true, 0, 10), "minimum_bad_hp_loss": number(true, 0, 1e12), "curse_duration_turns": number(true, 1, 100)}, "combat"},
	"NIRVANA_FIRE":            {values("ON_DEATH", "COMBAT_PASSIVE", "BEFORE_HIT"), map[string]ParamSpec{"revive_hp_percent": number(false, 0, 1), "minimum_revive_hp": number(false, 0, 1e12), "clear_debuffs": {Type: "bool"}, "retaliation_damage_percent": number(false, 0, 10), "minimum_retaliation_damage": number(false, 0, 1e12)}, "combat"},
	"DREAM_UNRAVEL":           {values("ON_CAST", "ON_ACTION"), map[string]ParamSpec{"base_conversion_units": number(false, 0, 100), "mana_percent_per_unit": number(false, 0, 10), "shield_percent_per_unit": number(false, 0, 10), "clear_control_states": {Type: "bool"}}, "combat"},
	"FORGET_INTENT_INTERRUPT": {values("BEFORE_ENEMY_ATTACK"), map[string]ParamSpec{"chance": number(true, 0, 1), "disable_target_gu_turns": number(false, 0, 10), "max_successful_activations": number(false, 1, 100), "backfire_after_max": {Type: "bool"}}, "combat"},
	"FORGET_WILL_CONFUSE":     {values("BEFORE_ENEMY_ATTACK"), map[string]ParamSpec{"activation_chance": number(false, 0, 1), "hit_user_chance": number(false, 0, 1), "hit_self_chance": number(false, 0, 1), "miss_chance": number(false, 0, 1), "backfire_chance": number(false, 0, 1), "backfire_disable_turns": number(false, 0, 10)}, "combat"},
}

func init() {
	register := func(trigger, domain string, codes ...string) {
		for _, code := range codes {
			if hook, exists := NativeHooks[code]; !exists {
				NativeHooks[code] = NativeHook{Triggers: values(trigger), Params: map[string]ParamSpec{}, Domain: domain}
			} else {
				hook.Triggers[trigger] = true
			}
		}
	}
	register("ON_CONSUME", "item_use", "SOVEREIGN_BODY_TRANSFORM", "SACRIFICE_APTITUDE", "RANK_PROGRESS", "ADD_SOUL_MAX", "ADD_LIFESPAN_YEARS", "RESTORE_SOUL_HP", "RESTORE_MENTAL", "INCREASE_TIME_SPEED", "REDUCE_STRESS", "ADD_THOUGHT_MAX", "DECREASE_TIME_SPEED", "MAN_BEAST_BURIAL", "POISON_VOW", "SPRING_EGG_HATCH", "STONE_APERTURE", "THIRD_WATCH", "RESTORE_HUNGER", "RESTORE_HP", "INCREASE_HUMAN_QI", "RESTORE_EARTH_QI", "RESTORE_HEAVEN_QI", "HEAL_APERTURE", "INCREASE_APERTURE_AREA", "INCREASE_APERTURE_TIME_RATE", "RANDOM_CONSTITUTION", "RANDOM_RACE", "OPEN_MINE_CHEST", "OPEN_IMMORTAL_MINE_CHEST", "ADD_DAO_MARKS", "IMMORTAL_ONLY", "MORTAL_ONLY", "MIN_RANK", "MAX_RANK", "RESTORE_MANA", "SOUL_BOOST", "REMOVE_NEGATIVE_STATUS", "LIFESPAN_BOOST", "CHANGE_RACE", "EXIT_SPATIAL_REALM")
	register("ALWAYS_ON", "gamefx", "STAT_MODS", "REFINE_SUCCESS", "DAO_MARK_MODIFIER", "PATH_DAMAGE_MODIFIER", "PATH_RESISTANCE_MODIFIER", "RACE_DAMAGE_MODIFIER", "RACE_RESISTANCE_MODIFIER", "LUCK", "MAX_MANA", "MANA_REGEN", "CULTIVATION_EFFICIENCY", "DAO_MARK_GAIN", "HEALING_RECEIVED", "HP_REGEN", "MENTAL_RESISTANCE", "SOUL_DEFENSE", "SOUL_REGEN", "TENACITY", "CC_RESIST", "APERTURE_STRESS", "NON_CONFLICTING_DAO_MARKS", "IMMORTAL", "DISABLE_HP_REGEN", "DISABLE_MANA_REGEN", "DISABLE_RESOURCE_REGEN", "DISABLE_SOUL_REGEN")
	register("ON_TICK", "growth", "MANA_REGEN_TICK", "THOUGHT_REGEN_TICK", "SOUL_REGEN_TICK", "GENERATE_STONES_TICK", "GENERATE_IMMORTAL_STONES_TICK")
	register("ALWAYS_ON", "growth", "GROWTH_DURATION_HOURS")
	register("ON_USE", "world", "RECORD_MESSAGE")
	register("ON_DEATH", "combat", "NIRVANA_FIRE")
	register("ON_TURN_START", "combat", "POEM_INTENT", "DIVINE_BEAN_ABUNDANCE", "SIX_EXTREMITIES_ARRAY", "MYRIAD_YEAR_WARSHIP", "YAMA_EMPEROR", "FAREWELL_CURSE", "SWORD_MARK", "DRY_WOOD_SPRING")
	register("ON_BATTLE_START", "combat", "SEIZE_INITIATIVE", "SAT_KHI_GUARANTEE", "PERM_STEAL_GU", "TEMP_STEAL_GU", "SUPPRESS_PRIORITY", "SUMMON_WOLF_FRONTLINE", "SUMMON_MONSTER", "INSTANT_WIN", "LETHAL_LUCK_WARD", "MYRIAD_YEAR_WARSHIP", "EIGHT_EXTREMITIES_LUCK", "FIVE_REGIONAL_BOUNDARY", "SOUL_CONCEALMENT_ARRAY", "FAMILIAR_FACE", "NINETY_NINE_ARRAY", "DRAGON_PALACE_DOMINION", "HEAVEN_NET_RESTRICTION", "ALL_PEOPLE_UNITED", "GREAT_NIGHT", "TOTEM_GOLDEN_BELL")
	register("ON_CAST", "combat", "SUMMON_WOLF_FRONTLINE", "FULL_SOUL_HEAL", "SWORD_DRAGON_TRANSFORM", "BORROW_PAST_POWER", "UNRAVEL_RIDDLE", "CHAOS_GAMBLE", "SUMMON_MONSTER", "LUCK_ROULETTE", "DREAM_UNRAVEL", "BLOOD_SACRIFICE", "CALCULATED_STRIKE", "TIME_REWIND", "FORM_BEAR", "FORM_EAGLE", "BONE_THORNS", "METAL_AEGIS", "WOOD_REGEN_BURST", "WATER_CLEANSE", "WATER_MIRROR_SHIELD", "EARTH_SHIELD", "WIND_HASTE_BURST", "LIGHT_PURIFY", "DARK_STEALTH", "STAR_THOUGHT_RESTORE", "CLOUD_MIST", "TIEP_THIEN_DIA", "VAN_DAN_DONG_TAM", "QI_WALL", "SPATIAL_SHIFT", "FORMATION_DOMAIN", "REVERSE_FLOW_SEAL", "FAMILIAR_FACE", "MYRIAD_SELF_GIANT_HAND", "NINETY_NINE_ARRAY", "SOUL_CONCEALMENT_ARRAY", "FIVE_REGIONAL_BOUNDARY", "INFINITE_WIND", "THREE_QI_RETURN", "FUTURE_SELF", "LANDSCAPE_AS_BEFORE", "YAMA_EMPEROR", "NO_EXTREME_SPACE", "REFINE_SELF", "SOUL_BEAST_SUMMON", "BLOOD_GOD_CHILD", "THIRTY_THREE_HEAVEN_LIGHT", "DRY_WOOD_SPRING", "IMMORTAL_BINDING_VINE", "RETREAT_SELF")
	register("ON_ACTION", "combat", "LUCK_ROULETTE", "DREAM_UNRAVEL", "CALCULATED_STRIKE", "BLOOD_SACRIFICE", "TIME_REWIND", "SOUL_SHOCK", "SOUL_DEVOUR", "BEAST_PHANTOM_STRIKE", "HEAVY_LIFESTEAL", "BLEED_BURST", "MANA_EAT", "MANA_STEAL", "BUFF_STEAL", "PACK_FRENZY", "POISON_DETONATE", "SWORD_ARMOR_PIERCE", "EXECUTE_STRIKE", "BONE_SPIKE", "FORM_BEAR", "FORM_EAGLE", "BONE_THORNS", "METAL_AEGIS", "ARMOR_BREAK_HEAVY", "WOOD_REGEN_BURST", "WOOD_BIND", "WATER_CLEANSE", "WATER_MIRROR_SHIELD", "FIRE_INFERNO", "IGNITE_BURST", "EARTH_SHIELD", "EARTH_GRAVITY", "WIND_HASTE_BURST", "LIGHTNING_STUN_BURST", "CHAIN_LIGHTNING", "ICE_DEEP_FREEZE", "ICE_FROSTBITE", "LIGHT_BLINDING_FLASH", "LIGHT_PURIFY", "DARK_STEALTH", "DARK_CURSE_WEAK", "STAR_FIRE_RAIN", "STAR_THOUGHT_RESTORE", "SOUND_DISRUPT_SILENCE", "SOUND_RESONANCE", "CLOUD_MIST", "HEAVEN_SUPPRESSION", "TIEP_THIEN_DIA", "HUMAN_WILL", "VAN_DAN_DONG_TAM", "QI_WALL", "QI_EXPLOSION", "SPATIAL_SHIFT", "DIMENSION_SLASH", "RULE_IMPOSITION", "JUSTICE_VERDICT", "FORMATION_DOMAIN", "BLINDING_MIRAGE", "PAINTING_IMPRISONMENT", "PASSION_OVERLOAD", "GU_DETONATION", "REVERSE_FLOW_SEAL", "GHOST_HAND_STEAL", "LEAD_SOUL_TO_DREAM", "MYRIAD_SELF_GIANT_HAND", "FAMILIAR_FACE", "FIVE_FINGER_SWORD", "TIME_CUTTER", "SPRING_SCISSORS_SUMMER_FAN", "DRAGON_BREATH_SWORD", "THUNDER_GOD_CITY", "FIVE_REGIONAL_BOUNDARY", "SONG_OF_SEPARATION", "SONG_OF_DESTINY", "FAREWELL_FRIEND_WIND", "INFINITE_WIND", "BLOOD_DEMON_DISINTEGRATION", "YAMA_EMPEROR", "IMMEASURABLE_QI_SEA", "THREE_QI_RETURN", "THREE_MEN_MAKE_TIGER", "WORDS_ARE_FRIGHTENING", "FATE_VANQUISHED", "HEAVEN_NET_RESTRICTION", "FUTURE_SELF", "LANDSCAPE_AS_BEFORE", "MYRIAD_DISMANTLE", "YOU_DU_STRENGTH_TIGER", "SWORD_WAVE_TRIPLE", "SWORD_MARK_DEATH", "SOUL_EXPLOSION_DANG_HUN", "SOUL_BEAST_SUMMON", "STEAL_LIFE", "CHAOS_CAVE_DRILL", "NO_EXTREME_SPACE", "LIGHT_TIME_FLYING_BLADE", "ALL_PEOPLE_UNITED", "HERO_AMONG_MEN", "QI_JUE_RETURN_LIFE", "RAGING_ANGRY_BIRD", "EXTREME_ICE_CRYSTAL_COFFIN", "THIRTY_THREE_HEAVEN_LIGHT", "DRY_WOOD_SPRING", "IMMORTAL_BINDING_VINE", "BLOOD_GOD_CHILD", "TOTEM_KUN_PENG", "TOTEM_GOLDEN_BELL", "REFINE_SELF", "MYRIAD_STAR_FIREFLY", "FIVE_RESTRICTION_DEEP_LIGHT", "CHARM_GHOST_COME", "DEVOUR_HEAVEN_EARTH", "KNOCKBACK", "PULL_TARGET", "SWAP_TARGET", "RETREAT_SELF")
	register("ON_HIT", "combat", "BYPASS_SHIELD", "BEAST_PHANTOM_STRIKE", "SOUL_SHOCK", "MANA_STEAL", "SWORD_ARMOR_PIERCE", "EXECUTE_STRIKE", "BONE_SPIKE", "ARMOR_BREAK_HEAVY", "WOOD_BIND", "FIRE_INFERNO", "LIGHTNING_STUN_BURST", "CHAIN_LIGHTNING", "ICE_DEEP_FREEZE", "ICE_FROSTBITE", "LIGHT_BLINDING_FLASH", "DARK_CURSE_WEAK", "SOUND_DISRUPT_SILENCE", "SOUND_RESONANCE", "DIMENSION_SLASH", "JUSTICE_VERDICT", "BLINDING_MIRAGE", "PAINTING_IMPRISONMENT", "FIVE_FINGER_SWORD", "TIME_CUTTER", "SPRING_SCISSORS_SUMMER_FAN", "DRAGON_BREATH_SWORD", "FAREWELL_FRIEND_WIND", "WORDS_ARE_FRIGHTENING", "THREE_MEN_MAKE_TIGER", "YOU_DU_STRENGTH_TIGER", "SWORD_WAVE_TRIPLE", "SWORD_MARK_DEATH", "CHAOS_CAVE_DRILL", "RAGING_ANGRY_BIRD", "EXTREME_ICE_CRYSTAL_COFFIN", "IMMORTAL_BINDING_VINE", "FIVE_RESTRICTION_DEEP_LIGHT", "CHARM_GHOST_COME", "KNOCKBACK", "PULL_TARGET", "SWAP_TARGET")
	register("COMBAT_PASSIVE", "combat", "DOUBLE_DAMAGE_DEALT", "REVERSE_CALAMITY", "LETHAL_LUCK_WARD", "BERSERK_STATE", "PACK_FRENZY", "DAO_ADAPTIVE_RESIST", "MANA_EAT", "BONE_THORNS", "WATER_MIRROR_SHIELD", "HUMAN_WILL", "PASSION_OVERLOAD", "REVERSE_FLOW_SEAL", "FAMILIAR_FACE", "NINETY_NINE_ARRAY", "SONG_OF_DESTINY", "BLOOD_STAINED_ROBE", "YAMA_EMPEROR", "EATING_LOSS_BLESSING", "NIRVANA_FIRE", "TOTEM_GOLDEN_BELL", "QI_JUE_RETURN_LIFE", "ALL_PEOPLE_UNITED", "GREAT_NIGHT")
	register("ON_ACTION", "combat", "DAO_ADAPTIVE_RESIST")
	register("BEFORE_HIT", "combat", "DAO_ADAPTIVE_RESIST", "REVERSE_FLOW_SEAL", "DRAGON_PALACE_DOMINION", "NINETY_NINE_ARRAY", "BLOOD_STAINED_ROBE", "EATING_LOSS_BLESSING", "TOTEM_GOLDEN_BELL", "QI_JUE_RETURN_LIFE")
	NativeHooks["GUARANTEED_HIT"] = NativeHook{
		Triggers: values("ON_ACTION", "ON_HIT", "COMBAT_PASSIVE"),
		Params:   map[string]ParamSpec{},
		Domain:   "combat",
	}
	NativeHooks["PERSIST_STATUS_ON_HIT"] = NativeHook{
		Triggers: values("ON_HIT", "ON_ACTION"),
		Params: map[string]ParamSpec{
			"status_key": {Type: "string", Required: true},
			"pvp_only":   {Type: "bool", Required: false},
		},
		Domain: "combat",
	}
	NativeHooks["CLEANSE_REGIONAL_WALL"] = NativeHook{
		Triggers: values("ON_BATTLE_START"),
		Params:   map[string]ParamSpec{},
		Domain:   "combat",
	}
	NativeHooks["FAKE_REGIONAL_WALL"] = NativeHook{
		Triggers: values("ON_BATTLE_START"),
		Params:   map[string]ParamSpec{},
		Domain:   "combat",
	}
	register("BEFORE_HIT", "combat", "FIRST_TURN_EVADE", "LETHAL_LUCK_WARD")
	register("COMBAT_PASSIVE", "combat", "DOUBLE_DAMAGE_DEALT", "REVERSE_CALAMITY", "LETHAL_LUCK_WARD")
	register("CAPABILITY", "capability", "MARKETPLACE_ACCESS", "REMOTE_TRADE", "CHAT_CONNECT", "CHAT_PERCEIVE", "IGNORE_MOVEMENT_RANGE", "IGNORE_MONSTER_ENCOUNTER", "LOOT_PREVIEW", "DISGUISE", "CHANGE_FORM_DISGUISE", "PERFECT_DISGUISE", "CONCEAL_GU", "SEVER_CLUES", "PORTAL_CREATE", "RANDOM_PORTAL", "CAPTURE_BONUS")
	register("ALWAYS_ON", "derived_player", "IMMUNE_DREAM_STATUS")
	NativeHooks["DEDUCTION_INSTANT_FINISH"] = NativeHook{
		Triggers: values("ALWAYS_ON"),
		Params: map[string]ParamSpec{
			"base_chance":   number(false, 0, 1),
			"per_rank_step": number(false, 0, 1),
			"min_chance":    number(false, 0, 1),
			"max_chance":    number(false, 0, 1),
		},
		Domain: "deduction",
	}
	// Schemas below describe parameters present in the authoritative CSV data.
	// Hooks without parameters retain the closed EMPTY schema.
	valueSchema := map[string]ParamSpec{"value": {Type: "number", Required: true}}
	for _, code := range []string{"ADD_LIFESPAN_YEARS", "ADD_SOUL_MAX", "ADD_THOUGHT_MAX", "DECREASE_TIME_SPEED", "GENERATE_IMMORTAL_STONES_TICK", "GENERATE_STONES_TICK", "GROWTH_DURATION_HOURS", "INCREASE_TIME_SPEED", "MANA_REGEN_TICK", "REDUCE_STRESS", "RESTORE_MENTAL", "RESTORE_SOUL_HP", "SOUL_REGEN_TICK", "THOUGHT_REGEN_TICK", "ADD_DAO_MARKS", "IMMORTAL_ONLY"} {
		hook := NativeHooks[code]
		hook.Params = valueSchema
		NativeHooks[code] = hook
	}
	// Regen ticks keep an optional value: handlers default missing/zero to the
	// historical rates (mana 0.5, thought/soul 1.0), matching the old
	// import-time bridge. Authored blocks may omit it.
	optionalValue := map[string]ParamSpec{"value": {Type: "number", Required: false}}
	for _, code := range []string{"MANA_REGEN_TICK", "THOUGHT_REGEN_TICK", "SOUL_REGEN_TICK"} {
		hook := NativeHooks[code]
		hook.Params = optionalValue
		NativeHooks[code] = hook
	}
	hook := NativeHooks["CAPTURE_BONUS"]
	hook.Params = valueSchema
	NativeHooks["CAPTURE_BONUS"] = hook
	stepsSchema := map[string]ParamSpec{"steps": {Type: "number", Required: false}}
	for _, code := range []string{"KNOCKBACK", "PULL_TARGET", "RETREAT_SELF"} {
		h := NativeHooks[code]
		h.Params = stepsSchema
		NativeHooks[code] = h
	}
	// Portal abilities may declare a maximum teleport range in world cells.
	// Missing/zero range falls back to the rank table in handlers.GuTeleportRange.
	for _, code := range []string{"PORTAL_CREATE", "RANDOM_PORTAL"} {
		portalHook := NativeHooks[code]
		portalHook.Params = map[string]ParamSpec{"range": {Type: "integer", Required: false}}
		NativeHooks[code] = portalHook
	}
	// SUMMON_MONSTER: authored params stay minimal (monster id + author scale);
	// the combat builder injects the template stats (hp/damage/defense/name)
	// resolved from MonsterRegistry into the compiled values.
	hook = NativeHooks["SUMMON_MONSTER"]
	hook.Params = map[string]ParamSpec{
		"monster_id":    {Type: "string", Required: true},
		"hp_scale":      number(false, 0.05, 2),
		"damage_scale":  number(false, 0.05, 2),
		"defense_scale": number(false, 0, 10),
		"stat_source":   {Type: "string", Enum: values("TEMPLATE", "ACTOR")},
		"rank_scale":    {Type: "bool"},
		"rank":          {Type: "integer"},
		"speed":         number(false, 0, 1e6),
		"name":          {Type: "string"},
		"id_suffix":     {Type: "string"},
		"log_message":   {Type: "string"},
		"position":      {Type: "string", Enum: values("frontline", "mid", "back")},
	}
	NativeHooks["SUMMON_MONSTER"] = hook
	// Summoning Gu now fire from ON_BATTLE_START (see Gu.csv); ON_CAST stays
	// accepted so older authored rows keep resolving through the same hook.
	for _, code := range []string{"SUMMON_WOLF_FRONTLINE", "SUMMON_MONSTER"} {
		hook := NativeHooks[code]
		hook.Triggers = values("ON_CAST", "ON_BATTLE_START")
		NativeHooks[code] = hook
	}
	hook = NativeHooks["SEIZE_INITIATIVE"]
	hook.Params = map[string]ParamSpec{"lifespan_ratio": {Type: "number", Required: true}}
	NativeHooks["SEIZE_INITIATIVE"] = hook
	hook = NativeHooks["SAT_KHI_GUARANTEE"]
	hook.Params = map[string]ParamSpec{"sat_khi_threshold": {Type: "number", Required: true}}
	NativeHooks["SAT_KHI_GUARANTEE"] = hook
	hook = NativeHooks["BERSERK_STATE"]
	hook.Params = map[string]ParamSpec{"damage_modifier_in_primitive": {Type: "bool", Required: false}}
	NativeHooks["BERSERK_STATE"] = hook
	hook = NativeHooks["EATING_LOSS_BLESSING"]
	hook.Params = map[string]ParamSpec{
		"counter_add_in_primitive":     {Type: "bool", Required: false},
		"damage_modifier_in_primitive": {Type: "bool", Required: false},
		"heal_in_primitive":            {Type: "bool", Required: false},
		"log_in_primitive":             {Type: "bool", Required: false},
	}
	NativeHooks["EATING_LOSS_BLESSING"] = hook
	hook = NativeHooks["RECORD_MESSAGE"]
	hook.Params = map[string]ParamSpec{"range": {Type: "integer", Required: false}}
	NativeHooks["RECORD_MESSAGE"] = hook
	hook = NativeHooks["PERM_STEAL_GU"]
	hook.Params = map[string]ParamSpec{
		"base_chance":      number(false, 0, 1),
		"chance_delta":     number(false, 0, 1),
		"vital_penalty":    number(false, 0, 1),
		"target_selection": {Type: "string"},
	}
	NativeHooks["PERM_STEAL_GU"] = hook
	hook = NativeHooks["TEMP_STEAL_GU"]
	hook.Params = map[string]ParamSpec{
		"base_chance":      number(false, 0, 1),
		"chance_delta":     number(false, 0, 1),
		"vital_penalty":    number(false, 0, 1),
		"target_selection": {Type: "string"},
		"duration":         {Type: "string"},
	}
	NativeHooks["TEMP_STEAL_GU"] = hook

	anyValueSchema := map[string]ParamSpec{"value": {Type: "any", Required: false}}
	for _, code := range []string{
		"RESTORE_HUNGER", "RESTORE_HP", "INCREASE_HUMAN_QI", "RESTORE_EARTH_QI", "RESTORE_HEAVEN_QI", "HEAL_APERTURE", "INCREASE_APERTURE_AREA", "INCREASE_APERTURE_TIME_RATE", "RANDOM_CONSTITUTION", "RANDOM_RACE", "OPEN_MINE_CHEST", "OPEN_IMMORTAL_MINE_CHEST",
		"MORTAL_ONLY", "MIN_RANK", "MAX_RANK", "RESTORE_MANA", "SOUL_BOOST", "REMOVE_NEGATIVE_STATUS", "LIFESPAN_BOOST", "CHANGE_RACE", "EXIT_SPATIAL_REALM", "ADD_DAO_MARKS", "SPRING_EGG_HATCH",
		"STAT_MODS", "REFINE_SUCCESS", "DAO_MARK_MODIFIER", "PATH_DAMAGE_MODIFIER", "PATH_RESISTANCE_MODIFIER", "LUCK",
		"MAX_MANA", "MANA_REGEN", "CULTIVATION_EFFICIENCY", "DAO_MARK_GAIN", "HEALING_RECEIVED", "HP_REGEN",
		"MENTAL_RESISTANCE", "SOUL_DEFENSE", "SOUL_REGEN", "TENACITY", "CC_RESIST", "APERTURE_STRESS",
		"NON_CONFLICTING_DAO_MARKS", "IMMORTAL", "DISABLE_HP_REGEN", "DISABLE_MANA_REGEN", "DISABLE_RESOURCE_REGEN", "DISABLE_SOUL_REGEN",
	} {
		hook := NativeHooks[code]
		hook.Params = anyValueSchema
		NativeHooks[code] = hook
	}
	numberMapSchema := map[string]ParamSpec{"value": {Type: "number_map", Required: true}}
	for _, code := range []string{"STAT_MODS", "DAO_MARK_MODIFIER", "PATH_DAMAGE_MODIFIER", "PATH_RESISTANCE_MODIFIER", "RACE_DAMAGE_MODIFIER", "RACE_RESISTANCE_MODIFIER"} {
		hook := NativeHooks[code]
		hook.Params = numberMapSchema
		NativeHooks[code] = hook
	}
}

type Metadata struct {
	SchemaVersion       int                  `json:"schema_version"`
	Triggers            []string             `json:"triggers"`
	Targets             []string             `json:"targets"`
	Operations          []OperationMetadata  `json:"operations"`
	Stats               []string             `json:"stats"`
	Statuses            []string             `json:"statuses"`
	Resources           []string             `json:"resources"`
	Units               []string             `json:"units"`
	StackPolicies       []string             `json:"stack_policies"`
	ConditionFields     []string             `json:"condition_fields"`
	ConditionValueTypes map[string]string    `json:"condition_value_types"`
	ConditionOperators  []string             `json:"condition_operators"`
	NativeCodes         []string             `json:"native_codes"`
	NativeHooks         []NativeHookMetadata `json:"native_hooks"`
}
type OperationMetadata struct {
	Name           string   `json:"name"`
	SubTypes       []string `json:"sub_types"`
	Triggers       []string `json:"triggers"`
	Targets        []string `json:"targets"`
	AllowedFields  []string `json:"allowed_fields"`
	RequiredFields []string `json:"required_fields"`
	CostGroup      string   `json:"cost_group"`
	MinRankGate    int      `json:"min_rank_gate"`
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
		p := PowerCostCatalog[name]
		ops = append(ops, OperationMetadata{name, sortedKeys(d.SubTypes), sortedKeys(d.Triggers), sortedKeys(d.Targets), sortedKeys(d.Allowed), append([]string(nil), d.Required...), string(p.Group), p.MinRankGate})
	}
	hooks := make([]NativeHookMetadata, 0, len(NativeHooks))
	for _, code := range sortedKeys(NativeHooks) {
		h := NativeHooks[code]
		hooks = append(hooks, NativeHookMetadata{code, h.Domain, sortedKeys(h.Triggers), h.Params})
	}
	return Metadata{
		SchemaVersion:       BlockSchemaVersion,
		Triggers:            sortedKeys(TriggerCatalog),
		Targets:             sortedKeys(TargetCatalog),
		Operations:          ops,
		Stats:               sortedKeys(StatCatalog),
		Statuses:            sortedKeys(StatusCatalog),
		Resources:           sortedKeys(ResourceCatalog),
		Units:               sortedKeys(UnitCatalog),
		StackPolicies:       sortedKeys(StackPolicyCatalog),
		ConditionFields:     sortedKeys(ConditionFields),
		ConditionValueTypes: ConditionValueTypes,
		ConditionOperators:  sortedKeys(ConditionOperators),
		NativeCodes:         sortedKeys(NativeHooks),
		NativeHooks:         hooks,
	}
}
