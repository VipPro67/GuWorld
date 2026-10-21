package admincatalog

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/authz"
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

func TestBuiltinRoleResourceMatrix(t *testing.T) {
	tests := []struct {
		role string
		want []string
	}{
		{"player", []string{}},
		{"observer", []string{"apertures", "arena_rankings", "arena_shop", "audit", "combat_history", "constitution_dictionary", "dungeons", "game_data", "gift_codes", "gu_dictionary", "gus", "item_dictionary", "loot_tables", "map_cells", "map_drops", "monitoring", "monster_spawns", "monsters", "players", "quests", "race_dictionary", "recipes", "regions", "sect_mines", "sects", "shop", "status_dictionary", "world_bosses"}},
		{"support_admin", []string{"audit", "gus", "players"}},
		{"content_admin", []string{"cache", "constitution_dictionary", "game_data", "gu_dictionary", "images", "item_dictionary", "loot_tables", "monsters", "quests", "race_dictionary", "recipes", "status_dictionary"}},
		{"game_admin", []string{"apertures", "arena_rankings", "arena_shop", "audit", "combat_history", "dungeons", "gift_codes", "gus", "map_cells", "map_drops", "monitoring", "monster_spawns", "players", "regions", "sect_mines", "sects", "shop", "world_bosses"}},
		{"finance_viewer", []string{"billing"}},
		{"super_admin", resourceKeys(All())},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			permissions := authz.PermissionsForRole(tt.role)
			allowed := func(permission string) bool {
				return contains(permissions, "admin.*") || contains(permissions, permission)
			}
			got := resourceKeys(Visible(allowed))
			sort.Strings(got)
			sort.Strings(tt.want)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("visible resources = %v, want %v", got, tt.want)
			}
		})
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func resourceKeys(resources []Resource) []string {
	keys := make([]string, len(resources))
	for i := range resources {
		keys[i] = resources[i].Key
	}
	return keys
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
