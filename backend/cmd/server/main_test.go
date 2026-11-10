package main

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestEnvBool(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"True", true},
		{"yes", true},
		{"on", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"off", false},
		{"", false},
	}
	for _, tc := range tests {
		os.Setenv("TEST_ENV_BOOL", tc.val)
		if got := envBool("TEST_ENV_BOOL"); got != tc.want {
			t.Errorf("envBool(%q) = %v, want %v", tc.val, got, tc.want)
		}
	}
	os.Unsetenv("TEST_ENV_BOOL")
}

func TestRealtimeOriginChecker(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,https://app.guworld.com")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	checker := realtimeOriginChecker()

	// Same origin (no origin header) is allowed
	req1 := httptest.NewRequest("GET", "/api/ws", nil)
	if !checker(req1) {
		t.Error("expected empty origin to be allowed")
	}

	// Allowed origin
	req2 := httptest.NewRequest("GET", "/api/ws", nil)
	req2.Header.Set("Origin", "http://localhost:3000")
	if !checker(req2) {
		t.Error("expected http://localhost:3000 to be allowed")
	}

	// Disallowed origin
	req3 := httptest.NewRequest("GET", "/api/ws", nil)
	req3.Header.Set("Origin", "http://malicious.com")
	if checker(req3) {
		t.Error("expected http://malicious.com to be rejected")
	}
}
