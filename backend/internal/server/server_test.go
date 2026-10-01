package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type memorySyncer struct {
	buf bytes.Buffer
}

func (m *memorySyncer) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

func (m *memorySyncer) Sync() error {
	return nil
}

func TestRequestLogging_SuccessInfoLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mem := &memorySyncer{}
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), mem, zapcore.InfoLevel)
	origLogger := logger.L
	logger.L = zap.New(core)
	defer func() {
		logger.L = origLogger
	}()

	r := gin.New()
	r.Use(requestLogging())
	r.Use(monitor.Middleware())

	r.GET("/test-success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test-success", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	lines := strings.Split(strings.TrimSpace(mem.buf.String()), "\n")
	var completedLog map[string]interface{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			continue
		}
		if parsed["msg"] == "http_request_completed" {
			completedLog = parsed
		}
	}

	if completedLog == nil {
		t.Fatal("expected http_request_completed log at InfoLevel for status 200")
	}
	if completedLog["level"] != "info" {
		t.Errorf("expected level info, got %v", completedLog["level"])
	}
}

func TestRequestLogging_ErrorResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mem := &memorySyncer{}
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), mem, zapcore.DebugLevel)
	origLogger := logger.L
	logger.L = zap.New(core)
	defer func() {
		logger.L = origLogger
	}()

	r := gin.New()
	r.Use(requestLogging())
	r.Use(monitor.Middleware())

	r.POST("/test-error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
	})

	body := bytes.NewBufferString(`{"x":5003,"y":5004}`)
	req, _ := http.NewRequest(http.MethodPost, "/test-error", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	lines := strings.Split(strings.TrimSpace(mem.buf.String()), "\n")
	var errorLog, completedLog map[string]interface{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			continue
		}
		if parsed["msg"] == "http_request_error" {
			errorLog = parsed
		}
		if parsed["msg"] == "http_request_completed" {
			completedLog = parsed
		}
	}

	if errorLog == nil {
		t.Fatal("expected http_request_error log")
	}
	if completedLog == nil {
		t.Fatal("expected http_request_completed log")
	}

	// Verify http_request_error
	if errorLog["trace_id"] == nil || errorLog["trace_id"] == "" {
		t.Errorf("expected non-empty trace_id in http_request_error, got %v", errorLog["trace_id"])
	}
	if errorLog["error"] != "Transaction commit failure" {
		t.Errorf("expected error field 'Transaction commit failure', got %v", errorLog["error"])
	}
	if errorLog["response_body"] == nil {
		t.Error("expected response_body in http_request_error")
	}
	if errorLog["request_body"] == nil {
		t.Error("expected request_body in http_request_error")
	}

	// Verify http_request_completed
	if completedLog["trace_id"] == nil || completedLog["trace_id"] == "" {
		t.Errorf("expected non-empty trace_id in http_request_completed, got %v", completedLog["trace_id"])
	}
	if completedLog["error"] != "Transaction commit failure" {
		t.Errorf("expected error field in http_request_completed, got %v", completedLog["error"])
	}
	if completedLog["response_body"] == nil {
		t.Error("expected response_body in http_request_completed")
	}
	if completedLog["request_body"] == nil {
		t.Error("expected request_body in http_request_completed")
	}
}

func TestAdminRoutePermissionInventory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	declared := make(map[string]string)
	for _, route := range adminRouteSpecs() {
		key := route.method + " /api/admin" + route.path
		if route.permission == "" {
			t.Errorf("%s has no declared permission", key)
		}
		if route.permission == "admin" {
			t.Errorf("%s uses the forbidden catch-all admin permission", key)
		}
		if previous, exists := declared[key]; exists {
			t.Errorf("%s is declared more than once (%q and %q)", key, previous, route.permission)
		}
		declared[key] = route.permission
	}

	for _, route := range SetupRouter(nil).Routes() {
		if !strings.HasPrefix(route.Path, "/api/admin/") {
			continue
		}
		key := route.Method + " " + route.Path
		if _, exists := declared[key]; !exists {
			t.Errorf("%s is registered without a permission declaration", key)
		}
		delete(declared, key)
	}
	for key, permission := range declared {
		t.Errorf("permission inventory contains unregistered route %s (%s)", key, permission)
	}
}

func TestAdminMutationRoutesDeclareAuditAction(t *testing.T) {
	mutationMethods := map[string]bool{
		http.MethodPost: true, http.MethodPut: true,
		http.MethodPatch: true, http.MethodDelete: true,
	}
	declared := make(map[string]string)
	for _, route := range adminRouteSpecs() {
		if !mutationMethods[route.method] {
			continue
		}
		key := route.method + " /api/admin" + route.path
		if strings.TrimSpace(route.auditAction) == "" {
			t.Errorf("%s has no declared audit action", key)
		}
		if !strings.HasPrefix(route.auditAction, "admin.") {
			t.Errorf("%s audit action %q must use admin. namespace", key, route.auditAction)
		}
		if strings.TrimSpace(route.auditTargetType) == "" {
			t.Errorf("%s has no declared audit target type", key)
		}
		declared[key] = route.auditAction
	}
	for _, route := range SetupRouter(nil).Routes() {
		if strings.HasPrefix(route.Path, "/api/admin/") && mutationMethods[route.Method] {
			key := route.Method + " " + route.Path
			if _, ok := declared[key]; !ok {
				t.Errorf("registered mutation %s has no audit declaration", key)
			}
		}
	}
}

func TestAdminRoutePermissionsUseExpectedSensitivePolicies(t *testing.T) {
	want := map[string]string{
		http.MethodGet + " /monitor/system":             "monitoring.read",
		http.MethodPost + " /monitor/requests/reset":    "monitoring.reset",
		http.MethodGet + " /players/:id":                "players.read",
		http.MethodPut + " /players/:id":                "players.update",
		http.MethodPost + " /players/:id/items":         "players.inventory.grant",
		http.MethodPost + " /players/:id/gus":           "players.inventory.grant",
		http.MethodPost + " /data/import":               "game_data.import_all",
		http.MethodPost + " /data/import/commit":        "game_data.import_all",
		http.MethodPost + " /data/import/:table":        "game_data.import",
		http.MethodPost + " /data/import/:table/commit": "game_data.import",
		http.MethodPost + " /data/export/:table":        "game_data.export",
		http.MethodPost + " /clear-cache":               "cache.reload",
		http.MethodPost + " /clear-redis-cache":         "cache.flush",
		http.MethodPut + " /players/:id/stones":         "players.resources.adjust",
		http.MethodPut + " /players/:id/roles":          "admin_roles.assign",
		http.MethodGet + " /audit-logs":                 "audit.read",
	}

	for _, route := range adminRouteSpecs() {
		key := route.method + " " + route.path
		if permission, exists := want[key]; exists {
			if route.permission != permission {
				t.Errorf("%s permission = %q, want %q", key, route.permission, permission)
			}
			delete(want, key)
		}
	}
	for key, permission := range want {
		t.Errorf("missing sensitive route permission %s = %s", key, permission)
	}
}

func TestSensitiveAdminRoutesDeclareSafetyCeremony(t *testing.T) {
	want := map[string]struct{ reason, reauth, idempotency bool }{
		http.MethodPost + " /clear-redis-cache":         {true, true, true},
		http.MethodPost + " /data/import/commit":        {true, false, true},
		http.MethodPost + " /data/import/:table/commit": {true, false, true},
		http.MethodPut + " /players/:id/stones":         {true, false, true},
		http.MethodPut + " /players/:id/roles":          {true, true, true},
	}
	for _, route := range adminRouteSpecs() {
		key := route.method + " " + route.path
		expected, ok := want[key]
		if !ok {
			continue
		}
		if route.reasonRequired != expected.reason || route.reauthenticationRequired != expected.reauth || route.idempotencyKeyRequired != expected.idempotency || route.confirmationTarget == "" {
			t.Errorf("%s has incomplete safety policy: %+v", key, route)
		}
		delete(want, key)
	}
	for key := range want {
		t.Errorf("missing sensitive route %s", key)
	}
	for _, route := range adminRouteSpecs() {
		if route.method == http.MethodDelete && (!route.reasonRequired || route.confirmationTarget == "" || !strings.HasSuffix(route.permission, ".delete")) {
			t.Errorf("DELETE %s lacks delete permission/reason/target confirmation", route.path)
		}
	}
}
