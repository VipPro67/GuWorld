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
		act := buildSimActionFromGu(&actor, &coreSpec, "GU_SKILL")
		allActions = append(allActions, act)
	}

	// Support Gu
	for _, supID := range req.Loadout.SupportGuIds {
		if supSpec, ok := models.GuRegistry[supID]; ok {
			act := buildSimActionFromGu(&actor, &supSpec, "GU_SKILL")
			allActions = append(allActions, act)
		}
	}

	// Killer moves
	for _, kmID := range req.Loadout.KillerMoveIds {
		km, slots, err := loadKillerMoveTx(tx, player.ID, kmID)
		if err != nil {
			continue
		}
		act := buildSimActionFromKillerMove(km, slots)
		allActions = append(allActions, act)
	}

	// Basic attack fallback
	allActions = append(allActions, simulator.SimAction{
		ActionType: "BASIC_ATTACK",
		Name:       "Đánh thường",
		Damage:     playerStats["attack"],
		ManaCost:   0,
	})

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

	// 4. Run Simulator
	logs := simulator.Simulate(simState)

	// 5. Apply outcome to DB (unchanged)
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

		updateQuery = `UPDATE players SET hp = $1, active_combat = NULL, temp_loot = $2, primeval_stones = $3, materials_inventory = $4, gu_inventory = $5 WHERE id = $6`
		args = append(args, finalPlayerHP, tempLootJSON, player.PrimevalStones, matJSON, guJSON, player.ID)
	} else {
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Chiến đấu hoàn tất",
		"logs":    logs,
		"winner":  winner,
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
		if v, ok := m["bleed_effect"]; ok {
			act.BleedPerTurn += v
		}
		if v, ok := m["poison"]; ok {
			act.PoisonPerTurn += v
		}
		if v, ok := m["stun"]; ok {
			act.StunTurns = int(v)
		}
		if v, ok := m["lifesteal"]; ok {
			act.LifeStealPct += v / 100.0
		}
		if v, ok := m["blood_drain"]; ok {
			act.LifeStealPct += v / 100.0
		}
		if v, ok := m["healing"]; ok {
			act.Healing += v
		}
	}

	return act
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
		if v, ok := effects["bleed_effect"]; ok {
			act.BleedPerTurn += v
		}
		if v, ok := effects["poison"]; ok {
			act.PoisonPerTurn += v
		}
		if v, ok := effects["stun"]; ok {
			act.StunTurns = int(v)
		}
		if v, ok := effects["lifesteal"]; ok {
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
		if v, ok := effects["healing"]; ok {
			act.Healing += v
		}
	}

	return act
}
