package main

import (
	"fmt"
	"log"

	nexus "github.com/krustd/nexus-sdk"
	"github.com/krustd/nexus-sdk/registry"
	"github.com/krustd/nexus-sdk/registry/balancer"
)

func main() {
	// 客户端也需要连 etcd（只用 registry 部分，service 段可以不写）
	nexus.MustSetup("config/config.toml")
	defer nexus.Shutdown()

	// -------------------------------------------------------
	// 方式 1: 简单发现（一次性）
	// -------------------------------------------------------
	instances, err := nexus.DiscoverHTTP("user-service")
	if err != nil {
		log.Fatal(err)
	}
	for _, inst := range instances {
		fmt.Printf("found: %s %s (weight=%d)\n", inst.Address, inst.Protocol, inst.Weight)
	}

	// -------------------------------------------------------
	// 方式 2: Resolver（推荐）—— 自动 Watch + 缓存 + 负载均衡
	// -------------------------------------------------------
	resolver, err := registry.NewResolver(
		nexus.GetRegistry(),
		"user-service",
		registry.WithProtocol(registry.ProtocolHTTP),
		registry.WithPicker(balancer.NewWeightedRoundRobin()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer resolver.Close()

	// 模拟多次调用
	for i := 0; i < 10; i++ {
		inst, err := resolver.Resolve()
		if err != nil {
			log.Printf("resolve error: %v", err)
			continue
		}
		// 拿到地址后发 HTTP 请求
		url := fmt.Sprintf("http://%s/", inst.Address)
		fmt.Printf("[%d] → %s (weight=%d)\n", i, url, inst.Weight)
	}

	// -------------------------------------------------------
	// 方式 3: 同时注册 HTTP + gRPC 的场景
	// -------------------------------------------------------
	// nexus.MustSetupMulti("config/config.toml",
	//     &registry.ServiceInstance{
	//         Name: "order-service", Protocol: registry.ProtocolHTTP,
	//         Address: "10.0.0.2:8080", Weight: 10,
	//     },
	//     &registry.ServiceInstance{
	//         Name: "order-service", Protocol: registry.ProtocolGRPC,
	//         Address: "10.0.0.2:9090", Weight: 10,
	//     },
	// )
}
