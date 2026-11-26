package gamefx

import (
	"math"
	"testing"
)

func TestResolveEnvSkillModifiers(t *testing.T) {
	cfg := DefaultEnvDaoConfig

	t.Run("Natural resonance and conflict", func(t *testing.T) {
		env := &EnvCombatContext{
			NaturalDaoMarks: map[string]float64{"fire": 15000},
		}
		aligned := ResolveEnvSkillModifiers(cfg, env, "fire", "p1", 7, false, false)
		if aligned.PowerMult <= 1.15 || aligned.PowerMult > 1.351 {
			t.Fatalf("expected aligned PowerMult in (1.15, 1.35], got %.3f", aligned.PowerMult)
		}

		conflict := ResolveEnvSkillModifiers(cfg, env, "water", "p1", 7, false, false)
		if conflict.PowerMult >= 1.0 || conflict.PowerMult < cfg.NaturalMaxPenalty {
			t.Fatalf("expected conflict PowerMult < 1.0, got %.3f", conflict.PowerMult)
		}

		// HasNonConflictingDao immune to natural conflict
		nonConf := ResolveEnvSkillModifiers(cfg, env, "water", "p1", 7, false, true)
		if nonConf.PowerMult != 1.0 {
			t.Fatalf("expected HasNonConflictingDao immune to natural conflict, got %.3f", nonConf.PowerMult)
		}
	})

	t.Run("Dao Domain overlord vs enemy and non-conflicting suppression", func(t *testing.T) {
		env := &EnvCombatContext{
			DomainOwnerID:    "venerable-1",
			DomainPath:       "refinement",
			DomainControlPct: 100,
		}
		owner := ResolveEnvSkillModifiers(cfg, env, "refinement", "venerable-1", 9, false, false)
		if owner.PowerMult != cfg.DomainOwnerMaxMult {
			t.Fatalf("expected owner PowerMult %.2f, got %.2f", cfg.DomainOwnerMaxMult, owner.PowerMult)
		}

		ally := ResolveEnvSkillModifiers(cfg, env, "refinement", "ally-1", 7, true, false)
		if ally.PowerMult != cfg.DomainAllyMaxMult {
			t.Fatalf("expected ally PowerMult %.2f, got %.2f", cfg.DomainAllyMaxMult, ally.PowerMult)
		}

		// Enemy with HasNonConflictingDao is STILL suppressed in enemy Dao Domain
		enemy := ResolveEnvSkillModifiers(cfg, env, "fire", "enemy-1", 9, false, true)
		if math.Abs(enemy.PowerMult-cfg.DomainEnemyFloor) > 1e-9 {
			t.Fatalf("expected enemy PowerMult %.2f, got %.2f", cfg.DomainEnemyFloor, enemy.PowerMult)
		}
		if enemy.ManaTaxMult <= 1.0 || enemy.SuccessChanceDelta >= 0 || enemy.BackfireBonus <= 0 {
			t.Fatalf("expected stability penalties in enemy domain, got %+v", enemy)
		}

		// Mortal Rank 4 protection
		mortal := ResolveEnvSkillModifiers(cfg, env, "fire", "mortal-1", 4, false, false)
		if mortal.PowerMult < cfg.MortalMinFloor || mortal.SuccessChanceDelta != 0 {
			t.Fatalf("expected mortal floor %.2f and no success penalty, got %+v", cfg.MortalMinFloor, mortal)
		}
	})
}
