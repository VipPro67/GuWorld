package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRecipeCatalogConditionalRequestKeepsCachedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recipes := []models.RefiningRecipe{{ID: "recipe_GU_TEST_1_1", TargetGuID: "GU_TEST", TargetRank: 1, Variant: 1}}

	first := httptest.NewRecorder()
	firstContext, _ := gin.CreateTestContext(first)
	firstContext.Request = httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
	writeRecipeCatalogResponseAtRevision(firstContext, recipes, 7)
	if first.Code != http.StatusOK {
		t.Fatalf("first response status = %d, want 200", first.Code)
	}
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first response did not include an ETag")
	}

	revalidated := httptest.NewRecorder()
	revalidatedContext, _ := gin.CreateTestContext(revalidated)
	revalidatedContext.Request = httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
	revalidatedContext.Request.Header.Set("If-None-Match", etag)
	writeRecipeCatalogResponseAtRevision(revalidatedContext, recipes, 7)
	revalidatedContext.Writer.WriteHeaderNow()
	if revalidated.Code != http.StatusNotModified {
		t.Fatalf("conditional response status = %d, want 304 (request=%q response=%q)", revalidated.Code, revalidatedContext.GetHeader("If-None-Match"), revalidated.Header().Get("ETag"))
	}
	if revalidated.Body.Len() != 0 {
		t.Fatalf("304 response body = %q, want empty", revalidated.Body.String())
	}
}

func TestRecipeCatalogAdminInvalidationChangesRevision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recipes := []models.RefiningRecipe{{ID: "recipe_GU_TEST_1_1", TargetGuID: "GU_TEST", TargetRank: 1, Variant: 1}}

	before := httptest.NewRecorder()
	beforeContext, _ := gin.CreateTestContext(before)
	beforeContext.Request = httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
	beforeRevision := currentRecipeCatalogRevision()
	writeRecipeCatalogResponseAtRevision(beforeContext, recipes, beforeRevision)
	oldETag := before.Header().Get("ETag")
	oldRevision := before.Header().Get("X-Catalog-Revision")

	// Recipe admin create/update/delete handlers call this only after a successful write.
	afterRevision := invalidateRecipeCatalog()

	after := httptest.NewRecorder()
	afterContext, _ := gin.CreateTestContext(after)
	afterContext.Request = httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
	afterContext.Request.Header.Set("If-None-Match", oldETag)
	writeRecipeCatalogResponseAtRevision(afterContext, recipes, afterRevision)
	if after.Code != http.StatusOK {
		t.Fatalf("response after invalidation status = %d, want 200", after.Code)
	}
	if got := after.Header().Get("X-Catalog-Revision"); got == oldRevision {
		t.Fatalf("catalog revision did not change: %q", got)
	}
	if got := after.Header().Get("ETag"); got == oldETag {
		t.Fatalf("ETag did not change after catalog invalidation: %q", got)
	}

	var decoded []models.RefiningRecipe
	if err := json.Unmarshal(after.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(decoded) != 1 || decoded[0].ID != recipes[0].ID {
		t.Fatalf("response recipes = %#v, want cached recipe retained", decoded)
	}
}
