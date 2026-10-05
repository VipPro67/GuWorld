package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type AuditLogResponse struct {
	ID             string                 `json:"id"`
	PlayerID       string                 `json:"player_id"`
	ActorUsername  string                 `json:"actor_username"`
	Action         string                 `json:"action"`
	EffectiveRoles []string               `json:"effective_roles"`
	Permission     string                 `json:"effective_permission"`
	EntityType     *string                `json:"entity_type"`
	EntityID       *string                `json:"entity_id"`
	ChangeData     interface{}            `json:"change_data"`
	Reason         string                 `json:"reason"`
	RequestID      string                 `json:"request_id"`
	RequestSummary string                 `json:"request_summary"`
	Metadata       map[string]interface{} `json:"metadata"`
	IPAddress      *string                `json:"ip_address"`
	Success        bool                   `json:"success"`
	ErrorMessage   *string                `json:"error_message"`
	CreatedAt      string                 `json:"created_at"`
}

// GetMyAuditLogs returns the authenticated player's audit logs with support for limit, offset, and action filtering
func GetMyAuditLogs(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	action := c.Query("action")
	entityType := c.Query("entity_type")
	successStr := c.Query("success")
	search := c.Query("search")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, COALESCE(player_id::text, ''), actor_username, action, effective_roles, effective_permission, entity_type, entity_id, change_data, reason, request_id, COALESCE(request_summary, ''), metadata, ip_address, success, error_message, created_at
		FROM audit_logs
		WHERE player_id = $1
	`
	args := []interface{}{playerID}
	argCount := 1

	if action != "" {
		argCount++
		query += " AND action = $" + strconv.Itoa(argCount)
		args = append(args, action)
	}

	if entityType != "" {
		argCount++
		query += " AND entity_type = $" + strconv.Itoa(argCount)
		args = append(args, entityType)
	}

	if successStr != "" {
		success, err := strconv.ParseBool(successStr)
		if err == nil {
			argCount++
			query += " AND success = $" + strconv.Itoa(argCount)
			args = append(args, success)
		}
	}

	if search != "" {
		argCount++
		query += " AND (action ILIKE $" + strconv.Itoa(argCount) + " OR request_summary ILIKE $" + strconv.Itoa(argCount) + " OR entity_id ILIKE $" + strconv.Itoa(argCount) + " OR error_message ILIKE $" + strconv.Itoa(argCount) + ")"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query audit logs: " + err.Error()})
		return
	}
	defer rows.Close()

	logs := []AuditLogResponse{}
	for rows.Next() {
		var logRow AuditLogResponse
		var metaJSON, rolesJSON, changeJSON []byte
		err := rows.Scan(
			&logRow.ID, &logRow.PlayerID, &logRow.ActorUsername, &logRow.Action, &rolesJSON, &logRow.Permission, &logRow.EntityType,
			&logRow.EntityID, &changeJSON, &logRow.Reason, &logRow.RequestID, &logRow.RequestSummary, &metaJSON, &logRow.IPAddress,
			&logRow.Success, &logRow.ErrorMessage, &logRow.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan audit log: " + err.Error()})
			return
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &logRow.Metadata)
		}
		_ = json.Unmarshal(rolesJSON, &logRow.EffectiveRoles)
		_ = json.Unmarshal(changeJSON, &logRow.ChangeData)
		logs = append(logs, logRow)
	}

	c.JSON(http.StatusOK, logs)
}
