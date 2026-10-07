package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
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
	worldMonsterDefaultSpeed = 0.25
	// Monster AOI is deliberately independent from the 300-cell player
	// position broadcast index. The client investigation radius is capped at
	// 25 cells, so 32-cell buckets keep a view to at most 3x3 small buckets.
	WorldMonsterAOIBucket         = 32
	WorldMonsterActivationRadius  = 25
	worldMonsterMaxBuckets        = 128
	worldMonsterMaxPerTick        = 512
	worldMonsterTickBudget        = 150 * time.Millisecond
	worldMonsterBucketQueryRadius = 35
)

type WorldMonster = cache.WorldMonster
type StatusEffect = cache.WorldStatusEffect

type WorldCombatEvent struct {
	EventID        string         `json:"event_id"`
	ServerTime     time.Time      `json:"server_time"`
	RegionID       string         `json:"region_id"`
	MapID          string         `json:"world_map_id"`
	MapInstanceID  string         `json:"map_instance_id,omitempty"`
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
	worldMonstersInBucket     = cache.WorldMonstersInBucket
	materializeWorldMonster   = cache.MaterializeWorldMonster
	casWorldMonster           = cache.CASWorldMonster
	loadWorldCombatPlayers    = loadWorldCombatPlayerSnapshots
	applyWorldMonsterAction   = applyWorldMonsterAttack
	applyWorldPlayerCounter   = applyWorldCounterattack
	resolveWorldMonsterCell   = ResolveCell
	deriveWorldMonsterRegion  = DeriveRegionAt
	isWorldMonsterSafeZone    = IsSafeZone
	worldMonsterCellCleared   = func(region string, x, y int) bool { return isCellClearedRecently(region, x, y, clearedCellCooldown) }
	loadWorldMonsterSpawnRank = func(ctx context.Context, spawnID string) (int, error) {
		var rank int
		err := db.DB.QueryRowContext(ctx, `SELECT min_rank FROM monster_spawns WHERE id::text=$1`, spawnID).Scan(&rank)
		return rank, err
	}
)

type worldCombatPlayer struct {
	ID       uuid.UUID
	HP       float64
	Region   string
	X, Y     int
	Dead     bool
	Race     string
	Lifespan float64
	Defense  float64
	Attack   float64
	Statuses models.StatusList
}

// deadInCoiChet mirrors gamefx.IsDeadState for rows loaded by raw SQL instead of
// the player service: Cõi Chết is derived from lifespan + race, never stored.
func (p worldCombatPlayer) deadInCoiChet() bool {
	probe := models.Player{}
	probe.Race = p.Race
	probe.Lifespan = p.Lifespan
	return gamefx.IsDeadState(&probe)
}

type worldMonsterActionResult struct {
	Applied, Died, Rejected bool
	Damage, HP, Shield      float64
}

func loadWorldCombatPlayerSnapshots(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]worldCombatPlayer, error) {
	out := make(map[uuid.UUID]worldCombatPlayer, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := db.DB.QueryContext(ctx, `
		SELECT p.id, p.hp, p.current_region, p.x, p.y, p.lifespan, COALESCE(p.race, ''), p.rank, p.sub_rank,
		       p.statuses, p.growth_data, p.path_masteries,
		       COALESCE(ia.dao_marks, '{}'::jsonb), `+db.GuInventoryAggregate("p.id")+`
		FROM players p
		LEFT JOIN immortal_apertures ia ON ia.player_id = p.id
		WHERE p.id = ANY($1::uuid[])`, pq.Array(func() []string {
		out := make([]string, len(ids))
		for i := range ids {
			out[i] = ids[i].String()
		}
		return out
	}()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p models.Player
		var snap worldCombatPlayer
		if err := rows.Scan(&snap.ID, &snap.HP, &snap.Region, &snap.X, &snap.Y, &snap.Lifespan, &snap.Race,
			&p.Rank, &p.SubRank, &p.Statuses, &p.GrowthData, &p.PathMasteries, &p.DaoMarks, &p.GuInventory); err != nil {
			return nil, err
		}
		snap.Dead = snap.deadInCoiChet()
		p.ID, p.HP, p.CurrentRegion, p.X, p.Y = snap.ID, snap.HP, snap.Region, snap.X, snap.Y
		snap.Statuses = p.Statuses
		snap.Defense = calculateWorldPlayerDefense(&p)
		snap.Attack = calculateWorldPlayerAttack(&p)
		out[snap.ID] = snap
	}
	return out, rows.Err()
}

// calculateWorldPlayerAttack mirrors the query-free portions of passive stat
// calculation. World ticks batch-load all required data and must not issue a
// territory query for every nearby player.
func calculateWorldPlayerAttack(p *models.Player) float64 {
	attack := 10 + float64(p.Rank-1)*10 + float64(p.SubRank-1)*2 + p.GrowthData.Boosts.Attack
	if p.DaoMarks != nil {
		attack += float64(p.DaoMarks["strength"]) / 10
	}
	best := map[string]models.GuInstance{}
	for _, gu := range p.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if old, ok := best[gu.GuID]; !ok || gu.Rank > old.Rank {
				best[gu.GuID] = gu
			}
		}
	}
	allPct := 0.0
	for _, gu := range best {
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			continue
		}
		mastery, dao := gamefx.GuStatMultipliers(p.PathMasteries, p.DaoMarks, spec.Path)
		attack += models.GuStatValue(spec, "ATTACK") * mastery * dao
		if compiled := models.CompiledGuEffects(spec); compiled != nil {
			allPct += compiled.StatValue(gameplayeffects.TriggerOnAction, "ALL", "MULTIPLY_PERCENT")
		}
	}
	for _, st := range p.Statuses.Active() {
		if st.ID == "buff_atk" {
			attack += st.Value
		}
	}
	return attack * (1 + allPct) * statusStatFactor(p, "attack")
}

// calculateWorldPlayerDefense is the query-free subset of the simulator stat
// builder needed by a world attack. All inputs are supplied by the batch load.
func calculateWorldPlayerDefense(p *models.Player) float64 {
	defense := 5 + float64(p.Rank-1)*5 + float64(p.SubRank-1) + p.GrowthData.Boosts.Defense
	best := map[string]models.GuInstance{}
	for _, gu := range p.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if old, ok := best[gu.GuID]; !ok || gu.Rank > old.Rank {
				best[gu.GuID] = gu
			}
		}
	}
	allPct := 0.0
	for _, gu := range best {
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			continue
		}
		mastery, dao := gamefx.GuStatMultipliers(p.PathMasteries, p.DaoMarks, spec.Path)
		defense += models.GuStatValue(spec, "DEFENSE") * mastery * dao
		if compiled := models.CompiledGuEffects(spec); compiled != nil {
			allPct += compiled.StatValue(gameplayeffects.TriggerOnAction, "ALL", "MULTIPLY_PERCENT")
		}
	}
	for _, st := range p.Statuses.Active() {
		if st.ID == "buff_def" {
			defense += st.Value
		}
	}
	return defense * (1 + allPct) * statusStatFactor(p, "defense")
}

// worldStatusDefense returns the shield, the fractional damage reduction, and
// whether Bất Tử (LAST_STAND) is active — overworld combat does not run the
// turn-based simulator, so the HP floor has to be handed to ResolveBasicDamage
// as minHP.
func worldStatusDefense(statuses models.StatusList) (shield, reduction float64, lastStand bool) {
	for _, st := range statuses.Active() {
		if st.ID == "virtual_shield" {
			shield += math.Max(0, st.Value)
		}
		switch {
		case strings.EqualFold(st.ID, statusfx.StatusInvincible), strings.EqualFold(st.ID, "invulnerable"):
			reduction = 1
		case strings.EqualFold(st.ID, statusfx.StatusLastStand):
			lastStand = true
		case strings.EqualFold(st.ID, statusfx.StatusDamageReduction):
			value := st.Value
			if value > 1 {
				value /= 100
			}
			if value > reduction {
				reduction = value
			}
		}
	}
	return
}

func setWorldVirtualShield(statuses *models.StatusList, value float64) {
	out := (*statuses)[:0]
	for _, st := range *statuses {
		if st.ID != "virtual_shield" {
			out = append(out, st)
			continue
		}
		if value > 0 {
			st.Value, value = value, 0
			out = append(out, st)
		}
	}
	*statuses = out
}

func applyWorldMonsterAttack(ctx context.Context, monster WorldMonster, target worldCombatPlayer, actionToken string) (worldMonsterActionResult, error) {
	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return worldMonsterActionResult{}, err
	}
	defer tx.Rollback()
	var claimed string
	if err := tx.QueryRowContext(ctx, `INSERT INTO world_monster_actions(action_token, monster_id, player_id)
		VALUES($1,$2,$3) ON CONFLICT DO NOTHING RETURNING action_token`, actionToken, monster.InstanceID, target.ID).Scan(&claimed); err != nil {
		if err == sql.ErrNoRows {
			var replay worldMonsterActionResult
			err = tx.QueryRowContext(ctx, `SELECT applied,died,damage,remaining_hp,remaining_shield FROM world_monster_actions WHERE action_token=$1`, actionToken).
				Scan(&replay.Applied, &replay.Died, &replay.Damage, &replay.HP, &replay.Shield)
			return replay, err
		}
		return worldMonsterActionResult{}, err
	}
	var hp float64
	var region string
	var x, y int
	var lifespan float64
	var race string
	var statuses models.StatusList
	if err := tx.QueryRowContext(ctx, `SELECT hp,current_region,x,y,lifespan,COALESCE(race,''),statuses FROM players WHERE id=$1 FOR UPDATE`, target.ID).
		Scan(&hp, &region, &x, &y, &lifespan, &race, &statuses); err != nil {
		return worldMonsterActionResult{}, err
	}
	dead := worldCombatPlayer{Race: race, Lifespan: lifespan}.deadInCoiChet()
	if dead || hp <= 0 || region != monster.RegionID || x != target.X || y != target.Y || isWorldMonsterSafeZone(region, x, y) || math.Hypot(float64(x)-monster.X, float64(y)-monster.Y) > monster.AttackRange {
		return worldMonsterActionResult{Rejected: true}, nil
	}
	statuses.Expire(time.Now())
	shield, reduction, lastStand := worldStatusDefense(statuses)
	minHP := 0.0
	if lastStand {
		minHP = simulator.LastStandMinHP
	}
	resolved := simulator.ResolveBasicDamage(monster.BaseAttack, target.Defense, reduction, hp, shield, minHP)
	setWorldVirtualShield(&statuses, resolved.Shield)
	statusJSON, err := json.Marshal(statuses)
	if err != nil {
		return worldMonsterActionResult{}, err
	}
	died := hp > 0 && resolved.HP <= 0
	if _, err = tx.ExecContext(ctx, `UPDATE players SET hp=$1,statuses=$2,version=version+1 WHERE id=$3`, resolved.HP, statusJSON, target.ID); err != nil {
		return worldMonsterActionResult{}, err
	}
	if died {
		killerName := monster.MonsterTemplateID
		if template, ok := models.MonsterRegistry[monster.MonsterTemplateID]; ok && template.Name != "" {
			killerName = template.Name
		}
		applyDeathDropRecord(tx, &target.ID, nil, false, DeathPos{Region: region, X: x, Y: y, KillerName: killerName})
	}
	result := worldMonsterActionResult{Applied: true, Died: died, Damage: resolved.HPDamage + resolved.ShieldAbsorbed, HP: resolved.HP, Shield: resolved.Shield}
	if _, err = tx.ExecContext(ctx, `UPDATE world_monster_actions SET applied=$2,died=$3,damage=$4,remaining_hp=$5,remaining_shield=$6 WHERE action_token=$1`, actionToken, result.Applied, result.Died, result.Damage, result.HP, result.Shield); err != nil {
		return worldMonsterActionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return worldMonsterActionResult{}, err
	}
	return result, nil
}

func applyWorldCounterattack(ctx context.Context, monster WorldMonster, target worldCombatPlayer, commandID string, now time.Time) (cache.WorldMonsterHitResult, WorldMonster, float64, error) {
	damage := simulator.ResolveBaseDamage(math.Max(1, target.Attack), monster.BaseDefense)
	outcome, updated, err := cache.HitWorldMonster(ctx, monster.RegionID, monster.InstanceID, commandID, monster.Revision, damage, now,
		time.Duration(monster.RespawnSeconds)*time.Second, monsterTTL(monster.RespawnSeconds))
	return outcome, updated, damage, err
}

type monsterSpawnRow struct {
	ID, RegionID, MonsterID       string
	X, Y                          int
	RespawnSeconds, Rank, SubRank int
	IsBoss                        bool
}

func monsterTTL(respawnSeconds int) time.Duration {
	if respawnSeconds < 1 {
		respawnSeconds = 300
	}
	return time.Duration(respawnSeconds)*time.Second + worldMonsterSafetyTTL
}

func materializeSpawn(ctx context.Context, spawn monsterSpawnRow, now time.Time) (WorldMonster, bool, error) {
	if isWorldMonsterSafeZone(spawn.RegionID, spawn.X, spawn.Y) {
		return WorldMonster{}, false, fmt.Errorf("monster spawn %q is inside a safe zone", spawn.ID)
	}
	template, ok := models.MonsterRegistry[spawn.MonsterID]
	if !ok {
		return WorldMonster{}, false, fmt.Errorf("unknown monster template %q", spawn.MonsterID)
	}
	maxHP, attack, defense := models.ScaleMonsterStats(template.MaxHP, template.BaseAttack, template.BaseDefense, spawn.Rank, spawn.SubRank, template.Type)
	aggroRadius, leashRadius := worldMonsterAggroProfile(spawn.Rank)
	attackRange, speed := 1.5, worldMonsterDefaultSpeed
	patrolMinX, patrolMinY, patrolMaxX, patrolMaxY := 0, 0, 0, 0
	if spawn.IsBoss {
		// Bosses can pursue players throughout their owning region, but remain
		// confined to that region's rectangle.
		patrolMinX, patrolMinY, patrolMaxX, patrolMaxY = monsterPatrolBounds(spawn.RegionID)
		leashRadius = patrolRadius(spawn.X, spawn.Y, patrolMinX, patrolMinY, patrolMaxX, patrolMaxY)
		aggroRadius = leashRadius
		attackRange, speed = 2, 0.4
	}
	monster := WorldMonster{
		InstanceID: uuid.NewString(), SpawnID: spawn.ID, MonsterTemplateID: spawn.MonsterID, IsBoss: spawn.IsBoss, Rank: spawn.Rank, SubRank: spawn.SubRank, RegionID: spawn.RegionID,
		X: float64(spawn.X), Y: float64(spawn.Y), SpawnX: float64(spawn.X), SpawnY: float64(spawn.Y),
		HP: maxHP, MaxHP: maxHP, State: MonsterStateIdle, Revision: 1, LastUpdateAt: now,
		Statuses: []StatusEffect{}, RespawnSeconds: spawn.RespawnSeconds, AggroRadius: aggroRadius, AttackRange: attackRange,
		LeashRadius: leashRadius, Speed: speed, AttackIntervalMS: 2000, BaseAttack: attack, BaseDefense: defense,
		PatrolMinX: patrolMinX, PatrolMinY: patrolMinY, PatrolMaxX: patrolMaxX, PatrolMaxY: patrolMaxY,
	}
	bx, by := monsterBucket(spawn.X, spawn.Y)
	return materializeWorldMonster(ctx, monster, bx, by, monsterTTL(spawn.RespawnSeconds))
}

// worldMonsterAggroProfile keeps ordinary monsters passive until a player is
// very close. Higher-rank monsters notice players slightly farther away, but
// never regain the old 12-cell aggro / 24-cell chase range.
func worldMonsterAggroProfile(rank int) (aggroRadius, leashRadius float64) {
	if rank < 1 {
		rank = 1
	}
	if rank > 9 {
		rank = 9
	}
	aggroRadius = 2 + float64(rank-1)*0.5
	leashRadius = aggroRadius + 3
	return aggroRadius, leashRadius
}

func applyWorldMonsterTuning(ctx context.Context, monster *WorldMonster) {
	if monster.Rank < 1 {
		cell := resolveWorldMonsterCell(int(math.Round(monster.SpawnX)), int(math.Round(monster.SpawnY)))
		if cell.Content == monster.MonsterTemplateID && cell.Rank > 0 {
			monster.Rank, monster.SubRank = cell.Rank, cell.SubRank
		} else if rank, err := loadWorldMonsterSpawnRank(ctx, monster.SpawnID); err == nil && rank > 0 {
			monster.Rank = rank
		}
	}
	effectiveRank := monster.Rank
	if effectiveRank < 1 {
		effectiveRank = 1
	}
	if monster.IsBoss {
		monster.PatrolMinX, monster.PatrolMinY, monster.PatrolMaxX, monster.PatrolMaxY = monsterPatrolBounds(monster.RegionID)
		monster.LeashRadius = patrolRadius(int(math.Round(monster.SpawnX)), int(math.Round(monster.SpawnY)), monster.PatrolMinX, monster.PatrolMinY, monster.PatrolMaxX, monster.PatrolMaxY)
		monster.AggroRadius, monster.Speed = monster.LeashRadius, 0.4
		monster.AttackRange = 2
	} else {
		monster.AggroRadius, monster.LeashRadius = worldMonsterAggroProfile(effectiveRank)
		monster.Speed = worldMonsterDefaultSpeed
	}
}

func monsterPatrolBounds(regionID string) (minX, minY, maxX, maxY int) {
	if region := ZoneByID(regionID); region != nil && region.MaxX >= region.MinX && region.MaxY >= region.MinY {
		return region.MinX, region.MinY, region.MaxX, region.MaxY
	}
	return 0, 0, WorldWidth - 1, WorldHeight - 1
}

func patrolRadius(spawnX, spawnY, minX, minY, maxX, maxY int) float64 {
	maxDistance := 0.0
	for _, x := range []int{minX, maxX} {
		for _, y := range []int{minY, maxY} {
			maxDistance = math.Max(maxDistance, math.Hypot(float64(x-spawnX), float64(y-spawnY)))
		}
	}
	return maxDistance
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

func proceduralMonsterSpawns(player WorldPosEntry, radius int) []monsterSpawnRow {
	spawns := make([]monsterSpawnRow, 0, 64)
	minX, maxX := max(0, player.X-radius), min(WorldWidth-1, player.X+radius)
	minY, maxY := max(0, player.Y-radius), min(WorldHeight-1, player.Y+radius)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx, dy := int64(x-player.X), int64(y-player.Y)
			if dx*dx+dy*dy > int64(radius)*int64(radius) {
				continue
			}
			cell := resolveWorldMonsterCell(x, y)
			if cell.Type != "Monster" || cell.Content == "" || deriveWorldMonsterRegion(x, y) != player.Region || worldMonsterCellCleared(player.Region, x, y) {
				continue
			}
			spawns = append(spawns, monsterSpawnRow{
				ID: fmt.Sprintf("procedural:%s:%d:%d", player.Region, x, y), RegionID: player.Region,
				MonsterID: cell.Content, X: x, Y: y, RespawnSeconds: int(clearedCellCooldown.Seconds()), Rank: cell.Rank, SubRank: cell.SubRank,
			})
		}
	}
	return spawns
}

func proceduralMonsterCell(spawnID string) (region string, x, y int, ok bool) {
	const prefix = "procedural:"
	if !strings.HasPrefix(spawnID, prefix) {
		return "", 0, 0, false
	}
	parts := strings.Split(strings.TrimPrefix(spawnID, prefix), ":")
	if len(parts) != 3 {
		return "", 0, 0, false
	}
	x, errX := strconv.Atoi(parts[1])
	y, errY := strconv.Atoi(parts[2])
	if errX != nil || errY != nil {
		return "", 0, 0, false
	}
	return parts[0], x, y, true
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
		mapID := player.MapID
		if mapID == "" {
			mapID = worldMapIDForRegion(player.Region)
		}
		minBX, minBY := monsterBucket(player.X-radius, player.Y-radius)
		maxBX, maxBY := monsterBucket(player.X+radius, player.Y+radius)
		for _, scopeRegion := range regionScopeIDs(player.Region) {
			for bx := minBX; bx <= maxBX && len(buckets) < limit; bx++ {
				for by := minBY; by <= maxBY && len(buckets) < limit; by++ {
					key := fmt.Sprintf("%s:%s:%s:%d:%d", mapID, player.InstanceID, scopeRegion, bx, by)
					if _, exists := seen[key]; exists {
						continue
					}
					seen[key] = struct{}{}
					buckets = append(buckets, ActiveWorldBucket{BX: bx, BY: by, X: player.X, Y: player.Y, Region: scopeRegion, MapID: mapID, InstanceID: player.InstanceID})
				}
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
	if entry.MapID != "world_map" || entry.InstanceID != "" {
		c.JSON(http.StatusOK, gin.H{"monsters": []WorldMonster{}, "server_time": time.Now().UTC()})
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
		AND region_id = ANY($1) AND x BETWEEN $2 AND $3 AND y BETWEEN $4 AND $5 ORDER BY id`, pq.Array(regionScopeIDs(entry.Region)), entry.X-radius, entry.X+radius, entry.Y-radius, entry.Y+radius)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot load monster spawns"})
		return
	}
	defer rows.Close()
	monsters := make([]WorldMonster, 0, 32)
	occupied := make(map[[2]int]struct{}, 32)
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
		occupied[[2]int{spawn.X, spawn.Y}] = struct{}{}
		if monster.State != MonsterStateDefeated {
			monsters = append(monsters, monster)
		}
		if created {
			broadcastWorldCombat(monster, "monster_spawned", nil)
		}
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot read monster spawns"})
		return
	}
	_ = rows.Close()
	// Persistent Boss map_cells are the source of truth for shared boss spawns.
	// They are materialized as normal world actors, so the same AI can chase,
	// attack, be defeated, and respawn them for every player in that region.
	scopeRegions := regionScopeIDs(entry.Region)
	bossRows, bossErr := db.DB.QueryContext(c.Request.Context(), `
		SELECT region_id, x, y, COALESCE(content_id,''),
		       CASE WHEN COALESCE(metadata->>'rank','') ~ '^[0-9]+$' THEN (metadata->>'rank')::int ELSE 1 END,
		       CASE WHEN COALESCE(metadata->>'sub_rank','') ~ '^[0-9]+$' THEN (metadata->>'sub_rank')::int ELSE 1 END,
		       COALESCE(respawn_seconds,300)
		FROM map_cells WHERE is_active=TRUE AND lower(cell_type)='boss' AND region_id = ANY($1)
		ORDER BY region_id,x,y`, pq.Array(scopeRegions))
	if bossErr != nil {
		logger.S.Errorf("load map-cell bosses in region %s: %v", entry.Region, bossErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot load map-cell bosses"})
		return
	}
	defer bossRows.Close()
	bossSpawnIDs := make(map[string]map[string]struct{}, len(scopeRegions))
	for _, region := range scopeRegions {
		bossSpawnIDs[region] = make(map[string]struct{})
	}
	for bossRows.Next() {
		var spawn monsterSpawnRow
		spawn.IsBoss = true
		if err := bossRows.Scan(&spawn.RegionID, &spawn.X, &spawn.Y, &spawn.MonsterID, &spawn.Rank, &spawn.SubRank, &spawn.RespawnSeconds); err != nil {
			continue
		}
		if spawn.MonsterID == "" {
			continue
		}
		spawn.ID = fmt.Sprintf("mapcell:%s:%d:%d", spawn.RegionID, spawn.X, spawn.Y)
		bossSpawnIDs[spawn.RegionID][spawn.ID] = struct{}{}
		if _, found, cleared := PersistentCellAt(spawn.RegionID, spawn.X, spawn.Y); found && cleared {
			continue
		}
		if spawn.Rank < 1 {
			spawn.Rank = 1
		}
		if spawn.SubRank < 1 {
			spawn.SubRank = 1
		}
		monster, created, err := materializeSpawn(c.Request.Context(), spawn, now)
		if err != nil {
			logger.S.Warnf("materialize map-cell boss %s: %v", spawn.ID, err)
			continue
		}
		if spawnWithinPlayerRadius(spawn, entry, radius) {
			occupied[[2]int{spawn.X, spawn.Y}] = struct{}{}
		}
		if monster.State != MonsterStateDefeated && math.Hypot(monster.X-float64(entry.X), monster.Y-float64(entry.Y)) <= float64(radius) {
			monsters = append(monsters, monster)
		}
		if created && math.Hypot(monster.X-float64(entry.X), monster.Y-float64(entry.Y)) <= float64(radius) {
			broadcastWorldCombat(monster, "monster_spawned", map[string]any{"boss": true})
		}
	}
	if err := bossRows.Err(); err != nil {
		logger.S.Errorf("read map-cell bosses in region %s: %v", entry.Region, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot read map-cell bosses"})
		return
	}
	for _, region := range scopeRegions {
		if err := cache.PruneWorldBosses(c.Request.Context(), region, bossSpawnIDs[region]); err != nil {
			logger.S.Warnf("prune stale map-cell bosses in region %s: %v", region, err)
		}
	}
	// Actors may have chased far from their persistent spawn coordinate. Read
	// their current AOI buckets as well, so polling/reconnect still sees them.
	minBX, minBY := monsterBucket(entry.X-radius, entry.Y-radius)
	maxBX, maxBY := monsterBucket(entry.X+radius, entry.Y+radius)
	seenActors := make(map[string]struct{}, len(monsters))
	for _, monster := range monsters {
		seenActors[monster.InstanceID] = struct{}{}
	}
	for _, scopeRegion := range regionScopeIDs(entry.Region) {
		for bx := minBX; bx <= maxBX; bx++ {
			for by := minBY; by <= maxBY; by++ {
				actors, bucketErr := worldMonstersInBucket(c.Request.Context(), scopeRegion, bx, by)
				if bucketErr != nil {
					continue
				}
				for _, monster := range actors {
					if monster.State == MonsterStateDefeated || math.Hypot(monster.X-float64(entry.X), monster.Y-float64(entry.Y)) > float64(radius) {
						continue
					}
					if _, seen := seenActors[monster.InstanceID]; seen {
						continue
					}
					seenActors[monster.InstanceID] = struct{}{}
					monsters = append(monsters, monster)
				}
			}
		}
	}
	for _, spawn := range proceduralMonsterSpawns(entry, radius) {
		if _, exists := occupied[[2]int{spawn.X, spawn.Y}]; exists {
			continue
		}
		monster, created, err := materializeSpawn(c.Request.Context(), spawn, now)
		if err != nil {
			continue
		}
		if monster.State != MonsterStateDefeated {
			monsters = append(monsters, monster)
		}
		if created {
			broadcastWorldCombat(monster, "monster_spawned", map[string]any{"procedural": true})
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
	mapID := worldMapIDForRegion(monster.RegionID)
	event := WorldCombatEvent{EventID: uuid.NewString(), ServerTime: time.Now().UTC(), RegionID: monster.RegionID, MapID: mapID, EntityID: monster.InstanceID, EntityRevision: monster.Revision, EventType: eventType, Position: WorldPosition{monster.X, monster.Y}, Payload: authoritative}
	targets := map[uuid.UUID]struct{}{}
	for _, p := range worldPosIndex.NearbyInScope(int(monster.X), int(monster.Y), WorldMonsterActivationRadius, "", mapID, "") {
		if !isRegionWithinScope(monster.RegionID, p.Region) {
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
		if !isRegionWithinScope(monster.RegionID, p.Region) || !active(p.ID) || isWorldMonsterSafeZone(p.Region, p.X, p.Y) {
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
	targetCandidates := candidates
	if monster.State == MonsterStateAttack && monster.TargetID != "" {
		targetCandidates = make([]WorldPosEntry, 0, 1)
		for _, candidate := range candidates {
			if candidate.ID == monster.TargetID {
				targetCandidates = append(targetCandidates, candidate)
				break
			}
		}
	}
	target, found := selectMonsterTarget(monster, targetCandidates, active)
	if monster.State == MonsterStateReturn {
		moveMonsterToward(&monster, monster.SpawnX, monster.SpawnY, dt)
		if math.Hypot(monster.X-monster.SpawnX, monster.Y-monster.SpawnY) < 0.1 {
			monster.X = monster.SpawnX
			monster.Y = monster.SpawnY
			monster.State = MonsterStateIdle
			monster.HP = monster.MaxHP
		}
	} else if !found || (!monster.IsBoss && math.Hypot(target.X-monster.SpawnX, target.Y-monster.SpawnY) > monster.LeashRadius) {
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
		return monster, map[string]string{MonsterStateChase: "monster_targeted", MonsterStateAttack: "monster_encounter_started", MonsterStateReturn: "monster_returned", MonsterStateIdle: "monster_returned"}[monster.State]
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
	// Check every crossed cell so a delayed tick cannot tunnel through a safe
	// zone merely because its final endpoint lies outside it.
	segments := int(math.Ceil(step * 2))
	if segments < 1 {
		segments = 1
	}
	for i := 1; i <= segments; i++ {
		ratio := float64(i) / float64(segments)
		sx, sy := monster.X+(nx-monster.X)*ratio, monster.Y+(ny-monster.Y)*ratio
		ix, iy := int(math.Round(sx)), int(math.Round(sy))
		outsidePatrol := monster.IsBoss && (ix < monster.PatrolMinX || ix > monster.PatrolMaxX || iy < monster.PatrolMinY || iy > monster.PatrolMaxY)
		if (!monster.IsBoss && !inWorld(ix, iy)) || outsidePatrol || isWorldMonsterSafeZone(monster.RegionID, ix, iy) {
			return
		}
	}
	if monster.IsBoss || math.Hypot(nx-monster.SpawnX, ny-monster.SpawnY) <= monster.LeashRadius {
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
		mapID := entry.MapID
		if mapID == "" {
			mapID = worldMapIDForRegion(entry.Region)
		}
		if err != nil || mapID != "world_map" || entry.InstanceID != "" || !isWSConnected(playerID) {
			return false
		}
		return checkRegionAction(entry.Region, "combat") == nil && checkRegionAction(entry.Region, "monster_combat") == nil
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
	bossBuckets := make([]ActiveWorldBucket, 0, 32)
	seenBosses := make(map[string]struct{}, 32)
	seenBossRegions := make(map[string]struct{}, len(regions)*2)
	for _, region := range regions {
		for _, scopeRegion := range regionScopeIDs(region) {
			if _, seen := seenBossRegions[scopeRegion]; seen {
				continue
			}
			seenBossRegions[scopeRegion] = struct{}{}
			bosses, err := cache.WorldBossesInRegion(ctx, scopeRegion)
			if err != nil {
				continue
			}
			for _, boss := range bosses {
				if _, seen := seenBosses[boss.InstanceID]; seen {
					continue
				}
				seenBosses[boss.InstanceID] = struct{}{}
				bx, by := monsterBucket(int(boss.X), int(boss.Y))
				bossBuckets = append(bossBuckets, ActiveWorldBucket{BX: bx, BY: by, X: int(boss.X), Y: int(boss.Y), Region: boss.RegionID, MapID: worldMapIDForRegion(boss.RegionID), BossScope: true})
			}
		}
	}
	cursor := worldMonsterBucketCursor.Add(uint64(worldMonsterMaxBuckets)) - uint64(worldMonsterMaxBuckets)
	buckets := append(bossBuckets, rotatingMonsterBucketWindow(allBuckets, worldMonsterMaxBuckets, cursor)...)
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
		candidates := online
		if !b.BossScope {
			candidates = worldPosIndex.NearbyInScope(bucketCenterX, bucketCenterY, worldMonsterBucketQueryRadius, "", b.MapID, b.InstanceID)
		}
		for monsterIndex, monster := range monsters {
			if monster.IsBoss != b.BossScope {
				continue
			}
			if processed >= worldMonsterMaxPerTick || time.Now().After(deadline) {
				worldMonsterMetrics.skipped.Add(uint64(len(monsters) - monsterIndex))
				break
			}
			processed++
			// Refresh cached monsters as well as newly materialized ones so a deploy
			// takes effect immediately instead of waiting for their Redis TTL.
			tuned := monster
			applyWorldMonsterTuning(ctx, &tuned)
			if monster.State == MonsterStateDefeated && !monster.RespawnAt.IsZero() && !now.Before(monster.RespawnAt) {
				replacement := tuned
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
			next, eventType := tickWorldMonster(tuned, candidates, now, func(id string) bool {
				parsed, parseErr := uuid.Parse(id)
				return parseErr == nil && isWSConnected(parsed)
			})
			attemptEncounter := next.State == MonsterStateAttack &&
				(monster.State != MonsterStateAttack || next.TargetID != monster.TargetID || next.NextAttackAt.IsZero() || !next.NextAttackAt.After(now))
			if attemptEncounter {
				next.NextAttackAt = now.Add(2 * time.Second)
			}
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
			if attemptEncounter && next.TargetID != "" {
				if err := startWorldMonsterEncounter(ctx, next, next.TargetID); err != nil {
					logger.S.Warnf("start world monster encounter %s -> %s: %v", next.InstanceID, next.TargetID, err)
					resetWorldMonsterAfterEncounter(ctx, next)
				}
			}
		}
	}
	worldMonsterMetrics.processed.Add(uint64(processed))
}
