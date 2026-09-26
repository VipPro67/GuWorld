package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateAndNormalizeAdminGuEffectsRejectsInvalidData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	raw := json.RawMessage(`{"schema_version":2,"blocks":[{"id":"BAD","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STATUS","status_id":"NOT_REAL","turns":1,"chance":1,"stack_policy":"STACK"}]}]}`)

	if validateAndNormalizeAdminGuEffects(ctx, "GU_TEST", &raw) {
		t.Fatal("invalid effect was accepted")
	}
	if recorder.Code != 400 {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	var response struct {
		Error  string `json:"error"`
		Issues []struct {
			Path string `json:"path"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error == "" || len(response.Issues) == 0 || response.Issues[0].Path != "$.blocks[0].operations[0].status_id" {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestValidateAndNormalizeAdminGuEffectsAcceptsBlockV2(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	raw := json.RawMessage(`{"schema_version":2,"blocks":[{"id":"HIT","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":12,"unit":"FLAT"}]}]}`)

	if !validateAndNormalizeAdminGuEffects(ctx, "GU_TEST", &raw) {
		t.Fatal("valid Block v2 effect was rejected")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, ok := envelope["schema_version"]; !ok {
		t.Fatal("schema_version was lost")
	}
}

func TestValidateAndNormalizeAdminGuEffectsRejectsLegacyNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	raw := json.RawMessage(`{"active":{"reveal_all_stealth":true}}`)

	if validateAndNormalizeAdminGuEffects(ctx, "GU_TEST", &raw) {
		t.Fatal("legacy namespace was accepted")
	}
}
