package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestSlotInteractionProjectsCsvChanceAndMode(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		mode       string
		params     string
		thiefRank  int
		targetRank int
		vital      bool
		want       float64
	}{
		{"permanent-base", "PERM_STEAL_GU", "PERMANENT_STEAL", `{"base_chance":0.99,"chance_delta":0.1,"vital_penalty":0.5,"target_selection":"RANDOM"}`, 9, 9, false, 0.99},
		{"permanent-vital", "PERM_STEAL_GU", "PERMANENT_STEAL", `{"base_chance":0.99,"chance_delta":0.1,"vital_penalty":0.5,"target_selection":"RANDOM"}`, 9, 9, true, 0.495},
		{"temporary-rank-delta-vital", "TEMP_STEAL_GU", "TEMPORARY_STEAL", `{"base_chance":0.4,"chance_delta":0.05,"vital_penalty":0.5,"target_selection":"RANDOM","duration":"FULL_COMBAT"}`, 8, 4, true, 0.3},
		{"temporary-vital-before-cap", "TEMP_STEAL_GU", "TEMPORARY_STEAL", `{"base_chance":0.4,"chance_delta":0.05,"vital_penalty":0.5,"target_selection":"RANDOM","duration":"FULL_COMBAT"}`, 20, 1, true, 0.675},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := models.GuDictionaryItem{ID: "TEST_STEAL", Effects: models.GuEffects{Blocks: []effects.Block{{
				ID: "STEAL", Trigger: effects.TriggerOnBattleStart, Target: effects.Target{Selector: effects.TargetSingle},
				Operations: []effects.Operation{{Op: effects.OpSlotInteraction, SubType: tt.mode, Params: []byte(tt.params)}},
			}}}}
			if !hasStealMechanic(spec, tt.code) {
				t.Fatal("typed slot interaction was not discovered by PvP handler")
			}
			if got := slotInteractionChance(spec, tt.code, tt.thiefRank, tt.targetRank, tt.vital); math.Abs(got-tt.want) > 1e-12 {
				t.Fatalf("slot interaction chance = %v, want %v", got, tt.want)
			}
		})
	}
}
