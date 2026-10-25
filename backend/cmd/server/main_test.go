package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
)

func TestResolveRealtimeMapPathFromNestedServerDirectory(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate server source directory")
	}
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })
	if err := os.Chdir(filepath.Dir(sourceFile)); err != nil {
		t.Fatal(err)
	}

	path := resolveRealtimeMapPath("data/mmo/maps", "world_map")
	if filepath.Base(path) != "world_map.json" {
		t.Fatalf("map path=%q, want world_map.json", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("resolved map path %q is unavailable: %v", path, err)
	}
}

func TestResolveRealtimeProfilePathsFromNestedServerDirectory(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate server source directory")
	}
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })
	if err := os.Chdir(filepath.Dir(sourceFile)); err != nil {
		t.Fatal(err)
	}

	resolved := resolveRealtimeProfilePaths("data/mmo/gu/transpiled_profiles.json")
	for _, path := range strings.Split(resolved, ",") {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("resolved profile catalog %q is unavailable: %v", path, err)
		}
	}
	catalog, err := realtimeeffects.LoadProfileCatalogs(resolved)
	if err != nil {
		t.Fatalf("could not load profile catalogs from nested server directory: %v", err)
	}
	if len(catalog.Profiles) != 4255 {
		t.Fatalf("loaded profile count=%d, want 4255", len(catalog.Profiles))
	}
}
