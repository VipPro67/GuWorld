package admincatalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCatalogIsCompleteSafeAllowlist(t *testing.T) {
	seen := map[string]bool{}
	validRisk := map[string]bool{"medium": true, "high": true, "critical": true}
	for _, resource := range All() {
		if resource.Key == "" || resource.Domain == "" || resource.Label == "" || resource.FrontendRoute == "" || resource.AuditActionPrefix == "" {
			t.Fatalf("resource has incomplete metadata: %+v", resource)
		}
		if seen[resource.Key] {
			t.Fatalf("duplicate resource key %q", resource.Key)
		}
		seen[resource.Key] = true
		if len(resource.Capabilities) == 0 || len(resource.ReadPermissions)+len(resource.WritePermissions) == 0 {
			t.Fatalf("resource %q has no capabilities or permissions", resource.Key)
		}
		if !validRisk[resource.RiskLevel] {
			t.Fatalf("resource %q has invalid risk level %q", resource.Key, resource.RiskLevel)
		}
		if resource.AuditActionPrefix != "admin." {
			t.Fatalf("resource %q has an audit prefix that does not match persisted admin actions", resource.Key)
		}
		if resource.RestartRequired {
			t.Fatalf("resource %q claims restart is required although admin mutations reload in-process", resource.Key)
		}
	}

	payload, err := json.Marshal(All())
	if err != nil {
		t.Fatal(err)
	}
	serialized := strings.ToLower(string(payload))
	for _, forbidden := range []string{"information_schema", "table_name", "column_name", "select ", "insert ", "update ", "delete from"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("catalog exposes forbidden database/SQL metadata %q", forbidden)
		}
	}
}

func TestVisibleUsesDeclaredPermissionsAndCopiesSlices(t *testing.T) {
	visible := Visible(func(permission string) bool { return permission == "players.read" })
	if len(visible) != 1 || visible[0].Key != "players" {
		t.Fatalf("unexpected resources: %+v", visible)
	}
	visible[0].Capabilities[0] = "changed"
	if All()[2].Capabilities[0] == "changed" {
		t.Fatal("Visible returned catalog-owned slice")
	}
}
