package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestApplyPathMasteryGain_MortalVsImmortal(t *testing.T) {
	player := &models.Player{}
	player.ID = uuid.New()
	player.Rank = 2 // Cap is 2 * 10 = 20 points per battle
	player.PathMasteries = make(models.PathMasteriesMap)
	player.ActiveCombat = &models.CombatState{
		PathMasteryGained: 0,
	}

	// 1. Mortal source (sourceRank = 2) gains 15 points
	gained := applyPathMasteryGain(player, "strength", 15, 2)
	if gained != 15 {
		t.Fatalf("expected gained 15, got %d", gained)
	}
	if player.ActiveCombat.PathMasteryGained != 15 {
		t.Fatalf("expected PathMasteryGained 15, got %d", player.ActiveCombat.PathMasteryGained)
	}
	if player.PathMasteries["strength"] != 15 {
		t.Fatalf("expected strength path mastery 15, got %d", player.PathMasteries["strength"])
	}

	// 2. Mortal source tries to gain 10 more points, but only 5 remaining under cap 20
	gained = applyPathMasteryGain(player, "strength", 10, 2)
	if gained != 5 {
		t.Fatalf("expected gained clamped to 5, got %d", gained)
	}
	if player.ActiveCombat.PathMasteryGained != 20 {
		t.Fatalf("expected PathMasteryGained 20, got %d", player.ActiveCombat.PathMasteryGained)
	}
	if player.PathMasteries["strength"] != 20 {
		t.Fatalf("expected strength path mastery 20, got %d", player.PathMasteries["strength"])
	}

	// 3. Further mortal gain is blocked (0 points)
	gained = applyPathMasteryGain(player, "strength", 10, 2)
	if gained != 0 {
		t.Fatalf("expected gained 0 when capped, got %d", gained)
	}

	// 4. Immortal source (sourceRank = 6, Tiên Cổ / Sát chiêu Tiên đạo) is UNLIMITED
	// It should succeed and gain full points even though PathMasteryGained >= capLimit (20)
	gained = applyPathMasteryGain(player, "strength", 10, 6)
	if gained != 10 {
		t.Fatalf("expected immortal source to bypass cap and gain 10, got %d", gained)
	}
	if player.PathMasteries["strength"] != 30 {
		t.Fatalf("expected strength path mastery 30, got %d", player.PathMasteries["strength"])
	}

	// 5. Another immortal gain (Rank 7)
	gained = applyPathMasteryGain(player, "sword", 25, 7)
	if gained != 25 {
		t.Fatalf("expected immortal source to gain 25, got %d", gained)
	}
	if player.PathMasteries["sword"] != 25 {
		t.Fatalf("expected sword path mastery 25, got %d", player.PathMasteries["sword"])
	}
}
