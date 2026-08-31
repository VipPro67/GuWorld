package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
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

// ── HELPERS ───────────────────────────────────────────────────────────

func getPathMasteryMultiplier(player *models.Player, paths []string) float64 {
	return gamefx.PathMasteryMultiplier(player.PathMasteries, paths)
}

func GetPortalManaCost(player *models.Player, spec *models.GuDictionaryItem) float64 {
	baseCost := spec.BaseManaCost
	if baseCost <= 0 {
		baseCost = 10.0 // default fallback
	}

	var computedBase float64
	if player.Rank >= 6 {
		// Immortal: flat cost in Tiên Nguyên
		computedBase = baseCost
	} else {
		// Mortal: % of MaxMana in Chân Nguyên
		computedBase = (baseCost / 100.0) * player.MaxMana
	}

	if spec.Rank == player.Rank {
		return computedBase
	}
	// spec.Rank > player.Rank
	diff := spec.Rank - player.Rank
	return computedBase * math.Pow(2, float64(diff))
}

func ValidateGuUsable(player *models.Player, gu *models.GuInstance, spec *models.GuDictionaryItem, actionRequiresEquipped bool) error {
	if gu.Status == "DEAD" {
		return fmt.Errorf("Không thể sử dụng Cổ trùng ĐÃ CHẾT: %s", spec.Name)
	}

	if actionRequiresEquipped && !gu.Equipped {
		return fmt.Errorf("Cổ trùng chưa được trang bị: %s", spec.Name)
	}

	// Mortal users cannot use immortal Gu that cost mana.
	if !gamefx.CanUseGuWithMana(player.Rank, spec) {
		return fmt.Errorf("Tu vi %s chưa đủ để sử dụng Tiên Cổ %s (Rank %d) có hao phí!", getRankName(player.Rank), spec.Name, spec.Rank)
	}

	// Rule 3: Energy cost
	cost := gamefx.EnergyCost(player, spec)
	if spec.Effects.Active != nil && (spec.Effects.Active.PortalCreate || spec.Effects.Active.RandomPortal) {
		if spec.Rank < player.Rank {
			return fmt.Errorf("Không thể thực hiện dịch chuyển: Cấp độ Cổ trùng (%d) thấp hơn tu vi bản thân (%d)!", spec.Rank, player.Rank)
		}
		cost = GetPortalManaCost(player, spec)
	}

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
	dmg, def, cost, err := CalculateGuEffectForCombatant(ctx.Attacker, &spec, true)
	if err != nil {
		return Action{}, err
	}

	action := Action{
		Name:     spec.Name,
		ManaCost: cost,
	}

	if spec.Effects.Active == nil {
		return action, nil
	}

	active := spec.Effects.Active

	if active.DamageFlat > 0 {
		action.Dmg += dmg
	}
	if active.ShieldFlat > 0 {
		action.Def += def
	}

	if active.Range > 0 {
		action.Logs = append(action.Logs, fmt.Sprintf("%s kéo dài khoảng cách tấn công lên %.0f thước.", spec.Name, active.Range))
	}
	if active.CurveAttack {
		action.Logs = append(action.Logs, fmt.Sprintf("%s uốn lượn đường bay tấn công vòng qua phòng ngự trực diện.", spec.Name))
	}
	if active.ManaRegen > 0 {
		ctx.Attacker.CurrentMana += active.ManaRegen
		if ctx.Attacker.CurrentMana > ctx.Attacker.MaxMana {
			ctx.Attacker.CurrentMana = ctx.Attacker.MaxMana
		}
		action.IsUtility = true
		action.Logs = append(action.Logs, fmt.Sprintf("%s hồi phục %.1f Chân Nguyên.", spec.Name, active.ManaRegen))
	}
	if active.WoodTransformation {
		action.Def += 30.0
		action.Logs = append(action.Logs, fmt.Sprintf("%s hóa thân thành yêu tinh cây, tăng thêm phòng ngự (+30.0 Thủ).", spec.Name))
	}
	if active.BloodDrain {
		action.LifeStealRate += 0.5
		action.Logs = append(action.Logs, fmt.Sprintf("%s kích hoạt cơ chế hút máu từ địch.", spec.Name))
	}
	if active.Replicate {
		action.Logs = append(action.Logs, fmt.Sprintf("%s sinh sản nhân bản ma cổ.", spec.Name))
	}
	if active.ApplyStatus == "BLEED" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "bleed",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây trạng thái xuất huyết (trừ %.1f HP mỗi lượt).", spec.Name, active.StatusTickDamage))
	}
	if active.ApplyStatus == "POISON" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "poison",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây trạng thái trúng độc (trừ %.1f HP mỗi lượt).", spec.Name, active.StatusTickDamage))
	}
	if active.ApplyStatus == "STUN" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "stun",
			Value: 1,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây choáng (%d lượt).", spec.Name, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "SLOW" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "slow",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây chậm (%.0f%% speed, %d lượt).", spec.Name, active.StatusTickDamage, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "SLOW1" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "slow1",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây chậm (%.0f%% speed, %d lượt, không thể kháng).", spec.Name, active.StatusTickDamage, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "SLOW_DAMAGE" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "slow_damage",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây chậm sát thương (%.0f%% speed và trừ %.1f HP mỗi lượt, %d lượt).", spec.Name, active.StatusTickDamage, active.StatusTickDamage, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "WEAK" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "weak",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây yếu (%.0f%% giảm dmg, %d lượt).", spec.Name, active.StatusTickDamage, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "FREEZE" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "freeze",
			Value: 1,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây đóng băng (%d lượt).", spec.Name, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "SILENCE" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "silence",
			Value: 1,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s gây im lặng (%d lượt).", spec.Name, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "THORNS" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "thorns_buff",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s nhận Khiên Gai (phản %.0f%% sát thương, %d lượt).", spec.Name, active.StatusTickDamage, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "REGEN" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "regen",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s nhận Hồi Sinh (%.0f%% HP max/lượt, %d lượt).", spec.Name, active.StatusTickDamage, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "ATK_UP" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "atk_up",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s nhận +%.0f%% Công Kích (%d lượt).", spec.Name, active.StatusTickDamage, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "DEF_UP" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "def_up",
			Value: active.StatusTickDamage,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s nhận +%.0f%% Phòng Thủ (%d lượt).", spec.Name, active.StatusTickDamage, active.StatusDurationTicks))
	}
	if active.ApplyStatus == "TAUNT" {
		action.StatusEffects = append(action.StatusEffects, models.StatusEffect{
			Type:  "taunt",
			Value: 1,
			Turns: active.StatusDurationTicks,
		})
		action.Logs = append(action.Logs, fmt.Sprintf("%s khiêu khích (%d lượt).", spec.Name, active.StatusDurationTicks))
	}
	if active.LifeStealRate > 0 {
		action.LifeStealRate += active.LifeStealRate
	}
	if active.RefineManaTier > 0 {
		boost := active.RefineManaTier * 0.20
		action.Dmg += action.Dmg * boost
		action.IsUtility = true
		action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s nâng phẩm Chân Nguyên (+%.0f%% Sát thương).", spec.Name, active.RefineManaTier*20))
	}
	if active.DamageMult > 1.0 {
		action.Dmg *= active.DamageMult
		action.IsUtility = true
		action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s, tăng uy lực (x%.0f sát thương).", spec.Name, active.DamageMult))
	}
	if active.CleansingPower > 0 {
		action.IsUtility = true
		action.Logs = append(action.Logs, fmt.Sprintf("Kích hoạt %s! Thanh tẩy dị lực và loại bỏ tạp chất trong Không khiếu.", spec.Name))
	}
	if active.SummonPhantom {
		action.Dmg += ctx.Attacker.BaseAttack
		action.Logs = append(action.Logs, fmt.Sprintf("%s triệu hồi thú lực hư ảnh, gây %.1f sát thương lực đạo.", spec.Name, ctx.Attacker.BaseAttack))
	}
	if active.SummonGrowthPhantoms {
		fg := ctx.Attacker.FullyGrown
		if len(fg) > 0 && active.PhantomDamageScale > 0 {
			count := int(active.SummonCountPerTick * float64(spec.Rank))
			if count <= 0 {
				count = 1
			}
			if count > len(fg) {
				count = len(fg)
			}
			totalPhantomDmg := 0.0
			for i := 0; i < count; i++ {
				gu := fg[i]
				baseStat := gu.Attack
				if baseStat <= 0 {
					baseStat = gu.HP * 0.1 // fallback: use HP/10 as base
				}
				dmg := baseStat * active.PhantomDamageScale
				totalPhantomDmg += dmg
				action.Logs = append(action.Logs, fmt.Sprintf("Hư ảnh của %s tấn công gây %.1f sát thương.", gu.GuID, dmg))
			}
			action.Dmg += totalPhantomDmg
			action.Logs = append(action.Logs, fmt.Sprintf("%s triệu hồi %d hư ảnh từ các cổ đã khai thác, tổng sát thương %.1f.", spec.Name, count, totalPhantomDmg))
		}
	}
	if active.GenerateGu {
		action.Logs = append(action.Logs, fmt.Sprintf("%s bắt đầu ngưng tụ Sinh Cơ Diệp.", spec.Name))
	}

	return action, nil
}

func ApplyPostDamageEffects(ctx *CombatContext, playerAction *Action, enemyAction *Action, playerHit bool, enemyHit bool, playerNetDmg float64, enemyNetDmg float64) {
	// 1. Lifesteal – heal HP first, excess becomes VirtualShield
	if playerHit && playerNetDmg > 0 && playerAction.LifeStealRate > 0 {
		heal := playerNetDmg * playerAction.LifeStealRate
		hpNeeded := ctx.Attacker.MaxHP - ctx.Attacker.HP
		if heal <= hpNeeded {
			ctx.Attacker.HP += heal
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu hút máu địch, hồi phục %.1f HP.", heal))
		} else {
			ctx.Attacker.HP = ctx.Attacker.MaxHP
			excess := heal - hpNeeded
			ctx.Attacker.VirtualShield += excess
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu hút máu địch (%.1f), HP đầy, chuyển %.1f thành khiên hộ thể.", heal, excess))
		}
	}

	if enemyHit && enemyNetDmg > 0 && enemyAction.LifeStealRate > 0 {
		heal := enemyNetDmg * enemyAction.LifeStealRate
		hpNeeded := ctx.Defender.MaxHP - ctx.Defender.HP
		if heal <= hpNeeded {
			ctx.Defender.HP += heal
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Kẻ địch hút máu đối phương, hồi phục %.1f HP.", heal))
		} else {
			ctx.Defender.HP = ctx.Defender.MaxHP
			excess := heal - hpNeeded
			ctx.Defender.VirtualShield += excess
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Kẻ địch hút máu đối phương (%.1f), HP đầy, chuyển %.1f thành khiên hộ thể.", heal, excess))
		}
	}

	// 2. Status Effects
	if playerHit && playerNetDmg > 0 && len(playerAction.StatusEffects) > 0 {
		ctx.Defender.StatusEffects = append(ctx.Defender.StatusEffects, playerAction.StatusEffects...)
	}

	if enemyHit && enemyNetDmg > 0 && len(enemyAction.StatusEffects) > 0 {
		ctx.Defender.StatusEffects = append(ctx.Defender.StatusEffects, enemyAction.StatusEffects...)
	}
}

// ── GENERIC EVENT DISPATCHER ─────────────────────────────────────────────

func HandleGuEvent(player *models.Player, specs models.GuDictionaryItem, guInstance models.GuInstance, eventTrigger string) (string, error) {
	if specs.Effects.Other == nil {
		return "", nil
	}
	var logs []string
	for _, mechanic := range specs.Effects.Other {
		if mechanic.EventTrigger == eventTrigger {
			msg, err := handleActionCode(player, specs, guInstance, mechanic)
			if err != nil {
				return "", err
			}
			if msg != "" {
				logs = append(logs, msg)
			}
		}
	}
	if len(logs) == 0 {
		return "", nil
	}
	return strings.Join(logs, " "), nil
}

func handleActionCode(player *models.Player, spec models.GuDictionaryItem, gu models.GuInstance, mechanic models.SpecialMechanic) (string, error) {
	var params map[string]interface{}
	if len(mechanic.Params) > 0 {
		_ = json.Unmarshal(mechanic.Params, &params)
	}
	actionCode := mechanic.ActionCode
	switch actionCode {
	case "INCREASE_APTITUDE":
		val, _ := params["value"].(float64)
		value := int(val)
		if value <= 0 {
			value = 1
		}
		player.AptitudePercent += value
		if player.AptitudePercent > 99 {
			player.AptitudePercent = 99
		}
		delta := float64(value)
		if delta > 0 {
			player.MaxMana += delta
			if player.CurrentMana > player.MaxMana {
				player.CurrentMana = player.MaxMana
			}
		}
		return fmt.Sprintf("Tăng tư chất Không khiếu lên %d%%.", player.AptitudePercent), nil
	case "SACRIFICE_APTITUDE":
		val, _ := params["value"].(float64)
		value := int(val)
		if value <= 0 {
			value = 5 // Default boost for blood skull
		}
		if player.AptitudePercent >= 99 {
			return "", fmt.Errorf("Tư chất đã đạt mức tối đa (99%%), không thể hiến tế thêm!")
		}
		player.AptitudePercent += value
		if player.AptitudePercent > 99 {
			player.AptitudePercent = 99
		}
		delta := float64(value)
		if delta > 0 {
			player.MaxMana += delta
			if player.CurrentMana > player.MaxMana {
				player.CurrentMana = player.MaxMana
			}
		}
		return fmt.Sprintf("Hiến tế huyết mạch thành công! Tư chất Không khiếu tăng lên %d%%.", player.AptitudePercent), nil
	case "RANK_PROGRESS":
		if spec.Rank != player.Rank {
			return "", fmt.Errorf("%s chỉ dùng cho %s", spec.Name, getRankName(spec.Rank))
		}
		if player.SubRank >= 4 {
			return "", fmt.Errorf("Xá Lợi Cổ không thể đột phá đại cảnh giới")
		}
		player.SubRank++
		stageNames := map[int]string{1: "Sơ Giai", 2: "Trung Giai", 3: "Cao Giai", 4: "Đỉnh Phong"}
		return fmt.Sprintf("Đột phá! Đạo hữu luyện hóa %s, tu vi nâng cao lên %s %s!", spec.Name, getRankName(player.Rank), stageNames[player.SubRank]), nil
	case "APERTURE_PETRIFICATION":
		player.SubRank = 4
		player.MaxMana -= 15
		if player.MaxMana < 20 {
			player.MaxMana = 20
		}
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		return fmt.Sprintf("%s đẩy tu vi lên Đỉnh Phong nhưng làm Không khiếu hóa đá.", spec.Name), nil
	case "GIVE_STONES":
		val, _ := params["value"].(float64)
		amount := val
		if amount == 0 {
			amount = 500
		}
		player.PrimevalStones += int64(amount)
		return fmt.Sprintf("Nhận ngay %d Nguyên Thạch.", int64(amount)), nil
	case "SOUL_BOOST":
		val, _ := params["value"].(float64)
		if player.DaoMarks == nil {
			player.DaoMarks = make(models.DaoMarksMap)
		}
		player.DaoMarks["soul"] += int(val)
		return fmt.Sprintf("Hồn phách của đạo hữu được tráng kiện (+%.0f điểm Hồn lực).", val), nil
	case "FORCE_RANK_UP":
		player.SubRank = 1
		player.Rank++
		return fmt.Sprintf("Đột phá đại cảnh giới lên %s Sơ Giai!", getRankName(player.Rank)), nil
	case "MANA_POLLUTION":
		player.MaxMana = math.Round(player.MaxMana * 0.90)
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		return "Chân Nguyên bị ô nhiễm nặng nề (giảm 10% Chân Nguyên tối đa)!", nil
	case "ENSLAVE_BEAST_TIER":
		val, _ := params["value"].(float64)
		return fmt.Sprintf("Luyện hóa thành công, thu phục thú nô cấp %.0f.", val), nil
	case "ADD_LIFESPAN_YEARS":
		val, _ := params["value"].(float64)
		years := val
		player.Lifespan += years
		return fmt.Sprintf("Cổ trùng Thọ Nguyên hòa tan vào không khiếu! Đạo hữu được ban thêm vĩnh viễn %.0f năm thọ mệnh.", years), nil
	case "ADD_SOUL_MAX":
		val, _ := params["value"].(float64)
		if player.MaxSoul == 0 {
			player.MaxSoul = 10
		}
		player.MaxSoul += int(val)
		return fmt.Sprintf("Linh Hồn tối đa tăng thêm %.0f điểm.", val), nil
	case "RESTORE_SOUL_HP":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 10
		}
		before := player.SoulHP
		player.SoulHP += int(val)
		if player.SoulHP > player.MaxSoul {
			player.SoulHP = player.MaxSoul
		}
		restored := player.SoulHP - before
		return fmt.Sprintf("Hồn phách được dưỡng hồi (+%d Hồn Phách).", restored), nil
	// ── New action codes ──────────────────────────────────────────────
	case "TIME_ACCELERATION":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 3.0
		}
		boost := player.MaxMana * 0.10 * val
		player.CurrentMana += boost
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		return fmt.Sprintf("%s gia tốc dòng chảy thời gian, hồi phục %.0f Chân Nguyên!", spec.Name, boost), nil
	case "RECORD":
		return fmt.Sprintf("%s ghi lại toàn bộ hình ảnh và âm thanh trong khu vực.", spec.Name), nil
	case "CONTRACT_OATH":
		return fmt.Sprintf("%s kết khế ước — hai bên không thể phản bội.", spec.Name), nil
	case "PORTAL_CREATE":
		return fmt.Sprintf("%s khắc ấn tọa độ không gian tại vị trí hiện tại.", spec.Name), nil
	case "REGENERATE_TERRAIN":
		return fmt.Sprintf("%s hồi phục địa hình xung quanh, xóa bỏ dấu vết chiến đấu.", spec.Name), nil
	case "HOT_TICK":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 5.0
		}
		player.HP += val
		if player.HP > gamefx.EffectiveMaxHP(player) {
			player.HP = gamefx.EffectiveMaxHP(player)
		}
		return "", nil
	case "GENERATE_STONES_TICK":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 1.0
		}
		player.PrimevalStones += int64(val)
		return "", nil
	case "MANA_REGEN_TICK":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 0.5
		}
		player.CurrentMana += val
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		return "", nil
	case "THOUGHT_REGEN_TICK":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 1.0
		}
		player.CurrentThought += val
		if player.CurrentThought > player.MaxThought {
			player.CurrentThought = player.MaxThought
		}
		return "", nil
	case "GENERATE_IMMORTAL_STONES_TICK":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 1.0
		}
		player.ImmortalStones += int64(val)
		return fmt.Sprintf("Sản xuất %d Tiên Nguyên Thạch.", int64(val)), nil
	case "REDUCE_STRESS":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 5.0
		}
		player.ApertureStress -= val
		if player.ApertureStress < 0 {
			player.ApertureStress = 0
		}
		return fmt.Sprintf("Giảm %.0f áp lực tinh thần.", val), nil
	case "RESTORE_MENTAL":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = player.MaxThought
		}
		before := player.CurrentThought
		player.CurrentThought += val
		if player.CurrentThought > player.MaxThought {
			player.CurrentThought = player.MaxThought
		}
		if player.CurrentThought < 0 {
			player.CurrentThought = 0
		}
		restored := player.CurrentThought - before
		return fmt.Sprintf("Hồi phục %.0f Niệm Lực.", restored), nil
	case "INCREASE_TIME_SPEED":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 0.1
		}
		player.TimeSpeedMultiplier += val
		return fmt.Sprintf("%s hòa tan vào không khiếu, dòng thời gian vĩnh viễn tăng +%.2f!", spec.Name, val), nil
	default:
		return "", fmt.Errorf("Chưa hỗ trợ action code: %s", actionCode)
	}
}

// ApplyGrowthProgress calculates and applies growth progress for a Gu instance.
// Returns the delta boosts applied.
func ApplyGrowthProgress(gu *models.GuInstance, spec *models.GuDictionaryItem, elapsedHours float64) (attack, defense, hp, speed float64) {
	if spec.Effects.Growth == nil || elapsedHours <= 0 {
		return 0, 0, 0, 0
	}
	g := spec.Effects.Growth
	maxHours := g.HoursToMax
	if maxHours <= 0 {
		maxHours = 168 // default 1 week
	}
	oldHours := gu.GrowthHours
	newHours := oldHours + elapsedHours
	if newHours > maxHours {
		newHours = maxHours
	}
	progress := newHours / maxHours
	if progress > 1.0 {
		progress = 1.0
	}
	oldProgress := oldHours / maxHours

	attack = g.MaxAttack * (progress - oldProgress)
	defense = g.MaxDefense * (progress - oldProgress)
	hp = g.MaxHP * (progress - oldProgress)
	speed = g.MaxSpeed * (progress - oldProgress)

	gu.GrowthHours = newHours
	gu.EquippedAt = nil // reset after processing
	return
}

// HandleConsumeEvent is kept for backward compatibility, delegates to handleActionCode
func HandleConsumeEvent(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem, mechanic models.SpecialMechanic) (string, error) {
	return handleActionCode(player, spec, gu, mechanic)
}

func tickStatusEffects(player *models.Player, ctx *CombatContext) {
	if player.ActiveCombat == nil {
		return
	}

	// Tick enemy status effects
	var nextEnemyEffects []models.StatusEffect
	for _, se := range player.ActiveCombat.EnemyStatusEffects {
		if se.Turns > 0 {
			switch se.Type {
			case "bleed", "poison":
				player.ActiveCombat.EnemyHP -= se.Value
				if player.ActiveCombat.EnemyHP < 0 {
					player.ActiveCombat.EnemyHP = 0
				}
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Địch chịu %.1f sát thương từ hiệu ứng %s (còn %d lượt).", se.Value, se.Type, se.Turns-1))
			case "regen":
				player.ActiveCombat.EnemyHP += se.Value
				if player.ActiveCombat.EnemyHP > player.ActiveCombat.EnemyMaxHP {
					player.ActiveCombat.EnemyHP = player.ActiveCombat.EnemyMaxHP
				}
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Địch hồi %.1f HP từ Hồi Sinh (còn %d lượt).", se.Value, se.Turns-1))
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
			switch se.Type {
			case "bleed", "poison":
				player.HP -= se.Value
				if player.HP < 0 {
					player.HP = 0
				}
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu chịu %.1f sát thương từ hiệu ứng %s (còn %d lượt).", se.Value, se.Type, se.Turns-1))
			case "regen":
				player.HP += se.Value
				if player.HP > gamefx.EffectiveMaxHP(player) {
					player.HP = gamefx.EffectiveMaxHP(player)
				}
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("-> Đạo hữu hồi %.1f HP từ Hồi Sinh (còn %d lượt).", se.Value, se.Turns-1))
			}
			se.Turns--
			if se.Turns > 0 {
				nextPlayerEffects = append(nextPlayerEffects, se)
			}
		}
	}
	player.ActiveCombat.PlayerStatusEffects = nextPlayerEffects
}
