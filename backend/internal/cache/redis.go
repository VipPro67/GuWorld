package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var Client *redis.Client
var Ctx = context.Background()

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
		logger.S.Info("Redis host or port not configured, skipping Redis connection")
		return nil
	}

	options := &redis.Options{
		Addr:         redisHost + ":" + redisPort,
		Username:     redisUsername,
		Password:     redisPassword,
		DB:           0,
		DialTimeout:  redisTimeout,
		ReadTimeout:  redisTimeout,
		WriteTimeout: redisTimeout,
	}

	if redisSSL {
		options.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	Client = redis.NewClient(options)

	// Test the connection
	if err := Client.Ping(Ctx).Err(); err != nil {
		logger.S.Warnf("Failed to connect to Redis: %v", err)
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
func SetAndQueueJSON(entityType, entityID string, value interface{}) error {
	if Client == nil {
		return nil
	}

	key := entityType + ":" + entityID
	err := SetJSON(key, value, 0) // No expiration for pending DB writes
	if err != nil {
		return err
	}

	// Add to pending set to be processed by the batch persister
	return Client.SAdd(Ctx, "pending_db_writes:"+entityType, entityID).Err()
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

