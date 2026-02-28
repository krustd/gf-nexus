# Nexus-Gateway

基于 GoFrame 的微服务 API 网关，提供泛化调用、安全控制、韧性保护和可观测性。
通过 [Nexus-Config](../Nexus-Config) 配置中心实现运行时参数热更新，无需重启即可调整限流、熔断、JWT 密钥等。

## 架构

```
                         ┌─────────────────────────────────────────────┐
  Client Request         │              Nexus-Gateway                  │
  ─────────────────►     │                                             │
  POST /api/user-service │  Trace → RequestID → Logging → CORS        │
       /GetUserInfo      │  → IPFilter → RateLimit → JWT              │
                         │  → CircuitBreaker → ProxyHandler            │
                         │       │                                     │
                         │       ├─ ResolverPool (etcd 服务发现)        │
                         │       ├─ LoadBalancer (RR/Random/WRR)       │
                         │       └─ HTTP/gRPC 反向代理转发              │
                         └──────┬───────────────────┬─────────────────┘
                                │                   │
                    ┌───────────┴───────┐   ┌───────┴───────┐
                    ▼                   ▼   ▼               ▼
              user-service        order-service        Nexus-Config
              (多实例)             (多实例)             (动态配置)
```

## 核心功能

### 1. 泛化调用

通过通配符路由 `/api/:service/*method`，网关自动完成服务发现和转发：

```
GET  /api/user-service/v1/users?page=1
POST /api/order-service/v1/orders
```

- 从 URL 提取 `:service`，通过 etcd 查找可用实例
- 负载均衡选择实例（Round Robin / Random / Weighted Round Robin）
- 自动检测服务协议，HTTP 直接转发，gRPC 服务自动 JSON ↔ Protobuf 转码
- 透明转发请求头、Body、Query 参数

### 2. 动态配置热更新

网关配置分为两层：

| 层级 | 存储位置 | 更新方式 | 包含内容 |
| ---- | -------- | -------- | -------- |
| **静态配置** | 本地 `config.toml` | 重启生效 | etcd 连接、监听地址、配置中心连接 |
| **动态配置** | Nexus-Config 配置中心 | 秒级热更新 | JWT、限流、熔断、CORS、IP 黑白名单、负载均衡策略 |

通过配置中心 Admin API 修改并发布配置后，网关自动感知变更并即时生效。

### 3. 安全与访问控制

| 功能 | 说明 |
| ---- | ---- |
| **JWT 鉴权** | 非对称加密（RS256 / EdDSA），JWKS 多密钥，支持平滑密钥轮换 |
| **身份透传** | 校验通过后注入 `X-User-Id`、`X-User-Role` 到下游请求头 |
| **IP 黑白名单** | 支持精确 IP 和 CIDR 网段匹配，实时生效 |
| **CORS** | 统一处理跨域，OPTIONS 预检直接返回 204 |

### 4. 韧性与稳定性

| 功能 | 说明 |
| ---- | ---- |
| **限流** | 令牌桶算法，可动态调整 QPS 和突发容量 |
| **熔断** | 按服务名独立熔断，滑动窗口统计错误率，closed → open → half-open |
| **超时控制** | 连接超时 + 响应超时，独立配置 |

### 5. 可观测性

| 功能 | 说明 |
| ---- | ---- |
| **Trace** | 生成/传递 `X-Trace-Id`，注入 Context |
| **Request ID** | 生成/传递 `X-Request-Id` |
| **请求日志** | 记录 method、path、status、latency、client_ip、trace_id |
| **Prometheus 指标** | `gateway_requests_total`、`gateway_request_duration_seconds`、`gateway_circuit_breaker_state` |

## 快速开始

### 1. 本地配置 (config.toml)

```toml
[gateway]

[gateway.registry]
endpoints    = ["127.0.0.1:2379"]
dial_timeout = 5
prefix       = "/nexus/services"

[gateway.server]
addr = ":8080"

[gateway.config_center]
server_addr  = "http://127.0.0.1:8082"   # Nexus-Config 分发服务地址
namespace    = "nexus-gateway"
config_key   = "gateway.yaml"
poll_timeout = 30
retry_delay  = 5

[gateway.timeout]
connect_ms  = 3000
response_ms = 10000

[gateway.metrics]
enabled = true
path    = "/metrics"
```

### 2. 动态配置 (gateway.yaml)

通过配置中心 Admin API 发布，示例见 [example/gateway.yaml](example/gateway.yaml)。

```yaml
jwt:
  enabled: true
  keys:
    - kid: "key-2024-01"
      algorithm: RS256
      public_key: |
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
        -----END PUBLIC KEY-----
  skip_paths: [/health, /metrics]

rate_limit:
  enabled: true
  rate: 1000.0
  burst: 2000

circuit:
  enabled: true
  error_threshold: 0.5
  min_requests: 20
  window_sec: 30
  cooldown_sec: 15

cors:
  enabled: true
  allow_origins: ["*"]

ip_filter:
  enabled: false
  mode: blacklist
  addresses: []

balancer:
  strategy: round_robin
```

### 3. 启动网关

```go
package main

import (
    "os"
    "os/signal"
    "syscall"

    nexus "github.com/krustd/nexus-gateway"
)

func main() {
    nexus.MustSetup("config.toml")

    go func() {
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        <-quit
        nexus.Shutdown()
        os.Exit(0)
    }()

    nexus.Start()
}
```

### 4. 请求示例

```bash
# 通过网关访问 user-service
curl -H "Authorization: Bearer <token>" \
     http://localhost:8080/api/user-service/v1/users?page=1

# 健康检查
curl http://localhost:8080/health

# Prometheus 指标
curl http://localhost:8080/metrics
```

## 配置中心下发

通过 Nexus-Config Admin API 管理动态配置：

```bash
# 1. 创建命名空间
curl -X POST http://127.0.0.1:8081/api/v1/namespaces/ \
  -H "Content-Type: application/json" \
  -d '{"id":"nexus-gateway","name":"Nexus Gateway"}'

# 2. 保存草稿（将 gateway.yaml 内容作为 value）
curl -X POST http://127.0.0.1:8081/api/v1/configs/draft \
  -H "Content-Type: application/json" \
  -d '{"namespace":"nexus-gateway","key":"gateway.yaml","format":"yaml","value":"..."}'

# 3. 发布（网关秒级生效，无需重启）
curl -X POST http://127.0.0.1:8081/api/v1/configs/publish \
  -H "Content-Type: application/json" \
  -d '{"namespace":"nexus-gateway","key":"gateway.yaml"}'
```

后续修改配置只需重复步骤 2、3，网关自动感知变更。

## JWT 密钥轮换

网关使用非对称加密（RS256 / EdDSA）验证 JWT，通过 JWKS 多密钥机制实现平滑轮换：

```
1. 生成新密钥对，分配新 kid（如 key-2024-02）
2. 在配置中心 keys 数组中添加新公钥（保留旧 key）→ 发布 → 网关热加载
3. 签发方切换到新私钥签发 token
4. 等待旧 token 自然过期（根据 token TTL）
5. 从配置中心移除旧 kid → 发布 → 网关热加载
```

整个过程零停机，新旧 token 在过渡期内均可正常验证。

## 中间件执行顺序

```
请求 → Trace → RequestID → Logging(计时) → CORS → IPFilter → RateLimit → JWT
     → [路由匹配]
     → CircuitBreaker(按 service) → ProxyHandler(服务发现 + 转发)
     → Logging(记录日志 + 指标) → 响应
```

## 统一错误响应

```json
{
    "code": 1001,
    "message": "missing authorization token",
    "trace_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| 错误码 | 含义 | HTTP 状态码 |
| ------ | ---- | ----------- |
| 1001 | JWT 校验失败 | 401 |
| 1002 | IP 被拦截 | 403 |
| 1003 | 限流 | 429 |
| 1004 | 熔断 | 503 |
| 1005 | 服务未找到 | 502 |
| 1006 | 后端超时 | 504 |
| 1007 | 后端错误 | 502 |

## 目录结构

```
Nexus-Gateway/
├── gateway.go                  # 顶层入口: Setup / Start / Shutdown + 配置中心集成
├── internal/
│   └── response.go             # 共享类型: 错误码、统一响应、trace_id context
├── config/
│   ├── config.go               # 静态配置（TOML）+ 动态配置结构体定义
│   └── dynamic.go              # DynamicConfigHolder（atomic.Pointer 热更新）
├── gateway/
│   ├── gateway.go              # 核心: 中间件链组装 + 路由绑定 + 启动
│   ├── proxy.go                # HTTP 反向代理
│   ├── grpc_proxy.go           # HTTP→gRPC 转码代理
│   ├── response.go             # 响应工具 re-export
│   └── resolver_pool.go        # 按服务名懒加载 Resolver + 策略热更新
├── middleware/
│   ├── trace.go                # Trace ID
│   ├── requestid.go            # Request ID
│   ├── logging.go              # 请求日志 + 指标
│   ├── cors.go                 # CORS（动态配置）
│   ├── jwt.go                  # JWT 鉴权: JWKS 多密钥 + RS256/EdDSA
│   ├── ipfilter.go             # IP 黑白名单（动态配置）
│   ├── ratelimit.go            # 令牌桶限流（动态配置）
│   └── circuitbreaker.go       # 按服务熔断（动态配置）
├── metrics/
│   └── metrics.go              # Prometheus 指标
└── example/
    ├── main.go                 # 示例启动
    ├── config.toml             # 本地配置示例
    └── gateway.yaml            # 配置中心动态配置示例
```

## 依赖

- [Nexus-Registry](../Nexus-Registry) - etcd 服务注册发现 + 负载均衡
- [Nexus-Config](../Nexus-Config) - 配置中心 SDK（长轮询 + 本地缓存）
- [GoFrame v2](https://github.com/gogf/gf) - HTTP 服务框架
- [golang-jwt](https://github.com/golang-jwt/jwt) - JWT 解析（RS256 / EdDSA）
- [Prometheus Client](https://github.com/prometheus/client_golang) - 指标采集
