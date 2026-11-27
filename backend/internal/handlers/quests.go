package handlers

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type QuestReward struct {
	Type   string  `json:"type"`
	ID     string  `json:"id,omitempty"`
	Amount float64 `json:"amount"`
	Label  string  `json:"label"`
}

type QuestCondition struct {
	Mode   string `json:"mode"`             // "state", "event", "hybrid"
	Metric string `json:"metric"`           // e.g. "rank", "has_vital_gu", "feed_gu"
	Target string `json:"target,omitempty"` // optional target filter
	Count  int64  `json:"count"`
}

type QuestDefinition struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Desc           string         `json:"desc"`
	ReqDesc        string         `json:"req_desc"`
	Message        string         `json:"message"`
	Rewards        []QuestReward  `json:"rewards"`
	Chapter        int            `json:"chapter"`
	StepOrder      int            `json:"step_order"`
	PrevQuestID    string         `json:"prev_quest_id"`
	GuideTip       string         `json:"guide_tip"`
	TargetRoute    string         `json:"target_route"`
	UnlockFeatures []string       `json:"unlock_features"`
	Condition      QuestCondition `json:"condition"`
	Check          func(*models.Player) bool
}

type questResponse struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Desc           string         `json:"desc"`
	ReqDesc        string         `json:"req_desc"`
	Rewards        []QuestReward  `json:"rewards"`
	Chapter        int            `json:"chapter"`
	StepOrder      int            `json:"step_order"`
	PrevQuestID    string         `json:"prev_quest_id,omitempty"`
	GuideTip       string         `json:"guide_tip,omitempty"`
	TargetRoute    string         `json:"target_route,omitempty"`
	UnlockFeatures []string       `json:"unlock_features,omitempty"`
	Condition      QuestCondition `json:"condition"`
	Status         string         `json:"status"` // COMPLETED, ACTIVE, LOCKED
	CurrentCount   int64          `json:"current_count"`
	TargetCount    int64          `json:"target_count"`
	CanClaim       bool           `json:"can_claim"`
	IsCurrent      bool           `json:"is_current"`
}

type storyProgressPayload struct {
	Count         int64           `json:"count"`
	SeenEvents    map[string]bool `json:"seen_events,omitempty"`
	AutoCompleted bool            `json:"auto_completed,omitempty"`
}

var legacyAbsorbedQuests = map[string]bool{
	"quest_1_awaken":             true,
	"quest_2_vital_gu":           true,
	"quest_3_rank_2":             true,
	"quest_4_refine_gu":          true,
	"quest_5_fame":               true,
	"quest_6_rank_3":             true,
	"quest_8_rank_4":             true,
	"quest_11_gu_collector":      true,
	"quest_12_rank_5":            true,
	"quest_15_immortal_gu_owner": true,
	"quest_17_rank_6":            true,
	"quest_18_human_qi":          true,
	"quest_19_soul_growth":       true,
	"quest_20_exploration":       true,
}

var questRegistry []QuestDefinition

func csvPath(filename string) string {
	candidates := []string{
		filepath.Join("data", filename),
		filepath.Join("backend", "data", filename),
		filepath.Join("cmd", "server", "data", filename),
		filepath.Join("backend", "cmd", "server", "data", filename),
		filepath.Join("..", "data", filename),
		filepath.Join("..", "..", "data", filename),
		filepath.Join("..", "cmd", "server", "data", filename),
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

func parseQuestCondition(raw string, legacyID string) QuestCondition {
	var cond QuestCondition
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &cond)
	}
	if cond.Metric != "" {
		if cond.Count <= 0 {
			cond.Count = 1
		}
		if cond.Mode == "" {
			cond.Mode = "state"
		}
		return cond
	}
	// Fallback mapping for legacy IDs if condition column is empty
	switch legacyID {
	case "quest_1_awaken":
		return QuestCondition{Mode: "state", Metric: "rank", Count: 1}
	case "quest_2_vital_gu":
		return QuestCondition{Mode: "state", Metric: "has_vital_gu", Count: 1}
	case "quest_3_rank_2":
		return QuestCondition{Mode: "state", Metric: "rank", Count: 2}
	case "quest_4_refine_gu":
		return QuestCondition{Mode: "hybrid", Metric: "refine_or_rank2_gu", Count: 1}
	case "quest_5_fame":
		return QuestCondition{Mode: "state", Metric: "cleared_cells", Count: 5}
	case "quest_6_rank_3":
		return QuestCondition{Mode: "state", Metric: "rank", Count: 3}
	case "quest_8_rank_4":
		return QuestCondition{Mode: "state", Metric: "rank", Count: 4}
	case "quest_11_gu_collector":
		return QuestCondition{Mode: "state", Metric: "gu_count", Count: 10}
	case "quest_12_rank_5":
		return QuestCondition{Mode: "state", Metric: "rank", Count: 5}
	case "quest_15_immortal_gu_owner":
		return QuestCondition{Mode: "state", Metric: "rank6_gu", Count: 1}
	case "quest_17_rank_6":
		return QuestCondition{Mode: "state", Metric: "rank", Count: 6}
	case "quest_18_human_qi":
		return QuestCondition{Mode: "state", Metric: "human_qi", Count: 500}
	case "quest_19_soul_growth":
		return QuestCondition{Mode: "state", Metric: "soul_hp", Count: 100}
	case "quest_20_exploration":
		return QuestCondition{Mode: "state", Metric: "visited_subzones", Count: 20}
	default:
		return QuestCondition{Mode: "state", Metric: "rank", Count: 1}
	}
}

func evaluateStateMetric(metric string, targetCount int64, p *models.Player) (bool, int64) {
	if p == nil {
		return false, 0
	}
	if targetCount <= 0 {
		targetCount = 1
	}
	switch metric {
	case "rank":
		cur := int64(p.Rank)
		return cur >= targetCount, cur
	case "has_vital_gu":
		for _, g := range p.GuInventory {
			if g.IsVital {
				return true, 1
			}
		}
		return false, 0
	case "refine_or_rank2_gu":
		for _, g := range p.GuInventory {
			if spec, ok := models.GetGu(g.GuID, g.Rank); ok && spec.Rank >= 2 {
				return true, 1
			} else if g.Rank >= 2 {
				return true, 1
			}
		}
		return false, 0
	case "cleared_cells":
		cur := int64(len(p.ClearedCells))
		return cur >= targetCount, cur
	case "gu_count":
		cur := int64(len(p.GuInventory))
		return cur >= targetCount, cur
	case "visited_subzones":
		cur := int64(len(p.VisitedSubZones))
		return cur >= targetCount, cur
	case "human_qi":
		cur := int64(p.HumanQi)
		return cur >= targetCount, cur
	case "soul_hp":
		cur := int64(p.SoulHP)
		return cur >= targetCount, cur
	case "rank6_gu":
		for _, g := range p.GuInventory {
			if spec, ok := models.GetGu(g.GuID, g.Rank); ok && spec.Rank >= 6 {
				return true, 1
			} else if g.Rank >= 6 {
				return true, 1
			}
		}
		return false, 0
	case "total_dao_marks":
		var sum int64
		for _, v := range p.DaoMarks {
			sum += v
		}
		return sum >= targetCount, sum
	case "pet_or_tame":
		if db.DB != nil {
			var cnt int64
			if err := db.DB.QueryRow(`SELECT COUNT(*) FROM non_player_actors WHERE owner_player_id = $1 AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'`, p.ID).Scan(&cnt); err == nil && cnt >= targetCount {
				return true, cnt
			}
		}
		return false, 0
	case "sect_member":
		if db.DB != nil {
			var cnt int64
			if err := db.DB.QueryRow(`SELECT COUNT(*) FROM sect_members WHERE player_id = $1`, p.ID).Scan(&cnt); err == nil && cnt >= targetCount {
				return true, cnt
			}
		}
		return false, 0
	case "myriad_tribulation_count":
		if db.DB != nil {
			var cnt int64
			if err := db.DB.QueryRow(`SELECT COALESCE(myriad_tribulation_count, 0) FROM immortal_apertures WHERE player_id = $1`, p.ID).Scan(&cnt); err == nil {
				return cnt >= targetCount, cnt
			}
		}
		return false, 0
	default:
		return false, 0
	}
}

func parseStoryProgress(raw json.RawMessage) storyProgressPayload {
	var p storyProgressPayload
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	if p.SeenEvents == nil {
		p.SeenEvents = make(map[string]bool)
	}
	return p
}

func EvaluateQuestProgress(q QuestDefinition, p *models.Player, rawProgress json.RawMessage) (bool, int64, int64) {
	target := q.Condition.Count
	if target <= 0 {
		target = 1
	}
	prog := parseStoryProgress(rawProgress)
	switch q.Condition.Mode {
	case "event":
		return prog.Count >= target, prog.Count, target
	case "hybrid":
		stateOk, stateCur := evaluateStateMetric(q.Condition.Metric, target, p)
		if stateOk {
			return true, target, target
		}
		if prog.Count >= target {
			return true, prog.Count, target
		}
		if stateCur > prog.Count {
			return false, stateCur, target
		}
		return false, prog.Count, target
	default: // "state"
		ok, cur := evaluateStateMetric(q.Condition.Metric, target, p)
		return ok, cur, target
	}
}

func CheckQuest(questID string, p *models.Player) bool {
	if q := FindQuest(questID); q != nil {
		ok, _, _ := EvaluateQuestProgress(*q, p, nil)
		return ok
	}
	return false
}

// TrackStoryQuestProgress records server-confirmed gameplay actions for story quests with sourceEventID deduplication.
func TrackStoryQuestProgress(tx *sql.Tx, playerID uuid.UUID, sourceEventID, eventType, targetID string, amount int64) error {
	if tx == nil || amount <= 0 || eventType == "" {
		return nil
	}
	if len(questRegistry) == 0 {
		return nil
	}

	// Load existing progress rows for this player
	existing := make(map[string]services.QuestProgress)
	rows, err := tx.Query(`SELECT player_id, quest_id, status, progress_data, completed_at, created_at, updated_at FROM player_quest_progress WHERE player_id = $1`, playerID.String())
	if err != nil {
		return err
	}
	for rows.Next() {
		var qp services.QuestProgress
		if scanErr := rows.Scan(&qp.PlayerID, &qp.QuestID, &qp.Status, &qp.ProgressData, &qp.CompletedAt, &qp.CreatedAt, &qp.UpdatedAt); scanErr == nil {
			existing[qp.QuestID] = qp
		}
	}
	_ = rows.Close()

	// Find the first uncompleted quest or matching quests whose prerequisite is completed
	for _, q := range questRegistry {
		if qp, ok := existing[q.ID]; ok && qp.Status == "COMPLETED" {
			continue
		}
		if q.Condition.Mode != "event" && q.Condition.Mode != "hybrid" {
			continue
		}
		if q.Condition.Metric != eventType {
			continue
		}
		if q.Condition.Target != "" && q.Condition.Target != targetID {
			continue
		}
		// Ensure prerequisite is completed (or this is the current active step)
		if q.PrevQuestID != "" {
			prev, prevOk := existing[q.PrevQuestID]
			if !prevOk || prev.Status != "COMPLETED" {
				continue
			}
		}

		var prog storyProgressPayload
		if qp, ok := existing[q.ID]; ok {
			prog = parseStoryProgress(qp.ProgressData)
		} else {
			prog = storyProgressPayload{SeenEvents: make(map[string]bool)}
		}
		if sourceEventID != "" && prog.SeenEvents[sourceEventID] {
			continue
		}
		if sourceEventID != "" {
			prog.SeenEvents[sourceEventID] = true
		}
		prog.Count += amount
		raw, _ := json.Marshal(prog)
		if err := QuestSvc.Upsert(tx, &services.QuestProgress{
			PlayerID:     playerID.String(),
			QuestID:      q.ID,
			Status:       "ACTIVE",
			ProgressData: raw,
		}); err != nil {
			return err
		}
	}
	return nil
}

func minRankForChapter(chapter int) int {
	switch chapter {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	case 4:
		return 5
	case 5:
		return 6
	case 6:
		return 7
	case 7:
		return 9
	default:
		return 1
	}
}

func maxRankForChapter(chapter int) int {
	switch chapter {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 4
	case 4:
		return 5
	case 5:
		return 6
	case 6:
		return 8
	case 7:
		return 9
	default:
		return 9
	}
}

// SyncPlayerStoryQuests retroactively auto-completes newbie/practice steps in chapters below player.Rank.
func SyncPlayerStoryQuests(p *models.Player, existing map[string]services.QuestProgress) {
	if p == nil || p.Rank <= 1 || len(questRegistry) == 0 {
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	changed := false
	for _, q := range questRegistry {
		if qp, ok := existing[q.ID]; ok && qp.Status == "COMPLETED" {
			continue
		}
		// Only auto-complete steps in chapters strictly below player's current rank,
		// and preserve unclaimed rewards for the 14 legacy absorbed quests.
		if p.Rank <= maxRankForChapter(q.Chapter) {
			continue
		}
		if legacyAbsorbedQuests[q.ID] {
			continue
		}
		raw, _ := json.Marshal(storyProgressPayload{Count: q.Condition.Count, AutoCompleted: true})
		now := time.Now()
		qp := services.QuestProgress{
			PlayerID:     p.ID.String(),
			QuestID:      q.ID,
			Status:       "COMPLETED",
			ProgressData: raw,
			CompletedAt:  &now,
		}
		if err := QuestSvc.Upsert(tx, &qp); err != nil {
			return
		}
		existing[q.ID] = qp
		changed = true
	}
	if changed {
		_ = tx.Commit()
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
	rows, err := db.DB.Query(`
		SELECT id, title, description, req_desc, message, rewards::text,
		       chapter, step_order, prev_quest_id, guide_tip, target_route,
		       unlock_features::text, condition::text
		FROM quests
		ORDER BY chapter, step_order, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var loaded []QuestDefinition
	for rows.Next() {
		var id, title, desc, reqDesc, message, rewardsStr string
		var chapter, stepOrder int
		var prevQuestID, guideTip, targetRoute, unlockFeaturesStr, conditionStr string
		if err := rows.Scan(
			&id, &title, &desc, &reqDesc, &message, &rewardsStr,
			&chapter, &stepOrder, &prevQuestID, &guideTip, &targetRoute,
			&unlockFeaturesStr, &conditionStr,
		); err != nil {
			return nil, err
		}
		var rewards []QuestReward
		if rewardsStr != "" && rewardsStr != "[]" {
			if err := json.Unmarshal([]byte(rewardsStr), &rewards); err != nil {
				logger.S.Warnf("failed to parse rewards JSON for quest %s: %v", id, err)
			}
		}
		var unlockFeatures []string
		if unlockFeaturesStr != "" && unlockFeaturesStr != "[]" {
			_ = json.Unmarshal([]byte(unlockFeaturesStr), &unlockFeatures)
		}
		cond := parseQuestCondition(conditionStr, id)
		qID := id
		loaded = append(loaded, QuestDefinition{
			ID:             qID,
			Title:          title,
			Desc:           desc,
			ReqDesc:        reqDesc,
			Message:        message,
			Rewards:        rewards,
			Chapter:        chapter,
			StepOrder:      stepOrder,
			PrevQuestID:    prevQuestID,
			GuideTip:       guideTip,
			TargetRoute:    targetRoute,
			UnlockFeatures: unlockFeatures,
			Condition:      cond,
			Check: func(p *models.Player) bool {
				return CheckQuest(qID, p)
			},
		})
	}
	sort.SliceStable(loaded, func(i, j int) bool {
		if loaded[i].Chapter != loaded[j].Chapter {
			return loaded[i].Chapter < loaded[j].Chapter
		}
		if loaded[i].StepOrder != loaded[j].StepOrder {
			return loaded[i].StepOrder < loaded[j].StepOrder
		}
		return loaded[i].ID < loaded[j].ID
	})
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

	getOptional := func(row []string, col string) string {
		if idx, ok := headerMap[col]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	var loadedQuests []QuestDefinition
	for i := 1; i < len(records); i++ {
		row := records[i]
		id := strings.TrimSpace(row[headerMap["id"]])
		title := strings.TrimSpace(row[headerMap["title"]])
		desc := strings.TrimSpace(row[headerMap["desc"]])
		reqDesc := strings.TrimSpace(row[headerMap["req_desc"]])
		message := strings.TrimSpace(row[headerMap["message"]])
		rewardsStr := strings.TrimSpace(row[headerMap["rewards"]])

		var rewards []QuestReward
		if rewardsStr != "" {
			err = json.Unmarshal([]byte(rewardsStr), &rewards)
			if err != nil {
				return fmt.Errorf("failed to parse rewards JSON on line %d: %w", i+1, err)
			}
		}

		chapter := 1
		if chStr := getOptional(row, "chapter"); chStr != "" {
			if v, convErr := strconv.Atoi(chStr); convErr == nil && v > 0 {
				chapter = v
			}
		}
		stepOrder := i
		if stStr := getOptional(row, "step_order"); stStr != "" {
			if v, convErr := strconv.Atoi(stStr); convErr == nil {
				stepOrder = v
			}
		}
		prevQuestID := getOptional(row, "prev_quest_id")
		guideTip := getOptional(row, "guide_tip")
		targetRoute := getOptional(row, "target_route")
		var unlockFeatures []string
		if ufStr := getOptional(row, "unlock_features"); ufStr != "" && ufStr != "[]" {
			_ = json.Unmarshal([]byte(ufStr), &unlockFeatures)
		}
		cond := parseQuestCondition(getOptional(row, "condition"), id)

		qID := id
		loadedQuests = append(loadedQuests, QuestDefinition{
			ID:             qID,
			Title:          title,
			Desc:           desc,
			ReqDesc:        reqDesc,
			Message:        message,
			Rewards:        rewards,
			Chapter:        chapter,
			StepOrder:      stepOrder,
			PrevQuestID:    prevQuestID,
			GuideTip:       guideTip,
			TargetRoute:    targetRoute,
			UnlockFeatures: unlockFeatures,
			Condition:      cond,
			Check: func(p *models.Player) bool {
				return CheckQuest(qID, p)
			},
		})
	}

	sort.SliceStable(loadedQuests, func(i, j int) bool {
		if loadedQuests[i].Chapter != loadedQuests[j].Chapter {
			return loadedQuests[i].Chapter < loadedQuests[j].Chapter
		}
		if loadedQuests[i].StepOrder != loadedQuests[j].StepOrder {
			return loadedQuests[i].StepOrder < loadedQuests[j].StepOrder
		}
		return loadedQuests[i].ID < loadedQuests[j].ID
	})

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
	itemRewards := make(map[string]int64)
	for _, r := range rewards {
		switch r.Type {
		case "human_qi":
			player.HumanQi += r.Amount
			if player.HumanQi > MaxHumanQi {
				player.HumanQi = MaxHumanQi
			}
		case "primeval_stones":
			player.PrimevalStones += float64(r.Amount)
		case "immortal_stones":
			player.ImmortalStones += float64(r.Amount)
		case "dao_mark":
			if player.DaoMarks == nil {
				player.DaoMarks = make(models.DaoMarksMap)
			}
			player.DaoMarks[r.ID] += int64(r.Amount)
		case "item":
			itemRewards[r.ID] += int64(r.Amount)
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
	if len(itemRewards) > 0 {
		if err := InventorySvc.BulkAddItems(tx, player.ID.String(), "materials", itemRewards); err != nil {
			return err
		}
	}
	return nil
}

func buildQuestUpdateQuery(player *models.Player) (string, []interface{}) {
	return "UPDATE players SET human_qi = $1, primeval_stones = $2, immortal_stones = $3 WHERE id = $4",
		[]interface{}{player.HumanQi, player.PrimevalStones, player.ImmortalStones, player.ID}
}

func ListQuests(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to load player", err))
		return
	}
	_ = tx.Commit()

	progressList, err := QuestSvc.ListByPlayer(authID.String())
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to load quest progress", err))
		return
	}
	progressMap := make(map[string]services.QuestProgress, len(progressList))
	for _, qp := range progressList {
		progressMap[qp.QuestID] = qp
	}

	SyncPlayerStoryQuests(&player, progressMap)

	currentQuestID := ""
	unlockedFeaturesSet := make(map[string]bool)
	resp := make([]questResponse, 0, len(questRegistry))

	for _, q := range questRegistry {
		qp, hasRow := progressMap[q.ID]
		isDone := hasRow && qp.Status == "COMPLETED"
		if isDone {
			for _, f := range q.UnlockFeatures {
				unlockedFeaturesSet[f] = true
			}
		}

		prevDone := q.PrevQuestID == ""
		if !prevDone {
			if prevQP, pOk := progressMap[q.PrevQuestID]; pOk && prevQP.Status == "COMPLETED" {
				prevDone = true
			} else if player.Rank > maxRankForChapter(q.Chapter) {
				prevDone = true
			}
		}

		status := "LOCKED"
		if isDone {
			status = "COMPLETED"
		} else if prevDone {
			status = "ACTIVE"
		}

		canClaim := false
		var curCount int64
		targetCount := q.Condition.Count
		if targetCount <= 0 {
			targetCount = 1
		}
		if !isDone && prevDone {
			canClaim, curCount, targetCount = EvaluateQuestProgress(q, &player, qp.ProgressData)
		} else if isDone {
			curCount = targetCount
		}

		isCurrent := false
		if !isDone && currentQuestID == "" {
			currentQuestID = q.ID
			isCurrent = true
			if status == "LOCKED" {
				status = "ACTIVE"
				canClaim, curCount, targetCount = EvaluateQuestProgress(q, &player, qp.ProgressData)
			}
		}

		resp = append(resp, questResponse{
			ID:             q.ID,
			Title:          q.Title,
			Desc:           q.Desc,
			ReqDesc:        q.ReqDesc,
			Rewards:        q.Rewards,
			Chapter:        q.Chapter,
			StepOrder:      q.StepOrder,
			PrevQuestID:    q.PrevQuestID,
			GuideTip:       q.GuideTip,
			TargetRoute:    q.TargetRoute,
			UnlockFeatures: q.UnlockFeatures,
			Condition:      q.Condition,
			Status:         status,
			CurrentCount:   curCount,
			TargetCount:    targetCount,
			CanClaim:       canClaim,
			IsCurrent:      isCurrent,
		})
	}

	unlockedFeatures := make([]string, 0, len(unlockedFeaturesSet))
	for f := range unlockedFeaturesSet {
		unlockedFeatures = append(unlockedFeatures, f)
	}
	sort.Strings(unlockedFeatures)

	c.JSON(http.StatusOK, gin.H{
		"quests":            resp,
		"current_quest_id":  currentQuestID,
		"unlocked_features": unlockedFeatures,
	})
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
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to fetch player stats", err))
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if quest.PrevQuestID != "" && player.Rank <= maxRankForChapter(quest.Chapter) {
		prevQP, prevErr := QuestSvc.Get(tx, player.ID.String(), quest.PrevQuestID)
		if prevErr != nil || prevQP.Status != "COMPLETED" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cần hoàn thành bước nhiệm vụ trước đó trước!"})
			return
		}
	}

	var rawProgress json.RawMessage
	if existingQP, qErr := QuestSvc.Get(tx, player.ID.String(), req.QuestID); qErr == nil && existingQP != nil {
		rawProgress = existingQP.ProgressData
	}

	canClaim, _, _ := EvaluateQuestProgress(*quest, &player, rawProgress)
	if !canClaim {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa đạt yêu cầu: " + quest.ReqDesc})
		return
	}

	if err := applyQuestRewardsTx(tx, &player, quest.Rewards); err != nil {
		httperr.Respond(c, httperr.Internal("Failed to apply rewards", err))
		return
	}

	completed, err := QuestSvc.Complete(tx, player.ID.String(), req.QuestID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error completing quest", err))
		return
	}
	if !completed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nhiệm vụ này đã được hoàn thành trước đó"})
		return
	}

	qiGained := 0.0

	updateQuery, updateArgs := buildQuestUpdateQuery(&player)
	_, err = tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to update player", err))
		return
	}

	if player.Rank >= 6 {
		_, err = tx.Exec(`
			UPDATE immortal_apertures 
			SET dao_marks = $1 
			WHERE player_id = $2`,
			player.DaoMarks, player.ID,
		)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update aperture Dao Marks", err))
			return
		}
	}
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		if db.IsApertureCapacity(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		httperr.Respond(c, httperr.Internal("Failed to update player Gus", err))
		return
	}

	// Removed automatic referral milestone check

	auditlog.LogContext(c.Request.Context(), tx, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "quest.complete",
		EntityType:     "quest",
		EntityID:       quest.ID,
		RequestSummary: fmt.Sprintf("Hoàn thành nhiệm vụ %s (%s).%s", quest.ID, quest.Title, buildRewardLabel(quest.Rewards)),
		Metadata: map[string]interface{}{
			"quest_id":    quest.ID,
			"quest_title": quest.Title,
			"rewards":     quest.Rewards,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Transaction commit failure", err))
		return
	}

	rewardStr := buildRewardLabel(quest.Rewards)

	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": quest.Message + rewardStr + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained)})
}
