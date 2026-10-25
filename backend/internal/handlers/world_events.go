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
	ID            string `json:"id"`
	Username      string `json:"username"`
	AvatarURL     string `json:"avatar_url"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	Region        string `json:"region_id"`
	MapID         string `json:"world_map_id"`
	InstanceID    string `json:"map_instance_id,omitempty"`
	TS            int64  `json:"ts"`
	ScopeResolved bool   `json:"-"`
}

var worldPosHub = struct {
	sync.Mutex
	subs map[string]map[chan WorldPosEvent]struct{}
}{subs: map[string]map[chan WorldPosEvent]struct{}{}}

func subscribeWorldPos(playerID string) chan WorldPosEvent {
	EnsureWorldPosLoaded(playerID)
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
	EnsureWorldPosLoaded(payload.ID)
	if payload.MapID == "" {
		payload.MapID = worldMapIDForRegion(payload.Region)
	}
	if previous, ok := worldPosIndex.Get(payload.ID); ok {
		if !payload.ScopeResolved && payload.InstanceID == "" && previous.InstanceID != "" && previous.MapID == payload.MapID {
			payload.InstanceID = previous.InstanceID
		}
	}

	worldPosIndex.Set(WorldPosEntry{
		ID:         payload.ID,
		Username:   payload.Username,
		AvatarURL:  payload.AvatarURL,
		X:          payload.X,
		Y:          payload.Y,
		Region:     payload.Region,
		MapID:      payload.MapID,
		InstanceID: payload.InstanceID,
		TS:         payload.TS,
	})

	nearby := worldPosIndex.NearbyInScope(payload.X, payload.Y, WorldPosInterestRadius, payload.ID, payload.MapID, payload.InstanceID)

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

// EnsureWorldPosLoaded makes sure a player has an entry in the position
// index so interest-managed fan-out and nearby queries can find it immediately.
func EnsureWorldPosLoaded(playerID string) {
	if _, ok := worldPosIndex.Get(playerID); ok {
		return
	}
	var e WorldPosEntry
	var ts time.Time
	err := db.DB.QueryRow(`
		SELECT p.id, p.username, COALESCE(p.avatar_url, ''), p.x, p.y, COALESCE(p.current_region, ''),
		       COALESCE(rwm.world_map_id,'world_map'), COALESCE(p.map_instance_id::text,''), COALESCE(p.last_mana_update, NOW())
		FROM players p LEFT JOIN region_world_maps rwm ON rwm.region_id=p.current_region
		WHERE p.id = $1
	`, playerID).Scan(&e.ID, &e.Username, &e.AvatarURL, &e.X, &e.Y, &e.Region, &e.MapID, &e.InstanceID, &ts)
	if err != nil {
		return
	}
	e.TS = ts.UnixMilli()
	worldPosIndex.Set(e)
}
