package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// AutoCombatStartRequest payload for starting automated combat
type AutoCombatStartRequest struct {
	Loadout simulator.CombatLoadout `json:"loadout" binding:"required"`
}

// CombatantInfo returned to frontend for visual replay
type CombatantInfo struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Kind          string            `json:"kind"`
	MaxHP         int               `json:"max_hp"`
	HP            int               `json:"hp"`
	MaxMana       int               `json:"max_mana"`
	Mana          int               `json:"mana"`
	GridX         int               `json:"grid_x"`
	GridY         int               `json:"grid_y"`
	Shield        int               `json:"shield"`
	Damage        float64           `json:"damage,omitempty"`
	Defense       float64           `json:"defense,omitempty"`
	Speed         float64           `json:"speed,omitempty"`
	KindDetail    string            `json:"kind_detail,omitempty"`
	Rank          int               `json:"rank,omitempty"`
	SubRank       int               `json:"sub_rank,omitempty"`
	DaoMarks      map[string]int    `json:"dao_marks,omitempty"`
	PathMasteries map[string]string `json:"path_masteries,omitempty"`
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

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong trạng thái chiến đấu."})
		return
	}

	isVirtual := player.ActiveCombat.CombatMode == "virtual"

	seed := time.Now().UnixNano()
	simState := &simulator.CombatState{
		Turn:        0,
		Seed:        seed,
		BossPhase:   0,
		VirtualMode: isVirtual,
	}

	// ── Pre-compute SimActions using real combat formulas ──

	playerStats := CalculatePlayerStats(&player)
	actor := models.CombatantFromPlayer(&player, playerStats)

	var allActions []simulator.SimAction

	// ── Build owned equipped+ACTIVE Gu sets for validation ──
	ownedGuTypeSet := make(map[string]bool)       // gu_id → true
	ownedGuInstanceSet := make(map[string]bool)   // instance_id → true
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			ownedGuTypeSet[gu.GuID] = true
			ownedGuInstanceSet[gu.InstanceID] = true
		}
	}

	// All selected Gu (core + support, no distinction)
	for _, guID := range req.Loadout.GuIds {
		if !ownedGuTypeSet[guID] {
			continue
		}
		if spec, ok := models.GetGuAny(guID); ok {
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

	// ── Build active Gu set for killer move component validation ──
	activeGuSet := ownedGuInstanceSet

	// Killer moves
	for _, kmID := range req.Loadout.KillerMoveIds {
		km, slots, err := loadKillerMoveTx(tx, player.ID, kmID)
		if err != nil {
			continue
		}
		// Skip killer move if any component Gu is missing or inactive
		if !activeGuSet[km.CorePlayerGuID] {
			continue
		}
		allSlotsValid := true
		for _, slot := range slots {
			if !activeGuSet[slot.PlayerGuID] {
				allSlotsValid = false
				break
			}
		}
		if !allSlotsValid {
			continue
		}
		act := buildSimActionFromKillerMove(km, slots)
		act.SourceID = kmID
		allActions = append(allActions, act)
	}

	// Basic attack fallback
	allActions = append(allActions, simulator.SimAction{
		ActionType: "BASIC_ATTACK",
		Name:       "Đánh thường",
		Damage:     playerStats["attack"],
		ManaCost:   0,
	})

	// ── Apply global damage multipliers from loadout Gus (e.g. Thái Độ Cổ) ──
	globalMultiplier := 1.0
	scanMultiplier := func(spec models.GuDictionaryItem) {
		if spec.Effects.Active != nil && spec.Effects.Active.DamageMult > 1.0 {
			globalMultiplier *= spec.Effects.Active.DamageMult
		}
	}
	for _, guID := range req.Loadout.GuIds {
		if spec, ok := models.GetGuAny(guID); ok {
			scanMultiplier(spec)
		}
	}
	if globalMultiplier > 1.0 {
		for i := range allActions {
			allActions[i].Damage *= globalMultiplier
		}
	}

	// ── Apply persistent tactics (priority + conditions) to actions ──
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
		// BASIC_ATTACK always lowest priority
		if act.ActionType == "BASIC_ATTACK" {
			act.Priority = 999
		}
	}

	// ── Virtual Shield from loadout ──
	virtualShield := 0
	scanShield := func(spec models.GuDictionaryItem) {
		if spec.Effects.Active != nil && spec.Effects.Active.ShieldFlat > 0 {
			virtualShield += int(spec.Effects.Active.ShieldFlat)
		}
	}
	for _, guID := range req.Loadout.GuIds {
		if spec, ok := models.GetGuAny(guID); ok {
			scanShield(spec)
		}
	}

	playerDaoMarks := make(simulator.DaoMarks)
	for k, v := range player.DaoMarks {
		playerDaoMarks[k] = v
	}

	// ── Scan inventory for healing items ──
	healPotency := 0.0
	healCharges := 0
	invItems, _ := InventorySvc.ListByPlayer(authID.String())
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

	simPlayer := &simulator.CombatantAuto{
		ID:            player.ID.String(),
		Name:          player.Username,
		Kind:          "Player",
		GridX:         2,
		GridY:         2,
		MaxHP:         simMaxHP,
		HP:            simHP,
		MaxMana:       int(player.MaxMana),
		Mana:          simMana,
		BaseDmg:       int(playerStats["attack"]),
		BaseDef:       int(playerStats["defense"]),
		Speed:         playerStats["speed"],
		VirtualShield: virtualShield,
		DaoMarks:      playerDaoMarks,
		Loadout:       &req.Loadout,
		Actions:       allActions,

		AutoHealHP:      req.Loadout.AutoHealHP,
		HealThreshold:   req.Loadout.HealThreshold,
		AutoRestoreMana: isVirtual || req.Loadout.AutoRestoreMana,
		ManaThreshold:   req.Loadout.ManaThreshold,
		HealCharges:     healCharges,
		HealPotency:     healPotency,
		PrimevalStones:  simStones,
		ImmortalStones:  simImmortalStones,
		RefineManaTier:  getRefineManaTier(player.Rank, player.GuInventory),
	}

	simState.Allies = append(simState.Allies, simPlayer)

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
		simEnemy := &simulator.CombatantAuto{
			ID:      enemy.ID,
			Name:    enemy.Name,
			Kind:    "Enemy",
			GridX:   enemy.GridX,
			GridY:   enemy.GridY,
			MaxHP:   int(enemy.MaxHP),
			HP:      eHP,
			MaxMana: int(enemy.MaxMana),
			Mana:    enemyMana,
			BaseDmg: int(enemy.BaseAttack),
			BaseDef: int(enemy.BaseDefense),
			Speed:   enemy.Speed,
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
			spec, ok := models.GetGuAny(guInst.GuID)
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
	combatantInfos = append(combatantInfos, CombatantInfo{
		ID:      simPlayer.ID,
		Name:    simPlayer.Name,
		Kind:    "ally",
		MaxHP:   simPlayer.MaxHP,
		HP:      simPlayer.HP,
		MaxMana: simPlayer.MaxMana,
		Mana:    simPlayer.Mana,
		GridX:   simPlayer.GridX,
		GridY:   simPlayer.GridY,
		Shield:  virtualShield,
	})
	for _, a := range player.ActiveCombat.Allies {
		combatantInfos = append(combatantInfos, CombatantInfo{
			ID:      a.ID,
			Name:    a.Name,
			Kind:    "ally",
			MaxHP:   int(a.MaxHP),
			HP:      int(a.HP),
			MaxMana: int(a.MaxMana),
			Mana:    int(a.CurrentMana),
			GridX:   a.GridX,
			GridY:   a.GridY,
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
			} else if strings.Contains(logMsg.Detail, "Trừ 50% Nguyên thạch hiện tại") {
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
	var alliesAlive bool
	var finalPlayerHP int
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			alliesAlive = (a.HP > 0)
			finalPlayerHP = a.HP
		}
	}

	// ── Deduct items/stones used by simulator (skip for virtual mode) ──
	itemsUsed := simPlayer.ItemsUsedInBattle
	stonesSpent := 0
	immortalStonesSpent := 0

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
					if remove > remaining {
						remove = remaining
					}
					InventorySvc.RemoveItem(tx, authID.String(), inv.ItemID, "materials", remove)
					remaining -= remove
				}
			}
		}

		stonesSpent = player.PrimevalStones - simPlayer.PrimevalStones
		immortalStonesSpent = int(player.ImmortalStones - simPlayer.ImmortalStones)
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

	var lootMsg string
	if alliesAlive {
		if isVirtual {
			updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
			args = append(args, player.ID)
		} else {
			isExplore := player.CurrentRegion != "Southern_Border" && player.CurrentRegion != "Lobby"
			if player.ActiveCombat != nil && player.ActiveCombat.LootTableID != "" {
				lt, err := db.LoadLootTable(tx, player.ActiveCombat.LootTableID)
				if err == nil {
					loot := RollLoot(lt)

					if player.ActiveCombat.EnemyGuInventory != nil && len(player.ActiveCombat.EnemyGuInventory) > 0 {
						for _, eg := range player.ActiveCombat.EnemyGuInventory {
							if rand.Intn(100) < 10 {
								loot.Gus = append(loot.Gus, eg.GuID)
							}
						}
					}

					stoneQty := loot.Items["primeval_stone"]
					if isExplore {
						player.TempLoot.PrimevalStones += stoneQty
						for matID, qty := range loot.Items {
							if matID == "primeval_stone" {
								continue
							}
							if player.TempLoot.MaterialsInventory == nil {
								player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
							}
							player.TempLoot.MaterialsInventory[matID] += qty
						}
						for _, guID := range loot.Gus {
							guSpec, _ := models.GetGuAny(guID)
							newGu := models.GuInstance{
								InstanceID: uuid.New().String(), GuID: guID, Rank: guSpec.Rank, Hunger: 0, Status: "ACTIVE", Equipped: false, IsVital: false,
							}
							player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, newGu)
						}
					} else {
						player.PrimevalStones += stoneQty
						for matID, qty := range loot.Items {
							if matID == "primeval_stone" {
								continue
							}
							InventorySvc.AddItem(tx, player.ID.String(), matID, "materials", qty)
						}
						for _, guID := range loot.Gus {
							guSpec, _ := models.GetGuAny(guID)
							newGu := models.GuInstance{
								InstanceID: uuid.New().String(), GuID: guID, Rank: guSpec.Rank, Hunger: 0, Status: "ACTIVE", Equipped: false, IsVital: false,
							}
							player.GuInventory = append(player.GuInventory, newGu)
						}
					}

					if stoneQty > 0 || len(loot.Items) > 1 || len(loot.Gus) > 0 {
						lootMsg = fmt.Sprintf("Nhặt được: %d Nguyên Thạch, %d nguyên liệu, %d Cổ trùng.", stoneQty, len(loot.Items)-1, len(loot.Gus))
						if len(loot.Items) <= 1 && len(loot.Gus) == 0 {
							lootMsg = fmt.Sprintf("Nhặt được: %d Nguyên Thạch.", stoneQty)
						}
					}
				}
			}

			if lootMsg != "" {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "LOOT",
					Detail: lootMsg,
				})
			}

			// ── Mine combat victory: occupy mine + remove monster ──
			if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 {
				slotID := player.ActiveCombat.MineSlotID
				var rarity string
				var currentOccupierStr *string
				var monsterIDStr *string
				var lastClaimedAt *time.Time
				err = tx.QueryRow("SELECT rarity, occupier_id, monster_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE", slotID).Scan(&rarity, &currentOccupierStr, &monsterIDStr, &lastClaimedAt)
				if err == nil && currentOccupierStr != nil && lastClaimedAt != nil {
					stones, qi, chests := CalculateMineRewards(slotID, rarity, *lastClaimedAt)
					if stones > 0 || qi > 0.0 || chests > 0 {
						_ = AddMineRewardsToPlayerTx(tx, player.ID, stones/2, qi/2.0, chests/2)
						player.PrimevalStones += stones / 2
						if chests/2 > 0 {
							InventorySvc.AddItem(tx, player.ID.String(), "mine_chest", "materials", chests/2)
						}
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "MINE_LOOT",
							Detail: fmt.Sprintf("Cướp được %d Nguyên Thạch và %d rương từ đối thủ!", stones/2, chests/2),
						})
					}
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

			tempLootJSON, _ := json.Marshal(player.TempLoot)
			guJSON, _ := json.Marshal(player.GuInventory)

			updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, temp_loot = $3, primeval_stones = $4, gu_inventory = $5, aptitude_percent = $6, lifespan = $7 WHERE id = $8`
			args = append(args, finalPlayerHP, finalMana, tempLootJSON, player.PrimevalStones, guJSON, player.AptitudePercent, player.Lifespan, player.ID)
		}
	} else {
		if isVirtual {
			updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
			args = append(args, player.ID)
			logs = append(logs, simulator.CombatLog{
				Turn:   simState.Turn,
				Action: "BATTLE_END",
				Detail: "Thua cuộc trong đấu trường ảnh! Bản thể không bị ảnh hưởng.",
			})
		} else {
			player.HP = 1.0
			player.CurrentRegion = "Southern_Border"
			player.X = 50
			player.Y = 50
			player.ExploredCells = models.ExploredCellsList{"50,50"}
			player.TempLoot = models.TempLoot{
				GuInventory:        models.GuInventory{},
				MaterialsInventory: models.MaterialsInventory{},
				PrimevalStones:     0,
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
				Detail: "Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết nhưng rơi rớt mất sạch chiến lợi phẩm.",
			})

			tempLootJSON, _ := json.Marshal(player.TempLoot)
			exploredJSON, _ := json.Marshal(player.ExploredCells)
			updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, temp_loot = $3, current_region = $4, x = $5, y = $6, explored_cells = $7, aptitude_percent = $8, lifespan = $9, primeval_stones = $10 WHERE id = $11`
			args = append(args, player.HP, finalMana, tempLootJSON, player.CurrentRegion, player.X, player.Y, exploredJSON, player.AptitudePercent, player.Lifespan, player.PrimevalStones, player.ID)
		}
	}

	// Fire ON_COMBAT_END on equipped ACTIVE Gus
	var combatEndMsg string
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if spec, ok := models.GetGuAny(gu.GuID); ok {
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

	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat outcome"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit combat transaction"})
		return
	}

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

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "combat.finish",
		RequestSummary: fmt.Sprintf("winner=%s enemy=%s type=%s stonesSpent=%d itemsUsed=%d", winner, enemyName, enemyType, stonesSpent, itemsUsed),
		IPAddress:      c.ClientIP(), Success: alliesAlive,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":    "Chiến đấu hoàn tất",
		"logs":       logs,
		"winner":     winner,
		"combatants": combatantInfos,
	})
}

// ── helpers ────────────────────────────────────────────────────────────

func buildSimActionFromGu(actor *models.Combatant, spec *models.GuDictionaryItem, actionType string) (simulator.SimAction, error) {
	if actor.Rank >= 6 && actor.Rank < spec.Rank {
		return simulator.SimAction{}, fmt.Errorf("Tiên Khiếu không thể điều khiển Cổ trùng cấp cao hơn")
	}

	dmg, def, cost, err := CalculateGuEffectForCombatant(actor, spec)
	if err != nil {
		return simulator.SimAction{
			ActionType: actionType,
			Name:       spec.Name,
			ManaCost:   cost,
		}, nil
	}

	act := simulator.SimAction{
		ActionType: actionType,
		Name:       spec.Name,
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
		if active.ApplyStatus == "BLEED" {
			act.BleedPerTurn += active.StatusTickDamage
		}
		if active.ApplyStatus == "POISON" {
			act.PoisonPerTurn += active.StatusTickDamage
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

func buildSimActionFromKillerMove(km models.KillerMove, _ []models.KillerMoveSlot) simulator.SimAction {
	act := simulator.SimAction{
		ActionType: "KILLER_MOVE",
		Name:       km.Name,
		Damage:     km.DamageValue,
		Defense:    km.DefenseValue,
		ManaCost:   km.TotalManaCost,
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
			act.StunTurns = int(v)
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
	}

	act.TargetType = "SINGLE"
	return act
}
