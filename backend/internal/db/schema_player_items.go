package db

// Player-owned items, Gu instances, pets, progress tables and killer moves.
// player_inventory_items.item_id and player_recipe_progress.recipe_id are
// polymorphic (item ids + recipe key strings) so they have no FK.

func schemaPlayerItems() error {
	if err := execSchema("player_killer_moves", `
	CREATE TABLE IF NOT EXISTS player_killer_moves (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		name VARCHAR(80) NOT NULL,
		main_gu_instance_id VARCHAR(80) NOT NULL,
		main_gu_id VARCHAR(50) NOT NULL,
		main_gu_rank INTEGER NOT NULL DEFAULT 1,
		components JSONB NOT NULL DEFAULT '[]'::jsonb,
		selected_effects JSONB NOT NULL DEFAULT '[]'::jsonb,
		compiled_effects JSONB NOT NULL DEFAULT '[]'::jsonb,
		path VARCHAR(80) NOT NULL DEFAULT '',
		creation_path_mastery_level INTEGER NOT NULL DEFAULT 0,
		power_budget DOUBLE PRECISION NOT NULL DEFAULT 0,
		power_load DOUBLE PRECISION NOT NULL DEFAULT 0,
		harmony DOUBLE PRECISION NOT NULL DEFAULT 0,
		complexity DOUBLE PRECISION NOT NULL DEFAULT 0,
		success_chance DOUBLE PRECISION NOT NULL DEFAULT 0,
		mastery_xp INTEGER NOT NULL DEFAULT 0,
		is_equipped BOOLEAN NOT NULL DEFAULT false,
		balance_version INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("player_killer_moves_owner_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_killer_moves_owner ON player_killer_moves (player_id, created_at DESC);`); err != nil {
		return err
	}
	if err := execSchema("player_killer_moves_equipped_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_killer_moves_equipped ON player_killer_moves (player_id, is_equipped) WHERE is_equipped = true;`); err != nil {
		return err
	}

	if err := execSchema("player_gu", `
	CREATE TABLE IF NOT EXISTS player_gu (
		id VARCHAR(80) PRIMARY KEY,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		gu_id VARCHAR(50) NOT NULL,
		gu_rank INTEGER NOT NULL DEFAULT 1,
		rank INTEGER NOT NULL DEFAULT 1,
		FOREIGN KEY (gu_id, gu_rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE RESTRICT,
		level INTEGER NOT NULL DEFAULT 1,
		hunger DOUBLE PRECISION NOT NULL DEFAULT 100,
		state VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		location VARCHAR(30) NOT NULL DEFAULT 'INVENTORY',
		is_locked BOOLEAN NOT NULL DEFAULT false,
		equipped BOOLEAN NOT NULL DEFAULT false,
		is_vital BOOLEAN NOT NULL DEFAULT false,
		integration_time DOUBLE PRECISION NOT NULL DEFAULT 0,
		equipped_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
		growth_hours DOUBLE PRECISION NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_fed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_player_gu_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_player_gu ON player_gu (player_id, gu_id);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_player_equipped_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_player_equipped ON player_gu (player_id, equipped);`); err != nil {
		return err
	}
	// Hunger decay scans one player's live Gus by state on every login
	// (EvaluatePlayerHungerTx). This partial index keeps that scan off the
	// wider (player_id, gu_id) / (player_id, equipped) indexes.
	if err := execSchema("player_gu_player_state_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_player_state ON player_gu (player_id, state) WHERE state IN ('ACTIVE', 'SLEEPING', 'SEALED');`); err != nil {
		return err
	}
	if err := execSchema("player_gu_gu_id_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_gu_id ON player_gu (gu_id);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_gu_id_lower_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_gu_id_lower ON player_gu (LOWER(gu_id));`); err != nil {
		return err
	}
	// Tiên Cổ (rank>=6) duy nhất cross-rank toàn server: 1 gu_id chỉ 1 instance
	// đang sống.
	if err := execSchema("uq_player_gu_immortal", `
	CREATE UNIQUE INDEX IF NOT EXISTS uq_player_gu_immortal
	ON player_gu (gu_id) WHERE gu_rank >= 6 OR rank >= 6;`); err != nil {
		return err
	}
	if err := execSchema("player_gu_vital_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_vital ON player_gu (player_id) WHERE is_vital = true;`); err != nil {
		return err
	}

	if err := execSchema("player_gu_state_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_state ON player_gu (state);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_equipped_active_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_equipped_active ON player_gu (player_id, gu_id, gu_rank) WHERE equipped = true AND state = 'ACTIVE';`); err != nil {
		return err
	}
	if err := execSchema("player_gu_location_state_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_location_state ON player_gu (state, location) WHERE state IN ('ACTIVE', 'SLEEPING', 'EQUIPPING');`); err != nil {
		return err
	}
	if err := execSchema("player_gu_active_equipped_growth_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_active_equipped_growth ON player_gu (gu_id, gu_rank) WHERE equipped = true AND state = 'ACTIVE';`); err != nil {
		return err
	}

	if err := execSchema("player_inventory_items", `
	CREATE TABLE IF NOT EXISTS player_inventory_items (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		item_id VARCHAR(50) NOT NULL,
		category VARCHAR(30) NOT NULL DEFAULT 'materials',
		quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, item_id, category)
	);`); err != nil {
		return err
	}
	if err := execSchema("player_inventory_items_player_cat_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_inventory_items_player_cat ON player_inventory_items (player_id, category);`); err != nil {
		return err
	}

	if err := execSchema("player_recipe_progress", `
	CREATE TABLE IF NOT EXISTS player_recipe_progress (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		recipe_id VARCHAR(50) NOT NULL,
		completion_rate FLOAT NOT NULL DEFAULT 0,
		is_unlocked BOOLEAN NOT NULL DEFAULT false,
		deduce_count INT NOT NULL DEFAULT 0,
		last_deduce_at TIMESTAMPTZ,
		favorite BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, recipe_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("player_quest_progress", `
	CREATE TABLE IF NOT EXISTS player_quest_progress (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		quest_id VARCHAR(50) NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
		progress_data JSONB NOT NULL DEFAULT '{}',
		completed_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, quest_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("player_instance_sessions", `
	CREATE TABLE IF NOT EXISTS player_instance_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		instance_id VARCHAR(50) NOT NULL REFERENCES instance_templates(id) ON DELETE CASCADE,
		current_stage INTEGER NOT NULL DEFAULT 0,
		status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		temp_inventory JSONB NOT NULL DEFAULT '[]',
		created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
	);`); err != nil {
		return err
	}
	if err := execSchema("player_active_session_idx", `
	CREATE UNIQUE INDEX IF NOT EXISTS player_active_session_idx ON player_instance_sessions (player_id) WHERE status = 'ACTIVE';`); err != nil {
		return err
	}
	if err := execSchema("player_instance_sessions_player_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_instance_sessions_player ON player_instance_sessions (player_id);`); err != nil {
		return err
	}
	if err := execSchema("shared_spatial_instance_members", `
	CREATE TABLE IF NOT EXISTS shared_spatial_instance_members (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		instance_id UUID NOT NULL REFERENCES shared_spatial_instances(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		return_region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE RESTRICT,
		return_x INTEGER NOT NULL,
		return_y INTEGER NOT NULL,
		joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		exited_at TIMESTAMPTZ,
		exit_reason VARCHAR(30) NOT NULL DEFAULT ''
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_shared_spatial_members_one_active_player ON shared_spatial_instance_members(player_id) WHERE exited_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_shared_spatial_members_active_instance ON shared_spatial_instance_members(instance_id, player_id) WHERE exited_at IS NULL;
	`); err != nil {
		return err
	}

	if err := execSchema("player_inheritance_claims", `
	CREATE TABLE IF NOT EXISTS player_inheritance_claims (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		altar_id VARCHAR(50) NOT NULL,
		template_id VARCHAR(50) NOT NULL,
		claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		tier VARCHAR(20) NOT NULL DEFAULT 'MORTAL' CHECK (tier IN ('MORTAL', 'IMMORTAL')),
		rewards_snapshot JSONB NOT NULL DEFAULT '{}'
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_player_inheritance_unique_claim 
		ON player_inheritance_claims (player_id, altar_id) 
		WHERE tier = 'IMMORTAL';
	CREATE INDEX IF NOT EXISTS idx_player_inheritance_history 
		ON player_inheritance_claims (player_id, claimed_at);
	`); err != nil {
		return err
	}

	return nil
}
