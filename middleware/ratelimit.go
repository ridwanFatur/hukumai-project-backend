package middleware

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanFatur/hukumai-project-backend/models"
	"golang.org/x/time/rate"
)

// entry holds a limiter and its last-access time for cleanup.
type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// limiterStore is a thread-safe map of key → token-bucket limiter.
type limiterStore struct {
	mu       sync.Mutex
	entries  map[string]*entry
	r        rate.Limit
	burst    int
}

func newStore(r rate.Limit, burst int) *limiterStore {
	s := &limiterStore{
		entries: make(map[string]*entry),
		r:       r,
		burst:   burst,
	}
	// Periodically evict entries that have been idle for > 10 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.mu.Lock()
			for k, e := range s.entries {
				if time.Since(e.lastSeen) > 10*time.Minute {
					delete(s.entries, k)
				}
			}
			s.mu.Unlock()
		}
	}()
	return s
}

func (s *limiterStore) get(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.entries[key]; ok {
		e.lastSeen = time.Now()
		return e.limiter
	}
	lim := rate.NewLimiter(s.r, s.burst)
	s.entries[key] = &entry{limiter: lim, lastSeen: time.Now()}
	return lim
}

// tooManyRequests is the standard 429 response body.
func tooManyRequests(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error": "Terlalu banyak permintaan. Silakan coba lagi nanti.",
	})
	c.Abort()
}

// RateLimitIP returns a middleware that rate-limits by client IP.
//
// r     – sustained request rate (use rate.Every(time.Minute) / N for per-minute)
// burst – maximum burst size (how many back-to-back requests are allowed)
//
// Override defaults at runtime with the env var named by envKey (integer = requests/minute).
// Set envKey to "" to skip env override.
//
// Example:
//
//	// TODO: adjust RATE_LOGIN_RPM in .env to tune the login rate limit
//	RateLimitIP(rate.Every(time.Minute)/10, 5, "RATE_LOGIN_RPM")
func RateLimitIP(r rate.Limit, burst int, envKey string) gin.HandlerFunc {
	r, burst = applyEnvOverride(r, burst, envKey)
	store := newStore(r, burst)
	return func(c *gin.Context) {
		if !store.get(c.ClientIP()).Allow() {
			tooManyRequests(c)
			return
		}
		c.Next()
	}
}

// RateLimitUser returns a middleware that rate-limits by authenticated user ID.
// Must be placed after AuthMiddleware so that "user" is set in the Gin context.
//
// Override defaults at runtime with the env var named by envKey (integer = requests/minute).
// Set envKey to "" to skip env override.
//
// Example:
//
//	// TODO: adjust RATE_CHAT_RPM in .env to tune the chat rate limit
//	RateLimitUser(rate.Every(time.Minute)/20, 5, "RATE_CHAT_RPM")
func RateLimitUser(r rate.Limit, burst int, envKey string) gin.HandlerFunc {
	r, burst = applyEnvOverride(r, burst, envKey)
	store := newStore(r, burst)
	return func(c *gin.Context) {
		u, exists := c.Get("user")
		if !exists {
			// Auth middleware already rejects unauthenticated requests; skip if somehow missing.
			c.Next()
			return
		}
		key := strconv.FormatUint(uint64(u.(models.User).ID), 10)
		if !store.get(key).Allow() {
			tooManyRequests(c)
			return
		}
		c.Next()
	}
}

// applyEnvOverride reads envKey as an integer (requests/minute) and returns
// an updated (rate, burst) pair. Burst is set to 20% of the env value (min 1).
// If envKey is "" or the env var is unset/invalid, the originals are returned.
func applyEnvOverride(r rate.Limit, burst int, envKey string) (rate.Limit, int) {
	if envKey == "" {
		return r, burst
	}
	val := os.Getenv(envKey)
	if val == "" {
		return r, burst
	}
	rpm, err := strconv.Atoi(val)
	if err != nil || rpm <= 0 {
		return r, burst
	}
	newBurst := rpm / 5
	if newBurst < 1 {
		newBurst = 1
	}
	return rate.Every(time.Minute) / rate.Limit(rpm), newBurst
}
