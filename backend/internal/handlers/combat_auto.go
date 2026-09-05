package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// AutoCombatStartRequest payload for starting automated combat
type AutoCombatStartRequest struct {
	Loadout simulator.CombatLoadout `json:"loadout" binding:"required"`
}

// CombatantInfo returned to frontend for visual replay
type CombatantInfo struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	MaxHP         int            `json:"max_hp"`
	HP            int            `json:"hp"`
	MaxMana       int            `json:"max_mana"`
	Mana          int            `json:"mana"`
	GridX         int            `json:"grid_x"`
	GridY         int            `json:"grid_y"`
	Shield        int            `json:"shield"`
	Damage        float64        `json:"damage,omitempty"`
	Defense       float64        `json:"defense,omitempty"`
	Speed         float64        `json:"speed,omitempty"`
	KindDetail    string             `json:"kind_detail,omitempty"`
	Rank          int                `json:"rank,omitempty"`
	SubRank       int                `json:"sub_rank,omitempty"`
	DaoMarks      models.DaoMarksMap `json:"dao_marks,omitempty"`
	PathMasteries map[string]int     `json:"path_masteries,omitempty"`
	GuInventory   models.GuInventory `json:"gu_inventory,omitempty"`
}

// AutoCombatStart initializes and fully simulates a combat, returning the deterministic log.
func AutoCombatStart(c *gin.Context) {
	var req AutoCombatStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
		return
	}

	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong trạng thái chiến đấu."})
		return
	}

	isVirtual := player.ActiveCombat.CombatMode == "virtual"
	isTribulation := player.ActiveCombat.TribulationEventID != ""

	seed := time.Now().UnixNano()
	simState := &simulator.CombatState{
		Turn:          0,
		Seed:          seed,
		BossPhase:     0,
		VirtualMode:   isVirtual,
		ApertureHP:    int(player.ActiveCombat.ApertureHP),
		ApertureMaxHP: int(player.ActiveCombat.ApertureMaxHP),
	}

	simPlayer, err := buildSimCombatant(tx, &player, isVirtual, true, &req.Loadout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize initiator stats"})
		return
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
				GridX:   ally.GridX,
				GridY:   ally.GridY,
				MaxHP:   int(ally.MaxHP),
				HP:      int(ally.HP),
				MaxMana: 10,
				Mana:    10,
				BaseDmg: int(ally.BaseAttack),
				BaseDef: int(ally.BaseDefense),
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
					simAlly.HP = int(ally.HP) // Use their combat starting HP
					if simAlly.HP > simAlly.MaxHP {
						simAlly.HP = simAlly.MaxHP
					}
					simState.Allies = append(simState.Allies, simAlly)
				}
			}
		}
	}

	// ── Build enemies ──
	for idx := range player.ActiveCombat.Enemies {
		enemy := &player.ActiveCombat.Enemies[idx]
		eHP := int(enemy.HP)
		if eHP <= 0 {
			eHP = int(enemy.MaxHP)
		}
		enemyMana := int(enemy.CurrentMana)
		if isVirtual {
			eHP = int(enemy.MaxHP)
			enemyMana = int(enemy.MaxMana)
		}
		// Calculate enemy VirtualShield from their Gu inventory
		enemyShield := 0
		for _, guInst := range enemy.GuInventory {
			if guInst.Status != "ACTIVE" {
				continue
			}
			spec, ok := models.GetGu(guInst.GuID, guInst.Rank)
			if !ok {
				continue
			}
			if spec.Effects.Active != nil && spec.Effects.Active.ShieldFlat > 0 {
				enemyShield += int(spec.Effects.Active.ShieldFlat)
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
			MaxHP:         int(enemy.MaxHP),
			HP:            eHP,
			MaxMana:       int(enemy.MaxMana),
			Mana:          enemyMana,
			BaseDmg:       int(enemy.BaseAttack),
			BaseDef:       int(enemy.BaseDefense),
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
		// Auto-select Gu actions from enemy inventory
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
		// Always add basic attack as fallback
		enemyActions = append(enemyActions, simulator.SimAction{
			ActionType: "BASIC_ATTACK",
			Name:       "Đánh thường",
			Damage:     float64(enemy.BaseAttack),
			ManaCost:   0,
		})
		simEnemy.Actions = enemyActions
		simState.Enemies = append(simState.Enemies, simEnemy)
	}

	// ── Build combatant info for frontend replay ──
	var combatantInfos []CombatantInfo
	for _, ally := range simState.Allies {
		combatantInfos = append(combatantInfos, CombatantInfo{
			ID:      ally.ID,
			Name:    ally.Name,
			Kind:    "ally",
			MaxHP:   ally.MaxHP,
			HP:      ally.HP,
			MaxMana: ally.MaxMana,
			Mana:    ally.Mana,
			GridX:   ally.GridX,
			GridY:   ally.GridY,
			Shield:  ally.VirtualShield,
		})
	}
	for _, e := range player.ActiveCombat.Enemies {
		combatantInfos = append(combatantInfos, CombatantInfo{
			ID:            e.ID,
			Name:          e.Name,
			Kind:          "enemy",
			MaxHP:         int(e.MaxHP),
			HP:            int(e.HP),
			MaxMana:       int(e.MaxMana),
			Mana:          int(e.CurrentMana),
			GridX:         e.GridX,
			GridY:         e.GridY,
			Damage:        e.BaseAttack,
			Defense:       e.BaseDefense,
			Speed:         e.Speed,
			KindDetail:    e.Kind,
			Rank:          e.Rank,
			SubRank:       e.SubRank,
			DaoMarks:      e.DaoMarks,
			PathMasteries: e.PathMasteries,
			GuInventory:   e.GuInventory,
		})
	}

	// 4. Run Simulator
	logs := simulator.Simulate(simState)

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

	// 5. Apply outcome to DB
	enemiesAlive := false
	for _, e := range simState.Enemies {
		if e.HP > 0 {
			enemiesAlive = true
			break
		}
	}

	var alliesAlive bool
	var finalPlayerHP int
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			alliesAlive = (a.HP > 0) && !enemiesAlive
			finalPlayerHP = a.HP
		}
	}

	// ── Deduct items/stones used by simulator (skip for virtual mode) ──
	itemsUsed := simPlayer.ItemsUsedInBattle
	stonesSpent := int64(0)
	immortalStonesSpent := int64(0)

	if !isVirtual {
		if itemsUsed > 0 {
			remaining := itemsUsed
			invItems, _ := InventorySvc.ListByPlayer(authID.String())
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
					_, cat, _ := InventorySvc.GetQuantityInAnyCategory(tx, authID.String(), inv.ItemID, "consumables", "materials", "other")
					if cat == "" {
						cat = "consumables"
					}
					if err := InventorySvc.RemoveItem(tx, authID.String(), inv.ItemID, cat, remove); err != nil {
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
	}

	// ── Log item/mana usage ──
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

	// ── Grab final mana from sim ──
	var finalMana int
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			finalMana = a.Mana
			break
		}
	}

	var updateQuery string
	var args []interface{}

	// ── Combat type label for logs ──
	enemyTypeName := "Dã ngoại"
	if player.ActiveCombat != nil {
		switch player.ActiveCombat.EnemyType {
		case "MINE_PVE":
			enemyTypeName = "Mỏ Linh Thạch (Yêu quái)"
		case "PVP":
			enemyTypeName = "Đấu trường / PVP"
		}
	}
	logs = append([]simulator.CombatLog{{
		Turn:   0,
		Action: "COMBAT_TYPE",
		Detail: fmt.Sprintf("=== %s ===", enemyTypeName),
	}}, logs...)

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
			if player.ActiveCombat != nil && player.ActiveCombat.TargetPlayerID != "" && player.ActiveCombat.TargetPlayerID != "dummy" {
				targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
				if parseErr == nil {
					var playerRank, targetRank int
					errPR := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", player.ID).Scan(&playerRank)
					errTR := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", targetUUID).Scan(&targetRank)
					if errPR == nil && errTR == nil && playerRank > targetRank {
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = -1 WHERE player_id = $1", player.ID)
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", playerRank, targetUUID)
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", targetRank, player.ID)
						_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", playerRank, targetUUID)
						_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", targetRank, player.ID)
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "ARENA_RANK_UP",
							Detail: fmt.Sprintf("Chúc mừng! Đạo hữu đã vượt cấp khiêu chiến thành công, thăng lên Hạng %d Võ Đài!", targetRank),
						})
					}
				}
			}
			updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
			args = append(args, player.ID)
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

					if len(player.ActiveCombat.EnemyGuInventory) > 0 {
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
							for q := 0; q < qty; q++ {
								winnerID := partyMembers[rand.Intn(len(partyMembers))]
								if winnerID == player.ID {
									if err := InventorySvc.AddItem(tx, player.ID.String(), matID, "materials", 1); err != nil {
										logger.S.Errorf("Failed to add loot item %s to player %s: %v", matID, player.ID, err)
									}
									lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn nhặt được 1x %s.", itemName))
								} else {
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)
									if err := InventorySvc.AddItem(tx, winnerID.String(), matID, "materials", 1); err != nil {
										logger.S.Errorf("Failed to distribute loot %s to party member %s: %v", matID, winnerID, err)
									}
									lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s nhặt được 1x %s.", uname, itemName))
								}
							}
						}

						for _, gID := range loot.Gus {
							winnerID := partyMembers[rand.Intn(len(partyMembers))]
							guSpec, _ := models.GetGuAny(gID)
							guName := gID
							if guSpec.Name != "" {
								guName = guSpec.Name
							}
							newGu := models.GuInstance{
								InstanceID: uuid.New().String(),
								GuID:       gID,
								Rank:       guSpec.Rank,
								Hunger:     100,
								Status:     "ACTIVE",
								Equipped:   false,
							}
							if winnerID == player.ID {
								player.GuInventory = append(player.GuInventory, newGu)
								lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Bạn bắt được 1x %s!", guName))
							} else {
								var uname string
								_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)
								_ = db.AddPlayerGu(tx, winnerID.String(), newGu)
								lootDetailLogs = append(lootDetailLogs, fmt.Sprintf("Đồng đội %s thu phục thành công Cổ trùng: %s!", uname, guName))
							}
						}
					} else {
						stoneQty := loot.Items["primeval_stone"]
						player.PrimevalStones += int64(stoneQty)
						for matID, qty := range loot.Items {
							if matID == "primeval_stone" {
								continue
							}
							InventorySvc.AddItem(tx, player.ID.String(), matID, "materials", int64(qty))
						}
						for _, guID := range loot.Gus {
							guSpec, _ := models.GetGuAny(guID)
							newGu := models.GuInstance{
								InstanceID: uuid.New().String(), GuID: guID, Rank: guSpec.Rank, Hunger: 0, Status: "ACTIVE", Equipped: false, IsVital: false,
							}
							player.GuInventory = append(player.GuInventory, newGu)
						}

						var lootParts []string
						if stoneQty > 0 {
							lootParts = append(lootParts, fmt.Sprintf("%d Nguyên Thạch", stoneQty))
						}
						for matID, qty := range loot.Items {
							if matID == "primeval_stone" || qty <= 0 {
								continue
							}
							itemName := matID
							if item, ok := models.ItemRegistry[matID]; ok {
								itemName = item.Name
							}
							lootParts = append(lootParts, fmt.Sprintf("%dx %s", qty, itemName))
						}
						for _, guID := range loot.Gus {
							guName := guID
							if guSpec, ok := models.GetGuAny(guID); ok {
								guName = guSpec.Name
							}
							lootParts = append(lootParts, fmt.Sprintf("1x %s (Cổ trùng)", guName))
						}

						if len(lootParts) > 0 {
							lootMsg := "Nhặt được: " + strings.Join(lootParts, ", ") + "."
							lootDetailLogs = append(lootDetailLogs, lootMsg)
							notify.Send(tx, player.ID, "combat_loot", "Chiến lợi phẩm", lootMsg)
						}
					}
				}
			}
			markBossCleared(tx, &player)

			for _, lMsg := range lootDetailLogs {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "LOOT",
					Detail: lMsg,
				})
			}

			// ── Mine combat victory: occupy mine + remove monster ──
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
								logs = append(logs, simulator.CombatLog{
									Turn:   simState.Turn,
									Action: "MINE_LOOT",
									Detail: fmt.Sprintf("Cướp mỏ thành công! +%d %s, +%.0f Nhân Khí, +%d Rương Mỏ", stonesHalf, stoneName, qiHalf, chestsHalf),
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
				} else {
					logger.S.Errorf("Failed to update mine slot in auto combat victory: %v", errUpdate)
				}
			}

			if player.ClearedCells == nil {
				player.ClearedCells = make(models.ClearedCellsMap)
			}
			if !isTribulation {
				coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			}

			updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, primeval_stones = $3, aptitude_percent = $4, lifespan = $5, path_masteries = $6, cleared_cells = $7, version = version + 1 WHERE id = $8`
			args = append(args, finalPlayerHP, finalMana, player.PrimevalStones, player.AptitudePercent, player.Lifespan, player.PathMasteries, player.ClearedCells, player.ID)
		}
	} else {
		if isVirtual {
			updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
			args = append(args, player.ID)
			if err := InventorySvc.AddItem(tx, player.ID.String(), "arena_coin", "materials", 10); err != nil {
				logger.S.Errorf("Failed to award arena_coin to %s: %v", player.ID, err)
			} else {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "ARENA_REWARD",
					Detail: "Thua cuộc tại Võ Đài. Nhận an ủi 10 Xu Đấu Trường.",
				})
			}
			logs = append(logs, simulator.CombatLog{
				Turn:   simState.Turn,
				Action: "BATTLE_END",
				Detail: "Thua cuộc trong đấu trường ảnh! Bản thể không bị ảnh hưởng.",
			})
		} else {
			player.HP = 1.0
			defeatMsg := "Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết."
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
				dropSummary := applyDeathDrop(tx, &player, nil)
				if dropSummary != "" {
					defeatMsg = fmt.Sprintf("Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết nhưng rơi rớt mất: %s.", dropSummary)
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

			updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, current_region = $3, x = $4, y = $5, aptitude_percent = $6, lifespan = $7, primeval_stones = $8, path_masteries = $9, version = version + 1 WHERE id = $10`
			args = append(args, player.HP, finalMana, player.CurrentRegion, player.X, player.Y, player.AptitudePercent, player.Lifespan, player.PrimevalStones, player.PathMasteries, player.ID)
		}
	}

	// Fire ON_COMBAT_END on equipped ACTIVE Gus
	var combatEndMsg string
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				if msg, err := HandleGuEvent(&player, spec, gu, "ON_COMBAT_END"); err == nil && msg != "" {
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
		_, errAp := tx.Exec(`
			UPDATE immortal_apertures SET aperture_hp = $1, updated_at = NOW()
			WHERE id = (SELECT aperture_id FROM tribulation_events WHERE id = $2 AND player_id = $3)
		`, simState.ApertureHP, player.ActiveCombat.TribulationEventID, player.ID)
		if errAp != nil {
			logger.S.Errorf("Failed to persist aperture HP after tribulation combat: %v", errAp)
		}

		// Sync enemy HP using LEAST so watchtower/worker damage is never overwritten upward.
		finalByID := make(map[string]int)
		for _, e := range simState.Enemies {
			finalByID[e.ID] = e.HP
		}
		for _, en := range player.ActiveCombat.Enemies {
			finalHP, ok := finalByID[en.ID]
			if !ok {
				continue
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

	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat outcome"})
		return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat outcome"})
			return
		}
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 3, $1) WHERE id = $2", MaxHumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit combat transaction"})
		return
	}

	setWorldPosFromPlayer(&player)

	winner := "Phe Địch"
	if alliesAlive {
		winner = "Phe Ta"
	}

	// ── Save combat history (non-critical) ──
	enemyName := ""
	enemyType := ""
	var targetPlayerID string
	if player.ActiveCombat != nil {
		enemyName = player.ActiveCombat.EnemyName
		enemyType = player.ActiveCombat.EnemyType
		targetPlayerID = player.ActiveCombat.TargetPlayerID
	}
	go func() {
		summary := buildCombatSummaryFromLogs(logs, finalPlayerHP, itemsUsed, stonesSpent)
		combatantInfosIface := make([]interface{}, len(combatantInfos))
		for i, ci := range combatantInfos {
			combatantInfosIface[i] = ci
		}
		SaveCombatHistory(player.ID, winner, toInterfaceSlice(logs), summary, combatantInfosIface, enemyName, enemyType, player.CurrentRegion)

		// Also save history for PvP target player
		if enemyType == "PVP" && targetPlayerID != "" && targetPlayerID != "dummy" {
			pvpWinner := "Phe Ta"
			if !alliesAlive {
				pvpWinner = "Phe Địch"
			}
			SaveCombatHistory(uuid.MustParse(targetPlayerID), pvpWinner, toInterfaceSlice(logs), summary, combatantInfosIface, player.Username, "PVP", player.CurrentRegion)
		}
	}()
	c.JSON(http.StatusOK, gin.H{
		"message":         "Chiến đấu hoàn tất. Nhân Khí +3.",
		"logs":            logs,
		"winner":          winner,
		"combatants":      combatantInfos,
		"aperture_hp":     simState.ApertureHP,
		"aperture_max_hp": simState.ApertureMaxHP,
	})
}

// buildSimCombatant constructs a simulator.CombatantAuto from player + Gu loadout.
// Shared by AutoCombatStart (combat_auto.go) and AutoResolveCombat (combat_auto_resolve.go).
func buildSimCombatant(tx *sql.Tx, player *models.Player, isVirtual bool, isInitiator bool, reqLoadout *simulator.CombatLoadout) (*simulator.CombatantAuto, error) {
	playerStats := CalculatePlayerStats(player)
	actor := models.CombatantFromPlayer(player, playerStats)

	var allActions []simulator.SimAction

	// Build owned equipped+ACTIVE Gu sets for validation
	ownedGuTypeSet := make(map[string]bool)                  // gu_id → true
	ownedGuInstanceSet := make(map[string]bool)              // instance_id → true
	equippedInstByGuID := make(map[string]models.GuInstance) // gu_id → highest-rank equipped ACTIVE instance
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			ownedGuTypeSet[gu.GuID] = true
			ownedGuInstanceSet[gu.InstanceID] = true
			if best, ok := equippedInstByGuID[gu.GuID]; !ok || gu.Rank > best.Rank {
				equippedInstByGuID[gu.GuID] = gu
			}
		}
	}

	// Resolve a loadout GuID to its actual equipped instance spec (match ID + rank)
	loadoutSpec := func(guID string) (models.GuDictionaryItem, bool) {
		if inst, ok := equippedInstByGuID[guID]; ok {
			return models.GetGu(guID, inst.Rank)
		}
		return models.GuDictionaryItem{}, false
	}

	var selectedGuIDs []string
	var selectedKMIDs []string
	var autoHealHP bool
	var healThreshold float64
	var autoRestoreMana bool
	var manaThreshold float64

	if isInitiator && reqLoadout != nil {
		selectedGuIDs = reqLoadout.GuIds
		selectedKMIDs = reqLoadout.KillerMoveIds
		autoHealHP = reqLoadout.AutoHealHP
		healThreshold = reqLoadout.HealThreshold
		autoRestoreMana = reqLoadout.AutoRestoreMana
		manaThreshold = reqLoadout.ManaThreshold
	} else {
		// Non-initiator: use the player's saved combat loadout (gu_priorities)
		// when configured, otherwise fall back to all equipped active Gus.
		settings := player.CombatSettings
		if len(settings.GuPriorities) > 0 {
			for _, t := range settings.GuPriorities {
				if ownedGuTypeSet[t.GuID] {
					selectedGuIDs = append(selectedGuIDs, t.GuID)
				}
			}
		}
		if len(selectedGuIDs) == 0 {
			for guID := range equippedInstByGuID {
				selectedGuIDs = append(selectedGuIDs, guID)
			}
		}

		// Automatically select all active killer moves they have
		kmMap, slotsMap, errLoadBatch := loadKillerMovesBatch(tx, player.ID, nil)
		if errLoadBatch == nil {
			for kmID, km := range kmMap {
				slots := slotsMap[kmID]
				// Check if component Gus are active
				if ownedGuInstanceSet[km.CorePlayerGuID] {
					allSlotsValid := true
					for _, slot := range slots {
						if !ownedGuInstanceSet[slot.PlayerGuID] {
							allSlotsValid = false
							break
						}
					}
					if allSlotsValid {
						selectedKMIDs = append(selectedKMIDs, kmID)
					}
				}
			}
		}

		// Use saved settings
		autoHealHP = player.CombatSettings.AutoHealHP
		healThreshold = player.CombatSettings.HealThreshold
		autoRestoreMana = player.CombatSettings.AutoRestoreMana
		manaThreshold = player.CombatSettings.ManaThreshold
	}

	// Add selected Gus
	for _, guID := range selectedGuIDs {
		if !ownedGuTypeSet[guID] {
			continue
		}
		if spec, ok := loadoutSpec(guID); ok {
			if hasCombatActiveEffects(&spec) {
				act, err := buildSimActionFromGu(&actor, &spec, "GU_SKILL")
				if err != nil {
					continue
				}
				act.SourceID = guID
				allActions = append(allActions, act)
			}
		}
	}

	// Add killer moves
	if len(selectedKMIDs) > 0 {
		kmMap, slotsMap, errLoadBatch := loadKillerMovesBatch(tx, player.ID, selectedKMIDs)
		if errLoadBatch == nil {
			for _, kmID := range selectedKMIDs {
				km, ok := kmMap[kmID]
				if !ok {
					continue
				}
				slots := slotsMap[kmID]
				act := buildSimActionFromKillerMove(km, slots, player.DaoMarks)
				act.SourceID = kmID
				allActions = append(allActions, act)
			}
		}
	}

	// Basic attack fallback
	allActions = append(allActions, simulator.SimAction{
		ActionType: "BASIC_ATTACK",
		Name:       "Đánh thường",
		Damage:     playerStats["attack"],
		ManaCost:   0,
	})

	// Global damage multipliers from loadout Gus. Apply each GuID at most once,
	// using the highest-rank equipped instance for that GuID.
	globalMultiplier := computeGlobalGuDamageMultiplier(selectedGuIDs, loadoutSpec)
	if globalMultiplier > 1.0 {
		for i := range allActions {
			allActions[i].Damage *= globalMultiplier
		}
	}

	// Apply persistent tactics (priority + conditions) to actions
	settings := player.CombatSettings
	for i := range allActions {
		act := &allActions[i]
		switch act.ActionType {
		case "GU_SKILL":
			for _, t := range settings.GuPriorities {
				if t.GuID == act.SourceID {
					act.Priority = t.Priority
					act.UseCondition = t.UseCondition
					act.ConditionValue = t.ConditionValue
					break
				}
			}
		case "KILLER_MOVE":
			for _, t := range settings.KMPriorities {
				if t.KillerMoveID == act.SourceID {
					act.Priority = t.Priority
					act.UseCondition = t.UseCondition
					act.ConditionValue = t.ConditionValue
					break
				}
			}
		}
		if act.ActionType == "BASIC_ATTACK" {
			act.Priority = 999
		}
	}

	// Virtual Shield from loadout
	virtualShield := 0
	scanShield := func(spec models.GuDictionaryItem) {
		if spec.Effects.Active != nil && spec.Effects.Active.ShieldFlat > 0 {
			virtualShield += int(spec.Effects.Active.ShieldFlat)
		}
	}
	for _, guID := range selectedGuIDs {
		if spec, ok := loadoutSpec(guID); ok {
			scanShield(spec)
		}
	}

	playerDaoMarks := make(simulator.DaoMarks)
	for k, v := range player.DaoMarks {
		playerDaoMarks[k] = v
	}

	charFX := gamefx.Get(player)

	// Scan inventory for healing items
	healPotency := 0.0
	healCharges := 0
	invItems, _ := InventorySvc.ListByPlayer(player.ID.String())
	for _, inv := range invItems {
		if inv.Quantity <= 0 {
			continue
		}
		item, ok := models.ItemRegistry[inv.ItemID]
		if !ok || item.Type != "Consumable" {
			continue
		}
		if healVal, ok := item.Effects["restore_hp"]; ok && healVal > 0 {
			if healPotency == 0 {
				healPotency = healVal
			}
			healCharges += int(inv.Quantity)
		}
	}

	baseHP := player.BaseHP + playerStats["hp"]
	simMaxHP := int(baseHP)
	if simMaxHP <= 0 {
		simMaxHP = 100
	}
	simHP := simMaxHP
	if !isVirtual {
		simHP = int(player.HP)
	}
	if simHP > simMaxHP {
		simHP = simMaxHP
	}
	if simHP <= 0 {
		simHP = simMaxHP
	}

	simMana := int(player.MaxMana)
	if !isVirtual {
		simMana = int(player.CurrentMana)
	}

	simStones := player.PrimevalStones
	simImmortalStones := player.ImmortalStones
	if isVirtual {
		simStones = 999999
		simImmortalStones = 999999
	}

	if player.PathMasteries == nil {
		player.PathMasteries = make(models.PathMasteriesMap)
	}

	simCombatant := &simulator.CombatantAuto{
		ID:                   player.ID.String(),
		Name:                 player.Username,
		Kind:                 "Player",
		Rank:                 player.Rank,
		SubRank:              player.SubRank,
		GridX:                2,
		GridY:                3,
		MaxHP:                simMaxHP,
		HP:                   simHP,
		MaxMana:              int(player.MaxMana),
		Mana:                 simMana,
		BaseDmg:              int(playerStats["attack"]),
		BaseDef:              int(playerStats["defense"]),
		Speed:                playerStats["speed"],
		DaoMarks:             simulator.DaoMarks(player.DaoMarks),
		PathMasteries:        player.PathMasteries,
		PathMasteryGainLimit: gamefx.PathMasteryBattleGainLimit(player.Rank),
		DaoBonus:             charFX.DaoNganBonus,
		PathDamageBonus:      charFX.PathDamageBonus,
		PathResistance:       charFX.PathResistanceBonus,
		HealingReceived:      charFX.HealingReceivedBonus,
		MentalResistance:     charFX.MentalResistanceBonus,
		Actions:            allActions,
		CategoryPriorities: player.CombatSettings.CategoryPriorities,

		AutoHealHP:      autoHealHP,
		HealThreshold:   healThreshold,
		AutoRestoreMana: isVirtual || autoRestoreMana,
		ManaThreshold:   manaThreshold,
		HealCharges:     healCharges,
		HealPotency:     healPotency,
		PrimevalStones:  simStones,
		ImmortalStones:  simImmortalStones,
		RefineManaTier:  gamefx.RefineManaTier(player.Rank, player.GuInventory),
	}

	if isInitiator && reqLoadout != nil {
		simCombatant.Loadout = reqLoadout
	} else {
		simCombatant.Loadout = &simulator.CombatLoadout{
			GuIds:           selectedGuIDs,
			KillerMoveIds:   selectedKMIDs,
			AutoHealHP:      autoHealHP,
			HealThreshold:   healThreshold,
			AutoRestoreMana: autoRestoreMana,
			ManaThreshold:   manaThreshold,
		}
	}

	return simCombatant, nil
}
