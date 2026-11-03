package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestWithRealtimeWTPassesThroughNonConnect(t *testing.T) {
	apiCalled := false
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(http.StatusTeapot)
	})
	handlerCalled := false
	h := WithRealtimeWT(api, func(http.ResponseWriter, *http.Request, uuid.UUID, string) {
		handlerCalled = true
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/wt", nil))

	if !apiCalled || handlerCalled {
		t.Fatalf("apiCalled=%v handlerCalled=%v, want api only", apiCalled, handlerCalled)
	}
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestWithRealtimeWTRequiresToken(t *testing.T) {
	handlerCalled := false
	h := WithRealtimeWT(nil, func(http.ResponseWriter, *http.Request, uuid.UUID, string) {
		handlerCalled = true
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodConnect, RealtimeWTPath, nil))

	if handlerCalled {
		t.Fatal("handler must not run without a token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestWithRealtimeWTRejectsMalformedAuthorization(t *testing.T) {
	h := WithRealtimeWT(nil, func(http.ResponseWriter, *http.Request, uuid.UUID, string) {
		t.Fatal("handler must not run with a malformed header")
	})

	req := httptest.NewRequest(http.MethodConnect, RealtimeWTPath, nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
