package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/internal/server"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDebugEndpoints(t *testing.T) {
	r := server.SetupRouter(nil)

	// Test GET /debug/system
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/debug/system", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /debug/system, got %d (body: %s)", w.Code, w.Body.String())
	}

	var sysResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &sysResp); err != nil {
		t.Fatalf("failed to parse /debug/system response: %v", err)
	}
	if _, exists := sysResp["cpu"]; !exists {
		t.Errorf("expected 'cpu' key in system stats")
	}
	if _, exists := sysResp["memory"]; !exists {
		t.Errorf("expected 'memory' key in system stats")
	}
	if _, exists := sysResp["server"]; !exists {
		t.Errorf("expected 'server' key in system stats")
	}

	// Test GET /debug/requests/summary
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/debug/requests/summary", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 from /debug/requests/summary, got %d", w2.Code)
	}

	var reqResp monitor.RequestSummary
	if err := json.Unmarshal(w2.Body.Bytes(), &reqResp); err != nil {
		t.Fatalf("failed to parse /debug/requests/summary response: %v", err)
	}
	if reqResp.TotalRequests <= 0 {
		t.Errorf("expected TotalRequests > 0 after prior requests, got %d", reqResp.TotalRequests)
	}

	// Test GET /debug/pprof (should redirect to /debug/pprof/)
	wRedirect := httptest.NewRecorder()
	reqRedirect, _ := http.NewRequest(http.MethodGet, "/debug/pprof", nil)
	r.ServeHTTP(wRedirect, reqRedirect)
	if wRedirect.Code != http.StatusMovedPermanently || wRedirect.Header().Get("Location") != "/debug/pprof/" {
		t.Errorf("expected 301 redirect to /debug/pprof/, got %d location: %s", wRedirect.Code, wRedirect.Header().Get("Location"))
	}

	// Test GET /debug/pprof/
	wPprof := httptest.NewRecorder()
	reqPprof, _ := http.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	r.ServeHTTP(wPprof, reqPprof)
	if wPprof.Code != http.StatusOK {
		t.Errorf("expected 200 from /debug/pprof/, got %d", wPprof.Code)
	}

	// Test GET /debug/pprof/goroutine
	wPprofG := httptest.NewRecorder()
	reqPprofG, _ := http.NewRequest(http.MethodGet, "/debug/pprof/goroutine?debug=1", nil)
	r.ServeHTTP(wPprofG, reqPprofG)
	if wPprofG.Code != http.StatusOK {
		t.Errorf("expected 200 from /debug/pprof/goroutine, got %d", wPprofG.Code)
	}

	// Test GET /debug/pprof/heap
	wPprofH := httptest.NewRecorder()
	reqPprofH, _ := http.NewRequest(http.MethodGet, "/debug/pprof/heap", nil)
	r.ServeHTTP(wPprofH, reqPprofH)
	if wPprofH.Code != http.StatusOK {
		t.Errorf("expected 200 from /debug/pprof/heap, got %d", wPprofH.Code)
	}
}

func TestTraceRecordingAndHeader(t *testing.T) {
	r := server.SetupRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/debug/system", nil)
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
	if trace.Path != "/debug/system" {
		t.Errorf("expected trace path /debug/system, got %s", trace.Path)
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

func TestWSStatsProviderDecoupled(t *testing.T) {
	mockCalled := false
	handlers.SetWSStatsProvider(func() (int, int) {
		mockCalled = true
		return 42, 7
	})

	r := gin.New()
	r.GET("/debug/system", handlers.DebugSystemSummary)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/debug/system", nil)
	r.ServeHTTP(w, req)

	if !mockCalled {
		t.Errorf("expected WSStatsProvider callback to be called")
	}

	var res struct {
		WebSocket struct {
			ConnectedClients int `json:"connected_clients"`
			ActiveChannels   int `json:"active_channels"`
		} `json:"websocket"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.WebSocket.ConnectedClients != 42 || res.WebSocket.ActiveChannels != 7 {
		t.Errorf("expected 42 clients and 7 channels, got %+v", res.WebSocket)
	}
}
