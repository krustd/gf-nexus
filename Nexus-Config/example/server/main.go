package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/krustd/gf-nexus/nexus-config/admin"
	"github.com/krustd/gf-nexus/nexus-config/common"
	"github.com/krustd/gf-nexus/nexus-config/server"
	"github.com/krustd/gf-nexus/nexus-config/storage/sqlite"
)

func main() {
	ctx := context.Background()

	// 加载配置
	cfg, err := common.LoadServerConfig("example/server/config.toml")
	if err != nil {
		g.Log().Fatalf(ctx, "load config failed: %v", err)
	}

	// 初始化存储
	store, err := sqlite.NewSQLiteStorage(cfg.Database.FilePath)
	if err != nil {
		g.Log().Fatalf(ctx, "create storage failed: %v", err)
	}

	if err := store.Init(ctx); err != nil {
		g.Log().Fatalf(ctx, "init storage failed: %v", err)
	}
	defer store.Close()

	g.Log().Info(ctx, "storage initialized")

	// 创建配置变更通知器
	notifier := server.NewConfigNotifier()

	// 启动 Admin API 服务
	adminServer := g.Server("admin")
	admin.SetupRouter(adminServer, store, notifier)
	adminServer.SetAddr(cfg.Admin.Addr)
	adminServer.SetDumpRouterMap(false)
	go func() {
		g.Log().Infof(ctx, "admin server starting on %s", cfg.Admin.Addr)
		adminServer.Start()
	}()

	// 启动配置分发服务
	configServer := g.Server("config")
	server.SetupRouter(configServer, store, notifier)
	configServer.SetAddr(cfg.Server.Addr)
	configServer.SetDumpRouterMap(false)
	go func() {
		g.Log().Infof(ctx, "config server starting on %s", cfg.Server.Addr)
		configServer.Start()
	}()

	g.Log().Info(ctx, "all servers started successfully")
	g.Log().Infof(ctx, "Admin API: http://localhost%s", cfg.Admin.Addr)
	g.Log().Infof(ctx, "Config API: http://localhost%s", cfg.Server.Addr)

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	g.Log().Info(ctx, "shutting down servers...")
	adminServer.Shutdown()
	configServer.Shutdown()
	g.Log().Info(ctx, "servers stopped")
}
