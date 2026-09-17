package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
		// Acquire lock to serialize rank assignment across all concurrent requests
		_, errLock := tx.Exec("LOCK TABLE arena_rankings IN SHARE ROW EXCLUSIVE MODE")
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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Ensure caller has ranking
	_, err = EnsurePlayerArenaRanking(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thiết lập thứ hạng Võ Đài"})
		return
	}
	if errCommit := tx.Commit(); errCommit != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failure"})
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

	var dailyChallenges int
	var lastChallengeAt time.Time
	_ = db.DB.QueryRowContext(c.Request.Context(), "SELECT daily_challenges, last_challenge_at FROM arena_rankings WHERE player_id = $1", authID).Scan(&dailyChallenges, &lastChallengeAt)

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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang ở trong trạng thái chiến đấu!"})
		return
	}

	// Ensure rankings
	_, err = EnsurePlayerArenaRanking(tx, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thiết lập thứ hạng của bạn: " + err.Error()})
		return
	}
	_, err = EnsurePlayerArenaRanking(tx, targetUUID)
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

	var targetPlayer models.Player
	err = tx.QueryRow(`
		SELECT p.id, p.username, p.rank, p.sub_rank, p.hp, p.max_mana, p.path_masteries, `+db.GuInventoryAggregateEquipped("p.id")+`, COALESCE(ia.dao_marks, '{}'::jsonb), p.current_mana, p.base_hp, p.combat_settings
		FROM players p
		LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
		WHERE p.id = $1 FOR UPDATE OF p`, targetUUID).Scan(
		&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
		&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks, &targetPlayer.CurrentMana, &targetPlayer.BaseHP, &targetPlayer.CombatSettings,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đối thủ"})
		return
	}

	targetStats := CalculatePlayerStats(&targetPlayer)

	enemyHP := gamefx.EffectiveMaxHP(&targetPlayer)
	if enemyHP <= 0 {
		enemyHP = 100.0
	}

	enemyGuInventory := targetPlayer.GuInventory
	enemyRank := targetPlayer.Rank
	enemySubRank := targetPlayer.SubRank

	hasAttitudeGu := models.InventoryHasGuMechanic(targetPlayer.GuInventory, "ARENA_IDENTITY_MASK", true)
	if hasAttitudeGu {
		enemyRank = 1
		enemySubRank = 1
		enemyGuInventory = models.GuInventory{
			{InstanceID: "attitude_mask", GuID: "GU_MOONLIGHT", Hunger: 0, Status: "ACTIVE", Equipped: true},
		}
	}
	// Build PVP Combat State (virtual mode: full power, no real cost)
	pvpCombat := models.CombatState{
		EnemyTemplateID:     targetPlayer.ID.String(),
		EnemyName:           targetPlayer.Username,
		EnemyType:           "PVP",
		CombatMode:          "virtual",
		EnemyHP:             enemyHP,
		EnemyMaxHP:          enemyHP,
		EnemyMana:           targetPlayer.MaxMana,
		EnemyMaxMana:        targetPlayer.MaxMana,
		EnemyDamage:         targetStats["attack"],
		EnemyDefense:        targetStats["defense"],
		EnemySpeed:          targetStats["speed"],
		EnemyRank:           enemyRank,
		EnemySubRank:        enemySubRank,
		EnemyGuInventory:    enemyGuInventory,
		EnemyDaoMarks:       targetPlayer.DaoMarks,
		EnemyPathMasteries:  targetPlayer.PathMasteries,
		EnemyCombatSettings: &targetPlayer.CombatSettings,
		TargetPlayerID:      targetPlayer.ID.String(),
		EnemyMasked:         hasAttitudeGu,
	}
	enemyCharFX := gamefx.Get(&targetPlayer)
	pvpCombat.EnemyPathResistance = enemyCharFX.PathResistanceBonus
	pvpCombat.EnemyMentalResistance = enemyCharFX.MentalResistanceBonus

	// Full mana for attacker in virtual mode
	player.CurrentMana = player.MaxMana

	qiGained := AddHumanQi(&player, 3.0)

	player.ActiveCombat = InitializeCombatState(tx, &pvpCombat, &player)

	// Save to DB
	_, err = tx.Exec("UPDATE players SET active_combat = $1, human_qi = $2 WHERE id = $3", player.ActiveCombat, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize arena combat"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failed"})
		return
	}

	respondActionOnly(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Khiêu chiến Võ Đài bắt đầu! Nhân Khí +%.0f.", qiGained),
	})
}
