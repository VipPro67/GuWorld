package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/authz"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

type stubPrincipalResolver struct {
	principal authz.AdminPrincipal
	err       error
	called    bool
}

func TestAdminReasonRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, body string
		want       int
		downstream bool
	}{
		{"missing", `{}`, http.StatusBadRequest, false},
		{"blank", `{"reason":"  "}`, http.StatusBadRequest, false},
		{"present", `{"reason":"scheduled maintenance"}`, http.StatusNoContent, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			router := gin.New()
			router.POST("/action", AdminReasonRequired(), func(c *gin.Context) {
				called = true
				c.Status(http.StatusNoContent)
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/action", strings.NewReader(tt.body)))
			if response.Code != tt.want || called != tt.downstream {
				t.Fatalf("status=%d called=%v, want status=%d called=%v", response.Code, called, tt.want, tt.downstream)
			}
		})
	}
	t.Run("header", func(t *testing.T) {
		router := gin.New()
		router.POST("/action", AdminReasonRequired(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
		request := httptest.NewRequest(http.MethodPost, "/action", strings.NewReader(`{}`))
		request.Header.Set("X-Admin-Reason", "cleanup")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	})
}

func TestAdminConfirmationAndIdempotencyRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/players/:id", func(c *gin.Context) {
		c.Set("player_id", uuid.MustParse("00000000-0000-0000-0000-000000000001"))
		c.Next()
	}, AdminConfirmationRequired(func(c *gin.Context) string { return c.Param("id") }), AdminIdempotencyKeyRequired(), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodPost, "/players/player-7", strings.NewReader(`{"confirmation_target":"player-7"}`))
	request.Header.Set("Idempotency-Key", "change-player-7-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("valid ceremony status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/players/player-7", strings.NewReader(`{"confirmation_target":"player-8"}`))
	request.Header.Set("Idempotency-Key", "change-player-7-2")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("mismatched target status=%d, want 400", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/players/player-7", strings.NewReader(`{"confirmation_target":"player-7"}`))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency key status=%d, want 400", response.Code)
	}
}

func TestAuthRequiredReturnsServiceUnavailableWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalDB := db.DB
	db.DB = nil
	t.Cleanup(func() { db.DB = originalDB })

	playerID := uuid.New()
	token, err := auth.GenerateToken(playerID, "admin", "session")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	router := gin.New()
	router.GET("/check", AuthRequired(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/check", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
}

func (s *stubPrincipalResolver) Resolve(_ context.Context, _ uuid.UUID) (authz.AdminPrincipal, error) {
	s.called = true
	return s.principal, s.err
}

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

func TestRequirePermissionResponsesAndPrincipalContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	playerID := uuid.New()

	tests := []struct {
		name         string
		setPlayer    bool
		resolver     *stubPrincipalResolver
		wantStatus   int
		wantHandler  bool
		wantResolved bool
	}{
		{name: "unauthenticated", resolver: &stubPrincipalResolver{}, wantStatus: http.StatusUnauthorized},
		{name: "database unavailable", setPlayer: true, resolver: &stubPrincipalResolver{err: errors.New("db down")}, wantStatus: http.StatusServiceUnavailable, wantResolved: true},
		{name: "permission missing", setPlayer: true, resolver: &stubPrincipalResolver{principal: authz.AdminPrincipal{PlayerID: playerID, Permissions: map[string]struct{}{}}}, wantStatus: http.StatusForbidden, wantResolved: true},
		{name: "permission granted", setPlayer: true, resolver: &stubPrincipalResolver{principal: authz.AdminPrincipal{PlayerID: playerID, Roles: []string{authz.SuperAdminRole}, Permissions: map[string]struct{}{authz.AdminAccessPermission: {}}}}, wantStatus: http.StatusNoContent, wantHandler: true, wantResolved: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downstreamCalled := false
			router := gin.New()
			if tt.setPlayer {
				router.Use(func(c *gin.Context) {
					c.Set("player_id", playerID)
					c.Next()
				})
			}
			router.GET("/check", requirePermission(tt.resolver, authz.AdminAccessPermission), func(c *gin.Context) {
				downstreamCalled = true
				principal, ok := GetAdminPrincipal(c)
				if !ok || principal.PlayerID != playerID {
					t.Fatal("authorized handler did not receive the resolved principal")
				}
				requestPrincipal, ok := authz.PrincipalFromContext(c.Request.Context())
				if !ok || requestPrincipal.PlayerID != playerID {
					t.Fatal("principal was not attached to request context")
				}
				c.Status(http.StatusNoContent)
			})

			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/check", nil))
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if downstreamCalled != tt.wantHandler {
				t.Fatalf("downstream called = %v, want %v", downstreamCalled, tt.wantHandler)
			}
			if tt.resolver.called != tt.wantResolved {
				t.Fatalf("resolver called = %v, want %v", tt.resolver.called, tt.wantResolved)
			}
		})
	}
}
