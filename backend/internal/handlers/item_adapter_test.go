package handlers

import (
	"context"
	"encoding/json"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestExecuteItemConsumptionCompiledHP(t *testing.T) {
	raw := []byte(`[{"id":"HP_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}

	item := models.ItemSpec{
		ID:     "test_hp_potion",
		Name:   "Hồi Huyết Đan",
		Blocks: blocks,
	}

	var player models.Player
	player.Rank = 1
	player.HP = 20
	player.BaseHP = 100
	player.MaxMana = 100
	player.CurrentMana = 50

	msgs, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected messages from consumption")
	}
	if player.HP != 70 {
		t.Fatalf("expected HP to be 70, got %v", player.HP)
	}
}

func TestExecuteItemConsumptionCompiledMana(t *testing.T) {
	raw := []byte(`[{"id":"MANA_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_MANA","params":{"value":40}}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}

	item := models.ItemSpec{
		ID:     "test_mana_potion",
		Name:   "Hồi Nguyên Đan",
		Blocks: blocks,
	}

	var player models.Player
	player.Rank = 1
	player.HP = 100
	player.BaseHP = 100
	player.MaxMana = 100
	player.CurrentMana = 30

	msgs, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected messages from consumption")
	}
	if player.CurrentMana != 70 {
		t.Fatalf("expected CurrentMana to be 70, got %v", player.CurrentMana)
	}
}

func TestExecuteItemConsumptionMortalOnlyRejectsImmortal(t *testing.T) {
	raw := []byte(`[{"id":"MORTAL_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"MORTAL_ONLY","params":{"value":1}},{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}

	item := models.ItemSpec{
		ID:     "test_mortal_potion",
		Name:   "Phàm Dược",
		Blocks: blocks,
	}

	var player models.Player
	player.Rank = 6 // Immortal

	_, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err == nil {
		t.Fatal("expected mortal_only item to reject immortal player")
	}
}

func TestExecuteItemConsumptionMinRankRequirement(t *testing.T) {
	raw := []byte(`[{"id":"RANK3_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"MIN_RANK","params":{"value":3}},{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}

	item := models.ItemSpec{
		ID:     "test_rank3_potion",
		Name:   "Tam Chuyển Đan",
		Blocks: blocks,
	}

	var player models.Player
	player.Rank = 2

	_, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err == nil {
		t.Fatal("expected min_rank 3 item to reject rank 2 player")
	}
}

func TestExecuteItemConsumptionFallbackLegacyEffects(t *testing.T) {
	item := models.ItemSpec{
		ID:   "test_legacy_item",
		Name: "Cổ Dược Cũ",
		Effects: models.ItemEffects{
			"restore_hp": 25,
		},
	}

	var player models.Player
	player.Rank = 1
	player.HP = 50
	player.BaseHP = 100

	msgs, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected message from legacy consumption")
	}
	if player.HP != 75 {
		t.Fatalf("expected HP to be 75, got %v", player.HP)
	}
}
