package notify

import (
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
// Failures are logged and returned. Most callers treat notifications as
// best-effort, but transaction callers must handle the error because PostgreSQL
// aborts the transaction after a failed statement.
//
// When q is a *sql.DB the insert is retried up to 3 times on lock-timeout
// (55P03) or deadlock (40P01) errors, since these are transient contention
// issues caused by concurrent combat/hunger transactions holding player row
// locks (the notifications FK requires a KEY SHARE lock on players).
// When q is a *sql.Tx, a savepoint isolates an insert failure so the caller's
// transaction remains usable.
func Send(q Querier, playerID uuid.UUID, ntype, title, content string) error {
	if q == nil {
		logger.S.Errorf("notify: nil querier, skipping player=%s type=%s", playerID, ntype)
		return sql.ErrConnDone
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
	if db, ok := q.(*sql.DB); ok {
		const maxAttempts = 3
		var err error
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			_, err = db.Exec(insertSQL, n.ID, n.PlayerID, n.Type, n.Title, n.Content, n.CreatedAt)
			if err == nil {
				push(playerID, n)
				return nil
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
		return err
	}

	// Keep a failed best-effort insert from poisoning a caller-owned PostgreSQL
	// transaction. Using the caller's transaction also avoids waiting on player
	// row locks held by that same transaction while validating the foreign key.
	if tx, ok := q.(*sql.Tx); ok {
		if _, err := tx.Exec("SAVEPOINT notify_send"); err != nil {
			return err
		}
		if _, err := tx.Exec(insertSQL, n.ID, n.PlayerID, n.Type, n.Title, n.Content, n.CreatedAt); err != nil {
			logger.S.Errorf("notify: insert failed player=%s type=%s: %v", playerID, ntype, err)
			if _, rollbackErr := tx.Exec("ROLLBACK TO SAVEPOINT notify_send"); rollbackErr != nil {
				return rollbackErr
			}
			if _, releaseErr := tx.Exec("RELEASE SAVEPOINT notify_send"); releaseErr != nil {
				return releaseErr
			}
			return err
		}
		if _, err := tx.Exec("RELEASE SAVEPOINT notify_send"); err != nil {
			return err
		}
		// Do not publish before the owner commits: doing so can expose a row that
		// is not visible yet, or a notification for a transaction that rolls back.
		return nil
	}

	// Fallback for any other Querier: single attempt, no retry.
	if _, err := q.Exec(insertSQL, n.ID, n.PlayerID, n.Type, n.Title, n.Content, n.CreatedAt); err != nil {
		logger.S.Errorf("notify: insert failed player=%s type=%s: %v", playerID, ntype, err)
		return err
	}
	push(playerID, n)
	return nil
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
	_ = cache.PublishEventQuick("notif:"+playerID.String(), n)
}
