package admin

import (
	"os"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/krustd/gf-nexus/nexus-config/storage"
)

// SetupRouter 设置 Admin API 路由
func SetupRouter(s *ghttp.Server, store storage.Storage, notifier ConfigNotifier) {
	handler := NewHandler(store, notifier)

	// 检测静态文件路径（支持从不同目录运行）
	assetsPath := "web/dist/assets"
	indexPath := "web/dist/index.html"

	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		// 尝试从项目根目录路径
		assetsPath = "Nexus-Config/web/dist/assets"
		indexPath = "Nexus-Config/web/dist/index.html"
	}

	// API 路由组
	s.Group("/api/v1", func(group *ghttp.RouterGroup) {
		// Namespace 管理
		group.Group("/namespaces", func(g *ghttp.RouterGroup) {
			g.POST("/", handler.CreateNamespace)
			g.GET("/", handler.ListNamespaces)
			g.GET("/:id", handler.GetNamespace)
			g.DELETE("/:id", handler.DeleteNamespace)
		})

		// 配置管理
		group.Group("/configs", func(g *ghttp.RouterGroup) {
			g.POST("/draft", handler.SaveDraft)
			g.GET("/draft", handler.GetDraft)
			g.POST("/publish", handler.PublishConfig)
			g.GET("/published", handler.GetPublished)
			g.GET("/list", handler.ListConfigs)
			g.DELETE("/", handler.DeleteConfig)
		})

		// 灰度规则管理
		group.Group("/gray", func(g *ghttp.RouterGroup) {
			g.POST("/", handler.SaveGrayRule)
			g.GET("/", handler.GetGrayRule)
			g.DELETE("/", handler.DeleteGrayRule)
			g.GET("/list", handler.ListGrayRules)
		})
	})

	// 静态文件服务（Web UI）
	// 静态资源文件（JS、CSS 等）
	s.AddStaticPath("/assets", assetsPath)

	// SPA 路由支持：所有非 API 和非静态资源的请求都返回 index.html
	// 使用闭包捕获 indexPath
	indexPathFinal := indexPath
	s.BindHandler("/*", func(r *ghttp.Request) {
		path := r.URL.Path

		// API 请求跳过，交给 API 路由处理
		if len(path) >= 4 && path[:4] == "/api" {
			r.Middleware.Next()
			return
		}

		// 静态资源请求跳过
		if len(path) >= 7 && path[:7] == "/assets" {
			r.Middleware.Next()
			return
		}

		// 其他所有请求（包括根路径和前端路由）都返回 index.html
		r.Response.ServeFile(indexPathFinal)
	})
}
