package composition

import (
	"math"
	"reflect"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

// SupportBudgetInput describes one support Gu's rank and path affinity for Killer Move budget calculation.
type SupportBudgetInput struct {
	Rank         int
	PathAffinity float64
}

// KillerMovePowerBudget computes the unified Power Budget of a Killer Move from its core Gu rank
// and support Gu components using diminishing support efficiency (0.60 / sqrt(i)).
func KillerMovePowerBudget(mainRank int, supports []SupportBudgetInput) float64 {
	budget := effectcatalog.PowerBudget(mainRank)
	for i, sup := range supports {
		affinity := sup.PathAffinity
		if affinity <= 0 {
			affinity = 0.70
		}
		eff := 0.60 / math.Sqrt(float64(i+1))
		budget += effectcatalog.PowerBudget(sup.Rank) * eff * affinity
	}
	return math.Round(budget*100) / 100
}

// GuHousePowerBudget computes the unified Power Budget of an Immortal Gu House from its core Gu ranks
// (each contributing 85%) and mortal auxiliary Gu count (up to +25% bonus).
func GuHousePowerBudget(coreRanks []int, auxGuCount int) float64 {
	coreSum := 0.0
	for _, r := range coreRanks {
		coreSum += effectcatalog.PowerBudget(r) * 0.85
	}
	auxBonus := 0.0
	if auxGuCount > 0 {
		// Diminishing log curve reaching +25% at MaxAuxGuCount (100,000).
		auxBonus = math.Min(0.25, 0.25*math.Log10(1+float64(auxGuCount))/5.0)
	}
	return math.Round(coreSum*(1.0+auxBonus)*100) / 100
}

// ToPowerBlockInputs converts gameplayeffects.Block slices into effectcatalog.PowerBlockInput slices.
func ToPowerBlockInputs(blocks []gameplayeffects.Block) []effectcatalog.PowerBlockInput {
	out := make([]effectcatalog.PowerBlockInput, 0, len(blocks))
	for _, b := range blocks {
		conds := make([]any, len(b.Conditions))
		for i, c := range b.Conditions {
			conds[i] = c
		}
		ops := make([]effectcatalog.PowerOpInput, 0, len(b.Operations))
		for _, o := range b.Operations {
			var target *effectcatalog.PowerTargetInput
			if o.Target != nil {
				target = &effectcatalog.PowerTargetInput{
					Selector:        string(o.Target.Selector),
					BypassFrontline: o.Target.BypassFrontline,
				}
			}
			var chance *float64
			if o.Chance != 0 || o.Op == gameplayeffects.OpStatus {
				c := o.Chance
				chance = &c
			}
			ops = append(ops, effectcatalog.PowerOpInput{
				Op:      string(o.Op),
				SubType: o.SubType,
				Unit:    o.Unit,
				Status:  o.StatusID,
				Target:  target,
				Value:   o.Value,
				Turns:   o.Turns,
				Chance:  chance,
			})
		}
		out = append(out, effectcatalog.PowerBlockInput{
			ID:      b.ID,
			Trigger: string(b.Trigger),
			Target: effectcatalog.PowerTargetInput{
				Selector:        string(b.Target.Selector),
				BypassFrontline: b.Target.BypassFrontline,
			},
			Conditions: conds,
			Operations: ops,
		})
	}
	return out
}

// EffectLoad calculates the power load of an authored effect operation based on
// its opcode, value scaling, target selector, and duration/chance.
func EffectLoad(op gameplayeffects.Operation, target gameplayeffects.Target) float64 {
	base := 16.0
	switch op.Op {
	case gameplayeffects.OpDamage, gameplayeffects.OpDirectHP:
		base = 24
	case gameplayeffects.OpStatus:
		base = 20
	case gameplayeffects.OpShield, gameplayeffects.OpBarrier:
		base = 18
	case gameplayeffects.OpHeal:
		base = 17
	case gameplayeffects.OpCleanse:
		base = 12
	case gameplayeffects.OpStatMod:
		base = 14
	}
	valueFactor := 1 + math.Min(2.5, math.Log10(1+math.Abs(op.Value))/5)
	scope := strings.ToUpper(string(target.Selector))
	switch scope {
	case "ALL", "ALL_ENEMIES", "ALL_ALLIES":
		valueFactor *= 1.8
	case "ROW", "COL":
		valueFactor *= 1.35
	}
	if op.Op == gameplayeffects.OpStatus {
		valueFactor *= 1 + math.Min(0.6, float64(max(0, op.Turns-1))*0.12)
		chance := op.Chance
		if chance <= 0 {
			chance = 1
		}
		valueFactor *= 0.75 + math.Min(1, chance)*0.5
	}
	return base * valueFactor
}

// ScaleOperation scales an effect operation's value, chance, and duration by a harmony/power factor.
func ScaleOperation(op *gameplayeffects.Operation, scale float64) {
	if op == nil || scale >= 1 {
		return
	}
	if op.Op == gameplayeffects.OpStatus {
		chance := op.Chance
		if chance <= 0 {
			chance = 1
		}
		op.Chance = math.Max(0.05, chance*scale)
		op.Value *= scale
		if op.Turns > 1 {
			op.Turns = max(1, int(math.Ceil(float64(op.Turns)*scale)))
		}
		return
	}
	if op.Op == gameplayeffects.OpCleanse {
		return
	}
	if (op.Op == gameplayeffects.OpDamage || op.Op == gameplayeffects.OpShield) && strings.EqualFold(op.SubType, "MULTIPLIER") {
		op.Value = 1 + (op.Value-1)*scale
		return
	}
	op.Value *= scale
}

// SameOperationSemantics compares whether two operations share equivalent semantic fields.
func SameOperationSemantics(snapshot, authored gameplayeffects.Operation) bool {
	if snapshot.Target != nil {
		return false
	}
	return snapshot.Op == authored.Op && snapshot.SubType == authored.SubType && snapshot.Path == authored.Path &&
		snapshot.StatusID == authored.StatusID && snapshot.Stat == authored.Stat && snapshot.Unit == authored.Unit &&
		snapshot.Resource == authored.Resource && snapshot.StackPolicy == authored.StackPolicy && snapshot.Code == authored.Code &&
		reflect.DeepEqual(snapshot.Params, authored.Params)
}

// AdjustedSuccessChance computes runtime execution success chance factoring in creation mastery,
// current mastery level, and accumulated mastery experience.
func AdjustedSuccessChance(base float64, creationLevel, currentLevel, xp int) float64 {
	chance := base + float64(currentLevel-creationLevel)*0.035 + math.Min(.15, float64(xp)/1000*.15)
	if chance > .95 {
		return .95
	}
	if chance < .05 {
		return .05
	}
	return chance
}
