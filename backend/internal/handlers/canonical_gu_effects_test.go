package handlers

import (
	"encoding/json"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestCanonicalGuBlockV2ActionGeneration(t *testing.T) {
	actor := &models.Combatant{Rank: 6, HP: 1000, MaxHP: 1000, CurrentMana: 100}

	tests := []struct {
		name         string
		id           string
		rank         int
		effectsJSON  string
		expectedCode string
	}{
		{
			name: "Wisdom calculated strike",
			id:   "GU_WISDOM",
			rank: 9,
			effectsJSON: `{"schema_version":2,"blocks":[{"id":"THOUGHT_REGEN_TICK","target":{"selector":"SELF"},"trigger":"ON_TICK","operations":[{"op":"UNIQUE","code":"THOUGHT_REGEN_TICK","params":{"value":100}}]},{"id":"CALCULATED_STRIKE","target":{"selector":"TARGET"},"trigger":"ON_ACTION","operations":[{"op":"UNIQUE","code":"CALCULATED_STRIKE","params":{}}]}]}`,
			expectedCode: "CALCULATED_STRIKE",
		},
		{
			name: "Berserk state",
			id:   "GU_BERSERK",
			rank: 4,
			effectsJSON: `{"schema_version":2,"blocks":[{"id":"ACTIVE_SKILL","target":{"selector":"TARGET"},"trigger":"ON_ACTION","operations":[{"op":"DAMAGE","value":600,"sub_type":"FLAT"}]},{"id":"BERSERK_STATE","target":{"selector":"SELF"},"trigger":"COMBAT_PASSIVE","operations":[{"op":"UNIQUE","code":"BERSERK_STATE","params":{}}]}]}`,
			expectedCode: "BERSERK_STATE",
		},
		{
			name: "Metal Aegis and Armor Break",
			id:   "GU_ANCIENT_BODY_1",
			rank: 1,
			effectsJSON: `{"schema_version":2,"blocks":[{"id":"ACTIVE_SKILL","target":{"selector":"TARGET"},"trigger":"ON_ACTION","operations":[{"op":"SHIELD","value":30,"target":{"selector":"SELF"},"sub_type":"FLAT"},{"op":"UNIQUE","code":"METAL_AEGIS","params":{}},{"op":"UNIQUE","code":"ARMOR_BREAK_HEAVY","params":{}}]}]}`,
			expectedCode: "METAL_AEGIS",
		},
		{
			name: "Lightning Stun and Chain Lightning",
			id:   "GU_BLUE_DRAGON_1",
			rank: 1,
			effectsJSON: `{"schema_version":2,"blocks":[{"id":"ACTIVE_SKILL","target":{"selector":"TARGET"},"trigger":"ON_ACTION","operations":[{"op":"DAMAGE","value":30,"sub_type":"FLAT"},{"op":"UNIQUE","code":"LIGHTNING_STUN_BURST","params":{}},{"op":"UNIQUE","code":"CHAIN_LIGHTNING","params":{}}]}]}`,
			expectedCode: "LIGHTNING_STUN_BURST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var eff models.GuEffects
			if err := json.Unmarshal([]byte(tc.effectsJSON), &eff); err != nil {
				t.Fatalf("failed to unmarshal effects: %v", err)
			}

			spec := &models.GuDictionaryItem{
				ID:           tc.id,
				Name:         tc.name,
				Rank:         tc.rank,
				Effects:      eff,
				BaseManaCost: 1,
			}

			action, err := buildSimActionFromGu(actor, spec, "GU_SKILL")
			if err != nil {
				t.Fatalf("buildSimActionFromGu failed: %v", err)
			}

			found := false
			for _, m := range action.Mechanics {
				if m.ActionCode == tc.expectedCode {
					found = true
					break
				}
			}
			if !found {
				for _, op := range action.OperationEffects {
					if op.Action != nil {
						for _, m := range op.Action.Mechanics {
							if m.ActionCode == tc.expectedCode {
								found = true
								break
							}
						}
					}
				}
			}
			if !found {
				passives := combatMechanicsForSpec(*spec, false)
				for _, m := range passives {
					if m.ActionCode == tc.expectedCode {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("expected mechanic code %s not found in action or passives: %+v", tc.expectedCode, action)
			}
		})
	}
}
