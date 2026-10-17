package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRegionsCacheRebuildsAnEmptyCachedList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const key = "test_explore_regions_v2_empty"
	previousClient := cache.Client
	cache.Client = nil
	t.Cleanup(func() { cache.Client = previousClient })

	dictCacheMu.Lock()
	previousEntry, hadPreviousEntry := dictCache[key]
	dictCache[key] = dictCacheEntry{rawJSON: []byte("[]"), etag: `"empty"`}
	dictCacheMu.Unlock()
	t.Cleanup(func() {
		dictCacheMu.Lock()
		if hadPreviousEntry {
			dictCache[key] = previousEntry
		} else {
			delete(dictCache, key)
		}
		dictCacheMu.Unlock()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/explore/regions", nil)
	buildCalls := 0
	getCachedDictionaryPartWithError(ctx, key, &[]models.Region{}, func() (interface{}, error) {
		buildCalls++
		return []models.Region{{ID: "test_region", Name: "Test Region"}}, nil
	}, func(value interface{}) bool {
		switch regions := value.(type) {
		case *[]models.Region:
			return len(*regions) > 0
		case []models.Region:
			return len(regions) > 0
		default:
			return false
		}
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if buildCalls != 1 {
		t.Fatalf("build called %d times, want 1 for an empty cached list", buildCalls)
	}
	if !strings.Contains(recorder.Body.String(), "test_region") {
		t.Fatalf("response = %s, want rebuilt region list", recorder.Body.String())
	}
}

func TestRegionsCacheDoesNotCacheBuildErrorsAsEmptyLists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const key = "test_explore_regions_v2_build_error"
	previousClient := cache.Client
	cache.Client = nil
	t.Cleanup(func() { cache.Client = previousClient })

	dictCacheMu.Lock()
	previousEntry, hadPreviousEntry := dictCache[key]
	delete(dictCache, key)
	dictCacheMu.Unlock()
	t.Cleanup(func() {
		dictCacheMu.Lock()
		if hadPreviousEntry {
			dictCache[key] = previousEntry
		} else {
			delete(dictCache, key)
		}
		dictCacheMu.Unlock()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/explore/regions", nil)
	getCachedDictionaryPartWithError(ctx, key, &[]models.Region{}, func() (interface{}, error) {
		return nil, errors.New("database unavailable")
	}, func(value interface{}) bool {
		switch regions := value.(type) {
		case *[]models.Region:
			return len(*regions) > 0
		case []models.Region:
			return len(regions) > 0
		default:
			return false
		}
	})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}
	dictCacheMu.RLock()
	_, cached := dictCache[key]
	dictCacheMu.RUnlock()
	if cached {
		t.Fatal("failed region build was cached")
	}
}

func TestGetRegionDetailsEmptyParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/explore/regions//details", nil)
	GetRegionDetails(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
