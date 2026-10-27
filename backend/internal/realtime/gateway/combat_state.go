package gateway

import (
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	"github.com/vippro67/GuWorld/backend/internal/realtime/persistence"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

// combatCheckpoint captures the restorable part of a zone player's combat
// state at checkpoint time. Remaining durations are stored as ticks so a
// rejoin inside the same fixed-step clock restores exactly what was left.
func combatCheckpoint(state sim.CombatState, tick uint64, hp float64) *persistence.CombatState {
	if hp <= 0 {
		// A dead character must not carry stun/cooldown timers into a respawn.
		return nil
	}
	checkpoint := persistence.CombatState{Version: persistence.CombatStateVersion}
	checkpoint.GCDRemainingTicks = remainingTicks(state.GCDUntil, tick)
	checkpoint.BasicAttackRemainingTicks = remainingTicks(state.BasicAttackCooldownUntil, tick)
	checkpoint.RecoveryRemainingTicks = remainingTicks(state.RecoveryUntil, tick)
	checkpoint.RecoveryPolicy = state.RecoveryPolicy
	for slot := 1; slot <= 6; slot++ {
		checkpoint.CooldownRemainingTicks[slot] = remainingTicks(state.Cooldowns[slot], tick)
		checkpoint.CooldownSkillIDs[slot] = state.Skills[slot].ID
	}
	for _, status := range state.Statuses {
		if status.ExpireTick <= tick || status.ID == "" {
			continue
		}
		entry := persistence.StatusState{
			ID: status.ID, SourceID: string(status.SourceID),
			RemainingTicks:      status.ExpireTick - tick,
			DurationTicks:       status.DurationTicks,
			PeriodTicks:         status.PeriodTicks,
			DamagePerTick:       status.DamagePerTick,
			HealPerTick:         status.HealPerTick,
			MoveMultiplier:      status.MoveMultiplier,
			ManaDrainPerTick:    status.ManaDrainPerTick,
			ThoughtDrainPerTick: status.ThoughtDrainPerTick,
			Stacks:              status.Stacks,
			MaxStacks:           status.MaxStacks,
			StackPolicy:         status.StackPolicy,
			Debuff:              status.Debuff,
			PvPDrCategory:       status.PvPDrCategory,
		}
		if status.NextTick > tick {
			entry.NextTickInTicks = status.NextTick - tick
		} else if status.PeriodTicks > 0 {
			entry.NextTickInTicks = 1
		}
		checkpoint.Statuses = append(checkpoint.Statuses, entry)
	}
	if state.Hidden.ID != "" {
		checkpoint.Hidden = persistence.HiddenState{
			ID: state.Hidden.ID, HPPercentLTE: state.Hidden.HPPercentLTE,
			RequiredHits: state.Hidden.RequiredHits, Shield: state.Hidden.Shield,
			Heal: state.Hidden.Heal, ICDTicks: state.Hidden.ICDTicks,
			HitCount:            state.Hidden.HitCount,
			ReadyRemainingTicks: remainingTicks(state.Hidden.NextReadyTick, tick),
			Discovered:          state.Hidden.Discovered,
			TriggeredOnce:       state.Hidden.TriggeredOnce,
		}
	}
	if checkpoint.Empty() {
		return nil
	}
	return &checkpoint
}

// restoreCombat applies a durable checkpoint to a freshly admitted zone
// character at the target zone's current tick. Cooldowns are only restored
// when the authored skill still occupies the same hotbar slot; a different
// Gu in that slot inherits no cooldown.
func restoreCombat(target *sim.CombatState, durable persistence.CombatState, tick uint64) {
	if target == nil || durable.Version != persistence.CombatStateVersion {
		return
	}
	target.GCDUntil = deadline(tick, durable.GCDRemainingTicks)
	target.BasicAttackCooldownUntil = deadline(tick, durable.BasicAttackRemainingTicks)
	if durable.RecoveryRemainingTicks > 0 {
		target.RecoveryUntil = deadline(tick, durable.RecoveryRemainingTicks)
		target.RecoveryPolicy = durable.RecoveryPolicy
	}
	for slot := 1; slot <= 6; slot++ {
		if durable.CooldownRemainingTicks[slot] == 0 {
			continue
		}
		if target.Skills[slot].ID == "" || target.Skills[slot].ID != durable.CooldownSkillIDs[slot] {
			continue
		}
		target.Cooldowns[slot] = deadline(tick, durable.CooldownRemainingTicks[slot])
	}
	for _, status := range durable.Statuses {
		if status.ID == "" || status.RemainingTicks == 0 {
			continue
		}
		restored := sim.StatusEffect{
			ID: status.ID, SourceID: sim.EntityID(status.SourceID),
			ExpireTick:          deadline(tick, status.RemainingTicks),
			DurationTicks:       status.DurationTicks,
			PeriodTicks:         status.PeriodTicks,
			DamagePerTick:       status.DamagePerTick,
			HealPerTick:         status.HealPerTick,
			MoveMultiplier:      status.MoveMultiplier,
			ManaDrainPerTick:    status.ManaDrainPerTick,
			ThoughtDrainPerTick: status.ThoughtDrainPerTick,
			Stacks:              status.Stacks,
			MaxStacks:           status.MaxStacks,
			StackPolicy:         status.StackPolicy,
			Debuff:              status.Debuff,
			PvPDrCategory:       status.PvPDrCategory,
		}
		if restored.Stacks < 1 {
			restored.Stacks = 1
		}
		if status.NextTickInTicks > 0 {
			restored.NextTick = deadline(tick, status.NextTickInTicks)
		}
		target.Statuses = append(target.Statuses, restored)
	}
	if durable.Hidden.ID != "" && target.Hidden.ID == "" {
		target.Hidden = combatHidden(durable.Hidden, tick)
	}
}

func combatHidden(state persistence.HiddenState, tick uint64) combat.HiddenPassive {
	return combat.HiddenPassive{
		ID: state.ID, HPPercentLTE: state.HPPercentLTE, RequiredHits: state.RequiredHits,
		Shield: state.Shield, Heal: state.Heal, ICDTicks: state.ICDTicks,
		HitCount: state.HitCount, NextReadyTick: deadline(tick, state.ReadyRemainingTicks),
		Discovered: state.Discovered, TriggeredOnce: state.TriggeredOnce,
	}
}

func remainingTicks(deadlineTick, tick uint64) uint64 {
	if deadlineTick <= tick {
		return 0
	}
	return deadlineTick - tick
}

func deadline(tick, remaining uint64) uint64 {
	if remaining == 0 {
		return 0
	}
	return tick + remaining
}
