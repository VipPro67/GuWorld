package middleware

import (
	"strconv"
	"testing"
	"time"
)

func TestLocalSessionFallbackHandlesEmptyExpiredAndOverflow(t *testing.T) {
	localSessionMu.Lock()
	localSessionFallback = map[string]localSessionEntry{}
	localSessionMu.Unlock()

	if _, ok := getLocalSession("p1"); ok {
		t.Fatal("empty cache returned a hit")
	}

	setLocalSession("p1", "")
	if _, ok := getLocalSession("p1"); ok {
		t.Fatal("empty session id should not be cached")
	}

	setLocalSession("p1", "s1")
	if got, ok := getLocalSession("p1"); !ok || got != "s1" {
		t.Fatalf("cached session = (%q,%v), want (s1,true)", got, ok)
	}

	localSessionMu.Lock()
	localSessionFallback["p1"] = localSessionEntry{id: "s1", expires: time.Now().Add(-time.Second)}
	localSessionMu.Unlock()
	if _, ok := getLocalSession("p1"); ok {
		t.Fatal("expired entry returned a hit")
	}

	for i := 0; i < localSessionFallbackMax; i++ {
		setLocalSession(strconv.Itoa(i), "s")
	}
	setLocalSession("overflow", "s")
	if got, ok := getLocalSession("overflow"); !ok || got != "s" {
		t.Fatalf("entry missing after overflow reset: (%q,%v)", got, ok)
	}
}
