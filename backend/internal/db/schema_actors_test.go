package db

import (
	"strings"
	"testing"
)

func TestLegacyActorProjectionUsesMigrationIdentity(t *testing.T) {
	for name, statement := range map[string]string{
		"pet":         nonPlayerActorsPetProjectionSQL,
		"variant_npc": nonPlayerActorsVariantNPCProjectionSQL,
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(statement, "ON CONFLICT (legacy_source, legacy_id)") {
				t.Fatal("projection must be idempotent on canonical legacy identity")
			}
			if strings.Contains(statement, "ON CONFLICT (id)") {
				t.Fatal("projection must not hide cross-source actor ID collisions")
			}
			if !strings.Contains(statement, "version = non_player_actors.version + 1") {
				t.Fatal("projection refresh must advance optimistic-lock version")
			}
		})
	}
}

func TestLegacyActorCleanupCoversBothSources(t *testing.T) {
	for _, source := range []string{"PLAYER_PET", "APERTURE_VARIANT_NPC"} {
		if !strings.Contains(nonPlayerActorsLegacyCleanupSQL, source) {
			t.Fatalf("cleanup does not cover %s", source)
		}
	}
}
