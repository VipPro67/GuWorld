package handlers

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/db"
)

func TestShippedEffectDatasetsMatchRuntimeAdapters(t *testing.T) {
	for _, dataset := range []string{"item", "race", "constitution", "status"} {
		sources, err := db.ReadImportSources(dataset, nil)
		if err != nil {
			t.Fatal(err)
		}
		if issues := validateImportRuntimeEffectAdapters(sources); len(issues) > 0 {
			t.Fatalf("shipped %s effect dataset contains unsupported runtime definitions: %+v", dataset, issues[:min(5, len(issues))])
		}
	}
}

func TestImportEffectAdapterValidationRejectsUnsupportedItemCondition(t *testing.T) {
	effects := `[{"id":"RESTORE_HP","trigger":"ON_CONSUME","target":{"selector":"SELF"},"conditions":[{"field":"HP_PERCENT","operator":"LT","value":0.5}],"operations":[{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}}]}]`
	var content bytes.Buffer
	writer := csv.NewWriter(&content)
	if err := writer.Write([]string{"id", "name", "item_type", "rank", "effects"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Write([]string{"conditional", "Conditional", "Consumable", "1", effects}); err != nil {
		t.Fatal(err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	sources := map[string][]byte{"Item.csv": content.Bytes()}
	issues := validateImportRuntimeEffectAdapters(sources)
	if len(issues) != 1 || issues[0].Dataset != "item" || issues[0].Code != "unsupported_effect_context" {
		t.Fatalf("unsupported item condition was not blocked by preflight: %+v", issues)
	}
}
