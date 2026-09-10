package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/cache"
)

// WorldCellEvent is broadcast when a persistent map cell (boss / high resource
// / safe point) changes state: cleared or respawned. These events are rare and
// tiny, so fan-out is global (like chat) rather than interest-managed.
type WorldCellEvent struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	CellType  string `json:"cell_type"`
	ContentID string `json:"content_id"`
}

const (
	cellEventCleared = "world.cell.cleared"
	cellEventRespawn = "world.cell.respawn"
)

func broadcastWorldCell(name string, data WorldCellEvent) {
	eventType := "cleared"
	if name == cellEventRespawn {
		eventType = "respawn"
	}
	payload := map[string]any{
		"type":       eventType,
		"x":          data.X,
		"y":          data.Y,
		"cell_type":  data.CellType,
		"content_id": data.ContentID,
	}
	_ = cache.PublishEventQuick("world:cell", payload)
}
