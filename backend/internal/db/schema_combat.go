package db

// Combat history and arena ranking schema.

func schemaCombat() error {
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
	if err := execSchema("arena_rankings_points_idx", `
	CREATE INDEX IF NOT EXISTS idx_arena_rankings_points ON arena_rankings (points DESC);`); err != nil {
		return err
	}
	return nil
}
