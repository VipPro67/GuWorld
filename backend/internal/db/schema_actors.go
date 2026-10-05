package db

// Actor schema: persistent non-player individuals and the sole instance store
// for companion beasts and named variant NPCs. Static monster data remains in
// monster_templates; world presence and assignments are introduced separately.
func schemaActors() error {
	if err := execSchema("non_player_actors", `
	CREATE TABLE IF NOT EXISTS non_player_actors (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		actor_kind VARCHAR(32) NOT NULL,
		owner_player_id UUID REFERENCES players(id) ON DELETE CASCADE,
		faction_id UUID,
		aperture_id UUID REFERENCES immortal_apertures(id) ON DELETE SET NULL,

		template_source VARCHAR(32) NOT NULL,
		template_id VARCHAR(100) NOT NULL,
		species VARCHAR(100) NOT NULL DEFAULT '',
		race VARCHAR(50) NOT NULL DEFAULT '',
		name VARCHAR(100) NOT NULL DEFAULT '',

		rank INT NOT NULL DEFAULT 1,
		sub_rank INT NOT NULL DEFAULT 0,
		experience BIGINT NOT NULL DEFAULT 0,
		talent DOUBLE PRECISION NOT NULL DEFAULT 0,
		loyalty DOUBLE PRECISION NOT NULL DEFAULT 0,

		hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		max_hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		base_attack DOUBLE PRECISION NOT NULL DEFAULT 0,
		base_defense DOUBLE PRECISION NOT NULL DEFAULT 0,
		speed DOUBLE PRECISION NOT NULL DEFAULT 0,

		count INT NOT NULL DEFAULT 1,
		hunger DOUBLE PRECISION NOT NULL DEFAULT 0,
		max_hunger DOUBLE PRECISION NOT NULL DEFAULT 100,
		upkeep_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
		hunger_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

		lifecycle_state VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
		execution_state VARCHAR(20) NOT NULL DEFAULT 'STORED',
		behavior_profile VARCHAR(50) NOT NULL DEFAULT 'PASSIVE',
		ai_profile VARCHAR(50) NOT NULL DEFAULT '',

		capabilities JSONB NOT NULL DEFAULT '{}',
		traits JSONB NOT NULL DEFAULT '{}',
		progression JSONB NOT NULL DEFAULT '{}',
		formation_slot INT NOT NULL DEFAULT 0,
		path VARCHAR(50) NOT NULL DEFAULT '',
		role VARCHAR(50) NOT NULL DEFAULT 'NONE',

		legacy_source VARCHAR(32),
		legacy_id UUID,
		version BIGINT NOT NULL DEFAULT 1,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

		CONSTRAINT non_player_actors_kind_check CHECK (actor_kind IN
			('MONSTER','COMPANION_BEAST','VARIANT_NPC','WORLD_NPC','SECT_NPC','CONSTRUCT','BOSS')),
		CONSTRAINT non_player_actors_template_source_check CHECK (template_source IN
			('MONSTER_TEMPLATE','VARIANT_RACE','NPC_TEMPLATE','CUSTOM')),
		CONSTRAINT non_player_actors_lifecycle_check CHECK (lifecycle_state IN ('ACTIVE','DEAD','DISMISSED')),
		CONSTRAINT non_player_actors_execution_check CHECK (execution_state IN
			('STORED','DEPLOYING','TRAINING','WORKING','RECOVERING','DEAD','DISMISSED',
			 'WORLD_IDLE','MOVING','FOLLOWING','GUARDING','PATROLLING','GATHERING','COMBAT','RETURNING','INJURED')),
		CONSTRAINT non_player_actors_count_check CHECK (count >= 1),
		CONSTRAINT non_player_actors_hp_check CHECK (hp >= 0),
		CONSTRAINT non_player_actors_max_hp_check CHECK (max_hp > 0),
		CONSTRAINT non_player_actors_max_hunger_check CHECK (max_hunger > 0),
		CONSTRAINT non_player_actors_feedable_hunger_check CHECK (
			capabilities -> 'FEEDABLE' IS NULL OR capabilities -> 'FEEDABLE' = 'false'::jsonb
			OR (hunger >= 0 AND hunger <= max_hunger)),
		CONSTRAINT non_player_actors_json_objects_check CHECK (
			jsonb_typeof(capabilities) = 'object' AND jsonb_typeof(traits) = 'object' AND
			jsonb_typeof(progression) = 'object'),
		CONSTRAINT non_player_actors_owned_kind_check CHECK (
			actor_kind NOT IN ('COMPANION_BEAST','VARIANT_NPC','CONSTRUCT') OR owner_player_id IS NOT NULL),
		CONSTRAINT non_player_actors_version_check CHECK (version >= 1),
		CONSTRAINT non_player_actors_legacy_pair_check CHECK (
			(legacy_source IS NULL) = (legacy_id IS NULL)),
		CONSTRAINT non_player_actors_terminal_state_check CHECK (
			(lifecycle_state = 'ACTIVE' AND execution_state NOT IN ('DEAD','DISMISSED')) OR
			(lifecycle_state = 'DEAD' AND execution_state = 'DEAD') OR
			(lifecycle_state = 'DISMISSED' AND execution_state = 'DISMISSED'))
	);`); err != nil {
		return err
	}
	indexes := []struct{ name, ddl string }{
		{"non_player_actors_owner_kind_idx", `CREATE INDEX IF NOT EXISTS idx_non_player_actors_owner_kind ON non_player_actors (owner_player_id, actor_kind);`},
		{"non_player_actors_aperture_kind_idx", `CREATE INDEX IF NOT EXISTS idx_non_player_actors_aperture_kind ON non_player_actors (aperture_id, actor_kind);`},
		{"non_player_actors_execution_idx", `CREATE INDEX IF NOT EXISTS idx_non_player_actors_execution_state ON non_player_actors (execution_state);`},
		{"non_player_actors_template_idx", `CREATE INDEX IF NOT EXISTS idx_non_player_actors_template ON non_player_actors (template_source, template_id);`},
		{"non_player_actors_formation_idx", `CREATE INDEX IF NOT EXISTS idx_non_player_actors_formation ON non_player_actors (owner_player_id, formation_slot) WHERE actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE';`},
		{"non_player_actors_variant_formation_idx", `CREATE UNIQUE INDEX IF NOT EXISTS idx_non_player_actors_variant_formation ON non_player_actors (owner_player_id, formation_slot) WHERE actor_kind = 'VARIANT_NPC' AND lifecycle_state = 'ACTIVE' AND formation_slot > 0;`},
		{"non_player_actors_owned_formation_uidx", `CREATE UNIQUE INDEX IF NOT EXISTS idx_non_player_actors_owned_formation ON non_player_actors (owner_player_id, formation_slot) WHERE actor_kind IN ('COMPANION_BEAST','VARIANT_NPC') AND lifecycle_state = 'ACTIVE' AND formation_slot > 0;`},
		{"non_player_actors_active_idx", `CREATE INDEX IF NOT EXISTS idx_non_player_actors_active ON non_player_actors (actor_kind, execution_state) WHERE lifecycle_state = 'ACTIVE';`},
		{"non_player_actors_legacy_uidx", `CREATE UNIQUE INDEX IF NOT EXISTS idx_non_player_actors_legacy ON non_player_actors (legacy_source, legacy_id) WHERE legacy_source IS NOT NULL AND legacy_id IS NOT NULL;`},
	}
	for _, index := range indexes {
		if err := execSchema(index.name, index.ddl); err != nil {
			return err
		}
	}

	if err := execSchema("actor_gu_equipment", `
	CREATE TABLE IF NOT EXISTS actor_gu_equipment (
		actor_id UUID NOT NULL REFERENCES non_player_actors(id) ON DELETE CASCADE,
		player_gu_id VARCHAR(80) NOT NULL REFERENCES player_gu(id) ON DELETE CASCADE,
		slot INTEGER NOT NULL,
		equipped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (actor_id, player_gu_id),
		UNIQUE (player_gu_id),
		UNIQUE (actor_id, slot),
		CONSTRAINT actor_gu_equipment_slot_check CHECK (slot BETWEEN 1 AND 9)
	);`); err != nil {
		return err
	}
	if err := execSchema("actor_gu_equipment_guard", `
	CREATE OR REPLACE FUNCTION guard_actor_equipped_gu() RETURNS trigger AS $$
	BEGIN
		IF EXISTS (SELECT 1 FROM actor_gu_equipment WHERE player_gu_id=OLD.id) THEN
			RAISE EXCEPTION 'Gu % is equipped by a non-player actor', OLD.id USING ERRCODE='check_violation';
		END IF;
		RETURN NEW;
	END; $$ LANGUAGE plpgsql;
	DROP TRIGGER IF EXISTS trg_guard_actor_equipped_gu ON player_gu;
	CREATE TRIGGER trg_guard_actor_equipped_gu BEFORE UPDATE OF player_id,location,equipped,is_vital ON player_gu
	FOR EACH ROW EXECUTE FUNCTION guard_actor_equipped_gu();`); err != nil {
		return err
	}
	if err := execSchema("variant_npc_progression_normalization", `
	UPDATE non_player_actors
	SET sub_rank = GREATEST(1, LEAST(sub_rank, 4)),
	    max_hp = GREATEST(max_hp, 80 + rank * 40 + talent),
	    hp = CASE WHEN hp >= max_hp THEN GREATEST(max_hp, 80 + rank * 40 + talent)
	              ELSE LEAST(hp, GREATEST(max_hp, 80 + rank * 40 + talent)) END,
	    base_attack = GREATEST(base_attack, 8 + rank * 6 + talent * 0.12),
	    base_defense = GREATEST(base_defense, 5 + rank * 4 + talent * 0.08),
	    speed = GREATEST(speed, 8 + rank + talent * 0.02)
	WHERE actor_kind = 'VARIANT_NPC'
	  AND (sub_rank NOT BETWEEN 1 AND 4 OR base_attack <= 0 OR base_defense <= 0 OR speed <= 0);`); err != nil {
		return err
	}

	// Canonical store is solely non_player_actors. Legacy tables
	// player_pets / aperture_variant_npcs were dropped by the one-way
	// migration and are no longer referenced anywhere.
	return nil
}
