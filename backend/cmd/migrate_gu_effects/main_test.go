package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/effectmigration"
)

func TestRejectSourceDestinationIncludesReportAliases(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.json")
	if err := os.WriteFile(source, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(source)
	if rejectSourceDestination(abs, source) == nil {
		t.Fatal("direct source destination accepted")
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(source, link); err != nil {
		t.Fatal(err)
	}
	if rejectSourceDestination(abs, link) == nil {
		t.Fatal("symlink source destination accepted")
	}
	hard := filepath.Join(dir, "hard.json")
	if err := os.Link(source, hard); err != nil {
		t.Fatal(err)
	}
	if rejectSourceDestination(abs, hard) == nil {
		t.Fatal("hard-link source destination accepted")
	}
}

func TestMergeGuCandidateRetainsLegacyNamespaces(t *testing.T) {
	legacy := []byte(`{"active":{"damage_flat":2},"growth":{"max_hp":3}}`)
	envelope := []byte(`{"schema_version":2,"blocks":[{"id":"DAMAGE"}]}`)
	got, err := mergeCandidate(effectmigration.FormatGu, legacy, envelope)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err = json.Unmarshal(got, &root); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"active", "growth", "schema_version", "blocks"} {
		if _, ok := root[key]; !ok {
			t.Fatalf("missing %s in %s", key, got)
		}
	}
}
