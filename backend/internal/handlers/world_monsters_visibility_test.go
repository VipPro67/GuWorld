package handlers

import "testing"

func TestWorldMonsterVisibleToPlayer(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		spawnID  string
		playerID string
		want     bool
	}{
		{name: "public actor", playerID: "player-a", want: true},
		{name: "private owner", owner: "player-a", playerID: "player-a", want: true},
		{name: "private actor hidden from others", owner: "player-a", playerID: "player-b", want: false},
		{name: "ascension actor visible to others", owner: "player-a", spawnID: "ascension:player-a", playerID: "player-b", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monster := WorldMonster{OwnerPlayerID: tt.owner, SpawnID: tt.spawnID}
			if got := worldMonsterVisibleToPlayer(monster, tt.playerID); got != tt.want {
				t.Fatalf("worldMonsterVisibleToPlayer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAscensionWorldMonsterVisibleAtPosition(t *testing.T) {
	entry := WorldPosEntry{X: 435, Y: 508}
	tests := []struct {
		name   string
		actor  WorldMonster
		radius int
		want   bool
	}{
		{
			name: "nearby calamity is visible",
			actor: WorldMonster{
				SpawnID: "ascension:player-a", OwnerPlayerID: "player-a",
				X: 436, Y: 508, State: MonsterStateIdle,
			},
			radius: 2,
			want:   true,
		},
		{
			name:   "non-ascension actor is hidden on region-local maps",
			actor:  WorldMonster{SpawnID: "spawn:region:monster", X: 436, Y: 508, State: MonsterStateIdle},
			radius: 2,
		},
		{
			name: "calamity outside requested area is hidden",
			actor: WorldMonster{
				SpawnID: "ascension:player-a", OwnerPlayerID: "player-a",
				X: 500, Y: 508, State: MonsterStateIdle,
			},
			radius: 2,
		},
		{
			name: "defeated calamity is hidden",
			actor: WorldMonster{
				SpawnID: "ascension:player-a", OwnerPlayerID: "player-a",
				X: 436, Y: 508, State: MonsterStateDefeated,
			},
			radius: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ascensionWorldMonsterVisibleAtPosition(tt.actor, entry, tt.radius); got != tt.want {
				t.Fatalf("ascensionWorldMonsterVisibleAtPosition() = %v, want %v", got, tt.want)
			}
		})
	}
}
