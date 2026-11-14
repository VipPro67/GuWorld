package simulator

import "testing"

func TestGuHouseActionsRequireAnActiveUndestroyedHouse(t *testing.T) {
	action := &SimAction{IsFromGuHouse: true}
	actor := &CombatantAuto{GuHouse: &GuHouseState{ID: "house", HP: 100, Active: true}}
	if !checkActionCondition(action, actor, nil) {
		t.Fatal("valid Gu House action should be available")
	}
	actor.GuHouse.Active = false
	if checkActionCondition(action, actor, nil) {
		t.Fatal("inactive Gu House action should be unavailable")
	}
	actor.GuHouse.Active = true
	actor.GuHouse.IsDestroyed = true
	if checkActionCondition(action, actor, nil) {
		t.Fatal("destroyed Gu House action should be unavailable")
	}
	actor.GuHouse = nil
	if checkActionCondition(action, actor, nil) {
		t.Fatal("Gu House action without a house should be unavailable")
	}
}

func TestGuHouseAction_HungerAndManaDeduction(t *testing.T) {
	action := &SimAction{
		ActionType:     "KILLER_MOVE",
		Name:           "Trận Kích Pháo Lôi",
		IsFromGuHouse:  true,
		ComponentGuIDs: []string{"core_gu_1"},
		Damage:         200,
		ManaCost:       50,
		HungerCost:     2,
		TargetType:     "SINGLE",
	}

	actor := newTestCombatant("p1", "Cổ Tiên", 1000, 50)
	actor.Kind = "Player"
	actor.Mana = 100
	actor.MaxMana = 200
	actor.GuHouse = &GuHouseState{ID: "house_1", HP: 1000, Active: true}
	actor.GuHunger = map[string]int{"core_gu_1": 3}
	actor.Actions = []SimAction{*action}
	target := &CombatantAuto{ID: "e1", Name: "Địch", HP: 500, MaxHP: 500}

	// 1. Available initially (mana 100 >= 50, hunger 3 >= 2)
	if !combatActionAvailable(actor, action, target) {
		t.Fatal("action should be available initially")
	}
	if !checkActionCondition(action, actor, target) {
		t.Fatal("checkActionCondition should pass initially")
	}

	// 2. Simulate 1 turn of action execution
	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{actor},
		Enemies: []*CombatantAuto{target},
	}
	logs := Simulate(state)
	for _, l := range logs {
		t.Logf("Log: T%d | %s | %s", l.Turn, l.Action, l.Detail)
	}

	// Check Mana deducted: 100 - 50 = 50
	if actor.Mana != 50 {
		t.Fatalf("expected actor mana to be 50, got %.0f", actor.Mana)
	}

	// Check Hunger deducted: 3 - 2 = 1
	if actor.GuHunger["core_gu_1"] != 1 {
		t.Fatalf("expected core_gu_1 hunger to be 1, got %d", actor.GuHunger["core_gu_1"])
	}

	// Check GuUsedCounts recorded 2 casts worth of hunger
	if actor.GuUsedCounts["core_gu_1"] != 2 {
		t.Fatalf("expected GuUsedCounts to record 2, got %d", actor.GuUsedCounts["core_gu_1"])
	}

	// 3. Now hunger is 1, but HungerCost is 2 -> cannot cast anymore!
	if combatActionAvailable(actor, action, target) {
		t.Fatal("action should be unavailable when hunger (1) < cost (2)")
	}
	if checkActionCondition(action, actor, target) {
		t.Fatal("checkActionCondition should fail when hunger (1) < cost (2)")
	}
}

func TestGuHouse_PersonalSkillsBlockedWhileInside(t *testing.T) {
	houseAction := &SimAction{
		ActionType:    "KILLER_MOVE",
		Name:          "Trận Kích Pháo Lôi",
		IsFromGuHouse: true,
		Damage:        300,
	}
	exitAction := &SimAction{
		ActionType:    "EXIT_GU_HOUSE",
		Name:          "Rời Cổ Phòng",
		IsFromGuHouse: true,
	}
	personalGuSkill := &SimAction{
		ActionType: "GU_SKILL",
		Name:       "Nguyệt Quang Cổ",
		Damage:     80,
	}
	personalKillerMove := &SimAction{
		ActionType: "KILLER_MOVE",
		Name:       "Nguyệt Nhận Trảm",
		Damage:     150,
	}
	basicAttack := &SimAction{
		ActionType: "BASIC_ATTACK",
		Name:       "Đánh thường",
		Damage:     50,
	}

	actor := newTestCombatant("p1", "Cổ Tiên", 1000, 50)
	actor.Kind = "Player"
	actor.Mana = 500
	actor.GuHouse = &GuHouseState{ID: "house_1", HP: 1000, Active: true}
	target := &CombatantAuto{ID: "e1", Name: "Địch", HP: 500, MaxHP: 500}

	// 1. When inside Gu House (Active == true):
	// Trận Kích and Exit must be ALLOWED:
	if !checkActionCondition(houseAction, actor, target) {
		t.Fatal("houseAction should be allowed while inside active Gu House")
	}
	if !combatActionAvailable(actor, houseAction, target) {
		t.Fatal("combatActionAvailable should allow houseAction while inside")
	}
	if !checkActionCondition(exitAction, actor, target) {
		t.Fatal("exitAction should be allowed while inside active Gu House")
	}

	// Personal actions must be BLOCKED:
	if checkActionCondition(personalGuSkill, actor, target) {
		t.Fatal("personalGuSkill must be blocked while inside active Gu House")
	}
	if combatActionAvailable(actor, personalGuSkill, target) {
		t.Fatal("combatActionAvailable must block personalGuSkill while inside")
	}
	if checkActionCondition(personalKillerMove, actor, target) {
		t.Fatal("personalKillerMove must be blocked while inside active Gu House")
	}
	if combatActionAvailable(actor, personalKillerMove, target) {
		t.Fatal("combatActionAvailable must block personalKillerMove while inside")
	}
	if checkActionCondition(basicAttack, actor, target) {
		t.Fatal("physical basicAttack must be blocked while inside active Gu House")
	}
	if combatActionAvailable(actor, basicAttack, target) {
		t.Fatal("combatActionAvailable must block basicAttack while inside")
	}

	// 2. When outside Gu House (Active == false):
	actor.GuHouse.Active = false

	// Trận Kích must be BLOCKED:
	if checkActionCondition(houseAction, actor, target) {
		t.Fatal("houseAction must be blocked when outside Gu House")
	}
	if combatActionAvailable(actor, houseAction, target) {
		t.Fatal("combatActionAvailable must block houseAction when outside")
	}

	// Personal actions must now be ALLOWED:
	if !checkActionCondition(personalGuSkill, actor, target) {
		t.Fatal("personalGuSkill should be allowed when outside Gu House")
	}
	if !combatActionAvailable(actor, personalGuSkill, target) {
		t.Fatal("combatActionAvailable should allow personalGuSkill when outside")
	}
	if !checkActionCondition(personalKillerMove, actor, target) {
		t.Fatal("personalKillerMove should be allowed when outside Gu House")
	}
	if !combatActionAvailable(actor, personalKillerMove, target) {
		t.Fatal("combatActionAvailable should allow personalKillerMove when outside")
	}
	if !checkActionCondition(basicAttack, actor, target) {
		t.Fatal("basicAttack should be allowed when outside Gu House")
	}
	if !combatActionAvailable(actor, basicAttack, target) {
		t.Fatal("combatActionAvailable should allow basicAttack when outside")
	}
}

func TestSimulator_ExitGuHouseAction(t *testing.T) {
	actor := newTestCombatant("p1", "Cổ Tiên", 1000, 50)
	actor.GuHouse = &GuHouseState{ID: "house_1", HP: 1000, MaxHP: 1000, Active: true}
	actor.Actions = []SimAction{
		{
			ActionType:    "EXIT_GU_HOUSE",
			Name:          "Rời Cổ Phòng",
			TargetType:    "SELF",
			IsFromGuHouse: true,
		},
	}

	enemy := newTestCombatant("e1", "Kẻ Địch", 200, 10)
	enemy.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Name:       "Đánh thường",
			Damage:     50,
			TargetType: "SINGLE",
		},
	}

	state := &CombatState{
		Seed:    123,
		Allies:  []*CombatantAuto{actor},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)
	hasExitLog := false
	for _, l := range logs {
		if l.Action == "GU_HOUSE_EXIT" {
			hasExitLog = true
		}
	}

	if !hasExitLog {
		t.Fatal("expected GU_HOUSE_EXIT log event in battle logs")
	}
	if actor.GuHouse.Active {
		t.Fatal("expected GuHouse.Active to become false after EXIT_GU_HOUSE")
	}
}

func TestInteractive_GuHouseActionSelection(t *testing.T) {
	houseAction := SimAction{
		ActionType:    "KILLER_MOVE",
		SourceID:      "GU_HOUSE:h1:strike_1",
		Name:          "Trận Kích Pháo Lôi",
		IsFromGuHouse: true,
		Damage:        250,
		TargetType:    "SINGLE",
	}
	personalGu := SimAction{
		ActionType: "GU_SKILL",
		SourceID:   "moon_gu",
		InstanceID: "inst_moon_1",
		Name:       "Nguyệt Quang",
		Damage:     60,
		TargetType: "SINGLE",
	}

	actor := newTestCombatant("p1", "Cổ Tiên", 1000, 50)
	actor.Kind = "Player"
	actor.Mana = 200
	actor.GuHouse = &GuHouseState{ID: "h1", HP: 1000, MaxHP: 1000, Active: true}
	actor.Actions = []SimAction{houseAction, personalGu}

	enemy := newTestCombatant("e1", "Kẻ Địch", 500, 10)
	enemies := []*CombatantAuto{enemy}

	// 1. Trying to select personalGu while in Gu House -> must return error!
	personalPlan := []CombatActionSelection{
		{ActionType: "GU_SKILL", SourceID: "moon_gu", InstanceID: "inst_moon_1", TargetID: "e1"},
	}
	if _, err := selectCombatPlan(actor, enemies, personalPlan, 3); err == nil {
		t.Fatal("expected selectCombatPlan to fail when selecting personal Gu skill while inside Gu House")
	}

	// 2. Selecting house action -> must succeed!
	housePlan := []CombatActionSelection{
		{ActionType: "KILLER_MOVE", SourceID: "GU_HOUSE:h1:strike_1", TargetID: "e1"},
	}
	if selected, err := selectCombatPlan(actor, enemies, housePlan, 3); err != nil || len(selected) != 1 {
		t.Fatalf("expected selectCombatPlan to succeed for Trận Kích, got err: %v", err)
	}

	// 3. Selecting EXIT_GU_HOUSE -> must succeed!
	exitPlan := []CombatActionSelection{
		{ActionType: "EXIT_GU_HOUSE"},
	}
	if selected, err := selectCombatPlan(actor, enemies, exitPlan, 3); err != nil || len(selected) != 1 {
		t.Fatalf("expected selectCombatPlan to succeed for EXIT_GU_HOUSE, got err: %v", err)
	}
}
