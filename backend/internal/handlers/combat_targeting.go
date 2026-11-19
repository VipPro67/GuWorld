package handlers

import (
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func ResolveSingleTarget(team []models.Combatant, prefX, prefY int, hasBypass bool) (int, int, bool) {
	var alive []models.Combatant
	for _, c := range team {
		if c.HP > 0 {
			alive = append(alive, c)
		}
	}
	if len(alive) == 0 {
		return 0, 0, false
	}

	if hasBypass {
		for _, c := range alive {
			if c.GridX == prefX && c.GridY == prefY {
				return prefX, prefY, true
			}
		}
		// Fallback prioritizing front-to-back, then top-to-bottom.
		for col := 1; col <= 5; col++ {
			for targetY := 1; targetY <= 5; targetY++ {
				for _, c := range alive {
					if c.GridX == col && c.GridY == targetY {
						return c.GridX, c.GridY, true
					}
				}
			}
		}
		return alive[0].GridX, alive[0].GridY, true
	}

	for col := 1; col <= 5; col++ {
		var colAlive []models.Combatant
		for _, c := range alive {
			if c.GridX == col {
				colAlive = append(colAlive, c)
			}
		}
		if len(colAlive) > 0 {
			// Fallback: prioritize the topmost lane.
			for targetY := 1; targetY <= 5; targetY++ {
				for _, c := range colAlive {
					if c.GridY == targetY {
						return c.GridX, c.GridY, true
					}
				}
			}
			return colAlive[0].GridX, colAlive[0].GridY, true
		}
	}

	// Final fallback: prioritize front-to-back, then top-to-bottom.
	for col := 1; col <= 5; col++ {
		for targetY := 1; targetY <= 5; targetY++ {
			for _, c := range alive {
				if c.GridX == col && c.GridY == targetY {
					return c.GridX, c.GridY, true
				}
			}
		}
	}
	return alive[0].GridX, alive[0].GridY, true
}

func ResolveTargets(team []models.Combatant, targetType string, targetX, targetY int, hasBypass bool) []int {
	var targets []int

	switch strings.ToUpper(strings.TrimSpace(targetType)) {
	case "ROW", "COL", "COLUMN", "CROSS", "ALL", "ALL_ENEMIES":
		area := targetAreaCells(targetType, targetX, targetY)
		for idx, c := range team {
			if _, ok := area[[2]int{c.GridX, c.GridY}]; ok && c.HP > 0 {
				targets = append(targets, idx)
			}
		}
	default:
		resX, resY, found := ResolveSingleTarget(team, targetX, targetY, hasBypass)
		if found {
			for idx, c := range team {
				if c.HP > 0 && c.GridX == resX && c.GridY == resY {
					targets = append(targets, idx)
					break
				}
			}
		}
	}

	if len(targets) == 0 {
		found := false
		for col := 1; col <= 5; col++ {
			for targetY := 1; targetY <= 5; targetY++ {
				for idx, c := range team {
					if c.HP > 0 && c.GridX == col && c.GridY == targetY {
						targets = append(targets, idx)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			for idx, c := range team {
				if c.HP > 0 {
					targets = append(targets, idx)
					break
				}
			}
		}
	}

	return targets
}

func targetAreaCells(targetType string, targetX, targetY int) map[[2]int]struct{} {
	cells := make(map[[2]int]struct{}, 9)
	add := func(x, y int) {
		if x >= 1 && x <= 5 && y >= 1 && y <= 5 {
			cells[[2]int{x, y}] = struct{}{}
		}
	}
	start := func(center int) int { return min(3, max(1, center-1)) }
	switch strings.ToUpper(strings.TrimSpace(targetType)) {
	case "ROW":
		for y := start(targetY); y < start(targetY)+3; y++ {
			add(targetX, y)
		}
	case "COL", "COLUMN":
		for x := start(targetX); x < start(targetX)+3; x++ {
			add(x, targetY)
		}
	case "CROSS":
		for _, dy := range []int{-1, 1} {
			preferredStart := start(targetX)
			for distance := 0; distance < 3; distance++ {
				for _, startX := range []int{preferredStart - distance, preferredStart + distance} {
					if startX < 1 || startX > 3 || startX > targetX || startX+2 < targetX {
						continue
					}
					startY := targetY - (targetX-startX)*dy
					if startY < 1 || startY > 5 || startY+2*dy < 1 || startY+2*dy > 5 {
						continue
					}
					for offset := 0; offset < 3; offset++ {
						add(startX+offset, startY+offset*dy)
					}
					return cells
				}
			}
		}
	case "ALL", "ALL_ENEMIES":
		for x := start(targetX); x < start(targetX)+3; x++ {
			for y := start(targetY); y < start(targetY)+3; y++ {
				add(x, y)
			}
		}
	default:
		add(targetX, targetY)
	}
	return cells
}

func findCombatant(combatState *models.CombatState, id string) (*models.Combatant, bool, int) {
	for idx, c := range combatState.Allies {
		if c.ID == id {
			return &combatState.Allies[idx], true, idx
		}
	}
	for idx, c := range combatState.Enemies {
		if c.ID == id {
			return &combatState.Enemies[idx], false, idx
		}
	}
	return nil, false, -1
}
