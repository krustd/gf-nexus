package etcd

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/krustd/nexus-registry/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdRegistry etcd 实现
type EtcdRegistry struct {
	client *clientv3.Client
	config *registry.Config

	mu         sync.RWMutex
	registered map[string]clientv3.LeaseID
}

// 编译期检查：确保实现了 Registry 接口
var _ registry.Registry = (*EtcdRegistry)(nil)

func New(conf *registry.Config) (*EtcdRegistry, error) {
	if conf == nil {
		conf = registry.DefaultConfig()
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
		return nil, fmt.Errorf("nexus-etcd: connect: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), conf.DialTimeout())
	defer cancel()
	if _, err = client.MemberList(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("nexus-etcd: health check failed: %w", err)
	}

	return &EtcdRegistry{
		client:     client,
		config:     conf,
		registered: make(map[string]clientv3.LeaseID),
	}, nil
}

func (r *EtcdRegistry) Register(ctx context.Context, instance *registry.ServiceInstance) error {
	if err := instance.Validate(); err != nil {
		return err
	}

	lease, err := r.client.Grant(ctx, r.config.LeaseTTL)
	if err != nil {
		return fmt.Errorf("nexus-etcd: grant lease: %w", err)
	}

	val, err := instance.Marshal()
	if err != nil {
		return err
	}

	key := instance.BuildKey(r.config.Prefix)
	_, err = r.client.Put(ctx, key, val, clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("nexus-etcd: put %s: %w", key, err)
	}

	ch, err := r.client.KeepAlive(ctx, lease.ID)
	if err != nil {
		return fmt.Errorf("nexus-etcd: keepalive: %w", err)
	}
	go func() {
		for resp := range ch {
			_ = resp
		}
		log.Printf("[nexus-etcd] keepalive closed: %s", key)
	}()

	r.mu.Lock()
	r.registered[key] = lease.ID
	r.mu.Unlock()

	log.Printf("[nexus-etcd] registered: %s → %s (%s)", key, instance.Address, instance.Protocol)
	return nil
}

func (r *EtcdRegistry) Deregister(ctx context.Context, instance *registry.ServiceInstance) error {
	key := instance.BuildKey(r.config.Prefix)

	r.mu.Lock()
	leaseID, ok := r.registered[key]
	if ok {
		delete(r.registered, key)
	}
	r.mu.Unlock()

	if ok {
		if _, err := r.client.Revoke(ctx, leaseID); err != nil {
			return fmt.Errorf("nexus-etcd: revoke %s: %w", key, err)
		}
	} else {
		if _, err := r.client.Delete(ctx, key); err != nil {
			return fmt.Errorf("nexus-etcd: delete %s: %w", key, err)
		}
	}

	log.Printf("[nexus-etcd] deregistered: %s", key)
	return nil
}

func (r *EtcdRegistry) Discover(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	prefix := registry.ServicePrefix(r.config.Prefix, serviceName)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("nexus-etcd: discover %s: %w", serviceName, err)
	}

	instances := make([]*registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		inst, err := registry.UnmarshalInstance(kv.Value)
		if err != nil {
			log.Printf("[nexus-etcd] skip bad key %s: %v", string(kv.Key), err)
			continue
		}
		instances = append(instances, inst)
	}
	return instances, nil
}

func (r *EtcdRegistry) DiscoverByProtocol(ctx context.Context, serviceName string, protocol registry.Protocol) ([]*registry.ServiceInstance, error) {
	all, err := r.Discover(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	filtered := make([]*registry.ServiceInstance, 0)
	for _, inst := range all {
		if inst.Protocol == protocol {
			filtered = append(filtered, inst)
		}
	}
	return filtered, nil
}

func (r *EtcdRegistry) Watch(ctx context.Context, serviceName string) (<-chan registry.WatchEvent, error) {
	prefix := registry.ServicePrefix(r.config.Prefix, serviceName)
	eventCh := make(chan registry.WatchEvent, 64)
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
					var event registry.WatchEvent
					switch ev.Type {
					case clientv3.EventTypePut:
						event.Type = registry.EventTypePut
						inst, err := registry.UnmarshalInstance(ev.Kv.Value)
						if err != nil {
							continue
						}
						event.Instance = inst
					case clientv3.EventTypeDelete:
						event.Type = registry.EventTypeDelete
						event.Instance = &registry.ServiceInstance{ID: string(ev.Kv.Key)}
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

func (r *EtcdRegistry) Close(ctx context.Context) error {
	r.mu.RLock()
	leases := make(map[string]clientv3.LeaseID, len(r.registered))
	for k, v := range r.registered {
		leases[k] = v
	}
	r.mu.RUnlock()

	for key, leaseID := range leases {
		if _, err := r.client.Revoke(ctx, leaseID); err != nil {
			log.Printf("[nexus-etcd] revoke %s failed: %v", key, err)
		}
	}

	r.mu.Lock()
	r.registered = make(map[string]clientv3.LeaseID)
	r.mu.Unlock()

	return r.client.Close()
}
