package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
)

func TestAdminMutationAuditRecordsFailureAndDestructiveAction(t *testing.T) {
	monitor.GlobalAdminOperations.Reset()
	defer monitor.GlobalAdminOperations.Reset()
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("player_id", uuid.New()); c.Next() })
	router.POST("/", AdminMutationAudit("admin.commit_import_table", "game_data", "game_data.import", true), func(c *gin.Context) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid data"})
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)))

	snapshot := monitor.GlobalAdminOperations.Snapshot()
	if snapshot.FailedMutationsByAction["admin.commit_import_table"] != 1 ||
		snapshot.ImportFailuresByAction["admin.commit_import_table"] != 1 ||
		snapshot.DestructiveByAction["admin.commit_import_table"] != 1 {
		t.Fatalf("unexpected operation metrics: %#v", snapshot)
	}
}

func TestSanitizedAuditChangeRedactsSecretsRecursively(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/api/admin/test", strings.NewReader(`{"name":"safe","password":"p","nested":{"access_token":"t","payment_secret":"s","clientSecret":"camel","otp":"123"},"reason":"maintenance"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	change, reason := sanitizedAuditChange(readAndRestoreAuditBody(c), c)
	encoded, err := json.Marshal(change)
	if err != nil {
		t.Fatal(err)
	}
	got := string(encoded)
	for _, secret := range []string{`"p"`, `"t"`, `"s"`, `"camel"`, `"123"`} {
		if strings.Contains(got, secret) {
			t.Errorf("sanitized change leaked %s: %s", secret, got)
		}
	}
	if !strings.Contains(got, `"name":"safe"`) {
		t.Errorf("safe field missing: %s", got)
	}
	if reason != "maintenance" {
		t.Fatalf("reason = %q, want maintenance", reason)
	}
	if string(readBodyForTest(c)) == "" {
		t.Fatal("request body was not restored")
	}
}

func readBodyForTest(c *gin.Context) []byte {
	body := readAndRestoreAuditBody(c)
	return body
}

func TestAuditTargetIDPrefersStablePathIdentifiers(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: "player-1"}, {Key: "pet_id", Value: "pet-2"}}
	if got := auditTargetID(c, nil, nil); got != "pet-2" {
		t.Fatalf("target ID = %q, want most-specific nested target", got)
	}
}
