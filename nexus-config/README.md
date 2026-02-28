# Nexus-Config

Nexus-Config 是一个基于 GoFrame (gf) 框架的分布式配置中心，支持配置的实时推送、灰度发布和多格式配置管理。

## 核心特性

- **配置实时推送**：基于 HTTP Long Polling 的配置变更实时通知
- **灰度发布**：支持按百分比灰度发布新配置，基于客户端 ID 哈希分流
- **多格式支持**：支持 YAML、JSON、TOML、Properties 等多种配置格式
- **草稿管理**：支持草稿箱和正式版本分离，安全发布
- **RESTful API**：提供完善的配置管理 API
- **轻量级 SDK**：客户端 SDK 自动处理长轮询和本地缓存

## 架构设计

Nexus-Config 采用三层架构：

1. **控制层 (Admin API)**：提供配置的 CRUD 和灰度规则管理
2. **存储层 (Storage)**：支持 SQLite、MySQL 等多种存储后端
3. **分发层 (Server + SDK)**：负责配置推送和客户端订阅

```
┌─────────────┐
│  Admin API  │  (配置管理、灰度规则)
└──────┬──────┘
       │
┌──────▼──────┐
│   Storage   │  (SQLite/MySQL)
└──────┬──────┘
       │
┌──────▼──────┐
│   Server    │  (Long Polling 分发)
└──────┬──────┘
       │
┌──────▼──────┐
│  SDK Client │  (自动拉取、缓存)
└─────────────┘
```

## 快速开始

### 1. 安装依赖

```bash
cd Nexus-Config
go mod tidy
```

### 2. 启动配置中心服务

```bash
go run example/server/main.go
```

服务启动后：
- Admin API: `http://localhost:8081`
- Config API: `http://localhost:8082`
- Web UI: `http://localhost:8081` (需先构建前端，见下文)

### 3. 管理配置（可选）

运行 Admin 示例来创建配置：

```bash
go run example/admin/main.go
```

这会自动：
1. 创建命名空间 `myapp`
2. 保存草稿配置 `app.yaml`
3. 发布配置
4. 设置 30% 灰度规则

### 4. 启动客户端

```bash
go run example/client/main.go
```

客户端会：
- 首次拉取配置
- 启动长轮询监听配置变更
- 配置变更时自动触发回调

### 5. 启动 Web 管理界面（可选）

Web 管理界面提供可视化的配置管理，包括：
- **Monaco Editor**：支持 YAML/JSON/TOML/Properties 语法高亮编辑
- **Diff 对比视图**：可视化对比草稿和已发布版本的差异
- **灰度发布管理**：可视化设置灰度规则

#### 开发模式
```bash
# 安装依赖
cd web
npm install

# 启动开发服务器
npm run dev
```

访问：http://localhost:3000

#### 生产模式
```bash
# 构建前端
cd web
npm run build

# 启动后端服务（会自动服务前端）
cd ..
go run example/server/main.go
```

访问：http://localhost:8081

详细文档见 [web/README.md](web/README.md)

## 使用指南

### 服务端配置

创建 `config.toml`：

```toml
[database]
type = "sqlite"
file_path = "./config.db"

[admin]
addr = ":8081"

[server]
addr = ":8082"
```

### 客户端配置

创建 `client.toml`：

```toml
server_addr = "http://localhost:8082"
namespace = "myapp"
config_key = "app.yaml"
client_id = "client-001"
poll_timeout = 30
retry_delay = 5
```

### 代码示例

#### 服务端

```go
package main

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/krustd/nexus-config/admin"
	"github.com/krustd/nexus-config/common"
	"github.com/krustd/nexus-config/server"
	"github.com/krustd/nexus-config/storage/sqlite"
)

func main() {
	ctx := context.Background()
	cfg, _ := common.LoadServerConfig("config.toml")

	// 初始化存储
	store, _ := sqlite.NewSQLiteStorage(cfg.Database.FilePath)
	store.Init(ctx)
	defer store.Close()

	// 创建配置变更通知器
	notifier := server.NewConfigNotifier()

	// 启动 Admin API（需要传入 notifier 以便发布配置时通知客户端）
	adminServer := g.Server("admin")
	admin.SetupRouter(adminServer, store, notifier)
	adminServer.SetAddr(cfg.Admin.Addr)
	go adminServer.Start()

	// 启动配置分发服务
	configServer := g.Server("config")
	server.SetupRouter(configServer, store, notifier)
	configServer.SetAddr(cfg.Server.Addr)
	configServer.Start()
}
```

#### 客户端

```go
package main

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/krustd/nexus-config/common"
	"github.com/krustd/nexus-config/sdk"
)

type AppConfig struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
}

func main() {
	ctx := context.Background()
	cfg, _ := common.LoadClientConfig("client.toml")

	// 创建客户端
	client := sdk.NewClient(cfg)

	// 添加配置变更监听器
	client.AddChangeListener(func(version *common.ConfigVersion) {
		g.Log().Infof(ctx, "配置已变更: %s", version.MD5)
		// 执行热更新逻辑
	})

	// 启动客户端
	client.Start(ctx)
	defer client.Stop()

	// 获取配置
	var appCfg AppConfig
	client.GetValueAs(&appCfg)

	g.Log().Infof(ctx, "当前配置: %+v", appCfg)

	select {} // 保持运行
}
```

## API 文档

### Admin API

#### 创建命名空间

```bash
POST /api/v1/namespaces/
Content-Type: application/json

{
  "id": "myapp",
  "name": "我的应用",
  "description": "示例应用"
}
```

#### 保存草稿

```bash
POST /api/v1/configs/draft
Content-Type: application/json

{
  "namespace": "myapp",
  "key": "app.yaml",
  "value": "server:\n  port: 8080",
  "format": "yaml"
}
```

#### 发布配置

```bash
POST /api/v1/configs/publish
Content-Type: application/json

{
  "namespace": "myapp",
  "key": "app.yaml"
}
```

#### 设置灰度规则

```bash
POST /api/v1/gray/
Content-Type: application/json

{
  "namespace": "myapp",
  "key": "app.yaml",
  "percentage": 30,
  "enabled": true
}
```

### Config API

#### 长轮询配置

```bash
POST /api/v1/config/poll
Content-Type: application/json

{
  "namespace": "myapp",
  "key": "app.yaml",
  "client_id": "client-001",
  "md5": "abc123"
}
```

响应：

```json
{
  "changed": true,
  "version": {
    "namespace": "myapp",
    "key": "app.yaml",
    "md5": "def456",
    "value": "server:\n  port: 8080",
    "format": "yaml"
  }
}
```

## 灰度发布

Nexus-Config 支持基于百分比的灰度发布：

1. **编辑草稿**：在 Admin 中编辑配置草稿（新版本）
2. **设置灰度规则**：设置灰度百分比（0-100）并启用
3. **客户端分流**：基于 `client_id` 的哈希值自动分流
   - 命中灰度的客户端使用草稿版本
   - 未命中的客户端使用已发布版本
4. **全量发布**：确认无误后，发布配置并关闭灰度

### 灰度计算逻辑

```go
hash := fnv.New32a()
hash.Write([]byte(clientID))
hashValue := hash.Sum32()

if int(hashValue%100) < percentage {
    // 命中灰度，使用草稿版本
} else {
    // 使用已发布版本
}
```

## 配置格式支持

支持以下配置格式：

- **YAML** (推荐)
- **JSON**
- **TOML**
- **Properties**

SDK 会根据配置的 `format` 字段自动解析到 Go 结构体。

## 项目结构

```
Nexus-Config/
├── admin/              # Admin API (配置管理)
│   ├── dto.go          # 请求/响应 DTO
│   ├── handler.go      # 业务处理器
│   └── router.go       # 路由设置
├── server/             # 配置分发服务 (Long Polling)
│   ├── handler.go      # 配置分发处理器
│   ├── notifier.go     # 配置变更通知器
│   └── router.go       # 路由设置
├── storage/            # 存储层
│   ├── iface.go        # 存储接口定义
│   └── sqlite/         # SQLite 实现
│       └── sqlite.go
├── sdk/                # 客户端 SDK
│   ├── cache.go        # 本地缓存
│   └── client.go       # SDK 客户端
├── common/             # 公共定义
│   ├── types.go        # 数据模型
│   ├── config.go       # 配置加载
│   └── format.go       # 格式解析
├── web/                # Web 管理界面 (React + Monaco Editor)
│   ├── src/
│   │   ├── api/        # API 调用
│   │   ├── components/ # Monaco Editor 等组件
│   │   ├── pages/      # 页面
│   │   └── types/      # TypeScript 类型
│   ├── package.json
│   └── README.md
├── example/            # 示例代码
│   ├── server/         # 服务端示例
│   ├── client/         # 客户端示例
│   └── admin/          # Admin API 使用示例
├── config.go           # 顶层封装（可选）
├── main.go             # 主程序入口（可选）
└── go.mod              # Go 模块定义
```

## 技术栈

### 后端
- **框架**：GoFrame (gf)
- **存储**：GORM + SQLite/MySQL
- **通信**：HTTP Long Polling
- **配置格式**：YAML、JSON、TOML、Properties

### 前端（Web UI）
- **框架**：React 18 + TypeScript
- **构建**：Vite
- **UI 组件**：Ant Design 5
- **代码编辑器**：Monaco Editor（VS Code 内核）
- **路由**：React Router 6

## 许可证

MIT License

## 作者

krustd

## 相关项目

- [Nexus-Registry](../Nexus-Registry): 服务注册与发现
- [Nexus-Gateway](../Nexus-Gateway): API 网关（待开发）
