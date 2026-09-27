package effectmigration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFormatSpecificConverters(t *testing.T) {
	tests := []struct {
		name          string
		format        Format
		wantConverted int
		wantAmbiguous int
	}{
		{"gu", FormatGu, 5, 0}, {"item", FormatItem, 2, 0}, {"gamefx", FormatGameFX, 3, 0}, {"killer_move", FormatKiller, 3, 0}, {"aperture", FormatAperture, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Convert(tt.format, "fixture", fixture(t, tt.name))
			if len(r.Report.Converted) != tt.wantConverted || len(r.Report.Ambiguous) != tt.wantAmbiguous {
				t.Fatalf("report = %+v", r.Report)
			}
			if tt.format != FormatAperture {
				var envelope struct {
					SchemaVersion int               `json:"schema_version"`
					Blocks        []json.RawMessage `json:"blocks"`
				}
				if err := json.Unmarshal(r.Envelope, &envelope); err != nil || envelope.SchemaVersion != 2 || len(envelope.Blocks) != tt.wantConverted {
					t.Fatalf("envelope %s: %v", r.Envelope, err)
				}
			}
		})
	}
}

func TestAmbiguousNeverGuessesAndGateStaysClosed(t *testing.T) {
	r := ConvertGu("ambiguous", []byte(`{"active":{"sleep":2}}`))
	if len(r.Report.Ambiguous) != 1 || len(r.Report.SuggestedMapping) != 1 || r.Report.MigrationGate {
		t.Fatalf("report = %+v", r.Report)
	}
}

func TestDuplicateAliasesAndExplicitDivergenceAreReported(t *testing.T) {
	r := ConvertKillerMove("move", []byte(`{"lifesteal":0.1,"life_steal":0.2}`))
	if len(r.Report.Duplicate) != 1 || r.Report.MigrationGate {
		t.Fatalf("duplicate report = %+v", r.Report)
	}
	g := ConvertGu("gu", []byte(`{"active":{"damage_flat":1},"blocks":[{"id":"OTHER","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":2}]}]}`))
	if len(g.Report.DivergentExplicitBlockLegacy) != 1 {
		t.Fatalf("divergence report = %+v", g.Report)
	}
}

func TestReportJSONUsesStableMachineReadableArrays(t *testing.T) {
	r := ConvertItem("empty", []byte(`{}`))
	b, err := json.Marshal(r.Report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"converted", "golden_parity", "differential_parity", "exact_parity", "unsupported", "ambiguous", "duplicate", "divergent_explicit_block_legacy", "suggested_mapping"} {
		if _, ok := decoded[key].([]any); !ok {
			t.Fatalf("%s is not an array: %s", key, b)
		}
	}
}

func TestEvaluateMigrationGateLevels(t *testing.T) {
	finding := Finding{Format: FormatGu, SourceID: "fixture", Key: "damage_flat"}
	tests := []struct {
		name   string
		report Report
		code   string
		pass   bool
	}{
		{"unsupported", Report{Validated: true, Unsupported: []Finding{finding}}, "unsupported_effect", false},
		{"ambiguous", Report{Validated: true, Ambiguous: []Finding{finding}}, "ambiguous_effect", false},
		{"divergence", Report{Validated: true, DivergentExplicitBlockLegacy: []Finding{finding}}, "differential_divergence", false},
		{"missing parity", Report{Total: 1, Validated: true, Converted: []Finding{finding}}, "missing_parity", false},
		{"shadow mismatch", Report{Total: 1, Validated: true, Converted: []Finding{finding}, ExactParity: []Finding{finding}, ShadowMismatchCount: 1}, "shadow_mismatch", false},
		{"empty envelope", Report{Validated: true}, "", true},
		{"exact golden parity", Report{Total: 1, Validated: true, Converted: []Finding{finding}, ExactParity: []Finding{finding}}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateMigrationGate(tt.report)
			if got.MigrationGate != tt.pass || got.ErrorCode != tt.code {
				t.Fatalf("decision = %+v", got)
			}
		})
	}
}

func TestParityEvidenceMustCoverEachConvertedEffect(t *testing.T) {
	one := Finding{Format: FormatGu, SourceID: "fixture", Key: "one"}
	two := Finding{Format: FormatGu, SourceID: "fixture", Key: "two"}
	report := Report{Total: 2, Validated: true, Converted: []Finding{one, two}}
	RecordGoldenParity(&report, []Finding{one, one})
	if report.MigrationGate || report.ErrorCode != "missing_parity" || len(report.ExactParity) != 1 {
		t.Fatalf("duplicate evidence opened gate: %+v", report)
	}
}

func TestMalformedExplicitGuBlocksFail(t *testing.T) {
	r := ConvertGu("bad-blocks", []byte(`{"blocks":{"not":"an array"}}`))
	if r.Report.MigrationGate || len(r.Report.Unsupported) != 1 {
		t.Fatalf("malformed explicit blocks passed: %+v", r.Report)
	}
}

func TestConversionNeedsRealParityEvidence(t *testing.T) {
	r := ConvertGu("fixture", []byte(`{"active":{"damage_flat":2}}`))
	if !r.Report.Validated || r.Report.MigrationGate || len(r.Report.ExactParity) != 0 {
		t.Fatalf("conversion incorrectly established parity: %+v", r.Report)
	}
	RecordGoldenParity(&r.Report, append([]Finding(nil), r.Report.Converted...))
	if !r.Report.MigrationGate || len(r.Report.ExactParity) != 1 {
		t.Fatalf("golden evidence did not open gate: %+v", r.Report)
	}
}

func TestMatchingExplicitBlocksProvideDifferentialParity(t *testing.T) {
	legacy := []byte(`{"active":{"damage_flat":2}}`)
	first := ConvertGu("fixture", legacy)
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(first.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	withBlocks := []byte(`{"active":{"damage_flat":2},"blocks":` + string(envelope["blocks"]) + `}`)
	result := ConvertGu("fixture", withBlocks)
	if !result.Report.MigrationGate || len(result.Report.DifferentialParity) != 1 || len(result.Report.ExactParity) != 1 {
		t.Fatalf("matching differential result did not open gate: %+v", result.Report)
	}
}

func TestEmptyEnvelopePassesGate(t *testing.T) {
	r := ConvertItem("empty", []byte(`{}`))
	if !r.Report.Validated || !r.Report.MigrationGate || r.Report.Total != 0 {
		t.Fatalf("empty envelope report = %+v; envelope=%s", r.Report, r.Envelope)
	}
}

func TestGuDomainMappingsUseRuntimeTriggersAndUnits(t *testing.T) {
	r := ConvertGu("domain", []byte(`{"active":{"capture_bonus":0.2,"portal_create":true,"random_portal":false},"passive":{"time_speed_multiplier":2,"success_rate_bonus":15,"path_conflict_reduction":100},"growth":{"max_hp":10,"hours_to_max":24}}`))
	if !r.Report.Validated {
		t.Fatalf("candidate did not validate: %s", r.Envelope)
	}
	var envelope struct {
		Blocks []struct {
			Trigger    string      `json:"trigger"`
			Operations []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	wantTrigger := map[string]string{"CAPTURE_BONUS": "CAPABILITY", "PORTAL_CREATE": "CAPABILITY", "TIME_SPEED": "ALWAYS_ON", "REFINE_SUCCESS": "ALWAYS_ON", "PATH_CONFLICT_REDUCTION": "ALWAYS_ON", "HP": "ALWAYS_ON"}
	wantUnit := map[string]string{"TIME_SPEED": "RATIO", "REFINE_SUCCESS": "PERCENT", "PATH_CONFLICT_REDUCTION": "PERCENT"}
	for _, b := range envelope.Blocks {
		op := b.Operations[0]
		key := op.Code
		if key == "" {
			key = op.Stat
		}
		if got := wantTrigger[key]; got != "" && b.Trigger != got {
			t.Errorf("%s trigger = %s, want %s", key, b.Trigger, got)
		}
		if got := wantUnit[key]; got != "" && op.Unit != got {
			t.Errorf("%s unit = %s, want %s", key, op.Unit, got)
		}
	}
}

func TestBlockedActionsRemainStatusMetadata(t *testing.T) {
	r := ConvertGameFX("status", []byte(`{"blocked_actions":["explore"]}`))
	if len(r.Report.Converted) != 0 || len(r.Report.Unsupported) != 1 {
		t.Fatalf("blocked_actions was converted as an effect: %+v", r.Report)
	}
}
