package db

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestCombatSessionsLegacySchemaUpgrade(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL migration regression")
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	schema := fmt.Sprintf("combat_revision_%d", time.Now().UnixNano())
	if _, err := tx.Exec("CREATE SCHEMA " + schema + "; SET LOCAL search_path TO " + schema); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`CREATE TABLE players (id UUID PRIMARY KEY);
		INSERT INTO players VALUES ('00000000-0000-0000-0000-000000000001');
		CREATE TABLE combat_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			state JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (player_id)
		);
		INSERT INTO combat_sessions (player_id) VALUES ('00000000-0000-0000-0000-000000000001');`); err != nil {
		t.Fatal(err)
	}
	schemaExec = tx
	defer func() { schemaExec = nil }()
	if err := schemaCombat(); err != nil {
		t.Fatal(err)
	}
	if err := schemaCombat(); err != nil {
		t.Fatalf("second schema pass: %v", err)
	}
	var revision int64
	var status, mode string
	if err := tx.QueryRow(`SELECT revision, status, mode FROM combat_sessions WHERE player_id='00000000-0000-0000-0000-000000000001'`).Scan(&revision, &status, &mode); err != nil {
		t.Fatal(err)
	}
	if revision != 0 || status != "abandoned" || mode != "legacy" {
		t.Fatalf("legacy session: revision=%d status=%q mode=%q", revision, status, mode)
	}
	if _, err := tx.Exec(`INSERT INTO combat_sessions (player_id, mode, initial_state, loadout)
		VALUES ('00000000-0000-0000-0000-000000000001', 'practice', '{}', '{}')`); err != nil {
		t.Fatalf("new session after legacy row: %v", err)
	}
	var id, playerID, activeStatus string
	var participantID sql.NullString
	var initialState, loadout, commands, pendingPlans []byte
	var deadline sql.NullTime
	if err := tx.QueryRow(`SELECT id, player_id, participant_id, revision, status, initial_state, loadout, commands, pending_plans, deadline_at
		FROM combat_sessions WHERE (player_id = $1 OR participant_id = $1) AND status = 'active'`,
		"00000000-0000-0000-0000-000000000001").Scan(&id, &playerID, &participantID, &revision, &activeStatus,
		&initialState, &loadout, &commands, &pendingPlans, &deadline); err != nil {
		t.Fatalf("loadCombatSession query: %v", err)
	}
	if revision != 0 || activeStatus != "active" {
		t.Fatalf("new session: revision=%d status=%q", revision, activeStatus)
	}
}
