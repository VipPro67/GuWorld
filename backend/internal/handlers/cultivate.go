package handlers

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// BreakWallRequest - mortals choose amount of mana and stones to attempt breaking their aperture wall
type BreakWallRequest struct {
	ManaSpend float64 `json:"mana_spend" binding:"required"`
}

// BreakWallResponse returns updated player and result
type BreakWallResponse struct {
	Player  models.Player `json:"player"`
	Success bool          `json:"success"`
	Message string        `json:"message"`
}

// UseRelicGuRequest consumes a ranked relic gu item to instantly raise a sub-rank
type UseRelicGuRequest struct {
	RelicKey string `json:"relic_key" binding:"required"` // e.g., relic_gu_rank_1
}

// BreakWall handles mortal cultivation attempts (Rank 1-5)
func BreakWall(c *gin.Context) {
	var req BreakWallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
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

	// Only mortals (rank 1-5) can use wall breaking
	if player.Rank >= 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only Mortal cultivators can perform wall breaking"})
		return
	}

	if req.ManaSpend <= 0 || req.ManaSpend > player.CurrentMana {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mana spend amount"})
		return
	}

	// Deduct resources immediately
	player.CurrentMana -= req.ManaSpend
	player.LastManaUpdate = time.Now()

	// Aptitude multiplier (Giáp best, Đinh worst)
	aptMul := 0.8
	switch player.AptitudeType {
	case "Giáp":
		aptMul = 1.4
	case "Ất":
		aptMul = 1.1
	case "Bính":
		aptMul = 0.85
	case "Đinh":
		aptMul = 0.6
	}

	// Base success chance and progress calculation
	manaRatio := req.ManaSpend / math.Max(1.0, player.MaxMana)
	progressGain := manaRatio * (0.2 + 0.3*aptMul) / (float64(player.Rank) + float64(player.SubRank)/4.0)

	if player.ApertureStress >= 80.0 {
		progressGain *= 0.25
	}

	player.WallProgress += progressGain
	success := player.WallProgress >= 1.0
	player.ApertureStress += 5.0
	player.LastCultivateTime = time.Now()

	cracked := false
	if player.ApertureStress >= 100.0 {
		cracked = true
		player.ApertureStress = 50.0
		player.MaxMana = math.Max(20.0, math.Round(player.MaxMana*0.95))
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		if player.SubRank > 1 {
			player.SubRank -= 1
		} else if player.Rank > 1 {
			player.Rank -= 1
			player.SubRank = 4
		} else {
			player.SubRank = 1
		}
		player.WallProgress = 0.0
	}

	msg := "Hãy tiếp tục trùng kích không khiếu của bạn!"
	if cracked {
		msg = "Cảnh báo: Áp lực Không Khiếu vượt quá giới hạn (100%), Không khiếu bị nứt vỡ! Đạo hữu bị rớt tu vi và tổn thất vĩnh viễn 5% chân nguyên tối đa!"
	} else if success {
		chance := GetBreakthroughChance(player.Rank, player.AptitudeType)

		if rand.Float64() > chance {
			player.CurrentMana *= 0.8                                  // mất 20% chân nguyên
			player.WallProgress = math.Max(0, player.WallProgress-0.1) // mất 10% tiến độ

			msg = fmt.Sprintf(
				"Trùng kích thất bại! Xác suất %.0f%%. Vách không khiếu bị phản chấn, mất 20%% chân nguyên.",
				chance*100,
			)

			_, err := tx.Exec(`
				UPDATE players
				SET current_mana = $1,
					wall_progress = $2
				WHERE id = $3
			`, player.CurrentMana, player.WallProgress, player.ID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật"})
				return
			}

			tx.Commit()

			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"chance":  chance,
				"message": msg,
				"player":  player,
			})

			return
		}

		if player.Rank == 5 && player.SubRank == 4 {
			player.WallProgress = 1.0
			msg = "Bạn đã mài mòn vách không khiếu đến cực hạn Ngũ Chuyển Đỉnh Phong. Muốn tiến lên Lục Chuyển cần thực hiện Thăng Tiên, không thể dùng trùng kích thường."
		} else {
			player.SubRank += 1
			if player.SubRank > 4 {
				player.SubRank = 1
				player.Rank += 1
			}
			player.WallProgress = 0.0
			if player.Rank >= 6 {
				player.MaxMana = 999999999.0
			}
			if player.HumanQi > 100000.0 {
				player.HumanQi = 100000.0
			}
			msg = "Chúc mừng! Bạn đã trùng kích vỡ nát được một tầng khiếu, tiến lên tu vi cao hơn!"
		}
	}

	updateQuery := `
        UPDATE players 
        SET current_mana = $1, primeval_stones = $2, wall_progress = $3, max_mana = $4, sub_rank = $5, rank = $6, last_mana_update = $7, human_qi = $8, aperture_stress = $9, last_cultivate_time = $10
        WHERE id = $11
    `
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.WallProgress, player.MaxMana, player.SubRank, player.Rank, player.LastManaUpdate, player.HumanQi, player.ApertureStress, player.LastCultivateTime, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save cultivation progress"})
		return
	}

	err = CheckReferralMilestones(tx, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check referral milestones: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	actionName := "cultivate.break_wall"
	if cracked {
		actionName = "cultivate.wall_cracked"
	} else if success {
		actionName = "cultivate.breakthrough"
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: actionName,
		RequestSummary: fmt.Sprintf("rank=%d sub=%d progress=%.2f stress=%.0f manaSpent=%.0f", player.Rank, player.SubRank, player.WallProgress, player.ApertureStress, req.ManaSpend),
		IPAddress:      c.ClientIP(), Success: success && !cracked,
	})
	c.JSON(http.StatusOK, BreakWallResponse{Player: player, Success: success && !cracked, Message: msg})
}

func GetBreakthroughChance(rank int, aptitude string) float64 {
	// Xác suất nền theo tư chất
	var baseChance float64

	switch aptitude {
	case "A", "Giáp":
		baseChance = 0.98
	case "B", "Ất":
		baseChance = 0.92
	case "C", "Bính":
		baseChance = 0.85
	case "D", "Đinh":
		baseChance = 0.70
	default:
		baseChance = 0.60
	}

	// Rank càng cao càng khó
	rankPenalty := map[int]float64{
		1: 0.00, // Nhất -> Nhị
		2: 0.05, // Nhị -> Tam
		3: 0.10, // Tam -> Tứ
		4: 0.20, // Tứ -> Ngũ
		5: 0.35, // Ngũ -> Lục (chỉ dùng nếu sau này có)
	}

	chance := baseChance

	if penalty, ok := rankPenalty[rank]; ok {
		chance -= penalty
	}

	// Giới hạn
	if chance > 0.99 {
		chance = 0.99
	}

	if chance < 0.05 {
		chance = 0.05
	}

	return chance
}

// UseRelicGu consumes a relic item to instantly increase sub-rank
func UseRelicGu(c *gin.Context) {
	var req UseRelicGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
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

	// Check ownership of relic in materials inventory
	if player.MaterialsInventory == nil || player.MaterialsInventory[req.RelicKey] <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You do not possess that Relic Gu"})
		return
	}

	// Consume relic
	player.MaterialsInventory[req.RelicKey]--
	// Instant advancement
	player.SubRank += 1
	if player.SubRank > 4 {
		player.SubRank = 1
		player.Rank += 1
	}
	// Reset wall progress
	player.WallProgress = 0.0

	// Increase human_qi by 30 for breakthrough
	player.HumanQi += 30.0
	if player.HumanQi > 100000.0 {
		player.HumanQi = 100000.0
	}

	updateQuery := `
        UPDATE players SET materials_inventory = $1, sub_rank = $2, rank = $3, wall_progress = $4, human_qi = $5
        WHERE id = $6
    `
	_, err = tx.Exec(updateQuery, player.MaterialsInventory, player.SubRank, player.Rank, player.WallProgress, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply Relic Gu"})
		return
	}

	err = CheckReferralMilestones(tx, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check referral milestones: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"player": player, "message": "Relic Gu consumed. Instant advancement!"})
}

// Ascension attempts to break through from Rank 5 Peak to Rank 6 Gu Immortal
func Ascension(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
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

	// Must be Rank 5, SubRank 4 to ascend
	if player.Rank != 5 || player.SubRank != 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You must reach Rank 5 Peak (Đỉnh phong) to attempt Ascension"})
		return
	}

	// Ascension calculation: Heaven Qi, Earth Qi, Human Qi
	// For simplicity, we just check if HumanQi > threshold and add some randomness
	humanQiRequired := 100.0 // arbitrary threshold
	heavenQi := rand.Float64() * 50.0
	earthQi := rand.Float64() * 50.0

	totalQi := player.HumanQi + heavenQi + earthQi

	success := false
	msg := ""

	if totalQi > humanQiRequired && player.HumanQi > 40.0 {
		success = true
	} else {
		// Random chance based on HumanQi
		chance := player.HumanQi / humanQiRequired
		if rand.Float64() < chance {
			success = true
		}
	}

	if success {
		player.Rank = 6
		player.SubRank = 1
		player.HumanQi = 0.0 // reset after successful ascension
		if player.DaoMarks == nil {
			player.DaoMarks = models.DaoMarksMap{}
		}
		player.DaoMarks["Heaven"] = 100
		player.DaoMarks["Earth"] = 100

		// Set first tribulation to 10 days from now (simulated real-time)
		nextTrib := time.Now().Add(10 * 24 * time.Hour)
		player.NextTribulationTime = &nextTrib
		player.TribulationCount = 0
		player.ImmortalStones = 10.0 // Initial immortal stones

		msg = "Thăng tiên thành công! Bạn đã khai辟 Tiên Khiếu."

		updateQuery := `
			UPDATE players SET rank = $1, sub_rank = $2, human_qi = $3, dao_marks = $4, next_tribulation_time = $5, tribulation_count = $6, immortal_stones = $7
			WHERE id = $8
		`
		_, err = tx.Exec(updateQuery, player.Rank, player.SubRank, player.HumanQi, player.DaoMarks, player.NextTribulationTime, player.TribulationCount, player.ImmortalStones, player.ID)
	} else {
		// Failure penalty: Drop back to Rank 1 SubRank 1, lose Human Qi
		player.Rank = 1
		player.SubRank = 1
		player.HumanQi = 0.0
		player.WallProgress = 0.0

		msg = "Thăng tiên thất bại! Không khiếu vỡ nát, tu vi phế bỏ làm lại từ đầu."

		updateQuery := `
			UPDATE players SET rank = $1, sub_rank = $2, human_qi = $3, wall_progress = $4
			WHERE id = $5
		`
		_, err = tx.Exec(updateQuery, player.Rank, player.SubRank, player.HumanQi, player.WallProgress, player.ID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player after ascension"})
		return
	}

	err = CheckReferralMilestones(tx, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check referral milestones: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"player": player, "success": success, "message": msg})
}

// Tribulation handles the heavenly tribulation event for Immortals
func Tribulation(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
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

	if player.Rank < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only Gu Immortals face tribulations"})
		return
	}

	if player.NextTribulationTime == nil || time.Now().Before(*player.NextTribulationTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "It is not yet time for your tribulation"})
		return
	}

	// Trigger a Tribulation Boss encounter
	// We'll set the active combat state to a special boss
	bossHP := float64(player.Rank * 5000)
	bossDamage := float64(player.Rank * 500)
	bossDefense := float64(player.Rank * 200)

	player.ActiveCombat = StartCombat(tx, &models.CombatState{
		EnemyTemplateID: "monster_heavenly_tribulation",
		EnemyName:       "Thiên Kiếp (Heavenly Tribulation)",
		EnemyHP:         bossHP,
		EnemyMaxHP:      bossHP,
		EnemyDamage:     bossDamage,
		EnemyDefense:    bossDefense,
		ChargePoints:    0,
	}, &player)

	updateQuery := `UPDATE players SET active_combat = $1 WHERE id = $2`
	_, err = tx.Exec(updateQuery, player.ActiveCombat, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start tribulation"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"player": player, "message": "Thiên kiếp giáng lâm! Hãy chuẩn bị chiến đấu!"})
}

// CompleteQuestRequest defines body to complete a main quest
type CompleteQuestRequest struct {
	QuestID string `json:"quest_id" binding:"required"`
}

// ClearStress allows paying 100 Primeval Stones to reduce ApertureStress by 50%
func ClearStress(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
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

	if player.PrimevalStones < 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch (Cần 100 Nguyên Thạch)"})
		return
	}

	// Deduct stones and reduce stress
	player.PrimevalStones -= 100
	player.ApertureStress = math.Max(0.0, player.ApertureStress-50.0)

	updateQuery := `
		UPDATE players 
		SET primeval_stones = $1, aperture_stress = $2
		WHERE id = $3
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.ApertureStress, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player stress"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": "Không khiếu đã được tịnh hóa thành công! Giảm 50% áp lực.",
	})
}
