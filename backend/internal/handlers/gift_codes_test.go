package handlers

import (
	"fmt"
	"math/rand"
	"strings"
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

	result, err := applyGiftCodeRewardsTx(nil, player, rewards)
	if err != nil {
		t.Fatalf("applyGiftCodeRewardsTx failed: %v", err)
	}
	if len(result.NewGus) != 2 {
		t.Fatalf("expected 2 new Gus, got %d", len(result.NewGus))
	}
	if result.NewGus[0].GuID != "GU_BEAM_SWORD_4" || result.NewGus[0].Rank != 4 {
		t.Fatalf("unexpected first Gu: %+v", result.NewGus[0])
	}
	if result.NewGus[1].GuID != "GU_TREASURE_MOONLIGHT_KING" || result.NewGus[1].Rank != 5 {
		t.Fatalf("unexpected second Gu: %+v", result.NewGus[1])
	}
	if player.PrimevalStones != 50000 {
		t.Fatalf("unexpected stones: %v", player.PrimevalStones)
	}
}

func TestGiftCodeRewardLabelForResourcePoint(t *testing.T) {
	label := buildGiftCodeRewardLabel([]GiftCodeReward{
		{Type: "resource_point", ID: "HERB_VALLEY", Amount: 2, Label: "Thung Lũng Linh Thảo"},
	})
	if label == "" || !strings.Contains(label, "Thung Lũng Linh Thảo") || !strings.Contains(label, "2x") {
		t.Fatalf("unexpected resource_point label: %q", label)
	}
}

func TestPickFreeApertureCellAvoidsOccupiedCells(t *testing.T) {
	used := map[string]bool{}
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 399; i++ {
		if _, _, ok := pickFreeApertureCell(rng, used); !ok {
			t.Fatalf("unexpected failure while filling grid at %d", i)
		}
	}
	if len(used) != 399 {
		t.Fatalf("used cells = %d, want 399", len(used))
	}
	x, y, ok := pickFreeApertureCell(rng, used)
	if !ok {
		t.Fatal("expected to find the last free cell")
	}
	if used[fmt.Sprintf("%d,%d", x, y)] != true {
		t.Fatal("picked cell must be marked used")
	}
	if _, _, ok := pickFreeApertureCell(rng, used); ok {
		t.Fatal("a full grid must not hand out another cell")
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
