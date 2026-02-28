# Nexus-Config Web UI

Nexus-Config 配置中心的管理界面，基于 React + TypeScript + Vite + Ant Design + Monaco Editor 构建。

## 核心功能

### 1. 命名空间管理
- 创建、查看、删除命名空间
- 命名空间列表展示

### 2. 配置管理（核心功能）
- **Monaco Editor 编辑器**：支持 YAML/JSON/TOML/Properties 语法高亮
- **配置草稿**：编辑配置时自动保存为草稿
- **配置发布**：将草稿发布为正式版本
- **Diff 对比视图**：使用 Monaco Diff Editor 对比草稿和已发布版本
  - 左侧：已发布版本（Production）
  - 右侧：草稿版本（Draft）
  - 自动高亮差异

### 3. 灰度发布管理
- 可视化灰度百分比设置（0-100%）
- 灰度规则启用/禁用
- 灰度规则列表管理
- 进度条展示灰度比例

## 技术栈

- **框架**: React 18 + TypeScript
- **构建工具**: Vite
- **UI 组件**: Ant Design 5
- **代码编辑器**: Monaco Editor（VS Code 编辑器内核）
- **路由**: React Router 6
- **HTTP 客户端**: Axios
- **样式**: CSS + Ant Design 主题

## 快速开始

### 1. 安装依赖

```bash
cd web
npm install
```

### 2. 开发模式

```bash
npm run dev
```

访问：http://localhost:3000

开发模式会自动代理 API 请求到 `http://localhost:8081`（后端 Admin API）

### 3. 生产构建

```bash
npm run build
```

构建产物输出到 `dist/` 目录。

### 4. 预览构建结果

```bash
npm run preview
```

## 项目结构

```
web/
├── src/
│   ├── api/                  # API 调用封装
│   │   ├── request.ts        # Axios 实例配置
│   │   ├── namespace.ts      # 命名空间 API
│   │   ├── config.ts         # 配置管理 API
│   │   ├── gray.ts           # 灰度规则 API
│   │   └── index.ts
│   ├── components/           # 公共组件
│   │   ├── ConfigEditor.tsx  # Monaco Editor 封装
│   │   ├── DiffViewer.tsx    # Diff Editor 封装
│   │   └── index.ts
│   ├── layouts/              # 布局组件
│   │   └── MainLayout.tsx    # 主布局
│   ├── pages/                # 页面组件
│   │   ├── Namespaces.tsx    # 命名空间管理
│   │   ├── Configs.tsx       # 配置管理
│   │   ├── GrayRules.tsx     # 灰度规则管理
│   │   └── index.ts
│   ├── types/                # TypeScript 类型定义
│   │   └── index.ts
│   ├── App.tsx               # 根组件
│   ├── main.tsx              # 入口文件
│   ├── index.css             # 全局样式
│   └── vite-env.d.ts
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

## 页面功能详解

### 命名空间管理页面
- 创建命名空间（ID、名称、描述）
- 列表展示所有命名空间
- 点击"管理配置"跳转到配置管理页面
- 删除命名空间（带确认）

### 配置管理页面
- **创建配置**：
  - 配置键（key）
  - 配置格式（YAML/JSON/TOML/Properties）
  - 配置内容（Monaco Editor）

- **编辑配置**（抽屉弹出）：
  - 全屏 Monaco Editor 编辑器
  - 实时保存草稿
  - 支持语法高亮

- **对比视图**（抽屉弹出）：
  - 左右分屏展示已发布版本和草稿版本
  - 自动高亮差异行
  - 支持内联对比和并排对比

- **发布配置**：
  - 将草稿版本发布为正式版本
  - 只有存在草稿时才能发布

- **删除配置**：
  - 删除配置的所有版本（草稿和已发布）

### 灰度发布管理页面
- **创建灰度规则**：
  - 选择配置键
  - 设置灰度百分比（滑块控件）
  - 启用/禁用开关

- **编辑灰度规则**：
  - 调整灰度百分比
  - 切换启用状态

- **快速切换**：
  - 表格内一键启用/禁用灰度规则

- **可视化展示**：
  - 进度条展示灰度比例
  - 状态标签（已启用/已禁用）

## Monaco Editor 特性

### 支持的语言
- YAML
- JSON
- TOML（使用 INI 语法高亮）
- Properties（使用 INI 语法高亮）

### 编辑器功能
- 语法高亮
- 代码折叠
- 自动缩进
- 查找替换
- 多光标编辑
- 自动补全

### Diff Editor 功能
- 并排对比
- 内联对比
- 差异高亮
- 导航到下一个/上一个差异
- 只读模式（默认）

## API 集成

前端通过 Axios 调用后端 Admin API：

- `POST /api/v1/namespaces/` - 创建命名空间
- `GET /api/v1/namespaces/` - 获取命名空间列表
- `POST /api/v1/configs/draft` - 保存配置草稿
- `GET /api/v1/configs/draft` - 获取配置草稿
- `POST /api/v1/configs/publish` - 发布配置
- `GET /api/v1/configs/published` - 获取已发布配置
- `GET /api/v1/configs/list` - 获取配置列表
- `DELETE /api/v1/configs/` - 删除配置
- `POST /api/v1/gray/` - 保存灰度规则
- `GET /api/v1/gray/` - 获取灰度规则
- `DELETE /api/v1/gray/` - 删除灰度规则
- `GET /api/v1/gray/list` - 获取灰度规则列表

## 开发说明

### 代理配置
开发模式下，Vite 自动代理 `/api` 请求到 `http://localhost:8081`：

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8081',
      changeOrigin: true,
    },
  },
}
```

### 暗色主题
默认使用 Ant Design 暗色主题：

```typescript
// App.tsx
<ConfigProvider theme={{ algorithm: theme.darkAlgorithm }}>
  {/* ... */}
</ConfigProvider>
```

## 部署

### 方式 1: 与后端一起部署（推荐）

1. 构建前端：
```bash
cd web
npm run build
```

2. 启动后端服务：
```bash
cd ..
go run example/server/main.go
```

后端会自动服务前端静态文件（来自 `web/dist`）：
- Web UI: http://localhost:8081
- API: http://localhost:8081/api/v1

### 方式 2: 单独部署

使用 Nginx 或其他静态文件服务器部署 `dist/` 目录，并配置反向代理：

```nginx
server {
  listen 80;
  server_name config.example.com;

  root /path/to/web/dist;
  index index.html;

  # 前端路由
  location / {
    try_files $uri $uri/ /index.html;
  }

  # API 代理
  location /api/ {
    proxy_pass http://localhost:8081;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
  }
}
```

## 故障排除

### 1. Monaco Editor 加载失败
确保网络正常，Monaco Editor 会从 CDN 加载 worker 文件。

### 2. API 请求失败
检查后端服务是否启动：
```bash
curl http://localhost:8081/api/v1/namespaces/
```

### 3. 构建错误
清理依赖重新安装：
```bash
rm -rf node_modules package-lock.json
npm install
```

## 浏览器兼容性

- Chrome/Edge >= 90
- Firefox >= 88
- Safari >= 14

## 许可证

MIT License

## 作者

krustd
