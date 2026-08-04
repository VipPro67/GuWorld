package handlers

import (
	"crypto/sha256"
	"database/sql"
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
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const clearedCellCooldown = 15 * time.Minute

func GetCellSeed(x, y int, region string) int64 {
	respawnWindow := time.Now().Unix() / 180
	h := sha256.New()
	h.Write([]byte(region))
	binary.Write(h, binary.BigEndian, int64(x))
	binary.Write(h, binary.BigEndian, int64(y))
	binary.Write(h, binary.BigEndian, respawnWindow)
	return int64(binary.BigEndian.Uint64(h.Sum(nil)[:8]))
}

// GenerateCell procedurally generates the cell content based on region, x, y coordinates
func GenerateCell(x, y int, region string) models.Cell {
	if x == 50 && y == 50 {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: "Thanh Bản Thôn (Safe Village)"}
	}

	if x < 0 || x >= 100 || y < 0 || y >= 100 {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: "Biên Giới Cổ Giới (World Edge)"}
	}

	seed := GetCellSeed(x, y, region)
	rng := rand.New(rand.NewSource(seed))
	roll := rng.Intn(100)

	// 80% chance empty/PVP, 20% Monster
	if roll < 80 {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
	} else {
		return models.Cell{X: x, Y: y, Type: "Monster", Content: ""}
	}
}

func ResolveCell(tx *sql.Tx, region string, x int, y int) (models.Cell, error) {
	special, ok, err := db.FindSpecialCell(tx, region, x, y)
	if err == nil && ok {
		return special, nil
	}

	spawn, ok, err := db.FindFixedSpawn(tx, region, x, y)
	if err == nil && ok {
		seed := GetCellSeed(x, y, region)
		rng := rand.New(rand.NewSource(seed))
		t := "Monster"
		if spawn.IsBossSpawn {
			t = "Boss"
		}

		rank := spawn.MinRank
		if spawn.MaxRank > spawn.MinRank {
			rank += rng.Intn(spawn.MaxRank - spawn.MinRank + 1)
		}
		subRank := rng.Intn(5) + 1

		return models.Cell{X: x, Y: y, Type: t, Content: spawn.MonsterID, Rank: rank, SubRank: subRank}, nil
	}

	cell := GenerateCell(x, y, region)

	if cell.Type == "Monster" || cell.Type == "Boss" {
		seed := GetCellSeed(x, y, region)
		rng := rand.New(rand.NewSource(seed))
		
		monsterID, minRank, maxRank, err := db.PickDeterministicMonsterFromRegion(tx, region, seed)
		if err == nil && monsterID != "" {
			cell.Content = monsterID
			rank := minRank
			if maxRank > minRank {
				rank += rng.Intn(maxRank - minRank + 1)
			}
			cell.Rank = rank
			cell.SubRank = rng.Intn(5) + 1
		} else {
			cell.Rank = 1
			cell.SubRank = rng.Intn(5) + 1
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

	cleaned := cleanupExpiredClearedCells(&player, clearedCellCooldown)
	if cleaned {
		_, err = tx.Exec("UPDATE players SET cleared_cells = $1 WHERE id = $2", player.ClearedCells, player.ID)
		if err != nil {
			logger.S.Error("Failed to save cleaned up cleared_cells:", err)
		}
	}

	stats := CalculatePlayerStats(&player)
	visionRadius := int(stats["investigation"])
	if visionRadius < 1 {
		visionRadius = 1
	}

	// Cap the visual vision radius to 2 (corresponding to a 5x5 grid) to prevent UI breakage
	visualVisionRadius := visionRadius
	if visualVisionRadius > 2 {
		visualVisionRadius = 2
	}

	cells := []models.Cell{}
	for dx := -visualVisionRadius; dx <= visualVisionRadius; dx++ {
		for dy := -visualVisionRadius; dy <= visualVisionRadius; dy++ {
			cx := player.X + dx
			cy := player.Y + dy
			if cx >= 0 && cx < 100 && cy >= 0 && cy < 100 {
				cell, err := ResolveCell(tx, player.CurrentRegion, cx, cy)
				if err != nil {
					cell = GenerateCell(cx, cy, player.CurrentRegion)
				}
				if isCellClearedRecently(&player, cx, cy, clearedCellCooldown) {
					cell = models.Cell{X: cx, Y: cy, Type: "Empty", Content: ""}
				}
				cells = append(cells, cell)
			}
		}
	}

	type NearbyPlayer struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		X         int    `json:"x"`
		Y         int    `json:"y"`
	}
	nearbyPlayers := []NearbyPlayer{}
	rows, errQuery := tx.Query(`
		SELECT id, username, avatar_url, x, y 
		FROM players 
		WHERE current_region = $1 
		  AND id != $2 
		  AND (current_region IN ('qing_mao_mountain', 'gu_luan_slope', 'three_kings_inheritance') OR last_mana_update >= NOW() - INTERVAL '15 minutes')
		  AND x >= $3 AND x <= $4
		  AND y >= $5 AND y <= $6
	`, player.CurrentRegion, player.ID, player.X-visualVisionRadius, player.X+visualVisionRadius, player.Y-visualVisionRadius, player.Y+visualVisionRadius)
	if errQuery == nil {
		defer rows.Close()
		for rows.Next() {
			var np NearbyPlayer
			if errScan := rows.Scan(&np.ID, &np.Username, &np.AvatarURL, &np.X, &np.Y); errScan == nil {
				nearbyPlayers = append(nearbyPlayers, np)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		logger.S.Error("GetGrid commit warning:", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"x":              player.X,
		"y":              player.Y,
		"vision_radius":  visualVisionRadius,
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
	invItems, _ := InventorySvc.ListByPlayer(player.ID.String())
	seen := map[string]bool{}
	for _, inv := range invItems {
		if !seen[inv.ItemID] {
			seen[inv.ItemID] = true
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

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": errLeader.Error()})
		return
	}

	cleanupExpiredClearedCells(&player, clearedCellCooldown)

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

	hasFixedImmortalTravel := false
	for _, gu := range player.GuInventory {
		if gu.GuID == "GU_FIXED_IMMORTAL_TRAVEL" && gu.Equipped && gu.Status == "ACTIVE" {
			hasFixedImmortalTravel = true
			break
		}
	}

	if dist > movementRange && !hasFixedImmortalTravel {
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
	hasHeavenSecret := false
	for _, gu := range player.GuInventory {
		if gu.GuID == "GU_HEAVEN_SECRET" && gu.Equipped && gu.Status == "ACTIVE" {
			hasHeavenSecret = true
			break
		}
	}
	if hasHeavenSecret {
		visionRadius *= 2
	}
	addExploredCells(&player, visionRadius)

	// Check if cell is cleared recently (2 minutes cooldown)
	coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
	isCleared := isCellClearedRecently(&player, player.X, player.Y, clearedCellCooldown)

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
		SELECT id, username, rank, sub_rank, hp, max_mana, current_mana, base_hp, path_masteries, `+db.GuInventoryAggregate("players.id")+`, dao_marks
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
			EnemyAvatarURL:     targetPlayer.AvatarURL,
			EnemyType:          "PVP",
			CombatMode:         "real",
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
				var guSpec models.GuDictionaryItem
				var ok bool
				if cell.Rank > 0 {
					guSpec, ok = models.GetGu(cell.Content, cell.Rank)
				}
				if !ok {
					guSpec, _ = models.GetGuAny(cell.Content)
				}
				newGu := models.GuInstance{
					InstanceID: uuid.New().String(),
					GuID:       cell.Content,
					Rank:       guSpec.Rank,
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

			mon, err := db.LoadMonsterTemplate(tx, cell.Content)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải định nghĩa quái vật từ cơ sở dữ liệu"})
				return
			}

			rank := 1
			if cell.Rank > 0 {
				rank = cell.Rank
			}
			subRank := 1
			if cell.SubRank > 0 {
				subRank = cell.SubRank
			}

			combatState := models.BuildCombatStateFromMonster(mon, rank, subRank)
			player.ActiveCombat = &combatState
			logMsg = fmt.Sprintf("Nguy hiểm! Đạo hữu chạm trán %s tại tọa độ (%d, %d)!", mon.Name, player.X, player.Y)

			hasHeavenSecret := false
			for _, gu := range player.GuInventory {
				if gu.GuID == "GU_HEAVEN_SECRET" && gu.Equipped && gu.Status == "ACTIVE" {
					hasHeavenSecret = true
					break
				}
			}
			if hasHeavenSecret && mon.LootTableID != "" {
				lt, errLt := db.LoadLootTable(tx, mon.LootTableID)
				if errLt == nil {
					extraLootMsg := " [Thiên Cơ hiển lộ - Tỉ lệ rơi đồ: "
					first := true
					for _, entry := range lt.Entries {
						if !first {
							extraLootMsg += ", "
						}
						if entry.ItemID != "" {
							extraLootMsg += fmt.Sprintf("%s (%.1f%%)", entry.ItemID, entry.Chance*100)
						} else if entry.GuID != "" {
							extraLootMsg += fmt.Sprintf("%s (%.1f%%)", entry.GuID, entry.Chance*100)
						}
						first = false
					}
					extraLootMsg += "]"
					logMsg += extraLootMsg
				}
			}

		case "PVP":
			var targetPlayer models.Player
			err := tx.QueryRow(`
				SELECT id, username, rank, sub_rank, hp, max_mana, path_masteries, `+db.GuInventoryAggregate("players.id")+`, dao_marks
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
					EnemyType:      "PVP",
					CombatMode:     "real",
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
				adjX := player.X + ax
				adjY := player.Y + ay

				// check if already cleared
				if isCellClearedRecently(&player, adjX, adjY, clearedCellCooldown) {
					continue
				}

				adj, err := ResolveCell(tx, player.CurrentRegion, adjX, adjY)
				if err != nil {
					adj = GenerateCell(adjX, adjY, player.CurrentRegion)
				}

				if (adj.Type == "Monster" || adj.Type == "Boss") && rand.Intn(100) < 5 { // 5% chance
					combatTriggered = true

					mon, err := db.LoadMonsterTemplate(tx, adj.Content)
					if err == nil {
						rank := 1
						if adj.Rank > 0 {
							rank = adj.Rank
						}
						subRank := 1
						if adj.SubRank > 0 {
							subRank = adj.SubRank
						}
						combatState := models.BuildCombatStateFromMonster(mon, rank, subRank)
						player.ActiveCombat = &combatState
						logMsg = fmt.Sprintf("Phục kích! Quái thú %s lân cận (%d, %d) chủ động lao ra tấn công đạo hữu!", mon.Name, adjX, adjY)
					}
					// Mark the original monster cell as cleared
					if player.ClearedCells == nil {
						player.ClearedCells = make(models.ClearedCellsMap)
					}
					coordKeyAdj := fmt.Sprintf("%d,%d", adjX, adjY)
					player.ClearedCells[coordKeyAdj] = time.Now().Format(time.RFC3339)
				}
			}
		}
	}

	if player.ActiveCombat != nil {
		player.ActiveCombat = StartCombat(tx, player.ActiveCombat, &player)
	}

	// Fire ON_EXPLORE_MOVE on equipped Gus
	var exploreEventMsg string
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				if msg, err := HandleGuEvent(&player, spec, gu, "ON_EXPLORE_MOVE"); err == nil && msg != "" {
					exploreEventMsg = msg
				}
			}
		}
	}
	if exploreEventMsg != "" {
		logMsg += " " + exploreEventMsg
	}

	qiGained := AddHumanQi(&player, 1.0)

	// Update DB
	updateQuery := `
		UPDATE players
		SET current_mana = $1, x = $2, y = $3, explored_cells = $4, cleared_cells = $5, active_combat = $6, temp_loot = $7, last_mana_update = $8, primeval_stones = $9, hp = $10, max_mana = $11, human_qi = $12
		WHERE id = $13
	`
	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.X, player.Y, player.ExploredCells, player.ClearedCells, player.ActiveCombat, player.TempLoot, time.Now(), player.PrimevalStones, player.HP, player.MaxMana, player.HumanQi, player.ID)
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

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"log":              logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		"combat_triggered": combatTriggered,
		"stats":            newStats,
	})
}

// GetRegions fetches all active regions from the database
func GetRegions(c *gin.Context) {
	query := `
		SELECT id, name, description, width, height, min_rank, max_rank, is_active, created_at, updated_at
		FROM regions
		WHERE is_active = TRUE
		ORDER BY min_rank ASC
	`
	rows, err := db.DB.QueryContext(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải dữ liệu bản đồ"})
		return
	}
	defer rows.Close()

	var regions []models.Region
	for rows.Next() {
		var r models.Region
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Width, &r.Height, &r.MinRank, &r.MaxRank, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			continue
		}
		regions = append(regions, r)
	}

	c.JSON(http.StatusOK, regions)
}
