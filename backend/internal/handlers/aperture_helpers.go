package handlers

import (
	"database/sql"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// EvaluateEcosystem computes plant, herbivore, and carnivore metrics
func EvaluateEcosystem(allNodes []models.ApertureResourceNode) (plants, herbivores, carnivores int, overgrazing bool, starvationHerb bool, starvationCarn bool) {
	for _, n := range allNodes {
		if n.IsDestroyed {
			continue
		}
		area := len(n.Cells)
		if area == 0 {
			area = 1
		}
		switch n.NodeType {
		case "FARMING_ZONE":
			plants += area
		case "BREEDING_ZONE":
			herbivores += area
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
