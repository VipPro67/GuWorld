package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/cache"
)

// TournamentRealtimeEvent is the shared Redis Pub/Sub payload for Death Arena and Martial Arena.
type TournamentRealtimeEvent struct {
	Type         string      `json:"type"` // "tournament_started", "match_updated", "round_advanced", "round_intermission", "tournament_finished", "reward_available"
	TournamentID string      `json:"tournament_id"`
	Data         interface{} `json:"data,omitempty"`
}

func broadcastTournamentEvent(channel, eventType, tournamentID string, data interface{}) {
	_ = cache.PublishEventQuick(channel, TournamentRealtimeEvent{
		Type:         eventType,
		TournamentID: tournamentID,
		Data:         data,
	})
}

// BroadcastDeathArenaEvent publishes a death arena realtime event via Redis Pub/Sub.
func BroadcastDeathArenaEvent(eventType, tournamentID string, data interface{}) {
	broadcastTournamentEvent("death_arena:events", eventType, tournamentID, data)
}
