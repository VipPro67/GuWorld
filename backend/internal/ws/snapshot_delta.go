package ws

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

const (
	SnapshotBaselineTicks = 40
)

// SnapshotDeltaTracker encapsulates delta compression and baseline tracking
// for a single player session. It computes diffs without reflection and
// guarantees tick-monotonic baselines and removals.
type SnapshotDeltaTracker struct {
	baselineID uint64
	baseTick   uint64
	baseline   map[string]protocol.EntityState
	seen       map[string]struct{}
}

// NewSnapshotDeltaTracker creates an empty tracker ready for handshakes.
func NewSnapshotDeltaTracker() *SnapshotDeltaTracker {
	return &SnapshotDeltaTracker{
		baseline: make(map[string]protocol.EntityState),
		seen:     make(map[string]struct{}),
	}
}

// Initialize configures the baseline from the handshake Welcome payload.
func (t *SnapshotDeltaTracker) Initialize(welcome protocol.Welcome) error {
	var baseline protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &baseline); err != nil {
		return errors.New("welcome snapshot is invalid")
	}
	if !baseline.IsBaseline || baseline.BaselineID == 0 || baseline.BaselineID != welcome.BaselineID || baseline.Tick != welcome.ServerTick || (baseline.BaseTick != 0 && baseline.BaseTick != baseline.Tick) {
		return errors.New("welcome snapshot baseline does not match handshake")
	}
	t.baselineID = baseline.BaselineID
	t.baseTick = baseline.Tick
	t.baseline = make(map[string]protocol.EntityState, len(baseline.Entities))
	for _, entity := range baseline.Entities {
		t.baseline[entity.EntityID] = entity
	}
	t.seen = make(map[string]struct{})
	return nil
}

// Reset clears the baseline and seen state.
func (t *SnapshotDeltaTracker) Reset() {
	t.baselineID = 0
	t.baseTick = 0
	t.baseline = make(map[string]protocol.EntityState)
	t.seen = make(map[string]struct{})
}

// BaselineID returns the current baseline ID.
func (t *SnapshotDeltaTracker) BaselineID() uint64 {
	return t.baselineID
}

// BaseTick returns the current base tick.
func (t *SnapshotDeltaTracker) BaseTick() uint64 {
	return t.baseTick
}

// BuildDelta computes an incremental snapshot delta against the current baseline.
// When baseline interval expires, a fresh baseline is published and tracked.
func (t *SnapshotDeltaTracker) BuildDelta(snapshot sim.Snapshot, clientPlayerID string) protocol.Snapshot {
	current := make(map[string]protocol.EntityState, len(snapshot.Entities))
	var ackSeq uint64
	var clientPartyID string
	for _, entity := range snapshot.Entities {
		if string(entity.CharacterID) == clientPlayerID {
			clientPartyID = entity.PartyID
			break
		}
	}
	for _, entity := range snapshot.Entities {
		state := protocolEntityState(entity)
		if state.Stealthed {
			isSelf := state.EntityID == clientPlayerID
			isPartyMember := clientPartyID != "" && state.PartyID != "" && state.PartyID == clientPartyID
			if !isSelf && !isPartyMember {
				continue
			}
		}
		current[state.EntityID] = state
		if string(entity.CharacterID) == clientPlayerID {
			ackSeq = entity.AckSeq
		}
	}

	baselineDue := t.baselineID == 0 || snapshot.Tick >= t.baseTick+SnapshotBaselineTicks
	messageSnapshot := protocol.Snapshot{
		Tick:        snapshot.Tick,
		AckSeq:      ackSeq,
		Entities:    make([]protocol.EntityState, 0, len(current)),
		Projectiles: protocolProjectiles(snapshot.Projectiles),
	}

	if baselineDue {
		t.baselineID++
		t.baseTick = snapshot.Tick
		t.baseline = current
		t.seen = make(map[string]struct{})
		messageSnapshot.BaselineID = t.baselineID
		messageSnapshot.BaseTick = snapshot.Tick
		messageSnapshot.IsBaseline = true
		for _, entity := range snapshot.Entities {
			if st, ok := current[string(entity.ID)]; ok {
				messageSnapshot.Entities = append(messageSnapshot.Entities, st)
			}
		}
		return messageSnapshot
	}

	messageSnapshot.BaselineID = t.baselineID
	messageSnapshot.BaseTick = t.baseTick
	for _, entity := range snapshot.Entities {
		state, ok := current[string(entity.ID)]
		if !ok {
			continue
		}
		baseState, existedAtBaseline := t.baseline[state.EntityID]
		_, previouslySent := t.seen[state.EntityID]
		if !existedAtBaseline || !entityStateEqual(state, baseState) || previouslySent {
			messageSnapshot.Entities = append(messageSnapshot.Entities, state)
			t.seen[state.EntityID] = struct{}{}
		}
	}

	removed := make(map[string]struct{})
	for entityID := range t.baseline {
		if _, stillVisible := current[entityID]; !stillVisible {
			removed[entityID] = struct{}{}
		}
	}
	for entityID := range t.seen {
		if _, stillVisible := current[entityID]; !stillVisible {
			removed[entityID] = struct{}{}
		}
	}
	for entityID := range removed {
		messageSnapshot.Removed = append(messageSnapshot.Removed, entityID)
	}
	sort.Strings(messageSnapshot.Removed)
	return messageSnapshot
}

// entityStateEqual performs an explicit zero-reflection comparison of two EntityStates.
func entityStateEqual(a, b protocol.EntityState) bool {
	if a.EntityID != b.EntityID ||
		a.Revision != b.Revision ||
		a.Kind != b.Kind ||
		a.PartyID != b.PartyID ||
		a.PvPEnabled != b.PvPEnabled ||
		a.X != b.X ||
		a.Y != b.Y ||
		a.HP != b.HP ||
		a.MaxHP != b.MaxHP ||
		a.Shield != b.Shield ||
		a.Mana != b.Mana ||
		a.MaxMana != b.MaxMana ||
		a.TemplateID != b.TemplateID ||
		a.Name != b.Name ||
		a.AIState != b.AIState ||
		a.AggroTarget != b.AggroTarget ||
		a.CastSkillID != b.CastSkillID ||
		a.CastEndTick != b.CastEndTick ||
		a.CastWindup != b.CastWindup ||
		a.FormActive != b.FormActive ||
		a.Stealthed != b.Stealthed ||
		a.Flying != b.Flying {
		return false
	}
	if len(a.Statuses) != len(b.Statuses) {
		return false
	}
	for i := range a.Statuses {
		if a.Statuses[i] != b.Statuses[i] {
			return false
		}
	}
	if len(a.Cooldowns) != len(b.Cooldowns) {
		return false
	}
	for i := range a.Cooldowns {
		if a.Cooldowns[i] != b.Cooldowns[i] {
			return false
		}
	}
	return true
}
