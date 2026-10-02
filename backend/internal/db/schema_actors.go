package db

import "context"

// Actor schema: persistent non-player individuals. Static monster data remains
// in monster_templates; world presence and assignments are introduced by later
// releases so the foundation does not change live gameplay ownership yet.
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
		{"non_player_actors_active_idx", `CREATE INDEX IF NOT EXISTS idx_non_player_actors_active ON non_player_actors (actor_kind, execution_state) WHERE lifecycle_state = 'ACTIVE';`},
		{"non_player_actors_legacy_uidx", `CREATE UNIQUE INDEX IF NOT EXISTS idx_non_player_actors_legacy ON non_player_actors (legacy_source, legacy_id) WHERE legacy_source IS NOT NULL AND legacy_id IS NOT NULL;`},
	}
	for _, index := range indexes {
		if err := execSchema(index.name, index.ddl); err != nil {
			return err
		}
	}

	return backfillLegacyActors()
}

// backfillLegacyActors is deliberately idempotent. Release 1 leaves both
// legacy tables authoritative and only creates stable canonical projections.
func backfillLegacyActors() error {
	if _, err := DB.Exec(nonPlayerActorsPetProjectionSQL, nil); err != nil {
		return err
	}
	if _, err := DB.Exec(nonPlayerActorsVariantNPCProjectionSQL, nil); err != nil {
		return err
	}
	if _, err := DB.Exec(nonPlayerActorsLegacyCleanupSQL, nil); err != nil {
		return err
	}
	return nil
}

// SyncLegacyActorProjections refreshes the compatibility projection before a
// Release 1 roster read. Legacy tables remain authoritative until later
// cutovers, so the refresh also removes projections for released/dismissed rows.
func SyncLegacyActorProjections(ctx context.Context, playerID string) error {
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range []string{
		nonPlayerActorsPetProjectionSQL,
		nonPlayerActorsVariantNPCProjectionSQL,
		nonPlayerActorsLegacyCleanupSQL,
	} {
		if _, err := tx.ExecContext(ctx, statement, playerID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

const nonPlayerActorsPetProjectionSQL = `
	INSERT INTO non_player_actors (
		actor_kind, owner_player_id, template_source, template_id, species, name,
		rank, sub_rank, hp, max_hp, base_attack, base_defense, speed, count,
		hunger, max_hunger, upkeep_rate, hunger_updated_at, lifecycle_state,
		execution_state, behavior_profile, ai_profile, capabilities, traits,
		progression, legacy_source, legacy_id, created_at, updated_at)
	SELECT 'COMPANION_BEAST', p.player_id, 'MONSTER_TEMPLATE', p.monster_template_id,
	       p.monster_template_id, p.nickname, p.rank, p.sub_rank, p.hp, p.max_hp,
	       p.base_attack, p.base_defense, p.speed, p.count, p.hunger, p.max_hunger,
	       p.pet_hunger_per_hour, NOW(), 'ACTIVE',
	       CASE WHEN p.state = 'SLEEPING' THEN 'RECOVERING' ELSE 'STORED' END,
	       'DEFENSIVE_FOLLOWER', COALESCE(m.ai_profile, 'beast_basic'),
	       '{"MOVE":true,"FOLLOW":true,"GUARD":true,"COMBAT":true,"FEEDABLE":true,"CAPTURABLE":true,"STACKABLE":true,"MORTAL":true}'::jsonb,
	       jsonb_build_object('legacy_state', p.state),
	       jsonb_build_object('formation_slot', p.formation_slot),
	       'PLAYER_PET', p.id, p.captured_at, p.updated_at
	FROM player_pets p
	LEFT JOIN monster_templates m ON m.id = p.monster_template_id
	WHERE ($1::uuid IS NULL OR p.player_id = $1)
	ON CONFLICT (legacy_source, legacy_id)
		WHERE legacy_source IS NOT NULL AND legacy_id IS NOT NULL
	DO UPDATE SET
		owner_player_id = EXCLUDED.owner_player_id, template_id = EXCLUDED.template_id,
		species = EXCLUDED.species, name = EXCLUDED.name, rank = EXCLUDED.rank,
		sub_rank = EXCLUDED.sub_rank, hp = EXCLUDED.hp, max_hp = EXCLUDED.max_hp,
		base_attack = EXCLUDED.base_attack, base_defense = EXCLUDED.base_defense,
		speed = EXCLUDED.speed, count = EXCLUDED.count, hunger = EXCLUDED.hunger,
		max_hunger = EXCLUDED.max_hunger, upkeep_rate = EXCLUDED.upkeep_rate,
		hunger_updated_at = EXCLUDED.hunger_updated_at,
		execution_state = EXCLUDED.execution_state, ai_profile = EXCLUDED.ai_profile,
		capabilities = EXCLUDED.capabilities, traits = EXCLUDED.traits,
		progression = EXCLUDED.progression, updated_at = EXCLUDED.updated_at,
		version = non_player_actors.version + 1
	WHERE (non_player_actors.owner_player_id, non_player_actors.template_id,
	       non_player_actors.name, non_player_actors.rank, non_player_actors.sub_rank,
	       non_player_actors.hp, non_player_actors.max_hp, non_player_actors.base_attack,
	       non_player_actors.base_defense, non_player_actors.speed, non_player_actors.count,
	       non_player_actors.hunger, non_player_actors.max_hunger,
	       non_player_actors.upkeep_rate, non_player_actors.execution_state,
	       non_player_actors.ai_profile, non_player_actors.traits, non_player_actors.progression)
	      IS DISTINCT FROM
	      (EXCLUDED.owner_player_id, EXCLUDED.template_id, EXCLUDED.name, EXCLUDED.rank,
	       EXCLUDED.sub_rank, EXCLUDED.hp, EXCLUDED.max_hp, EXCLUDED.base_attack,
	       EXCLUDED.base_defense, EXCLUDED.speed, EXCLUDED.count, EXCLUDED.hunger,
	       EXCLUDED.max_hunger, EXCLUDED.upkeep_rate, EXCLUDED.execution_state,
	       EXCLUDED.ai_profile, EXCLUDED.traits, EXCLUDED.progression);`

const nonPlayerActorsVariantNPCProjectionSQL = `
	INSERT INTO non_player_actors (
		actor_kind, owner_player_id, aperture_id, template_source, template_id,
		species, race, name, rank, sub_rank, talent, loyalty, hp, max_hp, count,
		lifecycle_state, execution_state, behavior_profile, capabilities, traits,
		progression, legacy_source, legacy_id, created_at, updated_at)
	SELECT 'VARIANT_NPC', a.player_id, n.aperture_id, 'VARIANT_RACE', n.race,
	       n.race, n.race, n.name, n.rank, n.sub_rank, n.talent, n.loyalty,
	       100, 100, 1,
	       CASE WHEN n.status = 'DEAD' THEN 'DEAD' ELSE 'ACTIVE' END,
	       CASE n.status WHEN 'DEAD' THEN 'DEAD' WHEN 'TRAINING' THEN 'TRAINING'
	            WHEN 'IDLE' THEN 'STORED' ELSE 'WORKING' END,
	       'WORKER',
	       '{"MOVE":true,"FOLLOW":true,"GUARD":true,"COMBAT":true,"PRODUCE":true,"REFINE":true,"TRAIN":true,"EQUIP_GU":true}'::jsonb,
	       jsonb_build_object('path', n.path, 'role', n.role),
	       jsonb_build_object('legacy_status', n.status),
	       'APERTURE_VARIANT_NPC', n.id, n.created_at, n.updated_at
	FROM aperture_variant_npcs n
	JOIN immortal_apertures a ON a.id = n.aperture_id
	WHERE ($1::uuid IS NULL OR a.player_id = $1)
	ON CONFLICT (legacy_source, legacy_id)
		WHERE legacy_source IS NOT NULL AND legacy_id IS NOT NULL
	DO UPDATE SET
		owner_player_id = EXCLUDED.owner_player_id, aperture_id = EXCLUDED.aperture_id,
		template_id = EXCLUDED.template_id, species = EXCLUDED.species, race = EXCLUDED.race,
		name = EXCLUDED.name, rank = EXCLUDED.rank, sub_rank = EXCLUDED.sub_rank,
		talent = EXCLUDED.talent, loyalty = EXCLUDED.loyalty,
		lifecycle_state = EXCLUDED.lifecycle_state, execution_state = EXCLUDED.execution_state,
		capabilities = EXCLUDED.capabilities, traits = EXCLUDED.traits,
		progression = EXCLUDED.progression, updated_at = EXCLUDED.updated_at,
		version = non_player_actors.version + 1
	WHERE (non_player_actors.owner_player_id, non_player_actors.aperture_id,
	       non_player_actors.template_id, non_player_actors.race, non_player_actors.name,
	       non_player_actors.rank, non_player_actors.sub_rank, non_player_actors.talent,
	       non_player_actors.loyalty, non_player_actors.lifecycle_state,
	       non_player_actors.execution_state, non_player_actors.traits,
	       non_player_actors.progression)
	      IS DISTINCT FROM
	      (EXCLUDED.owner_player_id, EXCLUDED.aperture_id, EXCLUDED.template_id,
	       EXCLUDED.race, EXCLUDED.name, EXCLUDED.rank, EXCLUDED.sub_rank,
	       EXCLUDED.talent, EXCLUDED.loyalty, EXCLUDED.lifecycle_state,
	       EXCLUDED.execution_state, EXCLUDED.traits, EXCLUDED.progression);`

const nonPlayerActorsLegacyCleanupSQL = `
	DELETE FROM non_player_actors a
	WHERE ($1::uuid IS NULL OR a.owner_player_id = $1)
	  AND ((a.legacy_source = 'PLAYER_PET'
	       AND NOT EXISTS (SELECT 1 FROM player_pets p WHERE p.id = a.legacy_id))
	   OR (a.legacy_source = 'APERTURE_VARIANT_NPC'
	       AND NOT EXISTS (SELECT 1 FROM aperture_variant_npcs n WHERE n.id = a.legacy_id)));`
