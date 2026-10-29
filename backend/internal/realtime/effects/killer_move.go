package effects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"

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

func compileKillerMove(move KillerMove, slot int) (combat.Skill, bool) {
	if move.ID == "" || move.Name == "" || move.Rank < 1 || move.Rank > 10 ||
		move.ExpectedComponents < 1 || len(move.Components) != move.ExpectedComponents ||
		len(move.Effects) != 1 || move.SuccessChance <= 0 || move.SuccessChance > 1 ||
		move.BackfireStatus != "" || !finite(move.ManaCost) || move.ManaCost < 0 {
		return combat.Skill{}, false
	}
	block := move.Effects[0]
	if block.Trigger != gameplayeffects.TriggerOnAction || len(block.Conditions) != 0 || len(block.Operations) != 1 {
		return combat.Skill{}, false
	}
	op := block.Operations[0]
	target := block.Target
	if op.Target != nil {
		target = *op.Target
	}
	skill := combat.Skill{
		ID: "KILLER_MOVE_" + move.ID, Slot: slot, Name: move.Name, ManaCost: move.ManaCost,
		SuccessChance: move.SuccessChance, BackfireHPPercent: move.BackfireHPPercent,
		Range: 9, MaxTargets: 1, RequiresEnemyTarget: true,
	}
	if !move.HouseStrike {
		skill.KillerMoveID = move.ID
	}
	if move.DamageScale > 0 {
		skill.Damage = move.DamageScale
	} else {
		skill.Damage = 1
	}
	minHunger := math.MaxFloat64
	for _, component := range move.Components {
		if component.InstanceID == "" || !finite(component.Hunger) || component.Hunger < 1 {
			return combat.Skill{}, false
		}
		skill.GuComponentIDs = append(skill.GuComponentIDs, component.InstanceID)
		if component.Hunger < minHunger {
			minHunger = component.Hunger
		}
	}
	skill.GuHunger = minHunger

	switch op.Op {
	case gameplayeffects.OpDamage:
		if target.Selector != gameplayeffects.TargetSingle || !finite(op.Value) || op.Value <= 0 || op.Turns != 0 {
			return combat.Skill{}, false
		}
		skill.Mode = combat.ModeTarget
		skill.Damage = op.Value * skill.Damage
	case gameplayeffects.OpHeal:
		if target.Selector != gameplayeffects.TargetSelf || !finite(op.Value) || op.Value <= 0 || op.Turns != 0 || op.SubType == "PERCENT" {
			return combat.Skill{}, false
		}
		skill.Mode = combat.ModeSelf
		skill.RequiresEnemyTarget = false
		skill.Range = 0
		skill.Heal = op.Value
	case gameplayeffects.OpShield:
		if target.Selector != gameplayeffects.TargetSelf || !finite(op.Value) || op.Value <= 0 || op.Turns != 0 {
			return combat.Skill{}, false
		}
		skill.Mode = combat.ModeSelf
		skill.RequiresEnemyTarget = false
		skill.Range = 0
		skill.Shield = op.Value
	case gameplayeffects.OpCleanse:
		if target.Selector != gameplayeffects.TargetSelf || op.Turns != 0 {
			return combat.Skill{}, false
		}
		skill.Mode = combat.ModeSelf
		skill.RequiresEnemyTarget = false
		skill.Range = 0
		skill.Cleanse = true
	default:
		return combat.Skill{}, false
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
		return combat.Skill{}, false
	}
	return skill, true
}

func millisecondsToTicks(milliseconds uint64) uint64 {
	return (milliseconds*sim.TickRate + 999) / 1000
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
