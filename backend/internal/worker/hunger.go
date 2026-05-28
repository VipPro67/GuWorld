package worker

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// StartHungerWorker begins the background process for depleting Gu bug hunger.
func StartHungerWorker(db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		log.Printf("Starting background Hunger Worker with tick interval of %v", interval)
		for range ticker.C {
			processHungerDecay(db)
		}
	}()
}

func processHungerDecay(db *sql.DB) {
	// 1. Fetch all player IDs to process them individually in isolated transactions
	rows, err := db.Query("SELECT id FROM players")
	if err != nil {
		log.Printf("Hunger Worker Error: failed to fetch player list: %v", err)
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

	// 2. Process each player's inventory
	for _, pID := range playerIDs {
		err := updatePlayerGuHunger(db, pID)
		if err != nil {
			log.Printf("Hunger Worker Error updating player %s: %v", pID, err)
		}
	}
}

func updatePlayerGuHunger(db *sql.DB, playerID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Select and lock the row
	var username string
	var guInvRaw []byte
	var hp float64
	var eqRaw []byte

	err = tx.QueryRow(`
		SELECT username, gu_inventory, hp, equipped_gus 
		FROM players 
		WHERE id = $1 
		FOR UPDATE`, playerID).Scan(&username, &guInvRaw, &hp, &eqRaw)
	if err != nil {
		return err
	}

	var guInv models.GuInventory
	if err := json.Unmarshal(guInvRaw, &guInv); err != nil {
		return err
	}

	var equippedGus []string
	if err := json.Unmarshal(eqRaw, &equippedGus); err != nil {
		equippedGus = []string{}
	}

	updated := false
	playerHPBacklash := 0.0
	newInventory := models.GuInventory{} // never nil — marshals to [] not null

	for i := range guInv {
		gu := guInv[i]
		if gu.Status == "ACTIVE" || gu.Status == "SLEEPING" {
			specs, found := models.GuRegistry[gu.GuID]
			if !found {
				newInventory = append(newInventory, gu)
				continue
			}

			gu.Hunger -= specs.HungerRate
			updated = true

			if gu.Hunger <= 0 {
				gu.Hunger = 0
				// Vital Gu never sleeps — it either stays alive (fed) or dies
				if gu.IsVital {
					gu.Status = "DEAD"
					log.Printf("Cultivation Log: Player %s's VITAL Gu %s (%s) starved and DIED — HP backlash incoming", username, specs.Name, gu.InstanceID[:8])
					playerHPBacklash += 80.0
					// Dead Gu is deleted — do not append
				} else {
					// Roll for sleep vs death (20% Hibernation, 80% Dead)
					roll := rand.Intn(100)
					if roll < 20 {
						gu.Status = "SLEEPING"
						log.Printf("Cultivation Log: Player %s's Gu %s (%s) starved and went into HIBERNATION", username, specs.Name, gu.InstanceID[:8])
						newInventory = append(newInventory, gu)
					} else {
						gu.Status = "DEAD"
						log.Printf("Cultivation Log: Player %s's Gu %s (%s) starved and DIED", username, specs.Name, gu.InstanceID[:8])
						// Dead Gu is deleted, so we do not append it to newInventory
					}
				}
			} else {
				newInventory = append(newInventory, gu)
			}
		} else {
			// Keep other statuses (e.g. SEALED, DYING, or existing DEAD bugs if any, though regular dead are removed)
			if gu.Status != "DEAD" {
				newInventory = append(newInventory, gu)
			}
		}
	}

	// If no Gu was modified and no HP backlash occurred, we can return early
	if !updated && playerHPBacklash == 0 {
		return nil
	}

	// Apply HP Backlash if Vital Gu died
	if playerHPBacklash > 0 {
		hp -= playerHPBacklash
		if hp <= 0 {
			hp = 0
			// Reset combat state upon defeat
			_, err = tx.Exec(`UPDATE players SET hp = $1, active_combat = NULL WHERE id = $2`, hp, playerID)
		} else {
			_, err = tx.Exec(`UPDATE players SET hp = $1 WHERE id = $2`, hp, playerID)
		}
		if err != nil {
			return err
		}
	}

	// Re-evaluate equipped_gus array
	newEquippedGus := []string{} // never nil — marshals to [] not null
	for _, eqInstID := range equippedGus {
		stillExists := false
		for _, item := range newInventory {
			if item.InstanceID == eqInstID && item.Status == "ACTIVE" {
				stillExists = true
				break
			}
		}
		if stillExists {
			newEquippedGus = append(newEquippedGus, eqInstID)
		}
	}

	newGuInvRaw, err := json.Marshal(newInventory)
	if err != nil {
		return err
	}

	newEquippedGusRaw, err := json.Marshal(newEquippedGus)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE players SET gu_inventory = $1, equipped_gus = $2 WHERE id = $3`, string(newGuInvRaw), string(newEquippedGusRaw), playerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
