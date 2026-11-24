package simulator

import (
	"math/rand"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func TestApplyResourceTransferMovesManaAndPreservesAuthoredLog(t *testing.T) {
	actor := &CombatantAuto{ID: "actor", Name: "Người dùng", Mana: 95, MaxMana: 100}
	target := &CombatantAuto{ID: "target", Name: "Mục tiêu", Mana: 100, MaxMana: 100}
	logs := []CombatLog{}
	runner := &combatActionRunner{turn: 2, turnLogs: &logs}

	runner.applyResourceTransfer(actor, target, ResourceTransfer{
		Resource: "MANA", SubType: "PERCENT_OF_TARGET", Value: 0.25,
		Message: "[Trộm Đạo - Thâu Chân Nguyên] {actor} lấy {amount} từ {target}.", LogAction: "MANA_STEAL",
	})

	if target.Mana != 75 || actor.Mana != 100 {
		t.Fatalf("transfer balances actor=%v target=%v, want actor=100 target=75", actor.Mana, target.Mana)
	}
	if len(logs) != 1 || logs[0].Action != "MANA_STEAL" || logs[0].TargetId != target.ID || logs[0].Turn != 2 {
		t.Fatalf("transfer log metadata mismatch: %+v", logs)
	}
	want := "[Trộm Đạo - Thâu Chân Nguyên] Người dùng lấy 25 từ Mục tiêu."
	if logs[0].Detail != want {
		t.Fatalf("transfer log = %q, want %q", logs[0].Detail, want)
	}
}

func TestApplyResourceTransferMovesConfiguredHPAtMinimum(t *testing.T) {
	actor := &CombatantAuto{ID: "actor", Name: "Đạo Tặc", MaxHP: 1000, HP: 400}
	target := &CombatantAuto{ID: "target", Name: "Mục tiêu", MaxHP: 100, HP: 20}
	logs := []CombatLog{}
	runner := &combatActionRunner{turn: 3, turnLogs: &logs}
	runner.applyResourceTransfer(actor, target, ResourceTransfer{
		Resource: "HP", SubType: "PERCENT_OF_MAX_HP", Value: 0.15, MinAmount: 30, Phase: "LATE_RIDER",
		Message: "[Thâu Đạo - Thâu Sinh] Hút thẳng {amount} HP của {target} chuyển hóa thành sinh lực bản thân!", LogAction: "STEAL_LIFE",
	})
	if actor.HP != 430 || target.HP != 0 {
		t.Fatalf("HP transfer balances actor=%.0f target=%.0f; want 430/0", actor.HP, target.HP)
	}
	if len(logs) != 1 || logs[0].Action != "STEAL_LIFE" || logs[0].Damage != 30 || logs[0].TargetId != target.ID {
		t.Fatalf("HP transfer log mismatch: %+v", logs)
	}
}

func TestOperationResourceTransferAbsorbsTargetShieldAtEarlyRider(t *testing.T) {
	actor := &CombatantAuto{ID: "actor", Name: "Thực Tu", MaxHP: 1000, HP: 800}
	target := &CombatantAuto{ID: "target", Name: "Mục tiêu", MaxHP: 1000, HP: 1000, VirtualShield: 400}
	logs := []CombatLog{}
	summonCount := 0
	runner := combatActionRunner{state: &CombatState{Allies: []*CombatantAuto{actor}, Enemies: []*CombatantAuto{target}}, rng: rand.New(rand.NewSource(42)), turn: 1, turnLogs: &logs, summonCount: &summonCount}
	parent := &SimAction{ActionType: "GU_SKILL", Name: "Thôn Thiên Địa", TargetType: "SINGLE", OperationEffects: []SimActionOperation{{
		Compiled: gameplayeffects.CompiledOperation{BlockID: "DEVOUR", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpTransferResource},
		Action:   &SimAction{ActionType: "GU_EFFECT_OPERATION", TargetType: "SINGLE", IsEffectOperation: true, ResourceTransfers: []ResourceTransfer{{Resource: "HP", SubType: "PERCENT_OF_TARGET_SHIELD", Value: 0.5, Phase: "EARLY_RIDER", LogAction: "DEVOUR_HEAVEN_EARTH", Message: "Hút {amount} khiên của {target}."}}},
	}}}
	runner.runOne(actor, parent, true, []*CombatantAuto{target})
	if actor.HP != 1000 || target.VirtualShield != 200 {
		t.Fatalf("early shield absorption = actor HP %.0f / target shield %.0f; want 1000 / 200", actor.HP, target.VirtualShield)
	}
	count := 0
	for _, log := range logs {
		if log.Action == "DEVOUR_HEAVEN_EARTH" {
			count++
			if log.TargetId != target.ID || log.Detail != "Hút 200 khiên của Mục tiêu." || log.Damage != 0 {
				t.Fatalf("shield absorption log mismatch: %+v", log)
			}
		}
	}
	if count != 1 {
		t.Fatalf("shield transfer logs = %d, want exactly one; logs=%+v", count, logs)
	}
}
