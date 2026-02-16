package middleware

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/internal"
	"github.com/krustd/nexus-gateway/metrics"
)

// Logging 请求日志中间件，记录请求信息和延迟
func Logging() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		start := time.Now()

		r.Middleware.Next()

		latency := time.Since(start)
		ctx := r.GetCtx()
		status := r.Response.Status

		g.Log().Infof(ctx,
			"[gateway] %s %s | status=%d | latency=%v | ip=%s | trace_id=%s | request_id=%s",
			r.Method,
			r.URL.Path,
			status,
			latency,
			r.GetClientIp(),
			internal.GetTraceID(ctx),
			r.Header.Get("X-Request-Id"),
		)

		// 记录 Prometheus 指标
		metrics.RecordRequest(r.Method, r.URL.Path, status, latency)
	}
}
