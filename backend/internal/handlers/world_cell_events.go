package handlers

import "sync"

// WorldCellEvent is broadcast when a persistent map cell (boss / high resource
// / safe point) changes state: cleared or respawned. These events are rare and
// tiny, so fan-out is global (like chat) rather than interest-managed.
type WorldCellEvent struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	CellType  string `json:"cell_type"`
	ContentID string `json:"content_id"`
}

// cellEvent pairs an SSE event name with its payload.
type cellEvent struct {
	Name string
	Data WorldCellEvent
}

const (
	cellEventCleared = "world.cell.cleared"
	cellEventRespawn = "world.cell.respawn"
)

var worldCellHub = struct {
	sync.Mutex
	subs map[chan cellEvent]struct{}
}{subs: map[chan cellEvent]struct{}{}}

func subscribeWorldCell() chan cellEvent {
	ch := make(chan cellEvent, 8)
	worldCellHub.Lock()
	defer worldCellHub.Unlock()
	worldCellHub.subs[ch] = struct{}{}
	return ch
}

func unsubscribeWorldCell(ch chan cellEvent) {
	worldCellHub.Lock()
	defer worldCellHub.Unlock()
	delete(worldCellHub.subs, ch)
	close(ch)
}

func broadcastWorldCell(name string, data WorldCellEvent) {
	worldCellHub.Lock()
	defer worldCellHub.Unlock()
	for ch := range worldCellHub.subs {
		select {
		case ch <- cellEvent{Name: name, Data: data}:
		default:
		}
	}
}
