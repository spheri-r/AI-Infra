package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.Mutex
	rate     int
	window   time.Duration
}

type Visitor struct {
	count       int
	lastSeen    time.Time
	windowStart time.Time
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     requestsPerSecond,
		window:   time.Second,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	visitor, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &Visitor{
			count:       1,
			lastSeen:    now,
			windowStart: now,
		}
		return true
	}

	// Clean up old visitors periodically
	if now.Sub(visitor.lastSeen) > time.Minute {
		delete(rl.visitors, ip)
		rl.visitors[ip] = &Visitor{
			count:       1,
			lastSeen:    now,
			windowStart: now,
		}
		return true
	}

	// Reset window if it has passed
	if now.Sub(visitor.windowStart) >= rl.window {
		visitor.count = 1
		visitor.windowStart = now
		visitor.lastSeen = now
		return true
	}

	// Check if rate limit exceeded
	if visitor.count >= rl.rate {
		visitor.lastSeen = now
		return false
	}

	visitor.count++
	visitor.lastSeen = now
	return true
}

func RateLimit(requestsPerSecond int) gin.HandlerFunc {
	limiter := NewRateLimiter(requestsPerSecond)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": 1,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Cleanup old visitors periodically
func (rl *RateLimiter) StartCleanup() {
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for ip, visitor := range rl.visitors {
				if now.Sub(visitor.lastSeen) > time.Minute*5 {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
}
