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
