package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateGuCompositionContributionsAcceptsBlockV2Array(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	effects := json.RawMessage(`[{"id":"ACTIVE_SKILL","target":{"selector":"TARGET"},"trigger":"ON_ACTION","operations":[{"op":"DAMAGE","value":10,"sub_type":"FLAT"}]}]`)
	grants := json.RawMessage(`{"schema_version":1,"grants":[{"effect_block_id":"ACTIVE_SKILL","operation_index":0,"capability":"COMBAT.DAMAGE.FLAT.SINGLE_TARGET"}]}`)
	if _, ok := validateGuCompositionContributions(ctx, effects, grants, "GU_TEST"); !ok {
		t.Fatalf("valid array Block v2 rejected: %s", recorder.Body.String())
	}
}
