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

	// Decay hunger for ACTIVE pets based on elapsed hours since updated_at
	_, err = tx.Exec(`
		UPDATE player_pets
		SET hunger = GREATEST(hunger - pet_hunger_per_hour * EXTRACT(EPOCH FROM (NOW() - updated_at)) / 3600.0, 0),
		    updated_at = NOW()
		WHERE state = 'ACTIVE' AND hunger > 0
	`)
	if err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to decay hunger: %v", err)
		return
	}

	// SLEEPING pets decay at 50% rate
	_, err = tx.Exec(`
		UPDATE player_pets
		SET hunger = GREATEST(hunger - pet_hunger_per_hour * 0.5 * EXTRACT(EPOCH FROM (NOW() - updated_at)) / 3600.0, 0),
		    updated_at = NOW()
		WHERE state = 'SLEEPING' AND hunger > 0
	`)
	if err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to decay sleeping hunger: %v", err)
		return
	}

	// Starving pets: hunger <= 0 → death
	rows, err := tx.Query(`
		DELETE FROM player_pets
		WHERE hunger <= 0 AND state IN ('ACTIVE', 'SLEEPING')
		RETURNING id, player_id, nickname
	`)
	if err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to query starving pets: %v", err)
		return
	}
	defer rows.Close()

	// Collect starved pets for audit log within the same transaction
	type starvedPet struct {
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
		starved = append(starved, starvedPet{playerID: playerID, nickname: nickname})
	}
	rows.Close()

	// Insert audit logs inside the transaction for consistency
	for _, p := range starved {
		_, _ = tx.Exec(`
			INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
			VALUES ($1, 'pet.starve_death', 'player_pet', $2, $3, '{}', '', false, 'Pet chết đói', NOW())
		`, p.playerID, p.nickname+" chết đói")
	}

	if starvedCount > 0 {
		logger.S.Infof("Pet Hunger Worker: %d pets starved to death", starvedCount)
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Pet Hunger Worker: failed to commit: %v", err)
	}
}
