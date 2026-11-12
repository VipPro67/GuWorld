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
		composition_graph JSONB NOT NULL DEFAULT '{}'::jsonb,
		selected_effects JSONB NOT NULL DEFAULT '[]'::jsonb,
		compiled_effects JSONB NOT NULL DEFAULT '{"schema_version":2,"blocks":[]}'::jsonb,
		path VARCHAR(80) NOT NULL DEFAULT '',
		creation_path_mastery_level INTEGER NOT NULL DEFAULT 0,
		power_budget DOUBLE PRECISION NOT NULL DEFAULT 0,
		power_load DOUBLE PRECISION NOT NULL DEFAULT 0,
		harmony DOUBLE PRECISION NOT NULL DEFAULT 0,
		complexity DOUBLE PRECISION NOT NULL DEFAULT 0,
		success_chance DOUBLE PRECISION NOT NULL DEFAULT 0,
		combat_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
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
	if err := execSchema("player_killer_moves_combat_profile_col", `
	ALTER TABLE player_killer_moves ADD COLUMN IF NOT EXISTS combat_profile JSONB NOT NULL DEFAULT '{}'::jsonb;`); err != nil {
		return err
	}
	if err := execSchema("player_killer_moves_compiled_effects_envelope_default", `
	ALTER TABLE player_killer_moves ALTER COLUMN compiled_effects SET DEFAULT '{"schema_version":2,"blocks":[]}'::jsonb;`); err != nil {
		return err
	}
	if err := execSchema("player_killer_moves_composition_graph_col", `
	ALTER TABLE player_killer_moves ADD COLUMN IF NOT EXISTS composition_graph JSONB NOT NULL DEFAULT '{}'::jsonb;`); err != nil {
		return err
	}

	if err := execSchema("player_gu_houses", `
	CREATE TABLE IF NOT EXISTS player_gu_houses (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		name VARCHAR(80) NOT NULL,
		rank INTEGER NOT NULL DEFAULT 6,
		core_gu_instances JSONB NOT NULL DEFAULT '[]'::jsonb,
		aux_gu_count INTEGER NOT NULL DEFAULT 0,
		house_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
		current_durability DOUBLE PRECISION NOT NULL DEFAULT 100,
		is_equipped BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_houses_owner_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_houses_owner ON player_gu_houses (player_id, created_at DESC);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_houses_equipped_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_gu_houses_equipped ON player_gu_houses (player_id, is_equipped) WHERE is_equipped = true;`); err != nil {
		return err
	}
	if err := execSchema("player_combat_profiles_remove_realtime_fields", `
	UPDATE player_killer_moves
	SET combat_profile = combat_profile - ARRAY['realtime_windup_ms', 'realtime_cooldown_ms', 'realtime_effects']::text[]
	WHERE combat_profile ?| ARRAY['realtime_windup_ms', 'realtime_cooldown_ms', 'realtime_effects'];
	UPDATE player_gu_houses
	SET house_profile = jsonb_set(
		house_profile, '{actions}',
		(SELECT COALESCE(jsonb_agg(action - ARRAY['realtime_windup_ms', 'realtime_cooldown_ms', 'realtime_effects']::text[]), '[]'::jsonb)
		 FROM jsonb_array_elements(house_profile->'actions') AS actions(action)),
		true
	)
	WHERE jsonb_typeof(house_profile->'actions') = 'array'
	AND EXISTS (SELECT 1 FROM jsonb_array_elements(house_profile->'actions') AS actions(action)
		WHERE action ?| ARRAY['realtime_windup_ms', 'realtime_cooldown_ms', 'realtime_effects']);`); err != nil {
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
		refinement_progress DOUBLE PRECISION NOT NULL DEFAULT 100,
		durability DOUBLE PRECISION NOT NULL DEFAULT 100,
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
	if err := execSchema("player_killer_move_gu_reservation_owner_keys", `
	CREATE UNIQUE INDEX IF NOT EXISTS uq_player_gu_reservation_identity ON player_gu (id, player_id, gu_id, gu_rank, location, is_locked);
	CREATE UNIQUE INDEX IF NOT EXISTS uq_player_killer_moves_id_owner ON player_killer_moves (id, player_id);`); err != nil {
		return err
	}
	if err := execSchema("player_killer_move_gu_reservations", `
	CREATE TABLE IF NOT EXISTS player_killer_move_gu_reservations (
		gu_instance_id VARCHAR(80) PRIMARY KEY,
		killer_move_id UUID NOT NULL,
		player_id UUID NOT NULL,
		gu_id VARCHAR(50) NOT NULL,
		gu_rank INTEGER NOT NULL,
		gu_location VARCHAR(30) NOT NULL,
		gu_is_locked BOOLEAN NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT fk_km_gu_reservation_gu_state FOREIGN KEY (gu_instance_id, player_id, gu_id, gu_rank, gu_location, gu_is_locked)
			REFERENCES player_gu(id, player_id, gu_id, gu_rank, location, is_locked) ON DELETE NO ACTION ON UPDATE RESTRICT DEFERRABLE INITIALLY DEFERRED,
		CONSTRAINT fk_km_gu_reservation_move_owner FOREIGN KEY (killer_move_id, player_id)
			REFERENCES player_killer_moves(id, player_id) ON DELETE CASCADE ON UPDATE RESTRICT
	);
	CREATE INDEX IF NOT EXISTS idx_killer_move_gu_reservations_move ON player_killer_move_gu_reservations (killer_move_id);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_reservation_owner_keys", `
	CREATE UNIQUE INDEX IF NOT EXISTS uq_player_gu_houses_id_owner ON player_gu_houses (id, player_id);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_reservation_gu_key", `
	CREATE UNIQUE INDEX IF NOT EXISTS uq_player_gu_house_reservation_identity ON player_gu (id, player_id, gu_id, gu_rank, location, is_locked, state);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_core_reservations", `
	CREATE TABLE IF NOT EXISTS player_gu_house_core_reservations (
		gu_instance_id VARCHAR(80) PRIMARY KEY,
		gu_house_id UUID NOT NULL,
		player_id UUID NOT NULL,
		gu_id VARCHAR(50) NOT NULL,
		gu_rank INTEGER NOT NULL,
		gu_location VARCHAR(30) NOT NULL,
		gu_is_locked BOOLEAN NOT NULL,
		gu_state VARCHAR(30) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT fk_gu_house_reservation_gu_state FOREIGN KEY (gu_instance_id, player_id, gu_id, gu_rank, gu_location, gu_is_locked, gu_state)
			REFERENCES player_gu(id, player_id, gu_id, gu_rank, location, is_locked, state) ON DELETE NO ACTION ON UPDATE NO ACTION DEFERRABLE INITIALLY DEFERRED,
		CONSTRAINT fk_gu_house_reservation_house_owner FOREIGN KEY (gu_house_id, player_id)
			REFERENCES player_gu_houses(id, player_id) ON DELETE CASCADE ON UPDATE RESTRICT
	);
	CREATE INDEX IF NOT EXISTS idx_gu_house_core_reservations_house ON player_gu_house_core_reservations (gu_house_id);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_core_reservations_state_compat", `
	ALTER TABLE player_gu_house_core_reservations ADD COLUMN IF NOT EXISTS gu_state VARCHAR(30);
	UPDATE player_gu_house_core_reservations AS reservation SET gu_state=gu.state
	FROM player_gu AS gu WHERE gu.id=reservation.gu_instance_id AND reservation.gu_state IS NULL;
	DELETE FROM player_gu_house_core_reservations AS reservation WHERE NOT EXISTS (
		SELECT 1 FROM player_gu AS gu WHERE gu.id=reservation.gu_instance_id AND gu.player_id=reservation.player_id
	);
	ALTER TABLE player_gu_house_core_reservations ALTER COLUMN gu_state SET NOT NULL;
	ALTER TABLE player_gu_house_core_reservations DROP CONSTRAINT IF EXISTS fk_gu_house_reservation_gu_state;
	DO $$ BEGIN
		ALTER TABLE player_gu_house_core_reservations ADD CONSTRAINT fk_gu_house_reservation_gu_state
			FOREIGN KEY (gu_instance_id, player_id, gu_id, gu_rank, gu_location, gu_is_locked, gu_state)
			REFERENCES player_gu(id, player_id, gu_id, gu_rank, location, is_locked, state)
			ON DELETE NO ACTION ON UPDATE NO ACTION DEFERRABLE INITIALLY DEFERRED;
	END $$;`); err != nil {
		return err
	}
	if err := execSchema("player_killer_move_gu_reservations_state_compat", `
	ALTER TABLE player_killer_move_gu_reservations ADD COLUMN IF NOT EXISTS gu_id VARCHAR(50);
	ALTER TABLE player_killer_move_gu_reservations ADD COLUMN IF NOT EXISTS gu_rank INTEGER;
	ALTER TABLE player_killer_move_gu_reservations ADD COLUMN IF NOT EXISTS gu_location VARCHAR(30);
	ALTER TABLE player_killer_move_gu_reservations ADD COLUMN IF NOT EXISTS gu_is_locked BOOLEAN;
	UPDATE player_killer_move_gu_reservations AS reservation
	SET gu_id=gu.gu_id, gu_rank=gu.gu_rank, gu_location=gu.location, gu_is_locked=gu.is_locked
	FROM player_gu AS gu WHERE gu.id=reservation.gu_instance_id
	AND (reservation.gu_id IS NULL OR reservation.gu_rank IS NULL OR reservation.gu_location IS NULL OR reservation.gu_is_locked IS NULL);
	DELETE FROM player_killer_move_gu_reservations AS reservation
	WHERE NOT EXISTS (SELECT 1 FROM player_gu AS gu WHERE gu.id=reservation.gu_instance_id AND gu.player_id=reservation.player_id)
	OR NOT EXISTS (SELECT 1 FROM player_killer_moves AS move WHERE move.id=reservation.killer_move_id AND move.player_id=reservation.player_id);
	ALTER TABLE player_killer_move_gu_reservations ALTER COLUMN gu_id SET NOT NULL;
	ALTER TABLE player_killer_move_gu_reservations ALTER COLUMN gu_rank SET NOT NULL;
	ALTER TABLE player_killer_move_gu_reservations ALTER COLUMN gu_location SET NOT NULL;
	ALTER TABLE player_killer_move_gu_reservations ALTER COLUMN gu_is_locked SET NOT NULL;
	ALTER TABLE player_killer_move_gu_reservations DROP CONSTRAINT IF EXISTS player_killer_move_gu_reservations_gu_instance_id_fkey;
	ALTER TABLE player_killer_move_gu_reservations DROP CONSTRAINT IF EXISTS player_killer_move_gu_reservations_killer_move_id_fkey;
	ALTER TABLE player_killer_move_gu_reservations DROP CONSTRAINT IF EXISTS fk_km_gu_reservation_gu_owner;
	DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='player_killer_move_gu_reservations'::regclass AND conname='fk_km_gu_reservation_gu_state') THEN
			ALTER TABLE player_killer_move_gu_reservations ADD CONSTRAINT fk_km_gu_reservation_gu_state
				FOREIGN KEY (gu_instance_id, player_id, gu_id, gu_rank, gu_location, gu_is_locked)
				REFERENCES player_gu(id, player_id, gu_id, gu_rank, location, is_locked)
				ON DELETE NO ACTION ON UPDATE RESTRICT DEFERRABLE INITIALLY DEFERRED;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='player_killer_move_gu_reservations'::regclass AND conname='fk_km_gu_reservation_move_owner') THEN
			ALTER TABLE player_killer_move_gu_reservations ADD CONSTRAINT fk_km_gu_reservation_move_owner
				FOREIGN KEY (killer_move_id, player_id) REFERENCES player_killer_moves(id, player_id)
				ON DELETE CASCADE ON UPDATE RESTRICT;
		END IF;
	END $$;`); err != nil {
		return err
	}
	if err := execSchema("player_killer_move_gu_reservations_backfill", `
	WITH parsed AS (
		SELECT move.id AS move_id, move.player_id, component.value->>'instance_id' AS gu_instance_id,
			component.value->>'gu_id' AS gu_id,
			CASE WHEN component.value->>'rank' ~ '^[0-9]+$' THEN (component.value->>'rank')::INTEGER ELSE 0 END AS gu_rank
		FROM player_killer_moves AS move
		CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(move.components)='array' THEN move.components ELSE '[]'::jsonb END) AS component(value)
		WHERE move.is_equipped=true
	), invalid_moves AS (
		SELECT DISTINCT parsed.move_id FROM parsed
		LEFT JOIN player_gu AS gu ON gu.id=parsed.gu_instance_id
		WHERE parsed.gu_instance_id IS NULL OR parsed.gu_id IS NULL OR parsed.gu_rank < 1 OR gu.id IS NULL
			OR gu.player_id<>parsed.player_id OR gu.gu_id<>parsed.gu_id OR gu.gu_rank<>parsed.gu_rank
			OR gu.state='DEAD' OR gu.location<>'INVENTORY' OR gu.is_locked=true
		UNION
		SELECT id FROM player_killer_moves WHERE is_equipped=true
			AND (jsonb_typeof(components)<>'array'
				OR jsonb_array_length(CASE WHEN jsonb_typeof(components)='array' THEN components ELSE '[]'::jsonb END)=0
				OR jsonb_array_length(CASE WHEN jsonb_typeof(components)='array' THEN components ELSE '[]'::jsonb END)>6)
	), conflicting_gu AS (
		SELECT parsed.gu_instance_id FROM parsed
		WHERE NOT EXISTS (SELECT 1 FROM invalid_moves WHERE invalid_moves.move_id=parsed.move_id)
		GROUP BY parsed.gu_instance_id HAVING COUNT(DISTINCT parsed.move_id)>1
	), unsafe_moves AS (
		SELECT move_id FROM invalid_moves
		UNION
		SELECT parsed.move_id FROM parsed JOIN conflicting_gu USING (gu_instance_id)
	)
	UPDATE player_killer_moves AS move SET is_equipped=false,updated_at=NOW()
	WHERE move.is_equipped=true AND move.id IN (SELECT move_id FROM unsafe_moves);`); err != nil {
		return err
	}
	if err := execSchema("player_killer_move_gu_reservations_release_inactive", `
	DELETE FROM player_killer_move_gu_reservations AS reservation
	WHERE NOT EXISTS (
		SELECT 1 FROM player_killer_moves AS move
		CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(move.components)='array' THEN move.components ELSE '[]'::jsonb END) AS component(value)
		WHERE move.id=reservation.killer_move_id AND move.player_id=reservation.player_id AND move.is_equipped=true
			AND component.value->>'instance_id'=reservation.gu_instance_id
	);`); err != nil {
		return err
	}
	if err := execSchema("player_killer_move_gu_reservations_backfill_active", `
	WITH parsed AS (
		SELECT move.id AS move_id, move.player_id, component.value->>'instance_id' AS gu_instance_id,
			component.value->>'gu_id' AS gu_id,
			CASE WHEN component.value->>'rank' ~ '^[0-9]+$' THEN (component.value->>'rank')::INTEGER ELSE 0 END AS gu_rank
		FROM player_killer_moves AS move
		CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(move.components)='array' THEN move.components ELSE '[]'::jsonb END) AS component(value)
		WHERE move.is_equipped=true
	)
	INSERT INTO player_killer_move_gu_reservations(gu_instance_id,killer_move_id,player_id,gu_id,gu_rank,gu_location,gu_is_locked)
	SELECT DISTINCT gu.id,move.id,move.player_id,gu.gu_id,gu.gu_rank,gu.location,gu.is_locked
	FROM parsed
	JOIN player_killer_moves AS move ON move.id=parsed.move_id AND move.is_equipped=true
	JOIN player_gu AS gu ON gu.id=parsed.gu_instance_id AND gu.player_id=move.player_id
		AND gu.gu_id=parsed.gu_id AND gu.gu_rank=parsed.gu_rank
	WHERE gu.state<>'DEAD' AND gu.location='INVENTORY' AND gu.is_locked=false
	ON CONFLICT (gu_instance_id) DO NOTHING;`); err != nil {
		return err
	}
	if err := execSchema("player_killer_move_gu_reservations_fail_closed", `
	UPDATE player_killer_moves AS move SET is_equipped=false,updated_at=NOW()
	WHERE move.is_equipped=true AND EXISTS (
		SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(move.components)='array' THEN move.components ELSE '[]'::jsonb END) AS component(value)
		WHERE NOT EXISTS (
			SELECT 1 FROM player_killer_move_gu_reservations AS reservation
			WHERE reservation.gu_instance_id=component.value->>'instance_id' AND reservation.killer_move_id=move.id AND reservation.player_id=move.player_id
		)
	);`); err != nil {
		return err
	}
	if err := execSchema("player_killer_move_gu_reservations_cleanup_partial_backfill", `
	DELETE FROM player_killer_move_gu_reservations AS reservation
	WHERE NOT EXISTS (
		SELECT 1 FROM player_killer_moves AS move
		CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(move.components)='array' THEN move.components ELSE '[]'::jsonb END) AS component(value)
		WHERE move.id=reservation.killer_move_id AND move.player_id=reservation.player_id AND move.is_equipped=true
			AND component.value->>'instance_id'=reservation.gu_instance_id
	);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_core_reservations_backfill", `
	WITH parsed AS (
		SELECT house.id AS house_id, house.player_id, component.value #>> '{}' AS gu_instance_id
		FROM player_gu_houses AS house
		CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(house.core_gu_instances)='array' THEN house.core_gu_instances ELSE '[]'::jsonb END) AS component(value)
		WHERE house.is_equipped=true
	), invalid_houses AS (
		SELECT DISTINCT parsed.house_id FROM parsed
		LEFT JOIN player_gu AS gu ON gu.id=parsed.gu_instance_id
		WHERE parsed.gu_instance_id IS NULL OR gu.id IS NULL OR gu.player_id<>parsed.player_id
			OR gu.gu_rank<6 OR gu.state<>'ACTIVE' OR gu.location<>'INVENTORY' OR gu.is_locked=true
			OR EXISTS (SELECT 1 FROM player_killer_move_gu_reservations AS kmr WHERE kmr.gu_instance_id=parsed.gu_instance_id)
		UNION
		SELECT id FROM player_gu_houses WHERE is_equipped=true
			AND (jsonb_typeof(core_gu_instances)<>'array'
				OR jsonb_array_length(CASE WHEN jsonb_typeof(core_gu_instances)='array' THEN core_gu_instances ELSE '[]'::jsonb END)<2
				OR jsonb_array_length(CASE WHEN jsonb_typeof(core_gu_instances)='array' THEN core_gu_instances ELSE '[]'::jsonb END)>5)
		UNION
		SELECT parsed.house_id FROM parsed GROUP BY parsed.house_id
			HAVING COUNT(*)<>COUNT(DISTINCT parsed.gu_instance_id)
	), conflicting_gu AS (
		SELECT parsed.gu_instance_id FROM parsed
		WHERE NOT EXISTS (SELECT 1 FROM invalid_houses WHERE invalid_houses.house_id=parsed.house_id)
		GROUP BY parsed.gu_instance_id HAVING COUNT(DISTINCT parsed.house_id)>1
	), unsafe_houses AS (
		SELECT house_id FROM invalid_houses
		UNION
		SELECT parsed.house_id FROM parsed JOIN conflicting_gu USING (gu_instance_id)
	)
	UPDATE player_gu_houses AS house SET is_equipped=false,updated_at=NOW()
	WHERE house.is_equipped=true AND house.id IN (SELECT house_id FROM unsafe_houses);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_core_reservations_release_inactive", `
	DELETE FROM player_gu_house_core_reservations AS reservation
	WHERE NOT EXISTS (
		SELECT 1 FROM player_gu_houses AS house
		CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(house.core_gu_instances)='array' THEN house.core_gu_instances ELSE '[]'::jsonb END) AS component(value)
		WHERE house.id=reservation.gu_house_id AND house.player_id=reservation.player_id AND house.is_equipped=true
			AND component.value #>> '{}'=reservation.gu_instance_id
	);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_core_reservations_backfill_active", `
	WITH parsed AS (
		SELECT house.id AS house_id, house.player_id, component.value #>> '{}' AS gu_instance_id
		FROM player_gu_houses AS house
		CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(house.core_gu_instances)='array' THEN house.core_gu_instances ELSE '[]'::jsonb END) AS component(value)
		WHERE house.is_equipped=true
	)
	INSERT INTO player_gu_house_core_reservations(gu_instance_id,gu_house_id,player_id,gu_id,gu_rank,gu_location,gu_is_locked,gu_state)
	SELECT DISTINCT gu.id,house.id,house.player_id,gu.gu_id,gu.gu_rank,gu.location,gu.is_locked,gu.state
	FROM parsed
	JOIN player_gu_houses AS house ON house.id=parsed.house_id AND house.is_equipped=true
	JOIN player_gu AS gu ON gu.id=parsed.gu_instance_id AND gu.player_id=house.player_id
	WHERE gu.gu_rank>=6 AND gu.state='ACTIVE' AND gu.location='INVENTORY' AND gu.is_locked=false
	ON CONFLICT (gu_instance_id) DO NOTHING;`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_core_reservations_fail_closed", `
	UPDATE player_gu_houses AS house SET is_equipped=false,updated_at=NOW()
	WHERE house.is_equipped=true AND EXISTS (
		SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(house.core_gu_instances)='array' THEN house.core_gu_instances ELSE '[]'::jsonb END) AS component(value)
		WHERE NOT EXISTS (
			SELECT 1 FROM player_gu_house_core_reservations AS reservation
			WHERE reservation.gu_instance_id=component.value #>> '{}' AND reservation.gu_house_id=house.id AND reservation.player_id=house.player_id
		)
	);`); err != nil {
		return err
	}
	if err := execSchema("player_gu_house_core_reservations_cleanup_partial_backfill", `
	DELETE FROM player_gu_house_core_reservations AS reservation
	WHERE NOT EXISTS (
		SELECT 1 FROM player_gu_houses AS house
		CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(house.core_gu_instances)='array' THEN house.core_gu_instances ELSE '[]'::jsonb END) AS component(value)
		WHERE house.id=reservation.gu_house_id AND house.player_id=reservation.player_id AND house.is_equipped=true
			AND component.value #>> '{}'=reservation.gu_instance_id
	);`); err != nil {
		return err
	}

	if err := execSchema("player_gu_refinement_and_durability_cols", `
	ALTER TABLE player_gu ADD COLUMN IF NOT EXISTS refinement_progress DOUBLE PRECISION NOT NULL DEFAULT 100;
	ALTER TABLE player_gu ADD COLUMN IF NOT EXISTS durability DOUBLE PRECISION NOT NULL DEFAULT 100;
	UPDATE player_gu SET refinement_progress = 100 WHERE refinement_progress IS NULL OR refinement_progress <= 0;
	UPDATE player_gu SET durability = 100 WHERE durability IS NULL OR durability <= 0;`); err != nil {
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
		recipe_id VARCHAR(100) NOT NULL,
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
	// recipe_id is a derived key (recipe_<target_gu_id>_<rank>_<variant>) and the
	// longest seeded id is already 50 chars. Widen existing databases so a new
	// longer recipe name cannot fail player_recipe_progress inserts.
	if err := execSchema("player_recipe_progress.recipe_id width", `
	ALTER TABLE player_recipe_progress ALTER COLUMN recipe_id TYPE VARCHAR(100);`); err != nil {
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
