package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/krustd/nexus-config/admin"
	"github.com/krustd/nexus-config/common"
	"github.com/krustd/nexus-config/server"
	"github.com/krustd/nexus-config/storage/sqlite"
)

func main() {
	ctx := context.Background()

	// 加载配置（默认从 config.toml 加载）
	configPath := "config.toml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := common.LoadServerConfig(configPath)
	if err != nil {
		g.Log().Fatalf(ctx, "加载配置失败: %v", err)
	}

	// 初始化存储
	store, err := sqlite.NewSQLiteStorage(cfg.Database.FilePath)
	if err != nil {
		g.Log().Fatalf(ctx, "初始化存储失败: %v", err)
	}

	if err := store.Init(ctx); err != nil {
		g.Log().Fatalf(ctx, "初始化数据库失败: %v", err)
	}
	defer store.Close()

	g.Log().Info(ctx, "存储初始化成功")

	// 创建配置变更通知器
	notifier := server.NewConfigNotifier()

	// 启动 Admin API 服务（提供 Web UI + API）
	adminServer := g.Server("admin")
	admin.SetupRouter(adminServer, store, notifier)
	adminServer.SetAddr(cfg.Admin.Addr)
	adminServer.SetDumpRouterMap(false)
	go func() {
		g.Log().Infof(ctx, "Admin Server (含 Web UI) 启动于: %s", cfg.Admin.Addr)
		adminServer.Start()
	}()

	// 启动配置分发服务
	configServer := g.Server("config")
	server.SetupRouter(configServer, store, notifier)
	configServer.SetAddr(cfg.Server.Addr)
	configServer.SetDumpRouterMap(false)
	go func() {
		g.Log().Infof(ctx, "Config Server 启动于: %s", cfg.Server.Addr)
		configServer.Start()
	}()

	g.Log().Info(ctx, "所有服务启动成功")
	g.Log().Infof(ctx, "Web UI: http://localhost%s", cfg.Admin.Addr)
	g.Log().Infof(ctx, "Admin API: http://localhost%s/api/v1", cfg.Admin.Addr)
	g.Log().Infof(ctx, "Config API: http://localhost%s", cfg.Server.Addr)

	// 等待退出信号（Ctrl+C）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	g.Log().Info(ctx, "正在关闭服务...")
	adminServer.Shutdown()
	configServer.Shutdown()
	g.Log().Info(ctx, "服务已停止")
}
