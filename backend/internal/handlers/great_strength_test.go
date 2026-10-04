package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// TestGreatStrengthRank9PassiveStats verifies that GU_GREAT_STRENGTH_RANK9
// correctly applies flat ATTACK, DEFENSE, HP, and SPEED via ALWAYS_ON PASSIVE_STATS.
func TestGreatStrengthRank9PassiveStats(t *testing.T) {
	effectsJSON := `[{"id":"PASSIVE_STATS","target":{"selector":"SELF"},"trigger":"ALWAYS_ON","operations":[{"op":"STAT_MOD","stat":"ATTACK","value":10000000000,"sub_type":"ADD_FLAT"},{"op":"STAT_MOD","stat":"DEFENSE","value":50000000000,"sub_type":"ADD_FLAT"},{"op":"STAT_MOD","stat":"HP","value":100000000000,"sub_type":"ADD_FLAT"},{"op":"STAT_MOD","stat":"SPEED","value":18,"sub_type":"ADD_FLAT"}]}]`
	var effects models.GuEffects
	if err := effects.Scan(effectsJSON); err != nil {
		t.Fatalf("scan effects: %v", err)
	}

	oldRegistry := models.GuRegistry
	defer func() { models.GuRegistry = oldRegistry }()

	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_GREAT_STRENGTH_RANK9": {
			9: {
				ID:      "GU_GREAT_STRENGTH_RANK9",
				Name:    "Đại Lực Cổ",
				Rank:    9,
				Effects: effects,
			},
		},
	}

	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 9, SubRank: 1},
		BodyState:        models.BodyState{BaseHP: 1000, HP: 1000},
		GuInventory: models.GuInventory{
			{
				InstanceID: "great_strength_inst",
				GuID:       "GU_GREAT_STRENGTH_RANK9",
				Rank:       9,
				Status:     "ACTIVE",
				Equipped:   true,
				Hunger:     900,
			},
		},
	}

	baseStats := GetPassiveStats(&models.Player{
		CultivationState: models.CultivationState{Rank: 9, SubRank: 1},
		BodyState:        models.BodyState{BaseHP: 1000, HP: 1000},
	})

	stats := GetPassiveStats(player)
	if got := stats.Stats["attack"] - baseStats.Stats["attack"]; got != 10000000000 {
		t.Errorf("attack delta = %v, want 10000000000", got)
	}
	if got := stats.Stats["defense"] - baseStats.Stats["defense"]; got != 50000000000 {
		t.Errorf("defense delta = %v, want 50000000000", got)
	}
	if got := stats.Stats["hp"] - baseStats.Stats["hp"]; got != 100000000000 {
		t.Errorf("hp delta = %v, want 100000000000", got)
	}
	if got := stats.Stats["speed"] - baseStats.Stats["speed"]; got != 18 {
		t.Errorf("speed delta = %v, want 18", got)
	}
}
