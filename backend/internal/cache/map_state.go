package cache

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type MapCellState struct {
	ClearedBy   string    `json:"cleared_by"`
	ClearedAt   time.Time `json:"cleared_at"`
	MonsterID   string    `json:"monster_id,omitempty"`
	Rank        int       `json:"rank,omitempty"`
	SubRank     int       `json:"sub_rank,omitempty"`
	RespawnTTL  time.Duration `json:"respawn_ttl,omitempty"`
}

type MapResourceState struct {
	ItemID  string    `json:"item_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CellCoordKey struct {
	RegionID string
	X        int
	Y        int
}

func cellKey(regionID string, x, y int) string {
	return fmt.Sprintf("map:cell:%s:%d:%d", regionID, x, y)
}

func resourceKey(regionID string, x, y int) string {
	return fmt.Sprintf("map:resource:%s:%d:%d", regionID, x, y)
}

// SetCellCleared marks a monster/resource cell as cleared by a player.
// The key persists with a TTL equal to the respawn time. When the TTL expires,
// the cell automatically becomes available again.
func SetCellCleared(regionID string, x, y int, playerID string, ttl time.Duration) error {
	key := cellKey(regionID, x, y)
	data := MapCellState{
		ClearedBy:  playerID,
		ClearedAt:  time.Now(),
		RespawnTTL: ttl,
	}
	return SetJSON(key, data, ttl)
}

// IsCellCleared checks if a cell has been cleared and returns the cleared-by player info.
// Returns (cleared, clearedBy, err). redis.Nil means the key doesn't exist → cell is NOT cleared.
func IsCellCleared(regionID string, x, y int) (bool, string, error) {
	key := cellKey(regionID, x, y)
	var state MapCellState
	err := GetJSON(key, &state)
	if err == redis.Nil {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, state.ClearedBy, nil
}

// RemoveCellCleared removes the cleared state from a cell (e.g., for forced respawn).
func RemoveCellCleared(regionID string, x, y int) error {
	key := cellKey(regionID, x, y)
	return Delete(key)
}

// SetResourceCell marks a resource cell as picked up.
// The key persists with a TTL equal to the respawn time.
func SetResourceCell(regionID string, x, y int, itemID string, ttl time.Duration) error {
	key := resourceKey(regionID, x, y)
	data := MapResourceState{
		ItemID:    itemID,
		ExpiresAt: time.Now().Add(ttl),
	}
	return SetJSON(key, data, ttl)
}

// GetResourceCell checks if a resource cell has been picked up.
// Returns (hasResource, itemID, err). redis.Nil means the key doesn't exist → resource is available.
func GetResourceCell(regionID string, x, y int) (bool, string, error) {
	key := resourceKey(regionID, x, y)
	var state MapResourceState
	err := GetJSON(key, &state)
	if err == redis.Nil {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, state.ItemID, nil
}

// RemoveResourceCell removes the resource state from a cell.
func RemoveResourceCell(regionID string, x, y int) error {
	key := resourceKey(regionID, x, y)
	return Delete(key)
}

// GetWorldCellEventPayload creates a WorldCellEvent payload for Redis broadcast.
func GetWorldCellEventPayload(eventType string, regionID string, x, y int, cellType, contentID string) map[string]any {
	payload := map[string]any{
		"type":       eventType,
		"region_id":  regionID,
		"x":          x,
		"y":          y,
		"cell_type":  cellType,
	}
	if contentID != "" {
		payload["content_id"] = contentID
	}
	return payload
}

// BroadcastCellCleared publishes a cell_cleared event via Redis pub/sub.
func BroadcastCellCleared(regionID string, x, y int, cellType, contentID string) error {
	payload := GetWorldCellEventPayload("cell_cleared", regionID, x, y, cellType, contentID)
	return PublishEventQuick("world:cell", payload)
}

// BroadcastCellRespawn publishes a cell_respawned event via Redis pub/sub.
func BroadcastCellRespawn(regionID string, x, y int, cellType, contentID string) error {
	payload := GetWorldCellEventPayload("cell_respawned", regionID, x, y, cellType, contentID)
	return PublishEventQuick("world:cell", payload)
}

// GetResourceCellEventPayload creates a resource pickup event payload.
func GetResourceCellEventPayload(x, y int, itemName string) map[string]any {
	return map[string]any{
		"type":      "resource_pickedup",
		"x":         x,
		"y":         y,
		"item_name": itemName,
	}
}

// BroadcastResourcePickedUp publishes a resource_pickedup event via Redis pub/sub.
func BroadcastResourcePickedUp(x, y int, itemName string) error {
	payload := GetResourceCellEventPayload(x, y, itemName)
	return PublishEventQuick("world:cell", payload)
}

// BroadcastMonsterKilled publishes a monster_killed event via Redis pub/sub.
func BroadcastMonsterKilled(x, y int, monsterName, regionID string) error {
	payload := map[string]any{
		"type":       "monster_killed",
		"x":          x,
		"y":          y,
		"monster_name": monsterName,
		"region_id":  regionID,
	}
	return PublishEventQuick("world:cell", payload)
}

// BroadcastWildGuCaptured publishes a wild_gu_captured event via Redis pub/sub.
func BroadcastWildGuCaptured(x, y int, guID string) error {
	payload := map[string]any{
		"type":       "wild_gu_captured",
		"x":          x,
		"y":          y,
		"gu_id":      guID,
	}
	return PublishEventQuick("world:cell", payload)
}

// BroadcastMineMonsterKilled publishes a mine monster killed event via Redis pub/sub.
func BroadcastMineMonsterKilled(x, y int, mineType, rarity string) error {
	payload := map[string]any{
		"type":       "mine_monster_killed",
		"x":          x,
		"y":          y,
		"mine_type":  mineType,
		"rarity":     rarity,
	}
	return PublishEventQuick("world:cell", payload)
}