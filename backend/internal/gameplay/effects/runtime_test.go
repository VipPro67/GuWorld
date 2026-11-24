package effects

import (
	"math/rand"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestCompileCreatesImmutableTypedStatus(t *testing.T) {
	source := []Block{{ID: "hit", Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle}, Operations: []Operation{{Op: OpStatus, StatusID: statusfx.StatusBurn, Value: 12, Turns: 3, Chance: .5, StackPolicy: "MAX"}}}}
	compiled, err := (Compiler{}).Compile(source)
	if err != nil {
		t.Fatal(err)
	}
	source[0].Operations[0].Value = 99
	action := compiled.CompileCombatAction()
	if got := action.Operations[0].Status; got == nil || got.ID != statusfx.StatusBurn || got.Value != 12 || got.Turns != 3 || got.Chance != .5 || got.EffectID != "hit:0:0" || got.StackPolicy != "MAX" {
		t.Fatalf("unexpected compiled status: %+v", got)
	}
	action.Operations[0].Status.Value = 55
	if got := compiled.CompileCombatAction().Operations[0].Status.Value; got != 12 {
		t.Fatalf("compiled representation mutated: %v", got)
	}
}

func TestRevokeStatusCompilesPlayerStatusIDWithoutCombatNormalization(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		ID: "revoke_world_status", Trigger: TriggerOnBattleStart, Target: Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpRevokeStatus, StatusID: "regional_wall_suppression"}},
	}})
	if err != nil {
		t.Fatalf("compile REVOKE_STATUS: %v", err)
	}
	ops := compiled.AllOperations()
	if len(ops) != 1 || ops[0].StatusID != "regional_wall_suppression" {
		t.Fatalf("persistent status id was changed during compilation: %+v", ops)
	}
	if _, err := (Compiler{}).Compile([]Block{{
		ID: "conditional_revoke", Trigger: TriggerOnBattleStart, Target: Target{Selector: TargetSelf},
		Conditions: []Condition{{Field: "RANK", Operator: "GTE", Value: 1}},
		Operations: []Operation{{Op: OpRevokeStatus, StatusID: "regional_wall_suppression"}},
	}}); err == nil {
		t.Fatal("conditional REVOKE_STATUS was accepted even though the player adapter cannot evaluate conditions")
	}
}

func TestDisplaceCompileValidatesSetRowRangeAndInteger(t *testing.T) {
	base := Block{ID: "set_row", Trigger: TriggerOnBattleStart, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpDisplace, SubType: "SET_ROW", Value: 3}}}
	if _, err := (Compiler{}).Compile([]Block{base}); err != nil {
		t.Fatalf("valid SET_ROW failed to compile: %v", err)
	}
	for _, value := range []float64{0, 1.5, 6} {
		block := base
		block.Operations = []Operation{{Op: OpDisplace, SubType: "SET_ROW", Value: value}}
		if _, err := (Compiler{}).Compile([]Block{block}); err == nil {
			t.Errorf("invalid SET_ROW value %v compiled", value)
		}
	}
	conditional := base
	conditional.Conditions = []Condition{{Field: "RANK", Operator: "GTE", Value: 1}}
	if _, err := (Compiler{}).Compile([]Block{conditional}); err == nil {
		t.Fatal("conditioned battle-start displacement compiled despite unsupported adapter conditions")
	}
}

func TestStatusSuppressLogCompilesForDirectStatusModes(t *testing.T) {
	block := Block{ID: "wind", Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpStatus, StatusID: statusfx.StatusEvasion, Turns: 3, ApplicationMode: "DIRECT", SuppressLog: true}}}
	compiled, err := (Compiler{}).Compile([]Block{block})
	if err != nil {
		t.Fatal(err)
	}
	status := compiled.CompileCombatAction().Operations[0].Status
	if status == nil || !status.SuppressLog {
		t.Fatalf("compiled STATUS lost suppress_log: %+v", status)
	}
	if _, err := (Compiler{}).Compile([]Block{{ID: "invalid", Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpStatus, StatusID: statusfx.StatusEvasion, Turns: 3, SuppressLog: true}}}}); err == nil {
		t.Fatal("STATUS suppress_log without DIRECT application was accepted")
	}
	if _, err := (Compiler{}).Compile([]Block{{ID: "invalid", Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpDamage, Value: 1, SuppressLog: true}}}}); err == nil {
		t.Fatal("suppress_log on DAMAGE was accepted")
	}
}

func TestSetRuleCompilesEnabledFlag(t *testing.T) {
	enabled := true
	compiled, err := (Compiler{}).Compile([]Block{{ID: "immortal", Trigger: TriggerAlwaysOn, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpSetRule, SubType: "IMMORTAL", Enabled: &enabled}}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(Context{}, Event{Trigger: TriggerAlwaysOn})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Commands) != 1 || result.Commands[0].Operation != OpSetRule || result.Commands[0].SubType != "IMMORTAL" || !result.Commands[0].Enabled {
		t.Fatalf("SET_RULE command lost typed fields: %+v", result.Commands)
	}
	if _, err := (Compiler{}).Compile([]Block{{ID: "missing", Trigger: TriggerAlwaysOn, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpSetRule, SubType: "IMMORTAL"}}}}); err == nil {
		t.Fatal("SET_RULE without enabled flag was accepted")
	}
}

func TestHitRuleRequiresMatchingPhaseAndTarget(t *testing.T) {
	valid := Block{ID: "evade", Trigger: TriggerBeforeHit, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpHitRule, SubType: "FIRST_TURN_EVADE"}}}
	if _, err := (Compiler{}).Compile([]Block{valid}); err != nil {
		t.Fatalf("valid defensive HIT_RULE rejected: %v", err)
	}
	for _, block := range []Block{
		{ID: "bad-trigger", Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpHitRule, SubType: "FIRST_TURN_EVADE"}}},
		{ID: "bad-target", Trigger: TriggerBeforeHit, Target: Target{Selector: TargetSingle}, Operations: []Operation{{Op: OpHitRule, SubType: "FIRST_TURN_EVADE"}}},
		{ID: "bad-subtype", Trigger: TriggerBeforeHit, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpHitRule, SubType: "GUARANTEED"}}},
	} {
		if _, err := (Compiler{}).Compile([]Block{block}); err == nil {
			t.Errorf("invalid HIT_RULE combination compiled: %+v", block)
		}
	}
}

func TestConditionalCombatPassivesRequireExecutableLatePhase(t *testing.T) {
	condition := []Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.35}}
	valid := Block{
		ID: "PASSION", Trigger: TriggerCombatPassive, Target: Target{Selector: TargetSelf}, Conditions: condition,
		Operations: []Operation{{Op: OpDamageModifier, SubType: "MISSING_HP", Value: 1.5, Phase: "LATE_RIDER"}},
	}
	if _, err := (Compiler{}).Compile([]Block{valid}); err != nil {
		t.Fatalf("valid conditional passive rejected: %v", err)
	}
	invalid := valid
	invalid.Operations = []Operation{{Op: OpDamageModifier, SubType: "MISSING_HP", Value: 1.5}}
	if _, err := (Compiler{}).Compile([]Block{invalid}); err == nil {
		t.Fatal("conditional passive without an executable phase compiled")
	}
	for _, op := range []Operation{
		{Op: OpCleanse, SubType: "STATUS", StatusID: "STUN"},
		{Op: OpCleanse, SubType: "STATUS", StatusID: "STUN", Phase: "LATE_RIDER"},
	} {
		block := Block{ID: "CLEANSE", Trigger: TriggerCombatPassive, Target: Target{Selector: TargetSelf}, Operations: []Operation{op}}
		_, err := (Compiler{}).Compile([]Block{block})
		if op.Phase == "" && err == nil {
			t.Error("passive CLEANSE STATUS without phase compiled")
		}
		if op.Phase != "" && err != nil {
			t.Errorf("valid passive CLEANSE STATUS rejected: %v", err)
		}
	}
	wrongPhase := Block{ID: "CLEANSE", Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpCleanse, SubType: "STATUS", StatusID: "STUN", Phase: "LATE_RIDER"}}}
	if _, err := (Compiler{}).Compile([]Block{wrongPhase}); err == nil {
		t.Fatal("ON_ACTION cleanse with ignored LATE_RIDER phase compiled")
	}
}

func TestBerserkPrimitivesRejectNoOpsAndCompileConditionalImmunity(t *testing.T) {
	base := Block{ID: "BERSERK", Trigger: TriggerCombatPassive, Target: Target{Selector: TargetSelf}}
	if _, err := (Compiler{}).Compile([]Block{{ID: "zero_tenacity", Trigger: TriggerCombatPassive, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpTenacityModifier}}}}); err == nil {
		t.Fatal("zero TENACITY_MODIFIER compiled as a silent no-op")
	}
	valid := base
	valid.Conditions = []Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.3}}
	valid.Operations = []Operation{{Op: OpStatusImmunity, SubType: "STUN"}}
	compiled, err := (Compiler{}).Compile([]Block{valid})
	if err != nil {
		t.Fatalf("valid conditional STATUS_IMMUNITY rejected: %v", err)
	}
	ops := compiled.Operations(TriggerCombatPassive)
	if len(ops) != 1 || ops[0].Type != OpStatusImmunity || ops[0].SubType != "STUN" || len(ops[0].Conditions) != 1 {
		t.Fatalf("conditional status immunity lost compiled contract: %+v", ops)
	}
	invalid := valid
	invalid.Conditions = []Condition{{Field: "RANK", Operator: "GTE", Value: 1}}
	if _, err := (Compiler{}).Compile([]Block{invalid}); err == nil {
		t.Fatal("STATUS_IMMUNITY accepted a condition the combat projection cannot evaluate")
	}
}

func TestSlotInteractionValidatesModeAndConfiguration(t *testing.T) {
	valid := Block{ID: "steal", Trigger: TriggerOnBattleStart, Target: Target{Selector: TargetSingle}, Operations: []Operation{{
		Op: OpSlotInteraction, SubType: "PERMANENT_STEAL", Params: []byte(`{"base_chance":0.99,"chance_delta":0.1,"vital_penalty":0.5,"target_selection":"RANDOM"}`),
	}}}
	if _, err := (Compiler{}).Compile([]Block{valid}); err != nil {
		t.Fatalf("valid SLOT_INTERACTION rejected: %v", err)
	}
	for _, tc := range []struct {
		name  string
		block Block
	}{
		{"wrong-trigger", Block{ID: valid.ID, Trigger: TriggerOnAction, Target: valid.Target, Operations: valid.Operations}},
		{"wrong-target", Block{ID: valid.ID, Trigger: valid.Trigger, Target: Target{Selector: TargetSelf}, Operations: valid.Operations}},
		{"unknown-parameter", Block{ID: valid.ID, Trigger: valid.Trigger, Target: valid.Target, Operations: []Operation{{Op: OpSlotInteraction, SubType: "PERMANENT_STEAL", Params: []byte(`{"base_chance":0.2,"chance_delta":0.1,"vital_penalty":0.5,"target_selection":"RANDOM","silent":true}`)}}}},
		{"temporary-needs-duration", Block{ID: valid.ID, Trigger: valid.Trigger, Target: valid.Target, Operations: []Operation{{Op: OpSlotInteraction, SubType: "TEMPORARY_STEAL", Params: []byte(`{"base_chance":0.4,"chance_delta":0.05,"vital_penalty":0.5,"target_selection":"RANDOM"}`)}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := (Compiler{}).Compile([]Block{tc.block}); err == nil {
				t.Fatal("invalid SLOT_INTERACTION compiled")
			}
		})
	}
}

func TestHPCostCompilesExplicitEarlyRiderPhase(t *testing.T) {
	block := Block{ID: "GU_DETONATION", Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Operations: []Operation{{
		Op: OpHPCost, SubType: "CURRENT_HP_PERCENT", Value: 0.05, MinCost: 1, MinRemaining: 1,
		Phase: "EARLY_RIDER", LogAction: "GU_DETONATION", Message: "{actor}",
	}}}
	compiled, err := (Compiler{}).Compile([]Block{block})
	if err != nil {
		t.Fatalf("valid early HP_COST rejected: %v", err)
	}
	if got := compiled.AllOperations()[0].Phase; got != "EARLY_RIDER" {
		t.Fatalf("compiled HP_COST phase = %q, want EARLY_RIDER", got)
	}
	block.Operations[0].Phase = "LATE_RIDER"
	if _, err := (Compiler{}).Compile([]Block{block}); err == nil {
		t.Fatal("unsupported HP_COST phase compiled")
	}
}

func TestTransferResourceCompilesTypedCommand(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpTransferResource, SubType: "PERCENT_OF_TARGET", Resource: "MANA", Value: 0.25, Message: "transfer"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(Context{}, Event{Trigger: TriggerOnAction})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Commands) != 1 {
		t.Fatalf("commands = %d", len(result.Commands))
	}
	command := result.Commands[0]
	if command.Domain != DomainCombat || command.Operation != OpTransferResource || command.Resource != "MANA" || command.SubType != "PERCENT_OF_TARGET" || command.Value != 0.25 || command.Message != "transfer" {
		t.Fatalf("resource transfer command lost fields: %+v", command)
	}
	for _, invalid := range []Operation{
		{Op: OpTransferResource, SubType: "PERCENT_OF_TARGET", Resource: "MANA", Value: 0},
		{Op: OpTransferResource, SubType: "PERCENT_OF_TARGET", Resource: "SOUL", Value: 0.25},
		{Op: OpTransferResource, SubType: "FLAT", Resource: "MANA", Value: 10},
	} {
		if _, err := (Compiler{}).Compile([]Block{{Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle}, Operations: []Operation{invalid}}}); err == nil {
			t.Errorf("invalid resource transfer compiled: %+v", invalid)
		}
	}
}

func TestHPTransferCompilesPhaseAndMinimum(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpTransferResource, SubType: "PERCENT_OF_MAX_HP", Resource: "HP", Value: 0.15, MinAmount: 30, Phase: "LATE_RIDER", LogAction: "STEAL_LIFE", Message: "{actor} steals {amount} HP."}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(Context{}, Event{Trigger: TriggerOnAction})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Commands) != 1 {
		t.Fatalf("commands = %d", len(result.Commands))
	}
	command := result.Commands[0]
	if command.Operation != OpTransferResource || command.Resource != "HP" || command.SubType != "PERCENT_OF_MAX_HP" || command.Value != 0.15 || command.MinAmount != 30 || command.Phase != "LATE_RIDER" {
		t.Fatalf("HP transfer command lost authored parameters: %+v", command)
	}
}

func TestShieldAbsorptionCompilesAsEarlyHPTransfer(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpTransferResource, SubType: "PERCENT_OF_TARGET_SHIELD", Resource: "HP", Value: 0.5, Phase: "EARLY_RIDER", LogAction: "DEVOUR_HEAVEN_EARTH", Message: "{actor} absorbs {amount} shield."}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ops := compiled.Operations(TriggerOnAction)
	if len(ops) != 1 || ops[0].Resource != "HP" || ops[0].SubType != "PERCENT_OF_TARGET_SHIELD" || ops[0].Phase != "EARLY_RIDER" {
		t.Fatalf("shield absorption projection = %+v", ops)
	}
}

func TestHPCostCompilesTypedCombatCommand(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpHPCost, SubType: "CURRENT_HP_PERCENT", Value: 0.2, MinCost: 10, MinRemaining: 1, LogAction: "BLOOD_DEMON_DISINTEGRATION", Message: "{actor} pays {cost}."}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(Context{}, Event{Trigger: TriggerOnAction})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Commands) != 1 {
		t.Fatalf("commands = %d", len(result.Commands))
	}
	command := result.Commands[0]
	if command.Domain != DomainCombat || command.Operation != OpHPCost || command.Target != TargetSelf || command.Value != 0.2 || command.MinCost != 10 || command.MinRemaining != 1 {
		t.Fatalf("HP cost command lost fields: %+v", command)
	}
}

func TestNotifyCompilesForGuUseAndRejectsEmptyMessage(t *testing.T) {
	block := Block{ID: "record", Trigger: TriggerOnUse, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpNotify, Message: "{gu} ghi lại."}}}
	compiled, err := (Compiler{}).Compile([]Block{block})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateContextSupport(DomainGuUse, []Block{block}); err != nil {
		t.Fatalf("NOTIFY should be supported for Gu use: %v", err)
	}
	result, err := compiled.Evaluate(Context{}, Event{Trigger: TriggerOnUse})
	if err != nil || len(result.Commands) != 1 || result.Commands[0].Domain != DomainGuUse || result.Commands[0].Message != "{gu} ghi lại." {
		t.Fatalf("NOTIFY command = %+v, err=%v", result.Commands, err)
	}
	block.Operations[0].Message = "  "
	if _, err := (Compiler{}).Compile([]Block{block}); err == nil {
		t.Fatal("empty NOTIFY message should fail compilation")
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
	if len(Executors()) != 8 {
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

func TestCompileDeductionFinishChancePreservesValidatedParams(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		ID: "DEDUCTION_CHANCE", Trigger: TriggerAlwaysOn, Target: Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpDeductionFinishChance, Params: []byte(`{"base_chance":0.25,"per_rank_step":0.05,"max_chance":0.95}`)}},
	}})
	if err != nil {
		t.Fatalf("compile deduction chance: %v", err)
	}
	ops := compiled.Operations(TriggerAlwaysOn)
	if len(ops) != 1 || ops[0].Type != OpDeductionFinishChance || ops[0].Values["base_chance"] != 0.25 || ops[0].Values["max_chance"] != 0.95 {
		t.Fatalf("compiled deduction chance = %+v", ops)
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

func TestRaceConditionsMatchExactActorAndTargetKeys(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle},
		Conditions: []Condition{
			{Field: "ACTOR_RACE", Operator: "EQ", Value: "human"},
			{Field: "TARGET_RACE", Operator: "NE", Value: "rockman"},
		},
		Operations: []Operation{{Op: OpDamage, SubType: "FLAT", Value: 1}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	result, err := compiled.Evaluate(Context{
		Actor:  Snapshot{Race: "human"},
		Target: Snapshot{Race: "merman"},
	}, Event{Trigger: TriggerOnAction})
	if err != nil || len(result.Commands) != 1 {
		t.Fatalf("Evaluate(matching race keys) = (%+v, %v), want one command", result, err)
	}

	result, err = compiled.Evaluate(Context{
		Actor:  Snapshot{Race: "human"},
		Target: Snapshot{Race: "rockman"},
	}, Event{Trigger: TriggerOnAction})
	if err != nil || len(result.Commands) != 0 {
		t.Fatalf("Evaluate(non-matching target race) = (%+v, %v), want no commands", result, err)
	}

	result, err = compiled.Evaluate(Context{Actor: Snapshot{Race: "human"}}, Event{Trigger: TriggerOnAction})
	if err != nil || len(result.Commands) != 0 {
		t.Fatalf("Evaluate(missing target race) = (%+v, %v), want no commands and no error", result, err)
	}
}

func TestCompilerRejectsInvalidRaceConditionValues(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
	}{
		{name: "numeric race value", condition: Condition{Field: "TARGET_RACE", Operator: "EQ", Value: 1}},
		{name: "ordered race operator", condition: Condition{Field: "TARGET_RACE", Operator: "GT", Value: "human"}},
		{name: "string numeric value", condition: Condition{Field: "RANK", Operator: "EQ", Value: "six"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (Compiler{}).Compile([]Block{{
				Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle},
				Conditions: []Condition{tt.condition}, Operations: []Operation{{Op: OpDamage, SubType: "FLAT", Value: 1}},
			}})
			if err == nil {
				t.Fatalf("Compile(%+v) succeeded, want invalid condition error", tt.condition)
			}
		})
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

func TestCapabilityQueryReadsTypedGrant(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger: TriggerCapability,
		Target:  Target{Selector: TargetSelf},
		Operations: []Operation{{
			Op: OpGrantCapability, SubType: "CAPTURE_BONUS", Params: []byte(`{"value":0.15}`),
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	capability, ok := compiled.Capability("capture_bonus")
	if !ok || capability.Code != "CAPTURE_BONUS" || string(capability.Params) != `{"value":0.15}` {
		t.Fatalf("typed capability = %+v, %v", capability, ok)
	}
	if got := compiled.Capabilities(); len(got) != 1 || got[0].Code != "CAPTURE_BONUS" {
		t.Fatalf("typed capabilities = %+v", got)
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

func TestLastNativeParamFloatReturnsLastMatchingValue(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{
		{
			Trigger: TriggerAlwaysOn,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{{
				Op: OpUnique, Code: "GROWTH_DURATION_HOURS", Params: []byte(`{"value":100}`),
			}},
		},
		{
			Trigger: TriggerAlwaysOn,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{{
				Op: OpUnique, Code: "GROWTH_DURATION_HOURS", Params: []byte(`{"value":200}`),
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := compiled.LastNativeParamFloat(TriggerAlwaysOn, "growth_duration_hours", "value"); !ok || got != 200 {
		t.Fatalf("LastNativeParamFloat() = %v, %v; want 200, true", got, ok)
	}
}

func TestGrowthDurationHoursReadsTypedOperationAndLegacyHook(t *testing.T) {
	tests := []struct {
		name string
		op   Operation
	}{
		{name: "typed", op: Operation{Op: OpGrowthDuration, Value: 48}},
		{name: "legacy", op: Operation{Op: OpUnique, Code: "GROWTH_DURATION_HOURS", Params: []byte(`{"value":48}`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, err := (Compiler{}).Compile([]Block{{
				Trigger: TriggerAlwaysOn, Target: Target{Selector: TargetSelf}, Operations: []Operation{tt.op},
			}})
			if err != nil {
				t.Fatal(err)
			}
			if got, ok := compiled.GrowthDurationHours(); !ok || got != 48 {
				t.Fatalf("GrowthDurationHours() = %v, %v; want 48, true", got, ok)
			}
		})
	}
}

func TestUnscopedPassiveProjectionsExcludeConditionalBlocks(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		Trigger:    TriggerAlwaysOn,
		Target:     Target{Selector: TargetSelf},
		Conditions: []Condition{{Field: "ACTOR_RACE", Operator: "EQ", Value: "human"}},
		Operations: []Operation{{
			Op: OpStatMod, Stat: "ATTACK", SubType: "ADD_FLAT", Value: 200,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := compiled.StatValue(TriggerAlwaysOn, "ATTACK", "ADD_FLAT"); got != 0 {
		t.Fatalf("unscoped stat projection = %v, want 0", got)
	}
	if got := compiled.Passives(); len(got) != 0 {
		t.Fatalf("unscoped passive projection = %+v, want none", got)
	}
}

func TestNativeMechanicReturnsDefensiveCopy(t *testing.T) {
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
	mechanic, ok := compiled.NativeMechanic(" marketplace_access ")
	if !ok || mechanic.Code != "MARKETPLACE_ACCESS" {
		t.Fatalf("unexpected mechanic: %+v, %v", mechanic, ok)
	}
	mechanic.Params[0] = 'x'
	again, _ := compiled.NativeMechanic("MARKETPLACE_ACCESS")
	if string(again.Params) != `{}` {
		t.Fatalf("compiled mechanic was mutated: %s", again.Params)
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

func TestCompilerRejectsCounterAddSilentNoop(t *testing.T) {
	_, err := (Compiler{}).Compile([]Block{{
		ID: "STACK", Trigger: TriggerOnAction,
		Target:     Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpCounterAdd, Key: "stacks", Delta: 0}},
	}})
	if err == nil {
		t.Fatal("COUNTER_ADD with zero delta compiled as a silent no-op")
	}
}

func TestCompilerAcceptsTurnStartCounterAdd(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		ID: "POEM_INTENT", Trigger: TriggerOnTurnStart,
		Target:     Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpCounterAdd, Key: "thi_y", Delta: 1, Min: 0, Max: 10}},
	}})
	if err != nil {
		t.Fatalf("compile turn-start counter: %v", err)
	}
	ops := compiled.Operations(TriggerOnTurnStart)
	if len(ops) != 1 || ops[0].Type != OpCounterAdd || ops[0].Key != "thi_y" {
		t.Fatalf("compiled turn-start counter = %+v", ops)
	}
}

func TestCompilerAcceptsBeforeHitSelfCounterAdd(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		ID: "EATING_LOSS_STACK", Trigger: TriggerBeforeHit,
		Target:     Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpCounterAdd, Key: "eating_loss_stacks", Delta: 1, Min: 0, Max: 10}},
	}})
	if err != nil {
		t.Fatalf("compile before-hit counter: %v", err)
	}
	ops := compiled.Operations(TriggerBeforeHit)
	if len(ops) != 1 || ops[0].Type != OpCounterAdd || ops[0].Key != "eating_loss_stacks" {
		t.Fatalf("compiled before-hit counter = %+v", ops)
	}
	conditional := Block{
		ID: "CONDITIONAL_STACK", Trigger: TriggerBeforeHit,
		Target: Target{Selector: TargetSelf}, Conditions: []Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.3}},
		Operations: []Operation{{Op: OpCounterAdd, Key: "stacks", Delta: 1}},
	}
	if _, err := (Compiler{}).Compile([]Block{conditional}); err == nil {
		t.Fatal("conditional BEFORE_HIT COUNTER_ADD compiled without a condition-aware executor")
	}
}

func TestCompilerAcceptsCounterDeltaStatModifier(t *testing.T) {
	block := Block{ID: "STACK_STATS", Trigger: TriggerBeforeHit, Target: Target{Selector: TargetSelf}, Operations: []Operation{{
		Op: OpStatMod, SubType: "ADD_COUNTER_DELTA", Stat: "ATTACK", Value: 0, Key: "stacks", PerStack: 5,
	}}}
	compiled, err := (Compiler{}).Compile([]Block{block})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if got := compiled.Operations(TriggerBeforeHit); len(got) != 1 || got[0].Key != "stacks" || got[0].PerStack != 5 {
		t.Fatalf("compiled counter stat modifier = %+v", got)
	}
}

func TestCompilerAcceptsBeforeHitCounterNotification(t *testing.T) {
	block := Block{ID: "STACK_LOG", Trigger: TriggerBeforeHit, Target: Target{Selector: TargetSelf}, Operations: []Operation{{
		Op: OpNotify, Key: "stacks", Phase: "BEFORE_HIT", LogAction: "STACK_GAINED", Message: "{actor}: {counter_count}",
	}}}
	compiled, err := (Compiler{}).Compile([]Block{block})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if got := compiled.Operations(TriggerBeforeHit); len(got) != 1 || got[0].Type != OpNotify {
		t.Fatalf("compiled notification = %+v", got)
	}
}

func TestCompilerAcceptsThresholdRecovery(t *testing.T) {
	block := Block{ID: "RECOVERY", Trigger: TriggerBeforeHit, Target: Target{Selector: TargetSelf}, Operations: []Operation{{
		Op: OpThresholdRecovery, HPThreshold: 0.3, HealPercent: 0.4, MaxUses: 1,
		CleanseDebuffs: true, LogAction: "QI_JUE_RETURN_LIFE", Message: "{actor} heals {heal_amount}",
	}}}
	compiled, err := (Compiler{}).Compile([]Block{block})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	ops := compiled.Operations(TriggerBeforeHit)
	if len(ops) != 1 || ops[0].Type != OpThresholdRecovery || ops[0].HPThreshold != 0.3 || ops[0].HealPercent != 0.4 || ops[0].MaxUses != 1 {
		t.Fatalf("compiled threshold recovery = %+v", ops)
	}
}

func TestCompilerAcceptsFinalRiderCounterPayout(t *testing.T) {
	block := Block{
		ID: "EATING_LOSS_PAYOUT", Trigger: TriggerCombatPassive,
		Target: Target{Selector: TargetSelf}, Operations: []Operation{
			{Op: OpDamageModifier, SubType: "COUNTER_COUNT", Value: 1.3, PerStack: 0.15, Key: "eating_loss_stacks", Phase: "FINAL_RIDER"},
			{Op: OpHeal, SubType: "COUNTER_PERCENT", Value: 0.05, Key: "eating_loss_stacks", Phase: "FINAL_RIDER", Target: &Target{Selector: TargetSelf}},
		},
	}
	if _, err := (Compiler{}).Compile([]Block{block}); err != nil {
		t.Fatalf("compile counter payout: %v", err)
	}
	block.Operations[1].Key = ""
	if _, err := (Compiler{}).Compile([]Block{block}); err == nil {
		t.Fatal("HEAL COUNTER_PERCENT compiled without a counter key")
	}
}

func TestCompilerAcceptsBattleStartEndBattle(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		ID: "INSTANT_WIN", Trigger: TriggerOnBattleStart,
		Target:     Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpEndBattle, SubType: "FORCE_VICTORY", Target: &Target{Selector: TargetAllEnemies}}},
	}})
	if err != nil {
		t.Fatalf("compile END_BATTLE: %v", err)
	}
	ops := compiled.Operations(TriggerOnBattleStart)
	if len(ops) != 1 || ops[0].Type != OpEndBattle || ops[0].SubType != "FORCE_VICTORY" {
		t.Fatalf("compiled END_BATTLE = %+v", ops)
	}
}

func TestCompilerRejectsThresholdCheckWithoutExecutor(t *testing.T) {
	_, err := (Compiler{}).Compile([]Block{{
		ID: "THRESHOLD", Trigger: TriggerOnAction,
		Target:     Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpThresholdCheck, Key: "stacks", Threshold: 3}},
	}})
	if err == nil {
		t.Fatal("THRESHOLD_CHECK compiled without a runtime executor")
	}
}

func TestCompilerRejectsConditionalPersistentStackingStatus(t *testing.T) {
	_, err := (Compiler{}).Compile([]Block{{
		ID: "PERSISTENT_STATUS", Trigger: TriggerOnHit,
		Target:     Target{Selector: TargetSingle},
		Conditions: []Condition{{Field: "RANK", Operator: "GTE", Value: 1}},
		Operations: []Operation{{Op: OpApplyStackingStatus, StatusKey: "sinh_khi_tan", MaxStacks: 5, PvPOnly: true}},
	}})
	if err == nil {
		t.Fatal("conditional APPLY_STACKING_STATUS compiled though its post-combat consumer cannot evaluate conditions")
	}
}

func TestCompilerRejectsUnsupportedPersistentStatusScope(t *testing.T) {
	_, err := (Compiler{}).Compile([]Block{{
		ID: "PERSISTENT_STATUS", Trigger: TriggerOnHit,
		Target:     Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpApplyStackingStatus, StatusKey: "sinh_khi_tan", MaxStacks: 5}},
	}})
	if err == nil {
		t.Fatal("APPLY_STACKING_STATUS compiled without pvp_only=true")
	}
}

func TestCompilerRejectsOnActionPersistentStackingStatus(t *testing.T) {
	_, err := (Compiler{}).Compile([]Block{{
		ID: "PERSISTENT_STATUS", Trigger: TriggerOnAction,
		Target:     Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpApplyStackingStatus, StatusKey: "sinh_khi_tan", MaxStacks: 5, PvPOnly: true}},
	}})
	if err == nil {
		t.Fatal("ON_ACTION APPLY_STACKING_STATUS compiled although the persistent grant consumer requires a landed hit")
	}
}

func TestCompilerPreservesPreRiderHPCostPhase(t *testing.T) {
	compiled, err := (Compiler{}).Compile([]Block{{
		ID: "BLOOD_SACRIFICE", Trigger: TriggerOnAction,
		Target:     Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpHPCost, SubType: "CURRENT_HP_PERCENT", Target: &Target{Selector: TargetSelf}, Value: 0.15, MinCost: 1, MinRemaining: 1, Phase: "PRE_RIDER", LogAction: "BLOOD_SACRIFICE", Message: "{actor} sacrifices {actual_cost} HP"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ops := compiled.Operations(TriggerOnAction)
	if len(ops) != 1 || ops[0].Phase != "PRE_RIDER" {
		t.Fatalf("compiled HP_COST phase = %+v", ops)
	}
}
