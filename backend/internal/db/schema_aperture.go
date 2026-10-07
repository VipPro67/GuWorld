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

	// player_dao_path was an obsolete duplicate store. Preserve any legacy-only
	// path values as a fallback, while canonical immortal_apertures values win,
	// then remove the duplicate table so new writes have one source of truth.
	if err := execSchema("drop_legacy_player_dao_path", `
	DO $$
	BEGIN
		IF to_regclass('public.player_dao_path') IS NOT NULL THEN
			UPDATE players p
			SET path_masteries = COALESCE((
				SELECT jsonb_object_agg(
					regexp_replace(lower(btrim(legacy.path)), '\s+path$', ''),
					CASE lower(btrim(legacy.mastery))
						WHEN 'master' THEN '10000'::jsonb
						WHEN 'grandmaster' THEN '100000'::jsonb
						WHEN 'great grandmaster' THEN '1000000'::jsonb
						WHEN 'supreme grandmaster' THEN '10000000'::jsonb
						ELSE '0'::jsonb
					END
				)
				FROM player_dao_path legacy
				WHERE legacy.player_id = p.id
				  AND lower(btrim(legacy.mastery)) IN ('ordinary', 'master', 'grandmaster', 'great grandmaster', 'supreme grandmaster')
			), '{}'::jsonb) || COALESCE(p.path_masteries, '{}'::jsonb)
			WHERE EXISTS (
				SELECT 1 FROM player_dao_path legacy
				WHERE legacy.player_id = p.id
				  AND lower(btrim(legacy.mastery)) IN ('ordinary', 'master', 'grandmaster', 'great grandmaster', 'supreme grandmaster')
			);

			UPDATE immortal_apertures ia
			SET dao_marks = COALESCE((
				SELECT jsonb_object_agg(legacy.path, legacy.dao_marks)
				FROM player_dao_path legacy
				WHERE legacy.player_id = ia.player_id
			), '{}'::jsonb) || COALESCE(ia.dao_marks, '{}'::jsonb)
			WHERE EXISTS (SELECT 1 FROM player_dao_path legacy WHERE legacy.player_id = ia.player_id);
			DROP TABLE player_dao_path;
		END IF;
	END $$;`); err != nil {
		return err
	}

	// Index for fast state-based lookups (used by RegenerateImmortalApertures guard).
	if err := execSchema("immortal_apertures_state_idx", `
	CREATE INDEX IF NOT EXISTS idx_immortal_apertures_state ON immortal_apertures (state);
	`); err != nil {
		return err
	}
	if err := execSchema("immortal_apertures_healing_idx", `
	CREATE INDEX IF NOT EXISTS idx_immortal_apertures_healing ON immortal_apertures (aperture_hp) WHERE state != 'DESTROYED' AND aperture_hp < max_aperture_hp;
	`); err != nil {
		return err
	}

	// Ensure tribulation_pending_at column exists in case the table was created earlier.
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
	if err := execSchema("aperture_nodes_interval_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_nodes_interval ON aperture_resource_nodes (node_category, node_type, production_interval_hours);`); err != nil {
		return err
	}

	// Idempotent guard: without IS DISTINCT FROM the UPDATE rewrites every
	// matching row on each boot (row locks + WAL), tripping the 15s
	// statement_timeout once the table grows.
	if err := execSchema("aperture_resource_nodes_interval_10h_upgrade", `
	UPDATE aperture_resource_nodes SET production_interval_hours = 10 WHERE (node_category = 'RESOURCE' OR node_type IN ('FARMING_ZONE', 'BREEDING_ZONE', 'MINING_ZONE')) AND production_interval_hours IS DISTINCT FROM 10;
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
		prosperity DOUBLE PRECISION NOT NULL DEFAULT 50.0,
		gu_master_count INTEGER NOT NULL DEFAULT 0,
		gu_immortal_count INTEGER NOT NULL DEFAULT 0,
		food_stored DOUBLE PRECISION NOT NULL DEFAULT 0,
		food_per_capita DOUBLE PRECISION NOT NULL DEFAULT 1.0,
		population_growth_rate DOUBLE PRECISION NOT NULL DEFAULT 0.001,
		mortality_rate DOUBLE PRECISION NOT NULL DEFAULT 0.0005,
		created_at TIMESTAMP DEFAULT now(),
		updated_at TIMESTAMP DEFAULT now(),
		UNIQUE (aperture_id, race)
	);`); err != nil {
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
		last_error TEXT NOT NULL DEFAULT '',
		last_processed_at TIMESTAMP WITH TIME ZONE,
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
		effects JSONB NOT NULL DEFAULT '{"schema_version":2,"blocks":[]}',
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
	if err := execSchema("tribulation_events_player_status_idx", `
	CREATE INDEX IF NOT EXISTS idx_tribulation_events_player_status ON tribulation_events (player_id, status);`); err != nil {
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

	if err := execSchema("tribulation_events_aperture_idx", `
	CREATE INDEX IF NOT EXISTS idx_tribulation_events_aperture ON tribulation_events (aperture_id);`); err != nil {
		return err
	}
	if err := execSchema("tribulation_events_aperture_status_idx", `
	CREATE INDEX IF NOT EXISTS idx_tribulation_events_aperture_status ON tribulation_events (aperture_id, status);`); err != nil {
		return err
	}
	// The 15s/60s workers scan `WHERE status = 'ACTIVE'` directly, which the
	// (player_id, status) and (aperture_id, status) indexes cannot serve. Without
	// this, both seq-scan the ever-growing (never pruned) event log every tick.
	if err := execSchema("tribulation_events_active_idx", `
	CREATE INDEX IF NOT EXISTS idx_tribulation_events_active ON tribulation_events (status) WHERE status = 'ACTIVE';`); err != nil {
		return err
	}
	if err := execSchema("tribulation_enemies_event_dead_idx", `
	CREATE INDEX IF NOT EXISTS idx_tribulation_enemies_event_dead ON tribulation_enemies (event_id, is_dead);`); err != nil {
		return err
	}

	// Partial index so the boot cleanup below seeks ruins instead of
	// seq-scanning the whole node table.
	if err := execSchema("aperture_nodes_destroyed_idx", `
	CREATE INDEX IF NOT EXISTS idx_aperture_nodes_destroyed ON aperture_resource_nodes (id) WHERE is_destroyed = true;`); err != nil {
		return err
	}

	// Destroyed nodes are terminal losses. Remove legacy ruins now that all
	// dependent foreign keys have their ON DELETE behavior installed.
	if err := execSchema("aperture_destroyed_nodes_cleanup", `
	DELETE FROM aperture_resource_nodes WHERE is_destroyed = true;`); err != nil {
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
