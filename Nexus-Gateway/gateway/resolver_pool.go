package gateway

import (
	"fmt"
	"sync"

	"github.com/krustd/nexus-registry/registry"
)

// ResolverPool 按服务名懒加载并缓存 Resolver
type ResolverPool struct {
	mu        sync.RWMutex
	resolvers map[string]*registry.Resolver
	reg       registry.Registry
	picker    func() registry.Picker // 工厂函数，每个 Resolver 独立 Picker
}

func NewResolverPool(reg registry.Registry, pickerFactory func() registry.Picker) *ResolverPool {
	return &ResolverPool{
		resolvers: make(map[string]*registry.Resolver),
		reg:       reg,
		picker:    pickerFactory,
	}
}

// GetOrCreate 返回已有的 Resolver，或为该服务创建新的（含 Watch）
func (p *ResolverPool) GetOrCreate(serviceName string) (*registry.Resolver, error) {
	p.mu.RLock()
	r, ok := p.resolvers[serviceName]
	p.mu.RUnlock()
	if ok {
		return r, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// double check
	if r, ok := p.resolvers[serviceName]; ok {
		return r, nil
	}

	r, err := registry.NewResolver(
		p.reg,
		serviceName,
		registry.WithPicker(p.picker()),
	)
	if err != nil {
		return nil, fmt.Errorf("nexus-gateway: create resolver for %s: %w", serviceName, err)
	}

	p.resolvers[serviceName] = r
	return r, nil
}

// Close 关闭所有 Resolver
func (p *ResolverPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range p.resolvers {
		r.Close()
	}
	p.resolvers = make(map[string]*registry.Resolver)
}
