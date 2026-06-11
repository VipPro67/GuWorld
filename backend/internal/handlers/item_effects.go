package handlers

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type ItemEffectHandler func(player *models.Player, item models.ItemSpec) (string, error)

var ValidateItemEffects = map[string]ItemEffectHandler{
	"mortal_only":   mortalOnlyEffect,
	"immortal_only": immortalOnlyEffect,
	"min_rank":      minRankEffect,
	"max_rank":      maxRankEffect,
}

var ApplyItemEffects = map[string]ItemEffectHandler{
	"restore_hp":             restoreHPEffect,
	"restore_mana":           restoreManaEffect,
	"soul_boost":             soulBoostEffect,
	"dao_mark_gain":          daoMarkGainEffect,
	"remove_negative_status": removeNegativeStatusEffect,
	"lifespan_boost":         lifespanBoostEffect,
	"open_mine_chest":        openMineChestEffect,
	"heal_aperture":          healApertureEffect,
	"regen_hp_out_of_combat": regenHPOutOfCombatEffect,
}

func healApertureEffect(player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["heal_aperture"]
	if val <= 0 { return "", nil }
	if !isImmortalCultivator(player) {
		return "", fmt.Errorf("%s chỉ dành cho Cổ Tiên để trị liệu tiên khiếu.", item.Name)
	}
	player.ApertureDamage -= val
	if player.ApertureDamage < 0 { player.ApertureDamage = 0 }
	return fmt.Sprintf("Đạo hữu sử dụng %s, khôi phục %.0f hư tổn tiên khiếu!", item.Name, val), nil
}

func regenHPOutOfCombatEffect(player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["regen_hp_out_of_combat"]
	if val <= 0 { return "", nil }
	
	// We'll just return success here and the main `UseItem` will handle Redis TTL or we can handle it here
	// But `cache` package might cause circular dependency if imported? No, `handlers` imports `cache` in `marketplace.go`.
	return fmt.Sprintf("Đạo hữu sử dụng %s, kích hoạt trạng thái Hồi Phục Sinh Lực trong 10 phút!", item.Name), nil
}

func lifespanBoostEffect(player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["lifespan_boost"]
	player.MaxLifespan += val
	player.Lifespan += val
	return fmt.Sprintf("Đạo hữu sử dụng %s, gia tăng %.0f năm thọ nguyên tối đa và hồi phục %.0f năm tuổi thọ!", item.Name, val, val), nil
}

func restoreManaEffect(player *models.Player, item models.ItemSpec) (string, error) {
	value := item.Effects["restore_mana"]
	player.CurrentMana += value
	if player.CurrentMana > player.MaxMana {
		player.CurrentMana = player.MaxMana
	}
	return fmt.Sprintf("Đạo hữu sử dụng %s, phục hồi %.0f Chân Nguyên!", item.Name, value), nil
}

func restoreHPEffect(player *models.Player, item models.ItemSpec) (string, error) {
	value := item.Effects["restore_hp"]
	player.HP += value
	if player.HP > player.BaseHP {
		player.HP = player.BaseHP
	}
	return fmt.Sprintf("Đạo hữu sử dụng %s, hồi phục %.0f HP!", item.Name, value), nil
}

func mortalOnlyEffect(player *models.Player, item models.ItemSpec) (string, error) {
	if isImmortalCultivator(player) {
		return "", fmt.Errorf("%s chỉ bổ sung Chân Nguyên phàm cấp; Cổ Tiên cần Tiên Nguyên Thạch hoặc Tiên Khiếu ngưng tụ Tiên Nguyên.", item.Name)
	}
	return "", nil
}

func immortalOnlyEffect(player *models.Player, item models.ItemSpec) (string, error) {
	if !isImmortalCultivator(player) {
		return "", fmt.Errorf("%s chỉ dành cho Cổ Tiên.", item.Name)
	}
	return "", nil
}

func minRankEffect(player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["min_rank"]
	if float64(player.Rank) < val {
		return "", fmt.Errorf("Yêu cầu tu vi tối thiểu %s để sử dụng %s.", getRankName(int(val)), item.Name)
	}
	return "", nil
}

func maxRankEffect(player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["max_rank"]
	if float64(player.Rank) > val {
		return "", fmt.Errorf("Tu vi vượt quá giới hạn tối đa %s để sử dụng %s.", getRankName(int(val)), item.Name)
	}
	return "", nil
}

func soulBoostEffect(player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["soul_boost"]
	if player.DaoMarks == nil {
		player.DaoMarks = make(models.DaoMarksMap)
	}
	player.DaoMarks["soul"] += int(val)
	return fmt.Sprintf("Hồn phách của đạo hữu được tráng kiện (+%.0f điểm Hồn lực).", val), nil
}

func daoMarkGainEffect(player *models.Player, item models.ItemSpec) (string, error) {
	val := item.Effects["dao_mark_gain"]
	if val <= 0 {
		return "", nil
	}
	pathKey := strings.ToLower(strings.TrimSuffix(strings.ReplaceAll(item.Path, " ", ""), "Path"))
	if pathKey == "" {
		pathKey = "strength" // fallback
	}
	if player.DaoMarks == nil {
		player.DaoMarks = make(models.DaoMarksMap)
	}
	player.DaoMarks[pathKey] += int(val)
	return fmt.Sprintf("Nhận thêm %d Đạo Ngân %s.", int(val), item.Path), nil
}

func removeNegativeStatusEffect(player *models.Player, item models.ItemSpec) (string, error) {
	if player.ActiveCombat != nil && len(player.ActiveCombat.PlayerStatusEffects) > 0 {
		player.ActiveCombat.PlayerStatusEffects = []models.StatusEffect{}
		return fmt.Sprintf("%s giải trừ toàn bộ trạng thái tiêu cực của đạo hữu trong trận chiến.", item.Name), nil
	}
	return "", nil
}

func openMineChestEffect(player *models.Player, item models.ItemSpec) (string, error) {
	stones := 200 + rand.Intn(301)
	player.PrimevalStones += stones

	msg := fmt.Sprintf("Mở %s thành công!\n- Nhận được %d Nguyên Thạch\n", item.Name, stones)

	var materials []string
	for id, spec := range models.ItemRegistry {
		if spec.Type == "Material" {
			materials = append(materials, id)
		}
	}
	if len(materials) == 0 {
		materials = []string{"wild_boar_tusk", "jade_stone", "nine_beast_kings_eyes", "stone_man_rice"}
	}

	numMats := 1 + rand.Intn(3)
	if player.MaterialsInventory == nil {
		player.MaterialsInventory = make(models.MaterialsInventory)
	}
	for i := 0; i < numMats; i++ {
		matID := materials[rand.Intn(len(materials))]
		player.MaterialsInventory[matID]++
		matName := matID
		if spec, ok := models.ItemRegistry[matID]; ok {
			matName = spec.Name
		}
		msg += fmt.Sprintf("- Nhận được 1x %s\n", matName)
	}

	if rand.Float64() < 0.05 {
		var guIDs []string
		for id := range models.GuRegistry {
			guIDs = append(guIDs, id)
		}
		if len(guIDs) > 0 {
			guID := guIDs[rand.Intn(len(guIDs))]
			newInstance := models.GuInstance{
				InstanceID: "gu-" + uuid.New().String(),
				GuID:       guID,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
			}
			player.GuInventory = append(player.GuInventory, newInstance)
			guName := guID
			if spec, ok := models.GuRegistry[guID]; ok {
				guName = spec.Name
			}
			msg += fmt.Sprintf("- Cơ duyên xảo hợp! Đạo hữu nhận thêm Cổ trùng: [%s]!\n", guName)
		}
	}

	return msg, nil
}

