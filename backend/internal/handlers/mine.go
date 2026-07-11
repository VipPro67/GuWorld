package handlers

import (
	"context"
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

type MineRates struct {
	StonesPerMin float64
	QiPerMin     float64
	ChestChance  float64
}

var MineRarityRates = map[string]MineRates{
	"Truyền Thuyết": {StonesPerMin: 15.0, QiPerMin: 3.75, ChestChance: 0.01},
	"Sử Thi":        {StonesPerMin: 12.5, QiPerMin: 3.125, ChestChance: 0.0075},
	"Hiếm":          {StonesPerMin: 11.0, QiPerMin: 2.75, ChestChance: 0.006},
	"Thường":        {StonesPerMin: 10.0, QiPerMin: 2.5, ChestChance: 0.005},
}

type MineSlotResponse struct {
	ID                int        `json:"id"`
	Rarity            string     `json:"rarity"`
	OccupierID        *string    `json:"occupier_id"`
	OccupierName      *string    `json:"occupier_name"`
	MonsterID         *string    `json:"monster_id"`
	OccupiedAt        *time.Time `json:"occupied_at"`
	LastClaimedAt     *time.Time `json:"last_claimed_at"`
	AccumulatedStones int        `json:"accumulated_stones"`
	AccumulatedQi     float64    `json:"accumulated_qi"`
	AccumulatedChests int        `json:"accumulated_chests"`
}

func CalculateMineRewards(slotID int, rarity string, lastClaimedAt time.Time) (int, float64, int) {
	elapsed := time.Since(lastClaimedAt)
	elapsedMinutes := int(elapsed.Minutes())
	if elapsedMinutes <= 0 {
		return 0, 0.0, 0
	}

	rates, ok := MineRarityRates[rarity]
	if !ok {
		rates = MineRarityRates["Thường"]
	}

	stones := int(float64(elapsedMinutes) * rates.StonesPerMin)
	qi := float64(elapsedMinutes) * rates.QiPerMin

	// Stable chest roll
	seed := int64(slotID) + lastClaimedAt.Unix()
	r := rand.New(rand.NewSource(seed))
	chests := 0
	for i := 0; i < elapsedMinutes; i++ {
		if r.Float64() < rates.ChestChance {
			chests++
		}
	}

	return stones, qi, chests
}

func AddMineRewardsToPlayerTx(tx *sql.Tx, playerID uuid.UUID, stones int, qi float64, chests int) error {
	var currentStones int
	var currentQi float64
	var matsBytes []byte

	err := tx.QueryRow(`
		SELECT primeval_stones, human_qi, materials_inventory
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&currentStones, &currentQi, &matsBytes)
	if err != nil {
		return err
	}

	var mats models.MaterialsInventory
	if len(matsBytes) > 0 {
		json.Unmarshal(matsBytes, &mats)
	}
	if mats == nil {
		mats = make(models.MaterialsInventory)
	}

	currentStones += stones
	currentQi += qi
	if currentQi > 100000.0 {
		currentQi = 100000.0
	}

	if chests > 0 {
		mats["mine_chest"] += chests
	}

	matsJSON, err := json.Marshal(mats)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE players
		SET primeval_stones = $1, human_qi = $2, materials_inventory = $3
		WHERE id = $4
	`, currentStones, currentQi, matsJSON, playerID)
	return err
}

func GetMineSlots(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT m.id, m.rarity, m.occupier_id, m.monster_id, m.occupied_at, m.last_claimed_at, p.username
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
		var occupierID *string
		var monsterID *string
		var occupiedAt, lastClaimedAt *time.Time
		var occupierName sql.NullString

		err = rows.Scan(&id, &rarity, &occupierID, &monsterID, &occupiedAt, &lastClaimedAt, &occupierName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan mine slots"})
			return
		}

		var name *string
		if occupierName.Valid {
			name = &occupierName.String
		}

		stones := 0
		qi := 0.0
		chests := 0

		if occupierID != nil && lastClaimedAt != nil {
			stones, qi, chests = CalculateMineRewards(id, rarity, *lastClaimedAt)
		}

		slots = append(slots, MineSlotResponse{
			ID:                id,
			Rarity:            rarity,
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
	err = tx.QueryRow("SELECT rarity, occupier_id, monster_id FROM mine_slots WHERE id = $1 FOR UPDATE", req.SlotID).Scan(&rarity, &occupierID, &monsterIDCheck)
	if err != nil {
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

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Chiếm lĩnh thành công mỏ phẩm chất %s #%d!", rarity, req.SlotID)})
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
	var occupierID *string
	var lastClaimedAt *time.Time
	err = tx.QueryRow("SELECT rarity, occupier_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE", req.SlotID).Scan(&rarity, &occupierID, &lastClaimedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify slot"})
		return
	}

	if occupierID == nil || *occupierID != authID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không sở hữu mỏ này!"})
		return
	}

	// 2. Calculate rewards
	stones, qi, chests := CalculateMineRewards(req.SlotID, rarity, *lastClaimedAt)
	if stones == 0 && qi == 0.0 && chests == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa tích lũy đủ thời gian (tối thiểu 1 phút) để nhận thưởng!"})
		return
	}

	// 3. Update player rewards
	err = AddMineRewardsToPlayerTx(tx, authID, stones, qi, chests)
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

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
	var occupierID *string
	var lastClaimedAt *time.Time
	err = tx.QueryRow("SELECT rarity, occupier_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE", req.SlotID).Scan(&rarity, &occupierID, &lastClaimedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify slot"})
		return
	}

	if occupierID == nil || *occupierID != authID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không sở hữu mỏ này!"})
		return
	}

	// 2. Calculate and grant rewards
	stones, qi, chests := CalculateMineRewards(req.SlotID, rarity, *lastClaimedAt)
	if stones > 0 || qi > 0.0 || chests > 0 {
		err = AddMineRewardsToPlayerTx(tx, authID, stones, qi, chests)
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

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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

	// 1. Verify attacker is not in active combat
	var activeCombatBytes []byte
	var attackerHP float64
	var attackerCurrentMana float64
	var attackerMaxMana float64
	var attackerUsername string
	err = tx.QueryRow(`
		SELECT active_combat, hp, current_mana, max_mana, username
		FROM players WHERE id = $1 FOR UPDATE
	`, authID).Scan(&activeCombatBytes, &attackerHP, &attackerCurrentMana, &attackerMaxMana, &attackerUsername)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player info"})
		return
	}
	if len(activeCombatBytes) > 0 {
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
	err = tx.QueryRow("SELECT occupier_id, monster_id, monster_rank, monster_sub_rank FROM mine_slots WHERE id = $1", req.SlotID).Scan(&occupierIDStr, &monsterIDStr, &monsterRank, &monsterSubRank)
	if err != nil {
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

	// Load attacker details to build CombatState
	var attacker models.Player
	attacker.ID = authID
	attacker.Username = attackerUsername
	attacker.HP = attackerHP
	attacker.CurrentMana = attackerCurrentMana
	attacker.MaxMana = attackerMaxMana

	var pvpCombat models.CombatState

	if monsterIDStr != nil {
		// PVE Combat against Mine Guard Monster
		monsterTemplate, err := db.LoadMonsterTemplate(tx, *monsterIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy dữ liệu quái vật canh giữ"})
			return
		}
		maxHP, attack, defense := models.ScaleMonsterStats(monsterTemplate.MaxHP, monsterTemplate.BaseAttack, monsterTemplate.BaseDefense, monsterRank, monsterSubRank, monsterTemplate.Type)

		pvpCombat = models.CombatState{
			EnemyTemplateID:    monsterTemplate.ID,
			EnemyName:          "Yêu Quái Canh Mỏ: " + monsterTemplate.Name,
			EnemyType:          "MINE_PVE",
			EnemyHP:            maxHP,
			EnemyMaxHP:         maxHP,
			EnemyDamage:        attack,
			EnemyDefense:       defense,
			EnemyRank:          monsterRank,
			EnemySubRank:       monsterSubRank,
			EnemyGuInventory:   monsterTemplate.GuInventory,
			EnemyDaoMarks:      monsterTemplate.DaoMarks,
			EnemyPathMasteries: monsterTemplate.PathMasteries,
			AIProfile:          monsterTemplate.AIProfile,
			LootTableID:        monsterTemplate.LootTableID,
			TargetPlayerID:     "",
			MineSlotID:         req.SlotID,
		}
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
		var defender models.Player
		err = tx.QueryRow(`
			SELECT id, username, rank, sub_rank, hp, max_mana, path_masteries, gu_inventory, dao_marks, current_mana, base_hp
			FROM players WHERE id = $1 FOR UPDATE`, occupierUUID).Scan(
			&defender.ID, &defender.Username, &defender.Rank, &defender.SubRank,
			&defender.HP, &defender.MaxMana, &defender.PathMasteries, &defender.GuInventory, &defender.DaoMarks, &defender.CurrentMana, &defender.BaseHP,
		)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông tin đối thủ"})
			return
		}

		defenderStats := CalculatePlayerStats(&defender)

		enemyHP := defender.BaseHP
		if enemyHP <= 0 {
			enemyHP = 100.0
		}

		// Build PVP Mine Combat State
		pvpCombat = models.CombatState{
			EnemyTemplateID:    defender.ID.String(),
			EnemyName:          defender.Username,
			EnemyType:          "PVP",
			EnemyHP:            enemyHP,
			EnemyMaxHP:         enemyHP,
			EnemyMana:          defender.CurrentMana,
			EnemyMaxMana:       defender.MaxMana,
			EnemyDamage:        defenderStats["attack"],
			EnemyDefense:       defenderStats["defense"],
			EnemyRank:          defender.Rank,
			EnemySubRank:       defender.SubRank,
			EnemyGuInventory:   defender.GuInventory,
			EnemyDaoMarks:      defender.DaoMarks,
			EnemyPathMasteries: defender.PathMasteries,
			TargetPlayerID:     defender.ID.String(),
			MineSlotID:         req.SlotID,
		}
	}

	// Initialize and save
	attacker.ActiveCombat = StartCombat(tx, &pvpCombat, &attacker)

	combatBytes, _ := json.Marshal(attacker.ActiveCombat)
	_, err = tx.Exec("UPDATE players SET active_combat = $1 WHERE id = $2", combatBytes, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat state"})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Khởi động chiến đấu chiếm mỏ thành công!"})
}

// StartMineMonsterWorker runs periodically to assign monsters to abandoned mines
func StartMineMonsterWorker(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Minute) // check every hour
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("Mine Monster Worker stopped")
				return
			case <-ticker.C:
				AssignMonstersToMines()
			}
		}
	}()
	// Run once at startup (in background)
	go AssignMonstersToMines()
}

func AssignMonstersToMines() {
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// Find abandoned mines
	rows, err := tx.Query(`
		SELECT id, rarity 
		FROM mine_slots 
		WHERE occupier_id IS NULL AND monster_id IS NULL 
		AND abandoned_at < NOW() - INTERVAL '24 hours'
	`)
	if err != nil {
		log.Printf("Error finding abandoned mines: %v", err)
		return
	}
	defer rows.Close()

	type Slot struct {
		ID     int
		Rarity string
	}
	var abandonedSlots []Slot
	for rows.Next() {
		var s Slot
		if err := rows.Scan(&s.ID, &s.Rarity); err == nil {
			abandonedSlots = append(abandonedSlots, s)
		}
	}
	rows.Close()

	if len(abandonedSlots) == 0 {
		return
	}

	for _, slot := range abandonedSlots {
		var targetRank int
		switch slot.Rarity {
		case "Thường":
			targetRank = 2
		case "Hiếm":
			targetRank = 3
		case "Sử Thi":
			targetRank = 4
		case "Truyền Thuyết":
			targetRank = 5
		default:
			targetRank = 2
		}

		// Find a monster template
		var chosenMonsterID string
		_ = tx.QueryRow("SELECT id FROM monster_templates WHERE is_active = TRUE ORDER BY RANDOM() LIMIT 1").Scan(&chosenMonsterID)

		if chosenMonsterID != "" {
			_, err = tx.Exec("UPDATE mine_slots SET monster_id = $1, monster_rank = $2, monster_sub_rank = $3 WHERE id = $4", chosenMonsterID, targetRank, 1, slot.ID)
			if err != nil {
				log.Printf("Failed to assign monster to mine slot %d: %v", slot.ID, err)
			} else {
				log.Printf("Assigned monster %s to abandoned mine slot %d", chosenMonsterID, slot.ID)
			}
		}
	}

	_ = tx.Commit()
}

// StartMidnightMineWorker runs daily at 00:00 to assign monsters to empty mines
func StartMidnightMineWorker(ctx context.Context) {
	go func() {
		log.Println("Midnight Mine Worker - initial run at startup")
		AssignMonstersToEmptyMines()

		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		durationUntilMidnight := nextMidnight.Sub(now)

		select {
		case <-ctx.Done():
			return
		case <-time.After(durationUntilMidnight):
		}

		log.Println("Midnight Mine Worker - daily reset at 00:00")
		AssignMonstersToEmptyMines()

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Midnight Mine Worker stopped")
				return
			case <-ticker.C:
				log.Println("Midnight Mine Worker - daily reset at 00:00")
				AssignMonstersToEmptyMines()
			}
		}
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
		SELECT id, rarity 
		FROM mine_slots 
		WHERE occupier_id IS NULL AND monster_id IS NULL
	`)
	if err != nil {
		log.Printf("Error finding empty mines: %v", err)
		return
	}
	defer rows.Close()

	type Slot struct {
		ID     int
		Rarity string
	}
	var emptySlots []Slot
	for rows.Next() {
		var s Slot
		if err := rows.Scan(&s.ID, &s.Rarity); err == nil {
			emptySlots = append(emptySlots, s)
		}
	}
	rows.Close()

	if len(emptySlots) == 0 {
		return
	}

	for _, slot := range emptySlots {
		var targetRank int
		switch slot.Rarity {
		case "Thường":
			targetRank = 2
		case "Hiếm":
			targetRank = 3
		case "Sử Thi":
			targetRank = 4
		case "Truyền Thuyết":
			targetRank = 5
		default:
			targetRank = 2
		}

		var chosenMonsterID string
		_ = tx.QueryRow("SELECT id FROM monster_templates WHERE is_active = TRUE ORDER BY RANDOM() LIMIT 1").Scan(&chosenMonsterID)

		if chosenMonsterID != "" {
			_, err = tx.Exec("UPDATE mine_slots SET monster_id = $1, monster_rank = $2, monster_sub_rank = $3, abandoned_at = NOW() WHERE id = $4", chosenMonsterID, targetRank, 1, slot.ID)
			if err != nil {
				log.Printf("Failed to assign monster to mine slot %d: %v", slot.ID, err)
			} else {
				log.Printf("[Midnight] Assigned monster %s to empty mine slot %d", chosenMonsterID, slot.ID)
			}
		}
	}

	_ = tx.Commit()
}
