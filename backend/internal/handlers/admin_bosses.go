package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// AdminWorldBoss is the admin-facing projection of one live boss actor kept in
// Redis while the spawn is active (see cache.WorldMonster).
type AdminWorldBoss struct {
	RegionID          string     `json:"region_id"`
	InstanceID        string     `json:"instance_id"`
	SpawnID           string     `json:"spawn_id"`
	MonsterTemplateID string     `json:"monster_template_id"`
	Name              string     `json:"name"`
	Rank              int        `json:"rank"`
	SubRank           int        `json:"sub_rank"`
	X                 float64    `json:"x"`
	Y                 float64    `json:"y"`
	SpawnX            float64    `json:"spawn_x"`
	SpawnY            float64    `json:"spawn_y"`
	HP                float64    `json:"hp"`
	MaxHP             float64    `json:"max_hp"`
	State             string     `json:"state"`
	TargetID          string     `json:"target_id,omitempty"`
	RespawnSeconds    int        `json:"respawn_seconds"`
	RespawnAt         *time.Time `json:"respawn_at,omitempty"`
	Revision          int64      `json:"revision"`
	LastUpdateAt      time.Time  `json:"last_update_at"`
}

func newAdminWorldBoss(monster cache.WorldMonster) AdminWorldBoss {
	name := monster.MonsterTemplateID
	if template, ok := models.MonsterRegistry[monster.MonsterTemplateID]; ok && template.Name != "" {
		name = template.Name
	}
	out := AdminWorldBoss{
		RegionID: monster.RegionID, InstanceID: monster.InstanceID, SpawnID: monster.SpawnID,
		MonsterTemplateID: monster.MonsterTemplateID, Name: name,
		Rank: monster.Rank, SubRank: monster.SubRank,
		X: monster.X, Y: monster.Y, SpawnX: monster.SpawnX, SpawnY: monster.SpawnY,
		HP: monster.HP, MaxHP: monster.MaxHP, State: monster.State, TargetID: monster.TargetID,
		RespawnSeconds: monster.RespawnSeconds, Revision: monster.Revision, LastUpdateAt: monster.LastUpdateAt,
	}
	if !monster.RespawnAt.IsZero() {
		respawnAt := monster.RespawnAt
		out.RespawnAt = &respawnAt
	}
	return out
}

// AdminListWorldBosses returns every live boss currently held in Redis. Bosses
// are only materialized while a spawn is active, so this is the authoritative
// "where is the boss right now" view.
func AdminListWorldBosses(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	if cache.Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis không khả dụng — chưa thể đọc vị trí boss"})
		return
	}
	regionFilter := strings.TrimSpace(c.Query("region_id"))
	regions, err := cache.WorldBossRegionIDs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể liệt kê khu vực có boss: " + err.Error()})
		return
	}
	bosses := make([]AdminWorldBoss, 0, len(regions))
	for _, regionID := range regions {
		if regionFilter != "" && regionID != regionFilter {
			continue
		}
		members, err := cache.WorldBossesInRegion(c.Request.Context(), regionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Không thể đọc boss khu vực %s: %v", regionID, err)})
			return
		}
		for _, member := range members {
			bosses = append(bosses, newAdminWorldBoss(member))
		}
	}
	sort.Slice(bosses, func(i, j int) bool {
		if bosses[i].RegionID != bosses[j].RegionID {
			return bosses[i].RegionID < bosses[j].RegionID
		}
		if bosses[i].X != bosses[j].X {
			return bosses[i].X < bosses[j].X
		}
		return bosses[i].Y < bosses[j].Y
	})
	// Regions that own boss map_cells are listed even when no actor is alive, so
	// the admin UI can spawn them instead of waiting for a player snapshot.
	regionsWithCells := make([]string, 0)
	if db.DB != nil {
		if cellRows, err := db.DB.QueryContext(c.Request.Context(), `SELECT DISTINCT region_id FROM map_cells WHERE is_active=TRUE AND lower(cell_type)='boss' ORDER BY region_id`); err == nil {
			for cellRows.Next() {
				var regionID string
				if cellRows.Scan(&regionID) == nil {
					regionsWithCells = append(regionsWithCells, regionID)
				}
			}
			_ = cellRows.Close()
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": bosses, "total": len(bosses), "boss_regions": regionsWithCells})
}

type adminWorldBossUpdateRequest struct {
	X     *float64 `json:"x"`
	Y     *float64 `json:"y"`
	HP    *float64 `json:"hp"`
	State *string  `json:"state"`
}

func (req adminWorldBossUpdateRequest) empty() bool {
	return req.X == nil && req.Y == nil && req.HP == nil && req.State == nil
}

// applyAdminWorldBossUpdate mutates a live boss from an admin request. It is a
// pure value transform (bounds are passed in) so the rules stay unit-testable
// without Redis. The returned event name is empty when nothing changed.
func applyAdminWorldBossUpdate(monster *cache.WorldMonster, req adminWorldBossUpdateRequest, minX, minY, maxX, maxY int, now time.Time) (string, error) {
	if req.empty() {
		return "", fmt.Errorf("cần ít nhất một trường để cập nhật (x, y, hp hoặc state)")
	}
	if req.State != nil {
		switch strings.ToUpper(strings.TrimSpace(*req.State)) {
		case MonsterStateIdle, MonsterStateChase, MonsterStateAttack, MonsterStateReturn, MonsterStateDefeated:
		default:
			return "", fmt.Errorf("state không hợp lệ")
		}
	}
	for _, value := range []*float64{req.X, req.Y, req.HP} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return "", fmt.Errorf("toạ độ và HP phải là số hữu hạn")
		}
	}

	event := ""
	// Revive/defeat first: they reset position and HP, so later field writes win.
	if req.State != nil {
		switch strings.ToUpper(strings.TrimSpace(*req.State)) {
		case MonsterStateDefeated:
			monster.State = MonsterStateDefeated
			monster.HP = 0
			monster.TargetID = ""
			monster.RespawnAt = now.Add(monsterRespawnDelay(monster.RespawnSeconds))
			event = "monster_defeated"
		case MonsterStateIdle:
			monster.State = MonsterStateIdle
			monster.HP = monster.MaxHP
			monster.X, monster.Y = monster.SpawnX, monster.SpawnY
			monster.TargetID = ""
			monster.RespawnAt = time.Time{}
			event = "monster_respawned"
		default:
			state := strings.ToUpper(strings.TrimSpace(*req.State))
			if state != monster.State {
				monster.State = state
				monster.RespawnAt = time.Time{}
				event = "monster_updated"
			}
		}
	}

	if req.X != nil || req.Y != nil {
		x, y := monster.X, monster.Y
		if req.X != nil {
			x = *req.X
		}
		if req.Y != nil {
			y = *req.Y
		}
		if x < float64(minX) || x > float64(maxX) || y < float64(minY) || y > float64(maxY) {
			return "", fmt.Errorf("toạ độ (%d, %d) nằm ngoài khu vực %s", int(x), int(y), monster.RegionID)
		}
		if x != monster.X || y != monster.Y {
			monster.X, monster.Y = x, y
			monster.TargetID = ""
			if event == "" {
				event = "monster_moved"
			}
		}
	}

	if req.HP != nil {
		hp := math.Min(math.Max(*req.HP, 0), monster.MaxHP)
		if hp != monster.HP {
			monster.HP = hp
			if hp == 0 {
				monster.State = MonsterStateDefeated
				monster.TargetID = ""
				monster.RespawnAt = now.Add(monsterRespawnDelay(monster.RespawnSeconds))
				event = "monster_defeated"
			} else if monster.State == MonsterStateDefeated {
				monster.State = MonsterStateIdle
				monster.RespawnAt = time.Time{}
				event = "monster_respawned"
			} else if event == "" {
				event = "monster_updated"
			}
		}
	}

	if event == "" {
		return "", nil
	}
	monster.Revision++
	monster.LastUpdateAt = now
	return event, nil
}

// AdminUpdateWorldBoss teleports, heals, damages or revives one live boss. The
// write uses the revision CAS the workers use, so a concurrent tick cannot be
// clobbered and cannot clobber this update.
func AdminUpdateWorldBoss(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	if cache.Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis không khả dụng — chưa thể cập nhật boss"})
		return
	}
	regionID, instanceID := c.Param("region_id"), c.Param("instance_id")
	var req adminWorldBossUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	monster, exists, err := cache.GetWorldMonster(c.Request.Context(), regionID, instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc boss: " + err.Error()})
		return
	}
	// world:monster:{region}:{instance} also holds ordinary monsters, so this
	// endpoint must refuse anything outside the boss scope it was granted.
	if !exists || !monster.IsBoss {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy boss đang sống với region/instance này"})
		return
	}

	oldBucketX, oldBucketY := monsterBucket(int(monster.X), int(monster.Y))
	minX, minY, maxX, maxY := monsterPatrolBounds(regionID)
	event, err := applyAdminWorldBossUpdate(&monster, req, minX, minY, maxX, maxY, time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if event == "" {
		c.JSON(http.StatusOK, gin.H{"message": "Không có thay đổi nào được áp dụng"})
		return
	}

	newBucketX, newBucketY := monsterBucket(int(monster.X), int(monster.Y))
	written, err := casWorldMonster(c.Request.Context(), monster, monster, monster.Revision-1, oldBucketX, oldBucketY, newBucketX, newBucketY, monsterTTL(monster.RespawnSeconds))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi boss: " + err.Error()})
		return
	}
	if !written {
		c.JSON(http.StatusConflict, gin.H{"error": "Boss vừa được worker cập nhật, hãy thử lại"})
		return
	}
	broadcastWorldCombat(monster, event, map[string]any{"admin_id": adminID.String()})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật boss", "data": newAdminWorldBoss(monster)})
}

type adminWorldBossSpawnRequest struct {
	RegionID string `json:"region_id"`
	X        *int   `json:"x"`
	Y        *int   `json:"y"`
}

// AdminSpawnWorldBosses materializes or revives boss actors from the active
// boss map_cells of one region, so an admin can force a spawn instead of
// waiting for the next player AOI snapshot.
func AdminSpawnWorldBosses(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	if cache.Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis không khả dụng — chưa thể tạo boss"})
		return
	}
	var req adminWorldBossSpawnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	req.RegionID = strings.TrimSpace(req.RegionID)
	if req.RegionID == "" || len(req.RegionID) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "region_id là bắt buộc"})
		return
	}

	query := `
		SELECT x, y, COALESCE(content_id,''),
		       CASE WHEN COALESCE(metadata->>'rank','') ~ '^[0-9]+$' THEN (metadata->>'rank')::int ELSE 1 END,
		       CASE WHEN COALESCE(metadata->>'sub_rank','') ~ '^[0-9]+$' THEN (metadata->>'sub_rank')::int ELSE 1 END,
		       COALESCE(respawn_seconds,300)
		FROM map_cells
		WHERE region_id=$1 AND is_active=TRUE AND lower(cell_type)='boss'`
	args := []interface{}{req.RegionID}
	if req.X != nil && req.Y != nil {
		query += ` AND x=$2 AND y=$3`
		args = append(args, *req.X, *req.Y)
	}
	query += ` ORDER BY x,y`
	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc ô boss: " + err.Error()})
		return
	}
	defer rows.Close()

	now := time.Now().UTC()
	minX, minY, maxX, maxY := monsterPatrolBounds(req.RegionID)
	created, revived, alive := 0, 0, 0
	failures := make([]string, 0)
	for rows.Next() {
		var spawn monsterSpawnRow
		spawn.RegionID, spawn.IsBoss = req.RegionID, true
		if err := rows.Scan(&spawn.X, &spawn.Y, &spawn.MonsterID, &spawn.Rank, &spawn.SubRank, &spawn.RespawnSeconds); err != nil {
			failures = append(failures, "không đọc được một ô boss")
			continue
		}
		if spawn.MonsterID == "" {
			failures = append(failures, fmt.Sprintf("ô (%d,%d) thiếu content_id", spawn.X, spawn.Y))
			continue
		}
		spawn.ID = fmt.Sprintf("mapcell:%s:%d:%d", spawn.RegionID, spawn.X, spawn.Y)
		if spawn.Rank < 1 {
			spawn.Rank = 1
		}
		if spawn.SubRank < 1 {
			spawn.SubRank = 1
		}
		candidate, err := worldMonsterCandidateForSpawn(spawn, now)
		if err != nil {
			failures = append(failures, fmt.Sprintf("ô (%d,%d): %v", spawn.X, spawn.Y, err))
			continue
		}
		if err := resetPersistentBossCell(spawn.RegionID, spawn.X, spawn.Y); err != nil {
			failures = append(failures, fmt.Sprintf("ô (%d,%d): %v", spawn.X, spawn.Y, err))
			continue
		}
		bx, by := monsterBucket(spawn.X, spawn.Y)
		runtime, isNew, err := materializeWorldMonster(c.Request.Context(), candidate, bx, by, monsterTTL(spawn.RespawnSeconds))
		if err != nil {
			failures = append(failures, fmt.Sprintf("ô (%d,%d): %v", spawn.X, spawn.Y, err))
			continue
		}
		if isNew {
			created++
			broadcastWorldCombat(runtime, "monster_spawned", map[string]any{"boss": true, "admin_id": adminID.String()})
			continue
		}
		if runtime.State != MonsterStateDefeated {
			alive++
			continue
		}
		// Existing defeated actor keeps its instance; revive it in place.
		idle := MonsterStateIdle
		oldBucketX, oldBucketY := monsterBucket(int(runtime.X), int(runtime.Y))
		event, err := applyAdminWorldBossUpdate(&runtime, adminWorldBossUpdateRequest{State: &idle}, minX, minY, maxX, maxY, now)
		if err != nil || event == "" {
			alive++
			continue
		}
		newBucketX, newBucketY := monsterBucket(int(runtime.X), int(runtime.Y))
		written, err := casWorldMonster(c.Request.Context(), runtime, runtime, runtime.Revision-1, oldBucketX, oldBucketY, newBucketX, newBucketY, monsterTTL(runtime.RespawnSeconds))
		if err != nil || !written {
			failures = append(failures, fmt.Sprintf("ô (%d,%d): không thể hồi sinh", spawn.X, spawn.Y))
			continue
		}
		revived++
		broadcastWorldCombat(runtime, event, map[string]any{"boss": true, "admin_id": adminID.String()})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc ô boss: " + err.Error()})
		return
	}
	if created+revived+alive == 0 && len(failures) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Khu vực không có ô boss nào đang hoạt động"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%s: tạo mới %d, hồi sinh %d, đang sống %d", req.RegionID, created, revived, alive),
		"created": created, "revived": revived, "alive": alive, "errors": failures,
	})
}

// resetPersistentBossCell clears the map-cell respawn window so a forced spawn
// is not immediately skipped by the next AOI snapshot.
func resetPersistentBossCell(regionID string, x, y int) error {
	cellRegion := regionID
	if pc, found, _ := PersistentCellAt(regionID, x, y); found {
		cellRegion, x, y = pc.RegionID, pc.X, pc.Y
	}
	if db.DB != nil {
		if _, err := db.DB.ExecContext(context.Background(), `UPDATE map_cells SET last_cleared_at=NULL, cleared_by=NULL WHERE region_id=$1 AND x=$2 AND y=$3 AND lower(cell_type)='boss'`, cellRegion, x, y); err != nil {
			return err
		}
	}
	key := persistentCellKey{RegionID: cellRegion, X: x, Y: y}
	zoneReg.mu.Lock()
	if pc, ok := zoneReg.persistent[key]; ok {
		pc.cleared = false
		pc.LastCleared = time.Time{}
		zoneReg.persistent[key] = pc
	}
	zoneReg.mu.Unlock()
	broadcastWorldCell(cellEventRespawn, WorldCellEvent{RegionID: cellRegion, X: x, Y: y, CellType: "Boss"})
	return nil
}

// AdminPurgeWorldBosses removes every live boss actor of one region from Redis.
// The next AOI snapshot rematerializes them from static map-cell data, so this
// is a "reset spawns" operation rather than a content edit.
func AdminPurgeWorldBosses(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	if cache.Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis không khả dụng — chưa thể xoá boss"})
		return
	}
	regionID := strings.TrimSpace(c.Query("region_id"))
	if regionID == "" || len(regionID) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "region_id là bắt buộc"})
		return
	}
	if err := cache.PruneWorldBosses(c.Request.Context(), regionID, map[string]struct{}{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xoá boss: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xoá boss đang sống của khu vực; snapshot kế tiếp sẽ tạo lại từ dữ liệu tĩnh"})
}
