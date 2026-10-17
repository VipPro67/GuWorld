package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var defaultStore RateLimitStore

const rateLimitMultiplier = 5

func SetDefaultStore(s RateLimitStore) {
	defaultStore = s
}

func GetDefaultStore() RateLimitStore {
	return defaultStore
}

// RateLimit — backward-compatible per-IP sliding window (delegates to store).
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return RateLimitByIP(limit, window)
}

// RateLimitByIP limits requests per client IP.
func RateLimitByIP(limit int, window time.Duration) gin.HandlerFunc {
	limit *= rateLimitMultiplier
	return func(c *gin.Context) {
		key := "ip:" + c.ClientIP()
		ok, _ := defaultStore.Allow(key, limit, window)
		if !ok {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Tốc độ yêu cầu quá nhanh. Đạo hữu vui lòng đợi ít phút rồi thử lại!",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AllowPlayer checks whether a player is allowed to proceed under the rate limit.
func AllowPlayer(id uuid.UUID, limit int, window time.Duration) bool {
	if defaultStore == nil {
		return true
	}
	limit *= rateLimitMultiplier
	key := "player:" + id.String()
	allowed, _ := defaultStore.Allow(key, limit, window)
	return allowed
}

// RateLimitByPlayer limits requests per authenticated player.
// Must be placed AFTER AuthRequired().
func RateLimitByPlayer(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := GetPlayerID(c)
		if !ok {
			c.Next()
			return
		}
		if !AllowPlayer(id, limit, window) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Đạo hữu thao tác quá nhanh. Vui lòng đợi rồi thử lại!",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitByGroup limits requests across an entire API group (shared counter).
func RateLimitByGroup(groupName string, limit int, window time.Duration) gin.HandlerFunc {
	limit *= rateLimitMultiplier
	return func(c *gin.Context) {
		// Use IP as secondary key so different IPs have independent group counters
		key := "group:" + groupName + ":" + c.ClientIP()
		ok, _ := defaultStore.Allow(key, limit, window)
		if !ok {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Quá nhiều cổ sư đang sử dụng. Vui lòng đợi 1 chút",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
