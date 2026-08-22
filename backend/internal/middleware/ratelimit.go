package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/util"
)

const defaultChatLimit = 10
const defaultChatWindow = time.Minute

// RateLimiter is a sliding-window counter keyed by client IP.
type RateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
	now    func() time.Time
}

// NewRateLimiter constructs a limiter (max hits per window).
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	if max < 1 {
		max = defaultChatLimit
	}
	if window <= 0 {
		window = defaultChatWindow
	}
	return &RateLimiter{
		hits:   make(map[string][]time.Time),
		max:    max,
		window: window,
		now:    time.Now,
	}
}

// Allow records a hit and reports whether it is within the limit.
func (l *RateLimiter) Allow(key string) bool {
	now := l.now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.hits[key]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}

// ChatRateLimit returns middleware that limits POST /chat to 10 requests/minute/IP.
func ChatRateLimit(l *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}
		if !l.Allow(ip) {
			util.AbortError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, try again in a minute")
			return
		}
		c.Next()
	}
}
