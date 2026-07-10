package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type ArenaRanking struct {
	PlayerID  string    `json:"player_id"`
	Username  string    `json:"username"`
	Rank      int       `json:"rank"`
	Points    int       `json:"points"`
	UpdatedAt time.Time `json:"updated_at"`
}

func EnsurePlayerArenaRanking(tx *sql.Tx, playerID uuid.UUID) (int, error) {
	var rank int
	err := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", playerID).Scan(&rank)
	if err == sql.ErrNoRows {
		// Assign next rank
		var maxRank int
		_ = tx.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM arena_rankings").Scan(&maxRank)
		rank = maxRank + 1
		_, err = tx.Exec("INSERT INTO arena_rankings (player_id, rank, points) VALUES ($1, $2, 1000)", playerID, rank)
		if err != nil {
			return 0, err
		}
		_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", rank, playerID)
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
	_, _ = EnsurePlayerArenaRanking(tx, authID)
	tx.Commit()

	rows, err := db.DB.Query(`
		SELECT a.player_id, p.username, a.rank, a.points, a.updated_at
		FROM arena_rankings a
		JOIN players p ON a.player_id = p.id
		ORDER BY a.rank ASC
		LIMIT 20
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}
	defer rows.Close()

	leaderboard := []ArenaRanking{}
	for rows.Next() {
		var r ArenaRanking
		if errScan := rows.Scan(&r.PlayerID, &r.Username, &r.Rank, &r.Points, &r.UpdatedAt); errScan == nil {
			leaderboard = append(leaderboard, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
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

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang ở trong trạng thái chiến đấu!"})
		return
	}

	// Ensure rankings
	pRank, _ := EnsurePlayerArenaRanking(tx, player.ID)
	tRank, _ := EnsurePlayerArenaRanking(tx, targetUUID)

	// In arena, you can only challenge players within 5 ranks above you or below you
	diff := pRank - tRank
	if diff > 5 || diff < -5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đối thủ ở thứ hạng quá xa, chỉ thách đấu được lệch tối đa 5 hạng!"})
		return
	}

	var targetPlayer models.Player
	err = tx.QueryRow(`
		SELECT id, username, rank, sub_rank, hp, max_mana, path_masteries, gu_inventory, dao_marks, current_mana, base_hp
		FROM players WHERE id = $1 FOR UPDATE`, targetUUID).Scan(
		&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
		&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks, &targetPlayer.CurrentMana, &targetPlayer.BaseHP,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đối thủ"})
		return
	}

	targetStats := CalculatePlayerStats(&targetPlayer)

	enemyHP := targetPlayer.BaseHP
	if enemyHP <= 0 {
		enemyHP = 100.0
	}

	enemyGuInventory := targetPlayer.GuInventory
	enemyRank := targetPlayer.Rank
	enemySubRank := targetPlayer.SubRank

	hasAttitudeGu := false
	for _, gu := range targetPlayer.GuInventory {
		if gu.GuID == "GU_ATTITUDE_GU" && gu.Equipped && gu.Status == "ACTIVE" {
			hasAttitudeGu = true
			break
		}
	}
	if hasAttitudeGu {
		enemyRank = 1
		enemySubRank = 1
		enemyGuInventory = models.GuInventory{
			{InstanceID: "attitude_mask", GuID: "GU_MOONLIGHT", Hunger: 0, Status: "ACTIVE", Equipped: true},
		}
	}

	// Build PVP Combat State
	pvpCombat := models.CombatState{
		EnemyTemplateID:    targetPlayer.ID.String(),
		EnemyName:          targetPlayer.Username,
		EnemyType:          "PVP",
		EnemyHP:            enemyHP,
		EnemyMaxHP:         enemyHP,
		EnemyMana:          targetPlayer.CurrentMana,
		EnemyMaxMana:       targetPlayer.MaxMana,
		EnemyDamage:        targetStats["attack"],
		EnemyDefense:       targetStats["defense"],
		EnemyRank:          enemyRank,
		EnemySubRank:       enemySubRank,
		EnemyGuInventory:   enemyGuInventory,
		EnemyDaoMarks:      targetPlayer.DaoMarks,
		EnemyPathMasteries: targetPlayer.PathMasteries,
		TargetPlayerID:     targetPlayer.ID.String(),
	}

	// Swapping rank logic after battle is implemented inside CombatAttack when victory is achieved!
	// Wait, we need to handle swapping rank logic in CombatAttack when the PVP victory happens!
	// Let's check how we handle PVP victory in CombatAttack. Currently, it steals stones.
	// If the battle was an Arena battle (PVP target and arena), we swap ranks!
	// To distinguish normal PVP from Arena PVP, let's mark it by checking if both have rankings.
	// Let's add rank swap helper inside CombatAttack victory check.
	// If Victory and target is a PVP player:
	// We check if it's an arena match, swap ranks if pRank > tRank.
	
	player.ActiveCombat = InitializeCombatState(tx, &pvpCombat, &player)

	// Save to DB
	_, err = tx.Exec("UPDATE players SET active_combat = $1 WHERE id = $2", player.ActiveCombat, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize arena combat"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Khiêu chiến Võ Đài bắt đầu!",
		"player":  player,
	})
}
