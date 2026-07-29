package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartHungerWorker begins the background process for depleting Gu bug hunger.
func StartHungerWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Hunger Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Hunger Worker stopped")
				return
			case <-ticker.C:
				processHungerDecay(db)
			}
		}
	}()
}

func processHungerDecay(db *sql.DB) {
	// 1. Fetch all player IDs to process them individually in isolated transactions
	rows, err := db.Query("SELECT id FROM players")
	if err != nil {
		logger.S.Errorf("Hunger Worker Error: failed to fetch player list: %v", err)
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
			logger.S.Errorf("Hunger Worker Error updating player %s: %v", pID, err)
		}
	}
}

func updatePlayerGuHunger(db *sql.DB, playerID string) error {
	pUUID, _ := uuid.Parse(playerID)
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
	var tsm float64

	err = tx.QueryRow(`
		SELECT username, gu_inventory, hp, equipped_gus, time_speed_multiplier 
		FROM players 
		WHERE id = $1 
		FOR UPDATE`, playerID).Scan(&username, &guInvRaw, &hp, &eqRaw, &tsm)
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

	// Also fetch stones, mana, rank for ON_TICK effects
	var primevalStones int
	var currentMana, maxMana, immortalStones float64
	var playerRank int
	_ = tx.QueryRow(`SELECT primeval_stones, current_mana, max_mana, immortal_stones, rank FROM players WHERE id = $1`, playerID).Scan(&primevalStones, &currentMana, &maxMana, &immortalStones, &playerRank)

	updated := false
	playerHPBacklash := 0.0
	tickStones := 0
	tickMana := 0.0
	tickHP := 0.0
	tickImmortalStones := 0.0
	newInventory := models.GuInventory{} // never nil — marshals to [] not null

	for i := range guInv {
		gu := guInv[i]
		if gu.Status == "ACTIVE" || gu.Status == "SLEEPING" {
			specs, found := models.GetGuAny(gu.GuID)
			if !found {
				newInventory = append(newInventory, gu)
				continue
			}

			rate := tsm
			if rate <= 0 {
				rate = 1.0
			}
			gu.Hunger -= int(float64(specs.HungerRate) * rate)
			updated = true

			// Fire ON_TICK for equipped ACTIVE Gus
			if gu.Equipped && gu.Status == "ACTIVE" && specs.Effects.Other != nil {
				// Determine Thien Nguyen multiplier based on player rank vs gu rank
				tnMultiplier := 1.0
				if strings.HasPrefix(specs.ID, "GU_HEAVENLY_ESSENCE_LOTUS") {
					tnMultiplier = getThienNguyenMultiplier(playerRank, specs.Rank)
				}
				for _, mech := range specs.Effects.Other {
					if mech.EventTrigger != "ON_TICK" {
						continue
					}
					var params map[string]interface{}
					if len(mech.Params) > 0 {
						json.Unmarshal(mech.Params, &params)
					}
					switch mech.ActionCode {
					case "HOT_TICK":
						val, _ := params["value"].(float64)
						if val <= 0 {
							val = 5.0
						}
						tickHP += val * tnMultiplier
					case "GENERATE_STONES_TICK":
						val, _ := params["value"].(float64)
						if val <= 0 {
							val = 1.0
						}
						tickStones += int(float64(val) * tnMultiplier)
					case "MANA_REGEN_TICK":
						val, _ := params["value"].(float64)
						if val <= 0 {
							val = 0.5
						}
						tickMana += val * tnMultiplier
					case "GENERATE_IMMORTAL_STONES_TICK":
						val, _ := params["value"].(float64)
						if val <= 0 {
							val = 1.0
						}
						tickImmortalStones += val * tnMultiplier
					}
				}
			}

			if gu.Hunger <= 0 {
				gu.Hunger = 0
				// Vital Gu never sleeps — it either stays alive (fed) or dies
				if gu.IsVital {
					gu.Status = "DEAD"
					logger.S.Infof("Cultivation Log: Player %s's VITAL Gu %s (%s) starved and DIED — HP backlash incoming", username, specs.Name, gu.InstanceID[:8])
					playerHPBacklash += 80.0
					auditlog.Log(tx, auditlog.Entry{
						PlayerID: &pUUID, Action: "gu.starve_vital_death", EntityType: "gu_instance",
						EntityID: gu.InstanceID, RequestSummary: specs.Name + " | bản mệnh Cổ chết đói",
						Success: false, ErrorMessage: "Chết đói",
					})
				} else {
					// Roll for sleep vs death (20% Hibernation, 80% Dead)
					roll := rand.Intn(100)
					if roll < 20 {
						gu.Status = "SLEEPING"
						logger.S.Infof("Cultivation Log: Player %s's Gu %s (%s) starved and went into HIBERNATION", username, specs.Name, gu.InstanceID[:8])
						newInventory = append(newInventory, gu)
					} else {
						gu.Status = "DEAD"
						logger.S.Infof("Cultivation Log: Player %s's Gu %s (%s) starved and DIED", username, specs.Name, gu.InstanceID[:8])
						auditlog.Log(tx, auditlog.Entry{
							PlayerID: &pUUID, Action: "gu.starve_death", EntityType: "gu_instance",
							EntityID: gu.InstanceID, RequestSummary: specs.Name + " | Cổ chết đói",
							Success: false, ErrorMessage: "Chết đói",
						})
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

	// If no Gu was modified, no HP backlash, and no tick effects, return early
	if !updated && playerHPBacklash == 0 && tickStones == 0 && tickMana == 0 && tickHP == 0 && tickImmortalStones == 0 {
		return nil
	}

	// Apply ON_TICK stat changes
	if tickHP > 0 {
		hp += tickHP
		var baseHP float64
		_ = tx.QueryRow(`SELECT base_hp FROM players WHERE id = $1`, playerID).Scan(&baseHP)
		if hp > baseHP {
			hp = baseHP
		}
		updated = true
	}
	if tickStones > 0 {
		primevalStones += tickStones
	}
	if tickMana > 0 {
		currentMana += tickMana
		if currentMana > maxMana {
			currentMana = maxMana
		}
	}
	if tickImmortalStones > 0 {
		immortalStones += tickImmortalStones
	}

	// Apply HP Backlash if Vital Gu died
	if playerHPBacklash > 0 {
		hp -= playerHPBacklash
		if hp <= 0 {
			hp = 0
			_, err = tx.Exec(`UPDATE players SET hp = $1, primeval_stones = $2, current_mana = $3, max_mana = $4, immortal_stones = $5, active_combat = NULL WHERE id = $6`, hp, primevalStones, currentMana, maxMana, immortalStones, playerID)
		} else {
			_, err = tx.Exec(`UPDATE players SET hp = $1, primeval_stones = $2, current_mana = $3, max_mana = $4, immortal_stones = $5 WHERE id = $6`, hp, primevalStones, currentMana, maxMana, immortalStones, playerID)
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

	_, err = tx.Exec(`UPDATE players SET gu_inventory = $1, equipped_gus = $2, primeval_stones = $3, current_mana = $4, immortal_stones = $5 WHERE id = $6`, string(newGuInvRaw), string(newEquippedGusRaw), primevalStones, currentMana, immortalStones, playerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// getThienNguyenMultiplier returns the multiplier for Thien Nguyen Bao Lien series effects.
// Same rank = x2. Lower player rank = higher boost (high-rank gu helps low-rank cultivator more).
// Formula follows energy cost logic: multiplier = 2.0 * costFactor(guRank) / costFactor(playerRank).
// Immortal gu (6+) blocked for mortals (<6) in EquipGu already.
func getThienNguyenMultiplier(playerRank, guRank int) float64 {
	if playerRank == guRank {
		return 2.0
	}
	immortalCost := func(diff int) float64 {
		return math.Pow(100, float64(diff))
	}
	mortalCost := map[int]float64{1: 1, 2: 2, 3: 5, 4: 20, 5: 100}
	if playerRank >= 6 && guRank >= 6 {
		diff := playerRank - guRank
		if diff > 0 {
			return 2.0 / immortalCost(diff)
		}
		return 2.0 * immortalCost(-diff)
	}
	if playerRank >= 6 {
		return 2.0 / immortalCost(playerRank-guRank)
	}
	if guRank >= 6 {
		return 0
	}
	playerCost := mortalCost[playerRank]
	guCost := mortalCost[guRank]
	if playerCost == 0 || guCost == 0 {
		return 0
	}
	return 2.0 * guCost / playerCost
}
