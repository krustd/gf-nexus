package middleware

import (
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/config"
)

// CORS 处理跨域请求
func CORS(cfg config.CORSConfig) ghttp.HandlerFunc {
	if !cfg.Enabled {
		return passthrough
	}

	allowOriginSet := make(map[string]bool, len(cfg.AllowOrigins))
	allowAll := false
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			allowAll = true
		}
		allowOriginSet[o] = true
	}

	methods := strings.Join(cfg.AllowMethods, ", ")
	headers := strings.Join(cfg.AllowHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAgeSec)

	return func(r *ghttp.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			r.Middleware.Next()
			return
		}

		if allowAll || allowOriginSet[origin] {
			// CORS 规范：AllowCredentials 与 wildcard "*" 不能同时使用
			if allowAll && !cfg.AllowCredentials {
				r.Response.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				r.Response.Header().Set("Access-Control-Allow-Origin", origin)
				r.Response.Header().Set("Vary", "Origin")
			}
			r.Response.Header().Set("Access-Control-Allow-Methods", methods)
			r.Response.Header().Set("Access-Control-Allow-Headers", headers)
			r.Response.Header().Set("Access-Control-Max-Age", maxAge)
			if cfg.AllowCredentials {
				r.Response.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		// OPTIONS 预检请求直接返回
		if r.Method == "OPTIONS" {
			r.Response.WriteStatus(204)
			return
		}

		r.Middleware.Next()
	}
}

func passthrough(r *ghttp.Request) {
	r.Middleware.Next()
}
