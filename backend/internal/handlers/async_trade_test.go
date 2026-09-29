package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/server"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestAcceptTradeOfferRequiresAuth verifies the trade-offer accept route is
// behind the auth middleware, so unauthenticated callers get 401 rather than
// a 500 panic from a nil player context.
func TestAcceptTradeOfferRequiresAuth(t *testing.T) {
	r := server.SetupRouter(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/api/shop/bao-hoang/trade-offers/accept",
		nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", w.Code)
	}
}

func TestCreateTradeOfferRequiresAuth(t *testing.T) {
	r := server.SetupRouter(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/api/shop/bao-hoang/trade-offers",
		nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", w.Code)
	}
}

func TestCancelTradeOfferRequiresAuth(t *testing.T) {
	r := server.SetupRouter(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/api/shop/bao-hoang/trade-offers/cancel",
		nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", w.Code)
	}
}
