package handlers

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{16, 16},
	}

	for _, tt := range tests {
		got := nextPowerOf2(tt.input)
		if got != tt.expected {
			t.Errorf("nextPowerOf2(%d) = %d; want %d", tt.input, got, tt.expected)
		}
	}
}

func TestBracketRoundsCalculation(t *testing.T) {
	tests := []struct {
		teamCount   int
		totalRounds int
		slots       int
		byes        int
	}{
		{2, 1, 2, 0},
		{3, 2, 4, 1},
		{4, 2, 4, 0},
		{5, 3, 8, 3},
		{8, 3, 8, 0},
		{14, 4, 16, 2},
	}

	for _, tt := range tests {
		rounds := int(math.Ceil(math.Log2(float64(tt.teamCount))))
		slots := nextPowerOf2(tt.teamCount)
		byes := slots - tt.teamCount

		if rounds != tt.totalRounds {
			t.Errorf("teamCount %d: rounds = %d; want %d", tt.teamCount, rounds, tt.totalRounds)
		}
		if slots != tt.slots {
			t.Errorf("teamCount %d: slots = %d; want %d", tt.teamCount, slots, tt.slots)
		}
		if byes != tt.byes {
			t.Errorf("teamCount %d: byes = %d; want %d", tt.teamCount, byes, tt.byes)
		}
	}
}

func TestGetMatchCombatLogMissingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{}

	GetMatchCombatLog(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}
}

func TestDeathArenaLimitsConstants(t *testing.T) {
	if MaxTournamentTeams != 16 {
		t.Errorf("expected MaxTournamentTeams = 16, got %d", MaxTournamentTeams)
	}
	if MaxTournamentPlayers != 96 {
		t.Errorf("expected MaxTournamentPlayers = 96, got %d", MaxTournamentPlayers)
	}
	if RoundDuration <= 0 {
		t.Errorf("expected RoundDuration > 0, got %v", RoundDuration)
	}
}
