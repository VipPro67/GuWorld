package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestLineCellsHorizontal(t *testing.T) {
	cells := lineCells(5, 5, 8, 5)
	want := [][2]int{{6, 5}, {7, 5}, {8, 5}}
	if len(cells) != len(want) {
		t.Fatalf("expected %d cells, got %d: %v", len(want), len(cells), cells)
	}
	for i, c := range want {
		if cells[i] != c {
			t.Fatalf("cell[%d] = %v, want %v", i, cells[i], c)
		}
	}
}

func TestLineCellsVertical(t *testing.T) {
	cells := lineCells(3, 9, 3, 6)
	want := [][2]int{{3, 8}, {3, 7}, {3, 6}}
	if len(cells) != len(want) {
		t.Fatalf("expected %d cells, got %d: %v", len(want), len(cells), cells)
	}
	for i, c := range want {
		if cells[i] != c {
			t.Fatalf("cell[%d] = %v, want %v", i, cells[i], c)
		}
	}
}

func TestLineCellsDiagonal(t *testing.T) {
	cells := lineCells(0, 0, 3, 3)
	want := [][2]int{{1, 1}, {2, 2}, {3, 3}}
	if len(cells) != len(want) {
		t.Fatalf("expected %d cells, got %d: %v", len(want), len(cells), cells)
	}
	for i, c := range want {
		if cells[i] != c {
			t.Fatalf("cell[%d] = %v, want %v", i, cells[i], c)
		}
	}
}

func TestLineCellsSamePosition(t *testing.T) {
	if cells := lineCells(4, 4, 4, 4); len(cells) != 0 {
		t.Fatalf("expected no cells for same start/end, got %v", cells)
	}
}

func TestLineCellsEndsAtTarget(t *testing.T) {
	cells := lineCells(10, 20, 14, 22)
	last := cells[len(cells)-1]
	if last[0] != 14 || last[1] != 22 {
		t.Fatalf("expected last cell to be target, got %v", last)
	}
	// Every step must be adjacent to the previous one (Chebyshev distance 1).
	prevX, prevY := 10, 20
	for _, c := range cells {
		dx := c[0] - prevX
		dy := c[1] - prevY
		if dx < -1 || dx > 1 || dy < -1 || dy > 1 {
			t.Fatalf("non-adjacent step %v from %d,%d", c, prevX, prevY)
		}
		prevX, prevY = c[0], c[1]
	}
}

// TestGenerateCellStableAcrossCalls verifies cell content is deterministic and
// does not re-randomize between calls (no time-window shuffle, no map-order
// dependence). Regression for "monsters keep resetting" on the world map.
func TestGenerateCellStableAcrossCalls(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_test", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999,
			PoolWeights: models.ZonePools{
				CellWeights: map[string]int{"Monster": 25, "Resource": 15, "WildGu": 5, "Empty": 55},
				Resources:   []string{"moon_orchid_petal", "green_bamboo_wine", "jade_stone"},
				WildGus:     []string{"GU_LITTLE_LIGHT", "GU_JADE_SKIN"},
			}},
	}
	zoneReg.spawnPools = map[string][]models.MonsterSpawn{
		"dai_vuc_test": {
			{RegionID: "dai_vuc_test", MonsterID: "monster_wild_boar", Chance: 1.0, SpawnMode: "Random", MinRank: 1, MaxRank: 3, IsActive: true},
		},
	}
	zoneReg.mu.Unlock()

	// Compare many cells across repeated resolutions — content must be identical.
	for i := 0; i < 200; i++ {
		x := 100 + i%50
		y := 200 + i%50
		c1 := ResolveCell(x, y)
		c2 := ResolveCell(x, y)
		if c1.Type != c2.Type || c1.Content != c2.Content {
			t.Fatalf("cell (%d,%d) not stable: %+v vs %+v", x, y, c1, c2)
		}
	}
}
