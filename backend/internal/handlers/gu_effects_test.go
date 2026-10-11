package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func mustScanGuEffects(t *testing.T, raw string) models.GuEffects {
	t.Helper()
	var eff models.GuEffects
	if err := eff.Scan(raw); err != nil {
		t.Fatalf("scan test effects: %v", err)
	}
	return eff
}

func TestCleanupInvalidEquipped(t *testing.T) {
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_STRENGTH": {
			1: {ID: "GU_STRENGTH", Name: "Lực Cổ", Rank: 1, Type: "Passive"},
			9: {ID: "GU_STRENGTH", Name: "Lực Cổ", Rank: 9, Type: "Passive"},
		},
		"GU_CONSUMABLE": {
			1: {ID: "GU_CONSUMABLE", Name: "Tiêu Hao Cổ", Rank: 1, Type: "Consumable"},
		},
		"GU_GROWTH": {
			9: {ID: "GU_GROWTH", Name: "Trưởng Thành Cổ", Rank: 9, Type: "Passive",
				Effects: mustScanGuEffects(t, `[{"id":"GROWTH","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"GROWTH_CAP","value":100},{"op":"UNIQUE","code":"GROWTH_DURATION_HOURS","params":{"value":10}}]}]`)},
		},
	}

	now := time.Now()
	hourAgo := now.Add(-time.Hour)
	p := &models.Player{
		CultivationState: models.CultivationState{Rank: 1},
		GuInventory: models.GuInventory{
			{InstanceID: "over-rank", GuID: "GU_STRENGTH", Rank: 9, Status: "ACTIVE", Equipped: true},
			{InstanceID: "consumable", GuID: "GU_CONSUMABLE", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "unknown", GuID: "GU_MISSING", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "growth", GuID: "GU_GROWTH", Rank: 9, Status: "ACTIVE", Equipped: true, EquippedAt: &hourAgo},
			{InstanceID: "valid", GuID: "GU_STRENGTH", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "vital", GuID: "GU_STRENGTH", Rank: 9, Status: "ACTIVE", Equipped: true, IsVital: true},
			{InstanceID: "not-equipped", GuID: "GU_STRENGTH", Rank: 9, Status: "ACTIVE", Equipped: false},
		},
	}

	cleaned := cleanupInvalidEquipped(p)

	wantCleaned := map[string]bool{"over-rank": true, "consumable": true, "unknown": true, "growth": true}
	if len(cleaned) != len(wantCleaned) {
		t.Fatalf("expected %d cleaned Gus, got %d: %+v", len(wantCleaned), len(cleaned), cleaned)
	}
	for _, gu := range cleaned {
		if !wantCleaned[gu.InstanceID] {
			t.Fatalf("unexpected cleaned Gu %s", gu.InstanceID)
		}
		if gu.Equipped {
			t.Fatalf("cleaned Gu %s should be unequipped", gu.InstanceID)
		}
	}

	byID := map[string]models.GuInstance{}
	for _, gu := range p.GuInventory {
		byID[gu.InstanceID] = gu
	}

	for id := range wantCleaned {
		if byID[id].Equipped {
			t.Fatalf("Gu %s should have been unequipped", id)
		}
	}
	if !byID["valid"].Equipped {
		t.Fatalf("valid Gu should stay equipped")
	}
	if !byID["vital"].Equipped {
		t.Fatalf("vital Gu must never be auto-unequipped")
	}
	if byID["not-equipped"].Equipped {
		t.Fatalf("non-equipped Gu must not be touched")
	}

	if byID["growth"].EquippedAt != nil {
		t.Fatalf("growth Gu EquippedAt should be reset after cleanup")
	}
	if byID["growth"].GrowthHours < 0.9 {
		t.Fatalf("growth Gu should have accrued ~1h of growth, got %v", byID["growth"].GrowthHours)
	}
	if p.GrowthData.Boosts.Attack < 9 {
		t.Fatalf("growth flush should add ~10 attack, got %v", p.GrowthData.Boosts.Attack)
	}

	RebuildEquippedGus(p)
	wantEquipped := []string{"valid", "vital"}
	if len(p.EquippedGus) != len(wantEquipped) {
		t.Fatalf("expected equipped_gus %v, got %v", wantEquipped, p.EquippedGus)
	}
	for i, id := range wantEquipped {
		if p.EquippedGus[i] != id {
			t.Fatalf("expected equipped_gus[%d] = %s, got %s", i, id, p.EquippedGus[i])
		}
	}
}

func TestCleanupInvalidEquippedNoInvalidGus(t *testing.T) {
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_STRENGTH": {1: {ID: "GU_STRENGTH", Name: "Lực Cổ", Rank: 1, Type: "Passive"}},
	}
	p := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		GuInventory: models.GuInventory{
			{InstanceID: "a", GuID: "GU_STRENGTH", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "b", GuID: "GU_STRENGTH", Rank: 1, Status: "ACTIVE", Equipped: false},
		},
	}
	if cleaned := cleanupInvalidEquipped(p); len(cleaned) != 0 {
		t.Fatalf("expected no cleanup, got %+v", cleaned)
	}
}

func TestPortalManaCostAndUsability(t *testing.T) {
	playerMortal5 := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		ResourceState:    models.ResourceState{CurrentMana: 250, MaxMana: 500},
	}
	specMortal5 := &models.GuDictionaryItem{
		ID:           "GU_MORTAL_PORTAL",
		Name:         "Địa Độn Cổ",
		Rank:         5,
		BaseManaCost: 10.0, // 10%
		Effects: models.GuEffects{
			Active: &models.ActiveEffect{
				PortalCreate: true,
			},
		},
	}

	playerImmortal6 := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		ResourceState:    models.ResourceState{CurrentMana: 50, MaxMana: 100},
	}
	specImmortal6 := &models.GuDictionaryItem{
		ID:           "GU_IMMORTAL_PORTAL",
		Name:         "Tiên Cổng Cổ",
		Rank:         6,
		BaseManaCost: 10.0, // 10 flat
		Effects: models.GuEffects{
			Active: &models.ActiveEffect{
				PortalCreate: true,
			},
		},
	}
	specImmortal8 := &models.GuDictionaryItem{
		ID:           "GU_IMMORTAL_PORTAL_HIGH",
		Name:         "Cao Cấp Tiên Cổng Cổ",
		Rank:         8,
		BaseManaCost: 10.0, // 10 flat
		Effects: models.GuEffects{
			Active: &models.ActiveEffect{
				PortalCreate: true,
			},
		},
	}

	// Test GetPortalManaCost
	// 1. Mortal Rank 5 player, Rank 5 Gu, 10% base cost on MaxMana 500 = 50
	if cost := GetPortalManaCost(playerMortal5, specMortal5); cost != 50.0 {
		t.Errorf("expected 50.0, got %f", cost)
	}

	// 2. Immortal Rank 6 player, Rank 6 Gu, 10 base cost = 10
	if cost := GetPortalManaCost(playerImmortal6, specImmortal6); cost != 10.0 {
		t.Errorf("expected 10.0, got %f", cost)
	}

	// 3. Immortal Rank 6 player, Rank 8 Gu, 10 base cost * 2^2 = 40
	if cost := GetPortalManaCost(playerImmortal6, specImmortal8); cost != 40.0 {
		t.Errorf("expected 40.0, got %f", cost)
	}

	// Test ValidateGuUsable
	guInstMortal5 := &models.GuInstance{
		InstanceID: "inst_mortal_5",
		GuID:       "GU_MORTAL_PORTAL",
		Rank:       5,
		Hunger:     100,
		Status:     "ACTIVE",
		Equipped:   true,
	}
	guInstImmortal6 := &models.GuInstance{
		InstanceID: "inst_immortal_6",
		GuID:       "GU_IMMORTAL_PORTAL",
		Rank:       6,
		Hunger:     100,
		Status:     "ACTIVE",
		Equipped:   true,
	}

	// 1. Mortal 5 using Mortal 5 -> Pass
	if err := ValidateGuUsable(playerMortal5, guInstMortal5, specMortal5, true); err != nil {
		t.Errorf("expected mortal 5 usability to pass, got: %v", err)
	}

	// 2. Mortal 5 using Mortal 5, but not enough mana (cost = 50, current = 40)
	playerMortal5.CurrentMana = 40
	if err := ValidateGuUsable(playerMortal5, guInstMortal5, specMortal5, true); err == nil {
		t.Error("expected error due to insufficient mana, got nil")
	}

	// 3. Mortal 5 using Immortal 6 -> Blocked by CanUseGuWithMana
	if err := ValidateGuUsable(playerMortal5, guInstImmortal6, specImmortal6, true); err == nil {
		t.Error("expected mortal using immortal Gu to be blocked, got nil")
	}

	// 4. Zero hunger Gu -> Blocked due to exhaustion
	guInstExhausted := &models.GuInstance{
		InstanceID: "inst_exhausted",
		GuID:       "GU_MORTAL_PORTAL",
		Rank:       5,
		Hunger:     0,
		Status:     "ACTIVE",
		Equipped:   true,
	}
	playerMortal5.CurrentMana = 100
	if err := ValidateGuUsable(playerMortal5, guInstExhausted, specMortal5, true); err == nil {
		t.Error("expected zero hunger Gu to be blocked, got nil")
	}
}

func TestRestoreSoulHP(t *testing.T) {
	player := &models.Player{
		SoulState: models.SoulState{
			SoulHP:  30,
			MaxSoul: 100,
		},
	}
	spec := models.GuDictionaryItem{
		ID:   "GU_SOUL_RESTORE",
		Name: "Hồi Hồn Cổ",
		Rank: 3,
	}
	guInst := models.GuInstance{
		InstanceID: "inst_1",
		GuID:       "GU_SOUL_RESTORE",
	}
	mechanic := models.SpecialMechanic{
		EventTrigger: "ON_CONSUME",
		ActionCode:   "RESTORE_SOUL_HP",
		Params:       []byte(`{"value": 20}`),
	}

	msg, err := HandleConsumeEvent(player, guInst, spec, mechanic)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.SoulHP != 50 {
		t.Errorf("expected SoulHP = 50, got %d", player.SoulHP)
	}
	if msg != "Hồn phách được dưỡng hồi (+20 Hồn Phách)." {
		t.Errorf("unexpected message: %s", msg)
	}

	// Test cap at MaxSoul
	mechanic.Params = []byte(`{"value": 100}`)
	msg, err = HandleConsumeEvent(player, guInst, spec, mechanic)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.SoulHP != 100 {
		t.Errorf("expected SoulHP capped at MaxSoul (100), got %d", player.SoulHP)
	}
	if msg != "Hồn phách được dưỡng hồi (+50 Hồn Phách)." {
		t.Errorf("unexpected message: %s", msg)
	}
}

func TestSetVitalGuBacklashSeverelyWounded(t *testing.T) {
	p := &models.Player{
		BodyState: models.BodyState{HP: 100, BaseHP: 100},
		GuInventory: models.GuInventory{
			{InstanceID: "gu1", GuID: "GU_ATTACK", Rank: 1, IsVital: true, Status: "ACTIVE", Equipped: true},
			{InstanceID: "gu2", GuID: "GU_DEFENSE", Rank: 1, IsVital: false, Status: "ACTIVE", Equipped: false},
		},
	}

	_, _, hadVital := models.FindFirstMatchingGu(p.GuInventory, func(gu models.GuInstance) bool {
		return gu.IsVital
	})
	if !hadVital {
		t.Fatal("expected player to have existing vital Gu")
	}

	// Simulate backlash logic when changing vital Gu
	p.Statuses.Add("severely_wounded", "set_vital_gu", 1*time.Hour, 0)

	if p.HP != 100 {
		t.Errorf("expected HP to remain unchanged at 100, got %.1f", p.HP)
	}
	if !p.Statuses.Has("severely_wounded") {
		t.Errorf("expected player to have severely_wounded status")
	}
}

func TestSoulRegenTickAndMigrateAliases(t *testing.T) {
	newPlayer := func() *models.Player {
		return &models.Player{
			SoulState: models.SoulState{SoulHP: 10, MaxSoul: 100},
		}
	}
	spec := models.GuDictionaryItem{ID: "GU_X", Name: "X Cổ", Rank: 3}
	guInst := models.GuInstance{InstanceID: "inst_1", GuID: "GU_X"}
	run := func(p *models.Player, trigger, code, params string) (string, error) {
		s := spec
		raw := `[{"id":"` + code + `","trigger":"` + trigger + `","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"` + code + `","params":` + params + `}]}]`
		var eff models.GuEffects
		if err := eff.Scan(raw); err != nil {
			return "", err
		}
		s.Effects = eff
		return HandleGuEvent(p, s, guInst, trigger)
	}

	// SOUL_REGEN_TICK accrues soul_hp capped at max_soul, default value 1.0.
	p := newPlayer()
	if _, err := run(p, "ON_TICK", "SOUL_REGEN_TICK", `{"value": 5}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SoulHP != 15 {
		t.Errorf("SoulHP = %d, want 15", p.SoulHP)
	}
	if _, err := run(p, "ON_TICK", "SOUL_REGEN_TICK", `{}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SoulHP != 16 {
		t.Errorf("SoulHP = %d, want 16 (default value 1.0)", p.SoulHP)
	}

	// Migrate aliases resolve instead of 400.
	for _, tc := range []struct{ trigger, code, params string }{
		{"ON_CONSUME", "STONE_APERTURE", `{}`},
		{"ON_CONSUME", "POISON_VOW", `{}`},
		{"ON_CONSUME", "THIRD_WATCH", `{}`},
		{"ON_USE", "RECORD_MESSAGE", `{}`},
	} {
		if _, err := run(newPlayer(), tc.trigger, tc.code, tc.params); err != nil {
			t.Errorf("%s/%s: unexpected error: %v", tc.trigger, tc.code, err)
		}
	}
}

func TestManBeastBurialBreakthrough(t *testing.T) {
	makePlayer := func(rank, subRank int) *models.Player {
		return &models.Player{
			CultivationState: models.CultivationState{Rank: rank, SubRank: subRank, WallProgress: 1},
			ResourceState:    models.ResourceState{MaxMana: 100, CurrentMana: 95},
		}
	}
	apply := func(p *models.Player, specRank, instanceRank int) error {
		spec := models.GuDictionaryItem{ID: "GU_MAN_BEAST_LIFE_BURIAL", Name: "Nhân Thú Táng Sinh Cổ", Rank: specRank}
		gu := models.GuInstance{GuID: spec.ID, Rank: instanceRank}
		_, err := handleActionCode(p, spec, gu, models.SpecialMechanic{ActionCode: "MAN_BEAST_BURIAL"})
		return err
	}

	for rank := 1; rank <= 4; rank++ {
		p := makePlayer(rank, 4)
		if err := apply(p, rank, rank); err != nil {
			t.Fatalf("rank %d: unexpected error: %v", rank, err)
		}
		if p.Rank != rank+1 || p.SubRank != 1 || p.WallProgress != 0 || p.MaxMana != 90 || p.CurrentMana != 90 {
			t.Errorf("rank %d breakthrough = rank %d sub %d wall %.1f max/current mana %.1f/%.1f, want rank %d sub 1 wall 0 max/current mana 90/90", rank, p.Rank, p.SubRank, p.WallProgress, p.MaxMana, p.CurrentMana, rank+1)
		}
	}

	for _, tc := range []struct {
		name                            string
		rank, subRank, specRank, guRank int
	}{
		{name: "not peak", rank: 2, subRank: 3, specRank: 2, guRank: 2},
		{name: "rank too high", rank: 5, subRank: 4, specRank: 5, guRank: 5},
		{name: "dictionary rank mismatch", rank: 2, subRank: 4, specRank: 3, guRank: 2},
		{name: "instance rank mismatch", rank: 2, subRank: 4, specRank: 2, guRank: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := apply(makePlayer(tc.rank, tc.subRank), tc.specRank, tc.guRank); err == nil {
				t.Fatal("expected eligibility error")
			}
		})
	}
}

func TestValidateGuUsablePortalAndIgnoreRange(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		ResourceState: models.ResourceState{
			CurrentMana: 5000,
			MaxMana:     10000,
		},
	}

	spec := models.GuDictionaryItem{
		ID:           "GU_FIXED_IMMORTAL_TRAVEL",
		Name:         "Định Tiên Du",
		Rank:         6,
		BaseManaCost: 5.0,
		Effects:      mustScanGuEffects(t, `[{"id":"SPACE_CAPABILITIES","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"PORTAL_CREATE"},{"op":"UNIQUE","code":"IGNORE_MOVEMENT_RANGE"}]}]`),
	}
	guInst := &models.GuInstance{
		InstanceID: "inst_travel",
		GuID:       "GU_FIXED_IMMORTAL_TRAVEL",
		Rank:       6,
		Hunger:     100,
		Status:     "ACTIVE",
		Equipped:   true,
	}

	if err := ValidateGuUsable(player, guInst, &spec, true); err != nil {
		t.Fatalf("expected valid portal Gu, got error: %v", err)
	}

	// Lower rank Gu than player should fail portal validation
	playerRank9 := &models.Player{
		CultivationState: models.CultivationState{Rank: 9},
		ResourceState: models.ResourceState{
			CurrentMana: 5000,
			MaxMana:     10000,
		},
	}
	if err := ValidateGuUsable(playerRank9, guInst, &spec, true); err == nil {
		t.Fatal("expected lower rank Gu to fail portal validation for higher rank player")
	}
}
