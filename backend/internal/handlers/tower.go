package handlers

import (
	"database/sql"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const towerWeeklyLossLimit = 5

type towerProgress struct {
	BestFloor    int `json:"best_floor"`
	WeeklyLosses int `json:"weekly_losses"`
	LossLimit    int `json:"loss_limit"`
}

func ensureTowerProgress(tx *sql.Tx, playerID uuid.UUID, lock bool) (towerProgress, error) {
	if _, err := tx.Exec(`INSERT INTO infinite_tower_progress (player_id) VALUES ($1) ON CONFLICT (player_id) DO NOTHING`, playerID); err != nil {
		return towerProgress{}, err
	}
	if _, err := tx.Exec(`UPDATE infinite_tower_progress SET weekly_losses=0, week_started_at=date_trunc('week', CURRENT_DATE)::date, updated_at=NOW()
		WHERE player_id=$1 AND week_started_at < date_trunc('week', CURRENT_DATE)::date`, playerID); err != nil {
		return towerProgress{}, err
	}
	query := `SELECT best_floor, weekly_losses FROM infinite_tower_progress WHERE player_id=$1`
	if lock {
		query += ` FOR UPDATE`
	}
	var progress towerProgress
	if err := tx.QueryRow(query, playerID).Scan(&progress.BestFloor, &progress.WeeklyLosses); err != nil {
		return towerProgress{}, err
	}
	progress.LossLimit = towerWeeklyLossLimit
	return progress, nil
}

func GetInfiniteTowerState(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), &sql.TxOptions{ReadOnly: false})
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải tiến độ tháp.", err))
		return
	}
	defer tx.Rollback()
	progress, err := ensureTowerProgress(tx, playerID, false)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải tiến độ tháp.", err))
		return
	}
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải tiến độ tháp.", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"progress": progress, "next_floor": progress.BestFloor + 1})
}

func GetInfiniteTowerLeaderboard(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT p.id, p.username, t.best_floor, t.best_floor_at
		FROM infinite_tower_progress t JOIN players p ON p.id=t.player_id
		WHERE t.best_floor > 0 ORDER BY t.best_floor DESC, t.best_floor_at ASC NULLS LAST, p.username ASC LIMIT 100`)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải bảng xếp hạng tháp.", err))
		return
	}
	type entry struct {
		PlayerID  string `json:"player_id"`
		Username  string `json:"username"`
		BestFloor int    `json:"best_floor"`
		ReachedAt any    `json:"reached_at"`
	}
	entries := make([]entry, 0)
	for rows.Next() {
		var item entry
		var id uuid.UUID
		var reachedAt sql.NullTime
		if err := rows.Scan(&id, &item.Username, &item.BestFloor, &reachedAt); err != nil {
			rows.Close()
			httperr.Respond(c, httperr.Internal("Không thể đọc bảng xếp hạng tháp.", err))
			return
		}
		item.PlayerID = id.String()
		if reachedAt.Valid {
			item.ReachedAt = reachedAt.Time
		}
		entries = append(entries, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		httperr.Respond(c, httperr.Internal("Không thể đọc bảng xếp hạng tháp.", err))
		return
	}
	rows.Close()
	c.JSON(http.StatusOK, gin.H{"leaderboard": entries})
}

// ChallengeInfiniteTower starts the next untopped floor as a virtual combat.
// A challenge consumes a weekly attempt only when it is lost or abandoned.
func ChallengeInfiniteTower(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể mở chiến trường tháp.", err))
		return
	}
	defer tx.Rollback()
	if _, sessionErr := loadCombatSession(tx, playerID, nil, true); sessionErr == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu đang có phiên chiến đấu."})
		return
	} else if sessionErr != sql.ErrNoRows {
		httperr.Respond(c, httperr.Internal("Không thể kiểm tra phiên chiến đấu.", sessionErr))
		return
	}
	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhân vật."})
		return
	}
	if player.ActiveCombat != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "combat_active", "error": "Nhân vật đang chiến đấu."})
		return
	}
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Nhân vật đã chết, không thể vào tháp."})
		return
	}
	progress, err := ensureTowerProgress(tx, player.ID, true)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc tiến độ tháp.", err))
		return
	}
	if progress.WeeklyLosses >= progress.LossLimit {
		c.JSON(http.StatusForbidden, gin.H{"code": "tower_attempts_exhausted", "error": "Đạo hữu đã dùng hết lượt khiêu chiến tháp trong tuần."})
		return
	}
	monsters := make([]models.MonsterTemplate, 0, len(models.MonsterRegistry))
	for _, monster := range models.MonsterRegistry {
		switch strings.ToUpper(monster.Type) {
		case "NORMAL", "ELITE", "BOSS":
			monsters = append(monsters, monster)
		}
	}
	floor := progress.BestFloor + 1
	monster, found := towerMonsterForFloor(floor, monsters)
	if !found {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Chưa có quái vật khả dụng cho tháp."})
		return
	}
	rank := towerEnemyRank(floor)
	subRank := towerEnemySubRank(floor)
	// Build the template at its Rank 1 baseline, then apply the tower's shared
	// floor curve. Player rank never changes the enemy assigned to a floor.
	encounter := models.BuildCombatStateFromMonster(monster, 1, 1)
	growth := towerEnemyGrowth(floor)
	encounter.EnemyTemplateID = "tower_floor_" + intToString(floor)
	encounter.TowerFloor = floor
	encounter.EnemyName = "Tầng " + intToString(floor) + " · " + monster.Name
	encounter.EnemyType = "TOWER"
	encounter.CombatMode = "virtual"
	encounter.EnemyRank = rank
	encounter.EnemySubRank = subRank
	encounter.EnemyHP *= growth
	encounter.EnemyMaxHP *= growth
	encounter.EnemyDamage *= growth
	encounter.EnemyDefense *= growth
	encounter.AIProfile = monster.AIProfile
	encounter.LootTableID = ""
	player.ActiveCombat = InitializeCombatState(tx, &encounter, &player)
	if _, err := tx.Exec(`UPDATE players SET active_combat=$1 WHERE id=$2`, player.ActiveCombat, player.ID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể chuẩn bị tầng tháp.", err))
		return
	}
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể chuẩn bị tầng tháp.", err))
		return
	}
	PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	respondActionOnly(c, http.StatusOK, player, gin.H{"floor": floor, "active_combat": player.ActiveCombat, "progress": progress})
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

func towerMonsterTier(monsterType string) int {
	switch strings.ToUpper(monsterType) {
	case "NORMAL", "MONSTER":
		return 0
	case "ELITE":
		return 1
	default:
		return 2
	}
}

func towerFloorTier(floor int) int {
	if floor < 1 {
		floor = 1
	}
	if floor%10 == 0 {
		return 2 // boss floor overrides the elite cadence
	}
	if floor%5 == 0 {
		return 1
	}
	return 0
}

func towerTierEncounterIndex(floor, tier int) int {
	if floor < 1 {
		floor = 1
	}
	switch tier {
	case 1:
		return floor/5 - floor/10 - 1
	case 2:
		return floor/10 - 1
	default:
		return floor - floor/5 - 1
	}
}

func towerMonsterForFloor(floor int, monsters []models.MonsterTemplate) (models.MonsterTemplate, bool) {
	if floor < 1 {
		floor = 1
	}
	pools := [3][]models.MonsterTemplate{}
	for _, monster := range monsters {
		tier := towerMonsterTier(monster.Type)
		if tier < len(pools) {
			pools[tier] = append(pools[tier], monster)
		}
	}
	for tier := range pools {
		sort.Slice(pools[tier], func(i, j int) bool { return pools[tier][i].ID < pools[tier][j].ID })
	}
	tier := towerFloorTier(floor)
	if len(pools[tier]) == 0 {
		// Fall back to the next available lower tier if a data import removes
		// every monster from a scheduled tier.
		for fallback := tier - 1; fallback >= 0; fallback-- {
			if len(pools[fallback]) > 0 {
				tier = fallback
				break
			}
		}
	}
	if len(pools[tier]) == 0 {
		for fallback := tier + 1; fallback < len(pools); fallback++ {
			if len(pools[fallback]) > 0 {
				tier = fallback
				break
			}
		}
	}
	if len(pools[tier]) == 0 {
		return models.MonsterTemplate{}, false
	}
	index := towerTierEncounterIndex(floor, tier) % len(pools[tier])
	if index < 0 {
		index = 0
	}
	return pools[tier][index], true
}

var towerRankFloorStarts = [...]int{1, 10, 20, 30, 40, 90, 140, 190, 240}

func towerEnemyRank(floor int) int {
	if floor < 1 {
		return 1
	}
	rank := 1
	for nextRank := 2; nextRank <= 9; nextRank++ {
		if floor < towerRankFloorStarts[nextRank-1] {
			break
		}
		rank = nextRank
	}
	return rank
}

func towerEnemySubRank(floor int) int {
	rank := towerEnemyRank(floor)
	return min(floor-towerRankFloorStarts[rank-1]+1, 10)
}

func towerEnemyGrowth(floor int) float64 {
	if floor < 1 {
		floor = 1
	}
	rank := towerEnemyRank(floor)
	startFloor := towerRankFloorStarts[rank-1]
	startScale := models.RankScale(rank)
	if rank < 9 {
		endFloor := towerRankFloorStarts[rank]
		endScale := models.RankScale(rank + 1)
		progress := float64(floor-startFloor) / float64(endFloor-startFloor)
		return math.Exp(math.Log(startScale) + progress*(math.Log(endScale)-math.Log(startScale)))
	}
	// Rank 9 has no higher displayed rank, but the endless tower keeps scaling
	// by 100x across each additional 50-floor span.
	extraSpans := float64(floor-startFloor) / 50.0
	return math.Min(startScale*math.Pow(100, extraSpans), 1e22)
}

func settleInfiniteTower(tx *sql.Tx, playerID uuid.UUID, floor int, won bool) (int64, error) {
	if floor < 1 {
		return 0, nil
	}
	if _, err := ensureTowerProgress(tx, playerID, true); err != nil {
		return 0, err
	}
	if won {
		if _, err := tx.Exec(`UPDATE infinite_tower_progress SET best_floor=$2, best_floor_at=NOW(), updated_at=NOW() WHERE player_id=$1 AND best_floor<$2`, playerID, floor); err != nil {
			return 0, err
		}
		reward := int64(100 + min(floor, 1000)*5)
		if err := InventorySvc.AddItem(tx, playerID.String(), "arena_coin", "materials", reward); err != nil {
			return 0, err
		}
		return reward, nil
	}
	_, err := tx.Exec(`UPDATE infinite_tower_progress SET weekly_losses=weekly_losses+1, updated_at=NOW() WHERE player_id=$1`, playerID)
	return 0, err
}

func AbandonInfiniteTower(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể rời khỏi tháp.", err))
		return
	}
	defer tx.Rollback()
	if _, sessionErr := loadCombatSession(tx, playerID, nil, true); sessionErr != nil && sessionErr != sql.ErrNoRows {
		httperr.Respond(c, httperr.Internal("Không thể khóa phiên tháp.", sessionErr))
		return
	}
	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil || player.ActiveCombat == nil || player.ActiveCombat.EnemyType != "TOWER" {
		c.JSON(http.StatusConflict, gin.H{"error": "Không có trận tháp đang diễn ra."})
		return
	}
	floor := player.ActiveCombat.TowerFloor
	if floor <= 0 {
		floor = 1
	}
	if _, err := settleInfiniteTower(tx, player.ID, floor, false); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể ghi nhận lượt rời tháp.", err))
		return
	}
	if _, err := tx.Exec(`UPDATE combat_sessions SET status='abandoned', pending_actor='', deadline_at=NULL, updated_at=NOW() WHERE player_id=$1 AND status='active'`, player.ID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể kết thúc trận tháp.", err))
		return
	}
	if _, err := tx.Exec(`UPDATE players SET active_combat=NULL WHERE id=$1`, player.ID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể kết thúc trận tháp.", err))
		return
	}
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể rời khỏi tháp.", err))
		return
	}
	PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
	c.JSON(http.StatusOK, gin.H{"abandoned": true})
}
