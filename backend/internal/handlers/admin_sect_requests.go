package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func AdminListSectRequests(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok { return }
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT request FROM (
			SELECT r.created_at, jsonb_build_object('id',r.id,'request_type','join','sect_id',r.sect_id,'player_id',r.player_id,'username',p.username,'avatar_url',p.avatar_url,'player_rank',p.rank,'current_rank','','target_rank','','status',r.status,'created_at',r.created_at) AS request
			FROM sect_join_requests r JOIN players p ON p.id=r.player_id WHERE r.sect_id=$1
			UNION ALL
			SELECT r.created_at, jsonb_build_object('id',r.id,'request_type','promotion','sect_id',r.sect_id,'player_id',r.player_id,'username',p.username,'avatar_url',p.avatar_url,'player_rank',p.rank,'current_rank',r.current_rank,'target_rank',r.target_rank,'status',r.status,'created_at',r.created_at) AS request
			FROM sect_promotion_requests r JOIN players p ON p.id=r.player_id WHERE r.sect_id=$1
		) requests ORDER BY created_at`, c.Param("id"))
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải yêu cầu tông môn"}); return }
	defer rows.Close()
	items := make([]json.RawMessage, 0)
	for rows.Next() { var raw []byte; if err := rows.Scan(&raw); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc yêu cầu tông môn"}); return }; items = append(items, json.RawMessage(raw)) }
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách yêu cầu"}); return }
	c.JSON(http.StatusOK, items)
}

func AdminDeleteSectRequest(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	var table string
	switch c.Param("request_type") { case "join": table = "sect_join_requests"; case "promotion": table = "sect_promotion_requests"; default: c.JSON(http.StatusBadRequest, gin.H{"error": "Loại yêu cầu không hợp lệ"}); return }
	result, err := db.DB.ExecContext(c.Request.Context(), "DELETE FROM "+table+" WHERE id=$1 AND sect_id=$2 AND status='PENDING'", c.Param("request_id"), c.Param("id"))
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hủy yêu cầu tông môn"}); return }
	n, err := result.RowsAffected(); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận thao tác"}); return }
	if n == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy yêu cầu đang chờ trong tông môn này"}); return }
	auditlog.Logf(db.DB, &adminID, "admin.delete_sect_request", "sect=%s type=%s request=%s", c.Param("id"), c.Param("request_type"), c.Param("request_id"))
	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy yêu cầu đang chờ"})
}
