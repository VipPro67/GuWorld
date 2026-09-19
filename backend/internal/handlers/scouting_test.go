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

func TestScoutingInspectPayloadValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	viewerID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("player_id", viewerID)
		c.Next()
	})

	router.POST("/api/scouting/inspect", InspectLocalPlayer)

	// Missing TargetID & TargetUsername
	body, _ := json.Marshal(models.InspectRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/scouting/inspect", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty target, got %d", w.Code)
	}
}

func TestDivinationPayloadValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	viewerID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("player_id", viewerID)
		c.Next()
	})

	router.POST("/api/scouting/divine", ExecuteDivination)

	// Missing TargetUsername
	body, _ := json.Marshal(models.DivinationRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/scouting/divine", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty username, got %d", w.Code)
	}
}

func TestDivineGuPayloadValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	viewerID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("player_id", viewerID)
		c.Next()
	})

	router.POST("/api/scouting/divine-gu", ExecuteDivineGu)

	// Missing GuQuery
	body, _ := json.Marshal(models.DivineGuRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/scouting/divine-gu", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty gu query, got %d", w.Code)
	}
}

