# STACK

## 代码库定位
- 本仓库是工业低代码平台多模块单仓，核心目录为 `dev_core`、`dev_ide`、`datacenter`、`designer`、`runtime/node_agent`、`runtime/node_agent_front`。
- 后端统一入口由 `dev_core/src/index.js` 启动，核心 API 前缀为 `/api/v1`，健康检查为 `/health`（见 `dev_core/src/app.js`）。
- `runtime/node_agent` 是 Go 编写的节点执行器，`runtime/node_agent_front` 是其本地 Vue 管理界面。

## 语言与运行时
- JavaScript/Node.js：`dev_core`、`dev_ide`、`datacenter`、`runtime/node_agent_front`。
- TypeScript：`designer`（例如 `designer/src/stores/editor-store.ts`）。
- Go：`runtime/node_agent`（见 `runtime/node_agent/go.mod`，Go 1.24.0）。

## 后端技术栈（dev_core）
- Web 框架：`express`（`dev_core/package.json`，入口 `dev_core/src/app.js`）。
- ORM：`sequelize`（`dev_core/src/config/database.js` + `dev_core/src/models/index.js`）。
- 数据库驱动：`mysql2`、`pg`、`mssql`（驱动工厂 `dev_core/src/services/drivers/DriverFactory.js`）。
- 缓存与队列：`ioredis`（`dev_core/src/utils/redis.js`）。
- 对象存储：`minio`（`dev_core/src/services/storageService.js`）。
- 实时通信：`socket.io`（`dev_core/src/services/socketService.js`）。
- 设备消息协议：`mqtt`（`dev_core/src/services/mqttService.js` + `dev_core/src/services/protocols/MqttProtocol.js`）。
- 安全与中间件：`helmet`、`cors`、`express-rate-limit`、`jsonwebtoken`、`bcryptjs`。
- 时间库：`dayjs`（符合仓库规则，`dev_core/src/config/database.js` 等）。

## 前端技术栈
- `dev_ide`：Vue 3 + Vite + Pinia + Element Plus + vue-router + vue-i18n（`dev_ide/package.json`、`dev_ide/src/main.js`）。
- `datacenter`：Vue 3 + Vite + Pinia + Element Plus + Monaco + SQL 解析/格式化（`datacenter/package.json`）。
- `designer`：Vue 3 + TypeScript + Vite + Pinia + Element Plus + Konva + ECharts + GSAP + Vitest（`designer/package.json`）。
- `runtime/node_agent_front`：Vue 3 + Vite + Pinia + Element Plus（`runtime/node_agent_front/package.json`）。

## 节点执行器技术栈（runtime/node_agent）
- HTTP 路由：`gorilla/mux`（`runtime/node_agent/internal/web/handler/routes.go`）。
- 容器能力：Docker SDK（`runtime/node_agent/internal/agent/executor/docker.go`）。
- 本地执行：进程执行器（`runtime/node_agent/internal/agent/executor/process.go`）。
- 本地持久化：SQLite（`modernc.org/sqlite`，`runtime/node_agent/internal/agent/store/store.go`）。
- 配置框架：`viper`（`runtime/node_agent/cmd/main.go`）。

## 构建与包管理
- Node 模块统一使用 `pnpm`（根 AGENTS 规则）。
- 前端构建工具统一为 Vite（各子模块 `vite.config.js` / `vite.config.ts`）。
- Go 模块由 `runtime/node_agent/Makefile` 提供 `build/test/vet`。

## 环境配置
- 根环境变量模板：`.env_example`。
- 关键端口：`PORT=9099`（后端）、`VITE_IDE_PORT=9091`、`VITE_DATACENTER_PORT=9092`、`VITE_DESIGNER_PORT=9093`、`NODE_AGENT_PORT=8081`、`VITE_NODE_AGENT_FRONT_PORT=13000`。
- 数据与中间件：MySQL、Redis、MinIO、MQTT 在 `.env_example` 中提供默认项。

## 结论
- 当前仓库是“Node.js 业务聚合后端 + 多 Vue 前端 + Go 节点执行器”的混合技术栈。
- 业务核心在 `dev_core` 与 `designer`，运行时控制链路在 `dev_core -> runtime/node_agent -> runtime/node_agent_front`。
