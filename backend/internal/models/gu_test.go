package models

import "testing"

func TestGuEffectsScan(t *testing.T) {
	var effects GuEffects
	err := effects.Scan([]byte(`{"active":{"damage_flat":25.5,"curve_attack":true}}`))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if effects.Active == nil {
		t.Fatalf("Active is nil")
	}
	if effects.Active.DamageFlat != 25.5 {
		t.Errorf("DamageFlat = %v, want 25.5", effects.Active.DamageFlat)
	}
	if !effects.Active.CurveAttack {
		t.Errorf("CurveAttack = false, want true")
	}
}

func TestGetThienNguyenMultiplier(t *testing.T) {
	tests := []struct {
		playerRank int
		guRank     int
		want       float64
	}{
		{3, 3, 2.0},
		{6, 6, 2.0},
		{3, 4, 8.0}, // same rank < 6: 2 * mortalCost[4](20) / mortalCost[3](5) = 2 * 4 = 8
		{4, 3, 0.5}, // same rank < 6: 2 * mortalCost[3](5) / mortalCost[4](20) = 2 * 0.25 = 0.5
	}
	for _, tt := range tests {
		got := GetThienNguyenMultiplier(tt.playerRank, tt.guRank)
		if got != tt.want {
			t.Errorf("GetThienNguyenMultiplier(%d, %d) = %v; want %v", tt.playerRank, tt.guRank, got, tt.want)
		}
	}
}

func TestGetPlayerManaRegenTickBonus(t *testing.T) {
	guID := "GU_HEAVENLY_ESSENCE_LOTUS_MOCK_TEST"
	if GuRegistry == nil {
		GuRegistry = make(map[string]map[int]GuDictionaryItem)
	}
	GuRegistry[guID] = map[int]GuDictionaryItem{
		3: {
			ID:   guID,
			Name: "Thiên Nguyên Bảo Liên Mock Test",
			Rank: 3,
			Effects: GuEffects{
				Other: []SpecialMechanic{
					{
						EventTrigger: "ON_TICK",
						ActionCode:   "MANA_REGEN_TICK",
						Params:       []byte(`{"value":5.0}`),
					},
				},
			},
		},
	}

	inventory := []GuInstance{
		{
			GuID:     guID,
			Rank:     3,
			Equipped: true,
			Status:   "ACTIVE",
		},
	}

	// 5.0 * tnMultiplier(3 vs 3 is 2.0) = 10.0
	got := GetPlayerManaRegenTickBonus(3, inventory)
	if got != 10.0 {
		t.Errorf("GetPlayerManaRegenTickBonus = %v; want 10.0", got)
	}
}
