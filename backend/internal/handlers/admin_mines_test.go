package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/authz"
)

func setTestAdminPrincipal(c *gin.Context, adminID uuid.UUID) {
	principal := authz.AdminPrincipal{
		PlayerID: adminID,
		Roles:    []string{authz.SuperAdminRole},
		Permissions: map[string]struct{}{
			authz.AdminAccessPermission: {},
			"admin.*":                   {},
		},
	}
	c.Request = c.Request.WithContext(authz.WithPrincipal(c.Request.Context(), principal))
}

func TestAdminGetResourcePointTemplates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/world-resource-points/templates", nil)

	adminID := uuid.New()
	setTestAdminPrincipal(c, adminID)

	AdminGetResourcePointTemplates(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []ResourcePointTemplate `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatalf("expected non-empty templates, got 0")
	}
	foundValley := false
	for _, tpl := range resp.Data {
		if tpl.PointType == "HERB_VALLEY" {
			foundValley = true
			if tpl.Name == "" || tpl.Path == "" {
				t.Errorf("HERB_VALLEY template has missing name or path")
			}
			break
		}
	}
	if !foundValley {
		t.Errorf("expected HERB_VALLEY template in response")
	}
}

func TestAdminCreatePrimevalVeinValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/primeval-veins", nil)

	adminID := uuid.New()
	setTestAdminPrincipal(c, adminID)

	AdminCreatePrimevalVein(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty body, got %d", rec.Code)
	}
}
