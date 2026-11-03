package server

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// RealtimeWTPath is the HTTP/3 CONNECT endpoint that opens a WebTransport session.
const RealtimeWTPath = "/api/wt"

// RealtimeWTHandler upgrades an authenticated request to a WebTransport
// session. Identity comes from the auth check and the query tab id, never
// from the world join payload.
type RealtimeWTHandler func(w http.ResponseWriter, r *http.Request, playerID uuid.UUID, tabID string)

// WithRealtimeWT routes WebTransport CONNECT requests to handle before they
// reach the gin API handler.
//
// The endpoint cannot be registered on the gin engine: gin wraps the
// ResponseWriter in its own implementation, so the http3.Settingser assertion
// inside webtransport-go panics and the browser reports "Opening handshake
// failed". Dispatching here keeps the raw HTTP/3 ResponseWriter intact.
func WithRealtimeWT(api http.Handler, handle RealtimeWTHandler) http.Handler {
	if api == nil {
		api = http.NewServeMux()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect || r.URL.Path != RealtimeWTPath {
			api.ServeHTTP(w, r)
			return
		}

		identity, status, message := middleware.AuthenticateRequest(r)
		if identity == nil {
			logger.S.Warnf("realtime_wt_auth_failed status=%d", status)
			http.Error(w, message, status)
			return
		}

		tabID := strings.TrimSpace(r.URL.Query().Get("tab_id"))
		if len(tabID) > 100 {
			tabID = ""
		}
		if tabID != "" {
			middleware.ClaimActiveTab(identity.PlayerID, tabID)
		}
		logger.S.Infof("realtime_wt_connect player_id=%s tab=%t", identity.PlayerID, tabID != "")
		handle(w, r, identity.PlayerID, tabID)
	})
}
