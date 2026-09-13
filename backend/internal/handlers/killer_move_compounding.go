package handlers

import (
	"fmt"
	"math"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// CompoundEffectInfo describes an emergent effect created by combining Gu.
type CompoundEffectInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Value       float64  `json:"value"`
	Description string   `json:"description"`
	Sources     []string `json:"sources"`
}

// compoundRule defines when and how effects combine into an emergent effect.
type compoundRule struct {
	Name        string
	Requires    []string // all required effect keys (in active or compounded)
	Boosts      []string // optional effect keys that enhance
	MinCount    int      // minimum number of matching effects from Requires
	Description string
	Transform   func(coreValue float64, totalBoosts float64, numSupports int) (value float64, desc string)
	Type        string
}

var compoundRules = []compoundRule{
	{
		Name: "xuất huyết trọng thương", Type: "hemorrhage",
		Requires: []string{"damage", "bleed"},
		Boosts:   []string{"damage_multiplier"},
		MinCount: 2,
		Description: "Sát thương kết hợp với tổn thương máu tạo thành vết thương xuất huyết không ngừng. " +
			"Mỗi Cổ phụ trợ có hiệu ứng chảy máu gia tăng cường độ xuất huyết.",
		Transform: func(coreDamage float64, boost float64, num int) (float64, string) {
			bleedPerTurn := coreDamage*0.15 + float64(num)*5.0
			if boost > 0 {
				bleedPerTurn *= boost
			}
			bonusDmg := coreDamage * 0.1
			return bleedPerTurn + bonusDmg,
				fmt.Sprintf("Gây thêm %.0f sát thương xuất huyết mỗi lượt (cộng dồn %.0f sát thương tức thời)", bleedPerTurn, bonusDmg)
		},
	},
	{
		Name: "khiên sinh mệnh", Type: "regenerative_shield",
		Requires: []string{"add_base_defense", "defense", "heal", "restore_hp", "heal_based_on_strength"},
		Boosts:   []string{"defense_multiplier", "heal_multiplier"},
		MinCount: 2,
		Description: "Phòng ngự kết hợp sinh lực tạo thành lá chắn tự hồi phục. " +
			"Mỗi Cổ phụ trợ gia tăng khả năng tái tạo của khiên.",
		Transform: func(coreDef float64, boost float64, num int) (float64, string) {
			shieldVal := coreDef * (1.0 + float64(num)*0.15)
			regenPerTurn := shieldVal * 0.08
			if boost > 0 {
				shieldVal *= boost
				regenPerTurn *= boost
			}
			return shieldVal + regenPerTurn,
				fmt.Sprintf("Tạo khiên hộ thể %.0f điểm, tự hồi phục %.0f điểm mỗi lượt", shieldVal, regenPerTurn)
		},
	},
	{
		Name: "độc nhận phá thể", Type: "toxic_wound",
		Requires: []string{"damage", "poison"},
		Boosts:   []string{"damage_multiplier"},
		MinCount: 2,
		Description: "Sát thương tẩm độc ăn mòn đối thủ cả về thể xác lẫn chân nguyên. " +
			"Hiệu quả tăng theo số Cổ phụ trợ có thuộc tính độc.",
		Transform: func(coreDmg float64, boost float64, num int) (float64, string) {
			poisonDmg := coreDmg * 0.2 * (1.0 + float64(num)*0.1)
			bonusDmg := coreDmg * 0.15
			total := poisonDmg + bonusDmg
			if boost > 0 {
				total *= boost
			}
			return total,
				fmt.Sprintf("Vết thương nhiễm độc gây %.0f sát thương + %.0f độc tố mỗi lượt", bonusDmg, poisonDmg)
		},
	},
	{
		Name: "chấn động", Type: "concussive_blow",
		Requires: []string{"damage", "stun"},
		Boosts:   []string{"damage_multiplier"},
		MinCount: 2,
		Description: "Đòn tấn dung chấn động làm rối loạn thần hồn đối thủ. " +
			"Cổ phụ trợ gia tăng xác suất và thời gian choáng.",
		Transform: func(coreDmg float64, boost float64, num int) (float64, string) {
			stunChance := 20.0 + float64(num)*10.0
			if stunChance > 80.0 {
				stunChance = 80.0
			}
			bonusDmg := coreDmg * 0.05 * float64(num)
			if boost > 0 {
				bonusDmg *= boost
			}
			return bonusDmg,
				fmt.Sprintf("%.0f%% xác suất choáng đối thủ, gây thêm %.0f sát thương chấn động", stunChance, bonusDmg)
		},
	},
	{
		Name: "huyễn ảnh bộ", Type: "phantom_step",
		Requires: []string{"movement", "evasion", "stealth"},
		Boosts:   []string{"movement_bonus"},
		MinCount: 2,
		Description: "Thân pháp kết hợp ảo ảnh tạo ra bước di chuyển khó lường. " +
			"Tăng khả năng né tránh và tốc độ.",
		Transform: func(coreVal float64, boost float64, num int) (float64, string) {
			evasion := coreVal*(1.0+float64(num)*0.2) + float64(num)*3.0
			speed := coreVal * 0.5 * (1.0 + float64(num)*0.1)
			if boost > 0 {
				evasion *= boost
			}
			return evasion + speed,
				fmt.Sprintf("Tăng %.0f né tránh và %.0f tốc độ di chuyển", evasion, speed)
		},
	},
	{
		Name: "huyết lực", Type: "bloodthirsty_strength",
		Requires: []string{"damage", "blood_drain", "heal_based_on_strength"},
		Boosts:   []string{"damage_multiplier", "healing_multiplier"},
		MinCount: 2,
		Description: "Sát thương hút máu chuyển hóa thành sinh lực. " +
			"Càng nhiều Cổ huyết đạo, tỉ lệ hút máu càng cao.",
		Transform: func(coreDmg float64, boost float64, num int) (float64, string) {
			lifestealPct := 10.0 + float64(num)*5.0
			if lifestealPct > 40.0 {
				lifestealPct = 40.0
			}
			bonusDmg := coreDmg * 0.1 * float64(num)
			if boost > 0 {
				bonusDmg *= boost
				lifestealPct *= boost
			}
			return bonusDmg,
				fmt.Sprintf("Hút máu %.0f%% sát thương, gây thêm %.0f sát thương huyết lực", lifestealPct, bonusDmg)
		},
	},
	{
		Name: "hư ảnh đại lực", Type: "phantom_onslaught",
		Requires: []string{"summon_phantom", "damage", "add_base_attack"},
		Boosts:   []string{"damage_multiplier"},
		MinCount: 2,
		Description: "Triệu hồi hư ảnh chiến đấu cùng chủ nhân. " +
			"Mỗi Cổ phụ trợ gia tăng sức mạnh của hư ảnh.",
		Transform: func(coreVal float64, boost float64, num int) (float64, string) {
			phantomDmg := coreVal * 0.3 * (1.0 + float64(num)*0.15)
			if boost > 0 {
				phantomDmg *= boost
			}
			return phantomDmg,
				fmt.Sprintf("Triệu hồi hư ảnh gây thêm %.0f sát thương mỗi đòn", phantomDmg)
		},
	},
	{
		Name: "cuồng phong", Type: "chain_damage",
		Requires: []string{"damage", "damage_multiplier"},
		Boosts:   []string{"damage_multiplier"},
		MinCount: 2,
		Description: "Dồn dập sát thương liên hoàn. Mỗi Cổ phụ trợ công kích " +
			"gia tăng số lần đánh trong một lượt.",
		Transform: func(coreDmg float64, boost float64, num int) (float64, string) {
			hits := 1 + num/2
			if hits < 1 {
				hits = 1
			}
			totalDmg := coreDmg * float64(hits)
			if boost > 0 {
				totalDmg *= boost
			}
			return totalDmg,
				fmt.Sprintf("Đánh liên hoàn %d lần, tổng sát thương %.0f", hits, totalDmg)
		},
	},
	{
		Name: "phản thương kích", Type: "counter_thorns",
		Requires: []string{"add_base_defense", "defense", "damage"},
		Boosts:   []string{"defense_multiplier"},
		MinCount: 2,
		Description: "Phòng thủ phản đòn — khi bị tấn công, gây sát thương ngược lại đối thủ. " +
			"Mỗi Cổ phụ trợ gia tăng sát thương phản đòn.",
		Transform: func(coreDef float64, boost float64, num int) (float64, string) {
			thorns := coreDef * 0.2 * (1.0 + float64(num)*0.1)
			if boost > 0 {
				thorns *= boost
			}
			return thorns,
				fmt.Sprintf("Phản đòn %.0f sát thương lên kẻ tấn công", thorns)
		},
	},
	{
		Name: "tạo hóa chi lực", Type: "creation_power",
		Requires: []string{"generate_gu", "replicate", "wood_transformation"},
		Boosts:   []string{"utility_bonus"},
		MinCount: 1,
		Description: "Sức mạnh tạo hóa từ Cổ sinh sôi. Tăng cường hiệu quả hỗ trợ " +
			"và khả năng sinh tồn trong chiến đấu.",
		Transform: func(coreVal float64, boost float64, num int) (float64, string) {
			genPower := coreVal * (1.0 + float64(num)*0.25)
			if boost > 0 {
				genPower *= boost
			}
			return genPower,
				fmt.Sprintf("Gia tăng %.0f%% hiệu quả tạo sinh", genPower*100)
		},
	},
}

// SynergyRecipe defines a named killer move recipe from specific Gu combinations.
type SynergyRecipe struct {
	Name           string
	CoreGuIDs      []string
	SupportGuIDs   []string
	Description    string
	BonusDamage    float64
	BonusStability float64
	SpecialEffect  string
}

var synergyRecipes = []SynergyRecipe{
	{
		Name:           "Tứ Hải Tửu Trận",
		CoreGuIDs:      []string{"GU_LIQUOR_WORM"},
		SupportGuIDs:   []string{"GU_FOUR_FLAVORS_LIQUOR", "GU_SEVEN_FRAGRANCES_LIQUOR", "GU_NINE_EYES_LIQUOR"},
		Description:    "Tửu đạo chân truyền tứ đại tửu trùng phối hợp tinh luyện chân nguyên đến cực hạn.",
		BonusDamage:    1.0,
		BonusStability: 20,
		SpecialEffect:  "tửu trận — giảm mana cost 50%, tăng hiệu quả tinh luyện",
	},
	{
		Name:           "Đại Lực Thần Quyền",
		CoreGuIDs:      []string{"GU_CHARGING_CRASH"},
		SupportGuIDs:   []string{"GU_TEN_JUN_STRENGTH", "GU_JUN_STRENGTH", "GU_SELF_STRENGTH"},
		Description:    "Hội tụ khí lực ngàn quân đấm ra một quyền phá núi.",
		BonusDamage:    2.0,
		BonusStability: 10,
		SpecialEffect:  "xuyên phá — sát thương xuyên giáp 50%",
	},
	{
		Name:           "Âm Dương Lưỡng Nghi",
		CoreGuIDs:      []string{"GU_MOONLIGHT"},
		SupportGuIDs:   []string{"GU_LITTLE_LIGHT"},
		Description:    "Nguyệt quang và nhật quang hòa quyện tạo thành đòn tấn công âm dương.",
		BonusDamage:    1.8,
		BonusStability: 12,
		SpecialEffect:  "lưỡng nghi — sát thương gấp đôi nếu đối thủ là Cổ Sư âm hoặc dương thuần chất",
	},
	{
		Name:           "Huyết Sát Trận",
		CoreGuIDs:      []string{"GU_BLOOD_MOON"},
		SupportGuIDs:   []string{"GU_BLOOD_ASSET", "GU_BLOOD_SKULL"},
		Description:    "Huyết đạo sát trận dùng máu tế tăng uy lực sát chiêu.",
		BonusDamage:    2.2,
		BonusStability: 8,
		SpecialEffect:  "huyết tế — tiêu hao 10% HP hiện tại, tăng 50% sát thương",
	},
	{
		Name:           "Huyền Ảnh Sát",
		CoreGuIDs:      []string{"GU_BLOOD_MOON"},
		SupportGuIDs:   []string{"GU_SHADOW", "GU_SHADOW_STEPS"},
		Description:    "Huyết nguyệt hòa với bóng tối tạo ra đòn tấn công không thể né tránh.",
		BonusDamage:    1.6,
		BonusStability: 10,
		SpecialEffect:  "huyền ảnh — tấn công từ bóng tối, đối thủ không thể né tránh",
	},
	{
		Name:           "Ngũ Hành Sơn Hà",
		CoreGuIDs:      []string{"GU_CANOPY"},
		SupportGuIDs:   []string{"GU_IRON_BONE", "GU_EARTH_WORM"},
		Description:    "Ngưng tụ sơn hà chi lực bảo hộ Cổ Sư không thể phá vỡ.",
		BonusDamage:    0.5,
		BonusStability: 18,
		SpecialEffect:  "sơn hà — phòng ngự tăng 100%, chống mọi hiệu ứng khống chế",
	},
}

// detectCompounds checks which compound rules are triggered by the combined effects.
func detectCompounds(coreEffects map[string]float64, supportEffects []map[string]float64, activeEffects map[string]float64) []CompoundEffectInfo {
	allKeys := map[string]float64{}
	for k, v := range coreEffects {
		allKeys[k] += v
	}
	for _, se := range supportEffects {
		for k, v := range se {
			allKeys[k] += v
		}
		// also check for damage/attack keywords in support base_damage_bonus
		if baseDmg, ok := se["base_damage_bonus"]; ok && baseDmg > 0 {
			allKeys["damage"] += baseDmg
		}
		if baseDef, ok := se["base_defense_bonus"]; ok && baseDef > 0 {
			allKeys["defense"] += baseDef
		}
	}
	// also include compounded/active effects
	for k, v := range activeEffects {
		if v > 0 {
			allKeys[k] += v
		}
	}

	var compounds []CompoundEffectInfo
	usedKeys := map[string]bool{}

	for _, rule := range compoundRules {
		matchCount := 0
		for _, req := range rule.Requires {
			if v, ok := allKeys[req]; ok && v > 0 {
				matchCount++
			}
		}
		if matchCount < rule.MinCount {
			continue
		}

		totalBoost := 1.0
		for _, b := range rule.Boosts {
			if v, ok := allKeys[b]; ok && v > 0 {
				totalBoost *= v
			}
		}

		numMatched := 0
		for range rule.Requires {
			numMatched++
		}
		if numMatched < 1 {
			numMatched = 1
		}
		numSupports := len(supportEffects)
		coreVal := 1.0
		for _, req := range rule.Requires {
			if v, ok := allKeys[req]; ok && v > coreVal {
				coreVal = v
			}
		}
		if coreVal < 1.0 {
			coreVal = float64(len(rule.Requires))
		}

		val, desc := rule.Transform(coreVal, totalBoost, numSupports)
		if val <= 0 {
			continue
		}

		sources := []string{}
		for _, req := range rule.Requires {
			if v, ok := allKeys[req]; ok && v > 0 {
				sources = append(sources, req)
				usedKeys[req] = true
			}
		}

		compounds = append(compounds, CompoundEffectInfo{
			Name:        rule.Name,
			Type:        rule.Type,
			Value:       math.Round(val*100) / 100,
			Description: desc,
			Sources:     sources,
		})
	}

	return compounds
}

func detectSynergy(coreSpec *models.GuDictionaryItem, supportSpecs []*models.GuDictionaryItem) ([]CompoundEffectInfo, float64, float64, string) {
	if coreSpec == nil {
		return nil, 0, 0, ""
	}

	supportIDs := map[string]bool{}
	for _, s := range supportSpecs {
		supportIDs[s.ID] = true
	}

	for _, recipe := range synergyRecipes {
		coreMatch := false
		for _, cid := range recipe.CoreGuIDs {
			if strings.EqualFold(coreSpec.ID, cid) {
				coreMatch = true
				break
			}
		}
		if !coreMatch {
			continue
		}

		allFound := true
		for _, sid := range recipe.SupportGuIDs {
			if !supportIDs[sid] {
				allFound = false
				break
			}
		}
		if !allFound {
			continue
		}

		return []CompoundEffectInfo{
			{
				Name:        recipe.Name,
				Type:        "synergy",
				Value:       recipe.BonusDamage,
				Description: recipe.SpecialEffect,
				Sources:     append([]string{coreSpec.ID}, recipe.SupportGuIDs...),
			},
		}, recipe.BonusDamage, recipe.BonusStability, recipe.SpecialEffect
	}

	return nil, 0, 0, ""
}

// applyCompoundEffects takes base damage/defense/effects and applies compounding transforms.
// Returns (finalDamage, finalDefense, compoundedEffects map).
func applyCompoundEffects(baseDamage, baseDefense float64, effects map[string]float64, compounds []CompoundEffectInfo, damageMultiplier, defenseMultiplier float64) (float64, float64, map[string]float64) {
	if effects == nil {
		effects = map[string]float64{}
	}

	finalEffects := make(map[string]float64)
	for k, v := range effects {
		finalEffects[k] = v
	}

	finalDamage := baseDamage
	finalDefense := baseDefense

	for _, c := range compounds {
		switch c.Type {
		case "hemorrhage":
			finalDamage += c.Value * 0.3
			finalEffects["hemorrhage"] = c.Value * 0.7
			finalEffects["bleed"] = c.Value * 0.5
		case "regenerative_shield":
			finalDefense += c.Value * 0.6
			finalEffects["regen_per_turn"] = c.Value * 0.4
		case "toxic_wound":
			finalDamage += c.Value * 0.5
			finalEffects["poison"] = c.Value * 0.5
		case "concussive_blow":
			finalDamage += c.Value
			finalEffects["stun"] = c.Value * 0.1
		case "phantom_step":
			finalEffects["evasion"] = c.Value * 0.6
			finalEffects["speed_boost"] = c.Value * 0.4
		case "bloodthirsty_strength":
			finalDamage += c.Value
			finalEffects["life_steal"] = c.Value * 0.1
		case "phantom_onslaught":
			finalDamage += c.Value
			finalEffects["phantom_attack"] = c.Value
		case "chain_damage":
			finalDamage += c.Value * 0.5
			finalEffects["multi_hit"] = c.Value / finalDamage
		case "counter_thorns":
			finalDefense += c.Value * 0.3
			finalEffects["thorns_damage"] = c.Value
		case "creation_power":
			finalEffects["generation_boost"] = c.Value
		case "synergy":
			finalDamage *= 1.0 + c.Value*0.1
			finalEffects[c.Type+"_synergy"] = c.Value
		}
	}

	if damageMultiplier > 0 {
		finalDamage *= damageMultiplier
	}
	if defenseMultiplier > 0 {
		finalDefense *= defenseMultiplier
	}

	return math.Round(finalDamage*100) / 100, math.Round(finalDefense*100) / 100, finalEffects
}
