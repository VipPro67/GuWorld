package handlers

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const clearedCellCooldown = 15 * time.Minute

const (
	monsterCampSpacing = 48
	monsterCampRadius  = 6
)

type proceduralMonsterCamp struct {
	spawn    models.MonsterSpawn
	x, y     int
	distance int
}

func GetCellSeed(x, y int, zoneKey string) int64 {
	h := sha256.New()
	h.Write([]byte(zoneKey))
	binary.Write(h, binary.BigEndian, int64(x))
	binary.Write(h, binary.BigEndian, int64(y))
	return int64(binary.BigEndian.Uint64(h.Sum(nil)[:8]))
}

// cellPoolEntry is one weighted content candidate for a procedural cell: an
// item id (Rank 0) or a specific Gu rank variant.
type cellPoolEntry struct {
	ID     string
	Rank   int
	Weight float64
}

const (
	cellPoolResource = 1
	cellPoolWildGu   = 2
	// Tiên Cổ (rank >= 6) never spawns as a map cell; it only comes from gacha,
	// refining and auctions (see docs/GAME_DATA.md).
	maxWildGuRank = 5
)

// cellPoolCache memoizes rank-band derived pools. Building one scans
// ItemRegistry/GuRegistry (750+ entries), too heavy to repeat per cell, while
// the result depends only on (kind, rank band) — at most ~2*81 entries for the
// whole world. Cleared whenever the zone/item/Gu registries reload.
var cellPoolCache = struct {
	mu sync.Mutex
	m  map[[3]int][]cellPoolEntry
}{m: make(map[[3]int][]cellPoolEntry)}

func clearCellPoolCache() {
	cellPoolCache.mu.Lock()
	cellPoolCache.m = make(map[[3]int][]cellPoolEntry)
	cellPoolCache.mu.Unlock()
}

// zoneRankBand returns the inclusive rank band a zone draws content from.
func zoneRankBand(zone *models.Region) (int, int) {
	lo, hi := zone.MinRank, zone.MaxRank
	if lo < 1 {
		lo = 1
	}
	if lo > 9 {
		lo = 9
	}
	if hi < lo {
		hi = lo
	}
	if hi > 9 {
		hi = 9
	}
	return lo, hi
}

// resourcePool returns the weighted item pool for a zone. An explicit
// pool_weights.resources list wins; otherwise the pool is derived from the item
// registry within the zone's rank band (Material/Food only), weighted by
// effectiveDropLevel(rank, rarity), so a rank-6 zone never drops rank-1 filler.
func resourcePool(zone *models.Region) []cellPoolEntry {
	if ids := zone.PoolWeights.Resources; len(ids) > 0 {
		pool := make([]cellPoolEntry, 0, len(ids))
		for _, id := range ids {
			pool = append(pool, cellPoolEntry{ID: id, Weight: itemDropWeight(id)})
		}
		return pool
	}
	lo, hi := zoneRankBand(zone)
	return derivedCellPool(cellPoolResource, lo, hi)
}

// wildGuPool returns the weighted (Gu id, rank) pool for a zone. An explicit
// pool_weights.wild_gus list wins (still filtered to mortal ranks); otherwise
// the pool is derived from the Gu registry within the zone's rank band.
func wildGuPool(zone *models.Region) []cellPoolEntry {
	if ids := zone.PoolWeights.WildGus; len(ids) > 0 {
		pool := make([]cellPoolEntry, 0, len(ids))
		for _, id := range ids {
			variants := guVariants(id)
			if len(variants) == 0 {
				if _, known := models.GuRegistry[id]; known {
					// Only immortal variants: correctly excluded from the map.
					continue
				}
				// Unloaded/unknown id in a hand-authored pool: keep a neutral
				// entry so the authored pool is not silently empty.
				rank := zone.MinRank
				if rank < 1 {
					rank = 1
				}
				pool = append(pool, cellPoolEntry{ID: id, Rank: rank, Weight: 1.0})
				continue
			}
			pool = append(pool, variants...)
		}
		return pool
	}
	lo, hi := zoneRankBand(zone)
	return derivedCellPool(cellPoolWildGu, lo, hi)
}

// itemDropWeight resolves an item id to its drop weight. An id missing from the
// registry still spawns with a neutral weight, keeping hand-authored pools that
// reference unknown ids non-empty (legacy behaviour).
func itemDropWeight(id string) float64 {
	spec, ok := models.ItemRegistry[id]
	if !ok {
		return 1.0
	}
	return guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
}

// guVariants expands a Gu id into its mortal rank variants, each weighted by
// effectiveDropLevel(rank, rarity). Immortal (rank >= 6) variants are dropped.
func guVariants(id string) []cellPoolEntry {
	if models.HasImmortalRank(id) {
		return nil
	}
	ranks := models.GuRegistry[id]
	if len(ranks) == 0 {
		return nil
	}
	keys := make([]int, 0, len(ranks))
	for r := range ranks {
		keys = append(keys, r)
	}
	sort.Ints(keys)
	out := make([]cellPoolEntry, 0, len(keys))
	for _, r := range keys {
		if r > maxWildGuRank {
			continue
		}
		spec := ranks[r]
		out = append(out, cellPoolEntry{ID: id, Rank: r, Weight: guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))})
	}
	return out
}

// derivedCellPool builds (and caches) the registry-derived pool for a rank band.
func derivedCellPool(kind, lo, hi int) []cellPoolEntry {
	key := [3]int{kind, lo, hi}
	cellPoolCache.mu.Lock()
	defer cellPoolCache.mu.Unlock()
	if pool, ok := cellPoolCache.m[key]; ok {
		return pool
	}
	pool := buildDerivedCellPool(kind, lo, hi)
	cellPoolCache.m[key] = pool
	return pool
}

func buildDerivedCellPool(kind, lo, hi int) []cellPoolEntry {
	pool := make([]cellPoolEntry, 0, 64)
	switch kind {
	case cellPoolResource:
		ids := make([]string, 0, len(models.ItemRegistry))
		for id, spec := range models.ItemRegistry {
			if !resourceItemType(spec.Type) || spec.Rank < lo || spec.Rank > hi {
				continue
			}
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			pool = append(pool, cellPoolEntry{ID: id, Weight: itemDropWeight(id)})
		}
	case cellPoolWildGu:
		ids := make([]string, 0, len(models.GuRegistry))
		for id := range models.GuRegistry {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			for _, entry := range guVariants(id) {
				if entry.Rank < lo || entry.Rank > hi {
					continue
				}
				pool = append(pool, entry)
			}
		}
	}
	return pool
}

func resourceItemType(t string) bool {
	return strings.EqualFold(t, "Material") || strings.EqualFold(t, "Food")
}

// pickPoolEntry draws one weighted entry. Returns false only for an empty or
// zero-weight pool.
func pickPoolEntry(pool []cellPoolEntry, rng *rand.Rand) (cellPoolEntry, bool) {
	total := 0.0
	for _, e := range pool {
		total += e.Weight
	}
	if total <= 0 {
		return cellPoolEntry{}, false
	}
	roll := rng.Float64() * total
	for _, e := range pool {
		roll -= e.Weight
		if roll <= 0 {
			return e, true
		}
	}
	return pool[len(pool)-1], true
}

// pickWeightedType rolls a cell content type using zone weights. Weights come
// from a map, so iterate keys in sorted order to keep the pick deterministic
// (Go map iteration order is random and would shuffle cells every call).
func pickWeightedType(weights map[string]int, rng *rand.Rand) string {
	keys := make([]string, 0, len(weights))
	for k := range weights {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	total := 0
	for _, k := range keys {
		if w := weights[k]; w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return "Empty"
	}
	roll := rng.Intn(total)
	for _, k := range keys {
		w := weights[k]
		if w <= 0 {
			continue
		}
		roll -= w
		if roll < 0 {
			return k
		}
	}
	return "Empty"
}

// GenerateCell procedurally generates a low-tier cell (monster / resource /
// wild Gu / empty) from the containing zone's in-memory pools. No database
// access — content is deterministic per (zone, x, y) and stays stable so the
// map matches gameplay. Cleared cells are suppressed via Redis shared map
// state with respawn TTL, allowing synchronized clearing across all players.
func GenerateCell(x, y int) models.Cell {
	if cell, ok := cellGenCache.get(x, y); ok {
		return cell
	}
	cell := generateCellUncached(x, y)
	cellGenCache.put(x, y, cell)
	return cell
}

func generateCellUncached(x, y int) models.Cell {
	if !inWorld(x, y) {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: "Biên Giới Cổ Giới (World Edge)"}
	}
	return generateCellForZone(x, y, FindZoneAt(x, y))
}

func generateCellInRegion(regionID string, x, y int) models.Cell {
	if !inWorld(x, y) {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: "Biên Giới Bí Cảnh"}
	}
	zone := ZoneByID(regionID)
	if zone == nil {
		return generateCellUncached(x, y)
	}
	return generateCellForZone(x, y, zone)
}

func generateCellForZone(x, y int, zone *models.Region) models.Cell {
	if zone == nil || zone.ZoneType == ZoneTypeHub {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
	}
	zoneID := zone.ID
	seed := GetCellSeed(x, y, zoneID)
	rng := rand.New(rand.NewSource(seed))
	camp, hasCamp := findProceduralMonsterCamp(x, y, zone)
	if hasCamp && rng.Float64() < monsterCampSpawnChance(camp.distance) {
		return models.Cell{X: x, Y: y, Type: "Monster", Content: camp.spawn.MonsterID,
			Rank: camp.spawn.MinRank, SubRank: rng.Intn(5) + 1}
	}

	weights := zone.PoolWeights.CellWeights
	if len(weights) == 0 {
		weights = map[string]int{"Monster": 2, "Resource": 4, "WildGu": 1, "Empty": 87}
	} else {
		// Override to forcibly reduce density if DB has high values
		newW := make(map[string]int)
		for k, v := range weights {
			if k == "Empty" {
				newW[k] = v * 4
			} else {
				newW[k] = v / 2
				if k == "Monster" {
					// Keep a few wanderers outside authored-looking camps, while
					// making most ordinary spawns read as a group on the map.
					newW[k] /= 3
					if newW[k] == 0 && v > 0 {
						newW[k] = 1
					}
				}
			}
		}
		if newW["Empty"] == 0 {
			newW["Empty"] = 100
		}
		weights = newW
	}

	switch pickWeightedType(weights, rng) {
	case "Monster":
		// A zone only spawns monsters listed in monster_spawns. Zones without a
		// pool row stay empty instead of falling back to a random template.
		spawn, ok := pickMonsterFromPool(zoneID, rng, false)
		if !ok {
			return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
		}
		rank := spawn.MinRank
		if rank < 1 {
			rank = zone.MinRank
		}
		if rank < 1 {
			rank = 1
		}
		return models.Cell{X: x, Y: y, Type: "Monster", Content: spawn.MonsterID, Rank: rank, SubRank: rng.Intn(5) + 1}
	case "Resource":
		entry, ok := pickPoolEntry(resourcePool(zone), rng)
		if !ok {
			return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
		}
		return models.Cell{X: x, Y: y, Type: "Resource", Content: entry.ID}
	case "WildGu":
		entry, ok := pickPoolEntry(wildGuPool(zone), rng)
		if !ok {
			return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
		}
		// The captured Gu is looked up by (id, rank), so the cell must carry the
		// exact rank variant that was drawn, not the zone's floor rank.
		rank := entry.Rank
		if rank < 1 {
			rank = zone.MinRank
		}
		if rank < 1 {
			rank = 1
		}
		return models.Cell{X: x, Y: y, Type: "WildGu", Content: entry.ID, Rank: rank}
	default:
		return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
	}
}

// findProceduralMonsterCamp returns the nearest stable camp affecting a cell.
// Camp centers, monster type, and rank derive from region and coarse-grid seeds,
// so the same terrain resolves identically for every player and backend restart.
func findProceduralMonsterCamp(x, y int, zone *models.Region) (proceduralMonsterCamp, bool) {
	if zone == nil || zone.ZoneType == ZoneTypeHub || zone.ZoneType == ZoneTypeSafe || IsSafeZone(zone.ID, x, y) {
		return proceduralMonsterCamp{}, false
	}
	gridX, gridY := x/monsterCampSpacing, y/monsterCampSpacing
	best := proceduralMonsterCamp{distance: monsterCampRadius + 1}
	found := false
	for gy := gridY - 1; gy <= gridY+1; gy++ {
		for gx := gridX - 1; gx <= gridX+1; gx++ {
			seed := GetCellSeed(gx, gy, zone.ID+":monster-camp")
			rng := rand.New(rand.NewSource(seed))
			cx := gx*monsterCampSpacing + 8 + rng.Intn(monsterCampSpacing-16)
			cy := gy*monsterCampSpacing + 8 + rng.Intn(monsterCampSpacing-16)
			if !inWorld(cx, cy) {
				continue
			}
			distance := int(math.Hypot(float64(x-cx), float64(y-cy)))
			if distance > monsterCampRadius || (found && distance >= best.distance) {
				continue
			}
			centerZone := FindZoneAt(cx, cy)
			if centerZone == nil || centerZone.ID != zone.ID {
				continue
			}
			spawn, ok := pickMonsterFromPool(zone.ID, rng, false)
			if !ok {
				// No monster_spawns row for this zone: no camp.
				continue
			}
			minRank, maxRank := spawn.MinRank, spawn.MaxRank
			if minRank < 1 {
				minRank = zone.MinRank
			}
			if minRank < 1 {
				minRank = 1
			}
			if maxRank < minRank {
				maxRank = minRank
			}
			rank := minRank
			if maxRank > minRank {
				rank += rng.Intn(maxRank - minRank + 1)
			}
			best = proceduralMonsterCamp{spawn: models.MonsterSpawn{MonsterID: spawn.MonsterID, MinRank: rank}, x: cx, y: cy, distance: distance}
			found = true
		}
	}
	return best, found
}

func monsterCampSpawnChance(distance int) float64 {
	switch {
	case distance <= 2:
		return 0.8
	case distance <= 4:
		return 0.55
	default:
		return 0.3
	}
}

// ResolveCell resolves a world cell. Shared persistent cells (bosses, high
// resources, safe points) come from the in-memory map_cells index; everything
// else is generated procedurally. No database access in the hot path.
func ResolveCell(x, y int) models.Cell {
	return ResolveCellInRegion(DeriveRegionAt(x, y), x, y)
}

// ResolveCellInRegion resolves shared resources and portals within a logical
// region. Identical coordinates on separate layers/realms can hold distinct
// persistent cells.
func ResolveCellInRegion(regionID string, x, y int) models.Cell {
	if pc, found, cleared := PersistentCellAt(regionID, x, y); found {
		if cleared {
			return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
		}
		cell := pc.Cell
		cell.X = x
		cell.Y = y
		return cell
	}
	return generateCellInRegion(regionID, x, y)
}

func isCellClearedRecently(regionID string, x, y int, cooldown time.Duration) bool {
	cleared, _, err := cache.IsCellCleared(regionID, x, y)
	if err != nil {
		logger.S.Errorf("Failed to check Redis cell state for (%d,%d) in region %s: %v", x, y, regionID, err)
		return false
	}
	return cleared
}

// cleanupExpiredClearedCells removes cells from player.ClearedCells that have exceeded the cooldown duration.
// Returns true if any cells were removed. Kept for backward compatibility with legacy players.
func cleanupExpiredClearedCells(player *models.Player, cooldown time.Duration) bool {
	if player.ClearedCells == nil {
		return false
	}
	changed := false
	now := time.Now()
	for coord, timeStr := range player.ClearedCells {
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			delete(player.ClearedCells, coord)
			changed = true
			continue
		}
		if now.Sub(t) >= cooldown {
			delete(player.ClearedCells, coord)
			changed = true
		}
	}
	return changed
}
