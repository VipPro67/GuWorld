package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type GuHistoryLog struct {
	ID         string                 `json:"id"`
	InstanceID string                 `json:"instance_id"`
	GuID       string                 `json:"gu_id"`
	Rank       int                    `json:"rank"`
	EventType  string                 `json:"event_type"`
	ActorID    *string                `json:"actor_id,omitempty"`
	ActorName  string                 `json:"actor_name,omitempty"`
	TargetID   *string                `json:"target_id,omitempty"`
	TargetName string                 `json:"target_name,omitempty"`
	Summary    string                 `json:"summary"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// LogGuHistory records a history entry for a high-rank Gu instance (Rank >= 6).
// Lower-rank events are dropped: gu_history_logs chỉ theo vết Tiên Cổ.
func LogGuHistory(exec Execer, instanceID, guID string, rank int, eventType string, actorID, targetID *uuid.UUID, summary string, metadata map[string]interface{}) error {
	if instanceID == "" || guID == "" {
		return nil
	}
	if rank < 6 {
		return nil
	}

	if exec == nil {
		if DB == nil {
			return nil
		}
		exec = DB
	}

	metaJSON := "{}"
	if metadata != nil {
		if bytes, err := json.Marshal(metadata); err == nil {
			metaJSON = string(bytes)
		}
	}

	query := `
		INSERT INTO gu_history_logs (instance_id, gu_id, rank, event_type, actor_id, target_id, summary, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
	`
	_, err := exec.Exec(query, instanceID, guID, rank, eventType, actorID, targetID, summary, metaJSON)
	if err != nil {
		logger.S.Errorf("LogGuHistory failed for instance=%s event=%s: %v", instanceID, eventType, err)
	}
	return err
}

// FetchGuHistoryLogs retrieves the full lifecycle history timeline for a Gu instance.
func FetchGuHistoryLogs(db *sql.DB, instanceID string) ([]GuHistoryLog, error) {
	query := `
		SELECT g.id, g.instance_id, g.gu_id, g.rank, g.event_type,
		       g.actor_id, COALESCE(pa.username, ''),
		       g.target_id, COALESCE(pt.username, ''),
		       g.summary, g.metadata, g.created_at
		FROM gu_history_logs g
		LEFT JOIN players pa ON g.actor_id = pa.id
		LEFT JOIN players pt ON g.target_id = pt.id
		WHERE g.instance_id = $1
		ORDER BY g.created_at ASC
	`
	rows, err := db.Query(query, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query gu history: %w", err)
	}
	defer rows.Close()

	var logs []GuHistoryLog
	for rows.Next() {
		var l GuHistoryLog
		var actorID, targetID sql.NullString
		var metaRaw []byte
		err := rows.Scan(
			&l.ID, &l.InstanceID, &l.GuID, &l.Rank, &l.EventType,
			&actorID, &l.ActorName,
			&targetID, &l.TargetName,
			&l.Summary, &metaRaw, &l.CreatedAt,
		)
		if err != nil {
			return nil, err
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
		logs = []GuHistoryLog{}
	}
	return logs, nil
}
