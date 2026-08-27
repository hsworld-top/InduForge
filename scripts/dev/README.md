# 开发环境初始化

Linux 开发人员执行：

```bash
./scripts/dev/init-linux.sh
```

## 前置条件

- Docker 已安装并启动。
- Docker Compose 可用。

脚本不负责安装这些工具。

## 执行内容

- 生成根目录 `.env`，如果它不存在。
- 启动 `scripts/docker/docker-compose.dev.yml` 中的基础设施容器。
- 启动并检查单节点 NATS JetStream 工程运行内部总线。
- 创建 `if_core`、`if_data`、`if_dev_data`。
- 为 `if_dev_data` 启用开发态时序扩展。
- 默认不安装 Node 依赖，也不启动业务项目。
- `dev_core` 启动时会根据 `DB_AUTO_SCHEMA_SYNC=true` 自动同步 `if_core` 表结构和初始数据。

Windows + WSL2 开发时，建议只在 WSL2 执行本脚本；`pnpm install` 和业务项目启动在 Windows 侧执行。

Windows 原生 Collector 通过 `nats://127.0.0.1:18222` 连接 WSL2 Docker 中的 JetStream；本机监控端点为 `http://127.0.0.1:18223/jsz`。

Node 依赖由根目录 pnpm workspace 统一管理。只在仓库根目录执行一次 `pnpm install`，只提交根目录 `pnpm-lock.yaml`，不要在各业务子目录单独安装依赖。pnpm 在 workspace 子目录生成的 `node_modules/` 链接或提升目录属于安装产物。

脚本只处理基础设施。控制面和设计中心长期共用 `if_core`，不会创建 `if_design`。控制面结构 SQL 位于 `dev_core/db/schema/core-schema.sql`，由 `dev_core` 启动或离线安装命令调用。

## 内部脚本

- `init-linux.sh`：开发人员入口。
- `init-infra.mjs`：Node 版本的开发初始化实现，供已安装 `dev_core` 依赖的本地环境按需检查或复用。
- `init-infra.ps1`：Windows 开发初始化备用入口。

## 不执行内容

脚本不会启动任何业务项目。

开发人员需要自行启动后端和前端模块。

从仓库根目录执行任一 `pnpm run dev:*` 前，会自动检查根 `.env`：文件不存在时从 `.env.development.example` 生成，文件已存在时不覆盖本机配置。

`pnpm run dev:core` 使用 Air 监听 `dev_core` 源码变更并自动重新编译、重启服务。Air 需要预先安装并可通过 `air` 命令调用。

## 计算沙箱

本机开发 `data_service` 时，计算脚本仍在独立 Linux 容器中执行。macOS、Windows 和 Linux 均可通过 Docker 启动：

```bash
pnpm dev:compute-sandbox
```

脚本从根目录 `.env` 读取 `DATA_SERVICE_COMPUTE_SANDBOX_TOKEN`，构建开发镜像并将服务映射到 `127.0.0.1:18103`。重复执行是幂等的：容器已运行时只验证健康状态、JavaScript/Python 版本和隔离能力，不打印内部令牌。

常用维护命令：

```bash
pnpm dev:compute-sandbox:status
pnpm dev:compute-sandbox:logs
pnpm dev:compute-sandbox:stop
pnpm dev:compute-sandbox:rebuild
```

修改 `compute_sandbox/`、Dockerfile 或内部令牌后使用 `rebuild` 重建容器；普通前端和 `data_service` 代码修改不需要重建沙箱。
