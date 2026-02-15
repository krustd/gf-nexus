package registry

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EventType Watch 事件类型
type EventType int

const (
	EventTypePut    EventType = iota // 新增或更新
	EventTypeDelete                  // 删除（下线）
)

// WatchEvent 服务变更事件
type WatchEvent struct {
	Type     EventType
	Instance *ServiceInstance
}

// Registry 注册中心核心结构
type Registry struct {
	client *clientv3.Client
	config *Config

	// 已注册的本地实例 → leaseID 映射，用于反注册
	mu         sync.RWMutex
	registered map[string]clientv3.LeaseID // key: instance.BuildKey()
}

// New 创建注册中心
func New(conf *Config) (*Registry, error) {
	if conf == nil {
		conf = DefaultConfig()
	}
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	etcdConf := clientv3.Config{
		Endpoints:   conf.Endpoints,
		DialTimeout: conf.DialTimeout,
	}
	if conf.Username != "" {
		etcdConf.Username = conf.Username
		etcdConf.Password = conf.Password
	}

	client, err := clientv3.New(etcdConf)
	if err != nil {
		return nil, fmt.Errorf("nexus-registry: connect to etcd: %w", err)
	}

	// 健康检查：尝试连一下
	ctx, cancel := context.WithTimeout(context.Background(), conf.DialTimeout)
	defer cancel()
	_, err = client.MemberList(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("nexus-registry: etcd health check failed: %w", err)
	}

	return &Registry{
		client:     client,
		config:     conf,
		registered: make(map[string]clientv3.LeaseID),
	}, nil
}

// ---------- 服务注册 ----------

// Register 注册服务实例（带租约自动续期）
func (r *Registry) Register(ctx context.Context, instance *ServiceInstance) error {
	if err := instance.Validate(); err != nil {
		return err
	}

	// 1. 创建租约
	lease, err := r.client.Grant(ctx, r.config.LeaseTTL)
	if err != nil {
		return fmt.Errorf("nexus-registry: create lease: %w", err)
	}

	// 2. 序列化实例
	val, err := instance.Marshal()
	if err != nil {
		return err
	}

	// 3. 写入 etcd（绑定租约）
	key := instance.BuildKey(r.config.Prefix)
	_, err = r.client.Put(ctx, key, val, clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("nexus-registry: put key %s: %w", key, err)
	}

	// 4. 自动续租（后台 goroutine）
	ch, err := r.client.KeepAlive(ctx, lease.ID)
	if err != nil {
		return fmt.Errorf("nexus-registry: keepalive: %w", err)
	}

	// 消费 keepalive 响应，防止 channel 阻塞
	go func() {
		for {
			resp, ok := <-ch
			if !ok {
				log.Printf("[nexus-registry] keepalive channel closed for %s, instance may expire", key)
				return
			}
			_ = resp
		}
	}()

	// 5. 记录映射
	r.mu.Lock()
	r.registered[key] = lease.ID
	r.mu.Unlock()

	log.Printf("[nexus-registry] registered: %s → %s", key, instance.Address)
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

	// 撤销租约（会自动删除关联的 key）
	if ok {
		if _, err := r.client.Revoke(ctx, leaseID); err != nil {
			return fmt.Errorf("nexus-registry: revoke lease for %s: %w", key, err)
		}
	} else {
		// 兜底：直接删 key
		if _, err := r.client.Delete(ctx, key); err != nil {
			return fmt.Errorf("nexus-registry: delete key %s: %w", key, err)
		}
	}

	log.Printf("[nexus-registry] deregistered: %s", key)
	return nil
}

// ---------- 服务发现 ----------

// Discover 获取某个服务的所有实例（一次性查询）
func (r *Registry) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	prefix := ServicePrefix(r.config.Prefix, serviceName)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("nexus-registry: discover %s: %w", serviceName, err)
	}

	instances := make([]*ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		inst, err := UnmarshalInstance(kv.Value)
		if err != nil {
			log.Printf("[nexus-registry] skip invalid instance at key %s: %v", string(kv.Key), err)
			continue
		}
		instances = append(instances, inst)
	}
	return instances, nil
}

// DiscoverByProtocol 按协议过滤发现
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

// ---------- Watch 监听 ----------

// Watch 监听某个服务的变更事件（阻塞，通过 channel 推送事件）
// 调用方需要监听 ctx 取消来退出
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
							log.Printf("[nexus-registry] watch: skip invalid instance: %v", err)
							continue
						}
						event.Instance = inst

					case clientv3.EventTypeDelete:
						event.Type = EventTypeDelete
						// 删除事件的 Value 可能为空，从 Key 中提取信息
						event.Instance = &ServiceInstance{
							ID: string(ev.Kv.Key),
						}
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

// ---------- 生命周期 ----------

// Close 关闭注册中心，注销所有本地实例
func (r *Registry) Close(ctx context.Context) error {
	r.mu.RLock()
	keys := make([]string, 0, len(r.registered))
	leaseIDs := make([]clientv3.LeaseID, 0, len(r.registered))
	for k, v := range r.registered {
		keys = append(keys, k)
		leaseIDs = append(leaseIDs, v)
	}
	r.mu.RUnlock()

	// 逐个撤销租约
	for i, leaseID := range leaseIDs {
		if _, err := r.client.Revoke(ctx, leaseID); err != nil {
			log.Printf("[nexus-registry] revoke lease for %s failed: %v", keys[i], err)
		}
	}

	r.mu.Lock()
	r.registered = make(map[string]clientv3.LeaseID)
	r.mu.Unlock()

	return r.client.Close()
}

// Client 暴露底层 etcd client，方便高级用户自定义操作
func (r *Registry) Client() *clientv3.Client {
	return r.client
}

// ---------- 便捷全局函数 ----------

var (
	globalRegistry *Registry
	globalMu       sync.RWMutex
)

// Init 初始化全局注册中心
func Init(conf *Config) error {
	reg, err := New(conf)
	if err != nil {
		return err
	}
	globalMu.Lock()
	globalRegistry = reg
	globalMu.Unlock()
	return nil
}

// MustInit 初始化全局注册中心，失败 panic
func MustInit(conf *Config) {
	if err := Init(conf); err != nil {
		panic(fmt.Sprintf("nexus-registry: %v", err))
	}
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
