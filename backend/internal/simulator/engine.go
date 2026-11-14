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
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			return true
		}
	}
	return false
}

func hasStatus(c *CombatantAuto, statusID string) bool {
	if c == nil {
		return false
	}
	for _, eff := range c.ActiveEffects {
		if eff.ID == statusID && eff.Duration > 0 {
			return true
		}
	}
	return false
}

func mechanicFloat(mechanics []CombatMechanic, trigger, actionCode, key string, fallback float64) float64 {
	for _, mechanic := range mechanics {
		if mechanic.Trigger == trigger && mechanic.ActionCode == actionCode {
			if value, ok := mechanic.Values[key].(float64); ok {
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
		return 2, 2
	case "back":
		return 3, 2
	default:
		return 1, 2 // frontline
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
	rankFactor := gamefx.RankScale(rank)
	hpScale := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "hp_scale", 1)
	damageScale := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "damage_scale", 1)
	hp := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "hp", 0) * hpScale * rankFactor
	damage := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "damage", 0) * damageScale * rankFactor
	defense := mechanicFloat(ms, m.Trigger, "SUMMON_MONSTER", "defense", 0) * damageScale * rankFactor
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
	minion := &CombatantAuto{
		ID:      fmt.Sprintf("%s_summon_%d", actor.ID, summonCount),
		Name:    name + " (Triệu hồi)",
		Kind:    "Minion",
		Rank:    rank,
		GridX:   gridX,
		GridY:   gridY,
		MaxHP:   hp,
		HP:      hp,
		BaseDmg: damage,
		BaseDef: defense,
		Speed:   actor.Speed,
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
		GridY:   2, // Mid
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
	actor.GridX = 2 // Move actor to Mid
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
				}
			}
		}
	}
	apply(state.Allies, true)
	apply(state.Enemies, false)
	return logs
}

// applyBattleStartInstantWin resolves ON_BATTLE_START/INSTANT_WIN before the
// first turn: the carrier's side drops every opposing combatant outright,
// bypassing shields, damage immunity and Bất Tử. The normal end-of-turn win
// check then reports the victory.
func applyBattleStartInstantWin(state *CombatState) []CombatLog {
	var logs []CombatLog
	wipe := func(carriers, targets []*CombatantAuto) {
		for _, c := range carriers {
			if c == nil || c.HP <= 0 || !hasMechanic(c.Mechanics, "ON_BATTLE_START", "INSTANT_WIN") {
				continue
			}
			logs = append(logs, CombatLog{
				Turn:      0,
				ActorId:   c.ID,
				ActorName: c.Name,
				Action:    "INSTANT_WIN",
				Detail:    fmt.Sprintf("[%s] định đoạt chiến cuộc — toàn bộ địch nhân gục ngã!", c.Name),
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

// hasReverseCalamity reports whether a combatant carries Nghịch Chuyển Cổ's
// absolute reversal (COMBAT_PASSIVE/REVERSE_CALAMITY): every hostile effect
// aimed at the holder is undone, only raw damage passes through.
func hasReverseCalamity(c *CombatantAuto) bool {
	return c != nil && hasMechanic(c.Mechanics, "COMBAT_PASSIVE", "REVERSE_CALAMITY")
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

// pathBonus returns the maximum additive bonus for any path in paths from a
// per-path bonus map (e.g. character DaoBonus / PathDamageBonus / PathResistance).
func pathBonus(bonus map[string]float64, paths []string) float64 {
	if bonus == nil {
		return 0
	}
	if len(paths) == 0 {
		paths = []string{"strength"}
	}
	maxB := 0.0
	for _, p := range paths {
		clean := gamefx.CleanPath(p)
		if b := bonus[clean]; b > maxB {
			maxB = b
		}
	}
	return maxB
}

// ensureDaoProfile lazily builds the combatant's DaoCombatProfile.
func ensureDaoProfile(c *CombatantAuto) {
	if c == nil || c.DaoProfile != nil {
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
	dominantPath := c.DaoProfile.DominantPath
	dominantPurity := 0.0
	if dominantPath != "" {
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
	for _, col := range []int{1, 2, 3} {
		for _, targetY := range []int{1, 2, 3} {
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

func getTarget(enemies []*CombatantAuto) *CombatantAuto {
	var alive []*CombatantAuto
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
		var visible []*CombatantAuto
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
	case "ROW":
		var hits []*CombatantAuto
		for _, e := range alive {
			if e.GridX == targetX {
				hits = append(hits, e)
			}
		}
		if len(hits) > 0 {
			return hits
		}
		// fallback: first alive row
		if fallback := getFirstAliveDeterministic(alive); fallback != nil {
			return []*CombatantAuto{fallback}
		}
		return []*CombatantAuto{alive[0]}

	case "COL", "COLUMN":
		var hits []*CombatantAuto
		for _, e := range alive {
			if e.GridY == targetY {
				hits = append(hits, e)
			}
		}
		if len(hits) > 0 {
			return hits
		}
		if fallback := getFirstAliveDeterministic(alive); fallback != nil {
			return []*CombatantAuto{fallback}
		}
		return []*CombatantAuto{alive[0]}

	case "CROSS":
		var hits []*CombatantAuto
		for _, e := range alive {
			dx := e.GridX - targetX
			dy := e.GridY - targetY
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}
			if (e.GridX == targetX && e.GridY == targetY) || (dx == 1 && dy == 1) {
				hits = append(hits, e)
			}
		}
		if len(hits) > 0 {
			return hits
		}
		if fallback := getFirstAliveDeterministic(alive); fallback != nil {
			return []*CombatantAuto{fallback}
		}
		return []*CombatantAuto{alive[0]}

	case "ALL", "ALL_ENEMIES":
		return alive

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
			// 2. Match targetX, prioritize GridY 1 -> 3
			if best == nil {
				for _, yVal := range []int{1, 2, 3} {
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
		// 3. Fallback to columns 1, 2, 3, prioritizing GridY 1 -> 3 (standard non-bypass targeting)
		if best == nil {
			for _, col := range []int{1, 2, 3} {
				for _, yVal := range []int{1, 2, 3} {
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
			return []*CombatantAuto{best}
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
			// ── Tick DoTs for all combatants ──
			for _, combatants := range [][]*CombatantAuto{state.Allies, state.Enemies} {
				for _, c := range combatants {
					if c.HP <= 0 {
						continue
					}
					turnLogs = append(turnLogs, tickDoTs(c, turn)...)
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

					// Catalog-driven Poem Intent mechanic - start-of-turn logic.
					if hasMechanic(c.Mechanics, "ON_TURN_START", "POEM_INTENT") {
						c.CustomState["thi_y"] += 1
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

						if c.CustomState["thi_y"] >= 10 {
							// Immediately destroy all enemies' shields if reaching 10 at start of turn
							c.CustomState["thi_y"] = 0
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
						markStacks := 0
						for _, eff := range c.ActiveEffects {
							if eff.ID == statusfx.StatusSwordMark {
								markStacks++
							}
						}
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

		var runOne func(actor *CombatantAuto, a *SimAction, isAlly bool, enemyTargets []*CombatantAuto) actionOutcome
		runOne = func(actor *CombatantAuto, a *SimAction, isAlly bool, enemyTargets []*CombatantAuto) actionOutcome {
			res := actionOutcome{}
			operationDamageScale := 1.0
			guCostIDs := []string{}

			if (actor.IsSilenced || actor.IsFeared) && a.ActionType != "BASIC_ATTACK" {
				return res
			}

			if actor.HasGuSeal && (a.ActionType == "GU_SKILL" || a.ActionType == "KILLER_MOVE") {
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "GU_SEALED",
					Detail:    fmt.Sprintf("%s bị phong ấn Cổ, không thể dùng [%s] lượt này!", actor.Name, a.Name),
				})
				return res
			}

			if actor.HasThoughtChaos && (a.ActionType == "GU_SKILL" || a.ActionType == "KILLER_MOVE") {
				a.ManaCost *= 2.0
				if rng.Float64() < 0.30 {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "THOUGHT_CHAOS_FAIL",
						Detail:    fmt.Sprintf("[Trí Đạo - Tạp Niệm Loạn Ý] Đầu óc %s tràn ngập tạp niệm hỗn loạn! Niệm đầu bị phân tán, thi triển [%s] thất bại hoàn toàn!", actor.Name, a.Name),
					})
					return res
				} else {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "THOUGHT_CHAOS",
						Detail:    fmt.Sprintf("[Trí Đạo - Tạp Niệm Loạn Ý] %s gượng ép thôi động ý niệm, tiêu hao gấp đôi Chân Nguyên (%.0f Chân Nguyên)!", actor.Name, a.ManaCost),
					})
				}
			}

			if a.ActionType == "GU_SKILL" && (actor.Kind == "Player" || actor.Kind == "Ally" || actor.Kind == "VariantNPC") {
				guKey := a.InstanceID
				if guKey == "" {
					guKey = a.SourceID
				}
				if actor.GuHunger != nil {
					h, exists := actor.GuHunger[guKey]
					if !exists && a.SourceID != "" && guKey != a.SourceID {
						h, exists = actor.GuHunger[a.SourceID]
						if exists {
							guKey = a.SourceID
						}
					}
					if exists && h <= 0 {
						return res
					}
					if exists {
						guCostIDs = append(guCostIDs, guKey)
					}
				}
				if guKey != "" {
					guCostIDs = append(guCostIDs, "used:"+guKey)
				}
			}

			if a.ActionType == "KILLER_MOVE" && actor.Kind == "Player" && len(a.ComponentGuIDs) > 0 {
				canCast := true
				for _, guID := range a.ComponentGuIDs {
					h, exists := actor.GuHunger[guID]
					if !exists || h <= 0 {
						canCast = false
						break
					}
				}
				if !canCast {
					turnLogs = append(turnLogs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "KILLER_MOVE_UNAVAILABLE", Detail: fmt.Sprintf("%s không thể dùng [%s] vì thiếu Cổ thành phần hoặc Cổ đã kiệt sức!", actor.Name, a.Name)})
					return res
				}
				if actor.GuUsedCounts == nil {
					actor.GuUsedCounts = make(map[string]int)
				}
				for _, guID := range a.ComponentGuIDs {
					guCostIDs = append(guCostIDs, guID, "used:"+guID)
				}
			}

			guRank := 1
			if a.Rank > 0 {
				guRank = a.Rank
			}
			conflictMult := actionConflictMult(actor, a.Path)
			effectiveManaCost := a.ManaCost
			if effectiveManaCost > 0 {
				effectiveManaCost *= gamefx.ManaTaxMultiplier(conflictMult)
			}
			if (actor.Kind == "Player" || actor.Kind == "VariantNPC") && effectiveManaCost > 0 && actor.Mana < effectiveManaCost {
				turnLogs = append(turnLogs, CombatLog{Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "INSUFFICIENT_MANA", Detail: fmt.Sprintf("%s không đủ Chân Nguyên (cần %.0f, còn %.0f) để kích hoạt [%s]!", actor.Name, effectiveManaCost, actor.Mana, a.Name)})
				return res
			}
			if combatantPaysMana(actor) && effectiveManaCost > 0 {
				actor.Mana -= effectiveManaCost
			}
			if len(guCostIDs) > 0 {
				if actor.GuUsedCounts == nil {
					actor.GuUsedCounts = make(map[string]int)
				}
				for _, id := range guCostIDs {
					if strings.HasPrefix(id, "used:") {
						actor.GuUsedCounts[strings.TrimPrefix(id, "used:")]++
					} else {
						actor.GuHunger[id]--
					}
				}
			}
			backlashDamage := 0.0
			if a.BacklashDamage > 0 {
				backlashDamage = a.BacklashDamage
			} else if a.ActionType == "GU_SKILL" {
				backlashDamage = gamefx.HigherRankHPCost(guRank, actor.Rank, actor.MaxHP)
			}

			if backlashDamage > 0 {
				actor.HP -= backlashDamage
				if actor.HP < 0 {
					actor.HP = 0
				}
				guName := a.Name
				if guName == "" {
					guName = "Cổ trùng"
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "GU_BACKLASH",
					Damage:    backlashDamage,
					Detail:    fmt.Sprintf("%s chịu %d sát thương phản phệ do dùng %s (Rank %d > Rank %d)!", actor.Name, int(backlashDamage), guName, guRank, actor.Rank),
				})
				if actor.HP == 0 {
					actor.IsDead = true
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "DIED",
						Detail:    fmt.Sprintf("%s đã gục ngã do phản phệ Cổ!", actor.Name),
					})
					res.abort = true
					return res
				}
			}

			actionStartIndex := len(turnLogs)

			// ── Dao Conflict punishment (spec §3) ──
			conflictPenaltyMult := 1.0 // 0.5 when a non-fizzled backfire weakens the strike

			// Cast Failure (fizzle): conflicting dao marks shred the technique.
			if conflictMult < 0.70 && (a.ActionType == "GU_SKILL" || a.ActionType == "KILLER_MOVE") {
				if chance := gamefx.CastFailureChance(conflictMult); chance > 0 && rng.Float64() < chance {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "DAO_CAST_FAILED",
						Detail:    fmt.Sprintf("[Thôi Động Thất Bại] Đạo ngân dị chủng nhiễu loạn chân nguyên, sát chiêu [%s] tán loạn tiêu biến!", a.Name),
					})
					return res
				}
			}

			// Backfire: violent dao conflict wounds the caster directly (true damage).
			// Basic attacks (đánh thường) are exempt: the fallback path defaults to
			// Strength, so mixed-dao casters would otherwise self-wound every turn.
			if conflictMult < 0.60 && a.ActionType != "BASIC_ATTACK" && !a.IsKillerMoveOperation && !a.IsEffectOperation {
				if chance := gamefx.BackfireChance(conflictMult); chance > 0 && rng.Float64() < chance {
					satKhi := satKhiPiercesImmunity(actor)
					if !isImmuneToDamage(actor, satKhi) {
						bfDmg := actor.MaxHP * (0.08 + rng.Float64()*0.04)
						actor.HP -= bfDmg
						applyLastStandFloor(actor, satKhi)
						if actor.HP < 0 {
							actor.HP = 0
						}
						conflictPenaltyMult = 0.5
						backfireReason := daoBackfireExplanation(actor, a, conflictMult, chance, bfDmg)
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "DAO_BACKFIRE",
							Damage:    bfDmg,
							Detail:    fmt.Sprintf("[Phản Phệ Nội Thương] %s chịu %d sát thương nội thương. %s", actor.Name, int(bfDmg), backfireReason),
						})
						if actor.HP <= 0 {
							actor.IsDead = true
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "DIED",
								Detail:    fmt.Sprintf("%s đã bạo thể vong mạng do phản phệ Đạo Ngân. Lý do: %s HP còn %.0f/%.0f.", actor.Name, backfireReason, math.Max(0, actor.HP), actor.MaxHP),
							})
							res.abort = true
							return res
						}
					}
				}
			}

			if a.ActionType == "KILLER_MOVE" {
				moveLabel := "sát chiêu"
				if a.IsFromGuHouse {
					moveLabel = "Trận Kích"
				}
				chance := a.SuccessChance
				if chance <= 0 || chance > 1 {
					// Moves created by players always carry a server-computed
					// chance. Preserve legacy/synthetic actions as guaranteed casts.
					chance = 1
				}
				roll := rng.Float64()
				if roll >= chance {
					backfireDmg := a.BackfireDamage
					if backfireDmg <= 0 {
						backfireDmg = math.Round(actor.MaxHP * 0.12)
					}
					if actor.GuHouse != nil && !actor.GuHouse.IsDestroyed {
						turnLogs = append(turnLogs, CombatLog{
							Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "KILLER_MOVE_FAILED",
							Detail: fmt.Sprintf("%s thôi động %s [%s] thất bại (%.0f%% / tỉ lệ %.0f%%); Tiên Cổ Ốc [%s] đã che chắn áp lực phản phệ cho chủ nhân!", actor.Name, moveLabel, a.Name, roll*100, chance*100, actor.GuHouse.Name),
						})
					} else {
						actor.HP -= backfireDmg
						if actor.HP < 0 {
							actor.HP = 0
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "KILLER_MOVE_FAILED", Damage: backfireDmg,
							Detail: fmt.Sprintf("[Phản Phệ] %s thôi động %s [%s] thất bại (%.0f%% / tỉ lệ %.0f%%), chân nguyên bạo tẩu gây %.0f sát thương phản phệ!", actor.Name, moveLabel, a.Name, roll*100, chance*100, backfireDmg),
						})
						if a.BackfireStatus != "" {
							actor.HasThoughtChaos = true
							turnLogs = append(turnLogs, CombatLog{
								Turn: turn, ActorId: actor.ID, ActorName: actor.Name, Action: "BACKFIRE_STATUS",
								Detail: fmt.Sprintf("%s rơi vào trạng thái [%s] do chấn thương niệm đầu!", actor.Name, a.BackfireStatus),
							})
						}
					}
					actionKey := a.KillerMoveID
					if actionKey == "" {
						actionKey = a.SourceID
					}
					if actionKey != "" {
						if actor.ActionCooldowns == nil {
							actor.ActionCooldowns = make(map[string]int)
						}
						failCd := a.CooldownTurns / 2
						if failCd < 1 {
							failCd = 1
						}
						actor.ActionCooldowns[actionKey] = failCd
					}
					return res
				}
				if actor.KillerMoveUsedCounts == nil {
					actor.KillerMoveUsedCounts = make(map[string]int)
				}
				if a.KillerMoveID != "" {
					actor.KillerMoveUsedCounts[a.KillerMoveID]++
				}
				actionKey := a.KillerMoveID
				if actionKey == "" {
					actionKey = a.SourceID
				}
				if actionKey != "" && a.CooldownTurns > 0 {
					if actor.ActionCooldowns == nil {
						actor.ActionCooldowns = make(map[string]int)
					}
					actor.ActionCooldowns[actionKey] = a.CooldownTurns
				}
			}

			// ── BEFORE_ENEMY_ATTACK: FORGET_INTENT interrupt ──
			// The attack itself still lands; the attacker's Gu is sealed for
			// the following turns. Independent roll per defending owner.
			if hasEligibleHostileDamage(actor, a, enemyTargets, turn) && guActionHasAvailableHunger(actor, a) && !a.IsEffectOperation {
				for _, d := range enemyTargets {
					if d == nil || d.HP <= 0 || d.IsDead {
						continue
					}
					if !hasMechanic(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT") {
						continue
					}
					// Nghịch Chuyển Cổ: hostile Gu effects aimed at the holder
					// are reversed — the interrupt fizzles, the strike lands.
					if hasReverseCalamity(actor) {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "REVERSE_NEGATE",
							Detail:    fmt.Sprintf("Nghịch Chuyển Cổ đảo nghịch [Lãng Quên Ý Niệm] của %s về hư vô!", d.Name),
						})
						continue
					}
					chance := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "chance", 0)
					intentRoll := rng.Float64()
					if intentRoll >= chance {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   d.ID,
							ActorName: d.Name,
							Action:    "FORGET_INTENT_MISS",
							Detail:    fmt.Sprintf("[Lãng Quên Ý Niệm] của %s không kích hoạt: roll %.0f%% >= tỉ lệ %.0f%%!", d.Name, intentRoll*100, chance*100),
						})
						continue
					}
					if d.CustomState == nil {
						d.CustomState = make(map[string]int)
					}
					maxSucc := mechanicInt(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "max_successful_activations", 8)
					if d.CustomState["forget_intent_success"] >= maxSucc {
						if mechanicBool(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "backfire_after_max", true) {
							sealGuTurns(d, 1)
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   d.ID,
								ActorName: d.Name,
								Action:    "FORGET_INTENT_BACKFIRE",
								Detail:    fmt.Sprintf("%s đã lãng quên quá nhiều (%d lần), bị phản phệ phong ấn Cổ!", d.Name, maxSucc),
							})
						}
						continue
					}
					d.CustomState["forget_intent_success"]++
					disableTurns := mechanicInt(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "disable_target_gu_turns", 1)
					sealGuTurns(actor, disableTurns)
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "FORGET_INTENT",
						Detail:    fmt.Sprintf("%s quên mất ý niệm sau đòn này, [%s] bị phong ấn %d lượt! (%d/%d)", actor.Name, a.Name, disableTurns, d.CustomState["forget_intent_success"], maxSucc),
					})
				}
			}

			if (a.ActionType == "GU_SKILL" || a.ActionType == "BASIC_ATTACK" || a.ActionType == "KILLER_MOVE") && (actor.Kind == "Player" || actor.Kind == "Ally") && !state.VirtualMode {
				cleanPath := "strength"
				guRank := 1
				baseGain := 1
				if a.ActionType == "GU_SKILL" {
					guRank = a.Rank
					if guRank < 1 {
						guRank = 1
					}
					if len(a.Path) > 0 {
						cleanPath = gamefx.CleanPath(a.Path[0])
					}
					baseGain = gamefx.PathMasteryGuUseBaseGain(guRank)
				} else if a.ActionType == "KILLER_MOVE" {
					guRank = a.Rank
					if guRank < 1 {
						guRank = 1
					}
					if len(a.Path) > 0 {
						cleanPath = gamefx.CleanPath(a.Path[0])
					}
					baseGain = 2
					if guRank >= 6 {
						baseGain = 10
					}
				}

				isImmortal := guRank >= 6
				remaining := actor.PathMasteryGainLimit - actor.PathMasteryGained
				if isImmortal || remaining > 0 {
					if actor.PathMasteries == nil {
						actor.PathMasteries = make(map[string]int)
					}
					pts := actor.PathMasteries[cleanPath]
					currentLevel := gamefx.PathMasteryLevel(pts)
					pathGain := gamefx.CalculatePathMasteryGain(guRank, currentLevel, baseGain)

					if pathGain > 0 {
						clamped := false
						if !isImmortal {
							if pathGain > remaining {
								pathGain = remaining
								clamped = true
							}
							actor.PathMasteryGained += pathGain
							if actor.PathMasteryGained >= actor.PathMasteryGainLimit {
								clamped = true
							}
						}
						actor.PathMasteries[cleanPath] += pathGain

						newPts := actor.PathMasteries[cleanPath]
						newLvl := gamefx.PathMasteryLevel(newPts)
						lvlName := gamefx.PathMasteryLevelName(newLvl)

						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "PATH_MASTERY_GAIN",
							Detail:    fmt.Sprintf("-> Ngộ Đạo tăng tiến (%s)! +%d điểm Chân Ý %s (Cảnh giới: %s).", a.Name, pathGain, strings.Title(cleanPath), lvlName),
						})
						if clamped {
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "PATH_MASTERY_CAP",
								Detail:    fmt.Sprintf("-> %s đã đạt giới hạn %d điểm Chân Ý trong trận này, không thể ngộ đạo thêm.", actor.Name, actor.PathMasteryGainLimit),
							})
						}
					}
				}
			}

			// A composed move pays and schedules once, then delegates its authored
			// operations to the existing per-effect simulator adapter. The parent
			// has already passed cast failure, resource, backlash and cooldown rules.
			if a.ActionType == "KILLER_MOVE" && len(a.OperationEffects) > 0 {
				res.executed = true
				for _, effect := range coalesceKillerMoveDamageEffects(a) {
					if effect.Action == nil {
						continue
					}
					child := *effect.Action
					child.OperationConditions = append([]gameplayeffects.Condition(nil), effect.Compiled.Conditions...)
					child.IsKillerMoveOperation = true
					child.IsEffectOperation = true
					child.Rank = effect.SourceRank
					child.SourceID = a.SourceID
					out := runOne(actor, &child, isAlly, enemyTargets)
					res.executed = res.executed || out.executed
					res.hasDamage = res.hasDamage || out.hasDamage
					if out.abort {
						res.abort = true
						return res
					}
				}
				return res
			}

			resolved := resolveTargets(enemyTargets, a)
			if (a.IsKillerMoveOperation || a.IsEffectOperation) && strings.EqualFold(a.TargetType, "SELF") {
				resolved = []*CombatantAuto{actor}
			}
			if len(a.OperationConditions) > 0 {
				resolved = filterOperationTargets(actor, resolved, a.OperationConditions, a.Rank, a.Path, turn)
			}
			// ── BEFORE_ENEMY_ATTACK: FORGET_WILL confuse ──
			// Redirects (or fizzles) the just-resolved targets. First
			// defending owner that procs decides; backfire seals the owner.
			if len(resolved) > 0 && len(a.OperationEffects) == 0 && hasEligibleHostileDamage(actor, a, enemyTargets, turn) {
				for _, d := range enemyTargets {
					if d == nil || d.HP <= 0 || d.IsDead {
						continue
					}
					if !hasMechanic(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE") {
						continue
					}
					// Nghịch Chuyển Cổ: hostile Gu effects aimed at the holder
					// are reversed — the confusion fizzles, the strike lands.
					if hasReverseCalamity(actor) || isGuaranteedHit(actor, a) {
						reason := "Nghịch Chuyển Cổ đảo nghịch"
						if isGuaranteedHit(actor, a) {
							reason = "Đòn đánh Tất Trúng phớt lờ"
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "REVERSE_NEGATE",
							Detail:    fmt.Sprintf("%s [Lãng Quên Chí] của %s về hư vô!", reason, d.Name),
						})
						continue
					}
					willChance := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "activation_chance", 0.35)
					willRoll := rng.Float64()
					if willRoll >= willChance {
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   d.ID,
							ActorName: d.Name,
							Action:    "FORGET_WILL_INACTIVE",
							Detail:    fmt.Sprintf("[Lãng Quên Chí] của %s không kích hoạt: roll %.0f%% >= tỉ lệ %.0f%%!", d.Name, willRoll*100, willChance*100),
						})
						continue
					}
					if bf := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "backfire_chance", 0.2); rng.Float64() < bf {
						sealGuTurns(d, mechanicInt(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "backfire_disable_turns", 2))
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   d.ID,
							ActorName: d.Name,
							Action:    "FORGET_WILL_BACKFIRE",
							Detail:    fmt.Sprintf("%s bị phản phệ khi bóp méo ý chí, Cổ bị phong ấn!", d.Name),
						})
					}
					r := rng.Float64()
					hu := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "hit_user_chance", 0.15)
					hs := mechanicFloat(d.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "hit_self_chance", 0.35)
					switch {
					case r < hu:
						resolved = []*CombatantAuto{d}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "FORGET_WILL_REDIRECT",
							Detail:    fmt.Sprintf("%s quên mất mục tiêu, đòn [%s] lạc sang %s!", actor.Name, a.Name, d.Name),
						})
					case r < hu+hs:
						resolved = []*CombatantAuto{actor}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "FORGET_WILL_SELFHIT",
							Detail:    fmt.Sprintf("%s quên mất mục tiêu, tự đánh bản thân bằng [%s]!", actor.Name, a.Name),
						})
					default:
						resolved = nil
						res.executed = true
						res.hasDamage = false
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "FORGET_WILL_MISS",
							Detail:    fmt.Sprintf("%s quên mất mục tiêu, đòn [%s] đánh hụt hoàn toàn!", actor.Name, a.Name),
						})
						return res
					}
					break
				}
			}
			if len(resolved) == 0 {
				return res
			}

			res.executed = true
			res.hasDamage = a.ActionType == "BASIC_ATTACK" || a.Damage > 0 || a.DamageHPPercent > 0 || a.ShieldStripPercent > 0 || a.PhantomDmg > 0

			firstTarget := resolved[0]

			// Dao mark multiplier for this action's path (applies to damage, shield, heal)
			daoMult, hasDao := actorDaoMultiplier(actor, a.Path)

			// Buffer shield/heal/mana/cleanse logs — emit after action name
			var shieldLog, healLog, manaLog, cleanseLog *CombatLog

			// Apply shield to self
			if a.Defense > 0 {
				baseShield := a.Defense
				skillMult := a.ShieldMult
				if skillMult <= 0 {
					skillMult = 1.0
				}
				pendingShield := getPendingShieldMult(actor)
				combinedMult := skillMult
				if pendingShield > 1.0 {
					// Cộng dồn: skillMult + pendingShield - 1 (thay vì nhân).
					combinedMult = skillMult + pendingShield - 1.0
					actor.PendingShieldBuffs = nil // consume all pending shield buffs
				} else {
					pendingShield = 1.0
				}
				baseShield = a.Defense * combinedMult
				daoShield := 0.0
				if hasDao && daoMult > 1.0 {
					daoShield = baseShield * (daoMult - 1.0)
				}
				totalShield := baseShield + daoShield
				actor.VirtualShield += totalShield
				shieldReason := a.Reason
				conflictMult := actionConflictMult(actor, a.Path)
				if conflictMult < 1.0 {
					conflictMsg := fmt.Sprintf("Xung đột Đạo Ngân (uy lực x%.2f)", conflictMult)
					if shieldReason != "" {
						shieldReason += " | " + conflictMsg
					} else {
						shieldReason = conflictMsg
					}
				}
				sl := CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SHIELD_GAIN",
					Damage:    totalShield,
					Detail:    fmt.Sprintf("%s nhận %.0f Khiên Ảo!", actor.Name, totalShield),
					Breakdown: &DamageBreakdown{
						ShieldBase:      a.Defense,
						ShieldSkillMult: skillMult,
						ShieldBuffMult:  pendingShield,
						ShieldDaoMult:   daoMult,
						ShieldDaoBonus:  daoShield,
						ShieldFinal:     totalShield,
						Reason:          shieldReason,
						ShielderDefense: actor.StatProvenance["defense"],
					},
				}
				if daoShield > 0 {
					sl.Detail = fmt.Sprintf("%s nhận %.0f + %.0f Khiên Ảo do Đạo Ngân!", actor.Name, baseShield, daoShield)
				}
				shieldLog = &sl
			}

			// Apply healing to self
			if a.Healing > 0 {
				baseHeal := a.Healing
				daoHeal := 0.0
				if hasDao && daoMult > 1.0 {
					daoHeal = baseHeal * (daoMult - 1.0)
				}
				heal := (baseHeal + daoHeal) * effectiveHealingReceived(actor)
				actor.HP += heal
				if actor.HP > actor.MaxHP {
					actor.HP = actor.MaxHP
				}
				hl := CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "HEAL",
					Damage:    heal,
					Detail:    fmt.Sprintf("%s hồi phục %.0f HP!", actor.Name, heal),
					Breakdown: &DamageBreakdown{
						HealingBase:     baseHeal,
						HealingDaoBonus: daoHeal,
						HealingFinal:    heal,
						HealerHealing:   actor.StatProvenance["healing_received"],
					},
				}
				if daoHeal > 0 {
					hl.Detail = fmt.Sprintf("%s hồi phục %.0f HP (gốc %.0f + Đạo Ngân %.0f)!", actor.Name, heal, baseHeal, daoHeal)
				}
				healLog = &hl
			}

			// Apply mana restore to self
			if a.ManaRestore > 0 {
				needed := actor.MaxMana - actor.Mana
				restore := math.Min(needed, a.ManaRestore)
				if restore > 0 {
					actor.Mana += restore
					ml := CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "MANA_RESTORE",
						Damage:    restore,
						Detail:    fmt.Sprintf("%s khôi phục %.0f %s!", actor.Name, restore, actor.GetManaName()),
					}
					manaLog = &ml
				}
			}

			// Apply cleanse to self
			if a.CleanseDebuffs {
				var keptEffects []Effect
				cleansedCount := 0
				for _, eff := range actor.ActiveEffects {
					if eff.Type == "debuff" {
						cleansedCount++
					} else {
						keptEffects = append(keptEffects, eff)
					}
				}
				actor.ActiveEffects = keptEffects
				if len(actor.ActiveDoT) > 0 {
					cleansedCount += len(actor.ActiveDoT)
					actor.ActiveDoT = nil
				}
				if cleansedCount > 0 {
					ComputeEffectMultipliers(actor)
					cl := CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "CLEANSE",
						Detail:    fmt.Sprintf("%s tịnh hóa bản thân, xóa bỏ %d hiệu ứng bất lợi!", actor.Name, cleansedCount),
					}
					cleanseLog = &cl
				}
			}

			// ── Special Actions preprocessing ──
			if hasMechanic(a.Mechanics, "ON_CAST", "SUMMON_WOLF_FRONTLINE") {
				// Summoned beasts scale with the summoning Gu's dao marks, exactly
				// like the damage/shield/heal of the action itself (daoMult above).
				detail := spawnWolfFrontline(actor, isAlly, state, daoMult)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SUMMONED",
					Detail:    detail,
				})
			}
			// SUMMON_MONSTER: data-driven summon. The combat builder injects the
			// monster template stats (hp/damage/defense/name) resolved from
			// MonsterRegistry; the engine scales them by the rank of the summoning
			// Gu. Clamps keep a summon from out-scaling its caster.
			// ponytail: fixed clamps (2x caster HP, 3x caster damage) - tune these
			// multipliers if summons should hit harder or tankier.
			if m, ok := findMechanic(a.Mechanics, "ON_CAST", "SUMMON_MONSTER"); ok && !hasMinion(state, actor.ID) {
				_, detail := spawnSummonedMonster(actor, m, a.Rank, isAlly, state, summonCount, daoMult)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SUMMONED",
					Detail:    detail,
				})
				summonCount++
			}

			if hasMechanic(a.Mechanics, "ON_CAST", "FULL_SOUL_HEAL") {
				actor.HP = actor.MaxHP
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "HEAL",
					Detail:    fmt.Sprintf("%s sử dụng Cổ Gan Dạ, hồi phục hoàn toàn HP linh hồn nhưng mất lượt hành động!", actor.Name),
				})
				res.abort = true // Break action loop (Turn_Cost = 1)
				return res
			}

			if hasMechanic(a.Mechanics, "ON_CAST", "SWORD_DRAGON_TRANSFORM") {
				shieldGain := actor.HP - 1.0
				if shieldGain > 0 {
					actor.HP = 1.0
					actor.VirtualShield += shieldGain
					actor.ActiveEffects = append(actor.ActiveEffects, Effect{
						ID:       "sword_dragon_form",
						Type:     "buff",
						Value:    50,
						Duration: 3,
					})
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "TRANSFORM",
						Detail:    fmt.Sprintf("%s hóa thân thành Thượng Cổ Kiếm Long! Chuyển %.0f HP thành Khiên Ảo, tăng 50%% sát thương trong 3 lượt!", actor.Name, shieldGain),
					})
				}
			}

			if hasMechanic(a.Mechanics, "ON_CAST", "BORROW_PAST_POWER") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID:       "borrow_past_power",
					Type:     "buff",
					Value:    40,
					Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BUFF",
					Detail:    fmt.Sprintf("%s mượn sức mạnh đỉnh phong từ quá khứ (+40%% công kích trong 3 lượt)!", actor.Name),
				})
			}

			if hasMechanic(a.Mechanics, "ON_CAST", "UNRAVEL_RIDDLE") {
				dispelledCount := 0
				var keptEffects []Effect
				for _, eff := range firstTarget.ActiveEffects {
					if eff.Type == "buff" && dispelledCount < 2 {
						dispelledCount++
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "DISPEL",
							TargetId:  firstTarget.ID,
							Detail:    fmt.Sprintf("%s hóa giải hiệu ứng lợi ích [%s] của %s!", actor.Name, eff.ID, firstTarget.Name),
						})
					} else {
						keptEffects = append(keptEffects, eff)
					}
				}
				firstTarget.ActiveEffects = keptEffects

				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "INTERRUPT",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("%s sử dụng Giải Mê Cổ ngắt đòn tụ lực sát chiêu của %s!", actor.Name, firstTarget.Name),
				})
			}

			if hasMechanic(a.Mechanics, "ON_CAST", "CHAOS_GAMBLE") {
				effectsChoices := []string{"crit_500", "invulnerable_1_turn", "reduce_enemy_hp_90"}
				costChoices := []string{"aptitude_10", "lifespan_5", "luck_50"}

				chosenEffect := effectsChoices[rng.Intn(len(effectsChoices))]
				chosenCost := costChoices[rng.Intn(len(costChoices))]

				effectMsg := ""
				if chosenEffect == "crit_500" {
					effectMsg = "Tấn công chí mạng tăng 500%!"
				} else if chosenEffect == "invulnerable_1_turn" {
					effectMsg = "Miễn nhiễm mọi sát thương trong 1 lượt (Bất Hoại)!"
					actor.ActiveEffects = append(actor.ActiveEffects, Effect{
						ID:       "invulnerable",
						Type:     "buff",
						Value:    1,
						Duration: 1,
					})
				} else if chosenEffect == "reduce_enemy_hp_90" {
					effectMsg = "Giảm trực tiếp 90% lượng HP hiện tại của kẻ địch!"
					damageToEnemy := firstTarget.HP * 0.90
					firstTarget.HP -= damageToEnemy
					if applyLastStandFloor(firstTarget, false) {
						effectMsg = "Giảm trực tiếp 90% lượng HP hiện tại của kẻ địch — nhưng [Bất Tử] giữ lại 1 HP!"
					} else if firstTarget.HP < 1.0 {
						firstTarget.HP = 1.0
					}
				}

				costMsg := ""
				if chosenCost == "aptitude_10" {
					costMsg = "Trừ 10% Tư chất gốc vĩnh viễn!"
				} else if chosenCost == "lifespan_5" {
					costMsg = "Trừ 5 năm tuổi thọ vĩnh viễn!"
				} else if chosenCost == "luck_50" {
					costMsg = "Trừ 50% Nguyên Thạch hiện tại!"
				}

				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "CHAOS_GAMBLE",
					Detail:    fmt.Sprintf("[%s] kích hoạt Thần Thông Cổ Nhiệm Ý! %s Cái giá phải trả: %s", a.Name, effectMsg, costMsg),
				})

				if chosenEffect == "crit_500" {
					a.Damage *= 5.0
					operationDamageScale = 5.0
				}
			}

			// Vận Đạo: Cát Hung Chi Biến (LUCK_ROULETTE)
			if hasMechanic(a.Mechanics, "ON_CAST", "LUCK_ROULETTE") || hasMechanic(a.Mechanics, "ON_ACTION", "LUCK_ROULETTE") {
				hasCurseReverse := false
				curseIdx := -1
				for idx, eff := range actor.ActiveEffects {
					if eff.ID == statusfx.StatusLuckCurseReverse {
						hasCurseReverse = true
						curseIdx = idx
						break
					}
				}

				greatLuckChance := 15.0
				if hasCurseReverse {
					greatLuckChance = 65.0
					if curseIdx >= 0 && curseIdx < len(actor.ActiveEffects) {
						actor.ActiveEffects = append(actor.ActiveEffects[:curseIdx], actor.ActiveEffects[curseIdx+1:]...)
					}
				}

				roll := rng.Float64() * 100.0
				if roll < greatLuckChance {
					// Đại Cát
					a.Damage *= 3.0
					operationDamageScale = 3.0
					actor.Mana += a.ManaCost
					if actor.Mana > actor.MaxMana {
						actor.Mana = actor.MaxMana
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "LUCK_ROULETTE",
						Detail:    fmt.Sprintf("[Vận Đạo - Đại Cát!] Vận thế ngút trời! [%s] bạo phát x3.0 sát thương và phản hồi 100%% Chân Nguyên!", a.Name),
					})
				} else if roll < greatLuckChance+45.0 {
					// Trung Cát
					a.Damage *= 1.5
					operationDamageScale = 1.5
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "LUCK_ROULETTE",
						Detail:    fmt.Sprintf("[Vận Đạo - Trung Cát!] Khí vận hanh thông, [%s] gia tăng x1.5 sát thương!", a.Name),
					})
				} else if roll < greatLuckChance+45.0+30.0 {
					// Bình Vận
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "LUCK_ROULETTE",
						Detail:    fmt.Sprintf("[Vận Đạo - Bình Vận] Vận thế bình ổn, [%s] gây sát thương cơ bản.", a.Name),
					})
				} else {
					// Đại Hung
					a.Damage *= 0.5
					operationDamageScale = 0.5
					selfDmg := actor.HP * 0.10
					if selfDmg < 1.0 {
						selfDmg = 1.0
					}
					actor.HP -= selfDmg
					if actor.HP < 1.0 {
						actor.HP = 1.0
					}
					actor.ActiveEffects = append(actor.ActiveEffects, Effect{
						ID: statusfx.StatusLuckCurseReverse, Type: "buff", Value: 1, Duration: 2,
					})
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "LUCK_ROULETTE",
						Detail:    fmt.Sprintf("[Vận Đạo - Đại Hung!] Khí vận phản phệ, [%s] giảm 50%% sát thương, tự tổn %.0f HP! (Kích hoạt [Họa Hề Phúc Sở Ỷ]: Tăng mạnh vận khí hiệp kế!)", a.Name, selfDmg),
					})
				}
			}

			// Mộng Đạo: Giải Mộng Hóa Thần (DREAM_UNRAVEL)
			if hasMechanic(a.Mechanics, "ON_CAST", "DREAM_UNRAVEL") || hasMechanic(a.Mechanics, "ON_ACTION", "DREAM_UNRAVEL") {
				dispelledCount := 0
				newEffects := make([]Effect, 0, len(actor.ActiveEffects))
				for _, eff := range actor.ActiveEffects {
					if isDispellableEffect(eff.ID) {
						dispelledCount++
					} else {
						newEffects = append(newEffects, eff)
					}
				}
				actor.ActiveEffects = newEffects
				actor.IsStunned = false
				actor.IsFrozen = false
				actor.IsSleeping = false
				actor.IsDreamPrison = false
				actor.IsBound = false
				actor.IsSilenced = false
				actor.IsFeared = false

				manaGain := actor.MaxMana * (0.15 * float64(dispelledCount+1))
				shieldGain := actor.MaxHP * (0.10 * float64(dispelledCount+1))
				actor.Mana += manaGain
				if actor.Mana > actor.MaxMana {
					actor.Mana = actor.MaxMana
				}
				actor.VirtualShield += shieldGain

				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "DREAM_UNRAVEL",
					Detail:    fmt.Sprintf("[Mộng Đạo - Giải Mộng] %s giải trừ %d tầng mê chướng tâm lý, chuyển hóa thành %.0f Chân Nguyên và %.0f Mộng Giáp hộ thể!", actor.Name, dispelledCount, manaGain, shieldGain),
				})
			}

			// Huyết Đạo: Huyết Tế Cuồng Bạo (BLOOD_SACRIFICE)
			if hasMechanic(a.Mechanics, "ON_CAST", "BLOOD_SACRIFICE") || hasMechanic(a.Mechanics, "ON_ACTION", "BLOOD_SACRIFICE") {
				selfDmg := actor.HP * 0.15
				if selfDmg < 1.0 {
					selfDmg = 1.0
				}
				actor.HP -= selfDmg
				if actor.HP < 1.0 {
					actor.HP = 1.0
				}
				a.Damage *= 2.0
				operationDamageScale *= 2.0
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BLOOD_SACRIFICE",
					Detail:    fmt.Sprintf("[Huyết Đạo - Huyết Tế Cuồng Bạo] %s trích huyết hiến tế (tự tổn %.0f HP), kích phát sát thương [%s] tăng gấp đôi x2.0!", actor.Name, selfDmg, a.Name),
				})
			}

			// Thời Đạo: Quang Âm Đảo Lưu (TIME_REWIND)
			if hasMechanic(a.Mechanics, "ON_CAST", "TIME_REWIND") || hasMechanic(a.Mechanics, "ON_ACTION", "TIME_REWIND") {
				healAmt := actor.MaxHP * 0.35
				manaAmt := actor.MaxMana * 0.40
				actor.HP += healAmt
				if actor.HP > actor.MaxHP {
					actor.HP = actor.MaxHP
				}
				actor.Mana += manaAmt
				if actor.Mana > actor.MaxMana {
					actor.Mana = actor.MaxMana
				}
				cleansed := DispelActorStatuses(actor, statusfx.CategoryMentalCC, 1)
				cleansedStr := ""
				if len(cleansed) > 0 {
					cleansedStr = fmt.Sprintf(", tiêu trừ trạng thái [%s]", cleansed[0])
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "TIME_REWIND",
					Detail:    fmt.Sprintf("[Thời Đạo - Quang Âm Đảo Lưu] %s nghịch chuyển thời không, phục hồi %.0f HP và %.0f Chân Nguyên%s!", actor.Name, healAmt, manaAmt, cleansedStr),
				})
			}

			// Trí Đạo: Động Tất Sơ Hở (CALCULATED_STRIKE)
			if hasMechanic(a.Mechanics, "ON_CAST", "CALCULATED_STRIKE") || hasMechanic(a.Mechanics, "ON_ACTION", "CALCULATED_STRIKE") {
				a.Damage *= 1.35
				operationDamageScale *= 1.35
				targetName := "kẻ địch"
				if firstTarget != nil {
					targetName = firstTarget.Name
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "CALCULATED_STRIKE",
					Detail:    fmt.Sprintf("[Trí Đạo - Động Tất Sơ Hở] %s nhìn thấu toàn bộ sơ hở của %s, phát động đòn đánh tất trúng gia tăng 35%% uy lực!", actor.Name, targetName),
				})
			}

			// Hồn Đạo: Đãng Hồn Kích (SOUL_SHOCK)
			if hasMechanic(a.Mechanics, "ON_ACTION", "SOUL_SHOCK") || hasMechanic(a.Mechanics, "ON_HIT", "SOUL_SHOCK") {
				if firstTarget != nil {
					firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
						ID: statusfx.StatusSoulWound, Type: "debuff", Value: 20, Duration: 3,
					})
				}
				targetName := "kẻ địch"
				if firstTarget != nil {
					targetName = firstTarget.Name
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SOUL_SHOCK",
					Detail:    fmt.Sprintf("[Hồn Đạo - Đãng Hồn Kích] %s chấn động hồn phách, đánh thẳng vào Thần Hồn của %s vô thị Chân Nguyên Giáp và gây Hồn Thương!", actor.Name, targetName),
				})
			}

			// Hồn Đạo: Phệ Hồn Hóa Lực (SOUL_DEVOUR)
			if hasMechanic(a.Mechanics, "ON_ACTION", "SOUL_DEVOUR") {
				healAmt := actor.MaxHP * 0.20
				actor.HP += healAmt
				if actor.HP > actor.MaxHP {
					actor.HP = actor.MaxHP
				}
				if actor.CustomState == nil {
					actor.CustomState = make(map[string]int)
				}
				actor.CustomState["soul_devour_stacks"]++
				stk := actor.CustomState["soul_devour_stacks"]
				if stk > 5 {
					stk = 5
					actor.CustomState["soul_devour_stacks"] = 5
				}
				targetName := "kẻ địch"
				if firstTarget != nil {
					targetName = firstTarget.Name
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SOUL_DEVOUR",
					Detail:    fmt.Sprintf("[Hồn Đạo - Phệ Hồn Hóa Lực] %s cắn nuốt hồn lực của %s, hồi phục %.0f HP và tăng 10%% uy lực tấn công (tầng %d)!", actor.Name, targetName, healAmt, stk),
				})
			}

			// Lực Đạo: Thú Lực Hư Ảnh (BEAST_PHANTOM_STRIKE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "BEAST_PHANTOM_STRIKE") || hasMechanic(a.Mechanics, "ON_HIT", "BEAST_PHANTOM_STRIKE")) && firstTarget != nil {
				if rng.Float64() < 0.45 {
					phantomDmg := a.Damage * 0.75
					if phantomDmg < 50.0 {
						phantomDmg = 50.0
					}
					firstTarget.HP -= phantomDmg
					if firstTarget.HP < 0 {
						firstTarget.HP = 0
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "BEAST_PHANTOM_STRIKE",
						TargetId:  firstTarget.ID,
						Damage:    phantomDmg,
						Detail:    fmt.Sprintf("[Lực Đạo - Thú Lực Hư Ảnh] Tiếng thú gầm thét kinh thiên! Hư Ảnh bồi thêm một đòn bạt sơn bạt hải lên %s, gây thêm %.0f sát thương!", firstTarget.Name, phantomDmg),
					})
				}
			}

			// Huyết Đạo: Huyết Thao Thiết (HEAVY_LIFESTEAL)
			if hasMechanic(a.Mechanics, "ON_ACTION", "HEAVY_LIFESTEAL") {
				a.LifeStealPct = 0.50
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "HEAVY_LIFESTEAL",
					Detail:    fmt.Sprintf("[Huyết Đạo - Huyết Thao Thiết] %s vận chuyển huyết vụ, kích hoạt 50%% hút máu từ sát thương gây ra!", actor.Name),
				})
			}

			// Huyết Đạo: Huyết Bạo Trùng Điệp (BLEED_BURST)
			if hasMechanic(a.Mechanics, "ON_ACTION", "BLEED_BURST") && firstTarget != nil {
				bleedStacks := 0
				remainingEffects := make([]Effect, 0, len(firstTarget.ActiveEffects))
				for _, eff := range firstTarget.ActiveEffects {
					if eff.ID == statusfx.StatusBleed || strings.EqualFold(eff.ID, EffectBleed) {
						bleedStacks++
					} else {
						remainingEffects = append(remainingEffects, eff)
					}
				}
				if bleedStacks > 0 {
					firstTarget.ActiveEffects = remainingEffects
					detonateDmg := float64(bleedStacks*100) + firstTarget.MaxHP*0.08
					firstTarget.HP -= detonateDmg
					if firstTarget.HP < 0 {
						firstTarget.HP = 0
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "BLEED_BURST",
						TargetId:  firstTarget.ID,
						Damage:    detonateDmg,
						Detail:    fmt.Sprintf("[Huyết Đạo - Huyết Bạo Trùng Điệp] Kích nổ %d tầng huyết dịch trên người %s, tạo ra vụ nổ máu xé rách gây %.0f sát thương trực diện!", bleedStacks, firstTarget.Name, detonateDmg),
					})
				}
			}

			// Trộm Đạo: Thâu Chân Nguyên (MANA_STEAL)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "MANA_STEAL") || hasMechanic(a.Mechanics, "ON_HIT", "MANA_STEAL")) && firstTarget != nil {
				stolenMana := firstTarget.Mana * 0.25
				if stolenMana > 0 {
					firstTarget.Mana -= stolenMana
					actor.Mana += stolenMana
					if actor.Mana > actor.MaxMana {
						actor.Mana = actor.MaxMana
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "MANA_STEAL",
						TargetId:  firstTarget.ID,
						Detail:    fmt.Sprintf("[Trộm Đạo - Thâu Chân Nguyên] %s thôi động diệu thủ cướp đoạt %.0f Chân Nguyên từ %s nạp vào bản thân!", actor.Name, stolenMana, firstTarget.Name),
					})
				}
			}

			// Trộm Đạo: Thâu Thiên Hoán Nhật (BUFF_STEAL)
			if hasMechanic(a.Mechanics, "ON_ACTION", "BUFF_STEAL") && firstTarget != nil {
				stolenBuffName := ""
				targetRemaining := make([]Effect, 0, len(firstTarget.ActiveEffects))
				for _, eff := range firstTarget.ActiveEffects {
					if stolenBuffName == "" && eff.Type == "buff" && eff.ID != "invulnerable" {
						stolenBuffName = eff.ID
						actor.ActiveEffects = append(actor.ActiveEffects, eff)
					} else {
						targetRemaining = append(targetRemaining, eff)
					}
				}
				if stolenBuffName != "" {
					firstTarget.ActiveEffects = targetRemaining
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "BUFF_STEAL",
						TargetId:  firstTarget.ID,
						Detail:    fmt.Sprintf("[Trộm Đạo - Thâu Thiên Hoán Nhật] %s trộm lấy buff [%s] của %s mang về cho mình!", actor.Name, stolenBuffName, firstTarget.Name),
					})
				}
			}

			// Nô Đạo: Quân Đoàn Hợp Kích (PACK_FRENZY)
			if hasMechanic(a.Mechanics, "ON_ACTION", "PACK_FRENZY") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "PACK_FRENZY") {
				minionCount := 0
				combatantsList := state.Allies
				if !isAlly {
					combatantsList = state.Enemies
				}
				for _, c := range combatantsList {
					if c != nil && c.Kind == "Minion" && c.HP > 0 {
						minionCount++
					}
				}
				if minionCount > 0 {
					bonusScale := 1.0 + float64(minionCount)*0.15
					a.Damage *= bonusScale
					operationDamageScale *= bonusScale
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "PACK_FRENZY",
						Detail:    fmt.Sprintf("[Nô Đạo - Quân Đoàn Hợp Kích] %s chỉ huy %d dị thú bầy đàn hiệp đồng tác chiến, tăng %.0f%% uy lực sát thương!", actor.Name, minionCount, float64(minionCount)*15.0),
					})
				}
			}

			// Biến Hóa Đạo: Hóa Thân Cự Hùng (FORM_BEAR)
			if hasMechanic(a.Mechanics, "ON_CAST", "FORM_BEAR") || hasMechanic(a.Mechanics, "ON_ACTION", "FORM_BEAR") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusBearForm, Duration: 3,
				})
				shieldGain := actor.MaxHP * 0.25
				actor.VirtualShield += shieldGain
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FORM_BEAR",
					Detail:    fmt.Sprintf("[Biến Hóa Đạo - Hóa Thân Cự Hùng] %s hóa thân Viễn Cổ Cự Hùng, tăng 30%% Phòng thủ và ngưng tụ %.0f Hùng Giáp hộ thể!", actor.Name, shieldGain),
				})
			}

			// Biến Hóa Đạo: Hóa Thân Kim Điêu (FORM_EAGLE)
			if hasMechanic(a.Mechanics, "ON_CAST", "FORM_EAGLE") || hasMechanic(a.Mechanics, "ON_ACTION", "FORM_EAGLE") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusEagleForm, Duration: 3,
				})
				actor.Speed += 15
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FORM_EAGLE",
					Detail:    fmt.Sprintf("[Biến Hóa Đạo - Hóa Thân Kim Điêu] %s hóa thân Vân Lạc Kim Điêu, giương cánh tăng vọt Tốc độ và Né tránh!", actor.Name),
				})
			}

			// Biến Hóa Đạo: Vạn Vật Thích Ứng (DAO_ADAPTIVE_RESIST)
			if hasMechanic(a.Mechanics, "ON_ACTION", "DAO_ADAPTIVE_RESIST") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "DAO_ADAPTIVE_RESIST") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusAdaptiveResist, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "DAO_ADAPTIVE_RESIST",
					Detail:    fmt.Sprintf("[Biến Hóa Đạo - Vạn Vật Thích Ứng] %s cơ thể biến ảo thích ứng với đạo ngân công kích, tăng 25%% giảm thương nhận vào!", actor.Name),
				})
			}

			// Độc Đạo: Vạn Độc Bộc Phát (POISON_DETONATE)
			if hasMechanic(a.Mechanics, "ON_ACTION", "POISON_DETONATE") && firstTarget != nil {
				totalPoisonVal := 0.0
				keptEffects := make([]Effect, 0, len(firstTarget.ActiveEffects))
				for _, eff := range firstTarget.ActiveEffects {
					if eff.ID == statusfx.StatusPoison {
						totalPoisonVal += float64(eff.Value)
					} else {
						keptEffects = append(keptEffects, eff)
					}
				}
				if totalPoisonVal > 0 {
					firstTarget.ActiveEffects = keptEffects
					detonateDmg := totalPoisonVal * 2.5
					if detonateDmg < 60.0 {
						detonateDmg = 60.0
					}
					firstTarget.HP -= detonateDmg
					if firstTarget.HP < 0 {
						firstTarget.HP = 0
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "POISON_DETONATE",
						TargetId:  firstTarget.ID,
						Damage:    detonateDmg,
						Detail:    fmt.Sprintf("[Độc Đạo - Vạn Độc Bộc Phát] Kích nổ toàn bộ kịch độc trên người %s, độc khí nổ tung gây %.0f sát thương bạo liệt!", firstTarget.Name, detonateDmg),
					})
				}
			}

			// Kiếm Đạo: Vô Kiên Bất Tồi (SWORD_ARMOR_PIERCE)
			if hasMechanic(a.Mechanics, "ON_ACTION", "SWORD_ARMOR_PIERCE") || hasMechanic(a.Mechanics, "ON_HIT", "SWORD_ARMOR_PIERCE") {
				a.Damage *= 1.25
				operationDamageScale *= 1.25
				actor.CritRateBonus += 0.25
				targetName := "kẻ địch"
				if firstTarget != nil {
					targetName = firstTarget.Name
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SWORD_ARMOR_PIERCE",
					Detail:    fmt.Sprintf("[Kiếm Đạo - Vô Kiên Bất Tồi] %s ngưng tụ vô thượng Kiếm Ý, kiếm quang sắc bén xuyên phá hộ giáp của %s tăng 25%% sát thương!", actor.Name, targetName),
				})
			}

			// Đao Đạo: Trảm Diệt Đao Ý (EXECUTE_STRIKE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "EXECUTE_STRIKE") || hasMechanic(a.Mechanics, "ON_HIT", "EXECUTE_STRIKE")) && firstTarget != nil {
				if firstTarget.MaxHP > 0 && (firstTarget.HP/firstTarget.MaxHP) <= 0.35 {
					a.Damage *= 2.0
					operationDamageScale *= 2.0
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "EXECUTE_STRIKE",
						TargetId:  firstTarget.ID,
						Detail:    fmt.Sprintf("[Đao Đạo - Trảm Diệt Đao Ý] Nhận thấy %s suy yếu (dưới 35%% HP), %s chém ra Tuyệt Mệnh Nhất Đao nhân đôi x2.0 uy lực trảm sát!", firstTarget.Name, actor.Name),
					})
				}
			}

			// Cốt Đạo: Cốt Thứ Xuyên Tâm (BONE_SPIKE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "BONE_SPIKE") || hasMechanic(a.Mechanics, "ON_HIT", "BONE_SPIKE")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects,
					Effect{ID: statusfx.StatusBleed, Value: 30, Duration: 3},
					Effect{ID: statusfx.StatusArmorBreak, Value: 30, Duration: 3},
				)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BONE_SPIKE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Cốt Đạo - Cốt Thứ Xuyên Tâm] Phóng ra gai xương đâm xuyên huyết nhục %s, gây Xuất Huyết và Phá Giáp 30%%!", firstTarget.Name),
				})
			}

			// Cốt Đạo: Bạch Cốt Gai Khôi (BONE_THORNS)
			if hasMechanic(a.Mechanics, "ON_CAST", "BONE_THORNS") || hasMechanic(a.Mechanics, "ON_ACTION", "BONE_THORNS") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusBoneArmor, Duration: 3,
				})
				shieldGain := actor.MaxHP * 0.20
				actor.VirtualShield += shieldGain
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BONE_THORNS",
					Detail:    fmt.Sprintf("[Cốt Đạo - Bạch Cốt Gai Khôi] %s ngưng tụ cốt giáp gai nhọn hộ thể, tăng 20%% Phòng thủ, 35%% Phản thương và nhận %.0f Cốt Giáp!", actor.Name, shieldGain),
				})
			}

			// Thực Đạo: Thôn Phệ Bổ Khiếu (MANA_EAT)
			if hasMechanic(a.Mechanics, "ON_ACTION", "MANA_EAT") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "MANA_EAT") {
				manaGain := actor.MaxMana * 0.25
				actor.Mana += manaGain
				if actor.Mana > actor.MaxMana {
					actor.Mana = actor.MaxMana
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "MANA_EAT",
					Detail:    fmt.Sprintf("[Thực Đạo - Thôn Phệ Bổ Khiếu] %s nuốt trọn năng lượng dư thừa, chuyển hóa thành %.0f Chân Nguyên nạp đầy Không Khiếu!", actor.Name, manaGain),
				})
			}

			// Kim Đạo: Cương Thiết Kim Thân (METAL_AEGIS)
			if hasMechanic(a.Mechanics, "ON_CAST", "METAL_AEGIS") || hasMechanic(a.Mechanics, "ON_ACTION", "METAL_AEGIS") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusMetalAegis, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "METAL_AEGIS",
					Detail:    fmt.Sprintf("[Kim Đạo - Cương Thiết Kim Thân] %s ngưng tụ kim quang hộ thể, tăng 35%% Phòng thủ và giảm 15%% sát thương nhận vào!", actor.Name),
				})
			}

			// Kim Đạo: Phá Giáp Kim Mang (ARMOR_BREAK_HEAVY)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "ARMOR_BREAK_HEAVY") || hasMechanic(a.Mechanics, "ON_HIT", "ARMOR_BREAK_HEAVY")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusArmorBreak, Value: 40, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "ARMOR_BREAK_HEAVY",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Kim Đạo - Phá Giáp Kim Mang] Đòn đánh xé toạc hộ giáp của %s, giảm 40%% phòng thủ trong 3 lượt!", firstTarget.Name),
				})
			}

			// Mộc Đạo: Sinh Sinh Bất Tức (WOOD_REGEN_BURST)
			if hasMechanic(a.Mechanics, "ON_CAST", "WOOD_REGEN_BURST") || hasMechanic(a.Mechanics, "ON_ACTION", "WOOD_REGEN_BURST") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusRegen, Value: 10, Duration: 4,
				})
				manaGain := actor.MaxMana * 0.15
				actor.Mana = math.Min(actor.MaxMana, actor.Mana+manaGain)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "WOOD_REGEN_BURST",
					Detail:    fmt.Sprintf("[Mộc Đạo - Sinh Sinh Bất Tức] %s vận khởi linh mộc sinh cơ, hồi phục %.0f Chân Nguyên và kích hoạt hồi máu liên tục 4 lượt!", actor.Name, manaGain),
				})
			}

			// Mộc Đạo: Mộc Lao Trói Buộc (WOOD_BIND)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "WOOD_BIND") || hasMechanic(a.Mechanics, "ON_HIT", "WOOD_BIND")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusBind, Duration: 2,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "WOOD_BIND",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Mộc Đạo - Mộc Lao Trói Buộc] Cổ thụ rễ cây trồi lên trói chặt %s, phong tỏa hành động và triệt tiêu khả năng né tránh!", firstTarget.Name),
				})
			}

			// Thủy Đạo: Thủy Lưu Triều Tịch (WATER_CLEANSE)
			if hasMechanic(a.Mechanics, "ON_CAST", "WATER_CLEANSE") || hasMechanic(a.Mechanics, "ON_ACTION", "WATER_CLEANSE") {
				newEffects := make([]Effect, 0, len(actor.ActiveEffects))
				cleansedCount := 0
				for _, eff := range actor.ActiveEffects {
					if eff.ID == statusfx.StatusBurn || eff.ID == statusfx.StatusBleed || eff.ID == statusfx.StatusPoison {
						cleansedCount++
					} else {
						newEffects = append(newEffects, eff)
					}
				}
				actor.ActiveEffects = newEffects
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "WATER_CLEANSE",
					Detail:    fmt.Sprintf("[Thủy Đạo - Thủy Lưu Triều Tịch] Dòng nước tinh khiết bao bọc %s, cuốn trôi %d hiệu ứng thiêu đốt, độc và xuất huyết!", actor.Name, cleansedCount),
				})
			}

			// Thủy Đạo: Thủy Kính Hóa Giải (WATER_MIRROR_SHIELD)
			if hasMechanic(a.Mechanics, "ON_CAST", "WATER_MIRROR_SHIELD") || hasMechanic(a.Mechanics, "ON_ACTION", "WATER_MIRROR_SHIELD") {
				shieldGain := actor.MaxHP * 0.20
				healGain := actor.MaxHP * 0.10
				actor.VirtualShield += shieldGain
				actor.HP = math.Min(actor.MaxHP, actor.HP+healGain)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "WATER_MIRROR_SHIELD",
					Detail:    fmt.Sprintf("[Thủy Đạo - Thủy Kính Hóa Giải] %s ngưng tụ gương nước phản xạ, tạo %.0f Khiên Ảo và hồi phục %.0f HP!", actor.Name, shieldGain, healGain),
				})
			}

			// Hỏa Đạo: Liệt Hỏa Liêu Nguyên (FIRE_INFERNO)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "FIRE_INFERNO") || hasMechanic(a.Mechanics, "ON_HIT", "FIRE_INFERNO")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusBurn, Value: 8, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FIRE_INFERNO",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Hỏa Đạo - Liệt Hỏa Liêu Nguyên] Ngọn lửa hừng hực thiêu đốt %s, mỗi lượt cắn nuốt 8%% sinh lực!", firstTarget.Name),
				})
			}

			// Hỏa Đạo: Nộ Hỏa Bạo Phát (IGNITE_BURST)
			if hasMechanic(a.Mechanics, "ON_ACTION", "IGNITE_BURST") && firstTarget != nil {
				hasBurn := false
				for _, eff := range firstTarget.ActiveEffects {
					if eff.ID == statusfx.StatusBurn {
						hasBurn = true
						break
					}
				}
				if hasBurn {
					a.Damage *= 1.60
					operationDamageScale *= 1.60
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "IGNITE_BURST",
						TargetId:  firstTarget.ID,
						Detail:    fmt.Sprintf("[Hỏa Đạo - Nộ Hỏa Bạo Phát] Mục tiêu %s đang bốc cháy dữ dội! Đòn đánh phát nổ cộng hưởng tăng vọt 60%% sát thương!", firstTarget.Name),
					})
				}
			}

			// Thổ Đạo: Đại Địa Hộ Thuẫn (EARTH_SHIELD)
			if hasMechanic(a.Mechanics, "ON_CAST", "EARTH_SHIELD") || hasMechanic(a.Mechanics, "ON_ACTION", "EARTH_SHIELD") {
				shieldGain := actor.MaxHP * 0.30
				actor.VirtualShield += shieldGain
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusDefUp, Value: 25, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "EARTH_SHIELD",
					Detail:    fmt.Sprintf("[Thổ Đạo - Đại Địa Hộ Thuẫn] %s dẫn động địa khí, tạo %.0f Thổ Nham Giáp và tăng 25%% phòng thủ!", actor.Name, shieldGain),
				})
			}

			// Thổ Đạo: Địa Chấn Trọng Lực (EARTH_GRAVITY)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "EARTH_GRAVITY") || hasMechanic(a.Mechanics, "ON_HIT", "EARTH_GRAVITY")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusSlow, Value: 40, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "EARTH_GRAVITY",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Thổ Đạo - Địa Chấn Trọng Lực] Trọng lực đại địa giáng xuống đè nặng %s, làm chậm 40%% tốc độ!", firstTarget.Name),
				})
			}

			// Phong Đạo: Tật Phong Tốc Độ (WIND_HASTE_BURST)
			if hasMechanic(a.Mechanics, "ON_CAST", "WIND_HASTE_BURST") || hasMechanic(a.Mechanics, "ON_ACTION", "WIND_HASTE_BURST") {
				actor.ActiveEffects = append(actor.ActiveEffects,
					Effect{ID: statusfx.StatusHaste, Value: 40, Duration: 3},
					Effect{ID: statusfx.StatusEvasion, Value: 25, Duration: 3},
				)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "WIND_HASTE_BURST",
					Detail:    fmt.Sprintf("[Phong Đạo - Tật Phong Tốc Độ] %s ngự phong phi hành, thân pháp mau lẹ tăng 40%% Tốc độ và 25%% Né tránh!", actor.Name),
				})
			}

			// Lôi Đạo: Lôi Đình Vạn Quân (LIGHTNING_STUN_BURST)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "LIGHTNING_STUN_BURST") || hasMechanic(a.Mechanics, "ON_HIT", "LIGHTNING_STUN_BURST")) && firstTarget != nil {
				a.Damage *= 1.30
				operationDamageScale *= 1.30
				stunned := false
				if rng.Float64() < 0.40 {
					firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
						ID: statusfx.StatusStun, Duration: 1,
					})
					stunned = true
				}
				stunStr := ""
				if stunned {
					stunStr = " và gây Choáng váng 1 lượt"
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "LIGHTNING_STUN_BURST",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Lôi Đạo - Lôi Đình Vạn Quân] %s giáng cuồng lôi tăng 30%% uy lực%s lên %s!", actor.Name, stunStr, firstTarget.Name),
				})
			}

			// Lôi Đạo: Lôi Điện Truyền Dẫn (CHAIN_LIGHTNING)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "CHAIN_LIGHTNING") || hasMechanic(a.Mechanics, "ON_HIT", "CHAIN_LIGHTNING")) && len(enemyTargets) > 1 {
				chainDmg := a.Damage * 0.50
				if chainDmg < 30.0 {
					chainDmg = 30.0
				}
				for _, enemy := range enemyTargets {
					if enemy != nil && enemy.HP > 0 && enemy != firstTarget {
						enemy.HP -= chainDmg
						if enemy.HP < 0 {
							enemy.HP = 0
						}
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "CHAIN_LIGHTNING",
							TargetId:  enemy.ID,
							Damage:    chainDmg,
							Detail:    fmt.Sprintf("[Lôi Đạo - Lôi Điện Truyền Dẫn] Tia lôi điện phóng lan sang %s, gây thêm %.0f sát thương điện giật!", enemy.Name, chainDmg),
						})
						break
					}
				}
			}

			// Băng Đạo: Băng Phong Vạn Lý (ICE_DEEP_FREEZE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "ICE_DEEP_FREEZE") || hasMechanic(a.Mechanics, "ON_HIT", "ICE_DEEP_FREEZE")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusFreeze, Duration: 1,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "ICE_DEEP_FREEZE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Băng Đạo - Băng Phong Vạn Lý] Hàn khí ngưng kết đóng băng %s thành tượng tuyết, phong tỏa hoàn toàn hành động!", firstTarget.Name),
				})
			}

			// Băng Đạo: Hàn Băng Tê Cóng (ICE_FROSTBITE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "ICE_FROSTBITE") || hasMechanic(a.Mechanics, "ON_HIT", "ICE_FROSTBITE")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusSlowDamage, Value: 35, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "ICE_FROSTBITE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Băng Đạo - Hàn Băng Tê Cóng] Băng sương ăn mòn da thịt %s, vừa cấu rỉa sinh lực vừa làm chậm 35%% tốc độ!", firstTarget.Name),
				})
			}

			// Quang Đạo: Manh Mục Cực Quang (LIGHT_BLINDING_FLASH)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "LIGHT_BLINDING_FLASH") || hasMechanic(a.Mechanics, "ON_HIT", "LIGHT_BLINDING_FLASH")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusBlind, Value: 60, Duration: 2,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "LIGHT_BLINDING_FLASH",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Quang Đạo - Manh Mục Cực Quang] Cực quang chói lòa chiếu thẳng vào mắt %s, giảm mạnh 60%% độ chính xác!", firstTarget.Name),
				})
			}

			// Quang Đạo: Thánh Quang Phục Thể (LIGHT_PURIFY)
			if hasMechanic(a.Mechanics, "ON_CAST", "LIGHT_PURIFY") || hasMechanic(a.Mechanics, "ON_ACTION", "LIGHT_PURIFY") {
				healGain := actor.MaxHP * 0.25
				actor.HP = math.Min(actor.MaxHP, actor.HP+healGain)
				DispelActorStatuses(actor, statusfx.CategoryMentalCC, 1)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "LIGHT_PURIFY",
					Detail:    fmt.Sprintf("[Quang Đạo - Thánh Quang Phục Thể] Ánh sáng ấm áp bao phủ %s, hồi phục %.0f HP và thanh tẩy uế khí!", actor.Name, healGain),
				})
			}

			// Ám Đạo: Ám Dạ Ẩn Nấp (DARK_STEALTH)
			if hasMechanic(a.Mechanics, "ON_CAST", "DARK_STEALTH") || hasMechanic(a.Mechanics, "ON_ACTION", "DARK_STEALTH") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusStealth, Duration: 2,
				})
				actor.CritRateBonus += 0.40
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "DARK_STEALTH",
					Detail:    fmt.Sprintf("[Ám Đạo - Ám Dạ Ẩn Nấp] %s hòa tan vào bóng đêm vô ảnh vô hình, chuẩn bị đòn đánh bạo kích từ bóng tối!", actor.Name),
				})
			}

			// Ám Đạo: Ám Ảnh Hủ Thực (DARK_CURSE_WEAK)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "DARK_CURSE_WEAK") || hasMechanic(a.Mechanics, "ON_HIT", "DARK_CURSE_WEAK")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusWeak, Value: 30, Duration: 3,
				})
				firstTarget.Mana = math.Max(0, firstTarget.Mana-firstTarget.MaxMana*0.15)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "DARK_CURSE_WEAK",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Ám Đạo - Ám Ảnh Hủ Thực] Khí tức u tối nguyền rủa %s, làm suy yếu 30%% sát thương và ăn mòn Chân Nguyên!", firstTarget.Name),
				})
			}

			// Tinh Đạo: Tinh Quang Lạc Trần (STAR_FIRE_RAIN)
			if hasMechanic(a.Mechanics, "ON_ACTION", "STAR_FIRE_RAIN") {
				a.Damage *= 1.30
				operationDamageScale *= 1.30
				starShield := actor.MaxHP * 0.15
				actor.VirtualShield += starShield
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "STAR_FIRE_RAIN",
					Detail:    fmt.Sprintf("[Tinh Đạo - Tinh Quang Lạc Trần] Hàng ngàn ngôi sao sa oanh tạc phe địch, ngưng tụ %.0f Khiên Tinh Quang hộ thể!", starShield),
				})
			}

			// Tinh Đạo: Tinh Niệm Tụ Hồn (STAR_THOUGHT_RESTORE)
			if hasMechanic(a.Mechanics, "ON_CAST", "STAR_THOUGHT_RESTORE") || hasMechanic(a.Mechanics, "ON_ACTION", "STAR_THOUGHT_RESTORE") {
				manaGain := actor.MaxMana * 0.30
				actor.Mana = math.Min(actor.MaxMana, actor.Mana+manaGain)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "STAR_THOUGHT_RESTORE",
					Detail:    fmt.Sprintf("[Tinh Đạo - Tinh Niệm Tụ Hồn] %s câu thông vạn dặm tinh hà, niệm đầu thông suốt phục hồi %.0f Chân Nguyên!", actor.Name, manaGain),
				})
			}

			// Âm Đạo: Ma Âm Quán Nhĩ (SOUND_DISRUPT_SILENCE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "SOUND_DISRUPT_SILENCE") || hasMechanic(a.Mechanics, "ON_HIT", "SOUND_DISRUPT_SILENCE")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusSilence, Duration: 2,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SOUND_DISRUPT_SILENCE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Âm Đạo - Ma Âm Quán Nhĩ] Sóng âm chói tai đâm xuyên màng nhĩ %s, khóa chặt kỹ năng chỉ có thể đánh thường!", firstTarget.Name),
				})
			}

			// Âm Đạo: Ba Động Chấn Thần (SOUND_RESONANCE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "SOUND_RESONANCE") || hasMechanic(a.Mechanics, "ON_HIT", "SOUND_RESONANCE")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusSoulWound, Value: 25, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SOUND_RESONANCE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Âm Đạo - Ba Động Chấn Thần] Dao động âm ba cộng hưởng nội tạng %s, bỏ qua Chân Nguyên Giáp và gây Hồn Thương!", firstTarget.Name),
				})
			}

			// Vân Đạo: Vân Vụ Ảo Mộng (CLOUD_MIST)
			if hasMechanic(a.Mechanics, "ON_CAST", "CLOUD_MIST") || hasMechanic(a.Mechanics, "ON_ACTION", "CLOUD_MIST") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusCloudMist, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "CLOUD_MIST",
					Detail:    fmt.Sprintf("[Vân Đạo - Vân Vụ Ảo Mộng] Mây mù huyền ảo cuộn quanh %s, tăng 35%% Né tránh và giảm 20%% sát thương nhận vào!", actor.Name),
				})
			}

			// Thiên Đạo: Thiên Ý Khó Dò (HEAVEN_SUPPRESSION)
			if hasMechanic(a.Mechanics, "ON_ACTION", "HEAVEN_SUPPRESSION") && firstTarget != nil {
				stripped := 0
				kept := make([]Effect, 0, len(firstTarget.ActiveEffects))
				for _, eff := range firstTarget.ActiveEffects {
					if eff.Type == "buff" && stripped < 2 {
						stripped++
					} else {
						kept = append(kept, eff)
					}
				}
				firstTarget.ActiveEffects = kept
				a.Damage *= 1.35
				operationDamageScale *= 1.35
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "HEAVEN_SUPPRESSION",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Thiên Đạo - Thiên Ý Khó Dò] Thiên Đạo tổn có thừa mà bù không đủ! Xóa bỏ %d hiệu ứng có lợi của %s và tăng 35%% uy lực giáng phạt!", stripped, firstTarget.Name),
				})
			}

			// Thiên Đạo: Tiếp Thiên Tiếp Địa (TIEP_THIEN_DIA)
			if hasMechanic(a.Mechanics, "ON_CAST", "TIEP_THIEN_DIA") || hasMechanic(a.Mechanics, "ON_ACTION", "TIEP_THIEN_DIA") {
				manaGain := actor.MaxMana * 0.25
				actor.Mana = math.Min(actor.MaxMana, actor.Mana+manaGain)
				actor.IsStunned = false
				actor.IsFrozen = false
				actor.IsSleeping = false
				actor.IsDreamPrison = false
				actor.IsBound = false
				actor.IsSilenced = false
				actor.IsFeared = false
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "TIEP_THIEN_DIA",
					Detail:    fmt.Sprintf("[Thiên Đạo - Tiếp Thiên Tiếp Địa] %s câu thông ý chí Thiên Đạo, giải trừ toàn bộ khống chế và hồi %.0f Chân Nguyên!", actor.Name, manaGain),
				})
			}

			// Nhân Đạo: Nhân Định Thắng Thiên (HUMAN_WILL)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "HUMAN_WILL") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "HUMAN_WILL")) && actor.MaxHP > 0 {
				if (actor.HP / actor.MaxHP) <= 0.40 {
					a.Damage *= 1.50
					operationDamageScale *= 1.50
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "HUMAN_WILL",
						Detail:    fmt.Sprintf("[Nhân Đạo - Nhân Định Thắng Thiên] Nghịch cảnh bộc phát ý chí nhân đạo! %s bừng lên hy vọng bất khuất, tăng vọt 50%% uy lực chiêu thức!", actor.Name),
					})
				}
			}

			// Nhân Đạo: Vạn Dân Đồng Tâm (VAN_DAN_DONG_TAM)
			if hasMechanic(a.Mechanics, "ON_CAST", "VAN_DAN_DONG_TAM") || hasMechanic(a.Mechanics, "ON_ACTION", "VAN_DAN_DONG_TAM") {
				shieldGain := actor.MaxHP * 0.35
				actor.VirtualShield += shieldGain
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "VAN_DAN_DONG_TAM",
					Detail:    fmt.Sprintf("[Nhân Đạo - Vạn Dân Đồng Tâm] Hồng trần vạn dặm ngưng tụ nhân tâm, tạo %.0f Khiên Nhân Đạo kiên cố bảo vệ %s!", shieldGain, actor.Name),
				})
			}

			// Khí Đạo: Khí Tường Hộ Thể (QI_WALL)
			if hasMechanic(a.Mechanics, "ON_CAST", "QI_WALL") || hasMechanic(a.Mechanics, "ON_ACTION", "QI_WALL") {
				shieldGain := actor.MaxHP * 0.30
				actor.VirtualShield += shieldGain
				if actor.DamageTakenReduction < 0.30 {
					actor.DamageTakenReduction = 0.30
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "QI_WALL",
					Detail:    fmt.Sprintf("[Khí Đạo - Khí Tường Hộ Thể] %s ngưng tụ Nguyên Khí Tường vô hình, cản phá 30%% sát thương và nhận %.0f Khí Giáp!", actor.Name, shieldGain),
				})
			}

			// Khí Đạo: Khí Bạo Trảm Thiên (QI_EXPLOSION)
			if hasMechanic(a.Mechanics, "ON_ACTION", "QI_EXPLOSION") {
				a.Damage *= 1.40
				operationDamageScale *= 1.40
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "QI_EXPLOSION",
					Detail:    fmt.Sprintf("[Khí Đạo - Khí Bạo Trảm Thiên] Thiên địa nguyên khí nén ép cực hạn rồi nổ tung, sóng khí kình cuộn trào tăng 40%% sát thương của %s!", actor.Name),
				})
			}

			// Không Đạo: Hư Không Thuấn Di (SPATIAL_SHIFT)
			if hasMechanic(a.Mechanics, "ON_CAST", "SPATIAL_SHIFT") || hasMechanic(a.Mechanics, "ON_ACTION", "SPATIAL_SHIFT") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusEvasion, Value: 75, Duration: 1,
				})
				actor.EvasionChance = 0.75
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SPATIAL_SHIFT",
					Detail:    fmt.Sprintf("[Không Đạo - Hư Không Thuấn Di] %s bẻ cong không gian xung quanh, bước vào kẽ hở hư không né tránh 75%% đòn tấn công!", actor.Name),
				})
			}

			// Không Đạo: Thứ Nguyên Trảm (DIMENSION_SLASH)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "DIMENSION_SLASH") || hasMechanic(a.Mechanics, "ON_HIT", "DIMENSION_SLASH")) && firstTarget != nil {
				a.Damage *= 1.25
				operationDamageScale *= 1.25
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "DIMENSION_SLASH",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Không Đạo - Thứ Nguyên Trảm] Đao kiếm xé rách không gian, chém xuyên phòng tuyến giáng thẳng lên %s tăng 25%% uy lực!", firstTarget.Name),
				})
			}

			// Quy Tắc Đạo: Định Quy Đặt Củ (RULE_IMPOSITION)
			if hasMechanic(a.Mechanics, "ON_ACTION", "RULE_IMPOSITION") && firstTarget != nil {
				newEffects := make([]Effect, 0, len(firstTarget.ActiveEffects))
				stripped := 0
				for _, eff := range firstTarget.ActiveEffects {
					if eff.Type == "buff" {
						stripped++
					} else {
						newEffects = append(newEffects, eff)
					}
				}
				firstTarget.ActiveEffects = newEffects
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "RULE_IMPOSITION",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Quy Tắc Đạo - Định Quy Đặt Củ] Vô quy củ bất thành phương viên! Thiết lập điều luật chiến trường, xóa sạch %d Buff có lợi của %s!", stripped, firstTarget.Name),
				})
			}

			// Quy Tắc Đạo: Phán Quyết Công Lý (JUSTICE_VERDICT)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "JUSTICE_VERDICT") || hasMechanic(a.Mechanics, "ON_HIT", "JUSTICE_VERDICT")) && firstTarget != nil {
				a.Damage *= 1.35
				operationDamageScale *= 1.35
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "JUSTICE_VERDICT",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Quy Tắc Đạo - Phán Quyết Công Lý] Thiết diện vô tư, công lý phán xét trừng phạt %s tăng thêm 35%% sát thương!", firstTarget.Name),
				})
			}

			// Trận Đạo: Trận Địa Khống Chế (FORMATION_DOMAIN)
			if hasMechanic(a.Mechanics, "ON_CAST", "FORMATION_DOMAIN") || hasMechanic(a.Mechanics, "ON_ACTION", "FORMATION_DOMAIN") {
				shieldGain := actor.MaxHP * 0.20
				actor.VirtualShield += shieldGain
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusDefUp, Value: 25, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FORMATION_DOMAIN",
					Detail:    fmt.Sprintf("[Trận Đạo - Trận Địa Khống Chế] Cổ Trận khởi động bao phủ sàn đấu, tăng 25%% phòng thủ và tạo %.0f Trận Giáp cho %s!", shieldGain, actor.Name),
				})
			}

			// Huyễn Đạo: Huyễn Ảnh Mê Trận (BLINDING_MIRAGE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "BLINDING_MIRAGE") || hasMechanic(a.Mechanics, "ON_HIT", "BLINDING_MIRAGE")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects,
					Effect{ID: statusfx.StatusConfuse, Duration: 2},
					Effect{ID: statusfx.StatusBlind, Value: 40, Duration: 2},
				)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BLINDING_MIRAGE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Huyễn Đạo - Huyễn Ảnh Mê Trận] Hư ảnh trùng trùng điệp điệp làm %s rơi vào Mê Loạn và Mù quáng trong 2 lượt!", firstTarget.Name),
				})
			}

			// Họa Đạo: Họa Địa Vi Lao (PAINTING_IMPRISONMENT)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "PAINTING_IMPRISONMENT") || hasMechanic(a.Mechanics, "ON_HIT", "PAINTING_IMPRISONMENT")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusBind, Duration: 1,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "PAINTING_IMPRISONMENT",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Họa Đạo - Họa Địa Vi Lao] %s vung bút điểm mặc, nhốt chặt %s vào trong bức tranh thủy mặc phong tỏa hành động!", actor.Name, firstTarget.Name),
				})
			}

			// Tình Đạo: Tình Cổ Vô Hối (PASSION_OVERLOAD)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "PASSION_OVERLOAD") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "PASSION_OVERLOAD")) && actor.MaxHP > 0 {
				if (actor.HP / actor.MaxHP) <= 0.35 {
					a.Damage *= 1.50
					operationDamageScale *= 1.50
					actor.IsStunned = false
					actor.IsFrozen = false
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "PASSION_OVERLOAD",
						Detail:    fmt.Sprintf("[Tình Đạo - Tình Cổ Vô Hối] Tình sâu tựa hải, nghịch cảnh sinh tình! %s tiến vào trạng thái Si Tình bộc phát 50%% uy lực và kháng khống chế!", actor.Name),
					})
				}
			}

			// Luyện Đạo: Tự Bạo Cổ Trùng (GU_DETONATION)
			if hasMechanic(a.Mechanics, "ON_ACTION", "GU_DETONATION") {
				selfLoss := actor.HP * 0.05
				if selfLoss < 1.0 {
					selfLoss = 1.0
				}
				actor.HP -= selfLoss
				if actor.HP < 1.0 {
					actor.HP = 1.0
				}
				a.Damage *= 1.60
				operationDamageScale *= 1.60
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "GU_DETONATION",
					Detail:    fmt.Sprintf("[Luyện Đạo - Tự Bạo Cổ Trùng] %s quyết đoán kích nổ Chân Nguyên Cổ trùng, tạo ra làn sóng xung kích hủy diệt tăng 60%% sát thương!", actor.Name),
				})
			}

			// Sát Chiêu: Nghịch Lưu Hộ Thân Quỹ (REVERSE_FLOW_SEAL)
			if hasMechanic(a.Mechanics, "ON_ACTION", "REVERSE_FLOW_SEAL") || hasMechanic(a.Mechanics, "ON_CAST", "REVERSE_FLOW_SEAL") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusReverseFlowSeal, Duration: 3, Type: "buff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "REVERSE_FLOW_SEAL",
					Detail:    fmt.Sprintf("[Sát Chiêu - Nghịch Lưu Hộ Thân Quỹ] %s thôi động Nghịch Lưu Hà tạo thành hộ thân quỹ, phản phệ 100%% mọi sát thương nhận vào trong 3 hiệp!", actor.Name),
				})
			}

			// Trộm Đạo: Đâu Thiên Thâu Thủ (GHOST_HAND_STEAL)
			if hasMechanic(a.Mechanics, "ON_ACTION", "GHOST_HAND_STEAL") && firstTarget != nil {
				stolenMana := firstTarget.Mana * 0.30
				strippedShield := firstTarget.VirtualShield * 0.50
				firstTarget.Mana -= stolenMana
				firstTarget.VirtualShield -= strippedShield
				actor.Mana += stolenMana
				if actor.Mana > actor.MaxMana {
					actor.Mana = actor.MaxMana
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "GHOST_HAND_STEAL",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Trộm Đạo - Đâu Thiên Thâu Thủ] Quỷ thủ vô hình xuyên vào Không Khiếu của %s, cướp đoạt %.0f Chân Nguyên và bóc đi %.0f Khiên Ảo!", firstTarget.Name, stolenMana, strippedShield),
				})
			}

			// Mộng Đạo: Dẫn Hồn Nhập Mộng (LEAD_SOUL_TO_DREAM)
			if hasMechanic(a.Mechanics, "ON_ACTION", "LEAD_SOUL_TO_DREAM") && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusDreamPrison, Duration: 2,
				})
				firstTarget.IsDreamPrison = true
				soulDmg := actor.BaseDmg * 1.2
				if soulDmg < 40.0 {
					soulDmg = 40.0
				}
				firstTarget.HP -= soulDmg
				if firstTarget.HP < 0 {
					firstTarget.HP = 0
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "LEAD_SOUL_TO_DREAM",
					TargetId:  firstTarget.ID,
					Damage:    soulDmg,
					Detail:    fmt.Sprintf("[Mộng Đạo - Dẫn Hồn Nhập Mộng] Kéo hồn phách %s vào cõi mộng vĩnh hằng, gây %.0f sát thương hồn và phong ấn hoàn toàn hành động!", firstTarget.Name, soulDmg),
				})
			}

			// Lực Đạo: Vạn Ngã Đại Thủ Ấn (MYRIAD_SELF_GIANT_HAND)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "MYRIAD_SELF_GIANT_HAND") || hasMechanic(a.Mechanics, "ON_CAST", "MYRIAD_SELF_GIANT_HAND")) && firstTarget != nil {
				a.Damage *= 1.75
				operationDamageScale *= 1.75
				shatteredShield := firstTarget.VirtualShield
				firstTarget.VirtualShield = 0
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "MYRIAD_SELF_GIANT_HAND",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Lực Đạo - Vạn Ngã Đại Thủ Ấn] Bàn tay khổng lồ che trời đập xuống! Đập nát %.0f Khiên Ảo của %s và tăng 75%% uy lực sát thương!", shatteredShield, firstTarget.Name),
				})
			}

			// Biến Hóa Đạo: Gặp Mặt Từng Quen Biết (FAMILIAR_FACE)
			if hasMechanic(a.Mechanics, "ON_ACTION", "FAMILIAR_FACE") || hasMechanic(a.Mechanics, "ON_CAST", "FAMILIAR_FACE") {
				actor.ActiveEffects = append(actor.ActiveEffects,
					Effect{ID: statusfx.StatusFamiliarFace, Value: 50, Duration: 2},
					Effect{ID: statusfx.StatusEvasion, Value: 50, Duration: 2},
				)
				actor.EvasionChance = 0.50
				actor.CritRateBonus += 0.50
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FAMILIAR_FACE",
					Detail:    fmt.Sprintf("[Biến Hóa Đạo - Gặp Mặt Từng Quen Biết] %s ngụy trang biến hóa khôn lường, tăng 50%% Né Tránh và 50%% Tỉ lệ Bạo Kích!", actor.Name),
				})
			}

			// Kiếm Đạo: Ngũ Chỉ Quyền Tâm Kiếm (FIVE_FINGER_SWORD)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "FIVE_FINGER_SWORD") || hasMechanic(a.Mechanics, "ON_HIT", "FIVE_FINGER_SWORD")) && firstTarget != nil {
				a.Damage *= 1.65
				operationDamageScale *= 1.65
				actor.IgnoreDefensePct = math.Max(actor.IgnoreDefensePct, 0.50)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FIVE_FINGER_SWORD",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Kiếm Đạo - Ngũ Chỉ Quyền Tâm Kiếm] Năm ngón tay liên hoàn bắn kiếm khí (Cái, Thực, Trung, Vô Danh, Tiểu Chỉ), tăng 65%% sát thương và bỏ qua 50%% phòng ngự của %s!", firstTarget.Name),
				})
			}

			// Thời Đạo: Tuế Nguyệt Như Đao (TIME_CUTTER)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "TIME_CUTTER") || hasMechanic(a.Mechanics, "ON_HIT", "TIME_CUTTER")) && firstTarget != nil {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				lostMana := firstTarget.Mana * 0.25
				firstTarget.Mana -= lostMana
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusSlow1, Value: 35, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "TIME_CUTTER",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Thời Đạo - Tuế Nguyệt Như Đao] Năm tháng như đao chém thiên kiêu! Chém đứt sinh mệnh của %s, tăng 50%% sát thương, tước đoạt %.0f Chân Nguyên và làm chậm 35%% tuyệt đối!", firstTarget.Name, lostMana),
				})
			}

			// Thời Đạo: Xuân Tiễn Hạ Phiến (SPRING_SCISSORS_SUMMER_FAN)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "SPRING_SCISSORS_SUMMER_FAN") || hasMechanic(a.Mechanics, "ON_HIT", "SPRING_SCISSORS_SUMMER_FAN")) && firstTarget != nil {
				a.Damage *= 1.55
				operationDamageScale *= 1.55
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusArmorBreak, Value: 25, Duration: 3,
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SPRING_SCISSORS_SUMMER_FAN",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Thời Đạo - Xuân Tiễn Hạ Phiến] Kéo mùa xuân cắt đứt phòng tuyến, quạt mùa hè thiêu rụi phòng thủ! Tăng 55%% sát thương và giảm 25%% giáp của %s!", firstTarget.Name),
				})
			}

			// Băng/Kiếm Đạo: Long Tức Kiếm (DRAGON_BREATH_SWORD)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "DRAGON_BREATH_SWORD") || hasMechanic(a.Mechanics, "ON_HIT", "DRAGON_BREATH_SWORD")) && firstTarget != nil {
				a.Damage *= 1.45
				operationDamageScale *= 1.45
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusFreeze, Duration: 1,
				})
				firstTarget.IsFrozen = true
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "DRAGON_BREATH_SWORD",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Băng Kiếm - Long Tức Kiếm] Long ngâm vang dội, hàn băng kiếm khí đóng băng hoàn toàn %s và tăng 45%% sát thương!", firstTarget.Name),
				})
			}

			// Lôi Đạo: Lôi Thần Thành Phạt Tội (THUNDER_GOD_CITY)
			if hasMechanic(a.Mechanics, "ON_ACTION", "THUNDER_GOD_CITY") && len(enemyTargets) > 0 {
				var lowestHPEnemy *CombatantAuto
				for _, e := range enemyTargets {
					if e != nil && e.HP > 0 {
						if lowestHPEnemy == nil || e.HP < lowestHPEnemy.HP {
							lowestHPEnemy = e
						}
					}
				}
				if lowestHPEnemy != nil {
					strikeDmg := actor.BaseDmg * 0.9
					if strikeDmg < 35.0 {
						strikeDmg = 35.0
					}
					lowestHPEnemy.HP -= strikeDmg
					if lowestHPEnemy.HP < 0 {
						lowestHPEnemy.HP = 0
					}
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "THUNDER_GOD_CITY",
						TargetId:  lowestHPEnemy.ID,
						Damage:    strikeDmg,
						Detail:    fmt.Sprintf("[Tiên Cổ Ốc - Lôi Thần Thành] Pháo đài lôi điện oanh kích kẻ địch suy yếu nhất! Giáng cuồng lôi %.0f sát thương lên %s!", strikeDmg, lowestHPEnemy.Name),
					})
				}
			}

			// ── Sát Chiêu Tiên Đạo (Đợt 6 - Reverend Insanity Canonical Moves) ──

			// Âm Đạo: Ly Ca (SONG_OF_SEPARATION) - Phượng Cửu Ca
			if (hasMechanic(a.Mechanics, "ON_ACTION", "SONG_OF_SEPARATION") || hasMechanic(a.Mechanics, "ON_HIT", "SONG_OF_SEPARATION")) && firstTarget != nil {
				a.Damage *= 1.65
				operationDamageScale *= 1.65
				dispelled := 0
				var remainingEffects []Effect
				for _, eff := range firstTarget.ActiveEffects {
					if eff.Type == "buff" && dispelled < 2 {
						dispelled++
						continue
					}
					remainingEffects = append(remainingEffects, eff)
				}
				firstTarget.ActiveEffects = remainingEffects
				strippedShield := firstTarget.VirtualShield * 0.30
				firstTarget.VirtualShield -= strippedShield
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusWeak, Value: 25, Duration: 3, Type: "debuff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SONG_OF_SEPARATION",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Âm Đạo - Ly Ca] Khúc ca ly tán vạn vật vút lên! Tăng 65%% sát thương, giải trừ %d buff, phá hủy %.0f Giáp và khiến %s chịu trạng thái Suy Yếu (giảm 25%% sức chống cự)!", dispelled, strippedShield, firstTarget.Name),
				})
			}

			// Âm Đạo / Vận Đạo: Vận Mệnh Ca (SONG_OF_DESTINY) - Phượng Cửu Ca
			if hasMechanic(a.Mechanics, "ON_ACTION", "SONG_OF_DESTINY") {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				team := state.Allies
				opponents := state.Enemies
				if !isAlly {
					team = state.Enemies
					opponents = state.Allies
				}
				for _, ally := range team {
					if ally != nil && ally.HP > 0 {
						refreshCombatEffect(ally, Effect{ID: statusfx.StatusSongOfDestiny, Type: "buff", Value: 40, Duration: 1})
						manaGain := ally.MaxMana * 0.20
						ally.Mana = math.Min(ally.MaxMana, ally.Mana+manaGain)
					}
				}
				for _, opp := range opponents {
					if opp != nil && opp.HP > 0 {
						refreshCombatEffect(opp, Effect{ID: statusfx.StatusDestinySongWeak, Type: "debuff", Value: 30, Duration: 1})
					}
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SONG_OF_DESTINY",
					Detail:    fmt.Sprintf("[Âm Vận Đạo - Vận Mệnh Ca] Khúc ca vận mệnh chấn động thương khung! Tăng 40%% sát thương và hồi 20%% Chân Nguyên cho toàn đội, giảm 30%% sát thương của toàn bộ địch thủ!"),
				})
			}

			// Phong Đạo: Tống Hữu Phong (FAREWELL_FRIEND_WIND) - Vũ Dung
			if (hasMechanic(a.Mechanics, "ON_ACTION", "FAREWELL_FRIEND_WIND") || hasMechanic(a.Mechanics, "ON_HIT", "FAREWELL_FRIEND_WIND")) && firstTarget != nil {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusFarewellCurse, Value: 10, Duration: 4, Type: "debuff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FAREWELL_FRIEND_WIND",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Phong Đạo - Tống Hữu Phong] Vũ Dung vẫy tay áo tiễn bạn! Gió lốc xuyên thấu phòng ngự tăng 50%% sát thương và ám muội nguyền rủa Tống Hữu Chi Phong lên %s trong 4 hiệp!", firstTarget.Name),
				})
			}

			// Phong Đạo: Vô Hạn Phong (INFINITE_WIND) - Vũ Dung
			if hasMechanic(a.Mechanics, "ON_ACTION", "INFINITE_WIND") || hasMechanic(a.Mechanics, "ON_CAST", "INFINITE_WIND") {
				a.Damage *= 1.75
				operationDamageScale *= 1.75
				actor.ActiveEffects = append(actor.ActiveEffects,
					Effect{ID: statusfx.StatusHaste, Value: 60, Duration: 3, Type: "buff"},
					Effect{ID: statusfx.StatusEvasion, Value: 35, Duration: 3, Type: "buff"},
				)
				actor.EvasionChance = math.Max(actor.EvasionChance, 0.35)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "INFINITE_WIND",
					Detail:    fmt.Sprintf("[Phong Đạo - Vô Hạn Phong] Gió vô cùng vô tận bùng nổ! Tăng 75%% sát thương, +60%% Thần Tốc và +35%% Né Tránh cho %s trong 3 hiệp!", actor.Name),
				})
			}

			// Huyết Đạo: Huyết Ma Giải Thể (BLOOD_DEMON_DISINTEGRATION)
			if hasMechanic(a.Mechanics, "ON_ACTION", "BLOOD_DEMON_DISINTEGRATION") {
				hpCost := actor.HP * 0.20
				if hpCost < 10 {
					hpCost = 10
				}
				actor.HP -= hpCost
				if actor.HP < 1 {
					actor.HP = 1
				}
				a.Damage *= 2.20
				operationDamageScale *= 2.20
				a.LifeStealPct = 0.50
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BLOOD_DEMON_DISINTEGRATION",
					Damage:    hpCost,
					Detail:    fmt.Sprintf("[Huyết Đạo - Huyết Ma Giải Thể] %s hy sinh %.0f HP giải thể ma huyết, tăng bộc phát 120%% sát thương và nhận 50%% Hút Máu!", actor.Name, hpCost),
				})
			}

			// Hồn Đạo: Diêm Đế / Quỷ Quan Y (YAMA_EMPEROR) - Phương Nguyên
			if hasMechanic(a.Mechanics, "ON_ACTION", "YAMA_EMPEROR") || hasMechanic(a.Mechanics, "ON_CAST", "YAMA_EMPEROR") {
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusYamaEmperor, Duration: 4, Type: "buff",
				})
				actor.BaseDef += 50
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "YAMA_EMPEROR",
					Detail:    fmt.Sprintf("[Hồn Đạo - Diêm Đế] %s khoác lên mình Quỷ Quan Y giáng thế hóa thân Diêm La Đại Đế! +50%% Phòng Ngự và kích hoạt Diêm La Tử cắn nuốt hồn địch mỗi hiệp!", actor.Name),
				})
			}

			// Khí Đạo: Vô Lượng Khí Hải (IMMEASURABLE_QI_SEA) - Khí Hải Lão Tổ
			if hasMechanic(a.Mechanics, "ON_ACTION", "IMMEASURABLE_QI_SEA") {
				a.Damage *= 1.70
				operationDamageScale *= 1.70
				opponents := state.Enemies
				if !isAlly {
					opponents = state.Allies
				}
				for _, opp := range opponents {
					if opp != nil && opp.HP > 0 {
						opp.DamageMultiplier = math.Max(0.20, opp.DamageMultiplier-0.30)
						opp.ActiveEffects = append(opp.ActiveEffects, Effect{
							ID: statusfx.StatusSlow1, Value: 30, Duration: 3, Type: "debuff",
						})
					}
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "IMMEASURABLE_QI_SEA",
					Detail:    fmt.Sprintf("[Khí Đạo - Vô Lượng Khí Hải] Khí Hải Lão Tổ thôi động khí triều mênh mông như biển vô lượng! Tăng 70%% sát thương toàn diện, giảm 30%% Tấn Công và làm chậm 30%% toàn bộ quân địch!"),
				})
			}

			// Khí Đạo: Tam Khí Quy Lai (THREE_QI_RETURN) - Nguyên Thủy Tiên Tôn
			if hasMechanic(a.Mechanics, "ON_ACTION", "THREE_QI_RETURN") || hasMechanic(a.Mechanics, "ON_CAST", "THREE_QI_RETURN") {
				a.Damage *= 1.85
				operationDamageScale *= 1.85
				actor.IgnoreDefensePct = math.Max(actor.IgnoreDefensePct, 0.50)
				healAmt := actor.MaxHP * 0.30
				actor.HP = math.Min(actor.MaxHP, actor.HP+healAmt)
				actor.Mana = actor.MaxMana
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "THREE_QI_RETURN",
					Detail:    fmt.Sprintf("[Khí Đạo - Tam Khí Quy Lai] Thiên Địa Nhân tam khí quy nhất! %s hồi phục %.0f HP, đầy Chân Nguyên, tăng 85%% sát thương và bỏ qua 50%% phòng ngự!", actor.Name, healAmt),
				})
			}

			// Tín Đạo: Tam Nhân Thành Hổ (THREE_MEN_MAKE_TIGER)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "THREE_MEN_MAKE_TIGER") || hasMechanic(a.Mechanics, "ON_HIT", "THREE_MEN_MAKE_TIGER")) && firstTarget != nil {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects,
					Effect{ID: statusfx.StatusFear, Duration: 3, Type: "debuff"},
					Effect{ID: statusfx.StatusWeak, Value: 25, Duration: 3, Type: "debuff"},
				)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "THREE_MEN_MAKE_TIGER",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Tín Đạo - Tam Nhân Thành Hổ] Ba người nói thành hổ, giả ngưng tụ thành thực! Hung hổ ảo ảnh vồ nát tâm trí %s, tăng 50%% sát thương, áp đặt Sợ Hãi và Suy Yếu!", firstTarget.Name),
				})
			}

			// Tín Đạo: Nhân Ngôn Khả Úy (WORDS_ARE_FRIGHTENING)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "WORDS_ARE_FRIGHTENING") || hasMechanic(a.Mechanics, "ON_HIT", "WORDS_ARE_FRIGHTENING")) && firstTarget != nil {
				debuffCount := 0
				for _, eff := range firstTarget.ActiveEffects {
					if eff.Type == "debuff" {
						debuffCount++
					}
				}
				scalingMult := 1.40 + (float64(debuffCount) * 0.25)
				a.Damage *= scalingMult
				operationDamageScale *= scalingMult
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusSilence, Duration: 2, Type: "debuff",
				})
				firstTarget.IsSilenced = true
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "WORDS_ARE_FRIGHTENING",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Tín Đạo - Nhân Ngôn Khả Úy] Lời đồn thế gian như dao nhọn xé lòng! Kích hoạt %d debuff tăng %.0f%% sát thương và Câm Lặng %s trong 2 hiệp!", debuffCount, (scalingMult-1.0)*100, firstTarget.Name),
				})
			}

			// Thiên Đạo / Thời Đạo: Mệnh Bại (FATE_VANQUISHED) - Hồng Liên Ma Tôn
			if (hasMechanic(a.Mechanics, "ON_ACTION", "FATE_VANQUISHED") || hasMechanic(a.Mechanics, "ON_HIT", "FATE_VANQUISHED")) && firstTarget != nil {
				a.Damage *= 2.20
				operationDamageScale *= 2.20
				firstTarget.IsInvincible = false
				firstTarget.IsLastStand = false
				firstTarget.DamageTakenReduction = 0
				firstTarget.VirtualShield = 0
				var filteredEffects []Effect
				for _, eff := range firstTarget.ActiveEffects {
					if !strings.EqualFold(eff.ID, statusfx.StatusInvincible) &&
						!strings.EqualFold(eff.ID, statusfx.StatusLastStand) &&
						!strings.EqualFold(eff.ID, "invulnerable") &&
						!strings.EqualFold(eff.ID, "bat_tu") &&
						!strings.EqualFold(eff.ID, "bat_hoai") {
						filteredEffects = append(filteredEffects, eff)
					}
				}
				firstTarget.ActiveEffects = filteredEffects
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FATE_VANQUISHED",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Thiên Đạo - Mệnh Bại] Hồng Liên xé rách số mệnh! Phá vỡ hoàn toàn Bất Tử, Bất Hoại và Giáp ảo, giáng đòn chí mạng x2.2 sát thương lên %s!", firstTarget.Name),
				})
			}

			// Thiên Đạo: Lưới Trời Tuy Thưa (HEAVEN_NET_RESTRICTION) - Action
			if (hasMechanic(a.Mechanics, "ON_ACTION", "HEAVEN_NET_RESTRICTION") || hasMechanic(a.Mechanics, "ON_HIT", "HEAVEN_NET_RESTRICTION")) && firstTarget != nil {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				firstTarget.EvasionChance = 0
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusSlow1, Value: 40, Duration: 3, Type: "debuff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "HEAVEN_NET_RESTRICTION",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Thiên Đạo - Lưới Trời Tuy Thưa] Thiên võng giăng kín tứ phương! Tăng 50%% sát thương, tước đoạt toàn bộ Né Tránh và làm chậm 40%% của %s!", firstTarget.Name),
				})
			}

			// Trụ Đạo: Vị Lai Thân (FUTURE_SELF) - Phương Nguyên
			if hasMechanic(a.Mechanics, "ON_ACTION", "FUTURE_SELF") || hasMechanic(a.Mechanics, "ON_CAST", "FUTURE_SELF") {
				a.Damage *= 1.80
				operationDamageScale *= 1.80
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusFutureSelf, Value: 80, Duration: 3, Type: "buff",
				})
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusHaste, Value: 50, Duration: 3, Type: "buff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FUTURE_SELF",
					Detail:    fmt.Sprintf("[Trụ Đạo - Vị Lai Thân] Mượn sức mạnh từ tương lai đỉnh phong! %s triệu hoán Vị Lai Thân, tăng vọt 80%% Tấn Công và 50%% Tốc độ trong 3 hiệp!", actor.Name),
				})
			}

			// Trụ Đạo: Giang Sơn Như Cố (LANDSCAPE_AS_BEFORE) - Phương Nguyên
			if hasMechanic(a.Mechanics, "ON_ACTION", "LANDSCAPE_AS_BEFORE") || hasMechanic(a.Mechanics, "ON_CAST", "LANDSCAPE_AS_BEFORE") {
				shieldGain := actor.MaxHP * 0.40
				actor.VirtualShield += shieldGain
				var cleanedEffects []Effect
				for _, eff := range actor.ActiveEffects {
					if eff.Type != "debuff" {
						cleanedEffects = append(cleanedEffects, eff)
					}
				}
				actor.ActiveEffects = cleanedEffects
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "LANDSCAPE_AS_BEFORE",
					Detail:    fmt.Sprintf("[Trụ Đạo - Giang Sơn Như Cố] Giang sơn như cũ, cảnh cũ người xưa! %s khôi phục %.0f Giáp và tẩy sạch toàn bộ hiệu ứng suy yếu!", actor.Name, shieldGain),
				})
			}

			// Thực Đạo: Cật Khuy Thị Phúc (EATING_LOSS_BLESSING) - Action
			if hasMechanic(a.Mechanics, "ON_ACTION", "EATING_LOSS_BLESSING") {
				stacks := 0
				if actor.CustomState != nil {
					stacks = actor.CustomState["eating_loss_stacks"]
				}
				mult := 1.30 + (float64(stacks) * 0.15)
				a.Damage *= mult
				operationDamageScale *= mult
				healGain := float64(stacks) * 0.05 * actor.MaxHP
				if healGain > 0 {
					actor.HP = math.Min(actor.MaxHP, actor.HP+healGain)
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "EATING_LOSS_BLESSING",
					Detail:    fmt.Sprintf("[Thực Đạo - Cật Khuy Thị Phúc] Chuyển hóa %d tầng thiệt thòi thành phúc báo! Tăng %.0f%% sát thương và hồi phục %.0f HP!", stacks, (mult-1.0)*100, healGain),
				})
			}

			// Luật Đạo: Vạn Giải (MYRIAD_DISMANTLE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "MYRIAD_DISMANTLE") || hasMechanic(a.Mechanics, "ON_HIT", "MYRIAD_DISMANTLE")) && firstTarget != nil {
				a.Damage *= 1.60
				operationDamageScale *= 1.60
				firstTarget.VirtualShield = 0
				var nonBuffs []Effect
				for _, eff := range firstTarget.ActiveEffects {
					if eff.Type != "buff" {
						nonBuffs = append(nonBuffs, eff)
					}
				}
				firstTarget.ActiveEffects = nonBuffs
				actor.IgnoreDefensePct = 1.0
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "MYRIAD_DISMANTLE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Luật Đạo - Vạn Giải] Vạn vật quy nhất, pháp tắc tiêu giải! Hủy toàn bộ Hộ Thuẫn, xóa sạch mọi Buff và đánh xuyên 100%% phòng ngự của %s!", firstTarget.Name),
				})
			}

			// Sát Chiêu Đồ Đằng: U Đô Lực Bưu (YOU_DU_STRENGTH_TIGER)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "YOU_DU_STRENGTH_TIGER") || hasMechanic(a.Mechanics, "ON_HIT", "YOU_DU_STRENGTH_TIGER")) && firstTarget != nil {
				a.Damage *= 1.85
				operationDamageScale *= 1.85
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects,
					Effect{ID: statusfx.StatusBleed, Value: 20, Duration: 3, Type: "debuff"},
					Effect{ID: statusfx.StatusStun, Duration: 1, Type: "debuff"},
				)
				firstTarget.IsStunned = true
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "YOU_DU_STRENGTH_TIGER",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Đồ Đằng - U Đô Lực Bưu] U Đô Lực Bưu mọc cánh gầm vang! Tăng 85%% sát thương dã man, xé toạc sinh mệnh gây Xuất Huyết nặng và Choáng %s trong 1 hiệp!", firstTarget.Name),
				})
			}

			// ── Đại Điển Sát Chiêu Tiên Đạo (Đợt 7 - Reverend Insanity Canonical Moves 2) ──

			// Kiếm Đạo: Kiếm Lãng Tam Điệp (SWORD_WAVE_TRIPLE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "SWORD_WAVE_TRIPLE") || hasMechanic(a.Mechanics, "ON_HIT", "SWORD_WAVE_TRIPLE")) && firstTarget != nil {
				a.Damage *= 1.80
				operationDamageScale *= 1.80
				a.MultiHit = 3
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusBleed, Value: 25, Duration: 3, Type: "debuff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SWORD_WAVE_TRIPLE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Kiếm Đạo - Kiếm Lãng Tam Điệp] Ba đợt sóng kiếm trào dâng liên hoàn chém tới! Tăng 80%% sát thương (3 hit liên kích) và gây Xuất Huyết lên %s!", firstTarget.Name),
				})
			}

			// Kiếm Đạo: Kiếm Ngân Tác Mệnh (SWORD_MARK_DEATH)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "SWORD_MARK_DEATH") || hasMechanic(a.Mechanics, "ON_HIT", "SWORD_MARK_DEATH")) && firstTarget != nil {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects,
					Effect{ID: statusfx.StatusSwordMark, Duration: 4, Type: "debuff"},
					Effect{ID: statusfx.StatusSwordMark, Duration: 4, Type: "debuff"},
				)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SWORD_MARK_DEATH",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Kiếm Đạo - Kiếm Ngân Tác Mệnh] Khắc sâu 2 đạo kiếm ngân vào hồn phách %s! Kiếm ngân ăn mòn sinh mệnh mỗi hiệp và trảm sát khi dưới 25%% HP!", firstTarget.Name),
				})
			}

			// Hồn Đạo: Hồn Bạo Đãng Hồn Sơn (SOUL_EXPLOSION_DANG_HUN)
			if hasMechanic(a.Mechanics, "ON_ACTION", "SOUL_EXPLOSION_DANG_HUN") {
				a.Damage *= 1.60
				operationDamageScale *= 1.60
				opponents := state.Enemies
				if !isAlly {
					opponents = state.Allies
				}
				for _, opp := range opponents {
					if opp != nil && opp.HP > 0 {
						opp.ActiveEffects = append(opp.ActiveEffects, Effect{
							ID: statusfx.StatusStun, Duration: 1, Type: "debuff",
						})
						opp.IsStunned = true
					}
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SOUL_EXPLOSION_DANG_HUN",
					Detail:    fmt.Sprintf("[Hồn Đạo - Hồn Bạo Đãng Hồn Sơn] Tiếng chấn động Đãng Hồn Sơn vang vọng thiên địa! Bộc phá sát thương Hồn và Choáng toàn bộ phe địch trong 1 hiệp!"),
				})
			}

			// Hồn Đạo: Hồn Thú Triệu Lai (SOUL_BEAST_SUMMON)
			if hasMechanic(a.Mechanics, "ON_ACTION", "SOUL_BEAST_SUMMON") || hasMechanic(a.Mechanics, "ON_CAST", "SOUL_BEAST_SUMMON") {
				shieldGain := actor.MaxHP * 0.35
				actor.VirtualShield += shieldGain
				actor.DamageMultiplier += 0.30
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "SOUL_BEAST_SUMMON",
					Detail:    fmt.Sprintf("[Hồn Đạo - Hồn Thú Triệu Lai] Triệu hoán Thái Cổ Hồn Thú nhập thể hộ vệ! Nhận %.0f Hộ Thuẫn Hồn Thú và tăng 30%% Sát Thương!", shieldGain),
				})
			}

			// Thâu Đạo: Thâu Sinh (STEAL_LIFE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "STEAL_LIFE") || hasMechanic(a.Mechanics, "ON_HIT", "STEAL_LIFE")) && firstTarget != nil {
				stolenHP := firstTarget.MaxHP * 0.15
				if stolenHP < 30 {
					stolenHP = 30
				}
				firstTarget.HP -= stolenHP
				if firstTarget.HP < 0 {
					firstTarget.HP = 0
				}
				actor.HP = math.Min(actor.MaxHP, actor.HP+stolenHP)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "STEAL_LIFE",
					TargetId:  firstTarget.ID,
					Damage:    stolenHP,
					Detail:    fmt.Sprintf("[Thâu Đạo - Thâu Sinh] Đạo Tặc thần thông cướp đoạt sinh mệnh thọ nguyên! Hút thẳng %.0f HP của %s chuyển hóa thành sinh lực bản thân!", stolenHP, firstTarget.Name),
				})
			}

			// Vũ Đạo: Hỗn Động Toản (CHAOS_CAVE_DRILL)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "CHAOS_CAVE_DRILL") || hasMechanic(a.Mechanics, "ON_HIT", "CHAOS_CAVE_DRILL")) && firstTarget != nil {
				a.Damage *= 1.80
				operationDamageScale *= 1.80
				firstTarget.VirtualShield = 0
				actor.IgnoreDefensePct = 1.0
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "CHAOS_CAVE_DRILL",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Vũ Đạo - Hỗn Động Toản] Mũi khoan không gian xoắn nát hư không! Hủy sạch Hộ Thuẫn, bỏ qua 100%% phòng ngự và tăng 80%% sát thương lên %s!", firstTarget.Name),
				})
			}

			// Vũ Đạo: Vô Gian Đạo (NO_EXTREME_SPACE)
			if hasMechanic(a.Mechanics, "ON_ACTION", "NO_EXTREME_SPACE") || hasMechanic(a.Mechanics, "ON_CAST", "NO_EXTREME_SPACE") {
				actor.EvasionChance = math.Min(0.85, actor.EvasionChance+0.50)
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusEvasion, Value: 50, Duration: 3, Type: "buff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "NO_EXTREME_SPACE",
					Detail:    fmt.Sprintf("[Vũ Đạo - Vô Gian Đạo] %s hòa tan thân thể vào tầng lớp không gian vô tận! Tăng 50%% Né Tránh trong 3 hiệp!", actor.Name),
				})
			}

			// Trụ Đạo: Quang Âm Phi Nhận (LIGHT_TIME_FLYING_BLADE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "LIGHT_TIME_FLYING_BLADE") || hasMechanic(a.Mechanics, "ON_HIT", "LIGHT_TIME_FLYING_BLADE")) && firstTarget != nil {
				a.Damage *= 1.65
				operationDamageScale *= 1.65
				drainedMana := firstTarget.Mana * 0.30
				firstTarget.Mana -= drainedMana
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusSlow1, Value: 50, Duration: 3, Type: "debuff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "LIGHT_TIME_FLYING_BLADE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Trụ Đạo - Quang Âm Phi Nhận] Lưỡi đao thời gian phóng ra từ Trường Hà! Tăng 65%% sát thương, tước %.0f Chân Nguyên và làm chậm 50%% của %s!", drainedMana, firstTarget.Name),
				})
			}

			// Nhân Đạo: Nhân Trung Hào Kiệt (HERO_AMONG_MEN)
			if hasMechanic(a.Mechanics, "ON_ACTION", "HERO_AMONG_MEN") {
				missingHPRatio := 1.0 - (actor.HP / actor.MaxHP)
				heroMult := 1.30 + (missingHPRatio * 1.50)
				a.Damage *= heroMult
				operationDamageScale *= heroMult
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "HERO_AMONG_MEN",
					Detail:    fmt.Sprintf("[Nhân Đạo - Nhân Trung Hào Kiệt] Hào kiệt nghịch cảnh chuyển thương thành dũng! Máu hao hụt %.0f%% bộc phát tăng %.0f%% sát thương!", missingHPRatio*100, (heroMult-1.0)*100),
				})
			}

			// Viêm Đạo: Phẫn Nộ Đích Tiểu Điểu (RAGING_ANGRY_BIRD)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "RAGING_ANGRY_BIRD") || hasMechanic(a.Mechanics, "ON_HIT", "RAGING_ANGRY_BIRD")) && firstTarget != nil {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusBurn, Value: 15, Duration: 3, Type: "debuff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "RAGING_ANGRY_BIRD",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Viêm Đạo - Phẫn Nộ Đích Tiểu Điểu] Hỏa điểu phẫn nộ phát nổ kinh hoàng! Tăng 50%% sát thương và thiêu đốt %s trong 3 hiệp!", firstTarget.Name),
				})
			}

			// Băng Tuyết Đạo: Cực Băng Trấn Tiên Quan (EXTREME_ICE_CRYSTAL_COFFIN)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "EXTREME_ICE_CRYSTAL_COFFIN") || hasMechanic(a.Mechanics, "ON_HIT", "EXTREME_ICE_CRYSTAL_COFFIN")) && firstTarget != nil {
				a.Damage *= 1.40
				operationDamageScale *= 1.40
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects,
					Effect{ID: statusfx.StatusFreeze, Duration: 2, Type: "debuff"},
					Effect{ID: statusfx.StatusSilence, Duration: 2, Type: "debuff"},
				)
				firstTarget.IsFrozen = true
				firstTarget.IsSilenced = true
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "EXTREME_ICE_CRYSTAL_COFFIN",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Băng Tuyết - Trấn Tiên Quan] Quan tài băng tinh trấn áp! Đóng băng và Câm Lặng hoàn toàn %s trong 2 hiệp!", firstTarget.Name),
				})
			}

			// Quang Đạo: Tam Thập Tam Thiên Quang (THIRTY_THREE_HEAVEN_LIGHT)
			if hasMechanic(a.Mechanics, "ON_ACTION", "THIRTY_THREE_HEAVEN_LIGHT") || hasMechanic(a.Mechanics, "ON_CAST", "THIRTY_THREE_HEAVEN_LIGHT") {
				a.Damage *= 1.45
				operationDamageScale *= 1.45
				team := state.Allies
				opponents := state.Enemies
				if !isAlly {
					team = state.Enemies
					opponents = state.Allies
				}
				for _, ally := range team {
					if ally != nil && ally.HP > 0 {
						var clean []Effect
						for _, eff := range ally.ActiveEffects {
							if eff.Type != "debuff" {
								clean = append(clean, eff)
							}
						}
						ally.ActiveEffects = clean
					}
				}
				for _, opp := range opponents {
					if opp != nil && opp.HP > 0 {
						opp.ActiveEffects = append(opp.ActiveEffects, Effect{
							ID: statusfx.StatusBlind, Duration: 3, Type: "debuff",
						})
					}
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "THIRTY_THREE_HEAVEN_LIGHT",
					Detail:    fmt.Sprintf("[Quang Đạo - Tam Thập Tam Thiên Quang] 33 tầng thiên quang rọi sáng! Thanh tẩy toàn bộ phe ta và làm Mù Lòa toàn bộ quân địch!"),
				})
			}

			// Mộc Đạo: Khổn Tiên Đằng (IMMORTAL_BINDING_VINE)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "IMMORTAL_BINDING_VINE") || hasMechanic(a.Mechanics, "ON_HIT", "IMMORTAL_BINDING_VINE")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusBind, Duration: 2, Type: "debuff",
				})
				drainedMana := firstTarget.Mana * 0.20
				firstTarget.Mana -= drainedMana
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "IMMORTAL_BINDING_VINE",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Mộc Đạo - Khổn Tiên Đằng] Dây leo tiên đạo trói chặt %s trong 2 hiệp và hút %.0f Chân Nguyên!", firstTarget.Name, drainedMana),
				})
			}

			// Huyết Đạo: Huyết Thần Tử (BLOOD_GOD_CHILD)
			if hasMechanic(a.Mechanics, "ON_ACTION", "BLOOD_GOD_CHILD") || hasMechanic(a.Mechanics, "ON_CAST", "BLOOD_GOD_CHILD") {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				a.LifeStealPct = 0.40
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "BLOOD_GOD_CHILD",
					Detail:    fmt.Sprintf("[Huyết Đạo - Huyết Thần Tử] Huyết Thần Tử phân thân xuất thế trợ chiến! Tăng 50%% sát thương và nhận 40%% Hút Máu!"),
				})
			}

			// Sát Chiêu Đồ Đằng: Côn Bằng (TOTEM_KUN_PENG)
			if hasMechanic(a.Mechanics, "ON_ACTION", "TOTEM_KUN_PENG") {
				a.Damage *= 1.75
				operationDamageScale *= 1.75
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusHaste, Value: 50, Duration: 3, Type: "buff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "TOTEM_KUN_PENG",
					Detail:    fmt.Sprintf("[Đồ Đằng - Côn Bằng] Côn Bằng tung cánh bạt sơn cửu thiên! Tăng 75%% sát thương cuồng phong và nhận 50%% Thần Tốc!"),
				})
			}

			// Luyện Đạo: Luyện Kỷ (REFINE_SELF)
			if hasMechanic(a.Mechanics, "ON_ACTION", "REFINE_SELF") || hasMechanic(a.Mechanics, "ON_CAST", "REFINE_SELF") {
				actor.BaseDmg += actor.BaseDmg * 0.20
				actor.BaseDef += actor.BaseDef * 0.20
				actor.ActiveEffects = append(actor.ActiveEffects, Effect{
					ID: statusfx.StatusRefineSelf, Duration: 99, Type: "buff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "REFINE_SELF",
					Detail:    fmt.Sprintf("[Luyện Đạo - Luyện Kỷ] Đỉnh cao Luyện Đạo lấy thân làm phôi! Tăng vĩnh viễn 20%% Công và 20%% Thủ cho %s!", actor.Name),
				})
			}

			// Trí Đạo: Vạn Tinh Phi Huỳnh (MYRIAD_STAR_FIREFLY)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "MYRIAD_STAR_FIREFLY") || hasMechanic(a.Mechanics, "ON_HIT", "MYRIAD_STAR_FIREFLY")) && firstTarget != nil {
				a.Damage *= 1.60
				operationDamageScale *= 1.60
				drainedMana := firstTarget.Mana * 0.40
				firstTarget.Mana -= drainedMana
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "MYRIAD_STAR_FIREFLY",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Trí Đạo - Vạn Tinh Phi Huỳnh] Đom đóm ánh sao bùng cháy thiêu rụi ý niệm! Tăng 60%% sát thương và thiêu rụi %.0f Chân Nguyên của %s!", drainedMana, firstTarget.Name),
				})
			}

			// Cấm Đạo: Ngũ Cấm Huyền Quang Khí (FIVE_RESTRICTION_DEEP_LIGHT)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "FIVE_RESTRICTION_DEEP_LIGHT") || hasMechanic(a.Mechanics, "ON_HIT", "FIVE_RESTRICTION_DEEP_LIGHT")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusGuSeal, Duration: 3, Type: "debuff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FIVE_RESTRICTION_DEEP_LIGHT",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Cấm Đạo - Ngũ Cấm Huyền Quang Khí] Luồng sáng năm màu phong tỏa ngũ hành! Phong ấn Cổ trùng (Phong Cổ) của %s trong 3 hiệp!", firstTarget.Name),
				})
			}

			// Mị Đạo: Tử Quỷ Khoái Lai (CHARM_GHOST_COME)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "CHARM_GHOST_COME") || hasMechanic(a.Mechanics, "ON_HIT", "CHARM_GHOST_COME")) && firstTarget != nil {
				firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
					ID: statusfx.StatusCharm, Duration: 2, Type: "debuff",
				})
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "CHARM_GHOST_COME",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Mị Đạo - Tử Quỷ Khoái Lai] Tiếng gọi ma mị quyến rũ mê hồn! Áp đặt Mê Hoặc khiến %s thần hồn điên đảo trong 2 hiệp!", firstTarget.Name),
				})
			}

			// Thực Đạo: Thôn Thiên Địa (DEVOUR_HEAVEN_EARTH)
			if (hasMechanic(a.Mechanics, "ON_ACTION", "DEVOUR_HEAVEN_EARTH") || hasMechanic(a.Mechanics, "ON_HIT", "DEVOUR_HEAVEN_EARTH")) && firstTarget != nil {
				a.Damage *= 1.50
				operationDamageScale *= 1.50
				devouredShield := firstTarget.VirtualShield * 0.50
				firstTarget.VirtualShield -= devouredShield
				actor.HP = math.Min(actor.MaxHP, actor.HP+devouredShield)
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "DEVOUR_HEAVEN_EARTH",
					TargetId:  firstTarget.ID,
					Detail:    fmt.Sprintf("[Thực Đạo - Thôn Thiên Địa] Cắn nuốt thiên địa càn khôn! Nuốt chửng %.0f Giáp của %s chuyển hóa thành HP hồi phục bản thân!", devouredShield, firstTarget.Name),
				})
			}

			// Ngũ Giới Giới Vực Bích: Phản phệ khi ra chiêu nếu mang StatusFiveRegionalSuppression
			if hasStatus(actor, statusfx.StatusFiveRegionalSuppression) {
				selfHarm := actor.MaxHP * 0.05
				if selfHarm < 10 {
					selfHarm = 10
				}
				actor.HP -= selfHarm
				if actor.HP < 1 {
					actor.HP = 1
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FIVE_REGIONAL_SUPPRESSION",
					Detail:    fmt.Sprintf("[Ngũ Giới Áp Chế] Giới Vực Bích phản phệ kịch liệt khi thôi động sát chiêu, %s tự tổn %.0f HP!", actor.Name, selfHarm),
				})
			}

			// A normal Gu skill is paid, checked, and counted once by this parent
			// action. Its Block v2 operations retain their own target and conditions
			// and are each applied by the simulator exactly once here.
			if a.ActionType == "GU_SKILL" && len(a.OperationEffects) > 0 {
				for _, effect := range a.OperationEffects {
					if effect.Action == nil {
						continue
					}
					child := *effect.Action
					child.IsEffectOperation = true
					child.OperationConditions = append([]gameplayeffects.Condition(nil), effect.Compiled.Conditions...)
					child.Rank = effect.SourceRank
					child.SourceID = a.SourceID
					if child.Damage > 0 {
						child.Damage *= operationDamageScale
					} else {
						child.DamageMult *= operationDamageScale
					}
					child.DamageHPPercent *= operationDamageScale
					child.ShieldStripPercent *= operationDamageScale
					child.PhantomDmg *= operationDamageScale
					if child.Damage > 0 || child.DamageHPPercent > 0 || child.ShieldStripPercent > 0 || child.PhantomDmg > 0 {
						for _, mechanic := range a.Mechanics {
							if mechanic.Trigger == "ON_HIT" && !hasMechanic(child.Mechanics, mechanic.Trigger, mechanic.ActionCode) {
								child.Mechanics = append(child.Mechanics, mechanic)
							}
						}
					}
					out := runOne(actor, &child, isAlly, enemyTargets)
					res.executed = res.executed || out.executed
					res.hasDamage = res.hasDamage || out.hasDamage
					if out.abort {
						res.abort = true
						return res
					}
				}
				return res
			}

			// ── Apply damage & compound effects to all resolved targets ──
			totalDamageDealt := 0.0
			hasDamage := a.ActionType == "BASIC_ATTACK" || a.Damage > 0 ||
				a.DamageHPPercent > 0 || a.ShieldStripPercent > 0 || a.PhantomDmg > 0
			res.hasDamage = hasDamage

			if hasDamage {
				dmgMultiplier := 1.0
				for _, eff := range actor.ActiveEffects {
					if eff.ID == "sword_dragon_form" && eff.Duration >= 0 {
						dmgMultiplier *= 1.50
					}
					if eff.ID == "borrow_past_power" && eff.Duration >= 0 {
						dmgMultiplier *= 1.40
					}
				}

				// Apply weak debuff (reduces damage dealt) and atk_up buff
				dmgMultiplier *= actor.DamageMultiplier

				// COMBAT_PASSIVE is the canonical trigger; bare PASSIVE is
				// accepted for rows authored before the rename (same
				// ON_EVENT timing, unrelated to the passive section).
				if hasMechanic(actor.Mechanics, "PASSIVE", "DOUBLE_DAMAGE_DEALT") ||
					hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "DOUBLE_DAMAGE_DEALT") {
					dmgMultiplier *= 2.0
				}

				// Cộng dồn với skill mult (Gu damage_mult) thay vì nhân; xem skillMult bên dưới.
				pendingDamage := getPendingDamageMult(actor)
				if pendingDamage > 1.0 {
					actor.PendingDamageBuffs = nil // consume all pending damage buffs
				} else {
					pendingDamage = 1.0
				}

				shieldBypass := hasMechanic(a.Mechanics, "ON_HIT", "BYPASS_SHIELD") ||
					hasMechanic(a.Mechanics, "ON_ACTION", "SOUL_SHOCK") ||
					hasMechanic(a.Mechanics, "ON_HIT", "SOUL_SHOCK") ||
					hasMechanic(a.Mechanics, "ON_ACTION", "SOUND_RESONANCE") ||
					hasMechanic(a.Mechanics, "ON_HIT", "SOUND_RESONANCE") ||
					hasMechanic(a.Mechanics, "ON_ACTION", "FAREWELL_FRIEND_WIND") ||
					hasMechanic(a.Mechanics, "ON_HIT", "FAREWELL_FRIEND_WIND") ||
					hasMechanic(a.Mechanics, "ON_ACTION", "CHAOS_CAVE_DRILL") ||
					hasMechanic(a.Mechanics, "ON_HIT", "CHAOS_CAVE_DRILL") ||
					hasMechanic(a.Mechanics, "ON_ACTION", "STEAL_LIFE") ||
					hasMechanic(a.Mechanics, "ON_ACTION", "SOUL_EXPLOSION_DANG_HUN") ||
					(hasMechanic(actor.Mechanics, "ON_BATTLE_START", "SAT_KHI_GUARANTEE") && actor.CustomState != nil && actor.CustomState["sat_khi_level"] > 10)
				hitCount := a.MultiHit
				if hitCount < 1 {
					hitCount = 1
				}

				// Targets whose strike was negated by FIRST_TURN_EVADE carry
				// no percent riders (damage_hp_percent/shield strip below).
				evadedThisStrike := make(map[*CombatantAuto]bool)
				for _, target := range resolved {
					totalDmg := 0.0
					hadEvaded := target.CustomState != nil && target.CustomState["first_turn_evade_used"] == turn

					// Evasion: dodge chance from the target's EVASION status.
					satKhiGuaranteed := hasMechanic(actor.Mechanics, "ON_BATTLE_START", "SAT_KHI_GUARANTEE") && actor.CustomState != nil && actor.CustomState["sat_khi_level"] > 10
					sureHit := satKhiGuaranteed || isGuaranteedHit(actor, a)
					if missCh := target.EvasionChance; missCh > 0 && rng.Float64() < missCh && !sureHit {
						turnLogs = append(turnLogs, CombatLog{
							Turn:       turn,
							ActorId:    actor.ID,
							ActorName:  actor.Name,
							Action:     "EVADE_MISS",
							TargetId:   target.ID,
							SourceGuID: a.SourceID,
							Detail:     fmt.Sprintf("%s né tránh đòn [%s] của %s!", target.Name, a.Name, actor.Name),
							Breakdown: &DamageBreakdown{
								BaseDamage:     a.Damage,
								AttackerAttack: actor.StatProvenance["attack"],
								Reason:         "Né tránh thành công (Evasion)",
							},
						})
						continue
					}

					// Blind: chance to miss the target entirely
					if actor.IsBlinded && actor.AccuracyPenalty > 0 && !sureHit {
						if rng.Float64() < actor.AccuracyPenalty {
							turnLogs = append(turnLogs, CombatLog{
								Turn:       turn,
								ActorId:    actor.ID,
								ActorName:  actor.Name,
								Action:     "BLIND_MISS",
								TargetId:   target.ID,
								SourceGuID: a.SourceID,
								Detail:     fmt.Sprintf("%s bị mù, đòn tấn công [%s] đánh trượt %s!", actor.Name, a.Name, target.Name),
								Breakdown: &DamageBreakdown{
									BaseDamage:     a.Damage,
									AttackerAttack: actor.StatProvenance["attack"],
									Reason:         "Đánh trượt do trạng thái Mù (Blind)",
								},
							})
							continue
						}
					}

					targetDef := target.BaseDef * target.DefenseMultiplier
					// ON_HIT/BYPASS_SHIELD means one thing only: the hit ignores
					// VirtualShield (resolved in applyDamageToTarget). Defense
					// shredding belongs to ignore_defense_* fields, not here.
					// Apply ignore_defense from attacker
					if actor.IgnoreDefenseFlat > 0 {
						targetDef = math.Max(0, targetDef-actor.IgnoreDefenseFlat)
					}
					if actor.IgnoreDefensePct > 0 {
						targetDef = math.Max(0, targetDef*(1.0-actor.IgnoreDefensePct))
					}

					initialShield := target.VirtualShield
					initialHP := target.HP
					isCritHit := false
					hitPath := daoPathForAction(a.Path)
					natResVal, targetResonance := targetDaoFactors(target, hitPath)

					skillMult := a.DamageMult
					if skillMult <= 0 {
						skillMult = 1.0
					}
					// Cộng dồn Gu damage_mult với buff chờ: 1 + Σ(m-1).
					if pendingDamage > 1.0 {
						skillMult += pendingDamage - 1.0
					}

					totalHitDmg := 0.0
					for h := 0; h < hitCount; h++ {
						hitDmg := a.Damage * skillMult * dmgMultiplier * daoMult * conflictPenaltyMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path)) * targetResonance
						if h > 0 {
							hitDmg *= 0.7
						}
						// Critical hit check
						if actor.CritRateBonus > 0 && rng.Float64() < actor.CritRateBonus {
							hitDmg *= 1.5
							isCritHit = true
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "CRIT",
								TargetId:  target.ID,
								Detail:    fmt.Sprintf("%s chí mạng! Đòn %s gây 150%% sát thương!", actor.Name, a.Name),
							})
						}
						netDmg := hitDmg - targetDef
						netDmg *= (1.0 - natResVal)
						netDmg *= (1.0 - pathBonus(target.PathResistance, a.Path))
						netDmg = ResolveBaseDamage(netDmg, 0)
						totalHitDmg += netDmg
						totalDmg += applyDamageToTarget(state, actor, target, netDmg, &turnLogs, turn, shieldBypass, sureHit, !isAlly, rng)
					}
					if !hadEvaded && target.CustomState != nil && target.CustomState["first_turn_evade_used"] == turn {
						evadedThisStrike[target] = true
					}
					totalDamageDealt += totalDmg

					shieldAbsorbed := math.Max(0, initialShield-target.VirtualShield)
					hpLost := math.Max(0, initialHP-target.HP)
					if hpLost > 0 && natResVal > 0 {
						applyForeignDaoWound(target)
					}
					critMultVal := 1.0
					if isCritHit {
						critMultVal = 1.5
					}

					rawDmg := a.Damage * skillMult * dmgMultiplier * daoMult * conflictPenaltyMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path)) * targetResonance * critMultVal

					natResFinal := natResVal

					dmgReason := a.Reason
					if conflictMult < 1.0 {
						conflictMsg := fmt.Sprintf("Xung đột Đạo Ngân (uy lực x%.2f)", conflictMult)
						if dmgReason != "" {
							dmgReason += " | " + conflictMsg
						} else {
							dmgReason = conflictMsg
						}
					}
					breakdown := &DamageBreakdown{
						BaseDamage:         a.Damage,
						SkillMult:          skillMult,
						DaoMult:            daoMult,
						TargetDaoResonance: targetResonance,
						DaoPath:            hitPath,
						BuffMult:           dmgMultiplier,
						CritMult:           critMultVal,
						PathBonus:          pathBonus(actor.PathDamageBonus, a.Path),
						TargetDef:          targetDef,
						NaturalResist:      natResFinal,
						PathResist:         pathBonus(target.PathResistance, a.Path),
						DamageReduction:    target.DamageTakenReduction,
						ShieldAbsorbed:     shieldAbsorbed,
						HPDamage:           hpLost,
						FinalDamage:        totalHitDmg,
						RawDamage:          rawDmg,
						HitCount:           hitCount,
						Reason:             dmgReason,
						AttackerAttack:     actor.StatProvenance["attack"],
						DefenderDefense:    target.StatProvenance["defense"],
					}

					if totalHitDmg > 0 {
						var damageDetail string
						if shieldAbsorbed > 0 && hpLost > 0 {
							damageDetail = fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f sát thương (phá vỡ %.0f khiên, trừ %.0f HP, còn %.0f HP)", actor.Name, a.Name, target.Name, totalHitDmg, shieldAbsorbed, hpLost, target.HP)
						} else if shieldAbsorbed > 0 && hpLost == 0 {
							damageDetail = fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f sát thương (bị Khiên Ảo cản %.0f, còn %.0f khiên, %.0f HP)", actor.Name, a.Name, target.Name, totalHitDmg, shieldAbsorbed, target.VirtualShield, target.HP)
						} else {
							damageDetail = fmt.Sprintf("%s dùng [%s] lên %s, gây %.0f sát thương (còn %.0f HP)", actor.Name, a.Name, target.Name, totalHitDmg, target.HP)
						}

						landed := !evadedThisStrike[target] && (hpLost > 0 || shieldAbsorbed > 0)
						turnLogs = append(turnLogs, CombatLog{
							Turn:       turn,
							ActorId:    actor.ID,
							ActorName:  actor.Name,
							Action:     a.ActionType,
							TargetId:   target.ID,
							Damage:     totalHitDmg,
							IsCrit:     isCritHit,
							SourceGuID: a.SourceID,
							HitLanded:  landed,
							Detail:     damageDetail,
							Breakdown:  breakdown,
						})
					} else {
						if target.VirtualShield > 0 {
							breakdown.Reason = "Toàn bộ sát thương bị Khiên Ảo hấp thụ"
							turnLogs = append(turnLogs, CombatLog{
								Turn:       turn,
								ActorId:    actor.ID,
								ActorName:  actor.Name,
								Action:     a.ActionType,
								TargetId:   target.ID,
								Damage:     0,
								SourceGuID: a.SourceID,
								Detail:     fmt.Sprintf("%s dùng [%s] lên %s, toàn bộ sát thương bị Khiên Ảo hấp thụ (còn %.0f khiên, %.0f HP)", actor.Name, a.Name, target.Name, target.VirtualShield, target.HP),
								Breakdown:  breakdown,
							})
						} else if target.HP <= 0 {
							breakdown.Reason = "Mục tiêu đã gục ngã từ trước"
							turnLogs = append(turnLogs, CombatLog{
								Turn:       turn,
								ActorId:    actor.ID,
								ActorName:  actor.Name,
								Action:     a.ActionType,
								TargetId:   target.ID,
								Damage:     0,
								SourceGuID: a.SourceID,
								Detail:     fmt.Sprintf("%s dùng [%s] lên %s (mục tiêu đã gục ngã từ trước)!", actor.Name, a.Name, target.Name),
								Breakdown:  breakdown,
							})
						} else {
							breakdown.Reason = "Đối phương phòng thủ kiên cố hoặc miễn nhiễm toàn bộ sát thương"
							turnLogs = append(turnLogs, CombatLog{
								Turn:       turn,
								ActorId:    actor.ID,
								ActorName:  actor.Name,
								Action:     a.ActionType,
								TargetId:   target.ID,
								Damage:     0,
								SourceGuID: a.SourceID,
								Detail:     fmt.Sprintf("%s dùng [%s] lên %s, gây 0 sát thương do đối phương phòng thủ/miễn nhiễm (còn %.0f HP)!", actor.Name, a.Name, target.Name, target.HP),
								Breakdown:  breakdown,
							})
						}
					}

					if a.PhantomDmg > 0 {
						phantomPath := daoPathForAction(a.Path)
						phantomResist, phantomResonance := targetDaoFactors(target, phantomPath)
						phantomDef := target.BaseDef
						phantomNet := (a.PhantomDmg * dmgMultiplier * phantomResonance) - phantomDef
						phantomNet *= (1 - phantomResist)
						phantomNet *= (1.0 - pathBonus(target.PathResistance, a.Path))
						phantomNet = math.Max(1, phantomNet)
						phantomDealt := applyDamageToTarget(state, actor, target, phantomNet, &turnLogs, turn, shieldBypass, sureHit, !isAlly, rng)
						if phantomDealt > 0 && phantomResist > 0 {
							applyForeignDaoWound(target)
						}
						totalDamageDealt += phantomDealt
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "PHANTOM_ATTACK",
							TargetId:  target.ID,
							Damage:    phantomDealt,
							Detail:    fmt.Sprintf("Hư ảnh của %s tấn công thêm %.0f sát thương lên %s!", actor.Name, phantomDealt, target.Name),
						})
					}

					// ── Freeze break: if target is frozen and takes damage, 50% chance to break ──
					if target.IsFrozen && totalDmg > 0 {
						if rng.Float64() < 0.50 {
							var unfrozenKept []Effect
							for _, eff := range target.ActiveEffects {
								if eff.ID == EffectFreeze {
									turnLogs = append(turnLogs, CombatLog{
										Turn:      turn,
										ActorId:   target.ID,
										ActorName: target.Name,
										Action:    "FREEZE_BREAK",
										Detail:    fmt.Sprintf("Sức mạnh từ đòn tấn công phá vỡ băng giá! %s thoát đóng băng!", target.Name),
									})
									continue
								}
								unfrozenKept = append(unfrozenKept, eff)
							}
							target.ActiveEffects = unfrozenKept
							target.IsFrozen = false
						}
					}
				}

				if shieldLog != nil {
					turnLogs = append(turnLogs, *shieldLog)
				}
				if healLog != nil {
					turnLogs = append(turnLogs, *healLog)
				}
				if manaLog != nil {
					turnLogs = append(turnLogs, *manaLog)
				}
				if cleanseLog != nil {
					turnLogs = append(turnLogs, *cleanseLog)
				}

				// ── Multi-hit log ──
				if hitCount > 1 && len(resolved) > 0 {
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "MULTI_HIT",
						TargetId:  resolved[0].ID,
						Detail:    fmt.Sprintf("%s trúng %d đòn liên hoàn!", resolved[0].Name, hitCount),
					})
				}

				if a.ThornsDmg > 0 && totalDamageDealt > 0 {
					thornsNet := math.Max(1, a.ThornsDmg-float64(actor.BaseDef))
					applyDamageToTarget(state, firstTarget, actor, thornsNet, &turnLogs, turn, false, false, false, rng)
					turnLogs = append(turnLogs, CombatLog{
						Turn:      turn,
						ActorId:   actor.ID,
						ActorName: actor.Name,
						Action:    "THORNS",
						TargetId:  firstTarget.ID,
						Damage:    thornsNet,
						Detail:    fmt.Sprintf("Phản thương từ %s gây %.0f sát thương lên %s!", firstTarget.Name, thornsNet, actor.Name),
					})
				}

				// ── Thorns buff reflect: defender reflects % of damage dealt ──
				for _, target := range resolved {
					if target.ThornsDamage > 0 && totalDamageDealt > 0 {
						thornsNet := ApplyThornsDamage(target, actor, float64(totalDamageDealt), turn, &turnLogs)
						_ = thornsNet
					}
				}

				if a.LifeStealPct > 0 && totalDamageDealt > 0 {
					healAmount := totalDamageDealt * a.LifeStealPct
					hpNeeded := actor.MaxHP - actor.HP
					if healAmount <= hpNeeded {
						actor.HP += healAmount
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "LIFESTEAL",
							Damage:    healAmount,
							Detail:    fmt.Sprintf("%s hút %.0f máu từ sát chiêu, hồi phục HP!", actor.Name, healAmount),
						})
					} else {
						actor.HP = actor.MaxHP
						excess := healAmount - hpNeeded
						actor.VirtualShield += excess
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "LIFESTEAL_SHIELD",
							Damage:    excess,
							Detail:    fmt.Sprintf("%s hút %.0f máu từ sát chiêu, HP đầy, chuyển %.0f thành khiên!", actor.Name, healAmount, excess),
						})
					}
				}

				actionRank := a.Rank
				if actionRank <= 0 {
					actionRank = actor.Rank
				}

				// ── Percent-based direct effects (damage_hp_percent, shield strip) ──
				// A strike negated by FIRST_TURN_EVADE carries no riders.
				if a.DamageHPPercent > 0 || a.ShieldStripPercent > 0 {
					for _, target := range resolved {
						if target == nil || target.HP <= 0 {
							continue
						}
						if evadedThisStrike[target] {
							continue
						}
						if a.ShieldStripPercent > 0 && target.VirtualShield > 0 {
							stripPct := a.ShieldStripPercent
							if target.Rank > actionRank {
								supp, _ := statusfx.TargetSuppressionFactor(actionRank, target.Rank)
								stripPct *= supp
							}
							strip := math.Min(target.VirtualShield, target.VirtualShield*stripPct)
							target.VirtualShield -= strip
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "SHIELD_STRIP",
								TargetId:  target.ID,
								Damage:    strip,
								Detail:    fmt.Sprintf("[%s] tước %.0f khiên của %s (còn %.0f khiên)!", a.Name, strip, target.Name, target.VirtualShield),
							})
						}
						if a.DamageHPPercent > 0 && isImmuneToDamage(target, satKhiPiercesImmunity(actor)) {
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   actor.ID,
								ActorName: actor.Name,
								Action:    "HP_PERCENT_ABSORBED",
								TargetId:  target.ID,
								Detail:    fmt.Sprintf("%s miễn nhiễm %s nhờ Bất Hoại!", target.Name, a.Name),
							})
						} else if a.DamageHPPercent > 0 {
							hpPct := a.DamageHPPercent
							if target.Rank > actionRank {
								supp, _ := statusfx.TargetSuppressionFactor(actionRank, target.Rank)
								hpPct *= supp
							}
							if hpPct > 0 {
								direct := target.MaxHP * hpPct
								target.HP -= direct
								floored := applyLastStandFloor(target, satKhiPiercesImmunity(actor))
								if target.HP < 0 {
									target.HP = 0
								}
								hpPctDetail := fmt.Sprintf("[%s] đánh thẳng vào bản ngã, %s mất %.0f HP trực tiếp (còn %.0f HP)!", a.Name, target.Name, direct, target.HP)
								if floored {
									hpPctDetail = fmt.Sprintf("[%s] đánh thẳng %.0f HP vào %s nhưng [Bất Tử] giữ lại 1 HP!", a.Name, direct, target.Name)
								}
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "HP_PERCENT",
									TargetId:  target.ID,
									Damage:    direct,
									Detail:    hpPctDetail,
								})
							} else {
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   actor.ID,
									ActorName: actor.Name,
									Action:    "STATUS_IMMUNE",
									TargetId:  target.ID,
									Detail:    fmt.Sprintf("%s cảnh giới vượt trội, hoàn toàn miễn nhiễm sát thương trực tiếp từ [%s]!", target.Name, a.Name),
								})
							}
						}
					}
				}
			} else {
				detailMsg := fmt.Sprintf("%s dùng [%s]", actor.Name, a.Name)
				hasBuff := false
				buffKey := a.SourceID
				if buffKey == "" {
					buffKey = a.Name
				}
				if a.DamageMult > 1.0 {
					if actor.PendingDamageBuffs == nil {
						actor.PendingDamageBuffs = make(map[string]float64)
					}
					actor.PendingDamageBuffs[buffKey] = a.DamageMult
					detailMsg = fmt.Sprintf("%s dùng [%s], tăng %.1fx sát thương cho đòn tấn công kế tiếp!", actor.Name, a.Name, a.DamageMult)
					hasBuff = true
				}
				if a.ShieldMult > 1.0 {
					if actor.PendingShieldBuffs == nil {
						actor.PendingShieldBuffs = make(map[string]float64)
					}
					actor.PendingShieldBuffs[buffKey] = a.ShieldMult
					if hasBuff {
						detailMsg += fmt.Sprintf(" (và tăng %.1fx khiên kế tiếp)", a.ShieldMult)
					} else {
						detailMsg = fmt.Sprintf("%s dùng [%s], tăng %.1fx khiên cho kỹ năng tạo khiên kế tiếp!", actor.Name, a.Name, a.ShieldMult)
					}
				}

				targetID := firstTarget.ID
				if strings.EqualFold(a.TargetType, "SELF") {
					targetID = actor.ID
				}
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    a.ActionType,
					TargetId:  targetID,
					Detail:    detailMsg,
				})

				if shieldLog != nil {
					turnLogs = append(turnLogs, *shieldLog)
				}
				if healLog != nil {
					turnLogs = append(turnLogs, *healLog)
				}
				if manaLog != nil {
					turnLogs = append(turnLogs, *manaLog)
				}
				if cleanseLog != nil {
					turnLogs = append(turnLogs, *cleanseLog)
				}
			}

			// ── Apply status effects + self-buffs (always, no damage required) ──
			// ProcChance gates the whole bonus package (status ops, self
			// evasion, target weak, Gu seal); base damage already landed.
			procOK := a.ProcChance <= 0 || rng.Float64() < a.ProcChance
			if !procOK {
				turnLogs = append(turnLogs, CombatLog{
					Turn:      turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "PROC_FAIL",
					Detail:    fmt.Sprintf("[%s] không kích hoạt hiệu ứng bổ sung!", a.Name),
				})
			} else {
				allOps := collectAllStatusOps(a)
				actionRank := a.Rank
				if actionRank <= 0 {
					actionRank = actor.Rank
				}
				if len(allOps) > 0 {
					for i := range allOps {
						allOps[i].SourcePath = daoPathForAction(a.Path)
					}
					applyStatusOps(actor, resolved, allOps, actionRank, turn, rng, &turnLogs)
				}
				if a.TargetWeakPct > 0 {
					hasFR := false
					for _, op := range allOps {
						if op.ID == statusfx.StatusForgetReality {
							hasFR = true
							break
						}
					}
					if !hasFR {
						turns := a.BonusTurns
						if turns <= 0 {
							turns = 2
						}
						applyStatusOps(actor, resolved, []statusfx.StatusOp{{ID: statusfx.StatusWeak, Value: a.TargetWeakPct * 100, Turns: turns}}, actionRank, turn, rng, &turnLogs)
					}
				}
				if a.DisableGuTurns > 0 {
					for _, target := range resolved {
						if target == nil || target.HP <= 0 {
							continue
						}
						turns := a.DisableGuTurns
						if target.Rank > actionRank {
							supp, penalty := statusfx.TargetSuppressionFactor(actionRank, target.Rank)
							if supp <= 0 {
								turnLogs = append(turnLogs, CombatLog{
									Turn:      turn,
									ActorId:   target.ID,
									ActorName: target.Name,
									Action:    "STATUS_IMMUNE",
									Detail:    fmt.Sprintf("%s cảnh giới vượt trội, hoàn toàn miễn nhiễm hiệu ứng phong ấn Cổ từ [%s]!", target.Name, a.Name),
								})
								continue
							}
							turns -= penalty
							if turns < 1 {
								turns = 1
							}
						}
						// Nghịch Chuyển Cổ: hostile seal aimed at the holder is reversed.
						if hasReverseCalamity(target) {
							turnLogs = append(turnLogs, CombatLog{
								Turn:      turn,
								ActorId:   target.ID,
								ActorName: target.Name,
								Action:    "REVERSE_NEGATE",
								Detail:    fmt.Sprintf("Nghịch Chuyển Cổ đảo nghịch phong ấn Cổ của [%s] về hư vô!", a.Name),
							})
							continue
						}
						sealGuTurns(target, turns)
						turnLogs = append(turnLogs, CombatLog{
							Turn:      turn,
							ActorId:   target.ID,
							ActorName: target.Name,
							Action:    "GU_SEAL",
							Detail:    fmt.Sprintf("%s bị phong ấn Cổ trong %d lượt, không thể dùng Cổ!", target.Name, turns),
						})
					}
				}
			}
			// Merge action logs into a single consolidated log entry for this Gu/Action
			// Fast path: when the action produced a single log it already is its own
			// consolidated entry (the merge would only re-assign the same value back),
			// so skip the slice copy + consolidation scan for the common case.
			if len(turnLogs)-actionStartIndex > 1 {
				logs := make([]CombatLog, len(turnLogs)-actionStartIndex)
				copy(logs, turnLogs[actionStartIndex:])

				var useLog *CombatLog
				var costLog *CombatLog
				var gainLog *CombatLog
				var resultLogs []CombatLog
				var preservedLogs []CombatLog
				var extraBreakdown *DamageBreakdown

				for i := range logs {
					log := &logs[i]
					if strings.Contains(log.Detail, " dùng [") && useLog == nil {
						useLog = log
					} else if log.Action == "MANA_COST" {
						costLog = log
					} else if log.Action == "PATH_MASTERY_GAIN" {
						gainLog = log
					} else if log.Action == "FORGET_WILL_INACTIVE" || log.Action == "FORGET_INTENT" || log.Action == "FORGET_INTENT_MISS" || log.Action == "FORGET_INTENT_BACKFIRE" || log.Action == "REVERSE_NEGATE" {
						// Proc outcomes remain separate so replay consumers can identify
						// whether the interrupt activated, missed, or was negated.
						preservedLogs = append(preservedLogs, *log)
					} else if log.Action == "SHIELD_ABSORB" || log.Action == "SHIELD_BREAK" || log.Action == "SHIELD_BYPASS_ATTACK" || log.Action == "PATH_MASTERY_CAP" {
						// Skip redundant damage outcome logs already detailed in useLog
						continue
					} else {
						if log.Breakdown != nil && extraBreakdown == nil {
							// Preserve formula breakdowns (heal, etc.) from
							// logs folded into the consolidated entry.
							extraBreakdown = log.Breakdown
						}
						resultLogs = append(resultLogs, *log)
					}
				}

				if useLog != nil {
					if useLog.Breakdown == nil {
						useLog.Breakdown = extraBreakdown
					} else if extraBreakdown != nil {
						if extraBreakdown.ShieldFinal > 0 || extraBreakdown.ShieldBase > 0 {
							useLog.Breakdown.ShieldBase = extraBreakdown.ShieldBase
							useLog.Breakdown.ShieldSkillMult = extraBreakdown.ShieldSkillMult
							useLog.Breakdown.ShieldBuffMult = extraBreakdown.ShieldBuffMult
							useLog.Breakdown.ShieldDaoMult = extraBreakdown.ShieldDaoMult
							useLog.Breakdown.ShieldDaoBonus = extraBreakdown.ShieldDaoBonus
							useLog.Breakdown.ShieldFinal = extraBreakdown.ShieldFinal
							if useLog.Breakdown.ShielderDefense == nil {
								useLog.Breakdown.ShielderDefense = extraBreakdown.ShielderDefense
							}
						}
						if extraBreakdown.HealingFinal > 0 || extraBreakdown.HealingBase > 0 {
							useLog.Breakdown.HealingBase = extraBreakdown.HealingBase
							useLog.Breakdown.HealingDaoBonus = extraBreakdown.HealingDaoBonus
							useLog.Breakdown.HealingFinal = extraBreakdown.HealingFinal
							if useLog.Breakdown.HealerHealing == nil {
								useLog.Breakdown.HealerHealing = extraBreakdown.HealerHealing
							}
						}
					}
					mergedDetail := useLog.Detail

					cleanDetail := func(detail string, actorName string) string {
						prefix := actorName + " "
						if strings.HasPrefix(detail, prefix) {
							return strings.TrimPrefix(detail, prefix)
						}
						return detail
					}

					if costLog != nil {
						mergedDetail += " " + cleanDetail(costLog.Detail, actor.Name)
					}

					for _, rLog := range resultLogs {
						mergedDetail += ", " + cleanDetail(rLog.Detail, actor.Name)
					}

					if gainLog != nil {
						parts := strings.Split(gainLog.Detail, "! ")
						if len(parts) > 1 {
							mergedDetail += " (Ngộ Đạo " + strings.TrimSuffix(parts[1], ".") + ")"
						} else {
							mergedDetail += " (" + strings.TrimPrefix(strings.TrimSuffix(gainLog.Detail, "."), "-> ") + ")"
						}
					}

					useLog.Detail = mergedDetail
					turnLogs = append(turnLogs[:actionStartIndex], *useLog)
					turnLogs = append(turnLogs, preservedLogs...)
				}
			}
			return res
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
			runOne(actor, &action, isAlly, targets)
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
							out := runOne(actor, a, isAlly, enemyTargets)
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
					out := runOne(actor, chargedAct, isAlly, enemyTargets)
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
					maxActions := state.MaxGuActionsPerTurn
					if maxActions < 1 {
						maxActions = 3
					}
					if control != nil && control.MaxGuActions > 0 {
						maxActions = control.MaxGuActions
					}
					selectedPlan = BuildAutomaticCombatPlan(actor, enemyTargets, maxActions)
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
						runOne(actor, &action, isAlly, enemyTargets)
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

					out := runOne(actor, a, isAlly, enemyTargets)
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
				out := runOne(actor, fallbackAction, isAlly, enemyTargets)
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

	// Huyết Nhiễm Chinh Bào: Hấp thụ 40% sát thương nhận vào chuyển hóa thành Huyết Thuẫn
	if (hasStatus(target, statusfx.StatusBloodRobe) || actorHasMechanic(target, "BEFORE_HIT", "BLOOD_STAINED_ROBE") || actorHasMechanic(target, "COMBAT_PASSIVE", "BLOOD_STAINED_ROBE")) && damage > 0 {
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

	// Cật Khuy Thị Phúc (EATING_LOSS_BLESSING): Chịu thiệt là phúc, mỗi khi bị đánh tích lũy 1 tầng phúc lợi (+5 Công, +5 Thủ)
	if hasStatus(target, statusfx.StatusEatingLoss) || actorHasMechanic(target, "BEFORE_HIT", "EATING_LOSS_BLESSING") || actorHasMechanic(target, "COMBAT_PASSIVE", "EATING_LOSS_BLESSING") {
		if target.CustomState == nil {
			target.CustomState = make(map[string]int)
		}
		if target.CustomState["eating_loss_stacks"] < 10 {
			target.CustomState["eating_loss_stacks"]++
			target.BaseDmg += 5
			target.BaseDef += 5
			*logs = append(*logs, CombatLog{
				Turn:      turn,
				ActorId:   target.ID,
				ActorName: target.Name,
				Action:    "EATING_LOSS_BLESSING",
				Detail:    fmt.Sprintf("[Thực Đạo - Cật Khuy Thị Phúc] %s chịu đòn rèn giũa, tích lũy tầng thứ %d phúc lực (+5 Công, +5 Thủ)!", target.Name, target.CustomState["eating_loss_stacks"]),
			})
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

	// Khí Tuyệt Phùng Sinh (QI_JUE_RETURN_LIFE): Khi HP dưới 30% bộc phát hồi 40% HP và xua tan debuff
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
	if target.GuHouse != nil && !target.GuHouse.IsDestroyed {
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
		damage -= absorbed
		target.GuHouse.HP = 0
		target.GuHouse.IsDestroyed = true
		*logs = append(*logs, CombatLog{
			Turn:      turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "GU_HOUSE_DESTROYED",
			Damage:    absorbed,
			Detail:    fmt.Sprintf("[Tiên Cổ Ốc Vỡ Vụn] Kiến trúc phòng ngự %s của %s bị công phá! %s phải trực tiếp nghênh địch!", target.GuHouse.Name, target.Name, target.Name),
		})
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
	if action != nil && action.IsFromGuHouse && (actor == nil || actor.GuHouse == nil || actor.GuHouse.IsDestroyed) {
		return false
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
