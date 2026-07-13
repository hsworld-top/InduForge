# NodeAgent Front

NodeAgent 的 Web 管理界面，基于 Vue 3 + Element Plus 实现。

## 功能特性

- ✅ 节点状态总览
- ✅ 项目列表和管理
- ✅ 部署/启动/停止/重启项目
- ✅ 版本回滚
- ✅ 连接配置管理
- ✅ 日志查看和下载
- ✅ 实时状态监控

## 技术栈

- Vue 3 (Composition API)
- Vue Router 4
- Pinia (状态管理)
- Element Plus (UI 组件库)
- Axios (HTTP 客户端)
- Vite (构建工具)
- Sass (样式预处理器)

## 开发环境搭建

### 1. 安装依赖

请在仓库根目录统一安装全部 Node workspace 依赖：

```bash
pnpm install
```

### 2. 启动开发服务器

```bash
pnpm dev:agent-front
```

服务默认读取仓库根目录 `.env`，通过 `VITE_NODE_AGENT_FRONT_PORT` 决定前端端口，当前统一默认访问地址是 `http://localhost:18604`。

开发服务器会自动代理 API 请求到 `NODE_AGENT_PORT` 指定的 NodeAgent 后端，当前统一默认值是 `18103`。

## 构建部署

### 1. 构建生产版本

```bash
pnpm build:agent-front
```

构建产物将输出到 `dist/` 目录。

### 2. 预览构建结果

```bash
pnpm --filter node_agent_front preview
```

## 项目结构

```
src/
├── api/              # API 调用
│   └── nodeApi.js
├── components/        # 公共组件
├── router/          # 路由配置
│   └── index.js
├── store/            # Pinia 状态管理
│   └── nodeStore.js
├── views/            # 页面组件
│   ├── Dashboard.vue  # 节点状态总览
│   ├── Projects.vue   # 项目管理
│   ├── Profile.vue    # 连接配置
│   └── Logs.vue      # 日志查看
├── App.vue          # 根组件
└── main.js          # 入口文件
```

## API 接口

前端通过 `/api/v1` 前缀调用 NodeAgent 后端接口：

- `GET /api/v1/node/info` - 获取节点信息
- `GET /api/v1/node/status` - 获取节点状态
- `GET /api/v1/projects` - 获取项目列表
- `POST /api/v1/projects/{id}/deploy` - 部署项目
- `POST /api/v1/projects/{id}/start` - 启动项目
- `POST /api/v1/projects/{id}/stop` - 停止项目
- `POST /api/v1/projects/{id}/restart` - 重启项目
- `POST /api/v1/projects/{id}/rollback` - 回滚项目
- `GET /api/v1/projects/{id}/logs` - 获取项目日志
- `GET /api/v1/profile` - 获取连接配置
- `POST /api/v1/profile` - 保存连接配置

## 页面说明

### Dashboard (仪表板)

- 显示节点基本信息
- 展示运行中的项目列表
- 提供快速操作按钮

### Projects (项目管理)

- 完整的项目列表
- 部署新项目
- 项目操作：启动/停止/重启/回滚
- 项目配置管理

### Profile (连接配置)

- 查看和编辑 ConnectionProfile
- 敏感信息脱敏显示
- 支持多种认证方式

### Logs (日志查看)

- 实时日志展示
- 按级别过滤
- 关键词搜索
- 自动刷新
- 日志下载

## 开发指南

### 添加新页面

1. 在 `src/views/` 创建 `.vue` 文件
2. 在 `src/router/index.js` 中添加路由
3. 在导航中添加链接

### 添加 API

1. 在 `src/api/nodeApi.js` 中添加接口函数
2. 在 `src/store/nodeStore.js` 中添加状态管理

### 自定义样式

项目使用 Sass，支持：

- 变量定义
- 嵌套规则
- 混合宏
- 函数

## 注意事项

1. 开发时需要同时运行 NodeAgent 后端（默认端口 `18103`）
2. 构建后的静态文件可以部署到任何静态文件服务器
3. 生产环境建议配置反向代理，将 API 请求转发到 NodeAgent

## 许可证

MIT License
