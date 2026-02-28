# Nexus-Config 快速开始指南

## 步骤 1：启动配置中心服务

打开第一个终端，启动服务端：

```bash
cd Nexus-Config
go run example/server/main.go
```

你应该看到类似输出：

```
storage initialized
admin server starting on :8081
config server starting on :8082
all servers started successfully
Admin API: http://localhost:8081
Config API: http://localhost:8082
```

## 步骤 2：创建配置

打开第二个终端，运行 Admin 示例来创建配置：

```bash
cd Nexus-Config
go run example/admin/main.go
```

这个脚本会自动：
1. 创建命名空间 `myapp`
2. 保存配置草稿 `app.yaml`
3. 发布配置
4. 设置 30% 灰度规则

## 步骤 3：启动客户端测试

打开第三个终端，启动客户端：

```bash
cd Nexus-Config
go run example/client/main.go
```

客户端会：
- 首次拉取配置
- 启动长轮询监听配置变更
- 当配置变更时自动打印新配置

## 步骤 4：测试配置热更新

### 方式 1：使用 curl 发布新配置

在第四个终端执行：

```bash
# 更新草稿
curl -X POST http://localhost:8081/api/v1/configs/draft \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "myapp",
    "key": "app.yaml",
    "value": "server:\n  port: 9090\n  host: \"0.0.0.0\"\n\ndatabase:\n  dsn: \"mysql://localhost:3306/myapp_updated\"\n\nfeatures:\n  new_ui: true\n  beta_feature: true",
    "format": "yaml"
  }'

# 发布配置
curl -X POST http://localhost:8081/api/v1/configs/publish \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "myapp",
    "key": "app.yaml"
  }'
```

此时，客户端会立即收到配置变更通知！

### 方式 2：使用 Admin 示例修改灰度比例

```bash
# 修改灰度比例为 100%（全量灰度）
curl -X POST http://localhost:8081/api/v1/gray/ \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "myapp",
    "key": "app.yaml",
    "percentage": 100,
    "enabled": true
  }'
```

## 步骤 5：测试多客户端灰度

启动多个客户端，修改 `example/client/config.toml` 中的 `client_id`：

```toml
# 客户端 1
client_id = "client-001"

# 客户端 2
client_id = "client-002"

# 客户端 3
client_id = "client-003"
```

然后分别启动，观察不同客户端在灰度发布时拿到的配置是否不同（30% 的客户端会拿到草稿版本）。

## API 测试示例

### 查询命名空间列表

```bash
curl http://localhost:8081/api/v1/namespaces/
```

### 查询配置列表

```bash
curl "http://localhost:8081/api/v1/configs/list?namespace=myapp"
```

### 获取已发布配置

```bash
curl "http://localhost:8081/api/v1/configs/published?namespace=myapp&key=app.yaml"
```

### 获取草稿配置

```bash
curl "http://localhost:8081/api/v1/configs/draft?namespace=myapp&key=app.yaml"
```

### 查询灰度规则

```bash
curl "http://localhost:8081/api/v1/gray/?namespace=myapp&key=app.yaml"
```

## 常见问题

### Q: 客户端连接不上服务器？

A: 检查服务端是否正常启动，确认 `config.toml` 中的地址配置正确。

### Q: 配置变更后客户端没有收到通知？

A: 检查：
1. 配置是否已发布（不是草稿）
2. 服务端日志是否有报错
3. 客户端是否正常启动长轮询

### Q: 灰度不生效？

A: 检查：
1. 灰度规则是否启用（`enabled: true`）
2. 是否有草稿配置（灰度客户端使用草稿）
3. `client_id` 是否唯一

## 下一步

- 查看 [README.md](README.md) 了解完整功能
- 查看 [API 文档](README.md#api-文档)
- 集成到你的应用中
