package middleware

import (
	"sync"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/config"
	"github.com/krustd/nexus-gateway/internal"
)

// tokenBucket 令牌桶限流器
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64 // tokens per second
	lastTime time.Time
}

func (tb *tokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastTime = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// RateLimit 全局令牌桶限流中间件
func RateLimit(cfg config.RateLimitConfig) ghttp.HandlerFunc {
	if !cfg.Enabled {
		return passthrough
	}

	bucket := &tokenBucket{
		tokens:   float64(cfg.Burst),
		capacity: float64(cfg.Burst),
		rate:     cfg.Rate,
		lastTime: time.Now(),
	}

	return func(r *ghttp.Request) {
		if !bucket.Allow() {
			internal.GatewayError(r, internal.CodeRateLimited, "rate limit exceeded")
			return
		}
		r.Middleware.Next()
	}
}
