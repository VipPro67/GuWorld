package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type RateLimitStore interface {
	Allow(key string, limit int, window time.Duration) (bool, error)
}

// ── Redis Store ──────────────────────────────────────────────────────

type RedisStore struct{}

func NewRedisStore() *RedisStore {
	return &RedisStore{}
}

func (s *RedisStore) Allow(key string, limit int, window time.Duration) (bool, error) {
	rdb := cache.Client
	if rdb == nil {
		return false, fmt.Errorf("redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	windowStart := now.Add(-window)
	rKey := fmt.Sprintf("rl:%s:%d", key, now.Unix()/int64(window.Seconds()))

	pipe := rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, rKey, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	pipe.ZCard(ctx, rKey)
	pipe.ZAdd(ctx, rKey, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
	pipe.Expire(ctx, rKey, window+time.Second)

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count := cmds[1].(*redis.IntCmd).Val()
	return count < int64(limit), nil
}

// ── Memory Store ─────────────────────────────────────────────────────

type MemoryStore struct {
	mu      sync.Mutex
	entries map[string][]time.Time
}

func NewMemoryStore() *MemoryStore {
	ms := &MemoryStore{entries: make(map[string][]time.Time)}
	go ms.cleanup()
	return ms
}

func (s *MemoryStore) Allow(key string, limit int, window time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	active := s.entries[key][:0]
	for _, t := range s.entries[key] {
		if now.Sub(t) < window {
			active = append(active, t)
		}
	}

	if len(active) >= limit {
		s.entries[key] = active
		return false, nil
	}

	s.entries[key] = append(active, now)
	return true, nil
}

func (s *MemoryStore) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		s.mu.Lock()
		now := time.Now()
		for k, ts := range s.entries {
			var active []time.Time
			for _, t := range ts {
				if now.Sub(t) < time.Hour {
					active = append(active, t)
				}
			}
			if len(active) == 0 {
				delete(s.entries, k)
			} else {
				s.entries[k] = active
			}
		}
		s.mu.Unlock()
	}
}

// ── Fallback Store (Redis primary → Memory) ──────────────────────────

type FallbackStore struct {
	redis    *RedisStore
	memory   *MemoryStore
	useRedis bool
	mu       sync.RWMutex
}

func NewFallbackStore() *FallbackStore {
	fs := &FallbackStore{
		redis:    NewRedisStore(),
		memory:   NewMemoryStore(),
		useRedis: cache.Client != nil,
	}
	if fs.useRedis {
		logger.S.Info("Rate limiter using Redis backend")
	} else {
		logger.S.Info("Rate limiter using in-memory backend (Redis not available)")
	}
	return fs
}

func (s *FallbackStore) Allow(key string, limit int, window time.Duration) (bool, error) {
	s.mu.RLock()
	usingRedis := s.useRedis
	s.mu.RUnlock()

	if usingRedis {
		ok, err := s.redis.Allow(key, limit, window)
		if err != nil {
			logger.S.Warnf("Redis rate limit failed, falling back to memory: %v", err)
			s.mu.Lock()
			s.useRedis = false
			s.mu.Unlock()
			return s.memory.Allow(key, limit, window)
		}
		return ok, nil
	}

	return s.memory.Allow(key, limit, window)
}

func (s *FallbackStore) StartRedisWatchdog(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if cache.Client != nil {
				s.mu.Lock()
				if !s.useRedis {
					logger.S.Info("Redis recovered, switching back to Redis rate limiter")
					s.useRedis = true
				}
				s.mu.Unlock()
			}
		}
	}()
}
