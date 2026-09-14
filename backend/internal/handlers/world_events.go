package handlers

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

// WorldPosBroadcaster allows injection of websocket targeted broadcast without import cycle
var WorldPosBroadcaster func(channel string, payload []byte, targets map[uuid.UUID]struct{})

// WorldPosEvent is broadcast whenever a player's position changes on the
// shared world map so nearby clients can glide markers smoothly instead of
// waiting for a poll. Fan-out is interest-managed: an event is delivered only
// to subscribers (players) whose current position lies within
// WorldPosInterestRadius of the mover, so traffic scales with local density
// instead of the total number of connected clients.
type WorldPosEvent struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Region    string `json:"region_id"`
	TS        int64  `json:"ts"`
}

var worldPosHub = struct {
	sync.Mutex
	subs map[string]map[chan WorldPosEvent]struct{}
}{subs: map[string]map[chan WorldPosEvent]struct{}{}}

func subscribeWorldPos(playerID string) chan WorldPosEvent {
	ensureWorldPosLoaded(playerID)
	ch := make(chan WorldPosEvent, 1)
	worldPosHub.Lock()
	defer worldPosHub.Unlock()
	if worldPosHub.subs[playerID] == nil {
		worldPosHub.subs[playerID] = map[chan WorldPosEvent]struct{}{}
	}
	worldPosHub.subs[playerID][ch] = struct{}{}
	return ch
}

func unsubscribeWorldPos(playerID string, ch chan WorldPosEvent) {
	worldPosHub.Lock()
	defer worldPosHub.Unlock()
	if m := worldPosHub.subs[playerID]; m != nil {
		delete(m, ch)
		if len(m) == 0 {
			delete(worldPosHub.subs, playerID)
		}
	}
	close(ch)
}

// broadcastWorldPos upserts the mover into the position index (so interest
// resolution is always based on the freshest position) and fans the event out
// to subscribers within interest radius via WebSocket.
func broadcastWorldPos(payload WorldPosEvent) {
	playerID, _ := uuid.Parse(payload.ID)
	_ = playerID // suppress unused warning if not needed

	worldPosIndex.Set(WorldPosEntry{
		ID:        payload.ID,
		Username:  payload.Username,
		AvatarURL: payload.AvatarURL,
		X:         payload.X,
		Y:         payload.Y,
		Region:    payload.Region,
		TS:        payload.TS,
	})

	nearby := worldPosIndex.Nearby(payload.Region, payload.X, payload.Y, WorldPosInterestRadius, payload.ID)

	if WorldPosBroadcaster != nil {
		targets := make(map[uuid.UUID]struct{}, len(nearby))
		for _, e := range nearby {
			if nid, err := uuid.Parse(e.ID); err == nil {
				targets[nid] = struct{}{}
			}
		}
		WorldPosBroadcaster("world:pos", mustMarshal(payload), targets)
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// ensureWorldPosLoaded makes sure a subscriber has an entry in the position
// index so interest-managed fan-out can find it. Positions are loaded from the
// database on first use; movement broadcasts then keep it fresh.
func ensureWorldPosLoaded(playerID string) {
	if _, ok := worldPosIndex.Get(playerID); ok {
		return
	}
	var e WorldPosEntry
	var ts time.Time
	err := db.DB.QueryRow(`
		SELECT id, username, COALESCE(avatar_url, ''), x, y, COALESCE(current_region, ''), last_mana_update
		FROM players
		WHERE id = $1
	`, playerID).Scan(&e.ID, &e.Username, &e.AvatarURL, &e.X, &e.Y, &e.Region, &ts)
	if err != nil {
		return
	}
	e.TS = ts.UnixMilli()
	worldPosIndex.Set(e)
}
