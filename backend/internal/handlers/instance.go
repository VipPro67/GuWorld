package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

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
	Materials      map[string]int            `json:"materials"`
	PrimevalStones float64                   `json:"primeval_stones"`
	Attainment     *InstanceRewardAttainment `json:"attainment"`
	DaoMarks       map[string]int            `json:"dao_marks"`
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return
	}

	var template InstanceTemplate
	tplQuery := `
		SELECT id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours
		FROM instance_templates
		WHERE id = $1
	`
	err = db.DB.QueryRowContext(c.Request.Context(), tplQuery, session.InstanceID).Scan(
		&template.ID, &template.Name, &template.Type, &template.RequiredRank,
		&template.EntryItem, &template.Description, &template.Rewards, &template.MapData,
		&template.CooldownHours,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch instance template: " + err.Error()})
		return
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

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, name, type, required_rank, entry_item, description, rewards, cooldown_hours
		FROM instance_templates
		ORDER BY required_rank ASC, id ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch instance templates: " + err.Error()})
		return
	}
	defer rows.Close()

	items := []InstanceListItem{}
	for rows.Next() {
		var it InstanceListItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Type, &it.RequiredRank, &it.EntryItem, &it.Description, &it.Rewards, &it.CooldownHours); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan instance template: " + err.Error()})
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to iterate instance templates: " + err.Error()})
		return
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

	c.JSON(http.StatusOK, gin.H{"instances": items})
}

// EnterInstance initializes a new instance session
func EnterInstance(c *gin.Context) {
	var req EnterInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id is required"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction start failed"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang thám hiểm một phó bản khác!"})
		return
	}

	// 2. Fetch the instance template
	var template InstanceTemplate
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phó bản yêu cầu!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return
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
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo hữu cần đợi %d giờ %d phút nữa mới có thể vào lại phó bản này!", hours, minutes)})
			return
		}
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}
	}

	// 3. Load player stats
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player data: " + err.Error()})
		return
	}

	// 4. Validate requirements
	if player.Rank < template.RequiredRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cảnh giới của đạo hữu chưa đủ yêu cầu để vào phó bản này!"})
		return
	}

	if template.Type == "DREAM_REALM" && player.SoulHP <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Linh hồn đạo hữu đang ở trạng thái ngủ say / hôn mê, không thể thám hiểm mộng cảnh!"})
		return
	}

	// Consume entry item if required
	if template.EntryItem != "NONE" {
		qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), template.EntryItem, "materials")
		if err != nil || qty < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu thiếu vật phẩm \"" + template.EntryItem + "\" để tiến vào!"})
			return
		}
		if err := InventorySvc.RemoveItem(tx, player.ID.String(), template.EntryItem, "materials", 1); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to consume entry item: " + err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create instance session: " + err.Error()})
		return
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 2, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Tiến vào phó bản \"" + template.Name + "\" thành công! Nhân Khí +2.",
		"session":         session,
		"template":        template,
		"human_qi_gained": 2.0,
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failure"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có phó bản nào đang hoạt động!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch instance template: " + err.Error()})
		return
	}

	// 3. Load player
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player status: " + err.Error()})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	// 4. Parse map data stages
	var stages []InstanceStage
	if err := json.Unmarshal(template.MapData, &stages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse template map data"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy giai đoạn hiện tại của phó bản!"})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid min_soul_hp format in template"})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid material requirement template"})
				return
			}
			requiredAmt := int(amtFloat)
			qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), matID, "materials")
			if err != nil || qty < requiredAmt {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ nguyên liệu \"" + matID + "\" yêu cầu!"})
				return
			}
			// Deduct material
			if err := InventorySvc.RemoveItem(tx, player.ID.String(), matID, "materials", requiredAmt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct material: " + err.Error()})
				return
			}
		}

		// Gu bug requirement
		if guIDVal, hasGu := option.Requirements["gu_id"]; hasGu {
			guID, ok1 := guIDVal.(string)
			amtVal, hasAmt := option.Requirements["amount"]
			amtFloat, ok2 := amtVal.(float64)
			if !ok1 || !ok2 || !hasAmt {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid gu requirement template"})
				return
			}
			requiredAmt := int(amtFloat)

			// Check if player has enough Gu
			matchingGus := 0
			var keptGus []models.GuInstance
			var removedCount int

			for _, gi := range player.GuInventory {
				if gi.GuID == guID && removedCount < requiredAmt {
					removedCount++
				} else {
					keptGus = append(keptGus, gi)
				}
				if gi.GuID == guID {
					matchingGus++
				}
			}

			if matchingGus < requiredAmt {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có đủ Cổ trùng yêu cầu!"})
				return
			}
			player.GuInventory = keptGus
		}
	}

	// 6. Apply health / soul HP damage parsed from option text
	soulDeduct, normalDeduct := parseHPEffects(option.Text)
	player.SoulHP -= soulDeduct
	player.HP -= float64(normalDeduct)

	qiGained := AddHumanQi(&player, 3.0)

	var logResult string
	sessionFailed := false

	if player.HP <= 0 {
		player.HP = 1
		sessionFailed = true
		logResult = "Đạo hữu bị thương nặng dẫn đến thám hiểm phó bản thất bại!"
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session: " + err.Error()})
			return
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player: " + err.Error()})
			return
		}
		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, current_mana = $3, human_qi = $4
			WHERE id = $5
		`, player.HP, player.SoulHP, player.CurrentMana, player.HumanQi, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit session failure: " + err.Error()})
			return
		}

		responseWithPlayer(c, http.StatusOK, player, gin.H{
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stage: " + err.Error()})
			return
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player: " + err.Error()})
			return
		}
		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, human_qi = $3
			WHERE id = $4
		`, player.HP, player.SoulHP, player.HumanQi, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transition: " + err.Error()})
			return
		}

		// Reload session for response
		session.CurrentStage = nextStage
		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"message":  fmt.Sprintf("Hành động thành công! Nhân Khí +%.0f.", qiGained),
			"session":  session,
			"template": template,
		})
		return
	}

	// Complete phó bản! nextStage can be -1 (COMPLETED) or -2 (FAILED/QUIETLY EXIT)
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

				player.PathMasteries[cleanPath] += rewards.Attainment.Amount
				newLevel := gamefx.PathMasteryLevel(player.PathMasteries[cleanPath])
				logResult = "Chúc mừng đạo hữu xuất sắc vượt ải thành công, nhận được " + fmt.Sprintf("%d", rewards.Attainment.Amount) + " điểm Chân Ý lưu phái \"" + rewards.Attainment.Path + "\" (Cảnh giới: " + gamefx.PathMasteryLevelName(newLevel) + ")!"
			} else if len(rewards.DaoMarks) > 0 {
				// Bí cảnh: award Đạo Ngân (Dao Marks) directly.
				if _, err := addMultipleDaoMarksTx(tx, player.ID.String(), rewards.DaoMarks); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Đạo Ngân từ bí cảnh: " + err.Error()})
					return
				}
				daoTexts := make([]string, 0, len(rewards.DaoMarks))
				for path, amt := range rewards.DaoMarks {
					daoTexts = append(daoTexts, fmt.Sprintf("%d Đạo Ngân %s", amt, path))
				}
				logResult = "Chúc mừng đạo hữu chinh phục bí cảnh thành công, nhận được " + strings.Join(daoTexts, ", ") + "!"
			} else {
				logResult = "Chúc mừng đạo hữu đã chinh phục thành công phó bản này!"
			}
		}

		_, err = tx.Exec(`
			UPDATE player_instance_sessions
			SET status = 'COMPLETED'
			WHERE id = $1
		`, session.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete session: " + err.Error()})
			return
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player rewards: " + err.Error()})
			return
		}
		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, primeval_stones = $3, path_masteries = $4, human_qi = $5
			WHERE id = $6
		`, player.HP, player.SoulHP, player.PrimevalStones, player.PathMasteries, player.HumanQi, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player rewards: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit rewards: " + err.Error()})
			return
		}

		session.Status = "COMPLETED"
		responseWithPlayer(c, http.StatusOK, player, gin.H{
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fail session: " + err.Error()})
		return
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player: " + err.Error()})
		return
	}
	_, err = tx.Exec(`
		UPDATE players
		SET hp = $1, soul_hp = $2, human_qi = $3
		WHERE id = $4
	`, player.HP, player.SoulHP, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player status: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit fail session: " + err.Error()})
		return
	}

	session.Status = "FAILED"
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": "Đạo hữu thất bại thảm hại trong cuộc thám hiểm này!",
		"session": session,
	})
}
