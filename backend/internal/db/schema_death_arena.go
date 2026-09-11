package db

// Death Arena schema for weekly 6v6 elimination tournament.
func schemaDeathArena() error {
	// Tournament table
	if err := execSchema("death_arena_tournaments", `
	CREATE TABLE IF NOT EXISTS death_arena_tournaments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		status VARCHAR(20) NOT NULL DEFAULT 'REGISTRATION' CHECK (status IN ('REGISTRATION','LOCKED','IN_PROGRESS','FINISHED')),
		scheduled_at TIMESTAMPTZ NOT NULL,
		registration_started_at TIMESTAMPTZ NOT NULL,
		registration_closed_at TIMESTAMPTZ NOT NULL,
		started_at TIMESTAMPTZ,
		finished_at TIMESTAMPTZ,
		total_players INT NOT NULL DEFAULT 0,
		total_teams INT NOT NULL DEFAULT 0,
		current_round INT NOT NULL DEFAULT 0,
		total_rounds INT NOT NULL DEFAULT 0,
		champion_team_id UUID,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_tournaments_status", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_tournaments_status ON death_arena_tournaments(status);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_tournaments_scheduled", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_tournaments_scheduled ON death_arena_tournaments(scheduled_at);`); err != nil {
		return err
	}

	// Teams table
	if err := execSchema("death_arena_teams", `
	CREATE TABLE IF NOT EXISTS death_arena_teams (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tournament_id UUID NOT NULL REFERENCES death_arena_tournaments(id) ON DELETE CASCADE,
		team_name VARCHAR(100) NOT NULL,
		seed_number INT NOT NULL,
		eliminated BOOLEAN NOT NULL DEFAULT FALSE,
		eliminated_at_round INT,
		final_placement INT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_teams_tournament", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_teams_tournament ON death_arena_teams(tournament_id);`); err != nil {
		return err
	}

	// Team members table
	if err := execSchema("death_arena_team_members", `
	CREATE TABLE IF NOT EXISTS death_arena_team_members (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		team_id UUID NOT NULL REFERENCES death_arena_teams(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		current_hp DOUBLE PRECISION NOT NULL,
		max_hp DOUBLE PRECISION NOT NULL,
		current_mana DOUBLE PRECISION NOT NULL,
		max_mana DOUBLE PRECISION NOT NULL,
		eliminated BOOLEAN NOT NULL DEFAULT FALSE,
		eliminated_at_round INT,
		joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE(team_id, player_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_team_members_team", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_team_members_team ON death_arena_team_members(team_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_team_members_player", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_team_members_player ON death_arena_team_members(player_id);`); err != nil {
		return err
	}

	// Matches table
	if err := execSchema("death_arena_matches", `
	CREATE TABLE IF NOT EXISTS death_arena_matches (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tournament_id UUID NOT NULL REFERENCES death_arena_tournaments(id) ON DELETE CASCADE,
		round_number INT NOT NULL,
		bracket_position INT NOT NULL,
		team_a_id UUID NOT NULL REFERENCES death_arena_teams(id),
		team_b_id UUID REFERENCES death_arena_teams(id),
		winner_team_id UUID REFERENCES death_arena_teams(id),
		combat_log JSONB,
		status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','BYE','IN_PROGRESS','FINISHED')),
		reward_stones_per_member INT NOT NULL DEFAULT 0,
		reward_items JSONB DEFAULT '[]',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		finished_at TIMESTAMPTZ
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_matches_tournament", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_matches_tournament ON death_arena_matches(tournament_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_matches_round", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_matches_round ON death_arena_matches(tournament_id, round_number);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_matches_team_a", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_matches_team_a ON death_arena_matches(team_a_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_matches_team_b", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_matches_team_b ON death_arena_matches(team_b_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_matches_tourn_status_round", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_matches_tourn_status_round ON death_arena_matches(tournament_id, status, round_number, bracket_position);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_tm_player_team", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_tm_player_team ON death_arena_team_members(player_id, team_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_tourn_status_sched", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_tourn_status_sched ON death_arena_tournaments(status, scheduled_at DESC);`); err != nil {
		return err
	}

	// Champion rewards table
	if err := execSchema("death_arena_champion_rewards", `
	CREATE TABLE IF NOT EXISTS death_arena_champion_rewards (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tournament_id UUID NOT NULL REFERENCES death_arena_tournaments(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		placement INT NOT NULL,
		team_id UUID NOT NULL REFERENCES death_arena_teams(id),
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

	if err := execSchema("idx_death_arena_champion_rewards_tournament", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_champion_rewards_tournament ON death_arena_champion_rewards(tournament_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_death_arena_champion_rewards_player", `
	CREATE INDEX IF NOT EXISTS idx_death_arena_champion_rewards_player ON death_arena_champion_rewards(player_id);`); err != nil {
		return err
	}

	// Add death_arena_title column to players
	if err := execSchema("players_add_death_arena_title", `
	ALTER TABLE players ADD COLUMN IF NOT EXISTS death_arena_title VARCHAR(100) DEFAULT '';`); err != nil {
		return err
	}

	return nil
}