package handlers

import (
	"sync"
)

// WorldPosEvent is broadcast whenever a player's position changes on the
// shared world map so nearby clients can glide markers smoothly instead of
// waiting for a poll. Clients keep the full viewport-filter again locally.
type WorldPosEvent struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	TS        int64  `json:"ts"`
}

var worldPosHub = struct {
	sync.Mutex
	subs map[chan WorldPosEvent]struct{}
}{subs: map[chan WorldPosEvent]struct{}{}}

func subscribeWorldPos() chan WorldPosEvent {
	ch := make(chan WorldPosEvent, 1)
	worldPosHub.Lock()
	defer worldPosHub.Unlock()
	worldPosHub.subs[ch] = struct{}{}
	return ch
}

func unsubscribeWorldPos(ch chan WorldPosEvent) {
	worldPosHub.Lock()
	defer worldPosHub.Unlock()
	delete(worldPosHub.subs, ch)
	close(ch)
}

func broadcastWorldPos(payload WorldPosEvent) {
	worldPosHub.Lock()
	defer worldPosHub.Unlock()
	for ch := range worldPosHub.subs {
		select {
		case ch <- payload:
		default:
			select {
			case <-ch:
			default:
			}
			ch <- payload
		}
	}
}