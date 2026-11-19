package handlers

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildAuditFiltersSharesActorSubjectAndSearchParams(t *testing.T) {
	playerID := uuid.New()
	success := false
	where, args := buildAuditFilters("pet.capture", "player_pet", &playerID, &success, "wolf")
	want := " WHERE (player_id = $1 OR affected_player_id = $1) AND action = $2 AND entity_type = $3 AND success = $4 AND (action ILIKE $5 OR request_summary ILIKE $5 OR entity_id ILIKE $5 OR error_message ILIKE $5)"
	if where != want {
		t.Fatalf("unexpected where clause:\n got: %s\nwant: %s", where, want)
	}
	if len(args) != 5 {
		t.Fatalf("expected five bound arguments, got %d", len(args))
	}
	if args[0] != playerID || args[1] != "pet.capture" || args[2] != "player_pet" || args[3] != false || args[4] != "%wolf%" {
		t.Fatalf("unexpected filter args: %#v", args)
	}

	adminWhere, adminArgs := buildAuditFilters("", "", nil, nil, "")
	if adminWhere != " WHERE 1=1" || len(adminArgs) != 0 || strings.Contains(adminWhere, "player_id") {
		t.Fatalf("unexpected unscoped admin filters: %q %#v", adminWhere, adminArgs)
	}
}
