package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestPlayerIgnoresMonstersByFlyingStatus(t *testing.T) {
	now := time.Now()
	future := now.Add(5 * time.Minute)
	past := now.Add(-time.Minute)

	player := func(statuses models.StatusList) *models.Player {
		return &models.Player{CharacterState: models.CharacterState{Statuses: statuses}}
	}

	if PlayerIgnoresMonsters(nil) {
		t.Fatal("nil player must not ignore monsters")
	}
	if PlayerIgnoresMonsters(player(nil)) {
		t.Fatal("player without status must not ignore monsters")
	}
	if !PlayerIgnoresMonsters(player(models.StatusList{{ID: StatusFlying, ExpiresAt: &future}})) {
		t.Fatal("active flying status must ignore monsters")
	}
	if PlayerIgnoresMonsters(player(models.StatusList{{ID: StatusFlying, ExpiresAt: &past}})) {
		t.Fatal("expired flying status must not ignore monsters")
	}
}

func TestToggleFlyingStatusToggles(t *testing.T) {
	p := &models.Player{}

	msg, on := toggleFlyingStatus(p, "Thiên Lý Độn Cổ", 4, 0)
	if !on {
		t.Fatalf("first use must take flight: %q", msg)
	}
	if !PlayerIgnoresMonsters(p) {
		t.Fatal("flying player must ignore monsters after taking flight")
	}
	// Rank 4 with no CSV turns => 4 minute duration.
	active := p.Statuses.Active()
	if len(active) != 1 || active[0].ExpiresAt == nil {
		t.Fatalf("expected one timed flying status, got %+v", active)
	}
	if d := time.Until(*active[0].ExpiresAt); d < 3*time.Minute || d > 5*time.Minute {
		t.Fatalf("expected ~4m flight duration, got %v", d)
	}

	_, on = toggleFlyingStatus(p, "Thiên Lý Độn Cổ", 4, 0)
	if on {
		t.Fatal("second use must land early")
	}
	if PlayerIgnoresMonsters(p) {
		t.Fatal("landed player must not ignore monsters")
	}
}

func TestSyncPlayerFlyingRegistry(t *testing.T) {
	id := uuid.New()
	until := time.Now().Add(3 * time.Minute)

	p := &models.Player{Account: models.Account{ID: id}}
	p.Statuses = models.StatusList{{ID: StatusFlying, ExpiresAt: &until}}

	syncPlayerFlying(p)
	if !flyingPlayers.isFlying(id.String(), time.Now()) {
		t.Fatal("flying player should be registered as a monster-ignored target")
	}
	if flyingPlayers.isFlying(id.String(), until.Add(time.Second)) {
		t.Fatal("flying registration must expire with the status")
	}

	p.Statuses = nil
	syncPlayerFlying(p)
	if flyingPlayers.isFlying(id.String(), time.Now()) {
		t.Fatal("landed player must be cleared from the registry")
	}
}
