package server

import "net/http"

type RealtimeWTHandler any

// WithRealtimeWT preserves signature compatibility while passing requests directly to the API handler.
func WithRealtimeWT(api http.Handler, _ ...any) http.Handler {
	if api == nil {
		return http.NewServeMux()
	}
	return api
}
