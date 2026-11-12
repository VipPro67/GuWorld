package effectcatalog

import (
	"bytes"
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

func TestBlockV2AcceptsCataloguedOperation(t *testing.T) {
	raw := []byte(`{"schema_version":2,"blocks":[{"id":"BASIC_HIT","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10,"unit":"FLAT"}]}]}`)
	if issues := ValidateBlockV2(raw, Location{Source: "gu", Row: 2, Key: "GU_TEST"}); len(issues) != 0 {
		t.Fatalf("unexpected issues: %+v", issues)
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
