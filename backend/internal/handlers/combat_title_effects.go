package handlers

import (
	"database/sql"
	"encoding/json"
)

func equippedTitleUtilityBonus(tx *sql.Tx, playerID string) float64 {
	if tx == nil {
		return 0
	}
	var titleID string
	var earnedJSON []byte
	if err := tx.QueryRow(`SELECT COALESCE(selected_title,''),COALESCE(earned_achievements,'[]'::jsonb) FROM players WHERE id=$1`, playerID).Scan(&titleID, &earnedJSON); err != nil || titleID == "" {
		return 0
	}
	var owned bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM player_achievements WHERE player_id=$1 AND achievement_id=$2)`, playerID, titleID).Scan(&owned); err != nil {
		return 0
	}
	if !owned {
		var legacy []PlayerAchievementData
		if json.Unmarshal(earnedJSON, &legacy) != nil {
			return 0
		}
		for _, entry := range legacy {
			if entry.AchievementID == titleID {
				owned = true
				break
			}
		}
	}
	if !owned {
		return 0
	}
	for _, achievement := range achievementRegistry {
		if achievement.ID == titleID && achievement.TitleEffect != nil && achievement.TitleEffect.UtilityType == "aperture_yield" {
			return achievement.TitleEffect.UtilityPct
		}
	}
	return 0
}
