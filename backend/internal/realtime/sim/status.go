package sim

import (
	"math"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// StatusEffect is a server-owned timed status. It carries all scheduling data
// in ticks and is never exposed raw to clients.
type StatusEffect struct {
	ID                  string
	SourceID            EntityID
	ExpireTick          uint64
	DurationTicks       uint64
	PeriodTicks         uint64
	NextTick            uint64
	DamagePerTick       float64
	HealPerTick         float64
	MoveMultiplier      float64
	ManaDrainPerTick    float64
	ThoughtDrainPerTick float64
	Stacks              int
	MaxStacks           int
	StackPolicy         string
	Debuff              bool
	PvPDrCategory       string
}

func (w *World) applyStatusLocked(source, target *Entity, skill combat.Skill) {
	if target == nil || target.Vitals.HP <= 0 || skill.StatusID == "" || skill.StatusDurationTicks == 0 {
		return
	}
	duration := skill.StatusDurationTicks
	if source != nil && source.Kind == EntityPlayer && target.Kind == EntityPlayer {
		duration = applyPvPDurationDR(duration, skill.PvPDrCategory)
	}
	if duration == 0 {
		return
	}
	maxStacks := skill.StatusMaxStacks
	if maxStacks < 1 {
		maxStacks = 1
	}
	moveMultiplier := skill.StatusMoveMultiplier
	if moveMultiplier < 0 {
		moveMultiplier = 0
	} else if moveMultiplier == 0 && !skill.StatusDebuff {
		moveMultiplier = 1
	} else if moveMultiplier <= 0 && skill.StatusDebuff && (skill.StatusID == "FREEZE" || skill.StatusID == "STUN" || skill.StatusID == "ROOT" || skill.StatusID == "PETRIFY") {
		moveMultiplier = 0
	}
	policy := skill.StatusStackPolicy
	if policy == "" {
		policy = combat.StackRefresh
	}
	sourceID := EntityID("")
	if source != nil {
		sourceID = source.ID
	}
	now := w.tick
	for index := range target.Combat.Statuses {
		status := &target.Combat.Statuses[index]
		if status.ID != skill.StatusID {
			continue
		}
		switch policy {
		case combat.StackIgnore:
			return
		case combat.StackReplace:
			status.Stacks = 1
		case combat.StackStack:
			status.Stacks = minInt(status.Stacks+1, maxStacks)
		case combat.StackMax:
			status.Stacks = maxStacks
		default: // REFRESH
			if status.Stacks < 1 {
				status.Stacks = 1
			}
		}
		status.SourceID = sourceID
		status.ExpireTick = now + duration
		status.DurationTicks = duration
		status.PeriodTicks = skill.StatusPeriodTicks
		status.DamagePerTick = skill.StatusDamagePerTick
		status.HealPerTick = skill.StatusHealPerTick
		status.MoveMultiplier = moveMultiplier
		status.ManaDrainPerTick = skill.ManaDrainPerTick
		status.ThoughtDrainPerTick = skill.ThoughtDrainPerTick
		status.MaxStacks = maxStacks
		status.StackPolicy = policy
		status.Debuff = skill.StatusDebuff
		status.PvPDrCategory = skill.PvPDrCategory
		if skill.StatusPeriodTicks > 0 {
			status.NextTick = now + skill.StatusPeriodTicks
		} else {
			status.NextTick = 0
		}
		target.Revision++
		w.emitEvent(combat.Event{
			Type: combat.EventStatusApplied, SourceID: string(sourceID), TargetID: string(target.ID),
			SkillID: skill.ID, DurationTicks: duration, Amount: float64(status.Stacks), Reason: skill.StatusID,
			X: target.Transform.Position.X, Y: target.Transform.Position.Y,
		})
		return
	}
	status := StatusEffect{
		ID: skill.StatusID, SourceID: sourceID,
		ExpireTick: now + duration, DurationTicks: duration,
		PeriodTicks: skill.StatusPeriodTicks, DamagePerTick: skill.StatusDamagePerTick,
		HealPerTick: skill.StatusHealPerTick, MoveMultiplier: moveMultiplier,
		ManaDrainPerTick: skill.ManaDrainPerTick, ThoughtDrainPerTick: skill.ThoughtDrainPerTick,
		Stacks: 1, MaxStacks: maxStacks, StackPolicy: policy,
		Debuff: skill.StatusDebuff, PvPDrCategory: skill.PvPDrCategory,
	}
	if status.PeriodTicks > 0 {
		status.NextTick = now + status.PeriodTicks
	}
	target.Combat.Statuses = append(target.Combat.Statuses, status)
	target.Revision++
	w.emitEvent(combat.Event{
		Type: combat.EventStatusApplied, SourceID: string(sourceID), TargetID: string(target.ID),
		SkillID: skill.ID, DurationTicks: duration, Amount: 1, Reason: skill.StatusID,
		X: target.Transform.Position.X, Y: target.Transform.Position.Y,
	})
}

func (w *World) tickStatuses() {
	for _, entity := range w.entities {
		if len(entity.Combat.Statuses) == 0 || entity.Vitals.HP <= 0 {
			continue
		}
		remaining := entity.Combat.Statuses[:0]
		for index := range entity.Combat.Statuses {
			status := entity.Combat.Statuses[index]
			if w.tick > status.ExpireTick {
				w.emitEvent(combat.Event{
					Type: combat.EventStatusExpired, TargetID: string(entity.ID),
					Reason: status.ID, DurationTicks: status.DurationTicks,
					X: entity.Transform.Position.X, Y: entity.Transform.Position.Y,
				})
				entity.Revision++
				continue
			}
			if status.PeriodTicks > 0 && w.tick >= status.NextTick {
				stacks := status.Stacks
				if stacks < 1 {
					stacks = 1
				}
				source := w.entities[status.SourceID]
				if status.DamagePerTick > 0 {
					w.applyDamageLocked(source, entity, status.DamagePerTick*float64(stacks), status.ID)
				}
				if status.HealPerTick > 0 {
					healVal := status.HealPerTick * float64(stacks)
					if w.hasStatus(entity, "BURN") {
						healVal *= 0.5
					}
					before := entity.Vitals.HP
					entity.Vitals.HP = math.Min(entity.Vitals.MaxHP, entity.Vitals.HP+healVal)
					entity.Revision++
					w.emitEvent(combat.Event{
						Type: combat.EventHealApplied, SourceID: string(status.SourceID), TargetID: string(entity.ID),
						SkillID: status.ID, Amount: entity.Vitals.HP - before, HP: entity.Vitals.HP,
					})
				}
				if status.ManaDrainPerTick > 0 {
					drain := status.ManaDrainPerTick * float64(stacks)
					if entity.Combat.Mana < drain {
						w.emitEvent(combat.Event{
							Type: combat.EventStatusExpired, TargetID: string(entity.ID),
							Reason: status.ID, DurationTicks: status.DurationTicks,
							X: entity.Transform.Position.X, Y: entity.Transform.Position.Y,
						})
						entity.Revision++
						continue
					}
					entity.Combat.Mana -= drain
					entity.Revision++
				}
				if status.ThoughtDrainPerTick > 0 {
					drain := status.ThoughtDrainPerTick * float64(stacks)
					if entity.Combat.Thought < drain {
						w.emitEvent(combat.Event{
							Type: combat.EventStatusExpired, TargetID: string(entity.ID),
							Reason: status.ID, DurationTicks: status.DurationTicks,
							X: entity.Transform.Position.X, Y: entity.Transform.Position.Y,
						})
						entity.Revision++
						continue
					}
					entity.Combat.Thought -= drain
					entity.Revision++
				}
				status.NextTick += status.PeriodTicks
			}
			if entity.Vitals.HP <= 0 {
				break
			}
			remaining = append(remaining, status)
		}
		entity.Combat.Statuses = remaining
	}
}

func (w *World) cleanseDebuffsLocked(target *Entity) int {
	if target == nil {
		return 0
	}
	remaining := target.Combat.Statuses[:0]
	removed := 0
	for _, status := range target.Combat.Statuses {
		if status.Debuff {
			removed++
			w.emitEvent(combat.Event{
				Type: combat.EventStatusExpired, TargetID: string(target.ID),
				Reason: status.ID, DurationTicks: status.DurationTicks,
			})
			continue
		}
		remaining = append(remaining, status)
	}
	target.Combat.Statuses = remaining
	if removed > 0 {
		target.Revision++
	}
	return removed
}

func (w *World) removeStatusLocked(entity *Entity, id string) bool {
	if entity == nil || len(entity.Combat.Statuses) == 0 {
		return false
	}
	remaining := entity.Combat.Statuses[:0]
	removed := false
	for _, status := range entity.Combat.Statuses {
		if status.ID == id {
			removed = true
			w.emitEvent(combat.Event{
				Type: combat.EventStatusExpired, TargetID: string(entity.ID),
				Reason: status.ID, DurationTicks: status.DurationTicks,
				X: entity.Transform.Position.X, Y: entity.Transform.Position.Y,
			})
			continue
		}
		remaining = append(remaining, status)
	}
	entity.Combat.Statuses = remaining
	if removed {
		entity.Revision++
	}
	return removed
}

// castMovementMultiplier is the authored cast-time movement factor. It applies
// during windup (from the pending skill) and during the post-cast recovery
// window. Status effects compose on top of it through moveMultiplier.
func (w *World) castMovementMultiplier(entity *Entity) float64 {
	if entity == nil {
		return 1
	}
	if entity.Combat.Pending != nil {
		return entity.Combat.Pending.Skill.MovementMultiplier()
	}
	if w.tick < entity.Combat.RecoveryUntil {
		return (combat.Skill{MovementPolicy: entity.Combat.RecoveryPolicy}).MovementMultiplier()
	}
	return 1
}

func (w *World) moveMultiplier(entity *Entity) float64 {
	slow := 1.0
	haste := 1.0
	for _, status := range entity.Combat.Statuses {
		if status.MoveMultiplier == 0 || status.ID == "ROOT" || status.ID == "BIND" {
			return 0
		}
		if status.MoveMultiplier < 1.0 && status.MoveMultiplier < slow {
			slow = status.MoveMultiplier
		} else if status.MoveMultiplier > 1.0 && status.MoveMultiplier > haste {
			haste = status.MoveMultiplier
		}
	}
	mult := slow * haste
	if mult < 0.1 && slow > 0 {
		mult = 0.1
	} else if mult > 3.0 {
		mult = 3.0
	}
	return mult
}

func (w *World) isStunned(entity *Entity) bool {
	for _, status := range entity.Combat.Statuses {
		if status.Debuff && (status.ID == "STUN" || status.ID == "FREEZE" || status.ID == "PETRIFY" || status.ID == "FEAR" || status.ID == "SLEEP") {
			return true
		}
		if status.Debuff && status.MoveMultiplier == 0 && status.ID != "ROOT" && status.ID != "BIND" {
			return true
		}
	}
	return false
}

func (w *World) hasStatus(entity *Entity, id string) bool {
	for _, status := range entity.Combat.Statuses {
		if status.ID == id {
			return true
		}
	}
	return false
}

func applyPvPDurationDR(duration uint64, category string) uint64 {
	multiplier := 1.0
	switch category {
	case "HARD_CC":
		multiplier = 0.6
	case "SOFT_CC":
		multiplier = 0.75
	}
	result := uint64(math.Ceil(float64(duration) * multiplier))
	if result < 1 {
		result = 1
	}
	return result
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
