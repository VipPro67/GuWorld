package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const clearedCellCooldown = 15 * time.Minute

func GetCellSeed(x, y int, zoneKey string) int64 {
	// Deterministic per (zone, x, y): monster / resource content stays stable
	// on the map and matches what players actually encounter. Cell respawn is
	// handled separately via per-player cleared_cells cooldown, so a static
	// seed does not lock content forever for an individual player.
	h := sha256.New()
	h.Write([]byte(zoneKey))
	binary.Write(h, binary.BigEndian, int64(x))
	binary.Write(h, binary.BigEndian, int64(y))
	return int64(binary.BigEndian.Uint64(h.Sum(nil)[:8]))
}

// defaultResourcePool / defaultWildGuPool are fallback content pools used by
// zones that do not define pool_weights in Region.csv.
var defaultResourcePool = []string{"moon_orchid_petal", "green_bamboo_wine", "jade_stone", "boar_meat"}
var defaultWildGuPool = []string{"GU_LITTLE_LIGHT", "GU_JADE_SKIN", "GU_BRONZE_RELIC", "GU_FIRE_ARMOR"}

// pickWeightedType rolls a cell content type using zone weights. Weights come
// from a map, so iterate keys in sorted order to keep the pick deterministic
// (Go map iteration order is random and would shuffle cells every call).
func pickWeightedType(weights map[string]int, rng *rand.Rand) string {
	keys := make([]string, 0, len(weights))
	for k := range weights {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	total := 0
	for _, k := range keys {
		if w := weights[k]; w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return "Empty"
	}
	roll := rng.Intn(total)
	for _, k := range keys {
		w := weights[k]
		if w <= 0 {
			continue
		}
		roll -= w
		if roll < 0 {
			return k
		}
	}
	return "Empty"
}

// GenerateCell procedurally generates a low-tier cell (monster / resource /
// wild Gu / empty) from the containing zone's in-memory pools. No database
// access — content is deterministic per (zone, x, y) and stays stable so the
// map matches gameplay. Cleared cells are suppressed per player via the
// cleared_cells cooldown, which acts as the respawn mechanism.
func GenerateCell(x, y int) models.Cell {
	if !inWorld(x, y) {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: "Biên Giới Cổ Giới (World Edge)"}
	}

	zone := FindZoneAt(x, y)
	if zone == nil || zone.ZoneType == ZoneTypeHub {
		return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
	}
	zoneID := zone.ID
	seed := GetCellSeed(x, y, zoneID)
	rng := rand.New(rand.NewSource(seed))

	weights := zone.PoolWeights.CellWeights
	if len(weights) == 0 {
		weights = map[string]int{"Monster": 20, "Resource": 10, "WildGu": 3, "Empty": 67}
	}

	switch pickWeightedType(weights, rng) {
	case "Monster":
		monsterID := ""
		if spawn, ok := pickMonsterFromPool(zoneID, rng, false); ok {
			monsterID = spawn.MonsterID
		} else {
			monsterID = randomNormalMonsterID(rng)
		}
		rank := zone.MinRank
		if zone.MaxRank > zone.MinRank {
			rank += rng.Intn(zone.MaxRank - zone.MinRank + 1)
		}
		if rank < 1 {
			rank = 1
		}
		return models.Cell{X: x, Y: y, Type: "Monster", Content: monsterID, Rank: rank, SubRank: rng.Intn(5) + 1}
	case "Resource":
		pool := zone.PoolWeights.Resources
		if len(pool) == 0 {
			pool = defaultResourcePool
		}
		return models.Cell{X: x, Y: y, Type: "Resource", Content: pool[rng.Intn(len(pool))]}
	case "WildGu":
		pool := zone.PoolWeights.WildGus
		if len(pool) == 0 {
			pool = defaultWildGuPool
		}
		guID := pool[rng.Intn(len(pool))]
		rank := zone.MinRank
		if rank < 1 {
			rank = 1
		}
		return models.Cell{X: x, Y: y, Type: "WildGu", Content: guID, Rank: rank}
	default:
		return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
	}
}

// randomNormalMonsterID picks a Normal-type monster from the in-memory registry.
func randomNormalMonsterID(rng *rand.Rand) string {
	var ids []string
	for id, mon := range models.MonsterRegistry {
		if strings.EqualFold(mon.Type, "Normal") {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "monster_wild_boar"
	}
	// Sort so the pick is stable across calls (map iteration order is random).
	sort.Strings(ids)
	return ids[rng.Intn(len(ids))]
}

// ResolveCell resolves a world cell. Shared persistent cells (bosses, high
// resources, safe points) come from the in-memory map_cells index; everything
// else is generated procedurally. No database access in the hot path.
func ResolveCell(x, y int) models.Cell {
	if pc, found, cleared := PersistentCellAt(x, y); found {
		if cleared {
			return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
		}
		cell := pc.Cell
		cell.X = x
		cell.Y = y
		return cell
	}
	return GenerateCell(x, y)
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
			if cx >= 0 && cx < WorldWidth && cy >= 0 && cy < WorldHeight {
				cell := ResolveCell(cx, cy)
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
		WHERE id != $1
		  AND last_mana_update >= NOW() - INTERVAL '15 minutes'
		  AND x >= $2 AND x <= $3
		  AND y >= $4 AND y <= $5
	`, player.ID, player.X-visualVisionRadius, player.X+visualVisionRadius, player.Y-visualVisionRadius, player.Y+visualVisionRadius)
	if errQuery == nil {
		defer rows.Close()
		for rows.Next() {
			var np NearbyPlayer
			if errScan := rows.Scan(&np.ID, &np.Username, &np.AvatarURL, &np.X, &np.Y); errScan == nil {
				nearbyPlayers = append(nearbyPlayers, np)
			}
		}
	}

	// Always include party members regardless of distance for position sync
	if player.PartyID != nil {
		seenIDs := map[string]bool{}
		for _, np := range nearbyPlayers {
			seenIDs[np.ID] = true
		}
		partyRows, errParty := tx.Query(`
			SELECT id, username, avatar_url, x, y
			FROM players
			WHERE party_id = $1
			  AND id != $2
			  AND current_region = $3
		`, player.PartyID, player.ID, player.CurrentRegion)
		if errParty == nil {
			defer partyRows.Close()
			for partyRows.Next() {
				var np NearbyPlayer
				if errScan := partyRows.Scan(&np.ID, &np.Username, &np.AvatarURL, &np.X, &np.Y); errScan == nil {
					if !seenIDs[np.ID] {
						nearbyPlayers = append(nearbyPlayers, np)
						seenIDs[np.ID] = true
					}
				}
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
		"stats":          stats,
		"players_on_map": nearbyPlayers,
		"world_width":    WorldWidth,
		"world_height":   WorldHeight,
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

// resolveCellArrival places the player at (x, y), derives the region / subzone
// and resolves the cell content the player stepped onto (resource pickup, wild
// Gu capture, monster / boss combat or player-initiated PVP), including the
// small adjacency ambush chance (skipped when allowAmbush is false — used by
// free path movement so the player only fights when stepping onto an actual
// monster cell, not randomly on empty cells). It mutates player in-place and
// returns the log message plus whether a fight was started. attackTargetID is
// only used to explicitly attack another player standing on the destination.
func resolveCellArrival(tx *sql.Tx, player *models.Player, x, y int, attackTargetID string, allowAmbush bool) (string, bool, error) {
	player.X = x
	player.Y = y
	player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
	recordVisitedSubZone(player, SubZoneKey(player.X, player.Y))

	coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
	isCleared := isCellClearedRecently(player, player.X, player.Y, clearedCellCooldown)
	cell := ResolveCell(player.X, player.Y)
	logMsg := fmt.Sprintf("Di chuyển thành công đến (%d, %d).", player.X, player.Y)
	combatTriggered := false
	pvpTriggered := false

	// Player-initiated PVP: another active player stands on the destination.
	if attackTargetID != "" {
		var targetPlayer models.Player
		queryStr := `
			SELECT p.id, p.username, p.rank, p.sub_rank, p.hp, p.max_mana, p.current_mana, p.base_hp, p.path_masteries, ` + db.GuInventoryAggregate("p.id") + `, COALESCE(ia.dao_marks, '{}'::jsonb)
			FROM players p
			LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
			WHERE p.x = $1 AND p.y = $2 AND p.id != $3
			  AND p.last_mana_update >= NOW() - INTERVAL '15 minutes'
		`
		var errTarget error
		if player.PartyID != nil {
			queryStr += ` AND (p.party_id IS NULL OR p.party_id != $4)`
			errTarget = tx.QueryRow(queryStr, player.X, player.Y, player.ID, player.PartyID).Scan(
				&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
				&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.CurrentMana, &targetPlayer.BaseHP,
				&targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks,
			)
		} else {
			errTarget = tx.QueryRow(queryStr, player.X, player.Y, player.ID).Scan(
				&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
				&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.CurrentMana, &targetPlayer.BaseHP,
				&targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks,
			)
		}
		if errTarget == nil && attackTargetID == targetPlayer.ID.String() {
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
			enemyMaxHP := gamefx.EffectiveMaxHP(&targetPlayer)
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
				EnemySpeed:         targetStats["speed"],
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
	}

	if isCleared {
		logMsg = fmt.Sprintf("Di chuyển thành công đến (%d, %d). Vùng đất trống trải hoang tàn.", player.X, player.Y)
	} else if !pvpTriggered {
		switch cell.Type {
		case "Resource":
			if HasExplorationInventorySpace(player, 1) {
				if player.TempLoot.MaterialsInventory == nil {
					player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
				}
				player.TempLoot.MaterialsInventory[cell.Content]++
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Đạo hữu tìm thấy nguyên liệu hoang dã: 1x %s (được giữ tạm thời trong bản đồ)!", player.X, player.Y, cell.Content)

				MarkPersistentCellCleared(tx, player.ID, player.X, player.Y)

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Phát hiện 1x %s, nhưng Bách Bảo Nang đã đầy!", player.X, player.Y, cell.Content)
			}
		case "WildGu":
			if HasExplorationInventorySpace(player, 1) {
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

			mon, ok := models.MonsterRegistry[cell.Content]
			if !ok {
				// Fall back to the DB (registry may not have loaded this id).
				var err error
				mon, err = db.LoadMonsterTemplate(tx, cell.Content)
				if err != nil {
					return "", false, err
				}
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
					chanceMul := lootChanceMultiplier(rank)
					qtyMul := lootQuantityMultiplier(rank)
					extraLootMsg := fmt.Sprintf(" [Thiên Cơ hiển lộ (Rank %d x%.1f) - Tỉ lệ rơi đồ: ", rank, qtyMul)
					first := true
					for _, entry := range lt.Entries {
						if !first {
							extraLootMsg += ", "
						}
						scaledChance := math.Min(entry.Chance*chanceMul, 1.0)
						scaledMin := int(math.Round(float64(entry.MinQty) * qtyMul))
						scaledMax := int(math.Round(float64(entry.MaxQty) * qtyMul))
						if itemID := entry.ItemID; itemID != "" {
							extraLootMsg += fmt.Sprintf("%s x%d-%d (%.1f%%)", itemID, scaledMin, scaledMax, scaledChance*100)
						} else if entry.GuID != "" {
							extraLootMsg += fmt.Sprintf("%s (%.1f%%)", entry.GuID, scaledChance*100)
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
				SELECT p.id, p.username, p.rank, p.sub_rank, p.hp, p.max_mana, p.path_masteries, `+db.GuInventoryAggregate("p.id")+`, COALESCE(ia.dao_marks, '{}'::jsonb)
				FROM players p
				LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
				WHERE p.id != $1 
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

	if !combatTriggered && allowAmbush {
		for ax := -1; ax <= 1 && !combatTriggered; ax++ {
			for ay := -1; ay <= 1 && !combatTriggered; ay++ {
				if ax == 0 && ay == 0 {
					continue
				}
				adjX := player.X + ax
				adjY := player.Y + ay

				// check if already cleared
				if isCellClearedRecently(player, adjX, adjY, clearedCellCooldown) {
					continue
				}

				adj := ResolveCell(adjX, adjY)

				if (adj.Type == "Monster" || adj.Type == "Boss") && rand.Intn(100) < 5 { // 5% chance
					combatTriggered = true

					mon, ok := models.MonsterRegistry[adj.Content]
					if !ok {
						mon, _ = db.LoadMonsterTemplate(tx, adj.Content)
					}
					if mon.ID != "" {
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
		player.ActiveCombat = StartCombat(tx, player.ActiveCombat, player)
	}

	// Fire ON_EXPLORE_MOVE on equipped Gus
	var exploreEventMsg string
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				if msg, err := HandleGuEvent(player, spec, gu, "ON_EXPLORE_MOVE"); err == nil && msg != "" {
					exploreEventMsg = msg
				}
			}
		}
	}
	if exploreEventMsg != "" {
		logMsg += " " + exploreEventMsg
	}

	return logMsg, combatTriggered, nil
}

// MoveRequest specifies target position
type MoveRequest struct {
	X              int    `json:"x"`
	Y              int    `json:"y"`
	AttackTargetID string `json:"attack_target_id,omitempty"`
}

// MovePlayer moves the player on the shared world grid and triggers cell actions
func MovePlayer(c *gin.Context) {
	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.X < 0 || req.X >= WorldWidth || req.Y < 0 || req.Y >= WorldHeight {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
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
	logMsg, combatTriggered, errArrival := resolveCellArrival(tx, &player, req.X, req.Y, req.AttackTargetID, true)
	if errArrival != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải định nghĩa quái vật từ cơ sở dữ liệu"})
		return
	}

	qiGained := AddHumanQi(&player, 1.0)

	// Update DB
	updateQuery := `
		UPDATE players
		SET current_mana = $1, x = $2, y = $3, current_region = $4, cleared_cells = $5, active_combat = $6, temp_loot = $7, last_mana_update = $8, primeval_stones = $9, hp = $10, max_mana = $11, human_qi = $12, visited_subzones = $13
		WHERE id = $14
	`
	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.X, player.Y, player.CurrentRegion, player.ClearedCells, player.ActiveCombat, player.TempLoot, time.Now(), player.PrimevalStones, player.HP, player.MaxMana, player.HumanQi, player.VisitedSubZones, player.ID)
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

// lineCells returns the cells on a straight line from (x0,y0) to (x1,y1)
// using Bresenham's algorithm, including the target cell and excluding the
// starting cell.
func lineCells(x0, y0, x1, y1 int) [][2]int {
	dx := x1 - x0
	dy := y1 - y0
	sx := 1
	if dx < 0 {
		sx = -1
	}
	sy := 1
	if dy < 0 {
		sy = -1
	}
	adx := dx
	if adx < 0 {
		adx = -adx
	}
	ady := dy
	if ady < 0 {
		ady = -ady
	}
	err := adx - ady
	x, y := x0, y0
	var cells [][2]int
	for {
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -ady {
			err -= ady
			x += sx
		}
		if e2 < adx {
			err += adx
			y += sy
		}
		cells = append(cells, [2]int{x, y})
	}
	return cells
}

// MovePlayerPath moves the player along a straight path from its current cell
// to the requested cell, triggering the normal cell action on every traversed
// cell. Movement is still bounded by the player's movement stat (base +
// passive movement). The walk stops early if a fight is triggered on the way.
func MovePlayerPath(c *gin.Context) {
	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.X < 0 || req.X >= WorldWidth || req.Y < 0 || req.Y >= WorldHeight {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
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

	if dist == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đang ở vị trí này rồi."})
		return
	}

	// Free movement: the player may click anywhere in the world. The path is
	// walked cell by cell (each traversed cell triggers its action), stopping
	// early on the first combat. maxPathCells bounds the work of a single
	// request so the client chains requests for very long journeys.
	const maxPathCells = 1000
	path := lineCells(player.X, player.Y, req.X, req.Y)
	if len(path) > maxPathCells {
		path = path[:maxPathCells]
	}
	reached := len(path) < maxPathCells
	var logs []string
	combatTriggered := false
	for _, step := range path {
		logMsg, triggered, errArrival := resolveCellArrival(tx, &player, step[0], step[1], "", false)
		if errArrival != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải định nghĩa quái vật từ cơ sở dữ liệu"})
			return
		}
		logs = append(logs, logMsg)
		if triggered {
			combatTriggered = true
			reached = false
			break
		}
	}

	qiGained := AddHumanQi(&player, 1.0)

	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}
	_, err = tx.Exec(`
		UPDATE players
		SET current_mana = $1, x = $2, y = $3, current_region = $4, cleared_cells = $5, active_combat = $6, temp_loot = $7, last_mana_update = $8, primeval_stones = $9, hp = $10, max_mana = $11, human_qi = $12, visited_subzones = $13
		WHERE id = $14
	`, player.CurrentMana, player.X, player.Y, player.CurrentRegion, player.ClearedCells, player.ActiveCombat, player.TempLoot, time.Now(), player.PrimevalStones, player.HP, player.MaxMana, player.HumanQi, player.VisitedSubZones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	newStats := CalculatePlayerStats(&player)
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"log":              strings.Join(logs, "\n") + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		"logs":             logs,
		"combat_triggered": combatTriggered,
		"reached":          reached,
		"stats":            newStats,
	})
}

// WorldObjectView is a lightweight view of a persistent shared cell for the
// large world map (villages / tram nghi, bosses, high-tier resources).
type WorldObjectView struct {
	RegionID       string `json:"region_id"`
	X              int    `json:"x"`
	Y              int    `json:"y"`
	CellType       string `json:"cell_type"`
	ContentID      string `json:"content_id"`
	Rank           int    `json:"rank,omitempty"`
	RespawnSeconds int    `json:"respawn_seconds"`
	Cleared        bool   `json:"cleared"`
}

// GetExploreObjects returns the persistent map cells (map_cells) so the large
// world map can render villages, bosses and high-tier resource points.
func GetExploreObjects(c *gin.Context) {
	ensureZonesLoaded()

	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()

	out := make([]WorldObjectView, 0, len(zoneReg.persistent))
	for key, pc := range zoneReg.persistent {
		out = append(out, WorldObjectView{
			RegionID:       pc.RegionID,
			X:              pc.X,
			Y:              pc.Y,
			CellType:       pc.Type,
			ContentID:      pc.Content,
			Rank:           pc.Rank,
			RespawnSeconds: pc.RespawnSeconds,
			Cleared:        pc.cleared,
		})
		_ = key
	}
	c.JSON(http.StatusOK, out)
}

// NearbyCell is a resolved non-empty cell (monster / boss / resource / wild Gu)
// near the player, for the large world map markers.
type NearbyCell struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Type    string `json:"cell_type"`
	Content string `json:"content"`
	Rank    int    `json:"rank,omitempty"`
	SubRank int    `json:"sub_rank,omitempty"`
}

// GetNearbyCells resolves the cells around the player (or an explicit
// ?x=&y=&r= box) and returns the non-empty ones so the big map can draw
// monster / boss / resource markers. Content is deterministic per zone cell,
// matching the grid gameplay (ResolveCell).
func GetNearbyCells(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var x, y, r int
	fmt.Sscanf(c.Query("x"), "%d", &x)
	fmt.Sscanf(c.Query("y"), "%d", &y)
	fmt.Sscanf(c.Query("r"), "%d", &r)

	if r <= 0 {
		r = 300
	}
	if r > 300 {
		r = 300
	}

	if x == 0 && y == 0 {
		player, err := loadPlayerForStatus(db.DB, authID, false)
		if err == nil {
			x, y = player.X, player.Y
		}
	}

	// Resolve the full box of cells around the center and keep the non-empty
	// ones. Box is at most (2r+1)^2 cells; with r capped at 300 this stays a
	// bounded, symmetric area around the player so the canvas markers align.
	out := []NearbyCell{}
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			cx := x + dx
			cy := y + dy
			if cx < 0 || cx >= WorldWidth || cy < 0 || cy >= WorldHeight {
				continue
			}
			cell := ResolveCell(cx, cy)
			if cell.Type == "Empty" {
				continue
			}
			out = append(out, NearbyCell{X: cell.X, Y: cell.Y, Type: cell.Type, Content: cell.Content, Rank: cell.Rank, SubRank: cell.SubRank})
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetRegions fetches all active regions (zones) from the database
func GetRegions(c *gin.Context) {
	query := `
		SELECT id, name, description, width, height, min_rank, max_rank, is_active, created_at, updated_at,
		       COALESCE(parent_id,''), COALESCE(zone_type,'NGU_VUC'), COALESCE(min_x,0), COALESCE(min_y,0),
		       COALESCE(max_x,0), COALESCE(max_y,0), COALESCE(entry_x,0), COALESCE(entry_y,0)
		FROM regions
		WHERE is_active = TRUE
		ORDER BY min_rank ASC, name ASC
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
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Width, &r.Height, &r.MinRank, &r.MaxRank, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
			&r.ParentID, &r.ZoneType, &r.MinX, &r.MinY, &r.MaxX, &r.MaxY, &r.EntryX, &r.EntryY); err != nil {
			continue
		}
		regions = append(regions, r)
	}

	c.JSON(http.StatusOK, regions)
}

// GetWorld returns the shared world geometry for the frontend overview.
func GetWorld(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"width":           WorldWidth,
		"height":          WorldHeight,
		"large_zone_size": LargeZoneSize,
		"sub_zone_size":   SubZoneSize,
		"hub_region_id":   HubRegionID,
		"hub_x":           HubEntryPoint.X,
		"hub_y":           HubEntryPoint.Y,
	})
}
