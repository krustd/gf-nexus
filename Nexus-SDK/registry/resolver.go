package registry

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type Picker interface {
	Pick(instances []*ServiceInstance) (*ServiceInstance, error)
}

type Resolver struct {
	registry    Registry // ← 接口
	serviceName string
	protocol    Protocol
	picker      Picker
	prefix      string

	mu        sync.RWMutex
	instances []*ServiceInstance
	cancel    context.CancelFunc
}

type ResolverOption func(*Resolver)

func WithProtocol(p Protocol) ResolverOption {
	return func(r *Resolver) { r.protocol = p }
}

func WithPicker(p Picker) ResolverOption {
	return func(r *Resolver) { r.picker = p }
}

func WithPrefix(prefix string) ResolverOption {
	return func(r *Resolver) { r.prefix = prefix }
}

// NewResolver 参数是 Registry 接口，不是具体 struct
func NewResolver(reg Registry, serviceName string, opts ...ResolverOption) (*Resolver, error) {
	r := &Resolver{
		registry:    reg,
		serviceName: serviceName,
		prefix:      "/nexus/services",
	}
	for _, opt := range opts {
		opt(r)
	}
	if r.picker == nil {
		return nil, fmt.Errorf("nexus: resolver requires a picker")
	}

	instances, err := r.fetchInstances(context.Background())
	if err != nil {
		return nil, fmt.Errorf("nexus: initial discover: %w", err)
	}
	r.instances = instances

	watchCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	eventCh, err := reg.Watch(watchCtx, serviceName)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("nexus: watch: %w", err)
	}
	go r.watchLoop(watchCtx, eventCh)

	log.Printf("[nexus] resolver started: %s (%d instances)", serviceName, len(instances))
	return r, nil
}

func (r *Resolver) Resolve() (*ServiceInstance, error) {
	r.mu.RLock()
	instances := r.instances
	r.mu.RUnlock()
	if len(instances) == 0 {
		return nil, fmt.Errorf("nexus: no instance for %s", r.serviceName)
	}
	return r.picker.Pick(instances)
}

func (r *Resolver) GetInstances() []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make([]*ServiceInstance, len(r.instances))
	copy(cp, r.instances)
	return cp
}

func (r *Resolver) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Resolver) watchLoop(ctx context.Context, eventCh <-chan WatchEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-eventCh:
			if !ok {
				r.fullRefresh()
				return
			}
			r.handleEvent(ev)
		}
	}
}

func (r *Resolver) handleEvent(ev WatchEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch ev.Type {
	case EventTypePut:
		if ev.Instance == nil {
			return
		}
		if r.protocol != "" && ev.Instance.Protocol != r.protocol {
			return
		}
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

	case EventTypeDelete:
		if ev.Instance == nil {
			return
		}
		for i, inst := range r.instances {
			if inst.ID == ev.Instance.ID ||
				inst.BuildKey(r.prefix) == ev.Instance.ID {
				r.instances = append(r.instances[:i], r.instances[i+1:]...)
				break
			}
		}
	}
}

func (r *Resolver) fullRefresh() {
	instances, err := r.fetchInstances(context.Background())
	if err != nil {
		log.Printf("[nexus] refresh failed: %s: %v", r.serviceName, err)
		return
	}
	r.mu.Lock()
	r.instances = instances
	r.mu.Unlock()
}

func (r *Resolver) fetchInstances(ctx context.Context) ([]*ServiceInstance, error) {
	if r.protocol != "" {
		return r.registry.DiscoverByProtocol(ctx, r.serviceName, r.protocol)
	}
	return r.registry.Discover(ctx, r.serviceName)
}
