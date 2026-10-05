package db

import (
	"strings"
	"testing"
)

func TestCollectionRevisionTriggerDoesNotUpdatePlayers(t *testing.T) {
	sql := strings.ToLower(playerCollectionRevisionSchema)
	if strings.Contains(sql, "update players") {
		t.Fatal("collection revision trigger must not lock the hot players row")
	}
	if !strings.Contains(sql, "insert into player_collection_revisions") {
		t.Fatal("collection mutation must persist an independently readable revision")
	}
	if !strings.Contains(sql, "mod(txid_current(), 64)") {
		t.Fatal("collection counters must be sharded across concurrent transactions")
	}
	if !strings.Contains(sql, "after delete on players") {
		t.Fatal("deleting a player must clean up its collection counters")
	}
}

func TestCollectionRevisionTableIsExcludedFromDynamicTriggers(t *testing.T) {
	if !strings.Contains(playerCollectionRevisionSchema, "c.conrelid <> 'player_collection_revisions'::regclass") {
		t.Fatal("revision table must not install a recursive ownership trigger")
	}
}

func TestGoogleLoginLookupHasPlayerIndex(t *testing.T) {
	sql := strings.ToLower(playerGoogleIDIndexSchema)
	if !strings.Contains(sql, "on players (google_id)") {
		t.Fatal("Google login lookup must have an index on players.google_id")
	}
	if !strings.Contains(sql, "where google_id is not null") {
		t.Fatal("Google login index must omit players without a linked Google account")
	}
}
