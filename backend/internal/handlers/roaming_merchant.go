package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

// Roaming merchant tuning.
const (
	roamingMerchantTarget      = 20
	roamingMerchantStockMin    = 8
	roamingMerchantStockMax    = 15
	roamingMerchantMinLifetime = 10 * time.Minute
	roamingMerchantMaxLifetime = 30 * time.Minute
	roamingMerchantSellMarkup  = 1.40 // roaming merchant sells at base × this

	// roamingPlayerZoneChance is how often a new trader is placed in a wilderness
	// zone that has recently active players rather than anywhere on the map. The
	// rest of the spawns still go anywhere, so exploring the wild keeps paying off.
	roamingPlayerZoneChance = 0.75
	// roamingPresenceWindow is how recently a player must have been active to make
	// their zone count as "being played". last_activity is batch-flushed every 2
	// minutes, so this window is deliberately generous.
	roamingPresenceWindow = 15 * time.Minute
)

// StartRoamingMerchantWorker keeps a handful of roaming traders alive on the
// world map. Each tick it expires old traders and spawns new ones (with a fresh
// random stock), favouring wilderness zones where players are currently playing
// and announcing each spawn in that region's chat channel.
func StartRoamingMerchantWorker(ctx context.Context, database *sql.DB, interval time.Duration) {
	tick := func(ctx context.Context) { roamingMerchantTick(ctx, database) }
	go worker.SafeRun("Roaming Merchant", ctx, tick)
	worker.RunEvery(ctx, interval, "Roaming Merchant", tick)
}

func roamingMerchantTick(ctx context.Context, database *sql.DB) {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	// Only one node should spawn/expire at a time.
	var locked bool
	if err := tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock(hashtext('roaming_merchant_worker'))`).Scan(&locked); err != nil || !locked {
		return
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM merchants WHERE profile_id = 'roaming_trader' AND expires_at IS NOT NULL AND expires_at <= NOW()`); err != nil {
		logger.S.Warnf("roaming merchant: expire failed: %v", err)
		return
	}

	// Load the spawn map once per tick: every wilderness zone plus the live
	// signals (players playing there, traders already there) used to bias picks.
	candidates, err := loadRoamingRegionCandidates(ctx, tx, roamingPresenceWindow)
	if err != nil {
		logger.S.Warnf("roaming merchant: load candidate zones failed: %v", err)
		return
	}
	if len(candidates) == 0 {
		logger.S.Debugf("roaming merchant: no eligible zone to spawn in")
		return
	}

	var notices []roamingSpawnNotice

	var active int
	_ = tx.QueryRowContext(ctx, `SELECT count(*) FROM merchants m JOIN merchant_profiles p ON p.id=m.profile_id WHERE p.kind='ROAMING' AND m.is_active AND (m.expires_at IS NULL OR m.expires_at > NOW())`).Scan(&active)
	for i := active; i < roamingMerchantTarget; i++ {
		idx := pickRoamingRegion(candidates, rand.Float64)
		if idx < 0 {
			break
		}
		notice, err := spawnRoamingMerchant(ctx, tx, candidates[idx])
		if err != nil {
			logger.S.Warnf("roaming merchant: spawn failed: %v", err)
			break
		}
		// A trader spawned a moment ago already occupies that zone: count it so the
		// rest of this batch spreads out instead of stacking on one hotspot.
		candidates[idx].Traders++
		notices = append(notices, notice)
	}
	if err := tx.Commit(); err != nil {
		logger.S.Warnf("roaming merchant: commit failed: %v", err)
		return
	}
	// New/expired traders change every region's /merchant list; drop the cache.
	invalidateMerchantRegionCache("")
	// Announce in the spawned region's chat channel (not world) after commit so a
	// rolled-back spawn never leaks coordinates.
	for _, n := range notices {
		_ = SendSystemMessage("region", sql.NullString{String: n.regionID, Valid: true}, fmt.Sprintf(
			"【Thương Nhân Lang Thang】Một thương nhân bí ẩn vừa xuất hiện tại %s (%d, %d)! Hàng hiếm giá cao, nhanh chân kẻo lỡ.",
			n.regionName, n.x, n.y))
	}
}

// roamingRegionCandidate is one wilderness zone a trader may spawn in, with the
// live signals used to bias the choice.
type roamingRegionCandidate struct {
	ID      string
	Name    string
	MinX    int
	MinY    int
	MaxX    int
	MaxY    int
	Width   int
	Height  int
	Players int
	Traders int
}

// loadRoamingRegionCandidates lists every zone a roaming trader may spawn in
// (wilderness only — HUB/SAFE_ZONE already have the fixed village merchant)
// together with how many players are playing there and how many traders it hosts.
func loadRoamingRegionCandidates(ctx context.Context, tx *sql.Tx, presenceWindow time.Duration) ([]roamingRegionCandidate, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT r.id, COALESCE(r.name,''),
		       COALESCE(r.min_x,0), COALESCE(r.min_y,0), COALESCE(r.max_x,0), COALESCE(r.max_y,0),
		       COALESCE(r.width,100), COALESCE(r.height,100),
		       COALESCE(pv.n,0), COALESCE(tr.n,0)
		FROM regions r
		LEFT JOIN (
			SELECT current_region AS rid, count(*) AS n FROM players
			WHERE current_region <> '' AND last_activity > NOW() - make_interval(secs => $1)
			GROUP BY 1
		) pv ON pv.rid = r.id
		LEFT JOIN (
			SELECT m.region_id AS rid, count(*) AS n
			FROM merchants m
			JOIN merchant_profiles mp ON mp.id = m.profile_id
			WHERE mp.kind = 'ROAMING' AND m.is_active AND (m.expires_at IS NULL OR m.expires_at > NOW())
			GROUP BY 1
		) tr ON tr.rid = r.id
		WHERE r.is_active AND r.zone_type NOT IN ('HUB','SAFE_ZONE','SECRET_REALM')
	`, int(presenceWindow.Seconds()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []roamingRegionCandidate
	for rows.Next() {
		var z roamingRegionCandidate
		if err := rows.Scan(&z.ID, &z.Name, &z.MinX, &z.MinY, &z.MaxX, &z.MaxY, &z.Width, &z.Height, &z.Players, &z.Traders); err == nil {
			zones = append(zones, z)
		}
	}
	return zones, rows.Err()
}

// roamingZoneWeight damps zones that already hold traders so a batch of spawns
// spreads across a hotspot instead of stacking on one zone. With playerBonus set
// (inside the player-playing tier) zones with more players weigh more.
func roamingZoneWeight(z roamingRegionCandidate, playerBonus bool) float64 {
	w := 1 / float64(z.Traders+1)
	if playerBonus {
		w *= float64(z.Players + 1)
	}
	return w
}

// pickRoamingRegion chooses a spawn zone: with roamingPlayerZoneChance it draws
// only among zones where players are currently playing (falling through to the
// wilderness when nobody is playing), otherwise among all wilderness zones.
// Returns -1 when no zone matches. roll is injected (pass rand.Float64) so the
// choice stays testable.
func pickRoamingRegion(zones []roamingRegionCandidate, roll func() float64) int {
	if len(zones) == 0 {
		return -1
	}
	if roll() < roamingPlayerZoneChance {
		if idx := pickWeightedZone(zones, roll, true); idx >= 0 {
			return idx
		}
	}
	return pickWeightedZone(zones, roll, false)
}

// pickWeightedZone draws a zone index by roamingZoneWeight; playerOnly restricts
// the draw to zones with recently active players. The roll is only consumed when
// at least one zone is eligible, so an empty player tier costs no randomness.
func pickWeightedZone(zones []roamingRegionCandidate, roll func() float64, playerOnly bool) int {
	total := 0.0
	for _, z := range zones {
		if playerOnly && z.Players == 0 {
			continue
		}
		total += roamingZoneWeight(z, playerOnly)
	}
	if total <= 0 {
		return -1
	}
	target := roll() * total
	acc := 0.0
	for i, z := range zones {
		if playerOnly && z.Players == 0 {
			continue
		}
		acc += roamingZoneWeight(z, playerOnly)
		if target < acc {
			return i
		}
	}
	// Floating point fallback: hand out the last eligible zone.
	for i := len(zones) - 1; i >= 0; i-- {
		if !playerOnly || zones[i].Players > 0 {
			return i
		}
	}
	return -1
}

type roamingSpawnNotice struct {
	regionID   string
	regionName string
	x, y       int
}

func spawnRoamingMerchant(ctx context.Context, tx *sql.Tx, zone roamingRegionCandidate) (roamingSpawnNotice, error) {
	regionID, regionName := zone.ID, zone.Name
	x := randomCoord(zone.MinX, zone.MaxX, zone.Width)
	y := randomCoord(zone.MinY, zone.MaxY, zone.Height)

	merchantID := "roam_" + uuid.New().String()
	lifetime := roamingMerchantMinLifetime + time.Duration(rand.Int63n(int64(roamingMerchantMaxLifetime-roamingMerchantMinLifetime)))

	// Per-instance buy pool: this trader only wants a random handful of item ids,
	// at its own price range and quota.
	buyIDs, err := randomBuyPool(ctx, tx)
	if err != nil {
		return roamingSpawnNotice{}, err
	}
	buyTagsJSON, _ := json.Marshal(merchantBuyTags{ItemIDs: buyIDs})
	startRatio := 0.70 + rand.Float64()*0.30 // 0.70 → 1.00
	floorRatio := 0.10 + rand.Float64()*0.20 // 0.10 → 0.30
	if floorRatio > startRatio {
		floorRatio = startRatio
	}
	quota := 5 + rand.Intn(16) // 5 → 20

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO merchants (id, profile_id, region_id, x, y, is_active, expires_at, buy_tags, buy_ratio, buy_floor_ratio, quota_per_item)
		VALUES ($1, 'roaming_trader', $2, $3, $4, TRUE, NOW() + make_interval(secs => $5), $6::jsonb, $7, $8, $9)`,
		merchantID, regionID, x, y, int(lifetime.Seconds()), string(buyTagsJSON), startRatio, floorRatio, quota); err != nil {
		return roamingSpawnNotice{}, err
	}

	stock := roamingMerchantStockMin + rand.Intn(roamingMerchantStockMax-roamingMerchantStockMin+1)
	// Random stock from the rank<6 / rarity<6 Gu pool + their food.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO merchant_stock (merchant_id, item_id, rank, price, stock, sort_order)
		SELECT $1, p.id, p.rank,
		       GREATEST(0.001, COALESCE(NULLIF(p.base_price,0),1) * $3),
		       (1 + floor(random()*5))::int, 0
		FROM (
			SELECT g.id, g.rank, g.base_price FROM gu_dictionary g
			WHERE g.rank < 6 AND COALESCE(g.rarity,0) < 6
			UNION
			SELECT i.id, COALESCE(i.rank,1), i.base_price
			FROM gu_dictionary g
			JOIN LATERAL jsonb_array_elements_text(g.food_types) f(fid) ON TRUE
			JOIN item_dictionary i ON i.id = f.fid
			WHERE g.rank < 6 AND COALESCE(g.rarity,0) < 6
		) p
		ORDER BY random()
		LIMIT $2`,
		merchantID, stock, roamingMerchantSellMarkup)
	if err != nil {
		return roamingSpawnNotice{}, err
	}
	if err := insertRandomRecipeStock(ctx, tx, merchantID); err != nil {
		return roamingSpawnNotice{}, err
	}
	return roamingSpawnNotice{regionID: regionID, regionName: regionName, x: x, y: y}, nil
}

// insertRandomRecipeStock adds a few random refining recipes to a roaming
// merchant's sell stock, priced from the target Gu rank.
func insertRandomRecipeStock(ctx context.Context, tx *sql.Tx, merchantID string) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) AS rid, target_rank
		FROM refining_recipes
		ORDER BY random()
		LIMIT 3`)
	if err != nil {
		return err
	}
	type recipe struct {
		id   string
		rank int
	}
	var picked []recipe
	for rows.Next() {
		var r recipe
		if err := rows.Scan(&r.id, &r.rank); err == nil {
			picked = append(picked, r)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range picked {
		if r.rank <= 0 {
			r.rank = 1
		}
		price := pricing.Quantize(pricing.GuBasePriceByRank(r.rank) * roamingMerchantSellMarkup)
		if price < 0.001 {
			price = 0.001
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO merchant_stock (merchant_id, item_id, rank, price, stock, sort_order)
			VALUES ($1, $2, $3, $4, (1 + floor(random()*3))::int, 0)
			ON CONFLICT (merchant_id, item_id, rank) DO NOTHING`,
			merchantID, r.id, r.rank, price); err != nil {
			return err
		}
	}
	return nil
}

// randomBuyPool picks a random subset of entity ids (Gu rank<6 / rarity<6 +
// their food) for a roaming trader's personal buy list.
func randomBuyPool(ctx context.Context, tx *sql.Tx) ([]string, error) {
	count := 8 + rand.Intn(8) // 8 → 15
	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM (
			SELECT g.id FROM gu_dictionary g
			WHERE g.rank < 6 AND COALESCE(g.rarity,0) < 6
			UNION
			SELECT i.id FROM gu_dictionary g
			JOIN LATERAL jsonb_array_elements_text(g.food_types) f(fid) ON TRUE
			JOIN item_dictionary i ON i.id = f.fid
			WHERE g.rank < 6 AND COALESCE(g.rarity,0) < 6
			UNION
			SELECT ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text)
			FROM refining_recipes
		) p
		ORDER BY random()
		LIMIT $1`, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// randomCoord picks an inclusive coordinate from [min,max] when the region has a
// real bounding box, otherwise from [0,size).
func randomCoord(min, max, size int) int {
	if max > min {
		return min + rand.Intn(max-min+1)
	}
	if size <= 0 {
		size = 1
	}
	return rand.Intn(size)
}
