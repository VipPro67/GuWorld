package handlers

import (
	"database/sql"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// finalizeAutoCombatOutcome applies the shared tail of an auto-combat after the
// victory/defeat branch: ON_COMBAT_END Gu events, tribulation aperture/enemy
// sync, NaN guards, the player row persist, simulated allies' status updates,
// pet death/HP persistence, and the final player Gu sync.
//
// Shared by AutoCombatStart and AutoResolveCombat so post-combat state is
// identical on both paths.
func finalizeAutoCombatOutcome(tx *sql.Tx, player *models.Player, simState *simulator.CombatState, isVirtual, isTribulation, alliesAlive bool, updateQuery string, args []interface{}, finalPlayerHP, finalMana float64, before models.GuInventory, logs []simulator.CombatLog) ([]simulator.CombatLog, error) {
	// Fire ON_COMBAT_END on equipped ACTIVE Gus
	var combatEndMsg string
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				if msg, err := HandleGuEvent(player, spec, gu, "ON_COMBAT_END"); err == nil && msg != "" {
					combatEndMsg = msg
				}
			}
		}
	}
	if combatEndMsg != "" {
		playerMsg := player.Username + " chứng kiến: " + combatEndMsg
		logs = append(logs, simulator.CombatLog{
			Turn:   simState.Turn,
			Action: "GU_EVENT",
			Detail: playerMsg,
		})
	}

	// ── Tribulation combat: persist aperture HP and sync enemy HP back to the event ──
	if isTribulation && player.ActiveCombat != nil {
		apertureHP := simState.ApertureHP
		if math.IsNaN(apertureHP) || math.IsInf(apertureHP, 0) || apertureHP < 0 {
			apertureHP = 0
		}
		_, errAp := tx.Exec(`
			UPDATE immortal_apertures SET aperture_hp = $1, updated_at = NOW()
			WHERE id = (SELECT aperture_id FROM tribulation_events WHERE id = $2 AND player_id = $3)
		`, apertureHP, player.ActiveCombat.TribulationEventID, player.ID)
		if errAp != nil {
			logger.S.Errorf("Failed to persist aperture HP after tribulation combat: %v", errAp)
		}

		// Sync enemy HP using LEAST so watchtower/worker damage is never overwritten upward.
		finalByID := make(map[string]float64)
		for _, e := range simState.Enemies {
			finalByID[e.ID] = e.HP
		}
		for _, en := range player.ActiveCombat.Enemies {
			finalHP, ok := finalByID[en.ID]
			if !ok {
				continue
			}
			if math.IsNaN(finalHP) || math.IsInf(finalHP, 0) || finalHP < 0 {
				finalHP = 0
			}
			_, errEn := tx.Exec(`
				UPDATE tribulation_enemies
				SET hp = LEAST(hp, $1), is_dead = CASE WHEN LEAST(hp, $1) <= 0 THEN true ELSE is_dead END, updated_at = NOW()
				WHERE id = $2
			`, finalHP, en.ID)
			if errEn != nil {
				logger.S.Errorf("Failed to sync tribulation enemy %s HP: %v", en.ID, errEn)
			}
		}
	}

	if math.IsNaN(finalPlayerHP) || math.IsInf(finalPlayerHP, 0) {
		finalPlayerHP = 0
	}
	if math.IsNaN(finalMana) || math.IsInf(finalMana, 0) {
		finalMana = 0
	}
	if math.IsNaN(player.HP) || math.IsInf(player.HP, 0) {
		player.HP = 1.0
	}

	_, err := tx.Exec(updateQuery, args...)
	if err != nil {
		logger.S.Errorf("Failed to save combat outcome for player %s: %v", player.ID, err)
		return logs, err
	}

	// Update simulated allies' database status (HP, Mana, Stones, Items)
	if !isVirtual {
		for _, simAlly := range simState.Allies {
			if simAlly.ID == player.ID.String() || simAlly.Kind == "Pet" {
				continue
			}
			allyUUID := uuid.MustParse(simAlly.ID)

			// Deduct items/stones used by simulator for this ally
			allyItemsUsed := simAlly.ItemsUsedInBattle
			if allyItemsUsed > 0 {
				remaining := allyItemsUsed
				invItems, _ := InventorySvc.ListByPlayer(simAlly.ID)
				for _, inv := range invItems {
					if remaining <= 0 {
						break
					}
					item, ok := models.ItemRegistry[inv.ItemID]
					if !ok || item.Type != "Consumable" {
						continue
					}
					if _, hasHeal := item.Effects["restore_hp"]; hasHeal {
						remove := inv.Quantity
						if remove > int64(remaining) {
							remove = int64(remaining)
						}
						_, cat, _ := InventorySvc.GetQuantityInAnyCategory(tx, simAlly.ID, inv.ItemID, "consumables", "materials", "other")
						if cat == "" {
							cat = "consumables"
						}
						if err := InventorySvc.RemoveItem(tx, simAlly.ID, inv.ItemID, cat, remove); err != nil {
							logger.S.Warnf("Failed to remove consumable %s for ally %s: %v", inv.ItemID, simAlly.ID, err)
						}
						remaining -= int(remove)
					}
				}
			}

			allyHP := simAlly.HP
			if allyHP <= 0 {
				allyHP = 1
			}

			_, errUpd := tx.Exec(`
				UPDATE players
				SET hp = $1, current_mana = $2, primeval_stones = $3, immortal_stones = $4, active_combat = NULL, path_masteries = $5
				WHERE id = $6
			`, allyHP, simAlly.Mana, simAlly.PrimevalStones, simAlly.ImmortalStones, models.PathMasteriesMap(simAlly.PathMasteries), allyUUID)
			if errUpd != nil {
				logger.S.Errorf("Failed to update ally %s outcome: %v", simAlly.Name, errUpd)
			}
		}
	} else {
		for _, simAlly := range simState.Allies {
			if simAlly.ID == player.ID.String() || simAlly.Kind == "Pet" {
				continue
			}
			_, _ = tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", uuid.MustParse(simAlly.ID))
		}
	}

	// Resolve pet deaths and HP after combat
	if !isVirtual {
		for _, simAlly := range simState.Allies {
			if simAlly.Kind != "Pet" {
				continue
			}
			if simAlly.HP <= 0 {
				// Pet died in combat — delete permanently
				_, _ = tx.Exec(`DELETE FROM player_pets WHERE id = $1 AND player_id = $2`, simAlly.ID, player.ID)
				_, _ = tx.Exec(`
					INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
					VALUES ($1, 'pet.died_in_combat', 'player_pet', $2, $3, '{}', '', false, 'Pet chết trong chiến đấu', NOW())
				`, player.ID, simAlly.ID, fmt.Sprintf("%s chết khi chiến đấu với %s", simAlly.Name, player.ActiveCombat.EnemyName))
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "PET_DEATH",
					Detail: fmt.Sprintf("%s đã chết trong chiến đấu!", simAlly.Name),
				})
			} else {
				// Pet survived — update HP
				_, _ = tx.Exec(`UPDATE player_pets SET hp = $1, updated_at = NOW() WHERE id = $2 AND player_id = $3`,
					float64(simAlly.HP), simAlly.ID, player.ID)
			}
		}

		// Persist challenged/defender pet damage & death (mine / real PVP)
		defenderID := ""
		if player.ActiveCombat != nil {
			defenderID = player.ActiveCombat.TargetPlayerID
		}
		if defenderID != "" && defenderID != "dummy" {
			for _, simEnemy := range simState.Enemies {
				if simEnemy.Kind != "Pet" {
					continue
				}
				if simEnemy.HP <= 0 {
					_, _ = tx.Exec(`DELETE FROM player_pets WHERE id = $1 AND player_id = $2`, simEnemy.ID, defenderID)
					_, _ = tx.Exec(`
						INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
						VALUES ($1, 'pet.died_in_combat', 'player_pet', $2, $3, '{}', '', false, 'Pet chết trong chiến đấu', NOW())
					`, defenderID, simEnemy.ID, fmt.Sprintf("%s chết khi phòng thủ trước %s", simEnemy.Name, player.ActiveCombat.EnemyName))
					logs = append(logs, simulator.CombatLog{
						Turn:   simState.Turn,
						Action: "PET_DEATH",
						Detail: fmt.Sprintf("%s đã chết trong chiến đấu!", simEnemy.Name),
					})
				} else {
					_, _ = tx.Exec(`UPDATE player_pets SET hp = $1, updated_at = NOW() WHERE id = $2 AND player_id = $3`,
						float64(simEnemy.HP), simEnemy.ID, defenderID)
				}
			}
		}
	}

	if alliesAlive && !isVirtual {
		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			logger.S.Errorf("Failed to sync player gus for player %s: %v", player.ID, err)
			return logs, err
		}
	}
	return logs, nil
}
