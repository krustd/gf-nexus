# Nexus SDK

基于 etcd 官方客户端的微服务注册发现 SDK。

## 特性

- **一行注册**：`nexus.MustSetup("config.toml")`，配置全在 TOML 文件里
- **直连 etcd**：不依赖 gf 的 registry 封装，使用 `go.etcd.io/etcd/client/v3` 官方客户端
- **HTTP + gRPC 双协议**：同一个服务名下可以注册不同协议的实例
- **三种负载均衡**：Round Robin / Random / 加权轮询（Nginx 平滑算法）
- **实时感知**：Watch 机制 + 本地缓存，服务上下线秒级感知
- **优雅退出**：`defer nexus.Shutdown()` 自动反注册 + 关闭连接

## 快速开始

### 1. 配置文件

```toml
# config/config.toml

[nexus.registry]
endpoints    = ["127.0.0.1:2379"]
dial_timeout = 5
lease_ttl    = 15
prefix       = "/nexus/services"

[nexus.service]
name     = "user-service"
version  = "v1.0.0"
protocol = "http"
address  = "10.0.0.1:8080"
weight   = 10

[nexus.service.metadata]
region = "ap-northeast-1"
env    = "production"
```

### 2. 服务端（注册）

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/net/ghttp"
    nexus "github.com/krustd/nexus-registry"
)

func main() {
    nexus.MustSetup("config/config.toml")
    defer nexus.Shutdown()

    s := g.Server()
    s.BindHandler("/", func(r *ghttp.Request) {
        r.Response.Write("Hello")
    })
    s.SetPort(8080)
    s.Run()
}
```

### 3. 客户端（发现 + 负载均衡）

```go
resolver, _ := registry.NewResolver(
    nexus.GetRegistry(),
    "user-service",
    registry.WithProtocol(registry.ProtocolHTTP),
    registry.WithPicker(balancer.NewWeightedRoundRobin()),
)
defer resolver.Close()

inst, _ := resolver.Resolve()
url := fmt.Sprintf("http://%s/api/user", inst.Address)
```

## 项目结构

```
nexus-sdk/
├── nexus.go                    # 顶层入口（Setup / Shutdown / Discover）
├── registry/
│   ├── config.go               # 配置定义 + TOML 加载
│   ├── types.go                # ServiceInstance 定义
│   ├── registry.go             # 核心：Register / Discover / Watch / Close
│   ├── resolver.go             # Resolver：缓存 + Watch + 负载均衡
│   └── balancer/
│       └── balancer.go         # RoundRobin / Random / WeightedRoundRobin
└── example/
    ├── config.toml             # 示例配置
    ├── server/main.go          # 服务端示例
    └── client/main.go          # 客户端示例
```

## etcd 数据结构

```
/nexus/services/{服务名}/{实例ID} → JSON

例：
/nexus/services/user-service/user-service-10.0.0.1:8080
→ {"id":"user-service-10.0.0.1:8080","name":"user-service","protocol":"http","address":"10.0.0.1:8080","weight":10}
```
