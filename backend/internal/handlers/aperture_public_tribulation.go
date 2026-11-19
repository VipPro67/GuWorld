package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const publicApertureWorkerIntervalSeconds = 60.0

// ProcessPublicApertureTribulations advances natural tribulation time for
// public apertures. Their environmental events settle automatically because
// public apertures have no owner who can defend them.
func ProcessPublicApertureTribulations(ctx context.Context) error {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id::text FROM immortal_apertures
		WHERE player_id IS NULL AND public_state = 'PUBLIC'
		ORDER BY id LIMIT 100
	`)
	if err != nil {
		return fmt.Errorf("finding public apertures for natural tribulation: %w", err)
	}
	var apertureIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("reading public aperture for natural tribulation: %w", err)
		}
		apertureIDs = append(apertureIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterating public apertures for natural tribulation: %w", err)
	}
	rows.Close()
	if len(apertureIDs) == 0 {
		return nil
	}

	damageFraction, configured := publicApertureTribulationDamageFraction()
	for _, apertureID := range apertureIDs {
		if err := processPublicApertureTribulation(ctx, apertureID, damageFraction, configured); err != nil {
			return err
		}
	}
	return nil
}

func publicApertureTribulationDamageFraction() (float64, bool) {
	raw := strings.TrimSpace(os.Getenv("PUBLIC_APERTURE_TRIBULATION_HP_DAMAGE_FRACTION"))
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > 1 {
		return 0, false
	}
	return value, true
}

func processPublicApertureTribulation(ctx context.Context, apertureID string, damageFraction float64, configured bool) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning public environmental tribulation: %w", err)
	}
	defer tx.Rollback()

	var rank int
	var timeRate, hp, maxHP, internalYears, nextYear, rankStartYear float64
	var daoMarksJSON []byte
	err = tx.QueryRowContext(ctx, `
		SELECT aperture_rank, time_rate, aperture_hp, max_aperture_hp,
		       internal_years_passed, next_tribulation_year, rank_start_year,
		       COALESCE(dao_marks, '{}'::jsonb)
		FROM immortal_apertures
		WHERE id = $1 AND player_id IS NULL AND public_state = 'PUBLIC'
		FOR UPDATE
	`, apertureID).Scan(&rank, &timeRate, &hp, &maxHP, &internalYears, &nextYear, &rankStartYear, &daoMarksJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("locking public aperture environmental state: %w", err)
	}
	if timeRate <= 0 {
		timeRate = 1
	}
	newYears := internalYears + publicApertureWorkerIntervalSeconds*timeRate/86400
	if newYears < nextYear {
		if _, err := tx.ExecContext(ctx, `
			UPDATE immortal_apertures SET internal_years_passed = $2, updated_at = NOW()
			WHERE id = $1 AND public_state = 'PUBLIC'
		`, apertureID, newYears); err != nil {
			return fmt.Errorf("advancing public aperture internal time: %w", err)
		}
		return tx.Commit()
	}
	if !configured {
		if _, err := tx.ExecContext(ctx, `
			UPDATE immortal_apertures SET internal_years_passed = $2, updated_at = NOW()
			WHERE id = $1 AND public_state = 'PUBLIC'
		`, apertureID, newYears); err != nil {
			return fmt.Errorf("advancing public aperture time while environmental damage is unconfigured: %w", err)
		}
		return tx.Commit()
	}

	tribulationType := models.GetTribulationTypeAtYear(rank, rankStartYear, nextYear)
	if tribulationType == "" {
		_, tribulationType = models.GetNextTribulationInfo(rank, rankStartYear, internalYears)
	}
	tribulationType = models.SanitizeTribulationTypeForRank(rank, tribulationType)
	if tribulationType == "" {
		return fmt.Errorf("no tribulation type configured for public aperture rank %d", rank)
	}
	path := publicApertureTribulationPath(daoMarksJSON)
	newHP := math.Max(0, hp-maxHP*damageFraction)
	nextTribulationYear, _ := models.GetNextTribulationInfo(rank, rankStartYear, newYears)
	nextTribulationYear = guardNextTribulationYear(nextTribulationYear, newYears)

	var eventMetadata []byte
	if eventMetadata, err = json.Marshal(map[string]any{
		"source":          "public_environment",
		"damage_fraction": damageFraction,
	}); err != nil {
		return fmt.Errorf("encoding public environmental event metadata: %w", err)
	}
	eventID := uuid.New()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tribulation_events (
			id, aperture_id, player_id, tribulation_type, path, status, stage,
			aperture_hp_before, aperture_hp_after, reward_dao_marks,
			started_at, ended_at, archetype, hazard_metadata
		) VALUES ($1, $2, NULL, $3, $4, 'FAILED', 'RESOLVED', $5, $6, 0, NOW(), NOW(), 'ENVIRONMENTAL', $7::jsonb)
	`, eventID, apertureID, tribulationType, path, hp, newHP, string(eventMetadata)); err != nil {
		return fmt.Errorf("recording public environmental tribulation settlement: %w", err)
	}
	state := "DAMAGED"
	if newHP <= 0 {
		state = "DESTROYED"
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE immortal_apertures
		SET aperture_hp = $2, state = $3, internal_years_passed = $4,
		    next_tribulation_year = $5, updated_at = NOW()
		WHERE id = $1 AND public_state = 'PUBLIC'
	`, apertureID, newHP, state, newYears, nextTribulationYear); err != nil {
		return fmt.Errorf("applying public environmental tribulation settlement: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing public environmental tribulation settlement: %w", err)
	}
	return nil
}

func publicApertureTribulationPath(raw []byte) string {
	marks := map[string]int64{}
	if json.Unmarshal(raw, &marks) != nil {
		return "environment"
	}
	maxMarks := int64(0)
	paths := make([]string, 0, len(marks))
	for path, value := range marks {
		path = gamefx.CleanPath(path)
		if path == "" || value <= 0 {
			continue
		}
		if value > maxMarks {
			maxMarks = value
			paths = paths[:0]
			paths = append(paths, path)
		} else if value == maxMarks {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return "environment"
	}
	sort.Strings(paths)
	return paths[0]
}
