package handlers

import (
	"strings"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestAdaptBlockCombatOperationsPreservesTargetsAndConditions(t *testing.T) {
	actor := &models.Combatant{MaxHP: 120, BaseAttack: 45, GridX: 2, GridY: 3}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 3, Path: []string{"Fire Path"}}
	block := gameplayeffects.Block{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle, BypassFrontline: true},
		Conditions: []gameplayeffects.Condition{{Field: "TARGET_HP_PERCENT", Operator: "LTE", Value: 0.5}},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 24},
			{Op: gameplayeffects.OpStatus, StatusID: "STUN", Turns: 2, StackPolicy: "MAX"},
		},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1.25, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 {
		t.Fatalf("expected same-block operations to share one simulator action, got %d", len(effects))
	}
	got := effects[0]
	if got.Compiled.BlockID != "ACTION" || got.SourceRank != 3 || got.Action == nil {
		t.Fatalf("operation provenance was lost: %+v", got)
	}
	if got.Action.Damage != 24 || got.Action.DamageMult != 1.25 || got.Action.TargetType != "TARGET" || !got.Action.Bypass {
		t.Fatalf("combat projection mismatch: %+v", got.Action)
	}
	if got.Action.TargetX != 2 || got.Action.TargetY != 3 || len(got.Action.OperationConditions) != 1 {
		t.Fatalf("target position or conditions were lost: %+v", got.Action)
	}
	if len(got.Action.StatusOps) != 1 || got.Action.StatusOps[0].ID != "STUN" || got.Action.StatusOps[0].Turns != 2 ||
		got.Action.StatusOps[0].EffectID != "GU_TEST:ACTION:0:1" || got.Action.StatusOps[0].Context != "combat" || got.Action.StatusOps[0].StackPolicy != "MAX" {
		t.Fatalf("status operation was not compiled: %+v", got.Action.StatusOps)
	}
}

func TestAdaptBlockCombatOperationsCarriesChainDamage(t *testing.T) {
	actor := &models.Combatant{GridX: 1, GridY: 1}
	spec := &models.GuDictionaryItem{ID: "GU_CHAIN", Name: "Lôi Cổ", Rank: 1}
	block := gameplayeffects.Block{
		ID: "CHAIN", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetAllEnemies},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpChainDamage, Value: 0.5, MinDamage: 30, TargetCount: 1,
			LogAction: "CHAIN_LIGHTNING", Message: "{actor} chains to {target} for {damage}.",
		}},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || len(effects[0].Action.ChainDamages) != 1 {
		t.Fatalf("chain damage was not adapted: %+v", effects)
	}
	got := effects[0].Action.ChainDamages[0]
	if got.Fraction != 0.5 || got.MinDamage != 30 || got.TargetCount != 1 || got.LogAction != "CHAIN_LIGHTNING" {
		t.Fatalf("chain damage config changed in projection: %+v", got)
	}
}

func TestAdaptBlockCombatOperationsCarriesOnCastDispel(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "UNRAVEL_RIDDLE", Trigger: gameplayeffects.TriggerOnCast,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpDispel, SubType: "BUFFS", Value: 2,
			Target:    &gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
			LogAction: "DISPEL", Message: "{actor} removes {dispelled} buffs from {target}.",
		}},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_TEST", Name: "Giải Mê", Rank: 1}, 1, compiled.Operations(gameplayeffects.TriggerOnCast))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Compiled.Trigger != gameplayeffects.TriggerOnCast || effects[0].Action == nil || effects[0].Action.DispelBuffs != 2 || effects[0].Action.TargetType != "TARGET" {
		t.Fatalf("ON_CAST DISPEL was not adapted: %+v", effects)
	}
}

func TestAdaptBlockCombatOperationsCarriesProtectionDispel(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "FATE_VANQUISHED", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpDispel, SubType: "PROTECTION",
			Target:    &gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
			LogAction: "FATE_VANQUISHED", Message: "{actor} breaks {target}'s protections.",
		}},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_TEST", Name: "Mệnh Bại", Rank: 8}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || !effects[0].Action.DispelProtections {
		t.Fatalf("DISPEL PROTECTION was not adapted: %+v", effects)
	}
}

func TestAdaptBlockCombatOperationsCarriesOnCastSelfStatus(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "BORROW_PAST_POWER", Trigger: gameplayeffects.TriggerOnCast,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpStatus, StatusID: "ATK_UP", Value: 40, Turns: 3, Chance: 1,
			ApplicationMode: "DIRECT", Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			LogAction: "BUFF", Message: "{actor} borrows past power.",
		}},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_TEST", Name: "Mượn Lực", Rank: 1}, 1, compiled.Operations(gameplayeffects.TriggerOnCast))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || len(effects[0].Action.StatusOps) != 1 {
		t.Fatalf("ON_CAST STATUS was not adapted: %+v", effects)
	}
	status := effects[0].Action.StatusOps[0]
	if status.ID != "ATK_UP" || status.Value != 40 || status.Turns != 3 || status.ApplicationMode != "DIRECT" || status.LogAction != "BUFF" {
		t.Fatalf("ON_CAST STATUS projection changed: %+v", status)
	}
}

func TestAdaptBlockCombatRejectsMixedPartyManaLogPolicies(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "PARTY_MANA", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetAllAllies},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpHeal, SubType: "RESOURCE", Resource: "MANA", Unit: "PERCENT", Value: 20, SuppressLog: true},
			{Op: gameplayeffects.OpHeal, SubType: "RESOURCE", Resource: "MANA", Unit: "FLAT", Value: 10},
		},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err == nil || !strings.Contains(err.Error(), "mixes suppress_log policies") {
		t.Fatalf("mixed grouped party mana log policies error = %v", err)
	}
}

func TestCombatPassiveDamageModifierProjectsFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_BEAST_WHIP_1",
		Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "PACK_FRENZY", Trigger: gameplayeffects.TriggerCombatPassive,
			Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{
				Op: gameplayeffects.OpDamageModifier, SubType: "MINION_COUNT", Value: 1, PerMinion: 0.15,
				LogAction: "PACK_FRENZY", Message: "{actor} {minion_count} {damage_percent}",
			}},
		}}},
	}
	got := combatPassiveDamageModifiers(spec)
	if len(got) != 1 || got[0].SubType != "MINION_COUNT" || got[0].BaseMultiplier != 1 || got[0].PerMinion != 0.15 || got[0].LogAction != "PACK_FRENZY" {
		t.Fatalf("combat passive damage modifier projection = %+v", got)
	}
}

func TestCombatPassiveMissingHPModifierProjectsFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_BERSERK", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "BERSERK_STATE", Trigger: gameplayeffects.TriggerCombatPassive,
			Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{
				{Op: gameplayeffects.OpDamageModifier, SubType: "MISSING_HP", Value: 1, PerMissingHP: 0.8},
				{Op: gameplayeffects.OpUnique, Code: "BERSERK_STATE", Params: []byte(`{"damage_modifier_in_primitive":true}`)},
			},
		}}},
	}
	got := combatPassiveDamageModifiers(spec)
	if len(got) != 1 || got[0].SubType != "MISSING_HP" || got[0].BaseMultiplier != 1 || got[0].PerMissingHP != 0.8 {
		t.Fatalf("Berserk damage component projection = %+v", got)
	}
	mechanics := combatMechanicsForSpec(spec, false)
	if len(mechanics) != 1 || mechanics[0].Values["damage_modifier_in_primitive"] != true {
		t.Fatalf("Berserk compatibility parameter projection = %+v", mechanics)
	}
}

func TestCombatTurnStartCounterProjectsFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_POEM_IMMORTAL", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "POEM_INTENT", Trigger: gameplayeffects.TriggerOnTurnStart,
			Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpCounterAdd, Key: "thi_y", Delta: 1, Min: 0, Max: 10}},
		}}},
	}
	got := combatTurnStartCounterAdds(spec)
	if len(got) != 1 || got[0] != (simulator.CounterAdd{Key: "thi_y", Delta: 1, Min: 0, Max: 10}) {
		t.Fatalf("turn-start counter projection = %+v", got)
	}
}

func TestCombatBeforeHitCounterProjectsFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_EATING_LOSS_BLESSING_I", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "EATING_LOSS_STACK", Trigger: gameplayeffects.TriggerBeforeHit,
			Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpCounterAdd, Key: "eating_loss_stacks", Delta: 1, Min: 0, Max: 10}},
		}}},
	}
	got := combatBeforeHitCounterAdds(spec)
	want := simulator.CounterAdd{Key: "eating_loss_stacks", Delta: 1, Min: 0, Max: 10}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("before-hit counter projection = %+v, want [%+v]", got, want)
	}
}

func TestCombatBeforeHitCounterStatModifiersProjectFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_EATING_LOSS_BLESSING_I", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "EATING_LOSS_STACK", Trigger: gameplayeffects.TriggerBeforeHit,
			Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{
				Op: gameplayeffects.OpStatMod, SubType: "ADD_COUNTER_DELTA", Stat: "ATTACK", Key: "eating_loss_stacks", PerStack: 5,
			}},
		}}},
	}
	got := combatBeforeHitCounterStatModifiers(spec)
	want := simulator.CounterStatModifier{Key: "eating_loss_stacks", Stat: "ATTACK", PerCounter: 5}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("before-hit stat modifier projection = %+v, want [%+v]", got, want)
	}
}

func TestCombatBeforeHitCounterNotificationsProjectFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_EATING_LOSS_BLESSING_I", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "EATING_LOSS_STACK", Trigger: gameplayeffects.TriggerBeforeHit,
			Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{
				Op: gameplayeffects.OpNotify, Key: "eating_loss_stacks", Phase: "BEFORE_HIT",
				LogAction: "EATING_LOSS_BLESSING", Message: "{actor} gains stack {counter_count}",
			}},
		}}},
	}
	got := combatBeforeHitCounterNotifications(spec)
	want := simulator.CombatNotification{Action: "EATING_LOSS_BLESSING", Message: "{actor} gains stack {counter_count}", CounterKey: "eating_loss_stacks", Phase: "BEFORE_HIT"}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("before-hit notification projection = %+v, want [%+v]", got, want)
	}
}

func TestCombatBeforeHitRecoveryProjectsFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_QI_JUE_LIFE_I", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "RECOVERY", Trigger: gameplayeffects.TriggerBeforeHit,
			Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{
				Op: gameplayeffects.OpThresholdRecovery, HPThreshold: 0.3, HealPercent: 0.4, MaxUses: 1,
				CleanseDebuffs: true, LogAction: "QI_JUE_RETURN_LIFE", Message: "{actor} recovers {heal_amount}",
			}},
		}}},
	}
	got := combatBeforeHitRecoveries(spec)
	want := simulator.ThresholdRecovery{
		SourceID: spec.ID, OperationID: "RECOVERY:0:0", HPThreshold: 0.3, HealPercent: 0.4,
		MaxUses: 1, CleanseDebuffs: true, LogAction: "QI_JUE_RETURN_LIFE", Message: "{actor} recovers {heal_amount}",
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("before-hit recovery projection = %+v, want [%+v]", got, want)
	}
}

func TestCombatPassiveCounterPayoutProjectsFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_EATING_LOSS_BLESSING_I", Name: "Ăn Thiệt Là Phúc", Rank: 6,
		Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "EATING_LOSS_PAYOUT", Trigger: gameplayeffects.TriggerCombatPassive,
			Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{
				{Op: gameplayeffects.OpDamageModifier, SubType: "COUNTER_COUNT", Value: 1.3, PerStack: 0.15, Key: "eating_loss_stacks", Phase: "FINAL_RIDER"},
				{Op: gameplayeffects.OpHeal, SubType: "COUNTER_PERCENT", Value: 0.05, Key: "eating_loss_stacks", Phase: "FINAL_RIDER", Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}},
				{Op: gameplayeffects.OpNotify, Key: "eating_loss_stacks", Phase: "FINAL_RIDER", LogAction: "EATING_LOSS_BLESSING", Message: "{counter_count} {damage_percent} {heal_amount}"},
			},
		}}},
	}
	got := combatPassiveOperationEffects(spec)
	if len(got) != 3 {
		t.Fatalf("counter payout operations = %d, want 3", len(got))
	}
	modifier := got[0].Action.DamageModifiers
	if len(modifier) != 1 || modifier[0].SubType != "COUNTER_COUNT" || modifier[0].CounterKey != "eating_loss_stacks" || modifier[0].PerCounter != 0.15 {
		t.Fatalf("counter damage modifier projection = %+v", modifier)
	}
	heals := got[1].Action.CounterHeals
	if len(heals) != 1 || heals[0] != (simulator.CounterHeal{Key: "eating_loss_stacks", PerCounter: 0.05}) {
		t.Fatalf("counter heal projection = %+v", heals)
	}
	if notifications := got[2].Action.CombatNotifications; len(notifications) != 1 || notifications[0] != (simulator.CombatNotification{Action: "EATING_LOSS_BLESSING", Message: "{counter_count} {damage_percent} {heal_amount}", CounterKey: "eating_loss_stacks", Phase: "FINAL_RIDER", EffectID: "EATING_LOSS_PAYOUT:0:2"}) {
		t.Fatalf("counter payout notification projection = %+v", notifications)
	}
}

func TestCombatBattleStartEndCommandProjectsFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{ID: "GU_XUAN", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
		ID: "INSTANT_WIN", Trigger: gameplayeffects.TriggerOnBattleStart,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpEndBattle, SubType: "FORCE_VICTORY",
			Target:    &gameplayeffects.Target{Selector: gameplayeffects.TargetAllEnemies},
			LogAction: "INSTANT_WIN", Message: "{actor} ends the battle.",
		}},
	}}}}
	got := combatBattleStartEndCommands(spec)
	if len(got) != 1 || got[0] != (simulator.BattleEndCommand{Outcome: "FORCE_VICTORY", LogAction: "INSTANT_WIN", Message: "{actor} ends the battle."}) {
		t.Fatalf("battle-start end command projection = %+v", got)
	}
}

func TestBattleStartActorSummonAndSetRowProjectFromGuBlocks(t *testing.T) {
	const templateID = "monster_test_actor_summon"
	previous, existed := models.MonsterRegistry[templateID]
	models.MonsterRegistry[templateID] = models.MonsterTemplate{ID: templateID, Name: "Registry Name", MaxHP: 100, BaseAttack: 20, BaseDefense: 10}
	t.Cleanup(func() {
		if existed {
			models.MonsterRegistry[templateID] = previous
		} else {
			delete(models.MonsterRegistry, templateID)
		}
	})
	spec := models.GuDictionaryItem{ID: "GU_TEST", Rank: 5, Effects: models.GuEffects{Blocks: []gameplayeffects.Block{
		{
			ID: "SUMMON", Trigger: gameplayeffects.TriggerOnBattleStart,
			Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpSummon, Params: []byte(`{"monster_id":"monster_test_actor_summon","hp_scale":0.5,"damage_scale":0.3333333333,"defense_scale":0.5,"stat_source":"ACTOR","rank_scale":false,"name":"Configured Name","id_suffix":"stable"}`)}},
		},
		{
			ID: "ROW", Trigger: gameplayeffects.TriggerOnBattleStart,
			Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpDisplace, SubType: "SET_ROW", Value: 3}},
		},
	}}}
	mechanics := battleStartMechanics(spec)
	if len(mechanics) != 2 {
		t.Fatalf("battle-start mechanics = %+v, want summon and displacement", mechanics)
	}
	if mechanics[0].ActionCode != "SUMMON_MONSTER" || mechanics[0].Values["name"] != "Configured Name" || mechanics[0].Values["hp"] != float64(100) || mechanics[0].Values["stat_source"] != "ACTOR" {
		t.Fatalf("resolved summon mechanic = %+v", mechanics[0])
	}
	if mechanics[1].ActionCode != "DISPLACE" || mechanics[1].Values["direction"] != "SET_ROW" || mechanics[1].Values["steps"] != float64(3) {
		t.Fatalf("displacement mechanic = %+v", mechanics[1])
	}
}

func TestCombatDamageMitigationProjectsFromGuBlock(t *testing.T) {
	spec := models.GuDictionaryItem{
		ID: "GU_BLOOD_STAINED_ROBE", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "BLOOD_STAINED_ROBE", Trigger: gameplayeffects.TriggerBeforeHit,
			Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{
				Op: gameplayeffects.OpMitigateDamage, SubType: "CONVERT_TO_SHIELD", Value: 0.4,
				LogAction: "BLOOD_STAINED_ROBE", Message: "absorbs {mitigated}",
			}},
		}}},
	}
	got := combatDamageMitigations(spec)
	if len(got) != 1 || got[0].BlockID != "BLOOD_STAINED_ROBE" || got[0].Ratio != 0.4 || !got[0].ConvertToShield || got[0].LogAction != "BLOOD_STAINED_ROBE" {
		t.Fatalf("combat damage mitigation projection = %+v", got)
	}
}

func TestCombatPassiveConditionalOperationsProjectPhaseAndStatusScope(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "PASSION_OVERLOAD", Trigger: gameplayeffects.TriggerCombatPassive,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Conditions: []gameplayeffects.Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.35}},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamageModifier, SubType: "MISSING_HP", Value: 1.5, Phase: "LATE_RIDER", LogAction: "PASSION_OVERLOAD", Message: "{actor}"},
			{Op: gameplayeffects.OpCleanse, SubType: "STATUS", StatusID: "STUN", Phase: "LATE_RIDER"},
		},
	}
	if _, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block}); err != nil {
		t.Fatal(err)
	}
	spec := models.GuDictionaryItem{ID: "GU_FATE_BOND", Rank: 3, Path: []string{"Emotion Path"}, Effects: models.GuEffects{Blocks: []gameplayeffects.Block{block}}}
	got := combatPassiveOperationEffects(spec)
	if len(got) != 2 {
		t.Fatalf("passive operation projection count = %d, want 2", len(got))
	}
	if len(combatPassiveDamageModifiers(spec)) != 0 {
		t.Fatal("conditional/phase-bound modifier leaked into early passive modifier phase")
	}
	if got[0].Compiled.Type != gameplayeffects.OpDamageModifier || got[0].Compiled.Phase != "LATE_RIDER" || len(got[0].Compiled.Conditions) != 1 || len(got[0].SourcePaths) != 1 {
		t.Fatalf("conditional passive metadata lost: %+v", got[0])
	}
	if got[1].Action == nil || len(got[1].Action.CleanseStatusIDs) != 1 || got[1].Action.CleanseStatusIDs[0] != "STUN" {
		t.Fatalf("status-specific cleanse projection mismatch: %+v", got[1])
	}
}

func TestBerserkTypedPassivesProjectToCombatant(t *testing.T) {
	blocks := []gameplayeffects.Block{
		{ID: "TENACITY", Trigger: gameplayeffects.TriggerCombatPassive, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}, Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpTenacityModifier, PerMissingHP: 0.5}}},
		{ID: "STUN_IMMUNITY", Trigger: gameplayeffects.TriggerCombatPassive, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}, Conditions: []gameplayeffects.Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.3}}, Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpStatusImmunity, SubType: "STUN"}}},
	}
	spec := models.GuDictionaryItem{ID: "GU_BERSERK", Name: "Cuồng Bạo Cổ", Rank: 4, Effects: models.GuEffects{Blocks: blocks}}
	got := combatPassiveOperationEffects(spec)
	if len(got) != 2 {
		t.Fatalf("projected passive operations = %d, want 2", len(got))
	}
	if got[0].Compiled.Type != gameplayeffects.OpTenacityModifier || got[0].Compiled.PerMissingHP != 0.5 {
		t.Fatalf("Tenacity operation projection = %+v", got[0].Compiled)
	}
	if got[1].Compiled.Type != gameplayeffects.OpStatusImmunity || got[1].Compiled.SubType != "STUN" || len(got[1].Compiled.Conditions) != 1 {
		t.Fatalf("status immunity projection = %+v", got[1].Compiled)
	}
}

func TestCombatHostileEffectImmunityProjectsFromTypedPassive(t *testing.T) {
	spec := models.GuDictionaryItem{ID: "GU_REVERSE", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
		ID: "REVERSE_CALAMITY", Trigger: gameplayeffects.TriggerCombatPassive,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpEffectImmunity, SubType: "HOSTILE_EFFECTS"}},
	}}}}
	got := combatPassiveOperationEffects(spec)
	if len(got) != 1 || got[0].Compiled.Type != gameplayeffects.OpEffectImmunity || got[0].Compiled.SubType != "HOSTILE_EFFECTS" {
		t.Fatalf("hostile effect immunity projection = %+v", got)
	}
}

func TestCombatBeforeEnemyAttackGuSealProjectsTypedParams(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "FORGET_INTENT_SEAL", Trigger: gameplayeffects.TriggerBeforeEnemyAttack,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpGuSeal, SubType: "REACTIVE", Params: []byte(`{"counter_key":"forget_intent_success","chance":0.4,"turns":1,"max_activations":8,"backfire_at_limit":true,"backfire_turns":1}`)}},
	}
	spec := models.GuDictionaryItem{ID: "GU_FORGET_INTENT", Name: "Lãng Quên Ý Niệm", Rank: 7, Effects: models.GuEffects{Blocks: []gameplayeffects.Block{block}}}
	got := combatBeforeEnemyAttackOperationEffects(spec)
	if len(got) != 1 || got[0].Compiled.Type != gameplayeffects.OpGuSeal || got[0].Action == nil || got[0].Compiled.GuSeal == nil {
		t.Fatalf("typed GU_SEAL projection mismatch: %+v", got)
	}
	seal := got[0].Compiled.GuSeal
	if seal.CounterKey != "forget_intent_success" || seal.Chance != 0.4 || seal.Turns != 1 || seal.MaxActivations != 8 || !seal.BackfireAtLimit || seal.BackfireTurns != 1 {
		t.Fatalf("typed GU_SEAL params lost: %+v", seal)
	}
}

func TestAdaptBlockCombatStatusDetonation(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 300},
			{Op: gameplayeffects.OpDetonateStatus, StatusID: "BLEED", PerStack: 100, PerStrength: 0, MaxHPPercent: 0.08, MinDamage: 0, Phase: "EARLY_RIDER", LogAction: "BLEED_BURST", Message: "{actor} {stack_count} {target} {damage}"},
		},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	adapted, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_BLOOD_BLADE", Name: "Huyết Nhận", Rank: 6}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapted) != 1 || adapted[0].Action == nil || adapted[0].Action.Damage != 300 || len(adapted[0].Action.StatusDetonations) != 1 {
		t.Fatalf("status detonation adapter mismatch: %+v", adapted)
	}
	got := adapted[0].Action.StatusDetonations[0]
	if got.StatusID != "BLEED" || got.PerStack != 100 || got.MaxHPPercent != 0.08 || got.Phase != "EARLY_RIDER" || got.LogAction != "BLEED_BURST" {
		t.Fatalf("status detonation parameters were lost: %+v", got)
	}
}

func TestAdaptBlockCombatPreChainStatusAndNotification(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "LIGHTNING", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamage, SubType: "MULTIPLIER", Value: 1.3, Phase: "PRE_CHAIN"},
			{Op: gameplayeffects.OpStatus, StatusID: "STUN", Turns: 1, Chance: 0.4, ApplicationMode: "DIRECT", Phase: "PRE_CHAIN", SuppressLog: true},
			{Op: gameplayeffects.OpNotify, Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}, Phase: "PRE_CHAIN", LogAction: "LIGHTNING_STUN_BURST", Message: "{actor} strikes {target}."},
		},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	got, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_BLUE_DRAGON_1", Rank: 1}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected target and self operation groups, got %+v", got)
	}
	var hasDamageAndStatus, hasNotify bool
	for _, effect := range got {
		if effect.Action == nil {
			continue
		}
		if len(effect.Action.PreChainDamageModifiers) == 1 && effect.Action.PreChainDamageModifiers[0].Multiplier == 1.3 && len(effect.Action.StatusOps) == 1 && effect.Action.StatusOps[0].Phase == "PRE_CHAIN" && effect.Action.StatusOps[0].Chance == 0.4 {
			hasDamageAndStatus = true
		}
		if len(effect.Action.CombatNotifications) == 1 && effect.Action.CombatNotifications[0].Phase == "PRE_CHAIN" {
			hasNotify = true
		}
	}
	if !hasDamageAndStatus || !hasNotify {
		t.Fatalf("pre-chain operation metadata was lost: %+v", got)
	}
}

func TestAdaptBlockCombatHPCostPreservesEarlyPhase(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "GU_DETONATION", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpHPCost, SubType: "CURRENT_HP_PERCENT", Value: 0.05,
			MinCost: 1, MinRemaining: 1, Phase: "EARLY_RIDER", LogAction: "GU_DETONATION", Message: "{actor}",
		}},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	got, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_ASCEND_WELL_1", Rank: 1}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Action == nil || len(got[0].Action.HPCosts) != 1 || got[0].Action.HPCosts[0].Phase != "EARLY_RIDER" {
		t.Fatalf("early HP cost was not projected: %+v", got)
	}
}

func TestHPCostPrimitiveCombinesWithDamageAndLifesteal(t *testing.T) {
	actor := &models.Combatant{MaxHP: 1000, BaseAttack: 100, GridX: 0, GridY: 0}
	spec := &models.GuDictionaryItem{ID: "GU_BLOOD_DEMON", Name: "Huyết Ma Tiên Cổ", Rank: 6}
	block := gameplayeffects.Block{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 100},
			{Op: gameplayeffects.OpDamage, SubType: "MULTIPLIER", Value: 2.2},
			{Op: gameplayeffects.OpHeal, SubType: "LIFESTEAL", Value: 50},
			{Op: gameplayeffects.OpHPCost, SubType: "CURRENT_HP_PERCENT", Value: 0.2, MinCost: 10, MinRemaining: 1,
				LogAction: "BLOOD_DEMON_DISINTEGRATION", Message: "[Huyết Đạo - Huyết Ma Giải Thể] {actor} hy sinh {cost} HP giải thể ma huyết, tăng bộc phát 120% sát thương và nhận 50% Hút Máu!",
				Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}},
		},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	actions, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 {
		t.Fatalf("target damage and self cost should produce two operations, got %+v", actions)
	}
	var attack, cost *simulator.SimAction
	for _, action := range actions {
		if action.Action.TargetType == "SELF" {
			cost = action.Action
		} else {
			attack = action.Action
		}
	}
	if attack == nil || attack.Damage != 100 || attack.DamageMult != 2.2 || len(attack.DamageModifiers) != 0 || attack.LifeStealPct != 0.5 {
		t.Fatalf("attack operations were not composed: %+v", attack)
	}
	if cost == nil || len(cost.HPCosts) != 1 || cost.HPCosts[0].CurrentHPPercent != 0.2 || cost.HPCosts[0].MinCost != 10 || cost.HPCosts[0].MinRemaining != 1 {
		t.Fatalf("self HP cost was not projected: %+v", cost)
	}
	actorAuto := &simulator.CombatantAuto{ID: "blood", Name: "Ma Tu", Kind: "Player", MaxHP: 1000, HP: 1000, Speed: 100,
		Actions: []simulator.SimAction{{ActionType: "GU_SKILL", Name: spec.Name, SourceID: spec.ID, TargetType: "SINGLE", OperationEffects: actions}}}
	target := &simulator.CombatantAuto{ID: "enemy", Name: "Chính Đạo", Kind: "Enemy", MaxHP: 1000, HP: 220, Speed: 1}
	logs := simulator.Simulate(&simulator.CombatState{Seed: 1, Allies: []*simulator.CombatantAuto{actorAuto}, Enemies: []*simulator.CombatantAuto{target}})
	if actorAuto.HP != 910 || target.HP != 0 {
		t.Fatalf("HP cost/damage/lifesteal outcome = actor %.0f, target %.0f; want 910/0", actorAuto.HP, target.HP)
	}
	var costLogs int
	for _, log := range logs {
		if log.Action == "BLOOD_DEMON_DISINTEGRATION" {
			costLogs++
			if log.Damage != 200 || log.Detail != "[Huyết Đạo - Huyết Ma Giải Thể] Ma Tu hy sinh 200 HP giải thể ma huyết, tăng bộc phát 120% sát thương và nhận 50% Hút Máu!" {
				t.Fatalf("HP cost log mismatch: %+v", log)
			}
		}
	}
	if costLogs != 1 {
		t.Fatalf("HP cost log count = %d, want 1; logs=%+v", costLogs, logs)
	}
}

func TestAdaptBlockCombatMultiHitSharesDamageTargetGroup(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetAllEnemies},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 100},
			{Op: gameplayeffects.OpDamage, SubType: "MULTIPLIER", Value: 1.8},
			{Op: gameplayeffects.OpHitRule, SubType: "MULTI_HIT", Params: []byte(`{"count":3}`)},
			{Op: gameplayeffects.OpStatus, StatusID: "BLEED", Turns: 3, Chance: 1, ApplicationMode: "DIRECT", Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSingle}},
		},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_TRIPLE", Name: "Kiếm Lãng", Rank: 6}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	var attack, status *simulator.SimAction
	for _, effect := range effects {
		if effect.Action == nil {
			continue
		}
		if effect.Action.TargetType == "ALL_ENEMIES" {
			attack = effect.Action
		} else if effect.Action.TargetType == "TARGET" {
			status = effect.Action
		}
	}
	if attack == nil || attack.Damage != 100 || attack.DamageMult != 1.8 || attack.MultiHit != 3 {
		t.Fatalf("multi-hit did not stay with the AOE damage action: %+v", attack)
	}
	if status == nil || len(status.StatusOps) != 1 || status.StatusOps[0].ID != "BLEED" {
		t.Fatalf("primary-target bleed operation was not retained: %+v", status)
	}
}

func TestAdaptBlockCombatDirectStatusKeepsAuthoredDuration(t *testing.T) {
	spec := &models.GuDictionaryItem{ID: "GU_WORDS", Name: "Nhân Ngôn Khả Úy", Rank: 7}
	block := gameplayeffects.Block{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetAllEnemies},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpStatus, StatusID: "SILENCE", Value: 1, Turns: 2, Chance: 1,
			ApplicationMode: "DIRECT", StackPolicy: "STACK", SuppressLog: true,
		}},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(&models.Combatant{}, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || len(effects[0].Action.StatusOps) != 1 {
		t.Fatalf("expected one adapted direct status, got %+v", effects)
	}
	status := effects[0].Action.StatusOps[0]
	if status.Turns != 2 || status.Chance != 1 || status.ApplicationMode != "DIRECT" || !status.SuppressLog {
		t.Fatalf("direct status must preserve authored duration, chance, and logging at high Gu rank: %+v", status)
	}
}

func TestAdaptBlockCombatManaHealPercentUsesMaxMana(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100, MaxMana: 200, CurrentMana: 50}
	spec := &models.GuDictionaryItem{ID: "GU_STAR_FALL_I", Name: "Tinh Niệm Tụ Hồn", Rank: 3}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "STAR_THOUGHT_RESTORE", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpHeal, SubType: "RESOURCE", Resource: "MANA", Unit: "PERCENT", Value: 30,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Action.ManaRestore != 60 {
		t.Fatalf("mana restore = %+v; want 60 (30%% of MaxMana)", got)
	}
}

func TestAdaptBlockCombatTransferResource(t *testing.T) {
	actor := &models.Combatant{MaxMana: 100, CurrentMana: 20}
	spec := &models.GuDictionaryItem{ID: "GU_COIN_ATTACK_1", Name: "Thâu Tiền Lực Cổ", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "MANA_STEAL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpTransferResource, SubType: "PERCENT_OF_TARGET", Resource: "MANA", Value: 0.25,
			Message: "{actor} lấy {amount} từ {target}.",
		}, {
			Op: gameplayeffects.OpTransferResource, SubType: "PERCENT_OF_TARGET", Resource: "MANA", Value: 0.10,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	actions, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].Action == nil || len(actions[0].Action.ResourceTransfers) != 2 {
		t.Fatalf("resource transfer was not adapted: %+v", actions)
	}
	got := &actions[0].Action.ResourceTransfers[0]
	if got.Resource != "MANA" || got.SubType != "PERCENT_OF_TARGET" || got.Value != 0.25 || got.Message != "{actor} lấy {amount} từ {target}." {
		t.Fatalf("resource transfer fields lost: %+v", got)
	}
	if got := actions[0].Action.ResourceTransfers[1]; got.Value != 0.10 {
		t.Fatalf("second resource transfer lost or reordered: %+v", actions[0].Action.ResourceTransfers)
	}
}

func TestHPTransferOperationRunsBeforeItsDamageAtNativePhase(t *testing.T) {
	actorModel := &models.Combatant{MaxHP: 1000}
	spec := &models.GuDictionaryItem{ID: "GU_THEFT_LIFE_9", Name: "Thâu Sinh", Rank: 9}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "STEAL_LIFE", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 100},
			{Op: gameplayeffects.OpTransferResource, SubType: "PERCENT_OF_MAX_HP", Resource: "HP", Value: 0.15, MinAmount: 30, Phase: "LATE_RIDER", LogAction: "STEAL_LIFE", Message: "[Thâu Đạo - Thâu Sinh] Hút thẳng {amount} HP của {target} chuyển hóa thành sinh lực bản thân!"},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	actions, err := adaptBlockCombatOperations(actorModel, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	actor := &simulator.CombatantAuto{ID: "actor", Name: "Đạo Tặc", Kind: "Player", MaxHP: 1000, HP: 500, Speed: 100,
		Actions: []simulator.SimAction{{ActionType: "GU_SKILL", Name: spec.Name, SourceID: spec.ID, TargetType: "SINGLE", OperationEffects: actions}}}
	target := &simulator.CombatantAuto{ID: "target", Name: "Mục tiêu", Kind: "Enemy", MaxHP: 1000, HP: 250, Speed: 1}
	logs := simulator.Simulate(&simulator.CombatState{Seed: 2, Allies: []*simulator.CombatantAuto{actor}, Enemies: []*simulator.CombatantAuto{target}})
	if actor.HP != 650 || target.HP != 0 {
		t.Fatalf("typed HP transfer + action result = actor %.0f / target %.0f; want 650 / 0", actor.HP, target.HP)
	}
	stealLogs := 0
	for _, log := range logs {
		if log.Action == "STEAL_LIFE" {
			stealLogs++
			if log.Damage != 150 || log.TargetId != target.ID || !strings.Contains(log.Detail, "Hút thẳng 150 HP") {
				t.Fatalf("typed HP transfer log mismatch: %+v", log)
			}
		}
	}
	if stealLogs != 1 {
		t.Fatalf("STEAL_LIFE log count = %d, want 1; logs=%+v", stealLogs, logs)
	}
}

func TestBuildSimActionFromGuRoutesBlockV2ThroughOperationAdapter(t *testing.T) {
	actor := &models.Combatant{Rank: 3, HP: 120, MaxHP: 120, CurrentMana: 100, GridX: 2, GridY: 3}
	spec := &models.GuDictionaryItem{
		ID: "GU_TEST", Name: "Cổ thử", Rank: 3, Type: "Active", Path: []string{"Fire Path"}, BaseManaCost: 1,
		Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
			Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
			Conditions: []gameplayeffects.Condition{{Field: "TARGET_HP_PERCENT", Operator: "LTE", Value: 0.5}},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 24}},
		}}},
	}
	action, err := buildSimActionFromGu(actor, spec, "GU_SKILL")
	if err != nil {
		t.Fatal(err)
	}
	if len(action.OperationEffects) != 1 || action.OperationEffects[0].Action == nil {
		t.Fatalf("Block v2 action was not routed through operation adapter: %+v", action)
	}
	if action.ManaCost <= 0 || action.Damage != 24 || action.OperationEffects[0].Action.Damage != 24 || len(action.OperationEffects[0].Action.OperationConditions) != 1 {
		t.Fatalf("parent cost, damage projection, or conditions are wrong: %+v", action)
	}
}

func TestAdaptBlockCombatOperationsCarriesCounterAdd(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "STACK", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpCounterAdd, Key: "soul_devour_stacks", Delta: 1, Min: 0, Max: 5}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Action == nil || len(got[0].Action.CounterAdds) != 1 {
		t.Fatalf("COUNTER_ADD was not carried into combat action: %+v", got)
	}
	if add := got[0].Action.CounterAdds[0]; add.Key != "soul_devour_stacks" || add.Delta != 1 || add.Min != 0 || add.Max != 5 {
		t.Fatalf("COUNTER_ADD projection = %+v", add)
	}
}

func TestAdaptBlockCombatOperationsCarriesBonusStrike(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "PHANTOM", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpBonusStrike, Chance: 0.45, Value: 0.75, MinDamage: 50,
			LogAction: "BEAST_PHANTOM_STRIKE", Message: "{actor} hits {target} for {damage}.",
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(&models.Combatant{}, &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || len(effects[0].Action.BonusStrikes) != 1 {
		t.Fatalf("BONUS_STRIKE was not projected to the combat action: %+v", effects)
	}
	strike := effects[0].Action.BonusStrikes[0]
	if strike.Chance != 0.45 || strike.DamageModifier != 0.75 || strike.MinDamage != 50 || strike.LogAction != "BEAST_PHANTOM_STRIKE" {
		t.Fatalf("unexpected bonus strike projection: %+v", strike)
	}
}

func TestAdaptBlockCombatOperationsCarriesShieldAbsorption(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "DEVOUR", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 300000,
		}, {
			Op: gameplayeffects.OpDamage, SubType: "MULTIPLIER", Value: 1.5,
		}, {
			Op: gameplayeffects.OpTransferResource, Resource: "HP", SubType: "PERCENT_OF_TARGET_SHIELD", Value: 0.5,
			Phase: "EARLY_RIDER", LogAction: "DEVOUR_HEAVEN_EARTH", Message: "Nuốt {amount} khiên của {target}.",
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(&models.Combatant{MaxHP: 1000}, &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 6}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || effects[0].Action.Damage != 300000 || effects[0].Action.DamageMult != 1.5 {
		t.Fatalf("damage portion did not remain composed with absorption: %+v", effects)
	}
	transfers := effects[0].Action.ResourceTransfers
	if len(transfers) != 1 || transfers[0].SubType != "PERCENT_OF_TARGET_SHIELD" || transfers[0].Value != 0.5 || transfers[0].Phase != "EARLY_RIDER" {
		t.Fatalf("shield absorption was not projected: %+v", transfers)
	}
}

func TestBloodSacrificeHPcostMakesGuActionAvailable(t *testing.T) {
	spec := models.GuDictionaryItem{ID: "GU_BLOOD_ANCESTOR", Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
		Trigger: gameplayeffects.TriggerOnAction, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpHPCost, SubType: "CURRENT_HP_PERCENT", Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}, Value: 0.15, MinCost: 1, MinRemaining: 1, Phase: "PRE_RIDER"}},
	}}}}
	if !hasCombatActiveEffects(&spec) {
		t.Fatal("HP_COST-only Gu should remain selectable as a combat action")
	}
}

func TestAdaptBlockCombatOperationsFailsClosedOnUnsupportedStatModifier(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpStatMod, Stat: "ATTACK", Value: 5}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction)); err == nil {
		t.Fatal("expected turn-based STAT_MOD to fail closed until it has stat semantics")
	}
}

func TestAdaptBlockCombatOperationsFailsClosedOnHealAgainstEnemy(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpHeal, SubType: "FLAT", Value: 10}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction)); err == nil {
		t.Fatal("expected healing an enemy to fail closed")
	}
}

func TestAdaptBlockCombatOperationAppliesTypedGuaranteedHitRule(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpHitRule, SubType: "GUARANTEED"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || effects[0].Action.Bypass || !effects[0].Action.GuaranteedHit {
		t.Fatalf("typed guaranteed hit rule was not projected onto the action: %+v", effects)
	}
}

func TestAdaptBlockCombatOperationComposesLifestealWithDamage(t *testing.T) {
	actor := &models.Combatant{MaxHP: 1000}
	spec := &models.GuDictionaryItem{ID: "GU_BLOOD_BLADE", Name: "Huyết Thao Thiết", Rank: 6}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 100},
			{Op: gameplayeffects.OpHeal, SubType: "LIFESTEAL", Value: 50},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	actions, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].Action == nil {
		t.Fatalf("damage and lifesteal should share one combat operation: %+v", actions)
	}
	if actions[0].Action.Damage != 100 || actions[0].Action.LifeStealPct != 0.5 {
		t.Fatalf("composed damage/lifesteal = damage %.0f, lifesteal %.2f", actions[0].Action.Damage, actions[0].Action.LifeStealPct)
	}
	actorAuto := &simulator.CombatantAuto{
		ID: "blood", Name: "Huyết Cổ Sư", Kind: "Player", Rank: 6,
		MaxHP: 1000, HP: 500, MaxMana: 1000, Mana: 1000, Speed: 100,
		Actions: []simulator.SimAction{{
			ActionType: "GU_SKILL", Name: spec.Name, SourceID: spec.ID, Rank: spec.Rank,
			TargetType: "SINGLE", OperationEffects: actions,
		}},
	}
	target := &simulator.CombatantAuto{
		ID: "target", Name: "Mục tiêu", Kind: "Enemy", Rank: 1,
		MaxHP: 100, HP: 100, Speed: 1,
		Actions: []simulator.SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", TargetType: "SINGLE"}},
	}
	logs := simulator.Simulate(&simulator.CombatState{Seed: 1, Allies: []*simulator.CombatantAuto{actorAuto}, Enemies: []*simulator.CombatantAuto{target}})
	loggedLifesteal := false
	for _, log := range logs {
		if strings.Contains(log.Detail, "hút 50 máu") {
			loggedLifesteal = true
		}
	}
	if !loggedLifesteal || actorAuto.HP != 550 {
		t.Fatalf("typed damage/lifesteal execution did not restore 50 HP (actor HP %.0f, logged=%t)", actorAuto.HP, loggedLifesteal)
	}
}

func TestAdaptBlockCombatDamageMultiplierComposesWithAttainmentScale(t *testing.T) {
	actor := &models.Combatant{MaxHP: 1000}
	spec := &models.GuDictionaryItem{ID: "GU_STAR_FALL_I", Name: "Tinh Quang Lạc Trần", Rank: 6}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpShield, SubType: "PERCENT", Value: 15, Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}},
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 100},
			{Op: gameplayeffects.OpDamage, SubType: "MULTIPLIER", Value: 1.3},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	actions, err := adaptBlockCombatOperations(actor, spec, 0.7, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 || actions[0].Action == nil || actions[1].Action == nil {
		t.Fatalf("self shield and targeted damage should produce two combat operations: %+v", actions)
	}
	if actions[0].Action.TargetType != "SELF" || actions[0].Action.Defense != 150 {
		t.Fatalf("percent shield operation = %+v, want SELF shield 150", actions[0].Action)
	}
	if actions[1].Action.Damage != 100 || actions[1].Action.DamageMult < 0.909999 || actions[1].Action.DamageMult > 0.910001 {
		t.Fatalf("attainment scale × damage multiplier = %.2f, want 0.91", actions[1].Action.DamageMult)
	}
}

func TestAdaptBlockCombatOperationPassesThroughUniqueMechanics(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpUnique, Code: "CALCULATED_STRIKE"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || len(effects[0].Action.Mechanics) != 1 || effects[0].Action.Mechanics[0].ActionCode != "CALCULATED_STRIKE" {
		t.Fatalf("unique mechanic was not preserved on operation child action: %+v", effects)
	}
}

func TestAdaptBlockCombatDisplacementPrimitive(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpDisplace, SubType: "KNOCKBACK", Value: 2}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || len(effects[0].Action.Mechanics) != 1 {
		t.Fatalf("expected one compiled displacement mechanic: %+v", effects)
	}
	mechanic := effects[0].Action.Mechanics[0]
	if mechanic.ActionCode != "DISPLACE" || mechanic.Trigger != "ON_ACTION" || mechanic.Values["direction"] != "KNOCKBACK" || mechanic.Values["steps"] != float64(2) {
		t.Fatalf("displacement operation was not preserved: %+v", mechanic)
	}
}

func TestAdaptBlockCombatHitRuleBypassShield(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpHitRule, SubType: "BYPASS_SHIELD"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || len(effects[0].Action.Mechanics) != 1 {
		t.Fatalf("expected one compiled mechanic: %+v", effects)
	}
	mechanic := effects[0].Action.Mechanics[0]
	if mechanic.ActionCode != "BYPASS_SHIELD" || mechanic.Trigger != "ON_HIT" {
		t.Fatalf("bypass shield mechanic not adapted properly: %+v", mechanic)
	}
}

func TestAdaptBlockCombatTargetAllAllies(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetAllAllies},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpStatus, StatusID: "HASTE", Turns: 2, Value: 30}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || effects[0].Action.TargetType != "ALL_ALLIES" {
		t.Fatalf("target all allies was not preserved: %+v", effects)
	}
}

func TestAdaptBlockCombatAllPeopleUnited(t *testing.T) {
	actor := &models.Combatant{MaxHP: 1000}
	spec := &models.GuDictionaryItem{ID: "GU_ALL_PEOPLE_UNITED_I", Name: "Vạn Chúng Nhất Tâm", Rank: 7}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetAllAllies},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpShield, SubType: "PERCENT", Value: 30, Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}},
			{Op: gameplayeffects.OpStatus, StatusID: "ALL_PEOPLE_UNITED", Value: 35, Turns: 4, StackPolicy: "STACK"},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatalf("failed to adapt all people united: %v", err)
	}
	if len(effects) != 2 {
		t.Fatalf("expected 2 distinct operations due to target override, got %d", len(effects))
	}
}

func TestAdaptBlockCombatBoneThorns(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_BONE_ARMOR_1", Name: "Cốt Giáp Cổ", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "BONE_THORNS", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpStatus, StatusID: "BONE_ARMOR", Turns: 3, StackPolicy: "STACK"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatalf("failed to adapt bone thorns: %v", err)
	}
	if len(effects) != 1 || effects[0].Action == nil || effects[0].Action.TargetType != "SELF" || len(effects[0].Action.StatusOps) != 1 || effects[0].Action.StatusOps[0].ID != "BONE_ARMOR" {
		t.Fatalf("bone thorns not adapted properly: %+v", effects)
	}
}

func TestAdaptBlockCombatDryWoodSpring(t *testing.T) {
	actor := &models.Combatant{MaxHP: 1000}
	spec := &models.GuDictionaryItem{ID: "GU_DRY_WOOD_SPRING_I", Name: "Khô Mộc Phùng Xuân Tiên Cổ", Rank: 6}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetAllAllies},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpHeal, SubType: "FLAT", Value: 300000, Target: &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}},
			{Op: gameplayeffects.OpStatus, StatusID: "DRY_WOOD_SPRING", Turns: 3, StackPolicy: "STACK"},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatalf("failed to adapt dry wood spring: %v", err)
	}
	if len(effects) != 2 {
		t.Fatalf("expected 2 distinct operations due to target override, got %d", len(effects))
	}
}
