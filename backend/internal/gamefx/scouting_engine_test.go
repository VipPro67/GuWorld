package gamefx

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestBaseRankPower(t *testing.T) {
	if BaseRankPower(1) != 10.0 {
		t.Errorf("expected 10.0 for rank 1, got %f", BaseRankPower(1))
	}
	if BaseRankPower(6) != 100.0 {
		t.Errorf("expected 100.0 for rank 6, got %f", BaseRankPower(6))
	}
	if BaseRankPower(9) != 500.0 {
		t.Errorf("expected 500.0 for rank 9, got %f", BaseRankPower(9))
	}
}

func TestGetPathAttainmentMultiplier(t *testing.T) {
	masteries := models.PathMasteriesMap{
		"wisdom":      150,  // Level 1: Master (1.2)
		"information": 1200, // Level 2: Grandmaster (1.5)
	}

	mult := GetPathAttainmentMultiplier(masteries, "wisdom")
	if mult != 1.2 {
		t.Errorf("expected 1.2 for wisdom master, got %f", mult)
	}

	multMax := GetPathAttainmentMultiplier(masteries, "wisdom", "information")
	if multMax != 1.5 {
		t.Errorf("expected 1.5 for highest attainment (information grandmaster), got %f", multMax)
	}
}

func TestEvaluateDivinationOutcomes(t *testing.T) {
	// 1. Full Success: High Rank + Wisdom Tông Sư + Correct Clues vs Weak Target
	viewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 1500}},
	}
	target := &models.Player{
		Account:          models.Account{Username: "TargetMortal"},
		CultivationState: models.CultivationState{Rank: 2},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	viewerDao := models.DaoMarksMap{"wisdom": 1000} // +100%
	clues := models.DivinationRequest{
		TargetUsername:  "TargetMortal",
		SuspectedRegion: "hub_nam_cuong",
	}

	status, delta, _ := EvaluateDivination(viewer, target, nil, nil, viewerDao, nil, clues, 20.0)
	if status != "full" || delta < 30.0 {
		t.Errorf("expected full success with delta >= 30, got status=%s, delta=%f", status, delta)
	}

	// 2. Backfire: Low Rank Mortal divining Immortal with Perfect Disguise / High Dark Dao
	weakViewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 1},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	supremeTarget := &models.Player{
		Account:          models.Account{Username: "ImmortalVenerable"},
		CultivationState: models.CultivationState{Rank: 8},
		WorldState:       models.WorldState{CurrentRegion: "hub_trung_chau"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"dark": 15000}}, // Great Grandmaster (2.0)
	}
	targetDao := models.DaoMarksMap{"dark": 2000}
	blindClues := models.DivinationRequest{
		TargetUsername: "ImmortalVenerable",
	}

	weakStatus, weakDelta, backfireDamage := EvaluateDivination(weakViewer, supremeTarget, nil, nil, nil, targetDao, blindClues, 0.0)
	if weakStatus != "backfire" || weakDelta >= -30.0 {
		t.Errorf("expected backfire with delta < -30, got status=%s, delta=%f", weakStatus, weakDelta)
	}
	if backfireDamage <= 0 {
		t.Errorf("expected positive backfire damage, got %f", backfireDamage)
	}
}

func TestCalculateSeverCluesPower(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"sword": 1200}}, // Grandmaster 1.5
	}
	power := CalculateSeverCluesPower(player, nil)
	if power != 50.0*1.5 {
		t.Errorf("expected 75.0, got %f", power)
	}
}

func TestEvaluateGuDivination(t *testing.T) {
	// 1. High rank player divining mortal Gu -> Full success
	viewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 1500}},
	}
	mortalGu := models.GuDictionaryItem{
		ID:   "moonlight_gu",
		Name: "Nguyệt Quang Cổ",
		Rank: 1,
		Path: []string{"moon", "light"},
	}
	status, delta, _, _ := EvaluateGuDivination(viewer, nil, nil, mortalGu, 1, 10.0, 10.0)
	if status != "full" || delta < 20.0 {
		t.Errorf("expected full success for rank 5 divining rank 1 Gu, got status=%s, delta=%f", status, delta)
	}

	// 2. Rank 1 mortal trying to divine Rank 9 Immortal Gu -> Backfire
	weakViewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 1},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	immortalGu := models.GuDictionaryItem{
		ID:   "chun_qiu_chan",
		Name: "Xuân Thu Thiền",
		Rank: 6,
		Path: []string{"time"},
	}
	weakStatus, weakDelta, dmg, soulDmg := EvaluateGuDivination(weakViewer, nil, nil, immortalGu, 1, 150.0, 0.0)
	if weakStatus != "backfire" || weakDelta >= -40.0 {
		t.Errorf("expected backfire for rank 1 divining immortal Gu, got status=%s, delta=%f", weakStatus, weakDelta)
	}
	if dmg <= 0 || soulDmg <= 0 {
		t.Errorf("expected positive backfire damages, got dmg=%f, soulDmg=%f", dmg, soulDmg)
	}
}

