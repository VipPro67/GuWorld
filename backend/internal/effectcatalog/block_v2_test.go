package effectcatalog

import (
	"bytes"
	"strings"
	"testing"
)

func TestCanonicalBlockV2Envelope(t *testing.T) {
	array := []byte(`[{"id":"BASIC_HIT","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10,"unit":"FLAT"}]}]`)
	first, issues := CanonicalBlockV2Envelope(array, Location{Source: "test"})
	if len(issues) != 0 {
		t.Fatalf("array issues = %+v", issues)
	}
	second, issues := CanonicalBlockV2Envelope(first, Location{Source: "test"})
	if len(issues) != 0 || !bytes.Equal(first, second) {
		t.Fatalf("envelope not idempotent: %s / %s, issues=%+v", first, second, issues)
	}
	if _, issues := CanonicalBlockV2Envelope([]byte(`{"schema_version":3,"blocks":[]}`), Location{}); len(issues) == 0 {
		t.Fatal("unsupported schema version accepted")
	}
	if _, issues := CanonicalBlockV2Envelope([]byte(`{"schema_version":2,"blocks":[],"active":{"damage_flat":1}}`), Location{}); len(issues) == 0 || issues[0].Code != "unknown_field" {
		t.Fatal("mixed legacy sibling field was silently discarded")
	}
}

func TestResourceConversionValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"TRANSFORM","trigger":"ON_CAST","target":{"selector":"SELF"},"operations":[{"op":"RESOURCE_CONVERSION","sub_type":"CURRENT_HP_TO_SHIELD","value":1,"min_remaining":1,"log_action":"TRANSFORM","message":"{actor} converts {amount} HP."}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{Source: "test"}); len(issues) != 0 {
		t.Fatalf("valid conversion rejected: %+v", issues)
	}
	for _, raw := range []string{
		strings.Replace(valid, `"trigger":"ON_CAST"`, `"trigger":"ON_ACTION"`, 1),
		strings.Replace(valid, `"selector":"SELF"`, `"selector":"TARGET"`, 1),
		strings.Replace(valid, `"value":1`, `"value":0`, 1),
		strings.Replace(valid, `"min_remaining":1`, `"min_remaining":0`, 1),
		strings.Replace(valid, `"sub_type":"CURRENT_HP_TO_SHIELD"`, `"sub_type":"MANA_TO_SHIELD"`, 1),
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{Source: "test"}); len(issues) == 0 {
			t.Errorf("invalid conversion accepted: %s", raw)
		}
	}
}

func TestCounterPayoutNotifyValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"PAYOUT","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"operations":[{"op":"NOTIFY","key":"eating_loss_stacks","phase":"FINAL_RIDER","log_action":"EATING_LOSS_BLESSING","message":"{counter_count} stacks, +{damage_percent}%, heal {heal_amount}."}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{Source: "test"}); len(issues) != 0 {
		t.Fatalf("valid counter payout notification rejected: %+v", issues)
	}
	invalid := []string{
		strings.Replace(valid, `"phase":"FINAL_RIDER",`, "", 1),
		strings.Replace(valid, `"key":"eating_loss_stacks"`, `"key":"Eating-Loss"`, 1),
		strings.Replace(valid, `"trigger":"COMBAT_PASSIVE"`, `"trigger":"ON_ACTION"`, 1),
	}
	for _, raw := range invalid {
		if issues := ValidateBlockV2([]byte(raw), Location{Source: "test"}); len(issues) == 0 {
			t.Errorf("invalid counter payout notification accepted: %s", raw)
		}
	}
}

func TestBlockV2AcceptsCataloguedOperation(t *testing.T) {
	raw := []byte(`{"schema_version":2,"blocks":[{"id":"BASIC_HIT","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10,"unit":"FLAT"}]}]}`)
	if issues := ValidateBlockV2(raw, Location{Source: "gu", Row: 2, Key: "GU_TEST"}); len(issues) != 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
}

func TestCounterAddRejectsSilentNoOpsAndInvalidBounds(t *testing.T) {
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"STACK","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"COUNTER_ADD","key":"stacks","delta":0}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"STACK","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"COUNTER_ADD","key":"stacks","delta":1,"min":6,"max":5}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"STACK","trigger":"ON_HIT","target":{"selector":"SELF"},"operations":[{"op":"COUNTER_ADD","key":"stacks","delta":1}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"STACK","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"conditions":[{"field":"HP_PERCENT","operator":"LTE","value":0.3}],"operations":[{"op":"COUNTER_ADD","key":"stacks","delta":1}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid COUNTER_ADD accepted: %s", raw)
		}
	}
}

func TestCounterScaledDamageAndHealingValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"EATING_LOSS_PAYOUT","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"operations":[{"op":"DAMAGE_MODIFIER","sub_type":"COUNTER_COUNT","value":1.3,"per_stack":0.15,"key":"eating_loss_stacks","phase":"FINAL_RIDER"},{"op":"HEAL","sub_type":"COUNTER_PERCENT","target":{"selector":"SELF"},"value":0.05,"key":"eating_loss_stacks","phase":"FINAL_RIDER"}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid counter payout rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD_DAMAGE","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"operations":[{"op":"DAMAGE_MODIFIER","sub_type":"COUNTER_COUNT","value":1.3,"per_stack":0.15,"phase":"FINAL_RIDER"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_HEAL","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"operations":[{"op":"HEAL","sub_type":"COUNTER_PERCENT","target":{"selector":"SELF"},"value":0.05,"phase":"FINAL_RIDER"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_PHASE","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"operations":[{"op":"HEAL","sub_type":"COUNTER_PERCENT","target":{"selector":"SELF"},"value":0.05,"key":"eating_loss_stacks","phase":"LATE_RIDER"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid counter payout accepted: %s", raw)
		}
	}
}

func TestThresholdCheckWithoutExecutorIsRejected(t *testing.T) {
	raw := `{"schema_version":2,"blocks":[{"id":"THRESHOLD","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"THRESHOLD_CHECK","key":"stacks","threshold":3,"auto_reset":true}]}]}`
	if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
		t.Fatal("THRESHOLD_CHECK was accepted without a runtime executor")
	}
}

func TestMitigateDamageRequiresSupportedBeforeHitShape(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"BLOOD_STAINED_ROBE","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"MITIGATE_DAMAGE","sub_type":"CONVERT_TO_SHIELD","value":0.4,"log_action":"BLOOD_STAINED_ROBE","message":"absorbs {mitigated}"}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid MITIGATE_DAMAGE declaration rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD_TRIGGER","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"MITIGATE_DAMAGE","sub_type":"REDUCE","value":0.4}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_TARGET","trigger":"BEFORE_HIT","target":{"selector":"TARGET"},"operations":[{"op":"MITIGATE_DAMAGE","sub_type":"REDUCE","value":0.4}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_RATIO","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"MITIGATE_DAMAGE","sub_type":"REDUCE","value":1.1}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_CONDITION","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"conditions":[{"field":"HP_PERCENT","operator":"LTE","value":0.5}],"operations":[{"op":"MITIGATE_DAMAGE","sub_type":"REDUCE","value":0.4}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid MITIGATE_DAMAGE declaration accepted: %s", raw)
		}
	}
}

func TestPersistentStackingStatusOnlyAcceptsConsumedShape(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"PERSISTENT_STATUS","trigger":"ON_HIT","target":{"selector":"TARGET"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"sinh_khi_tan","max_stacks":2,"duration_hours":5,"dispel_rule":"DECREMENT_ONE","pvp_only":true,"tier_scaling":[{"tier":1,"stat_percent":-10},{"tier":2,"stat_percent":-25}]}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid persistent status declaration rejected: %+v", issues)
	}
	invalid := []string{
		`{"schema_version":2,"blocks":[{"id":"PERSISTENT_STATUS","trigger":"BEFORE_HIT","target":{"selector":"TARGET"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"sinh_khi_tan","max_stacks":5}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"PERSISTENT_STATUS","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"sinh_khi_tan","max_stacks":5,"pvp_only":true}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"PERSISTENT_STATUS","trigger":"ON_HIT","target":{"selector":"TARGET"},"conditions":[{"field":"RANK","operator":"GTE","value":1}],"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"sinh_khi_tan","max_stacks":5}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"PERSISTENT_STATUS","trigger":"ON_HIT","target":{"selector":"SELF"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"sinh_khi_tan","max_stacks":5,"pvp_only":true}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"PERSISTENT_STATUS","trigger":"ON_HIT","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"sinh_khi_tan","max_stacks":5,"pvp_only":true}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"PERSISTENT_STATUS","trigger":"ON_HIT","target":{"selector":"TARGET"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"sinh_khi_tan","max_stacks":5,"pvp_only":false}]}]}`,
	}
	for _, raw := range invalid {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("unconsumed persistent status shape accepted: %s", raw)
		}
	}
}

func TestValidatePersistentStatusKeysRequiresRegistryMembership(t *testing.T) {
	raw := []byte(`{"schema_version":2,"blocks":[{"id":"PERSISTENT_STATUS","trigger":"ON_HIT","target":{"selector":"TARGET"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"missing_status"}]}]}`)
	issues := ValidatePersistentStatusKeys(raw, map[string]bool{"sinh_khi_tan": true}, Location{Source: "gu", Key: "GU_TEST"})
	if len(issues) != 1 || issues[0].Code != "unknown_status_key" || issues[0].BlockID != "PERSISTENT_STATUS" {
		t.Fatalf("unknown status key validation issues = %+v", issues)
	}
	issues = ValidatePersistentStatusKeys(raw, map[string]bool{"missing_status": true}, Location{Source: "gu", Key: "GU_TEST"})
	if len(issues) != 0 {
		t.Fatalf("registered status key was rejected: %+v", issues)
	}
}

func TestValidateRevokeStatusIsBattleStartSelfOnly(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"REMOVE_WORLD_DEBUFF","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"REVOKE_STATUS","status_id":"regional_wall_suppression"}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid persistent status revoke rejected: %+v", issues)
	}
	invalid := []string{
		`{"schema_version":2,"blocks":[{"id":"REMOVE_WORLD_DEBUFF","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"REVOKE_STATUS","status_id":"regional_wall_suppression"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"REMOVE_WORLD_DEBUFF","trigger":"ON_BATTLE_START","target":{"selector":"TARGET"},"operations":[{"op":"REVOKE_STATUS","status_id":"regional_wall_suppression"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"REMOVE_WORLD_DEBUFF","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"REVOKE_STATUS","status_id":"Regional Wall"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"REMOVE_WORLD_DEBUFF","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"conditions":[{"field":"RANK","operator":"GTE","value":1}],"operations":[{"op":"REVOKE_STATUS","status_id":"regional_wall_suppression"}]}]}`,
	}
	for _, raw := range invalid {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid persistent status revoke accepted: %s", raw)
		}
	}
}

func TestValidateCounterThresholdHitRuleRequiresExecutableConfiguration(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"SAT_KHI","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"COUNTER_THRESHOLD","params":{"key":"sat_khi_level","threshold":10,"guaranteed":true,"bypass_shield":true,"bypass_immunity":true}}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid counter threshold hit rule rejected: %+v", issues)
	}
	invalid := []string{
		`{"schema_version":2,"blocks":[{"id":"SAT_KHI","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"COUNTER_THRESHOLD","params":{"key":"sat_khi_level","threshold":10,"guaranteed":true}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"SAT_KHI","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"COUNTER_THRESHOLD","params":{"key":"sat_khi_level","threshold":10}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"SAT_KHI","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"COUNTER_THRESHOLD","params":{"key":"sat_khi_level","threshold":1.5,"guaranteed":true}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"SAT_KHI","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"COUNTER_THRESHOLD","params":{"key":"sat_khi_level","threshold":10,"guaranteed":true,"silent_noop":true}}]}]}`,
	}
	for _, raw := range invalid {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid counter threshold hit rule accepted: %s", raw)
		}
	}
}

func TestValidateStackingStatusRequiresCompleteTierTable(t *testing.T) {
	validTiers := []StackingStatusTierValue{{Tier: 1, StatPercent: -10}, {Tier: 2, StatPercent: -25}}
	if err := ValidateStackingStatus("sinh_khi_tan", 2, 5, "DECREMENT_ONE", true, validTiers); err != nil {
		t.Fatalf("valid stacking status config rejected: %v", err)
	}
	invalid := []struct {
		name  string
		tiers []StackingStatusTierValue
	}{
		{name: "missing tier", tiers: validTiers[:1]},
		{name: "duplicate tier", tiers: []StackingStatusTierValue{{Tier: 1, StatPercent: -10}, {Tier: 1, StatPercent: -25}}},
		{name: "positive severity", tiers: []StackingStatusTierValue{{Tier: 1, StatPercent: 10}, {Tier: 2, StatPercent: -25}}},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateStackingStatus("sinh_khi_tan", 2, 5, "DECREMENT_ONE", true, tc.tiers); err == nil {
				t.Fatal("invalid tier table accepted")
			}
		})
	}
}

func TestHPCostAcceptsHistoricalPreRiderPhase(t *testing.T) {
	raw := `{"schema_version":2,"blocks":[{"id":"BLOOD_SACRIFICE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"HP_COST","sub_type":"CURRENT_HP_PERCENT","target":{"selector":"SELF"},"value":0.15,"min_cost":1,"min_remaining":1,"phase":"PRE_RIDER","log_action":"BLOOD_SACRIFICE","message":"{actor} sacrifices {actual_cost} HP"}]}]}`
	if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) != 0 {
		t.Fatalf("PRE_RIDER HP_COST rejected: %+v", issues)
	}
}

func TestBlockV2ValidatesLateConditionalPassiveOperations(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"PASSION","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"conditions":[{"field":"HP_PERCENT","operator":"LTE","value":0.35}],"operations":[{"op":"DAMAGE_MODIFIER","sub_type":"MISSING_HP","value":1.5,"per_missing_hp":0,"phase":"LATE_RIDER"},{"op":"CLEANSE","sub_type":"STATUS","status_id":"STUN","phase":"LATE_RIDER"}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid conditional passive rejected: %+v", issues)
	}
	invalid := []string{
		`{"schema_version":2,"blocks":[{"id":"PASSION","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"conditions":[{"field":"HP_PERCENT","operator":"LTE","value":0.35}],"operations":[{"op":"DAMAGE_MODIFIER","sub_type":"MISSING_HP","value":1.5,"per_missing_hp":0}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"CLEANSE","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"operations":[{"op":"CLEANSE","sub_type":"STATUS","status_id":"STUN"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"CLEANSE","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"CLEANSE","sub_type":"STATUS","status_id":"STUN","phase":"LATE_RIDER"}]}]}`,
	}
	for _, raw := range invalid {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid passive phase accepted: %s", raw)
		}
	}
}

func TestBlockV2DispelAllBuffsDoesNotRequireLimit(t *testing.T) {
	protection := `{"schema_version":2,"blocks":[{"id":"FATE_VANQUISHED","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DISPEL","sub_type":"PROTECTION","log_action":"FATE_VANQUISHED","message":"{actor} breaks {target}'s protections and removes {shield_stripped} shield."}]}]}`
	if issues := ValidateBlockV2([]byte(protection), Location{}); len(issues) != 0 {
		t.Fatalf("valid DISPEL PROTECTION rejected: %+v", issues)
	}
	valid := `{"schema_version":2,"blocks":[{"id":"RULE_IMPOSITION","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DISPEL","sub_type":"ALL_BUFFS"}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid DISPEL ALL_BUFFS rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD_DISPEL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DISPEL","sub_type":"BUFFS"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_DISPEL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DISPEL","sub_type":"ALL_BUFFS","value":2}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_DISPEL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DISPEL","sub_type":"PROTECTION","value":1}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid DISPEL accepted: %s", raw)
		}
	}
}

func TestBlockV2CounterStatModifierRequiresValidParameters(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"STACK_STATS","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","sub_type":"ADD_COUNTER_DELTA","stat":"ATTACK","value":0,"key":"stacks","per_stack":5}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid counter stat modifier rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"STACK_STATS","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","sub_type":"ADD_COUNTER_DELTA","stat":"ATTACK","value":0,"key":"stacks","per_stack":0}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"STACK_STATS","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","sub_type":"ADD_COUNTER_DELTA","stat":"LUCK","value":0,"key":"stacks","per_stack":5}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"STACK_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","sub_type":"ADD_COUNTER_DELTA","stat":"ATTACK","value":0,"key":"stacks","per_stack":5}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid counter stat modifier accepted: %s", raw)
		}
	}
}

func TestBlockV2ThresholdRecoveryRequiresBoundedValidConfiguration(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"RECOVERY","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"THRESHOLD_RECOVERY","hp_threshold":0.3,"heal_percent":0.4,"max_uses":1,"cleanse_debuffs":true,"log_action":"QI_JUE_RETURN_LIFE","message":"{actor} heals {heal_amount}"}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid threshold recovery rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"RECOVERY","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"THRESHOLD_RECOVERY","hp_threshold":0,"heal_percent":0.4,"max_uses":1,"cleanse_debuffs":true,"log_action":"QI_JUE_RETURN_LIFE","message":"recover"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"RECOVERY","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"THRESHOLD_RECOVERY","hp_threshold":0.3,"heal_percent":0.4,"max_uses":0,"cleanse_debuffs":true,"log_action":"QI_JUE_RETURN_LIFE","message":"recover"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"RECOVERY","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"THRESHOLD_RECOVERY","hp_threshold":0.3,"heal_percent":0.4,"max_uses":1,"cleanse_debuffs":true,"log_action":"QI_JUE_RETURN_LIFE","message":"recover"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid threshold recovery accepted: %s", raw)
		}
	}
}

func TestBlockV2SetRuleRequiresKnownRuleAndBoolean(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"IMMORTAL","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"SET_RULE","sub_type":"IMMORTAL","enabled":true}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid SET_RULE rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"IMMORTAL","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"SET_RULE","sub_type":"UNKNOWN","enabled":true}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"IMMORTAL","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"SET_RULE","sub_type":"IMMORTAL"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"IMMORTAL","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"SET_RULE","sub_type":"IMMORTAL","enabled":"yes"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid SET_RULE accepted: %s", raw)
		}
	}
}

func TestBlockV2ActorScaledSummonAndBattleStartRetreat(t *testing.T) {
	validSummon := `{"monster_id":"monster_wolf_leader","stat_source":"ACTOR","hp_scale":0.5,"damage_scale":0.3333333333,"defense_scale":0.5,"rank_scale":false,"name":"Địa Lang Vương","id_suffix":"dialang","log_message":"{actor} triệu hồi {summon} ({hp} HP{dao_note}) lên tiền tuyến để đỡ đòn!"}`
	if err := ValidateSummonParams([]byte(validSummon)); err != nil {
		t.Fatalf("valid actor-scaled summon rejected: %v", err)
	}
	if err := ValidateSummonParams([]byte(`{"monster_id":"monster_wolf_leader","log_message":"{actor} summons {unknown}"}`)); err == nil {
		t.Fatal("summon with unsupported log placeholder was accepted")
	}
	for _, raw := range []string{
		`{"monster_id":"monster_wolf_leader","rank":101}`,
		`{"monster_id":"monster_wolf_leader","id_suffix":"Wolf-1"}`,
		`{"monster_id":"monster_wolf_leader","name":"  "}`,
	} {
		if err := ValidateSummonParams([]byte(raw)); err == nil {
			t.Errorf("invalid summon parameters were accepted: %s", raw)
		}
	}
	validDisplace := `{"schema_version":2,"blocks":[{"id":"MOVE","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"DISPLACE","sub_type":"RETREAT","value":2}]}]}`
	if issues := ValidateBlockV2([]byte(validDisplace), Location{}); len(issues) != 0 {
		t.Fatalf("valid battle-start retreat rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"MOVE","trigger":"ON_BATTLE_START","target":{"selector":"TARGET"},"operations":[{"op":"DISPLACE","sub_type":"RETREAT","value":2}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"MOVE","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"DISPLACE","sub_type":"KNOCKBACK","value":2}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("unsupported battle-start displacement accepted: %s", raw)
		}
	}
}

func TestGameFXMapHooksRequireNumericMaps(t *testing.T) {
	for _, raw := range []string{
		`{"value":100}`,
		`{"value":{}}`,
		`{"value":{"fire":"strong"}}`,
	} {
		if err := ValidateNativeParams("PATH_RESISTANCE_MODIFIER", []byte(raw)); err == nil {
			t.Errorf("invalid path resistance params accepted: %s", raw)
		}
	}
	if err := ValidateNativeParams("PATH_RESISTANCE_MODIFIER", []byte(`{"value":{"fire":0.15}}`)); err != nil {
		t.Fatalf("valid path resistance map rejected: %v", err)
	}
}

func TestBlockV2TransferResourceValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"MANA_STEAL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET","resource":"MANA","value":0.25}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid transfer rejected: %+v", issues)
	}
	hpTransfer := `{"schema_version":2,"blocks":[{"id":"STEAL_LIFE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_MAX_HP","resource":"HP","value":0.15,"min_amount":30,"phase":"LATE_RIDER","log_action":"STEAL_LIFE","message":"{actor} steals {amount} HP from {target}."}]}]}`
	if issues := ValidateBlockV2([]byte(hpTransfer), Location{}); len(issues) != 0 {
		t.Fatalf("valid HP transfer rejected: %+v", issues)
	}
	shieldAbsorption := `{"schema_version":2,"blocks":[{"id":"DEVOUR","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET_SHIELD","resource":"HP","value":0.5,"phase":"EARLY_RIDER","log_action":"DEVOUR_HEAVEN_EARTH","message":"{actor} absorbs {amount} shield from {target}."}]}]}`
	if issues := ValidateBlockV2([]byte(shieldAbsorption), Location{}); len(issues) != 0 {
		t.Fatalf("valid shield absorption rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD_RESOURCE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET","resource":"SOUL","value":0.25}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_FRACTION","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET","resource":"MANA","value":0}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_TARGET","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET","resource":"MANA","value":0.25}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_TRIGGER","trigger":"ON_HIT","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET","resource":"MANA","value":0.25}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_HP_PHASE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_MAX_HP","resource":"HP","value":0.15,"min_amount":30,"phase":"EARLY_RIDER","log_action":"STEAL_LIFE","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_HP_PAIR","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET","resource":"HP","value":0.15}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_SHIELD_PHASE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET_SHIELD","resource":"HP","value":0.5,"phase":"LATE_RIDER","log_action":"DEVOUR_HEAVEN_EARTH","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_HP_CONDITION","trigger":"ON_ACTION","target":{"selector":"TARGET"},"conditions":[{"field":"RANK","operator":"GTE","value":1}],"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_MAX_HP","resource":"HP","value":0.15,"min_amount":30,"phase":"LATE_RIDER","log_action":"STEAL_LIFE","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_MANA_MESSAGE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_TARGET","resource":"MANA","value":0.25,"message":123}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_NULL_PHASE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_MAX_HP","resource":"HP","value":0.15,"min_amount":30,"phase":null,"log_action":"STEAL_LIFE","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_NULL_MIN","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TRANSFER_RESOURCE","sub_type":"PERCENT_OF_MAX_HP","resource":"HP","value":0.15,"min_amount":null,"phase":"LATE_RIDER","log_action":"STEAL_LIFE","message":"x"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid transfer accepted: %s", raw)
		}
	}
}

func TestBlockV2DeductionFinishChanceValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"DEDUCTION_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{"base_chance":0.25,"per_rank_step":0.05,"max_chance":0.95}}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid deduction chance rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{"base_chance":1.1}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{"min_chance":0.8,"max_chance":0.2}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{"chance":0.4}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{"base_chance":null}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_CHANCE","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"conditions":[{"field":"RANK","operator":"GTE","value":2}],"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{}}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid deduction chance accepted: %s", raw)
		}
	}
}

func TestBlockV2ValidatesNotifyMessage(t *testing.T) {
	valid := []byte(`{"schema_version":2,"blocks":[{"id":"RECORD_MESSAGE","trigger":"ON_USE","target":{"selector":"SELF"},"operations":[{"op":"NOTIFY","message":"{gu} records the scene."}]}]}`)
	if issues := ValidateBlockV2(valid, Location{Source: "gu"}); len(issues) != 0 {
		t.Fatalf("valid NOTIFY rejected: %+v", issues)
	}
	combat := []byte(`{"schema_version":2,"blocks":[{"id":"COMBAT_LOG","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"NOTIFY","message":"{actor} activates the effect.","log_action":"EFFECT_NOTIFY"}]}]}`)
	if issues := ValidateBlockV2(combat, Location{Source: "gu"}); len(issues) != 0 {
		t.Fatalf("combat NOTIFY rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"RECORD_MESSAGE","trigger":"ON_USE","target":{"selector":"SELF"},"operations":[{"op":"NOTIFY","message":" "}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"RECORD_MESSAGE","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"NOTIFY","message":"text"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"RECORD_MESSAGE","trigger":"ON_USE","target":{"selector":"TARGET"},"operations":[{"op":"NOTIFY","message":"text"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{Source: "gu"}); len(issues) == 0 {
			t.Errorf("invalid NOTIFY accepted: %s", raw)
		}
	}
}

func TestBlockV2ValidatesDirectStatusLogOverrides(t *testing.T) {
	valid := []byte(`{"schema_version":2,"blocks":[{"id":"BURN","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STATUS","status_id":"BURN","value":15,"turns":3,"chance":1,"application_mode":"DIRECT","phase":"PRE_DAMAGE","log_action":"RAGING_ANGRY_BIRD","message":"{actor} thiêu đốt {target}."},{"op":"STATUS","status_id":"BURN","turns":3,"application_mode":"DIRECT","suppress_log":true}]}]}`)
	if issues := ValidateBlockV2(valid, Location{}); len(issues) != 0 {
		t.Fatalf("valid direct status log override rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BURN","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STATUS","status_id":"BURN","turns":3,"chance":1,"log_action":"lowercase"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BURN","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STATUS","status_id":"BURN","turns":3,"chance":1,"message":"{target} burns."}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BURN","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STATUS","status_id":"BURN","turns":3,"suppress_log":"true"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BURN","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STATUS","status_id":"BURN","turns":3,"suppress_log":true}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid status log override accepted: %s", raw)
		}
	}
}

func TestBlockV2AcceptsPreChainStatusAndNotification(t *testing.T) {
	raw := []byte(`{"schema_version":2,"blocks":[{"id":"LIGHTNING","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.3,"phase":"PRE_CHAIN"},{"op":"STATUS","status_id":"STUN","turns":1,"chance":0.4,"application_mode":"DIRECT","phase":"PRE_CHAIN","suppress_log":true},{"op":"NOTIFY","target":{"selector":"SELF"},"phase":"PRE_CHAIN","log_action":"LIGHTNING_STUN_BURST","message":"{actor} strikes {target}; 40% chance to stun."}]}]}`)
	if issues := ValidateBlockV2(raw, Location{}); len(issues) != 0 {
		t.Fatalf("valid pre-chain status and notification rejected: %+v", issues)
	}
}

func TestBlockV2ValidatesHitRulePhaseAndTarget(t *testing.T) {
	validMultiHit := `{"schema_version":2,"blocks":[{"id":"TRIPLE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"HIT_RULE","sub_type":"MULTI_HIT","params":{"count":3}}]}]}`
	if issues := ValidateBlockV2([]byte(validMultiHit), Location{}); len(issues) != 0 {
		t.Fatalf("valid MULTI_HIT rejected: %+v", issues)
	}
	for _, subtype := range []string{"FIRST_TURN_EVADE", "LETHAL_LUCK_WARD"} {
		params := ""
		if subtype == "LETHAL_LUCK_WARD" {
			params = `,"params":{"chance":0.25,"shield_percent":0.35,"shield_min":100}`
		}
		valid := `{"schema_version":2,"blocks":[{"id":"DEFENSE","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"` + subtype + `"` + params + `}]}]}`
		if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
			t.Fatalf("valid defensive HIT_RULE %s rejected: %+v", subtype, issues)
		}
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"FIRST_TURN_EVADE"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"BEFORE_HIT","target":{"selector":"TARGET"},"operations":[{"op":"HIT_RULE","sub_type":"FIRST_TURN_EVADE"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"GUARANTEED"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"HIT_RULE","sub_type":"LETHAL_LUCK_WARD"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"HIT_RULE","sub_type":"LETHAL_LUCK_WARD","params":{"chance":2,"shield_percent":0.35,"shield_min":100}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"HIT_RULE","sub_type":"MULTI_HIT","params":{"count":1}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"BEFORE_HIT","target":{"selector":"TARGET"},"operations":[{"op":"HIT_RULE","sub_type":"MULTI_HIT","params":{"count":3}}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid HIT_RULE trigger/target combination accepted: %s", raw)
		}
	}
}

func TestBlockV2RejectsNativeMechanicTriggerWithoutConsumer(t *testing.T) {
	raw := `{"schema_version":2,"blocks":[{"id":"BAD_TRIGGER","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"MANA_EAT","params":{}}]}]}`
	if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
		t.Fatal("MANA_EAT BEFORE_HIT must be rejected because the simulator has no consumer")
	}
}

func TestBlockV2ValidatesTurnOrderRules(t *testing.T) {
	valid := []string{
		`{"schema_version":2,"blocks":[{"id":"SEIZE","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"TURN_ORDER_RULE","sub_type":"SEIZE_INITIATIVE","params":{"lifespan_ratio":3}}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"SUPPRESS","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"TURN_ORDER_RULE","sub_type":"SUPPRESS_ENEMY_PRIORITY"}]}]}`,
	}
	for _, raw := range valid {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) != 0 {
			t.Errorf("valid turn-order rule rejected: %+v", issues)
		}
	}
	invalid := []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"TURN_ORDER_RULE","sub_type":"SUPPRESS_ENEMY_PRIORITY"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_BATTLE_START","target":{"selector":"TARGET"},"operations":[{"op":"TURN_ORDER_RULE","sub_type":"SUPPRESS_ENEMY_PRIORITY"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"TURN_ORDER_RULE","sub_type":"SEIZE_INITIATIVE"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_BATTLE_START","target":{"selector":"SELF"},"operations":[{"op":"TURN_ORDER_RULE","sub_type":"SEIZE_INITIATIVE","params":{"lifespan_ratio":0}}]}]}`,
	}
	for _, raw := range invalid {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid turn-order rule accepted: %s", raw)
		}
	}
}

func TestCapabilityCatalogMatchesCapabilityHooks(t *testing.T) {
	for code, hook := range NativeHooks {
		if hook.Domain == "capability" {
			if !CapabilityCatalog[code] || !hook.Triggers["CAPABILITY"] {
				t.Errorf("capability hook %q is not represented by the typed catalog", code)
			}
		}
	}
	for code := range CapabilityCatalog {
		hook, ok := NativeHooks[code]
		if !ok || hook.Domain != "capability" || !hook.Triggers["CAPABILITY"] {
			t.Errorf("typed capability %q has no matching capability hook", code)
		}
	}
}

func TestArenaIdentityMaskRejectedWithoutConsumer(t *testing.T) {
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"MASK","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"GRANT_CAPABILITY","sub_type":"ARENA_IDENTITY_MASK"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"MASK","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"ARENA_IDENTITY_MASK","params":{}}]}]}`,
	} {
		issues := ValidateBlockV2([]byte(raw), Location{})
		if len(issues) == 0 {
			t.Fatalf("unimplemented capability accepted: %s", raw)
		}
	}
}

func TestGreatNightRejectsUnimplementedActionTrigger(t *testing.T) {
	raw := `{"schema_version":2,"blocks":[{"id":"GREAT_NIGHT","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"UNIQUE","code":"GREAT_NIGHT","params":{}}]}]}`
	if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
		t.Fatal("GREAT_NIGHT ON_ACTION accepted without an ON_ACTION consumer")
	}
}

func TestDamageModifierTargetHasStatusValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"IGNITE_BURST","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1},{"op":"DAMAGE_MODIFIER","sub_type":"TARGET_HAS_STATUS","value":1.6,"status_id":"BURN"}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid status-gated damage modifier rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","value":1},{"op":"DAMAGE_MODIFIER","sub_type":"TARGET_HAS_STATUS","value":1.6}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","value":1},{"op":"DAMAGE_MODIFIER","sub_type":"TARGET_HAS_STATUS","value":1.6,"status_id":"NOT_A_STATUS"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid status-gated damage modifier accepted: %s", raw)
		}
	}
}

func TestTargetDamageModifierValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"QI_SEA","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"TARGET_DAMAGE_MODIFIER","sub_type":"REDUCE","value":0.3,"min_multiplier":0.2,"log_action":"QI_SEA","message":"{actor} reduces enemy damage."}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid target damage modifier rejected: %+v", issues)
	}
	conditional := `{"schema_version":2,"blocks":[{"id":"QI_SEA","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"conditions":[{"field":"TARGET_HP_PERCENT","operator":"LTE","value":0.5}],"operations":[{"op":"TARGET_DAMAGE_MODIFIER","sub_type":"REDUCE","value":0.3,"min_multiplier":0.2}]}]}`
	if issues := ValidateBlockV2([]byte(conditional), Location{}); len(issues) != 0 {
		t.Fatalf("conditional target damage modifier rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"TARGET_DAMAGE_MODIFIER","sub_type":"REDUCE","value":0.3,"min_multiplier":0.2}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TARGET_DAMAGE_MODIFIER","sub_type":"REDUCE","value":1.2,"min_multiplier":0.2}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"TARGET_DAMAGE_MODIFIER","sub_type":"REDUCE","value":0.3,"min_multiplier":1.2}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid target damage modifier accepted: %s", raw)
		}
	}
}

func TestBonusStrikeValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"PHANTOM_STRIKE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"BONUS_STRIKE","chance":0.45,"value":0.75,"min_damage":50,"log_action":"BEAST_PHANTOM_STRIKE","message":"{actor} hits {target} for {damage}."}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid bonus strike rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"BONUS_STRIKE","chance":0,"value":0.75,"min_damage":50,"log_action":"BEAST_PHANTOM_STRIKE","message":"hit"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"BONUS_STRIKE","chance":0.45,"value":0,"min_damage":50,"log_action":"BEAST_PHANTOM_STRIKE","message":"hit"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"BONUS_STRIKE","chance":0.45,"value":0.75,"min_damage":50,"log_action":"BEAST_PHANTOM_STRIKE","message":"hit"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid bonus strike accepted: %s", raw)
		}
	}
}

func TestChainDamageValidation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"CHAIN","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"CHAIN_DAMAGE","value":0.5,"min_damage":30,"target_count":1,"log_action":"CHAIN_DAMAGE","message":"{actor} chains to {target} for {damage}."}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid chain damage rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"CHAIN_DAMAGE","value":0,"min_damage":30,"target_count":1,"log_action":"CHAIN_DAMAGE","message":"hit"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"CHAIN_DAMAGE","value":0.5,"min_damage":30,"target_count":0,"log_action":"CHAIN_DAMAGE","message":"hit"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"CHAIN_DAMAGE","value":0.5,"min_damage":30,"target_count":1,"log_action":"CHAIN_DAMAGE","message":"hit"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid chain damage accepted: %s", raw)
		}
	}
}

func TestDamageModifierMinionCountRequiresPassiveSelfTarget(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"PACK_FRENZY","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"operations":[{"op":"DAMAGE_MODIFIER","sub_type":"MINION_COUNT","value":1,"per_minion":0.15,"log_action":"PACK_FRENZY","message":"{actor} has {minion_count} minions (+{damage_percent}%)."}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid minion-count passive rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"COMBAT_PASSIVE","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE_MODIFIER","sub_type":"MINION_COUNT","value":1,"per_minion":0.15}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"operations":[{"op":"DAMAGE_MODIFIER","sub_type":"MINION_COUNT","value":1}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"COMBAT_PASSIVE","target":{"selector":"SELF"},"conditions":[{"field":"RANK","operator":"GTE","value":1}],"operations":[{"op":"DAMAGE_MODIFIER","sub_type":"MINION_COUNT","value":1,"per_minion":0.15}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid minion-count modifier accepted: %s", raw)
		}
	}
}

func TestDetonateStatusRequiresCombatTargetAndFormula(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"BLEED_BURST","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DETONATE_STATUS","status_id":"BLEED","per_stack":100,"per_strength":0,"max_hp_percent":0.08,"min_damage":0,"phase":"EARLY_RIDER","log_action":"BLEED_BURST","message":"{actor} detonates {stack_count} stacks on {target} for {damage}."}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid status detonation rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"COMBAT_PASSIVE","target":{"selector":"TARGET"},"operations":[{"op":"DETONATE_STATUS","status_id":"BLEED","per_stack":100,"per_strength":0,"max_hp_percent":0.08,"min_damage":0,"phase":"EARLY_RIDER","log_action":"X","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DETONATE_STATUS","status_id":"UNKNOWN","per_stack":100,"per_strength":0,"max_hp_percent":0.08,"min_damage":0,"phase":"EARLY_RIDER","log_action":"X","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DETONATE_STATUS","status_id":"BLEED","per_stack":0,"per_strength":0,"max_hp_percent":0,"min_damage":0,"phase":"EARLY_RIDER","log_action":"X","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"conditions":[{"field":"RANK","operator":"GTE","value":1}],"operations":[{"op":"DETONATE_STATUS","status_id":"BLEED","per_stack":100,"per_strength":0,"max_hp_percent":0.08,"min_damage":0,"phase":"EARLY_RIDER","log_action":"X","message":"x"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid status detonation accepted: %s", raw)
		}
	}
}

func TestHPCostRequiresBoundedSelfCombatConfiguration(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"BLOOD_DEMON_DISINTEGRATION","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"HP_COST","sub_type":"CURRENT_HP_PERCENT","value":0.2,"min_cost":10,"min_remaining":1,"log_action":"BLOOD_DEMON_DISINTEGRATION","message":"{actor} sacrifices {cost} HP."}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid HP cost rejected: %+v", issues)
	}
	earlyPhase := `{"schema_version":2,"blocks":[{"id":"GU_DETONATION","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"HP_COST","sub_type":"CURRENT_HP_PERCENT","value":0.05,"min_cost":1,"min_remaining":1,"phase":"EARLY_RIDER","log_action":"GU_DETONATION","message":"{actor}"}]}]}`
	if issues := ValidateBlockV2([]byte(earlyPhase), Location{}); len(issues) != 0 {
		t.Fatalf("valid early HP cost rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"HP_COST","sub_type":"CURRENT_HP_PERCENT","value":0.2,"min_cost":10,"min_remaining":1,"log_action":"X","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"HP_COST","sub_type":"CURRENT_HP_PERCENT","value":0.2,"min_cost":10,"min_remaining":1,"log_action":"X","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"HP_COST","sub_type":"CURRENT_HP_PERCENT","value":1.1,"min_cost":10,"min_remaining":1,"log_action":"X","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"SELF"},"conditions":[{"field":"RANK","operator":"GTE","value":1}],"operations":[{"op":"HP_COST","sub_type":"CURRENT_HP_PERCENT","value":0.2,"min_cost":10,"min_remaining":1,"log_action":"X","message":"x"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"HP_COST","sub_type":"CURRENT_HP_PERCENT","value":0.2,"min_cost":10,"min_remaining":1,"phase":"LATE_RIDER","log_action":"X","message":"x"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid HP cost accepted: %s", raw)
		}
	}
}

func TestGrantBodyModRequiresConsumeAndSelf(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"BODY","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"GRANT_BODY_MOD","sub_type":"SOVEREIGN_IMMORTAL"}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid body modification rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"BODY","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"GRANT_BODY_MOD","sub_type":"SOVEREIGN_IMMORTAL"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BODY","trigger":"ON_CONSUME","target":{"selector":"TARGET"},"operations":[{"op":"GRANT_BODY_MOD","sub_type":"SOVEREIGN_IMMORTAL"}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"BODY","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"GRANT_BODY_MOD","sub_type":"UNKNOWN"}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid body modification accepted: %s", raw)
		}
	}
}

func TestModifyAptitudeRequiresBoundedConsumeOperation(t *testing.T) {
	valid := `{"schema_version":2,"blocks":[{"id":"APTITUDE","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"MODIFY_APTITUDE","sub_type":"ADD_FLAT","value":5}]}]}`
	if issues := ValidateBlockV2([]byte(valid), Location{}); len(issues) != 0 {
		t.Fatalf("valid aptitude operation rejected: %+v", issues)
	}
	for _, raw := range []string{
		`{"schema_version":2,"blocks":[{"id":"APTITUDE","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"MODIFY_APTITUDE","sub_type":"ADD_FLAT","value":5}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"APTITUDE","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"MODIFY_APTITUDE","sub_type":"ADD_FLAT","value":5.5}]}]}`,
		`{"schema_version":2,"blocks":[{"id":"APTITUDE","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"MODIFY_APTITUDE","sub_type":"ADD_FLAT","value":100}]}]}`,
	} {
		if issues := ValidateBlockV2([]byte(raw), Location{}); len(issues) == 0 {
			t.Errorf("invalid aptitude operation accepted: %s", raw)
		}
	}
}

func TestPeriodicResourceRejectsUnsupportedPairs(t *testing.T) {
	tests := []struct {
		name, subType, resource string
		wantValid               bool
	}{
		{name: "mana hourly regen", subType: "REGEN_PER_HOUR", resource: "MANA", wantValid: true},
		{name: "primeval stones worker tick", subType: "GENERATION_WORKER_TICK", resource: "PRIMEVAL_STONE", wantValid: true},
		{name: "no worker mana", subType: "GENERATION_WORKER_TICK", resource: "MANA"},
		{name: "no hourly immortal stones", subType: "REGEN_PER_HOUR", resource: "IMMORTAL_STONE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePeriodicResource(tt.subType, tt.resource)
			if (err == nil) != tt.wantValid {
				t.Fatalf("ValidatePeriodicResource(%q, %q) error = %v, valid want %t", tt.subType, tt.resource, err, tt.wantValid)
			}
		})
	}
}

func TestBlockV2ValidatesConditionValueTypes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{
			name: "race key string",
			raw:  `[{"id":"RACE_GATE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"conditions":[{"field":"TARGET_RACE","operator":"EQ","value":"merman"}],"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1}]}]`,
			want: true,
		},
		{
			name: "race key cannot use numeric value",
			raw:  `[{"id":"RACE_GATE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"conditions":[{"field":"TARGET_RACE","operator":"EQ","value":1}],"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1}]}]`,
		},
		{
			name: "numeric condition cannot use string value",
			raw:  `[{"id":"RANK_GATE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"conditions":[{"field":"RANK","operator":"EQ","value":"six"}],"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1}]}]`,
		},
		{
			name: "race condition only supports equality operators",
			raw:  `[{"id":"RACE_GATE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"conditions":[{"field":"TARGET_RACE","operator":"GT","value":"merman"}],"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1}]}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateBlockV2([]byte(tt.raw), Location{Source: "test"})
			if got := len(issues) == 0; got != tt.want {
				t.Fatalf("ValidateBlockV2(%s) accepted = %t, want %t; issues=%+v", tt.raw, got, tt.want, issues)
			}
		})
	}
}

func TestBlockV2ReportsPreciseOperationLocation(t *testing.T) {
	raw := []byte(`{"schema_version":2,"legacy":{},"blocks":[{"id":"ON_DEATH_TEST","trigger":"ON_DEATH","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"REBORN","params":{"chance":2,"typo":true}}]}]}`)
	issues := ValidateBlockV2(raw, Location{Source: "item", Row: 7, Key: "phoenix"})
	if len(issues) < 3 {
		t.Fatalf("issues=%+v", issues)
	}
	found := false
	for _, issue := range issues {
		if issue.Code == "invalid_param" && issue.Path == "$.blocks[0].operations[0].params.chance" {
			found = true
			if issue.Source != "item" || issue.Row != 7 || issue.Key != "phoenix" || issue.BlockID != "ON_DEATH_TEST" || issue.OperationIndex == nil || *issue.OperationIndex != 0 {
				t.Fatalf("location=%+v", issue)
			}
		}
	}
	if !found {
		t.Fatalf("missing precise param issue: %+v", issues)
	}
}

func TestBlockV2RejectsDuplicateIDsAndUnknownFields(t *testing.T) {
	raw := []byte(`{"schema_version":2,"blocks":[{"id":"SAME","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","sub_type":"ADD_FLAT","stat":"ATTACK","value":1,"unit":"FLAT","stack_policy":"STACK","extra":1}]},{"id":"SAME","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","sub_type":"ADD_FLAT","stat":"ATTACK","value":1,"unit":"FLAT","stack_policy":"STACK"}]}]}`)
	issues := ValidateBlockV2(raw, Location{})
	codes := map[string]bool{}
	for _, i := range issues {
		codes[i.Code] = true
	}
	if !codes["unknown_field"] || !codes["duplicate_block_id"] {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestRankScaledDebuffContract(t *testing.T) {
	valid := []byte(`{"schema_version":2,"blocks":[{"id":"REGIONAL_DEBUFF","trigger":"ON_BATTLE_START","target":{"selector":"TARGET"},"operations":[{"op":"RANK_SCALED_DEBUFF","sub_type":"REGIONAL_WALL","params":{"strength_factor":0.5,"status_id":"regional_wall_suppression","duration_base_minutes":15,"duration_per_rank_minutes":5,"source_name":"Formation"}}]}]}`)
	if issues := ValidateBlockV2(valid, Location{}); len(issues) != 0 {
		t.Fatalf("valid rank-scaled debuff rejected: %+v", issues)
	}
	invalid := []byte(`{"schema_version":2,"blocks":[{"id":"REGIONAL_DEBUFF","trigger":"ON_ACTION","target":{"selector":"TARGET"},"conditions":[{"field":"SELF_HP_PERCENT","operator":"LTE","value":0.5}],"operations":[{"op":"RANK_SCALED_DEBUFF","sub_type":"REGIONAL_WALL","params":{"strength_factor":2,"status_id":"regional_wall_suppression","duration_base_minutes":15,"duration_per_rank_minutes":5,"source_name":"Formation","typo":true}}]}]}`)
	codes := map[string]bool{}
	for _, issue := range ValidateBlockV2(invalid, Location{}) {
		codes[issue.Code] = true
	}
	if !codes["incompatible_trigger_operation"] || !codes["invalid_rank_scaled_debuff_params"] || !codes["unsupported_conditions"] {
		t.Fatalf("invalid rank-scaled debuff was not rejected fail-closed: %+v", codes)
	}
}

func TestGuSealContract(t *testing.T) {
	valid := []byte(`{"schema_version":2,"blocks":[{"id":"FORGET_INTENT_SEAL","trigger":"BEFORE_ENEMY_ATTACK","target":{"selector":"SELF"},"operations":[{"op":"GU_SEAL","sub_type":"REACTIVE","params":{"counter_key":"forget_intent_success","chance":0.4,"turns":1,"max_activations":8,"backfire_at_limit":true,"backfire_turns":1}}]}]}`)
	if issues := ValidateBlockV2(valid, Location{}); len(issues) != 0 {
		t.Fatalf("valid GU_SEAL rejected: %+v", issues)
	}
	invalid := []byte(`{"schema_version":2,"blocks":[{"id":"FORGET_INTENT_SEAL","trigger":"ON_ACTION","target":{"selector":"SELF"},"conditions":[{"field":"SELF_HP_PERCENT","operator":"LTE","value":0.5}],"operations":[{"op":"GU_SEAL","sub_type":"REACTIVE","params":{"counter_key":"Bad Key","chance":1.4,"turns":1,"max_activations":8,"backfire_at_limit":true,"backfire_turns":1,"typo":true}}]}]}`)
	codes := map[string]bool{}
	for _, issue := range ValidateBlockV2(invalid, Location{}) {
		codes[issue.Code] = true
	}
	if !codes["incompatible_trigger_operation"] || !codes["invalid_gu_seal_params"] || !codes["unsupported_conditions"] {
		t.Fatalf("invalid GU_SEAL was not rejected fail-closed: %+v", codes)
	}
}

func TestLuckRouletteParamsAreStrictAndBounded(t *testing.T) {
	valid := []byte(`{"great_chance_percent":15,"curse_great_chance_bonus":50,"medium_range_percent":45,"neutral_range_percent":30,"great_damage_multiplier":3,"medium_damage_multiplier":1.5,"neutral_damage_multiplier":1,"bad_damage_multiplier":0.5,"great_mana_cost_multiplier":1,"bad_current_hp_loss_percent":0.1,"minimum_bad_hp_loss":1,"curse_duration_turns":2}`)
	if err := ValidateNativeParams("LUCK_ROULETTE", valid); err != nil {
		t.Fatalf("valid LUCK_ROULETTE params rejected: %v", err)
	}
	for _, invalid := range [][]byte{
		[]byte(`{"great_chance_percent":15,"curse_great_chance_bonus":50,"medium_range_percent":45,"neutral_range_percent":50,"great_damage_multiplier":3,"medium_damage_multiplier":1.5,"neutral_damage_multiplier":1,"bad_damage_multiplier":0.5,"great_mana_cost_multiplier":1,"bad_current_hp_loss_percent":0.1,"minimum_bad_hp_loss":1,"curse_duration_turns":2}`),
		[]byte(`{"great_chance_percent":15,"curse_great_chance_bonus":50,"medium_range_percent":45,"neutral_range_percent":30,"great_damage_multiplier":3,"medium_damage_multiplier":1.5,"neutral_damage_multiplier":1,"bad_damage_multiplier":0.5,"great_mana_cost_multiplier":1,"bad_current_hp_loss_percent":0.1,"minimum_bad_hp_loss":1,"curse_duration_turns":2,"unexpected":1}`),
	} {
		if err := ValidateNativeParams("LUCK_ROULETTE", invalid); err == nil {
			t.Fatalf("invalid LUCK_ROULETTE params accepted: %s", invalid)
		}
	}
}
