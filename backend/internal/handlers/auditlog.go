package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type AuditLogResponse struct {
	ID               string                 `json:"id"`
	PlayerID         string                 `json:"player_id"`
	AffectedPlayerID *string                `json:"affected_player_id"`
	ActorUsername    string                 `json:"actor_username"`
	Action           string                 `json:"action"`
	EffectiveRoles   []string               `json:"effective_roles"`
	Permission       string                 `json:"effective_permission"`
	EntityType       *string                `json:"entity_type"`
	EntityID         *string                `json:"entity_id"`
	ChangeData       interface{}            `json:"change_data"`
	Reason           string                 `json:"reason"`
	RequestID        string                 `json:"request_id"`
	RequestSummary   string                 `json:"request_summary"`
	Metadata         map[string]interface{} `json:"metadata"`
	IPAddress        *string                `json:"ip_address"`
	Success          bool                   `json:"success"`
	ErrorMessage     *string                `json:"error_message"`
	CreatedAt        string                 `json:"created_at"`
}

func buildAuditFilters(action, entityType string, playerID *uuid.UUID, success *bool, search string) (string, []interface{}) {
	clauses := make([]string, 0, 5)
	args := make([]interface{}, 0, 5)
	add := func(clause string, value interface{}) string {
		args = append(args, value)
		placeholder := "$" + strconv.Itoa(len(args))
		clauses = append(clauses, strings.ReplaceAll(clause, "?", placeholder))
		return placeholder
	}
	if playerID != nil {
		args = append(args, *playerID)
		placeholder := "$" + strconv.Itoa(len(args))
		clauses = append(clauses, "(player_id = "+placeholder+" OR affected_player_id = "+placeholder+")")
	}
	if action != "" {
		add("action = ?", action)
	}
	if entityType != "" {
		add("entity_type = ?", entityType)
	}
	if success != nil {
		add("success = ?", *success)
	}
	if search != "" {
		add("(action ILIKE ? OR request_summary ILIKE ? OR entity_id ILIKE ? OR error_message ILIKE ?)", "%"+search+"%")
	}
	if len(clauses) == 0 {
		return " WHERE 1=1", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func scanAuditLogRows(rows *sql.Rows) ([]AuditLogResponse, error) {
	logs := []AuditLogResponse{}
	for rows.Next() {
		var logRow AuditLogResponse
		var metaJSON, rolesJSON, changeJSON []byte
		if err := rows.Scan(&logRow.ID, &logRow.PlayerID, &logRow.AffectedPlayerID, &logRow.ActorUsername, &logRow.Action, &rolesJSON, &logRow.Permission, &logRow.EntityType, &logRow.EntityID, &changeJSON, &logRow.Reason, &logRow.RequestID, &logRow.RequestSummary, &metaJSON, &logRow.IPAddress, &logRow.Success, &logRow.ErrorMessage, &logRow.CreatedAt); err != nil {
			return nil, err
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &logRow.Metadata)
		}
		_ = json.Unmarshal(rolesJSON, &logRow.EffectiveRoles)
		_ = json.Unmarshal(changeJSON, &logRow.ChangeData)
		logs = append(logs, logRow)
	}
	return logs, rows.Err()
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

	var success *bool
	if parsed, parseErr := strconv.ParseBool(successStr); successStr != "" && parseErr == nil {
		success = &parsed
	}
	whereClause, args := buildAuditFilters(action, entityType, &playerID, success, search)
	query := `SELECT id, COALESCE(player_id::text, ''), affected_player_id::text, actor_username, action, effective_roles, effective_permission, entity_type, entity_id, change_data, reason, request_id, COALESCE(request_summary, ''), metadata, ip_address, success, error_message, created_at FROM audit_logs` + whereClause
	argCount := len(args)

	query += " ORDER BY created_at DESC, id DESC LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)
	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to query audit logs: "+err.Error(), err))
		return
	}
	defer rows.Close()

	logs, err := scanAuditLogRows(rows)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to scan audit logs: "+err.Error(), err))
		return
	}

	c.JSON(http.StatusOK, logs)
}
