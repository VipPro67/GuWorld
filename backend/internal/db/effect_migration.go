package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/effectmigration"
)

type EffectMigrationConflict struct{ Table, Key, Code, Reason string }
type EffectMigrationResult struct {
	Revision         int64
	Updated, Skipped int
	Conflicts        []EffectMigrationConflict
}

type effectOwner struct {
	table, keySQL, keyWhere string
	format                  effectmigration.Format
}

var migrationOwners = []effectOwner{
	{"gu_dictionary", `jsonb_build_object('id',id,'rank',rank)`, `id=$1 AND rank=$2`, effectmigration.FormatGu},
	{"item_dictionary", `jsonb_build_object('id',id)`, `id=$1`, effectmigration.FormatItem},
	{"race_dictionary", `jsonb_build_object('key',key)`, `key=$1`, effectmigration.FormatGameFX},
	{"constitution_dictionary", `jsonb_build_object('key',key)`, `key=$1`, effectmigration.FormatGameFX},
	{"status_dictionary", `jsonb_build_object('key',key)`, `key=$1`, effectmigration.FormatGameFX},
	{"killer_moves", `jsonb_build_object('id',id)`, `id=$1::uuid`, effectmigration.FormatKiller},
	// aperture_variant_events.effects has no runtime executor. Keep the owner
	// outside cutover rather than manufacturing an envelope with false parity.
}

func effectHash(raw []byte) (string, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// BackfillEffectsBlockV2 snapshots every effect owner before changing anything.
// Any unsupported legacy payload or hash mismatch aborts the whole transaction.
func BackfillEffectsBlockV2(ctx context.Context) (EffectMigrationResult, error) {
	result := EffectMigrationResult{}
	tx, err := DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(77202402)`); err != nil {
		return result, err
	}
	if err = tx.QueryRowContext(ctx, `INSERT INTO effect_catalog_revisions(status) VALUES('SNAPSHOT') RETURNING revision`).Scan(&result.Revision); err != nil {
		return result, err
	}

	for _, owner := range migrationOwners {
		rows, qerr := tx.QueryContext(ctx, fmt.Sprintf(`SELECT %s, effects::text, schema_version, catalog_revision, content_hash FROM %s FOR UPDATE`, owner.keySQL, owner.table))
		if qerr != nil {
			return result, qerr
		}
		type storedEffect struct {
			keyRaw, effects []byte
			schemaVersion   int
			oldRevision     int64
			storedHash      string
		}
		var stored []storedEffect
		for rows.Next() {
			var row storedEffect
			if err = rows.Scan(&row.keyRaw, &row.effects, &row.schemaVersion, &row.oldRevision, &row.storedHash); err != nil {
				rows.Close()
				return result, err
			}
			stored = append(stored, row)
		}
		if err = rows.Close(); err != nil {
			return result, err
		}
		for _, row := range stored {
			keyRaw, effects, schemaVersion, oldRevision, storedHash := row.keyRaw, row.effects, row.schemaVersion, row.oldRevision, row.storedHash
			currentHash, hashErr := effectHash(effects)
			if hashErr != nil {
				return result, hashErr
			}
			key := string(keyRaw)
			if storedHash != "" && storedHash != currentHash {
				result.Conflicts = append(result.Conflicts, EffectMigrationConflict{Table: owner.table, Key: key, Code: "content_hash_mismatch", Reason: "content_hash does not match current database payload"})
				continue
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO effect_catalog_snapshots(revision,table_name,row_key,effects_before,schema_version_before,catalog_revision_before,content_hash_before) VALUES($1,$2,$3::jsonb,$4::jsonb,$5,$6,$7) ON CONFLICT DO NOTHING`, result.Revision, owner.table, keyRaw, effects, schemaVersion, oldRevision, currentHash); err != nil {
				return result, err
			}
			if len(effectcatalog.ValidateBlockV2(effects, effectcatalog.Location{Source: owner.table, Key: key})) == 0 {
				result.Skipped++
				if err = updateEffectMetadata(ctx, tx, owner, keyRaw, effects, result.Revision, currentHash); err != nil {
					return result, err
				}
				continue
			}
			converted := effectmigration.Convert(owner.format, key, effects)
			decision := effectmigration.EvaluateMigrationGate(converted.Report)
			if !decision.MigrationGate {
				result.Conflicts = append(result.Conflicts, EffectMigrationConflict{Table: owner.table, Key: key, Code: decision.ErrorCode, Reason: decision.Reason})
				continue
			}
			newHash, _ := effectHash(converted.Envelope)
			if err = updateEffectMetadata(ctx, tx, owner, keyRaw, converted.Envelope, result.Revision, newHash); err != nil {
				return result, err
			}
			result.Updated++
		}
	}
	if len(result.Conflicts) != 0 {
		return result, fmt.Errorf("effect migration revision %d has %d conflict(s)", result.Revision, len(result.Conflicts))
	}
	if _, err = tx.ExecContext(ctx, `UPDATE effect_catalog_revisions SET status='VALIDATED', completed_at=now() WHERE revision=$1`, result.Revision); err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func keyArgs(raw []byte) ([]any, error) {
	var key map[string]any
	if err := json.Unmarshal(raw, &key); err != nil {
		return nil, err
	}
	if id, ok := key["id"]; ok {
		if rank, ranked := key["rank"]; ranked {
			return []any{id, rank}, nil
		}
		return []any{id}, nil
	}
	return []any{key["key"]}, nil
}
func updateEffectMetadata(ctx context.Context, tx *sql.Tx, owner effectOwner, keyRaw, effects []byte, revision int64, hash string) error {
	args, err := keyArgs(keyRaw)
	if err != nil {
		return err
	}
	args = append(args, string(effects), revision, hash)
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET effects=$%d::jsonb,schema_version=2,catalog_revision=$%d,content_hash=$%d WHERE %s`, owner.table, len(args)-2, len(args)-1, len(args), owner.keyWhere), args...)
	return err
}

// RollbackEffectMigration restores the exact snapshotted payload and metadata;
// it never attempts an incomplete reverse conversion.
func RollbackEffectMigration(ctx context.Context, revision int64) error {
	tx, err := DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(77202402)`); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT table_name,row_key,effects_before::text,schema_version_before,catalog_revision_before,content_hash_before FROM effect_catalog_snapshots WHERE revision=$1 ORDER BY table_name`, revision)
	if err != nil {
		return err
	}
	type snapshot struct {
		table        string
		key, effects []byte
		version      int
		oldRevision  int64
		hash         string
	}
	var snapshots []snapshot
	for rows.Next() {
		var s snapshot
		if err = rows.Scan(&s.table, &s.key, &s.effects, &s.version, &s.oldRevision, &s.hash); err != nil {
			rows.Close()
			return err
		}
		snapshots = append(snapshots, s)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for _, s := range snapshots {
		table, key, effects, version, oldRevision, hash := s.table, s.key, s.effects, s.version, s.oldRevision, s.hash
		var owner *effectOwner
		for i := range migrationOwners {
			if migrationOwners[i].table == table {
				owner = &migrationOwners[i]
				break
			}
		}
		if owner == nil {
			rows.Close()
			return fmt.Errorf("snapshot contains unsupported table %q", table)
		}
		args, e := keyArgs(key)
		if e != nil {
			rows.Close()
			return e
		}
		args = append(args, string(effects), version, oldRevision, hash)
		q := fmt.Sprintf(`UPDATE %s SET effects=$%d::jsonb,schema_version=$%d,catalog_revision=$%d,content_hash=$%d WHERE %s`, table, len(args)-3, len(args)-2, len(args)-1, len(args), owner.keyWhere)
		if _, err = tx.ExecContext(ctx, q, args...); err != nil {
			rows.Close()
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE effect_catalog_revisions SET status='ROLLED_BACK',completed_at=now() WHERE revision=$1`, revision); err != nil {
		return err
	}
	return tx.Commit()
}

// InstallEffectBlockV2Constraints is the second rollout stage. It refuses to
// install anything until every owner is block-only, then validates each check.
func InstallEffectBlockV2Constraints(ctx context.Context) error {
	tx, err := DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, owner := range migrationOwners {
		var legacy int
		q := fmt.Sprintf(`SELECT count(*) FROM %s WHERE jsonb_typeof(effects)<>'object' OR effects->>'schema_version'<>'2' OR jsonb_typeof(effects->'blocks')<>'array' OR effects-'schema_version'-'blocks'<>'{}'::jsonb`, owner.table)
		if err = tx.QueryRowContext(ctx, q).Scan(&legacy); err != nil {
			return err
		}
		if legacy != 0 {
			return fmt.Errorf("%s still has %d legacy effect row(s)", owner.table, legacy)
		}
		name := "chk_" + owner.table + "_effects_block_v2"
		if _, err = tx.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s; ALTER TABLE %s ADD CONSTRAINT %s CHECK (jsonb_typeof(effects)='object' AND effects->>'schema_version'='2' AND jsonb_typeof(effects->'blocks')='array' AND effects-'schema_version'-'blocks'='{}'::jsonb) NOT VALID; ALTER TABLE %s VALIDATE CONSTRAINT %s`, owner.table, name, owner.table, name, owner.table, name)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
