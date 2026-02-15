# Nexus-Registry

[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](LICENSE)

Nexus-Registry 是一个基于 etcd 的轻量级服务注册与发现组件，专为微服务架构设计。它提供了服务注册、服务发现、健康检查和负载均衡等核心功能，支持 HTTP 和 gRPC 协议。

## 特性

- 🚀 **轻量级**: 基于 etcd 的简单高效实现
- 🔄 **自动续租**: 内置健康检查，自动续租机制
- 📡 **实时监听**: 支持服务变更事件监听
- ⚖️ **负载均衡**: 内置多种负载均衡策略（轮询、随机、加权轮询）
- 🌐 **多协议**: 支持 HTTP 和 gRPC 协议
- 🛡️ **容错设计**: 完善的错误处理和资源清理机制
- 🎯 **易于集成**: 简洁的 API 设计，支持全局实例和局部实例

## 安装

```bash
go get github.com/krustd/nexus-registry
```

## 快速开始

### 1. 服务注册

```go
package main

import (
    "context"
    "log"
    "time"
    
    registry "github.com/krustd/nexus-registry"
)

func main() {
    // 初始化注册中心
    reg, err := registry.New(&registry.Config{
        Endpoints:   []string{"127.0.0.1:2379"},
        DialTimeout: 5 * time.Second,
        LeaseTTL:    15, // 15秒租约
        Prefix:      "/nexus/services",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer reg.Close(context.Background())
    
    // 定义服务实例
    instance := &registry.ServiceInstance{
        ID:       "user-service-1",
        Name:     "user-service",
        Version:  "v1.0.0",
        Protocol: registry.ProtocolHTTP,
        Address:  "10.0.0.1:8080",
        Weight:   10,
        Metadata: map[string]string{
            "region": "ap-northeast-1",
            "env":    "production",
        },
    }
    
    // 注册服务（自动续租）
    ctx := context.Background()
    if err := reg.Register(ctx, instance); err != nil {
        log.Fatal(err)
    }
    
    // 服务运行中...
}
```

### 2. 服务发现

```go
// 方式1: 简单发现
instances, err := reg.Discover(ctx, "user-service")
if err != nil {
    log.Fatal(err)
}
for _, inst := range instances {
    fmt.Printf("Found: %s %s\n", inst.Address, inst.Protocol)
}

// 方式2: 按协议过滤
grpcInstances, err := reg.DiscoverByProtocol(ctx, "user-service", registry.ProtocolGRPC)
if err != nil {
    log.Fatal(err)
}
```

### 3. 负载均衡与自动监听

```go
import "github.com/krustd/nexus-registry/balancer"

// 创建 Resolver（自动监听 + 本地缓存 + 负载均衡）
resolver, err := registry.NewResolver(reg, "user-service",
    registry.WithProtocol(registry.ProtocolHTTP),
    registry.WithPicker(balancer.NewRoundRobin()),
)
if err != nil {
    log.Fatal(err)
}
defer resolver.Close()

// 获取服务实例（自动负载均衡）
instance, err := resolver.Resolve()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Selected: %s\n", instance.Address)
```

## 核心概念

### ServiceInstance

服务实例是注册到 etcd 的最小单元，包含以下字段：

```go
type ServiceInstance struct {
    ID        string            `json:"id"`        // 实例唯一ID
    Name      string            `json:"name"`      // 服务名称
    Version   string            `json:"version"`   // 服务版本
    Protocol  Protocol          `json:"protocol"`  // 协议类型: http/grpc
    Address   string            `json:"address"`   // 监听地址
    Weight    int               `json:"weight"`    // 权重（用于负载均衡）
    Metadata  map[string]string `json:"metadata"`  // 扩展元数据
}
```

### 配置选项

```go
type Config struct {
    Endpoints   []string      // etcd 集群地址
    DialTimeout time.Duration // 连接超时
    LeaseTTL    int64         // 服务租约TTL（秒）
    Prefix      string        // etcd key 前缀
    Username    string        // etcd 认证用户名（可选）
    Password    string        // etcd 认证密码（可选）
}
```

## 负载均衡策略

Nexus-Registry 内置了多种负载均衡策略：

### 1. 轮询 (Round Robin)

```go
resolver, err := registry.NewResolver(reg, "user-service",
    registry.WithPicker(balancer.NewRoundRobin()),
)
```

### 2. 随机 (Random)

```go
resolver, err := registry.NewResolver(reg, "user-service",
    registry.WithPicker(balancer.NewRandom()),
)
```

### 3. 加权轮询 (Weighted Round Robin)

基于 Nginx 的平滑加权轮询算法，根据服务实例的权重分配请求：

```go
resolver, err := registry.NewResolver(reg, "user-service",
    registry.WithPicker(balancer.NewWeightedRoundRobin()),
)
```

## 高级用法

### 全局实例管理

```go
// 初始化全局注册中心
registry.MustInit(&registry.Config{
    Endpoints: []string{"127.0.0.1:2379"},
})

// 获取全局实例
reg := registry.GetGlobal()

// 优雅关闭
defer registry.Shutdown()
```

### 监听服务变更

```go
// 监听特定服务的变更事件
eventCh, err := reg.Watch(ctx, "user-service")
if err != nil {
    log.Fatal(err)
}

for event := range eventCh {
    switch event.Type {
    case registry.EventTypePut:
        fmt.Printf("服务上线: %s\n", event.Instance.Address)
    case registry.EventTypeDelete:
        fmt.Printf("服务下线: %s\n", event.Instance.ID)
    }
}
```

### 自定义负载均衡策略

实现 `Picker` 接口来自定义负载均衡策略：

```go
type CustomPicker struct{}

func (p *CustomPicker) Pick(instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
    // 自定义选择逻辑
    return instances[0], nil
}

// 使用自定义策略
resolver, err := registry.NewResolver(reg, "user-service",
    registry.WithPicker(&CustomPicker{}),
)
```

## API 参考

### Registry 接口

| 方法 | 描述 |
|------|------|
| `New(conf *Config) (*Registry, error)` | 创建注册中心实例 |
| `Register(ctx, instance) error` | 注册服务实例 |
| `Deregister(ctx, instance) error` | 注销服务实例 |
| `Discover(ctx, serviceName) ([]*ServiceInstance, error)` | 发现服务实例 |
| `DiscoverByProtocol(ctx, serviceName, protocol) ([]*ServiceInstance, error)` | 按协议发现服务 |
| `Watch(ctx, serviceName) (<-chan WatchEvent, error)` | 监听服务变更 |
| `Close(ctx) error` | 关闭注册中心 |

### Resolver 接口

| 方法 | 描述 |
|------|------|
| `NewResolver(reg, serviceName, opts...) (*Resolver, error)` | 创建解析器 |
| `Resolve() (*ServiceInstance, error)` | 获取一个服务实例 |
| `GetInstances() []*ServiceInstance` | 获取所有缓存的实例 |
| `Close()` | 关闭解析器 |

## 示例项目

完整示例代码请参考 [example/main.go](example/main.go)，包含：

- 服务注册示例
- 服务发现示例
- 负载均衡示例
- 事件监听示例

运行示例：

```bash
cd example
go run main.go
```

## 最佳实践

1. **优雅关闭**: 在应用退出时调用 `Close()` 或 `Shutdown()` 方法，确保资源正确释放
2. **错误处理**: 始终检查返回的错误，特别是网络相关操作
3. **超时控制**: 为所有上下文操作设置合理的超时时间
4. **租约TTL**: 根据业务需求调整租约TTL，通常建议为心跳间隔的3倍
5. **元数据使用**: 利用 Metadata 字段存储环境、区域等信息，便于服务管理和筛选

## 依赖

- Go 1.22+
- etcd v3.5.17+

## 许可证

GPL v3 License - 详见 [LICENSE](LICENSE) 文件

## 贡献

欢迎提交 Issue 和 Pull Request！

## 相关项目

- [Nexus-SDK](../Nexus-SDK) - 基于 Nexus-Registry 的完整 SDK 实现