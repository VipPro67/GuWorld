package db

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizedGuEffectsWritesCanonicalEnvelope(t *testing.T) {
	block := `{"id":"BASIC_HIT","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10,"unit":"FLAT"}]}`
	for _, raw := range []string{"[" + block + "]", `{"blocks":[` + block + `]}`, `{"schema_version":2,"blocks":[` + block + `]}`} {
		got, _, err := normalizedGuEffectsJSON(raw, "unit test")
		if err != nil {
			t.Fatalf("normalize %s: %v", raw, err)
		}
		var envelope struct {
			SchemaVersion int               `json:"schema_version"`
			Blocks        []json.RawMessage `json:"blocks"`
		}
		if err := json.Unmarshal([]byte(got), &envelope); err != nil || envelope.SchemaVersion != 2 || len(envelope.Blocks) != 1 {
			t.Fatalf("normalized = %s, err=%v", got, err)
		}
	}
	if _, _, err := normalizedGuEffectsJSON(`{"schema_version":3,"blocks":[]}`, "unit test"); err == nil {
		t.Fatal("unsupported envelope version accepted")
	}
}

func TestValidateCSVFiles(t *testing.T) {
	// Change working directory to backend root so csvPath candidates match
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %v", err)
	}
	// Try changing to backend dir (from backend/internal/db)
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}
	defer os.Chdir(oldWD)

	// 1. Test Gu.csv
	f, r, header, err := openCSV("Gu.csv")
	if err != nil {
		t.Fatalf("Failed to open Gu.csv: %v", err)
	}
	defer f.Close()

	col := func(name string) int { return colIndex(header, name) }
	idIdx := col("id")
	rankIdx := col("rank")
	effectsIdx := col("effects")
	foodIdx := col("food_types")
	pathIdx := col("path")

	if idIdx < 0 || rankIdx < 0 || effectsIdx < 0 {
		t.Fatalf("Gu.csv missing required columns")
	}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Error reading Gu.csv row: %v", err)
			continue
		}
		id := row[idIdx]
		rank := row[rankIdx]
		effectsRaw := row[effectsIdx]
		foodRaw := row[foodIdx]
		pathRaw := row[pathIdx]

		if id == "" {
			continue
		}

		// Verify effects is valid JSON
		effectsRawNormalized, _, err := normalizedGuEffectsJSON(effectsRaw, fmt.Sprintf("%s effects", id))
		if err != nil {
			t.Errorf("Gu %s (rank %s) has invalid effects: %v", id, rank, err)
			continue
		}
		var effEnvelope any
		if err := json.Unmarshal([]byte(effectsRawNormalized), &effEnvelope); err != nil {
			t.Errorf("Gu %s (rank %s) has invalid effects JSON: %v", id, rank, err)
		}

		// Verify food_types is valid JSON
		if foodRaw != "" && foodRaw != "none" && foodRaw != "unknown" {
			if strings.HasPrefix(foodRaw, "[") && strings.HasSuffix(foodRaw, "]") {
				var foodList []string
				if err := json.Unmarshal([]byte(foodRaw), &foodList); err != nil {
					t.Errorf("Gu %s (rank %s) has invalid food_types JSON: %v", id, rank, err)
				}
			}
		}

		// Verify path is valid JSON
		if pathRaw != "" {
			if strings.HasPrefix(pathRaw, "[") && strings.HasSuffix(pathRaw, "]") {
				var pathList []string
				if err := json.Unmarshal([]byte(pathRaw), &pathList); err != nil {
					t.Errorf("Gu %s (rank %s) has invalid path JSON: %v", id, rank, err)
				}
			}
		}
	}

	// 2. Test Recipe.csv
	rf, rr, rheader, err := openCSV("Recipe.csv")
	if err != nil {
		t.Fatalf("Failed to open Recipe.csv: %v", err)
	}
	defer rf.Close()

	rcol := func(name string) int { return colIndex(rheader, name) }
	targetIdx := rcol("target_gu_id")
	reqGusIdx := rcol("required_gus")
	reqMatsIdx := rcol("required_materials")

	if targetIdx < 0 || reqGusIdx < 0 || reqMatsIdx < 0 {
		t.Fatalf("Recipe.csv missing required columns")
	}

	for {
		row, err := rr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Error reading Recipe.csv row: %v", err)
			continue
		}
		target := row[targetIdx]
		if target == "" {
			continue
		}
		reqGusRaw := row[reqGusIdx]
		reqMatsRaw := row[reqMatsIdx]

		if reqGusRaw != "" && reqGusRaw != "{}" {
			var gus map[string]interface{}
			if err := json.Unmarshal([]byte(reqGusRaw), &gus); err != nil {
				t.Errorf("Recipe for %s has invalid required_gus: %v", target, err)
			}
		}
		if reqMatsRaw != "" && reqMatsRaw != "{}" {
			var mats map[string]interface{}
			if err := json.Unmarshal([]byte(reqMatsRaw), &mats); err != nil {
				t.Errorf("Recipe for %s has invalid required_materials: %v", target, err)
			}
		}
	}
}

func TestPreflightSources(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	sources := []struct {
		table    string
		filename string
		key      string
	}{
		{"gu", "Gu.csv", "id"},
		{"item", "Item.csv", "id"},
		{"race", "Race.csv", "key"},
		{"constitution", "Constitution.csv", "key"},
		{"status", "Status.csv", "key"},
	}

	for _, s := range sources {
		t.Run(s.table, func(t *testing.T) {
			err := preflightEffectCSVs([]effectCSVSource{{s.table, s.filename, s.key}})
			if err != nil {
				t.Errorf("%s preflight failed: %v", s.table, err)
			}
		})
	}
}

func TestGuPreflightUsesStagedStatusCSVRegistry(t *testing.T) {
	statusPath := filepath.Join(t.TempDir(), "Status.csv")
	guPath := filepath.Join(t.TempDir(), "Gu.csv")
	writeRows := func(path string, rows [][]string) {
		t.Helper()
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		writer := csv.NewWriter(f)
		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				f.Close()
				t.Fatal(err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	const statusKey = "staged_status_for_preflight_test"
	writeRows(statusPath, [][]string{{"key", "name"}, {statusKey, "Staged"}})
	effect := `{"schema_version":2,"blocks":[{"id":"STATUS","trigger":"ON_HIT","target":{"selector":"TARGET"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"` + statusKey + `","max_stacks":1,"duration_hours":1,"dispel_rule":"CLEAR_ALL","pvp_only":true,"tier_scaling":[{"tier":1,"stat_percent":-10}]}]}]}`
	writeRows(guPath, [][]string{{"id", "effects"}, {"GU_TEST", effect}})

	csvImportState.Lock()
	previousOverride := csvImportState.override
	t.Cleanup(func() {
		csvImportState.override = previousOverride
		csvImportState.Unlock()
	})
	csvImportState.override = map[string]string{"Gu.csv": guPath}
	if err := preflightEffectCSVs([]effectCSVSource{{source: "gu", filename: "Gu.csv", keyColumn: "id"}}); err == nil {
		t.Fatal("Gu-only preflight accepted a status key that exists only in an unstaged local Status.csv")
	}
	csvImportState.override = map[string]string{"Status.csv": statusPath, "Gu.csv": guPath}
	if err := preflightEffectCSVs([]effectCSVSource{{source: "gu", filename: "Gu.csv", keyColumn: "id"}}); err != nil {
		t.Fatalf("preflight rejected status key staged in the same batch: %v", err)
	}
}

func TestParseCSVPrice(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"250", 250},
		{"250.000", 250},
		{"1000", 1000},
		{"1000.000", 1000},
		{"1.000.000", 1000},
		{"4.000.000", 4000},
		{"22.000.000", 22000},
		{"250.000.000", 250000},
		{"3.500.000", 3500},
		{"35.000.000", 35000},
		{"550.000.000", 550000},
		{"14.000.000.000", 14000000},
		{"1000.5", 1000.5},
		{"1,000", 1000},
		{"250,000", 250000},
		{"14,000,000", 14000000},
		{"14,000,000.000", 14000000},
		{"", 0},
		{"0", 0},
	}

	for _, tc := range tests {
		got := parseCSVPrice(tc.input)
		if got != tc.expected {
			t.Errorf("parseCSVPrice(%q) = %f, want %f", tc.input, got, tc.expected)
		}
	}
}
