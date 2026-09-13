package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// buildSimCombatState constructs the deterministic simulator state for the
// player's active combat: initiator + pets + party allies + enemies.
//
// Shared by AutoCombatStart (manual replay) and AutoResolveCombat (skip battle)
// so both paths use identical combatants, turn order and calculations. The
// caller supplies the loadout (request loadout vs saved combat settings) and
// the already-derived isVirtual flag.
func buildSimCombatState(tx *sql.Tx, player *models.Player, loadout *simulator.CombatLoadout, isVirtual bool) (*simulator.CombatState, error) {
	simState := &simulator.CombatState{
		Turn:          0,
		Seed:          time.Now().UnixNano(),
		BossPhase:     0,
		VirtualMode:   isVirtual,
		ApertureHP:    player.ActiveCombat.ApertureHP,
		ApertureMaxHP: player.ActiveCombat.ApertureMaxHP,
	}

	simPlayer, err := buildSimCombatant(tx, player, isVirtual, true, loadout)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize initiator stats: %w", err)
	}
	simState.Allies = append(simState.Allies, simPlayer)

	// Load other party members as simulated allies
	for _, ally := range player.ActiveCombat.Allies {
		if ally.ID == player.ID.String() {
			continue
		}
		// Pet allies: build sim combatant directly from ally data
		if ally.Kind == "Pet" {
			simPet := &simulator.CombatantAuto{
				ID:      ally.ID,
				Name:    ally.Name,
				Kind:    "Pet",
				Rank:    ally.Rank,
				SubRank: ally.SubRank,
				GridX:   ally.GridX,
				GridY:   ally.GridY,
				MaxHP:   ally.MaxHP,
				HP:      ally.HP,
				MaxMana: 10,
				Mana:    10,
				BaseDmg: ally.BaseAttack,
				BaseDef: ally.BaseDefense,
				Speed:   ally.Speed,
				Actions: []simulator.SimAction{{
					ActionType: "BASIC_ATTACK",
					Name:       "Đánh thường",
					Damage:     ally.BaseAttack,
					ManaCost:   0,
				}},
			}
			simState.Allies = append(simState.Allies, simPet)
			continue
		}
		// Party member allies
		allyUUID, errParse := uuid.Parse(ally.ID)
		if errParse == nil {
			mPlayer, errMember := evaluateManaTxEquippedReadonly(tx, allyUUID)
			if errMember == nil {
				simAlly, errBuild := buildSimCombatant(tx, &mPlayer, isVirtual, false, nil)
				if errBuild == nil {
					simAlly.Kind = "Ally"
					simAlly.GridX = ally.GridX
					simAlly.GridY = ally.GridY
					simAlly.HP = ally.HP // Use their combat starting HP
					if simAlly.HP > simAlly.MaxHP {
						simAlly.HP = simAlly.MaxHP
					}
					simState.Allies = append(simState.Allies, simAlly)
				}
			}
		}
	}

	// Build enemies
	for idx := range player.ActiveCombat.Enemies {
		enemy := &player.ActiveCombat.Enemies[idx]
		simEnemy := buildSimEnemyCombatant(tx, enemy, player.ActiveCombat)
		simState.Enemies = append(simState.Enemies, simEnemy)
	}

	return simState, nil
}

// applyLoveGuPenalties scans the combat log for Chaos Gamble (Love Gu) penalties
// and applies them to the player. Shared by AutoCombatStart and AutoResolveCombat
// so both paths penalize identically.
func applyLoveGuPenalties(player *models.Player, logs []simulator.CombatLog) {
	loveAptitudePenalty := 0
	loveLifespanPenalty := 0.0
	loveStonesPenalty := false
	for _, logMsg := range logs {
		if logMsg.Action != "CHAOS_GAMBLE" {
			continue
		}
		if strings.Contains(logMsg.Detail, "Trừ 10% Tư chất gốc vĩnh viễn") {
			loveAptitudePenalty += 10
		} else if strings.Contains(logMsg.Detail, "Trừ 5 năm tuổi thọ vĩnh viễn") {
			loveLifespanPenalty += 5.0
		} else if strings.Contains(logMsg.Detail, "Trừ 50% Nguyên Thạch hiện tại") {
			loveStonesPenalty = true
		}
	}

	if loveAptitudePenalty > 0 {
		player.AptitudePercent -= loveAptitudePenalty
		if player.AptitudePercent < 1 {
			player.AptitudePercent = 1
		}
	}
	if loveLifespanPenalty > 0 && !gamefx.Get(player).Immortal {
		player.Lifespan = math.Max(0.0, player.Lifespan-loveLifespanPenalty)
		if player.Lifespan <= 0 {
			player.IsDead = true
		}
	}
	if loveStonesPenalty {
		player.PrimevalStones = player.PrimevalStones / 2
	}
}

// combatOutcome derives the post-simulation outcome from the sim state.
// alliesAlive is true only when the initiator is alive AND all enemies are dead.
// Returns the initiator's final HP too. Shared by AutoCombatStart and
// AutoResolveCombat so both paths judge victory identically.
func combatOutcome(simState *simulator.CombatState, playerID string) (enemiesAlive, alliesAlive bool, finalPlayerHP float64) {
	for _, e := range simState.Enemies {
		if e.HP > 0 {
			enemiesAlive = true
			break
		}
	}
	for _, a := range simState.Allies {
		if a.ID == playerID {
			alliesAlive = (a.HP > 0) && !enemiesAlive
			finalPlayerHP = a.HP
		}
	}
	return
}

// applySimConsumption deducts simulator-consumed healing items, primeval stones,
// immortal stones and equipped-Gu hunger from the player, returning the usage
// counts plus the corresponding log lines. Shared by AutoCombatStart and
// AutoResolveCombat so resource consumption is identical on both paths.
func applySimConsumption(tx *sql.Tx, player *models.Player, simPlayer *simulator.CombatantAuto, isVirtual bool, turn int) (itemsUsed int, stonesSpent, immortalStonesSpent int64, usageLogs []simulator.CombatLog) {
	itemsUsed = simPlayer.ItemsUsedInBattle

	if !isVirtual {
		if itemsUsed > 0 {
			remaining := itemsUsed
			invItems, _ := InventorySvc.ListByPlayer(player.ID.String())
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
					_, cat, _ := InventorySvc.GetQuantityInAnyCategory(tx, player.ID.String(), inv.ItemID, "consumables", "materials", "other")
					if cat == "" {
						cat = "consumables"
					}
					if err := InventorySvc.RemoveItem(tx, player.ID.String(), inv.ItemID, cat, remove); err != nil {
						logger.S.Warnf("Failed to remove consumable %s: %v", inv.ItemID, err)
					}
					remaining -= int(remove)
				}
			}
		}

		stonesSpent = player.PrimevalStones - simPlayer.PrimevalStones
		immortalStonesSpent = player.ImmortalStones - simPlayer.ImmortalStones
		player.PrimevalStones = simPlayer.PrimevalStones
		player.ImmortalStones = simPlayer.ImmortalStones

		if simPlayer != nil && len(simPlayer.GuUsedCounts) > 0 {
			beforeGus := append(models.GuInventory(nil), player.GuInventory...)
			hungerDeducted := false
			for idx := range player.GuInventory {
				g := &player.GuInventory[idx]
				if !g.Equipped {
					continue
				}
				if count, used := simPlayer.GuUsedCounts[g.GuID]; used && count > 0 {
					g.Hunger -= count
					if g.Hunger < 0 {
						g.Hunger = 0
					}
					hungerDeducted = true
				}
			}
			if hungerDeducted {
				if errSync := db.SyncPlayerGus(tx, player.ID.String(), beforeGus, player.GuInventory); errSync != nil {
					logger.S.Warnf("Failed to sync player Gu hunger after auto combat: %v", errSync)
				}
			}
		}
	}

	if itemsUsed > 0 {
		usageLogs = append(usageLogs, simulator.CombatLog{
			Turn:   turn,
			Action: "AUTO_ITEM_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %d dược liệu hồi phục trong chiến đấu.", itemsUsed),
		})
	}
	if stonesSpent > 0 {
		usageLogs = append(usageLogs, simulator.CombatLog{
			Turn:   turn,
			Action: "AUTO_STONE_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %d Nguyên Thạch hồi phục chân nguyên.", stonesSpent),
		})
	}
	if immortalStonesSpent > 0 {
		usageLogs = append(usageLogs, simulator.CombatLog{
			Turn:   turn,
			Action: "AUTO_STONE_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %d Tiên Nguyên Thạch hồi phục tiên nguyên.", immortalStonesSpent),
		})
	}
	return
}

// combatTypeLabel returns the human-readable enemy-type label used in the
// COMBAT_TYPE log header. Shared by both auto-combat paths.
func combatTypeLabel(cm *models.CombatState) string {
	if cm == nil {
		return "Dã ngoại"
	}
	switch cm.EnemyType {
	case "MINE_PVE":
		return "Mỏ Linh Thạch (Yêu quái)"
	case "PVP":
		return "Đấu trường / PVP"
	default:
		return "Dã ngoại"
	}
}
