package handlers

import "testing"

func TestCombatHumanQiReward(t *testing.T) {
	cases := []struct {
		rank int
		want float64
	}{
		{-1, 3}, {0, 3}, {1, 3}, {2, 6}, {3, 15}, {4, 60}, {5, 300}, {6, 300}, {9, 300},
	}
	for _, c := range cases {
		if got := combatHumanQiReward(c.rank); got != c.want {
			t.Errorf("combatHumanQiReward(%d) = %v; want %v", c.rank, got, c.want)
		}
	}
}
