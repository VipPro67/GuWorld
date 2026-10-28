package db

import "fmt"

// schemaRealtime owns durable idempotency records for the authoritative world.
// Rewards are inserted through domain services inside the same transaction that
// claims the idempotency key, so retries cannot duplicate items/Gu.
func schemaRealtime() error {
	if err := execSchema("realtime_loot_grants", `
	CREATE TABLE IF NOT EXISTS realtime_loot_grants (
		idempotency_key TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		death_id TEXT NOT NULL,
		recipient_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		template_id VARCHAR(80) NOT NULL,
		reward_version INTEGER NOT NULL,
		payload JSONB NOT NULL DEFAULT '{}'::jsonb,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("player_world_state_v2", `
	CREATE TABLE IF NOT EXISTS player_world_state_v2 (
		player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		hp DOUBLE PRECISION NOT NULL,
		shield DOUBLE PRECISION NOT NULL DEFAULT 0,
		map_version VARCHAR(80) NOT NULL DEFAULT '',
		content_version VARCHAR(80) NOT NULL DEFAULT '',
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("player_world_state_combat", `
	CREATE TABLE IF NOT EXISTS player_world_state_combat (
		player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		zone_id VARCHAR(120) NOT NULL DEFAULT '',
		instance_id VARCHAR(120) NOT NULL DEFAULT '',
		combat_state JSONB NOT NULL DEFAULT '{}'::jsonb,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("realtime_combat_instances", `
	CREATE TABLE IF NOT EXISTS realtime_combat_instances (
		id UUID PRIMARY KEY,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		mode VARCHAR(24) NOT NULL CHECK (mode IN ('training','arena','tournament','secret_realm')),
		status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','cancelled')),
		zone_id VARCHAR(80) NOT NULL,
		origin_zone_id VARCHAR(80) NOT NULL,
		payload JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		ended_at TIMESTAMPTZ
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_realtime_combat_active_player
	ON realtime_combat_instances (player_id) WHERE status = 'active';
	CREATE INDEX IF NOT EXISTS idx_realtime_combat_active_id
	ON realtime_combat_instances (id, player_id) WHERE status = 'active';`); err != nil {
		return err
	}
	if err := execSchema("realtime_combat_sessions", `
	CREATE TABLE IF NOT EXISTS realtime_combat_sessions (
		id UUID PRIMARY KEY,
		mode VARCHAR(24) NOT NULL CHECK (mode IN ('arena','tournament','sect_war','secret_realm','ascension')),
		status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','cancelled')),
		zone_id VARCHAR(80) NOT NULL,
		origin_zone_id VARCHAR(80) NOT NULL,
		payload JSONB NOT NULL DEFAULT '{}'::jsonb,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		ended_at TIMESTAMPTZ
	);
	CREATE TABLE IF NOT EXISTS realtime_combat_session_participants (
		session_id UUID NOT NULL REFERENCES realtime_combat_sessions(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active','eliminated','completed','cancelled')),
		payload JSONB NOT NULL DEFAULT '{}'::jsonb,
		disconnected_at TIMESTAMPTZ,
		PRIMARY KEY (session_id, player_id)
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_realtime_combat_session_active_player
	ON realtime_combat_session_participants (player_id) WHERE status = 'active';
	CREATE INDEX IF NOT EXISTS idx_realtime_combat_session_participant_lookup
	ON realtime_combat_session_participants (session_id, player_id);
	CREATE TABLE IF NOT EXISTS realtime_combat_session_events (
		id BIGSERIAL PRIMARY KEY,
		session_id UUID NOT NULL REFERENCES realtime_combat_sessions(id) ON DELETE CASCADE,
		event_id BIGINT NOT NULL,
		payload JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_realtime_combat_session_events
	ON realtime_combat_session_events (session_id, event_id);`); err != nil {
		return err
	}
	// Keep pre-existing databases compatible with new private realtime match
	// modes; CREATE TABLE IF NOT EXISTS does not refresh a CHECK constraint.
	if err := execSchema("realtime_combat_sessions_mode_check", `
	ALTER TABLE realtime_combat_sessions DROP CONSTRAINT IF EXISTS realtime_combat_sessions_mode_check;
	ALTER TABLE realtime_combat_sessions ADD CONSTRAINT realtime_combat_sessions_mode_check
	CHECK (mode IN ('arena','tournament','sect_war','secret_realm','ascension'));`); err != nil {
		return err
	}
	if err := execSchema("realtime_loot_grants_death_idx", `
	CREATE INDEX IF NOT EXISTS idx_realtime_loot_grants_death
	ON realtime_loot_grants (zone_id, death_id);`); err != nil {
		return err
	}
	if err := execSchema("realtime_loot_grants_recipient_idx", `
	CREATE INDEX IF NOT EXISTS idx_realtime_loot_grants_recipient
	ON realtime_loot_grants (recipient_id, created_at DESC);`); err != nil {
		return fmt.Errorf("realtime loot grant recipient index: %w", err)
	}
	return nil
}
