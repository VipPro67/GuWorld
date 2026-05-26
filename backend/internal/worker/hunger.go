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
	err = tx.QueryRow(`
		SELECT username, gu_inventory 
		FROM players 
		WHERE id = $1 
		FOR UPDATE`, playerID).Scan(&username, &guInvRaw)
	if err != nil {
		return err
	}

	var guInv models.GuInventory
	if err := json.Unmarshal(guInvRaw, &guInv); err != nil {
		return err
	}

	updated := false
	for i := range guInv {
		gu := &guInv[i]

		// Only active Gu bugs feel hunger
		if gu.Status == "ACTIVE" {
			specs, found := models.GuRegistry[gu.GuID]
			if !found {
				continue
			}

			gu.Hunger -= specs.HungerRate
			updated = true

			if gu.Hunger <= 0 {
				gu.Hunger = 0
				// Roll for sleep vs death
				// 20% Sleeping (hibernation), 80% Dead
				roll := rand.Intn(100)
				if roll < 20 {
					gu.Status = "SLEEPING"
					log.Printf("Cultivation Log: Player %s's Gu %s (%s) starved and went into HIBERNATION", username, specs.Name, gu.InstanceID[:8])
				} else {
					gu.Status = "DEAD"
					log.Printf("Cultivation Log: Player %s's Gu %s (%s) starved and DIED", username, specs.Name, gu.InstanceID[:8])
				}
			}
		}
	}

	// If no Gu was modified, we can just return
	if !updated {
		return nil
	}

	newGuInvRaw, err := json.Marshal(guInv)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE players SET gu_inventory = $1 WHERE id = $2`, newGuInvRaw, playerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
