package playerstate

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// countingScanner stands in for *sql.Row so the canonical projection can be
// checked without a database.
type countingScanner struct {
	n   int
	err error
}

func (s *countingScanner) Scan(dest ...any) error {
	s.n = len(dest)
	return s.err
}

func TestScanUsesEveryCanonicalColumn(t *testing.T) {
	scanner := &countingScanner{}
	if _, _, err := Scan(scanner); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if scanner.n != ColumnCount {
		t.Fatalf("Scan destinations = %d, ColumnCount = %d: update Columns, ColumnCount and Scan together", scanner.n, ColumnCount)
	}
}

func TestScanPropagatesDriverError(t *testing.T) {
	want := errors.New("boom")
	if _, _, err := Scan(&countingScanner{err: want}); !errors.Is(err, want) {
		t.Fatalf("Scan error = %v, want %v", err, want)
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(sql.ErrNoRows) {
		t.Fatal("IsNotFound(sql.ErrNoRows) = false")
	}
	if !IsNotFound(errors.Join(errors.New("wrapped"), sql.ErrNoRows)) {
		t.Fatal("IsNotFound should see a wrapped sql.ErrNoRows")
	}
	if IsNotFound(errors.New("other")) {
		t.Fatal("IsNotFound(other) = true")
	}
}

func TestShouldClearActiveCombat(t *testing.T) {
	playerID := uuid.New()
	partyID := uuid.New()
	other := uuid.New()

	cases := []struct {
		name     string
		partyID  *uuid.UUID
		leaderID string
		want     bool
	}{
		{"solo keeps combat", nil, "", false},
		{"party leader keeps combat", &partyID, playerID.String(), false},
		{"party member clears combat", &partyID, other.String(), true},
		{"missing leader keeps combat", &partyID, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldClearActiveCombat(playerID, tc.partyID, tc.leaderID); got != tc.want {
				t.Fatalf("ShouldClearActiveCombat = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExpiredStatusUpdate(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Minute)
	future := now.Add(time.Hour)
	player := &models.Player{}
	player.Statuses = models.StatusList{
		{ID: "expired", ExpiresAt: &past},
		{ID: "permanent"},
		{ID: "active", ExpiresAt: &future},
	}

	original, cleaned, changed := ExpiredStatusUpdate(player, now)
	if !changed {
		t.Fatal("expected expired status cleanup")
	}
	if !strings.Contains(original.(string), "expired") || !strings.Contains(original.(string), "permanent") {
		t.Fatalf("original statuses = %v, want both entries", original)
	}
	if strings.Contains(cleaned.(string), "expired") {
		t.Fatalf("cleaned statuses still contain expired entry: %v", cleaned)
	}
	if len(player.Statuses) != 2 {
		t.Fatalf("player statuses after expiry = %d, want 2", len(player.Statuses))
	}

	permanent := &models.Player{}
	permanent.Statuses = models.StatusList{{ID: "permanent"}}
	_, _, changed = ExpiredStatusUpdate(permanent, now)
	if changed {
		t.Fatal("permanent status should not trigger a repair")
	}
}
