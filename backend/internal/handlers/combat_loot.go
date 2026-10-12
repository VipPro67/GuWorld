package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/google/uuid"
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
	rolled := RolledLoot{
		Items: make(map[string]int),
		Gus:   []string{},
	}
	chanceMul := lootChanceMultiplier(rank)
	qtyMul := lootQuantityMultiplier(rank)

	for _, entry := range lt.Entries {
		if !lootEntryEligible(entry, rank) {
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

// RollMonsterLoot rolls a monster PvE kill's full loot: the shared per-rank
// baseline pool (loot_monster_common, gated by entry.Rank) plus the monster's
// own override table (rank-filtered the same way).
func RollMonsterLoot(tx *sql.Tx, overrideID string, rank int) (RolledLoot, error) {
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
	return loot, nil
}

// heavenSecretLootPreview builds the "[Thiên Cơ hiển lộ ...]" drop preview for
// the LOOT_PREVIEW Gu capability, matching exactly what RollMonsterLoot rolls
// (same pool + override, same rank filter). Returns "" when there is nothing
// to preview.
func heavenSecretLootPreview(tx *sql.Tx, overrideID string, rank int) string {
	chanceMul := lootChanceMultiplier(rank)
	qtyMul := lootQuantityMultiplier(rank)
	msg := fmt.Sprintf(" [Thiên Cơ hiển lộ (Rank %d x%.1f) - Tỉ lệ rơi đồ: ", rank, qtyMul)
	first := true
	appendPreview := func(entry models.LootEntry) {
		if !lootEntryEligible(entry, rank) {
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
			appendPreview(e)
		}
	}
	if overrideID != "" {
		if lt, err := db.LoadLootTable(tx, overrideID); err == nil {
			for _, e := range lt.Entries {
				appendPreview(e)
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
	if strings.EqualFold(cs.EnemyType, "Boss") || strings.EqualFold(cs.EnemyType, "World_Boss") || strings.EqualFold(cs.EnemyType, "Mine_PVE") {
		return true
	}
	for _, e := range cs.Enemies {
		if strings.EqualFold(e.Kind, "Boss") {
			return true
		}
	}
	return false
}

// GetGuDropChancePercent returns the drop chance percentage (0.0 - 100.0) based on Gu rank and whether enemy is a Boss vs Elite.
// Defeating a Boss gives a higher drop chance for Gu than Elites/Normal enemies.
// Immortal Gu (Rank >= 6) have significantly lower drop rates to ensure their rarity and uniqueness.
func GetGuDropChancePercent(rank int, isBoss bool) float64 {
	if isBoss {
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
	}

	// Elite Enemy
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
