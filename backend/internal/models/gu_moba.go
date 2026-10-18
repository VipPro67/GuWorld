package models

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// MobaCastMode is how a Gu is aimed in the realtime (MOBA-style) PvP mode.
type MobaCastMode string

const (
	MobaCastTarget    MobaCastMode = "TARGET"
	MobaCastDirection MobaCastMode = "DIRECTION"
	MobaCastGround    MobaCastMode = "GROUND"
	MobaCastSelf      MobaCastMode = "SELF"
)

// MobaShape is the hit area a cast resolves against once the cast completes.
type MobaShape string

const (
	MobaShapeSingle MobaShape = "SINGLE"
	MobaShapeLine   MobaShape = "LINE"
	MobaShapeCircle MobaShape = "CIRCLE"
	MobaShapeCone   MobaShape = "CONE"
)

// MobaProfile is the realtime skill card for one (gu_id, rank).
//
// It only describes aiming, shape and realtime timings. The damage, status and
// shield payload still comes from GuDictionaryItem.Effects, so the two data
// sources never duplicate each other.
type MobaProfile struct {
	GuID            string
	Rank            int
	CastMode        MobaCastMode
	Shape           MobaShape
	Radius          float64
	Length          float64
	Width           float64
	AngleDeg        float64
	CastRange       float64
	CastTimeMs      int
	RecoveryMs      int
	CooldownMs      int
	ProjectileSpeed float64
	Pierce          bool
	MaxTargets      int
	MoveDistance    float64
}

var (
	mobaProfilesMu sync.RWMutex
	mobaProfiles   = map[string]map[int]MobaProfile{}
)

// DefaultMobaProfile keeps every Gu castable in the realtime mode even when it
// was never authored in GuMoba.csv: single target, short range, no cooldown.
func DefaultMobaProfile(guID string, rank int) MobaProfile {
	return MobaProfile{
		GuID: guID, Rank: rank,
		CastMode: MobaCastTarget, Shape: MobaShapeSingle,
		CastRange: 8, MaxTargets: 1,
	}
}

// GetMobaProfile returns the authored realtime card for a Gu, if any.
func GetMobaProfile(guID string, rank int) (MobaProfile, bool) {
	mobaProfilesMu.RLock()
	defer mobaProfilesMu.RUnlock()
	byRank, ok := mobaProfiles[guID]
	if !ok {
		return MobaProfile{}, false
	}
	profile, ok := byRank[rank]
	return profile, ok
}

// ResolveMobaProfile returns the authored card or the default fallback.
func ResolveMobaProfile(guID string, rank int) MobaProfile {
	if profile, ok := GetMobaProfile(guID, rank); ok {
		return profile
	}
	return DefaultMobaProfile(guID, rank)
}

// LoadMobaProfiles refreshes the in-memory realtime skill cards. A missing table
// is tolerated (the mode falls back to defaults) so an older database does not
// block startup.
func LoadMobaProfiles(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT gu_id, rank, cast_mode, shape, radius, length, width, angle_deg,
		       cast_range, cast_time_ms, recovery_ms, cooldown_ms, projectile_speed,
		       pierce, max_targets, move_distance
		FROM gu_moba_profiles
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	next := make(map[string]map[int]MobaProfile)
	count := 0
	for rows.Next() {
		var profile MobaProfile
		if err := rows.Scan(
			&profile.GuID, &profile.Rank, &profile.CastMode, &profile.Shape,
			&profile.Radius, &profile.Length, &profile.Width, &profile.AngleDeg,
			&profile.CastRange, &profile.CastTimeMs, &profile.RecoveryMs, &profile.CooldownMs,
			&profile.ProjectileSpeed, &profile.Pierce, &profile.MaxTargets, &profile.MoveDistance,
		); err != nil {
			return fmt.Errorf("scan gu_moba_profile row: %w", err)
		}
		if next[profile.GuID] == nil {
			next[profile.GuID] = make(map[int]MobaProfile)
		}
		next[profile.GuID][profile.Rank] = profile
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate gu_moba_profiles: %w", err)
	}

	mobaProfilesMu.Lock()
	mobaProfiles = next
	mobaProfilesMu.Unlock()
	logger.S.Infof("Loaded %d realtime MOBA Gu profiles", count)
	return nil
}
