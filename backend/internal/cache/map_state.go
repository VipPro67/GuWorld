package cache

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type MapCellState struct {
	ClearedBy  string        `json:"cleared_by"`
	ClearedAt  time.Time     `json:"cleared_at"`
	MonsterID  string        `json:"monster_id,omitempty"`
	Rank       int           `json:"rank,omitempty"`
	SubRank    int           `json:"sub_rank,omitempty"`
	RespawnTTL time.Duration `json:"respawn_ttl,omitempty"`
}

type MapResourceState struct {
	ItemID    string    `json:"item_id"`
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

func instanceCellKey(regionID, instanceID string, x, y int) string {
	return fmt.Sprintf("map:cell:%s:instance:%s:%d:%d", regionID, instanceID, x, y)
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

// SetInstanceCellCleared isolates temporary cell state inside a map instance.
func SetInstanceCellCleared(regionID, instanceID string, x, y int, playerID string, ttl time.Duration) error {
	data := MapCellState{ClearedBy: playerID, ClearedAt: time.Now(), RespawnTTL: ttl}
	return SetJSON(instanceCellKey(regionID, instanceID, x, y), data, ttl)
}

// IsInstanceCellCleared reads temporary cell state scoped to one map instance.
func IsInstanceCellCleared(regionID, instanceID string, x, y int) (bool, string, error) {
	var state MapCellState
	err := GetJSON(instanceCellKey(regionID, instanceID, x, y), &state)
	if err == redis.Nil {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, state.ClearedBy, nil
}

// GetInstanceClearedCells checks coordinates against one instance's state.
func GetInstanceClearedCells(regionID, instanceID string, coords []CellCoordKey) (map[CellCoordKey]MapCellState, error) {
	out := make(map[CellCoordKey]MapCellState)
	if len(coords) == 0 || Client == nil {
		return out, nil
	}
	keys := make([]string, len(coords))
	for i, coord := range coords {
		keys[i] = instanceCellKey(regionID, instanceID, coord.X, coord.Y)
	}
	values, err := Client.MGet(Ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	for i, value := range values {
		raw, ok := value.(string)
		if !ok {
			continue
		}
		var state MapCellState
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			return nil, fmt.Errorf("decode cleared instance cell %s: %w", keys[i], err)
		}
		out[coords[i]] = state
	}
	return out, nil
}

// GetClearedCells checks a set of cells with one Redis round trip. The result
// only contains coordinates that currently have a cleared-state key.
func GetClearedCells(coords []CellCoordKey) (map[CellCoordKey]MapCellState, error) {
	out := make(map[CellCoordKey]MapCellState)
	if len(coords) == 0 || Client == nil {
		return out, nil
	}
	keys := make([]string, len(coords))
	for i, coord := range coords {
		keys[i] = cellKey(coord.RegionID, coord.X, coord.Y)
	}
	values, err := Client.MGet(Ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	for i, value := range values {
		raw, ok := value.(string)
		if !ok {
			continue
		}
		var state MapCellState
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			return nil, fmt.Errorf("decode cleared cell %s: %w", keys[i], err)
		}
		out[coords[i]] = state
	}
	return out, nil
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
