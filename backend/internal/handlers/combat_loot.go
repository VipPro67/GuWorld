package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
			if entry.GuID != "" {
				for i := 0; i < qty; i++ {
					rolled.Gus = append(rolled.Gus, entry.GuID)
				}
			}
		}
	}
	return rolled
}

func ApplyLoot(tx *sql.Tx, player *models.Player, loot RolledLoot) []string {
	isExplore := player.CurrentRegion != "Southern_Border" && player.CurrentRegion != "Lobby"
	var logs []string
	if !isExplore {
		// items go directly via InventorySvc in the loop below
	} else {
		if player.TempLoot.MaterialsInventory == nil {
			player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
		}
	}
	materialsToAdd := make(map[string]int)

	for itemID, qty := range loot.Items {
		if itemID == "primeval_stone" {
			if !isExplore {
				player.PrimevalStones += float64(qty)
				logs = append(logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch.", qty))
			} else {
				player.TempLoot.PrimevalStones += float64(qty)
				logs = append(logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch (giữ tạm thời).", qty))
			}
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
			if !isExplore {
				materialsToAdd[itemID] += qty
				logs = append(logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm thú dữ: %dx %s.", qty, name))
			} else {
				player.TempLoot.MaterialsInventory[itemID] += qty
				logs = append(logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm thú dữ: %dx %s (giữ tạm thời).", qty, name))
			}
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
		if !isExplore {
			player.GuInventory = append(player.GuInventory, newInstance)
			logs = append(logs, fmt.Sprintf("Chúc mừng đạo hữu thu phục được hoang dã Cổ trùng: [%s]!", guName))
		} else {
			player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, newInstance)
			logs = append(logs, fmt.Sprintf("Chúc mừng đạo hữu thu phục được hoang dã Cổ trùng: [%s] (giữ tạm thời)!", guName))
		}
	}
	return logs
}

func TransferTempLootToInventory(tx *sql.Tx, player *models.Player) {
	InventorySvc.BulkAddItems(tx, player.ID.String(), "materials", player.TempLoot.MaterialsInventory)
	player.PrimevalStones += player.TempLoot.PrimevalStones
	player.GuInventory = append(player.GuInventory, player.TempLoot.GuInventory...)
}
