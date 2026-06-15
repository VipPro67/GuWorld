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

	// 1. Fetch Player
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong trạng thái chiến đấu."})
		return
	}

	// 2. Set up deterministic seed
	seed := time.Now().UnixNano() // For a real deterministic replay, this seed could be saved in DB
	// For PVP, we could hash the two player IDs and a daily salt, etc.

	// 3. Build Simulator State
	simState := &simulator.CombatState{
		Turn:      0,
		Seed:      seed,
		BossPhase: 0,
	}

	// Calculate base stats based on existing models.Player logic
	playerStats := CalculatePlayerStats(&player)
	
	// Add stats from Loadout (Core Gu and Support Gu)
	if coreGuData, ok := models.GuRegistry[req.Loadout.CoreGuId]; ok {
		playerStats["attack"] += coreGuData.Effects["active"]["damage"] + coreGuData.Effects["active"]["add_base_attack"]
		playerStats["defense"] += coreGuData.Effects["active"]["defense"] + coreGuData.Effects["active"]["add_base_defense"]
	}
	for _, supId := range req.Loadout.SupportGuIds {
		if supData, ok := models.GuRegistry[supId]; ok {
			playerStats["attack"] += supData.Effects["active"]["damage"] + supData.Effects["active"]["add_base_attack"]
			playerStats["defense"] += supData.Effects["active"]["defense"] + supData.Effects["active"]["add_base_defense"]
		}
	}

	playerDaoMarks := make(simulator.DaoMarks)
	for k, v := range player.DaoMarks {
		playerDaoMarks[k] = v
	}

	// Determine pre-combat virtual shield from loadout
	// Very simple heuristic for demonstration based on the new logic
	virtualShield := 0
	if req.Loadout.CoreGuId == "thuy_trao_co" || req.Loadout.CoreGuId == "kim_giap_co" {
		virtualShield = 100000 // A huge shield as described
	}

	simPlayer := &simulator.CombatantAuto{
		ID:            player.ID.String(),
		Name:          player.Username,
		Kind:          "Player",
		GridRow:       2, // Default Backline
		MaxHP:         int(player.BaseHP),
		HP:            int(player.BaseHP),
		MaxMana:       int(player.MaxMana),
		Mana:          int(player.CurrentMana),
		BaseDmg:       int(playerStats["attack"]),
		BaseDef:       int(playerStats["defense"]),
		VirtualShield: virtualShield,
		DaoMarks:      playerDaoMarks,
		Loadout:       &req.Loadout,
		ActiveEffects: []simulator.Effect{},
	}
	// Normal players have low base HP
	if simPlayer.HP <= 0 {
		simPlayer.HP = 100
		simPlayer.MaxHP = 100
	}

	simState.Allies = append(simState.Allies, simPlayer)

	// Fetch enemies from player.ActiveCombat
	for i, enemy := range player.ActiveCombat.Enemies {
		eHP := int(enemy.HP)
		if eHP <= 0 {
			eHP = int(enemy.MaxHP)
		}
		simEnemy := &simulator.CombatantAuto{
			ID:            enemy.ID,
			Name:          enemy.Name,
			Kind:          "Enemy",
			GridRow:       1, // Default Frontline
			MaxHP:         int(enemy.MaxHP),
			HP:            eHP,
			MaxMana:       int(enemy.MaxMana),
			Mana:          int(enemy.CurrentMana),
			BaseDmg:       int(enemy.BaseAttack),
			BaseDef:       int(enemy.BaseDefense),
			VirtualShield: 0, 
			DaoMarks:      make(simulator.DaoMarks),
			ActiveEffects: []simulator.Effect{},
		}
		if enemy.Kind == "Boss" {
			simEnemy.Kind = "Boss"
			simEnemy.VirtualShield = 0
		}
		if i == 0 {
			simEnemy.GridRow = 1
		}
		simState.Enemies = append(simState.Enemies, simEnemy)
	}

	// 4. Run Simulator
	logs := simulator.Simulate(simState)

	// 5. Apply outcome to Database (Stateless transaction)
	// Deduct HP, finalize battle, give rewards, etc.
	var alliesAlive bool
	var finalPlayerHP int
	for _, a := range simState.Allies {
		if a.ID == player.ID.String() {
			alliesAlive = (a.HP > 0)
			finalPlayerHP = a.HP
		}
	}

	var updateQuery string
	var args []interface{}

	var lootMsg string
	if alliesAlive {
		// Player won
		isExplore := player.CurrentRegion != "Southern_Border" && player.CurrentRegion != "Lobby"
		if player.ActiveCombat != nil && player.ActiveCombat.LootTableID != "" {
			lt, err := db.LoadLootTable(tx, player.ActiveCombat.LootTableID)
			if err == nil {
				loot := RollLoot(lt)
				
				// Apply loot
				if player.ActiveCombat.EnemyGuInventory != nil && len(player.ActiveCombat.EnemyGuInventory) > 0 {
					for _, eg := range player.ActiveCombat.EnemyGuInventory {
						if rand.Intn(100) < 10 { // 10% chance
							loot.Gus = append(loot.Gus, eg.GuID)
						}
					}
				}

				stoneQty := loot.Items["primeval_stone"]
				if isExplore {
					player.TempLoot.PrimevalStones += stoneQty
					for matID, qty := range loot.Items {
						if matID == "primeval_stone" { continue }
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
						if matID == "primeval_stone" { continue }
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

		updateQuery = `UPDATE players SET hp = $1, active_combat = NULL, temp_loot = $2, primeval_stones = $3, materials_inventory = $4, gu_inventory = $5 WHERE id = $6`
		args = append(args, finalPlayerHP, tempLootJSON, player.PrimevalStones, matJSON, guJSON, player.ID)
	} else {
		// Player lost
		updateQuery = `UPDATE players SET hp = $1, active_combat = NULL WHERE id = $2`
		args = append(args, 0, player.ID)
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

	// 6. Return JSON log to client
	c.JSON(http.StatusOK, gin.H{
		"message": "Chiến đấu hoàn tất",
		"logs":    logs,
		"winner":  winner,
	})
}
