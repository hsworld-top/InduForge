# NodeAgent 与工程运行栈启动协议

## 1. 文档定位

- 本文档定义 `runtime/node_agent` 如何在节点侧托管一个工程运行栈。
- 服务站点以一个工程一个 `docker-compose` 为基本单元；原生采集站点以一个工程一组 collector 子进程为基本单元。
- 本协议覆盖发布制品落地、拓扑规划、compose 或原生进程计划生成、启动、停止、探活、回滚和状态回传。
- 页面运行语义、数据采集业务逻辑、报警规则计算细节不属于 `node_agent` 职责。

## 2. 当前已实现

### 2.1 已有基础

- NodeAgent 已具备部署执行、目录管理、状态回传和本地控制能力。
- NodeAgent 已具备进程、Docker、systemd 等执行器抽象。

### 2.2 当前缺口

- `client_engine`、`runtime_api` 和运行态数据引擎尚未实现，启动命令、工作目录和探活协议尚未冻结。
- `runtime_data_service` 组尚未实现，节点侧尚未形成按能力裁剪的工程 compose。
- `Runtime Topology Planner`、`runtime-topology.json`、`docker-compose.runtime.yml` 和 `native-processes.json` 尚未冻结。
- Windows Service 形态的原生采集站点安装、升级、回滚和诊断协议尚未实现。

## 3. 核心原则

1. `node_agent` 以工程为单位托管运行栈，而不是只托管单个进程。
2. 服务站点运行栈由一份生成后的 `docker-compose.runtime.yml` 描述；原生采集站点由 `native-processes.json` 描述。
3. `node_agent` 只理解运行拓扑、容器角色、健康检查和状态，不理解采集协议、查询语义和报警规则。
4. 工程未使用的能力不启动对应容器或原生进程。
5. 外部接入源只作为连接配置进入数据引擎，不由 `node_agent` 部署外部服务端。
6. IF 消息库只承载工程 MQTT 能力，不作为数据引擎内部消息总线。
7. 采集盒子可以作为独立站点运行采集子栈，通过 NATS JetStream 上行到服务站点。
8. 服务站点运行栈默认只有 `project_nginx` 对外暴露端口，其他容器只在 compose 内部网络访问。
9. Windows 和 Linux 原生采集站点默认不开放入站端口，只允许出站连接服务站点。

## 4. 工作目录约定

```text
runtime-data/
  projects/
    <projectId>/
      releases/
        <version>/
          package/
          runtime-topology.json
          docker-compose.runtime.yml
          native-processes.json
          .env
          logs/
          state/
      current/
      shared/
        secrets/
        volumes/
        collector-buffer/
```

说明：

- `package/` 保存解压后的 `.ifp` 制品。
- `runtime-topology.json` 是拓扑规划结果。
- `docker-compose.runtime.yml` 是容器模式最终启动文件，原生采集站点不生成该文件。
- `native-processes.json` 是原生采集站点的子进程、工作目录、健康检查和重启策略计划。
- `.env` 保存本版本运行环境变量，不保存明文业务密钥。
- `shared/secrets/` 保存节点本地密钥文件或解密后的临时 secret。
- `shared/volumes/` 保存工程级持久化卷映射。
- `shared/collector-buffer/` 保存采集站点本地 WAL 和断线缓冲。

原生采集站点推荐目录：

```text
Windows:
  C:\Program Files\InduForge\node_agent\
  C:\ProgramData\InduForge\node_agent\

Linux:
  /opt/induforge/node_agent/
  /var/lib/induforge/node_agent/
  /var/log/induforge/node_agent/
```

Windows 上 `node-agent.exe` 注册为 Windows Service；Linux 上 `node-agent` 注册为 systemd service。collector 不注册为多个系统服务，而是由 `node_agent` 按 `native-processes.json` 托管为子进程。

## 5. 启动流程

1. NodeAgent 获取部署命令。
2. 下载或接收 `.ifp` 到本地版本目录。
3. 解压并校验 `manifest.json`、包摘要和协议版本。
4. 读取 `DataDomainSnapshot`、`RuntimeSecuritySnapshot` 和运行能力图。
5. 读取节点资源 profile、部署 profile 和当前节点所属站点。
6. 调用 `Runtime Topology Planner` 生成 `runtime-topology.json`。
7. 根据拓扑和执行方式生成 `docker-compose.runtime.yml` 或 `native-processes.json`。
8. 按执行方式预创建工程网络、volume、secret、WAL 和日志目录。
9. 服务站点启动运行栈基础组件，例如 IF 内置库、NATS JetStream 和协调组件，且默认不映射到宿主机端口。
10. 服务站点等待基础组件 `/health` 或容器健康检查通过。
11. 服务站点启动 `data_engine_bootstrap` 一次性任务；原生采集站点执行本地采集配置校验。
12. `bootstrap` 或本地校验成功后，启动数据引擎角色容器或 collector 子进程。
13. 等待数据引擎角色健康。
14. 服务站点启动 `runtime_api`，准备 `client_engine` 静态资源、生成 `runtime-config.json` 并启动 `project_nginx`；采集站点跳过该步骤。
15. 服务站点更新本地工程入口记录；采集站点只回报采集状态、上游状态和本地缓冲状态。
16. 标记部署为 `RUNNING`；若非关键角色异常但主链路可用，标记为 `DEGRADED`。

## 6. Runtime Topology Planner

### 6.1 输入

| 输入 | 说明 |
| --- | --- |
| `manifest.json` | 工程制品基础信息 |
| `DataDomainSnapshot` | 数据连接、数据点、查询、采集、报警、计算定义 |
| `RuntimeSecuritySnapshot` | 运行态用户、角色和资源授权快照 |
| `runtimeCapabilities` | 发布阶段生成的运行能力图 |
| 节点资源 profile | CPU、内存、磁盘、端口池、是否允许 Docker |
| 部署 profile | 单机、热备、分片、副本数上限、执行方式等 |
| 当前站点 profile | `server` 或 `collector`，以及 `siteId`、`upstreamSiteId` |

### 6.2 输出

| 输出 | 说明 |
| --- | --- |
| `runtime-topology.json` | 运行拓扑、角色、副本、依赖和健康检查计划 |
| `docker-compose.runtime.yml` | 容器模式实际启动文件 |
| `native-processes.json` | 原生采集站点实际启动文件 |
| `.env` | 本版本运行环境变量 |
| 本地入口记录草案 | 工程宿主机端口、`project_nginx` 目标和 `runtime_api` 内部地址 |
| 上游连接计划 | 采集站点通过服务站点 `project_nginx` 入口连接 NATS JetStream 的地址、认证和主题 |

### 6.3 裁剪规则

| 条件 | 结果 |
| --- | --- |
| 未使用 IF关系库 | 不启动 `if_relation_store` |
| 未使用 IF时序库 | 不启动 `if_timeseries_store` |
| 未使用 IF实时库 | 不启动 `if_realtime_store` |
| 未使用 IF消息库 | 不启动 `if_message_store` |
| 无采集源 | 不启动 `data_engine_collector_*` |
| 无报警规则 | 不启动 `data_engine_alarm` |
| 无计算单元 | 不启动 `data_engine_compute` |
| 单实例轻量运行 | 可不启动 `engine_bus` 和 `engine_coord` |
| 多角色、多副本、HA 或分片 | 按拓扑启动 `engine_bus` 和 `engine_coord` |
| 当前站点为采集站点 | 只启动对应 collector、可选本地缓冲和诊断入口 |
| 当前站点为服务站点且采集外置 | 不启动现场协议 collector |

### 6.4 执行方式

| 执行方式 | 生成内容 | 执行器 |
| --- | --- | --- |
| `service-linux-container` | 完整 `docker-compose.runtime.yml` | Docker Compose |
| `collector-linux-container` | 仅采集子栈 `docker-compose.runtime.yml` | Docker Compose |
| `collector-windows-native` | `native-processes.json`、Windows Service 参数、本地目录计划 | Windows Service + 子进程 |
| `collector-linux-native` | `native-processes.json`、systemd 参数、本地目录计划 | systemd + 子进程 |

## 7. 运行栈容器角色

一个工程运行栈可包含以下容器。实际是否启动由拓扑决定。

| 容器角色 | 说明 |
| --- | --- |
| `project_nginx` | 工程入口，映射一个独立宿主机端口 |
| `client_engine` | 设计中心页面客户端运行引擎 |
| `runtime_api` | 工程运行网关，承接登录、权限、数据 API、报警动作和 WebSocket |
| `data_engine_collector_<protocol>` | 协议采集器，例如 S7、OPC UA、Modbus、MQTT |
| `data_engine_writer` | 实时值和时序数据批量写入器 |
| `data_engine_alarm` | 报警状态机与报警事件写入 |
| `data_engine_compute` | 计算任务和脚本沙箱 |
| `data_engine_scheduler` | 分片、租约、调度和选主 |
| `data_engine_bootstrap` | 一次性初始化和迁移任务 |
| `engine_bus` | 数据引擎内部可靠事件流，固定采用 NATS JetStream |
| `engine_coord` | 数据引擎内部协调、租约和选主 |
| `if_relation_store` | IF关系库 |
| `if_timeseries_store` | IF时序库 |
| `if_realtime_store` | IF实时库 |
| `if_message_store` | IF消息库，只承载 MQTT |

原生采集站点不启动上述容器角色，只启动 `data_engine_collector_<protocol>` 对应的本地进程。collector 进程复用同一份 `runtime-data-engine` 二进制，通过 `--role=collector` 和 `--protocol` 区分职责。

## 8. 启动参数契约

### 8.1 公共环境变量

| 变量 | 说明 |
| --- | --- |
| `INDUFORGE_PROJECT_ID` | 工程 ID |
| `INDUFORGE_PROJECT_CODE` | 工程编码 |
| `INDUFORGE_RELEASE_VERSION` | 当前版本号 |
| `INDUFORGE_PROJECT_DIR` | 当前版本 package 目录 |
| `INDUFORGE_TOPOLOGY_FILE` | `runtime-topology.json` 路径 |
| `INDUFORGE_RUNTIME_ENV` | `production`、`offline` 或 `development` |
| `INDUFORGE_SITE_ID` | 当前运行站点 ID |
| `INDUFORGE_SITE_TYPE` | `server` 或 `collector` |
| `INDUFORGE_UPSTREAM_SITE_ID` | 采集站点对应的服务站点 ID |
| `INDUFORGE_EXECUTION_MODE` | `service-linux-container`、`collector-windows-native`、`collector-linux-native` 等 |

### 8.2 runtime_api 参数

```bash
runtime-api \
  --project-dir=/app/project \
  --topology=/app/runtime-topology.json \
  --host=0.0.0.0 \
  --port=8080
```

### 8.3 数据引擎公共参数

采集器示例：

```bash
runtime-data-engine \
  --role=collector \
  --protocol=s7 \
  --project-dir=/app/project \
  --topology=/app/runtime-topology.json
```

原生采集站点示例：

```bash
runtime-data-engine \
  --role=collector \
  --protocol=s7 \
  --project-dir="<project-dir>" \
  --topology="<runtime-topology.json>" \
  --wal-dir="<collector-wal-dir>" \
  --upstream="<service-site-project-nginx>"
```

### 8.4 client_engine 参数

`client_engine` 是 Vue/TypeScript 静态运行时，不作为常驻 Node 后台启动。NodeAgent 需要生成 `runtime-config.json`，由 `project_nginx` 与静态资源一起托管。

`runtime-config.json` 至少包含：

```json
{
  "projectId": "proj_xxx",
  "projectCode": "factory_dashboard",
  "version": "2026.05.26-001",
  "apiBase": "/api",
  "wsBase": "/ws",
  "manifestUrl": "/project/manifest.json",
  "defaultLocale": "zh-CN"
}
```

## 9. 健康检查协议

### 9.1 单容器健康

每个长期运行容器或原生子进程至少提供：

```text
GET /health
GET /status
```

`/health` 只判断进程是否可服务。`/status` 返回角色、版本、启动时间、依赖状态、最后错误和关键指标摘要。

### 9.2 工程级健康

NodeAgent 聚合以下状态：

- compose 服务状态或原生子进程状态。
- 基础组件健康。
- 数据引擎角色健康。
- `project_nginx`、静态资源和 `runtime-config.json` 可达。
- `runtime_api` 健康。
- 工程入口可达性。
- 采集站点上游可达性和本地 WAL 积压。
- 关键角色是否降级。

工程状态枚举：

```text
STARTING
RUNNING
DEGRADED
FAILED
STOPPING
STOPPED
ROLLING_BACK
```

## 10. 端口暴露协议

服务站点工程运行栈默认只允许 `project_nginx` 映射宿主机端口。原生采集站点默认不开放入站端口。

| 容器角色 | 默认端口策略 |
| --- | --- |
| `project_nginx` | 映射到工程独立宿主机端口 |
| `client_engine` | 不映射宿主机端口 |
| `runtime_api` | 不映射宿主机端口 |
| `data_engine_collector_*` | 不映射宿主机端口 |
| `data_engine_writer` | 不映射宿主机端口 |
| `data_engine_alarm` | 不映射宿主机端口 |
| `data_engine_compute` | 不映射宿主机端口 |
| `data_engine_scheduler` | 不映射宿主机端口 |
| `engine_bus` | 不映射宿主机端口，通过 `project_nginx` 代理采集上行 |
| `engine_coord` | 不映射宿主机端口 |
| IF 内置库 | 不映射宿主机端口，除非部署 profile 显式启用受控入口 |

NodeAgent 生成 `docker-compose.runtime.yml` 时，应默认只为 `project_nginx` 写入 `ports`。其他服务使用 compose 内部网络 `expose` 或仅通过服务名访问。

原生采集站点的本地诊断端口只能绑定 `127.0.0.1`。调试端口只能通过运维模式临时开启。NodeAgent 必须记录开启人、开启范围、目标进程或容器、端口、原因和关闭时间。

## 11. 停止协议

1. 服务站点停止或禁用工程入口；采集站点停止接收新的采集调度。
2. 通知数据引擎停止接收新采集任务。
3. 数据引擎释放租约。
4. 等待 collector 写完本地 WAL；服务站点还需要等待 writer flush，超过超时后记录未完成状态。
5. 停止数据引擎角色容器或原生 collector 子进程。
6. 服务站点停止 `project_nginx`。
7. 服务站点停止本工程独占的内部组件和 IF 内置库。
8. 保存最后状态、错误摘要和 compose 或原生进程退出信息。

## 12. 回滚约定

- 回滚只允许切换到本地已有可用版本。
- 回滚必须重新加载该版本的 `runtime-topology.json`，并按执行方式加载 `docker-compose.runtime.yml` 或 `native-processes.json`。
- 如果旧版本拓扑缺失，NodeAgent 可基于旧版本制品重新生成拓扑。
- 回滚前必须停止或禁用当前版本工程入口。
- 回滚成功后重新启用工程入口。
- 回滚失败必须记录最后错误并上报平台。

## 13. 异常处理

### 下载失败

- 状态改为 `FAILED`。
- 保存下载错误信息。

### 解压或校验失败

- 拒绝生成拓扑。
- 上报制品错误。

### 拓扑规划失败

- 拒绝启动 compose 或原生进程计划。
- 保存 planner 输入摘要和错误原因。

### 基础组件启动失败

- 停止已启动容器。
- 状态改为 `FAILED`。

### bootstrap 或本地校验失败

- 不启动业务角色容器或 collector 子进程。
- 状态改为 `FAILED`。

### 非关键角色失败

- 若主访问链路仍可用，状态可标记为 `DEGRADED`。
- NodeAgent 应记录降级角色、失败次数和最后错误。

## 14. 非目标

- 不定义数据采集协议细节。
- 不定义报警规则执行细节。
- 不定义数据查询 SQL 方言。
- 不让 NodeAgent 直接连接用户外部数据库或设备。
- 不要求所有工程共享同一套数据引擎容器。

## 15. 关联文档

- [产品定义](../产品定义.md)
- [高层设计](../高层设计.md)
- [详细设计](../详细设计.md)
- [节点侧运行态数据引擎设计](../节点侧运行态数据引擎设计.md)
- [NodeAgent 后端概览](../node_agent/README.md)
- [Runtime 健康状态协议](./runtime-health-status-contract.md)
- [测试与质量策略](../测试与质量策略.md)
