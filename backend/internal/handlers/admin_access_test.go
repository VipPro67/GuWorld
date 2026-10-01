package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/authz"
)

func TestVisibleAdminResourcesFiltersByEveryRequiredPermission(t *testing.T) {
	resources := visibleAdminResources([]string{"players.read", "monitoring.read"})
	if len(resources) != 2 {
		t.Fatalf("got %d resources, want 2", len(resources))
	}
	if resources[0].Key != "players" || resources[1].Key != "monitoring" {
		t.Fatalf("unexpected visible resources: %+v", resources)
	}
}

func TestVisibleAdminResourcesAcceptsAdminWildcard(t *testing.T) {
	resources := visibleAdminResources([]string{"admin.*"})
	if len(resources) != len(adminResourceCatalog) {
		t.Fatalf("wildcard exposes %d resources, want %d", len(resources), len(adminResourceCatalog))
	}
}

func TestVisibleAdminResourcesFiltersMutationCapabilities(t *testing.T) {
	readOnly := visibleAdminResources([]string{"shop.read"})
	if len(readOnly) != 1 || strings.Join(readOnly[0].Capabilities, ",") != "read" {
		t.Fatalf("read-only shop capabilities = %+v", readOnly)
	}
	withUpdate := visibleAdminResources([]string{"shop.read", "shop.update"})
	if got := strings.Join(withUpdate[0].Capabilities, ","); got != "read,create,update,delete" {
		t.Fatalf("shop update capabilities = %q", got)
	}
}

func TestAdminGetResourcesUsesPrincipalPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	principal := authz.AdminPrincipal{
		PlayerID: uuid.New(),
		Permissions: map[string]struct{}{
			authz.AdminAccessPermission: {},
			"players.read":              {},
		},
	}
	router := gin.New()
	router.GET("/resources", func(c *gin.Context) {
		c.Request = c.Request.WithContext(authz.WithPrincipal(c.Request.Context(), principal))
		AdminGetResources(c)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/resources", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Resources []adminResource `json:"resources"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Resources) != 1 || response.Resources[0].Key != "players" {
		t.Fatalf("resources = %+v", response.Resources)
	}
}

func TestAdminResourceCatalogUsesConcreteRoutesAndModes(t *testing.T) {
	validModes := map[string]bool{"full_crud": true, "read_only": true, "domain_command_only": true, "import_export_only": true, "internal_not_exposed": true}
	for _, resource := range adminResourceCatalog {
		if resource.FrontendRoute != "/admin/"+resource.Key {
			t.Errorf("%s frontend route = %q", resource.Key, resource.FrontendRoute)
		}
		if !validModes[resource.ManagementMode] {
			t.Errorf("%s management mode = %q", resource.Key, resource.ManagementMode)
		}
		for _, permission := range resource.RequiredPermissions {
			if strings.HasPrefix(permission, "admin.") {
				t.Errorf("%s uses stale permission %q", resource.Key, permission)
			}
		}
	}
}

func TestCatalogPermissionsExposeWholeCatalog(t *testing.T) {
	permissions := catalogPermissions()
	resources := visibleAdminResources(permissions)
	if len(resources) != len(adminResourceCatalog) {
		t.Fatalf("got %d resources, want %d", len(resources), len(adminResourceCatalog))
	}
	if legacyAdminPermissionVersion == "" {
		t.Fatal("permission version must be usable as an invalidation marker")
	}
}

func TestAdminResourceJSONDoesNotExposeDatabaseMetadata(t *testing.T) {
	payload, err := json.Marshal(visibleAdminResources(catalogPermissions()))
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
