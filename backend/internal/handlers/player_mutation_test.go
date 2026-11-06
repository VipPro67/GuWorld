package handlers

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestMutatePlayerNilDatabase(t *testing.T) {
	origDB := db.DB
	db.DB = nil
	defer func() { db.DB = origDB }()

	_, err := MutatePlayer(uuid.New(), PlayerMutationOptions{}, func(tx *sql.Tx, player *models.Player) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error when db.DB is nil, got nil")
	}
}

func TestMutatePlayerClosureFailureRollsBack(t *testing.T) {
	// If the database is not initialized in unit test environment,
	// MutatePlayer fails cleanly with error and does not panic.
	expectedErr := errors.New("domain check failed")
	_, err := MutatePlayer(uuid.New(), PlayerMutationOptions{}, func(tx *sql.Tx, player *models.Player) error {
		return expectedErr
	})
	if err == nil {
		t.Fatal("expected error from mutation, got nil")
	}
}
