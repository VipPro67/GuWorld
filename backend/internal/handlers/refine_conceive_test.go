package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConceiveRecipeRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid request with minimal fields", func(t *testing.T) {
		body := []byte(`{"target_gu_id":"GU_SWORD_1","target_rank":1}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/refine/conceive", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var reqData ConceiveRecipeRequest
		if err := c.ShouldBindJSON(&reqData); err != nil {
			t.Fatalf("expected valid bind, got: %v", err)
		}
		if reqData.TargetGuID != "GU_SWORD_1" || reqData.TargetRank != 1 {
			t.Errorf("unexpected parsed data: %+v", reqData)
		}
	})

	t.Run("missing target_gu_id", func(t *testing.T) {
		body := []byte(`{"target_rank":3}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/refine/conceive", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var reqData ConceiveRecipeRequest
		if err := c.ShouldBindJSON(&reqData); err == nil {
			t.Fatalf("expected bind error for missing target_gu_id, got nil")
		}
	})

	t.Run("invalid rank out of bounds", func(t *testing.T) {
		body := []byte(`{"target_gu_id":"GU_SWORD_1","target_rank":10}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/refine/conceive", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var reqData ConceiveRecipeRequest
		if err := c.ShouldBindJSON(&reqData); err == nil {
			t.Fatalf("expected bind error for rank 10 (max 9), got nil")
		}
	})
}

func TestConceiveRecipeResponseSerialization(t *testing.T) {
	res := ConceiveRecipeResponse{
		Status:           "DEVIATED",
		IsDeviated:       true,
		RecipeID:         "recipe_GU_SWORD_LIGHT_2_2_1",
		GuID:             "GU_SWORD_LIGHT_2",
		GuName:           "Kiếm Quang Cổ",
		Rank:             2,
		CompletionRate:   24.5,
		IsInstant100:     false,
		Message:          "Thiên cơ rẽ nhánh!",
		CatalystConsumed: "Kiếm Tinh Thạch",
		ThoughtCost:      50,
		StonesCost:       150,
		Backlash:         nil,
	}

	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if parsed["status"] != "DEVIATED" || parsed["is_deviated"] != true {
		t.Errorf("unexpected status in serialized JSON: %v", parsed)
	}
	if parsed["completion_rate"] != 24.5 {
		t.Errorf("unexpected completion_rate: %v", parsed["completion_rate"])
	}
}

func TestExtractPlayerPathPoints(t *testing.T) {
	masteries := map[string]int{
		"wisdom":     1200,
		"refinement": 500,
		"sword":      300,
	}

	if pts := extractPlayerPathPoints(masteries, "wisdom"); pts != 1200 {
		t.Errorf("expected 1200, got %d", pts)
	}
	if pts := extractPlayerPathPoints(masteries, "Wisdom Path"); pts != 1200 {
		t.Errorf("expected 1200 for 'Wisdom Path', got %d", pts)
	}
	if pts := extractPlayerPathPoints(masteries, "[\"Sword Path\"]"); pts != 300 {
		t.Errorf("expected 300 for JSON string, got %d", pts)
	}
	if pts := extractPlayerPathPoints(nil, "wisdom"); pts != 0 {
		t.Errorf("expected 0 for nil map, got %d", pts)
	}
	if pts := extractPlayerPathPoints(masteries, "fire"); pts != 0 {
		t.Errorf("expected 0 for missing path, got %d", pts)
	}
}
