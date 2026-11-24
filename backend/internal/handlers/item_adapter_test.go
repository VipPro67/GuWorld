package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestExecuteItemConsumptionCompiledHP(t *testing.T) {
	raw := []byte(`[{"id":"HP_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}

	item := models.ItemSpec{
		ID:     "test_hp_potion",
		Name:   "Hồi Huyết Đan",
		Blocks: blocks,
	}

	var player models.Player
	player.Rank = 1
	player.HP = 20
	player.BaseHP = 100
	player.MaxMana = 100
	player.CurrentMana = 50

	msgs, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected messages from consumption")
	}
	if player.HP != 70 {
		t.Fatalf("expected HP to be 70, got %v", player.HP)
	}
}

func TestExecuteItemConsumptionTypedHeal(t *testing.T) {
	raw := []byte(`[{"id":"HEAL","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"HEAL","sub_type":"FLAT","resource":"HP","value":50}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}
	item := models.ItemSpec{ID: "typed_heal", Blocks: blocks}
	player := models.Player{}
	player.HP = 20
	player.BaseHP = 100
	if _, err := ExecuteItemConsumption(context.Background(), nil, &player, item); err != nil {
		t.Fatal(err)
	}
	if player.HP != 70 {
		t.Fatalf("HP = %v, want 70", player.HP)
	}
}

func TestItemApplicatorGrantsTypedHumanQi(t *testing.T) {
	player := &models.Player{}
	applicator := ItemApplicator{Player: player, Item: models.ItemSpec{ID: "human_qi_pill", Name: "Nhân Khí Đan"}}
	err := applicator.ApplyEffectCommand(context.Background(), gameplayeffects.Command{
		Operation: gameplayeffects.OpGrantResource, Resource: "HUMAN_QI", Value: 50000, Target: gameplayeffects.TargetSelf,
	})
	if err != nil {
		t.Fatalf("grant HUMAN_QI: %v", err)
	}
	if player.HumanQi != 50000 || len(applicator.Messages) != 1 {
		t.Fatalf("human qi grant result: qi=%v messages=%v", player.HumanQi, applicator.Messages)
	}
}

func TestItemApplicatorRequiresTransactionForApertureMutation(t *testing.T) {
	applicator := ItemApplicator{Player: &models.Player{}, Item: models.ItemSpec{ID: "aperture_pill"}}
	err := applicator.ApplyEffectCommand(context.Background(), gameplayeffects.Command{
		Operation: gameplayeffects.OpModifyAperture, SubType: "AREA_SIZE", Value: 500000, Target: gameplayeffects.TargetSelf,
	})
	if err == nil {
		t.Fatal("expected aperture mutation to require a transaction")
	}
}

func TestItemApplicatorRandomizesTypedTrait(t *testing.T) {
	previousConstitutions, previousRaces := gamefx.ConstitutionRegistry, gamefx.RaceRegistry
	t.Cleanup(func() {
		gamefx.ConstitutionRegistry = previousConstitutions
		gamefx.RaceRegistry = previousRaces
	})
	gamefx.ConstitutionRegistry = map[string]gamefx.ConstitutionDef{"one": {Key: "one", Name: "One"}}
	gamefx.RaceRegistry = map[string]gamefx.RaceDef{"one": {Key: "one", Name: "One"}}

	for _, trait := range []string{"CONSTITUTION", "RACE"} {
		player := &models.Player{}
		player.Constitution, player.Race = "old", "old"
		applicator := ItemApplicator{Player: player, Item: models.ItemSpec{ID: "random_trait", Name: "Đan Ngẫu Nhiên"}}
		err := applicator.ApplyEffectCommand(context.Background(), gameplayeffects.Command{
			Operation: gameplayeffects.OpRandomizeTrait, SubType: trait, Target: gameplayeffects.TargetSelf,
		})
		if err != nil {
			t.Fatalf("randomize %s: %v", trait, err)
		}
		got := player.Constitution
		if trait == "RACE" {
			got = player.Race
		}
		if got != "one" || len(applicator.Messages) != 1 {
			t.Errorf("randomize %s result: key=%q messages=%v", trait, got, applicator.Messages)
		}
	}
}

func TestItemApplicatorRequiresTransactionForApertureRepair(t *testing.T) {
	applicator := ItemApplicator{Player: &models.Player{}, Item: models.ItemSpec{ID: "aperture_repair"}}
	err := applicator.ApplyEffectCommand(context.Background(), gameplayeffects.Command{
		Operation: gameplayeffects.OpRepairAperture, Value: 10000, Target: gameplayeffects.TargetSelf,
	})
	if err == nil {
		t.Fatal("expected aperture repair to require a transaction")
	}
}

func TestItemApplicatorRequiresTransactionForRewardAndPlacement(t *testing.T) {
	tests := []gameplayeffects.Command{
		{Operation: gameplayeffects.OpOpenRewardContainer, SubType: "MORTAL_MINE", Target: gameplayeffects.TargetSelf},
		{Operation: gameplayeffects.OpOpenRewardContainer, SubType: "IMMORTAL_MINE", Target: gameplayeffects.TargetSelf},
		{Operation: gameplayeffects.OpPlaceWorldObject, SubType: "PRIMEVAL_SPRING", Target: gameplayeffects.TargetSelf},
	}
	for _, command := range tests {
		applicator := ItemApplicator{Player: &models.Player{}, Item: models.ItemSpec{ID: "world_item"}}
		if err := applicator.ApplyEffectCommand(context.Background(), command); err == nil {
			t.Errorf("%s/%s should require a transaction", command.Operation, command.SubType)
		}
	}
}

func TestExecuteItemConsumptionRejectsInvalidTypedHeal(t *testing.T) {
	raw := []byte(`[{"id":"HEAL","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"HEAL","sub_type":"PERCENT","resource":"HP","value":50}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}
	item := models.ItemSpec{ID: "invalid_heal", Blocks: blocks}
	player := models.Player{}
	player.HP = 20
	player.BaseHP = 100
	if _, err := ExecuteItemConsumption(context.Background(), nil, &player, item); err == nil {
		t.Fatal("expected unsupported HEAL subtype to be rejected")
	}
	if player.HP != 20 {
		t.Fatalf("HP changed despite invalid effect: %v", player.HP)
	}
}

func TestExecuteItemConsumptionTypedRequireRejectsMortalBeforeGrant(t *testing.T) {
	raw := []byte(`[{"id":"DAO","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"REQUIRE","sub_type":"IMMORTAL"},{"op":"GRANT_RESOURCE","resource":"DAO_MARK","value":1000}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}
	item := models.ItemSpec{ID: "dao_material", Blocks: blocks, Path: `["Fire Path"]`}
	player := models.Player{}
	player.Rank = 1
	if _, err := ExecuteItemConsumption(context.Background(), nil, &player, item); err == nil {
		t.Fatal("expected immortal requirement to reject mortal player")
	}
	if len(player.DaoMarks) != 0 {
		t.Fatalf("Dao Marks changed before requirement validation: %+v", player.DaoMarks)
	}
}

func TestItemApplicatorRejectsMortalRequirement(t *testing.T) {
	player := models.Player{}
	player.Rank = 1
	a := ItemApplicator{Player: &player, Item: models.ItemSpec{ID: "dao_material"}}
	err := a.ApplyEffectCommand(context.Background(), gameplayeffects.Command{
		Operation: gameplayeffects.OpRequire, SubType: "IMMORTAL", Target: gameplayeffects.TargetSelf,
	})
	if err == nil {
		t.Fatal("expected REQUIRE to reject mortal player")
	}
}

func TestExecuteItemConsumptionCompiledMana(t *testing.T) {
	raw := []byte(`[{"id":"MANA_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_MANA","params":{"value":40}}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}

	item := models.ItemSpec{
		ID:     "test_mana_potion",
		Name:   "Hồi Nguyên Đan",
		Blocks: blocks,
	}

	var player models.Player
	player.Rank = 1
	player.HP = 100
	player.BaseHP = 100
	player.MaxMana = 100
	player.CurrentMana = 30

	msgs, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected messages from consumption")
	}
	if player.CurrentMana != 70 {
		t.Fatalf("expected CurrentMana to be 70, got %v", player.CurrentMana)
	}
}

func TestExecuteItemConsumptionMortalOnlyRejectsImmortal(t *testing.T) {
	raw := []byte(`[{"id":"MORTAL_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"MORTAL_ONLY","params":{"value":1}},{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}

	item := models.ItemSpec{
		ID:     "test_mortal_potion",
		Name:   "Phàm Dược",
		Blocks: blocks,
	}

	var player models.Player
	player.Rank = 6 // Immortal

	_, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err == nil {
		t.Fatal("expected mortal_only item to reject immortal player")
	}
}

func TestExecuteItemConsumptionMinRankRequirement(t *testing.T) {
	raw := []byte(`[{"id":"RANK3_POTION","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"MIN_RANK","params":{"value":3}},{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}

	item := models.ItemSpec{
		ID:     "test_rank3_potion",
		Name:   "Tam Chuyển Đan",
		Blocks: blocks,
	}

	var player models.Player
	player.Rank = 2

	_, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err == nil {
		t.Fatal("expected min_rank 3 item to reject rank 2 player")
	}
}

func TestExecuteItemConsumptionFallbackLegacyEffects(t *testing.T) {
	item := models.ItemSpec{
		ID:   "test_legacy_item",
		Name: "Cổ Dược Cũ",
		Effects: models.ItemEffects{
			"restore_hp": 25,
		},
	}

	var player models.Player
	player.Rank = 1
	player.HP = 50
	player.BaseHP = 100

	msgs, err := ExecuteItemConsumption(context.Background(), nil, &player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected message from legacy consumption")
	}
	if player.HP != 75 {
		t.Fatalf("expected HP to be 75, got %v", player.HP)
	}
}

func TestExecuteItemConsumptionRejectsUnsupportedCommandBeforeMutation(t *testing.T) {
	raw := []byte(`[{"id":"MIXED","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_HP","params":{"value":50}},{"op":"STAT_MOD","stat":"ATTACK","sub_type":"ADD_FLAT","value":10}]}]`)
	var blocks gameplayeffects.BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}
	item := models.ItemSpec{ID: "test_mixed", Blocks: blocks}
	player := models.Player{}
	player.HP = 20
	player.BaseHP = 100
	if _, err := ExecuteItemConsumption(context.Background(), nil, &player, item); err == nil {
		t.Fatal("expected unsupported command error")
	}
	if player.HP != 20 {
		t.Fatalf("HP changed before validation: %v", player.HP)
	}
}
