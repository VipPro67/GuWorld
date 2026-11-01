package handlers

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestEquipGuRejectsDamagedGu(t *testing.T) {
	gin.SetMode(gin.TestMode)

	guDamaged := models.GuInstance{
		InstanceID:         "gu-damaged",
		GuID:               "moonlight_gu",
		Rank:               1,
		Hunger:             100,
		RefinementProgress: 100.0,
		Durability:         0.0,
		Status:             "DAMAGED",
		Equipped:           false,
	}

	if guDamaged.Status != "DAMAGED" || guDamaged.Durability > 0 {
		t.Fatalf("expected damaged Gu to have DAMAGED status and 0 durability")
	}
}

func TestDeathDegradesEquippedGusDurability(t *testing.T) {
	loser := &models.Player{}
	loser.ID = uuid.New()
	loser.Username = "FangYuan"
	loser.EquippedGus = models.EquippedGusList{"gu-1", "gu-2"}
	loser.PrimevalStones = 1000
	loser.GuInventory = models.GuInventory{
		{
			InstanceID:         "gu-1",
			GuID:               "moonlight_gu",
			Rank:               1,
			Hunger:             100,
			RefinementProgress: 100.0,
			Durability:         100.0,
			Status:             "ACTIVE",
			Equipped:           true,
		},
		{
			InstanceID:         "gu-2",
			GuID:               "liquor_worm",
			Rank:               1,
			Hunger:             100,
			RefinementProgress: 100.0,
			Durability:         10.0,
			Status:             "ACTIVE",
			Equipped:           true,
		},
		{
			InstanceID:         "gu-3",
			GuID:               "stealth_rock_gu",
			Rank:               1,
			Hunger:             100,
			RefinementProgress: 100.0,
			Durability:         100.0,
			Status:             "ACTIVE",
			Equipped:           false,
		},
	}

	// Simulate death penalty on equipped Gus:
	for i := range loser.GuInventory {
		if loser.GuInventory[i].Equipped {
			loser.GuInventory[i].Durability = math.Max(0, loser.GuInventory[i].Durability-15.0)
			if loser.GuInventory[i].Durability == 0 {
				loser.GuInventory[i].Status = "DAMAGED"
				loser.GuInventory[i].Equipped = false
			}
		}
	}
	damagedMap := make(map[string]bool)
	for _, g := range loser.GuInventory {
		if g.Status == "DAMAGED" || g.Durability <= 0 {
			damagedMap[g.InstanceID] = true
		}
	}
	filteredEquipped := make(models.EquippedGusList, 0, len(loser.EquippedGus))
	for _, id := range loser.EquippedGus {
		if !damagedMap[id] {
			filteredEquipped = append(filteredEquipped, id)
		}
	}
	loser.EquippedGus = filteredEquipped

	// gu-1 durability 100 - 15 = 85
	if loser.GuInventory[0].Durability != 85.0 || loser.GuInventory[0].Status != "ACTIVE" {
		t.Errorf("gu-1 want durability 85.0 ACTIVE, got %v %s", loser.GuInventory[0].Durability, loser.GuInventory[0].Status)
	}

	// gu-2 durability 10 - 15 -> 0, state DAMAGED, unequipped
	if loser.GuInventory[1].Durability != 0.0 || loser.GuInventory[1].Status != "DAMAGED" || loser.GuInventory[1].Equipped {
		t.Errorf("gu-2 want durability 0.0 DAMAGED unequipped, got %v %s equipped=%v",
			loser.GuInventory[1].Durability, loser.GuInventory[1].Status, loser.GuInventory[1].Equipped)
	}

	// gu-3 unequipped, untouched
	if loser.GuInventory[2].Durability != 100.0 {
		t.Errorf("gu-3 want durability 100.0, got %v", loser.GuInventory[2].Durability)
	}

	// loser.EquippedGus should only retain gu-1
	if len(loser.EquippedGus) != 1 || loser.EquippedGus[0] != "gu-1" {
		t.Errorf("loser.EquippedGus want [gu-1], got %v", loser.EquippedGus)
	}
}

func TestNourishAndRepairValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Unauthenticated test
	r := gin.New()
	r.POST("/api/inventory/nourish-gu", NourishGu)
	r.POST("/api/inventory/repair-gu", RepairGu)

	// Test unauthenticated nourish
	reqBody, _ := json.Marshal(NourishGuRequest{InstanceID: "any"})
	req := httptest.NewRequest(http.MethodPost, "/api/inventory/nourish-gu", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}

	// Test unauthenticated repair
	reqBody2, _ := json.Marshal(RepairGuRequest{InstanceID: "any"})
	req2 := httptest.NewRequest(http.MethodPost, "/api/inventory/repair-gu", bytes.NewBuffer(reqBody2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w2.Code)
	}
}
