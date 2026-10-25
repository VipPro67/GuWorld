package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRequireLegacyWorldModeBlocksRealtimeOwnedCharacter(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })

	playerID := uuid.New()
	SetRealtimeOwnershipChecker(func(id uuid.UUID) bool { return id == playerID })
	t.Cleanup(func() { SetRealtimeOwnershipChecker(nil) })

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("player_id", playerID)
		c.Next()
	})
	handlerCalled := false
	router.POST("/legacy-world-action", RequireLegacyWorldMode(), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/legacy-world-action", nil))
	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), http.StatusConflict)
	}
	if handlerCalled {
		t.Fatal("legacy handler ran for realtime-owned character")
	}
}

func TestRequireLegacyWorldModeAllowsUnownedAndRejectsMissingIdentity(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })
	SetRealtimeOwnershipChecker(func(uuid.UUID) bool { return false })
	t.Cleanup(func() { SetRealtimeOwnershipChecker(nil) })

	playerID := uuid.New()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("player_id", playerID)
		c.Next()
	})
	router.POST("/legacy", RequireLegacyWorldMode(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/legacy", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("unowned character status=%d, want %d", response.Code, http.StatusNoContent)
	}

	unauthenticated := gin.New()
	unauthenticated.POST("/legacy", RequireLegacyWorldMode(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response = httptest.NewRecorder()
	unauthenticated.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/legacy", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing identity status=%d, want %d", response.Code, http.StatusUnauthorized)
	}
}
