package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func TestDictionaryEndpointsRejectEmptyRegistries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = logger.Init()

	prevGu := models.GuRegistry
	prevItems := models.ItemRegistry
	t.Cleanup(func() {
		models.GuRegistry = prevGu
		models.ItemRegistry = prevItems
		InvalidateDictionaryMemoryCache()
	})

	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{}
	models.ItemRegistry = map[string]models.ItemSpec{}
	InvalidateDictionaryMemoryCache()

	WarmDictionaryCache()
	dictCacheMu.RLock()
	_, warmed := dictCache["game_dictionary_unified"]
	dictCacheMu.RUnlock()
	if warmed {
		t.Fatal("WarmDictionaryCache should not cache empty registries")
	}

	for _, tc := range []struct {
		name    string
		handler gin.HandlerFunc
		key     string
		empty   string
	}{
		{"unified", GetDictionary, "game_dictionary_unified", `{"gu":[],"items":{}}`},
		{"gus", GetGuDictionary, "game_dictionary_gus", `[]`},
		{"items", GetItemDictionary, "game_dictionary_items", `{}`},
	} {
		t.Run(tc.name+"_empty", func(t *testing.T) {
			// Seed a poisoned memory cache entry to verify it is evicted rather than served.
			dictCacheMu.Lock()
			dictCache[tc.key] = dictCacheEntry{rawJSON: []byte(tc.empty), etag: `"poison"`}
			dictCacheMu.Unlock()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/dictionary", nil)

			tc.handler(c)

			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d (body=%s)", w.Code, http.StatusServiceUnavailable, w.Body.String())
			}
			dictCacheMu.RLock()
			_, stillCached := dictCache[tc.key]
			dictCacheMu.RUnlock()
			if stillCached {
				t.Fatalf("poisoned cache entry %q was not evicted", tc.key)
			}
		})
	}

	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_TEST": {1: {ID: "GU_TEST", Name: "Test Gu", Rank: 1}},
	}
	models.ItemRegistry = map[string]models.ItemSpec{
		"ITEM_TEST": {ID: "ITEM_TEST", Name: "Test Item", Rank: 1},
	}
	InvalidateDictionaryMemoryCache()
	WarmDictionaryCache()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/dictionary", nil)
	GetDictionary(c)
	if w.Code != http.StatusOK {
		t.Fatalf("GetDictionary status = %d, want 200", w.Code)
	}
	var resp struct {
		Gu    []models.GuDictionaryItem    `json:"gu"`
		Items map[string]models.ItemSpec   `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Gu) != 1 || len(resp.Items) != 1 {
		t.Fatalf("got gu=%d items=%d, want 1 and 1", len(resp.Gu), len(resp.Items))
	}
}
