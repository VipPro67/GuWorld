package handlers

// WORLD-scope resource points are the canon public mines of the Gu World: they
// sit on the shared map, produce at plain real-time pace and can only be
// harvested while the player stands on them (see aperture_points.go). This file
// owns their seeding lifecycle:
//
//   - ensureWorldResourcePoints tops one region up to its caps and is safe to
//     call concurrently (transaction-scoped advisory lock + partial unique
//     index on (region_id, x, y) WHERE scope = 'WORLD').
//   - EnsureWorldResourcePointsSweep walks every active region and is wired to
//     a periodic worker, so m discarded/transplanted mountains grow back even
//     when nobody opens the world-point panel.
//   - The listing endpoint still calls the ensure helper once, so a brand new
//     region works before the worker's first tick.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	// worldNormalPointsPerRegion caps plain public mines so the world keeps the
	// scarcity that makes guarded/famous mountains matter.
	worldNormalPointsPerRegion = 2
	// worldFamousMinRegionRank gates the canon famous mountain (Núi Danh) to
	// regions whose strongest monster reaches rank 4.
	worldFamousMinRegionRank = 4
	// worldPointCellAttempts bounds the random placement retries per point.
	worldPointCellAttempts = 48
	// worldPointSweepInterval is the periodic top-up cadence.
	worldPointSweepInterval = 10 * time.Minute
)

// worldRegionSeed is the geometry/rank snapshot used to place one region's
// public mines.
type worldRegionSeed struct {
	ID         string
	MinX, MinY int
	MaxX, MaxY int
	MaxRank    int
	ZoneType   string
}

func loadWorldRegionSeed(tx *sql.Tx, regionID string) (worldRegionSeed, error) {
	var seed worldRegionSeed
	var width, height int
	err := tx.QueryRow(`
		SELECT id, COALESCE(width, 0), COALESCE(height, 0),
		       COALESCE(min_x, 0), COALESCE(min_y, 0), COALESCE(max_x, 0), COALESCE(max_y, 0),
		       COALESCE(max_rank, 1), COALESCE(zone_type, '')
		FROM regions WHERE id = $1
	`, regionID).Scan(&seed.ID, &width, &height, &seed.MinX, &seed.MinY, &seed.MaxX, &seed.MaxY, &seed.MaxRank, &seed.ZoneType)
	if err != nil {
		return worldRegionSeed{}, err
	}
	// Some legacy rows still carry zeroed bounds; fall back to width×height.
	if seed.MaxX <= seed.MinX {
		seed.MinX, seed.MaxX = 0, maxInt(width-1, 0)
	}
	if seed.MaxY <= seed.MinY {
		seed.MinY, seed.MaxY = 0, maxInt(height-1, 0)
	}
	return seed, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func worldRegionRNG(regionID string) *rand.Rand {
	h := fnv.New64a()
	_, _ = h.Write([]byte(regionID))
	return rand.New(rand.NewSource(int64(h.Sum64()) ^ time.Now().UnixNano()))
}

// worldPointSeedingZone limits public mines to wilderness maps. Parent regions
// (NGU_VUC), secret realms and the tribulation arena either are not walkable at
// these coordinates or are private instances.
func worldPointSeedingZone(zoneType string) bool {
	switch strings.ToUpper(strings.TrimSpace(zoneType)) {
	case "KHU_VUC", "BOSS_ZONE", "PVP_ZONE":
		return true
	default:
		return false
	}
}

// worldPointGradeForRank keeps the point grade anchored to the region's
// strongest content, mirroring the original lazy seeder.
func worldPointGradeForRank(rank int) string {
	switch {
	case rank >= 7:
		return "TOP"
	case rank >= 4:
		return "MID"
	default:
		return "LOW"
	}
}

// pickWorldPointCell finds a free, non-safe coordinate inside the region. The
// world harvest check uses Chebyshev distance on these same coordinates, so the
// point is reachable from the region's walkable area.
func pickWorldPointCell(tx *sql.Tx, region worldRegionSeed, rng *rand.Rand) (int, int, bool) {
	loX, hiX := maxInt(region.MinX+1, 0), region.MaxX-1
	loY, hiY := maxInt(region.MinY+1, 0), region.MaxY-1
	if hiX < loX || hiY < loY {
		return 0, 0, false
	}
	for attempt := 0; attempt < worldPointCellAttempts; attempt++ {
		x := loX + rng.Intn(hiX-loX+1)
		y := loY + rng.Intn(hiY-loY+1)
		if IsSafeZone(region.ID, x, y) {
			continue
		}
		var occupied bool
		if err := tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM aperture_resource_points
				WHERE scope = 'WORLD' AND region_id = $1 AND x = $2 AND y = $3
			)
		`, region.ID, x, y).Scan(&occupied); err != nil {
			return 0, 0, false
		}
		if occupied {
			continue
		}
		return x, y, true
	}
	return 0, 0, false
}

// ensureWorldResourcePoints tops one region up to its public-mine caps. It is
// idempotent and concurrency-safe: callers may run it inside any transaction
// (the advisory lock is released on commit/rollback) and duplicate placements
// are rejected by idx_aperture_world_points_region_cell.
func ensureWorldResourcePoints(ctx context.Context, tx *sql.Tx, regionID string) (addedNormal, addedFamous int, err error) {
	if tx == nil || regionID == "" {
		return 0, 0, nil
	}
	// Try-lock: when another request/sweep already owns this region we skip the
	// top-up instead of serializing every reader of the region behind it.
	var locked bool
	if err = tx.QueryRowContext(ctx, `
		SELECT pg_try_advisory_xact_lock(hashtext('world_resource_points:' || $1))
	`, regionID).Scan(&locked); err != nil {
		return 0, 0, fmt.Errorf("lock world resource points: %w", err)
	}
	if !locked {
		return 0, 0, nil
	}

	region, err := loadWorldRegionSeed(tx, regionID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	if !worldPointSeedingZone(region.ZoneType) {
		return 0, 0, nil
	}

	rank := region.MaxRank
	var monsterRank sql.NullInt64
	if err := tx.QueryRowContext(ctx, `
		SELECT MAX((metadata->>'rank')::int)
		FROM map_cells WHERE region_id = $1 AND cell_type = 'Monster' AND is_active = true
	`, regionID).Scan(&monsterRank); err == nil && monsterRank.Valid && int(monsterRank.Int64) > rank {
		rank = int(monsterRank.Int64)
	}
	grade := worldPointGradeForRank(rank)

	var normalCount, famousCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE COALESCE(point_class, 'NORMAL') <> 'FAMOUS'),
		       COUNT(*) FILTER (WHERE COALESCE(point_class, 'NORMAL') = 'FAMOUS')
		FROM aperture_resource_points WHERE scope = 'WORLD' AND region_id = $1
	`, regionID).Scan(&normalCount, &famousCount); err != nil {
		return 0, 0, err
	}

	rng := worldRegionRNG(regionID)

	// Anchor the canon famous mountain before the plain mines so random normal
	// placement can never occupy the strongest Boss cell and starve it.
	if rank >= worldFamousMinRegionRank && famousCount == 0 {
		if bossX, bossY, bossRank, ok := strongestBossCell(tx, regionID); ok {
			var occupied bool
			if err := tx.QueryRowContext(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM aperture_resource_points
					WHERE scope = 'WORLD' AND region_id = $1 AND x = $2 AND y = $3
				)
			`, regionID, bossX, bossY).Scan(&occupied); err == nil && !occupied {
				pick := weightedPointPick(rng, canonWorldTemplatePool(true), "MID")
				if pick.PointType != "" {
					res, execErr := tx.ExecContext(ctx, `
						INSERT INTO aperture_resource_points
							(scope, region_id, point_type, name, path, grade, point_class, level, vitality, stock, stock_cap, x, y, last_produced_at, state)
						VALUES ('WORLD', $1, $2, $3, $4, $5, 'FAMOUS', 1, 100, '{}'::jsonb, 0, $6, $7, NOW(), 'ACTIVE')
						ON CONFLICT DO NOTHING
					`, regionID, pick.PointType, pick.Name, pick.Path, worldPointGradeForRank(bossRank), bossX, bossY)
					if execErr != nil {
						return addedNormal, addedFamous, execErr
					}
					if n, rowsErr := res.RowsAffected(); rowsErr == nil && n > 0 {
						addedFamous++
					}
				}
			}
		}
	}
	pool := canonWorldTemplatePool(false)
	for attempts := 0; attempts < worldNormalPointsPerRegion*4 && normalCount < worldNormalPointsPerRegion; attempts++ {
		pick := weightedPointPick(rng, pool, grade)
		if pick.PointType == "" {
			break
		}
		x, y, ok := pickWorldPointCell(tx, region, rng)
		if !ok {
			break
		}
		res, execErr := tx.ExecContext(ctx, `
			INSERT INTO aperture_resource_points
				(scope, region_id, point_type, name, path, grade, point_class, level, vitality, stock, stock_cap, x, y, last_produced_at, state)
			VALUES ('WORLD', $1, $2, $3, $4, $5, 'NORMAL', 1, 100, '{}'::jsonb, 0, $6, $7, NOW(), 'ACTIVE')
			ON CONFLICT DO NOTHING
		`, regionID, pick.PointType, pick.Name, pick.Path, grade, x, y)
		if execErr != nil {
			return addedNormal, addedFamous, execErr
		}
		if n, rowsErr := res.RowsAffected(); rowsErr == nil && n == 0 {
			continue
		}
		normalCount++
		addedNormal++
	}

	return addedNormal, addedFamous, nil
}

// EnsureWorldResourcePointsSweep tops up every active wilderness region that is
// below its caps. The aggregate query keeps idle ticks to a single read once
// the world is full.
func EnsureWorldResourcePointsSweep(ctx context.Context) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT r.id
		FROM regions r
		LEFT JOIN aperture_resource_points p
			ON p.region_id = r.id AND p.scope = 'WORLD'
		WHERE r.is_active = true
		  AND UPPER(COALESCE(r.zone_type, '')) IN ('KHU_VUC', 'BOSS_ZONE', 'PVP_ZONE')
		GROUP BY r.id, r.max_rank
		HAVING COUNT(*) FILTER (WHERE COALESCE(p.point_class, 'NORMAL') <> 'FAMOUS') < $1
		    OR (
		        COUNT(*) FILTER (WHERE COALESCE(p.point_class, 'NORMAL') = 'FAMOUS') = 0
		        AND GREATEST(
		            COALESCE(r.max_rank, 1),
		            COALESCE((
		                SELECT MAX((mc.metadata->>'rank')::int) FROM map_cells mc
		                WHERE mc.region_id = r.id AND mc.cell_type = 'Monster' AND mc.is_active = true
		            ), 1)
		        ) >= $2
		        AND EXISTS (
		            SELECT 1 FROM map_cells b
		            WHERE b.region_id = r.id AND LOWER(b.cell_type) = 'boss' AND b.is_active = true
		        )
		    )
		ORDER BY r.id
	`, worldNormalPointsPerRegion, worldFamousMinRegionRank)
	if err != nil {
		logger.S.Errorf("World resource points sweep: query regions: %v", err)
		return
	}
	regionIDs := []string{}
	for rows.Next() {
		var regionID string
		if err := rows.Scan(&regionID); err == nil {
			regionIDs = append(regionIDs, regionID)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		logger.S.Errorf("World resource points sweep: read regions: %v", err)
		return
	}

	totalNormal, totalFamous := 0, 0
	for _, regionID := range regionIDs {
		if ctx.Err() != nil {
			return
		}
		addedNormal, addedFamous, err := sweepOneWorldRegion(ctx, regionID)
		if err != nil {
			logger.S.Errorf("World resource points sweep: region %s: %v", regionID, err)
			continue
		}
		totalNormal += addedNormal
		totalFamous += addedFamous
	}
	if totalNormal > 0 || totalFamous > 0 {
		logger.S.Infof("World resource points sweep: seeded %d normal + %d famous mines across %d deficient regions", totalNormal, totalFamous, len(regionIDs))
	}
}

func sweepOneWorldRegion(ctx context.Context, regionID string) (int, int, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()
	addedNormal, addedFamous, err := ensureWorldResourcePoints(ctx, tx, regionID)
	if err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return addedNormal, addedFamous, nil
}

// worldPointGuardStatus reports whether a live world actor (boss or regular
// monster) currently stands on the point's cell. The realtime world manager
// owns materialization, so the Redis actor cache is the authority: a mine is
// guarded exactly while something alive occupies its cell. Redis being
// unavailable fails open — the harvest still requires proximity, and a live
// boss usually intercepts the player in world combat anyway.
func worldPointGuardStatus(ctx context.Context, regionID string, x, y int) bool {
	bosses, err := cache.WorldBossesInRegion(ctx, regionID)
	if err == nil {
		for _, boss := range bosses {
			if strings.EqualFold(boss.State, MonsterStateDefeated) {
				continue
			}
			if math.Abs(boss.X-float64(x)) <= 1 && math.Abs(boss.Y-float64(y)) <= 1 {
				return true
			}
		}
	}
	bx, by := monsterBucket(x, y)
	monsters, err := cache.WorldMonstersInBucket(ctx, regionID, bx, by)
	if err != nil {
		return false
	}
	for _, monster := range monsters {
		if strings.EqualFold(monster.State, MonsterStateDefeated) {
			continue
		}
		if math.Abs(monster.X-float64(x)) <= 1 && math.Abs(monster.Y-float64(y)) <= 1 {
			return true
		}
	}
	return false
}

// StartWorldResourcePointWorker fills the world once at boot (so mines exist
// before the first player explores) and then tops regions up periodically.
func StartWorldResourcePointWorker(ctx context.Context) {
	go worker.SafeRun("World Resource Points boot sweep", ctx, EnsureWorldResourcePointsSweep)
	worker.RunEvery(ctx, worldPointSweepInterval, "World Resource Points", EnsureWorldResourcePointsSweep)
}
