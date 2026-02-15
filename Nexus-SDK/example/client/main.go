package main

import (
	"fmt"
	"log"

	nexus "github.com/krustd/nexus-sdk"
	"github.com/krustd/nexus-sdk/registry"
	"github.com/krustd/nexus-sdk/registry/balancer"
)

func main() {
	nexus.MustSetup("config/config.toml")
	defer nexus.Shutdown()

	// Resolver 依赖的是 registry.Registry 接口
	// 底层换实现，这里零改动
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

	for i := 0; i < 10; i++ {
		inst, err := resolver.Resolve()
		if err != nil {
			log.Printf("resolve: %v", err)
			continue
		}
		fmt.Printf("[%d] → http://%s (weight=%d)\n", i, inst.Address, inst.Weight)
	}
}
