package handlers

import (
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
)

// WorldCellEvent is broadcast when a persistent map cell (boss / high resource
// / safe point) changes state: cleared or respawned. These events are rare and
// tiny, so fan-out is global (like chat) rather than interest-managed.
type WorldCellEvent struct {
	RegionID  string `json:"region_id"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	CellType  string `json:"cell_type"`
	ContentID string `json:"content_id"`
}

const (
	cellEventCleared = "world.cell.cleared"
	cellEventRespawn = "world.cell.respawn"
)

// WorldCellBroadcaster allows injection of websocket targeted broadcast without import cycle.
// Used to push cell snapshots to specific players after movement.
var WorldCellBroadcaster func(channel string, payload []byte, targets map[uuid.UUID]struct{})

func broadcastWorldCell(name string, data WorldCellEvent) {
	eventType := "cleared"
	if name == cellEventRespawn {
		eventType = "respawn"
	}
	payload := map[string]any{
		"type":       eventType,
		"region_id":  data.RegionID,
		"x":          data.X,
		"y":          data.Y,
		"cell_type":  data.CellType,
		"content_id": data.ContentID,
	}
	_ = cache.PublishEventQuick("world:cell", payload)
}

// PushCellsToPlayer resolves non-empty cells in a rect around (cx, cy) and
// pushes the snapshot to a single player via the world:cells WS channel.
// Called after MovePlayerPath so the client receives fresh cell data without
// waiting for the next HTTP poll cycle.
func PushCellsToPlayer(playerID uuid.UUID, cx, cy, halfRange int) {
	if WorldCellBroadcaster == nil {
		return
	}
	EnsureWorldPosLoaded(playerID.String())
	entry, hasScope := worldPosIndex.Get(playerID.String())

	minX := cx - halfRange
	minY := cy - halfRange
	maxX := cx + halfRange
	maxY := cy + halfRange

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= WorldWidth {
		maxX = WorldWidth - 1
	}
	if maxY >= WorldHeight {
		maxY = WorldHeight - 1
	}
	if maxX < minX || maxY < minY {
		return
	}

	// Cap resolved area to prevent abuse
	const maxSide = 900
	if maxX-minX+1 > maxSide {
		maxX = minX + maxSide - 1
	}
	if maxY-minY+1 > maxSide {
		maxY = minY + maxSide - 1
	}

	cells := make([]NearbyCell, 0, 64)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			regionID := DeriveRegionAtOnMap(x, y, entry.MapID)
			if hasScope && entry.InstanceID != "" {
				regionID = entry.Region
			}
			cell := ResolveCellInRegion(regionID, x, y)
			if cell.Type == "Empty" {
				continue
			}
			if isCellClearedRecently(regionID, x, y, clearedCellCooldown) {
				continue
			}
			cells = append(cells, NearbyCell{
				X:       cell.X,
				Y:       cell.Y,
				Type:    cell.Type,
				Content: cell.Content,
				Rank:    cell.Rank,
				SubRank: cell.SubRank,
				Region:  regionID,
			})
		}
	}

	payload := map[string]any{
		"type":  "snapshot",
		"cells": cells,
	}

	targets := map[uuid.UUID]struct{}{playerID: {}}
	WorldCellBroadcaster("world:cells", mustMarshal(payload), targets)
}
