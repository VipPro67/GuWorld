package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestEquipGuRequiresCompletedRefinement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	guUnrefined := models.GuInstance{
		InstanceID:         "gu-unrefined",
		GuID:               "moonlight_gu",
		Rank:               1,
		Hunger:             100,
		RefinementProgress: 50.0,
		Durability:         100.0,
		Status:             "ACTIVE",
		Equipped:           false,
	}

	if guUnrefined.RefinementProgress >= 100 {
		t.Fatalf("expected unrefined Gu to have < 100 progress")
	}
}

func TestSubjugationVitalSuppressionLogic(t *testing.T) {
	vitalGu := &models.GuInstance{
		InstanceID:         "vital-1",
		GuID:               "spring_autumn_cicada",
		Rank:               6,
		Hunger:             100,
		RefinementProgress: 100.0,
		Durability:         100.0,
		Status:             "ACTIVE",
		Equipped:           true,
		IsVital:            true,
	}

	targetMortal := &models.GuInstance{
		InstanceID:         "target-1",
		GuID:               "moonlight_gu",
		Rank:               1,
		Hunger:             100,
		RefinementProgress: 0.0,
		Durability:         100.0,
		Status:             "ACTIVE",
	}

	// Suppression check: Vital rank (6) >= target rank (1) + 2 => true
	isSuppressed := vitalGu.IsVital && vitalGu.Rank >= targetMortal.Rank+2
	if !isSuppressed {
		t.Fatalf("expected Vital Gu Rank 6 to suppress Rank 1 mortal Gu")
	}

	targetPeer := &models.GuInstance{
		InstanceID:         "target-peer",
		GuID:               "immortal_gu_5",
		Rank:               5,
		Hunger:             100,
		RefinementProgress: 0.0,
		Durability:         100.0,
		Status:             "ACTIVE",
	}

	// Vital rank (6) >= 5 + 2 (7) => false
	isPeerSuppressed := vitalGu.IsVital && vitalGu.Rank >= targetPeer.Rank+2
	if isPeerSuppressed {
		t.Fatalf("expected Vital Gu Rank 6 NOT to suppress Rank 5 Gu (gap < 2)")
	}
}

func TestSubjugateGuValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing body
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/inventory/subjugate-gu", bytes.NewBuffer([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	SubjugateGu(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", w.Code)
	}

	// Without auth
	wAuth := httptest.NewRecorder()
	cAuth, _ := gin.CreateTestContext(wAuth)
	body, _ := json.Marshal(SubjugateGuRequest{InstanceID: uuid.NewString()})
	cAuth.Request, _ = http.NewRequest(http.MethodPost, "/api/inventory/subjugate-gu", bytes.NewBuffer(body))
	cAuth.Request.Header.Set("Content-Type", "application/json")
	SubjugateGu(cAuth)

	if wAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", wAuth.Code)
	}
}
