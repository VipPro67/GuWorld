package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)
// AutoResolveCombat runs a combat simulation using the player's saved combat settings
// and applies the results to the database. Used for "skip battle" feature.
// Returns: winner ("Phe Ta" or "Phe Địch"), logs, summary info map, error.
func AutoResolveCombat(tx *sql.Tx, player *models.Player) (string, []simulator.CombatLog, map[string]interface{}, error) {
	if player.ActiveCombat == nil {
		return "", nil, nil, fmt.Errorf("no active combat")
	}

	isVirtual := player.ActiveCombat.CombatMode == "virtual"
	isTribulation := player.ActiveCombat.TribulationEventID != ""

	seed := time.Now().UnixNano()
	simState := &simulator.CombatState{
		Turn:          0,
		Seed:          seed,
		BossPhase:     0,
		VirtualMode:   isVirtual,
		ApertureHP:    player.ActiveCombat.ApertureHP,
		ApertureMaxHP: player.ActiveCombat.ApertureMaxHP,
	}

	// Build loadout from player's saved combat settings
	loadout := simulator.CombatLoadout{
		AutoHealHP:      player.CombatSettings.AutoHealHP,
		HealThreshold:   player.CombatSettings.HealThreshold,
		AutoRestoreMana: player.CombatSettings.AutoRestoreMana,
		ManaThreshold:   player.CombatSettings.ManaThreshold,
	}

	simPlayer, err := buildSimCombatant(tx, player, isVirtual, true, &loadout)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to initialize initiator stats: %w", err)
	}
	simState.Allies = append(simState.Allies, simPlayer)

	// Load other party members as simulated allies
	for _, ally := range player.ActiveCombat.Allies {
		if ally.ID == player.ID.String() {
			continue
		}
		if ally.Kind == "Pet" {
			simPet := &simulator.CombatantAuto{
				ID:      ally.ID,
				Name:    ally.Name,
				Kind:    "Pet",
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
		allyUUID, errParse := uuid.Parse(ally.ID)
		if errParse == nil {
			mPlayer, errMember := evaluateManaTxEquippedReadonly(tx, allyUUID)
			if errMember == nil {
				simAlly, errBuild := buildSimCombatant(tx, &mPlayer, isVirtual, false, nil)
				if errBuild == nil {
					simAlly.Kind = "Ally"
					simAlly.GridX = ally.GridX
					simAlly.GridY = ally.GridY
					simAlly.HP = ally.HP
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
		eHP := enemy.HP
		if eHP <= 0 {
			eHP = enemy.MaxHP
		}
		enemyMana := enemy.CurrentMana
		if isVirtual {
			eHP = enemy.MaxHP
			enemyMana = enemy.MaxMana
		}
		enemyShield := 0.0
		for _, guInst := range enemy.GuInventory {
			if guInst.Status != "ACTIVE" {
				continue
			}
			spec, ok := models.GetGu(guInst.GuID, guInst.Rank)
			if !ok {
				continue
			}
			if spec.Effects.Active != nil && spec.Effects.Active.ShieldFlat > 0 {
				enemyShield += spec.Effects.Active.ShieldFlat
			}
		}

		enemyKind := "Enemy"
		if enemy.Kind == "Pet" {
			enemyKind = "Pet"
		}
		simEnemy := &simulator.CombatantAuto{
			ID:            enemy.ID,
			Name:          enemy.Name,
			Kind:          enemyKind,
			GridX:         enemy.GridX,
			GridY:         enemy.GridY,
			MaxHP:         enemy.MaxHP,
			HP:            eHP,
			MaxMana:       enemy.MaxMana,
			Mana:          enemyMana,
			BaseDmg:       enemy.BaseAttack,
			BaseDef:       enemy.BaseDefense,
			Speed:         enemy.Speed,
			VirtualShield: enemyShield,
			DaoMarks: func() simulator.DaoMarks {
				dm := make(simulator.DaoMarks)
				for k, v := range enemy.DaoMarks {
					dm[k] = v
				}
				return dm
			}(),
		}
		if enemy.Kind == "Boss" {
			simEnemy.Kind = "Boss"
		}
		var enemyActions []simulator.SimAction
		for _, guInst := range enemy.GuInventory {
			if guInst.Status != "ACTIVE" {
				continue
			}
			spec, ok := models.GetGu(guInst.GuID, guInst.Rank)
			if !ok {
				continue
			}
			if hasCombatActiveEffects(&spec) {
				act, err := buildSimActionFromGu(enemy, &spec, "GU_SKILL")
				if err != nil {
					continue
				}
				act.SourceID = guInst.GuID
				enemyActions = append(enemyActions, act)
			}
		}
		enemyActions = append(enemyActions, simulator.SimAction{
			ActionType: "BASIC_ATTACK",
			Name:       "Đánh thường",
			Damage:     float64(enemy.BaseAttack),
			ManaCost:   0,
		})
		simEnemy.Actions = enemyActions
		simState.Enemies = append(simState.Enemies, simEnemy)
	}

	// Run Simulator
	logs := simulator.Simulate(simState)

	enemiesAlive := false
	for _, e := range simState.Enemies {
		if e.HP > 0 {
			enemiesAlive = true
			break
		}
	}

	var alliesAlive bool
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			alliesAlive = (a.HP > 0) && !enemiesAlive
			break
		}
	}

	if !alliesAlive {
		return "Phe Địch", nil, nil, fmt.Errorf("AUTO_COMBAT_DEFEAT")
	}

	// Apply outcome to DB (simplified from AutoCombatStart)
	before := append(models.GuInventory(nil), player.GuInventory...)

	// Scan logs for Love Gu penalties
	loveAptitudePenalty := 0
	loveLifespanPenalty := 0.0
	loveStonesPenalty := false
	for _, logMsg := range logs {
		if logMsg.Action == "CHAOS_GAMBLE" {
			if strings.Contains(logMsg.Detail, "Trừ 10% Tư chất gốc vĩnh viễn") {
				loveAptitudePenalty += 10
			} else if strings.Contains(logMsg.Detail, "Trừ 5 năm tuổi thọ vĩnh viễn") {
				loveLifespanPenalty += 5.0
			} else if strings.Contains(logMsg.Detail, "Trừ 50% Nguyên Thạch hiện tại") {
				loveStonesPenalty = true
			}
		}
	}

	if loveAptitudePenalty > 0 {
		player.AptitudePercent -= loveAptitudePenalty
		if player.AptitudePercent < 1 {
			player.AptitudePercent = 1
		}
	}
	if loveLifespanPenalty > 0 {
		player.Lifespan = math.Max(1.0, player.Lifespan-loveLifespanPenalty)
	}
	if loveStonesPenalty {
		player.PrimevalStones = player.PrimevalStones / 2
	}

	var finalPlayerHP float64
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			finalPlayerHP = a.HP
		}
	}

	itemsUsed := simPlayer.ItemsUsedInBattle
	stonesSpent := int64(0)
	immortalStonesSpent := int64(0)

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

	var finalMana float64
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			finalMana = a.Mana
			break
		}
	}

	// Log item/mana usage
	if itemsUsed > 0 {
		logs = append(logs, simulator.CombatLog{
			Turn:   simState.Turn,
			Action: "AUTO_ITEM_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %d dược liệu hồi phục trong chiến đấu.", itemsUsed),
		})
	}
	if stonesSpent > 0 {
		logs = append(logs, simulator.CombatLog{
			Turn:   simState.Turn,
			Action: "AUTO_STONE_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %d Nguyên Thạch hồi phục chân nguyên.", stonesSpent),
		})
	}
	if immortalStonesSpent > 0 {
		logs = append(logs, simulator.CombatLog{
			Turn:   simState.Turn,
			Action: "AUTO_STONE_USAGE",
			Detail: fmt.Sprintf("Đã tự động dùng %d Tiên Nguyên Thạch hồi phục tiên nguyên.", immortalStonesSpent),
		})
	}

	var updateQuery string
	var args []interface{}

	enemyName := ""
	enemyType := ""
	if player.ActiveCombat != nil {
		enemyName = player.ActiveCombat.EnemyName
		switch player.ActiveCombat.EnemyType {
		case "MINE_PVE":
			enemyType = "Mỏ Linh Thạch (Yêu quái)"
		case "PVP":
			enemyType = "Đấu trường / PVP"
		default:
			enemyType = "Dã ngoại"
		}
	}
	logs = append([]simulator.CombatLog{{
		Turn:   0,
		Action: "COMBAT_TYPE",
		Detail: fmt.Sprintf("=== %s ===", enemyType),
	}}, logs...)

	summary := make(map[string]interface{})

	if alliesAlive {
		if isVirtual {
			if err := InventorySvc.AddItem(tx, player.ID.String(), "arena_coin", "materials", 50); err != nil {
				logger.S.Errorf("Failed to award arena_coin to %s: %v", player.ID, err)
			} else {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "ARENA_REWARD",
					Detail: "Chiến thắng Võ Đài! Nhận thưởng 50 Xu Đấu Trường.",
				})
			}
			updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
			args = append(args, player.ID)
			summary["reward"] = "50 Xu Đấu Trường"
		} else {
			var partyMembers []uuid.UUID
			for _, ally := range player.ActiveCombat.Allies {
				if ally.Kind == "Pet" {
					continue
				}
				allyUUID, errParse := uuid.Parse(ally.ID)
				if errParse == nil {
					partyMembers = append(partyMembers, allyUUID)
				}
			}

			var lootDetailLogs []string
			if player.ActiveCombat != nil && player.ActiveCombat.LootTableID != "" {
				lt, err := db.LoadLootTable(tx, player.ActiveCombat.LootTableID)
				if err == nil {
					loot := RollLoot(lt, player.ActiveCombat.EnemyRank)

					if player.ActiveCombat.EnemyGuInventory != nil && len(player.ActiveCombat.EnemyGuInventory) > 0 {
						for _, eg := range player.ActiveCombat.EnemyGuInventory {
							guSpec, _ := models.GetGuAny(eg.GuID)
							dropChance := GetGuDropChancePercent(guSpec.Rank, isBossEnemy(player.ActiveCombat))
							if rand.Float64()*100.0 < dropChance {
								isImmortal := guSpec.IsImmortal || guSpec.Rank >= 6 || models.HasImmortalRank(eg.GuID)
								if isImmortal {
									exists, err := db.CheckImmortalGuExists(tx, eg.GuID, guSpec.Rank, nil)
									if err == nil && exists {
										continue
									}
								}
								loot.Gus = append(loot.Gus, eg.GuID)
							}
						}
					}

					if len(partyMembers) > 1 {
						lootDetailLogs = append(lootDetailLogs, "=== PHÂN CHIA CHIẾN LỢI PHẨM TỔ ĐỘI ===")
						stoneQty := loot.Items["primeval_stone"]
						if stoneQty > 0 {
							splitStones := stoneQty / len(partyMembers)
							if splitStones < 1 {
								splitStones = 1
							}
							for _, memberID := range partyMembers {
								if memberID == player.ID {
									player.PrimevalStones += int64(splitStones)
									lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch.", splitStones))
								} else {
									_, _ = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", splitStones, memberID)
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", memberID).Scan(&uname)
									lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s nhận được %d Nguyên Thạch.", uname, splitStones))
								}
							}
						}

						for matID, qty := range loot.Items {
							if matID == "primeval_stone" {
								continue
							}
							itemName := matID
							if item, ok := models.ItemRegistry[matID]; ok {
								itemName = item.Name
							}
							splitQty := qty / len(partyMembers)
							if splitQty < 1 {
								splitQty = 1
							}
							for _, memberID := range partyMembers {
								if memberID == player.ID {
									_ = InventorySvc.AddItem(tx, memberID.String(), matID, "materials", int64(splitQty))
									lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn nhận được %dx %s.", splitQty, itemName))
								} else {
									_ = InventorySvc.AddItem(tx, memberID.String(), matID, "materials", int64(splitQty))
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", memberID).Scan(&uname)
									lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s nhận được %dx %s.", uname, splitQty, itemName))
								}
							}
						}
					} else {
						for matID, qty := range loot.Items {
							itemName := matID
							if item, ok := models.ItemRegistry[matID]; ok {
								itemName = item.Name
							}
							if matID == "primeval_stone" {
								player.PrimevalStones += int64(qty)
								lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Nhận được %d Nguyên Thạch.", qty))
							} else {
								_ = InventorySvc.AddItem(tx, player.ID.String(), matID, "materials", int64(qty))
								lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Nhận được %dx %s.", qty, itemName))
							}
						}
						for _, guID := range loot.Gus {
							_ = InventorySvc.AddItem(tx, player.ID.String(), guID, "gu", 1)
							lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Nhận được Cổ Trùng: %s!", guID))
						}
					}
					if len(lootDetailLogs) > 0 {
						lootStr := strings.Join(lootDetailLogs, "\n")
						notify.Send(tx, player.ID, "combat_loot", "Chiến Lợi Phẩm (Tự Động)", lootStr)
					}
					for _, l := range lootDetailLogs {
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "LOOT",
							Detail: l,
						})
					}
				}
			}

			// Mine occupation
			if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 {
				slotID := player.ActiveCombat.MineSlotID
				var rarity string
				var mineType string
				var currentOccupierStr *string
				var monsterIDStr *string
				var lastClaimedAt *time.Time
				err = tx.QueryRow("SELECT rarity, mine_type, occupier_id, monster_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE", slotID).Scan(&rarity, &mineType, &currentOccupierStr, &monsterIDStr, &lastClaimedAt)
				if err == nil && currentOccupierStr != nil {
					if defenderUUID, errParse := uuid.Parse(*currentOccupierStr); errParse == nil && defenderUUID != player.ID {
						notify.Send(tx, defenderUUID, "mine_lost", "Mỏ bị đánh chiếm!",
							fmt.Sprintf("Mỏ #%d (%s) của bạn đã bị đạo hữu [%s] tấn công và đánh chiếm thành công!", slotID, rarity, player.Username))
					}
					if lastClaimedAt != nil {
						stones, qi, chests := CalculateMineRewards(slotID, rarity, *lastClaimedAt)
						stonesHalf := stones / 2
						qiHalf := qi / 2.0
						chestsHalf := chests / 2
						if stonesHalf > 0 || qiHalf > 0.0 || chestsHalf > 0 {
							if errMine := AddMineRewardsToPlayerTx(tx, player.ID, stonesHalf, qiHalf, chestsHalf, mineType); errMine != nil {
								logger.S.Errorf("Failed to add mine rewards to player %s (slot %d): %v", player.ID, slotID, errMine)
							} else {
								if mineType == "immortal" {
									player.ImmortalStones += int64(stonesHalf)
								} else {
									player.PrimevalStones += int64(stonesHalf)
								}
								var stoneName = "Nguyên Thạch"
								if mineType == "immortal" {
									stoneName = "Tiên Nguyên Thạch"
								}
								mineLootMsg := fmt.Sprintf("Cướp mỏ thành công! +%d %s, +%.0f Nhân Khí, +%d Rương Mỏ", stonesHalf, stoneName, qiHalf, chestsHalf)
								notify.Send(tx, player.ID, "combat_loot", "Chiến Lợi Phẩm (Cướp Mỏ)", mineLootMsg)
								logs = append(logs, simulator.CombatLog{
									Turn:   simState.Turn,
									Action: "MINE_LOOT",
									Detail: mineLootMsg,
								})
							}
						} else {
							logs = append(logs, simulator.CombatLog{
								Turn:   simState.Turn,
								Action: "MINE_LOOT",
								Detail: "Mỏ đối thủ chưa tích lũy đủ tài nguyên để cướp.",
							})
						}
					}
				} else if err == nil {
					logs = append(logs, simulator.CombatLog{
						Turn:   simState.Turn,
						Action: "MINE_LOOT",
						Detail: "Mỏ không có người chiếm giữ, không thể cướp tài nguyên.",
					})
				}

				now := time.Now()
				_, errUpdate := tx.Exec(`
					UPDATE mine_slots
					SET occupier_id = $1, monster_id = NULL, occupied_at = $2, last_claimed_at = $3
					WHERE id = $4
				`, player.ID, now, now, slotID)
				if errUpdate == nil {
					logs = append(logs, simulator.CombatLog{
						Turn:   simState.Turn,
						Action: "MINE_OCCUPY",
						Detail: fmt.Sprintf("Chiếm lĩnh thành công mỏ phẩm chất %s #%d!", rarity, slotID),
					})
					summary["mine_occupied"] = slotID
				} else {
					logger.S.Errorf("Failed to update mine slot in auto combat victory: %v", errUpdate)
				}
			}

			if player.ClearedCells == nil {
				player.ClearedCells = make(models.ClearedCellsMap)
			}
			coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
			player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			clearedCellsJSON, _ := json.Marshal(player.ClearedCells)

			player.HP = float64(finalPlayerHP)
			updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, primeval_stones = $3, immortal_stones = $4, path_masteries = $5, aptitude_percent = $6, lifespan = $7, cleared_cells = $8, version = version + 1 WHERE id = $9`
			args = append(args, player.HP, finalMana, player.PrimevalStones, player.ImmortalStones, player.PathMasteries, player.AptitudePercent, player.Lifespan, clearedCellsJSON, player.ID)

			summary["loot"] = lootDetailLogs
		}
	} else {
		var defeatMsg string
		if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 && player.ActiveCombat.TargetPlayerID != "" {
			if defenderUUID, errParse := uuid.Parse(player.ActiveCombat.TargetPlayerID); errParse == nil && defenderUUID != player.ID {
				var rarity string
				_ = tx.QueryRow("SELECT rarity FROM mine_slots WHERE id = $1", player.ActiveCombat.MineSlotID).Scan(&rarity)
				notify.Send(tx, defenderUUID, "mine_defended", "Phòng thủ Mỏ thành công!",
					fmt.Sprintf("Đạo hữu [%s] đã tấn công Mỏ #%d (%s) của bạn nhưng đã bị lực lượng phòng thủ đánh bại!", player.Username, player.ActiveCombat.MineSlotID, rarity))
			}
		}
		if isTribulation {
			defeatMsg = "Đạo hữu kiệt sức trước quái tai kiếp! Tiên Khiếu phải gánh chịu phần sát thương còn lại."
		} else {
			player.CurrentRegion, player.X, player.Y = NearestHub(player.X, player.Y)
			dropSummary := applyDeathDrop(tx, player, nil)
			if dropSummary != "" {
				defeatMsg = fmt.Sprintf("Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết nhưng rơi rớt mất: %s.", dropSummary)
			} else {
				defeatMsg = "Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết."
			}
		}

		for _, ally := range player.ActiveCombat.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil && allyUUID != player.ID {
				_, _ = tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", allyUUID)
			}
		}

		logs = append(logs, simulator.CombatLog{
			Turn:   simState.Turn,
			Action: "BATTLE_END",
			Detail: defeatMsg,
		})

		updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, current_region = $3, x = $4, y = $5, aptitude_percent = $6, lifespan = $7, primeval_stones = $8, immortal_stones = $9, path_masteries = $10, version = version + 1 WHERE id = $11`
		args = append(args, player.HP, finalMana, player.CurrentRegion, player.X, player.Y, player.AptitudePercent, player.Lifespan, player.PrimevalStones, player.ImmortalStones, player.PathMasteries, player.ID)
	}

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

	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		logger.S.Errorf("Failed to save combat outcome for player %s: %v", player.ID, err)
		return "", nil, nil, fmt.Errorf("failed to save combat outcome: %w", err)
	}

	// Update simulated allies' database status (HP, Mana, Stones, Items)
	if !isVirtual {
		for _, simAlly := range simState.Allies {
			if simAlly.ID == player.ID.String() || simAlly.Kind == "Pet" {
				continue
			}
			allyUUID := uuid.MustParse(simAlly.ID)

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
			`, allyHP, simAlly.Mana, simAlly.PrimevalStones, simAlly.ImmortalStones, simAlly.PathMasteries, allyUUID)
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
				_, _ = tx.Exec(`UPDATE player_pets SET hp = $1, updated_at = NOW() WHERE id = $2 AND player_id = $3`,
					float64(simAlly.HP), simAlly.ID, player.ID)
			}
		}

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
			return "", nil, nil, fmt.Errorf("failed to save combat outcome: %w", err)
		}
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 3, $1) WHERE id = $2", MaxHumanQi, player.ID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to update human_qi: %w", err)
	}

	winner := "Phe Địch"
	if alliesAlive {
		winner = "Phe Ta"
	}

	// Save combat history in background
	go func() {
		summaryStr := buildCombatSummaryFromLogs(logs, finalPlayerHP, itemsUsed, stonesSpent)
		combatantInfosIface := make([]interface{}, 0)
		SaveCombatHistory(player.ID, winner, toInterfaceSlice(logs), summaryStr, combatantInfosIface, enemyName, enemyType, player.CurrentRegion)
	}()

	summary["winner"] = winner
	summary["enemy_name"] = enemyName
	summary["enemy_type"] = enemyType

	player.ActiveCombat = nil
	return winner, logs, summary, nil
}

// ── helpers ────────────────────────────────────────────────────────────

func scanKillerMoveRow(rows *sql.Rows) (models.KillerMove, error) {
	var m models.KillerMove
	err := rows.Scan(
		&m.ID, &m.PlayerID, &m.Name, &m.Path, &m.MoveType, &m.RankRequired,
		&m.CorePlayerGuID, &m.TotalManaCost, &m.SuccessRate, &m.DamageValue,
		&m.DefenseValue, &m.Cooldown, &m.ChannelTime, &m.Effects, &m.IsMastered,
		&m.IsReliable, &m.IsInstantCast, &m.CreatedAt, &m.UpdatedAt,
	)
	return m, err
}

func loadKillerMovesBatch(tx *sql.Tx, playerID uuid.UUID, ids []string) (map[string]models.KillerMove, map[string][]models.KillerMoveSlot, error) {
	var query string
	var args []any
	if ids == nil {
		query = `SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, success_rate, damage_value, defense_value, cooldown, channel_time, effects, is_mastered, is_reliable, is_instant_cast, created_at, updated_at 
		         FROM killer_moves WHERE player_id = $1`
		args = []any{playerID}
	} else {
		if len(ids) == 0 {
			return nil, nil, nil
		}
		query = `SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, success_rate, damage_value, defense_value, cooldown, channel_time, effects, is_mastered, is_reliable, is_instant_cast, created_at, updated_at 
		         FROM killer_moves WHERE player_id = $1 AND id = ANY($2)`
		args = []any{playerID, pq.Array(ids)}
	}

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	kmMap := make(map[string]models.KillerMove)
	var kmIDs []string
	for rows.Next() {
		m, err := scanKillerMoveRow(rows)
		if err != nil {
			return nil, nil, err
		}
		kmMap[m.ID] = m
		kmIDs = append(kmIDs, m.ID)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	slotsMap := make(map[string][]models.KillerMoveSlot)
	if len(kmIDs) > 0 {
		slotRows, err := tx.Query(`SELECT id, killer_move_id, player_gu_id, slot_type, slot_index, created_at 
		                           FROM killer_move_slots WHERE killer_move_id = ANY($1) 
		                           ORDER BY killer_move_id, slot_index ASC`, pq.Array(kmIDs))
		if err != nil {
			return nil, nil, err
		}
		defer slotRows.Close()

		for slotRows.Next() {
			var s models.KillerMoveSlot
			if err := slotRows.Scan(&s.ID, &s.KillerMoveID, &s.PlayerGuID, &s.SlotType, &s.SlotIndex, &s.CreatedAt); err != nil {
				return nil, nil, err
			}
			slotsMap[s.KillerMoveID] = append(slotsMap[s.KillerMoveID], s)
		}
		if err = slotRows.Err(); err != nil {
			return nil, nil, err
		}
	}

	return kmMap, slotsMap, nil
}

func buildSimActionFromGu(actor *models.Combatant, spec *models.GuDictionaryItem, actionType string) (simulator.SimAction, error) {
	// Mortal users cannot use immortal Gu that cost mana (skip the action).
	if !gamefx.CanUseGuWithMana(actor.Rank, spec) {
		return simulator.SimAction{}, fmt.Errorf("Tu vi %s chưa đủ để sử dụng Tiên Cổ %s (Rank %d) có hao phí!", getRankName(actor.Rank), spec.Name, spec.Rank)
	}

	dmg, def, cost, err := CalculateGuEffectForCombatant(actor, spec, false)
	if err != nil {
		return simulator.SimAction{
			ActionType: actionType,
			Name:       spec.Name,
			Path:       spec.Path,
			Rank:       spec.Rank,
			ManaCost:   cost,
		}, nil
	}

	act := simulator.SimAction{
		ActionType: actionType,
		Name:       spec.Name,
		Path:       spec.Path,
		Rank:       spec.Rank,
		Damage:     dmg,
		Defense:    def,
		ManaCost:   cost,
	}

	if actor.Rank < spec.Rank {
		act.BacklashDamage = 5.0 * float64(spec.Rank-actor.Rank)
	}

	if spec.Effects.Active != nil {
		active := spec.Effects.Active
		if active.ShieldFlat > 0 {
			act.Defense += active.ShieldFlat
		}
		// Registry-driven statuses (BLEED/POISON/BURN/FEAR/CONFUSE/STEALTH/
		// SLEEP/BLIND/CHARM/BIND + catalog). Resolved with real turns from
		// status_duration_ticks instead of a hardcoded 3.
		if active.ApplyStatus != "" {
			switch active.ApplyStatus {
			// Tier-based legacy statuses keep their existing field mapping.
			case "STUN", "SLOW", "SLOW1", "SLOW_DAMAGE", "WEAK", "FREEZE",
				"SILENCE", "THORNS", "REGEN", "ATK_UP", "DEF_UP", "TAUNT":
				// handled below
			default:
				if op, err := statusfx.ResolveStatus(active.ApplyStatus, active.StatusTickDamage, active.StatusDurationTicks); err == nil {
					act.StatusOps = append(act.StatusOps, op)
				} else {
					logger.S.Warnf("Gu %s: %v", spec.ID, err)
				}
			}
		}
		if active.ApplyStatus == "STUN" {
			act.StunTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "SLOW" {
			act.SlowTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "SLOW1" {
			act.Slow1Tier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "SLOW_DAMAGE" {
			act.SlowDamageTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "WEAK" {
			act.WeakTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "FREEZE" {
			act.FreezeTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "SILENCE" {
			act.SilenceTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "THORNS" {
			act.ThornsTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "REGEN" {
			act.RegenTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "ATK_UP" {
			act.AtkUpTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "DEF_UP" {
			act.DefUpTier = active.StatusDurationTicks
		}
		if active.ApplyStatus == "TAUNT" {
			act.TauntTier = active.StatusDurationTicks
		}
		if active.LifeStealRate > 0 {
			act.LifeStealPct += active.LifeStealRate
		}
		if active.HealFlat > 0 {
			act.Healing += active.HealFlat
		}
		if active.BloodDrain {
			act.LifeStealPct += 0.5
		}
		if active.SummonPhantom {
			act.PhantomDmg += actor.BaseAttack
		}
		if active.RefineManaTier > 0 {
			boost := active.RefineManaTier * 0.20
			act.Damage += act.Damage * boost
		}
		if active.WoodTransformation {
			act.Defense += 30.0
		}
		if active.DamageMult > 1.0 {
			act.DamageMult = active.DamageMult
		}
		if active.TargetSelector != "" {
			act.TargetType = active.TargetSelector
			act.TargetX = actor.GridX
			act.TargetY = actor.GridY
		}
		act.Bypass = active.CurveAttack
	}

	if act.TargetType == "" {
		act.TargetType = "SINGLE"
	}

	return act, nil
}

// hasCombatActiveEffects returns true if the Gu's active effects produce a meaningful combat action.
func hasCombatActiveEffects(spec *models.GuDictionaryItem) bool {
	if spec.Effects.Active == nil {
		return false
	}
	active := spec.Effects.Active
	return active.DamageFlat > 0 || active.DamageMult > 0 || active.ShieldFlat > 0 ||
		active.ApplyStatus != "" || active.LifeStealRate > 0 || active.HealFlat > 0 ||
		active.BloodDrain || active.SummonPhantom || active.RefineManaTier > 0 ||
		active.WoodTransformation || active.ManaRegen > 0 || active.CleansingPower > 0
}

func computeGlobalGuDamageMultiplier(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	multiplier := 1.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}

		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		if spec.Effects.Active.DamageMult > 1.0 {
			multiplier *= spec.Effects.Active.DamageMult
		}
	}
	return multiplier
}

// computeGlobalIgnoreDefense aggregates ignore_defense from all equipped active Gu (additive, capped at 0.8).
func computeGlobalIgnoreDefense(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) (flat float64, pct float64) {
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		flat += spec.Effects.Active.IgnoreDefense
		pct += spec.Effects.Active.IgnoreDefensePct
	}
	if pct > 0.8 {
		pct = 0.8
	}
	return flat, pct
}

// computeGlobalCritRate aggregates crit_rate_boost from all equipped active Gu (additive, capped at 0.8).
func computeGlobalCritRate(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	total := 0.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		total += spec.Effects.Active.CritRateBoost
	}
	if total > 0.8 {
		total = 0.8
	}
	return total
}

// computeGlobalSpeedBoost aggregates speed_boost_flat from all equipped active Gu (additive).
func computeGlobalSpeedBoost(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	total := 0.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		total += spec.Effects.Active.SpeedBoostFlat
	}
	return total
}

// computeGlobalAllStatsBuff aggregates all_stats_buff_percent from all equipped active Gu (additive, capped at 0.5).
func computeGlobalAllStatsBuff(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	total := 0.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok || spec.Effects.Active == nil {
			continue
		}
		total += spec.Effects.Active.AllStatsBuffPct
	}
	if total > 0.5 {
		total = 0.5
	}
	return total
}

func buildSimActionFromKillerMove(km models.KillerMove, _ []models.KillerMoveSlot, daoMarks models.DaoMarksMap) simulator.SimAction {
	act := simulator.SimAction{
		ActionType: "KILLER_MOVE",
		Name:       km.Name,
		Damage:     km.DamageValue,
		Defense:    km.DefenseValue,
		ManaCost:   km.TotalManaCost,
		Path:       []string{km.Path},
	}

	// Strip the dao bonus baked in at creation so the engine re-applies it
	// using the current dao marks (and reports it in the combat log).
	if mult, ok := simulator.DaoMultiplier(simulator.DaoMarks(daoMarks), act.Path); ok && mult > 1.0 {
		act.Damage /= mult
		act.Defense /= mult
	}

	var effects map[string]float64
	if err := json.Unmarshal(km.Effects, &effects); err == nil {
		if v, ok := effects["bleed"]; ok {
			act.BleedPerTurn += v
		}
		if v, ok := effects["poison"]; ok {
			act.PoisonPerTurn += v
		}
		if v, ok := effects["stun"]; ok {
			act.StunTier = int(v)
		}
		if v, ok := effects["life_steal"]; ok {
			act.LifeStealPct += v / 100.0
		}
		if v, ok := effects["blood_drain"]; ok {
			act.LifeStealPct += v / 100.0
		}
		if v, ok := effects["multi_hit"]; ok {
			act.MultiHit = int(v)
		}
		if v, ok := effects["phantom_attack"]; ok {
			act.PhantomDmg += v
		}
		if v, ok := effects["thorns_damage"]; ok {
			act.ThornsDmg += v
		}
		if v, ok := effects["heal"]; ok {
			act.Healing += v
		}
		// ── New tier-based effects ──
		if v, ok := effects["slow"]; ok {
			act.SlowTier = int(v)
		}
		if v, ok := effects["slow1"]; ok {
			act.Slow1Tier = int(v)
		}
		if v, ok := effects["slow_damage"]; ok {
			act.SlowDamageTier = int(v)
		}
		if v, ok := effects["weak"]; ok {
			act.WeakTier = int(v)
		}
		if v, ok := effects["freeze"]; ok {
			act.FreezeTier = int(v)
		}
		if v, ok := effects["silence"]; ok {
			act.SilenceTier = int(v)
		}
		if v, ok := effects["thorns_buff"]; ok {
			act.ThornsTier = int(v)
		}
		if v, ok := effects["regen"]; ok {
			act.RegenTier = int(v)
		}
		if v, ok := effects["atk_up"]; ok {
			act.AtkUpTier = int(v)
		}
		if v, ok := effects["def_up"]; ok {
			act.DefUpTier = int(v)
		}
		if v, ok := effects["taunt"]; ok {
			act.TauntTier = int(v)
		}
	}

	act.TargetType = "SINGLE"
	return act
}

