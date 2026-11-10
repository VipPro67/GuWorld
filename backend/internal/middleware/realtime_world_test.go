package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRequireLegacyWorldModeAllowsUnowned(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })

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
}
