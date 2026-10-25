package sim

import "math"

const MaxCollisionTiles = 16_777_216

// Collision is a read-only static world collision query. Implementations must
// be immutable while a World is stepping.
type Collision interface {
	Blocked(position Vec2, radius float64) bool
}

// GridCollision is a row-major tile collision map. World coordinates are in
// tile units, with the origin at the top-left corner of tile (0, 0).
type GridCollision struct {
	width, height int
	tiles         []bool
}

// NewGridCollision copies solidTiles so callers cannot mutate the map while
// simulation is running. A true entry marks a solid tile.
func NewGridCollision(width, height int, solidTiles []bool) (*GridCollision, error) {
	if width <= 0 || height <= 0 || width > MaxCollisionTiles/height || len(solidTiles) != width*height {
		return nil, ErrInvalidCollisionMap
	}
	return &GridCollision{width: width, height: height, tiles: append([]bool(nil), solidTiles...)}, nil
}

// Blocked reports whether a circular actor at position overlaps a solid tile
// or leaves the map bounds. Touching a solid tile counts as blocked.
func (grid *GridCollision) Blocked(position Vec2, radius float64) bool {
	if grid == nil || !finite(position.X) || !finite(position.Y) || !finite(radius) || radius < 0 {
		return true
	}
	if radius > float64(max(grid.width, grid.height)) || position.X+radius < 0 || position.Y+radius < 0 || position.X-radius > float64(grid.width) || position.Y-radius > float64(grid.height) {
		return true
	}
	minX := int(math.Floor(position.X - radius))
	maxX := int(math.Floor(position.X + radius))
	minY := int(math.Floor(position.Y - radius))
	maxY := int(math.Floor(position.Y + radius))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= grid.width || y >= grid.height {
				return true
			}
			if !grid.tiles[y*grid.width+x] {
				continue
			}
			closestX := math.Max(float64(x), math.Min(position.X, float64(x+1)))
			closestY := math.Max(float64(y), math.Min(position.Y, float64(y+1)))
			dx, dy := position.X-closestX, position.Y-closestY
			if dx*dx+dy*dy <= radius*radius {
				return true
			}
		}
	}
	return false
}
