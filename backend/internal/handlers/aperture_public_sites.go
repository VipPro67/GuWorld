package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// EnsureConfiguredPublicApertures idempotently materializes configured world
// sites. Consumed site rows stay terminal and are never respawned.
func EnsureConfiguredPublicApertures(ctx context.Context) error {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT site_key FROM public_aperture_site_configs
		WHERE is_active = TRUE AND lifecycle_state = 'UNSPAWNED'
		ORDER BY site_key LIMIT 50
	`)
	if err != nil {
		return fmt.Errorf("finding configured public aperture sites: %w", err)
	}
	var siteKeys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return fmt.Errorf("reading public aperture site key: %w", err)
		}
		siteKeys = append(siteKeys, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterating configured public aperture sites: %w", err)
	}
	rows.Close()

	for _, siteKey := range siteKeys {
		if err := materializePublicApertureSite(ctx, siteKey); err != nil {
			logger.S.Errorf("failed to materialize public aperture site %s: %v", siteKey, err)
		}
	}
	return nil
}

func materializePublicApertureSite(ctx context.Context, siteKey string) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning public aperture site creation: %w", err)
	}
	defer tx.Rollback()

	var (
		regionID string
		name     string
		x, y     float64
		rank     int
		grade    string
		area     int64
		daoMarks []byte
	)
	err = tx.QueryRowContext(ctx, `
		SELECT s.region_id, r.name, s.x, s.y, s.aperture_rank, s.grade, s.area_size, s.dao_marks
		FROM public_aperture_site_configs s
		JOIN regions r ON r.id = s.region_id
		WHERE s.site_key = $1 AND s.is_active = TRUE AND s.lifecycle_state = 'UNSPAWNED'
		FOR UPDATE OF s
	`, siteKey).Scan(&regionID, &name, &x, &y, &rank, &grade, &area, &daoMarks)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("locking configured site: %w", err)
	}

	var apertureID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO immortal_apertures (
			player_id, aperture_rank, public_state, grade, time_rate, aperture_hp, max_aperture_hp,
			area_size, immortal_essence, essence_production_per_year, internal_years_passed,
			next_tribulation_year, state, dao_marks, is_placed, placed_region, placed_x, placed_y,
			return_region, return_x, return_y
		)
		VALUES (NULL, $1, 'PUBLIC', $2, 10, 10000, 10000, $3, 0, 10, 0, 10, 'STABLE', $4::jsonb,
			TRUE, $5, $6, $7, $5, $6, $7)
		RETURNING id::text
	`, rank, grade, area, string(daoMarks), regionID, x, y).Scan(&apertureID)
	if err != nil {
		return fmt.Errorf("creating public aperture state: %w", err)
	}
	var siteDaoMarks models.DaoMarksMap
	if err := json.Unmarshal(daoMarks, &siteDaoMarks); err != nil {
		return fmt.Errorf("decoding configured public aperture dao marks: %w", err)
	}
	if err := GenerateApertureResourcePoints(tx, apertureID, "", siteDaoMarks, grade, seedPointCountForGrade(grade)); err != nil {
		return fmt.Errorf("seeding configured public aperture resource points: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO aperture_portals
			(aperture_id, owner_id, region_id, x, y, name, access_policy, concealment_power, is_active, updated_at)
		VALUES ($1, NULL, $2, $3, $4, $5, 'OPEN', 0, TRUE, NOW())
	`, apertureID, regionID, x, y, "Môn Hộ Phúc Địa "+name); err != nil {
		return fmt.Errorf("creating public aperture portal: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE public_aperture_site_configs
		SET aperture_id = $2, lifecycle_state = 'ACTIVE', updated_at = NOW()
		WHERE site_key = $1 AND lifecycle_state = 'UNSPAWNED'
	`, siteKey, apertureID); err != nil {
		return fmt.Errorf("linking configured site to aperture state: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing configured public aperture: %w", err)
	}
	publishAperturePortalMarkerChange("portal_spawn", regionID)
	return nil
}
