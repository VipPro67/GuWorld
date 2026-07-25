package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type RolledLoot struct {
	Items map[string]int
	Gus   []string
}

func RollLoot(lt models.LootTable) RolledLoot {
	rolled := RolledLoot{
		Items: make(map[string]int),
		Gus:   []string{},
	}
	for _, entry := range lt.Entries {
		if rand.Float64() <= entry.Chance {
			qty := 1
			if entry.MaxQty > entry.MinQty && entry.MinQty > 0 {
				qty = entry.MinQty + rand.Intn(entry.MaxQty-entry.MinQty+1)
			} else if entry.MinQty > 0 {
				qty = entry.MinQty
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

	for itemID, qty := range loot.Items {
		if itemID == "primeval_stone" {
			if !isExplore {
				player.PrimevalStones += qty
				logs = append(logs, fmt.Sprintf("Thu hoạch Nguyên Thạch: +%d Nguyên Thạch.", qty))
			} else {
				player.TempLoot.PrimevalStones += qty
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
				InventorySvc.AddItem(tx, player.ID.String(), itemID, "materials", qty)
				logs = append(logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm thú dữ: %dx %s.", qty, name))
			} else {
				player.TempLoot.MaterialsInventory[itemID] += qty
				logs = append(logs, fmt.Sprintf("Thu hoạch chiến lợi phẩm thú dữ: %dx %s (giữ tạm thời).", qty, name))
			}
		}
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
	for itemID, qty := range player.TempLoot.MaterialsInventory {
		InventorySvc.AddItem(tx, player.ID.String(), itemID, "materials", qty)
	}
	player.PrimevalStones += player.TempLoot.PrimevalStones
	player.GuInventory = append(player.GuInventory, player.TempLoot.GuInventory...)
}
