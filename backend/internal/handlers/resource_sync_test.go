package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRespondActionWithInventoryContainsAllVitalsAndProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	pID := uuid.New()
	var player models.Player
	player.ID = pID
	player.Rank = 3
	player.SubRank = 2
	player.WallProgress = 75.5
	player.CurrentMana = 150.0
	player.MaxMana = 200.0
	player.HP = 500.0
	player.BaseHP = 600.0
	player.AptitudePercent = 88
	player.PrimevalStones = 1200.0
	player.ImmortalStones = 15.0
	player.HumanQi = 50.0
	player.Lifespan = 80.0
	player.SoulHP = 90
	player.CurrentThought = 40
	player.StateRevision = 10

	respondActionWithInventory(c, http.StatusOK, player, gin.H{"custom_msg": "hello"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	requiredKeys := []string{
		"success",
		"gu_inventory",
		"materials_inventory",
		"primeval_stones",
		"immortal_stones",
		"human_qi",
		"hp",
		"base_hp",
		"current_mana",
		"max_mana",
		"current_thought",
		"soul_hp",
		"lifespan",
		"aptitude_percent",
		"rank",
		"sub_rank",
		"wall_progress",
		"state_revision",
		"custom_msg",
	}

	for _, key := range requiredKeys {
		if _, exists := resp[key]; !exists {
			t.Errorf("missing expected key %q in respondActionWithInventory payload", key)
		}
	}
}

func TestRespondActionGuOnlyContainsAllVitalsAndProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	pID := uuid.New()
	var player models.Player
	player.ID = pID
	player.Rank = 2
	player.SubRank = 1
	player.WallProgress = 20.0
	player.CurrentMana = 80.0
	player.MaxMana = 100.0
	player.HP = 300.0
	player.BaseHP = 350.0
	player.AptitudePercent = 65
	player.PrimevalStones = 300.0
	player.ImmortalStones = 0.0
	player.HumanQi = 10.0
	player.Lifespan = 99.0
	player.SoulHP = 100
	player.CurrentThought = 50
	player.StateRevision = 5

	respondActionGuOnly(c, http.StatusOK, player, gin.H{"extra_val": 42})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	requiredKeys := []string{
		"success",
		"gu_inventory",
		"primeval_stones",
		"immortal_stones",
		"human_qi",
		"hp",
		"base_hp",
		"current_mana",
		"max_mana",
		"current_thought",
		"soul_hp",
		"lifespan",
		"aptitude_percent",
		"rank",
		"sub_rank",
		"wall_progress",
		"state_revision",
		"extra_val",
	}

	for _, key := range requiredKeys {
		if _, exists := resp[key]; !exists {
			t.Errorf("missing expected key %q in respondActionGuOnly payload", key)
		}
	}
}
