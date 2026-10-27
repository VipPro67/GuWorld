package gateway

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	"github.com/vippro67/GuWorld/backend/internal/realtime/persistence"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

func TestCombatCheckpointRoundTripRestoresCooldownsStatusAndRecovery(t *testing.T) {
	state := sim.CombatState{
		GCDUntil:                 50,
		BasicAttackCooldownUntil: 40,
		RecoveryUntil:            45,
		RecoveryPolicy:           combat.MovementSlow,
	}
	state.Skills[1] = combat.Skill{ID: "GU_FROST", Slot: 1}
	state.Cooldowns[1] = 70
	state.Statuses = []sim.StatusEffect{{
		ID: "FREEZE", SourceID: "caster", ExpireTick: 30, DurationTicks: 40,
		PeriodTicks: 20, NextTick: 25, MoveMultiplier: 0, Stacks: 1, MaxStacks: 1,
		StackPolicy: combat.StackRefresh, Debuff: true, PvPDrCategory: "HARD_CC",
	}}
	state.Hidden = combat.HiddenPassive{ID: "HIDDEN", RequiredHits: 3, HitCount: 2, NextReadyTick: 60, ICDTicks: 20}

	checkpoint := combatCheckpoint(state, 10, 100)
	if checkpoint == nil {
		t.Fatal("live combat state produced an empty checkpoint")
	}
	if checkpoint.GCDRemainingTicks != 40 || checkpoint.BasicAttackRemainingTicks != 30 ||
		checkpoint.RecoveryRemainingTicks != 35 || checkpoint.CooldownRemainingTicks[1] != 60 {
		t.Fatalf("wrong remaining ticks: %+v", checkpoint)
	}
	if checkpoint.CooldownSkillIDs[1] != "GU_FROST" {
		t.Fatalf("cooldown skill id not captured: %+v", checkpoint)
	}
	if len(checkpoint.Statuses) != 1 || checkpoint.Statuses[0].RemainingTicks != 20 ||
		checkpoint.Statuses[0].NextTickInTicks != 15 || checkpoint.Statuses[0].PvPDrCategory != "HARD_CC" {
		t.Fatalf("wrong status checkpoint: %+v", checkpoint.Statuses)
	}
	if checkpoint.Hidden.HitCount != 2 || checkpoint.Hidden.ReadyRemainingTicks != 50 {
		t.Fatalf("wrong hidden checkpoint: %+v", checkpoint.Hidden)
	}

	restored := sim.CombatState{}
	restored.Skills[1] = combat.Skill{ID: "GU_FROST", Slot: 1}
	restoreCombat(&restored, *checkpoint, 200)
	if restored.GCDUntil != 240 || restored.BasicAttackCooldownUntil != 230 ||
		restored.RecoveryUntil != 235 || restored.RecoveryPolicy != combat.MovementSlow {
		t.Fatalf("wrong restored deadlines: %+v", restored)
	}
	if restored.Cooldowns[1] != 260 {
		t.Fatalf("wrong restored cooldown: %v", restored.Cooldowns[1])
	}
	if len(restored.Statuses) != 1 {
		t.Fatalf("status not restored: %+v", restored.Statuses)
	}
	status := restored.Statuses[0]
	if status.ExpireTick != 220 || status.NextTick != 215 || status.SourceID != "caster" || !status.Debuff || status.MoveMultiplier != 0 {
		t.Fatalf("wrong restored status: %+v", status)
	}
	if restored.Hidden.HitCount != 2 || restored.Hidden.NextReadyTick != 250 {
		t.Fatalf("wrong restored hidden passive: %+v", restored.Hidden)
	}
}

func TestCombatCheckpointSkipsCooldownForReplacedSkillAndDeadPlayer(t *testing.T) {
	state := sim.CombatState{}
	state.Skills[1] = combat.Skill{ID: "GU_OLD", Slot: 1}
	state.Cooldowns[1] = 70
	checkpoint := combatCheckpoint(state, 10, 100)
	if checkpoint == nil {
		t.Fatal("expected a cooldown checkpoint")
	}
	replaced := sim.CombatState{}
	replaced.Skills[1] = combat.Skill{ID: "GU_NEW", Slot: 1}
	restoreCombat(&replaced, *checkpoint, 10)
	if replaced.Cooldowns[1] != 0 {
		t.Fatalf("replaced skill inherited cooldown: %v", replaced.Cooldowns[1])
	}
	if combatCheckpoint(state, 10, 0) != nil {
		t.Fatal("dead player kept a restorable combat checkpoint")
	}
}

func TestCombatCheckpointEmptyStateIsNotPersisted(t *testing.T) {
	if checkpoint := combatCheckpoint(sim.CombatState{}, 10, 100); checkpoint != nil {
		t.Fatalf("idle combat state should not be persisted: %+v", checkpoint)
	}
	if durable := (persistence.CombatState{Version: persistence.CombatStateVersion}); !durable.Empty() {
		t.Fatal("empty durable state reported restorable")
	}
}
