package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// CombatAttackRequest defines payload to execute a combat action/killer move
type CombatAttackRequest struct {
	PlayerID      string   `json:"player_id" binding:"required"`
	TargetID      string   `json:"target_id"`
	EquippedGuIDs []string `json:"equipped_gu_ids"` // Instance IDs of Gus to execute
	IsCharge      bool     `json:"is_charge"`       // Whether player clicked the charge button
	TargetX       int      `json:"target_x"`
	TargetY       int      `json:"target_y"`
	ItemID        string   `json:"item_id"`    // Item used in combat
	UseStones     string   `json:"use_stones"` // 'primeval' or 'immortal'
}

// ExploreRequest defines payload for map node exploration
type ExploreRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	NodeID   string `json:"node_id" binding:"required"`
}

// CombatContext represents active battle parameters
type CombatContext struct {
	Attacker    *models.Combatant
	Defender    *models.Combatant
	DamageDealt float64
	ShieldAdded float64
	DamageBoost float64 // Multiplier from support Gus (e.g. 0.20 = +20%)
	Logs        []string
}

func getActorPathMasteryMultiplier(actor *models.Combatant, path string) float64 {
	masteryMultiplier := 1.0
	if actor.PathMasteries != nil {
		cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
		if mLevel, ok := actor.PathMasteries[cleanPath]; ok {
			switch mLevel {
			case "Ordinary":
				masteryMultiplier = 1.0
			case "Master":
				masteryMultiplier = 1.1
			case "Grandmaster":
				masteryMultiplier = 1.25
			case "Great Grandmaster":
				masteryMultiplier = 1.4
			case "Supreme Grandmaster":
				masteryMultiplier = 1.5
			}
		}
	}
	return masteryMultiplier
}

// CalculateGuEffectForCombatant computes lore-accurate adjustments for level penalty, regional suppression, and mana cost for any combatant
func CalculateGuEffectForCombatant(actor *models.Combatant, guSpec *models.GuDictionaryItem) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	// Rule 1: Level Penalty (Backlash)
	if actor.Rank < guSpec.Rank {
		actor.HP -= 50
		if actor.HP < 0 {
			actor.HP = 0
		}
		return 0, 0, 0, fmt.Errorf("Backlash: Cổ trùng Rank %d phản phệ Không Khiếu Rank %d của %s! (Trừ 50 HP)", guSpec.Rank, actor.Rank, actor.Name)
	}

	// Rule 2: Regional Suppression
	powerScale := 1.0

	activeEffects := models.GetGuEffectSection(*guSpec, "active")
	var baseDmg float64
	if dmg, ok := activeEffects["damage"]; ok {
		baseDmg = dmg
	} else if dmg, ok := activeEffects["add_base_attack"]; ok {
		baseDmg = dmg
	}

	var baseDef float64
	if def, ok := activeEffects["defense"]; ok {
		baseDef = def
	} else if def, ok := activeEffects["add_base_defense"]; ok {
		baseDef = def
	}

	cleanPath := strings.TrimSuffix(strings.ToLower(guSpec.Path), " path")

	// Cross-path penalty (if path attainment is less than Grandmaster, reduce power by 30%)
	attainmentLevel := 0
	if actor.PathMasteries != nil {
		if mLevel, ok := actor.PathMasteries[cleanPath]; ok {
			switch mLevel {
			case "Ordinary":
				attainmentLevel = 0
			case "Master":
				attainmentLevel = 1
			case "Grandmaster":
				attainmentLevel = 2
			case "Great Grandmaster":
				attainmentLevel = 3
			case "Supreme Grandmaster":
				attainmentLevel = 4
			}
		}
	}
	if attainmentLevel < 2 {
		powerScale *= 0.70
	}

	// Great Grandmaster/Supreme Grandmaster regional resonance (Natural Dao) (+50% power)
	if attainmentLevel >= 3 && actor.Kind == "Player" {
		currentRegion := ""
		playerUUID, parseErr := uuid.Parse(actor.ID)
		if parseErr == nil {
			_ = db.DB.QueryRow(`SELECT current_region FROM players WHERE id = $1`, playerUUID).Scan(&currentRegion)
		}
		isResonant := false
		switch strings.ToLower(currentRegion) {
		case "eastern_sea":
			isResonant = (cleanPath == "water")
		case "western_desert":
			isResonant = (cleanPath == "fire")
		case "northern_plains":
			isResonant = (cleanPath == "wind")
		case "southern_border":
			isResonant = (cleanPath == "earth")
		case "central_continent":
			isResonant = (cleanPath == "qi")
		}
		if isResonant {
			powerScale *= 1.50
		}
	}

	masteryMultiplier := getActorPathMasteryMultiplier(actor, guSpec.Path)

	daoMultiplier := 1.0
	if actor.DaoMarks != nil {
		if amt, ok := actor.DaoMarks[cleanPath]; ok && amt > 0 {
			if amt <= 10000 {
				daoMultiplier = 1.0 + float64(amt)/100.0
			} else {
				daoMultiplier = 1.0 + 100.0 + float64(amt-10000)/500.0
			}
		}
	}

	effectiveDamage = baseDmg * powerScale * masteryMultiplier * daoMultiplier
	effectiveDefense = baseDef * powerScale * masteryMultiplier * daoMultiplier

	cost := calculateEnergyCostForCombatant(actor, guSpec)

	return effectiveDamage, effectiveDefense, cost, nil
}

// CalculateGuEffect computes lore-accurate adjustments for level penalty, regional suppression, and mana cost
func CalculateGuEffect(player *models.Player, guSpec *models.GuDictionaryItem) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	actor := models.CombatantFromPlayer(player, CalculatePlayerStats(player))
	dmg, def, cost, err := CalculateGuEffectForCombatant(&actor, guSpec)
	player.HP = actor.HP
	return dmg, def, cost, err
}

// InitializeCombatState sets up many-vs-many combatants for a combat state
func InitializeCombatState(tx *sql.Tx, base *models.CombatState, player *models.Player) *models.CombatState {
	playerStats := CalculatePlayerStats(player)
	playerCombatant := models.CombatantFromPlayer(player, playerStats)
	// Base HP of players is fixed at 100
	playerCombatant.HP = player.HP
	playerCombatant.MaxHP = 100.0
	if playerCombatant.HP > playerCombatant.MaxHP {
		playerCombatant.HP = playerCombatant.MaxHP
	}
	if base.EnemyType == "PVP" {
		pHP := player.BaseHP
		if pHP <= 0 {
			pHP = 100.0
		}
		playerCombatant.HP = pHP
	}
	base.Allies = []models.Combatant{playerCombatant}

	enemyCombatant := models.CombatantFromActiveCombat(base)
	// Make sure enemy speed is set
	if enemyCombatant.Speed == 0 {
		enemyCombatant.Speed = 8.0 + float64(base.EnemyRank)*2.0 + float64(base.EnemySubRank)*0.5
	}
	base.Enemies = []models.Combatant{enemyCombatant}

	// Add nearby party members as Allies
	if player.PartyID != nil {
		var membersRaw []byte
		errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
		if errPart == nil {
			var members []PartyMember
			json.Unmarshal(membersRaw, &members)
			for _, m := range members {
				if m.ID != player.ID.String() {
					mUUID, errParse := uuid.Parse(m.ID)
					if errParse == nil {
						var mPlayer models.Player
						errMember := tx.QueryRow(`
							SELECT id, username, rank, sub_rank, hp, max_mana, current_mana, path_masteries, gu_inventory, dao_marks, body_mods, base_hp
							FROM players 
							WHERE id = $1 AND current_region = $2`,
							mUUID, player.CurrentRegion).Scan(
							&mPlayer.ID, &mPlayer.Username, &mPlayer.Rank, &mPlayer.SubRank,
							&mPlayer.HP, &mPlayer.MaxMana, &mPlayer.CurrentMana, &mPlayer.PathMasteries, &mPlayer.GuInventory, &mPlayer.DaoMarks, &mPlayer.BodyMods, &mPlayer.BaseHP,
						)
						if errMember == nil && (mPlayer.HP > 0 || base.EnemyType == "PVP") {
							mStats := CalculatePlayerStats(&mPlayer)
							mCombatant := models.CombatantFromPlayer(&mPlayer, mStats)
							mCombatant.MaxHP = 100.0
							if mCombatant.HP > mCombatant.MaxHP {
								mCombatant.HP = mCombatant.MaxHP
							}
							if base.EnemyType == "PVP" {
								mHP := mPlayer.BaseHP
								if mHP <= 0 {
									mHP = 100.0
								}
								mCombatant.HP = mHP
							}
							mCombatant.Kind = "Ally"
							base.Allies = append(base.Allies, mCombatant)
						}
					}
				}
			}
		}
	}

	// Add minion enemies for standard PVE encounters to support many-vs-many gameplay
	if base.TargetPlayerID == "" && base.EnemyTemplateID != "monster_heavenly_tribulation" && base.EnemyHP < 1000 {
		numMinions := rand.Intn(3) // 0 to 2 extra minions
		for i := 0; i < numMinions; i++ {
			minion := models.Combatant{
				ID:            fmt.Sprintf("minion_%d_%s", i, uuid.New().String()),
				Name:          fmt.Sprintf("Hộ Vệ %s", base.EnemyName),
				Kind:          "Normal",
				Rank:          base.EnemyRank,
				SubRank:       1,
				HP:            base.EnemyMaxHP * 0.4,
				MaxHP:         base.EnemyMaxHP * 0.4,
				CurrentMana:   base.EnemyMana * 0.4,
				MaxMana:       base.EnemyMaxMana * 0.4,
				BaseAttack:    base.EnemyDamage * 0.5,
				BaseDefense:   base.EnemyDefense * 0.5,
				AIProfile:     base.AIProfile,
				StatusEffects: []models.StatusEffect{},
			}
			base.Enemies = append(base.Enemies, minion)
		}
	}

	// Assign Grid Coordinates & default Speed stats for Allies
	allySlots := [][2]int{
		{2, 2}, // Caster center midline
		{2, 1}, // Top midline
		{2, 3}, // Bottom midline
		{3, 2}, // Center backline
		{3, 1}, // Top backline
		{3, 3}, // Bottom backline
		{1, 2}, // Center frontline
		{1, 1}, // Top frontline
		{1, 3}, // Bottom frontline
	}
	for idx := range base.Allies {
		if idx < len(allySlots) {
			base.Allies[idx].GridX = allySlots[idx][0]
			base.Allies[idx].GridY = allySlots[idx][1]
		}
		if base.Allies[idx].Speed == 0 {
			base.Allies[idx].Speed = 10.0 + float64(base.Allies[idx].Rank)*2.0 + float64(base.Allies[idx].SubRank)*0.5
		}
	}

	// Assign Grid Coordinates & default Speed stats for Enemies
	enemySlots := [][2]int{
		{1, 2}, // Leader frontline middle
		{1, 1}, // Frontline top
		{1, 3}, // Frontline bottom
		{2, 2}, // Midline middle
		{2, 1}, // Midline top
		{2, 3}, // Midline bottom
		{3, 2}, // Backline middle
		{3, 1}, // Backline top
		{3, 3}, // Backline bottom
	}
	for idx := range base.Enemies {
		if idx < len(enemySlots) {
			base.Enemies[idx].GridX = enemySlots[idx][0]
			base.Enemies[idx].GridY = enemySlots[idx][1]
		}
		if base.Enemies[idx].Speed == 0 {
			base.Enemies[idx].Speed = 8.0 + float64(base.Enemies[idx].Rank)*2.0 + float64(base.Enemies[idx].SubRank)*0.5
		}
	}

	return base
}

func StartCombat(tx *sql.Tx, base *models.CombatState, player *models.Player) *models.CombatState {
	initialized := InitializeCombatState(tx, base, player)

	// Sync this combat state to all allies in the database
	combatBytes, err := json.Marshal(initialized)
	if err == nil {
		for _, ally := range initialized.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil && allyUUID != player.ID {
				_, _ = tx.Exec("UPDATE players SET active_combat = $1 WHERE id = $2", combatBytes, allyUUID)
			}
		}
	}
	return initialized
}

func ResolveSingleTarget(team []models.Combatant, prefX, prefY int, hasBypass bool) (int, int, bool) {
	var alive []models.Combatant
	for _, c := range team {
		if c.HP > 0 {
			alive = append(alive, c)
		}
	}
	if len(alive) == 0 {
		return 0, 0, false
	}

	if hasBypass {
		// If preferred cell has an alive target, use it
		for _, c := range alive {
			if c.GridX == prefX && c.GridY == prefY {
				return prefX, prefY, true
			}
		}
		// Else fallback to any alive
		return alive[0].GridX, alive[0].GridY, true
	}

	// Non-bypass: check columns 1, then 2, then 3
	for col := 1; col <= 3; col++ {
		var colAlive []models.Combatant
		for _, c := range alive {
			if c.GridX == col {
				colAlive = append(colAlive, c)
			}
		}
		if len(colAlive) > 0 {
			// If preferred cell is in this column and alive, use it
			for _, c := range colAlive {
				if c.GridX == prefX && c.GridY == prefY {
					return c.GridX, c.GridY, true
				}
			}
			// Otherwise, pick the first in this column
			return colAlive[0].GridX, colAlive[0].GridY, true
		}
	}
	return alive[0].GridX, alive[0].GridY, true
}

func ResolveTargets(team []models.Combatant, targetType string, targetX, targetY int, hasBypass bool) []int {
	var targets []int

	switch targetType {
	case "ROW":
		for idx, c := range team {
			if c.HP > 0 && c.GridX == targetX {
				targets = append(targets, idx)
			}
		}
	case "COL":
		for idx, c := range team {
			if c.HP > 0 && c.GridY == targetY {
				targets = append(targets, idx)
			}
		}
	case "CROSS":
		// Center + 4 diagonals
		for idx, c := range team {
			if c.HP <= 0 {
				continue
			}
			dx := int(math.Abs(float64(c.GridX - targetX)))
			dy := int(math.Abs(float64(c.GridY - targetY)))
			if (c.GridX == targetX && c.GridY == targetY) || (dx == 1 && dy == 1) {
				targets = append(targets, idx)
			}
		}
	case "ALL":
		for idx, c := range team {
			if c.HP > 0 {
				targets = append(targets, idx)
			}
		}
	default: // "SINGLE"
		resX, resY, found := ResolveSingleTarget(team, targetX, targetY, hasBypass)
		if found {
			for idx, c := range team {
				if c.HP > 0 && c.GridX == resX && c.GridY == resY {
					targets = append(targets, idx)
					break
				}
			}
		}
	}

	// Fallback if no target matched
	if len(targets) == 0 {
		for idx, c := range team {
			if c.HP > 0 {
				targets = append(targets, idx)
				break
			}
		}
	}

	return targets
}

func applyDamageWithLogs(initiatorName, targetName string, target *models.Combatant, damage float64) string {
	if (target.Kind == "Player" || target.Kind == "Ally") && target.Rank >= 6 {
		apertureDmg := damage * 0.5
		target.ApertureDmg += apertureDmg
		damage = damage / 200.0
		// Optional: We can append text later, but returning string here is simple
	}

	if target.VirtualShield > 0 {
		if target.VirtualShield >= damage {
			target.VirtualShield -= damage
			return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (Bị hấp thụ hoàn toàn bởi Khiên ảo, còn %.1f Khiên).",
				initiatorName, damage, targetName, target.VirtualShield)
		} else {
			shieldDmg := target.VirtualShield
			hpDmg := damage - shieldDmg
			target.VirtualShield = 0
			target.HP -= hpDmg
			if target.HP < 0 {
				target.HP = 0
			}
			return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (Lớp Khiên ảo bị đánh vỡ, %.1f sát thương trừ thẳng vào HP, còn %.1f HP).",
				initiatorName, damage, targetName, hpDmg, target.HP)
		}
	} else {
		target.HP -= damage
		if target.HP < 0 {
			target.HP = 0
		}
		return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (%.1f sát thương trừ thẳng vào HP, còn %.1f HP).",
			initiatorName, damage, targetName, damage, target.HP)
	}
}

func findCombatant(combatState *models.CombatState, id string) (*models.Combatant, bool, int) {
	for idx, c := range combatState.Allies {
		if c.ID == id {
			return &combatState.Allies[idx], true, idx
		}
	}
	for idx, c := range combatState.Enemies {
		if c.ID == id {
			return &combatState.Enemies[idx], false, idx
		}
	}
	return nil, false, -1
}

func calculateLooseGuFailureChance(specs []models.GuDictionaryItem) float64 {
	if len(specs) <= 1 {
		return 0
	}
	paths := map[string]bool{}
	for _, spec := range specs {
		paths[spec.Path] = true
	}
	pathChaos := math.Max(0, float64(len(paths)-1)) * 0.08
	chance := float64(len(specs)-1)*0.12 + pathChaos
	if chance > 0.75 {
		chance = 0.75
	}
	return chance
}

// CombatAttack handles PVE/PVP turns and execution of actions simultaneously
func CombatAttack(c *gin.Context) {
	var req CombatAttackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get auth player ID
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

	// 1. Fetch & Lock Player Row
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// 2. Validate Combat State
	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong trạng thái chiến đấu."})
		return
	}

	if player.HP <= 0 && player.ActiveCombat.EnemyType != "PVP" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã kiệt sức. Dưỡng thương trước khi chiến đấu."})
		return
	}

	// Initialize Allies and Enemies lists if not populated yet
	if len(player.ActiveCombat.Allies) == 0 {
		player.ActiveCombat = InitializeCombatState(tx, player.ActiveCombat, &player)
	}

	// Process cooldowns
	if player.ActiveCombat.SkillCooldowns != nil {
		for k, v := range player.ActiveCombat.SkillCooldowns {
			if v > 0 {
				player.ActiveCombat.SkillCooldowns[k] = v - 1
			}
		}
	}

	var turnLogs []string

	// Process Debuffs for Allies
	for i := range player.ActiveCombat.Allies {
		c := &player.ActiveCombat.Allies[i]
		if c.HP <= 0 {
			continue
		}
		var activeEffs []models.StatusEffect
		for j := 0; j < len(c.StatusEffects); j++ {
			eff := c.StatusEffects[j]
			if eff.Turns > 0 {
				if eff.Type == "poison" || eff.Type == "bleed" {
					c.HP -= eff.Value
					turnLogs = append(turnLogs, fmt.Sprintf("%s mất %.1f HP do %s (Còn %d lượt).", c.Name, eff.Value, eff.Type, eff.Turns))
				}
				eff.Turns--
			}
			if eff.Turns > 0 {
				activeEffs = append(activeEffs, eff)
			}
		}
		c.StatusEffects = activeEffs
	}

	// Process Debuffs for Enemies
	for i := range player.ActiveCombat.Enemies {
		c := &player.ActiveCombat.Enemies[i]
		if c.HP <= 0 {
			continue
		}
		var activeEffs []models.StatusEffect
		for j := 0; j < len(c.StatusEffects); j++ {
			eff := c.StatusEffects[j]
			if eff.Turns > 0 {
				if eff.Type == "poison" || eff.Type == "bleed" {
					c.HP -= eff.Value
					turnLogs = append(turnLogs, fmt.Sprintf("%s mất %.1f HP do %s (Còn %d lượt).", c.Name, eff.Value, eff.Type, eff.Turns))
				}
				eff.Turns--
			}
			if eff.Turns > 0 {
				activeEffs = append(activeEffs, eff)
			}
		}
		c.StatusEffects = activeEffs
	}

	// Find the caster inside Allies list and synchronize their current HP/Mana
	var casterIdx = -1
	for idx, ally := range player.ActiveCombat.Allies {
		if ally.ID == player.ID.String() {
			casterIdx = idx
			if player.ActiveCombat.EnemyType != "PVP" {
				player.ActiveCombat.Allies[idx].HP = player.HP
			}
			player.ActiveCombat.Allies[idx].CurrentMana = player.CurrentMana
			break
		}
	}
	if casterIdx == -1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find player in combatants"})
		return
	}

	playerStats := CalculatePlayerStats(&player)
	player.ActiveCombat.Allies[casterIdx].BaseAttack = playerStats["attack"]
	player.ActiveCombat.Allies[casterIdx].BaseDefense = playerStats["defense"]

	playerCombatant := player.ActiveCombat.Allies[casterIdx]

	// 3. Resolve equipped Gus and compute player planned actions
	var selectedGus []models.GuInstance
	var selectedSpecs []models.GuDictionaryItem
	totalManaCost := 0.0
	looseGuFailureChance := 0.0

	isChargingThisTurn := false
	isUnleashingCharge := false
	isUsingItem := false
	isUsingStones := false

	if req.UseStones != "" {
		isUsingStones = true
		player.ActiveCombat.ChargePoints = 0
	} else if req.ItemID != "" {
		isUsingItem = true
		player.ActiveCombat.ChargePoints = 0

		if player.ActiveCombat.ItemsUsedCount >= 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ được dùng tối đa 3 vật phẩm trong mỗi trận chiến!"})
			return
		}
		if player.ActiveCombat.ItemCooldowns == nil {
			player.ActiveCombat.ItemCooldowns = make(map[string]int)
		}
		if player.ActiveCombat.ItemCooldowns[req.ItemID] > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm đang hồi chiêu (còn %d lượt)", player.ActiveCombat.ItemCooldowns[req.ItemID])})
			return
		}

		qty, ok := player.MaterialsInventory[req.ItemID]
		if !ok || qty <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm này trong túi đồ"})
			return
		}
		item, ok := models.ItemRegistry[req.ItemID]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại"})
			return
		}
		if item.Type != "Consumable" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này không thể dùng trong chiến đấu"})
			return
		}

		player.MaterialsInventory[req.ItemID]--
		if player.MaterialsInventory[req.ItemID] <= 0 {
			delete(player.MaterialsInventory, req.ItemID)
		}
		player.ActiveCombat.ItemsUsedCount++
		player.ActiveCombat.ItemCooldowns[req.ItemID] = 3 // 3 turns cooldown

	} else if req.IsCharge {
		player.ActiveCombat.ChargePoints++
		if player.ActiveCombat.ChargePoints >= 2 {
			isUnleashingCharge = true
			player.ActiveCombat.ChargePoints = 0
		} else {
			isChargingThisTurn = true
		}
	} else {
		// Reset charge points if normal turn is taken
		player.ActiveCombat.ChargePoints = 0

		// Verify and load chosen Gu bugs
		for _, instID := range req.EquippedGuIDs {
			found := false
			for _, inst := range player.GuInventory {
				if inst.InstanceID == instID {
					if inst.Status == "DEAD" {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể chiến đấu bằng Cổ trùng ĐÃ CHẾT: " + inst.GuID})
						return
					}
					spec, ok := models.GuRegistry[inst.GuID]
					if !ok {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy định nghĩa của Cổ trùng: " + inst.GuID})
						return
					}

					// Ràng buộc Bí cảnh (Secret Realm): Chỉ dùng Moon Path hoặc Earth Path
					if player.CurrentRegion == "three_kings_inheritance" {
						if spec.Path != "Moon Path" && spec.Path != "Earth Path" {
							c.JSON(http.StatusBadRequest, gin.H{"error": "Bí cảnh Tam Vương Truyền Thừa áp chế Cổ trùng không thuộc Nguyệt đạo hoặc Thổ đạo: " + spec.Name})
							return
						}
					}

					selectedGus = append(selectedGus, inst)
					selectedSpecs = append(selectedSpecs, spec)

					// Calculate mana cost for validation
					_, _, cost, errVal := CalculateGuEffectForCombatant(&playerCombatant, &spec)
					if errVal == nil {
						totalManaCost += cost
					}
					found = true
					break
				}
			}
			if !found {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Cổ trùng trong túi đồ: " + instID})
				return
			}
		}

		if playerCombatant.CurrentMana < totalManaCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
			return
		}
		looseGuFailureChance = calculateLooseGuFailureChance(selectedSpecs)
	}

	// Find the targeted enemy
	targetID := req.TargetID
	targetIdx := -1
	for idx, enemy := range player.ActiveCombat.Enemies {
		if enemy.HP > 0 {
			if targetID == "" || enemy.ID == targetID {
				targetIdx = idx
				break
			}
		}
	}
	// Fallback to first alive enemy
	if targetIdx == -1 {
		for idx, enemy := range player.ActiveCombat.Enemies {
			if enemy.HP > 0 {
				targetIdx = idx
				break
			}
		}
	}

	if len(selectedSpecs) > 1 {
		turnLogs = append(turnLogs, fmt.Sprintf("=== DÙNG CỔ RIÊNG LẺ (rủi ro hụt %.0f%%) ===", looseGuFailureChance*100))
	} else {
		turnLogs = append(turnLogs, "=== HÀNH ĐỘNG CHIẾN ĐẤU ===")
	}

	// Build the list of actions for this turn
	type CombatAction struct {
		InitiatorID string
		Speed       float64
		Execute     func() []string
	}
	var actions []CombatAction

	// 1. Add Caster (Player) actions
	if targetIdx != -1 {
		casterCombatant := &player.ActiveCombat.Allies[casterIdx]
		prefX := req.TargetX
		prefY := req.TargetY
		if prefX <= 0 {
			prefX = 1
		}
		if prefY <= 0 {
			prefY = 2
		}

		actions = append(actions, CombatAction{
			InitiatorID: player.ID.String(),
			Speed:       casterCombatant.Speed,
			Execute: func() []string {
				var logs []string
				refCaster, _, _ := findCombatant(player.ActiveCombat, player.ID.String())
				if refCaster == nil || refCaster.HP <= 0 {
					return nil
				}
				for _, eff := range refCaster.StatusEffects {
					if eff.Type == "stun" && eff.Turns > 0 {
						return []string{fmt.Sprintf("-> %s bị choáng và mất lượt!", refCaster.Name)}
					}
				}

				if isUsingStones {
					if req.UseStones == "immortal" {
						if player.ImmortalStones >= 1 {
							player.ImmortalStones -= 1
							refCaster.CurrentMana += 100.0 // Giả định 100 tiên nguyên
							if refCaster.CurrentMana > refCaster.MaxMana {
								refCaster.CurrentMana = refCaster.MaxMana
							}
							logs = append(logs, fmt.Sprintf("-> %s nghiền nát 1 Tiên Nguyên Thạch, khôi phục mạnh mẽ tiên nguyên!", refCaster.Name))
						} else {
							logs = append(logs, fmt.Sprintf("-> %s định dùng Tiên Nguyên Thạch nhưng không đủ!", refCaster.Name))
						}
					} else {
						if player.PrimevalStones >= 100 {
							player.PrimevalStones -= 100
							refCaster.CurrentMana += 50.0
							if refCaster.CurrentMana > refCaster.MaxMana {
								refCaster.CurrentMana = refCaster.MaxMana
							}
							logs = append(logs, fmt.Sprintf("-> %s hấp thu 100 Nguyên Thạch, khôi phục chân nguyên!", refCaster.Name))
						} else {
							logs = append(logs, fmt.Sprintf("-> %s định dùng Nguyên Thạch nhưng không đủ (cần 100)!", refCaster.Name))
						}
					}
				} else if isUsingItem {
					itemSpec := models.ItemRegistry[req.ItemID]
					healVal := itemSpec.Effects["restore_hp"]
					if healVal == 0 {
						healVal = 50.0
					}

					if refCaster.Rank >= 6 {
						refCaster.ApertureDmg -= healVal
						if refCaster.ApertureDmg < 0 {
							refCaster.ApertureDmg = 0
						}
						logs = append(logs, fmt.Sprintf("-> %s sử dụng %s, giảm %.1f hư tổn tiên khiếu!", refCaster.Name, itemSpec.Name, healVal))
					} else {
						refCaster.HP += healVal
						if refCaster.HP > refCaster.MaxHP {
							refCaster.HP = refCaster.MaxHP
						}
						logs = append(logs, fmt.Sprintf("-> %s sử dụng %s, khôi phục %.1f HP (HP hiện tại: %.1f).", refCaster.Name, itemSpec.Name, healVal, refCaster.HP))
					}
				} else if isChargingThisTurn {
					logs = append(logs, fmt.Sprintf("%s tích tụ Nguyên Khí, vận chuyển Chân Nguyên chuẩn bị cho Đại Sát Chiêu!", refCaster.Name))
				} else if isUnleashingCharge {
					targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", prefX, prefY, false)
					if len(targetsIdx) > 0 {
						targetEnemy := &player.ActiveCombat.Enemies[targetsIdx[0]]
						dmgValue := playerStats["attack"] * 2.5
						netDmg := dmgValue - targetEnemy.BaseDefense
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						dmgLog := applyDamageWithLogs(refCaster.Name, targetEnemy.Name, targetEnemy, netDmg)
						logs = append(logs, fmt.Sprintf("-> %s kích hoạt Bộc Phá Sát Chiêu (Unleash Charge)!", refCaster.Name), dmgLog)
					}
				} else {
					if looseGuFailureChance > 0 && rand.Float64() < looseGuFailureChance {
						wastedMana := totalManaCost * 0.5
						if wastedMana > refCaster.CurrentMana {
							wastedMana = refCaster.CurrentMana
						}
						refCaster.CurrentMana -= wastedMana
						logs = append(logs, fmt.Sprintf("-> %s cưỡng ép vận nhiều Cổ riêng lẻ, đạo ngân hỗn loạn khiến chiêu thất bại! Hao phí %.1f nguyên khí.", refCaster.Name, wastedMana))
						return logs
					}

					for _, spec := range selectedSpecs {
						dmg, def, cost, errGu := CalculateGuEffectForCombatant(refCaster, &spec)
						if errGu == nil && refCaster.CurrentMana >= cost {
							refCaster.CurrentMana -= cost

							activeEffects := models.GetGuEffectSection(spec, "active")
							isHealing := activeEffects["healing"] > 0 || activeEffects["restore_hp"] > 0 || activeEffects["heal_based_on_strength"] > 0 || spec.ID == "GU_NINE_LEAF_VITALITY_GRASS"
							isDefense := spec.Type == "Defense" || activeEffects["add_base_defense"] > 0 || activeEffects["defense"] > 0

							if isHealing {
								healVal := def
								if activeEffects["heal_based_on_strength"] > 0 {
									healVal = refCaster.BaseAttack * 1.5
								}
								if healVal <= 0 {
									healVal = 30.0
								}
								refCaster.HP += healVal
								if refCaster.HP > refCaster.MaxHP {
									refCaster.HP = refCaster.MaxHP
								}
								logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s, khôi phục %.1f HP bản thân (HP hiện tại: %.1f).", refCaster.Name, spec.Name, healVal, refCaster.HP))
							} else if isDefense {
								shieldVal := def * 10.0
								if shieldVal <= 0 {
									shieldVal = 150.0
								}
								refCaster.VirtualShield += shieldVal
								logs = append(logs, fmt.Sprintf("-> %s thi triển %s phòng thủ, nhận %.1f Khiên hộ thể (Virtual Shield: %.1f).", refCaster.Name, spec.Name, shieldVal, refCaster.VirtualShield))
							} else {
								hasBypass := activeEffects["assassination"] > 0 || activeEffects["flying"] > 0
								targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, spec.TargetType, prefX, prefY, hasBypass)
								if len(targetsIdx) > 0 {
									logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s!", refCaster.Name, spec.Name))
									for _, eIdx := range targetsIdx {
										tEnemy := &player.ActiveCombat.Enemies[eIdx]
										netDmg := dmg - tEnemy.BaseDefense
										if netDmg < 1.0 {
											netDmg = 1.0
										}
										dmgLog := applyDamageWithLogs(refCaster.Name, tEnemy.Name, tEnemy, netDmg)
										logs = append(logs, dmgLog)
									}
								}
							}
						}
					}
					if len(selectedSpecs) == 0 {
						targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", prefX, prefY, false)
						if len(targetsIdx) > 0 {
							targetEnemy := &player.ActiveCombat.Enemies[targetsIdx[0]]
							netDmg := 5.0 - targetEnemy.BaseDefense
							if netDmg < 1.0 {
								netDmg = 1.0
							}
							dmgLog := applyDamageWithLogs(refCaster.Name, targetEnemy.Name, targetEnemy, netDmg)
							logs = append(logs, fmt.Sprintf("-> %s đánh thường (Basic Attack)!", refCaster.Name), dmgLog)
						}
					}
				}
				return logs
			},
		})
	}

	// 2. Add other Allies actions
	for aIdx := range player.ActiveCombat.Allies {
		ally := &player.ActiveCombat.Allies[aIdx]
		if ally.ID != player.ID.String() && ally.HP > 0 {
			allyID := ally.ID
			actions = append(actions, CombatAction{
				InitiatorID: allyID,
				Speed:       ally.Speed,
				Execute: func() []string {
					var logs []string
					refAlly, _, _ := findCombatant(player.ActiveCombat, allyID)
					if refAlly == nil || refAlly.HP <= 0 {
						return nil
					}
					for _, eff := range refAlly.StatusEffects {
						if eff.Type == "stun" && eff.Turns > 0 {
							return []string{fmt.Sprintf("-> %s bị choáng và mất lượt!", refAlly.Name)}
						}
					}

					targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", 1, 2, false)
					if len(targetsIdx) > 0 {
						tEnemy := &player.ActiveCombat.Enemies[targetsIdx[0]]
						dmgVal := refAlly.BaseAttack
						if dmgVal <= 0 {
							dmgVal = 8.0
						}
						netDmg := dmgVal - tEnemy.BaseDefense
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						dmgLog := applyDamageWithLogs(refAlly.Name, tEnemy.Name, tEnemy, netDmg)
						logs = append(logs, fmt.Sprintf("-> Đồng minh %s đánh thường lên %s.", refAlly.Name, tEnemy.Name), dmgLog)
					}
					return logs
				},
			})
		}
	}

	// 3. Add Enemies actions
	for eIdx := range player.ActiveCombat.Enemies {
		enemy := &player.ActiveCombat.Enemies[eIdx]
		if enemy.HP > 0 {
			enemyID := enemy.ID
			actions = append(actions, CombatAction{
				InitiatorID: enemyID,
				Speed:       enemy.Speed,
				Execute: func() []string {
					var logs []string
					refEnemy, _, _ := findCombatant(player.ActiveCombat, enemyID)
					if refEnemy == nil || refEnemy.HP <= 0 {
						return nil
					}
					for _, eff := range refEnemy.StatusEffects {
						if eff.Type == "stun" && eff.Turns > 0 {
							return []string{fmt.Sprintf("-> %s bị choáng và mất lượt!", refEnemy.Name)}
						}
					}

					targetsIdx := ResolveTargets(player.ActiveCombat.Allies, "SINGLE", 2, 2, false)
					if len(targetsIdx) > 0 {
						targetAlly := &player.ActiveCombat.Allies[targetsIdx[0]]
						enemyCtx := &CombatContext{
							Attacker: refEnemy,
							Defender: targetAlly,
							Logs:     []string{},
						}
						enemyActions := BuildEnemyActions(enemyCtx, refEnemy)
						if len(enemyActions) > 0 {
							act := enemyActions[0]
							if act.ManaCost > 0 && refEnemy.CurrentMana >= act.ManaCost {
								refEnemy.CurrentMana -= act.ManaCost
							}

							isHealing := act.Def > 0 && (act.IsUtility || (len(act.StatusEffects) == 0 && act.Dmg == 0))
							if isHealing {
								healVal := act.Def
								refEnemy.HP += healVal
								if refEnemy.HP > refEnemy.MaxHP {
									refEnemy.HP = refEnemy.MaxHP
								}
								logs = append(logs, fmt.Sprintf("-> %s kích hoạt phòng ngự/trị liệu %s, hồi %.1f HP.", refEnemy.Name, act.Name, healVal))
							} else {
								netDmg := act.Dmg - targetAlly.BaseDefense
								if netDmg < 1.0 {
									netDmg = 1.0
								}
								dmgLog := applyDamageWithLogs(refEnemy.Name, targetAlly.Name, targetAlly, netDmg)
								logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s tấn công!", refEnemy.Name, act.Name), dmgLog)
							}
						} else {
							dmgVal := refEnemy.BaseAttack
							if dmgVal <= 0 {
								dmgVal = 5.0
							}
							netDmg := dmgVal - targetAlly.BaseDefense
							if netDmg < 1.0 {
								netDmg = 1.0
							}
							dmgLog := applyDamageWithLogs(refEnemy.Name, targetAlly.Name, targetAlly, netDmg)
							logs = append(logs, fmt.Sprintf("-> %s đánh thường!", refEnemy.Name), dmgLog)
						}
					}
					return logs
				},
			})
		}
	}

	// 4. Sort actions by speed descending
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Speed > actions[j].Speed
	})

	// 5. Execute actions sequentially
	for _, act := range actions {
		refActor, _, _ := findCombatant(player.ActiveCombat, act.InitiatorID)
		if refActor == nil || refActor.HP <= 0 {
			continue
		}
		resLogs := act.Execute()
		if len(resLogs) > 0 {
			turnLogs = append(turnLogs, resLogs...)
		}
	}

	// Sync Caster HP and Mana
	if player.ActiveCombat.EnemyType != "PVP" {
		player.HP = player.ActiveCombat.Allies[casterIdx].HP
	}
	player.CurrentMana = player.ActiveCombat.Allies[casterIdx].CurrentMana

	// Update HP for other database players in battle
	for _, ally := range player.ActiveCombat.Allies {
		if ally.ID != player.ID.String() {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil {
				allyHP := ally.HP
				if allyHP < 0 {
					allyHP = 0
				}
				if player.ActiveCombat.EnemyType != "PVP" {
					_, _ = tx.Exec("UPDATE players SET hp = $1 WHERE id = $2", allyHP, allyUUID)
				}
			}
		}
	}

	// 7. Check Victory or Defeat
	victory := true
	for _, enemy := range player.ActiveCombat.Enemies {
		if enemy.HP > 0 {
			victory = false
			break
		}
	}

	defeat := player.HP <= 0
	isPVP := player.ActiveCombat.TargetPlayerID != ""

	// Create a temporary combat context for logs
	ctx := &CombatContext{
		Attacker: &playerCombatant,
		Logs:     turnLogs,
	}

	if victory {
		ctx.Logs = append(ctx.Logs, "=== CHIẾN THẮNG VANG DỘI ===")

		// Integrate passive Gus
		IntegratePassiveGus(&player, &ctx.Logs)

		// Determine party members on the same cell
		var partyMembers []uuid.UUID
		if player.PartyID != nil {
			var membersRaw []byte
			errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
			if errPart == nil {
				var members []PartyMember
				json.Unmarshal(membersRaw, &members)
				for _, m := range members {
					mUUID, errParse := uuid.Parse(m.ID)
					if errParse == nil {
						var sameCell bool
						errCell := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE id = $1 AND x = $2 AND y = $3 AND current_region = $4)", mUUID, player.X, player.Y, player.CurrentRegion).Scan(&sameCell)
						if errCell == nil && sameCell {
							partyMembers = append(partyMembers, mUUID)
						}
					}
				}
			}
		}

		if isPVP {
			// PVP Victory
			stolenStones := 10
			if player.ActiveCombat.TargetPlayerID != "dummy" {
				targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
				if parseErr == nil {
					var targetStones int
					var targetHP float64
					var targetLootBytes []byte
					err = tx.QueryRow("SELECT primeval_stones, hp, temp_loot FROM players WHERE id = $1 FOR UPDATE", targetUUID).Scan(&targetStones, &targetHP, &targetLootBytes)
					if err == nil {
						stolenStones = int(float64(targetStones) * 0.10)
						if stolenStones < 1 {
							stolenStones = 1
						}
						if targetStones < stolenStones {
							stolenStones = targetStones
						}
						targetStones -= stolenStones
						targetHP -= 30.0
						if targetHP < 10.0 {
							targetHP = 10.0
						}

						// Transfer target's temp_loot to winner (player)
						var targetLoot models.TempLoot
						if len(targetLootBytes) > 0 {
							json.Unmarshal(targetLootBytes, &targetLoot)
						}

						player.TempLoot.PrimevalStones += targetLoot.PrimevalStones
						if player.TempLoot.MaterialsInventory == nil {
							player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
						}
						for matID, qty := range targetLoot.MaterialsInventory {
							player.TempLoot.MaterialsInventory[matID] += qty
						}
						player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, targetLoot.GuInventory...)

						// Empty target's temp_loot
						emptyLoot := models.TempLoot{
							GuInventory:        models.GuInventory{},
							MaterialsInventory: models.MaterialsInventory{},
							PrimevalStones:     0,
						}

						_, _ = tx.Exec("UPDATE players SET primeval_stones = $1, hp = $2, temp_loot = $3 WHERE id = $4", targetStones, targetHP, emptyLoot, targetUUID)

						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Thu hoạch toàn bộ chiến lợi phẩm tạm thời của bại binh! Hút được %d Nguyên Thạch, %d Cổ trùng, và các nguyên liệu khác từ đối phương.", targetLoot.PrimevalStones, len(targetLoot.GuInventory)))
					}

					// Arena rank swap check
					var playerRank, targetRank int
					errPR := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", player.ID).Scan(&playerRank)
					errTR := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", targetUUID).Scan(&targetRank)
					if errPR == nil && errTR == nil && playerRank > targetRank {
						// Swap ranks
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", playerRank, targetUUID)
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", targetRank, player.ID)
						_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", playerRank, targetUUID)
						_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", targetRank, player.ID)
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chúc mừng! Đạo hữu đã vượt cấp khiêu chiến thành công, thăng lên Hạng %d Võ Đài!", targetRank))
					}
				}
			}
			player.PrimevalStones += stolenStones
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiến thắng PVP vang dội! Đạo hữu cướp đoạt %d Nguyên Thạch từ đối phương!", stolenStones))
		} else {
			// PVE Victory
			isExplore := player.CurrentRegion == "qing_mao_mountain" || player.CurrentRegion == "gu_luan_slope" || player.CurrentRegion == "three_kings_inheritance"
			if player.ActiveCombat != nil && player.ActiveCombat.LootTableID != "" {
				lt, err := db.LoadLootTable(tx, player.ActiveCombat.LootTableID)
				if err == nil {
					loot := RollLoot(lt)
					if loot.Items != nil {
						delete(loot.Items, "primeval_stone")
					}
					if len(partyMembers) > 1 {
						ctx.Logs = append(ctx.Logs, "=== PHÂN CHIA CHIẾN LỢI PHẨM TỔ ĐỘI ===")
						stoneQty := loot.Items["primeval_stone"]
						if stoneQty > 0 {
							splitStones := stoneQty / len(partyMembers)
							if splitStones < 1 {
								splitStones = 1
							}
							for _, memberID := range partyMembers {
								if memberID == player.ID {
									if isExplore {
										player.TempLoot.PrimevalStones += splitStones
										ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch (giữ tạm thời).", splitStones))
									} else {
										player.PrimevalStones += splitStones
										ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch.", splitStones))
									}
								} else {
									if isExplore {
										var targetLoot models.TempLoot
										var targetLootBytes []byte
										_ = tx.QueryRow("SELECT temp_loot FROM players WHERE id = $1 FOR UPDATE", memberID).Scan(&targetLootBytes)
										if len(targetLootBytes) > 0 {
											json.Unmarshal(targetLootBytes, &targetLoot)
										}
										targetLoot.PrimevalStones += splitStones
										_, _ = tx.Exec("UPDATE players SET temp_loot = $1 WHERE id = $2", targetLoot, memberID)
									} else {
										_, _ = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", splitStones, memberID)
									}
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", memberID).Scan(&uname)
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đồng đội %s nhận được %d Nguyên Thạch.", uname, splitStones))
								}
							}
						}

						for matID, qty := range loot.Items {
							if matID == "primeval_stone" {
								continue
							}
							for q := 0; q < qty; q++ {
								winnerID := partyMembers[rand.Intn(len(partyMembers))]
								if winnerID == player.ID {
									if isExplore {
										if player.TempLoot.MaterialsInventory == nil {
											player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
										}
										player.TempLoot.MaterialsInventory[matID]++
										ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhặt được 1x %s (giữ tạm thời).", matID))
									} else {
										if player.MaterialsInventory == nil {
											player.MaterialsInventory = make(models.MaterialsInventory)
										}
										player.MaterialsInventory[matID]++
										ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhặt được 1x %s.", matID))
									}
								} else {
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)

									if isExplore {
										var targetLoot models.TempLoot
										var targetLootBytes []byte
										_ = tx.QueryRow("SELECT temp_loot FROM players WHERE id = $1 FOR UPDATE", winnerID).Scan(&targetLootBytes)
										if len(targetLootBytes) > 0 {
											json.Unmarshal(targetLootBytes, &targetLoot)
										}
										if targetLoot.MaterialsInventory == nil {
											targetLoot.MaterialsInventory = make(models.MaterialsInventory)
										}
										targetLoot.MaterialsInventory[matID]++
										_, _ = tx.Exec("UPDATE players SET temp_loot = $1 WHERE id = $2", targetLoot, winnerID)
									} else {
										var targetInv models.MaterialsInventory
										var targetInvBytes []byte
										_ = tx.QueryRow("SELECT materials_inventory FROM players WHERE id = $1 FOR UPDATE", winnerID).Scan(&targetInvBytes)
										if len(targetInvBytes) > 0 {
											json.Unmarshal(targetInvBytes, &targetInv)
										}
										if targetInv == nil {
											targetInv = make(models.MaterialsInventory)
										}
										targetInv[matID]++
										_, _ = tx.Exec("UPDATE players SET materials_inventory = $1 WHERE id = $2", targetInv, winnerID)
									}
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đồng đội %s nhặt được 1x %s.", uname, matID))
								}
							}
						}

						for _, gID := range loot.Gus {
							winnerID := partyMembers[rand.Intn(len(partyMembers))]
							newGu := models.GuInstance{
								InstanceID: uuid.New().String(),
								GuID:       gID,
								Hunger:     100,
								Status:     "ACTIVE",
								Equipped:   false,
							}
							if winnerID == player.ID {
								if isExplore {
									player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, newGu)
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn bắt được 1x %s (giữ tạm thời)!", gID))
								} else {
									player.GuInventory = append(player.GuInventory, newGu)
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn bắt được 1x %s!", gID))
								}
							} else {
								var uname string
								_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)

								if isExplore {
									var targetLoot models.TempLoot
									var targetLootBytes []byte
									_ = tx.QueryRow("SELECT temp_loot FROM players WHERE id = $1 FOR UPDATE", winnerID).Scan(&targetLootBytes)
									if len(targetLootBytes) > 0 {
										json.Unmarshal(targetLootBytes, &targetLoot)
									}
									targetLoot.GuInventory = append(targetLoot.GuInventory, newGu)
									_, _ = tx.Exec("UPDATE players SET temp_loot = $1 WHERE id = $2", targetLoot, winnerID)
								} else {
									var targetGuInv models.GuInventory
									var targetGuInvBytes []byte
									_ = tx.QueryRow("SELECT gu_inventory FROM players WHERE id = $1 FOR UPDATE", winnerID).Scan(&targetGuInvBytes)
									if len(targetGuInvBytes) > 0 {
										json.Unmarshal(targetGuInvBytes, &targetGuInv)
									}
									targetGuInv = append(targetGuInv, newGu)
									_, _ = tx.Exec("UPDATE players SET gu_inventory = $1 WHERE id = $2", targetGuInv, winnerID)
								}
								ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đồng đội %s thu phục thành công Cổ trùng: %s!", uname, gID))
							}
						}
					} else {
						lootLogs := ApplyLoot(&player, loot)
						ctx.Logs = append(ctx.Logs, lootLogs...)
					}
				} else {
					log.Printf("Failed to load loot table %s: %v", player.ActiveCombat.LootTableID, err)
				}
			} else {
				// No fallback stones in PVE victory as requested
				ctx.Logs = append(ctx.Logs, "Chiến đấu thắng lợi vang dội!")
			}

			// If it is a mine slot challenge victory:
			if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 {
				slotID := player.ActiveCombat.MineSlotID
				var rarity string
				var currentOccupierStr *string
				var lastClaimedAt *time.Time
				err = tx.QueryRow("SELECT rarity, occupier_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE", slotID).Scan(&rarity, &currentOccupierStr, &lastClaimedAt)
				if err == nil && currentOccupierStr != nil {
					currOccupierUUID, errParse := uuid.Parse(*currentOccupierStr)
					if errParse == nil && lastClaimedAt != nil {
						stones, qi, chests := CalculateMineRewards(slotID, rarity, *lastClaimedAt)
						if stones > 0 || qi > 0.0 || chests > 0 {
							_ = AddMineRewardsToPlayerTx(tx, currOccupierUUID, stones, qi, chests)
						}
					}
				}

				now := time.Now()
				_, errUpdate := tx.Exec(`
					UPDATE mine_slots
					SET occupier_id = $1, monster_id = NULL, occupied_at = $2, last_claimed_at = $3
					WHERE id = $4
				`, player.ID, now, now, slotID)
				if errUpdate == nil {
					ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiếm lĩnh thành công mỏ phẩm chất %s #%d!", rarity, slotID))
				} else {
					log.Printf("Failed to update mine slot occupier in combat victory: %v", errUpdate)
				}
			}

			if player.ClearedCells == nil {
				player.ClearedCells = make(models.ClearedCellsMap)
			}
			coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
			player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
		}

		// Clear active combat for all allies
		for _, ally := range player.ActiveCombat.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil {
				if allyUUID != player.ID {
					_, _ = tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", allyUUID)
				}
			}
		}
		player.ActiveCombat = nil
	} else if defeat {
		player.HP = 20.0
		defeat = true
		if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 {
			ctx.Logs = append(ctx.Logs, "Thách đấu mỏ linh thạch thất bại, không thể chiếm lĩnh mỏ.")
		} else {
			ctx.Logs = append(ctx.Logs, "Đại bại! Đạo hữu bất tỉnh nhân sự trước áp lực của đối thủ. May mắn thoát chết nhưng rơi rớt mất sạch chiến lợi phẩm thu thập trong chuyến đi này.")
		}

		// Transfer player's temp_loot to the PVP target if isPVP
		if isPVP && player.ActiveCombat.TargetPlayerID != "dummy" {
			targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
			if parseErr == nil {
				var targetLootBytes []byte
				errLoot := tx.QueryRow("SELECT temp_loot FROM players WHERE id = $1 FOR UPDATE", targetUUID).Scan(&targetLootBytes)
				if errLoot == nil {
					var targetLoot models.TempLoot
					if len(targetLootBytes) > 0 {
						json.Unmarshal(targetLootBytes, &targetLoot)
					}
					// Add player's temp_loot to target's temp_loot
					targetLoot.PrimevalStones += player.TempLoot.PrimevalStones
					if targetLoot.MaterialsInventory == nil {
						targetLoot.MaterialsInventory = make(models.MaterialsInventory)
					}
					for matID, qty := range player.TempLoot.MaterialsInventory {
						targetLoot.MaterialsInventory[matID] += qty
					}
					targetLoot.GuInventory = append(targetLoot.GuInventory, player.TempLoot.GuInventory...)

					_, _ = tx.Exec("UPDATE players SET temp_loot = $1 WHERE id = $2", targetLoot, targetUUID)
				}
			}
		}

		// Fleeing or dying: Clear TempLoot (loot lost)
		player.TempLoot = models.TempLoot{
			GuInventory:        models.GuInventory{},
			MaterialsInventory: models.MaterialsInventory{},
			PrimevalStones:     0,
		}

		// Reset position to safe zone
		player.CurrentRegion = "Southern_Border"
		player.X = 50
		player.Y = 50
		player.ExploredCells = models.ExploredCellsList{"50,50"}

		// Check if it was a tribulation fight
		if player.ActiveCombat != nil && player.ActiveCombat.EnemyTemplateID == "monster_heavenly_tribulation" {
			player.ImmortalStones = player.ImmortalStones * 0.5
			player.TribulationCount = 0
			nextTrib := time.Now().Add(10 * time.Minute)
			player.NextTribulationTime = &nextTrib

			// Thất bại độ kiếp nhưng sống: +200 human_qi
			player.HumanQi += 200.0
			if player.HumanQi > 100000.0 {
				player.HumanQi = 100000.0
			}
			ctx.Logs = append(ctx.Logs, "Độ kiếp thất bại nhưng may mắn giữ được mạng sống! Nhân khí +200. Tiên Nguyên Thạch bị hao tổn 50% và reset số lần độ kiếp.")
		}

		for _, ally := range player.ActiveCombat.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil {
				if allyUUID != player.ID {
					_, _ = tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", allyUUID)
				}
			}
		}
		player.ActiveCombat = nil
	} else {
		// Serialize and save updated combat state to all allies
		combatBytes, _ := json.Marshal(player.ActiveCombat)
		for _, ally := range player.ActiveCombat.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil && allyUUID != player.ID {
				_, _ = tx.Exec("UPDATE players SET active_combat = $1 WHERE id = $2", combatBytes, allyUUID)
			}
		}
	}

	// 8. Save Player state to DB
	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, primeval_stones = $3, materials_inventory = $4, gu_inventory = $5, cleared_cells = $6, active_combat = $7,
		    temp_loot = $8, current_region = $9, x = $10, y = $11, explored_cells = $12
		WHERE id = $13
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.PrimevalStones, player.MaterialsInventory, player.GuInventory, player.ClearedCells, player.ActiveCombat,
		player.TempLoot, player.CurrentRegion, player.X, player.Y, player.ExploredCells, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update combat results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"logs":    ctx.Logs,
		"enemy":   player.ActiveCombat,
		"victory": victory,
		"defeat":  defeat,
	})
}

// ExploreNode handles Phase 5 world exploration map nodes PVE encounters
func ExploreNode(c *gin.Context) {
	var req ExploreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get auth player ID
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// 1. Lock Player Row
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	// 2. Validate state
	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong chiến đấu! Hãy dứt điểm kẻ địch trước."})
		return
	}

	// Enforce daily entry limit if starting a new exploration run
	if player.CurrentRegion != req.NodeID {
		if errLimit := checkAndIncrementExploreLimit(tx, &player); errLimit != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errLimit.Error()})
			return
		}
	}

	// Deduct exploration energy. Mortals spend Chân Nguyên; Immortals spend a small Tiên Nguyên upkeep.
	exploreEnergyCost := 5.0
	if isImmortalCultivator(&player) {
		exploreEnergyCost = 0.25
	}
	if player.CurrentMana < exploreEnergyCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	player.CurrentMana -= exploreEnergyCost

	// Resolve map node name
	nodeName := "Khu Vực Bí Ẩn"
	switch req.NodeID {
	case "qing_mao_mountain":
		nodeName = "Thanh Mao Sơn (Southern Border)"
	case "gu_luan_slope":
		nodeName = "Cổ Loạn Sườn (Southern Border)"
	case "three_kings_inheritance":
		nodeName = "Tam Vương Truyền Thừa (Southern Border)"
	}

	// 3. Roll Random Event
	roll := rand.Intn(100)
	var logMsg string
	combatTriggered := false

	if roll < 40 {
		// 40% chance: Stones (1-3)
		stonesFound := 1 + rand.Intn(3)
		player.TempLoot.PrimevalStones += stonesFound
		logMsg = fmt.Sprintf("Khám phá hoàn tất tại %s. Đạo hữu may mắn phát hiện thấy %d viên Nguyên Thạch (được giữ tạm thời trong bản đồ)!", nodeName, stonesFound)
	} else if roll < 70 {
		// 30% chance: Combat
		combatTriggered = true

		// Spawn enemy based on DB-backed region monster tables.
		monsterID, err := db.PickRandomMonsterFromRegion(tx, req.NodeID)
		if err != nil || monsterID == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chọn quái vật từ cơ sở dữ liệu"})
			return
		}
		monTemplate, err := db.LoadMonsterTemplate(tx, monsterID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải quái vật từ cơ sở dữ liệu"})
			return
		}
		enemy := models.BuildCombatStateFromMonster(monTemplate)

		player.ActiveCombat = StartCombat(tx, &enemy, &player)
		logMsg = fmt.Sprintf("Nguy hiểm! Đột nhiên từ bụi cỏ rậm rạp tại %s, một con %s lao ra tấn công đạo hữu!", nodeName, enemy.EnemyName)
	} else if roll < 90 {
		// 20% chance: Materials
		foods := []string{"moon_orchid_petal", "green_bamboo_wine", "jade_stone", "boar_meat"}
		foodNames := []string{"Cánh hoa Nguyệt Lan", "Rượu Thanh Trúc", "Ngọc Thạch", "Thịt Lợn Rừng"}
		idx := rand.Intn(len(foods))
		foodID := foods[idx]
		foodName := foodNames[idx]

		if player.TempLoot.MaterialsInventory == nil {
			player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
		}
		player.TempLoot.MaterialsInventory[foodID]++
		logMsg = fmt.Sprintf("Đi qua khu rừng rậm ở %s, đạo hữu nhặt được nguyên liệu trân quý: 1x %s (được giữ tạm thời trong bản đồ).", nodeName, foodName)
	} else {
		// 10% chance: Nothing
		logMsg = fmt.Sprintf("Đạo hữu cẩn thận thăm dò địa khu %s nhưng cảnh vật hoang tàn, không tìm thấy thứ gì có giá trị.", nodeName)
	}

	if player.CurrentRegion != req.NodeID {
		player.ExploredCells = models.ExploredCellsList{"50,50"}
	}
	player.CurrentRegion = req.NodeID
	player.X = 50
	player.Y = 50
	addExploredCells(&player, 1)

	// 4. Update Database
	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, materials_inventory = $3, active_combat = $4, current_region = $5, x = $6, y = $7, explored_cells = $8,
		    temp_loot = $9, explore_count = $10, last_explore_reset = $11
		WHERE id = $12
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.MaterialsInventory, player.ActiveCombat, player.CurrentRegion, player.X, player.Y, player.ExploredCells,
		player.TempLoot, player.ExploreCount, player.LastExploreReset, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save exploration progress"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":           player,
		"log":              logMsg,
		"combat_triggered": combatTriggered,
	})
}

// Retreat handles escaping from active combat or exploration nodes
func Retreat(c *gin.Context) {
	playerIDStr := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	playerUUID, err := uuid.Parse(playerIDStr)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// Lock player row
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	logMsg := "Đạo hữu trở về an toàn từ hành trình."

	// Apply penalty if fleeing from active combat
	if player.ActiveCombat != nil {
		penaltyHP := 8.0
		penaltyStones := 10

		player.HP -= penaltyHP
		if player.HP < 1.0 {
			player.HP = 1.0 // Clamped at 1.0 to prevent dying
		}

		if player.PrimevalStones >= penaltyStones {
			player.PrimevalStones -= penaltyStones
		} else {
			player.PrimevalStones = 0
		}

		logMsg = fmt.Sprintf("Đạo hữu hốt hoảng tháo chạy khỏi %s! Bị phản phệ %.1f HP và mất %d Nguyên Thạch do vứt bỏ vật tư khi đào tẩu.",
			player.ActiveCombat.EnemyName, penaltyHP, penaltyStones)

		player.ActiveCombat = nil

		// Fleeing active combat: Clear TempLoot (loot lost)
		player.TempLoot = models.TempLoot{
			GuInventory:        models.GuInventory{},
			MaterialsInventory: models.MaterialsInventory{},
			PrimevalStones:     0,
		}
	} else {
		// Safe exit: Transfer TempLoot to permanent inventory, splitting it if group exploration
		var partyMembers []uuid.UUID
		if player.PartyID != nil {
			var membersRaw []byte
			errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
			if errPart == nil {
				var members []PartyMember
				json.Unmarshal(membersRaw, &members)
				for _, m := range members {
					mUUID, errParse := uuid.Parse(m.ID)
					if errParse == nil {
						var sameRegion bool
						errReg := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE id = $1 AND current_region = $2)", mUUID, player.CurrentRegion).Scan(&sameRegion)
						if errReg == nil && sameRegion {
							partyMembers = append(partyMembers, mUUID)
						}
					}
				}
			}
		}

		totalStones := player.TempLoot.PrimevalStones
		totalMats := player.TempLoot.MaterialsInventory
		totalGus := player.TempLoot.GuInventory

		if totalStones == 0 && len(totalMats) == 0 && len(totalGus) == 0 {
			logMsg = "Đạo hữu trở về an toàn từ hành trình. Không thu hoạch được tài nguyên nào."
		} else if len(partyMembers) > 1 {
			// Distribute loot among party members on the same map
			shares := make(map[uuid.UUID]*models.TempLoot)
			for _, mID := range partyMembers {
				shares[mID] = &models.TempLoot{
					GuInventory:        models.GuInventory{},
					MaterialsInventory: make(models.MaterialsInventory),
					PrimevalStones:     0,
				}
			}

			// 1. Split stones
			if totalStones > 0 {
				splitStones := totalStones / len(partyMembers)
				remainderStones := totalStones % len(partyMembers)
				for _, mID := range partyMembers {
					shares[mID].PrimevalStones = splitStones
				}
				shares[player.ID].PrimevalStones += remainderStones
			}

			// 2. Split materials
			for matID, qty := range totalMats {
				if qty > 0 {
					splitQty := qty / len(partyMembers)
					remainderQty := qty % len(partyMembers)
					for _, mID := range partyMembers {
						shares[mID].MaterialsInventory[matID] = splitQty
					}
					for i := 0; i < remainderQty; i++ {
						winnerID := partyMembers[rand.Intn(len(partyMembers))]
						shares[winnerID].MaterialsInventory[matID]++
					}
				}
			}

			// 3. Split Gus
			for _, gu := range totalGus {
				winnerID := partyMembers[rand.Intn(len(partyMembers))]
				shares[winnerID].GuInventory = append(shares[winnerID].GuInventory, gu)
			}

			// Apply active player shares to struct
			player.PrimevalStones += shares[player.ID].PrimevalStones
			if player.MaterialsInventory == nil {
				player.MaterialsInventory = make(models.MaterialsInventory)
			}
			for itemID, qty := range shares[player.ID].MaterialsInventory {
				player.MaterialsInventory[itemID] += qty
			}
			player.GuInventory = append(player.GuInventory, shares[player.ID].GuInventory...)

			// Apply other members' shares to DB
			for mID, share := range shares {
				if mID == player.ID {
					continue
				}
				var targetInvBytes, targetGuBytes []byte
				var targetStones int
				errLoad := tx.QueryRow(`
					SELECT primeval_stones, materials_inventory, gu_inventory 
					FROM players WHERE id = $1 FOR UPDATE`, mID,
				).Scan(&targetStones, &targetInvBytes, &targetGuBytes)
				if errLoad == nil {
					var targetInv models.MaterialsInventory
					var targetGu models.GuInventory
					if len(targetInvBytes) > 0 {
						json.Unmarshal(targetInvBytes, &targetInv)
					}
					if targetInv == nil {
						targetInv = make(models.MaterialsInventory)
					}
					if len(targetGuBytes) > 0 {
						json.Unmarshal(targetGuBytes, &targetGu)
					}

					targetStones += share.PrimevalStones
					for itemID, qty := range share.MaterialsInventory {
						targetInv[itemID] += qty
					}
					targetGu = append(targetGu, share.GuInventory...)

					_, _ = tx.Exec(`
						UPDATE players 
						SET primeval_stones = $1, materials_inventory = $2, gu_inventory = $3 
						WHERE id = $4`, targetStones, targetInv, targetGu, mID)
				}
			}

			// Format detailed group harvest report
			logMsg = "Đạo hữu trở về an toàn từ hành trình khám phá nhóm.\n"
			logMsg += "=== TỔNG THU HOẠCH TỔ ĐỘI ===\n"
			if totalStones > 0 {
				logMsg += fmt.Sprintf("- Nguyên Thạch: %d\n", totalStones)
			}
			for itemID, qty := range totalMats {
				if qty > 0 {
					logMsg += fmt.Sprintf("- %s: %d\n", itemID, qty)
				}
			}
			for _, gu := range totalGus {
				logMsg += fmt.Sprintf("- Cổ trùng: %s\n", gu.GuID)
			}

			logMsg += "\n=== PHÂN CHIA CHI TIẾT ===\n"
			for _, mID := range partyMembers {
				var uname string
				if mID == player.ID {
					uname = "Bạn"
				} else {
					_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", mID).Scan(&uname)
				}

				share := shares[mID]
				logMsg += fmt.Sprintf("[%s] nhận được:\n", uname)
				hasLoot := false
				if share.PrimevalStones > 0 {
					logMsg += fmt.Sprintf("  + %d Nguyên Thạch\n", share.PrimevalStones)
					hasLoot = true
				}
				for itemID, qty := range share.MaterialsInventory {
					if qty > 0 {
						logMsg += fmt.Sprintf("  + %dx %s\n", qty, itemID)
						hasLoot = true
					}
				}
				for _, gu := range share.GuInventory {
					logMsg += fmt.Sprintf("  + 1x Cổ trùng %s\n", gu.GuID)
					hasLoot = true
				}
				if !hasLoot {
					logMsg += "  (Không nhận được tài nguyên nào)\n"
				}
			}
		} else {
			// Individual safe exit: transfer all loot to player
			TransferTempLootToInventory(&player)

			// Format detailed individual harvest report
			logMsg = "Đạo hữu trở về an toàn từ hành trình.\n"
			logMsg += "=== TÀI NGUYÊN THU HOẠCH ===\n"
			if totalStones > 0 {
				logMsg += fmt.Sprintf("- Nguyên Thạch: +%d\n", totalStones)
			}
			for itemID, qty := range totalMats {
				if qty > 0 {
					logMsg += fmt.Sprintf("- %s: +%d\n", itemID, qty)
				}
			}
			for _, gu := range totalGus {
				logMsg += fmt.Sprintf("- Cổ trùng: %s\n", gu.GuID)
			}
		}

		// Clear TempLoot after transferring/splitting
		player.TempLoot = models.TempLoot{
			GuInventory:        models.GuInventory{},
			MaterialsInventory: models.MaterialsInventory{},
			PrimevalStones:     0,
		}
	}

	// Reset coordinates and exploration map on retreat
	player.X = 50
	player.Y = 50
	player.ExploredCells = models.ExploredCellsList{"50,50"}
	player.CurrentRegion = "Southern_Border"

	// Save player status
	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, active_combat = $3, x = $4, y = $5, explored_cells = $6, primeval_stones = $7,
		    gu_inventory = $8, materials_inventory = $9, temp_loot = $10, current_region = $11
		WHERE id = $12
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.ActiveCombat, player.X, player.Y, player.ExploredCells, player.PrimevalStones,
		player.GuInventory, player.MaterialsInventory, player.TempLoot, player.CurrentRegion, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player status after retreat"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player": player,
		"log":    logMsg,
	})
}

// RolledLoot represents the items and Gus rolled from a loot table
type RolledLoot struct {
	Items map[string]int
	Gus   []string
}

// RollLoot rolls items and Gus from a loot table according to entry chances
func RollLoot(lt models.LootTable) RolledLoot {
	rolled := RolledLoot{
		Items: make(map[string]int),
		Gus:   []string{},
	}
	for _, entry := range lt.Entries {
		if rand.Float64() <= entry.Chance {
			qty := 1
			if entry.MaxQty > entry.MinQty && entry.MinQty > 0 {
				qty = entry.MinQty + rand.Intn(entry.MaxQty-entry.MinQty+1)
			} else if entry.MinQty > 0 {
				qty = entry.MinQty
			}
			if entry.ItemID != "" {
				rolled.Items[entry.ItemID] += qty
			}
			if entry.GuID != "" {
				for i := 0; i < qty; i++ {
					rolled.Gus = append(rolled.Gus, entry.GuID)
				}
			}
		}
	}
	return rolled
}

// ApplyLoot applies the rolled loot to the player's inventory and returns logs
func ApplyLoot(player *models.Player, loot RolledLoot) []string {
	isExplore := player.CurrentRegion == "qing_mao_mountain" || player.CurrentRegion == "gu_luan_slope" || player.CurrentRegion == "three_kings_inheritance"
	var logs []string
	if !isExplore {
		if player.MaterialsInventory == nil {
			player.MaterialsInventory = make(models.MaterialsInventory)
		}
	} else {
		if player.TempLoot.MaterialsInventory == nil {
			player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
		}
	}

	for itemID, qty := range loot.Items {
		if itemID == "primeval_stone" {
			if !isExplore {
				player.PrimevalStones += qty
				logs = append(logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch.", qty))
			} else {
				player.TempLoot.PrimevalStones += qty
				logs = append(logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch (giữ tạm thời).", qty))
			}
		} else {
			name := itemID
			switch itemID {
			case "boar_meat":
				name = "Thịt Lợn Rừng"
			case "wild_boar_tusk":
				name = "Nanh Lợn Rừng"
			case "jade_stone":
				name = "Ngọc Thạch"
			case "moon_orchid_petal":
				name = "Cánh hoa Nguyệt Lan"
			case "green_bamboo_wine":
				name = "Rượu Thanh Trúc"
			}
			if !isExplore {
				player.MaterialsInventory[itemID] += qty
				logs = append(logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm thú dữ: %dx %s.", qty, name))
			} else {
				player.TempLoot.MaterialsInventory[itemID] += qty
				logs = append(logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm thú dữ: %dx %s (giữ tạm thời).", qty, name))
			}
		}
	}

	for _, guID := range loot.Gus {
		instID := "gu-" + uuid.New().String()
		newInstance := models.GuInstance{
			InstanceID: instID,
			GuID:       guID,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
		}
		guName := guID
		if spec, ok := models.GuRegistry[guID]; ok {
			guName = spec.Name
		}
		if !isExplore {
			player.GuInventory = append(player.GuInventory, newInstance)
			logs = append(logs, fmt.Sprintf("Chúc mừng đạo hữu thu phục được hoang dã Cổ trùng: [%s]!", guName))
		} else {
			player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, newInstance)
			logs = append(logs, fmt.Sprintf("Chúc mừng đạo hữu thu phục được hoang dã Cổ trùng: [%s] (giữ tạm thời)!", guName))
		}
	}
	return logs
}

// BuildEnemyActions compiles all active combat actions for the enemy based on their inventory and AI profile
func BuildEnemyActions(ctx *CombatContext, enemy *models.Combatant) []Action {
	var potentialActions []Action
	for _, eg := range enemy.GuInventory {
		if eg.Status == "ACTIVE" {
			spec, ok := models.GuRegistry[eg.GuID]
			if ok {
				action, err := BuildCombatAction(ctx, spec)
				if err == nil {
					potentialActions = append(potentialActions, action)
				}
			}
		}
	}

	selected := SelectActionsByAI(enemy.AIProfile, potentialActions, enemy.CurrentMana)

	if len(selected) == 0 {
		return BuildBasicEnemyAction(enemy)
	}

	return selected
}

// BuildBasicEnemyAction constructs a basic attack action for the enemy
func BuildBasicEnemyAction(enemy *models.Combatant) []Action {
	dmg := enemy.BaseAttack
	if dmg <= 0 {
		dmg = 5.0
	}
	name := "Đấm Tay Không (Basic Attack)"
	if enemy.Kind != "Player" {
		name = fmt.Sprintf("%s tấn công (Basic Attack)", enemy.Name)
	}
	return []Action{
		{
			Name:     name,
			Dmg:      dmg,
			ManaCost: 0,
			Logs:     []string{fmt.Sprintf("-> %s tung đòn đánh thường!", enemy.Name)},
		},
	}
}

// SelectActionsByAI selects enemy combat actions based on their AI profile and remaining mana
func SelectActionsByAI(profile string, potentialActions []Action, currentMana float64) []Action {
	if len(potentialActions) == 0 {
		return nil
	}

	var selected []Action
	remainingMana := currentMana

	switch profile {
	case "boss_burst":
		// Sort by dmg descending
		actions := make([]Action, len(potentialActions))
		copy(actions, potentialActions)
		for i := 0; i < len(actions)-1; i++ {
			for j := i + 1; j < len(actions); j++ {
				if actions[i].Dmg < actions[j].Dmg {
					actions[i], actions[j] = actions[j], actions[i]
				}
			}
		}
		count := 0
		for _, act := range actions {
			if count >= 2 {
				break
			}
			if act.ManaCost <= remainingMana {
				selected = append(selected, act)
				remainingMana -= act.ManaCost
				count++
			}
		}

	case "defensive":
		// Sort by defense descending
		actions := make([]Action, len(potentialActions))
		copy(actions, potentialActions)
		for i := 0; i < len(actions)-1; i++ {
			for j := i + 1; j < len(actions); j++ {
				if actions[i].Def < actions[j].Def {
					actions[i], actions[j] = actions[j], actions[i]
				}
			}
		}
		count := 0
		for _, act := range actions {
			if count >= 2 {
				break
			}
			if act.ManaCost <= remainingMana {
				selected = append(selected, act)
				remainingMana -= act.ManaCost
				count++
			}
		}

	case "boss_balanced":
		var attackActions []Action
		var defenseActions []Action
		for _, act := range potentialActions {
			if act.Dmg > 0 {
				attackActions = append(attackActions, act)
			}
			if act.Def > 0 {
				defenseActions = append(defenseActions, act)
			}
		}

		for i := 0; i < len(attackActions)-1; i++ {
			for j := i + 1; j < len(attackActions); j++ {
				if attackActions[i].Dmg < attackActions[j].Dmg {
					attackActions[i], attackActions[j] = attackActions[j], attackActions[i]
				}
			}
		}
		for i := 0; i < len(defenseActions)-1; i++ {
			for j := i + 1; j < len(defenseActions); j++ {
				if defenseActions[i].Def < defenseActions[j].Def {
					defenseActions[i], defenseActions[j] = defenseActions[j], defenseActions[i]
				}
			}
		}

		if len(defenseActions) > 0 && defenseActions[0].ManaCost <= remainingMana {
			selected = append(selected, defenseActions[0])
			remainingMana -= defenseActions[0].ManaCost
		}
		if len(attackActions) > 0 && attackActions[0].ManaCost <= remainingMana {
			alreadyPicked := false
			for _, s := range selected {
				if s.Name == attackActions[0].Name {
					alreadyPicked = true
				}
			}
			if !alreadyPicked {
				selected = append(selected, attackActions[0])
				remainingMana -= attackActions[0].ManaCost
			}
		}
		if len(selected) == 0 {
			for _, act := range potentialActions {
				if act.ManaCost <= remainingMana {
					selected = append(selected, act)
					break
				}
			}
		}

	default: // beast_basic or normal behaviour
		for _, act := range potentialActions {
			if act.ManaCost <= remainingMana {
				selected = append(selected, act)
				remainingMana -= act.ManaCost
				break
			}
		}
	}

	return selected
}

// TransferTempLootToInventory moves all temporary loot items, Gu, and stones into the player's permanent inventories.
func TransferTempLootToInventory(player *models.Player) {
	if player.MaterialsInventory == nil {
		player.MaterialsInventory = make(models.MaterialsInventory)
	}
	for itemID, qty := range player.TempLoot.MaterialsInventory {
		player.MaterialsInventory[itemID] += qty
	}
	player.PrimevalStones += player.TempLoot.PrimevalStones
	player.GuInventory = append(player.GuInventory, player.TempLoot.GuInventory...)
}

// checkAndIncrementExploreLimit resets and checks daily limits for player and party members
func checkAndIncrementExploreLimit(tx *sql.Tx, player *models.Player) error {
	now := time.Now()

	resetIfNeeded := func(pCount *int, pReset *time.Time, pID uuid.UUID) error {
		y1, m1, d1 := now.Date()
		y2, m2, d2 := pReset.Date()
		if y1 != y2 || m1 != m2 || d1 != d2 {
			*pCount = 0
			*pReset = now
			_, err := tx.Exec("UPDATE players SET explore_count = 0, last_explore_reset = $1 WHERE id = $2", now, pID)
			return err
		}
		return nil
	}

	// Check/reset player themselves
	if err := resetIfNeeded(&player.ExploreCount, &player.LastExploreReset, player.ID); err != nil {
		return err
	}
	if player.ExploreCount >= 5 {
		return fmt.Errorf("Đạo hữu đã hết lượt khám phá hôm nay (tối đa 5 lượt/ngày).")
	}

	// Check/reset party members if in a party
	if player.PartyID != nil {
		var membersRaw []byte
		errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
		if errPart == nil {
			var members []PartyMember
			json.Unmarshal(membersRaw, &members)
			for _, m := range members {
				mUUID, errParse := uuid.Parse(m.ID)
				if errParse == nil && mUUID != player.ID {
					var tCount int
					var tReset time.Time
					errLoad := tx.QueryRow("SELECT explore_count, last_explore_reset FROM players WHERE id = $1 FOR UPDATE", mUUID).Scan(&tCount, &tReset)
					if errLoad == nil {
						y1, m1, d1 := now.Date()
						y2, m2, d2 := tReset.Date()
						if y1 != y2 || m1 != m2 || d1 != d2 {
							tCount = 0
							tReset = now
							_, errUpd := tx.Exec("UPDATE players SET explore_count = 0, last_explore_reset = $1 WHERE id = $2", now, mUUID)
							if errUpd != nil {
								return errUpd
							}
						}
						if tCount >= 5 {
							return fmt.Errorf("Thành viên trong tổ đội (%s) đã hết lượt khám phá hôm nay (tối đa 5 lượt/ngày).", m.Username)
						}
					}
				}
			}

			// All checked out, increment count for other party members in database
			for _, m := range members {
				mUUID, errParse := uuid.Parse(m.ID)
				if errParse == nil && mUUID != player.ID {
					_, errInc := tx.Exec("UPDATE players SET explore_count = explore_count + 1 WHERE id = $1", mUUID)
					if errInc != nil {
						return errInc
					}
				}
			}
		}
	}

	// Increment for active player
	player.ExploreCount++
	player.LastExploreReset = now
	return nil
}

func IntegratePassiveGus(player *models.Player, logs *[]string) {
	newInventory := models.GuInventory{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			spec, ok := models.GuRegistry[gu.GuID]
			if ok && spec.Type == "Passive" {
				gu.IntegrationTime += 1.0
				if gu.IntegrationTime >= 5.0 {
					// Integrate permanently!
					var modName string
					var bonusText string
					switch gu.GuID {
					case "GU_JADE_SKIN":
						modName = "Jade Skin Mod"
						bonusText = "+40 Phòng thủ"
					case "GU_WHITE_BOAR":
						modName = "White Boar Strength"
						bonusText = "+15 Tấn công"
					case "GU_BLACK_BOAR":
						modName = "Black Boar Strength"
						bonusText = "+15 Tấn công"
					case "GU_COPPER_SKIN":
						modName = "Copper Skin Mod"
						bonusText = "+20 Phòng thủ"
					case "GU_IRON_BONE":
						modName = "Iron Bone"
						bonusText = "+100 Phòng thủ"
					default:
						modName = gu.GuID + " Integrated"
						bonusText = "các chỉ số bổ trợ"
					}

					// Add to player BodyMods if not already present
					alreadyExists := false
					for _, existingMod := range player.BodyMods {
						if existingMod == modName {
							alreadyExists = true
							break
						}
					}
					if !alreadyExists {
						player.BodyMods = append(player.BodyMods, modName)
					}

					*logs = append(*logs, fmt.Sprintf("✨ Thể chất cải tạo hoàn tất: Cổ trùng %s đã hoàn toàn dung nhập vào da thịt, khắc sâu Đạo Ngân lên cơ thể đạo hữu. Cổ trùng tiêu biến, đạo hữu nhận được vĩnh viễn %s!", spec.Name, bonusText))
					continue // Skip adding to newInventory, meaning the Gu is removed/consumed vĩnh viễn!
				}
			}
		}
		newInventory = append(newInventory, gu)
	}
	player.GuInventory = newInventory
	RebuildEquippedGus(player)
}
