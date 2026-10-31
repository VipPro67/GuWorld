package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestApplyGiftCodeRewardsTxWithGu(t *testing.T) {
	player := &models.Player{
		GuInventory: []models.GuInstance{},
	}
	player.ID = uuid.New()
	rewards := []GiftCodeReward{
		{Type: "gu", ID: "GU_BEAM_SWORD_4", Rank: 4, Amount: 1, Label: "Tứ Chuyển Mang Kiếm Cổ"},
		{Type: "gu", ID: "GU_TREASURE_MOONLIGHT_KING", Rank: 5, Amount: 1, Label: "Bảo Nguyệt Quang Vương Cổ"},
		{Type: "primeval_stones", Amount: 50000},
	}

	newGus, err := applyGiftCodeRewardsTx(nil, player, rewards)
	if err != nil {
		t.Fatalf("applyGiftCodeRewardsTx failed: %v", err)
	}
	if len(newGus) != 2 {
		t.Fatalf("expected 2 new Gus, got %d", len(newGus))
	}
	if newGus[0].GuID != "GU_BEAM_SWORD_4" || newGus[0].Rank != 4 {
		t.Fatalf("unexpected first Gu: %+v", newGus[0])
	}
	if newGus[1].GuID != "GU_TREASURE_MOONLIGHT_KING" || newGus[1].Rank != 5 {
		t.Fatalf("unexpected second Gu: %+v", newGus[1])
	}
	if player.PrimevalStones != 50000 {
		t.Fatalf("unexpected stones: %v", player.PrimevalStones)
	}
}

func TestGiftCodeRewardLabelFormatting(t *testing.T) {
	rewards := []GiftCodeReward{
		{Type: "gu", ID: "GU_FIRE_SEA", Rank: 4, Amount: 1},
		{Type: "primeval_stones", Amount: 100000},
	}
	label := buildGiftCodeRewardLabel(rewards)
	if label == "" {
		t.Fatal("expected non-empty reward label")
	}
	t.Logf("Generated label: %s", label)
}
