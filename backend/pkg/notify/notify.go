package notify

import (
	"database/sql"
	"sync"
	"time"

	"github.com/google/uuid"
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

// ---------------------------------------------------------------------------
// SSE fan-out hub (mirrors trade_events.go / chat_events.go pattern)
// ---------------------------------------------------------------------------

var hub = struct {
	sync.Mutex
	subs map[uuid.UUID]map[chan Notification]struct{}
}{subs: map[uuid.UUID]map[chan Notification]struct{}{}}

// Subscribe returns a channel that receives new notifications for the player.
func Subscribe(playerID uuid.UUID) chan Notification {
	ch := make(chan Notification, 16)
	hub.Lock()
	defer hub.Unlock()
	if hub.subs[playerID] == nil {
		hub.subs[playerID] = map[chan Notification]struct{}{}
	}
	hub.subs[playerID][ch] = struct{}{}
	return ch
}

// Unsubscribe removes the channel from the hub and closes it.
func Unsubscribe(playerID uuid.UUID, ch chan Notification) {
	hub.Lock()
	defer hub.Unlock()
	delete(hub.subs[playerID], ch)
	if len(hub.subs[playerID]) == 0 {
		delete(hub.subs, playerID)
	}
	close(ch)
}

// Send inserts a notification for a player and pushes it over SSE.
// Failures are logged but never returned — notifications are best-effort.
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
	if _, err := q.Exec(
		`INSERT INTO notifications (id, player_id, type, title, content, is_read, created_at)
		 VALUES ($1, $2, $3, $4, $5, FALSE, $6)`,
		n.ID, n.PlayerID, n.Type, n.Title, n.Content, n.CreatedAt,
	); err != nil {
		logger.S.Errorf("notify: insert failed player=%s type=%s: %v", playerID, ntype, err)
		return
	}
	push(playerID, n)
}

func push(playerID uuid.UUID, n Notification) {
	hub.Lock()
	defer hub.Unlock()
	for ch := range hub.subs[playerID] {
		select {
		case ch <- n:
		default:
			// Buffer full — drop the oldest to make room for the newest.
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- n:
			default:
			}
		}
	}
}
