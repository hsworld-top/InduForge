# NodeAgent

NodeAgent 是用于在节点上部署和运行 RuntimeEngine 的 Go 语言实现。它提供本地 Web 运维接口，支持多种执行方式（进程、Docker、systemd）。

## 功能特性

- ✅ 部署和管理 RuntimeEngine
- ✅ 支持多种执行方式：进程（默认）、Docker、systemd
- ✅ 健康检查和自动恢复
- ✅ 心跳上报
- ✅ RESTful API 接口
- ✅ 版本管理和原子切换
- ✅ 连接配置管理

## 快速开始

### 1. 安装依赖

```bash
# 安装 Go 1.24+
# 安装 Docker (可选)
```

### 2. 构建

```bash
# 从仓库根目录进入模块
cd runtime/node_agent

# 下载依赖
go mod tidy

# 构建
go build -o node_agent ./cmd/main.go
```

### 3. 配置

配置文件 `config.yaml` 在首次运行时会自动生成在当前目录。如需自定义，可以编辑该文件：

监听端口优先读取进程环境变量或仓库根目录 `.env` 中的 `NODE_AGENT_PORT`。当前统一默认端口是 `18103`。

```yaml
agent:
  id: "node-001"
  executor:
    type: "process"  # process | docker | systemd
    process:
      workDir: "/var/lib/node_agent/runtime"
      logDir: "/var/log/node_agent/runtime"
      binary: "runtime_engine"
```

### 4. 运行

```bash
# 直接运行（配置文件将自动生成）
./node_agent

# 或者后台运行
./node_agent --hidden
```

首次运行时，程序会：
1. 自动生成默认配置文件（如果不存在）
2. 引导您完成交互式配置（开机自启动、端口设置等）
3. 启动后台服务进程
4. 显示服务信息（PID、端口、访问地址等）
5. 等待您按键后关闭交互窗口（后台服务继续运行）

## API 接口

### 节点信息

- `GET /api/v1/node/info` - 获取节点信息
- `GET /api/v1/node/status` - 获取节点状态

### 项目管理

- `GET /api/v1/projects` - 列出所有项目
- `GET /api/v1/projects/{id}` - 获取项目详情
- `POST /api/v1/projects/{id}/deploy` - 部署项目
- `POST /api/v1/projects/{id}/start` - 启动项目
- `POST /api/v1/projects/{id}/stop` - 停止项目
- `POST /api/v1/projects/{id}/restart` - 重启项目
- `POST /api/v1/projects/{id}/rollback?version={version}` - 回滚项目

### 连接配置

- `GET /api/v1/profile?projectId={id}` - 获取连接配置
- `POST /api/v1/profile?projectId={id}` - 保存连接配置

### 日志和健康检查

- `GET /api/v1/projects/{id}/logs` - 获取项目日志
- `GET /health` - 健康检查
- `GET /status` - 状态检查

## Web 管理界面

NodeAgent 提供了完整的 Web 管理界面，基于 Vue 3 开发，位于独立项目：

**前端项目**: `runtime/node_agent_front`

### 前端功能

- ✅ 节点状态总览
- ✅ 项目列表和管理
- ✅ 部署/启动/停止/重启项目
- ✅ 版本回滚
- ✅ 连接配置管理
- ✅ 日志查看和下载

### 启动前端

请在仓库根目录执行：

```bash
pnpm install
pnpm dev:agent-front
```

前端默认在 `http://localhost:18604` 启动，并通过根目录 `.env` 的 `NODE_AGENT_PORT` 代理访问 NodeAgent API，默认后端地址是 `http://localhost:18103`。

## 使用示例

### 部署项目

```bash
curl -X POST http://127.0.0.1:18103/api/v1/projects/demo/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.0.0",
    "ifpPackage": "/path/to/demo.ifp",
    "connectionProfile": {
      "name": "demo-profile",
      "endpoint": "http://localhost:18103",
      "authType": "token",
      "authData": {"token": "xxx"}
    },
    "autoStart": true
  }'
```

### 启动项目

```bash
curl -X POST http://127.0.0.1:18103/api/v1/projects/demo/start
```

## 目录结构

```
/var/lib/node_agent/
├── data/                  # 数据存储
│   └── projects/          # 项目配置
│       └── {projectID}/
│           ├── project.json
│           ├── profile.json
│           └── versions/
│               ├── v1.0.0/
│               └── current -> versions/v1.0.0
└── runtime/               # 运行时文件
    └── {projectID}/       # 项目目录

/var/log/node_agent/
└── runtime/               # 日志文件
    └── {projectID}.log
```

## 执行方式

### 进程模式（默认）

最简单的方式，直接启动进程。

```yaml
executor:
  type: "process"
  process:
    workDir: "/var/lib/node_agent/runtime"
    logDir: "/var/log/node_agent/runtime"
    binary: "runtime_engine"
```

### Docker 模式

使用 Docker 容器运行。

```yaml
executor:
  type: "docker"
  docker:
    enabled: true
    socket: "/var/run/docker.sock"
    network: "node_agent"
```

### Systemd 模式（Linux）

使用 systemd 管理服务。

```yaml
executor:
  type: "systemd"
  systemd:
    enabled: true
    unitTemplate: "/etc/systemd/system/node_agent_{project}.service"
```

## 故障排查

### 查看日志

```bash
# NodeAgent 日志
tail -f /var/log/node_agent/agent.log

# 项目日志
tail -f /var/log/node_agent/runtime/{projectID}.log
```

### 检查健康状态

```bash
curl http://127.0.0.1:18103/health
```

### 检查节点状态

```bash
curl http://127.0.0.1:18103/api/v1/node/status
```

## 开发指南

### 添加新的执行器

1. 实现 `Executor` 接口
2. 在 `executor_factory.go` 中注册
3. 更新配置文件

### Web 界面开发

静态文件位于 `internal/web/static/dist/`。

使用以下命令构建：

```bash
cd internal/web/static
npm run build
```

然后将构建产物复制到 `dist/` 目录。

## 许可证

MIT License
