package handlers

import (
	"fmt"
	"math"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Action represents a dynamic combat action compiled from Gu effects
type Action struct {
	Name          string
	Dmg           float64
	Def           float64
	ManaCost      float64
	IsUtility     bool
	LifeStealRate float64
	StatusEffects []models.StatusEffect
	Logs          []string
}

// ── REGISTRIES ────────────────────────────────────────────────────────

type StatEffectHandler func(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64)
type CombatEffectHandler func(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error
type SupportEffectHandler func(player *models.Player, spec models.GuDictionaryItem) (string, error)
type ConsumableEffectHandler func(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error)

var StatEffectRegistry = map[string]StatEffectHandler{
	"add_base_attack":          addAttackStat,
	"add_base_defense":         addDefenseStat,
	"damage_multiplier_on_hit": addDamageOnHitStat,
	"soul_refinement":          addSoulStat,
	"storage":                  addStorageStat,
	"movement":                 addMovementStat,
	"investigation":            addInvestigationStat,
	"hearing":                  addHearingStat,
	"evasion":                  addEvasionStat,
	"refine_mana_tier":         addRefineManaTierStat,
	"cleansing_power":          addCleansingPowerStat,
}

var ActiveEffectRegistry = map[string]CombatEffectHandler{
	"damage":              handleDamageEffect,
	"add_base_attack":     handleDamageEffect,
	"add_base_defense":    handleDefenseEffect,
	"range":               handleRangeEffect,
	"curve_attack":        handleCurveAttackEffect,
	"mana_regen":          handleManaRegen,
	"wood_transformation": handleWoodTransformation,
	"blood_drain":         handleBloodDrain,
	"replicate":           handleReplicate,
	"bleed":               handleBleedEffect,
	"bleed_effect":        handleBleedEffect,
	"lifesteal":           handleLifeStealEffect,
	"refine_mana_tier":    handleRefineManaTier,
	"damage_multiplier":   handleDamageMultiplier,
	"cleansing_power":     handleCleansingPower,
	"summon_phantom":      handleSummonPhantom,
	"generate_gu":         handleGenerateGu,
}

var OutCombatSupportRegistry = map[string]SupportEffectHandler{
	"increase_aptitude": handleIncreaseAptitude,
	"share_cultivation": handleShareCultivation,
}

var ConsumableEffectRegistry = map[string]ConsumableEffectHandler{
	"rank_progress":          handleRankProgress,
	"raise_cultivation":      handleRankProgress,
	"aperture_petrification": handleAperturePetrification,
	"give_stones":            handleGiveStones,
	"soul_boost":             handleSoulBoost,
	"force_rank_up":          handleForceRankUp,
	"mana_pollution":         handleManaPollution,
	"enslave_beast_tier":     handleEnslaveBeastTier,
}

// ── HELPERS ───────────────────────────────────────────────────────────

func getPathMasteryMultiplier(player *models.Player, path string) float64 {
	masteryMultiplier := 1.0
	if player.PathMasteries != nil {
		if mLevel, ok := player.PathMasteries[path]; ok {
			switch mLevel {
			case "Ordinary":
				masteryMultiplier = 1.0
			case "Master":
				masteryMultiplier = 1.1
			case "Grandmaster":
				masteryMultiplier = 1.25
			case "Supreme Grandmaster":
				masteryMultiplier = 1.5
			}
		}
	}
	return masteryMultiplier
}

func ValidateGuUsable(player *models.Player, gu *models.GuInstance, spec *models.GuDictionaryItem, actionRequiresEquipped bool) error {
	if gu.Status == "DEAD" {
		return fmt.Errorf("Không thể sử dụng Cổ trùng ĐÃ CHẾT: %s", spec.Name)
	}

	if actionRequiresEquipped && !gu.Equipped {
		return fmt.Errorf("Cổ trùng chưa được trang bị: %s", spec.Name)
	}

	// Rule 1: Level Penalty (Backlash)
	if player.Rank < spec.Rank {
		player.HP -= 50
		if player.HP < 0 {
			player.HP = 0
		}
		return fmt.Errorf("Backlash: Cổ trùng Rank %d phản phệ Không Khiếu Rank %d! (Trừ 50 HP)", spec.Rank, player.Rank)
	}

	// Rule 3: Energy cost
	cost := calculateEnergyCost(player, spec)
	if player.CurrentMana < cost {
		return fmt.Errorf("%s", insufficientEnergyMessage(player))
	}

	return nil
}

func RemoveGuFromInventory(player *models.Player, instanceID string) {
	newInventory := models.GuInventory{}
	for _, gu := range player.GuInventory {
		if gu.InstanceID != instanceID {
			newInventory = append(newInventory, gu)
		}
	}
	player.GuInventory = newInventory
	RebuildEquippedGus(player)
}

func RebuildEquippedGus(player *models.Player) {
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}
}

func BuildCombatAction(ctx *CombatContext, spec models.GuDictionaryItem) (Action, error) {
	dmg, def, cost, err := CalculateGuEffectForCombatant(ctx.Attacker, &spec)
	if err != nil {
		return Action{}, err
	}

	action := Action{
		Name:     spec.Name,
		ManaCost: cost,
	}

	activeEffects := models.GetGuEffectSection(spec, "active")
	for effectKey := range activeEffects {
		if effectKey == "damage" || effectKey == "add_base_attack" {
			action.Dmg += dmg
		} else if effectKey == "defense" || effectKey == "add_base_defense" {
			action.Def += def
		} else if handler, ok := ActiveEffectRegistry[effectKey]; ok {
			if err := handler(ctx, &spec, &action); err != nil {
				return Action{}, err
			}
		}
	}

	return action, nil
}

func ApplyPostDamageEffects(ctx *CombatContext, playerAction *Action, enemyAction *Action, playerHit bool, enemyHit bool, playerNetDmg float64, enemyNetDmg float64) {
	// 1. Lifesteal
	if playerHit && playerNetDmg > 0 && playerAction.LifeStealRate > 0 {
		heal := playerNetDmg * playerAction.LifeStealRate
		ctx.Attacker.HP += heal
		if ctx.Attacker.HP > ctx.Attacker.MaxHP {
			ctx.Attacker.HP = ctx.Attacker.MaxHP
		}
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu hút máu địch, hồi phục %.1f HP.", heal))
	}

	if enemyHit && enemyNetDmg > 0 && enemyAction.LifeStealRate > 0 {
		heal := enemyNetDmg * enemyAction.LifeStealRate
		ctx.Defender.HP += heal
		if ctx.Defender.HP > ctx.Defender.MaxHP {
			ctx.Defender.HP = ctx.Defender.MaxHP
		}
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Kẻ địch hút máu đối phương, hồi phục %.1f HP.", heal))
	}

	// 2. Status Effects
	if playerHit && playerNetDmg > 0 && len(playerAction.StatusEffects) > 0 {
		ctx.Defender.StatusEffects = append(ctx.Defender.StatusEffects, playerAction.StatusEffects...)
	}

	if enemyHit && enemyNetDmg > 0 && len(enemyAction.StatusEffects) > 0 {
		ctx.Defender.StatusEffects = append(ctx.Defender.StatusEffects, enemyAction.StatusEffects...)
	}
}

// ── PASSIVE EFFECT HANDLERS ──────────────────────────────────────────

func addAttackStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["attack"] += guSectionValue(*spec, "passive", "add_base_attack") * getPathMasteryMultiplier(player, spec.Path)
}

func addDefenseStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["defense"] += guSectionValue(*spec, "passive", "add_base_defense") * getPathMasteryMultiplier(player, spec.Path)
}

func addDamageOnHitStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	if player.BaseHP <= 0 {
		return
	}
	missingHpRate := 1.0 - player.HP/player.BaseHP
	stats["damage_multiplier"] += missingHpRate * guSectionValue(*spec, "passive", "damage_multiplier_on_hit")
}

func addSoulStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["soul"] += guSectionValue(*spec, "passive", "soul_refinement")
}

func addStorageStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["storage"] += guSectionValue(*spec, "passive", "storage")
}

func addMovementStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["movement"] += guSectionValue(*spec, "passive", "movement")
}

func addInvestigationStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["investigation"] += guSectionValue(*spec, "passive", "investigation")
}

func addHearingStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["hearing"] += guSectionValue(*spec, "passive", "hearing")
}

func addEvasionStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["evasion"] += guSectionValue(*spec, "passive", "evasion")
}

func addRefineManaTierStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["healing"] += guSectionValue(*spec, "passive", "refine_mana_tier") * 10.0
}

func addCleansingPowerStat(player *models.Player, spec *models.GuDictionaryItem, stats map[string]float64) {
	stats["healing"] += guSectionValue(*spec, "passive", "cleansing_power") * 15.0
}

// ── ACTIVE COMBAT EFFECT HANDLERS ─────────────────────────────────────

func handleDamageEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	dmg, _, _, err := CalculateGuEffectForCombatant(ctx.Attacker, spec)
	if err != nil {
		return err
	}
	action.Dmg += dmg
	return nil
}

func handleDefenseEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	_, def, _, err := CalculateGuEffectForCombatant(ctx.Attacker, spec)
	if err != nil {
		return err
	}
	action.Def += def
	return nil
}

func handleRangeEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	val := guSectionValue(*spec, "active", "range")
	action.Logs = append(action.Logs, fmt.Sprintf("%s kéo dài khoảng cách tấn công lên %.0f thước.", spec.Name, val))
	return nil
}

func handleCurveAttackEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.Logs = append(action.Logs, fmt.Sprintf("%s uốn lượn đường bay tấn công vòng qua phòng ngự trực diện.", spec.Name))
	return nil
}

func handleManaRegen(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	value := guSectionValue(*spec, "active", "mana_regen")
	ctx.Attacker.CurrentMana += value
	if ctx.Attacker.CurrentMana > ctx.Attacker.MaxMana {
		ctx.Attacker.CurrentMana = ctx.Attacker.MaxMana
	}
	action.IsUtility = true
	action.Logs = append(action.Logs, fmt.Sprintf("%s hồi phục %.1f Chân Nguyên.", spec.Name, value))
	return nil
}

func handleWoodTransformation(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.Def += 30.0
	action.Logs = append(action.Logs, fmt.Sprintf("%s hóa thân thành yêu tinh cây, tăng thêm phòng ngự (+30.0 Thủ).", spec.Name))
	return nil
}

func handleBloodDrain(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.LifeStealRate += 0.5
	action.Logs = append(action.Logs, fmt.Sprintf("%s kích hoạt cơ chế hút máu từ địch.", spec.Name))
	return nil
}

func handleReplicate(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.Logs = append(action.Logs, fmt.Sprintf("%s sinh sản nhân bản ma cổ.", spec.Name))
	return nil
}

func handleBleedEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	val := guSectionValue(*spec, "active", "bleed")
	if val == 0 {
		val = guSectionValue(*spec, "active", "bleed_effect")
	}
	action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
		Type:  "bleed",
		Value: val,
		Turns: 3,
	})
	action.Logs = append(action.Logs, fmt.Sprintf("%s gây trạng thái xuất huyết (trừ %.1f HP mỗi lượt).", spec.Name, val))
	return nil
}

func handleLifeStealEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.LifeStealRate += guSectionValue(*spec, "active", "lifesteal")
	return nil
}

func handleRefineManaTier(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	tier := guSectionValue(*spec, "active", "refine_mana_tier")
	boost := tier * 0.20
	ctx.DamageBoost += boost
	action.IsUtility = true
	action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s nâng phẩm Chân Nguyên (+%.0f%% Sát thương).", spec.Name, boost*100))
	return nil
}

func handleDamageMultiplier(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	mult := guSectionValue(*spec, "active", "damage_multiplier")
	boost := mult - 1.0
	ctx.DamageBoost += boost
	action.IsUtility = true
	action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s, tăng uy lực (+%.0f%% Sát thương).", spec.Name, boost*100))
	return nil
}

func handleCleansingPower(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.IsUtility = true
	action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s! Thanh tẩy dị lực và loại bỏ tạp chất trong Không khiếu.", spec.Name))
	return nil
}

func handleSummonPhantom(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.Dmg += ctx.Attacker.BaseAttack
	action.Logs = append(action.Logs, fmt.Sprintf("%s triệu hồi thú lực hư ảnh, gây %.1f sát thương lực đạo.", spec.Name, ctx.Attacker.BaseAttack))
	return nil
}

func handleGenerateGu(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.Logs = append(action.Logs, fmt.Sprintf("%s bắt đầu ngưng tụ Sinh Cơ Diệp.", spec.Name))
	return nil
}

// ── OUT-OF-COMBAT SUPPORT EFFECT HANDLERS ─────────────────────────────

func handleIncreaseAptitude(player *models.Player, spec models.GuDictionaryItem) (string, error) {
	value := int(guSectionValue(spec, "support", "increase_aptitude"))
	if value <= 0 {
		value = 1
	}

	player.AptitudePercent += value
	if player.AptitudePercent > 99 {
		player.AptitudePercent = 99
	}

	player.MaxMana = float64(player.AptitudePercent)
	if player.CurrentMana > player.MaxMana {
		player.CurrentMana = player.MaxMana
	}

	return fmt.Sprintf("Tăng tư chất Không khiếu lên %d%%.", player.AptitudePercent), nil
}

func handleShareCultivation(player *models.Player, spec models.GuDictionaryItem) (string, error) {
	return fmt.Sprintf("Kích hoạt %s, sẵn sàng chia sẻ tu vi.", spec.Name), nil
}

// ── CONSUMABLE EFFECT HANDLERS ────────────────────────────────────────

func handleRankProgress(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	if spec.Rank != player.Rank {
		return "", fmt.Errorf("%s chỉ dùng cho %s", spec.Name, getRankName(spec.Rank))
	}

	if player.SubRank >= 4 {
		return "", fmt.Errorf("Xá Lợi Cổ không thể đột phá đại cảnh giới")
	}

	player.SubRank++

	stageNames := map[int]string{
		1: "Sơ Giai",
		2: "Trung Giai",
		3: "Cao Giai",
		4: "Đỉnh Phong",
	}

	return fmt.Sprintf("Đột phá! Đạo hữu luyện hóa %s, tu vi nâng cao lên %s %s!", spec.Name, getRankName(player.Rank), stageNames[player.SubRank]), nil
}

func handleAperturePetrification(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	player.SubRank = 4
	player.MaxMana -= 15

	if player.MaxMana < 20 {
		player.MaxMana = 20
	}

	if player.CurrentMana > player.MaxMana {
		player.CurrentMana = player.MaxMana
	}

	return fmt.Sprintf("%s đẩy tu vi lên Đỉnh Phong nhưng làm Không khiếu hóa đá.", spec.Name), nil
}

func handleGiveStones(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	amount := int(guSectionValue(spec, "support", "give_stones"))
	if amount == 0 {
		amount = 500
	}
	player.PrimevalStones += amount
	return fmt.Sprintf("Nhận ngay %d Nguyên Thạch.", amount), nil
}

func handleSoulBoost(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	val := guSectionValue(spec, "support", "soul_boost")
	if player.DaoMarks == nil {
		player.DaoMarks = make(models.DaoMarksMap)
	}
	player.DaoMarks["soul"] += int(val)
	return fmt.Sprintf("Hồn phách của đạo hữu được tráng kiện (+%.0f điểm Hồn lực).", val), nil
}

func handleForceRankUp(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	player.SubRank = 1
	player.Rank++
	return fmt.Sprintf("Đột phá đại cảnh giới lên %s Sơ Giai!", getRankName(player.Rank)), nil
}

func handleManaPollution(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	player.MaxMana = math.Round(player.MaxMana * 0.90) // lose 10%
	if player.CurrentMana > player.MaxMana {
		player.CurrentMana = player.MaxMana
	}
	return "Chân Nguyên bị ô nhiễm nặng nề (giảm 10% Chân Nguyên tối đa)!", nil
}

func handleEnslaveBeastTier(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	tier := guSectionValue(spec, "support", "enslave_beast_tier")
	return fmt.Sprintf("Luyện hóa thành công, thu phục thú nô cấp %.0f.", tier), nil
}

func tickStatusEffects(player *models.Player, ctx *CombatContext) {
	if player.ActiveCombat == nil {
		return
	}

	// Tick enemy status effects
	var nextEnemyEffects []models.StatusEffect
	for _, se := range player.ActiveCombat.EnemyStatusEffects {
		if se.Turns > 0 {
			if se.Type == "bleed" || se.Type == "poison" {
				player.ActiveCombat.EnemyHP -= se.Value
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Địch chịu %.1f sát thương từ hiệu ứng %s (còn %d lượt).", se.Value, se.Type, se.Turns-1))
			}
			se.Turns--
			if se.Turns > 0 {
				nextEnemyEffects = append(nextEnemyEffects, se)
			}
		}
	}
	player.ActiveCombat.EnemyStatusEffects = nextEnemyEffects

	// Tick player status effects
	var nextPlayerEffects []models.StatusEffect
	for _, se := range player.ActiveCombat.PlayerStatusEffects {
		if se.Turns > 0 {
			if se.Type == "bleed" || se.Type == "poison" {
				player.HP -= se.Value
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu chịu %.1f sát thương từ hiệu ứng %s (còn %d lượt).", se.Value, se.Type, se.Turns-1))
			}
			se.Turns--
			if se.Turns > 0 {
				nextPlayerEffects = append(nextPlayerEffects, se)
			}
		}
	}
	player.ActiveCombat.PlayerStatusEffects = nextPlayerEffects
}

func guSectionValue(spec models.GuDictionaryItem, section string, key string) float64 {
	return models.GetGuEffectSection(spec, section)[key]
}
