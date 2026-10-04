package db

import (
	"fmt"
	"strings"
)

const emptyBlockV2 = `{"schema_version":2,"blocks":[]}`

// effectCatalogColumns upgrades effect owners without requiring a destructive
// database reset. Block-only constraints are installed separately, only after
// the guarded backfill has proven that no legacy rows remain.
func effectCatalogColumns(table string) error {
	if _, ok := effectOwnerTables[table]; !ok {
		return fmt.Errorf("unsupported effect owner %q", table)
	}
	current, err := effectCatalogColumnsCurrent(table)
	if err != nil {
		return fmt.Errorf("inspect %s effect catalog columns: %w", table, err)
	}
	if current {
		return nil
	}
	return execSchema(table+"_effect_catalog_upgrade", fmt.Sprintf(`
	ALTER TABLE %s ADD COLUMN IF NOT EXISTS schema_version INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE %s ALTER COLUMN schema_version SET DEFAULT 2;
	ALTER TABLE %s ADD COLUMN IF NOT EXISTS catalog_revision BIGINT NOT NULL DEFAULT 0;
	ALTER TABLE %s ADD COLUMN IF NOT EXISTS content_hash TEXT NOT NULL DEFAULT '';
	ALTER TABLE %s ALTER COLUMN effects SET DEFAULT '{"schema_version":2,"blocks":[]}'::jsonb;
`, table, table, table, table, table))
}

// effectCatalogColumnsCurrent prevents an already-completed upgrade from
// issuing ALTER TABLE on every boot. Even ALTER ... IF NOT EXISTS takes a
// strong table lock in PostgreSQL, which can contend with a serving replica.
func effectCatalogColumnsCurrent(table string) (bool, error) {
	rows, err := schemaDB().Query(`
		SELECT column_name, data_type, is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = $1
		  AND column_name IN ('effects', 'schema_version', 'catalog_revision', 'content_hash')`, table)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	type columnInfo struct{ dataType, nullable, defaultValue string }
	columns := make(map[string]columnInfo, 4)
	for rows.Next() {
		var name, dataType, nullable, value string
		if err := rows.Scan(&name, &dataType, &nullable, &value); err != nil {
			return false, err
		}
		columns[name] = columnInfo{strings.ToLower(dataType), nullable, strings.ToLower(value)}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	effectsDefault := columns["effects"].defaultValue
	return len(columns) == 4 &&
		columns["effects"].dataType == "jsonb" && columns["effects"].nullable == "NO" &&
		columns["schema_version"].dataType == "integer" && columns["schema_version"].nullable == "NO" && columns["schema_version"].defaultValue == "2" &&
		columns["catalog_revision"].dataType == "bigint" && columns["catalog_revision"].nullable == "NO" && columns["catalog_revision"].defaultValue == "0" &&
		columns["content_hash"].dataType == "text" && columns["content_hash"].nullable == "NO" && strings.HasPrefix(columns["content_hash"].defaultValue, "''") &&
		strings.Contains(effectsDefault, "schema_version") &&
		strings.Contains(effectsDefault, "blocks"), nil
}

var effectOwnerTables = map[string]struct{}{
	"gu_dictionary": {}, "item_dictionary": {}, "race_dictionary": {},
	"constitution_dictionary": {}, "status_dictionary": {},
	"killer_moves": {}, "aperture_variant_events": {},
}

func schemaEffectCatalogMigration() error {
	if err := execSchema("effect_catalog_revisions", `
	CREATE TABLE IF NOT EXISTS effect_catalog_revisions (
		revision BIGSERIAL PRIMARY KEY,
		status VARCHAR(20) NOT NULL CHECK (status IN ('SNAPSHOT','VALIDATED','ACTIVE','ROLLED_BACK','CONFLICT')),
		content_hash TEXT NOT NULL DEFAULT '',
		conflicts JSONB NOT NULL DEFAULT '[]',
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
		completed_at TIMESTAMP WITH TIME ZONE
	);`); err != nil {
		return err
	}
	return execSchema("effect_catalog_snapshots", `
	CREATE TABLE IF NOT EXISTS effect_catalog_snapshots (
		revision BIGINT NOT NULL REFERENCES effect_catalog_revisions(revision) ON DELETE RESTRICT,
		table_name TEXT NOT NULL,
		row_key JSONB NOT NULL,
		effects_before JSONB NOT NULL,
		schema_version_before INTEGER NOT NULL,
		catalog_revision_before BIGINT NOT NULL,
		content_hash_before TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
		PRIMARY KEY (revision, table_name, row_key)
	);`)
}
