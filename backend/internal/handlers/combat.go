package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
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
		if mLevel, ok := actor.PathMasteries[path]; ok {
			switch mLevel {
			case "Ordinary":
				masteryMultiplier = 1.0
			case "Master":
				masteryMultiplier = 1.1
			case "Grandmaster":
				masteryMultiplier = 1.25
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

	// Extract base damage and base defense
	var baseDmg float64
	if dmg, ok := guSpec.Effects["damage"]; ok {
		baseDmg = dmg
	} else if dmg, ok := guSpec.Effects["add_base_attack"]; ok {
		baseDmg = dmg
	}

	var baseDef float64
	if def, ok := guSpec.Effects["defense"]; ok {
		baseDef = def
	} else if def, ok := guSpec.Effects["add_base_defense"]; ok {
		baseDef = def
	}

	masteryMultiplier := getActorPathMasteryMultiplier(actor, guSpec.Path)

	effectiveDamage = baseDmg * powerScale * masteryMultiplier
	effectiveDefense = baseDef * powerScale * masteryMultiplier

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
	if base.EnemyType == "PVP" {
		pHP := player.BaseHP
		if pHP <= 0 {
			pHP = 100.0
		}
		playerCombatant.HP = pHP
	}
	base.Allies = []models.Combatant{playerCombatant}

	enemyCombatant := models.CombatantFromActiveCombat(base)
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

	isChargingThisTurn := false
	isUnleashingCharge := false

	if req.IsCharge {
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

	var turnLogs []string
	turnLogs = append(turnLogs, "=== KÍCH HOẠT SÁT CHIÊU ===")

	// 4. Execute Caster (Player) actions
	if targetIdx != -1 {
		targetEnemy := &player.ActiveCombat.Enemies[targetIdx]
		casterCombatant := &player.ActiveCombat.Allies[casterIdx]

		if isChargingThisTurn {
			turnLogs = append(turnLogs, fmt.Sprintf("%s tích tụ Nguyên Khí, vận chuyển Chân Nguyên chuẩn bị cho Đại Sát Chiêu!", casterCombatant.Name))
		} else if isUnleashingCharge {
			dmgValue := playerStats["attack"] * 2.5
			netDmg := dmgValue - targetEnemy.BaseDefense
			if netDmg < 1.0 {
				netDmg = 1.0
			}
			targetEnemy.HP -= netDmg
			turnLogs = append(turnLogs, fmt.Sprintf("-> %s kích hoạt Bộc Phá Sát Chiêu (Unleash Charge) gây %.1f sát thương lên %s!", casterCombatant.Name, netDmg, targetEnemy.Name))
		} else {
			for _, spec := range selectedSpecs {
				dmg, def, cost, errGu := CalculateGuEffectForCombatant(casterCombatant, &spec)
				if errGu == nil && casterCombatant.CurrentMana >= cost {
					casterCombatant.CurrentMana -= cost

					isAOE := false
					if spec.Effects != nil && (spec.Effects["aoe"] > 0 || spec.Effects["damage_multiplier"] > 0) {
						isAOE = true
					}
					isHealing := false
					if spec.Effects != nil && (spec.Effects["healing"] > 0 || spec.Effects["restore_hp"] > 0) {
						isHealing = true
					}

					if isHealing {
						healVal := def
						if healVal <= 0 {
							healVal = 30.0
						}
						casterCombatant.HP += healVal
						if casterCombatant.HP > casterCombatant.MaxHP {
							casterCombatant.HP = casterCombatant.MaxHP
						}
						turnLogs = append(turnLogs, fmt.Sprintf("-> %s kích hoạt %s, khôi phục %.1f HP bản thân.", casterCombatant.Name, spec.Name, healVal))
					} else if isAOE {
						turnLogs = append(turnLogs, fmt.Sprintf("-> %s thi triển Sát Chiêu AOE %s quét sạch phe địch!", casterCombatant.Name, spec.Name))
						for eIdx := range player.ActiveCombat.Enemies {
							if player.ActiveCombat.Enemies[eIdx].HP > 0 {
								netDmg := dmg - player.ActiveCombat.Enemies[eIdx].BaseDefense
								if netDmg < 1.0 {
									netDmg = 1.0
								}
								player.ActiveCombat.Enemies[eIdx].HP -= netDmg
								turnLogs = append(turnLogs, fmt.Sprintf("   -> Gây %.1f sát thương lên %s.", netDmg, player.ActiveCombat.Enemies[eIdx].Name))
							}
						}
					} else {
						netDmg := dmg - targetEnemy.BaseDefense
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						targetEnemy.HP -= netDmg
						turnLogs = append(turnLogs, fmt.Sprintf("-> %s kích hoạt %s đánh trúng %s gây %.1f sát thương.", casterCombatant.Name, spec.Name, targetEnemy.Name, netDmg))
					}
				}
			}
			if len(selectedSpecs) == 0 {
				netDmg := 5.0 - targetEnemy.BaseDefense
				if netDmg < 1.0 {
					netDmg = 1.0
				}
				targetEnemy.HP -= netDmg
				turnLogs = append(turnLogs, fmt.Sprintf("-> %s đánh thường (Basic Attack) lên %s gây %.1f sát thương.", casterCombatant.Name, targetEnemy.Name, netDmg))
			}
		}
	}

	// 5. Execute Other Allies actions
	for aIdx := range player.ActiveCombat.Allies {
		ally := &player.ActiveCombat.Allies[aIdx]
		if ally.ID != player.ID.String() && ally.HP > 0 {
			var allyTarget *models.Combatant
			for eIdx := range player.ActiveCombat.Enemies {
				if player.ActiveCombat.Enemies[eIdx].HP > 0 {
					allyTarget = &player.ActiveCombat.Enemies[eIdx]
					break
				}
			}
			if allyTarget != nil {
				dmgVal := ally.BaseAttack
				if dmgVal <= 0 {
					dmgVal = 8.0
				}
				netDmg := dmgVal - allyTarget.BaseDefense
				if netDmg < 1.0 {
					netDmg = 1.0
				}
				allyTarget.HP -= netDmg
				turnLogs = append(turnLogs, fmt.Sprintf("-> Đồng minh %s đánh thường lên %s gây %.1f sát thương.", ally.Name, allyTarget.Name, netDmg))
			}
		}
	}

	// 6. Execute Enemy turns
	for eIdx := range player.ActiveCombat.Enemies {
		enemy := &player.ActiveCombat.Enemies[eIdx]
		if enemy.HP > 0 {
			var aliveAllies []*models.Combatant
			for aIdx := range player.ActiveCombat.Allies {
				if player.ActiveCombat.Allies[aIdx].HP > 0 {
					aliveAllies = append(aliveAllies, &player.ActiveCombat.Allies[aIdx])
				}
			}

			if len(aliveAllies) > 0 {
				targetAlly := aliveAllies[rand.Intn(len(aliveAllies))]
				enemyCtx := &CombatContext{
					Attacker: enemy,
					Defender: targetAlly,
					Logs:     []string{},
				}
				enemyActions := BuildEnemyActions(enemyCtx, enemy)
				if len(enemyActions) > 0 {
					act := enemyActions[0]
					netDmg := act.Dmg - targetAlly.BaseDefense
					if netDmg < 1.0 {
						netDmg = 1.0
					}
					if act.ManaCost > 0 && enemy.CurrentMana >= act.ManaCost {
						enemy.CurrentMana -= act.ManaCost
					}
					targetAlly.HP -= netDmg
					turnLogs = append(turnLogs, fmt.Sprintf("-> %s kích hoạt %s tấn công %s gây %.1f sát thương.", enemy.Name, act.Name, targetAlly.Name, netDmg))
				} else {
					dmgVal := enemy.BaseAttack
					if dmgVal <= 0 {
						dmgVal = 5.0
					}
					netDmg := dmgVal - targetAlly.BaseDefense
					if netDmg < 1.0 {
						netDmg = 1.0
					}
					targetAlly.HP -= netDmg
					turnLogs = append(turnLogs, fmt.Sprintf("-> %s đánh thường lên %s gây %.1f sát thương.", enemy.Name, targetAlly.Name, netDmg))
				}
			}
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
					err = tx.QueryRow("SELECT primeval_stones, hp FROM players WHERE id = $1 FOR UPDATE", targetUUID).Scan(&targetStones, &targetHP)
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
						_, _ = tx.Exec("UPDATE players SET primeval_stones = $1, hp = $2 WHERE id = $3", targetStones, targetHP, targetUUID)
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
			if player.ActiveCombat != nil && player.ActiveCombat.LootTableID != "" {
				lt, err := db.LoadLootTable(tx, player.ActiveCombat.LootTableID)
				if err == nil {
					loot := RollLoot(lt)
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
									player.PrimevalStones += splitStones
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch.", splitStones))
								} else {
									_, _ = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", splitStones, memberID)
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
									if player.MaterialsInventory == nil {
										player.MaterialsInventory = make(models.MaterialsInventory)
									}
									player.MaterialsInventory[matID]++
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhặt được 1x %s.", matID))
								} else {
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)

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
								player.GuInventory = append(player.GuInventory, newGu)
								ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn bắt được 1x %s!", gID))
							} else {
								var uname string
								_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)

								var targetGuInv models.GuInventory
								var targetGuInvBytes []byte
								_ = tx.QueryRow("SELECT gu_inventory FROM players WHERE id = $1 FOR UPDATE", winnerID).Scan(&targetGuInvBytes)
								if len(targetGuInvBytes) > 0 {
									json.Unmarshal(targetGuInvBytes, &targetGuInv)
								}
								targetGuInv = append(targetGuInv, newGu)
								_, _ = tx.Exec("UPDATE players SET gu_inventory = $1 WHERE id = $2", targetGuInv, winnerID)
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
				rewardStones := 5 + rand.Intn(11)
				if len(partyMembers) > 1 {
					splitStones := rewardStones / len(partyMembers)
					if splitStones < 1 {
						splitStones = 1
					}
					for _, memberID := range partyMembers {
						if memberID == player.ID {
							player.PrimevalStones += splitStones
							ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch.", splitStones))
						} else {
							_, _ = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", splitStones, memberID)
							var uname string
							_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", memberID).Scan(&uname)
							ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đồng đội %s nhận được %d Nguyên Thạch.", uname, splitStones))
						}
					}
				} else {
					player.PrimevalStones += rewardStones
					ctx.Logs = append(ctx.Logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch.", rewardStones))
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
		player.HP = 0
		defeat = true
		ctx.Logs = append(ctx.Logs, "Đại bại! Đạo hữu bất tỉnh nhân sự trước áp lực của đối thủ.")

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
		SET hp = $1, current_mana = $2, primeval_stones = $3, materials_inventory = $4, gu_inventory = $5, cleared_cells = $6, active_combat = $7
		WHERE id = $8
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.PrimevalStones, player.MaterialsInventory, player.GuInventory, player.ClearedCells, player.ActiveCombat, player.ID)
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
		player.PrimevalStones += stonesFound
		logMsg = fmt.Sprintf("Khám phá hoàn tất tại %s. Đạo hữu may mắn phát hiện thấy %d viên Nguyên Thạch!", nodeName, stonesFound)
	} else if roll < 70 {
		// 30% chance: Combat
		combatTriggered = true

		// Spawn enemy based on map node ID
		monsterID, err := db.PickRandomMonsterFromRegion(tx, req.NodeID)
		if err != nil {
			monsterID = "monster_wild_boar"
		}
		monTemplate, err := db.LoadMonsterTemplate(tx, monsterID)
		var enemy models.CombatState
		if err != nil {
			// fallback
			enemy = models.CombatState{
				EnemyTemplateID: "monster_wild_boar",
				EnemyName:       "Dã Trĩ (Wild Boar)",
				EnemyHP:         50,
				EnemyMaxHP:      50,
				EnemyDamage:     8,
				EnemyDefense:    2,
				EnemyType:       "Normal",
				EnemyMana:       100,
				EnemyMaxMana:    100,
				EnemyRank:       1,
				EnemySubRank:    1,
				AIProfile:       "beast_basic",
				LootTableID:     "loot_boar",
			}
		} else {
			enemy = models.BuildCombatStateFromMonster(monTemplate)
		}

		player.ActiveCombat = StartCombat(tx, &enemy, &player)
		logMsg = fmt.Sprintf("Nguy hiểm! Đột nhiên từ bụi cỏ rậm rạp tại %s, một con %s lao ra tấn công đạo hữu!", nodeName, enemy.EnemyName)
	} else if roll < 90 {
		// 20% chance: Materials
		foods := []string{"moon_orchid_petal", "green_bamboo_wine", "jade_stone", "boar_meat"}
		foodNames := []string{"Cánh hoa Nguyệt Lan", "Rượu Thanh Trúc", "Ngọc Thạch", "Thịt Lợn Rừng"}
		idx := rand.Intn(len(foods))
		foodID := foods[idx]
		foodName := foodNames[idx]

		player.MaterialsInventory[foodID]++
		logMsg = fmt.Sprintf("Đi qua khu rừng rậm ở %s, đạo hữu nhặt được nguyên liệu trân quý: 1x %s.", nodeName, foodName)
	} else {
		// 10% chance: Nothing
		logMsg = fmt.Sprintf("Đạo hữu cẩn thận thăm dò địa khu %s nhưng cảnh vật hoang tàn, không tìm thấy thứ gì có giá trị.", nodeName)
	}

	player.CurrentRegion = req.NodeID
	player.X = 50
	player.Y = 50
	addExploredCells(&player, 1)

	// 4. Update Database
	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, materials_inventory = $3, active_combat = $4, current_region = $5, x = $6, y = $7, explored_cells = $8
		WHERE id = $9
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.MaterialsInventory, player.ActiveCombat, player.CurrentRegion, player.X, player.Y, player.ExploredCells, player.ID)
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
	}

	// Save player status
	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, active_combat = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.ActiveCombat, player.ID)
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
	var logs []string
	if player.MaterialsInventory == nil {
		player.MaterialsInventory = make(models.MaterialsInventory)
	}

	for itemID, qty := range loot.Items {
		if itemID == "primeval_stone" {
			player.PrimevalStones += qty
			logs = append(logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch.", qty))
		} else {
			player.MaterialsInventory[itemID] += qty
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
			logs = append(logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm thú dữ: %dx %s.", qty, name))
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
		player.GuInventory = append(player.GuInventory, newInstance)

		guName := guID
		if spec, ok := models.GuRegistry[guID]; ok {
			guName = spec.Name
		}
		logs = append(logs, fmt.Sprintf("Chúc mừng đạo hữu thu phục được hoang dã Cổ trùng: [%s]!", guName))
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
