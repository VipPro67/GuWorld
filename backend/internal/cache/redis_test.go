package cache

import (
	"crypto/tls"
	"os"
	"testing"
)

func TestNewRedisTLSConfigDefaultsToVerifiedHost(t *testing.T) {
	config, err := newRedisTLSConfig("redis.internal", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if config.ServerName != "redis.internal" {
		t.Fatalf("ServerName = %q, want redis.internal", config.ServerName)
	}
	if config.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %d, want TLS 1.2", config.MinVersion)
	}
	if config.InsecureSkipVerify {
		t.Fatal("Redis TLS certificate verification must not be disabled")
	}
}

func TestNewRedisTLSConfigRejectsInvalidCA(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "invalid-ca-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("not a certificate"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := newRedisTLSConfig("redis.internal", "cache.example.com", file.Name()); err == nil {
		t.Fatal("expected invalid CA file to be rejected")
	}
}

func TestMapMGetResultsPreservesMissingPositions(t *testing.T) {
	got := mapMGetResults([]interface{}{nil, `{"id":"middle"}`, nil, `{"id":"last"}`})
	want := []string{"", `{"id":"middle"}`, "", `{"id":"last"}`}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
