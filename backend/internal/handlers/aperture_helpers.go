package handlers

import (
	"database/sql"
)

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
