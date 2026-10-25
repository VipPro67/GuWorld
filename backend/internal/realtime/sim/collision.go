package sim

import "math"

const MaxCollisionTiles = 16_777_216

// Collision is a read-only static world collision query. Implementations must
// be immutable while a World is stepping.
type Collision interface {
	Blocked(position Vec2, radius float64) bool
}

// TileCollision also exposes the immutable tile values used for map hashing.
type TileCollision interface {
	Collision
	SolidTile(x, y int) bool
}

// BorderCollision stores a procedural map whose edge tiles alone are solid.
type BorderCollision struct{ width, height int }

func NewBorderCollision(width, height int) (*BorderCollision, error) {
	if !validCollisionDimensions(width, height) {
		return nil, ErrInvalidCollisionMap
	}
	return &BorderCollision{width: width, height: height}, nil
}

func (border *BorderCollision) SolidTile(x, y int) bool {
	return x == 0 || y == 0 || x == border.width-1 || y == border.height-1
}

func (border *BorderCollision) Blocked(position Vec2, radius float64) bool {
	if border == nil {
		return true
	}
	return blockedTiles(border.width, border.height, position, radius, border.SolidTile)
}

// PackedCollision uses one bit per tile for authored collision layers.
type PackedCollision struct {
	width, height int
	tiles         []uint64
}

func NewPackedCollision(width, height int, solidTiles []int) (*PackedCollision, error) {
	if !validCollisionDimensions(width, height) || len(solidTiles) != width*height {
		return nil, ErrInvalidCollisionMap
	}
	packed := &PackedCollision{width: width, height: height, tiles: make([]uint64, (len(solidTiles)+63)/64)}
	for index, tile := range solidTiles {
		if tile != 0 && tile != 1 {
			return nil, ErrInvalidCollisionMap
		}
		if tile == 1 {
			packed.tiles[index/64] |= 1 << (index % 64)
		}
	}
	return packed, nil
}

func (packed *PackedCollision) SolidTile(x, y int) bool {
	index := y*packed.width + x
	return packed.tiles[index/64]&(1<<(index%64)) != 0
}

func (packed *PackedCollision) Blocked(position Vec2, radius float64) bool {
	if packed == nil {
		return true
	}
	return blockedTiles(packed.width, packed.height, position, radius, packed.SolidTile)
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
	if !validCollisionDimensions(width, height) || len(solidTiles) != width*height {
		return nil, ErrInvalidCollisionMap
	}
	return &GridCollision{width: width, height: height, tiles: append([]bool(nil), solidTiles...)}, nil
}

// Blocked reports whether a circular actor at position overlaps a solid tile
// or leaves the map bounds. Touching a solid tile counts as blocked.
func (grid *GridCollision) Blocked(position Vec2, radius float64) bool {
	if grid == nil {
		return true
	}
	return blockedTiles(grid.width, grid.height, position, radius, grid.SolidTile)
}

func (grid *GridCollision) SolidTile(x, y int) bool {
	return grid.tiles[y*grid.width+x]
}

func validCollisionDimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= MaxCollisionTiles/height
}

func blockedTiles(width, height int, position Vec2, radius float64, solid func(int, int) bool) bool {
	if !finite(position.X) || !finite(position.Y) || !finite(radius) || radius < 0 {
		return true
	}
	if radius > float64(max(width, height)) || position.X+radius < 0 || position.Y+radius < 0 || position.X-radius > float64(width) || position.Y-radius > float64(height) {
		return true
	}
	minX := int(math.Floor(position.X - radius))
	maxX := int(math.Floor(position.X + radius))
	minY := int(math.Floor(position.Y - radius))
	maxY := int(math.Floor(position.Y + radius))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= width || y >= height {
				return true
			}
			if !solid(x, y) {
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
