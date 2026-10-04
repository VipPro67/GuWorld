package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	MonsterStateIdle     = "IDLE"
	MonsterStateChase    = "CHASE"
	MonsterStateAttack   = "ATTACK"
	MonsterStateReturn   = "RETURN"
	MonsterStateDefeated = "DEFEATED"

	worldMonsterTickInterval = 250 * time.Millisecond
	worldMonsterSafetyTTL    = 5 * time.Minute
	worldMonsterMoveEventGap = 400 * time.Millisecond
	// Monster AOI is deliberately independent from the 300-cell player
	// position broadcast index. The client investigation radius is capped at
	// 25 cells, so 32-cell buckets keep a view to at most 3x3 small buckets.
	WorldMonsterAOIBucket        = 32
	WorldMonsterActivationRadius = 25
	worldMonsterMaxBuckets       = 128
	worldMonsterMaxPerTick       = 512
	worldMonsterTickBudget       = 150 * time.Millisecond
)

type WorldMonster = cache.WorldMonster
type StatusEffect = cache.WorldStatusEffect

type WorldCombatEvent struct {
	EventID        string         `json:"event_id"`
	ServerTime     time.Time      `json:"server_time"`
	RegionID       string         `json:"region_id"`
	EntityID       string         `json:"entity_id"`
	EntityRevision int64          `json:"entity_revision"`
	EventType      string         `json:"event_type"`
	Position       WorldPosition  `json:"position"`
	Payload        map[string]any `json:"payload,omitempty"`
}

type WorldPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

var WorldCombatBroadcaster func(channel string, payload []byte, targets map[uuid.UUID]struct{})

var worldMonsterMetrics struct {
	processed            atomic.Uint64
	skipped              atomic.Uint64
	conflicts            atomic.Uint64
	duplicates           atomic.Uint64
	kills                atomic.Uint64
	preventedDoubleKills atomic.Uint64
	eventFanout          atomic.Uint64
}

var worldMonsterBucketCursor atomic.Uint64

var (
	worldMonstersInBucket   = cache.WorldMonstersInBucket
	materializeWorldMonster = cache.MaterializeWorldMonster
	casWorldMonster         = cache.CASWorldMonster
)

type monsterSpawnRow struct {
	ID, RegionID, MonsterID       string
	X, Y                          int
	RespawnSeconds, Rank, SubRank int
}

func monsterTTL(respawnSeconds int) time.Duration {
	if respawnSeconds < 1 {
		respawnSeconds = 300
	}
	return time.Duration(respawnSeconds)*time.Second + worldMonsterSafetyTTL
}

func materializeSpawn(ctx context.Context, spawn monsterSpawnRow, now time.Time) (WorldMonster, bool, error) {
	template, ok := models.MonsterRegistry[spawn.MonsterID]
	if !ok {
		return WorldMonster{}, false, fmt.Errorf("unknown monster template %q", spawn.MonsterID)
	}
	maxHP, attack, defense := models.ScaleMonsterStats(template.MaxHP, template.BaseAttack, template.BaseDefense, spawn.Rank, spawn.SubRank, template.Type)
	monster := WorldMonster{
		InstanceID: uuid.NewString(), SpawnID: spawn.ID, MonsterTemplateID: spawn.MonsterID, RegionID: spawn.RegionID,
		X: float64(spawn.X), Y: float64(spawn.Y), SpawnX: float64(spawn.X), SpawnY: float64(spawn.Y),
		HP: maxHP, MaxHP: maxHP, State: MonsterStateIdle, Revision: 1, LastUpdateAt: now,
		Statuses: []StatusEffect{}, RespawnSeconds: spawn.RespawnSeconds, AggroRadius: 12, AttackRange: 1.5,
		LeashRadius: 24, Speed: 3, AttackIntervalMS: 2000, BaseAttack: attack, BaseDefense: defense,
	}
	bx, by := monsterBucket(spawn.X, spawn.Y)
	return materializeWorldMonster(ctx, monster, bx, by, monsterTTL(spawn.RespawnSeconds))
}

func monsterBucket(x, y int) (int, int) {
	div := func(v int) int {
		if v < 0 {
			return (v - (WorldMonsterAOIBucket - 1)) / WorldMonsterAOIBucket
		}
		return v / WorldMonsterAOIBucket
	}
	return div(x), div(y)
}

func spawnWithinPlayerRadius(spawn monsterSpawnRow, player WorldPosEntry, radius int) bool {
	dx := int64(spawn.X) - int64(player.X)
	dy := int64(spawn.Y) - int64(player.Y)
	r := int64(radius)
	return dx*dx+dy*dy <= r*r
}

func activeMonsterBuckets(players []WorldPosEntry, radius, limit int) []ActiveWorldBucket {
	if radius < 0 || limit <= 0 {
		return nil
	}
	capacity := len(players) * 9
	if capacity > limit {
		capacity = limit
	}
	buckets := make([]ActiveWorldBucket, 0, capacity)
	seen := make(map[string]struct{}, capacity)
	for _, player := range players {
		minBX, minBY := monsterBucket(player.X-radius, player.Y-radius)
		maxBX, maxBY := monsterBucket(player.X+radius, player.Y+radius)
		for bx := minBX; bx <= maxBX && len(buckets) < limit; bx++ {
			for by := minBY; by <= maxBY && len(buckets) < limit; by++ {
				key := fmt.Sprintf("%s:%d:%d", player.Region, bx, by)
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				buckets = append(buckets, ActiveWorldBucket{BX: bx, BY: by, X: player.X, Y: player.Y, Region: player.Region})
			}
		}
	}
	return buckets
}

func rotatingMonsterBucketWindow(buckets []ActiveWorldBucket, limit int, cursor uint64) []ActiveWorldBucket {
	if limit <= 0 || len(buckets) == 0 {
		return nil
	}
	if len(buckets) <= limit {
		return buckets
	}
	out := make([]ActiveWorldBucket, 0, limit)
	start := int(cursor % uint64(len(buckets)))
	for i := 0; i < limit; i++ {
		out = append(out, buckets[(start+i)%len(buckets)])
	}
	return out
}

// GetWorldMonsters materializes only static spawns intersecting the requested
// AOI and returns their authoritative Redis state.
func GetWorldMonsters(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	entry, ok := worldPosIndex.Get(playerID.String())
	if !ok {
		EnsureWorldPosLoaded(playerID.String())
		entry, ok = worldPosIndex.Get(playerID.String())
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Player position unavailable"})
		return
	}
	radius, _ := strconv.Atoi(c.DefaultQuery("r", strconv.Itoa(WorldMonsterActivationRadius)))
	if radius < 1 {
		radius = 1
	}
	if radius > WorldMonsterActivationRadius {
		radius = WorldMonsterActivationRadius
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id::text, region_id, monster_id, x, y, respawn_seconds, min_rank
		FROM monster_spawns WHERE is_active=TRUE AND is_boss_spawn=FALSE AND x IS NOT NULL AND y IS NOT NULL
		AND region_id=$1 AND x BETWEEN $2 AND $3 AND y BETWEEN $4 AND $5 ORDER BY id`, entry.Region, entry.X-radius, entry.X+radius, entry.Y-radius, entry.Y+radius)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot load monster spawns"})
		return
	}
	defer rows.Close()
	monsters := make([]WorldMonster, 0, 32)
	now := time.Now().UTC()
	for rows.Next() {
		var spawn monsterSpawnRow
		if err := rows.Scan(&spawn.ID, &spawn.RegionID, &spawn.MonsterID, &spawn.X, &spawn.Y, &spawn.RespawnSeconds, &spawn.Rank); err != nil {
			continue
		}
		if !spawnWithinPlayerRadius(spawn, entry, radius) {
			continue
		}
		if spawn.Rank < 1 {
			spawn.Rank = 1
		}
		spawn.SubRank = 1
		monster, created, err := materializeSpawn(c.Request.Context(), spawn, now)
		if err != nil {
			logger.S.Warnf("materialize monster spawn %s: %v", spawn.ID, err)
			continue
		}
		if monster.State != MonsterStateDefeated {
			monsters = append(monsters, monster)
		}
		if created {
			broadcastWorldCombat(monster, "monster_spawned", nil)
		}
	}
	c.JSON(http.StatusOK, gin.H{"monsters": monsters, "server_time": now})
}

func broadcastWorldCombat(monster WorldMonster, eventType string, payload map[string]any) {
	if WorldCombatBroadcaster == nil {
		return
	}
	authoritative := make(map[string]any, len(payload)+7)
	for key, value := range payload {
		authoritative[key] = value
	}
	authoritative["entity"] = monster
	authoritative["state"] = monster.State
	authoritative["target_id"] = monster.TargetID
	authoritative["x"] = monster.X
	authoritative["y"] = monster.Y
	authoritative["hp"] = monster.HP
	authoritative["max_hp"] = monster.MaxHP
	event := WorldCombatEvent{EventID: uuid.NewString(), ServerTime: time.Now().UTC(), RegionID: monster.RegionID, EntityID: monster.InstanceID, EntityRevision: monster.Revision, EventType: eventType, Position: WorldPosition{monster.X, monster.Y}, Payload: authoritative}
	targets := map[uuid.UUID]struct{}{}
	for _, p := range worldPosIndex.Nearby(int(monster.X), int(monster.Y), WorldMonsterActivationRadius, "") {
		if p.Region != monster.RegionID {
			continue
		}
		if id, err := uuid.Parse(p.ID); err == nil {
			targets[id] = struct{}{}
		}
	}
	worldMonsterMetrics.eventFanout.Add(uint64(len(targets)))
	WorldCombatBroadcaster("world:combat", mustMarshal(event), targets)
}

type monsterTickTarget struct {
	ID   string
	X, Y float64
}

func selectMonsterTarget(monster WorldMonster, candidates []WorldPosEntry, active func(string) bool) (monsterTickTarget, bool) {
	type ranked struct {
		monsterTickTarget
		d float64
	}
	valid := make([]ranked, 0, len(candidates))
	for _, p := range candidates {
		if p.Region != monster.RegionID || !active(p.ID) || IsSafeZone(p.Region, p.X, p.Y) {
			continue
		}
		d := math.Hypot(float64(p.X)-monster.X, float64(p.Y)-monster.Y)
		if d > monster.AggroRadius {
			continue
		}
		valid = append(valid, ranked{monsterTickTarget{p.ID, float64(p.X), float64(p.Y)}, d})
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].d == valid[j].d {
			return valid[i].ID < valid[j].ID
		}
		return valid[i].d < valid[j].d
	})
	if len(valid) == 0 {
		return monsterTickTarget{}, false
	}
	return valid[0].monsterTickTarget, true
}

func tickWorldMonster(monster WorldMonster, candidates []WorldPosEntry, now time.Time, active func(string) bool) (WorldMonster, string) {
	if monster.State == MonsterStateDefeated {
		return monster, ""
	}
	dt := now.Sub(monster.LastUpdateAt).Seconds()
	if dt < 0 {
		dt = 0
	}
	if dt > 5 {
		dt = 5
	}
	statusCount := len(monster.Statuses)
	monster.Statuses = expireMonsterStatuses(monster.Statuses, now)
	oldState := monster.State
	oldX, oldY := monster.X, monster.Y
	target, found := selectMonsterTarget(monster, candidates, active)
	if monster.State == MonsterStateReturn {
		moveMonsterToward(&monster, monster.SpawnX, monster.SpawnY, dt)
		if math.Hypot(monster.X-monster.SpawnX, monster.Y-monster.SpawnY) < 0.1 {
			monster.X = monster.SpawnX
			monster.Y = monster.SpawnY
			monster.State = MonsterStateIdle
			monster.HP = monster.MaxHP
		}
	} else if !found || math.Hypot(target.X-monster.SpawnX, target.Y-monster.SpawnY) > monster.LeashRadius {
		if monster.State != MonsterStateIdle {
			monster.State = MonsterStateReturn
			monster.TargetID = ""
		}
	} else {
		monster.TargetID = target.ID
		distance := math.Hypot(target.X-monster.X, target.Y-monster.Y)
		if distance <= monster.AttackRange {
			monster.State = MonsterStateAttack
		} else {
			monster.State = MonsterStateChase
			moveMonsterToward(&monster, target.X, target.Y, dt)
		}
	}
	monster.LastUpdateAt = now
	changed := oldState != monster.State || oldX != monster.X || oldY != monster.Y || len(monster.Statuses) != statusCount || dt > 0
	if changed {
		monster.Revision++
	}
	if oldState != monster.State {
		return monster, map[string]string{MonsterStateChase: "monster_targeted", MonsterStateAttack: "action_started", MonsterStateReturn: "monster_returned", MonsterStateIdle: "monster_returned"}[monster.State]
	}
	if oldX != monster.X || oldY != monster.Y {
		return monster, "monster_moved"
	}
	return monster, ""
}

func moveMonsterToward(monster *WorldMonster, tx, ty, dt float64) {
	dx, dy := tx-monster.X, ty-monster.Y
	distance := math.Hypot(dx, dy)
	if distance == 0 {
		return
	}
	step := math.Min(distance, monster.Speed*dt)
	nx, ny := monster.X+dx/distance*step, monster.Y+dy/distance*step
	if inWorld(int(math.Round(nx)), int(math.Round(ny))) && !IsSafeZone(monster.RegionID, int(math.Round(nx)), int(math.Round(ny))) && math.Hypot(nx-monster.SpawnX, ny-monster.SpawnY) <= monster.LeashRadius {
		monster.X, monster.Y = nx, ny
	}
}

func expireMonsterStatuses(statuses []StatusEffect, now time.Time) []StatusEffect {
	out := statuses[:0]
	for _, s := range statuses {
		if s.ExpiresAt.IsZero() || s.ExpiresAt.After(now) {
			out = append(out, s)
		}
	}
	return out
}

// StartWorldMonsterWorker runs one bounded ticker for all ordinary monsters.
func StartWorldMonsterWorker(ctx context.Context) {
	startWorldMonsterWorker(ctx, worldMonsterTickInterval)
}

func startWorldMonsterWorker(ctx context.Context, interval time.Duration) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				processWorldMonsterTick(ctx, now.UTC())
			}
		}
	}()
	return done
}

func processWorldMonsterTick(ctx context.Context, now time.Time) {
	deadline := time.Now().Add(worldMonsterTickBudget)
	online := worldPosIndex.OnlineEntries(-1, func(entry WorldPosEntry) bool {
		playerID, err := uuid.Parse(entry.ID)
		return err == nil && isWSConnected(playerID)
	})
	playersByRegion := make(map[string][]WorldPosEntry)
	for _, player := range online {
		playersByRegion[player.Region] = append(playersByRegion[player.Region], player)
	}
	regions := make([]string, 0, len(playersByRegion))
	for region := range playersByRegion {
		regions = append(regions, region)
	}
	sort.Strings(regions)
	allBuckets := make([]ActiveWorldBucket, 0, len(online)*9)
	for _, region := range regions {
		allBuckets = append(allBuckets, activeMonsterBuckets(playersByRegion[region], WorldMonsterActivationRadius, int(^uint(0)>>1))...)
	}
	cursor := worldMonsterBucketCursor.Add(uint64(worldMonsterMaxBuckets)) - uint64(worldMonsterMaxBuckets)
	buckets := rotatingMonsterBucketWindow(allBuckets, worldMonsterMaxBuckets, cursor)
	if len(allBuckets) > len(buckets) {
		worldMonsterMetrics.skipped.Add(uint64(len(allBuckets) - len(buckets)))
	}
	processed := 0
	for _, b := range buckets {
		if processed >= worldMonsterMaxPerTick || time.Now().After(deadline) {
			worldMonsterMetrics.skipped.Add(1)
			break
		}
		monsters, err := worldMonstersInBucket(ctx, b.Region, b.BX, b.BY)
		if err != nil {
			continue
		}
		bucketCenterX := b.BX*WorldMonsterAOIBucket + WorldMonsterAOIBucket/2
		bucketCenterY := b.BY*WorldMonsterAOIBucket + WorldMonsterAOIBucket/2
		candidates := worldPosIndex.Nearby(bucketCenterX, bucketCenterY, WorldMonsterAOIBucket/2+12, "")
		for monsterIndex, monster := range monsters {
			if processed >= worldMonsterMaxPerTick || time.Now().After(deadline) {
				worldMonsterMetrics.skipped.Add(uint64(len(monsters) - monsterIndex))
				break
			}
			processed++
			if monster.State == MonsterStateDefeated && !monster.RespawnAt.IsZero() && !now.Before(monster.RespawnAt) {
				replacement := monster
				replacement.InstanceID = uuid.NewString()
				replacement.HP = replacement.MaxHP
				replacement.State = MonsterStateIdle
				replacement.TargetID = ""
				replacement.Revision = 1
				replacement.Statuses = []StatusEffect{}
				replacement.RespawnAt = time.Time{}
				replacement.LastHitAt = time.Time{}
				replacement.LastSeenAt = time.Time{}
				replacement.LastUpdateAt = now
				replacement.X, replacement.Y = replacement.SpawnX, replacement.SpawnY
				bx, by := monsterBucket(int(replacement.X), int(replacement.Y))
				oldBX, oldBY := monsterBucket(int(monster.X), int(monster.Y))
				if ok, _ := cache.RespawnWorldMonster(ctx, monster, replacement, oldBX, oldBY, bx, by, monsterTTL(monster.RespawnSeconds)); ok {
					broadcastWorldCombat(replacement, "monster_respawned", map[string]any{"previous_instance_id": monster.InstanceID})
				}
				continue
			}
			next, eventType := tickWorldMonster(monster, candidates, now, func(id string) bool { parsed, e := uuid.Parse(id); return e == nil && isWSConnected(parsed) })
			if eventType == "monster_moved" {
				if !monster.LastSeenAt.IsZero() && now.Sub(monster.LastSeenAt) < worldMonsterMoveEventGap {
					eventType = ""
				} else {
					next.LastSeenAt = now
				}
			}
			if next.Revision == monster.Revision {
				continue
			}
			bx, by := monsterBucket(int(next.X), int(next.Y))
			oldBX, oldBY := monsterBucket(int(monster.X), int(monster.Y))
			ok, err := casWorldMonster(ctx, monster, next, monster.Revision, oldBX, oldBY, bx, by, monsterTTL(monster.RespawnSeconds))
			if err != nil || !ok {
				worldMonsterMetrics.conflicts.Add(1)
				continue
			}
			if eventType != "" {
				broadcastWorldCombat(next, eventType, nil)
			}
		}
	}
	worldMonsterMetrics.processed.Add(uint64(processed))
}
