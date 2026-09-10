package handlers

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func setupPvPTestRegistry(t *testing.T) {
	t.Helper()
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_SHIELD": {
			1: {
				ID:   "GU_SHIELD",
				Name: "Khiên Cổ",
				Rank: 1,
				Type: "Defense",
				Effects: models.GuEffects{
					Active: &models.ActiveEffect{ShieldFlat: 50.0},
				},
				BaseManaCost: 10,
			},
		},
		"GU_ATTACK": {
			1: {
				ID:   "GU_ATTACK",
				Name: "Công Cổ",
				Rank: 1,
				Type: "Attack",
				Effects: models.GuEffects{
					Active: &models.ActiveEffect{DamageFlat: 30.0, DamageMult: 1.0},
				},
				BaseManaCost: 15,
			},
		},
		"GU_HEAL": {
			1: {
				ID:   "GU_HEAL",
				Name: "Trị Liệu Cổ",
				Rank: 1,
				Type: "Defense",
				Effects: models.GuEffects{
					Active: &models.ActiveEffect{HealFlat: 40.0},
				},
				BaseManaCost: 12,
			},
		},
	}
	t.Cleanup(func() { models.GuRegistry = map[string]map[int]models.GuDictionaryItem{} })
}

func TestPvPDefenderGuOrderingMatchesChallenger(t *testing.T) {
	setupPvPTestRegistry(t)

	tactics := []models.GuTactic{
		{GuID: "GU_SHIELD", Priority: 1, UseCondition: "always", ConditionValue: 0},
		{GuID: "GU_ATTACK", Priority: 2, UseCondition: "always", ConditionValue: 0},
		{GuID: "GU_HEAL", Priority: 3, UseCondition: "hp_below", ConditionValue: 40},
	}
	categories := []string{"shield", "damage", "heal", "other"}

	combatSettings := models.AutoCombatConfig{
		GuPriorities:       tactics,
		CategoryPriorities: categories,
		AutoHealHP:         true,
		HealThreshold:      35,
		AutoRestoreMana:    true,
		ManaThreshold:      25,
	}

	pA_ID := uuid.New()
	pB_ID := uuid.New()

	invA := models.GuInventory{
		{InstanceID: "a_shield", GuID: "GU_SHIELD", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "a_attack", GuID: "GU_ATTACK", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "a_heal", GuID: "GU_HEAL", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
	}
	invB := models.GuInventory{
		{InstanceID: "b_shield", GuID: "GU_SHIELD", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "b_attack", GuID: "GU_ATTACK", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "b_heal", GuID: "GU_HEAL", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
	}

	playerA := &models.Player{}
	playerA.ID = pA_ID
	playerA.Username = "Challenger"
	playerA.Rank = 2
	playerA.SubRank = 1
	playerA.HP = 100
	playerA.BaseHP = 100
	playerA.MaxMana = 100
	playerA.CurrentMana = 100
	playerA.GuInventory = invA
	playerA.CombatSettings = combatSettings

	playerB := &models.Player{}
	playerB.ID = pB_ID
	playerB.Username = "Defender"
	playerB.Rank = 2
	playerB.SubRank = 1
	playerB.HP = 100
	playerB.BaseHP = 100
	playerB.MaxMana = 100
	playerB.CurrentMana = 100
	playerB.GuInventory = invB
	playerB.CombatSettings = combatSettings

	// 1. Build Challenger as simulator combatant
	simPlayer, err := buildSimCombatant(nil, playerA, false, false, nil)
	if err != nil {
		t.Fatalf("failed to build challenger combatant: %v", err)
	}

	// 2. Build Defender as enemy in CombatState and convert with buildSimEnemyCombatant
	defenderStats := CalculatePlayerStats(playerB)
	defenderCombatant := models.CombatantFromPlayer(playerB, defenderStats)

	pvpCombat := &models.CombatState{
		EnemyTemplateID:     playerB.ID.String(),
		EnemyName:           playerB.Username,
		EnemyType:           "PVP",
		CombatMode:          "virtual",
		EnemyHP:             100,
		EnemyMaxHP:          100,
		EnemyMana:           100,
		EnemyMaxMana:        100,
		EnemyDamage:         defenderStats["attack"],
		EnemyDefense:        defenderStats["defense"],
		EnemySpeed:          defenderStats["speed"],
		EnemyRank:           playerB.Rank,
		EnemySubRank:        playerB.SubRank,
		EnemyGuInventory:    filterGuLoadout(playerB.GuInventory, playerB.CombatSettings),
		EnemyCombatSettings: &combatSettings,
		TargetPlayerID:      playerB.ID.String(),
	}

	simEnemy := buildSimEnemyCombatant(nil, &defenderCombatant, pvpCombat)

	// Verify CategoryPriorities
	if len(simEnemy.CategoryPriorities) != len(simPlayer.CategoryPriorities) {
		t.Fatalf("CategoryPriorities length mismatch: player has %v, enemy has %v",
			simPlayer.CategoryPriorities, simEnemy.CategoryPriorities)
	}
	for i := range simPlayer.CategoryPriorities {
		if simPlayer.CategoryPriorities[i] != simEnemy.CategoryPriorities[i] {
			t.Errorf("CategoryPriority index %d mismatch: player has %s, enemy has %s",
				i, simPlayer.CategoryPriorities[i], simEnemy.CategoryPriorities[i])
		}
	}

	// Verify Action Priorities and Conditions match between Challenger and Defender
	findAction := func(actions []simulator.SimAction, sourceID string) *simulator.SimAction {
		for i := range actions {
			if actions[i].SourceID == sourceID {
				return &actions[i]
			}
		}
		return nil
	}

	for _, tCase := range tactics {
		actA := findAction(simPlayer.Actions, tCase.GuID)
		actB := findAction(simEnemy.Actions, tCase.GuID)

		if actA == nil {
			t.Fatalf("Challenger missing action for %s", tCase.GuID)
		}
		if actB == nil {
			t.Fatalf("Defender missing action for %s", tCase.GuID)
		}

		if actA.Priority != actB.Priority {
			t.Errorf("Priority mismatch for %s: Challenger=%d, Defender=%d",
				tCase.GuID, actA.Priority, actB.Priority)
		}
		if actA.UseCondition != actB.UseCondition {
			t.Errorf("UseCondition mismatch for %s: Challenger=%s, Defender=%s",
				tCase.GuID, actA.UseCondition, actB.UseCondition)
		}
		if actA.ConditionValue != actB.ConditionValue {
			t.Errorf("ConditionValue mismatch for %s: Challenger=%f, Defender=%f",
				tCase.GuID, actA.ConditionValue, actB.ConditionValue)
		}
	}

	// Verify fallback BASIC_ATTACK priority is 999 for both
	basicA := findAction(simPlayer.Actions, "")
	basicB := findAction(simEnemy.Actions, "")
	if basicA == nil || basicA.Priority != 999 {
		t.Errorf("Expected Challenger BASIC_ATTACK to have Priority 999, got %v", basicA)
	}
	if basicB == nil || basicB.Priority != 999 {
		t.Errorf("Expected Defender BASIC_ATTACK to have Priority 999, got %v", basicB)
	}

	// Verify AutoHealHP / AutoRestoreMana thresholds are preserved on Defender
	if simEnemy.AutoHealHP != combatSettings.AutoHealHP || simEnemy.HealThreshold != combatSettings.HealThreshold {
		t.Errorf("Defender AutoHealHP mismatch: got %v/%f, want %v/%f",
			simEnemy.AutoHealHP, simEnemy.HealThreshold, combatSettings.AutoHealHP, combatSettings.HealThreshold)
	}
	if simEnemy.AutoRestoreMana != combatSettings.AutoRestoreMana || simEnemy.ManaThreshold != combatSettings.ManaThreshold {
		t.Errorf("Defender AutoRestoreMana mismatch: got %v/%f, want %v/%f",
			simEnemy.AutoRestoreMana, simEnemy.ManaThreshold, combatSettings.AutoRestoreMana, combatSettings.ManaThreshold)
	}
}

func TestSelectActionsByPlayerTacticsConditions(t *testing.T) {
	combatSettings := &models.AutoCombatConfig{
		GuPriorities: []models.GuTactic{
			{GuID: "GU_HEAL", Priority: 1, UseCondition: "hp_below", ConditionValue: 30},
			{GuID: "GU_SHIELD", Priority: 2, UseCondition: "always", ConditionValue: 0},
			{GuID: "GU_ATTACK", Priority: 3, UseCondition: "always", ConditionValue: 0},
		},
		CategoryPriorities: []string{"shield", "heal", "damage", "other"},
	}

	potentialActions := []Action{
		{Name: "Trị Liệu Cổ", GuID: "GU_HEAL", Def: 40, IsUtility: true, ManaCost: 10, Priority: 1, UseCondition: "hp_below", ConditionValue: 30},
		{Name: "Khiên Cổ", GuID: "GU_SHIELD", Def: 50, ManaCost: 10, Priority: 2, UseCondition: "always"},
		{Name: "Công Cổ", GuID: "GU_ATTACK", Dmg: 30, ManaCost: 15, Priority: 3, UseCondition: "always"},
	}

	// Case 1: Enemy has 100% HP -> Heal (Priority 1) should be skipped because HP >= 30%.
	// Shield (Priority 2) and Attack (Priority 3) should be selected.
	enemyHighHP := &models.Combatant{
		Name:           "Defender",
		HP:             100,
		MaxHP:          100,
		CurrentMana:    100,
		MaxMana:        100,
		CombatSettings: combatSettings,
	}

	selected := SelectActionsByPlayerTactics(nil, enemyHighHP, potentialActions, 100)
	if len(selected) != 2 {
		t.Fatalf("expected 2 actions selected (Shield + Attack), got %d: %+v", len(selected), selected)
	}
	if selected[0].GuID != "GU_SHIELD" {
		t.Errorf("expected first action to be GU_SHIELD, got %s", selected[0].GuID)
	}
	if selected[1].GuID != "GU_ATTACK" {
		t.Errorf("expected second action to be GU_ATTACK, got %s", selected[1].GuID)
	}

	// Case 2: Enemy has 20% HP -> Heal (Priority 1) condition is met (HP < 30%).
	// Heal should be selected first, followed by Attack!
	enemyLowHP := &models.Combatant{
		Name:           "Defender",
		HP:             20,
		MaxHP:          100,
		CurrentMana:    100,
		MaxMana:        100,
		CombatSettings: combatSettings,
	}

	selectedLow := SelectActionsByPlayerTactics(nil, enemyLowHP, potentialActions, 100)
	if len(selectedLow) != 2 {
		t.Fatalf("expected 2 actions selected (Heal + Attack), got %d: %+v", len(selectedLow), selectedLow)
	}
	if selectedLow[0].GuID != "GU_HEAL" {
		t.Errorf("expected first action to be GU_HEAL when HP < 30%%, got %s", selectedLow[0].GuID)
	}
	if selectedLow[1].GuID != "GU_ATTACK" {
		t.Errorf("expected second action to be GU_ATTACK, got %s", selectedLow[1].GuID)
	}
}

func TestPvPCombatSimulationExecutionSymmetry(t *testing.T) {
	setupPvPTestRegistry(t)

	tactics := []models.GuTactic{
		{GuID: "GU_SHIELD", Priority: 1, UseCondition: "always", ConditionValue: 0},
		{GuID: "GU_ATTACK", Priority: 2, UseCondition: "always", ConditionValue: 0},
		{GuID: "GU_HEAL", Priority: 3, UseCondition: "hp_below", ConditionValue: 50},
	}

	combatSettings := models.AutoCombatConfig{
		GuPriorities:       tactics,
		CategoryPriorities: []string{"shield", "heal", "damage", "other"},
	}

	pA_ID := uuid.New()
	pB_ID := uuid.New()

	invA := models.GuInventory{
		{InstanceID: "a_shield", GuID: "GU_SHIELD", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "a_attack", GuID: "GU_ATTACK", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "a_heal", GuID: "GU_HEAL", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
	}
	invB := models.GuInventory{
		{InstanceID: "b_shield", GuID: "GU_SHIELD", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "b_attack", GuID: "GU_ATTACK", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "b_heal", GuID: "GU_HEAL", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
	}

	playerA := &models.Player{}
	playerA.ID = pA_ID
	playerA.Username = "Challenger"
	playerA.Rank = 2
	playerA.SubRank = 1
	playerA.HP = 200
	playerA.BaseHP = 200
	playerA.MaxMana = 200
	playerA.CurrentMana = 200
	playerA.GuInventory = invA
	playerA.CombatSettings = combatSettings

	playerB := &models.Player{}
	playerB.ID = pB_ID
	playerB.Username = "Defender"
	playerB.Rank = 2
	playerB.SubRank = 1
	playerB.HP = 200
	playerB.BaseHP = 200
	playerB.MaxMana = 200
	playerB.CurrentMana = 200
	playerB.GuInventory = invB
	playerB.CombatSettings = combatSettings

	simPlayer, err := buildSimCombatant(nil, playerA, false, false, nil)
	if err != nil {
		t.Fatalf("failed to build challenger: %v", err)
	}

	defenderStats := CalculatePlayerStats(playerB)
	defenderCombatant := models.CombatantFromPlayer(playerB, defenderStats)

	pvpCombat := &models.CombatState{
		EnemyTemplateID:     playerB.ID.String(),
		EnemyName:           playerB.Username,
		EnemyType:           "PVP",
		CombatMode:          "virtual",
		EnemyHP:             200,
		EnemyMaxHP:          200,
		EnemyMana:           200,
		EnemyMaxMana:        200,
		EnemyDamage:         defenderStats["attack"],
		EnemyDefense:        defenderStats["defense"],
		EnemySpeed:          defenderStats["speed"],
		EnemyRank:           playerB.Rank,
		EnemySubRank:        playerB.SubRank,
		EnemyGuInventory:    filterGuLoadout(playerB.GuInventory, playerB.CombatSettings),
		EnemyCombatSettings: &combatSettings,
		TargetPlayerID:      playerB.ID.String(),
	}

	simEnemy := buildSimEnemyCombatant(nil, &defenderCombatant, pvpCombat)

	simState := &simulator.CombatState{
		Allies:  []*simulator.CombatantAuto{simPlayer},
		Enemies: []*simulator.CombatantAuto{simEnemy},
	}

	logs := simulator.Simulate(simState)
	if len(logs) == 0 {
		t.Fatalf("expected combat logs, got empty")
	}

	// Verify both Challenger and Defender cast their Shield in the defense pass / turn 1
	var castShieldA, castShieldB bool
	var castAttackA, castAttackB bool
	for _, l := range logs {
		if l.ActorName == "Challenger" && strings.Contains(l.Detail, "Khiên Cổ") {
			castShieldA = true
		}
		if l.ActorName == "Defender" && strings.Contains(l.Detail, "Khiên Cổ") {
			castShieldB = true
		}
		if l.ActorName == "Challenger" && strings.Contains(l.Detail, "Công Cổ") {
			castAttackA = true
		}
		if l.ActorName == "Defender" && strings.Contains(l.Detail, "Công Cổ") {
			castAttackB = true
		}
	}

	if !castShieldA || !castShieldB {
		t.Errorf("expected both Challenger (%v) and Defender (%v) to cast Khiên Cổ (Priority 1)",
			castShieldA, castShieldB)
	}
	if !castAttackA || !castAttackB {
		t.Errorf("expected both Challenger (%v) and Defender (%v) to cast Công Cổ (Priority 2)",
			castAttackA, castAttackB)
	}
}
