package handlers

import (
	"container/list"
	"sync"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// cellCacheMaxEntries bounds the procedural cell cache. Procedural cells are a
// pure function of (x, y), so repeated rect/grid requests over the same area
// (many clients sharing a hub or route) hit the cache instead of recomputing
// the SHA-256 seed + zone lookup + weighted pick for every cell.
const cellCacheMaxEntries = 100_000

type cellCacheKey struct{ x, y int }

type cellCacheEntry struct {
	key  cellCacheKey
	cell models.Cell
}

type proceduralCellCache struct {
	mu      sync.Mutex
	cap     int
	entries map[cellCacheKey]*list.Element
	lru     *list.List
}

var cellGenCache = &proceduralCellCache{
	cap:     cellCacheMaxEntries,
	entries: make(map[cellCacheKey]*list.Element),
	lru:     list.New(),
}

func (c *proceduralCellCache) get(x, y int) (models.Cell, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := cellCacheKey{x, y}
	if el, ok := c.entries[k]; ok {
		c.lru.MoveToFront(el)
		return el.Value.(*cellCacheEntry).cell, true
	}
	return models.Cell{}, false
}

func (c *proceduralCellCache) put(x, y int, cell models.Cell) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := cellCacheKey{x, y}
	if el, ok := c.entries[k]; ok {
		el.Value.(*cellCacheEntry).cell = cell
		c.lru.MoveToFront(el)
		return
	}
	el := c.lru.PushFront(&cellCacheEntry{key: k, cell: cell})
	c.entries[k] = el
	if c.lru.Len() > c.cap {
		if last := c.lru.Back(); last != nil {
			c.lru.Remove(last)
			delete(c.entries, last.Value.(*cellCacheEntry).key)
		}
	}
}

// clear drops all cached cells. Called when the zone registry is (re)loaded,
// since procedural content depends on the zone layout.
func (c *proceduralCellCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[cellCacheKey]*list.Element)
	c.lru.Init()
}
