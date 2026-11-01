package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// ── HELPERS ───────────────────────────────────────────────────────────

func GetPortalManaCost(player *models.Player, spec *models.GuDictionaryItem) float64 {
	baseCost := spec.BaseManaCost
	if baseCost <= 0 {
		baseCost = 10.0 // default fallback
	}

	// Flat cost in Chân Nguyên / Tiên Nguyên.
	computedBase := baseCost

	if spec.Rank == player.Rank {
		return computedBase
	}
	// spec.Rank > player.Rank
	diff := spec.Rank - player.Rank
	return computedBase * math.Pow(2, float64(diff))
}

// defaultGuTeleportRange maps a Cổ's rank to the maximum Chebyshev distance
// (world cells) a teleport may cover when the effect does not declare an
// explicit PORTAL_CREATE/RANDOM_PORTAL params.range. 0 means unlimited.
var defaultGuTeleportRange = map[int]int{
	1: 100,
	2: 200,
	3: 400,
	4: 800,
	5: 1000,
	6: 3000,
	7: 10000,
	8: 20000,
	9: 30000,
}

// GuTeleportRange resolves the maximum teleport distance for a Gu spec.
// An explicit params.range on PORTAL_CREATE/RANDOM_PORTAL is authoritative.
// Without one, IGNORE_MOVEMENT_RANGE means unlimited; otherwise the rank table
// applies. 0 means unlimited.
func GuTeleportRange(spec *models.GuDictionaryItem) int {
	if spec == nil {
		return 0
	}
	for _, code := range []string{"PORTAL_CREATE", "RANDOM_PORTAL"} {
		if value := int(models.GuSpecCapabilityFloat(*spec, code, "range")); value > 0 {
			return value
		}
	}
	if _, ok := models.GuSpecCapability(*spec, "IGNORE_MOVEMENT_RANGE"); ok {
		return 0
	}
	return defaultGuTeleportRange[spec.Rank]
}

func ValidateGuUsable(player *models.Player, gu *models.GuInstance, spec *models.GuDictionaryItem, actionRequiresEquipped bool) error {
	if gu.Status == "DEAD" {
		return fmt.Errorf("Không thể sử dụng Cổ trùng ĐÃ CHẾT: %s", spec.Name)
	}

	if gu.Hunger <= 0 {
		return fmt.Errorf("Cổ trùng %s đã kiệt sức (độ no = 0), cần cho ăn trước khi sử dụng!", spec.Name)
	}

	if actionRequiresEquipped && !gu.Equipped {
		return fmt.Errorf("Cổ trùng chưa được trang bị: %s", spec.Name)
	}

	// Mortal users cannot use immortal Gu that cost mana.
	if !gamefx.CanUseGuWithMana(player.Rank, spec) {
		return fmt.Errorf("Tu vi %s chưa đủ để sử dụng Tiên Cổ %s (Rank %d) có hao phí!", models.GetRankName(player.Rank), spec.Name, spec.Rank)
	}

	// Rule 3: Energy cost
	cost := gamefx.EnergyCost(player, spec)
	_, portal := models.GuSpecCapability(*spec, "PORTAL_CREATE")
	_, randomPortal := models.GuSpecCapability(*spec, "RANDOM_PORTAL")
	_, ignoreRange := models.GuSpecCapability(*spec, "IGNORE_MOVEMENT_RANGE")
	if portal || randomPortal || ignoreRange {
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
	player.GuInventory = models.RemoveGuByInstanceID(player.GuInventory, instanceID)
	RebuildEquippedGus(player)
}

func RebuildEquippedGus(player *models.Player) {
	active := models.FilterEquippedActive(player.GuInventory)
	activeMap := make(map[string]struct{}, len(active))
	for _, gu := range active {
		activeMap[gu.InstanceID] = struct{}{}
	}

	newEquipped := make([]string, 0, len(active))
	seen := make(map[string]struct{}, len(active))
	for _, id := range player.EquippedGus {
		if _, ok := activeMap[id]; ok {
			if _, already := seen[id]; !already {
				newEquipped = append(newEquipped, id)
				seen[id] = struct{}{}
			}
		}
	}
	for _, gu := range active {
		if _, already := seen[gu.InstanceID]; !already {
			newEquipped = append(newEquipped, gu.InstanceID)
			seen[gu.InstanceID] = struct{}{}
		}
	}
	player.EquippedGus = newEquipped
}

// ── GENERIC EVENT DISPATCHER ─────────────────────────────────────────────

func HandleGuEvent(player *models.Player, specs models.GuDictionaryItem, guInstance models.GuInstance, eventTrigger string) (string, error) {
	return HandleGuEventTx(nil, player, specs, guInstance, eventTrigger)
}

// HandleGuEventTx is HandleGuEvent with an optional transaction. Some action
// codes write to the world (e.g. SPRING_EGG_HATCH plants a primeval vein) and
// therefore need the caller's tx; handlers without one pass nil.
func HandleGuEventTx(tx *sql.Tx, player *models.Player, specs models.GuDictionaryItem, guInstance models.GuInstance, eventTrigger string) (string, error) {
	compiled := models.CompiledGuEffects(specs)
	if compiled == nil {
		return "", nil
	}
	var logs []string
	for _, operation := range compiled.NativeMechanics() {
		if string(operation.Trigger) == eventTrigger {
			mechanic := models.SpecialMechanic{EventTrigger: eventTrigger, ActionCode: operation.Code, Params: operation.Params}
			msg, err := handleActionCodeTx(tx, player, specs, guInstance, mechanic)
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
	return handleActionCodeTx(nil, player, spec, gu, mechanic)
}

func handleActionCodeTx(tx *sql.Tx, player *models.Player, spec models.GuDictionaryItem, gu models.GuInstance, mechanic models.SpecialMechanic) (string, error) {
	var params map[string]interface{}
	if len(mechanic.Params) > 0 {
		_ = json.Unmarshal(mechanic.Params, &params)
	}
	actionCode := mechanic.ActionCode
	// Không Khiếu hóa đá (aperture_petrified) chặn mọi cơ chế lên rank.
	switch actionCode {
	case "RANK_PROGRESS", "MAN_BEAST_BURIAL", "FORCE_RANK_UP":
		if gamefx.Get(player).BlockedActions[gamefx.ActionBreakthrough] {
			return "", fmt.Errorf("Không Khiếu đã hóa đá, không thể đột phá tu vi!")
		}
	}
	switch actionCode {
	case "SOVEREIGN_BODY_TRANSFORM":
		if player.BodyMods.Has(models.BodyModSovereignImmortal) {
			return "", fmt.Errorf("Thân thể đã được cải tạo thành Tiên Thể Chí Tôn")
		}
		player.BodyMods = append(player.BodyMods, models.BodyModSovereignImmortal)
		return "Tiên Thai Chí Tôn hòa tan vào thân thể! Đạo hữu đã vĩnh viễn thành tựu Tiên Thể Chí Tôn, miễn nhiễm xung đột Đạo Ngân.", nil
	case "INCREASE_APTITUDE":
		val, _ := params["value"].(float64)
		value := int(val)
		if value <= 0 {
			value = 1
		}
		oldApt := player.AptitudePercent
		player.AptitudePercent += value
		if player.AptitudePercent > 99 {
			player.AptitudePercent = 99
		}

		if player.AptitudePercent >= 80 {
			player.AptitudeType = "Giáp"
		} else if player.AptitudePercent >= 60 {
			player.AptitudeType = "Ất"
		} else if player.AptitudePercent >= 40 {
			player.AptitudeType = "Bính"
		} else {
			player.AptitudeType = "Đinh"
		}

		delta := float64(player.AptitudePercent - oldApt)
		if delta > 0 {
			player.MaxMana += delta
			if player.Rank < 6 && player.MaxMana > 99 {
				player.MaxMana = 99
			}
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
		oldApt := player.AptitudePercent
		player.AptitudePercent += value
		if player.AptitudePercent > 99 {
			player.AptitudePercent = 99
		}

		if player.AptitudePercent >= 80 {
			player.AptitudeType = "Giáp"
		} else if player.AptitudePercent >= 60 {
			player.AptitudeType = "Ất"
		} else if player.AptitudePercent >= 40 {
			player.AptitudeType = "Bính"
		} else {
			player.AptitudeType = "Đinh"
		}

		delta := float64(player.AptitudePercent - oldApt)
		if delta > 0 {
			player.MaxMana += delta
			if player.Rank < 6 && player.MaxMana > 99 {
				player.MaxMana = 99
			}
			if player.CurrentMana > player.MaxMana {
				player.CurrentMana = player.MaxMana
			}
		}
		return fmt.Sprintf("Hiến tế huyết mạch thành công! Tư chất Không khiếu tăng lên %d%%.", player.AptitudePercent), nil
	case "RANK_PROGRESS":
		if spec.Rank != player.Rank {
			return "", fmt.Errorf("%s chỉ dùng cho %s", spec.Name, models.GetRankName(spec.Rank))
		}
		if player.SubRank >= 4 {
			return "", fmt.Errorf("Xá Lợi Cổ không thể đột phá đại cảnh giới")
		}
		player.SubRank++
		stageNames := map[int]string{1: "Sơ Giai", 2: "Trung Giai", 3: "Cao Giai", 4: "Đỉnh Phong"}
		return fmt.Sprintf("Đột phá! Đạo hữu luyện hóa %s, tu vi nâng cao lên %s %s!", spec.Name, models.GetRankName(player.Rank), stageNames[player.SubRank]), nil
	case "MAN_BEAST_BURIAL":
		if player.Rank < 1 || player.Rank > 4 {
			return "", fmt.Errorf("%s chỉ có thể dùng ở Rank 1–4", spec.Name)
		}
		if spec.Rank != player.Rank || gu.Rank != player.Rank {
			return "", fmt.Errorf("Rank Cổ trùng phải trùng với Rank bản thân (%d)", player.Rank)
		}
		if player.SubRank != 4 {
			return "", fmt.Errorf("%s chỉ có thể dùng khi đạt Đỉnh Phong", spec.Name)
		}
		player.Rank++
		player.SubRank = 1
		player.WallProgress = 0
		player.MaxMana -= 10
		if player.MaxMana < 20 {
			player.MaxMana = 20
		}
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		return fmt.Sprintf("%s cưỡng ép đột phá! Tu vi nâng lên %s Sơ Giai, Chân Nguyên tối đa giảm 10.", spec.Name, models.GetRankName(player.Rank)), nil
	case "APERTURE_PETRIFICATION", "STONE_APERTURE":
		if gu.Rank != player.Rank {
			return "", fmt.Errorf("%s chỉ dùng được khi rank Cổ trùng với rank bản thân (%s)", spec.Name, models.GetRankName(player.Rank))
		}
		player.SubRank = 4
		dur := gamefx.AperturePetrifiedDuration(player.Rank)
		player.Statuses.Add(gamefx.StatusAperturePetrified, spec.Name, dur, 0)
		label := fmt.Sprintf("%d giờ", int(dur.Hours()))
		if hours := int(dur.Hours()); hours%24 == 0 {
			label = fmt.Sprintf("%d ngày", hours/24)
		}
		return fmt.Sprintf("%s đẩy tu vi lên Đỉnh Phong nhưng làm Không khiếu hóa đá trong %s (không tự hồi Chân Nguyên, không thể đột phá lên cảnh giới cao hơn).", spec.Name, label), nil
	case "GIVE_STONES":
		val, _ := params["value"].(float64)
		amount := val
		if amount == 0 {
			amount = 500
		}
		player.PrimevalStones += float64(amount)
		return fmt.Sprintf("Nhận ngay %d Nguyên Thạch.", int64(amount)), nil
	case "SOUL_BOOST":
		val, _ := params["value"].(float64)
		if player.DaoMarks == nil {
			player.DaoMarks = make(models.DaoMarksMap)
		}
		player.DaoMarks["soul"] += int64(val)
		return fmt.Sprintf("Hồn phách của đạo hữu được tráng kiện (+%.0f điểm Hồn lực).", val), nil
	case "FORCE_RANK_UP":
		player.SubRank = 1
		player.Rank++
		return fmt.Sprintf("Đột phá đại cảnh giới lên %s Sơ Giai!", models.GetRankName(player.Rank)), nil
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
	case "ADD_THOUGHT_MAX":
		val, _ := params["value"].(float64)
		if player.MaxThought == 0 {
			player.MaxThought = 10
		}
		player.MaxThought += val
		return fmt.Sprintf("Niệm Lực tối đa tăng thêm %.0f điểm.", val), nil
	case "RESTORE_SOUL_HP":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 10
		}
		if player.MaxSoul <= 0 {
			player.MaxSoul = 100
		}
		before := player.SoulHP
		player.SoulHP += int(val)
		if player.SoulHP > player.MaxSoul {
			player.SoulHP = player.MaxSoul
		}
		restored := player.SoulHP - before
		return fmt.Sprintf("Hồn phách được dưỡng hồi (+%d Hồn Phách).", restored), nil
	// ── New action codes ──────────────────────────────────────────────
	// TIME_ACCELERATION also serves THIRD_WATCH (Tam Canh Cổ: time flow ×3).
	case "TIME_ACCELERATION", "THIRD_WATCH":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 3.0
		}
		player.TimeSpeedMultiplier *= val
		return fmt.Sprintf("%s hòa tan vào không khiếu, dòng thời gian tăng tốc x%.2f!", spec.Name, val), nil
	case "RECORD", "RECORD_MESSAGE":
		return fmt.Sprintf("%s ghi lại toàn bộ hình ảnh và âm thanh trong khu vực.", spec.Name), nil
	case "CONTRACT_OATH", "POISON_VOW":
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
		player.PrimevalStones += float64(val)
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
	case "SOUL_REGEN_TICK":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 1.0
		}
		if player.MaxSoul <= 0 {
			player.MaxSoul = 100
		}
		player.SoulHP += int(val)
		if player.SoulHP > player.MaxSoul {
			player.SoulHP = player.MaxSoul
		}
		return "", nil
	case "GENERATE_IMMORTAL_STONES_TICK":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 1.0
		}
		player.ImmortalStones += float64(val)
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
	case "DECREASE_TIME_SPEED":
		val, _ := params["value"].(float64)
		if val <= 0 {
			val = 0.1
		}
		player.TimeSpeedMultiplier -= val
		if player.TimeSpeedMultiplier < 0.1 {
			player.TimeSpeedMultiplier = 0.1
		}
		return fmt.Sprintf("%s hòa tan vào không khiếu, dòng thời gian vĩnh viễn giảm -%.2f!", spec.Name, val), nil
	case "SPRING_EGG_HATCH":
		if tx == nil {
			return "", fmt.Errorf("%s chỉ có thể cấy Nguyên Tuyền khi tiêu hao trực tiếp", spec.Name)
		}
		return springEggHatchEffect(tx, player, models.ItemSpec{Name: spec.Name}, 0)
	default:
		return "", fmt.Errorf("Chưa hỗ trợ action code: %s", actionCode)
	}
}

// ApplyGrowthProgress calculates and applies growth progress for a Gu instance.
// Returns the delta boosts applied.
func ApplyGrowthProgress(gu *models.GuInstance, spec *models.GuDictionaryItem, elapsedHours float64) (attack, defense, hp, speed float64) {
	g, ok := models.GrowthProfile(*spec)
	if !ok || elapsedHours <= 0 {
		return 0, 0, 0, 0
	}
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

// HandleConsumeEvent routes a single consume mechanic through the generic
// dispatcher so every ON_CONSUME path shares one loop. Kept for backward
// compatibility with existing callers/tests.
func HandleConsumeEvent(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem, mechanic models.SpecialMechanic) (string, error) {
	return handleActionCode(player, spec, gu, mechanic)
}
