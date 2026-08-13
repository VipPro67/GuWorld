package handlers

import "testing"

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
