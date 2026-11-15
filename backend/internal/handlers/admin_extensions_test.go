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

func testAdminContext(method, url, body string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != "" {
		req, _ = http.NewRequest(method, url, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}

	principal := authz.AdminPrincipal{
		PlayerID:    uuid.New(),
		Roles:       []string{authz.SuperAdminRole},
		Permissions: map[string]struct{}{"admin.*": {}},
	}
	req = req.WithContext(authz.WithPrincipal(context.Background(), principal))
	c.Request = req
	return w, c
}

func TestAdminPlayerAdvanced_InvalidIDs(t *testing.T) {
	// 1. List Gu Houses invalid player ID
	w, c := testAdminContext(http.MethodGet, "/api/admin/players/invalid/gu-houses", "")
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	AdminListPlayerGuHouses(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid player ID, got %d", w.Code)
	}

	// 2. Delete Gu House invalid house ID
	w, c = testAdminContext(http.MethodDelete, "/api/admin/players/"+uuid.New().String()+"/gu-houses/invalid", "")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}, {Key: "house_id", Value: "invalid"}}
	AdminDeletePlayerGuHouse(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid house ID, got %d", w.Code)
	}

	// 3. List Killer Moves invalid player ID
	w, c = testAdminContext(http.MethodGet, "/api/admin/players/invalid/killer-moves", "")
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	AdminListPlayerKillerMoves(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid player ID, got %d", w.Code)
	}

	// 4. Delete Killer Move invalid move ID
	w, c = testAdminContext(http.MethodDelete, "/api/admin/players/"+uuid.New().String()+"/killer-moves/invalid", "")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}, {Key: "move_id", Value: "invalid"}}
	AdminDeletePlayerKillerMove(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid move ID, got %d", w.Code)
	}
}

func TestAdminFeedback_InvalidIDs(t *testing.T) {
	// Update feedback invalid ID
	w, c := testAdminContext(http.MethodPut, "/api/admin/feedback/invalid/status", `{"status":"completed"}`)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	AdminUpdateFeedbackStatus(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid feedback ID, got %d", w.Code)
	}

	// Delete feedback invalid ID
	w, c = testAdminContext(http.MethodDelete, "/api/admin/feedback/invalid", "")
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	AdminDeleteFeedback(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid feedback ID, got %d", w.Code)
	}
}

func TestAdminMarketplace_InvalidIDs(t *testing.T) {
	// Cancel auction invalid ID
	w, c := testAdminContext(http.MethodDelete, "/api/admin/marketplace/auctions/invalid", "")
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	AdminCancelAuction(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid auction ID, got %d", w.Code)
	}
}

func TestAdminTeleport_InvalidIDs(t *testing.T) {
	// Update teleport array invalid ID
	w, c := testAdminContext(http.MethodPut, "/api/admin/teleport-arrays/invalid", `{"name":"Trận 1"}`)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	AdminUpdateTeleportArray(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid teleport array ID, got %d", w.Code)
	}
}

func TestAdminRegionLoot_Validation(t *testing.T) {
	// Create region loot invalid body
	w, c := testAdminContext(http.MethodPost, "/api/admin/region-loot-tables", `{}`)
	AdminCreateRegionLootTable(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty region loot body, got %d", w.Code)
	}
}
