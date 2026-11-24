package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func hasMechanic(mechanics []CombatMechanic, trigger, actionCode string) bool {
	displacementType := ""
	switch actionCode {
	case "KNOCKBACK":
		displacementType = "KNOCKBACK"
	case "PULL_TARGET":
		displacementType = "PULL"
	case "SWAP_TARGET":
		displacementType = "SWAP"
	case "RETREAT_SELF":
		displacementType = "RETREAT"
	}
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			return true
		}
		if displacementType != "" && mechanic.Trigger == trigger && mechanic.ActionCode == "DISPLACE" && mechanic.Values["direction"] == displacementType {
			return true
		}
	}
	return false
}

func hasStatus(c *CombatantAuto, statusID string) bool {
	count, _ := statusStacks(c, statusID)
	return count > 0
}

func mechanicFloat(mechanics []CombatMechanic, trigger, actionCode, key string, fallback float64) float64 {
	for _, mechanic := range mechanics {
		legacyDisplacement := mechanic.ActionCode == "DISPLACE" && key == "steps" && hasMechanic([]CombatMechanic{mechanic}, trigger, actionCode)
		if mechanic.Trigger == trigger && (mechanic.ActionCode == actionCode || legacyDisplacement) {
			if value, ok := mechanic.Values[key].(float64); ok {
				return value
			}
		}
	}
	return fallback
}

func mechanicFloatByCode(mechanics []CombatMechanic, actionCode, key string, fallback float64) float64 {
	for _, mechanic := range mechanics {
		if mechanic.ActionCode == actionCode {
			if value, ok := mechanic.Values[key].(float64); ok {
				return value
			}
		}
	}
	return fallback
}

func mechanicBoolByCode(mechanics []CombatMechanic, actionCode, key string, fallback bool) bool {
	for _, mechanic := range mechanics {
		if mechanic.ActionCode == actionCode {
			if value, ok := mechanic.Values[key].(bool); ok {
				return value
			}
		}
	}
	return fallback
}

func mechanicString(mechanics []CombatMechanic, trigger, actionCode, key, fallback string) string {
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			if value, ok := mechanic.Values[key].(string); ok {
				return value
			}
		}
	}
	return fallback
}

func mechanicInt(mechanics []CombatMechanic, trigger, actionCode, key string, fallback int) int {
	return int(mechanicFloat(mechanics, trigger, actionCode, key, float64(fallback)))
}

func mechanicBool(mechanics []CombatMechanic, trigger, actionCode, key string, fallback bool) bool {
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			if value, ok := mechanic.Values[key].(bool); ok {
				return value
			}
		}
	}
	return fallback
}

// hasMinion reports whether the combatant already has a summon on the field, so
// a repeated cast of the same Gu does not stack minions every turn.
func hasMinion(state *CombatState, actorID string) bool {
	prefix := actorID + "_summon_"
	for _, side := range [][]*CombatantAuto{state.Allies, state.Enemies} {
		for _, c := range side {
			if c.Kind == "Minion" && strings.HasPrefix(c.ID, prefix) {
				return true
			}
		}
	}
	return false
}

// summonGrid maps the authored position param onto the combat grid.
func summonGrid(position string) (int, int) {
	switch position {
	case "mid":
		return 3, 3
	case "back":
		return 5, 3
	default:
		return 1, 3 // frontline
	}
}

// findMechanic returns the first mechanic matching trigger+code.
func findMechanic(mechanics []CombatMechanic, trigger, actionCode string) (CombatMechanic, bool) {
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			return mechanic, true
		}
	}
	return CombatMechanic{}, false
}

// spawnSummonedMonster builds the SUMMON_MONSTER minion for actor and appends it
// to the matching side. rank is the rank of the summoning Gu. daoMult amplifies
// the stats after the anti-abuse clamps, exactly like the damage/shield/heal of
// the summoning Gu's own action. Returns the minion and its SUMMONED log detail.
func spawnSummonedMonster(actor *CombatantAuto, m CombatMechanic, rank int, isAlly bool, state *CombatState, summonCount int, daoMult float64) (*CombatantAuto, string) {
	ms := []CombatMechanic{m}
	rankFactor := 1.0
	if mechanicBool(ms, m.Trigger, "SUMMON_MONSTER", "rank_scale", true) {
		rankFactor = gamefx.RankScale(rank)
	}
	hpScale := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "hp_scale", 1)
	damageScale := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "damage_scale", 1)
	defenseScale := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "defense_scale", damageScale)
	statSource := mechanicString(ms, m.Trigger, "SUMMON_MONSTER", "stat_source", "TEMPLATE")
	hpBase := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "hp", 0)
	damageBase := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "damage", 0)
	defenseBase := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "defense", 0)
	if statSource == "ACTOR" {
		hpBase, damageBase, defenseBase = actor.MaxHP, actor.BaseDmg, actor.BaseDef
	}
	hp := hpBase * hpScale * rankFactor
	damage := damageBase * damageScale * rankFactor
	defense := defenseBase * defenseScale * rankFactor
	if hp > actor.MaxHP*2.0 {
		hp = actor.MaxHP * 2.0
	}
	if hp <= 0 {
		hp = actor.MaxHP * 0.5
	}
	if damage > actor.BaseDmg*3.0 {
		damage = actor.BaseDmg * 3.0
	}
	// Dao marks amplify the summon after the anti-abuse clamps, otherwise the
	// clamp would swallow the whole bonus at high dao mark counts.
	hp *= daoMult
	damage *= daoMult
	defense *= daoMult
	name := mechanicString(ms, m.Trigger, "SUMMON_MONSTER", "name", "Triệu hồi vật")
	gridX, gridY := summonGrid(mechanicString(ms, m.Trigger, "SUMMON_MONSTER", "position", "frontline"))
	minionRank := mechanicInt(ms, m.Trigger, "SUMMON_MONSTER", "rank", rank)
	minionSpeed := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "speed", actor.Speed)
	id := fmt.Sprintf("%s_summon_%d", actor.ID, summonCount)
	if suffix := mechanicString(ms, m.Trigger, "SUMMON_MONSTER", "id_suffix", ""); suffix != "" {
		id = actor.ID + "_" + suffix
	}
	minion := &CombatantAuto{
		ID:      id,
		Name:    name + " (Triệu hồi)",
		Kind:    "Minion",
		Rank:    minionRank,
		GridX:   gridX,
		GridY:   gridY,
		MaxHP:   hp,
		HP:      hp,
		BaseDmg: damage,
		BaseDef: defense,
		Speed:   minionSpeed,
	}
	if isAlly {
		state.Allies = append(state.Allies, minion)
	} else {
		state.Enemies = append(state.Enemies, minion)
	}
	detail := fmt.Sprintf("%s triệu hồi %s (%.0f HP, %.0f công) lên chiến trường!", actor.Name, name, hp, damage)
	if daoMult > 1.0 {
		detail = fmt.Sprintf("%s triệu hồi %s (%.0f HP, %.0f công, Đạo Ngân ×%.2f) lên chiến trường!", actor.Name, name, hp, damage, daoMult)
	}
	if template := mechanicString(ms, m.Trigger, "SUMMON_MONSTER", "log_message", ""); template != "" {
		daoNote := ""
		if daoMult > 1.0 {
			daoNote = fmt.Sprintf(", Đạo Ngân ×%.2f", daoMult)
		}
		detail = strings.NewReplacer(
			"{actor}", actor.Name,
			"{summon}", name,
			"{hp}", fmt.Sprintf("%.0f", hp),
			"{damage}", fmt.Sprintf("%.0f", damage),
			"{dao_note}", daoNote,
		).Replace(template)
	}
	return minion, detail
}

// spawnWolfFrontline builds the SUMMON_WOLF_FRONTLINE minion (Địa Lang Vương)
// for actor, pushes it to the frontline and pulls the caster back to mid.
func spawnWolfFrontline(actor *CombatantAuto, isAlly bool, state *CombatState, daoMult float64) string {
	minionHP := actor.MaxHP * 0.50 * daoMult
	minion := &CombatantAuto{
		ID:      actor.ID + "_dialang",
		Name:    "Địa Lang Vương (Triệu hồi)",
		Kind:    "Minion",
		GridX:   1, // Frontline
		GridY:   3, // Mid
		MaxHP:   minionHP,
		HP:      minionHP,
		BaseDmg: actor.BaseDmg / 3.0 * daoMult,
		BaseDef: actor.BaseDef / 2.0 * daoMult,
	}
	if isAlly {
		state.Allies = append(state.Allies, minion)
	} else {
		state.Enemies = append(state.Enemies, minion)
	}
	actor.GridX = 3 // Move actor to Mid
	detail := fmt.Sprintf("%s triệu hồi Địa Lang Vương (%.0f HP) lên tiền tuyến để đỡ đòn!", actor.Name, minionHP)
	if daoMult > 1.0 {
		detail = fmt.Sprintf("%s triệu hồi Địa Lang Vương (%.0f HP, Đạo Ngân ×%.2f) lên tiền tuyến để đỡ đòn!", actor.Name, minionHP, daoMult)
	}
	return detail
}

// applyBattleStartSummons fires ON_BATTLE_START summon hooks once, before the
// first turn, so summoning Gu reinforce the field at battle start instead of
// recasting their summon every turn. summonCount is shared with the per-turn
// ON_CAST path so minion IDs stay unique across both.
func applyBattleStartSummons(state *CombatState, summonCount *int) []CombatLog {
	var logs []CombatLog
	apply := func(combatants []*CombatantAuto, isAlly bool) {
		// Snapshot the length: summons append to the same slice.
		n := len(combatants)
		for i := 0; i < n; i++ {
			actor := combatants[i]
			if actor == nil || actor.Kind == "Minion" {
				continue
			}
			for _, m := range actor.Mechanics {
				if m.Trigger != "ON_BATTLE_START" {
					continue
				}
				switch m.ActionCode {
				case "SUMMON_MONSTER":
					if hasMinion(state, actor.ID) {
						continue
					}
					daoMult, _ := actorDaoMultiplier(actor, m.Path)
					_, detail := spawnSummonedMonster(actor, m, m.Rank, isAlly, state, *summonCount, daoMult)
					*summonCount++
					logs = append(logs, CombatLog{Turn: 0, ActorId: actor.ID, ActorName: actor.Name, Action: "SUMMONED", Detail: detail})
				case "SUMMON_WOLF_FRONTLINE":
					daoMult, _ := actorDaoMultiplier(actor, m.Path)
					detail := spawnWolfFrontline(actor, isAlly, state, daoMult)
					logs = append(logs, CombatLog{Turn: 0, ActorId: actor.ID, ActorName: actor.Name, Action: "SUMMONED", Detail: detail})
				case "DISPLACE":
					steps := mechanicInt([]CombatMechanic{m}, m.Trigger, m.ActionCode, "steps", 1)
					switch m.Values["direction"] {
					case "RETREAT":
						if steps < 1 {
							steps = 1
						}
						actor.GridX += steps
						if actor.GridX > 5 {
							actor.GridX = 5
						}
					case "SET_ROW":
						actor.GridX = steps
					}
				}
			}
		}
	}
	apply(state.Allies, true)
	apply(state.Enemies, false)
	return logs
}

// applyBattleStartInstantWin resolves typed FORCE_VICTORY and legacy
// ON_BATTLE_START/INSTANT_WIN before the
// first turn: the carrier's side drops every opposing combatant outright,
// bypassing shields, damage immunity and Bất Tử. The normal end-of-turn win
// check then reports the victory.
func applyBattleStartInstantWin(state *CombatState) []CombatLog {
	var logs []CombatLog
	wipe := func(carriers, targets []*CombatantAuto) {
		for _, c := range carriers {
			if c == nil || c.HP <= 0 {
				continue
			}
			var command *BattleEndCommand
			for i := range c.BattleStartEndCommands {
				if c.BattleStartEndCommands[i].Outcome == "FORCE_VICTORY" {
					command = &c.BattleStartEndCommands[i]
					break
				}
			}
			legacy := hasMechanic(c.Mechanics, "ON_BATTLE_START", "INSTANT_WIN")
			if command == nil && !legacy {
				continue
			}
			action, detail := "INSTANT_WIN", fmt.Sprintf("[%s] định đoạt chiến cuộc — toàn bộ địch nhân gục ngã!", c.Name)
			if command != nil {
				if command.LogAction != "" {
					action = command.LogAction
				}
				if command.Message != "" {
					detail = strings.NewReplacer("{actor}", c.Name, "{target_count}", fmt.Sprintf("%d", len(targets))).Replace(command.Message)
				}
			}
			logs = append(logs, CombatLog{
				Turn:      0,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    action,
				Detail:    detail,
			})
			for _, t := range targets {
				if t == nil {
					continue
				}
				t.HP = 0
				t.IsDead = true
			}
			return
		}
	}
	wipe(state.Allies, state.Enemies)
	wipe(state.Enemies, state.Allies)
	return logs
}

func actorHasMechanic(c *CombatantAuto, trigger, actionCode string) bool {
	if c == nil {
		return false
	}
	if hasMechanic(c.Mechanics, trigger, actionCode) {
		return true
	}
	for _, a := range c.Actions {
		if hasMechanic(a.Mechanics, trigger, actionCode) {
			return true
		}
	}
	return false
}

// counterThresholdHitRule reports whether an authored battle-start counter
// rule enables one of its hit effects for the combatant's current state.
// SAT_KHI_GUARANTEE remains a legacy fallback for persisted native definitions.
func counterThresholdHitRule(c *CombatantAuto, effect string) bool {
	if c == nil || c.CustomState == nil {
		return false
	}
	for _, mechanic := range c.Mechanics {
		if mechanic.Trigger != "ON_BATTLE_START" {
			continue
		}
		key := ""
		threshold := 10
		var enabled bool
		switch mechanic.ActionCode {
		case "HIT_RULE_COUNTER_THRESHOLD":
			key, _ = mechanic.Values["key"].(string)
			if value, ok := mechanic.Values["threshold"].(float64); ok {
				threshold = int(value)
			}
			enabled, _ = mechanic.Values[effect].(bool)
		case "SAT_KHI_GUARANTEE":
			key = "sat_khi_level"
			threshold = int(mechanicFloat([]CombatMechanic{mechanic}, "ON_BATTLE_START", "SAT_KHI_GUARANTEE", "sat_khi_threshold", 10))
			enabled = effect == "guaranteed" || effect == "bypass_shield" || effect == "bypass_immunity"
		default:
			continue
		}
		if enabled && key != "" && c.CustomState[key] > threshold {
			return true
		}
	}
	return false
}

// applyBattleStartBattlefieldsAndHouses executes battlefield killer moves and Immortal Gu House auras at turn 0.
func applyBattleStartBattlefieldsAndHouses(state *CombatState) []CombatLog {
	var logs []CombatLog
	applySide := func(team, opponents []*CombatantAuto, isAlly bool) {
		for _, c := range team {
			if c == nil || c.HP <= 0 {
				continue
			}
			// 1. Vạn Niên Đấu Phi Thuyền (MYRIAD_YEAR_WARSHIP)
			if actorHasMechanic(c, "ON_BATTLE_START", "MYRIAD_YEAR_WARSHIP") {
				for _, ally := range team {
					if ally != nil && ally.HP > 0 {
						ally.ActiveEffects = append(ally.ActiveEffects, Effect{
							ID: statusfx.StatusTimeSpeedSurge, Value: 50, Duration: 5, Type: "buff",
						})
					}
				}
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "MYRIAD_YEAR_WARSHIP",
					Detail:    fmt.Sprintf("[Tiên Cổ Ốc - Vạn Niên Đấu Phi Thuyền] %s ngự giá phi thuyền xé toạc Trường Hà Thời Gian, tăng 50%% Tốc độ cho toàn đội!", c.Name),
				})
			}

			// 2. Bát Cực Quán Trấn Vận (EIGHT_EXTREMITIES_LUCK)
			if actorHasMechanic(c, "ON_BATTLE_START", "EIGHT_EXTREMITIES_LUCK") {
				for _, opp := range opponents {
					if opp != nil && opp.HP > 0 {
						opp.ActiveEffects = append(opp.ActiveEffects,
							Effect{ID: statusfx.StatusLuckSiphon, Value: 20, Duration: 5, Type: "debuff"},
						)
						opp.CritRateBonus -= 0.20
						opp.EvasionChance = math.Max(0, opp.EvasionChance-0.25)
					}
				}
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "EIGHT_EXTREMITIES_LUCK",
					Detail:    fmt.Sprintf("[Tiên Cổ Ốc - Bát Cực Quán] %s tỏa sáng Trấn Vận Cổ Trận trấn áp khí vận, giảm mạnh 20%% Bạo kích và 25%% Né tránh của toàn bộ quân địch!", c.Name),
				})
			}

			// 3. Ngũ Giới Phân Tranh Trận (FIVE_REGIONAL_BOUNDARY)
			if actorHasMechanic(c, "ON_BATTLE_START", "FIVE_REGIONAL_BOUNDARY") {
				for _, opp := range opponents {
					if opp != nil && opp.HP > 0 {
						opp.ActiveEffects = append(opp.ActiveEffects, Effect{
							ID: statusfx.StatusFiveRegionalSuppression, Value: 1, Duration: 5, Type: "debuff",
						})
					}
				}
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "FIVE_REGIONAL_BOUNDARY",
					Detail:    fmt.Sprintf("[Chiến Trường Sát Chiêu - Ngũ Giới Phân Tranh] %s kích phát Giới Vực Bích dâng trào phân tách thiên địa, giam cầm toàn bộ quân địch trong trạng thái Áp Chế Giới Vực!", c.Name),
				})
			}

			// 4. Yểm Hồn Ma Trận (SOUL_CONCEALMENT_ARRAY)
			if actorHasMechanic(c, "ON_BATTLE_START", "SOUL_CONCEALMENT_ARRAY") {
				for _, ally := range team {
					if ally != nil && ally.HP > 0 {
						ally.ActiveEffects = append(ally.ActiveEffects,
							Effect{ID: statusfx.StatusStealth, Duration: 2, Type: "buff"},
							Effect{ID: statusfx.StatusEvasion, Value: 35, Duration: 3, Type: "buff"},
						)
						ally.EvasionChance = math.Max(ally.EvasionChance, 0.35)
					}
				}
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "SOUL_CONCEALMENT_ARRAY",
					Detail:    fmt.Sprintf("[Đại Trận - Yểm Hồn Ma Trận] %s thi triển sương mù hồn phách che phủ chiến trường, che giấu toàn đội trong trạng thái Ẩn Thân và tăng 35%% Né Tránh!", c.Name),
				})
			}

			// 5. Gặp Mặt Từng Quen Biết (FAMILIAR_FACE)
			if actorHasMechanic(c, "ON_BATTLE_START", "FAMILIAR_FACE") || actorHasMechanic(c, "COMBAT_PASSIVE", "FAMILIAR_FACE") {
				c.ActiveEffects = append(c.ActiveEffects,
					Effect{ID: statusfx.StatusFamiliarFace, Value: 50, Duration: 3, Type: "buff"},
					Effect{ID: statusfx.StatusEvasion, Value: 50, Duration: 3, Type: "buff"},
				)
				c.EvasionChance = math.Max(c.EvasionChance, 0.50)
				c.CritRateBonus += 1.0 // Guaranteed crit on first strike
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "FAMILIAR_FACE",
					Detail:    fmt.Sprintf("[Sát Chiêu - Gặp Mặt Từng Quen Biết] %s ngụy trang hoàn mỹ hòa mình vào hư không, nhận 50%% Né Tránh và đòn đánh đầu chắc chắn Bạo Kích!", c.Name),
				})
			}

			// 6. Cửu Cửu Liên Hoàn Trận (NINETY_NINE_ARRAY)
			if actorHasMechanic(c, "ON_BATTLE_START", "NINETY_NINE_ARRAY") || actorHasMechanic(c, "COMBAT_PASSIVE", "NINETY_NINE_ARRAY") {
				shieldGain := c.MaxHP * 0.40
				c.VirtualShield += shieldGain
				c.ActiveEffects = append(c.ActiveEffects, Effect{
					ID: statusfx.StatusNinetyNineShield, Value: 50, Duration: 5, Type: "buff",
				})
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "NINETY_NINE_ARRAY",
					Detail:    fmt.Sprintf("[Hộ Trận - Cửu Cửu Liên Hoàn] 99 tầng trận giáp khởi động, ngưng tụ %.0f Hộ Thuẫn và hấp thụ 50%% sát thương nhận vào!", shieldGain),
				})
			}

			// 7. Lưới Trời Tuy Thưa (HEAVEN_NET_RESTRICTION)
			if actorHasMechanic(c, "ON_BATTLE_START", "HEAVEN_NET_RESTRICTION") {
				for _, opp := range opponents {
					if opp != nil && opp.HP > 0 {
						opp.EvasionChance = math.Max(0, opp.EvasionChance-0.50)
					}
				}
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "HEAVEN_NET_RESTRICTION",
					Detail:    fmt.Sprintf("[Thiên Đạo - Lưới Trời Tuy Thưa] %s thôi động Khôi Khôi Thiên Võng phong tỏa toàn trường, giảm 50%% Né Tránh của toàn bộ phe địch!", c.Name),
				})
			}

			// 8. Vạn Chúng Nhất Tâm (ALL_PEOPLE_UNITED)
			if actorHasMechanic(c, "ON_BATTLE_START", "ALL_PEOPLE_UNITED") {
				for _, ally := range team {
					if ally != nil && ally.HP > 0 {
						shieldGain := ally.MaxHP * 0.30
						ally.VirtualShield += shieldGain
						ally.DamageMultiplier += 0.35
						ally.ActiveEffects = append(ally.ActiveEffects, Effect{
							ID: statusfx.StatusAllPeopleUnited, Duration: 4, Type: "buff",
						})
					}
				}
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "ALL_PEOPLE_UNITED",
					Detail:    fmt.Sprintf("[Nhân Đạo - Vạn Chúng Nhất Tâm] %s hiệu triệu vạn dân đồng lòng! Toàn đội nhận Khiên Nhân Đạo 30%% HP và tăng 35%% Sát Thương!", c.Name),
				})
			}

			// 9. Đại Dạ Di Thiên (GREAT_NIGHT)
			if actorHasMechanic(c, "ON_BATTLE_START", "GREAT_NIGHT") {
				for _, ally := range team {
					if ally != nil && ally.HP > 0 {
						ally.CritRateBonus += 0.50
						ally.ActiveEffects = append(ally.ActiveEffects,
							Effect{ID: statusfx.StatusStealth, Duration: 2, Type: "buff"},
							Effect{ID: statusfx.StatusGreatNight, Duration: 4, Type: "buff"},
						)
					}
				}
				for _, opp := range opponents {
					if opp != nil && opp.HP > 0 {
						opp.EvasionChance = math.Max(0, opp.EvasionChance-0.40)
					}
				}
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "GREAT_NIGHT",
					Detail:    fmt.Sprintf("[Ám Đạo - Đại Dạ Di Thiên] Màn đêm vĩnh hằng buông xuống! Toàn đội Ẩn Thân trong 2 hiệp, +50%% Bạo Kích và giảm 40%% Né Tránh phe địch!"),
				})
			}

			// 10. Đồ Đằng Kim Chung (TOTEM_GOLDEN_BELL)
			if actorHasMechanic(c, "ON_BATTLE_START", "TOTEM_GOLDEN_BELL") {
				c.ActiveEffects = append(c.ActiveEffects, Effect{
					ID: statusfx.StatusTotemGoldenBell, Duration: 4, Type: "buff",
				})
				logs = append(logs, CombatLog{
					Turn:      0,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "TOTEM_GOLDEN_BELL",
					Detail:    fmt.Sprintf("[Đồ Đằng - Kim Chung] Chuông vàng viễn cổ buông rủ ngưng tụ hào quang, kích hoạt 35%% Phản thương và phòng ngự bạo kích!"),
				})
			}
		}
	}
	applySide(state.Allies, state.Enemies, true)
	applySide(state.Enemies, state.Allies, false)
	return logs
}

// lifespanByRank approximates an enemy's thọ nguyên from its rank. Players use
// their tracked Lifespan; monsters have none, so this is the calibration knob
// for ON_BATTLE_START/SEIZE_INITIATIVE. Tune here, not per-Gu.
func lifespanByRank(rank, subRank int) float64 {
	table := [...]float64{10, 20, 50, 100, 200, 500, 1000, 2000, 5000}
	if rank < 1 {
		rank = 1
	}
	if rank > len(table) {
		rank = len(table)
	}
	return table[rank-1] + float64(subRank)*10
}

// combatantLifespan returns the tracked lifespan, falling back to the rank
// approximation for combatants (monsters) that have none.
func combatantLifespan(c *CombatantAuto) float64 {
	if c == nil {
		return 0
	}
	if c.Lifespan > 0 {
		return c.Lifespan
	}
	return lifespanByRank(c.Rank, c.SubRank)
}

// opponentPrimary returns the first combatant on the opposing side, used as the
// reference for lifespan-gated battle-start mechanics.
func opponentPrimary(state *CombatState, isAlly bool) *CombatantAuto {
	if isAlly {
		if len(state.Enemies) > 0 {
			return state.Enemies[0]
		}
		return nil
	}
	if len(state.Allies) > 0 {
		return state.Allies[0]
	}
	return nil
}

// seizeInitiative scans for a combatant that, on turn 1, may act before everyone
// else via ON_BATTLE_START/SEIZE_INITIATIVE when the opposing primary's lifespan
// does not exceed lifespan_ratio × its own. A seizer opposed by Cấm Cổ
// (ON_BATTLE_START/SUPPRESS_PRIORITY) is skipped and reported in suppressed so
// the caller can log it; turn order then falls back to normal speed.
// Returns the winner (or nil) plus the names of suppressed seizers.
func seizeInitiative(state *CombatState, allActors []*CombatantAuto) (*CombatantAuto, []string) {
	var suppressed []string
	for _, actor := range allActors {
		if !hasMechanic(actor.Mechanics, "ON_BATTLE_START", "SEIZE_INITIATIVE") {
			continue
		}
		isAlly := false
		for _, a := range state.Allies {
			if a.ID == actor.ID {
				isAlly = true
				break
			}
		}
		if sideHasPrioritySuppressor(state, allActors, !isAlly) {
			suppressed = append(suppressed, actor.Name)
			continue
		}
		opp := opponentPrimary(state, isAlly)
		selfLife := combatantLifespan(actor)
		if opp == nil || selfLife <= 0 {
			continue
		}
		ratio := mechanicFloat(actor.Mechanics, "ON_BATTLE_START", "SEIZE_INITIATIVE", "lifespan_ratio", 3)
		if combatantLifespan(opp) <= ratio*selfLife {
			return actor, suppressed
		}
	}
	return nil, suppressed
}

// sideHasPrioritySuppressor reports whether any combatant on the given side
// carries Cấm Cổ (ON_BATTLE_START/SUPPRESS_PRIORITY).
func sideHasPrioritySuppressor(state *CombatState, allActors []*CombatantAuto, isAlly bool) bool {
	for _, actor := range allActors {
		if onSide(state, actor, isAlly) && suppressesPriority(actor) {
			return true
		}
	}
	return false
}

// onSide reports whether actor fights on the given side.
func onSide(state *CombatState, actor *CombatantAuto, isAlly bool) bool {
	if actor == nil {
		return false
	}
	if isAlly {
		for _, a := range state.Allies {
			if a.ID == actor.ID {
				return true
			}
		}
		return false
	}
	for _, e := range state.Enemies {
		if e.ID == actor.ID {
			return true
		}
	}
	return false
}

// sidePriorityWouldFire reports whether a side carries a battle-start priority
// effect that would actually trigger on turn 1: a SEIZE_INITIATIVE whose
// lifespan condition passes, or a FIRST_TURN_EVADE carrier (deterministic —
// it negates the first strike it takes). Decides whether an opposing Cấm Cổ
// suppression "activates successfully".
func sidePriorityWouldFire(state *CombatState, isAlly bool) bool {
	var side []*CombatantAuto
	if isAlly {
		side = state.Allies
	} else {
		side = state.Enemies
	}
	for _, c := range side {
		if c == nil || c.HP <= 0 {
			continue
		}
		if hasMechanic(c.Mechanics, "ON_BATTLE_START", "SEIZE_INITIATIVE") {
			selfLife := combatantLifespan(c)
			opp := opponentPrimary(state, isAlly)
			if opp != nil && selfLife > 0 &&
				combatantLifespan(opp) <= mechanicFloat(c.Mechanics, "ON_BATTLE_START", "SEIZE_INITIATIVE", "lifespan_ratio", 3)*selfLife {
				return true
			}
		}
		if hasMechanic(c.Mechanics, "BEFORE_HIT", "FIRST_TURN_EVADE") {
			return true
		}
	}
	return false
}

// successfulSuppressor returns the fastest living Cấm Cổ holder on the given
// side when its suppression activates successfully — the opposing side
// carries a priority effect that would actually trigger. Nil otherwise.
// allActors must already be sorted by EffectiveSpeed descending.
func successfulSuppressor(state *CombatState, allActors []*CombatantAuto, isAlly bool) *CombatantAuto {
	if !sidePriorityWouldFire(state, !isAlly) {
		return nil
	}
	for _, a := range allActors {
		if a == nil || a.HP <= 0 || !suppressesPriority(a) {
			continue
		}
		if onSide(state, a, isAlly) {
			return a
		}
	}
	return nil
}

// immuneToHostileEffects reads the typed effect-immunity primitive. The native
// hook remains as a compatibility reader for persisted combat snapshots.
func immuneToHostileEffects(c *CombatantAuto) bool {
	if c == nil {
		return false
	}
	if hasMechanic(c.Mechanics, "COMBAT_PASSIVE", "REVERSE_CALAMITY") {
		return true
	}
	for _, effect := range c.PassiveOperationEffects {
		op := effect.Compiled
		if op.Type == gameplayeffects.OpEffectImmunity && op.SubType == "HOSTILE_EFFECTS" {
			return true
		}
	}
	return false
}

// suppressesPriority reports whether a combatant carries Cấm Cổ
// (ON_BATTLE_START/SUPPRESS_PRIORITY): enemy action-priority effects
// (seize initiative, first-turn evade) are ignored against its side, and a
// suppression that negates a would-trigger enemy priority lets its fastest
// living holder act first on turn 1.
func suppressesPriority(c *CombatantAuto) bool {
	return c != nil && hasMechanic(c.Mechanics, "ON_BATTLE_START", "SUPPRESS_PRIORITY")
}

// sealGuTurns seals a combatant's Gu for n full turns. Duration is n+1
// because durations tick at the next turn start before acting.
func sealGuTurns(c *CombatantAuto, turns int) {
	if c == nil {
		return
	}
	if turns < 1 {
		turns = 1
	}
	c.ActiveEffects = append(c.ActiveEffects, Effect{
		ID: statusfx.StatusGuSeal, Type: "debuff", Value: 1, Duration: turns + 1,
	})
	ComputeEffectMultipliers(c)
}

// applyRebornEmpower stacks ATK_UP + HASTE buffs and an immediate shield on
// a revived combatant. One stack per successful resurrection (useCount for
// the log). Zero params = no-op so legacy REBORN data is unaffected.
func applyRebornEmpower(c *CombatantAuto, mechanics []CombatMechanic, turn, useCount int, logs *[]CombatLog) {
	atk := mechanicFloat(mechanics, "ON_DEATH", "REBORN", "empower_attack_percent", 0)
	spd := mechanicFloat(mechanics, "ON_DEATH", "REBORN", "empower_speed_percent", 0)
	sh := mechanicFloat(mechanics, "ON_DEATH", "REBORN", "empower_shield_percent", 0)
	turns := mechanicInt(mechanics, "ON_DEATH", "REBORN", "empower_turns", 100)
	if atk <= 0 && spd <= 0 && sh <= 0 {
		return
	}
	if turns < 2 {
		turns = 2
	}
	if atk > 0 {
		c.ActiveEffects = append(c.ActiveEffects, Effect{
			ID: statusfx.StatusAtkUp, Type: "buff", Value: int(atk * 100), Duration: turns,
		})
	}
	if spd > 0 {
		c.ActiveEffects = append(c.ActiveEffects, Effect{
			ID: statusfx.StatusHaste, Type: "buff", Value: int(spd * 100), Duration: turns,
		})
	}
	if sh > 0 {
		c.VirtualShield += c.MaxHP * sh
	}
	ComputeEffectMultipliers(c)
	*logs = append(*logs, CombatLog{
		Turn:      turn,
		ActorId:   c.ID,
		ActorName: c.Name,
		Action:    "REBORN_EMPOWER",
		Detail:    fmt.Sprintf("%s hồi sinh lần %d: +%d%% công, +%d%% tốc, +%.0f khiên!", c.Name, useCount, int(atk*100), int(spd*100), c.MaxHP*sh),
	})
}

// DaoMultiplier returns the dao mark damage multiplier for a combatant's paths,
// using the unified DaoCombatProfile.
func DaoMultiplier(dm DaoMarks, paths []string) (float64, bool) {
	if dm == nil || len(paths) == 0 {
		return 1.0, false
	}
	daoMarks := make(map[string]int64, len(dm))
	for k, v := range dm {
		daoMarks[k] = v
	}
	profile := gamefx.BuildDaoCombatProfile(daoMarks, nil, false, nil)
	pathKey := paths[0]
	daoAmp, realmEff, conflictMult := profile.GetOffensiveFactors(pathKey)
	mult := (1.0 + (daoAmp-1.0)*realmEff) * conflictMult
	return mult, true
}

// pathBonus returns the additive bonus for the action's primary path from a
// per-path bonus map. Secondary paths do not participate unless an action
// explicitly opts into multi-path modifier behavior.
func pathBonus(bonus map[string]float64, paths []string) float64 {
	if bonus == nil {
		return 0
	}
	path := daoPathForAction(paths)
	if value := bonus[path]; value > 0 {
		return value
	}
	return 0
}

// ensureDaoProfile lazily builds the combatant's DaoCombatProfile.
func ensureDaoProfile(c *CombatantAuto) {
	if c == nil || c.DaoProfile != nil {
		return
	}
	if len(c.DaoMarks) == 0 && len(c.PathMasteries) == 0 && len(c.DaoBonus) == 0 {
		// Empty inputs produce neutral factors, which the profile getters default to.
		return
	}
	var daoMarks map[string]int64
	if c.DaoMarks != nil {
		daoMarks = make(map[string]int64, len(c.DaoMarks))
		for k, v := range c.DaoMarks {
			daoMarks[k] = v
		}
	}
	c.DaoProfile = gamefx.BuildDaoCombatProfileWithReduction(daoMarks, c.PathMasteries, c.HasNonConflictingDao, c.DaoConflictReductionPct, c.DaoBonus)
}

// actionConflictMult returns the dao conflict multiplier for an action's path
// (1.0 = no conflict). Non-conflicting characters always return 1.0.
func actionConflictMult(c *CombatantAuto, paths []string) float64 {
	if c == nil || c.HasNonConflictingDao {
		return 1.0
	}
	ensureDaoProfile(c)
	if c.DaoProfile == nil {
		return 1.0
	}
	pathKey := "strength"
	if len(paths) > 0 {
		pathKey = paths[0]
	}
	_, _, conflictMult := c.DaoProfile.GetOffensiveFactors(pathKey)
	return conflictMult
}

func daoBackfireExplanation(c *CombatantAuto, action *SimAction, conflictMult, chance, damage float64) string {
	if c == nil || action == nil {
		return ""
	}
	ensureDaoProfile(c)
	castPath := "strength"
	if len(action.Path) > 0 {
		castPath = gamefx.CleanPath(action.Path[0])
	}
	dominantPath := ""
	if c.DaoProfile != nil {
		dominantPath = c.DaoProfile.DominantPath
	}
	dominantPurity := 0.0
	if dominantPath != "" && c.DaoProfile != nil {
		if dominant, ok := c.DaoProfile.Paths[dominantPath]; ok {
			dominantPurity = dominant.Purity * 100
		}
	}
	name := action.Name
	if name == "" {
		name = "kỹ năng"
	}
	dominantSummary := "không có Đạo Ngân chủ đạo"
	if dominantPath != "" {
		dominantSummary = fmt.Sprintf("Đạo Ngân chủ đạo [%s], chiếm %.1f%%", dominantPath, dominantPurity)
	}
	return fmt.Sprintf("Kỹ năng [%s] thuộc lưu phái [%s] xung đột với %s; hệ số xung đột %.3f, xác suất phản phệ %.1f%%. Phản phệ gây %.0f sát thương (%.1f%% HP tối đa). Độ tinh thuần của Đạo Ngân chủ đạo không bảo đảm an toàn cho kỹ năng khác lưu phái.",
		name, castPath, dominantSummary, conflictMult, chance*100, damage, 100*damage/math.Max(1, c.MaxHP))
}

// actorDaoMultiplier returns the dao multiplier for an action's paths including
// per-path bonuses, realm efficiency, and internal conflict from DaoProfile.
func actorDaoMultiplier(c *CombatantAuto, paths []string) (float64, bool) {
	if c == nil {
		return 1.0, false
	}
	ensureDaoProfile(c)
	if c.DaoProfile == nil {
		return 1.0, true
	}
	pathKey := "strength"
	if len(paths) > 0 {
		pathKey = paths[0]
	}
	daoAmp, realmEff, conflictMult := c.DaoProfile.GetOffensiveFactors(pathKey)
	mult := (1.0 + (daoAmp-1.0)*realmEff) * conflictMult
	return mult, true
}

// resistControl returns true if the combatant's mental resistance causes a
// control effect to be resisted. Uses the provided rng for deterministic combat.
func resistControl(c *CombatantAuto, rng *rand.Rand) bool {
	if c == nil || c.MentalResistance <= 0 {
		return false
	}
	return rng.Float64() < c.MentalResistance
}

func cloneCombatState(state *CombatState) *CombatState {
	if state == nil {
		return nil
	}
	clone := &CombatState{
		Turn:          state.Turn,
		Seed:          state.Seed,
		BossPhase:     state.BossPhase,
		VirtualMode:   state.VirtualMode,
		ApertureHP:    state.ApertureHP,
		ApertureMaxHP: state.ApertureMaxHP,
		Allies:        make([]*CombatantAuto, len(state.Allies)),
		Enemies:       make([]*CombatantAuto, len(state.Enemies)),
	}
	for i, a := range state.Allies {
		clone.Allies[i] = cloneCombatant(a)
	}
	for i, e := range state.Enemies {
		clone.Enemies[i] = cloneCombatant(e)
	}
	return clone
}

func cloneCombatant(c *CombatantAuto) *CombatantAuto {
	if c == nil {
		return nil
	}
	clone := *c
	if c.DamageOutputModifiers != nil {
		clone.DamageOutputModifiers = append([]TargetDamageModifier(nil), c.DamageOutputModifiers...)
	}
	if c.Actions != nil {
		clone.Actions = make([]SimAction, len(c.Actions))
		copy(clone.Actions, c.Actions)
		for i := range clone.Actions {
			clone.Actions[i].Mechanics = cloneMechanics(c.Actions[i].Mechanics)
		}
	}
	clone.Mechanics = cloneMechanics(c.Mechanics)
	if c.ActiveEffects != nil {
		clone.ActiveEffects = make([]Effect, len(c.ActiveEffects))
		copy(clone.ActiveEffects, c.ActiveEffects)
	}
	if c.ActiveDoT != nil {
		clone.ActiveDoT = make([]DoTEffect, len(c.ActiveDoT))
		copy(clone.ActiveDoT, c.ActiveDoT)
	}
	if c.DaoMarks != nil {
		clone.DaoMarks = make(DaoMarks)
		for k, v := range c.DaoMarks {
			clone.DaoMarks[k] = v
		}
	}
	if c.CustomState != nil {
		clone.CustomState = make(map[string]int)
		for k, v := range c.CustomState {
			clone.CustomState[k] = v
		}
	} else {
		clone.CustomState = make(map[string]int)
	}
	clone.HasNonConflictingDao = c.HasNonConflictingDao
	clone.DaoConflictReductionPct = c.DaoConflictReductionPct
	clone.DaoProfile = c.DaoProfile
	return &clone
}

func cloneMechanics(source []CombatMechanic) []CombatMechanic {
	if source == nil {
		return nil
	}
	result := make([]CombatMechanic, len(source))
	copy(result, source)
	for i := range result {
		result[i].Params = append([]byte(nil), source[i].Params...)
		if source[i].Values != nil {
			result[i].Values = make(map[string]any, len(source[i].Values))
			for key, value := range source[i].Values {
				result[i].Values[key] = value
			}
		}
	}
	return result
}

func getFirstAliveDeterministic(alive []*CombatantAuto) *CombatantAuto {
	for col := 1; col <= 5; col++ {
		for targetY := 1; targetY <= 5; targetY++ {
			for _, e := range alive {
				if e.GridX == col && e.GridY == targetY {
					return e
				}
			}
		}
	}
	if len(alive) > 0 {
		return alive[0]
	}
	return nil
}

type combatGridCell struct{ x, y int }

func combatAreaCells(targetType string, targetX, targetY int) map[combatGridCell]struct{} {
	cells := make(map[combatGridCell]struct{}, 9)
	add := func(x, y int) {
		if x >= 1 && x <= 5 && y >= 1 && y <= 5 {
			cells[combatGridCell{x: x, y: y}] = struct{}{}
		}
	}
	lineStart := func(center int) int {
		return min(3, max(1, center-1))
	}
	switch strings.ToUpper(strings.TrimSpace(targetType)) {
	case "ROW":
		start := lineStart(targetY)
		for y := start; y < start+3; y++ {
			add(targetX, y)
		}
	case "COL", "COLUMN":
		start := lineStart(targetX)
		for x := start; x < start+3; x++ {
			add(x, targetY)
		}
	case "CROSS":
		for _, dy := range []int{-1, 1} {
			preferredStart := lineStart(targetX)
			for distance := 0; distance < 3; distance++ {
				for _, startX := range []int{preferredStart - distance, preferredStart + distance} {
					if startX < 1 || startX > 3 || startX > targetX || startX+2 < targetX {
						continue
					}
					startY := targetY - (targetX-startX)*dy
					if startY < 1 || startY > 5 || startY+2*dy < 1 || startY+2*dy > 5 {
						continue
					}
					for offset := 0; offset < 3; offset++ {
						add(startX+offset, startY+offset*dy)
					}
					return cells
				}
			}
		}
	case "ALL", "ALL_ENEMIES":
		startX, startY := lineStart(targetX), lineStart(targetY)
		for x := startX; x < startX+3; x++ {
			for y := startY; y < startY+3; y++ {
				add(x, y)
			}
		}
	default:
		add(targetX, targetY)
	}
	return cells
}

func getTarget(enemies []*CombatantAuto) *CombatantAuto {
	alive := make([]*CombatantAuto, 0, len(enemies))
	for _, e := range enemies {
		if e.HP > 0 {
			alive = append(alive, e)
		}
	}
	return getFirstAliveDeterministic(alive)
}

// resolveTargets returns indices of alive targets based on the action's targeting type.
// It mirrors the logic in handlers/combat_targeting.go's ResolveTargets.
func resolveTargets(enemies []*CombatantAuto, action *SimAction) []*CombatantAuto {
	targetType := strings.ToUpper(strings.TrimSpace(action.TargetType))
	if targetType == "" || targetType == "TARGET" {
		targetType = "SINGLE"
	}

	var alive []*CombatantAuto
	for _, e := range enemies {
		if e.HP > 0 {
			alive = append(alive, e)
		}
	}
	if len(alive) == 0 {
		return nil
	}

	// Stealth: single-target actions cannot select stealthed combatants unless
	// there are no other valid targets.
	if targetType == "SINGLE" {
		visible := alive[:0]
		for _, e := range alive {
			if !e.IsStealthed {
				visible = append(visible, e)
			}
		}
		if len(visible) > 0 {
			alive = visible
		}
	}

	targetX := action.TargetX
	targetY := action.TargetY

	switch targetType {
	case "ROW", "COL", "COLUMN", "CROSS", "ALL", "ALL_ENEMIES":
		hits := make([]*CombatantAuto, 0, len(alive))
		area := combatAreaCells(targetType, targetX, targetY)
		for _, e := range alive {
			if _, ok := area[combatGridCell{x: e.GridX, y: e.GridY}]; ok {
				hits = append(hits, e)
			}
		}
		if len(hits) > 0 {
			return hits
		}
		if fallback := getFirstAliveDeterministic(alive); fallback != nil {
			alive[0] = fallback
			return alive[:1]
		}
		return alive[:1]

	default: // "SINGLE"
		var best *CombatantAuto
		if action.Bypass {
			// 1. Exact match targetX, targetY
			for _, e := range alive {
				if e.GridX == targetX && e.GridY == targetY {
					best = e
					break
				}
			}
			// 2. Match targetX, prioritizing the topmost lane.
			if best == nil {
				for yVal := 1; yVal <= 5; yVal++ {
					for _, e := range alive {
						if e.GridX == targetX && e.GridY == yVal {
							best = e
							break
						}
					}
					if best != nil {
						break
					}
				}
			}
		}
		// 3. Fallback to front-to-back columns and top-to-bottom lanes.
		if best == nil {
			for col := 1; col <= 5; col++ {
				for yVal := 1; yVal <= 5; yVal++ {
					for _, e := range alive {
						if e.GridX == col && e.GridY == yVal {
							best = e
							break
						}
					}
					if best != nil {
						break
					}
				}
				if best != nil {
					break
				}
			}
		}
		if best == nil && len(alive) > 0 {
			best = getFirstAliveDeterministic(alive)
		}
		if best != nil {
			alive[0] = best
			return alive[:1]
		}
		return nil
	}
}

// actionOutcome reports the result of executing a single SimAction inside the
// interleaved battle loop.
type actionOutcome struct {
	executed    bool
	hasDamage   bool
	abort       bool
	firstTarget *CombatantAuto
}

// maxInfiniteRebornRewinds bounds a REBORN with max_uses=0 (declared unlimited).
// A rewind resets the turn counter, so an unlimited rewind escapes maxTurns and
// can hold the caller's transaction (and the player row lock) indefinitely.
// Only the unlimited rewind path is capped; explicit finite uses and in-place
// revives (which maxTurns already bounds) keep their configured behaviour.
const maxInfiniteRebornRewinds = 5

// Simulate runs the deterministic auto-battle using pre-computed actions.
func Simulate(state *CombatState) []CombatLog {
	for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
		for _, c := range combatants {
			if c.CustomState == nil {
				c.CustomState = make(map[string]int)
			}
		}
	}

	// Deterministic RNG seeded from the battle seed.
	rng := rand.New(rand.NewSource(state.Seed))
	if state.Interactive != nil {
		state.Interactive.PendingActor = ""
		state.Interactive.Consumed = 0
		state.Interactive.Error = ""
	}

	var logs []CombatLog
	rebornUses := make(map[string]int)
	summonCount := 0 // unique suffix for SUMMON_MONSTER minions

	// ON_BATTLE_START summon hooks resolve before the initial snapshot so the
	// minions are on the field from turn one and survive REBORN rewind.
	battleStartLogs := applyBattleStartSummons(state, &summonCount)
	battleStartLogs = append(battleStartLogs, applyBattleStartInstantWin(state)...)
	battleStartLogs = append(battleStartLogs, applyBattleStartBattlefieldsAndHouses(state)...)

	var initialState *CombatState
	hasReborn := false
	for _, side := range [][]*CombatantAuto{state.Allies, state.Enemies} {
		for _, c := range side {
			if hasMechanic(c.Mechanics, "ON_DEATH", "REBORN") {
				hasReborn = true
				break
			}
			for _, a := range c.Actions {
				if hasMechanic(a.Mechanics, "ON_DEATH", "REBORN") {
					hasReborn = true
					break
				}
			}
			if hasReborn {
				break
			}
		}
		if hasReborn {
			break
		}
	}
	if hasReborn {
		initialState = cloneCombatState(state)
	}

	startSnap := buildSnapshot(state)
	logs = append(logs, CombatLog{
		Turn:     0,
		Action:   "BATTLE_START",
		Detail:   "Trận chiến bắt đầu.",
		Snapshot: startSnap,
	})
	for i := range battleStartLogs {
		battleStartLogs[i].Snapshot = startSnap
	}
	logs = append(logs, battleStartLogs...)

	if len(state.InitialLogs) > 0 {
		for _, initLog := range state.InitialLogs {
			initLog.Turn = 0
			if initLog.Snapshot == nil {
				initLog.Snapshot = startSnap
			}
			logs = append(logs, initLog)
		}
	}

	// Resolve battle-start terminal effects before asking a manual actor for a
	// command. Otherwise interactive combat pauses on the player's first turn
	// even though a battle-start effect has already ended the fight.
	alliesAlive := hasLivingCombatant(state.Allies)
	enemiesAlive := hasLivingCombatant(state.Enemies)
	if !alliesAlive || !enemiesAlive {
		winner := "Phe Địch"
		if alliesAlive {
			winner = "Phe Ta"
		}
		logs = append(logs, CombatLog{
			Turn: 0, Action: "BATTLE_END",
			Detail:   fmt.Sprintf("Trận chiến kết thúc. %s chiến thắng!", winner),
			Snapshot: buildSnapshot(state),
		})
		return logs
	}

	maxTurns := 1000
	var initiative *InitiativeQueue
	lastStatusTick := -1
	heldManual := make(map[string]bool)
	actors := append(append([]*CombatantAuto{}, state.Allies...), state.Enemies...)
	queue := NewInitiativeQueue(actors)
	initiative = &queue

	// Pre-sort every combatant's actions stably based on explicit Priority if set,
	// preserving the player's exact per-Gu loadout sequence:
	for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
		for _, c := range combatants {
			if len(c.Actions) <= 1 {
				continue
			}
			sort.SliceStable(c.Actions, func(i, j int) bool {
				ai := c.Actions[i]
				aj := c.Actions[j]

				pI := ai.Priority
				pJ := aj.Priority
				if ai.ActionType == "BASIC_ATTACK" && pI == 0 {
					pI = 999
				} else if pI == 0 {
					pI = 500
				}
				if aj.ActionType == "BASIC_ATTACK" && pJ == 0 {
					pJ = 999
				} else if pJ == 0 {
					pJ = 500
				}

				if pI != pJ {
					return pI < pJ
				}

				if len(c.CategoryPriorities) > 0 {
					catI := getCategoryPriorityIndex(GetSimActionCategory(ai), c.CategoryPriorities)
					catJ := getCategoryPriorityIndex(GetSimActionCategory(aj), c.CategoryPriorities)
					if catI != catJ {
						return catI < catJ
					}
				} else {
					catI := GetSimActionCategory(ai)
					catJ := GetSimActionCategory(aj)
					if catI == "shield" && catJ != "shield" {
						return true
					}
					if catJ == "shield" && catI != "shield" {
						return false
					}
				}

				return false
			})
		}
	}

	for turn := 1; turn <= maxTurns; turn++ {
		state.Turn = turn
		var turnLogs []CombatLog
		activeActorID := ""
		shouldTick := true
		if initiative != nil {
			var combatTime float64
			var ok bool
			activeActorID, combatTime, ok = initiative.NextExcluding(heldManual)
			if !ok {
				break
			}
			if control := state.Interactive; control != nil && hasPairedPendingPlans(control) && control.ManualActors[activeActorID] {
				allReady := true
				for manualID := range control.ManualActors {
					for _, entry := range initiative.Entries {
						if entry.ActorID == manualID && entry.ReadyAt > combatTime+1e-9 {
							allReady = false
							break
						}
					}
					if !allReady {
						break
					}
				}
				if !allReady {
					heldManual[activeActorID] = true
					continue
				}
				heldManual = make(map[string]bool)
			}
			tickAt := int(math.Floor(combatTime + 1e-9))
			shouldTick = tickAt > lastStatusTick
			if shouldTick {
				lastStatusTick = tickAt
			}
		}
		turnLogs = append(turnLogs, CombatLog{
			Turn:   turn,
			Action: "TURN_START",
			Detail: fmt.Sprintf("=== LƯỢT %d ===", turn),
		})

		if shouldTick {
			// ── Converge legacy ActiveEffects DoT/HoT markers into ActiveDoT ──
			for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
				for _, c := range combatants {
					if c.HP <= 0 {
						continue
					}
					normalizeLegacyDoTStatuses(c)
				}
			}

			// ── Tick DoTs for all combatants ──
			for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
				for _, c := range combatants {
					if c.HP <= 0 {
						continue
					}
					turnLogs = appendTickDoTs(turnLogs, c, turn)
				}
			}

			// ── Tick resource drain effects (ESSENCE_LEAK) ──
			for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
				for _, c := range combatants {
					if c.HP <= 0 {
						continue
					}
					turnLogs = append(turnLogs, tickResourceDrain(c, turn)...)
				}
			}

			// ── Tenacity self-dispel (after DoT damage ticked, before durations tick) ──
			for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
				for _, c := range combatants {
					if c.HP <= 0 {
						continue
					}
					turnLogs = append(turnLogs, TryTenacityDispel(c, turn, rng)...)
				}
			}

			// ── Tick status effect durations and compute multipliers ──
			for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
				for _, c := range combatants {
					if c.HP <= 0 {
						continue
					}
					turnLogs = append(turnLogs, tickStatusEffects(c, turn)...)
					ComputeEffectMultipliers(c)
					typedPoemCounter := applyTurnStartCounterAdds(c)

					// Catalog-driven Poem Intent mechanic - start-of-turn logic.
					if typedPoemCounter || hasMechanic(c.Mechanics, "ON_TURN_START", "POEM_INTENT") {
						if !typedPoemCounter {
							c.CustomState["thi_y"] += 1
						}
						c.ActiveEffects = append(c.ActiveEffects, Effect{
							ID:       "thi_ho",
							Type:     "damage_reduction",
							Value:    40, // 40% reduction
							Duration: 2,  // 2 so it lasts until the end of the turn
						})
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   c.ID,
							ActorName: c.Name,
							Action:    "BUFF",
							Detail:    fmt.Sprintf("[Thơ Cổ] Tích tụ 1 tầng Thi Ý (hiện có %d tầng), tạo Thi Hộ giảm 40%% sát thương nhận vào trong lượt!", c.CustomState["thi_y"]),
						})

						if CheckThreshold(c, "thi_y", 10, true) {
							// Immediately destroy all enemies' shields if reaching 10 at start of turn
							var enemyTargets []*CombatantAuto
							isAlly := false
							for _, a := range state.Allies {
								if a.ID == c.ID {
									isAlly = true
									break
								}
							}
							if isAlly {
								enemyTargets = state.Enemies
							} else {
								enemyTargets = state.Allies
							}

							for _, e := range enemyTargets {
								if e.VirtualShield > 0 {
									e.VirtualShield = 0
								}
							}
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   c.ID,
								ActorName: c.Name,
								Action:    "SHIELD_BREAK",
								Detail:    fmt.Sprintf("[Thơ Cổ] Thi Ý viên mãn (10 tầng)! Phá vỡ hoàn toàn khiên ảo của toàn bộ kẻ địch và reset Thi Ý!"),
							})
						}
					}

					// Tiên Cổ Ốc: Thần Đậu Cung Sinh Cơ (DIVINE_BEAN_ABUNDANCE)
					if actorHasMechanic(c, "ON_TURN_START", "DIVINE_BEAN_ABUNDANCE") {
						isAlly := false
						for _, a := range state.Allies {
							if a.ID == c.ID {
								isAlly = true
								break
							}
						}
						var team []*CombatantAuto
						if isAlly {
							team = state.Allies
						} else {
							team = state.Enemies
						}
						for _, mate := range team {
							if mate != nil && mate.HP > 0 {
								cleansed := false
								newEffects := make([]Effect, 0, len(mate.ActiveEffects))
								for _, eff := range mate.ActiveEffects {
									if !cleansed && eff.Type == "debuff" {
										cleansed = true
										continue
									}
									newEffects = append(newEffects, eff)
								}
								mate.ActiveEffects = newEffects

								healGain := mate.MaxHP * 0.10
								mate.HP += healGain
								if mate.HP > mate.MaxHP {
									mate.HP = mate.MaxHP
								}
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   c.ID,
									ActorName: c.Name,
									Action:    "DIVINE_BEAN_ABUNDANCE",
									TargetId:  mate.ID,
									Detail:    fmt.Sprintf("[Tiên Cổ Ốc - Thần Đậu Cung] Thần Đậu nảy mầm tỏa sinh cơ cuồn cuộn, hồi phục %.0f HP và thanh tẩy dị trạng cho %s!", healGain, mate.Name),
								})
							}
						}
					}

					// Sát Trận: Lục Cực Tuyệt Sát Trận (SIX_EXTREMITIES_ARRAY)
					if actorHasMechanic(c, "ON_TURN_START", "SIX_EXTREMITIES_ARRAY") {
						isAlly := false
						for _, a := range state.Allies {
							if a.ID == c.ID {
								isAlly = true
								break
							}
						}
						var opponents []*CombatantAuto
						if isAlly {
							opponents = state.Enemies
						} else {
							opponents = state.Allies
						}
						for _, enemy := range opponents {
							if enemy != nil && enemy.HP > 0 {
								dmg := c.MaxHP*0.05 + float64(turn)*c.MaxHP*0.02
								enemy.HP -= dmg
								if enemy.HP < 0 {
									enemy.HP = 0
								}
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   c.ID,
									ActorName: c.Name,
									Action:    "SIX_EXTREMITIES_ARRAY",
									TargetId:  enemy.ID,
									Damage:    dmg,
									Detail:    fmt.Sprintf("[Sát Trận - Lục Cực Tuyệt Sát] Trận bàn xoay chuyển sát phạt kinh thiên, giáng %.0f sát thương tuyệt diệt lên %s!", dmg, enemy.Name),
								})
							}
						}
					}

					// Tiên Cổ Ốc: Vạn Niên Đấu Phi Thuyền - Nhật Chiếu Phi Kiếm chi viện mỗi lượt
					if actorHasMechanic(c, "ON_TURN_START", "MYRIAD_YEAR_WARSHIP") {
						isAlly := false
						for _, a := range state.Allies {
							if a.ID == c.ID {
								isAlly = true
								break
							}
						}
						var opponents []*CombatantAuto
						if isAlly {
							opponents = state.Enemies
						} else {
							opponents = state.Allies
						}
						for _, enemy := range opponents {
							if enemy != nil && enemy.HP > 0 {
								swordDmg := c.BaseDmg * 0.8
								if swordDmg < 30.0 {
									swordDmg = 30.0
								}
								enemy.HP -= swordDmg
								if enemy.HP < 0 {
									enemy.HP = 0
								}
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   c.ID,
									ActorName: c.Name,
									Action:    "MYRIAD_YEAR_WARSHIP",
									TargetId:  enemy.ID,
									Damage:    swordDmg,
									Detail:    fmt.Sprintf("[Tiên Cổ Ốc - Vạn Niên Đấu Phi Thuyền] Mũi phi kiếm Nhật Chiếu phóng ra từ boong tàu, chém xuyên phòng tuyến gây %.0f sát thương lên %s!", swordDmg, enemy.Name),
								})
								break
							}
						}
					}

					// Sát Chiêu: Diêm Đế / Quỷ Quan Y (YAMA_EMPEROR)
					if actorHasMechanic(c, "ON_TURN_START", "YAMA_EMPEROR") || hasStatus(c, statusfx.StatusYamaEmperor) {
						isAlly := false
						for _, a := range state.Allies {
							if a.ID == c.ID {
								isAlly = true
								break
							}
						}
						var opponents []*CombatantAuto
						if isAlly {
							opponents = state.Enemies
						} else {
							opponents = state.Allies
						}
						for _, enemy := range opponents {
							if enemy != nil && enemy.HP > 0 {
								soulLeech := c.BaseDmg * 0.75
								if soulLeech < 30.0 {
									soulLeech = 30.0
								}
								enemy.HP -= soulLeech
								if enemy.HP < 0 {
									enemy.HP = 0
								}
								c.HP += soulLeech
								if c.HP > c.MaxHP {
									c.HP = c.MaxHP
								}
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   c.ID,
									ActorName: c.Name,
									Action:    "YAMA_EMPEROR",
									TargetId:  enemy.ID,
									Damage:    soulLeech,
									Detail:    fmt.Sprintf("[Hồn Đạo - Diêm Đế] Diêm La Tử lao ra từ Quỷ Quan Y cắn nuốt %.0f Hồn Lực của %s hồi phục cho Diêm Đế!", soulLeech, enemy.Name),
								})
								break
							}
						}
					}

					// Tống Hữu Chi Phong phản phệ (StatusFarewellCurse)
					if hasStatus(c, statusfx.StatusFarewellCurse) {
						curseDmg := c.MaxHP * 0.05
						if curseDmg < 10.0 {
							curseDmg = 10.0
						}
						c.HP -= curseDmg
						if c.HP < 0 {
							c.HP = 0
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   c.ID,
							ActorName: c.Name,
							Action:    "FAREWELL_CURSE",
							Damage:    curseDmg,
							Detail:    fmt.Sprintf("[Phong Đạo - Tống Hữu Phong] Gió Tiễn Bạn vần vũ ăn mòn cốt nhục và hồn phách, %s tự tổn %.0f HP!", c.Name, curseDmg),
						})
					}

					// Kiếm Ngân Tác Mệnh phát tác (StatusSwordMark)
					if hasStatus(c, statusfx.StatusSwordMark) {
						markStacks, _ := statusStacks(c, statusfx.StatusSwordMark)
						if markStacks < 1 {
							markStacks = 1
						}
						markDmg := float64(markStacks) * 0.05 * c.MaxHP
						if markDmg < 20 {
							markDmg = 20
						}
						c.HP -= markDmg
						if c.HP <= c.MaxHP*0.25 {
							c.HP = 0
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   c.ID,
								ActorName: c.Name,
								Action:    "SWORD_MARK_EXECUTE",
								Damage:    markDmg,
								Detail:    fmt.Sprintf("[Kiếm Đạo - Kiếm Ngân Tác Mệnh] %d đạo kiếm ngân trong cơ thể %s phát tác xé toạc lục phủ ngũ tạng, trảm sát tức khắc!", markStacks, c.Name),
							})
						} else {
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   c.ID,
								ActorName: c.Name,
								Action:    "SWORD_MARK",
								Damage:    markDmg,
								Detail:    fmt.Sprintf("[Kiếm Đạo - Kiếm Ngân Tác Mệnh] %d đạo kiếm ngân róc thịt gọt xương, %s chịu %.0f sát thương kiếm ngân!", markStacks, c.Name, markDmg),
							})
						}
					}

					// Khô Mộc Phùng Xuân (StatusDryWoodSpring)
					if hasStatus(c, statusfx.StatusDryWoodSpring) {
						springHeal := c.MaxHP * 0.15
						c.HP = math.Min(c.MaxHP, c.HP+springHeal)
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   c.ID,
							ActorName: c.Name,
							Action:    "DRY_WOOD_SPRING",
							Detail:    fmt.Sprintf("[Mộc Đạo - Khô Mộc Phùng Xuân] Sinh cơ nảy lộc từ gốc mục, %s hồi phục %.0f HP!", c.Name, springHeal),
						})
					}
				}
			}

			// ── Virtual mode: restore mana for all combatants each turn ──
			if state.VirtualMode {
				for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
					for _, c := range combatants {
						if c.HP <= 0 {
							continue
						}
						c.Mana = c.MaxMana
					}
				}
			}

			if initiative != nil {
				for _, side := range [][]*CombatantAuto{state.Allies, state.Enemies} {
					for _, actor := range side {
						for key, remaining := range actor.ActionCooldowns {
							if remaining > 0 {
								actor.ActionCooldowns[key] = remaining - 1
							}
						}
					}
				}
			}
		}

		// ── Process actors ──
		var allActors []*CombatantAuto
		allActors = append(allActors, state.Allies...)
		allActors = append(allActors, state.Enemies...)

		sort.SliceStable(allActors, func(i, j int) bool {
			return allActors[i].EffectiveSpeed > allActors[j].EffectiveSpeed
		})

		// ── Thỉnh Đạo Hữu Tiên Hành Cổ: seize the opening turn ──
		// Once, on turn 1, the seizer moves ahead of everyone else (enemies act
		// after it, in normal speed order). Not a lockout.
		if turn == 1 {
			seizer, suppressed := seizeInitiative(state, allActors)
			for _, name := range suppressed {
				turnLogs = append(turnLogs, CombatLog{
					Turn:   turn,
					Action: "PRIORITY_SUPPRESSED",
					Detail: fmt.Sprintf("Cấm Cổ trấn áp tiên thủ của %s!", name),
				})
			}
			if seizer != nil {
				if initiative.Prioritize(seizer.ID) {
					activeActorID = seizer.ID
				}
				reordered := make([]*CombatantAuto, 0, len(allActors))
				reordered = append(reordered, seizer)
				for _, a := range allActors {
					if a != seizer {
						reordered = append(reordered, a)
					}
				}
				allActors = reordered
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   seizer.ID,
					ActorName: seizer.Name,
					Action:    "SEIZE_INITIATIVE",
					Detail:    fmt.Sprintf("[Thỉnh Đạo Hữu Tiên Hành] Thọ nguyên đối thủ chưa gấp ba lần đạo hữu — %s nắm quyền tiên thủ!", seizer.Name),
				})
			} else {
				// ── Cấm Cổ: trấn áp thành công thì người mang đi trước ──
				// A suppression that actually negates an enemy priority
				// effect seizes the opening for its holder: the fastest
				// living suppressor moves ahead of everyone. Mirror (both
				// sides succeed): the faster suppressor goes first.
				suppA := successfulSuppressor(state, allActors, true)
				suppB := successfulSuppressor(state, allActors, false)
				winner := suppA
				mirror := suppA != nil && suppB != nil
				if suppB != nil && (suppA == nil || suppB.EffectiveSpeed > suppA.EffectiveSpeed) {
					winner = suppB
				}
				if winner != nil {
					if initiative.Prioritize(winner.ID) {
						activeActorID = winner.ID
					}
					reordered := make([]*CombatantAuto, 0, len(allActors))
					reordered = append(reordered, winner)
					for _, a := range allActors {
						if a != winner {
							reordered = append(reordered, a)
						}
					}
					allActors = reordered
					detail := fmt.Sprintf("Cấm Cổ trấn áp ưu tiên của địch — %s giành quyền đi trước!", winner.Name)
					if mirror {
						detail = fmt.Sprintf("Song Cấm đối kháng — %s nhanh hơn, giành quyền đi trước!", winner.Name)
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   winner.ID,
						ActorName: winner.Name,
						Action:    "SUPPRESS_PRIORITY",
						Detail:    detail,
					})
				}
			}
		}
		if initiative != nil {
			for _, actor := range allActors {
				if actor.ID == activeActorID {
					allActors = []*CombatantAuto{actor}
					break
				}
			}
		}

		// Pre-compute confuse results once per turn to avoid RNG drift.
		confusedResult := make(map[*CombatantAuto]bool)
		for _, a := range allActors {
			if a.IsConfused {
				confusedResult[a] = rng.Float64() < 0.40
			}
		}

		// transitions for the opposing side after each executed action.
		// attacker lets the Bất Tử floor know whether Sát Khí is piercing; nil for
		// self-inflicted or non-combat damage.
		postActionChecks := func(opposing []*CombatantAuto, attacker *CombatantAuto) {
			for _, tgt := range opposing {
				if tgt.HP <= 0 && !tgt.IsDead {
					// Catch-all for any HP write site that did not apply the Bất Tử
					// floor itself. Idempotent: guarded sites already clamped.
					if applyLastStandFloor(tgt, satKhiPiercesImmunity(attacker)) {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   tgt.ID,
							ActorName: tgt.Name,
							Action:    "LAST_STAND_FLOORED",
							Detail:    fmt.Sprintf("%s suýt gục ngã nhưng [Bất Tử] giữ lại 1 HP!", tgt.Name),
						})
						continue
					}
					tgt.HP = 0
					tgt.IsDead = true
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   tgt.ID,
						ActorName: tgt.Name,
						Action:    "DIED",
						Detail:    fmt.Sprintf("%s đã gục ngã!", tgt.Name),
					})
				}
			}
			for _, tgt := range opposing {
				if tgt.Kind == "Boss" {
					turnLogs = append(turnLogs, CheckBossPhaseTransitions(state, tgt, turn)...)
				}
			}
		}

		actionRunner := &combatActionRunner{
			state: state, rng: rng, turn: turn, turnLogs: &turnLogs, summonCount: &summonCount,
		}

		// executePlannedAction is the single action executor for both ordinary
		// turns and paired PvP slots. Controllers may differ in how they choose
		// actions; they must not differ in how an accepted action is resolved.
		executePlannedAction := func(actor *CombatantAuto, item selectedCombatAction, isAlly bool, enemyTargets []*CombatantAuto, simultaneous, eligibleAtSlotStart bool) bool {
			if simultaneous && !eligibleAtSlotStart {
				return false
			}
			if !simultaneous && (actor.HP <= 0 || actor.IsDead || actor.IsStunned || actor.IsFrozen || actor.IsSleeping || actor.IsDreamPrison || actor.IsBound) {
				return false
			}
			action := *item.action
			if item.target != nil {
				action.TargetX, action.TargetY = item.target.GridX, item.target.GridY
			}
			if action.ActionType == "FLEE" {
				chance := FleeChance(actor, enemyTargets)
				if rng.Float64() < chance {
					state.FledBy = actor.ID
					turnLogs = append(turnLogs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "FLEE_SUCCESS", Detail: fmt.Sprintf("%s bỏ chạy thành công (tỉ lệ %.0f%%).", actor.Name, chance*100)})
					return true
				}
				turnLogs = append(turnLogs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "FLEE_FAILED", Detail: fmt.Sprintf("%s bỏ chạy thất bại (tỉ lệ %.0f%%), mất lượt này.", actor.Name, chance*100)})
				return false
			}
			if action.WarmupTurns > 0 {
				charged := action
				actor.ChargingAction, actor.ChargingTurnsLeft = &charged, action.WarmupTurns
				turnLogs = append(turnLogs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "START_CHARGING", Detail: fmt.Sprintf("%s bắt đầu tụ khí [%s].", actor.Name, action.Name)})
				return false
			}
			targets := enemyTargets
			if item.target != nil && item.target.ID != actor.ID && (item.target.HP <= 0 || item.target.IsDead) {
				targets = nil
			}
			actionRunner.runOne(actor, &action, isAlly, targets)
			postActionChecks(enemyTargets, actor)
			return false
		}

		// ── Pre-Attack Defense Pass ──
		// All alive combatants execute pure defensive actions (shield buffs) before attacks land.
		executedInPrePass := make(map[*CombatantAuto]map[int]bool)

		if initiative == nil {
			for _, actor := range allActors {
				if actor.HP <= 0 || actor.IsDead {
					continue
				}

				// Skip if CC'd
				isDisabled := false
				for _, eff := range actor.ActiveEffects {
					if (strings.EqualFold(eff.ID, EffectStun) || strings.EqualFold(eff.ID, statusfx.StatusStun)) && eff.Duration >= 0 {
						isDisabled = true
						break
					}
				}
				if actor.IsStunned || actor.IsFrozen || actor.IsSleeping || actor.IsBound || actor.IsSilenced || actor.IsFeared {
					isDisabled = true
				}
				if isDisabled {
					continue
				}

				// Determine target side
				isAlly := false
				for _, a := range state.Allies {
					if a.ID == actor.ID {
						isAlly = true
						break
					}
				}
				var enemyTargets []*CombatantAuto
				if isAlly {
					enemyTargets = state.Enemies
				} else {
					enemyTargets = state.Allies
				}

				if actor.IsCharmed {
					if isAlly {
						enemyTargets = state.Allies
					} else {
						enemyTargets = state.Enemies
					}
				}

				for ai := 0; ai < len(actor.Actions); ai++ {
					a := &actor.Actions[ai]
					if (a.Defense > 0 || a.DefUpTier > 0 || GetSimActionCategory(*a) == "shield") && a.Damage == 0 {
						var primaryTgt *CombatantAuto
						if len(enemyTargets) > 0 {
							primaryTgt = enemyTargets[0]
						}
						if checkActionCondition(a, actor, primaryTgt) {
							out := actionRunner.runOne(actor, a, isAlly, enemyTargets)
							postActionChecks(enemyTargets, actor)
							if out.executed {
								if executedInPrePass[actor] == nil {
									executedInPrePass[actor] = make(map[int]bool)
								}
								executedInPrePass[actor][ai] = true
							}
							if out.abort || actor.HP <= 0 || actor.IsDead {
								break
							}
						}
					}
				}
			}
		}

		// ── Sequential Combatant Turn Loop ──
		ccCheckedThisTurn := make(map[*CombatantAuto]bool)
		ccDisabledThisTurn := make(map[*CombatantAuto]bool)

		for _, actor := range allActors {
			if actor.HP <= 0 || actor.IsDead {
				continue
			}

			// Check CC and Auto-restores once per turn on actor's turn
			if !ccCheckedThisTurn[actor] {
				ccCheckedThisTurn[actor] = true

				if initiative == nil && actor.ActionCooldowns != nil {
					for k, cd := range actor.ActionCooldowns {
						if cd > 0 {
							actor.ActionCooldowns[k] = cd - 1
						}
					}
				}

				// Check stun
				isStunned := actor.IsStunned
				if !isStunned {
					for _, eff := range actor.ActiveEffects {
						if (strings.EqualFold(eff.ID, EffectStun) || strings.EqualFold(eff.ID, statusfx.StatusStun)) && eff.Duration >= 0 {
							isStunned = true
							break
						}
					}
				}
				if isStunned {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "STUNNED",
						Detail:    fmt.Sprintf("%s bị choáng, bỏ lượt!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				} else if actor.IsFrozen {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "FROZEN",
						Detail:    fmt.Sprintf("%s bị đóng băng, bỏ lượt!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				} else if actor.IsSleeping {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SLEEPING",
						Detail:    fmt.Sprintf("%s đang ngủ, bỏ lượt!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				} else if actor.IsDreamPrison {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "DREAM_PRISON",
						Detail:    fmt.Sprintf("[Mộng Đạo - Trầm Mộng] %s đang chìm sâu trong Mộng Cảnh, hoàn toàn mất quyền hành động!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				} else if actor.IsBound {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "BOUND",
						Detail:    fmt.Sprintf("%s bị trói, không thể hành động!", actor.Name),
					})
					ccDisabledThisTurn[actor] = true
				}

				if ccDisabledThisTurn[actor] && actor.ChargingAction != nil {
					backfireDmg := actor.ChargingAction.BackfireDamage
					if backfireDmg <= 0 {
						backfireDmg = math.Round(actor.MaxHP * 0.15)
					}
					actor.HP -= backfireDmg
					if actor.HP < 0 {
						actor.HP = 0
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "CHANNEL_INTERRUPTED",
						Damage:    backfireDmg,
						Detail:    fmt.Sprintf("[Đứt Đoạn Tụ Khí] %s bị khống chế khi đang ngưng tụ sát chiêu [%s], bị chân nguyên phản phệ mất %.0f HP!", actor.Name, actor.ChargingAction.Name, backfireDmg),
					})
					actor.ChargingAction = nil
					actor.ChargingTurnsLeft = 0
				}

				if !ccDisabledThisTurn[actor] {
					// Auto-use healing items
					if actor.AutoHealHP && actor.HP > 0 && actor.HealCharges > 0 && actor.ItemsUsedInBattle < 3 {
						hpPct := actor.HP / actor.MaxHP * 100
						if hpPct <= actor.HealThreshold {
							baseHeal := actor.HealPotency
							healVal := baseHeal * effectiveHealingReceived(actor)
							actor.HP += healVal
							if actor.HP > actor.MaxHP {
								actor.HP = actor.MaxHP
							}
							actor.HealCharges--
							actor.ItemsUsedInBattle++
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "ITEM_HEAL",
								Damage:    healVal,
								Detail:    fmt.Sprintf("%s tự động dùng dược liệu hồi phục %.0f HP! (đã dùng %d/3)", actor.Name, healVal, actor.ItemsUsedInBattle),
								Breakdown: &DamageBreakdown{
									HealingBase:   baseHeal,
									HealingFinal:  healVal,
									HealerHealing: actor.StatProvenance["healing_received"],
								},
							})
						}
					}
				}
			}

			if ccDisabledThisTurn[actor] {
				continue
			}

			// Determine target side
			isAlly := false
			for _, a := range state.Allies {
				if a.ID == actor.ID {
					isAlly = true
					break
				}
			}

			var enemyTargets []*CombatantAuto
			if isAlly {
				enemyTargets = state.Enemies
			} else {
				enemyTargets = state.Allies
			}

			if actor.IsCharmed {
				if isAlly {
					enemyTargets = state.Allies
				} else {
					enemyTargets = state.Enemies
				}
			}

			if actor.IsConfused && confusedResult[actor] {
				if isAlly {
					enemyTargets = state.Allies
				} else {
					enemyTargets = state.Enemies
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "CONFUSED",
					Detail:    fmt.Sprintf("%s bị mê loạn, tấn công nhầm đồng minh!", actor.Name),
				})
			}

			// Non-damaging support actions (shield, heal, buff) resolve without ending the turn pass.
			// The turn pass ends only when an attack / damage action (Damage > 0 or BASIC_ATTACK) is executed.
			executedThisTurn := make([]bool, len(actor.Actions))
			if preMap, ok := executedInPrePass[actor]; ok {
				for ai, done := range preMap {
					if done && ai < len(executedThisTurn) {
						executedThisTurn[ai] = true
					}
				}
			}
			executedDamageThisTurn := false

			if actor.ChargingAction != nil {
				actor.ChargingTurnsLeft--
				if actor.ChargingTurnsLeft <= 0 {
					chargedAct := actor.ChargingAction
					actor.ChargingAction = nil
					actor.ChargingTurnsLeft = 0
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "WARMUP_COMPLETE",
						Detail:    fmt.Sprintf("[Tụ Khí Hoàn Tất] %s đã ngưng tụ xong sát chiêu [%s], bộc phát kinh thiên động địa!", actor.Name, chargedAct.Name),
					})
					out := actionRunner.runOne(actor, chargedAct, isAlly, enemyTargets)
					postActionChecks(enemyTargets, actor)
					if out.abort {
						break
					}
					continue
				} else {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "CHARGING",
						Detail:    fmt.Sprintf("[Tụ Khí] %s tiếp tục ngưng tụ chân nguyên cho sát chiêu [%s] (còn %d hiệp)!", actor.Name, actor.ChargingAction.Name, actor.ChargingTurnsLeft),
					})
					continue
				}
			}

			if initiative != nil {
				control := state.Interactive
				var chosen *SimAction
				var target *CombatantAuto
				var selectedPlan []selectedCombatAction
				if control != nil && control.ManualActors[actor.ID] {
					if control.Consumed >= len(control.Commands) {
						control.PendingActor = actor.ID
						logs = append(logs, turnLogs...)
						logs = append(logs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "AWAIT_COMMAND", Snapshot: buildSnapshot(state)})
						return logs
					}
					command := control.Commands[control.Consumed]
					if hasPairedPendingPlans(control) {
						firstCommand, secondCommand := control.Commands[control.Consumed], control.Commands[control.Consumed+1]
						findActor := func(id string) (*CombatantAuto, bool) {
							for _, candidate := range state.Allies {
								if candidate != nil && candidate.ID == id {
									return candidate, true
								}
							}
							for _, candidate := range state.Enemies {
								if candidate != nil && candidate.ID == id {
									return candidate, false
								}
							}
							return nil, false
						}
						firstActor, firstIsAlly := findActor(firstCommand.ActorID)
						secondActor, secondIsAlly := findActor(secondCommand.ActorID)
						if firstActor == nil || secondActor == nil {
							control.Error = "locked PvP actor is missing from the combat roster"
							logs = append(logs, turnLogs...)
							return logs
						}
						planFor := func(pairedActor *CombatantAuto, pairedCommand CombatCommand, pairedIsAlly bool) ([]selectedCombatAction, error) {
							targets := state.Allies
							if pairedIsAlly {
								targets = state.Enemies
							}
							if pairedCommand.ActionType == "AUTO" {
								return BuildAutomaticCombatPlan(pairedActor, targets, state.MaxGuActionsPerTurn), nil
							}
							return resolveLockedCombatPlan(pairedActor, targets, pairedCommand.Plan)
						}
						firstPlan, firstErr := planFor(firstActor, firstCommand, firstIsAlly)
						secondPlan, secondErr := planFor(secondActor, secondCommand, secondIsAlly)
						if firstErr != nil || secondErr != nil {
							if firstErr != nil {
								control.Error = firstErr.Error()
							} else {
								control.Error = secondErr.Error()
							}
							logs = append(logs, turnLogs...)
							return logs
						}
						control.Consumed += 2
						type pairedAction struct {
							actor   *CombatantAuto
							isAlly  bool
							targets []*CombatantAuto
							item    selectedCombatAction
						}
						selfActions := make([]pairedAction, 0)
						firstTargetActions, secondTargetActions := make([]pairedAction, 0), make([]pairedAction, 0)
						addPlan := func(planActor *CombatantAuto, planAlly bool, plan []selectedCombatAction, actorTargets *[]pairedAction) {
							targets := state.Allies
							if planAlly {
								targets = state.Enemies
							}
							for _, item := range plan {
								entry := pairedAction{actor: planActor, isAlly: planAlly, targets: targets, item: item}
								if item.target != nil && item.target.ID == planActor.ID {
									selfActions = append(selfActions, entry)
								} else {
									*actorTargets = append(*actorTargets, entry)
								}
							}
						}
						addPlan(firstActor, firstIsAlly, firstPlan, &firstTargetActions)
						addPlan(secondActor, secondIsAlly, secondPlan, &secondTargetActions)
						resolvePlanAction := func(entry pairedAction, simultaneous, eligibleAtSlotStart bool) bool {
							return executePlannedAction(entry.actor, entry.item, entry.isAlly, entry.targets, simultaneous, eligibleAtSlotStart)
						}
						for _, entry := range selfActions {
							if resolvePlanAction(entry, false, false) {
								logs = append(logs, turnLogs...)
								return logs
							}
						}
						maxSlots := len(firstTargetActions)
						if len(secondTargetActions) > maxSlots {
							maxSlots = len(secondTargetActions)
						}
						for slot := 0; slot < maxSlots; slot++ {
							entries := make([]pairedAction, 0, 2)
							if slot < len(firstTargetActions) {
								entries = append(entries, firstTargetActions[slot])
							}
							if slot < len(secondTargetActions) {
								entries = append(entries, secondTargetActions[slot])
							}
							speed := func(entry pairedAction) float64 {
								if entry.actor.EffectiveSpeed > 0 {
									return entry.actor.EffectiveSpeed
								}
								return entry.actor.Speed
							}
							sameSpeed := false
							if len(entries) == 2 {
								sameSpeed = math.Abs(speed(entries[0])-speed(entries[1])) < 1e-9
								if speed(entries[1]) > speed(entries[0]) {
									entries[0], entries[1] = entries[1], entries[0]
								}
							}
							eligibleAtSlotStart := make(map[string]bool, len(entries))
							for _, entry := range entries {
								eligibleAtSlotStart[entry.actor.ID] = entry.actor.HP > 0 && !entry.actor.IsDead && !entry.actor.IsStunned && !entry.actor.IsFrozen && !entry.actor.IsSleeping && !entry.actor.IsDreamPrison && !entry.actor.IsBound
							}
							for entryIndex, entry := range entries {
								if resolvePlanAction(entry, sameSpeed && entryIndex > 0, eligibleAtSlotStart[entry.actor.ID]) {
									logs = append(logs, turnLogs...)
									return logs
								}
								// Equal-speed actions in the same slot are both committed
								// from the slot-start eligibility snapshot, so a knockout
								// caused by the first action must not cancel its peer.
								if !sameSpeed && !hasLivingCombatant(entry.targets) {
									break
								}
							}
						}
						otherID := firstActor.ID
						if otherID == activeActorID {
							otherID = secondActor.ID
						}
						initiative.Complete(otherID)
						continue
					}
					if command.ActorID != actor.ID {
						if len(control.ManualActors) < 2 || (command.ActionType != "PLAN" && command.ActionType != "AUTO") {
							control.Error = "command actor does not own the active turn"
							logs = append(logs, turnLogs...)
							return logs
						}
						// In shared PvP, the faster player holds their next activation
						// until the other locked plan reaches its own ReadyAt.
					} else if command.ActionType == "PLAN" {
						var planErr error
						selectedPlan, planErr = resolveLockedCombatPlan(actor, enemyTargets, command.Plan)
						if planErr != nil {
							control.Error = planErr.Error()
							logs = append(logs, turnLogs...)
							return logs
						}
					} else if command.ActorID == actor.ID && command.ActionType == "AUTO" {
						maxActions := state.MaxGuActionsPerTurn
						if control.MaxGuActions > 0 {
							maxActions = control.MaxGuActions
						}
						if maxActions <= 0 {
							maxActions = 3
						}
						selectedPlan = BuildAutomaticCombatPlan(actor, enemyTargets, maxActions)
						if len(selectedPlan) == 0 {
							control.Error = "no valid automatic action for active combatant"
							logs = append(logs, turnLogs...)
							return logs
						}
					} else if command.ActorID == actor.ID {
						var err error
						chosen, target, err = selectCombatCommand(actor, enemyTargets, command)
						if err != nil {
							control.Error = err.Error()
							logs = append(logs, turnLogs...)
							return logs
						}
					}
					if command.ActorID == actor.ID {
						control.Consumed++
					}
				} else {
					if state.MaxGuActionsPerTurn > 0 || (control != nil && control.MaxGuActions > 0) {
						maxActions := state.MaxGuActionsPerTurn
						if control != nil && control.MaxGuActions > 0 {
							maxActions = control.MaxGuActions
						}
						selectedPlan = BuildAutomaticCombatPlan(actor, enemyTargets, maxActions)
					} else {
						chosen, target = chooseAutoCombatAction(actor, enemyTargets)
					}
				}
				if len(selectedPlan) > 0 {
					for _, item := range selectedPlan {
						if executePlannedAction(actor, item, isAlly, enemyTargets, false, false) {
							logs = append(logs, turnLogs...)
							return logs
						}
						if !hasLivingCombatant(enemyTargets) {
							break
						}
					}
					continue
				}
				if chosen != nil {
					if chosen.ActionType == "FLEE" {
						chance := FleeChance(actor, enemyTargets)
						if rng.Float64() < chance {
							state.FledBy = actor.ID
							turnLogs = append(turnLogs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "FLEE_SUCCESS", Detail: fmt.Sprintf("%s bỏ chạy thành công (tỉ lệ %.0f%%).", actor.Name, chance*100)})
							logs = append(logs, turnLogs...)
							return logs
						}
						turnLogs = append(turnLogs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "FLEE_FAILED", Detail: fmt.Sprintf("%s bỏ chạy thất bại (tỉ lệ %.0f%%), mất lượt này.", actor.Name, chance*100)})
					} else if chosen.WarmupTurns > 0 {
						charged := *chosen
						if target != nil {
							charged.TargetX, charged.TargetY = target.GridX, target.GridY
						}
						actor.ChargingAction = &charged
						actor.ChargingTurnsLeft = chosen.WarmupTurns
						turnLogs = append(turnLogs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "START_CHARGING", Detail: fmt.Sprintf("%s bắt đầu tụ khí [%s].", actor.Name, chosen.Name)})
					} else {
						// A copy keeps the loadout's original targeting coordinates intact
						// when a player selects a different target for this activation.
						action := *chosen
						if target != nil {
							action.TargetX, action.TargetY = target.GridX, target.GridY
						}
						actionRunner.runOne(actor, &action, isAlly, enemyTargets)
						postActionChecks(enemyTargets, actor)
					}
				}
				continue
			}

			for {
				executedActionInLoop := false
				for ai := 0; ai < len(actor.Actions); ai++ {
					if executedThisTurn[ai] {
						continue
					}
					a := &actor.Actions[ai]

					var primaryTgt *CombatantAuto
					if len(enemyTargets) > 0 {
						primaryTgt = enemyTargets[0]
					}
					if !checkActionCondition(a, actor, primaryTgt) {
						continue
					}

					if a.WarmupTurns > 0 {
						actor.ChargingAction = a
						actor.ChargingTurnsLeft = a.WarmupTurns
						executedThisTurn[ai] = true
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "START_CHARGING",
							Detail:    fmt.Sprintf("[Tụ Khí Sát Chiêu] %s bắt đầu vận chuyển chân nguyên, ngưng tụ sát chiêu [%s] (cần %d hiệp tụ khí)!", actor.Name, a.Name, a.WarmupTurns),
						})
						executedDamageThisTurn = true
						break
					}

					out := actionRunner.runOne(actor, a, isAlly, enemyTargets)
					postActionChecks(enemyTargets, actor)
					if out.abort {
						executedDamageThisTurn = true
						break
					}
					if out.executed {
						executedThisTurn[ai] = true
						executedActionInLoop = true
						if out.hasDamage {
							executedDamageThisTurn = true
							break // Attack / damage action ends the turn pass!
						}
					}
				}

				if executedDamageThisTurn || !executedActionInLoop {
					break
				}
			}

			// Fallback: Basic attack if no attack / damage action was executed in this turn pass.
			// Routed through runOne so BEFORE_ENEMY_ATTACK (FORGET_INTENT/WILL),
			// backlash, mana, hunger and path-mastery behave exactly like a
			// normal action. runOne applies dao/path/crit multipliers itself,
			// hence the raw BaseDmg here.
			if !executedDamageThisTurn {
				fallbackAction := &SimAction{
					ActionType: "BASIC_ATTACK",
					Name:       "Đánh thường",
					Damage:     actor.BaseDmg,
					DamageMult: 1.0,
					ManaCost:   0,
					TargetType: "SINGLE",
					Path:       []string{"strength"},
				}
				out := actionRunner.runOne(actor, fallbackAction, isAlly, enemyTargets)
				_ = out // fallback always counts as the turn pass action
				postActionChecks(enemyTargets, actor)
			}
		}

		logs = append(logs, turnLogs...)
		if initiative != nil {
			initiative.Complete(activeActorID)
			for _, side := range [][]*CombatantAuto{state.Allies, state.Enemies} {
				for _, actor := range side {
					if actor.HP <= 0 || actor.IsDead {
						initiative.Remove(actor.ID)
						continue
					}
					ComputeEffectMultipliers(actor)
					initiative.SetSpeed(actor.ID, actor.EffectiveSpeed)
				}
			}
		}

		// ── Death triggers check (REBORN: rewind or in-place) ──
		// Allies may rewind the battle or stand back up; enemies only stand
		// back up in place — an enemy rewind would discard the fight fought.
		rewound := false
		var rewoundID string
		var rewindMechs []CombatMechanic
		tryReborn := func(c *CombatantAuto, allowRewind bool) (didRewind bool) {
			if c.HP > 0 || !hasMechanic(c.Mechanics, "ON_DEATH", "REBORN") || c.IsReviveLocked {
				return false
			}
			mechs := c.Mechanics
			maxUses := mechanicInt(mechs, "ON_DEATH", "REBORN", "max_uses", 1)
			// max_uses 0 = hồi sinh vô hạn (Xuân Thu Thiền).
			unlimited := maxUses == 0
			if maxUses < 0 {
				maxUses = 1
			}
			rewinds := allowRewind && strings.ToLower(mechanicString(mechs, "ON_DEATH", "REBORN", "mode", "rewind")) != "in_place"
			if unlimited && rewinds && rebornUses[c.ID] >= maxInfiniteRebornRewinds {
				return false
			}
			if !unlimited && rebornUses[c.ID] >= maxUses {
				return false
			}
			guaranteed := turn == 1 && mechanicBool(mechs, "ON_DEATH", "REBORN", "turn1_guaranteed", false)
			rebornChance := mechanicFloat(mechs, "ON_DEATH", "REBORN", "chance", 0)
			rebornRoll := rng.Float64()
			if !guaranteed && rebornRoll >= rebornChance {
				logs = append(logs, CombatLog{
					Turn:      turn,
					ActorId:   c.ID,
					ActorName: c.Name,
					Action:    "REBORN_MISS",
					Trigger:   "REBORN",
					Detail:    fmt.Sprintf("%s kích hoạt [Xuân Thu Thiền] thất bại: roll %.0f%% >= tỉ lệ %.0f%%!", c.Name, rebornRoll*100, rebornChance*100),
				})
				return false
			}
			rebornUses[c.ID]++
			if rewinds {
				return true
			}
			hpPct := mechanicFloat(mechs, "ON_DEATH", "REBORN", "revive_hp_percent", 0.5)
			c.HP = math.Max(1, c.MaxHP*hpPct)
			c.IsDead = false
			if mechanicBool(mechs, "ON_DEATH", "REBORN", "clear_debuffs", true) {
				var kept []Effect
				for _, eff := range c.ActiveEffects {
					if !isDispellableEffect(eff.ID) {
						kept = append(kept, eff)
					}
				}
				c.ActiveEffects = kept
				c.ActiveDoT = nil
			}
			applyRebornEmpower(c, mechs, turn, rebornUses[c.ID], &logs)
			ComputeEffectMultipliers(c)
			usesDetail := fmt.Sprintf("lần %d/%d", rebornUses[c.ID], maxUses)
			if unlimited {
				usesDetail = fmt.Sprintf("lần %d (vô hạn)", rebornUses[c.ID])
			}
			logs = append(logs, CombatLog{
				Turn:      turn,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    "TRIGGERED",
				Trigger:   "REBORN",
				Detail:    fmt.Sprintf("%s lãng quên cái chết, vùng dậy với %.0f HP (%s)!", c.Name, c.HP, usesDetail),
			})
			return false
		}
		for _, a := range state.Allies {
			if tryReborn(a, true) {
				rewound = true
				rewoundID = a.ID
				rewindMechs = a.Mechanics
				logs = append(logs, CombatLog{
					Turn:      turn,
					ActorId:   a.ID,
					ActorName: a.Name,
					Action:    "TRIGGERED",
					Trigger:   "REBORN",
					Detail:    "Xuân Thu Thiền kích hoạt! Nghịch chuyển thời không, quay về Lượt 1!",
				})
				break
			}
		}
		if !rewound {
			for _, e := range state.Enemies {
				tryReborn(e, false)
			}
		}
		if rewound {
			initialState.Turn = 0
			initialState.BossPhase = 0
			state.Allies = make([]*CombatantAuto, len(initialState.Allies))
			for i, a := range initialState.Allies {
				state.Allies[i] = cloneCombatant(a)
			}
			state.Enemies = make([]*CombatantAuto, len(initialState.Enemies))
			for i, e := range initialState.Enemies {
				state.Enemies[i] = cloneCombatant(e)
			}
			state.ApertureHP = initialState.ApertureHP
			state.ApertureMaxHP = initialState.ApertureMaxHP
			for _, na := range state.Allies {
				if na.ID == rewoundID {
					for i := 0; i < rebornUses[rewoundID]; i++ {
						applyRebornEmpower(na, rewindMechs, turn, i+1, &logs)
					}
					break
				}
			}
			turn = 0 // loops will increment it to 1
			continue
		}

		if state.CaptureTurnSnapshots {
			// Persist one full battlefield snapshot per turn for fights that may be
			// written to combat history, without attaching state to every event.
			logs = append(logs, CombatLog{
				Turn:     turn,
				Action:   "TURN_END",
				Detail:   fmt.Sprintf("Kết thúc lượt %d.", turn),
				Snapshot: buildSnapshot(state),
			})
		}

		// ── Win/Loss check ──
		alliesAlive := false
		for _, a := range state.Allies {
			if a.HP > 0 {
				alliesAlive = true
				break
			}
		}
		enemiesAlive := false
		for _, e := range state.Enemies {
			if e.HP > 0 {
				enemiesAlive = true
				break
			}
		}

		if !alliesAlive || !enemiesAlive {
			winner := "Phe Địch"
			if alliesAlive {
				winner = "Phe Ta"
			}
			endSnap := buildSnapshot(state)
			logs = append(logs, CombatLog{
				Turn:     turn,
				Action:   "BATTLE_END",
				Detail:   fmt.Sprintf("Trận chiến kết thúc. %s chiến thắng!", winner),
				Snapshot: endSnap,
			})
			break
		}

		if turn == maxTurns {
			endSnap := buildSnapshot(state)
			logs = append(logs, CombatLog{
				Turn:     turn,
				Action:   "BATTLE_END",
				Detail:   "Hết thời gian! Người thách đấu đã thua.",
				Snapshot: endSnap,
			})
		}
	}

	return logs
}

func hasLivingCombatant(combatants []*CombatantAuto) bool {
	for _, combatant := range combatants {
		if combatant != nil && combatant.HP > 0 && !combatant.IsDead {
			return true
		}
	}
	return false
}

func hasPairedPendingPlans(control *InteractiveCombat) bool {
	if control == nil || len(control.ManualActors) < 2 || control.Consumed+1 >= len(control.Commands) {
		return false
	}
	first, second := control.Commands[control.Consumed], control.Commands[control.Consumed+1]
	return first.ActorID != second.ActorID && control.ManualActors[first.ActorID] && control.ManualActors[second.ActorID] &&
		(first.ActionType == "PLAN" || first.ActionType == "AUTO") && (second.ActionType == "PLAN" || second.ActionType == "AUTO")
}

// buildSnapshot creates a CombatSnapshot from the current state.
func buildSnapshot(state *CombatState) *CombatSnapshot {
	snap := &CombatSnapshot{Turn: state.Turn, ApertureHP: state.ApertureHP, ApertureMaxHP: state.ApertureMaxHP}
	for _, a := range state.Allies {
		damageMultiplier := snapshotMultiplier(a.DamageMultiplier)
		defenseMultiplier := snapshotMultiplier(a.DefenseMultiplier)
		snap.Allies = append(snap.Allies, CombatantSnapshot{
			ID: a.ID, Name: a.Name, Kind: a.Kind,
			HP: a.HP, MaxHP: a.MaxHP, Mana: a.Mana, MaxMana: a.MaxMana,
			GridX: a.GridX, GridY: a.GridY, Shield: a.VirtualShield,
			Damage: a.BaseDmg * damageMultiplier, Defense: a.BaseDef * defenseMultiplier,
			Speed: a.EffectiveSpeed, IsDead: a.IsDead, Statuses: snapshotStatuses(a),
		})
	}
	for _, e := range state.Enemies {
		damageMultiplier := snapshotMultiplier(e.DamageMultiplier)
		defenseMultiplier := snapshotMultiplier(e.DefenseMultiplier)
		snap.Enemies = append(snap.Enemies, CombatantSnapshot{
			ID: e.ID, Name: e.Name, Kind: e.Kind,
			HP: e.HP, MaxHP: e.MaxHP, Mana: e.Mana, MaxMana: e.MaxMana,
			GridX: e.GridX, GridY: e.GridY, Shield: e.VirtualShield,
			Damage: e.BaseDmg * damageMultiplier, Defense: e.BaseDef * defenseMultiplier,
			Speed: e.EffectiveSpeed, IsDead: e.IsDead, Statuses: snapshotStatuses(e),
		})
	}
	return snap
}

func snapshotMultiplier(multiplier float64) float64 {
	if multiplier == 0 {
		return 1
	}
	return multiplier
}

func snapshotStatuses(c *CombatantAuto) []CombatStatusSnapshot {
	statuses := make([]CombatStatusSnapshot, 0, len(c.ActiveEffects)+len(c.ActiveDoT))
	for _, effect := range c.ActiveEffects {
		statuses = append(statuses, CombatStatusSnapshot{
			ID: effect.ID, Type: effect.Type, Tier: effect.Tier,
			Value: float64(effect.Value), RemainingTurns: effect.Duration,
		})
	}
	for _, effect := range c.ActiveDoT {
		statuses = append(statuses, CombatStatusSnapshot{
			ID: effect.Type, Type: "dot", Value: effect.Value,
			RemainingTurns: effect.Turns, DirectHP: effect.DirectHP,
		})
	}
	return statuses
}

func applyThresholdRecoveries(target *CombatantAuto, turn int, logs *[]CombatLog) {
	if target == nil || target.MaxHP <= 0 || len(target.BeforeHitRecoveries) == 0 {
		return
	}
	for _, recovery := range target.BeforeHitRecoveries {
		if target.HP > target.MaxHP*recovery.HPThreshold {
			continue
		}
		if target.CustomState == nil {
			target.CustomState = make(map[string]int)
		}
		stateKey := "threshold_recovery:" + recovery.SourceID + ":" + recovery.OperationID
		if target.CustomState[stateKey] >= recovery.MaxUses {
			continue
		}
		target.CustomState[stateKey]++
		heal := target.MaxHP * recovery.HealPercent
		beforeHeal := target.HP
		target.HP = math.Min(target.MaxHP, target.HP+heal)
		healed := target.HP - beforeHeal
		cleansed := 0
		if recovery.CleanseDebuffs {
			kept := make([]Effect, 0, len(target.ActiveEffects))
			for _, effect := range target.ActiveEffects {
				if effect.Type == "debuff" {
					cleansed++
					continue
				}
				kept = append(kept, effect)
			}
			target.ActiveEffects = kept
		}
		message := strings.NewReplacer(
			"{actor}", target.Name,
			"{heal_amount}", fmt.Sprintf("%.0f", healed),
			"{cleansed_count}", fmt.Sprintf("%d", cleansed),
		).Replace(recovery.Message)
		*logs = append(*logs, CombatLog{
			Turn: turn, ActorId: target.ID, ActorName: target.Name,
			Action: recovery.LogAction, Detail: message,
		})
	}
}

func applyDamageToTarget(state *CombatState, actor, target *CombatantAuto, damage float64, logs *[]CombatLog, turn int, shieldBypass bool, guaranteedHit bool, spillToAperture bool, rng *rand.Rand) float64 {
	if damage < 1 {
		damage = 1
	}

	// Định Tiên Du first turn evade: dodges the FIRST strike on turn 1 only.
	// Multi-hit/AoE follow-ups land normally, as do percent riders attached
	// to the evaded strike (skipped via evadedThisStrike in runOne).
	// Cấm Cổ suppresses this priority effect: the strike lands in normal order.
	if turn == 1 && hasMechanic(target.Mechanics, "BEFORE_HIT", "FIRST_TURN_EVADE") && (suppressesPriority(actor) || guaranteedHit) {
		reason := "Cấm Cổ trấn áp"
		if guaranteedHit {
			reason = "Đòn đánh Tất Trúng phá vỡ"
		}
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "PRIORITY_SUPPRESSED",
			Detail:    fmt.Sprintf("%s [Định Tiên Du] của %s — đòn đánh trúng như bình thường!", reason, target.Name),
		})
	} else if turn == 1 && hasMechanic(target.Mechanics, "BEFORE_HIT", "FIRST_TURN_EVADE") {
		if target.CustomState == nil {
			target.CustomState = make(map[string]int)
		}
		if target.CustomState["first_turn_evade_used"] != turn {
			target.CustomState["first_turn_evade_used"] = turn
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "EVADE",
				Detail:    fmt.Sprintf("%s kích hoạt Định Tiên Du, né tránh đòn đánh đầu tiên hoàn toàn!", target.Name),
			})
			return 0
		}
	}

	// Nghịch Lưu Hộ Thân Quỹ: Phản phệ hoàn toàn 100% sát thương ngược lại kẻ tấn công!
	if (actorHasMechanic(target, "BEFORE_HIT", "REVERSE_FLOW_SEAL") ||
		actorHasMechanic(target, "COMBAT_PASSIVE", "REVERSE_FLOW_SEAL") ||
		hasStatus(target, statusfx.StatusReverseFlowSeal)) && actor != target && (target.CustomState == nil || target.CustomState["reflecting_reverse_flow"] != 1) {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "REVERSE_FLOW_SEAL",
			Detail:    fmt.Sprintf("[Sát Chiêu - Nghịch Lưu Hộ Thân Quỹ] Nghịch Lưu Hà cuồn cuộn phản phệ hoàn toàn %.0f sát thương ngược lại %s!", damage, actor.Name),
		})
		if target.CustomState == nil {
			target.CustomState = make(map[string]int)
		}
		target.CustomState["reflecting_reverse_flow"] = 1
		applyDamageToTarget(state, target, actor, damage, logs, turn, false, false, false, rng)
		target.CustomState["reflecting_reverse_flow"] = 0
		return 0
	}

	// Long Cung Ngự Hồn: 35% xác suất làm kẻ tấn công bị Mê Loạn trước khi đánh trúng
	if actorHasMechanic(target, "BEFORE_HIT", "DRAGON_PALACE_DOMINION") && actor != target {
		if rng.Float64() < 0.35 {
			actor.ActiveEffects = append(actor.ActiveEffects, Effect{
				ID: statusfx.StatusConfuse, Duration: 2,
			})
			actor.IsConfused = true
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "DRAGON_PALACE_DOMINION",
				Detail:    fmt.Sprintf("[Long Cung Ngự Hồn] Khói Mộng Điệp tràn ngập làm %s hoa mắt mê loạn trong 2 hiệp!", actor.Name),
			})
		}
	}

	// Cửu Cửu Liên Hoàn Trận: Hấp thụ 50% sát thương nhận vào
	if hasStatus(target, statusfx.StatusNinetyNineShield) || actorHasMechanic(target, "BEFORE_HIT", "NINETY_NINE_ARRAY") {
		mitigated := damage * 0.50
		damage -= mitigated
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "NINETY_NINE_ARRAY",
			Detail:    fmt.Sprintf("[Cửu Cửu Liên Hoàn] Trùng điệp hộ thuẫn hấp thụ 50%% sát thương (triệt tiêu %.0f sát thương)!", mitigated),
		})
	}

	// Typed BEFORE_HIT mitigation runs at the same point as the former
	// BLOOD_STAINED_ROBE hook, after the regional absorption rule. If both typed
	// and legacy forms are present, typed data owns that rule exactly once.
	hasTypedBloodRobe := false
	for _, mitigation := range target.DamageMitigations {
		if mitigation.BlockID == "BLOOD_STAINED_ROBE" {
			hasTypedBloodRobe = true
		}
	}
	for _, mitigation := range target.DamageMitigations {
		if damage <= 0 {
			break
		}
		mitigated := damage * mitigation.Ratio
		damage -= mitigated
		if mitigation.ConvertToShield {
			target.VirtualShield += mitigated
		}
		if mitigation.Message != "" {
			detail := strings.NewReplacer(
				"{actor}", target.Name,
				"{target}", actor.Name,
				"{mitigated}", fmt.Sprintf("%.0f", mitigated),
			).Replace(mitigation.Message)
			*logs = append(*logs, CombatLog{
				Turn: turn, ActorId: target.ID, ActorName: target.Name,
				Action: mitigation.LogAction, Detail: detail,
			})
		}
	}

	// Huyết Nhiễm Chinh Bào compatibility for saved native definitions.
	if !hasTypedBloodRobe && (hasStatus(target, statusfx.StatusBloodRobe) || actorHasMechanic(target, "BEFORE_HIT", "BLOOD_STAINED_ROBE") || actorHasMechanic(target, "COMBAT_PASSIVE", "BLOOD_STAINED_ROBE")) && damage > 0 {
		mitigated := damage * 0.40
		damage -= mitigated
		target.VirtualShield += mitigated
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "BLOOD_STAINED_ROBE",
			Detail:    fmt.Sprintf("[Huyết Đạo - Huyết Nhiễm Chinh Bào] Chiến bào đỏ thẫm hấp thụ 40%% sát thương (%.0f), chuyển hóa thành Huyết Thuẫn!", mitigated),
		})
	}

	// Typed incoming-hit counters execute at this same historical passive point,
	// after avoidance/mitigation and before immunity/HP damage resolution.
	counterDeltas := make(map[string]int, len(target.BeforeHitCounterAdds))
	for _, counter := range target.BeforeHitCounterAdds {
		previous := 0
		if target.CustomState != nil {
			previous = target.CustomState[counter.Key]
		}
		current := ApplyCounter(target, counter.Key, counter.Delta, counter.Min, counter.Max)
		counterDeltas[counter.Key] += current - previous
	}
	typedEatingStatsApplied := false
	for _, modifier := range target.BeforeHitCounterStatModifiers {
		delta := counterDeltas[modifier.Key]
		if delta <= 0 {
			continue
		}
		amount := float64(delta) * modifier.PerCounter
		switch modifier.Stat {
		case "ATTACK":
			target.BaseDmg += amount
		case "DEFENSE":
			target.BaseDef += amount
		}
		if modifier.Key == "eating_loss_stacks" {
			typedEatingStatsApplied = true
		}
	}
	typedEatingNotificationSent := false
	for _, notification := range target.BeforeHitCounterNotifications {
		if counterDeltas[notification.CounterKey] <= 0 {
			continue
		}
		detail := renderCounterNotification(target, notification)
		*logs = append(*logs, CombatLog{
			Turn: turn, ActorId: target.ID, ActorName: target.Name,
			Action: notification.Action, Detail: detail,
		})
		if notification.CounterKey == "eating_loss_stacks" {
			typedEatingNotificationSent = true
		}
	}
	typedEatingStackAdded := counterDeltas["eating_loss_stacks"] > 0

	// Cật Khuy Thị Phúc (EATING_LOSS_BLESSING): Chịu thiệt là phúc, mỗi khi bị đánh tích lũy 1 tầng phúc lợi (+5 Công, +5 Thủ)
	if hasStatus(target, statusfx.StatusEatingLoss) || actorHasMechanic(target, "BEFORE_HIT", "EATING_LOSS_BLESSING") || actorHasMechanic(target, "COMBAT_PASSIVE", "EATING_LOSS_BLESSING") {
		if target.CustomState["eating_loss_stacks"] < 10 || counterDeltas["eating_loss_stacks"] > 0 {
			stacks := target.CustomState["eating_loss_stacks"]
			if !typedEatingStackAdded {
				stacks = addBoundedCounter(target, "eating_loss_stacks", 1, 0, 10)
			}
			if !typedEatingStatsApplied {
				target.BaseDmg += 5
				target.BaseDef += 5
			}
			if !typedEatingNotificationSent {
				*logs = append(*logs, CombatLog{
					Turn:      turn,
					ActorId:   target.ID,
					ActorName: target.Name,
					Action:    "EATING_LOSS_BLESSING",
					Detail:    fmt.Sprintf("[Thực Đạo - Cật Khuy Thị Phúc] %s chịu đòn rèn giũa, tích lũy tầng thứ %d phúc lực (+5 Công, +5 Thủ)!", target.Name, stacks),
				})
			}
		}
	}

	// Đồ Đằng Kim Chung: Phản phệ 35% sát thương nhận vào
	if (hasStatus(target, statusfx.StatusTotemGoldenBell) || actorHasMechanic(target, "BEFORE_HIT", "TOTEM_GOLDEN_BELL") || actorHasMechanic(target, "COMBAT_PASSIVE", "TOTEM_GOLDEN_BELL")) && actor != target && (target.CustomState == nil || target.CustomState["reflecting_golden_bell"] != 1) {
		refl := damage * 0.35
		if target.CustomState == nil {
			target.CustomState = make(map[string]int)
		}
		target.CustomState["reflecting_golden_bell"] = 1
		applyDamageToTarget(state, target, actor, refl, logs, turn, true, false, false, rng)
		target.CustomState["reflecting_golden_bell"] = 0
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "TOTEM_GOLDEN_BELL",
			Detail:    fmt.Sprintf("[Đồ Đằng - Kim Chung] Tiếng chuông vàng ngân vang chấn động, phản phệ %.0f sát thương ngược lại %s!", refl, actor.Name),
		})
	}

	applyThresholdRecoveries(target, turn, logs)

	// Khí Tuyệt Phùng Sinh (QI_JUE_RETURN_LIFE): legacy compatibility hook.
	if (actorHasMechanic(target, "BEFORE_HIT", "QI_JUE_RETURN_LIFE") || actorHasMechanic(target, "COMBAT_PASSIVE", "QI_JUE_RETURN_LIFE")) && target.HP <= target.MaxHP*0.30 {
		if target.CustomState == nil {
			target.CustomState = make(map[string]int)
		}
		if target.CustomState["qi_jue_used"] == 0 {
			target.CustomState["qi_jue_used"] = 1
			heal := target.MaxHP * 0.40
			target.HP = math.Min(target.MaxHP, target.HP+heal)
			var clean []Effect
			for _, eff := range target.ActiveEffects {
				if eff.Type != "debuff" {
					clean = append(clean, eff)
				}
			}
			target.ActiveEffects = clean
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "QI_JUE_RETURN_LIFE",
				Detail:    fmt.Sprintf("[Khí Đạo - Khí Tuyệt Phùng Sinh] Mượn Hy Vọng Khí chuyển bại thành thắng! %s bộc phát sinh cơ hồi phục %.0f HP và tẩy sạch dị trạng!", target.Name, heal),
			})
		}
	}

	// Bất Hoại blocks every damage path. The DoT / %MaxHP / SOUL_WOUND / BACKLASH /
	// THORNS routes write HP directly and are guarded at their own sites, so
	// this gate only covers the ordinary-strike path. Sát Khí (level > 10) pierces.
	satKhiPierce := satKhiPiercesImmunity(actor)
	if isImmuneToDamage(target, satKhiPierce) {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "BLOCKED",
			Detail:    fmt.Sprintf("%s miễn nhiễm mọi sát thương nhờ Bất Hoại!", target.Name),
		})
		return 0
	}
	if satKhiPierce && (target.IsInvincible || target.DamageTakenReduction >= 1.0) {
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "BLOCKED_BREAK",
			Detail:    fmt.Sprintf("Sát khí Tất Sát xuyên qua Bất Hoại! %s vẫn nhận sát thương!", target.Name),
		})
	}

	// ── Freeze break: taking damage has 50% chance to break freeze ──
	if target.IsFrozen && damage > 0 {
		if rng.Float64() < 0.50 {
			var unfrozenKept []Effect
			for _, eff := range target.ActiveEffects {
				if eff.ID == EffectFreeze {
					*logs = append(*logs, CombatLog{
						Turn:      turn,
						ActorId:   target.ID,
						ActorName: target.Name,
						Action:    "FREEZE_BREAK",
						Detail:    fmt.Sprintf("Sát thương phá vỡ băng giá! %s thoát đóng băng!", target.Name),
					})
					continue
				}
				unfrozenKept = append(unfrozenKept, eff)
			}
			target.ActiveEffects = unfrozenKept
			target.IsFrozen = false
		}
	}

	// ── Sleep break: taking damage wakes the target ──
	if target.IsSleeping && damage > 0 {
		var wokeKept []Effect
		for _, eff := range target.ActiveEffects {
			if eff.ID == statusfx.StatusSleep {
				*logs = append(*logs, CombatLog{
					Turn:      turn,
					ActorId:   target.ID,
					ActorName: target.Name,
					Action:    "SLEEP_BREAK",
					Detail:    fmt.Sprintf("%s bị đánh thức!", target.Name),
				})
				continue
			}
			wokeKept = append(wokeKept, eff)
		}
		target.ActiveEffects = wokeKept
		target.IsSleeping = false
	}

	// ── Immortal Gu House (Tiên Cổ Ốc) damage absorption ──
	if target.GuHouse != nil && !target.GuHouse.IsDestroyed && target.GuHouse.Active {
		if target.GuHouse.DamageReduction > 0 {
			damage *= (1.0 - math.Min(0.85, target.GuHouse.DamageReduction))
		}
		if target.GuHouse.HP >= damage {
			target.GuHouse.HP -= damage
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "GU_HOUSE_ABSORB",
				Damage:    damage,
				Detail:    fmt.Sprintf("[Tiên Cổ Ốc] %s che chắn cho %s, hấp thụ %.0f sát thương (Ốc còn %.0f/%.0f HP)!", target.GuHouse.Name, target.Name, damage, target.GuHouse.HP, target.GuHouse.MaxHP),
			})
			return 0
		}
		absorbed := target.GuHouse.HP
		target.GuHouse.HP = 0
		target.GuHouse.IsDestroyed = true
		target.GuHouse.Active = false

		// Phản phệ Vỡ Phòng (Structural Breach): Không tràn sát thương sang nhục thân (No Spillover),
		// nhưng Cổ Tiên bị văng ra chịu Choáng 2 lượt và tổn thương 30% sinh mệnh tối đa do phản chấn trận pháp.
		recoil := math.Floor(target.MaxHP * 0.30)
		target.HP = math.Max(1, target.HP-recoil)
		target.IsStunned = true
		target.ActiveEffects = append(target.ActiveEffects, Effect{
			ID:       statusfx.StatusStun,
			Duration: 2,
		})

		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "GU_HOUSE_DESTROYED",
			Damage:    absorbed,
			Detail:    fmt.Sprintf("[Tiên Cổ Ốc Vỡ Vụn - Không Tràn Sát Thương] Kiến trúc phòng ngự %s của %s đã hấp thụ toàn bộ đòn đánh trước khi sụp đổ hoàn toàn! Phản phệ trận pháp khiến %s bị hất văng, chịu Choáng 2 lượt và tổn thương 30%% sinh mệnh (-%.0f HP)!", target.GuHouse.Name, target.Name, target.Name, recoil),
		})
		return 0
	}

	// Phá Không Cổ shield bypass
	if shieldBypass {
		target.HP -= damage
		applyLastStandFloor(target, satKhiPierce)
		if spillToAperture {
			redirectApertureOverflow(state, target, logs, turn)
		} else if target.HP <= 0 {
			if !applyLethalLuckWard(target, logs, turn, rng) && !applyNirvanaRebirth(state, target, logs, turn) {
				target.HP = 0
			}
		}
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SHIELD_BYPASS_ATTACK",
			TargetId:  target.ID,
			Damage:    damage,
			Detail:    fmt.Sprintf("%s sử dụng Phá Không Cổ xuyên thẳng qua Chân Nguyên Giáp, gây %d sát thương lên %s!", actor.Name, int(damage), target.Name),
		})
		return damage
	}

	// Thơ Cổ: "thi_ho" damage reduction (40%)
	for _, eff := range target.ActiveEffects {
		if eff.ID == "thi_ho" && eff.Duration >= 0 {
			reducedDmg := damage * 0.40
			damage -= reducedDmg
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "DEFEND",
				Detail:    fmt.Sprintf("[Thơ Cổ] Thi Hộ cản lại 40%% sát thương (giảm %.0f sát thương)!", reducedDmg),
			})
			break
		}
	}

	// Registry-driven damage reduction (DAMAGE_REDUCTION_UP / INVINCIBLE).
	if target.DamageTakenReduction > 0 {
		if target.DamageTakenReduction >= 1.0 {
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "BLOCKED",
				Detail:    fmt.Sprintf("%s miễn nhiễm sát thương (Bất Tử)!", target.Name),
			})
			return 0
		}
		reduced := damage * target.DamageTakenReduction
		damage -= reduced
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   target.ID,
			ActorName: target.Name,
			Action:    "DAMAGE_REDUCTION",
			Detail:    fmt.Sprintf("%s giảm %.0f%% sát thương nhận vào (cản %.0f sát thương)!", target.Name, target.DamageTakenReduction*100, reduced),
		})
	}

	hpDamageDealt := 0.0

	if target.VirtualShield > 0 {
		// Thơ Cổ: Ghi nhận khiên và xuyên khiên
		if hasMechanic(actor.Mechanics, "ON_TURN_START", "POEM_INTENT") {
			stacks := actor.CustomState["thi_y"]

			// Xuyên khiên (chỉ 1 lần mỗi lượt)
			if stacks >= 3 && !shieldBypass && actor.CustomState["thi_y_pierce_turn"] != turn {
				actor.CustomState["thi_y_pierce_turn"] = turn

				usableMultiples := stacks / 3
				ignorePct := float64(usableMultiples) * 0.30
				if ignorePct > 1.0 {
					ignorePct = 1.0
				}

				shieldIgnored := target.VirtualShield * ignorePct
				if shieldIgnored > damage {
					shieldIgnored = damage
				}

				if shieldIgnored > 0 {
					target.HP -= shieldIgnored
					applyLastStandFloor(target, satKhiPierce)
					if target.HP < 0 {
						target.HP = 0
					}
					damage -= shieldIgnored
					hpDamageDealt += shieldIgnored
					*logs = append(*logs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SHIELD_PIERCE",
						Detail:    fmt.Sprintf("[Thơ Cổ] Với uy năng từ %d tầng Thi Ý, đòn đánh xuyên thẳng qua %.0f%% khiên (gây %.0f sát thương thẳng vào HP)!", stacks, ignorePct*100, shieldIgnored),
					})
				}
			}

			// Thu thập thông tin khi chạm khiên (chỉ 1 lần mỗi lượt)
			if actor.CustomState["thi_y_shield_turn"] != turn {
				actor.CustomState["thi_y_shield_turn"] = turn
				actor.CustomState["thi_y"] += 1
				*logs = append(*logs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BUFF",
					Detail:    fmt.Sprintf("[Thơ Cổ] Chạm trán khiên, thu thập tin tức! Nhận thêm 1 tầng Thi Ý (hiện có %d tầng)!", actor.CustomState["thi_y"]),
				})

				if actor.CustomState["thi_y"] >= 10 {
					actor.CustomState["thi_y"] = 0
					target.VirtualShield = 0
					*logs = append(*logs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "SHIELD_BREAK",
						Detail:    fmt.Sprintf("[Thơ Cổ] Thi Ý viên mãn (10 tầng)! Nhìn thấu vạn vật, phá nát toàn bộ khiên ảo của %s!", target.Name),
					})
				}
			}
		}

		if damage <= 0 {
			return hpDamageDealt
		}

		if target.VirtualShield >= damage {
			target.VirtualShield -= damage
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "SHIELD_ABSORB",
				Damage:    damage,
				Detail:    fmt.Sprintf("%s tấn công %s gây %.0f sát thương (còn %.0f khiên, %.0f HP)", actor.Name, target.Name, damage, target.VirtualShield, target.HP),
			})
			return hpDamageDealt // Trả về hpDamageDealt để caller không log double damage
		}

		absorbed := target.VirtualShield
		damage -= absorbed
		target.VirtualShield = 0
		target.HP -= damage
		applyLastStandFloor(target, satKhiPierce)
		if spillToAperture {
			redirectApertureOverflow(state, target, logs, turn)
		} else if target.HP <= 0 {
			if !applyLethalLuckWard(target, logs, turn, rng) && !applyNirvanaRebirth(state, target, logs, turn) {
				target.HP = 0
			}
		}
		hpDamageDealt += damage
		totalDmg := absorbed + damage
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SHIELD_BREAK",
			Damage:    totalDmg,
			Detail:    fmt.Sprintf("%s tấn công %s gây %.0f sát thương (phá vỡ %.0f khiên, còn %.0f HP)", actor.Name, target.Name, totalDmg, absorbed, target.HP),
		})
		return hpDamageDealt // Chỉ trả về lượng sát thương trừ vào HP để caller log
	}

	// Direct HP damage (no shield)
	if spillToAperture {
		target.HP -= damage
		redirectApertureOverflow(state, target, logs, turn)
		return hpDamageDealt + damage
	}
	minHP := 0.0
	if target.IsLastStand && !satKhiPiercesImmunity(actor) {
		minHP = LastStandMinHP
	}
	resolved := ResolveBasicDamage(damage, 0, 0, target.HP, 0, minHP)
	target.HP = resolved.HP
	if target.HP <= 0 {
		if !applyLethalLuckWard(target, logs, turn, rng) && !applyNirvanaRebirth(state, target, logs, turn) {
			target.HP = 0
		}
	}
	return hpDamageDealt + resolved.HPDamage
}

// redirectApertureOverflow spills damage that reduced a defending ally below 0 HP
// into the Immortal Aperture (Tiên Khiếu). Called only for tribulation combat where
// ApertureMaxHP > 0 and the attacker is an enemy.
func redirectApertureOverflow(state *CombatState, target *CombatantAuto, logs *[]CombatLog, turn int) {
	if state == nil || state.ApertureMaxHP <= 0 {
		return
	}
	if target.HP >= 0 {
		return
	}
	overflow := -target.HP
	target.HP = 0
	state.ApertureHP -= overflow
	if state.ApertureHP < 0 {
		state.ApertureHP = 0
	}
	*logs = append(*logs, CombatLog{
		Turn:      turn,
		ActorId:   target.ID,
		ActorName: target.Name,
		Action:    "APERTURE_DAMAGE",
		Damage:    overflow,
		Detail:    fmt.Sprintf("Tiên Khiếu chịu %.0f sát thương xuyên phá (còn %.0f/%.0f)!", overflow, state.ApertureHP, state.ApertureMaxHP),
	})
}

// checkActionCondition returns true if the action's use condition is met.
func checkActionCondition(action *SimAction, actor, target *CombatantAuto) bool {
	if action != nil && action.IsFromGuHouse && (actor == nil || actor.GuHouse == nil || actor.GuHouse.IsDestroyed || !actor.GuHouse.Active) {
		return false
	}
	if action != nil && actor != nil && actor.GuHouse != nil && !actor.GuHouse.IsDestroyed && actor.GuHouse.Active {
		if !action.IsFromGuHouse && action.ActionType != "EXIT_GU_HOUSE" && action.ActionType != "FLEE" {
			return false
		}
	}
	if action != nil && (action.ActionType == "KILLER_MOVE" || action.IsFromGuHouse) && actor != nil && actor.GuHunger != nil && len(action.ComponentGuIDs) > 0 {
		cost := action.HungerCost
		if cost <= 0 {
			cost = 1
		}
		for _, guID := range action.ComponentGuIDs {
			if hunger, present := actor.GuHunger[guID]; present && hunger < cost {
				return false
			}
		}
	}
	if action != nil && actor != nil && actor.ActionCooldowns != nil {
		actionKey := action.KillerMoveID
		if actionKey == "" {
			actionKey = action.SourceID
		}
		if actionKey != "" && actor.ActionCooldowns[actionKey] > 0 {
			return false
		}
	}
	switch action.UseCondition {
	case "":
		return true
	case "hp_below":
		if actor.MaxHP <= 0 {
			return true
		}
		return actor.HP/actor.MaxHP*100 <= action.ConditionValue
	case "mana_above":
		if actor.MaxMana <= 0 {
			return true
		}
		return actor.Mana/actor.MaxMana*100 >= action.ConditionValue
	case "enemy_hp_below":
		if target == nil || target.MaxHP <= 0 {
			return true
		}
		return float64(target.HP)/float64(target.MaxHP)*100 <= action.ConditionValue
	default:
		return true
	}
}

// GetSimActionCategory returns the primary category for a SimAction: "shield", "heal", "damage", or "other".
func GetSimActionCategory(act SimAction) string {
	if act.Defense > 0 || act.ShieldMult > 1.0 || act.DefUpTier > 0 || act.ActionType == "SHIELD" {
		return "shield"
	}
	if act.Healing > 0 || act.RegenTier > 0 || act.LifeStealPct > 0 || act.ActionType == "HEAL" {
		return "heal"
	}
	if act.Damage > 0 || act.DamageMult > 1.0 || act.ActionType == "BASIC_ATTACK" {
		return "damage"
	}
	return "other"
}

func getCategoryPriorityIndex(cat string, priorities []string) int {
	if len(priorities) == 0 {
		priorities = []string{"other", "shield", "heal", "damage"}
	}
	for idx, p := range priorities {
		if p == cat {
			return idx
		}
	}
	return 999
}

func getPendingDamageMult(actor *CombatantAuto) float64 {
	if len(actor.PendingDamageBuffs) == 0 {
		return 1.0
	}
	// Cộng dồn bonus: 1 + Σ(m-1) thay vì nhân.
	mult := 1.0
	for _, m := range actor.PendingDamageBuffs {
		if m > 0 {
			mult += m - 1.0
		}
	}
	return mult
}

func getPendingShieldMult(actor *CombatantAuto) float64 {
	if len(actor.PendingShieldBuffs) == 0 {
		return 1.0
	}
	// Cộng dồn bonus: 1 + Σ(m-1) thay vì nhân.
	mult := 1.0
	for _, m := range actor.PendingShieldBuffs {
		if m > 0 {
			mult += m - 1.0
		}
	}
	return mult
}
