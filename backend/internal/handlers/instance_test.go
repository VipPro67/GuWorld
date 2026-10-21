package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMaxDailyInstanceEntriesConstant(t *testing.T) {
	if MaxDailyInstanceEntries != 5 {
		t.Fatalf("MaxDailyInstanceEntries = %d, expected 5", MaxDailyInstanceEntries)
	}
}

func TestInstanceCacheGetters(t *testing.T) {
	instanceCacheMutex.Lock()
	instanceTemplatesCache = []InstanceListItem{
		{
			ID:           "test_dream",
			Name:         "Test Dream",
			Type:         "DREAM_REALM",
			RequiredRank: 1,
		},
	}
	instanceTemplatesMap = map[string]*InstanceTemplate{
		"test_dream": {
			ID:            "test_dream",
			Name:          "Test Dream",
			Type:          "DREAM_REALM",
			RequiredRank:  1,
			EntryItem:     "NONE",
			Description:   "Test Description",
			Rewards:       json.RawMessage("{}"),
			MapData:       json.RawMessage("[]"),
			CooldownHours: 24,
		},
	}
	instanceCacheMutex.Unlock()

	list := getCachedInstanceList()
	if len(list) != 1 || list[0].ID != "test_dream" {
		t.Fatalf("getCachedInstanceList() failed, got %+v", list)
	}

	tmpl, found := getCachedInstanceTemplate("test_dream")
	if !found || tmpl.Name != "Test Dream" {
		t.Fatalf("getCachedInstanceTemplate('test_dream') failed, got found=%v, tmpl=%+v", found, tmpl)
	}

	_, found = getCachedInstanceTemplate("non_existent")
	if found {
		t.Fatalf("getCachedInstanceTemplate('non_existent') returned true, expected false")
	}
}

// TestEnterDreamRealmDoesNotReReadConsumedBody guards against the regression
// where EnterDreamRealm parsed the JSON body and then delegated to
// EnterInstance, which tried to decode the same (already drained) body and
// answered "instance_id is required".
func TestEnterDreamRealmDoesNotReReadConsumedBody(t *testing.T) {
	instanceCacheMutex.Lock()
	instanceTemplatesCache = []InstanceListItem{
		{ID: "test_dream", Name: "Test Dream", Type: "DREAM_REALM", RequiredRank: 1},
	}
	instanceTemplatesMap = map[string]*InstanceTemplate{
		"test_dream": {
			ID:            "test_dream",
			Name:          "Test Dream",
			Type:          "DREAM_REALM",
			RequiredRank:  1,
			EntryItem:     "NONE",
			Description:   "Test Description",
			Rewards:       json.RawMessage("{}"),
			MapData:       json.RawMessage("[]"),
			CooldownHours: 24,
		},
	}
	instanceCacheMutex.Unlock()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/dream-realms/enter", strings.NewReader(`{"instance_id":"test_dream"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	EnterDreamRealm(c)

	// No player in the gin context, so the shared flow must stop at the auth
	// guard (401). A 400 here means the delegated request was not reusable.
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("EnterDreamRealm returned %d (%s), expected %d (auth guard reached)",
			recorder.Code, recorder.Body.String(), http.StatusUnauthorized)
	}
}
