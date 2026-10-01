package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPreviouslyUnguardedAdminHandlersRejectUnauthenticatedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		method  string
		route   string
		request string
		handler gin.HandlerFunc
	}{
		{name: "list gu history", method: http.MethodGet, route: "/api/admin/gu-history", request: "/api/admin/gu-history", handler: AdminListGuHistory},
		{name: "list player pets", method: http.MethodGet, route: "/api/admin/players/:id/pets", request: "/api/admin/players/player-1/pets", handler: AdminListPlayerPets},
		{name: "delete player pet", method: http.MethodDelete, route: "/api/admin/players/:id/pets/:pet_id", request: "/api/admin/players/player-1/pets/pet-1", handler: AdminDeletePlayerPet},
		{name: "create player pet", method: http.MethodPost, route: "/api/admin/players/:id/pets", request: "/api/admin/players/player-1/pets", handler: AdminCreatePlayerPet},
		{name: "update player pet", method: http.MethodPut, route: "/api/admin/players/:id/pets/:pet_id", request: "/api/admin/players/player-1/pets/pet-1", handler: AdminUpdatePlayerPet},
		{name: "list roles", method: http.MethodGet, route: "/api/admin/roles", request: "/api/admin/roles", handler: AdminListRoles},
		{name: "get player roles", method: http.MethodGet, route: "/api/admin/players/:id/roles", request: "/api/admin/players/player-1/roles", handler: AdminGetPlayerRoles},
		{name: "update player roles", method: http.MethodPut, route: "/api/admin/players/:id/roles", request: "/api/admin/players/player-1/roles", handler: AdminUpdatePlayerRoles},
		{name: "batch metrics", method: http.MethodGet, route: "/api/admin/batch-metrics", request: "/api/admin/batch-metrics", handler: GetBatchMetrics},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Handle(tt.method, tt.route, tt.handler)
			request := httptest.NewRequest(tt.method, tt.request, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
			}
		})
	}
}
