package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
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

func TestCompileWithDB(t *testing.T) {
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", "cKsQD[ER&.z=1Xe$,+9KaH+#0vmen^~h")
	}
	err := db.Connect()
	if err != nil {
		t.Skipf("cannot connect to db: %v", err)
	}
	defer db.DB.Close()

	if err := models.LoadGuRegistry(db.DB); err != nil {
		t.Fatalf("LoadGuRegistry failed: %v", err)
	}

	guEntries := make([]realtimeeffects.GuEntry, 0, len(models.GuRegistry))
	for id, ranks := range models.GuRegistry {
		for rank, item := range ranks {
			guEntries = append(guEntries, realtimeeffects.GuEntry{
				ID: id, Name: item.Name, Rank: rank, Type: item.Type, PrimaryRole: item.PrimaryRole,
				ManaCost: item.BaseManaCost,
			})
		}
	}
	t.Logf("guEntries count: %d", len(guEntries))

	// Check if GU_FLYING_SWORD rank 7 is in guEntries
	var foundFlying7 bool
	for _, e := range guEntries {
		if e.ID == "GU_FLYING_SWORD" && e.Rank == 7 {
			foundFlying7 = true
			break
		}
	}
	t.Logf("foundFlying7: %v", foundFlying7)

	catalog, err := realtimeeffects.LoadProfileCatalogs("../../data/mmo/gu/transpiled_profiles.json")
	if err != nil {
		t.Fatalf("LoadProfileCatalogs failed: %v", err)
	}

	_, compileErr := realtimeeffects.CompileCatalog(catalog, guEntries, realtimeeffects.DefaultMVP())
	if compileErr != nil {
		t.Fatalf("CompileCatalog failed: %v", compileErr)
	}
}
