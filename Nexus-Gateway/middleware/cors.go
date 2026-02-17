package middleware

import (
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/config"
)

// CORS 处理跨域请求（动态读取配置）
func CORS(holder *config.DynamicConfigHolder) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		cfg := holder.Load().CORS

		if !cfg.Enabled {
			r.Middleware.Next()
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			r.Middleware.Next()
			return
		}

		// 判断 origin 是否允许
		allowAll := false
		allowed := false
		for _, o := range cfg.AllowOrigins {
			if o == "*" {
				allowAll = true
				allowed = true
				break
			}
			if o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			methods := strings.Join(cfg.AllowMethods, ", ")
			headers := strings.Join(cfg.AllowHeaders, ", ")
			maxAge := strconv.Itoa(cfg.MaxAgeSec)

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
