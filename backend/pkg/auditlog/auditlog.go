package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
)

type Querier interface {
	Exec(string, ...interface{}) (sql.Result, error)
}

type Entry struct {
	PlayerID            *uuid.UUID
	AffectedPlayerID    *uuid.UUID
	ActorUsername       string
	Action              string
	EffectiveRoles      []string
	EffectivePermission string
	EntityType          string
	EntityID            string
	ChangeData          interface{}
	Reason              string
	RequestID           string
	RequestSummary      string
	Metadata            map[string]interface{}
	IPAddress           string
	Success             bool
	ErrorMessage        string
}

// LogContext copies the request trace ID into an entry before persisting it.
// Explicit RequestID values take precedence for callers with operation IDs.
func LogContext(ctx context.Context, q Querier, entry Entry) {
	if entry.RequestID == "" {
		entry.RequestID = logger.TraceIDFromContext(ctx)
	}
	Log(q, entry)
}

// LogCheckedContext is the context-aware form of LogChecked.
func LogCheckedContext(ctx context.Context, q Querier, entry Entry) error {
	if entry.RequestID == "" {
		entry.RequestID = logger.TraceIDFromContext(ctx)
	}
	return LogChecked(q, entry)
}

func Log(q Querier, entry Entry) {
	if err := LogChecked(q, entry); err != nil {
		fields := []zap.Field{zap.String("action", entry.Action), zap.Any("player_id", entry.PlayerID), zap.String("request_id", entry.RequestID)}
		logger.LogError(nil, "audit_write_failed", err, fields...)
	}
}

func recordWriteFailure(entry Entry, err error) {
	monitor.GlobalAdminOperations.RecordAuditWriteFailure()
}

// LogChecked records an audit entry and returns persistence failures to
// callers that require the audit write to be atomic with their transaction.
func LogChecked(q Querier, entry Entry) error {
	var metaBytes []byte
	if entry.Metadata != nil {
		var err error
		metaBytes, err = json.Marshal(entry.Metadata)
		if err != nil {
			err = fmt.Errorf("marshal audit metadata: %w", err)
			recordWriteFailure(entry, err)
			return err
		}
	}
	metaJSON := string(metaBytes)
	if metaJSON == "" {
		metaJSON = "{}"
	}
	var pid interface{}
	if entry.PlayerID != nil {
		pid = *entry.PlayerID
	}
	var affectedPID interface{}
	if entry.AffectedPlayerID != nil {
		affectedPID = *entry.AffectedPlayerID
	}
	rolesJSON, err := json.Marshal(entry.EffectiveRoles)
	if err != nil {
		err = fmt.Errorf("marshal audit roles: %w", err)
		recordWriteFailure(entry, err)
		return err
	}
	changeJSON, err := json.Marshal(entry.ChangeData)
	if err != nil {
		err = fmt.Errorf("marshal audit change: %w", err)
		recordWriteFailure(entry, err)
		return err
	}
	if string(changeJSON) == "null" {
		changeJSON = []byte("{}")
	}

	execDB := q
	if execDB == nil && db.DB != nil {
		execDB = db.DB
	}
	if execDB == nil {
		err := fmt.Errorf("auditlog: nil querier for action=%s", entry.Action)
		recordWriteFailure(entry, err)
		return err
	}

	const insertSQL = `INSERT INTO audit_logs (player_id, affected_player_id, actor_username, action, effective_roles, effective_permission, entity_type, entity_id, change_data, reason, request_id, request_summary, metadata, ip_address, success, error_message, created_at) VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13::jsonb,$14,$15,$16,$17)`

	if tx, ok := execDB.(*sql.Tx); ok {
		if _, err := tx.Exec("SAVEPOINT auditlog_insert"); err != nil {
			recordWriteFailure(entry, err)
			return err
		}
		if _, err := tx.Exec(
			insertSQL,
			pid, affectedPID, entry.ActorUsername, entry.Action, string(rolesJSON), entry.EffectivePermission, entry.EntityType, entry.EntityID, string(changeJSON), entry.Reason, entry.RequestID, entry.RequestSummary, metaJSON, entry.IPAddress, entry.Success, entry.ErrorMessage, time.Now().UTC(),
		); err != nil {
			recordWriteFailure(entry, err)
			_, _ = tx.Exec("ROLLBACK TO SAVEPOINT auditlog_insert")
			_, _ = tx.Exec("RELEASE SAVEPOINT auditlog_insert")
			return err
		}
		if _, err := tx.Exec("RELEASE SAVEPOINT auditlog_insert"); err != nil {
			recordWriteFailure(entry, err)
			return err
		}
		return nil
	}

	_, err = execDB.Exec(
		insertSQL,
		pid, affectedPID, entry.ActorUsername, entry.Action, string(rolesJSON), entry.EffectivePermission, entry.EntityType, entry.EntityID, string(changeJSON), entry.Reason, entry.RequestID, entry.RequestSummary, metaJSON, entry.IPAddress, entry.Success, entry.ErrorMessage, time.Now().UTC(),
	)
	if err != nil {
		recordWriteFailure(entry, err)
		return err
	}
	return nil
}

func Logf(q Querier, playerID *uuid.UUID, action, format string, args ...interface{}) {
	Log(q, Entry{
		PlayerID:       playerID,
		Action:         action,
		RequestSummary: fmt.Sprintf(format, args...),
		Success:        true,
	})
}
