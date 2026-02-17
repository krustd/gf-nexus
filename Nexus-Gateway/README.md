# Nexus-Gateway

基于 GoFrame 的微服务 API 网关，提供泛化调用、安全控制、韧性保护和可观测性。

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
                         │       └─ HTTP 反向代理转发                   │
                         └──────────────┬──────────────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
              user-service        order-service       pay-service
              (多实例)             (多实例)             (多实例)
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
- 透明转发请求头、Body、Query 参数

### 2. 安全与访问控制

| 功能            | 说明                                                   |
| --------------- | ------------------------------------------------------ |
| **JWT 鉴权**    | 校验 Bearer Token（HS256/RS256），提取 user_id 和 role |
| **身份透传**    | 校验通过后注入 `X-User-Id`、`X-User-Role` 到下游请求头 |
| **IP 黑白名单** | 支持精确 IP 和 CIDR 网段匹配                           |
| **CORS**        | 统一处理跨域，OPTIONS 预检直接返回 204                 |

### 3. 韧性与稳定性

| 功能         | 说明                                                                          |
| ------------ | ----------------------------------------------------------------------------- |
| **限流**     | 令牌桶算法，配置 QPS 和突发容量                                               |
| **熔断**     | 按服务名独立熔断，滑动窗口统计错误率，支持 closed → open → half-open 状态转换 |
| **超时控制** | 连接超时 + 响应超时，独立配置                                                 |

### 4. 可观测性

| 功能                | 说明                                                                                          |
| ------------------- | --------------------------------------------------------------------------------------------- |
| **Trace**           | 生成/传递 `X-Trace-Id`，注入 Context                                                          |
| **Request ID**      | 生成/传递 `X-Request-Id`                                                                      |
| **请求日志**        | 记录 method、path、status、latency、client_ip、trace_id                                       |
| **Prometheus 指标** | `gateway_requests_total`、`gateway_request_duration_seconds`、`gateway_circuit_breaker_state` |

## 快速开始

### 配置文件

```toml
[gateway]

[gateway.registry]
endpoints    = ["127.0.0.1:2379"]
dial_timeout = 5
prefix       = "/nexus/services"

[gateway.server]
addr = ":8080"

[gateway.balancer]
strategy = "round_robin"    # round_robin / random / weighted_round_robin

[gateway.jwt]
enabled    = true
secret     = "your-secret-key"
algorithm  = "HS256"        # HS256 / RS256
skip_paths = ["/health", "/metrics"]

[gateway.ip_filter]
enabled   = false
mode      = "blacklist"     # blacklist / whitelist
addresses = []

[gateway.rate_limit]
enabled = true
rate    = 1000.0            # tokens/s
burst   = 2000

[gateway.circuit]
enabled         = true
error_threshold = 0.5       # 50% 错误率触发熔断
min_requests    = 20
window_sec      = 30
cooldown_sec    = 15

[gateway.timeout]
connect_ms  = 3000
response_ms = 10000

[gateway.cors]
enabled       = true
allow_origins = ["*"]
allow_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"]
allow_headers = ["Content-Type", "Authorization", "X-Request-Id"]

[gateway.metrics]
enabled = true
path    = "/metrics"
```

### 启动网关

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

### 请求示例

```bash
# 通过网关访问 user-service
curl -H "Authorization: Bearer <token>" \
     http://localhost:8080/api/user-service/v1/users?page=1

# 健康检查
curl http://localhost:8080/health

# Prometheus 指标
curl http://localhost:8080/metrics
```

## 中间件执行顺序

```
请求 → Trace → RequestID → Logging(计时) → CORS → IPFilter → RateLimit → JWT
     → [路由匹配]
     → CircuitBreaker(按 service) → ProxyHandler(服务发现 + 转发)
     → Logging(记录日志 + 指标) → 响应
```

## 统一错误响应

网关拦截的错误统一返回 JSON 格式：

```json
{
    "code": 1001,
    "message": "missing authorization token",
    "trace_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| 错误码 | 含义         | HTTP 状态码 |
| ------ | ------------ | ----------- |
| 1001   | JWT 校验失败 | 401         |
| 1002   | IP 被拦截    | 403         |
| 1003   | 限流         | 429         |
| 1004   | 熔断         | 503         |
| 1005   | 服务未找到   | 502         |
| 1006   | 后端超时     | 504         |
| 1007   | 后端错误     | 502         |

## 目录结构

```
Nexus-Gateway/
├── gateway.go                  # 顶层入口: Setup / MustSetup / Start / Shutdown
├── internal/
│   └── response.go             # 共享类型: 错误码、统一响应、trace_id context
├── config/
│   └── config.go               # TOML 配置加载
├── gateway/
│   ├── gateway.go              # 核心: 中间件链组装 + 路由绑定 + 启动
│   ├── proxy.go                # 泛化调用反向代理
│   ├── response.go             # 响应工具 re-export
│   └── resolver_pool.go        # 按服务名懒加载 Resolver 缓存
├── middleware/
│   ├── trace.go                # Trace ID
│   ├── requestid.go            # Request ID
│   ├── logging.go              # 请求日志 + 指标
│   ├── cors.go                 # CORS
│   ├── jwt.go                  # JWT 鉴权 + 身份透传
│   ├── ipfilter.go             # IP 黑白名单
│   ├── ratelimit.go            # 令牌桶限流
│   └── circuitbreaker.go       # 按服务熔断
├── metrics/
│   └── metrics.go              # Prometheus 指标
└── example/
    ├── main.go                 # 示例启动
    └── config.toml             # 示例配置
```

## 依赖

- [Nexus-Registry](../Nexus-Registry) - etcd 服务注册发现 + 负载均衡
- [GoFrame v2](https://github.com/gogf/gf) - HTTP 服务框架
- [golang-jwt](https://github.com/golang-jwt/jwt) - JWT 解析
- [Prometheus Client](https://github.com/prometheus/client_golang) - 指标采集

