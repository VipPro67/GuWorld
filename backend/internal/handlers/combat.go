package handlers

import (
	"fmt"
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
	Attacker    *models.Player
	Defender    *models.CombatState
	DamageDealt float64
	ShieldAdded float64
	DamageBoost float64 // Multiplier from support Gus (e.g. 0.20 = +20%)
	Logs        []string
}

// GuAbility strategy pattern interface
type GuAbility interface {
	Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error
}

// MoonlightAbility deals damage
type MoonlightAbility struct{}
func (m *MoonlightAbility) Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error {
	dmg, _, cost, err := CalculateGuEffect(ctx.Attacker, guSpec)
	if err != nil {
		ctx.Logs = append(ctx.Logs, err.Error())
		return err
	}
	ctx.DamageDealt += dmg
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Triển khai %s! Tiêu hao %.1f Chân Nguyên, gây %.1f sát thương.", guSpec.Name, cost, dmg))
	return nil
}

// JadeSkinAbility adds shields/defense
type JadeSkinAbility struct{}
func (j *JadeSkinAbility) Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error {
	_, def, cost, err := CalculateGuEffect(ctx.Attacker, guSpec)
	if err != nil {
		ctx.Logs = append(ctx.Logs, err.Error())
		return err
	}
	ctx.ShieldAdded += def
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kích hoạt %s! Tiêu hao %.1f Chân Nguyên, tăng %.1f phòng ngự da ngọc.", guSpec.Name, cost, def))
	return nil
}

// WhiteBoarAbility deals damage and adds shield
type WhiteBoarAbility struct{}
func (w *WhiteBoarAbility) Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error {
	dmg, def, cost, err := CalculateGuEffect(ctx.Attacker, guSpec)
	if err != nil {
		ctx.Logs = append(ctx.Logs, err.Error())
		return err
	}
	ctx.DamageDealt += dmg
	ctx.ShieldAdded += def
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Triển khai %s! Tiêu hao %.1f Chân Nguyên, gây %.1f sát thương và tăng %.1f phòng ngự lực đạo.", guSpec.Name, cost, dmg, def))
	return nil
}

// MoonRaimentAbility adds shields
type MoonRaimentAbility struct{}
func (mr *MoonRaimentAbility) Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error {
	_, def, cost, err := CalculateGuEffect(ctx.Attacker, guSpec)
	if err != nil {
		ctx.Logs = append(ctx.Logs, err.Error())
		return err
	}
	ctx.ShieldAdded += def
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kích hoạt %s! Tiêu hao %.1f Chân Nguyên, tăng %.1f phòng ngự xiêm y ánh trăng.", guSpec.Name, cost, def))
	return nil
}

// WhiteJadeAbility deals damage and adds massive shield
type WhiteJadeAbility struct{}
func (wj *WhiteJadeAbility) Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error {
	dmg, def, cost, err := CalculateGuEffect(ctx.Attacker, guSpec)
	if err != nil {
		ctx.Logs = append(ctx.Logs, err.Error())
		return err
	}
	ctx.DamageDealt += dmg
	ctx.ShieldAdded += def
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Triển khai %s! Tiêu hao %.1f Chân Nguyên, gây %.1f sát thương và phủ bộ da ngọc trắng phòng ngự %.1f.", guSpec.Name, cost, dmg, def))
	return nil
}

// LiquorWormAbility boosts damage
type LiquorWormAbility struct{}
func (l *LiquorWormAbility) Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error {
	_, _, cost, err := CalculateGuEffect(ctx.Attacker, guSpec)
	if err != nil {
		ctx.Logs = append(ctx.Logs, err.Error())
		return err
	}
	ctx.DamageBoost += 0.20 // 20% damage boost
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kích hoạt %s! Tiêu hao %.1f Chân Nguyên, tinh luyện chất lượng chân nguyên (+20%% Sát thương).", guSpec.Name, cost))
	return nil
}

// FourFlavorsLiquorWormAbility boosts damage even more
type FourFlavorsLiquorWormAbility struct{}
func (f *FourFlavorsLiquorWormAbility) Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error {
	_, _, cost, err := CalculateGuEffect(ctx.Attacker, guSpec)
	if err != nil {
		ctx.Logs = append(ctx.Logs, err.Error())
		return err
	}
	ctx.DamageBoost += 0.40 // 40% damage boost
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kích hoạt %s! Tiêu hao %.1f Chân Nguyên, kích phát chân nguyên chất lượng cực cao (+40%% Sát thương).", guSpec.Name, cost))
	return nil
}

// CleansingWaterAbility cleanses negative effects
type CleansingWaterAbility struct{}
func (cw *CleansingWaterAbility) Execute(ctx *CombatContext, guSpec *models.GuDictionaryItem) error {
	_, _, cost, err := CalculateGuEffect(ctx.Attacker, guSpec)
	if err != nil {
		ctx.Logs = append(ctx.Logs, err.Error())
		return err
	}
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kích hoạt %s! Tiêu hao %.1f Chân Nguyên, thanh tẩy dị lực và loại bỏ tạp chất trong Không khiếu.", guSpec.Name, cost))
	return nil
}

// AbilityRegistry maps Gu ID to its ability executor logic
var AbilityRegistry = map[string]GuAbility{
	"GU_MOONLIGHT":            &MoonlightAbility{},
	"GU_JADE_SKIN":            &JadeSkinAbility{},
	"GU_WHITE_BOAR":           &WhiteBoarAbility{},
	"GU_MOON_RAIMENT":         &MoonRaimentAbility{},
	"GU_WHITE_JADE":           &WhiteJadeAbility{},
	"GU_LIQUOR_WORM":          &LiquorWormAbility{},
	"GU_FOUR_FLAVORS_LIQUOR":  &FourFlavorsLiquorWormAbility{},
	"GU_BLACK_BOAR":           &WhiteBoarAbility{}, // Reuses WhiteBoarAbility strength mechanics
	"GU_CLEANSING_WATER":      &CleansingWaterAbility{},
}

// CalculateGuEffect computes lore-accurate adjustments for level penalty, regional suppression, and mana cost
func CalculateGuEffect(player *models.Player, guSpec *models.GuDictionaryItem) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	// Rule 1: Level Penalty (Backlash)
	if player.Rank < guSpec.Rank {
		player.HP -= 50
		if player.HP < 0 {
			player.HP = 0
		}
		return 0, 0, 0, fmt.Errorf("Backlash: Cổ trùng Rank %d phản phệ Không Khiếu Rank %d! (Trừ 50 HP)", guSpec.Rank, player.Rank)
	}

	// Rule 2: Regional Suppression
	effectiveRank := guSpec.Rank
	powerScale := 1.0
	if guSpec.OriginRegion != player.CurrentRegion {
		effectiveRank = guSpec.Rank - 1
		if effectiveRank < 1 {
			effectiveRank = 1
		}
		powerScale = float64(effectiveRank) / float64(guSpec.Rank)
	}

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

	// Extract player's mastery for guSpec.Path
	masteryMultiplier := 1.0
	if player.PathMasteries != nil {
		if mLevel, ok := player.PathMasteries[guSpec.Path]; ok {
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

	effectiveDamage = baseDmg * powerScale * masteryMultiplier
	effectiveDefense = baseDef * powerScale * masteryMultiplier

	// Rule 3: Mana cost calculation
	var baseManaPct float64
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	} else {
		baseManaPct = 0.10
	}
	cost := baseManaPct * player.MaxMana

	// If player.Rank > guSpec.Rank, divide by 10 for each rank difference
	if player.Rank > guSpec.Rank {
		rankDiff := player.Rank - guSpec.Rank
		for i := 0; i < rankDiff; i++ {
			cost /= 10.0
		}
	}

	// Sub-rank division: Divide by 2 for each sub-rank above initial (Sơ Giai)
	divisor := 1
	for i := 1; i < player.SubRank; i++ {
		divisor *= 2
	}
	cost /= float64(divisor)

	return effectiveDamage, effectiveDefense, cost, nil
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

	if player.HP <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã kiệt sức. Dưỡng thương trước khi chiến đấu."})
		return
	}

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
					selectedGus = append(selectedGus, inst)
					selectedSpecs = append(selectedSpecs, spec)

					// Calculate mana cost for validation
					_, _, cost, err := CalculateGuEffect(&player, &spec)
					if err == nil {
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

		if player.CurrentMana < totalManaCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Chân Nguyên để kích hoạt sát chiêu."})
			return
		}
	}

	// 4. Construct Combat Context
	ctx := &CombatContext{
		Attacker: &player,
		Defender: player.ActiveCombat,
		Logs:     []string{},
	}

	playerStats := CalculatePlayerStats(&player)

	// Define simultaneous action struct
	type Action struct {
		Name      string
		Dmg       float64
		Def       float64
		ManaCost  float64
		IsUtility bool
	}

	var playerActions []Action

	if isChargingThisTurn {
		ctx.Logs = append(ctx.Logs, "Đạo hữu tích tụ Nguyên Khí, vận chuyển Chân Nguyên chuẩn bị cho Đại Sát Chiêu!")
	} else if isUnleashingCharge {
		dmgValue := playerStats["attack"] * 2.5
		playerActions = append(playerActions, Action{
			Name: "Bộc Phá Sát Chiêu (Unleash Charge)",
			Dmg:  dmgValue,
		})
	} else {
		for _, spec := range selectedSpecs {
			dmg, def, cost, err := CalculateGuEffect(&player, &spec)
			if err != nil {
				continue
			}
			isUtil := spec.ID == "GU_LIQUOR_WORM" || spec.ID == "GU_FOUR_FLAVORS_LIQUOR" || spec.ID == "GU_CLEANSING_WATER"
			playerActions = append(playerActions, Action{
				Name:      spec.Name,
				Dmg:       dmg,
				Def:       def,
				ManaCost:  cost,
				IsUtility: isUtil,
			})
		}
		if len(playerActions) == 0 {
			playerActions = append(playerActions, Action{
				Name: "Đấm Tay Không (Basic Attack)",
				Dmg:  5.0,
			})
		}
	}

	// 5. Build Enemy Actions
	var enemyActions []Action
	enemyCurrentMana := 100.0
	enemyMaxMana := 100.0

	isPVP := player.ActiveCombat.TargetPlayerID != ""
	if isPVP && player.ActiveCombat.TargetPlayerID != "dummy" {
		targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
		if parseErr == nil {
			var enemyGuInventory models.GuInventory
			err = tx.QueryRow("SELECT gu_inventory, current_mana, max_mana FROM players WHERE id = $1", targetUUID).Scan(&enemyGuInventory, &enemyCurrentMana, &enemyMaxMana)
			if err == nil {
				count := 0
				for _, eg := range enemyGuInventory {
					if eg.Equipped && eg.Status == "ACTIVE" && count < 2 {
						spec, ok := models.GuRegistry[eg.GuID]
						if ok {
							dmg, def, cost, _ := CalculateGuEffect(&player, &spec)
							isUtil := spec.ID == "GU_LIQUOR_WORM" || spec.ID == "GU_FOUR_FLAVORS_LIQUOR" || spec.ID == "GU_CLEANSING_WATER"
							enemyActions = append(enemyActions, Action{
								Name:      spec.Name,
								Dmg:       dmg,
								Def:       def,
								ManaCost:  cost,
								IsUtility: isUtil,
							})
							count++
						}
					}
				}
			}
		}
	}

	if len(enemyActions) == 0 {
		enemyRoll := rand.Intn(100)
		if enemyRoll < 60 {
			attackName := "Cắn Xé"
			if isPVP {
				attackName = "Đòn Đánh Thường"
			} else {
				switch player.ActiveCombat.EnemyName {
				case "Mãng Xà Dung Nham (Lava Python)":
					attackName = "Dung Nham Phún Trào"
				case "Thiết Giáp Hùng (Iron Armored Bear)":
					attackName = "Hùng Chưởng"
				case "Độc Lang Vương (Wolf Leader)":
					attackName = "Thiên Lang Trảo"
				case "Dã Trĩ Vương (Boar King)":
					attackName = "Húc Sừng"
				case "Thạch Hầu (Stone Ape)":
					attackName = "Quăng Đá"
				case "Dã Trĩ (Wild Boar)":
					attackName = "Húc Đầu"
				}
			}
			enemyActions = append(enemyActions, Action{
				Name: attackName,
				Dmg:  player.ActiveCombat.EnemyDamage,
			})
		} else if enemyRoll < 85 {
			defName := "Phòng Thủ"
			if !isPVP {
				switch player.ActiveCombat.EnemyName {
				case "Mãng Xà Dung Nham (Lava Python)":
					defName = "Hỏa Khiên Hộ Thể"
				case "Thiết Giáp Hùng (Iron Armored Bear)":
					defName = "Thiết Giáp Thân Thể"
				case "Độc Lang Vương (Wolf Leader)":
					defName = "Lang Hống Hộ Vệ"
				case "Dã Trĩ Vương (Boar King)":
					defName = "Bản Năng Dã Tính"
				}
			}
			enemyActions = append(enemyActions, Action{
				Name: defName,
				Def:  player.ActiveCombat.EnemyDefense + 5.0,
			})
		} else {
			enemyActions = append(enemyActions, Action{
				Name: "Gầm Rú Tích Lực (Monster Charge)",
			})
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("%s tích tụ Nguyên Khí chuẩn bị cho đợt tấn công tiếp theo!", player.ActiveCombat.EnemyName))
		}
	}

	// 6. Execute Abilities Simultaneously & Apply Clash Logic
	maxSteps := len(playerActions)
	if len(enemyActions) > maxSteps {
		maxSteps = len(enemyActions)
	}

	ctx.Logs = append(ctx.Logs, "=== KÍCH HOẠT SÁT CHIÊU ĐỒNG THỜI ===")

	for step := 0; step < maxSteps; step++ {
		var pAct *Action
		var eAct *Action

		if step < len(playerActions) {
			pAct = &playerActions[step]
		}
		if step < len(enemyActions) {
			eAct = &enemyActions[step]
		}

		pActive := false
		if pAct != nil {
			if player.CurrentMana >= pAct.ManaCost {
				player.CurrentMana -= pAct.ManaCost
				pActive = true
			} else {
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu kiệt Chân Nguyên! Kích hoạt %s thất bại.", pAct.Name))
			}
		}

		eActive := false
		if eAct != nil {
			if isPVP && eAct.ManaCost > 0 {
				if enemyCurrentMana >= eAct.ManaCost {
					enemyCurrentMana -= eAct.ManaCost
					eActive = true
				} else {
					ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Kẻ địch kiệt Chân Nguyên! Kích hoạt %s thất bại.", eAct.Name))
				}
			} else {
				eActive = true
			}
		}

		if pActive && eActive {
			pDmg := pAct.Dmg
			pDef := pAct.Def
			eDmg := eAct.Dmg
			eDef := eAct.Def

			// Utility triggers
			if pAct.Name == "Tửu Trùng" {
				ctx.DamageBoost += 0.20
				ctx.Logs = append(ctx.Logs, "Đạo hữu kích hoạt Tửu Trùng nâng phẩm Chân Nguyên (+20% Sát thương).")
			} else if pAct.Name == "Tứ Vị Tửu Trùng" {
				ctx.DamageBoost += 0.40
				ctx.Logs = append(ctx.Logs, "Đạo hữu kích hoạt Tứ Vị Tửu Trùng cường hóa Chân Nguyên (+40% Sát thương).")
			} else if pAct.Name == "Tịnh Thủy Cổ" {
				ctx.Logs = append(ctx.Logs, "Đạo hữu thanh tẩy ý chí bám bẩn trên thân thể.")
			}

			pDmg = pDmg * (1.0 + ctx.DamageBoost)

			// Clash Mechanic: Both execute offensive strikes
			if pDmg > 0 && eDmg > 0 {
				if pDmg > eDmg {
					netDmg := pDmg - eDmg
					ctx.Logs = append(ctx.Logs, fmt.Sprintf("SÁT CHIÊU VA CHẠM! %s của đạo hữu (%.1f Dmg) áp đảo hoàn toàn %s của địch (%.1f Dmg)!", pAct.Name, pDmg, eAct.Name, eDmg))
					netDmg -= player.ActiveCombat.EnemyDefense
					if netDmg < 1.0 {
						netDmg = 1.0
					}
					player.ActiveCombat.EnemyHP -= netDmg
					ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đòn công phá vượt qua phòng ngự, gây %.1f sát thương lên địch.", netDmg))
				} else if eDmg > pDmg {
					netDmg := eDmg - pDmg
					ctx.Logs = append(ctx.Logs, fmt.Sprintf("SÁT CHIÊU VA CHẠM! %s của địch (%.1f Dmg) đè bẹp %s của đạo hữu (%.1f Dmg)!", eAct.Name, eDmg, pAct.Name, pDmg))
					netDmg -= playerStats["defense"]
					if netDmg < 1.0 {
						netDmg = 1.0
					}
					player.HP -= netDmg
					ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu chịu %.1f sát thương dội ngược.", netDmg))
				} else {
					ctx.Logs = append(ctx.Logs, fmt.Sprintf("Lực lượng cân bằng! Sát chiêu %s và %s va chạm nổ tung, tự triệt tiêu (Đồng %.1f Dmg)!", pAct.Name, eAct.Name, pDmg))
				}
			} else {
				// One attacks, one defends/support
				if pDmg > 0 {
					netDmg := pDmg
					if eDef > 0 {
						netDmg -= eDef
						if netDmg < 0 {
							netDmg = 0
						}
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đạo hữu đánh ra %s (%.1f Dmg), nhưng bị %s của địch (%.1f Thủ) hấp thụ bớt.", pAct.Name, pDmg, eAct.Name, eDef))
					} else {
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đạo hữu dùng %s đánh trúng đối thủ %.1f sát thương.", pAct.Name, pDmg))
					}
					if netDmg > 0 {
						netDmg -= player.ActiveCombat.EnemyDefense
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						player.ActiveCombat.EnemyHP -= netDmg
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Gây %.1f sát thương.", netDmg))
					}
				}

				if eDmg > 0 {
					netDmg := eDmg
					if pDef > 0 {
						netDmg -= pDef
						if netDmg < 0 {
							netDmg = 0
						}
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kẻ địch bổ ra %s (%.1f Dmg), nhưng bị %s của đạo hữu (%.1f Thủ) chống đỡ.", eAct.Name, eDmg, pAct.Name, pDef))
					} else {
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kẻ địch kích hoạt %s đánh trúng đạo hữu %.1f sát thương.", eAct.Name, eDmg))
					}
					if netDmg > 0 {
						netDmg -= playerStats["defense"]
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						player.HP -= netDmg
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu mất %.1f HP.", netDmg))
					}
				}

				if pDmg == 0 && eDmg == 0 {
					if pDef > 0 {
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đạo hữu kích hoạt %s tạo %.1f lá chắn bảo vệ.", pAct.Name, pDef))
					}
					if eDef > 0 {
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kẻ địch dựng lên phòng ngự %s (+%.1f Thủ).", eAct.Name, eDef))
					}
				}
			}
		} else if pActive {
			pDmg := pAct.Dmg * (1.0 + ctx.DamageBoost)
			if pDmg > 0 {
				netDmg := pDmg - player.ActiveCombat.EnemyDefense
				if netDmg < 1.0 {
					netDmg = 1.0
				}
				player.ActiveCombat.EnemyHP -= netDmg
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đạo hữu thi triển %s đánh lén thành công, gây %.1f sát thương.", pAct.Name, netDmg))
			} else if pAct.Def > 0 {
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đạo hữu tăng cường phòng hộ từ %s (+%.1f Thủ).", pAct.Name, pAct.Def))
			}
		} else if eActive {
			eDmg := eAct.Dmg
			if eDmg > 0 {
				netDmg := eDmg - playerStats["defense"]
				if netDmg < 1.0 {
					netDmg = 1.0
				}
				player.HP -= netDmg
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kẻ địch chớp thời cơ dùng %s, đánh thọt đạo hữu %.1f sát thương.", eAct.Name, netDmg))
			} else if eAct.Def > 0 {
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("Kẻ địch tập trung phòng thủ bằng %s (+%.1f Thủ).", eAct.Name, eAct.Def))
			}
		}

		if player.ActiveCombat.EnemyHP <= 0 || player.HP <= 0 {
			break
		}
	}

	// 7. Check Victory or Defeat
	victory := player.ActiveCombat.EnemyHP <= 0
	defeat := false

	if victory {
		player.ActiveCombat.EnemyHP = 0

		if player.ActiveCombat.TargetPlayerID != "" {
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
				}
			}
			player.PrimevalStones += stolenStones
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiến thắng PVP vang dội! Đạo hữu cướp đoạt %d Nguyên Thạch từ đối phương!", stolenStones))
		} else {
			// PVE Victory
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiến thắng! Đạo hữu đã tiêu diệt %s!", player.ActiveCombat.EnemyName))

			rewardStones := 5 + rand.Intn(11)
			player.PrimevalStones += rewardStones
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch.", rewardStones))

			rollMat := rand.Intn(100)
			if rollMat < 50 {
				if player.MaterialsInventory == nil {
					player.MaterialsInventory = make(models.MaterialsInventory)
				}
				player.MaterialsInventory["boar_meat"]++
				ctx.Logs = append(ctx.Logs, "Thu hoạch chiến lợi phẩm thú dữ: 1x Thịt Lợn Rừng.")
			} else if rollMat < 80 {
				if player.MaterialsInventory == nil {
					player.MaterialsInventory = make(models.MaterialsInventory)
				}
				player.MaterialsInventory["wild_boar_tusk"]++
				ctx.Logs = append(ctx.Logs, "Thu hoạch chiến lợi phẩm thú dữ: 1x Nanh Lợn Rừng.")
			}

			if player.ClearedCells == nil {
				player.ClearedCells = make(models.ClearedCellsMap)
			}
			coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
			player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
		}

		player.ActiveCombat = nil
	} else if player.HP <= 0 {
		player.HP = 0
		defeat = true
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đại bại! Đạo hữu bất tỉnh nhân sự trước áp lực của %s.", player.ActiveCombat.EnemyName))
		player.ActiveCombat = nil
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
		"player":   player,
		"logs":     ctx.Logs,
		"enemy":    player.ActiveCombat,
		"victory":  victory,
		"defeat":   defeat,
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

	// Deduct 5 mana
	exploreManaCost := 5.0
	if player.CurrentMana < exploreManaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Chân Nguyên. Vui lòng nghỉ ngơi tĩnh tọa để hồi phục."})
		return
	}
	player.CurrentMana -= exploreManaCost

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
		var enemy models.CombatState
		if req.NodeID == "three_kings_inheritance" {
			enemies := []models.CombatState{
				{EnemyName: "Mãng Xà Dung Nham (Lava Python)", EnemyHP: 150, EnemyMaxHP: 150, EnemyDamage: 24, EnemyDefense: 10},
				{EnemyName: "Thiết Giáp Hùng (Iron Armored Bear)", EnemyHP: 200, EnemyMaxHP: 200, EnemyDamage: 18, EnemyDefense: 18},
			}
			enemy = enemies[rand.Intn(len(enemies))]
		} else if req.NodeID == "gu_luan_slope" {
			enemies := []models.CombatState{
				{EnemyName: "Độc Lang Vương (Wolf Leader)", EnemyHP: 100, EnemyMaxHP: 100, EnemyDamage: 15, EnemyDefense: 6},
				{EnemyName: "Dã Trĩ Vương (Boar King)", EnemyHP: 120, EnemyMaxHP: 120, EnemyDamage: 12, EnemyDefense: 8},
			}
			enemy = enemies[rand.Intn(len(enemies))]
		} else {
			// Qing Mao Mountain default
			enemies := []models.CombatState{
				{EnemyName: "Dã Trĩ (Wild Boar)", EnemyHP: 50, EnemyMaxHP: 50, EnemyDamage: 8, EnemyDefense: 2},
				{EnemyName: "Thạch Hầu (Stone Ape)", EnemyHP: 60, EnemyMaxHP: 60, EnemyDamage: 10, EnemyDefense: 3},
			}
			enemy = enemies[rand.Intn(len(enemies))]
		}

		player.ActiveCombat = &enemy
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

