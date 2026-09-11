package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/cache"
)

// DeathArenaRealtimeEvent represents a WebSocket broadcast payload for death arena
type DeathArenaRealtimeEvent struct {
	Type         string      `json:"type"` // "tournament_started", "match_updated", "round_advanced", "tournament_finished"
	TournamentID string      `json:"tournament_id"`
	Data         interface{} `json:"data,omitempty"`
}

// BroadcastDeathArenaEvent publishes a death arena realtime event via Redis Pub/Sub
func BroadcastDeathArenaEvent(eventType, tournamentID string, data interface{}) {
	_ = cache.PublishEventQuick("death_arena:events", DeathArenaRealtimeEvent{
		Type:         eventType,
		TournamentID: tournamentID,
		Data:         data,
	})
}
