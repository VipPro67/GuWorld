package handlers

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func snapshotEquippedTitleEffect(tx *sql.Tx, playerID string, enemyType string) *models.TitleEffectSnapshot {
	if tx == nil || !isPveMonsterType(enemyType) {
		return nil
	}
	var titleID string
	var earnedJSON []byte
	if err := tx.QueryRow(`SELECT COALESCE(selected_title,''), COALESCE(earned_achievements,'[]'::jsonb) FROM players WHERE id=$1`, playerID).Scan(&titleID, &earnedJSON); err != nil || titleID == "" {
		return nil
	}
	var owned bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM player_achievements WHERE player_id=$1 AND achievement_id=$2)`, playerID, titleID).Scan(&owned); err != nil {
		return nil
	}
	if !owned {
		var legacy []PlayerAchievementData
		if json.Unmarshal(earnedJSON, &legacy) != nil {
			return nil
		}
		for _, entry := range legacy {
			if entry.AchievementID == titleID {
				owned = true
				break
			}
		}
	}
	if !owned {
		return nil
	}
	for _, achievement := range achievementRegistry {
		if achievement.ID == titleID && achievement.TitleEffect != nil {
			return &models.TitleEffectSnapshot{
				AchievementID: titleID,
				MonsterType:   achievement.TitleEffect.PveMonsterType,
				DamagePct:     achievement.TitleEffect.PveDamagePct,
			}
		}
	}
	return nil
}

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

func isPveMonsterType(monsterType string) bool {
	switch strings.ToLower(strings.TrimSpace(monsterType)) {
	case "normal", "elite", "boss", "human cultivator", "humancultivator", "tribulation_boss", "tribulation_monster":
		return true
	default:
		return false
	}
}
