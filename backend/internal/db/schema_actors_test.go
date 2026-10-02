package db

import (
	"os"
	"strings"
	"testing"
)

func TestActorSchemaDeclaresNPCGameplayStorage(t *testing.T) {
	for _, fragment := range []string{"actor_gu_equipment", "UNIQUE (player_gu_id)", "UNIQUE (actor_id, slot)", "idx_non_player_actors_variant_formation"} {
		source, err := os.ReadFile("schema_actors.go")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(source), fragment) {
			t.Fatalf("actor schema missing %q", fragment)
		}
	}
}
