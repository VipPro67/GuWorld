package db

// schemaRealtime cleans up and drops legacy realtime tables.
func schemaRealtime() error {
	return execSchema("drop_legacy_realtime_tables", `
		DROP TABLE IF EXISTS realtime_combat_session_events CASCADE;
		DROP TABLE IF EXISTS realtime_combat_session_participants CASCADE;
		DROP TABLE IF EXISTS realtime_combat_sessions CASCADE;
		DROP TABLE IF EXISTS realtime_combat_instances CASCADE;
		DROP TABLE IF EXISTS realtime_loot_grants CASCADE;
		DROP TABLE IF EXISTS player_world_state_combat CASCADE;
		DROP TABLE IF EXISTS player_world_state_v2 CASCADE;
	`)
}
