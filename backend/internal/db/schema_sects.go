package db

// Sect schema. sect_missions.target_id is polymorphic (item ids for GATHER,
// gu ids for DONATE) so it has no FK.

func schemaSects() error {
	if err := execSchema("sects", `
	CREATE TABLE IF NOT EXISTS sects (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL,
		region VARCHAR(50) NOT NULL,
		map_x INT NOT NULL DEFAULT 50,
		map_y INT NOT NULL DEFAULT 50,
		shield_until TIMESTAMP WITH TIME ZONE,
		buildings JSONB NOT NULL DEFAULT '{"main_hall": 1, "defensive_array": 1, "spirit_vein": 1, "treasury": 1, "training_ground": 1}',
		funds BIGINT NOT NULL DEFAULT 0,
		level INTEGER NOT NULL DEFAULT 1,
		require_approval BOOLEAN NOT NULL DEFAULT FALSE,
		require_promotion_approval BOOLEAN NOT NULL DEFAULT FALSE,
		last_maintenance_at TIMESTAMP WITH TIME ZONE,
		last_salary_paid_at TIMESTAMP WITH TIME ZONE,
		last_territory_tick_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_relocated_at TIMESTAMP WITH TIME ZONE,
		defensive_array_ready_at TIMESTAMP WITH TIME ZONE,
		unpaid_maintenance_count INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		return err
	}

	// Existing databases need the column added explicitly (fresh installs get it
	// from the CREATE above). Tracks consecutive failed daily maintenance fees at
	// level 1; reaching the grace limit disbands the sect.
	if err := execSchema("sects_add_unpaid_maintenance_count", `
		ALTER TABLE sects ADD COLUMN IF NOT EXISTS unpaid_maintenance_count INTEGER NOT NULL DEFAULT 0;`); err != nil {
		return err
	}

	// last_vault_decay_at belonged to the removed vault-decay feature; no code
	// reads or writes it any more, so existing databases drop it too.
	if err := execSchema("sects_drop_last_vault_decay_at", `
		ALTER TABLE sects DROP COLUMN IF EXISTS last_vault_decay_at;`); err != nil {
		return err
	}

	// Map markers and sect war lookups filter by region.
	if err := execSchema("sects_region_idx", `
	CREATE INDEX IF NOT EXISTS idx_sects_region ON sects (region);`); err != nil {
		return err
	}

	if err := execSchema("sect_missions", `
	CREATE TABLE IF NOT EXISTS sect_missions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		title VARCHAR(100) NOT NULL,
		description TEXT NOT NULL,
		mission_type VARCHAR(30) NOT NULL,
		target_id VARCHAR(50) NOT NULL,
		target_amount BIGINT NOT NULL DEFAULT 1,
		reward_contribution BIGINT NOT NULL DEFAULT 0,
		required_member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
		expires_at TIMESTAMP WITHOUT TIME ZONE,
		UNIQUE (sect_id, title)
	);`); err != nil {
		return err
	}
	if err := execSchema("sect_missions_expires_idx", `
	CREATE INDEX IF NOT EXISTS idx_sect_missions_expires ON sect_missions (expires_at);`); err != nil {
		return err
	}

	if err := execSchema("sect_members", `
	CREATE TABLE IF NOT EXISTS sect_members (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE UNIQUE,
		member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
		contribution_points BIGINT NOT NULL DEFAULT 0,
		joined_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		is_defending BOOLEAN NOT NULL DEFAULT FALSE
	);`); err != nil {
		return err
	}
	if err := execSchema("sect_members_sect_rank_idx", `
	CREATE INDEX IF NOT EXISTS idx_sect_members_sect_rank ON sect_members (sect_id, member_rank);`); err != nil {
		return err
	}
	// Legacy columns removed: master_id was never written, the territory_* pair
	// fed the deleted passive-contribution credit.
	if err := execSchema("sect_members_drop_legacy", `
		ALTER TABLE sect_members
			DROP COLUMN IF EXISTS master_id,
			DROP COLUMN IF EXISTS territory_pending,
			DROP COLUMN IF EXISTS territory_credited_at,
			DROP COLUMN IF EXISTS territory_credited_today;`); err != nil {
		return err
	}

	if err := execSchema("sect_join_requests", `
	CREATE TABLE IF NOT EXISTS sect_join_requests (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
		created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		UNIQUE (sect_id, player_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("sect_promotion_requests", `
	CREATE TABLE IF NOT EXISTS sect_promotion_requests (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		current_rank VARCHAR(30) NOT NULL,
		target_rank VARCHAR(30) NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
		created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		UNIQUE (sect_id, player_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("sect_promotion_requests_idx", `
	CREATE INDEX IF NOT EXISTS idx_sect_promotion_requests ON sect_promotion_requests (sect_id, status, created_at);`); err != nil {
		return err
	}

	if err := execSchema("player_sect_missions", `
	CREATE TABLE IF NOT EXISTS player_sect_missions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		mission_id UUID NOT NULL REFERENCES sect_missions(id) ON DELETE CASCADE,
		current_progress BIGINT NOT NULL DEFAULT 0,
		status VARCHAR(30) NOT NULL DEFAULT 'IN_PROGRESS',
		assigned_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		expires_at TIMESTAMP WITHOUT TIME ZONE,
		UNIQUE (player_id, mission_id)
	);`); err != nil {
		return err
	}
	// Index on foreign key to prevent full table scans on DELETE FROM sect_missions CASCADE
	if err := execSchema("player_sect_missions_mission_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_sect_missions_mission ON player_sect_missions (mission_id);`); err != nil {
		return err
	}
	// Composite index for background expiry worker
	if err := execSchema("player_sect_missions_status_expires_idx", `
	CREATE INDEX IF NOT EXISTS idx_player_sect_missions_status_expires ON player_sect_missions (status, expires_at);`); err != nil {
		return err
	}

	if err := execSchema("sect_defenders", `
	CREATE TABLE IF NOT EXISTS sect_defenders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		defender_type VARCHAR(20) NOT NULL DEFAULT 'member',
		player_id UUID REFERENCES players(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL DEFAULT '',
		monster_id VARCHAR(50) REFERENCES monster_templates(id) ON DELETE CASCADE,
		monster_rank INTEGER NOT NULL DEFAULT 1,
		monster_sub_rank INTEGER NOT NULL DEFAULT 1,
		power DOUBLE PRECISION NOT NULL DEFAULT 0,
		assigned_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
	);`); err != nil {
		return err
	}
	if err := execSchema("uq_sect_defender_player", `
	CREATE UNIQUE INDEX IF NOT EXISTS uq_sect_defender_player ON sect_defenders (player_id) WHERE defender_type = 'member' AND player_id IS NOT NULL;`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_defenders_sect", `
	CREATE INDEX IF NOT EXISTS idx_sect_defenders_sect ON sect_defenders (sect_id);`); err != nil {
		return err
	}

	if err := execSchema("sect_vault_materials", `
	CREATE TABLE IF NOT EXISTS sect_vault_materials (
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		item_id VARCHAR(100) NOT NULL,
		quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (sect_id, item_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_vault_materials_item", `
	CREATE INDEX IF NOT EXISTS idx_sect_vault_materials_item ON sect_vault_materials (item_id) WHERE quantity > 0;`); err != nil {
		return err
	}

	if err := execSchema("sect_vault_gu", `
	CREATE TABLE IF NOT EXISTS sect_vault_gu (
		id VARCHAR(80) PRIMARY KEY,
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		gu_id VARCHAR(50) NOT NULL,
		rank INTEGER NOT NULL DEFAULT 1 CHECK (rank >= 1 AND rank <= 9),
		quantity BIGINT NOT NULL DEFAULT 1 CHECK (quantity >= 0),
		instance_id VARCHAR(80) DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT chk_sect_vault_gu_immortal_qty CHECK (rank < 6 OR quantity <= 1)
	);`); err != nil {
		return err
	}
	// cost was a snapshot of the contribution price (= dictionary base price since
	// the VaultCost rule changed) and hunger was the donated Gu's hunger, unused
	// since vault hunger decay was removed and withdrawals always start at 1.
	// Both are derived on read now, so existing databases drop them.
	if err := execSchema("sect_vault_gu_drop_derived_columns", `
		ALTER TABLE sect_vault_gu
			DROP COLUMN IF EXISTS cost,
			DROP COLUMN IF EXISTS hunger;`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_vault_gu_sect_gu", `
	CREATE INDEX IF NOT EXISTS idx_sect_vault_gu_sect_gu ON sect_vault_gu (sect_id, gu_id, rank);`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_vault_gu_gu_id", `
	CREATE INDEX IF NOT EXISTS idx_sect_vault_gu_gu_id ON sect_vault_gu (gu_id, rank);`); err != nil {
		return err
	}
	if err := execSchema("idx_sect_vault_gu_gu_id_lower", `
	CREATE INDEX IF NOT EXISTS idx_sect_vault_gu_gu_id_lower ON sect_vault_gu (LOWER(gu_id), rank);`); err != nil {
		return err
	}
	// Tiên Cổ (rank>=6) trong kho: quantity luôn 1, 1 gu_id chỉ 1 row toàn server.
	if err := execSchema("uq_sect_vault_gu_immortal", `
	CREATE UNIQUE INDEX IF NOT EXISTS uq_sect_vault_gu_immortal
	ON sect_vault_gu (gu_id) WHERE rank >= 6 AND quantity > 0;`); err != nil {
		return err
	}

	if err := execSchema("sect_vault_recipes", `
	CREATE TABLE IF NOT EXISTS sect_vault_recipes (
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		recipe_id VARCHAR(100) NOT NULL,
		completion_rate FLOAT NOT NULL DEFAULT 100.0 CHECK (completion_rate >= 0 AND completion_rate <= 100.0),
		unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (sect_id, recipe_id)
	);`); err != nil {
		return err
	}

	return nil
}
