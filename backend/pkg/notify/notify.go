package notify

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Querier abstracts the database transaction or connection needed to insert a
// notification. Both *sql.DB and *sql.Tx satisfy this interface.
type Querier interface {
	Exec(string, ...interface{}) (sql.Result, error)
}

// Notification is a single player-facing notification (bell icon).
type Notification struct {
	ID        string    `json:"id"`
	PlayerID  string    `json:"player_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// Send inserts a notification for a player and publishes it to Redis for WebSocket broadcast.
// Failures are logged but never returned — notifications are best-effort.
//
// When q is a *sql.DB the insert is retried up to 3 times on lock-timeout
// (55P03) or deadlock (40P01) errors, since these are transient contention
// issues caused by concurrent combat/hunger transactions holding player row
// locks (the notifications FK requires a KEY SHARE lock on players).
// When q is a *sql.Tx, retry is skipped because the transaction is already
// aborted after a PostgreSQL error.
func Send(q Querier, playerID uuid.UUID, ntype, title, content string) {
	if q == nil {
		logger.S.Errorf("notify: nil querier, skipping player=%s type=%s", playerID, ntype)
		return
	}
	n := Notification{
		ID:        uuid.New().String(),
		PlayerID:  playerID.String(),
		Type:      ntype,
		Title:     title,
		Content:   content,
		IsRead:    false,
		CreatedAt: time.Now().UTC(),
	}

	const insertSQL = `INSERT INTO notifications (id, player_id, type, title, content, is_read, created_at)
		 VALUES ($1, $2, $3, $4, $5, FALSE, $6)`

	// For bare *sql.DB connections retry on transient lock contention.
	// For *sql.Tx the caller's transaction is already aborted; don't retry.
	if db, ok := q.(*sql.DB); ok {
		const maxAttempts = 3
		var err error
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			_, err = db.Exec(insertSQL, n.ID, n.PlayerID, n.Type, n.Title, n.Content, n.CreatedAt)
			if err == nil {
				push(playerID, n)
				return
			}
			if !isRetryableDBError(err) || attempt == maxAttempts {
				break
			}
			logger.S.Warnf("notify: insert attempt %d/%d failed (retrying) player=%s type=%s: %v",
				attempt, maxAttempts, playerID, ntype, err)
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			// Reuse a fresh UUID so a partial insert on a previous attempt doesn't
			// cause a duplicate-key conflict on retry.
			n.ID = uuid.New().String()
		}
		logger.S.Errorf("notify: insert failed player=%s type=%s: %v", playerID, ntype, err)
		return
	}

	// Fallback for *sql.Tx or any other Querier: single attempt, no retry.
	if _, err := q.Exec(insertSQL, n.ID, n.PlayerID, n.Type, n.Title, n.Content, n.CreatedAt); err != nil {
		logger.S.Errorf("notify: insert failed player=%s type=%s: %v", playerID, ntype, err)
		return
	}
	push(playerID, n)
}

// isRetryableDBError reports whether err is a transient PostgreSQL error that
// is safe to retry: lock timeout (55P03) or deadlock detected (40P01).
func isRetryableDBError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "55P03") || contains(msg, "40P01") ||
		contains(msg, "lock timeout") || contains(msg, "deadlock detected")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func push(playerID uuid.UUID, n Notification) {
	_ = cache.PublishEvent(context.Background(), "notif:"+playerID.String(), n)
}
