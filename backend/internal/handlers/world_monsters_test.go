package handlers

import (
	"testing"
	"time"
)

func TestMonsterRespawnDelayDefaultsForUnsetSpawnValue(t *testing.T) {
	if got, want := monsterRespawnDelay(0), 300*time.Second; got != want {
		t.Fatalf("monsterRespawnDelay(0) = %s, want %s", got, want)
	}
	if got, want := monsterRespawnDelay(-1), 300*time.Second; got != want {
		t.Fatalf("monsterRespawnDelay(-1) = %s, want %s", got, want)
	}
	if got, want := monsterRespawnDelay(45), 45*time.Second; got != want {
		t.Fatalf("monsterRespawnDelay(45) = %s, want %s", got, want)
	}
}
