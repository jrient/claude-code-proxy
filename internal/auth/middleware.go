package auth

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements token bucket rate limiting per API key
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[int64]*tokenBucket
}

type tokenBucket struct {
	tokens    float64
	maxTokens float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[int64]*tokenBucket),
	}
}

func (rl *RateLimiter) Allow(keyID int64, rateLimit int) bool {
	if rateLimit <= 0 {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.buckets[keyID]
	if !ok {
		bucket = &tokenBucket{
			tokens:     float64(rateLimit),
			maxTokens:  float64(rateLimit),
			refillRate: float64(rateLimit) / 60.0, // per second, limit is per minute
			lastRefill: time.Now(),
		}
		rl.buckets[keyID] = bucket
	}

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastRefill = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}
	return false
}

// AuthMiddleware validates the API key from the request
func AuthMiddleware(keyManager *APIKeyManager, rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract API key from header
		apiKey := c.GetHeader("X-Api-Key")
		if apiKey == "" {
			// Also check Authorization: Bearer header
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"type":    "error",
				"error":   gin.H{"type": "authentication_error", "message": "Missing API key"},
			})
			c.Abort()
			return
		}

		// Validate the key
		key, err := keyManager.ValidateKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"type":    "error",
				"error":   gin.H{"type": "authentication_error", "message": "Invalid API key"},
			})
			c.Abort()
			return
		}

		// Check rate limit
		if !rateLimiter.Allow(key.ID, key.RateLimit) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"type":    "error",
				"error":   gin.H{"type": "rate_limit_error", "message": "Rate limit exceeded"},
			})
			c.Abort()
			return
		}

		// Store key info in context
		c.Set("api_key", key)
		c.Set("api_key_id", key.ID)
		c.Next()
	}
}

// AdminAuthMiddleware checks admin password
func AdminAuthMiddleware(password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check Authorization header
		auth := c.GetHeader("Authorization")
		if auth == "" {
			// Check cookie
			auth, _ = c.Cookie("admin_token")
		}

		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Simple bearer token check
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			c.Abort()
			return
		}

		c.Next()
	}
}
