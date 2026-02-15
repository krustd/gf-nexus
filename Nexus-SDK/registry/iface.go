package registry

import "context"

// Registry 注册中心接口
// 业务方面向这个接口编程，底层可以是 etcd / consul / nacos / zookeeper
type Registry interface {
	// Register 注册服务实例（带自动续期）
	Register(ctx context.Context, instance *ServiceInstance) error

	// Deregister 注销服务实例
	Deregister(ctx context.Context, instance *ServiceInstance) error

	// Discover 发现某个服务的所有实例
	Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error)

	// DiscoverByProtocol 按协议过滤发现
	DiscoverByProtocol(ctx context.Context, serviceName string, protocol Protocol) ([]*ServiceInstance, error)

	// Watch 监听服务变更
	Watch(ctx context.Context, serviceName string) (<-chan WatchEvent, error)

	// Close 关闭连接，注销所有本地实例
	Close(ctx context.Context) error
}
