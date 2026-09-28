package effects

import (
	"math/rand"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestCompileCreatesImmutableTypedStatus(t *testing.T) {
	source := []Block{{ID: "hit", Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle}, Operations: []Operation{{Op: OpStatus, StatusID: statusfx.StatusBurn, Value: 12, Turns: 3, Chance: .5}}}}
	compiled, err := (Compiler{}).Compile(source)
	if err != nil {
		t.Fatal(err)
	}
	source[0].Operations[0].Value = 99
	action := compiled.CompileCombatAction()
	if got := action.Operations[0].Status; got == nil || got.ID != statusfx.StatusBurn || got.Value != 12 || got.Turns != 3 || got.Chance != .5 {
		t.Fatalf("unexpected compiled status: %+v", got)
	}
	action.Operations[0].Status.Value = 55
	if got := compiled.CompileCombatAction().Operations[0].Status.Value; got != 12 {
		t.Fatalf("compiled representation mutated: %v", got)
	}
}
func TestCompilerRejectsUnknownVocabulary(t *testing.T) {
	tests := []Block{
		{Trigger: "NOPE", Target: Target{Selector: TargetSelf}},
		{Trigger: TriggerOnAction, Target: Target{Selector: "NOPE"}},
		{Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Conditions: []Condition{{Field: "NOPE", Operator: "EQ"}}},
		{Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: "NOPE"}}},
	}
	for i, definition := range tests {
		if _, err := (Compiler{}).Compile([]Block{definition}); err == nil {
			t.Fatalf("case %d accepted", i)
		}
	}
}
func TestEvaluateDeterministicContextAndBudgets(t *testing.T) {
	ops := make([]Operation, DefaultLimits.MaxEventsPerAction+1)
	for i := range ops {
		ops[i] = Operation{Op: OpDamage, Value: 1}
	}
	compiled, err := (Compiler{}).Compile([]Block{{Trigger: TriggerOnHit, Target: Target{Selector: TargetSingle}, Operations: ops}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := Context{RNG: rand.New(rand.NewSource(42)), Actor: Snapshot{ID: "a"}, Target: Snapshot{ID: "b"}, Rank: 5, Mastery: 7, DaoMarks: map[string]int64{"fire": 10}, Path: []string{"fire"}, CatalogRevision: 3}
	result, err := compiled.Evaluate(ctx, Event{Trigger: TriggerOnHit})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated || len(result.Commands) != DefaultLimits.MaxEventsPerAction {
		t.Fatalf("budget not enforced: %+v", result)
	}
	ctx.Depth = DefaultLimits.MaxTriggerDepth
	result, _ = compiled.Evaluate(ctx, Event{Trigger: TriggerOnHit})
	if !result.Truncated || len(result.Commands) != 0 {
		t.Fatalf("depth not enforced: %+v", result)
	}
}
func TestDomainExecutorsCoverAllDomains(t *testing.T) {
	if len(Executors()) != 7 {
		t.Fatalf("got %d executors", len(Executors()))
	}
}

func TestEvaluatePreservesTypedCommandAndOperationTarget(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerOnHit, Target: Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpStatMod, SubType: "ADD_FLAT", Stat: "speed", Path: "time", Value: 4, Target: &Target{Selector: TargetSelf}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(Context{}, Event{Trigger: TriggerOnHit})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Commands) != 1 {
		t.Fatalf("commands = %d", len(result.Commands))
	}
	command := result.Commands[0]
	if command.Operation != OpStatMod || command.SubType != "ADD_FLAT" || command.Stat != "SPEED" || command.Path != "time" || command.Target != TargetSelf {
		t.Fatalf("typed command data lost: %+v", command)
	}
}

func TestCompileNormalizesConditionOperator(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{Trigger: TriggerOnHit, Target: Target{Selector: TargetSelf}, Conditions: []Condition{{Field: "rank", Operator: "gte", Value: 2}}, Operations: []Operation{{Op: OpHeal, Value: 1}}}})
	if err != nil {
		t.Fatal(err)
	}
	result, _ := compiled.Evaluate(Context{Rank: 2}, Event{Trigger: TriggerOnHit})
	if len(result.Commands) != 1 {
		t.Fatal("normalized condition did not match")
	}
}

func TestCapabilityQueryReturnsDefensiveCopy(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerCapability,
		Target:  Target{Selector: TargetSelf},
		Operations: []Operation{{
			Op: OpUnique, Code: "MARKETPLACE_ACCESS", Params: []byte(`{}`),
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	capability, ok := compiled.Capability("marketplace_access")
	if !ok || capability.Code != "MARKETPLACE_ACCESS" || string(capability.Params) != `{}` {
		t.Fatalf("unexpected capability: %+v, %v", capability, ok)
	}
	capability.Params[0] = 'x'
	again, _ := compiled.Capability("MARKETPLACE_ACCESS")
	if string(again.Params) != `{}` {
		t.Fatalf("compiled capability was mutated: %s", again.Params)
	}
}

func TestCapabilityQueryIgnoresNonUniqueOperations(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerCapability,
		Target:  Target{Selector: TargetSelf},
		Operations: []Operation{{
			Op: OpHeal, SubType: "FLAT", Code: "MARKETPLACE_ACCESS", Value: 1,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := compiled.Capability("MARKETPLACE_ACCESS"); ok {
		t.Fatal("non-UNIQUE operation granted a capability")
	}
}

func TestOperationsAndStatValueAreTypedDefensiveProjections(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{Trigger: TriggerAlwaysOn, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpStatMod, Stat: "INVESTIGATION", SubType: "ADD_FLAT", Value: 3}}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := compiled.StatValue(TriggerAlwaysOn, "investigation", "add_flat"); got != 3 {
		t.Fatalf("stat = %v", got)
	}
	ops := compiled.Operations(TriggerAlwaysOn)
	ops[0].Value = 99
	if got := compiled.StatValue(TriggerAlwaysOn, "INVESTIGATION", "ADD_FLAT"); got != 3 {
		t.Fatalf("projection mutated: %v", got)
	}
}
