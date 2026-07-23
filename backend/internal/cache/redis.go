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
