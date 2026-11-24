package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestDeductionFinishChanceTypedAndLegacyDefinitionsMatch(t *testing.T) {
	cases := []struct {
		name   string
		typed  string
		legacy string
		want   float64
	}{
		{
			name:   "authored bounds",
			typed:  `{"schema_version":2,"blocks":[{"id":"DEDUCTION_FINISH_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{"base_chance":0.25,"per_rank_step":0.05,"min_chance":0.1,"max_chance":0.95}}]}]}`,
			legacy: `{"schema_version":2,"blocks":[{"id":"DEDUCTION_FINISH_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"DEDUCTION_INSTANT_FINISH","params":{"base_chance":0.25,"per_rank_step":0.05,"min_chance":0.1,"max_chance":0.95}}]}]}`,
			want:   0.35,
		},
		{
			name:   "legacy defaults",
			typed:  `{"schema_version":2,"blocks":[{"id":"DEDUCTION_FINISH_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"DEDUCTION_FINISH_CHANCE","params":{}}]}]}`,
			legacy: `{"schema_version":2,"blocks":[{"id":"DEDUCTION_FINISH_CHANCE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"DEDUCTION_INSTANT_FINISH","params":{}}]}]}`,
			want:   0.09,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			typedSpec := scanDeductionGu(t, tc.typed)
			legacySpec := scanDeductionGu(t, tc.legacy)
			typedChance, typedOK := deductionFinishChance(typedSpec, 9, 7)
			legacyChance, legacyOK := deductionFinishChance(legacySpec, 9, 7)
			if !typedOK || !legacyOK || typedChance != tc.want || legacyChance != tc.want {
				t.Fatalf("typed=(%v,%t), legacy=(%v,%t), want both %v", typedChance, typedOK, legacyChance, legacyOK, tc.want)
			}
		})
	}
}

func scanDeductionGu(t *testing.T, raw string) models.GuDictionaryItem {
	t.Helper()
	var effects models.GuEffects
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan Gu effect: %v", err)
	}
	return models.GuDictionaryItem{ID: "GU_DEDUCTION_TEST", Rank: 9, Effects: effects}
}
