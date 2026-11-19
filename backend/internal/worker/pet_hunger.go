package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func StartPetHungerWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	RunEvery(ctx, interval, "Pet Hunger Worker", func(ctx context.Context) {
		processPetHungerDecay(ctx, db)
	})
}

func processPetHungerDecay(ctx context.Context, db *sql.DB) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to begin tx: %v", err)
		return
	}
	defer tx.Rollback()

	// Decay active companion beasts using the dedicated hunger clock.
	_, err = tx.Exec(`
		UPDATE non_player_actors
		SET hunger = GREATEST(hunger - upkeep_rate * EXTRACT(EPOCH FROM (NOW() - hunger_updated_at)) / 3600.0, 0),
		    hunger_updated_at = NOW(), updated_at = NOW(), version = version + 1
		WHERE actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'
		  AND execution_state = 'STORED' AND hunger > 0
	`)
	if err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to decay hunger: %v", err)
		return
	}

	// SLEEPING pets decay at 50% rate
	_, err = tx.Exec(`
		UPDATE non_player_actors
		SET hunger = GREATEST(hunger - upkeep_rate * 0.5 * EXTRACT(EPOCH FROM (NOW() - hunger_updated_at)) / 3600.0, 0),
		    hunger_updated_at = NOW(), updated_at = NOW(), version = version + 1
		WHERE actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'
		  AND execution_state = 'RECOVERING' AND hunger > 0
	`)
	if err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to decay sleeping hunger: %v", err)
		return
	}

	// Delete starving pets and write their audit rows in one statement and transaction.
	var starvedCount int
	operationID := uuid.NewString()
	err = tx.QueryRow(`
		WITH starved AS (
			DELETE FROM non_player_actors
			WHERE hunger <= 0 AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'
			  AND execution_state IN ('STORED', 'RECOVERING')
			RETURNING id, owner_player_id, name
		), audit AS (
			INSERT INTO audit_logs (player_id, affected_player_id, action, entity_type, entity_id, request_id, request_summary, metadata, ip_address, success, error_message, created_at)
			SELECT owner_player_id, owner_player_id, 'pet.starve_death', 'player_pet', id::text, $1,
			       name || ' chết đói', '{}', '', false, 'Pet chết đói', NOW()
			FROM starved
			RETURNING 1
		)
		SELECT count(*) FROM starved
	`, operationID).Scan(&starvedCount)
	if err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to delete and audit starving pets: %v", err)
		return
	}

	if starvedCount > 0 {
		logger.S.Infof("Pet Hunger Worker: %d pets starved to death", starvedCount)
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to commit: %v", err)
	}
}
