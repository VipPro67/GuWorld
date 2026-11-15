package handlers

// WorldCellEvent describes a persistent map-cell transition before it is
// converted to a scoped World Change Feed event.
type WorldCellEvent struct {
	RegionID   string `json:"region_id"`
	InstanceID string `json:"map_instance_id,omitempty"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	CellType   string `json:"cell_type"`
	ContentID  string `json:"content_id"`
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
	publishWorldCellChange(eventType, data)
}
