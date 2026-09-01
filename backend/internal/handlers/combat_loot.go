package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type RolledLoot struct {
	Items map[string]int
	Gus   []string
}

func lootChanceMultiplier(rank int) float64 {
	switch rank {
	case 1:
		return 1.0
	case 2:
		return 1.1
	case 3:
		return 1.2
	case 4:
		return 1.4
	case 5:
		return 1.6
	case 6:
		return 1.0
	case 7:
		return 1.5
	case 8:
		return 2.5
	case 9:
		return 4.0
	default:
		if rank > 9 {
			return 4.0
		}
		return 1.0
	}
}

func lootQuantityMultiplier(rank int) float64 {
	switch rank {
	case 1:
		return 1.0
	case 2:
		return 2.0
	case 3:
		return 3.0
	case 4:
		return 4.0
	case 5:
		return 5.0
	case 6:
		return 1.0
	case 7:
		return 3.0
	case 8:
		return 5.0
	case 9:
		return 8.0
	default:
		if rank > 9 {
			return 8.0
		}
		return 1.0
	}
}

func RollLoot(lt models.LootTable, rank int) RolledLoot {
	rolled := RolledLoot{
		Items: make(map[string]int),
		Gus:   []string{},
	}
	chanceMul := lootChanceMultiplier(rank)
	qtyMul := lootQuantityMultiplier(rank)

	for _, entry := range lt.Entries {
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

func ApplyLoot(tx *sql.Tx, player *models.Player, loot RolledLoot) []string {
	var logs []string
	materialsToAdd := make(map[string]int)

	for itemID, qty := range loot.Items {
		if itemID == "primeval_stone" {
			player.PrimevalStones += int64(qty)
			logs = append(logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch.", qty))
		} else {
			name := itemID
			switch itemID {
			case "boar_meat":
				name = "Thịt Lợn Rừng"
			case "wild_boar_tusk":
				name = "Nanh Lợn Rừng"
			case "jade_stone":
				name = "Ngọc Thạch"
			case "moon_orchid_petal":
				name = "Cánh hoa Nguyệt Lan"
			case "green_bamboo_wine":
				name = "Rượu Thanh Trúc"
			}
			materialsToAdd[itemID] += qty
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
				logger.S.Errorf("Failed to persist looted Gu %s to DB for player %s: %v", guID, player.ID, err)
				continue // Skip if DB insertion failed
			}
		}

		player.GuInventory = append(player.GuInventory, newInstance)
		logs = append(logs, fmt.Sprintf("Chúc mừng đạo hữu thu phục được hoang dã Cổ trùng: [%s]!", guName))

		// Broadcast ONLY AFTER SUCCESSFUL ADDITION TO PLAYER INVENTORY IN DB
		isImmortal := guSpec.IsImmortal || guSpec.Rank >= 6 || models.HasImmortalRank(guID)
		if isImmortal {
			bossName := "Thái Cổ Yêu Quái"
			if player.ActiveCombat != nil && player.ActiveCombat.EnemyName != "" {
				bossName = player.ActiveCombat.EnemyName
			}
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
			return 35.0 // 35% (Boss - Cổ Nhất Chuyển)
		case 2:
			return 25.0 // 25% (Boss - Cổ Nhị Chuyển)
		case 3:
			return 18.0 // 18% (Boss - Cổ Tam Chuyển)
		case 4:
			return 12.0 // 12% (Boss - Cổ Tứ Chuyển)
		case 5:
			return 8.0  // 8%  (Boss - Cổ Ngũ Chuyển)
		case 6:
			return 2.5  // 2.5% (Boss - Tiên Cổ Lục Chuyển)
		case 7:
			return 1.5  // 1.5% (Boss - Tiên Cổ Thất Chuyển)
		case 8:
			return 0.8  // 0.8% (Boss - Tiên Cổ Bát Chuyển)
		case 9:
			return 0.3  // 0.3% (Boss - Tiên Cổ Cửu Chuyển)
		default:
			if rank >= 6 {
				return 0.3
			}
			return 15.0
		}
	}

	// Elite / Normal Enemy
	switch rank {
	case 1:
		return 20.0 // 20%
	case 2:
		return 15.0 // 15%
	case 3:
		return 10.0 // 10%
	case 4:
		return 6.0  // 6%
	case 5:
		return 4.0  // 4%
	case 6:
		return 1.0  // 1.0% (Tiên Cổ Lục Chuyển)
	case 7:
		return 0.5  // 0.5% (Tiên Cổ Thất Chuyển)
	case 8:
		return 0.2  // 0.2% (Tiên Cổ Bát Chuyển)
	case 9:
		return 0.1  // 0.1% (Tiên Cổ Cửu Chuyển)
	default:
		if rank >= 6 {
			return 0.1
		}
		return 8.0
	}
}
