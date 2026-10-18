package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type MineRates struct {
	StonesPerMin float64
	QiPerMin     float64
	ChestChance  float64
}

var MineRarityRates = map[string]MineRates{
	"Siêu Cấp":      {StonesPerMin: 60.0, QiPerMin: 15.0, ChestChance: 0.5},
	"Truyền Thuyết": {StonesPerMin: 30.0, QiPerMin: 7.5, ChestChance: 0.2},
	"Sử Thi":        {StonesPerMin: 25.0, QiPerMin: 6.25, ChestChance: 0.15},
	"Hiếm":          {StonesPerMin: 22.0, QiPerMin: 5.5, ChestChance: 0.12},
	"Thường":        {StonesPerMin: 15.0, QiPerMin: 3.0, ChestChance: 0.08},
}

type MineSlotResponse struct {
	ID                int        `json:"id"`
	Rarity            string     `json:"rarity"`
	MineType          string     `json:"mine_type"`
	OccupierID        *string    `json:"occupier_id"`
	OccupierName      *string    `json:"occupier_name"`
	MonsterID         *string    `json:"monster_id"`
	OccupiedAt        *time.Time `json:"occupied_at"`
	LastClaimedAt     *time.Time `json:"last_claimed_at"`
	AccumulatedStones int64      `json:"accumulated_stones"`
	AccumulatedQi     float64    `json:"accumulated_qi"`
	AccumulatedChests int        `json:"accumulated_chests"`
}

func CalculateMineRewards(slotID int, rarity, mineType string, lastClaimedAt time.Time) (int64, float64, int) {
	elapsed := time.Since(lastClaimedAt)
	elapsedMinutes := int(elapsed.Minutes())
	if elapsedMinutes <= 0 {
		return 0, 0.0, 0
	}

	rates, ok := MineRarityRates[rarity]
	if !ok {
		rates = MineRarityRates["Thường"]
	}

	stones := int64(float64(elapsedMinutes) * rates.StonesPerMin)
	qi := float64(elapsedMinutes) * rates.QiPerMin

	// Stable chest roll
	seed := int64(slotID) + lastClaimedAt.Unix()
	r := rand.New(rand.NewSource(seed))
	chests := 0
	for i := 0; i < elapsedMinutes; i++ {
		chance := rates.ChestChance
		if mineType == "immortal" {
			chance *= immortalChestDecay(i)
		}
		if r.Float64() < chance {
			chests++
		}
	}

	return stones, qi, chests
}

// immortalChestDecay: hệ số nhân tỉ lệ rương của Tiên Mạch theo số phút tích
// lũy (anti-AFK). Giảm 50% mỗi 2 giờ (120 phút), sàn 20% base; reset mỗi claim
// vì phút được đếm lại từ lastClaimedAt.
func immortalChestDecay(minute int) float64 {
	decay := math.Pow(0.5, float64(minute)/120.0)
	if decay < 0.2 {
		return 0.2
	}
	return decay
}

// mineChestItemID trả về item rương theo loại mỏ: Tiên Mạch nhả Rương Mỏ Tiên
// Thạch (tỉ lệ bí phương/vật phẩm rank >5 cao hơn), còn lại là rương thường.
func mineChestItemID(mineType string) string {
	if mineType == "immortal" {
		return "mine_chest_immortal"
	}
	return "mine_chest"
}

func AddMineRewardsToPlayerTx(tx *sql.Tx, playerID uuid.UUID, stones int64, qi float64, chests int, mineType string) error {
	var stonesAdd, immortalAdd float64
	if mineType == "immortal" {
		immortalAdd = float64(stones)
	} else {
		stonesAdd = float64(stones)
	}

	_, err := tx.Exec(fmt.Sprintf(`
		UPDATE players
		SET primeval_stones = primeval_stones + $1,
		    immortal_stones = immortal_stones + $2,
		    human_qi = LEAST(human_qi + $3, %d)
		WHERE id = $4
	`, int(MaxHumanQi)), stonesAdd, immortalAdd, qi, playerID)
	if err != nil {
		return err
	}

	if chests > 0 {
		if err := InventorySvc.AddItem(tx, playerID.String(), mineChestItemID(mineType), "materials", int64(chests)); err != nil {
			return err
		}
	}
	return nil
}

func GetMineSlots(c *gin.Context) {
	var cachedSlots []MineSlotResponse
	if err := cache.GetJSON("mine_slots_list", &cachedSlots); err == nil && len(cachedSlots) > 0 {
		c.JSON(http.StatusOK, cachedSlots)
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT m.id, m.rarity, m.mine_type, m.occupier_id, m.monster_id, m.occupied_at, m.last_claimed_at, p.username
		FROM mine_slots m
		LEFT JOIN players p ON m.occupier_id = p.id
		ORDER BY m.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load mine slots"})
		return
	}
	defer rows.Close()

	var slots []MineSlotResponse
	for rows.Next() {
		var id int
		var rarity string
		var mineType string
		var occupierID *string
		var monsterID *string
		var occupiedAt, lastClaimedAt *time.Time
		var occupierName sql.NullString

		err = rows.Scan(&id, &rarity, &mineType, &occupierID, &monsterID, &occupiedAt, &lastClaimedAt, &occupierName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan mine slots"})
			return
		}

		var name *string
		if occupierName.Valid {
			name = &occupierName.String
		}

		var stones int64 = 0
		qi := 0.0
		chests := 0

		if occupierID != nil && lastClaimedAt != nil {
			stones, qi, chests = CalculateMineRewards(id, rarity, mineType, *lastClaimedAt)
		}

		slots = append(slots, MineSlotResponse{
			ID:                id,
			Rarity:            rarity,
			MineType:          mineType,
			OccupierID:        occupierID,
			OccupierName:      name,
			MonsterID:         monsterID,
			OccupiedAt:        occupiedAt,
			LastClaimedAt:     lastClaimedAt,
			AccumulatedStones: stones,
			AccumulatedQi:     qi,
			AccumulatedChests: chests,
		})
	}

	_ = cache.SetJSON("mine_slots_list", slots, 10*time.Second)
	c.JSON(http.StatusOK, slots)
}

func OccupyMineSlot(c *gin.Context) {
	var req struct {
		SlotID int `json:"slot_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot ID is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Verify player is not in active combat
	var inCombat bool
	err = tx.QueryRow("SELECT active_combat IS NOT NULL FROM players WHERE id = $1", authID).Scan(&inCombat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check combat status"})
		return
	}
	if inCombat {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang ở trong trạng thái chiến đấu!"})
		return
	}

	// 2. Verify player does not already occupy a mine slot
	var currentOccupiedSlot int
	err = tx.QueryRow("SELECT COALESCE(id, 0) FROM mine_slots WHERE occupier_id = $1", authID).Scan(&currentOccupiedSlot)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check current occupation"})
		return
	}
	if currentOccupiedSlot > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo hữu đang chiếm giữ mỏ #%d. Chỉ có thể chiếm giữ tối đa 1 mỏ tại một thời điểm!", currentOccupiedSlot)})
		return
	}

	// 3. Verify slot is empty and load its info
	var rarity string
	var occupierID *string
	var monsterIDCheck *string
	err = tx.QueryRow("SELECT rarity, occupier_id, monster_id FROM mine_slots WHERE id = $1 FOR UPDATE NOWAIT", req.SlotID).Scan(&rarity, &occupierID, &monsterIDCheck)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Mỏ đang có người thao tác, vui lòng thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ không tồn tại"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to select slot"})
		}
		return
	}
	if occupierID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ này đã bị người khác chiếm giữ!"})
		return
	}
	if monsterIDCheck != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ có yêu quái canh giữ! Hãy đánh bại nó trước khi chiếm đóng."})
		return
	}

	// 4. Perform occupation
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE mine_slots
		SET occupier_id = $1, occupied_at = $2, last_claimed_at = $3
		WHERE id = $4
	`, authID, now, now, req.SlotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to occupy slot"})
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}
	invalidateMineSlotsCache()

	respondActionOnly(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Chiếm lĩnh thành công mỏ phẩm chất %s #%d! Nhân Khí +5.", rarity, req.SlotID)})
}

func ClaimMineRewards(c *gin.Context) {
	var req struct {
		SlotID int `json:"slot_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot ID is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Verify and lock slot
	var rarity string
	var mineType string
	var occupierID *string
	var lastClaimedAt *time.Time
	err = tx.QueryRow("SELECT rarity, mine_type, occupier_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE NOWAIT", req.SlotID).Scan(&rarity, &mineType, &occupierID, &lastClaimedAt)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Mỏ đang có người thao tác, vui lòng thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify slot"})
		return
	}

	if occupierID == nil || *occupierID != authID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không sở hữu mỏ này!"})
		return
	}

	// 2. Calculate rewards
	stones, qi, chests := CalculateMineRewards(req.SlotID, rarity, mineType, *lastClaimedAt)
	if stones == 0 && qi == 0.0 && chests == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa tích lũy đủ thời gian (tối thiểu 1 phút) để nhận thưởng!"})
		return
	}

	// 3. Update player rewards
	err = AddMineRewardsToPlayerTx(tx, authID, stones, qi, chests, mineType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add rewards to player"})
		return
	}

	// 4. Update last_claimed_at
	_, err = tx.Exec("UPDATE mine_slots SET last_claimed_at = $1 WHERE id = $2", time.Now(), req.SlotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset claim timer"})
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}
	invalidateMineSlotsCache()

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": "Nhận thưởng tích lũy thành công!",
		"rewards": gin.H{
			"stones": stones,
			"qi":     qi,
			"chests": chests,
		},
	})
}

func AbandonMineSlot(c *gin.Context) {
	var req struct {
		SlotID int `json:"slot_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot ID is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Verify and lock slot
	var rarity string
	var mineType string
	var occupierID *string
	var lastClaimedAt *time.Time
	err = tx.QueryRow("SELECT rarity, mine_type, occupier_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE NOWAIT", req.SlotID).Scan(&rarity, &mineType, &occupierID, &lastClaimedAt)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Mỏ đang có người thao tác, vui lòng thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify slot"})
		return
	}

	if occupierID == nil || *occupierID != authID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không sở hữu mỏ này!"})
		return
	}

	// 2. Calculate and grant rewards
	stones, qi, chests := CalculateMineRewards(req.SlotID, rarity, mineType, *lastClaimedAt)
	if stones > 0 || qi > 0.0 || chests > 0 {
		err = AddMineRewardsToPlayerTx(tx, authID, stones, qi, chests, mineType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add rewards to player"})
			return
		}
	}

	// 3. Clear mine slot
	_, err = tx.Exec(`
		UPDATE mine_slots
		SET occupier_id = NULL, occupied_at = NULL, last_claimed_at = NULL, abandoned_at = NOW()
		WHERE id = $1
	`, req.SlotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear mine slot"})
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}
	invalidateMineSlotsCache()

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": "Từ bỏ mỏ linh thạch thành công, đã nhận toàn bộ phần thưởng tích lũy!",
		"rewards": gin.H{
			"stones": stones,
			"qi":     qi,
			"chests": chests,
		},
	})
}

func AttackMineSlot(c *gin.Context) {
	var req struct {
		SlotID int `json:"slot_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot ID is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Load attacker and verify not in active combat
	attacker, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player info"})
		return
	}
	if attacker.ActiveCombat != nil && !attacker.ActiveCombat.IsEmpty() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang ở trong trạng thái chiến đấu!"})
		return
	}

	// 2. Verify player does not already occupy a mine slot
	var currentOccupiedSlot int
	err = tx.QueryRow("SELECT COALESCE(id, 0) FROM mine_slots WHERE occupier_id = $1", authID).Scan(&currentOccupiedSlot)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check current occupation"})
		return
	}
	if currentOccupiedSlot > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo hữu đang chiếm giữ mỏ #%d. Chỉ có thể chiếm giữ tối đa 1 mỏ tại một thời điểm!", currentOccupiedSlot)})
		return
	}

	// 3. Load slot info and occupier
	var occupierIDStr *string
	var monsterIDStr *string
	var monsterRank, monsterSubRank int
	err = tx.QueryRow("SELECT occupier_id, monster_id, monster_rank, monster_sub_rank FROM mine_slots WHERE id = $1 FOR UPDATE NOWAIT", req.SlotID).Scan(&occupierIDStr, &monsterIDStr, &monsterRank, &monsterSubRank)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Mỏ đang có người giao tranh hoặc thao tác, vui lòng thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ không tồn tại"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load slot occupier"})
		}
		return
	}
	if occupierIDStr == nil && monsterIDStr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ này trống, hãy dùng lệnh chiếm đóng thay vì tấn công!"})
		return
	}

	var pvpCombat models.CombatState

	if monsterIDStr != nil {
		// PVE Combat against Mine Guard Monster
		monsterTemplate, err := db.LoadMonsterTemplate(tx, *monsterIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy dữ liệu quái vật canh giữ"})
			return
		}
		pvpCombat = models.BuildCombatStateFromMonsterWithRegion(monsterTemplate, monsterRank, monsterSubRank, regionMonsterGuPool(attacker.CurrentRegion))
		pvpCombat.EnemyName = "Yêu Quái Canh Mỏ: " + monsterTemplate.Name
		pvpCombat.MineSlotID = req.SlotID
	} else {
		// PVP Combat against player occupier
		occupierUUID, err := uuid.Parse(*occupierIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse occupier UUID"})
			return
		}

		if occupierUUID == authID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thể tấn công mỏ của chính mình!"})
			return
		}

		// 4. Load defender (occupier) details
		defender, err := evaluateManaTxEquippedReadonly(tx, occupierUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông tin đối thủ"})
			return
		}

		defenderStats := CalculatePlayerStats(&defender)

		enemyHP := defender.HP
		if enemyHP <= 0 {
			enemyHP = 10.0
		}
		enemyMaxHP := gamefx.EffectiveMaxHP(&defender)
		if enemyMaxHP <= 0 {
			enemyMaxHP = 100.0
		}

		// Build PVP Mine Combat State (real mode: actual HP/mana, real resource cost)
		pvpCombat = models.CombatState{
			EnemyTemplateID:     defender.ID.String(),
			EnemyName:           defender.Username,
			EnemyType:           "PVP",
			CombatMode:          "real",
			EnemyHP:             enemyHP,
			EnemyMaxHP:          enemyMaxHP,
			EnemyMana:           defender.CurrentMana,
			EnemyMaxMana:        defender.MaxMana,
			EnemyDamage:         defenderStats["attack"],
			EnemyDefense:        defenderStats["defense"],
			EnemySpeed:          defenderStats["speed"],
			EnemyRank:           defender.Rank,
			EnemySubRank:        defender.SubRank,
			EnemyGuInventory:    defender.GuInventory,
			EnemyDaoMarks:       defender.DaoMarks,
			EnemyPathMasteries:  defender.PathMasteries,
			EnemyCombatSettings: &defender.CombatSettings,
			TargetPlayerID:      defender.ID.String(),
			MineSlotID:          req.SlotID,
		}
		gamefx.ApplyEnemyEffects(&pvpCombat, &defender)
	}

	// Initialize and save
	attacker.ActiveCombat = StartCombat(tx, &pvpCombat, &attacker)
	qiGained := AddHumanQi(&attacker, 3.0)

	combatBytes, mErr := json.Marshal(attacker.ActiveCombat)
	if mErr != nil {
		// NaN/Inf in the combat state makes json.Marshal fail; an empty payload
		// would poison the jsonb column, so fail the request cleanly instead.
		logger.S.Errorf("StartMineCombat: failed to marshal combat state for player %s: %v", authID, mErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat state"})
		return
	}
	_, err = tx.Exec("UPDATE players SET active_combat = $1, human_qi = $2 WHERE id = $3", string(combatBytes), attacker.HumanQi, authID)
	if err != nil {
		logger.S.Errorf("StartMineCombat: failed to save combat state for player %s: %v", authID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat state"})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	respondActionOnly(c, http.StatusOK, attacker, gin.H{"message": fmt.Sprintf("Khởi động chiến đấu chiếm mỏ thành công! Nhân Khí +%.0f.", qiGained)})
}

// StartMineMonsterWorker runs periodically to assign monsters to abandoned mines
func StartMineMonsterWorker(ctx context.Context) {
	run := func(ctx context.Context) { AssignMonstersToMines() }
	worker.RunEvery(ctx, 10*time.Minute, "Mine Monster", run) // check every 10 minutes
	// Run once at startup (in background)
	go worker.SafeRun("Mine Monster", ctx, run)
}

// pickRandomMineMonsterIDs returns up to n random active monster IDs in a
// single query. Prefers Elite/Boss templates; falls back to any active template
// when the preferred pool is empty. Replaces per-slot ORDER BY RANDOM() loops
// that previously re-sorted monster_templates once per abandoned mine slot.
func pickRandomMineMonsterIDs(tx *sql.Tx, n int) []string {
	if n <= 0 {
		return nil
	}
	ids := make([]string, 0, n)
	rows, err := tx.Query(`
		SELECT id FROM monster_templates
		WHERE is_active = TRUE
		  AND (LOWER(type) LIKE '%elite%' OR LOWER(type) LIKE '%boss%')
		ORDER BY RANDOM() LIMIT $1
	`, n)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) > 0 {
		return ids
	}
	rows2, err := tx.Query(`
		SELECT id FROM monster_templates WHERE is_active = TRUE ORDER BY RANDOM() LIMIT $1
	`, n)
	if err != nil {
		return nil
	}
	defer rows2.Close()
	for rows2.Next() {
		var id string
		if err := rows2.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func AssignMonstersToMines() {
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// Find abandoned mines
	rows, err := tx.Query(`
		SELECT id, rarity, mine_type 
		FROM mine_slots 
		WHERE occupier_id IS NULL AND monster_id IS NULL 
		AND (
			(rarity = 'Siêu Cấp' AND abandoned_at < NOW() - INTERVAL '1 hour') OR
			(rarity = 'Truyền Thuyết' AND abandoned_at < NOW() - INTERVAL '2 hours') OR
			(rarity = 'Sử Thi' AND abandoned_at < NOW() - INTERVAL '4 hours') OR
			(rarity NOT IN ('Siêu Cấp', 'Truyền Thuyết', 'Sử Thi') AND abandoned_at < NOW() - INTERVAL '6 hours')
		)
	`)
	if err != nil {
		logger.S.Errorf("Error finding abandoned mines: %v", err)
		return
	}
	defer rows.Close()

	type Slot struct {
		ID       int
		Rarity   string
		MineType string
	}
	var abandonedSlots []Slot
	for rows.Next() {
		var s Slot
		if err := rows.Scan(&s.ID, &s.Rarity, &s.MineType); err == nil {
			abandonedSlots = append(abandonedSlots, s)
		}
	}
	rows.Close()

	if len(abandonedSlots) == 0 {
		return
	}

	// Single random pick for the whole batch — replaces per-slot ORDER BY RANDOM().
	picks := pickRandomMineMonsterIDs(tx, len(abandonedSlots))

	for i, slot := range abandonedSlots {
		var targetRank int
		if slot.MineType == "immortal" {
			switch slot.Rarity {
			case "Siêu Cấp":
				targetRank = 9
			case "Truyền Thuyết":
				targetRank = 8
			case "Sử Thi":
				targetRank = 7
			default:
				targetRank = 6
			}
		} else {
			switch slot.Rarity {
			case "Siêu Cấp":
				targetRank = 6
			case "Truyền Thuyết":
				targetRank = 5
			case "Sử Thi":
				targetRank = 4
			case "Hiếm":
				targetRank = 3
			default:
				targetRank = 2
			}
		}

		// Round-robin through the pre-fetched picks so different slots still
		// get different monsters when more slots than picks exist.
		chosenMonsterID := ""
		if len(picks) > 0 {
			chosenMonsterID = picks[i%len(picks)]
		}

		if chosenMonsterID != "" {
			_, err = tx.Exec("UPDATE mine_slots SET monster_id = $1, monster_rank = $2, monster_sub_rank = $3 WHERE id = $4", chosenMonsterID, targetRank, 1, slot.ID)
			if err != nil {
				logger.S.Errorf("Failed to assign monster to mine slot %d: %v", slot.ID, err)
			} else {
				logger.S.Infof("Assigned monster %s to abandoned mine slot %d", chosenMonsterID, slot.ID)
			}
		}
	}

	_ = tx.Commit()
}

// StartMidnightMineWorker runs daily at 00:00 to assign monsters to empty mines
func StartMidnightMineWorker(ctx context.Context) {
	run := func(ctx context.Context) {
		logger.S.Info("Midnight Mine Worker - daily reset at 00:00")
		AssignMonstersToEmptyMines()
	}

	go func() {
		logger.S.Info("Midnight Mine Worker - initial run at startup")
		worker.SafeRun("Midnight Mine", ctx, func(ctx context.Context) { AssignMonstersToEmptyMines() })

		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		durationUntilMidnight := nextMidnight.Sub(now)

		select {
		case <-ctx.Done():
			return
		case <-time.After(durationUntilMidnight):
		}

		// Aligned to midnight; RunEvery starts its own ticker from here so the
		// daily reset does not drift with the boot time.
		worker.SafeRun("Midnight Mine", ctx, run)
		worker.RunEvery(ctx, 24*time.Hour, "Midnight Mine", run)
	}()
}

// AssignMonstersToEmptyMines assigns monsters to all empty mine slots (no time condition)
func AssignMonstersToEmptyMines() {
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id, rarity, mine_type 
		FROM mine_slots 
		WHERE occupier_id IS NULL AND monster_id IS NULL
	`)
	if err != nil {
		logger.S.Errorf("Error finding empty mines: %v", err)
		return
	}
	defer rows.Close()

	type Slot struct {
		ID       int
		Rarity   string
		MineType string
	}
	var emptySlots []Slot
	for rows.Next() {
		var s Slot
		if err := rows.Scan(&s.ID, &s.Rarity, &s.MineType); err == nil {
			emptySlots = append(emptySlots, s)
		}
	}
	rows.Close()

	if len(emptySlots) == 0 {
		return
	}

	picks := pickRandomMineMonsterIDs(tx, len(emptySlots))

	for i, slot := range emptySlots {
		var targetRank int
		if slot.MineType == "immortal" {
			switch slot.Rarity {
			case "Siêu Cấp":
				targetRank = 9
			case "Truyền Thuyết":
				targetRank = 8
			case "Sử Thi":
				targetRank = 7
			default:
				targetRank = 6
			}
		} else {
			switch slot.Rarity {
			case "Siêu Cấp":
				targetRank = 6
			case "Truyền Thuyết":
				targetRank = 5
			case "Sử Thi":
				targetRank = 4
			case "Hiếm":
				targetRank = 3
			default:
				targetRank = 2
			}
		}

		chosenMonsterID := ""
		if len(picks) > 0 {
			chosenMonsterID = picks[i%len(picks)]
		}

		if chosenMonsterID != "" {
			_, err = tx.Exec("UPDATE mine_slots SET monster_id = $1, monster_rank = $2, monster_sub_rank = $3, abandoned_at = NOW() WHERE id = $4", chosenMonsterID, targetRank, 1, slot.ID)
			if err != nil {
				logger.S.Errorf("Failed to assign monster to mine slot %d: %v", slot.ID, err)
			} else {
				logger.S.Infof("[Midnight] Assigned monster %s to empty mine slot %d", chosenMonsterID, slot.ID)
			}
		}
	}

	_ = tx.Commit()
	invalidateMineSlotsCache()
}

func invalidateMineSlotsCache() {
	_ = cache.Delete("mine_slots_list")
}
