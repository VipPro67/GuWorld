package handlers

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// TestUpdateRefinePlayerResourcesEquippedGusIsJSONB locks the write shape of
// the post-refine players UPDATE: players.equipped_gus is a JSONB column, so
// EquippedGusList must be persisted through its driver.Valuer (JSON array),
// never as a Postgres array literal. A session resolve for a player with
// non-empty equipped_gus used to fail every attempt with
// "invalid input syntax for type json", leaving the session stuck ACTIVE and
// re-broadcasting the world-chat announcement on every retry.
//
// Requires a live database (set DB_PASSWORD); skips otherwise.
func TestUpdateRefinePlayerResourcesEquippedGusIsJSONB(t *testing.T) {
	if os.Getenv("DB_PASSWORD") == "" {
		t.Setenv("DB_PASSWORD", "gu_pass_123")
	}
	if err := db.Connect(); err != nil {
		t.Skipf("cannot connect to db: %v", err)
	}
	defer db.DB.Close()

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	playerID := uuid.New()
	expiry := time.Now().Add(10 * time.Minute).UTC()
	player := models.Player{}
	player.ID = playerID
	player.Username = "refine-update-test"
	player.EquippedGus = models.EquippedGusList{"inst-1", "inst-2"}
	player.Statuses = models.StatusList{{ID: "refine_fatigue", ExpiresAt: &expiry, Source: "refine", Value: 0}}
	player.PathMasteries = models.PathMasteriesMap{"refinement": 42}

	if _, err := tx.Exec(`
		INSERT INTO players (id, username, aptitude_type, aptitude_percent, current_mana, max_mana, current_region)
		VALUES ($1, $2, 'thien', 50, 30, 40, 'tan_thu_coc_nam_cuong')`,
		playerID, player.Username); err != nil {
		t.Fatalf("insert fixture player: %v", err)
	}

	if err := updateRefinePlayerResources(tx, &player); err != nil {
		t.Fatalf("updateRefinePlayerResources: %v", err)
	}

	var equippedRaw []byte
	var statusesRaw []byte
	var masteriesRaw []byte
	if err := tx.QueryRow(
		`SELECT equipped_gus, statuses, path_masteries FROM players WHERE id = $1`, playerID,
	).Scan(&equippedRaw, &statusesRaw, &masteriesRaw); err != nil {
		t.Fatalf("read back player row: %v", err)
	}

	var equipped []string
	if err := json.Unmarshal(equippedRaw, &equipped); err != nil {
		t.Fatalf("equipped_gus must be valid JSON: %v (raw: %s)", err, equippedRaw)
	}
	want := []string{"inst-1", "inst-2"}
	if len(equipped) != len(want) || equipped[0] != want[0] || equipped[1] != want[1] {
		t.Fatalf("equipped_gus roundtrip = %v, want %v", equipped, want)
	}

	var statuses models.StatusList
	if err := json.Unmarshal(statusesRaw, &statuses); err != nil {
		t.Fatalf("statuses must be valid JSON: %v (raw: %s)", err, statusesRaw)
	}
	if !statuses.Has("refine_fatigue") {
		t.Fatalf("statuses roundtrip lost refine_fatigue: %s", statusesRaw)
	}

	var masteries map[string]interface{}
	if err := json.Unmarshal(masteriesRaw, &masteries); err != nil {
		t.Fatalf("path_masteries must be valid JSON: %v (raw: %s)", err, masteriesRaw)
	}
	if masteries["refinement"] != float64(42) {
		t.Fatalf("path_masteries roundtrip = %v, want refinement=42", masteries)
	}
}
