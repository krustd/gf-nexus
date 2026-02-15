package registry

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Registry 注册中心核心
type Registry struct {
	client *clientv3.Client
	config *Config

	mu         sync.RWMutex
	registered map[string]clientv3.LeaseID
}

// New 创建注册中心实例
func New(conf *Config) (*Registry, error) {
	if conf == nil {
		conf = DefaultConfig()
	}
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	etcdConf := clientv3.Config{
		Endpoints:   conf.Endpoints,
		DialTimeout: conf.DialTimeout(),
	}
	if conf.Username != "" {
		etcdConf.Username = conf.Username
		etcdConf.Password = conf.Password
	}

	client, err := clientv3.New(etcdConf)
	if err != nil {
		return nil, fmt.Errorf("nexus: connect etcd: %w", err)
	}

	// 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), conf.DialTimeout())
	defer cancel()
	if _, err = client.MemberList(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("nexus: etcd health check failed: %w", err)
	}

	return &Registry{
		client:     client,
		config:     conf,
		registered: make(map[string]clientv3.LeaseID),
	}, nil
}

// Register 注册服务实例（带租约自动续期）
func (r *Registry) Register(ctx context.Context, instance *ServiceInstance) error {
	if err := instance.Validate(); err != nil {
		return err
	}

	lease, err := r.client.Grant(ctx, r.config.LeaseTTL)
	if err != nil {
		return fmt.Errorf("nexus: grant lease: %w", err)
	}

	val, err := instance.Marshal()
	if err != nil {
		return err
	}

	key := instance.BuildKey(r.config.Prefix)
	_, err = r.client.Put(ctx, key, val, clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("nexus: put %s: %w", key, err)
	}

	ch, err := r.client.KeepAlive(ctx, lease.ID)
	if err != nil {
		return fmt.Errorf("nexus: keepalive: %w", err)
	}
	go func() {
		for resp := range ch {
			_ = resp
		}
		log.Printf("[nexus] keepalive channel closed: %s", key)
	}()

	r.mu.Lock()
	r.registered[key] = lease.ID
	r.mu.Unlock()

	log.Printf("[nexus] registered: %s → %s (%s)", key, instance.Address, instance.Protocol)
	return nil
}

// Deregister 注销服务实例
func (r *Registry) Deregister(ctx context.Context, instance *ServiceInstance) error {
	key := instance.BuildKey(r.config.Prefix)

	r.mu.Lock()
	leaseID, ok := r.registered[key]
	if ok {
		delete(r.registered, key)
	}
	r.mu.Unlock()

	if ok {
		if _, err := r.client.Revoke(ctx, leaseID); err != nil {
			return fmt.Errorf("nexus: revoke %s: %w", key, err)
		}
	} else {
		if _, err := r.client.Delete(ctx, key); err != nil {
			return fmt.Errorf("nexus: delete %s: %w", key, err)
		}
	}

	log.Printf("[nexus] deregistered: %s", key)
	return nil
}

// Discover 获取某个服务的所有实例
func (r *Registry) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	prefix := ServicePrefix(r.config.Prefix, serviceName)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("nexus: discover %s: %w", serviceName, err)
	}

	instances := make([]*ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		inst, err := UnmarshalInstance(kv.Value)
		if err != nil {
			log.Printf("[nexus] skip bad instance %s: %v", string(kv.Key), err)
			continue
		}
		instances = append(instances, inst)
	}
	return instances, nil
}

// DiscoverByProtocol 按协议过滤
func (r *Registry) DiscoverByProtocol(ctx context.Context, serviceName string, protocol Protocol) ([]*ServiceInstance, error) {
	all, err := r.Discover(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	filtered := make([]*ServiceInstance, 0)
	for _, inst := range all {
		if inst.Protocol == protocol {
			filtered = append(filtered, inst)
		}
	}
	return filtered, nil
}

// Watch 监听某个服务的变更
func (r *Registry) Watch(ctx context.Context, serviceName string) (<-chan WatchEvent, error) {
	prefix := ServicePrefix(r.config.Prefix, serviceName)
	eventCh := make(chan WatchEvent, 64)
	watchCh := r.client.Watch(ctx, prefix, clientv3.WithPrefix())

	go func() {
		defer close(eventCh)
		for {
			select {
			case <-ctx.Done():
				return
			case resp, ok := <-watchCh:
				if !ok {
					return
				}
				for _, ev := range resp.Events {
					var event WatchEvent
					switch ev.Type {
					case clientv3.EventTypePut:
						event.Type = EventTypePut
						inst, err := UnmarshalInstance(ev.Kv.Value)
						if err != nil {
							continue
						}
						event.Instance = inst
					case clientv3.EventTypeDelete:
						event.Type = EventTypeDelete
						event.Instance = &ServiceInstance{ID: string(ev.Kv.Key)}
					}
					select {
					case eventCh <- event:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return eventCh, nil
}

// Close 关闭，注销所有实例
func (r *Registry) Close(ctx context.Context) error {
	r.mu.RLock()
	leases := make(map[string]clientv3.LeaseID, len(r.registered))
	for k, v := range r.registered {
		leases[k] = v
	}
	r.mu.RUnlock()

	for key, leaseID := range leases {
		if _, err := r.client.Revoke(ctx, leaseID); err != nil {
			log.Printf("[nexus] revoke %s failed: %v", key, err)
		}
	}

	r.mu.Lock()
	r.registered = make(map[string]clientv3.LeaseID)
	r.mu.Unlock()

	return r.client.Close()
}

// Client 暴露底层 etcd client
func (r *Registry) Client() *clientv3.Client {
	return r.client
}

// GetConfig 获取配置
func (r *Registry) GetConfig() *Config {
	return r.config
}

// ---------- 全局单例 ----------

var (
	globalRegistry *Registry
	globalMu       sync.RWMutex
)

// SetGlobal 设置全局注册中心
func SetGlobal(r *Registry) {
	globalMu.Lock()
	globalRegistry = r
	globalMu.Unlock()
}

// GetGlobal 获取全局注册中心
func GetGlobal() *Registry {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalRegistry
}

// Shutdown 优雅关闭全局注册中心
func Shutdown() error {
	globalMu.Lock()
	reg := globalRegistry
	globalRegistry = nil
	globalMu.Unlock()
	if reg != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return reg.Close(ctx)
	}
	return nil
}
