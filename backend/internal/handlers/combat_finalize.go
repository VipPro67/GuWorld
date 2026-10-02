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
			return logs, errAp
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
				return logs, errEn
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
		logger.S.Errorf("finalize player outcome update failed for player %s: %v", player.ID, err)
		return logs, fmt.Errorf("player outcome update: %w", err)
	}

	// Update simulated allies' database status (HP, Mana, Stones, Items)
	// NOTE: Pet and VariantNPC allies are persisted by their dedicated blocks
	// below; only real player allies are handled here.
	if !isVirtual {
		for _, simAlly := range simState.Allies {
			if simAlly.ID == player.ID.String() || simAlly.Kind == "Pet" || simAlly.Kind == "VariantNPC" {
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
						if _, spErr := tx.Exec("SAVEPOINT sp_ally_item"); spErr == nil {
							if err := InventorySvc.RemoveItem(tx, simAlly.ID, inv.ItemID, cat, remove); err != nil {
								logger.S.Warnf("Failed to remove consumable %s for ally %s: %v", inv.ItemID, simAlly.ID, err)
								_, _ = tx.Exec("ROLLBACK TO SAVEPOINT sp_ally_item")
							} else {
								_, _ = tx.Exec("RELEASE SAVEPOINT sp_ally_item")
							}
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
				logger.S.Errorf("finalize ally update failed for ally %s (%s): %v", simAlly.Name, simAlly.ID, errUpd)
				return logs, fmt.Errorf("ally %s update: %w", simAlly.ID, errUpd)
			}
		}
	} else {
		for _, simAlly := range simState.Allies {
			if simAlly.ID == player.ID.String() || simAlly.Kind == "Pet" || simAlly.Kind == "VariantNPC" {
				continue
			}
			allyID := simAlly.ID
			if allyUUID, errParse := uuid.Parse(allyID); errParse == nil {
				if _, spErr := tx.Exec("SAVEPOINT sp_reset_ally"); spErr == nil {
					if _, err := tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", allyUUID); err != nil {
						_, _ = tx.Exec("ROLLBACK TO SAVEPOINT sp_reset_ally")
					} else {
						_, _ = tx.Exec("RELEASE SAVEPOINT sp_reset_ally")
					}
				}
			}
		}

	}

	// Resolve pet deaths and HP after combat
	if !isVirtual {
		for _, simAlly := range simState.Allies {
			if simAlly.Kind != "Pet" {
				continue
			}
			if _, spErr := tx.Exec("SAVEPOINT sp_pet_ally"); spErr == nil {
				var errPet error
				if simAlly.HP <= 0 {
					// Pet died in combat — delete permanently
					_, errPet = tx.Exec(`DELETE FROM non_player_actors WHERE id = $1 AND owner_player_id = $2 AND actor_kind = 'COMPANION_BEAST'`, simAlly.ID, player.ID)
					if errPet == nil {
						_, _ = tx.Exec(`
							INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
							VALUES ($1, 'pet.died_in_combat', 'player_pet', $2, $3, '{}', '', false, 'Pet chết trong chiến đấu', NOW())
						`, player.ID, simAlly.ID, fmt.Sprintf("%s chết khi chiến đấu với %s", simAlly.Name, player.ActiveCombat.EnemyName))
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "PET_DEATH",
							Detail: fmt.Sprintf("%s đã chết trong chiến đấu!", simAlly.Name),
						})
					}
				} else {
					// Pet survived — update HP
					_, errPet = tx.Exec(`UPDATE non_player_actors SET hp = $1, updated_at = NOW(), version = version + 1 WHERE id = $2 AND owner_player_id = $3 AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'`,
						float64(simAlly.HP), simAlly.ID, player.ID)
				}
				if errPet != nil {
					logger.S.Warnf("Failed to update/delete pet %s for player %s: %v", simAlly.ID, player.ID, errPet)
					_, _ = tx.Exec("ROLLBACK TO SAVEPOINT sp_pet_ally")
				} else {
					_, _ = tx.Exec("RELEASE SAVEPOINT sp_pet_ally")
				}
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
				if _, spErr := tx.Exec("SAVEPOINT sp_pet_enemy"); spErr == nil {
					var errPet error
					if simEnemy.HP <= 0 {
						_, errPet = tx.Exec(`DELETE FROM non_player_actors WHERE id = $1 AND owner_player_id = $2 AND actor_kind = 'COMPANION_BEAST'`, simEnemy.ID, defenderID)
						if errPet == nil {
							_, _ = tx.Exec(`
								INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
								VALUES ($1, 'pet.died_in_combat', 'player_pet', $2, $3, '{}', '', false, 'Pet chết trong chiến đấu', NOW())
							`, defenderID, simEnemy.ID, fmt.Sprintf("%s chết khi phòng thủ trước %s", simEnemy.Name, player.ActiveCombat.EnemyName))
							logs = append(logs, simulator.CombatLog{
								Turn:   simState.Turn,
								Action: "PET_DEATH",
								Detail: fmt.Sprintf("%s đã chết trong chiến đấu!", simEnemy.Name),
							})
						}
					} else {
						_, errPet = tx.Exec(`UPDATE non_player_actors SET hp = $1, updated_at = NOW(), version = version + 1 WHERE id = $2 AND owner_player_id = $3 AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'`,
							float64(simEnemy.HP), simEnemy.ID, defenderID)
					}
					if errPet != nil {
						logger.S.Warnf("Failed to update/delete defender pet %s: %v", simEnemy.ID, errPet)
						_, _ = tx.Exec("ROLLBACK TO SAVEPOINT sp_pet_enemy")
					} else {
						_, _ = tx.Exec("RELEASE SAVEPOINT sp_pet_enemy")
					}
				}
			}
		}
	}

	if !isVirtual {
		persistNPC := func(combatant *simulator.CombatantAuto, ownerID interface{}) error {
			if combatant.Kind != "VariantNPC" {
				return nil
			}
			for guID, hunger := range combatant.GuHunger {
				if _, err := tx.Exec(`UPDATE player_gu SET hunger=$1,updated_at=NOW() WHERE id=$2
					AND EXISTS (SELECT 1 FROM actor_gu_equipment WHERE actor_id=$3 AND player_gu_id=$2)`, hunger, guID, combatant.ID); err != nil {
					return err
				}
			}
			if combatant.HP <= 0 {
				result, err := tx.Exec(`UPDATE non_player_actors SET hp=0,lifecycle_state='DEAD',execution_state='DEAD',formation_slot=0,updated_at=NOW(),version=version+1
					WHERE id=$1 AND owner_player_id=$2 AND actor_kind='VARIANT_NPC' AND lifecycle_state='ACTIVE'`, combatant.ID, ownerID)
				if err != nil {
					return err
				}
				if n, _ := result.RowsAffected(); n != 1 {
					return fmt.Errorf("variant NPC %s combat result conflicted", combatant.ID)
				}
				return releaseActorGu(tx, combatant.ID)
			} else {
				result, err := tx.Exec(`UPDATE non_player_actors SET hp=$1,updated_at=NOW(),version=version+1
					WHERE id=$2 AND owner_player_id=$3 AND actor_kind='VARIANT_NPC' AND lifecycle_state='ACTIVE'`, combatant.HP, combatant.ID, ownerID)
				if err != nil {
					return err
				}
				if n, _ := result.RowsAffected(); n != 1 {
					return fmt.Errorf("variant NPC %s combat result conflicted", combatant.ID)
				}
				return nil
			}
		}
		for _, ally := range simState.Allies {
			if err := persistNPC(ally, player.ID); err != nil {
				logger.S.Errorf("finalize attacker NPC persist failed for %s: %v", ally.ID, err)
				return logs, fmt.Errorf("attacker NPC %s persist: %w", ally.ID, err)
			}
		}
		defenderID := ""
		if player.ActiveCombat != nil {
			defenderID = player.ActiveCombat.TargetPlayerID
		}
		if defenderID != "" && defenderID != "dummy" {
			for _, enemy := range simState.Enemies {
				if err := persistNPC(enemy, defenderID); err != nil {
					logger.S.Errorf("finalize defender NPC persist failed for %s: %v", enemy.ID, err)
					return logs, fmt.Errorf("defender NPC %s persist: %w", enemy.ID, err)
				}
			}
		}
	}

	if alliesAlive && !isVirtual {
		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			logger.S.Errorf("finalize player Gu sync failed for player %s: %v", player.ID, err)
			return logs, fmt.Errorf("player Gu sync: %w", err)
		}
	}
	return logs, nil
}
