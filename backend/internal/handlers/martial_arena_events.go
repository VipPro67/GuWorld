package handlers

import "github.com/vippro67/GuWorld/backend/internal/cache"

// MartialArenaRealtimeEvent is the WebSocket broadcast payload for martial arena.
type MartialArenaRealtimeEvent struct {
	Type         string      `json:"type"` // "tournament_started", "match_updated", "round_advanced", "round_intermission", "tournament_finished"
	TournamentID string      `json:"tournament_id"`
	Data         interface{} `json:"data,omitempty"`
}

// BroadcastMartialArenaEvent publishes a martial arena realtime event via Redis Pub/Sub.
func BroadcastMartialArenaEvent(eventType, tournamentID string, data interface{}) {
	_ = cache.PublishEventQuick("martial_arena:events", MartialArenaRealtimeEvent{
		Type:         eventType,
		TournamentID: tournamentID,
		Data:         data,
	})
}
