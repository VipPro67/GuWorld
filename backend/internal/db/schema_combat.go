package db

// Combat history and arena ranking schema.

func schemaCombat() error {
	if err := execSchema("combat_sessions", `
	CREATE TABLE IF NOT EXISTS combat_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		participant_id UUID REFERENCES players(id) ON DELETE CASCADE,
		mode VARCHAR(32) NOT NULL,
		status VARCHAR(16) NOT NULL DEFAULT 'active',
		initial_state JSONB NOT NULL,
		loadout JSONB NOT NULL,
		commands JSONB NOT NULL DEFAULT '[]',
		revision BIGINT NOT NULL DEFAULT 0,
		pending_actor VARCHAR(128) NOT NULL DEFAULT '',
		deadline_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("combat_sessions_participant_id_col", `
	ALTER TABLE combat_sessions ADD COLUMN IF NOT EXISTS participant_id UUID REFERENCES players(id) ON DELETE CASCADE;`); err != nil {
		return err
	}
	// CREATE TABLE IF NOT EXISTS does not evolve installations that already
	// have combat_sessions. Ensure the status column exists before its partial
	// indexes are created below.
	if err := execSchema("combat_sessions_status_col", `
	ALTER TABLE combat_sessions ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'active';`); err != nil {
		return err
	}
	if err := execSchema("combat_session_commands", `
	CREATE TABLE IF NOT EXISTS combat_session_commands (
		session_id UUID NOT NULL REFERENCES combat_sessions(id) ON DELETE CASCADE,
		command_id UUID NOT NULL,
		revision BIGINT NOT NULL,
		PRIMARY KEY (session_id, command_id),
		UNIQUE (session_id, revision)
	);`); err != nil {
		return err
	}
	if err := execSchema("infinite_tower_progress", `
	CREATE TABLE IF NOT EXISTS infinite_tower_progress (
		player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		best_floor INTEGER NOT NULL DEFAULT 0,
		best_floor_at TIMESTAMPTZ,
		weekly_losses INTEGER NOT NULL DEFAULT 0,
		week_started_at DATE NOT NULL DEFAULT (date_trunc('week', CURRENT_DATE)::date),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("infinite_tower_best_floor_at_col", `
	ALTER TABLE infinite_tower_progress ADD COLUMN IF NOT EXISTS best_floor_at TIMESTAMPTZ;`); err != nil {
		return err
	}
	if err := execSchema("infinite_tower_best_floor_at_backfill", `
	UPDATE infinite_tower_progress SET best_floor_at = updated_at
	WHERE best_floor > 0 AND best_floor_at IS NULL;`); err != nil {
		return err
	}
	if err := execSchema("combat_sessions_player_active_idx", `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_combat_sessions_player_active
	ON combat_sessions (player_id) WHERE status = 'active';`); err != nil {
		return err
	}
	if err := execSchema("combat_sessions_deadline_idx", `
	CREATE INDEX IF NOT EXISTS idx_combat_sessions_deadline
	ON combat_sessions (deadline_at) WHERE status = 'active';`); err != nil {
		return err
	}
	if err := execSchema("combat_sessions_participant_active_idx", `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_combat_sessions_participant_active
	ON combat_sessions (participant_id) WHERE status = 'active' AND participant_id IS NOT NULL;`); err != nil {
		return err
	}
	if err := execSchema("combat_history", `
	CREATE TABLE IF NOT EXISTS combat_history (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		winner VARCHAR(20) NOT NULL DEFAULT 'Phe Địch',
		logs JSONB NOT NULL DEFAULT '[]',
		summary JSONB NOT NULL DEFAULT '{}',
		enemy_name VARCHAR(100) NOT NULL DEFAULT '',
		enemy_type VARCHAR(50) NOT NULL DEFAULT '',
		region VARCHAR(50) NOT NULL DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		combatants JSONB NOT NULL DEFAULT '[]'
	);`); err != nil {
		return err
	}

	if err := execSchema("arena_rankings", `
	CREATE TABLE IF NOT EXISTS arena_rankings (
		player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		rank INTEGER UNIQUE NOT NULL,
		points INTEGER DEFAULT 1000,
		daily_challenges INTEGER NOT NULL DEFAULT 0,
		last_challenge_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("combat_history_player_created_idx", `
	CREATE INDEX IF NOT EXISTS idx_combat_history_player_created ON combat_history (player_id, created_at DESC);`); err != nil {
		return err
	}
	if err := execSchema("combat_history_created_id_idx", `
	CREATE INDEX IF NOT EXISTS idx_combat_history_created_id ON combat_history (created_at DESC, id);`); err != nil {
		return err
	}
	if err := execSchema("combat_history_winner_created_idx", `
	CREATE INDEX IF NOT EXISTS idx_combat_history_winner_created ON combat_history (winner, created_at DESC);`); err != nil {
		return err
	}
	if err := execSchema("combat_history_pve_cleanup_idx", `
	CREATE INDEX IF NOT EXISTS idx_combat_history_pve_cleanup ON combat_history (created_at, id) WHERE UPPER(enemy_type) <> 'PVP';`); err != nil {
		return err
	}
	if err := execSchema("combat_history_drop_realtime_death_unique_idx", `
	DROP INDEX IF EXISTS idx_combat_history_realtime_death;`); err != nil {
		return err
	}
	if err := execSchema("arena_rankings_points_idx", `
	CREATE INDEX IF NOT EXISTS idx_arena_rankings_points ON arena_rankings (points DESC);`); err != nil {
		return err
	}
	return nil
}
