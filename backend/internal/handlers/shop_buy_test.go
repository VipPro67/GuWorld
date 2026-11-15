package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

func TestBuyShopItem_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	reqBody := BuyShopRequest{
		PlayerID: uuid.New().String(),
		ItemID:   "GU_RAIN_GATE_5",
		Quantity: 1,
		Rank:     5,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/api/shop/buy", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	BuyShopItem(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
	}
}

func TestBuyShopItem_ApertureMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	authID := uuid.New()
	c.Set("player_id", authID)

	otherID := uuid.New()
	reqBody := BuyShopRequest{
		PlayerID: otherID.String(),
		ItemID:   "GU_RAIN_GATE_5",
		Quantity: 1,
		Rank:     5,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/api/shop/buy", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	BuyShopItem(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d", w.Code)
	}
}

func TestBuyShopItem_ApertureCapacityErrorClassification(t *testing.T) {
	// Verify that aperture capacity error is correctly identified as a 400 Bad Request
	capErr := &db.ApertureCapacityError{Current: 80, Max: 80}
	if !db.IsApertureCapacity(capErr) {
		t.Fatalf("expected ApertureCapacityError to be recognized by IsApertureCapacity")
	}

	legacyErr := db.ErrApertureCapacity
	if !db.IsApertureCapacity(legacyErr) {
		t.Fatalf("expected ErrApertureCapacity to be recognized by IsApertureCapacity")
	}
}
