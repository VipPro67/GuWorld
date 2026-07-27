package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func StartGrowthWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Growth Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Growth Worker stopped")
				return
			case <-ticker.C:
				processGrowthTick(db)
			}
		}
	}()
}

func processGrowthTick(db *sql.DB) {
	rows, err := db.Query("SELECT id FROM players")
	if err != nil {
		logger.S.Errorf("Growth Worker Error: failed to fetch player list: %v", err)
		return
	}
	defer rows.Close()

	var playerIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			playerIDs = append(playerIDs, id)
		}
	}

	for _, pID := range playerIDs {
		err := updatePlayerGrowth(db, pID)
		if err != nil {
			logger.S.Errorf("Growth Worker Error updating player %s: %v", pID, err)
		}
	}
}

func updatePlayerGrowth(db *sql.DB, playerID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var guInvRaw []byte
	var growthDataRaw []byte
	var baseHP float64
	var hp float64

	err = tx.QueryRow(`
		SELECT gu_inventory, growth_data, base_hp, hp
		FROM players
		WHERE id = $1
		FOR UPDATE`, playerID).Scan(&guInvRaw, &growthDataRaw, &baseHP, &hp)
	if err != nil {
		return err
	}

	var guInv models.GuInventory
	if err := json.Unmarshal(guInvRaw, &guInv); err != nil {
		return err
	}

	var growthData models.GrowthData
	if err := json.Unmarshal(growthDataRaw, &growthData); err != nil {
		growthData = models.GrowthData{
			Boosts:     models.GrowthBoosts{},
			FullyGrown: []models.FullyGrownInfo{},
		}
	}

	updated := false
	hasGrowthGu := false
	now := time.Now()

	for i := range guInv {
		gu := &guInv[i]
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}

		spec, ok := models.GetGuAny(gu.GuID)
		if !ok || spec.Effects.Growth == nil || spec.Effects.Growth.HoursToMax <= 0 {
			continue
		}

		hasGrowthGu = true

		// Check if this Gu type is already fully grown
		isFullyGrown := false
		for _, fg := range growthData.FullyGrown {
			if fg.GuID == gu.GuID {
				isFullyGrown = true
				break
			}
		}
		if isFullyGrown {
			gu.EquippedAt = nil
			updated = true
			continue
		}

		// Initialize EquippedAt for legacy Gus equipped before growth feature existed
		if gu.EquippedAt == nil || gu.EquippedAt.IsZero() {
			gu.EquippedAt = &now
			updated = true
			continue
		}

		elapsed := now.Sub(*gu.EquippedAt).Hours()
		if elapsed <= 0 {
			continue
		}

		atk, def, hpBoost, spd := ApplyGrowthProgressWorker(gu, spec, elapsed)
		if atk != 0 || def != 0 || hpBoost != 0 || spd != 0 {
			growthData.Boosts.Attack += atk
			growthData.Boosts.Defense += def
			growthData.Boosts.HP += hpBoost
			growthData.Boosts.Speed += spd
			baseHP += hpBoost
			updated = true
		}

		if gu.GrowthHours >= spec.Effects.Growth.HoursToMax {
			growthData.FullyGrown = append(growthData.FullyGrown, models.FullyGrownInfo{
				GuID:    gu.GuID,
				Attack:  spec.Effects.Growth.MaxAttack,
				Defense: spec.Effects.Growth.MaxDefense,
				HP:      spec.Effects.Growth.MaxHP,
				Speed:   spec.Effects.Growth.MaxSpeed,
			})
		}
	}

	if !updated && !hasGrowthGu {
		return nil
	}

	newGuInvRaw, _ := json.Marshal(guInv)
	growthDataRawUpdated, _ := json.Marshal(growthData)

	_, err = tx.Exec(`UPDATE players SET gu_inventory = $1, growth_data = $2, base_hp = $3 WHERE id = $4`,
		string(newGuInvRaw), string(growthDataRawUpdated), baseHP, playerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func ApplyGrowthProgressWorker(gu *models.GuInstance, spec models.GuDictionaryItem, elapsedHours float64) (attack, defense, hp, speed float64) {
	if spec.Effects.Growth == nil || elapsedHours <= 0 {
		return 0, 0, 0, 0
	}
	g := spec.Effects.Growth
	maxHours := g.HoursToMax
	if maxHours <= 0 {
		maxHours = 168
	}
	oldHours := gu.GrowthHours
	newHours := oldHours + elapsedHours
	if newHours > maxHours {
		newHours = maxHours
	}
	progress := newHours / maxHours
	if progress > 1.0 {
		progress = 1.0
	}
	oldProgress := oldHours / maxHours

	attack = g.MaxAttack * (progress - oldProgress)
	defense = g.MaxDefense * (progress - oldProgress)
	hp = g.MaxHP * (progress - oldProgress)
	speed = g.MaxSpeed * (progress - oldProgress)

	gu.GrowthHours = newHours
	now := time.Now()
	gu.EquippedAt = &now
	return
}
