package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

const MaxDailyInstanceEntries = 5

var (
	instanceTemplatesCache []InstanceListItem
	instanceTemplatesMap   map[string]*InstanceTemplate
	instanceCacheMutex     sync.RWMutex
)

// InitInstanceCache loads all instance_templates into memory cache.
func InitInstanceCache() error {
	if db.DB == nil {
		return nil
	}
	rows, err := db.DB.Query(`
		SELECT id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours
		FROM instance_templates
		ORDER BY required_rank ASC, id ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to fetch instance templates for cache: %w", err)
	}
	defer rows.Close()

	var list []InstanceListItem
	tmplMap := make(map[string]*InstanceTemplate)

	for rows.Next() {
		var t InstanceTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.RequiredRank, &t.EntryItem, &t.Description, &t.Rewards, &t.MapData, &t.CooldownHours); err != nil {
			return fmt.Errorf("failed to scan instance template: %w", err)
		}

		if t.Type != "SPATIAL_REALM" {
			list = append(list, InstanceListItem{
				ID:            t.ID,
				Name:          t.Name,
				Type:          t.Type,
				RequiredRank:  t.RequiredRank,
				EntryItem:     t.EntryItem,
				Description:   t.Description,
				Rewards:       t.Rewards,
				CooldownHours: t.CooldownHours,
			})
		}
		tCopy := t
		tmplMap[t.ID] = &tCopy
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate instance templates: %w", err)
	}

	instanceCacheMutex.Lock()
	instanceTemplatesCache = list
	instanceTemplatesMap = tmplMap
	instanceCacheMutex.Unlock()

	return nil
}

func getCachedInstanceList() []InstanceListItem {
	instanceCacheMutex.RLock()
	defer instanceCacheMutex.RUnlock()
	if len(instanceTemplatesCache) == 0 {
		return nil
	}
	res := make([]InstanceListItem, len(instanceTemplatesCache))
	copy(res, instanceTemplatesCache)
	return res
}

func getCachedInstanceTemplate(id string) (InstanceTemplate, bool) {
	instanceCacheMutex.RLock()
	t, found := instanceTemplatesMap[id]
	instanceCacheMutex.RUnlock()
	if found && t != nil {
		return *t, true
	}
	return InstanceTemplate{}, false
}

type InstanceTemplate struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Type          string          `json:"type"` // "INHERITANCE_RIGHTEOUS", "INHERITANCE_DEMONIC", "DREAM_REALM", "SECRET_REALM"
	RequiredRank  int             `json:"required_rank"`
	EntryItem     string          `json:"entry_item"`
	Description   string          `json:"description"`
	Rewards       json.RawMessage `json:"rewards"`
	MapData       json.RawMessage `json:"map_data"`
	CooldownHours int             `json:"cooldown_hours"`
}

type PlayerInstanceSession struct {
	ID            uuid.UUID       `json:"id"`
	PlayerID      uuid.UUID       `json:"player_id"`
	InstanceID    string          `json:"instance_id"`
	CurrentStage  int             `json:"current_stage"`
	Status        string          `json:"status"` // "ACTIVE", "COMPLETED", "FAILED"
	TempInventory json.RawMessage `json:"temp_inventory"`
}

type EnterInstanceRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
}

type ActionInstanceRequest struct {
	OptionIndex int `json:"option_index"`
}

type InstanceRewardAttainment struct {
	Path   string `json:"path"`
	Amount int    `json:"amount"`
}

type InstanceReward struct {
	Materials      map[string]int64          `json:"materials"`
	PrimevalStones float64                   `json:"primeval_stones"`
	Attainment     *InstanceRewardAttainment `json:"attainment"`
	DaoMarks       map[string]int64          `json:"dao_marks"`
}

type InstanceOption struct {
	Text         string                 `json:"text"`
	NextStage    int                    `json:"next_stage"`
	Requirements map[string]interface{} `json:"requirements"`
}

type InstanceStage struct {
	StageIndex  int              `json:"stage_index"`
	Description string           `json:"description"`
	Options     []InstanceOption `json:"options"`
}

// Helper to parse HP and Soul HP effects from option text
func parseHPEffects(text string) (int, int) {
	// returns (soulHPDeduct, normalHPDeduct)
	soulReg := regexp.MustCompile(`(\d+)\s+Soul\s+HP`)
	normalReg := regexp.MustCompile(`(\d+)\s+HP`)

	var soulHP, normalHP int

	soulMatches := soulReg.FindStringSubmatch(text)
	if len(soulMatches) > 1 {
		val, _ := strconv.Atoi(soulMatches[1])
		soulHP = val
	}

	cleanText := soulReg.ReplaceAllString(text, "")
	normalMatches := normalReg.FindStringSubmatch(cleanText)
	if len(normalMatches) > 1 {
		val, _ := strconv.Atoi(normalMatches[1])
		normalHP = val
	}

	return soulHP, normalHP
}

// GetActiveInstance retrieves player's active instance session
func GetActiveInstance(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var session PlayerInstanceSession
	query := `
		SELECT id, player_id, instance_id, current_stage, status, temp_inventory
		FROM player_instance_sessions
		WHERE player_id = $1 AND status = 'ACTIVE'
		LIMIT 1
	`
	err := db.DB.QueryRowContext(c.Request.Context(), query, playerID).Scan(
		&session.ID, &session.PlayerID, &session.InstanceID,
		&session.CurrentStage, &session.Status, &session.TempInventory,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{"active": false})
		} else {
			httperr.Respond(c, httperr.Internal("Database error: "+err.Error(), err))
		}
		return
	}

	var template InstanceTemplate
	var found bool
	template, found = getCachedInstanceTemplate(session.InstanceID)
	if !found {
		tplQuery := `
			SELECT id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours
			FROM instance_templates
			WHERE id = $1 AND type <> 'SPATIAL_REALM'
		`
		err = db.DB.QueryRowContext(c.Request.Context(), tplQuery, session.InstanceID).Scan(
			&template.ID, &template.Name, &template.Type, &template.RequiredRank,
			&template.EntryItem, &template.Description, &template.Rewards, &template.MapData,
			&template.CooldownHours,
		)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to fetch instance template: "+err.Error(), err))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"active":   true,
		"session":  session,
		"template": template,
	})
}

// InstanceListItem is a template summary for the list endpoint, enriched with
// per-player availability info (cooldown remaining).
type InstanceListItem struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Type              string          `json:"type"`
	RequiredRank      int             `json:"required_rank"`
	EntryItem         string          `json:"entry_item"`
	Description       string          `json:"description"`
	Rewards           json.RawMessage `json:"rewards"`
	CooldownHours     int             `json:"cooldown_hours"`
	CooldownRemaining int             `json:"cooldown_remaining"` // seconds until re-entry allowed; 0 = ready
}

// ListInstances returns all instance templates the player can view, with
// per-instance cooldown remaining for this player.
func ListInstances(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	items := getCachedInstanceList()
	if len(items) == 0 {
		if err := InitInstanceCache(); err == nil {
			items = getCachedInstanceList()
		}
	}

	if len(items) == 0 {
		rows, err := db.DB.QueryContext(c.Request.Context(), `
			SELECT id, name, type, required_rank, entry_item, description, rewards, cooldown_hours
			FROM instance_templates
			WHERE type <> 'SPATIAL_REALM'
			ORDER BY required_rank ASC, id ASC
		`)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to fetch instance templates: "+err.Error(), err))
			return
		}
		defer rows.Close()

		for rows.Next() {
			var it InstanceListItem
			if err := rows.Scan(&it.ID, &it.Name, &it.Type, &it.RequiredRank, &it.EntryItem, &it.Description, &it.Rewards, &it.CooldownHours); err != nil {
				httperr.Respond(c, httperr.Internal("Failed to scan instance template: "+err.Error(), err))
				return
			}
			items = append(items, it)
		}
		if err := rows.Err(); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to iterate instance templates: "+err.Error(), err))
			return
		}
	}

	// Enrich with cooldown remaining per instance for this player (computed in
	// SQL to avoid timezone drift on TIMESTAMP WITHOUT TIME ZONE columns).
	cdMap := map[string]int64{}
	cdRows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT instance_id,
		       GREATEST(0, EXTRACT(EPOCH FROM (it.cooldown_hours::interval - (NOW() - s.created_at))))::int
		FROM player_instance_sessions s
		JOIN instance_templates it ON it.id = s.instance_id
		WHERE s.player_id = $1
		ORDER BY s.instance_id, s.created_at DESC
	`, playerID)
	if err == nil {
		defer cdRows.Close()
		for cdRows.Next() {
			var instID string
			var remain int64
			if err := cdRows.Scan(&instID, &remain); err != nil {
				continue
			}
			cdMap[instID] = remain
		}
	}

	for i := range items {
		items[i].CooldownRemaining = int(cdMap[items[i].ID])
	}

	var dailyEntriesUsed int
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*)
		FROM player_instance_sessions s
		JOIN instance_templates it ON it.id = s.instance_id
		WHERE s.player_id = $1
		  AND s.created_at >= CURRENT_DATE
		  AND it.type IN ('DREAM_REALM', 'SECRET_REALM')
	`, playerID).Scan(&dailyEntriesUsed)

	c.JSON(http.StatusOK, gin.H{
		"instances":          items,
		"daily_entries_used": dailyEntriesUsed,
		"daily_entries_max":  MaxDailyInstanceEntries,
	})
}

// EnterInstance initializes a new instance session
func EnterInstance(c *gin.Context) {
	var req EnterInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id is required"})
		return
	}

	enterInstanceWithRequest(c, req, "")
}

// enterInstanceWithRequest runs the entry flow for an already-parsed request.
// Delegating handlers (e.g. Dream Realms) must use this instead of calling
// EnterInstance again, because the JSON body can only be decoded once.
// requiredType, when non-empty, restricts the template type.
func enterInstanceWithRequest(c *gin.Context, req EnterInstanceRequest, requiredType string) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if template, found := getCachedInstanceTemplate(req.InstanceID); found && template.Type == "SPATIAL_REALM" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bí cảnh bản đồ dùng API spatial riêng"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Transaction start failed", err))
		return
	}
	defer tx.Rollback()

	// 1. Check if already in an active session
	var dummy uuid.UUID
	err = tx.QueryRow(`
		SELECT id FROM player_instance_sessions
		WHERE player_id = $1 AND status = 'ACTIVE'
		LIMIT 1
	`, playerID).Scan(&dummy)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang thám hiểm một mộng cảnh hoặc bí cảnh khác!"})
		return
	}

	// 2. Fetch the instance template
	template, found := getCachedInstanceTemplate(req.InstanceID)
	if !found {
		err = tx.QueryRow(`
			SELECT id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours
			FROM instance_templates
			WHERE id = $1
		`, req.InstanceID).Scan(
			&template.ID, &template.Name, &template.Type, &template.RequiredRank,
			&template.EntryItem, &template.Description, &template.Rewards, &template.MapData,
			&template.CooldownHours,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mộng cảnh yêu cầu!"})
			} else {
				httperr.Respond(c, httperr.Internal("Database error: "+err.Error(), err))
			}
			return
		}
	}

	// Enforce the requested instance type, when the caller restricts it.
	if requiredType != "" && template.Type != requiredType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không phải mộng cảnh hợp lệ!"})
		return
	}

	// 2a. Enforce daily limit per player for DREAM_REALM and SECRET_REALM
	if template.Type == "DREAM_REALM" || template.Type == "SECRET_REALM" {
		var dailyEntries int
		err = tx.QueryRow(`
			SELECT COUNT(*)
			FROM player_instance_sessions s
			JOIN instance_templates it ON it.id = s.instance_id
			WHERE s.player_id = $1
			  AND s.created_at >= CURRENT_DATE
			  AND it.type IN ('DREAM_REALM', 'SECRET_REALM')
		`, playerID).Scan(&dailyEntries)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Database error checking daily limit: "+err.Error(), err))
			return
		}
		if dailyEntries >= MaxDailyInstanceEntries {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo hữu đã hết lượt thám hiểm mộng cảnh hôm nay (tối đa %d lượt/ngày)!", MaxDailyInstanceEntries)})
			return
		}
	}

	// 2b. Enforce cooldown from the most recent session for this instance.
	if template.CooldownHours > 0 {
		var remainingSeconds int
		err = tx.QueryRow(`
			SELECT GREATEST(0, EXTRACT(EPOCH FROM (($1::interval) - (NOW() - created_at))))::int
			FROM player_instance_sessions
			WHERE player_id = $2 AND instance_id = $3
			ORDER BY created_at DESC LIMIT 1
		`, fmt.Sprintf("%d hours", template.CooldownHours), playerID, req.InstanceID).Scan(&remainingSeconds)
		if err == nil && remainingSeconds > 0 {
			hours := remainingSeconds / 3600
			minutes := (remainingSeconds % 3600) / 60
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo hữu cần đợi %d giờ %d phút nữa mới có thể vào lại mộng cảnh này!", hours, minutes)})
			return
		}
		if err != nil && err != sql.ErrNoRows {
			httperr.Respond(c, httperr.Internal("Database error: "+err.Error(), err))
			return
		}
	}

	// 3. Load player stats
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to load player data: "+err.Error(), err))
		return
	}

	// 4. Validate requirements
	if player.Rank < template.RequiredRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cảnh giới của đạo hữu chưa đủ yêu cầu để vào mộng cảnh này!"})
		return
	}

	if template.Type == "DREAM_REALM" && player.SoulHP <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Linh hồn đạo hữu đang ở trạng thái ngủ say / hôn mê, không thể thám hiểm mộng cảnh!"})
		return
	}

	// Consume entry item if required
	if template.EntryItem != "NONE" {
		qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), template.EntryItem)
		if err != nil || qty < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu thiếu vật phẩm \"" + template.EntryItem + "\" để tiến vào!"})
			return
		}
		if err := InventorySvc.RemoveItem(tx, player.ID.String(), template.EntryItem, "materials", 1); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to consume entry item: "+err.Error(), err))
			return
		}
	}

	// 5. Create player session
	var session PlayerInstanceSession
	session.ID = uuid.New()
	session.PlayerID = playerID
	session.InstanceID = template.ID
	session.CurrentStage = 0
	session.Status = "ACTIVE"
	session.TempInventory = json.RawMessage("[]")

	_, err = tx.Exec(`
		INSERT INTO player_instance_sessions (id, player_id, instance_id, current_stage, status, temp_inventory)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, session.ID, session.PlayerID, session.InstanceID, session.CurrentStage, session.Status, session.TempInventory)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to create instance session: "+err.Error(), err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Failed to commit: "+err.Error(), err))
		return
	}

	if template.EntryItem != "NONE" {
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID:       &playerID,
			Action:         "instance.enter",
			EntityType:     "instance_template",
			EntityID:       template.ID,
			RequestSummary: fmt.Sprintf("Tiến vào mộng cảnh %s | Tiêu hao vé vào: 1x %s", template.Name, template.EntryItem),
			Metadata: map[string]interface{}{
				"instance_id":   template.ID,
				"instance_name": template.Name,
				"entry_item":    template.EntryItem,
			},
			IPAddress: c.ClientIP(),
			Success:   true,
		})
	}

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message":  "Tiến vào mộng cảnh \"" + template.Name + "\" thành công!",
		"session":  session,
		"template": template,
	})
}

// ActionInstance processes player action inside the instance
func ActionInstance(c *gin.Context) {
	var req ActionInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "option_index is required"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Transaction failure", err))
		return
	}
	defer tx.Rollback()

	// 1. Get active session
	var session PlayerInstanceSession
	err = tx.QueryRow(`
		SELECT id, player_id, instance_id, current_stage, status, temp_inventory
		FROM player_instance_sessions
		WHERE player_id = $1 AND status = 'ACTIVE'
		LIMIT 1
	`, playerID).Scan(
		&session.ID, &session.PlayerID, &session.InstanceID,
		&session.CurrentStage, &session.Status, &session.TempInventory,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có mộng cảnh nào đang hoạt động!"})
		} else {
			httperr.Respond(c, httperr.Internal("Database error: "+err.Error(), err))
		}
		return
	}

	// 2. Fetch the template
	var template InstanceTemplate
	err = tx.QueryRow(`
		SELECT id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours
		FROM instance_templates
		WHERE id = $1
	`, session.InstanceID).Scan(
		&template.ID, &template.Name, &template.Type, &template.RequiredRank,
		&template.EntryItem, &template.Description, &template.Rewards, &template.MapData,
		&template.CooldownHours,
	)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to fetch instance template: "+err.Error(), err))
		return
	}

	// 3. Load player
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to load player status: "+err.Error(), err))
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	// 4. Parse map data stages
	var stages []InstanceStage
	if err := json.Unmarshal(template.MapData, &stages); err != nil {
		httperr.Respond(c, httperr.Internal("Failed to parse template map data", err))
		return
	}

	var currentStage *InstanceStage
	for i := range stages {
		if stages[i].StageIndex == session.CurrentStage {
			currentStage = &stages[i]
			break
		}
	}

	if currentStage == nil {
		httperr.Respond(c, httperr.Internal("Không tìm thấy giai đoạn hiện tại của phó bản!", currentStage))
		return
	}

	if req.OptionIndex < 0 || req.OptionIndex >= len(currentStage.Options) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lựa chọn không hợp lệ!"})
		return
	}

	option := currentStage.Options[req.OptionIndex]

	// 5. Verify option requirements
	if option.Requirements != nil {
		// Min Soul HP requirement
		if minSoulVal, hasMinSoul := option.Requirements["min_soul_hp"]; hasMinSoul {
			minSoulFloat, ok := minSoulVal.(float64)
			if !ok {
				httperr.Respond(c, httperr.Internal("Invalid min_soul_hp format in template", nil))
				return
			}
			if float64(player.SoulHP) < minSoulFloat {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Hồn phách của đạo hữu quá yếu đuối, không đủ yêu cầu để chọn phương án này!"})
				return
			}
		}

		// Material requirement
		if matIDVal, hasMat := option.Requirements["material_id"]; hasMat {
			matID, ok1 := matIDVal.(string)
			amtVal, hasAmt := option.Requirements["amount"]
			amtFloat, ok2 := amtVal.(float64)
			if !ok1 || !ok2 || !hasAmt {
				httperr.Respond(c, httperr.Internal("Invalid material requirement template", nil))
				return
			}
			requiredAmt := int64(amtFloat)
			qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), matID)
			if err != nil || qty < requiredAmt {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ nguyên liệu \"" + matID + "\" yêu cầu!"})
				return
			}
			// Deduct material
			if err := InventorySvc.RemoveItem(tx, player.ID.String(), matID, "materials", requiredAmt); err != nil {
				httperr.Respond(c, httperr.Internal("Failed to deduct material: "+err.Error(), err))
				return
			}
		}

		// Gu bug requirement
		if guIDVal, hasGu := option.Requirements["gu_id"]; hasGu {
			guID, ok1 := guIDVal.(string)
			amtVal, hasAmt := option.Requirements["amount"]
			amtFloat, ok2 := amtVal.(float64)
			if !ok1 || !ok2 || !hasAmt {
				httperr.Respond(c, httperr.Internal("Invalid gu requirement template", nil))
				return
			}
			requiredAmt := int(amtFloat)

			// Check if player has enough Gu
			matchingGus := 0
			var keptGus []models.GuInstance
			var consumedIDs []string
			var removedCount int

			for _, gi := range player.GuInventory {
				if gi.GuID == guID && !gi.IsVital && !gi.Equipped && removedCount < requiredAmt {
					removedCount++
					consumedIDs = append(consumedIDs, gi.InstanceID)
				} else {
					keptGus = append(keptGus, gi)
				}
				if gi.GuID == guID && !gi.IsVital && !gi.Equipped {
					matchingGus++
				}
			}

			if matchingGus < requiredAmt {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có đủ Cổ trùng yêu cầu (Bản Mệnh/Trang bị không được dùng)!"})
				return
			}
			player.GuInventory = keptGus
			// Explicit delete — SyncPlayerGus no longer deletes.
			if len(consumedIDs) > 0 {
				if err := db.RemovePlayerGus(tx, consumedIDs); err != nil {
					httperr.Respond(c, httperr.Internal("Failed to consume Gu for instance", err))
					return
				}
			}
		}
	}

	// 6. Apply health / soul HP damage parsed from option text
	soulDeduct, normalDeduct := parseHPEffects(option.Text)
	if template.Type == "DREAM_REALM" && soulDeduct > 0 {
		dreamMarks := int64(0)
		if player.DaoMarks != nil {
			dreamMarks = player.DaoMarks["dream"]
		}
		hasDreamGu := false
		for _, g := range models.FilterEquippedActive(player.GuInventory) {
			if spec, ok := models.GetGu(g.GuID, g.Rank); ok {
				for _, p := range spec.Path {
					if strings.Contains(strings.ToLower(p), "dream") {
						hasDreamGu = true
						break
					}
				}
			}
		}
		reduction := 0.0
		if dreamMarks > 0 {
			reduction += math.Min(0.3, float64(dreamMarks)/1000.0*0.1)
		}
		if hasDreamGu {
			reduction += 0.25
		}
		if reduction > 0.5 {
			reduction = 0.5
		}
		if reduction > 0 {
			saved := int(float64(soulDeduct) * reduction)
			soulDeduct -= saved
		}
	}
	player.SoulHP -= soulDeduct
	player.HP -= float64(normalDeduct)

	qiGained := 0.0

	var logResult string
	sessionFailed := false

	if player.HP <= 0 {
		player.HP = 1
		sessionFailed = true
		logResult = "Đạo hữu bị thương nặng dẫn đến thám hiểm mộng cảnh thất bại!"
	}

	if player.SoulHP <= 0 {
		player.SoulHP = 1
		// 80% mana reduction as a coma penalty
		player.CurrentMana = player.CurrentMana * 0.2
		sessionFailed = true
		logResult = "Hồn phách của đạo hữu kiệt quệ hoàn toàn rơi vào trạng thái hôn mê thảm hại! Thám hiểm mộng cảnh thất bại."
	}

	if sessionFailed {
		_, err = tx.Exec(`
			UPDATE player_instance_sessions
			SET status = 'FAILED'
			WHERE id = $1
		`, session.ID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update session: "+err.Error(), err))
			return
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update player: "+err.Error(), err))
			return
		}
		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, current_mana = $3, human_qi = $4
			WHERE id = $5
		`, player.HP, player.SoulHP, player.CurrentMana, player.HumanQi, player.ID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update player: "+err.Error(), err))
			return
		}

		if err := tx.Commit(); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to commit session failure: "+err.Error(), err))
			return
		}

		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"message": "Phó bản thất bại! " + logResult,
			"session": session,
		})
		return
	}

	// 7. Move stage or complete
	nextStage := option.NextStage
	if nextStage >= 0 {
		// Move to next stage
		_, err = tx.Exec(`
			UPDATE player_instance_sessions
			SET current_stage = $1
			WHERE id = $2
		`, nextStage, session.ID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update stage: "+err.Error(), err))
			return
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update player: "+err.Error(), err))
			return
		}
		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, human_qi = $3
			WHERE id = $4
		`, player.HP, player.SoulHP, player.HumanQi, player.ID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update player: "+err.Error(), err))
			return
		}

		if err := tx.Commit(); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to commit transition: "+err.Error(), err))
			return
		}

		if len(option.Requirements) > 0 {
			reqBytes, _ := json.Marshal(option.Requirements)
			auditlog.Log(db.DB, auditlog.Entry{
				PlayerID:       &playerID,
				Action:         "instance.choose_option",
				EntityType:     "instance_session",
				EntityID:       session.ID.String(),
				RequestSummary: fmt.Sprintf("Kỳ ngộ mộng cảnh %s: %s | Tiêu hao: %s", template.Name, option.Text, string(reqBytes)),
				Metadata: map[string]interface{}{
					"instance_id":  template.ID,
					"option_text":  option.Text,
					"requirements": option.Requirements,
				},
				IPAddress: c.ClientIP(),
				Success:   true,
			})
		}

		// Reload session for response
		session.CurrentStage = nextStage
		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"message":  fmt.Sprintf("Hành động thành công! Nhân Khí +%.0f.", qiGained),
			"session":  session,
			"template": template,
		})
		return
	}

	// Complete mộng cảnh! nextStage can be -1 (COMPLETED) or -2 (FAILED/QUIETLY EXIT)
	if nextStage == -1 {
		// Award rewards
		var rewards InstanceReward
		if err := json.Unmarshal(template.Rewards, &rewards); err == nil {
			// Add materials
			if rewards.Materials != nil {
				InventorySvc.BulkAddItems(tx, player.ID.String(), "materials", rewards.Materials)
			}

			// Add stones
			player.PrimevalStones += rewards.PrimevalStones

			// Add path attainment (True Meaning / Chân Ý) as Cảnh Giới Lưu Phái points
			if rewards.Attainment != nil && rewards.Attainment.Path != "" {
				if player.PathMasteries == nil {
					player.PathMasteries = make(models.PathMasteriesMap)
				}
				// Clean and normalize path name to lowercase
				cleanPath := strings.TrimSuffix(strings.ToLower(rewards.Attainment.Path), " path")

				attainmentGained := rewards.Attainment.Amount
				extraDreamMsg := ""
				// Mộng Đạo: Thấu Triệt Mộng Cảnh - tăng Chân Ý nếu có căn cơ Mộng Đạo
				if cleanPath == "dream" || (player.DaoMarks != nil && player.DaoMarks["dream"] > 0) {
					dreamMarks := int64(0)
					if player.DaoMarks != nil {
						dreamMarks = player.DaoMarks["dream"]
					}
					bonusAttainment := int(float64(attainmentGained) * (0.25 + float64(dreamMarks)/2000.0))
					if bonusAttainment > 0 {
						attainmentGained += bonusAttainment
						extraDreamMsg = fmt.Sprintf(" [Mộng Đạo - Thấu Triệt Mộng Cảnh: nhận thêm +%d Chân Ý nhờ căn cơ Mộng Đạo!]", bonusAttainment)
					}
				}

				player.PathMasteries[cleanPath] += attainmentGained
				newLevel := gamefx.PathMasteryLevel(player.PathMasteries[cleanPath])
				logResult = "Chúc mừng đạo hữu xuất sắc vượt ải thành công, nhận được " + fmt.Sprintf("%d", attainmentGained) + " điểm Chân Ý lưu phái \"" + rewards.Attainment.Path + "\" (Cảnh giới: " + gamefx.PathMasteryLevelName(newLevel) + ")!" + extraDreamMsg
			} else if len(rewards.DaoMarks) > 0 {
				// Bí cảnh: award Đạo Ngân (Dao Marks) directly.
				if _, err := addMultipleDaoMarksTx(tx, player.ID.String(), rewards.DaoMarks); err != nil {
					httperr.Respond(c, httperr.Internal("Lỗi cộng Đạo Ngân từ mộng cảnh: "+err.Error(), err))
					return
				}
				daoTexts := make([]string, 0, len(rewards.DaoMarks))
				for path, amt := range rewards.DaoMarks {
					daoTexts = append(daoTexts, fmt.Sprintf("%d Đạo Ngân %s", amt, path))
				}
				logResult = "Chúc mừng đạo hữu chinh phục mộng cảnh thành công, nhận được " + strings.Join(daoTexts, ", ") + "!"
			} else {
				logResult = "Chúc mừng đạo hữu đã chinh phục thành công mộng cảnh này!"
			}
		}

		_, err = tx.Exec(`
			UPDATE player_instance_sessions
			SET status = 'COMPLETED'
			WHERE id = $1
		`, session.ID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to complete session: "+err.Error(), err))
			return
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update player rewards: "+err.Error(), err))
			return
		}
		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, primeval_stones = $3, path_masteries = $4, human_qi = $5
			WHERE id = $6
		`, player.HP, player.SoulHP, player.PrimevalStones, player.PathMasteries, player.HumanQi, player.ID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Failed to update player rewards: "+err.Error(), err))
			return
		}

		if err := tx.Commit(); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to commit rewards: "+err.Error(), err))
			return
		}

		session.Status = "COMPLETED"
		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"message": "Phó bản hoàn thành! " + logResult + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
			"session": session,
		})
		return
	}

	// If nextStage == -2, failed quietly or left inheritance
	_, err = tx.Exec(`
		UPDATE player_instance_sessions
		SET status = 'FAILED'
		WHERE id = $1
	`, session.ID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to fail session: "+err.Error(), err))
		return
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		httperr.Respond(c, httperr.Internal("Failed to update player: "+err.Error(), err))
		return
	}
	_, err = tx.Exec(`
		UPDATE players
		SET hp = $1, soul_hp = $2, human_qi = $3
		WHERE id = $4
	`, player.HP, player.SoulHP, player.HumanQi, player.ID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to update player status: "+err.Error(), err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Failed to commit fail session: "+err.Error(), err))
		return
	}

	session.Status = "FAILED"
	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": "Đạo hữu thất bại thảm hại trong cuộc thám hiểm này!",
		"session": session,
	})
}
