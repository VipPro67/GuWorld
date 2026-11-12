package db

// Teleport array schema. Defines player-built and sect-built teleportation
// arrays across the Five Regions (Ngũ Vực) map.

func schemaTeleport() error {
	if err := execSchema("teleport_arrays", `
	CREATE TABLE IF NOT EXISTS teleport_arrays (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) NOT NULL,
		owner_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		sect_id VARCHAR(50) REFERENCES sects(id) ON DELETE SET NULL,
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		x INT NOT NULL,
		y INT NOT NULL,
		tier VARCHAR(20) NOT NULL DEFAULT 'TIER_1',
		max_range INT NOT NULL DEFAULT 1000,
		is_cross_region BOOLEAN NOT NULL DEFAULT FALSE,
		direction_type VARCHAR(20) NOT NULL DEFAULT 'TWO_WAY',
		access_policy VARCHAR(20) NOT NULL DEFAULT 'PUBLIC',
		fee_enabled BOOLEAN NOT NULL DEFAULT FALSE,
		fee_currency VARCHAR(30) NOT NULL DEFAULT 'primeval_stones',
		fee_amount BIGINT NOT NULL DEFAULT 0,
		vault_balance BIGINT NOT NULL DEFAULT 0,
		current_durability INT NOT NULL DEFAULT 100,
		max_durability INT NOT NULL DEFAULT 100,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		effect_enabled BOOLEAN NOT NULL DEFAULT FALSE,
		effect_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT uq_teleport_array_cell UNIQUE (region_id, x, y)
	);`); err != nil {
		return err
	}
	if err := execSchema("teleport_arrays_effect_profile", `
	ALTER TABLE teleport_arrays ADD COLUMN IF NOT EXISTS effect_enabled BOOLEAN NOT NULL DEFAULT FALSE;
	ALTER TABLE teleport_arrays ADD COLUMN IF NOT EXISTS effect_profile JSONB NOT NULL DEFAULT '{}'::jsonb;
	CREATE UNIQUE INDEX IF NOT EXISTS uq_teleport_array_owner_key ON teleport_arrays(id, owner_id);`); err != nil {
		return err
	}
	if err := execSchema("teleport_array_gu_reservations", `
	CREATE TABLE IF NOT EXISTS teleport_array_gu_reservations (
		gu_instance_id VARCHAR(80) PRIMARY KEY,
		array_id UUID NOT NULL,
		owner_id UUID NOT NULL,
		gu_id VARCHAR(50) NOT NULL,
		gu_rank INTEGER NOT NULL,
		gu_location VARCHAR(30) NOT NULL,
		gu_is_locked BOOLEAN NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT fk_teleport_array_gu_reservation_gu FOREIGN KEY (gu_instance_id, owner_id, gu_id, gu_rank, gu_location, gu_is_locked)
			REFERENCES player_gu(id, player_id, gu_id, gu_rank, location, is_locked) ON DELETE NO ACTION ON UPDATE RESTRICT DEFERRABLE INITIALLY DEFERRED,
		CONSTRAINT fk_teleport_array_gu_reservation_array FOREIGN KEY (array_id, owner_id)
			REFERENCES teleport_arrays(id, owner_id) ON DELETE CASCADE ON UPDATE RESTRICT
	);
	CREATE INDEX IF NOT EXISTS idx_teleport_array_gu_reservations_array ON teleport_array_gu_reservations(array_id);`); err != nil {
		return err
	}

	if err := execSchema("teleport_arrays_indexes", `
	CREATE INDEX IF NOT EXISTS idx_teleport_arrays_owner ON teleport_arrays(owner_id);
	CREATE INDEX IF NOT EXISTS idx_teleport_arrays_sect ON teleport_arrays(sect_id);
	CREATE INDEX IF NOT EXISTS idx_teleport_arrays_region_pos ON teleport_arrays(region_id, x, y);
	CREATE INDEX IF NOT EXISTS idx_teleport_arrays_active ON teleport_arrays(is_active);
	`); err != nil {
		return err
	}

	if err := execSchema("teleport_array_logs", `
	CREATE TABLE IF NOT EXISTS teleport_array_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		array_id UUID NOT NULL REFERENCES teleport_arrays(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		from_region VARCHAR(50) NOT NULL,
		from_x INT NOT NULL,
		from_y INT NOT NULL,
		to_region VARCHAR(50) NOT NULL,
		to_x INT NOT NULL,
		to_y INT NOT NULL,
		fee_paid BIGINT NOT NULL DEFAULT 0,
		fee_currency VARCHAR(30) NOT NULL DEFAULT 'primeval_stones',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}

	if err := execSchema("teleport_array_logs_indexes", `
	CREATE INDEX IF NOT EXISTS idx_teleport_array_logs_array ON teleport_array_logs(array_id);
	CREATE INDEX IF NOT EXISTS idx_teleport_array_logs_user ON teleport_array_logs(user_id);
	`); err != nil {
		return err
	}

	return nil
}
