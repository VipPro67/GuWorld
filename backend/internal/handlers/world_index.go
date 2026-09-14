package handlers

import (
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
	region string
	bx, by int
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

func (idx *positionIndex) bucketKey(region string, x, y int) worldBucketKey {
	bx := x / idx.bucket
	by := y / idx.bucket
	if x < 0 {
		bx--
	}
	if y < 0 {
		by--
	}
	return worldBucketKey{region: region, bx: bx, by: by}
}

// Set upserts an entry, moving it between buckets if its position changed.
func (idx *positionIndex) Set(e WorldPosEntry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	old, ok := idx.byID[e.ID]
	if ok && old != nil {
		oldKey := idx.bucketKey(old.Region, old.X, old.Y)
		if b := idx.grid[oldKey]; b != nil {
			delete(b, e.ID)
			if len(b) == 0 {
				delete(idx.grid, oldKey)
			}
		}
	}

	entry := e
	idx.byID[e.ID] = &entry
	key := idx.bucketKey(e.Region, e.X, e.Y)
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
// radius r around (x, y), excluding excludeID.
func (idx *positionIndex) Nearby(region string, x, y, r int, excludeID string) []WorldPosEntry {
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
	kmin := idx.bucketKey(region, minBX, minBY)
	kmax := idx.bucketKey(region, maxBX, maxBY)

	out := make([]WorldPosEntry, 0, 16)
	for bx := kmin.bx; bx <= kmax.bx; bx++ {
		for by := kmin.by; by <= kmax.by; by++ {
			for _, e := range idx.grid[worldBucketKey{region: region, bx: bx, by: by}] {
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
		key := idx.bucketKey(e.Region, e.X, e.Y)
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
