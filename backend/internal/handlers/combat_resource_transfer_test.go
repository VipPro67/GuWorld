package handlers

import (
	"strings"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestTypedTransferResourceMatchesLegacyManaSteal(t *testing.T) {
	message := "[Trộm Đạo - Thâu Chân Nguyên] {actor} thôi động diệu thủ cướp đoạt {amount} Chân Nguyên từ {target} nạp vào bản thân!"
	legacy := simulator.SimAction{
		ActionType: "GU_SKILL", Name: "Thâu Tiền Lực Cổ", Damage: 100,
		Mechanics: []simulator.CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "MANA_STEAL"}},
	}
	blocks := []gameplayeffects.Block{{
		ID: "ACTIVE_SKILL", Trigger: gameplayeffects.TriggerOnAction,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpTransferResource, SubType: "PERCENT_OF_TARGET", Resource: "MANA", Value: 0.25, Message: message},
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 100},
		},
	}}
	compiled, err := (gameplayeffects.Compiler{}).Compile(blocks)
	if err != nil {
		t.Fatal(err)
	}
	adapted, err := adaptBlockCombatOperations(&models.Combatant{ID: "actor", Name: "Người dùng", MaxHP: 1000, MaxMana: 100}, &models.GuDictionaryItem{ID: "GU_COIN_ATTACK_1", Name: "Thâu Tiền Lực Cổ", Rank: 1}, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	typed := simulator.SimAction{ActionType: "GU_SKILL", Name: "Thâu Tiền Lực Cổ", OperationEffects: adapted}

	nativeLogs, nativeActor, nativeTarget := simulateResourceTransferAction(legacy)
	typedLogs, typedActor, typedTarget := simulateResourceTransferAction(typed)
	if nativeActor.Mana != typedActor.Mana || nativeTarget.Mana != typedTarget.Mana {
		t.Fatalf("resource balances differ: native actor=%v target=%v; typed actor=%v target=%v", nativeActor.Mana, nativeTarget.Mana, typedActor.Mana, typedTarget.Mana)
	}
	if nativeActor.Mana != 45 || nativeTarget.Mana != 75 {
		t.Fatalf("unexpected transferred balances: actor=%v target=%v", nativeActor.Mana, nativeTarget.Mana)
	}
	if got := transferDetail(nativeLogs); got != transferDetail(typedLogs) || !strings.Contains(got, "cướp đoạt 25 Chân Nguyên") {
		t.Fatalf("transfer detail differs: native=%q typed=%q", transferDetail(nativeLogs), transferDetail(typedLogs))
	}
}

func simulateResourceTransferAction(action simulator.SimAction) ([]simulator.CombatLog, *simulator.CombatantAuto, *simulator.CombatantAuto) {
	actor := &simulator.CombatantAuto{ID: "actor", Name: "Người dùng", Rank: 1, HP: 1000, MaxHP: 1000, Mana: 20, MaxMana: 100, Speed: 20, GridX: 1, GridY: 1, Actions: []simulator.SimAction{action}}
	target := &simulator.CombatantAuto{ID: "target", Name: "Mục tiêu", Rank: 1, HP: 1, MaxHP: 1, Mana: 100, MaxMana: 100, Speed: 5, GridX: 1, GridY: 1}
	state := &simulator.CombatState{Seed: 7, Allies: []*simulator.CombatantAuto{actor}, Enemies: []*simulator.CombatantAuto{target}}
	logs := simulator.Simulate(state)
	return logs, actor, target
}

func transferDetail(logs []simulator.CombatLog) string {
	for _, log := range logs {
		if marker := "[Trộm Đạo - Thâu Chân Nguyên]"; strings.Contains(log.Detail, marker) {
			return log.Detail[strings.Index(log.Detail, marker):]
		}
	}
	return ""
}
