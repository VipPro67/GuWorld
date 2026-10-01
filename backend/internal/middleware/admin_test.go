package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminRequiredRejectsRequestWithoutAuthenticatedPlayer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	downstreamCalled := false
	router := gin.New()
	router.GET("/api/admin/check", AdminRequired(), func(c *gin.Context) {
		downstreamCalled = true
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/admin/check", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
	if downstreamCalled {
		t.Fatal("admin middleware called the protected handler")
	}
}
