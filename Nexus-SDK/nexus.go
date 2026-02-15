// Package nexus 是 Nexus 微服务 SDK 的顶层入口。
//
//	nexus.MustSetup("config/config.toml")
//	defer nexus.Shutdown()
package nexus

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/krustd/nexus-sdk/registry"
	"github.com/krustd/nexus-sdk/registry/etcd"
)

var currentInstance *registry.ServiceInstance

// Setup 从 TOML 配置文件初始化并注册当前服务
func Setup(configPath string) error {
	conf, err := registry.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("nexus: load config: %w", err)
	}

	// ★ 唯一绑定具体实现的地方，将来换 consul 只改这一行
	reg, err := etcd.New(&conf.Registry)
	if err != nil {
		return fmt.Errorf("nexus: create registry: %w", err)
	}
	registry.SetGlobal(reg)

	instance := conf.Service.ToInstance()
	ctx, cancel := context.WithTimeout(context.Background(), conf.Registry.DialTimeout())
	defer cancel()

	if err := reg.Register(ctx, instance); err != nil {
		return fmt.Errorf("nexus: register: %w", err)
	}

	currentInstance = instance
	log.Printf("[nexus] ✅ %s (%s) at %s", instance.Name, instance.Protocol, instance.Address)
	return nil
}

// MustSetup 同 Setup，失败 panic
func MustSetup(configPath string) {
	if err := Setup(configPath); err != nil {
		panic(err)
	}
}

// SetupMulti 注册多个实例（HTTP + gRPC 同服务场景）
func SetupMulti(configPath string, instances ...*registry.ServiceInstance) error {
	conf, err := registry.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("nexus: load config: %w", err)
	}

	reg, err := etcd.New(&conf.Registry)
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

// Discover 发现某个服务的所有实例
func Discover(serviceName string) ([]*registry.ServiceInstance, error) {
	reg := registry.GetGlobal()
	if reg == nil {
		return nil, fmt.Errorf("nexus: not initialized")
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

// Shutdown 反注册 + 关闭连接
func Shutdown() {
	reg := registry.GetGlobal()
	if reg == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

// GetRegistry 获取底层 Registry 接口
func GetRegistry() registry.Registry {
	return registry.GetGlobal()
}
