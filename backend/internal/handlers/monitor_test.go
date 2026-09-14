package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/internal/server"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestPublicDebugEndpointsAreNotRegistered(t *testing.T) {
	r := server.SetupRouter(nil)

	endpoints := []string{
		"/debug/pprof/", "/debug/pprof/profile", "/debug/pprof/trace",
		"/debug/pprof/goroutine/stack", "/debug/db-pool", "/debug/system",
		"/debug/requests/summary", "/api/debug/monsters",
	}
	for _, endpoint := range endpoints {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 from removed endpoint %s, got %d", endpoint, w.Code)
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/debug/pprof/symbol", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 from removed endpoint POST /debug/pprof/symbol, got %d", w.Code)
	}
}

func TestTraceRecordingAndHeader(t *testing.T) {
	r := server.SetupRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/not-found", nil)
	r.ServeHTTP(w, req)

	traceID := w.Header().Get("X-Trace-ID")
	if traceID == "" {
		t.Errorf("expected X-Trace-ID header in response")
	}

	// Check if trace exists in tracker
	trace, exists := monitor.GlobalTracker.GetTraceByID(traceID)
	if !exists {
		t.Fatalf("expected trace %s to be recorded in GlobalTracker", traceID)
	}
	if trace.Path != "/not-found" {
		t.Errorf("expected trace path /not-found, got %s", trace.Path)
	}
}

func TestAdminMonitorUnauthorized(t *testing.T) {
	r := server.SetupRouter(nil)

	// Access without token should be 401
	endpoints := []string{
		"/api/admin/monitor/system",
		"/api/admin/monitor/requests/summary",
		"/api/admin/monitor/requests/traces",
	}

	for _, ep := range endpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, ep, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for %s, got %d", ep, w.Code)
		}
	}
}
