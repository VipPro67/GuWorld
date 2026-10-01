package auditlog

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type Querier interface {
	Exec(string, ...interface{}) (sql.Result, error)
}

type Entry struct {
	PlayerID       *uuid.UUID
	Action         string
	EntityType     string
	EntityID       string
	RequestSummary string
	Metadata       map[string]interface{}
	IPAddress      string
	Success        bool
	ErrorMessage   string
}

func Log(q Querier, entry Entry) {
	if err := LogChecked(q, entry); err != nil {
		logger.S.Warnf("auditlog: failed to insert action=%s player=%v: %v", entry.Action, entry.PlayerID, err)
	}
}

// LogChecked records an audit entry and returns persistence failures to
// callers that require the audit write to be atomic with their transaction.
func LogChecked(q Querier, entry Entry) error {
	var metaBytes []byte
	if entry.Metadata != nil {
		var err error
		metaBytes, err = json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("marshal audit metadata: %w", err)
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

	execDB := q
	if execDB == nil && db.DB != nil {
		execDB = db.DB
	}
	if execDB == nil {
		return fmt.Errorf("auditlog: nil querier for action=%s", entry.Action)
	}

	_, err := execDB.Exec(
		`INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10)`,
		pid, entry.Action, entry.EntityType, entry.EntityID, entry.RequestSummary, metaJSON, entry.IPAddress, entry.Success, entry.ErrorMessage, time.Now().UTC(),
	)
	if err != nil {
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
