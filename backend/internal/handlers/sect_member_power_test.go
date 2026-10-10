package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestComputeMemberPowerWithTerritoryUsesPreloadedTerritory(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 2, SubRank: 1},
		BodyState:        models.BodyState{BaseHP: 100},
	}
	territory := SectTerritoryInfo{InTerritory: true, MainHallLvl: 2}

	got := computeMemberPowerWithTerritory(player, territory)
	// Base attack 20 and defense 10 each receive the 14% hall-2 territory buff;
	// base HP contributes 100/20, then rank 2 applies the 1.15 multiplier.
	want := (20*1.14 + 10*1.14 + 100.0/20) * 1.15
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("computeMemberPowerWithTerritory() = %v, want %v", got, want)
	}
}
