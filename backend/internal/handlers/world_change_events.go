package handlers

import (
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/worldfeed"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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

// setWorldSnapshotRevision tags a scoped HTTP snapshot with the feed position
// the client can use to detect duplicates and gaps in subsequent deltas.
func setWorldSnapshotRevision(c *gin.Context, scope worldfeed.Scope) {
	setWorldRevisionHeaders(c, scope, "X-World")
}

func beginWorldSnapshotRevision(c *gin.Context, scope worldfeed.Scope) {
	setWorldRevisionHeaders(c, scope, "X-World-Start")
}

func setWorldRevisionHeaders(c *gin.Context, scope worldfeed.Scope, prefix string) {
	if scope.RegionID == "" {
		return
	}
	if scope.WorldMapID == "" {
		scope.WorldMapID = "world_map"
	}
	epoch, revision, err := worldfeed.CurrentSequence(scope)
	if err != nil {
		return
	}
	if prefix == "X-World" {
		c.Header("X-World-Revision", strconv.FormatInt(revision, 10))
		c.Header("X-World-Epoch", epoch)
		c.Header("X-World-Revision-Region", scope.RegionID)
		c.Header("X-World-Revision-Map", scope.WorldMapID)
		c.Header("X-World-Revision-Instance", scope.InstanceID)
	} else {
		c.Header(prefix+"-Revision", strconv.FormatInt(revision, 10))
		c.Header(prefix+"-Epoch", epoch)
	}
	exposed := c.Writer.Header().Get("Access-Control-Expose-Headers")
	for _, header := range []string{"X-World-Revision", "X-World-Epoch", "X-World-Revision-Region", "X-World-Revision-Map", "X-World-Revision-Instance", "X-World-Start-Revision", "X-World-Start-Epoch"} {
		if !strings.Contains(strings.ToLower(exposed), strings.ToLower(header)) {
			if exposed != "" {
				exposed += ", "
			}
			exposed += header
		}
	}
	c.Writer.Header().Set("Access-Control-Expose-Headers", exposed)
}

func worldMonsterChangeEvent(action string, monster WorldMonster) worldfeed.Event {
	x, y := int(math.Round(monster.X)), int(math.Round(monster.Y))
	return newWorldChangeEvent("world_monster", action, monster.RegionID, "", x, y, map[string]any{"actor": monster})
}

func worldMonsterActionFromAdmin(event string) string {
	return strings.TrimPrefix(event, "monster_")
}

func publishWorldChange(entityType, action, regionID, instanceID string, x, y int, data map[string]any) {
	event := newWorldChangeEvent(entityType, action, regionID, instanceID, x, y, data)
	if event.EntityType != "" {
		publishWorldChanges([]worldfeed.Event{event})
	}
}

func publishWorldChanges(events []worldfeed.Event) {
	if err := worldfeed.PublishBatch(events); err != nil {
		logger.S.Warnf("publish %d world changes: %v", len(events), err)
	}
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
