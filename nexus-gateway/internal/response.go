package internal

import (
	"context"
	"net/http"

	"github.com/gogf/gf/v2/net/ghttp"
)

// 网关错误码
const (
	CodeOK              = 0
	CodeJWTInvalid      = 1001
	CodeIPBlocked       = 1002
	CodeRateLimited     = 1003
	CodeCircuitOpen     = 1004
	CodeServiceNotFound = 1005
	CodeBackendTimeout  = 1006
	CodeBackendError    = 1007
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

type contextKey string

const traceIDKey contextKey = "trace_id"

func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func WriteSuccess(r *ghttp.Request, data interface{}) {
	r.Response.WriteJson(Response{
		Code:    CodeOK,
		Message: "ok",
		Data:    data,
		TraceID: GetTraceID(r.GetCtx()),
	})
}

func WriteError(r *ghttp.Request, httpStatus int, code int, message string) {
	r.Response.WriteStatus(httpStatus)
	r.Response.WriteJson(Response{
		Code:    code,
		Message: message,
		TraceID: GetTraceID(r.GetCtx()),
	})
}

// GatewayError 返回标准网关错误响应
func GatewayError(r *ghttp.Request, code int, msg string) {
	switch code {
	case CodeJWTInvalid:
		WriteError(r, http.StatusUnauthorized, code, msg)
	case CodeIPBlocked:
		WriteError(r, http.StatusForbidden, code, msg)
	case CodeRateLimited:
		WriteError(r, http.StatusTooManyRequests, code, msg)
	case CodeCircuitOpen:
		WriteError(r, http.StatusServiceUnavailable, code, msg)
	case CodeServiceNotFound:
		WriteError(r, http.StatusBadGateway, code, msg)
	case CodeBackendTimeout:
		WriteError(r, http.StatusGatewayTimeout, code, msg)
	case CodeBackendError:
		WriteError(r, http.StatusBadGateway, code, msg)
	default:
		WriteError(r, http.StatusInternalServerError, code, msg)
	}
}
