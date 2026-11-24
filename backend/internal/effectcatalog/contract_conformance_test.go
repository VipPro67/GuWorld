package effectcatalog_test

import (
	"encoding/json"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func assertAccepted(t *testing.T, block effects.Block) {
	t.Helper()
	envelope := struct {
		SchemaVersion int             `json:"schema_version"`
		Blocks        []effects.Block `json:"blocks"`
	}{2, []effects.Block{block}}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if issues := effectcatalog.ValidateBlockV2(raw, effectcatalog.Location{}); len(issues) != 0 {
		t.Fatalf("validator rejected %s: %+v", raw, issues)
	}
	if _, err := (effects.Compiler{}).Compile(envelope.Blocks); err != nil {
		t.Fatalf("compiler rejected %s: %v", raw, err)
	}
}

func baseBlock() effects.Block {
	return effects.Block{ID: "CONFORMANCE", Trigger: effects.TriggerOnAction, Target: effects.Target{Selector: effects.TargetSingle}, Operations: []effects.Operation{{Op: effects.OpDamage, SubType: "FLAT", Value: 1, Unit: "FLAT"}}}
}

func TestAdminMetadataVocabularyIsValidatorCompilerConformant(t *testing.T) {
	metadata := effectcatalog.CatalogMetadata()
	for _, trigger := range metadata.Triggers {
		b := baseBlock()
		b.Trigger = effects.Trigger(trigger)
		b.Target.Selector = effects.TargetSelf
		b.Operations = []effects.Operation{{Op: effects.OpHeal, SubType: "FLAT", Value: 1, Unit: "FLAT"}}
		assertAccepted(t, b)
	}
	for _, target := range metadata.Targets {
		b := baseBlock()
		b.Target.Selector = effects.Selector(target)
		assertAccepted(t, b)
	}
	for _, field := range metadata.ConditionFields {
		b := baseBlock()
		value := any(1)
		if metadata.ConditionValueTypes[field] == "string" {
			value = "human"
		}
		b.Conditions = []effects.Condition{{Field: field, Operator: "EQ", Value: value}}
		assertAccepted(t, b)
	}
	for _, operator := range metadata.ConditionOperators {
		b := baseBlock()
		b.Conditions = []effects.Condition{{Field: "RANK", Operator: operator, Value: 1}}
		assertAccepted(t, b)
	}
	for _, stat := range metadata.Stats {
		b := baseBlock()
		b.Trigger = effects.TriggerAlwaysOn
		b.Target.Selector = effects.TargetSelf
		b.Operations = []effects.Operation{{Op: effects.OpStatMod, SubType: "ADD_FLAT", Stat: stat, Value: 1, Unit: "FLAT", StackPolicy: "STACK"}}
		assertAccepted(t, b)
	}
	for _, resource := range metadata.Resources {
		b := baseBlock()
		b.Trigger = effects.TriggerAlwaysOn
		b.Target.Selector = effects.TargetSelf
		b.Operations = []effects.Operation{{Op: effects.OpHeal, SubType: "RESOURCE", Resource: resource, Value: 1, Unit: "FLAT"}}
		assertAccepted(t, b)
	}
	for _, unit := range metadata.Units {
		b := baseBlock()
		b.Operations[0].Unit = unit
		assertAccepted(t, b)
	}
	for _, policy := range metadata.StackPolicies {
		b := baseBlock()
		b.Trigger = effects.TriggerAlwaysOn
		b.Target.Selector = effects.TargetSelf
		b.Operations = []effects.Operation{{Op: effects.OpStatMod, SubType: "ADD_FLAT", Stat: "ATTACK", Value: 1, Unit: "FLAT", StackPolicy: policy}}
		assertAccepted(t, b)
	}
	for _, definition := range metadata.Operations {
		if definition.Name == "MITIGATE_DAMAGE" || definition.Name == "RESOURCE_CONVERSION" || definition.Name == "THRESHOLD_RECOVERY" || definition.Name == "TENACITY_MODIFIER" || definition.Name == "STATUS_IMMUNITY" || definition.Name == "EFFECT_IMMUNITY" || definition.Name == "RANK_SCALED_DEBUFF" || definition.Name == "GU_SEAL" || definition.Name == "TARGET_REDIRECTION" || definition.Name == "OUTCOME_ROULETTE" || definition.Name == "SLOT_INTERACTION" || definition.Name == "REVIVE" {
			continue // Dedicated tests cover their trigger/target-only shapes.
		}
		if definition.Name == "NOTIFY" || definition.Name == "STATUS" || definition.Name == "RESTORE_RESOURCE" || definition.Name == "REPAIR_APERTURE" || definition.Name == "GROWTH_DURATION" || definition.Name == "MODIFY_MAX_RESOURCE" || definition.Name == "MODIFY_APTITUDE" || definition.Name == "ADVANCE_PROGRESSION" || definition.Name == "ADVANCE_MAJOR_REALM" || definition.Name == "PETRIFY_APERTURE" || definition.Name == "SUMMON" || definition.Name == "COUNTER_ADD" || definition.Name == "HP_COST" || definition.Name == "RESOURCE_CONVERSION" {
			assertAccepted(t, operationBlock(definition.Name, ""))
		}
		for _, subtype := range definition.SubTypes {
			if definition.Name == "STAT_MOD" && subtype == "ADD_COUNTER_DELTA" {
				continue // Dedicated trigger, target, and parameter contract below.
			}
			if definition.Name == "TRANSFER_RESOURCE" {
				// Transfer subtype validity depends on the resource/phase pair;
				// those combinations are covered by the operation-specific tests.
				continue
			}
			b := operationBlock(definition.Name, subtype)
			assertAccepted(t, b)
		}
	}
	statDelta := baseBlock()
	statDelta.Trigger = effects.TriggerBeforeHit
	statDelta.Target.Selector = effects.TargetSelf
	statDelta.Operations = []effects.Operation{{Op: effects.OpStatMod, SubType: "ADD_COUNTER_DELTA", Stat: "ATTACK", Value: 1, Key: "stacks", PerStack: 5}}
	assertAccepted(t, statDelta)
	thresholdRecovery := baseBlock()
	thresholdRecovery.Trigger = effects.TriggerBeforeHit
	thresholdRecovery.Target.Selector = effects.TargetSelf
	thresholdRecovery.Operations = []effects.Operation{{Op: effects.OpThresholdRecovery, HPThreshold: 0.3, HealPercent: 0.4, MaxUses: 1, CleanseDebuffs: true, LogAction: "RECOVERY", Message: "{actor} heals {heal_amount}"}}
	assertAccepted(t, thresholdRecovery)
	tenacity := baseBlock()
	tenacity.Trigger = effects.TriggerCombatPassive
	tenacity.Target.Selector = effects.TargetSelf
	tenacity.Operations = []effects.Operation{{Op: effects.OpTenacityModifier, PerMissingHP: 0.5}}
	assertAccepted(t, tenacity)
	statusImmunity := baseBlock()
	statusImmunity.Trigger = effects.TriggerCombatPassive
	statusImmunity.Target.Selector = effects.TargetSelf
	statusImmunity.Conditions = []effects.Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.3}}
	statusImmunity.Operations = []effects.Operation{{Op: effects.OpStatusImmunity, SubType: "STUN"}}
	assertAccepted(t, statusImmunity)
	for _, mode := range []string{"PERMANENT_STEAL", "TEMPORARY_STEAL"} {
		params := `{"base_chance":0.25,"chance_delta":0.1,"vital_penalty":0.5,"target_selection":"RANDOM"}`
		if mode == "TEMPORARY_STEAL" {
			params = `{"base_chance":0.4,"chance_delta":0.05,"vital_penalty":0.5,"target_selection":"RANDOM","duration":"FULL_COMBAT"}`
		}
		slotInteraction := baseBlock()
		slotInteraction.Trigger = effects.TriggerOnBattleStart
		slotInteraction.Target.Selector = effects.TargetSingle
		slotInteraction.Operations = []effects.Operation{{Op: effects.OpSlotInteraction, SubType: mode, Params: []byte(params)}}
		assertAccepted(t, slotInteraction)
	}
	for _, hook := range metadata.NativeHooks {
		b := baseBlock()
		b.Trigger = effects.Trigger(hook.Triggers[0])
		b.Target.Selector = effects.TargetSelf
		params := map[string]any{}
		for name, spec := range hook.Params {
			if spec.Required {
				params[name] = exampleParam(spec)
			}
		}
		raw, _ := json.Marshal(params)
		b.Operations = []effects.Operation{{Op: effects.OpUnique, Code: hook.Code, Params: raw}}
		assertAccepted(t, b)
	}
}

func TestWorldMapFlightStatusCanUseOnUseTrigger(t *testing.T) {
	b := baseBlock()
	b.Trigger = effects.TriggerOnUse
	b.Target.Selector = effects.TargetSelf
	b.Operations = []effects.Operation{{Op: effects.OpStatus, StatusID: "FLYING", Turns: 6, Chance: 1}}
	assertAccepted(t, b)
}

func operationBlock(name, subtype string) effects.Block {
	b := baseBlock()
	op := effects.Operation{Op: effects.OperationType(name), SubType: subtype, Value: 1, Unit: "FLAT"}
	switch name {
	case "SET_RULE":
		b.Trigger = effects.TriggerAlwaysOn
		b.Target.Selector = effects.TargetSelf
		op.Value = 0
		op.Unit = ""
		enabled := true
		op.Enabled = &enabled
	case "TRANSFER_RESOURCE":
		if subtype == "PERCENT_OF_MAX_HP" {
			op.Resource = "HP"
			op.Value = 0.15
			op.MinAmount = 30
			op.Phase = "LATE_RIDER"
			op.LogAction = "STEAL_LIFE"
			op.Message = "{actor} steals {amount} HP from {target}."
		} else {
			op.Resource = "MANA"
			op.Value = 0.25
		}
		op.Unit = ""
	case "STAT_MOD":
		b.Trigger = effects.TriggerAlwaysOn
		b.Target.Selector = effects.TargetSelf
		op.Stat = "ATTACK"
		op.StackPolicy = "STACK"
	case "STATUS":
		op.Unit = ""
		op.StatusID = "BLEED"
		op.Turns = 1
		op.Chance = 1
		op.StackPolicy = "STACK"
	case "BARRIER":
		op.Turns = 1
		op.StackPolicy = "STACK"
	case "SHIELD":
		b.Target.Selector = effects.TargetSelf
	case "HEAL":
		if subtype == "COUNTER_PERCENT" {
			b.Trigger = effects.TriggerCombatPassive
			b.Target.Selector = effects.TargetSelf
			op.Target = &effects.Target{Selector: effects.TargetSelf}
			op.Key = "test_counter"
			op.Phase = "FINAL_RIDER"
			op.Value = 0.05
		}
	case "CLEANSE":
		op.Value = 0
		op.Unit = ""
		if subtype == "STATUS" {
			op.StatusID = "STUN"
		}
	case "DISPEL":
		b.Trigger = effects.TriggerOnAction
		b.Target.Selector = effects.TargetSingle
		op.Value = 2
		if subtype == "ALL_BUFFS" || subtype == "PROTECTION" {
			op.Value = 0
		}
		op.Unit = ""
	case "REQUIRE":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Value = 0
		op.Unit = ""
	case "GRANT_RESOURCE":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		op.Resource = "DAO_MARK"
	case "RESTORE_RESOURCE":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		op.Resource = "EARTH_QI"
	case "MODIFY_MAX_RESOURCE":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Value = 1
		op.Unit = ""
		op.Resource = "SOUL"
	case "MODIFY_APTITUDE":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.SubType = "ADD_FLAT"
		op.Value = 1
		op.Unit = ""
	case "GRANT_BODY_MOD":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.SubType = subtype
		op.Value = 0
		op.Unit = ""
	case "ADVANCE_PROGRESSION":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Value = 0
		op.Unit = ""
	case "ADVANCE_MAJOR_REALM":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		op.Value = 0
	case "PETRIFY_APERTURE":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		op.Value = 0
	case "APPLY_STACKING_STATUS":
		op.StatusKey = "test_status"
		op.MaxStacks = 5
		op.PvPOnly = true
		op.Unit = ""
		op.Value = 0
	case "COUNTER_ADD":
		op.Key = "test_counter"
		op.Delta = 1
		op.Unit = ""
		op.Value = 0
	case "END_BATTLE":
		b.Trigger = effects.TriggerOnBattleStart
		b.Target.Selector = effects.TargetSelf
		op.Target = &effects.Target{Selector: effects.TargetAllEnemies}
		op.Value = 0
		op.Unit = ""
	case "THRESHOLD_CHECK":
		op.Key = "test_counter"
		op.Threshold = 3
		op.Unit = ""
		op.Value = 0
	case "MODIFY_APERTURE":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		if subtype == "TIME_RATE_MULTIPLY" {
			op.Value = 3
		}
	case "RANDOMIZE_TRAIT":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		op.Value = 0
	case "REPAIR_APERTURE":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
	case "OPEN_REWARD_CONTAINER", "PLACE_WORLD_OBJECT":
		b.Trigger = effects.TriggerOnConsume
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		op.Value = 0
	case "DISPLACE":
		op.Unit = ""
		if subtype == "SET_ROW" {
			b.Trigger = effects.TriggerOnBattleStart
			b.Target.Selector = effects.TargetSelf
			op.Value = 3
		} else if subtype == "RETREAT" {
			b.Target.Selector = effects.TargetSelf
		} else {
			b.Target.Selector = effects.TargetSingle
		}
		op.Value = 1
	case "DAMAGE_MODIFIER":
		op.Value = 1.3
		if subtype == "COUNTER_COUNT" {
			b.Trigger = effects.TriggerCombatPassive
			b.Target.Selector = effects.TargetSelf
			op.PerStack = 0.15
			op.Key = "test_counter"
			op.Phase = "FINAL_RIDER"
		} else if subtype == "TARGET_DEBUFF_COUNT" {
			b.Trigger = effects.TriggerOnAction
			b.Target.Selector = effects.TargetSingle
			op.PerTargetDebuff = 0.25
		} else if subtype == "TARGET_HAS_STATUS" {
			b.Trigger = effects.TriggerOnAction
			b.Target.Selector = effects.TargetSingle
			op.StatusID = "BURN"
		} else if subtype == "MINION_COUNT" {
			b.Trigger = effects.TriggerCombatPassive
			b.Target.Selector = effects.TargetSelf
			op.PerMinion = 0.15
		} else {
			b.Trigger = effects.TriggerOnAction
			b.Target.Selector = effects.TargetSingle
			op.PerMissingHP = 1.5
		}
		op.Unit = ""
	case "TARGET_DAMAGE_MODIFIER":
		b.Trigger = effects.TriggerOnAction
		b.Target.Selector = effects.TargetSingle
		op.SubType = "REDUCE"
		op.Value = 0.3
		op.MinMultiplier = 0.2
		op.Unit = ""
	case "DEFENSE_PENETRATION":
		b.Trigger = effects.TriggerOnAction
		b.Target.Selector = effects.TargetSingle
		op.Value = 0.5
		op.Unit = ""
	case "BONUS_STRIKE":
		b.Trigger = effects.TriggerOnAction
		b.Target.Selector = effects.TargetSingle
		op.Chance = 0.45
		op.Value = 0.75
		op.MinDamage = 50
		op.LogAction = "BONUS_STRIKE"
		op.Message = "{actor} hits {target} for {damage}."
	case "CHAIN_DAMAGE":
		b.Trigger = effects.TriggerOnAction
		b.Target.Selector = effects.TargetAllEnemies
		op.Value = 0.5
		op.MinDamage = 30
		op.TargetCount = 1
		op.LogAction = "CHAIN_DAMAGE"
		op.Message = "{actor} chains to {target} for {damage}."
	case "DETONATE_STATUS":
		b.Trigger = effects.TriggerOnAction
		b.Target.Selector = effects.TargetSingle
		op.StatusID = "BLEED"
		op.PerStack = 100
		op.PerStrength = 0
		op.MaxHPPercent = 0.08
		op.MinDamage = 0
		op.Phase = "EARLY_RIDER"
		op.LogAction = "STATUS_DETONATE"
		op.Message = "{actor} detonates {stack_count} stacks on {target} for {damage}."
	case "HP_COST":
		b.Trigger = effects.TriggerOnAction
		b.Target.Selector = effects.TargetSelf
		op.SubType = "CURRENT_HP_PERCENT"
		op.Value = 0.2
		op.MinCost = 10
		op.MinRemaining = 1
		op.Unit = ""
		op.LogAction = "BLOOD_DEMON_DISINTEGRATION"
		op.Message = "{actor} sacrifices {cost} HP."
	case "HIT_RULE":
		if subtype == "COUNTER_THRESHOLD" {
			b.Trigger = effects.TriggerOnBattleStart
			b.Target.Selector = effects.TargetSelf
			op.Params = json.RawMessage(`{"key":"test_counter","threshold":3,"guaranteed":true,"bypass_shield":true}`)
		} else if subtype == "MULTI_HIT" {
			b.Trigger = effects.TriggerOnAction
			b.Target.Selector = effects.TargetSingle
			op.Params = json.RawMessage(`{"count":3}`)
		} else if subtype == "FIRST_TURN_EVADE" {
			b.Trigger = effects.TriggerBeforeHit
			b.Target.Selector = effects.TargetSelf
		} else if subtype == "LETHAL_LUCK_WARD" {
			b.Trigger = effects.TriggerBeforeHit
			b.Target.Selector = effects.TargetSelf
			op.Params = json.RawMessage(`{"chance":0.25,"shield_percent":0.35,"shield_min":100}`)
		} else {
			b.Trigger = effects.TriggerOnAction
			b.Target.Selector = effects.TargetSingle
		}
		op.Unit = ""
		op.Value = 0
	case "TURN_ORDER_RULE":
		b.Trigger = effects.TriggerOnBattleStart
		b.Target.Selector = effects.TargetSelf
		op.Value = 0
		op.Unit = ""
		if subtype == "SEIZE_INITIATIVE" {
			op.Params = json.RawMessage(`{"lifespan_ratio":3}`)
		}
	case "GRANT_CAPABILITY":
		b.Trigger = effects.TriggerCapability
		b.Target.Selector = effects.TargetSelf
		op.Value = 0
		op.Unit = ""
		if subtype == "CAPTURE_BONUS" {
			op.Params = []byte(`{"value":0.15}`)
		}
	case "GROWTH_DURATION":
		b.Trigger = effects.TriggerAlwaysOn
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
	case "SUMMON":
		b.Trigger = effects.TriggerOnBattleStart
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		op.Value = 0
		op.Params = json.RawMessage(`{"monster_id":"monster_scholar_puppet","hp_scale":0.5,"damage_scale":0.33,"position":"frontline"}`)
	case "NOTIFY":
		b.Trigger = effects.TriggerOnUse
		b.Target.Selector = effects.TargetSelf
		op.Unit = ""
		op.Value = 0
		op.Message = "{gu} ghi lại cảnh vật."
	case "PERIODIC_RESOURCE":
		b.Trigger = effects.TriggerOnTick
		b.Target.Selector = effects.TargetSelf
		op.Resource = "MANA"
		if subtype == "GENERATION_WORKER_TICK" {
			op.Resource = "PRIMEVAL_STONE"
		}
		op.Unit = ""
	case "UNIQUE":
		return b
	}
	b.Operations = []effects.Operation{op}
	return b
}
func exampleParam(spec effectcatalog.ParamSpec) any {
	if len(spec.Enum) > 0 {
		for value := range spec.Enum {
			return value
		}
	}
	if spec.Type == "bool" {
		return true
	}
	if spec.Type == "string" {
		return "example"
	}
	if spec.Type == "integer" {
		if spec.Min != nil {
			return int(*spec.Min)
		}
		return 1
	}
	if spec.Type == "number_map" {
		return map[string]float64{"fire": 0.1}
	}
	if spec.Min != nil {
		return *spec.Min
	}
	return 1.0
}
