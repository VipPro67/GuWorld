package simulator

import (
	"fmt"
	"testing"
)

// Chaos Gamble records the cost it charged as a typed settlement. The action
// log consolidation folds the CHAOS_GAMBLE entry away, so the settlement must
// ride the surviving action log — the post-combat hook reads it from there
// instead of parsing the Vietnamese Detail text.
func TestChaosGambleSettlementSurvivesLogConsolidation(t *testing.T) {
	cases := []struct {
		cost       string
		wantCode   string
		wantAmount float64
	}{
		{"aptitude_10", SettlementAptitude10, 10},
		{"lifespan_5", SettlementLifespan5, 5},
		{"luck_50", SettlementLuck50, 50},
	}
	for _, tc := range cases {
		t.Run(tc.cost, func(t *testing.T) {
			ally := newTestCombatant("ally", "Player", 1000, 0)
			enemy := newTestCombatant("enemy", "Enemy", 2000, 0)
			params := fmt.Sprintf(`{"effect_choices":["crit_500"],"cost_choices":[%q]}`, tc.cost)
			ally.Actions = []SimAction{{
				ActionType: "GU_SKILL", Name: "Cổ đánh cược", SourceID: "GU_CHAOS", Rank: 1,
				Damage: 100, DamageMult: 1,
				Mechanics: []CombatMechanic{mech("ON_CAST", "CHAOS_GAMBLE", params)},
			}}

			logs := Simulate(&CombatState{Seed: 11, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
			found := 0
			for _, log := range logs {
				if log.Settlement == nil {
					continue
				}
				found++
				if log.Action != "GU_SKILL" {
					t.Fatalf("settlement rode a %q log; the fold should attach it to the action log", log.Action)
				}
				if log.Settlement.Code != tc.wantCode || log.Settlement.Amount != tc.wantAmount {
					t.Fatalf("settlement = %+v, want code %q amount %v", log.Settlement, tc.wantCode, tc.wantAmount)
				}
			}
			if found == 0 {
				t.Fatal("no consolidated log carried the Chaos Gamble settlement")
			}
		})
	}
}
