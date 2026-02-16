package middleware

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"

	"github.com/krustd/nexus-gateway/internal"
)

// Trace 生成或传递 trace_id，注入到 Context 和响应头
func Trace() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		ctx := internal.SetTraceID(r.GetCtx(), traceID)
		r.SetCtx(ctx)

		r.Response.Header().Set("X-Trace-Id", traceID)
		r.Middleware.Next()
	}
}
