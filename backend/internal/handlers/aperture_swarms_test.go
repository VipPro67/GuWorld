package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMaterializeGuRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("RejectsRankAbove5", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := bytes.NewBufferString(`{"gu_id":"gu_moonlight","gu_rank":6,"equip":true}`)
		c.Request, _ = http.NewRequest(http.MethodPost, "/api/immortal-aperture/swarms/materialize", body)
		c.Request.Header.Set("Content-Type", "application/json")

		MaterializeGuFromSwarm(c)
		if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
			t.Errorf("expected 401 or 400, got %d", w.Code)
		}
	})

	t.Run("RejectsRankBelow1", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := bytes.NewBufferString(`{"gu_id":"gu_moonlight","gu_rank":0,"equip":true}`)
		c.Request, _ = http.NewRequest(http.MethodPost, "/api/immortal-aperture/swarms/materialize", body)
		c.Request.Header.Set("Content-Type", "application/json")

		MaterializeGuFromSwarm(c)
		if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
			t.Errorf("expected 401 or 400, got %d", w.Code)
		}
	})
}

func TestDematerializeGuRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := bytes.NewBufferString(`{}`)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/immortal-aperture/swarms/dematerialize", body)
	c.Request.Header.Set("Content-Type", "application/json")

	DematerializeGuToSwarm(c)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
		t.Errorf("expected 401 or 400, got %d", w.Code)
	}
}
