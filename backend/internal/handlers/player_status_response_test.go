package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestWriteTrimmedPlayerStatusPreservesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	googleID := "google-user"
	player := models.Player{
		Account: models.Account{
			ID:          uuid.New(),
			Username:    "player",
			Email:       "private@example.test",
			GoogleID:    &googleID,
			RefCode:     "private-ref",
			FakeRank:    8,
			FakeSubRank: 4,
			FakeVitalGu: "private-gu",
		},
		ResourceState: models.ResourceState{
			LastHungerTickAt: time.Now().UTC(),
		},
		BodyState: models.BodyState{
			HP:             42,
			BodyMods:       models.BodyModsList{"private-mod"},
			DaoMarkDamage:  1.5,
			ApertureDamage: 2.5,
		},
		MaterialsInventory: models.MaterialsInventory{"stone": 17},
	}

	// Build the expected contract using the previous trim behavior.
	fullJSON, err := json.Marshal(player)
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]json.RawMessage
	if err := json.Unmarshal(fullJSON, &expected); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"email", "google_id", "ref_code", "last_hunger_tick_at",
		"dao_mark_damage", "aperture_damage", "body_mods",
		"fake_rank", "fake_sub_rank", "fake_vital_gu",
	} {
		delete(expected, key)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	writeTrimmedPlayerStatus(ctx, player)

	var actual map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &actual); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("status response changed unexpectedly\nactual keys: %v\nexpected keys: %v", mapKeys(actual), mapKeys(expected))
	}
}

func mapKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
