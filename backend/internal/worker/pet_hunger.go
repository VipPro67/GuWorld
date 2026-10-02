package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func StartPetHungerWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting Pet Hunger Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Pet Hunger Worker stopped")
				return
			case <-ticker.C:
				processPetHungerDecay(ctx, db)
			}
		}
	}()
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

	// Starving pets: hunger <= 0 → death
	rows, err := tx.Query(`
		DELETE FROM non_player_actors
		WHERE hunger <= 0 AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'
		  AND execution_state IN ('STORED', 'RECOVERING')
		RETURNING id, owner_player_id, name
	`)
	if err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to query starving pets: %v", err)
		return
	}
	defer rows.Close()

	// Collect starved pets for audit log within the same transaction
	type starvedPet struct {
		id       string
		playerID string
		nickname string
	}
	var starved []starvedPet
	starvedCount := 0
	for rows.Next() {
		var id, playerID, nickname string
		if errScan := rows.Scan(&id, &playerID, &nickname); errScan != nil {
			continue
		}
		starvedCount++
		starved = append(starved, starvedPet{id: id, playerID: playerID, nickname: nickname})
	}
	rows.Close()

	// Insert audit logs inside the transaction for consistency
	for _, p := range starved {
		if _, err := tx.Exec(`
			INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
			VALUES ($1, 'pet.starve_death', 'player_pet', $2, $3, '{}', '', false, 'Pet chết đói', NOW())
		`, p.playerID, p.id, p.nickname+" chết đói"); err != nil {
			logger.S.Errorf("Pet Hunger Worker: failed to audit starvation: %v", err)
			return
		}
	}

	if starvedCount > 0 {
		logger.S.Infof("Pet Hunger Worker: %d pets starved to death", starvedCount)
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to commit: %v", err)
	}
}
