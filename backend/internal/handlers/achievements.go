package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AchievementDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	HasFrame    bool   `json:"has_frame"`
	RewardType  string `json:"reward_type"`
	RewardID    string `json:"reward_id,omitempty"`
	RewardVal   float64 `json:"reward_val"`
	RewardLabel string `json:"reward_label"`
}

var achievementRegistry = []AchievementDef{
	{ID: "first_combat", Name: "Tân Thủ Nhập Môn", Description: "Chiến thắng trận chiến đầu tiên", Icon: "⚔️", RewardType: "primeval_stones", RewardVal: 500, RewardLabel: "500 Nguyên Thạch"},
	{ID: "rank_2", Name: "Luyện Khí Nhập Đạo", Description: "Đạt tới Rank 2", Icon: "🔥", RewardType: "primeval_stones", RewardVal: 1000, RewardLabel: "1.000 Nguyên Thạch"},
	{ID: "rank_3", Name: "Trúc Cơ Tầng 1", Description: "Đạt tới Rank 3", Icon: "🌿", RewardType: "primeval_stones", RewardVal: 3000, RewardLabel: "3.000 Nguyên Thạch"},
	{ID: "rank_4", Name: "Kết Đan Sơ Kỳ", Description: "Đạt tới Rank 4", Icon: "💎", HasFrame: true, RewardType: "primeval_stones", RewardVal: 8000, RewardLabel: "8.000 Nguyên Thạch"},
	{ID: "rank_5", Name: "Kết Đan Đại Thành", Description: "Đạt tới Rank 5", Icon: "💎", HasFrame: true, RewardType: "primeval_stones", RewardVal: 15000, RewardLabel: "15.000 Nguyên Thạch"},
	{ID: "rank_6", Name: "Nguyên Anh Cảnh Giới", Description: "Đạt tới Rank 6 - Tiên Cổ Chi Lộ", Icon: "👑", HasFrame: true, RewardType: "immortal_stones", RewardVal: 50, RewardLabel: "50 Tiên Thạch"},
	{ID: "rank_7", Name: "Hóa Thần Đại Năng", Description: "Đạt tới Rank 7", Icon: "👑", HasFrame: true, RewardType: "immortal_stones", RewardVal: 200, RewardLabel: "200 Tiên Thạch"},
	{ID: "rank_8", Name: "Đại Thừa Chí Tôn", Description: "Đạt tới Rank 8", Icon: "👑", HasFrame: true, RewardType: "immortal_stones", RewardVal: 500, RewardLabel: "500 Tiên Thạch"},
	{ID: "rank_9", Name: "Phi Thăng Chi Cảnh", Description: "Đạt tới Rank 9 - Đỉnh Cao Tiên Đạo", Icon: "🐉", HasFrame: true, RewardType: "immortal_stones", RewardVal: 1000, RewardLabel: "1.000 Tiên Thạch"},
	{ID: "tribulation_1", Name: "Vượt Kiếp Sơ Kỳ", Description: "Vượt qua 1 kiếp nạn", Icon: "⛈️", RewardType: "primeval_stones", RewardVal: 1000, RewardLabel: "1.000 Nguyên Thạch"},
	{ID: "tribulation_5", Name: "Lão Tổ Vượt Kiếp", Description: "Vượt qua 5 kiếp nạn", Icon: "⛈️", HasFrame: true, RewardType: "primeval_stones", RewardVal: 10000, RewardLabel: "10.000 Nguyên Thạch"},
	{ID: "tribulation_10", Name: "Cửu Tử Nhất Sinh", Description: "Vượt qua 10 kiếp nạn", Icon: "☠️", HasFrame: true, RewardType: "immortal_stones", RewardVal: 100, RewardLabel: "100 Tiên Thạch"},
	{ID: "gu_10", Name: "Bách Trùng Chủ", Description: "Sở hữu 10 loại Cổ trùng khác nhau", Icon: "🐛", RewardType: "primeval_stones", RewardVal: 2000, RewardLabel: "2.000 Nguyên Thạch"},
	{ID: "gu_25", Name: "Cổ Đạo Tinh Thông", Description: "Sở hữu 25 loại Cổ trùng khác nhau", Icon: "🦋", RewardType: "primeval_stones", RewardVal: 8000, RewardLabel: "8.000 Nguyên Thạch"},
	{ID: "gu_50", Name: "Vạn Trùng Vương", Description: "Sở hữu 50 loại Cổ trùng khác nhau", Icon: "🐉", HasFrame: true, RewardType: "immortal_stones", RewardVal: 50, RewardLabel: "50 Tiên Thạch"},
	{ID: "explorer_100", Name: "Lãng Khách Giang Hồ", Description: "Khám phá 100 ô địa đồ", Icon: "🗺️", RewardType: "primeval_stones", RewardVal: 1000, RewardLabel: "1.000 Nguyên Thạch"},
	{ID: "explorer_500", Name: "Phong Trần Mã Tặc", Description: "Khám phá 500 ô địa đồ", Icon: "🗺️", RewardType: "primeval_stones", RewardVal: 5000, RewardLabel: "5.000 Nguyên Thạch"},
	{ID: "explorer_1000", Name: "Đại Địa Du Tiên", Description: "Khám phá 1.000 ô địa đồ", Icon: "🌍", HasFrame: true, RewardType: "immortal_stones", RewardVal: 30, RewardLabel: "30 Tiên Thạch"},
	{ID: "millionaire", Name: "Phú Khả Địch Quốc", Description: "Tích lũy 1.000.000 Nguyên Thạch", Icon: "🪙", HasFrame: true, RewardType: "immortal_stones", RewardVal: 100, RewardLabel: "100 Tiên Thạch"},
	{ID: "human_qi_1000", Name: "Nhân Khí Bão Táp", Description: "Đạt 1.000 Nhân Khí", Icon: "💨", RewardType: "primeval_stones", RewardVal: 3000, RewardLabel: "3.000 Nguyên Thạch"},
	{ID: "human_qi_10000", Name: "Nhân Khí Thông Thiên", Description: "Đạt 10.000 Nhân Khí", Icon: "🌪️", HasFrame: true, RewardType: "immortal_stones", RewardVal: 30, RewardLabel: "30 Tiên Thạch"},
	{ID: "soul_awaken", Name: "Hồn Phách Tỉnh Giấc", Description: "Đạt Hồn Phách SOS (SoulHP > 0)", Icon: "👻", RewardType: "primeval_stones", RewardVal: 500, RewardLabel: "500 Nguyên Thạch"},
	{ID: "wall_crack_50", Name: "Nửa Bước Phá Vách", Description: "Phá vách 50%", Icon: "🧱", RewardType: "primeval_stones", RewardVal: 2000, RewardLabel: "2.000 Nguyên Thạch"},
	{ID: "wall_crack_100", Name: "Phá Vách Hoàn Mỹ", Description: "Phá vách 100%", Icon: "💥", HasFrame: true, RewardType: "immortal_stones", RewardVal: 50, RewardLabel: "50 Tiên Thạch"},
}

type PlayerAchievementData struct {
	AchievementID string `json:"achievement_id"`
	ClaimedAt     string `json:"claimed_at"`
}

type AchievementClaimRequest struct {
	AchievementID string `json:"achievement_id" binding:"required"`
}

type AchievementsResponse struct {
	Achievements []AchievementDef          `json:"achievements"`
	Earned       []PlayerAchievementData   `json:"earned"`
	SelectedTitle string                  `json:"selected_title"`
	SelectedFrame string                  `json:"selected_frame"`
}

func GetAchievements(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var earnedJSON []byte
	var selectedTitle, selectedFrame string
	err := db.DB.QueryRow(`SELECT COALESCE(earned_achievements, '[]'::jsonb), COALESCE(selected_title, ''), COALESCE(selected_frame, '') FROM players WHERE id = $1`, playerID).Scan(&earnedJSON, &selectedTitle, &selectedFrame)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}

	var earned []PlayerAchievementData
	if len(earnedJSON) > 0 {
		json.Unmarshal(earnedJSON, &earned)
	}
	if earned == nil {
		earned = []PlayerAchievementData{}
	}

	c.JSON(http.StatusOK, AchievementsResponse{
		Achievements:  achievementRegistry,
		Earned:        earned,
		SelectedTitle: selectedTitle,
		SelectedFrame: selectedFrame,
	})
}

func ClaimAchievement(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req AchievementClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	var ach *AchievementDef
	for i := range achievementRegistry {
		if achievementRegistry[i].ID == req.AchievementID {
			ach = &achievementRegistry[i]
			break
		}
	}
	if ach == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thành tựu không tồn tại"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	var earnedJSON []byte
	var selectedTitle, selectedFrame string
	tx.QueryRow(`SELECT COALESCE(earned_achievements, '[]'::jsonb), COALESCE(selected_title, ''), COALESCE(selected_frame, '') FROM players WHERE id = $1`, playerID).Scan(&earnedJSON, &selectedTitle, &selectedFrame)

	var earned []PlayerAchievementData
	if len(earnedJSON) > 0 {
		json.Unmarshal(earnedJSON, &earned)
	}
	if earned == nil {
		earned = []PlayerAchievementData{}
	}

	for _, e := range earned {
		if e.AchievementID == req.AchievementID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thành tựu này đã được nhận"})
			return
		}
	}

	if !checkAchievementCondition(ach, &player) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa đạt điều kiện thành tựu này"})
		return
	}

	earned = append(earned, PlayerAchievementData{
		AchievementID: req.AchievementID,
		ClaimedAt:     time.Now().Format(time.RFC3339),
	})

	earnedBytes, _ := json.Marshal(earned)

	// Apply reward
	if ach.RewardType == "primeval_stones" {
		player.PrimevalStones += int(ach.RewardVal)
	} else if ach.RewardType == "immortal_stones" {
		player.ImmortalStones += ach.RewardVal
	} else if ach.RewardType == "item" {
		if player.MaterialsInventory == nil {
			player.MaterialsInventory = make(models.MaterialsInventory)
		}
		player.MaterialsInventory[ach.RewardID] += int(ach.RewardVal)
	}

	_, err = tx.Exec(`UPDATE players SET earned_achievements = $1, primeval_stones = $2, immortal_stones = $3, materials_inventory = $4 WHERE id = $5`,
		earnedBytes, player.PrimevalStones, player.ImmortalStones, player.MaterialsInventory, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit"})
		return
	}

	auditlog.Logf(db.DB, &playerID, "achievement.claim", "achievementId=%s name=%s", ach.ID, ach.Name)

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": fmt.Sprintf("Nhận thành tựu [%s] %s! Phần thưởng: %s", ach.Icon, ach.Name, ach.RewardLabel),
	})
}

func checkAchievementCondition(ach *AchievementDef, player *models.Player) bool {
	switch ach.ID {
	case "first_combat":
		return true
	case "rank_2":
		return player.Rank >= 2
	case "rank_3":
		return player.Rank >= 3
	case "rank_4":
		return player.Rank >= 4
	case "rank_5":
		return player.Rank >= 5
	case "rank_6":
		return player.Rank >= 6
	case "rank_7":
		return player.Rank >= 7
	case "rank_8":
		return player.Rank >= 8
	case "rank_9":
		return player.Rank >= 9
	case "tribulation_1":
		return player.TribulationCount >= 1
	case "tribulation_5":
		return player.TribulationCount >= 5
	case "tribulation_10":
		return player.TribulationCount >= 10
	case "gu_10":
		return countDistinctGu(player) >= 10
	case "gu_25":
		return countDistinctGu(player) >= 25
	case "gu_50":
		return countDistinctGu(player) >= 50
	case "explorer_100":
		return len(player.ExploredCells) >= 100
	case "explorer_500":
		return len(player.ExploredCells) >= 500
	case "explorer_1000":
		return len(player.ExploredCells) >= 1000
	case "millionaire":
		return player.PrimevalStones >= 1000000
	case "human_qi_1000":
		return player.HumanQi >= 1000
	case "human_qi_10000":
		return player.HumanQi >= 10000
	case "soul_awaken":
		return player.SoulHP > 0
	case "wall_crack_50":
		return player.WallProgress >= 0.5
	case "wall_crack_100":
		return player.WallProgress >= 1.0
	}
	return false
}

func countDistinctGu(player *models.Player) int {
	seen := make(map[string]bool)
	for _, g := range player.GuInventory {
		seen[g.GuID] = true
	}
	return len(seen)
}

func SelectTitleFrame(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Title string `json:"title"`
		Frame string `json:"frame"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if req.Title != "" {
		valid := false
		for _, a := range achievementRegistry {
			if a.ID == req.Title {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Danh hiệu không tồn tại"})
			return
		}
	}

	if req.Frame != "" {
		valid := false
		for _, a := range achievementRegistry {
			if a.ID == req.Frame && a.HasFrame {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Frame không tồn tại hoặc không phải frame"})
			return
		}
	}

	_, err := db.DB.Exec(`UPDATE players SET selected_title = $1, selected_frame = $2 WHERE id = $3`, req.Title, req.Frame, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật danh hiệu và frame"})
}
