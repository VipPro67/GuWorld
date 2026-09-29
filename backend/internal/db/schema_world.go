package db

import "github.com/vippro67/GuWorld/backend/pkg/logger"

// World schema: regions/map, monsters, quests and instance templates. These are
// created before players so players.current_region can reference regions(id).

func schemaWorld() error {
	if err := execSchema("regions", `
	CREATE TABLE IF NOT EXISTS regions (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		width INT NOT NULL DEFAULT 100,
		height INT NOT NULL DEFAULT 100,
		min_rank INT NOT NULL DEFAULT 1,
		max_rank INT NOT NULL DEFAULT 5,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		parent_id VARCHAR(50) REFERENCES regions(id) ON DELETE CASCADE,
		zone_type VARCHAR(20) NOT NULL DEFAULT 'KHU_VUC',
		min_x INT NOT NULL DEFAULT 0,
		min_y INT NOT NULL DEFAULT 0,
		max_x INT NOT NULL DEFAULT 0,
		max_y INT NOT NULL DEFAULT 0,
		entry_x INT NOT NULL DEFAULT 0,
		entry_y INT NOT NULL DEFAULT 0,
		pool_weights JSONB NOT NULL DEFAULT '{}',
		paths JSONB NOT NULL DEFAULT '[]'
	);`); err != nil {
		return err
	}

	if err := execSchema("regions_paths_column", `
	ALTER TABLE regions ADD COLUMN IF NOT EXISTS paths JSONB NOT NULL DEFAULT '[]';`); err != nil {
		return err
	}

	if err := execSchema("regions_safe_zone", `
	ALTER TABLE regions ADD COLUMN IF NOT EXISTS safe_zone_x INT;
	ALTER TABLE regions ADD COLUMN IF NOT EXISTS safe_zone_y INT;`); err != nil {
		return err
	}

	if err := execSchema("monster_templates", `
	CREATE TABLE IF NOT EXISTS monster_templates (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		type VARCHAR(50) NOT NULL,
		max_hp DOUBLE PRECISION NOT NULL,
		base_attack DOUBLE PRECISION NOT NULL,
		base_defense DOUBLE PRECISION NOT NULL,
		gu_inventory JSONB NOT NULL DEFAULT '[]',
		gu_pool JSONB NOT NULL DEFAULT '[]',
		item_inventory JSONB NOT NULL DEFAULT '{}',
		dao_marks JSONB NOT NULL DEFAULT '{}',
		path_masteries JSONB NOT NULL DEFAULT '{}',
		ai_profile VARCHAR(50) NOT NULL DEFAULT 'beast_basic',
		loot_table_id VARCHAR(50) REFERENCES loot_tables(id) ON DELETE SET NULL,
		description TEXT,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		is_capturable BOOLEAN NOT NULL DEFAULT false,
		capture_chance DOUBLE PRECISION NOT NULL DEFAULT 0.20,
		pet_hunger_per_hour DOUBLE PRECISION NOT NULL DEFAULT 2.0,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`); err != nil {
		return err
	}
	// Add missing columns for existing databases without requiring a full wipe
	if err := execSchema("monster_templates_alter_cols", `
	ALTER TABLE monster_templates ADD COLUMN IF NOT EXISTS gu_pool JSONB NOT NULL DEFAULT '[]';
	ALTER TABLE monster_templates ADD COLUMN IF NOT EXISTS is_capturable BOOLEAN NOT NULL DEFAULT false;
	ALTER TABLE monster_templates ADD COLUMN IF NOT EXISTS capture_chance DOUBLE PRECISION NOT NULL DEFAULT 0.20;
	ALTER TABLE monster_templates ADD COLUMN IF NOT EXISTS pet_hunger_per_hour DOUBLE PRECISION NOT NULL DEFAULT 2.0;
	`); err != nil {
		logger.S.Warnf("Failed to alter monster_templates columns: %v", err)
	}
	if err := execSchema("monster_templates_loot_table_idx", `
	CREATE INDEX IF NOT EXISTS idx_monster_templates_loot_table ON monster_templates (loot_table_id);`); err != nil {
		return err
	}

	if err := execSchema("monster_spawns", `
	CREATE TABLE IF NOT EXISTS monster_spawns (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		monster_id VARCHAR(50) NOT NULL REFERENCES monster_templates(id) ON DELETE CASCADE,
		x INT,
		y INT,
		spawn_mode VARCHAR(50) NOT NULL DEFAULT 'Random',
		chance DOUBLE PRECISION NOT NULL DEFAULT 1.0,
		respawn_seconds INT NOT NULL DEFAULT 300,
		min_rank INT NOT NULL DEFAULT 1,
		max_rank INT NOT NULL DEFAULT 9,
		is_boss_spawn BOOLEAN NOT NULL DEFAULT FALSE,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("monster_spawns_region_idx", `
	CREATE INDEX IF NOT EXISTS idx_monster_spawns_region ON monster_spawns (region_id);`); err != nil {
		return err
	}
	if err := execSchema("monster_spawns_monster_idx", `
	CREATE INDEX IF NOT EXISTS idx_monster_spawns_monster ON monster_spawns (monster_id);`); err != nil {
		return err
	}

	if err := execSchema("map_cells", `
	CREATE TABLE IF NOT EXISTS map_cells (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		x INT NOT NULL,
		y INT NOT NULL,
		cell_type VARCHAR(50) NOT NULL,
		content_id VARCHAR(50),
		metadata JSONB NOT NULL DEFAULT '{}',
		respawn_seconds INT NOT NULL DEFAULT 300,
		last_cleared_at TIMESTAMPTZ,
		cleared_by UUID,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		UNIQUE (x, y)
	);`); err != nil {
		return err
	}
	if err := execSchema("map_cells_region_idx", `
	CREATE INDEX IF NOT EXISTS idx_map_cells_region ON map_cells (region_id);`); err != nil {
		return err
	}

	if err := execSchema("quests", `
	CREATE TABLE IF NOT EXISTS quests (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		req_desc TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL DEFAULT '',
		rewards JSONB NOT NULL DEFAULT '[]'
	);`); err != nil {
		return err
	}

	if err := execSchema("instance_templates", `
	CREATE TABLE IF NOT EXISTS instance_templates (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		type VARCHAR(50) NOT NULL,
		required_rank INTEGER NOT NULL,
		entry_item VARCHAR(50) NOT NULL,
		description TEXT NOT NULL,
		rewards JSONB NOT NULL DEFAULT '{}',
		map_data JSONB NOT NULL DEFAULT '[]',
		cooldown_hours INTEGER NOT NULL DEFAULT 24
	);`); err != nil {
		return err
	}

	if err := execSchema("map_ground_drops", `
	CREATE TABLE IF NOT EXISTS map_ground_drops (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		x INT NOT NULL,
		y INT NOT NULL,
		loser_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		loser_name VARCHAR(100) NOT NULL DEFAULT '',
		killer_id UUID REFERENCES players(id) ON DELETE SET NULL,
		killer_name VARCHAR(100) DEFAULT '',
		loot_data JSONB NOT NULL DEFAULT '{}',
		protected_until TIMESTAMP WITH TIME ZONE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("map_ground_drops_location_idx", `
	CREATE INDEX IF NOT EXISTS idx_map_ground_drops_location ON map_ground_drops (region_id, x, y);`); err != nil {
		return err
	}
	if err := execSchema("map_ground_drops_active_idx", `
	CREATE INDEX IF NOT EXISTS idx_map_ground_drops_active ON map_ground_drops (region_id, expires_at);`); err != nil {
		return err
	}

	return nil
}
