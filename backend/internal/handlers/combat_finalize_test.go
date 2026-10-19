package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

// Battle-start summons create Minion allies whose IDs are synthetic
// ("<uuid>_summon_0"). finalize must skip them before uuid.MustParse or it
// panics and the combat endpoint answers 500 with an empty body.
func TestIsRealPlayerAllySkipsSyntheticAllies(t *testing.T) {
	playerID := uuid.NewString()

	cases := []struct {
		name string
		ally *simulator.CombatantAuto
		want bool
	}{
		{"player", &simulator.CombatantAuto{ID: playerID, Kind: "Player"}, false},
		{"party member", &simulator.CombatantAuto{ID: uuid.NewString(), Kind: "Ally"}, true},
		{"summon minion", &simulator.CombatantAuto{ID: playerID + "_summon_0", Kind: "Minion"}, false},
		{"wolf minion", &simulator.CombatantAuto{ID: playerID + "_dialang", Kind: "Minion"}, false},
		{"pet", &simulator.CombatantAuto{ID: uuid.NewString(), Kind: "Pet"}, false},
		{"variant npc", &simulator.CombatantAuto{ID: uuid.NewString(), Kind: "VariantNPC"}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		if got := isRealPlayerAlly(tc.ally, playerID); got != tc.want {
			t.Errorf("%s: isRealPlayerAlly = %v, want %v", tc.name, got, tc.want)
		}
	}
}
