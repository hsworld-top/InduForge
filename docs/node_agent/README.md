# NodeAgent 详细功能文档

## 概述

NodeAgent 是用于在节点上部署和运行 RuntimeEngine 的核心组件，采用 Go 语言实现。它提供了完整的运行时管理能力，支持多种执行方式，并提供 RESTful API 接口供前端调用。

## 首次启动与初始化

### 运行模式

NodeAgent 支持两种运行模式：

**在线模式（Online）**
- 受运维中心统一管控
- 支持远程部署和操作
- 实时监控和告警
- 自动接收部署指令
- 需要通过运维中心审批

**离线模式（Offline）**
- 独立运行，不连接运维中心
- 手动导入工程包
- 本地管理启停
- 适合开发测试或隔离网络环境
- 无需审批，直接可用

### 初始化向导

首次启动 NodeAgent 前端时，会自动进入初始化向导，引导完成配置：

**步骤1：选择运行模式**
- 选择在线模式或离线模式
- 模式选择后不可更改

**步骤2：配置信息**

在线模式需要填写：
- 运维中心地址（如：http://192.168.1.100:9099）
- 用户名和密码（用于身份验证）
- 节点名称（仅允许字母、数字、中划线、下划线，3-50字符）
- 节点描述（可选）
- IP地址（自动获取或手动输入）
- 管理端口（默认8080）

离线模式仅需填写：
- 节点名称
- 节点描述（可选）

**步骤3：审批流程（仅在线模式）**

注册申请提交后：
- 如果申请人是 OPS_ADMIN 或 SYSTEM_ADMIN：自动审批通过，立即获取 token
- 如果申请人是其他角色：进入待审批状态，需要等待管理员审批
- 待审批期间，前端会每5秒自动轮询审批状态
- 也可以手动点击刷新按钮查询
- 审批通过后自动保存配置并启动心跳服务

**步骤4：完成**
- 显示初始化完成信息
- 点击"进入控制台"开始使用

### 注册认证流程

在线模式注册时的认证流程：

1. 用户在 NodeAgent 前端输入运维中心地址、用户名密码
2. NodeAgent 向运维中心发送注册请求（POST /api/v1/node-register/register-with-auth）
3. 运维中心验证用户身份和角色
4. 根据用户角色决定审批策略：
   - OPS_ADMIN/SYSTEM_ADMIN：自动审批，返回 approved 状态和 token
   - 其他角色：创建 pending 记录，等待人工审批
5. NodeAgent 前端轮询审批状态（GET /api/v1/node-register/:nodeId/approval-status）
6. 审批通过后，NodeAgent 保存配置到 config.yaml 并启动心跳服务

## 核心架构

### 模块组成

```
internal/
├── agent/                    # 核心 Agent 逻辑
│   ├── orchestrator/        # 状态机编排器
│   ├── executor/            # 执行器抽象层
│   ├── store/              # 本地存储管理
│   ├── health/             # 健康检查系统
│   └── network/            # 网络通信
├── pkg/                    # 公共包
│   ├── types/              # 类型定义
│   ├── logger/             # 日志系统
│   └── utils/              # 工具函数
└── web/handler/            # HTTP API 处理
```

## 功能特性详解

### 1. 执行器系统 (Executor)

#### 1.1 ProcessExecutor（进程执行器）

**功能描述**: 默认执行器，直接在节点上启动和管理 RuntimeEngine 进程。

**核心功能**:
- **进程启动**: 使用 `exec.CommandContext` 启动 RuntimeEngine
- **进程停止**: 发送 SIGINT 信号，必要时强制杀死
- **进程重启**: 先停止再启动，支持优雅重启
- **日志管理**: 实时输出日志到指定文件
- **PID 管理**: 记录并监控进程 PID

**文件位置**: `internal/agent/executor/process.go`

**主要方法**:
```go
type ProcessExecutor struct {
    Deploy(ctx context.Context, req DeployRequest) error  // 部署准备
    Start(ctx context.Context, projectID string) error   // 启动进程
    Stop(ctx context.Context, projectID string) error    // 停止进程
    Restart(ctx context.Context, projectID string) error // 重启进程
    GetStatus(ctx context.Context, projectID string) (*RuntimeStatus, error) // 获取状态
}
```

**目录结构**:
```
/var/lib/node_agent/runtime/
└── {projectID}/
    ├── current/              # 当前版本（软链接）
    │   └── runtime_engine    # RuntimeEngine 可执行文件
    └── versions/             # 版本目录
        ├── v1.0.0/
        └── v1.0.1/
```

**日志输出**:
- 位置: `/var/log/node_agent/runtime/{projectID}.log`
- 格式: 标准输出和错误输出
- 轮转: 需要外部工具（如 logrotate）管理

#### 1.2 DockerExecutor（Docker 执行器）

**功能描述**: 使用 Docker 容器运行 RuntimeEngine，提供隔离性和可移植性。

**核心功能**:
- **镜像管理**: 构建和管理 RuntimeEngine 镜像
- **容器生命周期**: 创建、启动、停止、删除容器
- **网络配置**: 独立的容器网络
- **卷挂载**: 支持数据持久化

**文件位置**: `internal/agent/executor/docker.go`

**主要方法**:
```go
type DockerExecutor struct {
    Deploy(ctx context.Context, req DeployRequest) error  // 构建镜像
    Start(ctx context.Context, projectID string) error   // 启动容器
    Stop(ctx context.Context, projectID string) error    // 停止容器
    Restart(ctx context.Context, projectID string) error // 重启容器
    GetStatus(ctx context.Context, projectID string) (*RuntimeStatus, error)
}
```

**容器命名规范**:
- 格式: `node_agent_{projectID}`
- 镜像命名: `{imagePrefix}_{projectID}:{version}`

**Docker 网络**:
- 默认网络: `node_agent`
- 端口映射: 通过配置指定
- 服务发现: 容器间通信

#### 1.3 执行器选择机制

**配置优先级**:
1. 部署请求中的 `executorType` 参数
2. 配置文件中的默认执行器
3. 环境变量 `NODE_AGENT_EXECUTOR_TYPE`

**故障转移**:
- Docker 不可用时自动回退到进程模式
- 自动日志记录切换原因
- 不中断服务运行

### 2. 状态机编排 (Orchestrator)

**功能描述**: 使用状态机确保项目操作的原子性和一致性。

**状态定义**:
```go
const (
    StateStopped     State = "stopped"     // 已停止
    StateDeploying   State = "deploying"  // 部署中
    StateRunning     State = "running"     // 运行中
    StateStopping    State = "stopping"    // 停止中
    StateError       State = "error"       // 错误
    StateRollingBack State = "rolling_back" // 回滚中
)
```

**状态转换图**:
```
Stopped ──[deploy]──> Deploying ──[health_ok]──> Running
                      │                           │
                      └──[error]────> Error ─────┘
                           │            │
Running ──[stop]────> Stopping ──> Stopped
  │
  ├──[restart]──> Running
  │
  ├──[rollback]──> RollingBack ──> Stopped
  │
  └──[error]─────> Error
```

**文件位置**: `internal/agent/orchestrator/orchestrator.go`

**主要方法**:
```go
type Orchestrator interface {
    Deploy(ctx context.Context, req DeployRequest) error    // 部署
    Start(ctx context.Context, projectID string) error    // 启动
    Stop(ctx context.Context, projectID string) error     // 停止
    Restart(ctx context.Context, projectID string) error  // 重启
    Rollback(ctx context.Context, projectID, version string) error // 回滚
    GetProjectStatus(ctx context.Context, projectID string) (*ProjectInfo, error) // 查询状态
}
```

**原子性保证**:
- 所有操作通过状态机验证
- 失败时自动回滚到上一状态
- 记录状态变更历史

### 3. 本地存储 (Store)

**功能描述**: 管理项目配置、版本和连接信息的本地存储。

**文件位置**: `internal/agent/store/store.go`

**存储结构**:
```
/var/lib/node_agent/data/
└── projects/
    └── {projectID}/
        ├── project.json        # 项目信息
        ├── profile.json        # 连接配置（600权限）
        ├── versions/           # 版本文件
        │   ├── v1.0.0/
        │   └── v1.0.1/
        └── current ─────────> versions/v1.0.1  # 软链接
```

**敏感信息处理**:
- `profile.json` 权限设为 `0600`（仅所有者可读写）
- `secrets` 字段自动清理，不持久化
- 脱敏显示敏感数据

**主要方法**:
```go
type Store interface {
    SaveProject(project *ProjectInfo) error              // 保存项目
    GetProject(projectID string) (*ProjectInfo, error)  // 获取项目
    ListProjects() ([]*ProjectInfo, error)             // 列出项目
    SaveConnectionProfile(projectID string, profile ConnectionProfile) error // 保存配置
    GetConnectionProfile(projectID string) (*ConnectionProfile, error) // 获取配置
    SwitchVersion(projectID, version string) error     // 切换版本（原子操作）
    GetCurrentVersion(projectID string) (string, error) // 获取当前版本
}
```

**原子性切换**:
```go
func (s *LocalStore) SwitchVersion(projectID, version string) error {
    projectDir := filepath.Join(s.dataDir, "projects", projectID)
    versionDir := filepath.Join(projectDir, "versions", version)
    currentLink := filepath.Join(projectDir, "current")

    // 原子性软链接切换
    return utils.Symlink(versionDir, currentLink)
}
```

### 4. 健康检查 (Health Checker)

**功能描述**: 定期监控 RuntimeEngine 的健康状态，自动发现和处理故障。

**文件位置**: `internal/agent/health/checker.go`

**配置选项**:
```yaml
runtime:
  healthCheck:
    enabled: true        # 是否启用
    interval: 30s        # 检查间隔
    timeout: 5s          # 超时时间
    retries: 3           # 重试次数
```

**检查流程**:
1. 定期向 `/health` 端点发送 HTTP 请求
2. 检查响应状态码（200 表示健康）
3. 记录失败次数
4. 超过阈值触发重启

**主要方法**:
```go
type HealthChecker struct {
    CheckRuntime(ctx context.Context, projectID, endpoint string) (*HealthCheckResult, error)
    MonitorProject(ctx context.Context, projectID, endpoint string)  // 启动监控
    GetProjectHealth(projectID string) *ProjectHealth                // 获取健康状态
}
```

**自动恢复**:
- 连续失败达到阈值时自动重启
- 重启后继续监控
- 记录恢复操作日志

### 5. 心跳上报 (Heartbeat)

**功能描述**: 定期向管理中心上报节点状态和系统指标。

**文件位置**: `internal/agent/network/heartbeat.go`

**配置选项**:
```yaml
network:
  heartbeat:
    enabled: true        # 是否启用
    interval: 10s         # 上报间隔
    endpoint: "http://manager:8080/api/v1/nodes/heartbeat"
```

**上报数据**:
```json
{
  "nodeId": "node-001",
  "timestamp": "2024-01-23T15:00:00Z",
  "status": "healthy",
  "metrics": {
    "cpu_usage": 45.2,
    "memory_usage": 62.8,
    "disk_usage": 33.5,
    "running_projects": 3,
    "hostname": "node-001"
  },
  "projects": ["demo", "api", "worker"]
}
```

**主要方法**:
```go
type HeartbeatSender struct {
    Start(ctx context.Context)                         // 启动心跳
    UpdateProjects(projects []string)                 // 更新项目列表
}
```

**收集指标**:
- CPU 使用率
- 内存使用率
- 磁盘使用率
- 运行项目数量
- 主机名
- 时间戳

### 6. API 接口 (HTTP API)

**功能描述**: 提供完整的 RESTful API，供前端调用。

**文件位置**: `internal/web/handler/api.go`

#### 6.1 节点信息接口

| 方法 | 路径 | 功能 | 响应示例 |
|------|------|------|----------|
| GET | `/api/v1/node/info` | 获取节点信息 | `{"id":"node-001","version":"1.0.0","executorType":"process"}` |
| GET | `/api/v1/node/status` | 获取节点状态 | `{"node":{"status":"healthy","projects":3}}` |

#### 6.2 项目管理接口

| 方法 | 路径 | 功能 | 请求体/参数 | 响应 |
|------|------|------|-------------|------|
| GET | `/api/v1/projects` | 列出所有项目 | - | `[]ProjectInfo` |
| GET | `/api/v1/projects/{id}` | 获取项目详情 | - | `ProjectInfo` |
| POST | `/api/v1/projects/{id}/deploy` | 部署项目 | `DeployRequest` | `{"status":"success"}` |
| POST | `/api/v1/projects/{id}/start` | 启动项目 | - | `{"status":"success"}` |
| POST | `/api/v1/projects/{id}/stop` | 停止项目 | - | `{"status":"success"}` |
| POST | `/api/v1/projects/{id}/restart` | 重启项目 | - | `{"status":"success"}` |
| POST | `/api/v1/projects/{id}/rollback` | 回滚版本 | `?version=v1.0.0` | `{"status":"success"}` |
| GET | `/api/v1/projects/{id}/logs` | 获取项目日志 | - | `[]LogEntry` |

**部署请求示例**:
```json
{
  "version": "1.0.0",
  "ifpPackage": "/path/to/demo.ifp",
  "executorType": "process",
  "connectionProfile": {
    "name": "demo-profile",
    "endpoint": "http://localhost:8080",
    "authType": "token",
    "authData": {"token": "xxx"},
    "metadata": {"env":"prod"}
  },
  "autoStart": true
}
```

#### 6.3 连接配置接口

| 方法 | 路径 | 功能 | 参数 | 响应 |
|------|------|------|------|------|
| GET | `/api/v1/profile` | 获取连接配置 | `?projectId=demo` | `ConnectionProfile` |
| POST | `/api/v1/profile` | 保存连接配置 | `?projectId=demo` + 请求体 | `{"status":"success"}` |

#### 6.4 健康检查接口

| 方法 | 路径 | 功能 | 响应 |
|------|------|------|------|
| GET | `/health` | 健康检查 | `{"status":"healthy","time":"..."}` |
| GET | `/status` | 详细状态 | `{"node":{},"projects":[]}` |

### 7. 配置管理

**文件位置**: `configs/config.yaml`

**完整配置示例**:
```yaml
agent:
  id: "node-001"
  mode: "online"  # online | offline（初始化向导后自动设置）
  
  # 在线模式配置（初始化后自动填充）
  online:
    centerUrl: "http://192.168.1.100:9099"
    nodeId: "uuid-generated-by-center"
    registrationToken: "token-generated-by-center"
  
  # 离线模式配置
  offline:
    enabled: true
  
  listen:
    host: "127.0.0.1"
    port: 8080

  executor:
    type: "process"          # 默认执行器
    process:
      workDir: "/var/lib/node_agent/runtime"
      logDir: "/var/log/node_agent/runtime"
      binary: "runtime_engine"
    docker:
      enabled: false
      socket: "/var/run/docker.sock"
      network: "node_agent"
      imagePrefix: "node_agent_runtime"

  runtime:
    workDir: "/var/lib/runtime"
    healthCheck:
      enabled: true
      interval: 30s
      timeout: 5s
      endpoint: "/health"
      retries: 3

  network:
    heartbeat:
      enabled: true
      interval: 10s
      endpoint: "http://manager:8080/api/v1/nodes/heartbeat"

logging:
  level: "info"
  format: "json"
  output: "stdout"
  file: "/var/log/node_agent/agent.log"

storage:
  type: "local"
  local:
    dataDir: "/var/lib/node_agent/data"
```

**配置说明**:
- `mode`: 节点运行模式（online/offline），由初始化向导设置
- `online.centerUrl`: 运维中心地址，初始化时填写
- `online.nodeId`: 节点ID，注册后由运维中心分配
- `online.registrationToken`: 注册令牌，审批通过后由运维中心生成

**配置优先级**:
1. 配置文件 (`config.yaml`)
2. 环境变量（如 `NODE_AGENT_CONFIG`）
3. 命令行参数

### 8. 日志系统

**文件位置**: `internal/pkg/logger/logger.go`

**日志级别**:
- `DEBUG`: 详细调试信息
- `INFO`: 一般信息
- `WARN`: 警告信息
- `ERROR`: 错误信息

**输出格式**:
- 控制台: 彩色输出，便于开发调试
- 文件: JSON 格式，便于解析

**使用示例**:
```go
logger.GlobalLogger.Info("启动项目", "project", projectID, "version", version)
logger.GlobalLogger.Error("启动失败", "error", err)
```

### 9. 构建和部署

**构建工具**: `Makefile`

**常用命令**:
```bash
make build      # 构建二进制
make run        # 构建并运行
make dev        # 直接运行（开发模式）
make clean      # 清理构建文件
make test       # 运行测试
make install    # 安装到系统
```

**Docker 构建**:
```bash
docker build -t node-agent:latest .
docker run -p 8080:8080 node-agent:latest
```

**系统服务**:
```bash
# 安装为系统服务
sudo make install

# 启动服务
sudo systemctl start node-agent

# 开机自启
sudo systemctl enable node-agent
```

## 部署流程

### 标准部署流程

1. **启动 NodeAgent**
   ```bash
   ./node_agent
   ```

2. **部署项目**
   ```bash
   curl -X POST http://127.0.0.1:8080/api/v1/projects/demo/deploy \
     -H "Content-Type: application/json" \
     -d '{"version":"1.0.0","autoStart":true}'
   ```

3. **启动项目**
   ```bash
   curl -X POST http://127.0.0.1:8080/api/v1/projects/demo/start
   ```

4. **监控状态**
   ```bash
   curl http://127.0.0.1:8080/api/v1/projects/demo
   ```

### 版本回滚流程

1. **查询可用版本**
   ```bash
   curl http://127.0.0.1:8080/api/v1/projects/demo
   ```

2. **执行回滚**
   ```bash
   curl -X POST "http://127.0.0.1:8080/api/v1/projects/demo/rollback?version=v0.9.0"
   ```

## 故障排查

### 常见问题

1. **项目启动失败**
   - 检查日志: `tail -f /var/log/node_agent/runtime/{projectID}.log`
   - 验证二进制文件是否存在
   - 检查权限

2. **Docker 执行器不可用**
   - 检查 Docker 服务状态
   - 验证 Docker Socket 权限
   - 查看自动回退日志

3. **健康检查失败**
   - 检查 `/health` 端点是否可达
   - 验证超时和重试配置
   - 查看健康检查日志

4. **心跳上报失败**
   - 检查网络连接
   - 验证管理中心地址
   - 查看心跳日志

### 日志位置

```
/var/log/node_agent/
├── agent.log              # NodeAgent 主日志
└── runtime/               # 项目运行时日志
    ├── demo.log
    └── api.log
```

## 扩展开发

### 添加新的执行器

1. 实现 `Executor` 接口
2. 在 `cmd/main.go` 中注册
3. 更新配置选项

### 添加 API 接口

1. 在 `internal/web/handler/api.go` 中添加方法
2. 在 `internal/web/handler/routes.go` 中注册路由
3. 更新前端 API 调用

### 自定义存储后端

1. 实现 `Store` 接口
2. 在配置中指定存储类型
3. 实现对应的存储逻辑

## 总结

NodeAgent 提供了完整的 RuntimeEngine 部署和管理能力，包括：

- ✅ 多执行器支持（进程、Docker）
- ✅ 状态机编排保证一致性
- ✅ 健康检查自动恢复
- ✅ 心跳上报实时监控
- ✅ 版本管理原子切换
- ✅ 连接配置安全存储
- ✅ 完整 REST API
- ✅ 日志记录和调试

通过这些功能，NodeAgent 可以可靠地在各种环境下部署和管理 RuntimeEngine，确保服务的稳定运行。
