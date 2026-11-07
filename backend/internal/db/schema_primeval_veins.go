package db

// primevalVeinsSchema repairs old installations before enforcing tile uniqueness.
const primevalVeinsSchema = `
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
	-- Repair legacy duplicates and install the arbiter in one atomic statement.
	DO $migration$
	BEGIN
		-- Block row-locking harvests as well as writes to avoid lock inversion.
		LOCK TABLE primeval_veins IN EXCLUSIVE MODE;
		CREATE TABLE IF NOT EXISTS primeval_vein_duplicate_archive (
			id UUID PRIMARY KEY,
			row_data JSONB NOT NULL,
			archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		-- Prefer a live vein, then the oldest original row; UUID breaks ties.
		-- Archive full rows without FKs so their production history survives.
		WITH ranked AS (
			SELECT id, ROW_NUMBER() OVER (
				PARTITION BY region_id, x, y
				ORDER BY (expires_at > NOW()) DESC, created_at ASC, id ASC
			) AS rn
			FROM primeval_veins
		), removed AS (
			DELETE FROM primeval_veins v USING ranked r
			WHERE v.id = r.id AND r.rn > 1
			RETURNING v.*
		)
		INSERT INTO primeval_vein_duplicate_archive (id, row_data)
		SELECT id, to_jsonb(removed) FROM removed;
		-- Reuse the fresh-install constraint index name; no redundant index.
		CREATE UNIQUE INDEX IF NOT EXISTS primeval_veins_region_id_x_y_key
			ON primeval_veins (region_id, x, y);
	END
	$migration$;
	CREATE INDEX IF NOT EXISTS idx_primeval_veins_region ON primeval_veins (region_id, x, y);
	CREATE INDEX IF NOT EXISTS idx_primeval_veins_expires ON primeval_veins (expires_at);`
