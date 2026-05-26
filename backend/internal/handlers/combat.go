package handlers

import (
	"fmt"
	"math/rand"
	"net/http"

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
		// Scale down base attributes according to regional rank suppression
		powerScale = float64(effectiveRank) / float64(guSpec.Rank)
	}

	effectiveDamage = guSpec.BaseDamage * powerScale
	effectiveDefense = guSpec.BaseDefense * powerScale

	// Rule 3: Mana cost calculation
	// Base cost: 10% of Max Mana
	cost := 0.10 * player.MaxMana

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

// CombatAttack handles PVE turns and execution of Killer Moves
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

	// 3. Resolve equipped Gus and compute mana requirements
	var selectedGus []models.GuInstance
	var selectedSpecs []models.GuDictionaryItem
	totalManaCost := 0.0

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
				// Backlash errors deduct HP but don't fail the verification stage
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

	// Check if enough mana
	if player.CurrentMana < totalManaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Chân Nguyên để kích hoạt sát chiêu."})
		return
	}

	// 4. Construct Combat Context
	ctx := &CombatContext{
		Attacker: &player,
		Defender: player.ActiveCombat,
		Logs:     []string{},
	}

	// 5. Execute abilities
	if len(selectedGus) > 0 {
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("--- Đạo hữu hợp thành SÁT CHIÊU từ %d Cổ trùng! ---", len(selectedGus)))
	} else {
		ctx.Logs = append(ctx.Logs, "Đạo hữu không sử dụng Cổ trùng nào, đành đấm tay không!")
		ctx.DamageDealt = 5.0 // Raw basic physical damage
	}

	for _, spec := range selectedSpecs {
		// Deduct mana cost
		_, _, cost, err := CalculateGuEffect(&player, &spec)
		if err == nil {
			player.CurrentMana -= cost
		}

		ability := AbilityRegistry[spec.ID]
		if ability != nil {
			ability.Execute(ctx, &spec)
		} else {
			// Fallback: raw damage equal to power if no ability strategy found
			ctx.DamageDealt += spec.BaseDamage
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("Sử dụng %s gây %.1f sát thương.", spec.Name, spec.BaseDamage))
		}
	}

	// 6. Apply Dao Marks and compute final damage output
	strengthMarks := player.DaoMarks["strength"]
	multiplier := 1.0 + (float64(strengthMarks) / 100.0)
	finalDamage := ctx.DamageDealt * (1.0 + ctx.DamageBoost) * multiplier

	// Subtract enemy defense
	netDamage := finalDamage - player.ActiveCombat.EnemyDefense
	if netDamage < 1.0 {
		netDamage = 1.0
	}

	player.ActiveCombat.EnemyHP -= netDamage
	ctx.Logs = append(ctx.Logs, fmt.Sprintf("Sát Chiêu hoàn tất! Dao Marks Lực Đạo (%d) khuếch đại sát thương. Gây %.1f sát thương lên %s.", strengthMarks, netDamage, player.ActiveCombat.EnemyName))

	// 7. Check Victory
	victory := player.ActiveCombat.EnemyHP <= 0
	defeat := false

	if victory {
		player.ActiveCombat.EnemyHP = 0
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiến thắng! Đạo hữu đã tiêu diệt %s!", player.ActiveCombat.EnemyName))

		// Seeding rewards
		rewardStones := 5 + rand.Intn(11) // 5 to 15 stones
		player.PrimevalStones += rewardStones
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm: +%d Nguyên Thạch.", rewardStones))

		// Material drop roll (50% meat, 30% tusk, 20% nothing)
		rollMat := rand.Intn(100)
		if rollMat < 50 {
			player.MaterialsInventory["boar_meat"]++
			ctx.Logs = append(ctx.Logs, "Thu hoạch thêm: 1x Thịt Lợn Rừng.")
		} else if rollMat < 80 {
			player.MaterialsInventory["wild_boar_tusk"]++
			ctx.Logs = append(ctx.Logs, "Thu hoạch thêm: 1x Nanh Lợn Rừng.")
		}

		// Clear combat state
		player.ActiveCombat = nil
	} else {
		// 8. Enemy counter-attacks!
		enemyNetDmg := player.ActiveCombat.EnemyDamage - ctx.ShieldAdded
		if enemyNetDmg < 1.0 {
			enemyNetDmg = 1.0
		}

		player.HP -= enemyNetDmg
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("%s gầm thét phản công! Triển khai sát thương %.1f. Lá chắn hấp thụ %.1f. Trừ %.1f HP của đạo hữu.", player.ActiveCombat.EnemyName, player.ActiveCombat.EnemyDamage, ctx.ShieldAdded, enemyNetDmg))

		if player.HP <= 0 {
			player.HP = 0
			defeat = true
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đạo hữu đã bại trận và gục ngã trước %s! Không khiếu chấn động dữ dội.", player.ActiveCombat.EnemyName))
			
			// Clear combat state on defeat
			player.ActiveCombat = nil
		}
	}

	// 9. Save Player rows
	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, primeval_stones = $3, materials_inventory = $4, gu_inventory = $5, active_combat = $6
		WHERE id = $7
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.PrimevalStones, player.MaterialsInventory, player.GuInventory, player.ActiveCombat, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player status"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Return updated state
	c.JSON(http.StatusOK, gin.H{
		"player":   player,
		"logs":     ctx.Logs,
		"enemy":    ctx.Defender, // Sends enemy state (nil if cleared)
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

	// 4. Update Database
	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, materials_inventory = $3, active_combat = $4
		WHERE id = $5
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.MaterialsInventory, player.ActiveCombat, player.ID)
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
		penaltyHP := 5.0
		penaltyMana := 5.0

		player.HP -= penaltyHP
		if player.HP < 1.0 {
			player.HP = 1.0 // Clamped at 1.0 to prevent dying
		}

		player.CurrentMana -= penaltyMana
		if player.CurrentMana < 0.0 {
			player.CurrentMana = 0.0
		}

		logMsg = fmt.Sprintf("Đạo hữu hốt hoảng tháo chạy khỏi %s! Không khiếu chấn động dữ dội, bị tổn thất %.1f HP và %.1f Chân Nguyên.", 
			player.ActiveCombat.EnemyName, penaltyHP, penaltyMana)
		
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

