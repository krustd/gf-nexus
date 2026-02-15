package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	registry "github.com/krustd/nexus-registry"
	"github.com/krustd/nexus-registry/balancer"
)

func main() {
	// ============================================================
	// 示例 1: 服务端 —— 注册服务
	// ============================================================

	serverExample()

	// ============================================================
	// 示例 2: 客户端 —— 发现服务 + 负载均衡
	// ============================================================

	// clientExample()
}

func serverExample() {
	// 1. 初始化注册中心
	registry.MustInit(&registry.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
		LeaseTTL:    15,
		Prefix:      "/nexus/services",
	})

	reg := registry.GetGlobal()

	// 2. 定义服务实例
	httpInstance := &registry.ServiceInstance{
		ID:       "user-service-http-10.0.0.1:8080",
		Name:     "user-service",
		Version:  "v1.0.0",
		Protocol: registry.ProtocolHTTP,
		Address:  "10.0.0.1:8080",
		Weight:   10,
		Metadata: map[string]string{
			"region": "ap-northeast-1",
			"env":    "production",
		},
	}

	grpcInstance := &registry.ServiceInstance{
		ID:       "user-service-grpc-10.0.0.1:9090",
		Name:     "user-service",
		Version:  "v1.0.0",
		Protocol: registry.ProtocolGRPC,
		Address:  "10.0.0.1:9090",
		Weight:   10,
		Metadata: map[string]string{
			"region": "ap-northeast-1",
			"env":    "production",
		},
	}

	// 3. 注册（自动续租）
	ctx := context.Background()
	if err := reg.Register(ctx, httpInstance); err != nil {
		log.Fatalf("register http instance: %v", err)
	}
	if err := reg.Register(ctx, grpcInstance); err != nil {
		log.Fatalf("register grpc instance: %v", err)
	}

	fmt.Println("✅ services registered, waiting for signal...")

	// 4. 等待退出信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("shutting down...")
	if err := registry.Shutdown(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	fmt.Println("✅ done")
}

func clientExample() {
	// 1. 初始化注册中心（客户端也需要连 etcd）
	registry.MustInit(&registry.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	})

	reg := registry.GetGlobal()

	// -------------------------------------------------------
	// 方式 A: 简单发现（一次性查询）
	// -------------------------------------------------------
	ctx := context.Background()
	instances, err := reg.Discover(ctx, "user-service")
	if err != nil {
		log.Fatalf("discover: %v", err)
	}
	for _, inst := range instances {
		fmt.Printf("found: %s %s %s\n", inst.Name, inst.Protocol, inst.Address)
	}

	// 只看 gRPC 的
	grpcInstances, _ := reg.DiscoverByProtocol(ctx, "user-service", registry.ProtocolGRPC)
	for _, inst := range grpcInstances {
		fmt.Printf("grpc: %s\n", inst.Address)
	}

	// -------------------------------------------------------
	// 方式 B: Resolver（推荐） —— 自动 Watch + 本地缓存 + 负载均衡
	// -------------------------------------------------------

	// Round Robin 负载均衡
	resolver, err := registry.NewResolver(reg, "user-service",
		registry.WithProtocol(registry.ProtocolHTTP),
		registry.WithPicker(balancer.NewRoundRobin()),
	)
	if err != nil {
		log.Fatalf("create resolver: %v", err)
	}
	defer resolver.Close()

	// 模拟 10 次请求
	for i := 0; i < 10; i++ {
		inst, err := resolver.Resolve()
		if err != nil {
			log.Printf("resolve error: %v", err)
			continue
		}
		fmt.Printf("[%d] → %s %s\n", i, inst.Address, inst.Protocol)
	}

	// -------------------------------------------------------
	// 方式 C: 加权轮询
	// -------------------------------------------------------

	weightedResolver, err := registry.NewResolver(reg, "user-service",
		registry.WithPicker(balancer.NewWeightedRoundRobin()),
	)
	if err != nil {
		log.Fatalf("create weighted resolver: %v", err)
	}
	defer weightedResolver.Close()

	for i := 0; i < 10; i++ {
		inst, _ := weightedResolver.Resolve()
		fmt.Printf("weighted[%d] → %s (weight=%d)\n", i, inst.Address, inst.Weight)
	}

	// -------------------------------------------------------
	// 方式 D: 直接 Watch 事件流（高级用法）
	// -------------------------------------------------------

	watchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	eventCh, err := reg.Watch(watchCtx, "user-service")
	if err != nil {
		log.Fatalf("watch: %v", err)
	}

	for ev := range eventCh {
		switch ev.Type {
		case registry.EventTypePut:
			fmt.Printf("🟢 UP: %s %s\n", ev.Instance.Name, ev.Instance.Address)
		case registry.EventTypeDelete:
			fmt.Printf("🔴 DOWN: %s\n", ev.Instance.ID)
		}
	}
}
