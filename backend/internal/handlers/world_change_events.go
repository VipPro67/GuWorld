package handlers

import (
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/worldfeed"
)

func newWorldChangeEvent(entityType, action, regionID, instanceID string, x, y int, data map[string]any) worldfeed.Event {
	if strings.TrimSpace(regionID) == "" {
		return worldfeed.Event{}
	}
	mapID := worldMapIDForRegion(regionID)
	if mapID == "" {
		mapID = "world_map"
	}
	return worldfeed.Event{
		EntityType: entityType,
		Action:     action,
		Scope: worldfeed.Scope{
			RegionID:   regionID,
			WorldMapID: mapID,
			InstanceID: instanceID,
		},
		X: x, Y: y, Data: data,
	}
}

func publishWorldChange(entityType, action, regionID, instanceID string, x, y int, data map[string]any) {
	event := newWorldChangeEvent(entityType, action, regionID, instanceID, x, y, data)
	if event.EntityType != "" {
		_ = worldfeed.Publish(event)
	}
}

func publishWorldChanges(events []worldfeed.Event) {
	_ = worldfeed.PublishBatch(events)
}

func queueWorldChange(events *[]worldfeed.Event, event worldfeed.Event) {
	if event.EntityType == "" {
		return
	}
	if events == nil {
		_ = worldfeed.Publish(event)
		return
	}
	*events = append(*events, event)
}

// publishWorldDropsChanged runs after the owning transaction commits.
func publishWorldDropsChanged(regionID string, x, y int) {
	publishWorldChange("ground_drop", "changed", regionID, "", x, y, nil)
}

// publishWorldPointsChanged runs after the owning transaction commits.
func publishWorldPointsChanged(regionID string, x, y int) {
	publishWorldChange("resource_point", "changed", regionID, "", x, y, nil)
}

func publishWorldCellChange(action string, data WorldCellEvent) {
	publishWorldChange("map_cell", action, data.RegionID, data.InstanceID, data.X, data.Y, map[string]any{
		"cell_type":  data.CellType,
		"content_id": data.ContentID,
	})
}

func newWorldCellChange(action string, data WorldCellEvent) worldfeed.Event {
	return newWorldChangeEvent("map_cell", action, data.RegionID, data.InstanceID, data.X, data.Y, map[string]any{
		"cell_type":  data.CellType,
		"content_id": data.ContentID,
	})
}
