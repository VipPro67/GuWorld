package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// GetGuHistory returns the complete lifecycle history timeline for a Gu instance.
func GetGuHistory(c *gin.Context) {
	instanceID := c.Param("instance_id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id parameter is required"})
		return
	}

	logs, err := db.FetchGuHistoryLogs(db.DB, instanceID)
	if err != nil {
		logger.S.Errorf("GetGuHistory failed for instance_id %s: %v", instanceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Gu history logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": instanceID,
		"total":       len(logs),
		"history":     logs,
	})
}

// AdminListGuHistory lists gu_history_logs with pagination and search for the Admin panel.
func AdminListGuHistory(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	search := c.Query("search")
	eventType := c.Query("event_type")

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		like := "%" + search + "%"
		whereClause += fmt.Sprintf(" AND (g.instance_id ILIKE $%d OR g.gu_id ILIKE $%d OR g.summary ILIKE $%d OR pa.username ILIKE $%d OR pt.username ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx)
		args = append(args, like)
		argIdx++
	}

	if eventType != "" {
		whereClause += fmt.Sprintf(" AND g.event_type = $%d", argIdx)
		args = append(args, eventType)
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM gu_history_logs g
		LEFT JOIN players pa ON g.actor_id = pa.id
		LEFT JOIN players pt ON g.target_id = pt.id
		%s
	`, whereClause)

	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count gu history logs"})
		return
	}

	query := fmt.Sprintf(`
		SELECT g.id, g.instance_id, g.gu_id, g.rank, g.event_type,
		       g.actor_id, COALESCE(pa.username, ''),
		       g.target_id, COALESCE(pt.username, ''),
		       g.summary, g.metadata, g.created_at
		FROM gu_history_logs g
		LEFT JOIN players pa ON g.actor_id = pa.id
		LEFT JOIN players pt ON g.target_id = pt.id
		%s
		ORDER BY g.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, limit, offset)
	rows, err := db.DB.QueryContext(c.Request.Context(), query, queryArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query gu history logs"})
		return
	}
	defer rows.Close()

	var logs []db.GuHistoryLog
	for rows.Next() {
		var l db.GuHistoryLog
		var actorID, targetID sql.NullString
		var metaRaw []byte
		err := rows.Scan(
			&l.ID, &l.InstanceID, &l.GuID, &l.Rank, &l.EventType,
			&actorID, &l.ActorName,
			&targetID, &l.TargetName,
			&l.Summary, &metaRaw, &l.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning gu history row"})
			return
		}
		if actorID.Valid {
			s := actorID.String
			l.ActorID = &s
		}
		if targetID.Valid {
			s := targetID.String
			l.TargetID = &s
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &l.Metadata)
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []db.GuHistoryLog{}
	}

	c.JSON(http.StatusOK, gin.H{
		"page":  page,
		"limit": limit,
		"total": total,
		"logs":  logs,
	})
}
