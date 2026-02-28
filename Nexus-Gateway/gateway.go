// Package nexus 是 Nexus-Gateway 的顶层入口。
//
//	nexus.MustSetup("config/config.toml")
//	nexus.Start() // 阻塞
package nexus

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/krustd/gf-nexus/nexus-config/common"
	"github.com/krustd/gf-nexus/nexus-config/sdk"
	"github.com/krustd/gf-nexus/nexus-gateway/config"
	"github.com/krustd/gf-nexus/nexus-gateway/gateway"
	"github.com/krustd/gf-nexus/nexus-registry/registry"
	"github.com/krustd/gf-nexus/nexus-registry/registry/etcd"
)

var (
	gw           *gateway.Gateway
	configClient *sdk.Client
	holder       *config.DynamicConfigHolder
)

// Setup 从 TOML 配置文件初始化网关
func Setup(configPath string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("nexus-gateway: load config: %w", err)
	}

	// 连接 etcd 注册中心
	regCfg := &registry.Config{
		Endpoints:      cfg.Registry.Endpoints,
		DialTimeoutSec: cfg.Registry.DialTimeoutSec,
		Prefix:         cfg.Registry.Prefix,
		Username:       cfg.Registry.Username,
		Password:       cfg.Registry.Password,
	}
	reg, err := etcd.New(regCfg)
	if err != nil {
		return fmt.Errorf("nexus-gateway: create registry: %w", err)
	}

	// 初始化动态配置持有器
	holder = config.NewDynamicConfigHolder()

	// 连接配置中心
	if cfg.ConfigCenter.ServerAddr != "" {
		if err := setupConfigCenter(cfg.ConfigCenter); err != nil {
			reg.Close(context.Background())
			return fmt.Errorf("nexus-gateway: setup config center: %w", err)
		}
	} else {
		log.Println("[nexus-gateway] config center not configured, using defaults")
	}

	gw, err = gateway.New(cfg, holder, reg)
	if err != nil {
		reg.Close(context.Background())
		if configClient != nil {
			configClient.Stop()
		}
		return fmt.Errorf("nexus-gateway: create gateway: %w", err)
	}

	log.Printf("[nexus-gateway] initialized, addr=%s", cfg.Server.Addr)
	return nil
}

// setupConfigCenter 连接配置中心，首次拉取动态配置并启动长轮询
func setupConfigCenter(ccCfg config.ConfigCenterConfig) error {
	clientID := ccCfg.ClientID
	if clientID == "" {
		clientID, _ = os.Hostname()
	}

	sdkCfg := &common.ClientConfig{
		ServerAddr:  ccCfg.ServerAddr,
		Namespace:   ccCfg.Namespace,
		ConfigKey:   ccCfg.ConfigKey,
		ClientID:    clientID,
		PollTimeout: ccCfg.PollTimeout,
		RetryDelay:  ccCfg.RetryDelay,
	}

	configClient = sdk.NewClient(sdkCfg)

	// 注册配置变更监听器
	configClient.AddChangeListener(func(version *common.ConfigVersion) {
		dynCfg := &config.DynamicConfig{}
		if err := common.ParseConfig(version.Value, common.ConfigFormat(version.Format), dynCfg); err != nil {
			log.Printf("[nexus-gateway] parse dynamic config failed: %v", err)
			return
		}
		holder.Store(dynCfg)
		log.Printf("[nexus-gateway] dynamic config updated, md5=%s", version.MD5)
	})

	// 启动客户端（首次拉取 + 长轮询）
	if err := configClient.Start(context.Background()); err != nil {
		return fmt.Errorf("start config client: %w", err)
	}

	// 尝试将首次拉取的配置解析到 holder
	if version, err := configClient.GetConfig(); err == nil {
		dynCfg := &config.DynamicConfig{}
		if err := common.ParseConfig(version.Value, common.ConfigFormat(version.Format), dynCfg); err != nil {
			log.Printf("[nexus-gateway] parse initial dynamic config failed: %v", err)
		} else {
			holder.Store(dynCfg)
			log.Println("[nexus-gateway] initial dynamic config loaded from config center")
		}
	} else {
		log.Printf("[nexus-gateway] initial config fetch not available, using defaults: %v", err)
	}

	return nil
}

// MustSetup 同 Setup，失败 panic
func MustSetup(configPath string) {
	if err := Setup(configPath); err != nil {
		panic(err)
	}
}

// Start 启动网关（阻塞）
func Start() {
	if gw == nil {
		panic("nexus-gateway: not initialized, call Setup first")
	}
	gw.Start()
}

// Shutdown 优雅关闭
func Shutdown() {
	if configClient != nil {
		configClient.Stop()
	}
	if gw != nil {
		gw.Shutdown()
		log.Println("[nexus-gateway] shutdown complete")
	}
}
