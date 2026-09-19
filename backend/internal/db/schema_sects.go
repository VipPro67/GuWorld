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
		map_x INT NOT NULL DEFAULT 50,
		map_y INT NOT NULL DEFAULT 50,
		shield_until TIMESTAMP WITH TIME ZONE,
		buildings JSONB NOT NULL DEFAULT '{"main_hall": 1, "defensive_array": 1, "spirit_vein": 1, "treasury": 1, "training_ground": 1}',
		vault_gu JSONB NOT NULL DEFAULT '[]',
		vault_materials JSONB NOT NULL DEFAULT '{}',
		vault_recipes JSONB NOT NULL DEFAULT '[]',
		funds BIGINT NOT NULL DEFAULT 0,
		level INTEGER NOT NULL DEFAULT 1,
		require_approval BOOLEAN NOT NULL DEFAULT FALSE,
		require_promotion_approval BOOLEAN NOT NULL DEFAULT FALSE,
		last_maintenance_at TIMESTAMP WITH TIME ZONE,
		last_salary_paid_at TIMESTAMP WITH TIME ZONE,
		last_vault_decay_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_territory_tick_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}

	if err := execSchema("sects_territory_columns", `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sects' AND column_name='map_x') THEN
			ALTER TABLE sects ADD COLUMN map_x INT NOT NULL DEFAULT 50;
			ALTER TABLE sects ADD COLUMN map_y INT NOT NULL DEFAULT 50;
			ALTER TABLE sects ADD COLUMN shield_until TIMESTAMP WITH TIME ZONE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sects' AND column_name='buildings') THEN
			ALTER TABLE sects ADD COLUMN buildings JSONB NOT NULL DEFAULT '{"main_hall": 1, "defensive_array": 1, "spirit_vein": 1, "treasury": 1, "training_ground": 1}';
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sects' AND column_name='last_territory_tick_at') THEN
			ALTER TABLE sects ADD COLUMN last_territory_tick_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
		END IF;
	END $$;`); err != nil {
		return err
	}

	if err := execSchema("sects_last_vault_decay_column", `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sects' AND column_name='last_vault_decay_at') THEN
			ALTER TABLE sects ADD COLUMN last_vault_decay_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
		END IF;
	END $$;`); err != nil {
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
		target_amount BIGINT NOT NULL DEFAULT 1,
		reward_contribution BIGINT NOT NULL DEFAULT 0,
		required_member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
		expires_at TIMESTAMP WITHOUT TIME ZONE,
		UNIQUE (sect_id, title)
	);`); err != nil {
		return err
	}
	if err := execSchema("sect_missions_bigint_upgrade", `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sect_missions' AND column_name='target_amount' AND data_type IN ('integer', 'int')) THEN
			ALTER TABLE sect_missions ALTER COLUMN target_amount TYPE BIGINT USING COALESCE(target_amount, 0)::BIGINT;
		END IF;
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sect_missions' AND column_name='reward_contribution' AND data_type IN ('integer', 'int')) THEN
			ALTER TABLE sect_missions ALTER COLUMN reward_contribution TYPE BIGINT USING COALESCE(reward_contribution, 0)::BIGINT;
		END IF;
	END $$;`); err != nil {
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
		territory_credited_today INTEGER NOT NULL DEFAULT 0,
		is_defending BOOLEAN NOT NULL DEFAULT FALSE
	);`); err != nil {
		return err
	}
	if err := execSchema("sect_members_defending_column", `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sect_members' AND column_name='is_defending') THEN
			ALTER TABLE sect_members ADD COLUMN is_defending BOOLEAN NOT NULL DEFAULT FALSE;
		END IF;
	END $$;`); err != nil {
		return err
	}
	if err := execSchema("sect_members_sect_idx", `
	CREATE INDEX IF NOT EXISTS idx_sect_members_sect ON sect_members (sect_id);`); err != nil {
		return err
	}
	if err := execSchema("sect_members_sect_rank_idx", `
	CREATE INDEX IF NOT EXISTS idx_sect_members_sect_rank ON sect_members (sect_id, member_rank);`); err != nil {
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
	if err := execSchema("sect_promotion_requests_idx", `
	CREATE INDEX IF NOT EXISTS idx_sect_promotion_requests ON sect_promotion_requests (sect_id, status, created_at);`); err != nil {
		return err
	}

	if err := execSchema("player_sect_missions", `
	CREATE TABLE IF NOT EXISTS player_sect_missions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		mission_id UUID NOT NULL REFERENCES sect_missions(id) ON DELETE CASCADE,
		current_progress BIGINT NOT NULL DEFAULT 0,
		status VARCHAR(30) NOT NULL DEFAULT 'IN_PROGRESS',
		assigned_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		expires_at TIMESTAMP WITHOUT TIME ZONE,
		UNIQUE (player_id, mission_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("player_sect_missions_bigint_upgrade", `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='player_sect_missions' AND column_name='current_progress' AND data_type IN ('integer', 'int')) THEN
			ALTER TABLE player_sect_missions ALTER COLUMN current_progress TYPE BIGINT USING COALESCE(current_progress, 0)::BIGINT;
		END IF;
	END $$;`); err != nil {
		return err
	}
	if err := execSchema("player_sect_missions_player_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_sect_missions_player ON player_sect_missions (player_id);`); err != nil {
		return err
	}

	// Drop old mine tables if they exist
	if err := execSchema("drop_sect_mines", `
	DROP TABLE IF EXISTS sect_mine_defenders;
	DROP TABLE IF EXISTS sect_mines;
	`); err != nil {
		return err
	}

	if err := execSchema("sect_defenders", `
	CREATE TABLE IF NOT EXISTS sect_defenders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
	if err := execSchema("uq_sect_defender_player", `
	CREATE UNIQUE INDEX IF NOT EXISTS uq_sect_defender_player ON sect_defenders (player_id) WHERE defender_type = 'member' AND player_id IS NOT NULL;`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_defenders_sect", `
	CREATE INDEX IF NOT EXISTS idx_sect_defenders_sect ON sect_defenders (sect_id);`); err != nil {
		return err
	}

	if err := execSchema("sect_vault_materials", `
	CREATE TABLE IF NOT EXISTS sect_vault_materials (
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		item_id VARCHAR(100) NOT NULL,
		quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (sect_id, item_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_vault_materials_item", `
	CREATE INDEX IF NOT EXISTS idx_sect_vault_materials_item ON sect_vault_materials (item_id) WHERE quantity > 0;`); err != nil {
		return err
	}

	if err := execSchema("sect_vault_gu", `
	CREATE TABLE IF NOT EXISTS sect_vault_gu (
		id VARCHAR(80) PRIMARY KEY,
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		gu_id VARCHAR(50) NOT NULL,
		rank INTEGER NOT NULL DEFAULT 1 CHECK (rank >= 1 AND rank <= 9),
		quantity BIGINT NOT NULL DEFAULT 1 CHECK (quantity >= 0),
		cost BIGINT NOT NULL DEFAULT 0,
		hunger DOUBLE PRECISION NOT NULL DEFAULT 100.0 CHECK (hunger >= 0),
		instance_id VARCHAR(80) DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_vault_gu_sect_gu", `
	CREATE INDEX IF NOT EXISTS idx_sect_vault_gu_sect_gu ON sect_vault_gu (sect_id, gu_id, rank);`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_vault_gu_gu_id", `
	CREATE INDEX IF NOT EXISTS idx_sect_vault_gu_gu_id ON sect_vault_gu (gu_id, rank);`); err != nil {
		return err
	}

	if err := execSchema("sect_vault_recipes", `
	CREATE TABLE IF NOT EXISTS sect_vault_recipes (
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		recipe_id VARCHAR(100) NOT NULL,
		completion_rate FLOAT NOT NULL DEFAULT 100.0 CHECK (completion_rate >= 0 AND completion_rate <= 100.0),
		unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (sect_id, recipe_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_vault_recipes_sect", `
	CREATE INDEX IF NOT EXISTS idx_sect_vault_recipes_sect ON sect_vault_recipes (sect_id);`); err != nil {
		return err
	}

	return nil
}

