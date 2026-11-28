package handlers

// BroadcastMartialArenaEvent publishes a martial arena realtime event via Redis Pub/Sub.
func BroadcastMartialArenaEvent(eventType, tournamentID string, data interface{}) {
	broadcastTournamentEvent("martial_arena:events", eventType, tournamentID, data)
}
