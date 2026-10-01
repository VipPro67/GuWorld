package handlers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVisibleAdminResourcesFiltersByEveryRequiredPermission(t *testing.T) {
	resources := visibleAdminResources([]string{"admin.players", "admin.monitoring"})
	if len(resources) != 2 {
		t.Fatalf("got %d resources, want 2", len(resources))
	}
	if resources[0].Key != "players" || resources[1].Key != "monitoring" {
		t.Fatalf("unexpected visible resources: %+v", resources)
	}
}

func TestLegacyAdminPermissionsExposeWholeCatalog(t *testing.T) {
	permissions := legacyAdminPermissions()
	resources := visibleAdminResources(permissions)
	if len(resources) != len(adminResourceCatalog) {
		t.Fatalf("got %d resources, want %d", len(resources), len(adminResourceCatalog))
	}
	if legacyAdminPermissionVersion == "" {
		t.Fatal("permission version must be usable as an invalidation marker")
	}
}

func TestAdminResourceJSONDoesNotExposeDatabaseMetadata(t *testing.T) {
	payload, err := json.Marshal(visibleAdminResources(legacyAdminPermissions()))
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	serialized := strings.ToLower(string(payload))
	for _, forbidden := range []string{"schema", "table_name", "column_name", "sql_metadata", "information_schema"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("catalog contains forbidden database metadata %q: %s", forbidden, serialized)
		}
	}
}
