package handlers

import (
	"encoding/json"
	"testing"
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
