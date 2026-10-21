package handlers

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

const dailyMissionTZ = "Asia/Ho_Chi_Minh"

type dailyMissionDef struct {
	ID          string
	Title       string
	Description string
	Type        string
	TargetID    string
	Target      int64
	MinRank     int
	MaxRank     int
}

type dailyMissionState struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	TargetID    string `json:"target_id"`
	Target      int64  `json:"target"`
	Progress    int64  `json:"progress"`
	RewardType  string `json:"reward_type"`
	Reward      int64  `json:"reward"`
	Completed   bool   `json:"completed"`
	Claimable   bool   `json:"claimable"`
	Claimed     bool   `json:"claimed"`
}

var dailyMissionPool = []dailyMissionDef{
	{ID: "explore_2", Title: "Thăm dò địa mạch", Description: "Khám phá 2 khu vực nhỏ khác nhau.", Type: "explore", Target: 2},
	{ID: "explore_3", Title: "Mở rộng hành trình", Description: "Khám phá 3 khu vực nhỏ khác nhau.", Type: "explore", Target: 3},
	{ID: "pve_normal", Title: "Săn quái thường", Description: "Thắng 1 trận PvE với quái thường.", Type: "pve_victory", TargetID: "Normal", Target: 1},
	{ID: "pve_elite", Title: "Chinh phục tinh anh", Description: "Thắng 1 trận PvE với quái tinh anh.", Type: "pve_victory", TargetID: "Elite", Target: 1, MinRank: 3},
	{ID: "pve_boss", Title: "Thảo phạt Boss", Description: "Thắng 1 trận PvE với Boss.", Type: "pve_victory", TargetID: "Boss", Target: 1, MinRank: 5},
	{ID: "cultivate_1", Title: "Chuyên cần tu luyện", Description: "Thực hiện 1 lần trùng kích tu vi.", Type: "cultivate", Target: 1, MaxRank: 5},
	{ID: "refine_1", Title: "Luyện cổ mỗi ngày", Description: "Luyện thành công 1 Cổ trùng.", Type: "refine", Target: 1},
	{ID: "collect_3", Title: "Thu thập tài nguyên", Description: "Thu hoạch 3 lần tài nguyên.", Type: "collect", Target: 3},
	{ID: "collect_5", Title: "Tích trữ nguyên liệu", Description: "Thu hoạch 5 lần tài nguyên.", Type: "collect", Target: 5},
	{ID: "aperture_collect", Title: "Thu hoạch Tiên Khiếu", Description: "Thu hoạch tài nguyên trong Tiên Khiếu 1 lần.", Type: "aperture_collect", Target: 1, MinRank: 6},
}

func dailyMissionLocation() *time.Location {
	loc, err := time.LoadLocation(dailyMissionTZ)
	if err != nil {
		return time.FixedZone("UTC+07", 7*60*60)
	}
	return loc
}

func dailyMissionDay(now time.Time) (string, time.Time) {
	loc := dailyMissionLocation()
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return start.Format("2006-01-02"), start.AddDate(0, 0, 1)
}

func ensureDailyMissionSet(tx *sql.Tx, playerID uuid.UUID, rank int, now time.Time) error {
	dayKey, _ := dailyMissionDay(now)
	var lockedRank int
	if err := tx.QueryRow(`SELECT rank FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&lockedRank); err != nil {
		return err
	}
	rank = lockedRank
	err := tx.QueryRow(`SELECT rank_snapshot FROM player_daily_mission_rank_snapshots WHERE player_id=$1 AND day_key=$2::date`, playerID, dayKey).Scan(&rank)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec(`INSERT INTO player_daily_mission_rank_snapshots(player_id,day_key,rank_snapshot) VALUES($1,$2::date,$3) ON CONFLICT DO NOTHING`, playerID, dayKey, lockedRank); err != nil {
			return err
		}
		if err := tx.QueryRow(`SELECT rank_snapshot FROM player_daily_mission_rank_snapshots WHERE player_id=$1 AND day_key=$2::date`, playerID, dayKey).Scan(&rank); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM player_daily_missions WHERE player_id=$1 AND day_key=$2::date`, playerID, dayKey).Scan(&count); err != nil {
		return err
	}
	if count == 5 {
		return nil
	}
	if _, err := tx.Exec(`DELETE FROM player_daily_missions WHERE player_id=$1 AND day_key=$2::date`, playerID, dayKey); err != nil {
		return err
	}
	pool := make(map[string][]dailyMissionDef, 5)
	for _, mission := range dailyMissionPool {
		if rank >= mission.MinRank && (mission.MaxRank == 0 || rank <= mission.MaxRank) {
			pool[mission.Type] = append(pool[mission.Type], mission)
		}
	}
	if len(pool) < 5 {
		return fmt.Errorf("daily mission pool has fewer than five options for rank %d", rank)
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(playerID.String() + ":" + dayKey))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	types := make([]string, 0, len(pool))
	for missionType := range pool {
		types = append(types, missionType)
	}
	sort.Strings(types)
	rng.Shuffle(len(types), func(i, j int) { types[i], types[j] = types[j], types[i] })
	if len(types) > 5 {
		types = types[:5]
	}
	for _, missionType := range types {
		variants := pool[missionType]
		mission := variants[rng.Intn(len(variants))]
		rewardType, reward := "primeval_stones", int64(25*max(1, rank))
		if rank >= 6 {
			rewardType, reward = "immortal_stones", int64(max(1, rank-5))
		}
		if _, err := tx.Exec(`INSERT INTO player_daily_missions(player_id,day_key,mission_id,rank_snapshot,mission_type,target_id,target_amount,reward_type,reward_amount)
			VALUES($1,$2::date,$3,$4,$5,$6,$7,$8,$9)`, playerID, dayKey, mission.ID, rank, mission.Type, mission.TargetID, mission.Target, rewardType, reward); err != nil {
			return err
		}
	}
	return nil
}

// TrackDailyMissionProgress records only server-confirmed gameplay actions.
// The source event key is unique per player and day, making retried hooks safe.
func TrackDailyMissionProgress(tx *sql.Tx, playerID uuid.UUID, rank int, sourceEventID, missionType, targetID string, amount int64) error {
	if tx == nil || playerID == uuid.Nil || sourceEventID == "" || missionType == "" || amount <= 0 {
		return nil
	}
	now := time.Now()
	dayKey, _ := dailyMissionDay(now)
	if err := ensureDailyMissionSet(tx, playerID, rank, now); err != nil {
		return err
	}
	result, err := tx.Exec(`INSERT INTO player_daily_mission_events(player_id,day_key,source_event_id)
		VALUES($1,$2::date,$3) ON CONFLICT DO NOTHING`, playerID, dayKey, sourceEventID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return nil
	}
	_, err = tx.Exec(`UPDATE player_daily_missions
		SET progress=LEAST(target_amount, progress+$1),
			completed_at=CASE WHEN LEAST(target_amount, progress+$1) >= target_amount THEN COALESCE(completed_at,NOW()) ELSE completed_at END
		WHERE player_id=$2 AND day_key=$3::date AND mission_type=$4
		AND (target_id='' OR LOWER(target_id)=LOWER($5)) AND claimed_at IS NULL`, amount, playerID, dayKey, missionType, targetID)
	return err
}

func GetDailyMissions(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()
	var rank int
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT rank FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&rank); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải cảnh giới"})
		return
	}
	now := time.Now()
	dayKey, resetAt := dailyMissionDay(now)
	if err := ensureDailyMissionSet(tx, playerID, rank, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo nhiệm vụ hằng ngày"})
		return
	}
	rows, err := tx.QueryContext(c.Request.Context(), `SELECT mission_id,mission_type,target_id,target_amount,progress,reward_type,reward_amount,completed_at,claimed_at
		FROM player_daily_missions WHERE player_id=$1 AND day_key=$2::date ORDER BY mission_id`, playerID, dayKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhiệm vụ hằng ngày"})
		return
	}
	missions := make([]dailyMissionState, 0, 5)
	for rows.Next() {
		var m dailyMissionState
		var missionType, targetID string
		var completedAt, claimedAt sql.NullTime
		if err := rows.Scan(&m.ID, &missionType, &targetID, &m.Target, &m.Progress, &m.RewardType, &m.Reward, &completedAt, &claimedAt); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc nhiệm vụ hằng ngày"})
			return
		}
		m.Type, m.TargetID = missionType, targetID
		for _, def := range dailyMissionPool {
			if def.ID == m.ID {
				m.Title, m.Description = def.Title, def.Description
				break
			}
		}
		m.Completed = completedAt.Valid
		m.Claimed = claimedAt.Valid
		m.Claimable = m.Completed && !m.Claimed
		missions = append(missions, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc nhiệm vụ hằng ngày"})
		return
	}
	rows.Close()
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu nhiệm vụ hằng ngày"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"day_key": dayKey, "reset_at": resetAt, "missions": missions})
}

func ClaimDailyMission(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		MissionID string `json:"mission_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã nhiệm vụ"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()
	dayKey, _ := dailyMissionDay(time.Now())
	var rank int
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT rank FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&rank); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải cảnh giới"})
		return
	}
	if err := ensureDailyMissionSet(tx, playerID, rank, time.Now()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo nhiệm vụ hằng ngày"})
		return
	}
	var rewardType string
	var reward int64
	err = tx.QueryRowContext(c.Request.Context(), `UPDATE player_daily_missions SET claimed_at=NOW()
		WHERE player_id=$1 AND day_key=$2::date AND mission_id=$3 AND completed_at IS NOT NULL AND claimed_at IS NULL
		RETURNING reward_type,reward_amount`, playerID, dayKey, req.MissionID).Scan(&rewardType, &reward)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusConflict, gin.H{"error": "Nhiệm vụ chưa hoàn thành, đã nhận thưởng hoặc đã hết hạn"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể nhận thưởng"})
		return
	}
	var query string
	switch rewardType {
	case "primeval_stones":
		query = `UPDATE players SET primeval_stones=primeval_stones+$1 WHERE id=$2`
	case "immortal_stones":
		query = `UPDATE players SET immortal_stones=immortal_stones+$1 WHERE id=$2`
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Loại phần thưởng không hợp lệ"})
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), query, reward, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cấp phần thưởng"})
		return
	}
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhân vật"})
		return
	}
	rewardName := "Nguyên Thạch"
	if rewardType == "immortal_stones" {
		rewardName = "Tiên Nguyên Thạch"
	}
	auditlog.Log(tx, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "daily_mission.claim",
		EntityType:     "daily_mission",
		EntityID:       req.MissionID,
		RequestSummary: fmt.Sprintf("Nhận thưởng nhiệm vụ hằng ngày %s ngày %s: +%d %s", req.MissionID, dayKey, reward, rewardName),
		Metadata: map[string]interface{}{
			"mission_id":    req.MissionID,
			"day_key":       dayKey,
			"reward_type":   rewardType,
			"reward_amount": reward,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu phần thưởng"})
		return
	}
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": "Đã nhận thưởng nhiệm vụ", "reward_type": rewardType, "reward": reward})
}
