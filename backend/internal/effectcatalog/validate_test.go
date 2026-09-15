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
		{"unknown status", `{"active":{"apply_status":"NOPE"}}`, "$.active.apply_status"},
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
