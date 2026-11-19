package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestLayerTransitionSameMapAllowsDirectly(t *testing.T) {
	player := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 3},
		BodyState:        models.BodyState{BaseHP: 1000, HP: 1000},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_nam_cuong"},
	}

	// Nam Cương to Trung Châu (both on world_map)
	err := checkLayerTransitionGuard(context.Background(), nil, player, "dai_vuc_nam_cuong", "dai_vuc_trung_chau")
	if err != nil {
		t.Fatalf("expected nil for same map, got %v", err)
	}
}

func TestMortalAscendingToWhiteHeavenDeniedAndDamaged(t *testing.T) {
	player := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 3},
		BodyState:        models.BodyState{BaseHP: 1000, HP: 1000},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_trung_chau"},
	}

	// Trung Châu (world_map) to Bạch Thiên (bai_thien)
	err := checkLayerTransitionGuard(context.Background(), nil, player, "dai_vuc_trung_chau", "dai_vuc_bach_thien")
	if err == nil {
		t.Fatalf("expected error for mortal ascending to White Heaven without Gu house, got nil")
	}

	me, ok := err.(*movementError)
	if !ok {
		t.Fatalf("expected *movementError, got %T", err)
	}
	if me.status != http.StatusForbidden {
		t.Errorf("status = %d, want %d", me.status, http.StatusForbidden)
	}
	// Player took 90% damage
	if player.HP > 100 {
		t.Errorf("player HP = %f, expected <= 100", player.HP)
	}
	// Status severely_wounded added
	if !player.Statuses.Has("severely_wounded") {
		t.Errorf("expected severely_wounded status on player")
	}
}

func TestGuImmortalAscendingToWhiteHeavenAllowed(t *testing.T) {
	player := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 6},
		BodyState:        models.BodyState{BaseHP: 10000, HP: 10000},
		ResourceState:    models.ResourceState{CurrentMana: 100, MaxMana: 100},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_trung_chau"},
	}

	err := checkLayerTransitionGuard(context.Background(), nil, player, "dai_vuc_trung_chau", "dai_vuc_bach_thien")
	if err != nil {
		t.Fatalf("expected nil for Rank 6 immortal, got %v", err)
	}

	msg, errEffects := applyLayerTransitionEffects(context.Background(), nil, player, "world_map", "bai_thien")
	if errEffects != nil {
		t.Fatalf("unexpected error applying effects: %v", errEffects)
	}
	if player.CurrentMana != 50 {
		t.Errorf("mana = %.0f, want 50 (50 consumed)", player.CurrentMana)
	}
	if msg == "" {
		t.Errorf("expected transition message, got empty string")
	}
}

func TestMortalDescendingToAbyssDenied(t *testing.T) {
	player := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 3},
		BodyState:        models.BodyState{BaseHP: 1000, HP: 1000},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_nam_cuong"},
	}

	// Nam Cương (world_map) to Địa Uyên (dia_uyen)
	err := checkLayerTransitionGuard(context.Background(), nil, player, "dai_vuc_nam_cuong", "dai_vuc_dia_uyen")
	if err == nil {
		t.Fatalf("expected error for mortal Rank 3 descending to Abyss, got nil")
	}

	me, ok := err.(*movementError)
	if !ok {
		t.Fatalf("expected *movementError, got %T", err)
	}
	if me.status != http.StatusForbidden {
		t.Errorf("status = %d, want %d", me.status, http.StatusForbidden)
	}
	if player.HP > 500 {
		t.Errorf("player HP = %f, expected <= 500", player.HP)
	}
	if !player.Statuses.Has("dia_uyen_miasma") {
		t.Errorf("expected dia_uyen_miasma status on player")
	}
}

func TestRank5CultivatorDescendingToAbyssAllowed(t *testing.T) {
	player := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 5},
		BodyState:        models.BodyState{BaseHP: 5000, HP: 5000},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_nam_cuong"},
	}

	err := checkLayerTransitionGuard(context.Background(), nil, player, "dai_vuc_nam_cuong", "dai_vuc_dia_uyen")
	if err != nil {
		t.Fatalf("expected nil for Rank 5 master entering Abyss, got %v", err)
	}
}

func TestSkyfallDescentEffects(t *testing.T) {
	player := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 6},
		BodyState:        models.BodyState{BaseHP: 10000, HP: 10000},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_bach_thien"},
	}

	msg, err := applyLayerTransitionEffects(context.Background(), nil, player, "bai_thien", "world_map")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == "" {
		t.Errorf("expected Skyfall message, got empty string")
	}
	if player.HP != 10000 {
		t.Errorf("rank 6 immortal should not take fall damage, got HP %f", player.HP)
	}
}

func TestBlackHeavenEntryGuards(t *testing.T) {
	// Rank 6 mortal/immortal trying to enter Black Heaven directly without rank 7/8
	p6 := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 6},
		BodyState:        models.BodyState{BaseHP: 10000, HP: 10000},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_bach_thien"},
	}
	err := checkLayerTransitionGuard(context.Background(), nil, p6, "dai_vuc_bach_thien", "dai_vuc_hac_thien")
	if err == nil {
		t.Fatalf("expected error for Rank 6 entering Black Heaven, got nil")
	}

	// Rank 7 immortal trying to enter Black Heaven directly without rank 8
	p7 := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 7},
		BodyState:        models.BodyState{BaseHP: 20000, HP: 20000},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_bach_thien"},
	}
	err7 := checkLayerTransitionGuard(context.Background(), nil, p7, "dai_vuc_bach_thien", "dai_vuc_hac_thien")
	if err7 == nil {
		t.Fatalf("expected error for Rank 7 entering Black Heaven, got nil")
	}

	// Rank 8 entering Black Heaven smoothly
	p8 := &models.Player{
		Account:          models.Account{ID: uuid.New()},
		CultivationState: models.CultivationState{Rank: 8},
		BodyState:        models.BodyState{BaseHP: 50000, HP: 50000},
		WorldState:       models.WorldState{CurrentRegion: "dai_vuc_bach_thien"},
	}
	err8 := checkLayerTransitionGuard(context.Background(), nil, p8, "dai_vuc_bach_thien", "dai_vuc_hac_thien")
	if err8 != nil {
		t.Fatalf("expected nil for Rank 8 entering Black Heaven, got %v", err8)
	}
	msg8, errEffects := applyLayerTransitionEffects(context.Background(), nil, p8, "bai_thien", "hac_thien")
	if errEffects != nil || msg8 == "" {
		t.Fatalf("expected smooth transition message for Rank 8, got err=%v, msg=%s", errEffects, msg8)
	}
}
