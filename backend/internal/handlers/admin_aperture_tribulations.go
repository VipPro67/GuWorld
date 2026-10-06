package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

func AdminListApertureTribulations(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok { return }
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 100 { limit = 50 }
	if offset < 0 { offset = 0 }
	apertureID := c.Param("id")
	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM tribulation_events WHERE aperture_id=$1`, apertureID).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm lịch sử tai kiếp"}); return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT to_jsonb(e) || jsonb_build_object(
			'player_username', p.username,
			'enemies', COALESCE((
				SELECT jsonb_agg(to_jsonb(enemy) ORDER BY enemy.created_at, enemy.id)
				FROM tribulation_enemies enemy WHERE enemy.event_id=e.id
			), '[]'::jsonb)
		)
		FROM tribulation_events e JOIN players p ON p.id=e.player_id
		WHERE e.aperture_id=$1
		ORDER BY e.started_at DESC NULLS LAST, e.id
		LIMIT $2 OFFSET $3`, apertureID, limit, offset)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải lịch sử tai kiếp"}); return }
	defer rows.Close()
	items := make([]json.RawMessage, 0)
	for rows.Next() { var raw []byte; if err := rows.Scan(&raw); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc lịch sử tai kiếp"}); return }; items = append(items, json.RawMessage(raw)) }
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc lịch sử tai kiếp"}); return }
	c.JSON(http.StatusOK, gin.H{"data": items, "total": total, "limit": limit, "offset": offset})
}
