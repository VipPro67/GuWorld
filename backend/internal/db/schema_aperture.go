package db

// Immortal aperture (không khiếu) schema and tribulation events.
// aperture_resource_nodes.required_gu_id is a comma-joined list of gu ids, not
// a single FK target.

func schemaAperture() error {
	if err := execSchema("immortal_apertures", `
	CREATE TABLE IF NOT EXISTS immortal_apertures (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL UNIQUE REFERENCES players(id) ON DELETE CASCADE,
		grade VARCHAR(20) NOT NULL,
		time_rate DOUBLE PRECISION NOT NULL DEFAULT 10,
		aperture_hp BIGINT NOT NULL DEFAULT 10000,
		max_aperture_hp BIGINT NOT NULL DEFAULT 10000,
		area_size BIGINT NOT NULL DEFAULT 3000000,
		immortal_essence BIGINT NOT NULL DEFAULT 0,
		essence_production_per_year INTEGER NOT NULL DEFAULT 10,
		internal_years_passed DOUBLE PRECISION NOT NULL DEFAULT 0,
		next_tribulation_year DOUBLE PRECISION NOT NULL DEFAULT 10,
		tribulation_pending_at TIMESTAMP WITH TIME ZONE,
		tribulation_count INTEGER NOT NULL DEFAULT 0,
		rank_start_year DOUBLE PRECISION NOT NULL DEFAULT 0,
		earth_calamity_count INTEGER NOT NULL DEFAULT 0,
		heavenly_tribulation_count INTEGER NOT NULL DEFAULT 0,
		grand_tribulation_count INTEGER NOT NULL DEFAULT 0,
		myriad_tribulation_count INTEGER NOT NULL DEFAULT 0,
		chaos_calamity_count INTEGER NOT NULL DEFAULT 0,
		state VARCHAR(30) NOT NULL DEFAULT 'STABLE',
		dao_marks JSONB NOT NULL DEFAULT '{"strength": 0}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`); err != nil {
		return err
	}

	// Ensure tribulation_pending_at column exists in case the table was created earlier.
	if err := execSchema("immortal_apertures_tribulation_pending_at_upgrade", `
	ALTER TABLE immortal_apertures ADD COLUMN IF NOT EXISTS tribulation_pending_at TIMESTAMP WITH TIME ZONE;
	`); err != nil {
		return err
	}

	if err := execSchema("aperture_resource_nodes", `
	CREATE TABLE IF NOT EXISTS aperture_resource_nodes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		node_type VARCHAR(50) NOT NULL,
		name VARCHAR(100) NOT NULL,
		path VARCHAR(50),
		level INTEGER NOT NULL DEFAULT 1,
		hp INTEGER NOT NULL DEFAULT 1000,
		max_hp INTEGER NOT NULL DEFAULT 1000,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		production_type VARCHAR(50),
		production_amount INTEGER NOT NULL DEFAULT 0,
		production_interval_hours INTEGER NOT NULL DEFAULT 24,
		last_collected_at TIMESTAMP WITH TIME ZONE,
		is_destroyed BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		node_category VARCHAR(20) NOT NULL DEFAULT 'RESOURCE',
		defense_power INTEGER NOT NULL DEFAULT 0,
		attack_range INTEGER NOT NULL DEFAULT 0,
		hp_bonus INTEGER NOT NULL DEFAULT 0,
		damage_reduction INTEGER NOT NULL DEFAULT 0,
		blocks_movement BOOLEAN NOT NULL DEFAULT false,
		required_gu_id VARCHAR(50),
		required_gu_rank INTEGER DEFAULT 0,
		zone_type VARCHAR(50),
		food_path VARCHAR(50),
		item_tier VARCHAR(10)
	);`); err != nil {
		return err
	}
	if err := execSchema("aperture_nodes_category_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_nodes_category ON aperture_resource_nodes (aperture_id, node_category);`); err != nil {
		return err
	}

	if err := execSchema("aperture_variant_humans", `
	CREATE TABLE IF NOT EXISTS aperture_variant_humans (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		race VARCHAR(50) NOT NULL,
		population INTEGER NOT NULL DEFAULT 0,
		happiness DOUBLE PRECISION NOT NULL DEFAULT 70.0,
		oppression DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		rebellion_rate DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		labor_intensity VARCHAR(20) NOT NULL DEFAULT 'NORMAL',
		created_at TIMESTAMP DEFAULT now(),
		updated_at TIMESTAMP DEFAULT now(),
		UNIQUE (aperture_id, race)
	);`); err != nil {
		return err
	}

	if err := execSchema("tribulation_events", `
	CREATE TABLE IF NOT EXISTS tribulation_events (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		tribulation_type VARCHAR(30) NOT NULL,
		path VARCHAR(50) NOT NULL,
		difficulty INTEGER NOT NULL DEFAULT 1,
		status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		aperture_hp_before BIGINT NOT NULL,
		aperture_hp_after BIGINT,
		reward_dao_marks BIGINT NOT NULL DEFAULT 0,
		started_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		ended_at TIMESTAMP WITH TIME ZONE
	);`); err != nil {
		return err
	}
	if err := execSchema("tribulation_events_player_idx", `
	CREATE INDEX IF NOT EXISTS idx_tribulation_events_player ON tribulation_events (player_id);`); err != nil {
		return err
	}

	if err := execSchema("tribulation_enemies", `
	CREATE TABLE IF NOT EXISTS tribulation_enemies (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		event_id UUID NOT NULL REFERENCES tribulation_events(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL,
		enemy_type VARCHAR(50) NOT NULL,
		path VARCHAR(50) NOT NULL,
		hp BIGINT NOT NULL,
		max_hp BIGINT NOT NULL,
		attack BIGINT NOT NULL,
		defense BIGINT NOT NULL,
		target_node_id UUID REFERENCES aperture_resource_nodes(id) ON DELETE SET NULL,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		is_boss BOOLEAN NOT NULL DEFAULT false,
		is_dead BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP DEFAULT now(),
		updated_at TIMESTAMP DEFAULT now()
	);`); err != nil {
		return err
	}
	if err := execSchema("tribulation_enemies_event_idx", `
	CREATE INDEX IF NOT EXISTS idx_tribulation_enemies_event ON tribulation_enemies (event_id);`); err != nil {
		return err
	}

	if err := execSchema("node_outputs", `
	CREATE TABLE IF NOT EXISTS node_outputs (
		item_id VARCHAR(100) PRIMARY KEY,
		item_name VARCHAR(100) NOT NULL,
		item_type VARCHAR(50) NOT NULL,
		item_category VARCHAR(50),
		food_path VARCHAR(50),
		item_tier VARCHAR(10),
		rank INT NOT NULL DEFAULT 1,
		gu_eaten_count INT DEFAULT 0,
		is_recipe_material BOOLEAN DEFAULT false,
		description TEXT,
		created_at TIMESTAMPTZ DEFAULT now()
	);`); err != nil {
		return err
	}
	if err := execSchema("node_outputs_category_idx", `
	CREATE INDEX IF NOT EXISTS idx_node_outputs_category ON node_outputs (item_category);`); err != nil {
		return err
	}
	if err := execSchema("node_outputs_food_path_idx", `
	CREATE INDEX IF NOT EXISTS idx_node_outputs_food_path ON node_outputs (food_path);`); err != nil {
		return err
	}

	return nil
}
