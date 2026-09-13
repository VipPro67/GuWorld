package handlers

import (
	"database/sql"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func ArePathsConflicting(path1, path2 string) bool {
	p1 := strings.ToLower(strings.TrimSpace(path1))
	p2 := strings.ToLower(strings.TrimSpace(path2))
	if p1 == p2 {
		return false
	}
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	// e.g. "earth path:wood path"
	conflicts := map[string]bool{
		"fire path:water path": true,
		"metal path:wood path": true,
		"earth path:wood path": true, // Wood roots break earth
		"fire path:ice path":   true,
	}
	return conflicts[p1+":"+p2]
}

// ChebyshevDistance returns Chebyshev distance between two coordinates
func ChebyshevDistance(x1, y1, x2, y2 int) int {
	dx := x1 - x2
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y2
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// CheckNodeConflict returns true if a node is in conflict with any other node
func CheckNodeConflict(node models.ApertureResourceNode, allNodes []models.ApertureResourceNode) bool {
	for _, other := range allNodes {
		if other.ID == node.ID || other.IsDestroyed {
			continue
		}
		if ChebyshevDistance(node.X, node.Y, other.X, other.Y) <= 2 {
			if ArePathsConflicting(node.Path, other.Path) {
				return true
			}
		}
	}
	return false
}

// EvaluateEcosystem computes plant, herbivore, and carnivore metrics
func EvaluateEcosystem(allNodes []models.ApertureResourceNode) (plants, herbivores, carnivores int, overgrazing bool, starvationHerb bool, starvationCarn bool) {
	for _, n := range allNodes {
		if n.IsDestroyed {
			continue
		}
		switch n.NodeType {
		case "MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS",
			"TRI_FARM", "TRI_FARM_ADV", "HOA_FARM", "TIN_FARM", "AM_FARM", "DOC_FARM",
			"BAC_FARM", "QUANG_FARM", "KIEM_FARM", "THUY_FARM", "CHUA_FARM", "CAM_FARM",
			"KIM_FARM", "VANG_FARM", "HUYET_FARM", "THAT_FARM", "THAY_THUAT_FARM", "LUAT_FARM",
			"THUAT_FARM", "THOI_THUAT_FARM", "LUC_FARM", "THOI_FARM", "PHONG_FARM",
			"NO_FARM", "NO_FARM_ADV":
			plants++
		case "SPIRIT_DEER_PASTURE":
			herbivores++
		case "SILVER_WOLF_DEN", "FEATHERMAN_CANYON", "HAIRY_MAN_SETTLEMENT", "ROCKMAN_TRIBE":
			carnivores++
		}
	}

	starvationHerb = herbivores > 0 && plants == 0
	starvationCarn = carnivores > 0 && herbivores == 0
	overgrazing = herbivores > plants && carnivores == 0 && plants > 0
	return
}

func updateVariantHumanOnNodeChange(tx *sql.Tx, apertureID string, nodeType string, change int) error {
	var race string
	switch nodeType {
	case "HAIRY_MAN_SETTLEMENT":
		race = "HAIRY_MAN"
	case "ROCKMAN_TRIBE":
		race = "ROCKMAN"
	case "FEATHERMAN_CANYON":
		race = "FEATHERMAN"
	default:
		return nil
	}

	// Update population
	_, err := tx.Exec(`
		INSERT INTO aperture_variant_humans (aperture_id, race, population, happiness, oppression, rebellion_rate, labor_intensity)
		VALUES ($1, $2, GREATEST(0, $3), 70.0, 0.0, 0.0, 'NORMAL')
		ON CONFLICT (aperture_id, race) 
		DO UPDATE SET population = GREATEST(0, aperture_variant_humans.population + $3), updated_at = NOW()
	`, apertureID, race, change)
	return err
}
