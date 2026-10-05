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
)

func AdminListCombatHistory(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	page, limit := adminPage(c, 25, 100)
	search := strings.TrimSpace(c.Query("search"))
	pattern := "%" + search + "%"
	filter := `UPPER(h.enemy_type) = 'PVP' AND ($1 = '' OR h.id::text ILIKE $2 OR h.player_id::text ILIKE $2 OR p.username ILIKE $2 OR h.enemy_name ILIKE $2 OR h.region ILIKE $2 OR h.winner ILIKE $2)`

	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*) FROM combat_history h
		LEFT JOIN players p ON p.id = h.player_id
		WHERE `+filter, search, pattern).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm lịch sử PVP"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT h.id, h.player_id, COALESCE(p.username, ''), h.winner, h.summary,
		       h.enemy_name, h.enemy_type, h.region,
		       COALESCE(jsonb_array_length(h.logs), 0), h.created_at
		FROM combat_history h
		LEFT JOIN players p ON p.id = h.player_id
		WHERE `+filter+`
		ORDER BY h.created_at DESC, h.id
		LIMIT $3 OFFSET $4`, search, pattern, limit, (page-1)*limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải lịch sử PVP"})
		return
	}
	defer rows.Close()

	data := make([]gin.H, 0, limit)
	for rows.Next() {
		var id, playerID, username, winner, enemyName, enemyType, region string
		var summary []byte
		var logsCount int
		var createdAt time.Time
		if err := rows.Scan(&id, &playerID, &username, &winner, &summary, &enemyName, &enemyType, &region, &logsCount, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc lịch sử PVP"})
			return
		}
		data = append(data, gin.H{"id": id, "player_id": playerID, "player_username": username, "winner": winner, "summary": json.RawMessage(summary), "enemy_name": enemyName, "enemy_type": enemyType, "region": region, "logs_count": logsCount, "created_at": createdAt})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc lịch sử PVP"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "total": total, "page": page, "limit": limit})
}

func AdminGetCombatHistory(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID lịch sử combat không hợp lệ"})
		return
	}
	var playerID, username, winner, enemyName, enemyType, region string
	var logs, summary, combatants []byte
	var createdAt time.Time
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT h.player_id, COALESCE(p.username, ''), h.winner, h.logs, h.summary,
		       h.combatants, h.enemy_name, h.enemy_type, h.region, h.created_at
		FROM combat_history h
		LEFT JOIN players p ON p.id = h.player_id
		WHERE h.id = $1 AND UPPER(h.enemy_type) = 'PVP'`, id,
	).Scan(&playerID, &username, &winner, &logs, &summary, &combatants, &enemyName, &enemyType, &region, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy lịch sử PVP"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải chi tiết lịch sử PVP"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "player_id": playerID, "player_username": username, "winner": winner, "logs": json.RawMessage(logs), "summary": json.RawMessage(summary), "combatants": json.RawMessage(combatants), "enemy_name": enemyName, "enemy_type": enemyType, "region": region, "created_at": createdAt})
}

func adminPage(c *gin.Context, defaultLimit, maxLimit int) (int, int) {
	page, limit := 1, defaultLimit
	_, _ = fmt.Sscanf(c.DefaultQuery("page", "1"), "%d", &page)
	_, _ = fmt.Sscanf(c.DefaultQuery("limit", fmt.Sprint(defaultLimit)), "%d", &limit)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}
	return page, limit
}
