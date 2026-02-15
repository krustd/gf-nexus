package registry

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Picker 负载均衡选择器接口（与 balancer 包解耦）
type Picker interface {
	Pick(instances []*ServiceInstance) (*ServiceInstance, error)
}

// Resolver 服务解析器
// 维护一份本地缓存的实例列表，后台 Watch 自动更新
// 业务方通过 Resolve() 获取一个实例
type Resolver struct {
	registry    *Registry
	serviceName string
	protocol    Protocol // 过滤协议，空则不过滤
	picker      Picker

	mu        sync.RWMutex
	instances []*ServiceInstance

	cancel context.CancelFunc
}

// ResolverOption Resolver 配置选项
type ResolverOption func(*Resolver)

// WithProtocol 按协议过滤
func WithProtocol(p Protocol) ResolverOption {
	return func(r *Resolver) {
		r.protocol = p
	}
}

// WithPicker 设置负载均衡策略
func WithPicker(p Picker) ResolverOption {
	return func(r *Resolver) {
		r.picker = p
	}
}

// NewResolver 创建 Resolver 并启动后台 Watch
func NewResolver(reg *Registry, serviceName string, opts ...ResolverOption) (*Resolver, error) {
	r := &Resolver{
		registry:    reg,
		serviceName: serviceName,
	}
	for _, opt := range opts {
		opt(r)
	}
	if r.picker == nil {
		return nil, fmt.Errorf("nexus-registry: resolver requires a picker (load balancer)")
	}

	// 初始拉取
	ctx := context.Background()
	instances, err := r.fetchInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("nexus-registry: initial discover: %w", err)
	}
	r.instances = instances

	// 启动后台 Watch
	watchCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	eventCh, err := reg.Watch(watchCtx, serviceName)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("nexus-registry: watch: %w", err)
	}

	go r.watchLoop(watchCtx, eventCh)

	log.Printf("[nexus-registry] resolver started for %s, initial instances: %d", serviceName, len(instances))
	return r, nil
}

// Resolve 获取一个服务实例（经过负载均衡）
func (r *Resolver) Resolve() (*ServiceInstance, error) {
	r.mu.RLock()
	instances := r.instances
	r.mu.RUnlock()

	if len(instances) == 0 {
		return nil, fmt.Errorf("nexus-registry: no available instance for %s", r.serviceName)
	}

	return r.picker.Pick(instances)
}

// GetInstances 获取所有缓存的实例（不经过负载均衡）
func (r *Resolver) GetInstances() []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cp := make([]*ServiceInstance, len(r.instances))
	copy(cp, r.instances)
	return cp
}

// Close 停止 Watch，释放资源
func (r *Resolver) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}

// watchLoop 后台消费 Watch 事件，增量更新本地缓存
func (r *Resolver) watchLoop(ctx context.Context, eventCh <-chan WatchEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-eventCh:
			if !ok {
				// channel 关闭，尝试全量刷新一次
				log.Printf("[nexus-registry] watch channel closed for %s, doing full refresh", r.serviceName)
				r.fullRefresh()
				return
			}
			r.handleEvent(ev)
		}
	}
}

// handleEvent 处理单个事件
func (r *Resolver) handleEvent(ev WatchEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch ev.Type {
	case EventTypePut:
		if ev.Instance == nil {
			return
		}
		// 过滤协议
		if r.protocol != "" && ev.Instance.Protocol != r.protocol {
			return
		}
		// 更新或新增
		found := false
		for i, inst := range r.instances {
			if inst.ID == ev.Instance.ID {
				r.instances[i] = ev.Instance
				found = true
				break
			}
		}
		if !found {
			r.instances = append(r.instances, ev.Instance)
		}
		log.Printf("[nexus-registry] instance updated: %s %s", ev.Instance.Name, ev.Instance.Address)

	case EventTypeDelete:
		if ev.Instance == nil {
			return
		}
		// 按 key (ID) 移除
		for i, inst := range r.instances {
			// Watch 删除事件中 ID 是完整 key，也可能匹配 Address
			if inst.ID == ev.Instance.ID || inst.BuildKey(r.registry.config.Prefix) == ev.Instance.ID {
				r.instances = append(r.instances[:i], r.instances[i+1:]...)
				log.Printf("[nexus-registry] instance removed: %s", ev.Instance.ID)
				break
			}
		}
	}
}

// fullRefresh 全量刷新实例列表
func (r *Resolver) fullRefresh() {
	ctx := context.Background()
	instances, err := r.fetchInstances(ctx)
	if err != nil {
		log.Printf("[nexus-registry] full refresh failed for %s: %v", r.serviceName, err)
		return
	}

	r.mu.Lock()
	r.instances = instances
	r.mu.Unlock()

	log.Printf("[nexus-registry] full refresh done for %s, instances: %d", r.serviceName, len(instances))
}

// fetchInstances 拉取实例并按协议过滤
func (r *Resolver) fetchInstances(ctx context.Context) ([]*ServiceInstance, error) {
	if r.protocol != "" {
		return r.registry.DiscoverByProtocol(ctx, r.serviceName, r.protocol)
	}
	return r.registry.Discover(ctx, r.serviceName)
}
