package storage

import (
	"testing"
)

func TestLoadR2ConfigOK(t *testing.T) {
	t.Setenv("CF_ENDPOINT", "https://1bfe206e356bd8c26b732cce6e2241a1.r2.cloudflarestorage.com")
	t.Setenv("CF_BUCKET", "gu-world")
	t.Setenv("CF_KEY", "testkey")
	t.Setenv("CF_SEC", "testsec")
	t.Setenv("CF_PUBLIC_BASE_URL", "https://cdn.cogioi.io.vn/")
	cfg, err := LoadR2Config()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg.PublicBase != "https://cdn.cogioi.io.vn" {
		t.Fatalf("trailing slash not trimmed: %q", cfg.PublicBase)
	}
}

func TestLoadR2ConfigMissing(t *testing.T) {
	t.Setenv("CF_ENDPOINT", "")
	t.Setenv("CF_BUCKET", "gu-world")
	t.Setenv("CF_KEY", "k")
	t.Setenv("CF_SEC", "s")
	t.Setenv("CF_PUBLIC_BASE_URL", "https://cdn.cogioi.io.vn")
	if _, err := LoadR2Config(); err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}
