package composition

import (
	"math"
	"reflect"
	"strings"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

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
