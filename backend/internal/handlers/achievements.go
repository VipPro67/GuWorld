package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type AchievementDef struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	HasFrame    bool            `json:"has_frame"`
	RewardType  string          `json:"reward_type"`
	RewardID    string          `json:"reward_id,omitempty"`
	RewardVal   float64         `json:"reward_val"`
	RewardLabel string          `json:"reward_label"`
	TitleEffect *TitleEffectDef `json:"title_effect,omitempty"`
}

type TitleEffectDef struct {
	PveMonsterType string  `json:"pve_monster_type"`
	PveDamagePct   float64 `json:"pve_damage_pct"`
	UtilityType    string  `json:"utility_type"`
	UtilityPct     float64 `json:"utility_pct"`
}

var achievementRegistry = []AchievementDef{
	{ID: "first_combat", Name: "Sơ Chiến Thành Danh", Description: "Chiến thắng trận chiến đầu tiên với tư cách một Cổ Sư", Icon: "⚔️", RewardType: "primeval_stones", RewardVal: 50, RewardLabel: "50 Nguyên Thạch"},

	{ID: "rank_2", Name: "Nhị Chuyển Cổ Sư", Description: "Đột phá tới Nhị Chuyển Cổ Sư", Icon: "🔥", RewardType: "primeval_stones", RewardVal: 100, RewardLabel: "100 Nguyên Thạch"},
	{ID: "rank_3", Name: "Tam Chuyển Cường Giả", Description: "Đột phá tới Tam Chuyển Cổ Sư", Icon: "🌿", RewardType: "primeval_stones", RewardVal: 300, RewardLabel: "300 Nguyên Thạch"},
	{ID: "rank_4", Name: "Tứ Chuyển Bá Chủ", Description: "Bước vào cảnh giới Tứ Chuyển", Icon: "💎", HasFrame: true, RewardType: "primeval_stones", RewardVal: 8000, RewardLabel: "8.000 Nguyên Thạch"},
	{ID: "rank_5", Name: "Ngũ Chuyển Tông Sư", Description: "Trở thành Ngũ Chuyển Cổ Sư", Icon: "💎", HasFrame: true, RewardType: "primeval_stones", RewardVal: 15000, RewardLabel: "15.000 Nguyên Thạch"},
	{ID: "rank_6", Name: "Lục Chuyển Cổ Tiên", Description: "Khai mở Tiên Khiếu, bước vào hàng ngũ Cổ Tiên", Icon: "👑", HasFrame: true, RewardType: "immortal_stones", RewardVal: 50, RewardLabel: "50 Tiên Nguyên Thạch"},
	{ID: "rank_7", Name: "Thất Chuyển Đại Năng", Description: "Đạt tới Thất Chuyển Cổ Tiên", Icon: "👑", HasFrame: true, RewardType: "immortal_stones", RewardVal: 200, RewardLabel: "200 Tiên Nguyên Thạch"},
	{ID: "rank_8", Name: "Bát Chuyển Tôn Giả", Description: "Đứng trên đỉnh cao Bát Chuyển", Icon: "👑", HasFrame: true, RewardType: "immortal_stones", RewardVal: 500, RewardLabel: "500 Tiên Nguyên Thạch"},
	{ID: "rank_9", Name: "Cửu Chuyển Tôn Giả", Description: "Chạm tới cảnh giới tối cao của thiên địa", Icon: "🐉", HasFrame: true, RewardType: "immortal_stones", RewardVal: 1000, RewardLabel: "1.000 Tiên Nguyên Thạch"},

	{ID: "tribulation_1", Name: "Lần Đầu Độ Kiếp", Description: "Vượt qua một lần thiên kiếp", Icon: "⛈️", RewardType: "immortal_stones", RewardVal: 100, RewardLabel: "100 Tiên Nguyên Thạch"},
	{ID: "tribulation_5", Name: "Ngũ Kiếp Bất Diệt", Description: "Vượt qua năm lần thiên kiếp", Icon: "⛈️", HasFrame: true, RewardType: "immortal_stones", RewardVal: 1000, RewardLabel: "1.000 Tiên Nguyên Thạch"},
	{ID: "tribulation_10", Name: "Thiên Kiếp Bất Khuất", Description: "Vượt qua mười lần thiên kiếp", Icon: "☠️", HasFrame: true, RewardType: "immortal_stones", RewardVal: 10000, RewardLabel: "10.000 Tiên Nguyên Thạch"},

	{ID: "gu_10", Name: "Tiểu Cổ Tàng Gia", Description: "Thu thập 10 loại Cổ trùng khác nhau", Icon: "🐛", RewardType: "primeval_stones", RewardVal: 200, RewardLabel: "200 Nguyên Thạch"},
	{ID: "gu_25", Name: "Cổ Đạo Chuyên Gia", Description: "Thu thập 25 loại Cổ trùng khác nhau", Icon: "🦋", RewardType: "primeval_stones", RewardVal: 800, RewardLabel: "800 Nguyên Thạch"},
	{ID: "gu_50", Name: "Vạn Cổ Chi Chủ", Description: "Thu thập 50 loại Cổ trùng khác nhau", Icon: "🐉", HasFrame: true, RewardType: "primeval_stones", RewardVal: 2000, RewardLabel: "2.000 Nguyên Thạch"},

	{ID: "explorer_100", Name: "Lữ Hành Giả", Description: "Khám phá 100 vùng đất nhỏ", Icon: "🗺️", RewardType: "primeval_stones", RewardVal: 1000, RewardLabel: "1.000 Nguyên Thạch"},
	{ID: "explorer_500", Name: "Chu Du Ngũ Vực", Description: "Khám phá 500 vùng đất nhỏ", Icon: "🗺️", RewardType: "primeval_stones", RewardVal: 5000, RewardLabel: "5.000 Nguyên Thạch"},
	{ID: "explorer_1000", Name: "Dấu Chân Thiên Hạ", Description: "Chinh phục toàn bộ 600 vùng đất nhỏ", Icon: "🌍", HasFrame: true, RewardType: "immortal_stones", RewardVal: 30, RewardLabel: "30 Tiên Nguyên Thạch"},

	{ID: "millionaire", Name: "Phú Giáp Thiên Hạ", Description: "Tích lũy 1.000.000 Nguyên Thạch", Icon: "🪙", HasFrame: true, RewardType: "immortal_stones", RewardVal: 1000, RewardLabel: "1.000 Tiên Nguyên Thạch"},

	{ID: "human_qi_1000", Name: "Nhân Khí Hưng Thịnh", Description: "Đạt 1.000 Nhân Khí", Icon: "💨", RewardType: "primeval_stones", RewardVal: 3000, RewardLabel: "3.000 Nguyên Thạch"},
	{ID: "human_qi_10000", Name: "Nhân Khí Thông Thiên", Description: "Đạt 10.000 Nhân Khí", Icon: "🌪️", HasFrame: true, RewardType: "immortal_stones", RewardVal: 30, RewardLabel: "30 Tiên Nguyên Thạch"},
	{ID: "human_qi_100000", Name: "Nhân Khí Vạn Vật", Description: "Đạt 100.000 Nhân Khí", Icon: "🌪️", HasFrame: true, RewardType: "immortal_stones", RewardVal: 300, RewardLabel: "300 Tiên Nguyên Thạch"},

	{ID: "soul_awaken", Name: "Hồn Phách Hiển Hiện", Description: "Hồn phách thức tỉnh (SoulHP >= 100)", Icon: "👻", RewardType: "primeval_stones", RewardVal: 500, RewardLabel: "500 Nguyên Thạch"},

	{ID: "wall_crack_50", Name: "Bích Liệt Tiên Phong", Description: "Xung kích bích chướng Không Khiếu đạt 50%", Icon: "⚡", RewardType: "primeval_stones", RewardVal: 500, RewardLabel: "500 Nguyên Thạch"},
	{ID: "wall_crack_100", Name: "Phá Bích Đột Phá", Description: "Xung kích bích chướng Không Khiếu đạt 100%", Icon: "💥", HasFrame: true, RewardType: "primeval_stones", RewardVal: 1000, RewardLabel: "1.000 Nguyên Thạch"},
}

func init() {
	for i := range achievementRegistry {
		achievementRegistry[i].TitleEffect = titleEffectForAchievement(achievementRegistry[i].ID)
	}
}

func titleEffectForAchievement(id string) *TitleEffectDef {
	monsterType := "Normal"
	switch id {
	case "rank_2", "explorer_100", "explorer_500", "explorer_1000", "wall_crack_50", "wall_crack_100":
		monsterType = "Normal"
	case "rank_3", "rank_4", "gu_10", "gu_25", "gu_50", "human_qi_1000", "human_qi_10000", "human_qi_100000":
		monsterType = "Elite"
	case "rank_5", "rank_6", "rank_8", "rank_9", "millionaire":
		monsterType = "Boss"
	case "rank_7", "tribulation_1", "tribulation_5", "tribulation_10":
		monsterType = "Tribulation_Boss"
	case "soul_awaken":
		monsterType = "Tribulation_Monster"
	case "first_combat":
		monsterType = "Normal"
	}
	return &TitleEffectDef{PveMonsterType: monsterType, PveDamagePct: 2, UtilityType: "aperture_yield", UtilityPct: 5}
}

type PlayerAchievementData struct {
	AchievementID string `json:"achievement_id"`
	ClaimedAt     string `json:"claimed_at"`
}

type AchievementClaimRequest struct {
	AchievementID string `json:"achievement_id" binding:"required"`
}

type AchievementsResponse struct {
	Achievements  []AchievementDef        `json:"achievements"`
	Earned        []PlayerAchievementData `json:"earned"`
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
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(earned_achievements, '[]'::jsonb), COALESCE(selected_title, ''), COALESCE(selected_frame, '') FROM players WHERE id = $1`, playerID).Scan(&earnedJSON, &selectedTitle, &selectedFrame)
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
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT achievement_id, unlocked_at FROM player_achievements WHERE player_id = $1`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải danh sách thành tựu"})
		return
	}
	for rows.Next() {
		var entry PlayerAchievementData
		var unlockedAt time.Time
		if err := rows.Scan(&entry.AchievementID, &unlockedAt); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc thành tựu"})
			return
		}
		entry.ClaimedAt = unlockedAt.Format(time.RFC3339)
		found := false
		for _, existing := range earned {
			if existing.AchievementID == entry.AchievementID {
				found = true
				break
			}
		}
		if !found {
			earned = append(earned, entry)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc thành tựu"})
		return
	}
	rows.Close()

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

	player, err := PlayerSvc.Load(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	var earnedJSON []byte
	if err := tx.QueryRow(`SELECT COALESCE(earned_achievements, '[]'::jsonb) FROM players WHERE id = $1`, playerID).Scan(&earnedJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc thành tựu"})
		return
	}
	var earned []PlayerAchievementData
	if len(earnedJSON) > 0 {
		if err := json.Unmarshal(earnedJSON, &earned); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu thành tựu không hợp lệ"})
			return
		}
	}

	for _, e := range earned {
		if e.AchievementID == req.AchievementID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thành tựu này đã được nhận"})
			return
		}
	}
	var adminGranted bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM player_achievements WHERE player_id = $1 AND achievement_id = $2)`, playerID, req.AchievementID).Scan(&adminGranted); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc thành tựu"})
		return
	}
	if adminGranted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thành tựu này đã được nhận"})
		return
	}

	conditionMet, conditionErr := checkAchievementCondition(tx, ach, &player)
	if conditionErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra điều kiện thành tựu"})
		return
	}
	if !conditionMet {
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
		player.PrimevalStones += float64(ach.RewardVal)
	} else if ach.RewardType == "immortal_stones" {
		player.ImmortalStones += float64(ach.RewardVal)
	} else if ach.RewardType == "item" {
		if err := InventorySvc.AddItem(tx, playerID.String(), ach.RewardID, "materials", int64(ach.RewardVal)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật vật phẩm"})
			return
		}
	}
	if _, err := tx.Exec(`INSERT INTO player_achievements (player_id, achievement_id) VALUES ($1, $2) ON CONFLICT (player_id, achievement_id) DO NOTHING`, playerID, req.AchievementID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu thành tựu"})
		return
	}

	qiGained := 0.0

	_, err = tx.Exec(`UPDATE players SET earned_achievements = $1, primeval_stones = $2, immortal_stones = $3, human_qi = $4 WHERE id = $5`,
		earnedBytes, player.PrimevalStones, player.ImmortalStones, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit"})
		return
	}

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Nhận thành tựu [%s] %s! Phần thưởng: %s Nhân Khí +%.0f.", ach.Icon, ach.Name, ach.RewardLabel, qiGained),
	})
}

func checkAchievementCondition(tx *sql.Tx, ach *AchievementDef, player *models.Player) (bool, error) {
	switch ach.ID {
	case "first_combat":
		return len(player.ClearedCells) > 0, nil
	case "rank_2":
		return player.Rank >= 2, nil
	case "rank_3":
		return player.Rank >= 3, nil
	case "rank_4":
		return player.Rank >= 4, nil
	case "rank_5":
		return player.Rank >= 5, nil
	case "rank_6":
		return player.Rank >= 6, nil
	case "rank_7":
		return player.Rank >= 7, nil
	case "rank_8":
		return player.Rank >= 8, nil
	case "rank_9":
		return player.Rank >= 9, nil
	case "tribulation_1":
		return achievementTribulationCount(tx, player.ID, 1)
	case "tribulation_5":
		return achievementTribulationCount(tx, player.ID, 5)
	case "tribulation_10":
		return achievementTribulationCount(tx, player.ID, 10)
	case "gu_10":
		return countDistinctGu(player) >= 10, nil
	case "gu_25":
		return countDistinctGu(player) >= 25, nil
	case "gu_50":
		return countDistinctGu(player) >= 50, nil
	case "explorer_100":
		return len(player.VisitedSubZones) >= 100, nil
	case "explorer_500":
		return len(player.VisitedSubZones) >= 500, nil
	case "explorer_1000":
		return len(player.VisitedSubZones) >= 600, nil
	case "millionaire":
		return player.PrimevalStones >= 1000000, nil
	case "human_qi_1000":
		return player.HumanQi >= 1000, nil
	case "human_qi_10000":
		return player.HumanQi >= 10000, nil
	case "human_qi_100000":
		return player.HumanQi >= 100000, nil
	case "soul_awaken":
		return player.SoulHP >= 100, nil
	case "wall_crack_50":
		return player.WallProgress >= 0.5, nil
	case "wall_crack_100":
		return player.WallProgress >= 1.0, nil
	}
	return false, nil
}

func achievementTribulationCount(tx *sql.Tx, playerID uuid.UUID, target int) (bool, error) {
	var count int
	err := tx.QueryRow(`SELECT COALESCE(earth_calamity_count,0)+COALESCE(heavenly_tribulation_count,0)+COALESCE(grand_tribulation_count,0)+COALESCE(myriad_tribulation_count,0)+COALESCE(chaos_calamity_count,0) FROM immortal_apertures WHERE player_id=$1`, playerID).Scan(&count)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return count >= target, nil
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
		if !playerOwnsAchievement(c, playerID.String(), req.Title) {
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
		if !playerOwnsAchievement(c, playerID.String(), req.Frame) {
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

func playerOwnsAchievement(c *gin.Context, playerID, achievementID string) bool {
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM player_achievements WHERE player_id = $1 AND achievement_id = $2)`, playerID, achievementID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc thành tựu"})
		return false
	}
	if exists {
		return true
	}
	var earnedJSON []byte
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(earned_achievements, '[]'::jsonb) FROM players WHERE id = $1`, playerID).Scan(&earnedJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc thành tựu"})
		return false
	}
	var earned []PlayerAchievementData
	if err := json.Unmarshal(earnedJSON, &earned); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu thành tựu không hợp lệ"})
		return false
	}
	for _, entry := range earned {
		if entry.AchievementID == achievementID {
			return true
		}
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "Bạn chưa mở khóa danh hiệu hoặc frame này"})
	return false
}
