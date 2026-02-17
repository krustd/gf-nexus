package middleware

import (
	"sync"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/config"
	"github.com/krustd/nexus-gateway/internal"
)

// tokenBucket 令牌桶限流器，支持动态调整 rate 和 capacity
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64 // tokens per second
	lastTime time.Time
}

func (tb *tokenBucket) allow(rate float64, capacity float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()

	// 动态更新 rate 和 capacity
	tb.rate = rate
	if capacity != tb.capacity {
		tb.capacity = capacity
	}

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

// RateLimit 全局令牌桶限流中间件（动态读取配置）
func RateLimit(holder *config.DynamicConfigHolder) ghttp.HandlerFunc {
	bucket := &tokenBucket{
		tokens:   2000,
		capacity: 2000,
		rate:     1000,
		lastTime: time.Now(),
	}

	// 用首次加载的配置初始化
	if cfg := holder.Load(); cfg != nil {
		bucket.tokens = float64(cfg.RateLimit.Burst)
		bucket.capacity = float64(cfg.RateLimit.Burst)
		bucket.rate = cfg.RateLimit.Rate
	}

	return func(r *ghttp.Request) {
		cfg := holder.Load().RateLimit

		if !cfg.Enabled {
			r.Middleware.Next()
			return
		}

		if !bucket.allow(cfg.Rate, float64(cfg.Burst)) {
			internal.GatewayError(r, internal.CodeRateLimited, "rate limit exceeded")
			return
		}
		r.Middleware.Next()
	}
}
