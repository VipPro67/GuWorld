package db

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var (
	validMobaCastModes = map[string]bool{"TARGET": true, "DIRECTION": true, "GROUND": true, "SELF": true}
	validMobaShapes    = map[string]bool{"SINGLE": true, "LINE": true, "CIRCLE": true, "CONE": true}
)

// seedMobaProfilesFromCSV reads GuMoba.csv and upserts gu_moba_profiles.
//
// Every row is validated before the first write so a malformed CSV cannot leave
// the table half-updated. Rows referencing a missing (gu_id, rank) are rejected
// by the foreign key, which surfaces as an upsert error naming the row.
func seedMobaProfilesFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	col := func(name string) int { return colIndex(header, name) }
	idIdx := col("gu_id")
	rankIdx := col("rank")
	if idIdx < 0 || rankIdx < 0 {
		return fmt.Errorf("%s missing required columns (gu_id, rank)", filename)
	}

	type mobaRow struct {
		id, rank                                       string
		castMode, shape                                string
		radius, length, width, angle, castRange        float64
		projSpeed, moveDistance                        float64
		castTimeMs, recoveryMs, cooldownMs, maxTargets int
		pierce                                         bool
		line                                           int
	}

	seen := map[string]mobaRow{}
	order := []string{}
	line := 1
	for {
		line++
		values, readErr := r.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("%s line %d: %w", filename, line, readErr)
		}
		if len(values) == 0 {
			continue
		}
		get := func(idx int) string {
			if idx < 0 || idx >= len(values) {
				return ""
			}
			return strings.TrimSpace(values[idx])
		}
		id := get(idIdx)
		rank := get(rankIdx)
		if id == "" || rank == "" {
			continue
		}

		entry := mobaRow{
			id:         id,
			rank:       rank,
			castMode:   strings.ToUpper(defaultString(get(col("cast_mode")), "TARGET")),
			shape:      strings.ToUpper(defaultString(get(col("shape")), "SINGLE")),
			maxTargets: 1,
			line:       line,
		}
		if !validMobaCastModes[entry.castMode] {
			return fmt.Errorf("%s line %d: invalid cast_mode %q", filename, line, entry.castMode)
		}
		if !validMobaShapes[entry.shape] {
			return fmt.Errorf("%s line %d: invalid shape %q", filename, line, entry.shape)
		}

		numeric := []struct {
			column string
			idx    int
			dst    *float64
		}{
			{"radius", col("radius"), &entry.radius},
			{"length", col("length"), &entry.length},
			{"width", col("width"), &entry.width},
			{"angle_deg", col("angle_deg"), &entry.angle},
			{"cast_range", col("cast_range"), &entry.castRange},
			{"projectile_speed", col("projectile_speed"), &entry.projSpeed},
			{"move_distance", col("move_distance"), &entry.moveDistance},
		}
		for _, field := range numeric {
			value, parseErr := parseMobaFloat(get(field.idx), *field.dst)
			if parseErr != nil || value < 0 {
				return fmt.Errorf("%s line %d: invalid %s %q", filename, line, field.column, get(field.idx))
			}
			*field.dst = value
		}

		integers := []struct {
			column string
			idx    int
			dst    *int
		}{
			{"cast_time_ms", col("cast_time_ms"), &entry.castTimeMs},
			{"recovery_ms", col("recovery_ms"), &entry.recoveryMs},
			{"cooldown_ms", col("cooldown_ms"), &entry.cooldownMs},
			{"max_targets", col("max_targets"), &entry.maxTargets},
		}
		for _, field := range integers {
			value, parseErr := parseMobaInt(get(field.idx), *field.dst)
			if parseErr != nil || value < 0 {
				return fmt.Errorf("%s line %d: invalid %s %q", filename, line, field.column, get(field.idx))
			}
			*field.dst = value
		}
		if entry.maxTargets < 1 {
			entry.maxTargets = 1
		}
		entry.pierce = strings.EqualFold(get(col("pierce")), "true") || get(col("pierce")) == "1"

		key := id + "|" + rank
		if _, exists := seen[key]; !exists {
			order = append(order, key)
		}
		seen[key] = entry
	}

	for _, key := range order {
		entry := seen[key]
		if _, err := activeCSVImportDB().Exec(`
			INSERT INTO gu_moba_profiles (gu_id, rank, cast_mode, shape, radius, length, width, angle_deg,
				cast_range, cast_time_ms, recovery_ms, cooldown_ms, projectile_speed, pierce, max_targets, move_distance)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			ON CONFLICT (gu_id, rank) DO UPDATE
			SET cast_mode=EXCLUDED.cast_mode, shape=EXCLUDED.shape, radius=EXCLUDED.radius,
			    length=EXCLUDED.length, width=EXCLUDED.width, angle_deg=EXCLUDED.angle_deg,
			    cast_range=EXCLUDED.cast_range, cast_time_ms=EXCLUDED.cast_time_ms,
			    recovery_ms=EXCLUDED.recovery_ms, cooldown_ms=EXCLUDED.cooldown_ms,
			    projectile_speed=EXCLUDED.projectile_speed, pierce=EXCLUDED.pierce,
			    max_targets=EXCLUDED.max_targets, move_distance=EXCLUDED.move_distance
		`, entry.id, entry.rank, entry.castMode, entry.shape, entry.radius, entry.length, entry.width,
			entry.angle, entry.castRange, entry.castTimeMs, entry.recoveryMs, entry.cooldownMs,
			entry.projSpeed, entry.pierce, entry.maxTargets, entry.moveDistance); err != nil {
			return fmt.Errorf("%s line %d: upsert gu_moba_profile %s rank %s: %w", filename, entry.line, entry.id, entry.rank, err)
		}
	}

	logger.S.Infof("MOBA profiles seeded from %s (%d entries)", filename, len(order))
	return nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func parseMobaFloat(raw string, fallback float64) (float64, error) {
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(raw, 64)
}

func parseMobaInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}
