package models

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

func TestGetFlatGuEffectsPassiveOnlyDoesNotPanic(t *testing.T) {
	var effects GuEffects
	raw := `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"ADD_FLAT","value":10}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}
	gu := Gu{Effects: effects}
	effectsOut := GetFlatGuEffects(gu)
	if effectsOut["flat_damage"] != 10 {
		t.Fatalf("flat_damage = %v, want 10", effectsOut["flat_damage"])
	}
}

func TestGetFlatGuEffectsIncludesActiveCaptureBonus(t *testing.T) {
	gu := Gu{Effects: GuEffects{Active: &ActiveEffect{CaptureBonus: 0.25}}}
	effects := GetFlatGuEffects(gu)
	if effects["capture_bonus"] != 0.25 {
		t.Fatalf("capture_bonus = %v, want 0.25", effects["capture_bonus"])
	}
}

func TestGuEffectsScanRejectsStructurallyInvalidCatalogData(t *testing.T) {
	var effects GuEffects
	if err := effects.Scan([]byte(`{"active":[]}`)); err == nil {
		t.Fatal("invalid active namespace was accepted")
	}
}

func TestGuEffectsScanRejectsDeletedSupportSection(t *testing.T) {
	var effects GuEffects
	if err := effects.Scan([]byte(`{"support":{"heal_flat":10}}`)); err == nil {
		t.Fatal("deleted support section was accepted; healing lives in active.heal_flat")
	}
}

func TestStatusApplicationsUnmarshalAllForms(t *testing.T) {
	// Legacy string form.
	var legacy GuEffects
	if err := legacy.Scan([]byte(`{"active":{"apply_status":"BIND","status_duration_ticks":6}}`)); err != nil {
		t.Fatalf("string form rejected: %v", err)
	}
	if len(legacy.Active.ApplyStatus) != 1 || legacy.Active.ApplyStatus[0].ID != "BIND" || legacy.Active.ApplyStatus[0].Turns != 0 {
		t.Fatalf("string form parsed wrong: %+v", legacy.Active.ApplyStatus)
	}

	// Array form with multiple assembled statuses.
	var assembled GuEffects
	if err := assembled.Scan([]byte(`{"active":{"apply_status":[{"id":"BLEED","value":10,"turns":3},{"id":"BIND","turns":2}]}}`)); err != nil {
		t.Fatalf("array form rejected: %v", err)
	}
	if len(assembled.Active.ApplyStatus) != 2 || assembled.Active.ApplyStatus[1].ID != "BIND" || assembled.Active.ApplyStatus[1].Turns != 2 {
		t.Fatalf("array form parsed wrong: %+v", assembled.Active.ApplyStatus)
	}

	// Legacy object form.
	var single GuEffects
	if err := single.Scan([]byte(`{"active":{"apply_status":{"id":"STUN","turns":1}}}`)); err != nil {
		t.Fatalf("object form rejected: %v", err)
	}
	if len(single.Active.ApplyStatus) != 1 || single.Active.ApplyStatus[0].ID != "STUN" {
		t.Fatalf("object form parsed wrong: %+v", single.Active.ApplyStatus)
	}
}

func TestResolveStatusApplicationsLegacyEquivalence(t *testing.T) {
	// Legacy single-status row: tick damage set → direct value, top-level duration.
	legacy := &ActiveEffect{
		ApplyStatus:         StatusApplications{{ID: "BLEED"}},
		StatusTickDamage:    12,
		StatusDurationTicks: 4,
	}
	ops := legacy.ResolveStatusApplications()
	if len(ops) != 1 || ops[0].ID != "BLEED" || ops[0].Value != 12 || ops[0].Turns != 4 {
		t.Fatalf("legacy resolution changed: %+v", ops)
	}

	// Bare duration <= 5 keeps the tier pipeline.
	tier := &ActiveEffect{
		ApplyStatus:         StatusApplications{{ID: "WEAK"}},
		StatusDurationTicks: 2,
	}
	ops = tier.ResolveStatusApplications()
	if len(ops) != 1 || ops[0].Value == 0 || ops[0].Turns == 0 {
		t.Fatalf("tier resolution changed: %+v", ops)
	}

	// Assembled entries: each entry resolves with its own value/turns.
	assembled := &ActiveEffect{
		ApplyStatus: StatusApplications{{ID: "BLEED", Value: 10, Turns: 3}, {ID: "BIND", Turns: 2}},
	}
	ops = assembled.ResolveStatusApplications()
	if len(ops) != 2 {
		t.Fatalf("assembled entries dropped: %+v", ops)
	}
	if ops[0].Value != 10 || ops[0].Turns != 3 || ops[1].ID != "BIND" || ops[1].Turns != 2 {
		t.Fatalf("assembled resolution wrong: %+v", ops)
	}

	// FORGET_REALITY intensity from target_damage_reduce_percent.
	reality := &ActiveEffect{
		ApplyStatus:           StatusApplications{{ID: "FORGET_REALITY", Turns: 2}},
		TargetDamageReducePct: 0.4,
	}
	ops = reality.ResolveStatusApplications()
	if len(ops) != 1 || ops[0].Value != 40 {
		t.Fatalf("FORGET_REALITY intensity wrong: %+v", ops)
	}
}

func TestNormalizeLegacyShorthandBecomesEntries(t *testing.T) {
	var effects GuEffects
	if err := effects.Scan([]byte(`{"active":{"bleed":3}}`)); err != nil {
		t.Fatalf("shorthand rejected: %v", err)
	}
	if len(effects.Active.ApplyStatus) != 1 || effects.Active.ApplyStatus[0].ID != "BLEED" {
		t.Fatalf("shorthand not normalized to entries: %+v", effects.Active.ApplyStatus)
	}
	ops := effects.Active.ResolveStatusApplications()
	if len(ops) != 1 || ops[0].Value != 3 || ops[0].Turns != 3 {
		t.Fatalf("normalized shorthand resolves wrong: %+v", ops)
	}
}

func TestResolveStatusRankScaling(t *testing.T) {
	mk := func() *ActiveEffect {
		return &ActiveEffect{ApplyStatus: StatusApplications{
			{ID: "INVINCIBLE", Turns: 1, Chance: 0.35},
			{ID: "BLEED", Value: 10, Turns: 2},
		}}
	}
	// Rank 1: base values untouched.
	ops := mk().ResolveStatusApplicationsForRank(1)
	if ops[0].Chance != 0.35 || ops[0].Turns != 1 {
		t.Fatalf("rank1 = %+v, want chance 0.35 turns 1", ops[0])
	}
	// Rank 4: +0.15 chance, +1 turn. No-chance entry untouched.
	ops = mk().ResolveStatusApplicationsForRank(4)
	if ops[0].Chance != 0.50 || ops[0].Turns != 2 {
		t.Fatalf("rank4 = %+v, want chance 0.50 turns 2", ops[0])
	}
	if ops[1].Chance != 0 || ops[1].Turns != 2 {
		t.Fatalf("no-chance entry changed at rank4: %+v", ops[1])
	}
	// Rank 9: +0.40 chance, +2 turns.
	ops = mk().ResolveStatusApplicationsForRank(9)
	if ops[0].Chance != 0.75 || ops[0].Turns != 3 {
		t.Fatalf("rank9 = %+v, want chance 0.75 turns 3", ops[0])
	}
	// Legacy path carries chance raw without scaling.
	ops = mk().ResolveStatusApplications()
	if ops[0].Chance != 0.35 || ops[0].Turns != 1 {
		t.Fatalf("unranked = %+v, want chance 0.35 turns 1", ops[0])
	}
}

func TestGrowthProfileReadsBlockGrowth(t *testing.T) {
	var effects GuEffects
	raw := `[{"id":"GROWTH","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"GROWTH_CAP","value":10},{"op":"STAT_MOD","stat":"HP","sub_type":"GROWTH_CAP","value":20},{"op":"UNIQUE","code":"GROWTH_DURATION_HOURS","params":{"value":24}}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}
	spec := Gu{Effects: effects}
	profile, ok := GrowthProfile(spec)
	if !ok {
		t.Fatal("expected growth profile")
	}
	if profile.MaxAttack != 10 || profile.MaxHP != 20 || profile.HoursToMax != 24 {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestItemEffectCommandsAreSorted(t *testing.T) {
	item := ItemSpec{Effects: ItemEffects{"restore_mana": 2, "restore_hp": 1}}
	commands := item.EffectCommands()
	if len(commands) != 2 || commands[0].Code != "restore_hp" || commands[1].Code != "restore_mana" {
		t.Fatalf("unexpected commands: %+v", commands)
	}
}

func TestItemEffectsScan_Blocks(t *testing.T) {
	raw := `[{"id":"RESTORE_HUNGER","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_HUNGER","params":{"value":50}}]}]`
	var eff ItemEffects
	if err := eff.Scan(raw); err != nil {
		t.Fatalf("failed to scan item blocks: %v", err)
	}
	if eff["hunger"] != 50 {
		t.Fatalf("expected hunger 50, got: %v", eff)
	}
}

func TestCompiledProjectionReadsAuthoredBlocksOnly(t *testing.T) {
	var effects GuEffects
	raw := `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ACCURACY","sub_type":"ADD_FLAT","value":4},{"op":"STAT_MOD","stat":"INVESTIGATION","sub_type":"ADD_FLAT","value":7},{"op":"STAT_MOD","stat":"ATTACK","sub_type":"GROWTH_CAP","value":10},{"op":"UNIQUE","code":"GROWTH_DURATION_HOURS","params":{"value":24}}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}
	spec := Gu{Effects: effects}
	if got := GuStatValue(spec, "INVESTIGATION"); got != 7 {
		t.Fatalf("investigation = %v", got)
	}
	profile, ok := GrowthProfile(spec)
	if !ok || profile.MaxAttack != 10 || profile.HoursToMax != 24 {
		t.Fatalf("growth = %+v, %v", profile, ok)
	}
}

func TestGuStatValueReadsOnlyFlatAlwaysOnPassives(t *testing.T) {
	var effects GuEffects
	raw := `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"ADD_FLAT","value":12},{"op":"STAT_MOD","stat":"ATTACK","sub_type":"GROWTH_CAP","value":100}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}

	if got := GuStatValue(Gu{Effects: effects}, "ATTACK"); got != 12 {
		t.Fatalf("attack passive = %v, want 12 (growth cap must not be loaded as a passive)", got)
	}
}

func TestEquippedTimeSpeedBonusReadsBlockEffects(t *testing.T) {
	previous := GuRegistry
	t.Cleanup(func() { GuRegistry = previous })

	var effects GuEffects
	raw := `[{"id":"PASSIVE_TIME","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"TIME_SPEED","sub_type":"ADD_FLAT","value":2.5}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}
	GuRegistry = map[string]map[int]GuDictionaryItem{
		"GU_TIME": {
			1: {ID: "GU_TIME", Rank: 1, Effects: effects},
			2: {ID: "GU_TIME", Rank: 2, Effects: effects},
		},
	}

	inventory := []GuInstance{
		{GuID: "GU_TIME", Rank: 1, Equipped: true, Status: "ACTIVE"},
		{GuID: "GU_TIME", Rank: 2, Equipped: true, Status: "ACTIVE"},
		{GuID: "GU_TIME", Rank: 2, Equipped: true, Status: "ACTIVE"},
	}
	if got := EquippedTimeSpeedBonus(inventory); got != 2.5 {
		t.Fatalf("time speed bonus = %v, want 2.5", got)
	}
}

func TestGetFlatGuEffectsReadsBlockPassives(t *testing.T) {
	var effects GuEffects
	raw := `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"ADD_FLAT","value":12},{"op":"STAT_MOD","stat":"DEFENSE","sub_type":"ADD_FLAT","value":34},{"op":"STAT_MOD","stat":"ACCURACY","sub_type":"ADD_FLAT","value":5},{"op":"STAT_MOD","stat":"MOVEMENT","sub_type":"ADD_FLAT","value":2},{"op":"STAT_MOD","stat":"SPEED","sub_type":"ADD_FLAT","value":6},{"op":"STAT_MOD","stat":"EVASION","sub_type":"ADD_FLAT","value":15}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}

	flat := GetFlatGuEffects(Gu{Effects: effects})
	if flat["flat_damage"] != 12 || flat["flat_defense"] != 34 || flat["accuracy"] != 5 || flat["movement"] != 2 || flat["speed"] != 6 || flat["evasion"] != 15 {
		t.Fatalf("unexpected flat passive projection: %+v", flat)
	}
}

func TestImmortalTravelFirstTurnEvadeRank9Only(t *testing.T) {
	hasEvade := func(raw string) bool {
		var effects GuEffects
		if err := effects.Scan(raw); err != nil {
			t.Fatalf("scan travel effects: %v", err)
		}
		compiled := CompiledGuEffects(GuDictionaryItem{Effects: effects})
		if compiled == nil {
			t.Fatal("travel effects failed to compile")
		}
		for _, op := range compiled.NativeMechanics() {
			if string(op.Trigger) == "BEFORE_HIT" && op.Code == "FIRST_TURN_EVADE" {
				return true
			}
		}
		return false
	}

	// Rank 6-8: movement capabilities only, no evade.
	rankLow := `[{"id":"SPACE_CAPABILITIES","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"PORTAL_CREATE"},{"op":"UNIQUE","code":"IGNORE_MOVEMENT_RANGE"}]}]`
	if hasEvade(rankLow) {
		t.Fatal("rank 6-8 travel Gu must not carry FIRST_TURN_EVADE")
	}

	// Rank 9 (Gu.csv): capabilities plus first-strike evade.
	rank9 := `[{"id":"SPACE_CAPABILITIES","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"PORTAL_CREATE"},{"op":"UNIQUE","code":"IGNORE_MOVEMENT_RANGE"}]},{"id":"FIRST_TURN_EVADE","trigger":"BEFORE_HIT","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"FIRST_TURN_EVADE"}]}]`
	if issues := effectcatalog.ValidateBlockV2([]byte(rank9), effectcatalog.Location{Source: "gu", Key: "GU_FIXED_IMMORTAL_TRAVEL/9"}); len(issues) > 0 {
		t.Fatalf("rank 9 travel effects rejected by Admin import validator: %+v", issues)
	}
	if !hasEvade(rank9) {
		t.Fatal("rank 9 travel Gu must compile BEFORE_HIT/FIRST_TURN_EVADE")
	}
}
