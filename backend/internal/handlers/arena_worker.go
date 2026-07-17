package handlers

import (
	"context"
	"log"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
)

// StartArenaSettlementWorker runs every minute, checking if it is 22:00
func StartArenaSettlementWorker(ctx context.Context) {
	log.Println("Starting Arena Settlement Worker...")
	
	var lastSettlementDay int
	
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Arena Settlement Worker stopped.")
				return
			case t := <-ticker.C:
				if t.Hour() == 22 && t.Minute() == 0 {
					currentDay := t.YearDay()
					if lastSettlementDay != currentDay {
						log.Println("Running Daily Arena Settlement...")
						runArenaSettlement()
						lastSettlementDay = currentDay
					}
				}
			}
		}
	}()
}

func runArenaSettlement() {
	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("ArenaSettlement Error: Failed to start tx", err)
		return
	}
	defer tx.Rollback()

	// Get all rankings
	rows, err := tx.Query("SELECT player_id, rank FROM arena_rankings")
	if err != nil {
		log.Println("ArenaSettlement Error: Failed to query rankings", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var playerID string
		var rank int
		if err := rows.Scan(&playerID, &rank); err != nil {
			continue
		}

		rewardCoins := 50
		if rank == 1 {
			rewardCoins = 1000
		} else if rank == 2 {
			rewardCoins = 800
		} else if rank == 3 {
			rewardCoins = 500
		} else if rank <= 10 {
			rewardCoins = 200
		}

		// Update materials_inventory
		_, errUpdate := tx.Exec(`
			UPDATE players 
			SET materials_inventory = COALESCE(materials_inventory, '{}'::jsonb) || jsonb_build_object('arena_coin', COALESCE((materials_inventory->>'arena_coin')::int, 0) + $1)
			WHERE id = $2
		`, rewardCoins, playerID)
		
		if errUpdate != nil {
			log.Printf("ArenaSettlement Error: Failed to give reward to %s: %v\n", playerID, errUpdate)
		}
	}
	
	if err := tx.Commit(); err != nil {
		log.Println("ArenaSettlement Error: Failed to commit tx", err)
	} else {
		log.Println("Daily Arena Settlement completed successfully.")
	}
}
