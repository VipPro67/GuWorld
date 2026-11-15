package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/authz"
)

func TestAdminUpdateGiftCode_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest(http.MethodPut, "/api/admin/gift-codes/invalid-uuid", bytes.NewBufferString(`{"max_uses": 50}`))
	req.Header.Set("Content-Type", "application/json")
	principal := authz.AdminPrincipal{
		PlayerID:    uuid.New(),
		Roles:       []string{authz.SuperAdminRole},
		Permissions: map[string]struct{}{"admin.*": {}},
	}
	req = req.WithContext(authz.WithPrincipal(context.Background(), principal))
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	AdminUpdateGiftCode(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestAdminDeleteGiftCode_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest(http.MethodDelete, "/api/admin/gift-codes/invalid-uuid", nil)
	principal := authz.AdminPrincipal{
		PlayerID:    uuid.New(),
		Roles:       []string{authz.SuperAdminRole},
		Permissions: map[string]struct{}{"admin.*": {}},
	}
	req = req.WithContext(authz.WithPrincipal(context.Background(), principal))
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	AdminDeleteGiftCode(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", w.Code)
	}
}
