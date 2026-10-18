package match

import (
	"math"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// StatusTicksPerTurn converts a turn-based status duration into realtime ticks.
// A turn is roughly two seconds, which keeps authored durations meaningful
// without re-authoring Status.csv.
const StatusTicksPerTurn = 40

// BasicAttackSkill returns the universal basic attack available to all fighters.
// It costs 0 mana, has a fast 0.6s cooldown, and delivers basic physical/spiritual strike damage.
func BasicAttackSkill(attack float64) Skill {
	damage := math.Max(15, attack*0.75)
	return Skill{
		GuID: "BASIC_ATTACK",
		Rank: 1,
		Profile: models.MobaProfile{
			GuID:       "BASIC_ATTACK",
			Rank:       1,
			CastMode:   models.MobaCastTarget,
			Shape:      models.MobaShapeSingle,
			CastRange:  6.0,
			CastTimeMs: 80,
			RecoveryMs: 120,
			CooldownMs: 600,
			MaxTargets: 1,
		},
		Payload: SkillPayload{
			DamageFlat: damage,
		},
		ManaCost: 0,
	}
}

// BuildSkill compiles a Gu into a realtime skill: the aim/shape/timing card
// comes from gu_moba_profiles, the damage/status/shield payload from
// gu_dictionary.effects. A Gu with no MOBA row still produces a usable skill.
func BuildSkill(guID string, rank int) (Skill, bool) {
	if guID == "BASIC_ATTACK" {
		return BasicAttackSkill(20), true
	}
	spec, ok := models.GetGu(guID, rank)
	if !ok {
		return Skill{}, false
	}
	return Skill{
		GuID:     guID,
		Rank:     rank,
		Profile:  models.ResolveMobaProfile(guID, rank),
		Payload:  extractPayload(&spec),
		ManaCost: spec.BaseManaCost,
	}, true
}

// GuRef names one authored Gu rank.
type GuRef struct {
	GuID string
	Rank int
}

// BuildKit compiles a list of Gu into realtime skills, skipping unknown ones.
func BuildKit(refs []GuRef) []Skill {
	kit := make([]Skill, 0, len(refs))
	for _, ref := range refs {
		if skill, ok := BuildSkill(ref.GuID, ref.Rank); ok {
			kit = append(kit, skill)
		}
	}
	return kit
}

// extractPayload reads only the four Block v2 operations the realtime mode
// understands. Everything else (STAT_MOD, UNIQUE, CLEANSE, ...) is turn-based
// bookkeeping and is deliberately ignored.
func extractPayload(spec *models.GuDictionaryItem) SkillPayload {
	var payload SkillPayload
	for _, block := range spec.Effects.GetBlocks() {
		for _, op := range block.Operations {
			switch op.Op {
			case effects.OpDamage:
				if strings.EqualFold(op.SubType, "MULTIPLIER") {
					payload.DamageMult += op.Value
				} else {
					payload.DamageFlat += op.Value
				}
			case effects.OpShield:
				payload.ShieldFlat += op.Value
				if op.Target != nil && op.Target.Selector == effects.TargetSelf {
					payload.ShieldSelf = true
				}
			case effects.OpStatus:
				if op.StatusID == "" {
					continue
				}
				duration := op.Turns * StatusTicksPerTurn
				if duration <= 0 {
					duration = StatusTicksPerTurn
				}
				payload.Statuses = append(payload.Statuses, StatusApplication{
					ID:       strings.ToUpper(op.StatusID),
					Duration: duration,
					Chance:   op.Chance,
				})
			}
		}
	}
	return payload
}
