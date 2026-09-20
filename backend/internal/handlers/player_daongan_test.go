package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestCalculateDaoNganDetailed(t *testing.T) {
	daoMarks := models.DaoMarksMap{
		"fire":  50000,
		"water": 20000,
	}
	pathMasteries := map[string]int{
		"fire":  10000, // Great Grandmaster (Level 3)
		"water": 100,   // Master (Level 1)
	}

	result := CalculateDaoNganDetailed(daoMarks, pathMasteries, false)

	if result.TotalMarks != 70000 {
		t.Errorf("expected TotalMarks 70000, got %d", result.TotalMarks)
	}
	if result.DominantPath != "fire" {
		t.Errorf("expected DominantPath fire, got %s", result.DominantPath)
	}
	if len(result.Paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(result.Paths))
	}

	// First path should be fire (sorted by power)
	fireDetail := result.Paths[0]
	if fireDetail.Path != "fire" {
		t.Errorf("expected first path fire, got %s", fireDetail.Path)
	}
	if fireDetail.AttainmentLevel != 3 {
		t.Errorf("expected fire attainment level 3, got %d", fireDetail.AttainmentLevel)
	}
	if fireDetail.RealmControlPct != 45.0 {
		t.Errorf("expected fire realm control 45.0%%, got %f", fireDetail.RealmControlPct)
	}
	if fireDetail.RealmEfficiency != 1.25 {
		t.Errorf("expected fire realm efficiency 1.25, got %f", fireDetail.RealmEfficiency)
	}
	if fireDetail.PurityPct < 71.0 || fireDetail.PurityPct > 72.0 {
		t.Errorf("expected fire purity around 71.4%%, got %f", fireDetail.PurityPct)
	}
}

func TestCalculateDaoNgan_BackwardCompatibility(t *testing.T) {
	daoMarks := models.DaoMarksMap{
		"fire": 1000,
	}
	result := CalculateDaoNgan(daoMarks)
	if result.TotalMarks != 1000 {
		t.Errorf("expected 1000 marks, got %d", result.TotalMarks)
	}
	if len(result.Paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(result.Paths))
	}
	if result.Paths[0].Multiplier != 11.0 {
		t.Errorf("expected multiplier 11.0, got %f", result.Paths[0].Multiplier)
	}
}
