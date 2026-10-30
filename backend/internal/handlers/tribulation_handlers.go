package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// GetActiveTribulation returns the active tribulation event and its enemies for the player
func GetActiveTribulation(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 1. Fetch active event lock-free. Keep the same recency order as the
	// realtime event loader so the panel and the joined map never disagree.
	var ev models.TribulationEvent
	var endedAt sql.NullTime
	var hpAfter sql.NullFloat64
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, aperture_id, player_id, tribulation_type, path, status, aperture_hp_before, aperture_hp_after, reward_dao_marks, started_at, ended_at
		FROM tribulation_events
		WHERE player_id = $1 AND status = 'ACTIVE'
		ORDER BY started_at DESC
		LIMIT 1
	`, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.Status, &ev.ApertureHPBefore, &hpAfter, &ev.RewardDaoMarks, &ev.StartedAt, &endedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, nil)
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn kiếp nạn"})
		return
	}

	// If an active event exists, check if dead aperture suppression applies (zombie/undead immortals)
	var race string
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(race, '') FROM players WHERE id = $1`, playerID).Scan(&race)
	if raceHasDeadAperture(race) {
		tx, errTx := db.DB.Begin()
		if errTx == nil {
			defer tx.Rollback()
			if suppressed, errS := suppressTribulationsForDeadAperture(tx, playerID.String()); errS == nil && suppressed {
				_ = tx.Commit()
				c.JSON(http.StatusOK, nil)
				return
			}
		}
	}

	if hpAfter.Valid {
		ev.ApertureHPAfter = &hpAfter.Float64
	}
	if endedAt.Valid {
		ev.EndedAt = &endedAt.Time
	}

	// 2. Fetch enemies lock-free
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, event_id, name, enemy_type, path, hp, max_hp, attack, defense, target_node_id, x, y, is_boss, is_dead, rank, sub_rank
		FROM tribulation_enemies
		WHERE event_id = $1 AND is_dead = false
	`, ev.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy danh sách quái tai kiếp"})
		return
	}
	defer rows.Close()

	enemies := []models.TribulationEnemy{}
	for rows.Next() {
		var en models.TribulationEnemy
		var targetID sql.NullString
		err = rows.Scan(&en.ID, &en.EventID, &en.Name, &en.EnemyType, &en.Path, &en.HP, &en.MaxHP, &en.Attack, &en.Defense, &targetID, &en.X, &en.Y, &en.IsBoss, &en.IsDead, &en.Rank, &en.SubRank)
		if err == nil {
			if targetID.Valid {
				en.TargetNodeID = &targetID.String
			}
			enemies = append(enemies, en)
		}
	}

	var totalEnemies int
	if err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1
	`, ev.ID).Scan(&totalEnemies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đếm quái tai kiếp"})
		return
	}

	// The live realtime snapshot is authoritative while the player is inside
	// the encounter. Hide enemies that are already dead there even if the
	// durable is_dead write has not landed yet.
	if _, dead, ok := middleware.RealtimeTribulationEnemies(playerID); ok && len(dead) > 0 {
		live := enemies[:0]
		for _, en := range enemies {
			if dead[en.ID] {
				continue
			}
			live = append(live, en)
		}
		enemies = live
	}

	remaining := len(enemies)
	defeated := totalEnemies - remaining
	if defeated < 0 {
		defeated = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"event":             ev,
		"enemies":           enemies,
		"total_enemies":     totalEnemies,
		"defeated_enemies":  defeated,
		"remaining_enemies": remaining,
	})
}
