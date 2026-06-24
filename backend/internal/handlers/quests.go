package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type QuestReward struct {
	Type   string  `json:"type"`
	ID     string  `json:"id,omitempty"`
	Amount float64 `json:"amount"`
	Label  string  `json:"label"`
}

type QuestDefinition struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Desc    string        `json:"desc"`
	ReqDesc string        `json:"req_desc"`
	Message string        `json:"message"`
	Rewards []QuestReward `json:"rewards"`
	Check   func(*models.Player) bool
}

type questResponse struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Desc    string        `json:"desc"`
	ReqDesc string        `json:"req_desc"`
	Rewards []QuestReward `json:"rewards"`
}

var questRegistry []QuestDefinition

func init() {
	questRegistry = []QuestDefinition{
		// === CÁC NHIỆM VỤ GỐC CỦA BẠN ===
		{
			ID:      "quest_1_awaken",
			Title:   "Giác Tỉnh Không Khiếu",
			Desc:    "Khai mở không khiếu thành công",
			ReqDesc: "Cảnh giới Nhất Chuyển Sơ Giai trở lên",
			Message: "Hoàn thành nhiệm vụ 'Giác Tỉnh Không Khiếu'! Đạo hải dâng trào.",
			Check:   func(p *models.Player) bool { return p.Rank >= 1 },
			Rewards: []QuestReward{
				{Type: "gu", ID: "GU_GREEN_SILK", Amount: 1, Label: "Thanh Ti Cổ"},
				{Type: "item", ID: "ITEM_HEALING_PELLET_SMALL", Amount: 3, Label: "3 Tiểu Hồi Huyết Đan"},
			},
		},
		{
			ID:      "quest_2_vital_gu",
			Title:   "Cổ Trùng Bản Mệnh",
			Desc:    "Luyện hóa hoặc trang bị Cổ trùng",
			ReqDesc: "Đang trang bị hoặc có Cổ trùng bản mệnh",
			Message: "Hoàn thành nhiệm vụ 'Kích Hoạt Cổ Trùng Bản Mệnh'! Sức mạnh liên kết.",
			Check: func(p *models.Player) bool {
				for _, g := range p.GuInventory {
					if g.IsVital || g.Equipped {
						return true
					}
				}
				return false
			},
			Rewards: []QuestReward{
				{Type: "gu", ID: "GU_WHITE_JADE", Amount: 1, Label: "Bạch Ngọc Cổ"},
				{Type: "item", ID: "ITEM_MEDICINAL_PASTE", Amount: 2, Label: "2 Kim Sang Dược"},
			},
		},
		{
			ID:      "quest_3_rank_2",
			Title:   "Trùng Kích Nhị Chuyển",
			Desc:    "Đột phá vách khiếu thăng Nhị Chuyển",
			ReqDesc: "Tu vi đạt Nhị Chuyển",
			Message: "Hoàn thành nhiệm vụ 'Đột Phá Nhị Chuyển'! Danh vọng vang xa.",
			Check:   func(p *models.Player) bool { return p.Rank >= 2 },
			Rewards: []QuestReward{
				{Type: "gu", ID: "GU_WATER_SHIELD", Amount: 1, Label: "Thủy Tráo Cổ"},
				{Type: "item", ID: "ITEM_HEALING_PELLET_MEDIUM", Amount: 3, Label: "3 Trung Hồi Huyết Đan"},
			},
		},
		{
			ID:      "quest_4_refine_gu",
			Title:   "Luyện Chế Cổ Trùng Nhị Chuyển",
			Desc:    "Luyện thành công Cổ trùng bậc 2 trở lên",
			ReqDesc: "Có Cổ trùng bậc 2 trở lên",
			Message: "Hoàn thành nhiệm vụ 'Hợp Luyện Nhị Chuyển Cổ'! Kỳ tích luyện đạo.",
			Check: func(p *models.Player) bool {
				for _, g := range p.GuInventory {
					if spec, ok := models.GuRegistry[g.GuID]; ok && spec.Rank >= 2 {
						return true
					}
				}
				return false
			},
			Rewards: []QuestReward{
				{Type: "gu", ID: "GU_CANOPY", Amount: 1, Label: "Thiên Bồng Cổ"},
				{Type: "item", ID: "ITEM_BLOOD_GINSENG", Amount: 2, Label: "2 Huyết Sâm"},
			},
		},
		{
			ID:      "quest_5_fame",
			Title:   "Danh Chấn Thanh Bản Thôn",
			Desc:    "Chiến thắng quái thú dọn sạch 5 địa khu",
			ReqDesc: "Đã quét dọn 5/5 ô bản đồ",
			Message: "Hoàn thành nhiệm vụ 'Danh Chấn Thanh Bản Thôn'! Người người kính ngưỡng.",
			Check:   func(p *models.Player) bool { return len(p.ClearedCells) >= 5 },
			Rewards: []QuestReward{
				{Type: "gu", ID: "GU_FLYING_BONE_SHIELD", Amount: 1, Label: "Phi Cốt Thuẫn Cổ"},
				{Type: "item", ID: "mine_chest", Amount: 3, Label: "3 Rương Mỏ Linh Thạch"},
			},
		},
		{
			ID:      "quest_6_rank_5",
			Title:   "Đỉnh Phong Chuẩn Bị",
			Desc:    "Tu vi đỉnh phong, sẵn sàng độ kiếp thăng tiên",
			ReqDesc: "Tu vi đạt Ngũ Chuyển",
			Message: "Hoàn thành nhiệm vụ 'Đỉnh Phong Chuẩn Bị'! Cận kề thăng tiên.",
			Check:   func(p *models.Player) bool { return p.Rank >= 5 },
			Rewards: []QuestReward{
				{Type: "gu", ID: "GU_GOLDEN_BELL_SHIELD", Amount: 1, Label: "Kim Chung Tráo Cổ"},
				{Type: "item", ID: "ITEM_BLOOD_GINSENG", Amount: 5, Label: "5 Huyết Sâm"},
			},
		},

		// ==========================================
		// NHIỆM VỤ LINH THẠCH / NGUYÊN THẠCH (PRIMEVAL & IMMORTAL STONES)
		// ==========================================
		{
			ID:      "quest_7_stone_hoarder",
			Title:   "Tích Lũy Nguyên Thạch",
			Desc:    "Tích lũy vốn liếng ban đầu để bảo đảm hậu cần sinh tồn",
			ReqDesc: "Trong người sở hữu từ 5,000 Nguyên Thạch trở lên",
			Message: "Hoàn thành nhiệm vụ 'Tích Lũy Nguyên Thạch'! Cơ sở tài chính sống còn vững chắc.",
			Check:   func(p *models.Player) bool { return p.PrimevalStones >= 5000 },
			Rewards: []QuestReward{
				{Type: "item", ID: "ruou_thanh_truc", Amount: 5, Label: "5 Rượu Thanh Trúc"}, // Thức ăn cho Tửu Trùng
			},
		},
		{
			ID:      "quest_8_immortal_capital",
			Title:   "Nội Tình Tiên Khiếu",
			Desc:    "Tích lũy tài sản chiến lược để giao dịch trên Bảo Hoàng Thiên",
			ReqDesc: "Sở hữu từ 100 viên Tiên Nguyên Thạch trở lên",
			Message: "Hoàn thành nhiệm vụ 'Nội Tình Tiên Khiếu'! Thạch lượng dồi dào, bắt đầu có tiếng nói giữa các Cổ Tiên.",
			Check:   func(p *models.Player) bool { return p.ImmortalStones >= 100.0 },
			Rewards: []QuestReward{
				{Type: "dao_mark", ID: "refinement_path", Amount: 10, Label: "10 Đạo Ngân Luyện Đạo"},
			},
		},

		// ==========================================
		// NHIỆM VỤ CỔ TRÙNG (GU INVENTORY MECHANICALS)
		// ==========================================
		{
			ID:      "quest_9_liquor_worm",
			Title:   "Săn Tìm Tửu Trùng",
			Desc:    "Thu thập thần trùng có khả năng tinh luyện chân nguyên tiểu cảnh giới",
			ReqDesc: "Sở hữu Cổ trùng 'Tửu Trùng' (GU_LIQUOR_WORM) trong Không Khiếu",
			Message: "Hoàn thành nhiệm vụ 'Săn Tìm Tửu Trùng'! Chân nguyên được tinh luyện vượt cấp!",
			Check: func(p *models.Player) bool {
				for _, g := range p.GuInventory {
					if g.GuID == "GU_LIQUOR_WORM" {
						return true
					}
				}
				return false
			},
			Rewards: []QuestReward{
				{Type: "gu", ID: "GU_RIB_SHIELD", Amount: 1, Label: "Lặc Cốt Thuẫn Cổ"},
				{Type: "item", ID: "ITEM_HEALING_PELLET_MEDIUM", Amount: 5, Label: "5 Trung Hồi Huyết Đan"},
			},
		},
		{
			ID:      "quest_10_storage_tusita",
			Title:   "Khai Mở Hoa Đâu Suất",
			Desc:    "Sở hữu Cổ tồn trữ để giải quyết bài toán giới hạn không gian túi đồ",
			ReqDesc: "Sở hữu Cổ trùng 'Đâu Suất Hoa' (GU_TUSITA_FLOWER) trong Không Khiếu",
			Message: "Hoàn thành nhiệm vụ 'Khai Mở Hoa Đâu Suất'! Sức chứa túi đồ tăng mạnh, thoải mái tích trữ tài nguyên.",
			Check: func(p *models.Player) bool {
				for _, g := range p.GuInventory {
					if g.GuID == "GU_TUSITA_FLOWER" {
						return true
					}
				}
				return false
			},
			Rewards: []QuestReward{
				{Type: "primeval_stones", Amount: 1000, Label: "1000 Nguyên Thạch"},
			},
		},
		{
			ID:      "quest_11_gu_collector",
			Title:   "Vạn Vật Chi Linh",
			Desc:    "Nuôi dưỡng và thu thập một hệ sinh thái Cổ trùng đa dạng",
			ReqDesc: "Sở hữu từ 10 Cổ trùng trở lên trong cơ thể",
			Message: "Hoàn thành nhiệm vụ 'Vạn Vật Chi Linh'! Không khiếu của ngươi vô cùng phong phú.",
			Check:   func(p *models.Player) bool { return len(p.GuInventory) >= 10 },
			Rewards: []QuestReward{
				{Type: "gu", ID: "GU_IRON_BONE", Amount: 1, Label: "Thiết Cốt Cổ"},
				{Type: "item", ID: "mine_chest", Amount: 5, Label: "5 Rương Mỏ Linh Thạch"},
			},
		},
		{
			ID:      "quest_12_immortal_gu_owner",
			Title:   "Tiên Cổ Độc Nhất",
			Desc:    "Bứt phá phàm nhân, luyện hóa chí bảo chứa đựng một Đạo hoàn chỉnh",
			ReqDesc: "Sở hữu ít nhất 1 con Tiên Cổ (Phẩm cấp/Rank >= 6 trở lên)",
			Message: "Hoàn thành nhiệm vụ 'Tiên Cổ Độc Nhất'! Thiên địa rung chuyển, Tiên cổ duy nhất xuất thế!",
			Check: func(p *models.Player) bool {
				for _, g := range p.GuInventory {
					if spec, ok := models.GuRegistry[g.GuID]; ok && spec.Rank >= 6 {
						return true
					}
				}
				return false
			},
			Rewards: []QuestReward{
				{Type: "immortal_stones", Amount: 20, Label: "20 Tiên Nguyên Thạch"},
				{Type: "dao_mark", ID: "luck_path", Amount: 5, Label: "5 Đạo Ngân Vận Đạo"},
			},
		},

		// ==========================================
		// NHIỆM VỤ VẬT PHẨM / NGUYÊN LIỆU (MATERIALS INVENTORY)
		// ==========================================
		{
			ID:      "quest_13_gather_lan_flower",
			Title:   "Tích Trữ Nguyệt Lan Hoa",
			Desc:    "Thu thập cánh hoa nguyệt lan làm thức ăn tiêu hao định kỳ cho Nguyệt Quang Cổ",
			ReqDesc: "Túi đồ (MaterialsInventory) có ít nhất 50 Nguyệt Lan Hoa (nguyet_lan_hoa)",
			Message: "Hoàn thành nhiệm vụ 'Tích Trữ Nguyệt Lan Hoa'! Giải quyết thành công bài toán nuôi Cổ trùng sống sót.",
			Check: func(p *models.Player) bool {
				if p.MaterialsInventory == nil {
					return false
				}
				return p.MaterialsInventory["nguyet_lan_hoa"] >= 50
			},
			Rewards: []QuestReward{
				{Type: "primeval_stones", Amount: 500, Label: "500 Nguyên Thạch"},
			},
		},
		{
			ID:      "quest_14_immortal_material",
			Title:   "Thu Hoạch Thất Chuyển Tiên Tài",
			Desc:    "Khám phá hiểm cảnh, thu thập nguyên liệu cao cấp chứa đựng Đạo Ngân dồi dào",
			ReqDesc: "Trong túi đồ sở hữu từ 1 'Tử Kim Thần Bí' (tu_kim_than_bi) trở lên",
			Message: "Hoàn thành nhiệm vụ 'Thu Hoạch Thất Chuyển Tiên Tài'! Khối tử kim thạch này là phôi báu tuyệt vời để luyện Tiên Cổ.",
			Check: func(p *models.Player) bool {
				if p.MaterialsInventory == nil {
					return false
				}
				return p.MaterialsInventory["tu_kim_than_bi"] >= 1
			},
			Rewards: []QuestReward{
				{Type: "immortal_stones", Amount: 10, Label: "10 Tiên Nguyên Thạch"},
			},
		},
	}
}

func GetQuestDefinitions() []QuestDefinition {
	return questRegistry
}

func FindQuest(id string) *QuestDefinition {
	for _, q := range questRegistry {
		if q.ID == id {
			return &q
		}
	}
	return nil
}

func buildRewardLabel(rewards []QuestReward) string {
	var parts []string
	for _, r := range rewards {
		if r.Label != "" {
			parts = append(parts, r.Label)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " Nhận được " + strings.Join(parts, ", ") + "!"
}

func applyQuestRewardsTx(tx *sql.Tx, player *models.Player, rewards []QuestReward) error {
	for _, r := range rewards {
		switch r.Type {
		case "human_qi":
			player.HumanQi += r.Amount
			if player.HumanQi > 100000.0 {
				player.HumanQi = 100000.0
			}
		case "primeval_stones":
			player.PrimevalStones += int(r.Amount)
		case "immortal_stones":
			player.ImmortalStones += r.Amount
		case "dao_mark":
			if player.DaoMarks == nil {
				player.DaoMarks = make(models.DaoMarksMap)
			}
			player.DaoMarks[r.ID] += int(r.Amount)
		case "item":
			if player.MaterialsInventory == nil {
				player.MaterialsInventory = make(models.MaterialsInventory)
			}
			player.MaterialsInventory[r.ID] += int(r.Amount)
		case "gu":
			newGu := models.GuInstance{
				InstanceID: "gu-" + uuid.New().String(),
				GuID:       r.ID,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
			}
			player.GuInventory = append(player.GuInventory, newGu)
		}
	}
	return nil
}

func buildQuestUpdateQuery(player *models.Player) (string, []interface{}) {
	baseFields := []string{"human_qi", "completed_quests"}
	baseVals := []interface{}{player.HumanQi, player.CompletedQuests}
	extraFields := []string{}
	extraVals := []interface{}{}

	if player.PrimevalStones != 0 {
		extraFields = append(extraFields, "primeval_stones")
		extraVals = append(extraVals, player.PrimevalStones)
	}
	if player.ImmortalStones != 0 {
		extraFields = append(extraFields, "immortal_stones")
		extraVals = append(extraVals, player.ImmortalStones)
	}
	if len(player.DaoMarks) > 0 {
		extraFields = append(extraFields, "dao_marks")
		extraVals = append(extraVals, player.DaoMarks)
	}
	if len(player.MaterialsInventory) > 0 {
		extraFields = append(extraFields, "materials_inventory")
		extraVals = append(extraVals, player.MaterialsInventory)
	}
	if len(player.GuInventory) > 0 {
		extraFields = append(extraFields, "gu_inventory")
		extraVals = append(extraVals, player.GuInventory)
	}

	allFields := append(baseFields, extraFields...)
	allVals := append(baseVals, extraVals...)

	placeholders := make([]string, len(allFields))
	for i := range allFields {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	setClauses := make([]string, len(allFields))
	for i, f := range allFields {
		setClauses[i] = fmt.Sprintf("%s = %s", f, placeholders[i])
	}

	query := fmt.Sprintf("UPDATE players SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "),
		len(allFields)+1,
	)
	allVals = append(allVals, player.ID)

	return query, allVals
}

func ListQuests(c *gin.Context) {
	resp := make([]questResponse, 0, len(questRegistry))
	for _, q := range questRegistry {
		resp = append(resp, questResponse{
			ID:      q.ID,
			Title:   q.Title,
			Desc:    q.Desc,
			ReqDesc: q.ReqDesc,
			Rewards: q.Rewards,
		})
	}
	c.JSON(http.StatusOK, gin.H{"quests": resp})
}

func CompleteQuest(c *gin.Context) {
	var req CompleteQuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	quest := FindQuest(req.QuestID)
	if quest == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nhiệm vụ không tồn tại"})
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

	for _, q := range player.CompletedQuests {
		if q == req.QuestID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nhiệm vụ này đã được hoàn thành trước đó"})
			return
		}
	}

	if !quest.Check(&player) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa đạt yêu cầu: " + quest.ReqDesc})
		return
	}

	if err := applyQuestRewardsTx(tx, &player, quest.Rewards); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply rewards"})
		return
	}

	player.CompletedQuests = append(player.CompletedQuests, req.QuestID)

	updateQuery, updateArgs := buildQuestUpdateQuery(&player)
	_, err = tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player"})
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

	rewardStr := buildRewardLabel(quest.Rewards)
	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": quest.Message + rewardStr,
	})
}
