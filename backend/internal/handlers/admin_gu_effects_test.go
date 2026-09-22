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
	raw := json.RawMessage(`{"active":{"apply_status":"NOT_REAL"}}`)

	if validateAndNormalizeAdminGuEffects(ctx, &raw) {
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
	if response.Error == "" || len(response.Issues) != 1 || response.Issues[0].Path != "$.active.apply_status[0].id" {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestValidateAndNormalizeAdminGuEffectsCanonicalizesEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	raw := json.RawMessage(`{"damage_flat":12}`)

	if !validateAndNormalizeAdminGuEffects(ctx, &raw) {
		t.Fatal("valid legacy effect was rejected")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope) != 8 {
		t.Fatalf("sections = %d, want 8", len(envelope))
	}
}

func TestValidateAndNormalizeAdminGuEffectsExposesWarnings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	raw := json.RawMessage(`{"active":{"reveal_all_stealth":true}}`)

	if !validateAndNormalizeAdminGuEffects(ctx, &raw) {
		t.Fatal("legacy unwired effect was rejected")
	}
	warnings := guEffectWarnings(ctx)
	if len(warnings) != 1 || warnings[0].Path != "$.active.reveal_all_stealth" {
		t.Fatalf("warnings = %+v", warnings)
	}
}
