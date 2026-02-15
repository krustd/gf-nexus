package registry

import "context"

// Registry 注册中心接口
// 业务方面向此接口，底层实现可替换（etcd / consul / nacos）
type Registry interface {
	Register(ctx context.Context, instance *ServiceInstance) error
	Deregister(ctx context.Context, instance *ServiceInstance) error
	Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	DiscoverByProtocol(ctx context.Context, serviceName string, protocol Protocol) ([]*ServiceInstance, error)
	Watch(ctx context.Context, serviceName string) (<-chan WatchEvent, error)
	Close(ctx context.Context) error
}
