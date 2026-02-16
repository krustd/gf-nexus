// Package nexus 是 Nexus-Gateway 的顶层入口。
//
//	nexus.MustSetup("config/config.toml")
//	nexus.Start() // 阻塞
package nexus

import (
	"context"
	"fmt"
	"log"

	"github.com/krustd/nexus-gateway/config"
	"github.com/krustd/nexus-gateway/gateway"
	"github.com/krustd/nexus-registry/registry"
	"github.com/krustd/nexus-registry/registry/etcd"
)

var gw *gateway.Gateway

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

	gw, err = gateway.New(cfg, reg)
	if err != nil {
		// 避免 registry 连接泄漏
		reg.Close(context.Background())
		return fmt.Errorf("nexus-gateway: create gateway: %w", err)
	}

	log.Printf("[nexus-gateway] initialized, addr=%s", cfg.Server.Addr)
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
	if gw != nil {
		gw.Shutdown()
		log.Println("[nexus-gateway] shutdown complete")
	}
}
