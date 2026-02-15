package server

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/krustd/nexus-config/storage"
)

// SetupRouter 设置配置分发服务路由
func SetupRouter(s *ghttp.Server, store storage.Storage, notifier *ConfigNotifier) *Handler {
	handler := NewHandler(store, notifier)

	// 健康检查
	s.BindHandler("/health", handler.HealthCheck)

	// 配置拉取 API
	s.Group("/api/v1/config", func(group *ghttp.RouterGroup) {
		group.POST("/poll", handler.PollConfig)  // 长轮询
		group.POST("/get", handler.GetConfig)    // 立即获取
	})

	return handler
}
