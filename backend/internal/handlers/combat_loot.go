package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type RolledLoot struct {
	Items map[string]int
	Gus   []string
}

func lootChanceMultiplier(rank int) float64 {
	return gamefx.LootChanceMultiplier(rank)
}

func lootQuantityMultiplier(rank int) float64 {
	return gamefx.LootQuantityMultiplier(rank)
}

// lootMonsterCommonPoolID is the shared per-rank baseline loot table every
// monster PvE kill also rolls (entry.Rank gates which tier applies).
const lootMonsterCommonPoolID = "loot_monster_common"

// lootEntryEligible reports whether an entry may roll for a kill at the given
// enemy rank: jackpot entries always roll (rare bonus), rank 0 entries are
// universal/theme-bound, rank N entries only roll when the enemy rank is N.
func lootEntryEligible(e models.LootEntry, rank int) bool {
	return e.Jackpot || e.Rank == 0 || e.Rank == rank
}

func RollLoot(lt models.LootTable, rank int) RolledLoot {
	return rollLoot(lt, rank, true)
}

// rollRegionLootTable rolls every entry of a region loot table ignoring the
// per-entry rank gate. Region loot is already rank-scoped by the matching
// region_loot_tables row (min_rank/max_rank), so re-applying entry.Rank only
// opens gaps: a region-wide rule whose entries are tagged with a single rank
// would otherwise drop nothing at every other rank the rule covers.
func rollRegionLootTable(lt models.LootTable, rank int) RolledLoot {
	return rollLoot(lt, rank, false)
}

func rollLoot(lt models.LootTable, rank int, gateByRank bool) RolledLoot {
	rolled := RolledLoot{
		Items: make(map[string]int),
		Gus:   []string{},
	}
	chanceMul := lootChanceMultiplier(rank)
	qtyMul := lootQuantityMultiplier(rank)

	for _, entry := range lt.Entries {
		if gateByRank && !lootEntryEligible(entry, rank) {
			continue
		}
		effectiveChance := math.Min(entry.Chance*chanceMul, 1.0)
		if rand.Float64() <= effectiveChance {
			qty := 1
			effectiveMin := int(math.Round(float64(entry.MinQty) * qtyMul))
			effectiveMax := int(math.Round(float64(entry.MaxQty) * qtyMul))
			if effectiveMax > effectiveMin && effectiveMin > 0 {
				qty = effectiveMin + rand.Intn(effectiveMax-effectiveMin+1)
			} else if effectiveMin > 0 {
				qty = effectiveMin
			}
			if qty < 1 {
				qty = 1
			}
			if entry.ItemID != "" {
				rolled.Items[entry.ItemID] += qty
			}
			if entry.GuID != "" && !models.HasImmortalRank(entry.GuID) {
				for i := 0; i < qty; i++ {
					rolled.Gus = append(rolled.Gus, entry.GuID)
				}
			}
		}
	}
	return rolled
}

// RollRegionLoot rolls region-specific loot based on region + monster type + rank.
// This is Layer 3 of the 3-layer loot system: region-specific resources.
// The lookup includes the region's ancestors (e.g. a KHU_VUC inherits its
// parent NGU_VUC's rules) so broad regional identity applies to subregions.
func RollRegionLoot(tx *sql.Tx, regionID string, monsterType string, rank int) RolledLoot {
	rolled := RolledLoot{
		Items: make(map[string]int),
		Gus:   []string{},
	}

	if regionID == "" || monsterType == "" {
		return rolled
	}

	// 1. Try in-memory registry first (0 DB queries, instant resolution)
	if rules := models.GetRegionLootRules(regionID, monsterType, rank); len(rules) > 0 {
		for _, r := range rules {
			if rand.Float64() > r.Chance {
				continue
			}
			if lt, ok := models.GetLootTable(r.LootTableID); ok {
				regionLoot := rollRegionLootTable(lt, rank)
				for k, v := range regionLoot.Items {
					rolled.Items[k] += v
				}
				rolled.Gus = append(rolled.Gus, regionLoot.Gus...)
			}
		}
		return rolled
	}

	if tx == nil {
		return rolled
	}

	// 2. Fallback to SQL query if registry has no rules for this region
	scope := regionLootScope(tx, regionID)
	rows, err := tx.Query(`
		SELECT loot_table_id, chance FROM region_loot_tables
		WHERE region_id = ANY($1)
		  AND lower(monster_type) = lower($2) 
		  AND min_rank <= $3 
		  AND max_rank >= $3
		  AND is_active = TRUE
		  AND chance > 0
		ORDER BY region_id, monster_type, loot_table_id`, pq.Array(scope), monsterType, rank)
	if err != nil {
		logger.S.Warnf("RollRegionLoot: query failed: %v", err)
		return rolled
	}

	// Drain and close the rows BEFORE issuing LoadLootTable on the same tx:
	// a Tx pins one connection, so querying loot_tables while these rows are
	// still open desyncs the protocol (pq: unexpected Parse response) and
	// poisons the whole transaction (every later Exec fails with bad connection).
	type regionLootRow struct {
		lootTableID string
		chance      float64
	}
	var buffered []regionLootRow
	for rows.Next() {
		var r regionLootRow
		if err := rows.Scan(&r.lootTableID, &r.chance); err != nil {
			continue
		}
		buffered = append(buffered, r)
	}
	rows.Close()

	for _, r := range buffered {
		if rand.Float64() > r.chance {
			continue
		}

		if lt, err := db.LoadLootTable(tx, r.lootTableID); err == nil {
			regionLoot := rollRegionLootTable(lt, rank)
			for k, v := range regionLoot.Items {
				rolled.Items[k] += v
			}
			rolled.Gus = append(rolled.Gus, regionLoot.Gus...)
		}
	}

	return rolled
}

// regionLootScope returns the region id plus its ancestor ids so region loot
// rules defined on a parent NGU_VUC apply to its KHU_VUC children.
func regionLootScope(tx *sql.Tx, regionID string) []string {
	scope := []string{regionID}
	current := regionID
	for i := 0; i < 8; i++ {
		var parent sql.NullString
		if err := tx.QueryRow(`SELECT parent_id FROM regions WHERE id = $1`, current).Scan(&parent); err != nil {
			break
		}
		if !parent.Valid || parent.String == "" {
			break
		}
		scope = append(scope, parent.String)
		current = parent.String
	}
	return scope
}

// RollMonsterLoot rolls a monster PvE kill's full loot: the shared per-rank
// baseline pool (loot_monster_common, gated by entry.Rank) plus the monster's
// own override table (rank-filtered the same way) plus region-specific loot.
func RollMonsterLoot(tx *sql.Tx, overrideID string, rank int, regionID string, monsterType string) (RolledLoot, error) {
	loot := RolledLoot{
		Items: make(map[string]int),
		Gus:   []string{},
	}
	if pool, err := db.LoadLootTable(tx, lootMonsterCommonPoolID); err == nil {
		loot = RollLoot(pool, rank)
	} else {
		logger.S.Warnf("RollMonsterLoot: khong load duoc bể loot chung %s: %v", lootMonsterCommonPoolID, err)
	}
	if overrideID != "" {
		if lt, err := db.LoadLootTable(tx, overrideID); err == nil {
			override := RollLoot(lt, rank)
			for k, v := range override.Items {
				loot.Items[k] += v
			}
			loot.Gus = append(loot.Gus, override.Gus...)
		} else {
			logger.S.Warnf("RollMonsterLoot: khong load duoc loot table %s: %v", overrideID, err)
		}
	}

	// Layer 3: Region-specific loot (filter by monster_type)
	if regionID != "" && monsterType != "" {
		regionLoot := RollRegionLoot(tx, regionID, monsterType, rank)
		for k, v := range regionLoot.Items {
			loot.Items[k] += v
		}
		loot.Gus = append(loot.Gus, regionLoot.Gus...)
	}

	return loot, nil
}

// heavenSecretLootPreview builds the "[Thiên Cơ hiển lộ ...]" drop preview for
// the LOOT_PREVIEW Gu capability, matching exactly what RollMonsterLoot rolls
// (same pool + override + region loot, same rank filter). Returns "" when there is nothing
// to preview.
func heavenSecretLootPreview(tx *sql.Tx, overrideID string, rank int, regionID string, monsterType string) string {
	chanceMul := lootChanceMultiplier(rank)
	qtyMul := lootQuantityMultiplier(rank)
	msg := fmt.Sprintf(" [Thiên Cơ hiển lộ (Rank %d x%.1f) - Tỉ lệ rơi đồ: ", rank, qtyMul)
	first := true
	appendPreview := func(entry models.LootEntry, gateByRank bool) {
		if gateByRank && !lootEntryEligible(entry, rank) {
			return
		}
		if !first {
			msg += ", "
		}
		scaledChance := math.Min(entry.Chance*chanceMul, 1.0)
		scaledMin := int(math.Round(float64(entry.MinQty) * qtyMul))
		scaledMax := int(math.Round(float64(entry.MaxQty) * qtyMul))
		if itemID := entry.ItemID; itemID != "" {
			msg += fmt.Sprintf("%s x%d-%d (%.1f%%)", itemID, scaledMin, scaledMax, scaledChance*100)
		} else if entry.GuID != "" {
			msg += fmt.Sprintf("%s (%.1f%%)", entry.GuID, scaledChance*100)
		}
		first = false
	}
	if pool, err := db.LoadLootTable(tx, lootMonsterCommonPoolID); err == nil {
		for _, e := range pool.Entries {
			appendPreview(e, true)
		}
	}
	if overrideID != "" {
		if lt, err := db.LoadLootTable(tx, overrideID); err == nil {
			for _, e := range lt.Entries {
				appendPreview(e, true)
			}
		}
	}
	if regionID != "" && monsterType != "" {
		scope := regionLootScope(tx, regionID)
		rows, err := tx.Query(`
			SELECT loot_table_id FROM region_loot_tables
			WHERE region_id = ANY($1) AND lower(monster_type) = lower($2)
			  AND min_rank <= $3 AND max_rank >= $3
			  AND is_active = TRUE AND chance > 0
			ORDER BY region_id, monster_type, loot_table_id`, pq.Array(scope), monsterType, rank)
		if err == nil {
			// Buffer ids and close the rows before LoadLootTable: querying on
			// the same Tx while these rows are open poisons the connection.
			lootTableIDs := []string{}
			for rows.Next() {
				var lootTableID string
				if scanErr := rows.Scan(&lootTableID); scanErr != nil {
					continue
				}
				lootTableIDs = append(lootTableIDs, lootTableID)
			}
			rows.Close()
			for _, lootTableID := range lootTableIDs {
				if lt, loadErr := db.LoadLootTable(tx, lootTableID); loadErr == nil {
					for _, e := range lt.Entries {
						appendPreview(e, false)
					}
				}
			}
		}
	}
	if first {
		return ""
	}
	return msg + "]"
}

func ApplyLoot(tx *sql.Tx, player *models.Player, loot RolledLoot) []string {
	var logs []string
	materialsToAdd := make(map[string]int64)

	for itemID, qty := range loot.Items {
		if itemID == "primeval_stone" {
			player.PrimevalStones += float64(qty)
			logs = append(logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch.", qty))
		} else {
			name := itemID
			switch itemID {
			case "boar_meat":
				name = "Thịt Lợn Rừng"
			case "wild_boar_tusk":
				name = "Nanh Lợn Rừng"
			case "bear_gall":
				name = "Mật Gấu"
			case "moon_orchid_petal":
				name = "Cánh hoa Nguyệt Lan"
			case "green_bamboo_wine":
				name = "Rượu Thanh Trúc"
			}
			materialsToAdd[itemID] += int64(qty)
			logs = append(logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm thú dữ: %dx %s.", qty, name))
		}
	}

	if len(materialsToAdd) > 0 {
		InventorySvc.BulkAddItems(tx, player.ID.String(), "materials", materialsToAdd)
	}

	for _, guID := range loot.Gus {
		instID := "gu-" + uuid.New().String()
		guSpec, _ := models.GetGuAny(guID)
		newInstance := models.GuInstance{
			InstanceID: instID,
			GuID:       guID,
			Rank:       guSpec.Rank,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
		}
		guName := guID
		if guSpec.ID != "" {
			guName = guSpec.Name
		}

		if tx != nil {
			if err := db.AddPlayerGu(tx, player.ID.String(), newInstance); err != nil {
				if db.IsImmortalConflict(err) {
					logger.S.Warnf("Skipped looted immortal Gu %s: already exists in world", guID)
					continue
				}
				logger.S.Errorf("Failed to persist looted Gu %s to DB for player %s: %v", guID, player.ID, err)
				continue // Skip if DB insertion failed
			}
		}

		player.GuInventory = append(player.GuInventory, newInstance)
		logs = append(logs, fmt.Sprintf("Chúc mừng đạo hữu thu phục được hoang dã Cổ trùng: [%s]!", guName))

		bossName := "Thái Cổ Yêu Quái"
		if player.ActiveCombat != nil && player.ActiveCombat.EnemyName != "" {
			bossName = player.ActiveCombat.EnemyName
		}
		lootSummary := fmt.Sprintf("Cổ Sư [%s] bóc tách/thu phục thành công Cổ trùng [%s] R%d từ %s.", player.Username, guName, guSpec.Rank, bossName)
		_ = db.LogGuHistory(tx, newInstance.InstanceID, newInstance.GuID, guSpec.Rank, "CREATED_LOOT", &player.ID, nil, lootSummary, map[string]interface{}{
			"gu_name":   guName,
			"boss_name": bossName,
		})

		// Broadcast ONLY AFTER SUCCESSFUL ADDITION TO PLAYER INVENTORY IN DB
		isImmortal := guSpec.Rank >= 6 || models.HasImmortalRank(guID)
		if isImmortal {
			BroadcastImmortalGuLoot(player.ID, player.Username, bossName, guID, guSpec.Rank)
		}
	}
	return logs
}

// isBossEnemy returns true if the defeated enemy is a Boss/WorldBoss/MineBoss
func isBossEnemy(cs *models.CombatState) bool {
	if cs == nil {
		return false
	}
	if strings.EqualFold(cs.EnemyType, "Boss") || strings.EqualFold(cs.EnemyType, "World_Boss") {
		return true
	}
	for _, e := range cs.Enemies {
		if strings.EqualFold(e.Kind, "Boss") {
			return true
		}
	}
	return false
}

// guDropTier classifies an enemy type for Gu-drop purposes. Bosses roll the
// highest table, Elites the middle, Normal monsters the lowest.
func guDropTier(enemyType string) string {
	switch strings.ToUpper(strings.TrimSpace(enemyType)) {
	case "NORMAL", "MONSTER":
		return "normal"
	case "BOSS", "WORLD_BOSS", "WORLDBOSS", "WORLD BOSS", "TRIBULATION_BOSS":
		return "boss"
	default:
		return "elite"
	}
}

// guDropTierForCombat resolves the drop tier from a combat state, preserving
// isBossEnemy's broader boss detection (e.g. a Boss-typed combatant).
func guDropTierForCombat(cs *models.CombatState) string {
	if isBossEnemy(cs) {
		return "boss"
	}
	if cs != nil {
		return guDropTier(cs.EnemyType)
	}
	return "elite"
}

// GetGuDropChancePercent returns the drop chance percentage (0.0 - 100.0) for
// an equipped Gu of the given rank and enemy tier ("boss", "normal", "elite").
// Defeating a Boss gives a higher drop chance than an Elite, which in turn is
// higher than a plain Normal monster. Immortal Gu (Rank >= 6) have
// significantly lower rates to protect their rarity and uniqueness.
func GetGuDropChancePercent(rank int, tier string) float64 {
	switch tier {
	case "boss":
		switch rank {
		case 1:
			return 50.0 // 50% (Boss - Cổ Nhất Chuyển)
		case 2:
			return 40.0 // 40% (Boss - Cổ Nhị Chuyển)
		case 3:
			return 30.0 // 30% (Boss - Cổ Tam Chuyển)
		case 4:
			return 20.0 // 20% (Boss - Cổ Tứ Chuyển)
		case 5:
			return 15.0 // 15% (Boss - Cổ Ngũ Chuyển)
		case 6:
			return 1.0 // 1.0% (Boss - Tiên Cổ Lục Chuyển)
		case 7:
			return 0.5 // 0.5% (Boss - Tiên Cổ Thất Chuyển)
		case 8:
			return 0.2 // 0.2% (Boss - Tiên Cổ Bát Chuyển)
		case 9:
			return 0.05 // 0.05% (Boss - Tiên Cổ Cửu Chuyển)
		default:
			if rank >= 6 {
				return 0.05
			}
			return 15.0
		}
	case "normal":
		switch rank {
		case 1:
			return 15.0 // 15%
		case 2:
			return 10.0 // 10%
		case 3:
			return 6.0 // 6%
		case 4:
			return 3.0 // 3%
		case 5:
			return 1.5 // 1.5%
		case 6:
			return 0.05 // 0.05% (Tiên Cổ Lục Chuyển)
		case 7:
			return 0.02 // 0.02% (Tiên Cổ Thất Chuyển)
		case 8:
			return 0.01 // 0.01% (Tiên Cổ Bát Chuyển)
		case 9:
			return 0.005 // 0.005% (Tiên Cổ Cửu Chuyển)
		default:
			if rank >= 6 {
				return 0.005
			}
			return 1.5
		}
	default: // elite
		switch rank {
		case 1:
			return 40.0 // 40%
		case 2:
			return 30.0 // 30%
		case 3:
			return 20.0 // 20%
		case 4:
			return 12.0 // 12%
		case 5:
			return 8.0 // 8%
		case 6:
			return 0.3 // 0.3% (Tiên Cổ Lục Chuyển)
		case 7:
			return 0.1 // 0.1% (Tiên Cổ Thất Chuyển)
		case 8:
			return 0.05 // 0.05% (Tiên Cổ Bát Chuyển)
		case 9:
			return 0.01 // 0.01% (Tiên Cổ Cửu Chuyển)
		default:
			if rank >= 6 {
				return 0.01
			}
			return 8.0
		}
	}
}

// regionMonsterGuPool resolves the configured per-region Gu pool for monster
// combat, walking the region parent chain so a rule declared on a parent
// NGU_VUC applies to its KHU_VUC children. Child entries win over parents.
func regionMonsterGuPool(regionID string) []models.GuPoolEntry {
	if regionID == "" {
		return nil
	}
	var pool []models.GuPoolEntry
	seen := make(map[string]struct{})
	for _, id := range regionScopeIDs(regionID) {
		zone := ZoneByID(id)
		if zone == nil {
			continue
		}
		for _, e := range zone.PoolWeights.MonsterGus {
			if e.GuID == "" {
				continue
			}
			if _, dup := seen[e.GuID]; dup {
				continue
			}
			seen[e.GuID] = struct{}{}
			pool = append(pool, e)
		}
	}
	return pool
}
