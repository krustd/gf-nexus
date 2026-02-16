package gateway

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/internal"
)

// Re-export error codes
const (
	CodeOK              = internal.CodeOK
	CodeJWTInvalid      = internal.CodeJWTInvalid
	CodeIPBlocked       = internal.CodeIPBlocked
	CodeRateLimited     = internal.CodeRateLimited
	CodeCircuitOpen     = internal.CodeCircuitOpen
	CodeServiceNotFound = internal.CodeServiceNotFound
	CodeBackendTimeout  = internal.CodeBackendTimeout
	CodeBackendError    = internal.CodeBackendError
)

func GetTraceID(ctx context.Context) string                     { return internal.GetTraceID(ctx) }
func SetTraceID(ctx context.Context, id string) context.Context { return internal.SetTraceID(ctx, id) }
func WriteSuccess(r *ghttp.Request, data interface{})           { internal.WriteSuccess(r, data) }
func WriteError(r *ghttp.Request, httpStatus, code int, msg string) {
	internal.WriteError(r, httpStatus, code, msg)
}
func GatewayError(r *ghttp.Request, code int, msg string) { internal.GatewayError(r, code, msg) }
