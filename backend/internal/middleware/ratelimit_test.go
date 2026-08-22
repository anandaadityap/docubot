package middleware_test

import (
	"testing"
	"time"

	"github.com/supernand/docubot/backend/internal/middleware"
)

func TestRateLimiter_AllowsThenBlocks(t *testing.T) {
	l := middleware.NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow("1.1.1.1") {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	if l.Allow("1.1.1.1") {
		t.Fatal("4th hit should be blocked")
	}
	if !l.Allow("2.2.2.2") {
		t.Fatal("other IP should be allowed")
	}
}
