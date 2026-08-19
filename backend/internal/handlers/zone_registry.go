package handlers

import (
	"database/sql"
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// persistentCellKey identifies a shared, DB-backed cell by global coords.
type persistentCellKey struct{ X, Y int }

// persistentCell is a cell on the shared map: boss, high-tier resource or a
// safe point (village / portal). Only these cells touch the database; all
// low-tier content is generated procedurally.
type persistentCell struct {
	RegionID string
	models.Cell
	RespawnSeconds int
	LastCleared    time.Time
	cleared        bool
}

// zoneRegistry holds all world geometry and content pools in memory. It is
// loaded once at startup; changing data requires a backend restart (same
// convention as the other registries).
type zoneRegistry struct {
	mu         sync.RWMutex
	zones      []models.Region
	byID       map[string]*models.Region
	persistent map[persistentCellKey]persistentCell
	spawnPools map[string][]models.MonsterSpawn
}

var zoneReg = &zoneRegistry{}

// zoneType constants.
const (
	ZoneTypeHub     = "HUB"
	ZoneTypeSafe    = "SAFE_ZONE"
	ZoneTypeBoss    = "BOSS_ZONE"
	ZoneTypeSecret  = "SECRET_REALM"
	ZoneTypeDungeon = "DUNGEON"
	ZoneTypePvp     = "PVP_ZONE"
	ZoneTypeLarge   = "NGU_VUC"
	ZoneTypeNamed   = "KHU_VUC"
)

// zoneTypePriority ranks zone types from highest (most specific) to lowest.
// When a point falls inside overlapping zones, the zone with the highest
// priority wins; ties are broken by smallest area (innermost). Unknown zone
// types are treated as KHU_VUC priority.
var zoneTypePriority = map[string]int{
	ZoneTypeHub:     100,
	ZoneTypeSafe:    90,
	ZoneTypeBoss:    80,
	ZoneTypeSecret:  70,
	ZoneTypeDungeon: 60,
	ZoneTypePvp:     50,
	ZoneTypeNamed:   40,
	ZoneTypeLarge:   10,
}

// LoadZoneRegistry loads regions (zones), persistent map cells and monster
// spawn pools into memory. Called once at startup.
func LoadZoneRegistry(db *sql.DB) error {
	newReg := &zoneRegistry{
		byID:       make(map[string]*models.Region),
		persistent: make(map[persistentCellKey]persistentCell),
		spawnPools: make(map[string][]models.MonsterSpawn),
	}

	rows, err := db.Query(`
		SELECT id, name, COALESCE(description,''), COALESCE(width,100), COALESCE(height,100),
		       COALESCE(min_rank,1), COALESCE(max_rank,5), is_active,
		       COALESCE(parent_id,''), COALESCE(zone_type,'KHU_VUC'),
		       COALESCE(min_x,0), COALESCE(min_y,0), COALESCE(max_x,0), COALESCE(max_y,0),
		       COALESCE(entry_x,0), COALESCE(entry_y,0), COALESCE(pool_weights::text,'{}')
		FROM regions
		WHERE is_active = TRUE
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var z models.Region
		var active bool
		var poolRaw string
		if err := rows.Scan(&z.ID, &z.Name, &z.Description, &z.Width, &z.Height,
			&z.MinRank, &z.MaxRank, &active,
			&z.ParentID, &z.ZoneType,
			&z.MinX, &z.MinY, &z.MaxX, &z.MaxY,
			&z.EntryX, &z.EntryY, &poolRaw); err != nil {
			logger.S.Warnf("zone registry scan: %v", err)
			continue
		}
		if z.ZoneType == "" {
			z.ZoneType = ZoneTypeNamed
		}
		if poolRaw != "" && poolRaw != "{}" {
			_ = json.Unmarshal([]byte(poolRaw), &z.PoolWeights)
		}
		// legacy width/height backfill
		if z.MaxX == 0 && z.MaxY == 0 && z.Width > 1 && z.Height > 1 {
			z.MaxX = z.Width - 1
			z.MaxY = z.Height - 1
		}
		newReg.zones = append(newReg.zones, z)
		newReg.byID[z.ID] = &newReg.zones[len(newReg.zones)-1]
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Persistent shared cells (bosses, high resources, safe points).
	cells, err := db.Query(`
		SELECT region_id, x, y, cell_type, COALESCE(content_id,''), COALESCE(metadata::text,'{}'),
		       COALESCE(respawn_seconds,300), COALESCE(last_cleared_at, NOW() - INTERVAL '999 days'),
		       COALESCE(is_active, TRUE)
		FROM map_cells
		WHERE is_active = TRUE
	`)
	if err != nil {
		return err
	}
	defer cells.Close()
	for cells.Next() {
		var pc persistentCell
		var metaRaw string
		var isActive bool
		if err := cells.Scan(&pc.RegionID, &pc.X, &pc.Y, &pc.Type, &pc.Content, &metaRaw,
			&pc.RespawnSeconds, &pc.LastCleared, &isActive); err != nil {
			logger.S.Warnf("persistent cell scan: %v", err)
			continue
		}
		if metaRaw != "" && metaRaw != "{}" {
			var meta struct {
				Rank int `json:"rank"`
			}
			if json.Unmarshal([]byte(metaRaw), &meta) == nil {
				pc.Rank = meta.Rank
			}
		}
		key := persistentCellKey{X: pc.X, Y: pc.Y}
		// Cell is considered cleared while within its respawn window.
		if !pc.LastCleared.IsZero() && time.Since(pc.LastCleared) < time.Duration(pc.RespawnSeconds)*time.Second {
			pc.cleared = true
		}
		newReg.persistent[key] = pc
	}
	if err := cells.Err(); err != nil {
		return err
	}

	// Monster spawn pools per zone.
	spawns, err := db.Query(`
		SELECT region_id, monster_id, x, y, COALESCE(spawn_mode,'Random'), COALESCE(chance,1.0),
		       COALESCE(respawn_seconds,300), COALESCE(min_rank,1), COALESCE(max_rank,9),
		       COALESCE(is_boss_spawn,false), COALESCE(is_active,TRUE)
		FROM monster_spawns
		WHERE is_active = TRUE
		ORDER BY region_id, monster_id
	`)
	if err != nil {
		return err
	}
	defer spawns.Close()
	for spawns.Next() {
		var s models.MonsterSpawn
		var x, y sql.NullInt64
		if err := spawns.Scan(&s.RegionID, &s.MonsterID, &x, &y, &s.SpawnMode, &s.Chance,
			&s.RespawnSeconds, &s.MinRank, &s.MaxRank, &s.IsBossSpawn, &s.IsActive); err != nil {
			logger.S.Warnf("spawn pool scan: %v", err)
			continue
		}
		if x.Valid {
			xi := int(x.Int64)
			s.X = &xi
		}
		if y.Valid {
			yi := int(y.Int64)
			s.Y = &yi
		}
		newReg.spawnPools[s.RegionID] = append(newReg.spawnPools[s.RegionID], s)
	}
	if err := spawns.Err(); err != nil {
		return err
	}

	zoneReg.mu.Lock()
	zoneReg.zones = newReg.zones
	zoneReg.byID = newReg.byID
	zoneReg.persistent = newReg.persistent
	zoneReg.spawnPools = newReg.spawnPools
	zoneReg.mu.Unlock()

	// Procedural cell content depends on the zone layout, so any (re)load
	// invalidates cached cells.
	cellGenCache.clear()

	logger.S.Infof("Zone registry loaded: %d zones, %d persistent cells, %d spawn pools", len(newReg.zones), len(newReg.persistent), len(newReg.spawnPools))
	return nil
}

// ensureZonesLoaded lazily (re)loads the zone registry if it is empty. This
// covers backends that started before CSV data was imported through the Admin
// panel, so exploration does not hard-fail with a stale empty registry.
func ensureZonesLoaded() {
	zoneReg.mu.RLock()
	loaded := len(zoneReg.zones) > 0
	zoneReg.mu.RUnlock()
	if loaded {
		return
	}
	if err := LoadZoneRegistry(db.DB); err != nil {
		logger.S.Warnf("lazy zone registry load failed: %v", err)
	}
}

// zoneContains reports whether a point falls inside the zone rect (inclusive).
func zoneContains(z *models.Region, x, y int) bool {
	return x >= z.MinX && x <= z.MaxX && y >= z.MinY && y <= z.MaxY
}

func zoneArea(z *models.Region) int {
	return (z.MaxX - z.MinX + 1) * (z.MaxY - z.MinY + 1)
}

// FindZoneAt returns the active zone containing the point with the highest
// priority, breaking ties by smallest area (innermost). The hub (safe lobby)
// has the highest priority, so it naturally wins over any overlapping zone.
func FindZoneAt(x, y int) *models.Region {
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()
	var best *models.Region
	bestPriority := -1
	bestArea := 0
	for i := range zoneReg.zones {
		z := &zoneReg.zones[i]
		if !zoneContains(z, x, y) {
			continue
		}
		prio := zonePriorityOf(z.ZoneType)
		if best == nil || prio > bestPriority || (prio == bestPriority && zoneArea(z) < bestArea) {
			best = z
			bestPriority = prio
			bestArea = zoneArea(z)
		}
	}
	return best
}

// zonePriorityOf returns the resolution priority of a zone type. Unknown zone
// types fall back to the KHU_VUC (named) priority.
func zonePriorityOf(zoneType string) int {
	if prio, ok := zoneTypePriority[zoneType]; ok {
		return prio
	}
	return zoneTypePriority[ZoneTypeNamed]
}

// FindLargeZoneAt returns the NGU_VUC zone containing the point.
func FindLargeZoneAt(x, y int) *models.Region {
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()
	for i := range zoneReg.zones {
		z := &zoneReg.zones[i]
		if z.ZoneType == ZoneTypeLarge && zoneContains(z, x, y) {
			return z
		}
	}
	return nil
}

// ZoneByID returns the zone with the given id, or nil.
func ZoneByID(id string) *models.Region {
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()
	return zoneReg.byID[id]
}

// hubEntryPoint returns the hub (lobby) spawn coordinates.
func hubEntryPoint() (int, int) {
	if z := ZoneByID(HubRegionID); z != nil && z.EntryX != 0 && z.EntryY != 0 {
		return z.EntryX, z.EntryY
	}
	return HubEntryPoint.X, HubEntryPoint.Y
}

// NearestHub returns the region ID and entry coordinates of the closest HUB zone.
func NearestHub(x, y int) (string, int, int) {
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()

	var bestHub *models.Region
	var minDist float64 = -1

	for i := range zoneReg.zones {
		z := &zoneReg.zones[i]
		if z.ZoneType == ZoneTypeHub {
			dx := float64(z.EntryX - x)
			dy := float64(z.EntryY - y)
			dist := dx*dx + dy*dy
			if minDist < 0 || dist < minDist {
				minDist = dist
				bestHub = z
			}
		}
	}

	if bestHub != nil && bestHub.EntryX != 0 && bestHub.EntryY != 0 {
		return bestHub.ID, bestHub.EntryX, bestHub.EntryY
	}
	return HubRegionID, HubEntryPoint.X, HubEntryPoint.Y
}

// DeriveRegionAt resolves the current_region for a world position.
func DeriveRegionAt(x, y int) string {
	if z := FindZoneAt(x, y); z != nil {
		return z.ID
	}
	return HubRegionID
}

// SubZoneKey returns the small-zone key ("<large>:<col>:<row>") for a point.
func SubZoneKey(x, y int) string {
	z := FindLargeZoneAt(x, y)
	if z == nil {
		return HubRegionID
	}
	col := (x - z.MinX) / SubZoneSize
	row := (y - z.MinY) / SubZoneSize
	return z.ID + ":" + strconv.Itoa(col) + ":" + strconv.Itoa(row)
}

// MarkPersistentCellCleared records that a shared cell (boss / high resource)
// was cleared and persists the respawn state to map_cells.
func MarkPersistentCellCleared(tx *sql.Tx, playerID uuid.UUID, x, y int) {
	key := persistentCellKey{X: x, Y: y}
	zoneReg.mu.RLock()
	pc, ok := zoneReg.persistent[key]
	zoneReg.mu.RUnlock()
	if !ok {
		return
	}
	wasCleared := pc.cleared
	pc.LastCleared = time.Now()
	pc.cleared = true
	zoneReg.mu.Lock()
	zoneReg.persistent[key] = pc
	zoneReg.mu.Unlock()
	if tx != nil {
		_, _ = tx.Exec(`UPDATE map_cells SET last_cleared_at = NOW(), cleared_by = $1 WHERE x = $2 AND y = $3`, playerID, x, y)
	}
	if !wasCleared {
		broadcastWorldCell(cellEventCleared, WorldCellEvent{X: pc.X, Y: pc.Y, CellType: pc.Type, ContentID: pc.Content})
	}
}

// MarkPersistentCellClearedNoTx clears a shared cell in memory only (used when
// no transaction is available). The DB state is updated lazily on the next
// MarkPersistentCellCleared call for the same cell.
func MarkPersistentCellClearedNoTx(x, y int) {
	MarkPersistentCellCleared(nil, uuid.Nil, x, y)
}

// PersistentCellAt returns the shared cell at (x,y) if any. found=false means
// there is no shared cell; cleared=true means it exists but is inside its
// respawn window (treat as empty). When the respawn window has elapsed, the
// cell is lazily reset here (no background goroutine needed) and a respawn
// event is broadcast so map clients can refresh the marker.
func PersistentCellAt(x, y int) (pc persistentCell, found, cleared bool) {
	key := persistentCellKey{X: x, Y: y}
	zoneReg.mu.RLock()
	pc, ok := zoneReg.persistent[key]
	zoneReg.mu.RUnlock()
	if !ok {
		return pc, false, false
	}
	if pc.cleared && !pc.LastCleared.IsZero() && pc.RespawnSeconds > 0 &&
		time.Since(pc.LastCleared) >= time.Duration(pc.RespawnSeconds)*time.Second {
		zoneReg.mu.Lock()
		if cur, ok2 := zoneReg.persistent[key]; ok2 && cur.cleared {
			cur.cleared = false
			zoneReg.persistent[key] = cur
			pc = cur
		}
		zoneReg.mu.Unlock()
		broadcastWorldCell(cellEventRespawn, WorldCellEvent{X: pc.X, Y: pc.Y, CellType: pc.Type, ContentID: pc.Content})
		return pc, true, false
	}
	return pc, true, pc.cleared
}

// pickMonsterFromPool deterministically picks a monster from a zone's spawn
// pool using the provided seeded rng. wantBoss filters boss spawns.
func pickMonsterFromPool(regionID string, rng *rand.Rand, wantBoss bool) (models.MonsterSpawn, bool) {
	zoneReg.mu.RLock()
	pool := zoneReg.spawnPools[regionID]
	zoneReg.mu.RUnlock()

	var candidates []models.MonsterSpawn
	totalWeight := 0.0
	for _, s := range pool {
		if s.IsBossSpawn != wantBoss {
			continue
		}
		if wantBoss && s.IsBossSpawn && s.X != nil && s.Y != nil {
			// Fixed-position bosses are shared persistent cells; skip them here.
			continue
		}
		candidates = append(candidates, s)
		totalWeight += s.Chance
	}
	if len(candidates) == 0 {
		return models.MonsterSpawn{}, false
	}
	roll := rng.Float64() * totalWeight
	for _, s := range candidates {
		roll -= s.Chance
		if roll <= 0 {
			return s, true
		}
	}
	return candidates[len(candidates)-1], true
}

// markBossCleared marks the persistent boss cell at the player's position as
// cleared after a victorious boss fight on the shared map.
func markBossCleared(tx *sql.Tx, player *models.Player) {
	if player == nil || player.ActiveCombat == nil {
		return
	}
	et := strings.ToUpper(player.ActiveCombat.EnemyType)
	if et != "BOSS" && et != "WORLDBOSS" && et != "WORLD BOSS" {
		return
	}
	MarkPersistentCellCleared(tx, player.ID, player.X, player.Y)
}

// recordVisitedSubZone appends a small-zone key to the player's visited set
// if not already present (deduplicated, bounded by the world's sub-zone count).
func recordVisitedSubZone(player *models.Player, key string) {
	if key == "" {
		return
	}
	for _, k := range player.VisitedSubZones {
		if k == key {
			return
		}
	}
	player.VisitedSubZones = append(player.VisitedSubZones, key)
}
