package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

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
