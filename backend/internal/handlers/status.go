package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// rowQuerier matches the QueryRow surface shared by *sql.DB and *sql.Tx so
// status helpers work inside and outside transactions.
type rowQuerier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// execQuerier matches the Exec surface shared by *sql.DB and *sql.Tx.
type execQuerier interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// loadPlayerStatuses reads the raw statuses array for a player.
func loadPlayerStatuses(q rowQuerier, playerID uuid.UUID) (models.StatusList, error) {
	var statuses models.StatusList
	err := q.QueryRow(`SELECT statuses FROM players WHERE id = $1`, playerID).Scan(&statuses)
	if err != nil {
		return nil, err
	}
	return statuses, nil
}

// GrantPlayerStatus adds (or refreshes) a status on a player inside the given
// transaction. Duration semantics: d < 0 => registry default (permanent when
// the definition has none), d == 0 => forced permanent, d > 0 => explicit.
// It returns false when the status key is unknown to the registry.
func GrantPlayerStatus(q execQuerier, rq rowQuerier, playerID uuid.UUID, statusID, source string, d time.Duration, value float64) (bool, error) {
	if !gamefx.ValidStatuses()[statusID] {
		return false, fmt.Errorf("unknown status %q", statusID)
	}
	statuses, err := loadPlayerStatuses(rq, playerID)
	if err != nil {
		return false, err
	}
	if d < 0 {
		d = time.Duration(gamefx.StatusDefaultDuration(statusID)) * time.Minute
	}
	statuses.Add(statusID, source, d, value)
	raw, err := statuses.Value()
	if err != nil {
		return false, err
	}
	if _, err := q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, playerID); err != nil {
		return false, err
	}
	return true, nil
}

// RevokePlayerStatus removes every instance of a status id. Returns whether
// the status was present.
func RevokePlayerStatus(q execQuerier, rq rowQuerier, playerID uuid.UUID, statusID string) (bool, error) {
	statuses, err := loadPlayerStatuses(rq, playerID)
	if err != nil {
		return false, err
	}
	if !statuses.Has(statusID) {
		return false, nil
	}
	statuses.Remove(statusID)
	raw, err := statuses.Value()
	if err != nil {
		return false, err
	}
	if _, err := q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, playerID); err != nil {
		return false, err
	}
	return true, nil
}

// CleanExpiredPlayerStatuses drops expired entries and persists when anything
// changed. Safe to call on hot paths before gameplay checks.
func CleanExpiredPlayerStatuses(q execQuerier, rq rowQuerier, playerID uuid.UUID) error {
	statuses, err := loadPlayerStatuses(rq, playerID)
	if err != nil {
		return err
	}
	if !statuses.Expire(time.Now()) {
		return nil
	}
	raw, err := statuses.Value()
	if err != nil {
		return err
	}
	_, err = q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, playerID)
	return err
}

// statusBlockedMessage builds a Vietnamese error listing the statuses that
// block the given action.
func statusBlockedMessage(p *models.Player, action string) string {
	defs := gamefx.BlockingStatuses(p, action)
	if len(defs) == 0 {
		return "Hành động hiện bị chặn bởi trạng thái."
	}
	names := ""
	for i, def := range defs {
		if i > 0 {
			names += ", "
		}
		names += def.Name
	}
	return fmt.Sprintf("Không thể thực hiện do đang trong trạng thái: %s", names)
}
