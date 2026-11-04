package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/cache"
)

// WorldChangedEvent tells subscribed UI clients that the world snapshot for a
// region changed. The payload deliberately carries the location instead of the
// full row: the owning screen refetches its existing viewport projection, so
// the event stays tiny and the HTTP/RPC contract is unchanged.
type WorldChangedEvent struct {
	Type     string `json:"type"`
	RegionID string `json:"region_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
}

// publishWorldDropsChanged runs after the transaction commits so subscribers
// never observe a drop row that was rolled back.
func publishWorldDropsChanged(regionID string, x, y int) {
	if regionID == "" {
		return
	}
	_ = cache.PublishEventQuick("world:drops", WorldChangedEvent{
		Type: "changed", RegionID: regionID, X: x, Y: y,
	})
}

// publishWorldPointsChanged runs after the transaction commits for resource
// point seeding/harvest/transplant so the map panel refreshes on demand.
func publishWorldPointsChanged(regionID string, x, y int) {
	if regionID == "" {
		return
	}
	_ = cache.PublishEventQuick("world:points", WorldChangedEvent{
		Type: "changed", RegionID: regionID, X: x, Y: y,
	})
}
