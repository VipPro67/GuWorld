package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const clearedCellCooldown = 15 * time.Minute

func GetCellSeed(x, y int, zoneKey string) int64 {
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
// map matches gameplay. Cleared cells are suppressed via Redis shared map
// state with respawn TTL, allowing synchronized clearing across all players.
func GenerateCell(x, y int) models.Cell {
	if cell, ok := cellGenCache.get(x, y); ok {
		return cell
	}
	cell := generateCellUncached(x, y)
	cellGenCache.put(x, y, cell)
	return cell
}

func generateCellUncached(x, y int) models.Cell {
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
		weights = map[string]int{"Monster": 8, "Resource": 4, "WildGu": 1, "Empty": 87}
	} else {
		// Override to forcibly reduce density if DB has high values
		newW := make(map[string]int)
		for k, v := range weights {
			if k == "Empty" {
				newW[k] = v * 4
			} else {
				newW[k] = v / 2
			}
		}
		if newW["Empty"] == 0 {
			newW["Empty"] = 100
		}
		weights = newW
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
		type weightedResource struct {
			id     string
			weight float64
		}
		var pool []weightedResource
		var total float64
		for _, id := range zone.PoolWeights.Resources {
			w := 1.0
			if spec, ok := models.ItemRegistry[id]; ok {
				w = guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
			}
			if w <= 0 {
				w = 1.0
			}
			pool = append(pool, weightedResource{id: id, weight: w})
			total += w
		}
		if len(pool) == 0 {
			for _, id := range defaultResourcePool {
				w := 1.0
				if spec, ok := models.ItemRegistry[id]; ok {
					w = guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
				}
				pool = append(pool, weightedResource{id: id, weight: w})
				total += w
			}
		}
		itemID := ""
		r := rng.Float64() * total
		for _, e := range pool {
			r -= e.weight
			if r <= 0 {
				itemID = e.id
				break
			}
		}
		if itemID == "" {
			itemID = pool[0].id
		}
		return models.Cell{X: x, Y: y, Type: "Resource", Content: itemID}
	case "WildGu":
		type weightedGu struct {
			id     string
			weight float64
		}
		var pool []weightedGu
		var total float64
		for _, id := range zone.PoolWeights.WildGus {
			if models.HasImmortalRank(id) {
				continue
			}
			w := 1.0
			if spec, ok := models.GetGuAny(id); ok {
				w = guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
			}
			if w <= 0 {
				w = 1.0
			}
			pool = append(pool, weightedGu{id: id, weight: w})
			total += w
		}
		if len(pool) == 0 {
			for _, id := range defaultWildGuPool {
				pool = append(pool, weightedGu{id: id, weight: 1.0})
				total += 1.0
			}
		}
		guID := ""
		r := rng.Float64() * total
		for _, e := range pool {
			r -= e.weight
			if r <= 0 {
				guID = e.id
				break
			}
		}
		if guID == "" {
			guID = pool[0].id
		}
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

func isCellClearedRecently(regionID string, x, y int, cooldown time.Duration) bool {
	cleared, _, err := cache.IsCellCleared(regionID, x, y)
	if err != nil {
		logger.S.Errorf("Failed to check Redis cell state for (%d,%d) in region %s: %v", x, y, regionID, err)
		return false
	}
	return cleared
}

// cleanupExpiredClearedCells removes cells from player.ClearedCells that have exceeded the cooldown duration.
// Returns true if any cells were removed. Kept for backward compatibility with legacy players.
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
				if isCellClearedRecently(player.CurrentRegion, cx, cy, clearedCellCooldown) {
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
	seenIDs := map[string]bool{}

	// Radius-based nearby players come from the in-memory position index
	// (fast path). SQL fallback only while the index is cold.
	entries := worldPosIndex.Nearby(player.X, player.Y, visualVisionRadius, player.ID.String())
	if worldPosIndex.Len() == 0 {
		entries = queryNearbyPlayersSQL(c, player.ID, player.X, player.Y, visualVisionRadius)
	}
	for _, e := range entries {
		seenIDs[e.ID] = true
		nearbyPlayers = append(nearbyPlayers, NearbyPlayer{ID: e.ID, Username: e.Username, AvatarURL: e.AvatarURL, X: e.X, Y: e.Y})
	}

	// Always include party members regardless of distance for position sync
	if player.PartyID != nil {
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

	// Ground drops query within visible radius
	type GroundDropItem struct {
		ID             string         `json:"id"`
		RegionID       string         `json:"region_id"`
		X              int            `json:"x"`
		Y              int            `json:"y"`
		LoserID        string         `json:"loser_id"`
		LoserName      string         `json:"loser_name"`
		KillerID       string         `json:"killer_id,omitempty"`
		KillerName     string         `json:"killer_name,omitempty"`
		LootData       GroundLootData `json:"loot_data"`
		IsProtected    bool           `json:"is_protected"`
		ProtectedUntil *time.Time     `json:"protected_until,omitempty"`
		ExpiresAt      time.Time      `json:"expires_at"`
		CreatedAt      time.Time      `json:"created_at"`
	}

	groundDrops := []GroundDropItem{}
	minX := player.X - visualVisionRadius
	maxX := player.X + visualVisionRadius
	minY := player.Y - visualVisionRadius
	maxY := player.Y + visualVisionRadius

	dropRows, errDrops := tx.Query(`
		SELECT id, region_id, x, y, loser_id, loser_name, killer_id, killer_name,
		       loot_data, protected_until, expires_at, created_at
		FROM map_ground_drops
		WHERE region_id = $1
		  AND x BETWEEN $2 AND $3
		  AND y BETWEEN $4 AND $5
		  AND expires_at > NOW()
		ORDER BY created_at DESC
	`, player.CurrentRegion, minX, maxX, minY, maxY)
	if errDrops == nil {
		defer dropRows.Close()
		now := time.Now()
		for dropRows.Next() {
			var d GroundDropItem
			var killerID, killerName sql.NullString
			var lootBytes []byte
			var protectedUntil sql.NullTime

			if errScan := dropRows.Scan(
				&d.ID, &d.RegionID, &d.X, &d.Y, &d.LoserID, &d.LoserName,
				&killerID, &killerName, &lootBytes, &protectedUntil, &d.ExpiresAt, &d.CreatedAt,
			); errScan == nil {
				if killerID.Valid {
					d.KillerID = killerID.String
				}
				if killerName.Valid {
					d.KillerName = killerName.String
				}
				if protectedUntil.Valid {
					d.ProtectedUntil = &protectedUntil.Time
				}
				json.Unmarshal(lootBytes, &d.LootData)

				if protectedUntil.Valid && protectedUntil.Time.After(now) {
					if killerID.Valid && killerID.String != player.ID.String() {
						d.IsProtected = true
					}
				}

				groundDrops = append(groundDrops, d)
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
		"ground_drops":   groundDrops,
		"world_width":    WorldWidth,
		"world_height":   WorldHeight,
	})
}

// HasExplorationInventorySpace checks bag capacity of the real inventory.
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
	isCleared := isCellClearedRecently(player.CurrentRegion, player.X, player.Y, clearedCellCooldown)
	cell := ResolveCell(player.X, player.Y)
	logMsg := fmt.Sprintf("Di chuyển thành công đến (%d, %d).", player.X, player.Y)
	combatTriggered := false
	pvpTriggered := false

	// Player-initiated PVP: attack target player within attack range (default <= 2 tiles, plus stats bonus)
	if attackTargetID != "" {
		targetUUID, errUUID := uuid.Parse(attackTargetID)
		if errUUID == nil {
			var targetPlayer models.Player
			queryStr := `
				SELECT p.id, p.username, p.rank, p.sub_rank, p.hp, p.max_mana, p.current_mana, p.base_hp, p.x, p.y, p.avatar_url, p.path_masteries, ` + db.GuInventoryAggregate("p.id") + `, COALESCE(ia.dao_marks, '{}'::jsonb), p.combat_settings
				FROM players p
				LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
				WHERE p.id = $1 AND p.id != $2
				  AND p.last_mana_update >= NOW() - INTERVAL '15 minutes'
			`
			var errTarget error
			if player.PartyID != nil {
				queryStr += ` AND (p.party_id IS NULL OR p.party_id != $3)`
				errTarget = tx.QueryRow(queryStr, targetUUID, player.ID, player.PartyID).Scan(
					&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
					&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.CurrentMana, &targetPlayer.BaseHP,
					&targetPlayer.X, &targetPlayer.Y, &targetPlayer.AvatarURL,
					&targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks, &targetPlayer.CombatSettings,
				)
			} else {
				errTarget = tx.QueryRow(queryStr, targetUUID, player.ID).Scan(
					&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
					&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.CurrentMana, &targetPlayer.BaseHP,
					&targetPlayer.X, &targetPlayer.Y, &targetPlayer.AvatarURL,
					&targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks, &targetPlayer.CombatSettings,
				)
			}

			if errTarget == nil {
				// Distance check: Chebyshev distance max(|x1-x2|, |y1-y2|)
				dx := math.Abs(float64(targetPlayer.X - player.X))
				dy := math.Abs(float64(targetPlayer.Y - player.Y))
				dist := int(math.Max(dx, dy))

				// Range requirement: base 2 tiles + bonus stats (attack_range / combat_range)
				stats := CalculatePlayerStats(player)
				bonusRange := int(stats["attack_range"]) + int(stats["combat_range"])
				maxPvPRange := 2 + bonusRange

				// Safe zone check
				z1 := FindZoneAt(player.X, player.Y)
				z2 := FindZoneAt(targetPlayer.X, targetPlayer.Y)
				isSafe1 := z1 != nil && (z1.ZoneType == "HUB" || z1.ZoneType == "SAFE_ZONE")
				isSafe2 := z2 != nil && (z2.ZoneType == "HUB" || z2.ZoneType == "SAFE_ZONE")

				if isSafe1 || isSafe2 {
					return "Không thể tấn công trong Vùng An Toàn!", false, nil
				}

				if dist <= maxPvPRange {
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
						EnemyTemplateID:     targetPlayer.ID.String(),
						EnemyName:           targetPlayer.Username,
						EnemyAvatarURL:      targetPlayer.AvatarURL,
						EnemyType:           "PVP",
						CombatMode:          "real",
						EnemyHP:             enemyHP,
						EnemyMaxHP:          enemyMaxHP,
						EnemyMana:           targetPlayer.CurrentMana,
						EnemyMaxMana:        targetPlayer.MaxMana,
						EnemyDamage:         targetStats["attack"],
						EnemyDefense:        targetStats["defense"],
						EnemySpeed:          targetStats["speed"],
						EnemyRank:           targetPlayer.Rank,
						EnemySubRank:        targetPlayer.SubRank,
						EnemyGuInventory:    filterGuLoadout(targetPlayer.GuInventory, targetPlayer.CombatSettings),
						EnemyDaoMarks:       targetPlayer.DaoMarks,
						EnemyPathMasteries:  targetPlayer.PathMasteries,
						EnemyCombatSettings: &targetPlayer.CombatSettings,
						TargetPlayerID:      targetPlayer.ID.String(),
					}
					player.ActiveCombat = &pvpCombat
					logMsg = fmt.Sprintf("Khai chiến! Đạo hữu tấn công Cổ Sư %s tại khoảng cách %d ô!", targetPlayer.Username, dist)
				} else {
					return fmt.Sprintf("Khoảng cách quá xa (%d ô)! Cần tiếp cận đối phương trong vòng %d ô để khai chiến.", dist, maxPvPRange), false, nil
				}
			}
		}
	}

	if isCleared {
		logMsg = fmt.Sprintf("Di chuyển thành công đến (%d, %d). Vùng đất trống trải hoang tàn.", player.X, player.Y)
	} else if !pvpTriggered {
		switch cell.Type {
		case "Resource":
			itemName := cell.Content
			if item, ok := models.ItemRegistry[cell.Content]; ok {
				itemName = item.Name
			}
			if HasExplorationInventorySpace(player, 1) {
				if err := InventorySvc.AddItem(tx, player.ID.String(), cell.Content, "materials", 1); err != nil {
					logger.S.Errorf("Failed to add resource %s to player %s: %v", cell.Content, player.ID, err)
					// Propagate: a failed write poisons the tx (25P02 on next statement).
					return "", false, err
				}
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Đạo hữu tìm thấy nguyên liệu hoang dã: 1x %s!", player.X, player.Y, itemName)

				if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearedCellCooldown); err != nil {
					logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
					return "", false, err
				}
				broadcastWorldCell(cellEventCleared, WorldCellEvent{X: player.X, Y: player.Y, CellType: cell.Type, ContentID: cell.Content})

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Phát hiện 1x %s, nhưng Bách Bảo Nang đã đầy!", player.X, player.Y, itemName)
			}
		case "WildGu":
			var guSpec models.GuDictionaryItem
			var ok bool
			if cell.Rank > 0 {
				guSpec, ok = models.GetGu(cell.Content, cell.Rank)
			}
			if !ok {
				guSpec, _ = models.GetGuAny(cell.Content)
			}
			guName := cell.Content
			if guSpec.Name != "" {
				guName = guSpec.Name
			}
			if HasExplorationInventorySpace(player, 1) {
				newGu := models.GuInstance{
					InstanceID: uuid.New().String(),
					GuID:       cell.Content,
					Rank:       guSpec.Rank,
					Hunger:     100,
					Status:     "ACTIVE",
					Equipped:   false,
				}
				player.GuInventory = append(player.GuInventory, newGu)
				logMsg = fmt.Sprintf("Cơ duyên! Đạo hữu bắt gặp và thu phục thành công Cổ trùng hoang dã: %s!", guName)

				if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearedCellCooldown); err != nil {
					logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
					return "", false, err
				}
				cache.BroadcastWildGuCaptured(player.X, player.Y, cell.Content)

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Đạo hữu phát hiện thấy Cổ trùng hoang dã %s, tiếc là Bách Bảo Nang đã đầy nên để nó trốn thoát!", guName)
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

			_, _, hasHeavenSecret := models.FindEquippedGuByID(player.GuInventory, "GU_HEAVEN_SECRET")
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
				SELECT p.id, p.username, p.rank, p.sub_rank, p.hp, p.max_mana, p.path_masteries, `+db.GuInventoryAggregateEquipped("p.id")+`, COALESCE(ia.dao_marks, '{}'::jsonb), p.combat_settings
				FROM players p
				LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
				WHERE p.id != $1 
				LIMIT 1`, player.ID,
			).Scan(
				&targetPlayer.ID, &targetPlayer.Username, &targetPlayer.Rank, &targetPlayer.SubRank,
				&targetPlayer.HP, &targetPlayer.MaxMana, &targetPlayer.PathMasteries, &targetPlayer.GuInventory, &targetPlayer.DaoMarks, &targetPlayer.CombatSettings,
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
					EnemyName:           targetPlayer.Username,
					EnemyType:           "PVP",
					CombatMode:          "real",
					EnemyHP:             100.0,
					EnemyMaxHP:          100.0,
					EnemyDamage:         targetStats["attack"],
					EnemyDefense:        targetStats["defense"],
					EnemySpeed:          targetStats["speed"],
					EnemyRank:           targetPlayer.Rank,
					EnemySubRank:        targetPlayer.SubRank,
					EnemyGuInventory:    filterGuLoadout(targetPlayer.GuInventory, targetPlayer.CombatSettings),
					EnemyDaoMarks:       targetPlayer.DaoMarks,
					EnemyPathMasteries:  targetPlayer.PathMasteries,
					EnemyCombatSettings: &targetPlayer.CombatSettings,
					TargetPlayerID:      targetPlayer.ID.String(),
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
				if isCellClearedRecently(player.CurrentRegion, adjX, adjY, clearedCellCooldown) {
					continue
				}

				adj := ResolveCell(adjX, adjY)

				rank := 1
				if adj.Rank > 0 {
					rank = adj.Rank
				}
				ambushChance := 50 + (rank * 2) // Base 50% + 2% per rank (min 50%)

				if (adj.Type == "Monster" || adj.Type == "Boss") && rand.Intn(100) < ambushChance {
					combatTriggered = true

					mon, ok := models.MonsterRegistry[adj.Content]
					if !ok {
						mon, _ = db.LoadMonsterTemplate(tx, adj.Content)
					}
					if mon.ID != "" {
						subRank := 1
						if adj.SubRank > 0 {
							subRank = adj.SubRank
						}
						combatState := models.BuildCombatStateFromMonster(mon, rank, subRank)
						player.ActiveCombat = &combatState
						logMsg = fmt.Sprintf("Phục kích! Quái thú %s lân cận (%d, %d) chủ động lao ra tấn công đạo hữu!", mon.Name, adjX, adjY)
					}
					// Mark the original monster cell as cleared
					if err := cache.SetCellCleared(player.CurrentRegion, adjX, adjY, player.ID.String(), clearedCellCooldown); err != nil {
						logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, adjX, adjY, err)
					}
					cache.BroadcastMonsterKilled(adjX, adjY, mon.Name, player.CurrentRegion)
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

// hpTooLowForExplore reports whether a player with HP ≤ 20% max HP is trying
// to move into the wild. Movement is still allowed towards the safe hub so a
// wounded player can always regroup and heal.
func hpTooLowForExplore(player *models.Player, destX, destY int) bool {
	if player == nil || player.HP <= 0 {
		return true
	}
	maxHP := gamefx.EffectiveMaxHP(player)
	if maxHP <= 0 {
		maxHP = 1.0
	}
	if player.HP > 0.20*maxHP {
		return false
	}
	return DeriveRegionAt(destX, destY) != HubRegionID
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
		logger.S.Errorf("MovePlayer: failed to evaluate mana for player=%s: %v", authID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load player Aperture: %v", err)})
		return
	}
	if player.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể di chuyển!"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
		return
	}

	cleanupExpiredClearedCells(&player, clearedCellCooldown)

	if player.HP <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong trạng thái trọng thương, không thể tiếp tục di chuyển!"})
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

	// Auto-resolve combat if skip_battle is enabled
	if combatTriggered && player.CombatSettings.SkipBattle && player.ActiveCombat != nil {
		winner, _, skipSummary, err := AutoResolveCombat(tx, &player)
		if err != nil {
			logger.S.Errorf("Auto-resolve combat failed during move: %v", err)
		} else {
			combatTriggered = false // Combat was auto-resolved
			enemyName := "kẻ thù"
			if e, ok := skipSummary["enemy_name"].(string); ok {
				enemyName = e
			}
			if winner == "Phe Ta" {
				logMsg = fmt.Sprintf("Tự động chiến thắng %s tại (%d, %d)! Nhân Khí +3.", enemyName, player.X, player.Y)
			} else {
				logMsg = fmt.Sprintf("Tự động chiến đấu với %s tại (%d, %d) thất bại! Đạo hữu bị truyền tống về an toàn.", enemyName, player.X, player.Y)
			}
			if skipSummary != nil {
				if loot, ok := skipSummary["loot"].([]string); ok && len(loot) > 0 {
					logMsg += " " + strings.Join(loot, " ")
				}
				if mineOccupied, ok := skipSummary["mine_occupied"].(int); ok && mineOccupied > 0 {
					logMsg += fmt.Sprintf(" Đã chiếm mỏ %d!", mineOccupied)
				}
			}
			if err := tx.Commit(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
				return
			}
			ClearPendingPlayerQueues(player.ID.String())
			PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
			// Include skip summary in response
			respondActionWithInventory(c, http.StatusOK, player, gin.H{
				"log":              logMsg,
				"combat_triggered": false,
				"skip_result":      skipSummary,
				"stats":            CalculatePlayerStats(&player),
			})
			return
		}
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		logger.S.Errorf("MovePlayer: failed to sync Gu inventory for player=%s: %v", player.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}
	var combatJSON any
	if player.ActiveCombat != nil {
		b, err := json.Marshal(player.ActiveCombat)
		if err != nil {
			logger.S.Errorf("MovePlayer: failed to marshal active combat for player=%s: %v", player.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
			return
		}
		combatJSON = string(b)
	}
	if err := savePlayerMovement(tx, &player, combatJSON); err != nil {
		logger.S.Errorf("MovePlayer: failed to update player=%s at (%d,%d), region=%s, combat=%t: %v", player.ID, player.X, player.Y, player.CurrentRegion, player.ActiveCombat != nil, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}

	SyncPartyMembersLocation(tx, &player)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	broadcastWorldPos(WorldPosEvent{
		ID:        player.ID.String(),
		Username:  player.Username,
		AvatarURL: player.AvatarURL,
		X:         player.X,
		Y:         player.Y,
		TS:        time.Now().UnixMilli(),
	})

	PushCellsToPlayer(player.ID, player.X, player.Y, 300)

	if combatTriggered {
		PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	} else {
		PublishPlayerSnapshot(&player, PlayerEventUpdated)
	}

	// Calculate fresh stats
	newStats := CalculatePlayerStats(&player)

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"log":              logMsg,
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

// MovePathPlayerUpdate is a lightweight player snapshot for movement responses,
// avoiding sending huge inventories, recipes, and quest histories on high-frequency moves.
type MovePathPlayerUpdate struct {
	ID              uuid.UUID           `json:"id"`
	X               int                 `json:"x"`
	Y               int                 `json:"y"`
	CurrentRegion   string              `json:"current_region"`
	HP              float64             `json:"hp"`
	BaseHP          float64             `json:"base_hp"`
	CurrentMana     float64             `json:"current_mana"`
	MaxMana         float64             `json:"max_mana"`
	HumanQi         float64             `json:"human_qi"`
	PrimevalStones  int64               `json:"primeval_stones"`
	ImmortalStones  int64               `json:"immortal_stones"`
	VisitedSubzones []string            `json:"visited_subzones,omitempty"`
	ActiveCombat    *models.CombatState `json:"active_combat"`
	GuInventory     models.GuInventory  `json:"gu_inventory,omitempty"`
}

func buildMovePathPlayerUpdate(player *models.Player, beforeGus models.GuInventory) MovePathPlayerUpdate {
	update := MovePathPlayerUpdate{
		ID:              player.ID,
		X:               player.X,
		Y:               player.Y,
		CurrentRegion:   player.CurrentRegion,
		HP:              player.HP,
		BaseHP:          gamefx.EffectiveMaxHP(player),
		CurrentMana:     player.CurrentMana,
		MaxMana:         gamefx.EffectiveMaxMana(player),
		HumanQi:         player.HumanQi,
		PrimevalStones:  player.PrimevalStones,
		ImmortalStones:  player.ImmortalStones,
		VisitedSubzones: player.VisitedSubZones,
		ActiveCombat:    player.ActiveCombat,
	}
	if len(player.GuInventory) != len(beforeGus) {
		update.GuInventory = player.GuInventory
	}
	return update
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
		logger.S.Errorf("MovePlayerPath: failed to evaluate mana for player=%s: %v", authID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load player Aperture: %v", err)})
		return
	}
	if player.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể di chuyển!"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
		return
	}

	cleanupExpiredClearedCells(&player, clearedCellCooldown)

	if player.HP <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong trạng thái trọng thương, không thể tiếp tục di chuyển!"})
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

	// Free movement: the player may click anywhere in the world. The path is
	// walked cell by cell (each traversed cell triggers its action), stopping
	// early on the first combat. maxPathCells bounds the work of a single
	// request so the client chains requests for very long journeys. A smaller
	// chunk keeps server-side position commits fine-grained so other clients
	// see near-real-time movement and walking can be redirected sooner.
	const maxPathCells = 150
	path := lineCells(player.X, player.Y, req.X, req.Y)
	if len(path) > maxPathCells {
		path = path[:maxPathCells]
	}
	reached := len(path) < maxPathCells
	var eventLogs []string
	combatTriggered := false
	var lastStepLog string
	for _, step := range path {
		stepLog, triggered, errArrival := resolveCellArrival(tx, &player, step[0], step[1], "", true)
		if errArrival != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải định nghĩa quái vật từ cơ sở dữ liệu"})
			return
		}
		lastStepLog = stepLog
		// Filter out generic movement logs; only keep noteworthy events (encounters, discoveries, gathering, traps, pvp)
		if !strings.HasPrefix(stepLog, "Di chuyển thành công đến") {
			eventLogs = append(eventLogs, stepLog)
		}
		if triggered {
			combatTriggered = true
			reached = false
			break
		}
	}

	qiGained := 0.0

	// Auto-resolve combat if skip_battle is enabled
	if combatTriggered && player.CombatSettings.SkipBattle && player.ActiveCombat != nil {
		winner, _, skipSummary, err := AutoResolveCombat(tx, &player)
		if err != nil {
			logger.S.Errorf("Auto-resolve combat failed during path move: %v", err)
		} else {
			combatTriggered = false // Combat was auto-resolved
			enemyName := "kẻ thù"
			if e, ok := skipSummary["enemy_name"].(string); ok {
				enemyName = e
			}
			var autoLog string
			if winner == "Phe Ta" {
				autoLog = fmt.Sprintf("Tự động chiến thắng %s tại (%d, %d)! Nhân Khí +3.", enemyName, player.X, player.Y)
			} else {
				autoLog = fmt.Sprintf("Tự động chiến đấu với %s tại (%d, %d) thất bại! Đạo hữu bị truyền tống về an toàn.", enemyName, player.X, player.Y)
			}
			if skipSummary != nil {
				if loot, ok := skipSummary["loot"].([]string); ok && len(loot) > 0 {
					autoLog += " " + strings.Join(loot, " ")
				}
				if mineOccupied, ok := skipSummary["mine_occupied"].(int); ok && mineOccupied > 0 {
					autoLog += fmt.Sprintf(" Đã chiếm mỏ %d!", mineOccupied)
				}
			}
			eventLogs = append(eventLogs, autoLog)
			if err := tx.Commit(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
				return
			}
			finalLog := strings.Join(eventLogs, "\n")
			if qiGained > 0 {
				finalLog += fmt.Sprintf(" Nhân Khí +%.0f.", qiGained)
			}
			c.JSON(http.StatusOK, gin.H{
				"player":           buildMovePathPlayerUpdate(&player, before),
				"log":              finalLog,
				"combat_triggered": false,
				"skip_result":      skipSummary,
				"reached":          reached,
			})
			return
		}
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		logger.S.Errorf("MovePlayerPath: failed to sync Gu inventory for player=%s: %v", player.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}
	var combatJSON any
	if player.ActiveCombat != nil {
		b, err := json.Marshal(player.ActiveCombat)
		if err != nil {
			logger.S.Errorf("MovePlayerPath: failed to marshal active combat for player=%s: %v", player.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
			return
		}
		combatJSON = string(b)
	}
	if err := savePlayerMovement(tx, &player, combatJSON); err != nil {
		logger.S.Errorf("MovePlayerPath: failed to update player=%s at (%d,%d), region=%s, combat=%t: %v", player.ID, player.X, player.Y, player.CurrentRegion, player.ActiveCombat != nil, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}

	SyncPartyMembersLocation(tx, &player)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	broadcastWorldPos(WorldPosEvent{
		ID:        player.ID.String(),
		Username:  player.Username,
		AvatarURL: player.AvatarURL,
		X:         player.X,
		Y:         player.Y,
		TS:        time.Now().UnixMilli(),
	})

	if combatTriggered {
		PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	} else {
		PublishPlayerSnapshot(&player, PlayerEventUpdated)
	}

	var finalLog string
	if len(eventLogs) > 0 {
		finalLog = strings.Join(eventLogs, "\n")
		if !combatTriggered {
			finalLog += fmt.Sprintf("\nĐã di chuyển đến (%d, %d).", player.X, player.Y)
		}
	} else {
		if lastStepLog != "" {
			finalLog = lastStepLog
		} else {
			finalLog = fmt.Sprintf("Đã di chuyển đến (%d, %d).", player.X, player.Y)
		}
	}
	if qiGained > 0 {
		finalLog += fmt.Sprintf(" Nhân Khí +%.0f.", qiGained)
	}

	c.JSON(http.StatusOK, gin.H{
		"player":           buildMovePathPlayerUpdate(&player, before),
		"log":              finalLog,
		"combat_triggered": combatTriggered,
		"reached":          reached,
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
	Region  string `json:"region_id,omitempty"`
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
			regionID := DeriveRegionAt(cx, cy)
			if isCellClearedRecently(regionID, cx, cy, clearedCellCooldown) {
				continue
			}
			out = append(out, NearbyCell{X: cell.X, Y: cell.Y, Type: cell.Type, Content: cell.Content, Rank: cell.Rank, SubRank: cell.SubRank, Region: regionID})
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetCellsRect resolves the non-empty cells inside an explicit rectangle
// (min_x/min_y/max_x/max_y). Used by the world map to prefetch the exact
// visible area or a band ahead of the player. Bounded to prevent abuse.
func GetCellsRect(c *gin.Context) {
	if _, exists := middleware.GetPlayerID(c); !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var minX, minY, maxX, maxY int
	fmt.Sscanf(c.Query("min_x"), "%d", &minX)
	fmt.Sscanf(c.Query("min_y"), "%d", &minY)
	fmt.Sscanf(c.Query("max_x"), "%d", &maxX)
	fmt.Sscanf(c.Query("max_y"), "%d", &maxY)

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= WorldWidth {
		maxX = WorldWidth - 1
	}
	if maxY >= WorldHeight {
		maxY = WorldHeight - 1
	}
	if maxX < minX || maxY < minY {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rectangle tọa độ không hợp lệ."})
		return
	}

	// Bound the resolved area so a single request stays cheap.
	const maxSide = 900
	if maxX-minX+1 > maxSide {
		maxX = minX + maxSide - 1
	}
	if maxY-minY+1 > maxSide {
		maxY = minY + maxSide - 1
	}

	out := []NearbyCell{}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			cell := ResolveCell(x, y)
			if cell.Type == "Empty" {
				continue
			}
			regionID := DeriveRegionAt(x, y)
			if isCellClearedRecently(regionID, x, y, clearedCellCooldown) {
				continue
			}
			out = append(out, NearbyCell{X: cell.X, Y: cell.Y, Type: cell.Type, Content: cell.Content, Rank: cell.Rank, SubRank: cell.SubRank, Region: regionID})
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetNearbyPlayers returns a list of other active players within a given radius
func GetNearbyPlayers(c *gin.Context) {
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
		if entry, ok := worldPosIndex.Get(authID.String()); ok {
			x, y = entry.X, entry.Y
		} else if player, err := loadPlayerForStatus(db.DB, authID, false); err == nil {
			x, y = player.X, player.Y
		}
	}

	type NearbyPlayer struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		X         int    `json:"x"`
		Y         int    `json:"y"`
	}

	// Fast path: serve from the in-memory position index. Fall back to SQL only
	// while the index is still cold (e.g. very early startup before the first
	// sync worker reconcile).
	entries := worldPosIndex.Nearby(x, y, r, authID.String())
	if worldPosIndex.Len() == 0 {
		entries = queryNearbyPlayersSQL(c, authID, x, y, r)
	}

	nearbyPlayers := make([]NearbyPlayer, 0, len(entries))
	for _, e := range entries {
		nearbyPlayers = append(nearbyPlayers, NearbyPlayer{
			ID:        e.ID,
			Username:  e.Username,
			AvatarURL: e.AvatarURL,
			X:         e.X,
			Y:         e.Y,
		})
	}

	c.JSON(http.StatusOK, nearbyPlayers)
}

// queryNearbyPlayersSQL is the legacy SQL path for nearby players, kept as a
// cold-start fallback for the in-memory position index.
func queryNearbyPlayersSQL(c *gin.Context, authID uuid.UUID, x, y, r int) []WorldPosEntry {
	entries := []WorldPosEntry{}
	rows, errQuery := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, username, COALESCE(avatar_url, ''), x, y
		FROM players
		WHERE id != $1
		  AND last_mana_update >= NOW() - INTERVAL '15 minutes'
		  AND x >= $2 AND x <= $3
		  AND y >= $4 AND y <= $5
	`, authID, x-r, x+r, y-r, y+r)
	if errQuery != nil {
		logger.S.Error("Failed to fetch nearby players:", errQuery)
		return entries
	}
	defer rows.Close()
	for rows.Next() {
		var e WorldPosEntry
		if errScan := rows.Scan(&e.ID, &e.Username, &e.AvatarURL, &e.X, &e.Y); errScan == nil {
			e.TS = time.Now().UnixMilli()
			entries = append(entries, e)
		}
	}
	return entries
}

// GetRegions fetches all active regions (zones) from the database
func GetRegions(c *gin.Context) {
	var regions []models.Region
	if err := cache.GetJSON("explore_regions", &regions); err == nil && len(regions) > 0 {
		c.JSON(http.StatusOK, regions)
		return
	}

	query := `
		SELECT id, name, description, width, height, min_rank, max_rank, is_active, created_at, updated_at,
		       COALESCE(parent_id,''), COALESCE(zone_type,'KHU_VUC'), COALESCE(min_x,0), COALESCE(min_y,0),
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

	for rows.Next() {
		var r models.Region
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Width, &r.Height, &r.MinRank, &r.MaxRank, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
			&r.ParentID, &r.ZoneType, &r.MinX, &r.MinY, &r.MaxX, &r.MaxY, &r.EntryX, &r.EntryY); err != nil {
			continue
		}
		regions = append(regions, r)
	}

	_ = cache.SetJSON("explore_regions", regions, 24*time.Hour)
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

func savePlayerMovement(tx *sql.Tx, player *models.Player, combatJSON any) error {
	if ZoneByID(player.CurrentRegion) == nil {
		player.CurrentRegion = HubRegionID
	}

	// Movement awards resources and may mutate hunger/progress in the same
	// transaction. Persist the complete authoritative result synchronously so a
	// Redis snapshot cannot survive a rollback or overwrite a newer mutation.
	_, err := tx.Exec(`
		UPDATE players
		SET current_mana = $1, x = $2, y = $3, current_region = $4, cleared_cells = $5, active_combat = $6, last_mana_update = $7, primeval_stones = $8, hp = $9, max_mana = $10, human_qi = $11, visited_subzones = $12, version = version + 1
		WHERE id = $13
	`, player.CurrentMana, player.X, player.Y, player.CurrentRegion, player.ClearedCells, combatJSON, time.Now(), player.PrimevalStones, player.HP, player.MaxMana, player.HumanQi, player.VisitedSubZones, player.ID)
	if err == nil {
		player.Version++
	}
	return err
}
