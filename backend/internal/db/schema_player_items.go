package db

// Player-owned items, Gu instances, pets, progress tables and killer moves.
// player_inventory_items.item_id and player_recipe_progress.recipe_id are
// polymorphic (item ids + recipe key strings) so they have no FK.

func schemaPlayerItems() error {
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
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
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
	if err := execSchema("player_gu_gu_id_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_gu_id ON player_gu (gu_id);`); err != nil {
		return err
	}

	if err := execSchema("killer_moves", `
	CREATE TABLE IF NOT EXISTS killer_moves (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		name VARCHAR(80) NOT NULL,
		path VARCHAR(50) NOT NULL DEFAULT '',
		move_type VARCHAR(30) NOT NULL DEFAULT 'UTILITY',
		rank_required INTEGER NOT NULL DEFAULT 1,
		core_player_gu_id VARCHAR(80) NOT NULL REFERENCES player_gu(id) ON DELETE CASCADE,
		total_mana_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
		success_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
		damage_value DOUBLE PRECISION NOT NULL DEFAULT 0,
		defense_value DOUBLE PRECISION NOT NULL DEFAULT 0,
		cooldown INTEGER NOT NULL DEFAULT 0,
		channel_time INTEGER NOT NULL DEFAULT 1,
		effects JSONB NOT NULL DEFAULT '{}',
		is_mastered BOOLEAN NOT NULL DEFAULT FALSE,
		is_reliable BOOLEAN NOT NULL DEFAULT FALSE,
		is_instant_cast BOOLEAN NOT NULL DEFAULT FALSE,
		usage_count INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("killer_moves_player_path_idx", `
	CREATE INDEX IF NOT EXISTS idx_killer_moves_player_path ON killer_moves (player_id, path);`); err != nil {
		return err
	}

	if err := execSchema("killer_move_slots", `
	CREATE TABLE IF NOT EXISTS killer_move_slots (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		killer_move_id UUID NOT NULL REFERENCES killer_moves(id) ON DELETE CASCADE,
		player_gu_id VARCHAR(80) NOT NULL REFERENCES player_gu(id) ON DELETE CASCADE,
		slot_type VARCHAR(20) NOT NULL CHECK (slot_type IN ('CORE','SUPPORT')),
		slot_index INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (killer_move_id, slot_index)
	);`); err != nil {
		return err
	}
	if err := execSchema("idx_killer_move_slots_gu", `
	CREATE INDEX IF NOT EXISTS idx_killer_move_slots_gu ON killer_move_slots (player_gu_id);`); err != nil {
		return err
	}

	if err := execSchema("player_inventory_items", `
	CREATE TABLE IF NOT EXISTS player_inventory_items (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		item_id VARCHAR(50) NOT NULL,
		category VARCHAR(30) NOT NULL DEFAULT 'materials',
		quantity INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, item_id, category)
	);`); err != nil {
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

	if err := execSchema("player_dao_path", `
	CREATE TABLE IF NOT EXISTS player_dao_path (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		path VARCHAR(50) NOT NULL,
		mastery VARCHAR(50) NOT NULL DEFAULT 'Ordinary',
		dao_marks INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, path)
	);`); err != nil {
		return err
	}

	if err := execSchema("player_map_progress", `
	CREATE TABLE IF NOT EXISTS player_map_progress (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		region_id VARCHAR(50) NOT NULL DEFAULT '',
		cell_x INT NOT NULL,
		cell_y INT NOT NULL,
		explored BOOLEAN NOT NULL DEFAULT false,
		cleared BOOLEAN NOT NULL DEFAULT false,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, region_id, cell_x, cell_y)
	);`); err != nil {
		return err
	}

	if err := execSchema("player_pets", `
	CREATE TABLE IF NOT EXISTS player_pets (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		monster_template_id VARCHAR(50) NOT NULL REFERENCES monster_templates(id),
		nickname VARCHAR(60) NOT NULL DEFAULT '',
		rank INT NOT NULL DEFAULT 1,
		sub_rank INT NOT NULL DEFAULT 1,
		hp DOUBLE PRECISION NOT NULL,
		max_hp DOUBLE PRECISION NOT NULL,
		base_attack DOUBLE PRECISION NOT NULL,
		base_defense DOUBLE PRECISION NOT NULL,
		speed DOUBLE PRECISION NOT NULL DEFAULT 10,
		count INT NOT NULL DEFAULT 1,
		hunger DOUBLE PRECISION NOT NULL DEFAULT 100,
		max_hunger DOUBLE PRECISION NOT NULL DEFAULT 100,
		pet_hunger_per_hour DOUBLE PRECISION NOT NULL DEFAULT 2.0,
		state VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
		formation_slot INT NOT NULL DEFAULT 0,
		captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("player_pets_player_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_pets_player ON player_pets (player_id);`); err != nil {
		return err
	}
	if err := execSchema("player_pets_formation_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_pets_formation ON player_pets (player_id, formation_slot);`); err != nil {
		return err
	}
	if err := execSchema("player_pets_monster_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_pets_monster ON player_pets (player_id, monster_template_id);`); err != nil {
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

	return nil
}
