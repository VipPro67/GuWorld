package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	pkgdb "github.com/vippro67/GuWorld/backend/internal/db"
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

// processGrowthTick advances Gu growth for every equipped ACTIVE Gu that has a
// growth profile. A set-based SQL query narrows the work to only the players
// who actually own such a Gu (a tiny subset of the player base); the growth_data
// fold for those players is then done in Go.
func processGrowthTick(db *sql.DB) {
	players, err := findGrowthAffectedPlayers(db)
	if err != nil {
		logger.S.Errorf("Growth Worker Error finding affected players: %v", err)
		return
	}

	for _, pID := range players {
		if err := updatePlayerGrowth(db, pID); err != nil {
			logger.S.Errorf("Growth Worker Error updating player %s: %v", pID, err)
		}
	}
}

// findGrowthAffectedPlayers returns players who own at least one equipped ACTIVE
// Gu with a growth profile. This replaces the old full-table player scan.
func findGrowthAffectedPlayers(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT pg.player_id::text
		FROM player_gu pg
		JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
		WHERE pg.equipped AND pg.state = 'ACTIVE'
		  AND gd.effects->'growth' IS NOT NULL
		  AND COALESCE((gd.effects->'growth'->>'hours_to_max')::double precision, 0) > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playerIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			playerIDs = append(playerIDs, id)
		}
	}
	return playerIDs, rows.Err()
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

	err = tx.QueryRow(`
		SELECT `+pkgdb.GuInventoryAggregate("players.id")+`, growth_data, base_hp
		FROM players
		WHERE id = $1
		FOR UPDATE`, playerID).Scan(&guInvRaw, &growthDataRaw, &baseHP)
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

	hasGrowthGu := false
	now := time.Now()

	for i := range guInv {
		gu := &guInv[i]
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}

		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok || spec.Effects.Growth == nil || spec.Effects.Growth.HoursToMax <= 0 {
			continue
		}

		hasGrowthGu = true

		// Check if this Gu type is already fully grown
		isFullyGrown := false
		for _, fg := range growthData.FullyGrown {
			if fg.GuID == gu.GuID && fg.Rank == gu.Rank {
				isFullyGrown = true
				break
			}
		}
		if isFullyGrown {
			gu.EquippedAt = nil
			_, err = tx.Exec(`UPDATE player_gu SET equipped_at = NULL, updated_at = NOW() WHERE id = $1`, gu.InstanceID)
			if err != nil {
				return err
			}
			continue
		}

		// Initialize EquippedAt for legacy Gus equipped before growth feature existed
		if gu.EquippedAt == nil || gu.EquippedAt.IsZero() {
			gu.EquippedAt = &now
			_, err = tx.Exec(`UPDATE player_gu SET equipped_at = $1, updated_at = NOW() WHERE id = $2`, now, gu.InstanceID)
			if err != nil {
				return err
			}
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
		}

		if gu.GrowthHours >= spec.Effects.Growth.HoursToMax {
			growthData.FullyGrown = append(growthData.FullyGrown, models.FullyGrownInfo{
				GuID:    gu.GuID,
				Rank:    gu.Rank,
				Attack:  spec.Effects.Growth.MaxAttack,
				Defense: spec.Effects.Growth.MaxDefense,
				HP:      spec.Effects.Growth.MaxHP,
				Speed:   spec.Effects.Growth.MaxSpeed,
			})
		}

		// Persist growth_hours / equipped_at per Gu (no full-inventory rewrite).
		_, err = tx.Exec(`
			UPDATE player_gu SET growth_hours = $1, equipped_at = $2, updated_at = NOW() WHERE id = $3`,
			gu.GrowthHours, gu.EquippedAt, gu.InstanceID)
		if err != nil {
			return err
		}
	}

	if !hasGrowthGu {
		return nil
	}

	growthDataRawUpdated, _ := json.Marshal(growthData)

	_, err = tx.Exec(`UPDATE players SET growth_data = $1, base_hp = $2 WHERE id = $3`,
		string(growthDataRawUpdated), baseHP, playerID)
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
