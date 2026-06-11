package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
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

// GenerateCell procedurally generates the cell content based on region, x, y coordinates
func GenerateCell(x, y int, region string) models.Cell {
	respawnWindow := time.Now().Unix() / 180
	if x == 50 && y == 50 {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: "Thanh Bản Thôn (Safe Village)"}
	}

	if x < 0 || x >= 100 || y < 0 || y >= 100 {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: "Biên Giới Cổ Giới (World Edge)"}
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
		return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
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
		return models.Cell{X: x, Y: y, Type: "Resource", Content: res}
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
		return models.Cell{X: x, Y: y, Type: "Monster", Content: monster}
	} else {
		// Region-specific Gu bugs
		var gu string
		switch region {
		case "three_kings_inheritance":
			sub := rng.Intn(3)
			switch sub {
			case 0:
				gu = "GU_EARTH_COMMUNICATION"
			case 1:
				gu = "GU_TUSITA_FLOWER"
			default:
				gu = "GU_CLEANSING_WATER"
			}
		case "gu_luan_slope":
			sub := rng.Intn(3)
			switch sub {
			case 0:
				gu = "GU_WHITE_BOAR"
			case 1:
				gu = "GU_BLACK_BOAR"
			default:
				gu = "GU_SHADOW_STEPS"
			}
		default: // qing_mao_mountain
			sub := rng.Intn(3)
			switch sub {
			case 0:
				gu = "GU_MOONLIGHT"
			case 1:
				gu = "GU_JADE_SKIN"
			default:
				gu = "GU_LIQUOR_WORM"
			}
		}
		return models.Cell{X: x, Y: y, Type: "WildGu", Content: gu}
	}
}

func ResolveCell(tx *sql.Tx, region string, x int, y int) (models.Cell, error) {
	special, ok, err := db.FindSpecialCell(tx, region, x, y)
	if err == nil && ok {
		return special, nil
	}

	spawn, ok, err := db.FindFixedSpawn(tx, region, x, y)
	if err == nil && ok {
		return models.Cell{X: x, Y: y, Type: "Monster", Content: spawn.MonsterID}, nil
	}

	cell := GenerateCell(x, y, region)

	if cell.Type == "Monster" {
		monsterID, err := db.PickRandomMonsterFromRegion(tx, region)
		if err == nil && monsterID != "" {
			cell.Content = monsterID
		}
	}

	return cell, nil
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

func isCellClearedRecently(player *models.Player, x, y int, cooldown time.Duration) bool {
	if player.ClearedCells == nil {
		return false
	}
	coordKey := fmt.Sprintf("%d,%d", x, y)
	clearedTimeStr, ok := player.ClearedCells[coordKey]
	if !ok {
		return false
	}
	t, err := time.Parse(time.RFC3339, clearedTimeStr)
	if err != nil {
		return false
	}
	return time.Since(t) < cooldown
}

// cleanupExpiredClearedCells removes cells from player.ClearedCells that have exceeded the cooldown duration.
// Returns true if any cells were removed.
func cleanupExpiredClearedCells(player *models.Player, cooldown time.Duration) bool {
	if player.ClearedCells == nil {
		return false
	}
	changed := false
	now := time.Now()
	for coord, timeStr := range player.ClearedCells {
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			delete(player.ClearedCells, coord)
			changed = true
			continue
		}
		if now.Sub(t) >= cooldown {
			delete(player.ClearedCells, coord)
			changed = true
		}
	}
	return changed
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

	cleaned := cleanupExpiredClearedCells(&player, 2*time.Minute)
	if cleaned {
		_, err = tx.Exec("UPDATE players SET cleared_cells = $1 WHERE id = $2", player.ClearedCells, player.ID)
		if err != nil {
			log.Println("Failed to save cleaned up cleared_cells:", err)
		}
	}

	stats := CalculatePlayerStats(&player)
	visionRadius := int(stats["investigation"])
	if visionRadius < 1 {
		visionRadius = 1
	}

	cells := []models.Cell{}
	for dx := -visionRadius; dx <= visionRadius; dx++ {
		for dy := -visionRadius; dy <= visionRadius; dy++ {
			cx := player.X + dx
			cy := player.Y + dy
			if cx >= 0 && cx < 100 && cy >= 0 && cy < 100 {
				cell, err := ResolveCell(tx, player.CurrentRegion, cx, cy)
				if err != nil {
					cell = GenerateCell(cx, cy, player.CurrentRegion)
				}
				if isCellClearedRecently(&player, cx, cy, 2*time.Minute) && (cell.Type == "Resource" || cell.Type == "WildGu") {
					cell = models.Cell{X: cx, Y: cy, Type: "Empty", Content: ""}
				}
				cells = append(cells, cell)
			}
		}
	}

	type NearbyPlayer struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		X        int    `json:"x"`
		Y        int    `json:"y"`
	}
	nearbyPlayers := []NearbyPlayer{}
	rows, errQuery := tx.Query(`
		SELECT id, username, x, y 
		FROM players 
		WHERE current_region = $1 
		  AND id != $2 
		  AND (current_region IN ('qing_mao_mountain', 'gu_luan_slope', 'three_kings_inheritance') OR last_mana_update >= NOW() - INTERVAL '15 minutes')
		  AND x >= $3 AND x <= $4
		  AND y >= $5 AND y <= $6
	`, player.CurrentRegion, player.ID, player.X-visionRadius, player.X+visionRadius, player.Y-visionRadius, player.Y+visionRadius)
	if errQuery == nil {
		defer rows.Close()
		for rows.Next() {
			var np NearbyPlayer
			if errScan := rows.Scan(&np.ID, &np.Username, &np.X, &np.Y); errScan == nil {
				nearbyPlayers = append(nearbyPlayers, np)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Println("GetGrid commit warning:", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"x":              player.X,
		"y":              player.Y,
		"vision_radius":  visionRadius,
		"cells":          cells,
		"explored_cells": player.ExploredCells,
		"stats":          stats,
		"players_on_map": nearbyPlayers,
	})
}

// HasExplorationInventorySpace checks bag capacity including temporary loot
func HasExplorationInventorySpace(player *models.Player, incomingCount int) bool {
	stats := CalculatePlayerStats(player)
	maxSlots := int(stats["storage"])
	if player.Rank >= 6 {
		maxSlots = 999
	}

	currentSize := len(player.GuInventory)
	for _, qty := range player.MaterialsInventory {
		if qty > 0 {
			currentSize++
		}
	}

	currentSize += len(player.TempLoot.GuInventory)
	for _, qty := range player.TempLoot.MaterialsInventory {
		if qty > 0 {
			currentSize++
		}
	}

	return (currentSize + incomingCount) <= maxSlots
}

// MoveRequest specifies target position
type MoveRequest struct {
	X              int    `json:"x"`
	Y              int    `json:"y"`
	AttackTargetID string `json:"attack_target_id,omitempty"`
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

	cleanupExpiredClearedCells(&player, 2*time.Minute)

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

	if dist == 0 && req.AttackTargetID == "" {
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
	isCleared := isCellClearedRecently(&player, player.X, player.Y, 2*time.Minute)

	// Trigger cell action
	cell, err := ResolveCell(tx, player.CurrentRegion, player.X, player.Y)
	if err != nil {
		cell = GenerateCell(player.X, player.Y, player.CurrentRegion)
	}
	logMsg := fmt.Sprintf("Di chuyển thành công đến (%d, %d).", player.X, player.Y)
	combatTriggered := false
	pvpTriggered := false

	// Check if there is another active player (not in same party) at req.X, req.Y
	var targetPlayer models.Player
	var hasTargetPlayer bool
	queryStr := `
		SELECT id, username, rank, sub_rank, hp, max_mana, current_mana, base_hp, path_masteries, gu_inventory, dao_marks
		FROM players
		WHERE current_region = $1 AND x = $2 AND y = $3 AND id != $4
		  AND (current_region IN ('qing_mao_mountain', 'gu_luan_slope', 'three_kings_inheritance') OR last_mana_update >= NOW() - INTERVAL '15 minutes')
	`
	var errTarget error
	if player.PartyID != nil {
		queryStr += ` AND (party_id IS NULL OR party_id != $5)`
		errTarget = tx.QueryRow(queryStr, player.CurrentRegion, player.X, player.Y, player.ID, player.PartyID).Scan(
			&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
			&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.CurrentMana, &targetPlayer.BaseHP,
			&targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks,
		)
	} else {
		errTarget = tx.QueryRow(queryStr, player.CurrentRegion, player.X, player.Y, player.ID).Scan(
			&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
			&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.CurrentMana, &targetPlayer.BaseHP,
			&targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks,
		)
	}
	if errTarget == nil {
		hasTargetPlayer = true
	}

	if hasTargetPlayer && req.AttackTargetID == targetPlayer.ID.String() {
		combatTriggered = true
		pvpTriggered = true
		targetStats := CalculatePlayerStats(&targetPlayer)

		enemyHP := targetPlayer.HP
		if enemyHP <= 0 {
			enemyHP = targetPlayer.BaseHP
		}
		if enemyHP <= 0 {
			enemyHP = 100.0
		}
		enemyMaxHP := targetPlayer.BaseHP
		if enemyMaxHP <= 0 {
			enemyMaxHP = 100.0
		}

		pvpCombat := models.CombatState{
			EnemyTemplateID:    targetPlayer.ID.String(),
			EnemyName:          targetPlayer.Username,
			EnemyType:          "PVP",
			EnemyHP:            enemyHP,
			EnemyMaxHP:         enemyMaxHP,
			EnemyMana:          targetPlayer.CurrentMana,
			EnemyMaxMana:       targetPlayer.MaxMana,
			EnemyDamage:        targetStats["attack"],
			EnemyDefense:       targetStats["defense"],
			EnemyRank:          targetPlayer.Rank,
			EnemySubRank:       targetPlayer.SubRank,
			EnemyGuInventory:   targetPlayer.GuInventory,
			EnemyDaoMarks:      targetPlayer.DaoMarks,
			EnemyPathMasteries: targetPlayer.PathMasteries,
			TargetPlayerID:     targetPlayer.ID.String(),
		}
		player.ActiveCombat = &pvpCombat
		logMsg = fmt.Sprintf("Ngươi chủ động tuyên chiến và tấn công Cổ Sư %s tại tọa độ (%d, %d)!", targetPlayer.Username, player.X, player.Y)
	}

	if isCleared {
		logMsg = fmt.Sprintf("Di chuyển thành công đến (%d, %d). Vùng đất trống trải hoang tàn.", player.X, player.Y)
	} else if !pvpTriggered {
		switch cell.Type {
		case "Resource":
			if HasExplorationInventorySpace(&player, 1) {
				if player.TempLoot.MaterialsInventory == nil {
					player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
				}
				player.TempLoot.MaterialsInventory[cell.Content]++
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Đạo hữu tìm thấy nguyên liệu hoang dã: 1x %s (được giữ tạm thời trong bản đồ)!", player.X, player.Y, cell.Content)

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Phát hiện 1x %s, nhưng Bách Bảo Nang đã đầy!", player.X, player.Y, cell.Content)
			}
		case "WildGu":
			if HasExplorationInventorySpace(&player, 1) {
				newGu := models.GuInstance{
					InstanceID: uuid.New().String(),
					GuID:       cell.Content,
					Hunger:     100,
					Status:     "ACTIVE",
					Equipped:   false,
				}
				player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, newGu)
				logMsg = fmt.Sprintf("Cơ duyên! Đạo hữu bắt gặp và thu phục thành công Cổ trùng hoang dã: %s (được giữ tạm thời trong bản đồ)!", cell.Content)

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Đạo hữu phát hiện thấy Cổ trùng hoang dã %s, tiếc là Bách Bảo Nang đã đầy nên để nó trốn thoát!", cell.Content)
			}
		case "Monster", "Boss":
			combatTriggered = true

			// Load from database if possible
			mon, err := db.LoadMonsterTemplate(tx, cell.Content)
			if err == nil {
				logMsg = fmt.Sprintf("Đạo hữu chạm trán %s tại tọa độ (%d, %d)! (Quái vật được tải từ cơ sở dữ liệu)", mon.Name, player.X, player.Y)
				log.Println(logMsg)
				combatState := models.BuildCombatStateFromMonster(mon)
				player.ActiveCombat = &combatState
				logMsg = fmt.Sprintf("Nguy hiểm! Đạo hữu chạm trán %s tại tọa độ (%d, %d)!", mon.Name, player.X, player.Y)
			} else {
				// Fallback to hardcoded stats based on content string
				enemyHp := 60.0
				enemyDmg := 10.0
				enemyDef := 3.0
				name := cell.Content

				switch cell.Content {
				case "monster_lava_python", "Mãng Xà Dung Nham (Lava Python)":
					name = "Mãng Xà Dung Nham (Lava Python)"
					enemyHp = 150.0
					enemyDmg = 24.0
					enemyDef = 10.0
				case "monster_iron_bear", "Thiết Giáp Hùng (Iron Armored Bear)":
					name = "Thiết Giáp Hùng (Iron Armored Bear)"
					enemyHp = 200.0
					enemyDmg = 18.0
					enemyDef = 18.0
				case "monster_wolf_leader", "Độc Lang Vương (Wolf Leader)":
					name = "Độc Lang Vương (Wolf Leader)"
					enemyHp = 100.0
					enemyDmg = 15.0
					enemyDef = 6.0
				case "monster_boar_king", "Dã Trĩ Vương (Boar King)":
					name = "Dã Trĩ Vương (Boar King)"
					enemyHp = 120.0
					enemyDmg = 12.0
					enemyDef = 8.0
				case "monster_stone_ape", "Thạch Hầu (Stone Ape)":
					name = "Thạch Hầu (Stone Ape)"
					enemyHp = 60.0
					enemyDmg = 10.0
					enemyDef = 3.0
				case "monster_wild_boar", "Dã Trĩ (Wild Boar)":
					name = "Dã Trĩ (Wild Boar)"
					enemyHp = 50.0
					enemyDmg = 8.0
					enemyDef = 2.0
				}

				player.ActiveCombat = &models.CombatState{
					EnemyName:    name,
					EnemyHP:      enemyHp,
					EnemyMaxHP:   enemyHp,
					EnemyDamage:  enemyDmg,
					EnemyDefense: enemyDef,
				}
				logMsg = fmt.Sprintf("Nguy hiểm! Đạo hữu chạm trán với mãnh thú phục kích: %s tại tọa độ (%d, %d)!", name, player.X, player.Y)
			}
		case "PVP":
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

	if player.ActiveCombat != nil {
		player.ActiveCombat = StartCombat(tx, player.ActiveCombat, &player)
	}

	// Update DB
	updateQuery := `
		UPDATE players
		SET current_mana = $1, x = $2, y = $3, explored_cells = $4, gu_inventory = $5, materials_inventory = $6, cleared_cells = $7, active_combat = $8, temp_loot = $9, last_mana_update = $10
		WHERE id = $11
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.X, player.Y, player.ExploredCells, player.GuInventory, player.MaterialsInventory, player.ClearedCells, player.ActiveCombat, player.TempLoot, time.Now(), player.ID)
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
