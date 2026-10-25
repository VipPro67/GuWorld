// Package spatial provides a deterministic grid index for bounded zone queries.
package spatial

import (
	"math"
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

type cell struct{ x, y int64 }

type Index struct {
	cellSize float64
	cells    map[cell][]sim.Entity
	entities map[sim.EntityID]sim.Entity
}

func NewIndex(cellSize float64) *Index {
	if !isFinite(cellSize) || cellSize <= 0 {
		panic("spatial index cell size must be finite and positive")
	}
	return &Index{cellSize: cellSize, cells: make(map[cell][]sim.Entity), entities: make(map[sim.EntityID]sim.Entity)}
}

func (index *Index) Rebuild(entities []sim.Entity) {
	clear(index.cells)
	clear(index.entities)
	for _, entity := range entities {
		index.entities[entity.ID] = entity
		key := index.key(entity.Transform.Position)
		index.cells[key] = append(index.cells[key], entity)
	}
	for key := range index.cells {
		sort.Slice(index.cells[key], func(i, j int) bool { return index.cells[key][i].ID < index.cells[key][j].ID })
	}
}

func (index *Index) Entity(id sim.EntityID) (sim.Entity, bool) {
	entity, ok := index.entities[id]
	return entity, ok
}

// Query returns entities within a circular world-unit radius in stable ID order.
// The owner is always included, even if an invalid radius would otherwise omit it.
func (index *Index) Query(center sim.Vec2, radius float64, owner sim.EntityID) []sim.Entity {
	if !isFinite(center.X) || !isFinite(center.Y) || math.Abs(center.X) > sim.MaxWorldCoordinate || math.Abs(center.Y) > sim.MaxWorldCoordinate || !isFinite(radius) || radius < 0 || radius > sim.MaxWorldCoordinate {
		return nil
	}
	minX, maxX := index.key(sim.Vec2{X: center.X - radius, Y: center.Y}).x, index.key(sim.Vec2{X: center.X + radius, Y: center.Y}).x
	minY, maxY := index.key(sim.Vec2{X: center.X, Y: center.Y - radius}).y, index.key(sim.Vec2{X: center.X, Y: center.Y + radius}).y
	if maxX-minX > 256 || maxY-minY > 256 {
		return nil
	}
	radiusSquared := radius * radius
	visible := make(map[sim.EntityID]sim.Entity)
	if entity, ok := index.entities[owner]; ok {
		visible[owner] = entity
	}
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for _, entity := range index.cells[cell{x: x, y: y}] {
				position := entity.Transform.Position
				dx, dy := position.X-center.X, position.Y-center.Y
				if dx*dx+dy*dy <= radiusSquared {
					visible[entity.ID] = entity
				}
			}
		}
	}
	result := make([]sim.Entity, 0, len(visible))
	for _, entity := range visible {
		result = append(result, entity)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (index *Index) key(position sim.Vec2) cell {
	return cell{x: int64(math.Floor(position.X / index.cellSize)), y: int64(math.Floor(position.Y / index.cellSize))}
}

func isFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
