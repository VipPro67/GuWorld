package db

import (
	"os"
	"strings"
	"testing"
)

func TestLegacyActorMigrationPreservesIDsAndCanonicalFields(t *testing.T) {
	for name, statement := range map[string]string{
		"pet":         legacyPetsToActorsSQL,
		"variant_npc": legacyVariantNPCsToActorsSQL,
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(statement, "SELECT p.id") && !strings.Contains(statement, "SELECT n.id") {
				t.Fatal("migration must preserve the legacy UUID as the actor ID")
			}
			if strings.Contains(statement, "ON CONFLICT") {
				t.Fatal("migration must reject unrelated actor ID collisions")
			}
			if !strings.Contains(statement, "DELETE FROM non_player_actors") {
				t.Fatal("migration must replace a previous compatibility projection")
			}
		})
	}
	if !strings.Contains(legacyPetsToActorsSQL, "formation_slot") {
		t.Fatal("pet migration must preserve formation membership")
	}
	for _, field := range []string{"path", "role"} {
		if !strings.Contains(legacyVariantNPCsToActorsSQL, field) {
			t.Fatalf("variant NPC migration must preserve %s", field)
		}
	}
}

func TestActorSchemaDeclaresNPCGameplayStorage(t *testing.T) {
	for _, fragment := range []string{"actor_gu_equipment", "UNIQUE (player_gu_id)", "UNIQUE (actor_id, slot)", "idx_non_player_actors_variant_formation"} {
		found := strings.Contains(legacyPetsToActorsSQL, fragment) || strings.Contains(legacyVariantNPCsToActorsSQL, fragment)
		// The migration constants intentionally do not own new schema DDL; retain a
		// direct source-level guard for accidental removal from schemaActors.
		if found {
			continue
		}
		source, err := os.ReadFile("schema_actors.go")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(source), fragment) {
			t.Fatalf("actor schema missing %q", fragment)
		}
	}
}
