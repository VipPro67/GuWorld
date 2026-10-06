package handlers

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Mana evaluation adapters. These are thin delegations to the service layer and
// deliberately live outside player.go so that combat/tribulation callers do not
// depend on player.go for mana math. The real implementation is PlayerSvc.

// evaluateManaTx delegates to the service layer.
func evaluateManaTx(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return PlayerSvc.EvaluateAndPersistMana(tx, id)
}

// evaluateManaTxEquipped is like evaluateManaTx but loads only equipped/vital Gus.
// The lighter inventory payload is suitable for read endpoints (auth/me, challenge)
// where the full inventory is either already loaded or fetched lazily by the frontend.
func evaluateManaTxEquipped(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return PlayerSvc.EvaluateAndPersistManaEquipped(tx, id)
}

// evaluateManaTxEquippedReadonly loads an equipped-only player and projects
// status in memory without acquiring write locks or persisting. Used for
// reading opponent / ally snapshots in arena, mine, sect war, and exploration.
func evaluateManaTxEquippedReadonly(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return PlayerSvc.ProjectStatusEquipped(tx, id)
}
