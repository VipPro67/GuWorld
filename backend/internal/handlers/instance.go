package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type InstanceTemplate struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"` // "INHERITANCE_RIGHTEOUS", "INHERITANCE_DEMONIC", "DREAM_REALM"
	RequiredRank int             `json:"required_rank"`
	EntryItem    string          `json:"entry_item"`
	Description  string          `json:"description"`
	Rewards      json.RawMessage `json:"rewards"`
	MapData      json.RawMessage `json:"map_data"`
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
	PrimevalStones int                       `json:"primeval_stones"`
	Attainment     *InstanceRewardAttainment `json:"attainment"`
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
	err := db.DB.QueryRow(query, playerID).Scan(
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
		SELECT id, name, type, required_rank, entry_item, description, rewards, map_data
		FROM instance_templates
		WHERE id = $1
	`
	err = db.DB.QueryRow(tplQuery, session.InstanceID).Scan(
		&template.ID, &template.Name, &template.Type, &template.RequiredRank,
		&template.EntryItem, &template.Description, &template.Rewards, &template.MapData,
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
		SELECT id, name, type, required_rank, entry_item, description, rewards, map_data
		FROM instance_templates
		WHERE id = $1
	`, req.InstanceID).Scan(
		&template.ID, &template.Name, &template.Type, &template.RequiredRank,
		&template.EntryItem, &template.Description, &template.Rewards, &template.MapData,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phó bản yêu cầu!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return
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
		qty, ok := player.MaterialsInventory[template.EntryItem]
		if !ok || qty < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu thiếu vật phẩm \"" + template.EntryItem + "\" để tiến vào!"})
			return
		}
		player.MaterialsInventory[template.EntryItem]--
		if player.MaterialsInventory[template.EntryItem] == 0 {
			delete(player.MaterialsInventory, template.EntryItem)
		}

		_, err = tx.Exec(`
			UPDATE players SET materials_inventory = $1 WHERE id = $2
		`, player.MaterialsInventory, player.ID)
		if err != nil {
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

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Tiến vào phó bản \"" + template.Name + "\" thành công!",
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
		SELECT id, name, type, required_rank, entry_item, description, rewards, map_data
		FROM instance_templates
		WHERE id = $1
	`, session.InstanceID).Scan(
		&template.ID, &template.Name, &template.Type, &template.RequiredRank,
		&template.EntryItem, &template.Description, &template.Rewards, &template.MapData,
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
			playerQty, ok := player.MaterialsInventory[matID]
			if !ok || playerQty < requiredAmt {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ nguyên liệu \"" + matID + "\" yêu cầu!"})
				return
			}
			// Deduct material
			player.MaterialsInventory[matID] -= requiredAmt
			if player.MaterialsInventory[matID] == 0 {
				delete(player.MaterialsInventory, matID)
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

		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, current_mana = $3, materials_inventory = $4, gu_inventory = $5
			WHERE id = $6
		`, player.HP, player.SoulHP, player.CurrentMana, player.MaterialsInventory, player.GuInventory, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit session failure: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Phó bản thất bại! " + logResult,
			"session": session,
			"player":  player,
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

		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, materials_inventory = $3, gu_inventory = $4
			WHERE id = $5
		`, player.HP, player.SoulHP, player.MaterialsInventory, player.GuInventory, player.ID)
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
		c.JSON(http.StatusOK, gin.H{
			"message":  "Hành động thành công!",
			"session":  session,
			"template": template,
			"player":   player,
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
				if player.MaterialsInventory == nil {
					player.MaterialsInventory = make(models.MaterialsInventory)
				}
				for k, v := range rewards.Materials {
					player.MaterialsInventory[k] += v
				}
			}

			// Add stones
			player.PrimevalStones += rewards.PrimevalStones

			// Add path attainment (True Meaning / Chân Ý)
			if rewards.Attainment != nil && rewards.Attainment.Path != "" {
				if player.PathMasteries == nil {
					player.PathMasteries = make(models.PathMasteriesMap)
				}
				// Clean and normalize path name to lowercase
				cleanPath := strings.TrimSuffix(strings.ToLower(rewards.Attainment.Path), " path")
				
				currentLevelName := player.PathMasteries[cleanPath]
				currentLevel := 0
				switch currentLevelName {
				case "Ordinary":
					currentLevel = 0
				case "Master":
					currentLevel = 1
				case "Grandmaster":
					currentLevel = 2
				case "Great Grandmaster":
					currentLevel = 3
				case "Supreme Grandmaster":
					currentLevel = 4
				}

				// direct leap level based on Chân Ý amount
				leap := 1
				if rewards.Attainment.Amount >= 50 {
					leap = 2
				}
				newLevel := currentLevel + leap
				if newLevel > 4 {
					newLevel = 4
				}

				levelName := "Ordinary"
				switch newLevel {
				case 1:
					levelName = "Master"
				case 2:
					levelName = "Grandmaster"
				case 3:
					levelName = "Great Grandmaster"
				case 4:
					levelName = "Supreme Grandmaster"
				}

				player.PathMasteries[cleanPath] = levelName
				logResult = "Chúc mừng đạo hữu xuất sắc vượt ải thành công, nhận được Chân Ý thăng tiến \"" + rewards.Attainment.Path + "\" lên cảnh giới " + levelName + "!"
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

		_, err = tx.Exec(`
			UPDATE players
			SET hp = $1, soul_hp = $2, materials_inventory = $3, gu_inventory = $4, primeval_stones = $5, path_masteries = $6
			WHERE id = $7
		`, player.HP, player.SoulHP, player.MaterialsInventory, player.GuInventory, player.PrimevalStones, player.PathMasteries, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player rewards: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit rewards: " + err.Error()})
			return
		}

		session.Status = "COMPLETED"
		c.JSON(http.StatusOK, gin.H{
			"message": "Phó bản hoàn thành! " + logResult,
			"session": session,
			"player":  player,
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

	_, err = tx.Exec(`
		UPDATE players
		SET hp = $1, soul_hp = $2, materials_inventory = $3, gu_inventory = $4
		WHERE id = $5
	`, player.HP, player.SoulHP, player.MaterialsInventory, player.GuInventory, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player status: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit fail session: " + err.Error()})
		return
	}

	session.Status = "FAILED"
	c.JSON(http.StatusOK, gin.H{
		"message": "Đạo hữu thất bại thảm hại trong cuộc thám hiểm này!",
		"session": session,
		"player":  player,
	})
}
