package handlers

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Cell represents a cell on the grid map
type Cell struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Type    string `json:"cell_type"` // "Empty", "Resource", "Monster", "WildGu", "PVP"
	Content string `json:"content"`   // name/ID of item, beast, or Gu
}

// GenerateCell procedurally generates the cell content based on region, x, y coordinates
func GenerateCell(x, y int, region string) Cell {
	respawnWindow := time.Now().Unix() / 180
	if x == 50 && y == 50 {
		return Cell{X: x, Y: y, Type: "Empty", Content: "Thanh Bản Thôn (Safe Village)"}
	}

	if x < 0 || x >= 100 || y < 0 || y >= 100 {
		return Cell{X: x, Y: y, Type: "Empty", Content: "Biên Giới Cổ Giới (World Edge)"}
	}

	h := sha256.New()
	h.Write([]byte(region))
	binary.Write(h, binary.BigEndian, int64(x))
	binary.Write(h, binary.BigEndian, int64(y))
	binary.Write(h, binary.BigEndian, respawnWindow)
	seed := int64(binary.BigEndian.Uint64(h.Sum(nil)[:8]))

	rng := rand.New(rand.NewSource(seed))
	roll := rng.Intn(100)

	// 70% chance empty/PVP
	if roll < 70 {
		if roll >= 67 {
			return Cell{X: x, Y: y, Type: "PVP", Content: "cultivator"}
		}
		return Cell{X: x, Y: y, Type: "Empty", Content: ""}
	} else if roll < 85 {
		// Region-specific resources
		var res string
		switch region {
		case "qing_mao_mountain":
			if rng.Intn(10) < 8 {
				res = "moon_orchid_petal"
			} else {
				res = "green_bamboo_wine"
			}
		case "gu_luan_slope":
			if rng.Intn(10) < 6 {
				res = "jade_stone"
			} else {
				res = "boar_meat"
			}
		case "three_kings_inheritance":
			sub := rng.Intn(10)
			if sub < 4 {
				res = "wild_boar_tusk"
			} else if sub < 7 {
				res = "sour_wine"
			} else {
				res = "sweet_wine"
			}
		default:
			res = "moon_orchid_petal"
		}
		return Cell{X: x, Y: y, Type: "Resource", Content: res}
	} else if roll < 97 {
		// Region-specific monsters
		var monster string
		switch region {
		case "three_kings_inheritance":
			if rng.Intn(2) == 0 {
				monster = "Mãng Xà Dung Nham (Lava Python)"
			} else {
				monster = "Thiết Giáp Hùng (Iron Armored Bear)"
			}
		case "gu_luan_slope":
			if rng.Intn(2) == 0 {
				monster = "Độc Lang Vương (Wolf Leader)"
			} else {
				monster = "Dã Trĩ Vương (Boar King)"
			}
		default: // qing_mao_mountain
			if rng.Intn(2) == 0 {
				monster = "Dã Trĩ (Wild Boar)"
			} else {
				monster = "Thạch Hầu (Stone Ape)"
			}
		}
		return Cell{X: x, Y: y, Type: "Monster", Content: monster}
	} else {
		// Region-specific Gu bugs
		var gu string
		switch region {
		case "three_kings_inheritance":
			sub := rng.Intn(3)
			if sub == 0 {
				gu = "GU_EARTH_COMMUNICATION"
			} else if sub == 1 {
				gu = "GU_TUSITA_FLOWER"
			} else {
				gu = "GU_CLEANSING_WATER"
			}
		case "gu_luan_slope":
			sub := rng.Intn(3)
			if sub == 0 {
				gu = "GU_WHITE_BOAR"
			} else if sub == 1 {
				gu = "GU_BLACK_BOAR"
			} else {
				gu = "GU_SHADOW_STEPS"
			}
		default: // qing_mao_mountain
			sub := rng.Intn(3)
			if sub == 0 {
				gu = "GU_MOONLIGHT"
			} else if sub == 1 {
				gu = "GU_JADE_SKIN"
			} else {
				gu = "GU_LIQUOR_WORM"
			}
		}
		return Cell{X: x, Y: y, Type: "WildGu", Content: gu}
	}
}

// addExploredCells formats and appends visible coordinates to explored_cells JSONB list
func addExploredCells(player *models.Player, radius int) {
	exploredMap := make(map[string]bool)
	for _, cell := range player.ExploredCells {
		exploredMap[cell] = true
	}

	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			cx := player.X + dx
			cy := player.Y + dy
			if cx >= 0 && cx < 100 && cy >= 0 && cy < 100 {
				coord := fmt.Sprintf("%d,%d", cx, cy)
				exploredMap[coord] = true
			}
		}
	}

	newSlice := []string{}
	for k := range exploredMap {
		newSlice = append(newSlice, k)
	}
	player.ExploredCells = newSlice
}

// GetGrid retrieves the viewport viewport within vision range
func GetGrid(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	stats := CalculatePlayerStats(&player)
	visionRadius := int(stats["investigation"])
	if visionRadius < 1 {
		visionRadius = 1
	}

	cells := []Cell{}
	for dx := -visionRadius; dx <= visionRadius; dx++ {
		for dy := -visionRadius; dy <= visionRadius; dy++ {
			cx := player.X + dx
			cy := player.Y + dy
			if cx >= 0 && cx < 100 && cy >= 0 && cy < 100 {
				cells = append(cells, GenerateCell(cx, cy, player.CurrentRegion))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"x":              player.X,
		"y":              player.Y,
		"vision_radius":  visionRadius,
		"cells":          cells,
		"explored_cells": player.ExploredCells,
		"stats":          stats,
	})
}

// MoveRequest specifies target position
// MoveRequest specifies target position
type MoveRequest struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// MovePlayer moves the player on the 100x100 grid and triggers cell actions
func MovePlayer(c *gin.Context) {
	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.X < 0 || req.X >= 100 || req.Y < 0 || req.Y >= 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ vượt ra ngoài ranh giới bản đồ."})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đang trong trạng thái chiến đấu! Hãy kết thúc trận chiến trước."})
		return
	}

	stats := CalculatePlayerStats(&player)
	movementRange := int(stats["movement"])
	if movementRange < 1 {
		movementRange = 1
	}

	// Calculate Chebyshev distance
	dx := math.Abs(float64(req.X - player.X))
	dy := math.Abs(float64(req.Y - player.Y))
	dist := int(math.Max(dx, dy))

	if dist == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đang ở vị trí này rồi."})
		return
	}

	if dist > movementRange {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vị trí quá xa! Thân pháp hiện tại chỉ di chuyển tối đa %d ô.", movementRange)})
		return
	}

	// Movement no longer consumes mana
	player.X = req.X
	player.Y = req.Y

	// Add cells to explored list
	visionRadius := int(stats["investigation"])
	if visionRadius < 1 {
		visionRadius = 1
	}
	addExploredCells(&player, visionRadius)

	// Check if cell is cleared recently (2 minutes cooldown)
	coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
	isCleared := false
	if player.ClearedCells != nil {
		if clearedTimeStr, ok := player.ClearedCells[coordKey]; ok {
			if t, err := time.Parse(time.RFC3339, clearedTimeStr); err == nil {
				if time.Since(t) < 2*time.Minute {
					isCleared = true
				}
			}
		}
	}

	// Trigger cell action
	cell := GenerateCell(player.X, player.Y, player.CurrentRegion)
	logMsg := fmt.Sprintf("Di chuyển thành công đến (%d, %d).", player.X, player.Y)
	combatTriggered := false

	if isCleared {
		logMsg = fmt.Sprintf("Di chuyển thành công đến (%d, %d). Vùng đất trống trải hoang tàn.", player.X, player.Y)
	} else {
		if cell.Type == "Resource" {
			if HasInventorySpace(&player, 1) {
				if player.MaterialsInventory == nil {
					player.MaterialsInventory = make(models.MaterialsInventory)
				}
				player.MaterialsInventory[cell.Content]++
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Đạo hữu tìm thấy nguyên liệu hoang dã: 1x %s!", player.X, player.Y, cell.Content)

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Phát hiện 1x %s, nhưng Bách Bảo Nang đã đầy!", player.X, player.Y, cell.Content)
			}
		} else if cell.Type == "WildGu" {
			if HasInventorySpace(&player, 1) {
				newGu := models.GuInstance{
					InstanceID: uuid.New().String(),
					GuID:       cell.Content,
					Hunger:     100,
					Status:     "ACTIVE",
					Equipped:   false,
				}
				player.GuInventory = append(player.GuInventory, newGu)
				logMsg = fmt.Sprintf("Cơ duyên! Đạo hữu bắt gặp và thu phục thành công Cổ trùng hoang dã: %s!", cell.Content)

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Đạo hữu phát hiện thấy Cổ trùng hoang dã %s, tiếc là Bách Bảo Nang đã đầy nên để nó trốn thoát!", cell.Content)
			}
		} else if cell.Type == "Monster" {
			combatTriggered = true
			enemyHp := 60.0
			enemyDmg := 10.0
			enemyDef := 3.0

			if cell.Content == "Mãng Xà Dung Nham (Lava Python)" {
				enemyHp = 150.0
				enemyDmg = 24.0
				enemyDef = 10.0
			} else if cell.Content == "Thiết Giáp Hùng (Iron Armored Bear)" {
				enemyHp = 200.0
				enemyDmg = 18.0
				enemyDef = 18.0
			} else if cell.Content == "Độc Lang Vương (Wolf Leader)" {
				enemyHp = 100.0
				enemyDmg = 15.0
				enemyDef = 6.0
			} else if cell.Content == "Dã Trĩ Vương (Boar King)" {
				enemyHp = 120.0
				enemyDmg = 12.0
				enemyDef = 8.0
			} else if cell.Content == "Thạch Hầu (Stone Ape)" {
				enemyHp = 60.0
				enemyDmg = 10.0
				enemyDef = 3.0
			} else if cell.Content == "Dã Trĩ (Wild Boar)" {
				enemyHp = 50.0
				enemyDmg = 8.0
				enemyDef = 2.0
			}

			player.ActiveCombat = &models.CombatState{
				EnemyName:    cell.Content,
				EnemyHP:      enemyHp,
				EnemyMaxHP:   enemyHp,
				EnemyDamage:  enemyDmg,
				EnemyDefense: enemyDef,
			}
			logMsg = fmt.Sprintf("Nguy hiểm! Đạo hữu chạm trán với mãnh thú phục kích: %s tại tọa độ (%d, %d)!", cell.Content, player.X, player.Y)
		} else if cell.Type == "PVP" {
			var targetPlayer models.Player
			err := tx.QueryRow(`
				SELECT id, username, rank, sub_rank, hp, max_mana, path_masteries, gu_inventory, dao_marks
				FROM players 
				WHERE id != $1 
				LIMIT 1`, player.ID,
			).Scan(
				&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
				&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks,
			)

			combatTriggered = true
			if err != nil {
				player.ActiveCombat = &models.CombatState{
					EnemyName:      "Cổ Sư Lang Thang (Wandering Cultivator)",
					EnemyHP:        100.0,
					EnemyMaxHP:     100.0,
					EnemyDamage:    15.0,
					EnemyDefense:   5.0,
					TargetPlayerID: "dummy",
				}
				logMsg = "Đột ngột một Cổ Sư Lang Thang xuất hiện khiêu chiến đạo hữu!"
			} else {
				targetStats := CalculatePlayerStats(&targetPlayer)
				player.ActiveCombat = &models.CombatState{
					EnemyName:      targetPlayer.Username,
					EnemyHP:        100.0,
					EnemyMaxHP:     100.0,
					EnemyDamage:    targetStats["attack"],
					EnemyDefense:   targetStats["defense"],
					TargetPlayerID: targetPlayer.ID.String(),
				}
				logMsg = fmt.Sprintf("Đạo hữu bắt gặp một Cổ Sư khác đang du ngoạn: %s! Lập tức giao đấu!", targetPlayer.Username)
			}
		}
	}

	if !combatTriggered {
		for ax := -1; ax <= 1 && !combatTriggered; ax++ {
			for ay := -1; ay <= 1 && !combatTriggered; ay++ {
				if ax == 0 && ay == 0 {
					continue
				}
				adj := GenerateCell(player.X+ax, player.Y+ay, player.CurrentRegion)
				if adj.Type == "Monster" && rand.Intn(100) < 25 {
					combatTriggered = true
					player.ActiveCombat = &models.CombatState{EnemyName: adj.Content, EnemyHP: 70, EnemyMaxHP: 70, EnemyDamage: 11, EnemyDefense: 4}
					logMsg = fmt.Sprintf("Phục kích! %s chủ động lao ra tấn công đạo hữu tại (%d, %d)!", adj.Content, player.X, player.Y)
				}
			}
		}
	}

	// Update DB
	updateQuery := `
		UPDATE players
		SET current_mana = $1, x = $2, y = $3, explored_cells = $4, gu_inventory = $5, materials_inventory = $6, cleared_cells = $7, active_combat = $8
		WHERE id = $9
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.X, player.Y, player.ExploredCells, player.GuInventory, player.MaterialsInventory, player.ClearedCells, player.ActiveCombat, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	// Calculate fresh stats
	newStats := CalculatePlayerStats(&player)

	c.JSON(http.StatusOK, gin.H{
		"player":           player,
		"log":              logMsg,
		"combat_triggered": combatTriggered,
		"stats":            newStats,
	})
}
