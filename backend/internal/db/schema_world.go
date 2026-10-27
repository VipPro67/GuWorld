package db

// World schema: regions/map, monsters, quests and instance templates. These are
// created before players so players.current_region can reference regions(id).

func schemaWorld() error {
	if err := execSchema("world_map_layout_migrations", `
	CREATE TABLE IF NOT EXISTS world_map_layout_migrations (
		id VARCHAR(100) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}

	// World map catalogue is separate from regions so existing region CSVs and
	// player location rows remain valid while maps are introduced incrementally.
	if err := execSchema("world_maps", `
	CREATE TABLE IF NOT EXISTS world_maps (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		map_type VARCHAR(20) NOT NULL DEFAULT 'WORLD' CHECK (map_type IN ('WORLD','INSTANCE')),
		width INT NOT NULL DEFAULT 30000 CHECK (width > 0),
		height INT NOT NULL DEFAULT 20000 CHECK (height > 0),
		min_rank INT NOT NULL DEFAULT 1,
		max_rank INT NOT NULL DEFAULT 9,
		rules JSONB NOT NULL DEFAULT '{}',
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	INSERT INTO world_maps (id, name, description, map_type, width, height, min_rank, max_rank, is_active)
	VALUES
		('world_map', 'WorldMap', 'Năm đại vực của thế giới hiện tại.', 'WORLD', 30000, 20000, 1, 9, TRUE),
		('bai_thien', 'Bạch Thiên · Tầng 2', 'Đại vực tầng 2; đang chuẩn bị dữ liệu, lối lên tầng chưa mở.', 'WORLD', 30000, 20000, 6, 9, FALSE),
		('hac_thien', 'Hắc Thiên · Tầng 3', 'Đại vực tầng 3; đang chuẩn bị dữ liệu, lối lên tầng chưa mở.', 'WORLD', 30000, 20000, 8, 9, FALSE),
		('realtime_preview', 'MMO Preview', 'Map thử nghiệm Phaser realtime 64×48.', 'WORLD', 64, 48, 1, 9, TRUE),
		('misty_forest', 'Rừng Sương Mù', 'Map MMO Alpha PvE 64×48.', 'WORLD', 64, 48, 1, 9, TRUE),
		('bone_crypt', 'Hầm Xương', 'Map MMO Alpha PvE 64×48.', 'WORLD', 64, 48, 1, 9, TRUE),
		('crimson_ravine', 'Hẻm Núi Đỏ', 'Map MMO Alpha PvP 64×48.', 'WORLD', 64, 48, 1, 9, TRUE),
		('emerald_grotto', 'Hang Ngọc Lục Bảo', 'Map MMO Alpha Safe 64×48.', 'WORLD', 64, 48, 1, 9, TRUE)
	ON CONFLICT (id) DO NOTHING;
	UPDATE world_maps SET name='Bạch Thiên · Tầng 2', description='Đại vực tầng 2; đang chuẩn bị dữ liệu, lối lên tầng chưa mở.' WHERE id='bai_thien';
	UPDATE world_maps SET name='Hắc Thiên · Tầng 3', description='Đại vực tầng 3; đang chuẩn bị dữ liệu, lối lên tầng chưa mở.' WHERE id='hac_thien';

	`); err != nil {
		return err
	}

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
		paths JSONB NOT NULL DEFAULT '[]',
		rules JSONB NOT NULL DEFAULT '{}',
		safe_zone_x INT,
		safe_zone_y INT,
		atlas_x INT NOT NULL DEFAULT 0,
		atlas_y INT NOT NULL DEFAULT 0
	);`); err != nil {
		return err
	}
	if err := execSchema("region_world_maps", `
	CREATE TABLE IF NOT EXISTS region_world_maps (
		region_id VARCHAR(50) PRIMARY KEY REFERENCES regions(id) ON DELETE CASCADE,
		world_map_id VARCHAR(50) NOT NULL REFERENCES world_maps(id) ON DELETE RESTRICT,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_region_world_maps_map ON region_world_maps (world_map_id, region_id);
	INSERT INTO region_world_maps (region_id, world_map_id)
	SELECT id, 'world_map' FROM regions ON CONFLICT (region_id) DO NOTHING;
	`); err != nil {
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
	if err := execSchema("monster_templates_loot_table_idx", `
	CREATE INDEX IF NOT EXISTS idx_monster_templates_loot_table ON monster_templates (loot_table_id);`); err != nil {
		return err
	}

	if err := execSchema("monster_spawns", `
	CREATE TABLE IF NOT EXISTS monster_spawns (
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
		updated_at TIMESTAMP DEFAULT NOW(),
		PRIMARY KEY (region_id, monster_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("monster_spawns_region_monster_unique", `
	DELETE FROM monster_spawns a USING monster_spawns b
	WHERE a.ctid < b.ctid AND a.region_id = b.region_id AND a.monster_id = b.monster_id;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_monster_spawns_region_monster ON monster_spawns (region_id, monster_id);`); err != nil {
		return err
	}
	if err := execSchema("monster_spawns_monster_idx", `
	CREATE INDEX IF NOT EXISTS idx_monster_spawns_monster ON monster_spawns (monster_id);`); err != nil {
		return err
	}
	if err := execSchema("monster_spawns_region_xy_idx", `
	CREATE INDEX IF NOT EXISTS idx_monster_spawns_region_xy ON monster_spawns (region_id, x, y);`); err != nil {
		return err
	}

	// Durable action claims bridge Redis monster CAS with PostgreSQL player
	// updates. Replaying the same monster revision can never damage/drop twice.
	if err := execSchema("world_monster_actions", `
	CREATE TABLE IF NOT EXISTS world_monster_actions (
		action_token TEXT PRIMARY KEY,
		monster_id TEXT NOT NULL,
		player_id UUID NOT NULL,
		applied BOOLEAN NOT NULL DEFAULT FALSE,
		died BOOLEAN NOT NULL DEFAULT FALSE,
		damage DOUBLE PRECISION NOT NULL DEFAULT 0,
		remaining_hp DOUBLE PRECISION NOT NULL DEFAULT 0,
		remaining_shield DOUBLE PRECISION NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_world_monster_actions_created_at ON world_monster_actions(created_at);`); err != nil {
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
		is_active BOOLEAN NOT NULL DEFAULT TRUE
	);`); err != nil {
		return err
	}
	if err := execSchema("map_cells_region_coordinate_unique", `
	ALTER TABLE map_cells DROP CONSTRAINT IF EXISTS map_cells_x_y_key;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_map_cells_region_coordinate_unique ON map_cells (region_id, x, y);
	`); err != nil {
		return err
	}
	if err := execSchema("map_cells_region_idx", `
	CREATE INDEX IF NOT EXISTS idx_map_cells_region ON map_cells (region_id);`); err != nil {
		return err
	}

	// Primeval veins are dynamic world-map resource points. They stay out of
	// map_cells because they spawn/expire on their own schedule and carry a
	// shared accumulation pool instead of a static cell definition.
	if err := execSchema("primeval_veins", `
	CREATE TABLE IF NOT EXISTS primeval_veins (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		x INT NOT NULL,
		y INT NOT NULL,
		vein_type VARCHAR(20) NOT NULL DEFAULT 'mortal' CHECK (vein_type IN ('mortal','immortal')),
		rarity VARCHAR(50) NOT NULL,
		accumulated_since TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		produced_minutes INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMPTZ NOT NULL,
		UNIQUE (region_id, x, y)
	);
	CREATE INDEX IF NOT EXISTS idx_primeval_veins_region ON primeval_veins (region_id, x, y);
	CREATE INDEX IF NOT EXISTS idx_primeval_veins_expires ON primeval_veins (expires_at);`); err != nil {
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
		cooldown_hours INTEGER NOT NULL DEFAULT 24,
		spatial_region_id VARCHAR(50) REFERENCES regions(id) ON DELETE SET NULL,
		duration_minutes INTEGER NOT NULL DEFAULT 120,
		scope VARCHAR(20) NOT NULL DEFAULT 'SHARED' CHECK (scope IN ('SHARED','PARTY','SOLO')),
		rules JSONB NOT NULL DEFAULT '{}'
	);`); err != nil {
		return err
	}
	// Existing databases predate the dungeon scope/rules columns (fresh installs
	// get them from the CREATE above).
	if err := execSchema("instance_templates_dungeon_columns", `
	ALTER TABLE instance_templates ADD COLUMN IF NOT EXISTS scope VARCHAR(20) NOT NULL DEFAULT 'SHARED' CHECK (scope IN ('SHARED','PARTY','SOLO'));
	ALTER TABLE instance_templates ADD COLUMN IF NOT EXISTS rules JSONB NOT NULL DEFAULT '{}';`); err != nil {
		return err
	}
	if err := execSchema("region_loot_tables", `
	CREATE TABLE IF NOT EXISTS region_loot_tables (
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		loot_table_id VARCHAR(50) NOT NULL REFERENCES loot_tables(id) ON DELETE CASCADE,
		monster_type VARCHAR(50) NOT NULL,
		min_rank INT NOT NULL DEFAULT 1,
		max_rank INT NOT NULL DEFAULT 9,
		chance DOUBLE PRECISION NOT NULL DEFAULT 1.0,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		PRIMARY KEY (region_id, loot_table_id, monster_type)
	);`); err != nil {
		return err
	}

	if err := execSchema("region_loot_tables_lookup_idx", `
	CREATE INDEX IF NOT EXISTS idx_region_loot_lookup
		ON region_loot_tables(region_id, monster_type, is_active);`); err != nil {
		return err
	}

	return nil
}

// Shared spatial instances reference players/parties, so create them after
// schemaPlayers and schemaParties. They used to live in schemaWorld, which
// runs before players — that broke fresh database initialization.
func schemaSharedSpatialInstances() error {
	if err := execSchema("shared_spatial_instances", `
	CREATE TABLE IF NOT EXISTS shared_spatial_instances (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		template_id VARCHAR(50) NOT NULL REFERENCES instance_templates(id) ON DELETE CASCADE,
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		scope VARCHAR(20) NOT NULL DEFAULT 'SHARED' CHECK (scope IN ('SHARED','PARTY','SOLO')),
		owner_id UUID REFERENCES players(id) ON DELETE CASCADE,
		party_id UUID REFERENCES parties(id) ON DELETE CASCADE,
		status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','CLOSED')),
		opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMPTZ NOT NULL,
		closed_at TIMESTAMPTZ,
		close_reason VARCHAR(30) NOT NULL DEFAULT ''
	);`); err != nil {
		return err
	}
	// Existing databases predate the scope/owner/party columns; add them before
	// the partial indexes below reference scope (fresh installs get them above).
	if err := execSchema("shared_spatial_instances_dungeon_columns", `
	ALTER TABLE shared_spatial_instances ADD COLUMN IF NOT EXISTS scope VARCHAR(20) NOT NULL DEFAULT 'SHARED' CHECK (scope IN ('SHARED','PARTY','SOLO'));
	ALTER TABLE shared_spatial_instances ADD COLUMN IF NOT EXISTS owner_id UUID REFERENCES players(id) ON DELETE CASCADE;
	ALTER TABLE shared_spatial_instances ADD COLUMN IF NOT EXISTS party_id UUID REFERENCES parties(id) ON DELETE CASCADE;`); err != nil {
		return err
	}
	if err := execSchema("shared_spatial_instances_indexes", `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_shared_spatial_instances_one_active ON shared_spatial_instances(template_id) WHERE scope='SHARED' AND status='ACTIVE';
	CREATE UNIQUE INDEX IF NOT EXISTS idx_spatial_instances_party_active ON shared_spatial_instances(party_id) WHERE scope='PARTY' AND status='ACTIVE';
	CREATE UNIQUE INDEX IF NOT EXISTS idx_spatial_instances_solo_active ON shared_spatial_instances(owner_id) WHERE scope='SOLO' AND status='ACTIVE';
	CREATE INDEX IF NOT EXISTS idx_shared_spatial_instances_expiry ON shared_spatial_instances(expires_at) WHERE status='ACTIVE';
	`); err != nil {
		return err
	}
	return nil
}

// Ground drops reference players, so create them after schemaPlayers.
func schemaMapGroundDrops() error {
	if err := execSchema("map_ground_drops", `
	CREATE TABLE IF NOT EXISTS map_ground_drops (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		map_instance_id UUID,
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
	// Existing databases predate the dungeon instance column; add it before the
	// index below references it (fresh installs get it from the CREATE above).
	if err := execSchema("map_ground_drops_dungeon_columns", `
	ALTER TABLE map_ground_drops ADD COLUMN IF NOT EXISTS map_instance_id UUID;`); err != nil {
		return err
	}
	if err := execSchema("map_ground_drops_location_idx", `
	CREATE INDEX IF NOT EXISTS idx_map_ground_drops_location ON map_ground_drops (region_id, x, y);
	CREATE INDEX IF NOT EXISTS idx_map_ground_drops_instance ON map_ground_drops (region_id, map_instance_id, x, y);`); err != nil {
		return err
	}
	if err := execSchema("map_ground_drops_active_idx", `
	CREATE INDEX IF NOT EXISTS idx_map_ground_drops_active ON map_ground_drops (region_id, expires_at);`); err != nil {
		return err
	}
	if err := execSchema("map_ground_drops_xy_expires_idx", `
	CREATE INDEX IF NOT EXISTS idx_map_ground_drops_xy_expires ON map_ground_drops (x, y, expires_at);`); err != nil {
		return err
	}
	// Superseded by idx_map_ground_drops_xy_expires (same prefix plus the expiry
	// filter the viewport query always applies).
	if err := execSchema("map_ground_drops_xy_idx_drop", `
	DROP INDEX IF EXISTS idx_map_ground_drops_xy;`); err != nil {
		return err
	}

	return nil
}
