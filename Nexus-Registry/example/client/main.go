package main

import (
	"fmt"
	"log"

	nexus "github.com/krustd/nexus-registry"
	"github.com/krustd/nexus-registry/registry"
	"github.com/krustd/nexus-registry/registry/balancer"
)

func main() {
	nexus.MustSetup("config/config.toml")
	defer nexus.Shutdown()

	// nexus.GetRegistry() 返回的是 registry.Registry 接口
	// 直接传给 NewResolver 即可
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
