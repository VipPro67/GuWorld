package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func AdminListSectDefenders(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok { return }
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT to_jsonb(d) || jsonb_build_object(
			'player_username', COALESCE(p.username, ''),
			'monster_name', COALESCE(m.name, '')
		) FROM sect_defenders d
		LEFT JOIN players p ON p.id=d.player_id
		LEFT JOIN monster_templates m ON m.id=d.monster_id
		WHERE d.sect_id=$1 ORDER BY d.assigned_at, d.id`, c.Param("id"))
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách phòng thủ tông môn"}); return }
	defer rows.Close()
	items := make([]json.RawMessage, 0)
	for rows.Next() { var row []byte; if err := rows.Scan(&row); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu phòng thủ"}); return }; items = append(items, json.RawMessage(row)) }
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách phòng thủ"}); return }
	c.JSON(http.StatusOK, items)
}

func AdminRemoveSectDefender(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM sect_defenders WHERE id=$1 AND sect_id=$2`, c.Param("defender_id"), c.Param("id"))
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể rút lực lượng phòng thủ"}); return }
	n, err := result.RowsAffected(); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận thao tác"}); return }
	if n == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy lực lượng phòng thủ trong tông môn này"}); return }
	auditlog.Logf(db.DB, &adminID, "admin.remove_sect_defender", "sect=%s defender=%s", c.Param("id"), c.Param("defender_id"))
	c.JSON(http.StatusOK, gin.H{"message": "Đã rút lực lượng phòng thủ"})
}
