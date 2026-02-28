package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/krustd/gf-nexus/nexus-config/common"
	"github.com/krustd/gf-nexus/nexus-config/sdk"
)

// AppConfig 应用配置示例
type AppConfig struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Features map[string]bool `yaml:"features"`
}

func main() {
	ctx := context.Background()

	// 加载客户端配置
	cfg, err := common.LoadClientConfig("example/client/config.toml")
	if err != nil {
		g.Log().Fatalf(ctx, "load config failed: %v", err)
	}

	// 创建配置客户端
	client := sdk.NewClient(cfg)

	// 添加配置变更监听器
	client.AddChangeListener(func(version *common.ConfigVersion) {
		g.Log().Infof(ctx, "config changed! namespace=%s, key=%s, md5=%s",
			version.Namespace, version.Key, version.MD5)

		// 解析配置
		var appCfg AppConfig
		if err := common.ParseConfig(version.Value, common.ConfigFormat(version.Format), &appCfg); err != nil {
			g.Log().Errorf(ctx, "parse config failed: %v", err)
			return
		}

		g.Log().Infof(ctx, "new config: %+v", appCfg)

		// 这里可以执行配置热更新逻辑
		// 例如：重新加载数据库连接、更新特性开关等
	})

	// 启动客户端
	if err := client.Start(ctx); err != nil {
		g.Log().Fatalf(ctx, "start client failed: %v", err)
	}
	defer client.Stop()

	g.Log().Info(ctx, "config client started successfully")

	// 获取当前配置
	var appCfg AppConfig
	if err := client.GetValueAs(&appCfg); err != nil {
		g.Log().Warningf(ctx, "get config failed (will retry via long polling): %v", err)
	} else {
		g.Log().Infof(ctx, "current config: %+v", appCfg)
	}

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	g.Log().Info(ctx, "shutting down client...")
}
