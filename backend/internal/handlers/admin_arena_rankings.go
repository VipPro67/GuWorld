package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func AdminListArenaRankings(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	page, limit := adminPage(c, 25, 100)
	search := strings.TrimSpace(c.Query("search"))
	pattern := "%" + search + "%"
	filter := `($1 = '' OR a.player_id::text ILIKE $2 OR p.username ILIKE $2)`
	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM arena_rankings a JOIN players p ON p.id=a.player_id WHERE `+filter, search, pattern).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm xếp hạng đấu trường"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT a.player_id, p.username, a.rank, a.points, a.daily_challenges, a.last_challenge_at, a.updated_at
		FROM arena_rankings a JOIN players p ON p.id=a.player_id
		WHERE `+filter+`
		ORDER BY a.rank ASC, a.player_id
		LIMIT $3 OFFSET $4`, search, pattern, limit, (page-1)*limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải xếp hạng đấu trường"})
		return
	}
	defer rows.Close()
	data := make([]gin.H, 0, limit)
	for rows.Next() {
		var playerID, username string
		var rank, points, dailyChallenges int
		var lastChallengeAt, updatedAt sql.NullTime
		if err := rows.Scan(&playerID, &username, &rank, &points, &dailyChallenges, &lastChallengeAt, &updatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc xếp hạng đấu trường"})
			return
		}
		data = append(data, gin.H{"player_id": playerID, "username": username, "rank": rank, "points": points, "daily_challenges": dailyChallenges, "last_challenge_at": arenaNullableTime(lastChallengeAt), "updated_at": arenaNullableTime(updatedAt)})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc xếp hạng đấu trường"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "total": total, "page": page, "limit": limit})
}

type adminArenaRankingUpdate struct {
	Points          *int    `json:"points"`
	DailyChallenges *int    `json:"daily_challenges"`
	LastChallengeAt *string `json:"last_challenge_at"`
	Reason          string  `json:"reason"`
}

func AdminUpdateArenaRanking(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("player_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	var req adminArenaRankingUpdate
	if err := c.ShouldBindJSON(&req); err != nil || req.Points == nil || req.DailyChallenges == nil || req.LastChallengeAt == nil || strings.TrimSpace(req.Reason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần đủ điểm, số lượt thử, thời điểm thử gần nhất và lý do"})
		return
	}
	if *req.Points < 0 || *req.DailyChallenges < 0 || *req.DailyChallenges > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm không được âm; số lượt thử phải từ 0 đến 10"})
		return
	}
	lastChallengeAt, err := time.Parse(time.RFC3339, *req.LastChallengeAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời điểm thử gần nhất không hợp lệ"})
		return
	}
	var rank int
	err = db.DB.QueryRowContext(c.Request.Context(), `UPDATE arena_rankings SET points=$1, daily_challenges=$2, last_challenge_at=$3, updated_at=NOW() WHERE player_id=$4 RETURNING rank`, *req.Points, *req.DailyChallenges, lastChallengeAt, playerID).Scan(&rank)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Người chơi chưa có hạng đấu trường"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật xếp hạng đấu trường"})
		}
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.update_arena_ranking", EntityType: "arena_rankings", EntityID: playerID.String(), RequestSummary: fmt.Sprintf("rank=%d points=%d daily_challenges=%d last_challenge_at=%s reason=%s", rank, *req.Points, *req.DailyChallenges, lastChallengeAt.Format(time.RFC3339), strings.TrimSpace(req.Reason)), IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật dữ liệu đấu trường", "rank": rank})
}

func arenaNullableTime(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time
}
