package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// RealtimeWTHandler upgrades an authenticated request to a WebTransport
// session. Identity comes from the auth middleware and the query tab id, never
// from the world join payload.
type RealtimeWTHandler func(w http.ResponseWriter, r *http.Request, playerID uuid.UUID, tabID string)

// RegisterRealtimeWT mounts the WebTransport endpoint next to /api/ws so both
// transports can run during development. It is registered after SetupRouter
// because the WebTransport server itself is created in main.
func RegisterRealtimeWT(engine *gin.Engine, handler RealtimeWTHandler) {
	if engine == nil || handler == nil {
		return
	}
	engine.Handle(http.MethodConnect, "/api/wt", middleware.AuthRequired(), func(c *gin.Context) {
		playerID, exists := middleware.GetPlayerID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		tabID := strings.TrimSpace(c.Query("tab_id"))
		if len(tabID) > 100 {
			tabID = ""
		}
		handler(c.Writer, c.Request, playerID, tabID)
	})
}
