package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

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
	masteryMultiplier := 1.0
	if player.PathMasteries != nil {
		for _, path := range paths {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			mLevel, ok := player.PathMasteries[cleanPath]
			if !ok {
				mLevel, ok = player.PathMasteries[path]
			}
			if ok {
				mult := 1.0
				switch mLevel {
				case "Ordinary":
					mult = 1.0
				case "Master":
					mult = 1.1
				case "Grandmaster":
					mult = 1.25
				case "Supreme Grandmaster":
					mult = 1.5
				}
				if mult > masteryMultiplier {
					masteryMultiplier = mult
				}
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
	if active.GenerateGu {
		action.Logs = append(action.Logs, fmt.Sprintf("%s bắt đầu ngưng tụ Sinh Cơ Diệp.", spec.Name))
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

// ── OUT-OF-COMBAT EVENT HANDLERS ────────────────────────────────────────

func HandleConsumeEvent(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem, mechanic models.SpecialMechanic) (string, error) {
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
		if player.MaxMana < 20 { player.MaxMana = 20 }
		if player.CurrentMana > player.MaxMana { player.CurrentMana = player.MaxMana }
		return fmt.Sprintf("%s đẩy tu vi lên Đỉnh Phong nhưng làm Không khiếu hóa đá.", spec.Name), nil
	case "GIVE_STONES":
		val, _ := params["value"].(float64)
		amount := int(val)
		if amount == 0 { amount = 500 }
		player.PrimevalStones += amount
		return fmt.Sprintf("Nhận ngay %d Nguyên Thạch.", amount), nil
	case "SOUL_BOOST":
		val, _ := params["value"].(float64)
		if player.DaoMarks == nil { player.DaoMarks = make(models.DaoMarksMap) }
		player.DaoMarks["soul"] += int(val)
		return fmt.Sprintf("Hồn phách của đạo hữu được tráng kiện (+%.0f điểm Hồn lực).", val), nil
	case "FORCE_RANK_UP":
		player.SubRank = 1
		player.Rank++
		return fmt.Sprintf("Đột phá đại cảnh giới lên %s Sơ Giai!", getRankName(player.Rank)), nil
	case "MANA_POLLUTION":
		player.MaxMana = math.Round(player.MaxMana * 0.90) // lose 10%
		if player.CurrentMana > player.MaxMana { player.CurrentMana = player.MaxMana }
		return "Chân Nguyên bị ô nhiễm nặng nề (giảm 10% Chân Nguyên tối đa)!", nil
	case "ENSLAVE_BEAST_TIER":
		val, _ := params["value"].(float64)
		return fmt.Sprintf("Luyện hóa thành công, thu phục thú nô cấp %.0f.", val), nil
	case "ADD_LIFESPAN_YEARS":
		val, _ := params["value"].(float64)
		years := val
		player.Lifespan += years
		player.MaxLifespan += years
		return fmt.Sprintf("Cổ trùng Thọ Nguyên hòa tan vào không khiếu! Đạo hữu được ban thêm vĩnh viễn %.0f năm thọ mệnh.", years), nil
	default:
		return "", fmt.Errorf("Chưa hỗ trợ action code: %s", actionCode)
	}
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
				if player.ActiveCombat.EnemyHP < 0 {
					player.ActiveCombat.EnemyHP = 0
				}
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
				if player.HP < 0 {
					player.HP = 0
				}
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
