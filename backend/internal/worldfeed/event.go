// Package worldfeed defines and publishes coordinate-scoped changes to the
// shared world. It keeps event scope and wire shape consistent across domains.
package worldfeed

import (
	"context"
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
	targetsByScope := make(map[Scope][]uuid.UUID)
	deliveries := make([]Delivery, 0, len(events))
	for _, event := range events {
		if event.EventID == "" {
			event.EventID = uuid.NewString()
		}
		if err := validate(event); err != nil {
			return err
		}
		targets, found := targetsByScope[event.Scope]
		if !found {
			var err error
			targets, err = resolveTargets(event.Scope)
			if err != nil {
				return err
			}
			targetsByScope[event.Scope] = targets
		}
		deliveries = append(deliveries, Delivery{Event: event, Targets: targets})
	}
	return cache.PublishEventQuick(RedisChannel, Envelope{Deliveries: deliveries})
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
