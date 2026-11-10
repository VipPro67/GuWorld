package handlers

import (
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func IntegratePassiveGus(player *models.Player, logs *[]string) []string {
	var consumedIDs []string
	newInventory := models.GuInventory{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			spec, ok := models.GetGu(gu.GuID, gu.Rank)
			if ok && spec.Type == "Passive" {
				gu.IntegrationTime += 1.0
				if gu.IntegrationTime >= 5.0 {
					modName := spec.Name + " Integrated"
					bonusText := "các chỉ số bổ trợ"
					if profile, ok := models.GrowthProfile(spec); ok {
						bonusText = fmt.Sprintf("+%.0f Tấn công, +%.0f Phòng thủ, +%.0f HP, +%.0f Tốc độ", profile.MaxAttack, profile.MaxDefense, profile.MaxHP, profile.MaxSpeed)
					}

					alreadyExists := false
					for _, existingMod := range player.BodyMods {
						if existingMod == modName {
							alreadyExists = true
							break
						}
					}
					if !alreadyExists {
						player.BodyMods = append(player.BodyMods, modName)
					}

					*logs = append(*logs, fmt.Sprintf("✨ Thể chất cải tạo hoàn tất: Cổ trùng %s đã hoàn toàn dung nhập vào da thịt, khắc sâu Đạo Ngân lên cơ thể đạo hữu. Cổ trùng tiêu biến, đạo hữu nhận được vĩnh viễn %s!", spec.Name, bonusText))
					consumedIDs = append(consumedIDs, gu.InstanceID)
					continue
				}
			}
		}
		newInventory = append(newInventory, gu)
	}
	player.GuInventory = newInventory
	RebuildEquippedGus(player)
	return consumedIDs
}
