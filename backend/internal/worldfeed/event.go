// Package worldfeed defines and publishes coordinate-scoped changes to the
// shared world. It keeps event scope and wire shape consistent across domains.
package worldfeed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const RedisChannel = "world:change"

// Scope identifies the world layer that owns a coordinate. InstanceID is
// empty for public world areas and set for private or shared instances.
type Scope struct {
	RegionID   string `json:"region_id"`
	WorldMapID string `json:"world_map_id"`
	InstanceID string `json:"map_instance_id,omitempty"`
}

// Sequence ties one exact scope to its Redis epoch and latest feed revision.
type Sequence struct {
	Scope    Scope  `json:"scope"`
	Epoch    string `json:"epoch"`
	Revision int64  `json:"revision"`
}

// Event describes one committed change at a world coordinate. Data is optional
// when the client should refresh its scoped snapshot instead of applying a
// direct patch.
type Event struct {
	EventID    string         `json:"event_id"`
	EntityType string         `json:"entity_type"`
	Action     string         `json:"action"`
	Scope      Scope          `json:"scope"`
	X          int            `json:"x"`
	Y          int            `json:"y"`
	Data       map[string]any `json:"data,omitempty"`
	Epoch      string         `json:"epoch,omitempty"`
	Revision   int64          `json:"revision,omitempty"`
}

// Delivery pairs one committed event with its audience. Target IDs stay inside
// Redis; clients receive only Event.
type Delivery struct {
	Event   Event       `json:"event"`
	Targets []uuid.UUID `json:"targets"`
}

// Envelope carries committed events and their audiences between backend
// instances. Events sharing a scope reuse one audience lookup.
type Envelope struct {
	Deliveries []Delivery `json:"deliveries"`
}

// Publish validates and publishes an event after its owning transaction has
// committed. Redis Pub/Sub is intentionally best-effort; snapshots remain the
// durable source of truth.
func Publish(event Event) error {
	return PublishBatch([]Event{event})
}

// PublishBatch resolves each unique scope once and publishes the events in one
// Redis message, preserving the caller's event order.
func PublishBatch(events []Event) error {
	if len(events) == 0 {
		return nil
	}
	scopeKeys := make(map[Scope]string)
	scopeCounts := make(map[string]int)
	normalizedEvents := make([]Event, len(events))
	for i, event := range events {
		if event.EventID == "" {
			event.EventID = uuid.NewString()
		}
		if err := validate(event); err != nil {
			return err
		}
		scopeKey := scopeKeys[event.Scope]
		if scopeKey == "" {
			scopeKey = RevisionKey(event.Scope)
			scopeKeys[event.Scope] = scopeKey
		}
		scopeCounts[scopeKey]++
		normalizedEvents[i] = event
	}
	targetsByScope := make(map[Scope][]uuid.UUID)
	for _, event := range normalizedEvents {
		targets, found := targetsByScope[event.Scope]
		if !found {
			var err error
			targets, err = resolveTargets(event.Scope)
			if err != nil {
				return err
			}
			targetsByScope[event.Scope] = targets
		}
	}
	ranges, err := cache.AllocateWorldSequenceRanges(scopeCounts)
	if err != nil {
		return err
	}
	nextRevision := make(map[string]int64, len(ranges))
	for scopeKey, sequence := range ranges {
		nextRevision[scopeKey] = sequence.FirstRevision
	}
	deliveries := make([]Delivery, 0, len(events))
	for i := range normalizedEvents {
		event := normalizedEvents[i]
		scopeKey := scopeKeys[event.Scope]
		sequence := ranges[scopeKey]
		event.Epoch = sequence.Epoch
		event.Revision = nextRevision[scopeKey]
		nextRevision[scopeKey]++
		deliveries = append(deliveries, Delivery{Event: event, Targets: targetsByScope[event.Scope]})
	}
	return cache.PublishEventQuick(RedisChannel, Envelope{Deliveries: deliveries})
}

// RevisionKey returns the stable Redis key suffix for an exact region/map/
// instance scope without embedding user-controlled identifiers in Redis keys.
func RevisionKey(scope Scope) string {
	if scope.WorldMapID == "" {
		scope.WorldMapID = "world_map"
	}
	encoded, _ := json.Marshal(scope)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// CurrentSequence reads the revision attached to snapshots of this scope.
func CurrentSequence(scope Scope) (string, int64, error) {
	return cache.WorldSequence(RevisionKey(scope))
}

// CurrentSequences reads the cursor vector for a compound snapshot.
func CurrentSequences(scopes []Scope) ([]Sequence, error) {
	keys := make([]string, len(scopes))
	for i, scope := range scopes {
		keys[i] = RevisionKey(scope)
	}
	values, err := cache.WorldSequences(keys)
	if err != nil {
		return nil, err
	}
	sequences := make([]Sequence, 0, len(scopes))
	seen := make(map[Scope]struct{}, len(scopes))
	for _, scope := range scopes {
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		value := values[RevisionKey(scope)]
		sequences = append(sequences, Sequence{Scope: scope, Epoch: value.Epoch, Revision: value.FirstRevision})
	}
	return sequences, nil
}

// DecodeEnvelope validates an internal Redis message and returns its audience
// as a set for targeted delivery by each backend's local UI hub.
func DecodeEnvelope(payload []byte) ([]Delivery, error) {
	var envelope Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Deliveries) == 0 {
		return nil, errors.New("world-change envelope has no deliveries")
	}
	for _, delivery := range envelope.Deliveries {
		if err := validate(delivery.Event); err != nil {
			return nil, err
		}
	}
	return envelope.Deliveries, nil
}

func resolveTargets(scope Scope) ([]uuid.UUID, error) {
	if db.DB == nil {
		return nil, errors.New("database unavailable while resolving world-change audience")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	rows, err := db.DB.QueryContext(ctx, `
		WITH RECURSIVE region_scope(id) AS (
			SELECT $1::varchar
			UNION
			SELECT child.id FROM regions child JOIN region_scope parent ON child.parent_id = parent.id
		)
		SELECT p.id
		FROM players p
		LEFT JOIN region_world_maps rwm ON rwm.region_id = p.current_region
		WHERE p.current_region IN (SELECT id FROM region_scope)
		  AND COALESCE(rwm.world_map_id, 'world_map') = $2
		  AND COALESCE(p.map_instance_id::text, '') = $3
		  AND (p.last_activity >= NOW() - INTERVAL '5 minutes'
		       OR p.last_mana_update >= NOW() - INTERVAL '5 minutes')
	`, scope.RegionID, scope.WorldMapID, scope.InstanceID)
	if err != nil {
		logger.S.Warnf("resolve world-change audience: %v", err)
		return nil, err
	}
	defer rows.Close()

	targets := make([]uuid.UUID, 0)
	for rows.Next() {
		var playerID uuid.UUID
		if err := rows.Scan(&playerID); err != nil {
			logger.S.Warnf("scan world-change audience: %v", err)
			return nil, err
		}
		targets = append(targets, playerID)
	}
	if err := rows.Err(); err != nil {
		logger.S.Warnf("read world-change audience: %v", err)
		return nil, err
	}
	return targets, nil
}

func validate(event Event) error {
	if strings.TrimSpace(event.EventID) == "" {
		return errors.New("world change requires an event id")
	}
	if strings.TrimSpace(event.EntityType) == "" || strings.TrimSpace(event.Action) == "" {
		return errors.New("world change requires entity type and action")
	}
	if strings.TrimSpace(event.Scope.RegionID) == "" || strings.TrimSpace(event.Scope.WorldMapID) == "" {
		return errors.New("world change requires region and world map scope")
	}
	if event.Scope.RegionID == "aperture_interior" && strings.TrimSpace(event.Scope.InstanceID) == "" {
		return errors.New("aperture world change requires an instance scope")
	}
	return nil
}
