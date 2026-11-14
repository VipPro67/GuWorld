package simulator

import (
	"math"
	"sort"
)

// InitiativeEntry is the persistent scheduling state for one combatant. Time is
// simulated combat time, independent of wall clock time or client latency.
type InitiativeEntry struct {
	ActorID string  `json:"actor_id"`
	Speed   float64 `json:"speed"`
	Started float64 `json:"started"`
	ReadyAt float64 `json:"ready_at"`
	Order   int     `json:"order"`
}

// InitiativeQueue orders individual activations. Order is assigned once at
// battle start and provides a stable tie break across replay and reconnects.
type InitiativeQueue struct {
	Time    float64           `json:"time"`
	Entries []InitiativeEntry `json:"entries"`
}

// InitiativeInterval has diminishing speed returns while guaranteeing that
// faster combatants act more often. The concrete scale is a balance knob.
func InitiativeInterval(speed float64) float64 {
	return 1 / math.Sqrt(math.Max(1, speed))
}

func NewInitiativeQueue(actors []*CombatantAuto) InitiativeQueue {
	q := InitiativeQueue{Entries: make([]InitiativeEntry, 0, len(actors))}
	for order, actor := range actors {
		if actor == nil {
			continue
		}
		speed := actor.EffectiveSpeed
		if speed <= 0 {
			speed = actor.Speed
		}
		q.Entries = append(q.Entries, InitiativeEntry{
			ActorID: actor.ID, Speed: speed,
			ReadyAt: InitiativeInterval(speed), Order: order,
		})
	}
	return q
}

// Next advances to the next activation. The caller must call Complete after
// resolving the actor's one action, even when control effects skip that action.
func (q *InitiativeQueue) Next() (string, float64, bool) {
	return q.NextExcluding(nil)
}

// NextExcluding advances to the earliest ready actor not present in excluded.
// It is used while a faster PvP player holds a ready activation for a shared
// planning window; the held actor keeps its earned readiness.
func (q *InitiativeQueue) NextExcluding(excluded map[string]bool) (string, float64, bool) {
	if len(q.Entries) == 0 {
		return "", q.Time, false
	}
	sort.SliceStable(q.Entries, func(i, j int) bool {
		if q.Entries[i].ReadyAt == q.Entries[j].ReadyAt {
			return q.Entries[i].Order < q.Entries[j].Order
		}
		return q.Entries[i].ReadyAt < q.Entries[j].ReadyAt
	})
	for _, entry := range q.Entries {
		if excluded[entry.ActorID] {
			continue
		}
		q.Time = math.Max(q.Time, entry.ReadyAt)
		return entry.ActorID, q.Time, true
	}
	return "", q.Time, false
}

func (q *InitiativeQueue) Complete(actorID string) bool {
	for i := range q.Entries {
		entry := &q.Entries[i]
		if entry.ActorID == actorID {
			entry.Started = q.Time
			entry.ReadyAt = q.Time + InitiativeInterval(entry.Speed)
			return true
		}
	}
	return false
}

// Prioritize makes an opening-priority actor act at the queue's current time
// before all other ready actors. It is intended for one-shot battle-start
// effects such as SEIZE_INITIATIVE and does not alter later speed intervals.
func (q *InitiativeQueue) Prioritize(actorID string) bool {
	for i := range q.Entries {
		if q.Entries[i].ActorID != actorID {
			continue
		}
		q.Entries[i].ReadyAt = q.Time - 1e-9
		q.Entries[i].Started = q.Time
		return true
	}
	return false
}

// SetSpeed retains the fraction of an actor's current wait already earned.
// It never changes the action that is already ready at the current time.
func (q *InitiativeQueue) SetSpeed(actorID string, speed float64) bool {
	for i := range q.Entries {
		entry := &q.Entries[i]
		if entry.ActorID != actorID {
			continue
		}
		oldInterval := entry.ReadyAt - entry.Started
		progress := 0.0
		if oldInterval > 0 {
			progress = math.Max(0, math.Min(1, (q.Time-entry.Started)/oldInterval))
		}
		entry.Speed = speed
		entry.Started = q.Time - progress*InitiativeInterval(speed)
		entry.ReadyAt = q.Time + (1-progress)*InitiativeInterval(speed)
		return true
	}
	return false
}

func (q *InitiativeQueue) Remove(actorID string) {
	for i := range q.Entries {
		if q.Entries[i].ActorID == actorID {
			q.Entries = append(q.Entries[:i], q.Entries[i+1:]...)
			return
		}
	}
}
