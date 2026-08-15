package handlers

// World layout constants for the shared continuous world map.
//
// The world is a single grid of WorldWidth x WorldHeight cells, subdivided
// into a 3x2 grid of large zones (NGU_VUC), each LargeZoneSize x LargeZoneSize.
// Each large zone is further divided arithmetically into small zones
// (tiểu vực) of SubZoneSize x SubZoneSize, which drive deterministic content
// generation and nearby-query batching.
const (
	WorldWidth    = 30000
	WorldHeight   = 20000
	LargeZoneSize = 10000
	LargeZoneCols = 3
	LargeZoneRows = 2
	SubZoneSize   = 1000
)

// HubRegionID is the safe lobby zone (players start here, no monsters spawn).
const HubRegionID = "Southern_Border"

// HubEntryPoint is the default spawn / retreat coordinates when the hub zone
// geometry is not yet loaded.
var HubEntryPoint = struct{ X, Y int }{X: 15000, Y: 10000}

// worldBounds returns the inclusive world bounds.
func worldBounds() (int, int, int, int) {
	return 0, 0, WorldWidth - 1, WorldHeight - 1
}

// inWorld reports whether a coordinate is inside the world grid.
func inWorld(x, y int) bool {
	return x >= 0 && x < WorldWidth && y >= 0 && y < WorldHeight
}
