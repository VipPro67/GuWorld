package handlers

import (
	"encoding/csv"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

// TestGuKillEnergyPersistentStatusOnHit tests the full lifecycle of GU_KILL_ENERGY:
// 1. Loading and parsing from backend/data/Gu.csv.
// 2. GuPersistentHitStatus detection.
// 3. buildSimActionFromGu construction.
// 4. Combat simulation generating HitLanded logs with SourceGuID="GU_KILL_ENERGY".
// 5. persistentHitStatusKeys mapping.
// 6. Impact on player stats (reduction of attack, defense, speed).
func TestGuKillEnergyPersistentStatusOnHit(t *testing.T) {
	// 1. Read GU_KILL_ENERGY from backend/data/Gu.csv
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
			if strings.TrimPrefix(strings.TrimSpace(h), "\uFEFF") == name {
				return i
			}
		}
		return -1
	}

	idIdx := colIdx("id")
	rankIdx := colIdx("rank")
	nameIdx := colIdx("name")
	effectsIdx := colIdx("effects")

	if idIdx < 0 || rankIdx < 0 || nameIdx < 0 || effectsIdx < 0 {
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
		if row[idIdx] == "GU_KILL_ENERGY" {
			foundRow = row
			break
		}
	}

	if foundRow == nil {
		t.Fatalf("GU_KILL_ENERGY not found in data/Gu.csv")
	}

	// 2. Parse effects
	var effects models.GuEffects
	if err := effects.Scan(foundRow[effectsIdx]); err != nil {
		t.Fatalf("Failed to scan effects JSON of GU_KILL_ENERGY: %v", err)
	}

	spec := models.GuDictionaryItem{
		ID:           foundRow[idIdx],
		Name:         foundRow[nameIdx],
		Rank:         9,
		Effects:      effects,
		BaseManaCost: 5.0,
		Path:         []string{"Killing Path"},
	}

	// 3. Test GuPersistentHitStatus hook detection
	statusKey, pvpOnly, ok := models.GuPersistentHitStatus(spec)
	if !ok {
		t.Fatalf("models.GuPersistentHitStatus(spec) returned ok=false for GU_KILL_ENERGY")
	}
	if statusKey != "sinh_khi_tan" {
		t.Fatalf("statusKey = %q, want %q", statusKey, "sinh_khi_tan")
	}
	if !pvpOnly {
		t.Fatalf("pvpOnly = false, want true")
	}

	// 4. Test buildSimActionFromGu
	actor := &models.Combatant{
		ID:          "initiator-actor",
		Rank:        9,
		BaseAttack:  1000,
		BaseDefense: 1000,
		CurrentMana: 100,
		MaxMana:     100,
	}
	act, err := buildSimActionFromGu(actor, &spec, "GU_SKILL")
	if err != nil {
		t.Fatalf("buildSimActionFromGu failed: %v", err)
	}
	if act.SourceID != "GU_KILL_ENERGY" {
		t.Fatalf("act.SourceID = %q, want GU_KILL_ENERGY", act.SourceID)
	}
	if act.PersistentStatusKey != "sinh_khi_tan" {
		t.Fatalf("act.PersistentStatusKey = %q, want sinh_khi_tan", act.PersistentStatusKey)
	}
	if !act.PersistentPvPOnly {
		t.Fatalf("act.PersistentPvPOnly = false, want true")
	}

	// 5. Test Combat Simulation
	ally := &simulator.CombatantAuto{
		ID:      actor.ID,
		Name:    "Attacker",
		Rank:    9,
		HP:      10000,
		MaxHP:   10000,
		Actions: []simulator.SimAction{act},
	}
	enemyID := "defender-actor"
	enemy := &simulator.CombatantAuto{
		ID:      enemyID,
		Name:    "Defender",
		Rank:    9,
		HP:      1000000000000,
		MaxHP:   1000000000000,
		Actions: []simulator.SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}},
	}
	simState := &simulator.CombatState{
		Seed:    42,
		Turn:    1,
		Allies:  []*simulator.CombatantAuto{ally},
		Enemies: []*simulator.CombatantAuto{enemy},
	}

	logs := simulator.Simulate(simState)

	// Check if log contains hit with SourceGuID == "GU_KILL_ENERGY" and HitLanded == true
	foundHit := false
	for _, l := range logs {
		if l.SourceGuID == "GU_KILL_ENERGY" && l.HitLanded {
			foundHit = true
			break
		}
	}
	if !foundHit {
		t.Fatalf("Expected at least one CombatLog with SourceGuID=GU_KILL_ENERGY and HitLanded=true")
	}

	// 6. Test persistentHitStatusKeys mapping
	keys := persistentHitStatusKeys(simState)
	if keys["GU_KILL_ENERGY"] != "sinh_khi_tan" {
		t.Fatalf("persistentHitStatusKeys did not map GU_KILL_ENERGY to sinh_khi_tan: got %v", keys)
	}

	// Verify pure memory list manipulation: ApplySinhKhiTanHitToList
	var defenderStatuses models.StatusList
	tier, val := ApplySinhKhiTanHitToList(&defenderStatuses)
	if tier != 1 || val != -0.10 {
		t.Fatalf("tier = %d, val = %f; want 1, -0.10", tier, val)
	}
	if !defenderStatuses.Has("sinh_khi_tan") {
		t.Fatalf("Status sinh_khi_tan was not added to defender status list")
	}

	// Stack to Tier 2
	tier2, val2 := ApplySinhKhiTanHitToList(&defenderStatuses)
	if tier2 != 2 || val2 != -0.25 {
		t.Fatalf("tier2 = %d, val2 = %f; want 2, -0.25", tier2, val2)
	}

	// Verify gamefx resolves this debuff on player stats
	pDefender := &models.Player{}
	pDefender.Race = "human"
	pDefender.Statuses = defenderStatuses

	fx := gamefx.Get(pDefender)
	if fx.StatMods["attack"] != -0.25 {
		t.Fatalf("fx.StatMods[attack] = %f, want -0.25", fx.StatMods["attack"])
	}
	if fx.StatMods["defense"] != -0.25 {
		t.Fatalf("fx.StatMods[defense] = %f, want -0.25", fx.StatMods["defense"])
	}
	if fx.StatMods["speed"] != -0.25 {
		t.Fatalf("fx.StatMods[speed] = %f, want -0.25", fx.StatMods["speed"])
	}
}
