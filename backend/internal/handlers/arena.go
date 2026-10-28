package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type ArenaRanking struct {
	PlayerID        string    `json:"player_id"`
	Username        string    `json:"username"`
	AvatarURL       string    `json:"avatar_url"`
	Rank            int       `json:"rank"`
	Points          int       `json:"points"`
	DailyChallenges int       `json:"daily_challenges"`
	LastChallengeAt time.Time `json:"last_challenge_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func EnsurePlayerArenaRanking(tx *sql.Tx, playerID uuid.UUID) (int, error) {
	var rank int
	err := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", playerID).Scan(&rank)
	if err == sql.ErrNoRows {
		// Acquire transaction-scoped advisory lock to serialize rank assignment
		// without table-locking arena_rankings against concurrent readers/writers.
		_, errLock := tx.Exec("SELECT pg_advisory_xact_lock(7429104)")
		if errLock != nil {
			return 0, errLock
		}

		// Recheck in case another transaction assigned the rank while we were waiting for the lock
		err = tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", playerID).Scan(&rank)
		if err == nil {
			return rank, nil
		} else if err != sql.ErrNoRows {
			return 0, err
		}

		// Assign next rank
		var maxRank int
		_ = tx.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM arena_rankings").Scan(&maxRank)
		rank = maxRank + 1
		_, err = tx.Exec("INSERT INTO arena_rankings (player_id, rank, points) VALUES ($1, $2, 1000)", playerID, rank)
		if err != nil {
			return 0, err
		}
		_, err = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", rank, playerID)
		if err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	return rank, nil
}

func GetArenaLeaderboard(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var dailyChallenges int
	var lastChallengeAt time.Time
	var existingRank int

	// Fast path: load player's current arena rank & daily stats lock-free
	err := db.DB.QueryRowContext(c.Request.Context(), "SELECT rank, daily_challenges, last_challenge_at FROM arena_rankings WHERE player_id = $1", authID).
		Scan(&existingRank, &dailyChallenges, &lastChallengeAt)

	if err == sql.ErrNoRows {
		// Slow path for first-time visitors: initialize ranking
		tx, errTx := db.DB.BeginTx(c.Request.Context(), nil)
		if errTx != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		defer tx.Rollback()

		_, err = EnsurePlayerArenaRanking(tx, authID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thiết lập thứ hạng Võ Đài"})
			return
		}
		if errCommit := tx.Commit(); errCommit != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failure"})
			return
		}
		dailyChallenges = 0
		lastChallengeAt = time.Now()
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player ranking"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT a.player_id, p.username, p.avatar_url, a.rank, a.points, a.updated_at
		FROM arena_rankings a
		JOIN players p ON a.player_id = p.id
		ORDER BY a.rank ASC
		LIMIT 50
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}
	defer rows.Close()

	leaderboard := []ArenaRanking{}
	for rows.Next() {
		var r ArenaRanking
		if errScan := rows.Scan(&r.PlayerID, &r.Username, &r.AvatarURL, &r.Rank, &r.Points, &r.UpdatedAt); errScan == nil {
			leaderboard = append(leaderboard, r)
		}
	}

	now := time.Now()
	lastChallengeLocal := lastChallengeAt.In(now.Location())
	if lastChallengeLocal.YearDay() != now.YearDay() || lastChallengeLocal.Year() != now.Year() {
		dailyChallenges = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"leaderboard":      leaderboard,
		"daily_challenges": dailyChallenges,
	})
}

type ChallengeRequest struct {
	TargetPlayerID string `json:"target_player_id" binding:"required"`
}

func ChallengeArena(c *gin.Context) {
	var req ChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target player ID is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	targetUUID, err := uuid.Parse(req.TargetPlayerID)
	if err != nil || targetUUID == authID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mục tiêu khiêu chiến không hợp lệ"})
		return
	}
	if reason := RealtimeRuntimeError(); reason != "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "realtime_unavailable", "error": "Võ Đài realtime hiện chưa khả dụng."})
		return
	}
	if !middleware.IsRealtimeWorldOwned(targetUUID) {
		c.JSON(http.StatusConflict, gin.H{"code": "arena_opponent_offline", "error": "Đối thủ cần đang kết nối vào thế giới realtime để nhận lời khiêu chiến."})
		return
	}
	// Suspend both world leases in stable order so their last checkpoints become
	// the origin state before the private match is committed.
	participants := []uuid.UUID{authID, targetUUID}
	if participants[0].String() > participants[1].String() {
		participants[0], participants[1] = participants[1], participants[0]
	}
	var handedOff []uuid.UUID
	for _, participantID := range participants {
		handedOffNow, handoffErr := middleware.BeginRealtimeHandoff(participantID)
		if handoffErr != nil {
			for _, previous := range handedOff {
				middleware.AbortRealtimeHandoff(previous)
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "realtime_checkpoint_unavailable", "error": "Không thể đồng bộ trạng thái Võ Đài."})
			return
		}
		if handedOffNow {
			handedOff = append(handedOff, participantID)
		}
	}
	defer func() {
		for _, participantID := range handedOff {
			middleware.AbortRealtimeHandoff(participantID)
		}
	}()

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	for _, participantID := range participants {
		var lockedID uuid.UUID
		if err := tx.QueryRowContext(c.Request.Context(), `SELECT id FROM players WHERE id=$1 FOR UPDATE`, participantID).Scan(&lockedID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy một người tham gia Võ Đài."})
			return
		}
	}

	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	// Ensure rankings
	challengerRank, err := EnsurePlayerArenaRanking(tx, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thiết lập thứ hạng của bạn: " + err.Error()})
		return
	}
	challengedRank, err := EnsurePlayerArenaRanking(tx, targetUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thiết lập thứ hạng đối thủ: " + err.Error()})
		return
	}

	// Check daily challenges limit
	var dailyChallenges int
	var lastChallengeAt time.Time
	err = tx.QueryRow("SELECT daily_challenges, last_challenge_at FROM arena_rankings WHERE player_id = $1", player.ID).Scan(&dailyChallenges, &lastChallengeAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn thông tin Võ Đài: " + err.Error()})
		return
	}

	now := time.Now()
	lastChallengeLocal := lastChallengeAt.In(now.Location())
	if lastChallengeLocal.YearDay() != now.YearDay() || lastChallengeLocal.Year() != now.Year() {
		dailyChallenges = 0
	}
	if dailyChallenges >= 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hôm nay đạo hữu đã hết lượt khiêu chiến Võ Đài (Tối đa 10 lần/ngày)!"})
		return
	}

	_, err = tx.Exec("UPDATE arena_rankings SET daily_challenges = $1, last_challenge_at = $2 WHERE player_id = $3", dailyChallenges+1, now, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật lượt khiêu chiến: " + err.Error()})
		return
	}

	targetPlayer, err := evaluateManaTxEquipped(tx, targetUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đối thủ"})
		return
	}
	var originZones = map[uuid.UUID]string{}
	for _, participantID := range participants {
		var worldZone, regionID, zoneType, assignedMap, worldInstance string
		if err := tx.QueryRowContext(c.Request.Context(), `
			SELECT COALESCE(p.world_zone_id,''),COALESCE(p.current_region,''),COALESCE(r.zone_type,''),
				COALESCE(rwm.world_map_id,''),COALESCE(p.world_instance_id,'')
			FROM players p
			LEFT JOIN regions r ON r.id=p.current_region
			LEFT JOIN region_world_maps rwm ON rwm.region_id=p.current_region
			WHERE p.id=$1
		`, participantID).Scan(&worldZone, &regionID, &zoneType, &assignedMap, &worldInstance); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc vị trí người tham gia Võ Đài."})
			return
		}
		originZone := strings.TrimSpace(worldZone)
		if originZone == "" {
			if assignedMap == "" {
				assignedMap = "world_map"
			}
			originZone = realtimeMapID(regionID, zoneType, assignedMap)
		}
		if worldInstance != "" {
			c.JSON(http.StatusConflict, gin.H{"code": "combat_instance_conflict", "error": "Một người tham gia đang ở trong phiên realtime riêng."})
			return
		}
		var activePrivate bool
		if err := tx.QueryRowContext(c.Request.Context(), `
			SELECT EXISTS (
				SELECT 1 FROM realtime_combat_instances WHERE player_id=$1 AND status='active'
				UNION ALL
				SELECT 1 FROM realtime_combat_session_participants WHERE player_id=$1 AND status='active'
			)
		`, participantID).Scan(&activePrivate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra phiên realtime."})
			return
		}
		if activePrivate {
			c.JSON(http.StatusConflict, gin.H{"code": "combat_instance_conflict", "error": "Một người tham gia đang có phiên realtime chưa kết thúc."})
			return
		}
		originZones[participantID] = originZone
	}
	if originZones[authID] == "" || originZones[targetUUID] == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "Cả hai nhân vật cần có khu vực hiện tại để vào Võ Đài."})
		return
	}
	challengerHP := player.BaseHP
	challengedHP := targetPlayer.BaseHP
	if challengerHP <= 0 {
		challengerHP = gamefx.EffectiveMaxHP(&player)
	}
	if challengedHP <= 0 {
		challengedHP = gamefx.EffectiveMaxHP(&targetPlayer)
	}
	if challengerHP <= 0 || challengedHP <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể xác định sinh lực tối đa của người tham gia."})
		return
	}
	instanceID := uuid.New()
	challengerTeamID := "arena:" + instanceID.String() + ":challenger"
	challengedTeamID := "arena:" + instanceID.String() + ":challenged"
	challengerConfig, err := json.Marshal(RealtimeCombatInstanceVitals{
		CurrentHP: challengerHP, CurrentMana: gamefx.EffectiveMaxMana(&player), X: 19, Y: 24, CombatTeamID: challengerTeamID, OriginZoneID: originZones[authID],
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo trạng thái Võ Đài."})
		return
	}
	challengedConfig, err := json.Marshal(RealtimeCombatInstanceVitals{
		CurrentHP: challengedHP, CurrentMana: gamefx.EffectiveMaxMana(&targetPlayer), X: 45, Y: 24, CombatTeamID: challengedTeamID, OriginZoneID: originZones[targetUUID],
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo trạng thái Võ Đài."})
		return
	}
	metadata, err := json.Marshal(map[string]interface{}{
		"challenger_id": authID.String(), "challenged_id": targetUUID.String(),
		"challenger_name": player.Account.Username, "challenged_name": targetPlayer.Account.Username,
		"challenger_rank": challengerRank, "challenged_rank": challengedRank,
		"region": player.CurrentRegion,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thông tin Võ Đài."})
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `
		INSERT INTO realtime_combat_sessions (id,mode,zone_id,origin_zone_id,payload)
		VALUES ($1,'arena',$2,$3,$4)
	`, instanceID, practiceRealtimeMapID, originZones[authID], metadata); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở trận Võ Đài realtime."})
		return
	}
	for _, entry := range []struct {
		id      uuid.UUID
		payload []byte
	}{{authID, challengerConfig}, {targetUUID, challengedConfig}} {
		if _, err = tx.ExecContext(c.Request.Context(), `
			INSERT INTO realtime_combat_session_participants (session_id,player_id,payload)
			VALUES ($1,$2,$3)
		`, instanceID, entry.id, entry.payload); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Không thể đăng ký người tham gia vào Võ Đài realtime."})
			return
		}
	}
	qiGained := AddHumanQi(&player, 3.0)
	for _, participantID := range participants {
		if _, err = tx.ExecContext(c.Request.Context(), `
			UPDATE players SET world_zone_id=$2,world_instance_id=$3,
				world_location_revision=world_location_revision+1,
				human_qi=CASE WHEN id=$4 THEN $5 ELSE human_qi END
			WHERE id=$1
		`, participantID, practiceRealtimeMapID, "combat:"+instanceID.String(), authID, player.HumanQi); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Không thể chuyển người tham gia vào map Võ Đài."})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failed"})
		return
	}
	for _, participantID := range handedOff {
		middleware.FinishRealtimeHandoff(participantID)
	}
	PublishPlayerRealtimeEvent(authID, PlayerEventUpdated)
	PublishPlayerRealtimeEvent(targetUUID, PlayerEventUpdated)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     fmt.Sprintf("Khiêu chiến Võ Đài realtime bắt đầu! Nhân Khí +%.0f.", qiGained),
		"instance_id": instanceID, "zone_id": practiceRealtimeMapID,
	})
}
