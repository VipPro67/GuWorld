package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests []time.Time
}

var (
	clients = make(map[string]*rateLimiter)
	mu      sync.Mutex
)

func init() {
	// Periodic clean-up routine to prevent memory leaks from inactive IPs
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			mu.Lock()
			now := time.Now()
			for ip, limiter := range clients {
				var active []time.Time
				for _, t := range limiter.requests {
					// Clean up requests older than 1 hour
					if now.Sub(t) < time.Hour {
						active = append(active, t)
					}
				}
				if len(active) == 0 {
					delete(clients, ip)
				} else {
					limiter.requests = active
				}
			}
			mu.Unlock()
		}
	}()
}

// RateLimit implements sliding window rate limiting per IP address
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		limiter, exists := clients[ip]
		if !exists {
			limiter = &rateLimiter{}
			clients[ip] = limiter
		}

		// Filter request timestamps within the sliding window
		var active []time.Time
		for _, t := range limiter.requests {
			if now.Sub(t) < window {
				active = append(active, t)
			}
		}

		if len(active) >= limit {
			mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Tốc độ yêu cầu quá nhanh. Đạo hữu vui lòng đợi ít phút rồi thử lại!",
			})
			c.Abort()
			return
		}

		limiter.requests = append(active, now)
		mu.Unlock()
		c.Next()
	}
}
