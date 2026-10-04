package handlers

import (
	"encoding/csv"
	"io"
	"os"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestFiveElementsDivineBarrierArray(t *testing.T) {
	// 1. Read FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY from data/Gu.csv
	f, err := os.Open("../../data/Gu.csv")
	if err != nil {
		t.Fatalf("Failed to open data/Gu.csv: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		t.Fatalf("Failed to read header: %v", err)
	}

	colIdx := func(name string) int {
		for i, h := range header {
			if h == name {
				return i
			}
		}
		return -1
	}

	idIdx := colIdx("id")
	nameIdx := colIdx("name")
	effectsIdx := colIdx("effects")

	if idIdx < 0 || nameIdx < 0 || effectsIdx < 0 {
		t.Fatalf("Missing required columns in Gu.csv header")
	}

	var foundRow []string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read row error: %v", err)
		}
		if row[idIdx] == "FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY" {
			foundRow = row
			break
		}
	}

	if foundRow == nil {
		t.Fatalf("FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY not found in data/Gu.csv")
	}

	var effects models.GuEffects
	if err := effects.Scan(foundRow[effectsIdx]); err != nil {
		t.Fatalf("Failed to scan effects JSON: %v", err)
	}

	spec := models.GuDictionaryItem{
		ID:           foundRow[idIdx],
		Name:         foundRow[nameIdx],
		Rank:         9,
		Effects:      effects,
		BaseManaCost: 5.0,
		Path:         []string{"Killing Path"},
	}

	// 2. Register into memory mock registry or verify NativeMechanics directly
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		t.Fatalf("CompiledGuEffects returned nil")
	}
	mechanics := compiled.NativeMechanics()
	hasCleanse := false
	hasFakeWall := false
	for _, m := range mechanics {
		if m.Code == "CLEANSE_REGIONAL_WALL" {
			hasCleanse = true
		}
		if m.Code == "FAKE_REGIONAL_WALL" {
			hasFakeWall = true
		}
	}
	if !hasCleanse {
		t.Fatalf("Expected CLEANSE_REGIONAL_WALL in NativeMechanics")
	}
	if !hasFakeWall {
		t.Fatalf("Expected FAKE_REGIONAL_WALL in NativeMechanics")
	}

	// 3. Test CLEANSE_REGIONAL_WALL behavior
	oldRegistry := models.GuRegistry
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY": {9: spec},
	}
	defer func() { models.GuRegistry = oldRegistry }()

	p := &models.Player{}
	p.GuInventory = []models.GuInstance{
		{
			InstanceID: "inst_1",
			GuID:       "FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY",
			Rank:       9,
			Equipped:   true,
			Status:     "ACTIVE",
			Hunger:     100,
		},
	}
	future := time.Now().Add(15 * time.Minute)
	p.Statuses = models.StatusList{
		{
			ID:        "regional_wall_suppression",
			Source:    "Giới Vực Bích",
			Value:     -0.20,
			ExpiresAt: &future,
		},
	}

	if !playerHasBattleStartMechanic(p, "CLEANSE_REGIONAL_WALL") {
		t.Fatalf("playerHasBattleStartMechanic(CLEANSE_REGIONAL_WALL) should be true")
	}
	if !playerHasBattleStartMechanic(p, "FAKE_REGIONAL_WALL") {
		t.Fatalf("playerHasBattleStartMechanic(FAKE_REGIONAL_WALL) should be true")
	}

	// Run maybeCleanseRegionalWall
	maybeCleanseRegionalWall(p)

	if p.Statuses.Has("regional_wall_suppression") {
		t.Fatalf("maybeCleanseRegionalWall should have removed regional_wall_suppression")
	}

	// 4. Test FAKE_REGIONAL_WALL in PVP combat setup
	p.ActiveCombat = &models.CombatState{
		EnemyType: "PVP",
		Enemies: []models.Combatant{
			{
				ID:      "enemy_1",
				Rank:    9,
				SubRank: 1,
			},
		},
	}
	simEnemy := &simulator.CombatantAuto{
		ID:      "enemy_1",
		Rank:    9,
		BaseDmg: 1000,
		BaseDef: 1000,
		Speed:   1000,
	}
	simState := &simulator.CombatState{
		Enemies: []*simulator.CombatantAuto{simEnemy},
	}

	// Mirror combat_sim_build.go logic
	if p.ActiveCombat != nil && p.ActiveCombat.EnemyType == "PVP" && playerHasBattleStartMechanic(p, "FAKE_REGIONAL_WALL") {
		for i := range p.ActiveCombat.Enemies {
			if i >= len(simState.Enemies) || simState.Enemies[i] == nil {
				continue
			}
			pct, _ := gamefx.RegionalWallDebuff(p.ActiveCombat.Enemies[i].Rank, p.ActiveCombat.Enemies[i].SubRank)
			factor := 1 + pct
			simState.Enemies[i].BaseDmg *= factor
			simState.Enemies[i].BaseDef *= factor
			simState.Enemies[i].Speed *= factor
			if simState.Enemies[i].CustomState == nil {
				simState.Enemies[i].CustomState = map[string]int{}
			}
			simState.Enemies[i].CustomState["fake_regional_wall"] = 1
		}
	}

	// For Rank 9 SubRank 1, RegionalWallDebuff is -0.75
	// Factor = 1 - 0.75 = 0.25 -> BaseDmg/Def/Speed should be 250
	if simEnemy.BaseDmg != 250 {
		t.Fatalf("simEnemy.BaseDmg = %f, want 250", simEnemy.BaseDmg)
	}
	if simEnemy.BaseDef != 250 {
		t.Fatalf("simEnemy.BaseDef = %f, want 250", simEnemy.BaseDef)
	}
	if simEnemy.Speed != 250 {
		t.Fatalf("simEnemy.Speed = %f, want 250", simEnemy.Speed)
	}
	if simEnemy.CustomState["fake_regional_wall"] != 1 {
		t.Fatalf("fake_regional_wall custom state flag not set")
	}
}
