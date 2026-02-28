package middleware

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

// RequestID 生成或传递 X-Request-Id
func RequestID() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		r.Request.Header.Set("X-Request-Id", reqID)
		r.Response.Header().Set("X-Request-Id", reqID)
		r.Middleware.Next()
	}
}
