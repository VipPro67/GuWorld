package simulator

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// combatActionRunner owns the mutable combat context captured by the former
// Simulate.runOne closure. Keeping RNG, turn logs and summon IDs explicit makes
// the action runner extractable without changing resolution order.
type combatActionRunner struct {
	state       *CombatState
	rng         *rand.Rand
	turn        int
	turnLogs    *[]CombatLog
	summonCount *int
}

// runOne resolves one selected action against the current combat context.
func (r *combatActionRunner) runOne(actor *CombatantAuto, a *SimAction, isAlly bool, enemyTargets []*CombatantAuto) actionOutcome {
	res := actionOutcome{}
	if a != nil && !a.IsEffectOperation {
		a.PreAppliedEffectIDs = nil
	}
	operationDamageScale := 1.0
	guCostIDs := []string{}

	if (actor.IsSilenced || actor.IsFeared) && a.ActionType != "BASIC_ATTACK" {
		return res
	}

	if actor.GuHouse != nil && !actor.GuHouse.IsDestroyed && actor.GuHouse.Active && !a.IsFromGuHouse && a.ActionType != "FLEE" && a.ActionType != "EXIT_GU_HOUSE" {
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "GU_HOUSE_RESTRICTED",
			Detail:    fmt.Sprintf("%s đang dung nhập trong Tiên Cổ Phòng [%s], chỉ có thể thi triển Trận Kích của Cổ Phòng!", actor.Name, actor.GuHouse.Name),
		})
		return res
	}

	if a.ActionType == "EXIT_GU_HOUSE" {
		if actor.GuHouse != nil && actor.GuHouse.Active {
			actor.GuHouse.Active = false
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "GU_HOUSE_EXIT",
				Detail:    fmt.Sprintf("%s chủ động rút lui khỏi Tiên Cổ Phòng [%s], trở về trạng thái nhục thân!", actor.Name, actor.GuHouse.Name),
			})
		}
		return res
	}

	if a.ActionType == "ENTER_GU_HOUSE" {
		if actor.GuHouse != nil && !actor.GuHouse.IsDestroyed && !actor.GuHouse.Active {
			if actor.IsStunned || actor.HasGuSeal {
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "GU_HOUSE_ENTER_FAIL",
					Detail:    fmt.Sprintf("%s đang bị khống chế hoặc phong ấn Cổ, không thể dung nhập Tiên Cổ Phòng!", actor.Name),
				})
			} else {
				actor.GuHouse.Active = true
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "GU_HOUSE_ENTER",
					Detail:    fmt.Sprintf("%s một lần nữa dung nhập vào Tiên Cổ Phòng [%s]!", actor.Name, actor.GuHouse.Name),
				})
			}
		}
		return res
	}

	if actor.HasGuSeal && (a.ActionType == "GU_SKILL" || a.ActionType == "KILLER_MOVE") {
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "GU_SEALED",
			Detail:    fmt.Sprintf("%s bị phong ấn Cổ, không thể dùng [%s] lượt này!", actor.Name, a.Name),
		})
		return res
	}

	if actor.HasThoughtChaos && (a.ActionType == "GU_SKILL" || a.ActionType == "KILLER_MOVE") {
		a.ManaCost *= 2.0
		if r.rng.Float64() < 0.30 {
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "THOUGHT_CHAOS_FAIL",
				Detail:    fmt.Sprintf("[Trí Đạo - Tạp Niệm Loạn Ý] Đầu óc %s tràn ngập tạp niệm hỗn loạn! Niệm đầu bị phân tán, thi triển [%s] thất bại hoàn toàn!", actor.Name, a.Name),
			})
			return res
		} else {
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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

	if (a.ActionType == "KILLER_MOVE" || a.IsFromGuHouse) && actor.Kind == "Player" && len(a.ComponentGuIDs) > 0 {
		cost := a.HungerCost
		if cost <= 0 {
			cost = 1
		}
		canCast := true
		for _, guID := range a.ComponentGuIDs {
			h, exists := actor.GuHunger[guID]
			if !exists || h < cost {
				canCast = false
				break
			}
		}
		if !canCast {
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name, Action: "KILLER_MOVE_UNAVAILABLE", Detail: fmt.Sprintf("%s không thể dùng [%s] vì thiếu Cổ thành phần hoặc Cổ đã kiệt sức!", actor.Name, a.Name)})
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
	envMod := resolveEnvModifiersForAction(r.state, actor, a.Path)
	effectiveManaCost := a.ManaCost
	if effectiveManaCost > 0 {
		effectiveManaCost *= gamefx.ManaTaxMultiplier(conflictMult) * envMod.ManaTaxMult
	}
	if (actor.Kind == "Player" || actor.Kind == "VariantNPC") && effectiveManaCost > 0 && actor.Mana < effectiveManaCost {
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name, Action: "INSUFFICIENT_MANA", Detail: fmt.Sprintf("%s không đủ Chân Nguyên (cần %.0f, còn %.0f) để kích hoạt [%s]!", actor.Name, effectiveManaCost, actor.Mana, a.Name)})
		return res
	}
	if combatantPaysMana(actor) && effectiveManaCost > 0 {
		actor.Mana -= effectiveManaCost
	}
	if len(guCostIDs) > 0 {
		if actor.GuUsedCounts == nil {
			actor.GuUsedCounts = make(map[string]int)
		}
		cost := a.HungerCost
		if cost <= 0 {
			cost = 1
		}
		for _, id := range guCostIDs {
			if strings.HasPrefix(id, "used:") {
				actor.GuUsedCounts[strings.TrimPrefix(id, "used:")] += cost
			} else {
				actor.GuHunger[id] -= cost
				if actor.GuHunger[id] < 0 {
					actor.GuHunger[id] = 0
				}
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "GU_BACKLASH",
			Damage:    backlashDamage,
			Detail:    fmt.Sprintf("%s chịu %d sát thương phản phệ do dùng %s (Rank %d > Rank %d)!", actor.Name, int(backlashDamage), guName, guRank, actor.Rank),
		})
		if actor.HP == 0 {
			actor.IsDead = true
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "DIED",
				Detail:    fmt.Sprintf("%s đã gục ngã do phản phệ Cổ!", actor.Name),
			})
			res.abort = true
			return res
		}
	}

	actionStartIndex := len((*r.turnLogs))

	// ── Dao Conflict punishment (spec §3) ──
	conflictPenaltyMult := 1.0 // 0.5 when a non-fizzled backfire weakens the strike

	// Cast Failure (fizzle): conflicting dao marks shred the technique.
	if conflictMult < 0.70 && (a.ActionType == "GU_SKILL" || a.ActionType == "KILLER_MOVE") {
		if chance := gamefx.CastFailureChance(conflictMult); chance > 0 && r.rng.Float64() < chance {
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
		if chance := gamefx.BackfireChance(conflictMult); chance > 0 && r.rng.Float64() < chance {
			satKhi := satKhiPiercesImmunity(actor)
			if !isImmuneToDamage(actor, satKhi) {
				bfDmg := actor.MaxHP * (0.08 + r.rng.Float64()*0.04)
				actor.HP -= bfDmg
				applyLastStandFloor(actor, satKhi)
				if actor.HP < 0 {
					actor.HP = 0
				}
				conflictPenaltyMult = 0.5
				backfireReason := daoBackfireExplanation(actor, a, conflictMult, chance, bfDmg)
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "DAO_BACKFIRE",
					Damage:    bfDmg,
					Detail:    fmt.Sprintf("[Phản Phệ Nội Thương] %s chịu %d sát thương nội thương. %s", actor.Name, int(bfDmg), backfireReason),
				})
				if actor.HP <= 0 {
					actor.IsDead = true
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:      r.turn,
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
		if envMod.SuccessChanceDelta < 0 {
			floor := gamefx.DefaultEnvDaoConfig.SuccessChanceFloor
			if chance >= floor {
				chance = math.Max(floor, chance+envMod.SuccessChanceDelta)
			} else {
				chance = math.Max(0.05, chance+envMod.SuccessChanceDelta)
			}
		}
		roll := r.rng.Float64()
		if roll >= chance {
			backfireDmg := a.BackfireDamage
			if backfireDmg <= 0 {
				backfireDmg = math.Round(actor.MaxHP * 0.12)
			}
			if envMod.BackfireBonus > 0 {
				backfireDmg *= (1.0 + envMod.BackfireBonus)
			}
			if actor.GuHouse != nil && !actor.GuHouse.IsDestroyed {
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name, Action: "KILLER_MOVE_FAILED",
					Detail: fmt.Sprintf("%s thôi động %s [%s] thất bại (%.0f%% / tỉ lệ %.0f%%); Tiên Cổ Ốc [%s] đã che chắn áp lực phản phệ cho chủ nhân!", actor.Name, moveLabel, a.Name, roll*100, chance*100, actor.GuHouse.Name),
				})
			} else {
				actor.HP -= backfireDmg
				if actor.HP < 0 {
					actor.HP = 0
				}
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name, Action: "KILLER_MOVE_FAILED", Damage: backfireDmg,
					Detail: fmt.Sprintf("[Phản Phệ] %s thôi động %s [%s] thất bại (%.0f%% / tỉ lệ %.0f%%), chân nguyên bạo tẩu gây %.0f sát thương phản phệ!", actor.Name, moveLabel, a.Name, roll*100, chance*100, backfireDmg),
				})
				if a.BackfireStatus != "" {
					actor.HasThoughtChaos = true
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name, Action: "BACKFIRE_STATUS",
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

	r.dispatchPhase(combatPhaseBeforeEnemyAttack, combatPhaseContext{actor: actor, action: a, targets: enemyTargets})

	if (a.ActionType == "GU_SKILL" || a.ActionType == "BASIC_ATTACK" || a.ActionType == "KILLER_MOVE") && (actor.Kind == "Player" || actor.Kind == "Ally") && !r.state.VirtualMode {
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

				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "PATH_MASTERY_GAIN",
					Detail:    fmt.Sprintf("-> Ngộ Đạo tăng tiến (%s)! +%d điểm Chân Ý %s (Cảnh giới: %s).", a.Name, pathGain, strings.Title(cleanPath), lvlName),
				})
				if clamped {
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:      r.turn,
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
			child.SourceID = effect.SourceGuID
			child.InstanceID = effect.SourceInstanceID
			if child.InstanceID == "" {
				child.InstanceID = a.InstanceID
			}
			if child.SourceID == "" {
				child.SourceID = a.SourceID
			}
			out := r.runOne(actor, &child, isAlly, enemyTargets)
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
	} else if (a.IsKillerMoveOperation || a.IsEffectOperation) && strings.EqualFold(a.TargetType, "ALL_ALLIES") {
		var allies []*CombatantAuto
		if isAlly {
			allies = r.state.Allies
		} else {
			allies = r.state.Enemies
		}
		resolved = make([]*CombatantAuto, 0, len(allies))
		for _, member := range allies {
			if member != nil && member.HP > 0 && !member.IsDead {
				resolved = append(resolved, member)
			}
		}
	}
	if len(a.OperationConditions) > 0 {
		resolved = filterOperationTargets(actor, resolved, a.OperationConditions, a.Rank, a.Path, r.turn)
	}
	// ── BEFORE_ENEMY_ATTACK: FORGET_WILL confuse ──
	// Redirects (or fizzles) the just-resolved targets. First
	// defending owner that procs decides; backfire seals the owner.
	if len(resolved) > 0 && len(a.OperationEffects) == 0 && hasEligibleHostileDamage(actor, a, enemyTargets, r.turn) {
		for _, d := range enemyTargets {
			if d == nil || d.HP <= 0 || d.IsDead {
				continue
			}
			mechanicsSlice := d.Mechanics
			if !hasMechanic(mechanicsSlice, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE") {
				for _, eff := range d.PassiveOperationEffects {
					if eff.Action != nil && hasMechanic(eff.Action.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE") {
						mechanicsSlice = eff.Action.Mechanics
						break
					}
				}
			}
			if !hasMechanic(mechanicsSlice, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE") {
				continue
			}
			// Nghịch Chuyển Cổ: hostile Gu effects aimed at the holder
			// are reversed — the confusion fizzles, the strike lands.
			if immuneToHostileEffects(actor) || isGuaranteedHit(actor, a) {
				reason := "Nghịch Chuyển Cổ đảo nghịch"
				if isGuaranteedHit(actor, a) {
					reason = "Đòn đánh Tất Trúng phớt lờ"
				}
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "REVERSE_NEGATE",
					Detail:    fmt.Sprintf("%s [Lãng Quên Chí] của %s về hư vô!", reason, d.Name),
				})
				continue
			}
			willChance := mechanicFloat(mechanicsSlice, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "activation_chance", 0.35)
			willRoll := r.rng.Float64()
			if willRoll >= willChance {
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   d.ID,
					ActorName: d.Name,
					Action:    "FORGET_WILL_INACTIVE",
					Detail:    fmt.Sprintf("[Lãng Quên Chí] của %s không kích hoạt: roll %.0f%% >= tỉ lệ %.0f%%!", d.Name, willRoll*100, willChance*100),
				})
				continue
			}
			if bf := mechanicFloat(mechanicsSlice, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "backfire_chance", 0.2); r.rng.Float64() < bf {
				sealGuTurns(d, mechanicInt(mechanicsSlice, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "backfire_disable_turns", 2))
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   d.ID,
					ActorName: d.Name,
					Action:    "FORGET_WILL_BACKFIRE",
					Detail:    fmt.Sprintf("%s bị phản phệ khi bóp méo ý chí, Cổ bị phong ấn!", d.Name),
				})
			}
			roll := r.rng.Float64()
			hu := mechanicFloat(mechanicsSlice, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "hit_user_chance", 0.15)
			hs := mechanicFloat(mechanicsSlice, "BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE", "hit_self_chance", 0.35)
			switch {
			case roll < hu:
				resolved = []*CombatantAuto{d}
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FORGET_WILL_REDIRECT",
					Detail:    fmt.Sprintf("%s quên mất mục tiêu, đòn [%s] lạc sang %s!", actor.Name, a.Name, d.Name),
				})
			case roll < hu+hs:
				resolved = []*CombatantAuto{actor}
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "FORGET_WILL_SELFHIT",
					Detail:    fmt.Sprintf("%s quên mất mục tiêu, tự đánh bản thân bằng [%s]!", actor.Name, a.Name),
				})
			default:
				resolved = nil
				res.executed = true
				res.hasDamage = false
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
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
	for _, counter := range a.CounterAdds {
		for _, target := range resolved {
			ApplyCounter(target, counter.Key, counter.Delta, counter.Min, counter.Max)
		}
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
		totalShield := (baseShield + daoShield) * envMod.PowerMult
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
			Turn:      r.turn,
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
		heal := (baseHeal + daoHeal) * envMod.PowerMult * effectiveHealingReceived(actor)
		actor.HP += heal
		if actor.HP > actor.MaxHP {
			actor.HP = actor.MaxHP
		}
		hl := CombatLog{
			Turn:      r.turn,
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
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "MANA_RESTORE",
				Damage:    restore,
				Detail:    fmt.Sprintf("%s khôi phục %.0f %s!", actor.Name, restore, actor.GetManaName()),
			}
			manaLog = &ml
		}
	}
	if a.ManaRestoreTargetFlat > 0 || a.ManaRestoreTargetPercent > 0 {
		for _, target := range resolved {
			if target == nil || target.HP <= 0 || target.IsDead {
				continue
			}
			requested := a.ManaRestoreTargetFlat + target.MaxMana*a.ManaRestoreTargetPercent
			if requested <= 0 {
				continue
			}
			restored := math.Min(target.MaxMana-target.Mana, requested)
			if restored <= 0 {
				continue
			}
			target.Mana += restored
			if !a.ManaRestoreSuppressLog {
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name,
					TargetId: target.ID, Action: "MANA_RESTORE", Damage: restored,
					Detail: fmt.Sprintf("%s khôi phục %.0f %s!", target.Name, restored, target.GetManaName()),
				})
			}
		}
	}

	// Apply cleanse to the targets selected by a typed effect operation; legacy
	// action-level cleanses continue to affect only the actor.
	if a.CleanseDebuffs {
		cleanseTargets := []*CombatantAuto{actor}
		if (a.IsEffectOperation || a.IsKillerMoveOperation) && !strings.EqualFold(a.TargetType, "SELF") {
			cleanseTargets = resolved
		}
		totalCleansed := 0
		for _, cleanseTarget := range cleanseTargets {
			if cleanseTarget == nil {
				continue
			}
			var keptEffects []Effect
			cleansedCount := 0
			for _, eff := range cleanseTarget.ActiveEffects {
				if eff.Type == "debuff" {
					cleansedCount++
				} else {
					keptEffects = append(keptEffects, eff)
				}
			}
			cleanseTarget.ActiveEffects = keptEffects
			if len(cleanseTarget.ActiveDoT) > 0 {
				cleansedCount += len(cleanseTarget.ActiveDoT)
				cleanseTarget.ActiveDoT = nil
			}
			if cleansedCount > 0 {
				totalCleansed += cleansedCount
				ComputeEffectMultipliers(cleanseTarget)
			}
		}
		if totalCleansed > 0 {
			detail := fmt.Sprintf("%s tịnh hóa bản thân, xóa bỏ %d hiệu ứng bất lợi!", actor.Name, totalCleansed)
			if (a.IsEffectOperation || a.IsKillerMoveOperation) && strings.EqualFold(a.TargetType, "ALL_ALLIES") {
				detail = fmt.Sprintf("%s tịnh hóa toàn bộ phe mình, xóa bỏ %d hiệu ứng bất lợi!", actor.Name, totalCleansed)
			}
			cleanseLog = &CombatLog{Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name, Action: "CLEANSE", Detail: detail}
		}
	}
	if len(a.CleanseStatusIDs) > 0 {
		cleanseTargets := []*CombatantAuto{actor}
		if (a.IsEffectOperation || a.IsKillerMoveOperation) && !strings.EqualFold(a.TargetType, "SELF") {
			cleanseTargets = resolved
		}
		for _, cleanseTarget := range cleanseTargets {
			for _, statusID := range a.CleanseStatusIDs {
				cleanseCombatStatus(cleanseTarget, statusID)
			}
		}
	}

	// ── Special Actions preprocessing ──
	if hasMechanic(a.Mechanics, "ON_CAST", "SUMMON_WOLF_FRONTLINE") {
		// Summoned beasts scale with the summoning Gu's dao marks, exactly
		// like the damage/shield/heal of the action itself (daoMult above).
		detail := spawnWolfFrontline(actor, isAlly, r.state, daoMult)
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
	if m, ok := findMechanic(a.Mechanics, "ON_CAST", "SUMMON_MONSTER"); ok && !hasMinion(r.state, actor.ID) {
		_, detail := spawnSummonedMonster(actor, m, a.Rank, isAlly, r.state, *r.summonCount, daoMult)
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SUMMONED",
			Detail:    detail,
		})
		(*r.summonCount)++
	}

	if hasMechanic(a.Mechanics, "ON_CAST", "FULL_SOUL_HEAL") {
		actor.HP = actor.MaxHP
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "BUFF",
			Detail:    fmt.Sprintf("%s mượn sức mạnh đỉnh phong từ quá khứ (+40%% công kích trong 3 lượt)!", actor.Name),
		})
	}

	r.dispatchPhase(combatPhaseOnCast, combatPhaseContext{actor: actor, action: a, firstTarget: firstTarget, targets: enemyTargets})

	if hasMechanic(a.Mechanics, "ON_CAST", "CHAOS_GAMBLE") {
		effectsChoices := []string{"crit_500", "invulnerable_1_turn", "reduce_enemy_hp_90"}
		costChoices := []string{"aptitude_10", "lifespan_5", "luck_50"}
		for _, m := range a.Mechanics {
			if m.ActionCode == "CHAOS_GAMBLE" && len(m.Params) > 0 {
				var p struct {
					EffectChoices []string `json:"effect_choices"`
					CostChoices   []string `json:"cost_choices"`
				}
				if json.Unmarshal(m.Params, &p) == nil {
					if len(p.EffectChoices) > 0 {
						effectsChoices = p.EffectChoices
					}
					if len(p.CostChoices) > 0 {
						costChoices = p.CostChoices
					}
				}
			}
		}

		chosenEffect := effectsChoices[r.rng.Intn(len(effectsChoices))]
		chosenCost := costChoices[r.rng.Intn(len(costChoices))]

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
		settlement := &SettlementEffect{Code: chosenCost}
		if chosenCost == "aptitude_10" {
			costMsg = "Trừ 10% Tư chất gốc vĩnh viễn!"
			settlement.Amount = 10
		} else if chosenCost == "lifespan_5" {
			costMsg = "Trừ 5 năm tuổi thọ vĩnh viễn!"
			settlement.Amount = 5
		} else if chosenCost == "luck_50" {
			costMsg = "Trừ 50% Nguyên Thạch hiện tại!"
			settlement.Amount = 50
		}

		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:       r.turn,
			ActorId:    actor.ID,
			ActorName:  actor.Name,
			Action:     "CHAOS_GAMBLE",
			Detail:     fmt.Sprintf("[%s] kích hoạt Thần Thông Cổ Nhiệm Ý! %s Cái giá phải trả: %s", a.Name, effectMsg, costMsg),
			Settlement: settlement,
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

		greatLuckChance := mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "great_chance_percent", 15)
		if hasCurseReverse {
			greatLuckChance += mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "curse_great_chance_bonus", 50)
			if curseIdx >= 0 && curseIdx < len(actor.ActiveEffects) {
				actor.ActiveEffects = append(actor.ActiveEffects[:curseIdx], actor.ActiveEffects[curseIdx+1:]...)
			}
		}

		curseDuration := int(mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "curse_duration_turns", 2))
		roll := r.rng.Float64() * 100.0
		if roll < greatLuckChance {
			// Đại Cát
			multiplier := mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "great_damage_multiplier", 3)
			a.Damage *= multiplier
			operationDamageScale = multiplier
			actor.Mana += a.ManaCost * mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "great_mana_cost_multiplier", 1)
			if actor.Mana > actor.MaxMana {
				actor.Mana = actor.MaxMana
			}
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "LUCK_ROULETTE",
				Detail:    fmt.Sprintf("[Vận Đạo - Đại Cát!] Vận thế ngút trời! [%s] bạo phát x%.1f sát thương và phản hồi %.0f%% Chân Nguyên!", a.Name, multiplier, mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "great_mana_cost_multiplier", 1)*100),
			})
		} else if roll < greatLuckChance+mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "medium_range_percent", 45) {
			// Trung Cát
			multiplier := mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "medium_damage_multiplier", 1.5)
			a.Damage *= multiplier
			operationDamageScale = multiplier
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "LUCK_ROULETTE",
				Detail:    fmt.Sprintf("[Vận Đạo - Trung Cát!] Khí vận hanh thông, [%s] gia tăng x%.1f sát thương!", a.Name, multiplier),
			})
		} else if roll < greatLuckChance+mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "medium_range_percent", 45)+mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "neutral_range_percent", 30) {
			// Bình Vận
			multiplier := mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "neutral_damage_multiplier", 1)
			a.Damage *= multiplier
			operationDamageScale = multiplier
			detail := fmt.Sprintf("[Vận Đạo - Bình Vận] Vận thế bình ổn, [%s] gây sát thương cơ bản.", a.Name)
			if multiplier != 1 {
				detail = fmt.Sprintf("[Vận Đạo - Bình Vận] Vận thế bình ổn, [%s] gây sát thương x%.1f.", a.Name, multiplier)
			}
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "LUCK_ROULETTE",
				Detail:    detail,
			})
		} else {
			// Đại Hung
			multiplier := mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "bad_damage_multiplier", 0.5)
			a.Damage *= multiplier
			operationDamageScale = multiplier
			selfDmg := actor.HP * mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "bad_current_hp_loss_percent", 0.10)
			minimumSelfDamage := mechanicFloatByCode(a.Mechanics, "LUCK_ROULETTE", "minimum_bad_hp_loss", 1)
			if selfDmg < minimumSelfDamage {
				selfDmg = minimumSelfDamage
			}
			actor.HP -= selfDmg
			if actor.HP < 1.0 {
				actor.HP = 1.0
			}
			actor.ActiveEffects = append(actor.ActiveEffects, Effect{
				ID: statusfx.StatusLuckCurseReverse, Type: "buff", Value: 1, Duration: curseDuration,
			})
			damageChange := (1 - multiplier) * 100
			damageText := fmt.Sprintf("giảm %.0f%% sát thương", damageChange)
			if damageChange < 0 {
				damageText = fmt.Sprintf("tăng %.0f%% sát thương", -damageChange)
			}
			curseText := "Kích hoạt [Họa Hề Phúc Sở Ỷ]: Tăng mạnh vận khí hiệp kế!"
			if curseDuration != 2 {
				curseText = fmt.Sprintf("Kích hoạt [Họa Hề Phúc Sở Ỷ]: Tăng vận khí trong %d lượt!", curseDuration)
			}
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "LUCK_ROULETTE",
				Detail:    fmt.Sprintf("[Vận Đạo - Đại Hung!] Khí vận phản phệ, [%s] %s, tự tổn %.0f HP! (%s)", a.Name, damageText, selfDmg, curseText),
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
		if mechanicBoolByCode(a.Mechanics, "DREAM_UNRAVEL", "clear_control_states", true) {
			actor.IsStunned = false
			actor.IsFrozen = false
			actor.IsSleeping = false
			actor.IsDreamPrison = false
			actor.IsBound = false
			actor.IsSilenced = false
			actor.IsFeared = false
		}

		conversionUnits := mechanicFloatByCode(a.Mechanics, "DREAM_UNRAVEL", "base_conversion_units", 1) + float64(dispelledCount)
		manaPercent := mechanicFloatByCode(a.Mechanics, "DREAM_UNRAVEL", "mana_percent_per_unit", 0.15)
		shieldPercent := mechanicFloatByCode(a.Mechanics, "DREAM_UNRAVEL", "shield_percent_per_unit", 0.10)
		manaGain := actor.MaxMana * (manaPercent * conversionUnits)
		shieldGain := actor.MaxHP * (shieldPercent * conversionUnits)
		actor.Mana += manaGain
		if actor.Mana > actor.MaxMana {
			actor.Mana = actor.MaxMana
		}
		actor.VirtualShield += shieldGain

		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "DREAM_UNRAVEL",
			Detail:    fmt.Sprintf("[Mộng Đạo - Giải Mộng] %s giải trừ %d tầng mê chướng tâm lý, chuyển hóa thành %.0f Chân Nguyên và %.0f Mộng Giáp hộ thể!", actor.Name, dispelledCount, manaGain, shieldGain),
		})
	}

	// PRE_RIDER is the historical hook point for blood-sacrifice HP costs.
	r.applyHPCosts(combatPhaseContext{actor: actor, action: a}, "PRE_RIDER")

	// Huyết Đạo: Huyết Tế Cuồng Bạo (BLOOD_SACRIFICE), legacy persisted-hook fallback.
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "CALCULATED_STRIKE",
			Detail:    fmt.Sprintf("[Trí Đạo - Động Tất Sơ Hở] %s nhìn thấu toàn bộ sơ hở của %s, phát động đòn đánh tất trúng gia tăng 35%% uy lực!", actor.Name, targetName),
		})
	}

	// Hồn Đạo: Đãng Hồn Kích (SOUL_SHOCK)
	if hasMechanic(a.Mechanics, "ON_ACTION", "SOUL_SHOCK") || hasMechanic(a.Mechanics, "ON_HIT", "SOUL_SHOCK") {
		targetName := "kẻ địch"
		if firstTarget != nil {
			targetName = firstTarget.Name
		}
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SOUL_SHOCK",
			Detail:    fmt.Sprintf("[Hồn Đạo - Đãng Hồn Kích] %s chấn động hồn phách, đánh thẳng vào Thần Hồn của %s, bỏ qua Chân Nguyên Giáp!", actor.Name, targetName),
		})
	}

	// Hồn Đạo: Phệ Hồn Hóa Lực (SOUL_DEVOUR)
	if hasMechanic(a.Mechanics, "ON_ACTION", "SOUL_DEVOUR") {
		healAmt := actor.MaxHP * 0.20
		actor.HP += healAmt
		if actor.HP > actor.MaxHP {
			actor.HP = actor.MaxHP
		}
		stk := addBoundedCounter(actor, "soul_devour_stacks", 1, 0, 5)
		targetName := "kẻ địch"
		if firstTarget != nil {
			targetName = firstTarget.Name
		}
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SOUL_DEVOUR",
			Detail:    fmt.Sprintf("[Hồn Đạo - Phệ Hồn Hóa Lực] %s cắn nuốt hồn lực của %s, hồi phục %.0f HP và tăng 10%% uy lực tấn công (tầng %d)!", actor.Name, targetName, healAmt, stk),
		})
	}

	// Keep the typed operation and persisted native hook at the original RNG
	// position; moving this proc later changes every subsequent draw for a seed.
	r.dispatchPhase(combatPhaseBonusStrike, combatPhaseContext{
		actor: actor, action: a, firstTarget: firstTarget, didDamage: &res.hasDamage,
	})

	// Huyết Đạo: Huyết Thao Thiết (HEAVY_LIFESTEAL)
	if hasMechanic(a.Mechanics, "ON_ACTION", "HEAVY_LIFESTEAL") {
		a.LifeStealPct = 0.50
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "HEAVY_LIFESTEAL",
			Detail:    fmt.Sprintf("[Huyết Đạo - Huyết Thao Thiết] %s vận chuyển huyết vụ, kích hoạt 50%% hút máu từ sát thương gây ra!", actor.Name),
		})
	}

	// Huyết Đạo: Huyết Bạo Trùng Điệp (BLEED_BURST)
	r.applyStatusDetonations(combatPhaseContext{actor: actor, action: a, firstTarget: firstTarget}, "EARLY_RIDER")
	if hasMechanic(a.Mechanics, "ON_ACTION", "BLEED_BURST") && firstTarget != nil {
		bleedStacks, _ := removeStatusStacks(firstTarget, statusfx.StatusBleed, EffectBleed)
		if bleedStacks > 0 {
			detonateDmg := float64(bleedStacks*100) + firstTarget.MaxHP*0.08
			firstTarget.HP -= detonateDmg
			if firstTarget.HP < 0 {
				firstTarget.HP = 0
			}
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "BLEED_BURST",
				TargetId:  firstTarget.ID,
				Damage:    detonateDmg,
				Detail:    fmt.Sprintf("[Huyết Đạo - Huyết Bạo Trùng Điệp] Kích nổ %d tầng huyết dịch trên người %s, tạo ra vụ nổ máu xé rách gây %.0f sát thương trực diện!", bleedStacks, firstTarget.Name, detonateDmg),
			})
		}
	}

	// Typed resource-transfer operations and the legacy MANA_STEAL hook share
	// the same transfer semantics and execute at the hook's historical point.
	if firstTarget != nil {
		for _, transfer := range a.ResourceTransfers {
			if transfer.Resource == "MANA" {
				r.applyResourceTransfer(actor, firstTarget, transfer)
			}
		}
	}
	if (hasMechanic(a.Mechanics, "ON_ACTION", "MANA_STEAL") || hasMechanic(a.Mechanics, "ON_HIT", "MANA_STEAL")) && firstTarget != nil {
		r.applyResourceTransfer(actor, firstTarget, ResourceTransfer{
			Resource:  "MANA",
			SubType:   "PERCENT_OF_TARGET",
			Value:     0.25,
			Message:   "[Trộm Đạo - Thâu Chân Nguyên] {actor} thôi động diệu thủ cướp đoạt {amount} Chân Nguyên từ {target} nạp vào bản thân!",
			LogAction: "MANA_STEAL",
		})
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
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "BUFF_STEAL",
				TargetId:  firstTarget.ID,
				Detail:    fmt.Sprintf("[Trộm Đạo - Thâu Thiên Hoán Nhật] %s trộm lấy buff [%s] của %s mang về cho mình!", actor.Name, stolenBuffName, firstTarget.Name),
			})
		}
	}

	// Typed combat passives run at the former PACK_FRENZY position so modifiers
	// preserve their historical ordering relative to action riders.
	passiveAllies := r.state.Allies
	if !isAlly {
		passiveAllies = r.state.Enemies
	}
	r.dispatchPhase(combatPhasePassiveDamageModifier, combatPhaseContext{actor: actor, action: a, damageScale: &operationDamageScale, firstTarget: firstTarget, allies: passiveAllies})

	// Nô Đạo: Quân Đoàn Hợp Kích (PACK_FRENZY) remains for persisted native hooks.
	if hasMechanic(a.Mechanics, "ON_ACTION", "PACK_FRENZY") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "PACK_FRENZY") {
		minionCount := 0
		combatantsList := r.state.Allies
		if !isAlly {
			combatantsList = r.state.Enemies
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
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
			ID: statusfx.StatusBearForm, Value: 30, Duration: 3,
		})
		shieldGain := actor.MaxHP * 0.25
		actor.VirtualShield += shieldGain
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "FORM_EAGLE",
			Detail:    fmt.Sprintf("[Biến Hóa Đạo - Hóa Thân Kim Điêu] %s hóa thân Vân Lạc Kim Điêu, giương cánh tăng vọt Tốc độ và Né tránh!", actor.Name),
		})
	}

	// Biến Hóa Đạo: Vạn Vật Thích Ứng (DAO_ADAPTIVE_RESIST)
	if hasMechanic(a.Mechanics, "ON_ACTION", "DAO_ADAPTIVE_RESIST") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "DAO_ADAPTIVE_RESIST") {
		actor.ActiveEffects = append(actor.ActiveEffects, Effect{
			ID: statusfx.StatusAdaptiveResist, Value: 25, Duration: 3,
		})
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "DAO_ADAPTIVE_RESIST",
			Detail:    fmt.Sprintf("[Biến Hóa Đạo - Vạn Vật Thích Ứng] %s cơ thể biến ảo thích ứng với đạo ngân công kích, tăng 25%% giảm thương nhận vào!", actor.Name),
		})
	}

	// Độc Đạo: Vạn Độc Bộc Phát (POISON_DETONATE)
	r.applyStatusDetonations(combatPhaseContext{actor: actor, action: a, firstTarget: firstTarget}, "LATE_RIDER")
	if hasMechanic(a.Mechanics, "ON_ACTION", "POISON_DETONATE") && firstTarget != nil {
		_, totalPoisonVal := removeStatusStacks(firstTarget, statusfx.StatusPoison)
		if totalPoisonVal > 0 {
			detonateDmg := totalPoisonVal * 2.5
			if detonateDmg < 60.0 {
				detonateDmg = 60.0
			}
			firstTarget.HP -= detonateDmg
			if firstTarget.HP < 0 {
				firstTarget.HP = 0
			}
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "WOOD_BIND",
			TargetId:  firstTarget.ID,
			Detail:    fmt.Sprintf("[Mộc Đạo - Mộc Lao Trói Buộc] Cổ thụ rễ cây trồi lên trói chặt %s, phong tỏa hành động và triệt tiêu khả năng né tránh!", firstTarget.Name),
		})
	}

	// Thủy Đạo: Thủy Lưu Triều Tịch (WATER_CLEANSE)
	if hasMechanic(a.Mechanics, "ON_CAST", "WATER_CLEANSE") || hasMechanic(a.Mechanics, "ON_ACTION", "WATER_CLEANSE") {
		cleansedCount, _ := removeStatusStacks(actor, statusfx.StatusBurn, statusfx.StatusBleed, statusfx.StatusPoison)
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "FIRE_INFERNO",
			TargetId:  firstTarget.ID,
			Detail:    fmt.Sprintf("[Hỏa Đạo - Liệt Hỏa Liêu Nguyên] Ngọn lửa hừng hực thiêu đốt %s, mỗi lượt cắn nuốt 8%% sinh lực!", firstTarget.Name),
		})
	}

	// Hỏa Đạo: Nộ Hỏa Bạo Phát (IGNITE_BURST)
	if hasMechanic(a.Mechanics, "ON_ACTION", "IGNITE_BURST") && firstTarget != nil {
		if hasStatus(firstTarget, statusfx.StatusBurn) {
			a.Damage *= 1.60
			operationDamageScale *= 1.60
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		if r.rng.Float64() < 0.40 {
			firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
				ID: statusfx.StatusStun, Duration: 1,
			})
			stunned = true
		}
		stunStr := ""
		if stunned {
			stunStr = " và gây Choáng váng 1 lượt"
		}
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "LIGHTNING_STUN_BURST",
			TargetId:  firstTarget.ID,
			Detail:    fmt.Sprintf("[Lôi Đạo - Lôi Đình Vạn Quân] %s giáng cuồng lôi tăng 30%% uy lực%s lên %s!", actor.Name, stunStr, firstTarget.Name),
		})
	}

	r.dispatchPhase(combatPhasePreChain, combatPhaseContext{actor: actor, action: a, damageScale: &operationDamageScale, firstTarget: firstTarget, targets: resolved})
	r.dispatchPhase(combatPhaseChainDamage, combatPhaseContext{actor: actor, action: a, firstTarget: firstTarget, targets: enemyTargets})

	// Băng Đạo: Băng Phong Vạn Lý (ICE_DEEP_FREEZE)
	if (hasMechanic(a.Mechanics, "ON_ACTION", "ICE_DEEP_FREEZE") || hasMechanic(a.Mechanics, "ON_HIT", "ICE_DEEP_FREEZE")) && firstTarget != nil {
		firstTarget.ActiveEffects = append(firstTarget.ActiveEffects, Effect{
			ID: statusfx.StatusFreeze, Duration: 1,
		})
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "TIEP_THIEN_DIA",
			Detail:    fmt.Sprintf("[Thiên Đạo - Tiếp Thiên Tiếp Địa] %s câu thông ý chí Thiên Đạo, giải trừ toàn bộ khống chế và hồi %.0f Chân Nguyên!", actor.Name, manaGain),
		})
	}

	r.applyPassiveOperationsPhase(combatPhaseContext{actor: actor, action: a, damageScale: &operationDamageScale, firstTarget: firstTarget}, "EARLY_RIDER")

	// Nhân Đạo: Nhân Định Thắng Thiên (HUMAN_WILL), legacy persisted-hook fallback.
	if (hasMechanic(a.Mechanics, "ON_ACTION", "HUMAN_WILL") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "HUMAN_WILL")) && actor.MaxHP > 0 {
		if (actor.HP / actor.MaxHP) <= 0.40 {
			a.Damage *= 1.50
			operationDamageScale *= 1.50
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "JUSTICE_VERDICT",
			TargetId:  firstTarget.ID,
			Detail:    fmt.Sprintf("[Quy Tắc Đạo - Phán Quyết Công Lý] Thiết diện vô tư, công lý phán xét trừng phạt %s tăng thêm 35%% sát thương!", firstTarget.Name),
		})
	}

	allies := r.state.Allies
	if !isAlly {
		allies = r.state.Enemies
	}
	r.dispatchPhase(combatPhaseOnActionRider, combatPhaseContext{actor: actor, action: a, damageScale: &operationDamageScale, firstTarget: firstTarget, targets: resolved, allies: allies})

	// Trận Đạo: Trận Địa Khống Chế (FORMATION_DOMAIN)
	if hasMechanic(a.Mechanics, "ON_CAST", "FORMATION_DOMAIN") || hasMechanic(a.Mechanics, "ON_ACTION", "FORMATION_DOMAIN") {
		shieldGain := actor.MaxHP * 0.20
		actor.VirtualShield += shieldGain
		actor.ActiveEffects = append(actor.ActiveEffects, Effect{
			ID: statusfx.StatusDefUp, Value: 25, Duration: 3,
		})
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "FORMATION_DOMAIN",
			Detail:    fmt.Sprintf("[Trận Đạo - Trận Địa Khống Chế] Cổ Trận khởi động bao phủ sàn đấu, tăng 25%% phòng thủ và tạo %.0f Trận Giáp cho %s!", shieldGain, actor.Name),
		})
	}

	// Huyễn Đạo: Huyễn Ảnh Mê Trận (BLINDING_MIRAGE)
	if (hasMechanic(a.Mechanics, "ON_ACTION", "BLINDING_MIRAGE") || hasMechanic(a.Mechanics, "ON_HIT", "BLINDING_MIRAGE")) && firstTarget != nil {
		firstTarget.ActiveEffects = append(firstTarget.ActiveEffects,
			Effect{ID: statusfx.StatusConfuse, Value: 1, Duration: 2},
			Effect{ID: statusfx.StatusBlind, Value: 40, Duration: 2},
		)
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "PAINTING_IMPRISONMENT",
			TargetId:  firstTarget.ID,
			Detail:    fmt.Sprintf("[Họa Đạo - Họa Địa Vi Lao] %s vung bút điểm mặc, nhốt chặt %s vào trong bức tranh thủy mặc phong tỏa hành động!", actor.Name, firstTarget.Name),
		})
	}

	r.applyPassiveOperationsPhase(combatPhaseContext{actor: actor, action: a, damageScale: &operationDamageScale, firstTarget: firstTarget, allies: allies}, "LATE_RIDER")

	// Tình Đạo: Tình Cổ Vô Hối (PASSION_OVERLOAD), legacy persisted-hook fallback.
	if (hasMechanic(a.Mechanics, "ON_ACTION", "PASSION_OVERLOAD") || hasMechanic(actor.Mechanics, "COMBAT_PASSIVE", "PASSION_OVERLOAD")) && actor.MaxHP > 0 {
		if (actor.HP / actor.MaxHP) <= 0.35 {
			a.Damage *= 1.50
			operationDamageScale *= 1.50
			actor.IsStunned = false
			actor.IsFrozen = false
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "PASSION_OVERLOAD",
				Detail:    fmt.Sprintf("[Tình Đạo - Tình Cổ Vô Hối] Tình sâu tựa hải, nghịch cảnh sinh tình! %s tiến vào trạng thái Si Tình bộc phát 50%% uy lực và kháng khống chế!", actor.Name),
			})
		}
	}

	r.applyHPCosts(combatPhaseContext{actor: actor, action: a}, "EARLY_RIDER")

	// Luyện Đạo: Tự Bạo Cổ Trùng (GU_DETONATION), legacy persisted-hook fallback.
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		team := r.state.Allies
		opponents := r.state.Enemies
		if !isAlly {
			team = r.state.Enemies
			opponents = r.state.Allies
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "INFINITE_WIND",
			Detail:    fmt.Sprintf("[Phong Đạo - Vô Hạn Phong] Gió vô cùng vô tận bùng nổ! Tăng 75%% sát thương, +60%% Thần Tốc và +35%% Né Tránh cho %s trong 3 hiệp!", actor.Name),
		})
	}

	// Huyết Đạo: Huyết Ma Giải Thể (BLOOD_DEMON_DISINTEGRATION)
	r.applyHPCosts(combatPhaseContext{actor: actor, action: a}, "LATE_RIDER")
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		opponents := r.state.Enemies
		if !isAlly {
			opponents = r.state.Allies
		}
		for _, opp := range opponents {
			if opp != nil && opp.HP > 0 {
				applyDamageOutputModifier(opp, TargetDamageModifier{Value: 0.30, MinMultiplier: 0.20})
				opp.ActiveEffects = append(opp.ActiveEffects, Effect{
					ID: statusfx.StatusSlow1, Value: 30, Duration: 3, Type: "debuff",
				})
			}
		}
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "LANDSCAPE_AS_BEFORE",
			Detail:    fmt.Sprintf("[Trụ Đạo - Giang Sơn Như Cố] Giang sơn như cũ, cảnh cũ người xưa! %s khôi phục %.0f Giáp và tẩy sạch toàn bộ hiệu ứng suy yếu!", actor.Name, shieldGain),
		})
	}

	// FINAL_RIDER is the historical payout point for EATING_LOSS_BLESSING.
	r.applyPassiveOperationsPhase(combatPhaseContext{
		actor: actor, action: a, damageScale: &operationDamageScale,
		firstTarget: firstTarget, targets: resolved, allies: allies,
	}, "FINAL_RIDER")

	// Thực Đạo: Cật Khuy Thị Phúc (EATING_LOSS_BLESSING) - compatibility log
	if hasMechanic(a.Mechanics, "ON_ACTION", "EATING_LOSS_BLESSING") || actorHasMechanic(actor, "COMBAT_PASSIVE", "EATING_LOSS_BLESSING") {
		stacks := 0
		if actor.CustomState != nil {
			stacks = actor.CustomState["eating_loss_stacks"]
		}
		mult := 1.30 + (float64(stacks) * 0.15)
		damageModifierInPrimitive := mechanicBool(a.Mechanics, "ON_ACTION", "EATING_LOSS_BLESSING", "damage_modifier_in_primitive", false) || mechanicBool(actor.Mechanics, "COMBAT_PASSIVE", "EATING_LOSS_BLESSING", "damage_modifier_in_primitive", false)
		if !damageModifierInPrimitive {
			a.Damage *= mult
			operationDamageScale *= mult
		}
		healGain := float64(stacks) * 0.05 * actor.MaxHP
		healInPrimitive := mechanicBool(a.Mechanics, "ON_ACTION", "EATING_LOSS_BLESSING", "heal_in_primitive", false) || mechanicBool(actor.Mechanics, "COMBAT_PASSIVE", "EATING_LOSS_BLESSING", "heal_in_primitive", false)
		if healGain > 0 && !healInPrimitive {
			actor.HP = math.Min(actor.MaxHP, actor.HP+healGain)
		}
		logInPrimitive := mechanicBool(a.Mechanics, "ON_ACTION", "EATING_LOSS_BLESSING", "log_in_primitive", false) || mechanicBool(actor.Mechanics, "COMBAT_PASSIVE", "EATING_LOSS_BLESSING", "log_in_primitive", false)
		if !logInPrimitive {
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "EATING_LOSS_BLESSING",
				Detail:    fmt.Sprintf("[Thực Đạo - Cật Khuy Thị Phúc] Chuyển hóa %d tầng thiệt thòi thành phúc báo! Tăng %.0f%% sát thương và hồi phục %.0f HP!", stacks, (mult-1.0)*100, healGain),
			})
		}
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		opponents := r.state.Enemies
		if !isAlly {
			opponents = r.state.Allies
		}
		for _, opp := range opponents {
			if opp != nil && opp.HP > 0 {
				opp.ActiveEffects = append(opp.ActiveEffects, Effect{
					ID: statusfx.StatusStun, Duration: 1, Type: "debuff",
				})
				opp.IsStunned = true
			}
		}
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "SOUL_BEAST_SUMMON",
			Detail:    fmt.Sprintf("[Hồn Đạo - Hồn Thú Triệu Lai] Triệu hoán Thái Cổ Hồn Thú nhập thể hộ vệ! Nhận %.0f Hộ Thuẫn Hồn Thú và tăng 30%% Sát Thương!", shieldGain),
		})
	}

	// Thâu Đạo: Thâu Sinh (STEAL_LIFE)
	r.applyOperationResourceTransfers(combatPhaseContext{actor: actor, action: a, firstTarget: firstTarget}, "LATE_RIDER")
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		team := r.state.Allies
		opponents := r.state.Enemies
		if !isAlly {
			team = r.state.Enemies
			opponents = r.state.Allies
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
					ID: statusfx.StatusBlind, Value: 35, Duration: 3, Type: "debuff",
				})
			}
		}
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "CHARM_GHOST_COME",
			TargetId:  firstTarget.ID,
			Detail:    fmt.Sprintf("[Mị Đạo - Tử Quỷ Khoái Lai] Tiếng gọi ma mị quyến rũ mê hồn! Áp đặt Mê Hoặc khiến %s thần hồn điên đảo trong 2 hiệp!", firstTarget.Name),
		})
	}

	// EARLY_RIDER resource transfers execute at the authored pre-damage position.
	r.applyOperationResourceTransfers(combatPhaseContext{actor: actor, action: a, firstTarget: firstTarget}, "EARLY_RIDER")

	// Thực Đạo: Thôn Thiên Địa (DEVOUR_HEAVEN_EARTH), legacy persisted-hook fallback.
	if (hasMechanic(a.Mechanics, "ON_ACTION", "DEVOUR_HEAVEN_EARTH") || hasMechanic(a.Mechanics, "ON_HIT", "DEVOUR_HEAVEN_EARTH")) && firstTarget != nil {
		a.Damage *= 1.50
		operationDamageScale *= 1.50
		devouredShield := firstTarget.VirtualShield * 0.50
		firstTarget.VirtualShield -= devouredShield
		actor.HP = math.Min(actor.MaxHP, actor.HP+devouredShield)
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
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
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "FIVE_REGIONAL_SUPPRESSION",
			Detail:    fmt.Sprintf("[Ngũ Giới Áp Chế] Giới Vực Bích phản phệ kịch liệt khi thôi động sát chiêu, %s tự tổn %.0f HP!", actor.Name, selfHarm),
		})
	}

	// A normal Gu skill is paid, checked, and counted once by this parent
	// action. Its Block v2 operations retain their own target and conditions
	// and are each applied by the simulator exactly once here.
	if a.ActionType == "GU_SKILL" && hasNonCastOperationEffects(a) {
		for _, effect := range a.OperationEffects {
			if effect.Compiled.Trigger == gameplayeffects.TriggerOnCast {
				continue // Cast-phase operations ran at the ON_CAST dispatcher position.
			}
			if effect.Action == nil {
				continue
			}
			child := *effect.Action
			if len(a.PreAppliedEffectIDs) > 0 {
				preChainModifiers := make([]PhaseDamageModifier, 0, len(child.PreChainDamageModifiers))
				for _, modifier := range child.PreChainDamageModifiers {
					if !a.PreAppliedEffectIDs[modifier.EffectID] {
						preChainModifiers = append(preChainModifiers, modifier)
					}
				}
				child.PreChainDamageModifiers = preChainModifiers
				statuses := make([]statusfx.StatusOp, 0, len(child.StatusOps))
				for _, status := range child.StatusOps {
					if !a.PreAppliedEffectIDs[status.EffectID] {
						statuses = append(statuses, status)
					}
				}
				child.StatusOps = statuses
				notifications := make([]CombatNotification, 0, len(child.CombatNotifications))
				for _, notification := range child.CombatNotifications {
					if !a.PreAppliedEffectIDs[notification.EffectID] {
						notifications = append(notifications, notification)
					}
				}
				child.CombatNotifications = notifications
			}
			child.StatusDetonations = nil // The parent consumed these at the native hook's historical phase.
			child.HPCosts = nil           // The parent charged these at the native hook's historical phase.
			child.IsEffectOperation = true
			child.OperationConditions = append([]gameplayeffects.Condition(nil), effect.Compiled.Conditions...)
			child.Rank = effect.SourceRank
			child.SourceID = effect.SourceGuID
			child.InstanceID = effect.SourceInstanceID
			if child.InstanceID == "" {
				child.InstanceID = a.InstanceID
			}
			if child.SourceID == "" {
				child.SourceID = a.SourceID
			}
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
			out := r.runOne(actor, &child, isAlly, enemyTargets)
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
	r.dispatchPhase(combatPhasePreDamage, combatPhaseContext{actor: actor, action: a, firstTarget: firstTarget, targets: resolved})
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
			counterThresholdHitRule(actor, "bypass_shield")
		hitCount := a.MultiHit
		if hitCount < 1 {
			hitCount = 1
		}

		// Targets whose strike was negated by FIRST_TURN_EVADE carry
		// no percent riders (damage_hp_percent/shield strip below).
		evadedThisStrike := make(map[*CombatantAuto]bool)
		for _, target := range resolved {
			totalDmg := 0.0
			hadEvaded := target.CustomState != nil && target.CustomState["first_turn_evade_used"] == r.turn

			// Evasion: dodge chance from the target's EVASION status.
			satKhiGuaranteed := counterThresholdHitRule(actor, "guaranteed")
			sureHit := satKhiGuaranteed || isGuaranteedHit(actor, a)
			if missCh := target.EvasionChance; missCh > 0 && r.rng.Float64() < missCh && !sureHit {
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:       r.turn,
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
				if r.rng.Float64() < actor.AccuracyPenalty {
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:       r.turn,
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
			raceDamageBonus := gamefx.RaceModifier(target.Race, actor.RaceDamageBonus)
			raceResistance := gamefx.RaceModifier(actor.Race, target.RaceResistance)

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
				hitDmg := a.Damage * skillMult * dmgMultiplier * daoMult * envMod.PowerMult * conflictPenaltyMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path)) * (1.0 + raceDamageBonus) * targetResonance
				if h > 0 {
					hitDmg *= 0.7
				}
				// Critical hit check
				if actor.CritRateBonus > 0 && r.rng.Float64() < actor.CritRateBonus {
					hitDmg *= 1.5
					isCritHit = true
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:      r.turn,
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
				netDmg *= (1.0 - raceResistance)
				netDmg = ResolveBaseDamage(netDmg, 0)
				totalHitDmg += netDmg
				totalDmg += applyDamageToTarget(r.state, actor, target, netDmg, r.turnLogs, r.turn, shieldBypass, sureHit, !isAlly, r.rng)
			}
			if !hadEvaded && target.CustomState != nil && target.CustomState["first_turn_evade_used"] == r.turn {
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

			rawDmg := a.Damage * skillMult * dmgMultiplier * daoMult * conflictPenaltyMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path)) * (1.0 + raceDamageBonus) * targetResonance * critMultVal

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
				RaceDamageBonus:    raceDamageBonus,
				TargetDef:          targetDef,
				NaturalResist:      natResFinal,
				PathResist:         pathBonus(target.PathResistance, a.Path),
				RaceResistance:     raceResistance,
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
			if raceDamageBonus != 0 {
				breakdown.RaceDamageTarget = target.Race
			}
			if raceResistance != 0 {
				breakdown.RaceResistAttacker = actor.Race
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
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:       r.turn,
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
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:       r.turn,
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
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:       r.turn,
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
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:       r.turn,
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
				phantomRaceDamage := gamefx.RaceModifier(target.Race, actor.RaceDamageBonus)
				phantomRaceResistance := gamefx.RaceModifier(actor.Race, target.RaceResistance)
				phantomDef := target.BaseDef
				phantomNet := (a.PhantomDmg * dmgMultiplier * (1 + phantomRaceDamage) * phantomResonance) - phantomDef
				phantomNet *= (1 - phantomResist)
				phantomNet *= (1.0 - pathBonus(target.PathResistance, a.Path))
				phantomNet *= (1.0 - phantomRaceResistance)
				phantomNet = math.Max(1, phantomNet)
				phantomDealt := applyDamageToTarget(r.state, actor, target, phantomNet, r.turnLogs, r.turn, shieldBypass, sureHit, !isAlly, r.rng)
				if phantomDealt > 0 && phantomResist > 0 {
					applyForeignDaoWound(target)
				}
				totalDamageDealt += phantomDealt
				phantomBreakdown := &DamageBreakdown{
					BaseDamage: a.PhantomDmg, DaoMult: 1, TargetDaoResonance: phantomResonance,
					DaoPath: phantomPath, NaturalResist: phantomResist,
					PathResist:      pathBonus(target.PathResistance, a.Path),
					RaceDamageBonus: phantomRaceDamage, RaceResistance: phantomRaceResistance,
					TargetDef: phantomDef, FinalDamage: phantomNet,
				}
				if phantomRaceDamage != 0 {
					phantomBreakdown.RaceDamageTarget = target.Race
				}
				if phantomRaceResistance != 0 {
					phantomBreakdown.RaceResistAttacker = actor.Race
				}
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   actor.ID,
					ActorName: actor.Name,
					Action:    "PHANTOM_ATTACK",
					TargetId:  target.ID,
					Damage:    phantomDealt,
					Breakdown: phantomBreakdown,
					Detail:    fmt.Sprintf("Hư ảnh của %s tấn công thêm %.0f sát thương lên %s!", actor.Name, phantomDealt, target.Name),
				})
			}

			// ── Freeze break: if target is frozen and takes damage, 50% chance to break ──
			if target.IsFrozen && totalDmg > 0 {
				if r.rng.Float64() < 0.50 {
					var unfrozenKept []Effect
					for _, eff := range target.ActiveEffects {
						if eff.ID == EffectFreeze {
							(*r.turnLogs) = append((*r.turnLogs), CombatLog{
								Turn:      r.turn,
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
			(*r.turnLogs) = append((*r.turnLogs), *shieldLog)
		}
		if healLog != nil {
			(*r.turnLogs) = append((*r.turnLogs), *healLog)
		}
		if manaLog != nil {
			(*r.turnLogs) = append((*r.turnLogs), *manaLog)
		}
		if cleanseLog != nil {
			(*r.turnLogs) = append((*r.turnLogs), *cleanseLog)
		}

		// ── Multi-hit log ──
		if hitCount > 1 && len(resolved) > 0 {
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    "MULTI_HIT",
				TargetId:  resolved[0].ID,
				Detail:    fmt.Sprintf("%s trúng %d đòn liên hoàn!", resolved[0].Name, hitCount),
			})
		}

		if a.ThornsDmg > 0 && totalDamageDealt > 0 {
			thornsNet := math.Max(1, a.ThornsDmg-float64(actor.BaseDef))
			applyDamageToTarget(r.state, firstTarget, actor, thornsNet, r.turnLogs, r.turn, false, false, false, r.rng)
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
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
				thornsNet := ApplyThornsDamage(target, actor, float64(totalDamageDealt), r.turn, r.turnLogs)
				_ = thornsNet
			}
		}

		if a.LifeStealPct > 0 && totalDamageDealt > 0 {
			healAmount := totalDamageDealt * a.LifeStealPct
			hpNeeded := actor.MaxHP - actor.HP
			if healAmount <= hpNeeded {
				actor.HP += healAmount
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
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
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
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

		// ── Percent-based direct HP effects ──
		if a.DamageHPPercent > 0 {
			for _, target := range resolved {
				if target == nil || target.HP <= 0 {
					continue
				}
				if evadedThisStrike[target] {
					continue
				}
				if a.DamageHPPercent > 0 && isImmuneToDamage(target, satKhiPiercesImmunity(actor)) {
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:      r.turn,
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
						(*r.turnLogs) = append((*r.turnLogs), CombatLog{
							Turn:      r.turn,
							ActorId:   actor.ID,
							ActorName: actor.Name,
							Action:    "HP_PERCENT",
							TargetId:  target.ID,
							Damage:    direct,
							Detail:    hpPctDetail,
						})
					} else {
						(*r.turnLogs) = append((*r.turnLogs), CombatLog{
							Turn:      r.turn,
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
		if len(a.CombatNotifications) == 0 {
			(*r.turnLogs) = append((*r.turnLogs), CombatLog{
				Turn:      r.turn,
				ActorId:   actor.ID,
				ActorName: actor.Name,
				Action:    a.ActionType,
				TargetId:  targetID,
				Detail:    detailMsg,
			})
		}

		if shieldLog != nil {
			(*r.turnLogs) = append((*r.turnLogs), *shieldLog)
		}
		if healLog != nil {
			(*r.turnLogs) = append((*r.turnLogs), *healLog)
		}
		if manaLog != nil {
			(*r.turnLogs) = append((*r.turnLogs), *manaLog)
		}
		if cleanseLog != nil {
			(*r.turnLogs) = append((*r.turnLogs), *cleanseLog)
		}
	}

	// ── Apply status effects + self-buffs (always, no damage required) ──
	// ProcChance gates the whole bonus package (status ops, self
	// evasion, target weak, Gu seal); base damage already landed.
	procOK := a.ProcChance <= 0 || r.rng.Float64() < a.ProcChance
	if !procOK {
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn:      r.turn,
			ActorId:   actor.ID,
			ActorName: actor.Name,
			Action:    "PROC_FAIL",
			Detail:    fmt.Sprintf("[%s] không kích hoạt hiệu ứng bổ sung!", a.Name),
		})
	} else {
		allOps := collectAllStatusOps(a)
		allOps = postDamageStatusOps(allOps)
		actionRank := a.Rank
		if actionRank <= 0 {
			actionRank = actor.Rank
		}
		if len(allOps) > 0 {
			for i := range allOps {
				allOps[i].SourcePath = daoPathForAction(a.Path)
			}
			applyStatusOps(actor, resolved, allOps, actionRank, r.turn, r.rng, r.turnLogs)
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
				applyStatusOps(actor, resolved, []statusfx.StatusOp{{ID: statusfx.StatusWeak, Value: a.TargetWeakPct * 100, Turns: turns}}, actionRank, r.turn, r.rng, r.turnLogs)
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
						(*r.turnLogs) = append((*r.turnLogs), CombatLog{
							Turn:      r.turn,
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
				if immuneToHostileEffects(target) {
					(*r.turnLogs) = append((*r.turnLogs), CombatLog{
						Turn:      r.turn,
						ActorId:   target.ID,
						ActorName: target.Name,
						Action:    "REVERSE_NEGATE",
						Detail:    fmt.Sprintf("Nghịch Chuyển Cổ đảo nghịch phong ấn Cổ của [%s] về hư vô!", a.Name),
					})
					continue
				}
				sealGuTurns(target, turns)
				(*r.turnLogs) = append((*r.turnLogs), CombatLog{
					Turn:      r.turn,
					ActorId:   target.ID,
					ActorName: target.Name,
					Action:    "GU_SEAL",
					Detail:    fmt.Sprintf("%s bị phong ấn Cổ trong %d lượt, không thể dùng Cổ!", target.Name, turns),
				})
			}
		}
	}
	for _, notification := range a.CombatNotifications {
		if notification.Phase != "" {
			continue // phase-bound notifications were emitted at their dispatcher seam
		}
		(*r.turnLogs) = append((*r.turnLogs), CombatLog{
			Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name,
			Action: notification.Action, Detail: notification.Message,
		})
	}
	// Merge action logs into a single consolidated log entry for this Gu/Action
	// Fast path: when the action produced a single log it already is its own
	// consolidated entry (the merge would only re-assign the same value back),
	// so skip the slice copy + consolidation scan for the common case.
	if len((*r.turnLogs))-actionStartIndex > 1 {
		logs := make([]CombatLog, len((*r.turnLogs))-actionStartIndex)
		copy(logs, (*r.turnLogs)[actionStartIndex:])

		var useLog *CombatLog
		var costLog *CombatLog
		var gainLog *CombatLog
		var resultLogs []CombatLog
		var preservedLogs []CombatLog
		var extraBreakdown *DamageBreakdown
		var foldedSettlement *SettlementEffect

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
				// Folded logs disappear, so carry their typed post-combat
				// settlement onto the surviving action entry; otherwise a
				// Love Gu cost would be lost with the log it came from.
				// Passives are logged before the "used [X]" entry, so this
				// cannot attach directly here.
				if log.Settlement != nil && foldedSettlement == nil {
					foldedSettlement = log.Settlement
				}
				resultLogs = append(resultLogs, *log)
			}
		}

		if useLog != nil {
			if useLog.Settlement == nil {
				useLog.Settlement = foldedSettlement
			}
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
			(*r.turnLogs) = append((*r.turnLogs)[:actionStartIndex], *useLog)
			(*r.turnLogs) = append((*r.turnLogs), preservedLogs...)
		}
	}
	return res
}

func resolveEnvModifiersForAction(state *CombatState, actor *CombatantAuto, paths []string) gamefx.EnvSkillModifiers {
	if state == nil || state.EnvDao == nil || actor == nil {
		return gamefx.EnvSkillModifiers{PowerMult: 1.0, ManaTaxMult: 1.0}
	}
	isAlly := false
	if state.EnvDao.DomainOwnerID != "" && actor.ID != state.EnvDao.DomainOwnerID {
		for _, a := range state.Allies {
			if a != nil && a.ID == state.EnvDao.DomainOwnerID {
				// Same side as the domain owner
				for _, al := range state.Allies {
					if al == actor {
						isAlly = true
						break
					}
				}
				break
			}
		}
	}
	return gamefx.ResolveEnvSkillModifiers(
		gamefx.DefaultEnvDaoConfig,
		state.EnvDao,
		daoPathForAction(paths),
		actor.ID,
		actor.Rank,
		isAlly,
		actor.HasNonConflictingDao,
	)
}
