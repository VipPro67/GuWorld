package handlers

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const clearedCellCooldown = 15 * time.Minute

const (
	monsterCampSpacing     = 48
	monsterCampRadius      = 6
	monsterCampEliteChance = 0.10
)

type proceduralMonsterCamp struct {
	spawn    models.MonsterSpawn
	elite    models.MonsterSpawn
	hasElite bool
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

// defaultResourcePool / defaultWildGuPool are fallback content pools used by
// zones that do not define pool_weights in Region.csv.
var defaultResourcePool = []string{"moon_orchid_petal", "green_bamboo_wine", "jade_stone", "boar_meat"}
var defaultWildGuPool = []string{"GU_LITTLE_LIGHT", "GU_JADE_SKIN", "GU_BRONZE_RELIC", "GU_FIRE_ARMOR"}

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
	if hasCamp && camp.hasElite && x == camp.x && y == camp.y {
		return models.Cell{X: x, Y: y, Type: "Monster", Content: camp.elite.MonsterID,
			Rank: camp.elite.MinRank, SubRank: rng.Intn(5) + 1}
	}
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
		monsterID := ""
		if spawn, ok := pickMonsterFromPool(zoneID, rng, false); ok {
			monsterID = spawn.MonsterID
		} else {
			monsterID = randomNormalMonsterID(rng)
		}
		rank := zone.MinRank
		if zone.MaxRank > zone.MinRank {
			rank += rng.Intn(zone.MaxRank - zone.MinRank + 1)
		}
		if rank < 1 {
			rank = 1
		}
		return models.Cell{X: x, Y: y, Type: "Monster", Content: monsterID, Rank: rank, SubRank: rng.Intn(5) + 1}
	case "Resource":
		type weightedResource struct {
			id     string
			weight float64
		}
		var pool []weightedResource
		var total float64
		for _, id := range zone.PoolWeights.Resources {
			w := 1.0
			if spec, ok := models.ItemRegistry[id]; ok {
				w = guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
			}
			if w <= 0 {
				w = 1.0
			}
			pool = append(pool, weightedResource{id: id, weight: w})
			total += w
		}
		if len(pool) == 0 {
			for _, id := range defaultResourcePool {
				w := 1.0
				if spec, ok := models.ItemRegistry[id]; ok {
					w = guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
				}
				pool = append(pool, weightedResource{id: id, weight: w})
				total += w
			}
		}
		itemID := ""
		r := rng.Float64() * total
		for _, e := range pool {
			r -= e.weight
			if r <= 0 {
				itemID = e.id
				break
			}
		}
		if itemID == "" {
			itemID = pool[0].id
		}
		return models.Cell{X: x, Y: y, Type: "Resource", Content: itemID}
	case "WildGu":
		type weightedGu struct {
			id     string
			weight float64
		}
		var pool []weightedGu
		var total float64
		for _, id := range zone.PoolWeights.WildGus {
			if models.HasImmortalRank(id) {
				continue
			}
			w := 1.0
			if spec, ok := models.GetGuAny(id); ok {
				w = guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
			}
			if w <= 0 {
				w = 1.0
			}
			pool = append(pool, weightedGu{id: id, weight: w})
			total += w
		}
		if len(pool) == 0 {
			for _, id := range defaultWildGuPool {
				pool = append(pool, weightedGu{id: id, weight: 1.0})
				total += 1.0
			}
		}
		guID := ""
		r := rng.Float64() * total
		for _, e := range pool {
			r -= e.weight
			if r <= 0 {
				guID = e.id
				break
			}
		}
		if guID == "" {
			guID = pool[0].id
		}
		rank := zone.MinRank
		if rank < 1 {
			rank = 1
		}
		return models.Cell{X: x, Y: y, Type: "WildGu", Content: guID, Rank: rank}
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
			monsterID := ""
			minRank, maxRank := zone.MinRank, zone.MaxRank
			if ok {
				monsterID, minRank, maxRank = spawn.MonsterID, spawn.MinRank, spawn.MaxRank
			}
			if monsterID == "" {
				monsterID = randomNormalMonsterID(rng)
			}
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
			best = proceduralMonsterCamp{spawn: models.MonsterSpawn{MonsterID: monsterID, MinRank: rank}, x: cx, y: cy, distance: distance}
			eliteRNG := rand.New(rand.NewSource(GetCellSeed(gx, gy, zone.ID+":monster-camp:elite")))
			if eliteRNG.Float64() < monsterCampEliteChance {
				if eliteSpawn, foundElite := pickMonsterFromPoolOfType(zone.ID, eliteRNG, "Elite"); foundElite {
					eliteMinRank, eliteMaxRank := eliteSpawn.MinRank, eliteSpawn.MaxRank
					if eliteMinRank < 1 {
						eliteMinRank = minRank
					}
					if eliteMinRank < 1 {
						eliteMinRank = 1
					}
					if eliteMaxRank < eliteMinRank {
						eliteMaxRank = eliteMinRank
					}
					eliteRank := eliteMinRank
					if eliteMaxRank > eliteMinRank {
						eliteRank += eliteRNG.Intn(eliteMaxRank - eliteMinRank + 1)
					}
					best.elite = models.MonsterSpawn{MonsterID: eliteSpawn.MonsterID, MinRank: eliteRank}
					best.hasElite = true
				}
			}
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

// randomNormalMonsterID picks a Normal-type monster from the in-memory registry.
func randomNormalMonsterID(rng *rand.Rand) string {
	var ids []string
	for id, mon := range models.MonsterRegistry {
		if strings.EqualFold(mon.Type, "Normal") {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "monster_wild_boar"
	}
	// Sort so the pick is stable across calls (map iteration order is random).
	sort.Strings(ids)
	return ids[rng.Intn(len(ids))]
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
