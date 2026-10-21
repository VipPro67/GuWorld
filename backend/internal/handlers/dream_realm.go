package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// ListDreamRealms returns all DREAM_REALM templates with per-player cooldown info.
func ListDreamRealms(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Pull only DREAM_REALM type from the shared instance_templates table.
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, name, type, required_rank, entry_item, description, rewards, cooldown_hours
		FROM instance_templates
		WHERE type = 'DREAM_REALM'
		ORDER BY required_rank ASC, id ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách mộng cảnh: " + err.Error()})
		return
	}
	defer rows.Close()

	var items []InstanceListItem
	for rows.Next() {
		var it InstanceListItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Type, &it.RequiredRank, &it.EntryItem, &it.Description, &it.Rewards, &it.CooldownHours); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc mộng cảnh: " + err.Error()})
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi duyệt danh sách mộng cảnh: " + err.Error()})
		return
	}

	// Enrich with cooldown remaining per instance for this player.
	cdMap := map[string]int64{}
	cdRows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT s.instance_id,
		       GREATEST(0, EXTRACT(EPOCH FROM (it.cooldown_hours::interval - (NOW() - s.created_at))))::int
		FROM player_instance_sessions s
		JOIN instance_templates it ON it.id = s.instance_id
		WHERE s.player_id = $1
		  AND it.type = 'DREAM_REALM'
		ORDER BY s.instance_id, s.created_at DESC
	`, playerID)
	if err == nil {
		defer cdRows.Close()
		for cdRows.Next() {
			var instID string
			var remain int64
			if err := cdRows.Scan(&instID, &remain); err != nil {
				continue
			}
			cdMap[instID] = remain
		}
	}
	for i := range items {
		items[i].CooldownRemaining = int(cdMap[items[i].ID])
	}

	// Count daily mộng cảnh entries today.
	var dailyUsed int
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*)
		FROM player_instance_sessions s
		JOIN instance_templates it ON it.id = s.instance_id
		WHERE s.player_id = $1
		  AND s.created_at >= CURRENT_DATE
		  AND it.type = 'DREAM_REALM'
	`, playerID).Scan(&dailyUsed)

	c.JSON(http.StatusOK, gin.H{
		"instances":          items,
		"daily_entries_used": dailyUsed,
		"daily_entries_max":  MaxDailyInstanceEntries,
	})
}

// GetActiveDreamSession retrieves the player's active DREAM_REALM session (if any).
func GetActiveDreamSession(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var session PlayerInstanceSession
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT s.id, s.player_id, s.instance_id, s.current_stage, s.status, s.temp_inventory
		FROM player_instance_sessions s
		JOIN instance_templates it ON it.id = s.instance_id
		WHERE s.player_id = $1
		  AND s.status = 'ACTIVE'
		  AND it.type = 'DREAM_REALM'
		LIMIT 1
	`, playerID).Scan(
		&session.ID, &session.PlayerID, &session.InstanceID,
		&session.CurrentStage, &session.Status, &session.TempInventory,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{"active": false})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn phiên mộng cảnh: " + err.Error()})
		}
		return
	}

	template, found := getCachedInstanceTemplate(session.InstanceID)
	if !found {
		err = db.DB.QueryRowContext(c.Request.Context(), `
			SELECT id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours
			FROM instance_templates
			WHERE id = $1 AND type = 'DREAM_REALM'
		`, session.InstanceID).Scan(
			&template.ID, &template.Name, &template.Type, &template.RequiredRank,
			&template.EntryItem, &template.Description, &template.Rewards, &template.MapData,
			&template.CooldownHours,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải mẫu mộng cảnh: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"active":   true,
		"session":  session,
		"template": template,
	})
}

// EnterDreamRealm creates a new DREAM_REALM session for the player.
// It delegates to the shared entry flow, restricted to DREAM_REALM templates.
func EnterDreamRealm(c *gin.Context) {
	var req EnterInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id là bắt buộc"})
		return
	}

	// Delegate to the shared entry flow, restricted to DREAM_REALM templates.
	// Calling EnterInstance(c) here would fail because the JSON body has
	// already been consumed by the binding above.
	enterInstanceWithRequest(c, req, "DREAM_REALM")
}

// ActionDreamRealm processes a player action inside an active DREAM_REALM session.
// It ensures the active session belongs to a DREAM_REALM template, then delegates.
func ActionDreamRealm(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify active session is a DREAM_REALM.
	var sessionInstanceID string
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT s.instance_id
		FROM player_instance_sessions s
		JOIN instance_templates it ON it.id = s.instance_id
		WHERE s.player_id = $1
		  AND s.status = 'ACTIVE'
		  AND it.type = 'DREAM_REALM'
		LIMIT 1
	`, playerID).Scan(&sessionInstanceID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có mộng cảnh nào đang hoạt động!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
		}
		return
	}

	// Delegate to shared ActionInstance handler.
	ActionInstance(c)
}

// ExitDreamRealm allows a player to abandon (fail) their active DREAM_REALM session.
func ExitDreamRealm(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi bắt đầu giao dịch"})
		return
	}
	defer tx.Rollback()

	var session PlayerInstanceSession
	err = tx.QueryRow(`
		SELECT s.id, s.player_id, s.instance_id, s.current_stage, s.status, s.temp_inventory
		FROM player_instance_sessions s
		JOIN instance_templates it ON it.id = s.instance_id
		WHERE s.player_id = $1
		  AND s.status = 'ACTIVE'
		  AND it.type = 'DREAM_REALM'
		LIMIT 1
	`, playerID).Scan(
		&session.ID, &session.PlayerID, &session.InstanceID,
		&session.CurrentStage, &session.Status, &session.TempInventory,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có mộng cảnh nào đang hoạt động!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
		}
		return
	}

	template, _ := getCachedInstanceTemplate(session.InstanceID)

	_, err = tx.Exec(`
		UPDATE player_instance_sessions SET status = 'FAILED' WHERE id = $1
	`, session.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thoát mộng cảnh: " + err.Error()})
		return
	}

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải dữ liệu nhân vật: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "dream_realm.exit",
		EntityType:     "instance_template",
		EntityID:       session.InstanceID,
		RequestSummary: fmt.Sprintf("Đạo hữu thoát khỏi mộng cảnh %s sớm", template.Name),
		Metadata: map[string]interface{}{
			"instance_id":   session.InstanceID,
			"instance_name": template.Name,
			"stage":         session.CurrentStage,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": "Đã thoát khỏi mộng cảnh thành công!",
	})
}
