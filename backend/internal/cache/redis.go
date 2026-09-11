package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var Client *redis.Client
var Ctx = context.Background()

// PendingWriteTTL is the TTL for pending write cache entries.
// If the batch persister doesn't flush within this window, the entry is evicted
// to avoid stale data accumulating indefinitely.
const PendingWriteTTL = 5 * time.Minute

// Connect initializes the Redis connection using REDIS_URL
func Connect() error {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisUsername := os.Getenv("REDIS_USERNAME")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisSSL := os.Getenv("REDIS_SSL") == "true"
	redisTimeout := 10 * time.Second // Default timeout

	if timeoutStr := os.Getenv("REDIS_TIMEOUT"); timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr + "ms"); err == nil {
			redisTimeout = t
		}
	}

	if redisHost == "" || redisPort == "" {
		return fmt.Errorf("Redis is required: REDIS_HOST and REDIS_PORT must be configured")
	}

	options := &redis.Options{
		Addr:            redisHost + ":" + redisPort,
		Username:        redisUsername,
		Password:        redisPassword,
		DB:              0,
		DialTimeout:     redisTimeout,
		ReadTimeout:     redisTimeout,
		WriteTimeout:    redisTimeout,
		PoolSize:        50,
		MinIdleConns:    10,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 30 * time.Minute,
	}

	if redisSSL {
		options.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	Client = redis.NewClient(options)

	// Test the connection
	if err := Client.Ping(Ctx).Err(); err != nil {
		logger.S.Errorf("Failed to connect to Redis: %v", err)
		Client = nil // Set to nil to indicate Redis is not available
		return err
	}
	return nil
}

// SetJSON marshals a value to JSON and stores it in Redis with the specified expiration
func SetJSON(key string, value interface{}, expiration time.Duration) error {
	if Client == nil {
		return nil // Fail silently if Redis is not configured
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return Client.Set(Ctx, key, data, expiration).Err()
}

// GetJSON retrieves a value from Redis by key and unmarshals it into the provided dest interface pointer
func GetJSON(key string, dest interface{}) error {
	if Client == nil {
		return redis.Nil
	}

	val, err := Client.Get(Ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Delete removes one or more keys from Redis
func Delete(keys ...string) error {
	if Client == nil {
		return nil
	}
	return Client.Del(Ctx, keys...).Err()
}

// SetAndQueueJSON stores the JSON representation of a value in the cache and adds its ID to the pending writes set for the given entity type.
// A Lua script makes SET + SADD atomic; otherwise a failure between the two
// commands would leave an orphan cache entry that never gets persisted.
// A TTL (PendingWriteTTL) is applied so that orphaned entries are eventually evicted.
func SetAndQueueJSON(entityType, entityID string, value interface{}) error {
	if Client == nil {
		return nil
	}

	key := entityType + ":" + entityID
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return Client.Eval(Ctx, `
		redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[3])
		redis.call('SADD', KEYS[2], ARGV[2])
		return 1
	`, []string{key, "pending_db_writes:" + entityType}, data, entityID, PendingWriteTTL.Milliseconds()).Err()
}

// AcknowledgePendingJSON removes a pending marker only when the cached payload
// is still exactly the value that was persisted. If a producer has replaced the
// payload in the meantime, the marker is deliberately retained for the next flush.
func AcknowledgePendingJSON(entityType, entityID, expectedPayload string) (bool, error) {
	if Client == nil {
		return false, nil
	}
	result, err := Client.Eval(Ctx, `
		if redis.call('GET', KEYS[1]) ~= ARGV[1] then
			return 0
		end
		redis.call('SREM', KEYS[2], ARGV[2])
		redis.call('DEL', KEYS[1])
		return 1
	`, []string{entityType + ":" + entityID, "pending_db_writes:" + entityType}, expectedPayload, entityID).Int()
	return result == 1, err
}

// GetPendingIDs retrieves all entity IDs that have pending database writes for a given entity type.
func GetPendingIDs(entityType string) ([]string, error) {
	if Client == nil {
		return nil, redis.Nil
	}
	return Client.SMembers(Ctx, "pending_db_writes:"+entityType).Result()
}

// RemovePendingIDs removes the specified entity IDs from the pending writes set.
func RemovePendingIDs(entityType string, entityIDs ...string) error {
	if Client == nil || len(entityIDs) == 0 {
		return nil
	}

	// Convert strings to interface{} for SRem
	args := make([]interface{}, len(entityIDs))
	for i, id := range entityIDs {
		args[i] = id
	}

	return Client.SRem(Ctx, "pending_db_writes:"+entityType, args...).Err()
}

// PublishEvent publishes a JSON-marshaled payload to the specified Redis channel.
func PublishEvent(ctx context.Context, channel string, payload interface{}) error {
	if Client == nil {
		return nil
	}

	cb := GetCircuitBreaker()
	if !cb.AllowRequest() {
		logger.S.Warnf("Redis circuit breaker is %s, dropping publish to %s", cb.GetState(), channel)
		return ErrCircuitOpen
	}

	data, err := json.Marshal(payload)
	if err != nil {
		cb.RecordFailure()
		return err
	}

	err = Client.Publish(ctx, channel, data).Err()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = fmt.Errorf("redis circuit breaker is open")

// PublishEventQuick publishes an event with a short timeout (100ms) to prevent blocking.
// Use this when calling from handlers without a request-scoped context.
func PublishEventQuick(channel string, payload interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return PublishEvent(ctx, channel, payload)
}

// SubscribeChannels subscribes to one or more Redis Pub/Sub channels or patterns.
// Note: If using patterns, use PSubscribe in the caller or adjust this function.
func SubscribeChannels(ctx context.Context, channels ...string) *redis.PubSub {
	if Client == nil {
		return nil
	}
	return Client.Subscribe(ctx, channels...)
}

// GetManyJSON retrieves multiple values from Redis by keys (MGET).
func GetManyJSON(keys []string) ([]string, error) {
	if Client == nil {
		return nil, redis.Nil
	}
	if len(keys) == 0 {
		return []string{}, nil
	}

	results, err := Client.MGet(Ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	var jsonStrs []string
	for _, res := range results {
		if res != nil {
			if str, ok := res.(string); ok {
				jsonStrs = append(jsonStrs, str)
			}
		}
	}
	return jsonStrs, nil
}

// SetCombatState stores active combat state in Redis with 15 minutes TTL.
func SetCombatState(playerID string, state interface{}) error {
	if Client == nil {
		return nil
	}
	key := "combat_state:" + playerID
	return SetJSON(key, state, 15*time.Minute)
}

// GetCombatState retrieves active combat state from Redis.
func GetCombatState(playerID string, dest interface{}) error {
	if Client == nil {
		return redis.Nil
	}
	key := "combat_state:" + playerID
	return GetJSON(key, dest)
}

// ClearCombatState removes active combat state from Redis.
func ClearCombatState(playerID string) error {
	if Client == nil {
		return nil
	}
	key := "combat_state:" + playerID
	return Delete(key)
}

type TemporaryStatusData struct {
	ID        string    `json:"id"`
	Source    string    `json:"source,omitempty"`
	Value     float64   `json:"value,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SetPlayerTemporaryStatus stores a temporary status in Redis with an automatic TTL expiration.
func SetPlayerTemporaryStatus(playerID, statusID, source string, value float64, ttl time.Duration) error {
	if Client == nil {
		return nil
	}
	key := fmt.Sprintf("player_status:%s:%s", playerID, statusID)
	data := TemporaryStatusData{
		ID:        statusID,
		Source:    source,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
	return SetJSON(key, data, ttl)
}

// HasPlayerTemporaryStatus checks whether a temporary status is active for a player in Redis.
func HasPlayerTemporaryStatus(playerID, statusID string) bool {
	if Client == nil {
		return false
	}
	key := fmt.Sprintf("player_status:%s:%s", playerID, statusID)
	n, err := Client.Exists(Ctx, key).Result()
	return err == nil && n > 0
}

// RemovePlayerTemporaryStatus explicitly removes a temporary status from Redis.
func RemovePlayerTemporaryStatus(playerID, statusID string) error {
	if Client == nil {
		return nil
	}
	key := fmt.Sprintf("player_status:%s:%s", playerID, statusID)
	return Delete(key)
}

// GetPlayerTemporaryStatuses retrieves all active temporary statuses for a player from Redis.
func GetPlayerTemporaryStatuses(playerID string) ([]TemporaryStatusData, error) {
	if Client == nil {
		return nil, redis.Nil
	}
	pattern := fmt.Sprintf("player_status:%s:*", playerID)
	var cursor uint64
	var keys []string

	for {
		batch, nextCursor, err := Client.Scan(Ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return []TemporaryStatusData{}, nil
	}

	payloads, err := GetManyJSON(keys)
	if err != nil {
		return nil, err
	}

	statuses := make([]TemporaryStatusData, 0, len(payloads))
	for _, pStr := range payloads {
		var s TemporaryStatusData
		if err := json.Unmarshal([]byte(pStr), &s); err == nil {
			statuses = append(statuses, s)
		}
	}
	return statuses, nil
}
