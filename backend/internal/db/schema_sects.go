package db

// Sect schema. sect_missions.target_id is polymorphic (item ids for GATHER,
// gu ids for DONATE) so it has no FK; sect_members.master_id is a legacy
// unused column and stays unconstrained.

func schemaSects() error {
	if err := execSchema("sects", `
	CREATE TABLE IF NOT EXISTS sects (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL,
		region VARCHAR(50) NOT NULL,
		vault_gu JSONB NOT NULL DEFAULT '[]',
		vault_materials JSONB NOT NULL DEFAULT '{}',
		vault_recipes JSONB NOT NULL DEFAULT '[]',
		funds BIGINT NOT NULL DEFAULT 0,
		level INTEGER NOT NULL DEFAULT 1,
		require_approval BOOLEAN NOT NULL DEFAULT FALSE,
		require_promotion_approval BOOLEAN NOT NULL DEFAULT FALSE
	);`); err != nil {
		return err
	}

	if err := execSchema("npcs", `
	CREATE TABLE IF NOT EXISTS npcs (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		sect_id VARCHAR(50) REFERENCES sects(id) ON DELETE SET NULL,
		role VARCHAR(50) DEFAULT 'DISCIPLE',
		aptitude_percent INTEGER DEFAULT 50,
		created_at TIMESTAMP DEFAULT now()
	);`); err != nil {
		return err
	}

	if err := execSchema("sect_missions", `
	CREATE TABLE IF NOT EXISTS sect_missions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		title VARCHAR(100) NOT NULL,
		description TEXT NOT NULL,
		mission_type VARCHAR(30) NOT NULL,
		target_id VARCHAR(50) NOT NULL,
		target_amount INTEGER NOT NULL DEFAULT 1,
		reward_contribution INTEGER NOT NULL DEFAULT 0,
		required_member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
		expires_at TIMESTAMP WITHOUT TIME ZONE,
		UNIQUE (sect_id, title)
	);`); err != nil {
		return err
	}

	if err := execSchema("sect_members", `
	CREATE TABLE IF NOT EXISTS sect_members (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE UNIQUE,
		member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
		contribution_points BIGINT NOT NULL DEFAULT 0,
		master_id VARCHAR(50) DEFAULT NULL,
		joined_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		territory_pending DOUBLE PRECISION NOT NULL DEFAULT 0,
		territory_credited_at DATE,
		territory_credited_today INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		return err
	}
	if err := execSchema("sect_members_sect_idx", `
	CREATE INDEX IF NOT EXISTS idx_sect_members_sect ON sect_members (sect_id);`); err != nil {
		return err
	}

	if err := execSchema("sect_join_requests", `
	CREATE TABLE IF NOT EXISTS sect_join_requests (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
		created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		UNIQUE (sect_id, player_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("sect_promotion_requests", `
	CREATE TABLE IF NOT EXISTS sect_promotion_requests (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		current_rank VARCHAR(30) NOT NULL,
		target_rank VARCHAR(30) NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
		created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		UNIQUE (sect_id, player_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("player_sect_missions", `
	CREATE TABLE IF NOT EXISTS player_sect_missions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		mission_id UUID NOT NULL REFERENCES sect_missions(id) ON DELETE CASCADE,
		current_progress INTEGER NOT NULL DEFAULT 0,
		status VARCHAR(30) NOT NULL DEFAULT 'IN_PROGRESS',
		assigned_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		expires_at TIMESTAMP WITHOUT TIME ZONE,
		UNIQUE (player_id, mission_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("player_sect_missions_player_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_sect_missions_player ON player_sect_missions (player_id);`); err != nil {
		return err
	}

	if err := execSchema("sect_mines", `
	CREATE TABLE IF NOT EXISTS sect_mines (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		rarity VARCHAR(50) NOT NULL,
		occupier_sect_id VARCHAR(50) REFERENCES sects(id) ON DELETE SET NULL,
		occupied_at TIMESTAMP WITH TIME ZONE,
		shield_until TIMESTAMP WITH TIME ZONE,
		last_claimed_at TIMESTAMP WITH TIME ZONE,
		pending_funds DOUBLE PRECISION NOT NULL DEFAULT 0,
		pending_materials DOUBLE PRECISION NOT NULL DEFAULT 0
	);`); err != nil {
		return err
	}

	if err := execSchema("sect_mine_defenders", `
	CREATE TABLE IF NOT EXISTS sect_mine_defenders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		mine_id INTEGER NOT NULL REFERENCES sect_mines(id) ON DELETE CASCADE,
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		defender_type VARCHAR(20) NOT NULL DEFAULT 'member',
		player_id UUID REFERENCES players(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL DEFAULT '',
		monster_id VARCHAR(50) REFERENCES monster_templates(id) ON DELETE CASCADE,
		monster_rank INTEGER NOT NULL DEFAULT 1,
		monster_sub_rank INTEGER NOT NULL DEFAULT 1,
		power DOUBLE PRECISION NOT NULL DEFAULT 0,
		assigned_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
	);`); err != nil {
		return err
	}
	if err := execSchema("uq_sect_mine_defender_player", `
	CREATE UNIQUE INDEX IF NOT EXISTS uq_sect_mine_defender_player ON sect_mine_defenders (player_id) WHERE defender_type = 'member' AND player_id IS NOT NULL;`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_mine_defenders_mine", `
	CREATE INDEX IF NOT EXISTS idx_sect_mine_defenders_mine ON sect_mine_defenders (mine_id);`); err != nil {
		return err
	}

	return nil
}
