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
		aperture_hp DOUBLE PRECISION NOT NULL DEFAULT 10000,
		max_aperture_hp DOUBLE PRECISION NOT NULL DEFAULT 10000,
		area_size BIGINT NOT NULL DEFAULT 3000000,
		immortal_essence BIGINT NOT NULL DEFAULT 0,
		essence_production_per_year INTEGER NOT NULL DEFAULT 10,
		internal_years_passed DOUBLE PRECISION NOT NULL DEFAULT 0,
		next_tribulation_year DOUBLE PRECISION NOT NULL DEFAULT 10,
		tribulation_pending_at TIMESTAMP WITH TIME ZONE,
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

	// Index for fast state-based lookups (used by RegenerateImmortalApertures guard).
	if err := execSchema("immortal_apertures_state_idx", `
	CREATE INDEX IF NOT EXISTS idx_immortal_apertures_state ON immortal_apertures (state);
	`); err != nil {
		return err
	}

	// Ensure tribulation_pending_at column exists in case the table was created earlier.
	if err := execSchema("immortal_apertures_tribulation_pending_at_upgrade", `
	ALTER TABLE immortal_apertures ADD COLUMN IF NOT EXISTS tribulation_pending_at TIMESTAMP WITH TIME ZONE;
	`); err != nil {
		return err
	}

	if err := execSchema("immortal_apertures_remove_total_tribulation_count", `
	ALTER TABLE immortal_apertures DROP COLUMN IF EXISTS tribulation_count;
	`); err != nil {
		return err
	}

	if err := execSchema("immortal_apertures_float_upgrade", `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='immortal_apertures' AND column_name='aperture_hp' AND data_type IN ('bigint', 'integer', 'int')) THEN
			ALTER TABLE immortal_apertures ALTER COLUMN aperture_hp TYPE DOUBLE PRECISION USING aperture_hp::DOUBLE PRECISION;
			ALTER TABLE immortal_apertures ALTER COLUMN max_aperture_hp TYPE DOUBLE PRECISION USING max_aperture_hp::DOUBLE PRECISION;
		END IF;
	END $$;`); err != nil {
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
		production_amount BIGINT NOT NULL DEFAULT 0,
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
		item_tier VARCHAR(10),
		production_key VARCHAR(100)
	);`); err != nil {
		return err
	}
	if err := execSchema("aperture_resource_nodes_prod_amount_upgrade", `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='aperture_resource_nodes' AND column_name='production_amount' AND data_type IN ('integer', 'int')) THEN
			ALTER TABLE aperture_resource_nodes ALTER COLUMN production_amount TYPE BIGINT USING COALESCE(production_amount, 0)::BIGINT;
		END IF;
	END $$;`); err != nil {
		return err
	}

	if err := execSchema("aperture_resource_nodes_production_key_upgrade", `
	ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS production_key VARCHAR(100);
	`); err != nil {
		return err
	}

	if err := execSchema("aperture_resource_nodes_interval_10h_upgrade", `
	UPDATE aperture_resource_nodes SET production_interval_hours = 10 WHERE node_category = 'RESOURCE' OR node_type IN ('FARMING_ZONE', 'BREEDING_ZONE', 'MINING_ZONE');
	`); err != nil {
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

	// --- Phase 1: Variant Human Economy columns ---
	if err := execSchema("variant_humans_prosperity_upgrade", `
	ALTER TABLE aperture_variant_humans ADD COLUMN IF NOT EXISTS prosperity DOUBLE PRECISION NOT NULL DEFAULT 50.0;
	ALTER TABLE aperture_variant_humans ADD COLUMN IF NOT EXISTS gu_master_count INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE aperture_variant_humans ADD COLUMN IF NOT EXISTS gu_immortal_count INTEGER NOT NULL DEFAULT 0;
	`); err != nil {
		return err
	}

	// --- Phase 1: Node Worker Assignment ---
	if err := execSchema("aperture_node_workers", `
	CREATE TABLE IF NOT EXISTS aperture_node_workers (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		node_id UUID NOT NULL REFERENCES aperture_resource_nodes(id) ON DELETE CASCADE,
		variant_human_id UUID NOT NULL REFERENCES aperture_variant_humans(id) ON DELETE CASCADE,
		assigned_population INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		UNIQUE (node_id, variant_human_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("aperture_node_workers_node_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_node_workers_node ON aperture_node_workers (node_id);
	`); err != nil {
		return err
	}
	if err := execSchema("aperture_node_workers_variant_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_node_workers_variant ON aperture_node_workers (variant_human_id);
	`); err != nil {
		return err
	}

	// --- Phase 1: Refinement Queue ---
	if err := execSchema("aperture_refinement_queue", `
	CREATE TABLE IF NOT EXISTS aperture_refinement_queue (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		recipe_id VARCHAR(100) NOT NULL,
		mode VARCHAR(20) NOT NULL DEFAULT 'FIXED_COUNT',
		target_quantity INTEGER NOT NULL DEFAULT 1,
		current_quantity INTEGER NOT NULL DEFAULT 0,
		priority INTEGER NOT NULL DEFAULT 0,
		enabled BOOLEAN NOT NULL DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`); err != nil {
		return err
	}
	if err := execSchema("aperture_refinement_queue_aperture_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_refinement_queue_aperture ON aperture_refinement_queue (aperture_id);
	`); err != nil {
		return err
	}

	// --- Phase 2: Society Simulation columns ---
	if err := execSchema("variant_humans_society_upgrade", `
	ALTER TABLE aperture_variant_humans ADD COLUMN IF NOT EXISTS food_stored DOUBLE PRECISION NOT NULL DEFAULT 0;
	ALTER TABLE aperture_variant_humans ADD COLUMN IF NOT EXISTS food_per_capita DOUBLE PRECISION NOT NULL DEFAULT 1.0;
	ALTER TABLE aperture_variant_humans ADD COLUMN IF NOT EXISTS population_growth_rate DOUBLE PRECISION NOT NULL DEFAULT 0.001;
	ALTER TABLE aperture_variant_humans ADD COLUMN IF NOT EXISTS mortality_rate DOUBLE PRECISION NOT NULL DEFAULT 0.0005;
	`); err != nil {
		return err
	}

	// --- Phase 2: Settlements ---
	if err := execSchema("aperture_variant_settlements", `
	CREATE TABLE IF NOT EXISTS aperture_variant_settlements (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		race VARCHAR(50) NOT NULL,
		name VARCHAR(100),
		settlement_type VARCHAR(50) NOT NULL DEFAULT 'VILLAGE',
		population_capacity INTEGER NOT NULL DEFAULT 10000,
		housing INTEGER NOT NULL DEFAULT 0,
		food_storage INTEGER NOT NULL DEFAULT 5000,
		security DOUBLE PRECISION NOT NULL DEFAULT 50.0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		UNIQUE (aperture_id, race)
	);`); err != nil {
		return err
	}
	if err := execSchema("aperture_variant_settlements_aperture_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_variant_settlements_aperture ON aperture_variant_settlements (aperture_id);
	`); err != nil {
		return err
	}

	// --- Phase 2: Variant Events (rebellion, genius birth, migration, etc.) ---
	if err := execSchema("aperture_variant_events", `
	CREATE TABLE IF NOT EXISTS aperture_variant_events (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		event_type VARCHAR(50) NOT NULL,
		race VARCHAR(50),
		severity VARCHAR(20) NOT NULL DEFAULT 'LOW',
		description TEXT,
		effects JSONB NOT NULL DEFAULT '{}',
		resolved BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		resolved_at TIMESTAMP WITH TIME ZONE
	);`); err != nil {
		return err
	}
	if err := execSchema("aperture_variant_events_aperture_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_variant_events_aperture ON aperture_variant_events (aperture_id, resolved);
	`); err != nil {
		return err
	}

	// --- Phase 3: Variant Human NPCs (Gu Masters, Gu Immortals) ---
	if err := execSchema("aperture_variant_npcs", `
	CREATE TABLE IF NOT EXISTS aperture_variant_npcs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		race VARCHAR(50) NOT NULL,
		name VARCHAR(100) NOT NULL,
		rank INTEGER NOT NULL DEFAULT 1,
		sub_rank INTEGER NOT NULL DEFAULT 0,
		path VARCHAR(50) NOT NULL DEFAULT 'REFINEMENT',
		role VARCHAR(50) NOT NULL DEFAULT 'NONE',
		loyalty DOUBLE PRECISION NOT NULL DEFAULT 100.0,
		talent DOUBLE PRECISION NOT NULL DEFAULT 50.0,
		status VARCHAR(20) NOT NULL DEFAULT 'IDLE',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`); err != nil {
		return err
	}
	if err := execSchema("aperture_variant_npcs_aperture_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_variant_npcs_aperture ON aperture_variant_npcs (aperture_id);
	`); err != nil {
		return err
	}
	if err := execSchema("aperture_variant_npcs_race_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_variant_npcs_race ON aperture_variant_npcs (aperture_id, race);
	`); err != nil {
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
		aperture_hp_before DOUBLE PRECISION NOT NULL,
		aperture_hp_after DOUBLE PRECISION,
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

	if err := execSchema("tribulation_events_float_upgrade", `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tribulation_events' AND column_name='aperture_hp_before' AND data_type IN ('bigint', 'integer', 'int')) THEN
			ALTER TABLE tribulation_events ALTER COLUMN aperture_hp_before TYPE DOUBLE PRECISION USING aperture_hp_before::DOUBLE PRECISION;
			ALTER TABLE tribulation_events ALTER COLUMN aperture_hp_after TYPE DOUBLE PRECISION USING aperture_hp_after::DOUBLE PRECISION;
		END IF;
	END $$;`); err != nil {
		return err
	}

	if err := execSchema("tribulation_enemies", `
	CREATE TABLE IF NOT EXISTS tribulation_enemies (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		event_id UUID NOT NULL REFERENCES tribulation_events(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL,
		enemy_type VARCHAR(50) NOT NULL,
		path VARCHAR(50) NOT NULL,
		hp DOUBLE PRECISION NOT NULL,
		max_hp DOUBLE PRECISION NOT NULL,
		attack DOUBLE PRECISION NOT NULL,
		defense DOUBLE PRECISION NOT NULL,
		target_node_id UUID REFERENCES aperture_resource_nodes(id) ON DELETE SET NULL,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		is_boss BOOLEAN NOT NULL DEFAULT false,
		is_dead BOOLEAN NOT NULL DEFAULT false,
		rank INTEGER NOT NULL DEFAULT 6,
		sub_rank INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP DEFAULT now(),
		updated_at TIMESTAMP DEFAULT now()
	);`); err != nil {
		return err
	}
	if err := execSchema("tribulation_enemies_event_idx", `
	CREATE INDEX IF NOT EXISTS idx_tribulation_enemies_event ON tribulation_enemies (event_id);`); err != nil {
		return err
	}

	if err := execSchema("tribulation_enemies_rank_cols", `
	ALTER TABLE tribulation_enemies ADD COLUMN IF NOT EXISTS rank INTEGER NOT NULL DEFAULT 6;
	ALTER TABLE tribulation_enemies ADD COLUMN IF NOT EXISTS sub_rank INTEGER NOT NULL DEFAULT 1;
	`); err != nil {
		return err
	}

	if err := execSchema("tribulation_enemies_float_upgrade", `
	DO $$
	BEGIN
		ALTER TABLE tribulation_enemies ALTER COLUMN hp TYPE DOUBLE PRECISION USING hp::DOUBLE PRECISION;
		ALTER TABLE tribulation_enemies ALTER COLUMN max_hp TYPE DOUBLE PRECISION USING max_hp::DOUBLE PRECISION;
		ALTER TABLE tribulation_enemies ALTER COLUMN attack TYPE DOUBLE PRECISION USING attack::DOUBLE PRECISION;
		ALTER TABLE tribulation_enemies ALTER COLUMN defense TYPE DOUBLE PRECISION USING defense::DOUBLE PRECISION;
	EXCEPTION WHEN OTHERS THEN
		NULL;
	END $$;`); err != nil {
		return err
	}

	return nil
}

// HasActiveTribulationOrEnemies returns true if the specified aperture has any active tribulation events
// (status = 'ACTIVE') OR any non-dead tribulation enemies (is_dead = false).
func HasActiveTribulationOrEnemies(q Queryer, apertureID string) (bool, error) {
	if q == nil {
		q = DB
	}
	var count int
	query := `
		SELECT 
			(SELECT COUNT(*) FROM tribulation_events WHERE aperture_id = $1 AND status = 'ACTIVE') +
			(SELECT COUNT(*) FROM tribulation_enemies te JOIN tribulation_events ev ON te.event_id = ev.id WHERE ev.aperture_id = $1 AND te.is_dead = false)
	`
	err := q.QueryRow(query, apertureID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
