// Package nexus 是 Nexus 微服务 SDK 的顶层入口。
//
// 业务方只需要一行代码即可完成服务注册：
//
//	nexus.MustSetup("config/config.toml")
//
// 优雅退出：
//
//	defer nexus.Shutdown()
package Nexus_SDK

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/krustd/nexus-sdk/registry"
)

// instance 记录当前注册的服务实例，用于 Shutdown 时反注册
var currentInstance *registry.ServiceInstance

// ================================================================
// 一行注册 API
// ================================================================

// Setup 从 TOML 配置文件初始化注册中心并注册当前服务。
//
//	err := nexus.Setup("config/config.toml")
func Setup(configPath string) error {
	// 1. 加载配置
	conf, err := registry.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("nexus: load config: %w", err)
	}

	// 2. 创建 Registry
	reg, err := registry.New(&conf.Registry)
	if err != nil {
		return fmt.Errorf("nexus: create registry: %w", err)
	}
	registry.SetGlobal(reg)

	// 3. 构建 ServiceInstance
	instance := conf.Service.ToInstance()

	// 4. 注册
	ctx, cancel := context.WithTimeout(context.Background(), conf.Registry.DialTimeout())
	defer cancel()

	if err := reg.Register(ctx, instance); err != nil {
		return fmt.Errorf("nexus: register service: %w", err)
	}

	currentInstance = instance
	log.Printf("[nexus] ✅ service registered: %s (%s) at %s",
		instance.Name, instance.Protocol, instance.Address)

	return nil
}

// MustSetup 同 Setup，失败直接 panic（适合在 main 中使用）
//
//	func main() {
//	    nexus.MustSetup("config/config.toml")
//	    defer nexus.Shutdown()
//	    // ... 启动 gf server
//	}
func MustSetup(configPath string) {
	if err := Setup(configPath); err != nil {
		panic(err)
	}
}

// ================================================================
// 同时注册多协议（HTTP + gRPC 同服务）
// ================================================================

// SetupMulti 从配置文件加载，并注册多个自定义实例。
// 适用于一个进程同时暴露 HTTP + gRPC 的场景。
//
//	nexus.MustSetupMulti("config/config.toml",
//	    &registry.ServiceInstance{Name: "user-svc", Protocol: registry.ProtocolHTTP, Address: ":8080"},
//	    &registry.ServiceInstance{Name: "user-svc", Protocol: registry.ProtocolGRPC, Address: ":9090"},
//	)
func SetupMulti(configPath string, instances ...*registry.ServiceInstance) error {
	conf, err := registry.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("nexus: load config: %w", err)
	}

	reg, err := registry.New(&conf.Registry)
	if err != nil {
		return fmt.Errorf("nexus: create registry: %w", err)
	}
	registry.SetGlobal(reg)

	ctx, cancel := context.WithTimeout(context.Background(), conf.Registry.DialTimeout())
	defer cancel()

	for _, inst := range instances {
		if err := reg.Register(ctx, inst); err != nil {
			return fmt.Errorf("nexus: register %s: %w", inst.ID, err)
		}
	}

	log.Printf("[nexus] ✅ %d instances registered", len(instances))
	return nil
}

// MustSetupMulti 同 SetupMulti，失败 panic
func MustSetupMulti(configPath string, instances ...*registry.ServiceInstance) {
	if err := SetupMulti(configPath, instances...); err != nil {
		panic(err)
	}
}

// ================================================================
// 服务发现快捷方法
// ================================================================

// Discover 发现某个服务的所有实例
func Discover(serviceName string) ([]*registry.ServiceInstance, error) {
	reg := registry.GetGlobal()
	if reg == nil {
		return nil, fmt.Errorf("nexus: not initialized, call Setup first")
	}
	return reg.Discover(context.Background(), serviceName)
}

// DiscoverHTTP 发现 HTTP 实例
func DiscoverHTTP(serviceName string) ([]*registry.ServiceInstance, error) {
	reg := registry.GetGlobal()
	if reg == nil {
		return nil, fmt.Errorf("nexus: not initialized")
	}
	return reg.DiscoverByProtocol(context.Background(), serviceName, registry.ProtocolHTTP)
}

// DiscoverGRPC 发现 gRPC 实例
func DiscoverGRPC(serviceName string) ([]*registry.ServiceInstance, error) {
	reg := registry.GetGlobal()
	if reg == nil {
		return nil, fmt.Errorf("nexus: not initialized")
	}
	return reg.DiscoverByProtocol(context.Background(), serviceName, registry.ProtocolGRPC)
}

// ================================================================
// 生命周期
// ================================================================

// Shutdown 优雅关闭：反注册当前服务 + 关闭 etcd 连接
func Shutdown() {
	reg := registry.GetGlobal()
	if reg == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 反注册当前实例
	if currentInstance != nil {
		if err := reg.Deregister(ctx, currentInstance); err != nil {
			log.Printf("[nexus] deregister failed: %v", err)
		}
		currentInstance = nil
	}

	if err := registry.Shutdown(); err != nil {
		log.Printf("[nexus] shutdown failed: %v", err)
	}
	log.Println("[nexus] shutdown complete")
}

// GetRegistry 获取底层 Registry 实例（高级用法）
func GetRegistry() *registry.Registry {
	return registry.GetGlobal()
}
