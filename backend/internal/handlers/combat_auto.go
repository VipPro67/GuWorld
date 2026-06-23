package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

// AutoCombatStartRequest payload for starting automated combat
type AutoCombatStartRequest struct {
	Loadout simulator.CombatLoadout `json:"loadout" binding:"required"`
}

// CombatantInfo returned to frontend for visual replay
type CombatantInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	MaxHP   int    `json:"max_hp"`
	HP      int    `json:"hp"`
	MaxMana int    `json:"max_mana"`
	Mana    int    `json:"mana"`
	GridRow int    `json:"grid_row"`
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

	seed := time.Now().UnixNano()
	simState := &simulator.CombatState{
		Turn:      0,
		Seed:      seed,
		BossPhase: 0,
	}

	// ── Pre-compute SimActions using real combat formulas ──

	playerStats := CalculatePlayerStats(&player)
	actor := models.CombatantFromPlayer(&player, playerStats)

	var allActions []simulator.SimAction

	// Core Gu
	if coreSpec, ok := models.GuRegistry[req.Loadout.CoreGuId]; ok {
		if hasCombatActiveEffects(&coreSpec) {
			act := buildSimActionFromGu(&actor, &coreSpec, "GU_SKILL")
			act.SourceID = req.Loadout.CoreGuId
			allActions = append(allActions, act)
		}
	}

	// Support Gu
	for _, supID := range req.Loadout.SupportGuIds {
		if supSpec, ok := models.GuRegistry[supID]; ok {
			if hasCombatActiveEffects(&supSpec) {
				act := buildSimActionFromGu(&actor, &supSpec, "GU_SKILL")
				act.SourceID = supID
				allActions = append(allActions, act)
			}
		}
	}

	// Killer moves
	for _, kmID := range req.Loadout.KillerMoveIds {
		km, slots, err := loadKillerMoveTx(tx, player.ID, kmID)
		if err != nil {
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
		activeEff := models.GetGuEffectSection(spec, "active")
		if v, ok := activeEff["damage_multiplier"]; ok && v > 1.0 {
			globalMultiplier *= v
		}
	}
	if coreSpec, ok := models.GuRegistry[req.Loadout.CoreGuId]; ok {
		scanMultiplier(coreSpec)
	}
	for _, supID := range req.Loadout.SupportGuIds {
		if supSpec, ok := models.GuRegistry[supID]; ok {
			scanMultiplier(supSpec)
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
		for _, section := range []string{"active", "support", "passive", "house"} {
			eff := models.GetGuEffectSection(spec, section)
			for _, key := range []string{"virtual_shield", "orbit_shield", "team_shield"} {
				if val, exists := eff[key]; exists {
					virtualShield += int(val)
				}
			}
		}
	}
	if coreSpec, ok := models.GuRegistry[req.Loadout.CoreGuId]; ok {
		scanShield(coreSpec)
	}
	for _, supID := range req.Loadout.SupportGuIds {
		if supSpec, ok := models.GuRegistry[supID]; ok {
			scanShield(supSpec)
		}
	}

	playerDaoMarks := make(simulator.DaoMarks)
	for k, v := range player.DaoMarks {
		playerDaoMarks[k] = v
	}

	// ── Scan inventory for healing items ──
	healPotency := 0.0
	healCharges := 0
	for itemID, qty := range player.MaterialsInventory {
		if qty <= 0 {
			continue
		}
		item, ok := models.ItemRegistry[itemID]
		if !ok || item.Type != "Consumable" {
			continue
		}
		if healVal, ok := item.Effects["restore_hp"]; ok && healVal > 0 {
			if healPotency == 0 {
				healPotency = healVal
			}
			healCharges += qty
		}
	}

	simPlayer := &simulator.CombatantAuto{
		ID:            player.ID.String(),
		Name:          player.Username,
		Kind:          "Player",
		GridRow:       2,
		MaxHP:         100,
		HP:            int(player.HP),
		MaxMana:       int(player.MaxMana),
		Mana:          int(player.CurrentMana),
		BaseDmg:       int(playerStats["attack"]),
		BaseDef:       int(playerStats["defense"]),
		VirtualShield: virtualShield,
		DaoMarks:      playerDaoMarks,
		Loadout:       &req.Loadout,
		Actions:       allActions,

		AutoHealHP:      req.Loadout.AutoHealHP,
		HealThreshold:   req.Loadout.HealThreshold,
		AutoRestoreMana: req.Loadout.AutoRestoreMana,
		ManaThreshold:   req.Loadout.ManaThreshold,
		HealCharges:     healCharges,
		HealPotency:     healPotency,
		PrimevalStones:  player.PrimevalStones,
		ImmortalStones:  player.ImmortalStones,
	}
	if simPlayer.HP <= 0 || simPlayer.HP > 100 {
		simPlayer.HP = 100
	}

	simState.Allies = append(simState.Allies, simPlayer)

	// ── Build enemies ──
	for i, enemy := range player.ActiveCombat.Enemies {
		eHP := int(enemy.HP)
		if eHP <= 0 {
			eHP = int(enemy.MaxHP)
		}
		simEnemy := &simulator.CombatantAuto{
			ID:      enemy.ID,
			Name:    enemy.Name,
			Kind:    "Enemy",
			GridRow: 1,
			MaxHP:   int(enemy.MaxHP),
			HP:      eHP,
			MaxMana: int(enemy.MaxMana),
			Mana:    int(enemy.CurrentMana),
			BaseDmg: int(enemy.BaseAttack),
			BaseDef: int(enemy.BaseDefense),
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
		if i == 0 {
			simEnemy.GridRow = 1
		}
		// Give enemies a basic attack action
		simEnemy.Actions = []simulator.SimAction{
			{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: float64(simEnemy.BaseDmg), ManaCost: 0},
		}
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
		GridRow: simPlayer.GridRow,
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
			GridRow: a.GridX,
		})
	}
	for _, e := range player.ActiveCombat.Enemies {
		combatantInfos = append(combatantInfos, CombatantInfo{
			ID:      e.ID,
			Name:    e.Name,
			Kind:    "enemy",
			MaxHP:   int(e.MaxHP),
			HP:      int(e.HP),
			MaxMana: int(e.MaxMana),
			Mana:    int(e.CurrentMana),
			GridRow: e.GridX,
		})
	}

	// 4. Run Simulator
	logs := simulator.Simulate(simState)

	// 5. Apply outcome to DB
	var alliesAlive bool
	var finalPlayerHP int
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			alliesAlive = (a.HP > 0)
			finalPlayerHP = a.HP
		}
	}

	// ── Deduct items used by simulator ──
	itemsUsed := simPlayer.ItemsUsedInBattle
	if itemsUsed > 0 {
		remaining := itemsUsed
		for itemID, qty := range player.MaterialsInventory {
			if remaining <= 0 {
				break
			}
			if qty <= 0 {
				continue
			}
			item, ok := models.ItemRegistry[itemID]
			if !ok || item.Type != "Consumable" {
				continue
			}
			if _, hasHeal := item.Effects["restore_hp"]; hasHeal {
				remove := qty
				if remove > remaining {
					remove = remaining
				}
				player.MaterialsInventory[itemID] -= remove
				if player.MaterialsInventory[itemID] <= 0 {
					delete(player.MaterialsInventory, itemID)
				}
				remaining -= remove
			}
		}
	}

	// ── Deduct stones used by simulator ──
	stonesSpent := player.PrimevalStones - simPlayer.PrimevalStones
	immortalStonesSpent := player.ImmortalStones - simPlayer.ImmortalStones
	player.PrimevalStones = simPlayer.PrimevalStones
	player.ImmortalStones = simPlayer.ImmortalStones

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
			Detail: fmt.Sprintf("Đã tự động dùng %.0f Tiên Nguyên Thạch hồi phục tiên nguyên.", immortalStonesSpent),
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

	var lootMsg string
	if alliesAlive {
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
						newGu := models.GuInstance{
							InstanceID: uuid.New().String(), GuID: guID, Hunger: 0, Status: "ACTIVE", Equipped: false, IsVital: false,
						}
						player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, newGu)
					}
				} else {
					player.PrimevalStones += stoneQty
					for matID, qty := range loot.Items {
						if matID == "primeval_stone" {
							continue
						}
						if player.MaterialsInventory == nil {
							player.MaterialsInventory = make(models.MaterialsInventory)
						}
						player.MaterialsInventory[matID] += qty
					}
					for _, guID := range loot.Gus {
						newGu := models.GuInstance{
							InstanceID: uuid.New().String(), GuID: guID, Hunger: 0, Status: "ACTIVE", Equipped: false, IsVital: false,
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

		tempLootJSON, _ := json.Marshal(player.TempLoot)
		matJSON, _ := json.Marshal(player.MaterialsInventory)
		guJSON, _ := json.Marshal(player.GuInventory)

		updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, temp_loot = $3, primeval_stones = $4, materials_inventory = $5, gu_inventory = $6 WHERE id = $7`
		args = append(args, finalPlayerHP, finalMana, tempLootJSON, player.PrimevalStones, matJSON, guJSON, player.ID)
	} else {
		// ── Defeat: same logic as manual combat ──
		player.HP = 20.0
		player.CurrentRegion = "Southern_Border"
		player.X = 50
		player.Y = 50
		player.ExploredCells = models.ExploredCellsList{"50,50"}
		player.TempLoot = models.TempLoot{
			GuInventory:        models.GuInventory{},
			MaterialsInventory: models.MaterialsInventory{},
			PrimevalStones:     0,
		}

		// Clear allies' active combat
		for _, ally := range player.ActiveCombat.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil && allyUUID != player.ID {
				_, _ = tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", allyUUID)
			}
		}

		logs = append(logs, simulator.CombatLog{
			Turn:   simState.Turn,
			Action: "BATTLE_END",
			Detail: "Đại bại! Đạo hữu bất tỉnh nhân sự. May mắn thoát chết nhưng rơi rớt mất sạch chiến lợi phẩm và bị đưa về Nam Cương.",
		})

		tempLootJSON, _ := json.Marshal(player.TempLoot)
		exploredJSON, _ := json.Marshal(player.ExploredCells)
		updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, temp_loot = $3, current_region = $4, x = $5, y = $6, explored_cells = $7 WHERE id = $8`
		args = append(args, player.HP, finalMana, tempLootJSON, player.CurrentRegion, player.X, player.Y, exploredJSON, player.ID)
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
	if player.ActiveCombat != nil {
		enemyName = player.ActiveCombat.EnemyName
		enemyType = player.ActiveCombat.EnemyType
	}
	go func() {
		summary := buildCombatSummaryFromLogs(logs, finalPlayerHP, itemsUsed, stonesSpent)
		combatantInfosIface := make([]interface{}, len(combatantInfos))
		for i, ci := range combatantInfos {
			combatantInfosIface[i] = ci
		}
		SaveCombatHistory(player.ID, winner, toInterfaceSlice(logs), summary, combatantInfosIface, enemyName, enemyType, player.CurrentRegion)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":    "Chiến đấu hoàn tất",
		"logs":       logs,
		"winner":     winner,
		"combatants": combatantInfos,
	})
}

// ── helpers ────────────────────────────────────────────────────────────

func buildSimActionFromGu(actor *models.Combatant, spec *models.GuDictionaryItem, actionType string) simulator.SimAction {
	dmg, def, cost, err := CalculateGuEffectForCombatant(actor, spec)
	if err != nil {
		return simulator.SimAction{
			ActionType: actionType,
			Name:       spec.Name,
			ManaCost:   cost,
		}
	}

	active := models.GetGuEffectSection(*spec, "active")
	support := models.GetGuEffectSection(*spec, "support")

	act := simulator.SimAction{
		ActionType: actionType,
		Name:       spec.Name,
		Damage:     dmg,
		Defense:    def,
		ManaCost:   cost,
	}

	for _, m := range []map[string]float64{active, support} {
		if v, ok := m["bleed"]; ok {
			act.BleedPerTurn += v
		}

		if v, ok := m["poison"]; ok {
			act.PoisonPerTurn += v
		}
		if v, ok := m["stun"]; ok {
			act.StunTurns = int(v)
		}
		if v, ok := m["life_steal"]; ok {
			act.LifeStealPct += v / 100.0
		}
		if v, ok := m["blood_drain"]; ok {
			act.LifeStealPct += v / 100.0
		}
		if v, ok := m["heal"]; ok {
			act.Healing += v
		}
	}

	return act
}

// hasCombatActiveEffects returns true if the Gu's active effects produce a meaningful combat action.
func hasCombatActiveEffects(spec *models.GuDictionaryItem) bool {
	active := models.GetGuEffectSection(*spec, "active")
	if len(active) == 0 {
		return false
	}
	combatKeys := []string{
		"damage", "add_base_attack", "base_damage_bonus",
		"defense", "add_base_defense", "base_defense_bonus",
		"shield", "virtual_shield",
		"bleed", "poison", "stun",
		"life_steal", "blood_drain", "heal",
	}
	for _, k := range combatKeys {
		if _, ok := active[k]; ok {
			return true
		}
	}
	return false
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

	return act
}
