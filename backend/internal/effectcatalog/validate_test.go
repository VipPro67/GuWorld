package effectcatalog

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNormalizeLegacyFlatEffect(t *testing.T) {
	normalized, report, err := NormalizeAndValidate([]byte(`{"damage_flat": 12}`))
	if err != nil {
		t.Fatalf("NormalizeAndValidate() error = %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("unexpected validation errors: %+v", report.Issues)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(normalized, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope) != len(Sections) {
		t.Fatalf("got %d sections, want %d", len(envelope), len(Sections))
	}
	var active map[string]float64
	if err := json.Unmarshal(envelope["active"], &active); err != nil {
		t.Fatal(err)
	}
	if active["damage_flat"] != 12 {
		t.Fatalf("active damage_flat = %v, want 12", active["damage_flat"])
	}
}

func TestValidateRejectsInvalidShapesAndStatuses(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		path string
	}{
		{"invalid json", `{`, "$"},
		{"non object root", `[]`, "$"},
		{"unknown section", `{"combat":{}}`, "$.combat"},
		{"active not object", `{"active":[]}`, "$.active"},
		{"other not array", `{"other":{}}`, "$.other"},
		{"unknown status", `{"active":{"apply_status":"NOPE"}}`, "$.active.apply_status[0].id"},
		{"negative duration", `{"active":{"status_duration_ticks":-1}}`, "$.active.status_duration_ticks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, report, err := NormalizeAndValidate([]byte(tt.raw))
			if err == nil || !report.HasErrors() {
				t.Fatalf("expected validation error, got err=%v report=%+v", err, report)
			}
			if report.Issues[0].Path != tt.path {
				t.Fatalf("first issue path = %q, want %q", report.Issues[0].Path, tt.path)
			}
		})
	}
}

func TestValidateWarnsWhenRuntimeIgnoresKnownLegacyField(t *testing.T) {
	_, report, err := NormalizeAndValidate([]byte(`{"active":{"reveal_all_stealth":true}}`))
	if err != nil {
		t.Fatalf("NormalizeAndValidate() error = %v", err)
	}
	if len(report.Warnings()) != 1 {
		t.Fatalf("warnings = %+v, want one", report.Warnings())
	}
	if report.Warnings()[0].Path != "$.active.reveal_all_stealth" {
		t.Fatalf("warning path = %q", report.Warnings()[0].Path)
	}
}

func TestValidateOtherMechanicShape(t *testing.T) {
	valid := []byte(`{"other":[{"event_trigger":"ON_USE","action_code":"RECORD_MESSAGE","params":{"message":"ok"}}]}`)
	if _, report, err := NormalizeAndValidate(valid); err != nil || report.HasErrors() {
		t.Fatalf("valid mechanic rejected: err=%v report=%+v", err, report)
	}

	invalid := []byte(`{"other":[{"event_trigger":"ON_USE","params":{}}]}`)
	_, report, err := NormalizeAndValidate(invalid)
	if err == nil || !report.HasErrors() {
		t.Fatalf("invalid mechanic accepted: report=%+v", report)
	}
}

func TestValidateApplyStatusAuthoringForms(t *testing.T) {
	valid := []string{
		`{"active":{"apply_status":"BIND","status_duration_ticks":6}}`,                              // legacy string
		`{"active":{"apply_status":{"id":"BIND","turns":6}}}`,                                       // single object
		`{"active":{"apply_status":[{"id":"BLEED","value":10,"turns":3}]}}`,                         // array of one
		`{"active":{"apply_status":[{"id":"BLEED","value":10,"turns":3},{"id":"BIND","turns":2}]}}`, // assembled
	}
	for _, raw := range valid {
		if _, report, err := NormalizeAndValidate([]byte(raw)); err != nil || report.HasErrors() {
			t.Fatalf("valid apply_status rejected: %s: err=%v report=%+v", raw, err, report.Issues)
		}
	}

	invalid := []struct{ raw, path string }{
		{`{"active":{"apply_status":42}}`, "$.active.apply_status"},
		{`{"active":{"apply_status":{}}}`, "$.active.apply_status.id"},
		{`{"active":{"apply_status":[]}}`, "$.active.apply_status"},
		{`{"active":{"apply_status":[{"id":"NOPE"}]}}`, "$.active.apply_status[0].id"},
		{`{"active":{"apply_status":[{"id":"bind"}]}}`, "$.active.apply_status[0].id"},
		{`{"active":{"apply_status":[{"id":"BLEED","value":-1}]}}`, "$.active.apply_status[0].value"},
		{`{"active":{"apply_status":[{"id":"BLEED","turns":"many"}]}}`, "$.active.apply_status[0].turns"},
	}
	for _, tt := range invalid {
		_, report, err := NormalizeAndValidate([]byte(tt.raw))
		if err == nil || !report.HasErrors() {
			t.Fatalf("invalid apply_status accepted: %s", tt.raw)
		}
		if report.Issues[0].Path != tt.path {
			t.Fatalf("first issue path = %q, want %q (raw %s)", report.Issues[0].Path, tt.path, tt.raw)
		}
	}
}

func TestLegacyCatalogStatusIsPreservedWithWarning(t *testing.T) {
	normalized, report, err := NormalizeAndValidate([]byte(`{"active":{"apply_status":"ROOT","status_duration_ticks":2}}`))
	if err != nil {
		t.Fatalf("legacy catalog status rejected: %v", err)
	}
	if !bytes.Contains(normalized, []byte(`"apply_status":"ROOT"`)) {
		t.Fatalf("normalized effect lost legacy status: %s", normalized)
	}
	if len(report.Warnings()) != 1 || report.Warnings()[0].Code != "status_migrate" {
		t.Fatalf("warnings = %+v", report.Warnings())
	}
}

func TestValidateRejectsWrongTypedFieldValues(t *testing.T) {
	for _, raw := range []string{
		`{"active":{"damage_flat":"large"}}`,
		`{"active":{"curve_attack":1}}`,
		`{"active":{"status_duration_ticks":1.5}}`,
		`{"active":{"lifesteal_rate":1.5}}`,
		`{"active":{"target_selector":" row "}}`,
	} {
		if _, report, err := NormalizeAndValidate([]byte(raw)); err == nil || !report.HasErrors() {
			t.Errorf("invalid typed effect accepted: %s", raw)
		}
	}
}

func TestValidateRejectsUnknownMechanicCatalogValues(t *testing.T) {
	raw := []byte(`{"other":[{"event_trigger":"ON_TYP0","action_code":"MISSPELLED","params":{}}]}`)
	_, report, err := NormalizeAndValidate(raw)
	if err == nil || !report.HasErrors() {
		t.Fatalf("unclassified mechanic accepted: %+v", report)
	}
}

func TestValidateRejectsInvalidSupportedMechanicParams(t *testing.T) {
	_, report, err := NormalizeAndValidate([]byte(`{"other":[{"event_trigger":"ON_DEATH","action_code":"REBORN","params":{"chance":1.5}}]}`))
	if err == nil || !report.HasErrors() {
		t.Fatalf("expected out-of-range REBORN chance to fail, err=%v report=%+v", err, report)
	}
}

func TestValidateAcceptsExtendedRebornParams(t *testing.T) {
	raw := []byte(`{"other":[{"event_trigger":"ON_DEATH","action_code":"REBORN","params":{"chance":0.5,"mode":"in_place","max_uses":5,"turn1_guaranteed":true,"revive_hp_percent":0.5,"clear_debuffs":true,"empower_attack_percent":0.1,"empower_speed_percent":0.1,"empower_shield_percent":0.1,"empower_turns":100}}]}`)
	if _, _, err := NormalizeAndValidate(raw); err != nil {
		t.Fatalf("extended REBORN params rejected: %v", err)
	}
	// Legacy minimal form keeps working (defaults at runtime).
	if _, _, err := NormalizeAndValidate([]byte(`{"other":[{"event_trigger":"ON_DEATH","action_code":"REBORN","params":{"chance":0.7}}]}`)); err != nil {
		t.Fatalf("legacy REBORN params rejected: %v", err)
	}
	for _, bad := range []string{
		`{"other":[{"event_trigger":"ON_DEATH","action_code":"REBORN","params":{"chance":0.5,"mode":"rewind_too_far"}}]}`,
		`{"other":[{"event_trigger":"ON_DEATH","action_code":"REBORN","params":{"chance":0.5,"max_uses":99}}]}`,
	} {
		if _, report, err := NormalizeAndValidate([]byte(bad)); err == nil || !report.HasErrors() {
			t.Fatalf("invalid REBORN params accepted: %s", bad)
		}
	}
}

func TestValidateAcceptsForgetInterruptConfuse(t *testing.T) {
	good := []string{
		`{"other":[{"event_trigger":"BEFORE_ENEMY_ATTACK","action_code":"FORGET_INTENT_INTERRUPT","params":{"chance":0.4,"disable_target_gu_turns":1,"max_successful_activations":8,"backfire_after_max":true}}]}`,
		`{"other":[{"event_trigger":"BEFORE_ENEMY_ATTACK","action_code":"FORGET_WILL_CONFUSE","params":{"activation_chance":0.35,"hit_user_chance":0.15,"hit_self_chance":0.35,"miss_chance":0.5,"backfire_chance":0.2,"backfire_disable_turns":2}}]}`,
		`{"active":{"damage_hp_percent":0.4,"shield_flat_reduce_percent":0.4,"disable_gu_next_turn":true,"apply_status":"FORGET_EGO","status_duration_ticks":1}}`,
		`{"passive":{"anti_heaven_damage_reduction_percent":0.65,"forget_heaven_protection":true}}`,
		`{"passive":{"anti_dream_damage_reduction_percent":0.5,"immune_dream_status":true}}`,
	}
	for _, raw := range good {
		if _, _, err := NormalizeAndValidate([]byte(raw)); err != nil {
			t.Fatalf("forget effect rejected: %s: %v", raw, err)
		}
	}
}

func TestApplyStatusChanceRange(t *testing.T) {
	if _, _, err := NormalizeAndValidate([]byte(`{"active":{"apply_status":[{"id":"BLEED","turns":2,"chance":0.5}]}}`)); err != nil {
		t.Fatalf("valid chance rejected: %v", err)
	}
	if _, report, err := NormalizeAndValidate([]byte(`{"active":{"apply_status":[{"id":"BLEED","turns":2,"chance":1.5}]}}`)); err == nil || !report.HasErrors() {
		t.Fatal("chance > 1 accepted")
	}
}

func TestValidateBlocks(t *testing.T) {
	valid := []byte(`{
		"blocks": [
			{
				"trigger": "ON_ACTION",
				"target": { "selector": "TARGET", "bypass_frontline": true },
				"operations": [
					{ "op": "DAMAGE", "sub_type": "FLAT", "value": 150.0 },
					{ "op": "STATUS", "status_id": "BLEED", "value": 25.0, "turns": 3, "chance": 0.6 }
				]
			}
		]
	}`)
	if _, _, err := NormalizeAndValidate(valid); err != nil {
		t.Fatalf("valid blocks rejected: %v", err)
	}

	invalidCases := []struct {
		name string
		raw  string
		path string
	}{
		{"blocks not array", `{"blocks": {}}`, "$.blocks"},
		{"missing trigger", `{"blocks": [{"target": {}, "operations": [{"op": "DAMAGE"}]}]}`, "$.blocks[0].trigger"},
		{"invalid trigger", `{"blocks": [{"trigger": "FOOBAR", "target": {}, "operations": [{"op": "DAMAGE"}]}]}`, "$.blocks[0].trigger"},
		{"invalid target", `{"blocks": [{"trigger": "ON_ACTION", "target": [], "operations": [{"op": "DAMAGE"}]}]}`, "$.blocks[0].target"},
		{"empty operations", `{"blocks": [{"trigger": "ON_ACTION", "target": {}, "operations": []}]}`, "$.blocks[0].operations"},
		{"missing op type", `{"blocks": [{"trigger": "ON_ACTION", "target": {}, "operations": [{}]}]}`, "$.blocks[0].operations[0].op"},
		{"invalid op type", `{"blocks": [{"trigger": "ON_ACTION", "target": {}, "operations": [{"op": "MAGIC"}]}]}`, "$.blocks[0].operations[0].op"},
		{"status missing id", `{"blocks": [{"trigger": "ON_ACTION", "target": {}, "operations": [{"op": "STATUS"}]}]}`, "$.blocks[0].operations[0].status_id"},
		{"unknown status id", `{"blocks": [{"trigger": "ON_ACTION", "target": {}, "operations": [{"op": "STATUS", "status_id": "NOT_A_STATUS"}]}]}`, "$.blocks[0].operations[0].status_id"},
		{"negative op value", `{"blocks": [{"trigger": "ON_ACTION", "target": {}, "operations": [{"op": "DAMAGE", "value": -10}]}]}`, "$.blocks[0].operations[0].value"},
		{"chance out of range", `{"blocks": [{"trigger": "ON_ACTION", "target": {}, "operations": [{"op": "STATUS", "status_id": "BLEED", "chance": 1.5}]}]}`, "$.blocks[0].operations[0].chance"},
	}

	for _, tt := range invalidCases {
		t.Run(tt.name, func(t *testing.T) {
			_, report, err := NormalizeAndValidate([]byte(tt.raw))
			if err == nil || !report.HasErrors() {
				t.Fatalf("expected error for %s, got none", tt.name)
			}
			if report.Issues[0].Path != tt.path {
				t.Fatalf("%s: expected path %q, got %q (issue: %+v)", tt.name, tt.path, report.Issues[0].Path, report.Issues[0])
			}
		})
	}
}
