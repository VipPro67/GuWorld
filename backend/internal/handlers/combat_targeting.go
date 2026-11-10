package handlers

import (
	"math"
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
		// Fallback prioritizing GridX (1,2,3) then GridY (1,2,3)
		for _, col := range []int{1, 2, 3} {
			for _, targetY := range []int{1, 2, 3} {
				for _, c := range alive {
					if c.GridX == col && c.GridY == targetY {
						return c.GridX, c.GridY, true
					}
				}
			}
		}
		return alive[0].GridX, alive[0].GridY, true
	}

	for col := 1; col <= 3; col++ {
		var colAlive []models.Combatant
		for _, c := range alive {
			if c.GridX == col {
				colAlive = append(colAlive, c)
			}
		}
		if len(colAlive) > 0 {
			// Fallback: prioritize GridY = 1, then 2, then 3
			for _, targetY := range []int{1, 2, 3} {
				for _, c := range colAlive {
					if c.GridY == targetY {
						return c.GridX, c.GridY, true
					}
				}
			}
			return colAlive[0].GridX, colAlive[0].GridY, true
		}
	}

	// Final fallback: prioritize GridX (1,2,3) then GridY (1,2,3)
	for _, col := range []int{1, 2, 3} {
		for _, targetY := range []int{1, 2, 3} {
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
	case "ROW":
		for idx, c := range team {
			if c.HP > 0 && c.GridX == targetX {
				targets = append(targets, idx)
			}
		}
	case "COL", "COLUMN":
		for idx, c := range team {
			if c.HP > 0 && c.GridY == targetY {
				targets = append(targets, idx)
			}
		}
	case "CROSS":
		for idx, c := range team {
			if c.HP <= 0 {
				continue
			}
			dx := int(math.Abs(float64(c.GridX - targetX)))
			dy := int(math.Abs(float64(c.GridY - targetY)))
			if (c.GridX == targetX && c.GridY == targetY) || (dx == 1 && dy == 1) {
				targets = append(targets, idx)
			}
		}
	case "ALL", "ALL_ENEMIES":
		for idx, c := range team {
			if c.HP > 0 {
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
		for _, col := range []int{1, 2, 3} {
			for _, targetY := range []int{1, 2, 3} {
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
