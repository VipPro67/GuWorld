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
		b.Conditions = []effects.Condition{{Field: field, Operator: "EQ", Value: 1}}
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
		if definition.Name == "STATUS" {
			assertAccepted(t, operationBlock(definition.Name, ""))
		}
		for _, subtype := range definition.SubTypes {
			b := operationBlock(definition.Name, subtype)
			assertAccepted(t, b)
		}
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

func operationBlock(name, subtype string) effects.Block {
	b := baseBlock()
	op := effects.Operation{Op: effects.OperationType(name), SubType: subtype, Value: 1, Unit: "FLAT"}
	switch name {
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
	case "CLEANSE":
		op.Value = 0
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
	if spec.Type == "integer" {
		if spec.Min != nil {
			return int(*spec.Min)
		}
		return 1
	}
	if spec.Min != nil {
		return *spec.Min
	}
	return 1.0
}
