package handlers

import (
	"testing"

	"github.com/google/uuid"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func setupGuRegistry(t *testing.T) {
	t.Helper()
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_A": {1: {ID: "GU_A", Name: "A", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 10, DamageMult: 1.0}}},
			2: {ID: "GU_A", Name: "A", Rank: 2, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 12, DamageMult: 1.5}}}},
		"GU_B": {1: {ID: "GU_B", Name: "B", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 20, DamageMult: 1.2}}}},
		"GU_C": {1: {ID: "GU_C", Name: "C", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 30, DamageMult: 1.3}}}},
	}
	t.Cleanup(func() { models.GuRegistry = map[string]map[int]models.GuDictionaryItem{} })
}

func TestFilterGuLoadoutEmptyLoadout(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "b1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "c1", GuID: "GU_C", Rank: 1, Status: "SLEEPING", Equipped: true},
		{InstanceID: "a2", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: false},
	}
	got := filterGuLoadout(inv, models.AutoCombatConfig{})
	if len(got) != 2 {
		t.Fatalf("expected 2 Gus (equipped+ACTIVE combat), got %d: %+v", len(got), got)
	}
	if got[0].InstanceID != "a1" || got[1].InstanceID != "b1" {
		t.Fatalf("unexpected fallback set: %+v", got)
	}
}

func TestFilterGuLoadoutSubsetOrder(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "b1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "c1", GuID: "GU_C", Rank: 1, Status: "ACTIVE", Equipped: true},
	}
	cs := models.AutoCombatConfig{GuPriorities: []models.GuTactic{
		{GuID: "GU_C", Priority: 1},
		{GuID: "GU_A", Priority: 2},
	}}
	got := filterGuLoadout(inv, cs)
	if len(got) != 2 {
		t.Fatalf("expected 2 Gus, got %d: %+v", len(got), got)
	}
	if got[0].GuID != "GU_C" || got[1].GuID != "GU_A" {
		t.Fatalf("expected loadout order [GU_C, GU_A], got %+v", got)
	}
}

func TestFilterGuLoadoutUnknownFallback(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "b1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true},
	}
	cs := models.AutoCombatConfig{GuPriorities: []models.GuTactic{
		{GuID: "GU_ZZZ", Priority: 1},
	}}
	got := filterGuLoadout(inv, cs)
	if len(got) != 2 {
		t.Fatalf("expected fallback to all combat Gus (2), got %d: %+v", len(got), got)
	}
}

func TestFilterGuLoadoutPicksHighestRank(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a_low", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "a_high", GuID: "GU_A", Rank: 2, Status: "ACTIVE", Equipped: true},
	}
	cs := models.AutoCombatConfig{GuPriorities: []models.GuTactic{
		{GuID: "GU_A", Priority: 1},
	}}
	got := filterGuLoadout(inv, cs)
	if len(got) != 1 {
		t.Fatalf("expected 1 Gu, got %d: %+v", len(got), got)
	}
	if got[0].InstanceID != "a_high" {
		t.Fatalf("expected highest-rank instance a_high, got %+v", got)
	}
}

func TestStartCombat_AllyFiltering(t *testing.T) {
	playerID := "11111111-1111-1111-1111-111111111111"
	partyMemberID := "22222222-2222-2222-2222-222222222222"
	petID := "33333333-3333-3333-3333-333333333333"

	allies := []models.Combatant{
		{ID: playerID, Kind: "Player", Name: "Player1"},
		{ID: partyMemberID, Kind: "Ally", Name: "PartyMember"},
		{ID: petID, Kind: "Pet", Name: "Pet1"},
	}

	var notified []string
	for _, ally := range allies {
		if ally.Kind != "Ally" && ally.Kind != "Player" {
			continue
		}
		if ally.ID != playerID {
			notified = append(notified, ally.ID)
		}
	}

	if len(notified) != 1 || notified[0] != partyMemberID {
		t.Fatalf("expected only party member %s to be notified, got %+v", partyMemberID, notified)
	}
}

func TestFilterCombatGuInventory_WithExplicitSelectedGuIDs(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "b1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "c1", GuID: "GU_C", Rank: 1, Status: "ACTIVE", Equipped: true},
	}
	cs := models.AutoCombatConfig{GuPriorities: []models.GuTactic{
		{GuID: "GU_C", Priority: 1},
	}}

	// Explicit selectedGuIDs provided -> must return ONLY GU_B
	got := filterCombatGuInventory(inv, []string{"GU_B"}, cs)
	if len(got) != 1 || got[0].GuID != "GU_B" {
		t.Fatalf("expected only GU_B, got %+v", got)
	}
}

func TestFilterCombatGuInventory_FallbackToSettings(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "b1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "c1", GuID: "GU_C", Rank: 1, Status: "ACTIVE", Equipped: true},
	}
	cs := models.AutoCombatConfig{GuPriorities: []models.GuTactic{
		{GuID: "GU_C", Priority: 1},
		{GuID: "GU_A", Priority: 2},
	}}

	// No explicit selectedGuIDs provided -> fallback to settings (GU_C, GU_A)
	got := filterCombatGuInventory(inv, nil, cs)
	if len(got) != 2 || got[0].GuID != "GU_C" || got[1].GuID != "GU_A" {
		t.Fatalf("expected [GU_C, GU_A] from settings, got %+v", got)
	}
}

func TestBuildSimCombatantNoGuSkillsUsesBasicAttackOnly(t *testing.T) {
	setupGuRegistry(t)

	p := models.Player{}
	p.ID = uuid.New()
	p.Username = "Tester"
	p.Rank = 1
	p.BaseHP = 500
	p.HP = 500
	p.MaxMana = 100
	p.CurrentMana = 100
	p.GuInventory = models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 50},
	}
	// Saved settings would normally supply GU_A; the explicit empty loadout must win.
	p.CombatSettings = models.AutoCombatConfig{
		GuPriorities: []models.GuTactic{{GuID: "GU_A", Priority: 1}},
	}

	simPlayer, err := buildSimCombatant(nil, &p, false, true, &simulator.CombatLoadout{
		NoGuSkills:    true,
		AutoHealHP:    true,
		HealThreshold: 25,
	})
	if err != nil {
		t.Fatalf("buildSimCombatant failed: %v", err)
	}

	for _, act := range simPlayer.Actions {
		if act.ActionType == "GU_SKILL" {
			t.Fatalf("expected no GU_SKILL actions with NoGuSkills, got %+v", act)
		}
	}
	if len(simPlayer.Actions) != 1 || simPlayer.Actions[0].ActionType != "BASIC_ATTACK" {
		t.Fatalf("expected only BASIC_ATTACK, got %+v", simPlayer.Actions)
	}
	if !simPlayer.AutoHealHP || simPlayer.HealThreshold != 25 {
		t.Fatalf("expected loadout auto-heal settings to apply, got %+v", simPlayer)
	}
}

func TestFilterGuLoadoutRetainsPassiveMechanicGus(t *testing.T) {
	setupGuRegistry(t)
	models.GuRegistry["GU_PASSIVE_REBORN"] = map[int]models.GuDictionaryItem{
		1: {
			ID:   "GU_PASSIVE_REBORN",
			Name: "Thâm Trận Cổ",
			Rank: 1,
			Effects: models.GuEffects{
				Other: []models.SpecialMechanic{
					{EventTrigger: "ON_DEATH", ActionCode: "REBORN"},
				},
			},
		},
	}

	inv := models.GuInventory{
		{InstanceID: "pass1", GuID: "GU_PASSIVE_REBORN", Rank: 1, Status: "ACTIVE", Equipped: true},
	}
	got := filterGuLoadout(inv, models.AutoCombatConfig{})
	if len(got) != 1 || got[0].GuID != "GU_PASSIVE_REBORN" {
		t.Fatalf("expected passive mechanic Gu GU_PASSIVE_REBORN to be retained in filterGuLoadout, got %+v", got)
	}
}

func TestBuildSimEnemyCombatant_PopulatesPassiveMechanics(t *testing.T) {
	setupGuRegistry(t)
	models.GuRegistry["GU_PASSIVE_REBORN"] = map[int]models.GuDictionaryItem{
		1: {
			ID:   "GU_PASSIVE_REBORN",
			Name: "Thâm Trận Cổ",
			Rank: 1,
			Effects: models.GuEffects{
				Other: []models.SpecialMechanic{
					{EventTrigger: "ON_DEATH", ActionCode: "REBORN"},
				},
			},
		},
	}

	enemy := &models.Combatant{
		ID:         "enemy1",
		Name:       "Opponent",
		Rank:       3,
		BaseAttack: 100,
		GuInventory: models.GuInventory{
			{InstanceID: "pass1", GuID: "GU_PASSIVE_REBORN", Rank: 1, Status: "ACTIVE", Equipped: true},
		},
	}

	simEnemy := buildSimEnemyCombatant(nil, enemy, nil)
	if len(simEnemy.Mechanics) != 1 {
		t.Fatalf("expected 1 passive mechanic on simEnemy, got %d", len(simEnemy.Mechanics))
	}
	if simEnemy.Mechanics[0].Trigger != "ON_DEATH" || simEnemy.Mechanics[0].ActionCode != "REBORN" {
		t.Fatalf("unexpected mechanic on simEnemy: %+v", simEnemy.Mechanics[0])
	}
}

// setupSymmetryGuRegistry installs a Gu with every global combat modifier plus a
// passive mechanic, exercising the full buildSimCombatant stat pipeline.
func setupSymmetryGuRegistry(t *testing.T) {
	t.Helper()
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_OFFENSE": {1: {
			ID:   "GU_OFFENSE",
			Name: "Offense",
			Rank: 1,
			Effects: models.GuEffects{
				Active: &models.ActiveEffect{
					DamageFlat: 25, DamageMult: 1.4, SpeedBoostFlat: 7,
					CritRateBoost: 0.2, IgnoreDefense: 3, IgnoreDefensePct: 0.15, ShieldFlat: 40,
				},
				Other: []models.SpecialMechanic{{EventTrigger: "ON_DEATH", ActionCode: "REBORN"}},
			},
		}},
	}
	t.Cleanup(func() { models.GuRegistry = map[string]map[int]models.GuDictionaryItem{} })
}

func symmetryTestInventory() models.GuInventory {
	return models.GuInventory{
		{InstanceID: "off1", GuID: "GU_OFFENSE", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
	}
}

func symmetryTestSettings() models.AutoCombatConfig {
	return models.AutoCombatConfig{GuPriorities: []models.GuTactic{{GuID: "GU_OFFENSE", Priority: 2}}}
}

func newSymmetryPlayer(id, name string) models.Player {
	p := models.Player{}
	p.ID = uuid.MustParse(id)
	p.Username = name
	p.Rank = 3
	p.SubRank = 1
	p.BaseHP = 500
	p.HP = 500
	p.MaxMana = 200
	p.CurrentMana = 200
	p.PrimevalStones = 123
	p.ImmortalStones = 4
	p.GuInventory = symmetryTestInventory()
	p.DaoMarks = models.DaoMarksMap{"strength": 0}
	p.PathMasteries = models.PathMasteriesMap{}
	p.CombatSettings = symmetryTestSettings()
	return p
}

// TestBuildSimCombatant_AttackerDefenderSymmetry guards against attacker/defender
// drift: both sides must load the same active skills, passive mechanics and
// global buffs (speed, crit, ignore-defense, shield, mana tier) from the same
// equipped Gu loadout.
func TestBuildSimCombatant_AttackerDefenderSymmetry(t *testing.T) {
	setupSymmetryGuRegistry(t)

	attacker := newSymmetryPlayer("11111111-1111-1111-1111-111111111111", "Attacker")
	defender := newSymmetryPlayer("22222222-2222-2222-2222-222222222222", "Defender")

	simAtt, err := buildSimCombatant(nil, &attacker, true, true, nil)
	if err != nil {
		t.Fatalf("build attacker: %v", err)
	}
	simDef, err := buildSimCombatant(nil, &defender, true, false, nil)
	if err != nil {
		t.Fatalf("build defender: %v", err)
	}

	countGuSkills := func(s *simulator.CombatantAuto) int {
		n := 0
		for _, a := range s.Actions {
			if a.ActionType == "GU_SKILL" {
				n++
			}
		}
		return n
	}

	if simAtt.Speed != simDef.Speed {
		t.Fatalf("speed asymmetry: attacker %.2f vs defender %.2f", simAtt.Speed, simDef.Speed)
	}
	if simAtt.CritRateBonus != simDef.CritRateBonus || simAtt.CritRateBonus != 0.2 {
		t.Fatalf("crit asymmetry: attacker %.2f vs defender %.2f (want 0.2)", simAtt.CritRateBonus, simDef.CritRateBonus)
	}
	if simAtt.IgnoreDefenseFlat != simDef.IgnoreDefenseFlat || simAtt.IgnoreDefenseFlat != 3 {
		t.Fatalf("ignore-defense flat asymmetry: attacker %.2f vs defender %.2f", simAtt.IgnoreDefenseFlat, simDef.IgnoreDefenseFlat)
	}
	if simAtt.IgnoreDefensePct != simDef.IgnoreDefensePct || simAtt.IgnoreDefensePct != 0.15 {
		t.Fatalf("ignore-defense pct asymmetry: attacker %.2f vs defender %.2f", simAtt.IgnoreDefensePct, simDef.IgnoreDefensePct)
	}
	if simAtt.VirtualShield != simDef.VirtualShield || simAtt.VirtualShield != 40 {
		t.Fatalf("virtual shield asymmetry: attacker %.2f vs defender %.2f (want 40)", simAtt.VirtualShield, simDef.VirtualShield)
	}
	if simAtt.RefineManaTier != simDef.RefineManaTier {
		t.Fatalf("refine mana tier asymmetry: attacker %.2f vs defender %.2f", simAtt.RefineManaTier, simDef.RefineManaTier)
	}
	if len(simAtt.Mechanics) != len(simDef.Mechanics) || len(simAtt.Mechanics) != 1 {
		t.Fatalf("passive mechanics asymmetry: attacker %d vs defender %d (want 1)", len(simAtt.Mechanics), len(simDef.Mechanics))
	}
	if countGuSkills(simAtt) != 1 || countGuSkills(simDef) != 1 {
		t.Fatalf("active skill asymmetry: attacker %d vs defender %d (want 1)", countGuSkills(simAtt), countGuSkills(simDef))
	}
	if simAtt.PrimevalStones != 999999 || simDef.PrimevalStones != 999999 {
		t.Fatalf("virtual primeval stones asymmetry: attacker %.0f vs defender %.0f", simAtt.PrimevalStones, simDef.PrimevalStones)
	}
}

// TestBuildSimEnemyCombatant_PlayerFallbackGlobals verifies the monster/mock
// fallback path still grants players the same global buffs and virtual-mode
// recovery resources when no DB transaction is available.
func TestBuildSimEnemyCombatant_PlayerFallbackGlobals(t *testing.T) {
	setupSymmetryGuRegistry(t)

	enemy := &models.Combatant{
		ID:            "33333333-3333-3333-3333-333333333333",
		Name:          "Enemy",
		Kind:          "Player",
		Rank:          3,
		HP:            500,
		MaxHP:         500,
		CurrentMana:   200,
		MaxMana:       200,
		GuInventory:   symmetryTestInventory(),
		PathMasteries: models.PathMasteriesMap{},
	}
	settings := symmetryTestSettings()
	enemy.CombatSettings = &settings
	active := &models.CombatState{CombatMode: "virtual", EnemyType: "PVP"}

	simEnemy := buildSimEnemyCombatant(nil, enemy, active)

	if simEnemy.CritRateBonus != 0.2 {
		t.Fatalf("expected crit bonus 0.2, got %.2f", simEnemy.CritRateBonus)
	}
	if simEnemy.IgnoreDefenseFlat != 3 || simEnemy.IgnoreDefensePct != 0.15 {
		t.Fatalf("expected ignore defense 3/0.15, got %.2f/%.2f", simEnemy.IgnoreDefenseFlat, simEnemy.IgnoreDefensePct)
	}
	if len(simEnemy.Mechanics) != 1 {
		t.Fatalf("expected 1 passive mechanic, got %d", len(simEnemy.Mechanics))
	}
	if simEnemy.PrimevalStones != 999999 || simEnemy.ImmortalStones != 999999 || !simEnemy.AutoRestoreMana {
		t.Fatalf("expected virtual recovery resources, got stones=%0.f/%0.f autoRestore=%v", simEnemy.PrimevalStones, simEnemy.ImmortalStones, simEnemy.AutoRestoreMana)
	}
}

// TestBuildSimPlayerDefender_BailsOnNonPlayers ensures monsters and tx-less
// callers fall back to buildSimEnemyCombatant instead of hitting the DB.
func TestBuildSimPlayerDefender_BailsOnNonPlayers(t *testing.T) {
	active := &models.CombatState{EnemyType: "PVP", TargetPlayerID: "11111111-1111-1111-1111-111111111111"}

	if _, ok := buildSimPlayerDefender(nil, &models.Combatant{ID: active.TargetPlayerID, Kind: "Player"}, active, false); ok {
		t.Fatal("expected false for nil tx")
	}
	monster := &models.Combatant{ID: "monster_slime", Kind: "Normal"}
	if _, ok := buildSimPlayerDefender(nil, monster, active, false); ok {
		t.Fatal("expected false for monster combatant")
	}
}

func TestBuildSimActionFromGu_BlockV2(t *testing.T) {
	guSpec := models.GuDictionaryItem{
		ID:   "GU_BLOCK_V2_TEST",
		Name: "Mộng Yểm Cổ",
		Rank: 1,
		Path: []string{"Dream Path"},
		Effects: models.GuEffects{
			Blocks: []gameplayeffects.Block{
				{
					ID:      "ACTIVE_SKILL",
					Trigger: gameplayeffects.TriggerOnAction,
					Target:  gameplayeffects.Target{Selector: gameplayeffects.TargetAllEnemies, BypassFrontline: true},
					Operations: []gameplayeffects.Operation{
						{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 250, Path: "Dream Path"},
						{Op: gameplayeffects.OpStatus, StatusID: "SLEEP", Turns: 5, Chance: 1.0},
					},
				},
			},
		},
	}

	if !hasCombatActiveEffects(&guSpec) {
		t.Fatal("expected hasCombatActiveEffects == true for Block v2 active skill")
	}
	if !hasCombatEffects(&guSpec) {
		t.Fatal("expected hasCombatEffects == true for Block v2 active skill")
	}

	actor := &models.Combatant{
		ID:         "test_actor",
		Name:       "Test Actor",
		Rank:       1,
		BaseAttack: 50,
	}

	act, err := buildSimActionFromGu(actor, &guSpec, "GU_SKILL")
	if err != nil {
		t.Fatalf("unexpected error building sim action: %v", err)
	}

	if act.TargetType != "ALL_ENEMIES" {
		t.Errorf("expected TargetType ALL_ENEMIES, got %s", act.TargetType)
	}
	if !act.Bypass {
		t.Errorf("expected Bypass == true")
	}
	if act.Damage <= 0 {
		t.Errorf("expected Damage > 0, got %f", act.Damage)
	}
	if len(act.StatusOps) == 0 {
		t.Fatal("expected StatusOps to be populated from Block v2")
	}
	if act.StatusOps[0].ID != "SLEEP" || act.StatusOps[0].Turns != 5 {
		t.Errorf("expected SLEEP status for 5 turns, got %+v", act.StatusOps[0])
	}
}

func TestResolveTargetsAllEnemies(t *testing.T) {
	team := []models.Combatant{
		{ID: "c1", GridX: 1, GridY: 1, HP: 100},
		{ID: "c2", GridX: 2, GridY: 2, HP: 200},
		{ID: "c3", GridX: 3, GridY: 3, HP: 0}, // dead
	}

	targets := ResolveTargets(team, "ALL_ENEMIES", 0, 0, false)
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets for ALL_ENEMIES, got %d", len(targets))
	}
	if targets[0] != 0 || targets[1] != 1 {
		t.Errorf("expected targets [0, 1], got %v", targets)
	}

	// Also test op.Target overrides in buildSimActionFromGu
	guSpecOpTarget := models.GuDictionaryItem{
		ID:   "AOE_OP_GU",
		Name: "Quần Công Cổ",
		Rank: 1,
		Effects: models.GuEffects{
			Blocks: []gameplayeffects.Block{
				{
					ID:      "ACTIVE_SKILL",
					Trigger: gameplayeffects.TriggerOnAction,
					Target:  gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
					Operations: []gameplayeffects.Operation{
						{
							Op:      gameplayeffects.OpDamage,
							SubType: "FLAT",
							Value:   100,
							Target:  &gameplayeffects.Target{Selector: gameplayeffects.TargetAllEnemies, BypassFrontline: true},
						},
						{
							Op:      gameplayeffects.OpDamage,
							SubType: "MULTIPLIER",
							Value:   2.0,
						},
					},
				},
			},
		},
	}
	actor := &models.Combatant{
		ID:         "actor_test",
		Name:       "Actor",
		Rank:       1,
		BaseAttack: 50,
	}
	act, err := buildSimActionFromGu(actor, &guSpecOpTarget, "GU_SKILL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if act.TargetType != "ALL_ENEMIES" {
		t.Errorf("expected TargetType ALL_ENEMIES from op.Target, got %s", act.TargetType)
	}
	if !act.Bypass {
		t.Errorf("expected Bypass == true from op.Target")
	}
	if act.DamageMult != 2.0 {
		t.Errorf("expected DamageMult == 2.0, got %f", act.DamageMult)
	}
}

func TestBuildSimActionFromGu_ExtendedOperations(t *testing.T) {
	guSpec := models.GuDictionaryItem{
		ID:   "EXT_OPS_GU",
		Name: "Hỗn Hợp Cổ",
		Rank: 2,
		Effects: models.GuEffects{
			Blocks: []gameplayeffects.Block{
				{
					ID:      "ACTIVE_SKILL",
					Trigger: gameplayeffects.TriggerOnAction,
					Target:  gameplayeffects.Target{Selector: gameplayeffects.TargetAllEnemies},
					Operations: []gameplayeffects.Operation{
						{
							Op:      gameplayeffects.OpDamage,
							SubType: "MAX_HP_PERCENT",
							Value:   0.25,
						},
						{
							Op:       gameplayeffects.OpHeal,
							SubType:  "RESOURCE",
							Resource: "MANA",
							Value:    50,
							Target:   &gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
						},
						{
							Op:      gameplayeffects.OpBarrier,
							SubType: "PERCENT",
							Value:   0.15,
						},
						{
							Op:      gameplayeffects.OpCleanse,
							SubType: "ALL",
						},
					},
				},
			},
		},
	}

	actor := &models.Combatant{
		ID:         "actor_test",
		Name:       "Actor",
		Rank:       2,
		MaxHP:      1000,
		BaseAttack: 50,
	}

	act, err := buildSimActionFromGu(actor, &guSpec, "GU_SKILL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1. TargetType should remain ALL_ENEMIES even though one sub-op had TargetSelf
	if act.TargetType != "ALL_ENEMIES" {
		t.Errorf("expected TargetType ALL_ENEMIES, got %s", act.TargetType)
	}

	// 2. DamageHPPercent
	if act.DamageHPPercent != 0.25 {
		t.Errorf("expected DamageHPPercent == 0.25, got %f", act.DamageHPPercent)
	}

	// 3. ManaRestore
	if act.ManaRestore != 50 {
		t.Errorf("expected ManaRestore == 50, got %f", act.ManaRestore)
	}

	// 4. Barrier added to Defense (1000 * 0.15 = 150)
	if act.Defense < 150 {
		t.Errorf("expected Defense >= 150 from barrier, got %f", act.Defense)
	}

	// 5. CleanseDebuffs
	if !act.CleanseDebuffs {
		t.Errorf("expected CleanseDebuffs == true")
	}
}
