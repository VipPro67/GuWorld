package handlers

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// WorldPosIndexBucket is the side length (in world cells) of each spatial
// bucket in the position index. It matches the radius the map client polls and
// keeps Nearby() to a handful of buckets per lookup.
const WorldPosIndexBucket = 300

// WorldPosInterestRadius is how far (in world cells) a moving player's update
// is fanned out over SSE. It is a superset of the client's default viewport;
// clients still apply their own window filter, so this only bounds server-side
// fan-out (no global O(N^2) broadcast).
const WorldPosInterestRadius = 300

// WorldPosActiveWindow mirrors the "active player" semantics used by the SQL
// queries (last_mana_update within the window). The index TS field is the
// activity timestamp, so entries idle longer than this are dropped from
// Nearby() results.
const WorldPosActiveWindow = 15 * time.Minute

// WorldPosEntry is a player's current position + identity, kept in memory.
type WorldPosEntry struct {
	ID        string
	Username  string
	AvatarURL string
	X         int
	Y         int
	Region    string
	TS        int64 // activity timestamp, unix milliseconds
}

type worldBucketKey struct {
	bx, by int
}

// ActiveWorldBucket is a bounded snapshot used by shared world workers. X/Y
// identify a representative active player; Region prevents workers from
// ticking entities merely because another region overlaps in coordinates.
type ActiveWorldBucket struct {
	BX, BY int
	X, Y   int
	Region string
}

// positionIndex is the in-memory spatial index of active players. It is the
// fast path for nearby-player queries and the interest source for SSE fan-out.
// The database remains the durable source of truth; a periodic reconcile
// (StartWorldIndexSyncWorker) reloads active players so missed call sites or
// stale entries self-heal within one sync interval.
type positionIndex struct {
	mu     sync.RWMutex
	bucket int
	byID   map[string]*WorldPosEntry
	grid   map[worldBucketKey]map[string]*WorldPosEntry
}

var worldPosIndex = newPositionIndex(WorldPosIndexBucket)

func newPositionIndex(bucket int) *positionIndex {
	return &positionIndex{
		bucket: bucket,
		byID:   make(map[string]*WorldPosEntry),
		grid:   make(map[worldBucketKey]map[string]*WorldPosEntry),
	}
}

func (idx *positionIndex) bucketKey(x, y int) worldBucketKey {
	bx := x / idx.bucket
	by := y / idx.bucket
	if x < 0 {
		bx--
	}
	if y < 0 {
		by--
	}
	return worldBucketKey{bx: bx, by: by}
}

// Set upserts an entry, moving it between buckets if its position changed.
func (idx *positionIndex) Set(e WorldPosEntry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	old, ok := idx.byID[e.ID]
	if ok && old != nil {
		oldKey := idx.bucketKey(old.X, old.Y)
		if b := idx.grid[oldKey]; b != nil {
			delete(b, e.ID)
			if len(b) == 0 {
				delete(idx.grid, oldKey)
			}
		}
	}

	entry := e
	idx.byID[e.ID] = &entry
	key := idx.bucketKey(e.X, e.Y)
	b := idx.grid[key]
	if b == nil {
		b = make(map[string]*WorldPosEntry)
		idx.grid[key] = b
	}
	b[e.ID] = &entry
}

// Get returns the current entry for a player id.
func (idx *positionIndex) Get(id string) (WorldPosEntry, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	e, ok := idx.byID[id]
	if !ok || e == nil {
		return WorldPosEntry{}, false
	}
	return *e, true
}

// Nearby returns active entries whose position is within a Chebyshev box of
// radius r around (x, y), excluding excludeID. Proximity is based purely on
// world coordinates and is not split or masked by region boundaries.
func (idx *positionIndex) Nearby(x, y, r int, excludeID string) []WorldPosEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if r < 0 {
		r = 0
	}
	now := time.Now().UnixMilli()
	windowMillis := WorldPosActiveWindow.Milliseconds()

	minBX := x - r
	minBY := y - r
	maxBX := x + r
	maxBY := y + r
	kmin := idx.bucketKey(minBX, minBY)
	kmax := idx.bucketKey(maxBX, maxBY)

	out := make([]WorldPosEntry, 0, 16)
	for bx := kmin.bx; bx <= kmax.bx; bx++ {
		for by := kmin.by; by <= kmax.by; by++ {
			for _, e := range idx.grid[worldBucketKey{bx: bx, by: by}] {
				if e == nil || e.ID == excludeID {
					continue
				}
				if now-e.TS > windowMillis {
					continue
				}
				if e.X < minBX || e.X > maxBX || e.Y < minBY || e.Y > maxBY {
					continue
				}
				out = append(out, *e)
			}
		}
	}
	return out
}

// Reconcile replaces the entire index with the provided active-player snapshot.
// Used by the sync worker to bootstrap and self-heal.
func (idx *positionIndex) Reconcile(entries []WorldPosEntry) {
	byID := make(map[string]*WorldPosEntry, len(entries))
	grid := make(map[worldBucketKey]map[string]*WorldPosEntry)

	for i := range entries {
		e := entries[i]
		byID[e.ID] = &e
		key := idx.bucketKey(e.X, e.Y)
		b := grid[key]
		if b == nil {
			b = make(map[string]*WorldPosEntry)
			grid[key] = b
		}
		b[e.ID] = &e
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.byID = byID
	idx.grid = grid
}

// Len returns the number of entries currently tracked.
func (idx *positionIndex) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.byID)
}

// ActiveBuckets returns at most limit occupied buckets. It intentionally
// copies only scalar data while holding the read lock, so a fixed-tick worker
// never keeps the position index locked during Redis or database work.
func (idx *positionIndex) ActiveBuckets(limit int) []ActiveWorldBucket {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if limit <= 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	out := make([]ActiveWorldBucket, 0, limit)
	seen := make(map[string]struct{}, limit)
	for key, entries := range idx.grid {
		for _, e := range entries {
			if e == nil || now-e.TS > WorldPosActiveWindow.Milliseconds() {
				continue
			}
			identity := e.Region + ":" + strconv.Itoa(key.bx) + ":" + strconv.Itoa(key.by)
			if _, ok := seen[identity]; ok {
				continue
			}
			seen[identity] = struct{}{}
			out = append(out, ActiveWorldBucket{BX: key.bx, BY: key.by, X: e.X, Y: e.Y, Region: e.Region})
			if len(out) == limit {
				return out
			}
		}
	}
	return out
}

// ActiveEntries returns a bounded snapshot of currently active players. World
// runtimes with a different spatial granularity (for example monster AOI) use
// this instead of coupling their bucket size to the position broadcast index.
func (idx *positionIndex) ActiveEntries(limit int) []WorldPosEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if limit == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	capacity := len(idx.byID)
	if limit > 0 && limit < capacity {
		capacity = limit
	}
	out := make([]WorldPosEntry, 0, capacity)
	for _, e := range idx.byID {
		if e == nil || now-e.TS > WorldPosActiveWindow.Milliseconds() {
			continue
		}
		out = append(out, *e)
		if limit > 0 && len(out) == limit {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// OnlineEntries returns active position entries that also satisfy the live
// connection predicate. Filtering happens after ActiveEntries releases the
// index lock so connection providers cannot introduce lock-order inversions.
func (idx *positionIndex) OnlineEntries(limit int, connected func(WorldPosEntry) bool) []WorldPosEntry {
	if connected == nil {
		return nil
	}
	active := idx.ActiveEntries(-1)
	capacity := len(active)
	if limit > 0 && limit < capacity {
		capacity = limit
	}
	online := make([]WorldPosEntry, 0, capacity)
	for _, entry := range active {
		if !connected(entry) {
			continue
		}
		online = append(online, entry)
		if limit > 0 && len(online) == limit {
			break
		}
	}
	return online
}

// setWorldPosFromPlayer upserts a player's current position into the index.
// Called after any position-mutating commit (retreats, teleports, onboarding)
// that does not go through the move handlers (which already broadcast).
func setWorldPosFromPlayer(p *models.Player) {
	worldPosIndex.Set(WorldPosEntry{
		ID:        p.ID.String(),
		Username:  p.Username,
		AvatarURL: p.AvatarURL,
		X:         p.X,
		Y:         p.Y,
		Region:    p.CurrentRegion,
		TS:        time.Now().UnixMilli(),
	})
}
