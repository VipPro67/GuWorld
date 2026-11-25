package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func loveGuTestPlayer() *models.Player {
	player := &models.Player{}
	player.AptitudePercent = 50
	player.PrimevalStones = 100
	return player
}

// The Love Gu penalty must come from the typed settlement, not from the
// localized Detail, and it must survive log consolidation (the settlement is
// folded onto the action log instead of keeping a CHAOS_GAMBLE entry).
func TestApplyLoveGuPenaltiesUsesTypedSettlement(t *testing.T) {
	player := loveGuTestPlayer()
	applyLoveGuPenalties(player, []simulator.CombatLog{
		{
			Action:     "GU_SKILL",
			Detail:     "không có hình phạt nào",
			Settlement: &simulator.SettlementEffect{Code: simulator.SettlementAptitude10, Amount: 10},
		},
		{
			Action:     "CHAOS_GAMBLE",
			Detail:     "không có hình phạt nào",
			Settlement: &simulator.SettlementEffect{Code: simulator.SettlementLuck50, Amount: 50},
		},
	})

	if player.AptitudePercent != 40 {
		t.Fatalf("aptitude = %d, want 40 (typed aptitude_10)", player.AptitudePercent)
	}
	if player.PrimevalStones != 50 {
		t.Fatalf("primeval stones = %v, want 50 (typed luck_50)", player.PrimevalStones)
	}
}

func TestApplyLoveGuPenaltiesIgnoresDetailText(t *testing.T) {
	player := loveGuTestPlayer()
	applyLoveGuPenalties(player, []simulator.CombatLog{{
		Action: "CHAOS_GAMBLE",
		Detail: "Trừ 10% Tư chất gốc vĩnh viễn! Trừ 50% Nguyên Thạch hiện tại!",
	}})

	if player.AptitudePercent != 50 {
		t.Fatalf("aptitude = %d, want 50; Detail text must not be parsed", player.AptitudePercent)
	}
	if player.PrimevalStones != 100 {
		t.Fatalf("primeval stones = %v, want 100; Detail text must not be parsed", player.PrimevalStones)
	}
}
