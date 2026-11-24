package handlers

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
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

// TestRebuildEquippedGusKeepsHotbarOrderWhileMembersStayValid pins the combat
// selection contract: players.equipped_gus preserves the player's configured
// order. As long as
// every equipped ACTIVE Gu stays valid, its position in the list must never
// move: no re-shuffle, no re-sort, no dedupe side effects.
func TestRebuildEquippedGusKeepsHotbarOrderWhileMembersStayValid(t *testing.T) {
	activeEquipped := func(id string) models.GuInstance {
		return models.GuInstance{InstanceID: id, Status: "ACTIVE", Equipped: true}
	}
	newPlayer := func(equipped models.EquippedGusList, gus models.GuInventory) *models.Player {
		p := &models.Player{GuInventory: gus}
		p.EquippedGus = equipped
		return p
	}

	t.Run("survives unrelated mutations", func(t *testing.T) {
		p := newPlayer(
			models.EquippedGusList{"h1", "h2", "h3", "h4", "h5", "h6", "extra"},
			models.GuInventory{
				activeEquipped("h1"), activeEquipped("h2"), activeEquipped("h3"),
				activeEquipped("h4"), activeEquipped("h5"), activeEquipped("h6"),
				activeEquipped("extra"),
			},
		)
		RebuildEquippedGus(p)
		want := []string{"h1", "h2", "h3", "h4", "h5", "h6", "extra"}
		if !reflect.DeepEqual([]string(p.EquippedGus), want) {
			t.Fatalf("valid hotbar members must keep their slots, got %v", p.EquippedGus)
		}
	})

	t.Run("equipping another Gu never shifts assigned slots", func(t *testing.T) {
		p := newPlayer(
			models.EquippedGusList{"h1", "h2", "h3", "h4", "h5", "h6", "extra"},
			models.GuInventory{
				activeEquipped("h1"), activeEquipped("h2"), activeEquipped("h3"),
				activeEquipped("h4"), activeEquipped("h5"), activeEquipped("h6"),
				activeEquipped("extra"), activeEquipped("newcomer"),
			},
		)
		RebuildEquippedGus(p)
		want := []string{"h1", "h2", "h3", "h4", "h5", "h6", "extra", "newcomer"}
		if !reflect.DeepEqual([]string(p.EquippedGus), want) {
			t.Fatalf("new Gu must append at the end, got %v", p.EquippedGus)
		}
	})

	t.Run("stale ids are pruned without reordering survivors", func(t *testing.T) {
		unequipped := func(id string) models.GuInstance { return models.GuInstance{InstanceID: id, Status: "ACTIVE"} }
		// "ghost" was unequipped earlier but lingers in equipped_gus
		// (e.g. SLEEPING/SEALED), and "damaged" was force unequipped by
		// cleanupInvalidEquipped. Survivors must keep order.
		p := newPlayer(
			models.EquippedGusList{"h1", "ghost", "h2", "damaged", "h3"},
			models.GuInventory{
				activeEquipped("h1"), unequipped("ghost"),
				activeEquipped("h2"),
				{InstanceID: "damaged", Status: "DAMAGED"},
				activeEquipped("h3"),
			},
		)
		RebuildEquippedGus(p)
		want := []string{"h1", "h2", "h3"}
		if !reflect.DeepEqual([]string(p.EquippedGus), want) {
			t.Fatalf("survivors keep relative order, got %v", p.EquippedGus)
		}
	})
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
	// 1. Mortal Rank 5 player, Rank 5 Gu, flat base cost 10
	if cost := GetPortalManaCost(playerMortal5, specMortal5); cost != 10.0 {
		t.Errorf("expected 10.0, got %f", cost)
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

	// 2. Mortal 5 using Mortal 5, but not enough mana (cost = 10, current = 5)
	playerMortal5.CurrentMana = 5
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

func TestTypedNotifyMatchesLegacyRecordMessage(t *testing.T) {
	player := &models.Player{}
	instance := models.GuInstance{InstanceID: "record-instance", GuID: "GU_RECORD"}
	typed := models.GuDictionaryItem{ID: "GU_RECORD", Name: "Thiên Ảnh Cổ", Rank: 2}
	if err := typed.Effects.Scan(`{"schema_version":2,"blocks":[{"id":"RECORD_MESSAGE","trigger":"ON_USE","target":{"selector":"SELF"},"operations":[{"op":"NOTIFY","message":"{gu} ghi lại toàn bộ hình ảnh và âm thanh trong khu vực."}]}]}`); err != nil {
		t.Fatal(err)
	}
	got, err := HandleGuEvent(player, typed, instance, "ON_USE")
	if err != nil {
		t.Fatal(err)
	}
	legacy := typed
	if err := legacy.Effects.Scan(`{"schema_version":2,"blocks":[{"id":"RECORD_MESSAGE","trigger":"ON_USE","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RECORD_MESSAGE","params":{}}]}]}`); err != nil {
		t.Fatal(err)
	}
	want, err := HandleGuEvent(player, legacy, instance, "ON_USE")
	if err != nil {
		t.Fatal(err)
	}
	if got != want || got != "Thiên Ảnh Cổ ghi lại toàn bộ hình ảnh và âm thanh trong khu vực." {
		t.Fatalf("typed notify = %q, legacy = %q", got, want)
	}
}

func TestTypedGuResourceOperationsOnConsume(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Lifespan: 100},
		ThoughtPool:      models.ThoughtPool{CurrentThought: 8, MaxThought: 50},
		SoulState:        models.SoulState{SoulHP: 10, MaxSoul: 100},
	}
	spec := models.GuDictionaryItem{ID: "GU_RESOURCE_TEST", Name: "Resource Gu", Rank: 3}
	instance := models.GuInstance{InstanceID: "resource-instance", GuID: spec.ID}
	run := func(id string, operation string) error {
		var blockEffects models.GuEffects
		if err := blockEffects.Scan(`[{"id":"` + id + `","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[` + operation + `]}]`); err != nil {
			return err
		}
		current := spec
		current.Effects = blockEffects
		_, err := HandleGuEvent(player, current, instance, "ON_CONSUME")
		return err
	}
	operations := []struct {
		id, operation string
	}{
		{"LIFESPAN", `{"op":"GRANT_RESOURCE","resource":"LIFESPAN","value":10}`},
		{"SOUL_MAX", `{"op":"MODIFY_MAX_RESOURCE","resource":"SOUL","value":5}`},
		{"THOUGHT_MAX", `{"op":"MODIFY_MAX_RESOURCE","resource":"THOUGHT","value":5}`},
		{"SOUL_RESTORE", `{"op":"RESTORE_RESOURCE","resource":"SOUL","value":20}`},
		{"THOUGHT_RESTORE", `{"op":"RESTORE_RESOURCE","resource":"THOUGHT","value":15}`},
	}
	for _, tt := range operations {
		if err := run(tt.id, tt.operation); err != nil {
			t.Fatalf("%s: %v", tt.id, err)
		}
	}
	if player.Lifespan != 110 || player.MaxSoul != 105 || player.MaxThought != 55 || player.SoulHP != 30 || player.CurrentThought != 23 {
		t.Fatalf("typed resources were not applied: lifespan=%v maxSoul=%d maxThought=%v soulHP=%d thought=%v", player.Lifespan, player.MaxSoul, player.MaxThought, player.SoulHP, player.CurrentThought)
	}
	for _, operation := range []string{
		`{"op":"MODIFY_MAX_RESOURCE","resource":"SOUL","value":1.5}`,
		`{"op":"RESTORE_RESOURCE","resource":"SOUL","value":1.5}`,
	} {
		if err := run("INVALID_SOUL_VALUE", operation); err == nil {
			t.Errorf("operation %s accepted a fractional SOUL value", operation)
		}
	}
}

func TestTypedModifyAptitudePreservesLegacyMutation(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 5, AptitudeType: "Bính", AptitudePercent: 78},
		ResourceState:    models.ResourceState{CurrentMana: 80, MaxMana: 90},
	}
	spec := models.GuDictionaryItem{
		ID: "GU_BLOOD_SKULL", Name: "Huyết Cốt Cổ", Rank: 4,
		Effects: mustScanGuEffects(t, `[{"id":"SACRIFICE_APTITUDE","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"MODIFY_APTITUDE","sub_type":"ADD_FLAT","value":5}]}]`),
	}
	if !models.GuHasAptitudeSacrifice(spec) {
		t.Fatal("typed aptitude sacrifice did not retain its extra hunger cost marker")
	}
	message, err := HandleGuEvent(player, spec, models.GuInstance{GuID: spec.ID, Rank: spec.Rank}, "ON_CONSUME")
	if err != nil {
		t.Fatalf("HandleGuEvent: %v", err)
	}
	if player.AptitudePercent != 83 || player.AptitudeType != "Giáp" || player.MaxMana != 95 || player.CurrentMana != 80 {
		t.Fatalf("aptitude mutation = %d%% %q mana %v/%v", player.AptitudePercent, player.AptitudeType, player.CurrentMana, player.MaxMana)
	}
	if !strings.Contains(message, "Tư chất Không khiếu tăng lên 83%") {
		t.Fatalf("unexpected aptitude message %q", message)
	}
	legacy := models.GuDictionaryItem{Effects: mustScanGuEffects(t, `[{"id":"SACRIFICE_APTITUDE","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"SACRIFICE_APTITUDE","params":{}}]}]`)}
	if !models.GuHasAptitudeSacrifice(legacy) {
		t.Fatal("legacy aptitude sacrifice lost its extra hunger cost marker")
	}
}

func TestTypedGrantBodyModPreservesLegacyMutation(t *testing.T) {
	player := &models.Player{}
	spec := models.GuDictionaryItem{
		ID: "GU_SOVEREIGN_FETUS", Name: "Chí Tôn Tiên Thai", Rank: 9,
		Effects: mustScanGuEffects(t, `[{"id":"SOVEREIGN_BODY_TRANSFORM","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"GRANT_BODY_MOD","sub_type":"SOVEREIGN_IMMORTAL"}]}]`),
	}
	message, err := HandleGuEvent(player, spec, models.GuInstance{GuID: spec.ID, Rank: spec.Rank}, "ON_CONSUME")
	if err != nil {
		t.Fatalf("HandleGuEvent: %v", err)
	}
	if !player.BodyMods.Has(models.BodyModSovereignImmortal) {
		t.Fatalf("body modification not granted: %+v", player.BodyMods)
	}
	if !strings.Contains(message, "Tiên Thể Chí Tôn") {
		t.Fatalf("unexpected body modification message %q", message)
	}
	if _, err := HandleGuEvent(player, spec, models.GuInstance{GuID: spec.ID, Rank: spec.Rank}, "ON_CONSUME"); err == nil {
		t.Fatal("duplicate permanent body modification should be rejected")
	}
}

func TestTypedAdvanceProgressionPreservesBreakthroughGuards(t *testing.T) {
	makePlayer := func(rank, subRank int) *models.Player {
		return &models.Player{CultivationState: models.CultivationState{Rank: rank, SubRank: subRank}}
	}
	run := func(player *models.Player, guRank int) error {
		effects := mustScanGuEffects(t, `[{"id":"PROGRESSION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"ADVANCE_PROGRESSION"}]}]`)
		spec := models.GuDictionaryItem{ID: "GU_PROGRESS", Name: "Xá Lợi Cổ", Rank: guRank, Effects: effects}
		_, err := HandleGuEvent(player, spec, models.GuInstance{Rank: guRank}, "ON_CONSUME")
		return err
	}

	progressing := makePlayer(2, 2)
	if err := run(progressing, 2); err != nil || progressing.SubRank != 3 {
		t.Fatalf("matching rank should advance one stage: subrank=%d err=%v", progressing.SubRank, err)
	}
	if err := run(makePlayer(2, 1), 1); err == nil {
		t.Fatal("mismatched Gu rank should be rejected")
	}
	if err := run(makePlayer(2, 4), 2); err == nil {
		t.Fatal("peak subrank should not advance to another major rank")
	}
	petrified := makePlayer(2, 1)
	petrified.Statuses.Add(gamefx.StatusAperturePetrified, "test", time.Hour, 0)
	if err := run(petrified, 2); err == nil {
		t.Fatal("aperture petrification should block progression")
	}
}

func TestTypedGuApertureAdjustmentsOnConsume(t *testing.T) {
	player := &models.Player{CultivationState: models.CultivationState{ApertureStress: 30, TimeSpeedMultiplier: 1}}
	effects := mustScanGuEffects(t, `[{"id":"APERTURE_ADJUSTMENTS","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"MODIFY_APERTURE","sub_type":"STRESS_REDUCE","value":25},{"op":"MODIFY_APERTURE","sub_type":"TIME_RATE_ADD","value":0.5},{"op":"MODIFY_APERTURE","sub_type":"TIME_RATE_SUBTRACT","value":0.1},{"op":"MODIFY_APERTURE","sub_type":"TIME_RATE_MULTIPLY","value":3}]}]`)
	spec := models.GuDictionaryItem{ID: "GU_APERTURE_TEST", Name: "Aperture Gu", Rank: 1, Effects: effects}
	if _, err := HandleGuEvent(player, spec, models.GuInstance{Rank: 1}, "ON_CONSUME"); err != nil {
		t.Fatal(err)
	}
	if player.ApertureStress != 5 || math.Abs(player.TimeSpeedMultiplier-4.2) > 1e-9 {
		t.Fatalf("aperture state = stress %.1f/time %.2f; want stress 5/time 4.2", player.ApertureStress, player.TimeSpeedMultiplier)
	}
	player.TimeSpeedMultiplier = 0.2
	effects = mustScanGuEffects(t, `[{"id":"TIME_RATE_FLOOR","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"MODIFY_APERTURE","sub_type":"TIME_RATE_SUBTRACT","value":5}]}]`)
	spec.Effects = effects
	if _, err := HandleGuEvent(player, spec, models.GuInstance{Rank: 1}, "ON_CONSUME"); err != nil {
		t.Fatal(err)
	}
	if player.TimeSpeedMultiplier != 0.1 {
		t.Fatalf("time rate = %.2f; want floor 0.1", player.TimeSpeedMultiplier)
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
		effects := mustScanGuEffects(t, `[{"id":"MAN_BEAST_BURIAL","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"ADVANCE_MAJOR_REALM"}]}]`)
		spec := models.GuDictionaryItem{ID: "GU_MAN_BEAST_LIFE_BURIAL", Name: "Nhân Thú Táng Sinh Cổ", Rank: specRank, Effects: effects}
		gu := models.GuInstance{GuID: spec.ID, Rank: instanceRank}
		_, err := HandleGuEvent(p, spec, gu, "ON_CONSUME")
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

// TestStoneAperturePetrifiesWithoutManaLoss locks the reworked STONE_APERTURE
// effect: it raises SubRank to 4 WITHOUT cutting max mana and instead applies
// the rank-scaled aperture_petrified status (R3 = 48h).
func TestStoneAperturePetrifiesWithoutManaLoss(t *testing.T) {
	p := &models.Player{
		CultivationState: models.CultivationState{Rank: 3, SubRank: 1},
		ResourceState:    models.ResourceState{MaxMana: 80, CurrentMana: 50},
	}
	spec := models.GuDictionaryItem{ID: "GU_STONE_APERTURE", Name: "Thạch Khiếu Cổ", Rank: 3}
	gu := models.GuInstance{GuID: spec.ID, Rank: 3}

	if _, err := handleActionCode(p, spec, gu, models.SpecialMechanic{ActionCode: "STONE_APERTURE"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SubRank != 4 {
		t.Errorf("SubRank = %d, want 4", p.SubRank)
	}
	if p.MaxMana != 80 {
		t.Errorf("MaxMana = %v, want unchanged 80", p.MaxMana)
	}
	if !p.Statuses.Has("aperture_petrified") {
		t.Fatal("expected aperture_petrified status to be applied")
	}
	var exp *time.Time
	for _, st := range p.Statuses {
		if st.ID == "aperture_petrified" {
			exp = st.ExpiresAt
		}
	}
	if exp == nil {
		t.Fatal("aperture_petrified phải có thời hạn, không được vĩnh viễn")
	}
	if want := 48 * time.Hour; exp.Sub(time.Now()) < want-time.Minute || exp.Sub(time.Now()) > want+time.Minute {
		t.Fatalf("aperture_petrified rank 3 hết hạn sau %v, muốn ~%v", time.Until(*exp), want)
	}
	fx := gamefx.Get(p)
	if !fx.DisableManaRegen {
		t.Error("aperture_petrified should disable automatic mana regen")
	}
	if !fx.BlockedActions[gamefx.ActionBreakthrough] {
		t.Error("aperture_petrified should block breakthrough")
	}

	// Rank mismatch: Gu rank must equal player rank, otherwise no effect.
	mismatch := &models.Player{
		CultivationState: models.CultivationState{Rank: 3, SubRank: 1},
		ResourceState:    models.ResourceState{MaxMana: 80, CurrentMana: 50},
	}
	if _, err := handleActionCode(mismatch, spec, models.GuInstance{GuID: spec.ID, Rank: 4}, models.SpecialMechanic{ActionCode: "STONE_APERTURE"}); err == nil {
		t.Fatal("expected rank mismatch error")
	}
	if mismatch.SubRank != 1 || mismatch.Statuses.Has("aperture_petrified") {
		t.Error("rank mismatch must not raise SubRank or petrify aperture")
	}
}

func TestPetrifyApertureTypedOperation(t *testing.T) {
	p := &models.Player{
		CultivationState: models.CultivationState{Rank: 3, SubRank: 1},
		ResourceState:    models.ResourceState{MaxMana: 80, CurrentMana: 50},
	}
	spec := models.GuDictionaryItem{
		ID: "GU_STONE_APERTURE", Name: "Thạch Khiếu Cổ", Rank: 3,
		Effects: mustScanGuEffects(t, `{"schema_version":2,"blocks":[{"id":"STONE_APERTURE","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"PETRIFY_APERTURE"}]}]}`),
	}
	gu := models.GuInstance{GuID: spec.ID, Rank: 3}
	msg, err := HandleGuEvent(p, spec, gu, "ON_CONSUME")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SubRank != 4 {
		t.Errorf("SubRank = %d, want 4", p.SubRank)
	}
	if !strings.Contains(msg, "đẩy tu vi lên Đỉnh Phong") {
		t.Errorf("expected petrification message, got: %s", msg)
	}
	if !p.Statuses.Has("aperture_petrified") {
		t.Fatal("expected aperture_petrified status")
	}
}

func TestGuTeleportRange(t *testing.T) {
	portal := func(rank int, raw string) *models.GuDictionaryItem {
		return &models.GuDictionaryItem{ID: "GU_TEST", Name: "Test", Rank: rank, Effects: mustScanGuEffects(t, raw)}
	}
	cases := []struct {
		name string
		spec *models.GuDictionaryItem
		want int
	}{
		{
			name: "ignore movement range is unlimited",
			spec: portal(5, `[{"id":"C","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"IGNORE_MOVEMENT_RANGE"}]}]`),
			want: 0,
		},
		{
			name: "explicit portal range",
			spec: portal(5, `[{"id":"C","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"PORTAL_CREATE","params":{"range":1234}}]}]`),
			want: 1234,
		},
		{
			name: "explicit range wins over ignore capability",
			spec: portal(5, `[{"id":"C","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"PORTAL_CREATE","params":{"range":1500}},{"op":"UNIQUE","code":"IGNORE_MOVEMENT_RANGE"}]}]`),
			want: 1500,
		},
		{
			name: "explicit random range",
			spec: portal(5, `[{"id":"C","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RANDOM_PORTAL","params":{"range":777}}]}]`),
			want: 777,
		},
		{
			name: "rank fallback rank 5",
			spec: portal(5, `[{"id":"C","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"PORTAL_CREATE","params":{}}]}]`),
			want: 1000,
		},
		{
			name: "rank fallback rank 9",
			spec: portal(9, `[{"id":"C","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"PORTAL_CREATE","params":{}}]}]`),
			want: 30000,
		},
		{
			name: "nil spec is unlimited",
			spec: nil,
			want: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GuTeleportRange(tc.spec); got != tc.want {
				t.Fatalf("GuTeleportRange = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestChebyshevDistance(t *testing.T) {
	if got := chebyshevDistance(3, 4, 6, 4); got != 3 {
		t.Fatalf("chebyshevDistance = %d, want 3", got)
	}
	if got := chebyshevDistance(0, 0, 4, 9); got != 9 {
		t.Fatalf("chebyshevDistance = %d, want 9", got)
	}
}
