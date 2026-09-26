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
	for _, key := range []string{"converted", "exact_parity", "unsupported", "ambiguous", "duplicate", "divergent_explicit_block_legacy", "suggested_mapping"} {
		if _, ok := decoded[key].([]any); !ok {
			t.Fatalf("%s is not an array: %s", key, b)
		}
	}
}
