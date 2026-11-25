package handlers

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// MutatePlayerInTx must refuse to run without a caller-owned transaction; the
// only safe fallback is for the caller to use MutatePlayer, not for this seam to
// open a transaction of its own.
func TestMutatePlayerInTxRequiresTransaction(t *testing.T) {
	_, err := MutatePlayerInTx(nil, uuid.New(), PlayerMutationOptions{}, func(tx *sql.Tx, player *models.Player) error {
		t.Fatal("mutation closure must not run without a transaction")
		return nil
	})
	if err == nil {
		t.Fatal("expected error for nil transaction, got nil")
	}
	if !strings.Contains(err.Error(), "transaction is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Publishing is a no-op for a nil player so callers can publish unconditionally
// after commit without nil checks at every site.
func TestPublishPlayerMutationNilPlayerIsNoop(t *testing.T) {
	PublishPlayerMutation(nil, PlayerMutationOptions{})
}
