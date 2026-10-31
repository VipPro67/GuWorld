package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Position updates and watchtower damage are processed per event in isolated transactions with timeouts.
func ProcessActiveTribulationGrid() error {
	// 1. Fetch all active tribulation events outside of long-held transactions
	rows, err := db.DB.Query(`
		SELECT id, aperture_id, player_id, tribulation_type, started_at
		FROM tribulation_events 
		WHERE status = 'ACTIVE'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []ActiveTribulationEvent
	for rows.Next() {
		var ev ActiveTribulationEvent
		if err := rows.Scan(&ev.EventID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.StartedAt); err == nil {
			events = append(events, ev)
		}
	}
	rows.Close()

	// 2. Process each event in its own transaction so a single failing or slow event doesn't block the rest
	for _, ev := range events {
		if err := processActiveTribulationEvent(ev); err != nil {
			logger.S.Errorf("Error processing tribulation event %s for player %s: %v", ev.EventID, ev.PlayerID, err)
		}
	}
	return nil
}

func processActiveTribulationEvent(ev ActiveTribulationEvent) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SET LOCAL statement_timeout = '30000'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '10000'`); err != nil {
		return err
	}
	suppressed, err := suppressTribulationsForDeadAperture(tx, ev.PlayerID)
	if err != nil {
		return err
	}
	if suppressed {
		return tx.Commit()
	}

	// Freeze the tribulation while the player is offline: enemies hold their
	// position and never damage nodes. Bump started_at so the 24h auto-resolve
	// window only counts online time.
	if !isPlayerOnline(ev.PlayerID) {
		if _, err := tx.Exec(`UPDATE tribulation_events SET started_at = NOW() WHERE id = $1`, ev.EventID); err != nil {
			return fmt.Errorf("update tribulation started_at for offline player: %w", err)
		}
		return tx.Commit()
	}

	// Grace period after spawn/return: enemies stay idle for a short window.
	if time.Since(ev.StartedAt) < tribulationGracePeriod {
		return tx.Commit()
	}

	// The realtime zone owns monster movement, node damage and deaths while the
	// character fights inside the aperture. The durable rows are synced from the
	// live snapshot instead; running this worker in parallel would damage nodes
	// with invisible enemies and desync the encounter.
	var currentRegion string
	if err := tx.QueryRow(`SELECT COALESCE(current_region, '') FROM players WHERE id = $1`, ev.PlayerID).Scan(&currentRegion); err != nil {
		return err
	}
	if currentRegion == "aperture_tribulation" {
		return tx.Commit()
	}
	// Realtime owns the encounter whenever the character is inside the
	// aperture; the old off-screen grid simulation is retired. When the player
	// is engaged elsewhere, the tribulation freezes instead: enemies hold
	// position and never damage nodes. Bump started_at so the 24h overdue
	// window only counts time spent inside the encounter.
	if _, err := tx.Exec(`UPDATE tribulation_events SET started_at = NOW() WHERE id = $1`, ev.EventID); err != nil {
		return fmt.Errorf("update tribulation started_at for disengaged player: %w", err)
	}
	return tx.Commit()
}

func RegenerateImmortalApertures() error {
	// Cheap guard: skip the UPDATE entirely when no aperture needs healing.
	var needsHealing bool
	if err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM immortal_apertures WHERE state != 'DESTROYED' AND aperture_hp < max_aperture_hp)`).Scan(&needsHealing); err != nil {
		return err
	}
	if !needsHealing {
		return nil
	}

	// Heal aperture_hp by 0.5% of max_aperture_hp per minute (minimum 10 HP per minute)
	// If state is DAMAGED and aperture_hp reaches max_aperture_hp, set state back to STABLE
	_, err := db.DB.Exec(`
			UPDATE immortal_apertures
			SET 
				aperture_hp = LEAST(max_aperture_hp, aperture_hp + GREATEST(10, max_aperture_hp / 200)),
				state = CASE 
					WHEN (aperture_hp + GREATEST(10, max_aperture_hp / 200)) >= max_aperture_hp THEN
						CASE 
							WHEN NOT EXISTS (SELECT 1 FROM tribulation_events WHERE aperture_id = immortal_apertures.id AND status = 'ACTIVE')
							     AND NOT EXISTS (SELECT 1 FROM tribulation_enemies te JOIN tribulation_events ev ON te.event_id = ev.id WHERE ev.aperture_id = immortal_apertures.id AND te.is_dead = false)
							THEN 'STABLE'
							ELSE 'TRIBULATION_ACTIVE'
						END
					ELSE state
				END,
				updated_at = NOW()
			WHERE state != 'DESTROYED' AND aperture_hp < max_aperture_hp
		`)
	return err
}

func StartTribulationWorker(ctx context.Context) {
	worker.RunEvery(ctx, 60*time.Second, "Tribulation Worker", func(ctx context.Context) {
		if err := CheckAndTriggerTribulations(); err != nil {
			logger.S.Errorf("Tribulation background worker error: %v", err)
		}
		if errRegen := RegenerateImmortalApertures(); errRegen != nil {
			logger.S.Errorf("Aperture HP regeneration background worker error: %v", errRegen)
		}
	})

	worker.RunEvery(ctx, 15*time.Second, "Tribulation Grid Worker", func(ctx context.Context) {
		if err := ProcessActiveTribulationGrid(); err != nil {
			logger.S.Errorf("Tribulation grid worker error: %v", err)
		}
	})

	StartTribulationAutoSettleWorker(ctx)

	logger.S.Info("Tribulation and internal year worker started successfully (Interval: 60s)")
}
