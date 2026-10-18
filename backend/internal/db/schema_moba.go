package db

// Schema for the realtime (MOBA-style) PvP skill cards.
//
// gu_moba_profiles describes *how* a Gu is aimed and shaped in the realtime
// mode, plus its realtime timings. The damage/status/shield payload still comes
// from gu_dictionary.effects, so this table stays additive: a Gu without a row
// falls back to TARGET/SINGLE and remains castable.
//
// Columns are flat (not JSONB) so the generic CSV import workflow maps header
// names straight onto columns. `cast_range` avoids the reserved word RANGE.
func schemaMoba() error {
	return execSchema("gu_moba_profiles", `
	CREATE TABLE IF NOT EXISTS gu_moba_profiles (
		gu_id VARCHAR(50) NOT NULL,
		rank INTEGER NOT NULL,
		cast_mode VARCHAR(20) NOT NULL DEFAULT 'TARGET',
		shape VARCHAR(20) NOT NULL DEFAULT 'SINGLE',
		radius DOUBLE PRECISION NOT NULL DEFAULT 0,
		length DOUBLE PRECISION NOT NULL DEFAULT 0,
		width DOUBLE PRECISION NOT NULL DEFAULT 0,
		angle_deg DOUBLE PRECISION NOT NULL DEFAULT 0,
		cast_range DOUBLE PRECISION NOT NULL DEFAULT 8,
		cast_time_ms INTEGER NOT NULL DEFAULT 0,
		recovery_ms INTEGER NOT NULL DEFAULT 0,
		cooldown_ms INTEGER NOT NULL DEFAULT 0,
		projectile_speed DOUBLE PRECISION NOT NULL DEFAULT 0,
		pierce BOOLEAN NOT NULL DEFAULT false,
		max_targets INTEGER NOT NULL DEFAULT 1,
		move_distance DOUBLE PRECISION NOT NULL DEFAULT 0,
		PRIMARY KEY (gu_id, rank),
		FOREIGN KEY (gu_id, rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE CASCADE
	);`)
}
