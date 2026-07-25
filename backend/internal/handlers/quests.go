package handlers

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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

func csvPath(filename string) string {
	candidates := []string{
		filepath.Join("data", filename),
		filename,
		filepath.Join("backend", filename),
		filepath.Join("..", filename),
		filepath.Join("..", "..", filename),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filename
}

func CheckQuest(questID string, p *models.Player) bool {
	switch questID {
	case "quest_1_awaken":
		return p.Rank >= 1
	case "quest_2_vital_gu":
		for _, g := range p.GuInventory {
			if g.IsVital || g.Equipped {
				return true
			}
		}
		return false
	case "quest_3_rank_2":
		return p.Rank >= 2
	case "quest_4_refine_gu":
		for _, g := range p.GuInventory {
			if spec, ok := models.GetGuAny(g.GuID); ok && spec.Rank >= 2 {
				return true
			}
		}
		return false
	case "quest_5_fame":
		return len(p.ClearedCells) >= 5
	case "quest_6_rank_5":
		return p.Rank >= 5
	case "quest_7_stone_hoarder":
		return p.PrimevalStones >= 5000
	case "quest_8_immortal_capital":
		return p.ImmortalStones >= 100.0
	case "quest_9_liquor_worm":
		for _, g := range p.GuInventory {
			if g.GuID == "GU_LIQUOR_WORM" {
				return true
			}
		}
		return false
	case "quest_10_storage_tusita":
		for _, g := range p.GuInventory {
			if g.GuID == "GU_TUSITA_FLOWER" {
				return true
			}
		}
		return false
	case "quest_11_gu_collector":
		return len(p.GuInventory) >= 10
	case "quest_12_immortal_gu_owner":
		for _, g := range p.GuInventory {
			if spec, ok := models.GetGuAny(g.GuID); ok && spec.Rank >= 6 {
				return true
			}
		}
		return false
	case "quest_13_gather_lan_flower":
		qty, _ := InventorySvc.GetQuantity(db.DB, p.ID.String(), "nguyet_lan_hoa", "materials")
		return qty >= 50
	case "quest_14_immortal_material":
		qty, _ := InventorySvc.GetQuantity(db.DB, p.ID.String(), "tu_kim_than_bi", "materials")
		return qty >= 1
	default:
		return false
	}
}

// LoadQuestRegistry loads quest definitions from DB table (if populated) or falls back to CSV.
func LoadQuestRegistry() error {
	loadedQuests, err := loadQuestsFromDB()
	if err == nil && len(loadedQuests) > 0 {
		questRegistry = loadedQuests
		logger.S.Infof("Successfully loaded %d quests from database", len(questRegistry))
		return nil
	}

	// Fallback: load from CSV (used on first startup before import)
	return loadQuestsFromCSV()
}

// loadQuestsFromDB reads quest definitions from the quests table.
func loadQuestsFromDB() ([]QuestDefinition, error) {
	rows, err := db.DB.Query(`SELECT id, title, description, req_desc, message, rewards::text FROM quests ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var loaded []QuestDefinition
	for rows.Next() {
		var id, title, desc, reqDesc, message, rewardsStr string
		if err := rows.Scan(&id, &title, &desc, &reqDesc, &message, &rewardsStr); err != nil {
			return nil, err
		}
		var rewards []QuestReward
		if rewardsStr != "" && rewardsStr != "[]" {
			if err := json.Unmarshal([]byte(rewardsStr), &rewards); err != nil {
				logger.S.Warnf("failed to parse rewards JSON for quest %s: %v", id, err)
			}
		}
		qID := id
		loaded = append(loaded, QuestDefinition{
			ID:      qID,
			Title:   title,
			Desc:    desc,
			ReqDesc: reqDesc,
			Message: message,
			Rewards: rewards,
			Check: func(p *models.Player) bool {
				return CheckQuest(qID, p)
			},
		})
	}
	return loaded, nil
}

// loadQuestsFromCSV is the original CSV-based quest loader (kept as fallback).
func loadQuestsFromCSV() error {
	path := csvPath("Quest.csv")
	file, err := os.Open(path)
	if err != nil {
		logger.S.Warnf("no Quest.csv found, quest registry will be empty: %v", err)
		return nil
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read Quest.csv: %w", err)
	}

	if len(records) < 2 {
		logger.S.Warn("Quest.csv is empty, quest registry will be empty")
		return nil
	}

	headers := records[0]
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

	requiredCols := []string{"id", "title", "desc", "req_desc", "message", "rewards"}
	for _, col := range requiredCols {
		if _, ok := headerMap[col]; !ok {
			return fmt.Errorf("Quest.csv missing required column: %s", col)
		}
	}

	var loadedQuests []QuestDefinition
	for i := 1; i < len(records); i++ {
		row := records[i]
		id := row[headerMap["id"]]
		title := row[headerMap["title"]]
		desc := row[headerMap["desc"]]
		reqDesc := row[headerMap["req_desc"]]
		message := row[headerMap["message"]]
		rewardsStr := row[headerMap["rewards"]]

		var rewards []QuestReward
		if rewardsStr != "" {
			err = json.Unmarshal([]byte(rewardsStr), &rewards)
			if err != nil {
				return fmt.Errorf("failed to parse rewards JSON on line %d: %w", i+1, err)
			}
		}

		qID := id
		loadedQuests = append(loadedQuests, QuestDefinition{
			ID:      qID,
			Title:   title,
			Desc:    desc,
			ReqDesc: reqDesc,
			Message: message,
			Rewards: rewards,
			Check: func(p *models.Player) bool {
				return CheckQuest(qID, p)
			},
		})
	}

	questRegistry = loadedQuests
	logger.S.Infof("Successfully loaded %d quests from Quest.csv", len(questRegistry))
	return nil
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
			InventorySvc.AddItem(tx, player.ID.String(), r.ID, "materials", int(r.Amount))
		case "gu":
			guSpec, _ := models.GetGuAny(r.ID)
			newGu := models.GuInstance{
				InstanceID: "gu-" + uuid.New().String(),
				GuID:       r.ID,
				Rank:       guSpec.Rank,
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
	baseFields := []string{"human_qi"}
	baseVals := []interface{}{player.HumanQi}
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
	var cachedResp gin.H
	cacheKey := "game_quests_list"

	if err := cache.GetJSON(cacheKey, &cachedResp); err == nil && cachedResp != nil {
		c.JSON(http.StatusOK, cachedResp)
		return
	}

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

	cachedResp = gin.H{"quests": resp}
	_ = cache.SetJSON(cacheKey, cachedResp, 24*time.Hour)
	c.JSON(http.StatusOK, cachedResp)
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

	existing, _ := QuestSvc.Get(tx, player.ID.String(), req.QuestID)
	if existing != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nhiệm vụ này đã được hoàn thành trước đó"})
		return
	}

	if !quest.Check(&player) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa đạt yêu cầu: " + quest.ReqDesc})
		return
	}

	if err := applyQuestRewardsTx(tx, &player, quest.Rewards); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply rewards"})
		return
	}

	QuestSvc.Complete(tx, player.ID.String(), req.QuestID)

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

	auditlog.Logf(db.DB, &authID, "quest.complete", "questId=%s questTitle=%s", req.QuestID, quest.Title)
	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": quest.Message + rewardStr,
	})
}
