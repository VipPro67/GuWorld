package effects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

const (
	KillerMoveFirstSlot = 9
	KillerMoveLastSlot  = 12
)

type KillerMoveComponent struct {
	InstanceID string
	Hunger     float64
}

type KillerMove struct {
	ProfileVersion     int
	ID                 string
	Name               string
	Rank               int
	Components         []KillerMoveComponent
	ExpectedComponents int
	Effects            []gameplayeffects.Block
	ManaCost           float64
	SuccessChance      float64
	BackfireHPPercent  float64
	BackfireStatus     string
	Complexity         float64
	DamageScale        float64
	WindupMS           uint64
	CooldownMS         uint64
	// HouseStrike marks Trận Kích synthesized from an equipped Tiên Cổ Ốc.
	// House actions drain core-Gu hunger but never accrue killer-move mastery.
	HouseStrike bool
}

// BuildKillerMoveSkills projects at most four equipped moves into reserved
// slots 9–12. The compiler is deliberately strict: no turn-duration effects,
// conditional blocks, mixed operations, or unauthored target semantics enter
// realtime combat.
func BuildKillerMoveSkills(moves []KillerMove) []combat.Skill {
	limit := len(moves)
	if limit > KillerMoveLastSlot-KillerMoveFirstSlot+1 {
		limit = KillerMoveLastSlot - KillerMoveFirstSlot + 1
	}
	skills := make([]combat.Skill, 0, limit)
	for index, move := range moves[:limit] {
		if skill, ok := compileKillerMove(move, KillerMoveFirstSlot+index); ok {
			skills = append(skills, skill)
		}
	}
	return skills
}

// AppendKillerMoveLoadout adds supported moves and folds their authoritative
// source data into the session revision used for cooldown recovery.
func AppendKillerMoveLoadout(skills []combat.Skill, revision string, moves []KillerMove) ([]combat.Skill, string) {
	compiled := BuildKillerMoveSkills(moves)
	hash := sha256.New()
	_, _ = hash.Write([]byte(revision))
	for _, move := range moves {
		raw, _ := json.Marshal(move)
		_, _ = hash.Write(raw)
		_, _ = hash.Write([]byte{0})
	}
	return append(skills, compiled...), hex.EncodeToString(hash.Sum(nil))
}

// RealtimeOperationAllowed reports whether an individual operation is executable in realtime combat.
func RealtimeOperationAllowed(op gameplayeffects.Operation, target gameplayeffects.Target) bool {
	if op.Turns != 0 {
		return false
	}
	switch op.Op {
	case gameplayeffects.OpDamage:
		return target.Selector == gameplayeffects.TargetSingle && finite(op.Value) && op.Value > 0
	case gameplayeffects.OpHeal:
		return target.Selector == gameplayeffects.TargetSelf && finite(op.Value) && op.Value > 0 && !strings.EqualFold(op.SubType, "PERCENT")
	case gameplayeffects.OpShield:
		return target.Selector == gameplayeffects.TargetSelf && finite(op.Value) && op.Value > 0
	case gameplayeffects.OpCleanse:
		return target.Selector == gameplayeffects.TargetSelf
	default:
		return false
	}
}

// CompileKillerMoveWithDiagnostic compiles a single KillerMove into a combat.Skill,
// returning the compiled skill, a diagnostic reason string if compilation failed, and a success boolean.
func CompileKillerMoveWithDiagnostic(move KillerMove, slot int) (combat.Skill, string, bool) {
	if move.ID == "" || move.Name == "" {
		return combat.Skill{}, "ID hoặc tên sát chiêu không hợp lệ", false
	}
	if move.Rank < 1 || move.Rank > 10 {
		return combat.Skill{}, "Rank sát chiêu không hợp lệ", false
	}
	if move.ExpectedComponents < 1 || len(move.Components) != move.ExpectedComponents {
		return combat.Skill{}, "Số lượng thành phần Cổ trùng không khớp", false
	}
	if move.SuccessChance <= 0 || move.SuccessChance > 1 {
		return combat.Skill{}, "Tỉ lệ thành công không hợp lệ", false
	}
	if move.BackfireStatus != "" {
		return combat.Skill{}, "Trạng thái phản phệ không hợp lệ trong realtime", false
	}
	if !finite(move.ManaCost) || move.ManaCost < 0 {
		return combat.Skill{}, "Hao tổn chân nguyên không hợp lệ", false
	}

	skill := combat.Skill{
		ID: "KILLER_MOVE_" + move.ID, Slot: slot, Name: move.Name, ManaCost: move.ManaCost,
		SuccessChance: move.SuccessChance, BackfireHPPercent: move.BackfireHPPercent,
		Range: 9, MaxTargets: 1,
	}
	if !move.HouseStrike {
		skill.KillerMoveID = move.ID
	}
	damageScale := move.DamageScale
	if damageScale <= 0 {
		damageScale = 1
	}
	minHunger := math.MaxFloat64
	for _, component := range move.Components {
		if component.InstanceID == "" || !finite(component.Hunger) || component.Hunger < 1 {
			return combat.Skill{}, "Thành phần Cổ trùng chưa đủ độ no hoặc không hợp lệ", false
		}
		skill.GuComponentIDs = append(skill.GuComponentIDs, component.InstanceID)
		if component.Hunger < minHunger {
			minHunger = component.Hunger
		}
	}
	skill.GuHunger = minHunger
	if move.ProfileVersion < 2 {
		if len(move.Effects) != 1 || move.Effects[0].Trigger != gameplayeffects.TriggerOnAction {
			return combat.Skill{}, "Profile version 1 yêu cầu đúng 1 effect ON_ACTION", false
		}
	}

	hasRealtimeAction := false
	hasTargetEffect, hasSelfEffect, hasCleanse := false, false, false
	for _, block := range move.Effects {
		// ON_USE remains an out-of-combat action and is deliberately omitted
		// from the realtime profile. Every ON_ACTION block must be executable.
		if block.Trigger != gameplayeffects.TriggerOnAction {
			continue
		}
		hasRealtimeAction = true
		if len(block.Conditions) != 0 {
			return combat.Skill{}, "Effect có điều kiện chưa có executor realtime", false
		}
		if len(block.Operations) != 1 {
			return combat.Skill{}, "Mỗi khối hiệu ứng realtime chỉ được chứa đúng 1 thao tác", false
		}
		op := block.Operations[0]
		target := block.Target
		if op.Target != nil {
			target = *op.Target
		}
		if !RealtimeOperationAllowed(op, target) {
			return combat.Skill{}, "Effect chưa có executor realtime hoặc còn thời lượng theo lượt", false
		}
		switch op.Op {
		case gameplayeffects.OpDamage:
			hasTargetEffect = true
			skill.Damage += op.Value * damageScale
		case gameplayeffects.OpHeal:
			hasSelfEffect = true
			skill.Heal += op.Value * damageScale
		case gameplayeffects.OpShield:
			hasSelfEffect = true
			skill.Shield += op.Value * damageScale
		case gameplayeffects.OpCleanse:
			hasSelfEffect = true
			hasCleanse = true
			skill.Cleanse = true
		default:
			return combat.Skill{}, "Thao tác không được hỗ trợ trong realtime", false
		}
	}
	if !hasRealtimeAction {
		return combat.Skill{}, "Cần ít nhất một effect ON_ACTION để thi triển realtime", false
	}
	if (!hasTargetEffect && !hasSelfEffect) || (hasCleanse && hasTargetEffect) {
		if hasCleanse && hasTargetEffect {
			return combat.Skill{}, "Thanh tẩy chỉ có thể ghép với các effect SELF, không ghép với damage mục tiêu", false
		}
		return combat.Skill{}, "Cần ít nhất một effect ON_ACTION để thi triển realtime", false
	}
	if hasTargetEffect {
		skill.Mode = combat.ModeTarget
		skill.RequiresEnemyTarget = true
	} else {
		skill.Mode = combat.ModeSelf
		skill.Range = 0
	}

	// Realtime windup and cooldown are authored here from rank/complexity, not
	// derived from the retired warmup_turns/cooldown_turns counters.
	complexity := move.Complexity
	if !finite(complexity) || complexity < 1 {
		complexity = 1
	}
	windupMS, cooldownMS := move.WindupMS, move.CooldownMS
	if windupMS == 0 || cooldownMS == 0 {
		windupMS, cooldownMS = models.KillerMoveRealtimeTiming(move.Rank, complexity)
	}
	skill.WindupTicks = millisecondsToTicks(windupMS)
	skill.CooldownTicks = millisecondsToTicks(cooldownMS)
	if !skill.Valid() {
		return combat.Skill{}, "Skill cấu hình không hợp lệ với hệ thống combat", false
	}
	return skill, "", true
}

func compileKillerMove(move KillerMove, slot int) (combat.Skill, bool) {
	skill, _, ok := CompileKillerMoveWithDiagnostic(move, slot)
	return skill, ok
}

// ValidateKillerMoveSupport checks whether a KillerMove is supported in realtime combat,
// ignoring transient hunger so starving components do not block hotbar reordering or previews.
func ValidateKillerMoveSupport(move KillerMove) (bool, string) {
	if len(move.Components) == 0 {
		return false, "Cổ thành phần không khả dụng hoặc đã kiệt sức"
	}
	hasRealtimeEffect, hasOutsideCombatEffect := false, false
	for _, block := range move.Effects {
		hasRealtimeEffect = hasRealtimeEffect || block.Trigger == gameplayeffects.TriggerOnAction
		hasOutsideCombatEffect = hasOutsideCombatEffect || block.Trigger == gameplayeffects.TriggerOnUse
	}
	if !hasRealtimeEffect && hasOutsideCombatEffect {
		return false, "Sát chiêu này chỉ dùng ngoài combat"
	}
	probe := move
	probe.Components = make([]KillerMoveComponent, 0, len(move.Components))
	for _, component := range move.Components {
		hunger := component.Hunger
		if hunger < 1 {
			hunger = 1
		}
		probe.Components = append(probe.Components, KillerMoveComponent{
			InstanceID: component.InstanceID, Hunger: hunger,
		})
	}
	if probe.DamageScale <= 0 {
		probe.DamageScale = 1
	}
	_, reason, ok := CompileKillerMoveWithDiagnostic(probe, KillerMoveFirstSlot)
	return ok, reason
}

func millisecondsToTicks(milliseconds uint64) uint64 {
	return (milliseconds*sim.TickRate + 999) / 1000
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
