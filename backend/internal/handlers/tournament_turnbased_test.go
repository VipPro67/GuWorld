package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestTournamentWinningSidePrefersMoreSurvivors(t *testing.T) {
	sideA := []*simulator.CombatantAuto{{HP: 10, MaxHP: 100}}
	sideB := []*simulator.CombatantAuto{{HP: 50, MaxHP: 100}, {HP: 1, MaxHP: 100}}
	if got := tournamentWinningSide(sideA, sideB); got != "b" {
		t.Fatalf("winner side = %q, want b", got)
	}
}

func TestTournamentWinningSideBreaksTimeoutByRemainingHealthRatio(t *testing.T) {
	sideA := []*simulator.CombatantAuto{{HP: 50, MaxHP: 100}, {HP: 0, MaxHP: 100}}
	sideB := []*simulator.CombatantAuto{{HP: 40, MaxHP: 50}}
	if got := tournamentWinningSide(sideA, sideB); got != "b" {
		t.Fatalf("winner side = %q, want b (80%% remaining beats 50%%)", got)
	}
}

func TestTournamentWinningSideUsesStableSideAForExactTie(t *testing.T) {
	sideA := []*simulator.CombatantAuto{{HP: 50, MaxHP: 100}}
	sideB := []*simulator.CombatantAuto{{HP: 25, MaxHP: 50}}
	if got := tournamentWinningSide(sideA, sideB); got != "a" {
		t.Fatalf("winner side = %q, want stable tie-break a", got)
	}
}
