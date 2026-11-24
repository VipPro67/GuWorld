package simulator

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestNormalizeLegacyDoTStatuses(t *testing.T) {
	c := &CombatantAuto{
		MaxHP: 1000,
		HP:    1000,
		ActiveEffects: []Effect{
			{ID: statusfx.StatusBurn, Value: 8, Duration: 3},
			{ID: statusfx.StatusRegen, Value: 10, Duration: 4},
			{ID: statusfx.StatusSwordMark, Duration: 4, Type: "debuff"}, // marker only
			{ID: statusfx.StatusDefUp, Value: 25, Duration: 3},          // not a DoT
		},
	}

	normalizeLegacyDoTStatuses(c)

	if len(c.ActiveDoT) != 2 {
		t.Fatalf("ActiveDoT = %+v; want BURN + REGEN", c.ActiveDoT)
	}
	if c.ActiveDoT[0].Type != "burn" || c.ActiveDoT[0].Value != 8 || c.ActiveDoT[0].Turns != 3 {
		t.Fatalf("burn dot = %+v; want {burn 8 3}", c.ActiveDoT[0])
	}
	if c.ActiveDoT[1].Type != "regen" || c.ActiveDoT[1].Value != 10 || c.ActiveDoT[1].Turns != 4 {
		t.Fatalf("regen dot = %+v; want {regen 10 4}", c.ActiveDoT[1])
	}
	if len(c.ActiveEffects) != 2 {
		t.Fatalf("ActiveEffects = %+v; want SWORD_MARK marker + DEF_UP", c.ActiveEffects)
	}
	if !hasStatus(c, statusfx.StatusBurn) {
		t.Fatal("hasStatus(BURN) = false after normalization; want true from ActiveDoT")
	}
}

func TestNormalizeLegacyDoTStatusesTicks(t *testing.T) {
	c := &CombatantAuto{
		MaxHP: 1000,
		HP:    1000,
		ActiveEffects: []Effect{
			{ID: statusfx.StatusBurn, Value: 8, Duration: 3},
		},
	}
	normalizeLegacyDoTStatuses(c)
	logs := tickDoTs(c, 1)
	if len(logs) != 1 || logs[0].Action != "DOT_TICK" || logs[0].Damage != 8 {
		t.Fatalf("tickDoTs = %+v; want one DOT_TICK for 8", logs)
	}
	if c.HP != 992 {
		t.Fatalf("HP = %v; want 992", c.HP)
	}
}

func TestDirectStatusDotMatchesLegacyMarkerWithoutSourceScaling(t *testing.T) {
	actor := &CombatantAuto{
		ID: "actor", Name: "Source", Rank: 9, Race: "wolf",
		DaoMarks: DaoMarks{"fire": 50000}, PathMasteries: map[string]int{"fire": 1000000},
		RaceDamageBonus: map[string]float64{"human": 0.75},
	}
	ensureDaoProfile(actor)
	target := func() *CombatantAuto {
		return &CombatantAuto{
			ID: "target", Name: "Target", Rank: 9, Race: "human", HP: 500, MaxHP: 500,
			RaceResistance: map[string]float64{"wolf": 0.5},
		}
	}
	legacyTarget := target()
	legacyTarget.ActiveEffects = []Effect{{ID: statusfx.StatusBurn, Value: 15, Duration: 3}}
	normalizeLegacyDoTStatuses(legacyTarget)

	typedTarget := target()
	statusOp := statusfx.StatusOp{
		ID: statusfx.StatusBurn, Value: 15, Turns: 3, Chance: 1,
		ApplicationMode: "DIRECT", SourcePath: "fire",
	}
	var logs []CombatLog
	applyStatusOpTo(actor, typedTarget, statusfx.Registry[statusfx.StatusBurn], statusOp, actor.Rank, 1, nil, &logs)
	if len(legacyTarget.ActiveDoT) != 1 || len(typedTarget.ActiveDoT) != 1 {
		t.Fatalf("legacy DoT=%+v typed DoT=%+v; want one each", legacyTarget.ActiveDoT, typedTarget.ActiveDoT)
	}
	legacy, typed := legacyTarget.ActiveDoT[0], typedTarget.ActiveDoT[0]
	if legacy.Value != typed.Value || legacy.Turns != typed.Turns || legacy.SourcePath != typed.SourcePath ||
		legacy.SourceDaoMult != typed.SourceDaoMult || legacy.SourceRace != typed.SourceRace || legacy.RaceDamageBonus != typed.RaceDamageBonus {
		t.Fatalf("direct DoT source differs from native marker: legacy=%+v typed=%+v", legacy, typed)
	}
	legacyLogs := appendTickDoTs(nil, legacyTarget, 2)
	typedLogs := appendTickDoTs(nil, typedTarget, 2)
	if len(legacyLogs) != 1 || len(typedLogs) != 1 || legacyLogs[0].Damage != typedLogs[0].Damage || legacyTarget.HP != typedTarget.HP {
		t.Fatalf("direct DoT tick differs: legacy logs=%+v HP=%v typed logs=%+v HP=%v", legacyLogs, legacyTarget.HP, typedLogs, typedTarget.HP)
	}
	if actor.DaoProfile == nil || gamefx.DaoAmplification(50000) <= 1 {
		t.Fatal("test must exercise a nontrivial source Dao profile")
	}
}

func TestComputeEffectMultipliersIsIdempotent(t *testing.T) {
	c := &CombatantAuto{
		MaxHP:    1000,
		HP:       500,
		Tenacity: 0.20,
		Mechanics: []CombatMechanic{
			mech("COMBAT_PASSIVE", "BERSERK_STATE", `{}`),
		},
	}
	ComputeEffectMultipliers(c)
	first := c.Tenacity
	ComputeEffectMultipliers(c)
	ComputeEffectMultipliers(c)
	if c.Tenacity != first {
		t.Fatalf("Tenacity drifted across recomputes: %v -> %v", first, c.Tenacity)
	}
	if first <= 0.20 {
		t.Fatalf("Tenacity = %v; want the berserk bonus applied on top of 0.20", first)
	}
	if c.DamageMultiplier < 1.39 || c.DamageMultiplier > 1.41 {
		t.Fatalf("legacy BERSERK_STATE damage multiplier = %v, want 1.40", c.DamageMultiplier)
	}

	dataDriven := &CombatantAuto{
		MaxHP: 1000,
		HP:    500,
		PassiveOperationEffects: []SimActionOperation{{
			Compiled: gameplayeffects.CompiledOperation{Type: gameplayeffects.OpTenacityModifier, PerMissingHP: 0.5},
		}},
	}
	ComputeEffectMultipliers(dataDriven)
	if dataDriven.DamageMultiplier != 1.0 {
		t.Fatalf("data-driven BERSERK_STATE duplicated damage scaling in native projection: %v", dataDriven.DamageMultiplier)
	}
	if dataDriven.Tenacity <= 0 {
		t.Fatalf("typed BERSERK passive lost Tenacity projection: %v", dataDriven.Tenacity)
	}
	dataDriven.ActiveEffects = []Effect{{ID: EffectStun, Duration: 1}, {ID: EffectFreeze, Duration: 1}}
	dataDriven.PassiveOperationEffects = append(dataDriven.PassiveOperationEffects,
		SimActionOperation{Compiled: gameplayeffects.CompiledOperation{Type: gameplayeffects.OpStatusImmunity, SubType: "STUN", Conditions: []gameplayeffects.Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.3}}}},
		SimActionOperation{Compiled: gameplayeffects.CompiledOperation{Type: gameplayeffects.OpStatusImmunity, SubType: "FREEZE", Conditions: []gameplayeffects.Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.3}}}},
	)
	dataDriven.HP = 300
	ComputeEffectMultipliers(dataDriven)
	if dataDriven.IsStunned || dataDriven.IsFrozen {
		t.Fatalf("typed BERSERK immunity failed at 30%% HP: stunned=%v frozen=%v", dataDriven.IsStunned, dataDriven.IsFrozen)
	}
}
