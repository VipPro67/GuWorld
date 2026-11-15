package handlers

import "testing"

func TestWorldMonsterVisibleToPlayer(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		playerID string
		want     bool
	}{
		{name: "public actor", playerID: "player-a", want: true},
		{name: "private owner", owner: "player-a", playerID: "player-a", want: true},
		{name: "private actor hidden from others", owner: "player-a", playerID: "player-b", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monster := WorldMonster{OwnerPlayerID: tt.owner}
			if got := worldMonsterVisibleToPlayer(monster, tt.playerID); got != tt.want {
				t.Fatalf("worldMonsterVisibleToPlayer() = %v, want %v", got, tt.want)
			}
		})
	}
}
