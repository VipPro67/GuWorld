package handlers

import (
	"context"
	"database/sql"

	"github.com/vippro67/GuWorld/backend/internal/db"
)

// SectMembership is the read projection of "which sect does this player belong
// to, and at what rank". This module is the single home for the
// `FROM sect_members WHERE player_id = $1` query; handlers must not embed that
// SQL again. Mutations (join/leave/promote) stay with their owning handlers.
type SectMembership struct {
	SectID      string
	Rank        string
	IsDefending bool
}

// contextRowQuerier is satisfied by *sql.DB and *sql.Tx for context reads.
type contextRowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// LoadSectMembership reads the player's membership through the given querier.
// It returns sql.ErrNoRows when the player is not in a sect, matching the
// previous inline queries so callers keep their existing error handling.
func LoadSectMembership(q db.Queryer, playerID any) (SectMembership, error) {
	var m SectMembership
	err := q.QueryRow(`
		SELECT sect_id, member_rank, is_defending
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.SectID, &m.Rank, &m.IsDefending)
	return m, err
}

// LoadSectMembershipContext is LoadSectMembership for callers that already hold
// a request context.
func LoadSectMembershipContext(ctx context.Context, q contextRowQuerier, playerID any) (SectMembership, error) {
	var m SectMembership
	err := q.QueryRowContext(ctx, `
		SELECT sect_id, member_rank, is_defending
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.SectID, &m.Rank, &m.IsDefending)
	return m, err
}

// LoadSectID reads only the player's sect id. Prefer LoadSectMembership when
// the rank or defending flag is also needed.
func LoadSectID(q db.Queryer, playerID any) (string, error) {
	m, err := LoadSectMembership(q, playerID)
	if err != nil {
		return "", err
	}
	return m.SectID, nil
}
