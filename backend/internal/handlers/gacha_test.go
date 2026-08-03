package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGachaThoNguyenOnlyStones(t *testing.T) {
	// Find tho_nguyen pool
	var thoNguyenPool *GachaPool
	for i := range gachaPools {
		if gachaPools[i].ID == "tho_nguyen" {
			thoNguyenPool = &gachaPools[i]
		}
	}
	if thoNguyenPool == nil {
		t.Fatal("tho_nguyen pool not found")
	}

	// Verify all rewards in tho_nguyen are primeval_stones
	for _, r := range thoNguyenPool.Rewards {
		if r.Type != "primeval_stones" {
			t.Errorf("expected tho_nguyen reward type to be primeval_stones, got: %s", r.Type)
		}
	}
}

func TestGachaRatesSeparatePerRank(t *testing.T) {
	// Find da_thuong pool
	var daThuongPool *GachaPool
	for i := range gachaPools {
		if gachaPools[i].ID == "da_thuong" {
			daThuongPool = &gachaPools[i]
		}
	}
	if daThuongPool == nil {
		t.Fatal("da_thuong pool not found")
	}

	// Verify da_thuong rewards have separate gu_rank for rank 1..9
	ranksFound := make(map[string]bool)
	for _, r := range daThuongPool.Rewards {
		if r.Type == "gu_rank" {
			ranksFound[r.ID] = true
		}
	}
	for _, rankStr := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		if !ranksFound[rankStr] {
			t.Errorf("expected gu_rank %s to be present in da_thuong rewards", rankStr)
		}
	}
}

func TestGachaImmortalGuExclusionInLoop(t *testing.T) {
	// Mock GuRegistry with a couple of rank 6 Gu
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_IMMORTAL_A": {
			6: {ID: "GU_IMMORTAL_A", Name: "Immortal A", Rank: 6},
		},
		"GU_IMMORTAL_B": {
			6: {ID: "GU_IMMORTAL_B", Name: "Immortal B", Rank: 6},
		},
	}

	// Create excludedImmortal map with GU_IMMORTAL_A
	excluded := map[string]bool{
		"GU_IMMORTAL_A": true,
	}

	pc := precomputeGachaPools(excluded)
	
	// Since GU_IMMORTAL_A is excluded, pc.rollGu("6") should only ever roll GU_IMMORTAL_B
	for i := 0; i < 10; i++ {
		rolled := pc.rollGu("6")
		if rolled != "GU_IMMORTAL_B" {
			t.Errorf("expected GU_IMMORTAL_B, got: %s", rolled)
		}
	}

	// Now if GU_IMMORTAL_B is also rolled and added to excluded:
	excluded["GU_IMMORTAL_B"] = true
	pc = precomputeGachaPools(excluded)
	
	// Now both are excluded, rolling rank 6 should return empty string
	rolled := pc.rollGu("6")
	if rolled != "" {
		t.Errorf("expected empty string, got: %s", rolled)
	}
}
