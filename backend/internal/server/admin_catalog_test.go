package server

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/admincatalog"
	"github.com/vippro67/GuWorld/backend/internal/authz"
)

// This test makes route authorization and the public catalog fail together
// when a new admin domain is added without compile-time metadata.
func TestAdminResourceCatalogCoversRoutePermissions(t *testing.T) {
	catalogPermissions := map[string]string{}
	for _, resource := range admincatalog.All() {
		for _, permission := range append(append([]string{}, resource.ReadPermissions...), resource.WritePermissions...) {
			if owner, exists := catalogPermissions[permission]; exists {
				t.Fatalf("permission %q belongs to both %q and %q", permission, owner, resource.Key)
			}
			catalogPermissions[permission] = resource.Key
		}
	}

	routePermissions := map[string]bool{}
	hasResourcesRoute := false
	for _, route := range adminRouteSpecs() {
		if route.path == "/resources" {
			hasResourcesRoute = true
		}
		if route.permission == authz.AdminAccessPermission {
			continue
		}
		routePermissions[route.permission] = true
		if _, ok := catalogPermissions[route.permission]; !ok {
			t.Errorf("admin route %s %s permission %q has no catalog resource", route.method, route.path, route.permission)
		}
	}
	if !hasResourcesRoute {
		t.Error("GET /api/admin/resources is not registered")
	}
	for permission, resource := range catalogPermissions {
		if !routePermissions[permission] {
			t.Errorf("catalog resource %q declares unused permission %q", resource, permission)
		}
	}
}
