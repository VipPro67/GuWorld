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

// persistentCellKey identifies a shared, DB-backed cell by global world coords.
// Boss/resource/safe-point cells are intentionally global to the server, not
// scoped to a single player or a single region. The DB also enforces a global
// (x, y) uniqueness, so all players resolve the same boss marker from the same
// world coordinate.
type persistentCellKey struct {
	RegionID string
	X, Y     int
}

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
	if db == nil {
		return nil
	}
	newReg := &zoneRegistry{
		byID:       make(map[string]*models.Region),
		persistent: make(map[persistentCellKey]persistentCell),
		spawnPools: make(map[string][]models.MonsterSpawn),
	}

	rows, err := db.Query(`
		SELECT r.id, r.name, COALESCE(r.description,''), COALESCE(r.width,100), COALESCE(r.height,100),
		       COALESCE(min_rank,1), COALESCE(max_rank,5), is_active,
		       COALESCE(parent_id,''), COALESCE(zone_type,'KHU_VUC'),
		       COALESCE(min_x,0), COALESCE(min_y,0), COALESCE(max_x,0), COALESCE(max_y,0),
		       COALESCE(entry_x,0), COALESCE(entry_y,0), COALESCE(pool_weights::text,'{}'), COALESCE(rules::text,'{}'),
		       COALESCE(safe_zone_x,0), COALESCE(safe_zone_y,0), COALESCE(rwm.world_map_id,'world_map'),
		       COALESCE(r.atlas_x,0), COALESCE(r.atlas_y,0)
		FROM regions r LEFT JOIN region_world_maps rwm ON rwm.region_id = r.id
		WHERE is_active = TRUE
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var z models.Region
		var active bool
		var poolRaw, rulesRaw string
		var safeX, safeY sql.NullInt64
		if err := rows.Scan(&z.ID, &z.Name, &z.Description, &z.Width, &z.Height,
			&z.MinRank, &z.MaxRank, &active,
			&z.ParentID, &z.ZoneType,
			&z.MinX, &z.MinY, &z.MaxX, &z.MaxY,
			&z.EntryX, &z.EntryY, &poolRaw, &rulesRaw, &safeX, &safeY, &z.WorldMapID,
			&z.AtlasX, &z.AtlasY); err != nil {

			logger.S.Warnf("zone registry scan: %v", err)
			continue
		}
		if z.ZoneType == "" {
			z.ZoneType = ZoneTypeNamed
		}
		z.IsActive = active
		// Assign safe zone pointers if present
		if safeX.Valid {
			val := int(safeX.Int64)
			z.SafeZoneX = &val
		}
		if safeY.Valid {
			val := int(safeY.Int64)
			z.SafeZoneY = &val
		}
		if poolRaw != "" && poolRaw != "{}" {
			_ = json.Unmarshal([]byte(poolRaw), &z.PoolWeights)
		}
		z.Rules = json.RawMessage(rulesRaw)
		// legacy width/height backfill
		if z.MaxX == 0 && z.MaxY == 0 && z.Width > 1 && z.Height > 1 {
			z.MaxX = z.Width - 1
			z.MaxY = z.Height - 1
		}
		newReg.zones = append(newReg.zones, z)
	}
	for i := range newReg.zones {
		newReg.byID[newReg.zones[i].ID] = &newReg.zones[i]
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
		key := persistentCellKey{RegionID: pc.RegionID, X: pc.X, Y: pc.Y}
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
	// Rank-band content pools are derived from the item/Gu registries, which are
	// reloaded alongside the zone registry on every admin import.
	clearCellPoolCache()

	logger.S.Infof("Zone registry loaded: %d zones, %d persistent cells, %d spawn pools", len(newReg.zones), len(newReg.persistent), len(newReg.spawnPools))
	return nil
}

// ensureZonesLoaded lazily (re)loads the zone registry if it is empty. This
// covers backends that started before CSV data was imported through the Admin
// panel, so exploration does not hard-fail with a stale empty registry.
func ensureZonesLoaded() {
	if db.DB == nil {
		return
	}
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
		// SECRET_REALM bounds are reused coordinates owned by a running
		// shared spatial instance. They must never capture ordinary world-map
		// positions; instance movement keeps its explicitly assigned region.
		if strings.EqualFold(z.ZoneType, ZoneTypeSecret) {
			continue
		}
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

// FindZoneAtOnMap resolves overlapping coordinates only among regions assigned
// to a single world-map layer. An empty layer keeps the legacy global lookup.
func FindZoneAtOnMap(x, y int, mapID string) *models.Region {
	if mapID == "" {
		return FindZoneAt(x, y)
	}
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()
	var best *models.Region
	bestPriority, bestArea := -1, 0
	for i := range zoneReg.zones {
		z := &zoneReg.zones[i]
		if strings.EqualFold(z.ZoneType, ZoneTypeSecret) {
			continue
		}
		if z.WorldMapID != mapID || !zoneContains(z, x, y) {
			continue
		}
		prio, area := zonePriorityOf(z.ZoneType), zoneArea(z)
		if best == nil || prio > bestPriority || (prio == bestPriority && area < bestArea) {
			best, bestPriority, bestArea = z, prio, area
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
	if specific := FindZoneAt(x, y); specific != nil {
		if lz := largeZoneForRegion(specific.ID); lz != nil {
			return lz
		}
	}
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

// greatRegionID returns the NGU_VUC ancestor of a region. Regions without a
// known large zone (tests, legacy ids) fall back to themselves.
func greatRegionID(regionID string) string {
	if lz := largeZoneForRegion(regionID); lz != nil {
		return lz.ID
	}
	return regionID
}

// regionWorldPos converts a local map cell into world-map coordinates so maps
// in the same great region can be compared by real distance.
func regionWorldPos(regionID string, x, y int) (int, int) {
	if zone := ZoneByID(regionID); zone != nil {
		return zone.MinX + x, zone.MinY + y
	}
	return x, y
}

// regionScopeIDs returns a region and its containing regions on the same map.
// Shared entities assigned to a large parent region can therefore be seen by
// players whose current region is one of its child areas.
func regionScopeIDs(regionID string) []string {
	ids := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	child := ZoneByID(regionID)
	mapID := ""
	if child != nil {
		mapID = child.WorldMapID
	}
	for current := regionID; current != ""; {
		if _, duplicate := seen[current]; duplicate {
			break
		}
		seen[current] = struct{}{}
		ids = append(ids, current)
		zone := ZoneByID(current)
		if zone == nil || zone.ParentID == "" || zone.ParentID == current {
			break
		}
		parent := ZoneByID(zone.ParentID)
		if parent == nil || (mapID != "" && parent.WorldMapID != "" && parent.WorldMapID != mapID) {
			break
		}
		current = zone.ParentID
	}
	if len(ids) == 0 && regionID != "" {
		return []string{regionID}
	}
	return ids
}

// isRegionWithinScope reports whether candidateRegion is parentRegion itself or
// one of its descendant zones. The check is deliberately directional: sibling
// zones that merely share a large parent are NOT in scope, otherwise a rank-8
// hunter in a high-rank neighbour could aggro/chase a player standing in a
// low-rank zone.
func isRegionWithinScope(parentRegion, candidateRegion string) bool {
	if parentRegion == "" || candidateRegion == "" {
		return false
	}
	if parentRegion == candidateRegion {
		return true
	}
	for _, id := range regionScopeIDs(candidateRegion) {
		if id == parentRegion {
			return true
		}
	}
	return false
}

// clampRankToZoneBand caps an effective monster rank at the owning zone's max
// rank, so a stale or mis-authored spawn row can never field a monster stronger
// than the zone band advertises.
func clampRankToZoneBand(regionID string, rank int) int {
	zone := ZoneByID(regionID)
	if zone == nil || zone.MaxRank <= 0 || rank <= zone.MaxRank {
		return rank
	}
	return zone.MaxRank
}

// monsterCanRoamInto reports whether a monster of the given rank may occupy
// (x, y). Ordinary world monsters are only leashed to their spawn point, not to
// their zone, so without this check a rank-8 hunter can follow a player across
// a border into a zone whose band never spawns rank-8 monsters.
func monsterCanRoamInto(rank int, x, y int) bool {
	zone := FindZoneAt(x, y)
	if zone == nil || zone.MaxRank <= 0 {
		return true
	}
	return rank <= zone.MaxRank
}

func activeScopeRegions(minX, minY, maxX, maxY int, mapID, playerRegion string) []string {
	seen := make(map[string]struct{})
	var result []string
	add := func(id string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	for _, scope := range regionScopeIDs(playerRegion) {
		add(scope)
	}
	for _, zone := range rectZoneCandidates(minX, minY, maxX, maxY, mapID) {
		add(zone.ID)
		for _, scope := range regionScopeIDs(zone.ID) {
			add(scope)
		}
	}
	return result
}

func worldMapIDForRegion(regionID string) string {
	seen := make(map[string]struct{}, 4)
	for current := regionID; current != ""; {
		if _, exists := seen[current]; exists {
			break
		}
		seen[current] = struct{}{}
		region := ZoneByID(current)
		if region == nil {
			break
		}
		if region.WorldMapID != "" {
			return region.WorldMapID
		}
		current = region.ParentID
	}
	return "world_map"
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
	return nearestHubOnMap(x, y, "")
}

// NearestHubOnMap keeps respawns on the player's current world map layer when
// that layer has a HUB. Coordinates are reused between layers, so comparing
// every HUB globally can otherwise choose a hub on another map.
func NearestHubOnMap(x, y int, mapID string) (string, int, int) {
	return nearestHubOnMap(x, y, mapID)
}

// NearestRespawnPointOnMap prefers a HUB on the given map, then falls back to
// the nearest SAFE_ZONE, but only among zones whose entry lies inside the
// NGU_VUC containing (x, y). It never returns a point from another map layer or
// another đại vực.
func NearestRespawnPointOnMap(x, y int, mapID string) (string, int, int, bool) {
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()

	// Respawn must stay inside the NGU_VUC the player died in. Every Đại Vực
	// shares the same world-map layer ("world_map"), so a plain map filter would
	// let a death in one đại vực pull the nearest HUB/SAFE_ZONE from another.
	var anchor *models.Region
	for i := range zoneReg.zones {
		z := &zoneReg.zones[i]
		if z.ZoneType == ZoneTypeLarge && zoneContains(z, x, y) {
			anchor = z
			break
		}
	}

	var bestHub, bestSafe *models.Region
	var hubDist, safeDist float64 = -1, -1
	for i := range zoneReg.zones {
		z := &zoneReg.zones[i]
		if z.ZoneType != ZoneTypeHub && z.ZoneType != ZoneTypeSafe {
			continue
		}
		if z.EntryX == 0 || z.EntryY == 0 {
			continue
		}
		if mapID != "" && zoneWorldMapIDLocked(z) != mapID {
			continue
		}
		if anchor != nil && !zoneContains(anchor, z.EntryX, z.EntryY) {
			continue
		}
		dx := float64(z.EntryX - x)
		dy := float64(z.EntryY - y)
		dist := dx*dx + dy*dy
		if z.ZoneType == ZoneTypeHub {
			if hubDist < 0 || dist < hubDist {
				bestHub, hubDist = z, dist
			}
		} else if safeDist < 0 || dist < safeDist {
			bestSafe, safeDist = z, dist
		}
	}
	if bestHub != nil && bestHub.EntryX != 0 && bestHub.EntryY != 0 {
		return bestHub.ID, bestHub.EntryX, bestHub.EntryY, true
	}
	if bestSafe != nil && bestSafe.EntryX != 0 && bestSafe.EntryY != 0 {
		return bestSafe.ID, bestSafe.EntryX, bestSafe.EntryY, true
	}
	return "", 0, 0, false
}

func zoneWorldMapIDLocked(zone *models.Region) string {
	if zone == nil {
		return ""
	}
	if zone.WorldMapID != "" {
		return zone.WorldMapID
	}
	seen := make(map[string]struct{}, 4)
	for parentID := zone.ParentID; parentID != ""; {
		if _, exists := seen[parentID]; exists {
			break
		}
		seen[parentID] = struct{}{}
		parent := zoneReg.byID[parentID]
		if parent == nil {
			break
		}
		if parent.WorldMapID != "" {
			return parent.WorldMapID
		}
		if parent.ParentID == parentID {
			break
		}
		parentID = parent.ParentID
	}
	return "world_map"
}

func nearestHubOnMap(x, y int, mapID string) (string, int, int) {
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()

	var bestHub *models.Region
	var minDist float64 = -1
	var bestAnyHub *models.Region
	var minAnyDist float64 = -1

	for i := range zoneReg.zones {
		z := &zoneReg.zones[i]
		if z.ZoneType == ZoneTypeHub {
			dx := float64(z.EntryX - x)
			dy := float64(z.EntryY - y)
			dist := dx*dx + dy*dy
			if minAnyDist < 0 || dist < minAnyDist {
				minAnyDist = dist
				bestAnyHub = z
			}
			hubMapID := zoneWorldMapIDLocked(z)
			if mapID != "" && hubMapID != mapID {
				continue
			}
			if minDist < 0 || dist < minDist {
				minDist = dist
				bestHub = z
			}
		}
	}
	if bestHub == nil && mapID == "" {
		bestHub = bestAnyHub
	}

	if bestHub != nil && bestHub.EntryX != 0 && bestHub.EntryY != 0 {
		return bestHub.ID, bestHub.EntryX, bestHub.EntryY
	}
	if mapID != "" {
		return "", 0, 0
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

func DeriveRegionAtOnMap(x, y int, mapID string) string {
	if z := FindZoneAtOnMap(x, y, mapID); z != nil {
		return z.ID
	}
	return HubRegionID
}

// rectZoneCandidates snapshots only zones that can affect a rectangular map
// read. Preserve registry order so equal-priority, equal-area ties resolve the
// same way as FindZoneAtOnMap.
func rectZoneCandidates(minX, minY, maxX, maxY int, mapID string) []models.Region {
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()
	candidates := make([]models.Region, 0)
	for _, zone := range zoneReg.zones {
		if strings.EqualFold(zone.ZoneType, ZoneTypeSecret) || (mapID != "" && zone.WorldMapID != mapID) {
			continue
		}
		if zone.MaxX < minX || zone.MinX > maxX || zone.MaxY < minY || zone.MinY > maxY {
			continue
		}
		candidates = append(candidates, zone)
	}
	return candidates
}

func regionAtFromRectZones(x, y int, candidates []models.Region) string {
	bestID := HubRegionID
	bestPriority, bestArea := -1, 0
	for i := range candidates {
		zone := &candidates[i]
		if !zoneContains(zone, x, y) {
			continue
		}
		priority, area := zonePriorityOf(zone.ZoneType), zoneArea(zone)
		if priority > bestPriority || (priority == bestPriority && area < bestArea) {
			bestID, bestPriority, bestArea = zone.ID, priority, area
		}
	}
	return bestID
}

// IsSafeZone returns true if the coordinates belong to a HUB or SAFE_ZONE.
// A region whose ID maps to a hub/safe zone type only counts when the
// coordinates are actually inside that zone's rectangle — otherwise players
// standing in the wilderness between zones would be treated as safe.
func IsSafeZone(regionID string, x, y int) bool {
	if z := ZoneByID(regionID); z != nil && zoneContains(z, x, y) {
		return z.ZoneType == ZoneTypeHub || z.ZoneType == ZoneTypeSafe
	}
	if z := FindZoneAtOnMap(x, y, worldMapIDForRegion(regionID)); z != nil {
		if z.ZoneType == ZoneTypeHub || z.ZoneType == ZoneTypeSafe {
			return true
		}
	}
	return false
}

// GetSafePoint returns safe coordinates for a region (fallback to entry point)
func GetSafePoint(regionID string) (int, int) {
	if z := ZoneByID(regionID); z != nil {
		if z.SafeZoneX != nil && z.SafeZoneY != nil {
			return *z.SafeZoneX, *z.SafeZoneY
		}
		return z.EntryX, z.EntryY
	}
	// fallback to hub entry point
	return HubEntryPoint.X, HubEntryPoint.Y
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

func persistentCellForRegion(regionID string, x, y int) (persistentCell, bool) {
	for current := regionID; current != ""; {
		key := persistentCellKey{RegionID: current, X: x, Y: y}
		zoneReg.mu.RLock()
		pc, ok := zoneReg.persistent[key]
		zoneReg.mu.RUnlock()
		if ok {
			return pc, true
		}
		zone := ZoneByID(current)
		if zone == nil || zone.ParentID == current {
			break
		}
		current = zone.ParentID
	}
	return persistentCell{}, false
}

func MarkPersistentCellCleared(tx *sql.Tx, playerID uuid.UUID, regionID string, x, y int) error {
	pc, ok := persistentCellForRegion(regionID, x, y)
	if !ok {
		return nil
	}
	key := persistentCellKey{RegionID: pc.RegionID, X: x, Y: y}
	wasCleared := pc.cleared
	pc.LastCleared = time.Now()
	pc.cleared = true
	zoneReg.mu.Lock()
	zoneReg.persistent[key] = pc
	zoneReg.mu.Unlock()
	if tx != nil {
		if _, err := tx.Exec(`UPDATE map_cells SET last_cleared_at = NOW(), cleared_by = $1 WHERE region_id=$2 AND x = $3 AND y = $4`, playerID, pc.RegionID, x, y); err != nil {
			return err
		}
	}
	if !wasCleared {
		broadcastWorldCell(cellEventCleared, WorldCellEvent{RegionID: pc.RegionID, X: pc.X, Y: pc.Y, CellType: pc.Type, ContentID: pc.Content})
	}
	return nil
}

// MarkPersistentCellClearedNoTx clears a shared cell in memory only (used when
// no transaction is available). The DB state is updated lazily on the next
// MarkPersistentCellCleared call for the same cell.
func MarkPersistentCellClearedNoTx(x, y int) {
	_ = MarkPersistentCellCleared(nil, uuid.Nil, DeriveRegionAt(x, y), x, y)
}

// PersistentCellAt returns the shared cell at (x,y) if any. found=false means
// there is no shared cell; cleared=true means it exists but is inside its
// respawn window (treat as empty). When the respawn window has elapsed, the
// cell is lazily reset here (no background goroutine needed) and a respawn
// event is broadcast so map clients can refresh the marker.

func PersistentCellAt(regionID string, x, y int) (pc persistentCell, found, cleared bool) {
	pc, ok := persistentCellForRegion(regionID, x, y)
	if !ok {
		return pc, false, false
	}
	key := persistentCellKey{RegionID: pc.RegionID, X: x, Y: y}
	if pc.cleared && !pc.LastCleared.IsZero() && pc.RespawnSeconds > 0 &&
		time.Since(pc.LastCleared) >= time.Duration(pc.RespawnSeconds)*time.Second {
		zoneReg.mu.Lock()
		if cur, ok2 := zoneReg.persistent[key]; ok2 && cur.cleared {
			cur.cleared = false
			zoneReg.persistent[key] = cur
			pc = cur
		}
		zoneReg.mu.Unlock()
		broadcastWorldCell(cellEventRespawn, WorldCellEvent{RegionID: pc.RegionID, X: pc.X, Y: pc.Y, CellType: pc.Type, ContentID: pc.Content})
		return pc, true, false
	}
	return pc, true, pc.cleared
}

// eliteSpawnWeightFactor damps Elite templates so they stay rarer than Normal
// monsters now that both share the ambient procedural spawn pool.
const eliteSpawnWeightFactor = 0.2

// spawnPickWeight returns the weighted chance of a spawn row. Elite templates
// are damped by eliteSpawnWeightFactor; Ordinary (and Boss) rows use the raw
// configured chance.
func spawnPickWeight(s models.MonsterSpawn) float64 {
	if template, ok := models.MonsterRegistry[s.MonsterID]; ok && strings.EqualFold(template.Type, "Elite") {
		return s.Chance * eliteSpawnWeightFactor
	}
	return s.Chance
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
		totalWeight += spawnPickWeight(s)
	}
	if len(candidates) == 0 || totalWeight <= 0 {
		return models.MonsterSpawn{}, false
	}
	roll := rng.Float64() * totalWeight
	for _, s := range candidates {
		roll -= spawnPickWeight(s)
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
	region, x, y := player.CurrentRegion, player.X, player.Y
	if player.ActiveCombat.WorldBossCellRegionID != "" {
		region, x, y = player.ActiveCombat.WorldBossCellRegionID, player.ActiveCombat.WorldBossCellX, player.ActiveCombat.WorldBossCellY
	}
	_ = MarkPersistentCellCleared(tx, player.ID, region, x, y)
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
