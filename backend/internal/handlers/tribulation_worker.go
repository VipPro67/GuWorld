package handlers

import (
	"context"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

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

	logger.S.Info("Tribulation and internal year worker started successfully (Interval: 60s)")
}
