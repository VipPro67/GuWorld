package loot

import (
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGrantKeyValidationAndIdempotencyKey(t *testing.T) {
	key := GrantKey{ZoneID: "z", DeathID: "d", RecipientID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), TemplateID: "mob", RewardVersion: 1}
	if err := key.validate(); err != nil {
		t.Fatal(err)
	}
	other := key
	other.RecipientID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	if key.IdempotencyKey() == other.IdempotencyKey() {
		t.Fatal("different recipients produced the same idempotency key")
	}
	key.RewardVersion = 0
	if err := key.validate(); err == nil {
		t.Fatal("zero reward version accepted")
	}
	key.RewardVersion = 1
	key.RecipientID = uuid.Nil
	if err := key.validate(); err == nil {
		t.Fatal("nil recipient accepted")
	}
}

func TestGenerationScopedDeathIDsKeepLootKeysDistinct(t *testing.T) {
	base := GrantKey{
		ZoneID: "forest", DeathID: "generation-a:1:m:0",
		RecipientID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		TemplateID:  "mob", RewardVersion: 1,
	}
	nextGeneration := base
	nextGeneration.DeathID = "generation-b:1:m:0"
	if base.IdempotencyKey() == nextGeneration.IdempotencyKey() {
		t.Fatal("different world generations reused a loot idempotency key")
	}
	replayed := base
	if base.IdempotencyKey() != replayed.IdempotencyKey() {
		t.Fatal("replaying the same death changed its idempotency key")
	}
}

func TestRollFiltersChanceRankAndMissingCatalog(t *testing.T) {
	oldItem, hadItem := models.ItemRegistry["test_item"]
	models.ItemRegistry["test_item"] = models.ItemSpec{ID: "test_item", Type: "Material"}
	defer func() {
		if hadItem {
			models.ItemRegistry["test_item"] = oldItem
		} else {
			delete(models.ItemRegistry, "test_item")
		}
	}()
	table := models.LootTable{ID: "test", Entries: []models.LootEntry{
		{ItemID: "test_item", MinQty: 2, MaxQty: 2, Chance: 1},
		{ItemID: "missing_item", MinQty: 1, MaxQty: 1, Chance: 1},
		{ItemID: "test_item", MinQty: 1, MaxQty: 1, Chance: 0},
		{ItemID: "test_item", MinQty: 1, MaxQty: 1, Chance: 1, Rank: 5},
		{GuID: "missing_gu", Rank: 1, Chance: 1},
	}}
	service := NewServiceWithRand(nil, nil, rand.New(rand.NewSource(1)))
	rewards := service.roll(table, 0)
	if len(rewards) != 2 || rewards[0].ID != "test_item" || rewards[0].Quantity != 2 {
		t.Fatalf("unexpected rewards: %+v", rewards)
	}
}
