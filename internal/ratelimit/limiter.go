package ratelimit

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type LimiterRegistry struct {
	mu      sync.RWMutex
	clients map[string]*client
	rate    rate.Limit
	burst   int
}

// NewRegistry creates a new rate limiter registry
func NewRegistry(rps float64, burst int) *LimiterRegistry {
	r := &LimiterRegistry{
		clients: make(map[string]*client),
		rate:    rate.Limit(rps),
		burst:   burst,
	}

	// Periodically clean up old rate limiters (idle for more than 1 hour)
	go r.startCleanup(10 * time.Minute, 1 * time.Hour)

	return r
}

func (r *LimiterRegistry) GetLimiter(key string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, exists := r.clients[key]
	if !exists {
		limiter := rate.NewLimiter(r.rate, r.burst)
		r.clients[key] = &client{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	c.lastSeen = time.Now()
	return c.limiter
}

func (r *LimiterRegistry) startCleanup(interval, idleTimeout time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for key, c := range r.clients {
			if now.Sub(c.lastSeen) > idleTimeout {
				delete(r.clients, key)
			}
		}
		r.mu.Unlock()
	}
}

// GetEnvFloat helper
func GetEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}

// GetEnvInt helper
func GetEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// Middleware returns the Gin middleware for rate limiting
func (r *LimiterRegistry) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var key string
		merchant := auth.MerchantFrom(c)
		if merchant != nil && merchant.ID != 0 {
			key = fmt.Sprintf("merchant:%d", merchant.ID)
		} else if mID := c.Param("merchant_id"); mID != "" {
			key = fmt.Sprintf("merchant:%s", mID)
		} else {
			key = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		limiter := r.GetLimiter(key)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded - too many requests",
			})
			return
		}

		c.Next()
	}
}
