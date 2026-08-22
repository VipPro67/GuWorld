package handlers

import (
	"strings"
	"testing"
)

func TestSimulateBeastFight(t *testing.T) {
	winner, logs := SimulateBeastFight()
	if winner != "team_a" && winner != "team_b" && winner != "draw" {
		t.Errorf("Unexpected winner: %s", winner)
	}
	if len(logs) == 0 {
		t.Error("Empty battle logs")
	}
	if !strings.Contains(logs, "Trận chiến bắt đầu") {
		t.Error("Battle logs should contain the starting message")
	}
	t.Logf("Winner: %s, Log size: %d characters", winner, len(logs))
}
