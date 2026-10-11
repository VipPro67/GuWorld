package db

// Martial Arena (Đại Hội Võ Thuật) schema — periodic solo 1v1 tournament
// grouped by cultivation rank. Mirrors death_arena but combatant = one player
// (no teams), and no registration phase (players are auto-snapshotted online).
func schemaMartialArena() error {
	if err := execSchema("martial_tournaments", `
	CREATE TABLE IF NOT EXISTS martial_tournaments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		status VARCHAR(20) NOT NULL DEFAULT 'LOCKED' CHECK (status IN ('LOCKED','IN_PROGRESS','FINISHED')),
		rank INT NOT NULL,
		scheduled_at TIMESTAMPTZ NOT NULL,
		started_at TIMESTAMPTZ,
		finished_at TIMESTAMPTZ,
		total_players INT NOT NULL DEFAULT 0,
		current_round INT NOT NULL DEFAULT 0,
		total_rounds INT NOT NULL DEFAULT 0,
		champion_player_id UUID,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_tournaments_status", `
	CREATE INDEX IF NOT EXISTS idx_martial_tournaments_status ON martial_tournaments(status);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_tournaments_scheduled", `
	CREATE INDEX IF NOT EXISTS idx_martial_tournaments_scheduled ON martial_tournaments(scheduled_at);`); err != nil {
		return err
	}

	if err := execSchema("martial_combatants", `
	CREATE TABLE IF NOT EXISTS martial_combatants (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tournament_id UUID NOT NULL REFERENCES martial_tournaments(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		seed_number INT NOT NULL,
		rank INT NOT NULL,
		sub_rank INT NOT NULL,
		eliminated BOOLEAN NOT NULL DEFAULT FALSE,
		eliminated_at_round INT,
		final_placement INT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE(tournament_id, player_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_combatants_tournament", `
	CREATE INDEX IF NOT EXISTS idx_martial_combatants_tournament ON martial_combatants(tournament_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_combatants_player", `
	CREATE INDEX IF NOT EXISTS idx_martial_combatants_player ON martial_combatants(player_id, tournament_id);`); err != nil {
		return err
	}

	if err := execSchema("martial_matches", `
	CREATE TABLE IF NOT EXISTS martial_matches (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tournament_id UUID NOT NULL REFERENCES martial_tournaments(id) ON DELETE CASCADE,
		round_number INT NOT NULL,
		bracket_position INT NOT NULL,
		combatant_a_id UUID NOT NULL REFERENCES martial_combatants(id),
		combatant_b_id UUID REFERENCES martial_combatants(id),
		winner_combatant_id UUID REFERENCES martial_combatants(id),
		combat_log JSONB,
		status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','BYE','IN_PROGRESS','FINISHED')),
		reward_stones INT NOT NULL DEFAULT 0,
		reward_items JSONB DEFAULT '[]',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		finished_at TIMESTAMPTZ
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_matches_round", `
	CREATE INDEX IF NOT EXISTS idx_martial_matches_round ON martial_matches(tournament_id, round_number);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_matches_status", `
	CREATE INDEX IF NOT EXISTS idx_martial_matches_status ON martial_matches(tournament_id, status, round_number, bracket_position);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_matches_combatant_a", `
	CREATE INDEX IF NOT EXISTS idx_martial_matches_combatant_a ON martial_matches(combatant_a_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_matches_combatant_b", `
	CREATE INDEX IF NOT EXISTS idx_martial_matches_combatant_b ON martial_matches(combatant_b_id);`); err != nil {
		return err
	}

	if err := execSchema("martial_champion_rewards", `
	CREATE TABLE IF NOT EXISTS martial_champion_rewards (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tournament_id UUID NOT NULL REFERENCES martial_tournaments(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		placement INT NOT NULL,
		combatant_id UUID NOT NULL REFERENCES martial_combatants(id),
		title VARCHAR(100),
		reward_stones INT NOT NULL DEFAULT 0,
		reward_immortal_stones INT NOT NULL DEFAULT 0,
		reward_gu_id VARCHAR(100),
		reward_gu_rank INT,
		reward_items JSONB DEFAULT '[]',
		claimed BOOLEAN NOT NULL DEFAULT FALSE,
		claimed_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_rewards_tournament", `
	CREATE INDEX IF NOT EXISTS idx_martial_rewards_tournament ON martial_champion_rewards(tournament_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_martial_rewards_player", `
	CREATE INDEX IF NOT EXISTS idx_martial_rewards_player ON martial_champion_rewards(player_id);`); err != nil {
		return err
	}

	return nil
}
